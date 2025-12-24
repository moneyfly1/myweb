<template>
  <div class="invites-container">
    <el-card>
      <template #header>
        <div class="header-content">
          <span>我的邀请</span>
        </div>
      </template>

      <!-- 邀请统计 -->
      <div class="stats-section">
        <el-row :gutter="20">
          <el-col :xs="12" :sm="6">
            <div class="stat-card">
              <div class="stat-value">{{ stats.total_invites || 0 }}</div>
              <div class="stat-label">总邀请人数</div>
            </div>
          </el-col>
          <el-col :xs="12" :sm="6">
            <div class="stat-card">
              <div class="stat-value">{{ stats.registered_invites || 0 }}</div>
              <div class="stat-label">已注册人数</div>
            </div>
          </el-col>
          <el-col :xs="12" :sm="6">
            <div class="stat-card">
              <div class="stat-value">{{ stats.purchased_invites || 0 }}</div>
              <div class="stat-label">已购买人数</div>
            </div>
          </el-col>
          <el-col :xs="12" :sm="6">
            <div class="stat-card highlight">
              <div class="stat-value">¥{{ (stats.total_reward || 0).toFixed(2) }}</div>
              <div class="stat-label">累计奖励</div>
            </div>
          </el-col>
        </el-row>
        <!-- 显示可获得的奖励信息 -->
        <el-alert
          v-if="inviteRewardSettings.inviter_reward > 0 || inviteRewardSettings.invitee_reward > 0"
          title="邀请奖励说明"
          type="info"
          :closable="false"
          style="margin-top: 20px;"
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
      </div>

      <!-- 生成邀请码 -->
      <div class="generate-section">
        <el-button type="primary" @click="showGenerateDialog = true" :icon="Plus">
          生成新邀请码
        </el-button>
      </div>

      <!-- 我的邀请码列表 -->
      <div class="invite-codes-section">
        <h3>我的邀请码</h3>
        
        <!-- 桌面端表格 -->
        <div class="desktop-only">
          <el-table 
            :data="inviteCodes" 
            v-loading="loading"
            :empty-text="inviteCodes.length === 0 ? '暂无邀请码，点击上方按钮生成' : '暂无数据'"
            border
            stripe
          >
            <el-table-column prop="code" label="邀请码" min-width="100">
              <template #default="scope">
                <el-tag>{{ scope.row.code }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="invite_link" label="邀请链接" min-width="200" class-name="link-column">
              <template #default="scope">
                <div class="link-cell">
                  <el-input 
                    :value="scope.row.invite_link" 
                    readonly
                    size="small"
                  >
                    <template #append>
                      <el-button @click="copyLink(scope.row.invite_link)" :icon="DocumentCopy" />
                    </template>
                  </el-input>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="used_count" label="已使用" width="100" align="center">
              <template #default="scope">
                <span>{{ scope.row.used_count || 0 }} / {{ getMaxUses(scope.row.max_uses) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="expires_at" label="过期时间" width="180" class-name="expires-column">
              <template #default="scope">
                <span v-if="scope.row.expires_at && scope.row.expires_at !== 'null'">{{ formatDate(scope.row.expires_at) }}</span>
                <span v-else style="color: #909399;">永不过期</span>
              </template>
            </el-table-column>
            <el-table-column prop="is_valid" label="状态" width="100" align="center">
              <template #default="scope">
                <el-tag :type="getIsValid(scope.row) ? 'success' : 'danger'">
                  {{ getIsValid(scope.row) ? '有效' : '无效' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="150" align="center" class-name="action-column">
              <template #default="scope">
                <el-button 
                  type="primary" 
                  link 
                  size="small"
                  @click="copyLink(scope.row.invite_link)"
                  :icon="DocumentCopy"
                >
                  复制链接
                </el-button>
                <el-button 
                  type="danger" 
                  link 
                  size="small"
                  @click="deleteCode(scope.row)"
                  :icon="Delete"
                >
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
        
        <!-- 移动端卡片列表 -->
        <div class="mobile-only">
          <div v-loading="loading" class="mobile-invite-list">
            <div 
              v-for="code in inviteCodes" 
              :key="code.id"
              class="mobile-invite-card"
            >
              <div class="invite-card-header">
                <el-tag :type="getIsValid(code) ? 'success' : 'danger'" size="small">
                  {{ getIsValid(code) ? '有效' : '无效' }}
                </el-tag>
                <span class="invite-code">{{ code.code }}</span>
              </div>
              <div class="invite-card-body">
                <div class="invite-card-row">
                  <span class="invite-label">邀请链接：</span>
                  <div class="invite-link-wrapper">
                    <el-input 
                      :value="code.invite_link" 
                      readonly
                      size="small"
                      class="invite-link-input"
                    >
                      <template #append>
                        <el-button @click="copyLink(code.invite_link)" size="small" :icon="DocumentCopy" />
                      </template>
                    </el-input>
                  </div>
                </div>
                <div class="invite-card-row">
                  <span class="invite-label">已使用：</span>
                  <span class="invite-value">{{ code.used_count || 0 }} / {{ getMaxUses(code.max_uses) }}</span>
                </div>
                <div class="invite-card-row" v-if="code.expires_at && code.expires_at !== 'null'">
                  <span class="invite-label">过期时间：</span>
                  <span class="invite-value">{{ formatDate(code.expires_at) }}</span>
                </div>
                <div class="invite-card-row" v-else>
                  <span class="invite-label">过期时间：</span>
                  <span class="invite-value" style="color: #909399;">永不过期</span>
                </div>
              </div>
              <div class="invite-card-footer">
                <el-button 
                  type="primary" 
                  size="small"
                  @click="copyLink(code.invite_link)"
                  :icon="DocumentCopy"
                  class="mobile-action-btn"
                >
                  复制链接
                </el-button>
                <el-button 
                  type="danger" 
                  size="small"
                  @click="deleteCode(code)"
                  :icon="Delete"
                  class="mobile-action-btn"
                >
                  删除
                </el-button>
              </div>
            </div>
            <el-empty v-if="inviteCodes.length === 0 && !loading" description="暂无邀请码，点击上方按钮生成" />
          </div>
        </div>
      </div>

      <!-- 最近邀请记录 -->
      <div class="recent-invites-section" v-if="stats.recent_invites && stats.recent_invites.length > 0">
        <h3>最近邀请记录</h3>
        
        <!-- 桌面端表格 -->
        <div class="desktop-only">
          <el-table :data="stats.recent_invites" size="small">
            <el-table-column prop="invitee_username" label="被邀请人" width="120" />
            <el-table-column prop="invitee_email" label="邮箱" min-width="180" />
            <el-table-column prop="created_at" label="注册时间" width="180">
              <template #default="scope">
                {{ formatDate(scope.row.created_at) }}
              </template>
            </el-table-column>
            <el-table-column prop="has_purchased" label="已购买" width="100" align="center">
            <template #default="scope">
              <el-tag :type="scope.row.has_purchased ? 'success' : 'info'">
                {{ scope.row.has_purchased ? '是' : '否' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="total_consumption" label="累计消费" width="120" align="right">
            <template #default="scope">
              ¥{{ scope.row.total_consumption.toFixed(2) }}
            </template>
          </el-table-column>
          <el-table-column prop="reward_given" label="奖励状态" width="100" align="center">
            <template #default="scope">
              <el-tag :type="scope.row.reward_given ? 'success' : 'warning'">
                {{ scope.row.reward_given ? '已发放' : '未发放' }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
        </div>
        
        <!-- 移动端卡片列表 -->
        <div class="mobile-only">
          <div class="mobile-recent-list">
            <div 
              v-for="(invite, index) in stats.recent_invites" 
              :key="index"
              class="mobile-recent-card"
            >
              <div class="recent-card-header">
                <el-tag :type="invite.has_purchased ? 'success' : 'info'" size="small">
                  {{ invite.has_purchased ? '已购买' : '未购买' }}
                </el-tag>
                <span class="recent-time">{{ formatDate(invite.created_at) }}</span>
              </div>
              <div class="recent-card-body">
                <div class="recent-card-row">
                  <span class="recent-label">被邀请人：</span>
                  <span class="recent-value">{{ invite.invitee_username || '-' }}</span>
                </div>
                <div class="recent-card-row">
                  <span class="recent-label">邮箱：</span>
                  <span class="recent-value">{{ invite.invitee_email || '-' }}</span>
                </div>
                <div class="recent-card-row" v-if="invite.total_consumption !== undefined">
                  <span class="recent-label">累计消费：</span>
                  <span class="recent-value">¥{{ invite.total_consumption.toFixed(2) }}</span>
                </div>
                <div class="recent-card-row" v-if="invite.reward_given !== undefined">
                  <span class="recent-label">奖励状态：</span>
                  <el-tag :type="invite.reward_given ? 'success' : 'warning'" size="small">
                    {{ invite.reward_given ? '已发放' : '未发放' }}
                  </el-tag>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 生成邀请码对话框 -->
    <el-dialog
      v-model="showGenerateDialog"
      title="生成邀请码"
      :width="isMobile ? '95%' : '500px'"
      :close-on-click-modal="!isMobile"
    >
      <el-form :model="generateForm" :label-width="isMobile ? '0' : '120px'">
        <el-form-item label="最大使用次数">
          <el-input-number 
            v-model="generateForm.max_uses" 
            :min="1" 
            :max="1000"
            placeholder="留空表示无限制"
          />
          <div class="form-tip">邀请码最多可被使用多少次（留空表示无限制）</div>
        </el-form-item>
        <el-form-item label="有效期（天）">
          <el-input-number 
            v-model="generateForm.expires_days" 
            :min="1" 
            :max="365"
            placeholder="留空表示永不过期"
          />
          <div class="form-tip">邀请码有效期，留空表示永不过期</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer-buttons">
          <el-button @click="showGenerateDialog = false" class="mobile-action-btn">取消</el-button>
          <el-button type="primary" @click="generateCode" :loading="generating" class="mobile-action-btn">生成</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, DocumentCopy, Delete } from '@element-plus/icons-vue'
import { inviteAPI } from '@/utils/api'

const loading = ref(false)
const generating = ref(false)
const showGenerateDialog = ref(false)
const inviteCodes = ref([])
const isMobile = ref(window.innerWidth <= 768)

const handleResize = () => {
  isMobile.value = window.innerWidth <= 768
}

onMounted(async () => {
  window.addEventListener('resize', handleResize)
  handleResize()
  await loadInviteRewardSettings()
  await loadInviteCodes()
  await loadStats()
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

// 从系统配置获取奖励金额（只读显示）
const inviteRewardSettings = ref({
  inviter_reward: 0,
  invitee_reward: 0
})

// 加载邀请奖励配置
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
      console.warn('获取邀请奖励配置失败:', error)
    }
  }
}

const loadInviteCodes = async () => {
  loading.value = true
  try {
    const response = await inviteAPI.getMyInviteCodes()
    if (process.env.NODE_ENV === 'development') {
      console.log('邀请码列表完整响应:', response)
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
    }
    
    // 处理多种可能的响应格式
    if (response && response.data) {
      const responseData = response.data
      
      // 标准格式：{ success: true, data: { invite_codes: [...] } }
      if (responseData.success !== false && responseData.data) {
        if (responseData.data.invite_codes && Array.isArray(responseData.data.invite_codes)) {
          inviteCodes.value = responseData.data.invite_codes
          if (process.env.NODE_ENV === 'development') {
            console.log('✅ 成功加载邀请码（标准格式）:', inviteCodes.value.length, '个')
          }
        } else if (Array.isArray(responseData.data)) {
          inviteCodes.value = responseData.data
          if (process.env.NODE_ENV === 'development') {
            console.log('✅ 成功加载邀请码（data是数组）:', inviteCodes.value.length, '个')
          }
        } else {
          if (process.env.NODE_ENV === 'development') {
            console.warn('⚠️ data存在但不是预期格式:', responseData.data)
          }
          inviteCodes.value = []
        }
      } 
      else if (responseData.invite_codes && Array.isArray(responseData.invite_codes)) {
        inviteCodes.value = responseData.invite_codes
        if (process.env.NODE_ENV === 'development') {
          console.log('✅ 成功加载邀请码（直接格式）:', inviteCodes.value.length, '个')
        }
      }
      else if (Array.isArray(responseData.data)) {
        inviteCodes.value = responseData.data
        if (process.env.NODE_ENV === 'development') {
          console.log('✅ 成功加载邀请码（data数组格式）:', inviteCodes.value.length, '个')
        }
      }
      // 如果 success 为 false，显示错误信息
      else if (responseData.success === false) {
        const errorMsg = responseData.message || '获取邀请码列表失败'
        if (process.env.NODE_ENV === 'development') {
          console.error('❌ API返回失败:', errorMsg)
        }
        ElMessage.error(errorMsg)
        inviteCodes.value = []
      }
      else {
        if (process.env.NODE_ENV === 'development') {
          console.warn('⚠️ 未识别的响应格式:', {
            responseData,
            hasData: !!responseData.data,
            dataType: typeof responseData.data,
            dataKeys: responseData.data ? Object.keys(responseData.data) : []
          })
        }
        inviteCodes.value = []
      }
    } else {
      if (process.env.NODE_ENV === 'development') {
        console.warn('⚠️ 响应数据为空或格式异常:', {
          hasResponse: !!response,
          hasData: !!response?.data
        })
      }
      inviteCodes.value = []
    }
  } catch (error) {
    if (process.env.NODE_ENV === 'development') {
      console.error('获取邀请码列表错误:', error)
      console.error('错误详情:', {
        message: error.message,
        response: error.response,
        responseData: error.response?.data,
        responseStatus: error.response?.status
      })
    }
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
    if (process.env.NODE_ENV === 'development') {
      console.log('邀请统计响应:', response)
    }
    
    // 处理多种可能的响应格式
    if (response && response.data) {
      const responseData = response.data
      // 标准格式：{ success: true, data: { ... } }
      if (responseData.data) {
        stats.value = responseData.data
      }
      // 直接包含统计数据
      else if (responseData.total_invites !== undefined) {
        stats.value = responseData
      }
    }
  } catch (error) {
    console.error('获取邀请统计错误:', error)
    const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '未知错误'
    ElMessage.error('获取邀请统计失败: ' + errorMsg)
  }
}

const generateCode = async () => {
  generating.value = true
  try {
    // 准备请求数据（确保所有数值都是数字类型）
    const requestData = {
      max_uses: Number(generateForm.max_uses) || 0,
      reward_type: 'balance',
      inviter_reward: Number(inviteRewardSettings.value.inviter_reward) || 0,
      invitee_reward: Number(inviteRewardSettings.value.invitee_reward) || 0,
      min_order_amount: 0,
      new_user_only: true
    }
    
    // 如果有有效期天数，转换为 expires_at
    if (generateForm.expires_days && generateForm.expires_days > 0) {
      const expiresDate = new Date()
      expiresDate.setDate(expiresDate.getDate() + generateForm.expires_days)
      requestData.expires_at = expiresDate.toISOString()
    }
    
    const response = await inviteAPI.generateInviteCode(requestData)
    if (process.env.NODE_ENV === 'development') {
      console.log('生成邀请码响应:', response)
    }
    
    // 处理多种可能的响应格式
    const success = response?.data?.success !== false && 
                   (response?.data?.data?.code || response?.data?.code)
    
    if (success) {
      ElMessage.success('邀请码生成成功')
      showGenerateDialog.value = false
      // 重置表单
      Object.assign(generateForm, {
        max_uses: 10,
        expires_days: 30
      })
      // 重新加载邀请码列表和统计（确保数据刷新）
      await Promise.all([
        loadInviteCodes(),
        loadStats()
      ])
      if (process.env.NODE_ENV === 'development') {
        console.log('✅ 邀请码列表已刷新，当前数量:', inviteCodes.value.length)
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

const copyLink = (link) => {
  navigator.clipboard.writeText(link).then(() => {
    ElMessage.success('邀请链接已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败，请手动复制')
  })
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

onMounted(async () => {
  await loadInviteRewardSettings()
  await loadInviteCodes()
  await loadStats()
})
</script>

<style scoped lang="scss">
.invites-container {
  padding: 20px;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stats-section {
  margin-bottom: 30px;
  
  .stat-card {
    background: #f5f7fa;
    border-radius: 8px;
    padding: 20px;
    text-align: center;
    transition: all 0.3s;
    
    &:hover {
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    }
    
    &.highlight {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      
      .stat-label {
        color: rgba(255, 255, 255, 0.9);
      }
    }
    
    .stat-value {
      font-size: 28px;
      font-weight: bold;
      color: #303133;
      margin-bottom: 8px;
    }
    
    .stat-label {
      font-size: 14px;
      color: #909399;
    }
  }
}

.generate-section {
  margin-bottom: 30px;
}

.invite-codes-section,
.recent-invites-section {
  margin-top: 30px;
  
  :is(h3) {
    margin-bottom: 20px;
    font-size: 18px;
    font-weight: 600;
    color: #303133;
  }
}

.link-cell {
  .el-input {
    width: 100%;
  }
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.5;
}

/* 移除所有输入框的圆角和阴影效果，设置为简单长方形，只保留外部边框 */
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

  /* 表格在手机端优化 */
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

    /* 隐藏部分列在手机端 */
    .expires-column,
    .action-column {
      display: none;
    }

    /* 邀请链接列在手机端优化显示 */
    .link-column {
      min-width: 150px;
    }
  }
  
  /* 统计卡片优化 */
  .stats-section {
    margin-bottom: 15px;
    
    .el-row {
      margin: 0 -5px;
    }
    
    .el-col {
      padding: 0 5px;
      margin-bottom: 10px;
    }
    
    .stat-card {
      padding: 16px;
      text-align: center;
      background: #f8f9fa;
      border-radius: 8px;
      transition: all 0.3s ease;
      
      &:hover {
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
      }
      
      &.highlight {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: #ffffff;
        
        .stat-value,
        .stat-label {
          color: #ffffff;
        }
      }
      
      .stat-value {
        font-size: 24px;
        font-weight: 700;
        color: #303133;
        margin-bottom: 8px;
        word-break: break-all;
      }
      
      .stat-label {
        font-size: 13px;
        color: #909399;
        line-height: 1.4;
      }
    }
  }
  
  /* 生成邀请码按钮优化 */
  .generate-section {
    margin-bottom: 15px;
    
    .el-button {
      width: 100%;
      padding: 12px;
      min-height: 44px;
      font-size: 16px;
      font-weight: 500;
    }
  }
  
  /* 邀请码列表标题优化 */
  .invite-codes-section,
  .recent-invites-section {
    margin-bottom: 20px;
    
    :is(h3) {
      font-size: 16px;
      margin-bottom: 12px;
      font-weight: 600;
      color: #303133;
    }
  }

  /* 表格横向滚动 */
  :deep(.el-table__body-wrapper) {
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }

  /* 邀请链接列在手机端优化 */
  .link-cell {
    :deep(.el-input) {
      font-size: 11px;
    }
  }

  /* 生成邀请码对话框在手机端优化 */
  :deep(.el-dialog) {
    width: 95% !important;
    margin: 1vh auto !important;
    max-height: 98vh;
    border-radius: 12px;
    display: flex;
    flex-direction: column;
  }

  :deep(.el-dialog__header) {
    padding: 16px 16px 12px;
    flex-shrink: 0;
    border-bottom: 1px solid #ebeef5;
    
    .el-dialog__title {
      font-size: 18px;
      font-weight: 600;
    }
    
    .el-dialog__headerbtn {
      top: 12px;
      right: 12px;
      width: 36px;
      height: 36px;
    }
  }

  :deep(.el-dialog__body) {
    padding: 16px !important;
    flex: 1;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
    max-height: calc(98vh - 140px);
  }

  :deep(.el-dialog__footer) {
    padding: 12px 16px 16px;
    flex-shrink: 0;
    border-top: 1px solid #ebeef5;
    display: flex;
    flex-direction: column;
    gap: 10px;
    
    .dialog-footer-buttons {
      display: flex;
      flex-direction: column;
      gap: 10px;
      width: 100%;
      
      .mobile-action-btn {
        width: 100%;
        min-height: 48px;
        font-size: 16px;
        font-weight: 500;
        margin: 0 !important;
        border-radius: 8px;
      }
    }
  }

  :deep(.el-form-item) {
    margin-bottom: 20px;
    
    .el-form-item__label {
      width: 100% !important;
      text-align: left !important;
      margin-bottom: 8px !important;
      padding: 0 !important;
      font-weight: 500;
      font-size: 14px;
      color: #303133;
    }
    
    .el-form-item__content {
      margin-left: 0 !important;
      width: 100%;
    }
  }

  :deep(.el-input-number),
  :deep(.el-input) {
    width: 100% !important;
    
    .el-input__inner {
      font-size: 16px !important;
      min-height: 44px;
      padding: 0 12px;
    }
  }
  
  .form-tip {
    font-size: 12px;
    color: #909399;
    margin-top: 6px;
    line-height: 1.6;
  }
}

@media (max-width: 480px) {
  .invites-container {
    padding: 5px;
  }

  .stats-section {
    .stat-card {
      padding: 12px;
      
      .stat-value {
        font-size: 18px;
      }
      
      .stat-label {
        font-size: 11px;
      }
    }
  }

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

  /* 在超小屏幕上进一步优化表格 */
  :deep(.el-table__body-wrapper) {
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }
}
</style>

