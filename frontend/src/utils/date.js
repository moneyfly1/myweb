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
function setTimezone(timezone = DEFAULT_TIMEZONE) {
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

/**
 * formatDateTimeSafe 统一处理各种日期输入（含 Go sql.NullTime 序列化对象）：
 * - null/undefined/空串 → empty（默认 '-'）
 * - {Time, Valid:false} → empty
 * - {Time, Valid:true} → 按 format 格式化
 * - 字符串/Date → 按 format 格式化
 * 供各页面替换本地重复的 formatDate 实现。
 */
export function formatDateTimeSafe(date, format = 'YYYY-MM-DD HH:mm:ss', empty = '-') {
  if (date === null || date === undefined || date === '') return empty
  if (typeof date === 'object') {
    if (date.Valid === false) return empty
    if (date.Time) return createShanghaiDayjs(date.Time).format(format)
    return empty
  }
  return formatDateTime(date, format)
}
export function formatTime(date, format = 'HH:mm:ss') {
  if (!date) return ''
  return createShanghaiDayjs(date).format(format)
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
function parseLocation(locationStr) {
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

/**
 * 统一的登录位置文本（本地/内网判定）
 * 来源：Profile.vue:544、LoginHistory.vue:370、admin/Profile.vue:614 的 getLocationText
 * @param {string} location 位置（JSON / "国家, 城市" / 纯文本），非空时优先展示 formatLocation
 * @param {string} ip IP 地址
 * @param {object} [options] 可选参数
 * @param {string} [options.pendingText] 公网 IP 且无 location 时返回的「解析中」文案（Profile 页用）
 * @returns {string}
 */
export function getLocationText(location, ip, options = {}) {
  const { pendingText = '' } = options
  if (location) {
    return formatLocation(location)
  }
  if (ip && ip !== '未知') {
    if (isLocalOrPrivateIP(ip)) {
      return ip === '127.0.0.1' || ip === '::1' || ip === 'localhost' || ip.startsWith('127.')
        ? '本地'
        : '内网'
    }
    if (pendingText) return pendingText
  }
  return ''
}

/**
 * isLocalOrPrivateIP 判断是否为本机/内网/保留地址（与后端 netutil.IsPrivateOrReserved 对齐）。
 *
 * 历史缺陷：此前用 `ip.startsWith('172.')` 判定内网，而 172.0.x–172.15.x 与 172.32.x+
 * 都是公网地址，会被误标为"内网"；正确范围是 172.16.0.0/12。
 * 同时补齐了 169.254/16（链路本地）、100.64/10（CGNAT）等保留段。
 */
function isLocalOrPrivateIP(ip) {
  const value = String(ip || '').trim().toLowerCase()
  if (!value) return false
  if (value === 'localhost' || value === '::1' || value === '0:0:0:0:0:0:0:1') return true
  // IPv6 内网：fe80::/10（链路本地）、fc00::/7（唯一本地）
  if (/^fe[89ab][0-9a-f]:/.test(value)) return true
  if (/^f[cd][0-9a-f]{2}:/.test(value)) return true
  const v4 = value.replace(/^::ffff:/, '')
  const parts = v4.split('.')
  if (parts.length !== 4 || parts.some(p => !/^\d{1,3}$/.test(p))) return false
  const [a, b] = parts.map(Number)
  if (a === 10) return true
  if (a === 172 && b >= 16 && b <= 31) return true
  if (a === 192 && b === 168) return true
  if (a === 127) return true
  if (a === 169 && b === 254) return true
  if (a === 100 && b >= 64 && b <= 127) return true
  if (a === 0) return true
  if (a >= 224) return true
  return false
}
