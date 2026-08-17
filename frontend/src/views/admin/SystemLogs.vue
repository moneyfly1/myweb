<template>
  <div class="list-container admin-system-logs">
    <el-card>
      <template #header>
        <div class="card-header">
          <h2>系统日志</h2>
          <p>查看和管理系统运行日志</p>
        </div>
      </template>
      <div class="logs-filter">
        <div class="desktop-only system-log-filter-grid">
          <div class="system-log-filter-field">
              <el-form-item label="日志类型">
                <el-select v-model="filterForm.log_type" placeholder="选择日志类型" clearable @change="applyFilter">
                  <el-option label="全部" value="" />
                  <el-option label="错误" value="error" />
                  <el-option label="警告" value="warning" />
                  <el-option label="信息" value="info" />
                  <el-option label="调试" value="debug" />
                </el-select>
              </el-form-item>
          </div>
          <div class="system-log-filter-field">
              <el-form-item label="日志级别">
                <el-select v-model="filterForm.log_level" placeholder="选择日志级别" clearable @change="applyFilter">
                  <el-option label="全部" value="" />
                  <el-option label="严重" value="critical" />
                  <el-option label="错误" value="error" />
                  <el-option label="警告" value="warning" />
                  <el-option label="信息" value="info" />
                  <el-option label="调试" value="debug" />
                </el-select>
              </el-form-item>
          </div>
          <div class="system-log-filter-field">
              <el-form-item label="开始时间">
                <el-date-picker
                  v-model="filterForm.start_time"
                  type="datetime"
                  placeholder="选择开始时间"
                  format="YYYY-MM-DD HH:mm:ss"
                  value-format="YYYY-MM-DD HH:mm:ss"
                  @change="applyFilter"
                />
              </el-form-item>
          </div>
          <div class="system-log-filter-field">
              <el-form-item label="结束时间">
                <el-date-picker
                  v-model="filterForm.end_time"
                  type="datetime"
                  placeholder="选择结束时间"
                  format="YYYY-MM-DD HH:mm:ss"
                  value-format="YYYY-MM-DD HH:mm:ss"
                  @change="applyFilter"
                />
              </el-form-item>
          </div>
          <div class="system-log-filter-field keyword-field">
              <el-form-item label="关键词搜索">
                <el-input
                  v-model="filterForm.keyword"
                  placeholder="搜索日志内容、模块、用户等"
                  clearable
                  @input="debouncedApplyFilter"
                  @clear="applyFilter"
                />
              </el-form-item>
          </div>
          <div class="system-log-filter-field">
              <el-form-item label="任务类型">
                <el-select v-model="filterForm.task_type" placeholder="选择任务类型" clearable @change="applyFilter">
                  <el-option label="全部" value="" />
                  <el-option label="定时任务调度器" value="scheduler" />
                  <el-option label="邮件队列" value="email_queue" />
                  <el-option label="自动备份" value="auto_backup" />
                  <el-option label="节点更新" value="auto_node_update" />
                  <el-option label="节点健康检查" value="node_health_check" />
                  <el-option label="订阅到期检查" value="expiring_subscriptions" />
                  <el-option label="系统错误" value="system_error" />
                </el-select>
              </el-form-item>
          </div>
          <div class="filter-actions">
            <el-button type="primary" @click="applyFilter" :loading="loading">
              <el-icon><Search /></el-icon>
              搜索
            </el-button>
            <el-button @click="resetFilter">
              <el-icon><Refresh /></el-icon>
              重置
            </el-button>
            <el-button type="success" @click="exportLogs">
              <el-icon><Download /></el-icon>
              导出日志
            </el-button>
            <el-button type="warning" @click="clearLogs">
              <el-icon><Delete /></el-icon>
              清理日志
            </el-button>
          </div>
        </div>
        <div class="mobile-only">
          <el-form :model="filterForm" label-position="top">
            <el-form-item label="日志类型">
              <el-select v-model="filterForm.log_type" placeholder="选择日志类型" clearable class="full-width-control" @change="applyFilter">
                <el-option label="全部" value="" />
                <el-option label="错误" value="error" />
                <el-option label="警告" value="warning" />
                <el-option label="信息" value="info" />
                <el-option label="调试" value="debug" />
              </el-select>
            </el-form-item>
            <el-form-item label="日志级别">
              <el-select v-model="filterForm.log_level" placeholder="选择日志级别" clearable class="full-width-control" @change="applyFilter">
                <el-option label="全部" value="" />
                <el-option label="严重" value="critical" />
                <el-option label="错误" value="error" />
                <el-option label="警告" value="warning" />
                <el-option label="信息" value="info" />
                <el-option label="调试" value="debug" />
              </el-select>
            </el-form-item>
            <el-form-item label="开始时间">
              <el-date-picker
                v-model="filterForm.start_time"
                type="datetime"
                placeholder="选择开始时间"
                format="YYYY-MM-DD HH:mm:ss"
                value-format="YYYY-MM-DD HH:mm:ss"
                class="full-width-control"
                @change="applyFilter"
              />
            </el-form-item>
            <el-form-item label="结束时间">
              <el-date-picker
                v-model="filterForm.end_time"
                type="datetime"
                placeholder="选择结束时间"
                format="YYYY-MM-DD HH:mm:ss"
                value-format="YYYY-MM-DD HH:mm:ss"
                class="full-width-control"
                @change="applyFilter"
              />
            </el-form-item>
            <el-form-item label="关键词搜索">
              <el-input
                v-model="filterForm.keyword"
                placeholder="搜索日志内容、模块、用户等"
                clearable
                @input="debouncedApplyFilter"
                @clear="applyFilter"
              />
            </el-form-item>
            <el-form-item label="模块">
              <el-select v-model="filterForm.module" placeholder="选择模块" clearable class="full-width-control" @change="applyFilter">
                <el-option label="全部" value="" />
                <el-option label="用户管理" value="user" />
                <el-option label="订单管理" value="order" />
                <el-option label="支付系统" value="payment" />
                <el-option label="邮件系统" value="email" />
                <el-option label="系统配置" value="config" />
                <el-option label="认证系统" value="auth" />
              </el-select>
            </el-form-item>
            <el-form-item label="用户">
              <el-input
                v-model="filterForm.username"
                placeholder="输入用户名"
                clearable
                @input="debouncedApplyFilter"
                @clear="applyFilter"
              />
            </el-form-item>
          </el-form>
          <div class="filter-actions mobile-filter-actions">
            <el-button type="primary" @click="applyFilter" :loading="loading" class="mobile-action-btn">
              <el-icon><Search /></el-icon>
              搜索
            </el-button>
            <el-button @click="resetFilter" class="mobile-action-btn">
              <el-icon><Refresh /></el-icon>
              重置
            </el-button>
            <el-button type="success" @click="exportLogs" class="mobile-action-btn">
              <el-icon><Download /></el-icon>
              导出
            </el-button>
            <el-button type="warning" @click="clearLogs" class="mobile-action-btn">
              <el-icon><Delete /></el-icon>
              清理
            </el-button>
          </div>
        </div>
      </div>
      <div class="logs-stats">
        <el-row :gutter="20">
          <el-col :xs="12" :sm="12" :md="6">
            <el-card class="stat-card clickable" @click="filterByLevel('')">
              <div class="stat-content">
                <div class="stat-number">{{ logsStats.total || 0 }}</div>
                <div class="stat-label">总日志数</div>
              </div>
            </el-card>
          </el-col>
          <el-col :xs="12" :sm="12" :md="6">
            <el-card class="stat-card clickable" @click="filterByLevel('error')">
              <div class="stat-content">
                <div class="stat-number error">{{ logsStats.error || 0 }}</div>
                <div class="stat-label">错误日志</div>
              </div>
            </el-card>
          </el-col>
          <el-col :xs="12" :sm="12" :md="6">
            <el-card class="stat-card clickable" @click="filterByLevel('warning')">
              <div class="stat-content">
                <div class="stat-number warning">{{ logsStats.warning || 0 }}</div>
                <div class="stat-label">警告日志</div>
              </div>
            </el-card>
          </el-col>
          <el-col :xs="12" :sm="12" :md="6">
            <el-card class="stat-card clickable" @click="filterByLevel('info')">
              <div class="stat-content">
                <div class="stat-number info">{{ logsStats.info || 0 }}</div>
                <div class="stat-label">信息日志</div>
              </div>
            </el-card>
          </el-col>
        </el-row>
      </div>
      <div class="logs-table">
        <ResponsiveDataView
          :data="logsList"
          :loading="loading"
          :fields="mobileLogFields"
          empty-title="暂无日志数据"
          empty-description="调整筛选条件后可重新查询系统日志"
        >
          <template #table>
          <el-table
            :data="logsList"
            v-loading="loading"
            class="logs-data-table"
            stripe
            border
            empty-text=" "
            :default-sort="{ prop: 'timestamp', order: 'descending' }"
          >
            <el-table-column prop="timestamp" label="时间" width="180" sortable>
              <template #default="{ row }">
                {{ formatDate(row.timestamp) }}
              </template>
            </el-table-column>
            <el-table-column prop="level" label="级别" width="100">
              <template #default="{ row }">
                <el-tag :type="getLogLevelTagType(row.level)">
                  {{ getLogLevelText(row.level) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="action_type" label="任务类型" width="150">
              <template #default="{ row }">
                <el-tag v-if="row.action_type" size="small" type="info">
                  {{ getTaskTypeName(row.action_type) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="message" label="日志内容" min-width="300">
              <template #default="{ row }">
                <div class="log-message">
                  <div class="message-text">{{ row.message }}</div>
                  <div v-if="row.failure_reason" class="failure-reason-inline">
                    <el-tag type="warning" size="small">{{ row.failure_reason }}</el-tag>
                  </div>
                  <el-button
                    v-if="row.details || row.failure_reason"
                    type="text"
                    size="small"
                    @click="showLogDetails(row)"
                  >
                    详情
                  </el-button>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="ip_address" label="IP地址" width="140">
              <template #default="{ row }">
                <div>
                  <div>{{ row.ip_address }}</div>
                  <div v-if="row.location" class="location-text">
                    <el-tag size="small" type="info">{{ displayLocation(row.location) }}</el-tag>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="user_agent" label="用户代理" width="200">
              <template #default="{ row }">
                <el-tooltip :content="row.user_agent" placement="top">
                  <span class="user-agent-text">{{ truncateText(row.user_agent, 30) }}</span>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button
                  size="small"
                  type="primary"
                  @click="showLogDetails(row)"
                >
                  详情
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          </template>
          <template #header="{ item }">
            <div class="mobile-log-header">
              <el-tag :type="getLogLevelTagType(item.level)" size="small">
                {{ getLogLevelText(item.level) }}
              </el-tag>
              <span class="mobile-log-time">{{ formatDate(item.timestamp) }}</span>
            </div>
          </template>
          <template #actions="{ item }">
            <el-button size="small" type="primary" @click="showLogDetails(item)">
              查看详情
            </el-button>
          </template>
          <template #empty>
            <EmptyState
              title="暂无日志数据"
              description="调整筛选条件后可重新查询系统日志"
            />
          </template>
        </ResponsiveDataView>
        <div class="pagination-wrapper">
          <PaginationBar
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.size"
            :page-sizes="[10, 20, 50, 100]"
            :total="pagination.total"
            @size-change="handleSizeChange"
            @current-change="handleCurrentChange"
          />
        </div>
      </div>
    </el-card>
    <AppDrawer
      v-model="logDetailsVisible"
      title="日志详情"
      size="600px"
      mobile-size="100%"
    >
      <div v-if="selectedLog" class="log-details">
        <el-descriptions :column="isMobile ? 1 : 2" border>
          <el-descriptions-item label="时间">
            {{ formatDate(selectedLog.timestamp) }}
          </el-descriptions-item>
          <el-descriptions-item label="级别">
            <el-tag :type="getLogLevelTagType(selectedLog.level)">
              {{ getLogLevelText(selectedLog.level) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="模块">
            {{ selectedLog.module }}
          </el-descriptions-item>
          <el-descriptions-item label="用户">
            {{ selectedLog.username || '系统' }}
          </el-descriptions-item>
          <el-descriptions-item label="IP地址">
            {{ selectedLog.ip_address }}
          </el-descriptions-item>
          <el-descriptions-item label="地理位置" v-if="selectedLog.location">
            <el-tag type="info">{{ displayLocation(selectedLog.location) }}</el-tag>
            <div v-if="selectedLog.location_info" class="location-details">
              <div v-if="selectedLog.location_info.country">国家: {{ selectedLog.location_info.country }}</div>
              <div v-if="selectedLog.location_info.city">城市: {{ selectedLog.location_info.city }}</div>
              <div v-if="selectedLog.location_info.region">地区: {{ selectedLog.location_info.region }}</div>
            </div>
          </el-descriptions-item>
          <el-descriptions-item label="用户代理">
            {{ selectedLog.user_agent }}
          </el-descriptions-item>
        </el-descriptions>
        <div class="log-message-section">
          <h4>日志内容</h4>
          <div class="log-message-content">{{ selectedLog.message }}</div>
        </div>
        <div v-if="selectedLog.failure_reason" class="log-failure-reason">
          <h4>失败原因</h4>
          <div class="failure-reason-content">
            <el-tag type="warning" size="small">{{ selectedLog.failure_reason }}</el-tag>
          </div>
        </div>
        <div v-if="selectedLog.details" class="log-details-section">
          <h4>详细信息</h4>
          <pre class="log-details-content">{{ selectedLog.details }}</pre>
        </div>
        <div v-if="selectedLog.stack_trace" class="log-stack-section">
          <h4>堆栈跟踪</h4>
          <pre class="log-stack-content">{{ selectedLog.stack_trace }}</pre>
        </div>
        <div v-if="selectedLog.context" class="log-context-section">
          <h4>上下文信息</h4>
          <pre class="log-context-content">{{ JSON.stringify(selectedLog.context, null, 2) }}</pre>
        </div>
      </div>
      <template #footer>
        <FormActionBar
          cancel-text="关闭"
          submit-text="复制详情"
          :sticky="false"
          @cancel="closeLogDetails"
          @submit="copyLogDetails"
        />
      </template>
    </AppDrawer>
  </div>
</template>
<script>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { Search, Refresh, Download, Delete } from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
import { formatLocation } from '@/utils/date'
import { confirmClear } from '@/utils/confirmAction'
import AppDrawer from '@/components/AppDrawer.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import EmptyState from '@/components/EmptyState.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import { useMobile } from '@/composables/useMobile'
import { debounce } from '@/composables/useDebounce'
export default {
  name: 'AdminSystemLogs',
  components: {
    Search, Refresh, Download, Delete, AppDrawer, FormActionBar, PaginationBar, EmptyState, ResponsiveDataView
  },
  setup() {
    const isMobile = useMobile()
    const loading = ref(false)
    const logsList = ref([])
    const logsStats = ref({})
    const logDetailsVisible = ref(false)
    const selectedLog = ref(null)
    const mobileLogFields = computed(() => [
      {
        key: 'action_type',
        label: '任务类型',
        type: 'tag',
        tagType: () => 'info',
        formatter: value => value ? getTaskTypeName(value) : '-'
      },
      { key: 'module', label: '模块' },
      {
        key: 'message',
        label: '内容',
        fullWidth: true,
        formatter: value => truncateText(value, 50) || '-'
      },
      {
        key: 'failure_reason',
        label: '失败原因',
        type: 'tag',
        hideTagWhenEmpty: true,
        tagType: () => 'warning',
        formatter: value => value || '-'
      },
      { key: 'username', label: '用户' },
      { key: 'ip_address', label: 'IP' },
      {
        key: 'location',
        label: '地理位置',
        type: 'tag',
        hideTagWhenEmpty: true,
        tagType: () => 'info',
        formatter: value => value ? displayLocation(value) : '-'
      }
    ])
    const filterForm = reactive({
      log_type: '',
      log_level: '',
      start_time: '',
      end_time: '',
      keyword: '',
      task_type: '',
      module: '',
      username: ''
    })
    const pagination = reactive({
      page: 1,
      size: 10,
      total: 0
    })
    const loadLogs = async () => {
      loading.value = true
      try {
        const params = {
          page: pagination.page,
          size: pagination.size,
          ...filterForm
        }
        const response = await adminAPI.getSystemLogs(params)
        const data = response?.data?.data ?? response?.data ?? {}
        logsList.value = data.logs || []
        pagination.total = data.total || 0
      } catch (error) {
        const errorMsg = error.response?.data?.message || error.message || '加载日志失败'
        ElMessage.error(errorMsg)
        console.error('加载日志失败:', error)
      } finally {
        loading.value = false
      }
    }
    const buildStatsParams = () => {
      return {
        start_time: filterForm.start_time,
        end_time: filterForm.end_time,
        keyword: filterForm.keyword,
        task_type: filterForm.task_type,
        module: filterForm.module,
        username: filterForm.username
      }
    }
    const loadLogsStats = async () => {
      try {
        const response = await adminAPI.getLogsStats(buildStatsParams())
        if (response && response.data && response.data.success) {
          logsStats.value = response.data.data || {}
        } else {
          console.error('获取日志统计失败:', response?.data?.message || response?.message)
        }
      } catch (error) {
        console.error('获取日志统计失败:', error)
      }
    }
    const applyFilter = () => {
      pagination.page = 1
      loadLogs()
      loadLogsStats()
    }
    // 关键词输入实时生效，无需再次点击搜索按钮（500ms 防抖）
    const debouncedApplyFilter = debounce(applyFilter, 500)
    const resetFilter = () => {
      Object.keys(filterForm).forEach(key => {
        filterForm[key] = ''
      })
      pagination.page = 1
      loadLogs()
      loadLogsStats()
    }
    const filterByLevel = (level) => {
      filterForm.log_level = level
      pagination.page = 1
      loadLogs()
      loadLogsStats()
    }
    const exportLogs = async () => {
      try {
        const params = { ...filterForm }
        const response = await adminAPI.exportLogs(params)
        if (response && response.data) {
          if (response.data instanceof Blob) {
            const url = window.URL.createObjectURL(response.data)
            const a = document.createElement('a')
            a.href = url
            a.download = `system_logs_${new Date().toISOString().split('T')[0]}.csv`
            document.body.appendChild(a)
            a.click()
            document.body.removeChild(a)
            window.URL.revokeObjectURL(url)
            ElMessage.success('日志导出成功')
            return
          }
        }
        ElMessage.error('导出失败：响应格式不正确')
      } catch (error) {
        if (error.response && error.response.data instanceof Blob) {
          try {
            const text = await error.response.data.text()
            const errorData = JSON.parse(text)
            ElMessage.error(errorData.message || '导出失败')
          } catch (e) {
            ElMessage.error('导出失败')
          }
        } else {
          const errorMsg = error.response?.data?.message || error.message || '导出失败'
          ElMessage.error(errorMsg)
        }
        console.error('导出日志失败:', error)
      }
    }
    const clearLogs = async () => {
      try {
        await confirmClear('系统日志', {
          message: '确定要清理所有日志吗？此操作不可恢复。',
          title: '确认清理日志',
          confirmButtonText: '确认清理'
        })
        const response = await adminAPI.clearLogs()
        if (response && response.data && response.data.success) {
          ElMessage.success(response.data.message || '日志清理成功')
          loadLogs()
          loadLogsStats()
        } else {
          ElMessage.error((response?.data?.message || response?.message) || '清理失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          const errorMsg = error.response?.data?.message || error.message || '清理失败'
          ElMessage.error(errorMsg)
          console.error('清理日志失败:', error)
        }
      }
    }
    const showLogDetails = (log) => {
      selectedLog.value = log
      logDetailsVisible.value = true
    }
    const closeLogDetails = () => {
      logDetailsVisible.value = false
      selectedLog.value = null
    }
    const copyLogDetails = async () => {
      if (!selectedLog.value) return
      try {
        const logText = `
时间: ${formatDate(selectedLog.value.timestamp)}
级别: ${getLogLevelText(selectedLog.value.level)}
模块: ${selectedLog.value.module}
用户: ${selectedLog.value.username || '系统'}
IP地址: ${selectedLog.value.ip_address}
日志内容: ${selectedLog.value.message}
${selectedLog.value.details ? `详细信息: ${selectedLog.value.details}` : ''}
${selectedLog.value.stack_trace ? `堆栈跟踪: ${selectedLog.value.stack_trace}` : ''}
        `.trim()
        await navigator.clipboard.writeText(logText)
        ElMessage.success('日志详情已复制到剪贴板')
      } catch (error) {
        ElMessage.error('复制失败')
      }
    }
    const handleSizeChange = (size) => {
      pagination.size = size
      pagination.page = 1
      loadLogs()
    }
    const handleCurrentChange = (page) => {
      pagination.page = page
      loadLogs()
    }
    const formatDate = (dateString) => {
      if (!dateString) return ''
      const date = new Date(dateString)
      return date.toLocaleString('zh-CN')
    }
    const getLogLevelTagType = (level) => {
      const typeMap = {
        'critical': 'danger',
        'error': 'danger',
        'warning': 'warning',
        'info': 'info',
        'debug': ''
      }
      return typeMap[level] || ''
    }
    const getLogLevelText = (level) => {
      const textMap = {
        'critical': '严重',
        'error': '错误',
        'warning': '警告',
        'info': '信息',
        'debug': '调试'
      }
      return textMap[level] || level
    }
    const getTaskTypeName = (type) => {
      const nameMap = {
        'scheduler': '定时任务调度器',
        'email_queue': '邮件队列',
        'scheduler_email_queue': '邮件队列',
        'auto_backup': '自动备份',
        'scheduler_auto_backup': '自动备份',
        'auto_node_update': '节点更新',
        'scheduler_auto_node_update': '节点更新',
        'node_health_check': '节点健康检查',
        'scheduler_node_health_check': '节点健康检查',
        'expiring_subscriptions': '订阅到期检查',
        'scheduler_expiring_subscriptions': '订阅到期检查',
        'system_error': '系统错误',
        'security_login_success': '用户登录成功',
        'security_admin_login_success': '管理员登录',
        'security_login_attempt': '登录尝试',
        'security_login_failed': '登录失败',
        'security_login_blocked': '登录被阻止',
        'security_ip_blocked': 'IP封禁',
        'security_login_rate_limit': '登录限流',
        'security_register_success': '注册成功',
        'security_register_rate_limit': '注册限流',
        'security_register_ip_blocked': '注册IP封禁',
        'security_verify_code_rate_limit': '验证码限流',
        'security_user_unlock': '解禁用户',
        'security_user_enabled': '启用用户',
        'security_user_disabled': '禁用用户',
        'security_admin_login_as': '管理员代登',
        'security_password_change_failed': '修改密码失败',
        'security_reset_code_failed': '重置密码/验证码失败',
        'security_auth_token_invalid': 'Token无效/过期',
        'security_auth_token_blacklisted': 'Token已失效',
        'security_admin_forbidden': '非管理员访问管理端',
        'security_csrf_validation_failed': 'CSRF验证失败',
        'security_password_reset_requested': '请求密码重置',
        'security_admin_reset_password': '管理员重置密码',
        'security_refresh_token_invalid': '刷新令牌无效',
        'security_verification_code_failed': '验证码校验失败',
        'business_payment_callback_signature_failed': '支付回调签名失败',
        'business_payment_callback_order_not_found': '支付回调订单不存在',
        'business_payment_callback_amount_mismatch': '支付回调金额不一致',
        'business_payment_callback_process_failed': '支付回调处理失败',
        'business_subscription_validation_failed': '订阅校验未通过',
        'business_subscription_pull_not_found': '订阅拉取Token无效',
        'business_subscription_pull_query_failed': '订阅拉取查询失败',
        'business_order_payment_url_failed': '订单支付链接生成失败',
        'business_recharge_payment_url_failed': '充值支付链接生成失败',
        'business_refund_failed': '管理员退款失败',
        'business_refund_process_failed': '退款回退处理失败',
        'business_delete_user_failed': '删除用户失败',
        'business_subscription_convert_failed': '订阅转余额失败',
        'business_email_config_save_failed': '邮件配置保存失败',
        'business_payment_config_save_failed': '支付配置保存失败',
        'business_invite_reward_failed': '邀请奖励发放失败'
      }
      if (nameMap[type]) return nameMap[type]
      if (type && type.startsWith('scheduler_')) {
        const plainTaskType = type.substring('scheduler_'.length)
        return nameMap[plainTaskType] || `定时任务(${plainTaskType})`
      }
      return type
    }
    const truncateText = (text, length) => {
      if (!text) return ''
      return text.length > length ? text.substring(0, length) + '...' : text
    }
    const displayLocation = (loc) => {
      if (!loc) return '-'
      const result = formatLocation(loc)
      return result || loc
    }
    onMounted(() => {
      loadLogs()
      loadLogsStats()
    })
    return {
      isMobile,
      loading,
      logsList,
      logsStats,
      mobileLogFields,
      filterForm,
      pagination,
      logDetailsVisible,
      selectedLog,
      applyFilter,
      debouncedApplyFilter,
      resetFilter,
      filterByLevel,
      exportLogs,
      clearLogs,
      showLogDetails,
      closeLogDetails,
      copyLogDetails,
      handleSizeChange,
      handleCurrentChange,
      formatDate,
      getLogLevelTagType,
      getLogLevelText,
      getTaskTypeName,
      truncateText,
      displayLocation
    }
  }
}
</script>
<style scoped>
.admin-system-logs {
  padding: 20px;
}
.admin-system-logs > :deep(.el-card) {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  box-shadow: none;
}
.logs-filter {
  margin-bottom: 20px;
  padding: 16px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 8px;
}
.logs-filter :deep(.el-select),
.logs-filter :deep(.el-input),
.logs-filter :deep(.el-date-editor) {
  width: 100%;
}
.system-log-filter-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
  gap: 12px;
  align-items: end;
}
.system-log-filter-field,
.system-log-filter-grid :deep(.el-form-item),
.system-log-filter-grid :deep(.el-form-item__content) {
  min-width: 0;
}
.system-log-filter-grid :deep(.el-form-item) {
  margin-bottom: 0;
}
.system-log-filter-grid .keyword-field {
  grid-column: span 2;
}
@media (max-width: 1180px) {
  .system-log-filter-grid .keyword-field {
    grid-column: auto;
  }
}
.full-width-control,
.logs-data-table {
  width: 100%;
}
.filter-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
  align-items: center;
  min-width: 0;
  align-self: end;
}
.filter-actions .el-button {
  margin-left: 0;
}
.logs-stats {
  margin-bottom: 20px;
}
.logs-stats :deep(.el-card__body) {
  padding: 0;
}
.stat-card {
  text-align: center;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  box-shadow: none;
}
.stat-card.clickable {
  cursor: pointer;
  transition: border-color 0.2s ease, background-color 0.2s ease;
}
.stat-card.clickable:hover {
  background: #fafcff;
  border-color: #c6e2ff;
}
.stat-content {
  padding: 18px;
}
.stat-number {
  font-size: 26px;
  font-weight: 700;
  color: #303133;
  line-height: 1.2;
}
.stat-number.error {
  color: #f56c6c;
}
.stat-number.warning {
  color: #e6a23c;
}
.stat-number.info {
  color: #409eff;
}
.stat-label {
  font-size: 13px;
  color: #909399;
  margin-top: 8px;
}
.logs-table {
  margin-top: 20px;
}
.log-message {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.message-text {
  flex: 1;
  margin-right: 10px;
}
.user-agent-text {
  display: inline-block;
  max-width: 200px;
  overflow: clip;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.location-text {
  margin-top: 4px;
  font-size: 12px;
}
.location-details {
  margin-top: 8px;
  padding: 8px;
  background: #f5f7fa;
  border-radius: 4px;
  font-size: 12px;
  color: #606266;
}
.location-details div {
  margin: 4px 0;
}
.log-details {
  max-height: 600px;
  overflow-y: auto;
}
.log-message-section,
.log-details-section,
.log-stack-section,
.log-context-section {
  margin-top: 20px;
}
.log-message-section h4,
.log-details-section h4,
.log-stack-section h4,
.log-context-section h4 {
  margin: 0 0 10px 0;
  color: #333;
  font-size: 1rem;
}
.log-message-content {
  padding: 10px;
  background: #f8f9fa;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-word;
}
.log-details-content,
.log-stack-content,
.log-context-content {
  padding: 10px;
  background: #f8f9fa;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 200px;
  overflow-y: auto;
  font-family: monospace;
  font-size: 12px;
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
  .admin-system-logs {
    padding: 10px;
  }
  .card-header {
    flex-direction: column;
    gap: 10px;
    align-items: flex-start;
    :is(h2) {
      font-size: 18px;
    }
    :is(p) {
      font-size: 13px;
    }
  }
  .logs-filter {
    padding: 15px;
    margin-bottom: 16px;
  }
  .mobile-filter-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
    margin-top: 16px;
  }
  .mobile-action-btn {
    width: 100%;
    min-width: 0;
    min-height: 44px;
    font-size: 14px;
    margin: 0;
    border-radius: 6px;
    touch-action: manipulation;
  }
  .logs-stats {
    margin-bottom: 16px;
    .stat-card {
      margin-bottom: 12px;
    }
    .stat-content {
      padding: 16px;
    }
    .stat-number {
      font-size: 1.5rem;
    }
    .stat-label {
      font-size: 13px;
      margin-top: 8px;
    }
  }
  .pagination-wrapper {
    margin-top: 16px;
    :deep(.el-pagination) {
      flex-wrap: wrap;
      .el-pagination__sizes,
      .el-pagination__jump {
        display: none;
      }
    }
  }
}
.mobile-log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
}
.mobile-log-time {
  color: #909399;
  font-size: 12px;
  line-height: 1.4;
  text-align: right;
}
</style>
