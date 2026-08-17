<template>
  <div class="list-container admin-tickets">
    <div class="page-header">
      <h1>工单管理</h1>
      <div class="header-actions">
        <el-button @click="loadTickets" :loading="loading" class="refresh-btn">
          <el-icon><Refresh /></el-icon>
          <span class="desktop-only">刷新</span>
        </el-button>
      </div>
    </div>
    <div class="filter-bar">
      <el-input
        v-model="filters.keyword"
        placeholder="搜索工单编号、标题或内容"
        class="keyword-input"
        clearable
        @input="debouncedLoadTickets"
        @clear="loadTickets"
        @keyup.enter="loadTickets"
      />
      <el-select v-model="filters.status" placeholder="状态筛选" clearable class="filter-select" @change="loadTickets">
        <el-option label="待处理" value="pending" />
        <el-option label="处理中" value="processing" />
        <el-option label="已解决" value="resolved" />
        <el-option label="已关闭" value="closed" />
      </el-select>
      <el-select v-model="filters.type" placeholder="类型筛选" clearable class="filter-select" @change="loadTickets">
        <el-option label="技术问题" value="technical" />
        <el-option label="账单问题" value="billing" />
        <el-option label="账户问题" value="account" />
        <el-option label="其他" value="other" />
      </el-select>
      <el-select v-model="filters.priority" placeholder="优先级筛选" clearable class="filter-select" @change="loadTickets">
        <el-option label="低" value="low" />
        <el-option label="普通" value="normal" />
        <el-option label="高" value="high" />
        <el-option label="紧急" value="urgent" />
      </el-select>
      <el-button type="primary" @click="loadTickets">搜索</el-button>
      <el-button @click="resetFilters">重置</el-button>
    </div>
    <div class="stats-cards" v-if="statistics">
      <el-card class="stat-card">
        <div class="stat-item">
          <div class="stat-value">{{ statistics.total || 0 }}</div>
          <div class="stat-label">总工单</div>
        </div>
      </el-card>
      <el-card class="stat-card">
        <div class="stat-item">
          <div class="stat-value warning">{{ statistics.pending || 0 }}</div>
          <div class="stat-label">待处理</div>
        </div>
      </el-card>
      <el-card class="stat-card">
        <div class="stat-item">
          <div class="stat-value primary">{{ statistics.processing || 0 }}</div>
          <div class="stat-label">处理中</div>
        </div>
      </el-card>
      <el-card class="stat-card">
        <div class="stat-item">
          <div class="stat-value success">{{ statistics.resolved || 0 }}</div>
          <div class="stat-label">已解决</div>
        </div>
      </el-card>
    </div>
    <ResponsiveDataView
      class="tickets-data-view"
      :data="tickets"
      :loading="loading"
      :fields="mobileTicketFields"
      title-field="ticket_no"
      empty-title="暂无工单数据"
      empty-description="当前筛选条件下没有工单，调整筛选后可重新查询"
    >
      <template #table>
        <el-table :data="tickets" v-loading="loading" class="tickets-table" stripe border empty-text=" ">
          <el-table-column prop="ticket_no" label="工单编号" width="180" />
          <el-table-column prop="title" label="标题" min-width="200">
            <template #default="{ row }">
              <div class="ticket-title-cell">
                <span>{{ row.title }}</span>
                <el-badge
                  v-if="row.has_unread && (row.unread_replies > 0 || row.has_new_ticket)"
                  :value="row.unread_replies > 0 ? row.unread_replies : (row.has_new_ticket ? '新' : '')"
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
          <el-table-column label="提交用户" min-width="160">
            <template #default="{ row }">
              <span v-if="row.user && row.user.email">{{ row.user.email }}</span>
              <span v-else>用户ID: {{ row.user_id }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="replies_count" label="回复数" width="80" />
          <el-table-column prop="created_at" label="创建时间" width="180" />
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="viewTicket(row.id)">
                查看
                <el-badge
                  v-if="row.has_unread && (row.unread_replies > 0 || row.has_new_ticket)"
                  :value="row.unread_replies > 0 ? row.unread_replies : (row.has_new_ticket ? '新' : '')"
                  :max="99"
                  type="danger"
                  class="button-badge"
                />
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <EmptyState
          v-if="!loading && tickets.length === 0"
          class="desktop-empty-state"
          title="暂无工单数据"
          description="当前筛选条件下没有工单，调整筛选后可重新查询"
        />
      </template>
      <template #header="{ item }">
        <div class="ticket-card-header">
          <div class="ticket-no">{{ item.ticket_no }}</div>
          <div class="ticket-badges">
            <el-tag :type="getStatusTagType(item.status)" size="small">{{ getStatusText(item.status) }}</el-tag>
            <el-tag :type="getTypeTagType(item.type)" size="small">{{ getTypeText(item.type) }}</el-tag>
            <el-badge
              v-if="item.has_unread && (item.unread_replies > 0 || item.has_new_ticket)"
              :value="item.unread_replies > 0 ? item.unread_replies : (item.has_new_ticket ? '新' : '')"
              :max="99"
              type="danger"
            />
          </div>
        </div>
        <div class="ticket-card-title">{{ item.title }}</div>
      </template>
      <template #field-user="{ item }">
        {{ item.user && item.user.email ? item.user.email : '用户ID: ' + item.user_id }}
      </template>
      <template #field-priority="{ item }">
        <el-tag :type="getPriorityTagType(item.priority)" size="small">{{ getPriorityText(item.priority) }}</el-tag>
      </template>
      <template #field-replies_count="{ item }">
        {{ item.replies_count || 0 }}条回复
      </template>
      <template #field-created_at="{ item }">
        {{ formatTime(item.created_at) }}
      </template>
      <template #actions="{ item }">
        <el-button size="small" @click="viewTicket(item.id)">
          查看
          <el-badge
            v-if="item.has_unread && (item.unread_replies > 0 || item.has_new_ticket)"
            :value="item.unread_replies > 0 ? item.unread_replies : (item.has_new_ticket ? '新' : '')"
            :max="99"
            type="danger"
            class="button-badge"
          />
        </el-button>
      </template>
    </ResponsiveDataView>
    <PaginationBar
      v-model:current-page="pagination.page"
      v-model:page-size="pagination.size"
      :total="pagination.total"
      @change="loadTickets"
    />
    <AppDrawer
      :model-value="showDetailDialog"
      :title="currentTicket ? `工单详情 - ${currentTicket.ticket_no}` : '工单详情'"
      size="780px"
      mobile-size="100%"
      class="ticket-detail-drawer"
      @update:model-value="handleDetailDrawerChange"
    >
      <div v-if="currentTicket" class="ticket-detail">
        <div class="mobile-ticket-header" v-if="isMobile">
          <div class="mobile-ticket-badges">
            <el-tag :type="getStatusTagType(currentTicket.status)" size="small">{{ getStatusText(currentTicket.status) }}</el-tag>
            <el-tag :type="getTypeTagType(currentTicket.type)" size="small">{{ getTypeText(currentTicket.type) }}</el-tag>
            <el-tag :type="getPriorityTagType(currentTicket.priority)" size="small">{{ getPriorityText(currentTicket.priority) }}</el-tag>
          </div>
          <div class="mobile-ticket-title">{{ currentTicket.title }}</div>
        </div>
        <el-card class="ticket-info-card" shadow="never">
          <template #header>
            <div class="card-header">
              <span>基本信息</span>
              <div class="ticket-status-badges desktop-only">
                <el-tag :type="getStatusTagType(currentTicket.status)">{{ getStatusText(currentTicket.status) }}</el-tag>
                <el-tag :type="getTypeTagType(currentTicket.type)">{{ getTypeText(currentTicket.type) }}</el-tag>
                <el-tag :type="getPriorityTagType(currentTicket.priority)">{{ getPriorityText(currentTicket.priority) }}</el-tag>
              </div>
            </div>
          </template>
          <el-descriptions :column="isMobile ? 1 : 2" border>
            <el-descriptions-item label="工单编号">{{ currentTicket.ticket_no }}</el-descriptions-item>
            <el-descriptions-item label="用户邮箱">
              <span v-if="currentTicket.user && currentTicket.user.email">{{ currentTicket.user.email }}</span>
              <span v-else>用户ID: {{ currentTicket.user_id }}</span>
            </el-descriptions-item>
            <el-descriptions-item :label="isMobile ? '' : '标题'" :span="isMobile ? 1 : 2">
              <span v-if="isMobile" class="mobile-label">标题：</span>{{ currentTicket.title }}
            </el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ formatTime(currentTicket.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="更新时间">{{ formatTime(currentTicket.updated_at) }}</el-descriptions-item>
            <el-descriptions-item label="解决时间" v-if="currentTicket.resolved_at">
              {{ formatTime(currentTicket.resolved_at) }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
        <el-card class="ticket-content-card" shadow="never" :class="{ 'mobile-card': isMobile }">
          <template #header>
            <span>工单内容</span>
          </template>
          <div class="ticket-content-text">{{ currentTicket.content }}</div>
        </el-card>
        <el-card class="ticket-notes-card" shadow="never" :class="{ 'mobile-card': isMobile }" v-if="currentTicket.admin_notes">
          <template #header>
            <span>管理员备注</span>
          </template>
          <div class="admin-notes-text">{{ currentTicket.admin_notes }}</div>
        </el-card>
        <el-card class="ticket-replies-card" shadow="never" :class="{ 'mobile-card': isMobile }">
          <template #header>
            <div class="card-header">
              <span>回复记录 ({{ currentTicket.replies?.length || 0 }})</span>
            </div>
          </template>
          <div class="replies-list">
            <div 
              v-for="reply in currentTicket.replies" 
              :key="reply.id" 
              class="reply-item" 
              :class="{ 
                'admin-reply': isAdminReply(reply), 
                'user-reply': !isAdminReply(reply),
                'unread-reply': reply.is_unread,
                'mobile-reply': isMobile 
              }"
            >
              <div class="reply-header">
                <div class="reply-author">
                  <el-tag 
                    :type="isAdminReply(reply) ? 'success' : 'info'" 
                    size="small"
                    effect="dark"
                  >
                    {{ isAdminReply(reply) ? '管理员' : '用户' }}
                  </el-tag>
                  <el-badge 
                    v-if="reply.is_unread" 
                    value="新" 
                    type="danger"
                    class="reply-badge"
                  />
                  <span class="reply-user-id" :class="{ 'mobile-hidden': isMobile }">用户ID: {{ reply.user_id }}</span>
                  <span class="reply-user-id mobile-only" v-if="isMobile">{{ reply.user_id }}</span>
                </div>
                <span class="reply-time">{{ formatTime(reply.created_at) }}</span>
              </div>
              <div class="reply-content" :class="{ 'unread-content': reply.is_unread }">{{ reply.content }}</div>
            </div>
            <EmptyState
              v-if="!currentTicket.replies || currentTicket.replies.length === 0"
              class="empty-replies"
              title="暂无回复"
              description="该工单还没有回复记录"
              :icon-size="56"
            />
          </div>
        </el-card>
        <div class="ticket-actions" :class="{ 'mobile-card': isMobile }">
          <el-card shadow="never" v-if="!isMobile">
            <template #header>
              <span>操作</span>
            </template>
            <div class="action-buttons">
              <el-button @click="showStatusDialog = true">更新状态</el-button>
              <el-button @click="showNotesDialog = true">添加备注</el-button>
            </div>
          </el-card>
          <div class="mobile-action-buttons" v-else>
            <el-button 
              type="primary" 
              @click="showStatusDialog = true" 
              class="mobile-action-btn"
            >
              更新状态
            </el-button>
            <el-button 
              @click="showNotesDialog = true" 
              class="mobile-action-btn"
            >
              添加备注
            </el-button>
          </div>
        </div>
        <div class="ticket-reply-form" :class="{ 'mobile-card': isMobile }">
          <el-card shadow="never">
            <template #header>
              <span>回复工单</span>
            </template>
            <el-input
              v-model="replyContent"
              type="textarea"
              :rows="isMobile ? 5 : 4"
              placeholder="输入回复内容..."
              :maxlength="2000"
              show-word-limit
            />
            <el-button 
              type="primary" 
              @click="addReply" 
              class="reply-submit-btn"
              :class="{ 'is-mobile': isMobile }"
              :loading="replying"
              :block="isMobile"
            >
              发送回复
            </el-button>
          </el-card>
        </div>
      </div>
    </AppDrawer>
    <AppDialog
      v-model="showStatusDialog"
      title="更新工单状态"
      width="400px"
      mobile-width="92%"
      :loading="statusUpdating"
    >
      <el-select v-model="newStatus" placeholder="选择新状态" class="form-control">
        <el-option label="待处理" value="pending" />
        <el-option label="处理中" value="processing" />
        <el-option label="已解决" value="resolved" />
        <el-option label="已关闭" value="closed" />
      </el-select>
      <template #footer>
        <FormActionBar
          :loading="statusUpdating"
          submit-text="确定"
          @cancel="showStatusDialog = false"
          @submit="updateStatus"
        />
      </template>
    </AppDialog>
    <AppDialog
      v-model="showNotesDialog"
      title="添加管理员备注"
      width="500px"
      mobile-width="92%"
      :loading="notesUpdating"
    >
      <el-input
        v-model="adminNotes"
        type="textarea"
        :rows="5"
        placeholder="输入管理员备注..."
      />
      <template #footer>
        <FormActionBar
          :loading="notesUpdating"
          submit-text="确定"
          @cancel="showNotesDialog = false"
          @submit="updateNotes"
        />
      </template>
    </AppDialog>
  </div>
</template>
<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { Refresh, Search } from '@element-plus/icons-vue'
import { ticketAPI } from '@/utils/api'
import { useMobile } from '@/composables/useMobile'
import { debounce } from '@/composables/useDebounce'
import PaginationBar from '@/components/PaginationBar.vue'
import AppDrawer from '@/components/AppDrawer.vue'
import AppDialog from '@/components/AppDialog.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import EmptyState from '@/components/EmptyState.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
const loading = ref(false)
const replying = ref(false)
const statusUpdating = ref(false)
const notesUpdating = ref(false)
const tickets = ref([])
const showDetailDialog = ref(false)
const showStatusDialog = ref(false)
const showNotesDialog = ref(false)
const currentTicket = ref(null)
const replyContent = ref('')
const newStatus = ref('')
const adminNotes = ref('')
const statistics = ref(null)
const isMobile = useMobile()
const filters = reactive({
  keyword: '',
  status: '',
  type: '',
  priority: ''
})
const pagination = reactive({
  page: 1,
  size: 10,
  total: 0
})
const mobileTicketFields = computed(() => [
  { key: 'user', label: '提交用户' },
  { key: 'priority', label: '优先级' },
  { key: 'replies_count', label: '回复数' },
  { key: 'created_at', label: '创建时间' }
])
const loadTickets = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      size: pagination.size
    }
    if (filters.keyword && filters.keyword.trim()) params.keyword = filters.keyword.trim()
    if (filters.status && filters.status.trim()) params.status = filters.status.trim()
    if (filters.type && filters.type.trim()) params.type = filters.type.trim()
    if (filters.priority && filters.priority.trim()) params.priority = filters.priority.trim()
    const response = await ticketAPI.getAllTickets(params)
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
// 搜索输入实时生效，无需再次点击搜索按钮（500ms 防抖）
const debouncedLoadTickets = debounce(loadTickets, 500)
const loadStatistics = async () => {
  try {
    const response = await ticketAPI.getTicketStatistics()
    if (response.data.success) {
      statistics.value = response.data.data
    }
  } catch (error) {
    }
}
const viewTicket = async (ticketId) => {
  try {
    const response = await ticketAPI.getAdminTicket(ticketId)
    if (response.data.success) {
      currentTicket.value = response.data.data?.ticket || response.data.data
      showDetailDialog.value = true
      markTicketReadLocally(ticketId)
      window.dispatchEvent(new CustomEvent('ticket-viewed'))
      await loadTickets()
    }
  } catch (error) {
    ElMessage.error('加载工单详情失败: ' + (error.response?.data?.message || error.message))
  }
}
const isAdminReply = (reply) => {
  return reply?.is_admin === true || reply?.is_admin === 'true' || reply?.is_admin_reply === true
}
const markTicketReadLocally = (ticketId) => {
  tickets.value = tickets.value.map(ticket => {
    if (ticket.id !== ticketId) return ticket
    return {
      ...ticket,
      has_unread: false,
      has_new_ticket: false,
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
    console.error('[前端] 工单ID不存在:', currentTicket.value)
    ElMessage.error('工单信息不完整，请刷新后重试')
    return
  }
  replying.value = true
  try {
    const response = await ticketAPI.addReply(currentTicket.value.id, { content: replyContent.value })
    if (response.data && response.data.success) {
      ElMessage.success('回复成功')
      replyContent.value = ''
      await viewTicket(currentTicket.value.id)
      await loadStatistics()
    } else {
      console.error('[前端] 回复失败，响应数据:', response.data)
      ElMessage.error(response.data?.message || '回复失败')
    }
  } catch (error) {
    console.error('[前端] 回复异常:', error)
    console.error('[前端] 错误详情:', error.response?.data)
    ElMessage.error(error.response?.data?.detail || error.response?.data?.message || '回复失败')
  } finally {
    replying.value = false
  }
}
const updateStatus = async () => {
  if (!currentTicket.value || !newStatus.value) return
  statusUpdating.value = true
  try {
    const response = await ticketAPI.updateTicket(currentTicket.value.id, { status: newStatus.value })
    if (response.data.success) {
      ElMessage.success('状态更新成功')
      showStatusDialog.value = false
      newStatus.value = ''
      await viewTicket(currentTicket.value.id)
      await loadStatistics()
    }
  } catch (error) {
    ElMessage.error('更新状态失败')
  } finally {
    statusUpdating.value = false
  }
}
const updateNotes = async () => {
  if (!currentTicket.value) return
  notesUpdating.value = true
  try {
    const response = await ticketAPI.updateTicket(currentTicket.value.id, { admin_notes: adminNotes.value })
    if (response.data.success) {
      ElMessage.success('备注添加成功')
      showNotesDialog.value = false
      adminNotes.value = ''
      await viewTicket(currentTicket.value.id)
    }
  } catch (error) {
    ElMessage.error('添加备注失败')
  } finally {
    notesUpdating.value = false
  }
}
const closeDetailDialog = () => {
  showDetailDialog.value = false
  currentTicket.value = null
  replyContent.value = ''
  newStatus.value = ''
  adminNotes.value = ''
}
const handleDetailDrawerChange = (value) => {
  if (value) {
    showDetailDialog.value = true
    return
  }
  closeDetailDialog()
}
const formatTime = (timeStr) => {
  if (!timeStr) return '-'
  return new Date(timeStr).toLocaleString('zh-CN')
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
  if (!status) return 'info'
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
  if (!type) return 'info'
  return 'info'
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
  if (!priority) return 'info'
  const map = {
    low: 'info',
    normal: 'info',
    high: 'warning',
    urgent: 'danger'
  }
  return map[priority] || 'info'
}
const resetFilters = () => {
  filters.keyword = ''
  filters.status = ''
  filters.type = ''
  filters.priority = ''
  loadTickets()
}
onMounted(async () => {
  await Promise.all([loadStatistics(), loadTickets()])
})
</script>
<style scoped lang="scss">
.admin-tickets-container {
  padding: 20px;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.filter-bar {
  display: grid;
  grid-template-columns: minmax(260px, 1.4fr) repeat(3, minmax(150px, 0.8fr)) max-content max-content;
  align-items: end;
  gap: 10px;
  margin-bottom: 20px;
  padding: 14px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 8px;
}
.keyword-input {
  width: 100%;
  min-width: 0;
}
.filter-select {
  width: 100%;
  min-width: 0;
}
.form-control {
  width: 100%;
}
.tickets-table {
  width: 100%;
  margin-top: 20px;
}
.ticket-title-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}
.button-badge {
  margin-left: 4px;
}
.reply-badge {
  margin-left: 8px;
}
.reply-submit-btn {
  margin-top: 10px;
  &.is-mobile {
    width: 100%;
    margin-top: 12px;
  }
}
.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 20px;
  @media (max-width: 768px) {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }
}
.stat-card {
  text-align: center;
}
.stat-item {
  .stat-value {
    font-size: 32px;
    font-weight: bold;
    color: #409eff;
    &.warning {
      color: #e6a23c;
    }
    &.primary {
      color: #409eff;
    }
    &.success {
      color: #67c23a;
    }
  }
  .stat-label {
    margin-top: 8px;
    color: #666;
  }
}
.ticket-detail {
  .ticket-info-card,
  .ticket-content-card,
  .ticket-notes-card,
  .ticket-replies-card {
    margin-bottom: 20px;
  }
  .ticket-status-badges {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  .ticket-content-text,
  .admin-notes-text {
    white-space: pre-wrap;
    line-height: 1.6;
    word-break: break-word;
  }
  .replies-list {
    .reply-item {
      &.unread-reply {
        background: #fff8e6;
        border-left: 4px solid #faad14;
      }
      &.user-reply.unread-reply {
        background: #fff8e6;
        border-left: 4px solid #faad14;
      }
      .unread-content {
        font-weight: 500;
        color: #1a1a1a;
      }
    }
    .reply-item {
      padding: 15px;
      margin-bottom: 15px;
      background: #f5f5f5;
      border-radius: 4px;
      &.admin-reply {
        background: #e8f4fd;
      }
      .reply-header {
        display: flex;
        justify-content: space-between;
        margin-bottom: 10px;
        flex-wrap: wrap;
        gap: 8px;
        .reply-author {
          display: flex;
          align-items: center;
          gap: 10px;
          flex-wrap: wrap;
        }
        .reply-time {
          color: #999;
          font-size: 12px;
        }
      }
      .reply-content {
        white-space: pre-wrap;
        line-height: 1.6;
        word-break: break-word;
      }
    }
    .empty-replies {
      min-height: 180px;
    }
  }
  .action-buttons {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
  }
  .mobile-label {
    font-weight: 500;
    color: #666;
    margin-right: 8px;
  }
}
.mobile-ticket-header {
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
  margin-bottom: 16px;
  .mobile-ticket-badges {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
    margin-bottom: 12px;
  }
  .mobile-ticket-title {
    font-size: 18px;
    font-weight: 600;
    color: #333;
    line-height: 1.4;
  }
}
.mobile-card {
  margin-top: 16px !important;
  margin-bottom: 16px !important;
  :deep(.el-card__header) {
    padding: 12px 16px;
    font-size: 14px;
    font-weight: 600;
  }
  :deep(.el-card__body) {
    padding: 16px;
  }
}
.mobile-reply {
  padding: 12px !important;
  margin-bottom: 12px !important;
  .reply-header {
    margin-bottom: 8px !important;
    .reply-author {
      gap: 6px !important;
    }
    .reply-time {
      font-size: 11px !important;
      width: 100%;
      margin-top: 4px;
    }
  }
  .reply-content {
    font-size: 14px;
    line-height: 1.5;
  }
}
.mobile-only {
  display: none;
}
@media (max-width: 768px) {
  .mobile-only {
    display: inline;
  }
  .mobile-hidden {
    display: none !important;
  }
}
@media (max-width: 768px) {
  .admin-tickets-container {
    padding: 12px;
  }
  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 16px;
    :is(h1) {
      font-size: 20px;
      margin: 0;
    }
    .header-actions {
      width: 100%;
      display: flex;
      justify-content: flex-end;
    }
    .refresh-btn {
      width: auto;
      padding: 8px 12px;
    }
  }
  .filter-bar {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
    margin-bottom: 14px;
    padding: 12px;

    .keyword-input {
      grid-column: 1 / -1;
    }

    .filter-select {
      width: 100%;
    }

    .el-button {
      width: 100%;
      min-width: 0;
      min-height: 44px;
      margin-left: 0;
    }
  }
  .tickets-data-view {
    margin-top: 16px;
  }
  .ticket-detail-drawer {
    .ticket-detail {
      padding-bottom: 20px;
    }
  }
  .stat-card {
    padding: 12px;
    .stat-item {
      .stat-value {
        font-size: 24px;
      }
      .stat-label {
        font-size: 12px;
        margin-top: 4px;
      }
    }
  }
}
.ticket-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 10px;
  min-width: 0;
  margin-bottom: 10px;
}
.ticket-no {
  flex: 1;
  min-width: 0;
  font-weight: 700;
  font-size: 14px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ticket-badges {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: flex-end;
}
.ticket-card-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  line-height: 1.45;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
}
.desktop-empty-state {
  border: 1px solid #ebeef5;
  border-top: none;
  min-height: 220px;
}
.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}
@media (min-width: 769px) {
  .tickets-data-view :deep(.responsive-data-view__cards) {
    display: none !important;
  }
}
</style>
