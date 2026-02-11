<template>
  <div class="login-history-container">
    <div class="page-header">
      <h1>登录历史</h1>
      <p>查看您的账户登录记录</p>
    </div>
    <el-card class="history-card">
      <template #header>
        <div class="card-header">
          <i class="el-icon-time"></i>
          登录记录
        </div>
      </template>
      <div v-if="loading" class="loading-container">
        <el-skeleton :rows="5" animated />
      </div>
      <div class="desktop-only table-wrapper">
        <el-table 
          ref="historyTableRef"
          v-if="loginHistory.length > 0"
          :data="loginHistory" 
          stripe
          border
          style="width: 100%"
          @header-dragend="handleHistoryColumnResize"
        >
          <el-table-column prop="login_time" label="登录时间" :width="columnWidths.login_time" resizable>
            <template #default="scope">
              {{ formatTime(scope.row.login_time) }}
            </template>
          </el-table-column>
          <el-table-column prop="ip_address" label="IP地址/地区" :width="columnWidths.ip_address" resizable>
            <template #default="scope">
              <div style="display: flex; flex-direction: column; gap: 4px;">
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
        <el-empty v-else description="暂无登录记录" />
      </div>
      <div class="mobile-only">
        <div v-if="loginHistory.length > 0" class="mobile-history-list">
          <div 
            v-for="(item, index) in loginHistory" 
            :key="index"
            class="mobile-history-card"
          >
            <div class="history-card-header">
              <el-tag :type="item.status === 'success' ? 'success' : 'danger'" size="small">
                {{ item.status === 'success' ? '成功' : '失败' }}
              </el-tag>
              <span class="history-time">{{ formatTime(item.login_time) }}</span>
            </div>
            <div class="history-card-body">
              <div class="history-card-row">
                <span class="history-label">IP地址：</span>
                <el-tag type="info" size="small">{{ item.ip_address || '未知' }}</el-tag>
              </div>
              <div class="history-card-row" v-if="getLocationText(item.location, item.ip_address)">
                <span class="history-label">地区：</span>
                <el-tag type="success" size="small">
                  {{ getLocationText(item.location, item.ip_address) }}
                </el-tag>
              </div>
              <div class="history-card-row">
                <span class="history-label">设备：</span>
                <span class="history-value">{{ getDeviceInfo(item.user_agent) }}</span>
              </div>
            </div>
          </div>
        </div>
        <el-empty v-else description="暂无登录记录" />
      </div>
      <div v-if="loginHistory.length > 0" class="pagination-container">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
    <el-card class="stats-card">
      <template #header>
        <div class="card-header">
          <i class="el-icon-data-analysis"></i>
          登录统计
        </div>
      </template>
      <el-row :gutter="20">
        <el-col :xs="12" :sm="12" :md="6">
          <div class="stat-item">
            <div class="stat-value">{{ totalLogins }}</div>
            <div class="stat-label">总登录次数</div>
          </div>
        </el-col>
        <el-col :xs="12" :sm="12" :md="6">
          <div class="stat-item">
            <div class="stat-value">{{ uniqueIPs }}</div>
            <div class="stat-label">不同IP数量</div>
          </div>
        </el-col>
        <el-col :xs="12" :sm="12" :md="6">
          <div class="stat-item">
            <div class="stat-value">{{ uniqueCountries }}</div>
            <div class="stat-label">不同国家</div>
          </div>
        </el-col>
        <el-col :xs="12" :sm="12" :md="6">
          <div class="stat-item">
            <div class="stat-value">{{ lastLoginDays }}</div>
            <div class="stat-label">距上次登录(天)</div>
          </div>
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>
<script>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { userAPI } from '@/utils/api'
import dayjs from 'dayjs'
import { formatLocation } from '@/utils/date'
export default {
  name: 'LoginHistory',
  setup() {
    const loading = ref(false)
    const loginHistory = ref([])
    const historyTableRef = ref(null)
    const LOGIN_HISTORY_TABLE_STORAGE_KEY = 'user_login_history_table_settings'
    const columnWidths = reactive({
      login_time: 180,
      ip_address: 200,
      user_agent: 200,
      status: 100
    })
    const loadHistoryTableSettings = () => {
      try {
        const saved = localStorage.getItem(LOGIN_HISTORY_TABLE_STORAGE_KEY)
        if (saved) {
          const s = JSON.parse(saved)
          if (s.columnWidths) Object.assign(columnWidths, s.columnWidths)
        }
      } catch (e) {
        console.warn('加载登录历史表设置失败:', e)
      }
    }
    const saveHistoryTableSettings = () => {
      try {
        localStorage.setItem(LOGIN_HISTORY_TABLE_STORAGE_KEY, JSON.stringify({ columnWidths: { ...columnWidths } }))
      } catch (e) {
        console.warn('保存登录历史表设置失败:', e)
      }
    }
    const HISTORY_COLUMN_KEYS = ['login_time', 'ip_address', 'user_agent', 'status']
    let historyResizeTimer = null
    const handleHistoryColumnResize = () => {
      if (historyResizeTimer) clearTimeout(historyResizeTimer)
      historyResizeTimer = setTimeout(() => {
        if (historyTableRef.value && historyTableRef.value.$el) {
          const cells = historyTableRef.value.$el.querySelectorAll('.el-table__header-wrapper thead th')
          cells.forEach((cell, index) => {
            if (HISTORY_COLUMN_KEYS[index] && cell.offsetWidth > 0) columnWidths[HISTORY_COLUMN_KEYS[index]] = cell.offsetWidth
          })
          saveHistoryTableSettings()
        }
      }, 300)
    }
    const currentPage = ref(1)
    const pageSize = ref(20)
    const total = ref(0)
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
      if (!time) return '未知'
      return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
    }
    const getDeviceInfo = (userAgent) => {
      if (!userAgent) return '未知设备'
      if (userAgent.includes('Mobile')) {
        return '移动设备'
      } else if (userAgent.includes('Windows')) {
        return 'Windows设备'
      } else if (userAgent.includes('Mac')) {
        return 'Mac设备'
      } else if (userAgent.includes('Linux')) {
        return 'Linux设备'
      } else {
        return '其他设备'
      }
    }
    const handleSizeChange = (val) => {
      pageSize.value = val
      fetchLoginHistory()
    }
    const handleCurrentChange = (val) => {
      currentPage.value = val
      fetchLoginHistory()
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
      if (location) {
        return formatLocation(location)
      }
      if (ipAddress) {
        if (ipAddress === '127.0.0.1' || ipAddress === '::1' || ipAddress === 'localhost') {
          return '本地'
        }
        if (ipAddress.startsWith('192.168.') || ipAddress.startsWith('10.') || ipAddress.startsWith('172.')) {
          return '内网'
        }
      }
      return ''
    }
    onMounted(() => {
      loadHistoryTableSettings()
      fetchLoginHistory()
    })
    return {
      loading,
      loginHistory,
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
      totalLogins,
      uniqueIPs,
      uniqueCountries,
      lastLoginDays
    }
  }
}
</script>
<style scoped>
.login-history-container {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}
.page-header {
  margin-bottom: 2rem;
  text-align: center;
}
.page-header h1 {
  color: #303133;
  margin-bottom: 0.5rem;
}
.page-header :is(p) {
  color: #909399;
  margin: 0;
}
.history-card,
.stats-card {
  margin-bottom: 20px;
}
.card-header {
  display: flex;
  align-items: center;
  font-weight: 600;
  color: #303133;
}
.card-header :is(i) {
  margin-right: 8px;
  color: #409eff;
}
.loading-container {
  padding: 20px;
}
.user-agent-text {
  display: inline-block;
  max-width: 200px;
  overflow: clip;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: center;
}
.stat-item {
  text-align: center;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
}
.stat-value {
  font-size: 2rem;
  font-weight: bold;
  color: #409eff;
  margin-bottom: 8px;
}
.stat-label {
  color: #909399;
  font-size: 0.9rem;
}
.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}
.mobile-only {
  display: none;
  @media (max-width: 768px) {
    display: block;
  }
}
@media (max-width: 768px) {
  .login-history-container {
    padding: 10px;
  }
  .page-header {
    margin-bottom: 16px;
    :is(h1) {
      font-size: 20px;
    }
    :is(p) {
      font-size: 13px;
    }
  }
  .stat-item {
    padding: 15px;
    margin-bottom: 12px;
  }
  .stat-value {
    font-size: 1.5rem;
  }
  .stat-label {
    font-size: 13px;
  }
  .mobile-history-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .mobile-history-card {
    background: #ffffff;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    padding: 14px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }
  .history-card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    padding-bottom: 10px;
    border-bottom: 1px solid #f0f0f0;
    .history-time {
      font-size: 12px;
      color: #909399;
    }
  }
  .history-card-body {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .history-card-row {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    .history-label {
      font-weight: 600;
      color: #606266;
      min-width: 70px;
      flex-shrink: 0;
    }
    .history-value {
      color: #303133;
      flex: 1;
    }
  }
  .pagination-container {
    margin-top: 16px;
    :deep(.el-pagination) {
      justify-content: center;
      flex-wrap: wrap;
      .el-pagination__sizes,
      .el-pagination__jump {
        display: none;
      }
    }
  }
}
</style>
