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
  const normalizeRole = (role) => {
    if (role === 'admin' || role === true) return 'admin'
    if (role === 'user' || role === false) return 'user'
    if (role === 'all') return 'all'
    return isAdminPath() ? 'admin' : 'user'
  }
  const roleIsAdmin = (role) => normalizeRole(role) === 'admin'
  const getRememberKey = (isAdmin = false) => (isAdmin ? 'admin_remember' : 'user_remember')
  const getRoleKeys = (role) => {
    const adminRole = roleIsAdmin(role)
    return {
      token: adminRole ? 'admin_token' : 'user_token',
      user: adminRole ? 'admin_user' : 'user_data',
      refresh: adminRole ? 'admin_refresh_token' : 'user_refresh_token',
      remember: adminRole ? 'admin_remember' : 'user_remember',
      marker: adminRole ? 'logout_marker_admin' : 'logout_marker_user',
      isAdmin: adminRole
    }
  }
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
    if (isAdmin) return true
    const remember = secureStorage.get(getRememberKey(isAdmin))
    if (remember === true || remember === 'true') return true
    if (remember === false || remember === 'false') return false
    return hasValidLocalValue(isAdmin ? 'admin_token' : 'user_token')
  }
  const saveRememberPreference = (remember, isAdmin = false) => {
    // 偏好始终持久化，保证刷新时可读取正确的存储策略
    secureStorage.set(getRememberKey(isAdmin), isAdmin ? true : !!remember, false, REFRESH_TOKEN_TTL)
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
    const useSession = isAdmin ? true : !remember
    secureStorage.set(isAdmin ? 'admin_token' : 'user_token', accessToken, useSession, TOKEN_TTL)
  }
  const saveUser = (userData, isAdmin = false, remember = getRememberPreference(isAdmin)) => {
    const useSession = isAdmin ? true : !remember
    secureStorage.set(isAdmin ? 'admin_user' : 'user_data', userData, useSession, REFRESH_TOKEN_TTL)
  }
  const saveRefreshToken = (refreshToken, isAdmin = false, remember = getRememberPreference(isAdmin)) => {
    if (!refreshToken) return
    const key = isAdmin ? 'admin_refresh_token' : 'user_refresh_token'
    const useSession = isAdmin ? true : !remember
    secureStorage.set(key, refreshToken, useSession, REFRESH_TOKEN_TTL)
  }
  const token = ref(getInitialToken())
  const user = ref(getInitialUser())
  const loading = ref(false)
  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.is_admin || false)
  const activeRole = () => user.value?.is_admin ? 'admin' : 'user'
  const isRoleActive = (role) => token.value && normalizeRole(role) === activeRole()
  const clearRoleStorage = (role) => {
    const keys = getRoleKeys(role)
    secureStorage.remove(keys.token)
    secureStorage.remove(keys.user)
    secureStorage.remove(keys.refresh)
    secureStorage.remove(keys.remember)
  }
  const clearLegacyAuthStorage = () => {
    secureStorage.remove('token')
    secureStorage.remove('refresh_token')
    secureStorage.remove('user')
    secureStorage.remove('logout_marker')
  }
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
      const isAdminUser = !!userData.is_admin
      if (credentials.requireAdmin && !isAdminUser) {
        return {
          success: false,
          message: '该账号不是管理员账号'
        }
      }
      clearRoleStorage(isAdminUser ? 'admin' : 'user')
      token.value = access_token
      user.value = userData
      secureStorage.remove(isAdminUser ? 'logout_marker_admin' : 'logout_marker_user')
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
      const remember = isAdminUser ? true : !!credentials.remember
      saveRememberPreference(remember, isAdminUser)
      saveToken(access_token, isAdminUser, remember)
      saveUser(safeUserData, isAdminUser, remember)
      saveRefreshToken(refresh_token, isAdminUser, remember)
      resetRefreshFailed()
      setTimeout(() => {
        const themeStore = useThemeStore()
        themeStore.loadUserTheme().catch(() => {})
      }, 500)
      return { success: true, isAdmin: isAdminUser }
    } catch (error) {
      return handleApiError(error, '登录失败')
    } finally {
      loading.value = false
    }
  }
  const adminLogin = async (credentials) => login(credentials)
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
  const logout = (role = null) => {
    const targetRole = normalizeRole(role)
    if (targetRole === 'all') {
      const roles = ['admin', 'user']
      roles.forEach(item => {
        const keys = getRoleKeys(item)
        const roleToken = secureStorage.get(keys.token)
        const storedRefresh = secureStorage.get(keys.refresh)
        if (typeof window !== 'undefined' && roleToken) {
          fetch('/api/v1/auth/logout', {
            method: 'POST',
            credentials: 'include',
            headers: {
              'Content-Type': 'application/json',
              Authorization: `Bearer ${roleToken}`
            },
            body: JSON.stringify({ refresh_token: storedRefresh || '' })
          }).catch(() => {})
        }
        clearRoleStorage(item)
        secureStorage.set(keys.marker, true, false, 24 * 60 * 60 * 1000)
      })
      token.value = ''
      user.value = null
      clearLegacyAuthStorage()
      resetRefreshFailed()
      return
    }

    const keys = getRoleKeys(targetRole)
    const currentToken = isRoleActive(targetRole) ? token.value : secureStorage.get(keys.token)
    const storedRefresh = secureStorage.get(keys.refresh)
    if (typeof window !== 'undefined' && currentToken) {
      fetch('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${currentToken}`
        },
        body: JSON.stringify({ refresh_token: storedRefresh || '' })
      }).catch(() => {})
    }

    if (isRoleActive(targetRole)) {
      token.value = ''
      user.value = null
    }
    clearRoleStorage(targetRole)
    clearLegacyAuthStorage()
    secureStorage.set(keys.marker, true, false, 24 * 60 * 60 * 1000)
    resetRefreshFailed()
  }
  const clearAuthCache = () => {
    logout('all')
    if (typeof window !== 'undefined') {
      window.location.reload()
    }
  }
  const refreshToken = async () => {
    const role = isAdminPath() ? 'admin' : 'user'
    const keys = getRoleKeys(role)
    const storedRefresh = secureStorage.get(keys.refresh)
    if (!storedRefresh) {
      logout(role)
      return false
    }
    try {
      const response = await api.post('/auth/refresh', { refresh_token: storedRefresh })
      const responseData = response.data?.data || response.data
      const { access_token, refresh_token: newRefresh } = responseData
      token.value = access_token
      saveToken(access_token, keys.isAdmin)
      saveRefreshToken(newRefresh, keys.isAdmin)
      return true
    } catch (error) {
      logout(role)
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
    secureStorage.remove(isAdmin ? 'logout_marker_admin' : 'logout_marker_user')
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
