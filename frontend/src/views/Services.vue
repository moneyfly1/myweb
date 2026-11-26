<template>
  <div class="services-page">
    <div class="page-header">
      <div>
        <h2 class="page-title">验证码服务</h2>
        <p class="page-subtitle">
          一次性短信验证码，价格已含服务费
          <span v-if="catalog.multiplier && catalog.exchange_rate">
            （官方价 × {{ catalog.multiplier }} × 汇率 {{ catalog.exchange_rate }}）
          </span>
        </p>
      </div>
      <el-button :icon="Refresh" :loading="loading" @click="loadCatalog">刷新</el-button>
    </div>

    <el-alert v-if="catalogNotice" :title="catalogNotice" type="warning" :closable="false" show-icon class="notice" />

    <div class="filter-bar">
      <el-select v-model="filterCountry" placeholder="全部国家" clearable filterable class="filter-item">
        <el-option v-for="c in countryOptions" :key="c.id" :label="c.name" :value="c.id" />
      </el-select>
      <el-input v-model="filterKeyword" placeholder="搜索服务名称" clearable class="filter-item search" :prefix-icon="Search" />
      <el-radio-group v-model="activeTab" class="tab-group">
        <el-radio-button label="buy">购买服务</el-radio-button>
        <el-radio-button label="orders">我的订单</el-radio-button>
      </el-radio-group>
    </div>

    <template v-if="activeTab === 'buy'">
      <div v-loading="loading" class="service-grid">
        <el-empty v-if="!loading && filteredItems.length === 0" description="暂无可用服务" />
        <div v-for="item in filteredItems" :key="item.service_code + '-' + item.country_id" class="service-card">
          <div class="card-top">
            <div class="service-icon">
              <el-icon><Iphone /></el-icon>
            </div>
            <div class="service-meta">
              <div class="service-name">{{ item.service_name }}</div>
              <div class="service-country">
                <el-tag size="small" effect="plain">{{ item.country_name }}</el-tag>
                <span class="count">可用 {{ item.count }}</span>
              </div>
            </div>
          </div>
          <div class="card-bottom">
            <div class="price-box">
              <span class="price">¥{{ formatPrice(item.price_cny) }}</span>
              <span class="official">官方 ${{ item.official_price_usd.toFixed(2) }}</span>
            </div>
            <el-button type="primary" size="small" @click="openBuy(item)">购买</el-button>
          </div>
        </div>
      </div>
    </template>

    <template v-else>
      <div v-loading="ordersLoading" class="orders-wrap">
        <el-empty v-if="!ordersLoading && orders.length === 0" description="暂无验证码订单" />
        <el-table v-else :data="orders" class="orders-table" stripe>
          <el-table-column prop="service_name" label="服务" min-width="120" />
          <el-table-column prop="country_name" label="国家" min-width="100" />
          <el-table-column prop="phone_number" label="号码" min-width="140">
            <template #default="{ row }">{{ row.phone_number || '-' }}</template>
          </el-table-column>
          <el-table-column label="验证码" min-width="120">
            <template #default="{ row }">
              <span v-if="row.otp" class="otp-code">{{ row.otp }}</span>
              <span v-else-if="row.status === 'active'" class="otp-waiting">等待中…</span>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column label="金额" width="100">
            <template #default="{ row }">¥{{ formatPrice(row.price_cny) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <el-tag size="small" :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="时间" min-width="160" />
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.otp" size="small" @click="copy(row.otp)">复制验证码</el-button>
              <el-button v-if="row.status === 'active'" size="small" @click="finishOrder(row)">完成</el-button>
              <el-button v-if="row.status === 'active'" size="small" type="danger" plain @click="cancelOrder(row)">取消</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </template>

    <!-- 购买确认弹窗 -->
    <el-dialog v-model="buyVisible" title="购买验证码" width="440px" class="buy-dialog">
      <div class="buy-summary">
        <div class="buy-row"><span>服务</span><b>{{ current?.service_name }}</b></div>
        <div class="buy-row"><span>国家</span><b>{{ current?.country_name }}</b></div>
        <div class="buy-row"><span>官方价</span><span>${{ current?.official_price_usd?.toFixed(2) }}</span></div>
        <div class="buy-row"><span>售价</span><b class="buy-price">¥{{ formatPrice(current?.price_cny) }}</b></div>
        <div class="buy-row"><span>支付方式</span></div>
        <el-radio-group v-model="selectedPayment" class="pay-methods">
          <el-radio v-for="m in paymentMethods" :key="m.key" :label="m.key" border class="pay-method">
            {{ m.name || m.key }}
          </el-radio>
        </el-radio-group>
      </div>
      <template #footer>
        <el-button @click="buyVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="confirmBuy">确认支付</el-button>
      </template>
    </el-dialog>

    <!-- 支付弹窗 -->
    <el-dialog v-model="payVisible" title="扫码支付" width="400px" class="pay-dialog" @close="stopPolling">
      <div class="pay-box">
        <div v-if="qrCodeData" class="qr-wrap">
          <img :src="qrCodeData" alt="支付二维码" class="qr-img" />
          <div class="qr-tip">请使用支付宝/微信扫码支付</div>
        </div>
        <div v-else-if="payUrl" class="pay-link-box">
          <el-button type="primary" @click="openPayLink">前往支付页面</el-button>
        </div>
        <div class="pay-amount">应付：<b>¥{{ formatPrice(payAmount) }}</b></div>
        <div class="pay-status">{{ payStatusText }}</div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Iphone, Refresh, Search } from '@element-plus/icons-vue'
import api from '@/utils/api'
import { createPaymentQRCode } from '@/utils/payment'

const loading = ref(false)
const ordersLoading = ref(false)
const catalog = ref({ enabled: false, multiplier: 0, exchange_rate: 0, items: [] })
const catalogNotice = ref('')
const filterCountry = ref('')
const filterKeyword = ref('')
const activeTab = ref('buy')

const orders = ref([])
const paymentMethods = ref([])
const buyVisible = ref(false)
const creating = ref(false)
const current = ref(null)
const selectedPayment = ref('')

const payVisible = ref(false)
const payUrl = ref('')
const qrCodeData = ref('')
const payAmount = ref(0)
const payOrderNo = ref('')
const payStatusText = ref('等待支付中…')
let pollTimer = null

const countryOptions = computed(() => {
  const map = new Map()
  for (const it of catalog.value.items || []) {
    if (!map.has(it.country_id)) map.set(it.country_id, { id: it.country_id, name: it.country_name })
  }
  return Array.from(map.values()).sort((a, b) => a.name.localeCompare(b.name, 'zh'))
})

const filteredItems = computed(() => {
  const kw = filterKeyword.value.trim().toLowerCase()
  return (catalog.value.items || []).filter(it => {
    if (filterCountry.value && it.country_id !== filterCountry.value) return false
    if (kw && !String(it.service_name || '').toLowerCase().includes(kw) && !String(it.service_code || '').toLowerCase().includes(kw)) return false
    return true
  })
})

const formatPrice = v => Number(v || 0).toFixed(2)

const statusText = s => ({ pending: '待支付', active: '已激活', completed: '已完成', cancelled: '已取消', failed: '失败' }[s] || s)
const statusType = s => ({ pending: 'info', active: 'success', completed: 'success', cancelled: 'warning', failed: 'danger' }[s] || 'info')

const loadCatalog = async () => {
  loading.value = true
  try {
    const res = await api.get('/services/catalog')
    const data = res.data?.data || res.data || {}
    catalog.value = {
      enabled: !!data.enabled,
      multiplier: data.multiplier || 0,
      exchange_rate: data.exchange_rate || 0,
      items: data.items || []
    }
    catalogNotice.value = data.notice || ''
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '加载服务列表失败')
  } finally {
    loading.value = false
  }
}

const loadOrders = async () => {
  ordersLoading.value = true
  try {
    const res = await api.get('/services/orders', { params: { page: 1, size: 50 } })
    const data = res.data?.data || res.data || {}
    orders.value = data.items || []
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '加载订单失败')
  } finally {
    ordersLoading.value = false
  }
}

const loadPaymentMethods = async () => {
  try {
    const res = await api.get('/payment-methods/active')
    const data = res.data?.data || res.data || []
    paymentMethods.value = Array.isArray(data) ? data : []
    if (paymentMethods.value.length) selectedPayment.value = paymentMethods.value[0].key
  } catch (e) {
    paymentMethods.value = []
  }
}

const openBuy = item => {
  current.value = item
  buyVisible.value = true
  if (!paymentMethods.value.length) loadPaymentMethods()
}

const confirmBuy = async () => {
  if (!selectedPayment.value) {
    ElMessage.warning('请选择支付方式')
    return
  }
  creating.value = true
  try {
    const res = await api.post('/services/orders', {
      service_code: current.value.service_code,
      country_id: current.value.country_id,
      payment_method: selectedPayment.value
    })
    const data = res.data?.data || res.data || {}
    buyVisible.value = false
    payOrderNo.value = data.order_no
    payAmount.value = data.price_cny
    payUrl.value = data.payment_url || ''
    qrCodeData.value = ''
    if (payUrl.value) {
      try {
        qrCodeData.value = await createPaymentQRCode(payUrl.value, { width: 220 })
      } catch (e) {
        qrCodeData.value = ''
      }
    }
    payStatusText.value = '等待支付中…'
    payVisible.value = true
    startPolling()
    loadOrders()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '创建订单失败')
  } finally {
    creating.value = false
  }
}

const openPayLink = () => {
  if (payUrl.value) window.open(payUrl.value, '_blank')
}

const startPolling = () => {
  stopPolling()
  let attempts = 0
  pollTimer = setInterval(async () => {
    attempts++
    try {
      const res = await api.get(`/services/orders/${payOrderNo.value}`)
      const data = res.data?.data || res.data || {}
      if (data.status === 'active' || data.status === 'completed') {
        payStatusText.value = data.phone_number ? `已激活，号码：${data.phone_number}` : '已激活'
        stopPolling()
        loadOrders()
      } else if (data.status === 'failed') {
        payStatusText.value = '购买失败，已自动退款到余额'
        stopPolling()
        loadOrders()
      }
    } catch (e) { /* 忽略轮询错误 */ }
    if (attempts > 100) stopPolling()
  }, 3000)
}

const stopPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

const copy = async text => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

const finishOrder = async row => {
  try {
    await ElMessageBox.confirm('确认已收到验证码并完成该订单？', '完成订单', { type: 'info' })
    await api.post(`/services/orders/${row.order_no}/finish`)
    ElMessage.success('已完成')
    loadOrders()
  } catch (e) { /* 取消 */ }
}

const cancelOrder = async row => {
  try {
    await ElMessageBox.confirm('取消后将释放号码（平台会退款给我们），确认取消？', '取消订单', { type: 'warning' })
    await api.post(`/services/orders/${row.order_no}/cancel`)
    ElMessage.success('已取消')
    loadOrders()
  } catch (e) { /* 取消 */ }
}

onMounted(async () => {
  await Promise.all([loadCatalog(), loadOrders(), loadPaymentMethods()])
})
onUnmounted(stopPolling)
</script>

<style scoped>
.services-page { max-width: 1200px; margin: 0 auto; padding: 20px 16px 40px; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 16px; }
.page-title { font-size: 22px; margin: 0; color: #303133; }
.page-subtitle { margin: 6px 0 0; color: #909399; font-size: 13px; }
.notice { margin-bottom: 16px; }
.filter-bar { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; margin-bottom: 18px; }
.filter-item { width: 200px; }
.filter-item.search { flex: 1; min-width: 200px; max-width: 360px; }
.tab-group { margin-left: auto; }
.service-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 14px; min-height: 120px; }
.service-card { background: #fff; border: 1px solid #ebeef5; border-radius: 10px; padding: 14px; display: flex; flex-direction: column; gap: 12px; transition: box-shadow .2s, transform .2s; }
.service-card:hover { box-shadow: 0 6px 18px rgba(0,0,0,.08); transform: translateY(-2px); }
.card-top { display: flex; gap: 10px; align-items: center; }
.service-icon { width: 38px; height: 38px; border-radius: 10px; background: var(--theme-primary, #409eff); color: #fff; display: flex; align-items: center; justify-content: center; font-size: 20px; flex-shrink: 0; }
.service-meta { min-width: 0; }
.service-name { font-weight: 600; color: #303133; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.service-country { display: flex; align-items: center; gap: 8px; margin-top: 4px; }
.count { color: #909399; font-size: 12px; }
.card-bottom { display: flex; align-items: center; justify-content: space-between; }
.price-box { display: flex; flex-direction: column; }
.price { color: #f56c6c; font-weight: 700; font-size: 18px; }
.official { color: #909399; font-size: 12px; text-decoration: line-through; }
.orders-wrap { background: #fff; border: 1px solid #ebeef5; border-radius: 10px; padding: 4px; }
.otp-code { font-family: ui-monospace, Menlo, monospace; color: #67c23a; font-weight: 700; }
.otp-waiting { color: #e6a23c; }
.buy-summary { display: flex; flex-direction: column; gap: 10px; }
.buy-row { display: flex; justify-content: space-between; color: #606266; }
.buy-price { color: #f56c6c; font-size: 18px; }
.pay-methods { display: flex; flex-direction: column; gap: 8px; }
.pay-method { margin-right: 0 !important; }
.pay-box { display: flex; flex-direction: column; align-items: center; gap: 12px; }
.qr-wrap { display: flex; flex-direction: column; align-items: center; gap: 8px; }
.qr-img { width: 220px; height: 220px; border: 1px solid #ebeef5; border-radius: 8px; }
.qr-tip { color: #909399; font-size: 13px; }
.pay-amount { color: #606266; }
.pay-amount b { color: #f56c6c; font-size: 20px; }
.pay-status { color: #909399; font-size: 13px; }
@media (max-width: 768px) {
  .services-page { padding: 12px 10px 30px; }
  .page-header { flex-direction: column; }
  .filter-item, .filter-item.search { width: 100%; max-width: none; }
  .tab-group { margin-left: 0; width: 100%; display: flex; }
  .tab-group :deep(.el-radio-button) { flex: 1; }
  .tab-group :deep(.el-radio-button__inner) { width: 100%; }
  .service-grid { grid-template-columns: 1fr; }
  .orders-table { font-size: 12px; }
}
</style>
