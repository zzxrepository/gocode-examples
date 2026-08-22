import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { getModels, streamStatelessChat } from '@/api/chat'
import type { AttachmentMeta, ChatMessage, Conversation, ModelInfo } from '@/types/chat'

const STORAGE_KEY = 'eino-ai-chat.conversations.v1'
const SETTINGS_KEY = 'eino-ai-chat.temperature.v1'
const createID = () => crypto.randomUUID()
const now = () => new Date().toISOString()

function loadConversations(): Conversation[] {
  try { return JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '[]') as Conversation[] } catch { return [] }
}

export const useChatStore = defineStore('chat', () => {
  const conversations = ref<Conversation[]>(loadConversations())
  const models = ref<ModelInfo[]>([])
  const defaultModel = ref('')
  const temperature = ref(Number(localStorage.getItem(SETTINGS_KEY) ?? '0.7'))
  const generatingConversationID = ref<string | null>(null)
  let abortController: AbortController | undefined

  const isGenerating = computed(() => generatingConversationID.value !== null)
  const getConversation = (id: string) => conversations.value.find((item) => item.id === id)
  const persist = () => localStorage.setItem(STORAGE_KEY, JSON.stringify(conversations.value))

  async function loadModels() {
    try {
      const payload = await getModels()
      models.value = payload.models
      defaultModel.value = payload.default_model
    } catch {
      // 服务未启动时仍允许浏览本地记录；发送时会显示请求错误。
    }
  }

  function createConversation(): Conversation {
    const conversation: Conversation = { id: createID(), title: '新对话', model: defaultModel.value, skillID: 'general', createdAt: now(), updatedAt: now(), messages: [], enabledTools: [] }
    conversations.value.unshift(conversation)
    persist()
    return conversation
  }

  function renameConversation(id: string, title: string) {
    const conversation = getConversation(id)
    if (!conversation || !title.trim()) return
    conversation.title = title.trim(); conversation.updatedAt = now(); persist()
  }

  function removeConversation(id: string) {
    conversations.value = conversations.value.filter((item) => item.id !== id)
    persist()
  }

  function updateConversationModel(id: string, model: string) {
    const conversation = getConversation(id)
    if (!conversation) return
    conversation.model = model; conversation.updatedAt = now(); persist()
  }

  function updateTools(id: string, enabledTools: string[]) {
    const conversation = getConversation(id)
    if (!conversation) return
    conversation.enabledTools = enabledTools; conversation.updatedAt = now(); persist()
  }

  function updateSkill(id: string, skillID: string, skillTools: string[]) {
    const conversation = getConversation(id)
    if (!conversation) return
    conversation.skillID = skillID
    conversation.enabledTools = skillTools
    conversation.updatedAt = now(); persist()
  }

  function updateMessage(id: string, messageID: string, content: string) {
    const message = getConversation(id)?.messages.find((item) => item.id === messageID)
    if (!message || !content.trim()) return
    message.content = content.trim(); persist()
  }

  async function generate(conversation: Conversation, assistantMessage: ChatMessage) {
    generatingConversationID.value = conversation.id
    abortController = new AbortController()
    try {
      await streamStatelessChat({
        model: conversation.model || undefined,
        temperature: temperature.value,
        messages: conversation.messages.filter((message) => message.id !== assistantMessage.id).map(({ role, content }) => ({ role, content })),
      }, (delta) => { assistantMessage.content += delta }, abortController.signal)
      assistantMessage.status = 'complete'
    } catch (error) {
      assistantMessage.status = 'error'
      assistantMessage.content ||= error instanceof Error ? error.message : '聊天请求失败'
    } finally {
      conversation.updatedAt = now(); generatingConversationID.value = null; abortController = undefined; persist()
    }
  }

  async function sendMessage(id: string, content: string, attachments: AttachmentMeta[] = []) {
    const conversation = getConversation(id)
    if (!conversation || !content.trim() || isGenerating.value) return
    const userMessage: ChatMessage = { id: createID(), role: 'user', content: content.trim(), createdAt: now(), status: 'complete', attachments }
    const assistantMessage: ChatMessage = { id: createID(), role: 'assistant', content: '', createdAt: now(), status: 'streaming' }
    conversation.messages.push(userMessage, assistantMessage)
    if (conversation.title === '新对话') conversation.title = content.trim().slice(0, 30)
    conversation.updatedAt = now(); persist()
    await generate(conversation, assistantMessage)
  }

  function stopGenerating() { abortController?.abort() }

  async function regenerate(id: string) {
    const conversation = getConversation(id)
    if (!conversation || isGenerating.value) return
    const index = conversation.messages.map((message) => message.role).lastIndexOf('assistant')
    if (index < 0) return
    conversation.messages.splice(index, 1)
    const assistantMessage: ChatMessage = { id: createID(), role: 'assistant', content: '', createdAt: now(), status: 'streaming' }
    conversation.messages.push(assistantMessage)
    await generate(conversation, assistantMessage)
  }

  function saveTemperature(value: number) { temperature.value = value; localStorage.setItem(SETTINGS_KEY, String(value)) }

  return { conversations, models, defaultModel, temperature, isGenerating, loadModels, getConversation, createConversation, renameConversation, removeConversation, updateConversationModel, updateTools, updateSkill, updateMessage, sendMessage, stopGenerating, regenerate, saveTemperature }
})
