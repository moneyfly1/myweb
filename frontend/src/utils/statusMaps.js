/**
 * 统一的状态映射工具
 * 所有状态相关的映射都应该使用这个文件
 * 页面中禁止再内联状态映射，颜色/文案以本文件为准
 */

import { isExpired } from './date'

// 用户状态映射
export const USER_STATUS_MAP = {
  active: { text: '活跃', type: 'success' },
  inactive: { text: '待激活', type: 'info' },
  disabled: { text: '禁用', type: 'danger' },
  pending: { text: '待激活', type: 'warning' },
  device_overlimit: { text: '设备超限', type: 'warning' }
}

// 订阅状态映射
export const SUBSCRIPTION_STATUS_MAP = {
  active: { text: '活跃', type: 'success' },
  expired: { text: '已过期', type: 'danger' },
  pending: { text: '待激活', type: 'info' },
  disabled: { text: '已禁用', type: 'danger' },
  inactive: { text: '未激活', type: 'info' },
  paused: { text: '已暂停', type: 'warning' }
}

// 订单状态映射
export const ORDER_STATUS_MAP = {
  pending: { text: '待支付', type: 'warning' },
  paid: { text: '已支付', type: 'success' },
  completed: { text: '已完成', type: 'success' },
  refunding: { text: '退款中', type: 'warning' },
  cancelled: { text: '已取消', type: 'info' },
  failed: { text: '支付失败', type: 'danger' },
  expired: { text: '已过期', type: 'info' },
  refunded: { text: '已退款', type: 'info' }
}

// 工单状态映射
export const TICKET_STATUS_MAP = {
  pending: { text: '待处理', type: 'warning' },
  processing: { text: '处理中', type: 'primary' },
  resolved: { text: '已解决', type: 'success' },
  closed: { text: '已关闭', type: 'info' },
  cancelled: { text: '已取消', type: 'danger' }
}

// 工单类型映射
export const TICKET_TYPE_MAP = {
  technical: { text: '技术问题', type: 'primary' },
  billing: { text: '账单问题', type: 'warning' },
  account: { text: '账户问题', type: 'danger' },
  other: { text: '其他', type: 'info' }
}

// 工单优先级映射
export const TICKET_PRIORITY_MAP = {
  low: { text: '低', type: 'info' },
  normal: { text: '普通', type: 'info' },
  high: { text: '高', type: 'warning' },
  urgent: { text: '紧急', type: 'danger' }
}

// 节点状态映射
export const NODE_STATUS_MAP = {
  online: { text: '在线', type: 'success' },
  offline: { text: '离线', type: 'danger' },
  maintenance: { text: '维护中', type: 'warning' },
  unknown: { text: '未知', type: 'info' },
  timeout: { text: '超时', type: 'warning' },
  inactive: { text: '未激活', type: 'info' },
  pending: { text: '安装中', type: 'warning' },
  expired: { text: '已过期', type: 'info' },
  canceled: { text: '已取消', type: 'info' }
}

// 自定义节点状态映射
export const CUSTOM_NODE_STATUS_MAP = {
  active: { text: '活跃', type: 'success' },
  inactive: { text: '非活跃', type: 'info' },
  error: { text: '错误', type: 'danger' }
}

// 邮件状态映射
export const EMAIL_STATUS_MAP = {
  pending: { text: '待发送', type: 'warning' },
  sending: { text: '发送中', type: 'info' },
  sent: { text: '已发送', type: 'success' },
  failed: { text: '发送失败', type: 'danger' },
  bounced: { text: '被退回', type: 'info' },
  cancelled: { text: '已取消', type: 'info' }
}

// 邮件类型映射
export const EMAIL_TYPE_MAP = {
  verification: '邮箱验证',
  password_reset: '密码重置',
  password_changed: '密码已更改',
  welcome: '欢迎注册',
  subscription: '订阅配置',
  subscription_sent: '订阅发送',
  subscription_reset: '订阅重置',
  subscription_created: '订阅创建',
  subscription_expired: '订阅过期',
  expiration_reminder: '到期提醒',
  expiry_reminder: '到期提醒(旧)',
  renewal_confirmation: '续费确认',
  order_created: '订单创建',
  order_paid: '订单支付',
  payment_success: '支付成功',
  recharge_success: '充值到账',
  admin_notification: '管理员通知',
  admin_manual: '管理员手动邮件',
  account_deletion: '账号删除',
  user_created: '账户创建',
  abnormal_login_alert: '异常登录提醒',
  ticket_created: '工单创建',
  ticket_reply: '工单回复',
  ticket_replied: '工单回复',
  marketing: '营销推广'
}

// 套餐状态映射
export const PACKAGE_STATUS_MAP = {
  active: { text: '启用', type: 'success' },
  inactive: { text: '禁用', type: 'danger' }
}

// 优惠券状态映射
export const COUPON_STATUS_MAP = {
  active: { text: '有效', type: 'success' },
  inactive: { text: '无效', type: 'info' },
  expired: { text: '已过期', type: 'danger' }
}

// 佣金结算状态映射
export const COMMISSION_STATUS_MAP = {
  pending: { text: '待结算', type: 'warning' },
  paid: { text: '已结算', type: 'success' },
  cancelled: { text: '已取消', type: 'info' }
}

// 异常用户类型映射
export const ABNORMAL_TYPE_MAP = {
  disabled: { text: '账户禁用', type: 'danger' },
  device_over_limit: { text: '设备超限', type: 'danger' },
  frequent_reset: { text: '频繁重置', type: 'warning' },
  frequent_subscription: { text: '频繁订阅', type: 'danger' },
  inactive: { text: '长期未登录', type: 'info' },
  login_failed: { text: '登录失败', type: 'warning' },
  multi_ip: { text: '多IP访问', type: 'danger' },
  multi_location: { text: '多地区访问', type: 'warning' },
  multiple_abnormal: { text: '多重异常', type: 'error' },
  unverified: { text: '未验证邮箱', type: 'warning' },
  unknown: { text: '未知异常', type: 'info' }
}

// 支付方式映射（文案）
export const PAYMENT_METHOD_MAP = {
  alipay: '支付宝',
  wechat: '微信支付',
  wxpay: '微信支付',
  balance: '余额支付',
  manual: '手动充值',
  card: '银行卡',
  paypal: 'PayPal',
  other: '其他',
  yipay: '易支付',
  yipay_alipay: '易支付-支付宝',
  yipay_wxpay: '易支付-微信',
  yipay_qqpay: '易支付-QQ钱包',
  codepay: '码支付',
  codepay_alipay: '码支付-支付宝',
  codepay_wxpay: '码支付-微信',
  // 兼容后端直接回传中文
  '支付宝': '支付宝',
  '微信支付': '微信支付',
  '余额支付': '余额支付'
}

// 支付方式映射（el-tag 类型）
export const PAYMENT_METHOD_TYPE_MAP = {
  alipay: 'primary',
  wechat: 'success',
  wxpay: 'success',
  balance: 'warning',
  yipay: 'primary',
  yipay_alipay: 'primary',
  yipay_wxpay: 'success',
  yipay_qqpay: 'info',
  codepay: 'primary',
  codepay_alipay: 'primary',
  codepay_wxpay: 'success'
}

/**
 * 获取状态文本
 * 兼容两种映射结构：{ text, type } 对象映射 或 纯字符串映射（如 PAYMENT_METHOD_MAP）
 * @param {string} status - 状态值
 * @param {Object} map - 状态映射表
 * @returns {string} 状态文本
 */
export function getStatusText(status, map) {
  if (!status) return '-'
  const entry = map[status]
  if (typeof entry === 'string') return entry
  return entry?.text || status
}

/**
 * 获取状态类型（用于 el-tag 的 type 属性）
 * @param {string} status - 状态值
 * @param {Object} map - 状态映射表
 * @returns {string} 状态类型
 */
export function getStatusType(status, map) {
  if (!status) return 'info'
  const entry = map[status]
  if (typeof entry === 'string') return 'info'
  return entry?.type || 'info'
}

/**
 * 获取状态配置（包含文本和类型）
 * @param {string} status - 状态值
 * @param {Object} map - 状态映射表
 * @returns {Object} 状态配置 { text, type }
 */
export function getStatusConfig(status, map) {
  if (!status) return { text: '-', type: 'info' }
  const entry = map[status]
  if (typeof entry === 'string') return { text: entry, type: 'info' }
  return entry || { text: status, type: 'info' }
}

/**
 * 获取邮件类型文本
 * @param {string} type - 邮件类型
 * @returns {string} 邮件类型中文名称
 */
export function getEmailTypeText(type) {
  if (!type) return '-'
  return EMAIL_TYPE_MAP[type] || type
}

/**
 * 获取用户状态文本
 * @param {string} status - 状态值
 * @returns {string} 状态文本
 */
export function getUserStatusText(status) {
  return getStatusText(status, USER_STATUS_MAP)
}

/**
 * 获取用户状态类型
 * @param {string} status - 状态值
 * @returns {string} 状态类型
 */
export function getUserStatusType(status) {
  return getStatusType(status, USER_STATUS_MAP)
}

/**
 * 获取订阅状态文本
 * @param {string} status - 状态值
 * @returns {string} 状态文本
 */
export function getSubscriptionStatusText(status) {
  return getStatusText(status, SUBSCRIPTION_STATUS_MAP)
}

/**
 * 获取订阅状态类型
 * @param {string} status - 状态值
 * @returns {string} 状态类型
 */
export function getSubscriptionStatusType(status) {
  return getStatusType(status, SUBSCRIPTION_STATUS_MAP)
}

/**
 * 获取订阅状态（日期驱动，用于用户端订阅卡片）
 * 优先级：expireTime 存在时按是否过期判断，否则回退到 status 字段
 * @param {Object} sub - 订阅对象（含 expire_time / status 字段）
 * @param {Object} [options]
 * @param {string} [options.expireTime] - 可选的过期时间覆盖（默认取 sub.expire_time）
 * @returns {{ text: string, type: string }}
 */
export function getSubscriptionStatus(sub, { expireTime } = {}) {
  if (!sub) return { text: '未激活', type: 'info' }
  const exp = expireTime ?? sub.expire_time
  if (exp) {
    return isExpired(exp) ? { text: '已过期', type: 'danger' } : { text: '正常', type: 'success' }
  }
  if (sub.status === 'active') return { text: '正常', type: 'success' }
  if (sub.status === 'expired') return { text: '已过期', type: 'danger' }
  return { text: '未激活', type: 'info' }
}

/**
 * 获取订单状态文本
 * @param {string} status - 状态值
 * @returns {string} 状态文本
 */
export function getOrderStatusText(status) {
  return getStatusText(status, ORDER_STATUS_MAP)
}

/**
 * 获取订单状态类型
 * @param {string} status - 状态值
 * @returns {string} 状态类型
 */
export function getOrderStatusType(status) {
  return getStatusType(status, ORDER_STATUS_MAP)
}

/**
 * 获取工单状态文本
 * @param {string} status - 状态值
 * @returns {string} 状态文本
 */
export function getTicketStatusText(status) {
  return getStatusText(status, TICKET_STATUS_MAP)
}

/**
 * 获取工单状态类型
 * @param {string} status - 状态值
 * @returns {string} 状态类型
 */
export function getTicketStatusType(status) {
  return getStatusType(status, TICKET_STATUS_MAP)
}

/**
 * 获取工单类型文本
 * @param {string} type - 工单类型
 * @returns {string} 工单类型文本
 */
export function getTicketTypeText(type) {
  return getStatusText(type, TICKET_TYPE_MAP)
}

/**
 * 获取工单类型（el-tag type）
 * @param {string} type - 工单类型
 * @returns {string} 工单类型
 */
export function getTicketTypeType(type) {
  return getStatusType(type, TICKET_TYPE_MAP)
}

/**
 * 获取工单优先级文本
 * @param {string} priority - 优先级值
 * @returns {string} 优先级文本
 */
export function getTicketPriorityText(priority) {
  return getStatusText(priority, TICKET_PRIORITY_MAP)
}

/**
 * 获取工单优先级类型
 * @param {string} priority - 优先级值
 * @returns {string} 优先级类型
 */
export function getTicketPriorityType(priority) {
  return getStatusType(priority, TICKET_PRIORITY_MAP)
}

/**
 * 获取节点状态文本
 * @param {string} status - 状态值
 * @returns {string} 状态文本
 */
export function getNodeStatusText(status) {
  return getStatusText(status, NODE_STATUS_MAP)
}

/**
 * 获取节点状态类型
 * @param {string} status - 状态值
 * @returns {string} 状态类型
 */
export function getNodeStatusType(status) {
  return getStatusType(status, NODE_STATUS_MAP)
}

/**
 * 获取自定义节点状态文本
 * @param {string} status - 状态值
 * @returns {string} 状态文本
 */
export function getCustomNodeStatusText(status) {
  return getStatusText(status, CUSTOM_NODE_STATUS_MAP)
}

/**
 * 获取自定义节点状态类型
 * @param {string} status - 状态值
 * @returns {string} 状态类型
 */
export function getCustomNodeStatusType(status) {
  return getStatusType(status, CUSTOM_NODE_STATUS_MAP)
}

/**
 * 获取邮件状态文本
 * @param {string} status - 状态值
 * @returns {string} 状态文本
 */
export function getEmailStatusText(status) {
  return getStatusText(status, EMAIL_STATUS_MAP)
}

/**
 * 获取邮件状态类型
 * @param {string} status - 状态值
 * @returns {string} 状态类型
 */
export function getEmailStatusType(status) {
  return getStatusType(status, EMAIL_STATUS_MAP)
}

/**
 * 获取佣金结算状态文本
 * @param {string} status - 状态值
 * @returns {string} 状态文本
 */
export function getCommissionStatusText(status) {
  return getStatusText(status, COMMISSION_STATUS_MAP)
}

/**
 * 获取佣金结算状态类型
 * @param {string} status - 状态值
 * @returns {string} 状态类型
 */
export function getCommissionStatusType(status) {
  return getStatusType(status, COMMISSION_STATUS_MAP)
}

/**
 * 获取优惠券状态文本
 * @param {string} status - 状态值
 * @returns {string} 状态文本
 */
export function getCouponStatusText(status) {
  return getStatusText(status, COUPON_STATUS_MAP)
}

/**
 * 获取优惠券状态类型
 * @param {string} status - 状态值
 * @returns {string} 状态类型
 */
export function getCouponStatusType(status) {
  return getStatusType(status, COUPON_STATUS_MAP)
}

/**
 * 获取异常用户类型文本
 * @param {string} type - 异常类型
 * @returns {string} 异常类型文本
 */
export function getAbnormalTypeText(type) {
  return getStatusText(type, ABNORMAL_TYPE_MAP)
}

/**
 * 获取异常用户类型（el-tag type）
 * @param {string} type - 异常类型
 * @returns {string} 异常类型
 */
export function getAbnormalTypeType(type) {
  return getStatusType(type, ABNORMAL_TYPE_MAP)
}

/**
 * 获取支付方式文本
 * @param {string} method - 支付方式
 * @returns {string} 支付方式中文名称
 */
export function getPaymentMethodText(method) {
  if (!method) return '未知'
  return PAYMENT_METHOD_MAP[method] || method
}

/**
 * 获取支付方式（el-tag type）
 * PAYMENT_METHOD_TYPE_MAP 的值本身就是 el-tag 类型字符串，
 * 因此直接查表，不走 getStatusType 的字符串容错
 * @param {string} method - 支付方式
 * @returns {string} 支付方式类型
 */
export function getPaymentMethodType(method) {
  if (!method) return 'info'
  return PAYMENT_METHOD_TYPE_MAP[method] || 'info'
}
