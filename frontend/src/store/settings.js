import { defineStore } from 'pinia'
import { cachedAPI } from '@/utils/api'
const EMAIL_PATTERN = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/
export const useSettingsStore = defineStore('settings', {
  state: () => ({
    siteName: 'CBoard',
    siteFavicon: '',
    minPasswordLength: 8,
    // 系统设置 → 默认设备数。后台文案规定「0 表示不限制」，所以这里也保留 0，
    // 各表单引用它作为默认值，避免在界面里写死一个数字（如 5）与后台设置不一致。
    defaultDeviceLimit: 0,
    defaultTheme: 'default',
    // 是否允许用户自行删除（踢下线）设备：由系统设置 → 用户与注册控制
    // 关闭后用户端设备管理不显示「移除」按钮，只能升级设备数量
    allowUserDeleteDevice: true,
    loading: false,
    error: null
  }),
  getters: {
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
        // 默认设备数：0 是合法值（表示不限制），解析失败时保留原值以免误覆盖
        const deviceLimitValue = settings.default_subscription_device_limit
        if (deviceLimitValue !== undefined && deviceLimitValue !== null) {
          const parsed = typeof deviceLimitValue === 'number' ? deviceLimitValue : parseInt(deviceLimitValue, 10)
          if (!Number.isNaN(parsed) && parsed >= 0) this.defaultDeviceLimit = parsed
        }
        this.defaultTheme = settings.default_theme || 'light'
        this.allowUserDeleteDevice = settings.allow_user_delete_device !== false
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
    getEmailError(email) {
      if (!email) return '请输入邮箱'
      if (!EMAIL_PATTERN.test(email)) return '邮箱格式不正确'
      return null
    },
  }
}) 
