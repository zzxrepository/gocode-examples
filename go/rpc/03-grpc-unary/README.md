# gRPC教程-简单RPC（二）

> 书店场景：订单服务调用图书目录服务，传一个 ISBN，得到一本书的详情。这是最常见的一问一答调用。

这个示例对应教程中的简单模式（Simple RPC）：客户端发送一次请求，服务端返回一次响应。

## 本节先懂这些

**一元 RPC（Unary RPC）就是一次提问、一次回答。**例如订单服务问图书目录：“ISBN 为 X 的书价格是多少？”目录服务返回一份图书详情。这和普通函数最像，也是内部服务调用最常见的模式。

```text
客户端构造 BookRequest → gRPC 编码并发送 → 服务端 Route/GetBook 执行业务
       ← gRPC 解码响应 ← 服务端返回 BookInfo
```

`.proto` 生成两侧要用的代码：服务端实现生成的 `Server` 接口并注册；客户端基于一条可复用的 `ClientConn` 创建生成的 client，再调用其方法。这里的 `conn` 是到底层服务的连接管理对象，不应该为每次查书重新创建；一次具体调用的超时、取消等信息放在 `context.Context` 中。

gRPC 默认使用 HTTP/2 传输、Proto 编码消息，但业务代码不必直接处理字节流。需要记住的是：调用返回的 `error` 表示连接、超时或服务端框架错误；即使 RPC 成功，响应中的业务状态仍可能表示“图书不存在”等业务失败。

## 目录结构

```text
.
├── cmd
│   ├── client
│   │   └── main.go
│   └── server
│       └── main.go
├── proto
│   └── simple.proto
├── go.mod
└── README.md
```

## 生成 proto 代码

```bash
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/simple.proto
```

## 运行服务端

```bash
go run ./cmd/server
```

正常输出类似：

```text
:8000 net.Listing...
```

## 运行客户端

另开一个终端：

```bash
go run ./cmd/client
```

正常输出类似：

```text
code:200 value:"hello grpc"
```
