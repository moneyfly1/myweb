<template>
  <div class="list-container login-history-container">
    <div class="stats-row" style="margin-top: 0;">
      <div class="stat-card">
        <div class="stat-number">{{ totalLogins }}</div>
        <div class="stat-label">总登录次数</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ uniqueIPs }}</div>
        <div class="stat-label">不同IP数量</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ uniqueCountries }}</div>
        <div class="stat-label">不同国家</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ lastLoginDays }}</div>
        <div class="stat-label">距上次登录(天)</div>
      </div>
    </div>
    <el-card class="list-card history-card">
      <template #header>
        <div class="card-header">
          <i class="el-icon-time"></i>
          登录记录
        </div>
      </template>
      <div class="table-wrapper">
        <el-table
          ref="historyTableRef"
          :data="loginHistory"
          v-loading="loading"
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
      </div>
      <div class="mobile-card-list" v-if="loginHistory.length > 0 || !loading">
        <div
          v-for="(item, index) in loginHistory"
          :key="index"
          class="mobile-card"
        >
          <div class="card-row">
            <span class="label">状态</span>
            <span class="value">
              <el-tag :type="item.status === 'success' ? 'success' : 'danger'" size="small">
                {{ item.status === 'success' ? '成功' : '失败' }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">登录时间</span>
            <span class="value">{{ formatTime(item.login_time) }}</span>
          </div>
          <div class="card-row">
            <span class="label">IP地址</span>
            <span class="value">
              <el-tag type="info" size="small">{{ item.ip_address || '未知' }}</el-tag>
            </span>
          </div>
          <div class="card-row" v-if="getLocationText(item.location, item.ip_address)">
            <span class="label">地区</span>
            <span class="value">
              <el-tag type="success" size="small">
                {{ getLocationText(item.location, item.ip_address) }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">设备</span>
            <span class="value">{{ getDeviceInfo(item.user_agent) }}</span>
          </div>
        </div>
        <el-empty v-if="loginHistory.length === 0 && !loading" description="暂无登录记录" />
      </div>
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
    const pageSize = ref(10)
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
<style scoped lang="scss">
@use '@/styles/list-common.scss';

.user-agent-text {
  display: inline-block;
  max-width: 200px;
  overflow: clip;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
