<template>
  <div class="admin-dashboard">
    <el-row :gutter="20">
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-number">{{ stats.totalUsers }}</div>
            <div class="stat-label">总用户数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-number">{{ stats.activeSubscriptions }}</div>
            <div class="stat-label">活跃订阅</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-number">{{ stats.totalOrders }}</div>
            <div class="stat-label">总订单数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
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
                  <div class="user-info">
                    <div class="user-name">{{ scope.row.username }}</div>
                    <div class="user-email">{{ scope.row.email }}</div>
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
                  <div class="order-info">
                    <div class="order-no">{{ scope.row.order_no }}</div>
                    <div class="order-amount">¥{{ formatMoney(scope.row.amount) }}</div>
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
                  <div class="abnormal-info">
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
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useApi } from '@/utils/api'
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

    const loadStats = async () => {
      try {
        const response = await api.get('/admin/stats')
        console.log('Stats API Response:', response)
        if (response && response.data) {
          if (response.data.success && response.data.data) {
            const data = response.data.data
            console.log('Stats Data:', data)
            stats.value = {
              totalUsers: Number(data.totalUsers) || 0,
              activeSubscriptions: Number(data.activeSubscriptions) || 0,
              totalOrders: Number(data.totalOrders) || 0,
              totalRevenue: Number(data.totalRevenue) || 0
            }
            console.log('Updated Stats:', stats.value)
          } else {
            console.error('获取统计数据失败 - 响应格式错误:', response.data)
          }
        } else {
          console.error('获取统计数据失败 - 无响应数据:', response)
        }
      } catch (error) {
        console.error('获取统计数据异常:', error)
        if (error.response) {
          console.error('错误响应:', error.response.data)
        }
      }
    }

    const loadRecentUsers = async () => {
      try {
        const response = await api.get('/admin/users/recent')
        if (response.data && response.data.success && response.data.data) {
          recentUsers.value = response.data.data
        } else {
          recentUsers.value = []
        }
      } catch (error) {
        recentUsers.value = []
      }
    }

    const loadRecentOrders = async () => {
      try {
        const response = await api.get('/admin/orders/recent')
        if (response.data && response.data.success && response.data.data) {
          recentOrders.value = response.data.data
        } else {
          recentOrders.value = []
        }
      } catch (error) {
        recentOrders.value = []
      }
    }

    const loadAbnormalUsers = async () => {
      try {
        const response = await api.get('/admin/users/abnormal')
        if (response.data && response.data.success && response.data.data) {
          // 只显示前5个异常用户
          abnormalUsers.value = response.data.data.slice(0, 5)
        } else {
          abnormalUsers.value = []
        }
      } catch (error) {
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

    const formatMoney = (value) => {
      if (value === null || value === undefined || value === '') return '0.00'
      const num = typeof value === 'string' ? parseFloat(value) : value
      if (isNaN(num)) return '0.00'
      return num.toFixed(2)
    }

    onMounted(() => {
      loadStats()
      loadRecentUsers()
      loadRecentOrders()
      loadAbnormalUsers()
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
      formatMoney
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

/* 移动端优化 */
@media (max-width: 768px) {
  .admin-dashboard {
    padding: 0;
  }
  
  /* 统计卡片垂直排列 */
  .admin-dashboard :deep(.el-row) {
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
  
  .stat-card {
    margin-bottom: 0;
    
    :deep(.el-card__body) {
      padding: 16px 12px !important;
    }
    
    .stat-content {
      padding: 12px;
    }
    
    .stat-number {
      font-size: 1.75em;
    }
    
    .stat-label {
      font-size: 0.875rem;
      margin-top: 8px;
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
  }
  
  /* 确保所有el-col在移动端都是100%宽度 */
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

/* 仪表盘卡片样式 */
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
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 4px;
}

.card-title {
  font-weight: 600;
  color: #303133;
  font-size: 16px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
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

/* 表格容器 */
.table-container {
  flex: 1;
  overflow: hidden;
  padding: 16px;
}

.table-container .el-table {
  height: 100%;
}

.table-container .el-table__body-wrapper {
  max-height: 300px;
  overflow-y: auto;
}

/* 用户信息样式 */
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

/* 订单信息样式 */
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

/* 异常用户信息样式 */
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

/* 表格行样式 */
.table-container .el-table__row {
  height: 48px;
}

.table-container .el-table__row:hover {
  background-color: #f5f7fa;
}

/* 标签样式优化 */
.table-container .el-tag {
  border-radius: 12px;
  font-size: 11px;
  padding: 2px 8px;
  height: 20px;
  line-height: 16px;
}

/* 滚动条样式 */
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

/* 移除所有输入框的圆角和阴影效果，设置为简单长方形 */
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
</style> 