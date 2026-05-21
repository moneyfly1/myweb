import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/store/auth'
import { useThemeStore } from '@/store/theme'
import { secureStorage } from '@/utils/api'
import { ElMessage } from '@/utils/elementPlusServices'

const SECURE_STORAGE_KEY = 'cboard_secure_'
const ACCESS_TOKEN_TTL = 60 * 60 * 1000
const LOGIN_HANDOFF_STORAGE_PREFIX = 'cboard_login_handoff_'
const UserLayout = () => import('@/components/layout/UserLayout.vue')
const AdminLayout = () => import('@/components/layout/AdminLayout.vue')

const routes = [
  { path: '/', redirect: '/dashboard' },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/UnifiedAuth.vue'),
    meta: { requiresGuest: true }
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/UnifiedAuth.vue'),
    meta: { requiresGuest: true }
  },
  {
    path: '/forgot-password',
    name: 'ForgotPassword',
    component: () => import('@/views/UnifiedAuth.vue'),
    meta: { requiresGuest: true }
  },
  {
    path: '/admin/login',
    name: 'AdminLogin',
    component: () => import('@/views/UnifiedAuth.vue'),
    meta: { requiresGuest: true, adminLogin: true }
  },
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
const ADMIN_USER_TTL = 30 * 24 * 60 * 60 * 1000 // 30天
const getStorageMode = (key) => {
  const storageKey = `${SECURE_STORAGE_KEY}${key}`
  try {
    if (sessionStorage.getItem(storageKey)) return 'session'
    if (localStorage.getItem(storageKey)) return 'local'
  } catch (e) {
    if (process.env.NODE_ENV === 'development') console.debug('getStorageMode failed', e)
  }
  return null
}
const saveAdminAuth = (adminToken, adminUser) => {
  try {
    const adminData = typeof adminUser === 'string' ? JSON.parse(adminUser) : adminUser
    if (adminData?.is_admin) {
      secureStorage.set('admin_token', adminToken, false, ACCESS_TOKEN_TTL)
      secureStorage.set('admin_user', adminData, false, ADMIN_USER_TTL)
    }
  } catch (e) {
    if (process.env.NODE_ENV === 'development') console.debug('saveAdminAuth parse failed', e)
  }
}
const readLoginHandoff = (sessionKey) => {
  if (!sessionKey) return null
  const localStorageKey = `${LOGIN_HANDOFF_STORAGE_PREFIX}${sessionKey}`
  const raw = sessionStorage.getItem(sessionKey) || localStorage.getItem(localStorageKey)
  sessionStorage.removeItem(sessionKey)
  localStorage.removeItem(localStorageKey)
  if (!raw) return null
  try {
    return JSON.parse(raw)
  } catch (e) {
    if (process.env.NODE_ENV === 'development') console.debug('readLoginHandoff parse failed', e)
    return null
  }
}
router.beforeEach(async (to, from, next) => {
  if (to.meta.title) document.title = `${to.meta.title} - CBoard`
  try {
    const authStore = useAuthStore()

    // 访客页面（登录/注册等）快速通过，跳过所有 token 检查
    if (to.meta.requiresGuest) {
      if (to.meta.adminLogin) {
        if (secureStorage.get('admin_token') && secureStorage.get('admin_user')) {
          return next('/admin/dashboard')
        }
        return next()
      }
      if (secureStorage.get('user_token') && secureStorage.get('user_data')) {
        return next('/dashboard')
      }
      return next()
    }

    const { sessionKey } = to.query
    if (sessionKey) {
      const loginData = readLoginHandoff(sessionKey)
      if (loginData) {
        if (Date.now() - loginData.timestamp > 300000) {
          ElMessage.error('登录信息已过期')
          return next('/login')
        }
        if (loginData.adminToken) saveAdminAuth(loginData.adminToken, loginData.adminUser)
        if (loginData.adminRefreshToken) {
          secureStorage.set('admin_refresh_token', loginData.adminRefreshToken, false, ADMIN_USER_TTL)
        }
        const userData = { ...loginData.user, is_admin: false }
        const useSessionStorage = loginData.storage !== 'local'
        secureStorage.set('user_token', loginData.token, useSessionStorage, ACCESS_TOKEN_TTL)
        secureStorage.set('user_data', userData, useSessionStorage, ADMIN_USER_TTL)
        if (loginData.refreshToken) {
          secureStorage.set('user_refresh_token', loginData.refreshToken, useSessionStorage, ADMIN_USER_TTL)
        }
        authStore.setAuth(loginData.token, userData, useSessionStorage)
        useThemeStore().loadUserTheme().catch(() => {})
        return next({ path: to.path.startsWith('/admin') ? '/dashboard' : to.path, query: { ...to.query, sessionKey: undefined }, replace: true })
      }
    }
    const isAdminPath = to.path.startsWith('/admin')
    const userStorageMode = getStorageMode('user_token') || getStorageMode('user_refresh_token') || getStorageMode('user_data')

    // 优先检查对应角色的token
    const roleTokenKey = isAdminPath ? 'admin_token' : 'user_token'
    const roleUserKey = isAdminPath ? 'admin_user' : 'user_data'
    let storedToken = secureStorage.get(roleTokenKey)
    let storedUser = secureStorage.get(roleUserKey)

    // 如果当前路径需要的角色token不存在，尝试用对应角色的refresh cookie刷新
    if (!storedToken || !storedUser) {
      // 如果访问管理员路径但admin_token不存在，尝试刷新admin token
      if (isAdminPath) {
        try {
          const storedRefresh = secureStorage.get('admin_refresh_token')
          if (!storedRefresh) {
            return next('/admin/login')
          }
          const { default: axios } = await import('axios')
          const refreshResponse = await axios.post(
            '/api/v1/auth/refresh',
            { refresh_token: storedRefresh },
            { timeout: 5000 }
          )
          const { access_token, refresh_token: newRefresh } = refreshResponse.data?.data || refreshResponse.data || {}
          if (access_token) {
            secureStorage.set('admin_token', access_token, false, ACCESS_TOKEN_TTL)
            if (newRefresh) secureStorage.set('admin_refresh_token', newRefresh, false, ADMIN_USER_TTL)
            storedToken = access_token
            storedUser = secureStorage.get('admin_user')
            if (!storedUser) {
              return next('/admin/login')
            }
          } else {
            return next('/admin/login')
          }
        } catch {
          return next('/admin/login')
        }
      } else {
        // 访问用户路径但user_token不存在，尝试用用户 refresh token 刷新
        try {
          const storedRefresh = secureStorage.get('user_refresh_token')
          if (!storedRefresh) {
            // 无 refresh token，让后续的 requiresAuth 检查处理
          } else {
            const { default: axios } = await import('axios')
            const refreshResponse = await axios.post(
              '/api/v1/auth/refresh',
              { refresh_token: storedRefresh },
              { timeout: 5000 }
            )
            const { access_token, refresh_token: newRefresh } = refreshResponse.data?.data || refreshResponse.data || {}
            if (access_token) {
              const useSessionStorage = userStorageMode !== 'local'
              secureStorage.set('user_token', access_token, useSessionStorage, ACCESS_TOKEN_TTL)
              if (newRefresh) secureStorage.set('user_refresh_token', newRefresh, useSessionStorage, ADMIN_USER_TTL)
              storedToken = access_token
              storedUser = secureStorage.get('user_data')
              if (!storedUser) {
                // user_data也过期了，无法恢复身份
                // 不做跳转，让后续的 requiresAuth 检查处理
              }
            }
          }
        } catch {
          // 用户refresh也失败，不做跳转，让后续的 requiresAuth 检查处理
        }
      }
    }

    if (storedToken && storedUser) {
      const userData = typeof storedUser === 'string' ? JSON.parse(storedUser) : storedUser

      // 验证token和路径的角色是否匹配
      const tokenIsAdmin = userData.is_admin === true

      if (isAdminPath && !tokenIsAdmin) {
        // 访问管理员路径但token不是管理员，重定向到用户页面
        return next('/dashboard')
      }

      if (!authStore.isAuthenticated || authStore.token !== storedToken) {
        const useSessionStorage = isAdminPath ? false : userStorageMode !== 'local'
        authStore.setAuth(storedToken, userData, useSessionStorage)
        useThemeStore().loadUserTheme().catch(() => {})
      }
    }
    const hasRoleAuth = !!(storedToken && storedUser)
    if (to.meta.requiresAuth && !hasRoleAuth) return next(isAdminPath ? '/admin/login' : '/login')
    if (to.meta.requiresAdmin && !authStore.isAdmin) return next('/admin/login')
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
