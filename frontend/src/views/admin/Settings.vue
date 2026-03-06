<template>
  <div class="admin-settings">
    <el-card>
      <template #header>
        <span>系统设置</span>
      </template>
      <el-tabs v-model="activeTab" type="border-card">
        <el-tab-pane label="基本设置" name="general">
          <el-form :model="generalSettings" :rules="generalRules" ref="generalFormRef" v-bind="formLayout" class="settings-form">
            <el-form-item label="网站名称" prop="site_name">
              <el-input v-model="generalSettings.site_name" />
            </el-form-item>
            <el-form-item label="网站描述" prop="site_description">
              <el-input v-model="generalSettings.site_description" type="textarea" />
            </el-form-item>
            <el-form-item label="网站域名" prop="domain_name">
              <el-input v-model="generalSettings.domain_name" placeholder="例如: example.com (不需要 http://)" />
              <div class="form-tip">用于生成订阅地址和邮件链接。留空则使用请求域名。</div>
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
            <el-divider content-position="left">客服联系方式</el-divider>
            <el-form-item label="售后QQ" prop="support_qq">
              <el-input v-model="generalSettings.support_qq" placeholder="请输入售后QQ号码" />
              <div class="form-tip">帮助中心显示，留空不显示。</div>
            </el-form-item>
            <el-form-item label="售后邮箱" prop="support_email">
              <el-input v-model="generalSettings.support_email" placeholder="例如: support@example.com" />
              <div class="form-tip">帮助中心显示，留空不显示。</div>
            </el-form-item>
            <el-divider content-position="left">用户界面设置</el-divider>
            <el-form-item label="统一认证页面">
              <el-switch v-model="generalSettings.unified_auth_enabled" />
              <div class="form-tip">开启后使用集成登录页；关闭后使用传统分离页面。</div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveGeneralSettings" :class="{ 'full-width': isMobile }">保存基本设置</el-button>
            </el-form-item>
            <el-divider content-position="left">GeoIP 数据库管理</el-divider>
            <el-form-item label="数据库状态">
              <div v-if="geoipStatus">
                <el-tag :type="geoipStatus.enabled ? 'success' : 'warning'" style="margin-right: 10px;">
                  {{ geoipStatus.enabled ? '已启用' : '未启用' }}
                </el-tag>
                <span v-if="geoipStatus.active_database" style="color: #67c23a; font-size: 12px;">
                  当前使用: {{ geoipStatus.active_database }}
                </span>
                <span v-else style="color: #f56c6c; font-size: 12px;">未找到数据库文件</span>
              </div>
            </el-form-item>
            <el-form-item label="已安装数据库" v-if="geoipStatus && geoipStatus.databases && geoipStatus.databases.length > 0">
              <el-table :data="geoipStatus.databases" style="width: 100%; margin-top: 10px;" size="small">
                <el-table-column prop="name" label="数据库名称" min-width="180">
                  <template #default="scope">
                    <span>{{ scope.row.name }}</span>
                    <el-tag v-if="scope.row.active" type="success" size="small" style="margin-left: 8px;">使用中</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="size" label="大小" width="100" />
                <el-table-column prop="modified" label="更新时间" width="160" />
                <el-table-column label="操作" width="100">
                  <template #default="scope">
                    <el-button
                      v-if="!scope.row.active"
                      type="primary"
                      size="small"
                      @click="switchDatabase(scope.row.path)"
                      :loading="switchingDatabase"
                    >
                      切换
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-form-item>
            <el-form-item label="选择数据库类型">
              <el-radio-group v-model="geoipDatabaseType">
                <el-radio label="dbip">DB-IP City Lite（推荐，中国数据详细）</el-radio>
                <el-radio label="geolite2">GeoLite2 City（MaxMind）</el-radio>
              </el-radio-group>
              <div class="form-tip" style="margin-top: 8px;">
                <div v-if="geoipDatabaseType === 'dbip'">
                  • DB-IP 提供更详细的中国城市数据<br>
                  • 完全免费，无需注册<br>
                  • 文件大小约 125MB
                </div>
                <div v-else>
                  • GeoLite2 是广泛使用的数据库<br>
                  • 部分中国 IP 城市数据不完整<br>
                  • 文件大小约 60MB
                </div>
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="updateGeoIPDatabase" :loading="geoipUpdating" :class="{ 'full-width': isMobile }">
                {{ geoipUpdating ? '下载中...' : '下载/更新数据库' }}
              </el-button>
              <div class="form-tip" style="margin-top: 10px;">
                点击下载最新的 GeoIP 数据库。建议每月更新一次以获取最新的 IP 地址分配信息。
              </div>
            </el-form-item>
            <el-divider content-position="left">缓存管理</el-divider>
            <el-form-item label="Redis 缓存">
              <el-button type="danger" @click="flushCache" :loading="cacheClearing" :class="{ 'full-width': isMobile }">
                {{ cacheClearing ? '清除中...' : '清除所有缓存' }}
              </el-button>
              <div class="form-tip" style="margin-top: 10px;">
                清除所有 Redis 缓存数据。适用于代码更新、数据不一致等情况。清除后系统会自动重新缓存。
              </div>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="注册设置" name="registration">
          <el-form :model="registrationSettings" v-bind="formLayout" class="settings-form">
            <el-form-item label="开放注册">
              <el-switch v-model="registrationSettings.registration_enabled" />
            </el-form-item>
            <el-form-item label="邮箱验证">
              <el-switch v-model="registrationSettings.email_verification_required" />
            </el-form-item>
            <el-form-item label="最小密码长度" prop="min_password_length">
              <el-input v-model.number="registrationSettings.min_password_length" type="number" class="short-input" />
            </el-form-item>
            <el-form-item label="邀请码注册">
              <el-switch v-model="registrationSettings.invite_code_required" />
            </el-form-item>
            <el-divider content-position="left">新用户默认订阅设置</el-divider>
            <el-form-item label="默认设备数" prop="default_subscription_device_limit">
              <el-input v-model.number="registrationSettings.default_subscription_device_limit" type="number" class="short-input" />
              <div class="form-tip">新注册用户默认允许的设备数量</div>
            </el-form-item>
            <el-form-item label="默认订阅时长（月）" prop="default_subscription_duration_months">
              <el-input v-model.number="registrationSettings.default_subscription_duration_months" type="number" class="short-input" />
              <div class="form-tip">新注册用户默认订阅有效期</div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveRegistrationSettings" :class="{ 'full-width': isMobile }">保存注册设置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="通知设置" name="notification">
          <div class="notification-container">
            <!-- 客户通知部分 -->
            <el-card class="notification-section" shadow="never">
              <template #header>
                <div class="section-header">
                  <span class="section-title">客户通知</span>
                  <el-tag type="info" size="small">控制发送给客户的邮件通知</el-tag>
                </div>
              </template>
              <el-form :model="notificationSettings" v-bind="formLayout" class="settings-form">
                <el-form-item label="系统通知"><el-switch v-model="notificationSettings.system_notifications" /></el-form-item>
                <el-form-item label="邮件通知"><el-switch v-model="notificationSettings.email_notifications" /></el-form-item>
                <el-form-item label="订阅到期提醒"><el-switch v-model="notificationSettings.subscription_expiry_notifications" /></el-form-item>
                <el-form-item label="提醒冷却(小时)">
                  <el-input-number
                    v-model="notificationSettings.subscription_expiry_reminder_cooldown_hours"
                    :min="0"
                    :max="720"
                    :step="1"
                  />
                  <div class="form-tip">同一用户同主题到期提醒的最小发送间隔，0 表示不限制。</div>
                </el-form-item>
                <el-form-item label="单用户每日上限">
                  <el-input-number
                    v-model="notificationSettings.subscription_expiry_reminder_daily_limit"
                    :min="0"
                    :max="20"
                    :step="1"
                  />
                  <div class="form-tip">同一用户每天最多接收多少封到期提醒，0 表示不限制。</div>
                </el-form-item>
                <el-form-item label="新用户注册通知"><el-switch v-model="notificationSettings.new_user_notifications" /></el-form-item>
                <el-form-item label="新订单通知"><el-switch v-model="notificationSettings.new_order_notifications" /></el-form-item>
                <el-form-item>
                  <el-button type="primary" @click="saveNotificationSettings" :class="{ 'full-width': isMobile }">保存客户通知设置</el-button>
                </el-form-item>
              </el-form>
            </el-card>

            <!-- 管理员通知部分 -->
            <el-card class="notification-section" shadow="never">
              <template #header>
                <div class="section-header">
                  <span class="section-title">管理员通知</span>
                  <el-tag type="warning" size="small">支持邮件、Telegram 和 Bark 三种方式</el-tag>
                </div>
              </template>
              <el-form :model="adminNotificationSettings" v-bind="formLayout" class="settings-form">
                <el-form-item label="启用管理员通知"><el-switch v-model="adminNotificationSettings.admin_notification_enabled" /></el-form-item>

                <el-divider content-position="left">通知方式</el-divider>

                <!-- 邮件通知 -->
                <el-form-item label="邮件通知"><el-switch v-model="adminNotificationSettings.admin_email_notification" /></el-form-item>
                <template v-if="adminNotificationSettings.admin_email_notification">
                  <el-form-item label="管理员邮箱">
                    <el-input v-model="adminNotificationSettings.admin_notification_email" placeholder="请输入接收邮箱" />
                  </el-form-item>
                  <el-form-item>
                    <el-button type="primary" @click="testNotification('email')" :loading="testingStates.email" :class="{ 'full-width': isMobile }">测试邮件通知</el-button>
                  </el-form-item>
                </template>

                <!-- Telegram 通知 -->
                <el-form-item label="Telegram 通知"><el-switch v-model="adminNotificationSettings.admin_telegram_notification" /></el-form-item>
                <template v-if="adminNotificationSettings.admin_telegram_notification">
                  <el-form-item label="Bot Token">
                    <el-input v-model="adminNotificationSettings.admin_telegram_bot_token" type="password" show-password />
                    <div class="form-tip">在 @BotFather 获取</div>
                  </el-form-item>
                  <el-form-item label="Chat ID">
                    <el-input v-model="adminNotificationSettings.admin_telegram_chat_id" type="password" show-password />
                    <div class="form-tip">发送消息给 @userinfobot 获取</div>
                  </el-form-item>
                  <el-form-item>
                    <el-button type="primary" @click="testNotification('telegram')" :loading="testingStates.telegram" :class="{ 'full-width': isMobile }">测试 Telegram 通知</el-button>
                  </el-form-item>
                </template>

                <!-- Bark 通知 -->
                <el-form-item label="Bark 通知"><el-switch v-model="adminNotificationSettings.admin_bark_notification" /></el-form-item>
                <template v-if="adminNotificationSettings.admin_bark_notification">
                  <el-form-item label="服务器地址">
                    <el-input v-model="adminNotificationSettings.admin_bark_server_url" placeholder="默认: https://api.day.app" />
                  </el-form-item>
                  <el-form-item label="Device Key">
                    <el-input v-model="adminNotificationSettings.admin_bark_device_key" type="password" show-password />
                  </el-form-item>
                  <el-form-item>
                    <el-button type="primary" @click="testNotification('bark')" :loading="testingStates.bark" :class="{ 'full-width': isMobile }">测试 Bark 通知</el-button>
                  </el-form-item>
                </template>

                <el-divider content-position="left">通知事件开关</el-divider>
                <div class="notification-events-grid">
                   <el-form-item label="订单支付成功"><el-switch v-model="adminNotificationSettings.admin_notify_order_paid" /></el-form-item>
                   <el-form-item label="新用户注册"><el-switch v-model="adminNotificationSettings.admin_notify_user_registered" /></el-form-item>
                   <el-form-item label="重置密码"><el-switch v-model="adminNotificationSettings.admin_notify_password_reset" /></el-form-item>
                   <el-form-item label="发送订阅"><el-switch v-model="adminNotificationSettings.admin_notify_subscription_sent" /></el-form-item>
                   <el-form-item label="重置订阅"><el-switch v-model="adminNotificationSettings.admin_notify_subscription_reset" /></el-form-item>
                   <el-form-item label="订阅到期"><el-switch v-model="adminNotificationSettings.admin_notify_subscription_expired" /></el-form-item>
                   <el-form-item label="管理员创建用户"><el-switch v-model="adminNotificationSettings.admin_notify_user_created" /></el-form-item>
                   <el-form-item label="订阅创建"><el-switch v-model="adminNotificationSettings.admin_notify_subscription_created" /></el-form-item>
                </div>

                <el-divider content-position="left">安全告警</el-divider>
                <el-form-item label="管理员账户异常登录告警">
                  <el-switch v-model="adminNotificationSettings.admin_abnormal_login_alert_enabled" />
                  <div class="form-tip">开启后，管理员在新设备或异地登录时会收到邮件与站内告警（管理员个人通知设置中需同时开启「异常登录/设备告警」）</div>
                </el-form-item>

                <el-form-item>
                  <el-button type="primary" @click="saveAdminNotificationSettings" :class="{ 'full-width': isMobile }">保存管理员通知设置</el-button>
                </el-form-item>
              </el-form>
            </el-card>
          </div>
        </el-tab-pane>
        <el-tab-pane label="公告管理" name="announcement">
          <el-form :model="announcementSettings" v-bind="formLayout" class="settings-form">
            <el-form-item label="启用公告">
              <el-switch v-model="announcementSettings.announcement_enabled" />
              <div class="form-tip">用户登录时会看到公告弹窗</div>
            </el-form-item>
            <el-form-item label="公告内容" prop="announcement_content">
              <el-input v-model="announcementSettings.announcement_content" type="textarea" :rows="8" placeholder="支持HTML格式" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveAnnouncementSettings" :class="{ 'full-width': isMobile }">保存公告设置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="主题设置" name="theme">
          <el-form :model="themeSettings" v-bind="formLayout" class="settings-form">
            <el-form-item label="默认主题" prop="default_theme">
              <el-select v-model="themeSettings.default_theme" :style="{ width: isMobile ? '100%' : '300px' }">
                <el-option 
                  v-for="theme in themeOptions" 
                  :key="theme.value" 
                  :label="theme.label" 
                  :value="theme.value"
                >
                  <span :style="{ backgroundColor: theme.color }" class="theme-color-block"></span>
                  {{ theme.label }}
                </el-option>
              </el-select>
            </el-form-item>
            <el-form-item label="允许用户自定义">
              <el-switch v-model="themeSettings.allow_user_theme" />
            </el-form-item>
            <el-form-item label="可用主题">
              <el-checkbox-group v-model="themeSettings.available_themes" :class="['theme-checkbox-group', { 'mobile': isMobile }]">
                <el-checkbox v-for="theme in themeOptions" :key="theme.value" :label="theme.value">
                  <span class="flex-align-center">
                    <span :style="{ backgroundColor: theme.color }" class="theme-color-block-sm"></span>
                    {{ theme.label }}
                  </span>
                </el-checkbox>
              </el-checkbox-group>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveThemeSettings" :class="{ 'full-width': isMobile }">保存主题设置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="节点健康检查" name="node-health">
          <el-alert title="节点健康检查设置" description="配置自动健康检查参数。默认使用TCP连接测试。" type="info" :closable="false" class="mb-20" />
          <el-form :model="nodeHealthSettings" v-bind="formLayout" class="settings-form">
            <el-form-item label="检查间隔(分钟)">
              <el-input v-model.number="nodeHealthSettings.check_interval" type="number" class="short-input" />
              <div class="form-tip">建议30-60分钟</div>
            </el-form-item>
            <el-form-item label="最大允许延迟(ms)">
              <el-input v-model.number="nodeHealthSettings.max_latency" type="number" class="short-input" />
              <div class="form-tip">超过此延迟标记为超时，建议3000ms</div>
            </el-form-item>
            <el-form-item label="测试超时时间(秒)">
              <el-input v-model.number="nodeHealthSettings.test_timeout" type="number" class="short-input" />
              <div class="form-tip">单节点超时时间，建议5秒</div>
            </el-form-item>
            <el-form-item label="测速网站URL">
              <el-input v-model="nodeHealthSettings.test_url" placeholder="例如: https://ping.pe" style="width: 100%; max-width: 400px;" />
              <div class="form-tip">
                用于测试节点延迟的网页地址。<br />
                推荐使用: https://ping.pe<br />
                留空则使用TCP直接连接测试
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveNodeHealthSettings" :class="{ 'full-width': isMobile }">保存节点健康检查设置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="安全设置" name="security">
          <el-form :model="securitySettings" v-bind="formLayout" class="settings-form">
            <el-form-item label="登录失败限制" prop="login_fail_limit">
              <el-input v-model.number="securitySettings.login_fail_limit" type="number" class="short-input" />
            </el-form-item>
            <el-form-item label="锁定时间(分钟)" prop="login_lock_time">
              <el-input v-model.number="securitySettings.login_lock_time" type="number" class="short-input" />
            </el-form-item>
            <el-form-item label="会话超时(分钟)" prop="session_timeout">
              <el-input v-model.number="securitySettings.session_timeout" type="number" class="short-input" />
            </el-form-item>
            <el-form-item label="启用IP白名单">
              <el-switch v-model="securitySettings.ip_whitelist_enabled" />
            </el-form-item>
            <el-form-item label="IP白名单" v-if="securitySettings.ip_whitelist_enabled">
              <el-input v-model="securitySettings.ip_whitelist" type="textarea" :rows="3" placeholder="每行一个IP地址" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveSecuritySettings" :class="{ 'full-width': isMobile }">保存安全设置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="备份设置" name="backup">
          <el-alert title="数据库自动备份" description="配置数据库自动备份到 Gitee 或 GitHub。" type="info" :closable="false" class="mb-20" />
          <el-form :model="backupSettings" v-bind="formLayout" class="settings-form">
            <el-divider content-position="left">备份目标选择</el-divider>
            <el-form-item label="备份目标">
              <el-radio-group v-model="backupSettings.backup_target">
                <el-radio label="gitee">Gitee</el-radio>
                <el-radio label="github">GitHub</el-radio>
              </el-radio-group>
              <div class="form-tip">选择备份上传的目标平台。</div>
            </el-form-item>
            <el-divider content-position="left">Gitee 配置</el-divider>
            <el-form-item label="启用 Gitee 备份">
              <el-switch v-model="backupSettings.backup_gitee_enabled" />
              <div class="form-tip">备份自动上传到 Gitee 仓库。</div>
            </el-form-item>
            <template v-if="backupSettings.backup_gitee_enabled">
              <el-form-item label="Access Token">
                <el-input v-model="backupSettings.backup_gitee_token" type="password" show-password placeholder="Gitee Access Token" style="width: 100%; max-width: 400px;" />
                <div class="form-tip">需要 "projects" 权限。</div>
              </el-form-item>
              <el-form-item label="仓库所有者">
                <el-input v-model="backupSettings.backup_gitee_owner" placeholder="例如：moneyfly" style="width: 100%; max-width: 400px;" />
              </el-form-item>
              <el-form-item label="仓库名称">
                <el-input v-model="backupSettings.backup_gitee_repo" placeholder="例如：backup" style="width: 100%; max-width: 400px;" />
                <div class="form-tip">格式：YYYY-MM-DD/backup.zip</div>
              </el-form-item>
              <el-form-item>
                <el-button 
                  type="primary" 
                  @click="testGiteeConnection" 
                  :loading="testingStates.gitee"
                  :disabled="!backupSettings.backup_gitee_token"
                  :class="{ 'full-width': isMobile }"
                >
                  测试连接
                </el-button>
              </el-form-item>
            </template>
            <el-divider content-position="left">GitHub 配置</el-divider>
            <el-form-item label="启用 GitHub 备份">
              <el-switch v-model="backupSettings.backup_github_enabled" />
              <div class="form-tip">备份自动上传到 GitHub 仓库。</div>
            </el-form-item>
            <template v-if="backupSettings.backup_github_enabled">
              <el-form-item label="Access Token">
                <el-input v-model="backupSettings.backup_github_token" type="password" show-password placeholder="GitHub Personal Access Token" style="width: 100%; max-width: 400px;" />
                <div class="form-tip">需要 "repo" 权限。</div>
              </el-form-item>
              <el-form-item label="仓库所有者">
                <el-input v-model="backupSettings.backup_github_owner" placeholder="例如：moneyfly1" style="width: 100%; max-width: 400px;" />
              </el-form-item>
              <el-form-item label="仓库名称">
                <el-input v-model="backupSettings.backup_github_repo" placeholder="例如：backup" style="width: 100%; max-width: 400px;" />
                <div class="form-tip">格式：YYYY-MM-DD/backup.zip</div>
              </el-form-item>
              <el-form-item>
                <el-button 
                  type="primary" 
                  @click="testGitHubConnection" 
                  :loading="testingStates.github"
                  :disabled="!backupSettings.backup_github_token"
                  :class="{ 'full-width': isMobile }"
                >
                  测试连接
                </el-button>
              </el-form-item>
            </template>
            <el-divider content-position="left">自动备份设置</el-divider>
            <el-form-item label="启用自动备份">
              <el-switch v-model="backupSettings.backup_auto_enabled" />
            </el-form-item>
            <el-form-item label="备份间隔(小时)" v-if="backupSettings.backup_auto_enabled">
              <el-input v-model.number="backupSettings.backup_auto_interval" type="number" :min="1" class="short-input" />
              <div class="form-tip">建议12-24小时。</div>
            </el-form-item>
            <el-divider content-position="left">手动备份</el-divider>
            <el-form-item>
              <el-button type="success" @click="createManualBackup" :loading="creatingBackup" :class="{ 'full-width': isMobile }">
                立即备份
              </el-button>
              <div class="form-tip" style="margin-top: 10px;">立即创建备份并上传（如已启用远程备份）。</div>
            </el-form-item>
            <el-form-item v-if="uploadStatus || uploadTaskId">
              <el-alert 
                :title="uploadStatus?.status === 'uploading' ? (uploadTarget === 'github' ? '正在上传到GitHub...' : '正在上传到Gitee...') : uploadStatus?.status === 'success' ? '上传成功' : uploadStatus?.status === 'failed' ? '上传失败' : '准备上传...'"
                :type="uploadStatus?.status === 'uploading' ? 'info' : uploadStatus?.status === 'success' ? 'success' : uploadStatus?.status === 'failed' ? 'error' : 'info'"
                :closable="uploadStatus?.status !== 'uploading'"
                @close="stopStatusPolling"
                show-icon
              >
                <template #default>
                  <div v-if="uploadStatus?.status === 'uploading' || !uploadStatus">
                    <el-progress 
                      :percentage="uploadStatus?.progress || 0" 
                      :status="uploadStatus?.status === 'uploading' ? null : uploadStatus?.status === 'success' ? 'success' : 'exception'"
                      :stroke-width="8"
                    />
                    <div style="margin-top: 8px; font-size: 12px; color: #909399;">
                      {{ uploadStatus?.message || '正在准备上传...' }}
                      <span v-if="uploadStatus?.file_size"> | 文件大小: {{ (uploadStatus.file_size / 1024 / 1024).toFixed(2) }} MB</span>
                    </div>
                  </div>
                  <div v-else>
                    <div>{{ uploadStatus.message }}</div>
                    <div v-if="uploadStatus.error" style="margin-top: 4px; font-size: 12px; color: #f56c6c;">错误: {{ uploadStatus.error }}</div>
                    <div v-if="uploadStatus.finish_time" style="margin-top: 4px; font-size: 12px; color: #909399;">
                      完成时间: {{ new Date(uploadStatus.finish_time).toLocaleString() }}
                    </div>
                  </div>
                </template>
              </el-alert>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveBackupSettings" :class="{ 'full-width': isMobile }">保存备份设置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>
<script>
import { ref, reactive, onMounted, onBeforeUnmount, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { useApi, adminAPI } from '@/utils/api'
import { useThemeStore } from '@/store/theme'
const THEME_OPTIONS = [
  { label: '浅色主题', value: 'light', color: '#409EFF' },
  { label: '深色主题', value: 'dark', color: '#1a1a1a' },
  { label: '蓝色主题', value: 'blue', color: '#1890ff' },
  { label: '绿色主题', value: 'green', color: '#52c41a' },
  { label: '紫色主题', value: 'purple', color: '#722ed1' },
  { label: '橙色主题', value: 'orange', color: '#fa8c16' },
  { label: '红色主题', value: 'red', color: '#f5222d' },
  { label: '青色主题', value: 'cyan', color: '#13c2c2' },
  { label: 'Luck主题', value: 'luck', color: '#FFD700' },
  { label: 'Aurora主题', value: 'aurora', color: '#7B68EE' },
  { label: '跟随系统', value: 'auto', color: '#909399' }
]
export default {
  name: 'AdminSettings',
  components: { Plus },
  setup() {
    const api = useApi()
    const isMobile = ref(window.innerWidth <= 768)
    const themeStore = useThemeStore()
    const activeTab = ref('general')
    const generalFormRef = ref()
    const uploadUrl = '/api/v1/admin/upload'
    const testingStates = reactive({
      email: false,
      telegram: false,
      bark: false,
      gitee: false,
      github: false
    })
    const creatingBackup = ref(false)
    const geoipStatus = ref(null)
    const geoipUpdating = ref(false)
    const geoipDatabaseType = ref('dbip')
    const switchingDatabase = ref(false)
    const cacheClearing = ref(false)
    const uploadStatus = ref(null)
    const uploadTaskId = ref(null)
    const uploadTarget = ref('gitee')
    const uploadStatusInterval = ref(null)
    const formLayout = computed(() => ({
      labelWidth: isMobile.value ? '0' : '120px',
      labelPosition: isMobile.value ? 'top' : 'right'
    }))
    const generalSettings = reactive({
      site_name: '', site_description: '', domain_name: '', site_logo: '',
      default_theme: 'default', support_qq: '', support_email: '', unified_auth_enabled: false
    })
    const registrationSettings = reactive({
      registration_enabled: true, email_verification_required: true,
      min_password_length: 8, invite_code_required: false,
      default_subscription_device_limit: 3, default_subscription_duration_months: 1
    })
    const notificationSettings = reactive({
      system_notifications: true, email_notifications: true,
      subscription_expiry_notifications: true,
      subscription_expiry_reminder_cooldown_hours: 24,
      subscription_expiry_reminder_daily_limit: 1,
      new_user_notifications: true, new_order_notifications: true
    })
    const securitySettings = reactive({
      login_fail_limit: 5, login_lock_time: 30, session_timeout: 120,
      ip_whitelist_enabled: false, ip_whitelist: '',
      abnormal_login_alert_enabled: true
    })
    const themeSettings = reactive({
      default_theme: 'light', allow_user_theme: true,
      available_themes: THEME_OPTIONS.map(t => t.value)
    })
    const adminNotificationSettings = reactive({
      admin_notification_enabled: false,
      admin_email_notification: false, admin_notification_email: '',
      admin_telegram_notification: false, admin_telegram_bot_token: '', admin_telegram_chat_id: '',
      admin_bark_notification: false, admin_bark_server_url: 'https://api.day.app', admin_bark_device_key: '',
      admin_notify_order_paid: false, admin_notify_user_registered: false,
      admin_notify_password_reset: false, admin_notify_subscription_sent: false,
      admin_notify_subscription_reset: false, admin_notify_subscription_expired: false,
      admin_notify_user_created: false, admin_notify_subscription_created: false,
      admin_abnormal_login_alert_enabled: true
    })
    const announcementSettings = reactive({ announcement_enabled: false, announcement_content: '' })
    const nodeHealthSettings = reactive({
      check_interval: 30, max_latency: 3000, test_timeout: 5, test_url: 'https://ping.pe'
    })
    const backupSettings = reactive({
      backup_target: 'gitee',
      backup_gitee_enabled: false, backup_gitee_token: '',
      backup_gitee_owner: 'moneyfly', backup_gitee_repo: 'backup',
      backup_github_enabled: false, backup_github_token: '',
      backup_github_owner: 'moneyfly1', backup_github_repo: 'backup',
      backup_auto_enabled: false, backup_auto_interval: 24
    })
    const generalRules = {
      site_name: [{ required: true, message: '请输入网站名称', trigger: 'blur' }]
    }
    const formatFileSize = (bytes) => {
      if (!bytes) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    }
    const toBool = (val) => val === true || val === 'true'
    const loadGeoIPStatus = async () => {
      try {
        const res = await api.get('/admin/settings/geoip/status')
        geoipStatus.value = res.data?.data || res.data || {}
      } catch (e) { console.error('GeoIP Status Error', e) }
    }
    const loadSettings = async () => {
      try {
        const { data: res } = await api.get('/admin/settings')
        const data = res?.data || res || {}
        if (data.general) {
          Object.assign(generalSettings, data.general)
          generalSettings.unified_auth_enabled = toBool(data.general.unified_auth_enabled)
          if (data.general.node_health_check_interval) nodeHealthSettings.check_interval = +data.general.node_health_check_interval
          if (data.general.node_max_latency) nodeHealthSettings.max_latency = +data.general.node_max_latency
          if (data.general.node_test_timeout) nodeHealthSettings.test_timeout = +data.general.node_test_timeout
          if (data.general.test_url) nodeHealthSettings.test_url = data.general.test_url
        }
        if (data.node_health) {
          Object.assign(nodeHealthSettings, data.node_health)
          if (data.node_health.test_url) nodeHealthSettings.test_url = data.node_health.test_url
        }
        if (data.registration) Object.assign(registrationSettings, data.registration)
        if (data.notification) {
          Object.assign(notificationSettings, data.notification)
          const cooldown = parseInt(notificationSettings.subscription_expiry_reminder_cooldown_hours, 10)
          notificationSettings.subscription_expiry_reminder_cooldown_hours = Number.isNaN(cooldown) ? 24 : cooldown
          const dailyLimit = parseInt(notificationSettings.subscription_expiry_reminder_daily_limit, 10)
          notificationSettings.subscription_expiry_reminder_daily_limit = Number.isNaN(dailyLimit) ? 1 : dailyLimit
        }
        if (data.security) Object.assign(securitySettings, data.security)
        if (data.announcement) Object.assign(announcementSettings, data.announcement)
        if (data.theme) {
          Object.assign(themeSettings, data.theme)
          if (typeof themeSettings.available_themes === 'string') {
            try { themeSettings.available_themes = JSON.parse(themeSettings.available_themes) }
            catch { themeSettings.available_themes = THEME_OPTIONS.map(t => t.value) }
          }
        }
        if (data.admin_notification) {
          const stringFields = ['admin_telegram_chat_id', 'admin_telegram_bot_token', 'admin_bark_device_key', 'admin_notification_email', 'admin_bark_server_url']
          Object.keys(data.admin_notification).forEach(key => {
            if (stringFields.includes(key)) adminNotificationSettings[key] = String(data.admin_notification[key] || '')
            else adminNotificationSettings[key] = toBool(data.admin_notification[key])
          })
        }
        if (data.backup) {
          Object.assign(backupSettings, data.backup)
          backupSettings.backup_target = backupSettings.backup_target || 'gitee'
          backupSettings.backup_gitee_enabled = toBool(backupSettings.backup_gitee_enabled)
          backupSettings.backup_github_enabled = toBool(backupSettings.backup_github_enabled)
          backupSettings.backup_auto_enabled = toBool(backupSettings.backup_auto_enabled)
          backupSettings.backup_auto_interval = parseInt(backupSettings.backup_auto_interval) || 24
        }
      } catch (error) {
        ElMessage.error('加载设置失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const handleSave = async (apiCall, successMsg, validationRef = null) => {
      try {
        if (validationRef) await validationRef.validate()
        const res = await apiCall()
        if (res.data?.success !== false) {
          ElMessage.success(successMsg)
          return true
        }
        throw new Error(res.data?.message || '保存失败')
      } catch (error) {
        console.error(`保存失败:`, error)
        ElMessage.error(error.response?.data?.message || error.message || '保存失败')
        return false
      }
    }
    const saveGeneralSettings = async () => {
      const data = { ...generalSettings, unified_auth_enabled: generalSettings.unified_auth_enabled }
      const success = await handleSave(
        () => api.put('/admin/settings/general', data), 
        '基本设置保存成功', 
        generalFormRef.value
      )
      if (success) await loadSettings()
    }
    const saveRegistrationSettings = () => handleSave(() => api.put('/admin/settings/registration', registrationSettings), '注册设置保存成功')
    const saveNotificationSettings = () => handleSave(() => api.put('/admin/settings/notification', notificationSettings), '通知设置保存成功')
    const saveSecuritySettings = () => handleSave(() => api.put('/admin/settings/security', securitySettings), '安全设置保存成功')
    const saveAnnouncementSettings = () => handleSave(() => api.put('/admin/settings/announcement', announcementSettings), '公告设置保存成功')
    const saveBackupSettings = () => handleSave(() => api.put('/admin/settings/backup', backupSettings), '备份设置保存成功')
    const saveThemeSettings = async () => {
      const success = await handleSave(() => api.put('/admin/settings/theme', themeSettings), '主题设置保存成功')
      if (success && themeSettings.default_theme) await themeStore.setTheme(themeSettings.default_theme)
    }
    const saveNodeHealthSettings = () => {
      const data = {
        node_health_check_interval: String(nodeHealthSettings.check_interval),
        node_max_latency: String(nodeHealthSettings.max_latency),
        node_test_timeout: String(nodeHealthSettings.test_timeout),
        test_url: nodeHealthSettings.test_url
      }
      return handleSave(() => api.put('/admin/settings/node_health', data), '节点健康检查设置保存成功')
    }
    const saveAdminNotificationSettings = () => {
      const data = {}
      for (const [key, val] of Object.entries(adminNotificationSettings)) {
        data[key] = typeof val === 'boolean' ? (val ? 'true' : 'false') : (val || '')
      }
      return handleSave(() => adminAPI.updateAdminNotificationSettings(data), '管理员通知设置保存成功')
    }
    const updateGeoIPDatabase = async () => {
      geoipUpdating.value = true
      await handleSave(
        () => api.post('/admin/settings/geoip/update', { type: geoipDatabaseType.value }),
        `${geoipDatabaseType.value === 'dbip' ? 'DB-IP' : 'GeoLite2'} 数据库下载成功`
      )
      await loadGeoIPStatus()
      geoipUpdating.value = false
    }

    const switchDatabase = async (path) => {
      switchingDatabase.value = true
      await handleSave(
        () => api.post('/admin/settings/geoip/switch', { path }),
        '数据库切换成功'
      )
      await loadGeoIPStatus()
      switchingDatabase.value = false
    }

    const flushCache = async () => {
      try {
        await ElMessageBox.confirm('确定要清除所有缓存吗？此操作不可撤销。', '警告', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        cacheClearing.value = true
        const res = await api.post('/admin/settings/cache/flush')
        if (res.data.success) {
          ElMessage.success('缓存已清除')
        } else {
          ElMessage.error(res.data.message || '清除失败')
        }
      } catch (e) {
        if (e !== 'cancel') {
          ElMessage.error('清除失败: ' + (e.response?.data?.message || e.message))
        }
      } finally {
        cacheClearing.value = false
      }
    }

    const testNotification = async (type) => {
      const apiMap = {
        email: { api: adminAPI.testAdminEmailNotification, msg: '邮件测试消息已发送' },
        telegram: { api: adminAPI.testAdminTelegramNotification, msg: 'Telegram 测试消息已发送' },
        bark: { api: adminAPI.testAdminBarkNotification, msg: 'Bark 测试消息已发送' }
      }
      if (!apiMap[type]) return
      testingStates[type] = true
      try {
        const res = await apiMap[type].api()
        if (res.data.success) ElMessage.success(apiMap[type].msg)
        else ElMessage.error(res.data.message || '测试失败')
      } catch (e) {
        ElMessage.error('测试失败: ' + (e.response?.data?.message || e.message))
      } finally {
        testingStates[type] = false
      }
    }
    const testGiteeConnection = async () => {
      const { backup_gitee_token: token, backup_gitee_owner: owner, backup_gitee_repo: repo } = backupSettings
      if (!token || !owner || !repo) return ElMessage.error('请填写完整的 Gitee 配置')
      testingStates.gitee = true
      try {
        await api.put('/admin/settings/backup', backupSettings) // 先保存
        const res = await api.post('/admin/backup/test-gitee', { token, owner, repo })
        if (res.data?.success !== false) ElMessage.success('连接成功！' + (res.data?.data?.message || ''))
        else ElMessage.error(res.data?.message || '测试失败')
      } catch (e) {
        ElMessage.error('测试失败: ' + (e.response?.data?.message || e.message))
      } finally {
        testingStates.gitee = false
      }
    }
    const testGitHubConnection = async () => {
      const { backup_github_token: token, backup_github_owner: owner, backup_github_repo: repo } = backupSettings
      if (!token || !owner || !repo) return ElMessage.error('请填写完整的 GitHub 配置')
      testingStates.github = true
      try {
        await api.put('/admin/settings/backup', backupSettings) // 先保存
        const res = await api.post('/admin/backup/test-github', { token, owner, repo })
        if (res.data?.success !== false) ElMessage.success('连接成功！' + (res.data?.data?.message || ''))
        else ElMessage.error(res.data?.message || '测试失败')
      } catch (e) {
        ElMessage.error('测试失败: ' + (e.response?.data?.message || e.message))
      } finally {
        testingStates.github = false
      }
    }
    const checkUploadStatus = async (taskId, target) => {
      try {
        const res = await api.get(`/admin/backup/upload-status/${taskId}?target=${target || 'gitee'}`)
        const status = res.data?.data || res.data
        if (status) {
          uploadStatus.value = status
          if (status.status === 'success' || status.status === 'failed') {
            if (uploadStatusInterval.value) {
              clearInterval(uploadStatusInterval.value)
              uploadStatusInterval.value = null
            }
            if (status.status === 'success') {
              ElMessage.success(status.message || '上传成功')
            } else {
              ElMessage.error(status.message || status.error || '上传失败')
            }
          }
        }
      } catch (e) {
        console.error('查询上传状态失败:', e)
      }
    }
    const startStatusPolling = (taskId, target) => {
      if (uploadStatusInterval.value) {
        clearInterval(uploadStatusInterval.value)
      }
      checkUploadStatus(taskId, target)
      uploadStatusInterval.value = setInterval(() => {
        checkUploadStatus(taskId, target)
      }, 2000)
    }
    const stopStatusPolling = () => {
      if (uploadStatusInterval.value) {
        clearInterval(uploadStatusInterval.value)
        uploadStatusInterval.value = null
      }
      uploadStatus.value = null
      uploadTaskId.value = null
    }
    const createManualBackup = async () => {
      creatingBackup.value = true
      uploadStatus.value = null
      uploadTaskId.value = null
      try {
        // 先保存备份设置，确保后端使用最新的备份目标
        await api.put('/admin/settings/backup', backupSettings)
        const res = await api.post('/admin/backup', {}, { timeout: 60000 })
        if (res.data?.success !== false) {
          const d = res.data.data || res.data
          let msg = '备份文件创建成功！'
          if (d.filename) msg += ` 文件: ${d.filename}`
          if (d.size) msg += ` (${(d.size/1024/1024).toFixed(2)} MB)`
          const uploadInfo = d.github || d.gitee || {}
          if (uploadInfo.async && uploadInfo.task_id) {
            uploadTaskId.value = uploadInfo.task_id
            uploadTarget.value = uploadInfo.target || (d.github ? 'github' : 'gitee')
            uploadStatus.value = {
              status: 'uploading',
              progress: 0,
              message: '正在准备上传...',
              start_time: new Date().toISOString(),
              file_name: d.filename || '',
              file_size: d.size || 0
            }
            const platformName = uploadTarget.value === 'github' ? 'GitHub' : 'Gitee'
            msg += ' | ' + (uploadInfo.message || `正在后台上传到${platformName}...`)
          ElMessage.success(msg)
            startStatusPolling(uploadInfo.task_id, uploadTarget.value)
          } else if (d.github?.uploaded || d.gitee?.uploaded) {
            const platformName = d.github ? 'GitHub' : 'Gitee'
            msg += ` | 已上传 ${platformName}`
            ElMessage.success(msg)
          } else if (d.github?.error || d.gitee?.error) {
            const platformName = d.github ? 'GitHub' : 'Gitee'
            const error = d.github?.error || d.gitee?.error
            msg += ` | ${platformName} 失败: ${error}`
            ElMessage.warning(msg)
          } else {
            ElMessage.success(msg)
          }
        } else {
          ElMessage.error(res.data?.message || '备份失败')
        }
      } catch (e) {
        if (e.message?.includes('timeout')) {
          ElMessage.warning('请求超时，备份可能仍在后台进行')
        } else {
          ElMessage.error('备份失败: ' + (e.response?.data?.message || e.message))
        }
      } finally {
        creatingBackup.value = false
      }
    }
    const handleLogoSuccess = (res) => {
      const url = res?.data?.url || res?.url
      if (res?.success || url) {
        generalSettings.site_logo = url || ''
        ElMessage.success('Logo上传成功')
      } else ElMessage.error('Logo上传失败')
    }
    const beforeLogoUpload = (file) => {
      if (!file.type.startsWith('image/')) return ElMessage.error('只能上传图片!') && false
      if (file.size / 1024 / 1024 >= 2) return ElMessage.error('大小不能超过 2MB!') && false
      return true
    }
    const handleResize = () => isMobile.value = window.innerWidth <= 768
    onMounted(() => {
      loadSettings()
      loadGeoIPStatus()
      window.addEventListener('resize', handleResize)
    })
    onBeforeUnmount(() => {
      window.removeEventListener('resize', handleResize)
      stopStatusPolling()
    })
    return {
      activeTab, isMobile, formLayout,
      generalSettings, generalRules, generalFormRef,
      registrationSettings, notificationSettings, securitySettings,
      themeSettings, adminNotificationSettings, announcementSettings,
      nodeHealthSettings, backupSettings,
      uploadUrl, themeOptions: THEME_OPTIONS,
      testingStates, geoipStatus, geoipUpdating, geoipDatabaseType, switchingDatabase, creatingBackup, cacheClearing,
      uploadStatus, uploadTaskId, stopStatusPolling,
      saveGeneralSettings, saveRegistrationSettings, saveNotificationSettings,
      saveSecuritySettings, saveThemeSettings, saveAnnouncementSettings,
      saveNodeHealthSettings, saveAdminNotificationSettings, saveBackupSettings,
      testNotification, testGiteeConnection, testGitHubConnection, createManualBackup,
      updateGeoIPDatabase, switchDatabase, flushCache, handleLogoSuccess, beforeLogoUpload, formatFileSize
    }
  }
}
</script>
<style scoped>
.admin-settings { padding: 20px; }
.avatar-uploader { text-align: center; }
.avatar-uploader .avatar { width: 100px; height: 100px; display: block; object-fit: cover; }
.avatar-uploader .el-upload {
  border: 1px dashed #d9d9d9;
  border-radius: 6px;
  cursor: pointer;
  position: relative;
  overflow: clip;
  transition: var(--el-transition-duration-fast);
}
.avatar-uploader .el-upload:hover { border-color: var(--el-color-primary); }
.avatar-uploader-icon { font-size: 28px; color: #8c939d; width: 100px; height: 100px; text-align: center; line-height: 100px; }
.form-tip { font-size: 12px; color: #909399; margin-top: 5px; line-height: 1.5; }
.mb-20 { margin-bottom: 20px; }
.short-input { width: 200px; }
.theme-color-block { display: inline-block; width: 12px; height: 12px; border-radius: 2px; margin-right: 8px; }
.theme-color-block-sm { display: inline-block; width: 14px; height: 14px; border-radius: 2px; }
.flex-align-center { display: inline-flex; align-items: center; gap: 6px; }
.theme-checkbox-group { display: flex; flex-wrap: wrap; gap: 16px; }
.theme-checkbox-group :deep(.el-checkbox) { min-width: 120px; margin-right: 0; }
.notification-events-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 10px; }

/* 通知设置容器样式 - 左右布局 */
.notification-container {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.notification-section {
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  height: fit-content;
}

.notification-section :deep(.el-card__header) {
  background-color: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
  padding: 16px 20px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

:deep(.el-input__wrapper), :deep(.el-textarea__inner) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
}
:deep(.el-input__inner) {
  border: none !important;
  box-shadow: none !important;
  background: transparent !important;
  height: 100%;
}
:deep(.el-input__wrapper:hover), :deep(.el-input__wrapper.is-focus),
:deep(.el-textarea__inner:hover), :deep(.el-textarea__inner:focus) {
  border-color: var(--el-color-primary) !important;
}
.settings-form :deep(.el-form-item) { margin-bottom: 18px; }
.settings-form :deep(.el-form-item__label) { font-weight: 500; }

@media (max-width: 768px) {
  .admin-settings { padding: 10px; }
  .short-input { width: 100%; }
  .settings-form :deep(.el-form-item) { margin-bottom: 20px; }
  .settings-form :deep(.el-form-item__label) { margin-bottom: 8px; }
  .full-width { width: 100%; }
  .theme-checkbox-group.mobile { flex-direction: column; gap: 12px; }
  .theme-checkbox-group.mobile :deep(.el-checkbox) { width: 100%; }
  .admin-settings :deep(.el-card) { border-radius: 0; box-shadow: none; border: none; }
  .admin-settings :deep(.el-card__header), .admin-settings :deep(.el-card__body) { padding: 15px; }
  .admin-settings :deep(.el-tabs__nav-wrap) { overflow-x: auto; }
  .admin-settings :deep(.el-divider) { margin: 20px 0; }
  .notification-events-grid { grid-template-columns: 1fr; }

  /* 移动端改为上下布局 */
  .notification-container {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  .section-header { flex-direction: column; align-items: flex-start; gap: 8px; }
}
</style>
