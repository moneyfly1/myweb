<template>
  <div class="forgot-container">
    <div class="forgot-card">
      <div class="forgot-header">
        <img v-if="settings.siteLogo" :src="settings.siteLogo" :alt="settings.siteName" class="logo" />
        <h1>{{ settings.siteName }}</h1>
        <p>输入您的邮箱地址，我们将发送验证码</p>
      </div>

      <el-form
        ref="forgotFormRef"
        :model="forgotForm"
        :rules="forgotRules"
        label-width="0"
        class="forgot-form"
      >
        <el-form-item prop="email">
          <el-input
            v-model="forgotForm.email"
            placeholder="邮箱地址"
            prefix-icon="Message"
            size="large"
          />
        </el-form-item>

        <el-form-item prop="verificationCode">
          <div class="verification-code-group">
            <el-input
              v-model="forgotForm.verificationCode"
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

        <el-form-item prop="newPassword">
          <el-input
            v-model="forgotForm.newPassword"
            type="password"
            placeholder="新密码"
            prefix-icon="Lock"
            size="large"
            show-password
          />
        </el-form-item>

        <el-form-item prop="confirmPassword">
          <el-input
            v-model="forgotForm.confirmPassword"
            type="password"
            placeholder="确认新密码"
            prefix-icon="Lock"
            size="large"
            show-password
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            class="forgot-button"
            :loading="loading"
            @click="handleResetPassword"
          >
            重置密码
          </el-button>
        </el-form-item>
      </el-form>

      <div class="forgot-footer">
        <p>
          <router-link to="/login">返回登录</router-link>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useSettingsStore } from '@/store/settings'
import { api } from '@/utils/api'

const router = useRouter()
const settingsStore = useSettingsStore()

const loading = ref(false)
const forgotFormRef = ref()
const sendingCode = ref(false)
const countdown = ref(0)
let countdownTimer = null

const forgotForm = reactive({
  email: '',
  verificationCode: '',
  newPassword: '',
  confirmPassword: ''
})

const settings = computed(() => settingsStore)

const canSendCode = computed(() => {
  return forgotForm.email && forgotForm.email.includes('@')
})

const forgotRules = computed(() => ({
  email: [
    { required: true, message: '请输入邮箱地址', trigger: 'blur' },
    { 
      type: 'email', 
      message: '邮箱格式不正确，请输入有效的邮箱地址', 
      trigger: 'blur' 
    }
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
    { 
      min: 8, 
      max: 50, 
      message: '密码长度至少 8 位，最多 50 位', 
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
    { required: true, message: '请确认新密码', trigger: 'blur' },
    { 
      validator: (rule, value, callback) => {
        if (!value) {
          callback(new Error('请确认新密码'))
          return
        }
        if (value !== forgotForm.newPassword) {
          callback(new Error('两次输入密码不一致，请重新输入'))
        } else {
          callback()
        }
      }, 
      trigger: 'blur' 
    }
  ]
}))

const handleSendVerificationCode = async () => {
  if (!forgotForm.email) {
    ElMessage.warning('请先填写邮箱地址')
    return
  }
  
  sendingCode.value = true
  
  try {
    const response = await api.post('/auth/forgot-password', {
      email: forgotForm.email
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
    // 改进错误提示，显示更友好的错误信息
    if (error.response?.data) {
      const errorData = error.response.data
      // 优先显示 detail 字段的错误信息
      if (errorData.detail) {
        ElMessage.error(errorData.detail)
      } else if (errorData.message) {
        ElMessage.error(errorData.message)
      } else {
        ElMessage.error('发送验证码失败，请检查邮箱地址是否正确')
      }
    } else if (error.message) {
      ElMessage.error(error.message)
    } else {
      ElMessage.error('发送验证码失败，请重试')
    }
  } finally {
    sendingCode.value = false
  }
}

const handleResetPassword = async () => {
  try {
    await forgotFormRef.value.validate()
    
    loading.value = true
    
    const response = await api.post('/auth/reset-password', {
      email: forgotForm.email,
      verification_code: forgotForm.verificationCode,
      new_password: forgotForm.newPassword
    })
    
    ElMessage.success('密码重置成功！')
    
    setTimeout(() => {
      router.push('/login')
    }, 1500)
    
  } catch (error) {
    // 改进错误提示，显示更友好的错误信息
    if (error.response?.data) {
      const errorData = error.response.data
      // 优先显示 detail 字段的错误信息
      if (errorData.detail) {
        ElMessage.error(errorData.detail)
      } else if (errorData.message) {
        ElMessage.error(errorData.message)
      } else {
        ElMessage.error('重置密码失败，请检查输入信息')
      }
    } else if (error.message) {
      ElMessage.error(error.message)
    } else {
      ElMessage.error('重置密码失败，请重试')
    }
  } finally {
    loading.value = false
  }
}

// 组件卸载时清理定时器
onUnmounted(() => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
})
</script>

<style lang="scss" scoped>
.forgot-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--success-color) 100%);
  padding: 20px;
}

.forgot-card {
  background: var(--background-color);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  padding: 40px;
  width: 100%;
  max-width: 400px;
}

.forgot-header {
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

.forgot-form {
  .forgot-button {
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

.verification-code-group {
  display: flex;
  align-items: center;
  gap: 8px;
  
  .verification-code-input {
    flex: 2;
    min-width: 0; // 允许缩小
  }
  
  .send-code-button {
    flex: 1;
    min-width: 100px;
    max-width: 140px;
    white-space: nowrap;
    font-size: 14px;
    padding: 0 12px;
  }
}

.forgot-footer {
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

// 响应式设计
@media (max-width: 768px) {
  .forgot-container {
    padding: 10px;
    min-height: 100vh;
    padding-top: 20px;
  }
  
  .forgot-card {
    padding: 24px 16px;
    margin: 0;
    max-width: 100%;
    border-radius: 8px;
  }
  
  .forgot-header {
    margin-bottom: 24px;
    
    :is(h1) {
      font-size: 20px;
      margin-bottom: 8px;
    }
    
    :is(p) {
      font-size: 13px;
    }
  }
  
  /* 输入框手机端优化 */
  :deep(.el-input__inner) {
    height: 48px !important; /* 手机端增大高度，防止iOS自动缩放 */
    font-size: 16px !important; /* 16px防止iOS自动缩放 */
    padding: 0 14px;
  }
  
  :deep(.el-input--large .el-input__inner) {
    height: 48px !important;
    font-size: 16px !important;
  }
  
  /* 按钮手机端优化 */
  .forgot-button {
    width: 100%;
    min-height: 48px;
    font-size: 16px;
    font-weight: 500;
  }
  
  .verification-code-group {
    gap: 6px;
    
    .verification-code-input {
      flex: 2.5; // 手机端验证码输入框更宽
    }
    
    .send-code-button {
      flex: 1;
      min-width: 90px;
      max-width: 120px;
      min-height: 48px;
      font-size: 14px;
      padding: 0 10px;
      white-space: nowrap;
    }
  }
  
  .forgot-footer {
    margin-top: 20px;
    
    :is(a) {
      font-size: 14px;
      padding: 8px 0;
      min-height: 44px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
    }
  }
}
</style>
