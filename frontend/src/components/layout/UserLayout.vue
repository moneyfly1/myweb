<template>
  <div class="user-layout" :class="{ 'sidebar-collapsed': sidebarCollapsed }">
    <!-- 顶部导航栏（仅桌面端显示，移动端保持原有顶部导航） -->
    <header class="header" v-if="!isMobile">
      <div class="header-left">
        <div class="logo" @click="$router.push('/dashboard')">
          <i class="el-icon-s-home"></i>
          <span class="logo-text" v-show="!sidebarCollapsed">CBoard</span>
        </div>
        <button 
          class="menu-toggle" 
          @click.stop="toggleSidebar"
          type="button"
          aria-label="切换菜单"
        >
          <i :class="sidebarCollapsed ? 'el-icon-menu' : 'el-icon-close'"></i>
          <span class="toggle-text">{{ sidebarCollapsed ? '展开' : '折叠' }}</span>
        </button>
      </div>
      
      <div class="header-right">
        <!-- 主题切换 -->
        <el-dropdown @command="handleThemeChange" class="theme-dropdown" v-if="!isMobile">
          <el-button type="text" class="theme-btn">
            <i class="el-icon-brush"></i>
            <span class="theme-text">主题</span>
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
        
        <!-- 用户菜单 -->
        <el-dropdown @command="handleUserCommand" class="user-dropdown">
          <div class="user-info">
            <el-avatar :size="32" :src="userAvatar">
              {{ userInitials }}
            </el-avatar>
            <span class="user-name" v-show="!sidebarCollapsed">{{ user.username }}</span>
            <i class="el-icon-arrow-down"></i>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-if="hasAdminAccess" command="backToAdmin" divided>
                <i class="el-icon-back"></i>
                返回管理员后台
              </el-dropdown-item>
              <el-dropdown-item command="profile">
                <i class="el-icon-user"></i>
                个人资料
              </el-dropdown-item>
              <el-dropdown-item command="settings">
                <i class="el-icon-setting"></i>
                基础设置
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
    
    <!-- 移动端顶部导航栏 -->
    <header class="header mobile-header" v-if="isMobile">
      <div class="mobile-header-container">
        <!-- 左侧：Logo和菜单按钮 -->
        <div class="mobile-header-left">
          <button 
            class="mobile-menu-toggle" 
            @click.stop="toggleMobileNav"
            type="button"
            aria-label="切换菜单"
          >
            <i :class="mobileNavExpanded ? 'el-icon-close' : 'el-icon-menu'"></i>
            <span class="toggle-text">{{ mobileNavExpanded ? '收起' : '菜单' }}</span>
          </button>
          <div class="mobile-logo">
            <div class="logo-icon">
              <i class="el-icon-s-home"></i>
            </div>
            <span class="logo-text">CBoard</span>
          </div>
        </div>
        
        <!-- 右侧：主题切换和用户信息 -->
        <div class="mobile-header-right">
          <!-- 移动端主题切换 -->
          <el-dropdown @command="handleThemeChange" class="mobile-theme-dropdown">
            <el-button type="text" class="mobile-theme-btn">
              <i class="el-icon-brush"></i>
              <span class="theme-text">主题</span>
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
          
          <el-dropdown @command="handleUserCommand" class="mobile-user-dropdown">
            <div class="mobile-user-info">
              <el-avatar :size="36" :src="userAvatar">
                {{ userInitials }}
              </el-avatar>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-if="hasAdminAccess" command="backToAdmin" divided>
                  <i class="el-icon-back"></i>
                  返回管理员后台
                </el-dropdown-item>
                <el-dropdown-item command="profile">
                  <i class="el-icon-user"></i>
                  个人资料
                </el-dropdown-item>
                <el-dropdown-item command="settings">
                  <i class="el-icon-setting"></i>
                  基础设置
                </el-dropdown-item>
                <el-dropdown-item divided command="logout">
                  <i class="el-icon-switch-button"></i>
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
      
      <!-- 移动端导航菜单 - 从左侧弹出 -->
      <div class="nav-menu mobile" :class="{ active: mobileNavExpanded }" v-if="isMobile">
        <!-- 菜单标题 -->
        <div class="mobile-menu-header">
          <span class="menu-title">菜单</span>
          <button 
            class="menu-close-btn" 
            @click.stop="toggleMobileNav"
            type="button"
            aria-label="关闭菜单"
          >
            <i class="el-icon-close"></i>
          </button>
        </div>
        <!-- 菜单项 -->
        <div class="mobile-menu-content">
          <router-link
            v-for="item in menuItems"
            :key="item.path"
            :to="item.path"
            class="nav-link"
            :class="{ 
              active: $route.path === item.path,
              'admin-back': item.isAdminBack
            }"
            @click="mobileNavExpanded = false"
          >
            <i :class="item.icon"></i>
            <span>{{ item.title }}</span>
          </router-link>
        </div>
      </div>
    </header>
    
    <!-- 左侧导航栏（仅桌面端） -->
    <aside class="sidebar" :class="{ collapsed: sidebarCollapsed }" v-if="!isMobile">
      <nav class="sidebar-nav">
        <div class="nav-section">
          <div class="nav-section-title" v-show="!sidebarCollapsed">用户中心</div>
          <template v-for="item in menuItems" :key="item.path">
            <router-link 
              v-if="!item.isAdminBack"
              :to="item.path"
              class="nav-item"
              :class="{ 
                active: $route.path === item.path
              }"
            >
              <i :class="item.icon"></i>
              <span class="nav-text" v-show="!sidebarCollapsed">{{ item.title }}</span>
            </router-link>
            <div
              v-else
              class="nav-item admin-back"
              @click="returnToAdmin"
              style="cursor: pointer;"
            >
              <i :class="item.icon"></i>
              <span class="nav-text" v-show="!sidebarCollapsed">{{ item.title }}</span>
            </div>
          </template>
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
                v-for="item in menuItems" 
                :key="item.path"
                class="nav-menu-item"
                :class="{ 
                  'active': $route.path === item.path,
                  'admin-back': item.isAdminBack
                }"
                @click="handleNavMenuClick(item)"
              >
                <i :class="item.icon"></i>
                <span>{{ item.title }}</span>
                <i class="el-icon-check" v-if="$route.path === item.path"></i>
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


  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/store/auth'
import { useThemeStore } from '@/store/theme'
import { ElMessage } from 'element-plus'
import { secureStorage } from '@/utils/secureStorage'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

// 响应式数据
// 从localStorage读取侧边栏状态（桌面端），移动端默认隐藏
const getInitialSidebarState = () => {
  if (typeof window !== 'undefined') {
    const isMobileCheck = window.innerWidth <= 992
    if (isMobileCheck) {
      return true // 移动端默认隐藏
    }
    const saved = localStorage.getItem('userSidebarCollapsed')
    return saved === 'true' ? true : false
  }
  return false
}
const sidebarCollapsed = ref(getInitialSidebarState())
const mobileNavExpanded = ref(false)
const isMobile = ref(false)

// 计算属性
const themeStore = useThemeStore()
const currentTheme = computed(() => themeStore.currentTheme)
const themes = computed(() => themeStore.availableThemes)
const user = computed(() => authStore.user)
const userAvatar = computed(() => user.value?.avatar || '')
const userInitials = computed(() => {
  if (!user.value?.username) return ''
  return user.value.username.substring(0, 2).toUpperCase()
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

// 菜单项
const menuItems = computed(() => {
  const items = [
    {
      path: '/dashboard',
      title: '仪表盘',
      icon: 'el-icon-s-home',
      badge: null
    },
    {
      path: '/subscription',
      title: '订阅管理',
      icon: 'el-icon-connection',
      badge: null
    },
    {
      path: '/devices',
      title: '设备管理',
      icon: 'el-icon-mobile-phone',
      badge: null
    },
    {
      path: '/packages',
      title: '套餐购买',
      icon: 'el-icon-shopping-cart-2',
      badge: null
    },
    {
      path: '/orders',
      title: '订单记录',
      icon: 'el-icon-document',
      badge: null
    },
    {
      path: '/nodes',
      title: '节点列表',
      icon: 'el-icon-location',
      badge: null
    },
    {
      path: '/tickets',
      title: '工单中心',
      icon: 'el-icon-s-ticket',
      badge: null
    },
    {
      path: '/invites',
      title: '我的邀请',
      icon: 'el-icon-user',
      badge: null
    },
    {
      path: '/help',
      title: '帮助中心',
      icon: 'el-icon-question',
      badge: null
    }
  ]
  
  // 如果有管理员信息，添加返回管理员后台的选项
  if (secureStorage.get('admin_token') && secureStorage.get('admin_user')) {
    items.push({
      path: '#admin-back',
      title: '回到管理员后台',
      icon: 'el-icon-back',
      badge: null,
      isAdminBack: true
    })
  }
  
  return items
})

// 方法
const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
  // 保存到localStorage（仅在桌面端）
  if (!isMobile.value) {
    localStorage.setItem('userSidebarCollapsed', sidebarCollapsed.value.toString())
  }
}

const toggleMobileNav = () => {
  mobileNavExpanded.value = !mobileNavExpanded.value
}

const closeMobileNav = () => {
  mobileNavExpanded.value = false
}

// 处理导航点击
const handleNavClick = (event) => {
  // 不阻止默认行为，让 router-link 正常工作
  // 移动端点击后关闭导航菜单
  if (isMobile.value) {
    closeMobileNav()
    mobileNavExpanded.value = false
  }
  // 桌面端点击导航项后不关闭侧边栏，保持展开状态
  // 让 router-link 正常处理导航，不调用 preventDefault 或 stopPropagation
}

// 处理移动端导航菜单点击
const handleNavMenuClick = (item) => {
  if (item.isAdminBack) {
    returnToAdmin()
    return
  }
  handleNavClick(item)
}

// 获取当前页面标题
const getCurrentPageTitle = () => {
  const titleMap = {
    '/dashboard': '仪表盘',
    '/subscription': '订阅管理',
    '/devices': '设备管理',
    '/packages': '套餐购买',
    '/orders': '订单记录',
    '/nodes': '节点列表',
    '/tickets': '工单中心',
    '/help': '帮助中心',
    '/profile': '个人资料',
    '/settings': '设置'
  }
  return titleMap[route.path] || '用户中心'
}

// 返回管理员后台
const returnToAdmin = () => {
  const adminToken = secureStorage.get('admin_token')
  const adminUser = secureStorage.get('admin_user')
  
  if (adminToken && adminUser) {
    try {
      // 解析管理员用户数据（如果是字符串则解析）
      let adminUserData = adminUser
      if (typeof adminUserData === 'string') {
        try {
          adminUserData = JSON.parse(adminUserData)
        } catch (e) {
          ElMessage.error('管理员信息格式错误')
          return
        }
      }
      
      // 验证是否为管理员
      if (!adminUserData.is_admin) {
        ElMessage.error('该账户不是管理员')
        return
      }
      
      // 恢复管理员认证信息到 authStore（使用 localStorage）
      authStore.setAuth(adminToken, adminUserData, false)
      
      // 清除用户相关的 sessionStorage 信息
      secureStorage.remove('user_token')
      secureStorage.remove('user_data')
      secureStorage.remove('user_refresh_token')
      
      // 跳转到管理员后台
      router.push('/admin/dashboard')
      
      ElMessage.success('已返回管理员后台')
    } catch (error) {
      if (process.env.NODE_ENV === 'development') {
        console.error('返回管理员后台失败:', error)
      }
      ElMessage.error('返回管理员后台失败，请重新登录')
    }
  } else {
    ElMessage.warning('未找到管理员信息，请重新登录')
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

// 检查是否有管理员访问权限
const hasAdminAccess = computed(() => {
  const adminToken = secureStorage.get('admin_token')
  const adminUser = secureStorage.get('admin_user')
  return !!(adminToken && adminUser)
})

const handleUserCommand = (command) => {
  // 关闭移动端导航菜单
  if (isMobile.value) {
    mobileNavExpanded.value = false
  }
  
  switch (command) {
    case 'backToAdmin':
      returnToAdmin()
      break
    case 'profile':
      router.push('/profile')
      break
    case 'settings':
      router.push('/settings')
      break
    case 'logout':
      authStore.logout()
      router.push('/login')
      break
  }
}


const checkMobile = () => {
  const wasMobile = isMobile.value
  isMobile.value = window.innerWidth <= 992
  // 移动端时，侧边栏默认隐藏（需要点击按钮显示）
  // 桌面端时，恢复之前的状态
  if (isMobile.value) {
    // 移动端：侧边栏默认隐藏
    sidebarCollapsed.value = true
  } else {
    // 从移动端切换到桌面端时，恢复桌面端状态
    if (wasMobile) {
      const savedState = localStorage.getItem('userSidebarCollapsed')
      if (savedState !== null) {
        sidebarCollapsed.value = savedState === 'true'
      } else {
        sidebarCollapsed.value = false // 桌面端默认展开
      }
    }
  }
  // 如果从移动端切换到桌面端，关闭移动端菜单
  if (wasMobile && !isMobile.value) {
    mobileNavExpanded.value = false
  }
}

// 监听路由变化，自动折叠移动端导航栏
watch(() => route.path, () => {
  if (isMobile.value) {
    mobileNavExpanded.value = false
  }
})

// 生命周期
onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
  
  // 监听来自父窗口的消息
  window.addEventListener('message', handleMessage)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
  window.removeEventListener('message', handleMessage)
})

// 处理来自父窗口的消息
const handleMessage = (event) => {
  if (event.data.type === 'SET_AUTH') {
    // 设置认证信息
    localStorage.setItem('token', event.data.token)
    localStorage.setItem('user', JSON.stringify(event.data.user))
    
    // 保存管理员信息，用于返回管理员后台
    if (event.data.adminToken && event.data.adminUser) {
      localStorage.setItem('admin_token', event.data.adminToken)
      localStorage.setItem('admin_user', event.data.adminUser)
    }
    
    // 更新认证状态
    authStore.setAuth(event.data.token, event.data.user)
    
    ElMessage.success('已自动登录用户后台')
  }
}
</script>

<style scoped lang="scss">
@use '@/styles/global.scss' as *;

:root {
  --primary-color: #42a5f5;
  --secondary-color: #6777ef;
  --text-color: #1c273c;
  --bg-color: #f7fafd;
  --card-bg: #fff;
  --card-shadow: 0 2px 12px rgba(0,0,0,0.06);
  --sidebar-width: 240px !important;
  --sidebar-collapsed-width: 64px !important;
  --header-height: 70px;
  --content-padding: 20px;
}

.user-layout {
  min-height: 100vh;
  background-color: var(--bg-color);
  overflow-x: hidden;
}

/* 顶部导航栏样式 */
.header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: var(--header-height, 70px);
  background: linear-gradient(135deg, var(--theme-primary, #409EFF) 0%, var(--theme-primary, #409EFF) 100%);
  border-bottom: none;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  z-index: 1000;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  
  .sidebar-collapsed & {
    padding-left: 20px !important;
    
    .header-left {
      margin-left: 0 !important;
      padding-left: 0 !important;
    }
  }
  
  @include respond-to(sm) {
    padding: 0;
    height: auto;
    min-height: 64px;
    /* 确保不会遮挡安全区域 */
    padding-top: env(safe-area-inset-top, 0);
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
    border-bottom: none;
  }
  
      &.mobile-header {
        @include respond-to(sm) {
          display: block;
          background: linear-gradient(135deg, var(--theme-primary, #409EFF) 0%, var(--theme-primary, #409EFF) 100%);
          position: relative;
          overflow: visible;
          border-bottom: none;
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
          pointer-events: none;
          transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
          
          .mobile-header-container {
            pointer-events: auto;
          }
        }
      }
  
  /* 移动端顶部栏容器 */
  .mobile-header-container {
    @include respond-to(sm) {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 12px 16px;
      position: relative;
      z-index: 1001;
      min-height: 64px;
      /* 确保容器本身不可点击，只有内部元素可点击 */
      pointer-events: none;
      
      /* 恢复子元素的点击事件 */
      > * {
        pointer-events: auto;
      }
    }
  }
  
  .mobile-header-left {
    @include respond-to(sm) {
      display: flex;
      align-items: center;
      gap: 12px;
      flex: 1;
      /* 限制点击区域，避免上半部分误触 */
      min-height: 44px;
      max-height: 44px;
      align-self: center;
    }
  }
  
  .mobile-header-right {
    @include respond-to(sm) {
      display: flex;
      align-items: center;
      gap: 8px;
      flex: 0 0 auto;
    }
  }
  
      .mobile-menu-toggle {
        @include respond-to(sm) {
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 6px;
          padding: 8px 16px;
          min-width: 70px;
          height: 40px;
          border-radius: 25px;
          background: rgba(255, 255, 255, 0.25);
          border: 1px solid rgba(255, 255, 255, 0.3);
          color: #ffffff;
          font-size: 14px;
          font-weight: 600;
          cursor: pointer;
          transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
          flex-shrink: 0;
          position: relative;
          z-index: 1002;
          -webkit-tap-highlight-color: transparent;
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1), 0 2px 4px rgba(0, 0, 0, 0.06);
          backdrop-filter: blur(10px);
          -webkit-backdrop-filter: blur(10px);
          
          &:hover {
            background: rgba(255, 255, 255, 0.35);
            border-color: rgba(255, 255, 255, 0.5);
            box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15), 0 4px 8px rgba(0, 0, 0, 0.1);
            transform: translateY(-2px) scale(1.02);
          }
          
          &:active {
            background: rgba(255, 255, 255, 0.2);
            transform: scale(0.98);
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
          }
          
          :is(i) {
            pointer-events: none;
            color: #ffffff;
            font-weight: bold;
            font-size: 18px;
          }
          
          .toggle-text {
            font-size: 14px;
            font-weight: 600;
            color: #ffffff;
            white-space: nowrap;
            pointer-events: none;
          }
        }
      }
  
  .mobile-logo {
    @include respond-to(sm) {
      display: flex;
      align-items: center;
      gap: 10px;
      cursor: default;
      padding: 0;
      background: transparent;
      border: none;
      transition: all 0.3s ease;
      flex: 0 0 auto;
      min-width: 0;
      max-width: calc(100% - 60px);
      position: relative;
      z-index: 1002;
      overflow: hidden;
      pointer-events: none;
      
      .logo-icon {
        display: none;
      }
      
      .logo-text {
        font-size: 18px;
        font-weight: bold;
        color: #ffffff;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        letter-spacing: 0;
      }
    }
  }
  
  .mobile-theme-dropdown {
    @include respond-to(sm) {
      .mobile-theme-btn {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 6px;
        padding: 8px 16px;
        min-width: 60px;
        height: 40px;
        border-radius: 25px;
        background: rgba(255, 255, 255, 0.2);
        border: none;
        color: #ffffff;
        font-size: 14px;
        font-weight: 600;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1), 0 2px 4px rgba(0, 0, 0, 0.06);
        backdrop-filter: blur(10px);
        -webkit-backdrop-filter: blur(10px);
        
        &:hover {
          background: rgba(255, 255, 255, 0.3);
          color: #ffffff;
          box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15), 0 4px 8px rgba(0, 0, 0, 0.1);
          transform: translateY(-2px) scale(1.02);
        }
        
        :is(i) {
          color: #ffffff;
          font-weight: 500;
          font-size: 16px;
        }
        
        .theme-text {
          font-size: 14px;
          font-weight: 600;
          color: #ffffff;
          white-space: nowrap;
        }
      }
    }
  }
  
  .mobile-user-dropdown {
    @include respond-to(sm) {
      .mobile-user-info {
        display: flex;
        align-items: center;
        cursor: pointer;
        padding: 6px 12px;
        border-radius: 25px;
        background: rgba(255, 255, 255, 0.2);
        border: none;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1), 0 2px 4px rgba(0, 0, 0, 0.06);
        backdrop-filter: blur(10px);
        -webkit-backdrop-filter: blur(10px);
        
        &:hover {
          background: rgba(255, 255, 255, 0.3);
          box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15), 0 4px 8px rgba(0, 0, 0, 0.1);
          transform: translateY(-2px) scale(1.02);
        }
        
        :deep(.el-avatar) {
          border: none;
        }
        
        :deep(.el-avatar__inner) {
          color: #ffffff;
        }
      }
    }
  }
  
  .theme-dropdown {
    .theme-btn {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 6px;
      padding: 8px 18px;
      min-width: 70px;
      height: 40px;
      border-radius: 25px;
      color: #ffffff;
      background: rgba(255, 255, 255, 0.2);
      font-size: 14px;
      font-weight: 600;
      transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
      border: none;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1), 0 2px 4px rgba(0, 0, 0, 0.06);
      backdrop-filter: blur(10px);
      -webkit-backdrop-filter: blur(10px);
      
      &:hover {
        background: rgba(255, 255, 255, 0.3);
        color: #ffffff;
        box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15), 0 4px 8px rgba(0, 0, 0, 0.1);
        transform: translateY(-2px) scale(1.02);
      }
      
      :is(i) {
        color: #ffffff;
        font-weight: 500;
        font-size: 16px;
      }
      
      .theme-text {
        font-size: 14px;
        font-weight: 600;
        color: #ffffff;
        white-space: nowrap;
      }
    }
  }
  
  .header-left {
    display: flex;
    align-items: center;
    gap: 16px;
    flex: 0 0 auto;
    
    @include respond-to(sm) {
      display: none;
    }
    
    .logo {
      display: flex;
      align-items: center;
      gap: 8px;
      cursor: pointer;
      position: relative;
      z-index: 1002;
      
      :is(i) {
        font-size: 24px;
        color: #ffffff;
      }
      
      .logo-text {
        font-size: 18px;
        font-weight: 600;
        color: #ffffff;
      }
    }
    
    .menu-toggle {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 6px;
      cursor: pointer;
      padding: 10px 20px;
      border-radius: 25px;
      font-size: 14px;
      font-weight: 600;
      color: #ffffff;
      z-index: 1003;
      position: relative;
      background: rgba(255, 255, 255, 0.25);
      border: 1px solid rgba(255, 255, 255, 0.3);
      -webkit-tap-highlight-color: transparent;
      touch-action: manipulation;
      transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
      min-width: 80px;
      min-height: 40px;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1), 0 2px 4px rgba(0, 0, 0, 0.06);
      backdrop-filter: blur(10px);
      -webkit-backdrop-filter: blur(10px);
      
      &:hover {
        background: rgba(255, 255, 255, 0.35);
        border-color: rgba(255, 255, 255, 0.5);
        box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15), 0 4px 8px rgba(0, 0, 0, 0.1);
        transform: translateY(-2px) scale(1.02);
      }
      
      &:active {
        background: rgba(255, 255, 255, 0.2);
        transform: translateY(0) scale(0.98);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      }
      
      :is(i) {
        pointer-events: none;
        font-weight: bold;
        color: #ffffff;
        font-size: 18px;
      }
      
      .toggle-text {
        font-size: 14px;
        font-weight: 600;
        color: #ffffff;
        white-space: nowrap;
        pointer-events: none;
      }
    }
  }
  
  .header-center {
    flex: 1;
    display: flex;
    justify-content: center;
    align-items: center;
    
    @include respond-to(sm) {
      display: none;
    }
    
    .nav-menu.desktop {
      display: flex;
      gap: 8px;
      align-items: center;
      
      .nav-link {
        color: var(--text-color);
        text-decoration: none;
        font-weight: 500;
        padding: 8px 16px;
        border-radius: 6px;
        transition: all 0.3s ease;
        position: relative;
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 14px;
        white-space: nowrap;
        
        :is(i) {
          font-size: 16px;
        }
        
        &:hover,
        &.active {
          background: var(--primary-color);
          color: #fff;
        }
        
        &.admin-back {
          background: #f0f9ff;
          border-left: 3px solid #3b82f6;
          
          &:hover {
            background: #e0f2fe;
            color: #1d4ed8;
          }
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
      display: none;
    }
    
    .user-dropdown {
      .user-info {
        display: flex;
        align-items: center;
        gap: 8px;
        cursor: pointer;
        padding: 6px 16px;
        border-radius: 25px;
        background: rgba(255, 255, 255, 0.2);
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1), 0 2px 4px rgba(0, 0, 0, 0.06);
        backdrop-filter: blur(10px);
        -webkit-backdrop-filter: blur(10px);
        
        &:hover {
          background: rgba(255, 255, 255, 0.3);
          box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15), 0 4px 8px rgba(0, 0, 0, 0.1);
          transform: translateY(-2px) scale(1.02);
        }
        
        .user-name {
          font-weight: 600;
          color: #ffffff;
        }
        
        :is(i) {
          font-size: 12px;
          color: #ffffff;
        }
      }
    }
  }
  
  /* 移动端导航菜单 - 与管理员后端一致 */
  .nav-menu.mobile {
    @include respond-to(sm) {
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
      
      &.active {
        transform: translateX(0);
      }
      
      /* 菜单标题区域 */
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
      
      /* 菜单内容区域 */
      .mobile-menu-content {
        flex: 1;
        overflow-y: auto;
        -webkit-overflow-scrolling: touch;
        padding: 12px 0;
      }
    }
    
    .nav-link {
      @include respond-to(sm) {
        position: relative;
        z-index: 1004 !important;
        pointer-events: auto !important;
        cursor: pointer !important;
        -webkit-tap-highlight-color: transparent;
        display: flex !important;
        align-items: center !important;
        gap: 14px !important;
        padding: 14px 20px !important;
        border-radius: 0 !important;
        margin: 0 !important;
        background: transparent !important;
        border: none !important;
        transition: all 0.3s ease !important;
        font-size: 15px !important;
        font-weight: 500 !important;
        color: #303133 !important;
        width: 100% !important;
        text-align: left !important;
        justify-content: flex-start !important;
        min-height: 48px !important;
        
        :is(i) {
          font-size: 20px !important;
          color: inherit !important;
          width: 24px !important;
          text-align: center !important;
          flex-shrink: 0 !important;
        }
        
        &:hover,
        &:active {
          background: #f5f7fa !important;
          color: var(--theme-primary) !important;
        }
        
        &.active {
          background: #ecf5ff !important;
          color: var(--theme-primary) !important;
          font-weight: 600 !important;
          
          &::before {
            content: '';
            position: absolute;
            left: 0;
            top: 0;
            bottom: 0;
            width: 4px;
            background-color: var(--theme-primary);
          }
        }
        
        &.admin-back {
          background: #fff7e6 !important;
          border-left: 3px solid #ffc107 !important;
          
          &:hover {
            background: #fffbe6 !important;
          }
        }
      }
    }
  }
}

.nav-link {
  color: var(--text-color);
  text-decoration: none;
  font-weight: 500;
  padding: 0.75rem 1.5rem;
  border-radius: 8px;
  transition: all 0.3s ease;
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
  z-index: 1003;
  pointer-events: auto;
  cursor: pointer;
  -webkit-tap-highlight-color: rgba(0, 0, 0, 0.1);
  
  :is(i) {
    font-size: 1.1rem;
    pointer-events: none;
  }
  
  span {
    pointer-events: none;
  }
  
  &:hover,
  &.active {
    background: var(--primary-color);
    color: #fff;
    transform: translateY(-2px);
  }
  
  &:active {
    opacity: 0.8;
    transform: translateY(0);
  }
  
  &.admin-back {
    background: #f0f9ff;
    border-left: 3px solid #3b82f6;
    margin-top: 10px;
    
    &:hover {
      background: #e0f2fe;
      color: #1d4ed8;
    }
  }
  
  @include respond-to(sm) {
    width: 100%;
    text-align: left;
    padding: 15px;
    margin: 5px 0;
    justify-content: flex-start;
    min-height: 44px;
    touch-action: manipulation;
  }
}

/* 左侧导航栏样式（仅桌面端） */
.sidebar {
  position: fixed;
  top: var(--header-height);
  left: 0;
  width: var(--sidebar-width);
  height: calc(100vh - var(--header-height));
  background: white;
  border-right: 1px solid #e4e7ed;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  z-index: 999;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
  pointer-events: auto;
  box-sizing: border-box;
  
  &.collapsed {
    width: 0 !important;
    min-width: 0 !important;
    max-width: 0 !important;
    overflow: hidden !important;
    border-right: none !important;
    display: none !important;
  }
  
  @include respond-to(sm) {
    display: none; // 移动端不显示侧边栏
  }
  
  .sidebar-nav {
    padding: 20px 0;
    transition: padding 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    
    .nav-section {
      margin-bottom: 24px;
      transition: margin-bottom 0.3s cubic-bezier(0.4, 0, 0.2, 1);
      
      .nav-section-title {
        padding: 0 20px 8px;
        font-size: 12px;
        font-weight: 600;
        color: #666;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        transition: opacity 0.3s ease;
      }
      
      .nav-item {
        display: flex;
        align-items: center;
        padding: 12px 20px;
        color: var(--text-color);
        text-decoration: none;
        transition: all 0.3s ease;
        position: relative;
        cursor: pointer;
        user-select: none;
        -webkit-tap-highlight-color: transparent;
        pointer-events: auto !important;
        z-index: 10;
        
        // 确保 router-link 可以正常点击
        :deep(a) {
          display: flex;
          align-items: center;
          width: 100%;
          pointer-events: auto !important;
          z-index: 11;
        }
        
        &:hover {
          background-color: #f5f7fa;
          color: var(--primary-color);
        }
        
        &.active {
          background-color: var(--primary-color);
          color: white;
          font-weight: 600;
          
          &::before {
            content: '';
            position: absolute;
            left: 0;
            top: 0;
            bottom: 0;
            width: 3px;
            background-color: white;
          }
        }
        
        :is(i) {
          font-size: 18px;
          margin-right: 12px;
          width: 20px;
          text-align: center;
          flex-shrink: 0;
          pointer-events: none;
          transition: margin-right 0.3s ease, font-size 0.3s ease;
        }
        
        .nav-text {
          font-weight: 500;
          flex: 1;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
          pointer-events: none;
          transition: opacity 0.3s ease;
        }
        
        &.admin-back {
          background: #f0f9ff;
          border-left: 3px solid #3b82f6;
          
          &:hover {
            background: #e0f2fe;
            color: #1d4ed8;
          }
          
          &.active {
            background-color: var(--primary-color);
            color: white;
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
    margin-left: 0 !important;
    width: 100% !important;
    padding-left: 0 !important;
    
    .content-wrapper {
      padding-left: var(--content-padding) !important;
      margin-left: 0 !important;
    }
    
    .breadcrumb {
      margin-left: 0 !important;
      padding-left: var(--content-padding) !important;
    }
  }
  
  @include respond-to(sm) {
    margin-left: 0 !important;
    margin-top: calc(64px + env(safe-area-inset-top, 0));
    width: 100% !important;
  }
  
  .content-wrapper {
    padding: var(--content-padding);
    width: 100%;
    max-width: none;
    box-sizing: border-box;
    margin-left: 0;
    margin-right: 0;
    background: transparent;
    
    @include respond-to(sm) {
      /* 完全移除顶部padding和margin，让内容紧贴导航栏 */
      padding: 0 12px 20px 12px;
      margin-top: 0 !important;
      padding-top: 0 !important;
    }
    
    /* 全局覆盖：确保所有页面内容在移动端都没有顶部空隙 */
    @include respond-to(sm) {
      :deep(.list-container),
      :deep(.page-header),
      :deep(.page-container) {
        margin-top: 0 !important;
        padding-top: 0 !important;
      }
    }
    
    /* 移动端导航栏 */
    .mobile-nav-bar {
      margin-top: 0 !important;
      margin-bottom: 8px;
      /* 增强可见性：使用浅蓝色背景 */
      background: linear-gradient(180deg, #e3f2fd 0%, #f5f8ff 100%);
      border-radius: 8px;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
      border: 1px solid rgba(66, 165, 245, 0.3);
      overflow: hidden;
      
      .mobile-nav-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 14px 16px;
        cursor: pointer;
        user-select: none;
        -webkit-tap-highlight-color: transparent;
        transition: background-color 0.2s;
        
        &:active {
          background-color: #f5f7fa;
        }
        
        .nav-current-path {
          display: flex;
          align-items: center;
          gap: 8px;
          flex: 1;
          
          :is(i) {
            font-size: 18px;
            color: var(--primary-color);
          }
          
          .current-title {
            font-size: 15px;
            font-weight: 600;
            color: #303133;
          }
        }
        
        .nav-expand-icon {
          font-size: 16px;
          color: #909399;
          transition: transform 0.3s ease;
          
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
        
        .nav-menu-item {
          display: flex;
          align-items: center;
          padding: 12px 16px;
          cursor: pointer;
          user-select: none;
          -webkit-tap-highlight-color: transparent;
          transition: background-color 0.2s;
          position: relative;
          
          &:active {
            background-color: #f5f7fa;
          }
          
          &:hover {
            background-color: #fafafa;
          }
          
          &.active {
            background-color: var(--primary-color);
            color: white;
            
            i:first-child {
              color: white;
            }
            
            .el-icon-check {
              margin-left: auto;
              color: white;
            }
          }
          
          &.admin-back {
            background: #f0f9ff;
            border-left: 3px solid #3b82f6;
            margin-top: 8px;
            
            &:hover {
              background: #e0f2fe;
            }
            
            &.active {
              background: var(--primary-color);
              color: white;
            }
          }
          
          i:first-child {
            font-size: 18px;
            margin-right: 12px;
            width: 20px;
            text-align: center;
            flex-shrink: 0;
            color: #606266;
          }
          
          span {
            flex: 1;
            font-size: 14px;
            font-weight: 500;
            color: #303133;
          }
          
          .el-icon-check {
            margin-left: auto;
            font-size: 16px;
            color: var(--primary-color);
          }
        }
      }
      
      /* 过渡动画 */
      .slide-down-enter-active,
      .slide-down-leave-active {
        transition: all 0.3s ease;
        overflow: hidden;
      }
      
      .slide-down-enter-from {
        max-height: 0;
        opacity: 0;
      }
      
      .slide-down-enter-to {
        max-height: calc(100vh - 200px);
        opacity: 1;
      }
      
      .slide-down-leave-from {
        max-height: calc(100vh - 200px);
        opacity: 1;
      }
      
      .slide-down-leave-to {
        max-height: 0;
        opacity: 0;
      }
    }
    
    .breadcrumb {
      margin-bottom: 20px;
      padding: 12px 16px;
      background: white;
      border-radius: var(--border-radius);
      box-shadow: var(--box-shadow);
      
      @include respond-to(sm) {
        margin-bottom: 12px;
        padding: 10px 12px;
        border-radius: 6px;
        overflow-x: auto;
        -webkit-overflow-scrolling: touch;
        
        // 隐藏滚动条但保留功能
        &::-webkit-scrollbar {
          display: none;
        }
        scrollbar-width: none;
        
        :deep(.el-breadcrumb) {
          font-size: 13px;
          white-space: nowrap;
          
          .el-breadcrumb__item {
            max-width: 120px;
            overflow: hidden;
            text-overflow: ellipsis;
            
            &:last-child {
              max-width: none;
            }
          }
          
          .el-breadcrumb__inner {
            display: inline-block;
            max-width: 100%;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
        }
      }
      
      // 移动端面包屑简化显示
      @media (max-width: 480px) {
        padding: 8px 10px;
        
        :deep(.el-breadcrumb) {
          font-size: 12px;
          
          .el-breadcrumb__separator {
            margin: 0 4px;
          }
        }
      }
    }
    
    .page-content {
      min-height: calc(100vh - 70px - 80px);
      width: 100%;
      
      @include respond-to(sm) {
        min-height: calc(100vh - 60px - 60px);
      }
    }
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

</style> 