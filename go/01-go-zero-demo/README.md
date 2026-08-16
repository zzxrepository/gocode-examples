# go-zero 三服务入门示例

这个示例专门服务于 go-zero 微服务教程。它刻意把业务做小：用户注册、登录、令牌校验，以及文章 CRUD；重点是观察 API 网关如何把 REST 请求转换为两次不同的 zRPC 调用。

## 架构

~~~text
curl / browser
      |
      v
gateway :8180  (goctl API 生成的 Handler / Logic)
      |                         |
      | zRPC                    | zRPC
      v                         v
user :9181                 post :9182
令牌与用户状态                文章状态
~~~

网关的 `gateway.api` 是外部 REST 契约；`user/user.proto` 和 `post/post.proto` 是内部 gRPC 契约。三者都提交到仓库，执行 `make generate` 可重新生成 go-zero 的骨架代码。

示例使用 MySQL 保存用户和文章，Redis 保存 JWT 对应的登录会话。它们是完整业务闭环的基础设施，但不是本教程的讲解重点：读者应把注意力放在 `ServiceContext` 如何注入依赖、Gateway 如何调用 RPC Client，以及服务如何只访问自己拥有的数据。

## 目录

~~~text
01-go-zero-demo/
├── gateway/
│   ├── gateway.api               # REST DSL
│   ├── etc/gateway.yaml          # 直连 User/Post RPC 的配置
│   └── internal/{handler,logic,svc,types}
├── user/
│   ├── user.proto                # 用户 RPC 契约
│   └── internal/{logic,server,svc}
├── post/
│   ├── post.proto                # 文章 RPC 契约
│   └── internal/{logic,server,svc}
├── Makefile
└── go.mod
~~~

## 环境与启动

本项目不修改终端全局 Go 版本。Makefile 固定使用 `/Users/mmzhang/DevTools/SDK/go/1.26.5`，并显式设置 `GOROOT`，避免继承公司项目使用的 Go 1.22 环境。

~~~bash
make docker-up
make test
make run-user
make run-post
make run-gateway
~~~

也可以使用 `make run-all`；它适合快速观察输出，停止时需自行终止三个进程。Docker 使用宿主机端口 MySQL `3307`、Redis `6380`，因此可与 Kratos 示例的 `3306`、`6379` 同时运行。

三个服务已启动后，可执行一次完整的 REST → User RPC → Post RPC 冒烟验证：

~~~bash
make smoke
~~~

该命令会生成一个带时间戳的测试账号，完成注册、携带令牌创建文章并读取文章列表。默认访问 `http://127.0.0.1:8180`；如需调整网关地址，可设置 `GOZERO_GATEWAY_ADDR`。

## API smoke test

~~~bash
curl -X POST http://localhost:8180/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","password":"secret"}'

curl -X POST http://localhost:8180/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","password":"secret"}'

curl http://localhost:8180/api/v1/posts
~~~

将登录响应中的 `token` 作为 `Authorization: Bearer <token>` 发送，便可创建、修改或删除文章。

## 生成命令

安装 `goctl`、`protoc`、`protoc-gen-go` 和 `protoc-gen-go-grpc` 后执行：

~~~bash
make generate
~~~

生成器会重建带有 “Code scaffolded by goctl” 标记的文件。业务实现放在 Logic、ServiceContext 等文件中；重新生成后应使用 Git diff 审核并恢复有意修改的实现。
