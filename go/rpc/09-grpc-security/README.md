# gRPC教程-安全认证

> 书店场景：订单服务调用库存服务时，TLS 保护传输内容，token 让库存服务验证调用方身份。

这个示例演示 TLS 证书认证和基于 metadata 的 token 认证。

## 本节先懂这些

这里有两个容易混淆、但缺一不可的概念：

- **TLS** 解决“传输是否安全、连的是不是真服务”。它会加密订单和库存数据，并通过证书帮助客户端确认服务端身份。
- **token 认证**解决“调用者是谁、是否有权限”。库存服务收到请求后，从 gRPC metadata 中读取 token，决定订单服务能不能访问。

可以把 TLS 想成防窃听、防假冒的安全通道；把 token 想成进入通道后出示的工作证。只开 TLS 而不校验 token，任何能连上网络的人仍可能调用服务；只传 token 而不用 TLS，token 可能在传输中泄露。

gRPC metadata 类似 HTTP header，适合传 trace ID、认证凭证、灰度标记等小型附加信息，不应用来传大对象或机密日志。示例中的证书仅用于本地学习；生产环境应使用受信任的证书、定期轮换凭证，并绝不把真实 token 写进仓库。

## 生成 proto 代码

```bash
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/simple.proto
```

## 运行

```bash
go run ./cmd/server
```

另开终端：

```bash
go run ./cmd/client
```
