/**
 * 安全地打开新窗口
 * 防止 Tabnabbing 攻击
 *
 * @param {string} url - 要打开的 URL
 * @param {string} target - 目标窗口名称，默认 '_blank'
 * @param {string} features - 窗口特性，默认 'noopener,noreferrer'
 * @returns {Window|null} 新窗口对象或 null
 */
export function safeOpen(url, target = '_blank', features = 'noopener,noreferrer') {
  if (!url) {
    console.warn('safeOpen: URL is required')
    return null
  }

  try {
    // 验证 URL 格式
    const urlObj = new URL(url, window.location.origin)

    // 检查协议是否安全
    const safeProtocols = ['http:', 'https:', 'mailto:', 'tel:', 'sms:']
    if (!safeProtocols.includes(urlObj.protocol)) {
      // 对于自定义协议（如 shadowrocket://），允许但记录警告
      if (process.env.NODE_ENV === 'development') {
        console.warn(`safeOpen: Opening URL with custom protocol: ${urlObj.protocol}`)
      }
    }

    // 打开新窗口
    const newWindow = window.open(url, target, features)

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

/**
 * 安全地打开外部链接
 * 专门用于打开外部网站
 *
 * @param {string} url - 外部 URL
 * @returns {Window|null} 新窗口对象或 null
 */
export function safeOpenExternal(url) {
  return safeOpen(url, '_blank', 'noopener,noreferrer,nofollow')
}

/**
 * 安全地打开应用内链接
 * 用于打开同域名下的页面
 *
 * @param {string} url - 内部 URL
 * @returns {Window|null} 新窗口对象或 null
 */
export function safeOpenInternal(url) {
  return safeOpen(url, '_blank', 'noopener')
}

export default {
  safeOpen,
  safeOpenExternal,
  safeOpenInternal
}
