<template>
  <div class="email-detail-admin">
    <div class="page-header">
      <div class="header-left">
        <el-button @click="$router.go(-1)" icon="ArrowLeft">
          返回
        </el-button>
        <h1>邮件详情</h1>
      </div>
      <div class="header-actions">
        <el-button 
          v-if="emailDetail && emailDetail.status === 'failed'" 
          type="warning" 
          @click="retryEmail"
          :loading="retryLoading"
        >
          <el-icon><Refresh /></el-icon>
          重试发送
        </el-button>
        <el-button type="danger" @click="deleteEmail">
          <el-icon><Delete /></el-icon>
          删除邮件
        </el-button>
      </div>
    </div>

    <div v-if="loading" class="loading-container">
      <el-skeleton :rows="10" animated />
    </div>

    <div v-else-if="emailDetail" class="email-detail-content">
      <el-card class="detail-card">
        <template #header>
          <div class="card-header">
            <span>基本信息</span>
            <el-tag :type="getStatusTagType(emailDetail.status)">
              {{ getStatusText(emailDetail.status) }}
            </el-tag>
          </div>
        </template>
        
        <el-descriptions :column="2" border>
          <el-descriptions-item label="邮件ID">{{ emailDetail.id }}</el-descriptions-item>
          <el-descriptions-item label="收件人">{{ emailDetail.to_email }}</el-descriptions-item>
          <el-descriptions-item label="邮件主题">{{ emailDetail.subject }}</el-descriptions-item>
          <el-descriptions-item label="模板名称">{{ emailDetail.template_name }}</el-descriptions-item>
          <el-descriptions-item label="优先级">
            <el-tag :type="getPriorityTagType(emailDetail.priority)" size="small">
              {{ getPriorityText(emailDetail.priority) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="重试次数">
            <span :class="{ 'text-danger': emailDetail.retry_count > 0 }">
              {{ emailDetail.retry_count }}/{{ emailDetail.max_retries || 3 }}
            </span>
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDate(emailDetail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="发送时间" v-if="emailDetail.sent_at">
            {{ formatDate(emailDetail.sent_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="处理时间" v-if="emailDetail.processing_time">
            {{ emailDetail.processing_time }}ms
          </el-descriptions-item>
        </el-descriptions>
      </el-card>
      <el-card class="detail-card" v-if="emailDetail.template_data">
        <template #header>
          <span>模板数据</span>
        </template>
        
        <div class="template-data">
          <el-input
            v-model="formattedTemplateData"
            type="textarea"
            :rows="8"
            readonly
            class="template-data-input"
          />
          <div class="template-data-actions">
            <el-button size="small" @click="copyTemplateData">
              <el-icon><CopyDocument /></el-icon>
              复制数据
            </el-button>
            <el-button size="small" @click="downloadTemplateData">
              <el-icon><Download /></el-icon>
              下载JSON
            </el-button>
          </div>
        </div>
      </el-card>
      <el-card class="detail-card" v-if="emailDetail.error_message">
        <template #header>
          <span>错误信息</span>
        </template>
        
        <el-alert
          :title="emailDetail.error_message"
          type="error"
          :description="emailDetail.error_details || '无详细错误信息'"
          show-icon
          :closable="false"
          class="error-alert"
        />
        
        <div v-if="emailDetail.error_details" class="error-details">
          <h4>详细错误信息：</h4>
          <el-input
            v-model="emailDetail.error_details"
            type="textarea"
            :rows="4"
            readonly
          />
        </div>
      </el-card>
      <el-card class="detail-card" v-if="emailDetail.smtp_response">
        <template #header>
          <span>SMTP响应</span>
        </template>
        
        <div class="smtp-response">
          <el-input
            v-model="emailDetail.smtp_response"
            type="textarea"
            :rows="4"
            readonly
            placeholder="SMTP服务器响应信息"
          />
        </div>
      </el-card>
      <el-card class="detail-card" v-if="emailDetail.content">
        <template #header>
          <span>邮件内容</span>
        </template>
        
        <div class="email-content-preview">
          <el-tabs v-model="contentViewMode">
            <el-tab-pane label="HTML预览" name="preview">
              <div 
                class="email-html-preview" 
                v-html="sanitizedEmailContent"
              ></div>
            </el-tab-pane>
            <el-tab-pane label="HTML源码" name="source">
              <el-input
                v-model="emailDetail.content"
                type="textarea"
                :rows="20"
                readonly
                class="email-source-code"
              />
            </el-tab-pane>
          </el-tabs>
        </div>
      </el-card>
      <el-card class="detail-card">
        <template #header>
          <span>发送历史</span>
        </template>
        
        <el-timeline>
          <el-timeline-item
            :timestamp="formatDate(emailDetail.created_at)"
            placement="top"
            type="primary"
          >
            <h4>邮件创建</h4>
            <p>邮件已添加到发送队列</p>
          </el-timeline-item>
          
          <el-timeline-item
            v-if="emailDetail.status === 'sending'"
            :timestamp="formatDate(emailDetail.updated_at)"
            placement="top"
            type="warning"
          >
            <h4>发送中</h4>
            <p>正在尝试发送邮件</p>
          </el-timeline-item>
          
          <el-timeline-item
            v-if="emailDetail.status === 'sent'"
            :timestamp="formatDate(emailDetail.sent_at)"
            placement="top"
            type="success"
          >
            <h4>发送成功</h4>
            <p>邮件已成功发送到收件人</p>
          </el-timeline-item>
          
          <el-timeline-item
            v-if="emailDetail.status === 'failed'"
            :timestamp="formatDate(emailDetail.updated_at)"
            placement="top"
            type="danger"
          >
            <h4>发送失败</h4>
            <p>邮件发送失败，错误：{{ emailDetail.error_message }}</p>
          </el-timeline-item>
          
          <el-timeline-item
            v-if="emailDetail.retry_count > 0"
            :timestamp="formatDate(emailDetail.updated_at)"
            placement="top"
            type="info"
          >
            <h4>重试发送</h4>
            <p>第 {{ emailDetail.retry_count }} 次重试发送</p>
          </el-timeline-item>
        </el-timeline>
      </el-card>
    </div>

    <div v-else class="error-container">
      <el-empty description="邮件不存在或已被删除">
        <el-button type="primary" @click="$router.go(-1)">返回</el-button>
      </el-empty>
    </div>
  </div>
</template>

<script>
import { ref, reactive, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Refresh, Delete, CopyDocument, Download } from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
import DOMPurify from 'dompurify'

export default {
  name: 'EmailDetail',
  components: {
    ArrowLeft, Refresh, Delete, CopyDocument, Download
  },
  setup() {
    const route = useRoute()
    const router = useRouter()
    
    const loading = ref(false)
    const retryLoading = ref(false)
    const emailDetail = ref(null)
    const contentViewMode = ref('preview')
    
    // 获取邮件详情
    const fetchEmailDetail = async () => {
      const emailId = route.params.id
      if (!emailId) {
        ElMessage.error('邮件ID不能为空')
        return
      }
      
      loading.value = true
      try {
        const response = await adminAPI.getEmailDetail(emailId)
        if (response.success) {
          emailDetail.value = response.data
        } else {
          ElMessage.error('获取邮件详情失败')
        }
      } catch (error) {
        ElMessage.error('获取邮件详情失败')
      } finally {
        loading.value = false
      }
    }
    const retryEmail = async () => {
      if (!emailDetail.value) return
      
      try {
        await ElMessageBox.confirm(
          `确定要重试发送邮件到 ${emailDetail.value.to_email} 吗？`,
          '确认重试',
          { type: 'warning' }
        )
        
        retryLoading.value = true
        const response = await adminAPI.retryEmail(emailDetail.value.id)
        
        if (response.success) {
          ElMessage.success('邮件重试成功')
          // 重新获取邮件详情
          await fetchEmailDetail()
        } else {
          ElMessage.error('邮件重试失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('邮件重试失败')
        }
      } finally {
        retryLoading.value = false
      }
    }
    const deleteEmail = async () => {
      if (!emailDetail.value) return
      
      try {
        await ElMessageBox.confirm(
          `确定要删除发送到 ${emailDetail.value.to_email} 的邮件吗？`,
          '确认删除',
          { type: 'warning' }
        )
        
        const response = await adminAPI.deleteEmailFromQueue(emailDetail.value.id)
        
        if (response.success) {
          ElMessage.success('邮件删除成功')
          router.go(-1)
        } else {
          ElMessage.error('邮件删除失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('邮件删除失败')
        }
      }
    }
    
    // 复制模板数据
    const copyTemplateData = async () => {
      try {
        await navigator.clipboard.writeText(formattedTemplateData.value)
        ElMessage.success('模板数据已复制到剪贴板')
      } catch (error) {
        ElMessage.error('复制失败')
      }
    }
    const downloadTemplateData = () => {
      const blob = new Blob([formattedTemplateData.value], { type: 'application/json' })
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `email-template-data-${emailDetail.value.id}.json`
      a.click()
      window.URL.revokeObjectURL(url)
      ElMessage.success('模板数据下载成功')
    }
    
    // HTML内容清理函数，防止XSS攻击
    const sanitizeHtml = (html) => {
      if (!html) return ''
      try {
        return DOMPurify.sanitize(html, {
          ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'b', 'i', 'u', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li', 'a', 'div', 'span', 'blockquote', 'pre', 'code', 'table', 'thead', 'tbody', 'tr', 'th', 'td', 'img'],
          ALLOWED_ATTR: ['href', 'target', 'style', 'class', 'id', 'src', 'alt', 'width', 'height'],
          ALLOW_DATA_ATTR: false
        })
      } catch (error) {
        console.error('sanitizeHtml 错误:', error)
        return html
      }
    }
    
    // 清理后的邮件内容（计算属性）
    const sanitizedEmailContent = computed(() => {
      if (!emailDetail.value?.content) return ''
      return sanitizeHtml(emailDetail.value.content)
    })
    
    // 格式化模板数据
    const formattedTemplateData = computed(() => {
      if (!emailDetail.value?.template_data) return ''
      try {
        const data = typeof emailDetail.value.template_data === 'string' 
          ? JSON.parse(emailDetail.value.template_data)
          : emailDetail.value.template_data
        return JSON.stringify(data, null, 2)
      } catch {
        return emailDetail.value.template_data
      }
    })
    const getStatusTagType = (status) => {
      const statusMap = {
        pending: 'warning',
        sending: 'info',
        sent: 'success',
        failed: 'danger',
        cancelled: 'info'
      }
      return statusMap[status] || 'info'
    }
    
    // 状态文本
    const getStatusText = (status) => {
      const statusMap = {
        pending: '待发送',
        sending: '发送中',
        sent: '已发送',
        failed: '发送失败',
        cancelled: '已取消'
      }
      return statusMap[status] || status
    }
    
    // 优先级标签类型
    const getPriorityTagType = (priority) => {
      const priorityMap = {
        high: 'danger',
        medium: 'warning',
        low: 'info'
      }
      return priorityMap[priority] || 'info'
    }
    
    // 优先级文本
    const getPriorityText = (priority) => {
      const priorityMap = {
        high: '高',
        medium: '中',
        low: '低'
      }
      return priorityMap[priority] || priority
    }
    
    // 格式化日期
    const formatDate = (dateString) => {
      if (!dateString) return '-'
      return new Date(dateString).toLocaleString('zh-CN')
    }
    
    onMounted(() => {
      fetchEmailDetail()
    })
    
    return {
      loading,
      retryLoading,
      emailDetail,
      contentViewMode,
      sanitizedEmailContent,
      fetchEmailDetail,
      retryEmail,
      deleteEmail,
      copyTemplateData,
      downloadTemplateData,
      formattedTemplateData,
      getStatusTagType,
      getStatusText,
      getPriorityTagType,
      getPriorityText,
      formatDate
    }
  }
}
</script>

<style scoped>
.email-detail-admin {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 15px;
}

.header-left h1 {
  margin: 0;
  color: #333;
  font-size: 1.8rem;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.loading-container {
  padding: 40px;
}

.email-detail-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.detail-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.template-data {
  position: relative;
}

.template-data-input {
  margin-bottom: 15px;
}

.template-data-actions {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
}

.error-alert {
  margin-bottom: 20px;
}

.error-details :is(h4) {
  margin: 20px 0 10px 0;
  color: #333;
  font-size: 1rem;
}

.smtp-response {
  padding: 10px 0;
}

.text-danger {
  color: #f56c6c;
}

.error-container {
  padding: 60px 20px;
  text-align: center;
}

@media (max-width: 768px) {
  .email-detail-admin {
    padding: 10px;
  }
  
  .page-header {
    flex-direction: column;
    gap: 15px;
    align-items: flex-start;
  }
  
  .header-actions {
    width: 100%;
    justify-content: space-between;
  }
  
  .template-data-actions {
    flex-direction: column;
  }
}

/* 移除所有输入框的圆角和阴影效果，设置为简单长方形 */
:deep(.el-input__wrapper) {
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
  border-color: #1677ff !important;
  box-shadow: none !important;
}

.email-content-preview {
  margin-top: 10px;
}

.email-html-preview {
  min-height: 400px;
  padding: 20px;
  background: #ffffff;
  border: 1px solid #e9ecef;
  border-radius: 4px;
  overflow: auto;
}

.email-html-preview :deep(img) {
  max-width: 100%;
  height: auto;
}

.email-html-preview :deep(table) {
  width: 100%;
  border-collapse: collapse;
}

.email-html-preview :deep(.btn) {
  display: inline-block;
  padding: 12px 30px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  text-decoration: none;
  border-radius: 25px;
  font-weight: 500;
  margin: 20px 0;
}

.email-source-code {
  font-family: 'Courier New', monospace;
  font-size: 12px;
}
</style>
