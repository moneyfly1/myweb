import {
  Box,
  Cellphone,
  Connection,
  Iphone,
  Monitor,
  QuestionFilled,
  VideoCamera
} from '@element-plus/icons-vue'

/**
 * 统一的设备类型映射（text / color / icon）
 * 来源：Devices.vue 本地 DEVICE_TYPE_MAP（getDeviceIcon/getDeviceTypeName/getDeviceTypeColor）
 */
export const DEVICE_TYPE_MAP = {
  mobile: { text: '手机', color: 'primary', icon: Cellphone },
  desktop: { text: '电脑', color: 'success', icon: Monitor },
  tablet: { text: '平板', color: 'warning', icon: Iphone },
  router: { text: '路由器', color: '', icon: Connection },
  tv_box: { text: '电视盒子', color: 'danger', icon: VideoCamera },
  server: { text: '服务器', color: 'info', icon: Box },
  unknown: { text: '未知', color: 'info', icon: QuestionFilled }
}

/** 设备类型 → 显示名称（未知类型返回「未知」） */
export function getDeviceTypeName(deviceType) {
  return DEVICE_TYPE_MAP[deviceType]?.text || '未知'
}

/** 设备类型 → el-tag type（与 Devices.vue 原实现一致：空字符串回退到 unknown 的 info） */
export function getDeviceTypeColor(deviceType) {
  return DEVICE_TYPE_MAP[deviceType]?.color || DEVICE_TYPE_MAP.unknown.color
}

/** 设备类型 → 图标组件（未知类型返回 QuestionFilled） */
export function getDeviceTypeIcon(deviceType) {
  return DEVICE_TYPE_MAP[deviceType]?.icon || DEVICE_TYPE_MAP.unknown.icon
}

/**
 * 根据 User-Agent 推断设备类型名称
 * 来源：Profile.vue / LoginHistory.vue 的 getDeviceInfo（逐字提取）
 * 注意：Mobile 判断使用大小写敏感的 'Mobile' 子串，与来源实现保持一致
 */
export function getDeviceTypeFromUA(userAgent) {
  if (!userAgent) return '未知设备'
  if (userAgent.includes('Mobile')) {
    return '移动设备'
  } else if (userAgent.includes('Windows')) {
    return 'Windows设备'
  } else if (userAgent.includes('Mac')) {
    return 'Mac设备'
  } else if (userAgent.includes('Linux')) {
    return 'Linux设备'
  } else {
    return '其他设备'
  }
}

/**
 * 文本截断（超过 maxLength 加省略号）
 * 来源：Devices.vue / admin/Subscriptions.vue 的 truncateUserAgent
 * @param {string} text
 * @param {number} maxLength 默认 50
 * @param {string} empty 空值兜底，默认「未知」
 */
export function truncateText(text, maxLength = 50, empty = '未知') {
  if (!text) return empty
  return text.length > maxLength ? text.substring(0, maxLength) + '...' : text
}

export default {
  DEVICE_TYPE_MAP,
  getDeviceTypeName,
  getDeviceTypeColor,
  getDeviceTypeIcon,
  getDeviceTypeFromUA,
  truncateText
}
