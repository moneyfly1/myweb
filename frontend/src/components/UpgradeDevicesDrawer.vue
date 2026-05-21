<template>
  <el-drawer
    v-model="drawerVisible"
    title="升级设备数量"
    direction="rtl"
    :size="isMobile ? '92%' : '460px'"
    :close-on-click-modal="false"
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
          <div class="summary-item">
            <span class="summary-label">升级后</span>
            <strong>{{ targetDeviceLimit }}</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">延长时间</span>
            <strong>{{ upgradeForm.additionalDays || 0 }} 天</strong>
          </div>
        </div>
      </section>

      <section class="upgrade-panel">
        <div class="panel-title">
          <span>升级内容</span>
          <small>至少增加 1 个设备</small>
        </div>
        <div class="form-item-block">
          <div class="form-row-label">
            <span>增加设备数量</span>
            <em>升级后共 {{ targetDeviceLimit }} 个设备</em>
          </div>
          <div class="device-stepper">
            <el-button @click="changeDeviceCount(-1)" :disabled="upgradeForm.additionalDevices <= 1" circle size="small" aria-label="减少设备">
              <el-icon><Minus /></el-icon>
            </el-button>
            <el-input-number
              v-model="upgradeForm.additionalDevices"
              :min="1"
              :max="500"
              :controls="false"
              class="device-number"
              @change="calculateUpgradeCost"
            />
            <el-button @click="changeDeviceCount(1)" circle size="small" aria-label="增加设备">
              <el-icon><Plus /></el-icon>
            </el-button>
            <span class="device-unit">个设备</span>
          </div>
        </div>
        <div class="form-item-block">
          <div class="form-row-label">
            <span>延长到期时间</span>
            <em>{{ upgradeForm.additionalDays > 0 ? `约 ${additionalMonths} 个月` : '可选' }}</em>
          </div>
          <el-select
            v-model="upgradeForm.additionalDays"
            @change="calculateUpgradeCost"
            class="duration-select"
            placeholder="请选择延长的月数"
          >
            <el-option label="不延长" :value="0" />
            <el-option
              v-for="months in monthOptions"
              :key="months"
              :label="`${months} 个月（${months * 30} 天）`"
              :value="months * 30"
            />
          </el-select>
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
  </el-drawer>
  <el-dialog
    v-model="paymentQRVisible"
    title="扫码支付"
    :width="isMobile ? '92%' : '520px'"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    class="payment-qr-dialog"
    :center="true"
    append-to-body
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
            style="width: 100%;"
          >
            <el-icon class="btn-icon"><Wallet /></el-icon>
            跳转支付宝App支付
          </el-button>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { Loading, Wallet, InfoFilled, Plus, Minus } from '@element-plus/icons-vue'
import { orderAPI, parsePaymentMethods, useApi, userAPI, userLevelAPI, cachedAPI, pendingPaymentStorage } from '@/utils/api'
import { getRemainingDays as getRemainingDaysUtil } from '@/utils/date'
import { safeNavigate } from '@/utils/safeOpen'

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
const paymentStatusCheckTimer = ref(null)
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
  const url = order.payment_url || order.payment_qr_code
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
    const QRCodeLib = (await import('qrcode')).default
    paymentQRCode.value = await QRCodeLib.toDataURL(url, qrOptions)
    paymentQRVisible.value = true
    startPaymentStatusCheck()
  } catch (error) {
    ElMessage.error('生成二维码失败: ' + (error.response?.data?.message || error.message))
  }
}

const startPaymentStatusCheck = () => {
  cleanupPaymentStatusCheck()
  checkUpgradeOrderStatus(true)
  paymentStatusCheckTimer.value = setInterval(async () => {
    if (!upgradeOrder.value?.order_no) {
      cleanupPaymentStatusCheck()
      return
    }
    await checkUpgradeOrderStatus(true)
  }, 2000)
}

const cleanupPaymentStatusCheck = () => {
  if (paymentStatusCheckTimer.value) {
    clearInterval(paymentStatusCheckTimer.value)
    paymentStatusCheckTimer.value = null
  }
  cleanupPaymentManualWatcher()
}

const cleanupPaymentManualWatcher = () => {
  if (paymentManualVisibilityHandler) {
    document.removeEventListener('visibilitychange', paymentManualVisibilityHandler)
    paymentManualVisibilityHandler = null
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
.upgrade-drawer {
  :deep(.el-drawer__header) {
    margin-bottom: 0;
    padding: 18px 22px;
    border-bottom: 1px solid #e5e7eb;
  }

  :deep(.el-drawer__title) {
    font-size: 17px;
    font-weight: 700;
    color: #111827;
  }

  :deep(.el-drawer__body) {
    padding: 18px 20px 22px;
    background: #f7f9fc;
    overflow-y: auto;
  }

  :deep(.el-drawer__footer) {
    padding: 14px 18px;
    border-top: 1px solid #e5e7eb;
    background: #ffffff;
  }
}

.upgrade-content {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.upgrade-hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  padding: 18px;
  border: 1px solid #dbe4f0;
  border-radius: 8px;
  background: linear-gradient(135deg, #ffffff 0%, #eef6ff 100%);
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

.device-stepper {
  display: grid;
  grid-template-columns: 32px 104px 32px 1fr;
  align-items: center;
  gap: 8px;
}

.device-stepper .el-button {
  width: 32px;
  height: 32px;
  margin: 0;
}

.device-number {
  width: 104px;
}

.device-number :deep(.el-input__wrapper) {
  box-shadow: 0 0 0 1px #dbe4f0 inset;
}

.device-number :deep(.el-input__inner) {
  text-align: center;
  font-weight: 700;
  color: #111827;
}

.device-unit {
  font-size: 13px;
  color: #64748b;
}

.duration-select {
  width: 100%;
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

.payment-qr-dialog {
  :deep(.el-dialog) {
    border-radius: 24px;
    overflow: hidden;
    background: linear-gradient(180deg, #f8fbff 0%, #ffffff 100%);
    box-shadow: 0 24px 80px rgba(15, 23, 42, 0.22);
  }

  :deep(.el-dialog__header) {
    margin-right: 0;
    padding: 22px 24px 12px;
    border-bottom: 1px solid rgba(37, 99, 235, 0.08);
  }

  :deep(.el-dialog__title) {
    font-size: 24px;
    font-weight: 700;
    color: #0f172a;
    letter-spacing: 0.02em;
  }

  :deep(.el-dialog__headerbtn) {
    top: 22px;
    right: 20px;
  }

  :deep(.el-dialog__body) {
    padding: 0 24px 24px;
  }
}

.payment-qr-container {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.payment-summary-card {
  padding: 18px 20px;
  border-radius: 20px;
  background: linear-gradient(135deg, #eff6ff 0%, #f8fafc 100%);
  border: 1px solid rgba(59, 130, 246, 0.14);
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
  border-radius: 24px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.8);
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
  border-radius: 12px;
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
  border-radius: 24px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
  border: 1px solid #dbeafe;
  box-shadow: 0 16px 40px rgba(37, 99, 235, 0.12);
  display: flex;
  align-items: center;
  justify-content: center;
}

.qr-code img {
  width: 232px;
  height: 232px;
  display: block;
  border-radius: 16px;
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
  border-radius: 14px;
  font-weight: 600;
}

.btn-icon {
  margin-right: 6px;
}

@media (max-width: 768px) {
  .upgrade-drawer {
    :deep(.el-drawer__header) {
      padding: 14px 16px;
    }

    :deep(.el-drawer__body) {
      padding: 14px;
    }

    :deep(.el-drawer__footer) {
      padding: 12px 14px 14px;
    }
  }

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

  .summary-grid {
    grid-template-columns: 1fr;
  }

  .device-stepper {
    grid-template-columns: 32px 1fr 32px;
  }

  .device-number {
    width: 100%;
  }

  .device-unit {
    grid-column: 1 / -1;
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

  .payment-qr-dialog {
    :deep(.el-dialog) {
      border-radius: 20px;
    }

    :deep(.el-dialog__header) {
      padding: 18px 18px 10px;
    }

    :deep(.el-dialog__title) {
      font-size: 20px;
    }

    :deep(.el-dialog__body) {
      padding: 0 18px 18px;
    }
  }

  .payment-summary-card,
  .qr-panel {
    padding: 16px;
    border-radius: 18px;
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
</style>
