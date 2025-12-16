<template>
  <div class="admin-layout" :class="{ 'sidebar-collapsed': sidebarCollapsed }">
    <!-- 顶部导航栏 -->
    <header class="header">
      <div class="header-left">
        <button 
          class="menu-toggle" 
          @click.stop="toggleSidebar"
          type="button"
          aria-label="切换菜单"
        >
          <i :class="sidebarCollapsed ? 'el-icon-menu' : 'el-icon-close'"></i>
          <span class="menu-toggle-text">菜单</span>
        </button>
        <div class="logo" @click="$router.push('/admin/dashboard')">
          <img src="/vite.svg" alt="Logo" class="logo-img">
          <span class="logo-text" v-show="!sidebarCollapsed">CBoard 管理后台</span>
        </div>
      </div>
      
      <div class="header-center">
        <div class="quick-stats">
          <div class="stat-item">
            <i class="el-icon-user"></i>
            <span>{{ stats.users || 0 }}</span>
            <small>用户</small>
          </div>
          <div class="stat-item">
            <i class="el-icon-connection"></i>
            <span>{{ stats.subscriptions || 0 }}</span>
            <small>订阅</small>
          </div>
          <div class="stat-item">
            <i class="el-icon-money"></i>
            <span>¥{{ formatMoney(stats.revenue) }}</span>
            <small>收入</small>
          </div>
        </div>
      </div>
      
      <div class="header-right">
        <!-- 主题切换 -->
        <el-dropdown @command="handleThemeChange" class="theme-dropdown">
          <el-button type="text" class="theme-btn">
            <i class="el-icon-brush"></i>
            <span class="theme-text" :style="{ color: getCurrentThemeColor() }">{{ getCurrentThemeLabel() }}</span>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item 
                v-for="theme in themes" 
                :key="theme.value"
                :command="theme.value"
                :class="{ active: currentTheme === theme.value }"
              >
                <i class="el-icon-check" v-if="currentTheme === theme.value"></i>
                {{ theme.label }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        
        <!-- 管理员菜单 -->
        <el-dropdown @command="handleAdminCommand" class="admin-dropdown">
          <div class="admin-info">
            <el-avatar :size="32" :src="adminAvatar">
              {{ adminInitials }}
            </el-avatar>
            <span class="admin-name" v-show="!isMobile">{{ admin.username }}</span>
            <i class="el-icon-arrow-down"></i>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">
                <i class="el-icon-user"></i>
                个人资料
              </el-dropdown-item>
              <el-dropdown-item command="settings">
                <i class="el-icon-setting"></i>
                系统设置
              </el-dropdown-item>
              <el-dropdown-item command="logs">
                <i class="el-icon-document"></i>
                系统日志
              </el-dropdown-item>
              <el-dropdown-item divided command="logout">
                <i class="el-icon-switch-button"></i>
                退出登录
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </header>

    <!-- 侧边栏 -->
    <aside class="sidebar" :class="{ collapsed: sidebarCollapsed }">
      <!-- 移动端菜单标题 -->
      <div class="mobile-menu-header" v-if="isMobile">
        <span class="menu-title">菜单</span>
        <button 
          class="menu-close-btn" 
          @click.stop="toggleSidebar"
          type="button"
          aria-label="关闭菜单"
        >
          <i class="el-icon-close"></i>
        </button>
      </div>
      <nav class="sidebar-nav">
        <div class="nav-section">
          <div class="nav-section-title" v-show="!sidebarCollapsed || isMobile">概览</div>
          <router-link 
            to="/admin/dashboard"
            class="nav-item"
            :class="{ active: $route.path === '/admin/dashboard' }"
            @click="handleNavClick"
          >
            <i class="el-icon-s-home"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">仪表盘</span>
          </router-link>
        </div>
        
        <div class="nav-section">
          <div class="nav-section-title" v-show="!sidebarCollapsed || isMobile">用户管理</div>
          <router-link 
            to="/admin/users"
            class="nav-item"
            :class="{ active: $route.path === '/admin/users' }"
            @click="handleNavClick"
          >
            <i class="el-icon-user"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">用户列表</span>
          </router-link>
          <router-link 
            to="/admin/abnormal-users"
            class="nav-item"
            :class="{ active: $route.path === '/admin/abnormal-users' }"
            @click="handleNavClick"
          >
            <i class="el-icon-warning"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">异常用户</span>
          </router-link>
          <router-link 
            to="/admin/config-update"
            class="nav-item"
            :class="{ active: $route.path === '/admin/config-update' }"
            @click="handleNavClick"
          >
            <i class="el-icon-refresh"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">配置更新</span>
          </router-link>
          <a 
            href="#"
            class="nav-item"
            :class="{ active: $route.path === '/admin/subscriptions' || $route.path.startsWith('/admin/subscriptions') }"
            @click.prevent="goToSubscriptions"
          >
            <i class="el-icon-connection"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">订阅管理</span>
          </a>
        </div>
        
        <div class="nav-section">
          <div class="nav-section-title" v-show="!sidebarCollapsed || isMobile">订单管理</div>
          <router-link 
            to="/admin/orders"
            class="nav-item"
            :class="{ active: $route.path === '/admin/orders' }"
            @click="handleNavClick"
          >
            <i class="el-icon-shopping-cart-2"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">订单列表</span>
          </router-link>
          <router-link 
            to="/admin/packages"
            class="nav-item"
            :class="{ active: $route.path === '/admin/packages' }"
            @click="handleNavClick"
          >
            <i class="el-icon-goods"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">套餐管理</span>
          </router-link>
        </div>
        
        <div class="nav-section">
          <div class="nav-section-title" v-show="!sidebarCollapsed || isMobile">系统管理</div>
          <router-link 
            to="/admin/notifications"
            class="nav-item"
            :class="{ active: $route.path === '/admin/notifications' }"
            @click="handleNavClick"
          >
            <i class="el-icon-bell"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">通知管理</span>
          </router-link>
          <router-link 
            to="/admin/config"
            class="nav-item"
            :class="{ active: $route.path === '/admin/config' }"
            @click="handleNavClick"
          >
            <i class="el-icon-setting"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">配置管理</span>
          </router-link>
          <router-link 
            to="/admin/payment-config"
            class="nav-item"
            :class="{ active: $route.path === '/admin/payment-config' }"
            @click="handleNavClick"
          >
            <i class="el-icon-wallet"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">支付配置</span>
          </router-link>
          <router-link 
            to="/admin/email-queue"
            class="nav-item"
            :class="{ active: $route.path === '/admin/email-queue' }"
            @click="handleNavClick"
          >
            <i class="el-icon-message"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">邮件队列</span>
          </router-link>
          <router-link 
            to="/admin/statistics"
            class="nav-item"
            :class="{ active: $route.path === '/admin/statistics' }"
            @click="handleNavClick"
          >
            <i class="el-icon-data-analysis"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">数据统计</span>
          </router-link>
        </div>
        
        <div class="nav-section">
          <div class="nav-section-title" v-show="!sidebarCollapsed || isMobile">其他管理</div>
          <router-link 
            to="/admin/invites"
            class="nav-item"
            :class="{ active: $route.path === '/admin/invites' }"
            @click="handleNavClick"
          >
            <i class="el-icon-user-solid"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">邀请管理</span>
          </router-link>
          <router-link 
            to="/admin/tickets"
            class="nav-item"
            :class="{ active: $route.path === '/admin/tickets' }"
            @click="handleNavClick"
          >
            <i class="el-icon-s-order"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">工单管理</span>
          </router-link>
          <router-link 
            to="/admin/coupons"
            class="nav-item"
            :class="{ active: $route.path === '/admin/coupons' }"
            @click="handleNavClick"
          >
            <i class="el-icon-ticket"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">优惠券管理</span>
          </router-link>
          <router-link 
            to="/admin/user-levels"
            class="nav-item"
            :class="{ active: $route.path === '/admin/user-levels' }"
            @click="handleNavClick"
          >
            <i class="el-icon-medal"></i>
            <span class="nav-text" v-show="!sidebarCollapsed || isMobile">用户等级管理</span>
          </router-link>
        </div>
      </nav>
    </aside>

    <!-- 主内容区域 -->
    <main class="main-content">
      <div class="content-wrapper">
        <!-- 移动端导航栏（可展开/折叠） -->
        <div class="mobile-nav-bar" v-if="isMobile">
          <div class="mobile-nav-header" @click="mobileNavExpanded = !mobileNavExpanded">
            <div class="nav-current-path">
              <i class="el-icon-location"></i>
              <span class="current-title">{{ route.meta.title || getCurrentPageTitle() }}</span>
            </div>
            <i 
              class="el-icon-arrow-down nav-expand-icon" 
              :class="{ 'expanded': mobileNavExpanded }"
            ></i>
          </div>
          <transition name="slide-down">
            <div class="mobile-nav-menu" v-show="mobileNavExpanded">
              <div 
                v-for="section in menuSections" 
                :key="section.title"
                class="nav-section-menu"
              >
                <div class="section-title">{{ section.title }}</div>
                <div 
                  v-for="item in section.items" 
                  :key="item.path"
                  class="nav-menu-item"
                  :class="{ 'active': $route.path === item.path }"
                  @click="handleNavMenuClick(item.path)"
                >
                  <i :class="item.icon"></i>
                  <span>{{ item.title }}</span>
                  <i class="el-icon-check" v-if="$route.path === item.path"></i>
                </div>
              </div>
            </div>
          </transition>
        </div>
        
        <!-- 桌面端面包屑导航 -->
        <div class="breadcrumb" v-if="showBreadcrumb && !isMobile">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item 
              v-for="item in breadcrumbItems" 
              :key="item.path"
              :to="item.path"
            >
              {{ item.title }}
            </el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        
        <!-- 页面内容 -->
        <div class="page-content">
          <router-view />
        </div>
      </div>
    </main>

    <!-- 移动端遮罩 -->
    <div 
      v-if="isMobile && !sidebarCollapsed" 
      class="mobile-overlay"
      @click.stop="sidebarCollapsed = true"
      @touchstart.stop
    />
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

// 响应式数据
// 从localStorage读取侧边栏状态（桌面端），移动端默认隐藏
const getInitialSidebarState = () => {
  if (typeof window !== 'undefined') {
    const isMobileCheck = window.innerWidth <= 768
    if (isMobileCheck) {
      return true // 移动端默认隐藏
    }
    const saved = localStorage.getItem('sidebarCollapsed')
    return saved === 'true' ? true : false
  }
  return false
}
const sidebarCollapsed = ref(getInitialSidebarState())
const stats = ref({})
const isMobile = ref(false)
const mobileNavExpanded = ref(false)

// 计算属性
const themeStore = useThemeStore()
const currentTheme = computed(() => themeStore.currentTheme)
const themes = computed(() => themeStore.availableThemes)
const admin = computed(() => authStore.user)
const adminAvatar = computed(() => admin.value?.avatar || '')
const adminInitials = computed(() => {
  if (!admin.value?.username) return ''
  return admin.value.username.substring(0, 2).toUpperCase()
})

const showBreadcrumb = computed(() => {
  return route.meta.showBreadcrumb !== false
})

const breadcrumbItems = computed(() => {
  const items = []
  if (route.meta.breadcrumb) {
    items.push(...route.meta.breadcrumb)
  }
  return items
})

// 获取当前主题标签
const getCurrentThemeLabel = () => {
  const theme = themes.value.find(t => t.value === currentTheme.value)
  return theme ? theme.label : '主题'
}

// 获取当前主题颜色
const getCurrentThemeColor = () => {
  const theme = themes.value.find(t => t.value === currentTheme.value)
  return theme ? theme.color : '#409EFF'
}

// 方法
const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
  // 保存到localStorage（仅在桌面端）
  if (!isMobile.value) {
    localStorage.setItem('sidebarCollapsed', sidebarCollapsed.value.toString())
  }
}

const handleThemeChange = async (themeName) => {
  const result = await themeStore.setTheme(themeName)
  if (!result.success) {
    ElMessage.warning(result.message || '主题保存失败，仅本地生效')
  } else {
    ElMessage.success('主题已保存到云端')
  }
}

const handleAdminCommand = (command) => {
  switch (command) {
    case 'profile':
      router.push('/admin/profile')
      break
    case 'settings':
      router.push('/admin/settings')
      break
    case 'logs':
      router.push('/admin/system-logs')
      break
    case 'logout':
      authStore.logout()
      router.push('/admin/login')
      break
  }
}

// 获取当前页面标题
const getCurrentPageTitle = () => {
  const titleMap = {
    '/admin/dashboard': '仪表盘',
    '/admin/users': '用户列表',
    '/admin/subscriptions': '订阅管理',
    '/admin/orders': '订单列表',
    '/admin/packages': '套餐管理',
    '/admin/abnormal-users': '异常用户',
    '/admin/notifications': '通知管理',
    '/admin/config': '配置管理',
    '/admin/payment-config': '支付配置',
    '/admin/statistics': '数据统计',
    '/admin/email-queue': '邮件队列',
    '/admin/config-update': '配置更新',
    '/admin/tickets': '工单管理',
    '/admin/coupons': '优惠券管理'
  }
  return titleMap[route.path] || '管理后台'
}

// 菜单分组
const menuSections = computed(() => {
  return [
    {
      title: '概览',
      items: [
        { path: '/admin/dashboard', title: '仪表盘', icon: 'el-icon-s-home' }
      ]
    },
    {
      title: '用户管理',
      items: [
        { path: '/admin/users', title: '用户列表', icon: 'el-icon-user' },
        { path: '/admin/abnormal-users', title: '异常用户', icon: 'el-icon-warning' },
        { path: '/admin/subscriptions', title: '订阅管理', icon: 'el-icon-connection' },
        { path: '/admin/config-update', title: '配置更新', icon: 'el-icon-refresh' }
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
        { path: '/admin/notifications', title: '通知管理', icon: 'el-icon-bell' },
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
})

// 处理移动端导航菜单点击
const handleNavMenuClick = (path) => {
  router.push(path)
  mobileNavExpanded.value = false
  // 移动端点击菜单项后自动关闭侧边栏
  if (isMobile.value) {
    sidebarCollapsed.value = true
  }
}

// 处理侧边栏导航项点击
const handleNavClick = (event) => {
  // 移动端点击导航项后自动关闭侧边栏
  if (isMobile.value) {
    sidebarCollapsed.value = true
  }
  // 确保路由跳转正常进行，不阻止默认行为
  // router-link 会自动处理导航，这里只负责关闭侧边栏
  // 不调用 event.preventDefault() 或 event.stopPropagation()
}

// 专门处理订阅管理导航（使用编程式导航）
const goToSubscriptions = () => {
  // 移动端点击后自动关闭侧边栏
  if (isMobile.value) {
    sidebarCollapsed.value = true
  }
  // 使用编程式导航
  router.push('/admin/subscriptions').then(() => {
    }).catch(err => {
    // 如果路由跳转失败，尝试直接跳转
    if (err.name !== 'NavigationDuplicated') {
      window.location.href = '/admin/subscriptions'
    }
  })
}

const formatMoney = (value) => {
  if (value === null || value === undefined || value === '') return '0.00'
  const num = typeof value === 'string' ? parseFloat(value) : value
  if (isNaN(num)) return '0.00'
  return num.toFixed(2)
}

const loadStats = async () => {
  try {
    // 检查是否已认证
    if (!authStore.isAuthenticated) {
      stats.value = { users: 0, subscriptions: 0, revenue: 0 }
      return
    }
    
    const response = await adminAPI.getDashboard()
    if (response.data && response.data.success && response.data.data) {
      const data = response.data.data
      // 映射后端字段到前端字段
      stats.value = {
        users: Number(data.totalUsers) || 0,
        subscriptions: Number(data.activeSubscriptions) || 0,
        revenue: Number(data.totalRevenue) || 0
      }
    } else {
      stats.value = { users: 0, subscriptions: 0, revenue: 0 }
    }
  } catch (error) {
    console.error('加载统计数据失败:', error)
    stats.value = { users: 0, subscriptions: 0, revenue: 0 }
  }
}

const checkMobile = () => {
  const wasMobile = isMobile.value
  isMobile.value = window.innerWidth <= 768
  // 移动端时，侧边栏默认隐藏（需要点击按钮显示）
  // 桌面端时，恢复之前的状态
  if (isMobile.value) {
    // 移动端：侧边栏默认隐藏
    sidebarCollapsed.value = true
  } else {
    // 从移动端切换到桌面端时，恢复桌面端状态
    if (wasMobile) {
      const savedState = localStorage.getItem('sidebarCollapsed')
      if (savedState !== null) {
        sidebarCollapsed.value = savedState === 'true'
      } else {
        sidebarCollapsed.value = false // 桌面端默认展开
      }
    }
  }
}

// 监听路由变化，自动折叠移动端导航栏
watch(() => route.path, () => {
  if (isMobile.value) {
    mobileNavExpanded.value = false
    // 移动端路由切换时自动关闭侧边栏
    sidebarCollapsed.value = true
  }
})

// 生命周期
onMounted(() => {
  checkMobile()
  loadStats()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})
</script>

<style scoped lang="scss">
@use '@/styles/global.scss' as *;
.admin-layout {
  display: flex;
  height: 100vh;
  background-color: var(--theme-background);
  overflow-x: hidden;
  
  @include respond-to(sm) {
    position: relative;
    width: 100%;
    overflow-x: hidden;
  }
}

.header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: var(--header-height);
  background: var(--theme-background);
  border-bottom: 1px solid var(--theme-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  z-index: 1000;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  color: var(--theme-text);
  
  // 根据主题调整阴影
  .theme-dark &,
  .theme-aurora & {
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  }
  
  @include respond-to(sm) {
    padding: 0 12px;
    height: 50px;
  }
  
  .header-left {
    display: flex;
    align-items: center;
    gap: 16px;
    flex: 0 0 auto;
    
    @include respond-to(sm) {
      gap: 8px;
      flex: 1;
      justify-content: flex-start; // 确保内容靠左对齐
    }
    
    .logo {
      display: flex;
      align-items: center;
      gap: 8px;
      cursor: pointer;
      flex: 1;
      min-width: 0;
      
      @include respond-to(sm) {
        gap: 6px;
        flex: 1;
        min-width: 0;
      }
      
      .logo-img {
        width: 32px;
        height: 32px;
        flex-shrink: 0;
        
        @include respond-to(sm) {
          width: 24px;
          height: 24px;
        }
      }
      
      .logo-text {
        font-size: 18px;
        font-weight: 600;
        color: var(--theme-primary);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        
        @include respond-to(sm) {
          font-size: 14px;
          max-width: 100px;
        }
      }
    }
    
    .menu-toggle {
      display: none;
      cursor: pointer;
      padding: 8px;
      border-radius: 4px;
      font-size: 20px;
      color: var(--theme-text);
      
      &:hover {
        background-color: var(--theme-hover-bg, rgba(0, 0, 0, 0.05));
        
        .theme-dark &,
        .theme-aurora & {
          background-color: rgba(255, 255, 255, 0.1);
        }
      }
      z-index: 1003;
      position: relative;
      background: none;
      border: none;
      -webkit-tap-highlight-color: transparent;
      touch-action: manipulation;
      
      &:active {
        background-color: var(--theme-hover-bg, rgba(0, 0, 0, 0.1));
        opacity: 0.8;
        
        .theme-dark &,
        .theme-aurora & {
          background-color: rgba(255, 255, 255, 0.15);
        }
      }
      
      :is(i) {
        pointer-events: none;
      }
      
      .menu-toggle-text {
        display: none;
      }
      
      @include respond-to(sm) {
        display: flex !important;
        align-items: center;
        justify-content: center;
        gap: 6px;
        padding: 8px 12px;
        font-size: 16px;
        min-width: auto;
        min-height: 44px;
        flex-shrink: 0;
        background-color: var(--theme-background);
        border: 1px solid var(--theme-border);
        border-radius: 6px;
        color: var(--theme-text);
        font-weight: 500;
        transition: all 0.2s ease;
        
        .menu-toggle-text {
          font-size: 14px;
          color: var(--theme-text);
          font-weight: 500;
          white-space: nowrap;
          display: inline-block !important;
        }
        
        :is(i) {
          font-size: 18px;
          color: var(--theme-text);
          flex-shrink: 0;
        }
        
        &:hover {
          background-color: var(--theme-hover-bg, rgba(0, 0, 0, 0.05));
          border-color: var(--theme-primary);
          
          .theme-dark &,
          .theme-aurora & {
            background-color: rgba(255, 255, 255, 0.1);
            border-color: var(--theme-primary);
          }
        }
        
        &:active {
          background-color: var(--theme-hover-bg, rgba(0, 0, 0, 0.1));
          transform: scale(0.98);
          
          .theme-dark &,
          .theme-aurora & {
            background-color: rgba(255, 255, 255, 0.15);
          }
        }
      }
    }
  }
  
  .header-center {
    flex: 1;
    display: flex;
    justify-content: center;
    
    @include respond-to(sm) {
      display: none;
    }
    
    .quick-stats {
      display: flex;
      gap: 24px;
      
      @include respond-to(sm) {
        display: none;
      }
      
      .stat-item {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 4px;
        
        :is(i) {
          font-size: 20px;
          color: var(--theme-primary);
        }
        
        span {
          font-size: 18px;
          font-weight: 600;
          color: var(--theme-text);
        }
        
        small {
          font-size: 12px;
          color: var(--theme-text);
          opacity: 0.7;
        }
      }
    }
  }
  
  .header-right {
    display: flex;
    align-items: center;
    gap: 16px;
    flex: 0 0 auto;
    
    @include respond-to(sm) {
      gap: 8px;
    }
    
    .theme-dropdown {
      .el-button {
        padding: 8px 12px;
        border-radius: 4px;
        display: flex;
        align-items: center;
        gap: 6px;
        color: var(--theme-text);
        
        @include respond-to(sm) {
          padding: 6px 10px;
          font-size: 14px;
          gap: 4px;
        }
        
        .theme-text {
          font-size: 14px;
          font-weight: 500;
          transition: color 0.3s ease;
          
          @include respond-to(sm) {
            font-size: 13px;
          }
        }
        
        :is(i) {
          font-size: 16px;
          
          @include respond-to(sm) {
            font-size: 14px;
          }
        }
        
        &:hover {
          background-color: var(--theme-hover-bg, rgba(0, 0, 0, 0.05));
          
          .theme-dark &,
          .theme-aurora & {
            background-color: rgba(255, 255, 255, 0.1);
          }
        }
      }
    }
    
    .admin-dropdown {
      .admin-info {
        display: flex;
        align-items: center;
        gap: 8px;
        cursor: pointer;
        padding: 4px 8px;
        border-radius: 4px;
        
        @include respond-to(sm) {
          gap: 4px;
          padding: 2px 4px;
        }
        
        &:hover {
          background-color: var(--theme-hover-bg, rgba(0, 0, 0, 0.05));
          
          .theme-dark &,
          .theme-aurora & {
            background-color: rgba(255, 255, 255, 0.1);
          }
        }
        
        .admin-name {
          font-weight: 500;
          color: var(--theme-text);
          
          @include respond-to(sm) {
            display: none;
          }
        }
        
        :is(i) {
          @include respond-to(sm) {
            display: none;
          }
        }
        
        :deep(.el-avatar) {
          @include respond-to(sm) {
            width: 28px !important;
            height: 28px !important;
          }
        }
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
  background: var(--sidebar-bg-color, white);
  border-right: 1px solid var(--theme-border);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  z-index: 999;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
  
  &.collapsed {
    width: var(--sidebar-collapsed-width);
  }
  
  @include respond-to(sm) {
    // 移动端：侧边栏作为覆盖层，不影响主内容布局
    position: fixed;
    top: 50px;
    left: 0;
    width: 280px;
    max-width: 85vw;
    height: calc(100vh - 50px);
    z-index: 1002;
    background: #ffffff;
    box-shadow: 2px 0 16px rgba(0, 0, 0, 0.15);
    transform: translateX(-100%);
    transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
    border-right: 1px solid #e4e7ed;
    
    // 当侧边栏展开时（collapsed=false）
    &:not(.collapsed) {
      transform: translateX(0);
    }
    
    // 当侧边栏折叠时（collapsed=true）- 完全隐藏
    &.collapsed {
      transform: translateX(-100%);
    }
    
    /* 移动端菜单标题 */
    .mobile-menu-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 16px 20px;
      background: #f5f7fa;
      border-bottom: 1px solid #e4e7ed;
      box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
      
      .menu-title {
        font-size: 18px;
        font-weight: 700;
        color: #303133;
        letter-spacing: 0.5px;
      }
      
      .menu-close-btn {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 36px;
        height: 36px;
        border-radius: 8px;
        background: #ffffff;
        border: 1px solid #dcdfe6;
        color: #606266;
        font-size: 18px;
        font-weight: normal;
        cursor: pointer;
        transition: all 0.2s ease;
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
        
        &:hover {
          background: #f5f7fa;
          border-color: #c0c4cc;
          color: #303133;
        }
        
        &:active {
          background: #e4e7ed;
          transform: scale(0.95);
        }
      }
    }
  }
  
  .sidebar-nav {
    padding: 20px 0;
    
    @include respond-to(sm) {
      padding: 12px 0;
    }
    
    .nav-section {
      margin-bottom: 24px;
      
      @include respond-to(sm) {
        margin-bottom: 16px;
      }
      
      .nav-section-title {
        padding: 0 20px 8px;
        font-size: 12px;
        font-weight: 600;
        color: #666;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        
        @include respond-to(sm) {
          padding: 12px 20px 8px;
          font-size: 13px;
          color: #909399;
          font-weight: 700;
        }
      }
      
      .nav-item {
        display: flex;
        align-items: center;
        padding: 12px 20px;
        color: var(--sidebar-text-color, var(--theme-text));
        text-decoration: none;
        transition: all 0.3s ease;
        position: relative;
        cursor: pointer;
        user-select: none;
        -webkit-tap-highlight-color: transparent;
        touch-action: manipulation;
        pointer-events: auto;
        z-index: 1;
        
        @include respond-to(sm) {
          padding: 14px 20px;
          font-size: 15px;
          min-height: 48px;
          color: #303133;
          font-weight: 500;
        }
        
        &:active {
          background-color: var(--sidebar-hover-bg, #f5f7fa);
          
          @include respond-to(sm) {
            background-color: #f5f7fa;
          }
        }
        
        &:hover {
          background-color: var(--sidebar-hover-bg, #f5f7fa);
          color: var(--theme-primary);
          
          @include respond-to(sm) {
            background-color: #f5f7fa;
            color: var(--theme-primary);
          }
        }
        
        &.active {
          background-color: var(--sidebar-active-bg, var(--theme-primary));
          color: white;
          font-weight: 600;
          
          &::before {
            content: '';
            position: absolute;
            left: 0;
            top: 0;
            bottom: 0;
            width: 3px;
            background-color: var(--theme-primary);
            
            @include respond-to(sm) {
              width: 4px;
              background-color: var(--theme-primary);
            }
          }
          
          @include respond-to(sm) {
            background-color: #ecf5ff;
            color: var(--theme-primary);
            
            &::before {
              background-color: var(--theme-primary);
            }
          }
        }
        
        :is(i) {
          font-size: 18px;
          margin-right: 12px;
          width: 20px;
          text-align: center;
          flex-shrink: 0;
          
          @include respond-to(sm) {
            font-size: 20px;
            margin-right: 14px;
            width: 24px;
            color: inherit;
          }
        }
        
        .nav-text {
          font-weight: 500;
          flex: 1;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
          
          @include respond-to(sm) {
            color: inherit;
          }
        }
      }
    }
  }
}

.main-content {
  flex: 1;
  margin-left: var(--sidebar-width);
  margin-top: var(--header-height);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  width: calc(100% - var(--sidebar-width));
  
  .sidebar-collapsed & {
    margin-left: var(--sidebar-collapsed-width);
    width: calc(100% - var(--sidebar-collapsed-width));
  }
  
  @include respond-to(sm) {
    // 移动端：主内容区域始终占据全屏，不受侧边栏影响
    position: relative;
    margin-left: 0 !important;
    margin-top: 50px;
    width: 100% !important;
    left: 0 !important;
    padding-left: 0 !important;
    padding-right: 0 !important;
    z-index: 1;
  }
  
  .content-wrapper {
    padding: var(--content-padding);
    width: 100%;
    box-sizing: border-box;
    margin-left: 0;
    // 确保背景色与页面一致，让内容紧贴左侧
    background: transparent;
    
    @include respond-to(sm) {
      padding: 12px;
      padding-bottom: 20px;
      margin-left: 0 !important;
      width: 100% !important;
      max-width: 100% !important;
    }
    
      .breadcrumb {
        margin-bottom: 20px;
        padding: 12px 16px;
        background: white;
        border-radius: var(--border-radius);
        box-shadow: var(--box-shadow);
      }
      
      /* 移动端导航栏 */
      .mobile-nav-bar {
        margin-bottom: 12px;
        background: #ffffff;
        border-radius: 10px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08);
        overflow: hidden;
        width: 100%;
        box-sizing: border-box;
        
        .mobile-nav-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 14px 16px;
          cursor: pointer;
          user-select: none;
          -webkit-tap-highlight-color: transparent;
          transition: background-color 0.2s;
          min-height: 48px;
          
          &:active {
            background-color: #f5f7fa;
          }
          
          .nav-current-path {
            display: flex;
            align-items: center;
            gap: 10px;
            flex: 1;
            min-width: 0;
            
            :is(i) {
              font-size: 18px;
              color: #409eff;
              flex-shrink: 0;
            }
            
            .current-title {
              font-size: 15px;
              font-weight: 600;
              color: #303133;
              white-space: nowrap;
              overflow: hidden;
              text-overflow: ellipsis;
            }
          }
          
          .nav-expand-icon {
            font-size: 16px;
            color: #909399;
            transition: transform 0.3s ease;
            flex-shrink: 0;
            margin-left: 8px;
            
            &.expanded {
              transform: rotate(180deg);
            }
          }
        }
        
        .mobile-nav-menu {
          border-top: 1px solid #e4e7ed;
          max-height: calc(100vh - 200px);
          overflow-y: auto;
          -webkit-overflow-scrolling: touch;
          
          .nav-section-menu {
            padding: 8px 0;
            
            .section-title {
              padding: 10px 16px 8px;
              font-size: 12px;
              font-weight: 600;
              color: #909399;
              text-transform: uppercase;
              letter-spacing: 0.5px;
            }
            
            .nav-menu-item {
              display: flex;
              align-items: center;
              padding: 12px 16px 12px 40px;
              cursor: pointer;
              transition: background-color 0.2s;
              -webkit-tap-highlight-color: transparent;
              position: relative;
              
              &:active {
                background-color: #f5f7fa;
              }
              
              &.active {
                background-color: #ecf5ff;
                color: var(--theme-primary);
                
                &::before {
                  content: '';
                  position: absolute;
                  left: 0;
                  top: 0;
                  bottom: 0;
                  width: 3px;
                  background-color: var(--theme-primary);
                }
              }
              
              i:first-child {
                font-size: 18px;
                margin-right: 12px;
                width: 20px;
                text-align: center;
              }
              
              span {
                flex: 1;
                font-size: 14px;
                font-weight: 500;
              }
              
              i.el-icon-check {
                font-size: 16px;
                color: var(--theme-primary);
                margin-left: auto;
              }
            }
          }
        }
      }
    
    .page-content {
      min-height: calc(100vh - var(--header-height) - 80px);
      width: 100%;
      
      @include respond-to(sm) {
        min-height: calc(100vh - 50px - 60px);
        width: 100% !important;
        margin-left: 0 !important;
        padding-left: 0 !important;
        padding-right: 0 !important;
      }
    }
  }
}

.mobile-overlay {
  position: fixed;
  top: 50px;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 1001;
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
  animation: fadeIn 0.3s ease;
  touch-action: none;
  -webkit-tap-highlight-color: transparent;
  
  @include respond-to(sm) {
    top: 50px;
    z-index: 1001;
  }
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

/* 滑动展开动画 */
.slide-down-enter-active {
  transition: all 0.3s ease-out;
}

.slide-down-leave-active {
  transition: all 0.3s ease-in;
}

.slide-down-enter-from {
  opacity: 0;
  max-height: 0;
  transform: translateY(-10px);
}

.slide-down-leave-to {
  opacity: 0;
  max-height: 0;
  transform: translateY(-10px);
}
</style> 