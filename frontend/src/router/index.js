import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/store/auth'
import { useThemeStore } from '@/store/theme'
import { secureStorage } from '@/utils/api'
import { useApi } from '@/utils/api'
const SECURE_STORAGE_KEY = 'cboard_secure_'
const ACCESS_TOKEN_TTL = 60 * 60 * 1000
const UserLayout = () => import('@/components/layout/UserLayout.vue')
const AdminLayout = () => import('@/components/layout/AdminLayout.vue')
let _unifiedAuthPromise = null
const getUnifiedAuthEnabled = () => {
  if (!_unifiedAuthPromise) {
    _unifiedAuthPromise = useApi().get('/settings/public-settings')
      .then(response => {
        const settings = response.data?.data || response.data || {}
        return settings.unified_auth_enabled === true || settings.unified_auth_enabled === 'true'
      })
      .catch(() => false)
  }
  return _unifiedAuthPromise
}
const routes = [
  { path: '/', redirect: '/dashboard' },
  {
    path: '/login',
    name: 'Login',
    component: async () => {
      const unifiedAuthEnabled = await getUnifiedAuthEnabled()
      return unifiedAuthEnabled
        ? (await import('@/views/UnifiedAuth.vue')).default
        : (await import('@/views/Login.vue')).default
    },
    meta: { requiresGuest: true }
  },
  {
    path: '/register',
    name: 'Register',
    component: async () => {
      const unifiedAuthEnabled = await getUnifiedAuthEnabled()
      return unifiedAuthEnabled
        ? (await import('@/views/UnifiedAuth.vue')).default
        : (await import('@/views/Register.vue')).default
    },
    meta: { requiresGuest: true }
  },
  {
    path: '/forgot-password',
    name: 'ForgotPassword',
    component: async () => {
      const unifiedAuthEnabled = await getUnifiedAuthEnabled()
      return unifiedAuthEnabled
        ? (await import('@/views/UnifiedAuth.vue')).default
        : (await import('@/views/ForgotPassword.vue')).default
    },
    meta: { requiresGuest: true }
  },
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
      { path: 'tutorials', name: 'SoftwareTutorials', component: () => import('@/components/tutorials/SoftwareTutorials.vue'), meta: { title: '软件教程', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '软件教程', path: '/tutorials' }] } },
      { path: 'invites', name: 'Invites', component: () => import('@/views/Invites.vue'), meta: { title: '我的邀请', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '我的邀请', path: '/invites' }] } },
      { path: 'knowledge', name: 'Knowledge', component: () => import('@/views/Knowledge.vue'), meta: { title: '知识库', breadcrumb: [{ title: '首页', path: '/dashboard' }, { title: '知识库', path: '/knowledge' }] } },
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
      { path: 'logs', name: 'AdminLogs', component: () => import('@/views/admin/Logs.vue'), meta: { title: '日志管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '日志管理', path: '/admin/logs' }] } },
      { path: 'system-logs', name: 'AdminSystemLogs', component: () => import('@/views/admin/SystemLogs.vue'), meta: { title: '系统日志', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '系统日志', path: '/admin/system-logs' }] } },
      { path: 'coupons', name: 'AdminCoupons', component: () => import('@/views/admin/Coupons.vue'), meta: { title: '优惠券管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '优惠券管理', path: '/admin/coupons' }] } },
      { path: 'tickets', name: 'AdminTickets', component: () => import('@/views/admin/Tickets.vue'), meta: { title: '工单管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '工单管理', path: '/admin/tickets' }] } },
      { path: 'invites', name: 'AdminInvites', component: () => import('@/views/admin/Invites.vue'), meta: { title: '邀请管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '邀请管理', path: '/admin/invites' }] } },
      { path: 'user-levels', name: 'AdminUserLevels', component: () => import('@/views/admin/UserLevels.vue'), meta: { title: '用户等级管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '用户等级管理', path: '/admin/user-levels' }] } },
      { path: 'knowledge', name: 'AdminKnowledge', component: () => import('@/views/admin/Knowledge.vue'), meta: { title: '知识库管理', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '知识库管理', path: '/admin/knowledge' }] } },
      { path: 'analytics', name: 'AdminAnalytics', component: () => import('@/views/admin/Analytics.vue'), meta: { title: '用户分析', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '用户分析', path: '/admin/analytics' }] } },
      { path: 'promotions', name: 'AdminPromotions', component: () => import('@/views/admin/Promotions.vue'), meta: { title: '营销活动', breadcrumb: [{ title: '管理后台', path: '/admin/dashboard' }, { title: '营销活动', path: '/admin/promotions' }] } }
    ]
  },
  { path: '/:pathMatch(.*)*', name: 'NotFound', component: () => import('@/views/NotFound.vue') }
]
const router = createRouter({ history: createWebHistory(), routes })
const hasValidLocalSecureValue = (key) => {
  try {
    const item = localStorage.getItem(`${SECURE_STORAGE_KEY}${key}`)
    if (!item) return false
    const data = JSON.parse(item)
    if (data.expiry && Date.now() > data.expiry) {
      localStorage.removeItem(`${SECURE_STORAGE_KEY}${key}`)
      return false
    }
    return true
  } catch (e) {
    return false
  }
}
const saveAdminAuth = (adminToken, adminUser) => {
  try {
    const adminData = typeof adminUser === 'string' ? JSON.parse(adminUser) : adminUser
    if (adminData?.is_admin) {
      secureStorage.set('admin_token', adminToken, false, ACCESS_TOKEN_TTL)
      secureStorage.set('admin_user', adminData, false, ACCESS_TOKEN_TTL)
    }
  } catch (e) {
    if (process.env.NODE_ENV === 'development') console.debug('saveAdminAuth parse failed', e)
  }
}
router.beforeEach(async (to, from, next) => {
  if (to.meta.title) document.title = `${to.meta.title} - CBoard`
  try {
    const authStore = useAuthStore()
    const { sessionKey } = to.query
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
        secureStorage.set('user_token', loginData.token, true, ACCESS_TOKEN_TTL)
        secureStorage.set('user_data', userData, true, ACCESS_TOKEN_TTL)
        authStore.setAuth(loginData.token, userData, true)
        useThemeStore().loadUserTheme().catch(() => {})
        return next({ path: to.path.startsWith('/admin') ? '/dashboard' : to.path, query: { ...to.query, sessionKey: undefined }, replace: true })
      }
    }
    const isAdminPath = to.path.startsWith('/admin')
    const roleTokenKey = isAdminPath ? 'admin_token' : 'user_token'
    const roleUserKey = isAdminPath ? 'admin_user' : 'user_data'
    const storedToken = secureStorage.get(roleTokenKey)
    const storedUser = secureStorage.get(roleUserKey)
    if (storedToken && storedUser) {
      const userData = typeof storedUser === 'string' ? JSON.parse(storedUser) : storedUser
      if ((isAdminPath && userData.is_admin) || (!isAdminPath && !userData.is_admin)) {
        if (!authStore.isAuthenticated || authStore.token !== storedToken) {
          const useSessionStorage = !hasValidLocalSecureValue(roleTokenKey)
          authStore.setAuth(storedToken, userData, useSessionStorage)
          useThemeStore().loadUserTheme().catch(() => {})
        }
      }
    }
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
