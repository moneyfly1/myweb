<template>
  <div class="log-list">
    <div class="filter-bar">
      <el-input v-model="filter.keyword" placeholder="收件人邮箱" clearable style="width: 220px" @keyup.enter="fetch" />
      <el-select v-model="filter.email_type" placeholder="邮件类型" clearable style="width: 140px" />
      <el-select v-model="filter.status" placeholder="状态" clearable style="width: 120px">
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
        style="width: 360px"
      />
      <el-button type="primary" @click="fetch" :loading="loading">搜索</el-button>
      <el-button @click="resetFilter">重置</el-button>
    </div>
    <el-table v-loading="loading" :data="list" stripe border>
      <el-table-column prop="created_at" label="创建时间" width="180" />
      <el-table-column prop="to_email" label="收件人" width="200" />
      <el-table-column prop="subject" label="主题" min-width="200" show-overflow-tooltip />
      <el-table-column prop="email_type" label="类型" width="120" />
      <el-table-column prop="status" label="状态" width="100" />
      <el-table-column prop="retry_count" label="重试次数" width="90" />
      <el-table-column prop="sent_at" label="发送时间" width="180" />
      <el-table-column prop="error_message" label="错误信息" min-width="180" show-overflow-tooltip />
    </el-table>
    <el-pagination
      v-model:current-page="page"
      :page-size="pageSize"
      :total="total"
      layout="total, prev, pager, next, sizes"
      :page-sizes="[10, 20, 50]"
      @current-change="fetch"
      @size-change="onSizeChange"
      class="pagination"
    />
  </div>
</template>
<script setup>
import { ref, onMounted } from 'vue'
import { adminAPI } from '@/utils/api'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const filter = ref({
  keyword: '',
  email_type: '',
  status: '',
  timeRange: null
})

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

onMounted(fetch)
</script>
<style scoped>
.log-list { padding: 0; }
.filter-bar { display: flex; flex-wrap: wrap; gap: 12px; margin-bottom: 16px; align-items: center; }
.pagination { margin-top: 16px; justify-content: flex-end; }
</style>
