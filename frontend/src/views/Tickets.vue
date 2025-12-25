<template>
  <div class="tickets-container">
    <div class="page-header">
      <h1>工单中心</h1>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon><Plus /></el-icon>
        创建工单
      </el-button>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <el-select v-model="filters.status" placeholder="状态筛选" clearable style="width: 150px" @change="handleFilterChange">
        <el-option label="待处理" value="pending" />
        <el-option label="处理中" value="processing" />
        <el-option label="已解决" value="resolved" />
        <el-option label="已关闭" value="closed" />
      </el-select>
      <el-select v-model="filters.type" placeholder="类型筛选" clearable style="width: 150px" @change="handleFilterChange">
        <el-option label="技术问题" value="technical" />
        <el-option label="账单问题" value="billing" />
        <el-option label="账户问题" value="account" />
        <el-option label="其他" value="other" />
      </el-select>
      <el-button @click="loadTickets">刷新</el-button>
    </div>

    <!-- 工单列表 -->
    <el-table :data="tickets" v-loading="loading" style="width: 100%">
      <el-table-column prop="ticket_no" label="工单编号" width="180" />
      <el-table-column prop="title" label="标题">
        <template #default="{ row }">
          <div style="display: flex; align-items: center; gap: 8px;">
            <span>{{ row.title }}</span>
            <el-badge 
              v-if="row.has_unread && row.unread_replies > 0" 
              :value="row.unread_replies" 
              :max="99"
              type="danger"
            />
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="type" label="类型" width="100">
        <template #default="{ row }">
          <el-tag :type="getTypeTagType(row.type)">{{ getTypeText(row.type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="getStatusTagType(row.status)">{{ getStatusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="priority" label="优先级" width="100">
        <template #default="{ row }">
          <el-tag :type="getPriorityTagType(row.priority)">{{ getPriorityText(row.priority) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="180" />
      <el-table-column label="操作" width="150">
        <template #default="{ row }">
          <el-button size="small" @click="viewTicket(row.id)">
            查看
            <el-badge 
              v-if="row.has_unread && row.unread_replies > 0" 
              :value="row.unread_replies" 
              :max="99"
              type="danger"
              style="margin-left: 4px;"
            />
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 手机端卡片式列表 -->
    <div class="mobile-ticket-list" v-loading="loading">
      <div 
        v-for="ticket in tickets" 
        :key="ticket.id"
        class="mobile-ticket-card"
      >
        <div class="ticket-card-header">
          <div style="display: flex; align-items: center; gap: 8px; flex: 1;">
            <h4>{{ ticket.title }}</h4>
            <el-badge 
              v-if="ticket.has_unread && ticket.unread_replies > 0" 
              :value="ticket.unread_replies" 
              :max="99"
              type="danger"
            />
          </div>
          <el-tag :type="getStatusTagType(ticket.status)" size="small">
            {{ getStatusText(ticket.status) }}
          </el-tag>
        </div>
        <div class="ticket-card-body">
          <div class="ticket-card-row">
            <span class="label">工单编号：</span>
            <span class="value">{{ ticket.ticket_no }}</span>
          </div>
          <div class="ticket-card-row">
            <span class="label">类型：</span>
            <span class="value">
              <el-tag :type="getTypeTagType(ticket.type)" size="small">
                {{ getTypeText(ticket.type) }}
              </el-tag>
            </span>
          </div>
          <div class="ticket-card-row">
            <span class="label">优先级：</span>
            <span class="value">
              <el-tag :type="getPriorityTagType(ticket.priority)" size="small">
                {{ getPriorityText(ticket.priority) }}
              </el-tag>
            </span>
          </div>
          <div class="ticket-card-row">
            <span class="label">创建时间：</span>
            <span class="value">{{ ticket.created_at }}</span>
          </div>
        </div>
        <div class="ticket-card-actions">
          <el-button 
            type="primary" 
            size="small" 
            @click="viewTicket(ticket.id)"
            style="width: 100%"
          >
            查看详情
          </el-button>
        </div>
      </div>
      <div v-if="!loading && tickets.length === 0" class="empty-state">
        <p>暂无工单</p>
      </div>
    </div>

    <!-- 分页 -->
    <el-pagination
      v-model:current-page="pagination.page"
      v-model:page-size="pagination.size"
      :total="pagination.total"
      :page-sizes="[10, 20, 50, 100]"
      layout="total, sizes, prev, pager, next, jumper"
      @size-change="loadTickets"
      @current-change="loadTickets"
      style="margin-top: 20px; justify-content: center"
    />

    <!-- 创建工单对话框 -->
    <el-dialog 
      v-model="showCreateDialog" 
      title="创建工单" 
      :width="isMobile ? '95%' : '600px'"
      class="create-ticket-dialog"
    >
      <el-form 
        :model="ticketForm" 
        :rules="ticketRules" 
        ref="ticketFormRef" 
        :label-width="isMobile ? '0' : '100px'"
        :label-position="isMobile ? 'top' : 'right'"
        class="ticket-form"
      >
        <el-form-item :label="isMobile ? '' : '标题'" prop="title">
          <template #label v-if="isMobile">
            <span class="form-label">*标题</span>
          </template>
          <el-input 
            v-model="ticketForm.title" 
            placeholder="请输入工单标题"
            :size="isMobile ? 'large' : 'default'"
          />
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '类型'" prop="type">
          <template #label v-if="isMobile">
            <span class="form-label">*类型</span>
          </template>
          <el-select 
            v-model="ticketForm.type" 
            placeholder="请选择类型"
            :size="isMobile ? 'large' : 'default'"
            style="width: 100%"
          >
            <el-option label="技术问题" value="technical" />
            <el-option label="账单问题" value="billing" />
            <el-option label="账户问题" value="account" />
            <el-option label="其他" value="other" />
          </el-select>
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '优先级'" prop="priority">
          <template #label v-if="isMobile">
            <span class="form-label">优先级</span>
          </template>
          <el-select 
            v-model="ticketForm.priority" 
            placeholder="请选择优先级"
            :size="isMobile ? 'large' : 'default'"
            style="width: 100%"
          >
            <el-option label="低" value="low" />
            <el-option label="普通" value="normal" />
            <el-option label="高" value="high" />
            <el-option label="紧急" value="urgent" />
          </el-select>
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '内容'" prop="content">
          <template #label v-if="isMobile">
            <span class="form-label">*内容</span>
          </template>
          <el-input
            v-model="ticketForm.content"
            type="textarea"
            :rows="isMobile ? 5 : 6"
            placeholder="请详细描述您的问题"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer-buttons">
          <el-button 
            @click="showCreateDialog = false"
            :size="isMobile ? 'large' : 'default'"
            :style="isMobile ? 'width: 100%; margin-bottom: 10px;' : ''"
          >
            取消
          </el-button>
          <el-button 
            type="primary" 
            @click="createTicket" 
            :loading="creating"
            :size="isMobile ? 'large' : 'default'"
            :style="isMobile ? 'width: 100%;' : ''"
          >
            创建
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 工单详情对话框 -->
    <el-dialog v-model="showDetailDialog" title="工单详情" width="800px">
      <div v-if="currentTicket">
        <div class="ticket-detail-header">
          <h3>{{ currentTicket.title }}</h3>
          <div class="ticket-meta">
            <el-tag :type="getStatusTagType(currentTicket.status)">{{ getStatusText(currentTicket.status) }}</el-tag>
            <el-tag :type="getTypeTagType(currentTicket.type)">{{ getTypeText(currentTicket.type) }}</el-tag>
            <span>工单编号: {{ currentTicket.ticket_no }}</span>
          </div>
        </div>
        <div class="ticket-content">
          <p>{{ currentTicket.content }}</p>
        </div>
        <div class="ticket-replies">
          <h4>回复记录 ({{ currentTicket.replies?.length || 0 }})</h4>
          <div v-if="currentTicket.replies && currentTicket.replies.length > 0">
            <div 
              v-for="reply in currentTicket.replies" 
              :key="reply.id" 
              class="reply-item" 
              :class="{ 
                'admin-reply': reply.is_admin === 'true' || reply.is_admin_reply,
                'user-reply': reply.is_admin !== 'true' && !reply.is_admin_reply
              }"
            >
              <div class="reply-header">
                <div class="reply-author-info">
                  <el-icon v-if="reply.is_admin === 'true' || reply.is_admin_reply" class="admin-icon">
                    <UserFilled />
                  </el-icon>
                  <el-tag 
                    :type="reply.is_admin === 'true' || reply.is_admin_reply ? 'success' : 'info'" 
                    size="small"
                    effect="dark"
                  >
                    {{ reply.is_admin === 'true' || reply.is_admin_reply ? '管理员回复' : '我的回复' }}
                  </el-tag>
                </div>
                <span class="reply-time">{{ reply.created_at }}</span>
              </div>
              <div class="reply-content" :class="{ 'admin-content': reply.is_admin === 'true' || reply.is_admin_reply }">
                {{ reply.content }}
              </div>
            </div>
          </div>
          <div v-else class="empty-replies">
            <p>暂无回复</p>
          </div>
        </div>
        <div class="ticket-reply-form">
          <el-input
            v-model="replyContent"
            type="textarea"
            :rows="3"
            placeholder="输入回复内容"
          />
          <el-button type="primary" @click="addReply" style="margin-top: 10px">发送回复</el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, UserFilled } from '@element-plus/icons-vue'
import { ticketAPI } from '@/utils/api'

// 检测是否为移动端
const isMobile = ref(window.innerWidth <= 768)

const handleResize = () => {
  isMobile.value = window.innerWidth <= 768
}

onMounted(() => {
  window.addEventListener('resize', handleResize)
  loadTickets()
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})

const loading = ref(false)
const creating = ref(false)
const tickets = ref([])
const showCreateDialog = ref(false)
const showDetailDialog = ref(false)
const currentTicket = ref(null)
const replyContent = ref('')
const ticketFormRef = ref(null)

const filters = reactive({
  status: '',
  type: ''
})

const pagination = reactive({
  page: 1,
  size: 20,
  total: 0
})

const ticketForm = reactive({
  title: '',
  content: '',
  type: 'other',
  priority: 'normal'
})

const ticketRules = {
  title: [{ required: true, message: '请输入工单标题', trigger: 'blur' }],
  content: [{ required: true, message: '请输入工单内容', trigger: 'blur' }],
  type: [{ required: true, message: '请选择工单类型', trigger: 'change' }]
}

const loadTickets = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      size: pagination.size
    }
    if (filters.status) params.status = filters.status
    if (filters.type) params.type = filters.type
    
    const response = await ticketAPI.getUserTickets(params)
    if (response.data && response.data.success) {
      tickets.value = response.data.data?.tickets || []
      pagination.total = response.data.data?.total || 0
    } else {
      ElMessage.error(response.data?.message || '加载工单列表失败')
    }
  } catch (error) {
    const errorMsg = error.response?.data?.message || error.message || '加载工单列表失败'
    ElMessage.error(errorMsg)
  } finally {
    loading.value = false
  }
}

const createTicket = async () => {
  if (!ticketFormRef.value) return
  await ticketFormRef.value.validate(async (valid) => {
    if (valid) {
      creating.value = true
      try {
        const response = await ticketAPI.createTicket(ticketForm)
        if (response.data.success) {
          ElMessage.success('工单创建成功')
          showCreateDialog.value = false
          ticketForm.title = ''
          ticketForm.content = ''
          ticketForm.type = 'other'
          ticketForm.priority = 'normal'
          loadTickets()
        }
      } catch (error) {
        ElMessage.error('创建工单失败')
      } finally {
        creating.value = false
      }
    }
  })
}

const viewTicket = async (ticketId) => {
  try {
    const response = await ticketAPI.getTicket(ticketId)
    console.log('[用户前端] 获取工单详情响应:', response)
    if (response.data && response.data.success) {
      // 后端返回的数据结构是 { data: { ticket: {...} } }
      const ticketData = response.data.data?.ticket || response.data.data
      if (!ticketData || !ticketData.id) {
        console.error('[用户前端] 工单数据格式错误:', response.data)
        ElMessage.error('工单数据格式错误，请刷新后重试')
        return
      }
      currentTicket.value = ticketData
      console.log('[用户前端] 当前工单:', currentTicket.value, '工单ID:', currentTicket.value.id)
      showDetailDialog.value = true
      
      // 用户查看工单后，刷新工单列表和未读数量
      // 延迟一下，确保后端已记录已读状态
      setTimeout(async () => {
        await loadTickets()
        // 触发父组件刷新未读数量（通过 window 事件）
        window.dispatchEvent(new CustomEvent('ticket-viewed'))
      }, 500)
    } else {
      ElMessage.error(response.data?.message || '加载工单详情失败')
    }
  } catch (error) {
    console.error('[用户前端] 加载工单详情异常:', error)
    console.error('[用户前端] 错误详情:', error.response?.data)
    ElMessage.error(error.response?.data?.detail || error.response?.data?.message || '加载工单详情失败')
  }
}

const addReply = async () => {
  if (!replyContent.value.trim()) {
    ElMessage.warning('请输入回复内容')
    return
  }
  if (!currentTicket.value || !currentTicket.value.id) {
    console.error('[用户前端] 工单ID不存在:', currentTicket.value)
    ElMessage.error('工单信息不完整，请刷新后重试')
    return
  }
  console.log('[用户前端] 开始回复工单:', currentTicket.value.id, replyContent.value)
  try {
    const response = await ticketAPI.addReply(currentTicket.value.id, { content: replyContent.value })
    console.log('[用户前端] 回复响应:', response)
    if (response.data && response.data.success) {
      ElMessage.success('回复成功')
      replyContent.value = ''
      // 重新加载工单详情以显示新回复
      await viewTicket(currentTicket.value.id)
      // 刷新工单列表以更新未读状态
      await loadTickets()
    } else {
      console.error('[用户前端] 回复失败，响应数据:', response.data)
      ElMessage.error(response.data?.message || '回复失败')
    }
  } catch (error) {
    console.error('[用户前端] 回复异常:', error)
    console.error('[用户前端] 错误详情:', error.response?.data)
    ElMessage.error(error.response?.data?.detail || error.response?.data?.message || '回复失败')
  }
}

const getStatusText = (status) => {
  const map = {
    pending: '待处理',
    processing: '处理中',
    resolved: '已解决',
    closed: '已关闭',
    cancelled: '已取消'
  }
  return map[status] || status
}

const getStatusTagType = (status) => {
  const map = {
    pending: 'warning',
    processing: 'primary',
    resolved: 'success',
    closed: 'info',
    cancelled: 'danger'
  }
  // 确保返回有效的类型值，如果找不到则返回 'info' 作为默认值
  return map[status] || 'info'
}

const getTypeText = (type) => {
  const map = {
    technical: '技术问题',
    billing: '账单问题',
    account: '账户问题',
    other: '其他'
  }
  return map[type] || type
}

const getTypeTagType = (type) => {
  // 根据类型返回不同的标签类型
  const map = {
    technical: 'primary',
    billing: 'warning',
    account: 'danger',
    other: 'info'
  }
  return map[type] || 'info'
}

const getPriorityText = (priority) => {
  const map = {
    low: '低',
    normal: '普通',
    high: '高',
    urgent: '紧急'
  }
  return map[priority] || priority
}

const getPriorityTagType = (priority) => {
  const map = {
    low: 'info',
    normal: 'primary',  // 普通优先级使用 primary，不能是空字符串
    high: 'warning',
    urgent: 'danger'
  }
  // 确保返回有效的类型值，如果找不到则返回 'info' 作为默认值
  return map[priority] || 'info'
}

const handleFilterChange = () => {
  pagination.page = 1 // 重置到第一页
  loadTickets()
}
</script>

<style scoped lang="scss">
.tickets-container {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.filter-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
}

.ticket-detail-header {
  margin-bottom: 20px;
  :is(h3) {
    margin: 0 0 10px 0;
  }
}

.ticket-meta {
  display: flex;
  gap: 10px;
  align-items: center;
}

.ticket-content {
  margin: 20px 0;
  padding: 15px;
  background: #f5f5f5;
  border-radius: 4px;
}

.ticket-replies {
  margin: 20px 0;
  :is(h4) {
    margin-bottom: 15px;
  }
}

.reply-item {
  margin-bottom: 15px;
  padding: 15px;
  border-radius: 8px;
  transition: all 0.3s ease;
  
  &.user-reply {
    background: #f5f5f5;
    border-left: 3px solid #909399;
  }
  
  &.admin-reply {
    background: linear-gradient(135deg, #e8f4fd 0%, #d4edff 100%);
    border-left: 4px solid #409eff;
    box-shadow: 0 2px 8px rgba(64, 158, 255, 0.15);
    animation: highlightAdminReply 0.5s ease;
  }
}

@keyframes highlightAdminReply {
  0% {
    transform: scale(1);
    box-shadow: 0 2px 8px rgba(64, 158, 255, 0.15);
  }
  50% {
    transform: scale(1.02);
    box-shadow: 0 4px 16px rgba(64, 158, 255, 0.3);
  }
  100% {
    transform: scale(1);
    box-shadow: 0 2px 8px rgba(64, 158, 255, 0.15);
  }
}

.reply-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-size: 12px;
  
  .reply-author-info {
    display: flex;
    align-items: center;
    gap: 8px;
    
    .admin-icon {
      color: #409eff;
      font-size: 16px;
    }
  }
  
  .reply-time {
    color: #666;
    font-size: 12px;
  }
}

.reply-content {
  color: #333;
  line-height: 1.6;
  font-size: 14px;
  
  &.admin-content {
    color: #1a1a1a;
    font-weight: 500;
    font-size: 15px;
  }
}

.ticket-reply-form {
  margin-top: 20px;
}

/* 修复输入框嵌套问题 - 移除内部边框和嵌套效果 */
:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
  pointer-events: auto !important;
}

:deep(.el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background-color: transparent !important;
  pointer-events: auto !important;
}

:deep(.el-input__wrapper:hover) {
  border-color: #c0c4cc !important;
  box-shadow: none !important;
  background-color: #ffffff !important;
}

:deep(.el-input__wrapper.is-focus) {
  border-color: #1677ff !important;
  box-shadow: none !important;
  background-color: #ffffff !important;
}

:deep(.el-input__wrapper.is-focus:hover) {
  background-color: #ffffff !important;
}

/* 确保输入框内部所有子元素背景透明 */
:deep(.el-input__wrapper > *) {
  background-color: transparent !important;
  background: transparent !important;
}

/* 移除 textarea 的嵌套边框 */
:deep(.el-textarea__inner) {
  border-radius: 0 !important;
  border: 1px solid #dcdfe6 !important;
  box-shadow: none !important;
  background-color: #ffffff !important;
}

:deep(.el-textarea__inner:hover) {
  border-color: #c0c4cc !important;
}

:deep(.el-textarea__inner:focus) {
  border-color: #1677ff !important;
  box-shadow: none !important;
}

/* 修复 select 下拉框的嵌套问题 */
:deep(.el-select .el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
  pointer-events: auto !important;
}

/* 创建工单对话框样式 */
.create-ticket-dialog {
  :deep(.el-dialog) {
    .el-dialog__body {
      padding: 20px;
    }
  }
}

/* 手机端优化 */
@media (max-width: 768px) {
  /* 创建工单对话框全屏显示 */
  .create-ticket-dialog {
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
      flex-shrink: 0;
      padding: 16px !important;
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
      flex: 1;
      overflow-y: auto;
      padding: 16px !important;
      -webkit-overflow-scrolling: touch;
    }
    
    :deep(.el-dialog__footer) {
      flex-shrink: 0;
      padding: 12px 16px 16px 16px !important;
      border-top: 1px solid #e5e7eb;
    }
  }
  .tickets-container {
    padding: 10px;
  }
  
  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 16px;
    
    :is(h1) {
      font-size: 1.5rem;
      margin: 0;
    }
    
    .el-button {
      width: 100%;
      min-height: 44px;
      font-size: 16px;
    }
  }
  
  .filter-bar {
    flex-direction: column;
    gap: 10px;
    margin-bottom: 16px;
    
    .el-select {
      width: 100% !important;
    }
    
    .el-button {
      width: 100%;
      min-height: 44px;
      font-size: 16px;
    }
  }
  
  /* 表格在手机端隐藏，使用卡片式布局 */
  :deep(.el-table) {
    display: none;
  }
  
  /* 手机端卡片式列表 */
  .mobile-ticket-list {
    display: block;
  }
  
  /* 分页优化 */
  :deep(.el-pagination) {
    flex-wrap: wrap;
    justify-content: center;
    
    .el-pagination__sizes,
    .el-pagination__jump {
      display: none;
    }
    
    .el-pagination__total {
      display: none;
    }
    
    .btn-prev,
    .btn-next {
      padding: 8px 12px;
      min-width: 40px;
      min-height: 40px;
    }
    
    .number {
      min-width: 36px;
      height: 36px;
      line-height: 36px;
      font-size: 14px;
    }
  }
  
  /* 对话框优化（非创建工单对话框） */
  :deep(.el-dialog:not(.create-ticket-dialog .el-dialog)) {
    width: 95% !important;
    margin: 2vh auto !important;
    max-height: 96vh;
    overflow-y: auto;
    border-radius: 8px;
  }
  
  /* 表单优化 */
  .ticket-form {
    :deep(.el-form-item) {
      margin-bottom: 20px;
      
      .el-form-item__label {
        width: 100% !important;
        text-align: left;
        margin-bottom: 8px;
        padding: 0;
        font-size: 14px;
        font-weight: 500;
        color: #333;
        line-height: 1.5;
        display: block;
      }
      
      .el-form-item__content {
        width: 100%;
        margin-left: 0 !important;
        
        .el-input,
        .el-select {
          width: 100% !important;
        }
        
        .el-textarea {
          width: 100% !important;
        }
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
  
  /* 对话框底部按钮优化 */
  .dialog-footer-buttons {
    display: flex;
    flex-direction: column;
    gap: 10px;
    width: 100%;
    
    .el-button {
      width: 100%;
      margin: 0 !important;
      min-height: 44px;
      font-size: 16px;
      border-radius: 6px;
    }
  }
  
  .ticket-detail-header {
    :is(h3) {
      font-size: 1.25rem;
      margin-bottom: 12px;
    }
  }
  
  .ticket-meta {
    flex-wrap: wrap;
    gap: 8px;
    font-size: 0.875rem;
  }
  
  .ticket-content {
    padding: 12px;
    font-size: 0.875rem;
    line-height: 1.6;
  }
  
  .ticket-replies {
    :is(h4) {
      font-size: 1rem;
      margin-bottom: 12px;
    }
  }
  
  .reply-item {
    padding: 12px;
    margin-bottom: 12px;
  }
  
  .reply-header {
    font-size: 0.75rem;
    margin-bottom: 8px;
  }
  
  .reply-content {
    font-size: 0.875rem;
    line-height: 1.6;
  }
  
  .ticket-reply-form {
    margin-top: 16px;
    
    .el-button {
      width: 100%;
      margin-top: 12px;
      min-height: 44px;
      font-size: 16px;
    }
  }
}

/* 桌面端隐藏手机端列表 */
.mobile-ticket-list {
  display: none;
}

@media (min-width: 769px) {
  .mobile-ticket-list {
    display: none;
  }
}

/* 手机端卡片样式 */
.mobile-ticket-card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.ticket-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  
  :is(h4) {
    margin: 0;
    font-size: 1rem;
    font-weight: 600;
    color: #333;
    flex: 1;
    padding-right: 12px;
  }
}

.ticket-card-body {
  margin-bottom: 12px;
}

.ticket-card-row {
  display: flex;
  margin-bottom: 8px;
  font-size: 0.875rem;
  
  &:last-child {
    margin-bottom: 0;
  }
  
  .label {
    color: #666;
    min-width: 80px;
    font-weight: 500;
  }
  
  .value {
    color: #333;
    flex: 1;
    word-break: break-word;
  }
}

.ticket-card-actions {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #e5e7eb;
}

.empty-state {
  text-align: center;
  padding: 40px 20px;
  color: #999;
  
  :is(p) {
    margin: 0;
    font-size: 0.9rem;
  }
}
</style>


