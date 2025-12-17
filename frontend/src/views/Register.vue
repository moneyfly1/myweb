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
            <span v-if="inviteCodeInfo.success" style="color: #67c23a;">
              ✓ 邀请码有效，注册后可获得 {{ inviteCodeInfo.data?.invitee_reward || 0 }} 元奖励
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
import { settingsAPI } from '@/utils/api'

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

// 计算属性
const settings = computed(() => settingsStore)

// 表单验证规则
const registerRules = computed(() => ({
  email: [
    { required: true, message: '请选择邮箱类型', trigger: 'change' }
  ],
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 20, message: '用户名长度在 2 到 20 个字符', trigger: 'blur' },
    { 
      pattern: /^[a-zA-Z0-9_]+$/, 
      message: '用户名只能包含字母、数字和下划线', 
      trigger: 'blur' 
    }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: minPasswordLength.value, max: 50, message: `密码长度在 ${minPasswordLength.value} 到 50 个字符`, trigger: 'blur' },
    { 
      pattern: /^(?=.*[A-Za-z])(?=.*\d)/, 
      message: '密码必须包含字母和数字', 
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
    { min: 6, max: 6, message: '验证码为6位数字', trigger: 'blur' }
  ] : [],
  inviteCode: inviteCodeRequired.value ? [
    { required: true, message: '请输入邀请码', trigger: 'blur' }
  ] : []
}))

// 计算是否可以发送验证码
const canSendCode = computed(() => {
  return registerForm.emailPrefix && registerForm.emailDomain
})

// 发送验证码
const handleSendVerificationCode = async () => {
  // 验证邮箱是否完整
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

// 方法
const handleRegister = async () => {
  try {
    await registerFormRef.value.validate()
    
    loading.value = true
    
    const response = await authAPI.register({
      email: registerForm.email,
      username: registerForm.username,
      password: registerForm.password,
      verification_code: registerForm.verificationCode,
      invite_code: registerForm.inviteCode || null
    })
    
    // 注册成功，跳转到登录页面
    if (response.data) {
      // 注册成功，显示友好提示并跳转到登录页面
      ElMessage.success('注册成功！请登录')
      
      // 跳转到登录页面，并传递用户名和邮箱，让浏览器提示保存密码
      router.push({
        path: '/login',
        query: {
          username: registerForm.username,
          email: registerForm.email,
          registered: 'true'
        }
      })
    } else {
      ElMessage.error('注册失败，请重试')
    }
    
  } catch (error) {
    if (error.response?.data?.detail) {
      ElMessage.error(error.response.data.detail)
    } else {
      ElMessage.error('注册失败，请重试')
    }
  } finally {
    loading.value = false
  }
}

// 检查注册是否允许
const checkRegistrationEnabled = async () => {
  try {
    const response = await settingsAPI.getPublicSettings()
    const settings = response.data?.data || response.data || {}
    // 从系统设置中获取注册开关状态
    registrationEnabled.value = settings.allowRegistration !== false
    inviteCodeRequired.value = settings.inviteCodeRequired === true
    emailVerificationRequired.value = settings.emailVerificationRequired !== false
    minPasswordLength.value = settings.minPasswordLength || 8
    // 更新表单验证规则（触发重新计算）
    registerFormRef.value?.clearValidate()
    
    // 如果注册被禁用，显示提示
    if (!registrationEnabled.value) {
      ElMessage.warning('注册功能已禁用，请联系管理员')
    }
  } catch (error) {
    // 如果检查失败，默认允许注册（向后兼容）
    registrationEnabled.value = true
    inviteCodeRequired.value = false
    emailVerificationRequired.value = true
    minPasswordLength.value = 8
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

// 监听邀请码变化（防抖）
let validateTimeout = null
watch(() => registerForm.inviteCode, (newCode) => {
  if (validateTimeout) {
    clearTimeout(validateTimeout)
  }
  
  if (newCode && newCode.trim()) {
    // 延迟验证，避免频繁请求
    validateTimeout = setTimeout(() => {
      validateInviteCode(newCode)
    }, 500)
  } else {
    inviteCodeInfo.value = null
  }
})

// 组件挂载时检查注册状态和URL参数中的邀请码
onMounted(async () => {
  await checkRegistrationEnabled()
  
  // 从URL参数中获取邀请码
  if (route.query.invite) {
    registerForm.inviteCode = route.query.invite
    // 验证邀请码
    await validateInviteCode(route.query.invite)
  }
})

// 组件卸载时清理定时器
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
    
    a {
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
}
</style> 