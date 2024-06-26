package model

import (
	"database/sql/driver"
	"encoding/json"
)

type Inventory struct {
	BaseModel
	Goods   int32 `gorm:"type:int;index"`
	Stocks  int32 `gorm:"type:int"`
	Version int32 `gorm:"type:int"`
}
type InventoryNew struct {
	BaseModel
	Goods   int32 `gorm:"type:int;index"`
	Stocks  int32 `gorm:"type:int"`
	Version int32 `gorm:"type:int"`
	Freeze  int32 `gorm:"type:int"`
}
type Delivery struct {
	Goods   int32 `gorm:"type:int;index"`
	Nums    int32 `gorm:"type:int"`
	OrderSn int32 `gorm:"type:int;varchar(255)"`
	Status  int32 `gorm:"type:int;varchar(255)"`
}
type StockSellDetail struct {
	OrderSn string          `gorm:"type:varchar(255);index:idx_order_sn,unique"`
	Status  int32           `gorm:"type:varchar(255)"`
	Detail  GoodsDetailList `gorm:"type:varchar(255)"`
}

func (StockSellDetail) TableName() string {
	return "stockselldetail"
}

type GoodsDetail struct {
	Goods int32
	Num   int32
}

type GoodsDetailList []GoodsDetail

func (g GoodsDetailList) Value() (driver.Value, error) {
	return json.Marshal(g)

}
func (g *GoodsDetailList) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &g)

}

//	type Stock struct {
//		BaseModel
//		Name    string
//		Address string
//	}
//type InventoryHistory struct {
//	user   int32
//	inventory  int32
//	nums   int32
//	order  int32
//	status int32
//}
