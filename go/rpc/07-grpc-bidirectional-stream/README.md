# gRPC教程-双向流式RPC（五）

> 书店场景：客服持续询问图书库存，库存服务持续返回对应结果；双方能在同一连接中交替发送消息。

这个示例对应双向流式 RPC：客户端和服务端都可以通过同一个 RPC 流连续发送消息。

## 本节先懂这些

**双向流式 RPC 让双方在同一条连接上各自持续发送消息。**客服可以接连询问多本书的库存，库存服务也可以逐条回答；它不要求必须“发一条、收一条”严格交替。示例为了容易观察，采用了简单的一问一答。

stream 同时有 `Send` 和 `Recv` 两个方向。真实业务若两边都会持续主动发消息，通常用两个 goroutine 分别负责发送和接收，避免一端因为等待 `Recv` 而阻塞发送。`CloseSend` 只表示“客户端不再发送”，不等于立即关闭接收方向；仍可以继续读服务端已经发出的消息。

双向流适合实时协作、订阅与交互式会话，不适合普通查详情接口：它要维护长连接、处理断线重连、消息顺序和背压，复杂度明显高于一元 RPC。

## 目录结构

```text
.
├── cmd
│   ├── client
│   │   └── main.go
│   └── server
│       └── main.go
├── proto
│   └── both_stream.proto
├── go.mod
└── README.md
```

## 生成 proto 代码

```bash
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/both_stream.proto
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
from stream server answer: the 1 question is stream client rpc 0
from stream server answer: the 2 question is stream client rpc 1
from stream server answer: the 3 question is stream client rpc 2
from stream server answer: the 4 question is stream client rpc 3
from stream server answer: the 5 question is stream client rpc 4
```
