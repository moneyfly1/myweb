<template>
  <div class="config-update">
    <el-card class="action-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>操作控制</span>
        </div>
      </template>
      <div class="action-buttons">
        <div class="action-buttons-row">
          <el-button 
            type="primary" 
            :loading="loading.start"
            @click="startUpdate"
            :disabled="status.is_running"
            class="action-btn"
          >
            <el-icon><VideoPlay /></el-icon>
            开始更新
          </el-button>
          <el-button 
            type="warning" 
            :loading="loading.stop"
            @click="stopUpdate"
            :disabled="!status.is_running"
            class="action-btn"
          >
            <el-icon><VideoPause /></el-icon>
            停止更新
          </el-button>
        </div>
        <div class="action-buttons-row">
          <el-button 
            type="info" 
            :loading="loading.test"
            @click="testUpdate"
            :disabled="status.is_running"
            class="action-btn"
          >
            <el-icon><View /></el-icon>
            测试更新
          </el-button>
          <el-button 
            type="success" 
            :loading="loading.refresh"
            @click="refreshStatus"
            class="action-btn"
          >
            <el-icon><Refresh /></el-icon>
            刷新状态
          </el-button>
        </div>
      </div>
    </el-card>
    <el-card style="margin-bottom: 20px;" class="config-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>配置设置</span>
          <el-button type="primary" @click="saveConfig" :loading="loading.save" class="save-config-btn">
            <el-icon><Check /></el-icon>
            保存配置
          </el-button>
        </div>
      </template>
      
      <el-form :model="config" :label-width="isMobile ? '0' : '120px'" class="config-form">
        <el-form-item label="节点源URL" :label-width="isMobile ? '0' : undefined">
          <div class="form-item-content">
            <div v-for="(url, index) in config.urls" :key="index" class="url-item">
              <el-input 
                v-model="config.urls[index]" 
                placeholder="请输入节点源URL"
                class="config-input"
              />
              <el-button 
                type="danger" 
                @click="removeUrl(index)"
                :disabled="config.urls.length <= 1"
                class="delete-btn"
              >
                <el-icon><Delete /></el-icon>
                删除
              </el-button>
            </div>
            <el-button type="primary" @click="addUrl" class="add-btn">
              <el-icon><Plus /></el-icon>
              添加URL
            </el-button>
          </div>
        </el-form-item>
        
        <el-form-item label="目标目录" :label-width="isMobile ? '0' : undefined">
          <el-input 
            v-model="config.target_dir" 
            placeholder="配置文件保存目录"
            class="config-input"
          />
        </el-form-item>
        
        <div class="form-row-group">
          <el-form-item label="v2ray文件名" :label-width="isMobile ? '0' : '120px'">
            <el-input 
              v-model="config.v2ray_file" 
              placeholder="v2ray配置文件名称"
              class="config-input"
            />
          </el-form-item>
          
          <el-form-item label="clash文件名" :label-width="isMobile ? '0' : '120px'">
            <el-input 
              v-model="config.clash_file" 
              placeholder="clash配置文件名称"
              class="config-input"
            />
          </el-form-item>
        </div>
        
        <el-form-item label="更新间隔" :label-width="isMobile ? '0' : undefined">
          <div class="interval-selector">
            <el-select 
              v-model="intervalUnit" 
              placeholder="选择单位"
              class="interval-unit-select"
            >
              <el-option label="分钟" value="minute" />
              <el-option label="小时" value="hour" />
              <el-option label="天" value="day" />
            </el-select>
            <el-input-number 
              v-model="intervalValue" 
              :min="1" 
              :max="intervalUnit === 'minute' ? 1440 : intervalUnit === 'hour' ? 24 : 30"
              placeholder="间隔数值"
              class="config-input-number interval-number"
              @change="updateInterval"
            />
            <div class="interval-tip">
              <span>当前间隔: {{ formatInterval(config.update_interval) }}</span>
            </div>
          </div>
        </el-form-item>
        
        <el-form-item label="启用定时任务" :label-width="isMobile ? '0' : undefined">
          <div class="schedule-switch-container">
            <el-switch 
              v-model="config.enable_schedule" 
              @change="handleScheduleChange"
            />
            <span class="schedule-tip" v-if="config.enable_schedule && config.update_interval">
              将每 {{ formatInterval(config.update_interval) }} 自动更新
            </span>
          </div>
        </el-form-item>
        
        <el-form-item label="过滤关键词" :label-width="isMobile ? '0' : undefined">
          <div class="form-item-content">
            <div class="filter-keywords-tip">
              <el-alert
                type="info"
                :closable="false"
                show-icon
                style="margin-bottom: 12px;"
              >
                <template #default>
                  <div style="font-size: 13px;">
                    <p style="margin: 0 0 4px 0;"><strong>过滤说明：</strong></p>
                    <p style="margin: 0;">• 过滤针对从所有订阅源获取的节点</p>
                    <p style="margin: 0;">• 如果节点名称或服务器地址包含任何关键词，该节点将被过滤掉，不会导入数据库</p>
                    <p style="margin: 0;">• 关键词匹配不区分大小写</p>
                    <p style="margin: 0;">• 支持多个关键词，每个关键词独立过滤</p>
                  </div>
                </template>
              </el-alert>
            </div>
            <div v-for="(keyword, index) in config.filter_keywords" :key="index" class="keyword-item">
              <el-input 
                v-model="config.filter_keywords[index]" 
                placeholder="输入关键词（将过滤包含此关键词的节点）"
                class="config-input"
              />
              <el-button 
                type="danger" 
                @click="removeKeyword(index)"
                class="delete-btn"
              >
                <el-icon><Delete /></el-icon>
                删除
              </el-button>
            </div>
            <el-button type="primary" @click="addKeyword" class="add-btn">
              <el-icon><Plus /></el-icon>
              添加关键词
            </el-button>
          </div>
        </el-form-item>
      </el-form>
    </el-card>
    <el-card style="margin-bottom: 20px;" class="files-card" shadow="never">
      <template #header>
        <span>生成的文件</span>
      </template>
      <div class="desktop-only">
        <el-table :data="fileList" style="width: 100%" class="files-table">
          <el-table-column prop="name" label="文件名" width="200" />
          <el-table-column prop="path" label="路径" />
          <el-table-column prop="size" label="大小" width="120">
            <template #default="scope">
              {{ formatFileSize(scope.row.size) }}
            </template>
          </el-table-column>
          <el-table-column prop="modified" label="修改时间" width="180">
            <template #default="scope">
              {{ formatTime(scope.row.modified) }}
            </template>
          </el-table-column>
          <el-table-column prop="exists" label="状态" width="100">
            <template #default="scope">
              <el-tag :type="scope.row.exists ? 'success' : 'danger'">
                {{ scope.row.exists ? '存在' : '不存在' }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <div class="mobile-files-list" v-if="isMobile">
        <div 
          v-for="file in fileList" 
          :key="file.name"
          class="mobile-file-card"
        >
          <div class="file-header">
            <div class="file-name">{{ file.name }}</div>
            <el-tag :type="file.exists ? 'success' : 'danger'" size="small">
              {{ file.exists ? '存在' : '不存在' }}
            </el-tag>
          </div>
          <div class="file-info">
            <div class="file-row">
              <span class="file-label">路径：</span>
              <span class="file-value">{{ file.path || '-' }}</span>
            </div>
            <div class="file-row">
              <span class="file-label">大小：</span>
              <span class="file-value">{{ formatFileSize(file.size) }}</span>
            </div>
            <div class="file-row">
              <span class="file-label">修改时间：</span>
              <span class="file-value">{{ formatTime(file.modified) }}</span>
            </div>
          </div>
        </div>
        <div v-if="fileList.length === 0" class="empty-files">
          <el-empty description="暂无文件信息" :image-size="80" />
        </div>
      </div>
    </el-card>
    <el-card class="logs-card" shadow="never">
      <template #header>
        <div class="card-header">
          <div class="log-header-left">
            <span>更新日志</span>
            <el-tag v-if="isLogPolling" type="success" size="small" class="live-indicator">
              <el-icon><VideoPlay /></el-icon>
              实时更新中
            </el-tag>
            <el-tag v-if="newLogCount > 0" type="info" size="small" class="new-log-indicator">
              <el-icon><Bell /></el-icon>
              {{ newLogCount }} 条新日志
            </el-tag>
          </div>
          <div class="log-header-buttons">
            <el-button type="primary" size="small" @click="refreshLogs" class="log-btn">
              <el-icon><Refresh /></el-icon>
              刷新日志
            </el-button>
            <el-button type="warning" size="small" @click="clearLogs" class="log-btn">
              <el-icon><Delete /></el-icon>
              清理日志
            </el-button>
          </div>
        </div>
      </template>
      
      <div class="log-container">
        <div 
          v-for="(log, index) in logs" 
          :key="index" 
          class="log-item"
          :class="log.level"
        >
          <span class="log-time">{{ formatTime(log.timestamp) }}</span>
          <span class="log-level">{{ log.level.toUpperCase() }}</span>
          <span class="log-message">{{ log.message }}</span>
        </div>
        <div v-if="logs.length === 0" class="no-logs">
          暂无日志
        </div>
      </div>
    </el-card>

  </div>
</template>

<script>
import { ref, reactive, onMounted, onUnmounted, computed, onBeforeUnmount } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  VideoPlay, VideoPause, Timer, Document, Clock, Refresh, 
  Check, Delete, Plus, View, Bell
} from '@element-plus/icons-vue'
import { configUpdateAPI } from '@/utils/api'
import api from '@/utils/api'

export default {
  name: 'ConfigUpdate',
  components: {
    VideoPlay, VideoPause, Timer, Document, Clock, Refresh,
    Check, Delete, Plus, View, Bell
  },
  setup() {
    const status = ref({
      is_running: false,
      scheduled_enabled: false,
      last_update: null,
      next_update: null,
      config_exists: false
    })
    
    const config = reactive({
      urls: [''],
      target_dir: './uploads/config',
      v2ray_file: 'xr',
      clash_file: 'clash.yaml',
      update_interval: 3600,
      enable_schedule: false,
      filter_keywords: []
    })
    const intervalUnit = ref('hour')
    const intervalValue = ref(1)
    
    const fileList = ref([])
    const logs = ref([])
    const isLogPolling = ref(false)
    const newLogCount = ref(0)
    
    const loading = reactive({
      start: false,
      stop: false,
      test: false,
      refresh: false,
      save: false,
    })
    const isMobile = ref(false)
    const checkMobile = () => {
      isMobile.value = window.innerWidth <= 768
    }
    let statusPollingInterval = null
    let refreshInterval = null
    let logPollingInterval = null
    const startStatusPolling = () => {
      if (statusPollingInterval) {
        clearInterval(statusPollingInterval)
      }
      
      statusPollingInterval = setInterval(async () => {
        await getStatus()
        if (!status.value.is_running) {
          stopStatusPolling()
          stopRefreshInterval()
          stopLogPolling()
        }
      }, 1000)
    }
    const stopStatusPolling = () => {
      if (statusPollingInterval) {
        clearInterval(statusPollingInterval)
        statusPollingInterval = null
      }
    }
    const startRefreshInterval = () => {
      if (refreshInterval) {
        clearInterval(refreshInterval)
      }
      
      refreshInterval = setInterval(() => {
        if (!loading.refresh && status.value.is_running) {
          getStatus()
        }
      }, 10000)
    }
    const stopRefreshInterval = () => {
      if (refreshInterval) {
        clearInterval(refreshInterval)
        refreshInterval = null
      }
    }
    const startLogPolling = () => {
      if (logPollingInterval) {
        clearInterval(logPollingInterval)
      }
      
      isLogPolling.value = true
      logPollingInterval = setInterval(async () => {
        try {
          if (!loading.refresh && status.value.is_running) {
            await getLogs()
          } else if (!status.value.is_running) {
            stopLogPolling()
          }
        } catch (error) {
          stopLogPolling()
        }
      }, 2000)
    }
    const stopLogPolling = () => {
      if (logPollingInterval) {
        clearInterval(logPollingInterval)
        logPollingInterval = null
      }
      isLogPolling.value = false
    }
    const getStatus = async () => {
      try {
        const response = await configUpdateAPI.getStatus()
        if (response.data.success) {
          status.value = response.data.data
        } else {
          }
      } catch (error) {
        ElMessage.error('获取状态失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const getConfig = async () => {
      try {
        const response = await configUpdateAPI.getConfig()
        if (response.data.success) {
          Object.assign(config, response.data.data)
          initIntervalSelector()
        } else {
          }
      } catch (error) {
        ElMessage.error('获取配置失败: ' + (error.response?.data?.message || error.message))
      }
    }
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
      if (intervalUnit.value === 'minute') {
        seconds = (intervalValue.value || 1) * 60
      } else if (intervalUnit.value === 'hour') {
        seconds = (intervalValue.value || 1) * 3600
      } else if (intervalUnit.value === 'day') {
        seconds = (intervalValue.value || 1) * 86400
      }
      config.update_interval = seconds
    }
    const formatInterval = (seconds) => {
      if (!seconds) return '未设置'
      if (seconds < 60) {
        return `${seconds}秒`
      } else if (seconds < 3600) {
        return `${Math.floor(seconds / 60)}分钟`
      } else if (seconds < 86400) {
        return `${Math.floor(seconds / 3600)}小时`
      } else {
        return `${Math.floor(seconds / 86400)}天`
      }
    }
    const handleScheduleChange = async (value) => {
      if (value) {
        if (!config.update_interval || config.update_interval < 60) {
          ElMessage.warning('请先设置更新间隔（至少1分钟）')
          config.enable_schedule = false
          return
        }
        await saveConfig()
        ElMessage.success(`定时任务已启用，将每 ${formatInterval(config.update_interval)} 自动更新`)
      } else {
        ElMessage.info('定时任务已禁用')
      }
    }
    const getFiles = async () => {
      try {
        const response = await configUpdateAPI.getFiles()
        if (response.data.success) {
          const files = response.data.data
          fileList.value = [
            {
              name: 'v2ray配置',
              path: files.v2ray?.path || '',
              size: files.v2ray?.size || 0,
              modified: files.v2ray?.modified || null,
              exists: files.v2ray?.exists || false
            },
            {
              name: 'clash配置',
              path: files.clash?.path || '',
              size: files.clash?.size || 0,
              modified: files.clash?.modified || null,
              exists: files.clash?.exists || false
            }
          ]
        } else {
          }
      } catch (error) {
        ElMessage.error('获取文件列表失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const getLogs = async () => {
      try {
        const response = await configUpdateAPI.getLogs()
        if (response.data.success) {
          const oldLogCount = logs.value.length
          logs.value = response.data.data
          if (logs.value.length > oldLogCount) {
            newLogCount.value = logs.value.length - oldLogCount
            setTimeout(() => {
              const logContainer = document.querySelector('.log-container')
              if (logContainer) {
                logContainer.scrollTop = logContainer.scrollHeight
              }
              setTimeout(() => {
                newLogCount.value = 0
              }, 3000)
            }, 100)
          }
        } else {
          }
      } catch (error) {
        ElMessage.error('获取日志失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const startUpdate = async () => {
      loading.start = true
      try {
        const response = await configUpdateAPI.startUpdate()
        
        if (response.data.success) {
          ElMessage.success('更新任务已启动')
          startStatusPolling()
          startRefreshInterval()
          startLogPolling()
          await Promise.all([getStatus(), getLogs()])
        } else {
          ElMessage.error(response.data.message || '启动失败')
        }
      } catch (error) {
        ElMessage.error('启动失败: ' + (error.response?.data?.message || error.message))
      } finally {
        loading.start = false
      }
    }
    const stopUpdate = async () => {
      loading.stop = true
      try {
        const response = await configUpdateAPI.stopUpdate()
        
        if (response.data.success) {
          ElMessage.success('更新任务已停止')
          stopStatusPolling()
          stopRefreshInterval()
          stopLogPolling()
          await getStatus()
        } else {
          ElMessage.error(response.data.message || '停止失败')
        }
      } catch (error) {
        ElMessage.error('停止失败: ' + (error.response?.data?.message || error.message))
      } finally {
        loading.stop = false
      }
    }
    const testUpdate = async () => {
      loading.test = true
      try {
        const response = await configUpdateAPI.testUpdate()
        
        if (response.data.success) {
          ElMessage.success('测试任务已启动')
          startStatusPolling()
          startRefreshInterval()
          startLogPolling()
          await Promise.all([getStatus(), getLogs()])
        } else {
          ElMessage.error(response.data.message || '启动测试失败')
        }
      } catch (error) {
        ElMessage.error('启动测试失败: ' + (error.response?.data?.message || error.message))
      } finally {
        loading.test = false
      }
    }
    const refreshStatus = async () => {
      loading.refresh = true
      try {
        await Promise.all([getStatus(), getFiles(), getLogs()])
        ElMessage.success('状态已刷新')
      } catch (error) {
        ElMessage.error('刷新失败')
      } finally {
        loading.refresh = false
      }
    }
    const saveConfig = async () => {
      loading.save = true
      try {
        updateInterval()
        const configToSave = {
          ...config,
          urls: (config.urls || []).filter(url => url && url.trim()),
          filter_keywords: (config.filter_keywords || []).filter(keyword => keyword && keyword.trim())
        }
        if (!configToSave.urls || configToSave.urls.length === 0) {
          ElMessage.error('至少需要配置一个节点源URL')
          loading.save = false
          return
        }
        
        const response = await configUpdateAPI.updateConfig(configToSave)
        
        if (response.data.success) {
          ElMessage.success('配置已保存')
          if (config.enable_schedule) {
            await getStatus()
            if (!status.value.scheduled_enabled) {
              ElMessage.info('定时任务将在下一个间隔时间自动启动')
            }
          }
        } else {
          ElMessage.error(response.data.message || '保存失败')
        }
      } catch (error) {
        ElMessage.error('保存失败: ' + (error.response?.data?.message || error.message))
      } finally {
        loading.save = false
      }
    }
    const refreshLogs = async () => {
      await getLogs()
    }
    const clearLogs = async () => {
      try {
        await ElMessageBox.confirm('确定要清理所有日志吗？', '确认清理', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        
        const response = await configUpdateAPI.clearLogs()
        
        if (response.data.success) {
          ElMessage.success('日志已清理')
          await getLogs()
        } else {
          ElMessage.error(response.data.message || '清理失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('清理失败: ' + (error.response?.data?.message || error.message))
        }
      }
    }
    const addUrl = () => {
      if (!config.urls) {
        config.urls = ['']
      } else {
        config.urls.push('')
      }
    }
    const removeUrl = (index) => {
      if (config.urls && config.urls.length > 1) {
        config.urls.splice(index, 1)
      } else if (config.urls && config.urls.length === 1) {
        ElMessage.warning('至少需要保留一个节点源URL')
      }
    }
    const addKeyword = () => {
      if (!config.filter_keywords) {
        config.filter_keywords = ['']
      } else {
        config.filter_keywords.push('')
      }
    }
    const removeKeyword = (index) => {
      if (config.filter_keywords && config.filter_keywords.length > index) {
        config.filter_keywords.splice(index, 1)
      }
    }
    const formatTime = (timeStr) => {
      if (!timeStr) return '从未'
      try {
        return new Date(timeStr).toLocaleString()
      } catch {
        return timeStr
      }
    }
    const formatFileSize = (bytes) => {
      if (!bytes) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    }
    onMounted(async () => {
      checkMobile()
      window.addEventListener('resize', checkMobile)
      await Promise.all([getStatus(), getConfig(), getFiles(), getLogs()])
      if (status.value.is_running) {
        startLogPolling()
      }
    })
    
    onBeforeUnmount(() => {
      window.removeEventListener('resize', checkMobile)
    })
    onUnmounted(() => {
      stopStatusPolling()
      stopRefreshInterval()
      stopLogPolling()
    })
    
    return {
      status,
      config,
      fileList,
      logs,
      loading,
      isLogPolling,
      newLogCount,
      isMobile,
      intervalUnit,
      intervalValue,
      startUpdate,
      stopUpdate,
      testUpdate,
      refreshStatus,
      saveConfig,
      refreshLogs,
      clearLogs,
      addUrl,
      removeUrl,
      addKeyword,
      removeKeyword,
      formatTime,
      formatFileSize,
      updateInterval,
      formatInterval,
      handleScheduleChange
    }
  }
}
</script>

<style scoped lang="scss">
.config-update {
  padding: 20px;
  max-width: 100%;
  box-sizing: border-box;
  
  @media (max-width: 768px) {
    padding: 12px;
    width: 100% !important;
    max-width: 100% !important;
    margin: 0 !important;
    overflow-x: hidden;
  }
}


.action-card {
  @media (max-width: 768px) {
    margin-bottom: 16px !important;
  }
}

.action-buttons {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
  
  .action-buttons-row {
    display: flex;
    gap: 12px;
    width: 100%;
    
    @media (max-width: 768px) {
      flex-direction: column;
      gap: 12px;
      align-items: center;
    }
    
    .action-btn {
      flex: 1;
      height: 44px;
      font-size: 16px;
      font-weight: 500;
      margin: 0;
      
      :deep(.el-icon) {
        margin-right: 6px;
        font-size: 16px;
      }
      
      @media (max-width: 768px) {
        width: 100% !important;
        max-width: 100% !important;
        min-width: 100% !important;
        height: 44px !important;
        font-size: 16px !important;
        margin: 0 !important;
      }
    }
  }
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  
  @media (max-width: 768px) {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }
  
  .save-config-btn {
    height: 44px;
    padding: 0 24px;
    font-size: 16px;
    font-weight: 500;
    min-width: 160px;
    
    :deep(.el-icon) {
      margin-right: 6px;
      font-size: 16px;
    }
    
    @media (max-width: 768px) {
      width: 100% !important;
      min-width: 100% !important;
      max-width: 100% !important;
      height: 44px !important;
      font-size: 16px !important;
      margin: 0 auto !important;
    }
  }
}

.log-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}
.log-header-buttons {
  display: flex;
  gap: 10px;
  
  @media (min-width: 769px) {
    flex-direction: row;
    align-items: center;
  }
  
  @media (max-width: 768px) {
    flex-direction: column;
    width: 100%;
    gap: 12px;
    align-items: center;
  }
  
  .log-btn {
    @media (min-width: 769px) {
      min-width: 100px;
      height: 32px;
    }
    
    @media (max-width: 768px) {
      width: 100% !important;
      max-width: 100% !important;
      min-width: 100% !important;
      height: 44px !important;
      font-size: 16px !important;
      font-weight: 500 !important;
      margin: 0 auto !important;
    }
  }
}

.live-indicator {
  animation: pulse 2s infinite;
}

.new-log-indicator {
  animation: bounce 1s ease-in-out;
}

@keyframes pulse {
  0% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
  100% {
    opacity: 1;
  }
}

@keyframes bounce {
  0%, 20%, 50%, 80%, 100% {
    transform: translateY(0);
  }
  40% {
    transform: translateY(-3px);
  }
  60% {
    transform: translateY(-2px);
  }
}

.config-form {
  @media (max-width: 768px) {
    width: 100% !important;
    max-width: 100% !important;
    margin: 0 !important;
    padding: 0 !important;
    
    :deep(.el-form-item) {
      margin-bottom: 20px;
      width: 100% !important;
      max-width: 100% !important;
      display: flex;
      flex-direction: column;
      
      .el-form-item__label {
        width: 100% !important;
        max-width: 100% !important;
        text-align: left;
        margin-bottom: 8px;
        padding: 0;
        font-weight: 600;
        color: #1e293b;
        font-size: 0.95rem;
      }
      
      .el-form-item__content {
        width: 100% !important;
        max-width: 100% !important;
        margin-left: 0 !important;
      }
    }
  }
}

.form-item-content {
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  position: relative;
  z-index: 1;
  
  .add-btn {
    width: 100%;
    height: 44px;
    font-size: 16px;
    font-weight: 500;
    box-sizing: border-box;
    margin-top: 8px;
    position: relative;
    z-index: 10;
    pointer-events: auto;
    cursor: pointer;
    
    :deep(.el-icon) {
      margin-right: 6px;
      font-size: 16px;
    }
    
    @media (max-width: 768px) {
      width: 100% !important;
      max-width: 100% !important;
      min-width: 100% !important;
      height: 44px !important;
      font-size: 16px !important;
      margin: 8px auto 0 auto !important;
    }
  }
  
  @media (max-width: 768px) {
    width: 100% !important;
    max-width: 100% !important;
    padding: 0 !important;
    margin: 0 !important;
  }
}

/* 表单行组 - 并排显示 */
.form-row-group {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  width: 100%;
  margin-bottom: 20px;
  
  @media (max-width: 768px) {
    grid-template-columns: 1fr;
    gap: 0;
    margin-bottom: 0;
  }
  
  :deep(.el-form-item) {
    margin-bottom: 0;
    
    @media (max-width: 768px) {
      margin-bottom: 20px;
    }
  }
}

.url-item, .keyword-item {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
  align-items: center;
  width: 100%;
  box-sizing: border-box;
  position: relative;
  z-index: 1;
  
  .config-input {
    flex: 1;
    min-width: 0;
  }
  
  .delete-btn {
    flex-shrink: 0;
    min-width: 100px;
    height: 44px;
    font-size: 16px;
    font-weight: 500;
    position: relative;
    z-index: 10;
    pointer-events: auto;
    cursor: pointer;
    
    :deep(.el-icon) {
      margin-right: 6px;
      font-size: 16px;
    }
    
    @media (max-width: 768px) {
      width: 100% !important;
      min-width: 100% !important;
      max-width: 100% !important;
      height: 44px !important;
      font-size: 16px !important;
      margin: 0 !important;
    }
  }
  
  @media (max-width: 768px) {
    flex-direction: column;
    gap: 12px;
    margin-bottom: 16px;
    padding: 12px;
    background: linear-gradient(135deg, rgba(66, 165, 245, 0.05) 0%, rgba(102, 126, 234, 0.05) 100%);
    border: 1.5px solid rgba(66, 165, 245, 0.2);
    border-radius: 10px;
    align-items: center;
    
    .config-input {
      width: 100% !important;
      max-width: 100% !important;
    }
  }
}
.config-input {
  width: 100%;
  
  :deep(.el-input__wrapper) {
    min-height: 44px;
  }
  
  :deep(.el-input__inner) {
    font-size: 15px;
    padding: 12px 14px;
  }
  
  @media (max-width: 768px) {
    width: 100% !important;
    max-width: 100% !important;
    min-width: 100% !important;
    box-sizing: border-box;
    margin: 0 auto !important;
    
    :deep(.el-input__wrapper) {
      width: 100% !important;
      max-width: 100% !important;
      min-width: 100% !important;
      min-height: 44px !important;
      border-radius: 8px;
      border: 2px solid rgba(66, 165, 245, 0.2);
      background: rgba(255, 255, 255, 0.95);
      box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
      transition: all 0.3s ease;
      
      &:hover {
        border-color: rgba(66, 165, 245, 0.4);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
      }
      
      &.is-focus {
        border-color: #409eff;
        box-shadow: 0 4px 16px rgba(66, 165, 245, 0.2);
      }
    }
    
    :deep(.el-input__inner) {
      font-size: 16px !important;
      padding: 12px 14px;
      width: 100% !important;
      max-width: 100% !important;
      box-sizing: border-box;
    }
  }
}

.config-input-number {
  :deep(.el-input__wrapper) {
    min-height: 44px;
  }
  
  :deep(.el-input__inner) {
    font-size: 15px;
    padding: 12px 14px;
  }
  
  @media (max-width: 768px) {
    width: 100% !important;
    max-width: 100% !important;
    min-width: 100% !important;
    box-sizing: border-box;
    margin: 0 auto !important;
    
    :deep(.el-input__wrapper) {
      width: 100% !important;
      max-width: 100% !important;
      min-width: 100% !important;
      min-height: 44px !important;
      border-radius: 8px;
      border: 2px solid rgba(66, 165, 245, 0.2);
      background: rgba(255, 255, 255, 0.95);
      box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
    }
    
    :deep(.el-input__inner) {
      font-size: 16px !important;
    }
  }
}

.log-container {
  max-height: 400px;
  overflow-y: auto;
  overflow-x: auto;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  padding: 10px;
  background-color: #f5f7fa;
  width: 100%;
  box-sizing: border-box;
  
  @media (max-width: 768px) {
    width: 100% !important;
    max-width: 100% !important;
    padding: 12px;
    word-wrap: break-word;
    word-break: break-all;
    overflow-wrap: break-word;
  }
}

.log-item {
  display: flex;
  gap: 10px;
  margin-bottom: 5px;
  padding: 5px;
  border-radius: 3px;
  font-family: monospace;
  font-size: 12px;
  width: 100%;
  box-sizing: border-box;
  align-items: flex-start;
  
  @media (max-width: 768px) {
    flex-wrap: wrap;
    gap: 6px;
    padding: 8px;
    font-size: 11px;
  }
}

.log-item.info {
  background-color: #e1f3d8;
}

.log-item.warning {
  background-color: #fdf6ec;
}

.log-item.error {
  background-color: #fef0f0;
}

.log-item.success {
  background-color: #e1f3d8;
}

.log-time {
  color: #666;
  min-width: 150px;
  flex-shrink: 0;
  
  @media (max-width: 768px) {
    min-width: auto;
    width: 100%;
    margin-bottom: 4px;
    font-size: 10px;
  }
}

.log-level {
  font-weight: bold;
  min-width: 60px;
  flex-shrink: 0;
  
  @media (max-width: 768px) {
    min-width: auto;
    width: auto;
    margin-right: 8px;
  }
}

.log-message {
  flex: 1;
  word-wrap: break-word;
  word-break: break-all;
  overflow-wrap: break-word;
  
  @media (max-width: 768px) {
    width: 100%;
    flex: none;
  }
}

.no-logs {
  text-align: center;
  color: #999;
  padding: 20px;
}

.form-tip {
  color: #999;
  font-size: 12px;
  margin-top: 5px;
}

.dialog-footer {
  text-align: right;
}

/* 确保按钮可以点击 */
:deep(.el-button) {
  pointer-events: auto !important;
  cursor: pointer !important;
  position: relative;
  z-index: 10;
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

.interval-selector {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  
  .interval-unit-select {
    width: 140px;
    flex-shrink: 0;
    
    :deep(.el-input__wrapper) {
      min-height: 44px;
    }
  }
  
  .interval-number {
    flex: 1;
    min-width: 0;
    max-width: 200px;
    
    :deep(.el-input__wrapper) {
      min-height: 44px;
    }
  }
  
  .interval-tip {
    flex-shrink: 0;
    color: #606266;
    font-size: 14px;
    white-space: nowrap;
    
    @media (min-width: 769px) {
      display: none;
    }
  }
  
  @media (max-width: 768px) {
    flex-direction: column;
    align-items: center;
    gap: 12px;
    width: 100% !important;
    
    .interval-unit-select {
      width: 100% !important;
      max-width: 100% !important;
      min-width: 100% !important;
      
      :deep(.el-input__wrapper) {
        width: 100% !important;
        min-height: 44px !important;
      }
    }
    
    .interval-number {
      width: 100% !important;
      max-width: 100% !important;
      min-width: 100% !important;
      
      :deep(.el-input__wrapper) {
        width: 100% !important;
        min-height: 44px !important;
      }
    }
    
    .interval-tip {
      width: 100% !important;
      margin-top: 8px;
      padding: 8px 12px;
      background: rgba(66, 165, 245, 0.1);
      border-radius: 6px;
      font-size: 14px;
      color: #409eff;
      text-align: center;
      white-space: normal;
    }
  }
}

.schedule-switch-container {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  
  @media (max-width: 768px) {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }
  
  .schedule-tip {
    color: #67c23a;
    font-size: 0.9rem;
    margin-left: 8px;
    
    @media (max-width: 768px) {
      margin-left: 0;
      padding: 8px 12px;
      background: rgba(103, 194, 58, 0.1);
      border-radius: 6px;
      width: 100%;
    }
  }
}

.config-card,
.files-card,
.logs-card {
  @media (max-width: 768px) {
    width: 100% !important;
    max-width: 100% !important;
    margin-left: 0 !important;
    margin-right: 0 !important;
    box-sizing: border-box;
    
    :deep(.el-card__body) {
      padding: 12px !important;
      width: 100% !important;
      max-width: 100% !important;
      box-sizing: border-box;
      overflow-x: hidden;
    }
    
    :deep(.el-card__header) {
      padding: 12px !important;
      width: 100% !important;
      max-width: 100% !important;
      box-sizing: border-box;
    }
  }
}

.files-table {
  @media (max-width: 768px) {
    width: 100% !important;
    max-width: 100% !important;
    
    :deep(table) {
      width: 100% !important;
      max-width: 100% !important;
    }
    
    :deep(.el-table__body-wrapper) {
      overflow-x: auto;
    }
  }
}
</style>
