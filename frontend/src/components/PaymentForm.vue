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
      <div class="payment-tips">
        <el-alert
          v-if="selectedPaymentMethod === 'alipay'"
          title="支付宝支付提示"
          type="info"
          :closable="false"
          show-icon
        >
          <p>1. 点击支付按钮后将跳转到支付宝</p>
          <p>2. 请在支付宝中完成支付</p>
          <p>3. 支付完成后将自动返回</p>
        </el-alert>
        <el-alert
          v-if="selectedPaymentMethod === 'wechat'"
          title="微信支付提示"
          type="info"
          :closable="false"
          show-icon
        >
          <p>1. 点击支付按钮后将显示二维码</p>
          <p>2. 请使用微信扫描二维码完成支付</p>
          <p>3. 支付完成后将自动刷新状态</p>
        </el-alert>
      </div>
    </el-card>
    <el-dialog
      v-model="wechatQRVisible"
      :title="selectedPaymentMethod === 'alipay' ? '支付宝支付' : '微信支付'"
      width="400px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
      <div class="wechat-qr-container">
        <div class="qr-code-wrapper">
          <div v-if="wechatQRCode" class="qr-code">
            <img :src="wechatQRCode" alt="微信支付二维码" />
          </div>
          <div v-else class="qr-loading">
            <el-icon class="is-loading"><Loading /></el-icon>
            <p>正在生成二维码...</p>
          </div>
        </div>
        <div class="qr-tips">
          <p>请使用{{ selectedPaymentMethod === 'alipay' ? '支付宝' : '微信' }}扫描二维码完成支付</p>
          <p>支付完成后请勿关闭此窗口</p>
        </div>
        <div class="qr-actions">
          <el-button @click="checkPaymentStatus" :loading="isCheckingStatus">
            检查支付状态
          </el-button>
          <el-button type="primary" @click="wechatQRVisible = false">
            支付完成
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
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { CircleCheckFilled, CircleCloseFilled, Loading } from '@element-plus/icons-vue'
import { useApi } from '@/utils/api'
import { formatDateTime } from '@/utils/date'

export default {
  name: 'PaymentForm',
  components: {
    CircleCheckFilled,
    CircleCloseFilled,
    Loading
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
    const paymentResult = reactive({
      success: false,
      message: '',
      data: null,
      error_code: '',
      error_message: ''
    })
    let statusCheckTimer = null
    
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
    })
    onUnmounted(() => {
      if (statusCheckTimer) {
        clearInterval(statusCheckTimer)
      }
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
      onPaymentMethodChange,
      getPaymentButtonText,
      handlePayment,
      checkPaymentStatus,
      retryPayment,
      handleResultClose,
      getPaymentDescription,
      formatDateTime
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
.alipay-icon {
  background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="%2300a0e9" d="M22.319 4.609c-.977.377-2.04.777-3.18 1.196-1.14.419-2.38.839-3.72 1.259-1.34.42-2.78.84-4.32 1.26-1.54.42-3.18.84-4.92 1.26-1.74.42-3.58.84-5.52 1.26-1.94.42-3.98.84-6.12 1.26v1.5c2.14.42 4.18.84 6.12 1.26 1.94.42 3.78.84 5.52 1.26 1.74.42 3.38.84 4.92 1.26 1.54.42 2.98.84 4.32 1.26 1.34.42 2.58.84 3.72 1.259 1.14.419 2.203.819 3.18 1.196.977.377 1.84.754 2.58 1.131.74.377 1.36.754 1.86 1.131.5.377.88.754 1.14 1.131.26.377.4.754.4 1.131 0 .377-.14.754-.4 1.131-.26.377-.64.754-1.14 1.131-.5.377-1.12.754-1.86 1.131-.74.377-1.603.754-2.58 1.131-.977.377-2.04.777-3.18 1.196-1.14.419-2.38.839-3.72 1.259-1.34.42-2.78.84-4.32 1.26-1.54.42-3.18.84-4.92 1.26-1.74.42-3.58.84-5.52 1.26-1.94.42-3.98.84-6.12 1.26v1.5c2.14.42 4.18.84 6.12 1.26 1.94.42 3.78.84 5.52 1.26 1.74.42 3.38.84 4.92 1.26 1.54.42 2.98.84 4.32 1.26 1.34.42 2.58.84 3.72 1.259 1.14.419 2.203.819 3.18 1.196.977.377 1.84.754 2.58 1.131.74.377 1.36.754 1.86 1.131.5.377.88.754 1.14 1.131.26.377.4.754.4 1.131 0 .377-.14.754-.4 1.131-.26.377-.64.754-1.14 1.131-.5.377-1.12.754-1.86 1.131-.74.377-1.603.754-2.58 1.131z"/></svg>') no-repeat center;
  background-size: contain;
}
.wechat-icon {
  background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="%2307c160" d="M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.17 4.203 3.002 5.55a.59.59 0 0 1 .213.665l-.39 1.48c-.019.07-.048.141-.048.212 0 .163.13.295.29.295a.326.326 0 0 0 .167-.054l1.903-1.114a.864.864 0 0 1 .717-.098 10.16 10.16 0 0 0 2.837.403c.276 0 .543-.027.811-.05-.857-2.578.157-4.972 1.932-6.446 1.703-1.415 4.882-1.932 6.109-.207 1.227 1.725.792 4.82-.207 6.109-1.932 1.703-4.972 1.703-6.109.207-1.227-1.725-.792-4.82.207-6.109 1.932-1.703 4.972-1.703 6.109-.207z"/></svg>') no-repeat center;
  background-size: contain;
}
.paypal-icon {
  background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="%230073B6" d="M7.076 21.337H2.47a.641.641 0 0 1-.633-.74L4.944.901C5.026.382 5.474 0 5.998 0h7.46c2.57 0 4.578.543 5.69 1.81 1.01 1.15 1.304 2.42 1.012 4.287-.023.143-.047.288-.077.437-.983 5.05-4.349 6.797-8.647 6.797h-2.19c-.524 0-.968.382-1.05.9l-1.12 7.106zm14.146-14.42a3.35 3.35 0 0 0-.105-.726c-1.263-5.05-4.349-6.797-8.647-6.797H5.998c-.524 0-.968.382-1.05.9L2.47 20.597h4.606l1.12-7.106c.082-.518.526-.9 1.05-.9h2.19c4.298 0 7.384-1.747 8.647-6.797.023-.143.047-.288.077-.437z"/></svg>') no-repeat center;
  background-size: contain;
}
.stripe-icon {
  background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="%23635BFF" d="M13.976 9.15c-2.172-.806-3.356-1.426-3.356-2.409 0-.831.683-1.305 1.901-1.305 2.227 0 4.515.858 6.09 1.631l.89-5.494C18.252.274 15.697 0 12.165 0 9.667 0 7.589.654 6.104 1.872 4.56 3.147 3.757 4.992 3.757 7.218c0 4.039 2.467 5.76 6.476 7.219 2.585.92 3.445 1.574 3.445 2.583 0 .98-.84 1.407-2.354 1.407-1.905 0-4.357-.932-5.9-1.756L4.717 21.35c1.57.921 3.71 1.65 6.305 1.65 2.66 0 4.812-.654 6.218-1.85 1.531-1.305 2.227-3.147 2.227-5.4 0-3.77-2.227-5.4-6.491-7.1z"/></svg>') no-repeat center;
  background-size: contain;
}
.bank-icon {
  background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="%23606266" d="M12 2L2 7v10c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V7l-10-5zM4 17V9h16v8H4zm2-6h2v4H6v-4zm4 0h2v4h-2v-4zm4 0h2v4h-2v-4z"/></svg>') no-repeat center;
  background-size: contain;
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
  margin-top: 20px;
}
.payment-tips .el-alert {
  margin-bottom: 10px;
}
.payment-tips p {
  margin: 5px 0;
  font-size: 14px;
}
.wechat-qr-container {
  text-align: center;
}
.qr-code-wrapper {
  margin-bottom: 20px;
}
.qr-code img {
  max-width: 200px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
}
.qr-loading {
  padding: 40px;
  color: #909399;
}
.qr-loading .el-icon {
  font-size: 24px;
  margin-bottom: 10px;
}
.qr-tips {
  margin-bottom: 20px;
  color: #606266;
}
.qr-tips p {
  margin: 5px 0;
}
.qr-actions .el-button {
  margin: 0 10px;
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
</style>
