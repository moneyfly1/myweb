<template>
  <div class="list-container subscription-container">
    <!-- 到期预警横幅 -->
    <el-alert
      v-if="subscription && getRemainingDays(subscription) > 0 && getRemainingDays(subscription) <= 7"
      :title="`订阅将在 ${getRemainingDays(subscription)} 天后到期，请及时续费！`"
      type="warning"
      show-icon
      :closable="false"
      style="margin-bottom: 16px;"
    >
      <template #default>
        <router-link to="/packages">
          <el-button type="warning" size="small" style="margin-top:4px;">立即续费</el-button>
        </router-link>
      </template>
    </el-alert>
    <el-card class="subscription-card" v-loading="loading">
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
                <el-progress
                  :percentage="Math.min(100, Math.round(((subscription.onlineDevices || subscription.current_devices || 0) / (subscription.device_limit || subscription.maxDevices || 1)) * 100))"
                  :color="((subscription.onlineDevices || subscription.current_devices || 0) / (subscription.device_limit || subscription.maxDevices || 1)) >= 0.9 ? '#f56c6c' : ((subscription.onlineDevices || subscription.current_devices || 0) / (subscription.device_limit || subscription.maxDevices || 1)) >= 0.7 ? '#e6a23c' : '#67c23a'"
                  :show-text="false"
                  style="margin-top:4px;"
                />
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
          <router-link to="/packages">
            <el-button type="primary">
              立即订阅
            </el-button>
          </router-link>
        </el-empty>
      </div>
      <!-- 设备满载提示 -->
      <el-alert
        v-if="subscription && isDeviceFull(subscription)"
        title="设备数量已达上限，无法连接新设备"
        type="error"
        show-icon
        :closable="false"
        style="margin-bottom: 16px;"
      >
        <template #default>
          <el-button type="danger" size="small" style="margin-top:4px;" @click="showUpgradeDrawer = true">
            立即升级设备数量
          </el-button>
        </template>
      </el-alert>
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
        <router-link to="/packages">
          <el-button
            type="warning"
            class="action-btn renew-btn"
          >
            续费订阅
          </el-button>
        </router-link>
        <el-button
          type="primary"
          class="action-btn upgrade-btn"
          @click="showUpgradeDrawer = true"
          v-if="isSubscriptionActive(subscription)"
        >
          升级设备数量
        </el-button>
      </div>
      <UpgradeDevicesDrawer
        v-model="showUpgradeDrawer"
        :subscription="subscription"
        :on-success="handleUpgradeSuccess"
      />
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
              <router-link to="/packages">
                <el-button type="primary">
                  立即续费
                </el-button>
              </router-link>
            </div>
          </template>
        </el-alert>
      </div>
    </el-card>
  </div>
</template>
<script>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from '@/utils/elementPlusServices'
import { DocumentCopy } from '@element-plus/icons-vue'
import { subscriptionAPI, userAPI } from '@/utils/api'
import { formatDate as formatDateUtil, getRemainingDays as getRemainingDaysUtil, isExpired as isExpiredUtil } from '@/utils/date'
import { copyToClipboard as copyText } from '@/utils/textSelection'
import UpgradeDevicesDrawer from '@/components/UpgradeDevicesDrawer.vue'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
dayjs.extend(timezone)
export default {
  name: 'Subscription',
  components: {
    DocumentCopy,
    UpgradeDevicesDrawer
  },
  setup() {
    const loading = ref(false)
    const subscription = ref(null)
    const resetLoading = ref(false)
    const sendEmailLoading = ref(false)
    const sendEmailRequesting = ref(false)
    const showUpgradeDrawer = ref(false)
    let refreshPromise = null
    const handleUpgradeSuccess = async () => {
      await refreshSubscription()
    }
    onMounted(() => {
      refreshSubscription()
      const handleSubscriptionUpdate = async () => {
        if (!refreshPromise) {
          await refreshSubscription()
        }
      }
      const handleUserInfoUpdate = async () => {
        if (!refreshPromise) {
          await refreshSubscription()
        }
      }
      window.addEventListener('subscription-updated', handleSubscriptionUpdate)
      window.addEventListener('user-info-updated', handleUserInfoUpdate)
      onUnmounted(() => {
        window.removeEventListener('subscription-updated', handleSubscriptionUpdate)
        window.removeEventListener('user-info-updated', handleUserInfoUpdate)
      })
    })
    const refreshSubscription = async () => {
      if (refreshPromise) return refreshPromise
      refreshPromise = fetchSubscription().finally(() => {
        refreshPromise = null
      })
      return refreshPromise
    }
    const fetchSubscription = async () => {
      loading.value = true
      try {
        // 并发加载订阅和用户信息，提高页面加载速度
        let [subscriptionResponse, userResponse] = await Promise.allSettled([
          subscriptionAPI.getUserSubscription().catch(subscriptionError => {
            console.error('获取订阅信息失败', subscriptionError)
            return null
          }),
          userAPI.getUserInfo().catch(userError => {
            console.error('获取用户信息失败', userError)
            return null
          })
        ]).then(results => [
          results[0].status === 'fulfilled' ? results[0].value : null,
          results[1].status === 'fulfilled' ? results[1].value : null
        ])
        if (subscriptionResponse && subscriptionResponse.data && subscriptionResponse.data.success) {
          const subscriptionData = subscriptionResponse.data.data
          let onlineDevices = subscriptionData.current_devices || subscriptionData.currentDevices || 0
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
        await nextTick()
        setTimeout(() => {
          generateQRCodes()
        }, 200)
      } catch (error) {
        console.error('获取订阅信息失败:', error)
        ElMessage.error(`获取订阅信息失败: ${error.message || '未知错误'}`)
      } finally {
        loading.value = false
      }
    }
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
                expiryDisplayName = `到期时间${expireDate.format('YYYY-MM-DD HH:mm:ss')}`
              }
            } catch (e) {
              expiryDisplayName = subscription.value.expire_time
            }
          }
          qrData = `sub://${encodedUrl}#${encodeURIComponent(expiryDisplayName)}`
        }
        await nextTick()
        const qrElement = document.getElementById('subscription-qrcode')
        if (qrElement && qrData) {
          const QRCode = (await import('qrcode')).default
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
      await copyText(url, '链接已复制到剪贴板')
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
        await refreshSubscription()
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
    const isDeviceFull = (sub) => {
      if (!sub) return false
      const online = sub.onlineDevices || sub.current_devices || 0
      const limit = sub.device_limit || sub.maxDevices || 0
      return limit > 0 && online >= limit
    }
    return {
      subscription,
      resetLoading,
      sendEmailLoading,
      showUpgradeDrawer,
      copyUrl,
      resetSubscription,
      sendSubscriptionToEmail,
      formatDate,
      getRemainingDays,
      getStatusType,
      getStatusText,
      isSubscriptionActive,
      isDeviceFull
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
  :is(p) {
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
  :is(h3) {
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
  :is(h4) {
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
    :is(p) {
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
      :is(img) {
        max-width: 256px;
        width: 100%;
        height: auto;
        display: block;
      }
    }
    .qr-loading {
      padding: 40px;
      color: var(--el-text-color-secondary, #6b7280);
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
      color: var(--el-text-color-secondary, #6b7280);
      font-size: 13px;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 5px;
    }
  }
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
.upgrade-dialog {
  .upgrade-content {
    padding: 10px 0;
  }
  .current-subscription-info,
  .upgrade-options,
  .cost-calculation,
  .payment-method {
    margin-bottom: 24px;
    :is(h4) {
      color: #333;
      font-size: 1.1rem;
      margin-bottom: 16px;
      font-weight: 600;
    }
  }
  .form-hint {
    color: var(--el-text-color-secondary, #6b7280);
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
    :is(p) {
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
@media (max-width: 768px) {
  .subscription-container {
    padding: 10px !important;
    width: 100% !important;
    max-width: 100% !important;
  }
  .subscription-card {
    border-radius: 8px;
    margin: 0;
    box-shadow: 0 2px 12px rgba(0,0,0,0.06);
    width: 100%;
    box-sizing: border-box;
    :deep(.el-card__header) {
      padding: 16px;
      .card-header {
        flex-direction: column;
        align-items: flex-start;
        gap: 8px;
        :is(h2) {
          font-size: 1.25rem;
        }
        :is(p) {
          font-size: 0.875rem;
        }
      }
    }
    :deep(.el-card__body) {
      padding: 16px;
    }
  }
  .subscription-status {
    padding: 16px;
    margin-bottom: 20px;
    .status-item {
      margin-bottom: 16px;
      .status-label {
        font-size: 0.875rem;
        margin-bottom: 6px;
      }
      .status-value {
        font-size: 1rem;
      }
    }
  }
  .subscription-urls {
    :is(h3) {
      font-size: 1.1rem;
      margin-bottom: 16px;
    }
  }
  .url-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
    margin-bottom: 20px;
    .url-label {
      min-width: auto;
      width: 100%;
      font-size: 0.875rem;
    }
    .url-content {
      width: 100%;
      :deep(.el-input) {
        width: 100%;
      }
      :deep(.el-input__wrapper) {
        font-size: 14px;
      }
      :deep(.el-button) {
        padding: 8px 12px;
        font-size: 14px;
      }
    }
  }
  .qr-code-section {
    :is(h4) {
      font-size: 1rem;
      margin-bottom: 16px;
    }
    .qr-codes {
      flex-direction: column;
      gap: 20px;
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
      :is(img) {
        width: 180px;
        height: 180px;
      }
    }
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
  .upgrade-drawer {
  :deep(.el-drawer__body) {
    padding: 20px;
    overflow-y: auto;
  }
  :deep(.el-drawer__footer) {
    padding: 16px 20px;
    border-top: 1px solid #eee;
  }
  .drawer-footer {
    display: flex;
    gap: 12px;
    justify-content: flex-end;
  }
  .upgrade-content {
    h4 {
      font-size: 14px;
      font-weight: 600;
      color: #333;
      margin: 0 0 12px 0;
    }
  }
  .current-subscription-info {
    margin-bottom: 20px;
  }
  .upgrade-options {
    margin-bottom: 20px;
  }
  .form-item-block {
    margin-bottom: 16px;
    .form-label {
      font-size: 13px;
      color: #606266;
      margin-bottom: 8px;
    }
  }
  .device-input-row {
    display: flex;
    align-items: center;
    gap: 8px;
    .device-input-hint {
      font-size: 13px;
      color: #606266;
    }
  }
  .form-hint {
    font-size: 12px;
    color: var(--el-text-color-secondary, #6b7280);
    margin-top: 6px;
  }
  .cost-calculation {
    margin-bottom: 20px;
  }
  .final-amount {
    color: #f56c6c;
    font-weight: 700;
    font-size: 16px;
  }
  .payment-method {
    h4 { margin-bottom: 10px; }
    .balance-info {
      font-size: 13px;
      color: #606266;
      margin-bottom: 10px;
    }
    .el-radio {
      display: block;
      margin-bottom: 8px;
    }
  }
}
}
</style>
