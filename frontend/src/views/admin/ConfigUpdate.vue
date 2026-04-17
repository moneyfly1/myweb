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
          <el-button type="primary" @click="saveConfig" :loading="loading.save" class="save-config-btn">
            <el-icon><Check /></el-icon>
            <span>保存配置</span>
          </el-button>
        </div>
      </template>

      <el-form :model="config" label-position="top" class="config-form">
        
        <!-- 节点源 URL 配置 -->
        <div class="form-section">
          <div class="section-title">
            <el-icon><Connection /></el-icon> 节点源列表
          </div>
          <div class="url-list" ref="urlListRef">
            <div v-for="(item, index) in config.urls" :key="item.uid" class="list-item-wrapper">
              <div class="input-with-action" :class="{ 'manual-node-item': item.isManual }">
                <div class="drag-handle">
                  <el-icon><Rank /></el-icon>
                </div>
                <template v-if="item.isManual">
                  <div class="manual-node-label styled-input">
                    <span class="index-badge" v-if="!isMobile">{{ index + 1 }}</span>
                    <el-icon><User /></el-icon>
                    <span>手动节点</span>
                    <el-tag size="small" type="warning" style="margin-left: 8px;">固定</el-tag>
                  </div>
                </template>
                <template v-else>
                  <el-input
                    v-model="item.value"
                    placeholder="请输入订阅/节点源 URL"
                    class="styled-input"
                  >
                    <template #prefix v-if="!isMobile">
                      <span class="index-badge">{{ index + 1 }}</span>
                    </template>
                  </el-input>
                </template>
                <el-button
                  v-if="!item.isManual"
                  type="danger"
                  plain
                  @click="removeUrl(index)"
                  :disabled="realUrlCount <= 1"
                  class="action-btn-side"
                >
                  <el-icon><Delete /></el-icon>
                </el-button>
                <div v-else class="action-btn-side placeholder-btn"></div>
              </div>
            </div>
            <el-button type="primary" plain @click="addUrl" class="add-item-btn">
              <el-icon><Plus /></el-icon> 添加订阅源
            </el-button>
          </div>
        </div>

        <el-divider border-style="dashed" />

        <!-- 定时任务配置 -->
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

          <div class="keyword-container">
            <!-- 网格化关键词列表 -->
            <div class="keyword-grid" v-if="config.filter_keywords.length > 0">
              <div v-for="(keyword, index) in config.filter_keywords" :key="index" class="list-item-wrapper">
                <div class="input-with-action">
                  <el-input 
                    v-model="config.filter_keywords[index]" 
                    placeholder="输入关键词"
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
             <el-tag v-if="isLogPolling" type="success" size="small" class="status-badge pulse">获取中...</el-tag>
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
      
      <div class="log-viewer terminal-style" ref="logViewerRef">
        <div v-if="logs.length === 0" class="empty-logs">
           <el-icon :size="40"><Monitor /></el-icon>
           <span>等待系统输出日志...</span>
        </div>
        <template v-else>
          <div 
            v-for="(log, index) in logs" 
            :key="index" 
            class="log-line"
          >
            <div class="log-meta">
              <span class="log-time">{{ extractLogTime(log) }}</span>
              <span class="log-level-badge" :class="extractLogLevelClass(log)">
                {{ extractLogLevelText(log) }}
              </span>
            </div>
            <span class="log-message" :class="extractLogLevelClass(log)">{{ extractLogMessage(log) }}</span>
          </div>
        </template>
      </div>
    </el-card>

    <!-- 移动端底部固定保存按钮 -->
    <div class="mobile-save-bar mobile-only">
      <el-button type="primary" @click="saveConfig" :loading="loading.save" size="large" style="width: 100%;">
        <el-icon><Check /></el-icon>
        <span>保存配置</span>
      </el-button>
    </div>

  </div>
</template>

<script>
import { ref, reactive, computed, onMounted, onUnmounted, onBeforeUnmount, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  VideoPlay, VideoPause, View, Refresh, Check, Delete, Plus,
  Connection, Filter, Warning, Document, Rank, Monitor, User
} from '@element-plus/icons-vue'
import { configUpdateAPI } from '@/utils/api'
import Sortable from 'sortablejs'

export default {
  name: 'ConfigUpdate',
  components: {
    VideoPlay, VideoPause, View, Refresh, Check, Delete, Plus,
    Connection, Filter, Warning, Document, Rank, Monitor, User
  },
  setup() {
    // 状态定义
    const status = ref({
      is_running: false,
      scheduled_enabled: false,
      last_update: null
    })
    
    const config = reactive({
      urls: [], // [{ uid: string, value: string }]
      update_interval: 3600,
      enable_schedule: false,
      filter_keywords: []
    })
    
    const intervalUnit = ref('hour')
    const intervalValue = ref(1)
    const logs = ref([])
    const isLogPolling = ref(false)
    const isMobile = ref(false)
    
    const urlListRef = ref(null)
    const logViewerRef = ref(null)
    
    const loading = reactive({
      start: false, stop: false, test: false, refresh: false, save: false,
    })

    let pollingTimer = null // 统一的轮询定时器
    let sortableInstance = null
    const MANUAL_NODE_UID = '__manual_node__'

    const realUrlCount = computed(() => config.urls.filter(item => !item.isManual).length)

    const checkMobile = () => {
      isMobile.value = window.innerWidth <= 768
    }

    const generateUid = () => {
      return Date.now().toString(36) + Math.random().toString(36).substring(2)
    }

    // ========== 日志格式化工具 ==========
    const extractLogTime = (log) => {
      const timeStr = log && (log.timestamp || log.time || log.date);
      if (!timeStr) {
        const d = new Date();
        return `${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}:${String(d.getSeconds()).padStart(2,'0')}`;
      }
      try {
        const d = new Date(timeStr);
        if (isNaN(d.getTime())) return timeStr.substring(0, 8); // fallback
        const HH = String(d.getHours()).padStart(2, '0');
        const mm = String(d.getMinutes()).padStart(2, '0');
        const ss = String(d.getSeconds()).padStart(2, '0');
        return `${HH}:${mm}:${ss}`;
      } catch {
        return '00:00:00';
      }
    }

    const extractLogLevelClass = (log) => {
      const lvl = (log && log.level) ? String(log.level).toLowerCase() : 'info';
      if (['error', 'err', 'critical', 'fatal'].includes(lvl)) return 'error';
      if (['warn', 'warning'].includes(lvl)) return 'warn';
      if (['success', 'ok'].includes(lvl)) return 'success';
      if (['debug', 'trace'].includes(lvl)) return 'debug';
      return 'info';
    }

    const extractLogLevelText = (log) => {
      const lvl = (log && log.level) ? String(log.level).toUpperCase() : 'INFO';
      return lvl.length > 5 ? lvl.substring(0, 5) : lvl.padEnd(5, ' ');
    }

    const extractLogMessage = (log) => {
      if (!log) return '';
      const msg = log.message || log.msg || log.text || (typeof log === 'string' ? log : '');
      if (typeof msg === 'object') {
        try { return JSON.stringify(msg); } catch { return String(msg); }
      }
      return msg;
    }

    const scrollToBottom = () => {
      nextTick(() => {
        if (logViewerRef.value) {
          logViewerRef.value.scrollTop = logViewerRef.value.scrollHeight
        }
      })
    }

    // ========== 时间间隔转换 ==========
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

    const formatInterval = (seconds) => {
      if (!seconds) return '未设置'
      if (seconds < 60) return `${seconds}秒`
      if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟`
      if (seconds < 86400) return `${Math.floor(seconds / 3600)}小时`
      return `${Math.floor(seconds / 86400)}天`
    }

    // ========== API 通信 ==========
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
          const rawInterval = data.schedule_interval ?? data.update_interval ?? 3600
          const interval = typeof rawInterval === 'number' ? rawInterval : (parseInt(rawInterval) || 3600)

          const rawUrls = Array.isArray(data.urls) ? (data.urls.length ? data.urls : ['']) : ['']
          config.urls = rawUrls.map(url => ({ uid: generateUid(), value: url }))

          // 插入手动节点固定项
          const manualPos = typeof data.manual_node_position === 'number' ? data.manual_node_position : config.urls.length
          const clampedPos = Math.min(Math.max(0, manualPos), config.urls.length)
          config.urls.splice(clampedPos, 0, { uid: MANUAL_NODE_UID, value: '', isManual: true })

          config.filter_keywords = Array.isArray(data.filter_keywords) ? data.filter_keywords : []
          config.enable_schedule = !!data.enable_schedule
          config.update_interval = interval
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
          scrollToBottom()
        }
      } catch (error) {
        console.error('日志获取失败', error)
      }
    }

    const refreshLogs = async () => {
      loading.refresh = true
      await getLogs()
      loading.refresh = false
    }

    // ========== 实时刷新控制 (核心修复点) ==========
    const startPolling = () => {
      if (pollingTimer) clearTimeout(pollingTimer)
      isLogPolling.value = true

      const poll = async () => {
        try {
          await getStatus()
          await getLogs() // 强制同步获取最新日志，确保完全实时
        } finally {
          // 如果状态仍然在运行，1.5秒后继续拉取
          if (status.value.is_running) {
            pollingTimer = setTimeout(poll, 1500)
          } else {
            stopPolling()
            await getLogs() // 任务结束，最后拉取一次最终日志
          }
        }
      }
      
      // 首次延时启动
      pollingTimer = setTimeout(poll, 1000)
    }

    const stopPolling = () => {
      if (pollingTimer) {
        clearTimeout(pollingTimer)
        pollingTimer = null
      }
      isLogPolling.value = false
    }

    // ========== 交互操作 ==========
    const startUpdate = async () => {
      loading.start = true
      try {
        const response = await configUpdateAPI.startUpdate()
        if (response.data.success) {
          ElMessage.success('更新任务已启动，正在执行...')
          await getLogs() 
          startPolling() // 启动强大的高频轮询
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
          stopPolling()
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
          startPolling()
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

    const saveConfig = async () => {
      loading.save = true
      try {
        updateInterval()
        const urls = (config.urls || []).filter(item => !item.isManual).map(item => item.value).filter(val => val && String(val).trim())
        const filter_keywords = (config.filter_keywords || []).filter(keyword => keyword && String(keyword).trim())

        // 计算手动节点在纯URL列表中的位置
        const manualIndex = (config.urls || []).findIndex(item => item.isManual)
        const manual_node_position = manualIndex >= 0 ? (config.urls || []).slice(0, manualIndex).filter(item => !item.isManual).length : urls.length

        if (!urls.length) {
          ElMessage.warning('至少需要配置一个节点源URL')
          loading.save = false
          return
        }

        const configToSave = {
          urls,
          filter_keywords,
          enable_schedule: config.enable_schedule,
          schedule_interval: config.update_interval,
          manual_node_position
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

    // ========== 列表维护 ==========
    const addUrl = () => {
      if (!config.urls) config.urls = []
      config.urls.push({ uid: generateUid(), value: '' })
    }
    const removeUrl = (index) => {
      if (config.urls[index]?.isManual) return
      if (realUrlCount.value > 1) config.urls.splice(index, 1)
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

    // ========== 生命周期 ==========
    onMounted(async () => {
      checkMobile()
      window.addEventListener('resize', checkMobile)
      
      // 并发加载配置、状态和日志，提高页面加载速度
      await Promise.all([
        getConfig(),
        getStatus(),
        getLogs()
      ])
      
      // 如果进入页面时就在运行，接管轮询
      if (status.value.is_running) {
        startPolling()
      }

      if (urlListRef.value) {
        sortableInstance = Sortable.create(urlListRef.value, {
          animation: 150,
          handle: '.drag-handle',
          ghostClass: 'sortable-ghost',
          fallbackOnBody: true,
          onEnd: (evt) => {
            const { oldIndex, newIndex } = evt
            if (oldIndex !== newIndex && oldIndex !== undefined && newIndex !== undefined) {
              const movedItem = config.urls.splice(oldIndex, 1)[0]
              config.urls.splice(newIndex, 0, movedItem)
            }
          }
        })
      }
    })

    onUnmounted(() => {
      stopPolling()
      if (sortableInstance) {
        sortableInstance.destroy()
        sortableInstance = null
      }
    })
    
    onBeforeUnmount(() => window.removeEventListener('resize', checkMobile))


    return {
      status, config, logs, loading, isLogPolling, isMobile, realUrlCount,
      intervalUnit, intervalValue, urlListRef, logViewerRef,
      startUpdate, stopUpdate, testUpdate, refreshStatus, saveConfig,
      refreshLogs, clearLogs, addUrl, removeUrl, addKeyword, removeKeyword,
      updateInterval, formatInterval, handleScheduleChange,
      extractLogTime, extractLogLevelClass, extractLogLevelText, extractLogMessage
    }
  }
}
</script>

<style scoped lang="scss">
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
    padding-bottom: 80px;
    background-color: $bg-color;
    min-height: 100vh;
  }
}

.list-card {
  margin-bottom: 20px;
  border-radius: $card-border-radius;
  overflow: visible;
  
  :deep(.el-card__header) {
    padding: 16px 20px;
    border-bottom: 1px solid #f0f2f5;
    background: #fff;
    overflow: visible;
    @media (max-width: $mobile-break) { padding: 12px 16px; }
  }
  
  :deep(.el-card__body) {
    padding: 20px;
    @media (max-width: $mobile-break) { padding: 16px; }
  }
}

.card-header {
  display: flex; justify-content: space-between; align-items: center; flex-wrap: nowrap;
  .header-left { display: flex; align-items: center; gap: 10px; }
  .header-actions { display: flex; gap: 10px; &.compact { gap: 6px; } }
  .save-config-btn {
    flex-shrink: 0;
    @media (max-width: $mobile-break) {
      padding: 6px 12px;
      font-size: 12px;
    }
  }
  .header-title {
    flex-shrink: 1; min-width: 0;
  }
}

/* --- 1. 面板样式 --- */
.control-panel {
  display: flex; flex-direction: column; gap: 16px;
  .mobile-status-bar {
    display: flex; align-items: center; justify-content: space-between;
    background: #f0f9eb; padding: 8px 12px; border-radius: 6px; border: 1px solid #e1f3d8;
    .label { font-size: 14px; color: #606266; }
  }
  .action-grid {
    display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px;
    @media (max-width: $mobile-break) { grid-template-columns: repeat(2, 1fr); gap: 8px; }
    .grid-btn {
      height: auto; padding: 10px 8px; width: 100%; border-radius: 6px; transition: all 0.2s;
      .btn-content { display: flex; flex-direction: column; align-items: center; gap: 4px; span { font-size: 12px; font-weight: 500; } }
      &:hover { transform: translateY(-1px); box-shadow: 0 2px 8px rgba(0,0,0,0.08); }
      @media (max-width: $mobile-break) { padding: 8px 6px; margin: 0; .btn-content span { font-size: 11px; } }
    }
  }
}

/* --- 2. 表单与列表样式 --- */
.config-form {
  .form-section {
    margin-bottom: 24px; &:last-child { margin-bottom: 0; }
    .section-title { font-size: 14px; font-weight: 600; color: #606266; margin-bottom: 12px; display: flex; align-items: center; gap: 6px; .info-icon { color: #e6a23c; cursor: help; } }
  }

  .list-item-wrapper {
    margin-bottom: 12px; border: 1px solid #dcdfe6; border-radius: 6px; overflow: hidden; transition: border-color 0.2s; background: #fff;
    &:focus-within { border-color: $primary-color; }
    .input-with-action {
      display: flex; gap: 0; align-items: stretch;
      .styled-input {
        flex: 1; min-width: 0;
        :deep(.el-input__wrapper), :deep(.el-input__inner) { box-shadow: none !important; border: none !important; border-radius: 0 !important; background: transparent !important; }
        :deep(.el-input__wrapper) { padding: 8px 12px; &.is-focus { box-shadow: none !important; } }
        .index-badge { color: #909399; font-size: 12px; margin-right: 4px; font-weight: 600; }
      }
      .action-btn-side { width: 44px; flex-shrink: 0; border-radius: 0; border-left: 1px solid #dcdfe6; }
    }
  }
  
  .add-item-btn { width: 100%; border-style: dashed; margin-top: 4px; height: 40px; }

  /* 新增：过滤关键词网格布局 */
  .keyword-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 12px;
    margin-bottom: 12px;

    /* 覆盖内部元素的通用 margin */
    .list-item-wrapper {
      margin-bottom: 0;
    }

    /* 响应式调整 */
    @media (max-width: $mobile-break) {
      grid-template-columns: repeat(2, 1fr);
    }
    @media (max-width: 480px) {
      grid-template-columns: 1fr;
    }
  }
  
  .schedule-section {
    .schedule-row { display: flex; align-items: stretch; gap: 0; background: #fff; border: 1px solid #dcdfe6; border-radius: 6px; overflow: hidden; }
    .schedule-item { display: flex; align-items: center; gap: 10px; padding: 0 14px; min-height: 40px; .schedule-label { font-size: 14px; color: #606266; font-weight: 500; white-space: nowrap; } }
    .switch-item { flex: 0 0 auto; border-right: 1px solid #dcdfe6; }
    .schedule-divider { width: 1px; background: #dcdfe6; flex-shrink: 0; }
    .interval-item { flex: 1; min-width: 0; padding-right: 0; border-right: none; .schedule-label { flex-shrink: 0; } }
    .interval-inputs {
      flex: 1; display: flex; min-width: 0;
      .unit-select, .value-input { flex: 1; min-width: 0; }
      .unit-select { :deep(.el-input), :deep(.el-input__wrapper) { height: 36px !important; min-height: 36px !important; } :deep(.el-input__wrapper) { border: none !important; border-radius: 0 !important; border-right: 1px solid #ebeef5; box-shadow: none !important; background: transparent !important; } :deep(.el-input__inner) { border: none !important; box-shadow: none !important; } }
      .value-input { :deep(.el-input-number), :deep(.el-input), :deep(.el-input__wrapper) { height: 36px !important; min-height: 36px !important; } :deep(.el-input__wrapper) { border: none !important; border-radius: 0 !important; box-shadow: none !important; background: transparent !important; } :deep(.el-input__inner) { height: 34px; line-height: 34px; border: none !important; box-shadow: none !important; } :deep(.el-input-number__decrease), :deep(.el-input-number__increase) { border-radius: 0 !important; } }
    }
    .desc-text { font-size: 12px; color: #909399; margin-top: 8px; padding-left: 2px; }
    
    @media (max-width: $mobile-break) {
      .schedule-row { flex-direction: column; border-radius: 6px; }
      .schedule-item { border-right: none; border-bottom: 1px solid #dcdfe6; min-height: 44px; padding: 0 14px; &:last-child { border-bottom: none; } }
      .switch-item { justify-content: space-between; }
      .schedule-divider { display: none; }
      .interval-item { flex-direction: column; align-items: stretch; gap: 8px; padding: 12px 14px; .schedule-label { margin-bottom: 0; } }
      .interval-inputs { .unit-select, .value-input { :deep(.el-input__wrapper) { height: 40px !important; min-height: 40px !important; } } .value-input { :deep(.el-input__inner) { height: 38px; line-height: 38px; } :deep(.el-input-number__decrease), :deep(.el-input-number__increase) { border-radius: 0 !important; } } }
    }
  }
  .mini-alert { margin-bottom: 12px; padding: 8px 16px; }
}

/* --- 3. 专业级终端日志样式 (核心重构) --- */
.terminal-style {
  background-color: #1e1e1e; /* VS Code 默认黑底 */
  border-radius: 6px;
  padding: 12px 0; /* 上下内边距，左右不留防止悬停背景截断 */
  height: 380px;
  overflow-y: auto;
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.5;
  color: #cccccc;
  scroll-behavior: smooth;

  /* 滚动条美化 */
  &::-webkit-scrollbar { width: 10px; height: 10px; }
  &::-webkit-scrollbar-track { background: transparent; }
  &::-webkit-scrollbar-thumb { background: #424242; border: 2px solid #1e1e1e; border-radius: 5px; &:hover { background: #4f4f4f; } }

  .empty-logs {
    height: 100%; display: flex; flex-direction: column; align-items: center; justify-content: center; color: #5c6370; gap: 12px; font-size: 14px;
    .el-icon { opacity: 0.5; }
  }

  .log-line {
    display: flex;
    padding: 2px 16px; /* 控制两边边距 */
    word-break: break-all;
    transition: background-color 0.1s;
    animation: fadeInLog 0.2s ease-in;

    /* 鼠标悬停微亮效果 */
    &:hover { background-color: #2a2d2e; }

    .log-meta {
      display: flex;
      flex-shrink: 0;
      width: 140px; /* 固定头部宽度，保证对齐 */
      gap: 12px;
      user-select: none;
    }

    .log-time {
      color: #858585; /* 暗灰色时间戳 */
      font-weight: normal;
    }

    .log-level-badge {
      font-weight: 600;
      letter-spacing: 0.5px;
      
      /* 日志级别色彩体系 */
      &.info    { color: #3b8eea; } /* 蓝色 */
      &.warn    { color: #cca700; } /* 黄色 */
      &.error   { color: #f48771; } /* 红色 */
      &.success { color: #89d185; } /* 绿色 */
      &.debug   { color: #8b949e; } /* 灰蓝色 */
    }

    .log-message {
      flex: 1;
      white-space: pre-wrap; /* 允许换行保留空格 */
      
      /* 使错误和警告的正文也有一点颜色倾向 */
      &.error { color: #f48771; opacity: 0.9; }
      &.warn  { color: #e5c00b; opacity: 0.9; }
    }
    
    @media (max-width: $mobile-break) {
      flex-direction: column;
      gap: 2px;
      margin-bottom: 6px;
      .log-meta { width: 100%; }
      .log-message { padding-left: 0; }
    }
  }
}

@keyframes fadeInLog {
  from { opacity: 0; transform: translateX(-4px); }
  to { opacity: 1; transform: translateX(0); }
}

.drag-handle {
  cursor: grab; padding: 0 8px; display: flex; align-items: center; color: #909399; font-size: 18px;
  &:hover { color: #409eff; }
  &:active { cursor: grabbing; }
}
.manual-node-item {
  background: #fdf6ec;
  .manual-node-label {
    flex: 1; min-width: 0; display: flex; align-items: center; gap: 8px; padding: 8px 12px;
    font-size: 14px; color: #e6a23c; font-weight: 500;
    .index-badge { color: #909399; font-size: 12px; margin-right: 4px; font-weight: 600; }
  }
  .placeholder-btn { width: 44px; flex-shrink: 0; }
}

.sortable-ghost { opacity: 0.4; background: #e6f7ff !important; border: 1px dashed #409eff; }
.status-badge.pulse { animation: pulse 2s infinite; }
@keyframes pulse { 0% { opacity: 1; } 50% { opacity: 0.6; } 100% { opacity: 1; } }
.desktop-only { @media (max-width: $mobile-break) { display: none !important; } }
.mobile-only { @media (min-width: 769px) { display: none !important; } }

.mobile-save-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 12px 16px;
  background: #fff;
  box-shadow: 0 -2px 12px rgba(0, 0, 0, 0.1);
  z-index: 999;
}
</style>