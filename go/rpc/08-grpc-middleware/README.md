# gRPC教程-go-grpc-middleware

> 书店场景：订单服务调用任何下游前，都统一完成身份认证、请求日志、trace 记录和 panic 恢复，而不是在每个业务方法中重复这些代码。

这个示例演示拦截器链：TLS、token 认证、请求日志和 panic recovery。

原仓库使用 `github.com/grpc-ecosystem/go-grpc-middleware` v1。这里更新为 gRPC-Go 内置的 `grpc.ChainUnaryInterceptor`，教学目标一致，依赖更少，也更适合当前 Go 1.26.5 环境。

## 本节先懂这些

**拦截器（interceptor）就是 RPC 的统一门卫。**一次“创建订单”请求到达业务方法前，常常都要做同样的事：检查 token、记录请求耗时、提取 trace ID、从 panic 中恢复。把这些代码复制到每个 `GetBook`、`CreateOrder` 中既容易漏写，也会让业务代码变得难读。

Unary interceptor 的形状可以理解为：

```text
请求 → 日志 → 认证 → 业务方法 → 日志记录结果 → 响应
```

`grpc.ChainUnaryInterceptor` 把多个拦截器串成链；每个拦截器决定“继续调用下一个”还是“直接拒绝”。例如认证失败应直接返回 `Unauthenticated`，不再进入图书查询逻辑；recovery 拦截器则把意外 panic 转成可记录的服务端错误，防止整个进程崩溃。

拦截器适合横切关注点，不适合承载业务规则。“图书是否可售”属于业务方法；“调用方是否带合法身份”才属于认证拦截器。流式 RPC 使用 stream interceptor，不能直接套用 unary interceptor。

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
