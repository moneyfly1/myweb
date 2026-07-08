<template>
  <div class="list-container dashboard-container">
    <div class="breadcrumb">首页 / 仪表盘</div>
    <div class="page-header">
      <div class="page-title">
        <h1>仪表盘</h1>
      </div>
      <div class="actions">
        <el-button
          :type="checkedIn ? 'info' : 'warning'"
          :loading="checkinLoading"
          :disabled="checkedIn"
          @click="handleCheckin"
        >
          {{ checkedIn ? '已签到' : '签到' }}
        </el-button>
        <el-button type="primary" @click="showRechargeDialog">
          充值
        </el-button>
        <el-button type="success" @click="goToPackages">
          立即续费
        </el-button>
      </div>
    </div>
    <el-alert
      v-if="userInfo.has_special_nodes"
      type="success"
      show-icon
      :closable="false"
      class="special-user-alert"
    >
      <template #title>专线用户</template>
      <template #default>
        当前账号已开通专线节点，{{ specialNodeModeText }}。
      </template>
    </el-alert>
    <!-- 到期预警横幅 -->
    <el-alert
      v-if="dashboardRemainingDays > 0 && dashboardRemainingDays <= 7"
      :title="`您的订阅将在 ${dashboardRemainingDays} 天后到期，请及时续费！`"
      type="warning"
      show-icon
      :closable="false"
      class="expiry-alert"
    >
      <template #default>
        <router-link to="/packages">
          <el-button type="warning" size="small" class="expiry-renew-btn">立即续费</el-button>
        </router-link>
      </template>
    </el-alert>
    <LoadingState v-if="dashboardLoading" text="正在加载仪表盘..." />
    <div v-else class="stats-grid">
      <div class="stat-card balance-card">
        <div class="stat-icon">
          <el-icon><Wallet /></el-icon>
        </div>
        <div class="stat-content">
          <div class="balance-main">
            <div class="stat-value">{{ typeof userInfo.balance === 'string' ? userInfo.balance : (userInfo.balance || 0).toFixed(2) }}</div>
            <div class="stat-label">账户余额</div>
          </div>
        </div>
      </div>
      <div class="stat-card level-card" :style="levelThemeStyle">
        <div class="stat-icon level-icon">
          <el-icon><Medal /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value level-name">
            {{ userInfo.user_level?.name || userInfo.membership || '普通会员' }}
          </div>
          <div class="stat-label">当前等级</div>
          <el-tag
            v-if="userInfo.user_level && userInfo.user_level.discount_rate < 1.0"
            class="level-discount-tag"
          >
            {{ (userInfo.user_level.discount_rate * 10).toFixed(1) }}折
          </el-tag>
        </div>
      </div>
      <div class="stat-card remaining-time-card">
        <div class="stat-icon">
          <el-icon><Clock /></el-icon>
        </div>
        <div class="stat-content">
          <div class="remaining-time-main">
            <div class="stat-value remaining-time-value">
              <span class="time-number">{{ dashboardRemainingDays }}</span>
              <span class="time-unit">天</span>
            </div>
            <p class="stat-label">剩余天数</p>
          </div>
        </div>
      </div>
      <div
        class="stat-card device-card"
        :class="{
          'device-overlimit': isDeviceOverlimit,
          'device-warning': isDeviceWarning
        }"
      >
        <div class="stat-icon">
          <el-icon><Cellphone /></el-icon>
        </div>
        <div class="stat-content">
          <div class="device-count-wrapper">
            <span
              class="device-count"
              :class="{
                'device-overlimit-count': isDeviceOverlimit,
                'device-warning-count': isDeviceWarning
              }"
            >
              {{ userInfo.online_devices || subscriptionInfo.currentDevices || 0 }}
            </span>
            <span class="device-separator">/</span>
            <span class="device-limit">
              {{ userInfo.total_devices || subscriptionInfo.maxDevices || 0 }}
            </span>
          </div>
          <p class="stat-label">当前设备 / 可用设备</p>
          <div v-if="isDeviceOverlimit" class="device-alert">
            <el-icon class="inline-icon"><WarningFilled /></el-icon>
            <span>设备数量超过限制！</span>
          </div>
        </div>
      </div>
    </div>
    <div class="dashboard-main-aside grid main-aside">
      <div class="left-content section-stack">
        <div class="card subscription-card dashboard-section-card">
          <div class="card-header">
            <div>
              <h3 class="card-title">
                <el-icon class="title-icon"><Link /></el-icon>
                订阅快捷操作
              </h3>
            </div>
            <router-link to="/subscription">
              <el-button size="small">进入订阅管理</el-button>
            </router-link>
          </div>
          <div class="card-body">
            <div class="software-category">
              <h4 class="category-title">
                <el-icon class="title-icon"><Lightning /></el-icon>
                Clash系列软件
              </h4>
              <div class="subscription-buttons">
                <div class="subscription-group">
                  <el-dropdown @command="handleClashCommand" trigger="click">
                    <el-button type="primary" class="clash-btn">
                      <el-icon><Lightning /></el-icon>
                      Clash
                      <el-icon><ArrowDown /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="copy-clash">复制订阅</el-dropdown-item>
                        <el-dropdown-item command="import-clash">一键导入</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
                <div class="subscription-group">
                  <el-dropdown @command="handleFlashCommand" trigger="click">
                    <el-button type="primary" class="flash-btn">
                      <el-icon><Lightning /></el-icon>
                      Flash
                      <el-icon><ArrowDown /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="copy-flash">复制订阅</el-dropdown-item>
                        <el-dropdown-item command="import-flash">一键导入</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
                <div class="subscription-group">
                  <el-dropdown @command="handleClashPartyCommand" trigger="click">
                    <el-button type="primary" class="clash-party-btn">
                      <el-icon><Box /></el-icon>
                      Clash Part
                      <el-icon><ArrowDown /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="copy-clash-party">复制订阅</el-dropdown-item>
                        <el-dropdown-item command="import-clash-party">一键导入</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
                <div class="subscription-group">
                  <el-dropdown @command="handleClashVergeCommand" trigger="click">
                    <el-button type="primary" class="clash-verge-btn">
                      <el-icon><Lightning /></el-icon>
                      Clash Verge
                      <el-icon><ArrowDown /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="copy-clash-verge">复制订阅</el-dropdown-item>
                        <el-dropdown-item command="import-clash-verge">一键导入</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </div>
            </div>
            <div class="software-category">
              <h4 class="category-title">
                <el-icon class="title-icon"><Aim /></el-icon>
                V2Ray系列软件
              </h4>
              <div class="subscription-buttons">
                <div class="subscription-group">
                  <el-button type="info" class="universal-btn" @click="copyUniversalSubscription">
                    <el-icon><Aim /></el-icon>
                    复制通用订阅
                  </el-button>
                </div>
                <div class="subscription-group">
                  <el-button type="info" class="hiddify-btn" @click="copyHiddifySubscription">
                    <el-icon><View /></el-icon>
                    复制 Hiddify Next 订阅
                  </el-button>
                </div>
                <div class="subscription-group">
                  <el-button class="qr-toggle-btn" @click="showQRCode = !showQRCode">
                    <el-icon><Picture /></el-icon>
                    {{ showQRCode ? '收起扫码二维码' : '显示扫码二维码' }}
                  </el-button>
                </div>
              </div>
            </div>
            <div class="software-category">
              <h4 class="category-title">
                <el-icon class="title-icon"><Iphone /></el-icon>
                iOS软件
              </h4>
              <div class="subscription-buttons">
                <div class="subscription-group">
                  <el-dropdown @command="handleShadowrocketCommand" trigger="click">
                    <el-button type="success" class="shadowrocket-btn">
                      <el-icon><Iphone /></el-icon>
                      Shadowrocket
                      <el-icon><ArrowDown /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="copy-shadowrocket">复制订阅</el-dropdown-item>
                        <el-dropdown-item command="import-shadowrocket">一键导入</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </div>
            </div>
            <div v-if="showQRCode" class="qr-code-section">
              <h4 class="section-title">
                <el-icon class="title-icon"><Picture /></el-icon>
                二维码
              </h4>
              <div class="qr-code-container">
                <div class="qr-code">
                  <img :src="qrCodeUrl" alt="订阅二维码" v-if="qrCodeUrl">
                  <div v-else class="qr-placeholder">
                    <el-icon><Picture /></el-icon>
                    <p>二维码生成中...</p>
                  </div>
                </div>
                <p class="qr-tip">扫描二维码即可在Shadowrocket中添加订阅</p>
              </div>
            </div>
          </div>
        </div>
        <div class="card recent-order-card dashboard-section-card">
          <div class="card-header">
            <div>
              <h3 class="card-title">
                <el-icon class="title-icon"><Document /></el-icon>
                最近订单
              </h3>
            </div>
            <router-link to="/orders">
              <el-button size="small">订单记录</el-button>
            </router-link>
          </div>
            <div class="table-wrapper compact-order-table">
            <table class="dashboard-table">
              <thead>
                <tr>
                  <th>订单内容</th>
                  <th>金额</th>
                  <th>状态</th>
                  <th>时间</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>套餐购买与续费</td>
                  <td>¥29.00</td>
                  <td><el-tag size="small" type="success">已支付</el-tag></td>
                  <td>最近</td>
                  <td>
                    <router-link to="/orders">
                      <el-button size="small">详情</el-button>
                    </router-link>
                  </td>
                </tr>
                <tr>
                  <td>账户余额充值</td>
                  <td>¥100.00</td>
                  <td><el-tag size="small" type="warning">待支付</el-tag></td>
                  <td>待处理</td>
                  <td>
                    <el-button size="small" type="primary" @click="showRechargeDialog">继续支付</el-button>
                  </td>
                </tr>
                <tr>
                  <td>升级设备数量</td>
                  <td>按新增数量计算</td>
                  <td><el-tag size="small" type="warning">待支付</el-tag></td>
                  <td>待处理</td>
                  <td>
                    <el-button size="small" type="primary" @click="showUpgradeDrawer = true">继续支付</el-button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
      <div class="right-content section-stack">
        <div class="card device-management-card dashboard-section-card">
          <div class="card-header">
            <div>
              <h3 class="card-title">
                <el-icon class="title-icon"><Cellphone /></el-icon>
                设备管理
              </h3>
            </div>
          </div>
          <div class="card-body">
            <div
              class="device-summary"
              :class="{
                'device-summary-danger': isDeviceOverlimit,
                'device-summary-warning': isDeviceWarning
              }"
            >
              <div class="device-summary-icon">
                <el-icon><Cellphone /></el-icon>
              </div>
              <div>
                <div class="device-summary-value">
                  {{ userInfo.online_devices || subscriptionInfo.currentDevices || 0 }}
                  <span>/ {{ userInfo.total_devices || subscriptionInfo.maxDevices || 0 }}</span>
                </div>
                <div class="device-summary-label">在线设备 / 可用设备</div>
              </div>
            </div>
            <div v-if="isDeviceOverlimit" class="notice danger device-notice">
              设备数量超过限制，请管理设备或升级设备数量。
            </div>
            <div v-else class="notice success device-notice">
              当前设备数量正常，可以进入设备管理查看在线设备。
            </div>
            <div class="button-row device-actions-row">
              <router-link to="/devices">
                <el-button type="primary">
                  <el-icon><Cellphone /></el-icon>
                  管理设备
                </el-button>
              </router-link>
              <el-button
                type="success"
                @click="showUpgradeDrawer = true"
                :disabled="!(userInfo.total_devices || subscriptionInfo.maxDevices)"
              >
                <el-icon><Top /></el-icon>
                升级设备数量
              </el-button>
            </div>
          </div>
        </div>
        <div class="card tutorial-card dashboard-section-card">
          <div class="card-header">
            <div>
              <h3 class="card-title">
                <el-icon class="title-icon"><Reading /></el-icon>
                软件下载与教程
              </h3>
            </div>
            <router-link to="/help">
              <el-button size="small">全部教程</el-button>
            </router-link>
          </div>
          <div class="card-body">
            <div class="dashboard-client-list">
              <div
                v-for="platform in dashboardClientGroups"
                :key="platform.name"
                class="ticket-item dashboard-client-row"
              >
                <div>
                  <div class="item-title">{{ platform.name }}：{{ platform.clientNames }}</div>
                  <div class="item-meta">立即下载 · 安装教程</div>
                </div>
                <div class="button-row">
                  <el-dropdown
                    v-if="platform.apps.length > 1"
                    trigger="click"
                    @command="downloadDashboardClient"
                  >
                    <el-button type="primary" size="small">
                      下载
                      <el-icon><ArrowDown /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item
                          v-for="app in platform.apps"
                          :key="app.downloadKey"
                          :command="app.downloadKey"
                        >
                          {{ app.name }}
                        </el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                  <el-button
                    v-else
                    type="primary"
                    size="small"
                    @click="downloadDashboardClient(platform.apps[0].downloadKey)"
                  >
                    下载
                  </el-button>
                  <el-dropdown
                    v-if="platform.apps.length > 1"
                    trigger="click"
                    @command="openDashboardClientTutorial"
                  >
                    <el-button size="small">
                      教程
                      <el-icon><ArrowDown /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item
                          v-for="app in platform.apps"
                          :key="app.clientId"
                          :command="app.clientId"
                        >
                          {{ app.name }}
                        </el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                  <el-button
                    v-else
                    size="small"
                    @click="openDashboardClientTutorial(platform.apps[0].clientId)"
                  >
                    教程
                  </el-button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <AppDialog
      v-model="rechargeDialogVisible"
      title="账户充值"
      width="500px"
      mobile-width="92%"
      :loading="rechargeLoading"
      class="recharge-dialog"
    >
      <el-form :model="rechargeForm" :rules="rechargeRules" ref="rechargeFormRef" :label-width="isMobile ? '0' : '100px'">
        <el-form-item prop="amount" :label="isMobile ? '' : '充值金额'">
          <template v-if="isMobile">
            <div class="mobile-label">充值金额</div>
          </template>
          <el-input-number
            v-model="rechargeForm.amount"
            :min="0.01"
            :step="1"
            :precision="2"
            placeholder="请输入充值金额"
            class="recharge-amount-input"
            :controls-position="isMobile ? 'right' : 'right'"
          >
            <template #prepend>¥</template>
          </el-input-number>
          <div class="amount-tips">
            <p>默认金额20元，可自定义金额</p>
            <div class="quick-amounts">
              <el-button 
                v-for="amount in quickAmounts" 
                :key="amount"
                size="small"
                :type="rechargeForm.amount === amount ? 'primary' : 'default'"
                @click="selectQuickAmount(amount)"
                class="quick-amount-btn"
              >
                ¥{{ amount }}
              </el-button>
            </div>
          </div>
        </el-form-item>
        <el-form-item label="支付方式" v-if="!isMobile || rechargePaymentMethods.length > 0">
          <template v-if="isMobile">
            <div class="mobile-label">支付方式</div>
          </template>
          <el-radio-group v-model="rechargePaymentMethod" @change="handleRechargePaymentMethodChange">
            <el-radio
              v-for="method in rechargePaymentMethods"
              :key="method.key"
              :label="method.key"
              class="recharge-payment-radio"
            >
              {{ method.name || method.key }}
            </el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <div v-if="rechargeQRCode" class="recharge-qr-section">
        <h4>请使用{{ getRechargePaymentMethodName() }}扫描二维码完成支付</h4>
        <div class="qr-code-wrapper">
          <img :src="rechargeQRCode" alt="支付二维码" class="qr-code-img" />
        </div>
        <p class="qr-tip">支付完成后，余额将自动到账</p>
        <div v-if="isMobile && rechargePaymentUrl && (rechargePaymentUrl.includes('alipay') || rechargePaymentUrl.includes('alipays'))" class="recharge-payment-actions">
          <el-button 
            type="success"
            size="large"
            @click="openAlipayAppForRecharge"
            class="recharge-alipay-btn"
          >
            <el-icon class="btn-leading-icon"><Wallet /></el-icon>
            跳转到支付宝支付
          </el-button>
        </div>
      </div>
      <template #footer>
        <FormActionBar :loading="rechargeLoading">
          <el-button :disabled="rechargeLoading" @click="closeRechargeDialog">关闭</el-button>
          <el-button 
            type="primary" 
            @click="createRecharge" 
            :loading="rechargeLoading"
            :disabled="!!rechargeQRCode"
          >
            {{ rechargeQRCode ? '支付中...' : '确认充值' }}
          </el-button>
        </FormActionBar>
      </template>
    </AppDialog>

    <!-- 升级设备数量抽屉 -->
    <UpgradeDevicesDrawer
      v-model="showUpgradeDrawer"
      :subscription="dashboardUpgradeSubscription"
      :on-success="handleUpgradeSuccess"
    />
  </div>
</template>
<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { ElMessage, ElNotification } from '@/utils/elementPlusServices'
import {
  Aim,
  ArrowDown,
  Box,
  Cellphone,
  Clock,
  CopyDocument,
  Cpu,
  Document,
  InfoFilled,
  Iphone,
  Lightning,
  Link,
  Medal,
  Monitor,
  Picture,
  Promotion,
  Reading,
  Top,
  Trophy,
  View,
  Wallet,
  WarningFilled
} from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'
import AppDialog from '@/components/AppDialog.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import LoadingState from '@/components/LoadingState.vue'
import UpgradeDevicesDrawer from '@/components/UpgradeDevicesDrawer.vue'
import { userAPI, subscriptionAPI, softwareConfigAPI, rechargeAPI, settingsAPI, checkinAPI, useApi, cachedAPI, pendingPaymentStorage } from '@/utils/api'
import { formatDate as formatDateUtil, getRemainingDays, isExpired as isExpiredUtil } from '@/utils/date'
import { copyToClipboard as copyText } from '@/utils/textSelection'
import { safeNavigate, safeOpen, safeOpenApp } from '@/utils/safeOpen'
import { sanitizeBasicHtml, sanitizePlainText } from '@/utils/sanitizeHtml'
import { useMobile } from '@/composables/useMobile'
import { usePaymentStatusPolling } from '@/composables/usePaymentStatusPolling'
import { createQRCodeDataURL } from '@/utils/qrcode'
const router = useRouter()
const api = useApi()
const sanitizeHtml = sanitizeBasicHtml
const userInfo = ref({
  username: '用户',
  email: '',
  membership: '普通会员',
  expire_time: null,
  expiryDate: '未设置',
  remaining_days: 0,
  online_devices: 0,
  total_devices: 0,
  balance: '0.00',
  speed_limit: '不限速',
  subscription_url: '',
  subscription_status: 'inactive',
  clashUrl: '',
  universalUrl: '',
  qrcodeUrl: ''
})
const subscriptionInfo = ref({
  currentDevices: 0,
  maxDevices: 0,
  remainingDays: 0,
  expiryDate: '未设置',
  status: 'inactive'
})
const checkinLoading = ref(false)
const dashboardLoading = ref(true)
const checkedIn = ref(false)
const handleCheckin = async () => {
  checkinLoading.value = true
  try {
    const res = await checkinAPI.checkin()
    const data = res.data?.data || res.data
    checkedIn.value = true
    userInfo.value.balance = data.balance
    ElMessage.success(`签到成功！获得 ¥${data.amount} 奖励`)
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '签到失败')
  } finally {
    checkinLoading.value = false
  }
}
const loadCheckinStatus = async () => {
  try {
    const res = await checkinAPI.getStatus()
    const data = res.data?.data || res.data
    checkedIn.value = data.checked_in
  } catch (e) {}
}
const rechargeDialogVisible = ref(false)
const rechargeForm = ref({
  amount: 20
})
const rechargeRules = {
  amount: [
    { required: true, message: '请输入充值金额', trigger: 'blur' },
    { type: 'number', min: 0.01, message: '充值金额必须大于0', trigger: 'blur' }
  ]
}
const rechargeFormRef = ref()
const rechargeLoading = ref(false)
const rechargeQRCode = ref('')
const rechargePaymentUrl = ref('') // 保存支付URL，用于跳转支付宝App
const rechargePaymentMethod = ref('alipay')
const rechargePaymentMethods = ref([])
const isMobile = useMobile()
let resizeRafId = null
const quickAmounts = [20, 50, 100, 200, 500, 1000]
const loadRechargePaymentMethods = async () => {
  try {
    const response = await api.get('/payment-methods/active')
    if (response && response.data) {
      let methods = []
      if (response.data.success && response.data.data) {
        methods = Array.isArray(response.data.data) ? response.data.data : []
      } else if (Array.isArray(response.data)) {
        methods = response.data
      } else if (response.data.data && Array.isArray(response.data.data)) {
        methods = response.data.data
      }
      rechargePaymentMethods.value = methods
      if (methods.length > 0) {
        rechargePaymentMethod.value = methods[0].key
      }
    }
  } catch (error) {
    rechargePaymentMethods.value = [{ key: 'alipay', name: '支付宝' }]
  }
}
const handleRechargePaymentMethodChange = (value) => {
}
const softwareConfig = ref({
  clash_windows_url: '',
  v2rayn_url: '',
  clash_party_windows_url: '',
  clash_verge_windows_url: '',
  hiddify_windows_url: '',
  flash_windows_url: '',
  clash_android_url: '',
  v2rayng_url: '',
  hiddify_android_url: '',
  flash_macos_url: '',
  clash_party_macos_url: '',
  clash_verge_macos_url: '',
  shadowrocket_url: ''
})
const activePlatform = ref('Windows')
const showQRCode = ref(false)
const showUpgradeDrawer = ref(false)
const platformIconMap = {
  windows: Monitor,
  android: Cellphone,
  macos: Cpu,
  ios: Iphone,
  linux: Cpu,
  mobile: Cellphone
}
const getPlatformIcon = (icon) => platformIconMap[icon] || Monitor
const platforms = ref([
  {
    name: 'Windows',
    icon: 'windows',
    apps: [
      {
        name: 'Clash for Windows',
        version: 'Latest',
        downloadKey: 'clash_windows_url',
        clientId: 'clash-windows'
      },
      {
        name: 'V2rayN',
        version: 'Latest',
        downloadKey: 'v2rayn_url',
        clientId: 'v2rayn',
        githubKey: 'v2rayn'
      },
      {
        name: 'Clash Part',
        version: 'Latest',
        downloadKey: 'clash_party_windows_url',
        clientId: 'clash-party',
        githubKey: 'clash-party'
      },
      {
        name: 'Clash Verge',
        version: 'Latest',
        downloadKey: 'clash_verge_windows_url',
        clientId: 'clash-verge',
        githubKey: 'clash-verge'
      },
      {
        name: 'Hiddify',
        version: 'Latest',
        downloadKey: 'hiddify_windows_url',
        clientId: 'hiddify',
        githubKey: 'hiddify'
      },
      {
        name: 'FlClash',
        version: 'Latest',
        downloadKey: 'flash_windows_url',
        clientId: 'flclash',
        githubKey: 'flclash'
      }
    ]
  },
  {
    name: 'Android',
    icon: 'android',
    apps: [
      {
        name: 'Clash Meta',
        version: 'Latest',
        downloadKey: 'clash_android_url',
        clientId: 'clash-meta'
      },
      {
        name: 'V2rayNG',
        version: 'Latest',
        downloadKey: 'v2rayng_url',
        clientId: 'v2rayng',
        githubKey: 'v2rayng'
      },
      {
        name: 'Hiddify',
        version: 'Latest',
        downloadKey: 'hiddify_android_url',
        clientId: 'hiddify',
        githubKey: 'hiddify'
      }
    ]
  },
  {
    name: 'macOS',
    icon: 'macos',
    apps: [
      {
        name: 'FlClash',
        version: 'Latest',
        downloadKey: 'flash_macos_url',
        clientId: 'flclash',
        githubKey: 'flclash'
      },
      {
        name: 'Clash Part',
        version: 'Latest',
        downloadKey: 'clash_party_macos_url',
        clientId: 'clash-party',
        githubKey: 'clash-party'
      },
      {
        name: 'Clash Verge',
        version: 'Latest',
        downloadKey: 'clash_verge_macos_url',
        clientId: 'clash-verge',
        githubKey: 'clash-verge'
      }
    ]
  },
  {
    name: 'iOS',
    icon: 'ios',
    apps: [
      {
        name: 'Shadowrocket',
        version: 'Latest',
        downloadKey: 'shadowrocket_url',
        clientId: 'shadowrocket'
      }
    ]
  }
])
const qrCodeUrl = ref('')
async function generateSubQRCode() {
  const data = userInfo.value.qrcodeUrl || userInfo.value.universalUrl
  if (!data) { qrCodeUrl.value = ''; return }
  try {
    qrCodeUrl.value = await createQRCodeDataURL(data, { width: 200, margin: 2, errorCorrectionLevel: 'M' })
  } catch {
    qrCodeUrl.value = ''
  }
}
const isDeviceOverlimit = computed(() => {
  const onlineDevices = userInfo.value.online_devices || subscriptionInfo.value.currentDevices || 0
  const deviceLimit = userInfo.value.total_devices || subscriptionInfo.value.maxDevices || 0
  return deviceLimit > 0 && onlineDevices > deviceLimit
})
const isDeviceWarning = computed(() => {
  const onlineDevices = userInfo.value.online_devices || subscriptionInfo.value.currentDevices || 0
  const deviceLimit = userInfo.value.total_devices || subscriptionInfo.value.maxDevices || 0
  if (deviceLimit <= 0 || onlineDevices > deviceLimit) return false
  return onlineDevices >= Math.ceil(deviceLimit * 0.8)
})
const specialNodeModeText = computed(() => {
  const lineMode = userInfo.value.special_node_subscription_type === 'special_only' ? '仅显示专线线路' : '显示专线和普通线路'
  const deviceMode = userInfo.value.special_node_unlimited_devices ? '设备不限制' : '设备跟随系统限制'
  return `${lineMode}，${deviceMode}`
})
const normalizeHexColor = (color, fallback = '#409eff') => {
  if (typeof color !== 'string') return fallback
  const trimmed = color.trim()
  if (/^#[0-9a-fA-F]{6}$/.test(trimmed)) return trimmed
  if (/^#[0-9a-fA-F]{3}$/.test(trimmed)) {
    return `#${trimmed.slice(1).split('').map(char => char + char).join('')}`
  }
  return fallback
}
const hexToRgba = (color, alpha) => {
  const hex = normalizeHexColor(color)
  const value = parseInt(hex.slice(1), 16)
  const red = (value >> 16) & 255
  const green = (value >> 8) & 255
  const blue = value & 255
  return `rgba(${red}, ${green}, ${blue}, ${alpha})`
}
const levelThemeStyle = computed(() => {
  const levelColor = normalizeHexColor(userInfo.value.user_level?.color)
  return {
    '--level-color': levelColor,
    '--level-bg-soft': hexToRgba(levelColor, 0.08),
    '--level-shadow-soft': hexToRgba(levelColor, 0.16),
    '--level-shadow-medium': hexToRgba(levelColor, 0.28),
    '--level-shadow-strong': hexToRgba(levelColor, 0.34)
  }
})
const upgradeProgressStyle = computed(() => {
  const rawPercentage = Number(userInfo.value.upgrade_progress?.percentage || 0)
  const percentage = Number.isFinite(rawPercentage) ? Math.min(Math.max(rawPercentage, 0), 100) : 0
  const nextLevelColor = normalizeHexColor(userInfo.value.next_level?.color, '#67c23a')
  return {
    '--upgrade-progress': `${percentage}%`,
    '--next-level-color': nextLevelColor,
    '--next-level-color-soft': hexToRgba(nextLevelColor, 0.72)
  }
})
const dashboardUpgradeSubscription = computed(() => ({
  device_limit: userInfo.value.total_devices || subscriptionInfo.value.maxDevices || 0,
  maxDevices: userInfo.value.total_devices || subscriptionInfo.value.maxDevices || 0,
  expire_time: subscriptionInfo.value.expiryDate || userInfo.value.expire_time,
  expiryDate: subscriptionInfo.value.expiryDate || userInfo.value.expire_time
}))

const dashboardClientGroups = computed(() => platforms.value.map(platform => ({
  ...platform,
  clientNames: (platform.apps || []).map(app => app.name).join(' / ')
})).filter(platform => platform.apps?.length))

const dashboardRemainingDays = computed(() => {
  const days = getRemainingDays(subscriptionInfo.value.expiryDate || userInfo.value.expire_time || userInfo.value.expiryDate)
  return Number.isFinite(days) ? days : 0
})

const handleUpgradeSuccess = async () => {
  cachedAPI.clearUserCache()
  await Promise.all([loadUserInfo(), loadSubscriptionInfo()])
}

const formatDate = (dateString) => {
  if (!dateString) return '未知'
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN')
}
const loadUserInfo = async () => {
  dashboardLoading.value = true
  try {
    // 使用缓存的 API，减少重复请求
    const dashboardResponse = await cachedAPI.getUserInfo()
    if (dashboardResponse.data && dashboardResponse.data.success) {
      const dashboardData = dashboardResponse.data.data
      userInfo.value = {
        ...dashboardData,
        balance: dashboardData.balance || '0.00',
        clashUrl: dashboardData.clashUrl || dashboardData.subscription?.clashUrl || '',
        universalUrl: dashboardData.universalUrl || dashboardData.subscription?.universalUrl || '',
        qrcodeUrl: dashboardData.qrcodeUrl || dashboardData.subscription?.qrcodeUrl || '',
        expiryDate: dashboardData.expiryDate || dashboardData.expire_time || dashboardData.subscription?.expiryDate || dashboardData.subscription?.expire_time || '未设置',
        expire_time: dashboardData.expire_time || dashboardData.expiryDate || dashboardData.subscription?.expire_time || dashboardData.subscription?.expiryDate || '未设置',
        remaining_days: dashboardData.remainingDays || dashboardData.remaining_days || dashboardData.subscription?.remainingDays || dashboardData.subscription?.remaining_days || 0,
        subscription_status: dashboardData.subscription?.status || dashboardData.subscription_status || 'inactive',
        has_special_nodes: !!(dashboardData.has_special_nodes || dashboardData.subscription?.has_special_nodes),
        special_node_count: dashboardData.special_node_count || dashboardData.subscription?.special_node_count || 0,
        special_node_subscription_type: dashboardData.special_node_subscription_type || dashboardData.subscription?.special_node_subscription_type || 'both',
        special_node_unlimited_devices: !!(dashboardData.special_node_unlimited_devices || dashboardData.subscription?.special_node_unlimited_devices)
      }
      const calculatedRemainingDays = dashboardData.remainingDays || dashboardData.remaining_days || dashboardData.subscription?.remainingDays || dashboardData.subscription?.remaining_days || 0
      subscriptionInfo.value = {
        currentDevices: dashboardData.subscription?.currentDevices || 0,
        maxDevices: dashboardData.subscription?.maxDevices || 0,
        remainingDays: calculatedRemainingDays,
        expiryDate: dashboardData.expiryDate || dashboardData.expire_time || dashboardData.subscription?.expiryDate || dashboardData.subscription?.expire_time || '未设置',
        status: dashboardData.subscription?.status || dashboardData.subscription_status || 'inactive'
      }
      if (dashboardData.notice) {
        handleAnnouncement(dashboardData.notice)
      }
      generateSubQRCode()
    } else {
      throw new Error('用户信息加载失败')
    }
  } catch (error) {
    try {
      const subscriptionResponse = await subscriptionAPI.getUserSubscription()
      if (subscriptionResponse.data && subscriptionResponse.data.success) {
        const subscriptionData = subscriptionResponse.data.data
        userInfo.value = {
          username: '用户',
          email: '',
          membership: '普通会员',
          expire_time: null,
          expiryDate: subscriptionData.expiryDate || '未设置',
          remaining_days: subscriptionData.remainingDays || 0,
          online_devices: 0,
          total_devices: 0,
          balance: '0.00',
          subscription_url: subscriptionData.subscription_url || '',
          subscription_status: subscriptionData.status || 'inactive',
          clashUrl: subscriptionData.clashUrl || '',
          universalUrl: subscriptionData.universalUrl || '',
          qrcodeUrl: subscriptionData.qrcodeUrl || '',
          has_special_nodes: !!subscriptionData.has_special_nodes,
          special_node_count: subscriptionData.special_node_count || 0,
          special_node_subscription_type: subscriptionData.special_node_subscription_type || 'both',
          special_node_unlimited_devices: !!subscriptionData.special_node_unlimited_devices
        }
        ElMessage.warning('部分信息加载失败，但订阅地址可用')
      } else {
        throw new Error('订阅API也返回空数据')
      }
    } catch (fallbackError) {
      ElMessage.error('加载用户信息失败，请刷新页面重试')
    }
  } finally {
    dashboardLoading.value = false
  }
}
const handleAnnouncement = (notice) => {
  if (!notice || !notice.enabled || !notice.content) {
    return
  }
  const content = String(notice.content).trim()
  if (!content) {
    return
  }
  const sanitizedContent = sanitizePlainText(content)
  ElNotification({
    title: '系统公告',
    message: sanitizedContent,
    type: 'info',
    position: 'bottom-right',
    duration: 0,
    dangerouslyUseHTMLString: false,
    showClose: true
  })
}
const loadSubscriptionInfo = async () => {
  try {
    const response = await cachedAPI.getUserSubscription()
    if (response.data && response.data.success) {
      subscriptionInfo.value = response.data.data
      } else {
      subscriptionInfo.value = {
        currentDevices: 0,
        maxDevices: 0,
        remainingDays: 0,
        expiryDate: '未设置',
        status: 'inactive'
      }
    }
  } catch (error) {
    subscriptionInfo.value = {
      currentDevices: 0,
      maxDevices: 0,
      remainingDays: 0,
      expiryDate: '未设置',
      status: 'inactive'
    }
  }
}
const showRechargeDialog = () => {
  rechargeDialogVisible.value = true
  rechargeForm.value.amount = 20
  rechargeQRCode.value = ''
  rechargePaymentUrl.value = ''
  currentRechargeOrderNo.value = null
  loadRechargePaymentMethods()
  cleanupRechargeStatusCheck()
}
const openAlipayAppForRecharge = () => {
  if (!rechargePaymentUrl.value) {
    ElMessage.error('支付链接不存在')
    return
  }
  const alipayAppUrl = `alipays://platformapi/startapp?saId=10000007&qrcode=${encodeURIComponent(rechargePaymentUrl.value)}`
  try {
    cleanupRechargeManualWatchers()
    rechargeManualVisibilityHandler = async () => {
      if (document.visibilityState === 'visible' && currentRechargeOrderNo.value) {
        await checkRechargeStatus()
        cleanupRechargeManualWatchers()
      }
    }
    document.addEventListener('visibilitychange', rechargeManualVisibilityHandler)
    rechargeManualFocusHandler = async () => {
      if (currentRechargeOrderNo.value) {
        await checkRechargeStatus()
        cleanupRechargeManualWatchers()
      }
    }
    window.addEventListener('focus', rechargeManualFocusHandler)
    safeNavigate(alipayAppUrl, { allowAppProtocols: true })
    setTimeout(() => {
      ElMessage.info('如果未跳转到支付宝，请使用支付宝扫描上方二维码完成支付')
    }, 3000)
  } catch (error) {
    ElMessage.error('跳转失败，请使用支付宝扫描二维码完成支付')
  }
}
const selectQuickAmount = (amount) => {
  rechargeForm.value.amount = amount
}
const getRechargePaymentMethodName = () => {
  const method = rechargePaymentMethods.value.find(m => m.key === rechargePaymentMethod.value)
  return method?.name || method?.key || '支付宝'
}
const createRecharge = async () => {
  try {
    await rechargeFormRef.value.validate()
    if (rechargeForm.value.amount <= 0) {
      ElMessage.error('充值金额必须大于0')
      return
    }
    rechargeLoading.value = true
    const response = await rechargeAPI.createRecharge({
      amount: rechargeForm.value.amount,
      payment_method: rechargePaymentMethod.value
    })
    if (response.data && response.data.success !== false) {
      const data = response.data.data
      if (data.payment_error) {
        ElMessage.warning(data.payment_error || '支付链接生成失败')
        return
      }
      const paymentUrl = data.payment_url || data.payment_qr_code
      if (!paymentUrl) {
        ElMessage.error('支付链接生成失败，请稍后重试')
        return
      }
      const rechargeId = data.id || data.recharge_id
      const rechargeOrderNo = data.order_no
      if (!rechargeId || !rechargeOrderNo) {
        console.error('充值订单信息不完整:', data)
        ElMessage.error('充值订单创建失败，订单信息缺失')
        return
      }
      pendingPaymentStorage.save(rechargeOrderNo, 'recharge')
      const isYipay = rechargePaymentMethod.value && (
        rechargePaymentMethod.value.includes('yipay') || 
        rechargePaymentMethod.value.includes('易支付') ||
        rechargePaymentMethod.value.includes('codepay') ||
        rechargePaymentMethod.value.includes('码支付')
      )
      if (isYipay) {
        if (paymentUrl) {
          currentRechargeOrderNo.value = rechargeOrderNo
          rechargePaymentUrl.value = paymentUrl
          ElMessage.info('正在跳转到支付页面...')
          safeNavigate(paymentUrl, { allowAppProtocols: true })
          startRechargeStatusCheck()
        } else {
          ElMessage.error('支付链接不存在')
        }
      } else {
        rechargePaymentUrl.value = paymentUrl
        currentRechargeOrderNo.value = rechargeOrderNo
        try {
          const qrOptions = {
            width: isMobile.value ? 200 : 256,
            margin: 2,
            color: {
              dark: '#000000',
              light: '#FFFFFF'
            },
            errorCorrectionLevel: 'M'
          }
          const qrCodeDataURL = await createQRCodeDataURL(paymentUrl, qrOptions)
          rechargeQRCode.value = qrCodeDataURL
          ElMessage.success('充值订单创建成功，请扫描二维码完成支付')
          startRechargeStatusCheck()
        } catch (qrError) {
          rechargeQRCode.value = paymentUrl
          ElMessage.success('充值订单创建成功，请扫描二维码完成支付')
          startRechargeStatusCheck()
        }
      }
    } else {
      ElMessage.error(response.data?.message || '创建充值订单失败')
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.detail || '创建充值订单失败')
  } finally {
    rechargeLoading.value = false
  }
}
let rechargeStatusInterval = null
let rechargeVisibilityHandler = null
let rechargeFocusHandler = null
let rechargeStatusTimeoutId = null
let rechargeStatusRequest = null
let rechargeManualVisibilityHandler = null
let rechargeManualFocusHandler = null
const currentRechargeOrderNo = ref(null)
const cleanupRechargeManualWatchers = () => {
  if (rechargeManualVisibilityHandler) {
    document.removeEventListener('visibilitychange', rechargeManualVisibilityHandler)
    rechargeManualVisibilityHandler = null
  }
  if (rechargeManualFocusHandler) {
    window.removeEventListener('focus', rechargeManualFocusHandler)
    rechargeManualFocusHandler = null
  }
}
const cleanupRechargeStatusCheck = () => {
  cleanupRechargeManualWatchers()
}
const { startPolling: startRechargeStatusCheck } = usePaymentStatusPolling({
  intervalMs: 5000,
  timeoutMs: 30 * 60 * 1000,
  shouldPoll: () => !!currentRechargeOrderNo.value,
  poll: () => checkRechargeStatus(),
  onCleanup: cleanupRechargeManualWatchers,
})
const closeRechargeDialog = () => {
  cleanupRechargeStatusCheck()
  rechargeDialogVisible.value = false
  rechargeQRCode.value = ''
  rechargePaymentUrl.value = ''
  currentRechargeOrderNo.value = null
}
const checkRechargeStatus = async () => {
  if (!currentRechargeOrderNo.value) {
    return
  }
  if (rechargeStatusRequest) return rechargeStatusRequest
  rechargeStatusRequest = (async () => {
  try {
    const response = await rechargeAPI.getRechargeStatus(currentRechargeOrderNo.value)
    if (!response || !response.data) {
      return
    }
    if (response.data.success === false) {
      return
    }
    const rechargeData = response.data.data
    if (!rechargeData) {
      return
    }
    if (rechargeData.status === 'paid') {
      cleanupRechargeStatusCheck()
      ElMessage.success('充值成功！余额已到账')
      pendingPaymentStorage.clear()
      await cachedAPI.refreshUserState({ includeSubscription: false })
      await loadUserInfo()
      window.dispatchEvent(new CustomEvent('user-info-updated'))
      closeRechargeDialog()
    } else if (rechargeData.status === 'cancelled') {
      cleanupRechargeStatusCheck()
      pendingPaymentStorage.clear()
      closeRechargeDialog()
      ElMessage.warning('充值订单已取消')
    }
  } catch (error) {
    if (error.response?.status === 404) {
      cleanupRechargeStatusCheck()
    }
  } finally {
    rechargeStatusRequest = null
  }
  })()
  return rechargeStatusRequest
}
const loadSoftwareConfig = async () => {
  try {
    // 使用缓存的 API，减少重复请求
    const response = await cachedAPI.getSoftwareConfig()
    if (response.data && response.data.success) {
      softwareConfig.value = response.data.data || {}
    }
  } catch (error) {
    }
}
const downloadApp = async (appName) => {
  const clientKeyMap = {
    'clash_windows_url': null, // Clash for Windows 使用配置的链接
    'v2rayn_url': 'v2rayn',
    'clash_party_windows_url': 'clash-party',
    'clash_party_macos_url': 'clash-party',
    'clash_verge_windows_url': 'clash-verge',
    'clash_verge_macos_url': 'clash-verge',
    'hiddify_windows_url': 'hiddify',
    'hiddify_android_url': 'hiddify',
    'flash_windows_url': 'flclash',
    'flash_macos_url': 'flclash',
    'clash_android_url': null, // Clash Meta 使用配置的链接
    'v2rayng_url': 'v2rayng',
    'shadowrocket_url': null // Shadowrocket 使用 App Store 链接
  }
  const clientKey = clientKeyMap[appName]
  const configUrl = softwareConfig.value[appName]
  if (configUrl) {
    safeOpen(configUrl)
    return
  }
  if (appName === 'shadowrocket_url') {
    safeOpen('https://apps.apple.com/app/shadowrocket/id932747118')
    return
  }
  if (clientKey) {
    try {
      ElMessage.info('正在获取最新下载链接...')
      const { getClientDownloadUrl, getClientReleasesUrl } = await import('@/utils/githubDownload')
      const downloadUrl = await getClientDownloadUrl(clientKey, softwareConfig.value || {})
      safeOpen(downloadUrl)
      ElMessage.success('已打开下载页面')
    } catch (error) {
      console.error('获取下载链接失败:', error)
      try {
        const { getClientReleasesUrl } = await import('@/utils/githubDownload')
        const releasesUrl = getClientReleasesUrl(clientKey)
        if (releasesUrl) {
          safeOpen(releasesUrl)
          ElMessage.warning('已打开发布页面，请手动选择下载')
        } else {
          ElMessage.error('无法获取下载链接，请联系管理员')
        }
      } catch (err) {
        ElMessage.error('下载链接获取失败，请联系管理员')
      }
    }
  } else {
    ElMessage.error('下载链接未配置，请联系管理员')
  }
}
const openTutorial = (app) => {
  const clientId = typeof app === 'string'
    ? app.replace(/^\/help#?/, '').replace(/^#/, '')
    : app?.clientId
  if (clientId) {
    router.push({ path: '/help', query: { client: clientId } })
    return
  }
  router.push('/help')
}
const openTutorialByPlatform = (platformName) => {
  const platform = platforms.value.find(item => item.name === platformName)
  const app = platform?.apps?.[0]
  if (app) {
    openTutorial(app)
    return
  }
  router.push('/help')
}
const downloadDashboardClient = (downloadKey) => {
  if (!downloadKey) {
    ElMessage.error('下载链接未配置，请联系管理员')
    return
  }
  downloadApp(downloadKey)
}
const openDashboardClientTutorial = (clientId) => {
  if (!clientId) {
    router.push('/help')
    return
  }
  openTutorial(clientId)
}
const goToPackages = () => {
  router.push('/packages')
}
const loadDevices = async () => {
  try {
    await loadUserInfo()
  } catch (error) {
  }
}
const executeCommand = (command, handlers) => {
  const handler = handlers[command]
  if (handler) {
    handler()
  }
}
const getExpiryName = (withSuffix = true) => {
  const expiryDateValue = userInfo.value?.expiryDate
  if (!expiryDateValue || expiryDateValue === '未设置') {
    return ''
  }
  const expiryDate = new Date(expiryDateValue)
  if (isNaN(expiryDate.getTime())) {
    return ''
  }
  const year = expiryDate.getFullYear()
  const month = String(expiryDate.getMonth() + 1).padStart(2, '0')
  const day = String(expiryDate.getDate()).padStart(2, '0')
  return `到期时间${year}-${month}-${day}${withSuffix ? '_到期' : ''}`
}
const ensureSubscriptionUrl = (url, errorMessage = '订阅地址不可用，请先购买套餐或刷新页面重试') => {
  if (!url) {
    ElMessage.error(errorMessage)
    return false
  }
  return true
}
const copySubscriptionUrl = async (url, successMessage, errorMessage) => {
  if (!ensureSubscriptionUrl(url, errorMessage)) {
    return
  }
  await copyText(url, successMessage)
}
const importClashBasedSubscription = (client, successMessage) => {
  const clashUrl = userInfo.value?.clashUrl
  if (!ensureSubscriptionUrl(clashUrl)) {
    return
  }
  try {
    oneclickImport(client, clashUrl, getExpiryName(true))
    ElMessage.success(successMessage)
  } catch (error) {
    ElMessage.error('一键导入失败，请手动复制订阅地址')
  }
}
const handleClashCommand = (command) => executeCommand(command, {
  'copy-clash': copyClashSubscription,
  'import-clash': importClashSubscription
})
const handleFlashCommand = (command) => executeCommand(command, {
  'copy-flash': copyFlashSubscription,
  'import-flash': importFlashSubscription
})
const handleClashPartyCommand = (command) => executeCommand(command, {
  'copy-clash-party': copyClashPartySubscription,
  'import-clash-party': importClashPartySubscription
})
const handleClashVergeCommand = (command) => executeCommand(command, {
  'copy-clash-verge': copyClashVergeSubscription,
  'import-clash-verge': importClashVergeSubscription
})
const handleShadowrocketCommand = (command) => executeCommand(command, {
  'copy-shadowrocket': copyShadowrocketSubscription,
  'import-shadowrocket': importShadowrocketSubscription
})
const copyClashSubscription = () => copySubscriptionUrl(
  userInfo.value?.clashUrl,
  'Clash 订阅地址已复制到剪贴板',
  'Clash 订阅地址不可用，请刷新页面重试'
)
const copyFlashSubscription = () => copySubscriptionUrl(
  userInfo.value?.clashUrl,
  'Flash 订阅地址已复制到剪贴板'
)
const copyClashPartySubscription = () => copySubscriptionUrl(
  userInfo.value?.clashUrl,
  'Clash Part 订阅地址已复制到剪贴板'
)
const copyClashVergeSubscription = () => copySubscriptionUrl(
  userInfo.value?.clashUrl,
  'Clash Verge 订阅地址已复制到剪贴板'
)
const copyUniversalSubscription = () => copySubscriptionUrl(
  userInfo.value?.universalUrl,
  '通用订阅地址已复制到剪贴板',
  '订阅地址不可用，请先购买套餐'
)
const copyHiddifySubscription = () => copySubscriptionUrl(
  userInfo.value?.universalUrl,
  '通用订阅地址已复制到剪贴板',
  '通用订阅地址不可用'
)
const copyShadowrocketSubscription = () => copySubscriptionUrl(
  userInfo.value?.universalUrl,
  '通用订阅地址已复制到剪贴板'
)
const importClashSubscription = () => importClashBasedSubscription('clashx', '正在打开 Clash 客户端...')
const importFlashSubscription = () => importClashBasedSubscription('flash', '正在打开 Flash 客户端...')
const importClashPartySubscription = () => importClashBasedSubscription('clash-party', '正在打开 Clash Part 客户端...')
const importClashVergeSubscription = () => importClashBasedSubscription('clash-verge', '正在打开 Clash Verge 客户端...')
const importShadowrocketSubscription = () => {
  const universalUrl = userInfo.value?.universalUrl
  if (!ensureSubscriptionUrl(universalUrl, '通用订阅地址不可用，请刷新页面重试')) {
    return
  }
  try {
    oneclickImport('shadowrocket', universalUrl, getExpiryName(false))
    ElMessage.success('正在打开 Shadowrocket 客户端...')
  } catch (error) {
    ElMessage.error('一键导入失败，请手动复制订阅地址')
  }
}
const refreshDevices = () => {
  loadDevices()
  ElMessage.success('设备列表已刷新')
}
const oneclickImport = (client, url, name = '') => {
  try {
    const clashCompatibleClients = new Set(['clashx', 'clash', 'flash', 'clash-party', 'clash-verge'])
    if (clashCompatibleClients.has(client)) {
      const baseUrl = `clash://install-config?url=${encodeURIComponent(url)}`
      const targetUrl = name ? `${baseUrl}&name=${encodeURIComponent(name)}` : baseUrl
      safeOpenApp(targetUrl)
      return
    }
    switch (client) {
      case 'shadowrocket':
        let shadowrocketUrl = `shadowrocket://add/sub://${btoa(url)}`
        if (name) {
          shadowrocketUrl += `#${encodeURIComponent(name)}`
        }
        safeOpenApp(shadowrocketUrl)
        break
      case 'ssr':
        safeOpenApp(`ssr://${btoa(url)}`)
        break
      case 'quantumult':
        safeOpenApp(`quantumult://resource?url=${encodeURIComponent(url)}`)
        break
      case 'quantumult_v2':
        safeOpenApp(`quantumult-x://resource?url=${encodeURIComponent(url)}`)
        break
      case 'v2rayng':
        safeOpenApp(`v2rayng://install-config?url=${encodeURIComponent(url)}`)
        break
      case 'hiddify':
        safeOpenApp(`hiddify://install-config?url=${encodeURIComponent(url)}`)
        break
      default:
        safeOpen(url)
    }
  } catch (error) {
    ElMessage.error('一键导入失败，请手动复制订阅地址')
  }
}
const checkAndShowAnnouncement = async () => {
  try {
    const response = await settingsAPI.getPublicSettings()
    const settings = response.data?.data || response.data || {}
    const isEnabled = settings.announcement_enabled === true || 
                      settings.announcement_enabled === 'true' || 
                      String(settings.announcement_enabled).toLowerCase() === 'true'
    if (isEnabled && settings.announcement_content && String(settings.announcement_content).trim()) {
      handleAnnouncement({
        enabled: isEnabled,
        content: settings.announcement_content
      })
    }
  } catch (error) {
  }
}
onMounted(() => {
  Promise.all([
    loadUserInfo(),
    loadSoftwareConfig(),
    loadCheckinStatus(),
    checkAndShowAnnouncement()
  ]).catch(err => {
    console.error('Dashboard 初始化失败:', err)
  })

  const handleSubscriptionUpdate = async (event) => {
    if (event?.detail?.refreshed) return
    cachedAPI.clearUserCache()
    await Promise.all([
      loadSubscriptionInfo(),
      loadUserInfo()
    ])
  }
  const handleUserInfoUpdate = async (event) => {
    if (event?.detail?.refreshed) return
    cachedAPI.clearUserCache()
    await loadUserInfo()
  }
  window.addEventListener('subscription-updated', handleSubscriptionUpdate)
  window.addEventListener('user-info-updated', handleUserInfoUpdate)
  onUnmounted(() => {
    window.removeEventListener('subscription-updated', handleSubscriptionUpdate)
    window.removeEventListener('user-info-updated', handleUserInfoUpdate)
  })
})
onUnmounted(() => {
  cleanupRechargeStatusCheck()
  if (typeof window !== 'undefined' && resizeRafId !== null) {
    window.cancelAnimationFrame(resizeRafId)
    resizeRafId = null
  }
})
</script>
<style scoped>
.dashboard-container {
  padding: 0;
  max-width: none;
  margin: 0;
  width: 100%;
}
.breadcrumb {
  margin-bottom: 12px;
  color: #606266;
  font-size: 13px;
  line-height: 1.4;
}
.dashboard-container > .page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
  padding: 16px;
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  box-shadow: none;
}
.dashboard-container > .page-header .page-title h1 {
  margin: 0;
  color: #303133;
  font-size: 22px;
  line-height: 1.25;
  font-weight: 700;
}
.dashboard-container > .page-header .page-title p {
  margin: 6px 0 0;
  color: #606266;
  line-height: 1.5;
}
.dashboard-container > .page-header .actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}
.dashboard-container > .page-header .actions .el-button {
  margin-left: 0;
  min-height: 44px;
  touch-action: manipulation;
}
.special-user-alert {
  margin-bottom: 20px;
  border-radius: 8px;
}
.expiry-alert {
  margin-bottom: 16px;
}
.expiry-renew-btn {
  margin-top: 4px;
}
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 14px;
}
.stat-card {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #dcdfe6;
  display: flex;
  align-items: flex-start;
  transition: background-color 0.2s ease, border-color 0.2s ease;
  &.level-card {
    --level-color: #409eff;
    --level-bg-soft: #fff;
    border-width: 1px;
    border-color: #dcdfe6;
    background: #fff;
    position: relative;
    overflow: clip;
    padding: 16px;
    &::before {
      content: none;
    }
    .level-card-inner {
      display: flex;
      align-items: flex-start;
      gap: 14px;
      width: 100%;
    }
    .level-left {
      flex-shrink: 0;
    }
    .level-content {
      flex: 1;
      min-width: 0;
    }
    .level-header {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 4px;
      flex-wrap: wrap;
      .level-name {
        color: var(--level-color);
        margin: 0;
        font-size: 22px;
        font-weight: 800;
        letter-spacing: 0;
        line-height: 1.2;
      }
      .level-discount-tag {
        color: #fff;
        background-color: var(--level-color);
        border: none;
        border-radius: 4px;
        font-size: 13px;
        font-weight: 700;
        padding: 0 8px;
        transition: opacity 0.2s ease;
        &:hover {
          opacity: 0.9;
        }
      }
    }
    .level-expiry {
      font-size: 13px;
      color: #909399;
      margin: 0 0 10px 0;
      display: flex;
      align-items: center;
      gap: 6px;
      font-weight: 400;
      .inline-icon {
        font-size: 14px;
        opacity: 0.7;
      }
    }
    .level-icon {
      background: #ecf5ff;
      border-color: transparent;
      color: var(--level-color);
      width: 46px;
      height: 46px;
      border-radius: 8px;
      font-size: 20px;
      transition: background-color 0.2s ease;
    }
    .upgrade-progress {
      margin-top: 12px;
      width: 100%;
      .progress-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 6px;
        .progress-label {
          font-size: 12px;
          color: #666;
          font-weight: 500;
        }
        .progress-percentage {
          font-size: 14px;
          color: #409eff;
          font-weight: 600;
        }
      }
      .progress-bar {
        --next-level-color: #67c23a;
        --next-level-color-soft: rgba(103, 194, 58, 0.72);
        --upgrade-progress: 0%;
        width: 100%;
        height: 10px;
        background-color: #f0f0f0;
        border-radius: 5px;
        overflow: clip;
        margin-bottom: 8px;
        .progress-fill {
          height: 100%;
          width: var(--upgrade-progress);
          background: var(--next-level-color);
          border-radius: 5px;
          transition: width 0.3s ease;
        }
      }
      .progress-text {
        font-size: 12px;
        color: #666;
        margin: 0 0 4px 0;
        line-height: 1.5;
        .inline-icon {
          margin-right: 4px;
          color: #67c23a;
        }
        .next-level-name {
          color: var(--next-level-color, #67c23a);
        }
      }
      .progress-tip {
        display: flex;
        align-items: flex-start;
        gap: 6px;
        font-size: 11px;
        color: var(--color-text-secondary, #909399);
        margin: 0;
        padding: 6px 8px;
        background: #f5f7fa;
        border-radius: 4px;
        line-height: 1.4;

        .progress-tip-icon {
          flex-shrink: 0;
          margin-top: 1px;
          color: #409eff;
          font-size: 13px;
        }
      }
    }
    .max-level-tip {
      margin-top: 10px;
      padding: 8px 10px;
      background: #fffbeb;
      border: 1px solid #fde68a;
      border-radius: 4px;
      color: #92400e;
      font-size: 12px;
      font-weight: 600;
      text-align: left;
      position: relative;
      overflow: clip;
      .inline-icon {
        margin-right: 8px;
        color: #d97706;
        font-size: 16px;
      }
    }
  }
}
.stat-card:hover {
  background: #fbfdff;
  border-color: var(--el-color-primary-light-7, #c6e2ff);
}
.stat-icon {
  width: 46px;
  height: 46px;
  min-width: 46px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 14px;
  font-size: 20px;
  border: 1px solid transparent;
}
.stat-card:nth-child(1) .stat-icon {
  background: #ecf5ff;
  border-color: transparent;
  color: #409eff;
}
.stat-card:nth-child(2) .stat-icon {
  background: #ecf5ff;
  border-color: transparent;
  color: #409eff;
}
.stat-card:nth-child(3) .stat-icon {
  background: #ecf5ff;
  border-color: transparent;
  color: #409eff;
}
.stat-card:nth-child(4) .stat-icon {
  background: #ecf5ff;
  border-color: transparent;
  color: #409eff;
}
.stat-title {
  font-size: 1.5rem;
  font-weight: 700;
  margin: 0 0 4px 0;
  color: var(--color-text, #1f2937);
}
.stat-subtitle {
  font-size: 0.875rem;
  color: var(--color-text-secondary, #6b7280);
  margin: 0;
  margin-top: 4px;
}
.device-card {
  position: relative;
  .device-count-wrapper {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-bottom: 4px;
  }
  .device-count {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--color-text, #1f2937);
    transition: color 0.3s ease;
  }
  .device-separator {
    font-size: 1.2rem;
    color: var(--color-text-secondary, #9ca3af);
    margin: 0 2px;
  }
  .device-limit {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--color-text-secondary, #6b7280);
  }
  .device-overlimit-count {
    color: #ef4444 !important;
    animation: blink 1s infinite;
  }
  .device-warning-count {
    color: #f59e0b !important;
  }
  .device-alert {
    margin-top: 8px;
    padding: 6px 10px;
    background: #fee2e2;
    border: 1px solid #fecaca;
    border-radius: 6px;
    color: #dc2626;
    font-size: 0.75rem;
    display: flex;
    align-items: center;
    gap: 6px;
    animation: blink 1s infinite;
    .inline-icon {
      font-size: 0.875rem;
    }
  }
  .upgrade-device-btn {
    margin-top: 10px;
    width: 100%;
  }
  &.device-overlimit {
    border-color: #ef4444 !important;
    background: #fef2f2 !important;
  }
  &.device-warning {
    border-color: #f59e0b !important;
    background: #fffbeb !important;
  }
}
@keyframes blink {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
.expiry-subtitle {
  word-break: break-word;
  line-height: 1.4;
  @media (max-width: 768px) {
    font-size: 0.75rem;
    line-height: 1.3;
  }
  @media (max-width: 480px) {
    font-size: 0.6875rem;
    line-height: 1.4;
  }
}
.balance-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  .stat-content {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    flex: 1;
    min-width: 0;
    gap: 12px;
  }
  .balance-main {
    flex: 1;
    min-width: 0;
  }
  .balance-actions {
    display: flex;
    gap: 8px;
    flex-shrink: 0;
  }
  .balance-actions .el-button {
    padding: 8px 16px;
    font-weight: 600;
    border-radius: 8px;
    white-space: nowrap;
    font-size: 0.8125rem;
    box-sizing: border-box;
    height: auto;
  }
  .balance-actions .el-button .el-icon {
    margin-right: 4px;
    font-size: 12px;
  }
  .recharge-btn {
    margin-left: 12px;
    padding: 8px 16px;
    font-weight: 600;
    border-radius: 8px;
    white-space: nowrap;
    font-size: 0.8125rem;
    flex-shrink: 0;
    box-sizing: border-box;
    max-width: fit-content;
    height: auto;
    .el-icon {
      margin-right: 4px;
      font-size: 12px;
    }
    @media (max-width: 768px) {
      padding: 6px 12px;
      font-size: 0.75rem;
      margin-left: 0;
      .el-icon {
        margin-right: 3px;
        font-size: 11px;
      }
    }
    @media (max-width: 480px) {
      padding: 8px 16px;
      font-size: 0.8125rem;
      border-radius: 8px;
      .el-icon {
        margin-right: 4px;
        font-size: 12px;
      }
    }
  }
}
.remaining-time-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  overflow: clip;
  .stat-content {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    flex: 1;
    min-width: 0;
    gap: 12px;
    box-sizing: border-box;
  }
  .remaining-time-main {
    flex: 1;
    min-width: 0;
    overflow: clip;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .remaining-time-value {
    display: flex;
    align-items: baseline;
    gap: 4px;
    margin: 0 0 4px 0;
  }
  .time-number {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--color-text, #1f2937);
    line-height: 1.3;
    margin: 0;
  }
  .time-unit {
    font-size: 1rem;
    font-weight: 600;
    color: var(--color-text-secondary, #6b7280);
  }
  .remaining-time-card .stat-subtitle {
    margin: 0;
    font-size: 0.875rem;
    color: var(--color-text-secondary, #6b7280);
    line-height: 1.4;
    word-break: break-word;
  }
  .renew-btn {
    margin-left: 12px;
    padding: 8px 16px;
    font-weight: 600;
    border-radius: 8px;
    white-space: nowrap;
    font-size: 0.8125rem;
    flex-shrink: 0;
    box-sizing: border-box;
    max-width: fit-content;
    height: auto;
    .el-icon {
      margin-right: 4px;
      font-size: 12px;
    }
    @media (max-width: 768px) {
      padding: 6px 12px;
      font-size: 0.75rem;
      margin-left: 0;
      .el-icon {
        margin-right: 3px;
        font-size: 11px;
      }
    }
    @media (max-width: 480px) {
      padding: 8px 16px;
      font-size: 0.8125rem;
      border-radius: 8px;
      .el-icon {
        margin-right: 4px;
        font-size: 12px;
      }
    }
  }
  @media (max-width: 768px) {
    padding: 16px 12px;
    .stat-content {
      flex-direction: row;
      align-items: center;
      gap: 12px;
    }
    .remaining-time-title {
      font-size: 0.75rem;
      margin-bottom: 6px;
      line-height: 1.2;
    }
    .time-number {
      font-size: 1.75rem;
    }
    .time-unit {
      font-size: 0.875rem;
    }
    .expiry-date {
      font-size: 0.75rem;
      margin-top: 6px;
      line-height: 1.3;
      word-break: break-word;
    }
    .renew-btn {
      margin-left: 0;
      padding: 6px 12px;
      font-size: 0.75rem;
      flex-shrink: 0;
      box-sizing: border-box;
      max-width: fit-content;
      height: auto;
      .el-icon {
        margin-right: 3px;
        font-size: 11px;
      }
    }
  }
  @media (max-width: 480px) {
    padding: 14px 12px;
    .stat-content {
      flex-direction: column;
      align-items: center;
      gap: 10px;
    }
    .remaining-time-main {
      width: 100%;
      text-align: center;
    }
    .remaining-time-title {
      font-size: 0.8125rem;
      margin-bottom: 8px;
    }
    .remaining-time-value {
      justify-content: center;
    }
    .time-number {
      font-size: 2rem;
    }
    .time-unit {
      font-size: 1rem;
    }
    .expiry-date {
      font-size: 0.6875rem;
      margin-top: 8px;
      line-height: 1.4;
      word-break: break-word;
      color: var(--color-text-secondary, #6b7280);
      text-align: center;
    }
    .renew-btn {
      margin-left: 0;
      width: auto;
      padding: 8px 16px;
      font-size: 0.8125rem;
      border-radius: 8px;
      box-sizing: border-box;
      max-width: fit-content;
      align-self: center;
      .el-icon {
        margin-right: 4px;
        font-size: 12px;
      }
    }
  }
}
.recharge-dialog {
  :deep(.el-form-item) {
    margin-bottom: 20px;
    @media (max-width: 768px) {
      margin-bottom: 16px;
    }
  }
  :deep(.el-form-item__label) {
    @media (max-width: 768px) {
      font-size: 14px;
      padding-bottom: 8px;
      width: 100% !important;
      text-align: left;
      margin-bottom: 8px;
      display: none; /* 移动端隐藏默认标签 */
    }
  }
  .mobile-label {
    font-size: 14px;
    font-weight: 500;
    color: #606266;
    margin-bottom: 8px;
    display: block;
    @media (min-width: 769px) {
      display: none;
    }
  }
  :deep(.el-form-item__content) {
    @media (max-width: 768px) {
      margin-left: 0 !important;
    }
  }
  :deep(.el-input-number) {
    width: 100%;
    @media (max-width: 768px) {
      width: 100%;
    }
    :deep(.el-input__wrapper) {
      @media (max-width: 768px) {
        padding: 8px 12px;
      }
    }
    :deep(.el-input__inner) {
      @media (max-width: 768px) {
        font-size: 16px; /* 防止iOS自动缩放 */
        height: 44px;
      }
    }
  }
  .recharge-amount-input {
    width: 100%;
  }
  :deep(.el-radio-group) {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    width: 100%;
  }
.recharge-payment-radio {
  margin: 0;
  min-height: 44px;
  padding: 6px 10px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  touch-action: manipulation;
  transition: border-color 0.16s ease, background-color 0.16s ease;

    &:hover {
      background-color: #f5f9ff;
      border-color: #c6e2ff;
    }

    &.is-checked {
      background-color: #ecf5ff;
      border-color: var(--el-color-primary);
    }
  }
  .amount-tips {
    margin-top: 12px;
    font-size: 12px;
    color: var(--color-text-secondary, #909399);
    @media (max-width: 768px) {
      margin-top: 12px;
      font-size: 12px;
    }
    :is(p) {
      margin-bottom: 12px;
      line-height: 1.5;
      @media (max-width: 768px) {
        margin-bottom: 10px;
        font-size: 12px;
      }
    }
    .quick-amounts {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin-top: 10px;
      @media (max-width: 768px) {
        gap: 8px;
        margin-top: 12px;
      }
      .quick-amount-btn {
        margin: 0;
        flex: 1 1 calc(33.333% - 6px);
        min-width: calc(33.333% - 6px);
        max-width: calc(33.333% - 6px);
        padding: 10px 8px;
        font-size: 13px;
        border-radius: 6px;
        @media (max-width: 480px) {
          flex: 1 1 calc(50% - 4px);
          min-width: calc(50% - 4px);
          max-width: calc(50% - 4px);
          padding: 12px 8px;
          font-size: 14px;
        }
      }
    }
  }
  .recharge-qr-section {
    margin-top: 20px;
    text-align: center;
    padding: 20px;
    background: #f5f7fa;
    border-radius: 8px;
    @media (max-width: 768px) {
      margin-top: 16px;
      padding: 16px;
      border-radius: 8px;
    }
    :is(h4) {
      margin-bottom: 15px;
      color: #303133;
      font-size: 16px;
      font-weight: 600;
      line-height: 1.4;
      @media (max-width: 768px) {
        font-size: 15px;
        margin-bottom: 12px;
        padding: 0 8px;
      }
    }
    .qr-code-wrapper {
      display: flex;
      justify-content: center;
      align-items: center;
      margin: 20px 0;
      @media (max-width: 768px) {
        margin: 16px 0;
      }
      .qr-code-img {
        max-width: 250px;
        max-height: 250px;
        width: 100%;
        height: auto;
        border: 1px solid #dcdfe6;
        border-radius: 8px;
        padding: 10px;
        background: var(--color-bg-card, #fff);
        box-sizing: border-box;
        @media (max-width: 768px) {
          max-width: 220px;
          max-height: 220px;
          padding: 10px;
        }
        @media (max-width: 480px) {
          max-width: 200px;
          max-height: 200px;
          padding: 8px;
        }
      }
    }
    .qr-tip {
      color: var(--color-text-secondary, #909399);
      font-size: 12px;
      margin-top: 12px;
      line-height: 1.5;
      padding: 0 8px;
      @media (max-width: 768px) {
        font-size: 12px;
        margin-top: 10px;
      }
    }
    .recharge-payment-actions {
      margin-top: 15px;
      @media (max-width: 768px) {
        margin-top: 12px;
      }
      .el-button {
        width: 100%;
        padding: 12px 20px;
        font-size: 15px;
        border-radius: 8px;
        font-weight: 600;
        @media (max-width: 480px) {
          padding: 14px 20px;
          font-size: 16px;
        }
      }
      .recharge-alipay-btn {
        width: 100%;
      }
      .btn-leading-icon {
        margin-right: 5px;
      }
    }
  }
}
.dashboard-main-aside {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(320px, 0.85fr);
  align-items: start;
  gap: 14px;
}
.section-stack {
  display: grid;
  gap: 14px;
}
.dashboard-section-card {
  margin-bottom: 0;
}
.card {
  background: #fff;
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  margin-bottom: 14px;
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
  font-size: 16px;
  font-weight: 700;
  margin: 0;
  color: #303133;
  display: flex;
  align-items: center;
  gap: 8px;
}
.card-sub {
  margin-top: 4px;
  color: #909399;
  font-size: 13px;
  line-height: 1.45;
}
.card-body {
  padding: 16px;
}
.button-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.notice {
  padding: 12px;
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  color: #606266;
  background: #f5f7fa;
  line-height: 1.5;
}
.notice.success {
  border-color: #e1f3d8;
  background: #f0f9eb;
  color: #67c23a;
}
.notice.danger {
  border-color: #fde2e2;
  background: #fef0f0;
  color: #f56c6c;
}
.table-wrapper {
  overflow-x: auto;
}
.dashboard-table {
  width: 100%;
  border-collapse: collapse;
  background: #fff;
  font-size: 14px;
}
.dashboard-table th,
.dashboard-table td {
  padding: 12px;
  border-bottom: 1px solid #ebeef5;
  text-align: left;
  vertical-align: middle;
  white-space: nowrap;
}
.dashboard-table th {
  background: #f5f7fa;
  color: #606266;
  font-weight: 700;
}
.dashboard-table tr:last-child td {
  border-bottom: 0;
}
.tutorial-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}
.tutorial-tab {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s ease, border-color 0.2s ease, color 0.2s ease;
  font-size: 0.875rem;
  font-weight: 500;
}
.tutorial-tab:hover {
  border-color: #409eff;
  background-color: #ecf5ff;
}
.tutorial-tab.active {
  border-color: #409eff;
  background-color: #409eff;
  color: white;
}
.tutorial-app {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 14px 0;
  border-bottom: 1px solid #ebeef5;
}
.tutorial-app:last-child {
  border-bottom: 0;
  padding-bottom: 0;
}
.app-info {
  display: flex;
  align-items: center;
  gap: 12px;
}
.app-name {
  font-size: 1rem;
  font-weight: 600;
  margin: 0 0 4px 0;
  color: var(--color-text, #1f2937);
}
.app-version {
  font-size: 0.875rem;
  color: var(--color-text-secondary, #6b7280);
  margin: 0;
}
.app-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}
.subscription-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 14px;
  @media (max-width: 768px) {
    gap: 10px;
    margin-bottom: 16px;
  }
  @media (max-width: 480px) {
    gap: 8px;
  }
}
.subscription-group {
  display: flex;
  min-width: 0;
  @media (max-width: 768px) {
    width: 100%;
  }
}
.subscription-group .el-button,
.subscription-group :deep(.el-button) {
  margin-left: 0;
  min-height: 44px;
  touch-action: manipulation;
}
.clash-btn {
  width: 100%;
}
.shadowrocket-btn {
  width: 100%;
}
.v2ray-btn {
  width: 100%;
}
.universal-btn {
  width: 100%;
}
.qr-code-section {
  text-align: center;
  padding: 16px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  background: #f8fbff;
}
.qr-code-container {
  margin-top: 12px;
}
.software-category {
  margin-bottom: 14px;
}
.category-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 700;
  color: #303133;
  margin: 0 0 12px;
}
.category-title .title-icon {
  color: #409eff;
}
.subscription-urls-section {
  margin-bottom: 14px;
}
.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 700;
  color: #303133;
  margin: 0 0 12px;
}
.section-title .title-icon {
  color: #409eff;
}
.url-display {
  display: grid;
  gap: 10px;
}
.url-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.url-item :is(label) {
  font-weight: 500;
  color: #606266;
  font-size: 13px;
  margin-bottom: 4px;
}
.url-input-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
  position: relative;
  width: 100%;
}
.url-input {
  flex: 1;
  min-width: 0; /* 防止flex子元素溢出 */
}
.copy-btn {
  min-width: 48px !important;
  max-width: 48px !important;
  height: 44px !important;
  padding: 8px 6px !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
  gap: 3px !important;
  flex-shrink: 0;
  border-radius: 4px;
  background-color: var(--color-bg-card, #fff) !important;
  border: 1px solid #dcdfe6 !important;
  touch-action: manipulation !important;
  color: var(--color-text, #000) !important;
  transition: background-color 0.2s ease, border-color 0.2s ease, color 0.2s ease;
  font-size: 11px !important;
  white-space: nowrap;
  overflow: clip;
  box-sizing: border-box;
  &:hover {
    background-color: #f5f7fa !important;
    border-color: #c0c4cc !important;
    color: var(--color-text, #000) !important;
  }
  &:active {
    background-color: #ebedf0 !important;
  }
  .el-icon {
    font-size: 11px !important;
    color: var(--color-text, #000) !important;
    flex-shrink: 0;
  }
  :is(span) {
    font-size: 11px !important;
    color: var(--color-text, #000) !important;
    font-weight: 400;
    line-height: 1;
    flex-shrink: 0;
  }
}
.qr-code-section {
  margin-bottom: 24px;
}
.qr-code-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 8px;
  border: 1px solid #dcdfe6;
}
.qr-code {
  width: 200px;
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-card, #fff);
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 8px;
}
.qr-code img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  border-radius: 8px;
}
.qr-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: #999;
}
.qr-placeholder .el-icon {
  font-size: 48px;
}
.qr-tip {
  font-size: 14px;
  color: #666;
  text-align: center;
  margin: 0;
}
.flash-btn {
  width: 100%;
  border-radius: 4px;
  padding: 10px 14px;
  font-weight: 600;
  transition: background-color 0.2s ease, border-color 0.2s ease;
  @media (max-width: 768px) {
    padding: 16px 20px;
    font-size: 15px;
  }
}
.clash-party-btn {
  width: 100%;
  border-radius: 4px;
  padding: 10px 14px;
  font-weight: 600;
  transition: background-color 0.2s ease, border-color 0.2s ease;
  @media (max-width: 768px) {
    padding: 16px 20px;
    font-size: 15px;
  }
}
.clash-verge-btn {
  width: 100%;
  border-radius: 4px;
  padding: 10px 14px;
  font-weight: 600;
  transition: background-color 0.2s ease, border-color 0.2s ease;
  @media (max-width: 768px) {
    padding: 16px 20px;
    font-size: 15px;
  }
}
.hiddify-btn {
  width: 100%;
  border-radius: 4px;
  padding: 10px 14px;
  font-weight: 600;
  transition: background-color 0.2s ease, border-color 0.2s ease;
  @media (max-width: 768px) {
    padding: 16px 20px;
    font-size: 15px;
  }
}
.qr-code img {
  width: 200px;
  height: 200px;
  border-radius: 8px;
}
.qr-tip {
  font-size: 0.875rem;
  color: var(--color-text-secondary, #6b7280);
  margin: 12px 0 0 0;
}
.device-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.device-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 8px;
  margin-bottom: 12px;
}
.device-info {
  display: flex;
  align-items: center;
  gap: 12px;
}
.device-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  background: #eef2ff;
  border: 1px solid #c7d2fe;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #4f46e5;
  font-size: 18px;
}
.device-name {
  font-size: 1rem;
  font-weight: 600;
  margin: 0 0 4px 0;
  color: var(--color-text, #1f2937);
}
.device-os, .device-ip {
  font-size: 0.875rem;
  color: var(--color-text-secondary, #6b7280);
  margin: 0;
}
.no-devices {
  text-align: center;
  padding: 40px 20px;
  color: var(--color-text-secondary, #9ca3af);
}
.no-devices .el-icon {
  font-size: 3rem;
  margin-bottom: 16px;
  display: block;
}
@media (max-width: 768px) {
  .dashboard-container {
    padding: 0;
  }
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 10px;
    margin-bottom: 16px;
    @media (max-width: 480px) {
      grid-template-columns: 1fr;
      gap: 12px;
    }
  }
  .level-card::before,
  .max-level-tip::before,
  .level-icon::before {
    animation: none !important;
    display: none;
  }
  .stats-grid {
    .stat-card {
      padding: 16px;
      display: flex;
      align-items: flex-start;
      gap: 12px;
      .stat-icon {
        width: 48px;
        height: 48px;
        font-size: 22px;
        margin-right: 0;
        flex-shrink: 0;
        border-radius: 10px;
      }
      .stat-content {
        flex: 1;
        min-width: 0;
        display: flex;
        flex-direction: column;
        gap: 6px;
        .stat-title {
          font-size: 1.25rem;
          margin: 0;
          word-break: break-word;
          line-height: 1.3;
          font-weight: 700;
        }
        .stat-subtitle {
          font-size: 0.8125rem;
          line-height: 1.4;
          word-break: break-word;
          margin: 0;
          color: var(--color-text-secondary, #6b7280);
        }
      }
    }
    .level-card {
      padding: 16px;
      .level-card-inner {
        gap: 14px;
      }
    .level-icon {
      width: 56px;
      height: 56px;
      font-size: 26px;
      border-radius: 8px;
    }
      .level-content {
        .level-header {
          margin-bottom: 10px;
          gap: 8px;
          .level-name {
            font-size: 1.5rem;
            line-height: 1.2;
          }
          .level-discount-tag {
            font-size: 12px;
            padding: 4px 10px;
          }
        }
        .level-expiry {
          font-size: 0.8125rem;
          margin-bottom: 12px;
        }
      }
    }
    .balance-card {
      .stat-content {
        flex-direction: column;
        align-items: stretch;
        gap: 12px;
      }
      .balance-main {
        flex: 1;
        min-width: 0;
        text-align: center;
      }
      .balance-actions {
        display: flex;
        gap: 8px;
        width: 100%;
      }
      .balance-actions .el-button {
        flex: 1;
        padding: 8px 12px;
        font-size: 0.75rem;
      }
      .recharge-btn {
        padding: 6px 12px;
        font-size: 0.75rem;
        flex-shrink: 0;
        white-space: nowrap;
      }
    }
    .device-card {
      .stat-content {
        width: 100%;
      }
      .device-count-wrapper {
        margin-bottom: 6px;
        .device-count {
          font-size: 1.5rem;
        }
        .device-separator {
          font-size: 1.1rem;
        }
        .device-limit {
          font-size: 1.5rem;
        }
      }
      .stat-subtitle {
        margin-top: 4px;
      }
    }
    .remaining-time-card {
      grid-column: 1 / -1; /* 占据整行 */
      padding: 16px;
      .stat-content {
        flex-direction: row;
        align-items: center;
        gap: 12px;
        width: 100%;
      }
      .remaining-time-main {
        flex: 1;
        min-width: 0;
      }
      .time-number {
        font-size: 1.25rem;
      }
      .time-unit {
        font-size: 0.875rem;
      }
      .stat-subtitle {
        font-size: 0.75rem;
        line-height: 1.3;
      }
      .renew-btn {
        padding: 6px 12px;
        font-size: 0.75rem;
        white-space: nowrap;
        flex-shrink: 0;
      }
    }
  }
  .dashboard-main-aside {
    grid-template-columns: 1fr;
    gap: 12px;
    .left-content,
    .right-content {
      width: 100%;
    }
  }
  .card {
    margin-bottom: 12px;
    .card-header {
      padding: 12px 16px;
      .card-title {
        font-size: 1rem;
        .title-icon {
          font-size: 16px;
          margin-right: 6px;
        }
      }
    }
    .card-body {
      padding: 16px;
    }
  }
  .tutorial-tabs {
    gap: 8px;
    margin-bottom: 16px;
    display: flex;
    flex-wrap: nowrap;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    padding-bottom: 4px; /* 预留滚动条空间 */
    &::-webkit-scrollbar {
      display: none;
    }
    .tutorial-tab {
      padding: 10px 16px;
      font-size: 0.8125rem;
      flex: 0 0 auto; /* 防止压缩 */
      white-space: nowrap;
      .platform-icon {
        font-size: 14px;
      }
    }
  }
  .subscription-buttons {
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    margin-bottom: 20px;
    .el-button {
      padding: 14px 12px;
      font-size: 14px;
      border-radius: 8px;
      font-weight: 600;
      transition: background-color 0.2s ease, border-color 0.2s ease;
      white-space: nowrap;
      overflow: clip;
      text-overflow: ellipsis;
      .el-icon {
        font-size: 14px;
        margin-right: 4px;
      }
    }
  }
  .software-category {
    margin-bottom: 24px;
    .category-title {
      font-size: 15px;
      margin-bottom: 14px;
      padding-bottom: 10px;
    }
  }
  .url-item {
    gap: 6px;
    :is(label) {
      font-size: 12px;
      margin-bottom: 2px;
    }
  }
  .url-input-wrapper {
    flex-direction: row !important;
    align-items: center !important;
    gap: 6px !important;
    width: 100% !important;
    .url-input {
      flex: 1 !important;
      min-width: 0 !important;
    }
    .copy-btn {
      min-width: 48px !important;
      max-width: 48px !important;
      height: 44px !important;
      padding: 8px 6px !important;
      font-size: 11px !important;
      flex-shrink: 0 !important;
      gap: 3px !important;
      .el-icon {
        font-size: 11px !important;
      }
      :is(span) {
        font-size: 11px !important;
      }
    }
  }
  .qr-code-container {
    padding: 16px;
    .qr-code {
      width: 160px;
      height: 160px;
    }
    .qr-tip {
      font-size: 0.8125rem;
      margin-top: 12px;
    }
  }
  .device-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
    padding: 14px;
    .device-info {
      width: 100%;
    }
    .device-actions {
      width: 100%;
      .el-button {
        width: 100%;
        margin-bottom: 8px;
        &:last-child {
          margin-bottom: 0;
        }
      }
    }
  }
}
@media (max-width: 480px) {
  .stats-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }
  .stat-card {
    padding: 16px;
    gap: 12px;
    .stat-icon {
      width: 48px;
      height: 48px;
      font-size: 22px;
      border-radius: 10px;
    }
    .stat-content {
      gap: 6px;
      .stat-title {
        font-size: 1.25rem;
        line-height: 1.3;
      }
      .stat-subtitle {
        font-size: 0.8125rem;
        line-height: 1.4;
      }
    }
  }
  .level-card {
    .level-icon {
      width: 56px;
      height: 56px;
      font-size: 26px;
    }
    .level-content {
      .level-header {
        .level-name {
          font-size: 1.5rem;
        }
      }
    }
  }
  .balance-card {
    .stat-content {
      flex-direction: row;
      align-items: center;
      gap: 12px;
    }
    .balance-main {
      flex: 1;
      min-width: 0;
    }
    .recharge-btn {
      padding: 8px 16px;
      font-size: 0.8125rem;
      flex-shrink: 0;
      white-space: nowrap;
    }
  }
  .device-card {
    .device-count-wrapper {
      .device-count,
      .device-limit {
        font-size: 1.5rem;
      }
    }
  }
  .remaining-time-card {
    .stat-content {
      flex-direction: row;
      align-items: center;
      gap: 12px;
    }
    .remaining-time-main {
      flex: 1;
      min-width: 0;
      gap: 4px;
    }
    .time-number {
      font-size: 1.25rem;
    }
    .time-unit {
      font-size: 0.875rem;
    }
    .stat-subtitle {
      font-size: 0.75rem;
      line-height: 1.3;
      text-align: left;
    }
    .renew-btn {
      padding: 8px 16px;
      font-size: 0.8125rem;
      flex-shrink: 0;
      white-space: nowrap;
    }
  }
  .card-body {
    padding: 12px;
  }
  .subscription-buttons {
    grid-template-columns: 1fr 1fr;
    gap: 8px;
    .el-button {
      padding: 12px 10px;
      font-size: 13px;
      border-radius: 8px;
      .el-icon {
        font-size: 12px;
        margin-right: 3px;
      }
    }
  }
  .url-input-wrapper {
    gap: 6px !important;
    .copy-btn {
      min-width: 44px !important;
      max-width: 44px !important;
      height: 44px !important;
      padding: 8px 5px !important;
      font-size: 10px !important;
      gap: 2px !important;
      .el-icon {
        font-size: 10px !important;
      }
      :is(span) {
        font-size: 10px !important;
      }
    }
  }
  .qr-code-container {
    .qr-code {
      width: 140px;
      height: 140px;
    }
  }
}

.order-entry-card {
  margin-bottom: 16px;
}

.order-entry-list {
  display: grid;
  gap: 10px;
}

.order-entry-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  background: #f8fbff;
  color: #303133;
  font-size: 14px;
}

.device-summary {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px;
  border: 1px solid #e1f3d8;
  border-radius: 8px;
  background: #f0f9eb;
}

.device-summary-warning {
  border-color: #faecd8;
  background: #fdf6ec;
}

.device-summary-danger {
  border-color: #fde2e2;
  background: #fef0f0;
}

.device-summary-icon {
  width: 44px;
  height: 44px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  background: #ecf5ff;
  color: #409eff;
  font-size: 20px;
}

.device-summary-value {
  color: #303133;
  font-size: 22px;
  font-weight: 800;
  line-height: 1.2;
}

.device-summary-value span {
  color: #606266;
  font-size: 16px;
  font-weight: 700;
}

.device-summary-label {
  margin-top: 4px;
  color: #909399;
  font-size: 13px;
}

.device-notice {
  margin-top: 12px;
}

.device-actions-row {
  margin-top: 14px;
}

.device-actions-row .el-button {
  margin-left: 0;
}

.compact-order-table .el-button {
  margin-left: 0;
}

.level-card {
  --level-bg-soft: #fff !important;
  border-width: 1px !important;
  border-color: #dcdfe6 !important;
  background: #fff !important;
  padding: 16px !important;
}

.level-card::before,
.level-icon::before {
  content: none !important;
  display: none !important;
}

.level-card .level-name {
  font-size: 22px !important;
  letter-spacing: 0 !important;
}

.level-card .level-discount-tag {
  border-radius: 4px !important;
}

.stat-icon,
.level-icon {
  width: 46px !important;
  height: 46px !important;
  border-radius: 8px !important;
  margin-right: 14px !important;
  background: #ecf5ff !important;
  border: 1px solid #d9ecff !important;
  color: #409eff !important;
  font-size: 20px !important;
}

.progress-bar {
  height: 8px !important;
  border-radius: 4px !important;
}

.progress-fill {
  background: #409eff !important;
  border-radius: 4px !important;
}

.dashboard-container .stats-grid {
  align-items: stretch;
}

.dashboard-container .stat-card {
  min-height: 80px;
  align-items: center !important;
  gap: 14px;
}

.dashboard-container .stat-card .stat-content {
  min-width: 0;
  flex: 1;
}

.dashboard-container .stat-value,
.dashboard-container .device-count,
.dashboard-container .device-limit,
.dashboard-container .time-number {
  color: #303133 !important;
  font-size: 22px !important;
  font-weight: 800 !important;
  line-height: 1.2 !important;
}

.dashboard-container .time-unit {
  color: #303133 !important;
  font-size: 18px !important;
  font-weight: 800 !important;
}

.dashboard-container .stat-label {
  margin: 4px 0 0 !important;
  color: #909399 !important;
  font-size: 13px !important;
  line-height: 1.35 !important;
}

.dashboard-container .balance-card,
.dashboard-container .remaining-time-card {
  justify-content: flex-start !important;
}

.dashboard-container .balance-card .stat-content,
.dashboard-container .remaining-time-card .stat-content {
  display: block !important;
}

.dashboard-container .remaining-time-value {
  margin: 0 !important;
}

.dashboard-container .device-count-wrapper {
  margin-bottom: 0 !important;
}

.dashboard-container .level-card .level-discount-tag {
  margin-top: 6px;
}

.dashboard-container > .breadcrumb {
  margin: 0 0 12px !important;
  color: #606266 !important;
  font-size: 13px !important;
  line-height: 1.4 !important;
}

.dashboard-container > .page-header {
  display: flex !important;
  align-items: flex-start !important;
  justify-content: space-between !important;
  gap: 16px !important;
  min-height: auto !important;
  padding: 16px !important;
  margin: 0 0 14px !important;
  border: 1px solid #dcdfe6 !important;
  border-radius: 8px !important;
  background: #fff !important;
  box-shadow: none !important;
  overflow: visible !important;
}

.dashboard-container > .page-header::before,
.dashboard-container > .page-header::after {
  display: none !important;
  content: none !important;
}

.dashboard-container > .page-header .page-title {
  min-width: 0 !important;
  max-width: 720px !important;
}

.dashboard-container > .page-header .page-title h1 {
  margin: 0 !important;
  color: #303133 !important;
  font-size: 22px !important;
  font-weight: 700 !important;
  line-height: 1.25 !important;
  letter-spacing: 0 !important;
}

.dashboard-container > .page-header .page-title p {
  margin: 6px 0 0 !important;
  color: #606266 !important;
  font-size: 14px !important;
  line-height: 1.5 !important;
  opacity: 1 !important;
}

.dashboard-container > .page-header .actions {
  display: flex !important;
  flex: 0 0 auto !important;
  flex-wrap: wrap !important;
  justify-content: flex-end !important;
  gap: 8px !important;
  margin: 0 !important;
}

.dashboard-container > .page-header .actions .el-button {
  min-width: 86px !important;
  min-height: 44px !important;
  margin: 0 !important;
  border-radius: 4px !important;
  font-weight: 500 !important;
  box-shadow: none !important;
  touch-action: manipulation !important;
}

.dashboard-container .stats-grid {
  grid-template-columns: repeat(4, minmax(0, 1fr)) !important;
  gap: 14px !important;
  margin: 0 0 14px !important;
}

.dashboard-container .stat-card {
  display: flex !important;
  align-items: center !important;
  min-height: 80px !important;
  padding: 16px !important;
  gap: 14px !important;
  border: 1px solid #dcdfe6 !important;
  border-radius: 8px !important;
  background: #fff !important;
  color: #303133 !important;
  box-shadow: none !important;
  transform: none !important;
}

.dashboard-container .stat-card:hover {
  background: #fbfdff !important;
  border-color: #c6e2ff !important;
  transform: none !important;
}

.dashboard-container .stat-icon,
.dashboard-container .level-icon {
  width: 46px !important;
  height: 46px !important;
  min-width: 46px !important;
  margin: 0 !important;
  border: 0 !important;
  border-radius: 8px !important;
  background: #ecf5ff !important;
  color: #409eff !important;
  box-shadow: none !important;
}

.dashboard-container .stat-value,
.dashboard-container .level-name,
.dashboard-container .device-count,
.dashboard-container .device-separator,
.dashboard-container .device-limit,
.dashboard-container .time-number,
.dashboard-container .time-unit {
  color: #409eff !important;
  font-size: 22px !important;
  font-weight: 800 !important;
  line-height: 1.2 !important;
}

.dashboard-container .time-unit {
  margin-left: 4px !important;
}

.dashboard-container .stat-label {
  margin: 4px 0 0 !important;
  color: #909399 !important;
  font-size: 13px !important;
  line-height: 1.4 !important;
}

.dashboard-container .dashboard-main-aside {
  display: grid !important;
  grid-template-columns: minmax(0, 1.45fr) minmax(320px, 0.85fr) !important;
  gap: 14px !important;
  align-items: start !important;
}

.dashboard-container .dashboard-section-card {
  border: 1px solid #dcdfe6 !important;
  border-radius: 8px !important;
  background: #fff !important;
  box-shadow: none !important;
}

.dashboard-container .card-header {
  min-height: auto !important;
  padding: 14px 16px !important;
  border-bottom: 1px solid #ebeef5 !important;
  background: #fff !important;
}

.dashboard-container .card-title {
  margin: 0 !important;
  color: #303133 !important;
  font-size: 16px !important;
  font-weight: 700 !important;
  line-height: 1.3 !important;
}

.dashboard-container .card-title .title-icon {
  display: none !important;
}

.dashboard-container .card-sub {
  margin: 4px 0 0 !important;
  color: #909399 !important;
  font-size: 13px !important;
  line-height: 1.45 !important;
}

@media (max-width: 768px) {
  .dashboard-container > .page-header {
    display: grid !important;
    gap: 10px !important;
    padding: 12px !important;
  }

  .dashboard-container > .page-header .actions,
  .dashboard-container > .page-header .actions .el-button {
    width: 100% !important;
  }

  .dashboard-container > .page-header .actions .el-button {
    min-height: 44px !important;
  }

  .recharge-payment-radio,
  .subscription-group .el-button,
  .subscription-group :deep(.el-button) {
    min-height: 44px;
  }

  .dashboard-container .stats-grid,
  .dashboard-container .dashboard-main-aside {
    grid-template-columns: 1fr !important;
  }
}
</style>
