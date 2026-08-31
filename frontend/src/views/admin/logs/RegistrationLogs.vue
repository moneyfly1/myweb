<template>
  <div class="log-list logs-page">
    <div class="filter-bar desktop-only">
      <el-input v-model="filter.keyword" placeholder="用户名/邮箱/邀请码" clearable class="filter-keyword" @input="debouncedFetch" @clear="fetch" @keyup.enter="fetch" />
      <el-select v-model="filter.status" placeholder="状态" clearable class="filter-select-sm" @change="fetch">
        <el-option label="成功" value="success" />
        <el-option label="失败" value="failed" />
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
        <el-form-item label="关键词">
          <el-input v-model="filter.keyword" placeholder="用户名/邮箱/邀请码" clearable @input="debouncedFetch" @clear="fetch" @keyup.enter="fetch" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filter.status" placeholder="状态" clearable class="full-width-control" @change="fetch">
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="filter.timeRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始"
            end-placeholder="结束"
            value-format="YYYY-MM-DD HH:mm:ss"
            class="full-width-control"
            @change="fetch"
          />
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
          <el-table v-loading="loading" :data="list" stripe border class="resizable-table">
            <el-table-column prop="created_at" label="时间" width="180" />
            <el-table-column prop="username" label="用户" width="120" />
            <el-table-column prop="email" label="邮箱" width="180" />
            <el-table-column prop="ip_address" label="IP" width="130" />
            <el-table-column prop="location" label="地区" width="120">
              <template #default="{ row }">{{ displayLocation(row.location) }}</template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="80" />
            <el-table-column prop="invite_code" label="邀请码" width="100" />
            <el-table-column prop="inviter_name" label="邀请人" width="100" />
            <el-table-column prop="reason" :min-width="reasonColWidth || 180" show-overflow-tooltip>
              <template #header>
                <div class="th-resizable">
                  <span>失败原因</span>
                  <span class="resize-handle" @mousedown.prevent="startResize($event, 'reason')" title="拖拽调整列宽">⋮</span>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </template>
      <template #header="{ item }">
        <div class="mobile-log-title">{{ item.username || item.email || '-' }}</div>
        <div class="mobile-log-subtitle">{{ item.created_at || '-' }}</div>
      </template>
      <template #default="{ item }">
        <MobileLogFields>
          <div class="mobile-log-field">
            <span class="mobile-log-label">状态</span>
            <span class="mobile-log-value">{{ item.status || '-' }}</span>
          </div>
          <div class="mobile-log-field" v-if="item.location">
            <span class="mobile-log-label">地区</span>
            <span class="mobile-log-value">{{ displayLocation(item.location) }}</span>
          </div>
          <div class="mobile-log-field field-full" v-if="item.reason">
            <span class="mobile-log-label">失败原因</span>
            <span class="mobile-log-value mobile-log-wrap">{{ item.reason }}</span>
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
import { ref } from 'vue'
import { adminAPI } from '@/utils/api'
import { useLogListPage } from '@/composables/useLogListPage'
import { formatLocation } from '@/utils/date'
import PaginationBar from '@/components/PaginationBar.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import MobileLogFields from '@/components/MobileLogFields.vue'

const displayLocation = (loc) => {
  if (!loc) return '-'
  const result = formatLocation(loc)
  return result || loc
}

const reasonColWidth = ref(160)
const {
  loading, list, total, page, pageSize, filter, isMobile, paginationLayout,
  fetchLogs: fetch, debouncedFetchLogs: debouncedFetch, resetFilter, onSizeChange,
  clearing, runCleanup
} = useLogListPage({
  fetcher: (params) => adminAPI.getRegistrationLogs(params),
  defaultFilter: () => ({ keyword: '', status: '', timeRange: null }),
  extraFilterKeys: ['status'],
  clearAction: () => adminAPI.cleanupData('registration_logs'),
  clearLabel: '注册日志'
})

function startResize(e, col) {
  const startX = e.clientX
  const startW = col === 'reason' ? reasonColWidth.value : 160
  const onMove = (e2) => {
    const dx = e2.clientX - startX
    const newW = Math.max(80, Math.min(500, startW + dx))
    if (col === 'reason') reasonColWidth.value = newW
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
</script>
<style scoped>
.filter-keyword,
.filter-select-sm,
.filter-date {
  width: 100%;
  min-width: 0;
}
.th-resizable { display: flex; align-items: center; justify-content: space-between; width: 100%; gap: 4px; }
.th-resizable span:first-child { flex: 1; overflow: hidden; text-overflow: ellipsis; }
.resize-handle { cursor: col-resize; padding: 0 4px; color: #909399; user-select: none; }
.resize-handle:hover { color: #409eff; }
</style>
