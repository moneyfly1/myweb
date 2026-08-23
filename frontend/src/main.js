import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { ElLoading } from 'element-plus/es/components/loading/index.mjs'
import { ElMessage } from 'element-plus/es/components/message/index.mjs'
import { ElMessageBox } from 'element-plus/es/components/message-box/index.mjs'
import { ElNotification } from 'element-plus/es/components/notification/index.mjs'
import { provideGlobalConfig } from 'element-plus/es/components/config-provider/src/hooks/use-global-config.mjs'
import 'element-plus/es/components/loading/style/css'
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/message-box/style/css'
import 'element-plus/es/components/notification/style/css'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import App from './App.vue'
import router from './router'
import { useSettingsStore } from './store/settings'
import { useAuthStore } from './store/auth'
import { useThemeStore } from './store/theme'
import { initApi, cachedAPI } from './utils/api'
import './styles/global.scss'

// 全局消息配置：统一时长、分组去重（避免操作频繁时消息堆叠、遮挡关键操作）
ElMessage.config({
  duration: 2500,
  grouping: true,
  max: 3
})
ElNotification.config({
  duration: 3500,
  grouping: true
})

// 尽早预热公共设置缓存，让路由守卫直接命中缓存
cachedAPI.getPublicSettings().catch(() => {})

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
app.use(router)
initApi(router, useAuthStore)
provideGlobalConfig({ locale: zhCn }, app, true)
app.use(ElLoading)
app.use(ElMessage)
app.use(ElMessageBox)
app.use(ElNotification)

app.config.errorHandler = (err, vm, info) => {
  if (process.env.NODE_ENV === 'development') {
    console.error('Vue error:', err, info)
  }
}
app.config.globalProperties.$auth = useAuthStore()
app.config.globalProperties.$settings = null

// mount 前同步应用已保存的主题，消除首屏主题闪烁
try {
  const savedTheme = typeof window !== 'undefined' ? localStorage.getItem('user-theme') : null
  if (savedTheme) {
    useThemeStore().applyTheme(savedTheme)
  }
} catch (e) {
  // 主题应用失败不影响启动
}

app.mount('#app')

// 并发初始化所有任务，避免串行延迟
Promise.all([
  // 加载设置并应用主题
  (async () => {
    try {
      const settingsStore = useSettingsStore()
      await settingsStore.loadSettings()
      app.config.globalProperties.$settings = settingsStore

      // 应用主题（若 mount 前已应用则无需重复，但确保默认主题生效）
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
