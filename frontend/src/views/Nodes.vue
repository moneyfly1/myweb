<template>
  <div class="list-container nodes-container">
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
      <div class="table-wrapper">
        <el-table 
          ref="nodeTableRef"
          :data="paginatedNodes" 
          v-loading="loading"
          style="width: 100%"
          class="desktop-table"
          border
          stripe
          @header-dragend="handleNodeColumnResize"
        >
          <el-table-column prop="name" label="节点名称" :min-width="columnWidths.name" resizable>
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
          <el-table-column prop="region" label="地区" :width="columnWidths.region" resizable>
            <template #default="{ row }">
              <el-tag :type="getRegionColor(row.region)">
                {{ row.region || '未知' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="type" label="类型" :width="columnWidths.type" resizable>
            <template #default="{ row }">
              <el-tag :type="getTypeColor(row.type)">
                {{ row.type || '未知' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" :width="columnWidths.status" resizable>
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)" size="small">
                {{ getStatusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
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
      <el-empty 
        v-if="!loading && filteredNodes.length === 0" 
        description="暂无节点信息"
      >
        <el-button type="primary" @click="refreshNodes">
          刷新节点列表
        </el-button>
      </el-empty>
    </el-card>
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
const NODES_TABLE_STORAGE_KEY = 'user_nodes_table_settings'
export default {
  name: 'Nodes',
  setup() {
    const loading = ref(false)
    const nodes = ref([])
    const nodeTableRef = ref(null)
    const columnWidths = reactive({
      name: 200,
      region: 120,
      type: 120,
      status: 120
    })
    const loadNodeTableSettings = () => {
      try {
        const saved = localStorage.getItem(NODES_TABLE_STORAGE_KEY)
        if (saved) {
          const s = JSON.parse(saved)
          if (s.columnWidths) Object.assign(columnWidths, s.columnWidths)
        }
      } catch (e) {
        console.warn('加载节点表设置失败:', e)
      }
    }
    const saveNodeTableSettings = () => {
      try {
        localStorage.setItem(NODES_TABLE_STORAGE_KEY, JSON.stringify({ columnWidths: { ...columnWidths } }))
      } catch (e) {
        console.warn('保存节点表设置失败:', e)
      }
    }
    const NODE_COLUMN_KEYS = ['name', 'region', 'type', 'status']
    let nodeResizeTimer = null
    const handleNodeColumnResize = () => {
      if (nodeResizeTimer) clearTimeout(nodeResizeTimer)
      nodeResizeTimer = setTimeout(() => {
        if (nodeTableRef.value && nodeTableRef.value.$el) {
          const cells = nodeTableRef.value.$el.querySelectorAll('.el-table__header-wrapper thead th')
          cells.forEach((cell, index) => {
            if (NODE_COLUMN_KEYS[index] && cell.offsetWidth > 0) columnWidths[NODE_COLUMN_KEYS[index]] = cell.offsetWidth
          })
          saveNodeTableSettings()
        }
      }, 300)
    }
    const filterRegion = ref('')
    const filterType = ref('')
    const isMobile = ref(window.innerWidth <= 768)
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
    const paginatedNodes = computed(() => {
      const start = (pagination.page - 1) * pagination.size
      const end = start + pagination.size
      return filteredNodes.value.slice(start, end)
    })
    watch([filterRegion, filterType], () => {
      pagination.page = 1
    })
    const handleSizeChange = (size) => {
      pagination.size = size
      pagination.page = 1 // 重置到第一页
    }
    const handlePageChange = (page) => {
      pagination.page = page
      if (typeof window !== 'undefined') {
        window.scrollTo({ top: 0, behavior: 'smooth' })
      }
    }
    const regions = computed(() => {
      const regionList = nodes.value
        .map(node => node.region)
        .filter(region => region && region.trim() !== '')
      return [...new Set(regionList)].sort()
    })
    const nodeTypes = computed(() => {
      const typeList = nodes.value
        .map(node => node.type)
        .filter(type => type && type.trim() !== '')
      return [...new Set(typeList)].sort()
    })
    const fetchNodes = async () => {
      loading.value = true
      try {
        const response = await nodeAPI.getNodes()
        if (response && response.data) {
          if (response.data.success && response.data.data) {
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
            nodes.value = []
          }
        } else {
          console.error('响应格式错误:', response)
          nodes.value = []
        }
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
    const updateNodeStats = () => {
      nodeStats.total = nodes.value.length
      nodeStats.online = nodes.value.filter(n => {
        const status = (n.status || '').toLowerCase()
        return status === 'online'
      }).length
      nodeStats.regions = regions.value.length
      nodeStats.types = nodeTypes.value.length
    }
    const refreshNodes = () => {
      fetchNodes()
    }
    const fetchSpeedMonitorStatus = async () => {
      try {
      } catch (error) {
      }
    }
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
    const getStatusType = (status) => {
      const statusMap = {
        online: 'success',
        offline: 'danger',
        timeout: 'warning',
        inactive: 'info'
      }
      return statusMap[status?.toLowerCase()] || 'info'
    }
    const getStatusText = (status) => {
      const statusMap = {
        online: '在线',
        offline: '离线',
        timeout: '超时',
        inactive: '未激活'
      }
      return statusMap[status?.toLowerCase()] || status || '未知'
    }
    const handleResize = () => {
      if (typeof window !== 'undefined') {
        isMobile.value = window.innerWidth <= 768
      }
    }
    onMounted(() => {
      loadNodeTableSettings()
      fetchNodes()
      if (typeof window !== 'undefined') {
        window.addEventListener('resize', handleResize)
      }
    })
    onUnmounted(() => {
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
      isMobile,
      nodeTableRef,
      columnWidths,
      handleNodeColumnResize
    }
  }
}
</script>
<style scoped lang="scss">
.nodes-container {
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
.table-wrapper {
  display: block;
  @media (max-width: 768px) {
    display: none;
  }
}
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