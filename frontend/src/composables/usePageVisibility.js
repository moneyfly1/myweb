import { onMounted, onUnmounted, ref } from 'vue'
import { onPageVisible } from '@/utils/pageVisibility'

/**
 * 页面可见性 Composable
 * 用于在页面重新可见时自动刷新数据
 * 
 * @param {Function} refreshFn - 刷新数据的函数
 * @param {Object} options - 配置选项
 * @param {boolean} options.autoRefresh - 是否自动刷新（默认 true）
 * @param {number} options.debounceMs - 防抖延迟（默认 500ms）
 * @returns {Object} { isVisible, refresh }
 */
export function usePageVisibility(refreshFn, options = {}) {
  const { autoRefresh = true, debounceMs = 500 } = options
  const isVisible = ref(true)
  let debounceTimer = null
  let unregisterHandler = null

  const refresh = () => {
    if (typeof refreshFn === 'function') {
      // 防抖处理
      if (debounceTimer) {
        clearTimeout(debounceTimer)
      }
      debounceTimer = setTimeout(() => {
        try {
          refreshFn()
        } catch (error) {
          console.error('刷新数据失败:', error)
        }
      }, debounceMs)
    }
  }

  onMounted(() => {
    if (autoRefresh && typeof refreshFn === 'function') {
      // 注册页面可见性处理器
      unregisterHandler = onPageVisible(() => {
        isVisible.value = true
        refresh()
      })
    }
  })

  onUnmounted(() => {
    if (debounceTimer) {
      clearTimeout(debounceTimer)
    }
    if (unregisterHandler) {
      unregisterHandler()
    }
  })

  return {
    isVisible,
    refresh
  }
}

