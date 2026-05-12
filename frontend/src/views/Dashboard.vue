<template>
  <div class="list-container dashboard-container">
    <div class="welcome-banner">
      <div class="banner-content">
        <div class="welcome-text">
          <h1 class="welcome-title">欢迎回来，{{ userInfo.username }}！</h1>
          <p class="welcome-subtitle">享受高速稳定的网络服务体验</p>
        </div>
        <div class="welcome-icon">
          <i class="fas fa-rocket"></i>
        </div>
      </div>
    </div>
    <!-- 到期预警横幅 -->
    <el-alert
      v-if="getRemainingDays(subscriptionInfo.expiryDate || userInfo.expire_time || userInfo.expiryDate) > 0 && getRemainingDays(subscriptionInfo.expiryDate || userInfo.expire_time || userInfo.expiryDate) <= 7"
      :title="`您的订阅将在 ${getRemainingDays(subscriptionInfo.expiryDate || userInfo.expire_time || userInfo.expiryDate)} 天后到期，请及时续费！`"
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
    <!-- 骨架屏 -->
    <div v-if="dashboardLoading" class="stats-grid">
      <el-skeleton v-for="i in 4" :key="i" :rows="3" animated style="padding:20px;background:#fff;border-radius:12px;" />
    </div>
    <div v-else class="stats-grid">
      <div class="stat-card level-card" :style="{ 
        borderColor: userInfo.user_level?.color || '#409eff',
        background: userInfo.user_level?.color ? `linear-gradient(135deg, ${userInfo.user_level.color}12 0%, ${userInfo.user_level.color}05 50%, ${userInfo.user_level.color}08 100%)` : 'linear-gradient(135deg, rgba(64, 158, 255, 0.08) 0%, rgba(64, 158, 255, 0.03) 50%, rgba(64, 158, 255, 0.05) 100%)',
        boxShadow: userInfo.user_level?.color ? `0 8px 32px ${userInfo.user_level.color}20, 0 2px 8px ${userInfo.user_level.color}15` : '0 8px 32px rgba(102, 126, 234, 0.15), 0 2px 8px rgba(102, 126, 234, 0.1)'
      }">
        <div class="level-card-inner">
          <div class="level-left">
            <div class="stat-icon level-icon" :style="{ 
              background: userInfo.user_level?.color ? `linear-gradient(135deg, ${userInfo.user_level.color}, ${userInfo.user_level.color}cc)` : 'linear-gradient(135deg, #667eea, #764ba2)',
              color: '#fff',
              boxShadow: userInfo.user_level?.color ? `0 8px 24px ${userInfo.user_level.color}50, 0 4px 12px ${userInfo.user_level.color}30` : '0 8px 24px rgba(102, 126, 234, 0.4), 0 4px 12px rgba(102, 126, 234, 0.25)'
            }">
              <i class="fas fa-crown"></i>
            </div>
          </div>
          <div class="stat-content level-content">
            <div class="level-header">
              <h3 class="stat-title level-name" :style="{ 
                color: userInfo.user_level?.color || '#409eff',
                textShadow: userInfo.user_level?.color ? `0 2px 8px ${userInfo.user_level.color}30` : '0 2px 8px rgba(64, 158, 255, 0.2)'
              }">
                {{ userInfo.user_level?.name || userInfo.membership || '普通会员' }}
              </h3>
              <el-tag 
                v-if="userInfo.user_level && userInfo.user_level.discount_rate < 1.0"
                class="level-discount-tag"
                :style="{ 
                  backgroundColor: userInfo.user_level.color || '#409eff', 
                  color: '#fff', 
                  border: 'none',
                  fontWeight: '700',
                  fontSize: '13px',
                  padding: '6px 14px',
                  borderRadius: '20px',
                  boxShadow: userInfo.user_level.color ? `0 4px 12px ${userInfo.user_level.color}40` : '0 4px 12px rgba(64, 158, 255, 0.3)'
                }"
              >
                {{ (userInfo.user_level.discount_rate * 10).toFixed(1) }}折
              </el-tag>
            </div>
            <p class="stat-subtitle level-expiry">
              <i class="fas fa-clock"></i>
              到期时间：{{ formatDate(userInfo.expire_time) }}
            </p>
            <div v-if="userInfo.upgrade_progress && userInfo.next_level" class="upgrade-progress">
              <div class="progress-header">
                <span class="progress-label">升级进度</span>
                <span class="progress-percentage">{{ userInfo.upgrade_progress.percentage || 0 }}%</span>
              </div>
              <div class="progress-bar">
                <div 
                  class="progress-fill" 
                  :style="{ 
                    width: `${userInfo.upgrade_progress.percentage || 0}%`,
                    backgroundColor: userInfo.next_level.color || '#67c23a'
                  }"
                ></div>
              </div>
              <p class="progress-text">
                <i class="fas fa-arrow-up"></i>
                距离 <strong :style="{ color: userInfo.next_level.color || '#67c23a' }">{{ userInfo.next_level.name }}</strong> 还需消费 ¥{{ (userInfo.upgrade_progress.remaining || 0).toFixed(2) }}
              </p>
              <p class="progress-tip">
                💡 累计消费达到要求后，系统会自动升级您的等级，享受更多优惠！
              </p>
            </div>
            <div v-else-if="userInfo.user_level" class="max-level-tip">
              <i class="fas fa-trophy"></i>
              您已达到最高等级，享受最大优惠！
            </div>
          </div>
        </div>
      </div>
      <div class="stat-card balance-card">
        <div class="stat-icon">
          <i class="fas fa-wallet"></i>
        </div>
        <div class="stat-content">
          <div class="balance-main">
            <h3 class="stat-title">¥ {{ typeof userInfo.balance === 'string' ? userInfo.balance : (userInfo.balance || 0).toFixed(2) }}</h3>
            <p class="stat-subtitle">账户余额</p>
          </div>
          <div class="balance-actions">
            <el-button
              :type="checkedIn ? 'info' : 'success'"
              size="small"
              :loading="checkinLoading"
              :disabled="checkedIn"
              @click="handleCheckin"
            >
              <i class="fas fa-calendar-check"></i>
              {{ checkedIn ? '已签到' : '签到' }}
            </el-button>
            <el-button
              type="primary"
              class="recharge-btn"
              @click="showRechargeDialog"
            >
              <i class="fas fa-plus"></i>
              充值
            </el-button>
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
          <i class="fas fa-mobile-alt"></i>
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
          <p class="stat-subtitle">在线设备/总设备数</p>
          <div v-if="isDeviceOverlimit" class="device-alert">
            <i class="fas fa-exclamation-triangle"></i>
            <span>设备数量超过限制！</span>
          </div>
          <el-button
            v-if="userInfo.total_devices || subscriptionInfo.maxDevices"
            size="small"
            class="upgrade-device-btn"
            :type="isDeviceOverlimit ? 'danger' : isDeviceWarning ? 'warning' : 'primary'"
            @click="showUpgradeDrawer = true"
          >
            <i class="fas fa-arrow-up"></i>
            升级设备数量
          </el-button>
        </div>
      </div>
      <div class="stat-card remaining-time-card">
        <div class="stat-icon">
          <i class="fas fa-clock"></i>
        </div>
        <div class="stat-content">
          <div class="remaining-time-main">
            <div class="remaining-time-value">
              <span class="time-number">{{ getRemainingDays(subscriptionInfo.expiryDate || userInfo.expire_time || userInfo.expiryDate) }}</span>
              <span class="time-unit">天</span>
            </div>
            <p class="stat-subtitle">到期时间：{{ formatDate(subscriptionInfo.expiryDate || userInfo.expire_time || userInfo.expiryDate) || '未设置' }}</p>
          </div>
          <el-button 
            type="primary" 
            class="renew-btn"
            @click="goToPackages"
          >
            <i class="fas fa-sync-alt"></i>
            续费
          </el-button>
        </div>
      </div>
    </div>
    <div class="main-content">
      <div class="left-content">
        <div class="card subscription-card">
          <div class="card-header">
            <h3 class="card-title">
              <i class="fas fa-link"></i>
              订阅地址
            </h3>
          </div>
          <div class="card-body">
            <div class="software-category">
              <h4 class="category-title">
                <i class="fas fa-bolt"></i>
                Clash系列软件
              </h4>
              <div class="subscription-buttons">
                <div class="subscription-group">
                  <el-dropdown @command="handleClashCommand" trigger="click">
                    <el-button type="primary" class="clash-btn">
                      <i class="fas fa-bolt"></i>
                      Clash
                      <i class="fas fa-chevron-down"></i>
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
                      <i class="fas fa-flash"></i>
                      Flash
                      <i class="fas fa-chevron-down"></i>
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
                  <el-dropdown @command="handleMohomoCommand" trigger="click">
                    <el-button type="primary" class="mohomo-btn">
                      <i class="fas fa-cube"></i>
                      Clash Part
                      <i class="fas fa-chevron-down"></i>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="copy-mohomo">复制订阅</el-dropdown-item>
                        <el-dropdown-item command="import-mohomo">一键导入</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
                <div class="subscription-group">
                  <el-dropdown @command="handleClashVergeCommand" trigger="click">
                    <el-button type="primary" class="clash-verge-btn">
                      <i class="fas fa-bolt"></i>
                      Clash Verge
                      <i class="fas fa-chevron-down"></i>
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
                <i class="fas fa-shield-alt"></i>
                V2Ray系列软件
              </h4>
              <div class="subscription-buttons">
                <div class="subscription-group">
                  <el-button type="info" class="universal-btn" @click="copyUniversalSubscription">
                    <i class="fas fa-shield-alt"></i>
                    复制通用订阅
                  </el-button>
                </div>
                <div class="subscription-group">
                  <el-button type="info" class="hiddify-btn" @click="copyHiddifySubscription">
                    <i class="fas fa-eye"></i>
                    复制 Hiddify Next 订阅
                  </el-button>
                </div>
              </div>
            </div>
            <div class="software-category">
              <h4 class="category-title">
                <i class="fas fa-rocket"></i>
                iOS软件
              </h4>
              <div class="subscription-buttons">
                <div class="subscription-group">
                  <el-dropdown @command="handleShadowrocketCommand" trigger="click">
                    <el-button type="success" class="shadowrocket-btn">
                      <i class="fas fa-rocket"></i>
                      Shadowrocket
                      <i class="fas fa-chevron-down"></i>
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
            <div class="subscription-urls-section">
              <h4 class="section-title">
                <i class="fas fa-link"></i>
                订阅地址
              </h4>
              <div class="url-display">
                <div class="url-item">
                  <label>Clash订阅地址</label>
                  <div class="url-input-wrapper">
                    <el-input 
                      :value="userInfo.clashUrl" 
                      readonly 
                      size="small"
                      class="url-input"
                    />
                    <el-button 
                      @click="copyClashSubscription" 
                      size="small"
                      class="copy-btn"
                    >
                      <i class="fas fa-copy"></i>
                      <span>复制</span>
                    </el-button>
                  </div>
                </div>
              </div>
            </div>
            <div class="qr-code-section">
              <h4 class="section-title">
                <i class="fas fa-qrcode"></i>
                二维码
              </h4>
              <div class="qr-code-container">
                <div class="qr-code">
                  <img :src="qrCodeUrl" alt="订阅二维码" v-if="qrCodeUrl">
                  <div v-else class="qr-placeholder">
                    <i class="fas fa-qrcode"></i>
                    <p>二维码生成中...</p>
                  </div>
                </div>
                <p class="qr-tip">扫描二维码即可在Shadowrocket中添加订阅</p>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="right-content">
        <div class="card tutorial-card">
          <div class="card-header">
            <h3 class="card-title">
              <i class="fas fa-graduation-cap"></i>
              使用教程
            </h3>
          </div>
          <div class="card-body">
            <div class="tutorial-tabs">
              <div 
                v-for="platform in platforms" 
                :key="platform.name"
                class="tutorial-tab"
                :class="{ active: activePlatform === platform.name }"
                @click="activePlatform = platform.name"
              >
                <i :class="platform.icon"></i>
                <span>{{ platform.name }}</span>
              </div>
            </div>
            <div class="tutorial-content">
              <div 
                v-for="platform in platforms" 
                :key="platform.name"
                v-show="activePlatform === platform.name"
                class="tutorial-platform"
              >
                <div 
                  v-for="app in platform.apps" 
                  :key="app.name"
                  class="tutorial-app"
                >
                  <div class="app-info">
                    <div class="app-details">
                      <h4 class="app-name">{{ app.name }}</h4>
                      <p class="app-version">{{ app.version }}</p>
                    </div>
                  </div>
                  <div class="app-actions">
                    <el-button type="primary" size="small" @click="downloadApp(app.downloadKey)">
                      立即下载
                    </el-button>
                    <el-button type="default" size="small" @click="openTutorial(app.tutorialUrl)">
                      安装教程
                    </el-button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <el-dialog
      v-model="rechargeDialogVisible"
      title="账户充值"
      :width="isMobile ? '90%' : '500px'"
      class="recharge-dialog"
      :close-on-click-modal="false"
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
            style="width: 100%"
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
              style="margin-right: 15px; margin-bottom: 10px;"
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
        <div v-if="isMobile && rechargePaymentUrl && (rechargePaymentUrl.includes('alipay') || rechargePaymentUrl.includes('alipays'))" class="recharge-payment-actions" style="margin-top: 15px;">
          <el-button 
            type="success"
            size="large"
            @click="openAlipayAppForRecharge"
            style="width: 100%;"
          >
            <el-icon style="margin-right: 5px;"><Wallet /></el-icon>
            跳转到支付宝支付
          </el-button>
        </div>
      </div>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="rechargeDialogVisible = false">取消</el-button>
          <el-button 
            type="primary" 
            @click="createRecharge" 
            :loading="rechargeLoading"
            :disabled="!!rechargeQRCode"
          >
            {{ rechargeQRCode ? '支付中...' : '确认充值' }}
          </el-button>
        </span>
      </template>
    </el-dialog>

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
import { ElMessage, ElMessageBox, ElNotification } from '@/utils/elementPlusServices'
import { Wallet } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'
import UpgradeDevicesDrawer from '@/components/UpgradeDevicesDrawer.vue'
import { userAPI, subscriptionAPI, softwareConfigAPI, rechargeAPI, settingsAPI, checkinAPI, useApi, cachedAPI, pendingPaymentStorage } from '@/utils/api'
import { formatDate as formatDateUtil, getRemainingDays } from '@/utils/date'
import { copyToClipboard as copyText } from '@/utils/textSelection'
import { safeNavigate, safeOpen, safeOpenApp } from '@/utils/safeOpen'
import { sanitizeBasicHtml, sanitizePlainText } from '@/utils/sanitizeHtml'
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
const isMobile = ref(typeof window !== 'undefined' ? window.innerWidth <= 768 : false)
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
  mihomo_windows_url: '',
  clash_verge_windows_url: '',
  sparkle_windows_url: '',
  hiddify_windows_url: '',
  flash_windows_url: '',
  clash_android_url: '',
  v2rayng_url: '',
  hiddify_android_url: '',
  flash_macos_url: '',
  mihomo_macos_url: '',
  clash_verge_macos_url: '',
  sparkle_macos_url: '',
  shadowrocket_url: ''
})
const activePlatform = ref('Windows')
const showQRCode = ref(false)
const showUpgradeDrawer = ref(false)
const platforms = ref([
  {
    name: 'Windows',
    icon: 'fab fa-windows',
    apps: [
      {
        name: 'Clash for Windows',
        version: 'Latest',
        downloadKey: 'clash_windows_url',
        tutorialUrl: '/help#clash-windows'
      },
      {
        name: 'V2rayN',
        version: 'Latest',
        downloadKey: 'v2rayn_url',
        tutorialUrl: '/help#v2rayn',
        githubKey: 'v2rayn'
      },
      {
        name: 'Clash Party',
        version: 'Latest',
        downloadKey: 'mihomo_windows_url',
        tutorialUrl: '/help#clash-party',
        githubKey: 'clash-party'
      },
      {
        name: 'Clash Verge',
        version: 'Latest',
        downloadKey: 'clash_verge_windows_url',
        tutorialUrl: '/help#clash-verge',
        githubKey: 'clash-verge'
      },
      {
        name: 'Hiddify',
        version: 'Latest',
        downloadKey: 'hiddify_windows_url',
        tutorialUrl: '/help#hiddify',
        githubKey: 'hiddify'
      },
      {
        name: 'FlClash',
        version: 'Latest',
        downloadKey: 'flash_windows_url',
        tutorialUrl: '/help#flclash',
        githubKey: 'flclash'
      }
    ]
  },
  {
    name: 'Android',
    icon: 'fab fa-android',
    apps: [
      {
        name: 'Clash Meta',
        version: 'Latest',
        downloadKey: 'clash_android_url',
        tutorialUrl: '/help#clash-meta'
      },
      {
        name: 'V2rayNG',
        version: 'Latest',
        downloadKey: 'v2rayng_url',
        tutorialUrl: '/help#v2rayng',
        githubKey: 'v2rayng'
      },
      {
        name: 'Hiddify',
        version: 'Latest',
        downloadKey: 'hiddify_android_url',
        tutorialUrl: '/help#hiddify',
        githubKey: 'hiddify'
      }
    ]
  },
  {
    name: 'macOS',
    icon: 'fab fa-apple',
    apps: [
      {
        name: 'FlClash',
        version: 'Latest',
        downloadKey: 'flash_macos_url',
        tutorialUrl: '/help#flclash',
        githubKey: 'flclash'
      },
      {
        name: 'Clash Party',
        version: 'Latest',
        downloadKey: 'mihomo_macos_url',
        tutorialUrl: '/help#clash-party',
        githubKey: 'clash-party'
      },
      {
        name: 'Clash Verge',
        version: 'Latest',
        downloadKey: 'clash_verge_macos_url',
        tutorialUrl: '/help#clash-verge',
        githubKey: 'clash-verge'
      }
    ]
  },
  {
    name: 'iOS',
    icon: 'fab fa-apple',
    apps: [
      {
        name: 'Shadowrocket',
        version: 'Latest',
        downloadKey: 'shadowrocket_url',
        tutorialUrl: '/help#shadowrocket'
      }
    ]
  }
])
const qrCodeUrl = computed(() => {
  if (userInfo.value.qrcodeUrl) {
    return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(userInfo.value.qrcodeUrl)}&ecc=M&margin=10`
  } else if (userInfo.value.universalUrl) {
    const subscriptionUrl = userInfo.value.universalUrl
    const encodedUrl = btoa(unescape(encodeURIComponent(subscriptionUrl)))
    let expiryDisplayName = '订阅'
    if (userInfo.value.expiryDate && userInfo.value.expiryDate !== '未设置') {
      try {
        const expireDate = new Date(userInfo.value.expiryDate)
        if (!isNaN(expireDate.getTime())) {
          const year = expireDate.getFullYear()
          const month = String(expireDate.getMonth() + 1).padStart(2, '0')
          const day = String(expireDate.getDate()).padStart(2, '0')
          expiryDisplayName = `到期时间${year}-${month}-${day}`
        }
      } catch (e) {
        expiryDisplayName = '订阅'
      }
    }
    const qrData = `sub://${encodedUrl}#${encodeURIComponent(expiryDisplayName)}`
    return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(qrData)}&ecc=M&margin=10`
  }
  return ''
})
const isDeviceOverlimit = computed(() => {
  const onlineDevices = userInfo.value.online_devices || subscriptionInfo.value.currentDevices || 0
  const deviceLimit = userInfo.value.total_devices || subscriptionInfo.value.maxDevices || 0
  return deviceLimit > 0 && onlineDevices > deviceLimit
})
const dashboardUpgradeSubscription = computed(() => ({
  device_limit: userInfo.value.total_devices || subscriptionInfo.value.maxDevices || 0,
  maxDevices: userInfo.value.total_devices || subscriptionInfo.value.maxDevices || 0,
  expire_time: subscriptionInfo.value.expiryDate || userInfo.value.expire_time,
  expiryDate: subscriptionInfo.value.expiryDate || userInfo.value.expire_time
}))

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
        subscription_status: dashboardData.subscription?.status || dashboardData.subscription_status || 'inactive'
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
          qrcodeUrl: subscriptionData.qrcodeUrl || ''
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
  return method ? method.name : '支付'
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
          const QRCode = await import('qrcode')
          const qrOptions = {
            width: isMobile.value ? 200 : 256,
            margin: 2,
            color: {
              dark: '#000000',
              light: '#FFFFFF'
            },
            errorCorrectionLevel: 'M'
          }
          const qrCodeDataURL = await QRCode.toDataURL(paymentUrl, qrOptions)
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
  if (rechargeStatusInterval) {
    clearInterval(rechargeStatusInterval)
    rechargeStatusInterval = null
  }
  if (rechargeVisibilityHandler) {
    document.removeEventListener('visibilitychange', rechargeVisibilityHandler)
    rechargeVisibilityHandler = null
  }
  if (rechargeFocusHandler) {
    window.removeEventListener('focus', rechargeFocusHandler)
    rechargeFocusHandler = null
  }
  if (rechargeStatusTimeoutId) {
    clearTimeout(rechargeStatusTimeoutId)
    rechargeStatusTimeoutId = null
  }
  cleanupRechargeManualWatchers()
}
const startRechargeStatusCheck = () => {
  cleanupRechargeStatusCheck()
  checkRechargeStatus()
  rechargeStatusInterval = setInterval(async () => {
    await checkRechargeStatus()
  }, 5000)
  rechargeVisibilityHandler = async () => {
    if (document.visibilityState === 'visible' && currentRechargeOrderNo.value) {
      await checkRechargeStatus()
    }
  }
  document.addEventListener('visibilitychange', rechargeVisibilityHandler)
  rechargeFocusHandler = async () => {
    if (currentRechargeOrderNo.value) {
      await checkRechargeStatus()
    }
  }
  window.addEventListener('focus', rechargeFocusHandler)
  rechargeStatusTimeoutId = setTimeout(() => {
    cleanupRechargeStatusCheck()
  }, 30 * 60 * 1000)
}
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
    } else if (rechargeData.status === 'cancelled' || rechargeData.status === 'failed') {
      cleanupRechargeStatusCheck()
      pendingPaymentStorage.clear()
      closeRechargeDialog()
      ElMessage.warning('充值订单已取消或失败')
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
      softwareConfig.value = response.data.data
    }
  } catch (error) {
    }
}
const downloadApp = async (appName) => {
  const clientKeyMap = {
    'clash_windows_url': null, // Clash for Windows 使用配置的链接
    'v2rayn_url': 'v2rayn',
    'mihomo_windows_url': 'clash-party',
    'mihomo_macos_url': 'clash-party',
    'clash_verge_windows_url': 'clash-verge',
    'clash_verge_macos_url': 'clash-verge',
    'sparkle_windows_url': 'sparkle',
    'sparkle_macos_url': 'sparkle',
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
const openTutorial = (url) => {
  if (url) {
    router.push(url)
    return
  }
  router.push('/help')
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
const handleMohomoCommand = (command) => executeCommand(command, {
  'copy-mohomo': copyMohomoSubscription,
  'import-mohomo': importMohomoSubscription
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
const copyMohomoSubscription = () => copySubscriptionUrl(
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
const importMohomoSubscription = () => importClashBasedSubscription('mohomo', '正在打开 Clash Part 客户端...')
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
const getDeviceIcon = (osName) => {
  const iconMap = {
    'Windows': 'fab fa-windows',
    'Android': 'fab fa-android',
    'iOS': 'fab fa-apple',
    'macOS': 'fab fa-apple',
    'Linux': 'fab fa-linux'
  }
  return iconMap[osName] || 'fas fa-mobile-alt'
}
const oneclickImport = (client, url, name = '') => {
  try {
    const clashCompatibleClients = new Set(['clashx', 'clash', 'flash', 'mohomo', 'sparkle', 'clash-verge'])
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
const handleResize = () => {
  if (resizeRafId !== null || typeof window === 'undefined') return
  resizeRafId = window.requestAnimationFrame(() => {
    resizeRafId = null
    isMobile.value = window.innerWidth <= 768
  })
}
onMounted(() => {
  if (typeof window !== 'undefined') {
    isMobile.value = window.innerWidth <= 768
    window.addEventListener('resize', handleResize, { passive: true })
  }

  // 并发加载所有初始化任务，提高首页加载速度
  Promise.all([
    loadUserInfo(),
    loadSoftwareConfig(),
    loadCheckinStatus(),
    checkAndShowAnnouncement() // 移除延迟，直接并发执行
  ]).catch(err => {
    console.error('Dashboard 初始化失败:', err)
  })

  const handleSubscriptionUpdate = async (event) => {
    if (event?.detail?.refreshed) return
    cachedAPI.clearUserCache()
    // 并发更新订阅和用户信息
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
  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', handleResize)
    if (resizeRafId !== null) {
      window.cancelAnimationFrame(resizeRafId)
      resizeRafId = null
    }
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
.welcome-banner {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 16px;
  padding: 40px;
  margin-bottom: 30px;
  color: white;
  position: relative;
  overflow: clip;
}
.welcome-banner::before {
  content: '';
  position: absolute;
  top: -50%;
  right: -50%;
  width: 200%;
  height: 200%;
  background: radial-gradient(circle, rgba(255,255,255,0.1) 0%, transparent 70%);
  animation: float 6s ease-in-out infinite;
}
@keyframes float {
  0%, 100% { transform: translateY(0px) rotate(0deg); }
  50% { transform: translateY(-20px) rotate(180deg); }
}
.banner-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  position: relative;
  z-index: 1;
}
.welcome-title {
  font-size: 2.5rem;
  font-weight: 700;
  margin: 0 0 10px 0;
}
.welcome-subtitle {
  font-size: 1.1rem;
  opacity: 0.9;
  margin: 0;
}
.welcome-icon {
  font-size: 4rem;
  opacity: 0.3;
}
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
  margin-bottom: 30px;
}
.stat-card {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.05);
  border: 1px solid #e5e7eb;
  display: flex;
  align-items: center;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  &.level-card {
    border-width: 2px;
    position: relative;
    overflow: clip;
    padding: 24px;
    &::before {
      content: '';
      position: absolute;
      top: -50%;
      right: -50%;
      width: 200%;
      height: 200%;
      background: radial-gradient(circle, rgba(255, 255, 255, 0.1) 0%, transparent 70%);
      opacity: 0;
      transition: opacity 0.5s ease;
    }
    &:hover::before {
      opacity: 1;
    }
    .level-card-inner {
      display: flex;
      align-items: flex-start;
      gap: 20px;
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
      margin-bottom: 12px;
      flex-wrap: wrap;
      .level-name {
        margin: 0;
        font-size: 2rem;
        font-weight: 800;
        letter-spacing: 1px;
        line-height: 1.2;
      }
      .level-discount-tag {
        flex-shrink: 0;
        transition: all 0.3s ease;
        &:hover {
          transform: scale(1.05);
          box-shadow: 0 6px 20px rgba(64, 158, 255, 0.4) !important;
        }
      }
    }
    .level-expiry {
      font-size: 0.95rem;
      color: #6b7280;
      margin: 0 0 16px 0;
      display: flex;
      align-items: center;
      gap: 6px;
      font-weight: 500;
      :is(i) {
        font-size: 14px;
        opacity: 0.7;
      }
    }
    .level-icon {
      width: 80px;
      height: 80px;
      border-radius: 20px;
      font-size: 32px;
      transition: all 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
      position: relative;
      overflow: clip;
      &::before {
        content: '';
        position: absolute;
        top: -50%;
        left: -50%;
        width: 200%;
        height: 200%;
        background: radial-gradient(circle, rgba(255, 255, 255, 0.3) 0%, transparent 70%);
        opacity: 0;
        transition: opacity 0.3s ease;
      }
      &:hover {
        transform: scale(1.1) rotate(10deg);
        &::before {
          opacity: 1;
          animation: rotate 2s linear infinite;
        }
      }
    }
    @keyframes rotate {
      from { transform: rotate(0deg); }
      to { transform: rotate(360deg); }
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
        width: 100%;
        height: 10px;
        background-color: #f0f0f0;
        border-radius: 5px;
        overflow: clip;
        margin-bottom: 8px;
        .progress-fill {
          height: 100%;
          background: linear-gradient(90deg, #67c23a 0%, #85ce61 100%);
          border-radius: 5px;
          transition: width 0.3s ease;
        }
      }
      .progress-text {
        font-size: 12px;
        color: #666;
        margin: 0 0 4px 0;
        line-height: 1.5;
        :is(i) {
          margin-right: 4px;
          color: #67c23a;
        }
      }
      .progress-tip {
        font-size: 11px;
        color: #909399;
        margin: 0;
        padding: 6px 8px;
        background: #f5f7fa;
        border-radius: 4px;
        line-height: 1.4;
      }
    }
    .max-level-tip {
      margin-top: 16px;
      padding: 14px 20px;
      background: linear-gradient(135deg, #f6d365 0%, #fda085 100%);
      border-radius: 12px;
      color: #fff;
      font-size: 14px;
      font-weight: 600;
      text-align: center;
      box-shadow: 0 4px 16px rgba(253, 160, 133, 0.4);
      position: relative;
      overflow: clip;
      &::before {
        content: '';
        position: absolute;
        top: -50%;
        left: -50%;
        width: 200%;
        height: 200%;
        background: radial-gradient(circle, rgba(255, 255, 255, 0.3) 0%, transparent 70%);
        animation: shimmer 3s ease-in-out infinite;
      }
      :is(i) {
        margin-right: 8px;
        color: #ffd700;
        font-size: 16px;
        filter: drop-shadow(0 2px 4px rgba(255, 215, 0, 0.5));
      }
    }
    @keyframes shimmer {
      0%, 100% { transform: translate(-50%, -50%) rotate(0deg); }
      50% { transform: translate(-50%, -50%) rotate(180deg); }
    }
  }
}
.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
}
.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  font-size: 24px;
  color: white;
}
.stat-card:nth-child(1) .stat-icon { background: linear-gradient(135deg, #667eea, #764ba2); }
.stat-card:nth-child(2) .stat-icon { background: linear-gradient(135deg, #4facfe, #00f2fe); }
.stat-card:nth-child(3) .stat-icon { background: linear-gradient(135deg, #43e97b, #38f9d7); }
.stat-card:nth-child(4) .stat-icon { background: linear-gradient(135deg, #f093fb, #f5576c); }
.stat-title {
  font-size: 1.5rem;
  font-weight: 700;
  margin: 0 0 4px 0;
  color: #1f2937;
}
.stat-subtitle {
  font-size: 0.875rem;
  color: #6b7280;
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
    color: #1f2937;
    transition: color 0.3s ease;
  }
  .device-separator {
    font-size: 1.2rem;
    color: #9ca3af;
    margin: 0 2px;
  }
  .device-limit {
    font-size: 1.5rem;
    font-weight: 700;
    color: #6b7280;
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
    :is(i) {
      font-size: 0.875rem;
    }
  }
  .upgrade-device-btn {
    margin-top: 10px;
    width: 100%;
  }
  &.device-overlimit {
    border-color: #ef4444 !important;
    background: linear-gradient(135deg, #fee2e2 0%, #fecaca 100%) !important;
    box-shadow: 0 4px 12px rgba(239, 68, 68, 0.3) !important;
    animation: blink-border 1s infinite;
  }
  &.device-warning {
    border-color: #f59e0b !important;
    background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%) !important;
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
@keyframes blink-border {
  0%, 100% {
    box-shadow: 0 4px 12px rgba(239, 68, 68, 0.3);
  }
  50% {
    box-shadow: 0 4px 20px rgba(239, 68, 68, 0.6);
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
  .balance-actions .el-button i {
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
    :is(i) {
      margin-right: 4px;
      font-size: 12px;
    }
    @media (max-width: 768px) {
      padding: 6px 12px;
      font-size: 0.75rem;
      margin-left: 0;
      :is(i) {
        margin-right: 3px;
        font-size: 11px;
      }
    }
    @media (max-width: 480px) {
      padding: 8px 16px;
      font-size: 0.8125rem;
      border-radius: 8px;
      :is(i) {
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
    color: #1f2937;
    line-height: 1.3;
    margin: 0;
  }
  .time-unit {
    font-size: 1rem;
    font-weight: 600;
    color: #6b7280;
  }
  .remaining-time-card .stat-subtitle {
    margin: 0;
    font-size: 0.875rem;
    color: #6b7280;
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
    :is(i) {
      margin-right: 4px;
      font-size: 12px;
    }
    @media (max-width: 768px) {
      padding: 6px 12px;
      font-size: 0.75rem;
      margin-left: 0;
      :is(i) {
        margin-right: 3px;
        font-size: 11px;
      }
    }
    @media (max-width: 480px) {
      padding: 8px 16px;
      font-size: 0.8125rem;
      border-radius: 8px;
      :is(i) {
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
      :is(i) {
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
      color: #6b7280;
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
      :is(i) {
        margin-right: 4px;
        font-size: 12px;
      }
    }
  }
}
.recharge-dialog {
  :deep(.el-dialog__body) {
    padding: 20px;
    @media (max-width: 768px) {
      padding: 16px;
    }
  }
  :deep(.el-dialog) {
    @media (max-width: 768px) {
      width: 90% !important;
      margin: 5vh auto !important;
      max-width: 400px;
    }
    @media (max-width: 480px) {
      width: 95% !important;
      margin: 2vh auto !important;
    }
  }
  :deep(.el-dialog__header) {
    @media (max-width: 768px) {
      padding: 16px 16px 12px;
    }
  }
  :deep(.el-dialog__title) {
    @media (max-width: 768px) {
      font-size: 18px;
    }
  }
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
  .amount-tips {
    margin-top: 12px;
    font-size: 12px;
    color: #909399;
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
        background: white;
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
      color: #909399;
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
    }
  }
  :deep(.el-dialog__footer) {
    padding: 16px 20px;
    border-top: 1px solid #e5e7eb;
    @media (max-width: 768px) {
      padding: 12px 16px;
      display: flex;
      gap: 10px;
    }
    .dialog-footer {
      display: flex;
      justify-content: flex-end;
      gap: 10px;
      width: 100%;
      @media (max-width: 768px) {
        flex-direction: row;
        gap: 10px;
      }
    }
    .el-button {
      @media (max-width: 768px) {
        flex: 1;
        margin: 0;
        padding: 10px 16px;
        font-size: 14px;
        border-radius: 6px;
      }
      @media (max-width: 480px) {
        padding: 12px 16px;
        font-size: 15px;
      }
    }
  }
}
.main-content {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 30px;
}
.card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.05);
  border: 1px solid #e5e7eb;
  margin-bottom: 20px;
}
.card-header {
  padding: 20px 24px 0;
}
.card-title {
  font-size: 1.25rem;
  font-weight: 600;
  margin: 0;
  color: #1f2937;
  display: flex;
  align-items: center;
  gap: 8px;
}
.card-body {
  padding: 20px 24px 24px;
}
.tutorial-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}
.tutorial-tab {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 0.875rem;
  font-weight: 500;
}
.tutorial-tab:hover {
  border-color: #3b82f6;
  background-color: #f8fafc;
}
.tutorial-tab.active {
  border-color: #3b82f6;
  background-color: #3b82f6;
  color: white;
}
.tutorial-app {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  margin-bottom: 12px;
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
  color: #1f2937;
}
.app-version {
  font-size: 0.875rem;
  color: #6b7280;
  margin: 0;
}
.app-actions {
  display: flex;
  gap: 8px;
}
.subscription-buttons {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 20px;
  @media (max-width: 768px) {
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    margin-bottom: 16px;
  }
  @media (max-width: 480px) {
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }
}
.subscription-group {
  display: flex;
  @media (max-width: 768px) {
    width: 100%;
  }
}
.clash-btn {
  background: linear-gradient(135deg, #667eea, #764ba2);
  border: none;
  width: 100%;
}
.shadowrocket-btn {
  background: linear-gradient(135deg, #f093fb, #f5576c);
  border: none;
  width: 100%;
}
.v2ray-btn {
  background: linear-gradient(135deg, #4facfe, #00f2fe);
  border: none;
  width: 100%;
}
.universal-btn {
  background: linear-gradient(135deg, #43e97b, #38f9d7);
  border: none;
  width: 100%;
}
.qr-code-section {
  text-align: center;
  padding-top: 20px;
  border-top: 1px solid #e5e7eb;
}
.qr-code-container {
  margin-top: 16px;
}
.software-category {
  margin-bottom: 24px;
}
.category-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #2c3e50;
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 2px solid #f0f0f0;
}
.category-title :is(i) {
  color: #667eea;
}
.subscription-urls-section {
  margin-bottom: 24px;
}
.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #2c3e50;
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 2px solid #f0f0f0;
}
.section-title :is(i) {
  color: #667eea;
}
.url-display {
  display: flex;
  flex-direction: column;
  gap: 16px;
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
  height: 28px !important;
  padding: 4px 6px !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
  gap: 3px !important;
  flex-shrink: 0;
  border-radius: 4px;
  background-color: #ffffff !important;
  border: 1px solid #dcdfe6 !important;
  color: #000000 !important;
  transition: all 0.2s ease;
  font-size: 11px !important;
  white-space: nowrap;
  overflow: clip;
  box-sizing: border-box;
  &:hover {
    background-color: #f5f7fa !important;
    border-color: #c0c4cc !important;
    color: #000000 !important;
  }
  &:active {
    background-color: #ebedf0 !important;
  }
  :is(i) {
    font-size: 11px !important;
    color: #000000 !important;
    flex-shrink: 0;
  }
  :is(span) {
    font-size: 11px !important;
    color: #000000 !important;
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
  padding: 20px;
  background: #f8f9fa;
  border-radius: 12px;
  border: 2px dashed #e0e0e0;
}
.qr-code {
  width: 200px;
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
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
.qr-placeholder :is(i) {
  font-size: 48px;
}
.qr-tip {
  font-size: 14px;
  color: #666;
  text-align: center;
  margin: 0;
}
.flash-btn {
  background: linear-gradient(135deg, #ff6b6b, #ee5a24);
  border: none;
  width: 100%;
  border-radius: 12px;
  padding: 14px 20px;
  font-weight: 600;
  transition: all 0.3s ease;
  @media (max-width: 768px) {
    padding: 16px 20px;
    font-size: 15px;
    border-radius: 16px;
    box-shadow: 0 4px 12px rgba(255, 107, 107, 0.3);
    &:active {
      transform: scale(0.98);
    }
  }
}
.mohomo-btn {
  background: linear-gradient(135deg, #4834d4, #686de0);
  border: none;
  width: 100%;
  border-radius: 12px;
  padding: 14px 20px;
  font-weight: 600;
  transition: all 0.3s ease;
  @media (max-width: 768px) {
    padding: 16px 20px;
    font-size: 15px;
    border-radius: 16px;
    box-shadow: 0 4px 12px rgba(72, 52, 212, 0.3);
    &:active {
      transform: scale(0.98);
    }
  }
}
.clash-verge-btn {
  background: linear-gradient(135deg, #feca57, #ff9ff3);
  border: none;
  width: 100%;
  border-radius: 12px;
  padding: 14px 20px;
  font-weight: 600;
  transition: all 0.3s ease;
  @media (max-width: 768px) {
    padding: 16px 20px;
    font-size: 15px;
    border-radius: 16px;
    box-shadow: 0 4px 12px rgba(254, 202, 87, 0.3);
    &:active {
      transform: scale(0.98);
    }
  }
}
.hiddify-btn {
  background: linear-gradient(135deg, #a8edea, #fed6e3);
  border: none;
  width: 100%;
  color: #333;
  border-radius: 12px;
  padding: 14px 20px;
  font-weight: 600;
  transition: all 0.3s ease;
  @media (max-width: 768px) {
    padding: 16px 20px;
    font-size: 15px;
    border-radius: 16px;
    box-shadow: 0 4px 12px rgba(168, 237, 234, 0.3);
    &:active {
      transform: scale(0.98);
    }
  }
}
.qr-code img {
  width: 200px;
  height: 200px;
  border-radius: 8px;
}
.qr-tip {
  font-size: 0.875rem;
  color: #6b7280;
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
  border: 1px solid #e5e7eb;
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
  background: linear-gradient(135deg, #667eea, #764ba2);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 18px;
}
.device-name {
  font-size: 1rem;
  font-weight: 600;
  margin: 0 0 4px 0;
  color: #1f2937;
}
.device-os, .device-ip {
  font-size: 0.875rem;
  color: #6b7280;
  margin: 0;
}
.no-devices {
  text-align: center;
  padding: 40px 20px;
  color: #9ca3af;
}
.no-devices :is(i) {
  font-size: 3rem;
  margin-bottom: 16px;
  display: block;
}
@media (max-width: 768px) {
  .dashboard-container {
    padding: 0;
  }
  .welcome-banner {
    margin: 0 -12px 12px -12px;
    border-radius: 0;
    padding: 16px 12px;
    .banner-content {
      flex-direction: column;
      text-align: center;
      gap: 8px;
      .welcome-text {
        .welcome-title {
          font-size: 1.25rem;
          margin-bottom: 4px;
        }
        .welcome-subtitle {
          font-size: 0.8125rem;
        }
      }
      .welcome-icon {
        font-size: 1.5rem;
        opacity: 0.2;
      }
    }
  }
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 10px;
    margin-bottom: 16px;
    @media (max-width: 480px) {
      grid-template-columns: 1fr;
      gap: 12px;
    }
    &.level-card::before,
    &.max-level-tip::before,
    .level-icon::before {
      animation: none !important;
      display: none;
    }
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
          color: #6b7280;
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
        border-radius: 12px;
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
  .main-content {
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
        :is(i) {
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
      :is(i) {
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
      border-radius: 16px;
      font-weight: 600;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
      transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
      white-space: nowrap;
      overflow: clip;
      text-overflow: ellipsis;
      &:active {
        transform: scale(0.98);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      }
      :is(i) {
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
      height: 28px !important;
      padding: 4px 6px !important;
      font-size: 11px !important;
      flex-shrink: 0 !important;
      gap: 3px !important;
      :is(i) {
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
  .welcome-title {
    font-size: 1.25rem;
  }
  .welcome-subtitle {
    font-size: 0.8125rem;
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
      border-radius: 14px;
      :is(i) {
        font-size: 12px;
        margin-right: 3px;
      }
    }
  }
  .url-input-wrapper {
    gap: 6px !important;
    .copy-btn {
      min-width: 46px !important;
      max-width: 46px !important;
      height: 28px !important;
      padding: 4px 5px !important;
      font-size: 10px !important;
      gap: 2px !important;
      :is(i) {
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
</style>
