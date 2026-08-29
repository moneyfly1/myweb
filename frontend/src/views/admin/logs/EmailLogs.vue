<template>
  <div class="log-list logs-page">
    <div class="filter-bar desktop-only">
      <el-input v-model="filter.keyword" placeholder="收件人邮箱" clearable class="filter-keyword" @input="debouncedFetch" @clear="fetch" @keyup.enter="fetch" />
      <el-select v-model="filter.email_type" placeholder="邮件类型" clearable class="filter-select-md" @change="fetch">
        <el-option v-for="(label, value) in EMAIL_TYPE_MAP" :key="value" :label="label" :value="value" />
      </el-select>
      <el-select v-model="filter.status" placeholder="状态" clearable class="filter-select-sm" @change="fetch">
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
        class="filter-date"
        @change="fetch"
      />
      <el-button type="primary" @click="fetch" :loading="loading">搜索</el-button>
      <el-button @click="resetFilter">重置</el-button>
          <el-button type="danger" plain :loading="clearing" @click="runCleanup">清空</el-button>
    </div>
    <div class="filter-bar mobile-only">
      <el-form label-position="top" class="mobile-filter-form">
        <el-form-item label="收件人"><el-input v-model="filter.keyword" placeholder="收件人邮箱" clearable @input="debouncedFetch" @clear="fetch" /></el-form-item>
        <el-form-item label="邮件类型">
          <el-select v-model="filter.email_type" placeholder="邮件类型" clearable class="full-width-control" @change="fetch">
            <el-option v-for="(label, value) in EMAIL_TYPE_MAP" :key="value" :label="label" :value="value" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filter.status" placeholder="状态" clearable class="full-width-control" @change="fetch">
            <el-option label="待发送" value="pending" />
            <el-option label="已发送" value="sent" />
            <el-option label="发送失败" value="failed" />
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
      title-field="to_email"
      empty-title="暂无数据"
    >
      <template #table>
        <div class="table-wrapper">
          <el-table v-loading="loading" :data="list" stripe border>
            <el-table-column prop="created_at" label="创建时间" width="180" />
            <el-table-column prop="to_email" label="收件人" width="200" />
            <el-table-column prop="subject" label="主题" min-width="200" show-overflow-tooltip />
            <el-table-column prop="email_type" label="类型" width="120">
              <template #default="{ row }">
                {{ getEmailTypeText(row.email_type) }}
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="getStatusColor(row.status)" size="small">{{ getStatusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="retry_count" label="重试" width="70" />
            <el-table-column prop="sent_at" label="发送时间" width="180" />
            <el-table-column prop="error_message" label="错误信息" min-width="180" show-overflow-tooltip />
          </el-table>
        </div>
      </template>
      <template #header="{ item }">
        <div class="mobile-log-title">{{ item.to_email || '-' }}</div>
        <div class="mobile-log-subtitle">{{ item.created_at || '-' }}</div>
      </template>
      <template #default="{ item }">
        <MobileLogFields>
          <div class="mobile-log-field">
            <span class="mobile-log-label">类型</span>
            <span class="mobile-log-value">{{ getEmailTypeText(item.email_type) }}</span>
          </div>
          <div class="mobile-log-field field-full">
            <span class="mobile-log-label">主题</span>
            <span class="mobile-log-value mobile-log-wrap">{{ item.subject || '-' }}</span>
          </div>
          <div class="mobile-log-field">
            <span class="mobile-log-label">状态</span>
            <span class="mobile-log-value">
              <el-tag :type="getStatusColor(item.status)" size="small">{{ getStatusText(item.status) }}</el-tag>
            </span>
          </div>
          <div class="mobile-log-field field-full" v-if="item.error_message">
            <span class="mobile-log-label">错误</span>
            <span class="mobile-log-value mobile-log-wrap">{{ item.error_message }}</span>
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
import { EMAIL_TYPE_MAP, getEmailTypeText, getEmailStatusText as getStatusText, getEmailStatusType as getStatusColor } from '@/utils/statusMaps'
import PaginationBar from '@/components/PaginationBar.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import MobileLogFields from '@/components/MobileLogFields.vue'

const {
  loading, list, total, page, pageSize, filter, isMobile, paginationLayout,
  fetchLogs: fetch, debouncedFetchLogs: debouncedFetch, resetFilter, onSizeChange,
  clearing, runCleanup
} = useLogListPage({
  fetcher: (params) => adminAPI.getEmailLogs(params),
  defaultFilter: () => ({ keyword: '', email_type: '', status: '', timeRange: null }),
  extraFilterKeys: ['email_type', 'status'],
  clearAction: () => adminAPI.cleanupData('email_queue'),
  clearLabel: '邮件日志'
})
</script>
<style scoped>
.filter-keyword,
.filter-select-sm,
.filter-select-md,
.filter-date {
  width: 100%;
  min-width: 0;
}
</style>
