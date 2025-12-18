<template>
  <div class="admin-settings">
    <el-card>
      <template #header>
        <span>系统设置</span>
      </template>

      <el-tabs v-model="activeTab" type="border-card">
        <!-- 基本设置 -->
        <el-tab-pane label="基本设置" name="general">
          <el-form 
            :model="generalSettings" 
            :rules="generalRules" 
            ref="generalFormRef" 
            :label-width="isMobile ? '0' : '120px'"
            :label-position="isMobile ? 'top' : 'right'"
            class="settings-form"
          >
            <el-form-item label="网站名称" prop="site_name">
              <el-input v-model="generalSettings.site_name" />
            </el-form-item>
            <el-form-item label="网站描述" prop="site_description">
              <el-input v-model="generalSettings.site_description" type="textarea" />
            </el-form-item>
            <el-form-item label="网站域名" prop="domain_name">
              <el-input 
                v-model="generalSettings.domain_name" 
                placeholder="例如: dy.moneyfly.top (不需要 http:// 或 https://)"
              />
              <div :class="['form-tip', { 'mobile': isMobile }]">
                用于生成订阅地址和邮件中的链接。如果留空，将使用请求的域名。
              </div>
            </el-form-item>
            <el-form-item label="网站Logo">
              <el-upload
                class="avatar-uploader"
                :action="uploadUrl"
                :show-file-list="false"
                :on-success="handleLogoSuccess"
                :before-upload="beforeLogoUpload"
              >
                <img v-if="generalSettings.site_logo" :src="generalSettings.site_logo" class="avatar" />
                <el-icon v-else class="avatar-uploader-icon"><Plus /></el-icon>
              </el-upload>
            </el-form-item>
            <el-form-item label="默认主题" prop="default_theme">
              <el-select v-model="generalSettings.default_theme">
                <el-option label="浅色主题" value="light" />
                <el-option label="深色主题" value="dark" />
                <el-option label="跟随系统" value="auto" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveGeneralSettings" :class="{ 'full-width': isMobile }">保存基本设置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <!-- 注册设置 -->
        <el-tab-pane label="注册设置" name="registration">
          <el-form 
            :model="registrationSettings" 
            :label-width="isMobile ? '0' : '120px'"
            :label-position="isMobile ? 'top' : 'right'"
            class="settings-form"
          >
            <el-form-item label="开放注册">
              <el-switch v-model="registrationSettings.registration_enabled" />
            </el-form-item>
            <el-form-item label="邮箱验证">
              <el-switch v-model="registrationSettings.email_verification_required" />
            </el-form-item>
            <el-form-item label="最小密码长度" prop="min_password_length">
              <el-input-number 
                v-model="registrationSettings.min_password_length" 
                :min="6" 
                :max="20"
                :style="{ width: isMobile ? '100%' : '200px' }"
              />
            </el-form-item>
            <el-form-item label="邀请码注册">
              <el-switch v-model="registrationSettings.invite_code_required" />
            </el-form-item>
            <el-divider content-position="left">新用户默认订阅设置</el-divider>
            <el-form-item label="默认设备数" prop="default_subscription_device_limit">
              <el-input-number 
                v-model="registrationSettings.default_subscription_device_limit" 
                :min="1" 
                :max="100"
                :style="{ width: isMobile ? '100%' : '200px' }"
              />
              <div :class="['form-tip', { 'mobile': isMobile }]">
                新注册用户默认允许的设备数量
              </div>
            </el-form-item>
            <el-form-item label="默认订阅时长（月）" prop="default_subscription_duration_months">
              <el-input-number 
                v-model="registrationSettings.default_subscription_duration_months" 
                :min="1" 
                :max="120"
                :style="{ width: isMobile ? '100%' : '200px' }"
              />
              <div :class="['form-tip', { 'mobile': isMobile }]">
                新注册用户默认订阅的有效期（单位：月）
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveRegistrationSettings" :class="{ 'full-width': isMobile }">保存注册设置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <!-- 通知设置 -->
        <el-tab-pane label="通知设置" name="notification">
          <el-tabs type="border-card" class="notification-tabs">
            <!-- 客户通知 -->
            <el-tab-pane label="客户通知" name="customer">
              <el-alert
                title="客户通知设置"
                description="这些设置控制发送给客户的通知（邮件通知）。"
                type="info"
                :closable="false"
                style="margin-bottom: 20px"
              />
              <el-form
                :model="notificationSettings"
                :label-width="isMobile ? '0' : '120px'"
                :label-position="isMobile ? 'top' : 'right'"
                class="settings-form"
              >
                <el-form-item label="系统通知">
                  <el-switch v-model="notificationSettings.system_notifications" />
                </el-form-item>
                <el-form-item label="邮件通知">
                  <el-switch v-model="notificationSettings.email_notifications" />
                </el-form-item>
                <el-form-item label="订阅到期提醒">
                  <el-switch v-model="notificationSettings.subscription_expiry_notifications" />
                </el-form-item>
                <el-form-item label="新用户注册通知">
                  <el-switch v-model="notificationSettings.new_user_notifications" />
                </el-form-item>
                <el-form-item label="新订单通知">
                  <el-switch v-model="notificationSettings.new_order_notifications" />
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" @click="saveNotificationSettings" :class="{ 'full-width': isMobile }">保存客户通知设置</el-button>
                </el-form-item>
              </el-form>
            </el-tab-pane>

            <!-- 管理员通知 -->
            <el-tab-pane label="管理员通知" name="admin">
              <el-alert
                title="管理员通知设置"
                description="配置管理员通知方式，当系统发生重要事件时，您将收到通知。支持邮件、Telegram 和 Bark 三种方式。"
                type="info"
                :closable="false"
                style="margin-bottom: 20px"
              />
              <el-form
                :model="adminNotificationSettings"
                :label-width="isMobile ? '0' : '120px'"
                :label-position="isMobile ? 'top' : 'right'"
                class="settings-form"
              >
                <el-form-item label="启用管理员通知">
                  <el-switch v-model="adminNotificationSettings.admin_notification_enabled" />
                </el-form-item>

                <el-divider content-position="left">通知方式</el-divider>
                
                <el-form-item label="邮件通知">
                  <el-switch v-model="adminNotificationSettings.admin_email_notification" />
                </el-form-item>
                
                <el-form-item label="管理员邮箱" v-if="adminNotificationSettings.admin_email_notification">
                  <el-input
                    v-model="adminNotificationSettings.admin_notification_email"
                    placeholder="请输入接收通知的管理员邮箱"
                  />
                </el-form-item>
                
                <el-form-item v-if="adminNotificationSettings.admin_email_notification">
                  <el-button type="primary" @click="testAdminEmail" :loading="testingAdminEmail" :class="{ 'full-width': isMobile }">
                    测试邮件通知
                  </el-button>
                </el-form-item>

                <el-form-item label="Telegram 通知">
                  <el-switch v-model="adminNotificationSettings.admin_telegram_notification" />
                </el-form-item>
                
                <el-form-item label="Bot Token" v-if="adminNotificationSettings.admin_telegram_notification">
                  <el-input
                    v-model="adminNotificationSettings.admin_telegram_bot_token"
                    placeholder="请输入 Telegram Bot Token"
                    type="password"
                    show-password
                  />
                  <div :class="['form-tip', { 'mobile': isMobile }]">
                    在 @BotFather 创建机器人后获取
                  </div>
                </el-form-item>
                
                <el-form-item label="Chat ID" v-if="adminNotificationSettings.admin_telegram_notification">
                  <el-input
                    v-model="adminNotificationSettings.admin_telegram_chat_id"
                    placeholder="请输入 Telegram Chat ID"
                  />
                  <div :class="['form-tip', { 'mobile': isMobile }]">
                    发送消息给 @userinfobot 获取您的 Chat ID
                  </div>
                </el-form-item>
                
                <el-form-item v-if="adminNotificationSettings.admin_telegram_notification">
                  <el-button type="primary" @click="testAdminTelegram" :loading="testingAdminTelegram" :class="{ 'full-width': isMobile }">
                    测试 Telegram 通知
                  </el-button>
                </el-form-item>

                <el-form-item label="Bark 通知">
                  <el-switch v-model="adminNotificationSettings.admin_bark_notification" />
                </el-form-item>
                
                <el-form-item label="服务器地址" v-if="adminNotificationSettings.admin_bark_notification">
                  <el-input
                    v-model="adminNotificationSettings.admin_bark_server_url"
                    placeholder="https://api.day.app 或您的自建服务器地址"
                  />
                  <div :class="['form-tip', { 'mobile': isMobile }]">
                    默认: https://api.day.app，或填写您的自建 Bark 服务器地址
                  </div>
                </el-form-item>
                
                <el-form-item label="Device Key" v-if="adminNotificationSettings.admin_bark_notification">
                  <el-input
                    v-model="adminNotificationSettings.admin_bark_device_key"
                    placeholder="请输入 Bark Device Key"
                    type="password"
                    show-password
                  />
                  <div :class="['form-tip', { 'mobile': isMobile }]">
                    在 Bark 应用中获取您的 Device Key
                  </div>
                </el-form-item>
                
                <el-form-item v-if="adminNotificationSettings.admin_bark_notification">
                  <el-button type="primary" @click="testAdminBark" :loading="testingAdminBark" :class="{ 'full-width': isMobile }">
                    测试 Bark 通知
                  </el-button>
                </el-form-item>

                <el-divider content-position="left">通知功能选择</el-divider>
                
                <el-form-item label="订单支付成功">
                  <el-switch v-model="adminNotificationSettings.admin_notify_order_paid" />
                </el-form-item>
                
                <el-form-item label="新用户注册">
                  <el-switch v-model="adminNotificationSettings.admin_notify_user_registered" />
                </el-form-item>
                
                <el-form-item label="重置密码">
                  <el-switch v-model="adminNotificationSettings.admin_notify_password_reset" />
                </el-form-item>
                
                <el-form-item label="发送订阅">
                  <el-switch v-model="adminNotificationSettings.admin_notify_subscription_sent" />
                </el-form-item>
                
                <el-form-item label="重置订阅">
                  <el-switch v-model="adminNotificationSettings.admin_notify_subscription_reset" />
                </el-form-item>
                
                <el-form-item label="订阅到期">
                  <el-switch v-model="adminNotificationSettings.admin_notify_subscription_expired" />
                </el-form-item>
                
                <el-form-item label="管理员创建用户">
                  <el-switch v-model="adminNotificationSettings.admin_notify_user_created" />
                </el-form-item>
                
                <el-form-item label="订阅创建">
                  <el-switch v-model="adminNotificationSettings.admin_notify_subscription_created" />
                </el-form-item>

                <el-form-item>
                  <el-button type="primary" @click="saveAdminNotificationSettings" :class="{ 'full-width': isMobile }">
                    保存管理员通知设置
                  </el-button>
                </el-form-item>
              </el-form>
            </el-tab-pane>
          </el-tabs>
        </el-tab-pane>

        <!-- 公告管理 -->
        <el-tab-pane label="公告管理" name="announcement">
          <el-form 
            :model="announcementSettings" 
            :label-width="isMobile ? '0' : '120px'"
            :label-position="isMobile ? 'top' : 'right'"
            class="settings-form"
          >
            <el-form-item label="启用公告">
              <el-switch v-model="announcementSettings.announcement_enabled" />
              <div :class="['form-tip', { 'mobile': isMobile }]">
                开启后，用户登录时会看到公告弹窗
              </div>
            </el-form-item>
            <el-form-item label="公告内容" prop="announcement_content">
              <el-input 
                v-model="announcementSettings.announcement_content" 
                type="textarea" 
                :rows="8"
                placeholder="请输入公告内容，支持HTML格式"
              />
              <div :class="['form-tip', { 'mobile': isMobile }]">
                公告内容将在用户登录时以弹窗形式显示
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveAnnouncementSettings" :class="{ 'full-width': isMobile }">保存公告设置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <!-- 主题设置 -->
        <el-tab-pane label="主题设置" name="theme">
          <el-form 
            :model="themeSettings" 
            :label-width="isMobile ? '0' : '120px'"
            :label-position="isMobile ? 'top' : 'right'"
            class="settings-form"
          >
            <el-form-item label="默认主题" prop="default_theme">
              <el-select v-model="themeSettings.default_theme" :style="{ width: isMobile ? '100%' : '300px' }">
                <el-option label="浅色主题" value="light">
                  <span style="display: inline-block; width: 12px; height: 12px; background: #409EFF; border-radius: 2px; margin-right: 8px;"></span>
                  浅色主题
                </el-option>
                <el-option label="深色主题" value="dark">
                  <span style="display: inline-block; width: 12px; height: 12px; background: #1a1a1a; border-radius: 2px; margin-right: 8px;"></span>
                  深色主题
                </el-option>
                <el-option label="蓝色主题" value="blue">
                  <span style="display: inline-block; width: 12px; height: 12px; background: #1890ff; border-radius: 2px; margin-right: 8px;"></span>
                  蓝色主题
                </el-option>
                <el-option label="绿色主题" value="green">
                  <span style="display: inline-block; width: 12px; height: 12px; background: #52c41a; border-radius: 2px; margin-right: 8px;"></span>
                  绿色主题
                </el-option>
                <el-option label="紫色主题" value="purple">
                  <span style="display: inline-block; width: 12px; height: 12px; background: #722ed1; border-radius: 2px; margin-right: 8px;"></span>
                  紫色主题
                </el-option>
                <el-option label="橙色主题" value="orange">
                  <span style="display: inline-block; width: 12px; height: 12px; background: #fa8c16; border-radius: 2px; margin-right: 8px;"></span>
                  橙色主题
                </el-option>
                <el-option label="红色主题" value="red">
                  <span style="display: inline-block; width: 12px; height: 12px; background: #f5222d; border-radius: 2px; margin-right: 8px;"></span>
                  红色主题
                </el-option>
                <el-option label="青色主题" value="cyan">
                  <span style="display: inline-block; width: 12px; height: 12px; background: #13c2c2; border-radius: 2px; margin-right: 8px;"></span>
                  青色主题
                </el-option>
                <el-option label="Luck主题" value="luck">
                  <span style="display: inline-block; width: 12px; height: 12px; background: #FFD700; border-radius: 2px; margin-right: 8px;"></span>
                  Luck主题
                </el-option>
                <el-option label="Aurora主题" value="aurora">
                  <span style="display: inline-block; width: 12px; height: 12px; background: #7B68EE; border-radius: 2px; margin-right: 8px;"></span>
                  Aurora主题
                </el-option>
                <el-option label="跟随系统" value="auto">
                  <span style="display: inline-block; width: 12px; height: 12px; background: #909399; border-radius: 2px; margin-right: 8px;"></span>
                  跟随系统
                </el-option>
              </el-select>
            </el-form-item>
            <el-form-item label="允许用户自定义主题">
              <el-switch v-model="themeSettings.allow_user_theme" />
            </el-form-item>
            <el-form-item label="可用主题">
              <el-checkbox-group v-model="themeSettings.available_themes" :class="['theme-checkbox-group', { 'mobile': isMobile }]">
                <el-checkbox label="light">
                  <span style="display: inline-flex; align-items: center; gap: 6px;">
                    <span style="display: inline-block; width: 14px; height: 14px; background: #409EFF; border-radius: 2px;"></span>
                    浅色主题
                  </span>
                </el-checkbox>
                <el-checkbox label="dark">
                  <span style="display: inline-flex; align-items: center; gap: 6px;">
                    <span style="display: inline-block; width: 14px; height: 14px; background: #1a1a1a; border-radius: 2px;"></span>
                    深色主题
                  </span>
                </el-checkbox>
                <el-checkbox label="blue">
                  <span style="display: inline-flex; align-items: center; gap: 6px;">
                    <span style="display: inline-block; width: 14px; height: 14px; background: #1890ff; border-radius: 2px;"></span>
                    蓝色主题
                  </span>
                </el-checkbox>
                <el-checkbox label="green">
                  <span style="display: inline-flex; align-items: center; gap: 6px;">
                    <span style="display: inline-block; width: 14px; height: 14px; background: #52c41a; border-radius: 2px;"></span>
                    绿色主题
                  </span>
                </el-checkbox>
                <el-checkbox label="purple">
                  <span style="display: inline-flex; align-items: center; gap: 6px;">
                    <span style="display: inline-block; width: 14px; height: 14px; background: #722ed1; border-radius: 2px;"></span>
                    紫色主题
                  </span>
                </el-checkbox>
                <el-checkbox label="orange">
                  <span style="display: inline-flex; align-items: center; gap: 6px;">
                    <span style="display: inline-block; width: 14px; height: 14px; background: #fa8c16; border-radius: 2px;"></span>
                    橙色主题
                  </span>
                </el-checkbox>
                <el-checkbox label="red">
                  <span style="display: inline-flex; align-items: center; gap: 6px;">
                    <span style="display: inline-block; width: 14px; height: 14px; background: #f5222d; border-radius: 2px;"></span>
                    红色主题
                  </span>
                </el-checkbox>
                <el-checkbox label="cyan">
                  <span style="display: inline-flex; align-items: center; gap: 6px;">
                    <span style="display: inline-block; width: 14px; height: 14px; background: #13c2c2; border-radius: 2px;"></span>
                    青色主题
                  </span>
                </el-checkbox>
                <el-checkbox label="luck">
                  <span style="display: inline-flex; align-items: center; gap: 6px;">
                    <span style="display: inline-block; width: 14px; height: 14px; background: #FFD700; border-radius: 2px;"></span>
                    Luck主题
                  </span>
                </el-checkbox>
                <el-checkbox label="aurora">
                  <span style="display: inline-flex; align-items: center; gap: 6px;">
                    <span style="display: inline-block; width: 14px; height: 14px; background: #7B68EE; border-radius: 2px;"></span>
                    Aurora主题
                  </span>
                </el-checkbox>
                <el-checkbox label="auto">
                  <span style="display: inline-flex; align-items: center; gap: 6px;">
                    <span style="display: inline-block; width: 14px; height: 14px; background: #909399; border-radius: 2px;"></span>
                    跟随系统
                  </span>
                </el-checkbox>
              </el-checkbox-group>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveThemeSettings" :class="{ 'full-width': isMobile }">保存主题设置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>


        <!-- 安全设置 -->
        <el-tab-pane label="安全设置" name="security">
          <el-form 
            :model="securitySettings" 
            :label-width="isMobile ? '0' : '120px'"
            :label-position="isMobile ? 'top' : 'right'"
            class="settings-form"
          >
            <el-form-item label="登录失败限制" prop="login_fail_limit">
              <el-input-number 
                v-model="securitySettings.login_fail_limit" 
                :min="3" 
                :max="10"
                :style="{ width: isMobile ? '100%' : '200px' }"
              />
            </el-form-item>
            <el-form-item label="登录失败锁定时间(分钟)" prop="login_lock_time">
              <el-input-number 
                v-model="securitySettings.login_lock_time" 
                :min="5" 
                :max="60"
                :style="{ width: isMobile ? '100%' : '200px' }"
              />
            </el-form-item>
            <el-form-item label="会话超时时间(分钟)" prop="session_timeout">
              <el-input-number 
                v-model="securitySettings.session_timeout" 
                :min="15" 
                :max="1440"
                :style="{ width: isMobile ? '100%' : '200px' }"
              />
            </el-form-item>
            <el-form-item label="启用设备指纹">
              <el-switch v-model="securitySettings.device_fingerprint_enabled" />
              <el-text type="info" size="small" :class="{ 'mobile-tip': isMobile }">
                用于设备管理，识别订阅设备，不影响登录验证
              </el-text>
            </el-form-item>
            <el-form-item label="启用IP白名单">
              <el-switch v-model="securitySettings.ip_whitelist_enabled" />
            </el-form-item>
            <el-form-item label="IP白名单" v-if="securitySettings.ip_whitelist_enabled">
              <el-input v-model="securitySettings.ip_whitelist" type="textarea" rows="3" placeholder="每行一个IP地址" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveSecuritySettings" :class="{ 'full-width': isMobile }">保存安全设置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script>
import { ref, reactive, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { useApi } from '@/utils/api'
import { adminAPI } from '@/utils/api'
import { useThemeStore } from '@/store/theme'

export default {
  name: 'AdminSettings',
  components: {
    Plus
  },
  setup() {
    const api = useApi()
    const isMobile = ref(window.innerWidth <= 768)
    const themeStore = useThemeStore()
    const activeTab = ref('general')
    const generalFormRef = ref()
    const uploadUrl = '/api/v1/admin/upload'

    // 基本设置
    const generalSettings = reactive({
      site_name: '',
      site_description: '',
      domain_name: '',
      site_logo: '',
      default_theme: 'default'
    })

    const generalRules = {
      site_name: [
        { required: true, message: '请输入网站名称', trigger: 'blur' }
      ]
    }

    // 注册设置
    const registrationSettings = reactive({
      registration_enabled: true,
      email_verification_required: true,
      min_password_length: 8,
      invite_code_required: false,
      default_subscription_device_limit: 3,
      default_subscription_duration_months: 1
    })

    // 通知设置
    const notificationSettings = reactive({
      system_notifications: true,
      email_notifications: true,
      subscription_expiry_notifications: true,
      new_user_notifications: true,
      new_order_notifications: true
    })

    // 安全设置
    const securitySettings = reactive({
      login_fail_limit: 5,
      login_lock_time: 30,
      session_timeout: 120,
      device_fingerprint_enabled: true,
      ip_whitelist_enabled: false,
      ip_whitelist: ''
    })

    // 主题设置
    const themeSettings = reactive({
      default_theme: 'light',
      allow_user_theme: true,
      available_themes: ['light', 'dark', 'blue', 'green', 'purple', 'orange', 'red', 'cyan', 'luck', 'aurora', 'auto']
    })

    // 管理员通知设置
    const adminNotificationSettings = reactive({
      admin_notification_enabled: false,
      admin_email_notification: false,
      admin_telegram_notification: false,
      admin_bark_notification: false,
      admin_telegram_bot_token: '',
      admin_telegram_chat_id: '',
      admin_bark_server_url: 'https://api.day.app',
      admin_bark_device_key: '',
      admin_notification_email: '',
      admin_notify_order_paid: false,
      admin_notify_user_registered: false,
      admin_notify_password_reset: false,
      admin_notify_subscription_sent: false,
      admin_notify_subscription_reset: false,
      admin_notify_subscription_expired: false,
      admin_notify_user_created: false,
      admin_notify_subscription_created: false
    })

    // 公告设置
    const announcementSettings = reactive({
      announcement_enabled: false,
      announcement_content: ''
    })

    const testingAdminEmail = ref(false)
    const testingAdminTelegram = ref(false)
    const testingAdminBark = ref(false)

    const loadSettings = async () => {
      try {
        const response = await api.get('/admin/settings')
        // 检查响应格式 - 后端返回的是 ResponseBase 格式，数据在 response.data.data 中
        const settings = response.data?.data || response.data || {}
        // 加载各项设置
        if (settings.general) {
          Object.assign(generalSettings, settings.general)
          }
        if (settings.registration) {
          Object.assign(registrationSettings, settings.registration)
          }
        if (settings.notification) {
          Object.assign(notificationSettings, settings.notification)
          }
        if (settings.security) {
          Object.assign(securitySettings, settings.security)
          }
        if (settings.theme) {
          // 处理 available_themes，可能是字符串或数组
          const themeData = { ...settings.theme }
          if (themeData.available_themes) {
            if (typeof themeData.available_themes === 'string') {
              try {
                themeData.available_themes = JSON.parse(themeData.available_themes)
              } catch (e) {
                themeData.available_themes = ['light', 'dark', 'blue', 'green', 'purple', 'orange', 'red', 'cyan', 'luck', 'aurora', 'auto']
              }
            }
          }
          Object.assign(themeSettings, themeData)
          }
        if (settings.admin_notification) {
          Object.assign(adminNotificationSettings, settings.admin_notification)
          }
        if (settings.announcement) {
          Object.assign(announcementSettings, settings.announcement)
          }
      } catch (error) {
        ElMessage.error('加载设置失败: ' + (error.response?.data?.message || error.message || '未知错误'))
      }
    }

    const saveGeneralSettings = async () => {
      try {
        await generalFormRef.value.validate()
        const response = await api.put('/admin/settings/general', generalSettings)
        if (response.data && response.data.success !== false) {
          ElMessage.success('基本设置保存成功')
        } else {
          ElMessage.error(response.data?.message || '保存失败')
        }
      } catch (error) {
        console.error('保存基本设置失败:', error)
        ElMessage.error(error.response?.data?.message || '保存失败')
      }
    }

    const saveRegistrationSettings = async () => {
      try {
        const response = await api.put('/admin/settings/registration', registrationSettings)
        if (response.data && response.data.success !== false) {
          ElMessage.success('注册设置保存成功')
        } else {
          ElMessage.error(response.data?.message || '保存失败')
        }
      } catch (error) {
        console.error('保存注册设置失败:', error)
        ElMessage.error(error.response?.data?.message || '保存失败')
      }
    }

    const saveNotificationSettings = async () => {
      try {
        const response = await api.put('/admin/settings/notification', notificationSettings)
        if (response.data && response.data.success !== false) {
          ElMessage.success('通知设置保存成功')
        } else {
          ElMessage.error(response.data?.message || '保存失败')
        }
      } catch (error) {
        console.error('保存通知设置失败:', error)
        ElMessage.error(error.response?.data?.message || '保存失败')
      }
    }

    const saveSecuritySettings = async () => {
      try {
        const response = await api.put('/admin/settings/security', securitySettings)
        if (response.data && response.data.success !== false) {
          ElMessage.success('安全设置保存成功')
        } else {
          ElMessage.error(response.data?.message || '保存失败')
        }
      } catch (error) {
        console.error('保存安全设置失败:', error)
        ElMessage.error(error.response?.data?.message || '保存失败')
      }
    }

    const saveAnnouncementSettings = async () => {
      try {
        const response = await api.put('/admin/settings/announcement', announcementSettings)
        if (response.data && response.data.success !== false) {
          ElMessage.success('公告设置保存成功')
        } else {
          ElMessage.error(response.data?.message || '保存失败')
        }
      } catch (error) {
        console.error('保存公告设置失败:', error)
        ElMessage.error(error.response?.data?.message || '保存失败')
      }
    }

    const saveThemeSettings = async () => {
      try {
        const response = await api.put('/admin/settings/theme', themeSettings)
        if (response.data && response.data.success !== false) {
          // 立即应用主题设置
          if (themeSettings.default_theme) {
            await themeStore.setTheme(themeSettings.default_theme)
          }
          ElMessage.success('主题设置保存成功')
        } else {
          ElMessage.error(response.data?.message || '保存失败')
        }
      } catch (error) {
        console.error('保存主题设置失败:', error)
        ElMessage.error(error.response?.data?.message || '保存失败')
      }
    }

    const saveAdminNotificationSettings = async () => {
      try {
        await adminAPI.updateAdminNotificationSettings(adminNotificationSettings)
        ElMessage.success('管理员通知设置保存成功')
      } catch (error) {
        ElMessage.error('保存失败: ' + (error.response?.data?.message || error.message))
      }
    }

    const testAdminEmail = async () => {
      try {
        testingAdminEmail.value = true
        const response = await adminAPI.testAdminEmailNotification()
        if (response.data.success) {
          ElMessage.success('邮件测试消息已加入队列，请检查您的邮箱')
        } else {
          ElMessage.error(response.data.message || '测试失败')
        }
      } catch (error) {
        ElMessage.error('测试失败: ' + (error.response?.data?.message || error.message))
      } finally {
        testingAdminEmail.value = false
      }
    }

    const testAdminTelegram = async () => {
      try {
        testingAdminTelegram.value = true
        const response = await adminAPI.testAdminTelegramNotification()
        if (response.data.success) {
          ElMessage.success('Telegram 测试消息发送成功，请检查您的 Telegram')
        } else {
          ElMessage.error(response.data.message || '测试失败')
        }
      } catch (error) {
        ElMessage.error('测试失败: ' + (error.response?.data?.message || error.message))
      } finally {
        testingAdminTelegram.value = false
      }
    }

    const testAdminBark = async () => {
      try {
        testingAdminBark.value = true
        const response = await adminAPI.testAdminBarkNotification()
        if (response.data.success) {
          ElMessage.success('Bark 测试消息发送成功，请检查您的设备')
        } else {
          ElMessage.error(response.data.message || '测试失败')
        }
      } catch (error) {
        ElMessage.error('测试失败: ' + (error.response?.data?.message || error.message))
      } finally {
        testingAdminBark.value = false
      }
    }


    const handleLogoSuccess = (response) => {
      if (response && response.success) {
        generalSettings.site_logo = response.data?.url || response.url || ''
        ElMessage.success('Logo上传成功')
      } else if (response && response.data && response.data.url) {
        generalSettings.site_logo = response.data.url
        ElMessage.success('Logo上传成功')
      } else {
        ElMessage.error('Logo上传失败')
      }
    }

    const beforeLogoUpload = (file) => {
      const isImage = file.type.startsWith('image/')
      const isLt2M = file.size / 1024 / 1024 < 2

      if (!isImage) {
        ElMessage.error('只能上传图片文件!')
        return false
      }
      if (!isLt2M) {
        ElMessage.error('图片大小不能超过 2MB!')
        return false
      }
      return true
    }

    const handleResize = () => {
      isMobile.value = window.innerWidth <= 768
    }

    onMounted(() => {
      loadSettings()
      window.addEventListener('resize', handleResize)
    })

    onBeforeUnmount(() => {
      window.removeEventListener('resize', handleResize)
    })

    return {
      activeTab,
      generalSettings,
      isMobile,
      adminNotificationSettings,
      testingAdminEmail,
      testingAdminTelegram,
      testingAdminBark,
      saveAdminNotificationSettings,
      testAdminEmail,
      testAdminTelegram,
      testAdminBark,
      generalRules,
      registrationSettings,
      notificationSettings,
      securitySettings,
      themeSettings,
      generalFormRef,
      uploadUrl,
      saveGeneralSettings,
      saveRegistrationSettings,
      saveNotificationSettings,
      saveSecuritySettings,
      saveThemeSettings,
      handleLogoSuccess,
      beforeLogoUpload,
      announcementSettings,
      saveAnnouncementSettings
    }
  }
}
</script>

<style scoped>
.admin-settings {
  padding: 20px;
}

@media (max-width: 768px) {
  .admin-settings {
    padding: 10px;
  }
}

.avatar-uploader {
  text-align: center;
}

.avatar-uploader .avatar {
  width: 100px;
  height: 100px;
  display: block;
}

.avatar-uploader .el-upload {
  border: 1px dashed #d9d9d9;
  border-radius: 6px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  transition: var(--el-transition-duration-fast);
}

.avatar-uploader .el-upload:hover {
  border-color: var(--el-color-primary);
}

.avatar-uploader-icon {
  font-size: 28px;
  color: #8c939d;
  width: 100px;
  height: 100px;
  text-align: center;
  line-height: 100px;
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

/* 响应式样式 */
.settings-form :deep(.el-form-item) {
  margin-bottom: 18px;
}

.settings-form :deep(.el-form-item__label) {
  font-weight: 500;
  margin-bottom: 8px;
}

@media (max-width: 768px) {
  .settings-form :deep(.el-form-item) {
    margin-bottom: 20px;
  }
  
  .settings-form :deep(.el-form-item__label) {
    margin-bottom: 8px;
    padding-bottom: 0;
  }
  
  .settings-form :deep(.el-input),
  .settings-form :deep(.el-select),
  .settings-form :deep(.el-textarea),
  .settings-form :deep(.el-input-number) {
    width: 100% !important;
  }
  
  .full-width {
    width: 100%;
  }
  
  .theme-checkbox-group {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
  }
  
  .theme-checkbox-group.mobile {
    flex-direction: column;
    gap: 12px;
  }
  
  .theme-checkbox-group.mobile :deep(.el-checkbox) {
    width: 100%;
    margin-right: 0;
  }
  
  /* 通知设置中的嵌套标签页在移动端优化 */
  .notification-tabs {
    margin-top: 10px;
  }
  
  .notification-tabs :deep(.el-tabs__header) {
    margin-bottom: 15px;
  }
  
  .notification-tabs :deep(.el-tabs__nav-wrap) {
    overflow-x: auto;
  }
  
  /* 卡片在移动端的优化 */
  .admin-settings :deep(.el-card) {
    border-radius: 0;
    box-shadow: none;
    border: none;
  }
  
  .admin-settings :deep(.el-card__header) {
    padding: 15px;
    font-size: 16px;
  }
  
  .admin-settings :deep(.el-card__body) {
    padding: 15px;
  }
  
  /* 标签页在移动端的优化 */
  .admin-settings :deep(.el-tabs) {
    margin-top: 0;
  }
  
  .admin-settings :deep(.el-tabs__header) {
    margin-bottom: 15px;
  }
  
  .admin-settings :deep(.el-tabs__nav-wrap) {
    overflow-x: auto;
  }
  
  .admin-settings :deep(.el-tabs__item) {
    padding: 0 15px;
    font-size: 14px;
  }
  
  /* 分割线在移动端的优化 */
  .admin-settings :deep(.el-divider) {
    margin: 20px 0;
  }
  
  .admin-settings :deep(.el-divider__text) {
    font-size: 13px;
    padding: 0 10px;
  }
  
  /* 提示信息在移动端的优化 */
  .admin-settings :deep(.el-alert) {
    margin-bottom: 15px;
  }
  
  .admin-settings :deep(.el-alert__title) {
    font-size: 14px;
  }
  
  .admin-settings :deep(.el-alert__description) {
    font-size: 12px;
    margin-top: 5px;
  }
  
  /* 提示文字在移动端的优化 */
  .form-tip {
    font-size: 12px;
    color: #909399;
    margin-top: 5px;
    line-height: 1.5;
  }
  
  .form-tip.mobile {
    font-size: 11px;
    margin-top: 6px;
  }
  
  .mobile-tip {
    display: block;
    margin-top: 6px;
    margin-left: 0 !important;
    font-size: 11px;
  }
  
  /* 开关组件在移动端的优化 */
  .settings-form :deep(.el-switch) {
    margin-right: 0;
  }
}

/* 提示文字样式 */
.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 5px;
  line-height: 1.5;
}

/* 桌面端优化 */
@media (min-width: 769px) {
  .theme-checkbox-group {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
  }
  
  .theme-checkbox-group :deep(.el-checkbox) {
    min-width: 120px;
  }
}
</style> 