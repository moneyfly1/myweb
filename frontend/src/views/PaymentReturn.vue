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
          <p class="success-subtitle">订单已支付，套餐已开通</p>
          <el-descriptions :column="1" border style="max-width: 500px; margin: 30px auto;">
            <el-descriptions-item label="订单号">{{ orderNo }}</el-descriptions-item>
            <el-descriptions-item label="支付金额">¥{{ amount }}</el-descriptions-item>
            <el-descriptions-item label="支付状态">
              <el-tag type="success">已支付</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="套餐状态">
              <el-tag type="success">已开通</el-tag>
            </el-descriptions-item>
          </el-descriptions>
          <div class="success-actions" style="margin-top: 30px;">
            <el-button type="primary" size="large" @click="goToOrders">
              查看订单
            </el-button>
            <el-button size="large" @click="goToDashboard" style="margin-left: 10px;">
              前往仪表盘
            </el-button>
          </div>
        </div>
      </div>

      <div v-else-if="errorMessage" class="error-container">
        <el-alert
          :title="errorMessage"
          type="error"
          :closable="false"
          show-icon
        />
        <div class="error-actions" style="margin-top: 20px; text-align: center;">
          <el-button type="primary" @click="goToDashboard">
            前往仪表盘
          </el-button>
          <el-button @click="goToOrders" style="margin-left: 10px;">
            查看订单
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Loading, CircleCheckFilled } from '@element-plus/icons-vue'
import { useApi } from '@/utils/api'

export default {
  name: 'PaymentReturn',
  components: {
    Loading,
    CircleCheckFilled
  },
  setup() {
    const route = useRoute()
    const router = useRouter()
    const api = useApi()

    const orderNo = ref('')
    const amount = ref(0)
    const isLoading = ref(true)
    const paymentSuccess = ref(false)
    const errorMessage = ref('')

    const processPaymentReturn = async () => {
      try {
        isLoading.value = true
        errorMessage.value = ''

        // 从URL参数中提取订单号和支付状态
        const urlParams = route.query
        let orderNoParam = urlParams.out_trade_no || 
                          urlParams.order_no || 
                          urlParams.outTradeNo ||
                          urlParams.orderNo
        
        // 如果订单号包含逗号，说明参数重复了，取第一个
        if (orderNoParam && orderNoParam.includes(',')) {
          orderNoParam = orderNoParam.split(',')[0].trim()
          console.log('PaymentReturn: 检测到重复的订单号参数，已修正为:', orderNoParam)
        }
        
        // 检查是否是易支付同步回调（带有trade_status参数）
        const isYipayReturn = urlParams.trade_status && urlParams.pid
        const tradeStatus = urlParams.trade_status
        
        console.log('PaymentReturn: URL参数:', {
          orderNo: orderNoParam,
          isYipayReturn,
          tradeStatus,
          hasSign: !!urlParams.sign,
          pid: urlParams.pid
        })

        // 如果URL中没有订单号，尝试从用户最近订单中获取
        if (!orderNoParam) {
          console.log('PaymentReturn: URL参数中没有订单号，尝试从用户最近订单中获取')
          try {
            const { orderAPI } = await import('@/utils/api')

            const ordersResponse = await orderAPI.getUserOrders({ 
              page: 1, 
              size: 10
            })
            
            if (ordersResponse?.data?.success && ordersResponse.data.data?.orders) {
              const orders = ordersResponse.data.data.orders

              const now = Date.now()
              const recentOrder = orders.find(order => {
                const orderTime = new Date(order.created_at).getTime()
                const timeDiff = now - orderTime

                return (order.status === 'pending' || 
                       order.status === 'unpaid' || 
                       order.status === 'paid') &&
                       timeDiff < 5 * 60 * 1000
              })
              
              if (recentOrder && recentOrder.order_no) {
                orderNoParam = recentOrder.order_no
                console.log('PaymentReturn: 从用户最近订单中获取到订单号:', orderNoParam)
              } else if (orders.length > 0 && orders[0].order_no) {

                orderNoParam = orders[0].order_no
                console.log('PaymentReturn: 使用最近的订单号:', orderNoParam)
              }
            }
          } catch (error) {
            console.warn('PaymentReturn: 获取用户订单失败:', error)
          }
        }

        if (!orderNoParam) {
          errorMessage.value = '无法获取订单号，请稍后前往订单页面查看支付状态'
          isLoading.value = false
          setTimeout(() => {
            router.push('/orders')
          }, 2000)
          return
        }

        orderNo.value = orderNoParam
        console.log('PaymentReturn: 使用订单号:', orderNo.value)

        // 如果是易支付同步回调且支付状态为成功，立即验证并显示
        if (isYipayReturn && tradeStatus === 'TRADE_SUCCESS') {
          console.log('PaymentReturn: 检测到易支付同步回调，支付状态为TRADE_SUCCESS，立即查询订单')
          
          // 等待500ms，给异步回调一点处理时间
          await new Promise(resolve => setTimeout(resolve, 500))
          
          // 立即查询订单状态
          try {
            const response = await api.get(`/orders/${orderNo.value}/status`, {
              timeout: 5000
            })
            
            console.log('PaymentReturn: 同步回调后立即查询响应:', response)
            
            const orderData = response.data.data || response.data
            if (orderData && orderData.status === 'paid') {
              // 订单已支付，立即显示成功
              paymentSuccess.value = true
              isLoading.value = false
              amount.value = parseFloat(orderData.amount || 0)
              
              ElMessage.success('支付成功！套餐已开通！')
              
              // 刷新用户信息
              try {
                const { userAPI, subscriptionAPI } = await import('@/utils/api')
                await Promise.all([
                  userAPI.getUserInfo(),
                  subscriptionAPI.getSubscription()
                ])
                window.dispatchEvent(new CustomEvent('subscription-updated'))
                window.dispatchEvent(new CustomEvent('user-info-updated'))
              } catch (error) {
                console.warn('PaymentReturn: 刷新用户信息失败:', error)
              }
              
              // 2秒后跳转到订单页面
              setTimeout(() => {
                router.push('/orders')
              }, 2000)
              return
            } else {
              console.log('PaymentReturn: 同步回调后查询订单状态仍为:', orderData?.status, '继续轮询')
            }
          } catch (error) {
            console.warn('PaymentReturn: 同步回调后立即查询失败:', error)
            // 继续下面的轮询逻辑
          }
        }

        // 等待一下让后端处理回调
        await new Promise(resolve => setTimeout(resolve, 2000))


        let checkCount = 0
        const maxChecks = 15
        let orderData = null

        while (checkCount < maxChecks && !paymentSuccess.value) {
          checkCount++
          console.log(`PaymentReturn: 第${checkCount}次检查订单状态...`)
          
          try {
            const response = await api.get(`/orders/${orderNo.value}/status`, {
              timeout: 10000
            })

            console.log(`PaymentReturn: 第${checkCount}次查询订单状态响应:`, response)

            if (!response || !response.data) {
              console.warn(`PaymentReturn: 响应数据为空，第${checkCount}次`)
              if (checkCount >= maxChecks) {
                errorMessage.value = '无法获取订单状态，请稍后重试'
                isLoading.value = false
                return
              }
              await new Promise(resolve => setTimeout(resolve, 2000))
              continue
            }

            if (response.data.success === false) {
              console.warn(`PaymentReturn: API返回失败，第${checkCount}次:`, response.data.message)
              if (checkCount >= maxChecks) {
                errorMessage.value = response.data.message || '获取订单状态失败'
                isLoading.value = false
                return
              }
              await new Promise(resolve => setTimeout(resolve, 2000))
              continue
            }

            orderData = response.data.data || response.data
            if (!orderData) {
              console.warn(`PaymentReturn: 订单数据不存在，第${checkCount}次，响应:`, response.data)
              if (checkCount >= maxChecks) {
                errorMessage.value = '订单数据不存在'
                isLoading.value = false
                return
              }
              await new Promise(resolve => setTimeout(resolve, 2000))
              continue
            }

            console.log(`PaymentReturn: 订单数据:`, orderData, `订单状态:`, orderData.status)

            amount.value = parseFloat(orderData.amount || 0)
            
            if (!orderData.status) {
              console.warn(`PaymentReturn: 订单状态字段不存在，第${checkCount}次`)
              if (checkCount >= maxChecks) {
                errorMessage.value = '订单状态字段不存在'
                isLoading.value = false
                return
              }
              await new Promise(resolve => setTimeout(resolve, 2000))
              continue
            }


            if (orderData.status === 'paid') {
              paymentSuccess.value = true
              isLoading.value = false
              ElMessage.success('支付成功！回调成功！套餐已开通！')


              try {
                const { userAPI, subscriptionAPI } = await import('@/utils/api')
                await Promise.all([
                  userAPI.getUserInfo(),
                  subscriptionAPI.getSubscription()
                ])
              } catch (error) {
                console.warn('PaymentReturn: 刷新用户信息失败:', error)
              }


              window.dispatchEvent(new CustomEvent('subscription-updated'))
              window.dispatchEvent(new CustomEvent('user-info-updated'))


              setTimeout(async () => {
                try {
                  const { userAPI, subscriptionAPI } = await import('@/utils/api')
                  await Promise.all([
                    userAPI.getUserInfo(),
                    subscriptionAPI.getSubscription()
                  ])
                  window.dispatchEvent(new CustomEvent('subscription-updated'))
                  window.dispatchEvent(new CustomEvent('user-info-updated'))
                } catch (error) {
                  console.warn('PaymentReturn: 二次刷新用户信息失败:', error)
                }
              }, 500)


              setTimeout(() => {
                router.push('/orders')
              }, 2000)
              return
            }


            console.log(`PaymentReturn: 订单状态: ${orderData.status}，继续检查... (${checkCount}/${maxChecks})`)
            
            if (checkCount < maxChecks) {
              await new Promise(resolve => setTimeout(resolve, 2000))
            }
          } catch (error) {
            console.warn(`PaymentReturn: 第${checkCount}次检查订单状态失败:`, error)
            if (checkCount >= maxChecks) {
              errorMessage.value = '无法获取订单状态，请稍后前往订单页面查看'
              isLoading.value = false
              return
            }
            await new Promise(resolve => setTimeout(resolve, 2000))
          }
        }


        if (!paymentSuccess.value && orderData) {
          errorMessage.value = `订单状态：${orderData.status === 'pending' ? '待支付' : orderData.status === 'unpaid' ? '未支付' : orderData.status}，请检查支付状态或稍后前往订单页面查看`
          isLoading.value = false
        } else if (!paymentSuccess.value) {
          errorMessage.value = '订单状态未更新，请前往订单页面查看支付状态'
          isLoading.value = false
        }
      } catch (error) {
        console.error('处理支付返回失败:', error)
        errorMessage.value = '处理支付结果失败: ' + (error.message || '未知错误')
        isLoading.value = false
      }
    }

    const goToDashboard = () => {
      router.push('/dashboard')
    }

    const goToOrders = () => {
      router.push('/orders')
    }

    onMounted(() => {
      processPaymentReturn()
    })

    return {
      orderNo,
      amount,
      isLoading,
      paymentSuccess,
      errorMessage,
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

.success-message {
  font-size: 16px;
  color: #606266;
  margin: 10px 0;
}

.success-steps {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 40px;
  margin: 40px 0;
  flex-wrap: wrap;
}

.success-step {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  color: #303133;
}

.step-icon {
  font-size: 32px;
  color: #67c23a;
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
  
  .success-steps {
    gap: 20px;
  }
  
  .success-step {
    font-size: 14px;
  }
  
  .step-icon {
    font-size: 24px;
  }
}
</style>
