<template>
  <div class="register-container">
    <div class="register-card">
      <div class="register-header">
        <img v-if="settings.siteLogo" :src="settings.siteLogo" :alt="settings.siteName" class="logo" />
        <h1>{{ settings.siteName }}</h1>
        <p>创建您的账户</p>
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
        label-width="0"
        class="register-form"
      >
        <el-form-item prop="email">
          <div class="email-input-group">
            <el-input
              v-model="registerForm.emailPrefix"
              placeholder="邮箱前缀"
              prefix-icon="Message"
              size="large"
              class="email-prefix"
            />
            <span class="email-separator">@</span>
            <el-select
              v-model="registerForm.emailDomain"
              placeholder="选择邮箱类型"
              size="large"
              class="email-domain"
            >
              <el-option
                v-for="domain in allowedEmailDomains"
                :key="domain"
                :label="domain"
                :value="domain"
              />
            </el-select>
          </div>
        </el-form-item>

        <el-form-item prop="username">
          <el-input
            v-model="registerForm.username"
            placeholder="用户名"
            prefix-icon="User"
            size="large"
          />
        </el-form-item>

        <el-form-item prop="password">
          <el-input
            v-model="registerForm.password"
            type="password"
            placeholder="密码"
            prefix-icon="Lock"
            size="large"
            show-password
          />
        </el-form-item>

        <el-form-item prop="confirmPassword">
          <el-input
            v-model="registerForm.confirmPassword"
            type="password"
            placeholder="确认密码"
            prefix-icon="Lock"
            size="large"
            show-password
          />
        </el-form-item>

        <el-form-item prop="verificationCode" v-if="emailVerificationRequired">
          <div class="verification-code-group">
            <el-input
              v-model="registerForm.verificationCode"
              placeholder="请输入验证码"
              prefix-icon="Message"
              size="large"
              class="verification-code-input"
              maxlength="6"
              type="text"
              clearable
              autocomplete="off"
            />
            <el-button
              type="primary"
              size="large"
              class="send-code-button"
              :disabled="!canSendCode || countdown > 0"
              :loading="sendingCode"
              @click="handleSendVerificationCode"
            >
              {{ countdown > 0 ? `${countdown}秒后重试` : '发送验证码' }}
            </el-button>
          </div>
        </el-form-item>

        <el-form-item prop="inviteCode">
          <el-input
            v-model="registerForm.inviteCode"
            :placeholder="inviteCodeRequired ? '请输入邀请码（必填）' : '邀请码（可选，填写可获得注册奖励）'"
            prefix-icon="UserFilled"
            size="large"
            clearable
          />
          <div class="form-tip" v-if="inviteCodeInfo">
            <span v-if="inviteCodeInfo.is_valid || inviteCodeInfo.success" style="color: #67c23a;">
              ✓ 邀请码有效，注册后可获得 {{ inviteCodeInfo.invitee_reward || inviteCodeInfo.data?.invitee_reward || 0 }} 元奖励
            </span>
            <span v-else style="color: #f56c6c;">
              ✗ {{ inviteCodeInfo.message }}
            </span>
          </div>
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            class="register-button"
            :loading="loading"
            @click="handleRegister"
          >
            注册
          </el-button>
        </el-form-item>
      </el-form>

      <div class="register-footer" v-if="registrationEnabled">
        <p>已有账户？ <router-link to="/login">立即登录</router-link></p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { authAPI, inviteAPI } from '@/utils/api'
import { useSettingsStore } from '@/store/settings'
import { useAuthStore } from '@/store/auth'
import { settingsAPI, resetRefreshFailed } from '@/utils/api'
import { secureStorage } from '@/utils/secureStorage'
import { useThemeStore } from '@/store/theme'

const router = useRouter()
const route = useRoute()
const settingsStore = useSettingsStore()
const authStore = useAuthStore()

// 注册是否允许
const registrationEnabled = ref(true)
const inviteCodeRequired = ref(false) // 邀请码是否必填
const emailVerificationRequired = ref(true) // 邮箱验证是否必填
const minPasswordLength = ref(8) // 最小密码长度

// 响应式数据
const loading = ref(false)
const registerFormRef = ref()
const sendingCode = ref(false) // 发送验证码加载状态
const countdown = ref(0) // 倒计时
let countdownTimer = null // 倒计时定时器
const inviteCodeInfo = ref(null) // 邀请码验证信息

const registerForm = reactive({
  emailPrefix: '',
  emailDomain: 'qq.com', // 默认选择qq.com
  email: '', // 计算属性，由前缀和域名组成
  username: '',
  password: '',
  confirmPassword: '',
  verificationCode: '', // 验证码
  inviteCode: '' // 邀请码
})

// 允许的邮箱域名
const allowedEmailDomains = [
  'qq.com',
  'gmail.com', 
  '126.com',
  '163.com',
  'hotmail.com',
  'foxmail.com'
]

// 监听邮箱前缀和域名的变化，自动组合完整邮箱
watch([() => registerForm.emailPrefix, () => registerForm.emailDomain], ([prefix, domain]) => {
  if (prefix && domain) {
    registerForm.email = `${prefix}@${domain}`
  } else {
    registerForm.email = ''
  }
})

const settings = computed(() => settingsStore)

// 表单验证规则
const registerRules = computed(() => ({
  email: [
    { 
      validator: (rule, value, callback) => {
        if (!registerForm.emailPrefix || !registerForm.emailDomain) {
          callback(new Error('请填写完整的邮箱地址'))
          return
        }
        const email = `${registerForm.emailPrefix}@${registerForm.emailDomain}`
        const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
        if (!emailPattern.test(email)) {
          callback(new Error('邮箱格式不正确，请检查邮箱地址'))
          return
        }
        callback()
      }, 
      trigger: ['change', 'blur'] 
    }
  ],
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { 
      min: 2, 
      max: 20, 
      message: '用户名长度必须在 2 到 20 个字符之间', 
      trigger: 'blur' 
    },
    { 
      pattern: /^[a-zA-Z0-9_]+$/, 
      message: '用户名只能包含字母、数字和下划线，不能包含空格或特殊字符', 
      trigger: 'blur' 
    }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { 
      min: minPasswordLength.value, 
      max: 50, 
      message: `密码长度至少 ${minPasswordLength.value} 位，最多 50 位`, 
      trigger: 'blur' 
    },
    { 
      validator: (rule, value, callback) => {
        if (!value) {
          callback()
          return
        }
        // 检查密码是否包含字母和数字
        const hasLetter = /[A-Za-z]/.test(value)
        const hasDigit = /\d/.test(value)
        if (!hasLetter || !hasDigit) {
          callback(new Error('密码必须包含字母和数字'))
          return
        }
        // 检查密码强度（至少包含大小写字母、数字和特殊字符中的三种）
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

const canSendCode = computed(() => {
  return registerForm.emailPrefix && registerForm.emailDomain
})

const handleSendVerificationCode = async () => {
  if (!registerForm.emailPrefix || !registerForm.emailDomain) {
    ElMessage.warning('请先填写完整的邮箱地址')
    return
  }
  
  if (!registerForm.email) {
    ElMessage.warning('请先填写完整的邮箱地址')
    return
  }
  
  sendingCode.value = true
  
  try {
    const response = await authAPI.sendVerificationCode({
      email: registerForm.email,
      type: 'email'
    })
    
    ElMessage.success('验证码已发送，请查收邮箱')
    
    // 开始倒计时（60秒）
    countdown.value = 60
    if (countdownTimer) {
      clearInterval(countdownTimer)
    }
    countdownTimer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        clearInterval(countdownTimer)
        countdownTimer = null
      }
    }, 1000)
    
  } catch (error) {
    if (error.response?.data?.detail) {
      ElMessage.error(error.response.data.detail)
    } else {
      ElMessage.error('发送验证码失败，请重试')
    }
  } finally {
    sendingCode.value = false
  }
}

// 防抖处理：防止重复提交
let registerDebounceTimer = null

// 方法
const handleRegister = async () => {
  // 防抖：如果正在加载中，直接返回
  if (loading.value) {
    return
  }
  
  // 清除之前的防抖定时器
  if (registerDebounceTimer) {
    clearTimeout(registerDebounceTimer)
  }
  
  // 设置防抖：300ms内只能提交一次
  registerDebounceTimer = setTimeout(async () => {
    try {
      const valid = await registerFormRef.value.validate().catch(err => {
        console.error('表单验证失败:', err)
        ElMessage.warning('请检查表单输入是否正确')
        return false
      })
      
      if (!valid) {
        return
      }
      
      if (registerForm.password !== registerForm.confirmPassword) {
        ElMessage.error('两次输入的密码不一致，请重新输入')
        registerFormRef.value?.validateField('confirmPassword')
        return
      }
      
      // 再次检查loading状态，防止竞态条件
      if (loading.value) {
        return
      }
      
      loading.value = true
    
    // 构建注册请求数据
    const registerData = {
      email: registerForm.email,
      username: registerForm.username,
      password: registerForm.password,
      invite_code: registerForm.inviteCode || null
    }
    
    // 只有在需要邮箱验证时才传递验证码
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
        router.push({
          path: '/login',
          query: {
            username: registerForm.username,
            email: registerForm.email,
            registered: 'true'
          }
        })
      }
    } else {
      ElMessage.error('注册失败，请重试')
    }
    
  } catch (error) {
    console.error('注册错误:', error)
    
    if (error.response?.data) {
      const errorData = error.response.data
      let errorMessage = '注册失败，请重试'
      
      if (errorData.detail) {
        errorMessage = errorData.detail
      } else if (errorData.message) {
        errorMessage = errorData.message
      } else if (typeof errorData === 'string') {
        errorMessage = errorData
      }
      
      ElMessage.error(errorMessage)
      
      if (error.response.status === 400) {
        if (errorMessage.includes('邮箱') || errorMessage.includes('email')) {
          registerFormRef.value?.validateField('email')
        } else if (errorMessage.includes('用户名') || errorMessage.includes('username')) {
          registerFormRef.value?.validateField('username')
        } else if (errorMessage.includes('密码') || errorMessage.includes('password')) {
          registerFormRef.value?.validateField('password')
          registerFormRef.value?.validateField('confirmPassword')
        } else if (errorMessage.includes('验证码') || errorMessage.includes('verification')) {
          registerFormRef.value?.validateField('verificationCode')
        }
      }
    } else if (error.message) {
      if (error.message.includes('validate') || error.message.includes('验证')) {
        ElMessage.warning('请检查表单输入是否正确')
      } else {
        ElMessage.error(error.message)
      }
    } else {
      ElMessage.error('注册失败，请检查网络连接后重试')
    }
  } finally {
    loading.value = false
    registerDebounceTimer = null
  }
  }, 300)
}

// 检查注册是否允许
const checkRegistrationEnabled = async () => {
  try {
    const response = await settingsAPI.getPublicSettings()
    const settings = response.data?.data || response.data || {}
    registrationEnabled.value = settings.allowRegistration !== false
    inviteCodeRequired.value = settings.inviteCodeRequired === true
    
    // 检查邮箱验证开关：后端返回的是布尔值
    const emailVerificationValue = settings.email_verification_required !== undefined 
                                   ? settings.email_verification_required
                                   : (settings.emailVerificationRequired !== undefined 
                                      ? settings.emailVerificationRequired 
                                      : (settings.require_email_verification !== undefined 
                                         ? settings.require_email_verification 
                                         : true))
    // 如果值是布尔值 true 或字符串 "true"，则开启验证
    emailVerificationRequired.value = emailVerificationValue === true || emailVerificationValue === "true"
    
    minPasswordLength.value = settings.minPasswordLength || 8
    registerFormRef.value?.clearValidate()
    
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

onMounted(async () => {
  await checkRegistrationEnabled()

  if (route.query.invite) {
    registerForm.inviteCode = route.query.invite
    await validateInviteCode(route.query.invite)
  }
})

onUnmounted(() => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
})
</script>

<style scoped lang="scss">
.register-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--success-color) 100%);
  padding: 20px;
}

.register-card {
  background: var(--background-color);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  padding: 40px;
  width: 100%;
  max-width: 400px;
}

.register-header {
  text-align: center;
  margin-bottom: 30px;
  
  .logo {
    width: 60px;
    height: 60px;
    margin-bottom: 16px;
  }
  
  :is(h1) {
    margin: 0 0 8px 0;
    color: var(--text-color);
    font-size: 24px;
    font-weight: 600;
  }
  
  :is(p) {
    margin: 0;
    color: var(--text-color-secondary);
    font-size: 14px;
  }
}

.register-form {
  .register-button {
    width: 100%;
    height: 48px;
    font-size: 16px;
    font-weight: 500;
  }
  
  /* 移除所有输入框的圆角和阴影效果，设置为简单长方形 */
  :deep(.el-input__wrapper) {
    border-radius: 0 !important;
    box-shadow: none !important;
    border: 1px solid #dcdfe6 !important;
    background-color: #ffffff !important;
  }
  
  :deep(.el-select .el-input__wrapper) {
    border-radius: 0 !important;
    box-shadow: none !important;
    border: 1px solid #dcdfe6 !important;
    background-color: #ffffff !important;
  }
  
  /* 确保输入框内部所有元素的背景都是透明或白色 */
  :deep(.el-input__inner) {
    border-radius: 0 !important;
    border: none !important;
    box-shadow: none !important;
    background-color: transparent !important;
    background: transparent !important;
  }
  
  /* 确保输入框前缀图标容器背景透明 */
  :deep(.el-input__prefix) {
    background-color: transparent !important;
    background: transparent !important;
  }
  
  /* 确保输入框后缀图标容器背景透明 */
  :deep(.el-input__suffix) {
    background-color: transparent !important;
    background: transparent !important;
  }
  
  /* 确保输入框内部包装器背景透明 */
  :deep(.el-input__wrapper .el-input__inner) {
    background-color: transparent !important;
    background: transparent !important;
  }
  
  :deep(.el-input__wrapper:hover) {
    border-color: #c0c4cc !important;
    box-shadow: none !important;
    background-color: #ffffff !important;
  }
  
  :deep(.el-input__wrapper:hover .el-input__inner) {
    background-color: transparent !important;
    background: transparent !important;
  }
  
  :deep(.el-input__wrapper.is-focus) {
    border-color: #1677ff !important;
    box-shadow: none !important;
    background-color: #ffffff !important;
  }
  
  :deep(.el-input__wrapper.is-focus .el-input__inner) {
    background-color: transparent !important;
    background: transparent !important;
  }
  
  /* 确保聚焦时背景颜色不变 */
  :deep(.el-input__wrapper.is-focus:hover) {
    background-color: #ffffff !important;
  }
  
  :deep(.el-input__wrapper.is-focus:hover .el-input__inner) {
    background-color: transparent !important;
    background: transparent !important;
  }
  
  /* 确保所有状态的背景颜色都是白色 */
  :deep(.el-input__wrapper.is-disabled) {
    background-color: #f5f7fa !important;
  }
  
  /* 确保输入框内部所有可能的背景元素都是透明 */
  :deep(.el-input) {
    background-color: transparent !important;
    background: transparent !important;
  }
  
  /* 确保wrapper内部的所有子元素背景透明，但不影响wrapper本身 */
  :deep(.el-input__wrapper > *) {
    background-color: transparent !important;
    background: transparent !important;
  }
  
  /* 确保wrapper本身背景为白色（优先级更高） */
  :deep(.el-input__wrapper) {
    background-color: #ffffff !important;
    background: #ffffff !important;
  }
  
  /* 确保验证码输入框文字颜色可见 */
  :deep(.verification-code-input .el-input__inner) {
    color: #303133 !important;
    -webkit-text-fill-color: #303133 !important;
  }
  
  :deep(.verification-code-input .el-input__wrapper .el-input__inner) {
    color: #303133 !important;
    -webkit-text-fill-color: #303133 !important;
  }
  
  /* 确保验证码输入框在所有状态下文字都可见 */
  :deep(.verification-code-input .el-input__wrapper.is-focus .el-input__inner) {
    color: #303133 !important;
    -webkit-text-fill-color: #303133 !important;
  }
  
  :deep(.verification-code-input .el-input__wrapper:hover .el-input__inner) {
    color: #303133 !important;
    -webkit-text-fill-color: #303133 !important;
  }
  
  /* 确保输入框可以正常输入 */
  :deep(.verification-code-input .el-input__inner) {
    caret-color: #1677ff !important;
  }
  
  /* 手机端特殊处理：确保验证码输入框文字可见且可输入 */
  @media (max-width: 768px) {
    :deep(.verification-code-input .el-input__inner) {
      color: #303133 !important;
      -webkit-text-fill-color: #303133 !important;
      font-size: 16px !important; /* 防止iOS自动缩放 */
      opacity: 1 !important;
      caret-color: #1677ff !important;
    }
    
    :deep(.verification-code-input .el-input__wrapper .el-input__inner) {
      color: #303133 !important;
      -webkit-text-fill-color: #303133 !important;
      font-size: 16px !important;
      opacity: 1 !important;
    }
    
    :deep(.verification-code-input .el-input__wrapper.is-focus .el-input__inner) {
      color: #303133 !important;
      -webkit-text-fill-color: #303133 !important;
      font-size: 16px !important;
      opacity: 1 !important;
    }
    
    /* 确保输入框在手机端可以正常聚焦和输入 */
    :deep(.verification-code-input) {
      -webkit-user-select: text !important;
      user-select: text !important;
      pointer-events: auto !important;
    }
    
    :deep(.verification-code-input .el-input__wrapper) {
      pointer-events: auto !important;
    }
    
    :deep(.verification-code-input .el-input__inner) {
      pointer-events: auto !important;
      -webkit-user-select: text !important;
      user-select: text !important;
    }
  }
}

.email-input-group {
  display: flex;
  align-items: center;
  gap: 8px;
  
  .email-prefix {
    flex: 2; /* 邮箱前缀输入框占更多空间 */
  }
  
  .email-separator {
    font-size: 16px;
    font-weight: 500;
    color: var(--text-color-secondary);
    min-width: 20px;
    text-align: center;
  }
  
  .email-domain {
    flex: 1; /* 域名选择框占较少空间 */
    min-width: 100px;
  }
}

.verification-code-group {
  display: flex;
  align-items: center;
  gap: 8px;
  
  .verification-code-input {
    flex: 1;
  }
  
  .send-code-button {
    min-width: 120px;
    white-space: nowrap;
  }
}

.register-footer {
  text-align: center;
  margin-top: 24px;
  
  :is(p) {
    margin: 0;
    color: var(--text-color-secondary);
    font-size: 14px;
    
    :is(a) {
      color: var(--primary-color);
      text-decoration: none;
      
      &:hover {
        text-decoration: underline;
      }
    }
  }
}

.form-tip {
  margin-top: 8px;
  font-size: 12px;
  line-height: 1.5;
  padding: 0 4px;
}

// 响应式设计
@media (max-width: 480px) {
  .register-card {
    padding: 24px;
    margin: 10px;
  }
  
  .register-header h1 {
    font-size: 20px;
  }
  
  /* 手机端验证码输入框优化 */
  .verification-code-group {
    gap: 6px;
    
    .verification-code-input {
      flex: 2; /* 增加输入框占比，让它更长 */
      min-width: 0; /* 允许缩小 */
    }
    
    .send-code-button {
      min-width: 80px; /* 减小最小宽度 */
      max-width: 100px; /* 减小最大宽度 */
      font-size: 13px; /* 稍微减小字体 */
      padding: 0 10px; /* 减小内边距 */
      white-space: nowrap;
      flex-shrink: 0; /* 防止按钮被压缩 */
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
    background: transparent !important;
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
</style> 