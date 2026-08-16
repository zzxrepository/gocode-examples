# 01：Go 标准库 RPC——查询图书详情

这是整套教程的起点：不用 Proto、gRPC 或公司框架，只用 `net/rpc` 看清一次 RPC 的最小闭环。

```text
client.go                            service.go
BookStore.GetBook(ISBN)  ──HTTP──>  查图书数据并返回 BookInfo
```

## 本节先懂这些

**RPC 不是把代码搬到远端执行。**它是客户端把“方法名 + 参数”编码成网络报文发给服务端；服务端找到对应方法执行，再把结果编码回去。`client.Call("BookStore.GetBook", ...)` 看起来像本地函数调用，实际已经跨过了网络。

这份示例把 RPC 拆成四个最小零件：

1. **服务对象**：`BookStore` 是提供能力的人，`GetBook` 是它公开的能力。
2. **接口契约**：`BookRequest` 和 `BookInfo` 规定双方交换什么数据。客户端、服务端必须理解同样的字段。
3. **注册与监听**：`rpc.Register` 将 Go 方法名登记到 RPC 框架，`ListenAndServe` 开端口等待请求。
4. **连接与调用**：`rpc.DialHTTP` 建立连接，`Call` 发送请求并等待响应。

`net/rpc` 使用 Go 的 gob 编码，因此字段必须首字母大写；小写字段对反射不可见，传到对端时会丢失。它的固定方法签名也是一种早期的接口契约。后续 Proto/gRPC 会把这种契约从“双方手动保持一致”升级为“由 `.proto` 自动生成”。

## 运行

第一个终端启动服务端：

```bash
go run service.go
```

第二个终端发起远程调用：

```bash
go run client.go
```

## 要点

1. `rpc.Register(&BookStore{})` 将可远程调用的方法注册为 `BookStore.GetBook`。
2. `rpc.HandleHTTP()` 用 HTTP 承载 RPC 报文；这不是普通 REST API。
3. 客户端通过 `rpc.DialHTTP` 建立连接，再调用 `client.Call("BookStore.GetBook", args, &reply)`。
4. `net/rpc` 的服务端方法必须形如 `func (t *T) Method(args T1, reply *T2) error`，参与编码的类型和字段必须导出。

这套写法适合学习 RPC 原理；新项目通常更适合采用 gRPC 或公司统一框架。
