import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api, resetRefreshFailed } from '@/utils/api'
import { secureStorage } from '@/utils/api'
import { useThemeStore } from '@/store/theme'
export const useAuthStore = defineStore('auth', () => {
  const isAdminPath = () => typeof window !== 'undefined' && window.location.pathname.startsWith('/admin')
  const TOKEN_TTL = 60 * 60 * 1000
  const REFRESH_TOKEN_TTL = 30 * 24 * 60 * 60 * 1000
  const SECURE_STORAGE_KEY = 'cboard_secure_'
  const getRememberKey = (isAdmin = false) => (isAdmin ? 'admin_remember' : 'user_remember')
  const hasValidLocalValue = (key) => {
    if (typeof window === 'undefined') return false
    try {
      const item = localStorage.getItem(`${SECURE_STORAGE_KEY}${key}`)
      if (!item) return false
      const data = JSON.parse(item)
      if (data.expiry && Date.now() > data.expiry) {
        localStorage.removeItem(`${SECURE_STORAGE_KEY}${key}`)
        return false
      }
      return true
    } catch (error) {
      return false
    }
  }
  const getRememberPreference = (isAdmin = false) => {
    const remember = secureStorage.get(getRememberKey(isAdmin))
    if (remember === true || remember === 'true') return true
    if (remember === false || remember === 'false') return false
    return hasValidLocalValue(isAdmin ? 'admin_token' : 'user_token')
  }
  const saveRememberPreference = (remember, isAdmin = false) => {
    // 偏好始终持久化，保证刷新时可读取正确的存储策略
    secureStorage.set(getRememberKey(isAdmin), !!remember, false, REFRESH_TOKEN_TTL)
  }
  const getInitialToken = () => {
    if (typeof window === 'undefined') return ''
    const path = window.location.pathname
    if (path.startsWith('/admin')) {
      return secureStorage.get('admin_token') || ''
    }
    return secureStorage.get('user_token') || ''
  }
  const getInitialUser = () => {
    try {
      if (typeof window === 'undefined') return null
      const path = window.location.pathname
      if (path.startsWith('/admin')) {
        const adminUser = secureStorage.get('admin_user')
        if (adminUser) {
          return typeof adminUser === 'string' ? JSON.parse(adminUser) : adminUser
        }
        return null
      }
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
  const saveToken = (accessToken, isAdmin = false, remember = getRememberPreference(isAdmin)) => {
    if (isAdmin) {
      // 管理员token始终存localStorage，确保新标签页和刷新后可恢复
      secureStorage.set('admin_token', accessToken, false, TOKEN_TTL)
      return
    }
    const useSession = !remember
    secureStorage.set('user_token', accessToken, useSession, TOKEN_TTL)
  }
  const saveUser = (userData, isAdmin = false, remember = getRememberPreference(isAdmin)) => {
    if (isAdmin) {
      // 管理员user始终存localStorage，TTL延长到30天（依赖refresh cookie续期）
      secureStorage.set('admin_user', userData, false, REFRESH_TOKEN_TTL)
      return
    }
    const useSession = !remember
    secureStorage.set('user_data', userData, useSession, REFRESH_TOKEN_TTL)
  }
  const saveRefreshToken = (refreshToken, isAdmin = false, remember = getRememberPreference(isAdmin)) => {
    // Refresh Token 由后端通过 HttpOnly Cookie 管理，不再存储到前端
    void refreshToken
    void remember
    void isAdmin
  }
  const token = ref(getInitialToken())
  const user = ref(getInitialUser())
  const loading = ref(false)
  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.is_admin || false)
  const handleApiError = (error, defaultMessage) => {
    let message = defaultMessage
    if (error.response?.data) {
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
  const login = async (credentials) => {
    loading.value = true
    try {
      const response = await api.post('/auth/login-json', {
        username: credentials.username,
        password: credentials.password
      })
      const responseData = response.data?.data || response.data
      const { access_token, refresh_token, user: userData } = responseData
      if (!userData) {
        return {
          success: false,
          message: '登录响应格式错误'
        }
      }
      if (userData.is_admin) {
        return {
          success: false,
          message: '管理员账户请使用管理员登录页面登录'
        }
      }
      token.value = access_token
      user.value = userData
      secureStorage.remove('logout_marker')
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
      const remember = !!credentials.remember
      saveRememberPreference(remember, false)
      saveToken(access_token, false, remember)
      saveUser(safeUserData, false, remember)
      saveRefreshToken(refresh_token, false, remember)
      resetRefreshFailed()
      setTimeout(() => {
        const themeStore = useThemeStore()
        themeStore.loadUserTheme().catch(() => {})
      }, 500)
      return { success: true }
    } catch (error) {
      return handleApiError(error, '登录失败')
    } finally {
      loading.value = false
    }
  }
  const adminLogin = async (credentials) => {
    loading.value = true
    try {
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
      const responseData = response.data?.data || response.data
      const { access_token, refresh_token, user: userData } = responseData
      if (!userData) {
        return {
          success: false,
          message: '登录响应格式错误'
        }
      }
      if (!userData.is_admin) {
        return {
          success: false,
          message: '该账户不是管理员，请使用用户登录页面'
        }
      }
      token.value = access_token
      user.value = userData
      secureStorage.remove('logout_marker')
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
      const remember = !!credentials.remember
      saveRememberPreference(remember, true)
      saveToken(access_token, true, remember)
      saveUser(safeUserData, true, remember)
      saveRefreshToken(refresh_token, true, remember)
      resetRefreshFailed()
      setTimeout(() => {
        const themeStore = useThemeStore()
        themeStore.loadUserTheme().catch(() => {})
      }, 500)
      return { success: true }
    } catch (error) {
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
    const currentToken = token.value
    secureStorage.set('logout_marker', true, false, 24 * 60 * 60 * 1000)
    if (typeof window !== 'undefined' && currentToken) {
      fetch('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${currentToken}`
        }
      }).catch(() => {})
    }

    token.value = ''
    user.value = null
    secureStorage.remove('admin_token')
    secureStorage.remove('admin_user')
    secureStorage.remove('admin_refresh_token')
    secureStorage.remove('user_token')
    secureStorage.remove('user_data')
    secureStorage.remove('user_refresh_token')
    secureStorage.remove('token')
    secureStorage.remove('refresh_token')
    secureStorage.remove('user')
    secureStorage.remove('admin_remember')
    secureStorage.remove('user_remember')
    secureStorage.clear()
    resetRefreshFailed()
  }
  const clearAuthCache = () => {
    logout()
    if (typeof window !== 'undefined') {
      window.location.reload()
    }
  }
  const refreshToken = async () => {
    const isAdmin = isAdminPath()
    try {
      const response = await api.post('/auth/refresh', {}, {
        withCredentials: true,
        headers: { 'X-Auth-Role': isAdmin ? 'admin' : 'user' }
      })
      const responseData = response.data?.data || response.data
      const { access_token, refresh_token } = responseData
      token.value = access_token
      saveToken(access_token, isAdmin)
      saveRefreshToken(refresh_token, isAdmin)
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
    const remember = !useSessionStorage
    saveRememberPreference(remember, isAdmin)
    saveToken(newToken, isAdmin, remember)
    saveUser(newUser, isAdmin, remember)
    secureStorage.remove('logout_marker')
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
