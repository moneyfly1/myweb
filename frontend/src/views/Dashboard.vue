<template>
  <div class="list-container dashboard-container">
    <!-- 欢迎横幅 -->
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

    <!-- 统计卡片 -->
    <div class="stats-grid">
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

      <!-- 设备使用卡片已删除 -->

      <div class="stat-card balance-card">
        <div class="stat-icon">
          <i class="fas fa-wallet"></i>
        </div>
        <div class="stat-content">
          <div class="balance-main">
            <h3 class="stat-title">¥ {{ userInfo.balance || '0.00' }}</h3>
            <p class="stat-subtitle">账户余额</p>
          </div>
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

    <!-- 主要内容区域 -->
    <div class="main-content">
      <!-- 左侧内容 -->
      <div class="left-content">
        <!-- 订阅地址卡片 -->
        <div class="card subscription-card">
          <div class="card-header">
            <h3 class="card-title">
              <i class="fas fa-link"></i>
              订阅地址
            </h3>
          </div>
          <div class="card-body">
            <!-- Clash系列软件 -->
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
                  <el-dropdown @command="handleSparkleCommand" trigger="click">
                    <el-button type="primary" class="sparkle-btn">
                      <i class="fas fa-sparkles"></i>
                      Sparkle
                      <i class="fas fa-chevron-down"></i>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="copy-sparkle">复制订阅</el-dropdown-item>
                        <el-dropdown-item command="import-sparkle">一键导入</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </div>
            </div>

            <!-- V2Ray系列软件 -->
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

            <!-- Shadowrocket -->
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

            <!-- 订阅地址显示区域 -->
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
                <div class="url-item">
                  <label>通用订阅地址</label>
                  <div class="url-input-wrapper">
                    <el-input 
                      :value="userInfo.universalUrl" 
                      readonly 
                      size="small"
                      class="url-input"
                    />
                    <el-button 
                      @click="copyUniversalSubscription" 
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

            <!-- 二维码区域 -->
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

      <!-- 右侧内容 -->
      <div class="right-content">
        <!-- 使用教程卡片 -->
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


    <!-- 充值对话框 -->
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
            :min="20"
            :step="1"
            :precision="2"
            placeholder="请输入充值金额"
            style="width: 100%"
            :controls-position="isMobile ? 'right' : 'right'"
          >
            <template #prepend>¥</template>
          </el-input-number>
          <div class="amount-tips">
            <p>最低充值金额20元，可自定义金额</p>
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
      </el-form>
      
      <!-- 支付二维码 -->
      <div v-if="rechargeQRCode" class="recharge-qr-section">
        <h4>请使用支付宝扫描二维码完成支付</h4>
        <div class="qr-code-wrapper">
          <img :src="rechargeQRCode" alt="支付二维码" class="qr-code-img" />
        </div>
        <p class="qr-tip">支付完成后，余额将自动到账</p>
        
        <!-- 手机端跳转按钮 -->
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
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { Wallet } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'
import { userAPI, subscriptionAPI, softwareConfigAPI, rechargeAPI, settingsAPI } from '@/utils/api'
import { formatDate as formatDateUtil, getRemainingDays } from '@/utils/date'
import DOMPurify from 'dompurify'

const router = useRouter()

// HTML内容清理函数，防止XSS攻击
const sanitizeHtml = (html) => {
  if (!html) return ''
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'b', 'i', 'u', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li', 'a', 'div', 'span', 'blockquote', 'pre', 'code'],
    ALLOWED_ATTR: ['href', 'target', 'style', 'class', 'id'],
    ALLOW_DATA_ATTR: false
  })
}

// 响应式数据
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


// 充值相关
const rechargeDialogVisible = ref(false)
const rechargeForm = ref({
  amount: 20
})
const rechargeRules = {
  amount: [
    { required: true, message: '请输入充值金额', trigger: 'blur' },
    { type: 'number', min: 20, message: '充值金额不能少于20元', trigger: 'blur' }
  ]
}
const rechargeFormRef = ref()
const rechargeLoading = ref(false)
const rechargeQRCode = ref('')
const rechargePaymentUrl = ref('') // 保存支付URL，用于跳转支付宝App
const isMobile = ref(window.innerWidth <= 768)
const quickAmounts = [20, 50, 100, 200, 500, 1000]
const softwareConfig = ref({
  // Windows软件
  clash_windows_url: '',
  v2rayn_url: '',
  mihomo_windows_url: '',
  sparkle_windows_url: '',
  hiddify_windows_url: '',
  flash_windows_url: '',
  
  // Android软件
  clash_android_url: '',
  v2rayng_url: '',
  hiddify_android_url: '',
  
  // macOS软件
  flash_macos_url: '',
  mihomo_macos_url: '',
  sparkle_macos_url: '',
  
  // iOS软件
  shadowrocket_url: ''
})
const activePlatform = ref('Windows')
const showQRCode = ref(false)

// 平台配置
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
        name: 'Sparkle',
        version: 'Latest',
        downloadKey: 'sparkle_windows_url',
        tutorialUrl: '/help#sparkle',
        githubKey: 'sparkle'
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
        name: 'Sparkle',
        version: 'Latest',
        downloadKey: 'sparkle_macos_url',
        tutorialUrl: '/help#sparkle',
        githubKey: 'sparkle'
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

// 计算属性
const qrCodeUrl = computed(() => {
  if (userInfo.value.qrcodeUrl) {
    // 使用后台提供的二维码URL
    return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(userInfo.value.qrcodeUrl)}&ecc=M&margin=10`
  } else if (userInfo.value.universalUrl) {
    // 降级方案：使用通用订阅地址生成二维码
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

// 计算设备是否超过限制
const isDeviceOverlimit = computed(() => {
  const onlineDevices = userInfo.value.online_devices || subscriptionInfo.value.currentDevices || 0
  const deviceLimit = userInfo.value.total_devices || subscriptionInfo.value.maxDevices || 0
  return deviceLimit > 0 && onlineDevices > deviceLimit
})

const isDeviceWarning = computed(() => {
  const onlineDevices = userInfo.value.online_devices || subscriptionInfo.value.currentDevices || 0
  const deviceLimit = userInfo.value.total_devices || subscriptionInfo.value.maxDevices || 0
  return deviceLimit > 0 && onlineDevices >= deviceLimit * 0.8 && onlineDevices <= deviceLimit
})

// 方法
const formatDate = (dateString) => {
  if (!dateString) return '未知'
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN')
}

const loadUserInfo = async () => {
  try {
    const dashboardResponse = await userAPI.getUserInfo()
    if (dashboardResponse.data && dashboardResponse.data.success) {
      const dashboardData = dashboardResponse.data.data
      userInfo.value = {
        ...dashboardData,
        balance: dashboardData.balance || '0.00', // 确保余额字段被正确设置
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
    } else {
      throw new Error('用户信息加载失败')
    }
  } catch (error) {
    // 降级方案：尝试从订阅API获取订阅地址
    try {
      const subscriptionResponse = await subscriptionAPI.getUserSubscription()
      if (subscriptionResponse.data && subscriptionResponse.data.success) {
        const subscriptionData = subscriptionResponse.data.data
        // 设置基本的用户信息
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
          // 使用订阅API的地址
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
  }
}

// 获取订阅信息
const loadSubscriptionInfo = async () => {
  try {
    const response = await subscriptionAPI.getUserSubscription()
    if (response.data && response.data.success) {
      subscriptionInfo.value = response.data.data
      } else {
      // 用户可能没有订阅，设置默认值
      subscriptionInfo.value = {
        currentDevices: 0,
        maxDevices: 0,
        remainingDays: 0,
        expiryDate: '未设置',
        status: 'inactive'
      }
    }
  } catch (error) {
    // 用户可能没有订阅，设置默认值
    subscriptionInfo.value = {
      currentDevices: 0,
      maxDevices: 0,
      remainingDays: 0,
      expiryDate: '未设置',
      status: 'inactive'
    }
  }
}


// 充值相关方法
const showRechargeDialog = () => {
  rechargeDialogVisible.value = true
  rechargeForm.value.amount = 20
  rechargeQRCode.value = ''
  rechargePaymentUrl.value = ''
  currentRechargeOrderNo.value = null
  // 清除之前的定时器
  if (rechargeStatusInterval) {
    clearInterval(rechargeStatusInterval)
    rechargeStatusInterval = null
  }
}

// 跳转到支付宝App进行充值支付（参考购买套餐的方式）
const openAlipayAppForRecharge = () => {
  if (!rechargePaymentUrl.value) {
    ElMessage.error('支付链接不存在')
    return
  }
  
  // 生成支付宝App跳转链接
  // 支付宝App的URL Scheme格式：alipays://platformapi/startapp?saId=10000007&qrcode=支付URL
  const alipayAppUrl = `alipays://platformapi/startapp?saId=10000007&qrcode=${encodeURIComponent(rechargePaymentUrl.value)}`
  
  try {
    // 添加页面可见性监听，当用户从支付宝返回时立即检查支付状态
    const handleVisibilityChange = async () => {
      if (document.visibilityState === 'visible' && rechargeDialogVisible.value) {
        // 用户返回页面，立即检查支付状态
        await checkRechargeStatus()
        // 移除监听器
        document.removeEventListener('visibilitychange', handleVisibilityChange)
      }
    }
    document.addEventListener('visibilitychange', handleVisibilityChange)
    
    // 添加页面焦点监听，当用户切换回页面时检查支付状态
    const handleFocus = async () => {
      if (rechargeDialogVisible.value) {
        await checkRechargeStatus()
        window.removeEventListener('focus', handleFocus)
      }
    }
    window.addEventListener('focus', handleFocus)
    
    // 尝试打开支付宝App
    window.location.href = alipayAppUrl
    
    // 如果3秒后还在当前页面，说明可能没有安装支付宝App，提示用户
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

const createRecharge = async () => {
  try {
    await rechargeFormRef.value.validate()
    
    if (rechargeForm.value.amount < 20) {
      ElMessage.error('充值金额不能少于20元')
      return
    }
    
    rechargeLoading.value = true
    
    const response = await rechargeAPI.createRecharge(rechargeForm.value.amount, 'alipay')
    
    if (response.data && response.data.success !== false) {
      const data = response.data.data
      
      // 检查是否有支付错误
      if (data.payment_error) {
        ElMessage.warning(data.payment_error || '支付链接生成失败')
        return
      }
      
      // 获取支付URL（后端返回的是 payment_url）
      const paymentUrl = data.payment_url || data.payment_qr_code
      
      if (!paymentUrl) {
        ElMessage.error('支付链接生成失败，请稍后重试')
        return
      }
      
      // 验证充值订单信息是否存在
      const rechargeId = data.id || data.recharge_id
      const rechargeOrderNo = data.order_no
      if (!rechargeId || !rechargeOrderNo) {
        console.error('充值订单信息不完整:', data)
        ElMessage.error('充值订单创建失败，订单信息缺失')
        return
      }
      
      // 保存支付URL，用于跳转支付宝App
      rechargePaymentUrl.value = paymentUrl
      
      // 保存充值订单号，用于状态检查
      currentRechargeOrderNo.value = rechargeOrderNo
      
      // 使用qrcode库将支付URL生成为二维码图片（与订单支付相同的方式）
      try {
        const QRCode = await import('qrcode')
        // 根据设备类型调整二维码参数
        const qrOptions = {
          width: isMobile.value ? 200 : 256, // 手机端使用较小的尺寸
          margin: 2,
          color: {
            dark: '#000000',
            light: '#FFFFFF'
          },
          errorCorrectionLevel: 'M' // 使用中等纠错级别，避免二维码过于复杂
        }
        // 将支付URL生成为base64格式的二维码图片
        const qrCodeDataURL = await QRCode.toDataURL(paymentUrl, qrOptions)
        rechargeQRCode.value = qrCodeDataURL
        ElMessage.success('充值订单创建成功，请扫描二维码完成支付')
        
        // 开始检查支付状态（使用订单号，参考购买套餐的方式）
        startRechargeStatusCheck()
      } catch (qrError) {
        // 如果二维码生成失败，直接使用URL
        rechargeQRCode.value = paymentUrl
        ElMessage.success('充值订单创建成功，请扫描二维码完成支付')
        // 开始检查支付状态（使用订单号，参考购买套餐的方式）
        startRechargeStatusCheck()
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
const currentRechargeOrderNo = ref(null)

// 开始检查充值支付状态（参考购买套餐的方式）
const startRechargeStatusCheck = () => {
  // 清除之前的检查
  if (rechargeStatusInterval) {
    clearInterval(rechargeStatusInterval)
    rechargeStatusInterval = null
  }
  
  // 立即检查一次支付状态
  checkRechargeStatus()
  
  // 每2秒检查一次支付状态（提高检查频率，与购买套餐一致）
  rechargeStatusInterval = setInterval(async () => {
    await checkRechargeStatus()
  }, 2000)
  
  // 添加页面可见性监听，当用户从其他应用返回时立即检查
  const handleVisibilityChange = async () => {
    if (document.visibilityState === 'visible' && rechargeDialogVisible.value) {
      // 用户返回页面，立即检查支付状态
      await checkRechargeStatus()
    }
  }
  document.addEventListener('visibilitychange', handleVisibilityChange)
  
  // 添加页面焦点监听
  const handleFocus = async () => {
    if (rechargeDialogVisible.value) {
      await checkRechargeStatus()
    }
  }
  window.addEventListener('focus', handleFocus)
  
  // 30分钟后停止检查
  setTimeout(() => {
    if (rechargeStatusInterval) {
      clearInterval(rechargeStatusInterval)
      rechargeStatusInterval = null
    }
    document.removeEventListener('visibilitychange', handleVisibilityChange)
    window.removeEventListener('focus', handleFocus)
  }, 30 * 60 * 1000)
}

// 关闭充值对话框
const closeRechargeDialog = () => {
  // 清除支付状态检查定时器
  if (rechargeStatusInterval) {
    clearInterval(rechargeStatusInterval)
    rechargeStatusInterval = null
  }
  rechargeDialogVisible.value = false
  rechargeQRCode.value = ''
  rechargePaymentUrl.value = ''
  currentRechargeOrderNo.value = null
}

// 检查充值支付状态（使用订单号，支持主动查询支付状态）
const checkRechargeStatus = async () => {
  if (!currentRechargeOrderNo.value) {
    return
  }
  
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
      // 支付成功
      if (rechargeStatusInterval) {
        clearInterval(rechargeStatusInterval)
        rechargeStatusInterval = null
      }
      
      ElMessage.success('充值成功！余额已到账')
      closeRechargeDialog()
      
      // 刷新用户信息，确保余额更新
      await loadUserInfo()
      // 延迟再次刷新，确保余额显示正确（防止缓存问题）
      setTimeout(async () => {
        await loadUserInfo()
      }, 500)
    } else if (rechargeData.status === 'cancelled' || rechargeData.status === 'failed') {
      // 充值已取消或失败
      if (rechargeStatusInterval) {
        clearInterval(rechargeStatusInterval)
        rechargeStatusInterval = null
      }
      
      closeRechargeDialog()
      ElMessage.warning('充值订单已取消或失败')
    }
  } catch (error) {
    // 如果是 404 错误，说明订单不存在，停止轮询
    if (error.response?.status === 404) {
      console.warn('充值订单不存在，停止检查支付状态')
      if (rechargeStatusInterval) {
        clearInterval(rechargeStatusInterval)
        rechargeStatusInterval = null
      }
    } else {
      // 其他错误只记录，不停止轮询
      console.warn('检查充值状态失败:', error)
    }
  }
}


const loadSoftwareConfig = async () => {
  try {
    const response = await softwareConfigAPI.getSoftwareConfig()
    if (response.data && response.data.success) {
      // 后端返回的是ResponseBase格式，数据在response.data.data中
      softwareConfig.value = response.data.data
    }
  } catch (error) {
    }
}

const downloadApp = async (appName) => {
  // 客户端映射到 GitHub 仓库标识
  const clientKeyMap = {
    'clash_windows_url': null, // Clash for Windows 使用配置的链接
    'v2rayn_url': 'v2rayn',
    'mihomo_windows_url': 'clash-party',
    'mihomo_macos_url': 'clash-party',
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
  
  // 如果配置中有链接，优先使用配置的链接
  const configUrl = softwareConfig.value[appName]
  if (configUrl) {
    window.open(configUrl, '_blank')
    return
  }
  
  // 如果是 Shadowrocket，使用 App Store 链接
  if (appName === 'shadowrocket_url') {
    window.open('https://apps.apple.com/app/shadowrocket/id932747118', '_blank')
    return
  }
  
  // 如果有 GitHub 仓库，使用自动获取
  if (clientKey) {
    try {
      ElMessage.info('正在获取最新下载链接...')
      const { getClientDownloadUrl, getClientReleasesUrl } = await import('@/utils/githubDownload')
      const downloadUrl = await getClientDownloadUrl(clientKey)
      window.open(downloadUrl, '_blank')
      ElMessage.success('已打开下载页面')
    } catch (error) {
      console.error('获取下载链接失败:', error)
      // 备用：打开 releases 页面
      try {
        const { getClientReleasesUrl } = await import('@/utils/githubDownload')
        const releasesUrl = getClientReleasesUrl(clientKey)
        if (releasesUrl) {
          window.open(releasesUrl, '_blank')
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
  // 跳转到软件教程页面
  router.push('/help')
}

// 跳转到套餐页面
const goToPackages = () => {
  router.push('/packages')
}

const loadDevices = async () => {
  try {
    const response = await userAPI.getUserDevices()
    devices.value = response.data
  } catch (error) {
    }
}

const handleClashCommand = (command) => {
  if (command === 'copy-clash') {
    copyClashSubscription()
  } else if (command === 'import-clash') {
    importClashSubscription()
  }
}

const handleFlashCommand = (command) => {
  if (command === 'copy-flash') {
    copyFlashSubscription()
  } else if (command === 'import-flash') {
    importFlashSubscription()
  }
}

const handleMohomoCommand = (command) => {
  if (command === 'copy-mohomo') {
    copyMohomoSubscription()
  } else if (command === 'import-mohomo') {
    importMohomoSubscription()
  }
}

const handleSparkleCommand = (command) => {
  if (command === 'copy-sparkle') {
    copySparkleSubscription()
  } else if (command === 'import-sparkle') {
    importSparkleSubscription()
  }
}

const handleShadowrocketCommand = (command) => {
  if (command === 'copy-shadowrocket') {
    copyShadowrocketSubscription()
  } else if (command === 'import-shadowrocket') {
    importShadowrocketSubscription()
  }
}

const copyClashSubscription = () => {
  if (!userInfo.value.clashUrl) {
    ElMessage.error('Clash 订阅地址不可用，请刷新页面重试')
    return
  }
  
  try {
    copyToClipboard(userInfo.value.clashUrl, 'Clash 订阅地址已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败，请手动复制订阅地址')
  }
}

const copyShadowrocketSubscription = () => {
  if (!userInfo.value.universalUrl) {
    ElMessage.error('通用订阅地址不可用，请刷新页面重试')
    return
  }
  
  try {
    copyToClipboard(userInfo.value.universalUrl, '通用订阅地址已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败，请手动复制订阅地址')
  }
}

const copyUniversalSubscription = () => {
  if (!userInfo.value.universalUrl) {
    ElMessage.error('通用订阅地址不可用')
    return
  }
  
  copyToClipboard(userInfo.value.universalUrl, '通用订阅地址已复制到剪贴板')
}


// Flash相关方法
const copyFlashSubscription = () => {
  if (!userInfo.value.clashUrl) {
    ElMessage.error('Flash 订阅地址不可用，请刷新页面重试')
    return
  }
  
  try {
    copyToClipboard(userInfo.value.clashUrl, 'Flash 订阅地址已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败，请手动复制订阅地址')
  }
}

const importFlashSubscription = () => {
  if (!userInfo.value.clashUrl) {
    ElMessage.error('Flash 订阅地址不可用，请刷新页面重试')
    return
  }
  
  try {
    let url = userInfo.value.clashUrl
    let name = '' // 用于clash://install-config的name参数
    
    if (userInfo.value.expiryDate && userInfo.value.expiryDate !== '未设置') {
      // 格式化到期时间用于name参数，参考格式：到期时间YYYY-MM-DD_到期
      const expiryDate = new Date(userInfo.value.expiryDate)
      const year = expiryDate.getFullYear()
      const month = String(expiryDate.getMonth() + 1).padStart(2, '0')
      const day = String(expiryDate.getDate()).padStart(2, '0')
      name = `到期时间${year}-${month}-${day}_到期`
    }
    
    oneclickImport('flash', url, name)
    ElMessage.success('正在打开 Flash 客户端...')
  } catch (error) {
    ElMessage.error('一键导入失败，请手动复制订阅地址')
  }
}

// Clash Part相关方法
const copyMohomoSubscription = () => {
  if (!userInfo.value.clashUrl) {
    ElMessage.error('Clash Part 订阅地址不可用，请刷新页面重试')
    return
  }
  
  try {
    copyToClipboard(userInfo.value.clashUrl, 'Clash Part 订阅地址已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败，请手动复制订阅地址')
  }
}

const importMohomoSubscription = () => {
  if (!userInfo.value.clashUrl) {
    ElMessage.error('Clash Part 订阅地址不可用，请刷新页面重试')
    return
  }
  
  try {
    let url = userInfo.value.clashUrl
    let name = '' // 用于clash://install-config的name参数
    
    if (userInfo.value.expiryDate && userInfo.value.expiryDate !== '未设置') {
      // 格式化到期时间用于name参数，参考格式：到期时间YYYY-MM-DD_到期
      const expiryDate = new Date(userInfo.value.expiryDate)
      const year = expiryDate.getFullYear()
      const month = String(expiryDate.getMonth() + 1).padStart(2, '0')
      const day = String(expiryDate.getDate()).padStart(2, '0')
      name = `到期时间${year}-${month}-${day}_到期`
    }
    
    oneclickImport('mohomo', url, name)
    ElMessage.success('正在打开 Clash Part 客户端...')
  } catch (error) {
    ElMessage.error('一键导入失败，请手动复制订阅地址')
  }
}

// Sparkle相关方法
const copySparkleSubscription = () => {
  if (!userInfo.value.clashUrl) {
    ElMessage.error('Sparkle 订阅地址不可用，请刷新页面重试')
    return
  }
  
  try {
    copyToClipboard(userInfo.value.clashUrl, 'Sparkle 订阅地址已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败，请手动复制订阅地址')
  }
}

const importSparkleSubscription = () => {
  if (!userInfo.value.clashUrl) {
    ElMessage.error('Sparkle 订阅地址不可用，请刷新页面重试')
    return
  }
  
  try {
    let url = userInfo.value.clashUrl
    let name = '' // 用于clash://install-config的name参数
    
    if (userInfo.value.expiryDate && userInfo.value.expiryDate !== '未设置') {
      // 格式化到期时间用于name参数，参考格式：到期时间YYYY-MM-DD_到期
      const expiryDate = new Date(userInfo.value.expiryDate)
      const year = expiryDate.getFullYear()
      const month = String(expiryDate.getMonth() + 1).padStart(2, '0')
      const day = String(expiryDate.getDate()).padStart(2, '0')
      name = `到期时间${year}-${month}-${day}_到期`
    }
    
    oneclickImport('sparkle', url, name)
    ElMessage.success('正在打开 Sparkle 客户端...')
  } catch (error) {
    ElMessage.error('一键导入失败，请手动复制订阅地址')
  }
}

// Hiddify Next相关方法
const copyHiddifySubscription = () => {
  if (!userInfo.value.universalUrl) {
    ElMessage.error('通用订阅地址不可用')
    return
  }
  
  copyToClipboard(userInfo.value.universalUrl, '通用订阅地址已复制到剪贴板')
}

const copyToClipboard = async (text, message) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(message)
  } catch (error) {
    // 降级方案
    const textArea = document.createElement('textarea')
    textArea.value = text
    document.body.appendChild(textArea)
    textArea.select()
    document.execCommand('copy')
    document.body.removeChild(textArea)
    ElMessage.success(message)
  }
}

const importClashSubscription = () => {
  if (!userInfo.value.clashUrl) {
    ElMessage.error('Clash 订阅地址不可用，请刷新页面重试')
    return
  }
  
  try {
    let url = userInfo.value.clashUrl
    let name = '' // 用于clash://install-config的name参数
    
    if (userInfo.value.expiryDate && userInfo.value.expiryDate !== '未设置') {
      // 格式化到期时间用于name参数，参考格式：到期时间YYYY-MM-DD_到期
      const expiryDate = new Date(userInfo.value.expiryDate)
      const year = expiryDate.getFullYear()
      const month = String(expiryDate.getMonth() + 1).padStart(2, '0')
      const day = String(expiryDate.getDate()).padStart(2, '0')
      name = `到期时间${year}-${month}-${day}_到期`
    }
    
    oneclickImport('clashx', url, name)
    ElMessage.success('正在打开 Clash 客户端...')
  } catch (error) {
    ElMessage.error('一键导入失败，请手动复制订阅地址')
  }
}

const importShadowrocketSubscription = () => {
  if (!userInfo.value.universalUrl) {
    ElMessage.error('通用订阅地址不可用，请刷新页面重试')
    return
  }
  
  try {
    let url = userInfo.value.universalUrl
    let expiryName = ''
    
    if (userInfo.value.expiryDate && userInfo.value.expiryDate !== '未设置') {
      // 格式化到期时间作为订阅名称：到期时间 YYYY-MM-DD
      const expiryDate = new Date(userInfo.value.expiryDate)
      const year = expiryDate.getFullYear()
      const month = String(expiryDate.getMonth() + 1).padStart(2, '0')
      const day = String(expiryDate.getDate()).padStart(2, '0')
      expiryName = `到期时间${year}-${month}-${day}`
    }
    
    oneclickImport('shadowrocket', url, expiryName)
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

// 一键导入功能实现（参考原有实现）
const oneclickImport = (client, url, name = '') => {
  try {
    switch (client) {
      case 'clashx':
      case 'clash':
        // Clash for Windows/macOS/Android
        // 参考格式：clash://install-config?url=...&name=到期时间_到期
        if (name) {
          window.open(`clash://install-config?url=${encodeURIComponent(url)}&name=${encodeURIComponent(name)}`, '_blank')
        } else {
          window.open(`clash://install-config?url=${encodeURIComponent(url)}`, '_blank')
        }
        break
      case 'flash':
        // Flash (Clash系列)
        if (name) {
          window.open(`clash://install-config?url=${encodeURIComponent(url)}&name=${encodeURIComponent(name)}`, '_blank')
        } else {
          window.open(`clash://install-config?url=${encodeURIComponent(url)}`, '_blank')
        }
        break
      case 'mohomo':
        // Clash Part (Clash系列)
        if (name) {
          window.open(`clash://install-config?url=${encodeURIComponent(url)}&name=${encodeURIComponent(name)}`, '_blank')
        } else {
          window.open(`clash://install-config?url=${encodeURIComponent(url)}`, '_blank')
        }
        break
      case 'sparkle':
        // Sparkle (Clash系列)
        if (name) {
          window.open(`clash://install-config?url=${encodeURIComponent(url)}&name=${encodeURIComponent(name)}`, '_blank')
        } else {
          window.open(`clash://install-config?url=${encodeURIComponent(url)}`, '_blank')
        }
        break
      case 'shadowrocket':
        // Shadowrocket (iOS)
        // Shadowrocket URL 格式: shadowrocket://add/sub://<base64_url>#<name>
        // name 部分会显示为订阅名称，可以包含有效期信息
        let shadowrocketUrl = `shadowrocket://add/sub://${btoa(url)}`
        if (name) {
          // 如果有名称（有效期），添加到 URL 的 hash 部分
          shadowrocketUrl += `#${encodeURIComponent(name)}`
        }
        window.open(shadowrocketUrl, '_blank')
        break
      case 'ssr':
        // SSR客户端
        window.open(`ssr://${btoa(url)}`, '_blank')
        break
      case 'quantumult':
        // Quantumult
        window.open(`quantumult://resource?url=${encodeURIComponent(url)}`, '_blank')
        break
      case 'quantumult_v2':
        // Quantumult X
        window.open(`quantumult-x://resource?url=${encodeURIComponent(url)}`, '_blank')
        break
      case 'v2rayng':
        // V2rayNG
        window.open(`v2rayng://install-config?url=${encodeURIComponent(url)}`, '_blank')
        break
      case 'hiddify':
        // Hiddify Next (Android)
        window.open(`hiddify://install-config?url=${encodeURIComponent(url)}`, '_blank')
        break
      default:
        // 尝试通用方式
        window.open(url, '_blank')
    }
  } catch (error) {
    ElMessage.error('一键导入失败，请手动复制订阅地址')
  }
}

// 检查并显示公告
const checkAndShowAnnouncement = async () => {
  try {
    const response = await settingsAPI.getPublicSettings()
    const settings = response.data?.data || response.data || {}
    
    // 处理布尔值（可能是字符串 "true"/"false" 或布尔值）
    const isEnabled = settings.announcement_enabled === true || 
                      settings.announcement_enabled === 'true' || 
                      String(settings.announcement_enabled).toLowerCase() === 'true'
    
    // 每次登录都显示公告（如果启用），除非用户手动关闭
    // 不记录到 localStorage，这样每次登录都会显示
    if (isEnabled && settings.announcement_content && String(settings.announcement_content).trim()) {
      const content = String(settings.announcement_content).trim()
      const sanitizedContent = sanitizeHtml(content)
      const displayContent = sanitizedContent || content || '暂无公告内容'
      
      // 使用 ElNotification 在右下角显示公告
      // 每次登录都会显示，用户需要手动点击关闭按钮才会关闭
      ElNotification({
        title: '系统公告',
        message: displayContent,
        type: 'info',
        position: 'bottom-right',
        duration: 0, // 不自动关闭，需要用户手动关闭
        dangerouslyUseHTMLString: true,
        showClose: true // 显示关闭按钮
      })
    }
  } catch (error) {
    // 静默失败，不影响页面加载
    console.warn('获取公告失败:', error)
  }
}

// 生命周期
// 监听窗口大小变化
const handleResize = () => {
  if (typeof window !== 'undefined') {
    isMobile.value = window.innerWidth <= 768
  }
}

onMounted(() => {
  // 初始化窗口大小
  if (typeof window !== 'undefined') {
    isMobile.value = window.innerWidth <= 768
    window.addEventListener('resize', handleResize)
  }
  loadUserInfo()
  loadSubscriptionInfo()
  loadSoftwareConfig()
  // 延迟一下再检查公告，确保页面已经渲染完成
  setTimeout(() => {
    checkAndShowAnnouncement()
  }, 500)
})

onUnmounted(() => {
  if (rechargeStatusInterval) {
    clearInterval(rechargeStatusInterval)
    rechargeStatusInterval = null
  }
  // 移除窗口大小监听
  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', handleResize)
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

/* 欢迎横幅 */
.welcome-banner {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 16px;
  padding: 40px;
  margin-bottom: 30px;
  color: white;
  position: relative;
  overflow: hidden;
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

/* 统计卡片 */
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
    overflow: hidden;
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
      overflow: hidden;
      
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
        overflow: hidden;
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
      overflow: hidden;
      
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

/* 设备卡片样式 */
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

/* 余额卡片样式 */
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

/* 剩余时间卡片样式 */
.remaining-time-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  overflow: hidden;
  
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
    overflow: hidden;
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

/* 充值对话框样式 */
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

/* 主要内容区域 */
.main-content {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 30px;
}

/* 卡片通用样式 */
.card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.05);
  border: 1px solid #e5e7eb;
  margin-bottom: 20px;
}

.card-header {
  padding: 20px 24px 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
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


/* 教程卡片 */
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

/* 订阅卡片 */
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

/* 软件分类标题 */
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

/* 订阅地址显示区域 */
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

/* 复制按钮样式 */
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
  overflow: hidden;
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

/* 二维码区域 */
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

/* 新按钮样式 */
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

.sparkle-btn {
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

/* 设备卡片 */
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


/* 响应式设计 */
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
    
    /* 移动端禁用不必要的装饰动画以省电 */
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
    
    /* 等级卡片在移动端的优化 */
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
    
    /* 余额卡片在移动端的优化 */
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
        padding: 6px 12px;
        font-size: 0.75rem;
        flex-shrink: 0;
        white-space: nowrap;
      }
    }
    
    /* 设备卡片在移动端的优化 */
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
    
    /* 剩余时间卡片在移动端的特殊处理 */
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
    /* 优化：移动端改为横向滚动，避免换行占用过多纵向空间 */
    display: flex;
    flex-wrap: nowrap;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    padding-bottom: 4px; /* 预留滚动条空间 */
    
    /* 隐藏滚动条 */
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
      overflow: hidden;
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
  
  /* 等级卡片在小屏幕的优化 */
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
  
  /* 余额卡片在小屏幕的优化 */
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
  
  /* 设备卡片在小屏幕的优化 */
  .device-card {
    .device-count-wrapper {
      .device-count,
      .device-limit {
        font-size: 1.5rem;
      }
    }
  }
  
  /* 剩余时间卡片在小屏幕的优化 */
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
