<template>
  <div class="list-container email-queue-admin">
    <el-row :gutter="20" class="stats-overview">
      <el-col :span="6">
        <el-card class="stat-card clickable" @click="filterByStatus('')">
          <div class="stat-content">
            <div class="stat-number">{{ statistics.total || 0 }}</div>
            <div class="stat-label">总邮件数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card clickable" @click="filterByStatus('pending')">
          <div class="stat-content">
            <div class="stat-number success">{{ statistics.pending || 0 }}</div>
            <div class="stat-label">待发送</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card clickable" @click="filterByStatus('sent')">
          <div class="stat-content">
            <div class="stat-number warning">{{ statistics.sent || 0 }}</div>
            <div class="stat-label">已发送</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card clickable" @click="filterByStatus('failed')">
          <div class="stat-content">
            <div class="stat-number danger">{{ statistics.failed || 0 }}</div>
            <div class="stat-label">发送失败</div>
          </div>
        </el-card>
      </el-col>
    </el-row>
    
    <div class="mobile-action-bar">
      <div class="mobile-search-section">
        <div class="search-input-wrapper">
          <el-input
            v-model="filterForm.email"
            placeholder="搜索邮箱地址"
            class="mobile-search-input"
            clearable
            @keyup.enter="applyFilter"
          />
          <el-button 
            @click="applyFilter" 
            class="search-button-inside"
            type="default"
            plain
          >
            <el-icon><Search /></el-icon>
          </el-button>
        </div>
      </div>
      <div class="mobile-filter-buttons">
        <el-select 
          v-model="filterForm.status" 
          placeholder="选择状态" 
          clearable
          class="mobile-status-select"
        >
          <el-option label="待发送" value="pending" />
          <el-option label="发送中" value="sending" />
          <el-option label="已发送" value="sent" />
          <el-option label="发送失败" value="failed" />
          <el-option label="已取消" value="cancelled" />
        </el-select>
        <el-button 
          @click="resetFilter" 
          type="default"
          plain
        >
          <el-icon><Refresh /></el-icon>
          重置
        </el-button>
      </div>
    </div>
    
    <el-card class="filter-section desktop-only">
      <el-form :inline="true" :model="filterForm" class="filter-form">
        <el-form-item label="状态">
          <el-select v-model="filterForm.status" placeholder="选择状态" clearable style="width: 180px">
            <el-option label="待发送" value="pending" />
            <el-option label="发送中" value="sending" />
            <el-option label="已发送" value="sent" />
            <el-option label="发送失败" value="failed" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="filterForm.email" placeholder="搜索邮箱地址" clearable />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="applyFilter">
            <el-icon><Search /></el-icon>
            筛选
          </el-button>
          <el-button @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>
    
    <el-card class="list-card queue-list">
      <template #header>
        <div class="card-header">
          <span>邮件队列列表</span>
          <div style="display: flex; align-items: center; gap: 10px;">
            <div class="header-info">
              共 {{ pagination.total }} 条记录，第 {{ pagination.page }}/{{ pagination.pages }} 页
            </div>
            <div class="header-actions">
              <el-button @click="refreshQueue" :loading="loading">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
              <el-button type="warning" @click="clearFailedEmails">
                <el-icon><Delete /></el-icon>
                清空失败邮件
              </el-button>
              <el-button type="danger" @click="clearAllEmails">
                <el-icon><Delete /></el-icon>
                清空所有邮件
              </el-button>
            </div>
          </div>
        </div>
      </template>
      
      <div class="table-wrapper desktop-only">
        <el-table :data="emailList" v-loading="loading" stripe empty-text="暂无数据">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="to_email" label="收件人" min-width="200" />
          <el-table-column prop="subject" label="主题" min-width="250" />
          <el-table-column prop="email_type" label="邮件类型" width="120" />
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusTagType(row.status)">
                {{ getStatusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="retry_count" label="重试次数" width="100">
            <template #default="{ row }">
              <span :class="{ 'text-danger': row.retry_count > 0 }">
                {{ row.retry_count }}/{{ row.max_retries || 3 }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="180">
            <template #default="{ row }">
              {{ formatDate(row.created_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <div class="action-buttons">
                <el-button size="small" @click="viewEmailDetail(row)">
                  <el-icon><View /></el-icon>
                  详情
                </el-button>
                <el-button 
                  v-if="row.status === 'failed'" 
                  size="small" 
                  type="warning" 
                  @click="retryEmail(row)"
                >
                  <el-icon><Refresh /></el-icon>
                  重试
                </el-button>
                <el-button 
                  size="small" 
                  type="danger" 
                  @click="deleteEmail(row)"
                >
                  <el-icon><Delete /></el-icon>
                  删除
                </el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
      
      <div class="mobile-card-list mobile-only" v-if="emailList.length > 0">
        <div 
          v-for="email in emailList" 
          :key="email.id"
          class="mobile-card"
        >
          <div class="card-row"><span class="label">ID</span><span class="value">#{{ email.id }}</span></div>
          <div class="card-row"><span class="label">收件人</span><span class="value">{{ email.to_email }}</span></div>
          <div class="card-row"><span class="label">主题</span><span class="value">{{ email.subject }}</span></div>
          <div class="card-row"><span class="label">邮件类型</span><span class="value">{{ email.email_type }}</span></div>
          <div class="card-row">
            <span class="label">状态</span>
            <span class="value">
              <el-tag :type="getStatusTagType(email.status)">
                {{ getStatusText(email.status) }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">重试次数</span>
            <span class="value" :class="{ 'text-danger': email.retry_count > 0 }">
              {{ email.retry_count }}/{{ email.max_retries || 3 }}
            </span>
          </div>
          <div class="card-row"><span class="label">创建时间</span><span class="value">{{ formatDate(email.created_at) }}</span></div>
          <div class="card-actions">
            <el-button size="small" @click="viewEmailDetail(email)">
              <el-icon><View /></el-icon>
              详情
            </el-button>
            <el-button 
              v-if="email.status === 'failed'" 
              size="small" 
              type="warning" 
              @click="retryEmail(email)"
            >
              <el-icon><Refresh /></el-icon>
              重试
            </el-button>
            <el-button 
              size="small" 
              type="danger" 
              @click="deleteEmail(email)"
            >
              <el-icon><Delete /></el-icon>
              删除
            </el-button>
          </div>
        </div>
      </div>
      
      <div class="mobile-card-list mobile-only" v-if="emailList.length === 0 && !loading">
        <div class="empty-state">
          <i class="el-icon-message"></i>
          <p>暂无邮件数据</p>
        </div>
      </div>
      
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.size"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
    
    <el-dialog
      v-model="detailDialogVisible" 
      title="邮件详情" 
      width="70%"
      :close-on-click-modal="false"
      :class="isMobile ? 'mobile-dialog' : ''"
    >
      <div v-if="emailDetail" class="email-detail" v-loading="detailLoading">
        <el-descriptions :column="isMobile ? 1 : 2" border class="desktop-only">
          <el-descriptions-item label="邮件ID">{{ emailDetail.id }}</el-descriptions-item>
          <el-descriptions-item label="收件人">{{ emailDetail.to_email }}</el-descriptions-item>
          <el-descriptions-item label="主题">{{ emailDetail.subject }}</el-descriptions-item>
          <el-descriptions-item label="邮件类型">{{ emailDetail.email_type }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusTagType(emailDetail.status)">
              {{ getStatusText(emailDetail.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="优先级">
            <span>{{ emailDetail.priority || 'N/A' }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="重试次数">{{ emailDetail.retry_count }}/{{ emailDetail.max_retries }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDate(emailDetail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="发送时间" v-if="emailDetail.sent_at">
            {{ formatDate(emailDetail.sent_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="处理时间" v-if="emailDetail.processing_time">
            {{ emailDetail.processing_time }}ms
          </el-descriptions-item>
        </el-descriptions>
        
        <div class="mobile-detail-info mobile-only">
          <div class="detail-info-row"><span class="detail-label">邮件ID</span><span class="detail-value">#{{ emailDetail.id }}</span></div>
          <div class="detail-info-row"><span class="detail-label">收件人</span><span class="detail-value">{{ emailDetail.to_email }}</span></div>
          <div class="detail-info-row"><span class="detail-label">主题</span><span class="detail-value">{{ emailDetail.subject }}</span></div>
          <div class="detail-info-row"><span class="detail-label">邮件类型</span><span class="detail-value">{{ emailDetail.email_type }}</span></div>
          <div class="detail-info-row">
            <span class="detail-label">状态</span>
            <span class="detail-value">
              <el-tag :type="getStatusTagType(emailDetail.status)">
                {{ getStatusText(emailDetail.status) }}
              </el-tag>
            </span>
          </div>
          <div class="detail-info-row" v-if="emailDetail.priority"><span class="detail-label">优先级</span><span class="detail-value">{{ emailDetail.priority }}</span></div>
          <div class="detail-info-row"><span class="detail-label">重试次数</span><span class="detail-value">{{ emailDetail.retry_count }}/{{ emailDetail.max_retries }}</span></div>
          <div class="detail-info-row"><span class="detail-label">创建时间</span><span class="detail-value">{{ formatDate(emailDetail.created_at) }}</span></div>
          <div class="detail-info-row" v-if="emailDetail.sent_at"><span class="detail-label">发送时间</span><span class="detail-value">{{ formatDate(emailDetail.sent_at) }}</span></div>
          <div class="detail-info-row" v-if="emailDetail.processing_time"><span class="detail-label">处理时间</span><span class="detail-value">{{ emailDetail.processing_time }}ms</span></div>
        </div>
        
        <div class="detail-section">
          <h4>邮件内容</h4>
          <div v-if="emailDetail.content_type === 'html'" class="email-content-html">
            <div v-if="!emailDetail.content || !sanitizedEmailContent" class="email-content-empty">
              <el-empty description="邮件内容为空" />
            </div>
            <iframe 
              v-else-if="isEmailFullHtml && sanitizedEmailContent"
              :srcdoc="sanitizedEmailContent" 
              class="email-html-iframe"
              frameborder="0"
              sandbox="allow-same-origin"
              @load="onIframeLoad"
            ></iframe>
            <div v-else-if="sanitizedEmailContent" v-html="sanitizedEmailContent" class="email-html-content"></div>
          </div>
          <div v-else class="email-content-text">
            <el-input
              v-model="emailDetail.content"
              type="textarea"
              :rows="isMobile ? 8 : 10"
              readonly
              class="email-text-content"
            />
          </div>
        </div>
        
        <div class="detail-section" v-if="emailDetail.template_data">
          <h4>模板数据</h4>
          <el-input
            v-model="emailDetail.template_data"
            type="textarea"
            :rows="isMobile ? 6 : 8"
            readonly
            class="template-data-content"
          />
        </div>
        
        <div class="detail-section" v-if="emailDetail.error_message">
          <h4>错误信息</h4>
          <el-alert
            :title="emailDetail.error_message"
            type="error"
            :description="emailDetail.error_details || '无详细错误信息'"
            show-icon
            :closable="false"
            class="error-alert"
          />
        </div>
        
        <div class="detail-section" v-if="emailDetail.smtp_response">
          <h4>SMTP响应</h4>
          <el-input
            v-model="emailDetail.smtp_response"
            type="textarea"
            :rows="isMobile ? 4 : 6"
            readonly
            class="smtp-response-content"
          />
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer-buttons">
          <el-button @click="detailDialogVisible = false" class="mobile-action-btn">关闭</el-button>
          <el-button 
            v-if="emailDetail && emailDetail.status === 'failed'" 
            type="warning" 
            @click="retryEmailFromDetail"
            class="mobile-action-btn"
          >
            重试发送
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, onMounted, computed, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Search, View, Delete } from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
import { formatDateTime } from '@/utils/date'
import DOMPurify from 'dompurify'

// 状态映射常量，避免重复定义
const STATUS_MAP = {
  pending: { tag: 'warning', text: '待发送' },
  sending: { tag: 'info', text: '发送中' },
  sent: { tag: 'success', text: '已发送' },
  failed: { tag: 'danger', text: '发送失败' },
  cancelled: { tag: 'info', text: '已取消' }
}

// 统一处理 API 响应的函数
const handleResponse = (response, defaultErrorMsg) => {
  if (!response) {
    return { success: false, message: defaultErrorMsg || '请求失败' }
  }
  
  const data = response.data || response
  if (data?.success || (response && response.status >= 200 && response.status < 300)) {
    return { success: true, data: data?.data || data }
  } else {
    return { 
      success: false, 
      message: data?.message || defaultErrorMsg || '操作失败'
    }
  }
}

export default {
  name: 'EmailQueue',
  components: {
    Refresh, Search, View, Delete
  },
  setup() {
    const loading = ref(false)
    const detailDialogVisible = ref(false)
    const emailDetail = ref(null)
    const isMobile = ref(false)
    const detailLoading = ref(false)
    
    // 检查是否为移动设备
    const checkMobile = () => {
      isMobile.value = window.innerWidth <= 768
    }
    
    // 筛选表单
    const filterForm = reactive({
      status: '',
      email: ''
    })
    
    // 分页信息
    const pagination = reactive({
      page: 1,
      size: 20,
      total: 0,
      pages: 0
    })
    
    const emailList = ref([])
    
    // 统计数据
    const statistics = reactive({
      total: 0,
      pending: 0,
      sent: 0,
      failed: 0
    })

    // HTML内容清理缓存
    const sanitizeCache = new Map()
    let iframeLoadTimeout = null

    // 检查是否是完整的 HTML 文档
    const isFullHtmlDocument = (html) => {
      if (!html) return false
      const htmlLower = html.toLowerCase().trim()
      return htmlLower.includes('<!doctype html>') || 
             (htmlLower.includes('<html') && htmlLower.includes('<head') && htmlLower.includes('<body'))
    }

    // HTML内容清理函数，防止XSS攻击
    const sanitizeHtml = (html) => {
      if (!html || typeof html !== 'string') return String(html || '')
      
      const cacheKey = html.substring(0, 100) + html.length
      if (sanitizeCache.has(cacheKey) && sanitizeCache.get(cacheKey).original === html) {
        return sanitizeCache.get(cacheKey).sanitized
      }
      
      try {
        const sanitized = DOMPurify.sanitize(html, {
          ALLOWED_TAGS: null, 
          ALLOWED_ATTR: null, 
          ALLOW_DATA_ATTR: true,
          ALLOW_UNKNOWN_PROTOCOLS: true,
          KEEP_CONTENT: true,
          SAFE_FOR_TEMPLATES: true,
          FORBID_TAGS: ['script'], 
          FORBID_ATTR: ['onerror', 'onload', 'onclick', 'onmouseover', 'onmouseout', 'onfocus', 'onblur', 'onchange', 'onsubmit'], 
          RETURN_DOM: false,
          RETURN_DOM_FRAGMENT: false,
          USE_PROFILES: { html: true }
        })
        
        if (!sanitized || sanitized.trim() === '') return html
        
        if (sanitizeCache.size > 10) {
          sanitizeCache.delete(sanitizeCache.keys().next().value)
        }
        sanitizeCache.set(cacheKey, { original: html, sanitized })
        
        return sanitized
      } catch (error) {
        console.error('sanitizeHtml 错误:', error)
        return html
      }
    }
    
    // 使用 computed 缓存邮件内容
    const sanitizedEmailContent = computed(() => {
      return sanitizeHtml(emailDetail.value?.content)
    })
    
    const isEmailFullHtml = computed(() => {
      return isFullHtmlDocument(emailDetail.value?.content)
    })
    
    // iframe 加载完成后处理
    const onIframeLoad = (event) => {
      if (iframeLoadTimeout) clearTimeout(iframeLoadTimeout)
      
      iframeLoadTimeout = setTimeout(() => {
        try {
          const iframe = event.target
          if (!iframe || !iframe.contentDocument) return
          
          const doc = iframe.contentDocument
          const body = doc.body
          const html = doc.documentElement
          
          if (html && !html.style.backgroundColor) html.style.backgroundColor = '#f4f4f4'
          
          if (body) {
            if (!body.style.display) {
              body.style.display = 'flex'
              body.style.justifyContent = 'center'
              body.style.alignItems = 'flex-start'
            }
            if (!body.style.margin) body.style.margin = '0'
            if (!body.style.padding) body.style.padding = '20px'
            if (!body.style.backgroundColor) body.style.backgroundColor = '#f4f4f4'
            body.style.minHeight = '100vh'
            body.style.boxSizing = 'border-box'
          }
          
          setTimeout(() => {
            try {
              const scrollHeight = Math.max(
                doc.body?.scrollHeight || 0,
                doc.body?.offsetHeight || 0,
                doc.documentElement?.clientHeight || 0,
                doc.documentElement?.scrollHeight || 0,
                doc.documentElement?.offsetHeight || 0
              )
              if (scrollHeight > 0) {
                iframe.style.height = Math.min(scrollHeight + 40, 1200) + 'px'
              }
            } catch (e) {
              console.debug('无法计算 iframe 高度:', e)
            }
          }, 200)
        } catch (e) {
          console.debug('无法访问 iframe 内容:', e)
        }
      }, 50)
    }

    // 获取邮件队列列表
    const fetchEmailQueue = async () => {
      loading.value = true
      try {
        const params = { page: pagination.page, size: pagination.size, ...filterForm }
        const response = await adminAPI.getEmailQueue(params)
        const result = handleResponse(response, '获取邮件队列失败')
        
        if (result.success) {
          emailList.value = result.data.emails
          pagination.total = result.data.total
          pagination.pages = result.data.pages
        } else {
          ElMessage.error(result.message)
        }
      } catch (error) {
        ElMessage.error('获取邮件队列失败: ' + (error.response?.data?.message || error.message))
      } finally {
        loading.value = false
      }
    }
    
    // 获取统计数据
    const fetchStatistics = async () => {
      try {
        const response = await adminAPI.getEmailQueueStatistics()
        const result = handleResponse(response)
        
        if (result.success) {
          Object.assign(statistics, result.data)
        }
      } catch (error) {
        console.error('获取统计数据失败:', error)
      }
    }
    
    const refreshQueue = () => {
      fetchEmailQueue()
      fetchStatistics()
    }
    
    const applyFilter = () => {
      pagination.page = 1
      fetchEmailQueue()
      fetchStatistics() // 筛选时也更新统计数据
    }
    
    const resetFilter = () => {
      Object.assign(filterForm, { status: '', email: '' })
      pagination.page = 1
      fetchEmailQueue()
      fetchStatistics() // 重置时也更新统计数据
    }
    
    const filterByStatus = (status) => {
      filterForm.status = status
      pagination.page = 1
      fetchEmailQueue()
      fetchStatistics() // 按状态筛选时也更新统计数据
    }
    
    const viewEmailDetail = async (row) => {
      detailLoading.value = true
      emailDetail.value = null // 清空旧数据
      try {
        const response = await adminAPI.getEmailDetail(row.id)
        const result = handleResponse(response, '获取邮件详情失败')
        
        if (result.success) {
          emailDetail.value = result.data
          await nextTick()
          detailDialogVisible.value = true
        } else {
          ElMessage.error(result.message)
        }
      } catch (error) {
        ElMessage.error('获取邮件详情失败: ' + error.message)
      } finally {
        detailLoading.value = false
      }
    }
    
    const retryEmail = async (row) => {
      try {
        await ElMessageBox.confirm(`确定要重试发送邮件到 ${row.to_email} 吗？`, '确认重试', { type: 'warning' })
        
        const response = await adminAPI.retryEmail(row.id)
        const result = handleResponse(response, '邮件重试失败')
        
        if (result.success) {
          ElMessage.success('邮件重试成功')
          refreshQueue()
        } else {
          ElMessage.error(result.message)
        }
      } catch (error) {
        if (error !== 'cancel') ElMessage.error('邮件重试失败')
      }
    }
    
    const retryEmailFromDetail = async () => {
      if (emailDetail.value) {
        await retryEmail(emailDetail.value)
        detailDialogVisible.value = false
      }
    }
    
    const deleteEmail = async (row) => {
      try {
        await ElMessageBox.confirm(`确定要删除发送到 ${row.to_email} 的邮件吗？`, '确认删除', { type: 'warning' })
        
        const response = await adminAPI.deleteEmailFromQueue(row.id)
        const result = handleResponse(response, '邮件删除失败')
        
        if (result.success) {
          ElMessage.success('邮件删除成功')
          refreshQueue()
        } else {
          ElMessage.error(result.message)
        }
      } catch (error) {
        if (error !== 'cancel') ElMessage.error('邮件删除失败')
      }
    }
    
    // 清空邮件队列的通用函数
    const clearEmails = async (status, title, confirmText, type) => {
      try {
        await ElMessageBox.confirm(confirmText, title, { type })
        
        const response = await adminAPI.clearEmailQueue(status)
        const result = handleResponse(response, `${title}失败`)
        
        if (result.success) {
          ElMessage.success(`${title}成功`)
          refreshQueue()
        } else {
          ElMessage.error(result.message)
        }
      } catch (error) {
        if (error !== 'cancel') {
          const errorMsg = error.response?.data?.message || error.message || `${title}失败`
          ElMessage.error(errorMsg)
        }
      }
    }

    const clearFailedEmails = () => {
      clearEmails('failed', '确认清空失败邮件', '确定要清空所有失败的邮件吗？', 'warning')
    }
    
    const clearAllEmails = () => {
      clearEmails('', '确认清空所有邮件', '确定要清空所有邮件吗？此操作不可恢复！', 'error')
    }
    
    const handleSizeChange = (size) => {
      pagination.size = size
      pagination.page = 1
      fetchEmailQueue()
    }

    const handleCurrentChange = (page) => {
      pagination.page = page
      fetchEmailQueue()
    }
    
    // 根据状态获取 Tag 类型
    const getStatusTagType = (status) => {
      return STATUS_MAP[status]?.tag || 'info'
    }

    // 根据状态获取中文文本
    const getStatusText = (status) => {
      return STATUS_MAP[status]?.text || status
    }

    // 格式化日期
    const formatDate = (dateString) => {
      if (!dateString) return '-'
      return formatDateTime(dateString, 'YYYY-MM-DD HH:mm:ss')
    }

    onMounted(() => {
      checkMobile()
      window.addEventListener('resize', checkMobile)
      refreshQueue()
    })

    onUnmounted(() => {
      window.removeEventListener('resize', checkMobile)
    })

    return {
      loading,
      detailDialogVisible,
      detailLoading,
      emailDetail,
      sanitizedEmailContent,
      isEmailFullHtml,
      filterForm,
      pagination,
      emailList,
      statistics,
      isMobile,
      onIframeLoad,
      refreshQueue,
      applyFilter,
      resetFilter,
      filterByStatus,
      viewEmailDetail,
      retryEmail,
      retryEmailFromDetail,
      deleteEmail,
      clearFailedEmails,
      clearAllEmails,
      handleSizeChange,
      handleCurrentChange,
      getStatusTagType,
      getStatusText,
      formatDate
    }
  }
}
</script>

<style scoped lang="scss">
@use '@/styles/list-common.scss';

.empty-state {
  text-align: center;
  padding: 3rem 1rem;
  color: #999;
  
  :is(i) {
    font-size: 3rem;
    margin-bottom: 1rem;
    display: block;
  }
  
  :is(p) {
    font-size: 0.9rem;
    margin: 0;
    line-height: 1.5;
  }
}

.stats-overview {
  margin-bottom: 20px;
}

.stat-card.clickable {
  cursor: pointer;
  transition: all 0.3s ease;
  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  }
}

.stat-content {
  padding: 20px;
}

.stat-number {
  font-size: 2rem;
  font-weight: bold;
  color: #333;
  margin-bottom: 10px;
  &.success { color: #67c23a; }
  &.warning { color: #e6a23c; }
  &.danger { color: #f56c6c; }
}

.stat-label {
  color: #666;
  font-size: 0.9rem;
}

.filter-section {
  margin-bottom: 20px;
}

.filter-form {
  display: flex;
  flex-wrap: wrap;
  gap: 15px;
}

.queue-list {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  
  @media (max-width: 768px) {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }
}

.header-info {
  color: #666;
  font-size: 0.9rem;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.email-detail {
  max-height: 60vh;
  overflow-y: auto;
}

.detail-section {
  margin-top: 20px;
  
  :is(h4) {
    margin-bottom: 10px;
    color: #333;
    font-size: 1rem;
  }
}

.email-content-html {
  border: 1px solid #ddd;
  border-radius: 4px;
  max-height: 800px;
  overflow-y: auto;
  overflow-x: hidden;
  background-color: #f4f4f4;
  padding: 0;
  position: relative;
}

.email-html-iframe {
  width: 100%;
  min-height: 400px;
  max-height: 1000px;
  border: none;
  background-color: #f4f4f4;
  border-radius: 4px;
  display: block;
  overflow-y: auto;
  overflow-x: hidden;
  -webkit-overflow-scrolling: touch;
}

.email-html-content {
  width: 100%;
  min-height: 200px;
  padding: 20px;
  background-color: #f4f4f4;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  
  :deep(html), :deep(body) {
    margin: 0 !important;
    padding: 0 !important;
    width: 100% !important;
    height: auto !important;
    background-color: #f4f4f4 !important;
  }
  
  :deep(body) {
    display: flex !important;
    justify-content: center !important;
    align-items: flex-start !important;
    min-height: 100vh !important;
  }
  
  :deep(.email-container) {
    max-width: 600px !important;
    width: 100% !important;
    margin: 0 auto !important;
    background-color: #ffffff !important;
    box-shadow: 0 4px 12px rgba(0,0,0,0.1) !important;
    flex-shrink: 0 !important;
  }
  
  :deep(style) {
    display: block !important;
  }
}

.email-text-content,
.template-data-content,
.smtp-response-content {
  width: 100%;
}

.email-content-empty {
  padding: 40px;
  text-align: center;
  color: #909399;
}

.text-danger {
  color: #f56c6c;
}

/* 桌面端/移动端显示控制 */
.desktop-only {
  @media (max-width: 768px) { display: none !important; }
}

.mobile-only {
  display: none;
  @media (max-width: 768px) { display: block; }
  
  &.mobile-card-list {
    @media (max-width: 768px) {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
  }
}

/* 移动端详情信息 */
.mobile-detail-info {
  display: none;
  @media (max-width: 768px) {
    display: block;
    margin-bottom: 20px;
  }
  
  .detail-info-row {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    padding: 12px 0;
    border-bottom: 1px solid #f0f0f0;
    
    &:last-child { border-bottom: none; }
    
    .detail-label {
      font-weight: 600;
      color: #606266;
      font-size: 14px;
      min-width: 100px;
      flex-shrink: 0;
    }
    
    .detail-value {
      flex: 1;
      text-align: right;
      color: #303133;
      font-size: 14px;
      word-break: break-all;
    }
  }
}

/* 移动端卡片列表样式 */
.mobile-card-list {
  .mobile-card {
    background: #fff;
    border: 1px solid #e4e7ed;
    border-radius: 8px;
    padding: 16px;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  }
  
  .card-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 0;
    border-bottom: 1px solid #f0f0f0;
    &:last-of-type { border-bottom: none; }
    
    .label {
      font-weight: 600;
      color: #606266;
      font-size: 14px;
      min-width: 100px;
      flex-shrink: 0;
    }
    
    .value {
      flex: 1;
      text-align: right;
      color: #303133;
      font-size: 14px;
      word-break: break-all;
    }
  }
  
  .card-actions {
    display: flex;
    flex-direction: column;
    gap: 12px;
    width: 100%;
    margin-top: 12px;
  }
}

/* 移动端全局优化 */
@media (max-width: 768px) {
  .email-queue-admin { padding: 10px; }
  
  .header-actions, .action-buttons, .dialog-footer-buttons {
    display: flex;
    flex-direction: column;
    gap: 12px;
    width: 100%;
  }

  /* 统一所有移动端按钮样式 */
  .email-queue-admin .el-button:not(.search-button-inside) {
    width: 100% !important;
    min-width: 100% !important;
    max-width: 100% !important;
    height: 44px !important;
    font-size: 16px !important;
    font-weight: 500 !important;
    margin: 0 !important;
    border-radius: 6px !important;
    padding: 0 16px !important;
    
    :deep(.el-icon) {
      margin-right: 6px;
      font-size: 16px;
    }
  }
  
  .filter-form {
    flex-direction: column;
    align-items: stretch;
    .el-form-item {
      margin-bottom: 10px;
      width: 100% !important;
      
      .el-button + .el-button { margin-top: 12px !important; }
    }
  }
  
  /* 移动端对话框优化 */
  .mobile-dialog {
    :deep(.el-dialog) {
      width: 95% !important;
      margin: 5vh auto !important;
      max-height: 90vh;
      border-radius: 12px;
    }
    :deep(.el-dialog__body) {
      padding: 16px;
      max-height: calc(90vh - 140px);
      overflow-y: auto;
    }
  }
}

/* 移除所有输入框的圆角和阴影效果 */
:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
  &:hover { border-color: #c0c4cc !important; }
  &.is-focus { border-color: #1677ff !important; }
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
</style>