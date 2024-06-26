package main

import (
	"flag"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/opentracing/opentracing-go"
	"github.com/satori/go.uuid"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
	"go_srvs/order_srv/global"
	"go_srvs/order_srv/handler"
	"go_srvs/order_srv/initialize"
	"go_srvs/order_srv/proto"
	"go_srvs/order_srv/utils"
	"go_srvs/order_srv/utils/otgrpc"
	"go_srvs/order_srv/utils/register/consul"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	IP := flag.String("ip", "0.0.0.0", "ip 地址")
	Port := flag.Int("port", 50051, "端口号")

	initialize.InitLogger()
	initialize.InitConfig()
	initialize.InitDB()
	initialize.InitSrvConn()

	zap.S().Info(global.ServerConfig)

	flag.Parse()
	zap.S().Info("ip:", *IP)
	if *Port == 0 {
		*Port, _ = utils.GetFreePort()
	}

	zap.S().Info("port:", *Port)

	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: "192.168.11.130:6831",
		},
		ServiceName: "mxshop",
	}
	tracer, closer, err := cfg.NewTracer(jaegercfg.Logger(jaeger.StdLogger))
	if err != nil {
		panic(err)
	}
	opentracing.SetGlobalTracer(tracer)
	server := grpc.NewServer(grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)))

	proto.RegisterOrderServer(server, &handler.OrderServer{})
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *IP, *Port))
	if err != nil {
		panic("failed to listen" + err.Error())
	}
	grpc_health_v1.RegisterHealthServer(server, health.NewServer())
	go func() {
		err = server.Serve(lis)
		if err != nil {
			panic("failed to start grpc" + err.Error())
		}

	}()

	registryClient := consul.NewRegistryClient(global.ServerConfig.ConsulInfo.Host, global.ServerConfig.ConsulInfo.Port)
	serviceId := fmt.Sprintf("%s", uuid.NewV4())
	err = registryClient.Register(global.ServerConfig.Host, *Port, global.ServerConfig.Name, global.ServerConfig.Tags, serviceId)
	if err != nil {
		zap.S().Panic("服务器注册失败", err.Error())
	}
	zap.S().Debugf("启动服务器...端口%d", *Port)

	//监听库存归还topic
	c, _ := rocketmq.NewPushConsumer(
		consumer.WithNameServer([]string{"192.168.11.130:9876"}),
		consumer.WithGroupName("mxshop-order"),
	)

	if err := c.Subscribe("order_timeout", consumer.MessageSelector{}, handler.OrderTimeout); err != nil {
		fmt.Println("订阅失败", err.Error())
	}
	_ = c.Start()
	//time.Sleep(time.Hour)
	_ = c.Shutdown()

	//接受终止信号
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	_ = c.Shutdown()
	closer.Close()
	if err := registryClient.DeRegister(serviceId); err != nil {
		zap.S().Info("注销失败", err.Error())
	} else {
		zap.S().Info("注销成功")
	}
}
