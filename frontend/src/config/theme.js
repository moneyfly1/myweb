export const themeConfig = {
  light: {
    name: 'light',
    primary: '#409EFF',
    success: '#67C23A',
    warning: '#E6A23C',
    danger: '#F56C6C',
    info: '#909399',
    background: '#f5f7fa',
    text: '#303133',
    border: '#DCDFE6'
  },
  dark: {
    name: 'dark',
    primary: '#409EFF',
    success: '#67C23A',
    warning: '#E6A23C',
    danger: '#F56C6C',
    info: '#909399',
    background: '#1a1a1a',
    text: '#ffffff',
    border: '#4c4c4c'
  },
  blue: {
    name: 'blue',
    primary: '#1890ff',
    success: '#52c41a',
    warning: '#faad14',
    danger: '#ff4d4f',
    info: '#8c8c8c',
    background: '#f0f2f5',
    text: '#262626',
    border: '#d9d9d9'
  },
  green: {
    name: 'green',
    primary: '#52c41a',
    success: '#52c41a',
    warning: '#faad14',
    danger: '#ff4d4f',
    info: '#8c8c8c',
    background: '#f6ffed',
    text: '#262626',
    border: '#b7eb8f'
  },
  purple: {
    name: 'purple',
    primary: '#722ed1',
    success: '#52c41a',
    warning: '#faad14',
    danger: '#ff4d4f',
    info: '#8c8c8c',
    background: '#f9f0ff',
    text: '#262626',
    border: '#d3adf7'
  },
  orange: {
    name: 'orange',
    primary: '#fa8c16',
    success: '#52c41a',
    warning: '#faad14',
    danger: '#ff4d4f',
    info: '#8c8c8c',
    background: '#fff7e6',
    text: '#262626',
    border: '#ffd591'
  },
  red: {
    name: 'red',
    primary: '#f5222d',
    success: '#52c41a',
    warning: '#faad14',
    danger: '#ff4d4f',
    info: '#8c8c8c',
    background: '#fff1f0',
    text: '#262626',
    border: '#ffccc7'
  },
  cyan: {
    name: 'cyan',
    primary: '#13c2c2',
    success: '#52c41a',
    warning: '#faad14',
    danger: '#ff4d4f',
    info: '#8c8c8c',
    background: '#e6fffb',
    text: '#262626',
    border: '#87e8de'
  },
  luck: {
    name: 'luck',
    primary: '#FFD700',
    success: '#32CD32',
    warning: '#FFA500',
    danger: '#FF6347',
    info: '#9370DB',
    background: '#FFFEF0',
    text: '#2C2416',
    border: '#FFD700'
  },
  aurora: {
    name: 'aurora',
    primary: '#7B68EE',
    success: '#00CED1',
    warning: '#FF69B4',
    danger: '#FF1493',
    info: '#9370DB',
    background: '#0F0C1D',
    text: '#E6E6FA',
    border: '#4B0082'
  },
  default: {
    name: 'default',
    primary: '#409EFF',
    success: '#67C23A',
    warning: '#E6A23C',
    danger: '#F56C6C',
    info: '#909399',
    background: '#f5f7fa',
    text: '#303133',
    border: '#DCDFE6'
  }
}
export class ThemeManager {
  constructor() {
    if (typeof window !== 'undefined') {
      this.currentTheme = localStorage.getItem('theme') || 'default'
      this.applyTheme(this.currentTheme)
    } else {
      this.currentTheme = 'default'
    }
  }
  getCurrentTheme() {
    return this.currentTheme
  }
  getThemeConfig(themeName = null) {
    const theme = themeName || this.currentTheme
    return themeConfig[theme] || themeConfig.default
  }
  applyTheme(themeName) {
    if (typeof window === 'undefined' || typeof document === 'undefined') return
    const config = this.getThemeConfig(themeName)
    if (!config) return
    this.currentTheme = themeName
    if (typeof window !== 'undefined') {
      localStorage.setItem('theme', themeName)
    }
    const root = document.documentElement
    Object.keys(config).forEach(key => {
      if (key !== 'name') {
        root.style.setProperty(`--el-color-${key}`, config[key])
        root.style.setProperty(`--theme-${key}`, config[key])
      }
    })
    root.className = root.className.replace(/theme-\w+/g, '')
    root.classList.add(`theme-${themeName}`)
  }
  toggleTheme() {
    const themes = Object.keys(themeConfig)
    const currentIndex = themes.indexOf(this.currentTheme)
    const nextIndex = (currentIndex + 1) % themes.length
    this.applyTheme(themes[nextIndex])
  }
  getAllThemes() {
    return Object.keys(themeConfig).map(key => ({
      name: key,
      label: this.getThemeLabel(key),
      config: themeConfig[key]
    }))
  }
  getThemeLabel(themeName) {
    const labels = {
      light: '浅色主题',
      dark: '深色主题',
      blue: '蓝色主题',
      green: '绿色主题',
      purple: '紫色主题',
      orange: '橙色主题',
      red: '红色主题',
      cyan: '青色主题',
      luck: 'Luck主题',
      aurora: 'Aurora主题',
      auto: '跟随系统',
      default: '默认主题'
    }
    return labels[themeName] || themeName
  }
}
export const themeManager = new ThemeManager() 