<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ChatComposer from '@/components/ChatComposer.vue'
import ChatMessage from '@/components/ChatMessage.vue'
import { useChatStore } from '@/stores/chat'
import { usePreferencesStore } from '@/stores/preferences'

const route = useRoute()
const router = useRouter()
const chat = useChatStore()
const preferences = usePreferencesStore()
const messageList = ref<HTMLElement>()
const conversationID = computed(() => String(route.params.conversationId ?? 'new'))
const conversation = computed(() => chat.getConversation(conversationID.value))

async function resolveConversation() {
  if (conversationID.value === 'new' || !chat.getConversation(conversationID.value)) {
    const created = chat.createConversation()
    await router.replace(`/chat/${created.id}`)
  }
}
function scrollToBottom() { void nextTick(() => messageList.value?.scrollTo({ top: messageList.value.scrollHeight, behavior: 'smooth' })) }
function send(content: string, attachments: Parameters<typeof chat.sendMessage>[2]) { void chat.sendMessage(conversationID.value, content, attachments).then(scrollToBottom); scrollToBottom() }
function selectSkill(skillID: string, tools: string[]) {
  if (conversation.value) chat.updateSkill(conversation.value.id, skillID, tools)
}
function exportConversation() {
  if (!conversation.value) return
  const markdown = conversation.value.messages.map((message) => `## ${message.role === 'user' ? '你' : 'Aurora AI'}\n\n${message.content}`).join('\n\n')
  const url = URL.createObjectURL(new Blob([markdown], { type: 'text/markdown;charset=utf-8' }))
  const anchor = document.createElement('a'); anchor.href = url; anchor.download = `${conversation.value.title || 'conversation'}.md`; anchor.click(); URL.revokeObjectURL(url)
}

onMounted(() => void chat.loadModels())
watch(conversationID, () => void resolveConversation(), { immediate: true })
watch(() => conversation.value?.messages.length, scrollToBottom)
</script>

<template>
  <section v-if="conversation" class="chat-page">
    <header class="chat-header"><div><button class="mobile-menu" type="button" @click="preferences.toggleMobileSidebar">☰</button><strong>{{ conversation.title }}</strong><span class="local-badge">本地会话</span></div><div class="header-actions"><span class="skill-badge">{{ conversation.skillID === 'rag_qa' ? '文档问答' : conversation.skillID === 'blog_operator' ? '博客助手' : '通用助手' }}</span><button type="button" title="导出本地对话" @click="exportConversation">⇩ 导出</button></div></header>
    <div ref="messageList" class="message-list"><div v-if="conversation.messages.length === 0" class="welcome"><div class="welcome-logo">✦</div><h1>今天想学习什么？</h1><p>选择模型、上传资料，或在工具菜单中启用 RAG、MCP 与 Skill。</p><div class="suggestions"><button type="button" @click="send('帮我解释 Go 中的 context。', [])">解释一个 Go 概念</button><button type="button" @click="send('帮我为一个 RAG 系统设计后端接口。', [])">设计 RAG 接口</button><button type="button" @click="send('如何设计一个安全的 MCP 工具调用？', [])">理解 MCP 安全</button></div></div><ChatMessage v-for="message in conversation.messages" :key="message.id" :message="message" @update="chat.updateMessage(conversation.id, message.id, $event)" /><button v-if="!chat.isGenerating && conversation.messages.some((message) => message.role === 'assistant')" class="regenerate" type="button" @click="chat.regenerate(conversation.id)">↻ 重新生成最后一条回复</button></div>
    <ChatComposer :disabled="chat.isGenerating" :model="conversation.model" :skill-i-d="conversation.skillID" :models="chat.models" :tools="conversation.enabledTools" @send="send" @stop="chat.stopGenerating" @model-change="chat.updateConversationModel(conversation.id, $event)" @tools-change="chat.updateTools(conversation.id, $event)" @skill-change="selectSkill" />
  </section>
</template>
