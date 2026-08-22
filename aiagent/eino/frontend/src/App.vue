<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppSidebar from '@/components/AppSidebar.vue'
import { usePreferencesStore } from '@/stores/preferences'

const route = useRoute()
const preferences = usePreferencesStore()
const isAuthPage = computed(() => route.meta.layout === 'auth')
</script>

<template>
  <RouterView v-if="isAuthPage" />
  <main v-else class="app-shell">
    <AppSidebar :mobile-open="preferences.mobileSidebarOpen" />
    <RouterView />
  </main>
</template>
