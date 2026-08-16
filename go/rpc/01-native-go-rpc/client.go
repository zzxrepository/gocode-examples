package main

import (
	"fmt"
	"log"
	"net/rpc"
)

// 这两个类型必须和 service.go 保持一致；调用双方要共享同一份接口契约。
type BookRequest struct {
	ISBN string
}

type BookInfo struct {
	ISBN       string
	Title      string
	PriceCents int
	InStock    bool
}

func main() {
	// 1. 与服务端的 rpc.HandleHTTP 建立连接。
	client, err := rpc.DialHTTP("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// 2. 像调本地方法一样远程调用服务名.方法名。
	// 这是同步调用：Call 会等待服务端响应或返回错误。
	var reply BookInfo
	err = client.Call(
		"BookStore.GetBook",
		BookRequest{ISBN: "978-7-115-56678-9"},
		&reply,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("图书：%s，售价：%.2f 元，库存：%t\n", reply.Title, float64(reply.PriceCents)/100, reply.InStock)
}
