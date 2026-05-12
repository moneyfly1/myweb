import { ElMessage, ElMessageBox } from '@/utils/elementPlusServices'

/**
 * 统一的错误处理工具
 */

/**
 * 提取错误消息
 * @param {Error} error - 错误对象
 * @returns {string} 错误消息
 */
export function extractErrorMessage(error) {
  if (!error) return '未知错误'
  
  // Axios 错误
  if (error.response) {
    // 服务器返回错误状态码
    const status = error.response.status
    const message = error.response.data?.message || error.response.data?.error || error.message
    
    switch (status) {
      case 400:
        return `请求错误: ${message}`
      case 401:
        return '登录已过期，请重新登录'
      case 403:
        return '权限不足，无法执行此操作'
      case 404:
        return '请求的资源不存在'
      case 422:
        return `验证失败: ${message}`
      case 429:
        return '请求过于频繁，请稍后再试'
      case 500:
        return '服务器错误，请稍后重试'
      case 502:
        return '网关错误，请稍后重试'
      case 503:
        return '服务暂时不可用，请稍后重试'
      default:
        return `错误 (${status}): ${message}`
    }
  }
  
  // 网络错误
  if (error.code === 'ECONNABORTED') {
    return '请求超时，请检查网络连接'
  }
  
  if (error.code === 'ERR_NETWORK') {
    return '网络连接失败，请检查网络设置'
  }
  
  // 其他错误
  return error.message || '操作失败，请稍后重试'
}

/**
 * 显示错误消息
 * @param {Error|string} error - 错误对象或错误消息
 * @param {Object} options - 选项
 */
export function showError(error, options = {}) {
  const message = typeof error === 'string' ? error : extractErrorMessage(error)
  const duration = options.duration || 3000
  
  ElMessage.error({
    message,
    duration,
    showClose: options.showClose !== false
  })
  
  // 开发环境下打印详细错误
  if (process.env.NODE_ENV === 'development') {
    console.error('[API Error]', error)
  }
}

/**
 * 显示成功消息
 * @param {string} message - 成功消息
 * @param {Object} options - 选项
 */
export function showSuccess(message, options = {}) {
  ElMessage.success({
    message,
    duration: options.duration || 2000,
    showClose: options.showClose !== false
  })
}

/**
 * 显示警告消息
 * @param {string} message - 警告消息
 * @param {Object} options - 选项
 */
export function showWarning(message, options = {}) {
  ElMessage.warning({
    message,
    duration: options.duration || 3000,
    showClose: options.showClose !== false
  })
}

/**
 * 显示信息消息
 * @param {string} message - 信息消息
 * @param {Object} options - 选项
 */
export function showInfo(message, options = {}) {
  ElMessage.info({
    message,
    duration: options.duration || 3000,
    showClose: options.showClose !== false
  })
}

/**
 * 确认对话框
 * @param {string} message - 确认消息
 * @param {string} title - 对话框标题
 * @param {Object} options - 选项
 * @returns {Promise<boolean>} 是否确认
 */
export async function showConfirm(message, title = '提示', options = {}) {
  try {
    await ElMessageBox.confirm(message, title, {
      confirmButtonText: options.confirmText || '确定',
      cancelButtonText: options.cancelText || '取消',
      type: options.type || 'warning',
      ...options
    })
    return true
  } catch {
    return false
  }
}

/**
 * 异步操作包装器
 * 自动处理 loading 状态和错误提示
 * @param {Function} asyncFn - 异步函数
 * @param {Object} options - 选项
 * @returns {Promise<any>} 操作结果
 */
export async function withLoading(asyncFn, options = {}) {
  const {
    loadingRef,
    successMessage,
    errorMessage = '操作失败',
    onError,
    onSuccess
  } = options
  
  try {
    if (loadingRef) loadingRef.value = true
    
    const result = await asyncFn()
    
    if (successMessage) {
      showSuccess(successMessage)
    }
    
    if (onSuccess) {
      onSuccess(result)
    }
    
    return result
  } catch (error) {
    const message = typeof errorMessage === 'function' 
      ? errorMessage(error) 
      : errorMessage
    
    showError(error, { message })
    
    if (onError) {
      onError(error)
    }
    
    throw error
  } finally {
    if (loadingRef) loadingRef.value = false
  }
}
