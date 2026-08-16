# Blog Demo v1：从 `net/http` 源码理解一个 Web 服务

这是一个只使用 Go 标准库 `net/http` 的博客后端。它实现用户注册、登录、JWT 认证、Redis 会话校验和文章 CRUD；MySQL、Redis、Service、Repository 等业务分层与其他版本一致，HTTP 入口则完全建立在标准库的四个核心抽象之上：

```text
net.Listener  监听 TCP 连接
http.Server   管理连接、超时、HTTP 协议和请求分发
http.Handler  定义“如何处理一个 HTTP 请求”
http.ServeMux 根据 Method 与 URL 找到对应 Handler
```

本文以项目源代码和本机 Go 1.26.5 的 `net/http/server.go` 为线索，按照程序实际发生的顺序讲解：先注册路由和构造 Handler 链，再启动 TCP 监听，最后跟踪一条请求如何到达 Controller。

> 源码片段均保留理解主线所必需的语句；标准库中的 TLS、HTTP/2、连接状态、错误重试等无关细节会用注释或省略号说明。项目的模块声明使用 Go 1.26，Makefile 固定使用本机 Go 1.26.5。

## 阅读结构

标准库 HTTP 服务由两个阶段组成，后续内容也依照这条因果链组织：

~~~text
一、启动与注册
应用组装 -> Handler / HandlerFunc -> ServeMux 注册路由 -> 中间件包装 Handler

二、请求处理
Server 监听 TCP -> Serve 接收连接 -> c.serve 解析 HTTP ->
serverHandler 选择 Server.Handler -> ServeMux 匹配路由 -> Controller
~~~

各节分别回答一个明确问题：

| 节 | 要解决的问题 | 核心对象 |
| --- | --- | --- |
| 1 | 项目最终交给标准库的对象是什么 | `http.Server`、`ServeMux`、中间件链 |
| 2–4 | 什么对象可以处理请求，如何注册 | `Handler`、`HandlerFunc`、`Handle` |
| 5 | 如何保存路由并在请求时匹配 | `ServeMux` |
| 6 | 如何在 Handler 外面增加日志、恢复和认证 | `Middleware` |
| 7 | 谁监听端口，谁解析请求，谁选择 Handler | `Server`、`serverHandler` |
| 8–10 | 一次业务请求如何穿过 HTTP 层与业务分层 | Controller、Service、Repository |
| 11 | 如何运行和验证服务 | 配置、Docker、HTTP 接口 |

## 1. 先看本项目最终交给 `net/http` 的对象

入口 [cmd/api/main.go](cmd/api/main.go) 负责创建配置、数据库客户端、Repository、Service，然后调用：

```go
server.New(cfg, authSvc, postSvc).Run()
```

[internal/server/server.go](internal/server/server.go) 的关键逻辑是：

```go
mux := http.NewServeMux() // 创建项目私有路由器

// 注册公开路由。
mux.HandleFunc("POST /api/v1/auth/login", authController.Login)
mux.HandleFunc("GET /api/v1/posts/{id}", postController.Get)

// 为写接口单独包上认证中间件，再注册。
protected := middleware.Auth(authSvc)
mux.Handle("POST /api/v1/posts", protected(http.HandlerFunc(postController.Create)))

// 全局链：Logger(Recovery(mux))。
handler := middleware.Chain(mux, middleware.Logger, middleware.Recovery)

httpServer := &http.Server{
	Addr:              cfg.HTTPAddr,
	Handler:           handler,
	ReadHeaderTimeout: 5 * time.Second,
	ReadTimeout:       15 * time.Second,
	WriteTimeout:      15 * time.Second,
	IdleTimeout:       60 * time.Second,
}
```

所以本项目真正的对象关系是：

```text
http.Server.Handler
    │
    └── Logger(
          Recovery(
            *http.ServeMux
                ├── GET /api/v1/posts/{id} -> PostController.Get
                └── POST /api/v1/posts      -> Auth(PostController.Create)
          )
        )
```

这里有两个层次的中间件：

- `Logger`、`Recovery` 包住整个 `ServeMux`，因此对所有路由有效；
- `Auth` 只包住创建、更新、删除文章的三个 Handler，读取文章无需登录。

一次创建文章请求最终进入的顺序为：

```text
HTTP Server
  -> Logger
  -> Recovery
  -> ServeMux 路由匹配
  -> Auth
  -> PostController.Create
  -> PostService.Create
  -> PostRepository.Create
  -> MySQL
```

理解标准库 HTTP 服务时，要始终区分两个阶段：

```text
注册阶段（程序启动前）
  NewServeMux / Handle / HandleFunc / Middleware 包装

请求阶段（每个请求到来时）
  Server -> Handler.ServeHTTP -> ServeMux.ServeHTTP -> 具体 Controller
```

`Handle`、`HandleFunc` 只在注册阶段执行一次；Controller 与中间件的 `ServeHTTP` 才会随每个请求执行。

## 2. `http.Handler`：标准库 HTTP 的最小接口

标准库的核心定义位于 `net/http/server.go`：

```go
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}
```

这意味着，只要一个类型拥有：

```go
ServeHTTP(w http.ResponseWriter, r *http.Request)
```

它就是一个 HTTP Handler，可以被放进 `http.Server.Handler`、`ServeMux.Handle`，也可以被其他中间件包装。

两个参数分别代表：

| 参数 | 作用 | 常见操作 |
| --- | --- | --- |
| `http.ResponseWriter` | 向客户端构造响应 | `Header().Set`、`WriteHeader`、`Write` |
| `*http.Request` | 读取客户端请求 | `Method`、`URL`、`Header`、`Body`、`Context()` |

项目 Controller 的方法签名完全符合一个普通函数 Handler：

```go
func (ctl *PostController) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	post, err := ctl.posts.FindByID(r.Context(), id)
	if err != nil {
		writeSQLError(w, err)
		return
	}
	httpapi.WriteResponse(w, http.StatusOK, "ok", post)
}
```

但是方法或普通函数本身没有 `ServeHTTP` 方法，不能直接赋给需要 `http.Handler` 的位置。这正是 `http.HandlerFunc` 存在的原因。

## 3. `HandlerFunc`：把函数适配成接口对象

标准库的源码如下：

```go
// HandlerFunc 是一种“函数类型”。
type HandlerFunc func(ResponseWriter, *Request)

// 它为该函数类型实现了 Handler 接口。
func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	f(w, r)
}
```

这是一种 Go 中非常典型的适配器模式：没有创建额外结构体，只是把函数转换为具有 `ServeHTTP` 方法的命名函数类型。

```text
普通函数 / 方法值
func(w, r)
       │
       │ http.HandlerFunc(f)
       ▼
HandlerFunc 类型值
       │
       │ 拥有 ServeHTTP 方法
       ▼
http.Handler 接口值
```

例如本项目的受保护写接口：

```go
postController.Create
// 类型：func(http.ResponseWriter, *http.Request)

http.HandlerFunc(postController.Create)
// 类型：http.HandlerFunc，同时满足 http.Handler

protected(http.HandlerFunc(postController.Create))
// Auth 中间件返回新的 http.Handler
```

## 4. `Handle`、`HandleFunc` 与包级函数的区别

`ServeMux` 提供两个注册方法：

```go
func (mux *ServeMux) Handle(pattern string, handler Handler)
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request))
```

它们的第一个参数完全相同，都是路由 pattern；差异在第二个参数的类型：

| 方法 | 第二个参数 | 适用情况 |
| --- | --- | --- |
| `mux.Handle` | `http.Handler` | 已有一个 Handler，例如中间件包装后的结果 |
| `mux.HandleFunc` | 普通函数 `func(w, r)` | 直接把 Controller 方法或匿名函数注册为路由 |

Go 1.26.5 的关键源码说明了它们最终会汇合：

```go
func (mux *ServeMux) Handle(pattern string, handler Handler) {
	if use121 {
		mux.mux121.handle(pattern, handler)
	} else {
		mux.register(pattern, handler)
	}
}

func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	if use121 {
		mux.mux121.handleFunc(pattern, handler)
	} else {
		mux.register(pattern, HandlerFunc(handler))
		// 唯一的额外步骤：将普通函数转为 HandlerFunc。
	}
}
```

因此项目中的两种写法分别是：

```go
// 没有额外中间件：Controller 方法是普通函数，使用 HandleFunc 最直接。
mux.HandleFunc("GET /api/v1/posts", postController.List)

// 有 Auth：Auth 返回 http.Handler，使用 Handle。
mux.Handle("POST /api/v1/posts", protected(http.HandlerFunc(postController.Create)))
```

### 方法形式与包级形式

标准库还提供：

```go
func Handle(pattern string, handler Handler)
func HandleFunc(pattern string, handler func(ResponseWriter, *Request))
```

它们没有 `mux` 接收者，等价于向全局 `http.DefaultServeMux` 注册：

```go
func Handle(pattern string, handler Handler) {
	DefaultServeMux.register(pattern, handler)
}
```

本项目故意不用包级 `http.Handle` 和 `http.HandleFunc`，而是显式创建：

```go
mux := http.NewServeMux()
```

这样路由表属于当前 Server 实例，测试可创建独立 Mux，多个服务在同一进程内也不会意外共享全局路由表。与此同时，`http.Server{Handler: handler}` 明确指定了这个项目自己的 Handler 链，因此不会回退到 `DefaultServeMux`。

## 5. `ServeMux`：注册表与路由匹配器

### 5.1 `ServeMux` 内部保存什么

Go 1.26.5 中的结构如下：

```go
type ServeMux struct {
	mu     sync.RWMutex
	// 注册路由时写锁；请求匹配时读锁，保证并发安全。

	tree   routingNode
	// Go 1.22 新路由规则使用的匹配树，按 host、method、path segment 查找。

	index  routingIndex
	// 用于快速判断 pattern 关系与冲突的索引。

	mux121 serveMux121
	// 只在 GODEBUG=httpmuxgo121=1 时启用旧版 Go 1.21 路由兼容逻辑。
}

func NewServeMux() *ServeMux {
	return &ServeMux{}
}
```

`NewServeMux` 本身只是分配一个空路由器。路由树和索引会在第一次注册时逐步填充；创建 Mux 不会打开端口，也不会启动 goroutine。

### 5.2 `use121`：启动时选定新旧两套 ServeMux 规则

`mux121` 和 `use121` 不是每个 `ServeMux` 各自决定的开关，而是 `net/http` 包级变量。它们用于兼容 Go 1.22 之前的路由规则：Go 1.22 引入了 Method pattern、`{name}` 通配符和更严格的 pattern 冲突检查；旧项目在升级工具链时，可能需要暂时保留 Go 1.21 的匹配行为。

Go 1.26.5 的 [net/http/servemux121.go](https://cs.opensource.google/go/go/+/go1.26.5:src/net/http/servemux121.go) 中，决定逻辑如下：

```go
var httpmuxgo121 = godebug.New("httpmuxgo121")
// 创建名为 httpmuxgo121 的 GODEBUG 设置读取器。

var use121 bool
// 整个 net/http 包共享的最终开关；所有 ServeMux 读取同一个值。

func init() {
	if httpmuxgo121.Value() == "1" {
		use121 = true
		httpmuxgo121.IncNonDefault()
	}
}
```

包初始化发生在业务 `main` 执行前。`init` 只读取一次 `Value()` 并将结果写入普通 `bool`，之后 `Handle`、`HandleFunc`、`Handler`、`ServeHTTP` 都只读取 `use121`；即使程序运行中执行 `os.Setenv("GODEBUG", "httpmuxgo121=1")`，也不会切换已经选定的路由实现。这样能确保**注册阶段和请求阶段始终使用同一套规则**。

`httpmuxgo121.Value()` 的值来自运行时的 GODEBUG 配置。优先级可以这样理解：

```text
Go 工具根据 go.mod 的 go 版本生成二进制默认值
        + go.mod 的 godebug 指令或 //go:debug 指令
        ↓
运行时读取环境变量 GODEBUG（显式设置优先）
        ↓
net/http 初始化：Value() == "1" ?
        ├── 是：use121 = true，走 Go 1.21 兼容实现
        └── 否：use121 = false，走 Go 1.22+ 实现
```

最常见的显式兼容启动方式是：

```bash
GODEBUG=httpmuxgo121=1 go run ./cmd/api
```

不设置该变量时，Go 1.22 及更高版本声明的模块默认使用新实现。这个项目的 `go.mod` 声明 `go 1.26`，因此在常规启动（未显式设置 `GODEBUG=httpmuxgo121=1`）时结果为：

```text
httpmuxgo121.Value() == ""
use121 == false
```

因此本项目调用 `Handle`、`HandleFunc` 时走 `mux.register(...)`，收到请求时走 `mux.findHandler(...)`。项目使用的 `"GET /api/v1/posts/{id}"` 正是新规则的 Method + 路径变量 pattern；若强制启用 `httpmuxgo121=1`，它不会按这一语义解析，不能作为本项目的运行配置。

两条分支的状态完全隔离：`use121=false` 时数据写入 `tree`、`index`；`use121=true` 时数据写入 `mux121` 内部的旧版 map 和按长度排序的 slice。一个运行进程只能选择其中一套，不能让部分路由使用新规则、另一部分使用旧规则。

### 5.3 Go 1.22+ pattern 语法

本项目使用 Go 1.22 后引入的新 pattern 语法：

```text
GET /api/v1/posts/{id}
│   │               └── 单段路径变量，名称为 id
│   └────────────────── URL 路径
└────────────────────── HTTP Method 限制
```

请求 `GET /api/v1/posts/42` 匹配后，Controller 使用：

```go
idText := r.PathValue("id") // "42"
```

然后项目的 `parseID` 再执行 `strconv.ParseInt`，把字符串转换为 `int64`。路由器负责结构匹配和提取字符串，领域层仍应负责数值范围等业务校验。

ServeMux pattern 还支持主机匹配、精确路径、子树路径和多段通配符。这个项目只使用最适合 REST API 的“Method + 固定前缀 + 单段变量”形式。Go 1.22 起，非法 pattern 或两个无法确定优先级的冲突 pattern 会在注册时 panic；这是一种尽早暴露路由配置错误的策略。

### 5.4 注册过程

`mux.HandleFunc("GET /api/v1/posts/{id}", postController.Get)` 的注册过程可概括为：

```text
HandleFunc
  -> HandlerFunc(postController.Get)       将函数适配为 Handler
  -> mux.register(pattern, handler)        统一注册入口
  -> registerErr                            解析 pattern，找到可能冲突的已有路由
  -> index.possiblyConflictingPatterns      精确检查冲突
  -> tree.addPattern                        写入“请求匹配树”和 Handler
  -> index.addPattern                       写入“下一次注册的冲突检查索引”
```

标准库的统一入口是：

```go
func (mux *ServeMux) register(pattern string, handler Handler) {
	if err := mux.registerErr(pattern, handler); err != nil {
		panic(err)
	}
}
```

`registerErr` 会拒绝空 pattern、`nil` Handler、语法错误和冲突路由。例如两个同样能匹配请求、但没有明确“谁更具体”的 pattern 不能同时注册。路由表因此在启动阶段就固定下来；正常服务运行期间不应并发地动态修改路由。

#### 一条路由会变成什么数据

标准库不会把 `"GET /api/v1/posts/{id}"` 原样放进一个 `map[string]Handler`。它先把字符串解析为 `pattern`，拆成方法、主机和路径段：

~~~text
原始字符串：GET /api/v1/posts/{id}

pattern
├── method: "GET"
├── host:   ""                 // pattern 中没有限定 Host
└── segments:
    ├── "api"   （字面量）
    ├── "v1"    （字面量）
    ├── "posts" （字面量）
    └── "id"    （单段通配符，wild=true）
~~~

`tree` 的类型是 `routingNode`。一个节点既可以是中间节点，也可以是叶子节点：

~~~go
type routingNode struct {
	pattern *pattern
	handler Handler
	// 只有叶子节点保存完整 pattern 与注册时的 Handler。

	children   mapping[string, *routingNode]
	// 保存字面量分支，例如 "GET"、"posts"。

	emptyChild *routingNode
	// 保存单段通配符分支，例如 {id}。

	multiChild *routingNode
	// 保存多段通配符分支，例如 {path...} 或以 / 结尾的子树路由。
}
~~~

`tree.addPattern(pat, handler)` 按固定层次向下创建或复用节点：Host 在第一层、Method 在第二层、路径段从第三层开始。上例会形成下面的分支，最末端叶子才保存 `PostController.Get`：

~~~text
tree 根节点
└── host: "" 
    └── method: "GET"
        └── "api"
            └── "v1"
                └── "posts"
                    └── emptyChild       // {id}
                        └── leaf
                            ├── pattern: GET /api/v1/posts/{id}
                            └── handler: HandlerFunc(PostController.Get)
~~~

请求 `GET /api/v1/posts/42` 到来时，`tree.match` 依次走 `"" -> "GET" -> "api" -> "v1" -> "posts"`；最后一个 `42` 没有对应字面量子节点，于是走 `emptyChild`。匹配器在这一刻收集通配符值 `["42"]`，随后 `ServeMux.ServeHTTP` 才把它写入 Request，使 `r.PathValue("id")` 能返回 `"42"`。

字面量、单段通配符、多段通配符的尝试顺序是：

~~~text
字面量 child  ->  emptyChild（{id}）  ->  multiChild（{path...}）
~~~

这正好实现“更具体优先”。例如同时注册 `GET /api/v1/posts/latest` 和 `GET /api/v1/posts/{id}` 时，前者保存在 `"latest"` 子节点，后者保存在 `emptyChild`；请求 `/latest` 会先命中字面量分支。

#### `children` 为什么不是简单 `map`

`children` 的类型是标准库内部的泛型 `mapping[string, *routingNode]`。它会根据子节点数量改变底层表示：

~~~go
type mapping[K comparable, V any] struct {
	s []entry[K, V] // 子项较少时使用 slice
	m map[K]V       // 子项较多时使用 map
}

var maxSlice = 8
~~~

不超过 8 个键时，`add` 将节点追加到 slice，`find` 顺序查找；加入第 9 个键时，已有 slice 会转换为 Go `map`，后续通过哈希查找。路由树中大量节点只有很少的子分支，例如某个路径下可能只有 `"posts"` 和 `"users"` 两个子段。对这种小集合，slice 避免了单独分配 map 的成本；分支很多时再切换到 map，以降低查找成本。这是标准库为小型路由表与大型路由表同时做的自适应优化。

#### `index` 保存什么，为什么还需要它

`tree` 解决“请求来了如何找到 Handler”；`index` 不参与正常请求匹配，它只解决“注册新 pattern 时，哪些旧 pattern 值得拿来做冲突检查”。其核心结构是：

~~~go
type routingIndex struct {
	segments map[routingIndexKey][]*pattern
	multis   []*pattern
}

type routingIndexKey struct {
	pos int    // 路径段下标，从 0 开始
	s   string // 字面量；空字符串表示单段通配符
}
~~~

上面的 `GET /api/v1/posts/{id}` 会产生四个索引键：

~~~text
{pos: 0, s: "api"}   -> [该 pattern]
{pos: 1, s: "v1"}    -> [该 pattern]
{pos: 2, s: "posts"} -> [该 pattern]
{pos: 3, s: ""}      -> [该 pattern]   // {id}
~~~

以后注册 `GET /api/v1/posts/latest` 时，`index` 会优先找第 0、1、2 段的相同字面量和通配符候选，而不是遍历所有已注册路由；随后才调用 `conflictsWith` 作精确判断。`/latest` 比 `/{id}` 更具体，所以两者允许共存。以多段通配符结尾的 pattern，例如 `/files/{path...}`，会保存在 `multis` 中，因为它可能覆盖的范围更广，索引无法安全地用单一位置快速排除。

因此，两个结构保存的内容不同：

| 结构 | 保存内容 | 使用时机 | 是否保存 Handler |
| --- | --- | --- | --- |
| `tree` | Host、Method、路径段形成的决策树 | 每个请求到来时 | 是，叶子节点保存 |
| `index` | 路径段位置到 `pattern` 候选集合的索引 | 每次注册新路由时 | 否，只保存 `pattern` 指针 |

### 5.5 `Handler` 与 `ServeHTTP` 为什么是两个方法

`ServeMux.Handler(r)` 的作用是“只查找”：返回当前请求匹配的 Handler 与 pattern。

```go
func (mux *ServeMux) Handler(r *Request) (h Handler, pattern string) {
	if use121 {
		return mux.mux121.findHandler(r)
	}
	h, p, _, _ := mux.findHandler(r)
	return h, p
}
```

它不修改传入的 `Request`，所以也不会填充路径变量。适合诊断、包装或需要自行决定是否调用 Handler 的代码。

`ServeMux.ServeHTTP` 才是 `ServeMux` 实现 `http.Handler` 的方法，也是 HTTP Server 实际调用的方法：

```go
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
	if r.RequestURI == "*" {
		if r.ProtoAtLeast(1, 1) {
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(StatusBadRequest)
		return
	}

	var h Handler
	if use121 {
		h, _ = mux.mux121.findHandler(r)
	} else {
		h, r.Pattern, r.pat, r.matches = mux.findHandler(r)
		// 除了找到 h，还把匹配 pattern 和变量匹配结果保存到 Request。
	}
	h.ServeHTTP(w, r) // 跳转到具体路由对应的 Handler。
}
```

这解释了为什么 `r.PathValue("id")` 能在 `PostController.Get` 中读取到值：请求先经过 `ServeMux.ServeHTTP`，它在调用具体 Handler 之前写入了匹配信息。

## 6. 中间件：不是特殊机制，而是 Handler 组合

标准库没有名为 Middleware 的内建类型；项目用一个函数类型把这个组合关系显式表达出来：

```go
type Middleware func(http.Handler) http.Handler
```

它的含义是：“接收下游 Handler，返回一个包住下游的新 Handler”。

### 6.1 全局链的构造

项目实现：

```go
func Chain(next http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		next = middlewares[i](next)
	}
	return next
}
```

调用：

```go
handler := middleware.Chain(mux, middleware.Logger, middleware.Recovery)
```

循环从右向左，因此构造过程是：

```text
初始 next = mux
i = Recovery -> next = Recovery(mux)
i = Logger   -> next = Logger(Recovery(mux))
```

请求从最外层进入，执行顺序是：

```text
Logger before
  -> Recovery before
      -> mux.ServeHTTP
          -> 路由 Handler
      <- Recovery after
<- Logger after（记录耗时）
```

`Logger` 与 `Recovery` 的项目源码说明了这种“调用 next 前后都能工作”的结构：

```go
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r) // 调用内层 Handler
		log.Printf("method=%s path=%s duration=%s", r.Method, r.URL.Path, time.Since(started))
	})
}

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				httpapi.WriteResponse(w, http.StatusInternalServerError, "internal server error", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
```

### 6.2 路由级认证链

认证中间件也是同一模式：

```go
func Auth(authService *service.AuthService) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				httpapi.WriteResponse(w, http.StatusUnauthorized, "missing bearer token", nil)
				return // 不调用 next，Controller 不会执行
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := authService.ParseToken(r.Context(), token)
			if err != nil {
				httpapi.WriteResponse(w, http.StatusUnauthorized, err.Error(), nil)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

`Request` 不可变地携带 Context：`r.WithContext(ctx)` 返回新的 Request 值，而不是修改原请求。Controller 后续从 `r.Context()` 获取 `user_id`：

```go
userID := middleware.CurrentUserID(r.Context())
```

项目使用私有的 `contextKey` 类型而不是裸字符串，降低与其他包的 Context Key 冲突风险。

## 7. `http.Server`：谁真正监听端口

`ServeMux` 只负责“请求进来后选哪个 Handler”，它不监听网络。监听 TCP、维护连接、读取 HTTP 报文的是 `http.Server`。

### 7.1 `Server`：配置、Handler 入口与运行时状态

`Server` 是 HTTP 服务端的总配置和运行时管理者；零值也可用。它不保存具体路由，也不直接拥有 Controller。它通过 `Handler` 字段持有每个 HTTP 请求最终应交给谁的入口。以下是与项目和执行链直接相关的字段：

```go
type Server struct {
	Addr string
	// 监听的 host:port；为空时使用 ":http"，即端口 80。

	Handler Handler
	// 每个请求最终要调用的 Handler；为 nil 时回退到 http.DefaultServeMux。

	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	// 分别控制完整读取、请求头读取、响应写入、Keep-Alive 空闲等待时间。

	// ... TLS、HTTP/2、连接状态、优雅关闭等字段
}
```

完整 `Server` 还包含下列字段类别；理解它们的归属关系，比逐个背诵字段更重要：

| 字段类别 | 代表字段 | 负责什么 |
| --- | --- | --- |
| 监听与分发 | `Addr`、`Handler`、`DisableGeneralOptionsHandler` | 绑定地址，并选择请求入口 |
| 协议 | `TLSConfig`、`Protocols`、`TLSNextProto` | TLS、ALPN、HTTP/1 与 HTTP/2 |
| 读写边界 | `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout`、`MaxHeaderBytes` | 限制网络连接和 HTTP 报文读写 |
| 可观测性与 Context | `ErrorLog`、`ConnState`、`BaseContext`、`ConnContext` | 错误日志、连接状态和连接级上下文 |
| 内部生命周期 | `inShutdown`、`listeners`、`activeConn` | 供 `Serve`、`Close`、`Shutdown` 协调资源 |

`Handler` 是最关键的衔接字段：它的静态类型是接口 `http.Handler`，因此可以放入 `ServeMux`、`HandlerFunc`、自定义结构体 Handler，或多个中间件包装后的结果。相反，`Server` 并不关心路由树如何存储、Controller 如何创建或 SQL 如何执行。

~~~text
Addr                         决定 ListenAndServe 绑定到哪里
Handler                      决定 serverHandler 最终调用哪个请求处理器
超时和 MaxHeaderBytes        约束网络和 HTTP 协议读写
TLS/Protocols                决定协议协商
listeners / activeConn       让 Close、Shutdown 管理已监听和活跃连接
~~~

本项目设置的超时不是业务逻辑，而是网络边界策略：

```text
ReadHeaderTimeout = 5s   防止客户端极慢地发送请求头
ReadTimeout       = 15s  限制整个请求读取时间
WriteTimeout      = 15s  限制响应写入时间
IdleTimeout       = 60s  限制 Keep-Alive 连接等待下一请求的时间
```

`Server.Run` 调用的是：

```go
func (s *Server) Run() error {
	return s.httpServer.ListenAndServe()
}
```

### 7.2 `Server` 的关键方法如何协作

`Server` 的方法不是相互独立的工具函数，而是围绕同一组 Listener 和连接状态协作：

| 方法 | 输入 | 主要职责 | 关系 |
| --- | --- | --- | --- |
| `ListenAndServe` | 使用 `Server.Addr` | 创建 TCP Listener 并启动服务 | 内部调用 `net.Listen` 和 `Serve` |
| `Serve` | 外部传入 `net.Listener` | 接受连接、创建内部 `conn` 并启动连接 goroutine | 每条连接进入 `c.serve` |
| `ServeTLS` / `ListenAndServeTLS` | TLS 证书或 TLS Listener | 进行 TLS 和 ALPN 协商 | 最终仍把请求交给同一个 Handler 分发链 |
| `Shutdown` | `context.Context` | 停止接收新连接，关闭空闲连接，等待活跃连接结束 | 使用 `listeners` 和 `activeConn` |
| `Close` | 无 | 立即关闭 Listener 和活跃连接 | 不等待正在执行的 Handler |

它们的主调用关系是：

~~~text
ListenAndServe
  -> net.Listen("tcp", Addr)
  -> Server.Serve(listener)
       -> listener.Accept()
       -> go conn.serve(...)
            -> serverHandler{server}.ServeHTTP(...)
                 -> server.Handler.ServeHTTP(...)
~~~

`Serve` 允许调用方传入自行创建的 Listener，例如 Unix Socket、已经完成 TLS 包装的 Listener 或测试 Listener；`ListenAndServe` 则是 TCP 场景的便捷包装。`Shutdown` 一旦执行，Server 就进入关闭状态，后续 `Serve` 或 `ListenAndServe` 会返回 `ErrServerClosed`，不能把同一实例重新启动。

### 7.3 `ListenAndServe` 做了什么

Go 1.26.5 的关键源码如下：

```go
func (s *Server) ListenAndServe() error {
	if s.shuttingDown() {
		return ErrServerClosed
	}

	addr := s.Addr
	if addr == "" {
		addr = ":http"
	}

	ln, err := net.Listen("tcp", addr)
	// 在地址上创建 TCP Listener；端口被占用时在此返回错误。
	if err != nil {
		return err
	}

	return s.Serve(ln)
}
```

所以 `ListenAndServe` 是 `Server` 的一个方法，不是一个新的对象。它内部调用 `net.Listen("tcp", addr)` 创建监听器，再把监听器交给 `Server.Serve`。包级函数 `http.ListenAndServe(addr, handler)` 则只是一个便捷封装：

```go
func ListenAndServe(addr string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}
```

本项目显式创建 `http.Server`，而非调用包级函数，是因为需要配置超时，且后续可自然扩展为 `Shutdown` 优雅关闭。

### 7.4 `Serve` 如何接收连接

`Server.Serve` 的关键循环如下：

```go
func (s *Server) Serve(l net.Listener) error {
	// ... 初始化 HTTP/2、连接状态与基础 Context
	for {
		rw, err := l.Accept() // 阻塞等待一个新的 TCP 连接
		if err != nil {
			if s.shuttingDown() {
				return ErrServerClosed
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				// ... 临时错误退避重试
				continue
			}
			return err // 非临时错误结束 Serve
		}

		c := s.newConn(rw)
		go c.serve(connCtx) // 每个 TCP 连接交给独立 goroutine
	}
}
```

这里的并发单位首先是**连接**。对 HTTP/1.1 Keep-Alive 连接，连接 goroutine 可在同一条 TCP 连接上顺序读取多个请求；HTTP/2 又允许一条连接上有多个并发 stream。业务代码不应假设“一个连接只会有一个请求”，也不应把请求状态存到全局变量。

### 7.5 `serverHandler` 如何选择 Handler

`serverHandler` 是 `net/http` 的内部适配器。它只有一个字段，保存正在处理连接的 `*Server` 指针：

~~~go
type serverHandler struct {
	srv *Server
	// 不复制 Server；直接读取同一个 Server 的 Handler 和配置。
}
~~~

它在 `conn.serve` 已经解析出 `Request`、创建好 `ResponseWriter` 后被构造并调用。HTTP/1.1 请求处理循环中的关键调用是：

~~~go
serverHandler{c.server}.ServeHTTP(w, w.req)
// c.server 是当前连接归属的 *Server。
// w 是这一次请求的 ResponseWriter，w.req 是解析出的 *Request。
~~~

因此 `serverHandler` 并不是新的路由器，也不会保存路由表；它只把连接层的请求桥接到 `Server.Handler` 或默认路由器。它的 `ServeHTTP` 实现如下：

~~~go
func (sh serverHandler) ServeHTTP(rw ResponseWriter, req *Request) {
	handler := sh.srv.Handler
	if handler == nil {
		handler = DefaultServeMux
		// 没有显式配置 Server.Handler 时，才使用全局路由器。
	}
	if !sh.srv.DisableGeneralOptionsHandler && req.RequestURI == "*" && req.Method == "OPTIONS" {
		handler = globalOptionsHandler{}
		// 这是 HTTP 的 OPTIONS * 特殊请求，不是普通 URL 路由。
	}
	handler.ServeHTTP(rw, req)
	// serverHandler 到此结束；后续行为完全由被选中的 Handler 决定。
}
~~~

本项目的 `httpServer.Handler` 是 `Logger(Recovery(mux))`，不是 `nil`，所以 `serverHandler` 不会使用全局 `DefaultServeMux`。最终会调用外层 Logger 的 `ServeHTTP`，随后层层向内传递：

~~~text
serverHandler
  -> Server.Handler = Logger(Recovery(mux))
       -> Logger.ServeHTTP
            -> Recovery.ServeHTTP
                 -> ServeMux.ServeHTTP
                      -> Auth(PostController.Create)
~~~

## 8. 一次 `POST /api/v1/posts` 的完整执行路径

将前面的对象关系连起来，请求路径如下：

```text
1. 客户端建立或复用 TCP 连接，发送 HTTP 请求。
2. net.Listener.Accept 接受连接；Server 为连接运行 c.serve。
3. 标准库解析请求行、Header、Body，构造 *http.Request 与 ResponseWriter。
4. serverHandler 取得 httpServer.Handler。
5. Logger.ServeHTTP 记录开始时间，调用 Recovery。
6. Recovery.ServeHTTP 设置 defer recover，调用 ServeMux。
7. ServeMux.ServeHTTP 按 Method=POST、Path=/api/v1/posts 匹配路由。
8. 匹配到注册时保存的 Handler：Auth(HandlerFunc(PostController.Create))。
9. Auth 检查 Bearer Token，调用 AuthService.ParseToken，并将 user_id 放入新 Request Context。
10. PostController.Create 解码 JSON，请求 Service 创建文章。
11. PostService 调用 PostRepository；Repository 对 MySQL 执行 INSERT 和查询。
12. Controller 使用 httpapi.WriteResponse 写入 201 JSON。
13. 调用栈逐层返回；Recovery 没有捕获到 panic；Logger 记录耗时。
```

第 7 步的核心就是 `ServeMux.ServeHTTP`：它找到 Handler 后调用：

```go
h.ServeHTTP(w, r)
```

从标准库角度看，路由、认证中间件、Controller 都只是不同层级的 `http.Handler`。这也是 `net/http` 的设计核心：用一个极小接口把路由、框架、中间件和业务处理器统一起来。

## 9. 响应写入与请求体读取的边界

Controller 不直接使用 `json.NewEncoder`，而通过 [internal/httpapi/json.go](internal/httpapi/json.go) 的帮助函数：

```go
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
```

HTTP 响应遵循顺序：先设置 Header，再写状态码，最后写 Body。`WriteHeader` 之后再改普通 Header 不会影响已经发出的响应；第一次 `Write` 若尚未显式调用 `WriteHeader`，标准库会隐式发送 `200 OK`。因此项目在所有分支都通过同一个函数明确写入状态码。

请求 JSON 解码也集中在 `DecodeJSON`：

```go
decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
decoder.DisallowUnknownFields()
if err := decoder.Decode(dst); err != nil { ... }
if err := decoder.Decode(&struct{}{}); err != io.EOF { ... }
```

它依次限制请求体最大 1 MiB、拒绝未知字段、拒绝多个连续 JSON 值。HTTP Controller 负责把解码错误映射为 400；Service 与 Repository 不需要知道 JSON 格式或 `ResponseWriter` 的存在。

## 10. Controller、Service、Repository 的边界

去掉 Gin 不会改变业务分层：

```text
Controller
  HTTP 输入输出：读取 PathValue、Header、Body，决定 HTTP 状态码与 JSON
        ↓
Service
  用例编排：密码哈希、JWT 签发和校验、文章操作的业务参数
        ↓
Repository
  存储细节：SQL、Redis Key、数据库驱动调用
        ↓
MySQL / Redis
```

例如 `Auth` 中间件需要认证业务能力，因此它调用 `AuthService.ParseToken`；但它不写 Redis 命令。`PostController.Create` 读取 `user_id` 和 JSON；但它不写 SQL。这样的依赖方向使 HTTP 框架可以替换，而 Service 和 Repository 基本不受影响。

## 11. 启动与验证

本版本与其他学习目录使用不同的本地依赖端口，可同时启动：

```text
MySQL: 127.0.0.1:3307
Redis: 127.0.0.1:6380
HTTP:  :8081
```

```bash
make docker-up
make tidy
make test
make run
```

健康检查：

```bash
curl http://127.0.0.1:8081/health
```

完整接口调用见 [api/http.md](api/http.md)，数据库表结构见 [api/schema.sql](api/schema.sql)。停止本地依赖：

```bash
make docker-down
```
