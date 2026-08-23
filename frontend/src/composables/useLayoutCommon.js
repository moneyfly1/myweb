import { ref, computed, watch, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/store/auth'
import { useThemeStore } from '@/store/theme'
import { ElMessage } from '@/utils/elementPlusServices'
import { secureStorage, ticketAPI } from '@/utils/api'
import { useMobile } from '@/composables/useMobile'
import { initTextSelection, cleanupTextSelection } from '@/utils/textSelection'

/**
 * 用户端/管理端布局共享逻辑（AdminLayout 与 UserLayout 此前约 70% 的 script 重复）：
 * - 侧边栏折叠 / 移动端导航状态与持久化
 * - 主题状态（当前主题/可用主题/切换）
 * - 工单未读数轮询
 * - 菜单区折叠（collapsedSections）
 * - 文本选择工具初始化
 *
 * 用法：const layout = useLayoutCommon({ storageKey, initThemeLoad })
 */
export function useLayoutCommon({ storageKey = 'sidebarCollapsed', initThemeLoad = false } = {}) {
  const router = useRouter()
  const route = useRoute()
  const authStore = useAuthStore()
  const themeStore = useThemeStore()
  const isMobile = useMobile()

  const sidebarCollapsed = ref(false)
  const mobileNavExpanded = ref(false)
  const collapsedSections = ref({})
  const unreadCount = ref(0)
  let unreadCheckInterval = null
  let unreadCountRequest = null

  const user = computed(() => authStore.user)
  const currentTheme = computed(() => themeStore.currentTheme)
  const themes = computed(() => themeStore.availableThemes)
  const userAvatar = computed(() => user.value?.avatar || '')
  const userInitials = computed(() => user.value?.username?.substring(0, 2).toUpperCase() || '')
  const displayUsername = computed(() => user.value?.username || '用户')

  const toggleSidebar = () => {
    sidebarCollapsed.value = !sidebarCollapsed.value
    if (!isMobile.value) localStorage.setItem(storageKey, sidebarCollapsed.value)
  }

  const navigateTo = (path) => {
    router.push(path)
    mobileNavExpanded.value = false
    if (isMobile.value) sidebarCollapsed.value = true
  }

  const handleNavClick = () => {
    if (isMobile.value) sidebarCollapsed.value = true
  }

  const toggleSection = (title) => {
    collapsedSections.value[title] = !collapsedSections.value[title]
  }

  const checkMobile = () => {
    if (isMobile.value) {
      sidebarCollapsed.value = true
    } else {
      sidebarCollapsed.value = false
      localStorage.setItem(storageKey, 'false')
    }
  }

  const loadUnreadCount = async (apiCall) => {
    if (unreadCountRequest) return unreadCountRequest
    try {
      unreadCountRequest = (apiCall || ticketAPI.getUnreadCount())()
      const response = await unreadCountRequest
      if (response.data && response.data.success) {
        unreadCount.value = response.data.data?.count || 0
      }
    } catch (error) {
      // 未读消息数加载失败，不影响主功能
    } finally {
      unreadCountRequest = null
    }
  }

  const startUnreadPolling = (apiCall) => {
    loadUnreadCount(apiCall)
    unreadCheckInterval = setInterval(() => loadUnreadCount(apiCall), 30000)
  }

  const handleThemeChange = async (name) => {
    try {
      const res = await themeStore.setTheme(name)
      res.success ? ElMessage.success('主题已保存') : ElMessage.warning(res.message || '本地生效')
    } catch (error) {
      ElMessage.warning('主题保存失败，仅本地生效')
    }
  }

  const init = () => {
    // 恢复折叠偏好
    try {
      const stored = localStorage.getItem(storageKey)
      if (stored !== null) sidebarCollapsed.value = stored === 'true'
    } catch (e) { /* ignore */ }
    checkMobile()
    if (initThemeLoad) {
      themeStore.loadUserTheme().catch(() => {})
    }
    initTextSelection()
    watch(isMobile, () => checkMobile())
  }

  const cleanup = () => {
    if (unreadCheckInterval) {
      clearInterval(unreadCheckInterval)
      unreadCheckInterval = null
    }
    cleanupTextSelection()
  }

  onUnmounted(cleanup)

  return {
    router, route, isMobile,
    sidebarCollapsed, mobileNavExpanded, collapsedSections, unreadCount,
    user, currentTheme, themes, userAvatar, userInitials, displayUsername,
    toggleSidebar, navigateTo, handleNavClick, toggleSection, checkMobile,
    loadUnreadCount, startUnreadPolling, handleThemeChange, init, cleanup
  }
}
