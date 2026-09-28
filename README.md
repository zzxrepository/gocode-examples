# gocode-examples

本仓库保存 [gocode](https://github.com/zzxrepository/gocode) 教程中需要独立运行、测试或持续演进的示例项目。文档负责解释原理与学习路径；本仓库负责保存完整源码、配置样例和运行说明。

通用示例按语言划分；消息队列示例放在 `message-queue/<中间件>/<语言>/<项目>` 下。每个示例项目都有独立的依赖和运行入口。

| 项目 | 对应主题 | 运行入口 |
| --- | --- | --- |
| `go/01-http-demo` | `net/http` 服务端、`Handler`、`ServeMux` 与分层博客 API | `go run ./cmd/api` |
| `go/01-gin-demo` | Gin 路由、中间件、Context 与分层博客 API | `go run ./cmd/api` |
| `go/01-kratos-demo` | Kratos 微服务、网关、用户服务与文章服务 | 参见项目 README |
| [`message-queue/kafka/go/01-sarama-order-demo`](message-queue/kafka/go/01-sarama-order-demo/) | Sarama 订单 HTTP 实验：同步、异步、回调、消费者组与事务 | 参见项目 README |

## 约定

- 每个一级项目自行维护 `go.mod`、README、配置样例和测试。
- 根目录不设置统一 `go.mod`；各示例保持可独立运行，避免把无关教程耦合在一起。
- 需要多模块协作的项目可以在项目根目录使用 `go.work`，例如 `go/01-kratos-demo`。
- `.cache`、构建产物、日志和本地密钥不提交；配置文件应提供不含真实密钥的示例。
- 教程引用完整项目时，应链接到本仓库中相应项目的确定提交版本，保证文章与代码可复现。
