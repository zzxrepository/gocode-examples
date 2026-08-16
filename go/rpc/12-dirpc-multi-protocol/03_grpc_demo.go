//go:build ignore

// 订单服务一次查询多本书的库存：gRPC + Proto 生成客户端的调用形状。
package main

import (
	"context"
	"fmt"
)

type InventoryClient struct{ serviceName string }

func NewInventoryClient(serviceName string) (*InventoryClient, error) {
	return &InventoryClient{serviceName: serviceName}, nil
}

type BatchInventoryRequest struct {
	TraceID     string
	BookIDs     []string
	WarehouseID string
}

type Inventory struct {
	BookID    string
	Available int32
}

type BatchInventoryResponse struct {
	Items map[string]*Inventory
}

// 写法与 typed HTTP 几乎一样，区别由生成的 Proto 类型和底层 gRPC 传输承担。
func (c *InventoryClient) BatchGetInventory(ctx context.Context, req *BatchInventoryRequest) (*BatchInventoryResponse, error) {
	fmt.Printf("[gRPC] %s.BatchGetInventory: books=%v, warehouse=%s\n", c.serviceName, req.BookIDs, req.WarehouseID)
	items := make(map[string]*Inventory, len(req.BookIDs))
	for _, id := range req.BookIDs {
		items[id] = &Inventory{BookID: id, Available: 12}
	}
	return &BatchInventoryResponse{Items: items}, nil
}

var inventoryClient *InventoryClient

func initInventoryClient() error {
	var err error
	inventoryClient, err = NewInventoryClient("disf!book-inventory")
	return err
}

func main() {
	if err := initInventoryClient(); err != nil {
		panic(err)
	}
	resp, err := inventoryClient.BatchGetInventory(context.Background(), &BatchInventoryRequest{
		TraceID: "trace-order-1001", BookIDs: []string{"book-1", "book-2"}, WarehouseID: "shanghai-01",
	})
	if err != nil {
		panic(err)
	}
	for id, item := range resp.Items {
		fmt.Printf("%s: available=%d\n", id, item.Available)
	}
}
