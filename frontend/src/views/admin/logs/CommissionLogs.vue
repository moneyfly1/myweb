<template>
  <div class="log-list logs-page">
    <div class="filter-bar desktop-only">
      <el-input v-model="filter.keyword" placeholder="邀请人/被邀请人" clearable class="filter-keyword" @keyup.enter="fetch" />
      <el-select v-model="filter.commission_type" placeholder="佣金类型" clearable class="filter-select-sm">
        <el-option v-for="(label, value) in COMMISSION_TYPE_MAP" :key="value" :label="label" :value="value" />
      </el-select>
      <el-select v-model="filter.status" placeholder="状态" clearable class="filter-select-xs">
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
        class="filter-date"
      />
      <el-button type="primary" @click="fetch" :loading="loading">搜索</el-button>
      <el-button @click="resetFilter">重置</el-button>
    </div>
    <div class="filter-bar mobile-only">
      <el-form label-position="top" class="mobile-filter-form">
        <el-form-item label="关键词"><el-input v-model="filter.keyword" placeholder="邀请人/被邀请人" clearable /></el-form-item>
        <el-form-item label="佣金类型">
          <el-select v-model="filter.commission_type" placeholder="佣金类型" clearable class="full-width-control">
            <el-option v-for="(label, value) in COMMISSION_TYPE_MAP" :key="value" :label="label" :value="value" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filter.status" placeholder="状态" clearable class="full-width-control">
            <el-option label="待结算" value="pending" />
            <el-option label="已结算" value="paid" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker v-model="filter.timeRange" type="datetimerange" range-separator="至" start-placeholder="开始" end-placeholder="结束" value-format="YYYY-MM-DD HH:mm:ss" class="full-width-control" />
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
      title-field="inviter_name"
      empty-title="暂无数据"
    >
      <template #table>
        <div class="table-wrapper">
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
      </template>
      <template #header="{ item }">
        <div class="mobile-log-title">{{ item.inviter_name || '-' }}</div>
        <div class="mobile-log-subtitle">{{ item.created_at || '-' }}</div>
      </template>
      <template #default="{ item }">
        <MobileLogFields>
          <div class="mobile-log-field">
            <span class="mobile-log-label">被邀请人</span>
            <span class="mobile-log-value">{{ item.invitee_name || '-' }}</span>
          </div>
          <div class="mobile-log-field">
            <span class="mobile-log-label">类型</span>
            <span class="mobile-log-value">{{ getCommissionTypeText(item.commission_type) }}</span>
          </div>
          <div class="mobile-log-field">
            <span class="mobile-log-label">佣金</span>
            <span class="mobile-log-value text-green">+{{ (item.amount || 0).toFixed(2) }}</span>
          </div>
          <div class="mobile-log-field">
            <span class="mobile-log-label">状态</span>
            <span class="mobile-log-value">
              <el-tag :type="getStatusColor(item.status)" size="small">{{ getStatusText(item.status) }}</el-tag>
            </span>
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
import { adminAPI } from '@/utils/api'
import { useMobile } from '@/composables/useMobile'
import PaginationBar from '@/components/PaginationBar.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import MobileLogFields from '@/components/MobileLogFields.vue'

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
.filter-keyword,
.filter-select-xs,
.filter-select-sm,
.filter-date {
  width: 100%;
  min-width: 0;
}
.text-green { color: #67c23a; }
</style>
