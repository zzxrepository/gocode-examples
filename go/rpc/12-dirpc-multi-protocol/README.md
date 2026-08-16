# 12：生产环境的统一 RPC 框架——dirpc 调用形状

这一节模拟公司内部统一 RPC 框架的**客户端侧**使用方式，不是从零实现网络协议。真实的 dirpc 会在 HTTP、gRPC、Thrift 等通信库上统一封装服务发现、连接复用、负载均衡、超时、限流、熔断、链路追踪和指标。

为了脱离广告业务，所有示例统一为在线书店的订单服务：

| 文件 | 订单服务调用的下游 | 协议与重点 |
| --- | --- | --- |
| `01_http_typed_demo.go` | 图书目录服务 | IDL 生成的强类型 HTTP 请求和响应 |
| `02_http_raw_demo.go` | 优惠券服务 | 原生 HTTP 请求、body 与 header/trace 透传 |
| `03_grpc_demo.go` | 库存服务 | gRPC/Proto 风格的批量查询 |
| `04_thrift_demo.go` | 历史会员特征服务 | Thrift 风格的特征 key 与显式 trace 参数 |

## 无论协议如何，业务侧都是同一套路

```text
启动阶段：NewClient("disf!服务名") → 进程内复用 client
请求阶段：client.Method(ctx, req) → 检查 transport error → 检查业务错误码
```

服务名不是固定 IP。`disf!book-catalog` 表示“由服务发现系统解析图书目录服务的可用实例”。真实框架会选择一个实例并管理连接；所以 client 一般初始化一次、全局复用，而不在每次订单请求中重新创建。

`ctx` 是治理能力的入口：它应携带 deadline、取消信号、trace ID 和调用 metadata。HTTP 原生转发场景需要显式地把上游 header 放回派生出的 `ctx`；Thrift 的某些历史接口则把 trace 作为方法参数传递。

## 本节先懂这些

前面几节学习的是“怎样用某一种协议完成 RPC”；这里学习的是“公司为什么还要在协议之上再包一层框架”。订单服务不应该关心图书目录今天有 3 个实例还是 30 个实例，也不应该在每个调用点重复实现选机器、超时、监控和熔断。

dirpc 这类框架把职责分开：业务代码只表达“调用哪个服务、传什么请求”；框架负责把服务名解析为实例列表、选择实例、复用连接、记录指标，并在下游异常时执行超时、限流或熔断策略。因此 `NewClient("disf!book-catalog")` 通常只在启动阶段做一次，得到的 client 由整个进程复用。

四个协议的核心差异在接口契约和编码方式，而不是业务调用思想：

- **强类型 HTTP / gRPC**：IDL 生成请求和响应，字段错误尽量在编译期发现。
- **原生 HTTP**：自己管理 path、body 和 header，适合转发或没有 IDL 的服务；尤其要记得 trace、认证等 header 透传。
- **Thrift**：同样是 IDL RPC，但历史服务可能有自己的参数和 trace 形状。

无论哪一种，都要先处理 transport error（连接失败、超时、熔断），再处理业务结果（如“优惠券不可用”）。框架不能消灭故障，只能让故障变得可控、可观测。

## 运行

这些文件是 mock，不依赖内部包；文件头使用 `//go:build ignore`，不会参与项目构建。直接指定文件仍可以运行：

```bash
go run 01_http_typed_demo.go
go run 02_http_raw_demo.go
go run 03_grpc_demo.go
go run 04_thrift_demo.go
```

真实项目不需要手写 mock 的 `Client` 和请求类型，而是导入对应 IDL 生成包和公司 RPC SDK。
