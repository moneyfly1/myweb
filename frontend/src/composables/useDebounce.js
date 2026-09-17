import { ref, watch } from 'vue'

/**
 * 通用防抖函数
 * @param {Function} fn - 要防抖的函数
 * @param {Number} delay - 延迟时间（毫秒）
 * @returns {Function} - 防抖后的函数
 */
export function debounce(fn, delay = 300) {
  let timer = null

  const debouncedFn = function(...args) {
    if (timer) {
      clearTimeout(timer)
    }
    timer = setTimeout(() => {
      fn.apply(this, args)
      timer = null
    }, delay)
  }

  // 添加取消方法
  debouncedFn.cancel = () => {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
  }

  return debouncedFn
}
