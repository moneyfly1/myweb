import { ref, onMounted, onUnmounted } from 'vue'

/**
 * 统一的移动端检测 Composable
 * @param {number} breakpoint - 断点像素值，默认 768
 * @returns {import('vue').Ref<boolean>} isMobile
 */
export function useMobile(breakpoint = 768) {
  const isMobile = ref(false)
  
  const checkMobile = () => {
    isMobile.value = window.innerWidth <= breakpoint
  }
  
  onMounted(() => {
    checkMobile()
    window.addEventListener('resize', checkMobile)
  })
  
  onUnmounted(() => {
    window.removeEventListener('resize', checkMobile)
  })
  
  return isMobile
}
