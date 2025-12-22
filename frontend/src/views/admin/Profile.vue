<template>
  <div class="admin-profile-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <h2>个人资料</h2>
          <p>管理您的账户信息和密码</p>
        </div>
      </template>

      <el-tabs v-model="activeTab" type="border-card">
        <el-tab-pane label="基本信息" name="basic">
          <el-form
            ref="basicFormRef"
            :model="basicForm"
            :rules="basicRules"
            label-width="120px"
            class="profile-form"
          >
            <el-form-item label="用户名">
              <el-input v-model="basicForm.username" disabled />
              <small class="form-tip">用户名不可修改</small>
            </el-form-item>
            
            <el-form-item label="邮箱地址">
              <el-input v-model="basicForm.email" disabled />
              <small class="form-tip">邮箱地址不可修改</small>
            </el-form-item>
            
            <el-form-item label="显示名称" prop="display_name">
              <el-input v-model="basicForm.display_name" placeholder="请输入显示名称" />
            </el-form-item>
            
            <el-form-item label="头像">
              <el-upload
                class="avatar-uploader"
                :action="uploadUrl"
                :show-file-list="false"
                :on-success="handleAvatarSuccess"
                :before-upload="beforeAvatarUpload"
                :headers="uploadHeaders"
                accept="image/*"
              >
                <img v-if="basicForm.avatar_url" :src="basicForm.avatar_url" class="avatar" />
                <el-icon v-else class="avatar-uploader-icon"><Plus /></el-icon>
              </el-upload>
              <small class="form-tip">支持 JPG、PNG 格式，文件大小不超过 2MB</small>
            </el-form-item>
            
            <el-form-item label="手机号码" prop="phone">
              <el-input v-model="basicForm.phone" placeholder="请输入手机号码" />
            </el-form-item>
            
            <el-form-item label="个人简介" prop="bio">
              <el-input
                v-model="basicForm.bio"
                type="textarea"
                :rows="3"
                placeholder="请输入个人简介"
              />
            </el-form-item>
            
            <el-form-item>
              <el-button type="primary" @click="saveBasicInfo" :loading="basicLoading">
                保存基本信息
              </el-button>
              <el-button @click="resetBasicForm">重置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="修改密码" name="password">
          <el-form
            ref="passwordFormRef"
            :model="passwordForm"
            :rules="passwordRules"
            label-width="120px"
            class="profile-form"
          >
            <el-form-item label="当前密码" prop="current_password">
              <el-input
                v-model="passwordForm.current_password"
                type="password"
                placeholder="请输入当前密码"
                show-password
              />
            </el-form-item>
            
            <el-form-item label="新密码" prop="new_password">
              <el-input
                v-model="passwordForm.new_password"
                type="password"
                placeholder="请输入新密码"
                show-password
              />
              <small class="form-tip">密码长度至少8位，包含字母和数字</small>
            </el-form-item>
            
            <el-form-item label="确认新密码" prop="confirm_password">
              <el-input
                v-model="passwordForm.confirm_password"
                type="password"
                placeholder="请再次输入新密码"
                show-password
              />
            </el-form-item>
            
            <el-form-item>
              <el-button type="primary" @click="changePassword" :loading="passwordLoading">
                修改密码
              </el-button>
              <el-button @click="resetPasswordForm">重置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="安全设置" name="security">
          <div class="security-section">
            <h3>账户安全</h3>
            
            <el-form label-width="120px" class="profile-form">
                <el-form-item label="登录通知">
                <el-switch v-model="securityForm.login_notification" @change="toggleLoginNotification" />
                <small class="form-tip">在新设备登录时发送邮件通知</small>
              </el-form-item>
              
              <el-form-item label="通知邮箱" v-if="securityForm.login_notification">
                <el-input 
                  v-model="securityForm.notification_email" 
                  placeholder="请输入接收登录通知的邮箱地址"
                  type="email"
                />
                <small class="form-tip">登录通知将发送到此邮箱</small>
              </el-form-item>
              
              <el-form-item v-if="securityForm.login_notification">
                <el-button 
                  type="primary" 
                  @click="saveNotificationEmail"
                  :loading="securityLoading"
                  size="small"
                >
                  保存通知邮箱
                </el-button>
              </el-form-item>
              
              <el-form-item label="会话超时">
                <el-select v-model="securityForm.session_timeout" @change="updateSessionTimeout">
                  <el-option label="30分钟" value="30" />
                  <el-option label="1小时" value="60" />
                  <el-option label="2小时" value="120" />
                  <el-option label="4小时" value="240" />
                  <el-option label="永不超时" value="0" />
                </el-select>
                <small class="form-tip">设置登录会话的超时时间</small>
              </el-form-item>
            </el-form>
          </div>

          <div class="security-section">
            <h3>登录历史</h3>
            <el-table :data="loginHistory" style="width: 100%" v-loading="loginHistoryLoading">
              <el-table-column prop="login_time" label="登录时间" width="180">
                <template #default="{ row }">
                  {{ formatDate(row.login_time) }}
                </template>
              </el-table-column>
              <el-table-column prop="ip_address" label="IP地址/地区" width="200">
                <template #default="{ row }">
                  <div style="display: flex; flex-direction: column; gap: 4px;">
                    <el-tag type="info" size="small">{{ row.ip_address || '未知' }}</el-tag>
                    <el-tag 
                      v-if="getLocationText(row.location, row.ip_address)" 
                      type="success" 
                      size="small"
                    >
                      {{ getLocationText(row.location, row.ip_address) }}
                    </el-tag>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="设备信息" min-width="200">
                <template #default="{ row }">
                  <el-tooltip :content="row.user_agent || '未知'" placement="top">
                    <span>{{ getDeviceInfo(row.user_agent) }}</span>
                  </el-tooltip>
                </template>
              </el-table-column>
              <el-table-column prop="login_status" label="状态" width="100">
                <template #default="{ row }">
                  <el-tag :type="row.login_status === 'success' ? 'success' : 'danger'">
                    {{ row.login_status === 'success' ? '成功' : '失败' }}
                  </el-tag>
                </template>
              </el-table-column>
            </el-table>
            <div v-if="loginHistory.length === 0 && !loginHistoryLoading" style="text-align: center; padding: 20px; color: #999;">
              暂无登录历史记录
            </div>
          </div>
        </el-tab-pane>
        <el-tab-pane label="通知设置" name="notifications">
          <el-form label-width="120px" class="profile-form">
            <el-form-item label="邮件通知">
              <el-switch v-model="notificationForm.email_enabled" @change="toggleEmailNotification" />
              <small class="form-tip">接收重要的系统通知邮件</small>
            </el-form-item>
            
            <el-form-item label="系统通知">
              <el-switch v-model="notificationForm.system_notification" @change="toggleSystemNotification" />
              <small class="form-tip">接收系统维护、更新等通知</small>
            </el-form-item>
            
            <el-form-item label="安全通知">
              <el-switch v-model="notificationForm.security_notification" @change="toggleSecurityNotification" />
              <small class="form-tip">接收安全相关的通知</small>
            </el-form-item>
            
            <el-form-item label="通知频率">
              <el-select v-model="notificationForm.frequency" @change="updateNotificationFrequency">
                <el-option label="实时" value="realtime" />
                <el-option label="每小时" value="hourly" />
                <el-option label="每天" value="daily" />
                <el-option label="每周" value="weekly" />
              </el-select>
              <small class="form-tip">设置通知的发送频率</small>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { useAuthStore } from '@/store/auth'
import { adminAPI } from '@/utils/api'
import { formatLocation } from '@/utils/location'
import router from '@/router'
import { secureStorage } from '@/utils/secureStorage'

// 获取Cookie的工具函数
function getCookie(name) {
  const value = `; ${document.cookie}`
  const parts = value.split(`; ${name}=`)
  if (parts.length === 2) return parts.pop().split(';').shift()
  return null
}

export default {
  name: 'AdminProfile',
  components: {
    Plus
  },
  setup() {
    const activeTab = ref('basic')
    const basicFormRef = ref()
    const passwordFormRef = ref()
    
    const basicLoading = ref(false)
    const passwordLoading = ref(false)
    const securityLoading = ref(false)
    
    const uploadUrl = '/api/v1/admin/upload'
    
    const authStore = useAuthStore()
    
    // 上传请求头
    const uploadHeaders = computed(() => {
      const headers = {}
      const token = secureStorage.get('admin_token')
      if (token) {
        headers['Authorization'] = `Bearer ${token}`
      }
      const csrfToken = getCookie('csrf_token')
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken
      }
      return headers
    })

    const basicForm = reactive({
      username: '',
      email: '',
      display_name: '',
      avatar_url: '',
      phone: '',
      bio: ''
    })
    const passwordForm = reactive({
      current_password: '',
      new_password: '',
      confirm_password: ''
    })
    const securityForm = reactive({
      login_notification: true,
      notification_email: '',
      session_timeout: '120'
    })
    const notificationForm = reactive({
      email_enabled: true,
      system_notification: true,
      security_notification: true,
      frequency: 'realtime'
    })
    const loginHistory = ref([])
    const loginHistoryLoading = ref(false)
    const basicRules = {
      display_name: [
        { required: true, message: '请输入显示名称', trigger: 'blur' },
        { min: 2, max: 20, message: '显示名称长度在 2 到 20 个字符', trigger: 'blur' }
      ],
      phone: [
        { pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号码', trigger: 'blur' }
      ],
      bio: [
        { max: 200, message: '个人简介不能超过200个字符', trigger: 'blur' }
      ]
    }

    const passwordRules = {
      current_password: [
        { required: true, message: '请输入当前密码', trigger: 'blur' }
      ],
      new_password: [
        { required: true, message: '请输入新密码', trigger: 'blur' },
        { min: 8, message: '密码长度至少8位', trigger: 'blur' },
        { pattern: /^(?=.*[A-Za-z])(?=.*\d)/, message: '密码必须包含字母和数字', trigger: 'blur' }
      ],
      confirm_password: [
        { required: true, message: '请确认新密码', trigger: 'blur' },
        {
          validator: (rule, value, callback) => {
            if (value !== passwordForm.new_password) {
              callback(new Error('两次输入的密码不一致'))
            } else {
              callback()
            }
          },
          trigger: 'blur'
        }
      ]
    }
    const loadBasicInfo = async () => {
      try {
        const response = await adminAPI.getProfile()
        let data = null
        if (response && response.data) {
          if (response.data.success !== false && response.data.data) {
            data = response.data.data
          } else if (response.data) {
            data = response.data
          }
        } else if (response) {
          data = response
        }
        
        if (data) {
          Object.assign(basicForm, {
            username: data.username || '',
            email: data.email || '',
            display_name: data.display_name || '',
            avatar_url: data.avatar_url || '',
            phone: data.phone || '',
            bio: data.bio || ''
          })
          } else {
          ElMessage.error('获取个人信息失败：数据格式错误')
        }
      } catch (error) {
        ElMessage.error(`加载基本信息失败: ${error.message || '未知错误'}`)
      }
    }
    const saveBasicInfo = async () => {
      try {
        await basicFormRef.value.validate()
        basicLoading.value = true
        
        const response = await adminAPI.updateProfile(basicForm)
        let success = false
        let message = '保存失败'
        
        if (response && response.data) {
          if (response.data.success !== false) {
            success = true
            message = response.data.message || '基本信息保存成功'
            // 更新表单数据
            if (response.data.data) {
              Object.assign(basicForm, {
                display_name: response.data.data.display_name || basicForm.display_name,
                avatar_url: response.data.data.avatar_url || response.data.data.avatar || basicForm.avatar_url,
                phone: response.data.data.phone || basicForm.phone,
                bio: response.data.data.bio || basicForm.bio
              })
            }
          } else {
            message = response.data.message || '保存失败'
          }
        } else if (response) {
          if (response.success !== false) {
            success = true
            message = response.message || '基本信息保存成功'
          } else {
            message = response.message || '保存失败'
          }
        }
        
        if (success) {
          ElMessage.success(message)
          // 重新加载基本信息以确保数据同步
          await loadBasicInfo()
          if (authStore && authStore.updateUser) {
            authStore.updateUser(basicForm)
          }
        } else {
          ElMessage.error(message)
        }
      } catch (error) {
        const errorMessage = error.response?.data?.message || 
                           error.response?.data?.detail || 
                           error.message || 
                           '保存失败'
        ElMessage.error(errorMessage)
      } finally {
        basicLoading.value = false
      }
    }
    const resetBasicForm = () => {
      loadBasicInfo()
    }
    const changePassword = async () => {
      try {
        await passwordFormRef.value.validate()
        passwordLoading.value = true
        
        const response = await adminAPI.changePassword({
          current_password: passwordForm.current_password,
          new_password: passwordForm.new_password
        })
        
        let success = false
        let message = '密码修改失败'
        
        if (response && response.data) {
          success = response.data.success !== false
          message = response.data.message || (success ? '密码修改成功' : '密码修改失败')
        } else if (response) {
          success = response.success !== false
          message = response.message || (success ? '密码修改成功' : '密码修改失败')
        }
        
        if (success) {
          ElMessage.success('密码修改成功，请重新登录')
          Object.assign(passwordForm, {
            current_password: '',
            new_password: '',
            confirm_password: ''
          })
          passwordFormRef.value?.clearValidate()
          setTimeout(() => {
            authStore.logout()
            router.push('/admin/login')
          }, 1500)
        } else {
          ElMessage.error(message)
        }
      } catch (error) {
        const errorMessage = error.response?.data?.message || error.response?.data?.detail || error.message || '修改密码失败'
        ElMessage.error(errorMessage)
      } finally {
        passwordLoading.value = false
      }
    }
    const resetPasswordForm = () => {
      Object.assign(passwordForm, {
        current_password: '',
        new_password: '',
        confirm_password: ''
      })
      passwordFormRef.value?.clearValidate()
    }
    const handleAvatarSuccess = (response) => {
      if (response && response.success) {
        basicForm.avatar_url = response.data?.url || response.url || response.data?.avatar_url || ''
        ElMessage.success('头像上传成功')
        // 自动保存头像
        saveBasicInfo()
      } else if (response && response.data) {
        if (response.data.url) {
          basicForm.avatar_url = response.data.url
          ElMessage.success('头像上传成功')
          saveBasicInfo()
        } else if (response.data.avatar_url) {
          basicForm.avatar_url = response.data.avatar_url
          ElMessage.success('头像上传成功')
          saveBasicInfo()
        } else {
          ElMessage.error('头像上传失败：响应格式错误')
        }
      } else {
        ElMessage.error('头像上传失败')
      }
    }
    const beforeAvatarUpload = (file) => {
      const isJPG = file.type === 'image/jpeg'
      const isPNG = file.type === 'image/png'
      const isLt2M = file.size / 1024 / 1024 < 2

      if (!isJPG && !isPNG) {
        ElMessage.error('头像只能是 JPG 或 PNG 格式!')
        return false
      }
      if (!isLt2M) {
        ElMessage.error('头像大小不能超过 2MB!')
        return false
      }
      return true
    }
    const toggleLoginNotification = async (value) => {
      try {
        await adminAPI.updateSecuritySettings({
          login_notification: value,
          notification_email: securityForm.notification_email
        })
        ElMessage.success('设置已保存')
      } catch (error) {
        ElMessage.error('设置失败')
        securityForm.login_notification = !value
      }
    }
    const saveNotificationEmail = async () => {
      if (securityForm.login_notification && !securityForm.notification_email) {
        ElMessage.warning('请先输入通知邮箱地址')
        return
      }
      
      // 验证邮箱格式
      const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
      if (securityForm.notification_email && !emailPattern.test(securityForm.notification_email)) {
        ElMessage.warning('请输入正确的邮箱地址')
        return
      }
      
      securityLoading.value = true
      try {
        const response = await adminAPI.updateSecuritySettings({
          login_notification: securityForm.login_notification,
          notification_email: securityForm.notification_email
        })
        const success = response.success !== false && (response.data?.success !== false)
        const message = response.message || response.data?.message || '通知邮箱已保存'
        
        if (success) {
          ElMessage.success('通知邮箱已保存')
        } else {
          ElMessage.error(message || '保存失败')
        }
      } catch (error) {
        const errorMessage = error.response?.data?.message || 
                           error.response?.data?.detail || 
                           error.message || 
                           '保存失败'
        ElMessage.error(errorMessage)
      } finally {
        securityLoading.value = false
      }
    }
    const updateSessionTimeout = async (value) => {
      try {
        await adminAPI.updateSecuritySettings({
          session_timeout: value
        })
        ElMessage.success('会话超时设置已保存')
      } catch (error) {
        ElMessage.error('设置失败')
      }
    }
    const toggleEmailNotification = async (value) => {
      try {
        const response = await adminAPI.updateNotificationSettings({
          email_enabled: value
        })
        ElMessage.success('邮件通知设置已保存')
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '设置失败')
        notificationForm.email_enabled = !value
      }
    }
    const toggleSystemNotification = async (value) => {
      try {
        const response = await adminAPI.updateNotificationSettings({
          system_notification: value
        })
        ElMessage.success('系统通知设置已保存')
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '设置失败')
        notificationForm.system_notification = !value
      }
    }
    const toggleSecurityNotification = async (value) => {
      try {
        const response = await adminAPI.updateNotificationSettings({
          security_notification: value
        })
        ElMessage.success('安全通知设置已保存')
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '设置失败')
        notificationForm.security_notification = !value
      }
    }
    const updateNotificationFrequency = async (value) => {
      try {
        const response = await adminAPI.updateNotificationSettings({
          frequency: value
        })
        ElMessage.success('通知频率设置已保存')
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '设置失败')
      }
    }
    const loadLoginHistory = async () => {
      loginHistoryLoading.value = true
      try {
        const response = await adminAPI.getLoginHistory()
        console.log('[管理员个人资料] 登录历史响应:', response)
        let data = null
        
        // 处理不同的响应格式
        if (response && response.data) {
          if (response.data.success !== false) {
            if (Array.isArray(response.data.data)) {
              data = response.data.data
            } else if (response.data.data && Array.isArray(response.data.data)) {
              data = response.data.data
            } else {
              data = response.data.data
            }
          }
        } else if (response && Array.isArray(response)) {
          data = response
        }
        
        console.log('[管理员个人资料] 解析后的数据:', data)
        
        if (Array.isArray(data)) {
          loginHistory.value = data.map(item => ({
            login_time: item.login_time || '',
            ip_address: item.ip_address || '',
            location: item.location || '',
            country: item.country || '',
            city: item.city || '',
            user_agent: item.user_agent || '',
            login_status: item.login_status || item.status || 'success'
          }))
          console.log('[管理员个人资料] 处理后的登录历史:', loginHistory.value)
        } else if (data && data.login_history && Array.isArray(data.login_history)) {
          loginHistory.value = data.login_history
        } else {
          console.log('[管理员个人资料] 数据格式不正确，设置为空数组')
          loginHistory.value = []
        }
      } catch (error) {
        console.error('[管理员个人资料] 加载登录历史失败:', error)
        console.error('[管理员个人资料] 错误详情:', error.response?.data)
        ElMessage.error('加载登录历史失败: ' + (error.response?.data?.message || error.message || '未知错误'))
        loginHistory.value = []
      } finally {
        loginHistoryLoading.value = false
      }
    }

    // 加载安全设置
    const loadSecuritySettings = async () => {
      try {
        const response = await adminAPI.getSecuritySettings()
        if (response.success || response.data) {
          const data = response.data || response
          Object.assign(securityForm, {
            login_notification: data.login_notification !== undefined ? data.login_notification : true,
            notification_email: data.notification_email || '',
            session_timeout: data.session_timeout || '120'
          })
        }
      } catch (error) {
        }
    }
    const loadNotificationSettings = async () => {
      try {
        const response = await adminAPI.getNotificationSettings()
        let data = null
        if (response && response.data) {
          if (response.data.success !== false && response.data.data) {
            data = response.data.data
          } else if (response.data) {
            data = response.data
          }
        } else if (response) {
          data = response
        }
        
        if (data) {
          Object.assign(notificationForm, {
            email_enabled: data.email_enabled !== undefined ? data.email_enabled : true,
            system_notification: data.system_notification !== undefined ? data.system_notification : true,
            security_notification: data.security_notification !== undefined ? data.security_notification : true,
            frequency: data.frequency || 'realtime'
          })
          }
      } catch (error) {
        }
    }
    const formatDate = (dateString) => {
      if (!dateString) return ''
      const date = new Date(dateString)
      return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
      })
    }

    // 获取位置文本
    const getLocationText = (location, ipAddress) => {
      if (location) {
        return formatLocation(location)
      }
      // 如果没有location，检查是否为本地IP或内网IP
      if (ipAddress) {
        if (ipAddress === '127.0.0.1' || ipAddress === '::1' || ipAddress === 'localhost') {
          return '本地'
        }
        // 检查是否为内网IP（简单判断）
        if (ipAddress.startsWith('192.168.') || ipAddress.startsWith('10.') || ipAddress.startsWith('172.')) {
          return '内网'
        }
      }
      return ''
    }
    
    // 获取设备信息（简化显示）
    const getDeviceInfo = (userAgent) => {
      if (!userAgent) return '未知设备'
      const ua = userAgent.toLowerCase()
      if (ua.includes('mobile') || ua.includes('android') || ua.includes('iphone')) {
        return '移动设备'
      } else if (ua.includes('windows')) {
        return 'Windows'
      } else if (ua.includes('mac')) {
        return 'Mac'
      } else if (ua.includes('linux')) {
        return 'Linux'
      } else {
        return '其他设备'
      }
    }
    onMounted(() => {
      loadBasicInfo()
      loadSecuritySettings()
      loadNotificationSettings()
      loadLoginHistory()
    })

    return {
      activeTab,
      basicFormRef,
      passwordFormRef,
      basicLoading,
      passwordLoading,
      securityLoading,
      basicForm,
      passwordForm,
      securityForm,
      notificationForm,
      loginHistory,
      loginHistoryLoading,
      basicRules,
      passwordRules,
      uploadUrl,
      uploadHeaders,
      saveBasicInfo,
      resetBasicForm,
      changePassword,
      resetPasswordForm,
      handleAvatarSuccess,
      beforeAvatarUpload,
      toggleLoginNotification,
      saveNotificationEmail,
      updateSessionTimeout,
      toggleEmailNotification,
      toggleSystemNotification,
      toggleSecurityNotification,
      updateNotificationFrequency,
      formatDate,
      getLocationText,
      getDeviceInfo
    }
  }
}
</script>

<style scoped>
.admin-profile-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header h2 {
  margin: 0;
  color: #333;
  font-size: 1.5rem;
}

.card-header :is(p) {
  margin: 0;
  color: #666;
  font-size: 0.9rem;
}

.profile-form {
  max-width: 600px;
  margin-top: 20px;
}

.form-tip {
  color: #999;
  font-size: 12px;
  margin-top: 4px;
  display: block;
}

.avatar-uploader {
  text-align: center;
}

.avatar-uploader .avatar {
  width: 100px;
  height: 100px;
  border-radius: 50%;
  object-fit: cover;
}

.avatar-uploader .avatar-uploader-icon {
  font-size: 28px;
  color: #8c939d;
  width: 100px;
  height: 100px;
  line-height: 100px;
  text-align: center;
  border: 1px dashed #d9d9d9;
  border-radius: 50%;
}

.security-section {
  margin-bottom: 30px;
}

.security-section h3 {
  color: #333;
  margin-bottom: 20px;
  font-size: 1.2rem;
  border-bottom: 1px solid #eee;
  padding-bottom: 10px;
}

@media (max-width: 768px) {
  .admin-profile-container {
    padding: 10px;
  }
  
  .card-header {
    flex-direction: column;
    gap: 10px;
    align-items: flex-start;
  }
  
  .profile-form {
    max-width: 100%;
  }
}

:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
  padding: 0 !important;
}

:deep(.el-select .el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
  padding: 0 !important;
}

:deep(.el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background-color: transparent !important;
  padding: 0 11px !important;
}

:deep(.el-input__prefix),
:deep(.el-input__suffix) {
  background-color: transparent !important;
  border: none !important;
}

:deep(.el-input__wrapper:hover) {
  border-color: #c0c4cc !important;
  box-shadow: none !important;
}

:deep(.el-input__wrapper.is-focus) {
  border-color: #1677ff !important;
  box-shadow: none !important;
}

:deep(.el-textarea__inner) {
  border-radius: 0 !important;
  border: 1px solid #dcdfe6 !important;
  box-shadow: none !important;
}
</style>
