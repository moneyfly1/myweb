<template>
  <div class="services-page">
    <div class="page-header">
      <div>
        <h2 class="page-title">验证码服务</h2>
        <p class="page-subtitle">一次性短信验证码 · 购买后自动分配号码，短信验证码会自动显示在订单里</p>
      </div>
      <el-button :icon="Refresh" :loading="loading" @click="refreshAll">刷新</el-button>
    </div>

    <el-alert v-if="notice" :title="notice" type="warning" :closable="false" show-icon class="notice" />

    <div class="layout">
      <aside class="panel service-panel">
        <div class="panel-head">
          <el-input v-model="serviceKeyword" placeholder="搜索服务（WhatsApp / Telegram…）" clearable :prefix-icon="Search" size="small" />
        </div>
        <div v-loading="loading" class="service-list">
          <el-empty v-if="!loading && !filteredServices.length" description="暂无可用服务" :image-size="60" />
          <div
            v-for="s in filteredServices"
            :key="s.service_code"
            class="service-item"
            :class="{ active: selectedService && selectedService.service_code === s.service_code }"
            @click="selectService(s)"
          >
            <div class="service-main">
              <div class="service-name">{{ s.service_name }}</div>
              <div class="service-meta">{{ s.country_count }} 个国家可用</div>
            </div>
            <div class="service-price">¥{{ formatPrice(s.min_price_cny) }} 起</div>
          </div>
        </div>
      </aside>

      <section class="panel country-panel">
        <div class="panel-head country-head">
          <div>
            <div class="country-title">{{ selectedService ? selectedService.service_name + ' · 选择国家' : '选择国家' }}</div>
            <div class="country-sub">{{ selectedService ? '价格已含服务费，购买后自动分配号码' : '从左侧选择服务' }}</div>
          </div>
          <el-input
            v-if="selectedService"
            v-model="countryKeyword"
            placeholder="搜索国家"
            clearable
            size="small"
            class="country-search"
            :prefix-icon="Search"
          />
        </div>
        <div v-loading="countriesLoading" class="country-list">
          <el-empty v-if="!selectedService" description="从左侧选择一个服务" :image-size="70" />
          <el-empty v-else-if="!countriesLoading && !filteredCountries.length" description="该服务暂无可用国家" :image-size="70" />
          <div v-else class="country-table">
            <div class="country-row head">
              <span>国家</span><span class="c-count">库存</span><span class="c-price">售价</span><span class="c-op"></span>
            </div>
            <div v-for="c in filteredCountries" :key="c.country_id" class="country-row">
              <span class="c-name">{{ c.country_name }}</span>
              <span class="c-count">{{ formatCount(c.count) }}</span>
              <span class="c-price">¥{{ formatPrice(c.price_cny) }}</span>
              <span class="c-op"><el-button type="primary" size="small" @click="openBuy(c)">购买</el-button></span>
            </div>
          </div>
        </div>
      </section>
    </div>

    <div class="orders-section">
      <div class="orders-head">
        <h3>我的订单</h3>
        <el-button size="small" :icon="Refresh" :loading="ordersLoading" @click="loadOrders">刷新</el-button>
      </div>
      <div v-loading="ordersLoading" class="orders-wrap">
        <el-empty v-if="!ordersLoading && !orders.length" description="暂无验证码订单" :image-size="70" />
        <el-table v-else :data="orders" class="orders-table" stripe>
          <el-table-column prop="service_name" label="服务" min-width="120" />
          <el-table-column prop="country_name" label="国家" min-width="100" />
          <el-table-column label="号码" min-width="150">
            <template #default="{ row }"><span class="mono">{{ row.phone_number || '—' }}</span></template>
          </el-table-column>
          <el-table-column label="验证码" min-width="140">
            <template #default="{ row }">
              <span v-if="row.otp" class="otp-code">{{ row.otp }}</span>
              <span v-else-if="row.status === 'active'" class="otp-waiting">— 等待短信</span>
              <span v-else class="muted">—</span>
            </template>
          </el-table-column>
          <el-table-column label="金额" width="90">
            <template #default="{ row }">¥{{ formatPrice(row.price_cny) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <el-tag size="small" :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="时间" min-width="150" />
          <el-table-column label="操作" width="190" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.otp" size="small" @click="copy(row.otp)">复制验证码</el-button>
              <el-button v-else-if="row.phone_number" size="small" @click="copy(row.phone_number)">复制号码</el-button>
              <el-button v-if="row.status === 'active'" size="small" @click="finishOrder(row)">完成</el-button>
              <el-button v-if="row.status === 'active'" size="small" type="danger" plain @click="cancelOrder(row)">取消</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <el-dialog v-model="buyVisible" title="购买验证码" width="440px" class="buy-dialog">
      <div class="buy-summary">
        <div class="buy-row"><span>服务</span><b>{{ buyTarget?.service_name }}</b></div>
        <div class="buy-row"><span>国家</span><b>{{ buyTarget?.country_name }}</b></div>
        <div class="buy-row"><span>售价</span><b class="buy-price">¥{{ formatPrice(buyTarget?.price_cny) }}</b></div>
        <div class="buy-row buy-methods-label"><span>支付方式</span></div>
        <el-radio-group v-model="selectedPayment" class="pay-methods">
          <el-radio v-for="m in paymentMethods" :key="m.key" :label="m.key" border class="pay-method">
            {{ m.name || m.key }}
          </el-radio>
        </el-radio-group>
      </div>
      <template #footer>
        <el-button @click="buyVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="confirmBuy">确认支付 ¥{{ formatPrice(buyTarget?.price_cny) }}</el-button>
      </template>
    </el-dialog>

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
        <div v-if="payOrderNo" class="pay-order">订单号：{{ payOrderNo }}</div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Search } from '@element-plus/icons-vue'
import api from '@/utils/api'
import { createPaymentQRCode } from '@/utils/payment'

const loading = ref(false)
const notice = ref('')
const services = ref([])
const serviceKeyword = ref('')
const selectedService = ref(null)
const countries = ref([])
const countriesLoading = ref(false)
const countryKeyword = ref('')

const orders = ref([])
const ordersLoading = ref(false)
const paymentMethods = ref([])
const buyVisible = ref(false)
const creating = ref(false)
const buyTarget = ref(null)
const selectedPayment = ref('')

const payVisible = ref(false)
const payUrl = ref('')
const qrCodeData = ref('')
const payAmount = ref(0)
const payOrderNo = ref('')
const payStatusText = ref('等待支付中…')
let pollTimer = null

const filteredServices = computed(() => {
  const kw = serviceKeyword.value.trim().toLowerCase()
  if (!kw) return services.value
  return services.value.filter(s =>
    String(s.service_name || '').toLowerCase().includes(kw) ||
    String(s.service_code || '').toLowerCase().includes(kw)
  )
})

const filteredCountries = computed(() => {
  const kw = countryKeyword.value.trim().toLowerCase()
  if (!kw) return countries.value
  return countries.value.filter(c => String(c.country_name || '').toLowerCase().includes(kw))
})

const formatPrice = v => Number(v || 0).toFixed(2)
const formatCount = n => Number(n || 0).toLocaleString()
const statusText = s => ({ pending: '待支付', active: '已激活', completed: '已验证', cancelled: '已取消', failed: '已退款' }[s] || s)
const statusType = s => ({ pending: 'info', active: 'warning', completed: 'success', cancelled: 'info', failed: 'danger' }[s] || 'info')

const loadServices = async () => {
  loading.value = true
  try {
    const params = { _t: Date.now() }
    if (serviceKeyword.value.trim()) params.q = serviceKeyword.value.trim()
    const res = await api.get('/services/services', { params })
    const data = res.data?.data || res.data || {}
    services.value = data.items || []
    notice.value = data.notice || ''
    if (!selectedService.value && services.value.length) {
      selectService(services.value[0])
    }
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '加载服务列表失败')
  } finally {
    loading.value = false
  }
}

const selectService = async (svc) => {
  selectedService.value = svc
  countryKeyword.value = ''
  countries.value = []
  countriesLoading.value = true
  try {
    const res = await api.get('/services/countries', { params: { service: svc.service_code, _t: Date.now() } })
    const data = res.data?.data || res.data || {}
    countries.value = data.items || []
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '加载国家列表失败')
  } finally {
    countriesLoading.value = false
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
    if (paymentMethods.value.length && !selectedPayment.value) selectedPayment.value = paymentMethods.value[0].key
  } catch (e) {
    paymentMethods.value = []
  }
}

const openBuy = (country) => {
  buyTarget.value = {
    service_code: selectedService.value?.service_code,
    service_name: selectedService.value?.service_name,
    country_id: country.country_id,
    country_name: country.country_name,
    price_cny: country.price_cny
  }
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
      service_code: buyTarget.value.service_code,
      country_id: buyTarget.value.country_id,
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
    payStatusText.value = '等待支付中…支付成功后自动分配号码'
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
      if (data.status === 'completed' || (data.status === 'active' && data.otp)) {
        payStatusText.value = `验证码：${data.otp}（已复制）`
        copy(data.otp)
        stopPolling()
        loadOrders()
      } else if (data.status === 'active') {
        payStatusText.value = `已分配号码：${data.phone_number || ''}，等待短信…`
        loadOrders()
      } else if (data.status === 'failed' || data.payment_status === 'refunded') {
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
  if (!text) return
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

const refreshAll = async () => {
  await Promise.all([loadServices(), loadOrders()])
}

let serviceTimer = null
watch(serviceKeyword, () => {
  if (serviceTimer) clearTimeout(serviceTimer)
  serviceTimer = setTimeout(() => { loadServices() }, 400)
})

onMounted(async () => {
  await Promise.all([loadServices(), loadOrders(), loadPaymentMethods()])
})
onUnmounted(stopPolling)
</script>

<style scoped>
.services-page { max-width: 1200px; margin: 0 auto; padding: 20px 16px 40px; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 14px; }
.page-title { font-size: 22px; margin: 0; color: #303133; }
.page-subtitle { margin: 6px 0 0; color: #909399; font-size: 13px; }
.notice { margin-bottom: 14px; }
.layout { display: grid; grid-template-columns: 280px 1fr; gap: 14px; }
.panel { background: #fff; border: 1px solid #ebeef5; border-radius: 10px; display: flex; flex-direction: column; height: 620px; overflow: hidden; }
.panel-head { padding: 12px; border-bottom: 1px solid #ebeef5; }
.service-list { flex: 1; overflow-y: auto; padding: 6px; }
.service-item { display: flex; align-items: center; justify-content: space-between; padding: 10px 12px; border-radius: 8px; cursor: pointer; margin-bottom: 4px; transition: background .15s; }
.service-item:hover { background: #f5f7fa; }
.service-item.active { background: #ecf5ff; }
.service-item.active .service-name, .service-item.active .service-meta, .service-item.active .service-price { color: #409eff; }
.service-main { min-width: 0; }
.service-name { font-size: 14px; color: #303133; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.service-meta { font-size: 12px; color: #909399; margin-top: 3px; }
.service-price { font-size: 12px; color: #909399; flex-shrink: 0; margin-left: 8px; }
.country-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.country-title { font-size: 16px; font-weight: 600; color: #303133; }
.country-sub { font-size: 12px; color: #909399; margin-top: 4px; }
.country-search { width: 200px; }
.country-list { flex: 1; overflow-y: auto; padding: 0 12px 12px; }
.country-table { width: 100%; }
.country-row { display: grid; grid-template-columns: 1fr 120px 100px 88px; align-items: center; padding: 12px 8px; border-bottom: 1px solid #f2f3f5; font-size: 14px; }
.country-row.head { color: #909399; font-size: 12px; padding: 10px 8px; border-bottom: 1px solid #ebeef5; position: sticky; top: 0; background: #fff; }
.c-name { color: #303133; }
.c-count { color: #909399; font-size: 12px; }
.c-price { color: #f56c6c; font-weight: 700; }
.c-op { text-align: right; }
.orders-section { margin-top: 22px; }
.orders-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; }
.orders-head h3 { margin: 0; font-size: 16px; color: #303133; }
.orders-wrap { background: #fff; border: 1px solid #ebeef5; border-radius: 10px; padding: 4px; }
.mono { font-family: ui-monospace, Menlo, monospace; }
.otp-code { font-family: ui-monospace, Menlo, monospace; color: #67c23a; font-weight: 700; letter-spacing: 1px; }
.otp-waiting { color: #e6a23c; }
.muted { color: #c0c4cc; }
.buy-summary { display: flex; flex-direction: column; gap: 10px; }
.buy-row { display: flex; justify-content: space-between; color: #606266; }
.buy-price { color: #f56c6c; font-size: 20px; }
.buy-methods-label { margin-top: 4px; }
.pay-methods { display: flex; flex-direction: column; gap: 8px; }
.pay-method { margin-right: 0 !important; }
.pay-box { display: flex; flex-direction: column; align-items: center; gap: 10px; }
.qr-wrap { display: flex; flex-direction: column; align-items: center; gap: 8px; }
.qr-img { width: 220px; height: 220px; border: 1px solid #ebeef5; border-radius: 8px; }
.qr-tip { color: #909399; font-size: 13px; }
.pay-amount { color: #606266; }
.pay-amount b { color: #f56c6c; font-size: 20px; }
.pay-status { color: #909399; font-size: 13px; text-align: center; }
.pay-order { color: #c0c4cc; font-size: 12px; }
@media (max-width: 768px) {
  .services-page { padding: 12px 10px 30px; }
  .page-header { flex-direction: column; }
  .layout { grid-template-columns: 1fr; }
  .panel { height: auto; }
  .service-panel { height: 320px; }
  .country-panel { height: 60vh; }
  .country-search { width: 120px; }
  .country-row { grid-template-columns: 1fr 70px 80px 70px; font-size: 13px; }
  .orders-table { font-size: 12px; }
}
</style>
