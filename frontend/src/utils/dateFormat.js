import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'

// 设置默认语言为中文
dayjs.locale('zh-cn')

/**
 * 统一的日期格式化工具
 */

// 常用日期格式
export const DATE_FORMATS = {
  DATE: 'YYYY-MM-DD',
  DATETIME: 'YYYY-MM-DD HH:mm:ss',
  TIME: 'HH:mm:ss',
  DATE_SHORT: 'MM-DD',
  DATETIME_SHORT: 'MM-DD HH:mm',
  RELATIVE: 'relative'
}

/**
 * 格式化日期
 * @param {string|number|Date} date - 日期
 * @param {string} format - 格式，默认 DATE_FORMATS.DATETIME
 * @returns {string} 格式化后的日期字符串
 */
export function formatDate(date, format = DATE_FORMATS.DATETIME) {
  if (!date) return '-'
  
  if (format === DATE_FORMATS.RELATIVE) {
    return formatRelativeTime(date)
  }
  
  return dayjs(date).format(format)
}

/**
 * 格式化相对时间（如：3分钟前）
 * @param {string|number|Date} date - 日期
 * @returns {string} 相对时间字符串
 */
export function formatRelativeTime(date) {
  if (!date) return '-'
  
  const now = dayjs()
  const target = dayjs(date)
  const diff = now.diff(target, 'second')
  
  if (diff < 60) {
    return '刚刚'
  } else if (diff < 3600) {
    return `${Math.floor(diff / 60)}分钟前`
  } else if (diff < 86400) {
    return `${Math.floor(diff / 3600)}小时前`
  } else if (diff < 2592000) {
    return `${Math.floor(diff / 86400)}天前`
  } else if (diff < 31536000) {
    return `${Math.floor(diff / 2592000)}个月前`
  } else {
    return `${Math.floor(diff / 31536000)}年前`
  }
}

/**
 * 格式化金额
 * @param {number|string} amount - 金额
 * @param {number} decimals - 小数位数，默认2位
 * @returns {string} 格式化后的金额字符串
 */
export function formatMoney(amount, decimals = 2) {
  if (amount === null || amount === undefined) return '0.00'
  
  const num = typeof amount === 'string' ? parseFloat(amount) : amount
  return num.toFixed(decimals).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

/**
 * 格式化百分比
 * @param {number} value - 数值（0-1之间）
 * @param {number} decimals - 小数位数，默认1位
 * @returns {string} 格式化后的百分比字符串
 */
export function formatPercent(value, decimals = 1) {
  if (value === null || value === undefined) return '0%'
  return `${(value * 100).toFixed(decimals)}%`
}

/**
 * 格式化文件大小
 * @param {number} bytes - 字节数
 * @returns {string} 格式化后的文件大小
 */
export function formatFileSize(bytes) {
  if (bytes === 0) return '0 B'
  
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
}

/**
 * 格式化数字（添加千分位）
 * @param {number} num - 数字
 * @returns {string} 格式化后的数字字符串
 */
export function formatNumber(num) {
  if (num === null || num === undefined) return '0'
  return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}
