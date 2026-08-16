# Gin Blog Demo：从启动到一次请求的分层原理

这是一个使用 Gin、MySQL、Redis 和 JWT 实现的最小博客后端。它的目标不只是提供几个接口，而是演示一个 Go HTTP 服务如何把“网络请求、业务规则、数据存储”拆开组织。

项目提供用户注册、登录、JWT 鉴权和文章的增删改查。MySQL 保存用户与文章；Redis 保存有效登录会话，使服务能够在 JWT 自身未过期时主动让会话失效。

## 先建立整体模型

这个项目可以按两条方向理解：

```text
启动时：main 组装依赖

配置 -> MySQL / Redis 客户端 -> Repository -> Service -> Server / Controller -> 启动 Gin

收到请求时：请求向内，结果向外

HTTP Request -> Engine 路由树 -> Middleware -> Controller -> Service -> Repository -> MySQL / Redis
HTTP Response <- Engine 路由树 <- Middleware <- Controller <- Service <- Repository <- MySQL / Redis
```

`controller` 是 HTTP 控制层：接收 HTTP 请求、读取参数、调用业务用例、选择状态码并返回 JSON。它是许多 Go 项目中常见的 `handler` 命名的同一类职责；本项目统一采用 `controller` 命名。

需要区分两件事：

- `cmd/api/main.go` 的工作是依赖组装（composition root）：创建对象并把依赖交给下游。
- 真正的业务调用发生在请求到来之后：Controller 调用 Service，Service 调用 Repository。

因此，准确的依赖关系是：Repository 被注入 Service，Service 被注入 Controller；并不是 Repository 直接交给 Controller 使用。

## 目录与职责

```text
gin-demo/
├── cmd/api/main.go             # 程序入口：加载配置并组装全部依赖
├── internal/config/            # 从环境变量读取配置
├── internal/server/            # 创建 Gin Engine、注册路由和中间件
├── internal/controller/        # HTTP 控制层：绑定请求、调用 Service、返回响应
├── internal/middleware/        # 横切逻辑：JWT 认证与当前用户写入 Context
├── internal/service/           # 业务用例：注册、登录、文章操作
├── internal/repository/        # 数据访问：MySQL 查询与 Redis 会话读写
├── internal/model/             # 领域数据、请求/响应 DTO 与统一响应结构
├── api/                        # SQL 与 HTTP 调用示例
└── scripts/                    # 本地 MySQL、Redis 的 Docker 配置
```

`internal` 是 Go 的访问边界：项目外的其他 Go module 不能导入其中的包。它适合放服务内部实现，避免内部代码被外部项目依赖。

## 一、程序如何启动

入口在 [cmd/api/main.go](cmd/api/main.go)。它依次完成以下工作。

### 1. 读取配置

`config.Load()` 从环境变量读取 HTTP 地址、MySQL DSN、Redis 地址、JWT 密钥和 JWT 有效期；若未配置则使用本地开发默认值。

```text
HTTP_ADDR
MYSQL_DSN
REDIS_ADDR / REDIS_PASSWORD / REDIS_DB
JWT_SECRET / JWT_EXPIRE_HOURS
```

配置模块只负责读取和提供配置，不参与业务判断。

### 2. 创建基础设施客户端

`repository.NewMySQL` 创建 `*sql.DB`，设置连接池参数并通过 `Ping` 尽早确认 MySQL 可用。

`repository.NewRedis` 创建 `*redis.Client`，同样通过 `Ping` 验证连接。

这两个对象是连接池或客户端，不是具体的业务数据访问对象。程序退出时由 `defer db.Close()` 和 `defer rdb.Close()` 释放资源。

### 3. 创建 Repository

Repository 将底层存储能力包装成与业务有关的方法：

```text
*sql.DB        -> UserRepository、PostRepository
*redis.Client  -> SessionRepository
```

例如：

- `UserRepository.FindByUsername` 查询用户；
- `PostRepository.Update` 通过 `id` 和 `user_id` 共同限制更新范围；
- `SessionRepository.SaveToken` 将登录 Token 以 TTL 写入 Redis。

Repository 内可以出现 SQL、Redis Key 和存储驱动相关代码；这些细节不应散落在 Controller 或 Service 中。

### 4. 创建 Service

Service 表示业务用例，并接收它所依赖的 Repository：

```text
UserRepository + SessionRepository + JWT 配置 -> AuthService
PostRepository                                -> PostService
```

`AuthService` 负责密码哈希、密码核验、JWT 签发和 Redis 会话校验；`PostService` 负责把“当前用户”和文章操作组合成文章用例。

### 5. 创建 Server 并启动

最后，`server.New(cfg, authSvc, postSvc)` 创建 Gin Engine、Controller 和路由表；`srv.Run()` 才开始监听 HTTP 端口。

可以把 `main.go` 看作装配线：它知道“谁依赖谁”，但不承担注册、登录、查询文章等具体业务。

## 二、Gin 的 Engine、RouterGroup 与路由注册

[internal/server/server.go](internal/server/server.go) 是 HTTP 入口的配置层。

### Engine 是什么

`engine := gin.New()` 创建的是一个 `*gin.Engine`。它不是一个单独的“路由组对象”，而是 Gin 的总调度器：持有路由树、全局中间件、配置和 `sync.Pool` 管理的 `gin.Context`。

Gin 的 `Engine` 在结构上**内嵌**了一个根 `RouterGroup`。Gin 1.10.0 的初始化可以抽象为：

```text
Engine
├── root RouterGroup
│   ├── basePath: "/"
│   ├── Handlers: nil
│   └── engine: 指回这个 Engine
├── trees: 按 HTTP Method 划分的路由树
├── pool: 复用 gin.Context
└── 路由、代理、重定向等运行配置
```

下面是 Gin 1.10.0 的对应源码节选（`github.com/gin-gonic/gin/gin.go`）。为突出主线，省略了重定向、可信代理、模板等与路由注册无关的字段；注释说明这些字段在本项目中的意义。

```go
type Engine struct {
	RouterGroup // 匿名嵌入：Engine 自动拥有 RouterGroup 的 Use、Group、GET、POST 等方法

	// ... 重定向、代理、渲染等配置
	trees methodTrees // 按 HTTP Method 保存路由树，例如 GET 树、POST 树
	pool  sync.Pool   // 复用每次请求所需的 *gin.Context
}

func New(opts ...OptionFunc) *Engine {
	engine := &Engine{
		RouterGroup: RouterGroup{
			Handlers: nil, // 根路由组初始没有中间件
			basePath: "/", // 根路由组的路径前缀
			root:     true,
		},
		trees: make(methodTrees, 0, 9), // 初始为空，注册路由时按 Method 建树
		// ... 其他默认配置
	}

	engine.RouterGroup.engine = engine // 根组回指同一个 Engine
	engine.pool.New = func() any {
		return engine.allocateContext(engine.maxParams) // 为请求创建可复用 Context
	}
	return engine.With(opts...)
}
```

这里最重要的是匿名嵌入 `RouterGroup`。在 Go 中，嵌入字段的方法会提升到外层类型，所以 `engine.Use(...)`、`engine.Group(...)`、`engine.GET(...)` 实际上是对 `engine.RouterGroup` 调用相应方法。根路由组的 `engine` 字段又指回外层 Engine，因此不论从根组还是任意子组注册路由，最终都会写到同一份 `engine.trees` 中。

因此 `engine.GET(...)`、`engine.Use(...)` 和 `engine.Group(...)` 能直接调用：这些方法来自它内嵌的根 `RouterGroup`。`gin.New()` 创建的是**没有任何中间件**的 Engine；若使用 `gin.Default()`，它才会在 `New()` 后自动添加 `gin.Logger()` 和 `gin.Recovery()`。

Gin 的 `Default()` 对应实现很直接：

```go
func Default(opts ...OptionFunc) *Engine {
	engine := New()                 // 先创建空 Engine
	engine.Use(Logger(), Recovery()) // 再追加两个全局中间件
	return engine.With(opts...)
}
```

### `Use` 做了什么

项目显式调用：

```go
engine.Use(gin.Logger(), gin.Recovery())
```

这里的 `engine.Use` 实际调用根 `RouterGroup.Use`。它会把两个 `gin.HandlerFunc` 追加到根组的 `Handlers` 链中；此时没有注册具体 URL，也没有立刻执行中间件。

对应的关键源码节选在 Gin 1.10.0 的 `routergroup.go`：

```go
type RouterGroup struct {
	Handlers HandlersChain // 当前组已经积累的中间件处理函数
	basePath string        // 当前组的绝对路径前缀
	engine   *Engine       // 所属 Engine；所有子组共享它
	root     bool          // 只有嵌在 Engine 中的根组为 true
}

func (group *RouterGroup) Use(middleware ...HandlerFunc) IRoutes {
	group.Handlers = append(group.Handlers, middleware...)
	// 这里只修改组的“注册期状态”，不会执行 Logger 或 Recovery。
	return group.returnObj()
}
```

因此，本项目调用 `engine.Use(gin.Logger(), gin.Recovery())` 后，根组的状态可以理解为：

```text
engine.RouterGroup.Handlers = [Logger, Recovery]
```

`Use` 必须发生在对应路由注册之前。因为 Gin 在注册 `GET`、`POST` 等路由时，会把当前 `Handlers` 的快照与路由处理函数合并进路由树；后续才添加的中间件不会回写到已经注册完成的旧路由。

后续注册某条路由时，Gin 会将“所属组已经积累的中间件”与“该路由自己的处理函数”拼成一个最终处理链。对于 `GET /health`，其链路大致为：

```text
[Logger, Recovery, health endpoint function]
```

全局中间件只需注册一次，但会进入从根组派生出的每一条路由。

### `Group` 做了什么

```go
api := engine.Group("/api/v1")
posts := api.Group("/posts")
protected := posts.Group("")
protected.Use(middleware.Auth(authSvc))
```

`Group` 不会复制或创建一个新的 Engine，也不会创建一套新的 HTTP 服务。它只是创建一个轻量的 `RouterGroup` 描述对象，包含三项关键信息：

```text
父组 basePath + relativePath  -> 子组 basePath
父组 Handlers + Group 参数的 handlers -> 子组 Handlers
父组 engine                         -> 同一个 Engine
```

对应的关键源码节选如下：

```go
func (group *RouterGroup) Group(relativePath string, handlers ...HandlerFunc) *RouterGroup {
	return &RouterGroup{
		Handlers: group.combineHandlers(handlers),
		// 先复制父组 Handlers，再追加 Group(...) 传入的中间件。

		basePath: group.calculateAbsolutePath(relativePath),
		// 例如父组 "/api/v1" 与 relativePath "/posts" 组合成 "/api/v1/posts"。

		engine: group.engine,
		// 直接复用父组的 Engine，不会新建 Engine 或新的监听端口。
	}
}

func (group *RouterGroup) combineHandlers(handlers HandlersChain) HandlersChain {
	finalSize := len(group.Handlers) + len(handlers)
	mergedHandlers := make(HandlersChain, finalSize)
	copy(mergedHandlers, group.Handlers)
	copy(mergedHandlers[len(group.Handlers):], handlers)
	return mergedHandlers
}

func (group *RouterGroup) calculateAbsolutePath(relativePath string) string {
	return joinPaths(group.basePath, relativePath)
}
```

注意，`Group` 会复制当前处理链到新切片中。之后对 `protected` 调用 `Use(Auth)`，只会改变 `protected.Handlers`，不会反向改变 `posts.Handlers`；这正是公开读接口与需要登录的写接口能够共用 `/posts` 前缀、却拥有不同鉴权要求的原因。

上述分组的结果是：

| 分组变量 | `basePath` | 已继承的中间件 |
| --- | --- | --- |
| `engine` | `/` | `Logger`、`Recovery` |
| `api` | `/api/v1` | `Logger`、`Recovery` |
| `posts` | `/api/v1/posts` | `Logger`、`Recovery` |
| `protected` | `/api/v1/posts` | `Logger`、`Recovery`、`Auth` |

所以 `protected.POST("", postController.Create)` 注册的绝对路径是 `/api/v1/posts`，最终处理链为：

```text
[Logger, Recovery, Auth, PostController.Create]
```

而 `posts.GET("", postController.List)` 没有使用 `protected`，因此其处理链没有 `Auth`，可以匿名读取文章列表。

### `GET`、`POST` 最终如何注册

`RouterGroup.GET`、`POST` 等只是 `Handle` 的快捷方法。以 `posts.GET("/:id", postController.Get)` 为例，Gin 内部执行逻辑是：

```text
relativePath "/:id"
  -> 根据 posts.basePath 计算绝对路径 "/api/v1/posts/:id"
  -> 合并 posts 的中间件链与 PostController.Get
  -> 调用 engine.addRoute("GET", absolutePath, handlers)
  -> 写入 Engine 中 GET 方法对应的路由树
```

Gin 为不同 HTTP Method 维护独立路由树。请求到来时，它先按 `Request.Method` 找到对应树，再按 URL 路径匹配节点，并把 `:id` 这类路径参数写入 `gin.Context.Params`。Controller 中的 `c.Param("id")` 就是从这里读取参数。

对应的关键源码节选可以把“快捷方法 → 合并处理链 → 写入路由树”连起来看：

```go
// routergroup.go
func (group *RouterGroup) GET(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodGet, relativePath, handlers)
}

func (group *RouterGroup) handle(httpMethod, relativePath string, handlers HandlersChain) IRoutes {
	absolutePath := group.calculateAbsolutePath(relativePath)
	// "/api/v1/posts" + "/:id" -> "/api/v1/posts/:id"

	handlers = group.combineHandlers(handlers)
	// [Logger, Recovery] + [PostController.Get]
	// -> [Logger, Recovery, PostController.Get]

	group.engine.addRoute(httpMethod, absolutePath, handlers)
	return group.returnObj()
}

// gin.go
func (engine *Engine) addRoute(method, path string, handlers HandlersChain) {
	root := engine.trees.get(method) // 取得该 Method 的根节点，如 GET 根节点
	if root == nil {
		root = new(node)
		root.fullPath = "/"
		engine.trees = append(engine.trees, methodTree{method: method, root: root})
	}
	root.addRoute(path, handlers) // 将路径和已经合并好的链保存到基数树节点
}
```

这里的关键是：中间件不是在请求来临时再根据分组关系临时查找的。路由注册阶段就已经把每条路由的完整 `HandlersChain` 计算并保存好了。请求阶段主要做两件事：匹配路由树节点，执行节点内保存的处理链。

### 项目的路由表

项目的 [internal/server/server.go](internal/server/server.go) 按以下顺序组装。注释标出每次调用对 Engine 或 RouterGroup 状态的影响：

```go
engine := gin.New() // 创建一个 Engine，其中已经有 basePath 为 "/" 的根 RouterGroup
engine.Use(gin.Logger(), gin.Recovery())
// 根组 Handlers: [Logger, Recovery]

authController := controller.NewAuthController(authSvc)
postController := controller.NewPostController(postSvc)

engine.GET("/health", func(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
})
// 注册 GET /health
// 最终链: [Logger, Recovery, 匿名健康检查函数]

api := engine.Group("/api/v1")
// api.basePath: "/api/v1"
// api.Handlers: [Logger, Recovery]
api.POST("/auth/register", authController.Register)
api.POST("/auth/login", authController.Login)

posts := api.Group("/posts")
// posts.basePath: "/api/v1/posts"
// posts.Handlers: [Logger, Recovery]
posts.GET("", postController.List)
posts.GET("/:id", postController.Get)

protected := posts.Group("")
// protected.basePath 仍是 "/api/v1/posts"
// protected.Handlers 初始为 [Logger, Recovery]
protected.Use(middleware.Auth(authSvc))
// protected.Handlers 变为 [Logger, Recovery, Auth]
protected.POST("", postController.Create)
protected.PUT("/:id", postController.Update)
protected.DELETE("/:id", postController.Delete)
```

上述代码与项目实现一致；为阅读路由状态变化，省略了 `Server` 返回与其他无关代码。

```text
gin.Logger()    记录访问日志
gin.Recovery()  捕获 panic，避免单个请求导致进程退出
```

在注册路由前，Server 根据 Service 创建 Controller：

```text
AuthService -> AuthController
PostService -> PostController
```

路由分为公开接口和受保护接口：

| 路由 | 方法 | 是否鉴权 | Controller |
| --- | --- | --- | --- |
| `/health` | GET | 否 | 内联健康检查 |
| `/api/v1/auth/register` | POST | 否 | `AuthController.Register` |
| `/api/v1/auth/login` | POST | 否 | `AuthController.Login` |
| `/api/v1/posts` | GET | 否 | `PostController.List` |
| `/api/v1/posts/:id` | GET | 否 | `PostController.Get` |
| `/api/v1/posts` | POST | 是 | `PostController.Create` |
| `/api/v1/posts/:id` | PUT | 是 | `PostController.Update` |
| `/api/v1/posts/:id` | DELETE | 是 | `PostController.Delete` |

受保护的文章路由挂载 `middleware.Auth(authSvc)`。中间件先于对应 Controller 执行：认证失败时调用 `AbortWithStatusJSON` 直接返回 401；认证成功时把 `user_id` 写进 `gin.Context`，后续 Controller 可通过 `middleware.CurrentUserID(c)` 获取它。

## 三、`Run` 如何连接到 Go 标准库 HTTP 服务

项目的 `srv.Run()` 最终调用：

```go
engine.Run(s.cfg.HTTPAddr)
```

Gin 1.10.0 中 `Engine.Run` 的关键源码节选如下：

```go
func (engine *Engine) Run(addr ...string) (err error) {
	defer func() { debugPrintError(err) }()

	if engine.isUnsafeTrustedProxies() {
		// 仅打印“信任所有代理”的安全警告，不改变启动流程。
		debugPrint("[WARNING] You trusted all proxies ...")
	}

	address := resolveAddress(addr)
	// Run(":8080") 使用传入地址；未传入时使用 Gin 约定的默认地址。

	err = http.ListenAndServe(address, engine.Handler())
	// 将 Gin Engine 作为标准库 Handler 交给 HTTP Server；这里会阻塞。
	return
}
```

`http.ListenAndServe` 的 Go 标准库关键源码节选（`net/http/server.go`）进一步说明了它做的事：

```go
func ListenAndServe(addr string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	// 这里确实创建了 http.Server 结构体，但 ListenAndServe 本身不是“对象”。
	return server.ListenAndServe()
}

func (srv *Server) ListenAndServe() error {
	if srv.shuttingDown() {
		return ErrServerClosed
	}
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	// 创建 TCP 监听器；若端口已被占用，错误会在这里返回。
	if err != nil {
		return err
	}
	defer ln.Close()
	return srv.Serve(ln)
	// 循环接受连接、解析 HTTP 请求，并把请求交给 srv.Handler。
}
```

因此，它的本质是调用 Go 标准库函数，完成以下工作：

1. 在 `address` 上创建并监听 TCP socket；
2. 接收 HTTP 连接和请求；
3. 对每个请求调用传入的 `http.Handler`；
4. 直到服务关闭或发生监听错误，调用才返回。

`engine.Handler()` 在普通 HTTP/1.1、HTTP/2 TLS 场景下返回 `engine` 本身。原因是 `*gin.Engine` 实现了标准库的接口：

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```

也就是说，Gin 并没有替换 Go 的 HTTP 服务器；它只是实现了 `http.Handler`，成为标准库 HTTP 服务器的一种请求处理器。

Gin 中对应的 `Handler` 与 `ServeHTTP` 关键源码节选如下：

```go
func (engine *Engine) Handler() http.Handler {
	if !engine.UseH2C {
		return engine // *gin.Engine 本身实现 http.Handler
	}
	// H2C 场景会返回一个包装后的 Handler；普通项目通常不走这里。
	return h2c.NewHandler(engine, &http2.Server{})
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := engine.pool.Get().(*Context) // 从对象池取得本次请求专用的 Context
	c.writermem.reset(w)
	c.Request = req
	c.reset()

	engine.handleHTTPRequest(c) // 路由匹配，并执行当前路由的处理链

	engine.pool.Put(c) // 请求结束后归还 Context，供后续请求复用
}
```

```text
net/http.ListenAndServe(":8080", engine)
                 │
                 │ 收到一个 HTTP 请求
                 ▼
engine.ServeHTTP(w, req)
  ├── 从 sync.Pool 取得一个 gin.Context
  ├── 调用 handleHTTPRequest
  │     ├── 按请求 Method 选择路由树
  │     ├── 按 URL Path 匹配节点和路径参数
  │     └── 执行该路由的 HandlersChain
  └── 归还 gin.Context 到对象池
```

当 `handleHTTPRequest` 匹配到例如 `POST /api/v1/posts` 时，会将注册阶段已经合并好的处理链赋给 `c.handlers`，然后 `c.Next()` 按顺序执行：`Logger -> Recovery -> Auth -> PostController.Create`。`Auth` 调用 `AbortWithStatusJSON` 时会中断后续链条，因此 Controller 不会继续执行。

这两个行为对应 Gin `Context` 的关键源码节选：

```go
func (c *Context) Next() {
	c.index++
	for c.index < int8(len(c.handlers)) {
		c.handlers[c.index](c) // 按注册顺序依次执行 Logger、Recovery、Auth、Controller
		c.index++
	}
}

func (c *Context) Abort() {
	c.index = abortIndex // 将索引跳到链尾，阻止尚未执行的后续处理函数
}

func (c *Context) AbortWithStatusJSON(code int, jsonObj any) {
	c.Abort()
	c.JSON(code, jsonObj) // 写入 401 等状态码及 JSON 响应
}
```

需要注意：`Abort` 不会中止当前正在运行的中间件函数；它的作用是阻止当前函数返回后，`c.Next()` 继续执行后面的 Auth 或 Controller。因此认证中间件在写入错误响应后必须立刻 `return`，项目中的 `middleware.Auth` 正是这样实现的。

`Engine.Run` 适合本项目这种简单启动方式。需要自定义 `http.Server`（例如设置读写超时、优雅关闭或 TLS 配置）时，可以自行创建 `http.Server{Addr: ..., Handler: engine}` 并调用 `ListenAndServe`；Engine 仍作为 Handler 使用。

## 四、一次“创建文章”请求的完整链路

以 `POST /api/v1/posts` 为例，最能体现各层协作：

```text
客户端
  │ Authorization: Bearer <JWT> + JSON Body
  ▼
Gin Router
  ▼
Auth Middleware
  ├── AuthService.ParseToken
  │     ├── 校验 JWT 签名和过期时间
  │     └── SessionRepository.TokenExists -> Redis
  └── 将 user_id 写入 Gin Context
  ▼
PostController.Create
  ├── ShouldBindJSON 绑定 CreatePostRequest
  ├── 读取当前 user_id
  └── 调用 PostService.Create
  ▼
PostService.Create
  └── 组装 model.Post，调用 PostRepository.Create
  ▼
PostRepository.Create
  ├── INSERT INTO posts ...
  └── SELECT ... 查询刚创建的数据
  ▼
PostController.Create
  └── 返回 201 JSON Response
```

这条链路中的边界如下：

- Middleware 只关心“请求是否有合法身份”。
- Controller 只关心 HTTP：JSON 是否能绑定、路径参数是否合法、应该返回什么状态码。
- Service 只关心用例：谁在创建文章、需要传递哪些业务数据。
- Repository 只关心存储：具体执行什么 SQL 或 Redis 命令。

## 五、注册与登录为何需要 MySQL、JWT 和 Redis

### 注册

`AuthController.Register` 绑定 `RegisterRequest` 后调用 `AuthService.Register`。

Service 使用 bcrypt 将明文密码计算为 `password_hash`，再交给 `UserRepository.Create` 写入 MySQL。密码哈希属于安全业务规则，因此放在 Service，而不是 Controller 或 SQL 中。

### 登录

登录流程为：

```text
LoginRequest
  -> UserRepository.FindByUsername（MySQL 查询用户）
  -> bcrypt.CompareHashAndPassword（核验密码）
  -> issueToken（生成包含 user_id、username、过期时间的 JWT）
  -> SessionRepository.SaveToken（Redis 写入 Token 和 TTL）
  -> 返回 LoginResponse{token}
```

JWT 本身能证明“这个 Token 曾被服务端签发”，却不能天然支持服务端主动注销。这里额外把 Token 存进 Redis：后续请求除了验证 JWT，还要验证 Redis 中是否仍存在完全相同的 Token。这样删除 Redis Key 或等待 TTL 到期，即可让该会话失效。

## 六、Model 在项目中的位置

`internal/model` 不等同于数据库模型。它同时承载三类数据：

- 持久化或领域实体：`User`、`Post`；
- HTTP 请求和响应的数据结构：`RegisterRequest`、`LoginRequest`、`CreatePostRequest` 等；
- 统一返回格式：`Response{Code, Message, Data}`。

这种做法适合这个小型 Demo。项目进一步变大时，通常会把数据库实体、领域对象和 HTTP DTO 分开，避免数据库字段变更直接影响 API。

## 七、依赖方向与分层规则

当前项目的依赖方向为：

```text
server / controller / middleware
            ↓
         service
            ↓
        repository
            ↓
      MySQL / Redis
```

`model` 被多个层共享，`config` 主要在启动组装阶段使用。

维护时可以遵循这些规则：

1. 不在 Controller 中编写 SQL、Redis 命令、密码哈希或 JWT 签发逻辑。
2. 不在 Repository 中读取 Gin Context、拼 HTTP 状态码或返回 JSON。
3. Service 不依赖 Gin 的 `*gin.Context`；它接收标准 `context.Context` 和明确的业务参数。
4. Middleware 不直接访问数据库，而是复用 `AuthService.ParseToken`。
5. 新增一个功能时，通常按 `model -> repository -> service -> controller -> server route` 的顺序补齐；在 `main.go` 中完成新的依赖组装。

当前实现使用具体类型指针，例如 `*repository.UserRepository` 和 `*service.AuthService`，这是小型项目中直接、易读的写法。若需要为 Service 编写单元测试或支持多种存储实现，可进一步在 Service 层定义小接口，由 Repository 实现这些接口，再注入测试替身。

## 八、启动项目

先启动本地依赖：

```bash
make docker-up
```

默认配置：

```text
MySQL: blog:blog123@tcp(127.0.0.1:3306)/blog_demo
Redis: 127.0.0.1:6379
HTTP:  :8080
```

启动 API：

```bash
go mod tidy
make run
```

本项目使用本机 Go 1.26.5。`go.mod` 中的 `go 1.26.0` 表示语言与模块语义基线，`toolchain go1.26.5` 指定实际工具链；`Makefile` 也固定使用 `/Users/mmzhang/DevTools/SDK/go/1.26.5/bin/go`，因此直接执行 `make run` 或 `make test` 即可，无须手动切换 Go 环境。

健康检查：

```bash
curl http://127.0.0.1:8080/health
```

完整的注册、登录、文章接口调用命令见 [api/http.md](api/http.md)；数据库表结构见 [api/schema.sql](api/schema.sql)。停止本地依赖可执行：

```bash
make docker-down
```
