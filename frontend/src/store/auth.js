import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api, resetRefreshFailed } from '@/utils/api'
import { secureStorage } from '@/utils/secureStorage'
import { useThemeStore } from '@/store/theme'

export const useAuthStore = defineStore('auth', () => {
  const isAdminPath = () => typeof window !== 'undefined' && window.location.pathname.startsWith('/admin')
  const TOKEN_TTL = 24 * 60 * 60 * 1000
  const REFRESH_TOKEN_TTL = 7 * 24 * 60 * 60 * 1000

  const getInitialToken = () => {
    if (typeof window === 'undefined') return ''
    const path = window.location.pathname
    if (path.startsWith('/admin')) {
      // 管理员路径：只获取管理员 token
      return secureStorage.get('admin_token') || ''
    }
    // 用户路径：只获取用户 token
    return secureStorage.get('user_token') || ''
  }

  const getInitialUser = () => {
    try {
      if (typeof window === 'undefined') return null
      const path = window.location.pathname
      if (path.startsWith('/admin')) {
        // 管理员路径：只获取管理员用户信息
        const adminUser = secureStorage.get('admin_user')
        if (adminUser) {
          return typeof adminUser === 'string' ? JSON.parse(adminUser) : adminUser
        }
        return null
      }
      // 用户路径：只获取用户信息
      const userData = secureStorage.get('user_data')
      if (userData) {
        return typeof userData === 'string' ? JSON.parse(userData) : userData
      }
      return null
    } catch (error) {
      secureStorage.remove('user_data')
      secureStorage.remove('admin_user')
      return null
    }
  }

  const saveToken = (accessToken, isAdmin = false) => {
    if (isAdmin) {
      secureStorage.set('admin_token', accessToken, false, TOKEN_TTL)
    } else {
      secureStorage.set('user_token', accessToken, true, TOKEN_TTL)
    }
  }

  const saveUser = (userData, isAdmin = false) => {
    if (isAdmin) {
      secureStorage.set('admin_user', userData, false, TOKEN_TTL)
    } else {
      secureStorage.set('user_data', userData, true, TOKEN_TTL)
    }
  }

  const token = ref(getInitialToken())
  const user = ref(getInitialUser())
  const loading = ref(false)
  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.is_admin || false)

  const handleApiError = (error, defaultMessage) => {
    // 优先获取后端返回的详细错误信息
    let message = defaultMessage
    if (error.response?.data) {
      // 尝试多种可能的错误信息字段
      message = error.response.data.detail || 
                error.response.data.message || 
                error.response.data.error || 
                defaultMessage
    } else if (error.message) {
      message = error.message
    }
    console.error('登录错误:', error.response?.data || error.message)
    return {
      success: false,
      message
    }
  }

  // 普通用户登录（只允许非管理员）
  const login = async (credentials) => {
    loading.value = true
    try {
      const response = await api.post('/auth/login-json', {
        username: credentials.username,
        password: credentials.password
      })
      // 后端返回的数据结构: { success: true, data: { access_token, refresh_token, user } }
      const responseData = response.data?.data || response.data
      const { access_token, refresh_token, user: userData } = responseData
      
      // 检查 userData 是否存在
      if (!userData) {
        return {
          success: false,
          message: '登录响应格式错误'
        }
      }
      
      // 如果是管理员账户，不允许在用户登录页面登录
      if (userData.is_admin) {
        return {
          success: false,
          message: '管理员账户请使用管理员登录页面登录'
        }
      }
      
      token.value = access_token
      user.value = userData
      
      // 用户登录使用 sessionStorage
      // 只存储必要的用户信息，避免存储敏感数据
      const safeUserData = {
        id: userData.id,
        username: userData.username,
        email: userData.email,
        is_admin: userData.is_admin,
        is_verified: userData.is_verified,
        is_active: userData.is_active,
        theme: userData.theme,
        language: userData.language
      }
      secureStorage.set('user_token', access_token, true, TOKEN_TTL)
      secureStorage.set('user_data', safeUserData, true, TOKEN_TTL)
      secureStorage.set('user_refresh_token', refresh_token, true, REFRESH_TOKEN_TTL)
      
      resetRefreshFailed()
      // 延迟加载主题，避免阻塞登录流程
      setTimeout(() => {
        const themeStore = useThemeStore()
        themeStore.loadUserTheme().catch(() => {})
      }, 500)
      return { success: true }
    } catch (error) {
      // 登录失败时，直接返回错误信息
      // axios拦截器已经处理了登录API失败的情况，不会尝试刷新token
      return handleApiError(error, '登录失败')
    } finally {
      loading.value = false
    }
  }

  // 管理员登录（只允许管理员）
  const adminLogin = async (credentials) => {
    loading.value = true
    try {
      // 清除旧的认证信息，避免缓存问题
      secureStorage.remove('admin_token')
      secureStorage.remove('admin_user')
      secureStorage.remove('admin_refresh_token')
      secureStorage.remove('user_token')
      secureStorage.remove('user_data')
      secureStorage.remove('user_refresh_token')
      
      const response = await api.post('/auth/login-json', {
        username: credentials.username,
        password: credentials.password
      })
      // 后端返回的数据结构: { success: true, data: { access_token, refresh_token, user } }
      const responseData = response.data?.data || response.data
      const { access_token, refresh_token, user: userData } = responseData
      
      // 检查 userData 是否存在
      if (!userData) {
        return {
          success: false,
          message: '登录响应格式错误'
        }
      }
      
      // 验证是否为管理员
      if (!userData.is_admin) {
        return {
          success: false,
          message: '该账户不是管理员，请使用用户登录页面'
        }
      }
      
      token.value = access_token
      user.value = userData
      
      // 管理员登录使用 localStorage
      // 只存储必要的用户信息，避免存储敏感数据
      const safeUserData = {
        id: userData.id,
        username: userData.username,
        email: userData.email,
        is_admin: userData.is_admin,
        is_verified: userData.is_verified,
        is_active: userData.is_active,
        theme: userData.theme,
        language: userData.language
      }
      secureStorage.set('admin_token', access_token, false, TOKEN_TTL)
      secureStorage.set('admin_user', safeUserData, false, TOKEN_TTL)
      secureStorage.set('admin_refresh_token', refresh_token, false, REFRESH_TOKEN_TTL)
      
      resetRefreshFailed()
      // 延迟加载主题，避免阻塞登录流程
      setTimeout(() => {
        const themeStore = useThemeStore()
        themeStore.loadUserTheme().catch(() => {})
      }, 500)
      return { success: true }
    } catch (error) {
      // 登录失败时清除可能已保存的认证信息
      secureStorage.remove('admin_token')
      secureStorage.remove('admin_user')
      secureStorage.remove('admin_refresh_token')
      return handleApiError(error, '登录失败')
    } finally {
      loading.value = false
    }
  }

  const register = async (userData) => {
    loading.value = true
    try {
      const response = await api.post('/auth/register', userData)
      return { success: true, message: '注册成功', data: response.data }
    } catch (error) {
      return handleApiError(error, '注册失败')
    } finally {
      loading.value = false
    }
  }

  const logout = () => {
    token.value = ''
    user.value = null
    
    // 清除所有角色的token和用户数据（彻底清除，避免冲突）
    secureStorage.remove('admin_token')
    secureStorage.remove('admin_user')
    secureStorage.remove('admin_refresh_token')
    secureStorage.remove('user_token')
    secureStorage.remove('user_data')
    secureStorage.remove('user_refresh_token')
    
    secureStorage.remove('token')
    secureStorage.remove('refresh_token')
    secureStorage.remove('user')
    
    // 清除所有认证相关的缓存
    secureStorage.clear()
    
    // 重置刷新失败标志
    resetRefreshFailed()
  }
  
  // 清除所有认证缓存（用于解决缓存问题）
  const clearAuthCache = () => {
    logout()
    // 强制刷新页面以确保清除所有状态
    if (typeof window !== 'undefined') {
      window.location.reload()
    }
  }

  const refreshToken = async () => {
    const isAdmin = isAdminPath()
    const refreshTokenKey = isAdmin ? 'admin_refresh_token' : 'user_refresh_token'
    const refresh_token = secureStorage.get(refreshTokenKey)
    
    if (!refresh_token) {
      logout()
      return false
    }
    try {
      const response = await api.post('/auth/refresh', { refresh_token }, { withCredentials: true })
      const { access_token } = response.data
      token.value = access_token
      saveToken(access_token, isAdmin)
      return true
    } catch (error) {
      logout()
      return false
    }
  }

  const forgotPassword = async (email) => {
    loading.value = true
    try {
      await api.post('/auth/forgot-password', { email })
      return { success: true, message: '重置链接已发送到您的邮箱，请查收' }
    } catch (error) {
      return handleApiError(error, '发送失败')
    } finally {
      loading.value = false
    }
  }

  const updateUser = (userData) => {
    user.value = { ...user.value, ...userData }
    const isAdmin = user.value?.is_admin || false
    saveUser(user.value, isAdmin)
  }

  const changePassword = async (oldPassword, newPassword) => {
    loading.value = true
    try {
      await api.post('/users/change-password', {
        current_password: oldPassword,
        new_password: newPassword
      })
      return { success: true, message: '密码修改成功' }
    } catch (error) {
      return handleApiError(error, '密码修改失败')
    } finally {
      loading.value = false
    }
  }

  const setAuth = (newToken, newUser, useSessionStorage = false) => {
    token.value = newToken
    user.value = newUser
    const isAdmin = newUser?.is_admin || false
    if (useSessionStorage) {
      // 用户登录：使用 sessionStorage
      secureStorage.set('user_token', newToken, true, TOKEN_TTL)
      secureStorage.set('user_data', newUser, true, TOKEN_TTL)
    } else {
      // 管理员登录：使用 localStorage
      secureStorage.set('admin_token', newToken, false, TOKEN_TTL)
      secureStorage.set('admin_user', newUser, false, TOKEN_TTL)
    }
  }

  const setToken = (newToken) => {
    token.value = newToken
    const isAdmin = isAdminPath()
    saveToken(newToken, isAdmin)
  }

  const setUser = (newUser) => {
    user.value = newUser
    const isAdmin = newUser?.is_admin || false
    saveUser(newUser, isAdmin)
  }

  const getCurrentState = () => ({
    token: token.value,
    user: user.value,
    isAuthenticated: isAuthenticated.value,
    isAdmin: isAdmin.value,
    storedUser: secureStorage.get('user') || secureStorage.get('user_data') || secureStorage.get('admin_user'),
    storedToken: secureStorage.get('token') || secureStorage.get('user_token') || secureStorage.get('admin_token')
  })

  return {
    token,
    user,
    loading,
    isAuthenticated,
    isAdmin,
    login,
    adminLogin,
    register,
    logout,
    refreshToken,
    forgotPassword,
    updateUser,
    changePassword,
    setAuth,
    setToken,
    setUser,
    getCurrentState,
    clearAuthCache
  }
}) 