<template>
  <div class="profile-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <h1>个人资料</h1>
      <p>管理您的账户信息</p>
    </div>

    <div class="profile-content">
      <!-- 基本信息 -->
      <el-card class="profile-card">
        <template #header>
          <div class="card-header">
            <i class="el-icon-user"></i>
            基本信息
          </div>
        </template>

        <el-form
          ref="profileFormRef"
          :model="profileForm"
          :rules="profileRules"
          :label-width="isMobile ? '0' : '120px'"
          class="profile-form"
        >
          <el-form-item prop="username" :label="!isMobile ? '用户名' : ''">
            <template v-if="isMobile">
              <div class="mobile-label">
                <span>用户名</span>
              </div>
            </template>
            <el-input 
              v-model="profileForm.username" 
              disabled
              placeholder="用户名"
            >
              <template #prepend>
                <i class="el-icon-user"></i>
              </template>
            </el-input>
            <div class="form-tip">用户名不可修改</div>
          </el-form-item>

          <el-form-item prop="email" :label="!isMobile ? '邮箱' : ''">
            <template v-if="isMobile">
              <div class="mobile-label">
                <span>邮箱</span>
              </div>
            </template>
            <el-input 
              v-model="profileForm.email" 
              disabled
              placeholder="邮箱"
            >
              <template #prepend>
                <i class="el-icon-message"></i>
              </template>
            </el-input>
            <div class="form-tip">邮箱不可修改</div>
          </el-form-item>

          <el-form-item :label="!isMobile ? '注册时间' : ''">
            <template v-if="isMobile">
              <div class="mobile-label">
                <span>注册时间</span>
              </div>
            </template>
            <el-input 
              :value="formatTime(userInfo.created_at)" 
              disabled
            >
              <template #prepend>
                <i class="el-icon-time"></i>
              </template>
            </el-input>
          </el-form-item>

          <el-form-item :label="!isMobile ? '最后登录' : ''">
            <template v-if="isMobile">
              <div class="mobile-label">
                <span>最后登录</span>
              </div>
            </template>
            <el-input 
              :value="formatTime(userInfo.last_login)" 
              disabled
            >
              <template #prepend>
                <i class="el-icon-time"></i>
              </template>
            </el-input>
          </el-form-item>

          <el-form-item :label="!isMobile ? '账户状态' : ''">
            <template v-if="isMobile">
              <div class="mobile-label">
                <span>账户状态</span>
              </div>
            </template>
            <el-tag :type="getAccountStatusType(userInfo)">
              {{ getAccountStatusText(userInfo) }}
            </el-tag>
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 修改密码 -->
      <el-card class="password-card">
        <template #header>
          <div class="card-header">
            <i class="el-icon-lock"></i>
            修改密码
          </div>
        </template>

        <el-form
          ref="passwordFormRef"
          :model="passwordForm"
          :rules="passwordRules"
          :label-width="isMobile ? '0' : '120px'"
          class="password-form"
        >
          <el-form-item prop="oldPassword" :label="!isMobile ? '当前密码' : ''">
            <template v-if="isMobile">
              <div class="mobile-label">
                <span>当前密码</span>
              </div>
            </template>
            <el-input 
              v-model="passwordForm.oldPassword" 
              type="password"
              placeholder="请输入当前密码"
              show-password
              clearable
              autocomplete="current-password"
              class="password-input"
            >
              <template #prepend>
                <i class="el-icon-lock"></i>
              </template>
            </el-input>
          </el-form-item>

          <el-form-item prop="newPassword" :label="!isMobile ? '新密码' : ''">
            <template v-if="isMobile">
              <div class="mobile-label">
                <span>新密码</span>
              </div>
            </template>
            <el-input 
              v-model="passwordForm.newPassword" 
              type="password"
              placeholder="请输入新密码"
              show-password
              clearable
              autocomplete="new-password"
              class="password-input"
            >
              <template #prepend>
                <i class="el-icon-lock"></i>
              </template>
            </el-input>
            <div class="form-tip">密码长度不能少于6位</div>
          </el-form-item>

          <el-form-item prop="confirmPassword" :label="!isMobile ? '确认密码' : ''">
            <template v-if="isMobile">
              <div class="mobile-label">
                <span>确认密码</span>
              </div>
            </template>
            <el-input 
              v-model="passwordForm.confirmPassword" 
              type="password"
              placeholder="请再次输入新密码"
              show-password
              clearable
              autocomplete="new-password"
              class="password-input"
            >
              <template #prepend>
                <i class="el-icon-lock"></i>
              </template>
            </el-input>
          </el-form-item>

          <el-form-item>
            <el-button 
              type="primary" 
              @click="changePassword"
              :loading="passwordLoading"
            >
              修改密码
            </el-button>
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 账户安全 -->
      <el-card class="security-card">
        <template #header>
          <div class="card-header">
            <i class="el-icon-shield"></i>
            账户安全
          </div>
        </template>

        <div class="security-items">
          <div class="security-item">
            <div class="security-info">
              <div class="security-title">
                <i class="el-icon-time"></i>
                登录记录
              </div>
              <div class="security-desc">
                最后登录时间：{{ formatTime(userInfo.last_login) }}
              </div>
            </div>
            <div class="security-action">
              <el-button 
                type="info" 
                size="small"
                @click="viewLoginHistory"
              >
                查看登录历史
              </el-button>
            </div>
          </div>
        </div>
      </el-card>

      <!-- 订阅信息 -->
      <el-card class="subscription-card" v-if="subscriptionInfo">
        <template #header>
          <div class="card-header">
            <i class="el-icon-link"></i>
            订阅信息
          </div>
        </template>

        <div class="subscription-info">
          <div class="info-item">
            <span class="label">剩余时长：</span>
            <span class="value">{{ subscriptionInfo.remainingDays || 0 }} 天</span>
          </div>
          <div class="info-item">
            <span class="label">到期时间：</span>
            <span class="value">{{ subscriptionInfo.expiryDate || '未设置' }}</span>
          </div>
          <div class="info-item">
            <span class="label">设备限制：</span>
            <span class="value">{{ subscriptionInfo.currentDevices || 0 }}/{{ subscriptionInfo.maxDevices || 0 }} 个</span>
          </div>
          <div class="info-item">
            <span class="label">订阅状态：</span>
            <span class="value">
              <el-tag :type="subscriptionInfo.status === 'active' ? (subscriptionInfo.isExpiring ? 'warning' : 'success') : 'danger'">
                {{ subscriptionInfo.status === 'active' ? (subscriptionInfo.isExpiring ? '即将到期' : '正常') : '已过期' }}
              </el-tag>
            </span>
          </div>
        </div>

        <div class="subscription-actions">
          <el-button type="primary" @click="$router.push('/subscription')">
            管理订阅
          </el-button>
          <el-button type="success" @click="$router.push('/packages')">
            续费订阅
          </el-button>
        </div>
      </el-card>
    </div>

    <!-- 登录历史对话框 -->
    <el-dialog
      v-model="loginHistoryDialogVisible"
      title="登录历史"
      width="90%"
      :close-on-click-modal="false"
      class="login-history-dialog"
    >
      <div v-if="loginHistoryLoading" class="loading-container">
        <el-skeleton :rows="5" animated />
      </div>

      <el-table 
        v-else-if="loginHistory.length > 0"
        :data="loginHistory" 
        stripe
        style="width: 100%"
        max-height="400"
      >
        <el-table-column prop="login_time" label="登录时间" width="180">
          <template #default="scope">
            {{ formatTime(scope.row.login_time) }}
          </template>
        </el-table-column>
        
        <el-table-column prop="ip_address" label="IP地址/地区" width="180">
          <template #default="scope">
            <div style="display: flex; flex-direction: column; gap: 4px;">
              <el-tag type="info" size="small">{{ scope.row.ip_address || '未知' }}</el-tag>
              <el-tag 
                v-if="getLocationText(scope.row.location, scope.row.ip_address)" 
                type="success" 
                size="small"
              >
                {{ getLocationText(scope.row.location, scope.row.ip_address) }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column prop="user_agent" label="设备信息" min-width="150">
          <template #default="scope">
            <el-tooltip :content="scope.row.user_agent" placement="top">
              <span class="user-agent-text">
                {{ getDeviceInfo(scope.row.user_agent) }}
              </span>
            </el-tooltip>
          </template>
        </el-table-column>
        
        <el-table-column prop="login_status" label="状态" width="80">
          <template #default="scope">
            <el-tag :type="scope.row.login_status === 'success' ? 'success' : 'danger'" size="small">
              {{ scope.row.login_status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-else description="暂无登录记录" />

      <template #footer>
        <el-button @click="loginHistoryDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

  </div>
</template>

<script>
import { ref, reactive, onMounted, computed, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/store/auth'
import { userAPI, subscriptionAPI, authAPI, api } from '@/utils/api'
import { formatLocation } from '@/utils/location'
import dayjs from 'dayjs'

export default {
  name: 'Profile',
  setup() {
    const router = useRouter()
    const authStore = useAuthStore()
    
    // 检测移动端
    const windowWidth = ref(window.innerWidth)
    const isMobile = computed(() => windowWidth.value <= 768)
    
    const handleResize = () => {
      windowWidth.value = window.innerWidth
    }
    
    onMounted(() => {
      window.addEventListener('resize', handleResize)
    })
    
    onUnmounted(() => {
      window.removeEventListener('resize', handleResize)
    })
    const passwordLoading = ref(false)
    const emailLoading = ref(false)
    const profileFormRef = ref(null)
    const passwordFormRef = ref(null)
    const loginHistoryDialogVisible = ref(false)
    const loginHistoryLoading = ref(false)
    const loginHistory = ref([])
    
    const userInfo = ref({
      username: '',
      email: '',
      is_verified: false,
      last_login: null,
      created_at: null,
      status: 'active'
    })
    const subscriptionInfo = ref(null)

    const profileForm = reactive({
      username: '',
      email: ''
    })

    const passwordForm = reactive({
      oldPassword: '',
      newPassword: '',
      confirmPassword: ''
    })
    
    // 初始化时尝试从authStore获取基本信息
    const initUserInfo = () => {
      const authUser = authStore.user
      if (authUser) {
        userInfo.value.username = authUser.username || ''
        userInfo.value.email = authUser.email || ''
        profileForm.username = userInfo.value.username
        profileForm.email = userInfo.value.email
        }
    }

    // 表单验证规则
    const profileRules = {
      username: [
        { required: true, message: '请输入用户名', trigger: 'blur' }
      ],
      email: [
        { required: true, message: '请输入邮箱', trigger: 'blur' },
        { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
      ]
    }

    const passwordRules = {
      oldPassword: [
        { required: true, message: '请输入当前密码', trigger: 'blur' }
      ],
      newPassword: [
        { required: true, message: '请输入新密码', trigger: 'blur' },
        { min: 6, message: '密码长度不能少于6位', trigger: 'blur' },
        {
          validator: (rule, value, callback) => {
            // 安全检查：新密码不能等于旧密码
            if (value && passwordForm.oldPassword && value === passwordForm.oldPassword) {
              callback(new Error('新密码不能与当前密码相同'))
            } else {
              callback()
            }
          },
          trigger: 'blur'
        }
      ],
      confirmPassword: [
        { required: true, message: '请确认新密码', trigger: 'blur' },
        {
          validator: (rule, value, callback) => {
            if (value !== passwordForm.newPassword) {
              callback(new Error('两次输入的密码不一致'))
            } else {
              callback()
            }
          },
          trigger: 'blur'
        }
      ]
    }

    // 获取用户信息
    const fetchUserInfo = async () => {
      try {
        // 使用 /users/me 端点获取用户信息
        const response = await api.get('/users/me')
        
        // 检查响应结构
        let data = null
        if (response && response.data) {
          if (response.data.success && response.data.data) {
            data = response.data.data
          } else if (response.data.data) {
            data = response.data.data
          } else if (response.data) {
            data = response.data
          }
        }
        
        if (data) {
          // 更新用户信息 - 使用后端返回的字段名
          userInfo.value = {
            username: data.username || '',
            email: data.email || '',
            is_verified: data.is_verified !== undefined ? data.is_verified : false,
            last_login: data.last_login || data.lastLogin || data.last_login_time || null,
            created_at: data.created_at || data.createdAt || null,
            status: data.is_active !== undefined ? (data.is_active ? 'active' : 'inactive') : 'active'
          }
          
          // 更新表单数据
          profileForm.username = userInfo.value.username || ''
          profileForm.email = userInfo.value.email || ''
        } else {
          // 尝试从authStore获取基本信息
          const authUser = authStore.user
          if (authUser) {
            userInfo.value.username = authUser.username || ''
            userInfo.value.email = authUser.email || ''
            profileForm.username = userInfo.value.username
            profileForm.email = userInfo.value.email
          } else {
            ElMessage.error('获取用户信息失败：无法解析响应数据')
          }
        }
      } catch (error) {
        // 尝试从authStore获取基本信息
        const authUser = authStore.user
        if (authUser) {
          userInfo.value.username = authUser.username || ''
          userInfo.value.email = authUser.email || ''
          profileForm.username = userInfo.value.username
          profileForm.email = userInfo.value.email
        } else {
          ElMessage.error(`获取用户信息失败: ${error.response?.data?.message || error.message || '未知错误'}`)
        }
      }
    }

    // 获取订阅信息
    const fetchSubscriptionInfo = async () => {
      try {
        const response = await subscriptionAPI.getUserSubscription()
        if (response.data && response.data.success) {
          const data = response.data.data
          // 确保字段名正确映射
          subscriptionInfo.value = {
            remainingDays: data.remainingDays || data.remaining_days || 0,
            expiryDate: data.expiryDate || data.expiry_date || '未设置',
            currentDevices: data.currentDevices || data.current_devices || data.online_devices || 0,
            maxDevices: data.maxDevices || data.max_devices || data.device_limit || 0,
            isExpiring: data.isExpiring || data.is_expiring || false,
            status: data.status || 'expired'
          }
          } else {
          // 用户可能没有订阅，设置为默认值
          subscriptionInfo.value = {
            remainingDays: 0,
            expiryDate: '未订阅',
            currentDevices: 0,
            maxDevices: 0,
            isExpiring: false,
            status: 'expired'
          }
        }
      } catch (error) {
        // 用户可能没有订阅，设置为默认值
        subscriptionInfo.value = {
          remainingDays: 0,
          expiryDate: '未订阅',
          currentDevices: 0,
          maxDevices: 0,
          isExpiring: false,
          status: 'expired'
        }
      }
    }

    // 修改密码
    const changePassword = async () => {
      if (!passwordFormRef.value) {
        ElMessage.error('表单引用未初始化')
        return
      }
      
      // 安全检查：新密码不能等于旧密码
      if (passwordForm.newPassword && passwordForm.oldPassword && 
          passwordForm.newPassword === passwordForm.oldPassword) {
        ElMessage.error('新密码不能与当前密码相同')
        return
      }
      
      try {
        await passwordFormRef.value.validate()
      } catch (error) {
        return
      }
      
      // 防抖：如果正在加载中，直接返回
      if (passwordLoading.value) {
        return
      }
      
      passwordLoading.value = true
      try {
        // 使用正确的API端点
        const response = await userAPI.changePassword({
          current_password: passwordForm.oldPassword,
          new_password: passwordForm.newPassword
        })
        if (response.data && response.data.success) {
          ElMessage.success(response.data.message || '密码修改成功')
          
          // 清空表单
          passwordForm.oldPassword = ''
          passwordForm.newPassword = ''
          passwordForm.confirmPassword = ''
          
          // 重置表单验证状态
          if (passwordFormRef.value) {
            passwordFormRef.value.resetFields()
          }
        } else {
          ElMessage.error(response.data?.message || '密码修改失败：响应格式错误')
        }
        
      } catch (error) {
        const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '未知错误'
        ElMessage.error(`密码修改失败: ${errorMsg}`)
      } finally {
        passwordLoading.value = false
      }
    }

    // 获取登录历史
    const fetchLoginHistory = async () => {
      loginHistoryLoading.value = true
      try {
        const response = await userAPI.getLoginHistory()
        if (response.data && response.data.success) {
          const data = response.data.data
          if (Array.isArray(data)) {
            loginHistory.value = data.map(item => ({
              login_time: item.login_time || '',
              ip_address: item.ip_address || '',
              location: item.location || '',
              country: item.country || '',
              city: item.city || '',
              user_agent: item.user_agent || '',
              login_status: item.login_status || 'success'
            }))
          } else if (data.logins && Array.isArray(data.logins)) {
            loginHistory.value = data.logins.map(item => ({
              login_time: item.login_time || '',
              ip_address: item.ip_address || '',
              location: item.location || '',
              country: item.country || '',
              city: item.city || '',
              user_agent: item.user_agent || '',
              login_status: item.login_status || 'success'
            }))
          } else {
            loginHistory.value = []
          }
        } else {
          ElMessage.error('获取登录历史失败：响应格式错误')
        }
      } catch (error) {
        ElMessage.error(`获取登录历史失败: ${error.message || '未知错误'}`)
      } finally {
        loginHistoryLoading.value = false
      }
    }

    // 获取设备信息
    const getDeviceInfo = (userAgent) => {
      if (!userAgent) return '未知设备'
      if (userAgent.includes('Mobile')) {
        return '移动设备'
      } else if (userAgent.includes('Windows')) {
        return 'Windows设备'
      } else if (userAgent.includes('Mac')) {
        return 'Mac设备'
      } else if (userAgent.includes('Linux')) {
        return 'Linux设备'
      } else {
        return '其他设备'
      }
    }

    // 查看登录历史
    const viewLoginHistory = () => {
      loginHistoryDialogVisible.value = true
      fetchLoginHistory()
    }


    // 获取位置文本
    const getLocationText = (location, ipAddress) => {
      if (location) {
        return formatLocation(location)
      }
      if (ipAddress && ipAddress !== '未知' && ipAddress !== '127.0.0.1' && ipAddress !== '::1') {
        return '解析中...'
      }
      return ''
    }

    // 格式化时间
    const formatTime = (time) => {
      if (!time || time === 'null' || time === 'None' || time === null || time === undefined) {
        return '未知'
      }
      try {
        // 尝试使用dayjs解析
        const date = dayjs(time)
        if (date.isValid()) {
          return date.format('YYYY-MM-DD HH:mm:ss')
        }
        // 如果dayjs解析失败，尝试直接返回字符串
        if (typeof time === 'string' && time.trim() !== '') {
          return time
        }
        return '未知'
      } catch (error) {
        return '未知'
      }
    }

    // 获取账户状态类型
    const getAccountStatusType = (userInfo) => {
      if (!userInfo || !userInfo.status) return 'info'
      
      switch (userInfo.status) {
        case 'active':
          return 'success'
        case 'inactive':
        case 'disabled':
          return 'danger'
        case 'pending':
          return 'warning'
        default:
          return 'info'
      }
    }

    // 获取账户状态文本
    const getAccountStatusText = (userInfo) => {
      if (!userInfo || !userInfo.status) return '未知'
      
      switch (userInfo.status) {
        case 'active':
          return '正常'
        case 'inactive':
        case 'disabled':
          return '已禁用'
        case 'pending':
          return '待激活'
        default:
          return '未知'
      }
    }

    onMounted(() => {
      // 先初始化基本信息
      initUserInfo()
      // 然后获取完整信息
      fetchUserInfo()
      fetchSubscriptionInfo()
    })

    return {
      userInfo,
      subscriptionInfo,
      profileForm,
      passwordForm,
      profileFormRef,
      passwordFormRef,
      profileRules,
      passwordRules,
      passwordLoading,
      emailLoading,
      isMobile,
      loginHistoryDialogVisible,
      loginHistoryLoading,
      loginHistory,
      changePassword,
      viewLoginHistory,
      fetchLoginHistory,
      getDeviceInfo,
      formatTime,
      getLocationText,
      getAccountStatusType,
      getAccountStatusText
    }
  }
}
</script>

<style scoped>
.profile-container {
  padding: 0;
  max-width: none;
  margin: 0;
  width: 100%;
}

.page-header {
  margin-bottom: 1rem;
  text-align: center;
  
  @media (max-width: 768px) {
    margin-bottom: 0.75rem;
  }
}

.page-header h1 {
  color: #1677ff;
  font-size: 1.5rem;
  margin-bottom: 0.25rem;
  
  @media (max-width: 768px) {
    font-size: 1.25rem;
  }
}

.page-header :is(p) {
  color: #666;
  font-size: 0.875rem;
  
  @media (max-width: 768px) {
    font-size: 0.8125rem;
  }
}

.profile-content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  
  @media (max-width: 768px) {
    gap: 0.75rem;
  }
}

.profile-card,
.password-card,
.security-card,
.subscription-card {
  border-radius: 8px;
  box-shadow: 0 1px 6px rgba(0, 0, 0, 0.05);
  
  :deep(.el-card__header) {
    padding: 12px 16px;
    font-size: 0.9375rem;
  }
  
  :deep(.el-card__body) {
    padding: 12px 16px;
  }
  
  @media (max-width: 768px) {
    :deep(.el-card__header) {
      padding: 10px 12px;
      font-size: 0.875rem;
    }
    
    :deep(.el-card__body) {
      padding: 10px 12px;
    }
  }
}

.card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
  font-size: 0.9375rem;
  
  @media (max-width: 768px) {
    font-size: 0.875rem;
    gap: 0.375rem;
  }
}

.form-tip {
  font-size: 0.8125rem;
  color: #666;
  margin-top: 0.375rem;
  
  @media (max-width: 768px) {
    font-size: 0.75rem;
    margin-top: 0.25rem;
  }
}

.mobile-label {
  display: block;
  width: 100%;
  font-size: 14px;
  font-weight: 600;
  color: #333;
  margin-bottom: 8px;
  padding: 0;
}

.security-items {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  
  @media (max-width: 768px) {
    gap: 0.5rem;
  }
}

.security-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem;
  background: #f8f9fa;
  border-radius: 6px;
  
  @media (max-width: 768px) {
    padding: 0.625rem;
  }
}

.security-info {
  flex: 1;
}

.security-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
  color: #333;
  margin-bottom: 0.25rem;
  font-size: 0.9375rem;
  
  @media (max-width: 768px) {
    font-size: 0.875rem;
  }
}

.security-desc {
  color: #666;
  font-size: 0.8125rem;
  
  @media (max-width: 768px) {
    font-size: 0.75rem;
  }
}

.security-action {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  
  @media (max-width: 768px) {
    gap: 0.5rem;
  }
}

.subscription-info {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.75rem;
  margin-bottom: 1rem;
  
  @media (max-width: 768px) {
    gap: 0.5rem;
    margin-bottom: 0.75rem;
  }
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.375rem 0;
  
  @media (max-width: 768px) {
    padding: 0.25rem 0;
  }
}

.info-item .label {
  color: #666;
  font-weight: 500;
  font-size: 0.875rem;
  
  @media (max-width: 768px) {
    font-size: 0.8125rem;
  }
}

.info-item .value {
  color: #333;
  font-weight: 600;
  font-size: 0.9375rem;
  
  @media (max-width: 768px) {
    font-size: 0.875rem;
  }
}

.subscription-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: center;
  
  @media (max-width: 768px) {
    gap: 0.5rem;
  }
}

@media (max-width: 768px) {
  .profile-container {
    padding: 0;
  }
  
  .page-header {
    margin-bottom: 0.75rem;
    padding: 0 12px;
  }
  
  .profile-card,
  .password-card,
  .security-card,
  .subscription-card {
    border-radius: 0;
    margin: 0 -12px 0.75rem -12px;
    box-shadow: none;
    border-left: none;
    border-right: none;
  }
  
  .profile-card:first-child {
    margin-top: 0;
  }
  
  .el-form {
    .el-form-item {
      margin-bottom: 18px;
      
      .el-form-item__label {
        font-size: 14px;
        margin-bottom: 8px;
        width: 100% !important;
        text-align: left;
        padding: 0;
        line-height: 1.5;
        font-weight: 600;
        color: #333;
      }
      
      .el-form-item__content {
        width: 100%;
        
        .el-input,
        .el-select {
          width: 100%;
          
          :deep(.el-input__wrapper) {
            height: 48px;
            border-radius: 12px;
            border: 1px solid #dcdfe6;
            transition: all 0.3s ease;
          }
          
          :deep(.el-input__wrapper:hover) {
            border-color: #c0c4cc;
          }
          
          :deep(.el-input__wrapper.is-focus) {
            border-color: var(--theme-primary, #409EFF);
            box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.1);
          }
          
          :deep(.el-input__inner) {
            font-size: 15px;
            height: 48px;
            line-height: 48px;
            padding: 0 12px;
          }
        }
      }
    }
  }
  
  .mobile-label {
    font-size: 14px;
    font-weight: 600;
    color: #333;
    margin-bottom: 10px;
    display: block;
  }
  
  .form-tip {
    font-size: 13px;
    margin-top: 8px;
    color: #999;
    padding-left: 4px;
  }
  
  .profile-form,
  .password-form {
    :deep(.el-form-item) {
      .el-form-item__label {
        width: 100% !important;
        margin-bottom: 10px;
        padding: 0;
      }
      
      .el-form-item__content {
        width: 100%;
        margin-left: 0 !important;
      }
    }
  }
  
  @media (max-width: 768px) {
    .profile-form,
    .password-form {
      :deep(.el-form-item) {
        display: flex;
        flex-direction: column;
        align-items: stretch;
        
        .el-form-item__label {
          order: 1;
          width: 100% !important;
          margin-bottom: 10px;
          padding: 0;
          text-align: left;
        }
        
        .el-form-item__content {
          order: 2;
          width: 100% !important;
          margin-left: 0 !important;
          flex: 1;
        }
      }
    }
    
    .mobile-label {
      display: block;
      width: 100%;
      margin-bottom: 10px;
      font-size: 14px;
      font-weight: 600;
      color: #333;
    }
  }
  
  .el-button {
    border-radius: 16px;
    padding: 14px 24px;
    font-weight: 600;
    font-size: 15px;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    
    &:active {
      transform: scale(0.98);
    }
  }
  
  .security-items {
    gap: 0.5rem;
  }
  
  .security-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
    padding: 0.625rem;
    
    .security-info {
      width: 100%;
      
      .security-title {
        font-size: 0.875rem;
        margin-bottom: 4px;
      }
      
      .security-desc {
        font-size: 0.75rem;
      }
    }
    
      .security-action {
        width: 100%;
        justify-content: flex-start;
        flex-wrap: wrap;
        gap: 6px;
      
      .el-tag {
        margin-right: 0;
      }
      
      .el-button {
        width: 100%;
        margin: 0;
      }
    }
  }
  
  .subscription-info {
    grid-template-columns: 1fr;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
    
    .info-item {
      padding: 0.625rem;
      border-bottom: 1px solid #f0f0f0;
      
      &:last-child {
        border-bottom: none;
      }
      
      .label {
        font-size: 0.8125rem;
        display: block;
        margin-bottom: 3px;
      }
      
      .value {
        font-size: 0.875rem;
        display: block;
      }
    }
  }
  
  .subscription-actions {
    flex-direction: column;
    gap: 0.5rem;
    margin-top: 0;
    
    .el-button {
      width: 100%;
      margin: 0;
      height: 44px;
      border-radius: 12px;
      font-size: 0.875rem;
      font-weight: 600;
    }
  }
  
  .submit-btn {
    width: 100%;
    height: 44px;
    border-radius: 12px;
    font-size: 0.875rem;
    font-weight: 600;
    background: linear-gradient(135deg, var(--theme-primary, #409EFF) 0%, var(--theme-primary, #409EFF) 100%);
    box-shadow: 0 3px 10px rgba(64, 158, 255, 0.25);
    
    &:active {
      transform: scale(0.98);
      box-shadow: 0 2px 6px rgba(64, 158, 255, 0.2);
    }
  }
  
  .login-history-dialog {
    :deep(.el-dialog) {
      border-radius: 16px;
    }
    
    :deep(.el-dialog__header) {
      padding: 20px 20px 10px;
      border-bottom: 1px solid #f0f0f0;
    }
    
    :deep(.el-dialog__body) {
      padding: 20px;
    }
    
    .loading-container {
      padding: 20px;
    }
    
    .user-agent-text {
      display: inline-block;
      max-width: 150px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }
  
  @media (max-width: 768px) {
    .login-history-dialog {
      :deep(.el-dialog) {
        width: 95% !important;
        margin: 5vh auto !important;
        max-height: 90vh;
      }
      
      :deep(.el-dialog__body) {
        padding: 16px;
        max-height: calc(90vh - 120px);
        overflow-y: auto;
      }
      
      :deep(.el-table) {
        font-size: 13px;
      }
      
      :deep(.el-table th),
      :deep(.el-table td) {
        padding: 8px 4px;
      }
    }
  }
}

@media (max-width: 480px) {
  .profile-card,
  .password-card,
  .security-card,
  .subscription-card {
    :deep(.el-card__header) {
      padding: 8px 10px;
    }
    
    :deep(.el-card__body) {
      padding: 8px 10px;
    }
  }
}

/* 移除所有输入框的圆角和阴影效果，设置为简单长方形，只保留外部边框 */
:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
  pointer-events: auto !important;
}

:deep(.el-select .el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
  pointer-events: auto !important;
}

:deep(.el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background-color: transparent !important;
  pointer-events: auto !important;
}

:deep(.el-input__wrapper:hover) {
  border-color: #c0c4cc !important;
  box-shadow: none !important;
}

:deep(.el-input__wrapper.is-focus) {
  border-color: #1677ff !important;
  box-shadow: none !important;
}

/* 确保密码输入框可编辑 */
:deep(.el-input__wrapper.is-disabled) {
  pointer-events: none !important;
}

:deep(.el-input.is-disabled .el-input__inner) {
  pointer-events: none !important;
}

/* 密码输入框特殊样式 - 确保可以输入和显示 */
:deep(.password-input) {
  pointer-events: auto !important;
}

:deep(.password-input .el-input__wrapper) {
  pointer-events: auto !important;
  cursor: text !important;
}

:deep(.password-input .el-input__inner) {
  pointer-events: auto !important;
  cursor: text !important;
  color: #606266 !important;
}

:deep(.password-input .el-input__wrapper:not(.is-disabled)) {
  pointer-events: auto !important;
}

:deep(.password-input .el-input__wrapper:not(.is-disabled) .el-input__inner) {
  pointer-events: auto !important;
  color: #606266 !important;
}

/* 确保密码输入框的显示/隐藏按钮可用 */
:deep(.password-input .el-input__suffix) {
  pointer-events: auto !important;
}

:deep(.password-input .el-input__suffix .el-input__password) {
  pointer-events: auto !important;
  cursor: pointer !important;
}

/* 确保密码输入框在聚焦时可以正常输入 */
:deep(.password-input.is-focus .el-input__wrapper) {
  pointer-events: auto !important;
}

:deep(.password-input.is-focus .el-input__inner) {
  pointer-events: auto !important;
}

/* 移除 prepend 和 append 的边框，只保留外部边框 */
:deep(.el-input-group__prepend),
:deep(.el-input-group__append) {
  border-radius: 0 !important;
  border: none !important;
  background-color: #f5f7fa !important;
  pointer-events: none !important;
}

:deep(.el-input-group__prepend) {
  border-right: 1px solid #dcdfe6 !important;
}

:deep(.el-input-group__append) {
  border-left: 1px solid #dcdfe6 !important;
}
</style> 