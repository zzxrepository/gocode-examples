import { ref } from 'vue'
import { defineStore } from 'pinia'

export type Theme = 'light' | 'dark' | 'system'
const KEY = 'eino-ai-chat.preferences.v1'
const COMPACT_KEY = 'eino-ai-chat.compact-mode.v1'

function loadTheme(): Theme {
  const value = localStorage.getItem(KEY)
  return value === 'light' || value === 'dark' || value === 'system' ? value : 'system'
}

export const usePreferencesStore = defineStore('preferences', () => {
  const theme = ref<Theme>(loadTheme())
  const compactMode = ref(localStorage.getItem(COMPACT_KEY) === 'true')
  const mobileSidebarOpen = ref(false)

  function applyTheme() {
    const resolved = theme.value === 'system' ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light') : theme.value
    document.documentElement.dataset.theme = resolved
    document.documentElement.dataset.density = compactMode.value ? 'compact' : 'comfortable'
  }

  function setTheme(next: Theme) {
    theme.value = next
    localStorage.setItem(KEY, next)
    applyTheme()
  }

  function setCompactMode(next: boolean) {
    compactMode.value = next
    localStorage.setItem(COMPACT_KEY, String(next))
    document.documentElement.dataset.density = next ? 'compact' : 'comfortable'
  }

  function toggleMobileSidebar() { mobileSidebarOpen.value = !mobileSidebarOpen.value }
  function closeMobileSidebar() { mobileSidebarOpen.value = false }

  return { theme, compactMode, mobileSidebarOpen, applyTheme, setTheme, setCompactMode, toggleMobileSidebar, closeMobileSidebar }
})
