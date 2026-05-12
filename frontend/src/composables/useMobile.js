import { ref, onMounted, onUnmounted } from 'vue'

/**
 * 统一的移动端检测 Composable
 * @param {number} breakpoint - 断点像素值，默认 768
 * @returns {import('vue').Ref<boolean>} isMobile
 */
export function useMobile(breakpoint = 768) {
  const isMobile = ref(false)
  let rafId = null
  
  const checkMobile = () => {
    if (typeof window === 'undefined') return
    isMobile.value = window.innerWidth <= breakpoint
  }

  const scheduleCheckMobile = () => {
    if (rafId !== null) return
    rafId = window.requestAnimationFrame(() => {
      rafId = null
      checkMobile()
    })
  }
  
  onMounted(() => {
    checkMobile()
    window.addEventListener('resize', scheduleCheckMobile, { passive: true })
  })
  
  onUnmounted(() => {
    window.removeEventListener('resize', scheduleCheckMobile)
    if (rafId !== null) {
      window.cancelAnimationFrame(rafId)
      rafId = null
    }
  })
  
  return isMobile
}
