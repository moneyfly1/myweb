<template>
  <div class="abnormal-users">
    <el-card>
      <template #header>
        <div class="header-content">
          <span>异常用户</span>
          <div class="header-actions desktop-only">
            <el-button type="primary" @click="applyFilters">
              <el-icon><Search /></el-icon>
              查询
            </el-button>
            <el-button @click="resetFilters">
              <el-icon><Refresh /></el-icon>
              重置
            </el-button>
          </div>
        </div>
      </template>
      <div class="filter-section">
        <el-card shadow="never" class="filter-card">
          <div class="filter-content">
            <div class="filter-item">
              <label class="filter-label">时间段：</label>
              <el-date-picker
                v-model="filters.dateRange"
                type="daterange"
                range-separator="至"
                start-placeholder="开始日期"
                end-placeholder="结束日期"
                format="YYYY-MM-DD"
                value-format="YYYY-MM-DD"
                class="filter-date-picker"
                clearable
              />
            </div>
            <div class="filter-item">
              <label class="filter-label">订阅次数：</label>
              <el-input-number
                v-model="filters.subscriptionCount"
                :min="1"
                :max="100"
                placeholder="大于等于"
                class="filter-input-number"
              />
              <span class="filter-unit">次</span>
            </div>
            <div class="filter-item">
              <label class="filter-label">重置次数：</label>
              <el-input-number
                v-model="filters.resetCount"
                :min="1"
                :max="100"
                placeholder="大于等于"
                class="filter-input-number"
              />
              <span class="filter-unit">次</span>
            </div>
            <div class="mobile-filter-actions">
              <el-button type="primary" @click="applyFilters" class="mobile-action-btn">
                <el-icon><Search /></el-icon>
                查询
              </el-button>
              <el-button @click="resetFilters" class="mobile-action-btn">
                <el-icon><Refresh /></el-icon>
                重置
              </el-button>
            </div>
          </div>
        </el-card>
      </div>
      <div class="table-wrapper desktop-only">
        <el-table
          :data="abnormalUsers"
          style="width: 100%"
          v-loading="loading"
          :empty-text="abnormalUsers.length === 0 && !loading ? '暂无数据，请设置筛选条件后点击查询按钮' : ''"
        >
          <el-table-column prop="username" label="用户名" width="120">
            <template #default="scope">
              <el-button type="text" @click="viewUserDetails(scope.row.user_id)">
                {{ scope.row.username }}
              </el-button>
            </template>
          </el-table-column>
          <el-table-column prop="email" label="邮箱" width="200">
            <template #default="scope">
              <el-button type="text" @click="viewUserDetails(scope.row.user_id)">
                {{ scope.row.email }}
              </el-button>
            </template>
          </el-table-column>
          <el-table-column prop="abnormal_type" label="异常类型" width="150">
            <template #default="scope">
              <el-tag :type="abnormalTypeMap[scope.row.abnormal_type]?.tag || 'info'">
                {{ abnormalTypeMap[scope.row.abnormal_type]?.text || scope.row.abnormal_type }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="abnormal_count" label="异常次数" width="120" />
          <el-table-column prop="subscription_count" label="订阅次数" width="120" />
          <el-table-column prop="reset_count" label="重置次数" width="120" />
          <el-table-column prop="description" label="异常描述" />
          <el-table-column prop="last_activity" label="最后活动时间" width="180" />
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="scope">
              <el-button size="small" @click="viewUserDetails(scope.row.user_id)">
                <el-icon><View /></el-icon>
                查看详情
              </el-button>
              <el-button size="small" type="warning" @click="markAsNormal(scope.row)">
                <el-icon><Check /></el-icon>
                标记正常
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <div class="mobile-card-list" v-if="isMobile">
        <div v-if="abnormalUsers.length === 0 && !loading" class="empty-state">
          <el-empty description="暂无数据，请设置筛选条件后点击查询按钮" />
        </div>
        <div v-for="user in abnormalUsers" :key="user.user_id" class="mobile-card">
          <div class="card-header">
            <div class="user-info">
              <el-avatar :size="40" :src="user.avatar">
                {{ user.username?.charAt(0)?.toUpperCase() }}
              </el-avatar>
              <div class="user-text">
                <div class="username">{{ user.username }}</div>
                <div class="email">{{ user.email }}</div>
              </div>
            </div>
            <el-tag :type="abnormalTypeMap[user.abnormal_type]?.tag || 'info'" size="small">
              {{ abnormalTypeMap[user.abnormal_type]?.text || user.abnormal_type }}
            </el-tag>
          </div>
          <div class="card-body">
            <div class="card-row">
              <span class="label">异常次数</span>
              <span class="value highlight">{{ user.abnormal_count }}</span>
            </div>
            <div class="card-row" v-if="user.subscription_count !== undefined && user.subscription_count !== null">
              <span class="label">订阅次数</span>
              <span class="value highlight">{{ user.subscription_count }}</span>
            </div>
            <div class="card-row" v-if="user.reset_count !== undefined && user.reset_count !== null">
              <span class="label">重置次数</span>
              <span class="value highlight">{{ user.reset_count }}</span>
            </div>
            <div class="card-row" v-if="user.description">
              <span class="label">异常描述</span>
              <span class="value">{{ user.description }}</span>
            </div>
            <div class="card-row">
              <span class="label">最后活动</span>
              <span class="value">{{ formatDate(user.last_activity) }}</span>
            </div>
          </div>
          <div class="card-actions">
            <el-button type="primary" @click="viewUserDetails(user.user_id)" class="mobile-action-btn">
              <el-icon><View /></el-icon>
              查看详情
            </el-button>
            <el-button type="warning" @click="markAsNormal(user)" class="mobile-action-btn">
              <el-icon><Check /></el-icon>
              标记正常
            </el-button>
          </div>
        </div>
      </div>
    </el-card>
    <el-dialog
      v-model="showUserDetailsDialog"
      title="用户详细信息"
      :width="isMobile ? '95%' : '80%'"
      class="user-details-dialog"
    >
      <div v-if="userDetails" class="user-details">
        <el-card style="margin-bottom: 20px;" shadow="never">
          <template #header>
            <span>基本信息</span>
          </template>
          <el-descriptions :column="isMobile ? 1 : 3" border>
            <el-descriptions-item label="用户ID">{{ userDetails.user_info.id }}</el-descriptions-item>
            <el-descriptions-item label="用户名">{{ userDetails.user_info.username }}</el-descriptions-item>
            <el-descriptions-item label="邮箱">{{ userDetails.user_info.email }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="userDetails.user_info.is_active ? 'success' : 'danger'">
                {{ userDetails.user_info.is_active ? '活跃' : '禁用' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="管理员">
              <el-tag :type="userDetails.user_info.is_admin ? 'danger' : 'info'">
                {{ userDetails.user_info.is_admin ? '是' : '否' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="注册时间">{{ formatDate(userDetails.user_info.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="最后登录">{{ formatDate(userDetails.user_info.last_login) || '从未登录' }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
        <el-card style="margin-bottom: 20px;" shadow="never">
          <template #header>
            <span>统计信息</span>
          </template>
          <el-row :gutter="20">
            <el-col :xs="12" :sm="12" :md="6" :lg="6">
              <div class="stat-item">
                <div class="stat-number">{{ userDetails.statistics.total_subscriptions }}</div>
                <div class="stat-label">总订阅数</div>
              </div>
            </el-col>
            <el-col :xs="12" :sm="12" :md="6" :lg="6">
              <div class="stat-item">
                <div class="stat-number">{{ userDetails.statistics.total_orders }}</div>
                <div class="stat-label">总订单数</div>
              </div>
            </el-col>
            <el-col :xs="12" :sm="12" :md="6" :lg="6">
              <div class="stat-item">
                <div class="stat-number">{{ userDetails.statistics.total_resets }}</div>
                <div class="stat-label">总重置次数</div>
              </div>
            </el-col>
            <el-col :xs="12" :sm="12" :md="6" :lg="6">
              <div class="stat-item">
                <div class="stat-number">¥{{ userDetails.statistics.total_spent }}</div>
                <div class="stat-label">总消费</div>
              </div>
            </el-col>
          </el-row>
        </el-card>
        <el-card style="margin-bottom: 20px;" shadow="never" v-if="userDetails.subscription_resets && userDetails.subscription_resets.length > 0">
          <template #header>
            <span>订阅重置记录</span>
          </template>
          <el-table :data="userDetails.subscription_resets" style="width: 100%" :max-height="300">
            <el-table-column prop="reset_type" label="重置类型" width="120" />
            <el-table-column prop="reason" label="重置原因" />
            <el-table-column prop="device_count_before" label="重置前设备数" width="120" />
            <el-table-column prop="device_count_after" label="重置后设备数" width="120" />
            <el-table-column prop="reset_by" label="操作者" width="100" />
            <el-table-column prop="created_at" label="重置时间" width="180">
              <template #default="scope">
                {{ formatDate(scope.row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
        <el-card shadow="never" v-if="userDetails.recent_activities && userDetails.recent_activities.length > 0">
          <template #header>
            <span>最近活动</span>
          </template>
          <el-table :data="userDetails.recent_activities" style="width: 100%" :max-height="300">
            <el-table-column prop="activity_type" label="活动类型" width="120" />
            <el-table-column prop="description" label="描述" />
            <el-table-column prop="ip_address" label="IP地址" width="150" />
            <el-table-column prop="created_at" label="时间" width="180">
              <template #default="scope">
                {{ formatDate(scope.row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, View, Check, Search } from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'

const abnormalTypeMap = {
  disabled: { tag: 'danger', text: '账户禁用' },
  frequent_reset: { tag: 'warning', text: '频繁重置' },
  frequent_subscription: { tag: 'danger', text: '频繁订阅' },
  inactive: { tag: 'info', text: '长期未登录' },
  multiple_abnormal: { tag: 'error', text: '多重异常' },
  unverified: { tag: 'warning', text: '未验证邮箱' },
  unknown: { tag: 'info', text: '未知异常' }
}

export default {
  name: 'AbnormalUsers',
  components: {
    Refresh, View, Check, Search
  },
  setup() {
    const route = useRoute()
    const loading = ref(false)
    const abnormalUsers = ref([])
    const showUserDetailsDialog = ref(false)
    const userDetails = ref(null)
    const isMobile = ref(window.innerWidth <= 768)
    
    // 设置默认日期范围：本月1号到今天
    const getDefaultDateRange = () => {
      const now = new Date()
      const firstDay = new Date(now.getFullYear(), now.getMonth(), 1)
      const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
      return [
        firstDay.toISOString().split('T')[0],
        today.toISOString().split('T')[0]
      ]
    }
    
    const filters = reactive({
      dateRange: getDefaultDateRange(),  // 默认：本月1号到今天
      subscriptionCount: 10,  // 默认：订阅次数>=10次
      resetCount: 3  // 默认：重置次数>=3次
    })
    const handleResize = () => {
      isMobile.value = window.innerWidth <= 768
    }
    const loadAbnormalUsers = async () => {
      loading.value = true
      try {
        const params = {}
        // 日期范围：如果有值就传递，后端会使用默认值（本月1号到今天）
        if (filters.dateRange && filters.dateRange.length === 2) {
          params['date_range[]'] = filters.dateRange
        }
        // 订阅次数：如果有值就传递，后端默认值是10
        if (filters.subscriptionCount !== null && filters.subscriptionCount !== undefined && filters.subscriptionCount > 0) {
          params.subscription_count = filters.subscriptionCount.toString()
        }
        // 重置次数：如果有值就传递，后端默认值是3
        if (filters.resetCount !== null && filters.resetCount !== undefined && filters.resetCount > 0) {
          params.reset_count = filters.resetCount.toString()
        }
        const response = await adminAPI.getAbnormalUsers(params)
        if (response.data && response.data.success) {
          const data = response.data.data
          if (Array.isArray(data)) {
            abnormalUsers.value = data
          } else if (data && Array.isArray(data.users)) {
            abnormalUsers.value = data.users
          } else if (data && Array.isArray(data.list)) {
            abnormalUsers.value = data.list
          } else {
            abnormalUsers.value = []
          }
        } else {
          abnormalUsers.value = []
        }
      } catch (error) {
        ElMessage.error('加载异常用户失败: ' + (error.response?.data?.message || error.message))
        abnormalUsers.value = []
      } finally {
        loading.value = false
      }
    }
    const applyFilters = () => {
      loadAbnormalUsers()
    }
    const resetFilters = () => {
      filters.dateRange = getDefaultDateRange()  // 重置为默认日期范围
      filters.subscriptionCount = 10  // 重置为默认值
      filters.resetCount = 3  // 重置为默认值
      // 重置后自动查询
      loadAbnormalUsers()
    }
    const viewUserDetails = async (userId) => {
      try {
        const response = await adminAPI.getUserDetails(userId)
        if (response && response.data && response.data.success) {
          userDetails.value = response.data.data
          showUserDetailsDialog.value = true
        } else if (response && response.success) {
          userDetails.value = response.data
          showUserDetailsDialog.value = true
        } else {
          ElMessage.error('获取用户详情失败: ' + (response?.data?.message || response?.message || '未知错误'))
        }
      } catch (error) {
        ElMessage.error('获取用户详情失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const markAsNormal = async (user) => {
      try {
        await ElMessageBox.confirm(
          `确定要将用户 ${user.username} 标记为正常吗？这将从异常列表中移除该用户。`,
          '确认操作',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        ElMessage.success('用户已标记为正常')
        loadAbnormalUsers()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('操作失败')
        }
      }
    }
    const formatDate = (dateStr) => {
      if (!dateStr) return '-'
      return new Date(dateStr).toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      })
    }
    onMounted(() => {
      window.addEventListener('resize', handleResize)
      // 页面加载时自动查询（使用默认筛选条件：本月1号到今天，订阅次数>=10，重置次数>=3）
      loadAbnormalUsers()
      
      // 检查URL参数中是否有user_id，如果有则自动查看该用户详情
      const route = useRoute()
      const userId = route.query.user_id
      if (userId) {
        // 等待数据加载完成后再查看详情
        setTimeout(() => {
          viewUserDetails(Number(userId))
        }, 500)
      }
    })
    onUnmounted(() => {
      window.removeEventListener('resize', handleResize)
    })
    return {
      loading,
      abnormalUsers,
      filters,
      showUserDetailsDialog,
      userDetails,
      isMobile,
      abnormalTypeMap,
      loadAbnormalUsers,
      applyFilters,
      resetFilters,
      viewUserDetails,
      markAsNormal,
      formatDate
    }
  }
}
</script>

<style scoped lang="scss">
.abnormal-users {
  padding: 20px;
  @media (max-width: 768px) {
    padding: 12px;
  }
}
.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  @media (max-width: 768px) {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
}
.header-actions {
  display: flex;
  gap: 10px;
}
.filter-section {
  margin-bottom: 20px;
  @media (max-width: 768px) {
    margin-bottom: 16px;
  }
  .filter-card {
    background: #f8f9fa;
    border: 1px solid #e9ecef;
    .filter-content {
      display: flex;
      flex-wrap: wrap;
      gap: 20px;
      align-items: flex-end;
      @media (max-width: 768px) {
        flex-direction: column;
        gap: 16px;
        align-items: stretch;
      }
      .filter-item {
        display: flex;
        align-items: center;
        gap: 8px;
        flex: 1;
        min-width: 200px;
        @media (max-width: 768px) {
          flex-direction: column;
          align-items: stretch;
          min-width: auto;
          gap: 8px;
        }
        .filter-label {
          font-size: 14px;
          color: #606266;
          font-weight: 500;
          white-space: nowrap;
          min-width: 80px;
          @media (max-width: 768px) {
            min-width: auto;
            margin-bottom: 0;
            font-size: 13px;
          }
        }
        .filter-date-picker {
          flex: 1;
          width: 100%;
          @media (max-width: 768px) {
            width: 100%;
          }
        }
        .filter-input-number {
          flex: 1;
          width: 100%;
          @media (max-width: 768px) {
            width: 100%;
          }
        }
        .filter-unit {
          color: #909399;
          font-size: 14px;
          white-space: nowrap;
          @media (max-width: 768px) {
            font-size: 13px;
          }
        }
      }
      .mobile-filter-actions {
        display: none;
        @media (max-width: 768px) {
          display: flex;
          gap: 12px;
          width: 100%;
          margin-top: 8px;
          .mobile-action-btn {
            flex: 1;
            height: 44px;
            font-size: 16px;
          }
        }
      }
    }
  }
}
.table-wrapper {
  margin-top: 20px;
  @media (max-width: 768px) {
    display: none !important;
  }
}
.mobile-card-list {
  display: none;
  @media (max-width: 768px) {
    display: block;
    margin-top: 16px;
    .empty-state {
      padding: 60px 20px;
      text-align: center;
      background: #fff;
      border-radius: 8px;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    }
  }
  .mobile-card {
    background: #fff;
    border-radius: 8px;
    padding: 16px;
    margin-bottom: 12px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;
      padding-bottom: 12px;
      border-bottom: 1px solid #f0f0f0;
      .user-info {
        display: flex;
        align-items: center;
        gap: 12px;
        flex: 1;
        .user-text {
          flex: 1;
          .username {
            font-size: 16px;
            font-weight: 600;
            color: #333;
            margin-bottom: 4px;
          }
          .email {
            font-size: 14px;
            color: #666;
          }
        }
      }
    }
    .card-body {
      margin-bottom: 12px;
      .card-row {
        display: flex;
        align-items: center;
        margin-bottom: 12px;
        padding-bottom: 12px;
        border-bottom: 1px solid #f0f0f0;
        &:last-of-type {
          border-bottom: none;
          margin-bottom: 0;
          padding-bottom: 0;
        }
        .label {
          flex: 0 0 90px;
          font-size: 14px;
          color: #666;
          font-weight: 500;
        }
        .value {
          flex: 1;
          font-size: 14px;
          color: #333;
          word-break: break-word;
          &.highlight {
            color: #f56c6c;
            font-weight: 600;
            font-size: 16px;
          }
        }
      }
    }
    .card-actions {
      display: flex;
      gap: 8px;
      padding-top: 12px;
      border-top: 1px solid #f0f0f0;
      .mobile-action-btn {
        flex: 1;
        height: 44px;
        font-size: 16px;
        margin: 0;
      }
    }
  }
  .empty-state {
    padding: 40px 20px;
    text-align: center;
  }
}
.user-details-dialog {
  :deep(.el-dialog__body) {
    max-height: calc(100vh - 200px);
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
  }
}
.user-details {
  .stat-item {
    text-align: center;
    padding: 20px;
  }
  .stat-item .stat-number {
    font-size: 1.5rem;
    font-weight: bold;
    color: #409eff;
    margin-bottom: 5px;
  }
  .stat-item .stat-label {
    color: #606266;
    font-size: 12px;
  }
}
.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
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
</style>