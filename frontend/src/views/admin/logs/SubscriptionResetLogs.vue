<template>
  <div class="log-list logs-page">
    <div class="filter-bar desktop-only">
      <el-input v-model="filter.keyword" placeholder="用户名/邮箱/订阅链接" clearable class="filter-keyword" @input="debouncedFetch" @clear="fetch" @keyup.enter="fetch" />
      <el-select v-model="filter.reset_type" placeholder="重置类型" clearable class="filter-select-md" @change="fetch">
        <el-option v-for="(label, value) in RESET_TYPE_MAP" :key="value" :label="label" :value="value" />
      </el-select>
      <el-select v-model="filter.reset_by" placeholder="操作方" clearable class="filter-select-xs" @change="fetch">
        <el-option label="用户" value="user" />
        <el-option label="管理员" value="admin" />
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
          <el-button type="danger" plain :loading="clearing" @click="runCleanup">清空</el-button>
    </div>
    <div class="filter-bar mobile-only">
      <el-form label-position="top" class="mobile-filter-form">
        <el-form-item label="关键词"><el-input v-model="filter.keyword" placeholder="用户名/邮箱/订阅链接" clearable @input="debouncedFetch" @clear="fetch" /></el-form-item>
        <el-form-item label="重置类型">
          <el-select v-model="filter.reset_type" placeholder="重置类型" clearable class="full-width-control" @change="fetch">
            <el-option v-for="(label, value) in RESET_TYPE_MAP" :key="value" :label="label" :value="value" />
          </el-select>
        </el-form-item>
        <el-form-item label="操作方">
          <el-select v-model="filter.reset_by" placeholder="操作方" clearable class="full-width-control" @change="fetch">
            <el-option label="用户" value="user" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker v-model="filter.timeRange" type="datetimerange" range-separator="至" start-placeholder="开始" end-placeholder="结束" value-format="YYYY-MM-DD HH:mm:ss" class="full-width-control" @change="fetch" />
        </el-form-item>
        <div class="mobile-filter-actions">
          <el-button type="primary" @click="fetch" :loading="loading" class="mobile-action-btn">搜索</el-button>
          <el-button @click="resetFilter" class="mobile-action-btn">重置</el-button>
          <el-button type="danger" plain :loading="clearing" @click="runCleanup" class="mobile-action-btn">清空</el-button>
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
            <el-table-column prop="reset_type" label="重置类型" width="120">
              <template #default="{ row }">
                <el-tag :type="getResetTypeColor(row.reset_type)" size="small">{{ getResetTypeText(row.reset_type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="URL变化" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">
                <div class="reset-url">{{ (row.old_subscription_url || '').substring(0, 40) }}...</div>
                <div class="reset-arrow">↓</div>
                <div class="reset-url">{{ (row.new_subscription_url || '').substring(0, 40) }}...</div>
              </template>
            </el-table-column>
            <el-table-column label="设备数" width="100">
              <template #default="{ row }">
                {{ row.device_count_before ?? 0 }} → {{ row.device_count_after ?? 0 }}
              </template>
            </el-table-column>
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
              <el-tag :type="getResetTypeColor(item.reset_type)" size="small">{{ getResetTypeText(item.reset_type) }}</el-tag>
            </span>
          </div>
          <div class="mobile-log-field">
            <span class="mobile-log-label">操作方</span>
            <span class="mobile-log-value">{{ getResetByText(item.reset_by) }}</span>
          </div>
          <div class="mobile-log-field">
            <span class="mobile-log-label">设备数</span>
            <span class="mobile-log-value">{{ item.device_count_before ?? 0 }} → {{ item.device_count_after ?? 0 }}</span>
          </div>
          <div class="mobile-log-field field-full" v-if="item.old_subscription_url || item.new_subscription_url">
            <span class="mobile-log-label">URL变化</span>
            <span class="mobile-log-value mobile-log-wrap">{{ (item.old_subscription_url || '').substring(0, 30) }} → {{ (item.new_subscription_url || '').substring(0, 30) }}</span>
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
import { adminAPI } from '@/utils/api'
import { useLogListPage } from '@/composables/useLogListPage'
import PaginationBar from '@/components/PaginationBar.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import MobileLogFields from '@/components/MobileLogFields.vue'

const RESET_TYPE_MAP = {
  user_reset: '用户重置', admin_reset: '管理员重置', admin_batch_reset: '批量重置'
}
const getResetTypeText = (type) => RESET_TYPE_MAP[type] || type || '-'
const getResetTypeColor = (type) => {
  const map = { user_reset: '', admin_reset: 'warning', admin_batch_reset: 'danger' }
  return map[type] || 'info'
}
const getResetByText = (by) => {
  if (!by) return '-'
  if (by === 'user' || by === '用户') return '用户'
  if (by === 'admin' || by === '管理员') return '管理员'
  return by
}

const {
  loading, list, total, page, pageSize, filter, isMobile, paginationLayout,
  fetchLogs: fetch, debouncedFetchLogs: debouncedFetch, resetFilter, onSizeChange,
  clearing, runCleanup
} = useLogListPage({
  fetcher: (params) => adminAPI.getSubscriptionResetLogs(params),
  defaultFilter: () => ({ keyword: '', reset_type: '', reset_by: '', timeRange: null }),
  extraFilterKeys: ['reset_type', 'reset_by'],
  clearAction: () => adminAPI.cleanupData('subscription_reset_logs'),
  clearLabel: '订阅重置日志'
})
</script>
<style scoped>
.filter-keyword,
.filter-select-xs,
.filter-select-md,
.filter-date {
  width: 100%;
  min-width: 0;
}
.sub-text { font-size: 12px; color: #909399; }
.reset-arrow { text-align: center; color: #409eff; }
</style>
