<template>
  <div class="list-container tickets-container">
    <div class="breadcrumb">首页 / 工单中心</div>
    <div class="page-header">
      <div class="page-title">
        <h1>工单中心</h1>
      </div>
      <div class="actions">
        <el-button type="primary" @click="showCreateDialog = true">
          创建工单
        </el-button>
      </div>
    </div>
    <div class="tickets-workspace">
      <div class="card list-card">
        <div class="card-header">
          <h2 class="card-title">我的工单</h2>
          <el-button size="small" :loading="loading" @click="loadTickets">刷新</el-button>
        </div>
        <div class="card-body ticket-filter-body">
          <el-form :inline="true" :model="filters" class="ticket-filter-form list-filter-form">
            <el-form-item label="状态筛选">
              <el-select v-model="filters.status" placeholder="全部状态" clearable size="small" class="ticket-filter-select" @change="handleFilterChange">
                <el-option label="待处理" value="pending" />
                <el-option label="处理中" value="processing" />
                <el-option label="已解决" value="resolved" />
                <el-option label="已关闭" value="closed" />
              </el-select>
            </el-form-item>
            <el-form-item label="类型筛选">
              <el-select v-model="filters.type" placeholder="全部类型" clearable size="small" class="ticket-filter-select" @change="handleFilterChange">
                <el-option label="技术问题" value="technical" />
                <el-option label="账单问题" value="billing" />
                <el-option label="账户问题" value="account" />
                <el-option label="其他" value="other" />
              </el-select>
            </el-form-item>
            <el-form-item label="每页数量">
              <el-select v-model="pagination.size" placeholder="分页" size="small" class="ticket-page-size-select" @change="loadTickets">
                <el-option label="分页 10" :value="10" />
                <el-option label="分页 20" :value="20" />
                <el-option label="分页 50" :value="50" />
                <el-option label="分页 100" :value="100" />
              </el-select>
            </el-form-item>
          </el-form>
        </div>
        <ResponsiveDataView
          :data="tickets"
          :fields="mobileTicketFields"
          :loading="loading"
          title-field="title"
          empty-title="暂无工单"
        >
          <template #table>
            <div class="table-wrapper">
              <el-table 
                ref="ticketTableRef"
                :data="tickets" 
                v-loading="loading" 
                class="ticket-table"
                border
                stripe
                @header-dragend="handleTicketColumnResize"
              >
                <template #empty>
                  <EmptyState
                    title="暂无工单"
                    description="创建工单后可在这里跟进处理进度。"
                    :icon-size="48"
                  />
                </template>
                <el-table-column prop="ticket_no" label="工单编号" :width="columnWidths.ticket_no" resizable />
                <el-table-column prop="title" label="标题" :min-width="columnWidths.title" resizable>
                  <template #default="{ row }">
                    <div class="ticket-title-cell">
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
                        class="inline-badge"
                      />
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </template>
          <template #header="{ item }">
            <div class="ticket-mobile-header">
              <span>{{ item.title }}</span>
              <el-badge
                v-if="item.has_unread && item.unread_replies > 0"
                :value="item.unread_replies"
                :max="99"
                type="danger"
              />
            </div>
          </template>
          <template #actions="{ item }">
            <el-button type="primary" size="small" @click="viewTicket(item.id)">
              查看详情
            </el-button>
          </template>
        </ResponsiveDataView>
        <PaginationBar
          v-if="pagination.total > 0"
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.size"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          @size-change="loadTickets"
          @current-change="loadTickets"
        />
      </div>
    </div>
    <AppDialog 
      v-model="showCreateDialog" 
      title="创建工单" 
      width="600px"
      mobile-width="94%"
      :loading="creating"
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
            class="full-width-control"
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
            class="full-width-control"
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
        <FormActionBar
          :loading="creating"
          submit-text="创建"
          @cancel="showCreateDialog = false"
          @submit="createTicket"
        />
      </template>
    </AppDialog>
    <AppDialog
      v-model="showDetailDialog"
      title="工单详情"
      width="800px"
      mobile-width="94%"
      class="ticket-detail-dialog"
    >
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
          <EmptyState
            v-else
            class="empty-replies"
            title="暂无回复"
            description="工单有新进展后会显示在这里。"
            :icon-size="48"
          />
        </div>
        <div class="ticket-reply-form">
          <el-input
            v-model="replyContent"
            type="textarea"
            :rows="3"
            placeholder="输入回复内容"
            :disabled="replying"
          />
          <el-button
            type="primary"
            class="reply-submit-button"
            :loading="replying"
            :disabled="replying"
            @click="addReply"
          >
            发送回复
          </el-button>
        </div>
      </div>
    </AppDialog>
  </div>
</template>
<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { UserFilled } from '@element-plus/icons-vue'
import { ticketAPI } from '@/utils/api'
import { useMobile } from '@/composables/useMobile'
import { usePersistentTableColumns } from '@/composables/usePersistentTableColumns'
import {
  getTicketStatusText as getStatusText,
  getTicketStatusType as getStatusTagType,
  getTicketTypeText as getTypeText,
  getTicketTypeType as getTypeTagType,
  getTicketPriorityText as getPriorityText,
  getTicketPriorityType as getPriorityTagType
} from '@/utils/statusMaps'
import AppDialog from '@/components/AppDialog.vue'
import EmptyState from '@/components/EmptyState.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'

const TICKETS_TABLE_STORAGE_KEY = 'user_tickets_table_settings'
const ticketTableRef = ref(null)
const TICKET_COLUMN_KEYS = ['ticket_no', 'title', 'type', 'status', 'priority', 'created_at', 'actions']
const { columnWidths, handleColumnResize: handleTicketColumnResize } = usePersistentTableColumns(
  TICKETS_TABLE_STORAGE_KEY,
  {
    ticket_no: 180,
    title: 200,
    type: 100,
    status: 100,
    priority: 100,
    created_at: 180,
    actions: 150
  },
  TICKET_COLUMN_KEYS
)

const isMobile = useMobile()
onMounted(() => {
  loadTickets()
})
const loading = ref(false)
const creating = ref(false)
const replying = ref(false)
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
const mobileTicketFields = computed(() => [
  {
    key: 'status',
    label: '状态',
    type: 'tag',
    tagType: value => getStatusTagType(value),
    formatter: value => getStatusText(value)
  },
  { key: 'ticket_no', label: '工单编号' },
  {
    key: 'type',
    label: '类型',
    type: 'tag',
    tagType: value => getTypeTagType(value),
    formatter: value => getTypeText(value)
  },
  {
    key: 'priority',
    label: '优先级',
    type: 'tag',
    tagType: value => getPriorityTagType(value),
    formatter: value => getPriorityText(value)
  },
  { key: 'created_at', label: '创建时间' }
])
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
  if (replying.value) return
  replying.value = true
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
  } finally {
    replying.value = false
  }
}
const handleFilterChange = () => {
  pagination.page = 1 // 重置到第一页
  loadTickets()
}
</script>
<style scoped lang="scss">
.tickets-container {
  padding: 0;
}

:global(.user-layout .tickets-container) {
  display: block !important;
  max-width: none !important;
  width: 100% !important;
}

.tickets-workspace {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  align-items: start;
  gap: 14px;
  width: 100%;
}

:global(.user-layout .tickets-container .tickets-workspace) {
  display: grid !important;
  grid-template-columns: minmax(0, 1fr) !important;
  gap: 14px !important;
  width: 100% !important;
  max-width: none !important;
}

:global(.user-layout .tickets-container .list-card) {
  min-width: 0;
  width: 100% !important;
  max-width: none !important;
  justify-self: stretch !important;
}

.tickets-container .table-wrapper {
  min-width: 0;
  width: 100%;
}

:global(.user-layout .tickets-container .table-wrapper),
:global(.user-layout .tickets-container .ticket-table) {
  width: 100% !important;
}
.section-stack {
  display: grid;
  gap: 14px;
}
.card {
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  overflow: hidden;
}
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid #ebeef5;
}
.card-title {
  margin: 0;
  color: #303133;
  font-size: 16px;
  font-weight: 700;
}
.card-body {
  padding: 16px;
}

.ticket-filter-body {
  padding: 16px 16px 6px;
  border-bottom: 1px solid #ebeef5;
  background: #fff;
}

.ticket-filter-form {
  display: grid !important;
  grid-template-columns: repeat(3, minmax(150px, 1fr));
  align-items: end;
  gap: 12px;
  min-width: 0;

  :deep(.el-form-item) {
    margin-right: 0;
    margin-bottom: 0;
    min-width: 0;
  }

  :deep(.el-form-item__label) {
    font-weight: 500;
    color: #606266;
  }

  :deep(.el-form-item__content) {
    width: 100%;
    min-width: 0;
  }
}

.ticket-filter-select {
  width: 100%;
  min-width: 0;
}

.ticket-page-size-select {
  width: 120px;
}

:global(.user-layout .tickets-container .ticket-filter-form) {
  display: grid !important;
  grid-template-columns: repeat(3, minmax(150px, 1fr)) !important;
  align-items: end !important;
  gap: 12px !important;
}
@media (max-width: 1100px) {
  .ticket-filter-form,
  :global(.user-layout .tickets-container .ticket-filter-form) {
    grid-template-columns: repeat(2, minmax(0, 1fr)) !important;
  }
}
.side-full-button {
  width: 100%;
  margin-top: 2px;
}
.ticket-item {
  padding: 14px 0;
  border-bottom: 1px solid #ebeef5;
}
.ticket-item:last-child {
  border-bottom: 0;
}
.item-title {
  color: #303133;
  font-weight: 700;
}
.item-meta {
  margin-top: 6px;
  color: #909399;
  font-size: 12px;
  line-height: 1.5;
}

.ticket-filter-panel,
.create-ticket-mobile-action {
  margin-bottom: 0;
}

.ticket-table,
.full-width-control {
  width: 100%;
}

.ticket-title-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;

  span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.create-ticket-mobile-action {
  :deep(.el-button) {
    width: 100%;
  }
}

.ticket-mobile-header {
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

:global(.user-layout .tickets-container .mobile-card-actions) {
  padding: 12px 16px;
  border-top: 1px solid #ebeef5;
}

:global(.user-layout .tickets-container .mobile-card-actions .el-button) {
  width: 100%;
  min-height: 44px;
}

.inline-badge {
  margin-left: 4px;
}

.reply-submit-button {
  margin-top: 10px;
}

.ticket-form {
  :deep(.el-form-item) {
    margin-bottom: 18px;
  }

  :deep(.el-input__wrapper),
  :deep(.el-textarea__inner) {
    border-radius: 6px;
  }
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
  flex-wrap: wrap;
  color: #606266;
  font-size: 13px;
}
.ticket-content {
  margin: 20px 0;
  padding: 15px;
  background: #f8fafc;
  border: 1px solid #dcdfe6;
  border-radius: 8px;

  p {
    margin: 0;
    color: var(--theme-text, #303133);
    line-height: 1.7;
    word-break: break-word;
  }
}
.ticket-replies {
  margin: 20px 0;
  :is(h4) {
    margin-bottom: 15px;
  }
}
.empty-replies {
  min-height: 160px;
  padding: 24px 16px;
  border: 1px dashed #dcdfe6;
  border-radius: 8px;
  background: #fafafa;
}
.reply-item {
  margin-bottom: 15px;
  padding: 15px;
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  transition: border-color 0.18s ease, background-color 0.18s ease;
  &.user-reply {
    background: #f8fafc;
    border-left: 3px solid #909399;
  }
  &.admin-reply {
    background: #f4f8ff;
    border-color: #c6e2ff;
    border-left: 3px solid #409eff;
  }
}
.reply-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
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
  color: var(--theme-text, #303133);
  line-height: 1.6;
  font-size: 14px;
  white-space: pre-wrap;
  word-break: break-word;
  &.admin-content {
    color: var(--theme-text, #303133);
    font-weight: 500;
  }
}
.ticket-reply-form {
  margin-top: 20px;

  :deep(.el-textarea__inner) {
    border-radius: 6px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .reply-item {
    transition: none;
  }
}
@media (max-width: 768px) {
  .tickets-container {
    padding: 10px;
  }
  .tickets-workspace {
    grid-template-columns: 1fr;
  }

  .ticket-filter-form {
    display: grid !important;
    grid-template-columns: 1fr;

    :deep(.el-form-item) {
      margin-right: 0;
    }
  }

  :global(.user-layout .tickets-container .ticket-filter-form) {
    display: grid !important;
    grid-template-columns: 1fr !important;
  }

  .ticket-filter-select,
  .ticket-page-size-select {
    width: 100%;
  }
  .card-header {
    align-items: flex-start;
    flex-direction: column;

    :deep(.el-button) {
      width: 100%;
    }
  }
  .ticket-form {
    :deep(.el-form-item) {
      margin-bottom: 16px;
      .el-form-item__label {
        width: 100% !important;
        text-align: left;
        margin-bottom: 8px;
        padding: 0;
        font-size: 14px;
        font-weight: 500;
        color: var(--theme-text, #303133);
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
      color: var(--theme-text, #303133);
      margin-bottom: 8px;
      line-height: 1.5;
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
