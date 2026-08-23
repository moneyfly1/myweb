<template>
  <div class="list-container user-settings">
    <div class="breadcrumb">首页 / 设置</div>
    <div class="page-header">
      <div class="page-title">
        <h1>用户设置</h1>
      </div>
      <div class="actions">
        <el-button type="primary" @click="saveCurrentSetting" :loading="currentSettingSaving">
          保存当前设置
        </el-button>
      </div>
    </div>
    <div class="settings-workspace">
      <aside class="settings-nav-card">
        <button
          v-for="item in settingNavItems"
          :key="item.name"
          type="button"
          class="settings-nav-button"
          :class="{ active: activeSetting === item.name }"
          @click="handleSettingSelect(item.name)"
        >
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </button>
      </aside>

      <div class="card settings-tabs-card settings-panel-card">
        <div class="settings-panel-header">
          <div>
            <h2>
              <el-icon><component :is="activeSettingMeta.icon" /></el-icon>
              {{ activeSettingMeta.label }}
            </h2>
          </div>
        </div>
        <div class="settings-tab-body section-stack">
        <section v-if="activeSetting === 'profile'" class="setting-content setting-section" data-setting-section="profile">
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
                :http-request="handleAvatarUpload"
              >
                <img v-if="profileForm.avatar" :src="profileForm.avatar" class="avatar" />
                <div v-else class="avatar-uploader-icon avatar-fallback-text">
                  {{ (profileForm.nickname || profileForm.username || '用户').slice(0, 2).toUpperCase() }}
                </div>
              </el-upload>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveProfile" :loading="profileSaving">
                保存修改
              </el-button>
            </el-form-item>
          </el-form>
        </section>
        <section v-else-if="activeSetting === 'security'" class="setting-content setting-section" data-setting-section="security">
          <el-form :model="securityForm" :rules="securityRules" ref="securityFormRef" label-width="100px">
            <el-form-item label="当前密码" prop="currentPassword">
              <el-input
                v-model="securityForm.currentPassword"
                type="password"
                placeholder="请输入当前密码"
                show-password
                autocomplete="current-password"
              ></el-input>
            </el-form-item>
            <el-form-item label="新密码" prop="newPassword">
              <el-input
                v-model="securityForm.newPassword"
                type="password"
                :placeholder="`请输入新密码（至少 ${minPasswordLength} 位）`"
                show-password
                autocomplete="new-password"
              ></el-input>
              <div class="password-requirement">{{ passwordRequirementText }}</div>
            </el-form-item>
            <el-form-item label="确认密码" prop="confirmPassword">
              <el-input
                v-model="securityForm.confirmPassword"
                type="password"
                placeholder="请再次输入新密码"
                show-password
                autocomplete="new-password"
              ></el-input>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="changePassword" :loading="passwordChanging">
                修改密码
              </el-button>
            </el-form-item>
          </el-form>
        </section>
        <section v-else-if="activeSetting === 'notifications'" class="setting-content setting-section" data-setting-section="notifications">
          <el-form class="notification-settings settings-control-form" label-width="100px">
            <el-form-item label="邮件通知">
              <div class="setting-control-block">
                <el-switch
                  v-model="notificationForm.emailNotifications"
                  active-text="启用邮件通知"
                  inactive-text="禁用邮件通知"
                ></el-switch>
              </div>
            </el-form-item>
            <el-form-item label="登录告警">
              <div class="setting-control-block">
                <el-switch
                  v-model="notificationForm.abnormalLoginAlert"
                  active-text="接收告警通知"
                  inactive-text="不接收告警通知"
                ></el-switch>
                <div class="setting-hint">当检测到新设备或异地登录时，可通过邮件和站内通知提醒您。关闭后将不再发送此类告警。</div>
              </div>
            </el-form-item>
            <el-form-item label="通知类型">
              <el-checkbox-group v-model="notificationForm.notificationTypes" class="notification-type-grid">
                <el-checkbox v-for="item in notificationTypeOptions" :key="item.value" :label="item.value">{{ item.label }}</el-checkbox>
              </el-checkbox-group>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveNotificationSettings" :loading="notificationSaving">
                保存设置
              </el-button>
            </el-form-item>
          </el-form>
        </section>
        <section v-else class="setting-content setting-section" data-setting-section="preferences">
          <div class="preference-settings">
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
        </section>
        </div>
      </div>
      <div class="section-stack settings-side">
        <div class="settings-summary-card">
          <div class="settings-summary-avatar">
            {{ (profileForm.nickname || profileForm.username || '用户').slice(0, 2).toUpperCase() }}
          </div>
          <div class="settings-summary-title">{{ profileForm.nickname || profileForm.username || '未设置昵称' }}</div>
          <div class="settings-summary-sub">{{ profileForm.email || '未绑定邮箱' }}</div>
        </div>
      </div>
    </div>
    <AppDialog
      v-model="emailChangeDialogVisible"
      title="修改邮箱"
      width="500px"
      mobile-width="92%"
      :loading="emailChanging"
    >
      <el-form
        :model="emailChangeForm"
        :rules="emailChangeRules"
        ref="emailChangeFormRef"
        label-width="100px"
        class="email-change-form"
      >
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
        <FormActionBar
          :loading="emailChanging"
          submit-text="确认修改"
          @cancel="emailChangeDialogVisible = false"
          @submit="confirmEmailChange"
        />
      </template>
    </AppDialog>
  </div>
</template>
<script>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { Bell, Lock, Setting, Star, User } from '@element-plus/icons-vue'
import { useAuthStore } from '@/store/auth'
import { useThemeStore } from '@/store/theme'
import { api, authAPI, userAPI, settingsAPI } from '@/utils/api'
import { useMobile } from '@/composables/useMobile'
import FormActionBar from '@/components/FormActionBar.vue'
import AppDialog from '@/components/AppDialog.vue'
const notificationTypeOptions = [
  { value: 'system', label: '系统通知' },
  { value: 'security', label: '安全/密码通知' },
  { value: 'payment', label: '订单/充值通知' },
  { value: 'subscription', label: '订阅通知' },
  { value: 'ticket', label: '工单通知' },
  { value: 'marketing', label: '营销通知' }
]
const defaultNotificationTypes = notificationTypeOptions.map(item => item.value)
const settingNavItems = [
  { name: 'profile', label: '个人资料', icon: User },
  { name: 'security', label: '安全设置', icon: Lock },
  { name: 'notifications', label: '通知设置', icon: Bell },
  { name: 'preferences', label: '偏好设置', icon: Star }
]
export default {
  name: 'UserSettings',
  components: { Bell, Lock, Setting, Star, User, FormActionBar, AppDialog },
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
    const minPasswordLength = ref(8)
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
    const weakPasswords = [
      'password', '123456', '123456789', 'qwerty', 'abc123',
      'password123', 'admin', 'root', 'user', 'test',
      '12345678', 'password1', 'qwerty123', 'admin123'
    ]
    const passwordRequirementText = computed(() => (
      `密码长度至少 ${minPasswordLength.value} 位，需包含大写字母、小写字母、数字、特殊字符中的至少三种`
    ))
    const validatePasswordStrength = (value) => {
      if (!value) return '请输入新密码'
      if (value.length < minPasswordLength.value) {
        return `密码长度至少 ${minPasswordLength.value} 位`
      }

      const complexityCount = [
        /[A-Z]/.test(value),
        /[a-z]/.test(value),
        /\d/.test(value),
        /[!@#$%^&*()_+\-=[\]{}|;:,.<>?]/.test(value)
      ].filter(Boolean).length

      if (complexityCount < 3) {
        return '密码必须包含大小写字母、数字和特殊字符中的至少三种'
      }
      if (weakPasswords.includes(value.toLowerCase())) {
        return '密码过于简单，请使用更复杂的密码'
      }
      return null
    }
    const securityRules = computed(() => ({
      currentPassword: [
        { required: true, message: '请输入当前密码', trigger: 'blur' }
      ],
      newPassword: [
        { required: true, message: '请输入新密码', trigger: 'blur' },
        {
          validator: (rule, value, callback) => {
            const error = validatePasswordStrength(value)
            if (error) {
              callback(new Error(error))
              return
            }
            if (securityForm.currentPassword && value === securityForm.currentPassword) {
              callback(new Error('新密码不能与当前密码相同'))
              return
            }
            callback()
          },
          trigger: 'blur'
        }
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
    }))
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
    const currentSettingSaving = computed(() => {
      if (activeSetting.value === 'profile') return profileSaving.value
      if (activeSetting.value === 'security') return passwordChanging.value
      if (activeSetting.value === 'notifications') return notificationSaving.value
      if (activeSetting.value === 'preferences') return preferenceSaving.value
      return false
    })
    const activeSettingMeta = computed(() => (
      settingNavItems.find(item => item.name === activeSetting.value) || settingNavItems[0]
    ))
    const loadPasswordSettings = async () => {
      try {
        const response = await settingsAPI.getPublicSettings()
        const settings = response.data?.data || response.data || {}
        const value = settings.min_password_length !== undefined
          ? settings.min_password_length
          : settings.minPasswordLength
        const parsed = typeof value === 'number' ? value : parseInt(value)
        minPasswordLength.value = Number.isFinite(parsed) && parsed > 0 ? parsed : 8
      } catch (error) {
        minPasswordLength.value = 8
      }
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
        await authAPI.sendVerificationCode({ email: emailChangeForm.newEmail, type: 'email_change' })
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
        const response = await userAPI.updateProfile({
          username: profileForm.username || '',
          nickname: profileForm.nickname || '',
          avatar: profileForm.avatar || '',
          email: emailChangeForm.newEmail,
          verification_code: emailChangeForm.verificationCode
        })
        if (response.data && response.data.success !== false) {
          profileForm.email = emailChangeForm.newEmail
          emailChangeDialogVisible.value = false
          await loadUserInfo()
          ElMessage.success(response.data.message || '邮箱修改成功')
        } else {
          ElMessage.error(response.data?.message || '邮箱修改失败')
        }
      } catch (error) {
        const msg = error.response?.data?.message || error.response?.data?.detail || error.message || '邮箱修改失败'
        ElMessage.error('邮箱修改失败：' + msg)
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
    // 头像本地转 base64 预览（保存时随 profile 一并提交）
    const handleAvatarUpload = ({ file, onSuccess, onError }) => {
      const reader = new FileReader()
      reader.onload = (e) => {
        profileForm.avatar = e.target.result
        ElMessage.success('头像已选择，点击"保存修改"生效')
        onSuccess && onSuccess(file)
      }
      reader.onerror = () => {
        ElMessage.error('头像读取失败')
        onError && onError(new Error('头像读取失败'))
      }
      reader.readAsDataURL(file)
    }
    const saveCurrentSetting = () => {
      const actions = {
        profile: saveProfile,
        security: changePassword,
        notifications: saveNotificationSettings,
        preferences: savePreferenceSettings
      }
      actions[activeSetting.value]?.()
    }
    onMounted(() => {
      loadPasswordSettings()
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
      minPasswordLength,
      profileForm,
      securityForm,
      notificationForm,
      notificationTypeOptions,
      settingNavItems,
      activeSettingMeta,
      preferenceForm,
      emailChangeForm,
      profileRules,
      securityRules,
      emailChangeRules,
      passwordRequirementText,
      handleSettingSelect,
      currentSettingSaving,
      saveProfile,
      changePassword,
      saveNotificationSettings,
      savePreferenceSettings,
      saveCurrentSetting,
      showEmailChangeDialog,
      sendVerificationCode,
      confirmEmailChange,
      beforeAvatarUpload,
      handleAvatarUpload
    }
  }
}
</script>
<style scoped>
.user-settings {
  padding: 0;
  max-width: none;
  margin: 0;
  width: 100%;
}
.breadcrumb {
  margin-bottom: 12px;
  color: #606266;
  font-size: 13px;
  line-height: 1.4;
}
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
  padding: 16px;
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
}
.page-header h1 {
  margin: 0;
  color: #303133;
  font-size: 22px;
  line-height: 1.25;
  font-weight: 700;
}
.page-header p {
  margin: 6px 0 0;
  color: #606266;
  line-height: 1.5;
}
.actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.settings-main-aside {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(320px, 0.85fr);
  align-items: start;
  gap: 14px;
}

.settings-tabs-card {
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  overflow: hidden;
}

.settings-tabs :deep(.el-tabs__header) {
  margin: 0;
  padding: 0 16px;
  background: #fff;
  border-bottom: 1px solid #ebeef5;
}

.settings-tabs :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.settings-tabs :deep(.el-tabs__item) {
  height: 46px;
  color: #606266;
  font-weight: 600;
}

.settings-tabs :deep(.el-tabs__item.is-active) {
  color: #409eff;
}

.settings-tab-body {
  padding: 16px;
}

.setting-content {
  margin-bottom: 0;
  border: 0 !important;
  border-radius: 0;
  box-shadow: none !important;
  background: transparent !important;
  overflow: visible;
}

.setting-content :deep(.el-card) {
  border: 0 !important;
  box-shadow: none !important;
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
  padding: 0 0 14px;
  border-bottom: 0;
}

.setting-content :deep(.el-card__body) {
  padding: 0;
}

.setting-content :deep(.el-form) {
  max-width: none;
}

.setting-content :deep(.el-form-item__label) {
  color: #4b5563;
  font-weight: 600;
}

.user-settings :deep(.el-input__wrapper),
.user-settings :deep(.el-textarea__inner) {
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  box-shadow: none;
}

.password-requirement {
  margin-top: 6px;
  color: #606266;
  font-size: 12px;
  line-height: 1.5;
}

.user-settings :deep(.el-input__wrapper:hover),
.user-settings :deep(.el-textarea__inner:hover) {
  border-color: var(--el-color-primary);
  box-shadow: none;
}

.user-settings :deep(.el-input__wrapper.is-focus),
.user-settings :deep(.el-textarea__inner:focus) {
  border-color: var(--el-color-primary);
  box-shadow: none;
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

.notification-settings h4,
.preference-settings h4 {
  margin: 0 0 12px;
  color: #1f2937;
  font-size: 15px;
  font-weight: 700;
}

.setting-hint {
  margin: 8px 0 0;
  color: #606266;
  font-size: 13px;
  line-height: 1.6;
}

.settings-control-form {
  max-width: 720px;
}

.setting-control-block {
  width: 100%;
  min-width: 0;
  padding-top: 1px;
}

.notification-channel-form {
  max-width: 520px;
}

.notification-settings :deep(.el-checkbox-group),
.notification-type-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(148px, 1fr));
  gap: 10px;
  width: 100%;
}

.notification-settings :deep(.el-checkbox) {
  margin-right: 0;
  min-width: 0;
}

.el-divider {
  margin: 20px 0;
}

.full-width-control,
.mobile-full-width {
  width: 100%;
}

.email-change-form {
  max-width: 100%;
}

.section-stack {
  display: grid;
  gap: 14px;
}

.dialog-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid #ebeef5;
  color: #303133;
  font-size: 16px;
  font-weight: 700;
}

.dialog-body {
  padding: 16px;
}


.notice {
  padding: 12px 14px;
  border: 1px solid #faecd8;
  border-radius: 6px;
  background: #fdf6ec;
  color: #b88230;
  line-height: 1.55;
}

.settings-mobile {
  display: none;
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    padding: 14px;
  }
  .actions,
  .actions .el-button {
    width: 100%;
  }
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

  .user-settings :deep(.el-switch) {
    width: 100%;
    min-height: 42px;
    justify-content: space-between;
    padding: 10px 12px;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    margin-bottom: 10px;
  }

  .user-settings :deep(.el-switch__label) {
    flex: 1;
    font-size: 14px;
  }

  .mobile-setting-card :deep(.el-button) {
    width: 100%;
    min-height: 44px;
    margin: 0;
    touch-action: manipulation;
  }

  .email-change-form {
    :deep(.el-form-item) {
      display: block;
    }

    :deep(.el-form-item__label) {
      width: auto !important;
      justify-content: flex-start;
      margin-bottom: 6px;
      line-height: 1.4;
    }

    :deep(.el-form-item__content) {
      margin-left: 0 !important;
    }

    :deep(.el-input-group__append) {
      padding: 0;
    }

    :deep(.el-input-group__append .el-button) {
      min-width: 96px;
      min-height: 44px;
      margin: 0;
      touch-action: manipulation;
    }
  }

  .user-settings :deep(.el-divider) {
    margin: 16px 0;
  }
}

/* Refined user settings layout */
.settings-main-aside {
  grid-template-columns: minmax(0, 1fr) 260px;
  gap: 14px;
  align-items: start;
}
.settings-tabs-card {
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  background: #fff;
  overflow: hidden;
}
.settings-tabs-card :deep(.el-tabs__header) {
  margin: 0;
  padding: 0 16px;
  background: #f5f7fa;
  border-bottom: 1px solid #ebeef5;
}
.settings-tabs-card :deep(.el-tabs__item) {
  height: 48px;
  color: #606266;
  font-weight: 600;
}
.settings-tabs-card :deep(.el-tabs__item.is-active) {
  color: #409eff;
}
.settings-tab-body {
  padding: 0;
}
.settings-tab-body .setting-section {
  padding: 20px;
  border-bottom: 0;
}
.settings-tab-body .subsection-title {
  margin: 0 0 18px;
  padding-bottom: 12px;
  border-bottom: 1px solid #ebeef5;
  color: #303133;
  font-size: 16px;
}
.settings-tab-body .subsection-title .el-icon {
  color: #409eff;
}
.settings-tab-body :deep(.el-form) {
  max-width: 720px;
}
.settings-tab-body :deep(.el-form-item:last-child) {
  margin-bottom: 0;
}
.avatar-uploader {
  display: inline-block;
}
.avatar-uploader .avatar,
.avatar-fallback-text {
  width: 72px;
  height: 72px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  background: #ecf5ff;
  color: #409eff;
  font-size: 20px;
  font-weight: 800;
  object-fit: cover;
}
.settings-summary-card {
  padding: 18px 16px;
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  text-align: center;
}
.settings-summary-avatar {
  width: 64px;
  height: 64px;
  margin: 0 auto 10px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  background: #ecf5ff;
  color: #409eff;
  font-size: 20px;
  font-weight: 800;
}
.settings-summary-title {
  color: #303133;
  font-size: 16px;
  font-weight: 700;
  line-height: 1.4;
  word-break: break-word;
}
.settings-summary-sub {
  margin-top: 4px;
  color: #909399;
  font-size: 13px;
  line-height: 1.45;
  word-break: break-word;
}
.notification-settings h4,
.preference-settings h4 {
  margin-top: 0;
  color: #303133;
}
.notification-settings :deep(.el-checkbox-group),
.preference-settings :deep(.el-radio-group) {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px 12px;
}
.notification-settings :deep(.el-checkbox),
.preference-settings :deep(.el-radio) {
  margin-right: 0;
  min-width: 0;
}
@media (max-width: 900px) {
  .settings-main-aside {
    grid-template-columns: 1fr;
  }
  .settings-side {
    order: -1;
  }
  .settings-summary-card {
    display: none;
  }
}
@media (max-width: 768px) {
  .settings-desktop {
    display: block;
  }
  .settings-mobile {
    display: none;
  }
  .settings-tabs-card :deep(.el-tabs__header) {
    overflow-x: auto;
    padding: 0 10px;
  }
  .settings-tab-body .setting-section {
    padding: 16px 12px;
  }
  .settings-tab-body :deep(.el-form) {
    max-width: none;
  }
  .settings-tab-body :deep(.el-form-item) {
    display: block;
  }
  .settings-tab-body :deep(.el-form-item__label) {
    width: 100% !important;
    justify-content: flex-start;
    margin-bottom: 8px;
  }
  .settings-tab-body :deep(.el-form-item__content) {
    margin-left: 0 !important;
  }
  .notification-settings :deep(.el-checkbox-group),
  .preference-settings :deep(.el-radio-group) {
    grid-template-columns: 1fr;
  }
}

/* Final single-panel settings navigation */
.settings-workspace {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr) 260px;
  gap: 14px;
  align-items: start;
}
.settings-nav-card,
.settings-panel-card,
.settings-side {
  min-width: 0;
}
.settings-nav-card {
  display: grid;
  gap: 8px;
  padding: 10px;
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
}
.settings-nav-button {
  width: 100%;
  min-height: 42px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  color: #606266;
  font: inherit;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.settings-nav-button:hover {
  background: #f5f7fa;
  border-color: #ebeef5;
}
.settings-nav-button.active {
  background: #ecf5ff;
  border-color: #c6e2ff;
  color: #409eff;
}
.settings-panel-header {
  padding: 16px 20px;
  border-bottom: 1px solid #ebeef5;
  background: #f5f7fa;
}
.settings-panel-header h2 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  color: #303133;
  font-size: 18px;
  line-height: 1.35;
}
.settings-panel-header h2 .el-icon {
  color: #409eff;
}
.settings-panel-header p {
  margin: 6px 0 0;
  color: #606266;
  font-size: 13px;
  line-height: 1.5;
}
.settings-panel-card .settings-tab-body {
  padding: 0;
}
.settings-panel-card .setting-section {
  padding: 20px;
}
.settings-panel-card .subsection-title {
  display: none;
}
@media (max-width: 1100px) {
  .settings-workspace {
    grid-template-columns: 190px minmax(0, 1fr);
  }
  .settings-side {
    grid-column: 1 / -1;
    order: 3;
  }
}
@media (max-width: 768px) {
  .settings-workspace {
    grid-template-columns: 1fr;
  }
  .settings-nav-card {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .settings-nav-button {
    justify-content: center;
    text-align: center;
  }
  .settings-panel-header {
    padding: 14px 12px;
  }
  .settings-panel-card .setting-section {
    padding: 16px 12px;
  }
}
</style>
