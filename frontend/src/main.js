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
import { initApi, cachedAPI } from './utils/api'
import './styles/global.scss'
import './styles/mobile-buttons.scss'
import './styles/text-selection.css'
import { initTextSelection } from './utils/textSelection'

// 尽早预热公共设置缓存，让路由守卫直接命中缓存
cachedAPI.getPublicSettings().catch(() => {})

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
app.use(router)
initApi(router, useAuthStore)
app.use(ElementPlus, {
  locale: zhCn,
})

// 注册所有图标（多处组件使用字符串形式引用图标，需要全局注册）
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

// 并发初始化所有任务，避免串行延迟
Promise.all([
  // 初始化文本选择功能
  Promise.resolve().then(() => initTextSelection()),

  // 加载设置并应用主题
  (async () => {
    try {
      const settingsStore = useSettingsStore()
      await settingsStore.loadSettings()
      app.config.globalProperties.$settings = settingsStore

      // 应用主题
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
      console.error('初始化设置失败:', e)
    }
  })()
]).catch(err => {
  console.error('应用初始化失败:', err)
})