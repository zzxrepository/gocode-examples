# gRPC教程-Deadlines

> 书店场景：创建订单时查询库存；库存服务若不能在订单允许的时间内返回，调用必须取消而不是无限等待。

这个示例演示客户端为 RPC 调用设置超时时间。服务端模拟 4 秒耗时操作，客户端先用 2 秒超时触发 deadline，再用 5 秒超时成功拿到响应。

## 本节先懂这些

**deadline 是“这次调用最晚什么时候必须结束”，不是“希望服务端快一点”。**没有它，库存服务卡住时，订单请求会一直占着 goroutine、连接和内存；高峰期这些等待会像排队一样向上游扩散。

客户端用 `context.WithTimeout` 创建带截止时间的 `ctx`，再把它传给 RPC。gRPC 会把取消信号传到服务端；服务端也必须主动监听 `ctx.Done()`，才能及时停止数据库查询、循环或其他耗时工作。示例中的 `select` 正在演示“工作完成”和“调用已取消”谁先发生。

要区分两种超时：**建连超时**限制“能否连上库存服务”，**调用超时**限制“本次查库存能等多久”。生产代码通常关注后者，并用 `status.Code(err)` 判断 `DeadlineExceeded`，不要把它和“库存为 0”这样的业务结果混为一谈。deadline 应从上游向下游传递，不能每层都重新给一个更长的时间。

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
