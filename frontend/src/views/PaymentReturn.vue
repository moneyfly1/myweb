<template>
  <div class="payment-return-container">
    <div class="payment-return-content">
      <div v-if="isLoading" class="loading-container">
        <el-icon class="is-loading"><Loading /></el-icon>
        <p>正在处理支付结果...</p>
      </div>
      <div v-else-if="paymentSuccess" class="success-container">
        <div class="success-content">
          <el-icon class="success-icon"><CircleCheckFilled /></el-icon>
          <h2 class="success-title">支付成功！</h2>
          <p class="success-subtitle">订单已支付，{{ orderConfig.subtitle }}</p>
          <el-descriptions :column="1" border style="max-width: 500px; margin: 30px auto;">
            <el-descriptions-item label="订单号">{{ orderNo }}</el-descriptions-item>
            <el-descriptions-item label="支付金额">¥{{ amount }}</el-descriptions-item>
            <el-descriptions-item label="支付状态">
              <el-tag type="success">已支付</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="订单类型">
              <el-tag :type="orderConfig.tagType">{{ orderConfig.label }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="套餐状态" v-if="orderType !== 'recharge' && orderType !== 'device_upgrade'">
              <el-tag type="success">已开通</el-tag>
            </el-descriptions-item>
          </el-descriptions>
          <div class="success-actions" style="margin-top: 30px;">
            <el-button type="primary" size="large" @click="goToOrders">查看订单</el-button>
            <el-button size="large" @click="goToDashboard" style="margin-left: 10px;">前往仪表盘</el-button>
          </div>
        </div>
      </div>
      <div v-else-if="errorMessage" class="error-container">
        <el-alert :title="errorMessage" type="error" :closable="false" show-icon />
        <div class="error-actions" style="margin-top: 20px; text-align: center;">
          <el-button type="primary" @click="goToDashboard">前往仪表盘</el-button>
          <el-button @click="goToOrders" style="margin-left: 10px;">查看订单</el-button>
        </div>
      </div>
    </div>
  </div>
</template>
<script>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from '@/utils/elementPlusServices'
import { Loading, CircleCheckFilled } from '@element-plus/icons-vue'
import { useApi, pendingPaymentStorage } from '@/utils/api'
export default {
  name: 'PaymentReturn',
  components: { Loading, CircleCheckFilled },
  setup() {
    const route = useRoute()
    const router = useRouter()
    const api = useApi()
    const orderNo = ref('')
    const amount = ref(0)
    const isLoading = ref(true)
    const paymentSuccess = ref(false)
    const errorMessage = ref('')
    const orderType = ref('order')
    let redirectTimer = null
    const normalizeOrderType = (type) => {
      if (type === 'recharge' || type === 'device_upgrade' || type === 'custom_package' || type === 'order') {
        return type
      }
      return 'order'
    }
    const orderConfig = computed(() => {
      const configs = {
        recharge: { subtitle: '充值已到账', label: '账户充值', tagType: 'info' },
        device_upgrade: { subtitle: '设备已升级', label: '设备升级', tagType: 'warning' },
        custom_package: { subtitle: '自定义套餐已开通', label: '自定义套餐', tagType: 'success' },
        order: { subtitle: '套餐已开通', label: '套餐开通', tagType: 'success' }
      }
      return configs[normalizeOrderType(orderType.value)] || configs.order
    })
    const getOrderType = (no) => {
      if (no.startsWith('RCH')) return 'recharge'
      if (no.startsWith('UPG')) return 'device_upgrade'
      return 'order'
    }
    const extractOrderNoFromUrl = (query) => {
      let no = query.out_trade_no || query.order_no || query.outTradeNo || query.orderNo
      if (Array.isArray(no)) no = no[0]
      if (typeof no === 'string' && no.includes(',')) no = no.split(',')[0].trim()
      return no ? String(no).trim() : null
    }
    const getPendingPaymentOrderNo = () => pendingPaymentStorage.get()?.order_no || null
    const fetchRecentOrderNo = async () => {
      try {
        const { orderAPI } = await import('@/utils/api')
        const res = await orderAPI.getUserOrders({ page: 1, size: 10 })
        if (res?.data?.success && res.data.data?.orders?.length) {
          const now = Date.now()
          const recent = res.data.data.orders.find(o => {
            const diff = now - new Date(o.created_at).getTime()
            return ['pending', 'unpaid', 'paid'].includes(o.status) && diff < 5 * 60 * 1000
          })
          return recent?.order_no || res.data.data.orders[0]?.order_no
        }
      } catch (e) {
      }
      return null
    }
    const refreshUserState = async (data) => {
      try {
        const { cachedAPI } = await import('@/utils/api')
        const type = normalizeOrderType(data?.type || orderType.value)
        await cachedAPI.refreshUserState({
          includeSubscription: type !== 'recharge'
        })
      } catch (e) {
      }
    }
    const handlePaymentSuccess = async (data) => {
      paymentSuccess.value = true
      isLoading.value = false
      amount.value = parseFloat(data.amount || 0)
      orderType.value = normalizeOrderType(data.type || orderType.value)
      const messages = {
        recharge: '支付成功！充值已到账！',
        device_upgrade: '支付成功！设备已升级！',
        custom_package: '支付成功！自定义套餐已开通！',
        order: '支付成功！套餐已开通！'
      }
      const key = normalizeOrderType(data.type || orderType.value)
      ElMessage.success(messages[key])
      await refreshUserState(data)
      pendingPaymentStorage.clear()
      redirectTimer = setTimeout(() => router.push('/orders'), 2000)
    }
    const fetchOrderData = async (no) => {
      try {
        const res = await api.get(`/orders/${no}/status`, { timeout: 10000 }) // 保持原有的10s超时
        return res.data.data || res.data
      } catch (e) {
        return null
      }
    }
    const pollOrderStatus = async (no) => {
      const maxChecks = 15
      let lastOrderData = null
      for (let i = 0; i < maxChecks; i++) {
        const data = await fetchOrderData(no)
        if (data) {
          lastOrderData = data
          if (data.amount) amount.value = parseFloat(data.amount)
          if (data.status === 'paid') {
            return data // 成功，返回数据
          }
          if (['cancelled', 'failed', 'expired'].includes(data.status)) {
            pendingPaymentStorage.clear()
            const statusTextMap = {
              cancelled: '已取消',
              failed: '支付失败',
              expired: '已过期'
            }
            throw new Error(`订单状态：${statusTextMap[data.status] || data.status}`)
          }
        }
        if (i < maxChecks - 1) {
          await new Promise(r => setTimeout(r, 2000))
        }
      }
      if (lastOrderData) {
        const statusText = lastOrderData.status === 'pending' ? '待支付' : 
                           lastOrderData.status === 'unpaid' ? '未支付' : lastOrderData.status
        throw new Error(`订单状态：${statusText}，请检查支付状态或稍后前往订单页面查看`)
      }
      throw new Error('无法获取订单状态，请稍后前往订单页面查看')
    }
    const processPaymentReturn = async () => {
      try {
        isLoading.value = true
        errorMessage.value = ''
        let no = extractOrderNoFromUrl(route.query)
        if (!no) {
          no = getPendingPaymentOrderNo()
        }
        if (!no) {
          no = await fetchRecentOrderNo()
        }
        if (!no) {
          errorMessage.value = '无法获取订单号，请稍后前往订单页面查看支付状态'
          isLoading.value = false
          redirectTimer = setTimeout(() => router.push('/orders'), 2000)
          return
        }
        orderNo.value = no
        orderType.value = getOrderType(no)
        if (route.query.trade_status === 'TRADE_SUCCESS' && route.query.pid) {
          for (let i = 0; i < 5; i++) {
            await new Promise(r => setTimeout(r, 1000))
            const fastData = await fetchOrderData(no)
            if (fastData && fastData.status === 'paid') {
              await handlePaymentSuccess(fastData)
              return
            }
          }
        }
        await new Promise(r => setTimeout(r, 500))
        const data = await pollOrderStatus(no)
        await handlePaymentSuccess(data)
      } catch (error) {
        isLoading.value = false
        errorMessage.value = error.message || '处理支付结果失败'
      }
    }
    const goToDashboard = () => router.push('/dashboard')
    const goToOrders = () => router.push('/orders')
    onMounted(processPaymentReturn)
    onUnmounted(() => {
      if (redirectTimer) {
        clearTimeout(redirectTimer)
        redirectTimer = null
      }
    })
    return {
      orderNo,
      amount,
      isLoading,
      paymentSuccess,
      errorMessage,
      orderType,
      orderConfig,
      goToDashboard,
      goToOrders
    }
  }
}
</script>
<style scoped lang="scss">
.payment-return-container {
  min-height: 100vh;
  background: #f5f7fa;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 20px;
}
.payment-return-content {
  width: 100%;
  max-width: 800px;
}
.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #909399;
  min-height: 400px;
  .el-icon {
    font-size: 48px;
    margin-bottom: 20px;
  }
  p {
    margin: 0;
    font-size: 16px;
  }
}
.success-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 400px;
  padding: 40px 20px;
}
.success-content {
  text-align: center;
  max-width: 600px;
  width: 100%;
}
.success-subtitle {
  font-size: 16px;
  color: #909399;
  margin: 10px 0 20px 0;
}
.success-icon {
  font-size: 80px;
  color: #67c23a;
  margin-bottom: 20px;
}
.success-title {
  font-size: 28px;
  color: #303133;
  margin: 0 0 20px 0;
  font-weight: 600;
}
.success-actions {
  margin-top: 40px;
}
.error-container {
  text-align: center;
  padding: 40px 20px;
}
.error-actions {
  margin-top: 20px;
}
@media (max-width: 768px) {
  .success-icon {
    font-size: 60px;
  }
  .success-title {
    font-size: 24px;
  }
}
</style>
