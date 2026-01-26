import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/store/auth'
import { useThemeStore } from '@/store/theme'
import { secureStorage } from '@/utils/secureStorage'

const UserLayout = () => import('@/components/layout/UserLayout.vue')
const AdminLayout = () => import('@/components/layout/AdminLayout.vue')

const routes = [
  { path: '/', redirect: '/dashboard' },
  { path: '/login', name: 'Login', component: () => import('@/views/Login.vue'), meta: { requiresGuest: true } },
  { path: '/register', name: 'Register', component: () => import('@/views/Register.vue'), meta: { requiresGuest: true } },
  { path: '/forgot-password', name: 'ForgotPassword', component: () => import('@/views/ForgotPassword.vue'), meta: { requiresGuest: true } },
  { path: '/admin/login', name: 'AdminLogin', component: () => import('@/views/admin/AdminLogin.vue'), meta: { requiresGuest: true } },
  {
    path: '/',
    component: UserLayout,
    meta: { requiresAuth: true },
    children: [
      { path: 'dashboard', name: 'Dashboard', component: () => import('@/views/Dashboard.vue'), meta: { title: '仪表盘', breadcrumb: [{ title: '首页', path: '/dashboard' }] } },
      { path: 'subscription', name: 'Subscription', component: () => import('@/views/Subscription.vue'), meta: { title: '订阅管理', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '订阅管理', path: '/subscription' }] } },
      { path: 'devices', name: 'Devices', component: () => import('@/views/Devices.vue'), meta: { title: '设备管理', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '设备管理', path: '/devices' }] } },
      { path: 'packages', name: 'Packages', component: () => import('@/views/Packages.vue'), meta: { title: '套餐购买', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '套餐购买', path: '/packages' }] } },
      { path: 'orders', name: 'Orders', component: () => import('@/views/Orders.vue'), meta: { title: '订单记录', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '订单记录', path: '/orders' }] } },
      { path: 'nodes', name: 'Nodes', component: () => import('@/views/Nodes.vue'), meta: { title: '节点列表', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '节点列表', path: '/nodes' }] } },
      { path: 'help', name: 'Help', component: () => import('@/views/Help.vue'), meta: { title: '帮助中心', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '帮助中心', path: '/help' }] } },
      { path: 'profile', name: 'Profile', component: () => import('@/views/Profile.vue'), meta: { title: '个人资料', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '个人资料', path: '/profile' }] } },
      { path: 'login-history', name: 'LoginHistory', component: () => import('@/views/LoginHistory.vue'), meta: { title: '登录历史', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '个人资料', path: '/profile' }, { title: '登录历史', path: '/login-history' }] } },
      { path: 'tickets', name: 'Tickets', component: () => import('@/views/Tickets.vue'), meta: { title: '工单中心', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '工单中心', path: '/tickets' }] } },
      { path: 'settings', name: 'UserSettings', component: () => import('@/views/UserSettings.vue'), meta: { title: '用户设置', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '用户设置', path: '/settings' }] } },
      { path: 'tutorials', name: 'SoftwareTutorials', component: () => import('@/views/SoftwareTutorials.vue'), meta: { title: '软件教程', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '软件教程', path: '/tutorials' }] } },
      { path: 'invites', name: 'Invites', component: () => import('@/views/Invites.vue'), meta: { title: '我的邀请', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '我的邀请', path: '/invites' }] } },
      { path: 'payment/return', name: 'PaymentReturn', component: () => import('@/views/PaymentReturn.vue'), meta: { title: '支付返回', requiresAuth: true } }
    ]
  },
  {
    path: '/admin',
    component: AdminLayout,
    meta: { requiresAuth: true, requiresAdmin: true },
    children: [
      { path: '', redirect: '/admin/dashboard' },
      { path: 'dashboard', name: 'AdminDashboard', component: () => import('@/views/admin/Dashboard.vue'), meta: { title: '管理仪表盘', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }] } },
      { path: 'users', name: 'AdminUsers', component: () => import('@/views/admin/Users.vue'), meta: { title: '用户管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '用户管理', path: '/admin/users' }] } },
      { path: 'abnormal-users', name: 'AdminAbnormalUsers', component: () => import('@/views/admin/AbnormalUsers.vue'), meta: { title: '异常用户', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '异常用户', path: '/admin/abnormal-users' }] } },
      { path: 'config-update', name: 'AdminConfigUpdate', component: () => import('@/views/admin/ConfigUpdate.vue'), meta: { title: '节点更新', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '节点更新', path: '/admin/config-update' }] } },
      { path: 'nodes', name: 'AdminNodes', component: () => import('@/views/admin/Nodes.vue'), meta: { title: '节点管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '节点管理', path: '/admin/nodes' }] } },
      { path: 'custom-nodes', name: 'AdminCustomNodes', component: () => import('@/views/admin/CustomNodes.vue'), meta: { title: '专线节点管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '专线节点管理', path: '/admin/custom-nodes' }] } },
      { path: 'subscriptions', name: 'AdminSubscriptions', component: () => import('@/views/admin/Subscriptions.vue'), meta: { title: '订阅管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '订阅管理', path: '/admin/subscriptions' }] } },
      { path: 'orders', name: 'AdminOrders', component: () => import('@/views/admin/Orders.vue'), meta: { title: '订单管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '订单管理', path: '/admin/orders' }] } },
      { path: 'packages', name: 'AdminPackages', component: () => import('@/views/admin/Packages.vue'), meta: { title: '套餐管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '套餐管理', path: '/admin/packages' }] } },
      { path: 'payment-config', name: 'AdminPaymentConfig', component: () => import('@/views/admin/PaymentConfig.vue'), meta: { title: '支付配置', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '支付配置', path: '/admin/payment-config' }] } },
      { path: 'settings', name: 'AdminSettings', component: () => import('@/views/admin/Settings.vue'), meta: { title: '系统设置', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '系统设置', path: '/admin/settings' }] } },
      { path: 'config', name: 'AdminConfig', component: () => import('@/views/admin/Config.vue'), meta: { title: '配置管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '配置管理', path: '/admin/config' }] } },
      { path: 'statistics', name: 'AdminStatistics', component: () => import('@/views/admin/Statistics.vue'), meta: { title: '数据统计', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '数据统计', path: '/admin/statistics' }] } },
      { path: 'email-queue', name: 'AdminEmailQueue', component: () => import('@/views/admin/EmailQueue.vue'), meta: { title: '邮件队列管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '邮件队列管理', path: '/admin/email-queue' }] } },
      { path: 'email-detail/:id', name: 'AdminEmailDetail', component: () => import('@/views/admin/EmailDetail.vue'), meta: { title: '邮件详情', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '邮件队列管理', path: '/admin/email-queue' }, { title: '邮件详情', path: '/admin/email-detail' }] } },
      { path: 'profile', name: 'AdminProfile', component: () => import('@/views/admin/Profile.vue'), meta: { title: '个人资料', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '个人资料', path: '/admin/profile' }] } },
      { path: 'system-logs', name: 'AdminSystemLogs', component: () => import('@/views/admin/SystemLogs.vue'), meta: { title: '系统日志', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '系统日志', path: '/admin/system-logs' }] } },
      { path: 'coupons', name: 'AdminCoupons', component: () => import('@/views/admin/Coupons.vue'), meta: { title: '优惠券管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '优惠券管理', path: '/admin/coupons' }] } },
      { path: 'tickets', name: 'AdminTickets', component: () => import('@/views/admin/Tickets.vue'), meta: { title: '工单管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '工单管理', path: '/admin/tickets' }] } },
      { path: 'invites', name: 'AdminInvites', component: () => import('@/views/admin/Invites.vue'), meta: { title: '邀请管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '邀请管理', path: '/admin/invites' }] } },
      { path: 'user-levels', name: 'AdminUserLevels', component: () => import('@/views/admin/UserLevels.vue'), meta: { title: '用户等级管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '用户等级管理', path: '/admin/user-levels' }] } }
    ]
  },
  { path: '/:pathMatch(.*)*', name: 'NotFound', component: () => import('@/views/NotFound.vue') }
]

const router = createRouter({ history: createWebHistory(), routes })

// 辅助函数：统一处理管理员信息保存
const saveAdminAuth = (adminToken, adminUser) => {
  try {
    const adminData = typeof adminUser === 'string' ? JSON.parse(adminUser) : adminUser
    if (adminData?.is_admin) {
      secureStorage.set('admin_token', adminToken, false, 86400000)
      secureStorage.set('admin_user', adminData, false, 86400000)
    }
  } catch (e) {
    if (process.env.NODE_ENV === 'development') {
      console.warn('解析管理员信息失败:', e)
    }
  }
}

router.beforeEach(async (to, from, next) => {
  if (to.meta.title) document.title = `${to.meta.title} - CBoard`
  try {
    const authStore = useAuthStore()
    const { sessionKey, token, user } = to.query
    // 1. 处理 SessionKey 登录逻辑
    if (sessionKey) {
      const loginData = JSON.parse(sessionStorage.getItem(sessionKey) || 'null')
      if (loginData) {
        if (Date.now() - loginData.timestamp > 300000) {
          sessionStorage.removeItem(sessionKey)
          const { ElMessage } = await import('element-plus')
          ElMessage.error('登录信息已过期')
          return next('/login')
        }
        sessionStorage.removeItem(sessionKey)
        if (loginData.adminToken) saveAdminAuth(loginData.adminToken, loginData.adminUser)
        const userData = { ...loginData.user, is_admin: false }
        secureStorage.set('user_token', loginData.token, true, 86400000)
        secureStorage.set('user_data', userData, true, 86400000)
        authStore.setAuth(loginData.token, userData, true)
        useThemeStore().loadUserTheme().catch(() => {})
        return next({ path: to.path.startsWith('/admin') ? '/dashboard' : to.path, query: { ...to.query, sessionKey: undefined }, replace: true })
      }
    }
    if (token && user) {
      const userData = JSON.parse(decodeURIComponent(user))
      if (userData._adminToken) saveAdminAuth(userData._adminToken, userData._adminUser)
      const finalUser = { ...userData, is_admin: false }
      delete finalUser._adminToken; delete finalUser._adminUser
      secureStorage.set('user_token', token, true, 86400000)
      secureStorage.set('user_data', finalUser, true, 86400000)
      authStore.setAuth(token, finalUser, true)
      useThemeStore().loadUserTheme().catch(() => {})
      return next({ path: to.path, query: { ...to.query, token: undefined, user: undefined }, replace: true })
    }
    // 3. 恢复身份状态
    const isAdminPath = to.path.startsWith('/admin')
    const storedToken = secureStorage.get(isAdminPath ? 'admin_token' : 'user_token')
    const storedUser = secureStorage.get(isAdminPath ? 'admin_user' : 'user_data')
    if (storedToken && storedUser) {
      const userData = typeof storedUser === 'string' ? JSON.parse(storedUser) : storedUser
      if ((isAdminPath && userData.is_admin) || (!isAdminPath && !userData.is_admin)) {
        if (!authStore.isAuthenticated || authStore.token !== storedToken) {
          authStore.setAuth(storedToken, userData, !isAdminPath)
          useThemeStore().loadUserTheme().catch(() => {})
        }
      }
    }
    // 4. 路由权限守卫
    if (to.meta.requiresAuth && !authStore.isAuthenticated) return next(isAdminPath ? '/admin/login' : '/login')
    if (to.meta.requiresAdmin && !authStore.isAdmin) return next(authStore.isAuthenticated ? '/dashboard' : '/admin/login')
    if (to.meta.requiresGuest && authStore.isAuthenticated) {
      if (to.path === '/admin/login') return next(authStore.isAdmin ? '/admin/dashboard' : '/login')
      if (to.path === '/login') return next(authStore.isAdmin ? '/admin/login' : '/dashboard')
      return next(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
    }
    if (to.path === '/') return next(authStore.isAuthenticated ? (authStore.isAdmin ? '/admin/dashboard' : '/dashboard') : '/login')
    next()
  } catch (error) {
    if (process.env.NODE_ENV === 'development') {
      console.error('Router Guard Error:', error)
    }
    next()
  }
})

export default router