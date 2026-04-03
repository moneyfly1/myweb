<template>
  <div class="list-container invites-container">
    <div class="stats-row" style="margin-top: 0;">
      <div class="stat-card">
        <div class="stat-number">{{ stats.total_invites || 0 }}</div>
        <div class="stat-label">总邀请人数</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ stats.registered_invites || 0 }}</div>
        <div class="stat-label">已注册人数</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ stats.purchased_invites || 0 }}</div>
        <div class="stat-label">已购买人数</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">¥{{ (stats.total_reward || 0).toFixed(2) }}</div>
        <div class="stat-label">累计奖励</div>
      </div>
    </div>
    <el-alert
      v-if="inviteRewardSettings.inviter_reward > 0 || inviteRewardSettings.invitee_reward > 0"
      title="邀请奖励说明"
      type="info"
      :closable="false"
      class="reward-alert"
      style="margin-bottom: 12px;"
    >
      <template #default>
        <div style="line-height: 1.8;">
          <p v-if="inviteRewardSettings.inviter_reward > 0">
            <strong>邀请人奖励：</strong>当被邀请人首次购买套餐后，您将获得 <span style="color: #67c23a; font-weight: bold;">¥{{ inviteRewardSettings.inviter_reward.toFixed(2) }}</span> 的奖励
          </p>
          <p v-if="inviteRewardSettings.invitee_reward > 0">
            <strong>被邀请人奖励：</strong>新用户使用您的邀请码注册后，将立即获得 <span style="color: #409eff; font-weight: bold;">¥{{ inviteRewardSettings.invitee_reward.toFixed(2) }}</span> 的奖励
          </p>
        </div>
      </template>
    </el-alert>
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>我的邀请码</span>
          <div class="header-actions">
            <el-button type="primary" @click="showGenerateDialog = true" :icon="Plus">
              生成新邀请码
            </el-button>
          </div>
        </div>
      </template>
      <div class="mobile-only" style="margin-bottom: 12px;">
        <el-button type="primary" @click="showGenerateDialog = true" :icon="Plus" style="width: 100%;">
          生成新邀请码
        </el-button>
      </div>
      <div class="table-wrapper">
        <el-table
          ref="inviteTableRef"
          :data="inviteCodes"
          v-loading="loading"
          :empty-text="inviteCodes.length === 0 ? '暂无邀请码，点击上方按钮生成' : '暂无数据'"
          border
          stripe
          style="width: 100%"
          @header-dragend="handleInviteColumnResize"
        >
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
      <div class="mobile-card-list" v-if="inviteCodes.length > 0 || !loading">
        <div v-for="code in inviteCodes" :key="code.id" class="mobile-card">
          <div class="card-row">
            <span class="label">状态</span>
            <span class="value">
              <el-tag :type="getIsValid(code) ? 'success' : 'danger'" size="small">{{ getIsValid(code) ? '有效' : '无效' }}</el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">邀请码</span>
            <span class="value">{{ code.code }}</span>
          </div>
          <div class="card-row">
            <span class="label">已使用</span>
            <span class="value">{{ code.used_count || 0 }} / {{ getMaxUses(code.max_uses) }}</span>
          </div>
          <div class="card-row">
            <span class="label">过期时间</span>
            <span class="value">
              <span v-if="code.expires_at && code.expires_at !== 'null'">{{ formatDate(code.expires_at) }}</span>
              <span v-else class="text-muted">永不过期</span>
            </span>
          </div>
          <div class="card-row link-row">
            <span class="label">邀请链接</span>
            <span class="value">
              <el-input :value="code.invite_link" readonly size="small">
                <template #append>
                  <el-button @click="copyLink(code.invite_link)" :icon="DocumentCopy" />
                </template>
              </el-input>
            </span>
          </div>
          <div class="card-actions">
            <el-button type="primary" size="small" @click="copyLink(code.invite_link)" :icon="DocumentCopy">复制链接</el-button>
            <el-button type="danger" size="small" @click="deleteCode(code)" :icon="Delete">删除</el-button>
          </div>
        </div>
        <el-empty v-if="inviteCodes.length === 0 && !loading" description="暂无邀请码，点击上方按钮生成" />
      </div>
    </el-card>
    <el-card v-if="stats.recent_invites && stats.recent_invites.length > 0" class="list-card">
      <template #header>
        <div class="card-header">
          <span>最近邀请记录</span>
        </div>
      </template>
      <div class="table-wrapper">
        <el-table
          ref="recentTableRef"
          :data="stats.recent_invites"
          border
          stripe
          size="small"
          style="width: 100%"
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
      <div class="mobile-card-list">
        <div v-for="(invite, index) in stats.recent_invites" :key="index" class="mobile-card">
          <div class="card-row">
            <span class="label">状态</span>
            <span class="value">
              <el-tag :type="invite.has_purchased ? 'success' : 'info'" size="small">{{ invite.has_purchased ? '已购买' : '未购买' }}</el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">被邀请人</span>
            <span class="value">{{ invite.invitee_username || '-' }}</span>
          </div>
          <div class="card-row">
            <span class="label">邮箱</span>
            <span class="value">{{ invite.invitee_email || '-' }}</span>
          </div>
          <div class="card-row">
            <span class="label">注册时间</span>
            <span class="value">{{ formatDate(invite.created_at) }}</span>
          </div>
          <div class="card-row" v-if="invite.total_consumption !== undefined">
            <span class="label">累计消费</span>
            <span class="value">¥{{ invite.total_consumption.toFixed(2) }}</span>
          </div>
          <div class="card-row" v-if="invite.reward_given !== undefined">
            <span class="label">奖励状态</span>
            <el-tag :type="invite.reward_given ? 'success' : 'warning'" size="small">{{ invite.reward_given ? '已发放' : '未发放' }}</el-tag>
          </div>
        </div>
      </div>
    </el-card>
    <el-dialog
      v-model="showGenerateDialog"
      title="生成邀请码"
      :width="isMobile ? '100%' : '500px'"
      :close-on-click-modal="!isMobile"
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
            style="width: 100%"
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
            style="width: 100%"
            placeholder="留空表示永不过期"
          />
          <div class="form-tip">邀请码有效期，留空表示永不过期</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer-buttons">
          <el-button 
            @click="showGenerateDialog = false" 
            :size="isMobile ? 'large' : 'default'"
            :style="isMobile ? 'width: 100%; margin-bottom: 10px; min-height: 44px;' : ''"
          >
            取消
          </el-button>
          <el-button 
            type="primary" 
            @click="generateCode" 
            :loading="generating"
            :size="isMobile ? 'large' : 'default'"
            :style="isMobile ? 'width: 100%; min-height: 44px;' : ''"
          >
            生成
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>
<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, DocumentCopy, Delete } from '@element-plus/icons-vue'
import { inviteAPI } from '@/utils/api'
import { copyToClipboard as copyText } from '@/utils/textSelection'
const loading = ref(false)
const generating = ref(false)
const showGenerateDialog = ref(false)
const inviteCodes = ref([])
const isMobile = ref(window.innerWidth <= 768)
const inviteTableRef = ref(null)
const recentTableRef = ref(null)

const INVITE_STORAGE_KEY = 'invites_table_settings'
const RECENT_STORAGE_KEY = 'invites_recent_table_settings'

const columnWidths = reactive({
  code: 120,
  invite_link: 240,
  used_count: 100,
  expires_at: 180,
  status: 100,
  actions: 150
})
const recentColumnWidths = reactive({
  invitee_username: 120,
  invitee_email: 180,
  created_at: 180,
  has_purchased: 100,
  total_consumption: 120,
  reward_given: 100
})

const loadInviteSettings = () => {
  try {
    const saved = localStorage.getItem(INVITE_STORAGE_KEY)
    if (saved) {
      const s = JSON.parse(saved)
      if (s.columnWidths) Object.assign(columnWidths, s.columnWidths)
    }
  } catch (e) {
    console.warn('加载邀请码表设置失败:', e)
  }
}
const loadRecentSettings = () => {
  try {
    const saved = localStorage.getItem(RECENT_STORAGE_KEY)
    if (saved) {
      const s = JSON.parse(saved)
      if (s.columnWidths) Object.assign(recentColumnWidths, s.columnWidths)
    }
  } catch (e) {
    console.warn('加载最近邀请表设置失败:', e)
  }
}
const saveInviteSettings = () => {
  try {
    localStorage.setItem(INVITE_STORAGE_KEY, JSON.stringify({ columnWidths: { ...columnWidths } }))
  } catch (e) {
    console.warn('保存邀请码表设置失败:', e)
  }
}
const saveRecentSettings = () => {
  try {
    localStorage.setItem(RECENT_STORAGE_KEY, JSON.stringify({ columnWidths: { ...recentColumnWidths } }))
  } catch (e) {
    console.warn('保存最近邀请表设置失败:', e)
  }
}

const INVITE_COLUMN_KEYS = ['code', 'invite_link', 'used_count', 'expires_at', 'status', 'actions']
const RECENT_COLUMN_KEYS = ['invitee_username', 'invitee_email', 'created_at', 'has_purchased', 'total_consumption', 'reward_given']

let inviteResizeTimer = null
const handleInviteColumnResize = () => {
  if (inviteResizeTimer) clearTimeout(inviteResizeTimer)
  inviteResizeTimer = setTimeout(() => {
    if (inviteTableRef.value?.$el) {
      const cells = inviteTableRef.value.$el.querySelectorAll('.el-table__header-wrapper thead th')
      cells.forEach((cell, index) => {
        if (INVITE_COLUMN_KEYS[index] && cell.offsetWidth > 0) columnWidths[INVITE_COLUMN_KEYS[index]] = cell.offsetWidth
      })
      saveInviteSettings()
    }
  }, 300)
}
let recentResizeTimer = null
const handleRecentColumnResize = () => {
  if (recentResizeTimer) clearTimeout(recentResizeTimer)
  recentResizeTimer = setTimeout(() => {
    if (recentTableRef.value?.$el) {
      const cells = recentTableRef.value.$el.querySelectorAll('.el-table__header-wrapper thead th')
      cells.forEach((cell, index) => {
        if (RECENT_COLUMN_KEYS[index] && cell.offsetWidth > 0) recentColumnWidths[RECENT_COLUMN_KEYS[index]] = cell.offsetWidth
      })
      saveRecentSettings()
    }
  }, 300)
}

const handleResize = () => {
  isMobile.value = window.innerWidth <= 768
}
onMounted(async () => {
  window.addEventListener('resize', handleResize)
  handleResize()
  loadInviteSettings()
  loadRecentSettings()
  // 并发加载三个独立的数据源，提高页面加载速度
  await Promise.all([
    loadInviteRewardSettings(),
    loadInviteCodes(),
    loadStats()
  ])
})
onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
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
    await ElMessageBox.confirm(
      `确定要删除邀请码 "${code.code}" 吗？${code.used_count > 0 ? '（已有使用记录，将禁用而非删除）' : ''}`,
      '确认删除',
      { type: 'warning' }
    )
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
  try {
    const date = new Date(dateStr)
    if (isNaN(date.getTime())) return '-'
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  } catch (e) {
    return '-'
  }
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
@use '@/styles/list-common.scss';

.mobile-only {
  display: none !important;
  @media (max-width: 768px) {
    display: block !important;
  }
}
.reward-alert {
  margin-bottom: 12px;
  border-radius: 8px;
  border-left: 4px solid #409eff;
  :deep(.el-alert__content) {
    .el-alert__title {
      font-size: 15px;
      font-weight: 600;
      color: #303133;
    }
    .el-alert__description {
      font-size: 14px;
      line-height: 1.8;
      color: #606266;
      :is(p) {
        margin: 8px 0;
        :is(strong) {
          color: #303133;
          font-weight: 600;
        }
      }
    }
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
.mobile-card-list .link-row .value {
  flex: 1;
  min-width: 0;
  :deep(.el-input) {
    width: 100%;
  }
  :deep(.el-input__wrapper) {
    padding-right: 36px;
  }
}
.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.5;
}
:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
}
:deep(.el-input-number .el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
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
    .el-table__body-wrapper {
      overflow-x: auto;
      -webkit-overflow-scrolling: touch;
    }
    .expires-column,
    .action-column {
      display: none;
    }
    .link-column {
      min-width: 150px;
    }
  }
  :deep(.el-table__body-wrapper) {
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }
  .link-cell {
    :deep(.el-input) {
      font-size: 11px;
    }
  }
  .generate-invite-dialog {
    :deep(.el-dialog) {
      margin: 0 !important;
      width: 100% !important;
      max-width: 100% !important;
      height: 100vh !important;
      max-height: 100vh !important;
      border-radius: 0 !important;
      display: flex;
      flex-direction: column;
    }
    :deep(.el-dialog__header) {
      padding: 16px !important;
      flex-shrink: 0;
      border-bottom: 1px solid #e5e7eb;
      .el-dialog__title {
        font-size: 18px;
        font-weight: 600;
      }
      .el-dialog__headerbtn {
        top: 16px;
        right: 16px;
        width: 32px;
        height: 32px;
        .el-dialog__close {
          font-size: 20px;
        }
      }
    }
    :deep(.el-dialog__body) {
      padding: 16px !important;
      flex: 1;
      overflow-y: auto;
      -webkit-overflow-scrolling: touch;
    }
    :deep(.el-dialog__footer) {
      padding: 12px 16px 16px 16px !important;
      flex-shrink: 0;
      border-top: 1px solid #e5e7eb;
    }
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
  :deep(.el-input-number) {
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
  :deep(.el-card__body) {
    padding: 10px;
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
  :deep(.el-table__body-wrapper) {
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }
}
</style>
