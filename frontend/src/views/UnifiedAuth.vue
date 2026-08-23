<template>
  <div class="lux-auth-page">
    <div class="lux-overlay"></div>
    <div class="lux-container">
      <div class="lux-brand">
        <div class="brand-glow"></div>
        <div class="brand-top">
          <h1 class="brand-name">{{ settings.siteName || 'TurboCloud' }}</h1>
          <p class="brand-tagline">GLOBAL NETWORK</p>
        </div>
        <div class="brand-center">
          <transition name="fade" mode="out-in">
            <div :key="currentView">
              <h2 class="brand-headline" v-if="currentView === 'login'">{{ isAdminLoginRoute ? '管理员登录' : '欢迎回来' }}</h2>
              <h2 class="brand-headline" v-else-if="currentView === 'register'">加入我们</h2>
              <h2 class="brand-headline" v-else>找回密码</h2>
              <p class="brand-desc" v-if="currentView === 'login'">{{ isAdminLoginRoute ? '进入管理后台，处理用户、订阅、订单与系统配置。' : '安全极速的全球网络加速服务，IEPL 专线直连，畅享无限可能。' }}</p>
              <p class="brand-desc" v-else-if="currentView === 'register'">注册即享全球 80+ 优质节点，10Gbps 骨干带宽，全平台客户端支持。</p>
              <p class="brand-desc" v-else>输入您的注册邮箱，我们将发送验证码帮助您重置密码。</p>
            </div>
          </transition>
        </div>
        <div class="brand-bottom">
          <a href="#" class="brand-link">隐私政策</a>
          <a href="#" class="brand-link">服务条款</a>
        </div>
      </div>
      <div class="lux-form-area">
        <div class="mobile-only-logo">
          <h1>{{ settings.siteName || 'TurboCloud' }}</h1>
          <p>GLOBAL NETWORK</p>
        </div>
        <transition name="fade" mode="out-in">
          <div v-if="currentView === 'login'" key="login" class="lux-form">
            <h3 class="form-heading">{{ isAdminLoginRoute ? '管理员登录' : '登录' }}</h3>
            <el-form ref="loginFormRef" :model="loginForm" :rules="loginRules" @submit.prevent="handleLogin" label-position="top">
              <el-form-item prop="username">
                <el-input v-model="loginForm.username" placeholder="用户名或邮箱" size="large" :prefix-icon="User" clearable autocomplete="username" />
              </el-form-item>
              <el-form-item prop="password">
                <el-input v-model="loginForm.password" :type="isPasswordVisible ? 'text' : 'password'" placeholder="输入密码" size="large" :prefix-icon="Lock" clearable autocomplete="current-password" @keyup.enter="handleLogin" @focus="isPasswordFocused = true" @blur="isPasswordFocused = false">
                  <template #suffix>
                    <button
                      type="button"
                      class="password-toggle-button"
                      :aria-label="isPasswordVisible ? '隐藏密码' : '显示密码'"
                      @click="isPasswordVisible = !isPasswordVisible"
                    >
                      <el-icon class="password-toggle-icon"><View v-if="!isPasswordVisible" /><Hide v-else /></el-icon>
                    </button>
                  </template>
                </el-input>
              </el-form-item>
              <el-form-item>
                <div class="form-options">
                  <el-checkbox v-model="rememberMe">保持登录 30 天</el-checkbox>
                  <a href="#" @click.prevent="switchView('forgot')" class="lux-link">忘记密码？</a>
                </div>
              </el-form-item>
              <el-form-item>
                <button type="submit" class="lux-btn-gold" :disabled="isLoading" @click.prevent="handleLogin">
                  <span v-if="!isLoading">登 录</span><span v-else>登录中...</span>
                </button>
              </el-form-item>
            </el-form>
            <div v-if="!isAdminLoginRoute" class="form-switch">尚未注册？ <a href="#" @click.prevent="switchView('register')" class="lux-link-gold">立即注册</a></div>
            <div v-else class="form-switch"><router-link to="/login" class="lux-link-gold">返回用户登录</router-link></div>
          </div>
          <div v-else-if="currentView === 'register'" key="register" class="lux-form">
            <h3 class="form-heading">注册</h3>
            <el-alert v-if="!registrationEnabled" title="注册功能已禁用" type="warning" :closable="false" show-icon class="auth-alert">
              <template #default><p>系统管理员已关闭用户注册功能，请联系管理员获取账户。</p></template>
            </el-alert>
            <el-form v-if="registrationEnabled" ref="registerFormRef" :model="registerForm" :rules="registerRules" @submit.prevent="handleRegister" label-position="top">
              <el-form-item prop="username">
                <el-input v-model="registerForm.username" placeholder="用户名" size="large" :prefix-icon="User" clearable autocomplete="username" />
              </el-form-item>
              <el-form-item prop="email">
                <el-input v-model="registerForm.email" type="email" placeholder="电子邮箱（推荐 QQ 邮箱）" size="large" :prefix-icon="Message" clearable autocomplete="email" />
              </el-form-item>
              <el-form-item prop="verificationCode" :required="emailVerificationRequired">
                <div class="code-group">
                  <el-input v-model="registerForm.verificationCode" :placeholder="emailVerificationRequired ? '6位验证码（必填）' : '6位验证码（选填）'" size="large" :prefix-icon="Message" maxlength="6" clearable autocomplete="off" class="code-input" />
                  <el-button type="primary" size="large" :disabled="codeTimer > 0 || !registerForm.email || sendingCode" :loading="sendingCode" @click="sendVerificationCode('register')" class="code-btn">{{ codeTimer > 0 ? `${codeTimer}s` : '获取验证码' }}</el-button>
                </div>
              </el-form-item>
              <el-form-item v-if="inviteCodeRequired" prop="inviteCode" required>
                <el-input v-model="registerForm.inviteCode" placeholder="邀请码（必填）" size="large" :prefix-icon="Ticket" clearable />
                <div v-if="inviteCodeInfo" class="invite-tip">
                  <span v-if="inviteCodeInfo.is_valid || inviteCodeInfo.success" class="tip-ok">邀请码有效，注册后可获得 {{ inviteCodeInfo.invitee_reward || inviteCodeInfo.data?.invitee_reward || 0 }} 元奖励</span>
                  <span v-else class="tip-err">{{ inviteCodeInfo.message }}</span>
                </div>
              </el-form-item>
              <el-form-item prop="password">
                <el-input v-model="registerForm.password" :type="isRegPasswordVisible ? 'text' : 'password'" placeholder="设置密码（8位以上）" size="large" :prefix-icon="Lock" clearable autocomplete="new-password" @focus="isRegPasswordFocused = true" @blur="isRegPasswordFocused = false">
                  <template #suffix>
                    <button
                      type="button"
                      class="password-toggle-button"
                      :aria-label="isRegPasswordVisible ? '隐藏密码' : '显示密码'"
                      @click="isRegPasswordVisible = !isRegPasswordVisible"
                    >
                      <el-icon class="password-toggle-icon"><View v-if="!isRegPasswordVisible" /><Hide v-else /></el-icon>
                    </button>
                  </template>
                </el-input>
              </el-form-item>
              <el-form-item prop="confirmPassword">
                <el-input v-model="registerForm.confirmPassword" :type="isRegPasswordVisible ? 'text' : 'password'" placeholder="确认密码" size="large" :prefix-icon="Lock" clearable autocomplete="new-password" @keyup.enter="handleRegister" @focus="isRegPasswordFocused = true" @blur="isRegPasswordFocused = false">
                  <template #suffix>
                    <button
                      type="button"
                      class="password-toggle-button"
                      :aria-label="isRegPasswordVisible ? '隐藏密码' : '显示密码'"
                      @click="isRegPasswordVisible = !isRegPasswordVisible"
                    >
                      <el-icon class="password-toggle-icon"><View v-if="!isRegPasswordVisible" /><Hide v-else /></el-icon>
                    </button>
                  </template>
                </el-input>
              </el-form-item>
              <el-form-item>
                <button type="submit" class="lux-btn-outline" :disabled="isLoading" @click.prevent="handleRegister">
                  <span v-if="!isLoading">提 交 注 册</span><span v-else>注册中...</span>
                </button>
              </el-form-item>
            </el-form>
            <div class="form-switch">已有账号？ <a href="#" @click.prevent="switchView('login')" class="lux-link-gold">立即登录</a></div>
          </div>
          <div v-else-if="currentView === 'forgot'" key="forgot" class="lux-form">
            <a href="#" @click.prevent="switchView('login')" class="back-link">← 返回登录</a>
            <h3 class="form-heading">重置密码</h3>
            <p class="form-desc">请输入注册邮箱，通过验证码重置密码。</p>
            <el-form ref="forgotFormRef" :model="forgotForm" :rules="forgotRules" @submit.prevent="handleReset" label-position="top">
              <el-form-item prop="email">
                <el-input v-model="forgotForm.email" type="email" placeholder="注册邮箱" size="large" :prefix-icon="Message" clearable autocomplete="email" />
              </el-form-item>
              <el-form-item prop="verificationCode" required>
                <div class="code-group">
                  <el-input v-model="forgotForm.verificationCode" placeholder="6位验证码" size="large" :prefix-icon="Message" maxlength="6" clearable autocomplete="off" class="code-input" />
                  <el-button type="primary" size="large" :disabled="codeTimer > 0 || !forgotForm.email || sendingCode" :loading="sendingCode" @click="sendVerificationCode('forgot')" class="code-btn">{{ codeTimer > 0 ? `${codeTimer}s` : '获取验证码' }}</el-button>
                </div>
              </el-form-item>
              <el-form-item prop="newPassword">
                <el-input v-model="forgotForm.newPassword" :type="isForgotPasswordVisible ? 'text' : 'password'" placeholder="新密码（8位以上）" size="large" :prefix-icon="Lock" clearable autocomplete="new-password" @focus="isForgotPasswordFocused = true" @blur="isForgotPasswordFocused = false">
                  <template #suffix>
                    <button
                      type="button"
                      class="password-toggle-button"
                      :aria-label="isForgotPasswordVisible ? '隐藏密码' : '显示密码'"
                      @click="isForgotPasswordVisible = !isForgotPasswordVisible"
                    >
                      <el-icon class="password-toggle-icon"><View v-if="!isForgotPasswordVisible" /><Hide v-else /></el-icon>
                    </button>
                  </template>
                </el-input>
              </el-form-item>
              <el-form-item prop="confirmPassword">
                <el-input v-model="forgotForm.confirmPassword" :type="isForgotPasswordVisible ? 'text' : 'password'" placeholder="确认新密码" size="large" :prefix-icon="Lock" clearable autocomplete="new-password" @keyup.enter="handleReset" @focus="isForgotPasswordFocused = true" @blur="isForgotPasswordFocused = false">
                  <template #suffix>
                    <button
                      type="button"
                      class="password-toggle-button"
                      :aria-label="isForgotPasswordVisible ? '隐藏密码' : '显示密码'"
                      @click="isForgotPasswordVisible = !isForgotPasswordVisible"
                    >
                      <el-icon class="password-toggle-icon"><View v-if="!isForgotPasswordVisible" /><Hide v-else /></el-icon>
                    </button>
                  </template>
                </el-input>
              </el-form-item>
              <el-form-item>
                <button type="submit" class="lux-btn-white" :disabled="isLoading" @click.prevent="handleReset">
                  <span v-if="!isLoading">确 认 重 置</span><span v-else>重置中...</span>
                </button>
              </el-form-item>
            </el-form>
          </div>
        </transition>
      </div>
    </div>
    <div class="lux-copyright">&copy; {{ new Date().getFullYear() }} {{ settings.siteName || 'TurboCloud' }}. All rights reserved.</div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from '@/utils/elementPlusServices'
import { User, Lock, Message, Ticket, View, Hide } from '@element-plus/icons-vue'
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
const rememberMe = ref(true)

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

// 共享密码强度校验器 (避免 registerRules / forgotRules 重复定义)
const passwordValidator = (rule, value, callback) => {
  if (!value) { callback(); return }
  const hasLetter = /[A-Za-z]/.test(value)
  const hasDigit = /\d/.test(value)
  if (!hasLetter || !hasDigit) { callback(new Error('密码必须包含字母和数字')); return }
  let c = (/[a-z]/.test(value) ? 1 : 0) + (/[A-Z]/.test(value) ? 1 : 0) + (/\d/.test(value) ? 1 : 0) + (/[!@#$%^&*()_+\-=\[\]{}|;:,.<>?]/.test(value) ? 1 : 0)
  if (c < 3) { callback(new Error('密码强度不足，建议包含大小写字母、数字和特殊字符')); return }
  callback()
}

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
const isAdminLoginRoute = computed(() => route.path === '/admin/login')

const showNotification = (message, type = 'success') => {
  notification.message = message
  notification.type = type
  notification.show = true
  setTimeout(() => {
    notification.show = false
  }, 3000)
}

const switchView = (view) => {
  if (isAdminLoginRoute.value && view !== 'login') return
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
      validator: passwordValidator, trigger: 'blur'
    }
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: (rule, value, callback) => { callback(value !== registerForm.password ? new Error('两次输入密码不一致') : undefined) }, trigger: 'blur' }
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
      validator: passwordValidator, trigger: 'blur'
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
      password: loginForm.password,
      remember: rememberMe.value,
      requireAdmin: isAdminLoginRoute.value
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
  if (route.path === '/admin/login') {
    currentView.value = 'login'
  } else if (route.path === '/register') {
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
$auth-primary: #6366f1;
$auth-primary-dark: #4f46e5;
$auth-primary-soft: #e0e7ff;
$auth-panel: #111827;
$auth-panel-muted: #1f2937;
$auth-border: rgba(226, 232, 240, 0.16);
$auth-text-muted: #cbd5e1;

.lux-auth-page {
  min-height: 100vh;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f3f4f6;
  position: relative;
  font-family: 'Noto Sans SC', sans-serif;
}

.lux-overlay {
  position: absolute;
  inset: 0;
  background: transparent;
}

.lux-container {
  position: relative;
  z-index: 10;
  width: 100%;
  max-width: 1000px;
  margin: 16px;
  display: flex;
  min-height: 580px;
  background: $auth-panel;
  border: 1px solid $auth-border;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: none;
}

.lux-brand {
  width: 42%;
  padding: 48px 40px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  position: relative;
  overflow: hidden;
  border-right: 1px solid $auth-border;

  @media (max-width: 768px) { display: none; }
}

.brand-glow {
  position: absolute;
  inset: 0;
  background: rgba(99, 102, 241, 0.06);
  pointer-events: none;
}

.brand-top {
  position: relative;
  z-index: 1;
  .brand-name {
    font-size: 28px;
    font-weight: 600;
    color: #fff;
    letter-spacing: 0;
    margin: 0;
  }
  .brand-tagline {
    font-size: 12px;
    font-weight: 500;
    letter-spacing: 0;
    color: $auth-text-muted;
    margin-top: 4px;
  }
}

.brand-center {
  position: relative;
  z-index: 1;
  margin: 48px 0;
  .brand-headline {
    font-size: 32px;
    font-weight: 600;
    line-height: 1.3;
    color: #fff;
    margin: 0 0 16px;
  }
  .brand-desc {
    font-size: 14px;
    font-weight: 400;
    line-height: 1.8;
    color: $auth-text-muted;
    margin: 0;
  }
}

.brand-bottom {
  position: relative;
  z-index: 1;
  display: flex;
  gap: 24px;
  .brand-link {
    font-size: 13px;
    font-weight: 400;
    color: $auth-text-muted;
    text-decoration: none;
    transition: color 0.2s ease;
    &:hover { color: #fff; }
  }
}

.lux-form-area {
  flex: 1;
  padding: 40px 48px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  position: relative;
  background: $auth-panel-muted;

  @media (max-width: 768px) { padding: 32px 24px; }
}

.mobile-only-logo {
  display: none;
  text-align: center;
  margin-bottom: 32px;
  h1 { font-size: 22px; color: #fff; letter-spacing: 0; margin: 0; }
  p { font-size: 11px; color: $auth-text-muted; letter-spacing: 0; margin-top: 4px; }
  @media (max-width: 768px) { display: block; }
}

.lux-form {
  max-width: 400px;
  margin: 0 auto;
  width: 100%;
}

.form-heading {
  font-size: 24px;
  font-weight: 600;
  color: #fff;
  letter-spacing: 0;
  margin: 0 0 28px;
}

.form-desc {
  font-size: 13px;
  color: $auth-text-muted;
  font-weight: 400;
  margin: -16px 0 24px;
}

.auth-alert {
  margin-bottom: 20px;
}

.password-toggle-button {
  width: 36px;
  min-width: 36px;
  height: 36px;
  padding: 0;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: $auth-text-muted;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  touch-action: manipulation;
  transition: background-color 0.2s ease, color 0.2s ease;

  &:hover {
    color: #fff;
    background: rgba(255, 255, 255, 0.06);
  }

  &:focus-visible {
    outline: 2px solid rgba($auth-primary-soft, 0.65);
    outline-offset: 2px;
  }
}

.password-toggle-icon {
  pointer-events: none;
}

@media (max-width: 768px) {
  .password-toggle-button {
    width: 44px;
    min-width: 44px;
    height: 44px;
  }
}

.back-link {
  display: inline-flex;
  align-items: center;
  font-size: 13px;
  color: $auth-text-muted;
  text-decoration: none;
  margin-bottom: 20px;
  transition: color 0.2s ease;
  &:hover { color: #fff; }
}

// Element Plus 深色主题覆盖
.lux-auth-page :deep(.el-input__wrapper) {
  background: rgba(15, 23, 42, 0.72) !important;
  border: 1px solid $auth-border !important;
  border-radius: 8px !important;
  box-shadow: none !important;
  transition: background-color 0.2s ease, border-color 0.2s ease;
  &:hover, &.is-focus {
    background: rgba(15, 23, 42, 0.88) !important;
    border-color: rgba($auth-primary-soft, 0.45) !important;
    box-shadow: none !important;
  }
}
.lux-auth-page :deep(.el-input__inner) {
  color: #fff !important;
  &::placeholder { color: rgba(255,255,255,0.35) !important; }
}
.lux-auth-page :deep(.el-input__prefix .el-icon),
.lux-auth-page :deep(.el-input__suffix .el-icon) {
  color: $auth-text-muted !important;
}
.lux-auth-page :deep(.el-form-item__error) {
  color: #fca5a5 !important;
}
.lux-auth-page :deep(.el-checkbox__label) {
  color: $auth-text-muted !important;
  font-size: 13px !important;
}
.lux-auth-page :deep(.el-checkbox__inner) {
  background: transparent !important;
  border-color: $auth-border !important;
}
.lux-auth-page :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background: $auth-primary !important;
  border-color: $auth-primary !important;
}

.form-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.lux-link {
  font-size: 13px;
  color: $auth-primary-soft;
  text-decoration: none;
  transition: color 0.2s ease;
  &:hover { color: #fff; }
}

.lux-link-gold {
  color: $auth-primary-soft;
  font-weight: 600;
  text-decoration: none;
  margin-left: 4px;
  transition: color 0.2s ease;
  &:hover { color: #fff; }
}

.lux-btn-gold {
  width: 100%;
  padding: 14px;
  border: none;
  border-radius: 8px;
  background: $auth-primary;
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0;
  cursor: pointer;
  transition: background-color 0.2s ease, opacity 0.2s ease;
  box-shadow: none;
  &:hover:not(:disabled) {
    background: $auth-primary-dark;
    box-shadow: none;
  }
  &:disabled { opacity: 0.6; cursor: not-allowed; }
}

.lux-btn-outline {
  width: 100%;
  padding: 14px;
  border: 1px solid rgba($auth-primary-soft, 0.55);
  border-radius: 8px;
  background: transparent;
  color: $auth-primary-soft;
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0;
  cursor: pointer;
  transition: background-color 0.2s ease, color 0.2s ease, opacity 0.2s ease;
  &:hover:not(:disabled) {
    background: rgba($auth-primary, 0.2);
    color: #fff;
  }
  &:disabled { opacity: 0.6; cursor: not-allowed; }
}

.lux-btn-white {
  width: 100%;
  padding: 14px;
  border: none;
  border-radius: 8px;
  background: #fff;
  color: #111827;
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0;
  cursor: pointer;
  transition: background-color 0.3s ease, opacity 0.2s ease;
  box-shadow: none;
  &:hover:not(:disabled) { background: #e5e5e5; }
  &:disabled { opacity: 0.6; cursor: not-allowed; }
}

.form-switch {
  margin-top: 28px;
  text-align: center;
  font-size: 14px;
  color: $auth-text-muted;
  font-weight: 400;
}

.code-group {
  display: flex;
  gap: 8px;
  width: 100%;
  .code-input { flex: 1; }
  .code-btn {
    flex-shrink: 0;
    border-radius: 8px !important;
    min-width: 110px;
  }
}

.invite-tip {
  margin-top: 6px;
  font-size: 12px;
  .tip-ok { color: #86efac; }
  .tip-err { color: #fca5a5; }
}

.lux-copyright {
  position: absolute;
  bottom: 16px;
  width: 100%;
  text-align: center;
  font-size: 12px;
  color: rgba(226,232,240,0.5);
  font-weight: 400;
  z-index: 10;
}

// 过渡动画
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.4s ease, transform 0.4s ease;
}
.fade-enter-from { opacity: 0; transform: translateY(8px); }
.fade-leave-to { opacity: 0; transform: translateY(-8px); }
</style>
