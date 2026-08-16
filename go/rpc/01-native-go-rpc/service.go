package main

import (
	"fmt"
	"log"
	"net/http"
	"net/rpc"
)

// BookRequest 是客户端传给图书服务的请求。
// 字段必须导出（首字母大写），net/rpc 才能通过 gob 编码它。
type BookRequest struct {
	ISBN string
}

// BookInfo 是服务端写回给客户端的结果。字段也必须导出。
type BookInfo struct {
	ISBN       string
	Title      string
	PriceCents int
	InStock    bool
}

// BookStore 是可被远程调用的服务对象。
type BookStore struct{}

// GetBook 是服务端暴露的 RPC 方法。
//
// net/rpc 要求方法签名固定为：
//
//	func (t *T) Method(args T1, reply *T2) error
//
// 因此调用结果必须通过 reply 指针写回，而不是作为普通返回值返回。
func (s *BookStore) GetBook(req BookRequest, reply *BookInfo) error {
	if req.ISBN != "978-7-115-56678-9" {
		return fmt.Errorf("book not found: %s", req.ISBN)
	}

	*reply = BookInfo{
		ISBN:       req.ISBN,
		Title:      "Go 语言程序设计",
		PriceCents: 6990,
		InStock:    true,
	}
	return nil
}

func main() {
	// 1. 注册服务后，满足签名要求的 GetBook 可通过 "BookStore.GetBook" 远程调用。
	if err := rpc.Register(&BookStore{}); err != nil {
		log.Panicln(err)
	}

	// 2. 让 HTTP 承载 RPC 报文。它不是普通的 REST API。
	rpc.HandleHTTP()

	// 3. 监听端口并持续处理请求。
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Panicln(err)
	}
}
