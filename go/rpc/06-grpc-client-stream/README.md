# gRPC教程-客户端流式RPC（四）

> 书店场景：门店持续上传多条盘点记录，库存服务在接收完毕后返回一次汇总结果。

这个示例对应客户端流式 RPC：客户端连续发送多条请求，发送结束后服务端返回一次响应。

## 本节先懂这些

**客户端流式 RPC 是“连续上传，最后结算”。**门店盘点时可以把一条条库存记录持续发给库存服务，全部上传完后，服务端返回“已接收多少条、是否成功”的最终结果。它避免把大量记录先拼成一个超大的请求。

客户端先创建 stream，然后多次 `Send`；`CloseAndRecv` 有两个含义：告诉服务端“我发完了”，并等待最终响应。服务端则反复 `Recv`，收到 `io.EOF` 说明客户端正常结束发送，此时才 `SendAndClose` 返回汇总结果。

这里的关键边界是“谁负责结束流”：客户端必须显式关闭发送方向，否则服务端会一直等待下一条记录。生产中还要定义单条消息大小、总量上限和幂等标识，防止网络重试导致同一盘点记录被重复入库。

## 目录结构

```text
.
├── cmd
│   ├── client
│   │   └── main.go
│   └── server
│       └── main.go
├── proto
│   └── client_stream.proto
├── go.mod
└── README.md
```

## 生成 proto 代码

```bash
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/client_stream.proto
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
code:200 value:"ok"
```
