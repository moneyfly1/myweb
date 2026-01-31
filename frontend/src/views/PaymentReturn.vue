<template>
  <div class="payment-return-container">
    <div class="payment-return-content">
      <!-- 加载状态 -->
      <div v-if="isLoading" class="loading-container">
        <el-icon class="is-loading"><Loading /></el-icon>
        <p>正在处理支付结果...</p>
      </div>

      <!-- 成功状态 -->
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

      <!-- 错误状态 -->
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
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Loading, CircleCheckFilled } from '@element-plus/icons-vue'
import { useApi } from '@/utils/api'

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

    // UI 配置映射，替代 Template 中的大量 v-if/v-else
    const orderConfig = computed(() => {
      const configs = {
        recharge: { subtitle: '充值已到账', label: '账户充值', tagType: 'info' },
        device_upgrade: { subtitle: '设备已升级', label: '设备升级', tagType: 'warning' },
        default: { subtitle: '套餐已开通', label: '套餐开通', tagType: 'success' }
      }
      return configs[orderType.value] || configs.default
    })

    // 工具函数：根据订单号前缀判断类型
    const getOrderType = (no) => {
      if (no.startsWith('RCH')) return 'recharge'
      if (no.startsWith('UPG')) return 'device_upgrade'
      return 'order'
    }

    // 工具函数：从URL提取订单号（处理数组、逗号等情况）
    const extractOrderNoFromUrl = (query) => {
      let no = query.out_trade_no || query.order_no || query.outTradeNo || query.orderNo
      if (Array.isArray(no)) no = no[0]
      if (typeof no === 'string' && no.includes(',')) no = no.split(',')[0].trim()
      return no ? String(no).trim() : null
    }

    // 逻辑提取：获取最近订单
    const fetchRecentOrderNo = async () => {
      try {
        const { orderAPI } = await import('@/utils/api')
        const res = await orderAPI.getUserOrders({ page: 1, size: 10 })
        
        if (res?.data?.success && res.data.data?.orders?.length) {
          const now = Date.now()
          // 查找5分钟内的新订单
          const recent = res.data.data.orders.find(o => {
            const diff = now - new Date(o.created_at).getTime()
            return ['pending', 'unpaid', 'paid'].includes(o.status) && diff < 5 * 60 * 1000
          })
          return recent?.order_no || res.data.data.orders[0]?.order_no
        }
      } catch (e) {
        console.warn('Fetch recent order failed:', e)
      }
      return null
    }

    // 逻辑提取：刷新用户状态（余额/订阅）
    const refreshUserState = async (data) => {
      try {
        const { userAPI, subscriptionAPI } = await import('@/utils/api')
        const tasks = [userAPI.getUserInfo()]
        
        // 如果不是单纯的充值订单，通常需要刷新订阅状态
        if (data.type !== 'recharge' && orderType.value !== 'recharge') {
          tasks.push(subscriptionAPI.getSubscription())
        }

        await Promise.all(tasks)
        // 触发全局事件更新UI
        window.dispatchEvent(new CustomEvent('user-info-updated'))
        if (tasks.length > 1) {
          window.dispatchEvent(new CustomEvent('subscription-updated'))
        }
      } catch (e) {
        console.warn('Refresh user state failed:', e)
      }
    }

    // 统一处理支付成功逻辑
    const handlePaymentSuccess = async (data) => {
      paymentSuccess.value = true
      isLoading.value = false
      amount.value = parseFloat(data.amount || 0)
      
      const messages = {
        recharge: '支付成功！充值已到账！',
        device_upgrade: '支付成功！设备已升级！',
        default: '支付成功！套餐已开通！'
      }
      
      const key = (data.type === 'recharge' || orderType.value === 'recharge') ? 'recharge' :
                  (data.type === 'device_upgrade' || orderType.value === 'device_upgrade') ? 'device_upgrade' : 'default'
      
      ElMessage.success(messages[key])
      
      // 执行刷新逻辑：立即刷新 + 延迟刷新（双重保险）
      await refreshUserState(data)
      setTimeout(async () => await refreshUserState(data), 500)
      
      setTimeout(() => router.push('/orders'), 2000)
    }

    // 核心逻辑：查询API并返回完整数据对象
    const fetchOrderData = async (no) => {
      try {
        const res = await api.get(`/orders/${no}/status`, { timeout: 10000 }) // 保持原有的10s超时
        return res.data.data || res.data
      } catch (e) {
        console.warn('Fetch order status failed:', e)
        return null
      }
    }

    // 轮询逻辑：完全保留原有的重试机制
    const pollOrderStatus = async (no) => {
      const maxChecks = 15
      let lastOrderData = null

      for (let i = 0; i < maxChecks; i++) {
        const data = await fetchOrderData(no)
        
        if (data) {
          lastOrderData = data
          // 实时更新金额（原代码逻辑）
          if (data.amount) amount.value = parseFloat(data.amount)
          
          if (data.status === 'paid') {
            return data // 成功，返回数据
          }
        }
        
        // 未成功，等待2秒后重试
        if (i < maxChecks - 1) {
          await new Promise(r => setTimeout(r, 2000))
        }
      }
      
      // 轮询结束仍未支付，根据最后的状态决定报错信息
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

        // 1. 获取订单号
        let no = extractOrderNoFromUrl(route.query)
        if (!no) {
          no = await fetchRecentOrderNo()
        }

        if (!no) {
          // 无法获取订单号时的处理
          errorMessage.value = '无法获取订单号，请稍后前往订单页面查看支付状态'
          isLoading.value = false
          setTimeout(() => router.push('/orders'), 2000)
          return
        }

        orderNo.value = no
        orderType.value = getOrderType(no)

        // 2. 快速通道：如果是易支付同步回调且状态成功
        if (route.query.trade_status === 'TRADE_SUCCESS' && route.query.pid) {
          await new Promise(r => setTimeout(r, 500))
          const fastData = await fetchOrderData(no)
          if (fastData && fastData.status === 'paid') {
            await handlePaymentSuccess(fastData)
            return
          }
        }

        // 3. 常规轮询通道：先等待2秒让后端处理回调
        await new Promise(r => setTimeout(r, 2000))
        
        const data = await pollOrderStatus(no)
        await handlePaymentSuccess(data)

      } catch (error) {
        // 错误处理
        isLoading.value = false
        errorMessage.value = error.message || '处理支付结果失败'
        // 如果只是超时但没有明确报错，不要立即跳转，让用户看到错误信息
      }
    }

    const goToDashboard = () => router.push('/dashboard')
    const goToOrders = () => router.push('/orders')

    onMounted(processPaymentReturn)

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