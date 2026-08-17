<template>
  <div class="log-list logs-page">
    <div class="filter-bar desktop-only">
      <el-input v-model="filter.keyword" placeholder="收件人邮箱" clearable class="filter-keyword" @input="debouncedFetch" @clear="fetch" @keyup.enter="fetch" />
      <el-select v-model="filter.email_type" placeholder="邮件类型" clearable class="filter-select-md" @change="fetch">
        <el-option v-for="(label, value) in EMAIL_TYPE_MAP" :key="value" :label="label" :value="value" />
      </el-select>
      <el-select v-model="filter.status" placeholder="状态" clearable class="filter-select-sm" @change="fetch">
        <el-option label="待发送" value="pending" />
        <el-option label="已发送" value="sent" />
        <el-option label="发送失败" value="failed" />
      </el-select>
      <el-date-picker
        v-model="filter.timeRange"
        type="datetimerange"
        range-separator="至"
        start-placeholder="开始时间"
        end-placeholder="结束时间"
        value-format="YYYY-MM-DD HH:mm:ss"
        class="filter-date"
        @change="fetch"
      />
      <el-button type="primary" @click="fetch" :loading="loading">搜索</el-button>
      <el-button @click="resetFilter">重置</el-button>
    </div>
    <div class="filter-bar mobile-only">
      <el-form label-position="top" class="mobile-filter-form">
        <el-form-item label="收件人"><el-input v-model="filter.keyword" placeholder="收件人邮箱" clearable @input="debouncedFetch" @clear="fetch" /></el-form-item>
        <el-form-item label="邮件类型">
          <el-select v-model="filter.email_type" placeholder="邮件类型" clearable class="full-width-control" @change="fetch">
            <el-option v-for="(label, value) in EMAIL_TYPE_MAP" :key="value" :label="label" :value="value" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filter.status" placeholder="状态" clearable class="full-width-control" @change="fetch">
            <el-option label="待发送" value="pending" />
            <el-option label="已发送" value="sent" />
            <el-option label="发送失败" value="failed" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker v-model="filter.timeRange" type="datetimerange" range-separator="至" start-placeholder="开始" end-placeholder="结束" value-format="YYYY-MM-DD HH:mm:ss" class="full-width-control" @change="fetch" />
        </el-form-item>
        <div class="mobile-filter-actions">
          <el-button type="primary" @click="fetch" :loading="loading" class="mobile-action-btn">搜索</el-button>
          <el-button @click="resetFilter" class="mobile-action-btn">重置</el-button>
        </div>
      </el-form>
    </div>
    <ResponsiveDataView
      :data="list"
      :loading="loading"
      :fields="[]"
      title-field="to_email"
      empty-title="暂无数据"
    >
      <template #table>
        <div class="table-wrapper">
          <el-table v-loading="loading" :data="list" stripe border>
            <el-table-column prop="created_at" label="创建时间" width="180" />
            <el-table-column prop="to_email" label="收件人" width="200" />
            <el-table-column prop="subject" label="主题" min-width="200" show-overflow-tooltip />
            <el-table-column prop="email_type" label="类型" width="120">
              <template #default="{ row }">
                {{ getEmailTypeText(row.email_type) }}
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="getStatusColor(row.status)" size="small">{{ getStatusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="retry_count" label="重试" width="70" />
            <el-table-column prop="sent_at" label="发送时间" width="180" />
            <el-table-column prop="error_message" label="错误信息" min-width="180" show-overflow-tooltip />
          </el-table>
        </div>
      </template>
      <template #header="{ item }">
        <div class="mobile-log-title">{{ item.to_email || '-' }}</div>
        <div class="mobile-log-subtitle">{{ item.created_at || '-' }}</div>
      </template>
      <template #default="{ item }">
        <MobileLogFields>
          <div class="mobile-log-field">
            <span class="mobile-log-label">类型</span>
            <span class="mobile-log-value">{{ getEmailTypeText(item.email_type) }}</span>
          </div>
          <div class="mobile-log-field field-full">
            <span class="mobile-log-label">主题</span>
            <span class="mobile-log-value mobile-log-wrap">{{ item.subject || '-' }}</span>
          </div>
          <div class="mobile-log-field">
            <span class="mobile-log-label">状态</span>
            <span class="mobile-log-value">
              <el-tag :type="getStatusColor(item.status)" size="small">{{ getStatusText(item.status) }}</el-tag>
            </span>
          </div>
          <div class="mobile-log-field field-full" v-if="item.error_message">
            <span class="mobile-log-label">错误</span>
            <span class="mobile-log-value mobile-log-wrap">{{ item.error_message }}</span>
          </div>
        </MobileLogFields>
      </template>
    </ResponsiveDataView>
    <PaginationBar
      v-model:current-page="page"
      v-model:page-size="pageSize"
      :total="total"
      :layout="paginationLayout"
      :page-sizes="[10, 20, 50]"
      @current-change="fetch"
      @size-change="onSizeChange"
    />
  </div>
</template>
<script setup>
import { ref, onMounted, computed } from 'vue'
import { debounce } from '@/composables/useDebounce'
import { adminAPI } from '@/utils/api'
import { useMobile } from '@/composables/useMobile'
import { EMAIL_TYPE_MAP, getEmailTypeText } from '@/utils/statusMaps'
import PaginationBar from '@/components/PaginationBar.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import MobileLogFields from '@/components/MobileLogFields.vue'

const loading = ref(false)
const getStatusText = (status) => {
  const map = { pending: '待发送', sent: '已发送', failed: '失败' }
  return map[status] || status || '-'
}
const getStatusColor = (status) => {
  const map = { pending: 'warning', sent: 'success', failed: 'danger' }
  return map[status] || 'info'
}

const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const filter = ref({
  keyword: '',
  email_type: '',
  status: '',
  timeRange: null
})
const isMobile = useMobile()
const paginationLayout = computed(() => (isMobile.value ? 'total, prev, pager, next' : 'total, prev, pager, next, sizes'))

async function fetch() {
  loading.value = true
  try {
    const params = { page: page.value, page_size: pageSize.value }
    if (filter.value.keyword) params.keyword = filter.value.keyword
    if (filter.value.email_type) params.email_type = filter.value.email_type
    if (filter.value.status) params.status = filter.value.status
    if (filter.value.timeRange && filter.value.timeRange.length === 2) {
      params.start_time = filter.value.timeRange[0]
      params.end_time = filter.value.timeRange[1]
    }
    const res = await adminAPI.getEmailLogs(params)
    const data = res?.data?.data ?? res?.data ?? {}
    list.value = data.logs ?? []
    total.value = data.total ?? 0
  } catch (e) {
    list.value = []
  } finally {
    loading.value = false
  }
}

// 搜索输入实时生效，无需再次点击搜索按钮（500ms 防抖）
const debouncedFetch = debounce(fetch, 500)

function resetFilter() {
  filter.value = { keyword: '', email_type: '', status: '', timeRange: null }
  page.value = 1
  fetch()
}

function onSizeChange(size) {
  pageSize.value = size
  page.value = 1
  fetch()
}

onMounted(() => { fetch() })
</script>
<style scoped>
.filter-keyword,
.filter-select-sm,
.filter-select-md,
.filter-date {
  width: 100%;
  min-width: 0;
}
</style>
