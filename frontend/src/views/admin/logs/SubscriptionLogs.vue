<template>
  <div class="log-list logs-page">
    <div class="filter-bar desktop-only">
      <el-input v-model="filter.keyword" placeholder="用户名/邮箱/订阅链接" clearable class="filter-keyword" @input="debouncedFetch" @clear="fetch" @keyup.enter="fetch" />
      <el-select v-model="filter.action_type" placeholder="操作类型" clearable class="filter-select-sm" @change="fetch">
        <el-option v-for="(label, value) in ACTION_TYPE_MAP" :key="value" :label="label" :value="value" />
      </el-select>
      <el-select v-model="filter.action_by" placeholder="操作方" clearable class="filter-select-xs" @change="fetch">
        <el-option label="用户" value="user" />
        <el-option label="管理员" value="admin" />
        <el-option label="系统" value="system" />
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
      <div class="filter-actions">
      <el-button type="primary" @click="fetch" :loading="loading">搜索</el-button>
      <el-button @click="resetFilter">重置</el-button>
          <el-button type="danger" plain :loading="clearing" @click="runCleanup">清空</el-button>

      </div>    </div>
    <div class="filter-bar mobile-only">
      <el-form label-position="top" class="mobile-filter-form">
        <el-form-item label="关键词"><el-input v-model="filter.keyword" placeholder="用户名/邮箱/订阅链接" clearable @input="debouncedFetch" @clear="fetch" /></el-form-item>
        <el-form-item label="操作类型">
          <el-select v-model="filter.action_type" placeholder="操作类型" clearable class="full-width-control" @change="fetch">
            <el-option v-for="(label, value) in ACTION_TYPE_MAP" :key="value" :label="label" :value="value" />
          </el-select>
        </el-form-item>
        <el-form-item label="操作方">
          <el-select v-model="filter.action_by" placeholder="操作方" clearable class="full-width-control" @change="fetch">
            <el-option label="用户" value="user" />
            <el-option label="管理员" value="admin" />
            <el-option label="系统" value="system" />
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
              </template>
            </el-table-column>
            <el-table-column prop="action_type" label="操作类型" width="100">
              <template #default="{ row }">
                <el-tag :type="getActionTypeColor(row.action_type)" size="small">{{ getActionTypeText(row.action_type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="订阅方式" width="100">
              <template #default="{ row }">
                <el-tag v-if="getSubType(row)" size="small" :type="getSubType(row) === 'Clash订阅' ? '' : 'success'">{{ getSubType(row) }}</el-tag>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column label="软件/设备" width="150" show-overflow-tooltip>
              <template #default="{ row }">{{ getDeviceInfo(row) }}</template>
            </el-table-column>
            <el-table-column label="操作人" width="120">
              <template #default="{ row }">
                <span>{{ formatOperator(row) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="说明" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ row.description || '-' }}</template>
            </el-table-column>
            <el-table-column prop="ip_address" label="IP/地区" width="160" show-overflow-tooltip />
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
            <span class="mobile-log-label">操作</span>
            <span class="mobile-log-value">
              <el-tag :type="getActionTypeColor(item.action_type)" size="small">{{ getActionTypeText(item.action_type) }}</el-tag>
            </span>
          </div>
          <div class="mobile-log-field" v-if="getSubType(item)">
            <span class="mobile-log-label">订阅方式</span>
            <span class="mobile-log-value">{{ getSubType(item) }}</span>
          </div>
          <div class="mobile-log-field" v-if="getDeviceInfo(item)">
            <span class="mobile-log-label">软件/设备</span>
            <span class="mobile-log-value">{{ getDeviceInfo(item) }}</span>
          </div>
          <div class="mobile-log-field">
            <span class="mobile-log-label">操作人</span>
            <span class="mobile-log-value">{{ formatOperator(item) }}</span>
          </div>
          <div class="mobile-log-field field-full" v-if="item.description">
            <span class="mobile-log-label">说明</span>
            <span class="mobile-log-value mobile-log-wrap">{{ item.description || '-' }}</span>
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

const ACTION_TYPE_MAP = {
  create: '创建', update: '更新', delete: '删除',
  activate: '激活', deactivate: '停用', reset: '重置', access: '订阅访问'
}
const ACTION_BY_MAP = { user: '用户', admin: '管理员', system: '系统' }

const getActionTypeText = (type) => ACTION_TYPE_MAP[type] || type || '-'
const getActionTypeColor = (type) => {
  const map = { create: 'success', update: '', delete: 'danger', activate: 'success', deactivate: 'warning', reset: 'info', access: '' }
  return map[type] || 'info'
}

function formatOperator(row) {
  const by = ACTION_BY_MAP[row.action_by] || row.action_by || ''
  const user = row.action_by_user || ''
  if (by && user) return `${by}(${user})`
  return by || user || '-'
}

// 从 after_data 或描述中提取订阅类型
function getSubType(row) {
  const d = row.description || ''
  const m = d.match(/^\[(.+?)\]/)
  if (m) return m[1] === 'clash' ? 'Clash订阅' : m[1] === 'universal' ? '通用订阅' : m[1]
  // 回退到 before/after data
  try {
    const ad = JSON.parse(row.before_data || '{}')
    if (ad.type) return ad.type === 'clash' ? 'Clash订阅' : ad.type === 'universal' ? '通用订阅' : ad.type
  } catch {}
  return ''
}

// 从 after_data 或描述中提取设备信息
function getDeviceInfo(row) {
  const d = row.description || ''
  // 格式: "[clash] Shadowrocket iPhone 15" → 提取软件名+设备名
  const clean = d.replace(/^\[.+?\]\s*/, '')
  if (clean && clean !== '-' && clean !== d) return clean
  // 回退到 after_data
  try {
    const ad = JSON.parse(row.after_data || row.before_data || '{}')
    const parts = []
    if (ad.software_name && ad.software_name !== 'Unknown') parts.push(ad.software_name)
    if (ad.device_name && ad.device_name !== 'Unknown Device') parts.push(ad.device_name)
    return parts.join(' ') || '-'
  } catch {}
  return clean || '-'
}

const {
  loading, list, total, page, pageSize, filter, isMobile, paginationLayout,
  fetchLogs: fetch, debouncedFetchLogs: debouncedFetch, resetFilter, onSizeChange,
  clearing, runCleanup
} = useLogListPage({
  fetcher: (params) => adminAPI.getSubscriptionLogs(params),
  defaultFilter: () => ({ keyword: '', action_type: '', action_by: '', timeRange: null }),
  extraFilterKeys: ['action_type', 'action_by'],
  clearAction: () => adminAPI.cleanupData('subscription_logs'),
  clearLabel: '订阅日志'
})
</script>
<style scoped>
.filter-keyword,
.filter-select-xs,
.filter-select-sm,
.filter-date {
  width: 100%;
  min-width: 0;
}
.sub-text { font-size: 12px; color: #909399; }
</style>
