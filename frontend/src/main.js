import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import App from './App.vue'
import router from './router'
import { useSettingsStore } from './store/settings'
import { useAuthStore } from './store/auth'
import { useThemeStore } from './store/theme'
import { initApi } from './utils/api'
import './styles/global.scss'
import './styles/mobile-buttons.scss'
import './styles/text-selection.css'
import { initTextSelection } from './utils/textSelection'
const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
app.use(router)
initApi(router, useAuthStore)
app.use(ElementPlus, {
  locale: zhCn,
})
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}
app.config.errorHandler = (err, vm, info) => {
  if (process.env.NODE_ENV === 'development') {
    console.error('Vue error:', err, info)
  }
}
app.config.globalProperties.$auth = useAuthStore()
app.config.globalProperties.$settings = null
app.mount('#app')
setTimeout(() => {
  initTextSelection()
}, 200)
if (typeof window !== 'undefined') {
  import('axios').then(({ default: axios }) => {
    axios.get('/api/v1/settings/public-settings', {
      withCredentials: true,
      timeout: 5000
    }).catch(() => {
    })
  })
}
setTimeout(async () => {
  try {
    const settingsStore = useSettingsStore()
    await settingsStore.loadSettings()
    app.config.globalProperties.$settings = settingsStore
    setTimeout(async () => {
      try {
        const themeStore = useThemeStore()
        const userTheme = typeof window !== 'undefined' ? localStorage.getItem('user-theme') : null
        if (userTheme) {
          themeStore.applyTheme(userTheme)
        } else if (settingsStore.defaultTheme) {
          themeStore.applyTheme(settingsStore.defaultTheme)
        } else {
          themeStore.applyTheme('light')
        }
      } catch (e) {
      }
    }, 500)
  } catch (e) {
  }
}, 100)