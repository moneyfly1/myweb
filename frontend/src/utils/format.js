export function formatMoney(value, options = {}) {
  const { prefix = '¥', empty = '0.00' } = options
  if (value === null || value === undefined || value === '') return `${prefix}${empty}`
  const number = Number(value)
  if (Number.isNaN(number)) return `${prefix}${empty}`
  return `${prefix}${number.toFixed(2)}`
}

export function formatNumber(value, options = {}) {
  const { empty = '0', digits } = options
  if (value === null || value === undefined || value === '') return empty
  const number = Number(value)
  if (Number.isNaN(number)) return empty
  return typeof digits === 'number' ? number.toFixed(digits) : String(number)
}

export function formatPercent(value, options = {}) {
  const { empty = '-', digits = 0, ratio = false } = options
  if (value === null || value === undefined || value === '') return empty
  const number = Number(value)
  if (Number.isNaN(number)) return empty
  const normalized = ratio ? number * 100 : number
  return `${normalized.toFixed(digits)}%`
}

export function formatDays(value, empty = '-') {
  if (value === null || value === undefined || value === '') return empty
  const number = Number(value)
  if (Number.isNaN(number)) return empty
  return `${Math.floor(number)}天`
}

export function formatFileSize(bytes, empty = '-') {
  const number = Number(bytes)
  if (!Number.isFinite(number) || number < 0) return empty
  if (number === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(number) / Math.log(1024)), units.length - 1)
  return `${(number / Math.pow(1024, index)).toFixed(index === 0 ? 0 : 2)} ${units[index]}`
}

export function formatFallback(value, empty = '-') {
  if (value === null || value === undefined || value === '') return empty
  return value
}
