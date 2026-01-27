// 用户相关API统一管理
import { api } from '@/utils/api'

// 用户信息相关
export const getUserInfo = () => api.get('/user/info')
export const getUserStatistics = () => api.get('/user/stat')
export const getUserDashboard = () => api.get('/user/dashboard') // 聚合接口

// 订阅相关
export const getUserSubscription = () => api.get('/subscriptions/current')
export const getSubscriptionList = () => api.get('/subscriptions')

// 订单相关
export const getOrderList = (params) => api.get('/orders', { params })
export const getOrderDetail = (id) => api.get(`/orders/${id}`)
export const createOrder = (data) => api.post('/orders', data)

// 设备相关
export const getDeviceList = () => api.get('/devices')
export const addDevice = (data) => api.post('/devices', data)
export const deleteDevice = (id) => api.delete(`/devices/${id}`)

// 充值相关
export const createRecharge = (data) => api.post('/recharge', data)
export const getRechargeRecords = () => api.get('/recharge/records')

// 设置相关
export const updateProfile = (data) => api.put('/users/profile', data)
export const changePassword = (data) => api.post('/users/change-password', data)
