<template>
  <div class="user-settings-container">
    <div class="page-header">
      <h1>用户设置</h1>
      <p>管理您的账户设置和偏好</p>
    </div>

    <!-- 桌面端布局 -->
    <el-row :gutter="20" class="settings-desktop">
      <!-- 左侧设置菜单 -->
      <el-col :span="6">
        <el-card class="settings-menu">
          <template #header>
            <div class="card-header">
              <i class="el-icon-setting"></i>
              设置分类
            </div>
          </template>
          
          <el-menu
            :default-active="activeSetting"
            @select="handleSettingSelect"
            class="settings-menu-list"
          >
            <el-menu-item index="profile">
              <i class="el-icon-user"></i>
              <span>个人资料</span>
            </el-menu-item>
            <el-menu-item index="security">
              <i class="el-icon-lock"></i>
              <span>安全设置</span>
            </el-menu-item>
            <el-menu-item index="notifications">
              <i class="el-icon-bell"></i>
              <span>通知设置</span>
            </el-menu-item>
            <el-menu-item index="privacy">
              <i class="el-icon-view"></i>
              <span>隐私设置</span>
            </el-menu-item>
            <el-menu-item index="preferences">
              <i class="el-icon-star-on"></i>
              <span>偏好设置</span>
            </el-menu-item>
          </el-menu>
        </el-card>
      </el-col>

      <!-- 右侧设置内容 -->
      <el-col :span="18">
        <!-- 个人资料设置 -->
        <el-card v-if="activeSetting === 'profile'" class="setting-content">
          <template #header>
            <div class="card-header">
              <i class="el-icon-user"></i>
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
                <i v-else class="el-icon-plus avatar-uploader-icon"></i>
              </el-upload>
            </el-form-item>
            
            <el-form-item>
              <el-button type="primary" @click="saveProfile" :loading="profileSaving">
                保存修改
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <!-- 安全设置 -->
        <el-card v-if="activeSetting === 'security'" class="setting-content">
          <template #header>
            <div class="card-header">
              <i class="el-icon-lock"></i>
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

        <!-- 通知设置 -->
        <el-card v-if="activeSetting === 'notifications'" class="setting-content">
          <template #header>
            <div class="card-header">
              <i class="el-icon-bell"></i>
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
            
            <h4>通知类型</h4>
            <el-checkbox-group v-model="notificationForm.notificationTypes">
              <el-checkbox label="subscription">订阅相关通知</el-checkbox>
              <el-checkbox label="payment">支付相关通知</el-checkbox>
              <el-checkbox label="system">系统通知</el-checkbox>
              <el-checkbox label="marketing">营销通知</el-checkbox>
            </el-checkbox-group>
            
            <el-divider></el-divider>
            
            <el-button type="primary" @click="saveNotificationSettings" :loading="notificationSaving">
              保存设置
            </el-button>
          </div>
        </el-card>

        <!-- 隐私设置 -->
        <el-card v-if="activeSetting === 'privacy'" class="setting-content">
          <template #header>
            <div class="card-header">
              <i class="el-icon-view"></i>
              隐私设置
            </div>
          </template>
          
          <div class="privacy-settings">
            <h4>数据共享</h4>
            <el-switch
              v-model="privacyForm.dataSharing"
              active-text="允许数据共享"
              inactive-text="禁止数据共享"
            ></el-switch>
            <p class="privacy-tip">数据共享用于改进服务质量，不会泄露个人信息</p>
            
            <el-divider></el-divider>
            
            <h4>分析统计</h4>
            <el-switch
              v-model="privacyForm.analytics"
              active-text="启用使用统计"
              inactive-text="禁用使用统计"
            ></el-switch>
            
            <el-divider></el-divider>
            
            <el-button type="primary" @click="savePrivacySettings" :loading="privacySaving">
              保存设置
            </el-button>
          </div>
        </el-card>

        <!-- 偏好设置 -->
        <el-card v-if="activeSetting === 'preferences'" class="setting-content">
          <template #header>
            <div class="card-header">
              <i class="el-icon-star-on"></i>
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
              
              <el-form-item label="语言">
                <el-select v-model="preferenceForm.language" placeholder="选择语言">
                  <el-option label="简体中文" value="zh-CN"></el-option>
                  <el-option label="English" value="en-US"></el-option>
                </el-select>
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

    <!-- 手机端布局 -->
    <div class="settings-mobile">
      <!-- 手机端标签页选择 -->
      <el-card class="mobile-tabs-card">
        <el-tabs v-model="activeSetting" class="mobile-settings-tabs">
          <el-tab-pane label="个人资料" name="profile">
            <template #label>
              <span><i class="el-icon-user"></i> 个人资料</span>
            </template>
          </el-tab-pane>
          <el-tab-pane label="安全设置" name="security">
            <template #label>
              <span><i class="el-icon-lock"></i> 安全设置</span>
            </template>
          </el-tab-pane>
          <el-tab-pane label="通知设置" name="notifications">
            <template #label>
              <span><i class="el-icon-bell"></i> 通知设置</span>
            </template>
          </el-tab-pane>
          <el-tab-pane label="隐私设置" name="privacy">
            <template #label>
              <span><i class="el-icon-view"></i> 隐私设置</span>
            </template>
          </el-tab-pane>
          <el-tab-pane label="偏好设置" name="preferences">
            <template #label>
              <span><i class="el-icon-star-on"></i> 偏好设置</span>
            </template>
          </el-tab-pane>
        </el-tabs>
      </el-card>

      <!-- 手机端设置内容 -->
      <div class="mobile-settings-content">
        <!-- 个人资料设置 -->
        <el-card v-if="activeSetting === 'profile'" class="setting-content mobile-setting-card">
          <template #header>
            <div class="card-header">
              <i class="el-icon-user"></i>
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
                <i v-else class="el-icon-plus avatar-uploader-icon"></i>
              </el-upload>
            </el-form-item>
            
            <el-form-item>
              <el-button type="primary" @click="saveProfile" :loading="profileSaving" style="width: 100%">
                保存修改
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <!-- 安全设置 -->
        <el-card v-if="activeSetting === 'security'" class="setting-content mobile-setting-card">
          <template #header>
            <div class="card-header">
              <i class="el-icon-lock"></i>
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

        <!-- 通知设置 -->
        <el-card v-if="activeSetting === 'notifications'" class="setting-content mobile-setting-card">
          <template #header>
            <div class="card-header">
              <i class="el-icon-bell"></i>
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
            
            <h4>通知类型</h4>
            <el-checkbox-group v-model="notificationForm.notificationTypes" class="mobile-checkbox-group">
              <el-checkbox label="subscription">订阅相关通知</el-checkbox>
              <el-checkbox label="payment">支付相关通知</el-checkbox>
              <el-checkbox label="system">系统通知</el-checkbox>
              <el-checkbox label="marketing">营销通知</el-checkbox>
            </el-checkbox-group>
            
            <el-divider></el-divider>
            
            <el-button type="primary" @click="saveNotificationSettings" :loading="notificationSaving" style="width: 100%">
              保存设置
            </el-button>
          </div>
        </el-card>

        <!-- 隐私设置 -->
        <el-card v-if="activeSetting === 'privacy'" class="setting-content mobile-setting-card">
          <template #header>
            <div class="card-header">
              <i class="el-icon-view"></i>
              隐私设置
            </div>
          </template>
          
          <div class="privacy-settings">
            <h4>数据共享</h4>
            <el-switch
              v-model="privacyForm.dataSharing"
              active-text="允许数据共享"
              inactive-text="禁止数据共享"
            ></el-switch>
            <p class="privacy-tip">数据共享用于改进服务质量，不会泄露个人信息</p>
            
            <el-divider></el-divider>
            
            <h4>分析统计</h4>
            <el-switch
              v-model="privacyForm.analytics"
              active-text="启用使用统计"
              inactive-text="禁用使用统计"
            ></el-switch>
            
            <el-divider></el-divider>
            
            <el-button type="primary" @click="savePrivacySettings" :loading="privacySaving" style="width: 100%">
              保存设置
            </el-button>
          </div>
        </el-card>

        <!-- 偏好设置 -->
        <el-card v-if="activeSetting === 'preferences'" class="setting-content mobile-setting-card">
          <template #header>
            <div class="card-header">
              <i class="el-icon-star-on"></i>
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
              
              <el-form-item label="语言">
                <el-select v-model="preferenceForm.language" placeholder="选择语言" style="width: 100%">
                  <el-option label="简体中文" value="zh-CN"></el-option>
                  <el-option label="English" value="en-US"></el-option>
                </el-select>
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

    <!-- 修改邮箱对话框 -->
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
import { ref, reactive, onMounted, computed, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/store/auth'
import { useThemeStore } from '@/store/theme'
import { api } from '@/utils/api'

export default {
  name: 'UserSettings',
  setup() {
    const authStore = useAuthStore()
    const themeStore = useThemeStore()
    const activeSetting = ref('profile')
    
    // 移动端检测
    const windowWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1920)
    const isMobile = computed(() => {
      return windowWidth.value <= 768
    })
    
    // 监听窗口大小变化
    const handleResize = () => {
      if (typeof window !== 'undefined') {
        windowWidth.value = window.innerWidth
      }
    }
    
    // 表单引用
    const profileFormRef = ref()
    const securityFormRef = ref()
    const emailChangeFormRef = ref()
    
    // 加载状态
    const profileSaving = ref(false)
    const passwordChanging = ref(false)
    const notificationSaving = ref(false)
    const privacySaving = ref(false)
    const preferenceSaving = ref(false)
    const emailChanging = ref(false)
    const codeSending = ref(false)
    
    // 对话框状态
    const emailChangeDialogVisible = ref(false)
    
    // 表单数据
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
      notificationTypes: ['subscription', 'payment', 'system', 'marketing']  // 默认所有类型都开启
    })
    
    const privacyForm = reactive({
      dataSharing: true,
      analytics: true
    })
    
    const preferenceForm = reactive({
      theme: themeStore.currentTheme,
      language: 'zh-CN',
      timezone: 'Asia/Shanghai'
    })
    
    const emailChangeForm = reactive({
      newEmail: '',
      verificationCode: ''
    })
    
    // 表单验证规则
    const profileRules = {
      username: [
        { required: true, message: '请输入用户名', trigger: 'blur' },
        { min: 3, max: 20, message: '用户名长度在 3 到 20 个字符', trigger: 'blur' }
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
    
    // 处理设置选择
    const handleSettingSelect = (key) => {
      activeSetting.value = key
    }
    
    // 加载用户信息
    const loadUserInfo = async () => {
      try {
        // 从API获取最新用户信息
        const response = await api.get('/users/me')
        if (response.data && response.data.success && response.data.data) {
          const userData = response.data.data
          profileForm.username = userData.username || ''
          profileForm.email = userData.email || ''
          profileForm.nickname = userData.nickname || ''
          profileForm.avatar = userData.avatar || userData.avatar_url || ''
        } else {
          // 如果API失败，从authStore获取
          const user = authStore.user
          if (user) {
            profileForm.username = user.username || ''
            profileForm.email = user.email || ''
            profileForm.nickname = user.nickname || ''
            profileForm.avatar = user.avatar || ''
          }
        }
      } catch (error) {
        // 如果API失败，从authStore获取
        const user = authStore.user
        if (user) {
          profileForm.username = user.username || ''
          profileForm.email = user.email || ''
          profileForm.nickname = user.nickname || ''
          profileForm.avatar = user.avatar || ''
        }
      }
      
      // 加载通知设置
      try {
        const notificationResponse = await api.get('/users/notification-settings')
        if (notificationResponse.data && notificationResponse.data.success && notificationResponse.data.data) {
          const settings = notificationResponse.data.data
          notificationForm.emailNotifications = settings.email_notifications !== false  // 默认true
          // 解析 notification_types（可能是JSON字符串或数组）
          if (settings.notification_types) {
            if (typeof settings.notification_types === 'string') {
              try {
                notificationForm.notificationTypes = JSON.parse(settings.notification_types)
              } catch (e) {
                notificationForm.notificationTypes = ['subscription', 'payment', 'system', 'marketing']
              }
            } else if (Array.isArray(settings.notification_types)) {
              notificationForm.notificationTypes = settings.notification_types
            }
          }
        }
      } catch (error) {
        // 使用默认值
        notificationForm.emailNotifications = true
        notificationForm.notificationTypes = ['subscription', 'payment', 'system', 'marketing']
      }
      
      // 加载隐私设置
      try {
        const privacyResponse = await api.get('/users/privacy-settings')
        if (privacyResponse.data && privacyResponse.data.success && privacyResponse.data.data) {
          const settings = privacyResponse.data.data
          privacyForm.dataSharing = settings.data_sharing !== false  // 默认true
          privacyForm.analytics = settings.analytics !== false  // 默认true
        }
      } catch (error) {
        // 使用默认值
        privacyForm.dataSharing = true
        privacyForm.analytics = true
      }
      
      // 加载偏好设置（从用户信息中获取）
      try {
        const userResponse = await api.get('/users/me')
        if (userResponse.data && userResponse.data.success && userResponse.data.data) {
          const userData = userResponse.data.data
          if (userData.theme) preferenceForm.theme = userData.theme
          if (userData.language) preferenceForm.language = userData.language
          if (userData.timezone) preferenceForm.timezone = userData.timezone
        }
      } catch (error) {
        // 使用默认值
        console.warn('加载偏好设置失败:', error)
      }
    }
    
    // 保存个人资料
    const saveProfile = async () => {
      try {
        await profileFormRef.value.validate()
        profileSaving.value = true
        
        // 调用API保存个人资料
        const response = await api.put('/users/me', {
          username: profileForm.username,
          nickname: profileForm.nickname || '',
          avatar: profileForm.avatar
        })
        
        if (response.data && response.data.success) {
          // 重新加载用户信息以确保数据同步
          await loadUserInfo()
          // 更新本地用户信息
          if (authStore && authStore.updateUser) {
            authStore.updateUser(profileForm)
          }
          ElMessage.success(response.data.message || '个人资料保存成功')
        } else {
          ElMessage.error(response.data?.message || '保存失败')
        }
      } catch (error) {
        const errorMsg = error.response?.data?.message || error.message || '保存失败'
        ElMessage.error(errorMsg)
      } finally {
        profileSaving.value = false
      }
    }
    
    // 修改密码
    const changePassword = async () => {
      try {
        await securityFormRef.value.validate()
        passwordChanging.value = true
        
        // 调用API修改密码
        const response = await api.post('/users/change-password', {
          current_password: securityForm.currentPassword,
          new_password: securityForm.newPassword
        })
        
        if (response.data && response.data.success) {
          ElMessage.success(response.data.message || '密码修改成功')
          securityForm.currentPassword = ''
          securityForm.newPassword = ''
          securityForm.confirmPassword = ''
          // 重置表单验证状态
          if (securityFormRef.value) {
            securityFormRef.value.resetFields()
          }
        } else {
          ElMessage.error(response.data?.message || '密码修改失败')
        }
      } catch (error) {
        const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '密码修改失败'
        ElMessage.error(errorMsg)
      } finally {
        passwordChanging.value = false
      }
    }
    
    // 保存通知设置
    const saveNotificationSettings = async () => {
      try {
        notificationSaving.value = true
        
        // 调用API保存通知设置
        await api.put('/users/notification-settings', {
          email_notifications: notificationForm.emailNotifications,
          notification_types: notificationForm.notificationTypes
        })
        
        ElMessage.success('通知设置保存成功')
      } catch (error) {
        const errorMessage = error.response?.data?.detail || error.message || '保存失败'
        ElMessage.error('保存失败：' + errorMessage)
      } finally {
        notificationSaving.value = false
      }
    }
    
    // 保存隐私设置
    const savePrivacySettings = async () => {
      try {
        privacySaving.value = true
        
        // 调用API保存隐私设置
        await api.put('/users/privacy-settings', {
          data_sharing: privacyForm.dataSharing,
          analytics: privacyForm.analytics
        })
        
        ElMessage.success('隐私设置保存成功')
      } catch (error) {
        const errorMessage = error.response?.data?.detail || error.message || '保存失败'
        ElMessage.error('保存失败：' + errorMessage)
      } finally {
        privacySaving.value = false
      }
    }
    
    // 保存偏好设置
    const savePreferenceSettings = async () => {
      try {
        preferenceSaving.value = true
        
        // 保存主题设置
        if (themeStore && themeStore.setTheme) {
          const themeResult = await themeStore.setTheme(preferenceForm.theme)
          if (!themeResult.success) {
            ElMessage.warning(themeResult.message || '主题保存失败，仅本地生效')
            // 如果主题已在本地应用，继续保存其他设置
            if (!themeResult.localApplied) {
              return
            }
          } else {
            ElMessage.success('主题已保存到云端')
          }
        }
        
        // 保存其他偏好设置（使用 preferences 端点）
        const response = await api.put('/users/preferences', {
          theme: preferenceForm.theme,
          language: preferenceForm.language,
          timezone: preferenceForm.timezone
        })
        
        if (response.data && response.data.success) {
          ElMessage.success(response.data.message || '偏好设置保存成功')
        } else {
          ElMessage.error(response.data?.message || '偏好设置保存失败')
        }
      } catch (error) {
        const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '保存失败'
        ElMessage.error(errorMsg)
      } finally {
        preferenceSaving.value = false
      }
    }
    
    // 显示修改邮箱对话框
    const showEmailChangeDialog = () => {
      emailChangeForm.newEmail = ''
      emailChangeForm.verificationCode = ''
      emailChangeDialogVisible.value = true
    }
    
    // 发送验证码
    const sendVerificationCode = async () => {
      try {
        if (!emailChangeForm.newEmail) {
          ElMessage.warning('请先输入新邮箱')
          return
        }
        
        codeSending.value = true
        
        // 这里调用API发送验证码
        // await api.sendEmailVerificationCode(emailChangeForm.newEmail)
        
        ElMessage.success('验证码已发送到您的邮箱')
      } catch (error) {
        ElMessage.error('发送验证码失败：' + error.message)
      } finally {
        codeSending.value = false
      }
    }
    
    // 确认修改邮箱
    const confirmEmailChange = async () => {
      try {
        await emailChangeFormRef.value.validate()
        emailChanging.value = true
        
        // 这里调用API修改邮箱
        // await api.changeEmail(emailChangeForm.newEmail, emailChangeForm.verificationCode)
        
        ElMessage.success('邮箱修改成功')
        emailChangeDialogVisible.value = false
        profileForm.email = emailChangeForm.newEmail
      } catch (error) {
        ElMessage.error('邮箱修改失败：' + error.message)
      } finally {
        emailChanging.value = false
      }
    }
    
    // 头像上传前的处理
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
      // 初始化主题
      themeStore.initTheme()
      // 加载用户主题设置
      themeStore.loadUserTheme()
      // 初始化窗口大小
      if (typeof window !== 'undefined') {
        windowWidth.value = window.innerWidth
        window.addEventListener('resize', handleResize)
      }
    })
    
    onUnmounted(() => {
      // 清理窗口大小监听
      if (typeof window !== 'undefined') {
        window.removeEventListener('resize', handleResize)
      }
    })
    
    return {
      isMobile,
      windowWidth,
      activeSetting,
      profileFormRef,
      securityFormRef,
      emailChangeFormRef,
      profileSaving,
      passwordChanging,
      notificationSaving,
      privacySaving,
      preferenceSaving,
      emailChanging,
      codeSending,
      emailChangeDialogVisible,
      profileForm,
      securityForm,
      notificationForm,
      privacyForm,
      preferenceForm,
      emailChangeForm,
      profileRules,
      securityRules,
      emailChangeRules,
      handleSettingSelect,
      saveProfile,
      changePassword,
      saveNotificationSettings,
      savePrivacySettings,
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
.user-settings-container {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h1 {
  margin: 0 0 10px 0;
  color: #333;
}

.page-header :is(p) {
  margin: 0;
  color: #666;
}

.settings-menu {
  position: sticky;
  top: 20px;
}

.settings-menu-list {
  border-right: none;
}

.setting-content {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  font-weight: bold;
}

.card-header :is(i) {
  margin-right: 8px;
  color: #409eff;
}

:deep(.el-input__wrapper) {
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
  border-color: #409eff !important;
  box-shadow: none !important;
}

:deep(.el-select .el-input__wrapper) {
  border-radius: 0 !important;
}

:deep(.el-textarea__inner) {
  border-radius: 0 !important;
  border: 1px solid #dcdfe6 !important;
  box-shadow: none !important;
}

.avatar-uploader {
  text-align: center;
}

.avatar-uploader .el-upload {
  border: 1px dashed #d9d9d9;
  border-radius: 6px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
}

.avatar-uploader .el-upload:hover {
  border-color: #409eff;
}

.avatar-uploader-icon {
  font-size: 28px;
  color: #8c939d;
  width: 100px;
  height: 100px;
  line-height: 100px;
  text-align: center;
}

.avatar {
  width: 100px;
  height: 100px;
  display: block;
}

.security-options,
.notification-settings,
.privacy-settings,
.preference-settings {
  margin-top: 20px;
}

.security-options h4,
.notification-settings h4,
.privacy-settings h4,
.preference-settings h4 {
  margin: 0 0 15px 0;
  color: #333;
}

.security-tip,
.privacy-tip {
  margin: 10px 0 0 0;
  color: #666;
  font-size: 14px;
}

.el-divider {
  margin: 20px 0;
}

.dialog-footer {
  text-align: right;
}

/* 手机端布局 */
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
  
  .user-settings-container {
    padding: 10px;
  }
  
  .page-header {
    margin-bottom: 16px;
    
    :is(h1) {
      font-size: 1.5rem;
      margin-bottom: 8px;
    }
    
    :is(p) {
      font-size: 0.875rem;
    }
  }
  
  /* 手机端标签页 */
  .mobile-tabs-card {
    margin-bottom: 12px;
    
    :deep(.el-card__body) {
      padding: 12px;
    }
  }
  
  .mobile-settings-tabs {
    :deep(.el-tabs__header) {
      margin: 0;
    }
    
    :deep(.el-tabs__nav-wrap) {
      &::after {
        display: none;
      }
    }
    
    :deep(.el-tabs__item) {
      padding: 8px 12px;
      font-size: 0.875rem;
      height: auto;
      line-height: 1.5;
      
      :is(i) {
        font-size: 14px;
        margin-right: 4px;
      }
    }
    
    :deep(.el-tabs__content) {
      display: none;
    }
  }
  
  /* 手机端设置卡片 */
  .mobile-setting-card {
    margin-bottom: 12px;
    
    :deep(.el-card__header) {
      padding: 12px 16px;
      
      .card-header {
        font-size: 1rem;
        
        :is(i) {
          font-size: 16px;
        }
      }
    }
    
    :deep(.el-card__body) {
      padding: 16px;
    }
  }
  
  /* 手机端表单 */
  .mobile-form {
    :deep(.el-form-item) {
      margin-bottom: 18px;
      
      .el-form-item__label {
        width: 100% !important;
        text-align: left;
        margin-bottom: 8px;
        padding: 0;
        font-size: 14px;
        font-weight: 500;
        line-height: 1.5;
      }
      
      .el-form-item__content {
        width: 100%;
        
        .el-input,
        .el-select,
        .el-textarea {
          width: 100% !important;
        }
      }
    }
    
    :deep(.el-input__wrapper) {
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
      border-color: #409eff !important;
      box-shadow: none !important;
    }
    
    :deep(.el-select .el-input__wrapper) {
      border-radius: 0 !important;
    }
    
    :deep(.el-textarea__inner) {
      border-radius: 0 !important;
      border: 1px solid #dcdfe6 !important;
      box-shadow: none !important;
    }
  }
  
  /* 手机端头像上传 */
  .avatar-uploader {
    width: 100%;
    text-align: center;
    
    .el-upload {
      width: 100px;
      height: 100px;
      margin: 0 auto;
    }
    
    .avatar,
    .avatar-uploader-icon {
      width: 100px;
      height: 100px;
    }
  }
  
  /* 手机端单选组 */
  .mobile-radio-group {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 12px;
    
    :deep(.el-radio) {
      margin: 0;
      width: 100%;
      padding: 12px;
      border: 1px solid #e5e7eb;
      border-radius: 6px;
      transition: all 0.2s;
      
      &:hover {
        border-color: #409eff;
        background-color: #f0f9ff;
      }
      
      &.is-checked {
        border-color: #409eff;
        background-color: #ecf5ff;
      }
      
      .el-radio__label {
        padding-left: 8px;
        font-size: 14px;
      }
    }
  }
  
  /* 手机端复选框组 */
  .mobile-checkbox-group {
    display: flex;
    flex-direction: column;
    gap: 12px;
    
    :deep(.el-checkbox) {
      margin: 0;
      padding: 12px;
      border: 1px solid #e5e7eb;
      border-radius: 6px;
      transition: all 0.2s;
      
      &:hover {
        border-color: #409eff;
        background-color: #f0f9ff;
      }
      
      &.is-checked {
        border-color: #409eff;
        background-color: #ecf5ff;
      }
      
      .el-checkbox__label {
        padding-left: 8px;
        font-size: 14px;
      }
    }
  }
  
  /* 手机端开关 */
  :deep(.el-switch) {
    width: 100%;
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px;
    border: 1px solid #e5e7eb;
    border-radius: 6px;
    margin-bottom: 12px;
    
    .el-switch__label {
      font-size: 14px;
      flex: 1;
    }
  }
  
  /* 手机端按钮 */
  .mobile-setting-card {
    :deep(.el-button) {
      width: 100%;
      min-height: 44px;
      font-size: 16px;
      margin: 0;
    }
  }
  
  /* 手机端标题 */
  .security-options h4,
  .notification-settings h4,
  .privacy-settings h4,
  .preference-settings h4 {
    font-size: 1rem;
    margin-bottom: 12px;
    color: #333;
    font-weight: 600;
  }
  
  /* 手机端提示文字 */
  .security-tip,
  .privacy-tip {
    font-size: 0.8125rem;
    color: #666;
    line-height: 1.6;
    margin-top: 8px;
  }
  
  /* 手机端分割线 */
  :deep(.el-divider) {
    margin: 16px 0;
  }
  
  /* 手机端对话框 */
  :deep(.el-dialog) {
    width: 90% !important;
    margin: 5vh auto !important;
    max-height: 90vh;
    overflow-y: auto;
  }
  
  :deep(.el-dialog__body) {
    padding: 15px !important;
    max-height: calc(90vh - 120px);
    overflow-y: auto;
  }
  
  :deep(.el-dialog__header) {
    padding: 15px !important;
  }
  
  :deep(.el-dialog__footer) {
    padding: 15px !important;
    
    .el-button {
      width: 100%;
      margin: 0 0 10px 0 !important;
      min-height: 44px;
      font-size: 16px;
      
      &:last-child {
        margin-bottom: 0;
      }
    }
  }
}
</style>
