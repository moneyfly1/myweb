const NAVIGATION_PROTOCOLS = new Set(['http:', 'https:', 'mailto:', 'tel:', 'sms:'])
const APP_PROTOCOLS = new Set([
  'alipays:',
  'clash:',
  'hiddify:',
  'quantumult:',
  'quantumult-x:',
  'shadowrocket:',
  'ssr:',
  'sub:',
  'v2rayng:',
  'weixin:',
  'wxp:',
])

function normalizeSafeUrl(url, { allowAppProtocols = false, allowRelative = true } = {}) {
  if (!url) {
    return null
  }

  try {
    const rawUrl = String(url).trim()
    if (!rawUrl) return null

    const baseUrl = typeof window !== 'undefined' ? window.location.origin : 'https://localhost'
    const urlObj = new URL(rawUrl, baseUrl)
    const protocol = urlObj.protocol.toLowerCase()
    const isRelative = !/^[a-z][a-z0-9+.-]*:/i.test(rawUrl)

    if (isRelative && !allowRelative) {
      return null
    }

    if (!NAVIGATION_PROTOCOLS.has(protocol) && !(allowAppProtocols && APP_PROTOCOLS.has(protocol))) {
      return null
    }

    return isRelative ? urlObj.pathname + urlObj.search + urlObj.hash : urlObj.href
  } catch (error) {
    if (process.env.NODE_ENV === 'development') {
      console.warn('normalizeSafeUrl: invalid URL', error)
    }
    return null
  }
}

/**
 * 安全地打开新窗口，防止危险协议和 Tabnabbing。
 */
export function safeOpen(url, target = '_blank', features = 'noopener,noreferrer', options = {}) {
  const safeUrl = normalizeSafeUrl(url, options)
  if (!safeUrl) {
    if (process.env.NODE_ENV === 'development') {
      console.warn('safeOpen: blocked unsafe URL')
    }
    return null
  }

  try {
    const newWindow = window.open(safeUrl, target, features)
    // 确保 opener 被清除（双重保险）
    if (newWindow) {
      newWindow.opener = null
    }
    return newWindow
  } catch (error) {
    console.error('safeOpen: Failed to open URL', error)
    return null
  }
}

export function safeOpenApp(url) {
  return safeOpen(url, '_blank', 'noopener,noreferrer', { allowAppProtocols: true })
}

export function safeNavigate(url, { allowAppProtocols = false, replace = false } = {}) {
  const safeUrl = normalizeSafeUrl(url, { allowAppProtocols })
  if (!safeUrl || typeof window === 'undefined') {
    if (process.env.NODE_ENV === 'development') {
      console.warn('safeNavigate: blocked unsafe URL')
    }
    return false
  }
  if (replace) {
    window.location.replace(safeUrl)
  } else {
    window.location.href = safeUrl
  }
  return true
}

