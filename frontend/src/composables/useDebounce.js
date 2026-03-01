import { ref, watch } from 'vue'

/**
 * 搜索防抖 Composable
 * @param {Function} searchFn - 搜索函数
 * @param {Number} delay - 防抖延迟（毫秒）
 * @returns {Object} - 包含 searchValue 和 isSearching 的对象
 */
export function useDebounceSearch(searchFn, delay = 500) {
  const searchValue = ref('')
  const isSearching = ref(false)
  let searchTimer = null

  const debouncedSearch = () => {
    if (searchTimer) {
      clearTimeout(searchTimer)
    }

    isSearching.value = true
    searchTimer = setTimeout(async () => {
      try {
        await searchFn(searchValue.value)
      } finally {
        isSearching.value = false
      }
    }, delay)
  }

  // 监听搜索值变化
  watch(searchValue, () => {
    debouncedSearch()
  })

  // 清理函数
  const cleanup = () => {
    if (searchTimer) {
      clearTimeout(searchTimer)
      searchTimer = null
    }
  }

  return {
    searchValue,
    isSearching,
    cleanup
  }
}

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

/**
 * 节流函数
 * @param {Function} fn - 要节流的函数
 * @param {Number} delay - 延迟时间（毫秒）
 * @returns {Function} - 节流后的函数
 */
export function throttle(fn, delay = 300) {
  let lastTime = 0
  let timer = null

  const throttledFn = function(...args) {
    const now = Date.now()

    if (now - lastTime >= delay) {
      fn.apply(this, args)
      lastTime = now
    } else {
      if (timer) {
        clearTimeout(timer)
      }
      timer = setTimeout(() => {
        fn.apply(this, args)
        lastTime = Date.now()
        timer = null
      }, delay - (now - lastTime))
    }
  }

  // 添加取消方法
  throttledFn.cancel = () => {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
  }

  return throttledFn
}
