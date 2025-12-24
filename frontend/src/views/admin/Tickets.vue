<template>
  <div class="admin-tickets-container">
    <div class="page-header">
      <h1>工单管理</h1>
      <div class="header-actions">
        <el-button @click="loadTickets" :loading="loading" class="refresh-btn">
          <el-icon><Refresh /></el-icon>
          <span class="desktop-only">刷新</span>
        </el-button>
      </div>
    </div>

    <!-- 移动端筛选栏 -->
    <div class="mobile-action-bar" v-if="isMobile">
      <div class="mobile-search-section">
        <div class="search-input-wrapper">
          <el-input
            v-model="filters.keyword"
            placeholder="搜索工单编号、标题或内容"
            class="mobile-search-input"
            clearable
            @keyup.enter="loadTickets"
          />
          <el-button 
            @click="loadTickets" 
            class="search-button-inside"
            type="default"
            plain
          >
            <el-icon><Search /></el-icon>
          </el-button>
        </div>
      </div>
      <div class="mobile-filter-buttons">
        <el-button
          size="small"
          :type="showFilterDrawer ? 'primary' : 'default'"
          plain
          @click="showFilterDrawer = true"
        >
          <el-icon><Filter /></el-icon>
          筛选
        </el-button>
        <el-button size="small" type="default" plain @click="resetFilters">
          <el-icon><Refresh /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <!-- 桌面端筛选栏 -->
    <div class="filter-bar desktop-only">
      <el-input
        v-model="filters.keyword"
        placeholder="搜索工单编号、标题或内容"
        style="width: 250px"
        clearable
        @clear="loadTickets"
      />
      <el-select v-model="filters.status" placeholder="状态筛选" clearable style="width: 150px">
        <el-option label="待处理" value="pending" />
        <el-option label="处理中" value="processing" />
        <el-option label="已解决" value="resolved" />
        <el-option label="已关闭" value="closed" />
      </el-select>
      <el-select v-model="filters.type" placeholder="类型筛选" clearable style="width: 150px">
        <el-option label="技术问题" value="technical" />
        <el-option label="账单问题" value="billing" />
        <el-option label="账户问题" value="account" />
        <el-option label="其他" value="other" />
      </el-select>
      <el-select v-model="filters.priority" placeholder="优先级筛选" clearable style="width: 150px">
        <el-option label="低" value="low" />
        <el-option label="普通" value="normal" />
        <el-option label="高" value="high" />
        <el-option label="紧急" value="urgent" />
      </el-select>
      <el-button type="primary" @click="loadTickets">搜索</el-button>
    </div>

    <!-- 移动端筛选抽屉 -->
    <el-drawer
      v-model="showFilterDrawer"
      title="筛选条件"
      :size="isMobile ? '85%' : '400px'"
      direction="rtl"
    >
      <div class="filter-drawer-content">
        <el-form label-width="100px">
          <el-form-item label="状态">
            <el-select v-model="filters.status" placeholder="选择状态" clearable style="width: 100%">
              <el-option label="待处理" value="pending" />
              <el-option label="处理中" value="processing" />
              <el-option label="已解决" value="resolved" />
              <el-option label="已关闭" value="closed" />
            </el-select>
          </el-form-item>
          <el-form-item label="类型">
            <el-select v-model="filters.type" placeholder="选择类型" clearable style="width: 100%">
              <el-option label="技术问题" value="technical" />
              <el-option label="账单问题" value="billing" />
              <el-option label="账户问题" value="account" />
              <el-option label="其他" value="other" />
            </el-select>
          </el-form-item>
          <el-form-item label="优先级">
            <el-select v-model="filters.priority" placeholder="选择优先级" clearable style="width: 100%">
              <el-option label="低" value="low" />
              <el-option label="普通" value="normal" />
              <el-option label="高" value="high" />
              <el-option label="紧急" value="urgent" />
            </el-select>
          </el-form-item>
        </el-form>
        <div class="filter-drawer-actions">
          <el-button @click="resetFilters" style="width: 48%">重置</el-button>
          <el-button type="primary" @click="applyFilters" style="width: 48%">应用</el-button>
        </div>
      </div>
    </el-drawer>

    <!-- 统计卡片 -->
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

    <!-- 桌面端工单列表 -->
    <el-table :data="tickets" v-loading="loading" class="desktop-only" style="width: 100%; margin-top: 20px">
      <el-table-column prop="ticket_no" label="工单编号" width="180" />
      <el-table-column prop="title" label="标题" min-width="200">
        <template #default="{ row }">
          <div style="display: flex; align-items: center; gap: 8px;">
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
      <el-table-column prop="user_id" label="用户ID" width="100" />
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
              style="margin-left: 4px;"
            />
          </el-button>
          <el-button size="small" type="primary" @click="assignTicket(row)" v-if="!row.assigned_to">分配</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 移动端工单卡片列表 -->
    <div class="mobile-tickets-list" v-if="isMobile" v-loading="loading">
      <div
        v-for="ticket in tickets"
        :key="ticket.id"
        class="mobile-ticket-card"
        @click="viewTicket(ticket.id)"
      >
        <div class="ticket-card-header">
          <div class="ticket-no">{{ ticket.ticket_no }}</div>
          <div class="ticket-badges">
            <el-tag :type="getStatusTagType(ticket.status)" size="small">{{ getStatusText(ticket.status) }}</el-tag>
            <el-tag :type="getTypeTagType(ticket.type)" size="small">{{ getTypeText(ticket.type) }}</el-tag>
            <el-badge 
              v-if="ticket.has_unread && (ticket.unread_replies > 0 || ticket.has_new_ticket)" 
              :value="ticket.unread_replies > 0 ? ticket.unread_replies : (ticket.has_new_ticket ? '新' : '')" 
              :max="99"
              type="danger"
            />
          </div>
        </div>
        <div class="ticket-card-title">
          {{ ticket.title }}
        </div>
        <div class="ticket-card-info">
          <span class="info-item">
            <el-icon><User /></el-icon>
            {{ ticket.user_id }}
          </span>
          <span class="info-item">
            <el-icon><ChatLineRound /></el-icon>
            {{ ticket.replies_count || 0 }}条回复
          </span>
          <span class="info-item">
            <el-icon><Clock /></el-icon>
            {{ formatTime(ticket.created_at) }}
          </span>
        </div>
        <div class="ticket-card-actions">
          <el-button size="small" @click.stop="viewTicket(ticket.id)">查看</el-button>
          <el-button size="small" type="primary" @click.stop="assignTicket(ticket)" v-if="!ticket.assigned_to">分配</el-button>
        </div>
      </div>
      <div v-if="tickets.length === 0" class="empty-state">
        <el-empty description="暂无工单数据" />
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

    <!-- 工单详情对话框 -->
    <el-dialog
      v-model="showDetailDialog"
      :title="currentTicket ? `工单详情 - ${currentTicket.ticket_no}` : '工单详情'"
      :width="isMobile ? '100%' : '900px'"
      :fullscreen="isMobile"
      @close="closeDetailDialog"
      class="ticket-detail-dialog"
      :close-on-click-modal="!isMobile"
    >
      <div v-if="currentTicket" class="ticket-detail">
        <!-- 移动端顶部状态栏 -->
        <div class="mobile-ticket-header" v-if="isMobile">
          <div class="mobile-ticket-badges">
            <el-tag :type="getStatusTagType(currentTicket.status)" size="small">{{ getStatusText(currentTicket.status) }}</el-tag>
            <el-tag :type="getTypeTagType(currentTicket.type)" size="small">{{ getTypeText(currentTicket.type) }}</el-tag>
            <el-tag :type="getPriorityTagType(currentTicket.priority)" size="small">{{ getPriorityText(currentTicket.priority) }}</el-tag>
          </div>
          <div class="mobile-ticket-title">{{ currentTicket.title }}</div>
        </div>

        <!-- 工单基本信息 -->
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
            <el-descriptions-item label="用户ID">{{ currentTicket.user_id }}</el-descriptions-item>
            <el-descriptions-item :label="isMobile ? '' : '标题'" :span="isMobile ? 1 : 2">
              <span v-if="isMobile" class="mobile-label">标题：</span>{{ currentTicket.title }}
            </el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ formatTime(currentTicket.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="更新时间">{{ formatTime(currentTicket.updated_at) }}</el-descriptions-item>
            <el-descriptions-item label="分配给" v-if="currentTicket.assigned_to">
              {{ currentTicket.assigned_to }}
            </el-descriptions-item>
            <el-descriptions-item label="解决时间" v-if="currentTicket.resolved_at">
              {{ formatTime(currentTicket.resolved_at) }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>

        <!-- 工单内容 -->
        <el-card class="ticket-content-card" shadow="never" :class="{ 'mobile-card': isMobile }">
          <template #header>
            <span>工单内容</span>
          </template>
          <div class="ticket-content-text">{{ currentTicket.content }}</div>
        </el-card>

        <!-- 管理员备注 -->
        <el-card class="ticket-notes-card" shadow="never" :class="{ 'mobile-card': isMobile }" v-if="currentTicket.admin_notes">
          <template #header>
            <span>管理员备注</span>
          </template>
          <div class="admin-notes-text">{{ currentTicket.admin_notes }}</div>
        </el-card>

        <!-- 回复记录 -->
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
                'admin-reply': reply.is_admin === 'true', 
                'user-reply': reply.is_admin !== 'true',
                'unread-reply': reply.is_unread,
                'mobile-reply': isMobile 
              }"
            >
              <div class="reply-header">
                <div class="reply-author">
                  <el-tag 
                    :type="reply.is_admin === 'true' ? 'success' : 'info'" 
                    size="small"
                    effect="dark"
                  >
                    {{ reply.is_admin === 'true' ? '管理员' : '用户' }}
                  </el-tag>
                  <el-badge 
                    v-if="reply.is_unread" 
                    value="新" 
                    type="danger"
                    style="margin-left: 8px;"
                  />
                  <span class="reply-user-id" :class="{ 'mobile-hidden': isMobile }">用户ID: {{ reply.user_id }}</span>
                  <span class="reply-user-id mobile-only" v-if="isMobile">{{ reply.user_id }}</span>
                </div>
                <span class="reply-time">{{ formatTime(reply.created_at) }}</span>
              </div>
              <div class="reply-content" :class="{ 'unread-content': reply.is_unread }">{{ reply.content }}</div>
            </div>
            <div v-if="!currentTicket.replies || currentTicket.replies.length === 0" class="empty-replies">
              <el-empty description="暂无回复" :image-size="80" />
            </div>
          </div>
        </el-card>

        <!-- 操作区域 -->
        <div class="ticket-actions" :class="{ 'mobile-card': isMobile }">
          <el-card shadow="never" v-if="!isMobile">
            <template #header>
              <span>操作</span>
            </template>
            <div class="action-buttons">
              <el-button @click="showStatusDialog = true">更新状态</el-button>
              <el-button @click="showAssignDialog = true">分配工单</el-button>
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
              @click="showAssignDialog = true" 
              class="mobile-action-btn"
            >
              分配工单
            </el-button>
            <el-button 
              @click="showNotesDialog = true" 
              class="mobile-action-btn"
            >
              添加备注
            </el-button>
          </div>
        </div>

        <!-- 回复区域 -->
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
              :style="isMobile ? { marginTop: '12px', width: '100%' } : { marginTop: '10px' }" 
              :loading="replying"
              :block="isMobile"
            >
              发送回复
            </el-button>
          </el-card>
        </div>
      </div>
    </el-dialog>

    <!-- 更新状态对话框 -->
    <el-dialog v-model="showStatusDialog" title="更新工单状态" :width="isMobile ? '90%' : '400px'">
      <el-select v-model="newStatus" placeholder="选择新状态" style="width: 100%">
        <el-option label="待处理" value="pending" />
        <el-option label="处理中" value="processing" />
        <el-option label="已解决" value="resolved" />
        <el-option label="已关闭" value="closed" />
      </el-select>
      <template #footer>
        <div class="dialog-footer-buttons">
          <el-button @click="showStatusDialog = false" class="mobile-action-btn">取消</el-button>
          <el-button type="primary" @click="updateStatus" class="mobile-action-btn">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 分配工单对话框 -->
    <el-dialog v-model="showAssignDialog" title="分配工单" :width="isMobile ? '90%' : '400px'">
      <el-input-number v-model="assignToUserId" placeholder="输入管理员用户ID" style="width: 100%" />
      <template #footer>
        <div class="dialog-footer-buttons">
          <el-button @click="showAssignDialog = false" class="mobile-action-btn">取消</el-button>
          <el-button type="primary" @click="assignTicketConfirm" class="mobile-action-btn">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 添加备注对话框 -->
    <el-dialog v-model="showNotesDialog" title="添加管理员备注" :width="isMobile ? '90%' : '500px'">
      <el-input
        v-model="adminNotes"
        type="textarea"
        :rows="5"
        placeholder="输入管理员备注..."
      />
      <template #footer>
        <div class="dialog-footer-buttons">
          <el-button @click="showNotesDialog = false" class="mobile-action-btn">取消</el-button>
          <el-button type="primary" @click="updateNotes" class="mobile-action-btn">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Search, Filter, User, ChatLineRound, Clock } from '@element-plus/icons-vue'
import { ticketAPI } from '@/utils/api'

const loading = ref(false)
const replying = ref(false)
const tickets = ref([])
const showDetailDialog = ref(false)
const showStatusDialog = ref(false)
const showAssignDialog = ref(false)
const showNotesDialog = ref(false)
const showFilterDrawer = ref(false)
const currentTicket = ref(null)
const replyContent = ref('')
const newStatus = ref('')
const assignToUserId = ref(null)
const adminNotes = ref('')
const statistics = ref(null)
const isMobile = ref(window.innerWidth <= 768)

const filters = reactive({
  keyword: '',
  status: '',
  type: '',
  priority: ''
})

const pagination = reactive({
  page: 1,
  size: 20,
  total: 0
})

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
    // 使用管理端API获取工单详情（需要管理员权限）
    const response = await ticketAPI.getAdminTicket(ticketId)
    if (response.data.success) {
      // 修复：后端返回的数据结构是 { ticket: {...} }，需要取 ticket 字段
      currentTicket.value = response.data.data?.ticket || response.data.data
      showDetailDialog.value = true
      // 查看工单后刷新列表，清除未读提醒
      setTimeout(async () => {
        await loadTickets()
        // 触发未读数量刷新
        window.dispatchEvent(new CustomEvent('ticket-viewed'))
      }, 500)
    }
  } catch (error) {
    ElMessage.error('加载工单详情失败: ' + (error.response?.data?.message || error.message))
  }
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
  
  console.log('[前端] 开始回复工单:', currentTicket.value.id, replyContent.value)
  replying.value = true
  try {
    const response = await ticketAPI.addReply(currentTicket.value.id, { content: replyContent.value })
    console.log('[前端] 回复响应:', response)
    if (response.data && response.data.success) {
      ElMessage.success('回复成功')
      replyContent.value = ''
      viewTicket(currentTicket.value.id)
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
  
  try {
    const response = await ticketAPI.updateTicket(currentTicket.value.id, { status: newStatus.value })
    if (response.data.success) {
      ElMessage.success('状态更新成功')
      showStatusDialog.value = false
      newStatus.value = ''
      viewTicket(currentTicket.value.id)
      loadTickets()
    }
  } catch (error) {
    ElMessage.error('更新状态失败')
  }
}

const assignTicket = (ticket) => {
  currentTicket.value = ticket
  assignToUserId.value = null
  showAssignDialog.value = true
}

const assignTicketConfirm = async () => {
  if (!currentTicket.value || !assignToUserId.value) {
    ElMessage.warning('请输入管理员用户ID')
    return
  }
  
  try {
    const response = await ticketAPI.updateTicket(currentTicket.value.id, { assigned_to: assignToUserId.value })
    if (response.data.success) {
      ElMessage.success('分配成功')
      showAssignDialog.value = false
      assignToUserId.value = null
      viewTicket(currentTicket.value.id)
      loadTickets()
    }
  } catch (error) {
    ElMessage.error('分配失败')
  }
}

const updateNotes = async () => {
  if (!currentTicket.value) return
  
  try {
    const response = await ticketAPI.updateTicket(currentTicket.value.id, { admin_notes: adminNotes.value })
    if (response.data.success) {
      ElMessage.success('备注添加成功')
      showNotesDialog.value = false
      adminNotes.value = ''
      viewTicket(currentTicket.value.id)
    }
  } catch (error) {
    ElMessage.error('添加备注失败')
  }
}

const closeDetailDialog = () => {
  currentTicket.value = null
  replyContent.value = ''
  newStatus.value = ''
  assignToUserId.value = null
  adminNotes.value = ''
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
  showFilterDrawer.value = false
  loadTickets()
}

const applyFilters = () => {
  showFilterDrawer.value = false
  loadTickets()
}

const handleResize = () => {
  isMobile.value = window.innerWidth <= 768
}

onMounted(() => {
  loadTickets()
  loadStatistics()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
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
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
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
  
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
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
        background: linear-gradient(135deg, #fff7e6 0%, #ffecc7 100%);
        border-left: 4px solid #faad14;
        box-shadow: 0 2px 8px rgba(250, 173, 20, 0.2);
        animation: highlightUnreadReply 0.5s ease;
      }
      
      &.user-reply.unread-reply {
        background: linear-gradient(135deg, #fff7e6 0%, #ffecc7 100%);
        border-left: 4px solid #faad14;
      }
      
      .unread-content {
        font-weight: 500;
        color: #1a1a1a;
      }
    }
    
    @keyframes highlightUnreadReply {
      0% {
        transform: scale(1);
        box-shadow: 0 2px 8px rgba(250, 173, 20, 0.2);
      }
      50% {
        transform: scale(1.02);
        box-shadow: 0 4px 16px rgba(250, 173, 20, 0.4);
      }
      100% {
        transform: scale(1);
        box-shadow: 0 2px 8px rgba(250, 173, 20, 0.2);
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
      padding: 20px;
      text-align: center;
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

/* 移动端详情页样式 */
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

.mobile-action-buttons {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 12px;
  width: 100%;
  box-sizing: border-box;
  
  .mobile-action-btn,
  .el-button {
    width: 100%;
    height: 44px;
    margin: 0;
    font-size: 16px;
    border-radius: 6px;
    font-weight: 500;
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

/* 移动端样式 */
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

  // mobile-action-bar 样式已统一在 list-common.scss 中定义
  // 这里不再重复定义，使用统一样式

  .filter-bar.desktop-only {
    display: none;
  }

  .mobile-tickets-list {
    margin-top: 16px;
    
    .mobile-ticket-card {
      background: #fff;
      border-radius: 8px;
      padding: 16px;
      margin-bottom: 12px;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      cursor: pointer;
      transition: all 0.3s;
      
      &:active {
        transform: scale(0.98);
        box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
      }
      
      .ticket-card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;
        
        .ticket-no {
          font-weight: bold;
          font-size: 14px;
          color: #333;
        }
        
        .ticket-badges {
          display: flex;
          gap: 6px;
        }
      }
      
      .ticket-card-title {
        font-size: 16px;
        font-weight: 500;
        color: #333;
        margin-bottom: 12px;
        line-height: 1.4;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
      }
      
      .ticket-card-info {
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
        margin-bottom: 12px;
        font-size: 12px;
        color: #666;
        
        .info-item {
          display: flex;
          align-items: center;
          gap: 4px;
        }
      }
      
      .ticket-card-actions {
        display: flex;
        gap: 8px;
        padding-top: 12px;
        border-top: 1px solid #f0f0f0;
        
        .el-button {
          flex: 1;
        }
      }
    }
    
    .empty-state {
      padding: 40px 20px;
      text-align: center;
    }
  }

  .filter-drawer-content {
    padding: 20px 0;
    
    .filter-drawer-actions {
      display: flex;
      gap: 12px;
      margin-top: 24px;
      padding-top: 20px;
      border-top: 1px solid #f0f0f0;
    }
  }

  .ticket-detail-dialog {
    :deep(.el-dialog) {
      margin: 0;
      height: 100vh;
      display: flex;
      flex-direction: column;
    }
    
    :deep(.el-dialog__header) {
      padding: 16px;
      border-bottom: 1px solid #f0f0f0;
      position: sticky;
      top: 0;
      background: #fff;
      z-index: 10;
    }
    
    :deep(.el-dialog__title) {
      font-size: 16px;
      font-weight: 600;
    }
    
    :deep(.el-dialog__body) {
      padding: 16px;
      flex: 1;
      overflow-y: auto;
      -webkit-overflow-scrolling: touch;
    }
    
    :deep(.el-dialog__close) {
      font-size: 20px;
    }
    
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

/* 桌面端隐藏移动端元素 */
.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}

/* 移动端隐藏桌面端元素 */
@media (min-width: 769px) {
  .mobile-action-bar,
  .mobile-tickets-list {
    display: none !important;
  }
}
</style>


