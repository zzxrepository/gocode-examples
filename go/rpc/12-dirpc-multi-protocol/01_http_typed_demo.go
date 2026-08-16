//go:build ignore

// 订单服务通过 dirpc 调用图书目录服务：HTTP + IDL 生成的强类型客户端。
// 真实项目中 Client、BookRequest、BookResponse 都由接口定义自动生成；本文件只保留调用形状。
package main

import (
	"context"
	"fmt"
)

type CatalogClient struct{ serviceName string }

// 服务名交给服务发现解析，不要把 IP:port 写死在业务代码里。
func NewCatalogClient(serviceName string) (*CatalogClient, error) {
	return &CatalogClient{serviceName: serviceName}, nil
}

// 强类型请求：字段拼错会在编译期暴露，而不是运行时因 JSON key 写错而失败。
type BookRequest struct {
	ISBN    string
	StoreID string
}

type BookResponse struct {
	Errno      int
	Errmsg     string
	Title      string
	PriceCents int
}

// 第一个参数始终是 ctx。框架会从其中传递超时、取消和 trace 等上下文。
func (c *CatalogClient) GetBook(ctx context.Context, req *BookRequest) (*BookResponse, error) {
	fmt.Printf("[typed HTTP] %s.GetBook: isbn=%s, store=%s\n", c.serviceName, req.ISBN, req.StoreID)
	return &BookResponse{Title: "Go 语言程序设计", PriceCents: 6990}, nil
}

var catalogClient *CatalogClient

func initCatalogClient() error {
	var err error
	catalogClient, err = NewCatalogClient("disf!book-catalog")
	return err
}

func getBook(ctx context.Context, isbn string) (*BookResponse, error) {
	resp, err := catalogClient.GetBook(ctx, &BookRequest{ISBN: isbn, StoreID: "store-shanghai"})
	if err != nil { // 网络、超时、熔断等 transport error
		return nil, err
	}
	if resp.Errno != 0 { // HTTP 成功不等于业务成功
		return nil, fmt.Errorf("catalog error: %s", resp.Errmsg)
	}
	return resp, nil
}

func main() {
	if err := initCatalogClient(); err != nil {
		panic(err)
	}
	book, err := getBook(context.Background(), "978-7-115-56678-9")
	if err != nil {
		panic(err)
	}
	fmt.Printf("book=%s, price=%.2f\n", book.Title, float64(book.PriceCents)/100)
}
