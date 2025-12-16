import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/utils/api'

export const useThemeStore = defineStore('theme', () => {
  const getStoredTheme = () => {
    if (typeof window === 'undefined') return 'light'
    return localStorage.getItem('user-theme') || 'light'
  }
  const currentTheme = ref(getStoredTheme())
  const availableThemes = ref([
    { value: 'light', label: '浅色主题', icon: 'sunny', color: '#409EFF' },
    { value: 'dark', label: '深色主题', icon: 'moon', color: '#1a1a1a' },
    { value: 'blue', label: '蓝色主题', icon: 'water', color: '#1890ff' },
    { value: 'green', label: '绿色主题', icon: 'leaf', color: '#52c41a' },
    { value: 'purple', label: '紫色主题', icon: 'star', color: '#722ed1' },
    { value: 'orange', label: '橙色主题', icon: 'sunny', color: '#fa8c16' },
    { value: 'red', label: '红色主题', icon: 'fire', color: '#f5222d' },
    { value: 'cyan', label: '青色主题', icon: 'water', color: '#13c2c2' },
    { value: 'luck', label: 'Luck主题', icon: 'star', color: '#FFD700' },
    { value: 'aurora', label: 'Aurora主题', icon: 'star', color: '#7B68EE' },
    { value: 'auto', label: '跟随系统', icon: 'monitor', color: '#909399' }
  ])
  const loading = ref(false)

  const isDarkMode = computed(() => {
    if (typeof window === 'undefined') return false
    if (currentTheme.value === 'auto') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches
    }
    return currentTheme.value === 'dark'
  })

  const applyThemeLocally = (theme) => {
    currentTheme.value = theme
    if (typeof window !== 'undefined') {
      localStorage.setItem('user-theme', theme)
    }
    applyTheme(theme)
  }

  const setTheme = async (theme) => {
    try {
      loading.value = true
      applyThemeLocally(theme)
      try {
        const response = await api.put('/users/theme', { theme })
        if (response.data && response.data.success !== false) {
          return { success: true, message: '主题已保存' }
        } else {
          return {
            success: false,
            message: response.data?.message || '保存主题失败',
            localApplied: true
          }
        }
      } catch (apiError) {
        return {
          success: false,
          message: apiError.response?.data?.detail || apiError.message || '保存主题到云端失败',
          localApplied: true
        }
      }
    } catch (error) {
      try {
        applyThemeLocally(theme)
      } catch (e) {
      }
      return {
        success: false,
        message: error.response?.data?.detail || error.message || '设置主题失败',
        localApplied: true
      }
    } finally {
      loading.value = false
    }
  }

  const themeConfigs = {
    light: {
      primary: '#409EFF',
      success: '#67C23A',
      warning: '#E6A23C',
      danger: '#F56C6C',
      info: '#909399',
      bg: '#ffffff',
      bgPage: '#f2f3f5',
      text: '#303133',
      textSecondary: '#606266',
      border: '#dcdfe6',
      sidebarBg: '#f8f9fa',
      sidebarText: '#303133',
      sidebarHover: '#e9ecef',
      sidebarActive: '#409EFF'
    },
    dark: {
      primary: '#409EFF',
      success: '#67C23A',
      warning: '#E6A23C',
      danger: '#F56C6C',
      info: '#909399',
      bg: '#1a1a1a',
      bgPage: '#141414',
      text: '#E5EAF3',
      textSecondary: '#CFD3DC',
      border: '#4C4D4F',
      sidebarBg: '#1f1f1f',
      sidebarText: '#E5EAF3',
      sidebarHover: '#2a2a2a',
      sidebarActive: '#409EFF'
    },
    blue: {
      primary: '#1890ff',
      success: '#52c41a',
      warning: '#faad14',
      danger: '#ff4d4f',
      info: '#8c8c8c',
      bg: '#f0f2f5',
      bgPage: '#e6f7ff',
      text: '#262626',
      textSecondary: '#595959',
      border: '#d9d9d9',
      sidebarBg: '#e6f7ff',
      sidebarText: '#262626',
      sidebarHover: '#bae7ff',
      sidebarActive: '#1890ff'
    },
    green: {
      primary: '#52c41a',
      success: '#52c41a',
      warning: '#faad14',
      danger: '#ff4d4f',
      info: '#8c8c8c',
      bg: '#f6ffed',
      bgPage: '#f0f9ff',
      text: '#262626',
      textSecondary: '#595959',
      border: '#b7eb8f',
      sidebarBg: '#f6ffed',
      sidebarText: '#262626',
      sidebarHover: '#d9f7be',
      sidebarActive: '#52c41a'
    },
    purple: {
      primary: '#722ed1',
      success: '#52c41a',
      warning: '#faad14',
      danger: '#ff4d4f',
      info: '#8c8c8c',
      bg: '#f9f0ff',
      bgPage: '#f0f0ff',
      text: '#262626',
      textSecondary: '#595959',
      border: '#d3adf7',
      sidebarBg: '#f9f0ff',
      sidebarText: '#262626',
      sidebarHover: '#efdbff',
      sidebarActive: '#722ed1'
    },
    orange: {
      primary: '#fa8c16',
      success: '#52c41a',
      warning: '#faad14',
      danger: '#ff4d4f',
      info: '#8c8c8c',
      bg: '#fff7e6',
      bgPage: '#fffbe6',
      text: '#262626',
      textSecondary: '#595959',
      border: '#ffd591',
      sidebarBg: '#fff7e6',
      sidebarText: '#262626',
      sidebarHover: '#ffe7ba',
      sidebarActive: '#fa8c16'
    },
    red: {
      primary: '#f5222d',
      success: '#52c41a',
      warning: '#faad14',
      danger: '#ff4d4f',
      info: '#8c8c8c',
      bg: '#fff1f0',
      bgPage: '#fff0f0',
      text: '#262626',
      textSecondary: '#595959',
      border: '#ffccc7',
      sidebarBg: '#fff1f0',
      sidebarText: '#262626',
      sidebarHover: '#ffd4d0',
      sidebarActive: '#f5222d'
    },
    cyan: {
      primary: '#13c2c2',
      success: '#52c41a',
      warning: '#faad14',
      danger: '#ff4d4f',
      info: '#8c8c8c',
      bg: '#e6fffb',
      bgPage: '#e0f7ff',
      text: '#262626',
      textSecondary: '#595959',
      border: '#87e8de',
      sidebarBg: '#e6fffb',
      sidebarText: '#262626',
      sidebarHover: '#b5f5ec',
      sidebarActive: '#13c2c2'
    },
    luck: {
      primary: '#FFD700',
      success: '#32CD32',
      warning: '#FFA500',
      danger: '#FF6347',
      info: '#9370DB',
      bg: '#FFFEF0',
      bgPage: '#FFFACD',
      text: '#2C2416',
      textSecondary: '#5C4A3A',
      border: '#FFD700',
      sidebarBg: '#FFFEF0',
      sidebarText: '#2C2416',
      sidebarHover: '#FFF8DC',
      sidebarActive: '#FFD700'
    },
    aurora: {
      primary: '#7B68EE',
      success: '#00CED1',
      warning: '#FF69B4',
      danger: '#FF1493',
      info: '#9370DB',
      bg: '#0F0C1D',
      bgPage: '#1A1625',
      text: '#E6E6FA',
      textSecondary: '#D8BFD8',
      border: '#4B0082',
      sidebarBg: '#1A1625',
      sidebarText: '#E6E6FA',
      sidebarHover: '#2A1F3D',
      sidebarActive: '#7B68EE'
    }
  }

  const applyTheme = (theme) => {
    if (typeof window === 'undefined' || typeof document === 'undefined') return
    const root = document.documentElement
    let actualTheme = theme
    if (theme === 'auto') {
      actualTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
    }
    const config = themeConfigs[actualTheme] || themeConfigs.light
    if (!config) return
    const themeClasses = Object.keys(themeConfigs).map(t => `theme-${t}`)
    themeClasses.push('theme-auto', 'theme-default', 'theme-light')
    root.classList.remove(...themeClasses)
    root.classList.add(`theme-${theme}`)
    if (theme === 'auto') {
      root.classList.add(`theme-${actualTheme}`)
    }
    const elColorVars = {
      '--el-color-primary': config.primary,
      '--el-color-success': config.success,
      '--el-color-warning': config.warning,
      '--el-color-danger': config.danger,
      '--el-color-info': config.info,
      '--el-bg-color': config.bg,
      '--el-bg-color-page': config.bgPage,
      '--el-text-color-primary': config.text,
      '--el-text-color-regular': config.textSecondary,
      '--el-text-color-secondary': config.textSecondary,
      '--el-border-color': config.border,
      '--el-border-color-light': config.border,
      '--el-border-color-lighter': config.border,
      '--el-border-color-extra-light': config.border,
      '--el-fill-color': config.bgPage,
      '--el-fill-color-light': config.bgPage,
      '--el-fill-color-lighter': config.bgPage,
      '--el-fill-color-extra-light': config.bgPage,
      '--el-fill-color-blank': config.bg
    }
    const customVars = {
      '--primary-color': config.primary,
      '--success-color': config.success,
      '--warning-color': config.warning,
      '--danger-color': config.danger,
      '--info-color': config.info,
      '--background-color': config.bg,
      '--text-color': config.text,
      '--text-color-secondary': config.textSecondary,
      '--sidebar-bg-color': config.sidebarBg,
      '--sidebar-text-color': config.sidebarText,
      '--sidebar-hover-bg': config.sidebarHover,
      '--sidebar-active-bg': config.sidebarActive
    }
    Object.entries(elColorVars).forEach(([key, value]) => {
      root.style.setProperty(key, value)
    })
    Object.entries(customVars).forEach(([key, value]) => {
      root.style.setProperty(key, value)
    })
    document.body.style.backgroundColor = config.bg
    document.body.style.color = config.text
  }

  const loadUserTheme = async () => {
    try {
      const response = await api.get('/users/theme')
      if (response.data && response.data.theme) {
        currentTheme.value = response.data.theme
        if (typeof window !== 'undefined') {
          localStorage.setItem('user-theme', response.data.theme)
        }
        applyTheme(response.data.theme)
      }
    } catch (error) {
      applyTheme(currentTheme.value)
    }
  }

  const initTheme = () => {
    if (typeof window === 'undefined') return
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    mediaQuery.addEventListener('change', () => {
      if (currentTheme.value === 'auto') {
        applyTheme('auto')
      }
    })
    applyTheme(currentTheme.value)
  }

  return {
    currentTheme,
    availableThemes,
    loading,
    isDarkMode,
    setTheme,
    applyTheme,
    loadUserTheme,
    initTheme
  }
})
