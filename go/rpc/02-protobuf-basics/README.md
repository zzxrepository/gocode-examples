# gRPC教程-proto文件（一）

> 书店场景：先用 Proto 定义“根据 ISBN 查询图书”的请求和响应契约；本目录保留通用 `simple.proto`，重点是消息定义与代码生成机制。

这个示例只演示 protobuf 消息定义和 Go 代码生成，不包含 gRPC 服务。

## 本节先懂这些

**Proto（Protocol Buffers）是一份接口说明书。**调用前，图书目录服务和订单服务必须先约定：请求里有哪个字段（如 ISBN），响应里有哪个字段（如书名、价格）。`.proto` 就是这份约定的单一来源。

它主要定义两类东西：

- `message`：一条消息的数据结构，例如“查询图书请求”和“图书详情响应”。
- `service` / `rpc`：有哪些远程方法，例如 `GetBook(BookRequest) returns (BookInfo)`。

`protoc` 会把 `.proto` 生成 Go 类型和序列化代码，因此不需要手写 JSON 编解码，也不会让客户端和服务端各自维护一份容易漂移的结构体。Proto 的字段号（例如 `string isbn = 1` 中的 `1`）是线上兼容性的身份标识；字段改名通常安全，但已发布的字段号不能拿来表示另一个含义。

这一节只解决“双方如何说同一种语言”；下一节才让 gRPC 用这份契约建立真正的远程调用。

## 生成 proto 代码

```bash
protoc --go_out=. --go_opt=paths=source_relative proto/simple.proto
```
