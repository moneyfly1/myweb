<template>
  <div class="list-container subscription-container">
    <div class="breadcrumb">首页 / 订阅管理</div>
    <div class="page-header">
      <div class="page-title">
        <h1>订阅管理</h1>
      </div>
      <div class="actions">
        <el-button
          v-if="subscription && subscription.universal_url"
          type="primary"
          @click="copyUrl(buildSubscriptionUrl(subscription.universal_url))"
        >
          复制默认订阅
        </el-button>
        <el-button
          v-if="subscription && (subscription.subscription_id || subscription.clash_url)"
          @click="sendSubscriptionToEmail"
          :loading="sendEmailLoading"
        >
          发送到邮箱
        </el-button>
        <el-button
          v-if="subscription && (subscription.subscription_id || subscription.clash_url)"
          type="danger"
          plain
          @click="resetSubscription"
          :loading="resetLoading"
        >
          重置订阅
        </el-button>
      </div>
    </div>
    <!-- 到期预警横幅 -->
    <el-alert
      v-if="subscription && getRemainingDays(subscription) > 0 && getRemainingDays(subscription) <= 7"
      :title="`订阅将在 ${getRemainingDays(subscription)} 天后到期，请及时续费！`"
      type="warning"
      show-icon
      :closable="false"
      class="subscription-alert"
    >
      <template #default>
        <router-link to="/packages">
          <el-button type="warning" size="small" class="alert-action-button">立即续费</el-button>
        </router-link>
      </template>
    </el-alert>
    <div class="subscription-page-body" v-loading="loading">
      <template v-if="subscription">
        <el-alert
          v-if="subscription.has_special_nodes"
          type="success"
          show-icon
          :closable="false"
          class="special-user-alert"
        >
          <template #title>
            专线用户
          </template>
          <template #default>
            当前账号已开通专线节点，{{ getSpecialNodeModeText(subscription) }}。
          </template>
        </el-alert>
        <div class="stats-row subscription-stats-row">
          <div class="stat-card">
            <div class="stat-icon">A</div>
            <div>
              <div class="stat-value">
                {{ getStatusText(subscription) }}
              </div>
              <div class="stat-label">账号状态</div>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-icon">E</div>
            <div>
              <div class="stat-value">{{ formatDate(subscription.expire_time) }}</div>
              <div class="stat-label">到期时间</div>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-icon">T</div>
            <div>
              <div class="stat-value">{{ getRemainingDays(subscription) }} 天</div>
              <div class="stat-label">到期天数</div>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-icon">D</div>
            <div>
              <div class="stat-value">
                <el-tooltip content="在线设备数 / 允许最大设备数" placement="top">
                  <span>{{ subscription.onlineDevices || subscription.current_devices || 0 }}/{{ subscription.device_limit || subscription.maxDevices || 0 }}</span>
                </el-tooltip>
                <el-progress
                  :percentage="Math.min(100, Math.round(((subscription.onlineDevices || subscription.current_devices || 0) / (subscription.device_limit || subscription.maxDevices || 1)) * 100))"
                  :color="((subscription.onlineDevices || subscription.current_devices || 0) / (subscription.device_limit || subscription.maxDevices || 1)) >= 0.9 ? '#f56c6c' : ((subscription.onlineDevices || subscription.current_devices || 0) / (subscription.device_limit || subscription.maxDevices || 1)) >= 0.7 ? '#e6a23c' : '#67c23a'"
                  :show-text="false"
                  class="device-progress"
                />
              </div>
              <div class="stat-label">设备使用</div>
            </div>
          </div>
        </div>
        <el-alert
          v-if="subscription && isDeviceFull(subscription) && isSubscriptionActive(subscription)"
          title="设备数量已达上限，无法连接新设备"
          type="error"
          show-icon
          :closable="false"
          class="subscription-alert"
        >
          <template #default>
            <el-button type="danger" size="small" class="alert-action-button" @click="showUpgradeDrawer = true">
              立即升级设备数量
            </el-button>
          </template>
        </el-alert>
        <div class="subscription-main-aside" v-if="subscription && (subscription.subscription_id || subscription.clash_url)">
          <div class="section-stack">
            <div class="card protocol-card" v-if="availableProtocolOptions.length">
              <div class="card-header">
                <div>
                  <h2 class="card-title">协议排除</h2>
                  <div class="card-sub">
                    {{ selectedExcludedProtocols.length ? `已排除 ${selectedExcludedProtocols.length} 种协议` : '默认遵循后台系统协议过滤' }}
                  </div>
                </div>
                <el-button
                  text
                  type="primary"
                  size="small"
                  :disabled="!selectedExcludedProtocols.length"
                  @click="clearExcludedProtocols"
                >
                  清空
                </el-button>
              </div>
              <div class="card-body">
                <el-checkbox-group v-model="selectedExcludedProtocols" class="protocol-checkboxes chip-row">
                  <el-checkbox-button
                    v-for="protocol in availableProtocolOptions"
                    :key="protocol.value"
                    :label="protocol.value"
                  >
                    {{ protocol.label }}
                  </el-checkbox-button>
                </el-checkbox-group>
                <div class="item-meta protocol-meta">
                  当前{{ selectedExcludedProtocols.length ? `已排除 ${selectedExcludedProtocols.length} 种协议` : '未手动排除协议' }}。用户选择不同协议时，下方所有客户端订阅同步更新。
                </div>
              </div>
            </div>
            <div class="card subscription-urls-card">
              <div class="card-header">
                <div>
                  <h2 class="card-title">订阅地址</h2>
                </div>
                <el-tag :type="getStatusType(subscription)" size="small">{{ getStatusText(subscription) }}</el-tag>
              </div>
              <div class="table-wrapper subscription-desktop-list">
                <table class="subscription-table">
                  <thead>
                    <tr>
                      <th>客户端</th>
                      <th>订阅类型</th>
                      <th>地址</th>
                      <th>操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="item in primarySubscriptionRows" :key="item.key">
                      <td>{{ item.client }}</td>
                      <td><el-tag size="small">{{ item.type }}</el-tag></td>
                      <td>
                        <el-input :model-value="buildSubscriptionUrl(item.url)" readonly size="small" />
                      </td>
                      <td>
                        <el-button type="primary" size="small" @click="copyUrl(buildSubscriptionUrl(item.url))">
                          <el-icon><DocumentCopy /></el-icon>
                          复制
                        </el-button>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div class="subscription-mobile-list">
                <div
                  v-for="item in primarySubscriptionRows"
                  :key="item.key"
                  class="subscription-mobile-item"
                >
                  <div class="subscription-mobile-head">
                    <div>
                      <div class="subscription-mobile-title">{{ item.client }}</div>
                      <div class="subscription-mobile-sub">{{ item.type }}</div>
                    </div>
                    <el-button type="primary" size="small" @click="copyUrl(buildSubscriptionUrl(item.url))">
                      <el-icon><DocumentCopy /></el-icon>
                      复制
                    </el-button>
                  </div>
                  <el-input :model-value="buildSubscriptionUrl(item.url)" readonly size="small" />
                </div>
              </div>
            </div>
            <div class="card more-clients-card" v-if="moreClientSubscriptionRows.length">
              <div class="card-header">
                <div>
                  <h2 class="card-title">更多客户端订阅</h2>
                  <div class="card-sub">共 {{ moreClientSubscriptionRows.length }} 个客户端</div>
                </div>
                <el-tag size="small">{{ moreClientSubscriptionRows.length }} 项</el-tag>
              </div>
              <div class="table-wrapper subscription-desktop-list">
                <table class="subscription-table">
                  <thead>
                    <tr>
                      <th>客户端</th>
                      <th>适用系统</th>
                      <th>操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="item in moreClientSubscriptionRows" :key="item.key">
                      <td>{{ item.client }}</td>
                      <td>{{ item.platform }}</td>
                      <td>
                        <el-button type="primary" size="small" @click="copyUrl(buildSubscriptionUrl(item.url))"><el-icon><DocumentCopy /></el-icon>复制</el-button>
                        <el-button v-if="item.qr" size="small" @click="scrollToQrCode">二维码</el-button>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <el-collapse v-model="activeMoreClientPanels" class="subscription-mobile-collapse">
                <el-collapse-item name="more-clients">
                  <template #title>
                    <span>展开更多客户端</span>
                  </template>
                  <div class="subscription-mobile-list">
                    <div
                      v-for="item in moreClientSubscriptionRows"
                      :key="item.key"
                      class="subscription-mobile-item"
                    >
                      <div class="subscription-mobile-head">
                        <div>
                          <div class="subscription-mobile-title">{{ item.client }}</div>
                          <div class="subscription-mobile-sub">{{ item.platform }}</div>
                        </div>
                        <div class="subscription-mobile-actions">
                          <el-button type="primary" size="small" @click="copyUrl(buildSubscriptionUrl(item.url))">
                            <el-icon><DocumentCopy /></el-icon>
                            复制
                          </el-button>
                          <el-button v-if="item.qr" size="small" @click="scrollToQrCode">二维码</el-button>
                        </div>
                      </div>
                    </div>
                  </div>
                </el-collapse-item>
              </el-collapse>
            </div>
          </div>
          <div class="section-stack">
            <div class="card qr-card">
              <div class="card-header">
                <h2 class="card-title">订阅二维码</h2>
              </div>
              <div class="card-body">
                <div class="qr-codes">
                  <div class="qr-item">
                    <canvas ref="subscriptionQrCanvas"></canvas>
                    <p v-if="subscription.expire_time && subscription.expire_time !== '未设置'">
                      到期时间：{{ formatDate(subscription.expire_time) }}
                    </p>
                    <p v-else>通用订阅</p>
                  </div>
                </div>
                <div class="subscription-side-actions">
                  <el-button v-if="subscription.universal_url" @click="copyUrl(buildSubscriptionUrl(subscription.universal_url))">
                    复制链接
                  </el-button>
                  <el-button v-if="subscriptionQrReady" @click="downloadSubscriptionQr">
                    下载二维码
                  </el-button>
                </div>
              </div>
            </div>
            <div class="card actions-card">
              <div class="card-header">
                <h2 class="card-title">订阅操作</h2>
              </div>
              <div class="card-body">
                <div class="subscription-actions subscription-operation-actions">
                  <el-button
                    type="danger"
                    class="action-btn reset-btn"
                    @click="resetSubscription"
                    :loading="resetLoading"
                  >
                    重置订阅地址
                  </el-button>
                  <el-button
                    class="action-btn email-btn"
                    @click="sendSubscriptionToEmail"
                    :loading="sendEmailLoading"
                  >
                    发送到邮箱
                  </el-button>
                  <router-link to="/packages">
                    <el-button
                      type="success"
                      class="action-btn renew-btn"
                    >
                      立即续费
                    </el-button>
                  </router-link>
                  <el-button
                    type="warning"
                    class="action-btn upgrade-btn"
                    @click="showUpgradeDrawer = true"
                    v-if="isSubscriptionActive(subscription)"
                  >
                    升级设备数量
                  </el-button>
                </div>
                <el-alert
                  v-if="subscription && !isSubscriptionActive(subscription)"
                  title="订阅已过期"
                  type="warning"
                  :description="`您的订阅已于 ${formatDate(subscription.expire_time)} 过期，请及时续费以继续使用服务。`"
                  show-icon
                  :closable="false"
                />
              </div>
            </div>
          </div>
        </div>
        <div class="no-subscription card" v-else>
          <div class="card-body">
            <EmptyState
              title="您还没有可用订阅"
              description="购买套餐后会生成订阅地址、二维码和各客户端订阅入口。"
            />
            <div class="subscription-actions empty-subscription-actions">
              <router-link to="/packages">
                <el-button type="primary">购买套餐</el-button>
              </router-link>
              <router-link to="/orders">
                <el-button>查看订单</el-button>
              </router-link>
            </div>
          </div>
        </div>
      </template>
      <template v-else>
        <div class="stats-row subscription-stats-row">
          <div class="stat-card"><div class="stat-icon">A</div><div><div class="stat-value">未激活</div><div class="stat-label">账号状态</div></div></div>
          <div class="stat-card"><div class="stat-icon">E</div><div><div class="stat-value">未设置</div><div class="stat-label">到期时间</div></div></div>
          <div class="stat-card"><div class="stat-icon">T</div><div><div class="stat-value">0 天</div><div class="stat-label">到期天数</div></div></div>
          <div class="stat-card"><div class="stat-icon">D</div><div><div class="stat-value">0/0</div><div class="stat-label">设备使用</div></div></div>
        </div>
        <div class="no-subscription card">
          <div class="card-body">
            <EmptyState
              title="您还没有订阅"
              description="选择套餐后即可生成订阅地址和二维码。"
              :icon-size="64"
            >
              <template #action>
                <router-link to="/packages">
                  <el-button type="primary">
                    立即订阅
                  </el-button>
                </router-link>
              </template>
            </EmptyState>
          </div>
        </div>
      </template>
      <UpgradeDevicesDrawer
        v-model="showUpgradeDrawer"
        :subscription="subscription"
        :on-success="handleUpgradeSuccess"
      />
    </div>
  </div>
</template>
<script>
import { ref, onMounted, onUnmounted, nextTick, watch, computed } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { DocumentCopy } from '@element-plus/icons-vue'
import { subscriptionAPI, userAPI } from '@/utils/api'
import { formatDate as formatDateUtil, getRemainingDays as getRemainingDaysUtil, isExpired as isExpiredUtil } from '@/utils/date'
import { confirmReset } from '@/utils/confirmAction'
import { copyToClipboard as copyText } from '@/utils/textSelection'
import EmptyState from '@/components/EmptyState.vue'
import UpgradeDevicesDrawer from '@/components/UpgradeDevicesDrawer.vue'
import { drawQRCodeToCanvas } from '@/utils/qrcode'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
dayjs.extend(timezone)
export default {
  name: 'Subscription',
  components: {
    DocumentCopy,
    EmptyState,
    UpgradeDevicesDrawer
  },
  setup() {
    const loading = ref(false)
    const subscription = ref(null)
    const resetLoading = ref(false)
    const sendEmailLoading = ref(false)
    const sendEmailRequesting = ref(false)
    const showUpgradeDrawer = ref(false)
    const selectedExcludedProtocols = ref([])
    const activeMoreClientPanels = ref([])
    const subscriptionQrCanvas = ref(null)
    const subscriptionQrReady = ref(false)
    const availableProtocolOptions = [
      { label: 'AnyTLS', value: 'anytls' },
      { label: 'VMess', value: 'vmess' },
      { label: 'VLESS', value: 'vless' },
      { label: 'Trojan', value: 'trojan' },
      { label: 'Shadowsocks', value: 'ss' },
      { label: 'Hysteria2', value: 'hysteria2' },
      { label: 'TUIC', value: 'tuic' },
      { label: 'SOCKS', value: 'socks' },
      { label: 'HTTP', value: 'http' }
    ]
    const primarySubscriptionRows = computed(() => [
      {
        key: 'universal',
        client: '通用订阅',
        type: 'V2Ray / Shadowrocket',
        url: subscription.value?.universal_url
      },
      {
        key: 'clash',
        client: 'Clash / Clash Meta',
        type: '规则订阅',
        url: subscription.value?.clash_url
      }
    ].filter(item => item.url))
    const moreClientSubscriptionRows = computed(() => [
      {
        key: 'stash',
        client: 'Stash',
        platform: 'iOS / macOS',
        url: subscription.value?.stash_url
      },
      {
        key: 'surge',
        client: 'Surge',
        platform: 'iOS / macOS',
        url: subscription.value?.surge_url
      },
      {
        key: 'quantumultx',
        client: 'Quantumult X',
        platform: 'iOS',
        url: subscription.value?.quantumultx_url
      },
      {
        key: 'loon',
        client: 'Loon',
        platform: 'iOS',
        url: subscription.value?.loon_url
      },
      {
        key: 'singbox',
        client: 'Sing-Box',
        platform: '全平台',
        url: subscription.value?.singbox_url
      },
      {
        key: 'shadowrocket',
        client: 'Shadowrocket',
        platform: 'iOS',
        url: subscription.value?.shadowrocket_url,
        qr: true
      }
    ].filter(item => item.url))
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
    watch(selectedExcludedProtocols, () => {
      generateQRCodes()
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
            stash_url: subscriptionData.stash_url || '',
            surge_url: subscriptionData.surge_url || '',
            quantumultx_url: subscriptionData.quantumultx_url || '',
            loon_url: subscriptionData.loon_url || '',
            singbox_url: subscriptionData.singbox_url || '',
            shadowrocket_url: subscriptionData.shadowrocket_url || '',
            qrcode_url: subscriptionData.qrcode_url || subscriptionData.qrcodeUrl || ''
          }
          applySpecialNodeInfo(subscription.value, subscriptionData)
          if (userResponse && userResponse.data && userResponse.data.success) {
            const userData = userResponse.data.data
            if (userData.clashUrl) subscription.value.clash_url = userData.clashUrl
            if (userData.qrcodeUrl) subscription.value.qrcode_url = userData.qrcodeUrl
            applySpecialNodeInfo(subscription.value, userData)
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
            stash_url: userData.stashUrl || userData.stash_url || '',
            surge_url: userData.surgeUrl || userData.surge_url || '',
            quantumultx_url: userData.quantumultxUrl || userData.quantumultx_url || '',
            loon_url: userData.loonUrl || userData.loon_url || '',
            singbox_url: userData.singboxUrl || userData.singbox_url || '',
            shadowrocket_url: userData.shadowrocketUrl || userData.shadowrocket_url || '',
            qrcode_url: userData.qrcodeUrl || ''
          }
          applySpecialNodeInfo(subscription.value, userData)
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
        subscriptionQrReady.value = false
        let qrData = selectedExcludedProtocols.value.length ? '' : subscription.value.qrcode_url
        if (!qrData && subscription.value.universal_url) {
          const baseUrl = window.location.origin
          const excludedUrl = buildSubscriptionUrl(subscription.value.universal_url)
          const subscriptionUrl = excludedUrl.startsWith('http')
            ? excludedUrl
            : `${baseUrl}${excludedUrl}`
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
        const qrElement = subscriptionQrCanvas.value
        if (qrElement && qrData) {
          await drawQRCodeToCanvas(qrElement, qrData, {
            width: 200,
            margin: 2,
            color: { dark: '#000000', light: '#FFFFFF' },
            errorCorrectionLevel: 'M'
          })
          subscriptionQrReady.value = true
        }
      } catch (error) {
        console.error('生成二维码失败:', error)
      }
    }
    const copyUrl = async (url) => {
      const message = selectedExcludedProtocols.value.length
        ? `链接已复制，已排除 ${selectedExcludedProtocols.value.join(', ')}`
        : '链接已复制到剪贴板'
      await copyText(url, message)
    }
    const buildSubscriptionUrl = (url) => {
      if (!url) return ''
      if (!selectedExcludedProtocols.value.length) return url
      const separator = url.includes('?') ? '&' : '?'
      return `${url}${separator}exclude=${selectedExcludedProtocols.value.join(',')}`
    }
    const clearExcludedProtocols = () => {
      selectedExcludedProtocols.value = []
    }
    const scrollToQrCode = async () => {
      await nextTick()
      subscriptionQrCanvas.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
    const downloadSubscriptionQr = () => {
      const canvas = subscriptionQrCanvas.value
      if (!canvas) return
      const link = document.createElement('a')
      link.href = canvas.toDataURL('image/png')
      link.download = 'subscription-qrcode.png'
      link.click()
    }
    const resetSubscription = async () => {
      try {
        await confirmReset('订阅地址', {
          message: '重置订阅地址后，旧订阅链接会立即失效，已连接设备需要重新复制或扫码订阅。确认继续重置吗？',
          confirmButtonText: '确认重置'
        })
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
      if (subscription.expire_time && isExpiredUtil(subscription.expire_time)) return false
      if (subscription.status) return subscription.status === 'active'
      return false
    }
    const isDeviceFull = (sub) => {
      if (!sub) return false
      const online = sub.onlineDevices || sub.current_devices || 0
      const limit = sub.device_limit || sub.maxDevices || 0
      return limit > 0 && online >= limit
    }
    const applySpecialNodeInfo = (target, source = {}) => {
      if (!target || !source) return
      target.has_special_nodes = !!(source.has_special_nodes || source.subscription?.has_special_nodes)
      target.special_node_count = source.special_node_count || source.subscription?.special_node_count || 0
      target.special_node_subscription_type = source.special_node_subscription_type || source.subscription?.special_node_subscription_type || 'both'
      target.special_node_unlimited_devices = !!(source.special_node_unlimited_devices || source.subscription?.special_node_unlimited_devices)
    }
    const getSpecialNodeModeText = (sub) => {
      const lineMode = sub.special_node_subscription_type === 'special_only' ? '仅显示专线线路' : '显示专线和普通线路'
      const deviceMode = sub.special_node_unlimited_devices ? '设备不限制' : '设备跟随系统限制'
      return `${lineMode}，${deviceMode}`
    }
    return {
      subscription,
      resetLoading,
      sendEmailLoading,
      showUpgradeDrawer,
      selectedExcludedProtocols,
      activeMoreClientPanels,
      subscriptionQrCanvas,
      subscriptionQrReady,
      availableProtocolOptions,
      primarySubscriptionRows,
      moreClientSubscriptionRows,
      copyUrl,
      buildSubscriptionUrl,
      clearExcludedProtocols,
      scrollToQrCode,
      downloadSubscriptionQr,
      resetSubscription,
      sendSubscriptionToEmail,
      formatDate,
      getRemainingDays,
      getStatusType,
      getStatusText,
      isSubscriptionActive,
      isDeviceFull,
      getSpecialNodeModeText
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
.subscription-page-body {
  min-height: 260px;
}
.subscription-container > .page-header .actions {
  min-width: 0;
}
.subscription-stats-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 14px;
}
.subscription-main-aside {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(320px, 0.85fr);
  align-items: start;
  gap: 14px;
}
.section-stack {
  display: grid;
  gap: 14px;
}
.card {
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  overflow: hidden;
}
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid #ebeef5;
}
.card-title {
  margin: 0;
  color: #303133;
  font-size: 16px;
  font-weight: 700;
}
.card-sub {
  margin-top: 4px;
  color: #909399;
  font-size: 13px;
}
.card-body {
  padding: 16px;
}
.subscription-table {
  width: 100%;
  border-collapse: collapse;
  background: #fff;
  font-size: 14px;
}
.subscription-table th,
.subscription-table td {
  padding: 12px;
  border-bottom: 1px solid #ebeef5;
  text-align: left;
  vertical-align: middle;
}
.subscription-table th {
  background: #f5f7fa;
  color: #606266;
  font-weight: 700;
}
.subscription-table td:nth-child(3) {
  min-width: 260px;
}
.more-clients-card .subscription-table td:nth-child(3) {
  min-width: 0;
}
.more-clients-card .subscription-table td:last-child {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.subscription-mobile-list,
.subscription-mobile-collapse {
  display: none;
}
.subscription-mobile-item {
  min-width: 0;
  padding: 12px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  background: #fff;
}
.subscription-mobile-item + .subscription-mobile-item {
  margin-top: 10px;
}
.subscription-mobile-head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  align-items: start;
  min-width: 0;
}
.subscription-mobile-title {
  color: #303133;
  font-size: 14px;
  font-weight: 700;
  line-height: 1.35;
}
.subscription-mobile-sub {
  margin-top: 3px;
  color: #909399;
  font-size: 12px;
  line-height: 1.4;
}
.subscription-mobile-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}
.subscription-mobile-item :deep(.el-input) {
  margin-top: 10px;
  width: 100%;
}
.subscription-mobile-item :deep(.el-input__inner) {
  font-size: 12px;
}
.protocol-meta {
  margin-top: 12px;
  color: #909399;
  font-size: 13px;
  line-height: 1.45;
}
.chip {
  display: inline-flex;
  align-items: center;
  min-height: 30px;
  padding: 0 12px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  background: #fff;
  color: #606266;
  font-size: 13px;
  font-weight: 500;
}
.chip.active {
  border-color: #409eff;
  background: #ecf5ff;
  color: #409eff;
}
.masked-url {
  min-height: 32px;
  display: flex;
  align-items: center;
  padding: 0 11px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  background: #fff;
  color: #909399;
  letter-spacing: 1px;
}
.empty-subscription-notice {
  margin-top: 14px;
}
.subscription-side-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
  margin-top: 14px;
}
.qr-card .card-body {
  text-align: center;
}
.qr-codes {
  display: flex;
  justify-content: center;
  width: 100%;
}
.qr-item {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  max-width: 100%;
  text-align: center;
}
.qr-item canvas {
  display: block;
  margin: 0 auto;
}
.qr-item p {
  width: 100%;
  margin: 10px 0 0;
  color: #606266;
  font-size: 13px;
  line-height: 1.5;
  text-align: center;
}
.subscription-alert {
  margin-bottom: 16px;
}
.alert-action-button,
.device-progress {
  margin-top: 4px;
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
.special-user-alert {
  margin-bottom: 16px;
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
  .more-clients-collapse {
    margin-top: 16px;
    border: 1px solid #e4e7ed;
    border-radius: 8px;
    background: #fafafa;
    :deep(.el-collapse-item__header) {
      font-size: 14px;
      color: #409eff;
      background: transparent;
      border: none;
      padding: 12px 16px;
      font-weight: 500;
    }
    :deep(.el-collapse-item__wrap) {
      background: transparent;
      border: none;
    }
    :deep(.el-collapse-item__content) {
      padding: 0 16px 16px;
    }
  }
}
.protocol-exclude-panel {
  margin-bottom: 18px;
  padding: 14px 16px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  background: #f8fafc;

  .protocol-exclude-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 12px;
  }

  .exclude-title {
    color: #303133;
    font-size: 14px;
    font-weight: 600;
    line-height: 1.4;
  }

  .exclude-subtitle {
    margin-top: 2px;
    color: #909399;
    font-size: 12px;
    line-height: 1.4;
  }

  .protocol-checkboxes {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;

    :deep(.el-checkbox-button__inner) {
      border: 1px solid #dcdfe6;
      border-radius: 6px;
      padding: 7px 12px;
      line-height: 1;
    }

    :deep(.el-checkbox-button:first-child .el-checkbox-button__inner),
    :deep(.el-checkbox-button:last-child .el-checkbox-button__inner) {
      border-radius: 6px;
    }
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
.subscription-operation-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  align-items: stretch;
  .el-button,
  :deep(.el-button) {
    width: 100%;
    margin: 0;
  }
  :deep(a),
  a {
    display: block;
    width: 100%;
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
  .subscription-container > .page-header .actions {
    display: grid !important;
    grid-template-columns: repeat(auto-fit, minmax(96px, 1fr)) !important;
    gap: 8px !important;
    width: 100% !important;
    min-width: 0 !important;
  }
  .subscription-container > .page-header .actions .el-button {
    width: 100% !important;
    min-width: 0 !important;
    min-height: 44px;
    margin-left: 0 !important;
    padding: 6px 4px;
    font-size: 12px;
    line-height: 1.25;
    white-space: normal;
  }
  .subscription-container > .page-header .actions .el-button :deep(span) {
    min-width: 0;
    max-width: 100%;
    line-height: 1.25;
    white-space: normal;
  }
  .subscription-card {
    border-radius: 8px;
    margin: 0;
    border: 1px solid var(--el-border-color-lighter);
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
  .subscription-desktop-list {
    display: none !important;
  }
  .subscription-mobile-list {
    display: block;
  }
  .subscription-mobile-collapse {
    display: block;
    border-top: 0;

    :deep(.el-collapse-item__header) {
      height: auto;
      min-height: 44px;
      padding: 0 12px;
      color: #303133;
      font-weight: 600;
      line-height: 1.4;
    }

    :deep(.el-collapse-item__wrap) {
      border-bottom: 0;
    }

    :deep(.el-collapse-item__content) {
      padding: 12px;
    }
  }
  .more-clients-card > .subscription-mobile-collapse {
    margin-top: 0;
  }
  .subscription-mobile-head {
    grid-template-columns: 1fr;
  }
  .subscription-mobile-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    justify-content: stretch;

    .el-button {
      width: 100%;
      margin-left: 0;
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
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
    .action-btn {
      width: 100%;
      margin: 0;
      min-width: auto;
    }
  }
  .subscription-operation-actions {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    width: 100%;
    .action-btn {
      min-height: 40px;
      padding: 10px 8px;
      font-size: 0.875rem;
    }
    :deep(a),
    a {
      min-width: 0;
    }
  }
  .payment-qr-dialog {
    :deep(.el-dialog) {
      width: 92% !important;
      margin: 4vh auto !important;
      border-radius: 8px;
      max-width: 420px !important;
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
        width: 100%;
        min-height: 46px;
        font-size: 16px;
        margin: 0 !important;
        border-radius: 8px;
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
