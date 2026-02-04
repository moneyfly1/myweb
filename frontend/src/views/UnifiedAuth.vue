<template>
  <div class="unified-auth-container">
    <!-- 全局消息提示 -->
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
      <!-- 左侧：品牌/视觉区域 (桌面端显示) -->
      <div class="auth-brand-section">
        <div class="brand-content">
          <div class="brand-header">
            <div class="brand-logo">
              <img v-if="settings.siteLogo" :src="settings.siteLogo" :alt="settings.siteName" class="logo-img" />
              <div v-else class="logo-placeholder">
                <i class="ph-bold ph-rocket-launch"></i>
              </div>
              <span class="brand-name">{{ settings.siteName || 'TurboCloud' }}</span>
            </div>
          </div>

          <div class="brand-body">
            <div class="brand-badge">
              <span class="badge-dot"></span>
              IEPL 专线全线升级
            </div>
            <h2 class="brand-title">畅享全球网络<br>解锁流媒体限制</h2>
            <div class="brand-features">
              <div class="feature-item">
                <i class="ph-fill ph-check-circle"></i>
                <span>支持 Netflix, Disney+, ChatGPT</span>
              </div>
              <div class="feature-item">
                <i class="ph-fill ph-check-circle"></i>
                <span>晚高峰 4K/8K 极速秒开</span>
              </div>
              <div class="feature-item">
                <i class="ph-fill ph-check-circle"></i>
                <span>全平台客户端支持</span>
              </div>
            </div>
            
            <div class="brand-stats">
              <div class="stat-item">
                <span class="stat-value">80+</span>
                <span class="stat-label">全球节点</span>
              </div>
              <div class="stat-divider"></div>
              <div class="stat-item">
                <span class="stat-value">10Gbps</span>
                <span class="stat-label">骨干带宽</span>
              </div>
            </div>
          </div>

          <div class="brand-footer">
            <span>&copy; 2026 {{ settings.siteName || 'TurboCloud' }} Network.</span>
            <a href="#" class="footer-link">服务条款</a>
          </div>
        </div>
      </div>

      <!-- 右侧：表单交互区域 -->
      <div class="auth-form-section">
        <!-- 移动端头部 -->
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
          <transition name="fade" mode="out-in">
            <!-- 1. 登录表单 -->
            <div v-if="currentView === 'login'" key="login" class="auth-form">
              <div class="form-header">
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
                <el-form-item prop="username">
                  <el-input
                    v-model="loginForm.username"
                    placeholder="用户名或邮箱"
                    size="large"
                    prefix-icon="User"
                    clearable
                    autocomplete="username"
                  />
                </el-form-item>

                <el-form-item prop="password">
                  <el-input
                    v-model="loginForm.password"
                    type="password"
                    placeholder="输入密码"
                    size="large"
                    prefix-icon="Lock"
                    show-password
                    clearable
                    autocomplete="current-password"
                    @keyup.enter="handleLogin"
                  />
                </el-form-item>

                <el-form-item>
                  <div class="form-options">
                    <el-checkbox v-model="rememberMe">记住我</el-checkbox>
                    <el-link type="primary" @click="switchView('forgot')" :underline="false">
                      忘记密码？
                    </el-link>
                  </div>
                </el-form-item>

                <el-form-item>
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

              <div class="form-footer">
                <span class="footer-text">新用户？</span>
                <el-link type="primary" @click="switchView('register')" :underline="false">
                  注册免费试用
                </el-link>
              </div>
            </div>

            <!-- 2. 注册表单 -->
            <div v-else-if="currentView === 'register'" key="register" class="auth-form">
              <div class="form-header">
                <h1 class="form-title">注册账号</h1>
              </div>

              <!-- 注册已禁用提示 -->
              <el-alert
                v-if="!registrationEnabled"
                title="注册功能已禁用"
                type="warning"
                :closable="false"
                show-icon
                style="margin-bottom: 20px;"
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
                <el-form-item prop="username">
                  <el-input
                    v-model="registerForm.username"
                    placeholder="用户名"
                    size="large"
                    prefix-icon="User"
                    clearable
                    autocomplete="username"
                  />
                </el-form-item>

                <el-form-item prop="email">
                  <el-input
                    v-model="registerForm.email"
                    type="email"
                    placeholder="推荐使用 Gmail / Outlook"
                    size="large"
                    prefix-icon="Message"
                    clearable
                    autocomplete="email"
                  />
                </el-form-item>

                <!-- 验证码模块 -->
                <el-form-item prop="verificationCode" :required="emailVerificationRequired">
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

                <el-form-item prop="inviteCode" :required="inviteCodeRequired">
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

                <el-form-item prop="password">
                  <el-input
                    v-model="registerForm.password"
                    type="password"
                    placeholder="8位以上字符"
                    size="large"
                    prefix-icon="Lock"
                    show-password
                    clearable
                    autocomplete="new-password"
                  />
                </el-form-item>

                <el-form-item prop="confirmPassword">
                  <el-input
                    v-model="registerForm.confirmPassword"
                    type="password"
                    placeholder="确认密码"
                    size="large"
                    prefix-icon="Lock"
                    show-password
                    clearable
                    autocomplete="new-password"
                    @keyup.enter="handleRegister"
                  />
                </el-form-item>

                <el-form-item>
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

              <div class="form-footer">
                <span class="footer-text">已有账号？</span>
                <el-link type="primary" @click="switchView('login')" :underline="false">
                  立即登录
                </el-link>
              </div>
            </div>

            <!-- 3. 找回密码表单 -->
            <div v-else-if="currentView === 'forgot'" key="forgot" class="auth-form">
              <el-button
                text
                @click="switchView('login')"
                class="back-button"
              >
                <i class="ph-bold ph-arrow-left"></i>
                返回登录
              </el-button>

              <div class="form-header">
                <div class="forgot-icon">
                  <i class="ph-fill ph-lock-key-open"></i>
                </div>
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
                <el-form-item prop="email">
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

                  <!-- 验证码模块 -->
                  <el-form-item prop="verificationCode" required>
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

                <el-form-item prop="newPassword">
                  <el-input
                    v-model="forgotForm.newPassword"
                    type="password"
                    placeholder="输入新密码"
                    size="large"
                    prefix-icon="Lock"
                    show-password
                    clearable
                    autocomplete="new-password"
                  />
                </el-form-item>

                <el-form-item prop="confirmPassword">
                  <el-input
                    v-model="forgotForm.confirmPassword"
                    type="password"
                    placeholder="确认新密码"
                    size="large"
                    prefix-icon="Lock"
                    show-password
                    clearable
                    autocomplete="new-password"
                    @keyup.enter="handleReset"
                  />
                </el-form-item>

                <el-form-item>
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
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/store/auth'
import { useSettingsStore } from '@/store/settings'
import { authAPI, inviteAPI, settingsAPI } from '@/utils/api'
import { useThemeStore } from '@/store/theme'
import { secureStorage } from '@/utils/secureStorage'
import { resetRefreshFailed } from '@/utils/api'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()

// 响应式数据
const currentView = ref('login')
const isLoading = ref(false)
const sendingCode = ref(false)
const codeTimer = ref(0)
const rememberMe = ref(false)
const registrationEnabled = ref(true)
const inviteCodeRequired = ref(false)
const emailVerificationRequired = ref(true)
const minPasswordLength = ref(8)
const inviteCodeInfo = ref(null)

let countdownTimer = null

// 表单引用
const loginFormRef = ref()
const registerFormRef = ref()
const forgotFormRef = ref()

// 表单数据
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

// 通知
const notification = reactive({
  show: false,
  message: '',
  type: 'success'
})

const settings = computed(() => settingsStore)

// 显示通知
const showNotification = (message, type = 'success') => {
  notification.message = message
  notification.type = type
  notification.show = true
  setTimeout(() => {
    notification.show = false
  }, 3000)
}

// 切换视图
const switchView = (view) => {
  currentView.value = view
  // 重置表单
  if (view === 'login') {
    loginForm.password = ''
  } else if (view === 'register') {
    registerForm.password = ''
    registerForm.confirmPassword = ''
    registerForm.verificationCode = ''
  } else if (view === 'forgot') {
    forgotForm.newPassword = ''
    forgotForm.confirmPassword = ''
    forgotForm.verificationCode = ''
  }
  notification.show = false
}

// 表单验证规则
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

// 发送验证码
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
      // 注册验证码
      await authAPI.sendVerificationCode({
        email: email,
        type: 'email'
      })
    } else {
      // 忘记密码验证码 - 使用忘记密码接口
      await authAPI.forgotPassword({ email: email })
    }
    
    ElMessage.success('验证码已发送，请查收邮箱')
    
    // 开始倒计时（60秒）
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

// 登录处理
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
      await router.push('/dashboard')
    } else {
      ElMessage.error(result.message || '登录失败，请重试')
    }
  } catch (error) {
    let errorMessage = error.response?.data?.detail || 
                      error.response?.data?.message || 
                      error.message || 
                      '登录失败，请重试'
    
    // 处理不同状态码的错误
    if (error.response?.status === 403) {
      // 403 禁止访问 - 可能是账户被禁用或 CSRF 验证失败
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
        // 刷新页面以获取新的 CSRF token
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
      // 请求过于频繁或账户被锁定
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
      // 账户被禁用（即使不是 403 状态码）
      ElMessage({
        message: '账户已被禁用，无法使用服务。如有疑问，请联系管理员。',
        type: 'error',
        duration: 5000,
        showClose: true
      })
    } else if (errorMessage.includes('系统维护')) {
      // 系统维护中
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

// 注册处理
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
        
        const REFRESH_TOKEN_TTL = 7 * 24 * 60 * 60 * 1000
        
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
        
        authStore.setAuth(access_token, safeUserData, true)
        
        if (refresh_token) {
          secureStorage.set('user_refresh_token', refresh_token, true, REFRESH_TOKEN_TTL)
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

// 重置密码处理
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


// 验证邀请码
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

// 监听邀请码变化
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

// 检查注册设置
const checkRegistrationSettings = async () => {
  try {
    const response = await settingsAPI.getPublicSettings()
    const settings = response.data?.data || response.data || {}
    
    // 支持多种字段名格式：registration_enabled 或 allowRegistration
    const registrationValue = settings.registration_enabled !== undefined 
                            ? settings.registration_enabled
                            : (settings.allowRegistration !== undefined 
                               ? settings.allowRegistration 
                               : true)
    registrationEnabled.value = registrationValue === true || registrationValue === "true"
    
    // 支持多种字段名格式：invite_code_required 或 inviteCodeRequired
    const inviteCodeValue = settings.invite_code_required !== undefined 
                           ? settings.invite_code_required
                           : (settings.inviteCodeRequired !== undefined 
                              ? settings.inviteCodeRequired 
                              : false)
    inviteCodeRequired.value = inviteCodeValue === true || inviteCodeValue === "true"
    
    // 支持多种字段名格式：email_verification_required 或 emailVerificationRequired
    const emailVerificationValue = settings.email_verification_required !== undefined 
                                   ? settings.email_verification_required
                                   : (settings.emailVerificationRequired !== undefined 
                                      ? settings.emailVerificationRequired 
                                      : (settings.require_email_verification !== undefined 
                                         ? settings.require_email_verification 
                                         : true))
    emailVerificationRequired.value = emailVerificationValue === true || emailVerificationValue === "true"
    
    // 支持多种字段名格式：min_password_length 或 minPasswordLength
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

// 从URL参数中获取信息
onMounted(async () => {
  await checkRegistrationSettings()
  await settingsStore.loadSettings()
  
  // 从URL参数中获取用户名（注册成功后跳转）
  if (route.query.username) {
    loginForm.username = route.query.username
    if (route.query.registered === 'true') {
      ElMessage.success('注册成功！请输入密码登录')
    }
  }
  
  // 从URL参数中获取邀请码
  if (route.query.invite) {
    registerForm.inviteCode = route.query.invite
    await validateInviteCode(route.query.invite)
  }
  
  // 根据路由路径设置当前视图
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
  overflow: hidden;
  background-color: #f3f4f6;
}

.notification-alert {
  position: fixed;
  top: 20px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 9999;
  max-width: 500px;
  width: 90%;
}

.auth-wrapper {
  display: flex;
  min-height: 100vh;
  width: 100%;
}

.auth-brand-section {
  display: none;
  
  @media (min-width: 1024px) {
    display: flex;
    width: 50%;
    position: relative;
    overflow: hidden;
    background: linear-gradient(135deg, rgba(10, 10, 30, 0.92) 0%, rgba(37, 99, 235, 0.85) 100%);
    
    &::before {
      content: '';
      position: absolute;
      inset: 0;
      background-image: url('https://images.unsplash.com/photo-1451187580459-43490279c0fa?q=80&w=2072&auto=format&fit=crop');
      background-size: cover;
      background-position: center;
      opacity: 0.4;
      z-index: 0;
    }
  }
}

.brand-content {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 48px;
  width: 100%;
  color: white;
}

.brand-header {
  .brand-logo {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 32px;
    
    .logo-img {
      width: 40px;
      height: 40px;
      border-radius: 12px;
    }
    
    .logo-placeholder {
      width: 40px;
      height: 40px;
      border-radius: 12px;
      background: rgba(255, 255, 255, 0.1);
      display: flex;
      align-items: center;
      justify-content: center;
      color: white;
      font-size: 20px;
    }
    
    .brand-name {
      font-size: 20px;
      font-weight: 600;
      color: white;
    }
  }
}

.brand-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.brand-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 20px;
  font-size: 14px;
  margin-bottom: 24px;
  width: fit-content;
  
  .badge-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #10b981;
    animation: pulse 2s infinite;
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.brand-title {
  font-size: 36px;
  font-weight: 700;
  line-height: 1.2;
  margin-bottom: 32px;
  color: white;
}

.brand-features {
  display: flex;
  flex-direction: column;
  gap: 16px;
  
  .feature-item {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 16px;
    color: rgba(255, 255, 255, 0.9);
    
    i {
      font-size: 20px;
      color: #10b981;
    }
  }
}

.brand-footer {
  margin-top: auto;
  padding-top: 32px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  
  .brand-stats {
    display: flex;
    gap: 32px;
    margin-bottom: 24px;
    
    .stat-item {
      display: flex;
      flex-direction: column;
      
      .stat-value {
        font-size: 24px;
        font-weight: 700;
        color: white;
      }
      
      .stat-label {
        font-size: 14px;
        color: rgba(255, 255, 255, 0.7);
        margin-top: 4px;
      }
    }
  }
  
  .footer-link {
    color: rgba(255, 255, 255, 0.7);
    text-decoration: none;
    font-size: 14px;
    transition: color 0.3s;
    
    &:hover {
      color: white;
    }
  }
}

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
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      display: flex;
      align-items: center;
      justify-content: center;
      color: white;
      font-size: 16px;
    }
    
    .mobile-brand-name {
      font-size: 18px;
      font-weight: 600;
      color: #1f2937;
    }
  }
  
  .download-app-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    background: #f3f4f6;
    border-radius: 8px;
    color: #4b5563;
    text-decoration: none;
    font-size: 14px;
    transition: all 0.3s;
    
    &:hover {
      background: #e5e7eb;
      color: #1f2937;
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

.form-header {
  text-align: center;
  margin-bottom: 32px;
  
  .form-title {
    font-size: 28px;
    font-weight: 700;
    color: #1f2937;
    margin-bottom: 8px;
  }
  
  .form-subtitle {
    font-size: 14px;
    color: #6b7280;
  }
}

.form-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.submit-button {
  width: 100%;
  height: 44px;
  font-size: 16px;
  font-weight: 600;
}

.form-footer {
  text-align: center;
  margin-top: 24px;
  font-size: 14px;
  color: #6b7280;
  
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
  }
}

.invite-code-tip {
  margin-top: 8px;
  font-size: 12px;
  
  .tip-success {
    color: #10b981;
  }
  
  .tip-error {
    color: #ef4444;
  }
}

.back-button {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #6b7280;
  text-decoration: none;
  font-size: 14px;
  margin-bottom: 24px;
  transition: color 0.3s;
  
  &:hover {
    color: #1f2937;
  }
}

.forgot-icon {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: #fef3c7;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 24px;
  color: #f59e0b;
  font-size: 32px;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

// 去掉输入框内部嵌套效果，使用完全扁平样式
:deep(.el-input) {
  .el-input__wrapper {
    box-shadow: none !important;
    border: 1px solid #dcdfe6 !important;
    border-radius: 0 !important;
    background-color: #ffffff !important;
    padding: 0 15px !important;
    min-height: 44px !important;
    
    &:hover {
      box-shadow: none !important;
      border-color: #c0c4cc !important;
    }
    
    &.is-focus {
      box-shadow: none !important;
      border-color: #409eff !important;
    }
  }
  
  .el-input__inner {
    border: none !important;
    box-shadow: none !important;
    background: transparent !important;
    padding: 0 !important;
    height: 42px !important;
    line-height: 42px !important;
    font-size: 14px !important;
  }
  
  .el-input__prefix {
    left: 15px !important;
  }
  
  .el-input__suffix {
    right: 15px !important;
  }
}

// 去掉表单项的额外间距和嵌套
:deep(.el-form-item) {
  margin-bottom: 20px;
  
  .el-form-item__content {
    line-height: normal;
  }
}

// 确保输入框大小一致
:deep(.el-input--large) {
  .el-input__wrapper {
    min-height: 44px !important;
  }
  
  .el-input__inner {
    height: 42px !important;
    line-height: 42px !important;
  }
}

// 手机端验证码输入框优化
@media (max-width: 768px) {
  .verification-code-group {
    gap: 8px;
    flex-wrap: nowrap;
    
    .verification-code-input {
      flex: 1;
      min-width: 0; /* 允许缩小 */
    }
    
    .send-code-button {
      min-width: 90px;
      max-width: 120px;
      flex-shrink: 0; /* 防止按钮被压缩 */
      white-space: nowrap;
      font-size: 14px;
      padding: 0 12px;
    }
  }
  
  /* 手机端验证码输入框文字颜色和输入修复 */
  :deep(.verification-code-input) {
    -webkit-user-select: text !important;
    user-select: text !important;
    pointer-events: auto !important;
    touch-action: manipulation !important;
    -webkit-tap-highlight-color: transparent !important;
  }
  
  :deep(.verification-code-input .el-input__wrapper) {
    pointer-events: auto !important;
    touch-action: manipulation !important;
    -webkit-tap-highlight-color: transparent !important;
    background-color: #ffffff !important;
    min-height: 48px !important; /* 手机端增大高度，防止iOS自动缩放 */
  }
  
  :deep(.verification-code-input .el-input__inner) {
    color: #303133 !important;
    -webkit-text-fill-color: #303133 !important;
    font-size: 16px !important; /* 防止iOS自动缩放 */
    opacity: 1 !important;
    caret-color: #1677ff !important;
    -webkit-user-select: text !important;
    user-select: text !important;
    pointer-events: auto !important;
    touch-action: manipulation !important;
    -webkit-appearance: none !important;
    appearance: none !important;
    background-color: transparent !important;
    height: 46px !important;
    line-height: 46px !important;
  }
  
  :deep(.verification-code-input .el-input__wrapper .el-input__inner) {
    color: #303133 !important;
    -webkit-text-fill-color: #303133 !important;
    font-size: 16px !important;
    opacity: 1 !important;
    -webkit-appearance: none !important;
    appearance: none !important;
    background-color: transparent !important;
  }
  
  :deep(.verification-code-input .el-input__wrapper.is-focus .el-input__inner) {
    color: #303133 !important;
    -webkit-text-fill-color: #303133 !important;
    font-size: 16px !important;
    opacity: 1 !important;
    -webkit-appearance: none !important;
    appearance: none !important;
    background-color: transparent !important;
  }
  
  :deep(.verification-code-input .el-input__wrapper:hover .el-input__inner) {
    color: #303133 !important;
    -webkit-text-fill-color: #303133 !important;
    font-size: 16px !important;
    opacity: 1 !important;
  }
  
  /* 确保输入框在所有状态下都可以正常输入 */
  :deep(.verification-code-input input) {
    color: #303133 !important;
    -webkit-text-fill-color: #303133 !important;
    font-size: 16px !important;
    opacity: 1 !important;
    caret-color: #1677ff !important;
    -webkit-user-select: text !important;
    user-select: text !important;
    pointer-events: auto !important;
    touch-action: manipulation !important;
    -webkit-appearance: none !important;
    appearance: none !important;
  }
}

// 小屏幕手机端进一步优化
@media (max-width: 480px) {
  .verification-code-group {
    gap: 6px;
    
    .verification-code-input {
      flex: 2; /* 增加输入框占比，让它更长 */
      min-width: 0;
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