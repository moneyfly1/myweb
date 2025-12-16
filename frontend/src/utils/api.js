import axios from 'axios'
import { secureStorage } from '@/utils/secureStorage'

let _router = null
let _useAuthStore = null

export const initApi = (router, useAuthStore) => {
  _router = router
  _useAuthStore = useAuthStore
}

export const api = axios.create({
  baseURL: '/api/v1',
  timeout: 60000,  // 增加到60秒，避免长时间操作超时
  headers: {
    'Content-Type': 'application/json'
  },
  withCredentials: true
})

// 导入页面可见性工具
import { onPageVisible, isVisible } from '@/utils/pageVisibility'

// 页面可见性检测和自动重连
let reconnectTimer = null

// 注册页面可见性处理器
onPageVisible(() => {
  // 页面重新可见时，清除之前的重连定时器
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  // 测试连接是否正常
  testConnection()
})

// 测试连接是否正常
async function testConnection() {
  try {
    // 使用一个轻量级的健康检查端点，增加超时时间
    await axios.get('/api/v1/settings/public-settings', {
      timeout: 10000,  // 增加到 10 秒
      withCredentials: true
    })
  } catch (error) {
    // 连接失败，但不影响正常使用，静默处理
    if (error.code !== 'ECONNABORTED') {
      console.warn('连接测试失败，可能需要刷新页面:', error.message)
    }
  }
}

export const useApi = () => api

const handleLogout = () => {
  if (_useAuthStore) {
    const authStore = _useAuthStore()
    authStore.logout()
  }
  
  if (_router) {
    const currentPath = _router.currentRoute.value.path
    // 根据当前路径跳转到对应的登录页面
    if (currentPath.startsWith('/admin')) {
      if (currentPath !== '/admin/login') {
        _router.push('/admin/login')
      }
    } else {
      if (currentPath !== '/login' && currentPath !== '/forgot-password') {
        _router.push('/login')
      }
    }
  }
}

api.interceptors.request.use(
  config => {
    // 由于 baseURL 是 '/api/v1'，config.url 不包含 baseURL 前缀
    // 获取当前路径，用于判断是否在管理员后台
    const currentPath = typeof window !== 'undefined' ? window.location.pathname : ''
    const isInAdminPanel = currentPath.startsWith('/admin')
    
    // 定义公开API列表（不需要认证的API）
    const publicAPIs = [
      '/settings/public-settings',  // 公开设置（不需要认证）
      '/auth/login',  // 登录接口
      '/auth/register',  // 注册接口
      '/auth/login-json',  // JSON登录接口
      '/auth/refresh',  // 刷新token接口
      '/auth/forgot-password',  // 忘记密码
      '/auth/reset-password',  // 重置密码
      '/settings/announcements'  // 公告（可能不需要认证，取决于后端实现）
    ]
    
    // 判断是否为公开API
    const isPublicAPI = config.url && publicAPIs.some(api => config.url.startsWith(api))
    
    // 判断是否为管理员API：
    // 1. 路径以 '/admin' 开头
    // 2. 路径中包含 '/admin/'
    // 3. 特定的管理员API路径（即使不在 /admin 下）
    // 4. 如果在管理员后台（/admin路径下），则所有API都使用admin_token（包括/users/开头的API和/tickets/开头的API）
    const adminPaths = [
      '/admin',
      '/payment-config',  // 支付配置（管理员功能）
      '/software-config',  // 软件配置（管理员功能）
      '/config/admin',  // 配置管理
      '/notifications/admin',  // 通知管理
      '/tickets/admin',  // 工单管理（管理员）
      '/coupons/admin',  // 优惠券管理（管理员）
      '/announcements/admin'  // 公告管理（管理员）
    ]
    const isAdminAPI = config.url && (
      config.url.startsWith('/admin') || 
      config.url.includes('/admin/') ||
      adminPaths.some(path => config.url.startsWith(path)) ||
      (isInAdminPanel && (config.url.startsWith('/users/') || config.url.startsWith('/tickets/')))  // 在管理员后台时，/users/和/tickets/开头的API也使用admin_token
    )
    
    // 如果是公开API，不需要token，直接返回
    if (isPublicAPI) {
      return config
    }
    
    // 根据 API 类型获取对应的 token
    const token = isAdminAPI
      ? secureStorage.get('admin_token')
      : secureStorage.get('user_token')
    
    // 如果token不存在，在开发环境下输出警告（公开API除外）
    if (!token && process.env.NODE_ENV === 'development' && !isPublicAPI) {
      console.warn(`[API] 缺少 ${isAdminAPI ? 'admin' : 'user'} token for ${config.url}`)
    }
    
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    
    // 添加CSRF token（从Cookie中获取）
    const csrfToken = getCookie('csrf_token')
    if (csrfToken && (config.method === 'post' || config.method === 'put' || config.method === 'delete' || config.method === 'patch')) {
      config.headers['X-CSRF-Token'] = csrfToken
    }
    
    return config
  },
  error => Promise.reject(error)
)

function getCookie(name) {
  const value = `; ${document.cookie}`
  const parts = value.split(`; ${name}=`)
  if (parts.length === 2) return parts.pop().split(';').shift()
  return null
}

// 防止多个请求同时刷新 token（按角色区分）
let isRefreshing = { admin: false, user: false }
let failedQueue = []
let refreshFailed = { admin: false, user: false }

// 清除指定角色的 token
const clearRoleTokens = (isAdmin) => {
  if (isAdmin) {
    secureStorage.remove('admin_token')
    secureStorage.remove('admin_user')
    secureStorage.remove('admin_refresh_token')
  } else {
    secureStorage.remove('user_token')
    secureStorage.remove('user_data')
    secureStorage.remove('user_refresh_token')
  }
}

// 检查当前路径是否匹配 API 角色
const shouldHandleLogout = (isAdminAPI) => {
  const currentPath = typeof window !== 'undefined' ? window.location.pathname : ''
  return (isAdminAPI && currentPath.startsWith('/admin')) || (!isAdminAPI && !currentPath.startsWith('/admin'))
}

const processQueue = (error, token = null, isAdmin = null) => {
  const queueToProcess = isAdmin !== null
    ? failedQueue.filter(prom => prom.isAdmin === isAdmin)
    : failedQueue

  queueToProcess.forEach(prom => {
    if (error) {
      prom.reject(error)
    } else {
      prom.resolve(token)
    }
  })

  if (isAdmin !== null) {
    failedQueue = failedQueue.filter(prom => prom.isAdmin !== isAdmin)
  } else {
    failedQueue = []
  }
}

export const resetRefreshFailed = () => {
  refreshFailed.admin = false
  refreshFailed.user = false
}

api.interceptors.response.use(
  response => response,
  async error => {
    // 处理网络错误和超时
    if (!error.response) {
      if (error.code === 'ECONNABORTED' || error.message?.includes('timeout')) {
        // 请求超时，尝试重试一次
        if (error.config && !error.config._retry && !error.config._skipRetry) {
          error.config._retry = true
          console.warn('请求超时，正在重试...', error.config.url)
          return api.request(error.config)
        }
      } else if (error.code === 'ERR_NETWORK' || error.message?.includes('Network Error')) {
        // 网络错误，可能是连接断开
        console.warn('网络连接错误，请检查网络连接')
        // 如果页面可见，尝试重新连接
        if (isVisible() && !reconnectTimer) {
          reconnectTimer = setTimeout(() => {
            testConnection()
            reconnectTimer = null
          }, 2000)
        }
      }
    }
    if (error.config?.responseType === 'blob' && error.response?.data instanceof Blob) {
      try {
        const text = await error.response.data.text()
        error.response.data = JSON.parse(text)
      } catch (e) {}
    }
    if (error.response?.status === 401) {
      // 由于 baseURL 是 '/api/v1'，config.url 不包含 baseURL 前缀
      // 获取当前路径，用于判断是否在管理员后台
      const currentPath = typeof window !== 'undefined' ? window.location.pathname : ''
      const isInAdminPanel = currentPath.startsWith('/admin')
      
      // 判断是否为管理员API：路径以 '/admin' 开头，或者路径中包含 '/admin/'，或者特定的管理员API路径
      // 如果在管理员后台，/users/开头的API也使用admin_token
      const adminPaths = [
        '/admin',
        '/payment-config',
        '/software-config',
        '/config/admin',
        '/notifications/admin',
        '/tickets/admin',
        '/coupons/admin',
        '/announcements/admin'
      ]
      const isAdminAPI = error.config?.url && (
        error.config.url.startsWith('/admin') || 
        error.config.url.includes('/admin/') ||
        adminPaths.some(path => error.config.url.startsWith(path)) ||
        (isInAdminPanel && (error.config.url.startsWith('/users/') || error.config.url.startsWith('/tickets/')))  // 在管理员后台时，/users/和/tickets/开头的API也使用admin_token
      )
      const refreshKey = isAdminAPI ? 'admin' : 'user'

      if (refreshFailed[refreshKey]) {
        if (shouldHandleLogout(isAdminAPI)) {
          handleLogout()
        }
        return Promise.reject(error)
      }

      // 如果是登录API失败，不要尝试刷新token，直接返回错误
      if (error.config?.url?.includes('/auth/login-json') || error.config?.url?.includes('/auth/login')) {
        return Promise.reject(error)
      }

      if (error.config?.url?.includes('/auth/refresh')) {
        refreshFailed[refreshKey] = true
        clearRoleTokens(isAdminAPI)
        if (shouldHandleLogout(isAdminAPI)) {
          handleLogout()
        }
        return Promise.reject(error)
      }

      if (error.config && !error.config._retry) {
        if (isRefreshing[refreshKey]) {
          return new Promise((resolve, reject) => {
            failedQueue.push({ resolve, reject, isAdmin: isAdminAPI })
          })
            .then(token => {
              if (token) {
                error.config.headers.Authorization = `Bearer ${token}`
                return api(error.config)
              } else {
                return Promise.reject(new Error('Token刷新失败'))
              }
            })
            .catch(err => {
              return Promise.reject(err)
            })
        }

        error.config._retry = true
        isRefreshing[refreshKey] = true

        try {
          const refreshTokenKey = isAdminAPI ? 'admin_refresh_token' : 'user_refresh_token'
          const refreshToken = secureStorage.get(refreshTokenKey)

          const refreshResponse = await axios.post('/api/v1/auth/refresh', {}, {
            withCredentials: true,
            timeout: 5000,
            headers: refreshToken ? { Authorization: `Bearer ${refreshToken}` } : {}
          })
          const { access_token, refresh_token } = refreshResponse.data || {}
          if (access_token) {
            const TOKEN_TTL = 24 * 60 * 60 * 1000
            const REFRESH_TOKEN_TTL = 7 * 24 * 60 * 60 * 1000
            if (isAdminAPI) {
              secureStorage.set('admin_token', access_token, false, TOKEN_TTL)
              if (refresh_token) {
                secureStorage.set('admin_refresh_token', refresh_token, false, REFRESH_TOKEN_TTL)
              }
            } else {
              secureStorage.set('user_token', access_token, true, TOKEN_TTL)
              if (refresh_token) {
                secureStorage.set('user_refresh_token', refresh_token, true, REFRESH_TOKEN_TTL)
              }
            }

            if (_useAuthStore && shouldHandleLogout(isAdminAPI)) {
              const authStore = _useAuthStore()
              authStore.setToken(access_token)
            }

            error.config.headers.Authorization = `Bearer ${access_token}`
            processQueue(null, access_token, isAdminAPI)
            isRefreshing[refreshKey] = false
            return api(error.config)
          } else {
            throw new Error('Token刷新返回空值')
          }
        } catch (refreshError) {
          refreshFailed[refreshKey] = true
          clearRoleTokens(isAdminAPI)
          processQueue(refreshError, null, isAdminAPI)
          isRefreshing[refreshKey] = false
          if (shouldHandleLogout(isAdminAPI)) {
            handleLogout()
          }
          return Promise.reject(refreshError)
        }
      } else {
        clearRoleTokens(isAdminAPI)
        if (shouldHandleLogout(isAdminAPI)) {
          handleLogout()
        }
        return Promise.reject(error)
      }
    }
    return Promise.reject(error)
  }
)

export const authAPI = {
  login: (data) => api.post('/auth/login', data),
  register: (data) => api.post('/auth/register', data),
  sendVerificationCode: (data) => api.post('/auth/verification/send', data),
  resendVerificationCode: (data) => api.post('/auth/verification/send', data),
  forgotPassword: (data) => api.post('/auth/forgot-password', data),
  refreshToken: () => api.post('/auth/refresh-token')
}

export const userAPI = {
  getProfile: () => api.get('/users/profile'),
  updateProfile: (data) => api.put('/users/profile', data),
  changePassword: (data) => api.post('/users/change-password', data),
  getLoginHistory: () => api.get('/users/login-history'),
  getUserActivities: () => api.get('/users/activities'),
  getSubscriptionResets: () => api.get('/users/subscription-resets'),
  getUserInfo: () => api.get('/users/dashboard-info'),
  getAnnouncements: () => api.get('/announcements/'),
  getUserDevices: () => api.get('/users/devices'),
  getMyLevel: () => api.get('/users/my-level'),
  getUserLevels: (activeOnly = true) => api.get('/user-levels', { params: { active_only: activeOnly } })
}

export const rechargeAPI = {
  createRecharge: (amount, paymentMethod = 'alipay') => api.post('/recharge/create', { amount, payment_method: paymentMethod }),
  getRecharges: (params) => api.get('/recharge/', { params }),
  getRechargeDetail: (rechargeId) => api.get(`/recharge/${rechargeId}`),
  cancelRecharge: (rechargeId) => api.post(`/recharge/${rechargeId}/cancel`)
}

export const subscriptionAPI = {
  getUserSubscription: () => api.get('/subscriptions/user-subscription'),
  resetSubscription: () => api.post('/subscriptions/reset-subscription'),
  sendSubscriptionEmail: () => api.post('/subscriptions/send-subscription-email'),
  getDevices: () => api.get('/subscriptions/devices'),
  removeDevice: (deviceId) => api.delete(`/subscriptions/devices/${deviceId}`),
  getSSRSubscription: (key) => api.get(`/subscriptions/ssr/${key}`),
  getClashSubscription: (key) => api.get(`/subscriptions/clash/${key}`),
  convertToBalance: () => api.post('/subscriptions/convert-to-balance')
}

export const packageAPI = {
  getPackages: (params) => api.get('/packages/', { params }),
  getPackage: (packageId) => api.get(`/packages/${packageId}`),
  createPackage: (data) => api.post('/packages/', data),
  updatePackage: (packageId, data) => api.put(`/packages/${packageId}`, data),
  deletePackage: (packageId) => api.delete(`/packages/${packageId}`)
}

export const orderAPI = {
  upgradeDevices: (data) => api.post('/orders/upgrade-devices', data),
  createOrder: (data) => api.post('/orders/create', data),
  getUserOrders: (params) => api.get('/orders/', { params }),
  getOrderStatus: (orderNo) => api.get(`/orders/${orderNo}/status`),
  cancelOrder: (orderNo) => api.post(`/orders/${orderNo}/cancel`),
  getPackages: () => api.get('/packages/')
}

export const nodeAPI = {
  getNodes: () => api.get('/nodes/'),
  getNode: (nodeId) => api.get(`/nodes/${nodeId}`),
  testNode: (nodeId) => api.post(`/nodes/${nodeId}/test`),
  batchTestNodes: (nodeIds) => api.post('/nodes/batch-test', nodeIds),
  importFromClash: (clashConfig) => api.post('/nodes/import-from-clash', { clash_config: clashConfig }),
  getNodesStats: () => api.get('/admin/nodes/stats')
}

export const adminAPI = {
  getDashboard: () => api.get('/admin/dashboard'),
  getStats: () => api.get('/admin/stats'),
  getUsers: (params) => api.get('/admin/users', { params }),
  getUserStatistics: () => api.get('/admin/users/statistics'),
  getOrders: (params) => api.get('/admin/orders', { params }),
  createUser: (data) => api.post('/admin/users', data),
  getUser: (userId) => api.get(`/admin/users/${userId}`),
  updateUser: (userId, data) => api.put(`/admin/users/${userId}`, data),
  deleteUser: (userId) => api.delete(`/admin/users/${userId}`),
  loginAsUser: (userId) => api.post(`/admin/users/${userId}/login-as`),
  getAbnormalUsers: (params) => api.get('/admin/users/abnormal', { params }),
  getUserDetails: (userId) => api.get(`/admin/users/${userId}/details`),
  updateUserStatus: (userId, status) => api.put(`/admin/users/${userId}/status`, { status }),
  resetUserPassword: (userId, password) => api.post(`/admin/users/${userId}/reset-password`, { password }),
  unlockUserLogin: (userId) => api.post(`/admin/users/${userId}/unlock-login`),
  batchDeleteUsers: (userIds) => api.post('/admin/users/batch-delete', { user_ids: userIds }),
  batchEnableUsers: (userIds) => api.post('/admin/users/batch-enable', { user_ids: userIds }),
  batchDisableUsers: (userIds) => api.post('/admin/users/batch-disable', { user_ids: userIds }),
  batchVerifyUsers: (userIds) => api.post('/admin/users/batch-verify', { user_ids: userIds }),
  sendSubscriptionEmail: (userId) => api.post(`/admin/users/${userId}/send-subscription-email`),
  batchSendSubscriptionEmail: (userIds) => api.post('/admin/users/batch-send-subscription-email', { user_ids: userIds }),
  getExpiringUsers: (params) => api.get('/admin/users/expiring', { params }),
  batchSendExpireReminder: (userIds) => api.post('/admin/users/batch-expire-reminder', { user_ids: userIds }),
  getSubscriptions: (params) => api.get('/admin/subscriptions', { params }),
  createSubscription: (data) => api.post('/admin/subscriptions', data),
  updateSubscription: (subscriptionId, data) => api.put(`/admin/subscriptions/${subscriptionId}`, data),
  resetSubscription: (subscriptionId) => api.post(`/admin/subscriptions/${subscriptionId}/reset`),
  extendSubscription: (subscriptionId, days) => api.post(`/admin/subscriptions/${subscriptionId}/extend`, { days }),
  resetUserSubscription: (userId) => api.post(`/admin/subscriptions/user/${userId}/reset-all`),
  sendSubscriptionEmail: (userId) => api.post(`/admin/subscriptions/user/${userId}/send-email`),
  batchClearDevices: (data) => api.post('/admin/subscriptions/batch-clear-devices', data),
  exportSubscriptions: () => api.get('/admin/subscriptions/export', { responseType: 'blob' }),
  getAppleStats: () => api.get('/admin/subscriptions/apple-stats'),
  getOnlineStats: () => api.get('/admin/subscriptions/online-stats'),
  clearUserDevices: (userId) => api.delete(`/admin/subscriptions/user/${userId}/delete-all`),
  batchDeleteSubscriptions: (subscriptionIds) => api.post('/admin/subscriptions/batch-delete', { subscription_ids: subscriptionIds }),
  batchEnableSubscriptions: (subscriptionIds) => api.post('/admin/subscriptions/batch-enable', { subscription_ids: subscriptionIds }),
  batchDisableSubscriptions: (subscriptionIds) => api.post('/admin/subscriptions/batch-disable', { subscription_ids: subscriptionIds }),
  batchResetSubscriptions: (subscriptionIds) => api.post('/admin/subscriptions/batch-reset', { subscription_ids: subscriptionIds }),
  batchSendSubscriptionEmail: (subscriptionIds) => api.post('/admin/subscriptions/batch-send-email', { subscription_ids: subscriptionIds }),
  updateOrder: (orderId, data) => api.put(`/admin/orders/${orderId}`, data),
  getPackages: (params) => api.get('/admin/packages', { params }),
  createPackage: (data) => api.post('/admin/packages', data),
  updatePackage: (packageId, data) => api.put(`/admin/packages/${packageId}`, data),
  deletePackage: (packageId) => api.delete(`/admin/packages/${packageId}`),
  getEmailQueue: (params) => api.get('/admin/email-queue', { params }),
  resendEmail: (emailId) => api.post(`/admin/email-queue/${emailId}/resend`),
  getEmailDetail: (emailId) => api.get(`/admin/email-queue/${emailId}`),
  retryEmail: (emailId) => api.post(`/admin/email-queue/${emailId}/retry`),
  deleteEmailFromQueue: (emailId) => api.delete(`/admin/email-queue/${emailId}`),
  clearEmailQueue: (status) => api.post(`/admin/email-queue/clear${status ? `?status=${status}` : ''}`),
  getEmailQueueStatistics: () => api.get('/admin/email-queue/statistics'),
  getProfile: () => api.get('/admin/profile'),
  updateProfile: (data) => api.put('/admin/profile', data),
  changePassword: (data) => api.post('/admin/change-password', data),
  getLoginHistory: () => api.get('/admin/login-history'),
  getSecuritySettings: () => api.get('/admin/security-settings'),
  updateSecuritySettings: (data) => api.put('/admin/security-settings', data),
  getNotificationSettings: () => api.get('/admin/notification-settings'),
  updateNotificationSettings: (data) => api.put('/admin/notification-settings', data),
  updateAdminNotificationSettings: (settings) => api.put('/admin/settings/admin-notification', settings),
  testAdminEmailNotification: () => api.post('/admin/settings/admin-notification/test/email'),
  testAdminTelegramNotification: () => api.post('/admin/settings/admin-notification/test/telegram'),
  testAdminBarkNotification: () => api.post('/admin/settings/admin-notification/test/bark'),
  getSystemLogs: (params) => api.get('/admin/system-logs', { params }),
  getLogsStats: () => api.get('/admin/logs-stats'),
  exportLogs: (params) => api.get('/admin/export-logs', { params, responseType: 'blob' }),
  clearLogs: () => api.post('/admin/clear-logs'),
  getUserDevices: (userId) => api.get(`/admin/users/${userId}/devices`),
  getSubscriptionDevices: (subscriptionId) => api.get(`/admin/subscriptions/${subscriptionId}/devices`),
  getDeviceDetail: (deviceId) => api.get(`/admin/devices/devices/${deviceId}`),
  updateDeviceStatus: (deviceId, data) => api.put(`/admin/devices/devices/${deviceId}`, data),
  removeDevice: (deviceId) => api.delete(`/admin/devices/${deviceId}`),
  deleteUserDevice: (userId, deviceId) => api.delete(`/admin/users/${userId}/devices/${deviceId}`)
}

export const notificationAPI = {
  getUserNotifications: (params) => api.get('/notifications/user-notifications', { params }),
  getUnreadCount: () => api.get('/notifications/unread-count'),
  markAsRead: (notificationId) => api.post(`/notifications/${notificationId}/read`),
  markAllAsRead: () => api.post('/notifications/mark-all-read'),
  getNotifications: (params) => api.get('/notifications/admin/notifications', { params }),
  createNotification: (data) => api.post('/notifications/admin/notifications', data),
  updateNotification: (notificationId, data) => api.put(`/notifications/admin/notifications/${notificationId}`, data),
  deleteNotification: (notificationId) => api.delete(`/notifications/admin/notifications/${notificationId}`),
  deleteAdminNotification: (notificationId) => api.delete(`/notifications/admin/notifications/${notificationId}`),
  broadcastNotification: (data) => api.post('/notifications/admin/notifications/broadcast', data)
}

export const configAPI = {
  getConfigFiles: () => api.get('/config/admin/config-files'),
  getConfigFileContent: (fileName) => api.get(`/config/admin/config-files/${fileName}`),
  saveConfigFile: (fileName, content) => api.post(`/config/admin/config-files/${fileName}`, { content }),
  backupConfigFile: (fileName) => api.post(`/config/admin/config-files/${fileName}/backup`),
  restoreConfigFile: (fileName) => api.post(`/config/admin/config-files/${fileName}/restore`),
  getSystemConfig: () => api.get('/admin/system-config'),
  saveSystemConfig: (data) => api.post('/admin/system-config', data),
  getEmailConfig: () => api.get('/admin/email-config'),
  saveEmailConfig: (data) => api.post('/admin/email-config', data),
  getClashConfig: () => api.get('/admin/clash-config'),
  saveClashConfig: (content) => api.post('/admin/clash-config', { content }),
  getClashConfigInvalid: () => api.get('/admin/clash-config-invalid'),
  saveClashConfigInvalid: (content) => api.post('/admin/clash-config-invalid', { content }),
  getV2rayConfig: () => api.get('/admin/v2ray-config'),
  saveV2rayConfig: (content) => api.post('/admin/v2ray-config', { content }),
  getV2rayConfigInvalid: () => api.get('/admin/v2ray-config-invalid'),
  saveV2rayConfigInvalid: (content) => api.post('/admin/v2ray-config-invalid', { content }),
  exportConfig: () => api.get('/admin/export-config'),
  importConfig: (data) => api.post('/admin/import-config', data)
}

export const statisticsAPI = {
  getStatistics: () => api.get('/admin/statistics'),
  getUserTrend: () => api.get('/admin/statistics/user-trend'),
  getRevenueTrend: () => api.get('/admin/statistics/revenue-trend'),
  getUserStatistics: (params) => api.get('/admin/statistics/users', { params }),
  getSubscriptionStatistics: () => api.get('/admin/statistics/subscriptions'),
  getOrderStatistics: (params) => api.get('/admin/statistics/orders', { params }),
  getStatisticsOverview: () => api.get('/admin/statistics/overview'),
  exportStatistics: (type, format) => api.get('/admin/statistics/export', { params: { type, format } })
}

export const paymentAPI = {
  getPaymentMethods: () => api.get('/payment-methods/active'),
  createPayment: (data) => api.post('/payment/create', data),
  getPaymentStatus: (transactionId) => api.get(`/payment/transactions/${transactionId}`),
  getPaymentConfigs: (params) => api.get('/payment-config/', { params }),
  createPaymentConfig: (data) => api.post('/payment-config/', data),
  updatePaymentConfig: (configId, data) => api.put(`/payment-config/${configId}`, data),
  deletePaymentConfig: (configId) => api.delete(`/payment-config/${configId}`),
  exportPaymentConfigs: (params) => api.get('/payment-config/export', { params, responseType: 'blob' }),
  getPaymentConfigStats: () => api.get('/payment-config/stats/summary'),
  bulkEnablePaymentConfigs: (configIds) => api.post('/payment-config/bulk-enable', configIds),
  bulkDisablePaymentConfigs: (configIds) => api.post('/payment-config/bulk-disable', configIds),
  bulkDeletePaymentConfigs: (configIds) => api.post('/payment-config/bulk-delete', configIds),
  getPaymentTransactions: (params) => api.get('/admin/payment-transactions', { params }),
  getPaymentTransactionDetail: (transactionId) => api.get(`/admin/payment-transactions/${transactionId}`),
  getPaymentStats: () => api.get('/admin/payment-stats'),
  getConfigUpdateStatus: () => api.get('/admin/config-update/status'),
  startConfigUpdate: () => api.post('/admin/config-update/start'),
  stopConfigUpdate: () => api.post('/admin/config-update/stop'),
  testConfigUpdate: () => api.post('/admin/config-update/test'),
  getConfigUpdateLogs: (params) => api.get('/admin/config-update/logs', { params }),
  getConfigUpdateConfig: () => api.get('/admin/config-update/config'),
  updateConfigUpdateConfig: (data) => api.put('/admin/config-update/config', data),
  getConfigUpdateFiles: () => api.get('/admin/config-update/files'),
  getConfigUpdateSchedule: () => api.get('/admin/config-update/schedule'),
  updateConfigUpdateSchedule: (data) => api.put('/admin/config-update/schedule', data),
  startConfigUpdateSchedule: () => api.post('/admin/config-update/schedule/start'),
  stopConfigUpdateSchedule: () => api.post('/admin/config-update/schedule/stop'),
  clearConfigUpdateLogs: () => api.post('/admin/config-update/logs/clear')
}

export const settingsAPI = {
  getPublicSettings: () => api.get('/settings/public-settings'),
  getAnnouncements: (params) => api.get('/settings/announcements', { params }),
  getSystemSettings: () => api.get('/admin/settings'),
  updateSystemSettings: (data) => api.put('/admin/settings', data),
  getConfigsByCategory: (params) => api.get('/admin/configs', { params }),
  createConfig: (data) => api.post('/admin/configs', data),
  updateConfig: (configKey, data) => api.put(`/admin/configs/${configKey}`, data),
  deleteConfig: (configKey) => api.delete(`/admin/configs/${configKey}`),
  initializeConfigs: () => api.post('/admin/configs/initialize'),
  getAnnouncementsAdmin: (params) => api.get('/admin/announcements', { params }),
  getAnnouncementDetail: (announcementId) => api.get(`/admin/announcements/${announcementId}`),
  createAnnouncement: (data) => api.post('/admin/announcements', data),
  updateAnnouncement: (announcementId, data) => api.put(`/admin/announcements/${announcementId}`, data),
  deleteAnnouncement: (announcementId) => api.delete(`/admin/announcements/${announcementId}`),
  toggleAnnouncementStatus: (announcementId) => api.post(`/admin/announcements/${announcementId}/toggle-status`),
  toggleAnnouncementPin: (announcementId) => api.post(`/admin/announcements/${announcementId}/toggle-pin`),
  publishAnnouncement: (data) => api.post('/announcements/admin/publish', data),
  getAdminAnnouncements: (page = 1, size = 20) => api.get(`/announcements/admin/list?page=${page}&size=${size}`),
  getThemeConfigs: () => api.get('/admin/themes'),
  createThemeConfig: (data) => api.post('/admin/themes', data),
  updateThemeConfig: (themeId, data) => api.put(`/admin/themes/${themeId}`, data),
  deleteThemeConfig: (themeId) => api.delete(`/admin/themes/${themeId}`)
}

export const softwareConfigAPI = {
  getSoftwareConfig: () => api.get('/software-config/'),
  updateSoftwareConfig: (data) => api.put('/software-config/', data)
}

export const configUpdateAPI = {
  getStatus: () => api.get('/admin/config-update/status'),
  startUpdate: () => api.post('/admin/config-update/start'),
  stopUpdate: () => api.post('/admin/config-update/stop'),
  testUpdate: () => api.post('/admin/config-update/test'),
  getConfig: () => api.get('/admin/config-update/config'),
  updateConfig: (data) => api.put('/admin/config-update/config', data),
  getFiles: () => api.get('/admin/config-update/files'),
  getLogs: (params) => api.get('/admin/config-update/logs', { params }),
  clearLogs: () => api.post('/admin/config-update/logs/clear'),
  getNodeSources: () => api.get('/admin/config-update/node-sources'),
  updateNodeSources: (data) => api.put('/admin/config-update/node-sources', data),
  getFilterKeywords: () => api.get('/admin/config-update/filter-keywords'),
  updateFilterKeywords: (data) => api.put('/admin/config-update/filter-keywords', data)
}

export const ticketAPI = {
  createTicket: (data) => api.post('/tickets/', data),
  getUserTickets: (params) => api.get('/tickets/', { params }),
  getTicket: (ticketId) => api.get(`/tickets/${ticketId}`),
  getAdminTicket: (ticketId) => api.get(`/tickets/admin/${ticketId}`),  // 管理员专用
  addReply: (ticketId, data) => api.post(`/tickets/${ticketId}/replies`, data),
  addRating: (ticketId, data) => api.post(`/tickets/${ticketId}/rating`, data),
  getAllTickets: (params) => api.get('/tickets/admin/all', { params }),
  updateTicket: (ticketId, data) => api.put(`/tickets/admin/${ticketId}`, data),
  getTicketStatistics: () => api.get('/tickets/admin/statistics')
}

export const couponAPI = {
  getAvailableCoupons: () => api.get('/coupons/available'),
  validateCoupon: (data) => api.post('/coupons/validate', data),
  createCoupon: (data) => api.post('/coupons/admin', data),
  getAllCoupons: (params) => api.get('/coupons/admin', { params }),
  getCoupon: (couponId) => api.get(`/coupons/admin/${couponId}`),
  updateCoupon: (couponId, data) => api.put(`/coupons/admin/${couponId}`, data),
  deleteCoupon: (couponId) => api.delete(`/coupons/admin/${couponId}`),
  getCouponStatistics: () => api.get('/coupons/admin/statistics')
}

export const inviteAPI = {
  generateInviteCode: (data) => api.post('/invites', data),
  getMyInviteCodes: () => api.get('/invites/my-codes'),
  getInviteStats: () => api.get('/invites/stats'),
  getInviteRewardSettings: () => api.get('/invites/reward-settings'),
  validateInviteCode: (code) => api.get(`/invites/validate/${code}`),
  updateInviteCode: (codeId, data) => api.put(`/invites/${codeId}`, data),
  deleteInviteCode: (codeId) => api.delete(`/invites/${codeId}`),
  getAllInviteCodes: (params) => api.get('/admin/invites', { params }),
  getInviteRelations: (params) => api.get('/admin/invite-relations', { params }),
  getAdminInviteStatistics: () => api.get('/admin/invite-statistics'),
  batchDeleteInviteCodes: (codeIds) => api.post('/admin/invites/batch-delete', codeIds),
  batchDeleteInviteRelations: (relationIds) => api.post('/admin/invite-relations/batch-delete', relationIds)
}

export const userLevelAPI = {
  getUserLevels: (activeOnly = true) => api.get('/user-levels', { params: { active_only: activeOnly } }),
  getMyLevel: () => api.get('/users/my-level'),
  // 管理员API
  getAllLevels: (activeOnly, isActive) => {
    const params = {}
    if (isActive !== undefined) {
      params.is_active = isActive
    } else if (activeOnly !== undefined) {
      params.active_only = activeOnly
    }
    return api.get('/admin/user-levels', { params })
  },
  getLevelDetail: (levelId) => api.get(`/admin/user-levels/${levelId}`),
  createLevel: (data) => api.post('/admin/user-levels', data),
  updateLevel: (levelId, data) => api.put(`/admin/user-levels/${levelId}`, data),
  deleteLevel: (levelId) => api.delete(`/admin/user-levels/${levelId}`),
  upgradeUsers: (levelId, userIds) => api.post(`/admin/user-levels/${levelId}/upgrade-users`, userIds)
}

export default api