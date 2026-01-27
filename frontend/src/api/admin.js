// 管理员后台API统一管理
import { api } from '@/utils/api'

// 仪表盘相关
export const getDashboard = () => api.get('/admin/dashboard')
export const getStats = () => api.get('/admin/stats')
export const getRecentUsers = () => api.get('/admin/users/recent')
export const getRecentOrders = () => api.get('/admin/orders/recent')
export const getAbnormalUsers = (params) => api.get('/admin/users/abnormal', { params })
export const getExpiringSubscriptions = (params) => api.get('/admin/subscriptions/expiring', { params })

// 用户管理
export const getUsers = (params) => api.get('/admin/users', { params })
export const getUser = (id) => api.get(`/admin/users/${id}`)
export const createUser = (data) => api.post('/admin/users', data)
export const updateUser = (id, data) => api.put(`/admin/users/${id}`, data)
export const deleteUser = (id) => api.delete(`/admin/users/${id}`)
export const batchDeleteUsers = (userIds) => api.post('/admin/users/batch-delete', { user_ids: userIds })
export const batchEnableUsers = (userIds) => api.post('/admin/users/batch-enable', { user_ids: userIds })
export const batchDisableUsers = (userIds) => api.post('/admin/users/batch-disable', { user_ids: userIds })
export const getUserDetails = (id) => api.get(`/admin/users/${id}/details`)
export const getUserStatistics = () => api.get('/admin/users/statistics')
export const loginAsUser = (id) => api.post(`/admin/users/${id}/login-as`)

// 订单管理
export const getOrders = (params) => api.get('/admin/orders', { params })
export const getOrder = (id) => api.get(`/admin/orders/id/${id}`)
export const updateOrder = (id, data) => api.put(`/admin/orders/${id}`, data)
export const deleteOrder = (id) => api.delete(`/admin/orders/${id}`)
export const batchDeleteOrders = (orderIds) => api.post('/admin/orders/batch-delete', { order_ids: orderIds })
export const batchMarkPaid = (orderIds) => api.post('/admin/orders/bulk-mark-paid', { order_ids: orderIds })
export const batchCancelOrders = (orderIds) => api.post('/admin/orders/bulk-cancel', { order_ids: orderIds })
export const exportOrders = (params) => api.get('/admin/orders/export', { params, responseType: 'blob' })
export const getOrderStatistics = () => api.get('/admin/orders/statistics')

// 订阅管理
export const getSubscriptions = (params) => api.get('/admin/subscriptions', { params })
export const createSubscription = (data) => api.post('/admin/subscriptions', data)
export const updateSubscription = (id, data) => api.put(`/admin/subscriptions/${id}`, data)
export const resetSubscription = (id) => api.post(`/admin/subscriptions/${id}/reset`)
export const batchDeleteSubscriptions = (subscriptionIds) => api.post('/admin/subscriptions/batch-delete', { subscription_ids: subscriptionIds })
export const batchEnableSubscriptions = (subscriptionIds) => api.post('/admin/subscriptions/batch-enable', { subscription_ids: subscriptionIds })
export const batchDisableSubscriptions = (subscriptionIds) => api.post('/admin/subscriptions/batch-disable', { subscription_ids: subscriptionIds })

// 优惠券管理
export const getCoupons = (params) => api.get('/coupons/admin', { params })
export const getCoupon = (id) => api.get(`/coupons/admin/${id}`)
export const createCoupon = (data) => api.post('/coupons/admin', data)
export const updateCoupon = (id, data) => api.put(`/coupons/admin/${id}`, data)
export const deleteCoupon = (id) => api.delete(`/coupons/admin/${id}`)
export const getCouponStatistics = () => api.get('/coupons/admin/statistics')

// 邀请管理
export const getInviteCodes = (params) => api.get('/admin/invites', { params })
export const getInviteRelations = (params) => api.get('/admin/invite-relations', { params })
export const getInviteStatistics = () => api.get('/admin/invite-statistics')
export const batchDeleteInviteCodes = (ids) => api.post('/admin/invites/batch-delete', ids)
export const batchDeleteInviteRelations = (ids) => api.post('/admin/invite-relations/batch-delete', ids)

// 邮件队列
export const getEmailQueue = (params) => api.get('/admin/email-queue', { params })
export const getEmailDetail = (id) => api.get(`/admin/email-queue/${id}`)
export const resendEmail = (id) => api.post(`/admin/email-queue/${id}/resend`)
export const retryEmail = (id) => api.post(`/admin/email-queue/${id}/retry`)
export const deleteEmailFromQueue = (id) => api.delete(`/admin/email-queue/${id}`)
export const clearEmailQueue = (status) => api.post(`/admin/email-queue/clear${status ? `?status=${status}` : ''}`)
export const getEmailQueueStatistics = () => api.get('/admin/email-queue/statistics')

// 系统设置
export const getSystemSettings = () => api.get('/admin/settings')
export const updateSystemSettings = (data) => api.put('/admin/settings', data)
export const updateGeneralSettings = (data) => api.put('/admin/settings/general', data)
export const updateRegistrationSettings = (data) => api.put('/admin/settings/registration', data)
export const updateNotificationSettings = (data) => api.put('/admin/settings/notification', data)
export const updateSecuritySettings = (data) => api.put('/admin/settings/security', data)

// 配置管理
export const getConfigs = (params) => api.get('/admin/configs', { params })
export const getConfig = (key) => api.get(`/admin/configs/${key}`)
export const createConfig = (data) => api.post('/admin/configs', data)
export const updateConfig = (key, data) => api.put(`/admin/configs/${key}`, data)
export const deleteConfig = (key) => api.delete(`/admin/configs/${key}`)

// 系统日志
export const getSystemLogs = (params) => api.get('/admin/system-logs', { params })
export const getLogsStats = () => api.get('/admin/logs-stats')
export const exportLogs = (params) => api.get('/admin/export-logs', { params, responseType: 'blob' })
export const clearLogs = () => api.post('/admin/clear-logs')

// 统计相关
export const getStatistics = () => api.get('/admin/statistics')
export const getUserTrend = () => api.get('/admin/statistics/user-trend')
export const getRevenueTrend = () => api.get('/admin/statistics/revenue-trend')
export const getUserStatistics = (params) => api.get('/admin/statistics/users', { params })
export const getSubscriptionStatistics = () => api.get('/admin/statistics/subscriptions')
export const getOrderStatistics = (params) => api.get('/admin/statistics/orders', { params })
export const getStatisticsOverview = () => api.get('/admin/statistics/overview')
export const exportStatistics = (type, format) => api.get('/admin/statistics/export', { params: { type, format } })
export const getRegionStats = () => api.get('/admin/statistics/regions')
