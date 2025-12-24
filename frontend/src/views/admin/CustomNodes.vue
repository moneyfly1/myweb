<template>
  <div class="list-container admin-custom-nodes">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>专线节点管理</span>
          <div class="header-actions desktop-only">
            <el-button type="primary" @click="showAddDialog = true">
              <el-icon><Plus /></el-icon>
              创建专线节点
            </el-button>
            <el-button @click="loadCustomNodes" :loading="loading">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <!-- 筛选栏 -->
      <div class="filter-bar">
        <el-select v-model="filters.status" placeholder="状态" clearable style="width: 120px" @change="loadCustomNodes">
          <el-option label="全部" value="" />
          <el-option label="活跃" value="active" />
          <el-option label="非活跃" value="inactive" />
          <el-option label="错误" value="error" />
        </el-select>
        <el-select v-model="filters.is_active" placeholder="激活状态" clearable style="width: 120px" @change="loadCustomNodes">
          <el-option label="全部" value="" />
          <el-option label="已激活" value="true" />
          <el-option label="已禁用" value="false" />
        </el-select>
        <el-input
          v-model="searchKeyword"
          placeholder="搜索节点名称"
          clearable
          style="width: 200px"
          @keyup.enter="loadCustomNodes"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <div v-if="selectedNodes.length > 0" class="batch-actions">
          <el-button type="danger" @click="batchDelete" :loading="batchDeleting">
            批量删除 ({{ selectedNodes.length }})
          </el-button>
          <el-button type="primary" @click="handleBatchAssignClick">
            批量分配 ({{ selectedNodes.length }})
          </el-button>
        </div>
      </div>

      <!-- 专线节点列表 -->
      <el-table
        :data="customNodes"
        v-loading="loading"
        stripe
        style="width: 100%"
        @selection-change="handleSelectionChange"
        row-key="id"
        empty-text="暂无专线节点数据"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="节点名称" min-width="150" />
        <el-table-column prop="display_name" label="显示名称" min-width="150">
          <template #default="{ row }">
            <span v-if="row.display_name">{{ row.display_name }}</span>
            <span v-else style="color: #909399">默认</span>
          </template>
        </el-table-column>
        <el-table-column prop="protocol" label="协议" width="100" />
        <el-table-column prop="domain" label="域名" min-width="150" />
        <el-table-column prop="port" label="端口" width="80" />
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
        <el-table-column label="到期时间" width="180">
          <template #default="{ row }">
            <span v-if="row.expire_time">{{ formatTime(row.expire_time) }}</span>
            <span v-else-if="row.follow_user_expire" style="color: #909399">跟随用户</span>
            <span v-else style="color: #909399">永久</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="520" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="testNode(row)" :loading="row.testing">测试</el-button>
            <el-button size="small" type="success" @click="viewLink(row)">链接</el-button>
            <el-button size="small" type="warning" @click="assignSingleNode(row)">分配</el-button>
            <el-button size="small" type="primary" @click="editNode(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="deleteNode(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加/编辑节点对话框 -->
    <el-dialog
      v-model="showAddDialog"
      :title="editingNode ? '编辑专线节点' : '添加专线节点'"
      :width="isMobile ? '95%' : '700px'"
      class="custom-node-dialog"
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
                <div style="margin-left: 10px; margin-bottom: 4px;">• <strong>Hysteria2:</strong> hysteria2://（Hysteria v2）</div>
                <div style="margin-left: 10px; margin-bottom: 4px;">• <strong>TUIC:</strong> tuic://（TUIC 协议）</div>
                <div style="margin-left: 10px; margin-bottom: 4px;">• <strong>Naive:</strong> naive+https:// 或 naive://（Naive 协议）</div>
                <div style="margin-left: 10px; margin-bottom: 4px;">• <strong>Anytls:</strong> anytls://（Anytls 协议）</div>
                <div style="margin-top: 8px; color: #909399; font-size: 12px;">
                  提示：支持单个链接或批量导入（每行一个链接），系统会自动解析并提取节点信息。专线节点可以分配给指定用户使用。
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
                支持格式：vmess://、vless://、trojan://、ss://、hysteria2:// 等
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
              </el-card>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="手动填写" name="manual">
      <el-form :model="nodeForm" label-width="140px" :rules="rules" ref="nodeFormRef">
            <el-form-item label="节点名称" prop="name">
              <el-input v-model="nodeForm.name" placeholder="请输入节点名称" />
            </el-form-item>
            <el-form-item label="显示名称" prop="display_name">
              <el-input v-model="nodeForm.display_name" placeholder="可选，留空则使用默认名称" />
              <div style="color: #909399; font-size: 12px; margin-top: 5px">在订阅中显示的节点名称，留空则使用"专线定制-节点名称"</div>
            </el-form-item>
            <el-form-item label="协议类型" prop="protocol">
              <el-select v-model="nodeForm.protocol" placeholder="请选择协议类型" style="width: 100%">
                <el-option label="vmess" value="vmess" />
                <el-option label="vless" value="vless" />
                <el-option label="trojan" value="trojan" />
                <el-option label="ss" value="ss" />
                <el-option label="hysteria2" value="hysteria2" />
                <el-option label="tuic" value="tuic" />
                <el-option label="naive" value="naive" />
                <el-option label="anytls" value="anytls" />
          </el-select>
        </el-form-item>
            <el-form-item label="配置(JSON)" prop="config">
              <el-input
                v-model="nodeForm.config"
                type="textarea"
                :rows="6"
                placeholder='请输入节点配置JSON，例如: {"server":"example.com","port":443,"uuid":"xxx"}'
              />
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
      
      <!-- 编辑模式直接显示表单 -->
      <el-form v-if="editingNode" :model="nodeForm" label-width="140px" :rules="rules" ref="nodeFormRef">
        <el-form-item label="节点名称" prop="name">
          <el-input v-model="nodeForm.name" placeholder="请输入节点名称" />
        </el-form-item>
        <el-form-item label="显示名称" prop="display_name">
          <el-input v-model="nodeForm.display_name" placeholder="可选，留空则使用默认名称" />
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

    <!-- 节点链接对话框 -->
    <el-dialog
      v-model="showLinkDialog"
      title="节点订阅链接"
      :width="isMobile ? '95%' : '700px'"
      class="node-link-dialog"
    >
      <div v-if="nodeLink">
        <el-alert
          :title="`节点: ${nodeLink.name}`"
          type="info"
          :closable="false"
          style="margin-bottom: 20px"
        />
        <el-form-item label="订阅链接">
          <el-input
            v-model="nodeLink.link"
            type="textarea"
            :rows="4"
            readonly
            style="font-family: monospace; font-size: 12px"
          />
        </el-form-item>
        <div style="margin-top: 10px">
          <el-button @click="copyLink" type="primary">复制链接</el-button>
          <el-button @click="testNodeFromLink" :loading="testingFromLink">测试节点</el-button>
        </div>
        <el-alert
          title="提示"
          type="warning"
          :closable="false"
          style="margin-top: 15px"
        >
          <template #default>
            <div style="font-size: 12px">
              <p>1. 此链接可直接导入到 V2Ray、Clash 等客户端使用</p>
              <p>2. 请妥善保管此链接，避免泄露</p>
              <p>3. 如果节点不可用，请检查节点配置是否正确</p>
            </div>
          </template>
        </el-alert>
      </div>
    </el-dialog>

    <!-- 分配对话框（单个/批量共用） -->
    <el-dialog
      v-model="showAssignDialog"
      :title="assignMode === 'single' ? '分配专线节点' : '批量分配专线节点'"
      :width="isMobile ? '95%' : '800px'"
      class="assign-node-dialog"
    >
      <el-alert
        :title="assignMode === 'single' ? `节点: ${assigningNode?.name || ''}` : `已选择 ${selectedNodes.length} 个节点`"
        type="info"
        :closable="false"
        style="margin-bottom: 20px"
      />
      <el-form label-width="120px" v-loading="loadingAssignedUsers">
        <!-- 已分配用户列表 -->
        <div v-if="assignMode === 'single'" class="assigned-users-section">
          <div class="section-title">该节点当前已分配给</div>
          <el-table :data="assignedUsers" size="small" stripe style="margin-bottom: 20px" empty-text="暂无分配记录">
            <el-table-column prop="username" label="用户名" />
            <el-table-column prop="email" label="邮箱" min-width="150" />
            <el-table-column label="订阅模式" width="120">
              <template #default="{ row }">
                <el-tag size="small" :type="row.special_node_subscription_type === 'special_only' ? 'warning' : 'success'">
                  {{ row.special_node_subscription_type === 'special_only' ? '仅专线' : '全部订阅' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="专线到期" width="160">
              <template #default="{ row }">
                <span :class="{ 'text-danger': isExpired(row.special_node_expires_at) }">
                  {{ formatTime(row.special_node_expires_at) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" fixed="right">
              <template #default="{ row }">
                <el-button type="danger" size="small" link @click="handleUnassign(row)">取消分配</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <div class="section-title" v-if="assignMode === 'single'">新增分配</div>
        <el-form-item label="选择用户" required>
          <el-select
            v-model="selectedUserIds"
            multiple
            filterable
            placeholder="请选择要分配的用户（可多选）"
            style="width: 100%"
            :loading="loadingUsers"
          >
            <el-option
              v-for="user in users"
              :key="user.id"
              :label="`${user.username} (${user.email})`"
              :value="user.id"
            />
          </el-select>
          <div style="color: #909399; font-size: 12px; margin-top: 5px">
            已选择 {{ selectedUserIds.length }} 个用户，将为每个用户分配 {{ assignMode === 'single' ? '1' : selectedNodes.length }} 个节点
          </div>
        </el-form-item>
        <el-form-item label="订阅模式">
          <el-radio-group v-model="assignExtraData.subscription_type">
            <el-radio label="both">全部订阅（普通+专线）</el-radio>
            <el-radio label="special_only">仅专线节点</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="专线到期时间">
          <el-date-picker
            v-model="assignExtraData.expires_at"
            type="datetime"
            placeholder="选择到期时间（可选）"
            style="width: 100%"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
          />
          <div style="color: #909399; font-size: 12px; margin-top: 5px">不选则跟随用户普通订阅到期时间</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAssignDialog = false">取消</el-button>
        <el-button type="primary" @click="handleAssign" :loading="batchAssigning" :disabled="selectedUserIds.length === 0">
          确定分配
        </el-button>
      </template>
    </el-dialog>

  </div>
</template>

<script>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Search } from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'

export default {
  name: 'AdminCustomNodes',
  components: {
    Plus,
    Refresh,
    Search
  },
  setup() {
    const isMobile = ref(window.innerWidth <= 768)
    const loading = ref(false)
    const saving = ref(false)
    const parsing = ref(false)
    const customNodes = ref([])
    
    const handleResize = () => {
      isMobile.value = window.innerWidth <= 768
    }
    const showAddDialog = ref(false)
    const showLinkDialog = ref(false)
    const editingNode = ref(null)
    const searchKeyword = ref('')
    const nodeFormRef = ref(null)
    const nodeLink = ref(null)
    const testingFromLink = ref(false)
    const selectedNodes = ref([])
    const selectedUserIds = ref([])
    const users = ref([])
    const loadingUsers = ref(false)
    const batchDeleting = ref(false)
    const batchAssigning = ref(false)
    const addNodeTab = ref('link')
    const nodeLinkInput = ref('')
    const parsedNode = ref(null)
    const showAssignDialog = ref(false)
    const assignMode = ref('single')
    const assigningNode = ref(null)
    const assignedUsers = ref([])
    const loadingAssignedUsers = ref(false)
    const assignExtraData = reactive({
      subscription_type: 'both',
      expires_at: null
    })

    const filters = reactive({
      status: '',
      is_active: ''
    })

    const nodeForm = reactive({
      name: '',
      display_name: '',
      protocol: '',
      config: '',
      expire_time: null,
      follow_user_expire: false
    })

    const rules = {
      name: [{ required: true, message: '请输入节点名称', trigger: 'blur' }]
    }

    const loadCustomNodes = async () => {
      loading.value = true
      try {
        const params = {}
        if (filters.status) params.status = filters.status
        if (filters.is_active) params.is_active = filters.is_active
        if (searchKeyword.value) params.search = searchKeyword.value

        const response = await adminAPI.getCustomNodes(params)
        console.log('获取节点列表响应:', response)
        if (response && response.data) {
          if (response.data.success !== false) {
            // 后端返回格式: { success: true, data: { data: [...], total: 10, page: 1, size: 20 } }
            const responseData = response.data.data
            if (responseData && responseData.data && Array.isArray(responseData.data)) {
              // 分页格式
              customNodes.value = responseData.data
              console.log('加载节点列表成功（分页格式）:', customNodes.value.length, '个节点')
            } else if (Array.isArray(responseData)) {
              // 直接数组格式（兼容）
              customNodes.value = responseData
              console.log('加载节点列表成功（数组格式）:', customNodes.value.length, '个节点')
            } else if (Array.isArray(response.data.data)) {
              // 另一种可能的格式
              customNodes.value = response.data.data
              console.log('加载节点列表成功（直接格式）:', customNodes.value.length, '个节点')
            } else {
              console.warn('无法解析节点列表数据:', responseData)
              customNodes.value = []
            }
          } else {
            ElMessage.error(response.data.message || '获取专线节点列表失败')
            customNodes.value = []
          }
        } else {
          ElMessage.error('获取专线节点列表失败: 响应格式错误')
          customNodes.value = []
        }
      } catch (error) {
        console.error('加载专线节点列表错误:', error)
        ElMessage.error('加载专线节点列表失败: ' + (error.response?.data?.message || error.message))
        customNodes.value = []
      } finally {
        loading.value = false
      }
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
        const response = await adminAPI.createCustomNode({ node_link: firstLink, preview: true })
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
            type: nodeData.type || nodeData.protocol || '',
            server: server,
            port: port
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
        const response = await adminAPI.createCustomNode({ node_link: firstLink })
        console.log('创建节点响应:', response)
        // 检查响应格式：可能是 { data: { success: true, data: {...} } } 或直接 { success: true, data: {...} }
        if (response && response.data) {
          const success = response.data.success !== false && response.data.success !== undefined
          if (success || response.status === 201 || response.status === 200) {
            ElMessage.success('专线节点添加成功')
            showAddDialog.value = false
            resetForm()
            // 延迟一下再刷新，确保数据库已提交
            setTimeout(async () => {
              await loadCustomNodes()
            }, 300)
          } else {
            ElMessage.error(response.data?.message || '添加失败')
          }
        } else {
          ElMessage.error('响应格式错误')
        }
      } catch (error) {
        console.error('创建节点错误:', error)
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
                                 line.startsWith('hysteria2://') ||
                                 line.startsWith('tuic://') ||
                                 line.startsWith('naive') ||
                                 line.startsWith('anytls://')))

      if (links.length === 0) {
        ElMessage.warning('未找到有效的节点链接')
        return
      }

      saving.value = true
      try {
        const response = await adminAPI.importCustomNodeLinks(links)
        if (response.data && response.data.success) {
          const result = response.data
          ElMessage.success(
            `批量导入完成: 成功 ${result.imported} 个` +
            (result.error_count > 0 ? `, 失败 ${result.error_count} 个` : '')
          )
          if (result.errors && result.errors.length > 0) {
            console.warn('导入错误:', result.errors)
          }
          showAddDialog.value = false
          resetForm()
          await loadCustomNodes()
        } else {
          ElMessage.error(response.data?.message || '批量导入失败')
        }
      } catch (error) {
        ElMessage.error('批量导入失败: ' + (error.response?.data?.message || error.message))
      } finally {
        saving.value = false
      }
    }

    const toggleNodeStatus = async (node) => {
      try {
        const response = await adminAPI.updateCustomNode(node.id, {
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
      nodeForm.display_name = node.display_name || ''
      nodeForm.expire_time = node.expire_time
      nodeForm.follow_user_expire = node.follow_user_expire
      showAddDialog.value = true
    }

    const resetForm = () => {
      editingNode.value = null
      nodeForm.name = ''
      nodeForm.display_name = ''
      nodeForm.protocol = ''
      nodeForm.config = ''
      nodeForm.expire_time = null
      nodeForm.follow_user_expire = false
      addNodeTab.value = 'link'
      nodeLinkInput.value = ''
      parsedNode.value = null
      if (nodeFormRef.value) {
        nodeFormRef.value.clearValidate()
      }
    }

    const saveNode = async () => {
      if (!nodeFormRef.value) return
      
      await nodeFormRef.value.validate(async (valid) => {
        if (!valid) return

        if (editingNode.value) {
          // 编辑模式，只检查节点名称
          if (!nodeForm.name) {
            ElMessage.warning('请填写节点名称')
            return
          }
        } else {
          // 创建模式（手动填写），检查所有必填项
          if (!nodeForm.name || !nodeForm.protocol || !nodeForm.config) {
            ElMessage.warning('请填写必填项')
            return
          }
        }

        saving.value = true
        try {
          let response
          if (editingNode.value) {
            // 编辑模式，只更新部分字段
            const updateData = {
              name: nodeForm.name,
              display_name: nodeForm.display_name || '', // 确保即使为空也发送，以便清空字段
              expire_time: nodeForm.expire_time,
              follow_user_expire: nodeForm.follow_user_expire
            }
            response = await adminAPI.updateCustomNode(editingNode.value.id, updateData)
          } else {
            // 创建模式（手动填写）
            response = await adminAPI.createCustomNode({
              name: nodeForm.name,
              display_name: nodeForm.display_name,
              protocol: nodeForm.protocol,
              config: nodeForm.config,
              expire_time: nodeForm.expire_time,
              follow_user_expire: nodeForm.follow_user_expire
            })
          }

          // 检查响应格式：可能是 { data: { success: true, data: {...} } } 或直接 { success: true, data: {...} }
          const success = (response.data && (response.data.success !== false && response.data.success !== undefined)) || response.status === 201 || response.status === 200
          if (success) {
            ElMessage.success(editingNode.value ? '节点更新成功' : '节点创建成功')
            showAddDialog.value = false
            resetForm()
            // 延迟一下再刷新，确保数据库已提交
            setTimeout(async () => {
              await loadCustomNodes()
            }, 300)
          } else {
            ElMessage.error(response.data?.message || '保存失败')
          }
        } catch (error) {
          ElMessage.error('保存失败: ' + (error.response?.data?.message || error.message))
        } finally {
          saving.value = false
        }
      })
    }

    const deleteNode = async (node) => {
      try {
        await ElMessageBox.confirm(
          `确定要删除专线节点 "${node.name}" 吗？`,
          '确认删除',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        const response = await adminAPI.deleteCustomNode(node.id)
        if (response.data.success) {
          ElMessage.success('删除成功')
          await loadCustomNodes()
        } else {
          ElMessage.error(response.data.message || '删除失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('删除失败: ' + (error.response?.data?.message || error.message))
        }
      }
    }

    const viewLink = async (node) => {
      try {
        const response = await adminAPI.getCustomNodeLink(node.id)
        if (response.data && response.data.success) {
          nodeLink.value = response.data.data
          showLinkDialog.value = true
        } else {
          ElMessage.error(response.data?.message || '获取节点链接失败')
        }
      } catch (error) {
        ElMessage.error('获取节点链接失败: ' + (error.response?.data?.message || error.message))
      }
    }

    const copyLink = () => {
      if (nodeLink.value && nodeLink.value.link) {
        navigator.clipboard.writeText(nodeLink.value.link).then(() => {
          ElMessage.success('链接已复制到剪贴板')
        }).catch(() => {
          // 降级方案
          const textarea = document.createElement('textarea')
          textarea.value = nodeLink.value.link
          document.body.appendChild(textarea)
          textarea.select()
          document.execCommand('copy')
          document.body.removeChild(textarea)
          ElMessage.success('链接已复制到剪贴板')
        })
      }
    }

    const testNode = async (node) => {
      if (!node.testing) {
        node.testing = true
      }
      try {
        const response = await adminAPI.testCustomNode(node.id)
        if (response.data && response.data.success) {
          const result = response.data.data
          const statusText = {
            online: '在线',
            offline: '离线',
            timeout: '超时'
          }[result.status] || result.status
          const latencyText = result.latency > 0 ? `${result.latency}ms` : '超时'
          ElMessage.success(`测试完成: ${statusText}, 延迟: ${latencyText}`)
          // 更新节点状态
          node.status = result.status
          // 刷新列表
          await loadCustomNodes()
        } else {
          ElMessage.error(response.data?.message || '测试失败')
        }
      } catch (error) {
        ElMessage.error('测试失败: ' + (error.response?.data?.message || error.message))
      } finally {
        node.testing = false
      }
    }

    const testNodeFromLink = async () => {
      if (!nodeLink.value) return
      const nodeId = nodeLink.value.id || nodeLink.value.node_id
      if (!nodeId) {
        ElMessage.error('无法获取节点ID')
        return
      }
      testingFromLink.value = true
      try {
        const response = await adminAPI.testCustomNode(nodeId)
        if (response.data && response.data.success) {
          const result = response.data.data
          const statusText = {
            online: '在线',
            offline: '离线',
            timeout: '超时'
          }[result.status] || result.status
          const latencyText = result.latency > 0 ? `${result.latency}ms` : '超时'
          ElMessage.success(`测试完成: ${statusText}, 延迟: ${latencyText}`)
        } else {
          ElMessage.error(response.data?.message || '测试失败')
        }
      } catch (error) {
        ElMessage.error('测试失败: ' + (error.response?.data?.message || error.message))
      } finally {
        testingFromLink.value = false
      }
    }

    const cancelAddNode = () => {
      showAddDialog.value = false
      resetForm()
    }

    const getStatusType = (status) => {
      const statusMap = {
        active: 'success',
        inactive: 'info',
        error: 'danger'
      }
      return statusMap[status] || 'info'
    }
    const getStatusText = (status) => {
      const statusMap = {
        active: '活跃',
        inactive: '非活跃',
        error: '错误'
      }
      return statusMap[status] || status
    }

    const formatTime = (time) => {
      if (!time) return '-'
      const date = new Date(time)
      return date.toLocaleString('zh-CN')
    }

    const handleSelectionChange = (selection) => {
      selectedNodes.value = selection
    }

    const batchDelete = async () => {
      if (selectedNodes.value.length === 0) {
        ElMessage.warning('请选择要删除的节点')
        return
      }
      try {
        await ElMessageBox.confirm(
          `确定要删除选中的 ${selectedNodes.value.length} 个专线节点吗？此操作不可恢复！`,
          '确认批量删除',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        batchDeleting.value = true
        const nodeIds = selectedNodes.value.map(node => node.id)
        const response = await adminAPI.batchDeleteCustomNodes(nodeIds)
        if (response.data.success) {
          ElMessage.success('批量删除成功')
          selectedNodes.value = []
          await loadCustomNodes()
        } else {
          ElMessage.error(response.data.message || '批量删除失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('批量删除失败: ' + (error.response?.data?.message || error.message))
        }
      } finally {
        batchDeleting.value = false
      }
    }

    const loadUsers = async () => {
      if (users.value.length > 0) return
      loadingUsers.value = true
      try {
        const response = await adminAPI.getUsers({ page: 1, size: 1000 })
        if (response.data && response.data.success) {
          users.value = response.data.data?.users || response.data.data || []
        }
      } catch (error) {
        console.error('加载用户列表失败:', error)
        ElMessage.error('加载用户列表失败')
      } finally {
        loadingUsers.value = false
      }
    }

    const loadAssignedUsers = async (nodeId) => {
      loadingAssignedUsers.value = true
      try {
        const response = await adminAPI.getCustomNodeUsers(nodeId)
        if (response.data && response.data.success) {
          assignedUsers.value = response.data.data || []
        } else {
          ElMessage.error(response.data?.message || '加载已分配用户失败')
        }
      } catch (error) {
        console.error('加载已分配用户失败:', error)
        if (error.response?.status === 404) {
          ElMessage.warning('API 接口未找到，请确保后端已更新并重启')
        } else {
          ElMessage.error('加载已分配用户失败: ' + (error.response?.data?.message || error.message))
        }
      } finally {
        loadingAssignedUsers.value = false
      }
    }

    const isExpired = (time) => {
      if (!time) return false
      return new Date(time) < new Date()
    }

    const handleUnassign = async (user) => {
      try {
        await ElMessageBox.confirm(`确定要为用户 ${user.username} 取消分配此节点吗？`, '确认操作', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        const response = await adminAPI.unassignCustomNodeFromUser(user.id, assigningNode.value.id)
        if (response.data.success) {
          ElMessage.success('取消分配成功')
          await loadAssignedUsers(assigningNode.value.id)
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('操作失败')
        }
      }
    }

    const openAssignDialog = (mode, node = null) => {
      assignMode.value = mode
      assigningNode.value = node
      selectedUserIds.value = []
      assignedUsers.value = []
      assignExtraData.subscription_type = 'both'
      assignExtraData.expires_at = null
      loadUsers()
      if (mode === 'single' && node) {
        loadAssignedUsers(node.id)
      }
      showAssignDialog.value = true
    }
    const assignSingleNode = (node) => {
      openAssignDialog('single', node)
    }
    const handleBatchAssignClick = () => {
      if (selectedNodes.value.length === 0) {
        ElMessage.warning('请选择要分配的节点')
        return
      }
      openAssignDialog('batch')
    }
    const handleAssign = async () => {
      if (selectedUserIds.value.length === 0) {
        ElMessage.warning('请选择要分配的用户')
        return
      }
      batchAssigning.value = true
      try {
        let nodeIds = []
        if (assignMode.value === 'single') {
          if (!assigningNode.value) {
            ElMessage.error('节点信息不存在')
            return
          }
          nodeIds = [assigningNode.value.id]
        } else {
          if (selectedNodes.value.length === 0) {
            ElMessage.warning('请选择要分配的节点')
            return
          }
          nodeIds = selectedNodes.value.map(node => node.id)
        }
        const response = await adminAPI.batchAssignCustomNodes(nodeIds, selectedUserIds.value, {
          subscription_type: assignExtraData.subscription_type,
          expires_at: assignExtraData.expires_at
        })
        if (response.data.success) {
          const nodeCount = assignMode.value === 'single' ? 1 : selectedNodes.value.length
          ElMessage.success(`成功为 ${selectedUserIds.value.length} 个用户分配了 ${nodeCount} 个节点`)
          showAssignDialog.value = false
          if (assignMode.value === 'batch') {
            selectedNodes.value = []
          } else {
            await loadAssignedUsers(assigningNode.value.id)
          }
          selectedUserIds.value = []
          assigningNode.value = null
          await loadCustomNodes()
        } else {
          ElMessage.error(response.data.message || '分配失败')
        }
      } catch (error) {
        ElMessage.error('分配失败: ' + (error.response?.data?.message || error.message))
      } finally {
        batchAssigning.value = false
      }
    }

    onMounted(() => {
      loadCustomNodes()
      loadUsers()
      window.addEventListener('resize', handleResize)
    })
    
    onUnmounted(() => {
      window.removeEventListener('resize', handleResize)
    })

    return {
      isMobile,
      loading,
      saving,
      parsing,
      customNodes,
      showAddDialog,
      addNodeTab,
      nodeLinkInput,
      parsedNode,
      showLinkDialog,
      showAssignDialog,
      assignMode,
      assigningNode,
      assignedUsers,
      loadingAssignedUsers,
      assignExtraData,
      editingNode,
      searchKeyword,
      filters,
      nodeForm,
      nodeFormRef,
      rules,
      nodeLink,
      testingFromLink,
      selectedNodes,
      selectedUserIds,
      users,
      loadingUsers,
      batchDeleting,
      batchAssigning,
      loadCustomNodes,
      toggleNodeStatus,
      parseNodeLink,
      clearNodeLink,
      saveNodeFromLink,
      batchImportLinks,
      resetForm,
      editNode,
      saveNode,
      deleteNode,
      viewLink,
      copyLink,
      testNode,
      testNodeFromLink,
      cancelAddNode,
      getStatusType,
      getStatusText,
      formatTime,
      isExpired,
      handleSelectionChange,
      batchDelete,
      assignSingleNode,
      handleBatchAssignClick,
      handleAssign,
      handleUnassign,
      loadUsers,
    }
  }
}
</script>

<style scoped>
.admin-custom-nodes {
  padding: 20px;
}

.filter-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
  flex-wrap: wrap;
  align-items: center;
}

.batch-actions {
  display: flex;
  gap: 10px;
  margin-left: auto;
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

.assigned-users-section {
  margin-bottom: 30px;
  padding: 15px;
  background: #f8f9fa;
  border: 1px solid #ebeef5;
}

.section-title {
  font-size: 14px;
  font-weight: bold;
  margin-bottom: 15px;
  color: #606266;
  border-left: 4px solid #409eff;
  padding-left: 10px;
}

.text-danger {
  color: #f56c6c;
}

/* 移除输入框圆角 */
:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: 0 0 0 1px #dcdfe6 inset !important;
}

:deep(.el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background: transparent !important;
}

:deep(.el-input__wrapper::before),
:deep(.el-input__wrapper::after) {
  display: none !important;
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

/* 桌面端/移动端显示控制 */
.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}

.mobile-only {
  display: none;
  @media (max-width: 768px) {
    display: block;
  }
}

@media (max-width: 768px) {
  .admin-custom-nodes {
    padding: 10px;
  }
  
  .filter-bar {
    flex-direction: column;
    gap: 10px;
  }
  
  .filter-bar > * {
    width: 100% !important;
  }
  
  /* 对话框优化 */
  :deep(.custom-node-dialog),
  :deep(.node-link-dialog),
  :deep(.assign-node-dialog) {
    .el-dialog {
      width: 95% !important;
      margin: 2vh auto !important;
      max-height: 96vh;
    }
    
    .el-dialog__body {
      padding: 15px !important;
      max-height: calc(96vh - 140px);
      overflow-y: auto;
    }
    
    .el-dialog__footer {
      padding: 12px 15px 15px;
      
      .el-button {
        width: 100%;
        margin: 0 0 10px 0 !important;
        min-height: 44px;
        font-size: 16px;
        
        &:last-child {
          margin-bottom: 0;
        }
      }
    }
  }
  
  /* 表单优化 */
  :deep(.el-form-item) {
    margin-bottom: 18px;
    
    .el-form-item__label {
      font-size: 14px;
      margin-bottom: 8px;
    }
  }
  
  /* 表格优化 */
  .table-wrapper {
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    
    :deep(.el-table) {
      min-width: 800px;
      font-size: 12px;
      
      .el-table__cell {
        padding: 8px 4px;
      }
    }
  }
}
</style>
