import axios from 'axios'
import { secureStorage } from '@/utils/secureStorage'
import { onPageVisible, isVisible } from '@/utils/pageVisibility'


let _router = null
let _useAuthStore = null
let reconnectTimer = null
let csrfTokenCache = null

// Token 刷新状态管理
let isRefreshing = { admin: false, user: false }
let failedQueue = []
let refreshFailed = { admin: false, user: false }

// 常量定义
const BASE_URL = '/api/v1'
const TIMEOUT = 60000 // 60秒
const TEST_CONNECTION_TIMEOUT = 10000 // 10秒

// 公开 API 列表（不需要认证）
const PUBLIC_APIS = [
  '/settings/public-settings',
  '/auth/login',
  '/auth/register',
  '/auth/login-json',
  '/auth/refresh',
  '/auth/forgot-password',
  '/auth/reset-password'
]

// 管理员 API 路径特征
const ADMIN_PATHS = [
  '/admin',
  '/payment-config',
  '/software-config',
  '/config/admin',
  '/tickets/admin',
  '/coupons/admin'
]


export const initApi = (router, useAuthStore) => {
  _router = router
  _useAuthStore = useAuthStore
}

export const api = axios.create({
  baseURL: BASE_URL,
  timeout: TIMEOUT,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true
})

export const useApi = () => api
export const resetRefreshFailed = () => {
  refreshFailed.admin = false
  refreshFailed.user = false
}

// 页面可见性处理
onPageVisible(() => {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  testConnection()
})

async function testConnection() {
  try {
    await axios.get(`${BASE_URL}/settings/public-settings`, {
      timeout: TEST_CONNECTION_TIMEOUT,
      withCredentials: true
    })
  } catch (error) {
    if (error.code !== 'ECONNABORTED') {
      if (process.env.NODE_ENV === 'development') {
        console.warn('连接测试失败，可能需要刷新页面:', error.message)
      }
    }
  }
}

function getCookie(name) {
  const value = `; ${document.cookie}`
  const parts = value.split(`; ${name}=`)
  if (parts.length === 2) return parts.pop().split(';').shift()
  return null
}

const clearRoleTokens = (isAdmin) => {
  const prefix = isAdmin ? 'admin' : 'user'
  secureStorage.remove(`${prefix}_token`)
  secureStorage.remove(`${prefix}_${isAdmin ? 'user' : 'data'}`)
  secureStorage.remove(`${prefix}_refresh_token`)
}

const shouldHandleLogout = (isAdminAPI) => {
  const currentPath = typeof window !== 'undefined' ? window.location.pathname : ''
  return (isAdminAPI && currentPath.startsWith('/admin')) || (!isAdminAPI && !currentPath.startsWith('/admin'))
}

const handleLogout = () => {
  if (_useAuthStore) _useAuthStore().logout()
  if (_router) {
    const currentPath = _router.currentRoute.value.path
    if (currentPath.startsWith('/admin')) {
      if (currentPath !== '/admin/login') _router.push('/admin/login')
    } else {
      if (currentPath !== '/login' && currentPath !== '/forgot-password') _router.push('/login')
    }
  }
}

const processQueue = (error, token = null, isAdmin = null) => {
  const queueToProcess = isAdmin !== null ? failedQueue.filter(prom => prom.isAdmin === isAdmin) : failedQueue
  queueToProcess.forEach(prom => error ? prom.reject(error) : prom.resolve(token))
  failedQueue = isAdmin !== null ? failedQueue.filter(prom => prom.isAdmin !== isAdmin) : []
}

// =========================================================================================
// Request Interceptor
// =========================================================================================

api.interceptors.request.use(
  config => {
    const currentPath = typeof window !== 'undefined' ? window.location.pathname : ''
    const isInAdminPanel = currentPath.startsWith('/admin')
    
    // 1. 判断是否为公开 API
    if (config.url && PUBLIC_APIS.some(api => config.url.startsWith(api))) {
      return config
    }

    // 2. 判断是否为管理员 API
    const isAdminAPI = config.url && (
      config.url.startsWith('/admin') || 
      config.url.includes('/admin/') ||
      ADMIN_PATHS.some(path => config.url.startsWith(path)) ||
      (isInAdminPanel && (config.url.startsWith('/users/') || config.url.startsWith('/tickets/')))
    )

    // 3. 获取并注入 Token
    let token = isAdminAPI ? secureStorage.get('admin_token') : secureStorage.get('user_token')
    
    // 如果是用户端 API 但没有用户 token，尝试使用管理员 token（允许管理员查看用户端内容）
    if (!token && !isAdminAPI) {
      token = secureStorage.get('admin_token')
    }
    
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }

    // 4. CSRF Token 处理
    let csrfToken = csrfTokenCache || getCookie('csrf_token')
    if (!['get', 'head', 'options'].includes(config.method)) {
      if (!csrfToken) {
        csrfToken = getCookie('csrf_token')
        if (csrfToken) csrfTokenCache = csrfToken
      }
      if (csrfToken) config.headers['X-CSRF-Token'] = csrfToken
    }
    
    return config
  },
  error => Promise.reject(error)
)

// =========================================================================================
// Response Interceptor
// =========================================================================================

api.interceptors.response.use(
  response => {
    // 缓存 CSRF Token
    const csrfToken = response.headers['x-csrf-token'] || response.headers['X-CSRF-Token']
    if (csrfToken) {
      csrfTokenCache = csrfToken
    } else {
      const cookieToken = getCookie('csrf_token')
      if (cookieToken) csrfTokenCache = cookieToken
    }
    return response
  },
  async error => {
    // 1. 网络错误与超时处理
    if (!error.response) {
      if ((error.code === 'ECONNABORTED' || error.message?.includes('timeout')) && error.config && !error.config._retry) {
        error.config._retry = true
        if (process.env.NODE_ENV === 'development') {
          console.warn('请求超时，正在重试...', error.config.url)
        }
        return api.request(error.config)
      } else if (error.code === 'ERR_NETWORK' || error.message?.includes('Network Error')) {
        if (process.env.NODE_ENV === 'development') {
          console.warn('网络连接错误，请检查网络连接')
        }
        if (isVisible() && !reconnectTimer) {
          reconnectTimer = setTimeout(() => { testConnection(); reconnectTimer = null }, 2000)
        }
      }
      return Promise.reject(error)
    }

    // Blob 错误转 JSON
    if (error.config?.responseType === 'blob' && error.response?.data instanceof Blob) {
      try {
        const text = await error.response.data.text()
        error.response.data = JSON.parse(text)
      } catch (e) {}
    }

    // 2. 维护模式 (503)
    if (error.response?.status === 503 && error.response?.data?.maintenance_mode) {
      const { ElMessage } = await import('element-plus')
      ElMessage.error(error.response.data.message || '系统维护中，请稍后再试')
      return Promise.reject(error)
    }

    // 3. CSRF 验证失败 (403)
    if (error.response?.status === 403 && (error.response?.data?.message?.includes('CSRF') || error.response?.data?.csrf_token)) {
      const newCsrfToken = error.response?.data?.csrf_token || error.response?.headers?.['x-csrf-token'] || error.response?.headers?.['X-CSRF-Token']
      if (newCsrfToken && error.config && !error.config._csrfRetry) {
        csrfTokenCache = newCsrfToken
        if (['post', 'put', 'delete', 'patch'].includes(error.config?.method?.toLowerCase())) {
          error.config._csrfRetry = true
          error.config.headers['X-CSRF-Token'] = newCsrfToken
          if (process.env.NODE_ENV === 'development') {
            console.log('[API] CSRF验证失败，使用新token自动重试')
          }
          return api.request(error.config)
        }
      }
      if (!error.config?._csrfRetry) {
        const { ElMessage } = await import('element-plus')
        ElMessage.error(error.response?.data?.message || 'CSRF验证失败，请刷新页面后重试')
      }
      return Promise.reject(error)
    }

    // 4. Token 过期处理 (401)
    if (error.response?.status === 401) {
      const currentPath = typeof window !== 'undefined' ? window.location.pathname : ''
      const isInAdminPanel = currentPath.startsWith('/admin')
      const isAdminAPI = error.config?.url && (
        error.config.url.startsWith('/admin') || 
        error.config.url.includes('/admin/') || 
        ADMIN_PATHS.some(path => error.config.url.startsWith(path)) ||
        (isInAdminPanel && (error.config.url.startsWith('/users/') || error.config.url.startsWith('/tickets/')))
      )
      const refreshKey = isAdminAPI ? 'admin' : 'user'

      if (refreshFailed[refreshKey] || error.config?.url?.includes('/auth/login')) {
        if (shouldHandleLogout(isAdminAPI)) handleLogout()
        return Promise.reject(error)
      }
      
      if (error.config?.url?.includes('/auth/refresh')) {
        refreshFailed[refreshKey] = true
        clearRoleTokens(isAdminAPI)
        if (shouldHandleLogout(isAdminAPI)) handleLogout()
        return Promise.reject(error)
      }

      if (error.config && !error.config._retry) {
        if (isRefreshing[refreshKey]) {
          return new Promise((resolve, reject) => failedQueue.push({ resolve, reject, isAdmin: isAdminAPI }))
            .then(token => {
              error.config.headers.Authorization = `Bearer ${token}`
              return api(error.config)
            })
            .catch(err => Promise.reject(err))
        }

        error.config._retry = true
        isRefreshing[refreshKey] = true

        try {
          const refreshToken = secureStorage.get(isAdminAPI ? 'admin_refresh_token' : 'user_refresh_token')
          const refreshResponse = await axios.post('/api/v1/auth/refresh', {}, {
            withCredentials: true,
            timeout: 5000,
            headers: refreshToken ? { Authorization: `Bearer ${refreshToken}` } : {}
          })
          
          const { access_token, refresh_token } = refreshResponse.data || {}
          if (access_token) {
            const TOKEN_TTL = 86400000 // 24h
            const REFRESH_TTL = 604800000 // 7d
            const prefix = isAdminAPI ? 'admin' : 'user'
            
            secureStorage.set(`${prefix}_token`, access_token, !isAdminAPI, TOKEN_TTL)
            if (refresh_token) secureStorage.set(`${prefix}_refresh_token`, refresh_token, !isAdminAPI, REFRESH_TTL)
            
            if (_useAuthStore && shouldHandleLogout(isAdminAPI)) _useAuthStore().setToken(access_token)

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
          if (shouldHandleLogout(isAdminAPI)) handleLogout()
          return Promise.reject(refreshError)
        }
      } else {
        clearRoleTokens(isAdminAPI)
        if (shouldHandleLogout(isAdminAPI)) handleLogout()
        return Promise.reject(error)
      }
    }
    return Promise.reject(error)
  }
)

// =========================================================================================
// API 模块导出
// =========================================================================================

export const authAPI = {
  login: (data) => api.post('/auth/login', data),
  register: (data) => api.post('/auth/register', data),
  sendVerificationCode: (data) => api.post('/auth/verification/send', data),
  resendVerificationCode: (data) => api.post('/auth/verification/send', data),
  forgotPassword: (data) => api.post('/auth/forgot-password', data),
  resetPassword: (data) => api.post('/auth/reset-password', data),
  refreshToken: () => api.post('/auth/refresh-token')
}

export const userAPI = {
  getProfile: () => api.get('/users/me'),
  updateProfile: (data) => api.put('/users/me', data),
  changePassword: (data) => api.post('/users/change-password', data),
  getLoginHistory: () => api.get('/users/login-history'),
  getUserActivities: () => api.get('/users/activities'),
  getSubscriptionResets: () => api.get('/users/subscription-resets'),
  getUserInfo: () => api.get('/users/dashboard-info'),
  getUserStatistics: () => api.get('/user/stat'), // 用户统计信息
  getUserDashboard: () => api.get('/user/dashboard'), // 聚合接口
  getUserDevices: () => api.get('/users/devices'),
  getMyLevel: () => api.get('/users/my-level'),
  getUserLevels: (activeOnly = true) => api.get('/user-levels', { params: { active_only: activeOnly } }),
  // 设备相关
  getDeviceList: () => api.get('/devices'),
  addDevice: (data) => api.post('/devices', data),
  deleteDevice: (id) => api.delete(`/devices/${id}`)
}

export const rechargeAPI = {
  createRecharge: (data) => api.post('/recharge', data), // 通用创建充值（兼容）
  createRechargeWithMethod: (amount, method = 'alipay') => api.post('/recharge/', { amount, payment_method: method }),
  getRechargeRecords: () => api.get('/recharge/records'), // 充值记录
  getRecharges: (params) => api.get('/recharge/', { params }),
  getRechargeDetail: (id) => api.get(`/recharge/${id}`),
  getRechargeStatus: (orderNo) => api.get(`/recharge/status/${orderNo}`),
  cancelRecharge: (id) => api.post(`/recharge/${id}/cancel`)
}

export const subscriptionAPI = {
  getUserSubscription: () => api.get('/subscriptions/user-subscription'),
  resetSubscription: () => api.post('/subscriptions/reset-subscription'),
  sendSubscriptionEmail: () => api.post('/subscriptions/send-subscription-email'),
  getDevices: () => api.get('/subscriptions/devices'),
  removeDevice: (id) => api.delete(`/subscriptions/devices/${id}`),
  getSSRSubscription: (key) => api.get(`/subscriptions/ssr/${key}`),
  getClashSubscription: (key) => api.get(`/subscriptions/clash/${key}`),
  convertToBalance: () => api.post('/subscriptions/convert-to-balance')
}

export const packageAPI = {
  getPackages: (params) => api.get('/packages/', { params }),
  getPackage: (id) => api.get(`/packages/${id}`),
  createPackage: (data) => api.post('/packages/', data),
  updatePackage: (id, data) => api.put(`/packages/${id}`, data),
  deletePackage: (id) => api.delete(`/packages/${id}`)
}

export const orderAPI = {
  upgradeDevices: (data) => api.post('/orders/upgrade-devices', data),
  createOrder: (data) => api.post('/orders/', data),
  getOrderList: (params) => api.get('/orders', { params }), // 订单列表（兼容别名）
  getUserOrders: (params) => api.get('/orders/', { params }),
  getOrderDetail: (id) => api.get(`/orders/${id}`), // 订单详情
  getOrderStatus: (orderNo) => api.get(`/orders/${orderNo}/status`),
  cancelOrder: (orderNo) => api.post(`/orders/${orderNo}/cancel`),
  getPackages: () => api.get('/packages/')
}

export const nodeAPI = {
  getNodes: () => api.get('/nodes/'),
  getNode: (id) => api.get(`/nodes/${id}`),
  testNode: (id) => api.post(`/nodes/${id}/test`),
  batchTestNodes: (data) => api.post('/nodes/batch-test', data),
  importFromClash: (config) => api.post('/nodes/import-from-clash', { clash_config: config }),
  getNodesStats: () => api.get('/admin/nodes/stats'),
  getAdminNodes: (params) => api.get('/admin/nodes', { params }),
  getAdminNode: (id) => api.get(`/admin/nodes/${id}`),
  createNode: (data) => api.post('/admin/nodes', data),
  importNodeLinks: (links) => api.post('/admin/nodes/import-links', { links }),
  updateNode: (id, data) => api.put(`/admin/nodes/${id}`, data),
  deleteNode: (id) => api.delete(`/admin/nodes/${id}`),
  testNode: (id) => api.post(`/admin/nodes/${id}/test`),
  batchTestNodes: (nodeIds) => api.post('/admin/nodes/batch-test', { node_ids: nodeIds }),
  batchDeleteNodes: (nodeIds) => api.post('/admin/nodes/batch-delete', { node_ids: nodeIds })
}

export const adminAPI = {
  getDashboard: () => api.get('/admin/dashboard'),
  getStats: () => api.get('/admin/stats'),
  getUsers: (params) => api.get('/admin/users', { params }),
  getUserStatistics: () => api.get('/admin/users/statistics'),
  getOrders: (params) => api.get('/admin/orders', { params }),
  createUser: (data) => api.post('/admin/users', data),
  getUser: (id) => api.get(`/admin/users/${id}`),
  updateUser: (id, data) => api.put(`/admin/users/${id}`, data),
  deleteUser: (id) => api.delete(`/admin/users/${id}`),
  loginAsUser: (id) => api.post(`/admin/users/${id}/login-as`),
  getAbnormalUsers: (params) => api.get('/admin/users/abnormal', { params }),
  getUserDetails: (id) => api.get(`/admin/users/${id}/details`),
  updateUserStatus: (id, status) => api.put(`/admin/users/${id}/status`, { status }),
  resetUserPassword: (id, password) => api.post(`/admin/users/${id}/reset-password`, { password }),
  unlockUserLogin: (id) => api.post(`/admin/users/${id}/unlock-login`),
  batchDeleteUsers: (ids) => api.post('/admin/users/batch-delete', { user_ids: ids }),
  batchEnableUsers: (ids) => api.post('/admin/users/batch-enable', { user_ids: ids }),
  batchDisableUsers: (ids) => api.post('/admin/users/batch-disable', { user_ids: ids }),
  batchVerifyUsers: (ids) => api.post('/admin/users/batch-verify', { user_ids: ids }),
  sendUserSubEmail: (id) => api.post(`/admin/users/${id}/send-subscription-email`),
  batchSendSubEmail: (ids) => api.post('/admin/users/batch-send-subscription-email', { user_ids: ids }),
  getExpiringUsers: (params) => api.get('/admin/users/expiring', { params }),
  batchSendExpireReminder: (ids) => api.post('/admin/users/batch-expire-reminder', { user_ids: ids }),
  getExpiringSubscriptions: (params) => api.get('/admin/subscriptions/expiring', { params }),
  getSubscriptions: (params) => api.get('/admin/subscriptions', { params }),
  createSubscription: (data) => api.post('/admin/subscriptions', data),
  updateSubscription: (id, data) => api.put(`/admin/subscriptions/${id}`, data),
  resetSubscription: (id) => api.post(`/admin/subscriptions/${id}/reset`),
  extendSubscription: (id, days) => api.post(`/admin/subscriptions/${id}/extend`, { days }),
  resetUserSubscription: (id) => api.post(`/admin/subscriptions/user/${id}/reset-all`),
  sendSubEmail: (id) => api.post(`/admin/subscriptions/user/${id}/send-email`),
  sendSubscriptionEmail: (id) => api.post(`/admin/subscriptions/user/${id}/send-email`),
  batchClearDevices: (data) => api.post('/admin/subscriptions/batch-clear-devices', data),
  exportSubscriptions: () => api.get('/admin/subscriptions/export', { responseType: 'blob' }),
  getAppleStats: () => api.get('/admin/subscriptions/apple-stats'),
  getOnlineStats: () => api.get('/admin/subscriptions/online-stats'),
  clearUserDevices: (id) => api.delete(`/admin/subscriptions/user/${id}/delete-all`),
  batchDeleteSubscriptions: (ids) => api.post('/admin/subscriptions/batch-delete', { subscription_ids: ids }),
  batchEnableSubscriptions: (ids) => api.post('/admin/subscriptions/batch-enable', { subscription_ids: ids }),
  batchDisableSubscriptions: (ids) => api.post('/admin/subscriptions/batch-disable', { subscription_ids: ids }),
  batchResetSubscriptions: (ids) => api.post('/admin/subscriptions/batch-reset', { subscription_ids: ids }),
  batchSendAdminSubEmail: (ids) => api.post('/admin/subscriptions/batch-send-email', { subscription_ids: ids }),
  updateOrder: (id, data) => api.put(`/admin/orders/${id}`, data),
  getPackages: (params) => api.get('/admin/packages', { params }),
  createPackage: (data) => api.post('/admin/packages', data),
  updatePackage: (id, data) => api.put(`/admin/packages/${id}`, data),
  deletePackage: (id) => api.delete(`/admin/packages/${id}`),
  getEmailQueue: (params) => api.get('/admin/email-queue', { params }),
  resendEmail: (id) => api.post(`/admin/email-queue/${id}/resend`),
  getEmailDetail: (id) => api.get(`/admin/email-queue/${id}`),
  retryEmail: (id) => api.post(`/admin/email-queue/${id}/retry`),
  deleteEmailFromQueue: (id) => api.delete(`/admin/email-queue/${id}`),
  clearEmailQueue: (status) => api.post(`/admin/email-queue/clear${status ? `?status=${status}` : ''}`),
  getEmailQueueStatistics: () => api.get('/admin/email-queue/statistics'),
  getProfile: () => api.get('/admin/profile'),
  updateProfile: (data) => api.put('/admin/profile', data),
  changePassword: (data) => api.post('/admin/change-password', data),
  getLoginHistory: () => api.get('/admin/login-history'),
  getSecuritySettings: () => api.get('/admin/security-settings'),
  updateSecuritySettings: (data) => api.put('/admin/security-settings', data),
  getSystemLogs: (params) => api.get('/admin/system-logs', { params }),
  getLogsStats: () => api.get('/admin/logs-stats'),
  exportLogs: (params) => api.get('/admin/export-logs', { params, responseType: 'blob' }),
  clearLogs: () => api.post('/admin/clear-logs'),
  getUserDevices: (id) => api.get(`/admin/users/${id}/devices`),
  getSubscriptionDevices: (id) => api.get(`/admin/subscriptions/${id}/devices`),
  getAdminRechargeRecords: (params) => api.get('/recharge/admin', { params }),
  getDeviceDetail: (id) => api.get(`/admin/devices/devices/${id}`),
  updateDeviceStatus: (id, data) => api.put(`/admin/devices/devices/${id}`, data),
  removeDevice: (id) => api.delete(`/admin/devices/${id}`),
  deleteUserDevice: (userId, deviceId) => api.delete(`/admin/users/${userId}/devices/${deviceId}`),
  getAdminNodes: (params) => api.get('/admin/nodes', { params }),
  getAdminNode: (id) => api.get(`/admin/nodes/${id}`),
  createNode: (data) => api.post('/admin/nodes', data),
  importNodeLinks: (links) => api.post('/admin/nodes/import-links', { links }),
  updateNode: (id, data) => api.put(`/admin/nodes/${id}`, data),
  deleteNode: (id) => api.delete(`/admin/nodes/${id}`),
  testNode: (id) => api.post(`/admin/nodes/${id}/test`),
  batchTestNodes: (nodeIds) => api.post('/admin/nodes/batch-test', { node_ids: nodeIds }),
  batchDeleteNodes: (nodeIds) => api.post('/admin/nodes/batch-delete', { node_ids: nodeIds }),
  getNodesStats: () => api.get('/admin/nodes/stats'),
  // 专线节点管理
  getCustomNodes: (params) => api.get('/admin/custom-nodes', { params }),
  createCustomNode: (data) => api.post('/admin/custom-nodes', data),
  importCustomNodeLinks: (links) => api.post('/admin/custom-nodes/import-links', { links }),
  updateCustomNode: (id, data) => api.put(`/admin/custom-nodes/${id}`, data),
  deleteCustomNode: (id) => api.delete(`/admin/custom-nodes/${id}`),
  batchDeleteCustomNodes: (nodeIds) => api.post('/admin/custom-nodes/batch-delete', { node_ids: nodeIds }),
  batchAssignCustomNodes: (nodeIds, userIds, extraData = {}) => api.post('/admin/custom-nodes/batch-assign', { node_ids: nodeIds, user_ids: userIds, ...extraData }),
  getCustomNodeUsers: (id) => api.get(`/admin/custom-nodes/${id}/users`),
  testCustomNode: (id) => api.post(`/admin/custom-nodes/${id}/test`),
  batchTestCustomNodes: (nodeIds) => api.post('/admin/custom-nodes/batch-test', { node_ids: nodeIds }),
  getCustomNodeLink: (id) => api.get(`/admin/custom-nodes/${id}/link`),
  // 用户专线节点分配
  getUserCustomNodes: (userId) => api.get(`/admin/users/${userId}/custom-nodes`),
  assignCustomNodeToUser: (userId, customNodeId, extraData = {}) => api.post(`/admin/users/${userId}/custom-nodes`, { custom_node_id: customNodeId, ...extraData }),
  unassignCustomNodeFromUser: (userId, nodeId) => api.delete(`/admin/users/${userId}/custom-nodes/${nodeId}`),
  // 管理员通知设置
  updateAdminNotificationSettings: (data) => api.put('/admin/settings/admin-notification', data),
  testAdminEmailNotification: () => api.post('/admin/settings/admin-notification/test/email'),
  testAdminTelegramNotification: () => api.post('/admin/settings/admin-notification/test/telegram'),
  testAdminBarkNotification: () => api.post('/admin/settings/admin-notification/test/bark'),
}


export const configAPI = {
  getEmailConfig: () => api.get('/admin/email-config'),
  saveEmailConfig: (data) => api.post('/admin/email-config', data)
}

export const statisticsAPI = {
  getStatistics: () => api.get('/admin/statistics'),
  getUserTrend: () => api.get('/admin/statistics/user-trend'),
  getRevenueTrend: () => api.get('/admin/statistics/revenue-trend'),
  getUserStatistics: (params) => api.get('/admin/statistics/users', { params }),
  getSubscriptionStatistics: () => api.get('/admin/statistics/subscriptions'),
  getOrderStatistics: (params) => api.get('/admin/statistics/orders', { params }),
  getStatisticsOverview: () => api.get('/admin/statistics/overview'),
  exportStatistics: (type, format) => api.get('/admin/statistics/export', { params: { type, format } }),
  getRegionStats: () => api.get('/admin/statistics/regions')
}

export const paymentAPI = {
  getPaymentMethods: () => api.get('/payment-methods/active'),
  createPayment: (data) => api.post('/payment/', data),
  getPaymentStatus: (id) => api.get(`/payment/status/${id}`),
  getPaymentConfigs: (params) => api.get('/payment-config/', { params }),
  createPaymentConfig: (data) => api.post('/payment-config/', data),
  updatePaymentConfig: (id, data) => api.put(`/payment-config/${id}`, data),
  deletePaymentConfig: (id) => api.delete(`/payment-config/${id}`),
  exportPaymentConfigs: (params) => api.get('/payment-config/export', { params, responseType: 'blob' }),
  getPaymentConfigStats: () => api.get('/payment-config/stats/summary'),
  bulkEnablePaymentConfigs: (ids) => api.post('/payment-config/bulk-enable', ids),
  bulkDisablePaymentConfigs: (ids) => api.post('/payment-config/bulk-disable', ids),
  bulkDeletePaymentConfigs: (ids) => api.post('/payment-config/bulk-delete', ids),
  getPaymentTransactions: (params) => api.get('/admin/payment-transactions', { params }),
  getPaymentTransactionDetail: (id) => api.get(`/admin/payment-transactions/${id}`),
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
  getSystemSettings: () => api.get('/admin/settings'),
  updateSystemSettings: (data) => api.put('/admin/settings', data),
  getConfigsByCategory: (params) => api.get('/admin/configs', { params }),
  createConfig: (data) => api.post('/admin/configs', data),
  updateConfig: (key, data) => api.put(`/admin/configs/${key}`, data),
  deleteConfig: (key) => api.delete(`/admin/configs/${key}`),
  initializeConfigs: () => api.post('/admin/configs/initialize'),
  getThemeConfigs: () => api.get('/admin/themes'),
  createThemeConfig: (data) => api.post('/admin/themes', data),
  updateThemeConfig: (id, data) => api.put(`/admin/themes/${id}`, data),
  deleteThemeConfig: (id) => api.delete(`/admin/themes/${id}`)
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
  getTicket: (id) => api.get(`/tickets/${id}`),
  getAdminTicket: (id) => api.get(`/tickets/admin/${id}`),
  addReply: (id, data) => api.post(`/tickets/${id}/replies`, data),
  addRating: (id, data) => api.post(`/tickets/${id}/rating`, data),
  getAllTickets: (params) => api.get('/tickets/admin/all', { params }),
  updateTicket: (id, data) => api.put(`/tickets/admin/${id}`, data),
  getTicketStatistics: () => api.get('/tickets/admin/statistics'),
  getUnreadCount: () => api.get('/tickets/unread-count') // 获取未读回复数量
}

export const couponAPI = {
  getAvailableCoupons: () => api.get('/coupons/available'),
  validateCoupon: (data) => api.post('/coupons/validate', data),
  createCoupon: (data) => api.post('/coupons/admin', data),
  getAllCoupons: (params) => api.get('/coupons/admin', { params }),
  getCoupon: (id) => api.get(`/coupons/admin/${id}`),
  updateCoupon: (id, data) => api.put(`/coupons/admin/${id}`, data),
  deleteCoupon: (id) => api.delete(`/coupons/admin/${id}`),
  getCouponStatistics: () => api.get('/coupons/admin/statistics')
}

export const inviteAPI = {
  generateInviteCode: (data) => api.post('/invites', data),
  getMyInviteCodes: () => api.get('/invites/my-codes'),
  getInviteStats: () => api.get('/invites/stats'),
  getInviteRewardSettings: () => api.get('/invites/reward-settings'),
  validateInviteCode: (code) => api.get(`/invites/validate/${code}`),
  updateInviteCode: (id, data) => api.put(`/invites/${id}`, data),
  deleteInviteCode: (id) => api.delete(`/invites/${id}`),
  getAllInviteCodes: (params) => api.get('/admin/invites', { params }),
  getInviteRelations: (params) => api.get('/admin/invite-relations', { params }),
  getAdminInviteStatistics: () => api.get('/admin/invite-statistics'),
  batchDeleteInviteCodes: (ids) => api.post('/admin/invites/batch-delete', ids),
  batchDeleteInviteRelations: (ids) => api.post('/admin/invite-relations/batch-delete', ids)
}

export const userLevelAPI = {
  getUserLevels: (activeOnly = true) => api.get('/user-levels', { params: { active_only: activeOnly } }),
  getMyLevel: () => api.get('/users/my-level'),
  getAllLevels: (activeOnly, isActive) => {
    const params = {}
    if (isActive !== undefined) params.is_active = isActive
    else if (activeOnly !== undefined) params.active_only = activeOnly
    return api.get('/admin/user-levels', { params })
  },
  getLevelDetail: (id) => api.get(`/admin/user-levels/${id}`),
  createLevel: (data) => api.post('/admin/user-levels', data),
  updateLevel: (id, data) => api.put(`/admin/user-levels/${id}`, data),
  deleteLevel: (id) => api.delete(`/admin/user-levels/${id}`),
  upgradeUsers: (id, userIds) => api.post(`/admin/user-levels/${id}/upgrade-users`, userIds)
}

export default api
