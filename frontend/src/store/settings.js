import { defineStore } from 'pinia'
import { settingsAPI } from '@/utils/api'

const EMAIL_PATTERN = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/
const PASSWORD_PATTERNS = {
  letter: /[a-zA-Z]/,
  digit: /\d/,
  special: /[!@#$%^&*()_+\-=\[\]{}|;:,.<>?]/
}

export const useSettingsStore = defineStore('settings', {
  state: () => ({
    siteName: 'CBoard',
    siteDescription: '高性能面板系统',
    siteKeywords: '面板,管理,系统',
    siteLogo: '',
    siteFavicon: '',
    allowRegistration: true,
    requireEmailVerification: true,
    allowQqEmailOnly: true,
    minPasswordLength: 8,
    defaultTheme: 'default',
    allowUserTheme: true,
    availableThemes: ['default', 'dark', 'blue', 'green'],
    enablePayment: true,
    defaultPaymentMethod: '',
    paymentCurrency: 'CNY',
    enableAnnouncement: true,
    announcementPosition: 'top',
    maxAnnouncements: 5,
    loading: false,
    error: null
  }),

  getters: {
    siteTitle: (state) => state.siteName,
    currentTheme: (state) => {
      const userTheme = localStorage.getItem('user-theme')
      if (state.allowUserTheme && userTheme && state.availableThemes.includes(userTheme)) {
        return userTheme
      }
      return state.defaultTheme
    },
    canRegister: (state) => state.allowRegistration,
    needsEmailVerification: (state) => state.requireEmailVerification,
    emailRestriction: (state) => state.allowQqEmailOnly,
    paymentEnabled: (state) => state.enablePayment,
    announcementEnabled: (state) => state.enableAnnouncement
  },

  actions: {
    async loadSettings() {
      this.loading = true
      this.error = null
      try {
        const response = await settingsAPI.getPublicSettings()
        const settings = response.data?.data || response.data || {}
        this.siteName = settings.site_name || 'CBoard'
        this.siteDescription = settings.site_description || '高性能面板系统'
        this.siteKeywords = settings.site_keywords || '面板,管理,系统'
        this.siteLogo = settings.site_logo || ''
        this.siteFavicon = settings.site_favicon || ''
        this.allowRegistration = settings.allowRegistration !== false
        this.requireEmailVerification = settings.require_email_verification !== false
        this.allowQqEmailOnly = settings.allow_qq_email_only !== false
        this.minPasswordLength = settings.min_password_length || 8
        this.defaultTheme = settings.default_theme || 'light'
        this.allowUserTheme = settings.allow_user_theme !== false
        this.availableThemes = settings.available_themes || ['light', 'dark', 'blue', 'green', 'purple', 'orange', 'red', 'cyan', 'luck', 'aurora', 'auto']
        this.enablePayment = settings.enable_payment !== false
        this.defaultPaymentMethod = settings.default_payment_method || ''
        this.paymentCurrency = settings.payment_currency || 'CNY'
        this.enableAnnouncement = settings.enable_announcement !== false
        this.announcementPosition = settings.announcement_position || 'top'
        this.maxAnnouncements = settings.max_announcements || 5
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
    validateEmail(email) {
      if (!email) return false
      return EMAIL_PATTERN.test(email)
    },
    validatePassword(password) {
      if (!password || password.length < this.minPasswordLength) return false
      return PASSWORD_PATTERNS.letter.test(password) && 
             PASSWORD_PATTERNS.digit.test(password) && 
             PASSWORD_PATTERNS.special.test(password)
    },
    getPasswordError(password) {
      if (!password) return '请输入密码'
      if (password.length < this.minPasswordLength) {
        return `密码长度至少${this.minPasswordLength}位`
      }
      if (!PASSWORD_PATTERNS.letter.test(password)) return '密码必须包含字母'
      if (!PASSWORD_PATTERNS.digit.test(password)) return '密码必须包含数字'
      if (!PASSWORD_PATTERNS.special.test(password)) return '密码必须包含特殊字符'
      return null
    },
    getEmailError(email) {
      if (!email) return '请输入邮箱'
      if (!EMAIL_PATTERN.test(email)) return '邮箱格式不正确'
      return null
    },
    resetSettings() {
      this.siteName = 'CBoard'
      this.siteDescription = '高性能面板系统'
      this.siteKeywords = '面板,管理,系统'
      this.siteLogo = ''
      this.siteFavicon = ''
      this.allowRegistration = true
      this.requireEmailVerification = true
      this.allowQqEmailOnly = false
      this.minPasswordLength = 8
      this.defaultTheme = 'default'
      this.allowUserTheme = true
      this.availableThemes = ['default', 'dark', 'blue', 'green']
      this.enablePayment = true
      this.defaultPaymentMethod = ''
      this.paymentCurrency = 'CNY'
      this.enableAnnouncement = true
      this.announcementPosition = 'top'
      this.maxAnnouncements = 5
    }
  }
}) 