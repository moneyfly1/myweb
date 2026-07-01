<template>
  <div class="abnormal-users">
    <el-card>
      <template #header>
        <div class="header-content">
          <span>异常用户</span>
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
            <div class="desktop-filter-actions">
              <el-button type="primary" @click="applyFilters">
                <el-icon><Search /></el-icon>
                查询
              </el-button>
              <el-button @click="resetFilters">
                <el-icon><Refresh /></el-icon>
                重置
              </el-button>
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
      <ResponsiveDataView
        class="abnormal-users-data"
        :data="abnormalUsers"
        :fields="mobileAbnormalFields"
        :loading="loading"
        empty-title="暂无异常用户"
        empty-description="请设置筛选条件后点击查询按钮"
      >
        <template #table>
          <div class="table-wrapper">
            <el-table
              :data="abnormalUsers"
              class="data-table"
              v-loading="loading"
              stripe
              border
              empty-text=" "
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
            <EmptyState
              v-if="!loading && abnormalUsers.length === 0"
              class="desktop-empty-state"
              title="暂无异常用户"
              description="请设置筛选条件后点击查询按钮"
              action-text="查询"
              :loading="loading"
              @action="applyFilters"
            />
          </div>
        </template>

        <template #header="{ item }">
          <div class="abnormal-card-header">
            <div class="user-info">
              <el-avatar :size="40" :src="item.avatar">
                {{ item.username?.charAt(0)?.toUpperCase() }}
              </el-avatar>
              <button type="button" class="user-text-button" @click="viewUserDetails(item.user_id)">
                <span class="username">{{ item.username }}</span>
                <span class="email">{{ item.email }}</span>
              </button>
            </div>
            <el-tag :type="abnormalTypeMap[item.abnormal_type]?.tag || 'info'" size="small">
              {{ abnormalTypeMap[item.abnormal_type]?.text || item.abnormal_type }}
            </el-tag>
          </div>
        </template>

        <template #field-abnormal_type="{ item }">
          <el-tag :type="abnormalTypeMap[item.abnormal_type]?.tag || 'info'" size="small">
            {{ abnormalTypeMap[item.abnormal_type]?.text || item.abnormal_type }}
          </el-tag>
        </template>

        <template #field-abnormal_count="{ item }">
          <span class="mobile-highlight-value">{{ item.abnormal_count }}</span>
        </template>

        <template #field-subscription_count="{ item }">
          <span class="mobile-highlight-value">
            {{ item.subscription_count ?? '-' }}
          </span>
        </template>

        <template #field-reset_count="{ item }">
          <span class="mobile-highlight-value">
            {{ item.reset_count ?? '-' }}
          </span>
        </template>

        <template #field-description="{ item }">
          <span class="mobile-description-value">{{ item.description || '-' }}</span>
        </template>

        <template #field-last_activity="{ item }">
          {{ formatDate(item.last_activity) }}
        </template>

        <template #actions="{ item }">
          <div class="mobile-abnormal-actions">
            <el-button type="primary" @click="viewUserDetails(item.user_id)" class="mobile-action-btn">
              <el-icon><View /></el-icon>
              查看详情
            </el-button>
            <el-button type="warning" @click="markAsNormal(item)" class="mobile-action-btn">
              <el-icon><Check /></el-icon>
              标记正常
            </el-button>
          </div>
        </template>

        <template #empty>
          <EmptyState
            title="暂无异常用户"
            description="请设置筛选条件后点击查询按钮"
            action-text="查询"
            :loading="loading"
            @action="applyFilters"
          />
        </template>
      </ResponsiveDataView>
      <PaginationBar
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :total="total"
        @change="loadAbnormalUsers"
      />
    </el-card>
    <AppDrawer
      v-model="showUserDetailsDialog"
      title="用户详细信息"
      size="700px"
      mobile-size="100%"
      class="user-details-drawer"
    >
      <div v-if="userDetails" class="user-details">
        <el-card class="detail-card" shadow="never">
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
        <el-card class="detail-card" shadow="never">
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
        <el-card class="detail-card" shadow="never" v-if="userDetails.subscription_resets && userDetails.subscription_resets.length > 0">
          <template #header>
            <span>订阅重置记录</span>
          </template>
          <el-table :data="userDetails.subscription_resets" class="data-table" :max-height="300" stripe border>
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
          <el-table :data="userDetails.recent_activities" class="data-table" :max-height="300" stripe border>
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
    </AppDrawer>
  </div>
</template>
<script>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from '@/utils/elementPlusServices'
import { Refresh, View, Check, Search } from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
import { useMobile } from '@/composables/useMobile'
import AppDrawer from '@/components/AppDrawer.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import EmptyState from '@/components/EmptyState.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import { confirmWarning } from '@/utils/confirmAction'
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
    Refresh, View, Check, Search, AppDrawer, PaginationBar, EmptyState, ResponsiveDataView
  },
  setup() {
    const route = useRoute()
    const loading = ref(false)
    const abnormalUsers = ref([])
    const currentPage = ref(1)
    const pageSize = ref(10)
    const total = ref(0)
    const showUserDetailsDialog = ref(false)
    const userDetails = ref(null)
    const isMobile = useMobile()
    const mobileAbnormalFields = computed(() => [
      { key: 'abnormal_type', label: '异常类型' },
      { key: 'abnormal_count', label: '异常次数' },
      { key: 'subscription_count', label: '订阅次数' },
      { key: 'reset_count', label: '重置次数' },
      { key: 'description', label: '异常描述', fullWidth: true },
      { key: 'last_activity', label: '最后活动' }
    ])
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
    const loadAbnormalUsers = async () => {
      loading.value = true
      try {
        const params = {
          page: currentPage.value,
          size: pageSize.value
        }
        if (filters.dateRange && filters.dateRange.length === 2) {
          params['date_range[]'] = filters.dateRange
        }
        if (filters.subscriptionCount !== null && filters.subscriptionCount !== undefined && filters.subscriptionCount > 0) {
          params.subscription_count = filters.subscriptionCount.toString()
        }
        if (filters.resetCount !== null && filters.resetCount !== undefined && filters.resetCount > 0) {
          params.reset_count = filters.resetCount.toString()
        }
        const response = await adminAPI.getAbnormalUsers(params)
        if (response.data && response.data.success) {
          const data = response.data.data
          if (data && Array.isArray(data.users)) {
            abnormalUsers.value = data.users
            total.value = data.total || 0
          } else if (Array.isArray(data)) {
            abnormalUsers.value = data
            total.value = data.length
          } else if (data && Array.isArray(data.list)) {
            abnormalUsers.value = data.list
            total.value = data.total || 0
          } else {
            abnormalUsers.value = []
            total.value = 0
          }
        } else {
          abnormalUsers.value = []
          total.value = 0
        }
      } catch (error) {
        ElMessage.error('加载异常用户失败: ' + (error.response?.data?.message || error.message))
        abnormalUsers.value = []
      } finally {
        loading.value = false
      }
    }
    const applyFilters = () => {
      currentPage.value = 1
      loadAbnormalUsers()
    }
    const resetFilters = () => {
      filters.dateRange = getDefaultDateRange()  // 重置为默认日期范围
      filters.subscriptionCount = 10  // 重置为默认值
      filters.resetCount = 3  // 重置为默认值
      currentPage.value = 1
      pageSize.value = 10
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
        await confirmWarning(
          `确定要将用户 ${user.username} 标记为正常吗？这将从异常列表中移除该用户。`,
          { title: '确认操作' }
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
      loadAbnormalUsers()
      const route = useRoute()
      const userId = route.query.user_id
      if (userId) {
        setTimeout(() => {
          viewUserDetails(Number(userId))
        }, 500)
      }
    })
    return {
      loading,
      abnormalUsers,
      currentPage,
      pageSize,
      total,
      filters,
      showUserDetailsDialog,
      userDetails,
      isMobile,
      abnormalTypeMap,
      mobileAbnormalFields,
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
.filter-section {
  margin-bottom: 20px;
  @media (max-width: 768px) {
    margin-bottom: 16px;
  }
  .filter-card {
    background: #fff;
    border: 1px solid #ebeef5;
    border-radius: 8px;

      .filter-content {
      display: grid;
      grid-template-columns: minmax(260px, 1.3fr) minmax(180px, 0.85fr) minmax(180px, 0.85fr) max-content;
      gap: 16px;
      align-items: flex-end;
      @media (max-width: 1180px) {
        grid-template-columns: repeat(2, minmax(0, 1fr));
      }
      @media (max-width: 768px) {
        grid-template-columns: 1fr;
        gap: 16px;
        align-items: stretch;
      }
      .filter-item {
        display: flex;
        align-items: center;
        gap: 8px;
        min-width: 0;
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
      .desktop-filter-actions {
        display: flex;
        gap: 8px;
        justify-content: flex-end;
        min-width: max-content;
        @media (max-width: 768px) {
          display: none;
        }
        .el-button {
          margin-left: 0;
        }
      }
      .mobile-filter-actions {
        display: none;
        @media (max-width: 768px) {
          display: grid;
          grid-template-columns: repeat(2, minmax(0, 1fr));
          gap: 12px;
          width: 100%;
          margin-top: 8px;
          .mobile-action-btn {
            width: 100%;
            min-width: 0;
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
}
.data-table {
  width: 100%;
}
.abnormal-users-data {
  margin-top: 16px;
  :deep(.field-full .field-value) {
    width: 100%;
    text-align: left;
  }
}
.abnormal-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  min-width: 0;
  .user-info {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1;
    min-width: 0;
    overflow: hidden;
  }
}
.user-text-button {
  appearance: none;
  border: 0;
  background: transparent;
  padding: 0;
  margin: 0;
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  text-align: left;
  cursor: pointer;
  .username {
    max-width: 100%;
    font-size: 16px;
    font-weight: 600;
    color: #303133;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .email {
    max-width: 100%;
    font-size: 13px;
    color: #909399;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
    border-radius: 4px;
  }
}
.mobile-highlight-value {
  color: #f56c6c;
  font-weight: 700;
}
.mobile-description-value {
  color: #303133;
  line-height: 1.5;
}
.mobile-abnormal-actions {
  display: flex;
  gap: 8px;
  width: 100%;
  .mobile-action-btn {
    flex: 1;
    min-height: 44px;
    margin: 0;
    touch-action: manipulation;
  }
}
.user-details {
  .detail-card {
    margin-bottom: 20px;
  }

  .stat-item {
    text-align: center;
    padding: 16px;
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
.user-details-drawer {
  .user-details {
    width: 100%;
    max-width: 100%;
    min-width: 0;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }

  .detail-card {
    border: 1px solid #ebeef5;
    border-radius: 8px;
  }

  :deep(.el-table) {
    min-width: 560px;
  }
}
.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}
@media (max-width: 768px) {
  .user-details-drawer {
    .user-details {
      :deep(.el-card__body) {
        padding: 14px;
      }
    }

    :deep(.el-table__inner-wrapper) {
      overflow-x: auto;
    }
  }
}
</style>
