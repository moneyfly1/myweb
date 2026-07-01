<template>
  <AppDrawer
    v-model="drawerVisible"
    title="升级设备数量"
    size="500px"
    mobile-size="100%"
    :loading="upgradeLoading"
    class="upgrade-drawer"
    @open="handleUpgradeDialogOpen"
  >
    <div class="upgrade-content" v-if="subscription">
      <section class="upgrade-hero">
        <div>
          <div class="hero-eyebrow">设备扩容</div>
          <h3>按需增加可用设备</h3>
          <p>系统会根据当前订阅剩余时间计算费用，也可以同时延长到期时间。</p>
        </div>
        <div class="hero-metric">
          <span>{{ currentDeviceLimit }}</span>
          <small>当前设备</small>
        </div>
      </section>

      <section class="upgrade-panel">
        <div class="panel-title">
          <span>订阅概览</span>
          <small>剩余 {{ remainingDays }} 天</small>
        </div>
        <div class="summary-grid">
          <div class="summary-item">
            <span class="summary-label">当前设备</span>
            <strong>{{ currentDeviceLimit }}</strong>
          </div>
          <div class="summary-item highlight">
            <span class="summary-label">升级后</span>
            <strong>{{ targetDeviceLimit }}</strong>
          </div>
          <div class="summary-item" :class="{ highlight: upgradeForm.additionalDays > 0 }">
            <span class="summary-label">延长时间</span>
            <strong>{{ upgradeForm.additionalDays > 0 ? upgradeForm.additionalDays + ' 天' : '无' }}</strong>
          </div>
        </div>
      </section>

      <section class="upgrade-panel upgrade-main-panel">
        <div class="panel-title">
          <span>升级内容</span>
          <small>至少增加 1 个设备</small>
        </div>

        <!-- 设备数量 — 大卡片 Stepper -->
        <div class="form-item-block device-block">
          <div class="form-row-label">
            <span>增加设备数量</span>
            <em>升级后共 <strong>{{ targetDeviceLimit }}</strong> 个设备</em>
          </div>
          <div class="device-stepper-card">
            <button
              class="stepper-btn stepper-btn-minus"
              @click="changeDeviceCount(-1)"
              :disabled="upgradeForm.additionalDevices <= 1"
              aria-label="减少设备"
            >
              <el-icon><Minus /></el-icon>
            </button>
            <div class="stepper-display">
              <span class="stepper-number">{{ upgradeForm.additionalDevices }}</span>
              <span class="stepper-unit">个设备</span>
            </div>
            <button
              class="stepper-btn stepper-btn-plus"
              @click="changeDeviceCount(1)"
              aria-label="增加设备"
            >
              <el-icon><Plus /></el-icon>
            </button>
          </div>
          <!-- 设备数量进度条 -->
          <div class="device-dot-progress" v-if="targetDeviceLimit <= 30">
            <div class="dot-labels">
              <span>当前 {{ currentDeviceLimit }} 个</span>
              <span>升级后 {{ targetDeviceLimit }} 个</span>
            </div>
            <div class="dot-track">
              <span
                v-for="i in targetDeviceLimit"
                :key="i"
                class="dot"
                :class="i <= currentDeviceLimit ? 'dot-existing' : 'dot-new'"
              />
            </div>
          </div>
          <div class="device-bar-progress" v-else>
            <div class="bar-labels">
              <span>当前 {{ currentDeviceLimit }} 个</span>
              <span>升级后 {{ targetDeviceLimit }} 个</span>
            </div>
            <div class="bar-track">
              <div class="bar-fill bar-existing" :style="barProgressStyle" />
              <div class="bar-fill bar-new" :style="barProgressStyle" />
            </div>
          </div>
        </div>

        <!-- 延长到期时间 — 视觉月份卡片 -->
        <div class="form-item-block duration-block">
          <div class="form-row-label">
            <span>延长到期时间</span>
            <em>{{ upgradeForm.additionalDays > 0 ? `+${upgradeForm.additionalDays} 天（约 ${additionalMonths} 个月）` : '可选，也可以不延长' }}</em>
          </div>
          <div class="month-cards-grid">
            <button
              v-for="opt in monthCardOptions"
              :key="opt.days"
              class="month-card"
              :class="{ 'is-active': upgradeForm.additionalDays === opt.days, 'is-zero': opt.days === 0 }"
              @click="selectAdditionalDays(opt.days)"
            >
              <span class="month-card-num">{{ opt.label }}</span>
              <span class="month-card-days">{{ opt.sub }}</span>
            </button>
          </div>
          <!-- 到期日预览 -->
          <div class="expire-preview" v-if="subscription?.expire_time">
            <div class="expire-preview-row">
              <span class="expire-label">
                <el-icon><Timer /></el-icon>
                到期日
              </span>
              <span class="expire-old">{{ formatDate(subscription.expire_time) }}</span>
              <el-icon class="expire-arrow"><Right /></el-icon>
              <span class="expire-new">{{ formatDate(newExpireDate) }}</span>
            </div>
            <div class="expire-preview-badge" v-if="upgradeForm.additionalDays > 0">
              <el-icon><Timer /></el-icon>
              延长 +{{ upgradeForm.additionalDays }} 天
            </div>
          </div>
        </div>
      </section>

      <section class="upgrade-panel cost-panel" v-if="upgradeCost > 0">
        <div class="panel-title">
          <span>费用明细</span>
          <small>自动应用等级折扣</small>
        </div>
        <div class="cost-list">
          <div class="cost-row">
            <span>升级费用</span>
            <strong>¥{{ upgradeCost.toFixed(2) }}</strong>
          </div>
          <div class="cost-row discount-row" v-if="levelDiscount > 0">
            <span>等级折扣</span>
            <strong>-¥{{ levelDiscount.toFixed(2) }}</strong>
          </div>
          <div class="cost-row total-row">
            <span>应付金额</span>
            <strong>¥{{ finalAmount.toFixed(2) }}</strong>
          </div>
        </div>
      </section>

      <section class="upgrade-panel payment-method" v-if="finalAmount > 0 || upgradeForm.additionalDevices >= 1">
        <div class="panel-title">
          <span>支付方式</span>
          <small>余额 ¥{{ userBalance.toFixed(2) }}</small>
        </div>
        <div class="balance-info">
          <el-icon><Wallet /></el-icon>
          <span>账户余额</span>
          <strong>¥{{ userBalance.toFixed(2) }}</strong>
        </div>
        <div v-if="!availableUpgradePaymentMethods || availableUpgradePaymentMethods.length === 0" class="payment-loading-text">
          正在加载支付方式...
        </div>
        <el-radio-group v-model="paymentMethod" @change="handlePaymentMethodChange" class="payment-radio-list" v-else>
          <el-radio class="payment-radio-card" label="balance" :disabled="userBalance <= 0 || (finalAmount > 0 && userBalance < finalAmount)">
            <span class="pay-title">余额支付</span>
            <span v-if="finalAmount > 0 && userBalance >= finalAmount" class="pay-status success">余额充足</span>
            <span v-else-if="finalAmount > 0 && userBalance > 0" class="pay-status danger">还需 ¥{{ (finalAmount - userBalance).toFixed(2) }}</span>
          </el-radio>
          <template v-for="method in availableUpgradePaymentMethods" :key="method.key">
            <el-radio
              v-if="method && method.key && method.key !== 'balance'"
              class="payment-radio-card"
              :label="method.key"
            >
              <span class="pay-title">{{ method.name || method.key }}</span>
              <span class="pay-status">在线支付</span>
            </el-radio>
          </template>
        </el-radio-group>
      </section>
    </div>
    <template #footer>
      <div class="drawer-footer">
        <div class="footer-amount">
          <span>应付</span>
          <strong>¥{{ finalAmount.toFixed(2) }}</strong>
        </div>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button
          type="primary"
          @click="confirmUpgrade"
          :loading="upgradeLoading"
          :disabled="!upgradeForm.additionalDevices || upgradeForm.additionalDevices < 1"
        >
          {{ finalAmount > 0 ? '确认升级并支付' : '确认升级' }}
        </el-button>
      </div>
    </template>
  </AppDrawer>
  <AppDialog
    v-model="paymentQRVisible"
    title="扫码支付"
    width="520px"
    mobile-width="92%"
    class="payment-qr-dialog"
    :loading="false"
  >
    <div class="payment-qr-container" v-if="upgradeOrder">
      <div class="payment-summary-card">
        <div class="summary-header">
          <div>
            <div class="summary-label">支付金额</div>
            <div class="summary-amount">¥{{ parseFloat(upgradeOrder.actual_payment_amount || upgradeOrder.amount || 0).toFixed(2) }}</div>
          </div>
          <div class="summary-badge" v-if="upgradeOrder.additional_devices">
            +{{ upgradeOrder.additional_devices }}个设备
          </div>
        </div>
        <div class="summary-meta">
          <div class="meta-item">
            <span class="meta-key">订单号</span>
            <span class="meta-value">{{ upgradeOrder.order_no }}</span>
          </div>
          <div class="meta-item" v-if="upgradeOrder.additional_devices">
            <span class="meta-key">升级内容</span>
            <span class="meta-value">增加 {{ upgradeOrder.additional_devices }} 个设备</span>
          </div>
        </div>
      </div>

      <div class="qr-panel">
        <div class="qr-panel-header">
          <h4 v-if="isPaymentPageUrl">请在页面中完成支付</h4>
          <h4 v-else>请使用支付宝扫码</h4>
          <p>支付完成后会自动刷新升级结果</p>
        </div>
        <div class="qr-code-wrapper" :class="{ 'iframe-mode': isPaymentPageUrl }">
          <div v-if="isPaymentPageUrl" class="payment-page-iframe">
            <iframe
              :src="paymentUrl"
              frameborder="0"
              scrolling="auto"
              @load="startPaymentStatusCheck"
            ></iframe>
          </div>
          <div v-else-if="paymentQRCode" class="qr-code">
            <img
              :src="paymentQRCode.startsWith('data:') ? paymentQRCode : (paymentQRCode + '?t=' + Date.now())"
              alt="支付二维码"
              title="支付宝二维码"
              @error="onImageError"
              @load="onImageLoad"
            />
          </div>
          <div v-else class="qr-loading">
            <el-icon class="is-loading" :size="32"><Loading /></el-icon>
            <p>正在生成二维码...</p>
          </div>
        </div>
        <div class="payment-tips" v-if="!isPaymentPageUrl">
          <p class="tip-text"><el-icon><InfoFilled /></el-icon><span>请使用支付宝扫码支付</span></p>
        </div>
        <div class="payment-actions-container" v-if="isMobile && paymentUrl">
          <el-button
            type="success"
            size="large"
            class="payment-btn alipay-btn"
            @click="openAlipayApp"
          >
            <el-icon class="btn-icon"><Wallet /></el-icon>
            跳转支付宝App支付
          </el-button>
        </div>
      </div>
    </div>
  </AppDialog>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { Loading, Wallet, InfoFilled, Plus, Minus, Right, Timer } from '@element-plus/icons-vue'
import { orderAPI, parsePaymentMethods, useApi, userAPI, userLevelAPI, cachedAPI, pendingPaymentStorage } from '@/utils/api'
import { getRemainingDays as getRemainingDaysUtil } from '@/utils/date'
import { safeNavigate } from '@/utils/safeOpen'
import { usePaymentStatusPolling } from '@/composables/usePaymentStatusPolling'
import { createQRCodeDataURL } from '@/utils/qrcode'
import AppDrawer from '@/components/AppDrawer.vue'
import AppDialog from '@/components/AppDialog.vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  subscription: {
    type: Object,
    default: null
  },
  onSuccess: {
    type: Function,
    default: null
  }
})

const emit = defineEmits(['update:modelValue'])

const api = useApi()
const drawerVisible = computed({
  get: () => props.modelValue,
  set: value => emit('update:modelValue', value)
})

const upgradeLoading = ref(false)
const userBalance = ref(0)
const upgradeForm = ref({ additionalDevices: 5, additionalDays: 0 })
const monthOptions = ref([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12])
const upgradeCost = ref(0)
const levelDiscount = ref(0)
const finalAmount = ref(0)
const paymentMethod = ref('alipay')
const availableUpgradePaymentMethods = ref([])
const upgradeOrder = ref(null)
const paymentQRVisible = ref(false)
const paymentQRCode = ref(null)
const paymentUrl = ref('')
const paymentStatusRequest = ref(null)
let paymentManualVisibilityHandler = null
const isMobile = ref(typeof window !== 'undefined' ? window.innerWidth <= 768 : false)
let resizeRafId = null
const currentDeviceLimit = computed(() => props.subscription?.device_limit || props.subscription?.maxDevices || 0)
const targetDeviceLimit = computed(() => currentDeviceLimit.value + (upgradeForm.value.additionalDevices || 0))
const remainingDays = computed(() => getRemainingDays(props.subscription))
const additionalMonths = computed(() => Math.round((upgradeForm.value.additionalDays || 0) / 30))
const isPaymentPageUrl = computed(() => {
  if (!paymentUrl.value) return false
  const url = String(paymentUrl.value).toLowerCase()
  return url.includes('payapi/pay/payment') ||
         url.includes('submit.php') ||
         (url.startsWith('http') && !url.includes('qrcode') && !url.includes('qr.alipay') && !url.startsWith('weixin://') && !url.startsWith('wxp://'))
})

// 月份卡片选项
const monthCardOptions = computed(() => [
  { days: 30, label: '1 个月', sub: '30 天' },
  { days: 90, label: '3 个月', sub: '90 天' },
  { days: 180, label: '6 个月', sub: '180 天' },
  { days: 360, label: '12 个月', sub: '360 天' },
  { days: 60, label: '2 个月', sub: '60 天' },
  { days: 120, label: '4 个月', sub: '120 天' },
  { days: 240, label: '8 个月', sub: '240 天' },
  { days: 0, label: '不延长', sub: '仅升级设备' }
])

// 到期日预览
const newExpireDate = computed(() => {
  if (!props.subscription?.expire_time) return ''
  const current = new Date(props.subscription.expire_time)
  if (isNaN(current.getTime())) return ''
  const days = upgradeForm.value.additionalDays || 0
  const next = new Date(current.getTime() + days * 86400000)
  return next.toISOString()
})

// 设备进度条百分比 (用于 >30 设备时的条形图)
const existingPercent = computed(() => {
  if (!targetDeviceLimit.value) return 100
  const percentage = Math.round((currentDeviceLimit.value / targetDeviceLimit.value) * 100)
  return Math.min(Math.max(percentage, 0), 100)
})
const newPercent = computed(() => 100 - existingPercent.value)
const barProgressStyle = computed(() => ({
  '--existing-device-percent': `${existingPercent.value}%`,
  '--new-device-percent': `${newPercent.value}%`
}))

// 格式化日期显示
const formatDate = (dateStr) => {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return ''
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

// 选择延长天数
const selectAdditionalDays = (days) => {
  upgradeForm.value.additionalDays = days
  calculateUpgradeCost()
}

const handleResize = () => {
  if (resizeRafId !== null || typeof window === 'undefined') return
  resizeRafId = window.requestAnimationFrame(() => {
    resizeRafId = null
    isMobile.value = window.innerWidth <= 768
  })
}

const getRemainingDays = (subscription) => getRemainingDaysUtil(subscription?.expire_time)

const loadUpgradePaymentMethods = async () => {
  try {
    const response = await api.get('/payment-methods/active')
    const methods = parsePaymentMethods(response)
    availableUpgradePaymentMethods.value = methods
    if (methods.length > 0) {
      const firstMethod = methods.find(m => m.key && m.key !== 'balance') || methods[0]
      if (firstMethod?.key) {
        paymentMethod.value = firstMethod.key
      }
    }
  } catch (error) {
    ElMessage.error('加载支付方式失败: ' + (error.response?.data?.message || error.message))
    availableUpgradePaymentMethods.value = []
  }
}

const fetchUserInfo = async () => {
  try {
    const userResponse = await userAPI.getUserInfo()
    if (userResponse?.data?.success) {
      userBalance.value = parseFloat(userResponse.data.data.balance || 0)
    }
    try {
      await userLevelAPI.getMyLevel()
    } catch (e) {}
  } catch (error) {
    console.error('获取用户信息失败:', error)
  }
}

const handleUpgradeDialogOpen = async () => {
  upgradeForm.value = { additionalDevices: 1, additionalDays: 0 }
  upgradeCost.value = 0
  levelDiscount.value = 0
  finalAmount.value = 0
  paymentMethod.value = ''
  await Promise.all([loadUpgradePaymentMethods(), fetchUserInfo()])
  setTimeout(() => {
    calculateUpgradeCost()
    setTimeout(() => {
      if (userBalance.value >= finalAmount.value && finalAmount.value > 0) {
        paymentMethod.value = 'balance'
      } else if (availableUpgradePaymentMethods.value.length > 0) {
        paymentMethod.value = availableUpgradePaymentMethods.value[0]?.key || 'alipay'
      } else {
        paymentMethod.value = 'alipay'
      }
    }, 300)
  }, 500)
}

const calculateUpgradeCost = async () => {
  if (!props.subscription || !upgradeForm.value.additionalDevices) {
    upgradeCost.value = 0
    finalAmount.value = 0
    return
  }
  try {
    const response = await orderAPI.upgradeDevices({
      additional_devices: upgradeForm.value.additionalDevices,
      additional_days: upgradeForm.value.additionalDays || 0,
      payment_method: paymentMethod.value,
      use_balance: false,
      preview_only: true
    })
    if (response?.data?.success) {
      upgradeCost.value = parseFloat(response.data.data.upgrade_cost || 0)
      levelDiscount.value = parseFloat(response.data.data.level_discount || 0)
      finalAmount.value = parseFloat(response.data.data.final_amount ?? response.data.data.amount ?? 0)
    }
  } catch (error) {
    console.error('计算升级费用失败:', error)
  }
}

const handlePaymentMethodChange = () => {
  if (finalAmount.value > 0) calculateUpgradeCost()
}

const showPaymentQRCode = async (order) => {
  const url = String(order?.payment_url || order?.payment_qr_code || '').trim()
  if (!url) {
    paymentQRCode.value = ''
    paymentUrl.value = ''
    ElMessage.error(order?.payment_error || order?.note || '支付链接生成失败，请稍后重试')
    return
  }

  paymentUrl.value = url
  if (isPaymentPageUrl.value) {
    paymentQRCode.value = ''
    paymentQRVisible.value = true
    startPaymentStatusCheck()
    return
  }
  try {
    const qrOptions = {
      width: isMobile.value ? 200 : 256,
      margin: 2,
      color: { dark: '#000000', light: '#FFFFFF' },
      errorCorrectionLevel: 'M'
    }
    paymentQRCode.value = await createQRCodeDataURL(url, qrOptions)
    paymentQRVisible.value = true
    startPaymentStatusCheck()
  } catch (error) {
    console.error('生成支付二维码失败:', error)
    ElMessage.error(error.message || '二维码生成失败，请刷新页面重试')
  }
}

const checkUpgradeOrderStatus = async (isAutoCheck = false) => {
  if (!upgradeOrder.value?.order_no) return
  if (paymentStatusRequest.value) return paymentStatusRequest.value
  paymentStatusRequest.value = (async () => {
    try {
      const response = await orderAPI.getOrderStatus(upgradeOrder.value.order_no)
      if (response?.data?.success && response.data.data?.status === 'paid') {
        cleanupPaymentStatusCheck()
        paymentQRVisible.value = false
        ElMessage.success('支付成功，设备已升级！')
        pendingPaymentStorage.clear()
        await cachedAPI.refreshUserState()
        await props.onSuccess?.()
        window.dispatchEvent(new CustomEvent('subscription-updated'))
        window.dispatchEvent(new CustomEvent('user-info-updated'))
        upgradeForm.value = { additionalDevices: 5, additionalDays: 0 }
        upgradeCost.value = 0
        finalAmount.value = 0
        upgradeOrder.value = null
        paymentQRCode.value = null
      } else if (response?.data?.success && ['cancelled', 'failed', 'expired'].includes(response.data.data?.status)) {
        cleanupPaymentStatusCheck()
        paymentQRVisible.value = false
        pendingPaymentStorage.clear()
        ElMessage.warning('升级订单已取消或支付失败')
      } else if (!isAutoCheck) {
        ElMessage.warning('订单尚未支付，请完成支付')
      }
    } catch (error) {
      if (!isAutoCheck) ElMessage.error('检查订单状态失败: ' + (error.response?.data?.message || error.message))
    } finally {
      paymentStatusRequest.value = null
    }
  })()
  return paymentStatusRequest.value
}

const confirmUpgrade = async () => {
  if (!upgradeForm.value.additionalDevices || upgradeForm.value.additionalDevices < 1) {
    ElMessage.warning('请选择要增加的设备数量（至少1个）')
    return
  }
  try {
    upgradeLoading.value = true
    const response = await orderAPI.upgradeDevices({
      additional_devices: upgradeForm.value.additionalDevices,
      additional_days: upgradeForm.value.additionalDays || 0,
      payment_method: paymentMethod.value,
      use_balance: paymentMethod.value === 'balance'
    })
    if (response?.data?.success) {
      const data = response.data.data
      if (data.status === 'paid') {
        ElMessage.success('设备数量升级成功！')
        pendingPaymentStorage.clear()
        await cachedAPI.refreshUserState()
        drawerVisible.value = false
        await props.onSuccess?.()
        window.dispatchEvent(new CustomEvent('subscription-updated'))
        window.dispatchEvent(new CustomEvent('user-info-updated'))
      } else {
        const paymentUrlVal = data.payment_url || data.payment_qr_code
        if (!paymentUrlVal) {
          ElMessage.error('支付链接生成失败，请稍后重试')
          return
        }
        const paymentMethodName = data.payment_method || paymentMethod.value
        const isYipay = paymentMethodName && (
          paymentMethodName.includes('yipay') ||
          paymentMethodName.includes('易支付') ||
          paymentMethodName.includes('codepay') ||
          paymentMethodName.includes('码支付')
        )
        if (isYipay) {
          upgradeOrder.value = {
            ...data,
            additional_devices: upgradeForm.value.additionalDevices,
            additional_days: upgradeForm.value.additionalDays || 0
          }
          pendingPaymentStorage.save(upgradeOrder.value.order_no, 'device_upgrade')
          ElMessage.info('正在跳转到支付页面...')
          safeNavigate(paymentUrlVal, { allowAppProtocols: true })
          startPaymentStatusCheck()
        } else {
          upgradeOrder.value = {
            ...data,
            additional_devices: upgradeForm.value.additionalDevices,
            additional_days: upgradeForm.value.additionalDays || 0
          }
          pendingPaymentStorage.save(upgradeOrder.value.order_no, 'device_upgrade')
          drawerVisible.value = false
          await showPaymentQRCode(data)
        }
      }
    } else {
      ElMessage.error(response?.data?.message || '升级设备数量失败')
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '升级设备数量失败')
  } finally {
    upgradeLoading.value = false
  }
}

const openAlipayApp = () => {
  if (!paymentUrl.value) {
    ElMessage.error('支付链接不存在')
    return
  }
  const alipayAppUrl = `alipays://platformapi/startapp?saId=10000007&qrcode=${encodeURIComponent(paymentUrl.value)}`
  cleanupPaymentManualWatcher()
  paymentManualVisibilityHandler = async () => {
    if (document.visibilityState === 'visible' && upgradeOrder.value?.order_no) {
      await checkUpgradeOrderStatus()
      cleanupPaymentManualWatcher()
    }
  }
  document.addEventListener('visibilitychange', paymentManualVisibilityHandler)
  safeNavigate(alipayAppUrl, { allowAppProtocols: true })
  setTimeout(() => ElMessage.info('如果未跳转到支付宝，请使用支付宝扫描上方二维码完成支付'), 3000)
}

const onImageError = () => ElMessage.error('二维码加载失败')
const onImageLoad = () => {}

const cleanupPaymentManualWatcher = () => {
  if (paymentManualVisibilityHandler && typeof document !== 'undefined') {
    document.removeEventListener('visibilitychange', paymentManualVisibilityHandler)
    paymentManualVisibilityHandler = null
  }
}

const { startPolling: startPaymentStatusCheck, clearPolling: cleanupPaymentStatusCheck } = usePaymentStatusPolling({
  intervalMs: 3000,
  timeoutMs: 30 * 60 * 1000,
  shouldPoll: () => !!upgradeOrder.value?.order_no,
  poll: () => checkUpgradeOrderStatus(true),
  onCleanup: cleanupPaymentManualWatcher
})

const changeDeviceCount = (delta) => {
  const next = (upgradeForm.value.additionalDevices || 1) + delta
  if (next >= 1 && next <= 500) {
    upgradeForm.value.additionalDevices = next
    calculateUpgradeCost()
  }
}

onMounted(() => {
  if (typeof window !== 'undefined') {
    window.addEventListener('resize', handleResize, { passive: true })
  }
})

onUnmounted(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', handleResize)
    if (resizeRafId !== null) {
      window.cancelAnimationFrame(resizeRafId)
      resizeRafId = null
    }
  }
  cleanupPaymentStatusCheck()
})
</script>

<style scoped>
.upgrade-content {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin: -4px -2px 0;
  padding: 0 0 4px;
}

.upgrade-hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  padding: 18px;
  border: 1px solid #dbe4f0;
  border-radius: 8px;
  background: #f8fafc;
}

.hero-eyebrow {
  margin-bottom: 6px;
  font-size: 12px;
  font-weight: 700;
  color: #2563eb;
}

.upgrade-hero h3 {
  margin: 0;
  font-size: 20px;
  line-height: 1.25;
  font-weight: 800;
  color: #111827;
}

.upgrade-hero p {
  margin: 8px 0 0;
  max-width: 280px;
  font-size: 13px;
  line-height: 1.6;
  color: #64748b;
}

.hero-metric {
  width: 88px;
  min-width: 88px;
  padding: 12px 10px;
  border-radius: 8px;
  background: #111827;
  color: #ffffff;
  text-align: center;
}

.hero-metric span {
  display: block;
  font-size: 28px;
  line-height: 1;
  font-weight: 800;
}

.hero-metric small {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  color: #cbd5e1;
}

.upgrade-panel {
  padding: 16px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #ffffff;
}

.panel-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.panel-title span {
  font-size: 15px;
  font-weight: 700;
  color: #111827;
}

.panel-title small {
  font-size: 12px;
  color: #64748b;
  white-space: nowrap;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.summary-item {
  padding: 12px;
  border-radius: 8px;
  background: #f8fafc;
  border: 1px solid #edf2f7;
}

.summary-label {
  display: block;
  margin-bottom: 6px;
  font-size: 12px;
  color: #64748b;
}

.summary-item strong {
  font-size: 18px;
  line-height: 1;
  font-weight: 800;
  color: #111827;
}

.summary-item.highlight strong {
  color: #2563eb;
}

.summary-item.highlight {
  border-color: #bfd4f7;
  background: #f0f5ff;
}

.form-item-block {
  margin-bottom: 16px;
}

.form-item-block:last-child {
  margin-bottom: 0;
}

.form-row-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.form-row-label span {
  font-size: 13px;
  font-weight: 700;
  color: #374151;
}

.form-row-label em {
  font-style: normal;
  font-size: 12px;
  color: #64748b;
  text-align: right;
}

.form-row-label em strong {
  color: #2563eb;
  font-weight: 800;
}

/* ======== 升级主面板 ======== */
.upgrade-main-panel {
  padding: 18px 20px 10px;
}

/* ======== 设备卡片 Stepper ======== */
.device-block {
  margin-bottom: 24px;
}

.device-stepper-card {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 20px;
  padding: 14px 0 10px;
}

.stepper-btn {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  border: 2px solid #dbe4f0;
  border-radius: 8px;
  background: #ffffff;
  color: #374151;
  font-size: 20px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: border-color 0.16s ease, background-color 0.16s ease, color 0.16s ease, transform 0.16s ease;
  user-select: none;
}
.stepper-btn:hover:not(:disabled) {
  border-color: #2563eb;
  background: #eff6ff;
  color: #2563eb;
}
.stepper-btn:active:not(:disabled) {
  transform: translateY(1px);
}
.stepper-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.stepper-btn-plus {
  border-color: #2563eb;
  background: #2563eb;
  color: #ffffff;
}
.stepper-btn-plus:hover:not(:disabled) {
  background: #1d4ed8;
  border-color: #1d4ed8;
  color: #ffffff;
}

.stepper-display {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 100px;
  padding: 8px 24px;
  border-radius: 10px;
  background: #f0f5ff;
  border: 2px dashed #bfd4f7;
}

.stepper-number {
  font-size: 36px;
  line-height: 1.1;
  font-weight: 900;
  color: #1d4ed8;
  letter-spacing: 0;
  font-variant-numeric: tabular-nums;
}

.stepper-unit {
  margin-top: 4px;
  font-size: 13px;
  font-weight: 600;
  color: #64748b;
}

/* 设备圆点进度 (≤30 设备) */
.device-dot-progress {
  margin-top: 14px;
  padding: 10px 12px;
  border-radius: 10px;
  background: #f8fafc;
  border: 1px solid #edf2f7;
}

.dot-labels {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 11px;
  color: #94a3b8;
  font-weight: 500;
}

.dot-track {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  flex-shrink: 0;
  transition: background-color 0.2s ease;
}
.dot-existing {
  background: #94a3b8;
}
.dot-new {
  background: #2563eb;
}

/* 设备条形进度 (>30 设备) */
.device-bar-progress {
  margin-top: 14px;
  padding: 10px 12px;
  border-radius: 10px;
  background: #f8fafc;
  border: 1px solid #edf2f7;
}

.bar-labels {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 11px;
  color: #94a3b8;
  font-weight: 500;
}

.bar-track {
  --existing-device-percent: 100%;
  --new-device-percent: 0%;
  position: relative;
  height: 12px;
  border-radius: 6px;
  background: #e2e8f0;
  overflow: hidden;
}

.bar-fill {
  position: absolute;
  top: 0;
  height: 100%;
  border-radius: 6px;
  transition: width 0.35s ease;
}
.bar-existing {
  left: 0;
  width: var(--existing-device-percent);
  background: #94a3b8;
}
.bar-new {
  left: var(--existing-device-percent);
  width: var(--new-device-percent);
  background: #2563eb;
}

/* ======== 月份卡片选择器 ======== */
.duration-block {
  margin-bottom: 4px;
}

.month-cards-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  margin-top: 2px;
}

.month-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 14px 6px;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  background: #ffffff;
  cursor: pointer;
  transition: border-color 0.16s ease, background-color 0.16s ease;
  user-select: none;
  gap: 4px;
  font-family: inherit;
}
.month-card:hover {
  border-color: #93b4f0;
  background: #f8faff;
}
.month-card.is-active {
  border-color: #2563eb;
  background: #eff6ff;
}
.month-card.is-zero {
  border-style: dashed;
}
.month-card.is-zero.is-active {
  border-style: solid;
  border-color: #94a3b8;
  background: #f1f5f9;
  color: #475569;
}

.month-card-num {
  font-size: 17px;
  font-weight: 800;
  color: #111827;
  line-height: 1;
}
.month-card.is-active .month-card-num {
  color: #1d4ed8;
}
.month-card.is-zero.is-active .month-card-num {
  color: #475569;
}

.month-card-days {
  font-size: 11px;
  color: #94a3b8;
  font-weight: 500;
}
.month-card.is-active .month-card-days {
  color: #6093e8;
}
.month-card.is-zero.is-active .month-card-days {
  color: #94a3b8;
}

/* 到期日预览 */
.expire-preview {
  margin-top: 14px;
  padding: 14px 16px;
  border-radius: 10px;
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
}

.expire-preview-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #475569;
}

.expire-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-weight: 600;
  color: #374151;
  flex-shrink: 0;
}

.expire-old {
  color: #94a3b8;
  text-decoration: line-through;
  font-weight: 500;
}

.expire-arrow {
  color: #2563eb;
  flex-shrink: 0;
}

.expire-new {
  color: #059669;
  font-weight: 700;
}

.expire-preview-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
  padding: 6px 12px;
  border-radius: 8px;
  background: #dcfce7;
  color: #16a34a;
  font-size: 13px;
  font-weight: 700;
}

.cost-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.cost-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  font-size: 14px;
  color: #475569;
}

.cost-row strong {
  font-weight: 700;
  color: #111827;
}

.discount-row strong {
  color: #059669;
}

.total-row {
  margin-top: 2px;
  padding-top: 12px;
  border-top: 1px dashed #cbd5e1;
}

.total-row span,
.total-row strong {
  font-size: 18px;
  color: #2563eb;
}

.balance-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  background: #f8fafc;
  border: 1px solid #edf2f7;
  color: #475569;
  font-size: 13px;
}

.balance-info strong {
  margin-left: auto;
  color: #111827;
  font-size: 14px;
}

.payment-loading-text {
  padding: 10px 0;
  color: #909399;
  font-size: 13px;
}

.payment-radio-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.payment-radio-card {
  width: 100%;
  height: auto;
  margin: 0;
  padding: 11px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #ffffff;
}

.payment-radio-card.is-checked {
  border-color: #2563eb;
  background: #eff6ff;
}

.payment-radio-card :deep(.el-radio__label) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  padding-left: 8px;
  color: #111827;
}

.pay-title {
  font-size: 14px;
  font-weight: 700;
}

.pay-status {
  font-size: 12px;
  color: #64748b;
  white-space: nowrap;
}

.pay-status.success {
  color: #059669;
}

.pay-status.danger {
  color: #dc2626;
}

.drawer-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
}

.footer-amount {
  margin-right: auto;
}

.footer-amount span {
  display: block;
  margin-bottom: 2px;
  font-size: 12px;
  color: #64748b;
}

.footer-amount strong {
  font-size: 20px;
  line-height: 1;
  font-weight: 800;
  color: #2563eb;
}

.drawer-footer .el-button {
  min-width: 104px;
}

.payment-qr-container {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.payment-summary-card {
  padding: 18px 20px;
  border-radius: 10px;
  background: #f8fafc;
  border: 1px solid #dbe4f0;
}

.summary-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.summary-label {
  font-size: 13px;
  color: #64748b;
  margin-bottom: 6px;
}

.summary-amount {
  font-size: 32px;
  line-height: 1;
  font-weight: 800;
  color: #111827;
}

.summary-badge {
  flex-shrink: 0;
  padding: 8px 12px;
  border-radius: 999px;
  background: rgba(37, 99, 235, 0.1);
  color: #1d4ed8;
  font-size: 13px;
  font-weight: 600;
}

.summary-meta {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.meta-item {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  font-size: 14px;
}

.meta-key {
  color: #64748b;
  flex-shrink: 0;
}

.meta-value {
  color: #0f172a;
  font-weight: 500;
  text-align: right;
  word-break: break-all;
}

.qr-panel {
  padding: 20px;
  border-radius: 10px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
}

.qr-panel-header {
  text-align: center;
  margin-bottom: 18px;
}

.qr-panel-header h4 {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
}

.qr-panel-header p {
  margin: 8px 0 0;
  font-size: 13px;
  color: #64748b;
}

.qr-code-wrapper {
  display: flex;
  justify-content: center;
}

.qr-code-wrapper.iframe-mode {
  display: block;
}

.payment-page-iframe {
  width: 100%;
  min-height: 560px;
  border: 1px solid #dbeafe;
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
}

.payment-page-iframe iframe {
  width: 100%;
  min-height: 560px;
  border: none;
}

.qr-code,
.qr-loading {
  width: 280px;
  min-height: 280px;
  border-radius: 8px;
  background: #ffffff;
  border: 1px solid #dbeafe;
  display: flex;
  align-items: center;
  justify-content: center;
}

.qr-code img {
  width: 232px;
  height: 232px;
  display: block;
  border-radius: 8px;
  background: #fff;
}

.qr-loading {
  flex-direction: column;
  gap: 12px;
  color: #64748b;
}

.payment-tips {
  margin-top: 16px;
}

.tip-text {
  margin: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #334155;
  font-size: 14px;
  font-weight: 500;
}

.tip-text :deep(svg) {
  color: #2563eb;
}

.payment-actions-container {
  margin-top: 18px;
}

.alipay-btn {
  height: 46px;
  border-radius: 8px;
  font-weight: 600;
}

.btn-icon {
  margin-right: 6px;
}

@media (max-width: 768px) {
  .upgrade-hero {
    padding: 16px;
  }

  .upgrade-hero h3 {
    font-size: 18px;
  }

  .upgrade-hero p {
    max-width: none;
    font-size: 12px;
  }

  .hero-metric {
    width: 76px;
    min-width: 76px;
  }

  .hero-metric span {
    font-size: 24px;
  }

  .upgrade-panel {
    padding: 14px;
  }

  .upgrade-main-panel {
    padding: 14px 14px 6px;
  }

  .summary-grid {
    grid-template-columns: 1fr;
  }

  /* 设备 stepper — 移动端适配 */
  .device-stepper-card {
    gap: 14px;
    padding: 10px 0 6px;
  }
  .stepper-btn {
    width: 42px;
    height: 42px;
    border-radius: 8px;
  }
  .stepper-display {
    min-width: 84px;
    padding: 8px 18px;
  }
  .stepper-number {
    font-size: 30px;
  }

  /* 月份卡片 — 移动端 2 列 */
  .month-cards-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
  }
  .month-card {
    padding: 12px 6px;
  }
  .month-card-num {
    font-size: 15px;
  }

  /* 到期预览 */
  .expire-preview-row {
    flex-wrap: wrap;
    gap: 4px;
    font-size: 12px;
  }

  .payment-radio-card :deep(.el-radio__label) {
    align-items: flex-start;
    flex-direction: column;
    gap: 4px;
  }

  .drawer-footer {
    display: grid;
    grid-template-columns: 1fr 1fr;
  }

  .footer-amount {
    grid-column: 1 / -1;
  }

  .drawer-footer .el-button {
    width: 100%;
    min-width: 0;
    margin: 0;
  }

  .payment-summary-card,
  .qr-panel {
    padding: 16px;
    border-radius: 8px;
  }

  .summary-header,
  .meta-item {
    flex-direction: column;
    align-items: flex-start;
  }

  .meta-value {
    text-align: left;
  }

  .summary-amount {
    font-size: 28px;
  }

  .qr-code,
  .qr-loading {
    width: 100%;
    min-height: 252px;
  }

  .qr-code img {
    width: min(220px, 100% - 32px);
    height: min(220px, 100% - 32px);
  }
}

.upgrade-hero,
.upgrade-panel,
.payment-summary-card,
.qr-panel {
  border: 1px solid #dcdfe6 !important;
  border-radius: 8px !important;
  background: #ffffff !important;
  box-shadow: none !important;
}

.hero-metric,
.summary-item,
.month-card,
.device-stepper-card,
.payment-radio-card,
.stepper-display,
.qr-code,
.qr-loading {
  border-radius: 8px !important;
  background: #fff !important;
  box-shadow: none !important;
}

.hero-metric {
  border: 1px solid #d9ecff !important;
  background: #ecf5ff !important;
  color: #409eff !important;
}

.hero-metric small {
  color: #606266 !important;
}

.stepper-btn {
  border-radius: 8px !important;
  box-shadow: none !important;
}

.stepper-number,
.summary-amount,
.footer-price,
.month-card-price {
  color: #409eff !important;
  font-size: 22px !important;
  line-height: 1.2 !important;
}

.qr-code-wrapper,
.recharge-qr-section .qr-code-wrapper {
  border: 1px dashed #dcdfe6 !important;
  border-radius: 8px !important;
  background: #f8fafc !important;
}
</style>
