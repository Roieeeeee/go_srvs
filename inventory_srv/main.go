package main

import (
	"flag"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/satori/go.uuid"
	"go.uber.org/zap"
	"go_srvs/inventory_srv/global"
	"go_srvs/inventory_srv/handler"
	"go_srvs/inventory_srv/initialize"
	"go_srvs/inventory_srv/proto"
	"go_srvs/inventory_srv/utils"
	"go_srvs/inventory_srv/utils/register/consul"
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
	Port := flag.Int("port", 50053, "端口号")

	initialize.InitLogger()
	initialize.InitConfig()
	initialize.InitDB()

	zap.S().Info(global.ServerConfig)

	flag.Parse()
	zap.S().Info("ip:", *IP)
	if *Port == 0 {
		*Port, _ = utils.GetFreePort()
	}

	zap.S().Info("port:", *Port)

	server := grpc.NewServer()
	proto.RegisterInventoryServer(server, &handler.InventoryServer{})
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
		consumer.WithGroupName("mxshop-inventory"),
	)

	if err := c.Subscribe("order_reback", consumer.MessageSelector{}, handler.AutoReback); err != nil {
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
	if err := registryClient.DeRegister(serviceId); err != nil {
		zap.S().Info("注销失败", err.Error())
	} else {
		zap.S().Info("注销成功")
	}
}
