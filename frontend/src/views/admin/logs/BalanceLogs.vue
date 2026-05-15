<template>
  <div class="log-list logs-page">
    <div class="filter-bar desktop-only">
      <el-input v-model="filter.keyword" placeholder="用户名/邮箱" clearable style="width: 200px" @keyup.enter="fetch" />
      <el-select v-model="filter.change_type" placeholder="变更类型" clearable style="width: 120px">
        <el-option v-for="(label, value) in CHANGE_TYPE_MAP" :key="value" :label="label" :value="value" />
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
    <div class="filter-bar mobile-only">
      <el-form label-position="top" class="mobile-filter-form">
        <el-form-item label="关键词"><el-input v-model="filter.keyword" placeholder="用户名/邮箱" clearable /></el-form-item>
        <el-form-item label="变更类型">
          <el-select v-model="filter.change_type" placeholder="变更类型" clearable style="width: 100%">
            <el-option v-for="(label, value) in CHANGE_TYPE_MAP" :key="value" :label="label" :value="value" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker v-model="filter.timeRange" type="datetimerange" range-separator="至" start-placeholder="开始" end-placeholder="结束" value-format="YYYY-MM-DD HH:mm:ss" style="width: 100%" />
        </el-form-item>
        <div class="mobile-filter-actions">
          <el-button type="primary" @click="fetch" :loading="loading" class="mobile-action-btn">搜索</el-button>
          <el-button @click="resetFilter" class="mobile-action-btn">重置</el-button>
        </div>
      </el-form>
    </div>
    <div class="table-wrapper desktop-only">
    <el-table v-loading="loading" :data="list" stripe border>
      <el-table-column prop="created_at" label="时间" width="180" />
      <el-table-column label="用户" width="120">
        <template #default="{ row }">
          <span>{{ row.username || '-' }}</span>
          <div class="sub-text">ID: {{ row.user_id }}</div>
        </template>
      </el-table-column>
      <el-table-column prop="change_type" label="变更类型" width="100">
        <template #default="{ row }">
          <el-tag :type="getChangeTypeColor(row.change_type)" size="small">{{ getChangeTypeText(row.change_type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="amount" label="金额" width="100">
        <template #default="{ row }">
          <span :class="row.amount >= 0 ? 'text-green' : 'text-red'">{{ row.amount >= 0 ? '+' : '' }}{{ (row.amount || 0).toFixed(2) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="余额变化" width="140">
        <template #default="{ row }">{{ (row.balance_before || 0).toFixed(2) }} → {{ (row.balance_after || 0).toFixed(2) }}</template>
      </el-table-column>
      <el-table-column prop="order_no" label="关联订单" width="140" />
      <el-table-column prop="description" label="说明" min-width="160" show-overflow-tooltip />
      <el-table-column prop="operator_user" label="操作人" width="100" />
    </el-table>
    </div>
    <div class="mobile-only mobile-card-list">
      <div v-loading="loading" class="mobile-list-inner">
        <div v-for="row in list" :key="row.id" class="mobile-log-card">
          <div class="mobile-card-row"><span class="mobile-label">时间</span><span class="mobile-value">{{ row.created_at || '-' }}</span></div>
          <div class="mobile-card-row"><span class="mobile-label">用户</span><span class="mobile-value">{{ row.username || '-' }}</span></div>
          <div class="mobile-card-row"><span class="mobile-label">类型</span><span class="mobile-value"><el-tag :type="getChangeTypeColor(row.change_type)" size="small">{{ getChangeTypeText(row.change_type) }}</el-tag></span></div>
          <div class="mobile-card-row"><span class="mobile-label">金额</span><span class="mobile-value" :class="row.amount >= 0 ? 'text-green' : 'text-red'">{{ row.amount >= 0 ? '+' : '' }}{{ (row.amount || 0).toFixed(2) }}</span></div>
          <div class="mobile-card-row"><span class="mobile-label">余额</span><span class="mobile-value">{{ (row.balance_before || 0).toFixed(2) }} → {{ (row.balance_after || 0).toFixed(2) }}</span></div>
          <div class="mobile-card-row" v-if="row.description"><span class="mobile-label">说明</span><span class="mobile-value mobile-value-wrap">{{ row.description }}</span></div>
        </div>
        <el-empty v-if="list.length === 0 && !loading" description="暂无数据" />
      </div>
    </div>
    <el-pagination
      v-model:current-page="page"
      :page-size="pageSize"
      :total="total"
      :layout="paginationLayout"
      :page-sizes="[10, 20, 50]"
      @current-change="fetch"
      @size-change="onSizeChange"
      class="pagination"
    />
  </div>
</template>
<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { adminAPI } from '@/utils/api'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const CHANGE_TYPE_MAP = {
  recharge: '充值', consume: '消费', refund: '退款',
  commission: '佣金', gift: '赠送', admin_adjust: '管理员调整'
}
const getChangeTypeText = (type) => CHANGE_TYPE_MAP[type] || type || '-'
const getChangeTypeColor = (type) => {
  const map = { recharge: 'success', consume: 'danger', refund: 'warning', commission: '', gift: 'info', admin_adjust: 'warning' }
  return map[type] || 'info'
}

const filter = ref({
  keyword: '',
  change_type: '',
  timeRange: null
})
const isMobile = ref(false)
function checkMobile() { isMobile.value = window.innerWidth <= 768 }
const paginationLayout = computed(() => (isMobile.value ? 'total, prev, pager, next' : 'total, prev, pager, next, sizes'))

async function fetch() {
  loading.value = true
  try {
    const params = { page: page.value, page_size: pageSize.value }
    if (filter.value.keyword) params.keyword = filter.value.keyword
    if (filter.value.change_type) params.change_type = filter.value.change_type
    if (filter.value.timeRange && filter.value.timeRange.length === 2) {
      params.start_time = filter.value.timeRange[0]
      params.end_time = filter.value.timeRange[1]
    }
    const res = await adminAPI.getBalanceLogs(params)
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
  filter.value = { keyword: '', change_type: '', timeRange: null }
  page.value = 1
  fetch()
}

function onSizeChange(size) {
  pageSize.value = size
  page.value = 1
  fetch()
}

onMounted(() => { checkMobile(); window.addEventListener('resize', checkMobile); fetch() })
onUnmounted(() => { window.removeEventListener('resize', checkMobile) })
</script>
<style scoped>
.log-list { padding: 0; }
.filter-bar { display: flex; flex-wrap: wrap; gap: 12px; margin-bottom: 16px; align-items: center; }
.pagination { margin-top: 16px; justify-content: flex-end; }
.sub-text { font-size: 12px; color: #909399; }
.text-green { color: #67c23a; }
.text-red { color: #f56c6c; }
.desktop-only { display: block; }
.mobile-only { display: none; }
.mobile-filter-form { width: 100%; }
.mobile-filter-actions { display: flex; flex-direction: column; gap: 10px; margin-top: 12px; }
.mobile-action-btn { width: 100%; min-height: 44px; }
.table-wrapper { overflow-x: auto; }
@media (max-width: 768px) {
  .logs-page { padding: 0 4px; }
  .desktop-only { display: none !important; }
  .mobile-only { display: block !important; }
  .filter-bar.mobile-only { margin-bottom: 12px; }
  .mobile-list-inner { display: flex; flex-direction: column; gap: 12px; min-height: 120px; }
  .mobile-log-card { background: #fff; border: 1px solid #ebeef5; border-radius: 8px; padding: 12px 14px; box-shadow: 0 1px 3px rgba(0,0,0,.08); }
  .mobile-card-row { display: flex; margin-bottom: 8px; font-size: 14px; }
  .mobile-card-row:last-child { margin-bottom: 0; }
  .mobile-label { flex: 0 0 72px; color: #909399; }
  .mobile-value { flex: 1; word-break: break-all; }
  .mobile-value-wrap { white-space: pre-wrap; word-break: break-word; }
  .pagination { flex-wrap: wrap; justify-content: center; padding: 12px 0; }
}
</style>
