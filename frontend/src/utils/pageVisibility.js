/**
 * 页面可见性管理工具
 * 用于处理页面切换后的连接恢复和数据刷新
 */

let visibilityHandlers = []
let isPageVisible = true

// 初始化页面可见性监听
if (typeof document !== 'undefined') {
  // 监听页面可见性变化
  document.addEventListener('visibilitychange', () => {
    const wasVisible = isPageVisible
    isPageVisible = !document.hidden
    
    if (wasVisible && !isPageVisible) {
      // 页面变为隐藏
      console.debug('页面已隐藏')
    } else if (!wasVisible && isPageVisible) {
      // 页面重新可见
      console.debug('页面重新可见，触发刷新')
      triggerVisibilityHandlers()
    }
  })
  
  // 监听窗口焦点变化（作为备用）
  window.addEventListener('focus', () => {
    if (isPageVisible) {
      console.debug('窗口获得焦点，触发刷新')
      triggerVisibilityHandlers()
    }
  })
}

/**
 * 触发所有可见性处理器
 */
function triggerVisibilityHandlers() {
  visibilityHandlers.forEach(handler => {
    try {
      handler()
    } catch (error) {
      console.error('页面可见性处理器执行失败:', error)
    }
  })
}

/**
 * 注册页面可见性处理器
 * @param {Function} handler - 当页面重新可见时调用的函数
 * @returns {Function} 取消注册的函数
 */
export function onPageVisible(handler) {
  if (typeof handler !== 'function') {
    console.warn('onPageVisible: handler 必须是函数')
    return () => {}
  }
  
  visibilityHandlers.push(handler)
  
  // 返回取消注册的函数
  return () => {
    const index = visibilityHandlers.indexOf(handler)
    if (index > -1) {
      visibilityHandlers.splice(index, 1)
    }
  }
}

/**
 * 检查页面是否可见
 */
export function isVisible() {
  return isPageVisible
}

/**
 * 清除所有处理器
 */
export function clearHandlers() {
  visibilityHandlers = []
}

