<template>
  <div class="admin-layout" :class="{ 'sidebar-collapsed': sidebarCollapsed, 'is-mobile': isMobile }">
    <header class="header">
      <div class="header-left">
        <button class="menu-toggle" @click.stop="toggleSidebar" type="button" aria-label="切换菜单">
          <i :class="sidebarCollapsed ? 'el-icon-menu' : 'el-icon-close'"></i>
          <span class="menu-toggle-text">菜单</span>
        </button>
        <div class="logo" @click="navigateTo('/admin/dashboard')">
          <img src="/vite.svg" alt="Logo" class="logo-img">
          <span class="logo-text" v-show="!sidebarCollapsed">CBoard 管理后台</span>
        </div>
      </div>
      <div class="header-center">
        <div class="quick-stats">
          <div v-for="(val, key) in statConfig" :key="key" class="stat-item">
            <i :class="val.icon"></i>
            <span>{{ val.prefix }}{{ stats[key] || 0 }}</span>
            <small>{{ val.label }}</small>
          </div>
        </div>
      </div>
      <div class="header-right">
        <el-dropdown @command="handleThemeChange" class="theme-dropdown">
          <el-button type="text" class="theme-btn">
            <i class="el-icon-brush"></i>
            <span class="theme-text" :style="{ color: getCurrentThemeColor() }">{{ getCurrentThemeLabel() }}</span>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item 
                v-for="theme in themes" :key="theme.value" :command="theme.value"
                :class="{ active: currentTheme === theme.value }">
                <i class="el-icon-check" v-if="currentTheme === theme.value"></i>
                {{ theme.label }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-dropdown @command="handleAdminCommand" class="admin-dropdown">
          <div class="admin-info">
            <el-avatar :size="32" :src="adminAvatar">{{ adminInitials }}</el-avatar>
            <span class="admin-name" v-show="!isMobile">{{ admin.username }}</span>
            <i class="el-icon-arrow-down"></i>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-for="item in adminMenuOptions" :key="item.command" :command="item.command" :divided="item.divided">
                <i :class="item.icon"></i> {{ item.label }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </header>
    <aside class="sidebar" :class="{ collapsed: sidebarCollapsed }">
      <div class="mobile-menu-header" v-if="isMobile">
        <span class="menu-title">菜单</span>
        <button class="menu-close-btn" @click.stop="toggleSidebar" type="button"><i class="el-icon-close"></i></button>
      </div>
      <nav class="sidebar-nav">
        <div v-for="section in menuSections" :key="section.title" class="nav-section">
          <div
            class="nav-section-title"
            v-show="!sidebarCollapsed || isMobile"
            :class="{ collapsible: section.collapsible, collapsed: collapsedSections[section.title] }"
            @click="section.collapsible && toggleSection(section.title)"
          >
            <span>{{ section.title }}</span>
            <i v-if="section.collapsible" class="el-icon-arrow-down section-arrow"></i>
          </div>
          <div class="nav-items" :class="{ 'is-collapsed': section.collapsible && collapsedSections[section.title] && (!sidebarCollapsed || isMobile) }">
            <router-link
              v-for="item in section.items" :key="item.path" :to="item.path"
              class="nav-item" :class="{ active: isRouteActive(item.path) }" @click="handleNavClick">
              <i :class="item.icon"></i>
              <span class="nav-text" v-show="!sidebarCollapsed || isMobile">{{ item.title }}</span>
              <el-badge
                v-if="item.badge && item.badge > 0 && (!sidebarCollapsed || isMobile)"
                :value="item.badge"
                :max="99"
                type="danger"
                class="nav-badge"
              />
            </router-link>
          </div>
        </div>
      </nav>
    </aside>
    <main class="main-content">
      <div class="content-wrapper">
        <div class="mobile-nav-bar" v-if="isMobile">
          <div class="mobile-nav-header" @click="mobileNavExpanded = !mobileNavExpanded">
            <div class="nav-current-path">
              <i class="el-icon-location"></i>
              <span class="current-title">{{ currentPageTitle }}</span>
            </div>
            <i class="el-icon-arrow-down nav-expand-icon" :class="{ 'expanded': mobileNavExpanded }"></i>
          </div>
          <transition name="slide-down">
            <div class="mobile-nav-menu" v-show="mobileNavExpanded">
              <div v-for="section in menuSections" :key="section.title" class="nav-section-menu">
                <div
                  class="section-title"
                  :class="{ collapsible: section.collapsible, collapsed: collapsedSections[section.title] }"
                  @click="section.collapsible && toggleSection(section.title)"
                >
                  <span>{{ section.title }}</span>
                  <i v-if="section.collapsible" class="el-icon-arrow-down section-arrow"></i>
                </div>
                <div class="mobile-nav-items" :class="{ 'is-collapsed': section.collapsible && collapsedSections[section.title] }">
                  <div v-for="item in section.items" :key="item.path"
                    class="nav-menu-item" :class="{ 'active': isRouteActive(item.path) }"
                    @click="navigateTo(item.path)">
                    <i :class="item.icon"></i>
                    <span>{{ item.title }}</span>
                    <i class="el-icon-check" v-if="isRouteActive(item.path)"></i>
                  </div>
                </div>
              </div>
            </div>
          </transition>
        </div>
        <div class="breadcrumb" v-if="showBreadcrumb && !isMobile">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item v-for="item in breadcrumbItems" :key="item.path" :to="item.path">{{ item.title }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="page-content">
          <router-view />
        </div>
      </div>
    </main>
    <div v-if="isMobile && !sidebarCollapsed" class="mobile-overlay" @click.stop="sidebarCollapsed = true" />
    <div v-if="isMobile" class="mobile-tabbar">
      <div class="mobile-tab" :class="{ active: isRouteActive('/admin/dashboard') }" @click="navigateTo('/admin/dashboard')">
        <svg viewBox="0 0 512 512" class="tab-icon"><rect x="48" y="48" width="176" height="176" rx="20" ry="20" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/><rect x="288" y="48" width="176" height="176" rx="20" ry="20" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/><rect x="48" y="288" width="176" height="176" rx="20" ry="20" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/><rect x="288" y="288" width="176" height="176" rx="20" ry="20" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/></svg>
        <span class="mobile-tab-label">仪表盘</span>
      </div>
      <div class="mobile-tab" :class="{ active: isRouteActive('/admin/users') }" @click="navigateTo('/admin/users')">
        <svg viewBox="0 0 512 512" class="tab-icon"><path d="M402 168c-2.93 40.67-33.1 72-66 72s-63.12-31.32-66-72c-3-42.31 26.37-72 66-72s69 30.46 66 72z" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/><path d="M336 304c-65.17 0-127.84 32.37-143.54 95.41-2.08 8.34 3.15 16.59 11.72 16.59h263.65c8.57 0 13.77-8.25 11.72-16.59C463.85 336.36 401.18 304 336 304z" fill="none" stroke="currentColor" stroke-miterlimit="10" stroke-width="32"/><path d="M200 185.94c-2.34 32.48-26.72 58.06-53 58.06s-50.7-25.57-53-58.06C91.61 152.15 115.34 128 147 128s55.39 24.77 53 57.94z" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/><path d="M206 306c-18.05-8.27-37.93-11.45-59-11.45-52 0-102.1 25.85-114.65 76.2-1.65 6.64 2.53 13.25 9.37 13.25H154" fill="none" stroke="currentColor" stroke-linecap="round" stroke-miterlimit="10" stroke-width="32"/></svg>
        <span class="mobile-tab-label">用户</span>
      </div>
      <div class="mobile-tab" :class="{ active: isRouteActive('/admin/subscriptions') }" @click="navigateTo('/admin/subscriptions')">
        <svg viewBox="0 0 512 512" class="tab-icon"><path d="M400 240c-8.89-89.54-71-144-144-144-69 0-113.44 48.2-128 96-60 6-112 43.59-112 112 0 66 54 112 120 112h260c55 0 100-27.44 100-88 0-59.82-53-85.76-96-88z" fill="none" stroke="currentColor" stroke-linejoin="round" stroke-width="32"/></svg>
        <span class="mobile-tab-label">订阅</span>
      </div>
      <div class="mobile-tab" :class="{ active: isRouteActive('/admin/orders') }" @click="navigateTo('/admin/orders')">
        <svg viewBox="0 0 512 512" class="tab-icon"><path d="M80 176a16 16 0 00-16 16v216a16 16 0 0016 16h352a16 16 0 0016-16V192a16 16 0 00-16-16zm-8-48h368" fill="none" stroke="currentColor" stroke-linejoin="round" stroke-width="32"/><rect x="64" y="176" width="384" height="256" rx="16" ry="16" fill="none" stroke="currentColor" stroke-linejoin="round" stroke-width="32"/><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M160 240l48 48 96-96"/></svg>
        <span class="mobile-tab-label">订单</span>
      </div>
    </div>
  </div>
</template>
<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/store/auth'
import { useThemeStore } from '@/store/theme'
import { adminAPI, ticketAPI } from '@/utils/api'
const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const isMobile = ref(false)
const sidebarCollapsed = ref(true)
const mobileNavExpanded = ref(false)
const stats = ref({ users: 0, subscriptions: 0, revenue: 0 })
const statConfig = {
  users: { label: '用户', icon: 'el-icon-user', prefix: '' },
  subscriptions: { label: '订阅', icon: 'el-icon-connection', prefix: '' },
  revenue: { label: '收入', icon: 'el-icon-money', prefix: '¥' }
}
const adminMenuOptions = [
  { label: '个人资料', icon: 'el-icon-user', command: 'profile' },
  { label: '系统设置', icon: 'el-icon-setting', command: 'settings' },
  { label: '退出登录', icon: 'el-icon-switch-button', command: 'logout', divided: true }
]
const unreadTicketCount = ref(0)
const collapsedSections = ref({})
let unreadCheckInterval = null
const toggleSection = (title) => {
  collapsedSections.value[title] = !collapsedSections.value[title]
}
const loadUnreadTicketCount = async () => {
  try {
    const response = await ticketAPI.getUnreadCount()
    if (response.data && response.data.success) {
      unreadTicketCount.value = response.data.data?.count || 0
    }
  } catch (error) {
    // 未读消息数加载失败，不影响主功能
  }
}
const menuSections = computed(() => {
  const baseSections = [
    { title: '概览', collapsible: false, items: [{ path: '/admin/dashboard', title: '仪表盘', icon: 'el-icon-s-home' }] },
    {
      title: '用户管理',
      collapsible: true,
      items: [
        { path: '/admin/users', title: '用户列表', icon: 'el-icon-user' },
        { path: '/admin/abnormal-users', title: '异常用户', icon: 'el-icon-warning' },
        { path: '/admin/user-levels', title: '用户等级', icon: 'el-icon-medal' },
        { path: '/admin/invites', title: '邀请管理', icon: 'el-icon-user-solid' },
        {
          path: '/admin/tickets',
          title: '工单管理',
          icon: 'el-icon-s-order',
          badge: unreadTicketCount.value > 0 ? unreadTicketCount.value : null
        }
      ]
    },
    {
      title: '业务管理',
      collapsible: true,
      items: [
        { path: '/admin/subscriptions', title: '订阅管理', icon: 'el-icon-connection' },
        { path: '/admin/orders', title: '订单列表', icon: 'el-icon-shopping-cart-2' },
        { path: '/admin/packages', title: '套餐管理', icon: 'el-icon-goods' },
        { path: '/admin/coupons', title: '优惠券管理', icon: 'el-icon-ticket' }
      ]
    },
    {
      title: '节点管理',
      collapsible: true,
      items: [
        { path: '/admin/nodes', title: '节点列表', icon: 'el-icon-server' },
        { path: '/admin/custom-nodes', title: '专线节点', icon: 'el-icon-connection' },
        { path: '/admin/config-update', title: '节点更新', icon: 'el-icon-refresh' }
      ]
    },
    {
      title: '系统配置',
      collapsible: true,
      items: [
        { path: '/admin/config', title: '配置管理', icon: 'el-icon-setting' },
        { path: '/admin/payment-config', title: '支付配置', icon: 'el-icon-wallet' },
        { path: '/admin/email-queue', title: '邮件队列', icon: 'el-icon-message' }
      ]
    },
    {
      title: '数据与日志',
      collapsible: true,
      items: [
        { path: '/admin/statistics', title: '数据统计', icon: 'el-icon-data-analysis' },
        { path: '/admin/logs', title: '日志管理', icon: 'el-icon-document' },
        { path: '/admin/system-logs', title: '系统日志', icon: 'el-icon-tickets' }
      ]
    }
  ]
  return baseSections
})
const currentTheme = computed(() => themeStore.currentTheme)
const themes = computed(() => themeStore.availableThemes)
const admin = computed(() => authStore.user)
const adminAvatar = computed(() => admin.value?.avatar || '')
const adminInitials = computed(() => admin.value?.username?.substring(0, 2).toUpperCase() || '')
const showBreadcrumb = computed(() => route.meta.showBreadcrumb !== false)
const breadcrumbItems = computed(() => route.meta.breadcrumb || [])
const currentPageTitle = computed(() => {
  if (route.meta.title) return route.meta.title
  const allItems = menuSections.value.flatMap(s => s.items)
  return allItems.find(i => i.path === route.path)?.title || '管理后台'
})
const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
  if (!isMobile.value) localStorage.setItem('sidebarCollapsed', sidebarCollapsed.value)
}
const isRouteActive = (path) => route.path === path || (path !== '/admin/dashboard' && route.path.startsWith(path))
const navigateTo = (path) => {
  router.push(path)
  mobileNavExpanded.value = false
  if (isMobile.value) sidebarCollapsed.value = true
}
const handleNavClick = () => {
  if (isMobile.value) sidebarCollapsed.value = true
}
const handleThemeChange = async (themeName) => {
  const result = await themeStore.setTheme(themeName)
  result.success ? ElMessage.success('主题已保存') : ElMessage.warning(result.message || '保存失败')
}
const handleAdminCommand = (command) => {
  const routes = { profile: '/admin/profile', settings: '/admin/settings' }
  if (command === 'logout') {
    authStore.logout()
    router.push('/admin/login')
  } else {
    router.push(routes[command])
  }
}
const formatMoney = (val) => isNaN(parseFloat(val)) ? '0.00' : parseFloat(val).toFixed(2)
const loadStats = async () => {
  if (!authStore.isAuthenticated) return
  try {
    const { data } = await adminAPI.getDashboard()
    if (data?.success && data.data) {
      stats.value = {
        users: Number(data.data.totalUsers) || 0,
        subscriptions: Number(data.data.activeSubscriptions) || 0,
        revenue: formatMoney(data.data.totalRevenue)
      }
    }
  } catch (e) {
    console.error('Stats load error:', e)
  }
}
const checkMobile = () => {
  isMobile.value = window.innerWidth <= 768
  if (isMobile.value) {
    sidebarCollapsed.value = true
  } else {
    const saved = localStorage.getItem('sidebarCollapsed')
    sidebarCollapsed.value = saved === 'true'
  }
}
watch(() => route.path, () => {
  if (isMobile.value) {
    mobileNavExpanded.value = false
    sidebarCollapsed.value = true
  }
})
onMounted(() => {
  checkMobile()
  loadStats()
  window.addEventListener('resize', checkMobile)
  loadUnreadTicketCount()
  unreadCheckInterval = setInterval(() => {
    loadUnreadTicketCount()
  }, 30000)
  window.addEventListener('ticket-viewed', loadUnreadTicketCount)
  watch(() => route.path, (newPath) => {
    if (newPath === '/admin/tickets') {
      loadUnreadTicketCount()
    }
  })
})
onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
  window.removeEventListener('ticket-viewed', loadUnreadTicketCount)
  if (unreadCheckInterval) {
    clearInterval(unreadCheckInterval)
    unreadCheckInterval = null
  }
})
const getCurrentThemeLabel = () => themes.value.find(t => t.value === currentTheme.value)?.label || '主题'
const getCurrentThemeColor = () => themes.value.find(t => t.value === currentTheme.value)?.color || '#409EFF'
</script>
<style scoped lang="scss">
@use '@/styles/global.scss' as *;
.admin-layout {
  display: flex;
  height: 100vh;
  background-color: var(--theme-background);
  overflow-x: clip;
}
.header {
  position: fixed;
  top: 0; left: 0; right: 0;
  height: var(--header-height);
  background: var(--theme-background);
  border-bottom: 1px solid var(--theme-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  z-index: 1000;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  @include respond-to(sm) { height: 50px; padding: 0 12px; }
  .header-left {
    display: flex; align-items: center; gap: 16px;
    .logo { display: flex; align-items: center; gap: 8px; cursor: pointer; }
    .logo-text { font-size: 18px; font-weight: 600; color: var(--theme-primary); }
    .menu-toggle {
      display: none; border: 1px solid var(--theme-border); background: none; padding: 8px 12px; border-radius: 6px;
      @include respond-to(sm) { display: flex; align-items: center; gap: 6px; }
    }
  }
  .header-center {
    @include respond-to(sm) { display: none; }
    .quick-stats { display: flex; gap: 24px; }
    .stat-item { 
        display: flex; flex-direction: column; align-items: center;
        :is(i) { color: var(--theme-primary); font-size: 20px; }
        :is(span) { font-size: 18px; font-weight: 600; }
        :is(small) { font-size: 12px; opacity: 0.7; }
    }
  }
}
.sidebar {
  position: fixed;
  top: var(--header-height);
  left: 0;
  width: var(--sidebar-width);
  height: calc(100vh - var(--header-height));
  background: var(--sidebar-bg-color, white);
  border-right: 1px solid var(--theme-border);
  transition: all 0.3s ease;
  z-index: 999;
  overflow-y: auto;
  &.collapsed { width: var(--sidebar-collapsed-width); }
  @include respond-to(sm) {
    top: 50px; 
    width: 280px; 
    max-width: 85vw; 
    height: calc(100vh - 50px);
    transform: translateX(-100%);
    z-index: 1002; /* 确保菜单在遮罩层之上 */
    background: #ffffff; /* 确保背景是纯白色，不透明 */
    backdrop-filter: none; /* 移除模糊效果 */
    -webkit-backdrop-filter: none;
    &.collapsed { transform: translateX(-100%); }
    &:not(.collapsed) { 
      transform: translateX(0); 
      box-shadow: 2px 0 16px rgba(0,0,0,0.15);
      opacity: 1;
      visibility: visible;
    }
  }
  .nav-section {
    margin-bottom: 8px;
    .nav-section-title {
      padding: 12px 20px 8px;
      font-size: 12px;
      font-weight: 600;
      color: #909399;
      user-select: none;
      &.collapsible {
        cursor: pointer;
        display: flex;
        justify-content: space-between;
        align-items: center;
        border-radius: 4px;
        margin: 0 8px;
        padding: 10px 12px;
        &:hover { color: var(--theme-primary); background: var(--sidebar-hover-bg, #f5f7fa); }
        .section-arrow {
          font-size: 12px;
          transition: transform 0.3s ease;
        }
        &.collapsed .section-arrow {
          transform: rotate(-90deg);
        }
      }
    }
    .nav-items {
      overflow: hidden;
      max-height: 500px;
      transition: max-height 0.3s ease, opacity 0.3s ease;
      opacity: 1;
      &.is-collapsed {
        max-height: 0;
        opacity: 0;
      }
    }
    .nav-item {
      display: flex; 
      align-items: center; 
      padding: 12px 20px; 
      color: var(--theme-text); 
      text-decoration: none;
      transition: 0.3s;
      position: relative;
      pointer-events: auto;
      z-index: 1;
      :is(i) { margin-right: 12px; font-size: 18px; width: 20px; text-align: center; }
      .nav-badge {
        position: absolute;
        right: 8px;
        top: 50%;
        transform: translateY(-50%);
      }
      &:hover { background: var(--sidebar-hover-bg, #f5f7fa); color: var(--theme-primary); }
      &.active { background: var(--theme-primary); color: white; }
      @include respond-to(sm) { 
        padding: 14px 20px; 
        min-height: 48px; /* 确保点击区域足够大 */
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
      min-width: 40px;
      min-height: 40px;
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
  transition: 0.3s;
  .sidebar-collapsed & { 
    margin-left: var(--sidebar-collapsed-width); 
    width: calc(100% - var(--sidebar-collapsed-width));
  }
  @include respond-to(sm) { margin: 50px 0 0 0 !important; width: 100% !important; }
  .content-wrapper {
    padding: var(--content-padding);
    @include respond-to(sm) { padding: 12px; padding-bottom: 72px; }
  }
}
.mobile-nav-bar {
  margin-bottom: 12px; background: #fff; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.08);
  .mobile-nav-header { display: flex; justify-content: space-between; padding: 14px 16px; align-items: center; }
  .nav-expand-icon.expanded { transform: rotate(180deg); }
  .mobile-nav-menu { border-top: 1px solid #e4e7ed; max-height: 60vh; overflow-y: auto; }
  .nav-section-menu {
    .section-title {
      &.collapsible {
        cursor: pointer;
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding-right: 16px;
        &:hover { color: var(--theme-primary); }
        .section-arrow {
          font-size: 12px;
          transition: transform 0.3s ease;
        }
        &.collapsed .section-arrow {
          transform: rotate(-90deg);
        }
      }
    }
    .mobile-nav-items {
      overflow: hidden;
      max-height: 500px;
      transition: max-height 0.3s ease, opacity 0.3s ease;
      opacity: 1;
      &.is-collapsed {
        max-height: 0;
        opacity: 0;
      }
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
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.06);
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
  transition: color 0.2s;
  .tab-icon { width: 22px; height: 22px; }
  &.active { color: var(--theme-primary, #409EFF); }
}
.mobile-tab-label { font-size: 10px; line-height: 1; }
.mobile-overlay {
  position: fixed; 
  inset: 50px 0 0 0; 
  background: rgba(0,0,0,0.5); 
  z-index: 1001; 
  backdrop-filter: blur(2px);
  pointer-events: auto;
}
.slide-down-enter-active, .slide-down-leave-active { transition: all 0.3s ease; }
.slide-down-enter-from, .slide-down-leave-to { opacity: 0; max-height: 0; transform: translateY(-10px); }
</style>