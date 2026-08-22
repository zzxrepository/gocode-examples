<script setup lang="ts">
import { ref } from 'vue'
import { useChatStore } from '@/stores/chat'
import { usePreferencesStore } from '@/stores/preferences'
import { toolCatalog } from '@/types/chat'

const store = useChatStore()
const preferences = usePreferencesStore()
const temperature = ref(store.temperature)
function save() { store.saveTemperature(Number(temperature.value)) }
</script>

<template>
  <section class="settings-page">
    <header><RouterLink to="/chat/new">‹ 返回对话</RouterLink><h1>设置</h1><p>这些配置当前保存在浏览器；用户设置接口尚待后端实现。</p></header>
    <div class="settings-card">
      <h2>生成参数</h2>
      <label>Temperature <output>{{ Number(temperature).toFixed(1) }}</output><input v-model="temperature" type="range" min="0" max="2" step="0.1" @change="save" /></label>
      <p>数值越高，回答越有创造性。该值已经会传给当前的 `/api/chat/stream`。</p>
    </div>
    <div class="settings-card"><h2>界面偏好</h2><label class="setting-row"><span><strong>外观</strong><small>在浅色、深色与系统设置之间切换</small></span><select :value="preferences.theme" @change="preferences.setTheme(($event.target as HTMLSelectElement).value as 'light' | 'dark' | 'system')"><option value="system">跟随系统</option><option value="light">浅色</option><option value="dark">深色</option></select></label><label class="setting-row"><span><strong>紧凑布局</strong><small>缩小消息与侧栏间距</small></span><input :checked="preferences.compactMode" type="checkbox" @change="preferences.setCompactMode(($event.target as HTMLInputElement).checked)" /></label></div>
    <div class="settings-card"><h2>预留工具能力</h2><ul><li v-for="tool in toolCatalog" :key="tool.id"><strong>{{ tool.label }}</strong> — {{ tool.detail }}</li></ul></div>
  </section>
</template>
