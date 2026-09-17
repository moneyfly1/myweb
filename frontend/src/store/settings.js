import { defineStore } from 'pinia'
import { cachedAPI } from '@/utils/api'
const EMAIL_PATTERN = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/
export const useSettingsStore = defineStore('settings', {
  state: () => ({
    siteName: 'CBoard',
    siteFavicon: '',
    minPasswordLength: 8,
    defaultTheme: 'default',
    allowUserTheme: true,
    availableThemes: ['default', 'dark', 'blue', 'green'],
    loading: false,
    error: null
  }),
  getters: {
    currentTheme: (state) => {
      const userTheme = localStorage.getItem('user-theme')
      if (state.allowUserTheme && userTheme && state.availableThemes.includes(userTheme)) {
        return userTheme
      }
      return state.defaultTheme
    },
    // 密码长度提示文案的唯一来源，各表单统一引用，避免硬编码 6/8
    passwordMinLengthHint: (state) => `密码长度至少 ${state.minPasswordLength} 位`
  },
  actions: {
    async loadSettings() {
      this.loading = true
      this.error = null
      try {
        const response = await cachedAPI.getPublicSettings()
        const settings = response.data?.data || response.data || {}
        this.siteName = settings.site_name || 'CBoard'
        this.siteFavicon = settings.site_favicon || ''
        const minPasswordValue = settings.min_password_length !== undefined 
                               ? settings.min_password_length
                               : (settings.minPasswordLength !== undefined 
                                  ? settings.minPasswordLength 
                                  : 8)
        this.minPasswordLength = typeof minPasswordValue === 'number' ? minPasswordValue : (parseInt(minPasswordValue) || 8)
        this.defaultTheme = settings.default_theme || 'light'
        this.allowUserTheme = settings.allow_user_theme !== false
        this.availableThemes = settings.available_themes || ['light', 'dark', 'blue', 'green', 'purple', 'orange', 'red', 'cyan', 'luck', 'aurora', 'auto']
        document.title = this.siteName
        if (this.siteFavicon) {
          const link = document.querySelector("link[rel*='icon']") || document.createElement('link')
          link.type = 'image/x-icon'
          link.rel = 'shortcut icon'
          link.href = this.siteFavicon
          document.getElementsByTagName('head')[0].appendChild(link)
        }
      } catch (error) {
        this.error = error.message || '加载设置失败'
      } finally {
        this.loading = false
      }
    },
    setUserTheme(theme) {
      if (this.allowUserTheme && this.availableThemes.includes(theme)) {
        localStorage.setItem('user-theme', theme)
        this.applyTheme(theme)
      }
    },
    applyTheme(theme) {
      const themeClasses = ['default', 'dark', 'blue', 'green', 'light', 'purple', 'orange', 'red', 'cyan', 'luck', 'aurora', 'auto']
      document.documentElement.classList.remove(...themeClasses.map(t => `theme-${t}`))
      document.documentElement.classList.add(`theme-${theme}`)
      this.updateThemeVariables(theme)
    },
    updateThemeVariables(theme) {
      const root = document.documentElement
      const themeColors = {
        default: {
          '--primary-color': '#409eff',
          '--success-color': '#67c23a',
          '--warning-color': '#e6a23c',
          '--danger-color': '#f56c6c',
          '--info-color': '#909399',
          '--text-color': '#303133',
          '--text-color-secondary': '#606266',
          '--border-color': '#dcdfe6',
          '--background-color': '#ffffff',
          '--background-color-secondary': '#f5f7fa'
        },
        dark: {
          '--primary-color': '#409eff',
          '--success-color': '#67c23a',
          '--warning-color': '#e6a23c',
          '--danger-color': '#f56c6c',
          '--info-color': '#909399',
          '--text-color': '#ffffff',
          '--text-color-secondary': '#c0c4cc',
          '--border-color': '#4c4d4f',
          '--background-color': '#1d1e1f',
          '--background-color-secondary': '#2d2e2f'
        },
        blue: {
          '--primary-color': '#1890ff',
          '--success-color': '#52c41a',
          '--warning-color': '#faad14',
          '--danger-color': '#ff4d4f',
          '--info-color': '#8c8c8c',
          '--text-color': '#262626',
          '--text-color-secondary': '#595959',
          '--border-color': '#d9d9d9',
          '--background-color': '#ffffff',
          '--background-color-secondary': '#f0f2f5'
        },
        green: {
          '--primary-color': '#52c41a',
          '--success-color': '#389e0d',
          '--warning-color': '#d48806',
          '--danger-color': '#cf1322',
          '--info-color': '#8c8c8c',
          '--text-color': '#262626',
          '--text-color-secondary': '#595959',
          '--border-color': '#d9d9d9',
          '--background-color': '#ffffff',
          '--background-color-secondary': '#f6ffed'
        }
      }
      const colors = themeColors[theme] || themeColors.default
      Object.entries(colors).forEach(([key, value]) => {
        root.style.setProperty(key, value)
      })
    },
    initTheme() {
      this.applyTheme(this.currentTheme)
    },
    getEmailError(email) {
      if (!email) return '请输入邮箱'
      if (!EMAIL_PATTERN.test(email)) return '邮箱格式不正确'
      return null
    },
  }
}) 
