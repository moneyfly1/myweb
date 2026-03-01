import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import relativeTime from 'dayjs/plugin/relativeTime'
import utc from 'dayjs/plugin/utc'
import timezone from 'dayjs/plugin/timezone'
dayjs.locale('zh-cn')
dayjs.extend(relativeTime)
dayjs.extend(utc)
dayjs.extend(timezone)
const DEFAULT_TIMEZONE = 'Asia/Shanghai'
export function setTimezone(timezone = DEFAULT_TIMEZONE) {
  dayjs.tz.setDefault(timezone)
}
setTimezone()
const createShanghaiDayjs = (date) => {
  if (!date) return dayjs()
  let d
  if (typeof date === 'string' && date.includes('Z')) {
    d = dayjs.utc(date)
  } else if (typeof date === 'string' && date.match(/[+-]\d{2}:\d{2}$/)) {
    d = dayjs(date)
  } else if (typeof date === 'string' && date.includes('T')) {
    d = dayjs.utc(date)
  } else if (typeof date === 'string' && /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(date)) {
    d = dayjs.tz(date, DEFAULT_TIMEZONE)
  } else {
    d = dayjs(date)
  }
  return d.tz(DEFAULT_TIMEZONE)
}
export function formatDateTime(date, format = 'YYYY-MM-DD HH:mm:ss') {
  if (!date) return ''
  if (typeof date === 'string' && /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(date)) {
    return date
  }
  return createShanghaiDayjs(date).format(format)
}

// formatDate 是 formatDateTime 的别名，保持向后兼容
export const formatDate = formatDateTime
export function formatTime(date, format = 'HH:mm:ss') {
  if (!date) return ''
  return createShanghaiDayjs(date).format(format)
}
export function getRelativeTime(date) {
  if (!date) return ''
  return createShanghaiDayjs(date).fromNow()
}
export function getTimeDiff(date1, date2, unit = 'day') {
  return createShanghaiDayjs(date1).diff(createShanghaiDayjs(date2), unit)
}
export function isExpired(date) {
  if (!date) return true
  return createShanghaiDayjs(date).isBefore(createShanghaiDayjs())
}
export function isExpiringSoon(date, days = 7) {
  if (!date) return false
  const expiryDate = createShanghaiDayjs(date)
  const now = createShanghaiDayjs()
  const diffDays = expiryDate.diff(now, 'day')
  return diffDays >= 0 && diffDays <= days
}
export function getRemainingDays(date) {
  if (!date) return 0
  const expiryDate = createShanghaiDayjs(date)
  const now = createShanghaiDayjs()
  const diffDays = expiryDate.diff(now, 'day')
  return Math.max(0, diffDays)
}
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
export function getMonthName(month) {
  const monthNames = [
    '一月', '二月', '三月', '四月', '五月', '六月',
    '七月', '八月', '九月', '十月', '十一月', '十二月'
  ]
  return monthNames[month - 1] || ''
}
export function getWeekdayName(day) {
  const weekdayNames = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六']
  return weekdayNames[day] || ''
}
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
export function getCurrentTimestamp() {
  return createShanghaiDayjs().valueOf()
}
export function timestampToDate(timestamp) {
  return dayjs(timestamp).toDate()
}
export function dateToTimestamp(date) {
  return createShanghaiDayjs(date).valueOf()
}
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
  getTimezoneTime,
  parseLocation,
  formatLocation,
  getLocationTag
}
export function parseLocation(locationStr) {
  if (!locationStr) {
    return { country: '', city: '', region: '' }
  }
  try {
    const locationData = JSON.parse(locationStr)
    return {
      country: locationData.country || '',
      city: locationData.city || '',
      region: locationData.region || '',
      countryCode: locationData.country_code || ''
    }
  } catch (e) {
    if (locationStr.includes(',')) {
      const parts = locationStr.split(',').map(s => s.trim())
      return {
        country: parts[0] || '',
        city: parts[1] || '',
        region: parts[0] || '',
        countryCode: ''
      }
    }
    return {
      country: locationStr.trim(),
      city: '',
      region: locationStr.trim(),
      countryCode: ''
    }
  }
}
export function formatLocation(locationStr) {
  const location = parseLocation(locationStr)
  if (!location.country) {
    return ''
  }
  if (location.city) {
    return `${location.country}, ${location.city}`
  }
  return location.country
}
export function getLocationTag(locationStr, ipAddress = '') {
  const location = parseLocation(locationStr)
  if (!location.country) {
    return {
      text: '',
      tooltip: ipAddress || '未知位置'
    }
  }
  let text = location.country
  if (location.city) {
    text = `${location.country}, ${location.city}`
  }
  return {
    text,
    tooltip: ipAddress ? `${text} (${ipAddress})` : text
  }
}