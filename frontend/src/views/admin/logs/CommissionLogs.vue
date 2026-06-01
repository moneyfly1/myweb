<template>
  <div class="log-list logs-page">
    <div class="filter-bar desktop-only">
      <el-input v-model="filter.keyword" placeholder="邀请人/被邀请人" clearable style="width: 200px" @keyup.enter="fetch" />
      <el-select v-model="filter.commission_type" placeholder="佣金类型" clearable style="width: 120px">
        <el-option v-for="(label, value) in COMMISSION_TYPE_MAP" :key="value" :label="label" :value="value" />
      </el-select>
      <el-select v-model="filter.status" placeholder="状态" clearable style="width: 100px">
        <el-option label="待结算" value="pending" />
        <el-option label="已结算" value="paid" />
        <el-option label="已取消" value="cancelled" />
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
        <el-form-item label="关键词"><el-input v-model="filter.keyword" placeholder="邀请人/被邀请人" clearable /></el-form-item>
        <el-form-item label="佣金类型">
          <el-select v-model="filter.commission_type" placeholder="佣金类型" clearable style="width: 100%">
            <el-option v-for="(label, value) in COMMISSION_TYPE_MAP" :key="value" :label="label" :value="value" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filter.status" placeholder="状态" clearable style="width: 100%">
            <el-option label="待结算" value="pending" />
            <el-option label="已结算" value="paid" />
            <el-option label="已取消" value="cancelled" />
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
      <el-table-column label="邀请人" width="160" show-overflow-tooltip>
        <template #default="{ row }">{{ row.inviter_name }} <small class="text-muted">{{ row.inviter_email }}</small></template>
      </el-table-column>
      <el-table-column label="被邀请人" width="160" show-overflow-tooltip>
        <template #default="{ row }">{{ row.invitee_name }} <small class="text-muted">{{ row.invitee_email }}</small></template>
      </el-table-column>
      <el-table-column prop="commission_type" label="类型" width="110">
        <template #default="{ row }">
          <el-tag size="small" type="info">{{ getCommissionTypeText(row.commission_type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="amount" label="佣金" width="100">
        <template #default="{ row }">
          <span class="text-green">+{{ (row.amount || 0).toFixed(2) }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="order_no" label="关联订单" width="140" />
      <el-table-column prop="status" label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="getStatusColor(row.status)" size="small">{{ getStatusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="settled_at" label="结算时间" width="180" />
      <el-table-column prop="description" label="说明" min-width="160" show-overflow-tooltip />
    </el-table>
    </div>
    <div class="mobile-only mobile-card-list">
      <div v-loading="loading" class="mobile-list-inner">
        <div v-for="row in list" :key="row.id" class="mobile-log-card">
          <div class="mobile-card-row"><span class="mobile-label">时间</span><span class="mobile-value">{{ row.created_at || '-' }}</span></div>
          <div class="mobile-card-row"><span class="mobile-label">邀请人</span><span class="mobile-value">{{ row.inviter_name || '-' }}</span></div>
          <div class="mobile-card-row"><span class="mobile-label">被邀请人</span><span class="mobile-value">{{ row.invitee_name || '-' }}</span></div>
          <div class="mobile-card-row"><span class="mobile-label">类型</span><span class="mobile-value">{{ getCommissionTypeText(row.commission_type) }}</span></div>
          <div class="mobile-card-row"><span class="mobile-label">佣金</span><span class="mobile-value text-green">+{{ (row.amount || 0).toFixed(2) }}</span></div>
          <div class="mobile-card-row"><span class="mobile-label">状态</span><span class="mobile-value"><el-tag :type="getStatusColor(row.status)" size="small">{{ getStatusText(row.status) }}</el-tag></span></div>
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
import { ref, onMounted, computed } from 'vue'
import { adminAPI } from '@/utils/api'
import { useMobile } from '@/composables/useMobile'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const COMMISSION_TYPE_MAP = {
  register_reward: '注册奖励', order_commission: '订单佣金'
}
const STATUS_MAP = { pending: '待结算', paid: '已结算', cancelled: '已取消' }

const getCommissionTypeText = (type) => COMMISSION_TYPE_MAP[type] || type || '-'
const getStatusText = (status) => STATUS_MAP[status] || status || '-'
const getStatusColor = (status) => {
  const map = { pending: 'warning', paid: 'success', cancelled: 'info' }
  return map[status] || ''
}

const filter = ref({
  keyword: '',
  commission_type: '',
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
  filter.value = { keyword: '', commission_type: '', status: '', timeRange: null }
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
.log-list { padding: 0; }
.filter-bar { display: flex; flex-wrap: wrap; gap: 12px; margin-bottom: 16px; align-items: center; }
.pagination { margin-top: 16px; justify-content: flex-end; }
.text-green { color: #67c23a; }
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
  .pagination { flex-wrap: wrap; justify-content: center; padding: 12px 0; }
}
</style>
