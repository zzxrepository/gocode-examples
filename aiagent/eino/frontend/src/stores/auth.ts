import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export interface LocalUser {
  id: string
  displayName: string
  email: string
}

const STORAGE_KEY = 'eino-ai-chat.local-user.v1'

function loadUser(): LocalUser | null {
  try { return JSON.parse(localStorage.getItem(STORAGE_KEY) ?? 'null') as LocalUser | null } catch { return null }
}

// 后端尚未提供认证接口时，仅用于演示登录、注册页面和前端路由流程。
// 不保存密码，也绝不能用于生产认证。
export const useAuthStore = defineStore('auth', () => {
  const user = ref<LocalUser | null>(loadUser())
  const isAuthenticated = computed(() => user.value !== null)

  function save(nextUser: LocalUser) {
    user.value = nextUser
    localStorage.setItem(STORAGE_KEY, JSON.stringify(nextUser))
  }

  function signInLocal(account: string) {
    const normalized = account.trim()
    save({ id: `local-${crypto.randomUUID()}`, displayName: normalized.split('@')[0] || 'Aurora 用户', email: normalized.includes('@') ? normalized : '' })
  }

  function registerLocal(displayName: string, email: string) {
    save({ id: `local-${crypto.randomUUID()}`, displayName: displayName.trim(), email: email.trim() })
  }

  function signOut() {
    user.value = null
    localStorage.removeItem(STORAGE_KEY)
  }

  return { user, isAuthenticated, signInLocal, registerLocal, signOut }
})
