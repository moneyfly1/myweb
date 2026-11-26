<template>
  <div class="service-orders-page">
    <div class="page-header">
      <div>
        <h2 class="page-title">验证码订单</h2>
        <p class="page-subtitle">客户购买的验证码服务订单（一次性 / 租号），可查看号码与验证码</p>
      </div>
      <el-button :icon="Refresh" :loading="loading" @click="load">刷新</el-button>
    </div>

    <div class="stats-row">
      <div class="stat-card"><div class="stat-value">{{ stats.total }}</div><div class="stat-label">总订单</div></div>
      <div class="stat-card"><div class="stat-value text-warning">{{ stats.active }}</div><div class="stat-label">已激活</div></div>
      <div class="stat-card"><div class="stat-value text-success">{{ stats.completed }}</div><div class="stat-label">已验证</div></div>
      <div class="stat-card"><div class="stat-value text-danger">{{ stats.failed }}</div><div class="stat-label">失败</div></div>
      <div class="stat-card"><div class="stat-value text-info">{{ stats.refunded }}</div><div class="stat-label">已退款</div></div>
    </div>

    <div class="filter-card">
      <el-input v-model="filters.q" placeholder="订单号 / 用户邮箱 / 号码 / 服务" clearable class="filter-item search" :prefix-icon="Search" />
      <el-select v-model="filters.status" placeholder="订单状态" clearable class="filter-item">
        <el-option label="待支付" value="pending" />
        <el-option label="已激活" value="active" />
        <el-option label="已验证" value="completed" />
        <el-option label="已取消" value="cancelled" />
        <el-option label="已退款/失败" value="failed" />
      </el-select>
      <el-select v-model="filters.payment_status" placeholder="支付状态" clearable class="filter-item">
        <el-option label="待支付" value="pending" />
        <el-option label="已支付" value="paid" />
        <el-option label="已退款" value="refunded" />
      </el-select>
      <el-select v-model="filters.order_type" placeholder="类型" clearable class="filter-item">
        <el-option label="一次性验证码" value="once" />
        <el-option label="租号" value="rent" />
      </el-select>
      <el-date-picker v-model="dateRange" type="daterange" range-separator="至" start-placeholder="开始日期" end-placeholder="结束日期" value-format="YYYY-MM-DD" class="filter-item date-range" />
      <el-button type="primary" @click="search">查询</el-button>
      <el-button @click="reset">重置</el-button>
    </div>

    <div v-loading="loading" class="table-card">
      <el-empty v-if="!loading && !items.length" description="暂无验证码订单" />
      <el-table v-else :data="items" stripe class="orders-table">
        <el-table-column prop="order_no" label="订单号" min-width="190" />
        <el-table-column label="用户" min-width="170">
          <template #default="{ row }"><div>{{ row.user_email || row.user_name }}</div></template>
        </el-table-column>
        <el-table-column label="类型" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.order_type === 'rent' ? 'warning' : 'success'">
              {{ row.order_type === 'rent' ? '租号' : '一次性' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="service_name" label="服务" min-width="110" />
        <el-table-column prop="country_name" label="国家" min-width="90" />
        <el-table-column label="号码" min-width="140">
          <template #default="{ row }"><span class="mono">{{ row.phone_number || '—' }}</span></template>
        </el-table-column>
        <el-table-column label="验证码" min-width="120">
          <template #default="{ row }">
            <span v-if="row.otp" class="otp-code">{{ row.otp }}</span>
            <span v-else-if="row.status === 'active'" class="otp-waiting">等待短信</span>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="金额" width="90">
          <template #default="{ row }">¥{{ Number(row.price_cny || 0).toFixed(2) }}</template>
        </el-table-column>
        <el-table-column label="支付" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="payTagType(row.payment_status)">{{ payText(row.payment_status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="statusTagType(row.status)">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" min-width="150" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.otp" size="small" @click="copy(row.otp)">复制验证码</el-button>
            <el-button size="small" @click="showDetail(row)">详情</el-button>
            <el-button v-if="row.status === 'active'" size="small" type="danger" plain @click="cancelOrder(row)">取消</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pager">
        <el-pagination v-model:current-page="page" :page-size="size" :total="total" layout="prev, pager, next, total" @current-change="load" />
      </div>
    </div>

    <el-dialog v-model="detailVisible" title="订单详情" width="560px">
      <div v-if="detail" class="detail-box">
        <div class="d-row"><span>订单号</span><b class="mono">{{ detail.order_no }}</b></div>
        <div class="d-row"><span>用户</span><b>{{ detail.user_email || detail.user_name }}</b></div>
        <div class="d-row"><span>类型</span><b>{{ detail.order_type === 'rent' ? '租号' : '一次性验证码' }}</b></div>
        <div class="d-row"><span>服务 / 国家</span><b>{{ detail.service_name }} · {{ detail.country_name }}</b></div>
        <div class="d-row" v-if="detail.order_type === 'rent'"><span>租期</span><b>{{ detail.rent_duration_hours }} 小时（到期 {{ detail.rent_end_time || '—' }}）</b></div>
        <div class="d-row"><span>号码</span><b class="mono">{{ detail.phone_number || '—' }}</b></div>
        <div class="d-row"><span>验证码</span><b class="otp-code">{{ detail.otp || '—' }}</b></div>
        <div class="d-row"><span>金额 / 支付</span><b>¥{{ Number(detail.price_cny || 0).toFixed(2) }} · {{ detail.payment_method || '-' }} ({{ payText(detail.payment_status) }})</b></div>
        <div class="d-row"><span>状态</span><b>{{ statusText(detail.status) }}</b></div>
        <div class="d-row" v-if="detail.refund_reason"><span>退款原因</span><b>{{ detail.refund_reason }}</b></div>
        <div class="d-row"><span>创建时间</span><b>{{ detail.created_at }}</b></div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Search } from '@element-plus/icons-vue'
import api from '@/utils/api'

const loading = ref(false)
const items = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const filters = reactive({ q: '', status: '', payment_status: '', order_type: '' })
const dateRange = ref([])
const stats = ref({ total: 0, active: 0, completed: 0, failed: 0, refunded: 0 })
const detailVisible = ref(false)
const detail = ref(null)

const statusText = s => ({ pending: '待支付', active: '已激活', completed: '已验证', cancelled: '已取消', failed: '已退款/失败' }[s] || s)
const statusTagType = s => ({ pending: 'info', active: 'warning', completed: 'success', cancelled: 'info', failed: 'danger' }[s] || 'info')
const payText = s => ({ pending: '待支付', paid: '已支付', refunded: '已退款' }[s] || s)
const payTagType = s => ({ pending: 'info', paid: 'success', refunded: 'danger' }[s] || 'info')

const loadStats = async () => {
  try {
    const res = await api.get('/admin/services/orders/stats')
    stats.value = res.data?.data || res.data || stats.value
  } catch (e) { /* ignore */ }
}

const load = async () => {
  loading.value = true
  try {
    const params = { page: page.value, size: size.value, ...filters }
    if (dateRange.value && dateRange.value.length === 2) {
      params.start_date = dateRange.value[0]
      params.end_date = dateRange.value[1]
    }
    const res = await api.get('/admin/services/orders', { params })
    const data = res.data?.data || res.data || {}
    items.value = data.items || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

const search = () => { page.value = 1; load() }
const reset = () => {
  filters.q = ''
  filters.status = ''
  filters.payment_status = ''
  filters.order_type = ''
  dateRange.value = []
  page.value = 1
  load()
}

const copy = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

const showDetail = (row) => { detail.value = row; detailVisible.value = true }

const cancelOrder = async (row) => {
  try {
    await ElMessageBox.confirm(`确认取消订单 ${row.order_no}？将调用平台取消号码。`, '取消订单', { type: 'warning' })
    await ElMessage.post(`/services/orders/${row.order_no}/cancel`).catch(() => {})
    ElMessage.success('已提交取消')
    load()
  } catch (e) { /* cancel */ }
}

onMounted(() => { load(); loadStats() })
</script>

<style scoped>
.service-orders-page { padding: 20px; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-title { margin: 0; font-size: 22px; color: #303133; }
.page-subtitle { margin: 6px 0 0; color: #909399; font-size: 13px; }
.stats-row { display: grid; grid-template-columns: repeat(5, 1fr); gap: 12px; margin-bottom: 14px; }
.stat-card { background: #fff; border: 1px solid #ebeef5; border-radius: 10px; padding: 12px 16px; }
.stat-value { font-size: 22px; font-weight: 700; color: #303133; }
.stat-label { font-size: 12px; color: #909399; margin-top: 4px; }
.text-warning { color: #e6a23c; }
.text-success { color: #67c23a; }
.text-danger { color: #f56c6c; }
.text-info { color: #909399; }
.filter-card { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; background: #fff; border: 1px solid #ebeef5; border-radius: 10px; padding: 12px; margin-bottom: 14px; }
.filter-item { width: 180px; }
.filter-item.search { flex: 1; min-width: 220px; max-width: 340px; }
.filter-item.date-range { width: 260px; }
.table-card { background: #fff; border: 1px solid #ebeef5; border-radius: 10px; padding: 4px; }
.mono { font-family: ui-monospace, Menlo, monospace; }
.otp-code { font-family: ui-monospace, Menlo, monospace; color: #67c23a; font-weight: 700; letter-spacing: 1px; }
.otp-waiting { color: #e6a23c; }
.muted { color: #c0c4cc; }
.pager { display: flex; justify-content: flex-end; padding: 10px; }
.detail-box { display: flex; flex-direction: column; gap: 10px; }
.d-row { display: flex; justify-content: space-between; gap: 16px; font-size: 14px; color: #606266; }
.d-row b { color: #303133; text-align: right; word-break: break-all; }
@media (max-width: 768px) {
  .service-orders-page { padding: 12px; }
  .stats-row { grid-template-columns: repeat(2, 1fr); }
  .filter-item, .filter-item.search, .filter-item.date-range { width: 100%; max-width: none; }
  .page-header { flex-direction: column; gap: 10px; }
}
</style>
