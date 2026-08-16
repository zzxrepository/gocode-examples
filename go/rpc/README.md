# RPC：从 Go 标准库到生产框架

这是一条按难度重新编排的 RPC 学习路径。所有说明统一使用「在线书店」：用户在书店下单，订单服务需要调用图书目录、库存、会员和优惠券等下游服务。

RPC（Remote Procedure Call，远程过程调用）解决的是：让调用方用接近本地函数的写法调用另一台机器上的程序。一次调用背后仍然包含网络连接、请求编码、服务定位、超时、错误处理和响应解码。

```text
书店 Web / 订单服务
        |  调用远端方法（看似本地函数）
        v
图书目录 / 库存 / 会员 / 优惠券服务
```

## 学习顺序

| 顺序 | 目录 | 书店中的例子 | 学到什么 |
| --- | --- | --- | --- |
| 01 | [01-native-go-rpc](./01-native-go-rpc) | 查询一本书的详情 | RPC 最小闭环：注册、监听、连接、调用、编码 |
| 02 | [02-protobuf-basics](./02-protobuf-basics) | 约定图书请求和响应 | 用 `.proto` 定义跨语言接口契约 |
| 03 | [03-grpc-unary](./03-grpc-unary) | 查询一本书 | 一元 RPC：一次请求、一次响应 |
| 04 | [04-grpc-deadlines](./04-grpc-deadlines) | 库存查询不能无限等待 | `context`、超时、取消与标准错误码 |
| 05 | [05-grpc-server-stream](./05-grpc-server-stream) | 按页持续返回图书搜索结果 | 服务端流式 RPC |
| 06 | [06-grpc-client-stream](./06-grpc-client-stream) | 批量上传盘点记录 | 客户端流式 RPC |
| 07 | [07-grpc-bidirectional-stream](./07-grpc-bidirectional-stream) | 客服实时咨询图书库存 | 双向流式 RPC |
| 08 | [08-grpc-middleware](./08-grpc-middleware) | 所有下游调用统一鉴权和记录日志 | TLS、认证、日志、panic 恢复、拦截器 |
| 09 | [09-grpc-security](./09-grpc-security) | 订单服务安全地调用库存服务 | TLS 与 metadata token |
| 10 | [10-grpc-proto-validators](./10-grpc-proto-validators) | 下单参数先校验再进入业务 | Proto 字段校验 |
| 11 | [11-grpc-gateway](./11-grpc-gateway) | 浏览器用 HTTP/JSON，内部用 gRPC | gRPC-Gateway 与 HTTP API |
| 12 | [12-dirpc-multi-protocol](./12-dirpc-multi-protocol) | 在公司的统一框架中调用多种下游 | 服务发现、治理与 HTTP/gRPC/Thrift 适配 |

`02` 到 `11` 都是独立 Go module；进入对应目录运行命令即可。`01` 只使用 Go 标准库，`12` 是脱离内部依赖的 mock 代码，专门用于阅读调用形状。

## 运行前提

当前复制进来的 gRPC 子模块在各自的 `go.mod` 中声明了 `go 1.26.5`。请使用 Go 1.26.5 或更新版本运行这些章节；`01` 和 `12` 只依赖标准库，可单独运行。

## 先建立一张全景图

| 层次 | 代表内容 | 重点 |
| --- | --- | --- |
| RPC 本质 | `01-native-go-rpc` | 把方法名、参数和结果跨网络传过去 |
| 接口契约 | Proto / IDL | 调用双方必须对消息字段和方法定义达成一致 |
| 通信实现 | gRPC、HTTP、Thrift | 分别规定传输、序列化和调用协议 |
| 工程治理 | deadline、拦截器、TLS、校验、gateway | 让调用在生产环境可控、可观测、安全 |
| 公司框架 | dirpc | 统一封装服务发现、连接复用、限流、熔断、追踪等能力 |

## 推荐的阅读方法

先跑 `01`，亲眼看到客户端远程调用服务端；再顺序学习 `02` 到 `07`，理解 gRPC 的接口和四种通信模式；最后学习 `08` 到 `12`，把超时、安全、校验和公司项目中的调用模式串起来。

每一个远程调用都应优先检查三件事：是否传递了 `context.Context`、是否设置了超时、是否同时处理了网络错误和业务错误。

更完整的“概念—示例—生产注意点”对应关系见 [KNOWLEDGE_MAP.md](./KNOWLEDGE_MAP.md)。
