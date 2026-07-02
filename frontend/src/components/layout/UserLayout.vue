<template>
  <div class="user-layout" :class="{ 'sidebar-collapsed': sidebarCollapsed, 'is-mobile': isMobile }">
    <header class="header">
      <div class="header-left">
        <button class="menu-toggle" @click.stop="toggleSidebar" type="button" aria-label="切换菜单">
          <el-icon class="layout-icon"><component :is="sidebarCollapsed ? Menu : Close" /></el-icon>
          <span class="menu-toggle-text">菜单</span>
        </button>
        <router-link to="/dashboard" class="logo">
          <span class="logo-mark">C</span>
          <span class="logo-text">CBoard</span>
        </router-link>
      </div>
      <div class="header-right">
        <el-dropdown @command="handleThemeChange" class="theme-dropdown">
          <el-button class="theme-btn">
            <el-icon class="layout-icon"><Brush /></el-icon>
            <span class="theme-text">主题</span>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item 
                v-for="theme in themes" :key="theme.value" :command="theme.value"
                :class="{ active: currentTheme === theme.value }">
                <el-icon class="dropdown-icon" v-if="currentTheme === theme.value"><Check /></el-icon>
                {{ theme.label }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-dropdown @command="handleUserCommand" class="user-dropdown">
          <div class="user-info">
            <el-avatar :size="28" :src="userAvatar">{{ userInitials }}</el-avatar>
            <span class="user-name" v-show="!isMobile">{{ displayUsername }}</span>
            <el-icon class="layout-icon"><ArrowDown /></el-icon>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-if="hasAdminAccess" command="backToAdmin" divided>
                <el-icon class="dropdown-icon"><Back /></el-icon> 返回管理后台
              </el-dropdown-item>
              <el-dropdown-item command="settings">
                <el-icon class="dropdown-icon"><Setting /></el-icon> 设置
              </el-dropdown-item>
              <el-dropdown-item divided command="logout">
                <el-icon class="dropdown-icon"><SwitchButton /></el-icon> 退出登录
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </header>
    <aside class="sidebar" :class="{ collapsed: sidebarCollapsed }">
      <div class="mobile-menu-header" v-if="isMobile">
        <span class="menu-title">菜单</span>
        <button class="menu-close-btn" @click.stop="toggleSidebar" type="button" aria-label="关闭菜单">
          <el-icon class="layout-icon"><Close /></el-icon>
        </button>
      </div>
      <nav class="sidebar-nav">
        <div v-for="section in menuSections" :key="section.title" class="nav-section">
          <div class="nav-section-title" v-show="!sidebarCollapsed || isMobile">{{ section.title }}</div>
          <template v-for="item in section.items" :key="item.path">
            <div v-if="item.isAdminBack"
              class="nav-item admin-back"
              @click="returnToAdmin()">
              <el-icon class="nav-icon"><component :is="getIcon(item.icon)" /></el-icon>
              <span class="nav-text" v-show="!sidebarCollapsed || isMobile">{{ item.title }}</span>
            </div>
            <router-link v-else
              :to="item.path"
              class="nav-item" :class="{ active: isRouteActive(item.path) }"
              @click="handleNavClick">
              <el-icon class="nav-icon"><component :is="getIcon(item.icon)" /></el-icon>
              <span class="nav-text" v-show="!sidebarCollapsed || isMobile">{{ item.title }}</span>
              <el-badge
                v-if="item.badge && item.badge > 0 && (!sidebarCollapsed || isMobile)"
                :value="item.badge"
                :max="99"
                type="danger"
                class="nav-badge"
              />
            </router-link>
          </template>
        </div>
      </nav>
    </aside>
    <main class="main-content">
      <div class="content-wrapper">
        <div class="mobile-nav-bar" v-if="isMobile">
          <button
            type="button"
            class="mobile-nav-header"
            :aria-expanded="mobileNavExpanded"
            aria-controls="user-mobile-nav-menu"
            @click="mobileNavExpanded = !mobileNavExpanded"
          >
            <div class="nav-current-path">
              <el-icon class="layout-icon"><Location /></el-icon>
              <span class="current-title">{{ currentPageTitle }}</span>
            </div>
            <el-icon class="layout-icon nav-expand-icon" :class="{ 'expanded': mobileNavExpanded }"><ArrowDown /></el-icon>
          </button>
          <transition name="slide-down">
            <div id="user-mobile-nav-menu" class="mobile-nav-menu" v-show="mobileNavExpanded">
              <div v-for="section in menuSections" :key="section.title" class="nav-section-menu">
                <div class="section-title">{{ section.title }}</div>
                <template v-for="item in section.items" :key="item.path">
                  <div v-if="item.isAdminBack"
                    class="nav-menu-item admin-back"
                    @click="returnToAdmin()">
                    <el-icon class="nav-icon"><component :is="getIcon(item.icon)" /></el-icon>
                    <span>{{ item.title }}</span>
                  </div>
                  <router-link v-else :to="item.path"
                    class="nav-menu-item" :class="{ 'active': isRouteActive(item.path) }"
                    @click="navigateTo(item.path)">
                    <el-icon class="nav-icon"><component :is="getIcon(item.icon)" /></el-icon>
                    <span>{{ item.title }}</span>
                    <el-icon class="active-check-icon" v-if="isRouteActive(item.path)"><Check /></el-icon>
                  </router-link>
                </template>
              </div>
            </div>
          </transition>
        </div>
        <div class="page-content">
          <router-view />
        </div>
      </div>
    </main>
    <div v-if="isMobile && !sidebarCollapsed" class="mobile-overlay" @click.stop="sidebarCollapsed = true" />
    <div v-if="isMobile" class="mobile-tabbar">
      <router-link to="/dashboard" class="mobile-tab" :class="{ active: isRouteActive('/dashboard') }">
        <svg viewBox="0 0 512 512" class="tab-icon"><rect x="48" y="48" width="176" height="176" rx="20" ry="20" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/><rect x="288" y="48" width="176" height="176" rx="20" ry="20" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/><rect x="48" y="288" width="176" height="176" rx="20" ry="20" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/><rect x="288" y="288" width="176" height="176" rx="20" ry="20" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/></svg>
        <span class="mobile-tab-label">仪表盘</span>
      </router-link>
      <router-link to="/packages" class="mobile-tab" :class="{ active: isRouteActive('/packages') }">
        <svg viewBox="0 0 512 512" class="tab-icon"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M80 176a16 16 0 00-16 16v216a16 16 0 0016 16h352a16 16 0 0016-16V192a16 16 0 00-16-16zM80 176l48-80h256l48 80"/><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M256 320l-48-48h96l-48 48zm0 0v88"/></svg>
        <span class="mobile-tab-label">套餐购买</span>
      </router-link>
      <router-link to="/devices" class="mobile-tab" :class="{ active: isRouteActive('/devices') }">
        <svg viewBox="0 0 512 512" class="tab-icon"><rect x="128" y="16" width="256" height="480" rx="48" ry="48" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/><path d="M176 16h24a8 8 0 018 8h0a16 16 0 0016 16h64a16 16 0 0016-16h0a8 8 0 018-8h24" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/></svg>
        <span class="mobile-tab-label">设备管理</span>
      </router-link>
      <router-link to="/subscription" class="mobile-tab" :class="{ active: isRouteActive('/subscription') }">
        <svg viewBox="0 0 512 512" class="tab-icon"><path d="M208 352h-64a96 96 0 010-192h64M304 160h64a96 96 0 010 192h-64M176 256h160" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/></svg>
        <span class="mobile-tab-label">订阅</span>
      </router-link>
      <router-link to="/orders" class="mobile-tab" :class="{ active: isRouteActive('/orders') }">
        <svg viewBox="0 0 512 512" class="tab-icon"><path d="M160 96h192a48 48 0 0148 48v272l-48-32-48 32-48-32-48 32-48-32-48 32V144a48 48 0 0148-48z" fill="none" stroke="currentColor" stroke-linejoin="round" stroke-width="32"/><path d="M192 192h128M192 256h128M192 320h80" fill="none" stroke="currentColor" stroke-linecap="round" stroke-width="32"/></svg>
        <span class="mobile-tab-label">订单</span>
      </router-link>
    </div>
  </div>
</template>
<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  ArrowDown,
  Back,
  Brush,
  Cellphone,
  Check,
  Close,
  Connection,
  Document,
  House,
  Location,
  Menu,
  QuestionFilled,
  Reading,
  Setting,
  ShoppingCart,
  SwitchButton,
  Tickets,
  User
} from '@element-plus/icons-vue'
import { useAuthStore } from '@/store/auth'
import { useThemeStore } from '@/store/theme'
import { ElMessage } from '@/utils/elementPlusServices'
import { secureStorage } from '@/utils/api'
import { ticketAPI } from '@/utils/api'
import { useMobile } from '@/composables/useMobile'
import { initTextSelection, cleanupTextSelection } from '@/utils/textSelection'
import '@/styles/user-client-polish.scss'
import '@/styles/text-selection.css'
const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const isMobile = useMobile()
const sidebarCollapsed = ref(false)
const mobileNavExpanded = ref(false)
const unreadTicketReplies = ref(0)
let unreadCheckInterval = null
let unreadRepliesRequest = null
const user = computed(() => authStore.user)
const currentTheme = computed(() => themeStore.currentTheme)
const themes = computed(() => themeStore.availableThemes)
const userAvatar = computed(() => user.value?.avatar || '')
const userInitials = computed(() => user.value?.username?.substring(0, 2).toUpperCase() || '')
const displayUsername = computed(() => user.value?.username || '用户')
const hasAdminAccess = computed(() => !!(secureStorage.get('admin_token') && secureStorage.get('admin_user')))
const iconMap = {
  dashboard: House,
  subscription: Connection,
  devices: Cellphone,
  nodes: Location,
  packages: ShoppingCart,
  orders: Document,
  tickets: Tickets,
  invites: User,
  knowledge: Reading,
  help: QuestionFilled,
  tutorials: Reading,
  profile: User,
  loginHistory: Document,
  settings: Setting,
  back: Back
}
const getIcon = (icon) => iconMap[icon] || Document
const menuSections = computed(() => {
  const baseSections = [
    { 
      title: '概览', 
      items: [
        { path: '/dashboard', title: '仪表盘', icon: 'dashboard' }
      ] 
    },
    { 
      title: '服务管理', 
      items: [
        { path: '/subscription', title: '订阅管理', icon: 'subscription' },
        { path: '/devices', title: '设备管理', icon: 'devices' },
        { path: '/nodes', title: '节点列表', icon: 'nodes' }
      ] 
    },
    { 
      title: '订单服务', 
      items: [
        { path: '/packages', title: '套餐购买', icon: 'packages' },
        { path: '/orders', title: '订单记录', icon: 'orders' }
      ] 
    },
    { 
      title: '其他功能', 
      items: [
        { 
          path: '/tickets', 
          title: '工单中心', 
          icon: 'tickets',
          badge: unreadTicketReplies.value > 0 ? unreadTicketReplies.value : null
        },
        { path: '/invites', title: '我的邀请', icon: 'invites' },
        { path: '/knowledge', title: '知识库', icon: 'knowledge' },
        { path: '/help', title: '帮助中心', icon: 'help' },
        { path: '/tutorials', title: '软件教程', icon: 'tutorials' },
        { path: '/login-history', title: '登录历史', icon: 'loginHistory' },
        { path: '/settings', title: '用户设置', icon: 'settings' }
      ] 
    }
  ]
  if (hasAdminAccess.value) {
    baseSections.push({
      title: '管理功能',
      items: [
        { path: '#admin', title: '返回管理后台', icon: 'back', isAdminBack: true }
      ]
    })
  }
  return baseSections
})
const currentPageTitle = computed(() => {
  if (route.meta.title) return route.meta.title
  const allItems = menuSections.value.flatMap(s => s.items)
  return allItems.find(i => i.path === route.path)?.title || '用户中心'
})
const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
  if (!isMobile.value) localStorage.setItem('userSidebarCollapsed', sidebarCollapsed.value)
}
const isRouteActive = (path) => {
  if (path === '#admin') return false
  return route.path === path || (path !== '/dashboard' && route.path.startsWith(path))
}
const navigateTo = (path) => {
  if (path === '#admin') {
    returnToAdmin()
  } else {
    router.push(path)
  }
  mobileNavExpanded.value = false
  if (isMobile.value) sidebarCollapsed.value = true
}
const handleNavClick = () => {
  if (isMobile.value) sidebarCollapsed.value = true
}
const handleUserCommand = (command) => {
  const actions = {
    backToAdmin: returnToAdmin,
    settings: () => router.push('/settings'),
    logout: () => { authStore.logout('user'); router.push('/login') }
  }
  actions[command]?.()
}
const getCurrentThemeLabel = () => themes.value.find(t => t.value === currentTheme.value)?.label || '主题'
const loadUnreadTicketReplies = async () => {
  if (unreadRepliesRequest) return unreadRepliesRequest
  try {
    unreadRepliesRequest = ticketAPI.getUnreadCount()
    const response = await unreadRepliesRequest
    if (response.data && response.data.success) {
      unreadTicketReplies.value = response.data.data?.count || 0
    }
  } catch (error) {
    // 未读消息数加载失败，不影响主功能
  } finally {
    unreadRepliesRequest = null
  }
}
const returnToAdmin = () => {
  const token = secureStorage.get('admin_token')
  const userData = secureStorage.get('admin_user')
  try {
    const user = typeof userData === 'string' ? JSON.parse(userData) : userData
    if (!user?.is_admin) throw new Error('Not Admin')
    authStore.logout('user')
    authStore.setAuth(token, user, false)
    router.push('/admin/dashboard')
    ElMessage.success('已返回管理员后台')
  } catch (e) {
    ElMessage.error('返回失败，请重新登录')
  }
}
const handleThemeChange = async (name) => {
  const res = await themeStore.setTheme(name)
  res.success ? ElMessage.success('主题已同步') : ElMessage.warning('本地生效')
}
const checkMobile = () => {
  if (isMobile.value) {
    sidebarCollapsed.value = true
  } else {
    sidebarCollapsed.value = false
    localStorage.setItem('userSidebarCollapsed', 'false')
  }
}
watch(isMobile, () => {
  checkMobile()
})
watch(() => route.path, () => {
  if (isMobile.value) {
    mobileNavExpanded.value = false
    sidebarCollapsed.value = true
  }
  if (route.path === '/tickets') {
    loadUnreadTicketReplies()
  }
})
onMounted(() => {
  checkMobile()
  initTextSelection()
  loadUnreadTicketReplies()
  unreadCheckInterval = setInterval(() => {
    loadUnreadTicketReplies()
  }, 30000)
  window.addEventListener('ticket-viewed', loadUnreadTicketReplies)
})
onUnmounted(() => {
  window.removeEventListener('ticket-viewed', loadUnreadTicketReplies)
  cleanupTextSelection()
  if (unreadCheckInterval) {
    clearInterval(unreadCheckInterval)
    unreadCheckInterval = null
  }
})
</script>
<style scoped lang="scss">
@use '@/styles/global.scss' as *;
.layout-icon,
.dropdown-icon,
.nav-icon,
.active-check-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.dropdown-icon {
  margin-right: 6px;
}

.user-layout {
  --header-height: 60px;
  --sidebar-width: 200px;
  --sidebar-collapsed-width: 64px;
  --content-padding: 20px;
  display: flex;
  height: 100vh;
  background-color: #f5f7fa;
  overflow-x: clip;
}
.header {
  position: fixed;
  top: 0; left: 0; right: 0;
  height: var(--header-height);
  background: #fff;
  border-bottom: 1px solid #dcdfe6;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  z-index: 1000;
  @include respond-to(sm) { height: 50px; padding: 0 12px; }
  .header-left {
    display: flex; align-items: center; gap: 16px;
    .logo {
      display: flex;
      align-items: center;
      cursor: pointer;
      text-decoration: none;
      color: var(--theme-primary, #409eff);
      font-size: 20px;
      font-weight: 700;
    }
    .logo-text { color: var(--theme-primary, #409eff); }
    .menu-toggle {
      width: 34px;
      height: 34px;
      display: none;
      align-items: center;
      justify-content: center;
      border: 1px solid var(--theme-border);
      background: #fff;
      padding: 0;
      border-radius: 6px;
      color: #606266;
      cursor: pointer;
      transition: background-color 0.16s ease, border-color 0.16s ease, color 0.16s ease;
      &:hover {
        border-color: #c6e2ff;
        background: #f5f7fa;
        color: var(--theme-primary, #409eff);
      }
      .menu-toggle-text {
        display: none;
      }
      @include respond-to(sm) {
        display: flex;
      }
    }
  }
  .header-right {
    display: flex; align-items: center; gap: 16px;
    .theme-dropdown, .user-dropdown {
      .theme-btn, .user-info {
        min-height: 34px;
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 0 10px;
        border: 1px solid #dcdfe6;
        border-radius: 6px;
        background: #fff;
        color: #303133;
        cursor: pointer;
        transition: background-color 0.16s ease, border-color 0.16s ease, color 0.16s ease;
        &:hover { background: #f5f7fa; }
      }
      .theme-btn {
        height: 34px;
        --el-button-bg-color: #fff;
        --el-button-border-color: #dcdfe6;
        --el-button-hover-bg-color: #f5f7fa;
        --el-button-hover-border-color: #c6e2ff;
        --el-button-text-color: #303133;
        --el-button-hover-text-color: var(--theme-primary, #409eff);
      }
      .theme-text {
        color: inherit;
        font-weight: 500;
      }
      .user-info {
        height: 34px;
        .user-name { margin: 0 8px; }
      }
    }
  }
}
.sidebar {
  position: fixed;
  top: var(--header-height);
  left: 0;
  width: var(--sidebar-width);
  height: calc(100vh - var(--header-height));
  background: #fff;
  border-right: 1px solid #dcdfe6;
  transition: width 0.2s ease, transform 0.2s ease;
  z-index: 999;
  overflow-y: auto;
  &.collapsed { width: var(--sidebar-collapsed-width); }
  @include respond-to(sm) {
    top: 50px; 
    width: 280px; 
    max-width: 85vw; 
    height: calc(100vh - 50px);
    transform: translateX(-100%);
    z-index: 1002;
    background: #ffffff;
    backdrop-filter: none;
    -webkit-backdrop-filter: none;
    &.collapsed { transform: translateX(-100%); }
    &:not(.collapsed) { 
      transform: translateX(0); 
      box-shadow: none;
      border-right: 1px solid var(--theme-border);
      opacity: 1;
      visibility: visible;
    }
  }
  .nav-section {
    margin-bottom: 8px;
    .nav-section-title { padding: 14px 20px 8px; font-size: 12px; font-weight: 700; color: #909399; }
    .nav-item {
      display: flex; 
      align-items: center; 
      min-height: 42px;
      padding: 0 20px; 
      color: var(--theme-text); 
      text-decoration: none;
      transition: background-color 0.16s ease, color 0.16s ease;
      position: relative;
      pointer-events: auto;
      z-index: 1;
      .nav-icon { margin-right: 12px; font-size: 18px; width: 20px; flex-shrink: 0; }
      .nav-badge {
        position: absolute;
        right: 8px;
        top: 50%;
        transform: translateY(-50%);
      }
      &:hover { background: var(--sidebar-hover-bg, #f5f7fa); color: var(--theme-primary); }
      &.active { background: var(--theme-primary); color: white; }
      &.admin-back { background: #fff7e6; color: #faad14; }
      @include respond-to(sm) { 
        padding: 14px 20px; 
        min-height: 48px;
        cursor: pointer;
        -webkit-tap-highlight-color: rgba(0,0,0,0.1);
        &.active { background: #ecf5ff; color: var(--theme-primary); border-left: 4px solid var(--theme-primary); }
      }
    }
  }
  .mobile-menu-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 20px;
    border-bottom: 1px solid var(--theme-border);
    background: var(--sidebar-bg-color, white);
    position: sticky;
    top: 0;
    z-index: 10;
    .menu-title {
      font-size: 16px;
      font-weight: 600;
      color: var(--theme-text);
    }
    .menu-close-btn {
      background: none;
      border: none;
      padding: 8px;
      cursor: pointer;
      color: var(--theme-text);
      font-size: 20px;
      display: flex;
      align-items: center;
      justify-content: center;
      min-width: 44px;
      min-height: 44px;
      touch-action: manipulation;
      -webkit-tap-highlight-color: rgba(0,0,0,0.1);
      &:hover {
        background: var(--sidebar-hover-bg, #f5f7fa);
        border-radius: 4px;
      }
    }
  }
  .sidebar-nav {
    @include respond-to(sm) {
      overflow-y: auto;
      -webkit-overflow-scrolling: touch;
      height: calc(100% - 60px);
      padding-bottom: 20px;
    }
  }
}
.main-content {
  flex: 1;
  margin-left: var(--sidebar-width);
  margin-top: var(--header-height);
  width: calc(100% - var(--sidebar-width));
  height: calc(100vh - var(--header-height));
  overflow-y: auto;
  transition: margin-left 0.2s ease, width 0.2s ease;
  .sidebar-collapsed & {
    margin-left: var(--sidebar-collapsed-width);
    width: calc(100% - var(--sidebar-collapsed-width));
  }
  @include respond-to(sm) { margin: 50px 0 0 0 !important; width: 100% !important; height: calc(100vh - 50px); }
  .content-wrapper {
    padding: var(--content-padding);
    @include respond-to(sm) { padding: 12px; padding-bottom: 72px; }
  }
}
.mobile-nav-bar {
  margin-bottom: 12px; background: #fff; border: 1px solid #dcdfe6; border-radius: 8px;
  .mobile-nav-header {
    width: 100%;
    min-height: 48px;
    display: flex;
    justify-content: space-between;
    padding: 14px 16px;
    align-items: center;
    border: 0;
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: pointer;
    touch-action: manipulation;
  }
  .mobile-nav-header:focus-visible {
    outline: 2px solid var(--theme-primary, #409eff);
    outline-offset: -2px;
  }
  .nav-expand-icon.expanded { transform: rotate(180deg); }
  .mobile-nav-menu { border-top: 1px solid #e4e7ed; max-height: 60vh; overflow-y: auto; }
  .nav-section-menu {
    .section-title { padding: 12px 16px; font-size: 12px; font-weight: 600; color: #909399; }
    .nav-menu-item {
      display: flex; align-items: center; padding: 14px 16px; gap: 12px;
      cursor: pointer; transition: background-color 0.16s ease, color 0.16s ease; text-decoration: none; color: inherit;
      .nav-icon { font-size: 18px; width: 20px; flex-shrink: 0; }
      .active-check-icon { margin-left: auto; color: var(--theme-primary); }
      &:hover { background: #f5f7fa; }
      &.active { background: #ecf5ff; color: var(--theme-primary); }
      &.admin-back { background: #fff7e6; color: #faad14; }
    }
  }
}
.mobile-tabbar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 1000;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-around;
  background: var(--theme-background, #fff);
  border-top: 1px solid var(--theme-border, #e8e8e8);
  padding-bottom: env(safe-area-inset-bottom);
}
.mobile-tab {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  flex: 1;
  padding: 6px 0;
  cursor: pointer;
  color: #999;
  text-decoration: none;
  transition: color 0.2s;
  .tab-icon { width: 22px; height: 22px; }
  &.active { color: var(--theme-primary, #409EFF); }
}
.mobile-tab-label { font-size: 10px; line-height: 1; }
.mobile-overlay {
  position: fixed; 
  inset: 50px 0 0 0; 
  background: rgba(15, 23, 42, 0.48); 
  z-index: 1001; 
  backdrop-filter: none;
  -webkit-backdrop-filter: none;
  pointer-events: auto;
}
.slide-down-enter-active, .slide-down-leave-active { transition: opacity 0.2s ease, max-height 0.2s ease, transform 0.2s ease; }
.slide-down-enter-from, .slide-down-leave-to { opacity: 0; max-height: 0; transform: translateY(-10px); }
</style>
