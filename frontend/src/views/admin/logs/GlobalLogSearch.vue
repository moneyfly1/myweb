<template>
  <div class="global-log-search">
    <!-- 搜索条：一个输入框，跨全部日志类型 -->
    <div class="gls-bar">
      <el-input
        v-model="keyword"
        size="small"
        class="gls-bar__input"
        clearable
        placeholder="搜索用户：用户名 / 邮箱 / 用户ID / 订阅 token / IP"
        @keyup.enter="search(1)"
        @clear="reset"
      >
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>
      <el-button size="small" type="primary" :loading="loading" @click="search(1)">全库搜索</el-button>
      <!-- 重置：清空关键词并退出搜索状态，回到原来的分类日志视图 -->
      <el-button v-if="searched" size="small" @click="reset">重置</el-button>
      <el-select v-if="searched" v-model="sources" size="small" class="gls-bar__src" multiple
        collapse-tags placeholder="全部类型" @change="search(1)">
        <el-option v-for="s in sourceOptions" :key="s.value" :label="s.label" :value="s.value" />
      </el-select>
    </div>

    <!-- 命中概览 -->
    <div v-if="searched" class="gls-summary">
      <span>「{{ lastQuery }}」命中 <b>{{ total }}</b> 条</span>
      <template v-for="(v, k) in bySource" :key="k">
        <span class="an-dot">·</span>
        <el-tag size="small" effect="plain">{{ sourceLabel(k) }} {{ v }}</el-tag>
      </template>
      <span class="gls-summary__spacer"></span>
      <el-button v-if="total" size="small" link type="primary" @click="exportCsv">导出 CSV</el-button>
    </div>

    <!-- 结果表（手机端自动变卡片） -->
    <ResponsiveDataView
      v-if="searched"
      :data="items"
      :fields="mobileFields"
      :loading="loading"
      id-field="row_key"
      title-field="source_label"
      empty-title="没有匹配的日志"
      empty-description="换个关键词试试（用户名 / 邮箱 / 用户ID / token / IP）。"
    >
      <template #table>
        <el-table :data="items" size="small" v-loading="loading" class="an-table">
          <el-table-column prop="time" label="时间" width="150" />
          <el-table-column label="类型" width="92">
            <template #default="{ row }">
              <el-tag size="small" effect="plain" :type="tagType(row.source)">{{ row.source_label }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="用户" min-width="150">
            <template #default="{ row }">
              <div class="an-user">
                <span class="an-user__name">{{ row.username || (row.user_id ? `用户#${row.user_id}` : row.email || '—') }}</span>
                <span class="an-sub">{{ row.email || '' }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="action" label="动作/类型" min-width="140" show-overflow-tooltip />
          <el-table-column prop="detail" label="详情" min-width="260" show-overflow-tooltip />
          <el-table-column prop="extra" label="附加" min-width="120" show-overflow-tooltip />
          <el-table-column prop="ip" label="IP" width="130" show-overflow-tooltip />
        </el-table>
      </template>
      <template #header="{ item }">
        <div class="gls-card-head">
          <el-tag size="small" effect="plain" :type="tagType(item.source)">{{ item.source_label }}</el-tag>
          <span class="an-sub">{{ item.time }}</span>
        </div>
        <div class="an-user">
          <span class="an-user__name">{{ item.username || (item.user_id ? `用户#${item.user_id}` : item.email || '—') }}</span>
          <span v-if="sublineOf(item)" class="an-sub">{{ sublineOf(item) }}</span>
        </div>
      </template>
      <template #field-action="{ item }"><span class="an-kv">动作 <b>{{ item.action || '—' }}</b></span></template>
      <template #field-detail="{ item }"><span class="an-sub">{{ item.detail || '—' }}</span></template>
    </ResponsiveDataView>

    <div v-if="searched && total > 0" class="an-pager">
      <PaginationBar :current-page="page" :page-size="limit" :total="total"
        :page-sizes="[20, 50, 100, 200]"
        @update:current-page="search" @update:page-size="onLimitChange" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { Search } from '@element-plus/icons-vue'
import { api } from '@/utils/api'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import { apiErrorMessage } from '@/utils/format'

/// 日志全局搜索：一个搜索框把这个用户的**全部**日志（注册/订阅/余额/佣金/
/// 重置/邮件/管理员操作/登录）一次搜出来，按时间倒序合并。
/// 需求（2026-10-19）：原来要在 7 个 Tab 里分别搜，查一个问题得搜 7 次。
/// searching = true 表示"正在看全局搜索结果"（父组件据此隐藏分类 Tab）
const emit = defineEmits(['update:searching'])

const keyword = ref('')
const lastQuery = ref('')
const searched = ref(false)
const loading = ref(false)
const items = ref([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)
const bySource = ref({})
const sources = ref([])

const sourceOptions = [
  { value: 'registration', label: '注册' },
  { value: 'subscription', label: '订阅' },
  { value: 'balance', label: '余额' },
  { value: 'commission', label: '佣金' },
  { value: 'reset', label: '重置订阅' },
  { value: 'email', label: '邮件' },
  { value: 'audit', label: '管理员操作' },
  { value: 'login', label: '登录' }
]

const sourceLabel = (k) => (sourceOptions.find(s => s.value === k) || {}).label || k

const tagType = (s) => ({
  registration: 'success',
  subscription: 'primary',
  balance: 'warning',
  commission: 'warning',
  reset: 'info',
  email: 'info',
  audit: 'danger',
  login: 'primary'
}[s] || 'info')

/// 卡片副标题：主标题用的就是邮箱时（邮件日志没有用户名）不再重复显示邮箱，
/// 改为显示 IP；有用户名时才把邮箱放副标题。
const sublineOf = (it) => {
  const name = (it.username || '').trim()
  const email = (it.email || '').trim()
  if (!name) return it.ip || ''
  if (email && email.toLowerCase() !== name.toLowerCase()) return email
  return it.ip || ''
}

const mobileFields = [
  { key: 'action', label: '动作' },
  { key: 'detail', label: '详情', fullWidth: true }
]

const search = async (p = 1) => {
  const q = (keyword.value || '').trim()
  if (!q) {
    ElMessage.warning('请输入用户名 / 邮箱 / 用户ID / 订阅 token 或 IP')
    return
  }
  page.value = p
  loading.value = true
  searched.value = true
  lastQuery.value = q
  emit('update:searching', true)
  try {
    const res = await api.get('/admin/logs/global-search', {
      params: {
        q, page: page.value, limit: limit.value,
        sources: sources.value.length ? sources.value.join(',') : undefined
      }
    })
    const data = res.data?.data || {}
    // 给每行一个稳定 key（不同来源 id 会重复）
    items.value = (data.items || []).map((it, i) => ({ ...it, row_key: `${it.source}-${it.id}-${i}` }))
    total.value = data.total || 0
    bySource.value = data.by_source || {}
  } catch (e) {
    ElMessage.error(apiErrorMessage(e, '搜索失败'))
    items.value = []
    total.value = 0
    bySource.value = {}
  } finally {
    loading.value = false
  }
}

const onLimitChange = (size) => {
  limit.value = size
  search(1)
}

/// 重置：清空关键词/来源过滤并退出搜索状态 —— 父组件据此恢复分类 Tab 视图。
/// 没有这个出口时，搜索完只能刷新页面才能回到原来的日志分类（用户反馈）。
const reset = () => {
  keyword.value = ''
  searched.value = false
  items.value = []
  total.value = 0
  bySource.value = {}
  lastQuery.value = ''
  sources.value = []
  page.value = 1
  emit('update:searching', false)
}

const exportCsv = () => {
  let csv = '\uFEFF'
  csv += `日志全局搜索,${lastQuery.value}\n`
  csv += '时间,类型,用户,邮箱,动作,详情,附加,IP\n'
  items.value.forEach(r => {
    const esc = (v) => `"${String(v ?? '').replace(/"/g, '""')}"`
    csv += [r.time, r.source_label, r.username || '', r.email || '', r.action || '',
      r.detail || '', r.extra || '', r.ip || ''].map(esc).join(',') + '\n'
  })
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `日志搜索_${lastQuery.value}_${Date.now()}.csv`
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<style scoped>
.global-log-search {
  margin-bottom: 14px;
  padding: 10px 12px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  background: #fafbfc;
}
.gls-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.gls-bar__input { flex: 1 1 260px; max-width: 460px; }
.gls-bar__src { width: 200px; }
.gls-summary {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 8px;
  font-size: 12.5px;
  color: #606266;
}
.gls-summary b { color: #303133; }
.gls-summary__spacer { flex: 1 1 auto; }
.gls-card-head { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.an-dot { color: #dcdfe6; }
.an-user { display: flex; flex-direction: column; line-height: 1.3; min-width: 0; }
.an-user__name { font-weight: 600; color: #303133; }
.an-sub { font-size: 12px; color: #909399; }
.an-kv { font-size: 12.5px; color: #606266; }
.an-table :deep(.el-table__cell) { padding: 6px 0; }
.an-pager { display: flex; justify-content: flex-end; margin-top: 10px; }

@media (max-width: 768px) {
  .global-log-search { padding: 8px; }
  .gls-bar__input, .gls-bar__src { width: 100%; max-width: none; flex: 1 1 100%; }
  .gls-bar .el-button { width: 100%; }
  /* 手机上「全库搜索 / 重置」各占一半，避免按钮叠成两大块 */
  .gls-bar .el-button { width: calc(50% - 4px); flex: 0 0 calc(50% - 4px); }
  .gls-summary { font-size: 12px; }
  .gls-summary__spacer { flex: 1 1 100%; }
  .an-pager { justify-content: center; }
}
</style>
