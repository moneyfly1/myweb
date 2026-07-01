<template>
  <div class="log-list logs-page">
    <div class="filter-bar desktop-only">
      <el-input v-model="filter.keyword" placeholder="用户名/邮箱" clearable class="filter-keyword" @keyup.enter="fetch" />
      <el-select v-model="filter.change_type" placeholder="变更类型" clearable class="filter-select-sm">
        <el-option v-for="(label, value) in CHANGE_TYPE_MAP" :key="value" :label="label" :value="value" />
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
        <el-form-item label="关键词"><el-input v-model="filter.keyword" placeholder="用户名/邮箱" clearable /></el-form-item>
        <el-form-item label="变更类型">
          <el-select v-model="filter.change_type" placeholder="变更类型" clearable class="full-width-control">
            <el-option v-for="(label, value) in CHANGE_TYPE_MAP" :key="value" :label="label" :value="value" />
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
      title-field="username"
      empty-title="暂无数据"
    >
      <template #table>
        <div class="table-wrapper">
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
      </template>
      <template #header="{ item }">
        <div class="mobile-log-title">{{ item.username || '-' }}</div>
        <div class="mobile-log-subtitle">{{ item.created_at || '-' }}</div>
      </template>
      <template #default="{ item }">
        <MobileLogFields>
          <div class="mobile-log-field">
            <span class="mobile-log-label">类型</span>
            <span class="mobile-log-value">
              <el-tag :type="getChangeTypeColor(item.change_type)" size="small">{{ getChangeTypeText(item.change_type) }}</el-tag>
            </span>
          </div>
          <div class="mobile-log-field">
            <span class="mobile-log-label">金额</span>
            <span class="mobile-log-value" :class="item.amount >= 0 ? 'text-green' : 'text-red'">
              {{ item.amount >= 0 ? '+' : '' }}{{ (item.amount || 0).toFixed(2) }}
            </span>
          </div>
          <div class="mobile-log-field">
            <span class="mobile-log-label">余额</span>
            <span class="mobile-log-value">{{ (item.balance_before || 0).toFixed(2) }} → {{ (item.balance_after || 0).toFixed(2) }}</span>
          </div>
          <div class="mobile-log-field field-full" v-if="item.description">
            <span class="mobile-log-label">说明</span>
            <span class="mobile-log-value mobile-log-wrap">{{ item.description }}</span>
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
import { ref, onMounted, onUnmounted, computed } from 'vue'
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
const isMobile = useMobile()
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

onMounted(() => { fetch() })
onUnmounted(() => {})
</script>
<style scoped>
.filter-keyword,
.filter-select-sm,
.filter-date {
  width: 100%;
  min-width: 0;
}
.sub-text { font-size: 12px; color: #909399; }
.text-green { color: #67c23a; }
.text-red { color: #f56c6c; }
</style>
