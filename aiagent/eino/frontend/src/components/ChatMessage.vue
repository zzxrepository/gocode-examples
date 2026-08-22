<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ChatMessage } from '@/types/chat'
import { renderMarkdown } from '@/utils/markdown'

const props = defineProps<{ message: ChatMessage }>()
const emit = defineEmits<{ update: [content: string] }>()
const editing = ref(false)
const draft = ref('')
const copied = ref(false)
const renderedContent = computed(() => renderMarkdown(props.message.content))

function startEdit() { draft.value = props.message.content; editing.value = true }
function saveEdit() { if (draft.value.trim()) emit('update', draft.value); editing.value = false }
async function copyContent() {
  await navigator.clipboard.writeText(props.message.content)
  copied.value = true
  window.setTimeout(() => { copied.value = false }, 1200)
}
</script>

<template>
  <article class="message" :class="message.role">
    <div class="message-avatar">{{ message.role === 'user' ? '你' : '✦' }}</div>
    <div class="message-body">
      <p class="message-role">{{ message.role === 'user' ? '你' : 'Aurora AI' }}</p>
      <div v-if="message.attachments?.length" class="attachment-list"><span v-for="attachment in message.attachments" :key="attachment.id" class="attachment-chip">📎 {{ attachment.name }}<small>等待上传</small></span></div>
      <div v-if="message.toolRuns?.length" class="tool-runs"><div v-for="run in message.toolRuns" :key="run.id" class="tool-run"><span>{{ run.status === 'complete' ? '✓' : run.status === 'failed' ? '!' : '◌' }}</span><span><strong>{{ run.name }}</strong><small>{{ run.outputSummary || run.inputSummary || '正在调用工具' }}</small></span></div></div>
      <textarea v-if="editing" v-model="draft" class="message-editor" rows="4" @keydown.meta.enter="saveEdit" @keydown.ctrl.enter="saveEdit" />
      <div v-else class="message-content markdown-body" :class="{ error: message.status === 'error' }" v-html="message.content ? renderedContent : (message.status === 'streaming' ? '正在生成…' : '')"></div>
      <div v-if="message.citations?.length" class="citation-list"><a v-for="citation in message.citations" :key="citation.id" :href="citation.url" target="_blank" rel="noreferrer">{{ citation.source || citation.title }}</a></div>
      <span v-if="message.status === 'streaming'" class="streaming-dot" aria-label="正在生成"></span>
      <div v-if="message.status !== 'streaming' && message.content" class="message-actions"><template v-if="editing"><button type="button" @click="saveEdit">保存</button><button type="button" @click="editing = false">取消</button></template><template v-else><button type="button" @click="copyContent">{{ copied ? '已复制' : '复制' }}</button><button v-if="message.role === 'user'" type="button" @click="startEdit">编辑</button></template></div>
    </div>
  </article>
</template>
