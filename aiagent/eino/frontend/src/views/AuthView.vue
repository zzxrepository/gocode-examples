<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const registerMode = computed(() => route.name === 'register')
const displayName = ref('')
const account = ref('')
const password = ref('')
const confirmPassword = ref('')
const error = ref('')

function submit() {
  error.value = ''
  if (!account.value.trim() || !password.value) { error.value = '请输入账号和密码。'; return }
  if (registerMode.value) {
    if (!displayName.value.trim()) { error.value = '请输入你的名称。'; return }
    if (password.value.length < 6) { error.value = '密码至少需要 6 位。'; return }
    if (password.value !== confirmPassword.value) { error.value = '两次输入的密码不一致。'; return }
    auth.registerLocal(displayName.value, account.value)
  } else {
    auth.signInLocal(account.value)
  }
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/chat/new'
  void router.replace(redirect)
}
</script>

<template>
  <main class="auth-page">
    <section class="auth-card">
      <RouterLink to="/" class="auth-brand"><span class="brand-mark">✦</span> Aurora AI</RouterLink>
      <div class="auth-intro"><h1>{{ registerMode ? '创建你的账户' : '欢迎回来' }}</h1><p>{{ registerMode ? '注册后即可开始你的第一个对话。' : '登录后继续你的 Aurora AI 对话。' }}</p></div>
      <form class="auth-form" @submit.prevent="submit">
        <label v-if="registerMode">名称<input v-model="displayName" autocomplete="name" placeholder="例如：张明章" /></label>
        <label>邮箱或用户名<input v-model="account" :autocomplete="registerMode ? 'email' : 'username'" placeholder="name@example.com" /></label>
        <label>密码<input v-model="password" type="password" :autocomplete="registerMode ? 'new-password' : 'current-password'" placeholder="至少 6 位" /></label>
        <label v-if="registerMode">确认密码<input v-model="confirmPassword" type="password" autocomplete="new-password" placeholder="再次输入密码" /></label>
        <p v-if="error" class="auth-error">{{ error }}</p>
        <button class="auth-submit" type="submit">{{ registerMode ? '创建账户' : '登录' }}</button>
      </form>
      <p class="auth-switch">{{ registerMode ? '已有账户？' : '还没有账户？' }} <RouterLink :to="registerMode ? '/login' : '/register'">{{ registerMode ? '去登录' : '创建账户' }}</RouterLink></p>
      <p class="auth-local-notice">演示模式：当前 Go 后端尚未实现认证接口。这里不会保存密码，仅创建浏览器本地登录状态。</p>
    </section>
  </main>
</template>
