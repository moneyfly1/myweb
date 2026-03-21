<template>
  <div class="payment-form">
    <el-card class="payment-card">
      <template #header>
        <div class="card-header">
          <span>选择支付方式</span>
        </div>
      </template>
      <div class="order-info">
        <h3>订单信息</h3>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="订单号">{{ orderInfo.orderNo }}</el-descriptions-item>
          <el-descriptions-item label="套餐名称">{{ orderInfo.packageName }}</el-descriptions-item>
          <el-descriptions-item label="支付金额">
            <span class="amount">¥{{ orderInfo.amount }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="有效期">{{ orderInfo.duration }}天</el-descriptions-item>
        </el-descriptions>
      </div>
      <div class="payment-methods">
        <h3>支付方式</h3>
        <el-radio-group v-model="selectedPaymentMethod" @change="onPaymentMethodChange">
          <el-radio-button
            v-for="method in availablePaymentMethods"
            :key="method.key"
            :value="method.key"
          >
            <i :class="`payment-icon ${method.icon}-icon`"></i>
            {{ method.name }}
          </el-radio-button>
        </el-radio-group>
        <div class="payment-description" v-if="selectedPaymentMethod">
          <p>{{ getPaymentDescription() }}</p>
        </div>
      </div>
      <div class="payment-actions">
        <el-button
          type="primary"
          size="large"
          :loading="isProcessing"
          @click="handlePayment"
          :disabled="!selectedPaymentMethod"
        >
          {{ getPaymentButtonText() }}
        </el-button>
        <el-button
          size="large"
          @click="$emit('cancel')"
          :disabled="isProcessing"
        >
          取消支付
        </el-button>
      </div>
      <div class="payment-tips desktop-only">
        <el-alert
          v-if="selectedPaymentMethod === 'alipay'"
          type="info"
          :closable="false"
          show-icon
        >
          点击支付按钮后将显示二维码，使用支付宝扫描完成支付
        </el-alert>
        <el-alert
          v-if="selectedPaymentMethod === 'wechat'"
          type="info"
          :closable="false"
          show-icon
        >
          点击支付按钮后将显示二维码，使用微信扫描完成支付
        </el-alert>
      </div>
    </el-card>
    <el-dialog
      v-model="wechatQRVisible"
      title="扫码支付"
      :width="isMobileDevice ? '92%' : '450px'"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      class="payment-qr-dialog"
      @close="() => { if (statusCheckTimer) { clearInterval(statusCheckTimer); statusCheckTimer = null; } }"
    >
      <div class="payment-qr-container">
        <div class="order-info-compact">
          <div class="info-row">
            <span class="label">订单号</span>
            <span class="value">{{ orderInfo.orderNo }}</span>
          </div>
          <div class="info-row">
            <span class="label">套餐名称</span>
            <span class="value">{{ orderInfo.packageName }}</span>
          </div>
          <div class="info-row">
            <span class="label">支付金额</span>
            <span class="value amount">¥{{ orderInfo.amount }}</span>
          </div>
          <div class="info-row">
            <span class="label">支付方式</span>
            <span class="value">{{ selectedPaymentMethod === 'alipay' ? '支付宝' : selectedPaymentMethod === 'wechat' ? '微信支付' : '其他' }}</span>
          </div>
        </div>
        <div class="qr-code-wrapper-compact">
          <div v-if="wechatQRCode" class="qr-code">
            <img :src="wechatQRCode" alt="支付二维码" />
          </div>
          <div v-else class="qr-loading">
            <el-icon class="is-loading"><Loading /></el-icon>
            <p>生成中...</p>
          </div>
        </div>
        <div class="payment-actions-compact" v-if="isMobileDevice && selectedPaymentMethod === 'alipay'">
          <el-button
            type="success"
            size="default"
            @click="openAlipayApp"
            style="width: 100%;"
          >
            <el-icon style="margin-right: 5px;"><Wallet /></el-icon>
            打开支付宝App
          </el-button>
        </div>
      </div>
    </el-dialog>
    <el-dialog
      v-model="resultVisible"
      :title="paymentResult.success ? '支付成功' : '支付失败'"
      width="500px"
      :close-on-click-modal="false"
    >
      <div class="payment-result">
        <div v-if="paymentResult.success" class="success-result">
          <el-icon class="result-icon success"><CircleCheckFilled /></el-icon>
          <h3>支付成功！</h3>
          <p>您的订阅已激活，可以正常使用了。</p>
          <div class="result-details">
            <p><strong>订单号：</strong>{{ paymentResult.data?.order_no }}</p>
            <p><strong>支付金额：</strong>¥{{ paymentResult.data?.amount }}</p>
            <p><strong>支付时间：</strong>{{ formatDateTime(paymentResult.data?.paid_at) }}</p>
          </div>
        </div>
        <div v-else class="failed-result">
          <el-icon class="result-icon failed"><CircleCloseFilled /></el-icon>
          <h3>支付失败</h3>
          <p>{{ paymentResult.message }}</p>
          <div class="error-details">
            <p><strong>错误代码：</strong>{{ paymentResult.error_code }}</p>
            <p><strong>错误信息：</strong>{{ paymentResult.error_message }}</p>
          </div>
        </div>
      </div>
      <template #footer>
        <span class="dialog-footer">
          <el-button v-if="!paymentResult.success" @click="retryPayment">
            重试支付
          </el-button>
          <el-button type="primary" @click="handleResultClose">
            {{ paymentResult.success ? '完成' : '关闭' }}
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>
<script>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { CircleCheckFilled, CircleCloseFilled, Loading, Wallet } from '@element-plus/icons-vue'
import { useApi } from '@/utils/api'
import { formatDateTime } from '@/utils/date'
export default {
  name: 'PaymentForm',
  components: {
    CircleCheckFilled,
    CircleCloseFilled,
    Loading,
    Wallet
  },
  props: {
    orderInfo: {
      type: Object,
      required: true
    }
  },
  emits: ['success', 'cancel', 'error'],
  setup(props, { emit }) {
    const api = useApi()
    const selectedPaymentMethod = ref('')
    const isProcessing = ref(false)
    const isCheckingStatus = ref(false)
    const wechatQRVisible = ref(false)
    const wechatQRCode = ref('')
    const resultVisible = ref(false)
    const availablePaymentMethods = ref([])
    const isMobileDevice = ref(false)
    const paymentResult = reactive({
      success: false,
      message: '',
      data: null,
      error_code: '',
      error_message: ''
    })
    let statusCheckTimer = null

    // 检测是否为移动设备
    const detectMobileDevice = () => {
      const ua = navigator.userAgent.toLowerCase()
      const isMobile = /android|webos|iphone|ipad|ipod|blackberry|iemobile|opera mini/i.test(ua)
      const isSmallScreen = window.innerWidth <= 768
      isMobileDevice.value = isMobile || isSmallScreen
    }

    // 打开支付宝应用
    const openAlipayApp = () => {
      if (wechatQRCode.value) {
        // 尝试直接打开支付宝应用
        window.location.href = wechatQRCode.value
      }
    }
    const onPaymentMethodChange = (method) => {
      selectedPaymentMethod.value = method
    }
    const getPaymentButtonText = () => {
      return isProcessing.value ? '处理中...' : `立即支付 ¥${props.orderInfo.amount}`
    }
    const handlePayment = async () => {
      if (!selectedPaymentMethod.value) {
        ElMessage.warning('请选择支付方式')
        return
      }
      try {
        isProcessing.value = true
        const paymentData = {
          order_no: props.orderInfo.orderNo,
          amount: props.orderInfo.amount,
          currency: 'CNY',
          payment_method: selectedPaymentMethod.value,
          subject: `订阅套餐 - ${props.orderInfo.packageName}`,
          body: `购买${props.orderInfo.duration}天订阅套餐`
        }
        const response = await api.post('/payment/', paymentData)
        if (response.data && response.data.payment_url) {
          wechatQRCode.value = response.data.payment_url
          wechatQRVisible.value = true
          startStatusCheck()
        } else {
          throw new Error('获取支付链接失败')
        }
      } catch (error) {
        ElMessage.error(error.response?.data?.detail || '支付失败，请重试')
        emit('error', error)
      } finally {
        isProcessing.value = false
      }
    }
    const startStatusCheck = () => {
      statusCheckTimer = setInterval(async () => {
        await checkPaymentStatus()
      }, 3000)
    }
    const checkPaymentStatus = async () => {
      try {
        isCheckingStatus.value = true
        const response = await api.get(`/payment/transactions?order_no=${props.orderInfo.orderNo}`)
        const payments = response.data
        if (payments.length > 0) {
          const latestPayment = payments[0]
          if (latestPayment.status === 'success') {
            clearInterval(statusCheckTimer)
            wechatQRVisible.value = false
            paymentResult.success = true
            paymentResult.message = '支付成功'
            paymentResult.data = latestPayment
            resultVisible.value = true
            emit('success', latestPayment)
          } else if (latestPayment.status === 'failed') {
            clearInterval(statusCheckTimer)
            wechatQRVisible.value = false
            paymentResult.success = false
            paymentResult.message = '支付失败'
            paymentResult.error_code = 'PAYMENT_FAILED'
            paymentResult.error_message = '支付处理失败，请重试'
            resultVisible.value = true
            emit('error', new Error('支付失败'))
          }
        }
      } catch (error) {
        console.error('支付状态查询失败', error)
      } finally {
        isCheckingStatus.value = false
      }
    }
    const retryPayment = () => {
      resultVisible.value = false
      handlePayment()
    }
    const handleResultClose = () => {
      resultVisible.value = false
      if (paymentResult.success) {
        emit('success', paymentResult.data)
      }
    }
    const loadPaymentMethods = async () => {
      try {
        const response = await api.get('/payment-methods/active')
        availablePaymentMethods.value = response.data || []
        if (availablePaymentMethods.value.length > 0) {
          selectedPaymentMethod.value = availablePaymentMethods.value[0].key
        }
      } catch (error) {
        ElMessage.error('加载支付方式失败')
      }
    }
    const getPaymentDescription = () => {
      const method = availablePaymentMethods.value.find(m => m.key === selectedPaymentMethod.value)
      return method ? method.description : ''
    }
    onMounted(() => {
      loadPaymentMethods()
      detectMobileDevice()
      // 监听窗口大小变化
      window.addEventListener('resize', detectMobileDevice)
    })
    onUnmounted(() => {
      if (statusCheckTimer) {
        clearInterval(statusCheckTimer)
      }
      window.removeEventListener('resize', detectMobileDevice)
    })
    return {
      selectedPaymentMethod,
      isProcessing,
      isCheckingStatus,
      wechatQRVisible,
      wechatQRCode,
      resultVisible,
      paymentResult,
      availablePaymentMethods,
      isMobileDevice,
      onPaymentMethodChange,
      getPaymentButtonText,
      handlePayment,
      checkPaymentStatus,
      retryPayment,
      handleResultClose,
      getPaymentDescription,
      formatDateTime,
      openAlipayApp
    }
  }
}
</script>
<style scoped>
.payment-form {
  max-width: 800px;
  margin: 0 auto;
}
.payment-card {
  margin-bottom: 20px;
}
.card-header {
  font-size: 18px;
  font-weight: bold;
}
.order-info {
  margin-bottom: 30px;
}
.order-info h3 {
  margin-bottom: 15px;
  color: #303133;
  font-size: 16px;
}
.amount {
  color: #f56c6c;
  font-size: 18px;
  font-weight: bold;
}
.payment-methods {
  margin-bottom: 30px;
}
.payment-methods h3 {
  margin-bottom: 15px;
  color: #303133;
  font-size: 16px;
}
.payment-description {
  margin-top: 10px;
  padding: 10px;
  background-color: #f5f7fa;
  border-radius: 4px;
  color: #606266;
  font-size: 14px;
}
.payment-icon {
  display: inline-block;
  width: 20px;
  height: 20px;
  margin-right: 8px;
  vertical-align: middle;
}
.payment-actions {
  text-align: center;
  margin-bottom: 20px;
}
.payment-actions .el-button {
  margin: 0 10px;
  min-width: 120px;
}
.payment-tips {
  margin-top: 15px;
}
.payment-tips .el-alert {
  margin-bottom: 0;
}
.payment-tips p {
  margin: 5px 0;
  font-size: 14px;
}

.desktop-only {
  display: block;
}

@media (max-width: 768px) {
  .desktop-only {
    display: none !important;
  }
}

/* 紧凑型支付弹窗样式 - 参考订单记录的设计 */
.payment-qr-dialog {
  :deep(.el-dialog) {
    border-radius: 12px;
  }
  :deep(.el-dialog__header) {
    padding: 16px 20px;
    border-bottom: 1px solid #f0f0f0;
  }
  :deep(.el-dialog__body) {
    padding: 16px 20px;
  }
}

.payment-qr-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.order-info-compact {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 12px;
  .info-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 0;
    border-bottom: 1px solid #e9ecef;
    &:last-child {
      border-bottom: none;
    }
    .label {
      color: #666;
      font-size: 14px;
      font-weight: 500;
    }
    .value {
      color: #333;
      font-size: 14px;
      font-weight: 600;
      text-align: right;
      word-break: break-all;
      &.amount {
        color: #f56c6c;
        font-size: 18px;
      }
    }
  }
}

.qr-code-wrapper-compact {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 16px 0;
  .qr-code {
    display: inline-block;
    padding: 12px;
    background: #fff;
    border: 1px solid #e4e7ed;
    border-radius: 8px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.08);
    img {
      display: block;
      width: 200px;
      height: 200px;
      max-width: 100%;
    }
  }
  .qr-loading {
    text-align: center;
    padding: 40px 20px;
    color: #909399;
    .el-icon {
      font-size: 32px;
      margin-bottom: 12px;
    }
    p {
      margin: 0;
      font-size: 14px;
    }
  }
}

.payment-actions-compact {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.payment-result {
  text-align: center;
  padding: 20px 0;
}
.result-icon {
  font-size: 48px;
  margin-bottom: 15px;
}
.result-icon.success {
  color: #67c23a;
}
.result-icon.failed {
  color: #f56c6c;
}
.payment-result h3 {
  margin-bottom: 10px;
  color: #303133;
}
.payment-result p {
  margin-bottom: 15px;
  color: #606266;
}
.result-details,
.error-details {
  text-align: left;
  background: #f5f7fa;
  padding: 15px;
  border-radius: 4px;
  margin-top: 15px;
}
.result-details p,
.error-details p {
  margin: 5px 0;
  font-size: 14px;
}
.dialog-footer {
  text-align: right;
}
.dialog-footer .el-button {
  margin-left: 10px;
}

@media (max-width: 768px) {
  .desktop-only {
    display: none !important;
  }

  .payment-form {
    padding: 10px;
  }
  .payment-card {
    margin-bottom: 12px;
  }
  .order-info,
  .payment-methods {
    margin-bottom: 16px;
  }
  .payment-actions .el-button {
    width: 100%;
    margin: 8px 0;
  }

  .payment-qr-dialog {
    :deep(.el-dialog) {
      width: 92% !important;
      margin: 5vh auto !important;
      border-radius: 12px;
    }
    :deep(.el-dialog__header) {
      padding: 12px 16px;
    }
    :deep(.el-dialog__body) {
      padding: 12px 16px;
    }
  }

  .payment-qr-container {
    gap: 12px;
  }

  .order-info-compact {
    padding: 8px;
    .info-row {
      padding: 6px 0;
      .label {
        font-size: 13px;
      }
      .value {
        font-size: 13px;
        &.amount {
          font-size: 16px;
        }
      }
    }
  }

  .qr-code-wrapper-compact {
    padding: 8px 0;
    .qr-code {
      padding: 8px;
      img {
        width: 180px;
        height: 180px;
      }
    }
  }
}
</style>
