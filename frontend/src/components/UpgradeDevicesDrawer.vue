<template>
  <el-drawer
    v-model="drawerVisible"
    title="升级设备数量"
    direction="rtl"
    size="420px"
    :close-on-click-modal="false"
    class="upgrade-drawer"
    @open="handleUpgradeDialogOpen"
  >
    <div class="upgrade-content" v-if="subscription">
      <div class="current-subscription-info">
        <h4>当前订阅信息</h4>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="当前设备数">{{ subscription.device_limit || subscription.maxDevices || 0 }} 个</el-descriptions-item>
          <el-descriptions-item label="剩余天数">{{ getRemainingDays(subscription) }} 天</el-descriptions-item>
        </el-descriptions>
      </div>
      <div class="upgrade-options">
        <h4>升级选项</h4>
        <div class="form-item-block">
          <div class="form-label">增加设备数量</div>
          <div class="device-input-row">
            <el-button @click="changeDeviceCount(-1)" :disabled="upgradeForm.additionalDevices <= 1" circle size="small"><el-icon><Minus /></el-icon></el-button>
            <el-input-number
              v-model="upgradeForm.additionalDevices"
              :min="1"
              :max="500"
              :controls="false"
              style="width: 90px; text-align: center;"
              @change="calculateUpgradeCost"
            />
            <el-button @click="changeDeviceCount(1)" circle size="small"><el-icon><Plus /></el-icon></el-button>
            <span class="device-input-hint">个设备</span>
          </div>
          <div class="form-hint">升级后共 {{ (subscription.device_limit || subscription.maxDevices || 0) + (upgradeForm.additionalDevices || 0) }} 个设备</div>
        </div>
        <div class="form-item-block">
          <div class="form-label">延长到期时间（可选）</div>
          <el-select
            v-model="upgradeForm.additionalDays"
            @change="calculateUpgradeCost"
            style="width: 100%"
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
          <div class="form-hint" v-if="upgradeForm.additionalDays > 0">将延长 {{ upgradeForm.additionalDays }} 天</div>
        </div>
      </div>
      <div class="cost-calculation" v-if="upgradeCost > 0">
        <h4>费用明细</h4>
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="升级费用">¥{{ upgradeCost.toFixed(2) }}</el-descriptions-item>
          <el-descriptions-item label="等级折扣" v-if="levelDiscount > 0">
            -¥{{ levelDiscount.toFixed(2) }}
          </el-descriptions-item>
          <el-descriptions-item label="应付金额">
            <span class="final-amount">¥{{ finalAmount.toFixed(2) }}</span>
          </el-descriptions-item>
        </el-descriptions>
      </div>
      <div class="payment-method" v-if="finalAmount > 0 || upgradeForm.additionalDevices >= 1">
        <h4>支付方式</h4>
        <div class="balance-info">
          <span>账户余额：¥{{ userBalance.toFixed(2) }}</span>
        </div>
        <div v-if="!availableUpgradePaymentMethods || availableUpgradePaymentMethods.length === 0" style="color: #909399; padding: 10px;">
          正在加载支付方式...
        </div>
        <el-radio-group v-model="paymentMethod" @change="handlePaymentMethodChange" v-else>
          <el-radio label="balance" :disabled="userBalance <= 0 || (finalAmount > 0 && userBalance < finalAmount)">
            余额支付
            <span v-if="finalAmount > 0 && userBalance >= finalAmount" style="color: #67c23a; margin-left: 5px">（余额充足）</span>
            <span v-else-if="finalAmount > 0 && userBalance > 0" style="color: #f56c6c; margin-left: 5px">（余额不足，还需 ¥{{ (finalAmount - userBalance).toFixed(2) }}）</span>
          </el-radio>
          <template v-for="method in availableUpgradePaymentMethods" :key="method.key">
            <el-radio
              v-if="method && method.key && method.key !== 'balance'"
              :label="method.key"
            >
              {{ method.name || method.key }}
            </el-radio>
          </template>
        </el-radio-group>
      </div>
    </div>
    <template #footer>
      <div class="drawer-footer">
        <el-button @click="drawerVisible = false" style="width:48%">取消</el-button>
        <el-button
          type="primary"
          style="width:48%"
          @click="confirmUpgrade"
          :loading="upgradeLoading"
          :disabled="!upgradeForm.additionalDevices || upgradeForm.additionalDevices < 1"
        >
          确认升级并支付
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
          <h4>请使用支付宝扫码</h4>
          <p>支付完成后会自动刷新升级结果</p>
        </div>
        <div class="qr-code-wrapper">
          <div v-if="paymentQRCode" class="qr-code">
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
        <div class="payment-tips">
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
import { ElMessage } from 'element-plus'
import { Loading, Wallet, InfoFilled, Plus, Minus } from '@element-plus/icons-vue'
import { orderAPI, parsePaymentMethods, useApi, userAPI, userLevelAPI } from '@/utils/api'
import { getRemainingDays as getRemainingDaysUtil } from '@/utils/date'

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
const isMobile = ref(typeof window !== 'undefined' ? window.innerWidth <= 768 : false)

const handleResize = () => {
  isMobile.value = window.innerWidth <= 768
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
      finalAmount.value = parseFloat(response.data.data.amount || 0)
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
  if (paymentStatusCheckTimer.value) clearInterval(paymentStatusCheckTimer.value)
  paymentStatusCheckTimer.value = setInterval(async () => {
    if (!upgradeOrder.value?.order_no) {
      clearInterval(paymentStatusCheckTimer.value)
      return
    }
    await checkUpgradeOrderStatus(true)
  }, 2000)
}

const checkUpgradeOrderStatus = async (isAutoCheck = false) => {
  if (!upgradeOrder.value?.order_no) return
  try {
    const response = await orderAPI.getOrderStatus(upgradeOrder.value.order_no)
    if (response?.data?.success && response.data.data?.status === 'paid') {
      if (paymentStatusCheckTimer.value) clearInterval(paymentStatusCheckTimer.value)
      paymentQRVisible.value = false
      ElMessage.success('支付成功，设备已升级！')
      await props.onSuccess?.()
      window.dispatchEvent(new CustomEvent('subscription-updated'))
      window.dispatchEvent(new CustomEvent('user-info-updated'))
      setTimeout(async () => {
        await props.onSuccess?.()
        window.dispatchEvent(new CustomEvent('subscription-updated'))
        window.dispatchEvent(new CustomEvent('user-info-updated'))
      }, 500)
      upgradeForm.value = { additionalDevices: 5, additionalDays: 0 }
      upgradeCost.value = 0
      finalAmount.value = 0
      upgradeOrder.value = null
      paymentQRCode.value = null
    } else if (!isAutoCheck) {
      ElMessage.warning('订单尚未支付，请完成支付')
    }
  } catch (error) {
    if (!isAutoCheck) ElMessage.error('检查订单状态失败: ' + (error.response?.data?.message || error.message))
  }
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
        drawerVisible.value = false
        await props.onSuccess?.()
      } else {
        const paymentUrlVal = data.payment_url || data.payment_qr_code
        if (!paymentUrlVal) {
          ElMessage.error('支付链接生成失败，请稍后重试')
          return
        }
        const paymentMethodName = data.payment_method || paymentMethod.value
        const isYipay = paymentMethodName && (
          paymentMethodName.includes('yipay') ||
          paymentMethodName.includes('易支付')
        )
        if (isYipay) {
          ElMessage.info('正在跳转到支付页面...')
          window.location.href = paymentUrlVal
        } else {
          upgradeOrder.value = {
            ...data,
            additional_devices: upgradeForm.value.additionalDevices,
            additional_days: upgradeForm.value.additionalDays || 0
          }
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
  const checkStatus = async () => {
    if (document.visibilityState === 'visible' && paymentQRVisible.value && upgradeOrder.value?.order_no) {
      await checkUpgradeOrderStatus()
      document.removeEventListener('visibilitychange', checkStatus)
    }
  }
  document.addEventListener('visibilitychange', checkStatus)
  window.location.href = alipayAppUrl
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
    window.addEventListener('resize', handleResize)
  }
})

onUnmounted(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', handleResize)
  }
  if (paymentStatusCheckTimer.value) {
    clearInterval(paymentStatusCheckTimer.value)
    paymentStatusCheckTimer.value = null
  }
})
</script>

<style scoped>
.upgrade-drawer {
  :deep(.el-drawer__body) {
    padding: 20px;
  }
}

.drawer-footer {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.final-amount {
  color: #2563eb;
  font-weight: 700;
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
