<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import { usePreferencesStore } from '@/stores/preferences'

defineProps<{ mobileOpen: boolean }>()

const chat = useChatStore()
const auth = useAuthStore()
const preferences = usePreferencesStore()
const route = useRoute()
const router = useRouter()
const keyword = ref('')
const editingID = ref<string | null>(null)
const draftTitle = ref('')

const conversations = computed(() => chat.conversations.filter((item) => item.title.toLowerCase().includes(keyword.value.trim().toLowerCase())))
const currentID = computed(() => String(route.params.conversationId ?? ''))

function createConversation() {
  const conversation = chat.createConversation()
  preferences.closeMobileSidebar()
  void router.push(`/chat/${conversation.id}`)
}
function openConversation(id: string) { preferences.closeMobileSidebar(); void router.push(`/chat/${id}`) }
function startRename(id: string, title: string) { editingID.value = id; draftTitle.value = title }
function finishRename(id: string) { chat.renameConversation(id, draftTitle.value); editingID.value = null }
function removeConversation(id: string) {
  if (!window.confirm('删除这个对话？浏览器中的本地记录将被移除。')) return
  chat.removeConversation(id)
  if (currentID.value === id) void router.push('/chat/new')
}
function signOut() { auth.signOut(); void router.replace('/login') }
</script>

<template>
  <aside class="sidebar" :class="{ 'mobile-open': mobileOpen }">
    <div class="brand"><span class="brand-mark">✦</span><span>Aurora AI</span></div>
    <button class="new-chat" type="button" @click="createConversation"><span>＋</span> 新建对话</button>
    <label class="search"><span>⌕</span><input v-model="keyword" placeholder="搜索对话" /></label>
    <nav class="conversation-list" aria-label="对话历史">
      <p class="list-label">对话历史</p>
      <p v-if="conversations.length === 0" class="empty-list">还没有对话，开始问一个问题吧。</p>
      <div v-for="conversation in conversations" :key="conversation.id" class="conversation-row" :class="{ active: currentID === conversation.id }">
        <button class="conversation-title" type="button" @click="openConversation(conversation.id)"><span>◌</span><input v-if="editingID === conversation.id" v-model="draftTitle" aria-label="对话标题" @click.stop @keydown.enter="finishRename(conversation.id)" @blur="finishRename(conversation.id)" /><span v-else>{{ conversation.title }}</span></button>
        <div class="conversation-actions"><button type="button" title="重命名" @click.stop="startRename(conversation.id, conversation.title)">✎</button><button type="button" title="删除" @click.stop="removeConversation(conversation.id)">⌫</button></div>
      </div>
    </nav>
    <div class="sidebar-footer"><RouterLink to="/settings" class="sidebar-link">⚙ 设置</RouterLink><div class="user-row"><span class="user-avatar">{{ auth.user?.displayName.slice(0, 1) || '你' }}</span><span>{{ auth.user?.displayName || '本地用户' }}</span><button type="button" @click="signOut">退出</button></div><span class="storage-note">历史暂存于浏览器，接入数据库后跨端同步</span></div>
  </aside>
</template>
