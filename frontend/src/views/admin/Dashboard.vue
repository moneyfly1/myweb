<template>
  <div class="list-container admin-dashboard">
    <el-row :gutter="20" class="stat-row">
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-number">{{ stats.totalUsers }}</div>
            <div class="stat-label">总用户数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-number">{{ stats.activeSubscriptions }}</div>
            <div class="stat-label">活跃订阅</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-number">{{ stats.totalOrders }}</div>
            <div class="stat-label">总订单数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-number">¥{{ formatMoney(stats.totalRevenue) }}</div>
            <div class="stat-label">总收入</div>
          </div>
        </el-card>
      </el-col>
    </el-row>
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="8">
        <el-card class="dashboard-card">
          <template #header>
            <div class="card-header">
              <span class="card-title">最近注册用户</span>
              <el-badge :value="recentUsers.length" class="item-count" />
            </div>
          </template>
          <div class="table-container">
            <el-table
              :data="recentUsers.slice(0, 10)"
              style="width: 100%"
              :show-header="false"
              size="small"
              @row-click="goToUserSubscription"
            >
              <el-table-column width="40">
                <template #default="scope">
                  <el-avatar :size="24" class="user-avatar">
                    {{ scope.row.username?.charAt(0)?.toUpperCase() || 'U' }}
                  </el-avatar>
                </template>
              </el-table-column>
              <el-table-column>
                <template #default="scope">
                  <div class="user-info clickable-row">
                    <div class="user-name">{{ scope.row.username }}</div>
                    <div class="user-email">{{ scope.row.email }}</div>
                    <div class="user-time">{{ formatTimeAgo(scope.row.created_at) }}</div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column width="80" align="right">
                <template #default="scope">
                  <el-tag
                    :type="scope.row.status === 'active' ? 'success' : 'warning'"
                    size="small"
                    effect="plain"
                  >
                    {{ scope.row.status === 'active' ? '活跃' : '待激活' }}
                  </el-tag>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card class="dashboard-card">
          <template #header>
            <div class="card-header">
              <span class="card-title">最近订单</span>
              <el-badge :value="recentOrders.length" class="item-count" />
            </div>
          </template>
          <div class="table-container">
            <el-table
              :data="recentOrders.slice(0, 10)"
              style="width: 100%"
              :show-header="false"
              size="small"
              @row-click="goToOrderUserSubscription"
            >
              <el-table-column width="40">
                <template #default="scope">
                  <div class="order-icon">
                    <el-icon><ShoppingCart /></el-icon>
                  </div>
                </template>
              </el-table-column>
              <el-table-column>
                <template #default="scope">
                  <div class="order-info clickable-row">
                    <div class="order-no">{{ scope.row.order_no }}</div>
                    <div class="order-amount">¥{{ formatMoney(scope.row.amount) }}</div>
                    <div class="order-time">{{ formatTimeAgo(scope.row.created_at) }}</div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column width="80" align="right">
                <template #default="scope">
                  <el-tag
                    :type="getOrderStatusType(scope.row.status)"
                    size="small"
                    effect="plain"
                  >
                    {{ getOrderStatusText(scope.row.status) }}
                  </el-tag>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card class="dashboard-card">
          <template #header>
            <div class="card-header">
              <span class="card-title">异常客户</span>
              <div class="header-actions">
                <el-badge :value="abnormalUsers.length" class="item-count" />
                <el-button type="text" @click="goToAbnormalUsers" class="view-all-btn">
                  查看全部
                  <el-icon><ArrowRight /></el-icon>
                </el-button>
              </div>
            </div>
          </template>
          <div class="table-container">
            <el-table 
              :data="abnormalUsers.slice(0, 10)" 
              style="width: 100%"
              :show-header="false"
              size="small"
              @row-click="handleAbnormalUserClick"
            >
              <el-table-column width="40">
                <template #default="scope">
                  <div class="abnormal-icon">
                    <el-icon><Warning /></el-icon>
                  </div>
                </template>
              </el-table-column>
              <el-table-column>
                <template #default="scope">
                  <div class="abnormal-info" style="cursor: pointer;">
                    <div class="abnormal-user">{{ scope.row.username }}</div>
                    <div class="abnormal-email">{{ scope.row.email }}</div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column width="80" align="right">
                <template #default="scope">
                  <el-tag 
                    :type="getAbnormalTypeTag(scope.row.abnormal_type)" 
                    size="small"
                    effect="plain"
                  >
                    {{ scope.row.abnormal_count }}次
                  </el-tag>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-card>
      </el-col>
    </el-row>
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="24">
        <el-card class="dashboard-card expiring-card">
          <template #header>
            <div class="card-header expiring-header">
              <span class="card-title">七天内即将到期客户</span>
              <div class="header-actions expiring-actions">
                <el-select 
                  v-model="expiringFilter" 
                  placeholder="筛选" 
                  class="expiring-filter-select"
                  @change="loadExpiringSubscriptions"
                >
                  <el-option label="全部" value="all" />
                  <el-option label="今天到期" value="today" />
                  <el-option label="1-3天" value="1-3" />
                  <el-option label="4-7天" value="4-7" />
                </el-select>
                <el-button 
                  type="primary" 
                  class="batch-send-btn"
                  :disabled="!selectedExpiring || selectedExpiring.length === 0 || sendingExpireReminder"
                  @click="batchSendExpireReminder"
                >
                  <span class="batch-send-text">
                    {{ sendingExpireReminder ? '发送中...' : `批量发送 (${selectedExpiring ? selectedExpiring.length : 0})` }}
                  </span>
                </el-button>
              </div>
            </div>
          </template>
          <div class="table-container expiring-table-container desktop-only">
            <el-table 
              :data="expiringSubscriptions" 
              style="width: 100%"
              @selection-change="handleExpiringSelectionChange"
            >
              <el-table-column type="selection" width="55" />
              <el-table-column prop="username" label="用户名" width="120" />
              <el-table-column prop="email" label="邮箱" width="200" />
              <el-table-column prop="last_login" label="最后登录" width="160">
                <template #default="scope">
                  <span v-if="scope.row.last_login">{{ scope.row.last_login }}</span>
                  <span v-else style="color: #999;">未登录</span>
                </template>
              </el-table-column>
              <el-table-column prop="expire_time" label="到期时间" width="180">
                <template #default="scope">
                  <el-tag :type="getExpireTagType(scope.row.days_until_expire)" size="small">
                    {{ scope.row.expire_time }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="days_until_expire" label="剩余天数" width="100" align="center">
                <template #default="scope">
                  <span :style="{ color: getExpireColor(scope.row.days_until_expire) }">
                    {{ scope.row.days_until_expire }} 天
                  </span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="120" fixed="right">
                <template #default="scope">
                  <el-button
                    type="success"
                    size="small"
                    @click="sendExpireReminder([scope.row.user_id || scope.row.id])"
                  >
                    发送提醒
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
            <div v-if="!expiringSubscriptions || expiringSubscriptions.length === 0" class="empty-state">
              暂无即将到期的客户
            </div>
          </div>
          <div class="mobile-expiring-list mobile-only">
            <div v-if="!expiringSubscriptions || expiringSubscriptions.length === 0" class="empty-state-mobile">
              暂无即将到期的客户
            </div>
            <div 
              v-for="item in expiringSubscriptions" 
              :key="item.user_id || item.id"
              class="expiring-item-card"
            >
              <div class="expiring-item-header">
                <el-checkbox 
                  :model-value="selectedExpiring.includes(item.user_id || item.id)"
                  @change="toggleExpiringSelection(item)"
                  class="expiring-checkbox"
                />
                <div class="expiring-user-info">
                  <div class="expiring-username">{{ item.username }}</div>
                  <div class="expiring-email">{{ item.email }}</div>
                </div>
                <el-tag 
                  :type="getExpireTagType(item.days_until_expire)" 
                  size="small"
                  class="expiring-days-tag"
                >
                  {{ item.days_until_expire }} 天
                </el-tag>
              </div>
              <div class="expiring-item-body">
                <div class="expiring-item-row">
                  <span class="expiring-label">最后登录:</span>
                  <span class="expiring-value">{{ item.last_login || '未登录' }}</span>
                </div>
                <div class="expiring-item-row">
                  <span class="expiring-label">到期时间:</span>
                  <span class="expiring-value" :style="{ color: getExpireColor(item.days_until_expire) }">
                    {{ item.expire_time }}
                  </span>
                </div>
              </div>
              <div class="expiring-item-actions">
                <el-button
                  type="success"
                  size="small"
                  class="action-btn"
                  @click="sendExpireReminder([item.user_id || item.id])"
                >
                  发送提醒
                </el-button>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>
<script>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from '@/utils/elementPlusServices'
import { useRouter } from 'vue-router'
import { useApi } from '@/utils/api'
import { adminAPI } from '@/utils/api'
import { ArrowRight, ShoppingCart, Warning } from '@element-plus/icons-vue'
export default {
  name: 'AdminDashboard',
  components: {
    ArrowRight,
    ShoppingCart,
    Warning
  },
  setup() {
    const api = useApi()
    const router = useRouter()
    const stats = ref({
      totalUsers: 0,
      activeSubscriptions: 0,
      totalOrders: 0,
      totalRevenue: 0
    })
    const recentUsers = ref([])
    const recentOrders = ref([])
    const abnormalUsers = ref([])
    const expiringSubscriptions = ref([])
    const selectedExpiring = ref([])
    const expiringFilter = ref('all')
    const sendingExpireReminder = ref(false)
    const loadStats = async () => {
      try {
        const response = await adminAPI.getDashboard()
        if (response && response.data) {
          let data = null
          if (response.data.success === true && response.data.data) {
            data = response.data.data
          } else if (response.data.totalUsers !== undefined) {
            data = response.data
          } else if (response.data.data && typeof response.data.data === 'object') {
            data = response.data.data
          }
          if (data) {
            stats.value = {
              totalUsers: Number(data.totalUsers) || 0,
              activeSubscriptions: Number(data.activeSubscriptions) || 0,
              totalOrders: Number(data.totalOrders) || 0,
              totalRevenue: Number(data.totalRevenue) || 0
            }
          } else {
            console.error('无法解析数据，响应格式:', response.data)
            ElMessage.warning('数据格式异常，请查看控制台')
            stats.value = {
              totalUsers: 0,
              activeSubscriptions: 0,
              totalOrders: 0,
              totalRevenue: 0
            }
          }
        } else {
          console.error('获取统计数据失败 - 无响应数据:', response)
          ElMessage.error('获取统计数据失败: 无响应数据')
          stats.value = {
            totalUsers: 0,
            activeSubscriptions: 0,
            totalOrders: 0,
            totalRevenue: 0
          }
        }
      } catch (error) {
        console.error('获取统计数据异常:', error)
        console.error('错误详情:', {
          message: error.message,
          response: error.response?.data,
          status: error.response?.status,
          statusText: error.response?.statusText
        })
        ElMessage.error('获取统计数据失败: ' + (error.response?.data?.message || error.message || '未知错误'))
        stats.value = {
          totalUsers: 0,
          activeSubscriptions: 0,
          totalOrders: 0,
          totalRevenue: 0
        }
      }
    }
    const loadRecentUsers = async () => {
      try {
        const response = await api.get('/admin/users/recent')
        if (response && response.data) {
          if (response.data.success !== false) {
            recentUsers.value = response.data.data || []
          } else {
            recentUsers.value = []
          }
        } else {
          recentUsers.value = []
        }
      } catch (error) {
        console.error('加载最近用户失败:', error)
        recentUsers.value = []
      }
    }
    const loadRecentOrders = async () => {
      try {
        const response = await api.get('/admin/orders/recent')
        if (response && response.data) {
          if (response.data.success !== false) {
            recentOrders.value = response.data.data || []
          } else {
            recentOrders.value = []
          }
        } else {
          recentOrders.value = []
        }
      } catch (error) {
        console.error('加载最近订单失败:', error)
        recentOrders.value = []
      }
    }
    const loadAbnormalUsers = async () => {
      try {
        const response = await api.get('/admin/users/abnormal')
        if (response && response.data) {
          if (response.data.success !== false) {
            const data = response.data.data || []
            abnormalUsers.value = Array.isArray(data) ? data.slice(0, 5) : []
          } else {
            abnormalUsers.value = []
          }
        } else {
          abnormalUsers.value = []
        }
      } catch (error) {
        console.error('加载异常用户失败:', error)
        abnormalUsers.value = []
      }
    }
    const getOrderStatusType = (status) => {
      const statusMap = {
        'pending': 'warning',
        'paid': 'success',
        'cancelled': 'danger',
        'refunded': 'info'
      }
      return statusMap[status] || 'info'
    }
    const getOrderStatusText = (status) => {
      const statusMap = {
        'pending': '待支付',
        'paid': '已支付',
        'cancelled': '已取消',
        'refunded': '已退款'
      }
      return statusMap[status] || status
    }
    const getAbnormalTypeTag = (type) => {
      const typeMap = {
        'frequent_reset': 'warning',
        'frequent_subscription': 'danger',
        'multiple_abnormal': 'error'
      }
      return typeMap[type] || 'info'
    }
    const getAbnormalTypeText = (type) => {
      const typeMap = {
        'frequent_reset': '频繁重置',
        'frequent_subscription': '频繁订阅',
        'multiple_abnormal': '多重异常'
      }
      return typeMap[type] || type
    }
    const goToAbnormalUsers = () => {
      router.push('/admin/abnormal-users')
    }
    const handleAbnormalUserClick = (row) => {
      ElMessageBox.confirm(
        `选择操作：`,
        `用户 ${row.username || row.email}`,
        {
          distinguishCancelAndClose: true,
          confirmButtonText: '查看订阅列表',
          cancelButtonText: '查看异常详情',
          type: 'info',
          showClose: false
        }
      ).then(() => {
        goToUserSubscription(row)
      }).catch((action) => {
        if (action === 'cancel') {
          viewAbnormalUserDetails(row)
        }
      })
    }
    const viewAbnormalUserDetails = (row) => {
      const userId = row.user_id || row.id
      if (userId) {
        router.push({
          path: '/admin/abnormal-users',
          query: { user_id: userId }
        })
      } else {
        ElMessage.warning('无法获取用户ID')
      }
    }
    const goToUserSubscription = (row) => {
      const searchParam = row.email || row.username || row.id || row.user_id
      if (searchParam) {
        router.push({
          path: '/admin/subscriptions',
          query: { search: searchParam }
        })
      } else {
        ElMessage.warning('该用户信息不完整')
      }
    }
    const goToOrderUserSubscription = (row) => {
      const orderNo = row.order_no || row.orderNo
      if (orderNo) {
        router.push({
          path: '/admin/orders',
          query: { search: orderNo }
        })
      } else {
        ElMessage.warning('该订单号不存在')
      }
    }
    const formatMoney = (value) => {
      if (value === null || value === undefined || value === '') return '0.00'
      const num = typeof value === 'string' ? parseFloat(value) : value
      if (isNaN(num)) return '0.00'
      return num.toFixed(2)
    }
    const formatTimeAgo = (dateString) => {
      if (!dateString) return '未知'
      const date = new Date(dateString)
      const now = new Date()
      const diffMs = now - date
      const diffSec = Math.floor(diffMs / 1000)
      const diffMin = Math.floor(diffSec / 60)
      const diffHour = Math.floor(diffMin / 60)
      const diffDay = Math.floor(diffHour / 24)

      if (diffSec < 60) return '刚刚'
      if (diffMin < 60) return `${diffMin}分钟前`
      if (diffHour < 24) return `${diffHour}小时前`
      if (diffDay < 7) return `${diffDay}天前`
      if (diffDay < 30) return `${Math.floor(diffDay / 7)}周前`
      if (diffDay < 365) return `${Math.floor(diffDay / 30)}个月前`
      return `${Math.floor(diffDay / 365)}年前`
    }
    const loadExpiringSubscriptions = async () => {
      try {
        const params = { days: 7 }
        if (expiringFilter.value !== 'all') {
          params.filter = expiringFilter.value
        }
        const response = await adminAPI.getExpiringSubscriptions(params)
        if (response && response.data) {
          if (response.data.success !== false) {
            expiringSubscriptions.value = response.data.data || []
          } else {
            expiringSubscriptions.value = []
          }
        } else {
          expiringSubscriptions.value = []
        }
      } catch (error) {
        console.error('加载即将到期订阅失败:', error)
        expiringSubscriptions.value = []
      }
    }
    const handleExpiringSelectionChange = (selection) => {
      selectedExpiring.value = selection.map(item => item.user_id || item.id)
    }
    const toggleExpiringSelection = (item) => {
      const id = item.user_id || item.id
      const index = selectedExpiring.value.indexOf(id)
      if (index > -1) {
        selectedExpiring.value.splice(index, 1)
      } else {
        selectedExpiring.value.push(id)
      }
    }
    const getExpireTagType = (days) => {
      if (days <= 0) return 'danger'
      if (days <= 1) return 'warning'
      if (days <= 3) return 'warning'
      return 'info'
    }
    const getExpireColor = (days) => {
      if (days <= 0) return '#f56c6c'
      if (days <= 1) return '#e6a23c'
      if (days <= 3) return '#e6a23c'
      return '#409eff'
    }
    const sendExpireReminder = async (ids) => {
      try {
        sendingExpireReminder.value = true
        const response = await adminAPI.batchSendExpireReminder(ids)
        if (response.data && response.data.success) {
          ElMessage.success(`成功发送 ${ids.length} 条到期提醒`)
          selectedExpiring.value = []
        } else {
          ElMessage.error(response.data?.message || '发送失败')
        }
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '发送失败')
      } finally {
        sendingExpireReminder.value = false
      }
    }
    const batchSendExpireReminder = () => {
      if (selectedExpiring.value.length === 0) {
        ElMessage.warning('请先选择要发送提醒的客户')
        return
      }
      sendExpireReminder(selectedExpiring.value)
    }
    onMounted(async () => {
      try {
        await Promise.allSettled([
          loadStats(),
          loadRecentUsers(),
          loadRecentOrders(),
          loadAbnormalUsers(),
          loadExpiringSubscriptions()
        ])
      } catch (error) {
        console.error('加载仪表盘数据时发生错误:', error)
      }
    })
    return {
      stats,
      recentUsers,
      recentOrders,
      abnormalUsers,
      getOrderStatusType,
      getOrderStatusText,
      getAbnormalTypeTag,
      getAbnormalTypeText,
      goToAbnormalUsers,
      handleAbnormalUserClick,
      viewAbnormalUserDetails,
      goToUserSubscription,
      goToOrderUserSubscription,
      formatMoney,
      formatTimeAgo,
      expiringSubscriptions,
      selectedExpiring,
      expiringFilter,
      sendingExpireReminder,
      loadExpiringSubscriptions,
      handleExpiringSelectionChange,
      toggleExpiringSelection,
      getExpireTagType,
      getExpireColor,
      sendExpireReminder,
      batchSendExpireReminder
    }
  }
}
</script>
<style scoped>
.admin-dashboard {
  padding: 20px;
}
.stat-card {
  text-align: center;
}
.stat-content {
  padding: 20px;
}
.stat-number {
  font-size: 2em;
  font-weight: bold;
  color: #409EFF;
}
.stat-label {
  margin-top: 10px;
  color: #666;
}
@media (max-width: 768px) {
  .admin-dashboard {
    padding: 0;
  }
  .admin-dashboard :deep(.el-row:not(.stat-row)) {
    margin-left: 0 !important;
    margin-right: 0 !important;
    margin-top: 0 !important;
    .el-col {
      width: 100% !important;
      max-width: 100% !important;
      flex: 0 0 100% !important;
      padding-left: 0 !important;
      padding-right: 0 !important;
      margin-bottom: 12px !important;
    }
    &[style*="margin-top"] {
      margin-top: 0 !important;
    }
  }
  .admin-dashboard :deep(.stat-row) {
    margin-left: -4px !important;
    margin-right: -4px !important;
    .el-col {
      padding-left: 4px !important;
      padding-right: 4px !important;
      margin-bottom: 8px !important;
    }
  }
  .stat-card {
    margin-bottom: 0;
    :deep(.el-card__body) {
      padding: 10px 8px !important;
    }
    .stat-content {
      padding: 6px;
    }
    .stat-number {
      font-size: 1.4em;
    }
    .stat-label {
      font-size: 0.75rem;
      margin-top: 4px;
    }
  }
  .dashboard-card {
    height: auto !important;
    min-height: 300px;
    margin-bottom: 16px;
    :deep(.el-card__header) {
      padding: 12px 16px;
      .card-header {
        padding: 0;
        .card-title {
          font-size: 0.9375rem;
        }
        .item-count {
          :deep(.el-badge__content) {
            font-size: 11px;
            min-width: 18px;
            height: 18px;
            line-height: 18px;
            padding: 0 5px;
          }
        }
        .header-actions {
          .view-all-btn {
            font-size: 0.75rem;
            padding: 2px 6px;
          }
        }
      }
    }
    .table-container {
      padding: 12px;
      max-height: 250px;
      :deep(.el-table) {
        font-size: 0.8125rem;
        .el-table__body-wrapper {
          max-height: 200px;
        }
        .el-table__cell {
          padding: 8px 4px;
        }
      }
    }
  }
  .user-info,
  .order-info,
  .abnormal-info {
    padding-left: 6px;
    .user-name,
    .order-no,
    .abnormal-user {
      font-size: 0.8125rem;
    }
    .user-email,
    .order-amount,
    .abnormal-email {
      font-size: 0.75rem;
    }
    .user-time,
    .order-time {
      font-size: 0.6875rem;
      margin-top: 3px;
    }
  }
  :deep(.el-col) {
    flex: 0 0 100% !important;
    max-width: 100% !important;
    width: 100% !important;
  }
}
@media (max-width: 480px) {
  .admin-dashboard {
    padding: 8px;
  }
  .stat-card {
    .stat-content {
      padding: 10px;
    }
    .stat-number {
      font-size: 1.5em;
    }
    .stat-label {
      font-size: 0.8125rem;
    }
  }
  .dashboard-card {
    :deep(.el-card__header) {
      padding: 10px 12px;
    }
    .table-container {
      padding: 10px;
      :deep(.el-table) {
        font-size: 0.75rem;
        .el-table__cell {
          padding: 6px 2px;
        }
      }
    }
  }
}
.dashboard-card {
  height: 400px;
  display: flex;
  flex-direction: column;
}
.dashboard-card .el-card__body {
  flex: 1;
  padding: 0;
  display: flex;
  flex-direction: column;
}
.card-header {
  padding: 0 4px;
}
.card-title {
  font-weight: 600;
  color: #303133;
  font-size: 16px;
}
.item-count {
  margin-right: 8px;
}
.view-all-btn {
  padding: 4px 8px;
  font-size: 12px;
  color: #409eff;
}
.view-all-btn:hover {
  color: #66b1ff;
}
.table-container {
  flex: 1;
  overflow: clip;
  padding: 16px;
}
.table-container .el-table {
  height: 100%;
}
.table-container .el-table__body-wrapper {
  max-height: 300px;
  overflow-y: auto;
}
.user-avatar {
  background-color: #409eff;
  color: white;
  font-size: 12px;
  font-weight: 600;
}
.user-info {
  padding-left: 8px;
}
.user-name {
  font-weight: 500;
  color: #303133;
  font-size: 14px;
  line-height: 1.2;
}
.user-email {
  color: #909399;
  font-size: 12px;
  line-height: 1.2;
  margin-top: 2px;
}
.user-time {
  color: #67c23a;
  font-size: 11px;
  line-height: 1.2;
  margin-top: 2px;
  font-weight: 500;
}
.order-icon {
  width: 24px;
  height: 24px;
  background-color: #67c23a;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 12px;
}
.order-info {
  padding-left: 8px;
}
.order-no {
  font-weight: 500;
  color: #303133;
  font-size: 14px;
  line-height: 1.2;
}
.order-amount {
  color: #e6a23c;
  font-size: 12px;
  line-height: 1.2;
  margin-top: 2px;
  font-weight: 600;
}
.order-time {
  color: #67c23a;
  font-size: 11px;
  line-height: 1.2;
  margin-top: 2px;
  font-weight: 500;
}
.abnormal-icon {
  width: 24px;
  height: 24px;
  background-color: #f56c6c;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 12px;
}
.abnormal-info {
  padding-left: 8px;
}
.abnormal-user {
  font-weight: 500;
  color: #303133;
  font-size: 14px;
  line-height: 1.2;
}
.abnormal-email {
  color: #909399;
  font-size: 12px;
  line-height: 1.2;
  margin-top: 2px;
}
.table-container .el-table__row {
  height: 48px;
  cursor: pointer;
}
.table-container .el-table__row:hover {
  background-color: #f0f9ff;
}
.clickable-row {
  cursor: pointer;
}
.table-container .el-tag {
  border-radius: 12px;
  font-size: 11px;
  padding: 2px 8px;
  height: 20px;
  line-height: 16px;
}
.table-container .el-table__body-wrapper::-webkit-scrollbar {
  width: 4px;
}
.table-container .el-table__body-wrapper::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 2px;
}
.table-container .el-table__body-wrapper::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 2px;
}
.table-container .el-table__body-wrapper::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}
:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
}
:deep(.el-select .el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
}
:deep(.el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background-color: transparent !important;
}
:deep(.el-input__wrapper:hover) {
  border-color: #c0c4cc !important;
  box-shadow: none !important;
}
:deep(.el-input__wrapper.is-focus) {
  border-color: #1677ff !important;
  box-shadow: none !important;
}
.expiring-card {
  min-height: auto !important;
}
.expiring-header {
  flex-wrap: wrap;
  gap: 12px;
}
.expiring-actions {
  flex-wrap: wrap;
  gap: 8px;
}
.expiring-filter-select {
  min-width: 100px;
}
.batch-send-btn {
  white-space: nowrap;
}
.batch-send-text {
  font-size: 13px;
}
.empty-state {
  text-align: center;
  padding: 40px;
  color: #999;
}
.mobile-expiring-list {
  padding: 12px;
}
.empty-state-mobile {
  text-align: center;
  padding: 60px 20px;
  color: #999;
  font-size: 14px;
}
.expiring-item-card {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 12px;
  border: 1px solid #e9ecef;
}
.expiring-item-header {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 12px;
}
.expiring-checkbox {
  margin-top: 4px;
}
.expiring-user-info {
  flex: 1;
  min-width: 0;
}
.expiring-username {
  font-weight: 600;
  color: #303133;
  font-size: 15px;
  line-height: 1.4;
  margin-bottom: 4px;
  word-break: break-all;
}
.expiring-email {
  color: #909399;
  font-size: 13px;
  line-height: 1.4;
  word-break: break-all;
}
.expiring-days-tag {
  flex-shrink: 0;
  margin-top: 2px;
}
.expiring-item-body {
  padding: 10px 0;
  border-top: 1px solid #e9ecef;
  border-bottom: 1px solid #e9ecef;
  margin-bottom: 10px;
}
.expiring-item-row {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
  font-size: 13px;
}
.expiring-item-row:last-child {
  margin-bottom: 0;
}
.expiring-label {
  color: #909399;
  min-width: 70px;
  flex-shrink: 0;
}
.expiring-value {
  color: #303133;
  flex: 1;
  word-break: break-all;
}
.expiring-item-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.expiring-item-actions .action-btn {
  flex: 1;
  min-width: 100px;
}
@media (max-width: 768px) {
  .expiring-card {
    margin-bottom: 16px;
  }
  .expiring-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  .expiring-actions {
    width: 100%;
    flex-direction: column;
  }
  .expiring-filter-select {
    width: 100%;
    margin-bottom: 0;
  }
  .batch-send-btn {
    width: 100%;
  }
  .batch-send-text {
    font-size: 14px;
  }
  .expiring-table-container {
    display: none !important;
  }
  .mobile-expiring-list {
    display: block !important;
  }
  .expiring-item-card {
    padding: 14px;
  }
  .expiring-username {
    font-size: 16px;
  }
  .expiring-email {
    font-size: 14px;
  }
  .expiring-item-row {
    font-size: 14px;
  }
  .expiring-label {
    min-width: 80px;
  }
}
@media (min-width: 769px) {
  .mobile-expiring-list {
    display: none !important;
  }
  .expiring-table-container {
    display: block !important;
  }
}
@media (max-width: 480px) {
  .expiring-item-card {
    padding: 12px;
  }
  .expiring-username {
    font-size: 15px;
  }
  .expiring-email {
    font-size: 13px;
  }
  .expiring-item-actions {
    flex-direction: column;
  }
  .expiring-item-actions .action-btn {
    width: 100%;
    min-width: auto;
  }
}
.desktop-only {
  display: block;
}
.mobile-only {
  display: none;
}
@media (max-width: 768px) {
  .desktop-only {
    display: none !important;
  }
  .mobile-only {
    display: block !important;
  }
}
</style> 
