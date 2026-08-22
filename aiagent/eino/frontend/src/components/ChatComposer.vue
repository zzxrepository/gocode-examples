<script setup lang="ts">
import { computed, ref } from 'vue'
import { skillCatalog, toolCatalog, type AttachmentMeta } from '@/types/chat'

const props = defineProps<{ disabled: boolean; model: string; skillID: string; models: { id: string; label: string }[]; tools: string[] }>()
const emit = defineEmits<{ send: [content: string, attachments: AttachmentMeta[]]; stop: []; modelChange: [model: string]; toolsChange: [tools: string[]]; skillChange: [skillID: string, tools: string[]] }>()

const content = ref('')
const attachments = ref<AttachmentMeta[]>([])
const fileInput = ref<HTMLInputElement>()
const textarea = ref<HTMLTextAreaElement>()
const showCapabilities = ref(false)
const selectedSkill = computed(() => skillCatalog.find((item) => item.id === props.skillID) ?? skillCatalog[0])

function submit() {
  if (!content.value.trim() || props.disabled) return
  emit('send', content.value, attachments.value)
  content.value = ''; attachments.value = []; resize()
}
function onKeydown(event: KeyboardEvent) { if (event.key === 'Enter' && !event.shiftKey) { event.preventDefault(); submit() } }
function appendFiles(files: File[]) { attachments.value.push(...files.map((file) => ({ id: crypto.randomUUID(), name: file.name, size: file.size, type: file.type, localOnly: true }))) }
function addFiles(event: Event) { appendFiles(Array.from((event.target as HTMLInputElement).files ?? [])); (event.target as HTMLInputElement).value = '' }
function onPaste(event: ClipboardEvent) { const files = Array.from(event.clipboardData?.files ?? []); if (files.length) appendFiles(files) }
function onDrop(event: DragEvent) { appendFiles(Array.from(event.dataTransfer?.files ?? [])) }
function resize() { if (textarea.value) { textarea.value.style.height = 'auto'; textarea.value.style.height = `${Math.min(textarea.value.scrollHeight, 180)}px` } }
function toggleTool(id: string) { emit('toolsChange', props.tools.includes(id) ? props.tools.filter((item) => item !== id) : [...props.tools, id]) }
function selectSkill(id: string) { const skill = skillCatalog.find((item) => item.id === id); if (skill) emit('skillChange', skill.id, [...skill.tools]) }
</script>

<template>
  <section class="composer-wrap">
    <div v-if="attachments.length" class="pending-attachments"><span v-for="attachment in attachments" :key="attachment.id">📎 {{ attachment.name }} <button type="button" @click="attachments = attachments.filter((item) => item.id !== attachment.id)">×</button></span></div>
    <div class="composer" @dragover.prevent @drop.prevent="onDrop">
      <textarea ref="textarea" v-model="content" rows="1" placeholder="输入消息，Enter 发送，Shift + Enter 换行" :disabled="disabled" @input="resize" @keydown="onKeydown" @paste="onPaste" />
      <div class="composer-actions">
        <input ref="fileInput" class="visually-hidden" type="file" multiple @change="addFiles" />
        <button type="button" class="icon-button" title="添加资料（上传接口待实现）" :disabled="disabled" @click="fileInput?.click()">＋</button>
        <div class="capability-menu"><button type="button" class="capability-button" :class="{ enabled: tools.length }" @click="showCapabilities = !showCapabilities">工具与 Skill ⌄</button><div v-if="showCapabilities" class="capability-popover"><p class="capability-title">选择工作模式</p><button v-for="skill in skillCatalog" :key="skill.id" type="button" class="skill-option" :class="{ active: skill.id === selectedSkill.id }" @click="selectSkill(skill.id)"><strong>{{ skill.label }}</strong><small>{{ skill.detail }}</small></button><p class="capability-title">按需开启</p><label v-for="tool in toolCatalog" :key="tool.id" class="tool-option"><input type="checkbox" :checked="tools.includes(tool.id)" @change="toggleTool(tool.id)" /><span><strong>{{ tool.icon }} {{ tool.label }}</strong><small>{{ tool.detail }}</small></span></label><p class="tool-pending">选择会保存到会话；真正执行由后端 RAG/MCP/Skill 服务完成。</p></div></div>
        <select :value="model" aria-label="选择模型" @change="emit('modelChange', ($event.target as HTMLSelectElement).value)"><option value="">默认模型</option><option v-for="item in models" :key="item.id" :value="item.id">{{ item.label }}</option></select>
        <button v-if="disabled" type="button" class="stop-button" title="停止生成" @click="emit('stop')">■</button><button v-else type="button" class="send-button" :disabled="!content.trim()" title="发送" @click="submit">↑</button>
      </div>
    </div>
    <p class="composer-hint">{{ selectedSkill.label }}{{ tools.length ? ` · 已选择 ${tools.length} 项能力` : '' }}。文件、RAG、MCP 与 Skill 需要后端接口后才会执行。</p>
  </section>
</template>
