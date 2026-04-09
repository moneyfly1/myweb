<template>
  <div class="unified-auth-container">
    <transition name="fade">
      <el-alert
        v-if="notification.show"
        :title="notification.message"
        :type="notification.type === 'success' ? 'success' : 'error'"
        :closable="true"
        @close="notification.show = false"
        class="notification-alert"
        show-icon
      />
    </transition>
    <div class="auth-wrapper">
      <!-- 左侧品牌展示区 - 现代化极光 + 网格毛玻璃特效 -->
      <div class="auth-brand-section">
        <div class="glow-orb orb-1"></div>
        <div class="glow-orb orb-2"></div>
        <div class="glow-orb orb-3"></div>
        <div class="glass-grid"></div>

        <div class="brand-content">
          <div class="brand-body">
            <div class="brand-hero stagger-brand">
              <h2 class="brand-title">安全 · 极速 · 稳定</h2>
              <p class="brand-subtitle">您的全球网络加速专家</p>
            </div>
            <div class="brand-features stagger-brand">
              <div class="feature-card">
                <div class="feature-icon">
                  <i class="ph-fill ph-globe-hemisphere-west"></i>
                </div>
                <div class="feature-text">
                  <span class="feature-title">80+ 全球节点</span>
                  <span class="feature-desc">覆盖主流地区，智能选路</span>
                </div>
              </div>
              <div class="feature-card">
                <div class="feature-icon">
                  <i class="ph-fill ph-lightning"></i>
                </div>
                <div class="feature-text">
                  <span class="feature-title">10Gbps 骨干带宽</span>
                  <span class="feature-desc">4K/8K 流媒体秒开</span>
                </div>
              </div>
              <div class="feature-card">
                <div class="feature-icon">
                  <i class="ph-fill ph-shield-check"></i>
                </div>
                <div class="feature-text">
                  <span class="feature-title">IEPL 专线</span>
                  <span class="feature-desc">企业级链路，稳定不掉线</span>
                </div>
              </div>
            </div>
            <div class="brand-platforms stagger-brand">
              <span class="platform-label">全平台支持</span>
              <div class="platform-icons">
                <i class="ph-fill ph-windows-logo"></i>
                <i class="ph-fill ph-apple-logo"></i>
                <i class="ph-fill ph-android-logo"></i>
                <i class="ph-fill ph-linux-logo"></i>
              </div>
            </div>
          </div>
          <div class="brand-footer stagger-brand">
            <span>&copy; {{ new Date().getFullYear() }} {{ settings.siteName || 'TurboCloud' }}</span>
            <div class="footer-links">
              <a href="#" class="footer-link">服务条款</a>
              <a href="#" class="footer-link">隐私政策</a>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧表单区 -->
      <div class="auth-form-section">
        <div class="mobile-header">
          <div class="mobile-logo">
            <img v-if="settings.siteLogo" :src="settings.siteLogo" :alt="settings.siteName" class="logo-img-small" />
            <div v-else class="logo-placeholder-small">
              <i class="ph-bold ph-rocket-launch"></i>
            </div>
            <span class="mobile-brand-name">{{ settings.siteName || 'TurboCloud' }}</span>
          </div>
          <a href="#" class="download-app-btn">
            <i class="ph-bold ph-download-simple"></i>
            下载APP
          </a>
        </div>

        <div class="form-container">
          <transition name="slide-fade" mode="out-in">
            
            <!-- ====== 登录视图 ====== -->
            <div v-if="currentView === 'login'" key="login" class="auth-form">
              <div class="stagger-item">
                <AnimatedCharacter
                  :username-length="loginForm.username.length"
                  :is-password-focused="isPasswordFocused"
                  :is-password-visible="isPasswordVisible"
                />
              </div>
              <div class="form-header stagger-item">
                <h1 class="form-title">登录控制台</h1>
                <p class="form-subtitle">欢迎回来，连接从未如此简单。</p>
              </div>
              <el-form
                ref="loginFormRef"
                :model="loginForm"
                :rules="loginRules"
                @submit.prevent="handleLogin"
                label-position="top"
              >
                <el-form-item prop="username" class="stagger-item">
                  <el-input
                    v-model="loginForm.username"
                    placeholder="用户名或邮箱"
                    size="large"
                    prefix-icon="User"
                    clearable
                    autocomplete="username"
                  />
                </el-form-item>
                <el-form-item prop="password" class="stagger-item">
                  <el-input
                    v-model="loginForm.password"
                    :type="isPasswordVisible ? 'text' : 'password'"
                    placeholder="输入密码"
                    size="large"
                    prefix-icon="Lock"
                    clearable
                    autocomplete="current-password"
                    @keyup.enter="handleLogin"
                    @focus="isPasswordFocused = true"
                    @blur="isPasswordFocused = false"
                  >
                    <template #suffix>
                      <el-icon style="cursor:pointer" @click="isPasswordVisible = !isPasswordVisible">
                        <View v-if="!isPasswordVisible" />
                        <Hide v-else />
                      </el-icon>
                    </template>
                  </el-input>
                </el-form-item>
                <el-form-item class="stagger-item">
                  <div class="form-options">
                    <el-checkbox v-model="rememberMe">记住我</el-checkbox>
                    <el-link type="primary" @click="switchView('forgot')" :underline="false">
                      忘记密码？
                    </el-link>
                  </div>
                </el-form-item>
                <el-form-item class="stagger-item">
                  <el-button
                    type="primary"
                    size="large"
                    :loading="isLoading"
                    @click="handleLogin"
                    class="submit-button"
                  >
                    {{ isLoading ? '登录中...' : '立即连接' }}
                  </el-button>
                </el-form-item>
              </el-form>
              <div class="form-footer stagger-item">
                <span class="footer-text">新用户？</span>
                <el-link type="primary" @click="switchView('register')" :underline="false">
                  注册免费试用
                </el-link>
              </div>
            </div>

            <!-- ====== 注册视图 ====== -->
            <div v-else-if="currentView === 'register'" key="register" class="auth-form">
              <div class="stagger-item">
                <!-- 统一使用可爱的交互组件 -->
                <AnimatedCharacter
                  :username-length="registerForm.username.length"
                  :is-password-focused="isRegPasswordFocused"
                  :is-password-visible="isRegPasswordVisible"
                />
              </div>
              <div class="form-header stagger-item">
                <h1 class="form-title">注册账号</h1>
                <p class="form-subtitle">加入我们，体验极致网络。</p>
              </div>
              <el-alert
                v-if="!registrationEnabled"
                title="注册功能已禁用"
                type="warning"
                :closable="false"
                show-icon
                style="margin-bottom: 20px;"
                class="stagger-item"
              >
                <template #default>
                  <p>系统管理员已关闭用户注册功能，请联系管理员获取账户。</p>
                </template>
              </el-alert>
              <el-form
                v-if="registrationEnabled"
                ref="registerFormRef"
                :model="registerForm"
                :rules="registerRules"
                @submit.prevent="handleRegister"
                label-position="top"
              >
                <el-form-item prop="username" class="stagger-item">
                  <el-input
                    v-model="registerForm.username"
                    placeholder="用户名"
                    size="large"
                    prefix-icon="User"
                    clearable
                    autocomplete="username"
                  />
                </el-form-item>
                <el-form-item prop="email" class="stagger-item">
                  <el-input
                    v-model="registerForm.email"
                    type="email"
                    placeholder="推荐使用 QQ 邮箱"
                    size="large"
                    prefix-icon="Message"
                    clearable
                    autocomplete="email"
                  />
                  <div class="email-tip" style="margin-top: 8px; font-size: 12px; color: #909399;">
                    <i class="ph-fill ph-info"></i> 推荐使用 QQ 邮箱注册，接收验证码更稳定
                  </div>
                </el-form-item>
                <el-form-item prop="verificationCode" :required="emailVerificationRequired" class="stagger-item">
                  <div class="verification-code-group">
                    <el-input
                      v-model="registerForm.verificationCode"
                      :placeholder="emailVerificationRequired ? '6位验证码（必填）' : '6位验证码（选填）'"
                      size="large"
                      prefix-icon="Message"
                      maxlength="6"
                      class="verification-code-input"
                      clearable
                      autocomplete="off"
                    />
                    <el-button
                      type="primary"
                      size="large"
                      :disabled="codeTimer > 0 || !registerForm.email || sendingCode"
                      :loading="sendingCode"
                      @click="sendVerificationCode('register')"
                      class="send-code-button"
                    >
                      {{ codeTimer > 0 ? `${codeTimer}s 后重发` : '获取验证码' }}
                    </el-button>
                  </div>
                </el-form-item>
                <el-form-item prop="inviteCode" :required="inviteCodeRequired" class="stagger-item">
                  <el-input
                    v-model="registerForm.inviteCode"
                    :placeholder="inviteCodeRequired ? '邀请码（必填）' : '邀请码（选填，填写可获得注册奖励）'"
                    size="large"
                    prefix-icon="Ticket"
                    clearable
                  />
                  <div v-if="inviteCodeInfo" class="invite-code-tip">
                    <span v-if="inviteCodeInfo.is_valid || inviteCodeInfo.success" class="tip-success">
                      ✓ 邀请码有效，注册后可获得 {{ inviteCodeInfo.invitee_reward || inviteCodeInfo.data?.invitee_reward || 0 }} 元奖励
                    </span>
                    <span v-else class="tip-error">
                      ✗ {{ inviteCodeInfo.message }}
                    </span>
                  </div>
                </el-form-item>
                <el-form-item prop="password" class="stagger-item">
                  <el-input
                    v-model="registerForm.password"
                    :type="isRegPasswordVisible ? 'text' : 'password'"
                    placeholder="8位以上字符"
                    size="large"
                    prefix-icon="Lock"
                    clearable
                    autocomplete="new-password"
                    @focus="isRegPasswordFocused = true"
                    @blur="isRegPasswordFocused = false"
                  >
                    <template #suffix>
                      <el-icon style="cursor:pointer" @click="isRegPasswordVisible = !isRegPasswordVisible">
                        <View v-if="!isRegPasswordVisible" />
                        <Hide v-else />
                      </el-icon>
                    </template>
                  </el-input>
                </el-form-item>
                <el-form-item prop="confirmPassword" class="stagger-item">
                  <el-input
                    v-model="registerForm.confirmPassword"
                    :type="isRegPasswordVisible ? 'text' : 'password'"
                    placeholder="确认密码"
                    size="large"
                    prefix-icon="Lock"
                    clearable
                    autocomplete="new-password"
                    @keyup.enter="handleRegister"
                    @focus="isRegPasswordFocused = true"
                    @blur="isRegPasswordFocused = false"
                  >
                    <template #suffix>
                      <el-icon style="cursor:pointer" @click="isRegPasswordVisible = !isRegPasswordVisible">
                        <View v-if="!isRegPasswordVisible" />
                        <Hide v-else />
                      </el-icon>
                    </template>
                  </el-input>
                </el-form-item>
                <el-form-item class="stagger-item">
                  <el-button
                    type="primary"
                    size="large"
                    :loading="isLoading"
                    @click="handleRegister"
                    class="submit-button"
                  >
                    {{ isLoading ? '注册中...' : '立即注册' }}
                  </el-button>
                </el-form-item>
              </el-form>
              <div class="form-footer stagger-item">
                <span class="footer-text">已有账号？</span>
                <el-link type="primary" @click="switchView('login')" :underline="false">
                  立即登录
                </el-link>
                <span class="footer-text" style="margin-left: 12px;">忘记密码？</span>
                <el-link type="primary" @click="switchView('forgot')" :underline="false">
                  找回密码
                </el-link>
              </div>
            </div>

            <!-- ====== 忘记密码视图 ====== -->
            <div v-else-if="currentView === 'forgot'" key="forgot" class="auth-form">
              <el-button
                text
                @click="switchView('login')"
                class="back-button stagger-item"
              >
                <i class="ph-bold ph-arrow-left"></i>
                返回登录
              </el-button>
              
              <!-- 统一使用可爱的交互组件代替原先的锁形 Icon -->
              <div class="stagger-item">
                <AnimatedCharacter
                  :username-length="forgotForm.email.length"
                  :is-password-focused="isForgotPasswordFocused"
                  :is-password-visible="isForgotPasswordVisible"
                />
              </div>
              
              <div class="form-header stagger-item">
                <h1 class="form-title">重置密码</h1>
                <p class="form-subtitle">验证邮箱后即可设置新密码。</p>
              </div>
              <el-form
                ref="forgotFormRef"
                :model="forgotForm"
                :rules="forgotRules"
                @submit.prevent="handleReset"
                label-position="top"
              >
                <el-form-item prop="email" class="stagger-item">
                  <el-input
                    v-model="forgotForm.email"
                    type="email"
                    placeholder="name@company.com"
                    size="large"
                    prefix-icon="Message"
                    clearable
                    autocomplete="email"
                  />
                </el-form-item>
                <el-form-item prop="verificationCode" required class="stagger-item">
                  <div class="verification-code-group">
                    <el-input
                      v-model="forgotForm.verificationCode"
                      placeholder="6位验证码（必填）"
                      size="large"
                      prefix-icon="Message"
                      maxlength="6"
                      class="verification-code-input"
                      clearable
                      autocomplete="off"
                    />
                    <el-button
                      type="primary"
                      size="large"
                      :disabled="codeTimer > 0 || !forgotForm.email || sendingCode"
                      :loading="sendingCode"
                      @click="sendVerificationCode('forgot')"
                      class="send-code-button"
                    >
                      {{ codeTimer > 0 ? `${codeTimer}s 后重发` : '获取验证码' }}
                    </el-button>
                  </div>
                </el-form-item>
                <el-form-item prop="newPassword" class="stagger-item">
                  <el-input
                    v-model="forgotForm.newPassword"
                    :type="isForgotPasswordVisible ? 'text' : 'password'"
                    placeholder="输入新密码"
                    size="large"
                    prefix-icon="Lock"
                    clearable
                    autocomplete="new-password"
                    @focus="isForgotPasswordFocused = true"
                    @blur="isForgotPasswordFocused = false"
                  >
                    <template #suffix>
                      <el-icon style="cursor:pointer" @click="isForgotPasswordVisible = !isForgotPasswordVisible">
                        <View v-if="!isForgotPasswordVisible" />
                        <Hide v-else />
                      </el-icon>
                    </template>
                  </el-input>
                </el-form-item>
                <el-form-item prop="confirmPassword" class="stagger-item">
                  <el-input
                    v-model="forgotForm.confirmPassword"
                    :type="isForgotPasswordVisible ? 'text' : 'password'"
                    placeholder="确认新密码"
                    size="large"
                    prefix-icon="Lock"
                    clearable
                    autocomplete="new-password"
                    @keyup.enter="handleReset"
                    @focus="isForgotPasswordFocused = true"
                    @blur="isForgotPasswordFocused = false"
                  >
                    <template #suffix>
                      <el-icon style="cursor:pointer" @click="isForgotPasswordVisible = !isForgotPasswordVisible">
                        <View v-if="!isForgotPasswordVisible" />
                        <Hide v-else />
                      </el-icon>
                    </template>
                  </el-input>
                </el-form-item>
                <el-form-item class="stagger-item">
                  <el-button
                    type="primary"
                    size="large"
                    :loading="isLoading"
                    @click="handleReset"
                    class="submit-button"
                  >
                    {{ isLoading ? '重置中...' : '确认重置' }}
                  </el-button>
                </el-form-item>
              </el-form>
            </div>
          </transition>
        </div>
        <div class="mobile-footer">
          <span>&copy; 2026 {{ settings.siteName || 'TurboCloud' }} Network.</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import AnimatedCharacter from '@/components/AnimatedCharacter.vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/store/auth'
import { useSettingsStore } from '@/store/settings'
import { authAPI, inviteAPI, settingsAPI } from '@/utils/api'
import { useThemeStore } from '@/store/theme'
import { secureStorage } from '@/utils/api'
import { resetRefreshFailed } from '@/utils/api'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()

const currentView = ref('login')
const isLoading = ref(false)
const sendingCode = ref(false)
const codeTimer = ref(0)
const rememberMe = ref(false)

// Login States
const isPasswordFocused = ref(false)
const isPasswordVisible = ref(false)

// Register States
const isRegPasswordFocused = ref(false)
const isRegPasswordVisible = ref(false)

// Forgot Password States
const isForgotPasswordFocused = ref(false)
const isForgotPasswordVisible = ref(false)

const registrationEnabled = ref(true)
const inviteCodeRequired = ref(false)
const emailVerificationRequired = ref(true)
const minPasswordLength = ref(8)
const inviteCodeInfo = ref(null)

let countdownTimer = null

const loginFormRef = ref()
const registerFormRef = ref()
const forgotFormRef = ref()

const loginForm = reactive({
  username: '',
  password: ''
})

const registerForm = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  verificationCode: '',
  inviteCode: ''
})

const forgotForm = reactive({
  email: '',
  verificationCode: '',
  newPassword: '',
  confirmPassword: ''
})

const notification = reactive({
  show: false,
  message: '',
  type: 'success'
})

const settings = computed(() => settingsStore)

const showNotification = (message, type = 'success') => {
  notification.message = message
  notification.type = type
  notification.show = true
  setTimeout(() => {
    notification.show = false
  }, 3000)
}

const switchView = (view) => {
  currentView.value = view
  if (view === 'login') {
    loginForm.password = ''
    isPasswordVisible.value = false
  } else if (view === 'register') {
    registerForm.password = ''
    registerForm.confirmPassword = ''
    registerForm.verificationCode = ''
    isRegPasswordVisible.value = false
  } else if (view === 'forgot') {
    forgotForm.newPassword = ''
    forgotForm.confirmPassword = ''
    forgotForm.verificationCode = ''
    isForgotPasswordVisible.value = false
  }
  notification.show = false
}

const loginRules = {
  username: [
    { required: true, message: '请输入用户名或邮箱', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
  ]
}

const registerRules = computed(() => ({
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 20, message: '用户名长度必须在 2 到 20 个字符之间', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_]+$/, message: '用户名只能包含字母、数字和下划线', trigger: 'blur' }
  ],
  email: [
    { required: true, message: '请输入邮箱地址', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: minPasswordLength.value, max: 50, message: `密码长度至少 ${minPasswordLength.value} 位，最多 50 位`, trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (!value) {
          callback()
          return
        }
        const hasLetter = /[A-Za-z]/.test(value)
        const hasDigit = /\d/.test(value)
        if (!hasLetter || !hasDigit) {
          callback(new Error('密码必须包含字母和数字'))
          return
        }
        let complexityCount = 0
        if (/[a-z]/.test(value)) complexityCount++
        if (/[A-Z]/.test(value)) complexityCount++
        if (/\d/.test(value)) complexityCount++
        if (/[!@#$%^&*()_+\-=\[\]{}|;:,.<>?]/.test(value)) complexityCount++
        if (complexityCount < 3) {
          callback(new Error('密码强度不足，建议包含大小写字母、数字和特殊字符'))
          return
        }
        callback()
      },
      trigger: 'blur'
    }
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (value !== registerForm.password) {
          callback(new Error('两次输入密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ],
  verificationCode: emailVerificationRequired.value ? [
    { required: true, message: '请输入验证码', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (!value) {
          callback(new Error('请输入验证码'))
          return
        }
        if (value.length !== 6) {
          callback(new Error('验证码必须为6位数字'))
          return
        }
        if (!/^\d{6}$/.test(value)) {
          callback(new Error('验证码只能包含数字'))
          return
        }
        callback()
      },
      trigger: 'blur'
    }
  ] : [],
  inviteCode: inviteCodeRequired.value ? [
    { required: true, message: '请输入邀请码', trigger: 'blur' }
  ] : []
}))

const forgotRules = computed(() => ({
  email: [
    { required: true, message: '请输入邮箱地址', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
  ],
  verificationCode: [
    { required: true, message: '请输入验证码', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (!value) {
          callback(new Error('请输入验证码'))
          return
        }
        if (value.length !== 6) {
          callback(new Error('验证码必须为6位数字'))
          return
        }
        if (!/^\d{6}$/.test(value)) {
          callback(new Error('验证码只能包含数字'))
          return
        }
        callback()
      },
      trigger: 'blur'
    }
  ],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: minPasswordLength.value, max: 50, message: `密码长度至少 ${minPasswordLength.value} 位，最多 50 位`, trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (!value) {
          callback()
          return
        }
        const hasLetter = /[A-Za-z]/.test(value)
        const hasDigit = /\d/.test(value)
        if (!hasLetter || !hasDigit) {
          callback(new Error('密码必须包含字母和数字'))
          return
        }
        let complexityCount = 0
        if (/[a-z]/.test(value)) complexityCount++
        if (/[A-Z]/.test(value)) complexityCount++
        if (/\d/.test(value)) complexityCount++
        if (/[!@#$%^&*()_+\-=\[\]{}|;:,.<>?]/.test(value)) complexityCount++
        if (complexityCount < 3) {
          callback(new Error('密码强度不足，建议包含大小写字母、数字和特殊字符'))
          return
        }
        callback()
      },
      trigger: 'blur'
    }
  ],
  confirmPassword: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (value !== forgotForm.newPassword) {
          callback(new Error('两次输入密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}))

const sendVerificationCode = async (type) => {
  const email = type === 'register' ? registerForm.email : forgotForm.email
  if (!email) {
    ElMessage.warning('请先输入邮箱地址')
    return
  }
  if (!email.includes('@')) {
    ElMessage.warning('请输入有效的邮箱地址')
    return
  }
  sendingCode.value = true
  try {
    if (type === 'register') {
      await authAPI.sendVerificationCode({
        email: email,
        type: 'email'
      })
    } else {
      await authAPI.forgotPassword({ email: email })
    }
    ElMessage.success('验证码已发送，请查收邮箱')
    codeTimer.value = 60
    if (countdownTimer) {
      clearInterval(countdownTimer)
    }
    countdownTimer = setInterval(() => {
      codeTimer.value--
      if (codeTimer.value <= 0) {
        clearInterval(countdownTimer)
        countdownTimer = null
      }
    }, 1000)
  } catch (error) {
    if (error.response?.data?.detail) {
      ElMessage.error(error.response.data.detail)
    } else if (error.response?.data?.message) {
      ElMessage.error(error.response.data.message)
    } else {
      ElMessage.error('发送验证码失败，请重试')
    }
  } finally {
    sendingCode.value = false
  }
}

const handleLogin = async () => {
  if (!loginFormRef.value) return
  try {
    await loginFormRef.value.validate()
  } catch (error) {
    return
  }
  isLoading.value = true
  try {
    const result = await authStore.login({
      username: loginForm.username,
      password: loginForm.password
    })
    if (result.success) {
      ElMessage.success('登录成功')
      await router.push(result.isAdmin ? '/admin/dashboard' : '/dashboard')
    } else {
      ElMessage.error(result.message || '登录失败，请重试')
    }
  } catch (error) {
    let errorMessage = error.response?.data?.detail || 
                       error.response?.data?.message || 
                       error.message || 
                       '登录失败，请重试'
    if (error.response?.status === 403) {
      if (errorMessage.includes('账户已被禁用') || errorMessage.includes('账号已禁用')) {
        ElMessage({
          message: '账户已被禁用，无法使用服务。如有疑问，请联系管理员。',
          type: 'error',
          duration: 5000,
          showClose: true
        })
      } else if (errorMessage.includes('CSRF') || errorMessage.includes('csrf')) {
        ElMessage({
          message: '安全验证失败，请刷新页面后重试',
          type: 'error',
          duration: 5000,
          showClose: true
        })
        setTimeout(() => {
          window.location.reload()
        }, 2000)
      } else {
        ElMessage({
          message: errorMessage || '访问被拒绝，请刷新页面后重试',
          type: 'error',
          duration: 5000,
          showClose: true
        })
      }
    } else if (error.response?.status === 429) {
      if (errorMessage.includes('锁定') || errorMessage.includes('锁定15分钟')) {
        errorMessage = '登录失败次数过多，账户已被临时锁定15分钟，请稍后再试'
        ElMessage({
          message: errorMessage,
          type: 'error',
          duration: 5000,
          showClose: true
        })
      } else {
        errorMessage = '登录失败次数过多，请稍后再试'
        ElMessage.error(errorMessage)
      }
    } else if (errorMessage.includes('账户已被禁用') || errorMessage.includes('账号已禁用')) {
      ElMessage({
        message: '账户已被禁用，无法使用服务。如有疑问，请联系管理员。',
        type: 'error',
        duration: 5000,
        showClose: true
      })
    } else if (errorMessage.includes('系统维护')) {
      ElMessage({
        message: '系统维护中，请稍后再试',
        type: 'warning',
        duration: 5000,
        showClose: true
      })
    } else {
      ElMessage.error(errorMessage)
    }
  } finally {
    isLoading.value = false
  }
}

const handleRegister = async () => {
  if (!registerFormRef.value) return
  try {
    await registerFormRef.value.validate()
  } catch (error) {
    return
  }
  if (registerForm.password !== registerForm.confirmPassword) {
    ElMessage.error('两次输入的密码不一致，请重新输入')
    return
  }
  isLoading.value = true
  try {
    const registerData = {
      email: registerForm.email,
      username: registerForm.username,
      password: registerForm.password,
      invite_code: registerForm.inviteCode || null
    }
    if (emailVerificationRequired.value && registerForm.verificationCode) {
      registerData.verification_code = registerForm.verificationCode
    }
    const response = await authAPI.register(registerData)
    if (response.data) {
      const responseData = response.data?.data || response.data
      if (responseData.access_token && responseData.user) {
        const { access_token, refresh_token, user: userData } = responseData
        if (userData.is_admin) {
          ElMessage.warning('管理员账户请使用管理员登录页面登录')
          router.push('/login')
          return
        }
        const REFRESH_TOKEN_TTL = 30 * 24 * 60 * 60 * 1000
        const safeUserData = {
          id: userData.id,
          username: userData.username,
          email: userData.email,
          is_admin: userData.is_admin || false,
          is_verified: userData.is_verified,
          is_active: userData.is_active,
          theme: userData.theme,
          language: userData.language
        }
        authStore.setAuth(access_token, safeUserData, false)
        if (refresh_token) {
          secureStorage.set('user_refresh_token', refresh_token, false, REFRESH_TOKEN_TTL)
        }
        resetRefreshFailed()
        setTimeout(() => {
          const themeStore = useThemeStore()
          themeStore.loadUserTheme().catch(() => {})
        }, 500)
        ElMessage.success('注册成功！正在跳转到用户中心...')
        setTimeout(() => {
          router.push('/dashboard')
        }, 500)
      } else {
        ElMessage.success('注册成功！请登录')
        setTimeout(() => {
          switchView('login')
          loginForm.username = registerForm.username
        }, 1500)
      }
    } else {
      ElMessage.error('注册失败，请重试')
    }
  } catch (error) {
    const errorMessage = error.response?.data?.detail || 
                         error.response?.data?.message || 
                         error.message || 
                         '注册失败，请重试'
    ElMessage.error(errorMessage)
  } finally {
    isLoading.value = false
  }
}

const handleReset = async () => {
  if (!forgotFormRef.value) return
  try {
    await forgotFormRef.value.validate()
  } catch (error) {
    return
  }
  if (forgotForm.newPassword !== forgotForm.confirmPassword) {
    ElMessage.error('两次输入的密码不一致，请重新输入')
    return
  }
  isLoading.value = true
  try {
    const { api } = await import('@/utils/api')
    await api.post('/auth/reset-password', {
      email: forgotForm.email,
      verification_code: forgotForm.verificationCode,
      new_password: forgotForm.newPassword
    })
    ElMessage.success('密码重置成功！')
    setTimeout(() => {
      switchView('login')
    }, 1500)
  } catch (error) {
    const errorMessage = error.response?.data?.detail || 
                         error.response?.data?.message || 
                         error.message || 
                         '重置密码失败，请重试'
    ElMessage.error(errorMessage)
  } finally {
    isLoading.value = false
  }
}

const validateInviteCode = async (code) => {
  if (!code || code.trim() === '') {
    inviteCodeInfo.value = null
    return
  }
  try {
    const response = await inviteAPI.validateInviteCode(code.trim().toUpperCase())
    inviteCodeInfo.value = response.data || response
  } catch (error) {
    inviteCodeInfo.value = {
      success: false,
      message: error.response?.data?.message || '邀请码验证失败'
    }
  }
}

let validateTimeout = null
watch(() => registerForm.inviteCode, (newCode) => {
  if (validateTimeout) {
    clearTimeout(validateTimeout)
  }
  if (newCode && newCode.trim()) {
    validateTimeout = setTimeout(() => {
      validateInviteCode(newCode)
    }, 500)
  } else {
    inviteCodeInfo.value = null
  }
})

const checkRegistrationSettings = async () => {
  try {
    const response = await settingsAPI.getPublicSettings()
    const settings = response.data?.data || response.data || {}
    const registrationValue = settings.registration_enabled !== undefined 
                            ? settings.registration_enabled
                            : (settings.allowRegistration !== undefined 
                               ? settings.allowRegistration 
                               : true)
    registrationEnabled.value = registrationValue === true || registrationValue === "true"
    const inviteCodeValue = settings.invite_code_required !== undefined 
                           ? settings.invite_code_required
                           : (settings.inviteCodeRequired !== undefined 
                              ? settings.inviteCodeRequired 
                              : false)
    inviteCodeRequired.value = inviteCodeValue === true || inviteCodeValue === "true"
    const emailVerificationValue = settings.email_verification_required !== undefined 
                                   ? settings.email_verification_required
                                   : (settings.emailVerificationRequired !== undefined 
                                      ? settings.emailVerificationRequired 
                                      : (settings.require_email_verification !== undefined 
                                         ? settings.require_email_verification 
                                         : true))
    emailVerificationRequired.value = emailVerificationValue === true || emailVerificationValue === "true"
    const minPasswordValue = settings.min_password_length !== undefined 
                            ? settings.min_password_length
                            : (settings.minPasswordLength !== undefined 
                               ? settings.minPasswordLength 
                               : 8)
    minPasswordLength.value = typeof minPasswordValue === 'number' ? minPasswordValue : (parseInt(minPasswordValue) || 8)
    if (!registrationEnabled.value) {
      ElMessage.warning('注册功能已禁用，请联系管理员')
    }
  } catch (error) {
    registrationEnabled.value = true
    inviteCodeRequired.value = false
    emailVerificationRequired.value = true
    minPasswordLength.value = 8
  }
}

onMounted(async () => {
  // 并发加载注册设置和应用设置，提高登录页加载速度
  await Promise.all([
    checkRegistrationSettings(),
    settingsStore.loadSettings()
  ])
  if (route.query.username) {
    loginForm.username = route.query.username
    if (route.query.registered === 'true') {
      ElMessage.success('注册成功！请输入密码登录')
    }
  }
  if (route.query.invite) {
    registerForm.inviteCode = route.query.invite
    await validateInviteCode(route.query.invite)
  }
  if (route.path === '/register') {
    currentView.value = 'register'
  } else if (route.path === '/forgot-password') {
    currentView.value = 'forgot'
  } else {
    currentView.value = 'login'
  }
})

onUnmounted(() => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
  if (validateTimeout) {
    clearTimeout(validateTimeout)
  }
})
</script>

<style scoped lang="scss">
.unified-auth-container {
  min-height: 100vh;
  width: 100%;
  overflow: clip;
  background-color: #f8fafc;
}

.notification-alert {
  position: fixed;
  top: 20px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 9999;
  max-width: 500px;
  width: 90%;
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

.auth-wrapper {
  display: flex;
  min-height: 100vh;
  width: 100%;
}

/* ================== 左侧超现代 UI 重构 ================== */
.auth-brand-section {
  display: none;
  @media (min-width: 1024px) {
    display: flex;
    width: 50%;
    position: relative;
    overflow: hidden;
    background-color: #0b0f19; /* 极简的深夜色调 */
  }
}

/* 极光模糊球体 */
.glow-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.55;
  z-index: 0;
}
.orb-1 {
  top: -15%; left: -10%;
  width: 50vw; height: 50vw;
  background: #3b82f6;
  animation: drift 20s ease-in-out infinite alternate;
}
.orb-2 {
  bottom: -20%; right: -15%;
  width: 60vw; height: 60vw;
  background: #8b5cf6;
  animation: drift 25s ease-in-out infinite alternate-reverse;
}
.orb-3 {
  top: 30%; left: 30%;
  width: 40vw; height: 40vw;
  background: #06b6d4;
  animation: drift 22s ease-in-out infinite alternate;
}

@keyframes drift {
  0% { transform: translate(0, 0) scale(1); }
  100% { transform: translate(60px, 40px) scale(1.1); }
}

/* 科技网格背景层 */
.glass-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.04) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.04) 1px, transparent 1px);
  background-size: 40px 40px;
  z-index: 1;
  mask-image: linear-gradient(to bottom, rgba(0,0,0,1) 40%, rgba(0,0,0,0) 100%);
  -webkit-mask-image: linear-gradient(to bottom, rgba(0,0,0,1) 40%, rgba(0,0,0,0) 100%);
}

.brand-content {
  position: relative;
  z-index: 2;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 60px; /* 增加留白更透气 */
  width: 100%;
  color: white;
}

/* 左侧元素的错位渐显效果 */
@keyframes slideInLeft {
  from { opacity: 0; transform: translateX(-30px); }
  to { opacity: 1; transform: translateX(0); }
}

.stagger-brand {
  animation: slideInLeft 0.8s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  opacity: 0;
  &:nth-child(1) { animation-delay: 0.1s; }
  &:nth-child(2) { animation-delay: 0.2s; }
  &:nth-child(3) { animation-delay: 0.3s; }
  &:nth-child(4) { animation-delay: 0.4s; }
  &:nth-child(5) { animation-delay: 0.5s; }
}

.brand-header {
  .brand-logo {
    display: flex;
    align-items: center;
    gap: 14px;
    margin-bottom: 32px;
    .logo-img {
      width: 46px;
      height: 46px;
      border-radius: 14px;
      box-shadow: 0 4px 16px rgba(0,0,0,0.3);
    }
    .logo-placeholder {
      width: 46px;
      height: 46px;
      border-radius: 14px;
      background: rgba(255, 255, 255, 0.15);
      backdrop-filter: blur(12px);
      border: 1px solid rgba(255, 255, 255, 0.3);
      display: flex;
      align-items: center;
      justify-content: center;
      color: white;
      font-size: 24px;
    }
    .brand-name {
      font-size: 24px;
      font-weight: 800;
      color: white;
      letter-spacing: 0.5px;
    }
  }
}

.brand-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 48px;
}

.brand-hero {
  .brand-title {
    font-size: 42px;
    font-weight: 800;
    line-height: 1.2;
    margin-bottom: 16px;
    background: linear-gradient(135deg, #ffffff 0%, rgba(255,255,255,0.7) 100%);
    -webkit-background-clip: text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
    letter-spacing: 2px;
  }
  .brand-subtitle {
    font-size: 18px;
    color: rgba(255, 255, 255, 0.6);
    font-weight: 400;
    letter-spacing: 1px;
  }
}

.brand-features {
  display: flex;
  flex-direction: column;
  gap: 16px;

  .feature-card {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 16px 20px;
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 16px;
    backdrop-filter: blur(12px);
    transition: all 0.3s ease;

    &:hover {
      background: rgba(255, 255, 255, 0.1);
      transform: translateX(4px);
      border-color: rgba(255, 255, 255, 0.15);
    }
  }

  .feature-icon {
    width: 44px;
    height: 44px;
    border-radius: 12px;
    background: rgba(99, 102, 241, 0.3);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;

    i {
      font-size: 22px;
      color: #a5b4fc;
    }
  }

  .feature-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .feature-title {
    font-size: 15px;
    font-weight: 600;
    color: rgba(255, 255, 255, 0.95);
  }

  .feature-desc {
    font-size: 13px;
    color: rgba(255, 255, 255, 0.5);
  }
}

.brand-platforms {
  display: flex;
  align-items: center;
  gap: 16px;

  .platform-label {
    font-size: 13px;
    color: rgba(255, 255, 255, 0.5);
    font-weight: 500;
  }

  .platform-icons {
    display: flex;
    gap: 12px;

    i {
      font-size: 22px;
      color: rgba(255, 255, 255, 0.4);
      transition: color 0.2s;

      &:hover {
        color: rgba(255, 255, 255, 0.8);
      }
    }
  }
}

.brand-footer {
  margin-top: auto;
  padding-top: 32px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.6);

  .footer-links {
    display: flex;
    gap: 16px;
  }

  .footer-link {
    color: rgba(255, 255, 255, 0.6);
    text-decoration: none;
    transition: all 0.3s ease;
    &:hover {
      color: white;
      text-shadow: 0 0 8px rgba(255,255,255,0.4);
    }
  }
}

/* ================== 右侧表单区 ================== */
.auth-form-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: white;
  overflow-y: auto;
  @media (min-width: 1024px) {
    width: 50%;
  }
}

.mobile-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  border-bottom: 1px solid #e5e7eb;
  background: rgba(255,255,255,0.9);
  backdrop-filter: blur(10px);
  position: sticky;
  top: 0;
  z-index: 10;
  @media (min-width: 1024px) {
    display: none;
  }
  .mobile-logo {
    display: flex;
    align-items: center;
    gap: 12px;
    .logo-img-small {
      width: 32px;
      height: 32px;
      border-radius: 8px;
    }
    .logo-placeholder-small {
      width: 32px;
      height: 32px;
      border-radius: 8px;
      background: linear-gradient(135deg, #3b82f6 0%, #06b6d4 100%);
      display: flex;
      align-items: center;
      justify-content: center;
      color: white;
      font-size: 16px;
    }
    .mobile-brand-name {
      font-size: 18px;
      font-weight: 700;
      color: #0f172a;
    }
  }
  .download-app-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    background: #f1f5f9;
    border-radius: 8px;
    color: #334155;
    text-decoration: none;
    font-size: 14px;
    font-weight: 500;
    transition: all 0.3s;
    &:hover {
      background: #e2e8f0;
      color: #0f172a;
    }
  }
}

.form-container {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  @media (min-width: 1024px) {
    padding: 48px;
  }
}

.auth-form {
  width: 100%;
  max-width: 400px;
}

/* 右侧表单元素的阶梯渐显动画 */
@keyframes slideUpFade {
  from { opacity: 0; transform: translateY(20px); }
  to { opacity: 1; transform: translateY(0); }
}

.stagger-item {
  animation: slideUpFade 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  opacity: 0;
}

.auth-form > .stagger-item:nth-child(1) { animation-delay: 0.05s; }
.auth-form > .stagger-item:nth-child(2) { animation-delay: 0.1s; }
.auth-form .el-form-item:nth-child(1) { animation-delay: 0.15s; }
.auth-form .el-form-item:nth-child(2) { animation-delay: 0.2s; }
.auth-form .el-form-item:nth-child(3) { animation-delay: 0.25s; }
.auth-form .el-form-item:nth-child(4) { animation-delay: 0.3s; }
.auth-form .el-form-item:nth-child(5) { animation-delay: 0.35s; }
.auth-form > .stagger-item:last-child { animation-delay: 0.4s; }

.form-header {
  text-align: center;
  margin-bottom: 32px;
  .form-title {
    font-size: 28px;
    font-weight: 800;
    color: #0f172a;
    margin-bottom: 8px;
  }
  .form-subtitle {
    font-size: 15px;
    color: #64748b;
  }
}

.form-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

/* 更高级质感的按钮 */
.submit-button {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 12px;
  border: none;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 0 4px 14px 0 rgba(59, 130, 246, 0.25);
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(59, 130, 246, 0.4);
    background: linear-gradient(135deg, #60a5fa 0%, #3b82f6 100%);
  }
  
  &:active {
    transform: translateY(1px);
    box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
  }
}

.form-footer {
  text-align: center;
  margin-top: 24px;
  font-size: 14px;
  color: #64748b;
  .footer-text {
    margin-right: 8px;
  }
}

.verification-code-group {
  display: flex;
  gap: 12px;
  .verification-code-input {
    flex: 1;
  }
  .send-code-button {
    white-space: nowrap;
    border-radius: 12px;
    height: 46px; /* 对齐输入框高度 */
    transition: all 0.3s;
    &:not(:disabled):hover {
      transform: translateY(-1px);
      box-shadow: 0 4px 12px rgba(59, 130, 246, 0.2);
    }
  }
}

.invite-code-tip {
  margin-top: 8px;
  font-size: 12px;
  .tip-success { color: #10b981; }
  .tip-error { color: #ef4444; }
}

.back-button {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #64748b;
  text-decoration: none;
  font-size: 15px;
  margin-bottom: 20px;
  transition: all 0.3s;
  padding: 8px 0;
  &:hover {
    color: #0f172a;
    transform: translateX(-4px);
  }
}

/* 页面切换动画 */
.slide-fade-enter-active {
  transition: all 0.4s ease-out;
}
.slide-fade-leave-active {
  transition: all 0.3s cubic-bezier(1, 0.5, 0.8, 1);
}
.slide-fade-enter-from {
  transform: translateX(20px);
  opacity: 0;
}
.slide-fade-leave-to {
  transform: translateX(-20px);
  opacity: 0;
}

/* 深度定制 El-Input 增强现代丝滑感 */
:deep(.el-input) {
  .el-input__wrapper {
    box-shadow: 0 2px 6px rgba(0,0,0,0.02) !important;
    border: 1px solid #e2e8f0 !important;
    border-radius: 12px !important;
    background-color: #f8fafc !important;
    padding: 0 15px !important;
    min-height: 48px !important;
    transition: all 0.3s ease;
    
    &:hover {
      background-color: #ffffff !important;
      border-color: #cbd5e1 !important;
    }
    &.is-focus {
      background-color: #ffffff !important;
      border-color: #3b82f6 !important;
      box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.15) !important;
    }
  }
  .el-input__inner {
    border: none !important;
    box-shadow: none !important;
    background: transparent !important;
    padding: 0 !important;
    height: 46px !important;
    line-height: 46px !important;
    font-size: 15px !important;
    color: #1e293b;
    &::placeholder {
      color: #94a3b8;
    }
  }
  .el-input__prefix { left: 15px !important; color: #64748b; }
  .el-input__suffix { right: 15px !important; color: #64748b; }
}

:deep(.el-form-item) {
  margin-bottom: 24px;
  .el-form-item__content {
    line-height: normal;
  }
}

.mobile-footer {
  text-align: center;
  padding: 24px;
  font-size: 13px;
  color: #94a3b8;
  @media (min-width: 1024px) { display: none; }
}

@media (max-width: 768px) {
  .verification-code-group {
    gap: 8px;
    flex-wrap: nowrap;
    .verification-code-input {
      flex: 1;
      min-width: 0;
    }
    .send-code-button {
      min-width: 90px;
      max-width: 120px;
      flex-shrink: 0;
      white-space: nowrap;
      font-size: 14px;
      padding: 0 12px;
    }
  }
  /* 移动端特殊处理防缩放 */
  :deep(.verification-code-input .el-input__wrapper) {
    min-height: 48px !important; 
  }
  :deep(.verification-code-input .el-input__inner) {
    font-size: 16px !important; 
  }
}

@media (max-width: 480px) {
  .verification-code-group {
    gap: 6px;
    .verification-code-input {
      flex: 2;
    }
    .send-code-button {
      min-width: 80px;
      max-width: 100px;
      font-size: 13px;
      padding: 0 10px;
    }
  }
}
</style>