<template>
  <div class="system-logs-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <h2>系统日志</h2>
          <p>查看和管理系统运行日志</p>
        </div>
      </template>
      <div class="logs-filter">
        <div class="desktop-only">
          <el-row :gutter="20">
            <el-col :span="6">
              <el-form-item label="日志类型">
                <el-select v-model="filterForm.log_type" placeholder="选择日志类型" clearable>
                  <el-option label="全部" value="" />
                  <el-option label="错误" value="error" />
                  <el-option label="警告" value="warning" />
                  <el-option label="信息" value="info" />
                  <el-option label="调试" value="debug" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="日志级别">
                <el-select v-model="filterForm.log_level" placeholder="选择日志级别" clearable>
                  <el-option label="全部" value="" />
                  <el-option label="严重" value="critical" />
                  <el-option label="错误" value="error" />
                  <el-option label="警告" value="warning" />
                  <el-option label="信息" value="info" />
                  <el-option label="调试" value="debug" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="开始时间">
                <el-date-picker
                  v-model="filterForm.start_time"
                  type="datetime"
                  placeholder="选择开始时间"
                  format="YYYY-MM-DD HH:mm:ss"
                  value-format="YYYY-MM-DD HH:mm:ss"
                />
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="结束时间">
                <el-date-picker
                  v-model="filterForm.end_time"
                  type="datetime"
                  placeholder="选择结束时间"
                  format="YYYY-MM-DD HH:mm:ss"
                  value-format="YYYY-MM-DD HH:mm:ss"
                />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="20">
            <el-col :span="8">
              <el-form-item label="关键词搜索">
                <el-input
                  v-model="filterForm.keyword"
                  placeholder="搜索日志内容、模块、用户等"
                  clearable
                />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="任务类型">
                <el-select v-model="filterForm.task_type" placeholder="选择任务类型" clearable>
                  <el-option label="全部" value="" />
                  <el-option label="定时任务调度器" value="scheduler" />
                  <el-option label="邮件队列" value="email_queue" />
                  <el-option label="自动备份" value="auto_backup" />
                  <el-option label="节点更新" value="auto_node_update" />
                  <el-option label="节点健康检查" value="node_health_check" />
                  <el-option label="订阅到期检查" value="expiring_subscriptions" />
                  <el-option label="账户删除" value="account_deletion" />
                  <el-option label="系统错误" value="system_error" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>
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
              <el-select v-model="filterForm.log_type" placeholder="选择日志类型" clearable style="width: 100%">
                <el-option label="全部" value="" />
                <el-option label="错误" value="error" />
                <el-option label="警告" value="warning" />
                <el-option label="信息" value="info" />
                <el-option label="调试" value="debug" />
              </el-select>
            </el-form-item>
            <el-form-item label="日志级别">
              <el-select v-model="filterForm.log_level" placeholder="选择日志级别" clearable style="width: 100%">
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
                style="width: 100%"
              />
            </el-form-item>
            <el-form-item label="结束时间">
              <el-date-picker
                v-model="filterForm.end_time"
                type="datetime"
                placeholder="选择结束时间"
                format="YYYY-MM-DD HH:mm:ss"
                value-format="YYYY-MM-DD HH:mm:ss"
                style="width: 100%"
              />
            </el-form-item>
            <el-form-item label="关键词搜索">
              <el-input
                v-model="filterForm.keyword"
                placeholder="搜索日志内容、模块、用户等"
                clearable
              />
            </el-form-item>
            <el-form-item label="模块">
              <el-select v-model="filterForm.module" placeholder="选择模块" clearable style="width: 100%">
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
        <div class="desktop-only">
          <el-table
            :data="logsList"
            v-loading="loading"
            style="width: 100%"
            stripe
            border
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
        </div>
        <div class="mobile-only">
          <div v-loading="loading" class="mobile-logs-list">
            <div 
              v-for="log in logsList" 
              :key="log.id || log.timestamp"
              class="mobile-log-card"
              @click="showLogDetails(log)"
            >
              <div class="log-card-header">
                <el-tag :type="getLogLevelTagType(log.level)" size="small">
                  {{ getLogLevelText(log.level) }}
                </el-tag>
                <span class="log-time">{{ formatDate(log.timestamp) }}</span>
              </div>
              <div class="log-card-body">
                <div class="log-card-row">
                  <span class="log-label">模块：</span>
                  <span class="log-value">{{ log.module || '-' }}</span>
                </div>
                <div class="log-card-row">
                  <span class="log-label">内容：</span>
                  <span class="log-value log-message-text">{{ truncateText(log.message, 50) }}</span>
                </div>
                <div class="log-card-row" v-if="log.failure_reason">
                  <span class="log-label">失败原因：</span>
                  <el-tag type="warning" size="small">{{ log.failure_reason }}</el-tag>
                </div>
                <div class="log-card-row" v-if="log.username">
                  <span class="log-label">用户：</span>
                  <span class="log-value">{{ log.username }}</span>
                </div>
                <div class="log-card-row" v-if="log.ip_address">
                  <span class="log-label">IP：</span>
                  <span class="log-value">{{ log.ip_address }}</span>
                </div>
                <div class="log-card-row" v-if="log.location">
                  <span class="log-label">地理位置：</span>
                  <el-tag size="small" type="info">{{ displayLocation(log.location) }}</el-tag>
                </div>
              </div>
              <div class="log-card-footer">
                <el-button size="small" type="primary" @click.stop="showLogDetails(log)">
                  查看详情
                </el-button>
              </div>
            </div>
            <el-empty v-if="logsList.length === 0 && !loading" description="暂无日志数据" />
          </div>
        </div>
        <div class="pagination-wrapper">
          <el-pagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.size"
            :page-sizes="[20, 50, 100, 200]"
            :total="pagination.total"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handleSizeChange"
            @current-change="handleCurrentChange"
          />
        </div>
      </div>
    </el-card>
    <el-drawer
      v-model="logDetailsVisible"
      title="日志详情"
      :size="isMobile ? '92%' : '600px'"
      direction="rtl"
      :before-close="closeLogDetails"
      :lock-scroll="false"
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
        <div class="dialog-footer-buttons">
          <el-button @click="closeLogDetails" :class="{ 'mobile-action-btn': isMobile }" :style="isMobile ? { width: '100%', marginBottom: '10px' } : {}">关闭</el-button>
          <el-button type="primary" @click="copyLogDetails" :class="{ 'mobile-action-btn': isMobile }" :style="isMobile ? { width: '100%' } : {}">
            复制详情
          </el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>
<script>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Refresh, Download, Delete } from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
import { formatLocation } from '@/utils/date'
export default {
  name: 'AdminSystemLogs',
  components: {
    Search, Refresh, Download, Delete
  },
  setup() {
    const isMobile = ref(window.innerWidth <= 768)
    const loading = ref(false)
    const handleResize = () => {
      isMobile.value = window.innerWidth <= 768
    }
    const logsList = ref([])
    const logsStats = ref({})
    const logDetailsVisible = ref(false)
    const selectedLog = ref(null)
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
      size: 20,
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
        if (response && response.data && response.data.success) {
          logsList.value = response.data.data.logs || []
          pagination.total = response.data.data.total || 0
        } else {
          ElMessage.error((response?.data?.message || response?.message) || '加载日志失败')
        }
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
        await ElMessageBox.confirm(
          '确定要清理所有日志吗？此操作不可恢复！',
          '确认清理',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
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
        'account_deletion': '账户删除',
        'scheduler_account_deletion': '账户删除',
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
      window.addEventListener('resize', handleResize)
    })
    onUnmounted(() => {
      window.removeEventListener('resize', handleResize)
    })
    return {
      isMobile,
      loading,
      logsList,
      logsStats,
      filterForm,
      pagination,
      logDetailsVisible,
      selectedLog,
      applyFilter,
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
.system-logs-container {
  padding: 20px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.card-header h2 {
  margin: 0;
  color: #333;
  font-size: 1.5rem;
}
.card-header :is(p) {
  margin: 0;
  color: #666;
  font-size: 0.9rem;
}
.logs-filter {
  margin-bottom: 20px;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
}
.filter-actions {
  margin-top: 20px;
  text-align: center;
}
.logs-stats {
  margin-bottom: 20px;
}
.stat-card {
  text-align: center;
}
.stat-card.clickable {
  cursor: pointer;
  transition: all 0.3s ease;
}
.stat-card.clickable:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}
.stat-content {
  padding: 20px;
}
.stat-number {
  font-size: 2rem;
  font-weight: bold;
  color: #333;
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
  font-size: 0.9rem;
  color: #666;
  margin-top: 10px;
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
.pagination-wrapper {
  text-align: right;
  margin-top: 20px;
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
  .system-logs-container {
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
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin-top: 16px;
  }
  .mobile-action-btn {
    width: 100%;
    min-height: 44px;
    font-size: 16px;
    margin: 0 !important;
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
  .mobile-logs-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .mobile-log-card {
    background: #ffffff;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    padding: 14px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    transition: all 0.3s ease;
    &:active {
      transform: scale(0.98);
      box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
    }
  }
  .log-card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    padding-bottom: 10px;
    border-bottom: 1px solid #f0f0f0;
    .log-time {
      font-size: 12px;
      color: #909399;
    }
  }
  .log-card-body {
    margin-bottom: 12px;
  }
  .log-card-row {
    display: flex;
    margin-bottom: 8px;
    font-size: 14px;
    &:last-child {
      margin-bottom: 0;
    }
    .log-label {
      font-weight: 600;
      color: #606266;
      min-width: 60px;
      flex-shrink: 0;
    }
    .log-value {
      color: #303133;
      flex: 1;
      word-break: break-word;
      &.log-message-text {
        line-height: 1.5;
      }
    }
  }
  .log-card-footer {
    padding-top: 10px;
    border-top: 1px solid #f0f0f0;
    .el-button {
      width: 100%;
      min-height: 40px;
      font-size: 14px;
    }
  }
  .pagination-wrapper {
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
