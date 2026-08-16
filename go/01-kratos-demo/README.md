# Blog Microservices Demo

这是一个前后端分离的博客微服务 Demo。当前目录是独立的 Git worktree，检出 `kratos-microservices` 分支（标签 `v2-kratos-microservices`）；里面的每个服务都是独立 Go 项目。

## 服务

- `gateway`：外部统一入口，暴露 `/api/v1/*`
- `user-service`：用户注册、登录、JWT 生成和校验
- `post-service`：文章创建、列表、详情、更新、删除

## 版本说明

本仓库保留两个可同时打开的学习项目：

- [`../01-gin-demo`](../01-gin-demo)：基础版 Gin 单体应用，适合先理解登录、JWT、CRUD、MySQL、Redis 的最小闭环。
- 当前目录：Kratos 微服务版，把网关、用户服务、文章服务拆成独立 Go 项目，适合理解服务拆分、配置隔离、服务间 HTTP 调用和限流。

两个目录各自有独立工作区，因此无需在其中执行 `git switch`。单体版上的未提交学习改动也不会影响当前微服务目录。

当前目录使用 Go 1.26.5：每个服务的 `go.mod` 声明 `go 1.26.0` 与 `toolchain go1.26.5`，根 `go.work` 和 `Makefile` 也使用同一版本。

如需确认当前微服务源码来源：

```bash
git branch --show-current  # kratos-microservices
git describe --tags --exact-match # v2-kratos-microservices
```

## 目录结构

```text
gin-demo/
├── gateway/
│   ├── go.mod
│   ├── configs/
│   │   └── local.yaml
│   ├── cmd/
│   │   └── main.go
│   └── internal/
├── user-service/
│   ├── go.mod
│   ├── configs/
│   │   └── local.yaml
│   ├── migrations/
│   │   └── 001_users.sql
│   ├── cmd/
│   │   └── main.go
│   └── internal/
├── post-service/
│   ├── go.mod
│   ├── configs/
│   │   └── local.yaml
│   ├── migrations/
│   │   └── 001_posts.sql
│   ├── cmd/
│   │   └── main.go
│   └── internal/
├── api/
│   └── http.md
├── scripts/
│   ├── docker-compose.yml
│   ├── env.example
│   └── run-all.sh
├── go.work
├── Makefile
└── README.md
```

这种结构更接近真实微服务项目：

- 每个服务有自己的 `go.mod`
- 每个服务有自己的 `configs/local.yaml`
- 每个服务只读取自己的配置
- 每个服务的私有代码放在自己的 `internal`
- 根目录只负责本地编排、Docker、SQL 和文档

`go.work` 只是为了本地开发方便，把三个独立 Go module 放进同一个 workspace。真实公司里也可以把这三个目录拆成三个仓库。

## 调用关系

```text
frontend / curl
  -> gateway                 :8080
      -> user-service         :8081
      -> post-service         :8082

user-service
  -> MySQL users 表
  -> Redis session

post-service
  -> MySQL posts 表
```

为了降低本地启动成本，`user-service` 和 `post-service` 暂时使用同一个 MySQL 实例，但它们只操作自己负责的表，并且初始化 SQL 也放在各自服务的 `migrations/` 目录下。

这里刻意没有在 `posts.user_id` 上加数据库外键，因为在微服务里 `post-service` 不应该直接依赖 `user-service` 的表结构。它只保存 `user_id`，真正复杂的用户信息查询应该通过服务调用或事件同步来做。

## 配置

每个服务都有自己的配置文件：

```text
gateway/configs/local.yaml
user-service/configs/local.yaml
post-service/configs/local.yaml
```

默认端口：

```text
gateway:      :8080
user-service: :8081
post-service: :8082
MySQL:        127.0.0.1:3306
Redis:        127.0.0.1:6379
```

如果要使用其他配置文件，可以在根目录通过 Makefile 变量指定：

```bash
GATEWAY_CONFIG=configs/local.yaml USER_CONFIG=configs/local.yaml POST_CONFIG=configs/local.yaml make run-all
```

注意：这些配置路径是相对各自服务目录的。例如 `GATEWAY_CONFIG=configs/local.yaml` 指的是 `gateway/configs/local.yaml`。

## 限流

当前 Demo 有两层限流：

```text
gateway
  -> 对所有外部请求做 IP 级限流

user-service
  -> 对注册和登录做更严格的 IP 级限流
```

默认参数分别在：

```text
gateway/configs/local.yaml
user-service/configs/local.yaml
```

压测时可以调小 `gateway/configs/local.yaml`：

```yaml
rate_limit:
  rps: 2
  burst: 2
```

这个限流器是进程内限流，适合本地学习和单实例压测。多实例生产环境需要改成网关层限流，或用 Redis / 专门限流组件做分布式限流。

## 本地启动 MySQL 和 Redis

确保本机已安装 Docker，然后在根目录执行：

```bash
make docker-up
```

连接 MySQL：

```bash
docker exec -it blog-demo-mysql mysql -ublog -pblog123 blog_demo
```

连接 Redis：

```bash
docker exec -it blog-demo-redis redis-cli
```

停止本地依赖：

```bash
make docker-down
```

## 启动服务

整理依赖：

```bash
make tidy
```

启动所有服务：

```bash
make run-all
```

也可以单独启动：

```bash
make run-user
make run-post
make run-gateway
```

健康检查：

```bash
curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8081/health
curl http://127.0.0.1:8082/health
```

接口调用示例见 `api/http.md`。

## 测试

```bash
make test
```

注意：根目录现在只是 workspace，不是单独的 Go module，所以不要在根目录直接执行 `go test ./...`。如果不用 Makefile，可以执行：

```bash
go test ./gateway/... ./user-service/... ./post-service/...
```
