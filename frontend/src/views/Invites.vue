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
              <span>{{ scope.row.used_count }} / {{ scope.row.max_uses || '∞' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="expires_at" label="过期时间" width="180" class-name="expires-column">
            <template #default="scope">
              <span v-if="scope.row.expires_at">{{ formatDate(scope.row.expires_at) }}</span>
              <span v-else style="color: #909399;">永不过期</span>
            </template>
          </el-table-column>
          <el-table-column prop="is_valid" label="状态" width="100" align="center">
            <template #default="scope">
              <el-tag :type="scope.row.is_valid ? 'success' : 'danger'">
                {{ scope.row.is_valid ? '有效' : '无效' }}
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

      <!-- 最近邀请记录 -->
      <div class="recent-invites-section" v-if="stats.recent_invites && stats.recent_invites.length > 0">
        <h3>最近邀请记录</h3>
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
    </el-card>

    <!-- 生成邀请码对话框 -->
    <el-dialog
      v-model="showGenerateDialog"
      title="生成邀请码"
      width="500px"
    >
      <el-form :model="generateForm" label-width="120px">
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
        <el-button @click="showGenerateDialog = false">取消</el-button>
        <el-button type="primary" @click="generateCode" :loading="generating">生成</el-button>
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
        inviter_reward: response.data.data.inviter_reward || 0,
        invitee_reward: response.data.data.invitee_reward || 0
      }
    }
  } catch (error) {
    console.warn('获取邀请奖励配置失败:', error)
  }
}

const loadInviteCodes = async () => {
  loading.value = true
  try {
    const response = await inviteAPI.getMyInviteCodes()
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
    
    // 处理多种可能的响应格式
    if (response && response.data) {
      const responseData = response.data
      
      // 标准格式：{ success: true, data: { invite_codes: [...] } }
      if (responseData.success !== false && responseData.data) {
        if (responseData.data.invite_codes && Array.isArray(responseData.data.invite_codes)) {
          inviteCodes.value = responseData.data.invite_codes
          console.log('✅ 成功加载邀请码（标准格式）:', inviteCodes.value.length, '个')
        } else if (Array.isArray(responseData.data)) {
          inviteCodes.value = responseData.data
          console.log('✅ 成功加载邀请码（data是数组）:', inviteCodes.value.length, '个')
        } else {
          console.warn('⚠️ data存在但不是预期格式:', responseData.data)
          inviteCodes.value = []
        }
      } 
      // 直接包含 invite_codes（兼容格式）
      else if (responseData.invite_codes && Array.isArray(responseData.invite_codes)) {
        inviteCodes.value = responseData.invite_codes
        console.log('✅ 成功加载邀请码（直接格式）:', inviteCodes.value.length, '个')
      }
      // 如果 data 是数组（兼容格式）
      else if (Array.isArray(responseData.data)) {
        inviteCodes.value = responseData.data
        console.log('✅ 成功加载邀请码（data数组格式）:', inviteCodes.value.length, '个')
      }
      // 如果 success 为 false，显示错误信息
      else if (responseData.success === false) {
        const errorMsg = responseData.message || '获取邀请码列表失败'
        console.error('❌ API返回失败:', errorMsg)
        ElMessage.error(errorMsg)
        inviteCodes.value = []
      }
      else {
        console.warn('⚠️ 未识别的响应格式:', {
          responseData,
          hasData: !!responseData.data,
          dataType: typeof responseData.data,
          dataKeys: responseData.data ? Object.keys(responseData.data) : []
        })
        inviteCodes.value = []
      }
    } else {
      console.warn('⚠️ 响应数据为空或格式异常:', {
        hasResponse: !!response,
        hasData: !!response?.data
      })
      inviteCodes.value = []
    }
  } catch (error) {
    console.error('获取邀请码列表错误:', error)
    console.error('错误详情:', {
      message: error.message,
      response: error.response,
      responseData: error.response?.data,
      responseStatus: error.response?.status
    })
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
    console.log('邀请统计响应:', response)
    
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
    // 准备请求数据
    const requestData = {
      max_uses: generateForm.max_uses || 0,
      reward_type: 'balance',
      inviter_reward: inviteRewardSettings.value.inviter_reward || 0,
      invitee_reward: inviteRewardSettings.value.invitee_reward || 0,
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
    console.log('生成邀请码响应:', response)
    
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
      console.log('✅ 邀请码列表已刷新，当前数量:', inviteCodes.value.length)
    } else {
      const errorMsg = response?.data?.message || '生成邀请码失败'
      ElMessage.error(errorMsg)
    }
  } catch (error) {
    console.error('生成邀请码错误:', error)
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
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
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
  }
  
  /* 生成邀请码按钮优化 */
  .generate-section {
    margin-bottom: 15px;
    
    .el-button {
      width: 100%;
      padding: 12px;
    }
  }
  
  /* 邀请码列表标题优化 */
  .invite-codes-section {
    :is(h3) {
      font-size: 16px;
      margin-bottom: 12px;
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
    margin: 5vh auto !important;
  }

  :deep(.el-dialog__body) {
    padding: 15px;
  }

  :deep(.el-form-item__label) {
    width: 100% !important;
    text-align: left;
    margin-bottom: 5px;
  }

  :deep(.el-form-item__content) {
    margin-left: 0 !important;
    width: 100%;
  }

  :deep(.el-input-number),
  :deep(.el-input) {
    width: 100% !important;
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

