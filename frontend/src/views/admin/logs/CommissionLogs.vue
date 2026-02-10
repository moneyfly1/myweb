<template>
  <div class="log-list">
    <div class="filter-bar">
      <el-input v-model="filter.keyword" placeholder="邀请人/被邀请人" clearable style="width: 200px" @keyup.enter="fetch" />
      <el-input v-model="filter.inviter_id" placeholder="邀请人ID" clearable style="width: 100px" />
      <el-select v-model="filter.commission_type" placeholder="佣金类型" clearable style="width: 120px" />
      <el-select v-model="filter.status" placeholder="状态" clearable style="width: 100px" />
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
      <el-table-column prop="created_at" label="时间" width="180" />
      <el-table-column prop="inviter_id" label="邀请人ID" width="90" />
      <el-table-column prop="inviter_name" label="邀请人" width="120" />
      <el-table-column prop="invitee_id" label="被邀请人ID" width="100" />
      <el-table-column prop="invitee_name" label="被邀请人" width="120" />
      <el-table-column prop="commission_type" label="类型" width="100" />
      <el-table-column prop="amount" label="佣金金额" width="110" />
      <el-table-column prop="order_no" label="关联订单" width="140" />
      <el-table-column prop="status" label="状态" width="90" />
      <el-table-column prop="settled_at" label="结算时间" width="180" />
      <el-table-column prop="description" label="说明" min-width="160" show-overflow-tooltip />
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
  inviter_id: '',
  commission_type: '',
  status: '',
  timeRange: null
})

async function fetch() {
  loading.value = true
  try {
    const params = { page: page.value, page_size: pageSize.value }
    if (filter.value.keyword) params.keyword = filter.value.keyword
    if (filter.value.inviter_id) params.inviter_id = filter.value.inviter_id
    if (filter.value.commission_type) params.commission_type = filter.value.commission_type
    if (filter.value.status) params.status = filter.value.status
    if (filter.value.timeRange && filter.value.timeRange.length === 2) {
      params.start_time = filter.value.timeRange[0]
      params.end_time = filter.value.timeRange[1]
    }
    const res = await adminAPI.getCommissionLogs(params)
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
  filter.value = { keyword: '', inviter_id: '', commission_type: '', status: '', timeRange: null }
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
