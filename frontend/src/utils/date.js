import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import relativeTime from 'dayjs/plugin/relativeTime'
import utc from 'dayjs/plugin/utc'
import timezone from 'dayjs/plugin/timezone'

// 初始化 dayjs 插件和默认设置
dayjs.locale('zh-cn')
dayjs.extend(relativeTime)
dayjs.extend(utc)
dayjs.extend(timezone)

const DEFAULT_TIMEZONE = 'Asia/Shanghai'

/**
 * 设置 dayjs 的默认时区。
 * @param {string} timezone 目标时区，默认为 Asia/Shanghai
 */
export function setTimezone(timezone = DEFAULT_TIMEZONE) {
  dayjs.tz.setDefault(timezone)
}
setTimezone()

/**
 * 创建一个 dayjs 实例，并确保其处于北京时间 (Asia/Shanghai) 时区。
 * @param {string|number|Date|dayjs.Dayjs} date 原始日期时间值。
 * @returns {dayjs.Dayjs} 处于 Asia/Shanghai 时区的 dayjs 实例。
 */
const createShanghaiDayjs = (date) => {
  if (!date) return dayjs()
  
  // 核心处理逻辑：
  // 1. 对于 ISO 字符串 (带 Z 或时区偏移)，dayjs 会正确解析时区。
  // 2. 对于简单的 YYYY-MM-DD HH:mm:ss 字符串，
  //    如果它来自后端且表示 UTC 时间，我们应该使用 dayjs.utc(date) 解析。
  //    为了健壮性，我们假设后端返回的时间（尤其是不带时区信息的）都是 UTC，然后转换。
  // 3. 如果是 Date 对象或其他类型，直接 dayjs(date) 解析。

  let d
  if (typeof date === 'string' && date.includes('Z')) {
    // 明确的 UTC ISO 字符串
    d = dayjs.utc(date)
  } else if (typeof date === 'string' && date.match(/[+-]\d{2}:\d{2}$/)) {
    // 明确带时区偏移的字符串
    d = dayjs(date)
  } else if (typeof date === 'string' && (date.includes('T') || /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(date))) {
    // 假设 ISO 或简单 YYYY-MM-DD HH:mm:ss 字符串是 UTC 时间
    d = dayjs.utc(date)
  } else {
    // Date 对象，时间戳等
    d = dayjs(date)
  }

  // 确保转换为目标时区
  return d.tz(DEFAULT_TIMEZONE)
}

/**
 * 格式化日期时间。
 * @param {string|number|Date|dayjs.Dayjs} date 日期时间值。
 * @param {string} format 格式字符串，默认为 'YYYY-MM-DD HH:mm:ss'。
 * @returns {string} 格式化后的日期时间字符串。
 */
export function formatDateTime(date, format = 'YYYY-MM-DD HH:mm:ss') {
  if (!date) return ''
  
  // 如果输入已经是格式化的字符串，可能是后端已处理过的，直接返回
  if (typeof date === 'string' && /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(date)) {
    return date
  }
  
  return createShanghaiDayjs(date).format(format)
}

/**
 * 格式化日期。
 */
export function formatDate(date, format = 'YYYY-MM-DD') {
  if (!date) return ''
  return createShanghaiDayjs(date).format(format)
}

/**
 * 格式化时间。
 */
export function formatTime(date, format = 'HH:mm:ss') {
  if (!date) return ''
  return createShanghaiDayjs(date).format(format)
}

/**
 * 获取相对时间（如“3天前”）。
 */
export function getRelativeTime(date) {
  if (!date) return ''
  return createShanghaiDayjs(date).fromNow()
}

/**
 * 计算两个日期的时间差。
 */
export function getTimeDiff(date1, date2, unit = 'day') {
  return createShanghaiDayjs(date1).diff(createShanghaiDayjs(date2), unit)
}

/**
 * 检查日期是否已过期（早于当前时间）。
 */
export function isExpired(date) {
  if (!date) return true
  return createShanghaiDayjs(date).isBefore(createShanghaiDayjs())
}

/**
 * 检查日期是否即将过期。
 */
export function isExpiringSoon(date, days = 7) {
  if (!date) return false
  const expiryDate = createShanghaiDayjs(date)
  const now = createShanghaiDayjs()
  const diffDays = expiryDate.diff(now, 'day')
  return diffDays >= 0 && diffDays <= days
}

/**
 * 获取剩余天数。
 */
export function getRemainingDays(date) {
  if (!date) return 0
  const expiryDate = createShanghaiDayjs(date)
  const now = createShanghaiDayjs()
  const diffDays = expiryDate.diff(now, 'day')
  return Math.max(0, diffDays)
}

/**
 * 获取剩余时间（天、时、分、秒）。
 */
export function getRemainingTime(date) {
  if (!date) return { days: 0, hours: 0, minutes: 0, seconds: 0 }
  const expiryDate = createShanghaiDayjs(date)
  const now = createShanghaiDayjs()
  const diff = expiryDate.diff(now)
  
  if (diff <= 0) {
    return { days: 0, hours: 0, minutes: 0, seconds: 0 }
  }
  
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
  const seconds = Math.floor((diff % (1000 * 60)) / 1000)
  return { days, hours, minutes, seconds }
}

/**
 * 格式化剩余时间。
 */
export function formatRemainingTime(date) {
  if (!date || isExpired(date)) return '已过期'
  const remaining = getRemainingTime(date)
  
  if (remaining.days > 0) {
    return `${remaining.days}天${remaining.hours}小时`
  }
  if (remaining.hours > 0) {
    return `${remaining.hours}小时${remaining.minutes}分钟`
  }
  if (remaining.minutes > 0) {
    return `${remaining.minutes}分钟${remaining.seconds}秒`
  }
  return `${remaining.seconds}秒`
}

// --- 辅助和检查函数 ---

/**
 * 获取月份中文名。
 */
export function getMonthName(month) {
  const monthNames = [
    '一月', '二月', '三月', '四月', '五月', '六月',
    '七月', '八月', '九月', '十月', '十一月', '十二月'
  ]
  return monthNames[month - 1] || ''
}

/**
 * 获取星期中文名。
 */
export function getWeekdayName(day) {
  const weekdayNames = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六']
  return weekdayNames[day] || ''
}

/**
 * 通用日期比较函数。
 */
const checkDateSame = (date, unit) => {
  if (!date) return false
  return createShanghaiDayjs(date).isSame(createShanghaiDayjs(), unit)
}

export function isToday(date) {
  return checkDateSame(date, 'day')
}

export function isYesterday(date) {
  if (!date) return false
  return createShanghaiDayjs(date).isSame(createShanghaiDayjs().subtract(1, 'day'), 'day')
}

export function isThisWeek(date) {
  return checkDateSame(date, 'week')
}

export function isThisMonth(date) {
  return checkDateSame(date, 'month')
}

export function isThisYear(date) {
  return checkDateSame(date, 'year')
}

/**
 * 获取预设日期范围的起始和结束 dayjs 实例。
 */
export function getDateRange(range) {
  const now = createShanghaiDayjs()
  let start, end
  
  switch (range) {
    case 'today':
      start = now.startOf('day')
      end = now.endOf('day')
      break
    case 'yesterday':
      const yesterday = now.subtract(1, 'day')
      start = yesterday.startOf('day')
      end = yesterday.endOf('day')
      break
    case 'week':
      start = now.startOf('week')
      end = now.endOf('week')
      break
    case 'month':
      start = now.startOf('month')
      end = now.endOf('month')
      break
    case 'year':
      start = now.startOf('year')
      end = now.endOf('year')
      break
    default:
      start = now.startOf('day')
      end = now.endOf('day')
  }
  return { start, end }
}

/**
 * 格式化持续时间（秒）为中文描述。
 */
export function formatDuration(seconds) {
  if (!seconds || seconds < 0) return '0秒'
  
  const totalSecs = Math.floor(seconds)
  const days = Math.floor(totalSecs / (24 * 60 * 60))
  const hours = Math.floor((totalSecs % (24 * 60 * 60)) / (60 * 60))
  const minutes = Math.floor((totalSecs % (60 * 60)) / 60)
  const secs = totalSecs % 60
  
  let result = ''
  if (days > 0) result += `${days}天`
  if (hours > 0) result += `${hours}小时`
  if (minutes > 0) result += `${minutes}分钟`
  if (secs > 0 || result === '') result += `${secs}秒`
  return result
}

// --- 时间戳转换 ---

/**
 * 获取当前时间戳（毫秒）。
 */
export function getCurrentTimestamp() {
  return createShanghaiDayjs().valueOf()
}

/**
 * 时间戳转 Date 对象。
 */
export function timestampToDate(timestamp) {
  return dayjs(timestamp).toDate()
}

/**
 * Date 对象转时间戳（毫秒）。
 */
export function dateToTimestamp(date) {
  return createShanghaiDayjs(date).valueOf()
}

/**
 * 获取指定时区的 dayjs 实例。
 */
export function getTimezoneTime(date, timezone = DEFAULT_TIMEZONE) {
  return dayjs(date).tz(timezone)
}


export default {
  formatDateTime,
  formatDate,
  formatTime,
  getRelativeTime,
  getTimeDiff,
  isExpired,
  isExpiringSoon,
  getRemainingDays,
  getRemainingTime,
  formatRemainingTime,
  getMonthName,
  getWeekdayName,
  isToday,
  isYesterday,
  isThisWeek,
  isThisMonth,
  isThisYear,
  getDateRange,
  formatDuration,
  getCurrentTimestamp,
  timestampToDate,
  dateToTimestamp,
  setTimezone,
  getTimezoneTime
}