# gRPC教程-服务端流式RPC（三）

> 书店场景：用户搜索图书，目录服务会持续返回多批匹配结果。

这个示例对应服务端流式 RPC：客户端发送一次请求，服务端连续返回多条响应。

## 本节先懂这些

**服务端流式 RPC 是“一次请求，连续多次响应”。**图书搜索可能命中成千上万本书；如果等所有结果准备好再一次性返回，用户会等待很久、服务端也要把所有结果堆在内存里。流式返回可以让客户端先显示第一批，再继续接收后续结果。

服务端不再 `return *Response`，而是在循环中调用 `Send`；客户端拿到的也不是单个结果，而是 stream，并不断 `Recv`。`io.EOF` 不是异常，它表示服务端已经正常发送完全部结果。真正的网络错误、客户端取消和服务端错误则会以非 EOF 的 error 返回。

流不是无限免费传输：客户端读得慢会影响服务端发送，客户端应设置 deadline 并允许用户取消搜索；服务端每次 `Send` 都必须处理错误。适合搜索结果、日志订阅、进度通知；如果只需要一条图书详情，仍应使用一元 RPC。

## 目录结构

```text
.
├── cmd
│   ├── client
│   │   └── main.go
│   └── server
│       └── main.go
├── proto
│   └── server_stream.proto
├── go.mod
└── README.md
```

## 生成 proto 代码

```bash
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/server_stream.proto
```

## 运行

```bash
go run ./cmd/server
```

另开终端：

```bash
go run ./cmd/client
```

客户端输出类似：

```text
stream server grpc 0
stream server grpc 1
stream server grpc 2
stream server grpc 3
stream server grpc 4
```
