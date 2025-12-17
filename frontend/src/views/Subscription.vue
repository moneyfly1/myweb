<template>
  <div class="list-container subscription-container">
    <el-card class="subscription-card">
      <template #header>
        <div class="card-header">
          <h2>订阅管理</h2>
          <p>管理您的订阅信息和订阅地址</p>
        </div>
      </template>

      <div class="subscription-status" v-if="subscription">
        <el-row :gutter="20">
          <el-col :xs="24" :sm="12" :md="6" :lg="6" :xl="6">
            <div class="status-item">
              <div class="status-label">账号状态</div>
              <div class="status-value">
                <el-tag :type="getStatusType(subscription)">
                  {{ getStatusText(subscription) }}
                </el-tag>
              </div>
            </div>
          </el-col>
          <el-col :xs="24" :sm="12" :md="6" :lg="6" :xl="6">
            <div class="status-item">
              <div class="status-label">到期时间</div>
              <div class="status-value">{{ formatDate(subscription.expire_time) }}</div>
            </div>
          </el-col>
          <el-col :xs="24" :sm="12" :md="6" :lg="6" :xl="6">
            <div class="status-item">
              <div class="status-label">到期天数</div>
              <div class="status-value">{{ getRemainingDays(subscription) }} 天</div>
            </div>
          </el-col>
          <el-col :xs="24" :sm="12" :md="6" :lg="6" :xl="6">
            <div class="status-item">
              <div class="status-label">设备使用</div>
              <div class="status-value">
                <el-tooltip content="在线设备数 / 允许最大设备数" placement="top">
                  <span>{{ subscription.onlineDevices || subscription.current_devices || 0 }}/{{ subscription.device_limit || subscription.maxDevices || 0 }}</span>
                </el-tooltip>
              </div>
            </div>
          </el-col>
        </el-row>
      </div>

      <div class="subscription-urls" v-if="subscription && (subscription.subscription_id || subscription.clash_url)">
        <h3>订阅地址</h3>
        <div class="url-list">
          <div class="url-item">
            <div class="url-label">Clash订阅地址：</div>
            <div class="url-content">
              <el-input
                v-model="subscription.clash_url"
                readonly
                size="large"
              >
                <template #append>
                  <el-button @click="copyUrl(subscription.clash_url)">
                    <el-icon><DocumentCopy /></el-icon>
                    复制
                  </el-button>
                </template>
              </el-input>
            </div>
          </div>
          
          <div class="url-item">
            <div class="url-label">通用订阅地址：</div>
            <div class="url-content">
              <el-input
                v-model="subscription.universal_url"
                readonly
                size="large"
              >
                <template #append>
                  <el-button @click="copyUrl(subscription.universal_url)">
                    <el-icon><DocumentCopy /></el-icon>
                    复制
                  </el-button>
                </template>
              </el-input>
            </div>
          </div>
        </div>

        <div class="qr-code-section">
          <h4>订阅二维码（Shadowrocket扫码）</h4>
          <div class="qr-codes">
            <div class="qr-item">
              <canvas id="subscription-qrcode"></canvas>
              <p v-if="subscription.expire_time && subscription.expire_time !== '未设置'">
                到期时间：{{ formatDate(subscription.expire_time) }}
              </p>
              <p v-else>通用订阅</p>
            </div>
          </div>
        </div>
      </div>

      <div class="no-subscription" v-else>
        <el-empty description="您还没有订阅">
          <el-button type="primary" @click="$router.push('/packages')">
            立即订阅
          </el-button>
        </el-empty>
      </div>

      <div class="subscription-actions" v-if="subscription && (subscription.subscription_id || subscription.clash_url)">
        <el-button
          type="primary"
          class="action-btn reset-btn"
          @click="resetSubscription"
          :loading="resetLoading"
        >
          重置订阅地址
        </el-button>
        
        <el-button
          type="success"
          class="action-btn email-btn"
          @click="sendSubscriptionToEmail"
          :loading="sendEmailLoading"
        >
          发送到邮箱
        </el-button>
        
        <el-button
          type="warning"
          class="action-btn renew-btn"
          @click="$router.push('/packages')"
          v-if="!isSubscriptionActive(subscription)"
        >
          续费订阅
        </el-button>
        
        <el-button
          type="primary"
          class="action-btn upgrade-btn"
          @click="showUpgradeDialog = true"
          v-if="isSubscriptionActive(subscription)"
        >
          升级设备数量
        </el-button>
      </div>

      <el-dialog
        v-model="showUpgradeDialog"
        title="升级设备数量"
        width="90%"
        :close-on-click-modal="false"
        class="upgrade-dialog"
        :max-width="600"
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
            
            <el-form-item label="增加设备数量">
              <el-select
                v-model="upgradeForm.additionalDevices"
                @change="calculateUpgradeCost"
                style="width: 100%"
                placeholder="请选择增加的设备数量"
              >
                <el-option
                  v-for="count in deviceOptions"
                  :key="count"
                  :label="`${count} 个设备`"
                  :value="count"
                />
              </el-select>
              <div class="form-hint">将增加 {{ upgradeForm.additionalDevices || 0 }} 个设备（只能按5个递增）</div>
            </el-form-item>

            <el-form-item label="延长到期时间（可选）">
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
              <div class="form-hint">将延长 {{ upgradeForm.additionalDays || 0 }} 天（只能按月递增，1个月=30天）</div>
            </el-form-item>
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

          <div class="payment-method" v-if="finalAmount > 0">
            <h4>支付方式</h4>
            <div class="balance-info">
              <span>账户余额：¥{{ userBalance.toFixed(2) }}</span>
            </div>
            <el-radio-group v-model="paymentMethod" @change="handlePaymentMethodChange">
              <el-radio label="alipay">支付宝支付</el-radio>
              <el-radio label="balance" :disabled="userBalance <= 0">余额支付</el-radio>
              <el-radio label="mixed" :disabled="userBalance <= 0 || userBalance >= finalAmount">
                余额+支付宝（余额不足时）
              </el-radio>
            </el-radio-group>
            <div class="payment-amount" v-if="paymentMethod === 'mixed'">
              <p>余额支付：¥{{ Math.min(userBalance, finalAmount).toFixed(2) }}</p>
              <p>支付宝支付：¥{{ Math.max(0, finalAmount - userBalance).toFixed(2) }}</p>
            </div>
          </div>
        </div>

        <template #footer>
          <div class="dialog-footer">
            <el-button @click="showUpgradeDialog = false">取消</el-button>
            <el-button 
              type="primary" 
              @click="confirmUpgrade"
              :loading="upgradeLoading"
              :disabled="!upgradeForm.additionalDevices || upgradeForm.additionalDevices < 5"
            >
              确认升级设备数量并支付
            </el-button>
          </div>
        </template>
      </el-dialog>

      <div class="renewal-prompt" v-if="subscription && !isSubscriptionActive(subscription)">
        <el-alert
          title="订阅已过期"
          type="warning"
          :description="`您的订阅已于 ${formatDate(subscription.expire_time)} 过期，请及时续费以继续使用服务。`"
          show-icon
          :closable="false"
        >
          <template #default>
            <div class="renewal-actions">
              <el-button type="primary" @click="$router.push('/packages')">
                立即续费
              </el-button>
            </div>
          </template>
        </el-alert>
      </div>
    </el-card>

    <el-dialog
      v-model="paymentQRVisible"
      title="扫码支付"
      :width="isMobile ? '90%' : '500px'"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      class="payment-qr-dialog"
      :center="true"
      append-to-body
    >
      <div class="payment-qr-container" v-if="upgradeOrder">
        <div class="order-info">
          <el-descriptions :column="1" border size="small" direction="horizontal">
            <el-descriptions-item label="订单号">{{ upgradeOrder.order_no }}</el-descriptions-item>
            <el-descriptions-item label="金额">
              <span class="amount">¥{{ parseFloat(upgradeOrder.actual_payment_amount || upgradeOrder.amount || 0).toFixed(2) }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="内容" v-if="upgradeOrder.additional_devices">
               +{{ upgradeOrder.additional_devices }}个设备
            </el-descriptions-item>
          </el-descriptions>
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
          <p class="tip-text"><el-icon><InfoFilled /></el-icon> 请使用支付宝扫码支付</p>
        </div>
        
        <div class="payment-actions-container">
          <el-button 
            v-if="isMobile && paymentUrl"
            type="success"
            size="large"
            class="payment-btn alipay-btn"
            @click="openAlipayApp"
          >
            <el-icon class="btn-icon"><Wallet /></el-icon>
            跳转支付宝App支付
          </el-button>
          
          <el-button 
            type="primary" 
            size="large"
            class="payment-btn confirm-btn"
            @click="checkUpgradeOrderStatus" 
          >
            我已支付
          </el-button>

          <el-button 
            size="large"
            class="payment-btn cancel-btn"
            @click="paymentQRVisible = false" 
          >
            取消
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Loading, Wallet, DocumentCopy, InfoFilled } from '@element-plus/icons-vue'
import QRCode from 'qrcode'
import { subscriptionAPI, userAPI, orderAPI, userLevelAPI } from '@/utils/api'
import { formatDateTime, formatDate as formatDateUtil, getRemainingDays as getRemainingDaysUtil, isExpired as isExpiredUtil } from '@/utils/date'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
dayjs.extend(timezone)
import '@/styles/list-common.scss'

export default {
  name: 'Subscription',
  components: {
    Loading,
    Wallet,
    DocumentCopy,
    InfoFilled
  },
  setup() {
    const subscription = ref(null)
    const resetLoading = ref(false)
    const sendEmailLoading = ref(false)
    const sendEmailRequesting = ref(false)
    const showUpgradeDialog = ref(false)
    const upgradeLoading = ref(false)
    const userBalance = ref(0)
    const levelDiscountRate = ref(1.0)
    const upgradeForm = ref({
      additionalDevices: 5,
      additionalDays: 0
    })
    
    // 配置项
    const deviceOptions = ref([5, 10, 15, 20, 25, 30, 35, 40, 45, 50])
    const monthOptions = ref([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12])
    
    const upgradeCost = ref(0)
    const levelDiscount = ref(0)
    const finalAmount = ref(0)
    const paymentMethod = ref('alipay')
    const upgradeOrder = ref(null)
    const paymentQRVisible = ref(false)
    const paymentQRCode = ref(null)
    const paymentUrl = ref('')
    const paymentStatusCheckTimer = ref(null)
    const isMobile = ref(window.innerWidth <= 768)

    // 监听窗口大小变化
    const handleResize = () => {
      isMobile.value = window.innerWidth <= 768
    }

    onMounted(() => {
      window.addEventListener('resize', handleResize)
      fetchSubscription()
      fetchUserInfo()
    })

    onUnmounted(() => {
      window.removeEventListener('resize', handleResize)
      if (paymentStatusCheckTimer.value) {
        clearInterval(paymentStatusCheckTimer.value)
        paymentStatusCheckTimer.value = null
      }
    })

    // 获取订阅信息
    const fetchSubscription = async () => {
      try {
        let subscriptionResponse
        try {
          subscriptionResponse = await subscriptionAPI.getUserSubscription()
        } catch (subscriptionError) {
          subscriptionResponse = null
        }
        
        let userResponse
        try {
          userResponse = await userAPI.getUserInfo()
        } catch (userError) {
          userResponse = null
        }
        
        if (subscriptionResponse && subscriptionResponse.data && subscriptionResponse.data.success) {
          const subscriptionData = subscriptionResponse.data.data
          // 计算在线设备数（优先使用 current_devices，如果没有则从设备列表计算）
          let onlineDevices = subscriptionData.current_devices || subscriptionData.currentDevices || 0
          // 如果后端没有返回在线设备数，尝试从设备列表计算
          if (onlineDevices === 0 && subscriptionData.devices && Array.isArray(subscriptionData.devices)) {
            onlineDevices = subscriptionData.devices.filter(d => d.is_active !== false).length
          }
          
          subscription.value = {
            subscription_id: subscriptionData.subscription_id || subscriptionData.subscription_url,
            expire_time: subscriptionData.expire_time || subscriptionData.expiryDate,
            status: subscriptionData.status,
            onlineDevices: onlineDevices,
            current_devices: onlineDevices,
            device_limit: subscriptionData.device_limit || subscriptionData.maxDevices || 0,
            maxDevices: subscriptionData.device_limit || subscriptionData.maxDevices || 0,
            clash_url: subscriptionData.clash_url || subscriptionData.clashUrl || '',
            universal_url: subscriptionData.universal_url || '',
            qrcode_url: subscriptionData.qrcode_url || subscriptionData.qrcodeUrl || ''
          }
          
          if (userResponse && userResponse.data && userResponse.data.success) {
            const userData = userResponse.data.data
            if (userData.clashUrl) subscription.value.clash_url = userData.clashUrl
            if (userData.qrcodeUrl) subscription.value.qrcode_url = userData.qrcodeUrl
          }
        } else if (userResponse && userResponse.data && userResponse.data.success) {
          const userData = userResponse.data.data
          subscription.value = {
            subscription_id: userData.subscription_url,
            expire_time: userData.expire_time,
            status: userData.subscription_status,
            onlineDevices: userData.online_devices || 0,
            current_devices: userData.online_devices || 0,
            device_limit: userData.device_limit || userData.total_devices || 0,
            maxDevices: userData.device_limit || userData.total_devices || 0,
            clash_url: userData.clashUrl || '',
            universal_url: userData.universalUrl || '',
            qrcode_url: userData.qrcodeUrl || ''
          }
        } else {
          ElMessage.error('获取订阅信息失败：无法连接到服务器')
          return
        }
        
        setTimeout(() => {
          generateQRCodes()
        }, 100)
        
      } catch (error) {
        ElMessage.error(`获取订阅信息失败: ${error.message || '未知错误'}`)
      }
    }

    // 生成二维码
    const generateQRCodes = async () => {
      if (!subscription.value) return
      try {
        let qrData = subscription.value.qrcode_url
        if (!qrData && subscription.value.universal_url) {
          const baseUrl = window.location.origin
          const subscriptionUrl = subscription.value.universal_url.startsWith('http') 
            ? subscription.value.universal_url 
            : `${baseUrl}${subscription.value.universal_url}`
          const encodedUrl = btoa(unescape(encodeURIComponent(subscriptionUrl)))
          
          let expiryDisplayName = '订阅'
          if (subscription.value.expire_time && subscription.value.expire_time !== '未设置') {
            try {
              const expireDate = dayjs(subscription.value.expire_time).tz('Asia/Shanghai')
              if (expireDate.isValid()) {
                expiryDisplayName = `到期时间${expireDate.format('YYYY-MM-DD')}`
              }
            } catch (e) {
              expiryDisplayName = subscription.value.expire_time
            }
          }
          qrData = `sub://${encodedUrl}#${encodeURIComponent(expiryDisplayName)}`
        }
        
        const qrElement = document.getElementById('subscription-qrcode')
        if (qrElement && qrData) {
          await QRCode.toCanvas(qrElement, qrData, {
            width: 200,
            margin: 2,
            color: { dark: '#000000', light: '#FFFFFF' },
            errorCorrectionLevel: 'M'
          })
        }
      } catch (error) {
        console.error('生成二维码失败:', error)
      }
    }

    const copyUrl = async (url) => {
      if (!url) {
        ElMessage.warning('没有可复制的内容')
        return
      }
      try {
        await navigator.clipboard.writeText(url)
        ElMessage.success('链接已复制到剪贴板')
      } catch (error) {
        // 降级方案：使用传统的复制方法
        try {
          const textArea = document.createElement('textarea')
          textArea.value = url
          textArea.style.position = 'fixed'
          textArea.style.opacity = '0'
          document.body.appendChild(textArea)
          textArea.select()
          document.execCommand('copy')
          document.body.removeChild(textArea)
          ElMessage.success('链接已复制到剪贴板')
        } catch (fallbackError) {
          ElMessage.error('复制失败，请手动复制')
        }
      }
    }

    const resetSubscription = async () => {
      try {
        await ElMessageBox.confirm(
          '重置订阅地址将清空所有设备记录，确定要继续吗？',
          '确认重置',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        resetLoading.value = true
        const response = await subscriptionAPI.resetSubscription()
        if (response?.data?.success === false) {
          ElMessage.error(response.data.message || '重置失败')
          return
        }
        ElMessage.success('订阅地址已重置')
        await fetchSubscription()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(error.response?.data?.message || '重置失败')
        }
      } finally {
        resetLoading.value = false
      }
    }

    const sendSubscriptionToEmail = async () => {
      if (sendEmailRequesting.value) return
      try {
        sendEmailRequesting.value = true
        sendEmailLoading.value = true
        const response = await subscriptionAPI.sendSubscriptionEmail()
        if (response?.data?.success === false) {
          ElMessage.error(response.data.message || '发送失败')
          return
        }
        ElMessage.success('订阅地址已发送到您的邮箱')
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '发送失败')
      } finally {
        sendEmailLoading.value = false
        setTimeout(() => {
          sendEmailRequesting.value = false
        }, 2000)
      }
    }

    // 辅助函数
    const formatDate = (dateString) => formatDateUtil(dateString, 'YYYY-MM-DD HH:mm') || '未设置'
    
    const getRemainingDays = (subscription) => {
      if (!subscription || !subscription.expire_time) return 0
      return getRemainingDaysUtil(subscription.expire_time)
    }

    const getStatusType = (subscription) => {
      if (!subscription) return 'info'
      if (subscription.expire_time) {
        return isExpiredUtil(subscription.expire_time) ? 'danger' : 'success'
      }
      return subscription.status === 'active' ? 'success' : (subscription.status === 'expired' ? 'danger' : 'info')
    }

    const getStatusText = (subscription) => {
      if (!subscription) return '未激活'
      if (subscription.expire_time) {
        return isExpiredUtil(subscription.expire_time) ? '已过期' : '正常'
      }
      return subscription.status === 'active' ? '正常' : (subscription.status === 'expired' ? '已过期' : '未激活')
    }

    const isSubscriptionActive = (subscription) => {
      if (!subscription) return false
      if (subscription.status) return subscription.status === 'active'
      if (!subscription.expire_time) return false
      return !isExpiredUtil(subscription.expire_time)
    }

    const fetchUserInfo = async () => {
      try {
        const userResponse = await userAPI.getUserInfo()
        if (userResponse?.data?.success) {
          userBalance.value = parseFloat(userResponse.data.data.balance || 0)
        }
        try {
          const levelResponse = await userLevelAPI.getMyLevel()
          if (levelResponse?.data?.success && levelResponse.data.data?.discount_rate) {
            levelDiscountRate.value = parseFloat(levelResponse.data.data.discount_rate || 1.0)
          }
        } catch (e) {}
      } catch (error) {
        console.error('获取用户信息失败:', error)
      }
    }

    // 升级对话框逻辑
    const handleUpgradeDialogOpen = () => {
      upgradeForm.value = { additionalDevices: 5, additionalDays: 0 }
      upgradeCost.value = 0
      levelDiscount.value = 0
      finalAmount.value = 0
      paymentMethod.value = userBalance.value >= 0.01 ? 'balance' : 'alipay'
      fetchUserInfo()
      setTimeout(calculateUpgradeCost, 300)
    }

    const calculateUpgradeCost = async () => {
      if (!subscription.value || !upgradeForm.value.additionalDevices) {
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

    const confirmUpgrade = async () => {
      if (!upgradeForm.value.additionalDevices || upgradeForm.value.additionalDevices < 5) {
        ElMessage.warning('请选择要增加的设备数量（至少5个）')
        return
      }
      try {
        upgradeLoading.value = true
        const upgradeData = {
          additional_devices: upgradeForm.value.additionalDevices,
          additional_days: upgradeForm.value.additionalDays || 0,
          payment_method: paymentMethod.value,
          use_balance: paymentMethod.value === 'balance' || paymentMethod.value === 'mixed',
          balance_amount: paymentMethod.value === 'mixed' ? userBalance.value : (paymentMethod.value === 'balance' ? userBalance.value : null)
        }
        const response = await orderAPI.upgradeDevices(upgradeData)
        if (response?.data?.success) {
          const data = response.data.data
          if (data.status === 'paid') {
            ElMessage.success('设备数量升级成功！')
            showUpgradeDialog.value = false
            await fetchSubscription()
            await fetchUserInfo()
          } else {
            const paymentUrlVal = data.payment_url || data.payment_qr_code
            if (!paymentUrlVal) {
              ElMessage.error('支付链接生成失败，请稍后重试')
              return
            }
            upgradeOrder.value = {
              ...data,
              additional_devices: upgradeForm.value.additionalDevices,
              additional_days: upgradeForm.value.additionalDays || 0
            }
            showUpgradeDialog.value = false
            await showPaymentQRCode(data)
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

    // 支付相关逻辑
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
    
    const showPaymentQRCode = async (order) => {
      const url = order.payment_url || order.payment_qr_code
      paymentUrl.value = url
      try {
        paymentQRCode.value = await QRCode.toDataURL(url, {
          width: 256,
          margin: 2,
          color: { dark: '#000000', light: '#FFFFFF' },
          errorCorrectionLevel: 'M'
        })
        paymentQRVisible.value = true
        startPaymentStatusCheck()
      } catch (error) {
        ElMessage.error('生成二维码失败')
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
          await fetchSubscription()
          await fetchUserInfo()
          // 重置状态
          upgradeForm.value = { additionalDevices: 5, additionalDays: 0 }
          upgradeCost.value = 0
          finalAmount.value = 0
          upgradeOrder.value = null
          paymentQRCode.value = null
        } else if (!isAutoCheck) {
          ElMessage.warning('订单尚未支付，请完成支付')
        }
      } catch (error) {
        if (!isAutoCheck) ElMessage.error('检查订单状态失败')
      }
    }

    const onImageError = () => ElMessage.error('二维码加载失败')
    const onImageLoad = () => {}

    return {
      subscription,
      resetLoading,
      sendEmailLoading,
      showUpgradeDialog,
      upgradeLoading,
      upgradeForm,
      upgradeCost,
      levelDiscount,
      finalAmount,
      userBalance,
      paymentMethod,
      upgradeOrder,
      copyUrl,
      resetSubscription,
      sendSubscriptionToEmail,
      formatDate,
      getRemainingDays,
      getStatusType,
      getStatusText,
      isSubscriptionActive,
      calculateUpgradeCost,
      handlePaymentMethodChange,
      confirmUpgrade,
      handleUpgradeDialogOpen,
      paymentQRVisible,
      paymentQRCode,
      paymentUrl,
      openAlipayApp,
      checkUpgradeOrderStatus,
      onImageError,
      onImageLoad,
      isMobile,
      deviceOptions,
      monthOptions
    }
  }
}
</script>

<style scoped lang="scss">
.subscription-container {
  padding: 0;
  max-width: none;
  margin: 0;
  width: 100%;
}

.subscription-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 15px;
  
  h2 {
    margin: 0;
    color: #333;
    font-size: 1.5rem;
  }
  
  p {
    margin: 0;
    color: #666;
    font-size: 0.9rem;
  }
}

.subscription-status {
  margin-bottom: 30px;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
}

.status-item {
  text-align: left;
  .status-label {
    color: #666;
    font-size: 0.9rem;
    margin-bottom: 8px;
  }
  .status-value {
    color: #333;
    font-size: 1.1rem;
    font-weight: 600;
  }
}

.subscription-urls {
  margin-bottom: 30px;
  
  h3 {
    color: #333;
    margin-bottom: 20px;
    font-size: 1.2rem;
  }
}

.url-list {
  margin-bottom: 30px;
}

.url-item {
  display: flex;
  align-items: center;
  margin-bottom: 15px;
  gap: 15px;
  
  .url-label {
    min-width: 120px;
    color: #666;
    font-weight: 500;
  }
  
  .url-content {
    flex: 1;
  }
}

.qr-code-section {
  text-align: center;
  
  h4 {
    color: #333;
    margin-bottom: 20px;
    font-size: 1.1rem;
  }
  
  .qr-codes {
    display: flex;
    justify-content: center;
    gap: 40px;
  }
  
  .qr-item {
    text-align: center;
    p {
      margin-top: 10px;
      color: #666;
      font-size: 0.9rem;
    }
  }
}

.subscription-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
  margin-bottom: 20px;
  flex-wrap: wrap;
  
  .action-btn {
    padding: 12px 24px;
    font-weight: 600;
    font-size: 0.9375rem;
    border-radius: 8px;
    white-space: nowrap;
    min-width: 120px;
    box-sizing: border-box;
    
    .el-icon {
      margin-right: 6px;
    }
  }
}

.no-subscription {
  text-align: center;
  padding: 40px 20px;
}

/* 支付二维码对话框样式 */
.payment-qr-dialog {
  .payment-qr-container {
    padding: 10px 0;
  }

  .order-info {
    margin-bottom: 20px;
    .amount {
      color: #f56c6c;
      font-weight: 700;
      font-size: 1.1em;
    }
  }

  .qr-code-wrapper {
    text-align: center;
    margin: 20px 0;
    
    .qr-code {
      display: inline-block;
      padding: 15px;
      background: #fff;
      border: 1px solid #e4e7ed;
      border-radius: 8px;
      
      img {
        max-width: 256px;
        width: 100%;
        height: auto;
        display: block;
      }
    }
    
    .qr-loading {
      padding: 40px;
      color: #909399;
      .el-icon {
        font-size: 32px;
        margin-bottom: 12px;
      }
    }
  }

  .payment-tips {
    text-align: center;
    margin-bottom: 20px;
    .tip-text {
      color: #909399;
      font-size: 13px;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 5px;
    }
  }

  /* 支付按钮区域样式 - 优化核心 */
  .payment-actions-container {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 16px;
    margin-top: 24px;
    padding: 0 10px;
    
    .payment-btn {
      min-width: 120px;
      
      .btn-icon {
        margin-right: 5px;
      }
    }
  }
}

/* 升级设备数量对话框样式 */
.upgrade-dialog {
  .upgrade-content {
    padding: 10px 0;
  }

  .current-subscription-info,
  .upgrade-options,
  .cost-calculation,
  .payment-method {
    margin-bottom: 24px;
    h4 {
      color: #333;
      font-size: 1.1rem;
      margin-bottom: 16px;
      font-weight: 600;
    }
  }

  .form-hint {
    color: #909399;
    font-size: 0.875rem;
    margin-top: 8px;
  }

  .final-amount {
    color: #f56c6c;
    font-size: 1.2rem;
    font-weight: 600;
  }

  .balance-info {
    padding: 12px;
    background: #f5f7fa;
    border-radius: 4px;
    margin-bottom: 16px;
    color: #606266;
    font-weight: 500;
  }

  .payment-amount {
    margin-top: 12px;
    padding: 12px;
    background: #f0f9ff;
    border-radius: 4px;
    
    p {
      margin: 8px 0;
      color: #606266;
      &:first-child { color: #67c23a; font-weight: 500; }
      &:last-child { color: #409eff; font-weight: 500; }
    }
  }

  .dialog-footer {
    display: flex;
    justify-content: center;
    gap: 16px;
    padding: 0 10px;
    
    .el-button {
      min-width: 120px;
    }
  }
}

/* 响应式适配 */
@media (max-width: 768px) {
  .subscription-card {
    border-radius: 0;
    margin: 0 -12px;
    box-shadow: none;
    border-left: none;
    border-right: none;
    
    :deep(.el-card__header) {
      padding: 16px;
      .card-header {
        flex-direction: column;
        align-items: flex-start;
        gap: 8px;
      }
    }
  }

  .subscription-actions {
    flex-direction: column;
    gap: 12px;
    
    .action-btn {
      width: 100%;
      margin: 0;
      min-width: auto;
    }
  }

  /* 支付弹窗移动端适配 - 强制对齐修正 */
  .payment-qr-dialog {
    :deep(.el-dialog) {
      margin: 5vh auto !important;
      border-radius: 12px;
      max-width: 95% !important;
    }
    
    :deep(.el-dialog__body) {
      padding: 20px 15px;
    }

    .qr-code-wrapper .qr-code {
      padding: 10px;
      img {
        width: 180px;
        height: 180px;
      }
    }

    /* 移动端支付按钮堆叠显示 */
    .payment-actions-container {
      flex-direction: column; /* 垂直排列 */
      width: 100%;
      padding: 0; /* 移除容器内边距 */
      gap: 12px;
      
      .payment-btn {
        width: 100%; /* 强制占满宽度 */
        height: 46px; /* 统一高度，更易点击 */
        font-size: 16px;
        margin: 0 !important; /* 强制移除 Element Plus 默认的 margin-left */
        border-radius: 8px; /* 统一圆角 */
      }
      
      /* 将最主要的操作（跳转支付）放在最上面 */
      .alipay-btn {
        order: 1;
      }
      .confirm-btn {
        order: 2;
      }
      .cancel-btn {
        order: 3;
      }
    }
  }

  /* 升级弹窗移动端适配 */
  .upgrade-dialog {
    :deep(.el-dialog) {
      width: 95% !important;
      margin: 5vh auto !important;
      max-height: 90vh; /* 限制最大高度 */
      display: flex;
      flex-direction: column;
    }

    :deep(.el-dialog__body) {
      overflow-y: auto; /* 内容区可滚动 */
      flex: 1;
      padding: 10px 15px;
    }

    /* 支付按钮区域 */
    .dialog-footer {
      flex-direction: column;
      gap: 12px;
      padding-top: 10px; /* 增加底部内边距 */
      
      .el-button {
        width: 100%;
        height: 44px;
        margin: 0 !important;
        font-size: 16px;
      }
    }
  }
}
</style>