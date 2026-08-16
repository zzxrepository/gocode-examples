# AI Agent Chat

一个 Vue 3 前端与 Go 后端组成的聊天应用示例。后端代理百炼的 Anthropic 兼容接口（也支持 OpenAI Chat Completions），令牌仅保存在后端本地配置或环境变量中，浏览器不会接触模型令牌。

当前目录默认使用 `backend-v2`：它基于 Eino `ChatModelAgent + Runner`，监听 `8081` 端口并与 Vue API 兼容。`backend` 是保留的协议直连 v1 实现。

## 架构与模型切换

后端采用 **策略模式 + 工厂/注册表 + 适配器**：

```text
HTTP Handler → ChatService → Provider Registry → ChatProvider 策略实现 → 厂商 API
                                      ├── AnthropicProvider（百炼）
                                      └── OpenAIProvider（OpenAI 兼容服务）
```

- `ChatProvider` 是厂商模型的统一接口；每个厂商各自处理鉴权、请求格式和流式协议。
- `Registry` 根据配置中 `provider_id/model_id` 形式的模型引用解析对应策略，例如 `dashscope/qwen3.8-max`。
- 新增厂商时，实现一个 `ChatProvider` 并在 `internal/provider/provider.go` 注册工厂即可；HTTP 和前端无需修改。
- `ChatService` 负责多轮消息校验、模型选择与错误归一化；Go 的惯用写法是小接口与组合，而非 Java 风格的继承树。

```text
backend/
├── cmd/server/               # 程序入口与依赖组装
├── config/                   # 默认配置；local.yaml 存放本地密钥并被忽略
├── internal/config/          # Viper 配置加载与校验
├── internal/domain/          # 聊天、消息、模型等领域数据
├── internal/provider/        # Provider 策略、工厂注册表、厂商协议适配器
├── internal/service/         # 多轮对话与模型选择等业务编排
└── internal/httpapi/         # 路由、handler、CORS、trace 与统一响应封装
```

普通 JSON 接口统一返回：

```json
{ "code": 0, "message": "ok", "data": {}, "trace_id": "..." }
```

流式接口同样使用这个信封：`delta`、`done`、`error` 三类 SSE 事件都包含 `code`、`message`、`data` 与 `trace_id`。

## 运行后端

```bash
cd backend
make tidy
make run
```

项目的 `Makefile` 默认使用本机的 Go 1.26.5（`/Users/mmzhang/DevTools/SDK/go/1.26.5/bin/go`）；如需替换，执行时传入 `GO=/你的/go`。

默认读取 `backend/config/config.yaml`。Viper 同时支持以下环境变量覆盖：

- `AIAGENT_SERVER_ADDRESS`
- `AIAGENT_MODELS_DEFAULT`
- `AIAGENT_PROVIDER_<PROVIDER_ID>_API_TOKEN`，例如 `AIAGENT_PROVIDER_DASHSCOPE_API_TOKEN`

后端默认监听 `http://localhost:8080`，提供 `GET /api/health`、`GET /api/models` 和流式 `POST /api/chat/stream`。一个对话的历史消息会随每次请求发送，因此支持多轮对话；模型只能从配置中的 `models.providers[].models` 列表切换。

## 运行前端

```bash
cd frontend
npm install
npm run dev
```

打开 Vite 显示的地址（默认 `http://localhost:5173`）。开发服务器会默认把 `/api` 代理到 `http://localhost:8081`。若前后端分开部署，在前端的 `.env` 中设置 `VITE_API_BASE_URL`，并将该站点域名加入后端 `server.cors_origins`。
