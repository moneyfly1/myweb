<template>
  <div class="admin-invites">
    <el-card>
      <template #header>
        <div class="card-header-wrapper">
          <span>邀请管理</span>
          <div class="header-buttons">
            <el-button 
              type="default" 
              @click="showSettingsDialog = true"
              class="settings-button"
            >
              <el-icon><Setting /></el-icon>
              <span class="desktop-only">邀请设置</span>
            </el-button>
            <el-button type="primary" @click="loadData" class="refresh-button">
              <el-icon><Refresh /></el-icon>
              <span class="desktop-only">刷新</span>
            </el-button>
          </div>
        </div>
      </template>

      <el-tabs v-model="activeTab" type="border-card">

        <!-- 邀请码列表 -->
        <el-tab-pane label="邀请码列表" name="codes">
          <!-- 移动端搜索和筛选栏 -->
          <div class="mobile-action-bar" v-if="isMobile">
            <!-- 搜索栏（用户名或邮箱） -->
            <div class="mobile-search-section">
              <div class="search-input-wrapper">
                <el-input
                  v-model="codeFilterForm.user_query"
                  placeholder="邀请人账号或邮箱"
                  clearable
                  class="mobile-search-input"
                  @keyup.enter="searchCodes"
                />
                <el-button 
                  @click="searchCodes" 
                  class="search-button-inside"
                  type="default"
                  plain
                >
                  <el-icon><Search /></el-icon>
                </el-button>
              </div>
            </div>

            <!-- 筛选按钮组 -->
            <div class="mobile-filter-buttons">
              <el-dropdown @command="handleStatusFilter" trigger="click" placement="bottom-start">
                <el-button 
                  size="small" 
                  :type="codeFilterForm.is_active !== null ? 'primary' : 'default'"
                  plain
                >
                  <el-icon><Filter /></el-icon>
                  {{ getStatusFilterText() }}
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="">全部状态</el-dropdown-item>
                    <el-dropdown-item command="true">启用</el-dropdown-item>
                    <el-dropdown-item command="false">禁用</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>

              <el-button 
                size="small" 
                type="default" 
                plain
                @click="resetCodeFilter"
              >
                <el-icon><Refresh /></el-icon>
                重置
              </el-button>
            </div>
          </div>
          
          <!-- 桌面端筛选表单 -->
          <div class="desktop-only" style="margin-bottom: 20px;">
            <el-form :inline="true" :model="codeFilterForm" class="filter-form">
              <el-form-item label="邀请人">
                <el-input 
                  v-model="codeFilterForm.user_query" 
                  placeholder="账号或邮箱"
                  clearable
                  style="width: 200px;"
                />
              </el-form-item>
              <el-form-item label="邀请码">
                <el-input 
                  v-model="codeFilterForm.code" 
                  placeholder="搜索邀请码"
                  clearable
                  style="width: 200px;"
                />
              </el-form-item>
              <el-form-item label="状态">
                <el-select v-model="codeFilterForm.is_active" clearable placeholder="全部" style="width: 120px;">
                  <el-option label="启用" :value="true" />
                  <el-option label="禁用" :value="false" />
                </el-select>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="searchCodes">搜索</el-button>
                <el-button @click="resetCodeFilter">重置</el-button>
              </el-form-item>
            </el-form>
          </div>

          <!-- 批量操作工具栏 -->
          <div class="batch-actions" v-if="selectedCodes.length > 0" style="margin-bottom: 16px;">
            <div class="batch-info">
              <span>已选择 {{ selectedCodes.length }} 个邀请码</span>
            </div>
            <div class="batch-buttons">
              <el-button type="danger" @click="batchDeleteCodes" :loading="batchDeleting">
                <el-icon><Delete /></el-icon>
                批量删除
              </el-button>
              <el-button @click="clearCodeSelection">
                <el-icon><Refresh /></el-icon>
                取消选择
              </el-button>
            </div>
          </div>

          <el-table 
            :data="inviteCodes" 
            v-loading="codesLoading"
            border
            stripe
            style="width: 100%"
            :default-sort="{ prop: 'created_at', order: 'descending' }"
            @selection-change="handleCodeSelectionChange"
          >
            <el-table-column type="selection" width="50" />
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="code" label="邀请码" min-width="120">
              <template #default="scope">
                <el-tag>{{ scope.row.code }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="username" label="邀请人" min-width="150">
              <template #default="scope">
                <div>
                  <div>{{ scope.row.username || '未知用户' }}</div>
                  <div style="color: #909399; font-size: 12px;">{{ scope.row.user_email || scope.row.email || '无邮箱' }}</div>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="used_count" label="已使用" width="100" align="center">
              <template #default="scope">
                <span>{{ scope.row.used_count }} / {{ scope.row.max_uses || '∞' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="inviter_reward" label="邀请人奖励" width="120" align="right">
              <template #default="scope">
                <span style="color: #67c23a; font-weight: bold;">¥{{ (scope.row.inviter_reward || 0).toFixed(2) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="invitee_reward" label="被邀请人奖励" width="140" align="right">
              <template #default="scope">
                <span style="color: #409eff; font-weight: bold;">¥{{ (scope.row.invitee_reward || 0).toFixed(2) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="expires_at" label="过期时间" width="180" class-name="expires-column">
              <template #default="scope">
                <span v-if="scope.row.expires_at">{{ formatDate(scope.row.expires_at) }}</span>
                <span v-else style="color: #909399;">永不过期</span>
              </template>
            </el-table-column>
            <el-table-column prop="is_active" label="状态" width="100" align="center">
              <template #default="scope">
                <el-tag :type="scope.row.is_active ? 'success' : 'danger'">
                  {{ scope.row.is_active ? '启用' : '禁用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="180" sortable="custom">
              <template #default="scope">
                {{ formatDate(scope.row.created_at) }}
              </template>
            </el-table-column>
          </el-table>

          <div style="margin-top: 20px; display: flex; justify-content: center;">
            <el-pagination
              v-model:current-page="codePage"
              v-model:page-size="codePageSize"
              :page-sizes="[10, 20, 50, 100]"
              :total="codeTotal"
              layout="total, sizes, prev, pager, next, jumper"
              @size-change="loadInviteCodes"
              @current-change="loadInviteCodes"
            />
          </div>
        </el-tab-pane>

        <!-- 邀请关系列表 -->
        <el-tab-pane label="邀请关系" name="relations">
          <!-- 移动端搜索和筛选栏 -->
          <div class="mobile-action-bar" v-if="isMobile">
            <!-- 搜索栏（邀请人邮箱） -->
            <div class="mobile-search-section">
              <div class="search-input-wrapper">
                <el-input
                  v-model="relationFilterForm.inviter_query"
                  placeholder="邀请人账号或邮箱"
                  clearable
                  class="mobile-search-input"
                  @keyup.enter="searchRelations"
                />
                <el-button 
                  @click="searchRelations" 
                  class="search-button-inside"
                  type="default"
                  plain
                >
                  <el-icon><Search /></el-icon>
                </el-button>
              </div>
            </div>

            <!-- 筛选按钮组 -->
            <div class="mobile-filter-buttons">
              <el-button 
                size="small" 
                type="default" 
                plain
                @click="resetRelationFilter"
              >
                <el-icon><Refresh /></el-icon>
                重置
              </el-button>
            </div>
            
            <!-- 被邀请人邮箱搜索框 -->
            <div class="mobile-search-section" style="margin-top: 12px;">
              <div class="search-input-wrapper">
                <el-input
                  v-model="relationFilterForm.invitee_query"
                  placeholder="被邀请人账号或邮箱"
                  clearable
                  class="mobile-search-input"
                  @keyup.enter="searchRelations"
                />
                <el-button 
                  @click="searchRelations" 
                  class="search-button-inside"
                  type="default"
                  plain
                >
                  <el-icon><Search /></el-icon>
                </el-button>
              </div>
            </div>
          </div>
          
          <!-- 桌面端筛选表单 -->
          <div class="desktop-only" style="margin-bottom: 20px;">
            <el-form :inline="true" :model="relationFilterForm" class="filter-form">
              <el-form-item label="邀请人">
                <el-input 
                  v-model="relationFilterForm.inviter_query" 
                  placeholder="账号或邮箱"
                  clearable
                  style="width: 200px;"
                />
              </el-form-item>
              <el-form-item label="被邀请人">
                <el-input 
                  v-model="relationFilterForm.invitee_query" 
                  placeholder="账号或邮箱"
                  clearable
                  style="width: 200px;"
                />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="searchRelations">搜索</el-button>
                <el-button @click="resetRelationFilter">重置</el-button>
              </el-form-item>
            </el-form>
          </div>

          <!-- 批量操作工具栏 -->
          <div class="batch-actions" v-if="selectedRelations.length > 0" style="margin-bottom: 16px;">
            <div class="batch-info">
              <span>已选择 {{ selectedRelations.length }} 条邀请关系</span>
            </div>
            <div class="batch-buttons">
              <el-button type="danger" @click="batchDeleteRelations" :loading="batchDeleting">
                <el-icon><Delete /></el-icon>
                批量删除
              </el-button>
              <el-button @click="clearRelationSelection">
                <el-icon><Refresh /></el-icon>
                取消选择
              </el-button>
            </div>
          </div>

          <el-table 
            :data="inviteRelations" 
            v-loading="relationsLoading"
            border
            stripe
            style="width: 100%"
            @selection-change="handleRelationSelectionChange"
          >
            <el-table-column type="selection" width="50" />
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="invite_code" label="邀请码" width="150">
              <template #default="scope">
                <el-tag>{{ scope.row.invite_code }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="inviter_username" label="邀请人" width="180">
              <template #default="scope">
                <div>
                  <div>{{ scope.row.inviter_username }}</div>
                  <div style="color: #909399; font-size: 12px;">{{ scope.row.inviter_email || '无邮箱' }}</div>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="invitee_username" label="被邀请人" width="180">
              <template #default="scope">
                <div>
                  <div>{{ scope.row.invitee_username }}</div>
                  <div style="color: #909399; font-size: 12px;">{{ scope.row.invitee_email || '无邮箱' }}</div>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="inviter_reward_amount" label="邀请人奖励" width="140" align="right">
              <template #default="scope">
                <div>
                  <span style="color: #67c23a; font-weight: bold;">¥{{ (scope.row.inviter_reward_amount || 0).toFixed(2) }}</span>
                  <el-tag 
                    :type="scope.row.inviter_reward_given ? 'success' : 'warning'" 
                    size="small" 
                    style="margin-left: 8px;"
                  >
                    {{ scope.row.inviter_reward_given ? '已发放' : '未发放' }}
                  </el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="invitee_reward_amount" label="被邀请人奖励" width="140" align="right">
              <template #default="scope">
                <div>
                  <span style="color: #409eff; font-weight: bold;">¥{{ (scope.row.invitee_reward_amount || 0).toFixed(2) }}</span>
                  <el-tag 
                    :type="scope.row.invitee_reward_given ? 'success' : 'warning'" 
                    size="small" 
                    style="margin-left: 8px;"
                  >
                    {{ scope.row.invitee_reward_given ? '已发放' : '未发放' }}
                  </el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="invitee_total_consumption" label="累计消费" width="120" align="right">
              <template #default="scope">
                <span style="color: #e6a23c; font-weight: bold;">¥{{ (scope.row.invitee_total_consumption || 0).toFixed(2) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="注册时间" width="180">
              <template #default="scope">
                {{ formatDate(scope.row.created_at) }}
              </template>
            </el-table-column>
          </el-table>

          <div style="margin-top: 20px; display: flex; justify-content: center;">
            <el-pagination
              v-model:current-page="relationPage"
              v-model:page-size="relationPageSize"
              :page-sizes="[10, 20, 50, 100]"
              :total="relationTotal"
              layout="total, sizes, prev, pager, next, jumper"
              @size-change="loadInviteRelations"
              @current-change="loadInviteRelations"
            />
          </div>
        </el-tab-pane>

        <!-- 邀请统计 -->
        <el-tab-pane label="邀请统计" name="statistics">
          <el-row :gutter="20" class="statistics-row">
            <el-col :xs="12" :sm="12" :md="6">
              <el-card shadow="hover" class="stat-card">
                <div class="stat-content">
                  <div class="stat-value" style="color: #409eff;">
                    {{ statistics?.total_codes || 0 }}
                  </div>
                  <div class="stat-label">总邀请码数</div>
                </div>
              </el-card>
            </el-col>
            <el-col :xs="12" :sm="12" :md="6">
              <el-card shadow="hover" class="stat-card">
                <div class="stat-content">
                  <div class="stat-value" style="color: #67c23a;">
                    {{ statistics?.total_relations || 0 }}
                  </div>
                  <div class="stat-label">总邀请关系数</div>
                </div>
              </el-card>
            </el-col>
            <el-col :xs="12" :sm="12" :md="6">
              <el-card shadow="hover" class="stat-card">
                <div class="stat-content">
                  <div class="stat-value" style="color: #e6a23c;">
                    ¥{{ (statistics?.total_reward || 0).toFixed(2) }}
                  </div>
                  <div class="stat-label">总奖励金额</div>
                </div>
              </el-card>
            </el-col>
            <el-col :xs="12" :sm="12" :md="6">
              <el-card shadow="hover" class="stat-card">
                <div class="stat-content">
                  <div class="stat-value" style="color: #f56c6c;">
                    ¥{{ (statistics?.total_consumption || 0).toFixed(2) }}
                  </div>
                  <div class="stat-label">被邀请人总消费</div>
                </div>
              </el-card>
            </el-col>
          </el-row>
        </el-tab-pane>
      </el-tabs>
    </el-card>
    
    <!-- 邀请设置对话框 -->
    <el-dialog
      v-model="showSettingsDialog"
      title="邀请设置"
      :width="isMobile ? '95%' : '600px'"
      :close-on-click-modal="!isMobile"
      class="settings-dialog"
    >
      <div class="settings-dialog-content">
        <el-alert
          title="邀请奖励配置说明"
          type="info"
          :closable="false"
          class="settings-alert"
        >
          <template #default>
            <div class="alert-content">
              <p><strong>邀请人奖励：</strong>当被邀请人首次购买套餐后，邀请人将获得的奖励金额（元）</p>
              <p><strong>被邀请人奖励：</strong>新用户使用邀请码注册后，立即获得的奖励金额（元）</p>
              <p class="alert-note">注意：此设置将应用于所有新生成的邀请码，已生成的邀请码不受影响</p>
            </div>
          </template>
        </el-alert>
        
        <el-form :model="inviteSettings" label-width="0" class="invite-settings-form">
          <el-form-item prop="invite_inviter_reward" class="settings-form-item">
            <div class="form-item-wrapper">
              <div class="form-item-label">邀请人奖励（元）</div>
              <el-input-number 
                v-model="inviteSettings.invite_inviter_reward" 
                :min="0" 
                :max="10000"
                :precision="2"
                :step="1"
                class="settings-input-number"
                :controls-position="isMobile ? 'right' : 'right'"
              />
            </div>
          </el-form-item>
          <el-form-item prop="invite_invitee_reward" class="settings-form-item">
            <div class="form-item-wrapper">
              <div class="form-item-label">被邀请人奖励（元）</div>
              <el-input-number 
                v-model="inviteSettings.invite_invitee_reward" 
                :min="0" 
                :max="10000"
                :precision="2"
                :step="1"
                class="settings-input-number"
                :controls-position="isMobile ? 'right' : 'right'"
              />
            </div>
          </el-form-item>
        </el-form>
      </div>
      
      <template #footer>
        <div class="dialog-footer-buttons">
          <el-button @click="showSettingsDialog = false" class="mobile-action-btn">取消</el-button>
          <el-button 
            type="primary" 
            @click="saveInviteSettings" 
            :loading="savingSettings"
            class="mobile-action-btn"
          >
            保存设置
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Filter, Refresh, Setting, Delete } from '@element-plus/icons-vue'
import { inviteAPI } from '@/utils/api'
import { useApi } from '@/utils/api'

const api = useApi()
const activeTab = ref('codes')
const codesLoading = ref(false)
const relationsLoading = ref(false)
const savingSettings = ref(false)
const isMobile = ref(window.innerWidth <= 768)
const showSettingsDialog = ref(false)
const selectedCodes = ref([])
const selectedRelations = ref([])
const batchDeleting = ref(false)

// 邀请设置
const inviteSettings = reactive({
  invite_inviter_reward: 0.0,
  invite_invitee_reward: 0.0
})

// 加载邀请设置
const loadInviteSettings = async () => {
  try {
    const response = await api.get('/admin/settings')
    const settings = response.data?.data || response.data || {}
    if (settings.invite) {
      Object.assign(inviteSettings, settings.invite)
    }
  } catch (error) {
    console.error('加载邀请设置失败:', error)
    ElMessage.error('加载邀请设置失败: ' + (error.response?.data?.message || error.message || '未知错误'))
  }
}

// 保存邀请设置
const saveInviteSettings = async () => {
  savingSettings.value = true
  try {
    await api.put('/admin/settings/invite', inviteSettings)
    ElMessage.success('邀请设置保存成功')
    showSettingsDialog.value = false
  } catch (error) {
    console.error('保存邀请设置失败:', error)
    ElMessage.error('保存失败: ' + (error.response?.data?.message || error.message || '未知错误'))
  } finally {
    savingSettings.value = false
  }
}

// 邀请码列表
const inviteCodes = ref([])
const codePage = ref(1)
const codePageSize = ref(20)
const codeTotal = ref(0)
const codeFilterForm = reactive({
  user_query: '',
  code: '',
  is_active: null
})

// 邀请关系列表
const inviteRelations = ref([])
const relationPage = ref(1)
const relationPageSize = ref(20)
const relationTotal = ref(0)
const relationFilterForm = reactive({
  inviter_query: '',
  invitee_query: ''
})

// 统计数据
const statistics = reactive({
  total_codes: 0,
  total_relations: 0,
  total_reward: 0,
  total_consumption: 0
})

const loadInviteCodes = async () => {
  codesLoading.value = true
  try {
    const params = {
      page: codePage.value,
      size: codePageSize.value
    }
    if (codeFilterForm.user_query) {
      params.user_query = codeFilterForm.user_query
    }
    if (codeFilterForm.code) {
      params.code = codeFilterForm.code
    }
    if (codeFilterForm.is_active !== null) {
      params.is_active = codeFilterForm.is_active
    }
    
    const response = await inviteAPI.getAllInviteCodes(params)
    console.log('管理员邀请码列表响应:', response)
    
    // 处理多种可能的响应格式
    if (response && response.data) {
      const responseData = response.data
      // 标准格式：{ success: true, data: { invite_codes: [...], total: ... } }
      let codeList = []
      if (responseData.success !== false && responseData.data) {
        codeList = responseData.data.invite_codes || []
        codeTotal.value = responseData.data.total || 0
        console.log('✅ 成功加载邀请码:', codeList.length, '个，总计:', codeTotal.value)
      } 
      // 直接包含 invite_codes
      else if (responseData.invite_codes) {
        codeList = Array.isArray(responseData.invite_codes) ? responseData.invite_codes : []
        codeTotal.value = responseData.total || codeList.length
        console.log('✅ 成功加载邀请码（直接格式）:', codeList.length, '个')
      }
      // 如果 success 为 false，显示错误信息
      else if (responseData.success === false) {
        const errorMsg = responseData.message || '获取邀请码列表失败'
        ElMessage.error(errorMsg)
        codeList = []
        codeTotal.value = 0
      }
      else {
        console.warn('⚠️ 未识别的响应格式:', responseData)
        codeList = []
        codeTotal.value = 0
      }
      // 确保 is_active 是布尔值，并处理 max_uses 字段
      inviteCodes.value = codeList.map(code => {
        let maxUsesDisplay = null;
        if (code.max_uses && typeof code.max_uses === 'object' && code.max_uses.Valid) {
          maxUsesDisplay = code.max_uses.Int64;
        } else if (typeof code.max_uses === 'number') {
          maxUsesDisplay = code.max_uses;
        }
        return {
          ...code,
          is_active: code.is_active === true || code.is_active === 1 || code.is_active === '1',
          username: code.username || code.user?.username || '未知用户',
          user_email: code.user_email || code.email || code.user?.email || code.User?.Email || '无邮箱',
          max_uses_display: maxUsesDisplay,
        };
      })
    } else {
      console.warn('⚠️ 响应数据为空')
      inviteCodes.value = []
      codeTotal.value = 0
    }
  } catch (error) {
    console.error('加载邀请码列表失败:', error)
    const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '未知错误'
    ElMessage.error('加载邀请码列表失败: ' + errorMsg)
    inviteCodes.value = []
    codeTotal.value = 0
  } finally {
    codesLoading.value = false
  }
}

const loadInviteRelations = async () => {
  relationsLoading.value = true
  try {
    const params = {
      page: relationPage.value,
      size: relationPageSize.value
    }
    if (relationFilterForm.inviter_query) {
      params.inviter_query = relationFilterForm.inviter_query
    }
    if (relationFilterForm.invitee_query) {
      params.invitee_query = relationFilterForm.invitee_query
    }
    
    const response = await inviteAPI.getInviteRelations(params)
    console.log('管理员邀请关系列表完整响应:', response)
    console.log('响应数据结构:', {
      hasResponse: !!response,
      hasData: !!response?.data,
      responseDataType: typeof response?.data,
      responseDataKeys: response?.data ? Object.keys(response.data) : [],
      responseDataSuccess: response?.data?.success,
      responseDataMessage: response?.data?.message,
      hasNestedData: !!response?.data?.data,
      nestedDataKeys: response?.data?.data ? Object.keys(response.data.data) : []
    })
    
    // 处理多种可能的响应格式
    if (response && response.data) {
      const responseData = response.data
      
      // 标准格式：{ success: true, data: { relations: [...], total: ... } }
      if (responseData.success !== false && responseData.data) {
        if (responseData.data.relations && Array.isArray(responseData.data.relations)) {
          // 处理数据，确保所有字段都存在
          inviteRelations.value = responseData.data.relations.map(relation => ({
            ...relation,
            invite_code: relation.invite_code || '',
            inviter_username: relation.inviter_username || '',
            inviter_email: relation.inviter_email || '',
            invitee_username: relation.invitee_username || '',
            invitee_email: relation.invitee_email || '',
            inviter_reward_amount: relation.inviter_reward_amount || 0,
            invitee_reward_amount: relation.invitee_reward_amount || 0,
            invitee_total_consumption: relation.invitee_total_consumption || 0,
            inviter_reward_given: relation.inviter_reward_given || false,
            invitee_reward_given: relation.invitee_reward_given || false
          }))
          relationTotal.value = responseData.data.total || 0
          console.log('✅ 成功加载邀请关系（标准格式）:', inviteRelations.value.length, '条，总计:', relationTotal.value)
        } else {
          console.warn('⚠️ data存在但没有relations字段:', responseData.data)
          inviteRelations.value = []
          relationTotal.value = 0
        }
      } 
      else if (responseData.relations && Array.isArray(responseData.relations)) {
        // 处理数据，确保所有字段都存在
        inviteRelations.value = responseData.relations.map(relation => ({
          ...relation,
          invite_code: relation.invite_code || '',
          inviter_username: relation.inviter_username || '',
          inviter_email: relation.inviter_email || '',
          invitee_username: relation.invitee_username || '',
          invitee_email: relation.invitee_email || '',
          inviter_reward_amount: relation.inviter_reward_amount || 0,
          invitee_reward_amount: relation.invitee_reward_amount || 0,
          invitee_total_consumption: relation.invitee_total_consumption || 0,
          inviter_reward_given: relation.inviter_reward_given || false,
          invitee_reward_given: relation.invitee_reward_given || false
        }))
        relationTotal.value = responseData.total || inviteRelations.value.length
        console.log('✅ 成功加载邀请关系（直接格式）:', inviteRelations.value.length, '条')
      }
      // 如果 success 为 false，显示错误信息
      else if (responseData.success === false) {
        const errorMsg = responseData.message || '获取邀请关系列表失败'
        console.error('❌ API返回失败:', errorMsg)
        ElMessage.error(errorMsg)
        inviteRelations.value = []
        relationTotal.value = 0
      }
      else {
        console.warn('⚠️ 未识别的响应格式:', {
          responseData,
          hasData: !!responseData.data,
          dataType: typeof responseData.data,
          dataKeys: responseData.data ? Object.keys(responseData.data) : []
        })
        inviteRelations.value = []
        relationTotal.value = 0
      }
    } else {
      console.warn('⚠️ 响应数据为空或格式异常:', {
        hasResponse: !!response,
        hasData: !!response?.data
      })
      inviteRelations.value = []
      relationTotal.value = 0
    }
  } catch (error) {
    console.error('加载邀请关系列表失败:', error)
    console.error('错误详情:', {
      message: error.message,
      response: error.response,
      responseData: error.response?.data,
      responseStatus: error.response?.status,
      responseHeaders: error.response?.headers
    })
    const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '未知错误'
    ElMessage.error('加载邀请关系列表失败: ' + errorMsg)
    inviteRelations.value = []
    relationTotal.value = 0
  } finally {
    relationsLoading.value = false
  }
}

const loadStatistics = async () => {
  try {
    // 使用专门的管理员统计接口，避免请求大量数据
    const response = await inviteAPI.getAdminInviteStatistics()
    
    if (response?.data?.data) {
      statistics.total_codes = response.data.data.total_codes || 0
      statistics.total_relations = response.data.data.total_relations || 0
      statistics.total_reward = response.data.data.total_reward || 0
      statistics.total_consumption = response.data.data.total_consumption || 0
    } else {
      // 如果响应格式不对，初始化默认值
      statistics.total_codes = 0
      statistics.total_relations = 0
      statistics.total_reward = 0
      statistics.total_consumption = 0
    }
  } catch (error) {
    console.error('加载统计数据失败:', error)
    // 初始化默认值
    statistics.total_codes = 0
    statistics.total_relations = 0
    statistics.total_reward = 0
    statistics.total_consumption = 0
    // 如果统计接口失败，尝试从列表接口获取基本信息
    try {
      const codesResponse = await inviteAPI.getAllInviteCodes({ page: 1, size: 1 })
      const relationsResponse = await inviteAPI.getInviteRelations({ page: 1, size: 1 })
      
      if (codesResponse?.data?.data) {
        statistics.total_codes = codesResponse.data.data.total || 0
      }
      if (relationsResponse?.data?.data) {
        statistics.total_relations = relationsResponse.data.data.total || 0
      }
    } catch (fallbackError) {
      console.error('获取基本信息也失败:', fallbackError)
    }
  }
}

const searchCodes = () => {
  codePage.value = 1
  loadInviteCodes()
}

const resetCodeFilter = () => {
  Object.assign(codeFilterForm, {
    user_query: '',
    code: '',
    is_active: null
  })
  searchCodes()
}

const searchRelations = () => {
  relationPage.value = 1
  loadInviteRelations()
}

const resetRelationFilter = () => {
  Object.assign(relationFilterForm, {
    inviter_query: '',
    invitee_query: ''
  })
  searchRelations()
}

// 批量操作相关函数
const handleCodeSelectionChange = (selection) => {
  selectedCodes.value = selection
}

const clearCodeSelection = () => {
  selectedCodes.value = []
}

const handleRelationSelectionChange = (selection) => {
  selectedRelations.value = selection
}

const clearRelationSelection = () => {
  selectedRelations.value = []
}

const batchDeleteCodes = async () => {
  if (selectedCodes.value.length === 0) {
    ElMessage.warning('请先选择要删除的邀请码')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedCodes.value.length} 个邀请码吗？已使用的邀请码将被禁用而不是删除。`,
      '确认批量删除',
      {
        type: 'warning',
        confirmButtonText: '确定删除',
        cancelButtonText: '取消'
      }
    )

    batchDeleting.value = true
    
    const codeIds = selectedCodes.value.map(code => code.id)
    const response = await inviteAPI.batchDeleteInviteCodes(codeIds)
    
    const data = response.data?.data || {}
    const deletedCount = data.deleted_count || 0
    const disabledCount = data.disabled_count || 0
    
    let message = `成功删除 ${deletedCount} 个邀请码`
    if (disabledCount > 0) {
      message += `，已禁用 ${disabledCount} 个已使用的邀请码`
    }
    
    ElMessage.success(message)
    
    clearCodeSelection()
    loadInviteCodes()
    
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('批量删除失败: ' + (error.response?.data?.message || error.message || '未知错误'))
    }
  } finally {
    batchDeleting.value = false
  }
}

const batchDeleteRelations = async () => {
  if (selectedRelations.value.length === 0) {
    ElMessage.warning('请先选择要删除的邀请关系')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedRelations.value.length} 条邀请关系吗？此操作不可恢复。`,
      '确认批量删除',
      {
        type: 'warning',
        confirmButtonText: '确定删除',
        cancelButtonText: '取消'
      }
    )

    batchDeleting.value = true
    
    const relationIds = selectedRelations.value.map(relation => relation.id)
    await inviteAPI.batchDeleteInviteRelations(relationIds)
    
    ElMessage.success(`成功删除 ${selectedRelations.value.length} 条邀请关系`)
    
    clearRelationSelection()
    loadInviteRelations()
    
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('批量删除失败: ' + (error.response?.data?.message || error.message || '未知错误'))
    }
  } finally {
    batchDeleting.value = false
  }
}

const getStatusFilterText = () => {
  if (codeFilterForm.is_active === true) return '启用'
  if (codeFilterForm.is_active === false) return '禁用'
  return '状态筛选'
}

const handleStatusFilter = (command) => {
  if (command === '') {
    codeFilterForm.is_active = null
  } else if (command === 'true') {
    codeFilterForm.is_active = true
  } else if (command === 'false') {
    codeFilterForm.is_active = false
  }
  searchCodes()
}

const handleResize = () => {
  isMobile.value = window.innerWidth <= 768
}

const loadData = () => {
  loadInviteCodes()
  loadInviteRelations()
  loadStatistics()
}

const formatDate = (dateString) => {
  if (!dateString) return '-'
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

onMounted(() => {
  loadInviteSettings()
  loadData()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped lang="scss">
.admin-invites {
  padding: 20px;
  width: 100%;
  box-sizing: border-box;
  overflow-x: hidden;
}

/* 卡片头部样式 */
.card-header-wrapper {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  flex-wrap: wrap;
  gap: 10px;
}

.header-buttons {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
}

.settings-button,
.refresh-button {
  display: flex;
  align-items: center;
  gap: 5px;
}

.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}

/* 批量操作工具栏 */
.batch-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #f5f7fa;
  border-radius: 4px;
  margin-bottom: 16px;
}

.batch-info {
  font-size: 14px;
  color: #606266;
  font-weight: 500;
}

.batch-buttons {
  display: flex;
  gap: 10px;
}

@media (max-width: 768px) {
  .batch-actions {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }
  
  .batch-buttons {
    width: 100%;
    flex-direction: column;
  }
  
  .batch-buttons .el-button {
    width: 100%;
  }
}

.filter-form {
  :deep(.el-form-item) {
    margin-bottom: 10px;
  }
}

/* 移除所有输入框的圆角和阴影效果，设置为简单长方形，只保留外部边框 */
:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
}

:deep(.el-input-number) {
  width: 100%;
  box-sizing: border-box;
}

:deep(.el-input-number .el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
  width: 100%;
  box-sizing: border-box;
}

:deep(.el-input-number__increase),
:deep(.el-input-number__decrease) {
  box-sizing: border-box;
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
  border-color: #409eff !important;
  box-shadow: none !important;
}

/* 表格响应式样式 */
:deep(.el-table) {
  .el-table__cell {
    padding: 12px 0;
  }
}

/* 移动端搜索和筛选栏样式 */
.mobile-action-bar {
  display: none;
  padding: 16px;
  box-sizing: border-box;
  background: #f5f7fa;
  border-radius: 8px;
  margin-bottom: 16px;
}

.mobile-search-section {
  margin-bottom: 12px;
  width: 100%;
  box-sizing: border-box;
}

.search-input-wrapper {
  position: relative;
  width: 100%;
  display: flex;
  align-items: center;
}

.mobile-search-input {
  flex: 1;
  width: 100%;
  box-sizing: border-box;
  min-width: 0;
}

.search-button-inside {
  position: absolute;
  right: 4px;
  top: 50%;
  transform: translateY(-50%);
  background: rgba(255, 255, 255, 0.98);
  border: 2px solid rgba(255, 255, 255, 0.4);
  color: #667eea;
  border-radius: 8px;
  font-weight: 600;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
  z-index: 10;
  padding: 8px 12px;
  height: auto;
}

/* 移动端筛选表单 */
.mobile-filter-form {
  width: 100%;
  margin-bottom: 12px;
  box-sizing: border-box;
  
  :deep(.el-form) {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  
  :deep(.el-form-item) {
    margin-bottom: 0;
    width: 100%;
  }
  
  :deep(.el-form-item__content) {
    margin-left: 0 !important;
    width: 100%;
  }
}

.mobile-filter-input,
.mobile-filter-select {
  width: 100% !important;
  
  :deep(.el-input__wrapper) {
    border-radius: 6px;
  }
  
  :deep(.el-select__wrapper) {
    border-radius: 6px;
  }
}

/* 移动端操作按钮 */
.mobile-action-buttons {
  display: flex;
  gap: 10px;
  width: 100%;
  box-sizing: border-box;
}

.mobile-search-btn,
.mobile-reset-btn {
  flex: 1;
  height: 44px;
  font-size: 15px;
  font-weight: 500;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  box-sizing: border-box;
}

.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}

/* 手机端样式 */
@media (max-width: 768px) {
  .admin-invites {
    padding: 8px 5px;
    width: 100%;
    box-sizing: border-box;
    overflow-x: hidden;
  }
  
  /* 卡片优化 */
  :deep(.el-card) {
    width: 100%;
    box-sizing: border-box;
    margin: 0;
    
    .el-card__header {
      padding: 12px 10px;
      font-size: 14px;
    }
    
    .el-card__body {
      padding: 12px 10px;
      width: 100%;
      box-sizing: border-box;
      overflow-x: hidden;
    }
  }
  
  .mobile-action-bar {
    display: block !important;
    width: 100%;
    box-sizing: border-box;
    padding: 12px;
  }
  
  .mobile-search-section {
    margin-bottom: 12px;
  }
  
  .search-input-wrapper {
    height: 44px;
  }
  
  .mobile-search-input {
    :deep(.el-input__wrapper) {
      height: 44px;
      padding-right: 50px;
    }
    
    :deep(.el-input__inner) {
      height: 44px;
      line-height: 44px;
      font-size: 16px;
      padding-right: 50px;
    }
    
    :deep(.el-input-number) {
      width: 100%;
      
      .el-input__wrapper {
        height: 44px;
        padding-right: 50px;
      }
      
      .el-input__inner {
        height: 44px;
        line-height: 44px;
        font-size: 16px;
        padding-right: 50px;
      }
    }
  }
  
  .search-button-inside {
    height: 36px;
    padding: 6px 10px;
  }
  
  .mobile-filter-buttons {
    margin-bottom: 12px;
    
    .el-button {
      height: 40px;
      font-size: 14px;
    }
  }
  
  .mobile-filter-form {
    margin-bottom: 0;
  }
  
  .mobile-filter-input {
    :deep(.el-input__wrapper) {
      height: 44px;
    }
    
    :deep(.el-input-number__input) {
      height: 44px;
      line-height: 44px;
      font-size: 15px;
    }
  }

  /* 标签页优化 */
  :deep(.el-tabs) {
    width: 100%;
    box-sizing: border-box;
    overflow-x: hidden;
    
    .el-tabs__header {
      margin: 0;
      width: 100%;
      box-sizing: border-box;
    }
    
    .el-tabs__nav-wrap {
      width: 100%;
      box-sizing: border-box;
      overflow-x: auto;
      -webkit-overflow-scrolling: touch;
    }
    
    .el-tabs__item {
      padding: 0 10px;
      font-size: 13px;
      white-space: nowrap;
    }
    
    .el-tabs__content {
      width: 100%;
      box-sizing: border-box;
      overflow-x: hidden;
    }
  }
  
  /* 卡片头部移动端优化 */
  .card-header-wrapper {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }
  
  .header-buttons {
    width: 100%;
    display: flex;
    gap: 8px;
  }
  
  .settings-button,
  .refresh-button {
    flex: 1;
    height: 40px;
    font-size: 14px;
    justify-content: center;
  }
  
  /* 邀请设置对话框移动端优化 */
  .settings-dialog {
    :deep(.el-dialog) {
      width: 95% !important;
      margin: 2vh auto !important;
      max-height: 96vh;
      display: flex;
      flex-direction: column;
      
      .el-dialog__header {
        padding: 15px 15px 10px 15px;
        font-size: 16px;
        flex-shrink: 0;
        border-bottom: 1px solid #ebeef5;
      }
      
      .el-dialog__body {
        padding: 15px;
        flex: 1;
        overflow-y: auto;
        overflow-x: hidden;
        -webkit-overflow-scrolling: touch;
        min-height: 0;
      }
      
      .el-dialog__footer {
        padding: 10px 15px 15px 15px;
        flex-shrink: 0;
        border-top: 1px solid #ebeef5;
      }
    }
  }
  
  .settings-dialog-content {
    width: 100%;
    box-sizing: border-box;
    overflow-x: hidden;
  }
  
  /* 邀请设置表单移动端优化 */
  .invite-settings-form {
    width: 100% !important;
    max-width: 100% !important;
    box-sizing: border-box !important;
    overflow-x: hidden;
    padding: 0;
    margin: 0;
    
    :deep(.el-form-item) {
      margin-bottom: 24px;
      width: 100% !important;
      max-width: 100% !important;
      box-sizing: border-box !important;
      padding: 0;
      margin-left: 0 !important;
      margin-right: 0 !important;
      
      .el-form-item__label {
        display: none !important;
      }
      
      .el-form-item__content {
        margin-left: 0 !important;
        margin-right: 0 !important;
        width: 100% !important;
        max-width: 100% !important;
        box-sizing: border-box !important;
        padding: 0;
      }
    }
    
    .form-item-wrapper {
      width: 100% !important;
      max-width: 100% !important;
      display: flex;
      flex-direction: column;
      align-items: stretch;
      box-sizing: border-box !important;
      gap: 12px;
    }
    
    .form-item-label {
      font-size: 15px;
      font-weight: 500;
      color: #303133;
      line-height: 1.5;
      text-align: left !important;
      width: 100% !important;
      max-width: 100% !important;
      box-sizing: border-box !important;
      word-wrap: break-word;
      word-break: break-all;
      order: 1;
      margin: 0 !important;
      padding: 0 !important;
    }
    
    .settings-input-number {
      width: 100% !important;
      max-width: 100% !important;
      min-width: 0 !important;
      margin: 0 !important;
      box-sizing: border-box !important;
      order: 2;
      
      :deep(.el-input__wrapper) {
        width: 100% !important;
        max-width: 100% !important;
        height: 44px;
        box-sizing: border-box !important;
        padding: 0 40px 0 11px;
      }
      
      :deep(.el-input__inner) {
        width: 100% !important;
        height: 44px;
        line-height: 44px;
        font-size: 16px;
        text-align: left !important;
        padding: 0 !important;
        box-sizing: border-box;
      }
      
      :deep(.el-input-number__increase),
      :deep(.el-input-number__decrease) {
        width: 32px;
        height: 22px;
        right: 1px;
      }
    }
  }
  
  .settings-alert {
    margin-bottom: 16px !important;
    margin-left: 0 !important;
    margin-right: 0 !important;
    width: 100% !important;
    max-width: 100% !important;
    box-sizing: border-box !important;
    overflow-x: hidden;
    
    :deep(.el-alert__title) {
      font-size: 14px;
      font-weight: 600;
      margin-bottom: 8px;
      text-align: left !important;
      word-wrap: break-word;
      word-break: break-all;
    }
    
    :deep(.el-alert__content) {
      width: 100% !important;
      max-width: 100% !important;
      box-sizing: border-box !important;
      overflow-x: hidden;
    }
  }
  
  .alert-content {
    font-size: 12px;
    line-height: 1.6;
    text-align: left !important;
    width: 100% !important;
    max-width: 100% !important;
    box-sizing: border-box !important;
    word-wrap: break-word;
    word-break: break-all;
    
    :is(p) {
      margin: 0 0 6px 0;
      text-align: left !important;
      width: 100%;
      word-wrap: break-word;
      word-break: break-all;
      
      &:last-child {
        margin-bottom: 0;
      }
    }
    
    :is(strong) {
      color: #303133;
      font-weight: 600;
    }
  }
  
  .alert-note {
    color: #909399 !important;
    margin-top: 6px !important;
    font-size: 11px !important;
    line-height: 1.5 !important;
    text-align: left !important;
    word-wrap: break-word;
    word-break: break-all;
  }
  
  /* 对话框按钮优化 */
  .dialog-footer-buttons {
    display: flex;
    gap: 12px;
    width: 100%;
    box-sizing: border-box;
    
    .mobile-action-btn {
      flex: 1;
      height: 44px;
      font-size: 15px;
      margin: 0;
    }
  }

  /* 表单优化 */
  .filter-form {
    :deep(.el-form-item) {
      margin-bottom: 15px;
      display: block;
      width: 100%;
      
      .el-form-item__label {
        width: 100% !important;
        text-align: left;
        margin-bottom: 5px;
        padding: 0;
      }
      
      .el-form-item__content {
        margin-left: 0 !important;
        width: 100%;
      }
    }
    
    :deep(.el-input),
    :deep(.el-input-number),
    :deep(.el-select) {
      width: 100% !important;
    }
  }

  /* 表格优化 */
  :deep(.el-table) {
    font-size: 12px;
    
    .el-table__cell {
      padding: 8px 4px;
      word-break: break-word;
    }

    .el-table__header th {
      padding: 8px 4px;
      font-size: 12px;
      font-weight: 600;
    }
    
    /* 表格横向滚动 */
    .el-table__body-wrapper {
      overflow-x: auto;
      -webkit-overflow-scrolling: touch;
    }
  }

  /* 邀请设置容器 */
  .settings-container {
    max-width: 800px;
    margin: 0 auto;
    padding: 0 10px;
  }
  
  /* 邀请设置对话框样式 */
  .settings-dialog {
    :deep(.el-dialog) {
      .el-dialog__header {
        padding: 20px 20px 15px 20px;
        border-bottom: 1px solid #ebeef5;
      }
      
      .el-dialog__body {
        padding: 20px;
      }
      
      .el-dialog__footer {
        padding: 15px 20px 20px 20px;
        border-top: 1px solid #ebeef5;
      }
    }
  }
  
  .settings-dialog-content {
    width: 100%;
    box-sizing: border-box;
  }
  
  /* 邀请设置表单优化（桌面端） */
  .invite-settings-form {
    :deep(.el-form-item) {
      margin-bottom: 24px;
      
      .el-form-item__label {
        display: none !important;
      }
      
      .el-form-item__content {
        margin-left: 0 !important;
        width: 100%;
      }
    }
    
    .form-item-wrapper {
      width: 100%;
      max-width: 500px;
      display: flex;
      flex-direction: column;
      align-items: stretch;
      gap: 12px;
    }
    
    .form-item-label {
      font-size: 15px;
      font-weight: 500;
      color: #303133;
      line-height: 1.5;
      text-align: left;
      width: 100%;
      order: 1;
      margin: 0;
      padding: 0;
    }
    
    .settings-input-number {
      width: 100% !important;
      margin: 0;
      order: 2;
      
      :deep(.el-input__wrapper) {
        width: 100%;
      }
    }
  }
  
  /* 对话框按钮样式 */
  .dialog-footer-buttons {
    display: flex;
    gap: 12px;
    justify-content: flex-end;
    
    .mobile-action-btn {
      min-width: 100px;
    }
  }
  
  .settings-alert {
    margin-bottom: 24px;
    max-width: 600px;
    margin-left: auto;
    margin-right: auto;
    
    :deep(.el-alert__content) {
      width: 100%;
    }
  }
  
  .alert-content {
    line-height: 1.8;
    font-size: 14px;
    
    :is(p) {
      margin: 0 0 8px 0;
      
      &:last-child {
        margin-bottom: 0;
      }
    }
    
    :is(strong) {
      color: #303133;
      font-weight: 600;
    }
  }
  
  .alert-note {
    color: #909399 !important;
    margin-top: 10px !important;
    font-size: 13px !important;
  }

  /* 对话框优化 */
  :deep(.el-dialog) {
    width: 95% !important;
    margin: 5vh auto !important;
    
    .el-dialog__body {
      padding: 15px;
    }
  }

  /* 分页优化 */
  :deep(.el-pagination) {
    flex-wrap: wrap;
    justify-content: center;
    
    .el-pagination__sizes,
    .el-pagination__jump {
      margin-top: 10px;
      width: 100%;
      justify-content: center;
    }
  }
}

@media (max-width: 480px) {
  .admin-invites {
    padding: 5px 3px;
    width: 100%;
    box-sizing: border-box;
    overflow-x: hidden;
  }

  :deep(.el-card) {
    .el-card__header {
      padding: 10px 8px;
      font-size: 13px;
    }
    
    .el-card__body {
      padding: 10px 8px;
    }
  }
  
  .card-header-wrapper {
    gap: 10px;
  }
  
  .header-buttons {
    gap: 6px;
  }
  
  .settings-button,
  .refresh-button {
    height: 38px;
    font-size: 13px;
    padding: 0 10px;
  }
  
  .mobile-action-bar {
    padding: 10px 8px;
    width: 100%;
    box-sizing: border-box;
  }
  
  .mobile-filter-buttons .el-button {
    height: 38px;
    font-size: 12px;
    padding: 0 10px;
  }
  
  /* 邀请设置对话框小屏幕优化 */
  .settings-dialog {
    :deep(.el-dialog) {
      width: 98% !important;
      margin: 1vh auto !important;
      max-height: 98vh;
      
      .el-dialog__header {
        padding: 12px 12px 8px 12px;
        font-size: 15px;
      }
      
      .el-dialog__body {
        padding: 12px;
        max-height: calc(98vh - 120px);
      }
      
      .el-dialog__footer {
        padding: 8px 12px 12px 12px;
      }
    }
  }
  
  .invite-settings-form {
    .form-item-wrapper {
      gap: 10px;
    }
    
    .form-item-label {
      font-size: 14px;
    }
    
    .settings-input-number {
      :deep(.el-input__wrapper) {
        height: 42px;
      }
      
      :deep(.el-input__inner) {
        height: 42px;
        line-height: 42px;
        font-size: 15px;
      }
    }
  }
  
  .invite-settings-form {
    width: 100% !important;
    max-width: 100% !important;
    padding: 0 !important;
    margin: 0 !important;
    box-sizing: border-box !important;
    
    :deep(.el-form-item) {
      margin-bottom: 18px;
      width: 100% !important;
      max-width: 100% !important;
      padding: 0 !important;
      margin-left: 0 !important;
      margin-right: 0 !important;
      
      .el-form-item__label {
        font-size: 14px;
        margin-bottom: 8px;
        text-align: left !important;
        padding: 0 !important;
        width: 100% !important;
        max-width: 100% !important;
      }
      
      .el-form-item__content {
        width: 100% !important;
        max-width: 100% !important;
        padding: 0 !important;
        margin: 0 !important;
      }
    }
    
    .form-item-wrapper {
      width: 100% !important;
      max-width: 100% !important;
      gap: 6px;
    }
    
    .settings-input-number {
      width: 100% !important;
      max-width: 100% !important;
      margin: 0 !important;
      box-sizing: border-box !important;
      
      :deep(.el-input__wrapper) {
        height: 42px;
        width: 100% !important;
        max-width: 100% !important;
        padding: 0 38px 0 10px;
        box-sizing: border-box !important;
      }
      
      :deep(.el-input__inner) {
        height: 42px;
        line-height: 42px;
        font-size: 15px;
        text-align: left !important;
        padding: 0 !important;
        width: 100% !important;
        box-sizing: border-box;
      }
      
      :deep(.el-input-number__increase),
      :deep(.el-input-number__decrease) {
        width: 30px;
        height: 21px;
      }
    }
    
    .form-item-tip {
      font-size: 12px;
      text-align: left !important;
      padding: 0 !important;
      width: 100% !important;
      max-width: 100% !important;
      line-height: 1.5;
      margin: 0 !important;
    }
    
    .save-settings-btn {
      height: 42px;
      font-size: 14px;
      width: 100% !important;
      max-width: 100% !important;
      margin: 14px 0 0 0 !important;
      padding: 0 12px;
      box-sizing: border-box !important;
    }
  }
  
  .settings-alert {
    margin-bottom: 14px !important;
    width: 100% !important;
    max-width: 100% !important;
    margin-left: 0 !important;
    margin-right: 0 !important;
    box-sizing: border-box !important;
    
    :deep(.el-alert__title) {
      font-size: 13px;
      text-align: left !important;
      margin-bottom: 6px;
    }
    
    :deep(.el-alert__content) {
      width: 100% !important;
      max-width: 100% !important;
    }
  }
  
  .alert-content {
    font-size: 11px;
    line-height: 1.5;
    text-align: left !important;
    
    :is(p) {
      text-align: left !important;
      margin: 0 0 5px 0;
    }
  }
  
  .alert-note {
    font-size: 10px !important;
    margin-top: 5px !important;
    text-align: left !important;
    line-height: 1.4 !important;
  }

  :deep(.el-table) {
    font-size: 11px;
    
    .el-table__cell {
      padding: 6px 2px;
    }

    .el-table__header th {
      padding: 6px 2px;
      font-size: 11px;
    }
  }

  /* 对话框进一步优化 */
  :deep(.el-dialog) {
    width: 98% !important;
    margin: 2vh auto !important;
    
    .el-dialog__body {
      padding: 12px;
    }
  }
}

@media (min-width: 769px) {
  .mobile-action-bar {
    display: none !important;
  }
}

/* 统计卡片统一样式 */
.statistics-row {
  margin-bottom: 20px;
  
  .stat-card {
    height: 100%;
    transition: all 0.3s ease;
    
    &:hover {
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    }
  }
  
  .stat-content {
    text-align: center;
    padding: 10px;
  }
  
  .stat-value {
    font-size: 32px;
    font-weight: bold;
    margin-bottom: 10px;
    line-height: 1.2;
    word-break: break-all;
    overflow-wrap: break-word;
  }
  
  .stat-label {
    color: #909399;
    font-size: 14px;
    line-height: 1.4;
    word-break: break-all;
    overflow-wrap: break-word;
    min-height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  
  @media (max-width: 768px) {
    .stat-content {
      padding: 12px 8px;
    }
    
    .stat-value {
      font-size: 24px;
      margin-bottom: 8px;
    }
    
    .stat-label {
      font-size: 12px;
      min-height: 32px;
    }
  }
  
  @media (max-width: 480px) {
    .stat-content {
      padding: 10px 6px;
    }
    
    .stat-value {
      font-size: 20px;
      margin-bottom: 6px;
    }
    
    .stat-label {
      font-size: 11px;
      min-height: 28px;
    }
  }
}
</style>

