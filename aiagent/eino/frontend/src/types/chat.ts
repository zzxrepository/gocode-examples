export type MessageRole = 'system' | 'user' | 'assistant'
export type MessageStatus = 'complete' | 'streaming' | 'error'

export interface AttachmentMeta {
  id: string
  name: string
  size: number
  type: string
  localOnly: boolean
}

// RAG 和 MCP 的服务端执行过程会通过 SSE 回传，前端只负责清晰呈现。
export interface ToolRun {
  id: string
  name: string
  status: 'queued' | 'running' | 'complete' | 'failed'
  inputSummary?: string
  outputSummary?: string
}

export interface Citation {
  id: string
  title: string
  url: string
  source?: string
}

export interface ChatMessage {
  id: string
  role: MessageRole
  content: string
  createdAt: string
  status: MessageStatus
  attachments?: AttachmentMeta[]
  toolRuns?: ToolRun[]
  citations?: Citation[]
}

export interface Conversation {
  id: string
  title: string
  model: string
  skillID: string
  createdAt: string
  updatedAt: string
  messages: ChatMessage[]
  enabledTools: string[]
}

export interface ModelInfo {
  id: string
  provider: string
  model: string
  label: string
}

export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
  trace_id: string
}

export interface ChatRequest {
  model?: string
  messages: Pick<ChatMessage, 'role' | 'content'>[]
  temperature?: number
}

export const toolCatalog = [
  { id: 'rag', label: 'RAG 知识库', detail: '基于上传文件和知识库检索后回答', icon: '◇' },
  { id: 'mcp', label: 'MCP 工具', detail: '调用服务端已授权的外部能力', icon: '⌘' },
] as const

export const skillCatalog = [
  { id: 'general', label: '通用助手', detail: '普通问答、写作和代码解释', tools: [] },
  { id: 'rag_qa', label: '文档问答', detail: '优先检索已上传的资料并标注来源', tools: ['rag'] },
  { id: 'blog_operator', label: '博客助手', detail: '通过 MCP 查询、创建和管理博客内容', tools: ['mcp'] },
] as const
