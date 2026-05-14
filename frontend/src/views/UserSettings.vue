<template>
  <div class="list-container user-settings">
    <div class="page-header">
      <h1>用户设置</h1>
      <p>管理您的账户设置和偏好</p>
    </div>
    <el-row :gutter="20" class="settings-desktop">
      <el-col :span="6">
        <el-card class="settings-menu">
          <template #header>
            <div class="card-header">
              <el-icon><Setting /></el-icon>
              设置分类
            </div>
          </template>
          <el-menu
            :default-active="activeSetting"
            @select="handleSettingSelect"
            class="settings-menu-list"
          >
            <el-menu-item index="profile">
              <el-icon><User /></el-icon>
              <span>个人资料</span>
            </el-menu-item>
            <el-menu-item index="security">
              <el-icon><Lock /></el-icon>
              <span>安全设置</span>
            </el-menu-item>
            <el-menu-item index="notifications">
              <el-icon><Bell /></el-icon>
              <span>通知设置</span>
            </el-menu-item>
            <el-menu-item index="preferences">
              <el-icon><Star /></el-icon>
              <span>偏好设置</span>
            </el-menu-item>
          </el-menu>
        </el-card>
      </el-col>
      <el-col :span="18">
        <el-card v-if="activeSetting === 'profile'" class="setting-content">
          <template #header>
            <div class="card-header">
              <el-icon><User /></el-icon>
              个人资料
            </div>
          </template>
          <el-form :model="profileForm" :rules="profileRules" ref="profileFormRef" label-width="100px">
            <el-form-item label="用户名" prop="username">
              <el-input v-model="profileForm.username" placeholder="请输入用户名"></el-input>
            </el-form-item>
            <el-form-item label="邮箱" prop="email">
              <el-input v-model="profileForm.email" placeholder="请输入邮箱" disabled>
                <template #append>
                  <el-button @click="showEmailChangeDialog">修改</el-button>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item label="昵称" prop="nickname">
              <el-input v-model="profileForm.nickname" placeholder="请输入昵称"></el-input>
            </el-form-item>
            <el-form-item label="头像">
              <el-upload
                class="avatar-uploader"
                action="#"
                :show-file-list="false"
                :before-upload="beforeAvatarUpload"
              >
                <img v-if="profileForm.avatar" :src="profileForm.avatar" class="avatar" />
                <el-icon v-else class="avatar-uploader-icon"><Plus /></el-icon>
              </el-upload>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveProfile" :loading="profileSaving">
                保存修改
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
        <el-card v-if="activeSetting === 'security'" class="setting-content">
          <template #header>
            <div class="card-header">
              <el-icon><Lock /></el-icon>
              安全设置
            </div>
          </template>
          <el-form :model="securityForm" :rules="securityRules" ref="securityFormRef" label-width="100px">
            <el-form-item label="当前密码" prop="currentPassword">
              <el-input v-model="securityForm.currentPassword" type="password" placeholder="请输入当前密码"></el-input>
            </el-form-item>
            <el-form-item label="新密码" prop="newPassword">
              <el-input v-model="securityForm.newPassword" type="password" placeholder="请输入新密码"></el-input>
            </el-form-item>
            <el-form-item label="确认密码" prop="confirmPassword">
              <el-input v-model="securityForm.confirmPassword" type="password" placeholder="请再次输入新密码"></el-input>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="changePassword" :loading="passwordChanging">
                修改密码
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
        <el-card v-if="activeSetting === 'notifications'" class="setting-content">
          <template #header>
            <div class="card-header">
              <el-icon><Bell /></el-icon>
              通知设置
            </div>
          </template>
          <div class="notification-settings">
            <h4>邮件通知</h4>
            <el-switch
              v-model="notificationForm.emailNotifications"
              active-text="启用邮件通知"
              inactive-text="禁用邮件通知"
            ></el-switch>
            <el-divider></el-divider>
            <h4>异常登录/设备告警</h4>
            <p class="setting-hint">当检测到新设备或异地登录时，可通过邮件和站内通知提醒您。关闭后将不再发送此类告警。</p>
            <el-switch
              v-model="notificationForm.abnormalLoginAlert"
              active-text="接收告警通知"
              inactive-text="不接收告警通知"
            ></el-switch>
            <el-divider></el-divider>
            <h4>通知类型</h4>
            <el-checkbox-group v-model="notificationForm.notificationTypes">
              <el-checkbox v-for="item in notificationTypeOptions" :key="item.value" :label="item.value">{{ item.label }}</el-checkbox>
            </el-checkbox-group>
            <el-divider></el-divider>
            <el-button type="primary" @click="saveNotificationSettings" :loading="notificationSaving">
              保存设置
            </el-button>
          </div>
        </el-card>
        <el-card v-if="activeSetting === 'preferences'" class="setting-content">
          <template #header>
            <div class="card-header">
              <el-icon><Star /></el-icon>
              偏好设置
            </div>
          </template>
          <div class="preference-settings">
            <h4>界面设置</h4>
            <el-form label-width="120px">
              <el-form-item label="主题模式">
                <el-radio-group v-model="preferenceForm.theme">
                  <el-radio label="light">浅色主题</el-radio>
                  <el-radio label="dark">深色主题</el-radio>
                  <el-radio label="blue">蓝色主题</el-radio>
                  <el-radio label="green">绿色主题</el-radio>
                  <el-radio label="purple">紫色主题</el-radio>
                  <el-radio label="orange">橙色主题</el-radio>
                  <el-radio label="red">红色主题</el-radio>
                  <el-radio label="cyan">青色主题</el-radio>
                  <el-radio label="luck">Luck主题</el-radio>
                  <el-radio label="aurora">Aurora主题</el-radio>
                  <el-radio label="auto">跟随系统</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="时区">
                <el-select v-model="preferenceForm.timezone" placeholder="选择时区">
                  <el-option label="UTC+8 (北京时间)" value="Asia/Shanghai"></el-option>
                  <el-option label="UTC+0 (格林威治时间)" value="UTC"></el-option>
                </el-select>
              </el-form-item>
            </el-form>
            <el-divider></el-divider>
            <el-button type="primary" @click="savePreferenceSettings" :loading="preferenceSaving">
              保存设置
            </el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>
    <div class="settings-mobile">
      <el-card class="mobile-tabs-card">
        <el-tabs v-model="activeSetting" class="mobile-settings-tabs">
          <el-tab-pane label="个人资料" name="profile">
            <template #label>
              <span class="tab-label"><el-icon><User /></el-icon>个人资料</span>
            </template>
          </el-tab-pane>
          <el-tab-pane label="安全设置" name="security">
            <template #label>
              <span class="tab-label"><el-icon><Lock /></el-icon>安全设置</span>
            </template>
          </el-tab-pane>
          <el-tab-pane label="通知设置" name="notifications">
            <template #label>
              <span class="tab-label"><el-icon><Bell /></el-icon>通知设置</span>
            </template>
          </el-tab-pane>
          <el-tab-pane label="偏好设置" name="preferences">
            <template #label>
              <span class="tab-label"><el-icon><Star /></el-icon>偏好设置</span>
            </template>
          </el-tab-pane>
        </el-tabs>
      </el-card>
      <div class="mobile-settings-content">
        <el-card v-if="activeSetting === 'profile'" class="setting-content mobile-setting-card">
          <template #header>
            <div class="card-header">
              <el-icon><User /></el-icon>
              个人资料
            </div>
          </template>
          <el-form :model="profileForm" :rules="profileRules" ref="profileFormRef" class="mobile-form">
            <el-form-item label="用户名" prop="username">
              <el-input v-model="profileForm.username" placeholder="请输入用户名"></el-input>
            </el-form-item>
            <el-form-item label="邮箱" prop="email">
              <el-input v-model="profileForm.email" placeholder="请输入邮箱" disabled>
                <template #append>
                  <el-button @click="showEmailChangeDialog" size="small">修改</el-button>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item label="昵称" prop="nickname">
              <el-input v-model="profileForm.nickname" placeholder="请输入昵称"></el-input>
            </el-form-item>
            <el-form-item label="头像">
              <el-upload
                class="avatar-uploader"
                action="#"
                :show-file-list="false"
                :before-upload="beforeAvatarUpload"
              >
                <img v-if="profileForm.avatar" :src="profileForm.avatar" class="avatar" />
                <el-icon v-else class="avatar-uploader-icon"><Plus /></el-icon>
              </el-upload>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveProfile" :loading="profileSaving" style="width: 100%">
                保存修改
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
        <el-card v-if="activeSetting === 'security'" class="setting-content mobile-setting-card">
          <template #header>
            <div class="card-header">
              <el-icon><Lock /></el-icon>
              安全设置
            </div>
          </template>
          <el-form :model="securityForm" :rules="securityRules" ref="securityFormRef" class="mobile-form">
            <el-form-item label="当前密码" prop="currentPassword">
              <el-input v-model="securityForm.currentPassword" type="password" placeholder="请输入当前密码"></el-input>
            </el-form-item>
            <el-form-item label="新密码" prop="newPassword">
              <el-input v-model="securityForm.newPassword" type="password" placeholder="请输入新密码"></el-input>
            </el-form-item>
            <el-form-item label="确认密码" prop="confirmPassword">
              <el-input v-model="securityForm.confirmPassword" type="password" placeholder="请再次输入新密码"></el-input>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="changePassword" :loading="passwordChanging" style="width: 100%">
                修改密码
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
        <el-card v-if="activeSetting === 'notifications'" class="setting-content mobile-setting-card">
          <template #header>
            <div class="card-header">
              <el-icon><Bell /></el-icon>
              通知设置
            </div>
          </template>
          <div class="notification-settings">
            <h4>邮件通知</h4>
            <el-switch
              v-model="notificationForm.emailNotifications"
              active-text="启用邮件通知"
              inactive-text="禁用邮件通知"
            ></el-switch>
            <el-divider></el-divider>
            <h4>异常登录/设备告警</h4>
            <p class="setting-hint">当检测到新设备或异地登录时，可通过邮件和站内通知提醒您。关闭后将不再发送此类告警。</p>
            <el-switch
              v-model="notificationForm.abnormalLoginAlert"
              active-text="接收告警通知"
              inactive-text="不接收告警通知"
            ></el-switch>
            <el-divider></el-divider>
            <h4>通知类型</h4>
            <el-checkbox-group v-model="notificationForm.notificationTypes" class="mobile-checkbox-group">
              <el-checkbox v-for="item in notificationTypeOptions" :key="item.value" :label="item.value">{{ item.label }}</el-checkbox>
            </el-checkbox-group>
            <el-divider></el-divider>
            <el-button type="primary" @click="saveNotificationSettings" :loading="notificationSaving" style="width: 100%">
              保存设置
            </el-button>
          </div>
        </el-card>
        <el-card v-if="activeSetting === 'preferences'" class="setting-content mobile-setting-card">
          <template #header>
            <div class="card-header">
              <el-icon><Star /></el-icon>
              偏好设置
            </div>
          </template>
          <div class="preference-settings">
            <h4>界面设置</h4>
            <el-form class="mobile-form">
              <el-form-item label="主题模式">
                <el-radio-group v-model="preferenceForm.theme" class="mobile-radio-group">
                  <el-radio label="light">浅色主题</el-radio>
                  <el-radio label="dark">深色主题</el-radio>
                  <el-radio label="blue">蓝色主题</el-radio>
                  <el-radio label="green">绿色主题</el-radio>
                  <el-radio label="purple">紫色主题</el-radio>
                  <el-radio label="orange">橙色主题</el-radio>
                  <el-radio label="red">红色主题</el-radio>
                  <el-radio label="cyan">青色主题</el-radio>
                  <el-radio label="luck">Luck主题</el-radio>
                  <el-radio label="aurora">Aurora主题</el-radio>
                  <el-radio label="auto">跟随系统</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="时区">
                <el-select v-model="preferenceForm.timezone" placeholder="选择时区" style="width: 100%">
                  <el-option label="UTC+8 (北京时间)" value="Asia/Shanghai"></el-option>
                  <el-option label="UTC+0 (格林威治时间)" value="UTC"></el-option>
                </el-select>
              </el-form-item>
            </el-form>
            <el-divider></el-divider>
            <el-button type="primary" @click="savePreferenceSettings" :loading="preferenceSaving" style="width: 100%">
              保存设置
            </el-button>
          </div>
        </el-card>
      </div>
    </div>
    <el-dialog
      v-model="emailChangeDialogVisible"
      title="修改邮箱"
      :width="isMobile ? '90%' : '500px'"
      :close-on-click-modal="false"
    >
      <el-form :model="emailChangeForm" :rules="emailChangeRules" ref="emailChangeFormRef" label-width="100px">
        <el-form-item label="新邮箱" prop="newEmail">
          <el-input v-model="emailChangeForm.newEmail" placeholder="请输入新邮箱"></el-input>
        </el-form-item>
        <el-form-item label="验证码" prop="verificationCode">
          <el-input v-model="emailChangeForm.verificationCode" placeholder="请输入验证码">
            <template #append>
              <el-button @click="sendVerificationCode" :disabled="codeSending">
                {{ codeSending ? '发送中...' : '发送验证码' }}
              </el-button>
            </template>
          </el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="emailChangeDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="confirmEmailChange" :loading="emailChanging">
            确认修改
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>
<script>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { Bell, Lock, Plus, Setting, Star, User } from '@element-plus/icons-vue'
import { useAuthStore } from '@/store/auth'
import { useThemeStore } from '@/store/theme'
import { api } from '@/utils/api'
import { useMobile } from '@/composables/useMobile'
const notificationTypeOptions = [
  { value: 'system', label: '系统通知' },
  { value: 'security', label: '安全/密码通知' },
  { value: 'payment', label: '订单/充值通知' },
  { value: 'subscription', label: '订阅通知' },
  { value: 'ticket', label: '工单通知' },
  { value: 'marketing', label: '营销通知' }
]
const defaultNotificationTypes = notificationTypeOptions.map(item => item.value)
export default {
  name: 'UserSettings',
  components: { Bell, Lock, Plus, Setting, Star, User },
  setup() {
    const authStore = useAuthStore()
    const themeStore = useThemeStore()
    const activeSetting = ref('profile')
    const isMobile = useMobile()
    const profileFormRef = ref()
    const securityFormRef = ref()
    const emailChangeFormRef = ref()
    const profileSaving = ref(false)
    const passwordChanging = ref(false)
    const notificationSaving = ref(false)
    const preferenceSaving = ref(false)
    const emailChanging = ref(false)
    const codeSending = ref(false)
    const emailChangeDialogVisible = ref(false)
    const profileForm = reactive({
      username: '',
      email: '',
      nickname: '',
      avatar: ''
    })
    const securityForm = reactive({
      currentPassword: '',
      newPassword: '',
      confirmPassword: ''
    })
    const notificationForm = reactive({
      emailNotifications: true,
      abnormalLoginAlert: true,  // 异常登录/设备告警，默认开启
      notificationTypes: [...defaultNotificationTypes]  // 默认所有类型都开启
    })
    const preferenceForm = reactive({
      theme: themeStore.currentTheme,
      timezone: 'Asia/Shanghai'
    })
    const emailChangeForm = reactive({
      newEmail: '',
      verificationCode: ''
    })
    const profileRules = {
      username: [
        { required: true, message: '请输入用户名', trigger: 'blur' },
        { min: 2, max: 20, message: '用户名长度在 2 到 20 个字符', trigger: 'blur' }
      ],
      nickname: [
        { max: 50, message: '昵称长度不能超过 50 个字符', trigger: 'blur' }
      ]
    }
    const securityRules = {
      currentPassword: [
        { required: true, message: '请输入当前密码', trigger: 'blur' }
      ],
      newPassword: [
        { required: true, message: '请输入新密码', trigger: 'blur' },
        { min: 6, message: '密码长度不能少于 6 个字符', trigger: 'blur' }
      ],
      confirmPassword: [
        { required: true, message: '请再次输入新密码', trigger: 'blur' },
        {
          validator: (rule, value, callback) => {
            if (value !== securityForm.newPassword) {
              callback(new Error('两次输入密码不一致'))
            } else {
              callback()
            }
          },
          trigger: 'blur'
        }
      ]
    }
    const emailChangeRules = {
      newEmail: [
        { required: true, message: '请输入新邮箱', trigger: 'blur' },
        { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
      ],
      verificationCode: [
        { required: true, message: '请输入验证码', trigger: 'blur' },
        { len: 6, message: '验证码长度应为 6 位', trigger: 'blur' }
      ]
    }
    const handleSettingSelect = (key) => {
      activeSetting.value = key
    }
    const loadUserInfo = async () => {
      let loadedUser = null
      try {
        const response = await api.get('/users/me')
        if (response.data && response.data.success && response.data.data) {
          const userData = response.data.data
          loadedUser = userData
          profileForm.username = userData.username || ''
          profileForm.email = userData.email || ''
          profileForm.nickname = userData.nickname || ''
          profileForm.avatar = userData.avatar || userData.avatar_url || ''
        } else {
          const user = authStore.user
          if (user) {
            profileForm.username = user.username || ''
            profileForm.email = user.email || ''
            profileForm.nickname = user.nickname || ''
            profileForm.avatar = user.avatar || ''
          }
        }
      } catch (error) {
        const user = authStore.user
        if (user) {
          profileForm.username = user.username || ''
          profileForm.email = user.email || ''
          profileForm.nickname = user.nickname || ''
          profileForm.avatar = user.avatar || ''
        }
      }
      try {
        const notificationResponse = await api.get('/users/notification-settings')
        const settings = notificationResponse.data?.data || notificationResponse.data || {}
        if (settings.email_notifications !== undefined && settings.email_notifications !== null) {
          notificationForm.emailNotifications = settings.email_notifications === true || settings.email_notifications === 'true'
        } else if (settings.email_enabled !== undefined && settings.email_enabled !== null) {
          notificationForm.emailNotifications = settings.email_enabled === true || settings.email_enabled === 'true'
        } else {
          notificationForm.emailNotifications = true
        }
        if (settings.abnormal_login_alert !== undefined && settings.abnormal_login_alert !== null) {
          notificationForm.abnormalLoginAlert = settings.abnormal_login_alert === true || settings.abnormal_login_alert === 'true'
        } else {
          notificationForm.abnormalLoginAlert = true
        }
        if (settings.notification_types !== undefined && settings.notification_types !== null) {
          if (typeof settings.notification_types === 'string' && settings.notification_types.trim() !== '') {
            try {
              const parsed = JSON.parse(settings.notification_types)
              notificationForm.notificationTypes = Array.isArray(parsed) ? parsed : [...defaultNotificationTypes]
            } catch (e) {
              notificationForm.notificationTypes = [...defaultNotificationTypes]
            }
          } else if (Array.isArray(settings.notification_types) && settings.notification_types.length > 0) {
            notificationForm.notificationTypes = settings.notification_types
          } else {
            notificationForm.notificationTypes = [...defaultNotificationTypes]
          }
        } else {
          notificationForm.notificationTypes = [...defaultNotificationTypes]
        }
      } catch (error) {
        console.error('加载通知设置失败:', {
          error: error.message,
          response: error.response?.data,
          status: error.response?.status,
          url: '/users/notification-settings'
        })
        notificationForm.emailNotifications = true
        notificationForm.abnormalLoginAlert = true
        notificationForm.notificationTypes = [...defaultNotificationTypes]
      }
      try {
        const preferenceSource = loadedUser || authStore.user || {}
        if (preferenceSource && typeof preferenceSource === 'object') {
          if (preferenceSource.theme && typeof preferenceSource.theme === 'string') {
            preferenceForm.theme = preferenceSource.theme
          }
          if (preferenceSource.timezone && typeof preferenceSource.timezone === 'string') {
            preferenceForm.timezone = preferenceSource.timezone
          }
        }
      } catch (error) {
        console.error('加载用户设置失败:', {
          error: error.message,
          response: error.response?.data,
          status: error.response?.status,
          url: 'local-user-preferences'
        })
      }
    }
    const saveProfile = async () => {
      try {
        await profileFormRef.value.validate()
        profileSaving.value = true
        const response = await api.put('/users/me', {
          username: profileForm.username || '',
          nickname: profileForm.nickname || '',
          avatar: profileForm.avatar || ''
        })
        if (response.data && response.data.success !== false) {
          await loadUserInfo()
          if (authStore && authStore.updateUser) {
            authStore.updateUser(profileForm)
          }
          ElMessage.success(response.data.message || '个人资料保存成功')
        } else {
          ElMessage.error(response.data?.message || '保存失败')
        }
      } catch (error) {
        console.error('保存个人资料失败:', {
          error: error.message,
          response: error.response?.data,
          status: error.response?.status,
          requestData: {
            username: profileForm.username,
            nickname: profileForm.nickname,
            avatar: profileForm.avatar
          }
        })
        const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '保存失败'
        ElMessage.error(errorMsg)
      } finally {
        profileSaving.value = false
      }
    }
    const changePassword = async () => {
      try {
        await securityFormRef.value.validate()
        passwordChanging.value = true
        const response = await api.post('/users/change-password', {
          current_password: securityForm.currentPassword || '',
          new_password: securityForm.newPassword || ''
        })
        if (response.data && response.data.success !== false) {
          ElMessage.success(response.data.message || '密码修改成功')
          securityForm.currentPassword = ''
          securityForm.newPassword = ''
          securityForm.confirmPassword = ''
          if (securityFormRef.value) {
            securityFormRef.value.resetFields()
          }
        } else {
          ElMessage.error(response.data?.message || '密码修改失败')
        }
      } catch (error) {
        console.error('修改密码失败:', {
          error: error.message,
          response: error.response?.data,
          status: error.response?.status,
          url: '/users/change-password'
        })
        const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '密码修改失败'
        ElMessage.error(errorMsg)
      } finally {
        passwordChanging.value = false
      }
    }
    const saveNotificationSettings = async () => {
      try {
        notificationSaving.value = true
        const response = await api.put('/users/notification-settings', {
          email_notifications: notificationForm.emailNotifications,
          abnormal_login_alert: notificationForm.abnormalLoginAlert,
          notification_types: notificationForm.notificationTypes
        })
        if (response.data && response.data.success !== false) {
          ElMessage.success(response.data.message || '通知设置保存成功')
        } else {
          ElMessage.error(response.data?.message || '通知设置保存失败')
        }
      } catch (error) {
        console.error('保存通知设置失败:', {
          error: error.message,
          response: error.response?.data,
          status: error.response?.status,
          requestData: {
            email_notifications: notificationForm.emailNotifications,
            notification_types: notificationForm.notificationTypes
          }
        })
        const errorMessage = error.response?.data?.detail || error.response?.data?.message || error.message || '保存失败'
        ElMessage.error('保存失败：' + errorMessage)
      } finally {
        notificationSaving.value = false
      }
    }
    const savePreferenceSettings = async () => {
      try {
        preferenceSaving.value = true
        const themeChanged = themeStore && themeStore.currentTheme !== preferenceForm.theme
        let themeSaved = false
        let themeLocalApplied = false
        if (themeChanged && themeStore && themeStore.setTheme) {
          const themeResult = await themeStore.setTheme(preferenceForm.theme)
          themeSaved = themeResult.success
          themeLocalApplied = themeResult.localApplied || false
          if (!themeResult.success && !themeResult.localApplied) {
            ElMessage.error(themeResult.message || '主题保存失败')
            return
          }
        }
        try {
          const response = await api.put('/users/preferences', {
            timezone: preferenceForm.timezone
          })
          if (response.data && response.data.success !== false) {
            if (themeChanged) {
              if (themeSaved) {
                ElMessage.success('偏好设置保存成功')
              } else if (themeLocalApplied) {
                ElMessage.success('偏好设置保存成功（主题已本地应用）')
              } else {
                ElMessage.success('时区设置保存成功')
              }
            } else {
              ElMessage.success('时区设置保存成功')
            }
          } else {
            if (themeChanged && (themeSaved || themeLocalApplied)) {
              ElMessage.warning(response.data?.message || '时区保存失败，但主题已保存')
            } else {
              ElMessage.error(response.data?.message || '时区保存失败')
            }
          }
        } catch (timezoneError) {
          if (themeChanged && (themeSaved || themeLocalApplied)) {
            ElMessage.warning('时区保存失败，但主题已保存')
          } else {
            throw timezoneError
          }
        }
      } catch (error) {
        console.error('保存偏好设置失败:', {
          error: error.message,
          response: error.response?.data,
          status: error.response?.status,
          requestData: {
            theme: preferenceForm.theme,
            timezone: preferenceForm.timezone
          }
        })
        const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '保存失败'
        ElMessage.error(errorMsg)
      } finally {
        preferenceSaving.value = false
      }
    }
    const showEmailChangeDialog = () => {
      emailChangeForm.newEmail = ''
      emailChangeForm.verificationCode = ''
      emailChangeDialogVisible.value = true
    }
    const sendVerificationCode = async () => {
      try {
        if (!emailChangeForm.newEmail) {
          ElMessage.warning('请先输入新邮箱')
          return
        }
        codeSending.value = true
        await userAPI.sendVerificationCode({ email: emailChangeForm.newEmail, type: 'email_change' })
        ElMessage.success('验证码已发送到您的新邮箱')
      } catch (error) {
        const msg = error?.response?.data?.message || error.message || '发送失败'
        ElMessage.error('发送验证码失败：' + msg)
      } finally {
        codeSending.value = false
      }
    }
    const confirmEmailChange = async () => {
      try {
        await emailChangeFormRef.value.validate()
        emailChanging.value = true
        ElMessage.success('邮箱修改成功')
        emailChangeDialogVisible.value = false
        profileForm.email = emailChangeForm.newEmail
      } catch (error) {
        ElMessage.error('邮箱修改失败：' + error.message)
      } finally {
        emailChanging.value = false
      }
    }
    const beforeAvatarUpload = (file) => {
      const isJPG = file.type === 'image/jpeg' || file.type === 'image/png'
      const isLt2M = file.size / 1024 / 1024 < 2
      if (!isJPG) {
        ElMessage.error('上传头像图片只能是 JPG/PNG 格式!')
      }
      if (!isLt2M) {
        ElMessage.error('上传头像图片大小不能超过 2MB!')
      }
      return isJPG && isLt2M
    }
    onMounted(() => {
      loadUserInfo()
      themeStore.initTheme()
      themeStore.loadUserTheme()
    })
    return {
      isMobile,
      activeSetting,
      profileFormRef,
      securityFormRef,
      emailChangeFormRef,
      profileSaving,
      passwordChanging,
      notificationSaving,
      preferenceSaving,
      emailChanging,
      codeSending,
      emailChangeDialogVisible,
      profileForm,
      securityForm,
      notificationForm,
      notificationTypeOptions,
      preferenceForm,
      emailChangeForm,
      profileRules,
      securityRules,
      emailChangeRules,
      handleSettingSelect,
      saveProfile,
      changePassword,
      saveNotificationSettings,
      savePreferenceSettings,
      showEmailChangeDialog,
      sendVerificationCode,
      confirmEmailChange,
      beforeAvatarUpload
    }
  }
}
</script>
<style scoped>
.page-header {
  margin-bottom: 12px;
}

.settings-desktop {
  align-items: flex-start;
}

.settings-menu {
  position: sticky;
  top: 76px;

  :deep(.el-card__header) {
    padding: 14px 16px;
  }
}

.settings-menu-list {
  border-right: none;
}

.settings-menu-list :deep(.el-menu-item) {
  height: 42px;
  border-radius: 6px;
  color: #4b5563;
  font-weight: 500;
}

.settings-menu-list :deep(.el-menu-item.is-active) {
  background: #ecf5ff;
  color: var(--el-color-primary);
}

.setting-content {
  margin-bottom: 12px;
}

.card-header,
.tab-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  font-weight: 700;
}

.tab-label {
  gap: 6px;
  font-weight: 500;
}

.setting-content :deep(.el-card__header) {
  padding: 14px 16px;
}

.setting-content :deep(.el-card__body) {
  padding: 18px;
}

.setting-content :deep(.el-form) {
  max-width: 720px;
}

.setting-content :deep(.el-form-item__label) {
  color: #4b5563;
  font-weight: 600;
}

:deep(.el-input__wrapper),
:deep(.el-textarea__inner) {
  border-radius: 6px;
  box-shadow: 0 0 0 1px #dcdfe6 inset;
}

:deep(.el-input__wrapper:hover),
:deep(.el-input__wrapper.is-focus),
:deep(.el-textarea__inner:hover),
:deep(.el-textarea__inner:focus) {
  box-shadow: 0 0 0 1px var(--el-color-primary) inset;
}

.avatar-uploader {
  text-align: center;
}

.avatar-uploader .el-upload {
  border: 1px dashed #cfd8e3;
  border-radius: 8px;
  cursor: pointer;
  position: relative;
  overflow: clip;
  transition: border-color 0.16s ease, background-color 0.16s ease;
}

.avatar-uploader .el-upload:hover {
  border-color: var(--el-color-primary);
  background: #f8fbff;
}

.avatar-uploader-icon,
.avatar {
  width: 96px;
  height: 96px;
}

.avatar-uploader-icon {
  color: #8c939d;
  font-size: 28px;
  line-height: 96px;
  text-align: center;
}

.avatar {
  display: block;
  object-fit: cover;
}

.notification-settings,
.preference-settings {
  margin-top: 4px;
}

.notification-settings h4,
.preference-settings h4 {
  margin: 0 0 12px;
  color: #1f2937;
  font-size: 15px;
  font-weight: 700;
}

.setting-hint {
  margin: -4px 0 12px;
  color: #606266;
  font-size: 13px;
  line-height: 1.6;
}

.notification-channel-form {
  max-width: 520px;
}

.notification-settings :deep(.el-checkbox-group) {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(148px, 1fr));
  gap: 10px;
}

.notification-settings :deep(.el-checkbox) {
  margin-right: 0;
}

.el-divider {
  margin: 20px 0;
}

.dialog-footer {
  text-align: right;
}

.settings-mobile {
  display: none;
}

@media (max-width: 768px) {
  .settings-desktop {
    display: none;
  }

  .settings-mobile {
    display: block;
  }

  .page-header {
    margin-bottom: 12px;
  }

  .mobile-tabs-card {
    margin-bottom: 12px;

    :deep(.el-card__body) {
      padding: 0 10px;
    }
  }

  .mobile-settings-tabs {
    :deep(.el-tabs__header) {
      margin: 0;
    }

    :deep(.el-tabs__nav-wrap::after) {
      display: none;
    }

    :deep(.el-tabs__item) {
      height: 42px;
      padding: 0 12px;
      font-size: 13px;
      line-height: 42px;
    }

    :deep(.el-tabs__content) {
      display: none;
    }
  }

  .mobile-setting-card {
    margin-bottom: 12px;

    :deep(.el-card__header) {
      padding: 12px 14px;
    }

    :deep(.el-card__body) {
      padding: 14px;
    }
  }

  .mobile-form {
    :deep(.el-form-item) {
      margin-bottom: 18px;
    }

    :deep(.el-form-item__label) {
      width: 100% !important;
      margin-bottom: 8px;
      padding: 0;
      text-align: left;
      font-size: 14px;
      font-weight: 600;
      line-height: 1.5;
    }

    :deep(.el-form-item__content) {
      width: 100%;
    }

    :deep(.el-input),
    :deep(.el-select),
    :deep(.el-textarea) {
      width: 100%;
    }
  }

  .avatar-uploader {
    width: 100%;
    text-align: center;

    .el-upload {
      width: 96px;
      height: 96px;
      margin: 0 auto;
    }
  }

  .mobile-radio-group,
  .mobile-checkbox-group {
    display: grid;
    grid-template-columns: 1fr;
    gap: 10px;
    width: 100%;
  }

  .mobile-radio-group :deep(.el-radio),
  .mobile-checkbox-group :deep(.el-checkbox) {
    width: 100%;
    min-height: 42px;
    margin: 0;
    padding: 10px 12px;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    transition: border-color 0.16s ease, background-color 0.16s ease;
  }

  .mobile-radio-group :deep(.el-radio:hover),
  .mobile-checkbox-group :deep(.el-checkbox:hover) {
    background-color: #f5f9ff;
    border-color: #c6e2ff;
  }

  .mobile-radio-group :deep(.el-radio.is-checked),
  .mobile-checkbox-group :deep(.el-checkbox.is-checked) {
    background-color: #ecf5ff;
    border-color: var(--el-color-primary);
  }

  :deep(.el-switch) {
    width: 100%;
    min-height: 42px;
    justify-content: space-between;
    padding: 10px 12px;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    margin-bottom: 10px;
  }

  :deep(.el-switch__label) {
    flex: 1;
    font-size: 14px;
  }

  .mobile-setting-card :deep(.el-button) {
    width: 100%;
    min-height: 40px;
    margin: 0;
  }

  :deep(.el-divider) {
    margin: 16px 0;
  }
}
</style>
