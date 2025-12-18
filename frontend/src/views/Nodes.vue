<template>
  <div class="list-container nodes-container">
    <!-- 节点统计 -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-number">{{ nodeStats.total }}</div>
        <div class="stat-label">总节点数</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ nodeStats.online }}</div>
        <div class="stat-label">在线节点</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ nodeStats.regions }}</div>
        <div class="stat-label">地区数量</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ nodeStats.types }}</div>
        <div class="stat-label">节点类型</div>
      </div>
    </div>

    <!-- 节点列表 -->
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <i class="el-icon-connection"></i>
          节点列表
          <div class="header-actions">
            <el-select v-model="filterRegion" placeholder="选择地区" clearable>
              <el-option
                v-for="region in regions"
                :key="region"
                :label="region"
                :value="region"
              />
            </el-select>
            <el-select v-model="filterType" placeholder="选择类型" clearable>
              <el-option
                v-for="type in nodeTypes"
                :key="type"
                :label="type"
                :value="type"
              />
            </el-select>
            <el-button 
              type="primary" 
              size="small" 
              @click="refreshNodes"
              :loading="loading"
            >
              <i class="el-icon-refresh"></i>
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <el-table 
        :data="filteredNodes" 
        v-loading="loading"
        style="width: 100%"
      >
        <el-table-column prop="name" label="节点名称" min-width="150">
          <template #default="{ row }">
            <div class="node-name">
              <i :class="getNodeIcon(row.type)"></i>
              <span>{{ row.name }}</span>
              <el-tag 
                v-if="row.is_recommended" 
                type="success" 
                size="small"
              >
                推荐
              </el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="region" label="地区" width="120">
          <template #default="{ row }">
            <el-tag :type="getRegionColor(row.region)">
              {{ row.region || '未知' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="type" label="类型" width="120">
          <template #default="{ row }">
            <el-tag :type="getTypeColor(row.type)">
              {{ row.type || '未知' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="延迟" width="120">
          <template #default="{ row }">
            <div class="latency-cell">
              <span v-if="row.latency > 0" :class="getLatencyClass(row.latency)">
                {{ row.latency }}ms
              </span>
              <span v-else style="color: #909399">未测试</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="最后测试" width="150">
          <template #default="{ row }">
            <span v-if="row.last_test" class="last-test-time">
              {{ formatLastTestTime(row.last_test) }}
            </span>
            <span v-else style="color: #909399">从未测试</span>
          </template>
        </el-table-column>


        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button 
              size="small" 
              type="primary" 
              @click="testNode(row)" 
              :loading="row.testing"
            >
              测试
            </el-button>
            <el-button 
              size="small" 
              @click="viewNodeDetail(row)"
            >
              详情
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 空状态 -->
      <el-empty 
        v-if="!loading && filteredNodes.length === 0" 
        description="暂无节点信息"
      >
        <el-button type="primary" @click="refreshNodes">
          刷新节点列表
        </el-button>
      </el-empty>
    </el-card>

    <!-- 节点详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="节点详情"
      width="600px"
    >
      <div class="node-detail" v-if="selectedNode">
        <div class="detail-item">
          <span class="label">节点名称：</span>
          <span class="value">{{ selectedNode.name }}</span>
        </div>
        <div class="detail-item">
          <span class="label">地区：</span>
          <span class="value">{{ selectedNode.region }}</span>
        </div>
        <div class="detail-item">
          <span class="label">类型：</span>
          <span class="value">{{ selectedNode.type }}</span>
        </div>
        <div class="detail-item">
          <span class="label">状态：</span>
          <span class="value">
            <el-tag :type="selectedNode.status === 'online' ? 'success' : 'danger'">
              {{ selectedNode.status === 'online' ? '在线' : '离线' }}
            </el-tag>
          </span>
        </div>
        <div class="detail-item">
          <span class="label">在线时间：</span>
          <span class="value">{{ formatUptime(selectedNode.uptime) }}</span>
        </div>
        <div class="detail-item">
          <span class="label">延迟：</span>
          <span class="value">
            <span v-if="selectedNode.latency > 0" :class="getLatencyClass(selectedNode.latency)">
              {{ selectedNode.latency }}ms
            </span>
            <span v-else style="color: #909399">未测试</span>
          </span>
        </div>
        <div class="detail-item">
          <span class="label">最后测试：</span>
          <span class="value">
            <span v-if="selectedNode.last_test">
              {{ formatLastTestTime(selectedNode.last_test) }}
            </span>
            <span v-else style="color: #909399">从未测试</span>
          </span>
        </div>
        <div class="detail-item">
          <span class="label">描述：</span>
          <span class="value">{{ selectedNode.description || '暂无描述' }}</span>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { nodeAPI } from '@/utils/api'
import '@/styles/list-common.scss'

export default {
  name: 'Nodes',
  setup() {
    const loading = ref(false)
    const nodes = ref([])
    const filterRegion = ref('')
    const filterType = ref('')
    const detailDialogVisible = ref(false)
    const selectedNode = ref(null)

    const nodeStats = reactive({
      total: 0,
      online: 0,
      regions: 0,
      types: 0
    })

    // 过滤后的节点列表
    const filteredNodes = computed(() => {
      let result = nodes.value

      if (filterRegion.value) {
        result = result.filter(node => node.region === filterRegion.value)
      }

      if (filterType.value) {
        result = result.filter(node => node.type === filterType.value)
      }

      return result
    })

    // 获取地区列表
    const regions = computed(() => {
      const regionList = nodes.value
        .map(node => node.region)
        .filter(region => region && region.trim() !== '')
      return [...new Set(regionList)].sort()
    })

    // 获取节点类型列表
    const nodeTypes = computed(() => {
      const typeList = nodes.value
        .map(node => node.type)
        .filter(type => type && type.trim() !== '')
      return [...new Set(typeList)].sort()
    })

    // 获取节点列表 - 从数据库Clash配置获取真实数据
    const fetchNodes = async () => {
      loading.value = true
      try {
        const response = await nodeAPI.getNodes()
        // 处理API响应数据
        if (response.data && response.data.success && response.data.data) {
          // 后端返回格式: {success: true, data: [...]}
          if (Array.isArray(response.data.data)) {
            nodes.value = response.data.data.map(node => ({
              ...node,
              testing: false
            }))
          } else if (response.data.data.nodes && Array.isArray(response.data.data.nodes)) {
            nodes.value = response.data.data.nodes.map(node => ({
              ...node,
              testing: false
            }))
          } else {
            nodes.value = []
          }
        } else if (response.data && Array.isArray(response.data)) {
          nodes.value = response.data.map(node => ({
            ...node,
            testing: false
          }))
        } else if (response.data && response.data.nodes && Array.isArray(response.data.nodes)) {
          nodes.value = response.data.nodes.map(node => ({
            ...node,
            testing: false
          }))
        } else {
          nodes.value = []
        }
        
        // 计算统计数据
        updateNodeStats()
      } catch (error) {
        const errorMsg = error.response?.data?.message || error.message || '获取节点列表失败'
        ElMessage.error(`获取节点列表失败: ${errorMsg}`)
        console.error('获取节点列表错误:', error)
        nodes.value = []
      } finally {
        loading.value = false
      }
    }

    // 更新节点统计
    const updateNodeStats = () => {
      nodeStats.total = nodes.value.length
      // 支持多种状态格式：'online', 'Online', 'ONLINE'
      nodeStats.online = nodes.value.filter(n => {
        const status = (n.status || '').toLowerCase()
        return status === 'online'
      }).length
      nodeStats.regions = regions.value.length
      nodeStats.types = nodeTypes.value.length
    }

    // 刷新节点列表
    const refreshNodes = () => {
      fetchNodes()
    }

    // 获取测速监控状态
    const fetchSpeedMonitorStatus = async () => {
      try {
        // 这个API需要管理员权限，普通用户可能无法访问
        // 暂时注释掉，避免403错误
        // const response = await nodeAPI.getSpeedMonitorStatus()
        // // 处理API响应数据
        // if (response.data && response.data.data) {
        //   speedMonitorStatus.value = response.data.data
        // } else if (response.data) {
        //   speedMonitorStatus.value = response.data
        // }
      } catch (error) {
        // 静默处理错误
      }
    }

    // 查看节点详情
    const viewNodeDetail = (node) => {
      selectedNode.value = node
      detailDialogVisible.value = true
    }

    // 获取节点图标
    const getNodeIcon = (type) => {
      const icons = {
        ssr: 'el-icon-connection',
        ss: 'el-icon-connection',
        v2ray: 'el-icon-connection',
        vmess: 'el-icon-connection',
        trojan: 'el-icon-connection',
        vless: 'el-icon-connection',
        hysteria: 'el-icon-connection',
        hysteria2: 'el-icon-connection',
        tuic: 'el-icon-connection'
      }
      return icons[type] || 'el-icon-connection'
    }

    // 获取地区颜色
    const getRegionColor = (region) => {
      const colors = {
        '香港': 'success',
        '新加坡': 'warning',
        '日本': 'primary',
        '美国': 'info',
        '韩国': 'success'
      }
      return colors[region] || 'info'
    }

    // 获取类型颜色
    const getTypeColor = (type) => {
      const colors = {
        ssr: 'success',
        ss: 'success',
        v2ray: 'primary',
        vmess: 'primary',
        trojan: 'warning',
        vless: 'info',
        hysteria: 'danger',
        hysteria2: 'danger',
        tuic: 'warning'
      }
      return colors[type] || 'info'
    }


    // 格式化在线时间
    const formatUptime = (uptime) => {
      const days = Math.floor(uptime / 86400)
      const hours = Math.floor((uptime % 86400) / 3600)
      const minutes = Math.floor((uptime % 3600) / 60)
      
      if (days > 0) return `${days}天${hours}小时`
      if (hours > 0) return `${hours}小时${minutes}分钟`
      return `${minutes}分钟`
    }

    // 格式化最后测速时间
    const formatLastTestTime = (lastTestTime) => {
      if (!lastTestTime) return '从未测试'
      
      const now = new Date()
      const testTime = new Date(lastTestTime)
      const diffMs = now - testTime
      const diffMinutes = Math.floor(diffMs / 60000)
      
      if (diffMinutes < 1) return '刚刚'
      if (diffMinutes < 60) return `${diffMinutes}分钟前`
      
      const diffHours = Math.floor(diffMinutes / 60)
      if (diffHours < 24) return `${diffHours}小时前`
      
      const diffDays = Math.floor(diffHours / 24)
      if (diffDays < 7) return `${diffDays}天前`
      
      return testTime.toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      })
    }

    // 获取状态类型
    const getStatusType = (status) => {
      const statusMap = {
        online: 'success',
        offline: 'danger',
        timeout: 'warning'
      }
      return statusMap[status?.toLowerCase()] || 'info'
    }

    // 获取状态文本
    const getStatusText = (status) => {
      const statusMap = {
        online: '在线',
        offline: '离线',
        timeout: '超时'
      }
      return statusMap[status?.toLowerCase()] || status || '未知'
    }

    // 获取延迟颜色类
    const getLatencyClass = (latency) => {
      if (latency < 100) return 'latency-excellent'
      if (latency < 200) return 'latency-good'
      if (latency < 300) return 'latency-medium'
      return 'latency-poor'
    }

    // 测试单个节点
    const testNode = async (node) => {
      node.testing = true
      try {
        const response = await nodeAPI.testNode(node.id)
        if (response.data && response.data.success) {
          const result = response.data.data
          ElMessage.success(`测试完成: ${getStatusText(result.status)}, 延迟: ${result.latency > 0 ? result.latency + 'ms' : '超时'}`)
          // 更新节点信息
          node.status = result.status
          node.latency = result.latency
          node.last_test = result.tested_at
          // 更新统计
          updateNodeStats()
        } else {
          ElMessage.error(response.data?.message || '测试失败')
        }
      } catch (error) {
        ElMessage.error('测试失败: ' + (error.response?.data?.message || error.message))
      } finally {
        node.testing = false
      }
    }



    onMounted(() => {
      fetchNodes()
    })

    return {
      loading,
      nodes,
      filterRegion,
      filterType,
      detailDialogVisible,
      selectedNode,
      nodeStats,
      filteredNodes,
      regions,
      nodeTypes,
      fetchNodes,
      refreshNodes,
      viewNodeDetail,
      testNode,
      getNodeIcon,
      getRegionColor,
      getTypeColor,
      getStatusType,
      getStatusText,
      getLatencyClass,
      formatUptime,
      formatLastTestTime
    }
  }
}
</script>

<style scoped lang="scss">
.nodes-container {
  /* 使用 list-common.scss 的统一样式 */
  /* padding, max-width, margin 由 list-common.scss 统一管理 */
  padding: 0;
  max-width: none;
  margin: 0;
  width: 100%;
}

.page-header {
  margin-bottom: 2rem;
  text-align: left;
}

.page-header h1 {
  color: #1677ff;
  font-size: 2rem;
  margin-bottom: 0.5rem;
}

.page-header :is(p) {
  color: #666;
  font-size: 1rem;
}

.stats-card {
  margin-bottom: 2rem;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

.speed-status-card {
  margin-bottom: 2rem;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

.speed-status-content {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  padding: 1rem 0;
}

.status-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 0;
  border-bottom: 1px solid #f0f0f0;
}

.status-item:last-child {
  border-bottom: none;
}

.status-item .label {
  font-weight: 500;
  color: #666;
}

.status-item .value {
  color: #333;
  font-weight: 600;
}

.stats-content {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 2rem;
  padding: 1rem 0;
}

.stat-item {
  text-align: center;
}

.stat-number {
  font-size: 2.5rem;
  font-weight: bold;
  color: #1677ff;
  margin-bottom: 0.5rem;
}

.stat-label {
  color: #666;
  font-size: 0.9rem;
}

.nodes-card {
  margin-bottom: 2rem;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  font-weight: 600;
}

.header-actions {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.node-name {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.node-name i {
  font-size: 1.2rem;
  color: #1677ff;
}

.latency-cell {
  font-weight: 500;
}

.latency-excellent {
  color: #52c41a;
}

.latency-good {
  color: #1890ff;
}

.latency-medium {
  color: #faad14;
}

.latency-poor {
  color: #ff4d4f;
}

.last-test-time {
  color: #666;
  font-size: 0.875rem;
}

.speed-text {
  font-family: 'Courier New', monospace;
  color: #1677ff;
  font-weight: 500;
}

.node-detail {
  padding: 1rem 0;
}

.detail-item {
  display: flex;
  margin-bottom: 1rem;
  padding: 0.5rem 0;
  border-bottom: 1px solid #f0f0f0;
}

.detail-item:last-child {
  border-bottom: none;
}

.detail-item .label {
  width: 120px;
  font-weight: 500;
  color: #666;
}

.detail-item .value {
  flex: 1;
  color: #333;
}

@media (max-width: 768px) {
  .nodes-container {
    padding: 10px;
  }
  
  .stats-content {
    grid-template-columns: repeat(2, 1fr);
    gap: 1rem;
  }
  
  .stat-number {
    font-size: 2rem;
  }
  
  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
  }
  
  .header-actions {
    width: 100%;
    flex-wrap: wrap;
    flex-direction: column;
    gap: 10px;
    
    .el-select {
      width: 100% !important;
    }
    
    .el-button {
      width: 100%;
      min-height: 44px;
      font-size: 16px;
    }
  }
  
  /* 表格在手机端优化 */
  :deep(.el-table) {
    font-size: 0.875rem;
    
    .el-table__cell {
      padding: 8px 4px;
    }
  }
  
  /* 对话框优化 */
  :deep(.el-dialog) {
    width: 90% !important;
    margin: 5vh auto !important;
    max-height: 90vh;
    overflow-y: auto;
  }
  
  :deep(.el-dialog__body) {
    padding: 15px !important;
    max-height: calc(90vh - 120px);
    overflow-y: auto;
  }
  
  .node-detail {
    .detail-item {
      flex-direction: column;
      gap: 0.5rem;
      
      .label {
        width: auto;
        font-size: 0.875rem;
      }
      
      .value {
        font-size: 0.875rem;
      }
    }
  }
  
  .detail-item {
    flex-direction: column;
    gap: 0.5rem;
  }
  
  .detail-item .label {
    width: auto;
  }
}
</style> 