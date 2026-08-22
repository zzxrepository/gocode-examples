import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { usePreferencesStore } from './stores/preferences'
import './styles.css'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
usePreferencesStore(pinia).applyTheme()
app.use(router).mount('#app')
