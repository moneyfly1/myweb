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
                  <el-dropdown-item command="delete" :icon="Delete" :disabled="!selectedNodes.length" divided class="danger-menu-item">批量删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </template>
      <div class="filter-wrapper">
        <div class="filter-grid">
          <el-select v-model="filters.status" placeholder="状态" clearable @change="applyNodeFilters">
            <el-option label="全部状态" value="" />
            <el-option label="在线" value="online" />
            <el-option label="离线" value="offline" />
            <el-option label="超时" value="timeout" />
          </el-select>
          <el-select v-model="filters.is_active" placeholder="激活" clearable @change="applyNodeFilters">
            <el-option label="全部" value="" />
            <el-option label="已激活" value="true" />
            <el-option label="已禁用" value="false" />
          </el-select>
          <el-select v-model="filters.is_manual" placeholder="来源" clearable @change="applyNodeFilters">
            <el-option label="所有来源" value="" />
            <el-option label="手动添加" value="true" />
            <el-option label="自动采集" value="false" />
          </el-select>
          <el-select v-model="filters.region" placeholder="地区" clearable @change="applyNodeFilters">
            <el-option label="所有地区" value="" />
            <el-option v-for="r in regions" :key="r" :label="r" :value="r" />
          </el-select>
          <el-select v-model="filters.type" placeholder="类型" clearable @change="applyNodeFilters">
            <el-option label="所有类型" value="" />
            <el-option v-for="t in allNodeTypes" :key="t" :label="t" :value="t" />
          </el-select>
          <div class="search-box">
            <el-input
              v-model="searchKeyword"
              placeholder="搜索节点名称、服务器地址或域名..."
              clearable
              @input="debouncedApplyNodeFilters"
              @clear="applyNodeFilters"
              @keyup.enter="applyNodeFilters"
            >
              <template #prefix><el-icon><Search /></el-icon></template>
            </el-input>
          </div>
          <div class="filter-actions">
            <el-button type="primary" @click="applyNodeFilters">
              <el-icon><Search /></el-icon>
              搜索
            </el-button>
            <el-button @click="resetNodeFilters">
              <el-icon><Refresh /></el-icon>
              重置
            </el-button>
          </div>
        </div>
      </div>
      <div class="content-view" v-loading="loading">
        <div class="mobile-selection-bar" v-if="isMobile && nodes.length > 0">
          <el-checkbox
            v-model="isAllSelected"
            :indeterminate="isIndeterminate"
            @change="toggleMobileSelectAll"
          >全选 ({{ selectedNodes.length }})</el-checkbox>
        </div>
        <ResponsiveDataView
          :data="nodes"
          :fields="mobileNodeFields"
          :loading="loading"
          title-field="name"
          empty-title="暂无节点数据"
          empty-description="添加节点或调整筛选条件后可在这里查看节点列表"
        >
          <template #table>
            <el-table
              :data="nodes"
              stripe
              border
              @selection-change="handleSelectionChange"
              class="desktop-table"
              ref="tableRef"
            >
              <el-table-column type="selection" width="50" />
              <el-table-column prop="name" label="节点名称" min-width="180" show-overflow-tooltip />
              <el-table-column prop="region" label="地区" width="100" />
              <el-table-column label="来源" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.is_manual ? 'warning' : 'success'" size="small" effect="light">{{ row.is_manual ? '手动' : '采集' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="订阅#" width="70">
                <template #default="{ row }">
                  <span v-if="!row.is_manual && row.source_index">#{{ row.source_index }}</span>
                  <span v-else class="text-placeholder">-</span>
                </template>
              </el-table-column>
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
            <EmptyState
              v-if="!loading && nodes.length === 0"
              class="desktop-empty-state"
              title="暂无节点数据"
              description="添加节点或调整筛选条件后可在这里查看节点列表"
            />
          </template>
          <template #header="{ item }">
            <div class="mobile-node-header">
              <el-checkbox
                :model-value="isSelected(item)"
                @change="(val) => handleMobileSelect(item, val)"
                class="mobile-node-checkbox"
              />
              <div class="mobile-node-title" :title="item.name">{{ item.name || '-' }}</div>
              <el-tag size="small" :type="getStatusType(item.status)" effect="light">
                {{ getStatusText(item.status) }}
              </el-tag>
            </div>
          </template>
          <template #field-type="{ item }">
            <el-tag effect="plain" size="small">{{ item.type?.toUpperCase() || '-' }}</el-tag>
          </template>
          <template #field-source="{ item }">
            <el-tag :type="item.is_manual ? 'warning' : 'success'" size="small" effect="light">
              {{ item.is_manual ? '手动' : '采集' }}
              <span v-if="!item.is_manual && item.source_index"> #{{ item.source_index }}</span>
            </el-tag>
          </template>
          <template #field-latency="{ item }">
            <span :class="getLatencyClass(item.latency)">{{ formatLatency(item.latency) }}</span>
          </template>
          <template #actions="{ item }">
            <div class="mobile-node-actions">
              <el-switch
                v-model="item.is_active"
                @change="toggleNodeStatus(item)"
                size="small"
                inline-prompt
                active-text="开"
                inactive-text="关"
              />
              <div class="mobile-node-buttons">
                <el-button size="small" text bg @click="testNode(item)" :loading="item.testing">测试</el-button>
                <el-button size="small" text bg type="primary" @click="editNode(item)">编辑</el-button>
                <el-button size="small" text bg type="danger" @click="deleteNode(item)">删除</el-button>
              </div>
            </div>
          </template>
          <template #empty>
            <EmptyState
              title="暂无节点数据"
              description="添加节点或调整筛选条件后可在这里查看节点列表"
            />
          </template>
        </ResponsiveDataView>
      </div>
      <div class="pagination-wrapper">
        <PaginationBar
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.size"
          :total="pagination.total"
          background
          @current-change="loadNodes"
          @size-change="loadNodes"
        />
      </div>
    </el-card>
    <AppDrawer
      v-model="showAddDialog"
      :title="editingNode ? '编辑节点' : '添加节点'"
      size="600px"
      mobile-size="100%"
      :loading="saving || parsing"
      class="node-form-drawer"
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
                <el-select v-model="nodeForm.type" placeholder="选择节点类型" class="full-width-control">
                  <el-option-group label="代理协议">
                    <el-option label="VMess" value="vmess" />
                    <el-option label="VLESS" value="vless" />
                    <el-option label="Trojan" value="trojan" />
                    <el-option label="Shadowsocks (SS)" value="ss" />
                    <el-option label="ShadowsocksR (SSR)" value="ssr" />
                  </el-option-group>
                  <el-option-group label="现代协议">
                    <el-option label="Hysteria" value="hysteria" />
                    <el-option label="Hysteria2" value="hysteria2" />
                    <el-option label="TUIC" value="tuic" />
                    <el-option label="Naive" value="naive" />
                    <el-option label="AnyTLS" value="anytls" />
                  </el-option-group>
                  <el-option-group label="其他协议">
                    <el-option label="SOCKS" value="socks" />
                    <el-option label="SOCKS5" value="socks5" />
                    <el-option label="HTTP" value="http" />
                    <el-option label="HTTPS" value="https" />
                    <el-option label="WireGuard (WG)" value="wg" />
                  </el-option-group>
                </el-select>
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
          <div v-if="editingNode" class="link-generator">
            <div class="link-label">节点链接</div>
            <div class="link-box" v-if="nodeLink">
              <div class="link-text">{{ nodeLink }}</div>
              <el-button class="link-copy-btn" type="primary" link @click="copyNodeLink">
                <el-icon><DocumentCopy /></el-icon>
              </el-button>
            </div>
            <div v-else class="link-loading">加载中...</div>
          </div>
        </el-form>
      </div>
      <template #footer>
        <FormActionBar :loading="saving || parsing">
          <el-button :disabled="saving || parsing" @click="showAddDialog = false">取消</el-button>
          <template v-if="!editingNode && addNodeTab === 'link'">
            <el-button type="warning" plain @click="parseNodeLink" :loading="parsing">仅解析预览</el-button>
            <el-button type="primary" @click="batchImportLinks" :loading="saving" :disabled="!nodeLinkInput">批量导入</el-button>
          </template>
          <el-button v-else type="primary" @click="saveNode" :loading="saving">保存节点</el-button>
        </FormActionBar>
      </template>
    </AppDrawer>
  </div>
</template>
<script>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { 
  Plus, Refresh, Search, Connection, Delete, 
  DocumentCopy, Edit, MoreFilled 
} from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
import AppDrawer from '@/components/AppDrawer.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import EmptyState from '@/components/EmptyState.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import { confirmDelete } from '@/utils/confirmAction'
import { useMobile } from '@/composables/useMobile'
import { debounce } from '@/composables/useDebounce'
export default {
  name: 'AdminNodes',
  components: { 
    Plus, Refresh, Search, Connection, Delete, 
    DocumentCopy, Edit, MoreFilled, AppDrawer, FormActionBar, PaginationBar, EmptyState, ResponsiveDataView
  },
  setup() {
    // 所有支持的节点类型（完整列表）
    const allNodeTypes = [
      'vmess', 'vless', 'trojan', 'ss', 'ssr',
      'hysteria', 'hysteria2', 'tuic', 'naive', 'anytls',
      'socks', 'socks5', 'http', 'https', 'wg', 'wireguard'
    ]
    
    const isMobile = useMobile()
    const tableRef = ref(null)
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
    const nodeLinkValue = ref('')
    const parsedNode = ref(null)
    const filters = reactive({ status: '', is_active: '', is_manual: '', region: '', type: '' })
    const pagination = reactive({ page: 1, size: 10, total: 0 })
    const mobileNodeFields = computed(() => [
      { key: 'region', label: '地区' },
      { key: 'type', label: '类型' },
      { key: 'source', label: '来源' },
      { key: 'latency', label: '延迟' }
    ])
    const nodeForm = reactive({
      name: '', region: '', type: 'vmess', config: '',
      description: '', is_recommended: false, is_active: true
    })
    const loadNodes = async () => {
      loading.value = true
      try {
        const params = {
          page: pagination.page,
          size: pagination.size,
          ...filters,
          search: searchKeyword.value,
          _t: Date.now() // 缓存穿透参数，避免浏览器/代理返回旧列表
        }
        Object.keys(params).forEach(k => !params[k] && delete params[k])
        const res = await adminAPI.getAdminNodes(params)
        if (res.data?.success) {
          const raw = res.data.data
          const list = Array.isArray(raw) ? raw : (raw.nodes || raw.data || [])
          nodes.value = list.map(n => ({ ...n, testing: false }))
          pagination.total = raw.total || list.length
          const rSet = new Set(), tSet = new Set(allNodeTypes)
          list.forEach(n => { 
            if(n.region) rSet.add(n.region)
            if(n.type) tSet.add(n.type) // 补充任何新的协议类型
          })
          regions.value = Array.from(rSet).sort()
          types.value = Array.from(tSet).sort()
        }
      } catch (err) {
        ElMessage.error('加载失败: ' + err.message)
      } finally {
        loading.value = false
      }
    }
    const applyNodeFilters = () => {
      pagination.page = 1
      loadNodes()
    }
    // 搜索输入实时生效，无需再次点击搜索按钮（500ms 防抖）
    const debouncedApplyNodeFilters = debounce(applyNodeFilters, 500)
    const resetNodeFilters = () => {
      Object.assign(filters, { status: '', is_active: '', is_manual: '', region: '', type: '' })
      searchKeyword.value = ''
      pagination.page = 1
      loadNodes()
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
    const editNode = async (node) => {
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
      nodeLinkValue.value = ''
      showAddDialog.value = true
      // 异步获取真实节点链接
      try {
        const res = await adminAPI.getNodeLink(node.id)
        if (res.data?.success && res.data.data?.link) {
          nodeLinkValue.value = res.data.data.link
        }
      } catch (e) {
        console.warn('获取节点链接失败', e)
      }
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
        await confirmDelete('节点', 1, {
          message: `确认删除节点 "${node.name}"?`
        })
        await adminAPI.deleteNode(node.id)
        ElMessage.success('删除成功')
        // 先从本地列表立即移除，再向服务端重新拉取校准，保证页面马上不显示已删除节点
        nodes.value = nodes.value.filter(n => n.id !== node.id)
        if (pagination.total > 0) pagination.total -= 1
        selectedNodes.value = selectedNodes.value.filter(n => n.id !== node.id)
        tableRef.value?.clearSelection()
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
        await confirmDelete('节点', selectedNodes.value.length, {
          message: `确认删除选中的 ${selectedNodes.value.length} 个节点?`
        })
        deleting.value = true
        const deletedIds = selectedNodes.value.map(n => n.id)
        const res = await adminAPI.batchDeleteNodes(deletedIds)
        const deletedCount = res.data?.data?.deleted_count ?? 0
        if (!res.data?.success) {
          throw new Error(res.data?.message || '批量删除失败')
        }
        if (deletedCount > 0) {
          ElMessage.success(`批量删除成功，已删除 ${deletedCount} 个节点`)
        } else {
          // 节点实际已不存在（页面显示的是旧数据），直接刷新列表即可
          ElMessage.info(res.data?.message || '所选节点已不存在，已刷新列表')
        }
        // 先从本地列表立即移除所选节点，再向服务端重新拉取校准，
        // 避免因浏览器/代理缓存导致重新加载后仍显示旧节点
        nodes.value = nodes.value.filter(n => !deletedIds.includes(n.id))
        pagination.total = Math.max(0, pagination.total - deletedIds.length)
        selectedNodes.value = [] // 重置选中
        tableRef.value?.clearSelection()
        await loadNodes()
      } catch (error) {
        if (error === 'cancel') return
        const msg = error.response?.data?.message || error.message
        // 旧版后端返回 404"未删除任何节点"/"节点不存在"说明所选节点实际已删除
        // （页面显示的是旧数据），此时直接从本地移除并刷新列表即可，不再提示失败
        if (msg && (msg.includes('未删除任何节点') || msg.includes('节点不存在'))) {
          const staleIds = selectedNodes.value.map(n => n.id)
          nodes.value = nodes.value.filter(n => !staleIds.includes(n.id))
          pagination.total = Math.max(0, pagination.total - staleIds.length)
          selectedNodes.value = []
          tableRef.value?.clearSelection()
          ElMessage.info('所选节点已不存在，已刷新列表')
          await loadNodes()
          return
        }
        ElMessage.error('批量删除失败: ' + msg)
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
        const imported = res.data.data?.imported ?? res.data.imported ?? 0
        const skipped = res.data.data?.skipped ?? res.data.skipped ?? 0
        const failed = res.data.data?.failed ?? res.data.failed ?? 0
        const errors = res.data.data?.errors ?? []
        if (imported > 0) {
          ElMessage.success(`导入成功 ${imported} 个${skipped ? `，跳过重复 ${skipped} 个` : ''}${failed ? `，失败 ${failed} 个` : ''}`)
          showAddDialog.value = false
          loadNodes()
        } else if (failed > 0) {
          // 全部失败时保留弹窗，方便用户修改链接后重试
          ElMessage.error(`导入失败 ${failed} 个：${errors[0] || '未知原因'}`)
        } else {
          ElMessage.warning(`所选节点均已存在（重复），已跳过 ${skipped} 个`)
          showAddDialog.value = false
          loadNodes()
        }
      } catch(e) {
        ElMessage.error('导入出错')
      } finally { saving.value = false }
    }
    const resetForm = () => {
      editingNode.value = null
      Object.assign(nodeForm, { name: '', region: '', type: 'vmess', config: '', description: '', is_recommended: false, is_active: true })
      nodeLinkInput.value = ''
      nodeLinkValue.value = ''
      parsedNode.value = null
    }
    const getStatusType = (s) => ({ online: 'success', offline: 'danger', timeout: 'warning' }[s] || 'info')
    const getStatusText = (s) => ({ online: '在线', offline: '离线', timeout: '超时' }[s] || '未知')
    const formatLatency = (l) => l > 0 ? `${l}ms` : '-'
    const getLatencyClass = (l) => l <= 0 ? '' : l < 200 ? 'text-green' : l < 500 ? 'text-orange' : 'text-red'
    const nodeLink = computed(() => {
      if (!editingNode.value) return ''
      return nodeLinkValue.value || ''
    })
    const copyNodeLink = () => {
      navigator.clipboard.writeText(nodeLink.value)
      ElMessage.success('复制成功')
    }
    onMounted(() => {
      loadNodes()
    })
    return {
      isMobile, tableRef, loading, testing, deleting, saving, parsing,
      nodes, selectedNodes, showAddDialog, editingNode,
      filters, pagination, nodeForm, regions, types, allNodeTypes,
      searchKeyword, addNodeTab, nodeLinkInput, parsedNode, mobileNodeFields,
      loadNodes, applyNodeFilters, debouncedApplyNodeFilters, resetNodeFilters, handleSelectionChange, handleMobileSelect,
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
.danger-menu-item {
  color: var(--el-color-danger);
}
.text-placeholder {
  color: var(--el-text-color-placeholder);
}
.full-width-control {
  width: 100%;
}
@media (max-width: 768px) {
  .search-box {
    grid-column: 1 / -1;
  }
}
.desktop-table .status-badge {
  margin-top: 4px;
}
.text-green { color: var(--el-color-success); font-weight: 500; }
.text-orange { color: var(--el-color-warning); }
.text-red { color: var(--el-color-danger); }
.mobile-selection-bar {
  display: none;
  padding: 0 4px 10px;
}
.mobile-node-header {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.mobile-node-title {
  font-weight: 600;
  font-size: 15px;
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: clip;
  text-overflow: ellipsis;
}
.mobile-node-checkbox {
  margin-right: 0;
  height: auto;
}
.mobile-node-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
}
.mobile-node-buttons {
  display: flex;
  gap: 4px;
  min-width: 0;
  flex-wrap: wrap;
  justify-content: flex-end;
}
.dialog-scroll-content {
  min-width: 0;
}
@media (max-width: 768px) {
  .mobile-selection-bar {
    display: block;
  }
  .mobile-node-actions {
    align-items: flex-start;
  }
  .link-generator {
    margin-top: 10px;
  }
  .link-box {
    padding: 6px;
  }
  .link-text {
    font-size: 11px;
    max-height: 120px;
    overflow-y: auto;
  }
}
.link-generator {
  background: var(--el-fill-color-light);
  padding: 10px;
  border-radius: 4px;
  margin-top: 10px;
}
.link-label {
  font-size: 13px;
  color: var(--el-text-color-regular);
  margin-bottom: 6px;
}
.link-box {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  background: #fff;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  padding: 8px;
}
.link-text {
  flex: 1;
  min-width: 0;
  font-family: monospace;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
  line-height: 1.5;
  user-select: all;
}
.link-copy-btn {
  flex-shrink: 0;
}
.link-loading {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  padding: 4px 0;
}
</style>
