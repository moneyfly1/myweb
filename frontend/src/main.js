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

// 导入全局样式
import './styles/global.scss'
import './styles/mobile-buttons.scss'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// 初始化 API 模块，解决循环依赖
initApi(router, useAuthStore)

app.use(ElementPlus, {
  locale: zhCn,
})

// 注册所有图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.config.errorHandler = (err, vm, info) => {
  if (process.env.NODE_ENV === 'development') {
    console.error('Vue error:', err, info)
  }
}

// 全局属性
app.config.globalProperties.$auth = useAuthStore()
app.config.globalProperties.$settings = null

// 立即挂载应用
app.mount('#app')

// 页面加载时主动获取 CSRF Token
if (typeof window !== 'undefined') {
  // 使用轻量级的 GET 请求获取 CSRF Token
  import('axios').then(({ default: axios }) => {
    axios.get('/api/v1/settings/public-settings', {
      withCredentials: true,
      timeout: 5000
    }).catch(() => {
      // 忽略错误，不影响应用启动
    })
  })
}

// 异步加载设置和主题（不阻塞应用启动）
setTimeout(async () => {
  try {
    const settingsStore = useSettingsStore()
    await settingsStore.loadSettings()
    app.config.globalProperties.$settings = settingsStore
    
    // 延迟初始化主题
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
        // 忽略主题错误
      }
    }, 500)
  } catch (e) {
    // 忽略设置加载错误
  }
}, 100)
 