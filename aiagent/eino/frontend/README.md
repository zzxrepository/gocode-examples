# Eino AI Chat UI

这是与同级 Go `backend` 配套的 Vue 3 聊天前端。它使用 **Vite、Vue Router、Pinia、Axios**，界面采用接近 ChatGPT 的侧栏、会话、流式回复和底部输入框布局。

## 启动

先启动后端（默认端口 `8080`）：

```bash
cd ../backend
go run ./cmd/server
```

再启动前端：

```bash
cd ../frontend
npm install
npm run dev
```

浏览器访问 <http://localhost:5173>。开发环境中 Vite 会把 `/api/*` 代理给 `http://127.0.0.1:8080`，因此浏览器没有跨域问题。

## 当前已接通的后端能力

| 前端功能 | 后端接口 | 状态 |
| --- | --- | --- |
| 健康状态 | `GET /api/health` | 已实现 |
| 模型选择 | `GET /api/models` | 已实现 |
| 多轮上下文、流式回答、停止生成、重新生成 | `POST /api/chat/stream` | 已实现 |

后端目前是无状态的，因此会话标题、消息历史、温度、已选择工具暂时保存在浏览器 `localStorage`。这不会伪装成数据库持久化；清理浏览器站点数据后会消失。

## 已完成的前端能力与预留能力

- 本地新建、搜索、重命名、删除会话；导出 Markdown。
- 流式聊天、模型选择、Temperature、停止生成与重新生成。
- 登录、注册、退出登录及路由保护；当前为明确标识的本地演示会话，不会保存密码。
- 附件选择、RAG 知识库、MCP 工具、Skill 的界面和会话状态。
- 附件上传/解析、RAG 检索、MCP 执行、真实认证和会话持久化仍等待后端接口。组件不会自行拼接 URL，所有待实现 API 集中在 `src/api/chat.ts` 与 `src/api/auth.ts`。

学习项目的范围与实现顺序见 [docs/PRODUCT_AUDIT_AND_ROADMAP.md](docs/PRODUCT_AUDIT_AND_ROADMAP.md)，完整请求、响应和 SSE 契约见 [docs/API_CONTRACT.md](docs/API_CONTRACT.md)。

## 前端目录

```text
src/
  api/          # Axios、SSE 与待实现 API 的唯一入口
  components/   # 侧栏、消息、输入框
  router/       # Vue Router 路由表
  stores/       # Pinia 会话与 UI 状态
  types/        # 前后端共享的数据契约
  views/        # ChatView、SettingsView 页面
```

生产环境请复制 `.env.example` 为 `.env.local`，配置 `VITE_API_BASE_URL`；不要把令牌写进前端环境变量。
