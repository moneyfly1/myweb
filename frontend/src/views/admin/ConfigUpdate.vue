<template>
  <div class="list-container config-update-page">
    <!-- 1. 操作控制卡片 -->
    <el-card class="list-card control-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span class="header-title">系统操作</span>
          <div class="header-status desktop-only">
             <el-tag :type="status.is_running ? 'success' : 'info'" effect="dark">
                {{ status.is_running ? '正在更新' : '等待指令' }}
             </el-tag>
          </div>
        </div>
      </template>
      
      <div class="control-panel">
        <!-- 移动端状态展示 -->
        <div class="mobile-status-bar mobile-only" v-if="isMobile">
           <span class="label">当前状态:</span>
           <el-tag :type="status.is_running ? 'success' : 'info'" size="small" effect="dark">
              {{ status.is_running ? '运行中' : '已停止' }}
           </el-tag>
        </div>

        <div class="action-grid">
          <el-button 
            type="primary" 
            :loading="loading.start"
            @click="startUpdate"
            :disabled="status.is_running"
            class="grid-btn start-btn"
            size="default"
          >
            <div class="btn-content">
              <el-icon :size="16"><VideoPlay /></el-icon>
              <span>开始更新</span>
            </div>
          </el-button>
          
          <el-button 
            type="warning" 
            :loading="loading.stop"
            @click="stopUpdate"
            :disabled="!status.is_running"
            class="grid-btn stop-btn"
            size="default"
          >
            <div class="btn-content">
              <el-icon :size="16"><VideoPause /></el-icon>
              <span>停止更新</span>
            </div>
          </el-button>

          <el-button 
            type="info" 
            :loading="loading.test"
            @click="testUpdate"
            :disabled="status.is_running"
            class="grid-btn"
            plain
            size="default"
          >
            <div class="btn-content">
              <el-icon :size="16"><View /></el-icon>
              <span>测试运行</span>
            </div>
          </el-button>
          
          <el-button 
            type="success" 
            :loading="loading.refresh"
            @click="refreshStatus"
            class="grid-btn"
            plain
            size="default"
          >
            <div class="btn-content">
              <el-icon :size="16"><Refresh /></el-icon>
              <span>刷新状态</span>
            </div>
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- 2. 配置设置卡片 -->
    <el-card class="list-card config-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span class="header-title">参数配置</span>
          <div class="header-actions">
            <el-button type="primary" @click="saveConfig" :loading="loading.save">
              <el-icon><Check /></el-icon>
              <span class="desktop-only">保存配置</span>
              <span class="mobile-only">保存</span>
            </el-button>
          </div>
        </div>
      </template>

      <el-form :model="config" label-position="top" class="config-form">
        
        <!-- 节点源 URL 配置 -->
        <div class="form-section">
          <div class="section-title">
            <el-icon><Connection /></el-icon> 节点源列表
          </div>
          <div class="url-list">
            <div v-for="(url, index) in config.urls" :key="index" class="list-item-wrapper">
              <div class="input-with-action">
                <el-input 
                  v-model="config.urls[index]" 
                  placeholder="请输入订阅/节点源 URL"
                  class="styled-input"
                >
                  <template #prefix v-if="!isMobile">
                    <span class="index-badge">{{ index + 1 }}</span>
                  </template>
                </el-input>
                <el-button 
                  type="danger" 
                  plain 
                  @click="removeUrl(index)"
                  :disabled="config.urls.length <= 1"
                  class="action-btn-side"
                >
                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>
            </div>
            <el-button type="primary" plain @click="addUrl" class="add-item-btn">
              <el-icon><Plus /></el-icon> 添加订阅源
            </el-button>
          </div>
        </div>

        <el-divider border-style="dashed" />

        <!-- 定时任务配置：自动更新开关 + 更新间隔（统一高度） -->
        <div class="form-section schedule-section">
          <div class="schedule-row">
            <div class="schedule-item switch-item">
              <span class="schedule-label">自动更新</span>
              <el-switch v-model="config.enable_schedule" @change="handleScheduleChange" size="default" class="schedule-switch" />
            </div>
            <div class="schedule-divider" />
            <div class="schedule-item interval-item">
              <span class="schedule-label">更新间隔</span>
              <div class="interval-inputs">
                <el-select v-model="intervalUnit" class="unit-select" placeholder="单位" @change="updateInterval">
                  <el-option label="分钟" value="minute" />
                  <el-option label="小时" value="hour" />
                  <el-option label="天" value="day" />
                </el-select>
                <el-input-number 
                  v-model="intervalValue" 
                  :min="1" 
                  :max="intervalUnit === 'minute' ? 1440 : intervalUnit === 'hour' ? 24 : 30"
                  class="value-input"
                  controls-position="right"
                  @change="updateInterval"
                />
              </div>
            </div>
          </div>
          <div class="desc-text" v-if="config.enable_schedule">
            每 {{ formatInterval(config.update_interval) }} 执行一次
          </div>
        </div>

        <el-divider border-style="dashed" />

        <!-- 过滤关键词配置 -->
        <div class="form-section">
          <div class="section-title">
            <el-icon><Filter /></el-icon> 过滤关键词
            <el-tooltip content="包含这些关键词的节点将被自动丢弃" placement="top">
               <el-icon class="info-icon"><Warning /></el-icon>
            </el-tooltip>
          </div>
          
          <el-alert
            v-if="!config.filter_keywords.length"
            title="暂无过滤规则，所有节点都将被保留"
            type="info"
            :closable="false"
            show-icon
            class="mini-alert"
          />

          <div class="keyword-list">
            <div v-for="(keyword, index) in config.filter_keywords" :key="index" class="list-item-wrapper">
              <div class="input-with-action">
                <el-input 
                  v-model="config.filter_keywords[index]" 
                  placeholder="输入需过滤的关键词 (如: 轮子, 到期)"
                  class="styled-input"
                />
                <el-button 
                  type="danger" 
                  plain 
                  @click="removeKeyword(index)"
                  class="action-btn-side"
                >
                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>
            </div>
             <el-button type="primary" plain @click="addKeyword" class="add-item-btn">
              <el-icon><Plus /></el-icon> 添加关键词
            </el-button>
          </div>
        </div>

      </el-form>
    </el-card>

    <!-- 3. 日志卡片 -->
    <el-card class="list-card log-card" shadow="never">
      <template #header>
        <div class="card-header">
          <div class="header-left">
             <span class="header-title">运行日志</span>
             <el-tag v-if="isLogPolling" type="success" size="small" class="status-badge pulse">实时</el-tag>
          </div>
          <div class="header-actions compact">
            <el-button size="small" @click="refreshLogs" :loading="loading.refresh" circle>
              <el-icon><Refresh /></el-icon>
            </el-button>
            <el-button size="small" type="danger" plain @click="clearLogs" circle>
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
        </div>
      </template>
      
      <div class="log-viewer">
        <div v-if="logs.length === 0" class="empty-logs">
           <el-icon><Document /></el-icon>
           <span>暂无日志记录</span>
        </div>
        <div 
          v-else
          v-for="(log, index) in logs" 
          :key="index" 
          class="log-line"
          :class="(log && log.level) || 'info'"
        >
          <span class="log-ts">[{{ formatLogTime(log && (log.timestamp || log.time)) }}]</span>
          <span class="log-lvl" :class="(log && log.level) || 'info'">{{ ((log && log.level) || 'info').toUpperCase() }}</span>
          <span class="log-msg">{{ (log && log.message) || '' }}</span>
        </div>
      </div>
    </el-card>

  </div>
</template>

<script>
import { ref, reactive, onMounted, onUnmounted, onBeforeUnmount, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  VideoPlay, VideoPause, View, Refresh, Check, Delete, Plus, 
  Connection, Filter, Warning, Document
} from '@element-plus/icons-vue'
import { configUpdateAPI } from '@/utils/api'

export default {
  name: 'ConfigUpdate',
  components: {
    VideoPlay, VideoPause, View, Refresh, Check, Delete, Plus,
    Connection, Filter, Warning, Document
  },
  setup() {
    // 状态定义
    const status = ref({
      is_running: false,
      scheduled_enabled: false,
      last_update: null
    })
    
    const config = reactive({
      urls: [''],
      update_interval: 3600,
      enable_schedule: false,
      filter_keywords: []
    })
    
    const intervalUnit = ref('hour')
    const intervalValue = ref(1)
    const logs = ref([])
    const isLogPolling = ref(false)
    const isMobile = ref(false)
    
    const loading = reactive({
      start: false,
      stop: false,
      test: false,
      refresh: false,
      save: false,
    })

    let statusPollingInterval = null
    let refreshInterval = null
    let eventSource = null // SSE 连接

    // 辅助函数
    const checkMobile = () => {
      isMobile.value = window.innerWidth <= 768
    }

    const formatLogTime = (timeStr) => {
      if (!timeStr) return ''
      const date = new Date(timeStr)
      return `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}:${date.getSeconds().toString().padStart(2, '0')}`
    }

    const formatInterval = (seconds) => {
      if (!seconds) return '未设置'
      if (seconds < 60) return `${seconds}秒`
      if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟`
      if (seconds < 86400) return `${Math.floor(seconds / 3600)}小时`
      return `${Math.floor(seconds / 86400)}天`
    }

    // 逻辑控制
    const initIntervalSelector = () => {
      const seconds = config.update_interval || 3600
      if (seconds < 3600) {
        intervalUnit.value = 'minute'
        intervalValue.value = Math.floor(seconds / 60)
      } else if (seconds < 86400) {
        intervalUnit.value = 'hour'
        intervalValue.value = Math.floor(seconds / 3600)
      } else {
        intervalUnit.value = 'day'
        intervalValue.value = Math.floor(seconds / 86400)
      }
    }

    const updateInterval = () => {
      let seconds = 0
      if (intervalUnit.value === 'minute') seconds = (intervalValue.value || 1) * 60
      else if (intervalUnit.value === 'hour') seconds = (intervalValue.value || 1) * 3600
      else if (intervalUnit.value === 'day') seconds = (intervalValue.value || 1) * 86400
      config.update_interval = seconds
    }

    // API 调用（防御性处理，避免 response.data.data 为 undefined 导致页面报错空白）
    const getStatus = async () => {
      try {
        const response = await configUpdateAPI.getStatus()
        const data = response?.data?.data
        if (response?.data?.success && data && typeof data === 'object') {
          status.value = { ...status.value, ...data }
        }
      } catch (error) {
        console.error('状态获取失败', error)
      }
    }

    const getConfig = async () => {
      try {
        const response = await configUpdateAPI.getConfig()
        const data = response?.data?.data
        if (response?.data?.success && data && typeof data === 'object') {
          // 后端返回 schedule_interval，前端使用 update_interval
          const interval = data.update_interval ?? data.schedule_interval ?? 3600
          config.urls = Array.isArray(data.urls) ? (data.urls.length ? data.urls : ['']) : ['']
          config.filter_keywords = Array.isArray(data.filter_keywords) ? data.filter_keywords : []
          config.enable_schedule = !!data.enable_schedule
          config.update_interval = typeof interval === 'number' ? interval : 3600
          initIntervalSelector()
        }
      } catch (error) {
        ElMessage.error('获取配置失败')
      }
    }

    const getLogs = async () => {
      try {
        const response = await configUpdateAPI.getLogs()
        const data = response?.data?.data
        if (response?.data?.success && Array.isArray(data)) {
          logs.value = data
          // 自动滚动到底部
          nextTick(() => {
            const logContainer = document.querySelector('.log-container')
            if (logContainer) {
              logContainer.scrollTop = logContainer.scrollHeight
            }
          })
        }
      } catch (error) {
        console.error('日志获取失败', error)
      }
    }

    const refreshLogs = () => getLogs()

    const saveConfig = async () => {
      loading.save = true
      try {
        updateInterval()
        const urls = (config.urls || []).filter(url => url && String(url).trim())
        const filter_keywords = (config.filter_keywords || []).filter(keyword => keyword && String(keyword).trim())
        if (!urls.length) {
          ElMessage.warning('至少需要配置一个节点源URL')
          loading.save = false
          return
        }
        // 后端使用 schedule_interval 键名
        const configToSave = {
          urls,
          filter_keywords,
          enable_schedule: config.enable_schedule,
          schedule_interval: config.update_interval
        }

        const response = await configUpdateAPI.updateConfig(configToSave)
        if (response.data.success) {
          ElMessage.success('配置已保存')
          if (config.enable_schedule) await getStatus()
        } else {
          ElMessage.error(response.data.message || '保存失败')
        }
      } catch (error) {
        ElMessage.error('保存失败: ' + (error.response?.data?.message || error.message))
      } finally {
        loading.save = false
      }
    }

    // 轮询控制
    const startStatusPolling = () => {
      if (statusPollingInterval) clearInterval(statusPollingInterval)
      statusPollingInterval = setInterval(async () => {
        await getStatus()
        if (!status.value.is_running) {
          stopAllPolling()
        }
      }, 1000)
    }

    const stopAllPolling = () => {
      if (statusPollingInterval) { clearInterval(statusPollingInterval); statusPollingInterval = null; }
      if (refreshInterval) { clearInterval(refreshInterval); refreshInterval = null; }
      disconnectSSE()
      isLogPolling.value = false
    }

    // SSE 连接管理
    const connectSSE = () => {
      // 如果已有连接，先断开
      disconnectSSE()

      isLogPolling.value = true

      // 建立 SSE 连接
      eventSource = new EventSource('/api/admin/config-update/logs/stream')

      eventSource.onmessage = (event) => {
        try {
          const log = JSON.parse(event.data)
          logs.value.push(log)

          // 限制日志数量
          if (logs.value.length > 500) {
            logs.value = logs.value.slice(-500)
          }

          // 自动滚动到底部
          nextTick(() => {
            const viewer = document.querySelector('.log-viewer')
            if (viewer) {
              viewer.scrollTop = viewer.scrollHeight
            }
          })
        } catch (e) {
          console.error('解析日志失败:', e)
        }
      }

      eventSource.onerror = (error) => {
        console.error('SSE 连接错误:', error)
        disconnectSSE()

        // 如果任务还在运行，3秒后重连
        if (status.value.is_running) {
          setTimeout(() => {
            if (status.value.is_running) {
              connectSSE()
            }
          }, 3000)
        }
      }
    }

    const disconnectSSE = () => {
      if (eventSource) {
        eventSource.close()
        eventSource = null
      }
      isLogPolling.value = false
    }

    const startLogPolling = () => {
      // 使用 SSE 替代轮询
      connectSSE()
    }

    // 按钮操作
    const startUpdate = async () => {
      loading.start = true
      try {
        const response = await configUpdateAPI.startUpdate()
        if (response.data.success) {
          ElMessage.success('更新任务已启动')
          startStatusPolling()
          startLogPolling()
        } else {
          ElMessage.error(response.data.message || '启动失败')
        }
      } catch (error) {
        ElMessage.error('启动失败: ' + (error.message))
      } finally {
        loading.start = false
      }
    }

    const stopUpdate = async () => {
      loading.stop = true
      try {
        const response = await configUpdateAPI.stopUpdate()
        if (response.data.success) {
          ElMessage.success('已发送停止指令')
          stopAllPolling()
          await getStatus()
        }
      } catch (error) {
        ElMessage.error('停止失败')
      } finally {
        loading.stop = false
      }
    }

    const testUpdate = async () => {
      loading.test = true
      try {
        const response = await configUpdateAPI.testUpdate()
        if (response.data.success) {
          ElMessage.success('测试运行已启动')
          startStatusPolling()
          startLogPolling()
        }
      } catch (error) {
        ElMessage.error('启动测试失败')
      } finally {
        loading.test = false
      }
    }

    const refreshStatus = async () => {
      loading.refresh = true
      try {
        await Promise.all([getStatus(), getLogs()])
        ElMessage.success('状态已刷新')
      } finally {
        loading.refresh = false
      }
    }

    const clearLogs = async () => {
      try {
        await ElMessageBox.confirm('确定清空日志吗？', '提示', { type: 'warning' })
        const response = await configUpdateAPI.clearLogs()
        if (response.data.success) {
          logs.value = []
          ElMessage.success('日志已清空')
        }
      } catch (e) {
        ElMessage.error('清空日志失败: ' + (e.response?.data?.message || e.message))
      }
    }

    // 列表操作
    const addUrl = () => {
      if (!config.urls) config.urls = []
      config.urls.push('')
    }
    const removeUrl = (index) => {
      if (config.urls.length > 1) config.urls.splice(index, 1)
      else ElMessage.warning('至少保留一个URL')
    }
    const addKeyword = () => {
      if (!config.filter_keywords) config.filter_keywords = []
      config.filter_keywords.push('')
    }
    const removeKeyword = (index) => config.filter_keywords.splice(index, 1)
    const handleScheduleChange = (val) => {
      if (val && (!config.update_interval || config.update_interval < 60)) {
        ElMessage.warning('间隔不能小于1分钟')
        config.enable_schedule = false
      }
    }

    // 生命周期
    onMounted(async () => {
      checkMobile()
      window.addEventListener('resize', checkMobile)
      await Promise.all([getStatus(), getConfig(), getLogs()])
      if (status.value.is_running) {
        startStatusPolling()
        startLogPolling()
      }
    })

    onUnmounted(() => {
      stopAllPolling()
      disconnectSSE()
    })
    onBeforeUnmount(() => window.removeEventListener('resize', checkMobile))


    return {
      status, config, logs, loading, isLogPolling, isMobile,
      intervalUnit, intervalValue,
      startUpdate, stopUpdate, testUpdate, refreshStatus, saveConfig,
      refreshLogs, clearLogs, addUrl, removeUrl, addKeyword, removeKeyword,
      updateInterval, formatInterval, handleScheduleChange, formatLogTime
    }
  }
}
</script>

<style scoped lang="scss">
// 引入 Users.vue 的设计变量（模拟）
$mobile-break: 768px;
$bg-color: #f5f7fa;
$card-border-radius: 8px;
$primary-color: #409eff;

.list-container {
  padding: 20px;
  max-width: 100%;
  box-sizing: border-box;
  
  @media (max-width: $mobile-break) {
    padding: 12px;
    background-color: $bg-color; // 保持移动端背景一致
    min-height: 100vh;
  }
}

.list-card {
  margin-bottom: 20px;
  border-radius: $card-border-radius;
  overflow: visible; // 允许 dropdown 等溢出
  
  // 覆盖 Element 卡片默认样式，使其更紧凑
  :deep(.el-card__header) {
    padding: 16px 20px;
    border-bottom: 1px solid #f0f2f5;
    background: #fff;
    
    @media (max-width: $mobile-break) {
      padding: 12px 16px;
    }
  }
  
  :deep(.el-card__body) {
    padding: 20px;
    @media (max-width: $mobile-break) {
      padding: 16px;
    }
  }
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  
  .header-title {
    font-size: 16px;
    font-weight: 600;
    color: #1f2f3d;
    position: relative;
    padding-left: 12px;
    
    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 50%;
      transform: translateY(-50%);
      width: 4px;
      height: 16px;
      background-color: $primary-color;
      border-radius: 2px;
    }
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  
  .header-actions {
    display: flex;
    gap: 10px;
    
    &.compact {
      gap: 6px;
    }
  }
}

// 1. 操作控制面板样式
.control-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;

  .mobile-status-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: #f0f9eb;
    padding: 8px 12px;
    border-radius: 6px;
    border: 1px solid #e1f3d8;
    .label {
      font-size: 14px;
      color: #606266;
    }
  }
  
  .action-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 10px;
    
    @media (max-width: $mobile-break) {
      grid-template-columns: repeat(2, 1fr);
      gap: 8px;
    }

    .grid-btn {
      height: auto;
      padding: 10px 8px;
      width: 100%;
      border-radius: 6px;
      transition: all 0.2s;
      
      .btn-content {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 4px;
        
        span {
          font-size: 12px;
          font-weight: 500;
        }
      }
      
      &:hover {
        transform: translateY(-1px);
        box-shadow: 0 2px 8px rgba(0,0,0,0.08);
      }
      
      @media (max-width: $mobile-break) {
        padding: 8px 6px;
        margin: 0;
        
        .btn-content span { font-size: 11px; }
      }
    }
  }
}

// 2. 配置表单样式
.config-form {
  .form-section {
    margin-bottom: 24px;
    
    &:last-child {
      margin-bottom: 0;
    }

    .section-title {
      font-size: 14px;
      font-weight: 600;
      color: #606266;
      margin-bottom: 12px;
      display: flex;
      align-items: center;
      gap: 6px;
      
      .info-icon {
        color: #e6a23c;
        cursor: help;
      }
    }
  }

  // 只保留外层输入框，内部输入无边框
  .list-item-wrapper {
    margin-bottom: 12px;
    border: 1px solid #dcdfe6;
    border-radius: 6px;
    overflow: hidden;
    transition: border-color 0.2s;
    
    &:focus-within { border-color: $primary-color; }
    
    .input-with-action {
      display: flex;
      gap: 0;
      align-items: stretch;
      
      .styled-input {
        flex: 1;
        min-width: 0;
        
        :deep(.el-input__wrapper),
        :deep(.el-input__inner) {
          box-shadow: none !important;
          border: none !important;
          border-radius: 0 !important;
          background: transparent !important;
        }
        :deep(.el-input__wrapper) {
          padding: 8px 12px;
          &.is-focus { box-shadow: none !important; }
        }
        
        .index-badge {
          color: #909399;
          font-size: 12px;
          margin-right: 4px;
          font-weight: 600;
        }
      }
      
      .action-btn-side {
        width: 44px;
        flex-shrink: 0;
        border-radius: 0;
        border-left: 1px solid #dcdfe6;
      }
    }

    @media (max-width: $mobile-break) {
      background: #fff;
    }
  }
  
  .add-item-btn {
    width: 100%;
    border-style: dashed;
    margin-top: 4px;
    height: 40px;
  }
  
  // 定时任务：自动更新 + 更新间隔（三块统一尺寸）
  .schedule-section {
    .schedule-row {
      display: flex;
      align-items: stretch;
      gap: 0;
      background: #fff;
      border: 1px solid #dcdfe6;
      border-radius: 6px;
      overflow: hidden;
    }
    
    .schedule-item {
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 0 14px;
      min-height: 40px;
      
      .schedule-label {
        font-size: 14px;
        color: #606266;
        font-weight: 500;
        white-space: nowrap;
      }
    }
    
    .switch-item {
      flex: 0 0 auto;
      border-right: 1px solid #dcdfe6;
    }
    
    .schedule-divider {
      width: 1px;
      background: #dcdfe6;
      flex-shrink: 0;
    }
    
    .interval-item {
      flex: 1;
      min-width: 0;
      padding-right: 0;
      border-right: none;
      
      .schedule-label {
        flex-shrink: 0;
      }
    }
    
    .interval-inputs {
      flex: 1;
      display: flex;
      min-width: 0;
      
      .unit-select,
      .value-input {
        flex: 1;
        min-width: 0;
      }
      
      .unit-select {
        :deep(.el-input),
        :deep(.el-input__wrapper) {
          height: 36px !important;
          min-height: 36px !important;
        }
        :deep(.el-input__wrapper) {
          border: none !important;
          border-radius: 0 !important;
          border-right: 1px solid #ebeef5;
          box-shadow: none !important;
          background: transparent !important;
        }
        :deep(.el-input__inner) {
          border: none !important;
          box-shadow: none !important;
        }
      }
      
      .value-input {
        :deep(.el-input-number),
        :deep(.el-input),
        :deep(.el-input__wrapper) {
          height: 36px !important;
          min-height: 36px !important;
        }
        :deep(.el-input__wrapper) {
          border: none !important;
          border-radius: 0 !important;
          box-shadow: none !important;
          background: transparent !important;
        }
        :deep(.el-input__inner) {
          height: 34px;
          line-height: 34px;
          border: none !important;
          box-shadow: none !important;
        }
        /* 去掉增减按钮的弧形圆角，内镶弧框去除 */
        :deep(.el-input-number__decrease),
        :deep(.el-input-number__increase) {
          border-radius: 0 !important;
        }
      }
    }
    
    .desc-text {
      font-size: 12px;
      color: #909399;
      margin-top: 8px;
      padding-left: 2px;
    }
    
    @media (max-width: $mobile-break) {
      .schedule-row {
        flex-direction: column;
        border-radius: 6px;
      }
      
      .schedule-item {
        border-right: none;
        border-bottom: 1px solid #dcdfe6;
        min-height: 44px;
        padding: 0 14px;
        
        &:last-child { border-bottom: none; }
      }
      
      .switch-item {
        justify-content: space-between;
      }
      
      .schedule-divider { display: none; }
      
      .interval-item {
        flex-direction: column;
        align-items: stretch;
        gap: 8px;
        padding: 12px 14px;
        
        .schedule-label { margin-bottom: 0; }
      }
      
      .interval-inputs {
        .unit-select,
        .value-input {
          :deep(.el-input__wrapper) {
            height: 40px !important;
            min-height: 40px !important;
          }
        }
        .value-input {
          :deep(.el-input__inner) {
            height: 38px;
            line-height: 38px;
          }
          :deep(.el-input-number__decrease),
          :deep(.el-input-number__increase) {
            border-radius: 0 !important;
          }
        }
      }
    }
  }
  
  .mini-alert {
    margin-bottom: 12px;
    padding: 8px 16px;
  }
}

// 3. 日志样式 (Terminal 风格 - 优化版)
.log-viewer {
  background: #0d1117;
  border-radius: 6px;
  padding: 12px;
  height: 350px;
  overflow-y: auto;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  scroll-behavior: smooth;

  // 自定义滚动条
  &::-webkit-scrollbar {
    width: 8px;
  }

  &::-webkit-scrollbar-track {
    background: #161b22;
    border-radius: 4px;
  }

  &::-webkit-scrollbar-thumb {
    background: #30363d;
    border-radius: 4px;

    &:hover {
      background: #484f58;
    }
  }

  .empty-logs {
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: #5c6370;
    gap: 8px;
    font-size: 14px;
  }

  .log-line {
    display: flex;
    gap: 8px;
    margin-bottom: 4px;
    word-break: break-all;
    animation: fadeInLog 0.2s ease-in;

    .log-ts {
      color: #58a6ff;
      flex-shrink: 0;
      min-width: 70px;
      font-weight: 500;
    }

    .log-lvl {
      flex-shrink: 0;
      font-weight: bold;
      min-width: 50px;

      &.info { color: #3fb950; }
      &.warn, &.warning { color: #d29922; }
      &.err, &.error { color: #f85149; }
      &.success { color: #3fb950; }
      &.debug { color: #8b949e; }
    }

    .log-msg {
      color: #c9d1d9;
      flex: 1;
    }
  }
}

@keyframes fadeInLog {
  from {
    opacity: 0;
    transform: translateY(-3px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.status-badge.pulse {
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0% { opacity: 1; }
  50% { opacity: 0.6; }
  100% { opacity: 1; }
}

.desktop-only {
  @media (max-width: $mobile-break) {
    display: none !important;
  }
}
.mobile-only {
  @media (min-width: 769px) {
    display: none !important;
  }
}
</style>