<script setup>
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { computed, nextTick, onMounted, ref, watch } from 'vue'

const STORAGE_KEY = 'ai-agent-chat-history-v1'
const apiBase = import.meta.env.VITE_API_BASE_URL || ''

const conversations = ref(readConversations())
const currentID = ref(conversations.value[0]?.id || '')
const draft = ref('')
const isSending = ref(false)
const sidebarOpen = ref(false)
const showModelMenu = ref(false)
const availableModels = ref([])
const selectedModel = ref('')
const messagesPane = ref(null)
const bottomAnchor = ref(null)
const copiedMessageIndex = ref(-1)
let abortController
let scrollFrame

marked.setOptions({ breaks: true, gfm: true })

const currentConversation = computed(() =>
  conversations.value.find((item) => item.id === currentID.value),
)
const messages = computed(() => currentConversation.value?.messages || [])
const hasMessages = computed(() => messages.value.length > 0)

watch(conversations, persistConversations, { deep: true })
watch(messages, scrollToBottom, { deep: true })

function readConversations() {
  try {
    const saved = JSON.parse(localStorage.getItem(STORAGE_KEY))
    return Array.isArray(saved) ? saved : []
  } catch {
    return []
  }
}

function persistConversations() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(conversations.value))
}

function createConversation() {
  const conversation = {
    id: crypto.randomUUID?.() || `${Date.now()}-${Math.random()}`,
    title: '新对话',
    updatedAt: Date.now(),
    messages: [],
  }
  conversations.value.unshift(conversation)
  currentID.value = conversation.id
  sidebarOpen.value = false
  return conversation
}

function newConversation() {
  if (!isSending.value) createConversation()
}

function selectConversation(id) {
  if (isSending.value) return
  currentID.value = id
  sidebarOpen.value = false
}

function removeConversation(id) {
  if (isSending.value) return
  const index = conversations.value.findIndex((item) => item.id === id)
  if (index === -1) return
  conversations.value.splice(index, 1)
  if (currentID.value === id) currentID.value = conversations.value[0]?.id || ''
}

function clearConversations() {
  if (isSending.value || !conversations.value.length) return
  conversations.value = []
  currentID.value = ''
}

function updateDraft(event) {
  draft.value = event.target.value
  autoResize(event.target)
}

function autoResize(element) {
  element.style.height = 'auto'
  element.style.height = `${Math.min(element.scrollHeight, 200)}px`
}

function onKeydown(event) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    sendMessage()
  }
}

async function sendMessage(seedPrompt) {
  const text = (seedPrompt || draft.value).trim()
  if (!text || isSending.value) return

  let conversation = currentConversation.value
  if (!conversation) conversation = createConversation()

  conversation.messages.push({ role: 'user', content: text })
  if (conversation.messages.length === 1) conversation.title = makeTitle(text)
  conversation.updatedAt = Date.now()
  draft.value = ''
  isSending.value = true

  conversation.messages.push({ role: 'assistant', content: '' })
  // 必须从响应式数组取回代理对象；修改原始对象不会稳定触发流式 DOM 更新。
  const assistantMessage = conversation.messages[conversation.messages.length - 1]
  void scrollToBottom()
  abortController = new AbortController()

  try {
    const response = await fetch(`${apiBase}/api/chat/stream`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: selectedModel.value || undefined,
        messages: conversation.messages.slice(0, -1),
      }),
      signal: abortController.signal,
    })
    if (!response.ok || !response.body) {
      const error = await readError(response)
      throw new Error(error)
    }
    await consumeEventStream(response.body, (event, payload) => {
      if (event === 'delta') {
        assistantMessage.content += payload.data?.delta || ''
        // 流式 token 高频到达时，主动贴底比仅依赖深度 watch 更可靠。
        void scrollToBottom()
      }
      if (event === 'error') throw new Error(payload.message || '模型服务异常')
    })
    if (!assistantMessage.content.trim()) assistantMessage.content = '模型没有返回可显示的内容。'
  } catch (error) {
    if (error.name === 'AbortError') {
      assistantMessage.content ||= '已停止生成。'
    } else {
      assistantMessage.content = `抱歉，暂时无法完成请求：${error.message}`
    }
  } finally {
    conversation.updatedAt = Date.now()
    isSending.value = false
    abortController = undefined
  }
}

async function loadModels() {
  try {
    const response = await fetch(`${apiBase}/api/models`)
    if (!response.ok) throw new Error('无法获取模型列表')
    const payload = await response.json()
    availableModels.value = payload.data?.models || []
    selectedModel.value = payload.data?.default_model || availableModels.value[0]?.id || ''
  } catch {
    // 后端尚未启动时保留空状态，发送时会显示明确错误。
  }
}

function selectModel(modelID) {
  if (isSending.value) return
  selectedModel.value = modelID
  showModelMenu.value = false
}

const selectedModelLabel = computed(() =>
  availableModels.value.find((model) => model.id === selectedModel.value)?.label || 'AI Agent',
)

async function consumeEventStream(body, onEvent) {
  const reader = body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  while (true) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const blocks = buffer.split('\n\n')
    buffer = blocks.pop() || ''
    for (const block of blocks) {
      const event = block.match(/^event:\s*(.+)$/m)?.[1]?.trim() || 'message'
      const data = block
        .split('\n')
        .filter((line) => line.startsWith('data:'))
        .map((line) => line.slice(5).trim())
        .join('\n')
      if (data) onEvent(event, JSON.parse(data))
    }
  }
}

async function readError(response) {
  try {
    return (await response.json()).error || `请求失败（${response.status}）`
  } catch {
    return `请求失败（${response.status}）`
  }
}

function stopGenerating() {
  abortController?.abort()
}

function renderMarkdown(content) {
  if (!content) return ''
  return DOMPurify.sanitize(marked.parse(content, { async: false }), {
    USE_PROFILES: { html: true },
    ADD_ATTR: ['target'],
  })
}

async function copyMessage(content, index) {
  try {
    await navigator.clipboard.writeText(content)
    copiedMessageIndex.value = index
    window.setTimeout(() => {
      if (copiedMessageIndex.value === index) copiedMessageIndex.value = -1
    }, 1600)
  } catch {
    // 浏览器未授权剪贴板时静默失败，不影响阅读与对话。
  }
}

function makeTitle(text) {
  return text.length > 22 ? `${text.slice(0, 22)}…` : text
}

function formatDate(timestamp) {
  return new Intl.DateTimeFormat('zh-CN', { month: 'numeric', day: 'numeric' }).format(timestamp)
}

async function scrollToBottom() {
  await nextTick()
  cancelAnimationFrame(scrollFrame)
  scrollFrame = requestAnimationFrame(() => {
    if (!messagesPane.value) return
    // scrollIntoView 会让最近的可滚动祖先（messages-pane）对齐到底部。
    bottomAnchor.value?.scrollIntoView({ block: 'end', behavior: 'auto' })
    messagesPane.value.scrollTop = messagesPane.value.scrollHeight
  })
}

onMounted(() => {
  scrollToBottom()
  loadModels()
})
</script>

<template>
  <div class="app-shell">
    <div v-if="sidebarOpen" class="backdrop" @click="sidebarOpen = false" />

    <aside class="sidebar" :class="{ open: sidebarOpen }">
      <div class="sidebar-top">
        <button class="new-chat" type="button" @click="newConversation">
          <span class="new-chat-icon">＋</span>
          <span>新建对话</span>
          <kbd>⌘ K</kbd>
        </button>
      </div>

      <nav class="history" aria-label="对话历史">
        <div class="history-heading"><span>对话历史</span><span>{{ conversations.length }}</span></div>
        <div v-if="!conversations.length" class="empty-history">还没有对话记录</div>
        <div v-for="conversation in conversations" :key="conversation.id" class="history-item"
          :class="{ active: conversation.id === currentID }">
          <button type="button" class="history-select" @click="selectConversation(conversation.id)">
            <span class="history-dot" />
            <span class="history-copy"><span class="history-title">{{ conversation.title }}</span><span class="history-date">{{ formatDate(conversation.updatedAt) }}</span></span>
          </button>
          <button class="delete-chat" type="button" aria-label="删除对话" @click="removeConversation(conversation.id)">×</button>
        </div>
      </nav>

      <div class="sidebar-bottom">
        <button type="button" class="sidebar-action" @click="clearConversations"><span>⌫</span> 清空对话记录</button>
        <div class="profile"><span class="avatar">A</span><span class="profile-copy"><strong>本地 AI Agent</strong><small>Personal workspace</small></span><span class="more">•••</span></div>
      </div>
    </aside>

    <main class="chat-area">
      <header class="topbar">
        <button class="menu-button" type="button" aria-label="打开侧栏" @click="sidebarOpen = true">☰</button>
        <div class="model-switcher">
          <button class="model-picker" type="button" @click="showModelMenu = !showModelMenu">
            <span class="status-dot" />{{ selectedModelLabel }} <span class="chevron">⌄</span>
          </button>
          <div v-if="showModelMenu" class="model-menu">
            <button v-for="model in availableModels" :key="model.id" type="button"
              :class="{ selected: model.id === selectedModel }" @click="selectModel(model.id)">
              <span class="model-menu-dot" /><span><strong>{{ model.label }}</strong><small>{{ model.id }}</small></span><b v-if="model.id === selectedModel">✓</b>
            </button>
            <p v-if="!availableModels.length">后端未连接</p>
          </div>
        </div>
        <div class="topbar-spacer" />
        <button class="share-button" type="button"><span>↗</span> 分享</button>
        <button class="account-button" type="button" aria-label="账户">A</button>
      </header>

      <section ref="messagesPane" class="messages-pane">
        <div v-if="!hasMessages" class="welcome">
          <div class="welcome-kicker"><span /> AI AGENT WORKSPACE</div>
          <div class="welcome-mark">✦</div>
          <h1>从一个好问题开始</h1>
          <p>探索想法、组织信息、解决复杂任务。你的对话会保留在这个浏览器中。</p>
          <div class="suggestions">
            <button type="button" @click="sendMessage('帮我制定一个本周的学习计划')"><span class="suggestion-icon">◫</span><span><strong>制定学习计划</strong><small>结合目标安排本周任务</small></span><b>→</b></button>
            <button type="button" @click="sendMessage('帮我解释一个复杂概念，要通俗易懂')"><span class="suggestion-icon">◇</span><span><strong>解释复杂概念</strong><small>用清晰的方式快速理解</small></span><b>→</b></button>
            <button type="button" @click="sendMessage('帮我把下面的需求整理成开发任务清单')"><span class="suggestion-icon">⌘</span><span><strong>拆解开发任务</strong><small>把需求变成可执行步骤</small></span><b>→</b></button>
            <button type="button" @click="sendMessage('请帮我润色一段工作邮件')"><span class="suggestion-icon">✎</span><span><strong>润色一段邮件</strong><small>表达得更自然、专业</small></span><b>→</b></button>
          </div>
        </div>

        <div v-else class="message-list">
          <article v-for="(message, index) in messages" :key="index" class="message-row" :class="message.role">
            <template v-if="message.role === 'assistant'">
              <div class="assistant-avatar">✦</div>
              <div class="assistant-message">
                <div class="assistant-label"><span>AI AGENT</span><i /> <small>{{ isSending && index === messages.length - 1 ? '正在思考' : '刚刚' }}</small></div>
                <div class="message-content markdown-body" :class="{ pending: isSending && index === messages.length - 1 && !message.content }">
                  <div v-if="message.content" v-html="renderMarkdown(message.content)" />
                  <span v-else class="thinking"><i /><i /><i /></span>
                  <span v-if="isSending && index === messages.length - 1 && message.content" class="stream-cursor" />
                </div>
                <div v-if="message.content && !isSending" class="message-tools">
                  <button type="button" @click="copyMessage(message.content, index)">{{ copiedMessageIndex === index ? '已复制' : '复制回答' }}</button>
                  <button type="button" @click="sendMessage('请继续展开上一条回答中的关键内容')">继续展开</button>
                </div>
              </div>
            </template>
            <div v-else class="user-message">
              <div class="user-label">YOU</div>
              <div class="message-content">{{ message.content }}</div>
            </div>
          </article>
          <div ref="bottomAnchor" class="scroll-anchor" aria-hidden="true" />
        </div>
      </section>

      <footer class="composer-wrap">
        <div class="composer">
          <textarea v-model="draft" rows="1" placeholder="给 AI Agent 发送消息…" :disabled="isSending"
            @input="updateDraft" @keydown="onKeydown" />
          <div class="composer-actions">
            <button type="button" class="attach-button" aria-label="添加附件" title="添加附件">＋ <span>添加上下文</span></button>
            <span class="composer-spacer" />
            <button v-if="isSending" type="button" class="stop-button" @click="stopGenerating"><span /> 停止生成</button>
            <span v-if="!isSending" class="enter-hint">Enter 发送</span>
            <button v-if="!isSending" type="button" class="send-button" :disabled="!draft.trim()" aria-label="发送消息" @click="sendMessage">↑</button>
          </div>
        </div>
        <p class="disclaimer">AI Agent 可能会犯错；请核实重要信息。</p>
      </footer>
    </main>
  </div>
</template>
