# Aurora AI 前后端接口契约

所有 JSON 响应统一为：

```json
{"code":0,"message":"ok","data":{},"trace_id":"..."}
```

错误也保持相同外层结构。`trace_id` 必须写入响应头 `X-Trace-ID` 和服务端日志。所有需要登录的接口使用 `Authorization: Bearer <access_token>`。

## 1. 当前已实现接口

| 方法与路径 | 用途 |
| --- | --- |
| `GET /api/health` | 健康检查 |
| `GET /api/models` | 返回默认模型和当前可选模型 |
| `POST /api/chat/stream` | 无状态 SSE 聊天 |

当前聊天请求：

```json
{
  "model": "dashscope/qwen3.8-max",
  "temperature": 0.7,
  "messages": [{"role":"user","content":"你好"}]
}
```

SSE：

```text
event: delta
data: {"code":0,"message":"ok","data":{"delta":"你好"},"trace_id":"..."}

event: done
data: {"code":0,"message":"ok","data":{"done":true},"trace_id":"..."}
```

## 2. 认证（P0）

| 方法与路径 | 请求 | 返回 |
| --- | --- | --- |
| `POST /api/auth/register` | `username,email,password,display_name?` | `user,access_token,expires_at` |
| `POST /api/auth/login` | `account,password` | `user,access_token,expires_at` |
| `POST /api/auth/refresh` | refresh token 或安全 Cookie | 新 access token |
| `POST /api/auth/logout` | 无 | `{logged_out:true}` |
| `GET /api/me` | 无 | 当前用户 |

密码只保存强哈希；登录失败不要区分“用户不存在”与“密码错误”。前端不能保存模型供应商令牌、MCP 密钥或其他系统 JWT。

## 3. 会话与消息（P0）

| 方法与路径 | 请求 | 返回 |
| --- | --- | --- |
| `GET /api/conversations?cursor=&limit=30` | 无 | `items,next_cursor` |
| `POST /api/conversations` | `model?,skill_id?` | 对话对象 |
| `GET /api/conversations/{id}` | 无 | 对话与第一页消息 |
| `PATCH /api/conversations/{id}` | `title?,model?,skill_id?,enabled_tools?` | 对话对象 |
| `DELETE /api/conversations/{id}` | 无 | `{id,deleted:true}` |
| `GET /api/conversations/{id}/messages?cursor=&limit=50` | 无 | `items,next_cursor` |
| `POST /api/conversations/{id}/messages/stream` | 见下方 | SSE |

有状态流请求：

```json
{
  "content": "总结我上传的资料",
  "model": "dashscope/qwen3.8-max",
  "skill_id": "rag_qa",
  "enabled_tools": ["rag"],
  "attachment_ids": ["file_uuid"],
  "temperature": 0.7
}
```

服务端先保存用户消息，再产生助手消息。前端停止请求时服务端必须取消模型/工具 Context，并把助手消息标记为 `cancelled` 或 `partial`。

## 4. 文件与 RAG（P1）

| 方法与路径 | 请求 | 返回 |
| --- | --- | --- |
| `POST /api/files` | `multipart/form-data`，字段 `file` | `id,name,size,mime_type,status` |
| `GET /api/files?cursor=&limit=` | 无 | 文件列表 |
| `GET /api/files/{id}` | 无 | 元数据或短期下载 URL |
| `DELETE /api/files/{id}` | 无 | `{id,deleted:true}` |
| `POST /api/knowledge-bases` | `name,description?` | 知识库对象 |
| `POST /api/knowledge-bases/{id}/files` | `file_ids` | 入库任务 |

文件状态：`uploading`、`processing`、`ready`、`failed`。只有 `ready` 文件可参与检索。服务端需做所有权校验、MIME/大小限制、病毒扫描、文本提取/OCR、切块、Embedding 和向量检索。

RAG 的结果必须带出处，而非只返回模型文本：

```text
event: tool_result
data: {"code":0,"message":"ok","data":{"id":"rag_1","name":"rag","status":"complete","output_summary":"检索到 4 个片段"},"trace_id":"..."}

event: citation
data: {"code":0,"message":"ok","data":{"id":"source_1","title":"Go 并发笔记.pdf","url":"/api/files/file_uuid","source":"第 12 页"},"trace_id":"..."}
```

## 5. MCP（P2）

| 方法与路径 | 请求 | 返回 |
| --- | --- | --- |
| `GET /api/mcp/servers` | 无 | 当前用户可用的 MCP Server |
| `POST /api/mcp/servers/{id}/authorize` | OAuth/授权码流程 | 授权状态 |
| `GET /api/tools` | 无 | 当前用户允许使用的工具及 JSON Schema |
| `GET /api/tool-calls/{id}` | 无 | 工具调用状态与安全摘要 |
| `POST /api/tool-calls/{id}/confirm` | `{approved:true}` | 确认危险操作 |

Eino 后端是 MCP Client，浏览器不是 MCP Client。浏览器永远不直接携带下游系统令牌；服务端按当前 AuroraAI 用户取得授权并转发。删除、发布、写库等操作必须先输出确认事件：

```text
event: tool_confirmation_required
data: {"code":0,"message":"ok","data":{"tool_call_id":"...","title":"删除文章 #9","risk":"high"},"trace_id":"..."}
```

## 6. Skill（P2）

Skill 是受服务端管理的 Agent 行为配置，不能让前端随意传 system prompt。

| 方法与路径 | 请求 | 返回 |
| --- | --- | --- |
| `GET /api/skills` | 无 | 当前用户可选 Skill 列表 |
| `GET /api/skills/{id}` | 无 | skill 的名称、描述、允许工具、版本 |
| `POST /api/skills` | 管理员：`name,instructions,allowed_tools` | skill 对象 |
| `PATCH /api/skills/{id}` | 管理员更新 | 新版本 skill |

前端只在创建/更新会话时传 `skill_id`。后端负责拼入经过审核的指令、限制可用工具、记录 skill 版本，并审计每次调用。

## 7. SSE 事件

除 `delta`、`done`、`error` 外，有状态接口逐步支持：

```text
message_created              # 返回 user_message_id、assistant_message_id
tool_call                    # {id,name,status,input_summary}
tool_result                  # {id,name,status,output_summary}
citation                     # {id,title,url,source}
tool_confirmation_required   # 高风险写操作的用户确认卡片
usage                        # {input_tokens,output_tokens,latency_ms}
```

不要返回隐藏推理链；界面只展示可审计的工具过程、结果摘要和来源。
