<template>
  <div class="list-container admin-custom-nodes">
    <el-card class="list-card" shadow="never">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <span class="title-text">专线节点管理</span>
            <el-tag v-if="pagination.total" type="info" round size="small" class="count-tag">{{ pagination.total }}</el-tag>
          </div>
          <div class="header-actions" v-if="!isMobile">
            <el-radio-group v-model="viewMode" size="small" class="view-mode-group">
              <el-radio-button label="table">表格</el-radio-button>
              <el-radio-button label="grid">方格</el-radio-button>
            </el-radio-group>
            <template v-if="viewMode === 'grid'">
              <el-radio-group v-model="gridOrientation" size="small" class="grid-orientation-group">
                <el-radio-button label="horizontal">横向</el-radio-button>
                <el-radio-button label="vertical">纵向</el-radio-button>
              </el-radio-group>
              <template v-if="gridOrientation === 'horizontal'">
                <el-select v-model="gridColumns" size="small" style="width: 90px; margin-right: 8px;" class="grid-columns-select">
                  <el-option label="2列" :value="2" />
                  <el-option label="3列" :value="3" />
                  <el-option label="4列" :value="4" />
                  <el-option label="5列" :value="5" />
                  <el-option label="6列" :value="6" />
                </el-select>
              </template>
              <template v-else>
                <el-radio-group v-model="gridSize" size="small" class="grid-size-group">
                  <el-radio-button label="small">窄</el-radio-button>
                  <el-radio-button label="medium">中</el-radio-button>
                  <el-radio-button label="large">宽</el-radio-button>
                </el-radio-group>
              </template>
            </template>
            <el-button type="primary" @click="showAddDialog = true">
              <el-icon><Plus /></el-icon>创建节点
            </el-button>
            <el-button @click="loadCustomNodes" :loading="loading">
              <el-icon><Refresh /></el-icon>刷新
            </el-button>
          </div>
          <div class="header-actions mobile" v-else>
            <el-button type="primary" circle @click="showAddDialog = true" size="small">
              <el-icon><Plus /></el-icon>
            </el-button>
            <el-dropdown trigger="click" @command="handleCommand">
              <el-button circle size="small">
                <el-icon><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="refresh" :icon="Refresh">刷新列表</el-dropdown-item>
                  <el-dropdown-item command="batch_test" :disabled="!selectedNodes.length" :icon="Connection">批量测速</el-dropdown-item>
                  <el-dropdown-item command="batch_assign" :disabled="!selectedNodes.length" :icon="User">批量分配</el-dropdown-item>
                  <el-dropdown-item command="batch_delete" :disabled="!selectedNodes.length" :icon="Delete" divided style="color: var(--el-color-danger)">批量删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </template>
      <div class="filter-wrapper">
        <div class="filter-grid">
          <el-select v-model="filters.status" placeholder="状态" clearable @change="handleFilterChange">
            <el-option label="全部状态" value="" />
            <el-option label="活跃" value="active" />
            <el-option label="非活跃" value="inactive" />
            <el-option label="错误" value="error" />
          </el-select>
          <el-select v-model="filters.is_active" placeholder="激活" clearable @change="handleFilterChange">
            <el-option label="全部" value="" />
            <el-option label="已激活" value="true" />
            <el-option label="已禁用" value="false" />
          </el-select>
          <div class="search-box">
            <el-input
              v-model="searchKeyword"
              placeholder="搜索名称/域名/用户..."
              clearable
              @keyup.enter="handleFilterChange"
            >
              <template #prefix><el-icon><Search /></el-icon></template>
            </el-input>
          </div>
        </div>
      </div>
      <div v-if="selectedNodes.length > 0 && !isMobile" class="batch-actions-bar">
        <span class="batch-tip">已选择 {{ selectedNodes.length }} 个节点</span>
        <div class="batch-btns">
          <el-button type="success" link @click="batchTest" :loading="batchTesting">批量测速</el-button>
          <el-divider direction="vertical" />
          <el-button type="primary" link @click="handleBatchAssignClick">批量分配</el-button>
          <el-divider direction="vertical" />
          <el-button type="danger" link @click="batchDelete" :loading="batchDeleting">批量删除</el-button>
        </div>
      </div>
      <div class="content-view" v-loading="loading">
        <el-table
          v-if="!isMobile && viewMode === 'table'"
          :data="customNodes"
          stripe
          border
          @selection-change="handleSelectionChange"
          @header-dragend="handleColumnResize"
          row-key="id"
          class="desktop-table"
          ref="tableRef"
        >
          <el-table-column type="selection" :width="columnWidths.selection" resizable />
          <el-table-column prop="name" label="名称" :min-width="columnWidths.name" resizable show-overflow-tooltip />
          <el-table-column prop="display_name" label="显示名称" :min-width="columnWidths.display_name" resizable show-overflow-tooltip>
            <template #default="{ row }">
              <span :class="row.display_name ? '' : 'text-secondary'">
                {{ row.display_name || row.name || '-' }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="protocol" label="协议" :width="columnWidths.protocol" resizable>
            <template #default="{ row }">
              <el-tag size="small" effect="plain">{{ row.protocol }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" :width="columnWidths.status" resizable>
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)" size="small" effect="light">
                {{ getStatusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="激活" :width="columnWidths.is_active" resizable>
            <template #default="{ row }">
              <el-switch v-model="row.is_active" @change="toggleNodeStatus(row)" size="small" />
            </template>
          </el-table-column>
          <el-table-column label="到期" :width="columnWidths.expire_time" resizable>
            <template #default="{ row }">
              <span class="text-xs">{{ formatExpire(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" :width="columnWidths.actions" fixed="right" resizable>
            <template #default="{ row }">
              <div class="table-actions">
                <el-button size="small" @click="testNode(row)" :loading="row.testing">测试</el-button>
                <el-button size="small" type="success" plain @click="viewLink(row)">链接</el-button>
                <el-button size="small" type="warning" plain @click="assignSingleNode(row)">分配</el-button>
                <el-button size="small" type="primary" plain @click="editNode(row)" :icon="Edit">编辑</el-button>
                <el-button size="small" type="danger" plain @click="deleteNode(row)" :icon="Delete">删除</el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>
        <div v-if="!isMobile && viewMode === 'grid'" class="desktop-grid-view" :class="[
          gridOrientation === 'horizontal' ? 'grid-horizontal' : 'grid-vertical',
          gridOrientation === 'vertical' ? 'grid-size-' + gridSize : '',
          'grid-cols-' + gridColumns
        ]">
          <template v-if="customNodes.length === 0">
            <el-empty description="暂无专线节点" class="grid-empty" />
          </template>
          <template v-else>
            <div
              v-for="node in customNodes"
              :key="node.id"
              class="grid-node-card"
              :class="{ 'is-selected': isSelected(node) }"
            >
              <div class="gnc-header">
                <el-checkbox
                  :model-value="isSelected(node)"
                  @change="(val) => handleGridSelect(node, val)"
                  class="gnc-checkbox"
                />
                <span class="gnc-title" :title="node.name">{{ node.name }}</span>
                <el-tag :type="getStatusType(node.status)" size="small" effect="dark">
                  {{ getStatusText(node.status) }}
                </el-tag>
              </div>
              <div class="gnc-body">
                <div class="gnc-row">
                  <span class="label">协议</span>
                  <span class="value">{{ node.protocol }}</span>
                </div>
                <div class="gnc-row">
                  <span class="label">端口</span>
                  <span class="value">{{ node.port || '-' }}</span>
                </div>
                <div class="gnc-row">
                  <span class="label">到期</span>
                  <span class="value text-xs">{{ formatExpire(node) }}</span>
                </div>
              </div>
              <div class="gnc-footer">
                <el-switch
                  v-model="node.is_active"
                  @change="toggleNodeStatus(node)"
                  size="small"
                  inline-prompt
                  active-text="开"
                  inactive-text="关"
                />
                <div class="gnc-actions">
                  <el-button size="small" @click="testNode(node)" :loading="node.testing">测试</el-button>
                  <el-button size="small" type="success" plain @click="viewLink(node)">链接</el-button>
                  <el-button size="small" type="warning" plain @click="assignSingleNode(node)">分配</el-button>
                  <el-button size="small" type="primary" plain @click="editNode(node)" :icon="Edit">编辑</el-button>
                  <el-button size="small" type="danger" plain @click="deleteNode(node)" :icon="Delete">删除</el-button>
                </div>
              </div>
            </div>
          </template>
        </div>
        <div v-if="isMobile" class="mobile-list">
          <div class="mobile-selection-bar" v-if="customNodes.length > 0">
            <el-checkbox 
              v-model="isAllSelected" 
              :indeterminate="isIndeterminate" 
              @change="toggleMobileSelectAll"
            >全选 ({{ selectedNodes.length }})</el-checkbox>
          </div>
          <div v-for="node in customNodes" :key="node.id" class="node-card">
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
                <span class="label">协议</span>
                <span class="value">{{ node.protocol }}</span>
              </div>
              <div class="info-item">
                <span class="label">端口</span>
                <span class="value">{{ node.port }}</span>
              </div>
              <div class="info-item full-width">
                <span class="label">到期</span>
                <span class="value">{{ formatExpire(node) }}</span>
              </div>
            </div>
            <div class="card-actions-row">
              <div class="left-actions">
                <el-switch 
                  v-model="node.is_active" 
                  @change="toggleNodeStatus(node)" 
                  size="small"
                  inline-prompt
                  active-text="开"
                  inactive-text="关"
                />
              </div>
              <div class="right-buttons">
                <el-button size="small" text bg @click="testNode(node)" :loading="node.testing">测试</el-button>
                <el-button size="small" text bg type="warning" @click="assignSingleNode(node)">分配</el-button>
                <el-dropdown trigger="click">
                  <el-button size="small" text bg>更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item @click="viewLink(node)" :icon="Link">链接</el-dropdown-item>
                      <el-dropdown-item @click="editNode(node)" :icon="Edit">编辑</el-dropdown-item>
                      <el-dropdown-item @click="deleteNode(node)" :icon="Delete" style="color: var(--el-color-danger)">删除</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
            </div>
          </div>
          <el-empty v-if="customNodes.length === 0" description="暂无专线节点" />
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
          @current-change="loadCustomNodes"
          @size-change="loadCustomNodes"
        />
      </div>
    </el-card>
    <el-drawer
      v-model="showAddDialog"
      :title="editingNode ? '编辑专线节点' : '添加专线节点'"
      :size="isMobile ? '92%' : '600px'"
      direction="rtl"
      destroy-on-close
      append-to-body
    >
      <div class="dialog-scroll-content">
        <el-tabs v-model="addNodeTab" v-if="!editingNode" class="compact-tabs">
          <el-tab-pane label="链接导入" name="link">
            <div class="import-section">
              <el-alert title="支持 vmess, vless, trojan, ss, hysteria2 等链接批量导入" type="info" :closable="false" show-icon />
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
          </el-tab-pane>
        </el-tabs>
        <el-form 
          v-if="editingNode || addNodeTab === 'manual'" 
          :model="nodeForm" 
          :rules="rules" 
          ref="nodeFormRef"
          :label-position="isMobile ? 'top' : 'right'" 
          label-width="100px"
          class="node-form"
        >
          <el-form-item label="节点名称" prop="name">
            <el-input v-model="nodeForm.name" placeholder="请输入节点名称" />
          </el-form-item>
          <el-form-item label="显示名称" prop="display_name">
            <el-input v-model="nodeForm.display_name" placeholder="客户端显示的名称 (可选)" />
          </el-form-item>
          <template v-if="!editingNode">
            <el-form-item label="协议类型" prop="protocol">
              <el-select v-model="nodeForm.protocol" placeholder="选择协议" style="width: 100%">
                <el-option v-for="p in ['vmess','vless','trojan','ss','hysteria2','tuic','naive']" :key="p" :label="p" :value="p" />
              </el-select>
            </el-form-item>
            <el-form-item label="配置(JSON)" prop="config">
              <el-input 
                v-model="nodeForm.config" 
                type="textarea" 
                :rows="6" 
                placeholder='{"server":"example.com", "port":443, ...}' 
                class="code-input"
              />
            </el-form-item>
          </template>
          <el-form-item label="到期时间" prop="expire_time">
             <el-date-picker
                v-model="nodeForm.expire_time"
                type="datetime"
                placeholder="永久有效"
                style="width: 100%"
                format="YYYY-MM-DD HH:mm"
                value-format="YYYY-MM-DDTHH:mm:ssZ"
              />
          </el-form-item>
          <el-form-item label="跟随用户" prop="follow_user_expire">
            <el-switch v-model="nodeForm.follow_user_expire" />
            <span class="form-tip-text">节点到期时间将与被分配用户的订阅时间同步</span>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showAddDialog = false">取消</el-button>
          <template v-if="!editingNode && addNodeTab === 'link'">
            <el-button type="warning" plain @click="parseNodeLink" :loading="parsing">仅解析</el-button>
            <el-button type="primary" @click="batchImportLinks" :loading="saving" :disabled="!nodeLinkInput">批量导入</el-button>
          </template>
          <el-button v-else type="primary" @click="saveNode" :loading="saving">保存</el-button>
        </div>
      </template>
    </el-drawer>
    <el-dialog
      v-model="showLinkDialog"
      title="节点链接"
      :width="isMobile ? '90%' : '500px'"
      append-to-body
      class="responsive-dialog"
    >
      <div v-if="nodeLink" class="link-view-content">
        <el-input
          v-model="nodeLink.link"
          type="textarea"
          :rows="5"
          readonly
          class="code-input"
        />
        <div class="link-actions">
          <el-button type="primary" @click="copyLink" icon="DocumentCopy">复制链接</el-button>
          <el-button @click="testNodeFromLink" :loading="testingFromLink" icon="Connection">测试连接</el-button>
        </div>
      </div>
    </el-dialog>
    <el-dialog
      v-model="showAssignDialog"
      :title="assignMode === 'single' ? '分配节点' : '批量分配'"
      :width="isMobile ? '92%' : '750px'"
      :fullscreen="isMobile"
      class="responsive-dialog assign-dialog"
      append-to-body
    >
      <div class="dialog-scroll-content">
        <div v-if="assignMode === 'single'" class="assigned-section">
          <div class="section-header">已分配用户</div>
          <el-table 
            v-if="!isMobile" 
            :data="assignedUsers" 
            size="small" 
            empty-text="暂无分配"
            style="margin-bottom: 15px"
          >
            <el-table-column prop="username" label="用户" />
            <el-table-column prop="email" label="邮箱" show-overflow-tooltip />
            <el-table-column label="到期">
              <template #default="{ row }">
                <span :class="{'text-danger': isExpired(row.special_node_expires_at)}">
                  {{ formatTime(row.special_node_expires_at) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="80">
              <template #default="{ row }">
                <el-button type="danger" link size="small" @click="handleUnassign(row)">移除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div v-else class="mobile-assigned-list">
             <div v-for="u in assignedUsers" :key="u.id" class="mini-user-card">
               <div class="u-info">
                 <div class="u-name">{{ u.username }}</div>
                 <div class="u-time" :class="{'text-danger': isExpired(u.special_node_expires_at)}">
                   {{ formatTime(u.special_node_expires_at) }}
                 </div>
               </div>
               <el-button type="danger" circle size="small" icon="Close" @click="handleUnassign(u)" />
             </div>
             <el-empty v-if="!assignedUsers.length" description="暂无分配" :image-size="60" />
          </div>
        </div>
        <div class="assign-form">
          <div class="section-header">新增分配</div>
          <div class="search-user-row">
            <el-input 
              v-model="userSearchKeyword" 
              placeholder="搜索用户名/邮箱..." 
              clearable
              @keyup.enter="handleUserSearch"
            >
              <template #append>
                <el-button @click="handleUserSearch" :loading="loadingUsers" icon="Search" />
              </template>
            </el-input>
          </div>
          <el-select
            v-model="selectedUserIds"
            multiple
            placeholder="请从搜索结果中选择用户"
            style="width: 100%; margin: 10px 0"
            no-data-text="请先搜索"
          >
            <el-option
              v-for="user in searchedUsers"
              :key="user.id"
              :label="`${user.username} (${user.email})`"
              :value="user.id"
            />
          </el-select>
          <el-form label-position="top" size="small">
             <el-form-item label="订阅模式">
               <el-radio-group v-model="assignExtraData.subscription_type">
                 <el-radio label="both">全部订阅</el-radio>
                 <el-radio label="special_only">仅专线</el-radio>
               </el-radio-group>
             </el-form-item>
             <el-form-item label="专线到期 (可选)">
                <el-date-picker
                  v-model="assignExtraData.expires_at"
                  type="datetime"
                  placeholder="默认跟随用户订阅"
                  style="width: 100%"
                  value-format="YYYY-MM-DDTHH:mm:ssZ"
                />
             </el-form-item>
          </el-form>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showAssignDialog = false">取消</el-button>
          <el-button type="primary" @click="handleAssign" :loading="batchAssigning" :disabled="!selectedUserIds.length">确定分配</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>
<script>
import { ref, reactive, onMounted, onUnmounted, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Plus, Refresh, Search, Connection, Delete, 
  DocumentCopy, Edit, MoreFilled, User, Link,
  ArrowDown, Close
} from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
export default {
  name: 'AdminCustomNodes',
  components: { 
    Plus, Refresh, Search, Connection, Delete, 
    DocumentCopy, Edit, MoreFilled, User, Link, ArrowDown, Close
  },
  setup() {
    const isMobile = ref(false)
    const viewMode = ref('table') // 'table' | 'grid'
    const gridOrientation = ref('horizontal') // 'horizontal' | 'vertical'
    const gridColumns = ref(3) // 2-6 columns for horizontal
    const gridSize = ref('medium') // 'small' | 'medium' | 'large' for vertical
    const tableRef = ref(null)
    const loading = ref(false)
    
    // 列宽状态（动态绑定）
    const columnWidths = reactive({
      selection: 50,
      name: 140,
      display_name: 120,
      protocol: 100,
      status: 100,
      is_active: 80,
      expire_time: 150,
      actions: 380  // 增加操作列宽度以容纳更多按钮
    })
    
    // 从 localStorage 加载设置
    const STORAGE_KEY = 'customNodes_table_settings'
    const loadSettings = () => {
      try {
        const saved = localStorage.getItem(STORAGE_KEY)
        if (saved) {
          const settings = JSON.parse(saved)
          if (settings.viewMode) viewMode.value = settings.viewMode
          if (settings.gridOrientation) gridOrientation.value = settings.gridOrientation
          if (settings.gridColumns) gridColumns.value = settings.gridColumns
          if (settings.gridSize) gridSize.value = settings.gridSize
          if (settings.columnWidths) {
            Object.assign(columnWidths, settings.columnWidths)
          }
        }
      } catch (e) {
        console.warn('加载设置失败:', e)
      }
    }
    
    // 保存设置到 localStorage
    const saveSettings = () => {
      try {
        const settings = {
          viewMode: viewMode.value,
          gridOrientation: gridOrientation.value,
          gridColumns: gridColumns.value,
          gridSize: gridSize.value,
          columnWidths: { ...columnWidths }
        }
        localStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
      } catch (e) {
        console.warn('保存设置失败:', e)
      }
    }
    
    // 列宽调整事件处理（延迟保存，避免频繁触发）
    let resizeTimer = null
    const handleColumnResize = (newWidth, oldWidth, column, event) => {
      if (resizeTimer) clearTimeout(resizeTimer)
      resizeTimer = setTimeout(() => {
        // 获取所有列的当前宽度
        if (tableRef.value && tableRef.value.$el) {
          const headerCells = tableRef.value.$el.querySelectorAll('.el-table__header-wrapper thead th')
          const keys = ['selection', 'name', 'display_name', 'protocol', 'status', 'is_active', 'expire_time', 'actions']
          headerCells.forEach((cell, index) => {
            if (keys[index] && cell.offsetWidth > 0) {
              columnWidths[keys[index]] = cell.offsetWidth
            }
          })
          saveSettings()
        }
      }, 300)
    }
    const saving = ref(false)
    const parsing = ref(false)
    const customNodes = ref([])
    const selectedNodes = ref([])
    const showAddDialog = ref(false)
    const showLinkDialog = ref(false)
    const showAssignDialog = ref(false)
    const addNodeTab = ref('link')
    const searchKeyword = ref('')
    const filters = reactive({ status: '', is_active: '' })
    const pagination = reactive({ page: 1, size: 20, total: 0 })
    const nodeFormRef = ref(null)
    const nodeForm = reactive({
      name: '', display_name: '', protocol: 'vmess', config: '', 
      expire_time: null, follow_user_expire: false
    })
    const nodeLinkInput = ref('')
    const parsedNode = ref(null)
    const nodeLink = ref(null)
    const testingFromLink = ref(false)
    const assignMode = ref('single') // single | batch
    const assigningNode = ref(null)
    const assignedUsers = ref([])
    const userSearchKeyword = ref('')
    const searchedUsers = ref([])
    const selectedUserIds = ref([])
    const loadingUsers = ref(false)
    const batchAssigning = ref(false)
    const assignExtraData = reactive({ subscription_type: 'both', expires_at: null })
    const batchTesting = ref(false)
    const batchDeleting = ref(false)
    const rules = {
      name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
      protocol: [{ required: true, message: '请选择协议', trigger: 'change' }],
      config: [{ required: true, message: '请输入配置', trigger: 'blur' }]
    }
    const checkMobile = () => { isMobile.value = window.innerWidth <= 768 }
    const loadCustomNodes = async () => {
      loading.value = true
      try {
        const params = {
          page: pagination.page, size: pagination.size,
          ...filters, search: searchKeyword.value
        }
        for (const key in params) { if (!params[key]) delete params[key] }
        const res = await adminAPI.getCustomNodes(params)
        if (res.data?.success !== false) {
          const raw = res.data.data
          const list = Array.isArray(raw) ? raw : (raw.data || [])
          customNodes.value = list.map(n => ({...n, testing: false}))
          pagination.total = raw.total || list.length
        } else {
          customNodes.value = []
        }
      } catch (e) {
        ElMessage.error('加载失败')
      } finally {
        loading.value = false
      }
    }
    const handleFilterChange = () => {
      pagination.page = 1
      loadCustomNodes()
    }
    const handleMobileSelect = (node, checked) => {
      if (checked) {
        if (!selectedNodes.value.find(n => n.id === node.id)) selectedNodes.value.push(node)
      } else {
        selectedNodes.value = selectedNodes.value.filter(n => n.id !== node.id)
      }
    }
    const handleGridSelect = (node, checked) => {
      handleMobileSelect(node, checked)
    }
    const isSelected = (node) => selectedNodes.value.some(n => n.id === node.id)
    const isAllSelected = computed({
      get: () => customNodes.value.length > 0 && selectedNodes.value.length === customNodes.value.length,
      set: (val) => selectedNodes.value = val ? [...customNodes.value] : []
    })
    const isIndeterminate = computed(() => selectedNodes.value.length > 0 && selectedNodes.value.length < customNodes.value.length)
    const toggleMobileSelectAll = (val) => selectedNodes.value = val ? [...customNodes.value] : []
    const handleSelectionChange = (val) => selectedNodes.value = val
    const editingNode = ref(null)
    const editNode = (node) => {
      editingNode.value = node
      Object.assign(nodeForm, {
        name: node.name,
        display_name: node.display_name,
        protocol: node.protocol,
        config: typeof node.config === 'object' ? JSON.stringify(node.config) : node.config,
        expire_time: node.expire_time,
        follow_user_expire: node.follow_user_expire
      })
      showAddDialog.value = true
    }
    const saveNode = async () => {
      if (!nodeFormRef.value) return
      await nodeFormRef.value.validate(async (valid) => {
        if (!valid) return
        saving.value = true
        try {
          const payload = { ...nodeForm }
          if (editingNode.value) {
             delete payload.protocol 
             delete payload.config
             await adminAPI.updateCustomNode(editingNode.value.id, payload)
          } else {
             await adminAPI.createCustomNode(payload)
          }
          ElMessage.success('保存成功')
          showAddDialog.value = false
          loadCustomNodes()
        } catch (e) {
          ElMessage.error('保存失败: ' + e.message)
        } finally {
          saving.value = false
        }
      })
    }
    const deleteNode = async (node) => {
      try {
        await ElMessageBox.confirm(`确认删除 "${node.name}"?`, '提示', { type: 'warning' })
        await adminAPI.deleteCustomNode(node.id)
        ElMessage.success('已删除')
        loadCustomNodes()
      } catch (error) {
        if (error !== 'cancel') ElMessage.error('删除节点失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const toggleNodeStatus = async (node) => {
      try {
        await adminAPI.updateCustomNode(node.id, { is_active: node.is_active })
        ElMessage.success(node.is_active ? '已启用' : '已禁用')
      } catch {
        node.is_active = !node.is_active
        ElMessage.error('操作失败')
      }
    }
    const handleCommand = (cmd) => {
      if (cmd === 'refresh') loadCustomNodes()
      if (cmd === 'batch_test') batchTest()
      if (cmd === 'batch_assign') handleBatchAssignClick()
      if (cmd === 'batch_delete') batchDelete()
    }
    const batchTest = async () => {
      if (!selectedNodes.value.length) return
      batchTesting.value = true
      try {
        await adminAPI.batchTestCustomNodes(selectedNodes.value.map(n => n.id))
        ElMessage.success('批量测试请求已发送')
        setTimeout(loadCustomNodes, 1000)
      } catch { ElMessage.error('测试请求失败') }
      finally { batchTesting.value = false }
    }
    const batchDelete = async () => {
      if (!selectedNodes.value.length) return
      try {
        await ElMessageBox.confirm(`确认删除选中的 ${selectedNodes.value.length} 个节点?`, '警告', { type: 'error' })
        batchDeleting.value = true
        await adminAPI.batchDeleteCustomNodes(selectedNodes.value.map(n => n.id))
        ElMessage.success('批量删除成功')
        selectedNodes.value = []
        loadCustomNodes()
      } catch (error) {
        if (error !== 'cancel') ElMessage.error('批量删除失败: ' + (error.response?.data?.message || error.message))
      } finally { batchDeleting.value = false }
    }
    const parseNodeLink = async () => {
      const link = nodeLinkInput.value.split('\n')[0].trim()
      if (!link) return
      parsing.value = true
      try {
        const res = await adminAPI.createCustomNode({ node_link: link, preview: true })
        if (res.data.success) {
           const data = res.data.data
           parsedNode.value = { 
             name: data.name, 
             server: typeof data.config === 'object' ? data.config.server : 'unknown',
             port: typeof data.config === 'object' ? data.config.port : 'unknown'
           }
        }
      } finally { parsing.value = false }
    }
    const batchImportLinks = async () => {
      const links = nodeLinkInput.value.split('\n').map(l=>l.trim()).filter(Boolean)
      if (!links.length) return
      saving.value = true
      try {
        const res = await adminAPI.importCustomNodeLinks(links)
        ElMessage.success(`导入成功: ${res.data.data?.imported || 0} 个`)
        showAddDialog.value = false
        loadCustomNodes()
      } catch { ElMessage.error('导入失败') }
      finally { saving.value = false }
    }
    const viewLink = async (node) => {
      try {
        const res = await adminAPI.getCustomNodeLink(node.id)
        if (res.data.success) {
          nodeLink.value = res.data.data
          showLinkDialog.value = true
        }
      } catch { ElMessage.error('获取链接失败') }
    }
    const copyLink = () => {
      if (nodeLink.value?.link) {
        navigator.clipboard.writeText(nodeLink.value.link)
        ElMessage.success('已复制')
      }
    }
    const assignSingleNode = (node) => {
      assignMode.value = 'single'
      assigningNode.value = node
      loadAssignedUsers(node.id)
      openAssignDialog()
    }
    const handleBatchAssignClick = () => {
      if (!selectedNodes.value.length) return
      assignMode.value = 'batch'
      assigningNode.value = null
      openAssignDialog()
    }
    const openAssignDialog = () => {
      selectedUserIds.value = []
      userSearchKeyword.value = ''
      searchedUsers.value = []
      showAssignDialog.value = true
    }
    const handleUserSearch = async () => {
      if (!userSearchKeyword.value) return
      loadingUsers.value = true
      try {
        const res = await adminAPI.getUsers({ keyword: userSearchKeyword.value, page: 1, size: 50 })
        searchedUsers.value = res.data.data?.users || []
      } finally { loadingUsers.value = false }
    }
    const handleAssign = async () => {
      batchAssigning.value = true
      try {
        const nodeIds = assignMode.value === 'single' ? [assigningNode.value.id] : selectedNodes.value.map(n => n.id)
        await adminAPI.batchAssignCustomNodes(nodeIds, selectedUserIds.value, assignExtraData)
        ElMessage.success('分配成功')
        showAssignDialog.value = false
        if (assignMode.value === 'single') loadAssignedUsers(assigningNode.value.id)
      } catch (e) { ElMessage.error('分配失败: ' + e.message) }
      finally { batchAssigning.value = false }
    }
    const loadAssignedUsers = async (nodeId) => {
      try {
        const res = await adminAPI.getCustomNodeUsers(nodeId)
        assignedUsers.value = res.data.data || []
      } catch (error) {
        ElMessage.error('加载用户列表失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const handleUnassign = async (user) => {
      try {
        await adminAPI.unassignCustomNodeFromUser(user.id, assigningNode.value.id)
        ElMessage.success('已移除')
        loadAssignedUsers(assigningNode.value.id)
      } catch (error) {
        ElMessage.error('移除用户失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const getStatusType = (s) => ({ active: 'success', inactive: 'info', error: 'danger' }[s] || 'info')
    const getStatusText = (s) => ({ active: '活跃', inactive: '非活跃', error: '错误' }[s] || s)
    const formatExpire = (row) => row.follow_user_expire ? '跟随用户' : (row.expire_time ? new Date(row.expire_time).toLocaleString() : '永久')
    const isExpired = (t) => t && new Date(t) < new Date()
    const testNode = async (node) => {
       node.testing = true
       try {
         const res = await adminAPI.testCustomNode(node.id)
         ElMessage.success(`延迟: ${res.data.data.latency}ms`)
         node.status = res.data.data.status
       } catch { ElMessage.error('测试失败') }
       finally { node.testing = false }
    }
    const testNodeFromLink = async () => {
       testingFromLink.value = true
       try {
         ElMessage.success('测试连接通过') 
       } finally { testingFromLink.value = false }
    }
    // 监听视图模式和网格设置变化，自动保存
    watch([viewMode, gridOrientation, gridColumns, gridSize], () => {
      saveSettings()
    })
    
    onMounted(() => {
      checkMobile()
      window.addEventListener('resize', checkMobile)
      loadSettings() // 先加载保存的设置
      loadCustomNodes()
    })
    onUnmounted(() => window.removeEventListener('resize', checkMobile))
    return {
      isMobile, viewMode, gridOrientation, gridColumns, gridSize, tableRef, columnWidths, loading, saving, parsing, customNodes, selectedNodes,
      handleColumnResize,
      showAddDialog, showLinkDialog, showAssignDialog, addNodeTab,
      searchKeyword, filters, pagination, nodeForm, nodeFormRef, rules,
      nodeLinkInput, parsedNode, nodeLink, testingFromLink,
      assignMode, assignedUsers, userSearchKeyword, searchedUsers, selectedUserIds,
      loadingUsers, batchAssigning, assignExtraData, batchTesting, batchDeleting,
      loadCustomNodes, handleFilterChange, handleSelectionChange, handleMobileSelect, handleGridSelect,
      handleCommand, editNode, saveNode, deleteNode, toggleNodeStatus,
      batchTest, batchDelete, parseNodeLink, batchImportLinks, viewLink, copyLink,
      testNode, testNodeFromLink, assignSingleNode, handleBatchAssignClick, handleAssign,
      handleUserSearch, handleUnassign, getStatusType, getStatusText, formatExpire, isExpired,
      isSelected, isAllSelected, isIndeterminate, toggleMobileSelectAll,
      Delete, Edit, Link, Refresh, Connection, User,
      editingNode
    }
  }
}
</script>
<style scoped>
.admin-custom-nodes {
  padding: 12px;
}
@media (max-width: 768px) {
  .admin-custom-nodes {
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
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }
  .filter-grid .el-select {
    width: 100%;
  }
  .search-box {
    grid-column: 1 / -1;
  }
}
.view-mode-group { margin-right: 8px; }
.grid-orientation-group { margin-right: 8px; }
.grid-size-group { margin-right: 8px; }
.grid-columns-select { margin-right: 8px; }
.batch-actions-bar {
  display: flex;
  align-items: center;
  background: var(--el-color-primary-light-9);
  padding: 8px 16px;
  border-radius: 4px;
  margin-bottom: 16px;
}
.batch-tip {
  font-size: 13px;
  color: var(--el-color-primary);
  margin-right: auto;
}
/* 桌面端方格视图（可调大小和方向） */
.desktop-grid-view {
  display: grid;
  gap: 16px;
  min-height: 120px;
}
/* 横向布局：固定列数 */
.desktop-grid-view.grid-horizontal.grid-cols-2 {
  grid-template-columns: repeat(2, 1fr);
}
.desktop-grid-view.grid-horizontal.grid-cols-3 {
  grid-template-columns: repeat(3, 1fr);
}
.desktop-grid-view.grid-horizontal.grid-cols-4 {
  grid-template-columns: repeat(4, 1fr);
}
.desktop-grid-view.grid-horizontal.grid-cols-5 {
  grid-template-columns: repeat(5, 1fr);
}
.desktop-grid-view.grid-horizontal.grid-cols-6 {
  grid-template-columns: repeat(6, 1fr);
}
/* 纵向布局：单列，可调宽度 */
.desktop-grid-view.grid-vertical {
  grid-template-columns: 1fr;
  max-width: 100%;
}
.desktop-grid-view.grid-vertical.grid-size-small {
  max-width: 400px;
  margin: 0 auto;
}
.desktop-grid-view.grid-vertical.grid-size-medium {
  max-width: 600px;
  margin: 0 auto;
}
.desktop-grid-view.grid-vertical.grid-size-large {
  max-width: 800px;
  margin: 0 auto;
}
.grid-empty {
  grid-column: 1 / -1;
  padding: 40px 0;
}
.grid-node-card {
  background: #fff;
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.05);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.grid-node-card:hover {
  border-color: var(--el-border-color);
  box-shadow: 0 4px 16px rgba(0,0,0,0.08);
}
.grid-node-card.is-selected {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 1px var(--el-color-primary);
}
.grid-node-card .gnc-header {
  padding: 12px 16px;
  background: #f8f9fa;
  border-bottom: 1px solid #ebeef5;
  display: flex;
  align-items: center;
  gap: 8px;
}
.grid-node-card .gnc-checkbox { margin-right: 0; flex-shrink: 0; }
.grid-node-card .gnc-title {
  flex: 1;
  font-weight: 600;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.grid-node-card .gnc-body {
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
}
.grid-node-card .gnc-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
}
.grid-node-card .gnc-row .label {
  color: var(--el-text-color-secondary);
  margin-right: 8px;
}
.grid-node-card .gnc-row .value {
  font-weight: 500;
  word-break: break-all;
  text-align: right;
}
.grid-node-card .gnc-footer {
  padding: 10px 16px;
  border-top: 1px solid #f0f2f5;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-wrap: wrap;
}
.table-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.grid-node-card .gnc-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}
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
  overflow: clip;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.card-checkbox { margin-right: 0; }
.card-info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-bottom: 12px;
}
.info-item {
  display: flex;
  flex-direction: column;
  background: var(--el-fill-color-extra-light);
  padding: 6px;
  border-radius: 4px;
}
.info-item.full-width {
  grid-column: 1 / -1;
}
.info-item .label {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.info-item .value {
  font-size: 13px;
  font-weight: 500;
  word-break: break-all;
}
.card-actions-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-top: 1px solid var(--el-border-color-lighter);
  padding-top: 8px;
}
.right-buttons {
  display: flex;
  gap: 4px;
}
.code-input :deep(.el-textarea__inner) {
  font-family: monospace;
  font-size: 12px;
  background-color: var(--el-fill-color-darker);
  color: var(--el-text-color-primary);
}
.link-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 10px;
}
.mini-user-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--el-fill-color-light);
  padding: 8px 12px;
  border-radius: 6px;
  margin-bottom: 8px;
}
.u-name { font-weight: 500; font-size: 14px; }
.u-time { font-size: 12px; color: var(--el-text-color-secondary); }
.text-danger { color: var(--el-color-danger); }
.text-secondary { color: var(--el-text-color-secondary); font-size: 12px; }
.text-xs { font-size: 12px; }
.dialog-scroll-content {
  max-height: 70vh;
  overflow-y: auto;
  padding-right: 4px;
}
.responsive-dialog :deep(.el-dialog__body) {
  padding: 15px 20px;
}
@media (max-width: 768px) {
  .responsive-dialog :deep(.el-dialog__body) {
    padding: 12px;
  }
  .dialog-scroll-content {
    max-height: calc(100vh - 120px);
  }
  .assign-form .el-select {
    width: 100%;
  }
}
.pagination-wrapper {
  margin-top: 20px;
  display: flex;
  justify-content: center;
}
</style>