<template>
  <div class="list-container admin-nodes">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>节点管理</span>
          <div class="header-actions desktop-only">
            <el-button type="primary" @click="showAddDialog = true">
              <el-icon><Plus /></el-icon>
              添加节点
            </el-button>
            <el-button type="success" @click="batchTest" :loading="testing">
              <el-icon><Connection /></el-icon>
              批量测试
            </el-button>
            <el-button type="danger" @click="batchDelete" :loading="deleting">
              <el-icon><Delete /></el-icon>
              批量删除
            </el-button>
            <el-button @click="loadNodes" :loading="loading">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <!-- 筛选栏 -->
      <div class="filter-bar">
        <el-select v-model="filters.status" placeholder="状态" clearable style="width: 120px" @change="loadNodes">
          <el-option label="全部" value="" />
          <el-option label="在线" value="online" />
          <el-option label="离线" value="offline" />
          <el-option label="超时" value="timeout" />
        </el-select>
        <el-select v-model="filters.is_active" placeholder="激活状态" clearable style="width: 120px" @change="loadNodes">
          <el-option label="全部" value="" />
          <el-option label="已激活" value="true" />
          <el-option label="已禁用" value="false" />
        </el-select>
        <el-select v-model="filters.region" placeholder="地区" clearable style="width: 120px" @change="loadNodes">
          <el-option label="全部" value="" />
          <el-option v-for="region in regions" :key="region" :label="region" :value="region" />
        </el-select>
        <el-select v-model="filters.type" placeholder="类型" clearable style="width: 120px" @change="loadNodes">
          <el-option label="全部" value="" />
          <el-option v-for="type in types" :key="type" :label="type" :value="type" />
        </el-select>
        <el-input
          v-model="searchKeyword"
          placeholder="搜索节点名称"
          clearable
          style="width: 200px"
          @keyup.enter="loadNodes"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <!-- 节点列表 -->
      <el-table
        :data="nodes"
        v-loading="loading"
        stripe
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="节点名称" min-width="150" />
        <el-table-column prop="region" label="地区" width="100" />
        <el-table-column prop="type" label="类型" width="100" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="激活状态" width="100">
          <template #default="{ row }">
            <el-switch
              v-model="row.is_active"
              @change="toggleNodeStatus(row)"
            />
          </template>
        </el-table-column>
        <el-table-column prop="latency" label="延迟" width="100">
          <template #default="{ row }">
            <span v-if="row.latency > 0">{{ row.latency }}ms</span>
            <span v-else style="color: #909399">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="last_test" label="最后测试" width="180">
          <template #default="{ row }">
            <span v-if="row.last_test">{{ formatTime(row.last_test) }}</span>
            <span v-else style="color: #909399">未测试</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="testNode(row)" :loading="row.testing">
              测试
            </el-button>
            <el-button size="small" type="primary" @click="editNode(row)">
              编辑
            </el-button>
            <el-button size="small" type="danger" @click="deleteNode(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.size"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadNodes"
          @current-change="loadNodes"
        />
      </div>
    </el-card>

    <!-- 添加/编辑节点对话框 -->
    <el-dialog
      v-model="showAddDialog"
      :title="editingNode ? '编辑节点' : '添加节点'"
      width="700px"
    >
      <el-tabs v-model="addNodeTab" v-if="!editingNode">
        <el-tab-pane label="节点链接导入" name="link">
          <el-alert
            title="支持格式"
            type="info"
            :closable="false"
            style="margin-bottom: 20px"
          >
            <template #default>
              <div style="line-height: 1.8;">
                <div style="margin-bottom: 8px;"><strong>支持的节点协议格式：</strong></div>
                <div style="margin-left: 10px; margin-bottom: 4px;">• <strong>VMess:</strong> vmess://（支持 TCP/WS/gRPC/H2/HTTPUpgrade）</div>
                <div style="margin-left: 10px; margin-bottom: 4px;">• <strong>VLESS:</strong> vless://（支持 TCP/WS/gRPC，包括 Reality 和 XTLS）</div>
                <div style="margin-left: 10px; margin-bottom: 4px;">• <strong>Trojan:</strong> trojan://（支持 TCP/WS/gRPC）</div>
                <div style="margin-left: 10px; margin-bottom: 4px;">• <strong>Shadowsocks:</strong> ss://（标准 Shadowsocks）</div>
                <div style="margin-left: 10px; margin-bottom: 4px;">• <strong>ShadowsocksR:</strong> ssr://（ShadowsocksR）</div>
                <div style="margin-left: 10px; margin-bottom: 4px;">• <strong>Hysteria:</strong> hysteria://（Hysteria v1）</div>
                <div style="margin-left: 10px; margin-bottom: 4px;">• <strong>Hysteria2:</strong> hysteria2://（Hysteria v2）</div>
                <div style="margin-left: 10px; margin-bottom: 4px;">• <strong>TUIC:</strong> tuic://（TUIC 协议）</div>
                <div style="margin-left: 10px; margin-bottom: 4px;">• <strong>Naive:</strong> naive+https:// 或 naive://（Naive 协议）</div>
                <div style="margin-left: 10px; margin-bottom: 4px;">• <strong>Anytls:</strong> anytls://（Anytls 协议）</div>
                <div style="margin-top: 8px; color: #909399; font-size: 12px;">
                  提示：支持单个链接或批量导入（每行一个链接），系统会自动解析并提取节点信息
                </div>
              </div>
            </template>
          </el-alert>
          <el-form label-width="100px">
            <el-form-item label="节点链接" required>
              <el-input
                v-model="nodeLinkInput"
                type="textarea"
                :rows="8"
                placeholder="请输入节点链接，支持单个或多个链接（每行一个）"
              />
              <div style="margin-top: 10px; color: #909399; font-size: 12px;">
                支持格式：vmess://、vless://、trojan://、ss://、ssr://、hysteria://、hysteria2:// 等
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="parseNodeLink" :loading="parsing">
                解析并预览
              </el-button>
              <el-button @click="clearNodeLink">清空</el-button>
            </el-form-item>
            <el-form-item v-if="parsedNode" label="解析结果">
              <el-card style="background: #f5f7fa;">
                <div style="margin-bottom: 10px;">
                  <strong>节点名称：</strong>{{ parsedNode.name }}
                </div>
                <div style="margin-bottom: 10px;">
                  <strong>类型：</strong>{{ parsedNode.type }}
                </div>
                <div style="margin-bottom: 10px;">
                  <strong>服务器：</strong>{{ parsedNode.server }}:{{ parsedNode.port }}
                </div>
                <div v-if="parsedNode.region" style="margin-bottom: 10px;">
                  <strong>地区：</strong>{{ parsedNode.region }}
                </div>
              </el-card>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="手动填写" name="manual">
          <el-form :model="nodeForm" label-width="100px">
            <el-form-item label="节点名称" required>
              <el-input 
                v-model="nodeForm.name" 
                placeholder="请输入节点名称"
                :clearable="true"
              />
            </el-form-item>
            <el-form-item label="地区" required>
              <el-input 
                v-model="nodeForm.region" 
                placeholder="请输入地区"
                :clearable="true"
              />
            </el-form-item>
            <el-form-item label="类型" required>
              <el-select v-model="nodeForm.type" placeholder="请选择类型">
                <el-option label="vmess" value="vmess" />
                <el-option label="vless" value="vless" />
                <el-option label="trojan" value="trojan" />
                <el-option label="ss" value="ss" />
                <el-option label="ssr" value="ssr" />
              </el-select>
            </el-form-item>
            <el-form-item label="配置(JSON)">
              <el-input
                v-model="nodeForm.config"
                type="textarea"
                :rows="6"
                placeholder='请输入节点配置JSON，例如: {"server":"example.com","port":443,"uuid":"xxx"}'
              />
            </el-form-item>
            <el-form-item label="描述">
              <el-input
                v-model="nodeForm.description"
                type="textarea"
                :rows="3"
                placeholder="请输入节点描述"
              />
            </el-form-item>
            <el-form-item label="推荐节点">
              <el-switch v-model="nodeForm.is_recommended" />
            </el-form-item>
            <el-form-item label="激活状态">
              <el-switch v-model="nodeForm.is_active" />
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
      
      <!-- 编辑模式直接显示表单 -->
      <el-form v-if="editingNode" :model="nodeForm" label-width="100px">
        <el-form-item label="节点名称" required>
          <el-input 
            v-model="nodeForm.name" 
            placeholder="请输入节点名称"
            :clearable="true"
          />
        </el-form-item>
        <el-form-item label="地区" required>
          <el-input 
            v-model="nodeForm.region" 
            placeholder="请输入地区"
            :clearable="true"
          />
        </el-form-item>
        <el-form-item label="类型" required>
          <el-select v-model="nodeForm.type" placeholder="请选择类型">
            <el-option label="vmess" value="vmess" />
            <el-option label="vless" value="vless" />
            <el-option label="trojan" value="trojan" />
            <el-option label="ss" value="ss" />
            <el-option label="ssr" value="ssr" />
          </el-select>
        </el-form-item>
        <el-form-item label="配置(JSON)">
          <el-input
            v-model="nodeForm.config"
            type="textarea"
            :rows="6"
            placeholder='请输入节点配置JSON'
          />
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="nodeForm.description"
            type="textarea"
            :rows="3"
            placeholder="请输入节点描述"
          />
        </el-form-item>
        <el-form-item label="推荐节点">
          <el-switch v-model="nodeForm.is_recommended" />
        </el-form-item>
        <el-form-item label="激活状态">
          <el-switch v-model="nodeForm.is_active" />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="cancelAddNode">取消</el-button>
        <el-button 
          v-if="!editingNode && addNodeTab === 'link' && parsedNode"
          type="primary" 
          @click="saveNodeFromLink" 
          :loading="saving"
        >
          保存节点
        </el-button>
        <el-button 
          v-else-if="!editingNode && addNodeTab === 'link'"
          type="success" 
          @click="batchImportLinks" 
          :loading="saving"
        >
          批量导入
        </el-button>
        <el-button 
          v-else
          type="primary" 
          @click="saveNode" 
          :loading="saving"
        >
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Search, Connection, Delete } from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'

export default {
  name: 'AdminNodes',
  components: {
    Plus,
    Refresh,
    Search,
    Connection,
    Delete
  },
  setup() {
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

    const filters = reactive({
      status: '',
      is_active: '',
      region: '',
      type: ''
    })

    const pagination = reactive({
      page: 1,
      size: 20,
      total: 0
    })

    const nodeForm = reactive({
      name: '',
      region: '',
      type: '',
      config: '',
      description: '',
      is_recommended: false,
      is_active: true
    })

    const loadNodes = async () => {
      loading.value = true
      try {
        const params = {
          page: pagination.page,
          size: pagination.size
        }
        if (filters.status) params.status = filters.status
        if (filters.is_active) params.is_active = filters.is_active
        if (filters.region) params.region = filters.region
        if (filters.type) params.type = filters.type
        if (searchKeyword.value) params.search = searchKeyword.value

        const response = await adminAPI.getAdminNodes(params)
        if (response.data && response.data.success) {
          // 处理多种响应格式
          let nodeList = []
          
          // 检查响应数据结构
          if (Array.isArray(response.data.data)) {
            // 直接是数组
            nodeList = response.data.data
          } else if (response.data.data && Array.isArray(response.data.data.data)) {
            // 嵌套结构: {success: true, data: {data: [...], total: 100}}
            nodeList = response.data.data.data
            pagination.total = response.data.data.total || nodeList.length
            pagination.page = response.data.data.page || pagination.page
            pagination.size = response.data.data.size || pagination.size
          } else if (response.data.data && response.data.data.nodes && Array.isArray(response.data.data.nodes)) {
            // 另一种嵌套结构: {success: true, data: {nodes: [...], total: 100}}
            nodeList = response.data.data.nodes
            pagination.total = response.data.data.total || nodeList.length
          } else {
            // 如果都不是，尝试获取 total
            if (response.data.data && typeof response.data.data === 'object') {
              pagination.total = response.data.data.total || 0
            }
            nodeList = []
          }
          
          // 确保 nodeList 是数组
          if (!Array.isArray(nodeList)) {
            console.warn('节点列表不是数组格式:', nodeList)
            nodeList = []
          }
          
          nodes.value = nodeList.map(node => ({
            ...node,
            testing: false
          }))
          
          // 提取地区和类型
          const regionSet = new Set()
          const typeSet = new Set()
          nodeList.forEach(node => {
            if (node && node.region) regionSet.add(node.region)
            if (node && node.type) typeSet.add(node.type)
          })
          regions.value = Array.from(regionSet).sort()
          types.value = Array.from(typeSet).sort()
          
          // 如果没有分页信息，使用数据长度
          if (!pagination.total) {
            pagination.total = nodeList.length
          }
        } else {
          ElMessage.error(response.data?.message || '获取节点列表失败')
          nodes.value = []
        }
      } catch (error) {
        ElMessage.error('加载节点列表失败: ' + (error.response?.data?.message || error.message))
      } finally {
        loading.value = false
      }
    }

    const testNode = async (node) => {
      node.testing = true
      try {
        const response = await adminAPI.testNode(node.id)
        if (response.data.success) {
          ElMessage.success(`节点测试完成: ${response.data.data.status}, 延迟: ${response.data.data.latency}ms`)
          await loadNodes()
        } else {
          ElMessage.error(response.data.message || '测试失败')
        }
      } catch (error) {
        ElMessage.error('测试失败: ' + (error.response?.data?.message || error.message))
      } finally {
        node.testing = false
      }
    }

    const batchTest = async () => {
      if (selectedNodes.value.length === 0) {
        ElMessage.warning('请先选择要测试的节点')
        return
      }
      testing.value = true
      try {
        const nodeIds = selectedNodes.value.map(n => n.id)
        const response = await adminAPI.batchTestNodes(nodeIds)
        if (response.data.success) {
          ElMessage.success(`批量测试完成，共测试 ${response.data.data.length} 个节点`)
          await loadNodes()
        } else {
          ElMessage.error(response.data.message || '批量测试失败')
        }
      } catch (error) {
        ElMessage.error('批量测试失败: ' + (error.response?.data?.message || error.message))
      } finally {
        testing.value = false
      }
    }

    const batchDelete = async () => {
      if (selectedNodes.value.length === 0) {
        ElMessage.warning('请先选择要删除的节点')
        return
      }
      
      try {
        await ElMessageBox.confirm(
          `确定要删除选中的 ${selectedNodes.value.length} 个节点吗？此操作不可恢复！`,
          '确认批量删除',
          {
            confirmButtonText: '确定删除',
            cancelButtonText: '取消',
            type: 'warning',
            dangerouslyUseHTMLString: false
          }
        )
        
        deleting.value = true
        try {
          const nodeIds = selectedNodes.value.map(n => n.id)
          const response = await adminAPI.batchDeleteNodes(nodeIds)
          if (response.data.success) {
            ElMessage.success(response.data.message || `成功删除 ${response.data.data?.deleted_count || selectedNodes.value.length} 个节点`)
            selectedNodes.value = [] // 清空选择
            await loadNodes()
          } else {
            ElMessage.error(response.data.message || '批量删除失败')
          }
        } catch (error) {
          ElMessage.error('批量删除失败: ' + (error.response?.data?.message || error.message))
        } finally {
          deleting.value = false
        }
      } catch (error) {
        // 用户取消操作
        if (error !== 'cancel') {
          ElMessage.error('操作失败: ' + (error.response?.data?.message || error.message))
        }
      }
    }

    const toggleNodeStatus = async (node) => {
      try {
        const response = await adminAPI.updateNode(node.id, {
          is_active: node.is_active
        })
        if (response.data.success) {
          ElMessage.success(node.is_active ? '节点已启用' : '节点已禁用')
        } else {
          node.is_active = !node.is_active // 回滚
          ElMessage.error(response.data.message || '操作失败')
        }
      } catch (error) {
        node.is_active = !node.is_active // 回滚
        ElMessage.error('操作失败: ' + (error.response?.data?.message || error.message))
      }
    }

    const editNode = (node) => {
      editingNode.value = node
      nodeForm.name = node.name
      nodeForm.region = node.region
      nodeForm.type = node.type
      nodeForm.config = node.config || ''
      nodeForm.description = node.description || ''
      nodeForm.is_recommended = node.is_recommended || false
      nodeForm.is_active = node.is_active
      showAddDialog.value = true
    }

    const saveNode = async () => {
      if (!nodeForm.name || !nodeForm.region || !nodeForm.type) {
        ElMessage.warning('请填写必填项')
        return
      }

      saving.value = true
      try {
        let response
        if (editingNode.value) {
          response = await adminAPI.updateNode(editingNode.value.id, {
            name: nodeForm.name,
            region: nodeForm.region,
            type: nodeForm.type,
            config: nodeForm.config,
            description: nodeForm.description,
            is_recommended: nodeForm.is_recommended,
            is_active: nodeForm.is_active
          })
        } else {
          response = await adminAPI.createNode({
            name: nodeForm.name,
            region: nodeForm.region,
            type: nodeForm.type,
            config: nodeForm.config,
            description: nodeForm.description,
            is_recommended: nodeForm.is_recommended,
            is_active: nodeForm.is_active
          })
        }

        if (response.data.success) {
          ElMessage.success(editingNode.value ? '节点更新成功' : '节点创建成功')
          showAddDialog.value = false
          resetForm()
          await loadNodes()
        } else {
          ElMessage.error(response.data.message || '保存失败')
        }
      } catch (error) {
        ElMessage.error('保存失败: ' + (error.response?.data?.message || error.message))
      } finally {
        saving.value = false
      }
    }

    const deleteNode = async (node) => {
      try {
        await ElMessageBox.confirm(
          `确定要删除节点 "${node.name}" 吗？`,
          '确认删除',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        const response = await adminAPI.deleteNode(node.id)
        if (response.data.success) {
          ElMessage.success('删除成功')
          await loadNodes()
        } else {
          ElMessage.error(response.data.message || '删除失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('删除失败: ' + (error.response?.data?.message || error.message))
        }
      }
    }

    const handleSelectionChange = (selection) => {
      selectedNodes.value = selection
    }

    const resetForm = () => {
      editingNode.value = null
      nodeForm.name = ''
      nodeForm.region = ''
      nodeForm.type = ''
      nodeForm.config = ''
      nodeForm.description = ''
      nodeForm.is_recommended = false
      nodeForm.is_active = true
      addNodeTab.value = 'link'
      nodeLinkInput.value = ''
      parsedNode.value = null
    }

    const getStatusType = (status) => {
      const statusMap = {
        online: 'success',
        offline: 'danger',
        timeout: 'warning'
      }
      return statusMap[status] || 'info'
    }

    const getStatusText = (status) => {
      const statusMap = {
        online: '在线',
        offline: '离线',
        timeout: '超时'
      }
      return statusMap[status] || status
    }

    const formatTime = (time) => {
      if (!time) return '-'
      const date = new Date(time)
      return date.toLocaleString('zh-CN')
    }

    // 解析节点链接（预览）
    const parseNodeLink = async () => {
      if (!nodeLinkInput.value.trim()) {
        ElMessage.warning('请输入节点链接')
        return
      }

      parsing.value = true
      try {
        // 取第一行作为预览
        const firstLink = nodeLinkInput.value.split('\n')[0].trim()
        if (!firstLink) {
          ElMessage.warning('请输入有效的节点链接')
          return
        }

        // 调用后端解析（预览模式，不实际创建）
        const response = await adminAPI.createNode({ node_link: firstLink, preview: true })
        if (response.data && response.data.success) {
          // 解析成功，显示预览信息
          const nodeData = response.data.data
          let server = ''
          let port = ''
          
          // 从 config JSON 中提取服务器和端口
          if (nodeData.config) {
            try {
              const configObj = typeof nodeData.config === 'string' ? JSON.parse(nodeData.config) : nodeData.config
              server = configObj.server || configObj.Server || ''
              port = configObj.port || configObj.Port || ''
            } catch (e) {
              console.error('解析配置失败:', e)
            }
          }
          
          parsedNode.value = {
            name: nodeData.name || '',
            type: nodeData.type || '',
            server: server,
            port: port,
            region: nodeData.region || ''
          }
          ElMessage.success('节点链接解析成功')
        } else {
          ElMessage.error(response.data?.message || '解析失败')
        }
      } catch (error) {
        ElMessage.error('解析失败: ' + (error.response?.data?.message || error.message))
      } finally {
        parsing.value = false
      }
    }

    // 从配置中提取服务器地址
    const extractServerFromConfig = (config) => {
      if (!config) return ''
      try {
        const configObj = typeof config === 'string' ? JSON.parse(config) : config
        return configObj.server || ''
      } catch {
        return ''
      }
    }

    // 从配置中提取端口
    const extractPortFromConfig = (config) => {
      if (!config) return ''
      try {
        const configObj = typeof config === 'string' ? JSON.parse(config) : config
        return configObj.port || ''
      } catch {
        return ''
      }
    }

    // 清空节点链接输入
    const clearNodeLink = () => {
      nodeLinkInput.value = ''
      parsedNode.value = null
    }

    // 从链接保存单个节点
    const saveNodeFromLink = async () => {
      if (!nodeLinkInput.value.trim()) {
        ElMessage.warning('请输入节点链接')
        return
      }

      const firstLink = nodeLinkInput.value.split('\n')[0].trim()
      if (!firstLink) {
        ElMessage.warning('请输入有效的节点链接')
        return
      }

      saving.value = true
      try {
        const response = await adminAPI.createNode({ node_link: firstLink })
        if (response.data && response.data.success) {
          ElMessage.success('节点添加成功')
          showAddDialog.value = false
          resetForm()
          await loadNodes()
        } else {
          ElMessage.error(response.data?.message || '添加失败')
        }
      } catch (error) {
        ElMessage.error('添加失败: ' + (error.response?.data?.message || error.message))
      } finally {
        saving.value = false
      }
    }

    // 批量导入节点链接
    const batchImportLinks = async () => {
      if (!nodeLinkInput.value.trim()) {
        ElMessage.warning('请输入节点链接')
        return
      }

      // 分割链接（每行一个）
      const links = nodeLinkInput.value
        .split('\n')
        .map(line => line.trim())
        .filter(line => line && (line.startsWith('vmess://') || 
                                 line.startsWith('vless://') || 
                                 line.startsWith('trojan://') || 
                                 line.startsWith('ss://') || 
                                 line.startsWith('ssr://') || 
                                 line.startsWith('hysteria://') || 
                                 line.startsWith('hysteria2://')))

      if (links.length === 0) {
        ElMessage.warning('未找到有效的节点链接')
        return
      }

      saving.value = true
      try {
        const response = await adminAPI.importNodeLinks(links)
        if (response.data && response.data.success) {
          const result = response.data
          ElMessage.success(
            `批量导入完成: 成功 ${result.imported} 个, 跳过 ${result.skipped} 个` +
            (result.error_count > 0 ? `, 失败 ${result.error_count} 个` : '')
          )
          if (result.errors && result.errors.length > 0) {
            console.warn('导入错误:', result.errors)
          }
          showAddDialog.value = false
          resetForm()
          await loadNodes()
        } else {
          ElMessage.error(response.data?.message || '批量导入失败')
        }
      } catch (error) {
        ElMessage.error('批量导入失败: ' + (error.response?.data?.message || error.message))
      } finally {
        saving.value = false
      }
    }

    // 取消添加节点
    const cancelAddNode = () => {
      showAddDialog.value = false
      resetForm()
    }

    onMounted(() => {
      loadNodes()
    })

    return {
      loading,
      testing,
      deleting,
      saving,
      nodes,
      selectedNodes,
      showAddDialog,
      editingNode,
      searchKeyword,
      filters,
      pagination,
      nodeForm,
      regions,
      types,
      loadNodes,
      testNode,
      batchTest,
      batchDelete,
      toggleNodeStatus,
      editNode,
      saveNode,
      deleteNode,
      handleSelectionChange,
      getStatusType,
      getStatusText,
      formatTime,
      addNodeTab,
      nodeLinkInput,
      parsedNode,
      parsing,
      parseNodeLink,
      clearNodeLink,
      saveNodeFromLink,
      batchImportLinks,
      cancelAddNode,
      extractServerFromConfig,
      extractPortFromConfig
    }
  }
}
</script>

<style scoped>
.admin-nodes {
  padding: 20px;
}

.filter-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  gap: 10px;
}

/* 确保输入框内部没有嵌套元素和装饰性边框 */
:deep(.el-form-item__content .el-input .el-input__wrapper) {
  box-shadow: 0 0 0 1px #dcdfe6 inset !important;
  border-radius: 0 !important;
  padding: 1px 11px;
  background: #fff;
}

:deep(.el-form-item__content .el-input .el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px #c0c4cc inset !important;
}

:deep(.el-form-item__content .el-input.is-focus .el-input__wrapper) {
  box-shadow: 0 0 0 1px #409eff inset !important;
}

:deep(.el-form-item__content .el-input .el-input__inner) {
  border: none !important;
  box-shadow: none !important;
  background: transparent !important;
  padding: 0;
  height: 32px;
  line-height: 32px;
}

/* 移除输入框内部的所有装饰性元素 */
:deep(.el-input__wrapper) {
  background: #fff;
  border-radius: 0 !important;
  box-shadow: 0 0 0 1px #dcdfe6 inset !important;
}

:deep(.el-input__wrapper::before),
:deep(.el-input__wrapper::after) {
  display: none !important;
}

:deep(.el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background: transparent !important;
}

:deep(.el-input__inner::-webkit-inner-spin-button),
:deep(.el-input__inner::-webkit-outer-spin-button) {
  -webkit-appearance: none;
  margin: 0;
}

:deep(.el-input__inner[type="number"]) {
  -moz-appearance: textfield;
  appearance: textfield;
}

:deep(.el-select .el-input__wrapper) {
  border-radius: 0 !important;
}

@media (max-width: 768px) {
  .desktop-only {
    display: none;
  }
  
  .admin-nodes {
    padding: 10px;
  }
  
  .filter-bar {
    flex-direction: column;
  }
  
  .filter-bar > * {
    width: 100% !important;
  }
}
</style>

