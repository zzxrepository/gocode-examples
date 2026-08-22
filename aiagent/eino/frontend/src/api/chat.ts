import { apiBaseURL, BackendCapabilityError, http } from '@/api/http'
import type { ApiEnvelope, ChatRequest, ModelInfo } from '@/types/chat'

interface ModelsPayload {
  default_model: string
  models: ModelInfo[]
}

export async function getModels(): Promise<ModelsPayload> {
  const response = await http.get<ApiEnvelope<ModelsPayload>>('/api/models')
  return response.data.data
}

export async function streamStatelessChat(
  request: ChatRequest,
  onDelta: (delta: string) => void,
  signal?: AbortSignal,
): Promise<void> {
  const response = await fetch(`${apiBaseURL}/api/chat/stream`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'text/event-stream' },
    body: JSON.stringify(request),
    signal,
  })
  if (!response.ok || !response.body) throw new Error(`聊天请求失败（HTTP ${response.status}）`)

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const events = buffer.split(/\r?\n\r?\n/)
    buffer = events.pop() ?? ''
    for (const item of events) {
      const event = item.match(/^event:\s*(.+)$/m)?.[1] ?? 'message'
      const encoded = item.match(/^data:\s*(.+)$/m)?.[1]
      if (!encoded) continue
      const payload = JSON.parse(encoded) as ApiEnvelope<{ delta?: string }>
      if (event === 'delta') onDelta(payload.data?.delta ?? '')
      if (event === 'error') throw new Error(payload.message || '模型回复失败')
    }
  }
}

// 以下契约是前端唯一的待实现接口入口；不要在组件内手写 URL。
export const reservedAPI = {
  listConversations: () => Promise.reject(new BackendCapabilityError('会话列表')),
  createConversation: () => Promise.reject(new BackendCapabilityError('创建会话')),
  updateConversation: () => Promise.reject(new BackendCapabilityError('更新会话')),
  deleteConversation: () => Promise.reject(new BackendCapabilityError('删除会话')),
  streamConversationMessage: () => Promise.reject(new BackendCapabilityError('有状态流式对话')),
  uploadAttachment: () => Promise.reject(new BackendCapabilityError('文件上传')),
  listFiles: () => Promise.reject(new BackendCapabilityError('文件库')),
  listKnowledgeBases: () => Promise.reject(new BackendCapabilityError('RAG 知识库')),
  listMCPServers: () => Promise.reject(new BackendCapabilityError('MCP Server 列表')),
  listTools: () => Promise.reject(new BackendCapabilityError('MCP 工具配置')),
  confirmToolCall: () => Promise.reject(new BackendCapabilityError('危险工具确认')),
  listSkills: () => Promise.reject(new BackendCapabilityError('Skill 列表')),
  updateSettings: () => Promise.reject(new BackendCapabilityError('用户设置')),
}
