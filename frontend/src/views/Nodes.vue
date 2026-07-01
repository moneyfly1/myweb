<template>
  <div class="list-container nodes-container">
    <div class="breadcrumb">首页 / 节点列表</div>
    <div class="page-header">
      <div class="page-title">
        <h1>节点列表</h1>
      </div>
      <div class="actions">
        <el-button @click="refreshNodes" :loading="loading">
          刷新
        </el-button>
      </div>
    </div>

    <div class="card list-filter-card nodes-filter-card">
      <div class="card-body nodes-filter-body">
        <el-form :inline="true" class="nodes-filter-form list-filter-form">
          <el-form-item label="搜索">
            <el-input
              v-model="filterKeyword"
              placeholder="节点名称、地区、类型、说明"
              clearable
              class="nodes-keyword-input"
              @keyup.enter="applyFilters"
              @clear="applyFilters"
            />
          </el-form-item>
          <el-form-item label="地区筛选">
            <el-select v-model="filterRegion" placeholder="全部地区" clearable class="nodes-filter-select">
              <el-option
                v-for="region in regions"
                :key="region"
                :label="region"
                :value="region"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="类型筛选">
            <el-select v-model="filterType" placeholder="全部类型" clearable class="nodes-filter-select">
              <el-option
                v-for="type in nodeTypes"
                :key="type"
                :label="type"
                :value="type"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="来源筛选">
            <el-select v-model="filterSource" placeholder="全部来源" clearable class="nodes-filter-select">
              <el-option label="后台配置" value="manual" />
              <el-option label="自动采集" value="collect" />
            </el-select>
          </el-form-item>
          <el-form-item class="nodes-filter-actions">
            <el-button type="primary" @click="applyFilters">搜索</el-button>
            <el-button :disabled="!hasActiveFilters" @click="resetFilters">重置</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <div class="card">
      <div class="card-header">
        <h2 class="card-title">
            <el-icon class="header-icon"><Connection /></el-icon>
          可用节点
        </h2>
      </div>
      <div class="table-wrap">
        <ResponsiveDataView
          :data="paginatedNodes"
          :fields="mobileNodeFields"
          :loading="loading"
          title-field="name"
          empty-title="暂无节点信息"
          empty-description="可刷新后重试"
        >
          <template #table>
            <div class="table-wrapper">
              <el-table
                ref="nodeTableRef"
                :data="paginatedNodes"
                v-loading="loading"
                class="desktop-table nodes-table"
                border
                stripe
                @header-dragend="handleNodeColumnResize"
              >
                <template #empty>
                  <EmptyState
                    title="暂无节点信息"
                    description="可刷新后重试"
                    action-text="刷新节点列表"
                    :loading="loading"
                    @action="refreshNodes"
                  />
                </template>
                <el-table-column prop="name" label="节点名称" :min-width="columnWidths.name" resizable>
                  <template #default="{ row }">
                    <div class="node-name">
                      <el-icon class="node-type-icon"><Connection /></el-icon>
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
                <el-table-column label="说明" min-width="180" resizable>
                  <template #default="{ row }">
                    {{ row.description || row.remark || row.info || '适合日常使用' }}
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </template>
          <template #header="{ item }">
            <div class="mobile-node-header">
              <span>
                <el-icon class="node-type-icon"><Connection /></el-icon>
                {{ item.name }}
              </span>
              <el-tag v-if="item.is_recommended" type="success" size="small">
                推荐
              </el-tag>
            </div>
          </template>
          <template #empty>
            <EmptyState
              class="mobile-node-empty"
              title="暂无节点信息"
              description="可刷新后重试"
              action-text="刷新节点列表"
              :loading="loading"
              @action="refreshNodes"
            />
          </template>
        </ResponsiveDataView>
      </div>
    </div>

    <PaginationBar
      v-if="filteredNodes.length > 0"
      v-model:current-page="pagination.page"
      v-model:page-size="pagination.size"
      :total="filteredNodes.length"
      :page-sizes="[10, 20, 50, 100]"
      @size-change="handleSizeChange"
      @current-change="handlePageChange"
    />
  </div>
</template>
<script>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { Connection } from '@element-plus/icons-vue'
import { nodeAPI } from '@/utils/api'
import { usePersistentTableColumns } from '@/composables/usePersistentTableColumns'
import EmptyState from '@/components/EmptyState.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
const NODES_TABLE_STORAGE_KEY = 'user_nodes_table_settings'
export default {
  name: 'Nodes',
  components: {
    EmptyState,
    PaginationBar,
    ResponsiveDataView,
    Connection
  },
  setup() {
    const loading = ref(false)
    const nodes = ref([])
    const nodeTableRef = ref(null)
    const NODE_COLUMN_KEYS = ['name', 'region', 'type', 'source', 'status']
    const { columnWidths, handleColumnResize: handleNodeColumnResize } = usePersistentTableColumns(
      NODES_TABLE_STORAGE_KEY,
      {
      name: 200,
      region: 120,
      type: 120,
      source: 100,
      status: 120
      },
      NODE_COLUMN_KEYS
    )
    const filterRegion = ref('')
    const filterType = ref('')
    const filterSource = ref('')
    const filterKeyword = ref('')
    const hasActiveFilters = computed(() => Boolean(filterKeyword.value || filterRegion.value || filterType.value || filterSource.value))
    const pagination = reactive({
      page: 1,
      size: 10,
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
      const keyword = String(filterKeyword.value || '').trim().toLowerCase()
      if (keyword) {
        result = result.filter(node => [
          node.name,
          node.region,
          node.type,
          node.status,
          node.description,
          node.remark,
          node.info
        ].filter(Boolean).join(' ').toLowerCase().includes(keyword))
      }
      if (filterRegion.value) {
        result = result.filter(node => node.region === filterRegion.value)
      }
      if (filterType.value) {
        result = result.filter(node => node.type === filterType.value)
      }
      if (filterSource.value) {
        if (filterSource.value === 'manual') {
          result = result.filter(node => node.is_manual === true)
        } else if (filterSource.value === 'collect') {
          result = result.filter(node => node.is_manual === false || node.is_manual === undefined)
        }
      }
      return result
    })
    const paginatedNodes = computed(() => {
      const start = (pagination.page - 1) * pagination.size
      const end = start + pagination.size
      return filteredNodes.value.slice(start, end)
    })
    const mobileNodeFields = computed(() => [
      {
        key: 'region',
        label: '地区',
        type: 'tag',
        tagType: value => getRegionColor(value),
        formatter: value => value || '未知'
      },
      {
        key: 'type',
        label: '类型',
        type: 'tag',
        tagType: value => getTypeColor(value),
        formatter: value => value || '未知'
      },
      {
        key: 'status',
        label: '状态',
        type: 'tag',
        tagType: value => getStatusType(value),
        formatter: value => getStatusText(value)
      },
      {
        key: 'description',
        label: '说明',
        formatter: (_value, item) => item.description || item.remark || item.info || '适合日常使用'
      }
    ])
    watch([filterRegion, filterType, filterSource], () => {
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
    const resetFilters = () => {
      filterKeyword.value = ''
      filterRegion.value = ''
      filterType.value = ''
      filterSource.value = ''
      pagination.page = 1
    }
    const applyFilters = () => {
      pagination.page = 1
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
        // 代理协议
        vmess: 'primary',
        vless: 'info',
        trojan: 'warning',
        ss: 'success',
        ssr: 'success',
        hysteria: 'danger',
        hysteria2: 'danger',
        tuic: 'warning',
        naive: 'primary',
        anytls: 'info',
        // SOCKS 代理
        socks: 'warning',
        socks5: 'warning',
        // HTTP 代理
        http: 'info',
        https: 'info',
        // VPN 协议
        wg: 'success',
        wireguard: 'success',
        // 兼容性
        v2ray: 'primary'
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
    onMounted(() => {
      fetchNodes()
    })
    return {
      loading,
      nodes,
      filterKeyword,
      filterRegion,
      filterType,
      filterSource,
      hasActiveFilters,
      nodeStats,
      filteredNodes,
      paginatedNodes,
      mobileNodeFields,
      pagination,
      regions,
      nodeTypes,
      fetchNodes,
      refreshNodes,
      resetFilters,
      applyFilters,
      handleSizeChange,
      handlePageChange,
      getRegionColor,
      getTypeColor,
      getStatusType,
      getStatusText,
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
.stats-card {
  margin-bottom: 2rem;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}
.speed-status-card {
  margin-bottom: 2rem;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
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
.stat-label {
  color: #666;
  font-size: 0.9rem;
}
.nodes-card {
  margin-bottom: 14px;
  border-radius: 8px;
  border: 1px solid #dcdfe6;
}
.nodes-filter-card {
  margin-bottom: 14px;
}
.nodes-filter-body {
  padding: 16px;
}
.nodes-filter-form {
  display: grid;
  grid-template-columns: minmax(220px, 1.2fr) repeat(3, minmax(150px, 0.8fr)) minmax(150px, max-content);
  align-items: end;
  gap: 12px;
  width: 100%;

  :deep(.el-form-item) {
    margin: 0;
    min-width: 0;
  }

  :deep(.el-form-item__label) {
    color: #606266;
    font-weight: 600;
  }

  :deep(.el-form-item__content) {
    width: 100%;
    min-width: 0;
  }
}
.nodes-filter-select {
  width: 100%;
  min-width: 0;
}
.nodes-keyword-input {
  width: 100%;
  min-width: 0;
}
.nodes-filter-actions {
  justify-self: end;

  :deep(.el-form-item__content) {
    display: flex;
    flex-wrap: nowrap;
    gap: 8px;
  }
}
@media (max-width: 1100px) {
  .nodes-filter-form {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .nodes-filter-actions {
    justify-self: start;
  }
}
.stats-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 14px;
}
.stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px;
  background: #fff;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  text-align: left;
}
.stat-icon {
  width: 46px;
  height: 46px;
  display: grid;
  place-items: center;
  flex: 0 0 auto;
  border-radius: 8px;
  background: #ecf5ff;
  color: #409eff;
  font-weight: 900;
}
.stat-value {
  color: #409eff;
  font-size: 22px;
  font-weight: 800;
  line-height: 1.2;
}
.list-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}
.node-name {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.header-icon,
.node-type-icon {
  flex-shrink: 0;
  color: #1677ff;
}
.header-icon {
  margin-right: 6px;
}
.nodes-table {
  width: 100%;
}
.recommended-tag {
  margin-left: 8px;
}
.node-type-icon {
  font-size: 1.2rem;
}
.last-test-time {
  color: #666;
  font-size: 0.875rem;
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
  min-width: 0;
}
.mobile-node-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
  span {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    word-break: break-word;
    .node-type-icon {
      color: #409eff;
      font-size: 16px;
    }
  }
}
.mobile-node-empty {
  min-height: 180px;
  padding: 24px 16px;
}
@media (max-width: 768px) {
  .nodes-container { padding: 10px; }
  .nodes-filter-form {
    display: grid;
    grid-template-columns: 1fr;
    align-items: stretch;
  }
  .nodes-filter-actions {
    justify-self: stretch;
  }
  .nodes-filter-select,
  .nodes-keyword-input,
  .nodes-filter-actions,
  .nodes-filter-actions :deep(.el-form-item__content),
  .nodes-filter-actions :deep(.el-button) {
    width: 100%;
  }
  .stats-row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 480px) {
  .stats-row {
    grid-template-columns: 1fr;
  }
}
</style>
