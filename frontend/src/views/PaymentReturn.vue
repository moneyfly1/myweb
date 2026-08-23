<template>
  <div class="list-container payment-return-container">
    <div class="breadcrumb">首页 / 支付返回</div>
    <div class="page-header">
      <div class="page-title">
        <h1>支付返回</h1>
      </div>
    </div>
    <div class="payment-return-grid payment-state-grid">
      <div v-if="isLoading" class="payment-state-card active">
        <div class="dialog-title">
          正在处理支付结果
          <el-tag type="warning">当前状态</el-tag>
        </div>
        <LoadingState
          text="正在轮询订单状态，请稍候..."
          :size="32"
          class="dialog-body loading-container"
        />
      </div>
      <div v-else-if="paymentSuccess" class="payment-state-card active">
        <div class="dialog-title">
          支付成功
          <el-tag type="success">当前状态</el-tag>
        </div>
        <div class="dialog-body success-content">
          <el-icon class="success-icon"><CircleCheckFilled /></el-icon>
          <h2 class="success-title">订单已支付</h2>
          <p class="success-subtitle">{{ orderConfig.subtitle }}</p>
          <div class="summary-list">
            <div class="summary-row"><span>订单号</span><strong>{{ orderNo || '-' }}</strong></div>
            <div class="summary-row"><span>支付金额</span><strong>¥{{ amount }}</strong></div>
            <div class="summary-row"><span>订单类型</span><strong>{{ orderConfig.label }}</strong></div>
            <div class="summary-row"><span>状态</span><strong><el-tag type="success">已支付</el-tag></strong></div>
          </div>
          <div class="button-row">
            <el-button type="primary" @click="goToOrders">查看订单</el-button>
            <el-button @click="goToDashboard">前往仪表盘</el-button>
          </div>
        </div>
      </div>
      <div v-else class="payment-state-card active">
        <div class="dialog-title">
          处理失败
          <el-tag type="danger">当前状态</el-tag>
        </div>
        <div class="dialog-body error-container">
          <el-alert :title="errorMessage || '等待支付结果返回'" type="error" :closable="false" show-icon />
          <div class="button-row">
            <el-button type="primary" @click="goToOrders">查看订单</el-button>
            <el-button @click="goToDashboard">前往仪表盘</el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from '@/utils/elementPlusServices'
import { CircleCheckFilled } from '@element-plus/icons-vue'
import LoadingState from '@/components/LoadingState.vue'
import { useApi, pendingPaymentStorage } from '@/utils/api'
export default {
  name: 'PaymentReturn',
  components: { LoadingState, CircleCheckFilled },
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
        // 组件已卸载则停止轮询，避免劫持用户后续导航/请求
        if (unmountedFlag) return null
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
    let unmountedFlag = false
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
            if (unmountedFlag) return
            await new Promise(r => setTimeout(r, 1000))
            const fastData = await fetchOrderData(no)
            if (fastData && fastData.status === 'paid') {
              await handlePaymentSuccess(fastData)
              return
            }
          }
        }
        if (unmountedFlag) return
        await new Promise(r => setTimeout(r, 500))
        const data = await pollOrderStatus(no)
        if (data) await handlePaymentSuccess(data)
      } catch (error) {
        isLoading.value = false
        errorMessage.value = error.message || '处理支付结果失败'
      }
    }
    const goToDashboard = () => router.push('/dashboard')
    const goToOrders = () => router.push('/orders')
    onMounted(processPaymentReturn)
    onUnmounted(() => {
      unmountedFlag = true
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
  padding: 0;
  max-width: none;
  margin: 0;
  width: 100%;
}
.breadcrumb {
  margin-bottom: 12px;
  color: #606266;
  font-size: 13px;
}
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
  padding: 16px;
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
}
.page-header h1 {
  margin: 0;
  color: #303133;
  font-size: 22px;
  line-height: 1.25;
}
.page-header p {
  margin: 6px 0 0;
  color: #606266;
  line-height: 1.5;
}
.payment-return-grid {
  display: grid;
  max-width: 720px;
  margin: 0 auto;
}
.payment-state-card {
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  overflow: hidden;
}
.payment-state-card.active {
  border-color: #409eff;
}
.dialog-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 14px 16px;
  border-bottom: 1px solid #ebeef5;
  color: #303133;
  font-size: 16px;
  font-weight: 700;
}
.dialog-body {
  padding: 16px;
}
.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #909399;
  min-height: 260px;
  .el-icon {
    font-size: 48px;
    margin-bottom: 20px;
  }
  p {
    margin: 0;
    font-size: 16px;
  }
}
.success-content {
  text-align: center;
}
.success-subtitle {
  font-size: 16px;
  color: #909399;
  margin: 10px 0 20px 0;
}
.success-icon {
  font-size: 72px;
  color: #67c23a;
  margin-bottom: 20px;
}
.success-title {
  font-size: 24px;
  color: #303133;
  margin: 0 0 20px 0;
  font-weight: 600;
}
.summary-list {
  margin: 18px 0;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  overflow: hidden;
  text-align: left;
}
.summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-bottom: 1px solid #ebeef5;
  color: #606266;
  font-size: 13px;
}
.summary-row:last-child {
  border-bottom: none;
}
.summary-row strong {
  min-width: 0;
  color: #303133;
  font-weight: 600;
  text-align: right;
  word-break: break-word;
}
.button-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.error-container {
  min-height: 260px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}
@media (max-width: 768px) {
  .page-header {
    padding: 14px;
  }
  .cols-3 {
    grid-template-columns: 1fr;
  }
  .loading-container,
  .error-container {
    min-height: 220px;
  }
  .success-icon {
    font-size: 60px;
  }
  .success-title {
    font-size: 24px;
  }
  .button-row {
    flex-direction: column;
    .el-button {
      width: 100%;
      margin-left: 0;
    }
  }
}
</style>
