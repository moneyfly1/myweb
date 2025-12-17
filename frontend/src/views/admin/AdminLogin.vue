<template>
  <div class="admin-login-container">
    <div class="admin-login-box">
      <div class="admin-login-header">
        <h1>CBoard Modern</h1>
        <p>管理员登录</p>
      </div>
      
      <form class="admin-login-form" @submit.prevent="handleLogin">
        <div class="form-item">
          <input
            v-model="loginForm.username"
            type="text"
            placeholder="管理员用户名或邮箱"
            class="login-input"
            autocomplete="username"
            name="username"
            id="admin-username"
            required
          />
        </div>
        
        <div class="form-item">
          <input
            v-model="loginForm.password"
            type="password"
            placeholder="密码"
            class="login-input"
            autocomplete="current-password"
            name="password"
            id="admin-password"
            required
            @keyup.enter="handleLogin"
          />
        </div>
        
        <div class="form-item">
          <button
            type="submit"
            :disabled="loading"
            class="login-button"
          >
            {{ loading ? '登录中...' : '管理员登录' }}
          </button>
        </div>
      </form>
      
      <div class="admin-login-actions">
        <el-link type="primary" @click="$router.push('/login')">
          返回用户登录
        </el-link>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, reactive, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/store/auth'

export default {
  name: 'AdminLogin',
  setup() {
    const router = useRouter()
    const authStore = useAuthStore()
    
    const loginForm = reactive({
      username: '',
      password: ''
    })
    
    const loading = ref(false)
    
    const handleLogin = async () => {
      loading.value = true

      try {
        // 清除可能存在的旧缓存（双重保障）
        if (typeof window !== 'undefined') {
          const { secureStorage } = await import('@/utils/secureStorage')
          secureStorage.clear()
        }
        
        // 使用管理员登录方法（强制使用管理员路径）
        const result = await authStore.adminLogin(loginForm)

        if (result.success) {
          // 验证是否为管理员
          if (!authStore.isAdmin) {
            ElMessage.error('该账户不是管理员，请使用用户登录页面')
            authStore.logout()
            return
          }
          
          ElMessage.success('管理员登录成功')
          await nextTick()
          await router.push('/admin/dashboard')
        } else {
          ElMessage.error(result.message || '登录失败')
        }
      } catch (error) {
        ElMessage.error('登录失败，请重试')
      } finally {
        loading.value = false
      }
    }
    
    return {
      loginForm,
      loading,
      handleLogin
    }
  }
}
</script>

<style scoped>
.admin-login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  padding: 20px;
}

.admin-login-box {
  background: white;
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.1);
  padding: 40px;
  width: 100%;
  max-width: 400px;
}

.admin-login-header {
  text-align: center;
  margin-bottom: 30px;
}

.admin-login-header h1 {
  color: #f5576c;
  font-size: 28px;
  margin-bottom: 8px;
  font-weight: 600;
}

.admin-login-header p {
  color: #666;
  font-size: 14px;
  margin: 0;
}

.admin-login-form {
  margin-top: 20px;
}

.form-item {
  margin-bottom: 20px;
}

.login-input {
  width: 100%;
  height: 44px;
  padding: 0 16px;
  border: 1px solid #dcdfe6;
  border-radius: 0;
  font-size: 16px;
  outline: none;
  transition: border-color 0.3s;
  box-shadow: none !important;
  background-color: #ffffff !important;
}

.login-input:focus {
  border-color: #f5576c;
  box-shadow: none !important;
  background-color: #ffffff !important;
}

.login-input::placeholder {
  color: #a8abb2;
}

.login-button {
  width: 100%;
  height: 44px;
  font-size: 16px;
  font-weight: 500;
  background: #f5576c;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: background-color 0.3s;
}

.login-button:hover:not(:disabled) {
  background: #e0455a;
}

.login-button:disabled {
  background: #a8abb2;
  cursor: not-allowed;
}

.admin-login-actions {
  display: flex;
  justify-content: center;
  margin-top: 20px;
  font-size: 14px;
}

@media (max-width: 480px) {
  .admin-login-box {
    padding: 30px 20px;
  }
  
  .admin-login-header h1 {
    font-size: 24px;
  }
}
</style>

