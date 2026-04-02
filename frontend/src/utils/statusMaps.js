/**
 * 统一的状态映射工具
 * 所有状态相关的映射都应该使用这个文件
 */

// 用户状态映射
export const USER_STATUS_MAP = {
  active: { text: '活跃', type: 'success' },
  inactive: { text: '待激活', type: 'info' },
  disabled: { text: '禁用', type: 'danger' },
  device_overlimit: { text: '设备超限', type: 'warning' }
}

// 订阅状态映射
export const SUBSCRIPTION_STATUS_MAP = {
  active: { text: '活跃', type: 'success' },
  expired: { text: '已过期', type: 'danger' },
  pending: { text: '待激活', type: 'info' },
  disabled: { text: '已禁用', type: 'danger' }
}

// 订单状态映射
export const ORDER_STATUS_MAP = {
  pending: { text: '待支付', type: 'warning' },
  paid: { text: '已支付', type: 'success' },
  cancelled: { text: '已取消', type: 'info' },
  failed: { text: '支付失败', type: 'danger' },
  refunded: { text: '已退款', type: 'info' }
}

// 工单状态映射
export const TICKET_STATUS_MAP = {
  pending: { text: '待处理', type: 'warning' },
  processing: { text: '处理中', type: 'primary' },
  resolved: { text: '已解决', type: 'success' },
  closed: { text: '已关闭', type: 'info' }
}

// 工单优先级映射
export const TICKET_PRIORITY_MAP = {
  low: { text: '低', type: 'info' },
  normal: { text: '普通', type: '' },
  high: { text: '高', type: 'warning' },
  urgent: { text: '紧急', type: 'danger' }
}

// 节点状态映射
export const NODE_STATUS_MAP = {
  online: { text: '在线', type: 'success' },
  offline: { text: '离线', type: 'danger' },
  maintenance: { text: '维护中', type: 'warning' },
  unknown: { text: '未知', type: 'info' }
}

// 邮件状态映射
export const EMAIL_STATUS_MAP = {
  pending: { text: '待发送', type: 'warning' },
  sent: { text: '已发送', type: 'success' },
  failed: { text: '发送失败', type: 'danger' },
  bounced: { text: '被退回', type: 'info' }
}

// 套餐状态映射
export const PACKAGE_STATUS_MAP = {
  active: { text: '启用', type: 'success' },
  inactive: { text: '禁用', type: 'danger' }
}

// 支付方式映射
export const PAYMENT_METHOD_MAP = {
  alipay: '支付宝',
  wechat: '微信支付',
  balance: '余额支付',
  manual: '手动充值'
}

/**
 * 获取状态文本
 * @param {string} status - 状态值
 * @param {Object} map - 状态映射表
 * @returns {string} 状态文本
 */
export function getStatusText(status, map) {
  if (!status) return '-'
  return map[status]?.text || status
}

/**
 * 获取状态类型（用于 el-tag 的 type 属性）
 * @param {string} status - 状态值
 * @param {Object} map - 状态映射表
 * @returns {string} 状态类型
 */
export function getStatusType(status, map) {
  if (!status) return 'info'
  return map[status]?.type || 'info'
}

/**
 * 获取状态配置（包含文本和类型）
 * @param {string} status - 状态值
 * @param {Object} map - 状态映射表
 * @returns {Object} 状态配置 { text, type }
 */
export function getStatusConfig(status, map) {
  if (!status) return { text: '-', type: 'info' }
  return map[status] || { text: status, type: 'info' }
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
