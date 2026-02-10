<template>
  <div class="log-list">
    <div class="filter-bar">
      <el-input v-model="filter.keyword" placeholder="用户名/邮箱/订阅链接" clearable style="width: 220px" @keyup.enter="fetch" />
      <el-input v-model="filter.user_id" placeholder="用户ID" clearable style="width: 100px" />
      <el-select v-model="filter.action_type" placeholder="操作类型" clearable style="width: 120px" />
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
    <el-table v-loading="loading" :data="list" stripe border class="resizable-table">
      <el-table-column prop="created_at" label="时间" width="180" />
      <el-table-column prop="user_id" label="用户ID" width="90" />
      <el-table-column prop="username" label="用户名" width="120" />
      <el-table-column prop="user_email" label="邮箱" width="160" />
      <el-table-column prop="subscription_id" label="订阅ID" width="90" />
      <el-table-column prop="action_type" label="操作类型" width="120" />
      <el-table-column prop="action_by" label="操作方" width="90" />
      <el-table-column prop="action_by_user" label="操作人" width="100" />
      <el-table-column prop="description" :width="descriptionColWidth" min-width="80" show-overflow-tooltip>
        <template #header>
          <div class="th-resizable">
            <span>说明</span>
            <span class="resize-handle" @mousedown.prevent="startResize($event, 'description')" title="拖拽调整列宽">⋮</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="ip_address" label="IP/地区" width="140" show-overflow-tooltip />
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
const descriptionColWidth = ref(180)
const filter = ref({
  keyword: '',
  user_id: '',
  action_type: '',
  timeRange: null
})

function startResize(e, col) {
  const startX = e.clientX
  const startW = col === 'description' ? descriptionColWidth.value : 180
  const onMove = (e2) => {
    const dx = e2.clientX - startX
    const newW = Math.max(80, Math.min(500, startW + dx))
    if (col === 'description') descriptionColWidth.value = newW
  }
  const onUp = () => {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

async function fetch() {
  loading.value = true
  try {
    const params = { page: page.value, page_size: pageSize.value }
    if (filter.value.keyword) params.keyword = filter.value.keyword
    if (filter.value.user_id) params.user_id = filter.value.user_id
    if (filter.value.action_type) params.action_type = filter.value.action_type
    if (filter.value.timeRange && filter.value.timeRange.length === 2) {
      params.start_time = filter.value.timeRange[0]
      params.end_time = filter.value.timeRange[1]
    }
    const res = await adminAPI.getSubscriptionLogs(params)
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
  filter.value = { keyword: '', user_id: '', action_type: '', timeRange: null }
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
.th-resizable { display: flex; align-items: center; justify-content: space-between; width: 100%; gap: 4px; }
.th-resizable span:first-child { flex: 1; overflow: hidden; text-overflow: ellipsis; }
.resize-handle { cursor: col-resize; padding: 0 4px; color: #909399; user-select: none; }
.resize-handle:hover { color: #409eff; }
</style>
