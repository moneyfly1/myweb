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

      <!-- 桌面端表格 -->
      <div class="table-wrapper">
        <el-table 
          :data="paginatedNodes" 
          v-loading="loading"
          style="width: 100%"
          class="desktop-table"
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

        <!-- 移除延迟、最后测试和操作列，用户端不需要这些功能 -->
        </el-table>
        
        <!-- 分页组件 -->
        <div class="pagination-wrapper" v-if="filteredNodes.length > 0">
          <el-pagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.size"
            :page-sizes="[10, 20, 50, 100]"
            :total="filteredNodes.length"
            :layout="isMobile ? 'prev, pager, next' : 'total, sizes, prev, pager, next, jumper'"
            @size-change="handleSizeChange"
            @current-change="handlePageChange"
          />
        </div>
      </div>

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

    <!-- 移动端卡片式列表 -->
    <div class="mobile-card-list" v-if="paginatedNodes.length > 0">
      <div 
        v-for="node in paginatedNodes" 
        :key="node.id"
        class="mobile-node-card"
      >
        <div class="card-row">
          <span class="label">节点名称</span>
          <span class="value">
            <i :class="getNodeIcon(node.type)"></i>
            {{ node.name }}
            <el-tag 
              v-if="node.is_recommended" 
              type="success" 
              size="small"
              style="margin-left: 8px;"
            >
              推荐
            </el-tag>
          </span>
        </div>
        <div class="card-row">
          <span class="label">地区</span>
          <span class="value">
            <el-tag :type="getRegionColor(node.region)" size="small">
              {{ node.region || '未知' }}
            </el-tag>
          </span>
        </div>
        <div class="card-row">
          <span class="label">类型</span>
          <span class="value">
            <el-tag :type="getTypeColor(node.type)" size="small">
              {{ node.type || '未知' }}
            </el-tag>
          </span>
        </div>
        <div class="card-row">
          <span class="label">状态</span>
          <span class="value">
            <el-tag :type="getStatusType(node.status)" size="small">
              {{ getStatusText(node.status) }}
            </el-tag>
          </span>
        </div>
      </div>
      
      <!-- 移动端分页 -->
      <div class="mobile-pagination" v-if="filteredNodes.length > 0">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.size"
          :page-sizes="[10, 20, 50]"
          :total="filteredNodes.length"
          layout="prev, pager, next"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </div>
  </div>
</template>

<script>
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
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
    const isMobile = ref(window.innerWidth <= 768)

    // 分页配置
    const pagination = reactive({
      page: 1,
      size: 20,
      total: 0
    })

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

    // 分页后的节点列表
    const paginatedNodes = computed(() => {
      const start = (pagination.page - 1) * pagination.size
      const end = start + pagination.size
      return filteredNodes.value.slice(start, end)
    })

    // 监听过滤条件变化，重置到第一页
    watch([filterRegion, filterType], () => {
      pagination.page = 1
    })

    // 处理分页大小变化
    const handleSizeChange = (size) => {
      pagination.size = size
      pagination.page = 1 // 重置到第一页
    }

    // 处理页码变化
    const handlePageChange = (page) => {
      pagination.page = page
      // 滚动到顶部
      if (typeof window !== 'undefined') {
        window.scrollTo({ top: 0, behavior: 'smooth' })
      }
    }

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
        console.log('节点列表API响应:', response)
        
        // 处理API响应数据
        if (response && response.data) {
          if (response.data.success && response.data.data) {
            // 后端返回格式: {success: true, data: [...]}
            if (Array.isArray(response.data.data)) {
              nodes.value = response.data.data.map(node => ({
                ...node,
                testing: false
              }))
              console.log('加载的节点数量:', nodes.value.length)
            } else if (response.data.data.nodes && Array.isArray(response.data.data.nodes)) {
              nodes.value = response.data.data.nodes.map(node => ({
                ...node,
                testing: false
              }))
            } else {
              console.warn('响应数据格式不正确:', response.data.data)
              nodes.value = []
            }
          } else if (Array.isArray(response.data)) {
            nodes.value = response.data.map(node => ({
              ...node,
              testing: false
            }))
          } else if (response.data.nodes && Array.isArray(response.data.nodes)) {
            nodes.value = response.data.nodes.map(node => ({
              ...node,
              testing: false
            }))
          } else {
            console.warn('未识别的响应格式:', response.data)
            nodes.value = []
          }
        } else {
          console.error('响应格式错误:', response)
          nodes.value = []
        }
        
        // 计算统计数据
        updateNodeStats()
      } catch (error) {
        const errorMsg = error.response?.data?.message || error.message || '获取节点列表失败'
        ElMessage.error(`获取节点列表失败: ${errorMsg}`)
        console.error('获取节点列表错误:', error)
        console.error('错误详情:', error.response)
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

    // 移除查看节点详情功能（用户端不需要）

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


    // 获取状态类型
    const getStatusType = (status) => {
      const statusMap = {
        online: 'success',
        offline: 'danger',
        timeout: 'warning',
        inactive: 'info'
      }
      return statusMap[status?.toLowerCase()] || 'info'
    }

    // 获取状态文本
    const getStatusText = (status) => {
      const statusMap = {
        online: '在线',
        offline: '离线',
        timeout: '超时',
        inactive: '未激活'
      }
      return statusMap[status?.toLowerCase()] || status || '未知'
    }

    // 移除测试节点功能（用户端不需要）



    // 监听窗口大小变化
    const handleResize = () => {
      if (typeof window !== 'undefined') {
        isMobile.value = window.innerWidth <= 768
      }
    }

    onMounted(() => {
      fetchNodes()
      // 初始化窗口大小
      if (typeof window !== 'undefined') {
        window.addEventListener('resize', handleResize)
      }
    })

    onUnmounted(() => {
      // 清理窗口大小监听
      if (typeof window !== 'undefined') {
        window.removeEventListener('resize', handleResize)
      }
    })

    return {
      loading,
      nodes,
      filterRegion,
      filterType,
      nodeStats,
      filteredNodes,
      paginatedNodes,
      pagination,
      regions,
      nodeTypes,
      fetchNodes,
      refreshNodes,
      handleSizeChange,
      handlePageChange,
      getNodeIcon,
      getRegionColor,
      getTypeColor,
      getStatusType,
      getStatusText,
      isMobile
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

.node-name :is(i) {
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

/* 桌面端表格显示 */
.table-wrapper {
  display: block;
  
  @media (max-width: 768px) {
    display: none;
  }
}

/* 分页样式 */
.pagination-wrapper {
  margin-top: 20px;
  display: flex;
  justify-content: center;
  padding: 20px 0;
}

.mobile-pagination {
  margin-top: 20px;
  padding: 20px 0;
  display: flex;
  justify-content: center;
}

/* 移动端卡片列表显示 */
.mobile-card-list {
  display: none;
  
  @media (max-width: 768px) {
    display: block;
  }
}

.mobile-node-card {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  border-left: 4px solid #409eff;
  
  .card-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 0;
    border-bottom: 1px solid #f0f0f0;
    
    &:last-child {
      border-bottom: none;
    }
    
    .label {
      color: #909399;
      font-size: 14px;
      font-weight: 500;
      min-width: 80px;
    }
    
    .value {
      flex: 1;
      text-align: right;
      color: #303133;
      font-size: 14px;
      display: flex;
      align-items: center;
      justify-content: flex-end;
      gap: 8px;
      flex-wrap: wrap;
      
      :is(i) {
        font-size: 16px;
        color: #409eff;
      }
    }
  }
}

@media (max-width: 768px) {
  .nodes-container {
    padding: 10px;
  }
  
  .stats-row {
    grid-template-columns: repeat(2, 1fr) !important;
    gap: 10px;
    margin-bottom: 12px;
  }
  
  .stat-card {
    padding: 12px;
    
    .stat-number {
      font-size: 1.5rem;
    }
    
    .stat-label {
      font-size: 12px;
    }
  }
  
  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
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
  
  .list-card {
    margin-bottom: 12px;
    
    :deep(.el-card__header) {
      padding: 12px;
    }
    
    :deep(.el-card__body) {
      padding: 12px;
    }
  }
}
</style> 