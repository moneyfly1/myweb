<template>
  <div class="list-container admin-nodes">
    <el-card class="list-card" shadow="never">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <span class="title-text">节点管理</span>
            <el-tag v-if="pagination.total" type="info" round size="small" class="count-tag">{{ pagination.total }}</el-tag>
          </div>
          <div class="header-actions" v-if="!isMobile">
            <el-button type="primary" @click="handleAdd">
              <el-icon><Plus /></el-icon>添加节点
            </el-button>
            <el-button type="success" @click="batchTest" :loading="testing" :disabled="!selectedNodes.length">
              <el-icon><Connection /></el-icon>批量测试
            </el-button>
            <el-button type="danger" @click="batchDelete" :loading="deleting" :disabled="!selectedNodes.length">
              <el-icon><Delete /></el-icon>批量删除
            </el-button>
            <el-button @click="loadNodes" :loading="loading">
              <el-icon><Refresh /></el-icon>刷新
            </el-button>
          </div>
          <div class="header-actions mobile" v-else>
            <el-button type="primary" circle @click="handleAdd" size="small">
              <el-icon><Plus /></el-icon>
            </el-button>
            <el-dropdown trigger="click" @command="handleCommand">
              <el-button circle size="small">
                <el-icon><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="refresh" :icon="Refresh">刷新列表</el-dropdown-item>
                  <el-dropdown-item command="test" :icon="Connection" :disabled="!selectedNodes.length">批量测试</el-dropdown-item>
                  <el-dropdown-item command="delete" :icon="Delete" :disabled="!selectedNodes.length" divided style="color: var(--el-color-danger)">批量删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </template>
      <div class="filter-wrapper">
        <div class="filter-grid">
          <el-select v-model="filters.status" placeholder="状态" clearable @change="loadNodes">
            <el-option label="全部状态" value="" />
            <el-option label="在线" value="online" />
            <el-option label="离线" value="offline" />
            <el-option label="超时" value="timeout" />
          </el-select>
          <el-select v-model="filters.is_active" placeholder="激活" clearable @change="loadNodes">
            <el-option label="全部" value="" />
            <el-option label="已激活" value="true" />
            <el-option label="已禁用" value="false" />
          </el-select>
          <el-select v-model="filters.region" placeholder="地区" clearable @change="loadNodes">
            <el-option label="所有地区" value="" />
            <el-option v-for="r in regions" :key="r" :label="r" :value="r" />
          </el-select>
          <el-select v-model="filters.type" placeholder="类型" clearable @change="loadNodes">
            <el-option label="所有类型" value="" />
            <el-option v-for="t in types" :key="t" :label="t" :value="t" />
          </el-select>
          <div class="search-box">
            <el-input
              v-model="searchKeyword"
              placeholder="搜索节点名称..."
              clearable
              @keyup.enter="loadNodes"
            >
              <template #prefix><el-icon><Search /></el-icon></template>
            </el-input>
          </div>
        </div>
      </div>
      <div class="content-view" v-loading="loading">
        <el-table
          v-if="!isMobile"
          :data="nodes"
          stripe
          border
          @selection-change="handleSelectionChange"
          class="desktop-table"
        >
          <el-table-column type="selection" width="50" />
          <el-table-column prop="name" label="节点名称" min-width="180" show-overflow-tooltip />
          <el-table-column prop="region" label="地区" width="100" />
          <el-table-column prop="type" label="类型" width="90">
            <template #default="{ row }">
              <el-tag effect="plain" size="small">{{ row.type?.toUpperCase() }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-badge :is-dot="true" :type="getStatusType(row.status)" class="status-badge">
                <span>{{ getStatusText(row.status) }}</span>
              </el-badge>
            </template>
          </el-table-column>
          <el-table-column label="激活" width="80">
            <template #default="{ row }">
              <el-switch v-model="row.is_active" @change="toggleNodeStatus(row)" size="small" />
            </template>
          </el-table-column>
          <el-table-column label="延迟" width="100">
            <template #default="{ row }">
              <span :class="getLatencyClass(row.latency)">{{ formatLatency(row.latency) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="{ row }">
              <el-button-group>
                <el-button size="small" @click="testNode(row)" :loading="row.testing" :icon="Connection" title="测试" />
                <el-button size="small" type="primary" @click="editNode(row)" :icon="Edit" title="编辑" />
                <el-button size="small" type="danger" @click="deleteNode(row)" :icon="Delete" title="删除" />
              </el-button-group>
            </template>
          </el-table-column>
        </el-table>
        <div v-else class="mobile-list">
          <div class="mobile-selection-bar" v-if="nodes.length > 0">
            <el-checkbox 
              v-model="isAllSelected" 
              :indeterminate="isIndeterminate" 
              @change="toggleMobileSelectAll"
            >全选 ({{ selectedNodes.length }})</el-checkbox>
          </div>
          <div v-for="node in nodes" :key="node.id" class="node-card">
            <div class="card-header-row">
              <el-checkbox 
                :model-value="isSelected(node)" 
                @change="(val) => handleMobileSelect(node, val)"
                class="card-checkbox" 
              />
              <div class="node-title">{{ node.name }}</div>
              <el-tag size="small" :type="getStatusType(node.status)" effect="light">{{ getStatusText(node.status) }}</el-tag>
            </div>
            <div class="card-info-grid">
              <div class="info-item">
                <span class="label">地区</span>
                <span class="value">{{ node.region }}</span>
              </div>
              <div class="info-item">
                <span class="label">类型</span>
                <span class="value">{{ node.type?.toUpperCase() }}</span>
              </div>
              <div class="info-item">
                <span class="label">延迟</span>
                <span class="value" :class="getLatencyClass(node.latency)">{{ formatLatency(node.latency) }}</span>
              </div>
            </div>
            <div class="card-actions-row">
              <div class="left-actions">
                <el-switch 
                  v-model="node.is_active" 
                  @change="toggleNodeStatus(node)" 
                  size="small"
                  inline-prompt
                  active-text="开启"
                  inactive-text="关闭"
                />
              </div>
              <div class="right-buttons">
                <el-button size="small" text bg @click="testNode(node)" :loading="node.testing">测试</el-button>
                <el-button size="small" text bg type="primary" @click="editNode(node)">编辑</el-button>
                <el-button size="small" text bg type="danger" @click="deleteNode(node)">删除</el-button>
              </div>
            </div>
          </div>
          <el-empty v-if="nodes.length === 0" description="暂无数据" />
        </div>
      </div>
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.size"
          :total="pagination.total"
          :layout="isMobile ? 'prev, pager, next' : 'total, sizes, prev, pager, next, jumper'"
          :pager-count="isMobile ? 5 : 7"
          background
          @current-change="loadNodes"
          @size-change="loadNodes"
        />
      </div>
    </el-card>
    <el-dialog
      v-model="showAddDialog"
      :title="editingNode ? '编辑节点' : '添加节点'"
      :width="isMobile ? '100%' : '650px'"
      :fullscreen="isMobile"
      destroy-on-close
      class="responsive-dialog"
      append-to-body
    >
      <div class="dialog-scroll-content">
        <el-tabs v-model="addNodeTab" v-if="!editingNode" class="compact-tabs">
          <el-tab-pane label="链接导入" name="link">
            <div class="import-section">
              <el-alert title="支持 vmess, vless, trojan, ss, ssr 等链接批量导入" type="info" :closable="false" show-icon />
              <el-input
                v-model="nodeLinkInput"
                type="textarea"
                :rows="isMobile ? 8 : 6"
                placeholder="请粘贴节点链接，每行一个..."
                class="link-textarea"
              />
              <div class="parsed-preview" v-if="parsedNode">
                <div class="preview-title">解析预览</div>
                <div class="preview-row"><span>名称:</span> {{ parsedNode.name }}</div>
                <div class="preview-row"><span>地址:</span> {{ parsedNode.server }}:{{ parsedNode.port }}</div>
              </div>
            </div>
          </el-tab-pane>
          <el-tab-pane label="手动填写" name="manual">
            <div class="form-container">
            </div>
          </el-tab-pane>
        </el-tabs>
        <el-form 
          v-if="editingNode || addNodeTab === 'manual'" 
          :model="nodeForm" 
          :label-position="isMobile ? 'top' : 'right'" 
          label-width="80px"
          class="node-form"
        >
          <el-row :gutter="12">
            <el-col :span="isMobile ? 24 : 12">
              <el-form-item label="名称" required>
                <el-input v-model="nodeForm.name" placeholder="节点别名" />
              </el-form-item>
            </el-col>
            <el-col :span="isMobile ? 24 : 12">
              <el-form-item label="地区" required>
                <el-input v-model="nodeForm.region" placeholder="如: 香港" />
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="类型" required>
                <el-radio-group v-model="nodeForm.type" size="small" class="type-radio">
                  <el-radio-button label="vmess">VMess</el-radio-button>
                  <el-radio-button label="vless">VLESS</el-radio-button>
                  <el-radio-button label="trojan">Trojan</el-radio-button>
                  <el-radio-button label="ss">SS</el-radio-button>
                </el-radio-group>
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="配置">
                <el-input 
                  v-model="nodeForm.config" 
                  type="textarea" 
                  :rows="6" 
                  placeholder='{"server":"1.2.3.4", "port":443, ...}' 
                  class="code-input"
                />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="推荐">
                <el-switch v-model="nodeForm.is_recommended" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="激活">
                <el-switch v-model="nodeForm.is_active" />
              </el-form-item>
            </el-col>
          </el-row>
          <div v-if="editingNode && nodeLink" class="link-generator">
            <div class="link-label">节点链接</div>
            <div class="link-box">
              <div class="link-text">{{ nodeLink }}</div>
              <el-button type="primary" link @click="copyNodeLink">
                <el-icon><DocumentCopy /></el-icon>
              </el-button>
            </div>
          </div>
        </el-form>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showAddDialog = false">取消</el-button>
          <template v-if="!editingNode && addNodeTab === 'link'">
            <el-button type="warning" plain @click="parseNodeLink" :loading="parsing">仅解析预览</el-button>
            <el-button type="primary" @click="batchImportLinks" :loading="saving" :disabled="!nodeLinkInput">批量导入</el-button>
          </template>
          <el-button v-else type="primary" @click="saveNode" :loading="saving">保存节点</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>
<script>
import { ref, reactive, onMounted, onUnmounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Plus, Refresh, Search, Connection, Delete, 
  DocumentCopy, Edit, MoreFilled 
} from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
export default {
  name: 'AdminNodes',
  components: { 
    Plus, Refresh, Search, Connection, Delete, 
    DocumentCopy, Edit, MoreFilled 
  },
  setup() {
    const isMobile = ref(false)
    const loading = ref(false)
    const testing = ref(false)
    const deleting = ref(false)
    const saving = ref(false)
    const parsing = ref(false)
    const nodes = ref([])
    const selectedNodes = ref([])
    const showAddDialog = ref(false)
    const editingNode = ref(null)
    const searchKeyword = ref('')
    const regions = ref([])
    const types = ref([])
    const addNodeTab = ref('link')
    const nodeLinkInput = ref('')
    const parsedNode = ref(null)
    const filters = reactive({ status: '', is_active: '', region: '', type: '' })
    const pagination = reactive({ page: 1, size: 20, total: 0 })
    const nodeForm = reactive({
      name: '', region: '', type: 'vmess', config: '',
      description: '', is_recommended: false, is_active: true
    })
    const checkMobile = () => {
      isMobile.value = window.innerWidth <= 768
    }
    const loadNodes = async () => {
      loading.value = true
      try {
        const params = {
          page: pagination.page,
          size: pagination.size,
          ...filters,
          search: searchKeyword.value
        }
        Object.keys(params).forEach(k => !params[k] && delete params[k])
        const res = await adminAPI.getAdminNodes(params)
        if (res.data?.success) {
          const raw = res.data.data
          const list = Array.isArray(raw) ? raw : (raw.nodes || raw.data || [])
          nodes.value = list.map(n => ({ ...n, testing: false }))
          pagination.total = raw.total || list.length
          const rSet = new Set(), tSet = new Set()
          list.forEach(n => { if(n.region) rSet.add(n.region); if(n.type) tSet.add(n.type) })
          regions.value = Array.from(rSet).sort()
          types.value = Array.from(tSet).sort()
        }
      } catch (err) {
        ElMessage.error('加载失败: ' + err.message)
      } finally {
        loading.value = false
      }
    }
    const handleMobileSelect = (node, checked) => {
      if (checked) {
        if (!selectedNodes.value.find(n => n.id === node.id)) {
          selectedNodes.value.push(node)
        }
      } else {
        selectedNodes.value = selectedNodes.value.filter(n => n.id !== node.id)
      }
    }
    const isSelected = (node) => selectedNodes.value.some(n => n.id === node.id)
    const isAllSelected = computed({
      get: () => nodes.value.length > 0 && selectedNodes.value.length === nodes.value.length,
      set: (val) => toggleMobileSelectAll(val)
    })
    const isIndeterminate = computed(() => {
      return selectedNodes.value.length > 0 && selectedNodes.value.length < nodes.value.length
    })
    const toggleMobileSelectAll = (val) => {
      selectedNodes.value = val ? [...nodes.value] : []
    }
    const handleSelectionChange = (val) => selectedNodes.value = val
    const handleAdd = () => {
      resetForm()
      showAddDialog.value = true
    }
    const handleCommand = (cmd) => {
      const actions = { refresh: loadNodes, test: batchTest, delete: batchDelete }
      actions[cmd] && actions[cmd]()
    }
    const editNode = (node) => {
      editingNode.value = node
      Object.assign(nodeForm, {
        name: node.name || '',
        region: node.region || '',
        type: node.type || 'vmess',
        config: typeof node.config === 'object' ? JSON.stringify(node.config, null, 2) : (node.config || ''),
        description: node.description || '',
        is_recommended: !!node.is_recommended,
        is_active: node.is_active !== false
      })
      showAddDialog.value = true
    }
    const saveNode = async () => {
      if (!nodeForm.name || !nodeForm.region) return ElMessage.warning('请填写必填项')
      saving.value = true
      try {
        const payload = { ...nodeForm }
        const res = editingNode.value 
          ? await adminAPI.updateNode(editingNode.value.id, payload)
          : await adminAPI.createNode(payload)
        if (res.data.success) {
          ElMessage.success('保存成功')
          showAddDialog.value = false
          loadNodes()
        }
      } catch (err) {
        ElMessage.error('保存失败: ' + err.message)
      } finally {
        saving.value = false
      }
    }
    const deleteNode = async (node) => {
      try {
        await ElMessageBox.confirm(`确认删除节点 "${node.name}"?`, '警告', { type: 'warning' })
        await adminAPI.deleteNode(node.id)
        ElMessage.success('删除成功')
        loadNodes()
      } catch (error) {
        if (error !== 'cancel') ElMessage.error('删除节点失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const batchTest = async () => {
      testing.value = true
      try {
        await adminAPI.batchTestNodes(selectedNodes.value.map(n => n.id))
        ElMessage.success('批量测试请求已发送')
        setTimeout(loadNodes, 1000) // 稍作延迟刷新
      } catch (err) {
        ElMessage.error('测试失败')
      } finally {
        testing.value = false
      }
    }
    const batchDelete = async () => {
      try {
        await ElMessageBox.confirm(`确认删除选中的 ${selectedNodes.value.length} 个节点?`, '警告', { type: 'error' })
        deleting.value = true
        await adminAPI.batchDeleteNodes(selectedNodes.value.map(n => n.id))
        ElMessage.success('批量删除成功')
        selectedNodes.value = [] // 重置选中
        loadNodes()
      } catch (error) {
        if (error !== 'cancel') ElMessage.error('批量删除失败: ' + (error.response?.data?.message || error.message))
      } finally {
        deleting.value = false
      }
    }
    const testNode = async (node) => {
      node.testing = true
      try {
        const res = await adminAPI.testNode(node.id)
        if (res.data.success) {
          node.latency = res.data.data.latency
          node.status = res.data.data.status
          ElMessage.success(`延迟: ${node.latency}ms`)
        }
      } catch {
        ElMessage.error('测试失败')
      } finally {
        node.testing = false
      }
    }
    const toggleNodeStatus = async (node) => {
      try {
        await adminAPI.updateNode(node.id, { is_active: node.is_active })
        ElMessage.success(node.is_active ? '已启用' : '已禁用')
      } catch {
        node.is_active = !node.is_active
        ElMessage.error('状态更新失败')
      }
    }
    const parseNodeLink = async () => {
      const link = nodeLinkInput.value.split('\n')[0].trim()
      if (!link) return ElMessage.warning('请输入链接')
      parsing.value = true
      try {
        const res = await adminAPI.createNode({ node_link: link, preview: true })
        if (res.data.success) parsedNode.value = res.data.data
      } finally { parsing.value = false }
    }
    const batchImportLinks = async () => {
      const links = nodeLinkInput.value.split('\n').map(l => l.trim()).filter(Boolean)
      if (!links.length) return
      saving.value = true
      try {
        const res = await adminAPI.importNodeLinks(links)
        ElMessage.success(`导入成功 ${res.data.imported} 个`)
        showAddDialog.value = false
        loadNodes()
      } catch(e) {
        ElMessage.error('导入出错') 
      } finally { saving.value = false }
    }
    const resetForm = () => {
      editingNode.value = null
      Object.assign(nodeForm, { name: '', region: '', type: 'vmess', config: '', is_active: true })
      nodeLinkInput.value = ''
      parsedNode.value = null
    }
    const getStatusType = (s) => ({ online: 'success', offline: 'danger', timeout: 'warning' }[s] || 'info')
    const getStatusText = (s) => ({ online: '在线', offline: '离线', timeout: '超时' }[s] || '未知')
    const formatLatency = (l) => l > 0 ? `${l}ms` : '-'
    const getLatencyClass = (l) => l <= 0 ? '' : l < 200 ? 'text-green' : l < 500 ? 'text-orange' : 'text-red'
    const nodeLink = computed(() => {
      if (!editingNode.value || !nodeForm.config) return ''
      return `vmess://(预览链接)` 
    })
    const copyNodeLink = () => {
      navigator.clipboard.writeText(nodeLink.value)
      ElMessage.success('复制成功')
    }
    onMounted(() => {
      checkMobile()
      window.addEventListener('resize', checkMobile)
      loadNodes()
    })
    onUnmounted(() => window.removeEventListener('resize', checkMobile))
    return {
      isMobile, loading, testing, deleting, saving, parsing,
      nodes, selectedNodes, showAddDialog, editingNode,
      filters, pagination, nodeForm, regions, types,
      searchKeyword, addNodeTab, nodeLinkInput, parsedNode,
      loadNodes, handleSelectionChange, handleMobileSelect,
      handleAdd, handleCommand, editNode, saveNode, deleteNode,
      batchTest, batchDelete, testNode, toggleNodeStatus,
      parseNodeLink, batchImportLinks, copyNodeLink, nodeLink,
      getStatusType, getStatusText, getLatencyClass, formatLatency,
      isSelected, isAllSelected, isIndeterminate, toggleMobileSelectAll,
      Plus, Refresh, Search, Connection, Delete, DocumentCopy, Edit, MoreFilled
    }
  }
}
</script>
<style scoped>
.admin-nodes {
  padding: 12px;
}
@media (max-width: 768px) {
  .admin-nodes {
    padding: 10px;
  }
}
.list-card {
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.title-text {
  font-size: 16px;
  font-weight: 600;
  margin-right: 8px;
}
.header-actions {
  display: flex;
  gap: 8px;
}
.filter-wrapper {
  background: var(--el-fill-color-light);
  padding: 16px;
  border-radius: 6px;
  margin-bottom: 16px;
}
.filter-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}
.filter-grid .el-select {
  width: 140px;
}
.search-box {
  flex: 1;
  min-width: 200px;
}
@media (max-width: 768px) {
  .filter-wrapper {
    padding: 12px;
  }
  .filter-grid {
    display: grid;
    grid-template-columns: 1fr 1fr; /* 两列布局 */
    gap: 8px;
  }
  .filter-grid .el-select {
    width: 100%;
  }
  .search-box {
    grid-column: 1 / -1; /* 搜索框独占一行 */
    width: 100%;
  }
}
.desktop-table .status-badge {
  margin-top: 4px;
}
.text-green { color: var(--el-color-success); font-weight: 500; }
.text-orange { color: var(--el-color-warning); }
.text-red { color: var(--el-color-danger); }
.mobile-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.mobile-selection-bar {
  padding: 0 4px;
  margin-bottom: 4px;
}
.node-card {
  background: #fff;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 12px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.02);
  transition: all 0.2s;
}
.node-card:active {
  background: var(--el-fill-color-lighter);
}
.card-header-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px dashed var(--el-border-color-lighter);
}
.node-title {
  font-weight: 600;
  font-size: 15px;
  flex: 1;
  white-space: nowrap;
  overflow: clip;
  text-overflow: ellipsis;
}
.card-checkbox {
  margin-right: 0;
  height: auto;
}
.card-info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 8px;
  margin-bottom: 12px;
}
.info-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  background: var(--el-fill-color-extra-light);
  padding: 6px;
  border-radius: 4px;
}
.info-item .label {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  margin-bottom: 2px;
}
.info-item .value {
  font-size: 13px;
  font-weight: 500;
}
.card-actions-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.right-buttons {
  display: flex;
  gap: 4px;
}
.pagination-wrapper {
  margin-top: 20px;
  display: flex;
  justify-content: center;
}
.responsive-dialog :deep(.el-dialog__body) {
  padding: 10px 20px;
}
.dialog-scroll-content {
  max-height: 70vh;
  overflow-y: auto;
  padding-right: 4px;
}
@media (max-width: 768px) {
  .responsive-dialog :deep(.el-dialog__body) {
    padding: 12px;
  }
  .dialog-scroll-content {
    max-height: calc(100vh - 120px);
  }
  .link-generator {
    margin-top: 10px;
  }
}
.link-generator {
  background: var(--el-fill-color-light);
  padding: 10px;
  border-radius: 4px;
  margin-top: 10px;
}
.link-box {
  display: flex;
  align-items: center;
  background: #fff;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  padding: 4px 8px;
}
.link-text {
  flex: 1;
  font-family: monospace;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: clip;
  text-overflow: ellipsis;
  white-space: nowrap;
}
:deep(.el-input__wrapper) {
  box-shadow: 0 0 0 1px var(--el-border-color) inset !important;
}
:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--el-color-primary) inset !important;
}
</style>