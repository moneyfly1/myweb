<template>
  <div class="services-page">
    <div class="page-header">
      <div>
        <h2 class="page-title">验证码服务</h2>
        <p class="page-subtitle">一次性短信验证码 · 购买后自动分配号码，短信验证码会自动显示在订单里</p>
      </div>
      <div class="header-actions">
        <el-radio-group v-model="mode" size="small" class="mode-switch" @change="onModeChange">
          <el-radio-button label="once">一次性验证码</el-radio-button>
          <el-radio-button label="rent">租号</el-radio-button>
        </el-radio-group>
        <el-button :icon="Tickets" @click="scrollToOrders">我的订单</el-button>
        <el-button :icon="Refresh" :loading="loading" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <el-alert v-if="notice" :title="notice" type="warning" :closable="false" show-icon class="notice" />

    <div class="intro-card">
      <div class="intro-head">
        <div class="intro-title">
          <el-icon><Iphone /></el-icon>
          验证码服务是什么？
        </div>
        <el-button link type="primary" @click="introCollapsed = !introCollapsed">
          {{ introCollapsed ? '展开说明' : '收起说明' }}
        </el-button>
      </div>
      <div v-show="!introCollapsed" class="intro-body">
        <p class="intro-desc">
          用真实手机号帮你接收海外平台（WhatsApp / Telegram / Google / OpenAI 等）的<strong>短信验证码</strong>，
          适用于注册账号、登录验证等场景。号码由平台即时分配，一次性使用。
        </p>
        <div class="intro-steps">
          <div class="intro-step"><span class="step-no">1</span><div><b>选服务</b><p>左侧选择你要注册/登录的平台</p></div></div>
          <div class="intro-step"><span class="step-no">2</span><div><b>选国家</b><p>右侧选择国家（价格/库存不同）</p></div></div>
          <div class="intro-step"><span class="step-no">3</span><div><b>支付</b><p>支持码支付/支付宝，支付后自动分配号码</p></div></div>
          <div class="intro-step"><span class="step-no">4</span><div><b>收验证码</b><p>用该号码去目标平台注册 → 验证码自动显示在「我的订单」</p></div></div>
        </div>
        <div class="intro-tips">
          <span>💡 号码为一次性验证码用途，请勿用于违规场景；</span>
          <span>💡 购买后 10 分钟内未收到短信可取消；</span>
          <span>💡 平台无号/失败会自动退款到余额；</span>
          <span>💡 验证码收到后请点「完成」释放号码。</span>
        </div>
      </div>
    </div>

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
        <div v-loading="mode === 'rent' ? rentLoading : countriesLoading" class="country-list">
          <el-empty v-if="!selectedService" description="从左侧选择一个服务" :image-size="70" />
          <el-empty v-else-if="mode === 'once' && !countriesLoading && !filteredCountries.length" description="该服务暂无可用国家" :image-size="70" />
          <el-empty v-else-if="mode === 'rent' && !rentLoading && !filteredRentCountries.length" description="该服务暂无租号库存" :image-size="70" />
          <div v-else-if="mode === 'once'" class="country-table">
            <div class="country-row head">
              <span>国家</span>
              <span class="c-count sortable" @click="toggleSort('count')">库存 <span class="sort-arrow">{{ sortArrow('count') }}</span></span>
              <span class="c-price sortable" @click="toggleSort('price')">售价 <span class="sort-arrow">{{ sortArrow('price') }}</span></span>
              <span class="c-op"></span>
            </div>
            <div v-for="c in filteredCountries" :key="c.country_id" class="country-row">
              <span class="c-name">{{ c.country_name }}</span>
              <span class="c-count">{{ formatCount(c.count) }}</span>
              <span class="c-price">¥{{ formatPrice(c.price_cny) }}</span>
              <span class="c-op"><el-button type="primary" size="small" @click="openBuy(c)">购买</el-button></span>
            </div>
          </div>
          <div v-else class="country-table rent-table">
            <div class="country-row head rent-row">
              <span>国家</span>
              <span class="c-duration">租期</span>
              <span class="c-price">售价</span>
              <span class="c-op"></span>
            </div>
            <div v-for="c in filteredRentCountries" :key="c.country_id" class="country-row rent-row">
              <span class="c-name">{{ c.country_name }}</span>
              <span class="c-duration">
                <el-select v-model="rentSelection[c.country_id]" size="small" class="rent-select" @change="(v) => onRentDurationChange(c, v)">
                  <el-option
                    v-for="o in c.options"
                    :key="o.duration_hours"
                    :label="formatDuration(o.duration_hours) + ' · ¥' + formatPrice(o.price_cny)"
                    :value="o.duration_hours"
                  />
                </el-select>
              </span>
              <span class="c-price">¥{{ formatPrice(rentPrice(c)) }}</span>
              <span class="c-op"><el-button type="primary" size="small" @click="openRentBuy(c)">购买</el-button></span>
            </div>
          </div>
        </div>
      </section>
    </div>

    <div class="orders-section" id="my-orders">
      <div class="orders-head">
        <h3>我的订单</h3>
        <el-button size="small" :icon="Refresh" :loading="ordersLoading" @click="loadOrders">刷新</el-button>
      </div>
      <div v-loading="ordersLoading" class="orders-wrap">
        <el-empty v-if="!ordersLoading && !orders.length" description="暂无验证码订单" :image-size="70" />
        <el-table v-else-if="!isMobile" :data="orders" class="orders-table" stripe>
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
          <el-table-column label="到期" min-width="120">
            <template #default="{ row }">
              <span :class="{ 'text-danger': formatRemaining(row.expire_time) === '已到期' }">{{ formatRemaining(row.expire_time) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="时间" min-width="150" />
          <el-table-column label="操作" width="260" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.otp" size="small" @click="copy(row.otp)">复制验证码</el-button>
              <el-button v-else-if="row.phone_number" size="small" @click="copy(row.phone_number)">复制号码</el-button>
              <el-button v-if="row.activation_id" size="small" @click="openProlong(row)">续期</el-button>
              <el-button v-if="row.status === 'active'" size="small" @click="finishOrder(row)">完成</el-button>
              <el-button v-if="row.status === 'active'" size="small" type="danger" plain @click="cancelOrder(row)">取消</el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- 手机端：订单卡片（表格在手机上会挤成横条，看不到验证码） -->
        <div v-if="orders.length" class="orders-pager">
          <el-pagination
            v-model:current-page="orderPage"
            :page-size="orderPageSize"
            :total="orderTotal"
            layout="prev, pager, next, total"
            small
            @current-change="loadOrders"
          />
        </div>

        <div v-if="isMobile && orders.length" class="mob-orders">
          <div v-for="row in orders" :key="row.order_no" class="mob-order">
            <div class="mob-order-head">
              <div class="mob-order-title">{{ row.service_name }} <span class="muted">· {{ row.country_name }}</span></div>
              <el-tag size="small" :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
            </div>
            <div class="mob-order-line"><span class="k">号码</span><span class="mono">{{ row.phone_number || '—' }}</span></div>
            <div class="mob-order-line">
              <span class="k">验证码</span>
              <span v-if="row.otp" class="otp-code">{{ row.otp }}</span>
              <span v-else-if="row.status === 'active'" class="otp-waiting">— 等待短信</span>
              <span v-else class="muted">—</span>
            </div>
            <div class="mob-order-line" v-if="row.expire_time">
              <span class="k">到期</span>
              <span :class="{ 'text-danger': formatRemaining(row.expire_time) === '已到期' }">{{ formatRemaining(row.expire_time) }}（到期自动释放，不退款）</span>
            </div>
            <div class="mob-order-foot">
              <span class="mob-order-amount">¥{{ formatPrice(row.price_cny) }}</span>
              <span class="mob-order-time">{{ row.created_at }}</span>
            </div>
            <div class="mob-order-actions">
              <el-button v-if="row.otp" size="small" type="primary" plain @click="copy(row.otp)">复制验证码</el-button>
              <el-button v-else-if="row.phone_number" size="small" @click="copy(row.phone_number)">复制号码</el-button>
              <el-button v-if="row.status === 'active'" size="small" @click="finishOrder(row)">完成</el-button>
              <el-button v-if="row.status === 'active'" size="small" type="danger" plain @click="cancelOrder(row)">取消</el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <el-dialog v-model="buyVisible" title="购买验证码" width="460px" class="buy-dialog" align-center>
      <div class="buy-card">
        <div class="buy-hero">
          <div class="buy-service">{{ buyTarget?.service_name }}</div>
          <div class="buy-country">{{ buyTarget?.country_name }}</div>
          <div class="buy-country" v-if="buyTarget?.order_type === 'rent'">租期：{{ formatDuration(buyTarget?.duration_hours) }}</div>
          <div class="buy-amount"><span class="cur">¥</span>{{ formatPrice(buyTarget?.price_cny) }}</div>
          <div class="buy-amount-note">含服务费 · 购买后自动分配号码</div>
        </div>

        <div class="buy-section-title">支付方式</div>
        <div class="pay-grid">
          <div
            v-for="m in paymentMethods"
            :key="m.key"
            class="pay-card"
            :class="{ active: selectedPayment === m.key }"
            @click="selectedPayment = m.key"
          >
            <div class="pay-card-icon" :class="payIconClass(m.key)">
              <el-icon><Wallet /></el-icon>
            </div>
            <div class="pay-card-name">{{ m.name || m.key }}</div>
            <div class="pay-card-check" v-if="selectedPayment === m.key">
              <el-icon><Check /></el-icon>
            </div>
          </div>
        </div>
        <div v-if="!paymentMethods.length" class="pay-empty">暂无可用支付方式</div>
      </div>
      <template #footer>
        <div class="buy-footer">
          <el-button @click="buyVisible = false">取消</el-button>
          <el-button type="primary" :loading="creating" @click="confirmBuy">
            确认支付 ¥{{ formatPrice(buyTarget?.price_cny) }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog v-model="prolongVisible" title="号码续期" width="440px" class="prolong-dialog">
      <div class="prolong-box">
        <div class="prolong-row"><span>号码</span><b class="mono">{{ prolongTarget?.phone_number || '—' }}</b></div>
        <div class="prolong-row"><span>当前余额</span><b>¥{{ formatPrice(prolongBalance) }}</b></div>
        <div class="prolong-tip">续期使用余额支付，续期成功后号码可继续接收短信。</div>
        <div v-if="prolongLoading" class="prolong-loading">加载续期价格…</div>
        <div v-else class="prolong-options">
          <div v-for="o in prolongOptions" :key="o.duration_hours" class="prolong-option">
            <div class="po-left">
              <div class="po-duration">+{{ formatDuration(o.duration_hours) }}</div>
              <div class="po-price" v-if="o.price_known">¥{{ formatPrice(o.price_cny) }}</div>
              <div class="po-price muted" v-else>价格以平台为准</div>
            </div>
            <el-button type="primary" size="small" :disabled="!o.price_known" :loading="prolongSubmitting" @click="doProlong(o)">续期</el-button>
          </div>
        </div>
      </div>
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
import { Refresh, Search, Wallet, Check, Tickets, Iphone } from '@element-plus/icons-vue'
import api from '@/utils/api'
import { createPaymentQRCode } from '@/utils/payment'

const loading = ref(false)
const notice = ref('')
const introCollapsed = ref(typeof window !== 'undefined' && window.innerWidth <= 768)
const isMobile = ref(typeof window !== 'undefined' && window.innerWidth <= 768)
const scrollToOrders = () => {
  const el = document.getElementById('my-orders')
  if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
const services = ref([])
const serviceKeyword = ref('')
const selectedService = ref(null)
const mode = ref('once')
const rentCountries = ref([])
const rentSelection = ref({})
const rentLoading = ref(false)
const countries = ref([])
const countriesLoading = ref(false)
const countryKeyword = ref('')
const countrySort = ref('count_desc')
const sortArrow = (field) => {
  if (field === 'count') return countrySort.value === 'count_desc' ? '↓' : ''
  if (field === 'price') return countrySort.value === 'price_asc' ? '↑' : countrySort.value === 'price_desc' ? '↓' : ''
  return ''
}
const toggleSort = (field) => {
  if (field === 'count') {
    countrySort.value = 'count_desc'
  } else if (field === 'price') {
    countrySort.value = countrySort.value === 'price_asc' ? 'price_desc' : 'price_asc'
  }
  // 重新拉当前服务（带新排序）
  if (selectedService.value) selectService(selectedService.value)
}

const orders = ref([])
const ordersLoading = ref(false)
const orderPage = ref(1)
const orderPageSize = ref(20)
const orderTotal = ref(0)
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

// 号码续期
const prolongVisible = ref(false)
const prolongTarget = ref(null)
const prolongOptions = ref([])
const prolongBalance = ref(0)
const prolongLoading = ref(false)
const prolongSubmitting = ref(false)
const formatDuration = (h) => {
  const n = Number(h || 0)
  if (n >= 1440) return Math.round(n / 1440) + ' 天'
  if (n >= 24) return Math.round(n / 24) + ' 天'
  return n + ' 小时'
}
const openProlong = async (row) => {
  prolongTarget.value = row
  prolongVisible.value = true
  prolongOptions.value = []
  prolongBalance.value = 0
  prolongLoading.value = true
  try {
    const res = await api.get(`/services/orders/${row.order_no}/prolong-options`)
    const data = res.data?.data || res.data || {}
    prolongOptions.value = data.items || []
    prolongBalance.value = data.balance || 0
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '获取续期价格失败')
  } finally {
    prolongLoading.value = false
  }
}
const doProlong = async (opt) => {
  if (prolongSubmitting.value) return
  prolongSubmitting.value = true
  try {
    const res = await api.post(`/services/orders/${prolongTarget.value.order_no}/prolong`, { duration_hours: opt.duration_hours })
    ElMessage.success(res.data?.message || '续期成功')
    prolongVisible.value = false
    loadOrders()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '续期失败')
  } finally {
    prolongSubmitting.value = false
  }
}
let pollTimer = null

const filteredServices = computed(() => {
  const kw = serviceKeyword.value.trim().toLowerCase()
  if (!kw) return services.value
  return services.value.filter(s =>
    String(s.service_name || '').toLowerCase().includes(kw) ||
    String(s.service_code || '').toLowerCase().includes(kw)
  )
})

const filteredRentCountries = computed(() => {
  const kw = countryKeyword.value.trim().toLowerCase()
  const list = rentCountries.value || []
  if (!kw) return list
  return list.filter(c => String(c.country_name || '').toLowerCase().includes(kw))
})

const rentPrice = (c) => {
  const dur = rentSelection.value[c.country_id]
  const opt = (c.options || []).find(o => o.duration_hours === dur)
  return opt ? opt.price_cny : 0
}

const filteredCountries = computed(() => {
  const kw = countryKeyword.value.trim().toLowerCase()
  if (!kw) return countries.value
  return countries.value.filter(c => String(c.country_name || '').toLowerCase().includes(kw))
})

const formatPrice = v => Number(v || 0).toFixed(2)
// 支付方式图标配色：支付宝蓝、微信绿、码支付橙、其他主题色
const payIconClass = (key) => {
  const k = String(key || '').toLowerCase()
  if (k.includes('wx') || k.includes('wechat') || k.includes('weixin')) return 'icon-wechat'
  if (k.includes('codepay') || k.includes('码支付')) return 'icon-codepay'
  if (k.includes('alipay') || k.includes('支付宝')) return 'icon-alipay'
  return 'icon-default'
}
const formatCount = n => Number(n || 0).toLocaleString()
// 距离号码到期剩余时间（自动释放，不退款）
const formatRemaining = (expire) => {
  if (!expire) return '—'
  const t = new Date(String(expire).replace(' ', 'T')).getTime()
  if (!Number.isFinite(t)) return '—'
  const diff = t - Date.now()
  if (diff <= 0) return '已到期'
  const mins = Math.ceil(diff / 60000)
  if (mins >= 1440) return `剩 ${Math.floor(mins / 1440)} 天 ${Math.floor((mins % 1440) / 60)} 小时`
  if (mins >= 60) return `剩 ${Math.floor(mins / 60)} 小时 ${mins % 60} 分`
  return `剩 ${mins} 分钟`
}
const statusText = s => ({ pending: '待支付', active: '等待短信', completed: '已验证', cancelled: '已取消', failed: '已退款', expired: '到期释放' }[s] || s)
const statusType = s => ({ pending: 'info', active: 'warning', completed: 'success', cancelled: 'info', failed: 'danger', expired: 'info' }[s] || 'info')

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
  if (mode.value === 'rent') {
    await loadRentCountries(svc)
    return
  }
  countriesLoading.value = true
  try {
    const res = await api.get('/services/countries', { params: { service: svc.service_code, sort: countrySort.value, _t: Date.now() } })
    const data = res.data?.data || res.data || {}
    countries.value = data.items || []
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '加载国家列表失败')
  } finally {
    countriesLoading.value = false
  }
}

const onModeChange = async () => {
  countryKeyword.value = ''
  if (!selectedService.value) {
    if (services.value.length) selectService(services.value[0])
    return
  }
  if (mode.value === 'rent') {
    await loadRentCountries(selectedService.value)
  } else {
    await selectService(selectedService.value)
  }
  if (!paymentMethods.value.length) loadPaymentMethods()
}

const loadRentCountries = async (svc) => {
  rentLoading.value = true
  try {
    const params = { service: svc.service_code, _t: Date.now() }
    if (countrySort.value.startsWith('price')) params.sort = countrySort.value
    const res = await api.get('/services/rent/countries', { params })
    const data = res.data?.data || res.data || {}
    rentCountries.value = data.items || []
    const sel = {}
    for (const c of rentCountries.value) {
      if (c.options && c.options.length) sel[c.country_id] = c.options[0].duration_hours
    }
    rentSelection.value = sel
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '加载租号列表失败')
  } finally {
    rentLoading.value = false
  }
}

const onRentDurationChange = () => { /* 价格由 rentPrice 计算 */ }

const openRentBuy = (c) => {
  const dur = rentSelection.value[c.country_id]
  const opt = (c.options || []).find(o => o.duration_hours === dur)
  if (!opt) {
    ElMessage.warning('请选择租期')
    return
  }
  buyTarget.value = {
    service_code: selectedService.value?.service_code,
    service_name: selectedService.value?.service_name,
    country_id: c.country_id,
    country_name: c.country_name,
    price_cny: opt.price_cny,
    duration_hours: dur,
    order_type: 'rent'
  }
  buyVisible.value = true
  if (!paymentMethods.value.length) loadPaymentMethods()
}

const loadOrders = async (silent = false) => {
  if (!silent) ordersLoading.value = true
  try {
    const res = await api.get('/services/orders', { params: { page: orderPage.value, size: orderPageSize.value } })
    const data = res.data?.data || res.data || {}
    orders.value = data.items || []
    orderTotal.value = data.total || 0
  } catch (e) {
    if (!silent) ElMessage.error(e?.response?.data?.message || '加载订单失败')
  } finally {
    if (!silent) ordersLoading.value = false
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
    const isRent = buyTarget.value?.order_type === 'rent'
    const res = await api.post(isRent ? '/services/rent/orders' : '/services/orders', isRent ? {
      service_code: buyTarget.value.service_code,
      country_id: buyTarget.value.country_id,
      duration_hours: buyTarget.value.duration_hours,
      payment_method: selectedPayment.value
    } : {
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

// 订单列表自动刷新：有「已激活等短信」的订单时 3 秒刷一次，否则 10 秒一次；
// 页面切到后台（不可见）时跳过，避免无意义请求。
let ordersTimer = null
const startOrdersAutoRefresh = () => {
  stopOrdersAutoRefresh()
  ordersTimer = setInterval(async () => {
    if (typeof document !== 'undefined' && document.visibilityState !== 'visible') return
    if (ordersLoading.value) return
    const waiting = orders.value.some(o => o.status === 'active' && !o.otp)
    if (!waiting && Math.floor(Date.now() / 1000) % 10 !== 0) return
    await loadOrders(true)
  }, 3000)
}
const stopOrdersAutoRefresh = () => {
  if (ordersTimer) {
    clearInterval(ordersTimer)
    ordersTimer = null
  }
}

let serviceTimer = null
watch(serviceKeyword, () => {
  if (serviceTimer) clearTimeout(serviceTimer)
  serviceTimer = setTimeout(() => { loadServices() }, 400)
})

onMounted(async () => {
  await Promise.all([loadServices(), loadOrders(), loadPaymentMethods()])
  startOrdersAutoRefresh()
})
onUnmounted(() => {
  stopPolling()
  stopOrdersAutoRefresh()
})
</script>

<style scoped>
.services-page { max-width: 1200px; margin: 0 auto; padding: 20px 16px 40px; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 14px; }
.page-title { font-size: 22px; margin: 0; color: #303133; }
.page-subtitle { margin: 6px 0 0; color: #909399; font-size: 13px; }
.notice { margin-bottom: 14px; }
.intro-card { background: linear-gradient(135deg, #f0f7ff 0%, #fafcff 100%); border: 1px solid #d9ecff; border-radius: 10px; padding: 14px 16px; margin-bottom: 16px; }
.intro-head { display: flex; align-items: center; justify-content: space-between; }
.intro-title { display: flex; align-items: center; gap: 6px; font-size: 15px; font-weight: 600; color: #303133; }
.intro-body { margin-top: 10px; }
.intro-desc { font-size: 13px; color: #606266; line-height: 1.7; margin: 0 0 12px; }
.intro-steps { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; }
.intro-step { display: flex; gap: 8px; background: #fff; border: 1px solid #ebeef5; border-radius: 8px; padding: 10px 12px; }
.step-no { width: 22px; height: 22px; border-radius: 50%; background: #409eff; color: #fff; font-size: 12px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.intro-step b { font-size: 13px; color: #303133; }
.intro-step p { margin: 3px 0 0; font-size: 12px; color: #909399; line-height: 1.5; }
.intro-tips { display: flex; flex-wrap: wrap; gap: 8px 18px; margin-top: 12px; font-size: 12px; color: #909399; }
.header-actions { display: flex; gap: 8px; }
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
.sortable { cursor: pointer; user-select: none; }
.sortable:hover { color: #409eff; }
.sort-arrow { color: #409eff; font-size: 11px; }
.orders-pager { display: flex; justify-content: center; padding: 10px 0 4px; }
.text-danger { color: #f56c6c; }
.mode-switch { margin-right: 4px; }
.rent-row { grid-template-columns: 1fr 210px 90px 76px; }
.c-duration { display: flex; align-items: center; }
.rent-select { width: 100%; }
@media (max-width: 768px) {
  .mode-switch { width: 100%; display: flex; }
  .mode-switch :deep(.el-radio-button) { flex: 1; }
  .mode-switch :deep(.el-radio-button__inner) { width: 100%; }
  .rent-row { grid-template-columns: 1fr 130px 70px 60px; }
}
.orders-section { margin-top: 22px; }
.orders-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; }
.orders-head h3 { margin: 0; font-size: 16px; color: #303133; }
.orders-wrap { background: #fff; border: 1px solid #ebeef5; border-radius: 10px; padding: 4px; }
.mono { font-family: ui-monospace, Menlo, monospace; }
.otp-code { font-family: ui-monospace, Menlo, monospace; color: #67c23a; font-weight: 700; letter-spacing: 1px; }
.otp-waiting { color: #e6a23c; }
.muted { color: #c0c4cc; }
.buy-card { padding: 0 4px; }
.buy-hero { text-align: center; padding: 8px 0 18px; border-bottom: 1px dashed #ebeef5; }
.buy-service { font-size: 18px; font-weight: 600; color: #303133; }
.buy-country { font-size: 13px; color: #909399; margin-top: 4px; }
.buy-amount { margin-top: 14px; color: #f56c6c; font-weight: 700; font-size: 34px; line-height: 1; }
.buy-amount .cur { font-size: 20px; margin-right: 2px; }
.buy-amount-note { font-size: 12px; color: #c0c4cc; margin-top: 8px; }
.buy-section-title { font-size: 13px; color: #606266; font-weight: 600; margin: 16px 0 10px; }
.pay-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 10px; }
.pay-card { position: relative; display: flex; align-items: center; gap: 10px; border: 1px solid #e4e7ed; border-radius: 10px; padding: 12px; cursor: pointer; transition: all .15s; background: #fff; }
.pay-card:hover { border-color: #c6e2ff; background: #fafcff; }
.pay-card.active { border-color: #409eff; background: #ecf5ff; box-shadow: 0 0 0 2px rgba(64,158,255,.12); }
.pay-card-icon { width: 32px; height: 32px; border-radius: 8px; display: flex; align-items: center; justify-content: center; font-size: 17px; color: #fff; flex-shrink: 0; background: #409eff; }
.pay-card-icon.icon-alipay { background: #1677ff; }
.pay-card-icon.icon-wechat { background: #07c160; }
.pay-card-icon.icon-codepay { background: #ff7a00; }
.pay-card-icon.icon-default { background: #909399; }
.pay-card-name { font-size: 13px; color: #303133; line-height: 1.3; }
.pay-card.active .pay-card-name { color: #409eff; font-weight: 600; }
.pay-card-check { position: absolute; right: 8px; top: 8px; color: #409eff; font-size: 14px; }
.pay-empty { color: #909399; font-size: 13px; padding: 10px 0; }
.buy-footer { display: flex; justify-content: flex-end; gap: 10px; }
.pay-box { display: flex; flex-direction: column; align-items: center; gap: 10px; }
.qr-wrap { display: flex; flex-direction: column; align-items: center; gap: 8px; }
.qr-img { width: 220px; height: 220px; border: 1px solid #ebeef5; border-radius: 8px; }
.qr-tip { color: #909399; font-size: 13px; }
.pay-amount { color: #606266; }
.pay-amount b { color: #f56c6c; font-size: 20px; }
.pay-status { color: #909399; font-size: 13px; text-align: center; }
.pay-order { color: #c0c4cc; font-size: 12px; }
.prolong-box { display: flex; flex-direction: column; gap: 10px; }
.prolong-row { display: flex; justify-content: space-between; color: #606266; font-size: 14px; }
.prolong-tip { font-size: 12px; color: #909399; padding: 8px 0; border-bottom: 1px dashed #ebeef5; }
.prolong-loading { text-align: center; color: #909399; padding: 16px; }
.prolong-options { display: flex; flex-direction: column; gap: 8px; max-height: 320px; overflow-y: auto; }
.prolong-option { display: flex; align-items: center; justify-content: space-between; border: 1px solid #ebeef5; border-radius: 8px; padding: 10px 12px; }
.po-duration { font-size: 14px; color: #303133; font-weight: 600; }
.po-price { font-size: 12px; color: #f56c6c; margin-top: 2px; }
.po-price.muted { color: #c0c4cc; }
@media (max-width: 768px) {
  .services-page { padding: 10px 10px 24px; }
  .page-header { flex-direction: column; gap: 10px; }
  .page-title { font-size: 19px; }
  .page-subtitle { font-size: 12px; }
  .header-actions { width: 100%; }
  .header-actions .el-button { flex: 1; }
  .intro-card { padding: 12px; }
  .intro-desc { font-size: 12px; }
  .intro-steps { grid-template-columns: 1fr 1fr; gap: 8px; }
  .intro-step { padding: 8px 10px; }
  .intro-step b { font-size: 12px; }
  .intro-step p { font-size: 11px; }
  .intro-tips { gap: 6px; font-size: 11px; }

  .layout { display: block; }
  .panel { height: auto; border-radius: 12px; }
  .service-panel { margin-bottom: 12px; }
  .service-list { display: flex; overflow-x: auto; overflow-y: hidden; gap: 8px; padding: 10px; -webkit-overflow-scrolling: touch; scrollbar-width: none; }
  .service-list::-webkit-scrollbar { display: none; }
  .service-item { flex: 0 0 140px; flex-direction: column; align-items: flex-start; gap: 4px; padding: 10px 12px; margin: 0; border: 1px solid #ebeef5; border-radius: 10px; background: #fff; }
  .service-item.active { border-color: #409eff; background: #ecf5ff; }
  .service-name { font-size: 13px; max-width: 116px; }
  .service-meta { font-size: 11px; }
  .service-price { font-size: 11px; margin-left: 0; }
  .country-panel { height: 62vh; }
  .country-head { flex-wrap: wrap; }
  .country-title { font-size: 15px; }
  .country-search { width: 100%; margin-top: 6px; }
  .country-list { padding: 0 8px 10px; }
  .country-row { grid-template-columns: 1fr 62px 66px 58px; padding: 10px 6px; font-size: 13px; }
  .country-row.head { font-size: 11px; }
  .c-count { font-size: 11px; }
  .c-op :deep(.el-button) { padding: 5px 8px; font-size: 12px; }

  .orders-section { margin-top: 16px; }
  .orders-table { font-size: 12px; }
  .orders-wrap { overflow-x: auto; }
  .mob-orders { display: flex; flex-direction: column; gap: 10px; padding: 8px; }
  .mob-order { border: 1px solid #ebeef5; border-radius: 10px; padding: 12px; background: #fff; }
  .mob-order-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
  .mob-order-title { font-size: 14px; font-weight: 600; color: #303133; }
  .mob-order-line { display: flex; align-items: center; gap: 10px; font-size: 13px; color: #606266; padding: 3px 0; }
  .mob-order-line .k { color: #909399; font-size: 12px; width: 48px; flex-shrink: 0; }
  .mob-order-foot { display: flex; align-items: center; justify-content: space-between; margin-top: 8px; font-size: 12px; color: #909399; }
  .mob-order-amount { color: #f56c6c; font-weight: 700; font-size: 14px; }
  .mob-order-actions { display: flex; gap: 8px; margin-top: 10px; flex-wrap: wrap; }
  .mob-order-actions .el-button { margin-left: 0; }

  .buy-dialog :deep(.el-dialog) { width: 92% !important; margin: 0 auto; }
  .pay-grid { grid-template-columns: 1fr; }
  .pay-dialog :deep(.el-dialog) { width: 92% !important; }
  .qr-img { width: 190px; height: 190px; }
}
</style>
