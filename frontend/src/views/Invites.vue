<template>
  <div class="list-container invites-container">
    <div class="breadcrumb">首页 / 我的邀请</div>
    <div class="page-header">
      <div class="page-title">
        <h1>我的邀请</h1>
      </div>
      <div class="actions">
        <el-button type="primary" @click="showGenerateDialog = true">
          生成邀请码
        </el-button>
      </div>
    </div>
    <div class="stats-row invites-stats-row">
      <div class="stat-card">
        <div class="stat-icon">I</div>
        <div>
          <div class="stat-value">{{ stats.total_invites || 0 }}</div>
          <div class="stat-label">总邀请人数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">R</div>
        <div>
          <div class="stat-value">{{ stats.registered_invites || 0 }}</div>
          <div class="stat-label">已注册人数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">P</div>
        <div>
          <div class="stat-value">{{ stats.purchased_invites || 0 }}</div>
          <div class="stat-label">已购买人数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">¥</div>
        <div>
          <div class="stat-value">¥{{ (stats.total_reward || 0).toFixed(2) }}</div>
          <div class="stat-label">累计奖励</div>
        </div>
      </div>
    </div>
    <TipBlock
      v-if="inviteRewardSettings.inviter_reward > 0 || inviteRewardSettings.invitee_reward > 0"
      title="邀请奖励说明"
      type="info"
      :closable="false"
      class="reward-alert"
    >
      <div class="reward-summary">
        <div v-if="inviteRewardSettings.inviter_reward > 0" class="reward-item">
          <span class="reward-label">邀请人奖励</span>
          <span class="reward-value">¥{{ inviteRewardSettings.inviter_reward.toFixed(2) }}</span>
          <span class="reward-note">被邀请人首次购买套餐后发放</span>
        </div>
        <div v-if="inviteRewardSettings.invitee_reward > 0" class="reward-item">
          <span class="reward-label">被邀请人奖励</span>
          <span class="reward-value">¥{{ inviteRewardSettings.invitee_reward.toFixed(2) }}</span>
          <span class="reward-note">新用户使用邀请码注册后发放</span>
        </div>
      </div>
    </TipBlock>
    <div class="invites-workspace">
      <el-card class="list-card">
        <template #header>
          <div class="card-header">
            <span>我的邀请码</span>
          </div>
        </template>
        <ResponsiveDataView
          :data="inviteCodes"
          :fields="mobileInviteFields"
          :loading="loading"
          title-field="code"
          empty-title="暂无邀请码"
          empty-description="点击上方按钮生成邀请码"
        >
          <template #table>
            <div class="table-wrapper">
              <el-table
                ref="inviteTableRef"
                :data="inviteCodes"
                v-loading="loading"
                border
                stripe
                class="invite-table"
                @header-dragend="handleInviteColumnResize"
              >
                <template #empty>
                  <EmptyState
                    title="暂无邀请码"
                    description="点击上方按钮生成邀请码。"
                    action-text="生成邀请码"
                    :loading="loading"
                    @action="showGenerateDialog = true"
                  />
                </template>
                <el-table-column prop="code" label="邀请码" :min-width="columnWidths.code" :width="columnWidths.code" resizable>
                  <template #default="scope">
                    <el-tag>{{ scope.row.code }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="invite_link" label="邀请链接" :min-width="columnWidths.invite_link" resizable class-name="link-column">
                  <template #default="scope">
                    <div class="link-cell">
                      <el-input :value="scope.row.invite_link" readonly size="small">
                        <template #append>
                          <el-button @click="copyLink(scope.row.invite_link)" :icon="DocumentCopy" />
                        </template>
                      </el-input>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column prop="used_count" label="已使用" :width="columnWidths.used_count" resizable align="center">
                  <template #default="scope">
                    <span>{{ scope.row.used_count || 0 }} / {{ getMaxUses(scope.row.max_uses) }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="expires_at" label="过期时间" :width="columnWidths.expires_at" resizable>
                  <template #default="scope">
                    <span v-if="scope.row.expires_at && scope.row.expires_at !== 'null'">{{ formatDate(scope.row.expires_at) }}</span>
                    <span v-else class="text-muted">永不过期</span>
                  </template>
                </el-table-column>
                <el-table-column prop="is_valid" label="状态" :width="columnWidths.status" resizable align="center">
                  <template #default="scope">
                    <el-tag :type="getIsValid(scope.row) ? 'success' : 'danger'">
                      {{ getIsValid(scope.row) ? '有效' : '无效' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="操作" :width="columnWidths.actions" resizable align="center">
                  <template #default="scope">
                    <el-button type="primary" link size="small" @click="copyLink(scope.row.invite_link)" :icon="DocumentCopy">复制链接</el-button>
                    <el-button type="danger" link size="small" @click="deleteCode(scope.row)" :icon="Delete">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </template>
          <template #header="{ item }">
            <div class="invite-mobile-header">
              <span>{{ item.code }}</span>
              <el-tag :type="getIsValid(item) ? 'success' : 'danger'" size="small">{{ getIsValid(item) ? '有效' : '无效' }}</el-tag>
            </div>
          </template>
          <template #actions="{ item }">
            <el-button type="primary" size="small" @click="copyLink(item.invite_link)" :icon="DocumentCopy">复制链接</el-button>
            <el-button type="danger" size="small" @click="deleteCode(item)" :icon="Delete">删除</el-button>
          </template>
        </ResponsiveDataView>
      </el-card>
      <div class="section-stack invites-side">
        <el-card class="list-card">
          <template #header>
            <div class="card-header">
              <span>最近邀请记录</span>
            </div>
          </template>
          <ResponsiveDataView
            v-if="stats.recent_invites && stats.recent_invites.length > 0"
            :data="stats.recent_invites"
            :fields="mobileRecentFields"
            title-field="invitee_username"
            empty-title="暂无邀请记录"
          >
            <template #table>
              <div class="table-wrapper">
                <el-table
                  ref="recentTableRef"
                  :data="stats.recent_invites"
                  border
                  stripe
                  size="small"
                  class="invite-table"
                  @header-dragend="handleRecentColumnResize"
                >
                  <el-table-column prop="invitee_username" label="被邀请人" :width="recentColumnWidths.invitee_username" resizable />
                  <el-table-column prop="invitee_email" label="邮箱" :min-width="recentColumnWidths.invitee_email" resizable />
                  <el-table-column prop="created_at" label="注册时间" :width="recentColumnWidths.created_at" resizable>
                    <template #default="scope">{{ formatDate(scope.row.created_at) }}</template>
                  </el-table-column>
                  <el-table-column prop="has_purchased" label="已购买" :width="recentColumnWidths.has_purchased" resizable align="center">
                    <template #default="scope">
                      <el-tag :type="scope.row.has_purchased ? 'success' : 'info'" size="small">{{ scope.row.has_purchased ? '是' : '否' }}</el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="total_consumption" label="累计消费" :width="recentColumnWidths.total_consumption" resizable align="right">
                    <template #default="scope">¥{{ (scope.row.total_consumption || 0).toFixed(2) }}</template>
                  </el-table-column>
                  <el-table-column prop="reward_given" label="奖励状态" :width="recentColumnWidths.reward_given" resizable align="center">
                    <template #default="scope">
                      <el-tag :type="scope.row.reward_given ? 'success' : 'warning'" size="small">{{ scope.row.reward_given ? '已发放' : '未发放' }}</el-tag>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </template>
          </ResponsiveDataView>
          <div v-else class="card-body">
            <div class="ticket-item">
              <div class="item-title">暂无最近邀请记录</div>
              <div class="item-meta">邀请用户注册或购买后会在这里展示注册、购买和奖励状态。</div>
            </div>
          </div>
        </el-card>
      </div>
    </div>
    <AppDialog
      v-model="showGenerateDialog"
      title="生成邀请码"
      width="500px"
      mobile-width="94%"
      :loading="generating"
      class="generate-invite-dialog"
    >
      <el-form 
        :model="generateForm" 
        :label-width="isMobile ? '0' : '120px'"
        :label-position="isMobile ? 'top' : 'right'"
        class="generate-invite-form"
      >
        <el-form-item :label="isMobile ? '' : '最大使用次数'">
          <template #label v-if="isMobile">
            <span class="form-label">最大使用</span>
          </template>
          <el-input-number 
            v-model="generateForm.max_uses" 
            :min="1" 
            :max="1000"
            :size="isMobile ? 'large' : 'default'"
            :controls-position="isMobile ? 'right' : 'default'"
            class="full-width-control"
            placeholder="留空表示无限制"
          />
          <div class="form-tip">邀请码最多可被使用多少次（留空表示无限制）</div>
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '有效期（天）'">
          <template #label v-if="isMobile">
            <span class="form-label">有效期</span>
          </template>
          <el-input-number 
            v-model="generateForm.expires_days" 
            :min="1" 
            :max="365"
            :size="isMobile ? 'large' : 'default'"
            :controls-position="isMobile ? 'right' : 'default'"
            class="full-width-control"
            placeholder="留空表示永不过期"
          />
          <div class="form-tip">邀请码有效期，留空表示永不过期</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <FormActionBar
          :loading="generating"
          submit-text="生成"
          @cancel="showGenerateDialog = false"
          @submit="generateCode"
        />
      </template>
    </AppDialog>
  </div>
</template>
<script setup>
import { computed, ref, reactive, onMounted } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { DocumentCopy, Delete } from '@element-plus/icons-vue'
import { inviteAPI } from '@/utils/api'
import { copyToClipboard as copyText } from '@/utils/textSelection'
import { useMobile } from '@/composables/useMobile'
import { usePersistentTableColumns } from '@/composables/usePersistentTableColumns'
import { confirmDelete } from '@/utils/confirmAction'
import { formatDateTime } from '@/utils/date'
import { formatMoney } from '@/utils/format'
import AppDialog from '@/components/AppDialog.vue'
import EmptyState from '@/components/EmptyState.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import TipBlock from '@/components/TipBlock.vue'
const loading = ref(false)
const generating = ref(false)
const showGenerateDialog = ref(false)
const inviteCodes = ref([])
const isMobile = useMobile()
const inviteTableRef = ref(null)
const recentTableRef = ref(null)

const INVITE_STORAGE_KEY = 'invites_table_settings'
const RECENT_STORAGE_KEY = 'invites_recent_table_settings'

const INVITE_COLUMN_KEYS = ['code', 'invite_link', 'used_count', 'expires_at', 'status', 'actions']
const RECENT_COLUMN_KEYS = ['invitee_username', 'invitee_email', 'created_at', 'has_purchased', 'total_consumption', 'reward_given']
const { columnWidths, handleColumnResize: handleInviteColumnResize } = usePersistentTableColumns(
  INVITE_STORAGE_KEY,
  {
    code: 120,
    invite_link: 240,
    used_count: 100,
    expires_at: 180,
    status: 100,
    actions: 150
  },
  INVITE_COLUMN_KEYS
)
const { columnWidths: recentColumnWidths, handleColumnResize: handleRecentColumnResize } = usePersistentTableColumns(
  RECENT_STORAGE_KEY,
  {
    invitee_username: 120,
    invitee_email: 180,
    created_at: 180,
    has_purchased: 100,
    total_consumption: 120,
    reward_given: 100
  },
  RECENT_COLUMN_KEYS
)

onMounted(async () => {
  await Promise.all([
    loadInviteRewardSettings(),
    loadInviteCodes(),
    loadStats()
  ])
})
const stats = ref({
  total_invites: 0,
  registered_invites: 0,
  purchased_invites: 0,
  total_reward: 0,
  total_consumption: 0,
  recent_invites: []
})
const generateForm = reactive({
  max_uses: 10,
  expires_days: 30
})
const inviteRewardSettings = ref({
  inviter_reward: 0,
  invitee_reward: 0
})
const mobileInviteFields = computed(() => [
  {
    key: 'is_valid',
    label: '状态',
    type: 'tag',
    tagType: (_value, row) => getIsValid(row) ? 'success' : 'danger',
    formatter: (_value, row) => getIsValid(row) ? '有效' : '无效'
  },
  { key: 'used_count', label: '已使用', formatter: (_value, row) => `${row.used_count || 0} / ${getMaxUses(row.max_uses)}` },
  { key: 'expires_at', label: '过期时间', formatter: value => value && value !== 'null' ? formatDate(value) : '永不过期' },
  { key: 'invite_link', label: '邀请链接', type: 'copy', fullWidth: true }
])
const mobileRecentFields = computed(() => [
  {
    key: 'has_purchased',
    label: '状态',
    type: 'tag',
    tagType: value => value ? 'success' : 'info',
    formatter: value => value ? '已购买' : '未购买'
  },
  { key: 'invitee_username', label: '被邀请人', formatter: value => value || '-' },
  { key: 'invitee_email', label: '邮箱', formatter: value => value || '-' },
  { key: 'created_at', label: '注册时间', formatter: value => formatDate(value) },
  { key: 'total_consumption', label: '累计消费', formatter: value => value !== undefined ? formatMoney(value) : '-' },
  {
    key: 'reward_given',
    label: '奖励状态',
    type: 'tag',
    tagType: value => value ? 'success' : 'warning',
    formatter: value => value ? '已发放' : '未发放'
  }
])
const loadInviteRewardSettings = async () => {
  try {
    const response = await inviteAPI.getInviteRewardSettings()
    if (response?.data?.data) {
      inviteRewardSettings.value = {
        inviter_reward: parseFloat(response.data.data.inviter_reward) || 0,
        invitee_reward: parseFloat(response.data.data.invitee_reward) || 0
      }
    }
  } catch (error) {
    if (process.env.NODE_ENV === 'development') {
    }
  }
}
const loadInviteCodes = async () => {
  loading.value = true
  try {
    const response = await inviteAPI.getMyInviteCodes()
    if (response && response.data) {
      const responseData = response.data
      if (responseData.success !== false && responseData.data) {
        if (Array.isArray(responseData.data)) {
          inviteCodes.value = responseData.data
        } else {
          inviteCodes.value = []
        }
      }
      else if (responseData.success === false) {
        const errorMsg = responseData.message || '获取邀请码列表失败'
        ElMessage.error(errorMsg)
        inviteCodes.value = []
      }
      else if (Array.isArray(responseData)) {
        inviteCodes.value = responseData
      }
      else {
        inviteCodes.value = []
      }
    } else {
      inviteCodes.value = []
    }
  } catch (error) {
    const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '未知错误'
    ElMessage.error('获取邀请码列表失败: ' + errorMsg)
    inviteCodes.value = []
  } finally {
    loading.value = false
  }
}
const loadStats = async () => {
  try {
    const response = await inviteAPI.getInviteStats()
    if (response && response.data) {
      const responseData = response.data
      if (responseData.success !== false && responseData.data) {
        const backendStats = responseData.data
        stats.value = {
          total_invites: backendStats.total_invite_count || 0,
          registered_invites: backendStats.total_invite_relations || 0,
          purchased_invites: 0, // 后端未提供此字段，需要从邀请关系中统计
          total_reward: backendStats.total_invite_reward || 0,
          total_consumption: 0,
          recent_invites: [] // 后端未提供此字段
        }
      }
      else if (responseData.total_invite_count !== undefined) {
        stats.value = {
          total_invites: responseData.total_invite_count || 0,
          registered_invites: responseData.total_invite_relations || 0,
          purchased_invites: 0,
          total_reward: responseData.total_invite_reward || 0,
          total_consumption: 0,
          recent_invites: []
        }
      }
    }
  } catch (error) {
    const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '未知错误'
    ElMessage.error('获取邀请统计失败: ' + errorMsg)
  }
}
const generateCode = async () => {
  generating.value = true
  try {
    const requestData = {
      max_uses: Number(generateForm.max_uses) || 0,
      reward_type: 'balance',
      inviter_reward: Number(inviteRewardSettings.value.inviter_reward) || 0,
      invitee_reward: Number(inviteRewardSettings.value.invitee_reward) || 0,
      min_order_amount: 0,
      new_user_only: true
    }
    if (generateForm.expires_days && generateForm.expires_days > 0) {
      const expiresDate = new Date()
      expiresDate.setDate(expiresDate.getDate() + generateForm.expires_days)
      requestData.expires_at = expiresDate.toISOString()
    }
    const response = await inviteAPI.generateInviteCode(requestData)
    if (process.env.NODE_ENV === 'development') {
    }
    const success = response?.data?.success !== false && 
                   (response?.data?.data?.code || response?.data?.code)
    if (success) {
      ElMessage.success('邀请码生成成功')
      showGenerateDialog.value = false
      Object.assign(generateForm, {
        max_uses: 10,
        expires_days: 30
      })
      await Promise.all([
        loadInviteCodes(),
        loadStats()
      ])
      if (process.env.NODE_ENV === 'development') {
      }
    } else {
      const errorMsg = response?.data?.message || '生成邀请码失败'
      ElMessage.error(errorMsg)
    }
  } catch (error) {
    if (process.env.NODE_ENV === 'development') {
      console.error('生成邀请码错误:', error)
    }
    const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '未知错误'
    ElMessage.error('生成邀请码失败: ' + errorMsg)
  } finally {
    generating.value = false
  }
}
const copyLink = async (link) => {
  await copyText(link, '邀请链接已复制到剪贴板')
}
const deleteCode = async (code) => {
  try {
    await confirmDelete(`邀请码 "${code.code}"`, 1, {
      message: `确定要删除邀请码 "${code.code}" 吗？删除后该邀请链接将不可用。${code.used_count > 0 ? '已有使用记录时后端可能执行禁用处理。' : ''}`,
    })
    await inviteAPI.deleteInviteCode(code.id)
    ElMessage.success('删除成功')
    await loadInviteCodes()
    await loadStats()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + (error.response?.data?.message || error.message))
    }
  }
}
const formatDate = (dateStr) => {
  if (!dateStr || dateStr === 'null' || dateStr === null) return '-'
  return formatDateTime(dateStr, 'YYYY-MM-DD HH:mm') || '-'
}
const getMaxUses = (maxUses) => {
  if (!maxUses || maxUses === 'null' || maxUses === null) return '∞'
  if (typeof maxUses === 'object' && maxUses.Int64 !== undefined) {
    return maxUses.Valid ? maxUses.Int64 : '∞'
  }
  if (typeof maxUses === 'number') {
    return maxUses
  }
  return '∞'
}
const getIsValid = (row) => {
  if (row.is_valid !== undefined) {
    return row.is_valid
  }
  if (!row.is_active) {
    return false
  }
  if (row.expires_at && row.expires_at !== 'null' && row.expires_at !== null) {
    try {
      const expiresDate = new Date(row.expires_at)
      if (!isNaN(expiresDate.getTime()) && expiresDate < new Date()) {
        return false
      }
    } catch (e) {
    }
  }
  const maxUses = getMaxUses(row.max_uses)
  if (maxUses !== '∞' && (row.used_count || 0) >= maxUses) {
    return false
  }
  return true
}
</script>
<style scoped lang="scss">
.reward-alert {
  margin-bottom: 12px;
  border-left: 3px solid var(--el-color-primary);

  :deep(.el-alert__title) {
    font-size: 14px;
    font-weight: 600;
  }
}
.invites-stats-row {
  margin-top: 0;
}
.invites-workspace {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  align-items: start;
  gap: 14px;
}
:global(.user-layout .invites-container .invites-workspace) {
  display: grid !important;
  grid-template-columns: minmax(0, 1fr) !important;
  gap: 14px !important;
}
.invites-workspace .list-card {
  min-width: 0;
  width: 100%;
}
.invites-side {
  min-width: 0;
}
.invites-side .list-card {
  min-width: 0;
}
.invites-side .form-row {
  margin-bottom: 12px;
}
.reward-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 6px;
}
.reward-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 2px 10px;
  min-width: 0;
  padding: 8px 10px;
  border: 1px solid #d9ecff;
  border-radius: 6px;
  background: #f8fbff;
}
.reward-label {
  min-width: 0;
  color: #303133;
  font-weight: 600;
}
.reward-value {
  color: var(--el-color-primary);
  font-weight: 700;
  white-space: nowrap;
}
.reward-note {
  grid-column: 1 / -1;
  min-width: 0;
  color: #606266;
  font-size: 12px;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.invite-table,
.full-width-control {
  width: 100%;
}
.table-wrapper {
  display: block;
  min-width: 0;
}
.invite-mobile-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;

  span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 600;
  }
}
.text-muted {
  color: #909399;
}
.link-cell {
  :deep(.el-input-group__append) {
    padding: 0 2px;
  }
}
.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.5;
}
@media (max-width: 768px) {
  .invites-container {
    padding: 10px;
  }
  .stats-section {
    .stat-card {
      padding: 15px;
      .stat-value {
        font-size: 20px;
      }
      .stat-label {
        font-size: 12px;
      }
    }
  }
  .link-cell {
    :deep(.el-input) {
      font-size: 11px;
    }
  }
  .reward-summary {
    grid-template-columns: 1fr;
  }
  .reward-item {
    padding: 8px;
  }
  .invites-workspace {
    grid-template-columns: 1fr;
  }
  .generate-invite-form {
    :deep(.el-form-item) {
      margin-bottom: 24px;
      .el-form-item__label {
        width: 100% !important;
        text-align: left !important;
        margin-bottom: 8px !important;
        padding: 0 !important;
        font-weight: 500;
        font-size: 14px;
        color: #303133;
        line-height: 1.5;
        display: block;
      }
      .el-form-item__content {
        margin-left: 0 !important;
        width: 100%;
      }
    }
    .form-label {
      display: block;
      font-size: 14px;
      font-weight: 500;
      color: #333;
      margin-bottom: 8px;
      line-height: 1.5;
    }
  }
  .invites-container :deep(.el-input-number) {
    width: 100% !important;
    .el-input {
      width: 100% !important;
    }
    .el-input__wrapper {
      width: 100% !important;
      min-height: 44px;
    }
    .el-input__inner {
      font-size: 16px !important;
      min-height: 44px;
      padding: 0 12px;
      text-align: left;
      -webkit-appearance: none;
      appearance: none;
    }
    .el-input-number__decrease,
    .el-input-number__increase {
      width: 36px;
      height: 22px;
      line-height: 22px;
      font-size: 16px;
      border: none;
      background: #f5f7fa;
      color: #606266;
      &:hover {
        color: #409eff;
        background: #ecf5ff;
      }
      &:active {
        background: #d9ecff;
      }
    }
    &.is-controls-right {
      .el-input__wrapper {
        padding-right: 40px;
      }
      .el-input-number__decrease,
      .el-input-number__increase {
        right: 0;
        width: 36px;
        height: 22px;
      }
    }
  }
  .form-tip {
    font-size: 12px;
    color: #909399;
    margin-top: 8px;
    line-height: 1.6;
  }
}
@media (max-width: 480px) {
  .invites-container :deep(.el-card__body) {
    padding: 10px;
  }
}
</style>
