<template>
  <div class="list-container tickets-container">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>工单中心</span>
          <div class="header-actions">
            <el-select v-model="filters.status" placeholder="状态筛选" clearable size="small" style="width: 120px" @change="handleFilterChange">
              <el-option label="待处理" value="pending" />
              <el-option label="处理中" value="processing" />
              <el-option label="已解决" value="resolved" />
              <el-option label="已关闭" value="closed" />
            </el-select>
            <el-select v-model="filters.type" placeholder="类型筛选" clearable size="small" style="width: 120px" @change="handleFilterChange">
              <el-option label="技术问题" value="technical" />
              <el-option label="账单问题" value="billing" />
              <el-option label="账户问题" value="account" />
              <el-option label="其他" value="other" />
            </el-select>
            <el-button size="small" @click="loadTickets">刷新</el-button>
            <el-button type="primary" @click="showCreateDialog = true">
              <el-icon><Plus /></el-icon>
              创建工单
            </el-button>
          </div>
        </div>
      </template>
      <div class="mobile-only" style="margin-bottom: 12px;">
        <el-button type="primary" @click="showCreateDialog = true" style="width: 100%;">
          <el-icon><Plus /></el-icon>
          创建工单
        </el-button>
      </div>
      <div class="table-wrapper">
        <el-table 
          ref="ticketTableRef"
          :data="tickets" 
          v-loading="loading" 
          style="width: 100%"
          border
          stripe
          @header-dragend="handleTicketColumnResize"
        >
          <el-table-column prop="ticket_no" label="工单编号" :width="columnWidths.ticket_no" resizable />
          <el-table-column prop="title" label="标题" :min-width="columnWidths.title" resizable>
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
          <el-table-column prop="type" label="类型" :width="columnWidths.type" resizable>
            <template #default="{ row }">
              <el-tag :type="getTypeTagType(row.type)">{{ getTypeText(row.type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" :width="columnWidths.status" resizable>
            <template #default="{ row }">
              <el-tag :type="getStatusTagType(row.status)">{{ getStatusText(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="priority" label="优先级" :width="columnWidths.priority" resizable>
            <template #default="{ row }">
              <el-tag :type="getPriorityTagType(row.priority)">{{ getPriorityText(row.priority) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" :width="columnWidths.created_at" resizable />
          <el-table-column label="操作" :width="columnWidths.actions" resizable>
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
      </div>
      <div class="mobile-card-list" v-if="tickets.length > 0 || !loading">
        <div
          v-for="ticket in tickets"
          :key="ticket.id"
          class="mobile-card"
        >
          <div class="card-row">
            <span class="label">状态</span>
            <span class="value">
              <el-tag :type="getStatusTagType(ticket.status)" size="small">
                {{ getStatusText(ticket.status) }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">标题</span>
            <span class="value">
              <div style="display: flex; align-items: center; gap: 8px;">
                <span>{{ ticket.title }}</span>
                <el-badge
                  v-if="ticket.has_unread && ticket.unread_replies > 0"
                  :value="ticket.unread_replies"
                  :max="99"
                  type="danger"
                />
              </div>
            </span>
          </div>
          <div class="card-row">
            <span class="label">工单编号</span>
            <span class="value">{{ ticket.ticket_no }}</span>
          </div>
          <div class="card-row">
            <span class="label">类型</span>
            <span class="value">
              <el-tag :type="getTypeTagType(ticket.type)" size="small">
                {{ getTypeText(ticket.type) }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">优先级</span>
            <span class="value">
              <el-tag :type="getPriorityTagType(ticket.priority)" size="small">
                {{ getPriorityText(ticket.priority) }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">创建时间</span>
            <span class="value">{{ ticket.created_at }}</span>
          </div>
          <div class="card-actions">
            <el-button
              type="primary"
              size="small"
              @click="viewTicket(ticket.id)"
            >
              查看详情
              <el-badge
                v-if="ticket.has_unread && ticket.unread_replies > 0"
                :value="ticket.unread_replies"
                :max="99"
                type="danger"
                style="margin-left: 4px;"
              />
            </el-button>
          </div>
        </div>
        <el-empty v-if="tickets.length === 0 && !loading" description="暂无工单" />
      </div>
    <div class="pagination">
      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.size"
        :total="pagination.total"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="loadTickets"
        @current-change="loadTickets"
      />
    </div>
    </el-card>
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
import { ElMessage } from '@/utils/elementPlusServices'
import { Plus, UserFilled } from '@element-plus/icons-vue'
import { ticketAPI } from '@/utils/api'
import { useMobile } from '@/composables/useMobile'

const TICKETS_TABLE_STORAGE_KEY = 'user_tickets_table_settings'
const ticketTableRef = ref(null)
const columnWidths = reactive({
  ticket_no: 180,
  title: 200,
  type: 100,
  status: 100,
  priority: 100,
  created_at: 180,
  actions: 150
})
const loadTicketTableSettings = () => {
  try {
    const saved = localStorage.getItem(TICKETS_TABLE_STORAGE_KEY)
    if (saved) {
      const s = JSON.parse(saved)
      if (s.columnWidths) Object.assign(columnWidths, s.columnWidths)
    }
  } catch (e) {
    console.warn('加载工单表设置失败:', e)
  }
}
const saveTicketTableSettings = () => {
  try {
    localStorage.setItem(TICKETS_TABLE_STORAGE_KEY, JSON.stringify({ columnWidths: { ...columnWidths } }))
  } catch (e) {
    console.warn('保存工单表设置失败:', e)
  }
}
const TICKET_COLUMN_KEYS = ['ticket_no', 'title', 'type', 'status', 'priority', 'created_at', 'actions']
let ticketResizeTimer = null
const handleTicketColumnResize = () => {
  if (ticketResizeTimer) clearTimeout(ticketResizeTimer)
  ticketResizeTimer = setTimeout(() => {
    if (ticketTableRef.value && ticketTableRef.value.$el) {
      const cells = ticketTableRef.value.$el.querySelectorAll('.el-table__header-wrapper thead th')
      cells.forEach((cell, index) => {
        if (TICKET_COLUMN_KEYS[index] && cell.offsetWidth > 0) columnWidths[TICKET_COLUMN_KEYS[index]] = cell.offsetWidth
      })
      saveTicketTableSettings()
    }
  }, 300)
}

const isMobile = useMobile()
onMounted(() => {
  loadTicketTableSettings()
  loadTickets()
})
onUnmounted(() => {
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
  size: 10,
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
    if (response.data && response.data.success) {
      const ticketData = response.data.data?.ticket || response.data.data
      if (!ticketData || !ticketData.id) {
        ElMessage.error('工单数据格式错误，请刷新后重试')
        return
      }
      currentTicket.value = ticketData
      showDetailDialog.value = true
      markTicketReadLocally(ticketId)
      window.dispatchEvent(new CustomEvent('ticket-viewed'))
      await loadTickets()
    } else {
      ElMessage.error(response.data?.message || '加载工单详情失败')
    }
  } catch (error) {
    console.error('[用户前端] 加载工单详情异常:', error)
    console.error('[用户前端] 错误详情:', error.response?.data)
    ElMessage.error(error.response?.data?.detail || error.response?.data?.message || '加载工单详情失败')
  }
}
const markTicketReadLocally = (ticketId) => {
  tickets.value = tickets.value.map(ticket => {
    if (ticket.id !== ticketId) return ticket
    return {
      ...ticket,
      has_unread: false,
      unread_replies: 0
    }
  })
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
  try {
    const response = await ticketAPI.addReply(currentTicket.value.id, { content: replyContent.value })
    if (response.data && response.data.success) {
      ElMessage.success('回复成功')
      replyContent.value = ''
      await viewTicket(currentTicket.value.id)
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
  return map[priority] || 'info'
}
const handleFilterChange = () => {
  pagination.page = 1 // 重置到第一页
  loadTickets()
}
</script>
<style scoped lang="scss">
.mobile-only {
  display: none !important;
  @media (max-width: 768px) {
    display: block !important;
  }
}
.tickets-container {
  padding: 0;
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
:deep(.el-input__wrapper > *) {
  background-color: transparent !important;
  background: transparent !important;
}
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
:deep(.el-select .el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
  pointer-events: auto !important;
}
.create-ticket-dialog {
  :deep(.el-dialog) {
    .el-dialog__body {
      padding: 20px;
    }
  }
}
@media (max-width: 768px) {
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
  :deep(.el-table) {
    display: none;
  }
  :deep(.el-dialog:not(.create-ticket-dialog .el-dialog)) {
    width: 95% !important;
    margin: 2vh auto !important;
    max-height: 96vh;
    overflow-y: auto;
    border-radius: 8px;
  }
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
</style>
