<template>
  <div class="list-container login-history-container">
    <div class="breadcrumb">首页 / 登录历史</div>
    <div class="page-header">
      <div class="page-title">
        <h1>登录历史</h1>
      </div>
      <div class="actions">
        <el-button @click="fetchLoginHistory" :loading="loading">
          刷新
        </el-button>
      </div>
    </div>

    <div class="stats-row history-stats-row">
      <div class="stat-card">
        <div class="stat-icon">L</div>
        <div>
          <div class="stat-value">{{ totalLogins }}</div>
          <div class="stat-label">总登录次数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">IP</div>
        <div>
          <div class="stat-value">{{ uniqueIPs }}</div>
          <div class="stat-label">不同 IP 数量</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">C</div>
        <div>
          <div class="stat-value">{{ uniqueCountries }}</div>
          <div class="stat-label">不同国家</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">D</div>
        <div>
          <div class="stat-value">{{ lastLoginDays }}</div>
          <div class="stat-label">距上次登录天数</div>
        </div>
      </div>
    </div>

    <div class="card list-filter-card history-filter-card">
      <div class="card-body history-filter-body">
        <el-form :inline="true" :model="filters" class="history-filter-form list-filter-form">
          <el-form-item label="搜索">
            <el-input
              v-model="filters.keyword"
              placeholder="IP、地区、设备"
              clearable
              class="history-keyword-input"
              @keyup.enter="applyFilters"
              @clear="applyFilters"
            />
          </el-form-item>
          <el-form-item label="登录状态">
            <el-select v-model="filters.status" placeholder="全部状态" clearable class="history-filter-select" @change="applyFilters">
              <el-option label="成功" value="success" />
              <el-option label="失败" value="failed" />
            </el-select>
          </el-form-item>
          <el-form-item label="登录时间">
            <el-date-picker
              v-model="filters.date_range"
              type="daterange"
              range-separator="至"
              start-placeholder="开始日期"
              end-placeholder="结束日期"
              format="YYYY-MM-DD"
              value-format="YYYY-MM-DD"
              class="history-date-filter"
            />
          </el-form-item>
          <el-form-item class="history-filter-actions">
            <el-button type="primary" @click="applyFilters">搜索</el-button>
            <el-button :disabled="!hasActiveFilters" @click="resetFilters">重置</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <div class="card history-card">
      <div class="card-header">
        <h2 class="card-title">登录记录</h2>
      </div>
      <div class="table-wrap">
        <ResponsiveDataView
          :data="paginatedLoginHistory"
          :fields="mobileHistoryFields"
          :loading="loading"
          title-field="ip_address"
          empty-title="暂无登录记录"
          empty-description="登录后将展示时间、IP、地区、设备和状态。"
        >
          <template #table>
            <div class="table-wrapper">
              <el-table
                ref="historyTableRef"
                :data="paginatedLoginHistory"
                v-loading="loading"
                stripe
                border
                class="history-table"
                @header-dragend="handleHistoryColumnResize"
              >
                <template #empty>
                  <EmptyState
                    title="暂无登录记录"
                    description="登录后将展示时间、IP、地区、设备和状态。"
                    action-text="刷新"
                    :loading="loading"
                    @action="fetchLoginHistory"
                  />
                </template>
                <el-table-column prop="login_time" label="登录时间" :width="columnWidths.login_time" resizable>
                  <template #default="scope">
                    {{ formatTime(scope.row.login_time) }}
                  </template>
                </el-table-column>
                <el-table-column prop="ip_address" label="IP地址/地区" :width="columnWidths.ip_address" resizable>
                  <template #default="scope">
                    <div class="ip-location-stack">
                      <el-tag type="info" size="small">{{ scope.row.ip_address || '未知' }}</el-tag>
                      <el-tag
                        v-if="getLocationText(scope.row.location, scope.row.ip_address)"
                        type="success"
                        size="small"
                      >
                        {{ getLocationText(scope.row.location, scope.row.ip_address) }}
                      </el-tag>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column prop="user_agent" label="设备信息" :min-width="columnWidths.user_agent" resizable>
                  <template #default="scope">
                    <el-tooltip :content="scope.row.user_agent" placement="top">
                      <span class="user-agent-text">
                        {{ getDeviceInfo(scope.row.user_agent) }}
                      </span>
                    </el-tooltip>
                  </template>
                </el-table-column>
                <el-table-column prop="status" label="状态" :width="columnWidths.status" resizable>
                  <template #default="scope">
                    <el-tag :type="scope.row.status === 'success' ? 'success' : 'danger'">
                      {{ scope.row.status === 'success' ? '成功' : '失败' }}
                    </el-tag>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </template>
          <template #header="{ item }">
            <div class="mobile-history-header">
              <span>{{ item.ip_address || '未知 IP' }}</span>
              <el-tag :type="item.status === 'success' ? 'success' : 'danger'" size="small">
                {{ item.status === 'success' ? '成功' : '失败' }}
              </el-tag>
            </div>
          </template>
          <template #field-location="{ item }">
            <el-tag v-if="getLocationText(item.location, item.ip_address)" type="success" size="small">
              {{ getLocationText(item.location, item.ip_address) }}
            </el-tag>
            <span v-else>-</span>
          </template>
          <template #empty>
            <EmptyState
              title="暂无登录记录"
              description="登录后将展示时间、IP、地区、设备和状态。"
              action-text="刷新"
              :loading="loading"
              @action="fetchLoginHistory"
            />
          </template>
        </ResponsiveDataView>
      </div>
    </div>

    <PaginationBar
      v-if="filteredLoginHistory.length > 0"
      v-model:current-page="currentPage"
      v-model:page-size="pageSize"
      :total="filteredLoginHistory.length"
      :page-sizes="[10, 20, 50, 100]"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
    />
  </div>
</template>
<script>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { userAPI } from '@/utils/api'
import { Clock } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import { formatDateTimeSafe, getLocationText as getLocationTextUtil } from '@/utils/date'
import { getDeviceTypeFromUA as getDeviceInfo } from '@/utils/device'
import { usePersistentTableColumns } from '@/composables/usePersistentTableColumns'
import EmptyState from '@/components/EmptyState.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
export default {
  name: 'LoginHistory',
  components: {
    EmptyState,
    PaginationBar,
    ResponsiveDataView,
    Clock
  },
  setup() {
    const loading = ref(false)
    const loginHistory = ref([])
    const historyTableRef = ref(null)
    const LOGIN_HISTORY_TABLE_STORAGE_KEY = 'user_login_history_table_settings'
    const HISTORY_COLUMN_KEYS = ['login_time', 'ip_address', 'user_agent', 'status']
    const { columnWidths, handleColumnResize: handleHistoryColumnResize } = usePersistentTableColumns(
      LOGIN_HISTORY_TABLE_STORAGE_KEY,
      {
      login_time: 180,
      ip_address: 200,
      user_agent: 200,
      status: 100
      },
      HISTORY_COLUMN_KEYS
    )
    const currentPage = ref(1)
    const pageSize = ref(10)
    const total = ref(0)
    const filters = reactive({
      keyword: '',
      status: '',
      date_range: []
    })
    const hasActiveFilters = computed(() => Boolean(filters.keyword || filters.status || filters.date_range.length > 0))
    const filteredLoginHistory = computed(() => {
      const keyword = String(filters.keyword || '').trim().toLowerCase()
      const hasDateRange = Array.isArray(filters.date_range) && filters.date_range.length === 2
      const start = hasDateRange ? dayjs(filters.date_range[0]).startOf('day') : null
      const end = hasDateRange ? dayjs(filters.date_range[1]).endOf('day') : null
      return loginHistory.value.filter(item => {
        if (filters.status && item.status !== filters.status) {
          return false
        }
        if (hasDateRange) {
          const loginTime = dayjs(item.login_time)
          if (!loginTime.isValid() || loginTime.isBefore(start) || loginTime.isAfter(end)) {
            return false
          }
        }
        if (keyword) {
          const haystack = [
            item.ip_address,
            item.location,
            item.country,
            item.city,
            item.user_agent,
            getDeviceInfo(item.user_agent),
            getLocationText(item.location, item.ip_address)
          ].filter(Boolean).join(' ').toLowerCase()
          return haystack.includes(keyword)
        }
        return true
      })
    })
    const paginatedLoginHistory = computed(() => {
      const start = (currentPage.value - 1) * pageSize.value
      const end = start + pageSize.value
      return filteredLoginHistory.value.slice(start, end)
    })
    const fetchLoginHistory = async () => {
      loading.value = true
      try {
        const response = await userAPI.getLoginHistory()
        let data = null
        if (response && response.data) {
          if (response.data.success !== false) {
            if (Array.isArray(response.data.data)) {
              data = response.data.data
            } else if (response.data.data && Array.isArray(response.data.data)) {
              data = response.data.data
            } else {
              data = response.data.data
            }
          }
        } else if (response && Array.isArray(response)) {
          data = response
        }
        if (Array.isArray(data)) {
          loginHistory.value = data.map(item => ({
            login_time: item.login_time || '',
            ip_address: item.ip_address || '',
            location: item.location || '',
            country: item.country || '',
            city: item.city || '',
            user_agent: item.user_agent || '',
            login_status: item.login_status || item.status || 'success',
            status: item.login_status || item.status || 'success' // 兼容字段
          }))
          total.value = loginHistory.value.length
        } else if (data && data.logins && Array.isArray(data.logins)) {
          loginHistory.value = data.logins
          total.value = data.total || data.logins.length
        } else {
          loginHistory.value = []
          total.value = 0
        }
      } catch (error) {
        console.error('获取登录历史失败:', error)
        ElMessage.error(`获取登录历史失败: ${error.response?.data?.message || error.message || '未知错误'}`)
        loginHistory.value = []
        total.value = 0
      } finally {
        loading.value = false
      }
    }
    const formatTime = (time) => {
      return formatDateTimeSafe(time, 'YYYY-MM-DD HH:mm:ss', '未知')
    }
    const handleSizeChange = (val) => {
      pageSize.value = val
      currentPage.value = 1
    }
    const handleCurrentChange = (val) => {
      currentPage.value = val
    }
    const applyFilters = () => {
      currentPage.value = 1
    }
    const resetFilters = () => {
      filters.keyword = ''
      filters.status = ''
      filters.date_range = []
      currentPage.value = 1
    }
    const totalLogins = computed(() => {
      return loginHistory.value.length
    })
    const uniqueIPs = computed(() => {
      const ips = new Set(loginHistory.value.map(item => item.ip_address).filter(Boolean))
      return ips.size
    })
    const uniqueCountries = computed(() => {
      const countries = new Set(loginHistory.value.map(item => item.country).filter(Boolean))
      return countries.size
    })
    const lastLoginDays = computed(() => {
      if (loginHistory.value.length === 0) return 0
      const lastLogin = loginHistory.value[0]?.login_time
      if (!lastLogin) return 0
      return dayjs().diff(dayjs(lastLogin), 'day')
    })
    const getLocationText = (location, ipAddress) => {
      return getLocationTextUtil(location, ipAddress)
    }
    const mobileHistoryFields = computed(() => [
      {
        key: 'status',
        label: '状态',
        type: 'tag',
        tagType: value => value === 'success' ? 'success' : 'danger',
        formatter: value => value === 'success' ? '成功' : '失败'
      },
      { key: 'login_time', label: '登录时间', formatter: value => formatTime(value) },
      { key: 'ip_address', label: 'IP地址', formatter: value => value || '未知' },
      { key: 'location', label: '地区' },
      { key: 'user_agent', label: '设备', formatter: value => getDeviceInfo(value), fullWidth: true }
    ])
    onMounted(() => {
      fetchLoginHistory()
    })
    return {
      loading,
      loginHistory,
      filters,
      hasActiveFilters,
      filteredLoginHistory,
      paginatedLoginHistory,
      currentPage,
      pageSize,
      total,
      historyTableRef,
      columnWidths,
      handleHistoryColumnResize,
      fetchLoginHistory,
      formatTime,
      getDeviceInfo,
      getLocationText,
      handleSizeChange,
      handleCurrentChange,
      applyFilters,
      resetFilters,
      totalLogins,
      uniqueIPs,
      uniqueCountries,
      lastLoginDays,
      mobileHistoryFields
    }
  }
}
</script>
<style scoped lang="scss">
.user-agent-text {
  display: inline-block;
  max-width: 200px;
  overflow: clip;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.history-stats-row {
  margin-top: 0;
}

.history-filter-card {
  margin-bottom: 14px;
}

.history-filter-body {
  padding: 16px;
}

.history-filter-form {
  display: grid;
  grid-template-columns: minmax(220px, 1.15fr) minmax(150px, 0.7fr) minmax(300px, 1.3fr) minmax(150px, max-content);
  align-items: end;
  gap: 12px;
  width: 100%;

  :deep(.el-form-item) {
    margin: 0;
    min-width: 0;
  }

  :deep(.el-form-item__label) {
    color: #606266;
    font-weight: 600;
  }
}

.history-keyword-input {
  width: 100%;
  min-width: 0;
}

.history-filter-select {
  width: 100%;
  min-width: 0;
}

.history-date-filter {
  width: 100%;
  min-width: 0;
}

.history-filter-actions {
  justify-self: end;

  :deep(.el-form-item__content) {
    display: flex;
    flex-wrap: nowrap;
    gap: 8px;
  }
}

@media (max-width: 1100px) {
  .history-filter-form {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .history-filter-actions {
    justify-self: start;
  }
}

.history-table {
  width: 100%;
}

.card-header-icon {
  margin-right: 6px;
  color: #1677ff;
  flex-shrink: 0;
}

.ip-location-stack {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.mobile-history-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;

  span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 600;
  }
}

@media (max-width: 768px) {
  .history-filter-form {
    display: grid;
    grid-template-columns: 1fr;
    align-items: stretch;
  }

  .history-keyword-input,
  .history-filter-select,
  .history-date-filter,
  .history-filter-actions,
  .history-filter-actions :deep(.el-form-item__content),
  .history-filter-actions :deep(.el-button) {
    width: 100%;
  }
}
</style>
