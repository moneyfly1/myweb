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
          <div class="nav-section-title" v-show="!sidebarCollapsed || isMobile">{{ section.title }}</div>
          <router-link 
            v-for="item in section.items" :key="item.path" :to="item.path"
            class="nav-item" :class="{ active: isRouteActive(item.path) }" @click="handleNavClick">
            <i :class="item.icon"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">{{ item.title }}</span>
          </router-link>
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
                <div class="section-title">{{ section.title }}</div>
                <div v-for="item in section.items" :key="item.path" 
                  class="nav-menu-item" :class="{ 'active': isRouteActive(item.path) }"
                  @click="navigateTo(item.path)">
                  <i :class="item.icon"></i>
                  <span>{{ item.title }}</span>
                  <i class="el-icon-check" v-if="isRouteActive(item.path)"></i>
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
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/store/auth'
import { useThemeStore } from '@/store/theme'
import { adminAPI } from '@/utils/api'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const themeStore = useThemeStore()

// --- 状态定义 ---
const isMobile = ref(false)
const sidebarCollapsed = ref(true)
const mobileNavExpanded = ref(false)
const stats = ref({ users: 0, subscriptions: 0, revenue: 0 })

// --- 配置常量 ---
const statConfig = {
  users: { label: '用户', icon: 'el-icon-user', prefix: '' },
  subscriptions: { label: '订阅', icon: 'el-icon-connection', prefix: '' },
  revenue: { label: '收入', icon: 'el-icon-money', prefix: '¥' }
}

const adminMenuOptions = [
  { label: '个人资料', icon: 'el-icon-user', command: 'profile' },
  { label: '系统设置', icon: 'el-icon-setting', command: 'settings' },
  { label: '系统日志', icon: 'el-icon-document', command: 'logs' },
  { label: '退出登录', icon: 'el-icon-switch-button', command: 'logout', divided: true }
]

const menuSections = [
  { title: '概览', items: [{ path: '/admin/dashboard', title: '仪表盘', icon: 'el-icon-s-home' }] },
  { 
    title: '用户管理', 
    items: [
      { path: '/admin/users', title: '用户列表', icon: 'el-icon-user' },
      { path: '/admin/abnormal-users', title: '异常用户', icon: 'el-icon-warning' },
      { path: '/admin/config-update', title: '节点更新', icon: 'el-icon-refresh' },
      { path: '/admin/nodes', title: '节点管理', icon: 'el-icon-server' },
      { path: '/admin/subscriptions', title: '订阅管理', icon: 'el-icon-connection' }
    ] 
  },
  { 
    title: '订单管理', 
    items: [
      { path: '/admin/orders', title: '订单列表', icon: 'el-icon-shopping-cart-2' },
      { path: '/admin/packages', title: '套餐管理', icon: 'el-icon-goods' }
    ] 
  },
  {
    title: '系统管理',
    items: [
      { path: '/admin/config', title: '配置管理', icon: 'el-icon-setting' },
      { path: '/admin/payment-config', title: '支付配置', icon: 'el-icon-wallet' },
      { path: '/admin/email-queue', title: '邮件队列', icon: 'el-icon-message' },
      { path: '/admin/statistics', title: '数据统计', icon: 'el-icon-data-analysis' }
    ]
  },
  {
    title: '其他管理',
    items: [
      { path: '/admin/invites', title: '邀请管理', icon: 'el-icon-user-solid' },
      { path: '/admin/tickets', title: '工单管理', icon: 'el-icon-s-order' },
      { path: '/admin/coupons', title: '优惠券管理', icon: 'el-icon-ticket' },
      { path: '/admin/user-levels', title: '用户等级管理', icon: 'el-icon-medal' }
    ]
  }
]

// --- 计算属性 ---
const currentTheme = computed(() => themeStore.currentTheme)
const themes = computed(() => themeStore.availableThemes)
const admin = computed(() => authStore.user)
const adminAvatar = computed(() => admin.value?.avatar || '')
const adminInitials = computed(() => admin.value?.username?.substring(0, 2).toUpperCase() || '')
const showBreadcrumb = computed(() => route.meta.showBreadcrumb !== false)
const breadcrumbItems = computed(() => route.meta.breadcrumb || [])

const currentPageTitle = computed(() => {
  if (route.meta.title) return route.meta.title
  const allItems = menuSections.flatMap(s => s.items)
  return allItems.find(i => i.path === route.path)?.title || '管理后台'
})

// --- 方法 ---
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
  const routes = { profile: '/admin/profile', settings: '/admin/settings', logs: '/admin/system-logs' }
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

// --- 生命周期 & 监听 ---
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
})

onUnmounted(() => window.removeEventListener('resize', checkMobile))

const getCurrentThemeLabel = () => themes.value.find(t => t.value === currentTheme.value)?.label || '主题'
const getCurrentThemeColor = () => themes.value.find(t => t.value === currentTheme.value)?.color || '#409EFF'
</script>

<style scoped lang="scss">
@use '@/styles/global.scss' as *;

.admin-layout {
  display: flex;
  height: 100vh;
  background-color: var(--theme-background);
  overflow-x: hidden;
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
        i { color: var(--theme-primary); font-size: 20px; }
        span { font-size: 18px; font-weight: 600; }
        small { font-size: 12px; opacity: 0.7; }
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
    top: 50px; width: 280px; max-width: 85vw; height: calc(100vh - 50px);
    transform: translateX(-100%);
    &.collapsed { transform: translateX(-100%); }
    &:not(.collapsed) { transform: translateX(0); box-shadow: 2px 0 16px rgba(0,0,0,0.15); }
  }

  .nav-section {
    margin-bottom: 24px;
    .nav-section-title { padding: 12px 20px 8px; font-size: 12px; font-weight: 600; color: #909399; }
    .nav-item {
      display: flex; align-items: center; padding: 12px 20px; color: var(--theme-text); text-decoration: none;
      transition: 0.3s;
      i { margin-right: 12px; font-size: 18px; width: 20px; text-align: center; }
      &:hover { background: var(--sidebar-hover-bg, #f5f7fa); color: var(--theme-primary); }
      &.active { background: var(--theme-primary); color: white; }
      @include respond-to(sm) { 
        padding: 14px 20px; 
        &.active { background: #ecf5ff; color: var(--theme-primary); border-left: 4px solid var(--theme-primary); }
      }
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
    @include respond-to(sm) { padding: 12px; }
  }
}

.mobile-nav-bar {
  margin-bottom: 12px; background: #fff; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.08);
  .mobile-nav-header { display: flex; justify-content: space-between; padding: 14px 16px; align-items: center; }
  .nav-expand-icon.expanded { transform: rotate(180deg); }
  .mobile-nav-menu { border-top: 1px solid #e4e7ed; max-height: 60vh; overflow-y: auto; }
}

.mobile-overlay {
  position: fixed; inset: 50px 0 0 0; background: rgba(0,0,0,0.5); z-index: 1001; backdrop-filter: blur(2px);
}

.slide-down-enter-active, .slide-down-leave-active { transition: all 0.3s ease; }
.slide-down-enter-from, .slide-down-leave-to { opacity: 0; max-height: 0; transform: translateY(-10px); }
</style>