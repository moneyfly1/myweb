<template>
  <div class="user-layout" :class="{ 'sidebar-collapsed': sidebarCollapsed, 'is-mobile': isMobile }">
    <header class="header">
      <div class="header-container">
        <div class="header-left">
          <button v-if="isMobile" class="action-btn menu-toggle" @click="mobileNavExpanded = !mobileNavExpanded">
            <i :class="mobileNavExpanded ? 'el-icon-close' : 'el-icon-menu'"></i>
            <span>{{ mobileNavExpanded ? '收起' : '菜单' }}</span>
          </button>

          <div class="logo" @click="router.push('/dashboard')">
            <i class="el-icon-s-home"></i>
            <span class="logo-text" v-show="!sidebarCollapsed || isMobile">CBoard</span>
          </div>

          <button v-if="!isMobile" class="action-btn desktop-toggle" @click="toggleSidebar">
            <i :class="sidebarCollapsed ? 'el-icon-menu' : 'el-icon-close'"></i>
          </button>
        </div>

        <div class="header-right">
          <el-dropdown @command="handleThemeChange">
            <el-button type="text" class="action-btn theme-btn">
              <i class="el-icon-brush"></i>
              <span class="btn-text">主题</span>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-for="t in themes" :key="t.value" :command="t.value" :class="{ active: currentTheme === t.value }">
                  <i class="el-icon-check" v-if="currentTheme === t.value"></i>{{ t.label }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>

          <el-dropdown @command="handleUserCommand">
            <div class="user-info">
              <el-avatar :size="32" :src="userAvatar">{{ userInitials }}</el-avatar>
              <span class="user-name" v-if="!isMobile && !sidebarCollapsed">{{ user.username }}</span>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-if="hasAdminAccess" command="backToAdmin" divided><i class="el-icon-back"></i>返回管理</el-dropdown-item>
                <el-dropdown-item command="profile"><i class="el-icon-user"></i>个人资料</el-dropdown-item>
                <el-dropdown-item command="settings"><i class="el-icon-setting"></i>设置</el-dropdown-item>
                <el-dropdown-item divided command="logout"><i class="el-icon-switch-button"></i>退出</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>

      <transition name="slide-left">
        <div class="mobile-drawer" v-if="isMobile && mobileNavExpanded">
          <div class="drawer-mask" @click="mobileNavExpanded = false"></div>
          <div class="drawer-content">
            <div class="drawer-header">菜单 <i class="el-icon-close" @click="mobileNavExpanded = false"></i></div>
            <nav class="nav-list">
              <div v-for="item in menuItems" :key="item.path" @click="handleNavMenuClick(item)" 
                   class="nav-item" :class="{ active: $route.path === item.path, 'admin-back': item.isAdminBack }">
                <i :class="item.icon"></i><span>{{ item.title }}</span>
              </div>
            </nav>
          </div>
        </div>
      </transition>
    </header>

    <aside class="sidebar" v-if="!isMobile" :class="{ collapsed: sidebarCollapsed }">
      <nav class="sidebar-nav">
        <div class="nav-section-title" v-show="!sidebarCollapsed">用户中心</div>
        <template v-for="item in menuItems" :key="item.path">
          <div class="nav-item" :class="{ active: $route.path === item.path, 'admin-back': item.isAdminBack }" @click="handleNavMenuClick(item)">
            <i :class="item.icon"></i>
            <span class="nav-text" v-show="!sidebarCollapsed">{{ item.title }}</span>
          </div>
        </template>
      </nav>
    </aside>

    <main class="main-content">
      <div class="content-wrapper">
        <div class="mobile-path-bar" v-if="isMobile" @click="mobileNavExpanded = true">
          <i class="el-icon-location"></i>
          <span>{{ route.meta.title || getCurrentPageTitle() }}</span>
          <i class="el-icon-arrow-right"></i>
        </div>

        <el-breadcrumb v-if="showBreadcrumb && !isMobile" separator="/" class="breadcrumb">
          <el-breadcrumb-item v-for="item in breadcrumbItems" :key="item.path" :to="item.path">{{ item.title }}</el-breadcrumb-item>
        </el-breadcrumb>

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
const themeStore = useThemeStore()

// --- 状态管理 ---
const isMobile = ref(false)
const sidebarCollapsed = ref(localStorage.getItem('userSidebarCollapsed') === 'true')
const mobileNavExpanded = ref(false)

// --- 计算属性 ---
const user = computed(() => authStore.user)
const currentTheme = computed(() => themeStore.currentTheme)
const themes = computed(() => themeStore.availableThemes)
const userAvatar = computed(() => user.value?.avatar || '')
const userInitials = computed(() => user.value?.username?.substring(0, 2).toUpperCase() || '')
const hasAdminAccess = computed(() => !!(secureStorage.get('admin_token') && secureStorage.get('admin_user')))
const showBreadcrumb = computed(() => route.meta.showBreadcrumb !== false)
const breadcrumbItems = computed(() => route.meta.breadcrumb || [])

const menuItems = computed(() => {
  const baseItems = [
    { path: '/dashboard', title: '仪表盘', icon: 'el-icon-s-home' },
    { path: '/subscription', title: '订阅管理', icon: 'el-icon-connection' },
    { path: '/devices', title: '设备管理', icon: 'el-icon-mobile-phone' },
    { path: '/packages', title: '套餐购买', icon: 'el-icon-shopping-cart-2' },
    { path: '/orders', title: '订单记录', icon: 'el-icon-document' },
    { path: '/nodes', title: '节点列表', icon: 'el-icon-location' },
    { path: '/tickets', title: '工单中心', icon: 'el-icon-s-ticket' },
    { path: '/invites', title: '我的邀请', icon: 'el-icon-user' },
    { path: '/help', title: '帮助中心', icon: 'el-icon-question' }
  ]
  if (hasAdminAccess.value) {
    baseItems.push({ path: '#admin', title: '返回管理后台', icon: 'el-icon-back', isAdminBack: true })
  }
  return baseItems
})

// --- 方法 ---
const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
  localStorage.setItem('userSidebarCollapsed', sidebarCollapsed.value)
}

const handleNavMenuClick = (item) => {
  if (item.isAdminBack) returnToAdmin()
  else router.push(item.path)
  mobileNavExpanded.value = false
}

const handleUserCommand = (command) => {
  const actions = {
    backToAdmin: returnToAdmin,
    profile: () => router.push('/profile'),
    settings: () => router.push('/settings'),
    logout: () => { authStore.logout(); router.push('/login') }
  }
  actions[command]?.()
}

const returnToAdmin = () => {
  const token = secureStorage.get('admin_token')
  const userData = secureStorage.get('admin_user')
  try {
    const user = typeof userData === 'string' ? JSON.parse(userData) : userData
    if (!user?.is_admin) throw new Error('Not Admin')
    authStore.setAuth(token, user, false)
    secureStorage.remove('user_token')
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
  isMobile.value = window.innerWidth <= 992
  if (isMobile.value) sidebarCollapsed.value = true
}

const getCurrentPageTitle = () => {
  const map = { '/dashboard': '仪表盘', '/subscription': '订阅管理' /* ... 补全 */ }
  return map[route.path] || '用户中心'
}

// --- 生命周期 ---
watch(() => route.path, () => mobileNavExpanded.value = false)
onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
})
onUnmounted(() => window.removeEventListener('resize', checkMobile))
</script>

<style scoped lang="scss">
// 使用简化的 Flex 布局和变量
.header {
  position: fixed;
  top: 0; width: 100%; height: 64px;
  background: var(--theme-primary, #409EFF);
  z-index: 1000;
  color: white;

  .header-container {
    display: flex; justify-content: space-between; align-items: center;
    height: 100%; padding: 0 20px;
  }
}

.action-btn {
  background: rgba(255,255,255,0.2);
  border: none; border-radius: 20px;
  color: white; padding: 8px 15px;
  cursor: pointer; display: flex; align-items: center; gap: 8px;
  transition: 0.3s;
  &:hover { background: rgba(255,255,255,0.3); }
}

.sidebar {
  position: fixed; top: 64px; left: 0; bottom: 0;
  width: 240px; background: white; border-right: 1px solid #eee;
  transition: 0.3s;
  &.collapsed { width: 0; overflow: hidden; }
}

.nav-item {
  padding: 12px 20px; display: flex; align-items: center; gap: 12px;
  cursor: pointer; transition: 0.2s;
  &:hover { background: #f5f7fa; color: #409EFF; }
  &.active { background: #ecf5ff; color: #409EFF; border-right: 3px solid #409EFF; }
  &.admin-back { background: #fff7e6; color: #faad14; }
}

.main-content {
  margin-top: 64px;
  margin-left: 240px;
  transition: 0.3s;
  .sidebar-collapsed &, .is-mobile & { margin-left: 0; }
  .content-wrapper { padding: 20px; }
}

.mobile-drawer {
  position: fixed; inset: 0; z-index: 2000;
  .drawer-mask { position: absolute; inset: 0; background: rgba(0,0,0,0.5); }
  .drawer-content {
    position: absolute; left: 0; top: 0; bottom: 0; width: 280px;
    background: white; color: #333;
    .drawer-header { padding: 20px; border-bottom: 1px solid #eee; display: flex; justify-content: space-between; }
  }
}

.mobile-path-bar {
  background: white; padding: 12px; border-radius: 8px;
  display: flex; align-items: center; gap: 10px; margin-bottom: 15px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.05);
}
</style>