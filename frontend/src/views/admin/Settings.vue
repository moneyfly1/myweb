<template>
  <div class="list-container admin-settings">
    <!-- 顶部 Header -->
    <div class="page-header settings-page-header">
      <div class="header-copy">
        <h1>系统设置</h1>
        <p>集中管理站点、注册、安全、通知、备份和订阅输出规则。</p>
      </div>
      <div class="header-actions">
        <el-button :icon="Refresh" :loading="pageLoading" @click="refreshSettings">刷新</el-button>
        <el-button type="primary" :icon="Check" :loading="savingCurrent" @click="saveCurrentTab">保存当前页</el-button>
      </div>
    </div>

    <!-- 主体设置区域 -->
    <el-card class="settings-shell list-card" shadow="never">
      <el-tabs v-model="activeTab" class="settings-tabs" :tab-position="settingsTabPosition">
        
        <!-- ==================== 基本设置 ==================== -->
        <el-tab-pane label="基本设置" name="general">
          <div class="notification-layout">
            <!-- 左列：站点信息 + 客服与界面 -->
            <div class="notification-panel">
              <div class="panel-header">
                <h3>站点信息与客服</h3>
              </div>

              <el-form :model="generalSettings" :rules="generalRules" ref="generalFormRef" label-position="top" class="compact-form">
                <el-form-item label="网站名称" prop="site_name">
                  <el-input v-model="generalSettings.site_name" />
                </el-form-item>
                <el-form-item label="网站描述" prop="site_description">
                  <el-input v-model="generalSettings.site_description" type="textarea" :rows="2" />
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
                  <el-select v-model="generalSettings.default_theme" class="input-full">
                    <el-option label="浅色主题" value="light" />
                    <el-option label="深色主题" value="dark" />
                    <el-option label="跟随系统" value="auto" />
                  </el-select>
                </el-form-item>

                <div class="settings-section-title text-sm">客服与界面</div>
                <el-form-item label="售后QQ" prop="support_qq">
                  <el-input v-model="generalSettings.support_qq" placeholder="请输入售后QQ号码" />
                  <div class="form-tip">帮助中心显示，留空不显示。</div>
                </el-form-item>
                <el-form-item label="售后邮箱" prop="support_email">
                  <el-input v-model="generalSettings.support_email" placeholder="例如: support@example.com" />
                  <div class="form-tip">帮助中心显示，留空不显示。</div>
                </el-form-item>
                <div class="switch-card mb-3">
                  <div class="switch-card-content">
                    <span class="title">启用统一登录页</span>
                    <span class="desc">启用后登录、注册、找回密码使用统一入口。</span>
                  </div>
                  <el-switch v-model="generalSettings.unified_auth_enabled" />
                </div>

                <div class="mt-4">
                  <el-button type="primary" @click="saveGeneralSettings" :class="{ 'full-width': isMobile }">保存基本设置</el-button>
                </div>
              </el-form>
            </div>

            <!-- 右列：GeoIP 与缓存管理 -->
            <div class="notification-panel">
              <div class="panel-header">
                <h3>GeoIP 与缓存管理</h3>
              </div>

              <el-form label-position="top" class="compact-form">
                <el-form-item label="GeoIP 状态">
                  <div class="status-box" v-if="geoipStatus">
                    <el-tag :type="geoipStatus.enabled ? 'success' : 'warning'">
                      {{ geoipStatus.enabled ? '已启用' : '未启用' }}
                    </el-tag>
                    <span class="status-text" :class="geoipStatus.active_database ? 'text-success' : 'text-danger'">
                      {{ geoipStatus.active_database ? `当前: ${geoipStatus.active_database}` : '未找到数据库文件' }}
                    </span>
                  </div>
                </el-form-item>
                <el-form-item label="已安装数据库" v-if="geoipStatus && geoipStatus.databases && geoipStatus.databases.length > 0">
                  <el-table :data="geoipStatus.databases" border size="small" style="width: 100%;">
                    <el-table-column prop="name" label="名称" min-width="120">
                      <template #default="scope">
                        <span>{{ scope.row.name }}</span>
                        <el-tag v-if="scope.row.active" type="success" size="small" style="margin-left: 6px;">使用中</el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column prop="size" label="大小" width="70" />
                    <el-table-column prop="modified" label="更新" width="110" />
                    <el-table-column label="操作" width="60" align="center">
                      <template #default="scope">
                        <el-button v-if="!scope.row.active" type="primary" link size="small" @click="switchDatabase(scope.row.path)" :loading="switchingDatabase">切换</el-button>
                      </template>
                    </el-table-column>
                  </el-table>
                </el-form-item>
                <el-form-item label="更新数据库">
                  <el-radio-group v-model="geoipDatabaseType" class="radio-block-group radio-block-vertical">
                    <el-radio label="dbip">
                      <div class="radio-content">
                        <div class="radio-title">DB-IP City Lite <el-tag size="small" type="success">推荐</el-tag></div>
                        <div class="radio-desc">中国数据详细，完全免费，约 125MB</div>
                      </div>
                    </el-radio>
                    <el-radio label="geolite2">
                      <div class="radio-content">
                        <div class="radio-title">GeoLite2 City (MaxMind)</div>
                        <div class="radio-desc">广泛使用，部分中国数据欠佳，约 60MB</div>
                      </div>
                    </el-radio>
                  </el-radio-group>
                  <div style="margin-top: 12px;">
                    <el-button type="primary" plain @click="updateGeoIPDatabase" :loading="geoipUpdating" :class="{ 'full-width': isMobile }">
                      {{ geoipUpdating ? '下载中...' : '下载/更新数据库' }}
                    </el-button>
                    <div class="form-tip mt-2">建议每月更新一次以获取最新数据。</div>
                  </div>
                </el-form-item>

                <div class="settings-section-title text-sm">缓存管理</div>
                <el-form-item label="Redis 缓存">
                  <el-button type="danger" plain @click="flushCache" :loading="cacheClearing" :class="{ 'full-width': isMobile }">
                    {{ cacheClearing ? '清除中...' : '清除所有缓存' }}
                  </el-button>
                  <div class="form-tip mt-2">清除后系统会自动重新缓存。</div>
                </el-form-item>
              </el-form>
            </div>
          </div>
        </el-tab-pane>
        
        <!-- ==================== 注册设置 ==================== -->
        <el-tab-pane label="注册设置" name="registration">
          <div class="notification-layout">
            <div class="notification-panel">
              <div class="panel-header">
                <h3>注册策略</h3>
              </div>
              <el-form :model="registrationSettings" label-position="top" class="compact-form">
                <div class="switch-group">
                  <div class="switch-card">
                    <span class="title">开放注册</span>
                    <el-switch v-model="registrationSettings.registration_enabled" />
                  </div>
                  <div class="switch-card">
                    <span class="title">邮箱验证</span>
                    <el-switch v-model="registrationSettings.email_verification_required" />
                  </div>
                  <div class="switch-card">
                    <span class="title">邀请码注册</span>
                    <el-switch v-model="registrationSettings.invite_code_required" />
                  </div>
                </div>
                <el-form-item label="最小密码长度" class="mt-3">
                  <el-input-number v-model="registrationSettings.min_password_length" :min="4" :max="32" size="small" />
                </el-form-item>
                <div class="mt-3">
                  <el-button type="primary" @click="saveRegistrationSettings" :class="{ 'full-width': isMobile }">保存注册设置</el-button>
                </div>
              </el-form>
            </div>

            <div class="notification-panel">
              <div class="panel-header">
                <h3>新用户默认订阅</h3>
              </div>
              <el-form :model="registrationSettings" label-position="top" class="compact-form">
                <div class="backup-fields-row">
                  <el-form-item label="默认设备数">
                    <el-input-number v-model="registrationSettings.default_subscription_device_limit" :min="0" size="small" class="input-full" />
                    <div class="form-tip">0 表示无限制</div>
                  </el-form-item>
                  <el-form-item label="默认时长(月)">
                    <el-input-number v-model="registrationSettings.default_subscription_duration_months" :min="0" size="small" class="input-full" />
                    <div class="form-tip">新用户默认订阅有效期</div>
                  </el-form-item>
                </div>
              </el-form>
            </div>
          </div>
        </el-tab-pane>

        <!-- ==================== 邀请设置 ==================== -->
        <el-tab-pane label="邀请设置" name="invite">
          <div class="single-panel-wrapper">
            <div class="notification-panel">
              <div class="panel-header">
                <h3>邀请奖励配置</h3>
                <el-tag type="info" size="small" effect="plain">影响新邀请码</el-tag>
              </div>
              <el-form :model="inviteSettings" label-position="top" class="compact-form">
                <div class="inline-fields-row">
                  <el-form-item label="邀请人奖励(元)">
                    <el-input-number v-model="inviteSettings.inviter_reward" :min="0" :max="10000" :precision="2" :step="1" size="small" class="input-full" />
                  </el-form-item>
                  <el-form-item label="被邀请人奖励(元)">
                    <el-input-number v-model="inviteSettings.invitee_reward" :min="0" :max="10000" :precision="2" :step="1" size="small" class="input-full" />
                  </el-form-item>
                  <el-form-item label="触发订单金额(元)">
                    <el-input-number v-model="inviteSettings.min_order_amount" :min="0" :max="100000" :precision="2" :step="1" size="small" class="input-full" />
                  </el-form-item>
                </div>
                <div class="form-tip mb-3">邀请人奖励在被邀请用户满足订单金额后发放，0 表示不限金额。</div>

                <div class="switch-card mb-3">
                  <div class="switch-card-content">
                    <span class="title">仅限新用户</span>
                    <span class="desc">只有首次注册的新用户能使用邀请码奖励。</span>
                  </div>
                  <el-switch v-model="inviteSettings.new_user_only" />
                </div>

                <div class="mt-3">
                  <el-button type="primary" @click="saveInviteSettings" :class="{ 'full-width': isMobile }">保存邀请设置</el-button>
                </div>
              </el-form>
            </div>
          </div>
        </el-tab-pane>

        <!-- ==================== 通知设置 ==================== -->
        <el-tab-pane label="通知设置" name="notification">
          <div class="notification-layout">
            
            <!-- 用户通知模块 -->
            <div class="notification-panel">
              <div class="panel-header">
                <h3><el-icon><Message /></el-icon> 客户通知配置</h3>
                <el-tag type="info" size="small" effect="plain">渠道：邮件</el-tag>
              </div>
              
              <el-form :model="notificationSettings" label-position="top" class="compact-form">
                <div class="switch-card master-switch mb-3">
                  <div class="switch-card-content">
                    <span class="title">开启邮件通知</span>
                    <span class="desc">主开关，关闭后用户端将不会收到任何邮件通知。</span>
                  </div>
                  <el-switch v-model="notificationSettings.email_notifications" />
                </div>

                <div class="settings-section-title text-sm">客户事件通知矩阵</div>
                <el-table :data="customerEventSwitches" border stripe size="small" class="matrix-table">
                  <el-table-column prop="label" label="触发事件" min-width="140" />
                  <el-table-column label="邮件" width="80" align="center">
                    <template #default="{ row }">
                      <el-switch v-model="notificationSettings[row.channels.email]" size="small" />
                    </template>
                  </el-table-column>
                </el-table>

                <div class="settings-section-title text-sm">到期提醒规则</div>
                <div class="switch-card mb-3">
                  <div class="switch-card-content">
                    <span class="title">订阅到期自动提醒</span>
                    <span class="desc">开启后系统会在订阅到期前自动发送邮件提醒。</span>
                  </div>
                  <el-switch v-model="notificationSettings.subscription_expiry_notifications" />
                </div>
                <el-collapse-transition>
                  <div v-show="notificationSettings.subscription_expiry_notifications" class="expiry-config-row mb-3">
                    <el-form-item label="冷却时间(小时)">
                      <el-input-number v-model="notificationSettings.subscription_expiry_reminder_cooldown_hours" :min="0" :max="720" size="small" class="input-full" />
                      <div class="form-tip">同主题最小发送间隔，0不限制</div>
                    </el-form-item>
                    <el-form-item label="每日上限(封)">
                      <el-input-number v-model="notificationSettings.subscription_expiry_reminder_daily_limit" :min="0" :max="20" size="small" class="input-full" />
                      <div class="form-tip">每天最多接收的到期提醒数</div>
                    </el-form-item>
                  </div>
                </el-collapse-transition>

                <div class="mt-3">
                  <el-button type="primary" @click="saveNotificationSettings" :class="{ 'full-width': isMobile }">保存客户通知设置</el-button>
                </div>
              </el-form>
            </div>

            <!-- 管理员通知模块 -->
            <div class="notification-panel">
              <div class="panel-header warning-header">
                <h3><el-icon><Bell /></el-icon> 管理员告警配置</h3>
                <div class="header-tags">
                  <el-tag type="warning" size="small" effect="plain">邮件</el-tag>
                  <el-tag type="primary" size="small" effect="plain">Telegram</el-tag>
                  <el-tag type="danger" size="small" effect="plain">Bark</el-tag>
                </div>
              </div>

              <el-form :model="adminNotificationSettings" label-position="top" class="compact-form">
                <div class="switch-card master-switch mb-4" style="border-color: var(--el-color-warning-light-5);">
                  <div class="switch-card-content">
                    <span class="title">开启管理员告警</span>
                    <span class="desc">控制所有管理员通知的总开关，渠道配置保留不受影响。</span>
                  </div>
                  <el-switch v-model="adminNotificationSettings.admin_notification_enabled" />
                </div>

                <div class="settings-section-title text-sm">渠道配置与测试</div>
                <div class="admin-channels-list mb-4">
                  <!-- 邮件渠道 -->
                  <div class="channel-config-card" :class="{ 'is-disabled': !adminNotificationSettings.admin_email_notification }">
                    <div class="channel-header">
                      <div class="channel-title">邮件接收</div>
                      <el-switch v-model="adminNotificationSettings.admin_email_notification" />
                    </div>
                    <el-collapse-transition>
                      <div v-show="adminNotificationSettings.admin_email_notification" class="channel-body">
                        <el-input v-model="adminNotificationSettings.admin_notification_email" placeholder="接收邮箱地址, 例如 admin@example.com" size="small" />
                        <el-button size="small" plain @click="testNotification('email')" :loading="testingStates.email" class="mt-2 full-width">发送测试邮件</el-button>
                      </div>
                    </el-collapse-transition>
                  </div>

                  <!-- Telegram渠道 -->
                  <div class="channel-config-card" :class="{ 'is-disabled': !adminNotificationSettings.admin_telegram_notification }">
                    <div class="channel-header">
                      <div class="channel-title">Telegram Bot</div>
                      <el-switch v-model="adminNotificationSettings.admin_telegram_notification" />
                    </div>
                    <el-collapse-transition>
                      <div v-show="adminNotificationSettings.admin_telegram_notification" class="channel-body">
                        <el-input v-model="adminNotificationSettings.admin_telegram_bot_token" type="password" show-password placeholder="Bot Token" size="small" class="mb-2" />
                        <el-input v-model="adminNotificationSettings.admin_telegram_chat_id" type="password" show-password placeholder="Chat ID" size="small" />
                        <el-button size="small" plain @click="testNotification('telegram')" :loading="testingStates.telegram" class="mt-2 full-width">测试 Telegram</el-button>
                      </div>
                    </el-collapse-transition>
                  </div>

                  <!-- Bark渠道 -->
                  <div class="channel-config-card" :class="{ 'is-disabled': !adminNotificationSettings.admin_bark_notification }">
                    <div class="channel-header">
                      <div class="channel-title">Bark 推送</div>
                      <el-switch v-model="adminNotificationSettings.admin_bark_notification" />
                    </div>
                    <el-collapse-transition>
                      <div v-show="adminNotificationSettings.admin_bark_notification" class="channel-body">
                        <el-input v-model="adminNotificationSettings.admin_bark_server_url" placeholder="服务器URL (默认 https://api.day.app)" size="small" class="mb-2" />
                        <el-input v-model="adminNotificationSettings.admin_bark_device_key" type="password" show-password placeholder="Device Key" size="small" />
                        <el-button size="small" plain @click="testNotification('bark')" :loading="testingStates.bark" class="mt-2 full-width">测试 Bark</el-button>
                      </div>
                    </el-collapse-transition>
                  </div>
                </div>

                <div class="settings-section-title text-sm">系统安全告警</div>
                <div class="switch-card mb-4">
                  <div class="switch-card-content">
                    <span class="title">异常登录告警</span>
                    <span class="desc">管理员在新设备或异地登录时接收紧急通知</span>
                  </div>
                  <el-switch v-model="adminNotificationSettings.admin_abnormal_login_alert_enabled" />
                </div>

                <div class="settings-section-title text-sm">管理员事件矩阵</div>
                <!-- 优雅的多渠道矩阵 -->
                <el-table :data="adminNotificationEvents" border stripe size="small" class="matrix-table">
                  <el-table-column prop="label" label="触发事件" min-width="120" />
                  <el-table-column label="邮件" width="65" align="center">
                    <template #default="{ row }"><el-switch v-model="adminNotificationSettings[row.channels.email]" size="small" /></template>
                  </el-table-column>
                  <el-table-column label="TG" width="65" align="center">
                    <template #default="{ row }"><el-switch v-model="adminNotificationSettings[row.channels.telegram]" size="small" /></template>
                  </el-table-column>
                  <el-table-column label="Bark" width="65" align="center">
                    <template #default="{ row }"><el-switch v-model="adminNotificationSettings[row.channels.bark]" size="small" /></template>
                  </el-table-column>
                </el-table>

                <div class="mt-4">
                  <el-button type="primary" @click="saveAdminNotificationSettings" :class="{ 'full-width': isMobile }">保存告警设置</el-button>
                </div>
              </el-form>
            </div>
          </div>
        </el-tab-pane>

        <!-- ==================== 公告管理 ==================== -->
        <el-tab-pane label="公告管理" name="announcement">
          <div class="single-panel-wrapper">
            <div class="notification-panel">
              <div class="panel-header">
                <h3>登录公告</h3>
                <el-switch v-model="announcementSettings.announcement_enabled" active-text="启用" />
              </div>

              <el-form :model="announcementSettings" label-position="top" class="compact-form">
                <el-form-item label="公告内容 (支持 HTML)">
                  <el-input v-model="announcementSettings.announcement_content" type="textarea" :rows="10" placeholder="支持完整的 HTML 格式排版" />
                  <div class="form-tip">用户登录后台时将会弹出该公告提示。</div>
                </el-form-item>

                <div class="mt-3">
                  <el-button type="primary" @click="saveAnnouncementSettings" :class="{ 'full-width': isMobile }">保存公告</el-button>
                </div>
              </el-form>
            </div>
          </div>
        </el-tab-pane>

        <!-- ==================== 主题设置 ==================== -->
        <el-tab-pane label="主题设置" name="theme">
          <div class="single-panel-wrapper">
            <div class="notification-panel">
              <div class="panel-header">
                <h3>主题配置</h3>
              </div>

              <el-form :model="themeSettings" label-position="top" class="compact-form">
                <div class="theme-top-row">
                  <el-form-item label="全局默认主题" prop="default_theme">
                    <el-select v-model="themeSettings.default_theme" class="input-full">
                      <el-option v-for="theme in themeOptions" :key="theme.value" :label="theme.label" :value="theme.value">
                        <span :style="{ backgroundColor: theme.color }" class="theme-color-block"></span>
                        {{ theme.label }}
                      </el-option>
                    </el-select>
                  </el-form-item>
                  <div class="switch-card">
                    <span class="title">允许用户自定义主题</span>
                    <el-switch v-model="themeSettings.allow_user_theme" />
                  </div>
                </div>

                <div class="settings-section-title text-sm">开放给用户的主题</div>
                <el-checkbox-group v-model="themeSettings.available_themes" class="theme-checkbox-group">
                  <el-checkbox v-for="theme in themeOptions" :key="theme.value" :label="theme.value" border>
                    <span class="flex-align-center">
                      <span :style="{ backgroundColor: theme.color }" class="theme-color-block-sm"></span>
                      {{ theme.label }}
                    </span>
                  </el-checkbox>
                </el-checkbox-group>

                <div class="mt-3">
                  <el-button type="primary" @click="saveThemeSettings" :class="{ 'full-width': isMobile }">保存主题设置</el-button>
                </div>
              </el-form>
            </div>
          </div>
        </el-tab-pane>

        <!-- ==================== 节点健康检查 ==================== -->
        <el-tab-pane label="节点监控" name="node-health">
          <div class="single-panel-wrapper">
            <div class="notification-panel">
              <div class="panel-header">
                <h3>节点健康检测</h3>
                <el-tag type="info" size="small" effect="plain">TCP 握手</el-tag>
              </div>
              <el-form :model="nodeHealthSettings" label-position="top" class="compact-form">
                <div class="inline-fields-row">
                  <el-form-item label="检查频率(分钟)">
                    <el-input-number v-model="nodeHealthSettings.check_interval" :min="1" size="small" class="input-full" />
                    <div class="form-tip">建议 30-60 分钟</div>
                  </el-form-item>
                  <el-form-item label="最大延迟(ms)">
                    <el-input-number v-model="nodeHealthSettings.max_latency" :min="100" :step="100" size="small" class="input-full" />
                    <div class="form-tip">建议 3000ms</div>
                  </el-form-item>
                  <el-form-item label="测试超时(秒)">
                    <el-input-number v-model="nodeHealthSettings.test_timeout" :min="1" size="small" class="input-full" />
                    <div class="form-tip">建议 5 秒</div>
                  </el-form-item>
                </div>
                <el-form-item label="辅助测速 URL" class="mt-3">
                  <el-input v-model="nodeHealthSettings.test_url" placeholder="例如: https://ping.pe" />
                  <div class="form-tip">用于 HTTP 延迟测试，不填则仅测 TCP 端口。</div>
                </el-form-item>
                <div class="mt-3">
                  <el-button type="primary" @click="saveNodeHealthSettings" :class="{ 'full-width': isMobile }">保存监控配置</el-button>
                </div>
              </el-form>
            </div>
          </div>
        </el-tab-pane>

        <!-- ==================== 安全设置 ==================== -->
        <el-tab-pane label="安全设置" name="security">
          <div class="notification-layout">
            <div class="notification-panel">
              <div class="panel-header">
                <h3>会话与防爆破</h3>
              </div>
              <el-form :model="securitySettings" label-position="top" class="compact-form">
                <div class="inline-fields-row">
                  <el-form-item label="登录失败上限(次)">
                    <el-input-number v-model="securitySettings.login_fail_limit" :min="1" size="small" class="input-full" />
                  </el-form-item>
                  <el-form-item label="锁定时间(分钟)">
                    <el-input-number v-model="securitySettings.login_lock_time" :min="1" size="small" class="input-full" />
                  </el-form-item>
                  <el-form-item label="会话超时(分钟)">
                    <el-input-number v-model="securitySettings.session_timeout" :min="10" size="small" class="input-full" />
                  </el-form-item>
                </div>
                <div class="mt-3">
                  <el-button type="primary" @click="saveSecuritySettings" :class="{ 'full-width': isMobile }">保存安全设置</el-button>
                </div>
              </el-form>
            </div>

            <div class="notification-panel">
              <div class="panel-header">
                <h3>访问控制与告警</h3>
              </div>
              <el-form :model="securitySettings" label-position="top" class="compact-form">
                <div class="switch-card mb-3">
                  <div class="switch-card-content">
                    <span class="title">全局异常登录告警</span>
                    <span class="desc">用户在陌生IP登录时收到风险提示。</span>
                  </div>
                  <el-switch v-model="securitySettings.abnormal_login_alert_enabled" />
                </div>
              </el-form>
            </div>
          </div>
        </el-tab-pane>

        <!-- ==================== 备份设置 ==================== -->
        <el-tab-pane label="数据备份" name="backup">
          <div class="notification-layout">
            <!-- 左列：平台配置 -->
            <div class="notification-panel">
              <div class="panel-header">
                <h3>备份平台配置</h3>
              </div>

              <el-form :model="backupSettings" label-position="top" class="compact-form">
                <el-radio-group v-model="backupSettings.backup_target" class="radio-block-group mb-3" style="width: 100%;">
                  <el-radio label="gitee">
                    <div class="radio-content">
                      <div class="radio-title">Gitee (码云)</div>
                      <div class="radio-desc">国内速度快，稳定性好</div>
                    </div>
                  </el-radio>
                  <el-radio label="github">
                    <div class="radio-content">
                      <div class="radio-title">GitHub</div>
                      <div class="radio-desc">全球最大，适合海外机器</div>
                    </div>
                  </el-radio>
                </el-radio-group>

                <!-- Gitee 配置 -->
                <div class="platform-config-block" :class="{ 'is-active': backupSettings.backup_target === 'gitee' }">
                  <div class="block-header">
                    <span class="block-title">Gitee 接口配置</span>
                    <el-switch v-model="backupSettings.backup_gitee_enabled" active-text="启用" />
                  </div>
                  <el-collapse-transition>
                    <div v-show="backupSettings.backup_gitee_enabled" class="block-body">
                      <el-form-item label="Access Token">
                        <el-input v-model="backupSettings.backup_gitee_token" type="password" show-password placeholder="需具备 projects 权限" />
                      </el-form-item>
                      <div class="backup-fields-row">
                        <el-form-item label="仓库所有者">
                          <el-input v-model="backupSettings.backup_gitee_owner" placeholder="moneyfly" />
                        </el-form-item>
                        <el-form-item label="仓库名称">
                          <el-input v-model="backupSettings.backup_gitee_repo" placeholder="backup" />
                        </el-form-item>
                      </div>
                      <el-button type="success" plain @click="testGiteeConnection" :loading="testingStates.gitee" :disabled="!backupSettings.backup_gitee_token" size="small">测试连通性</el-button>
                    </div>
                  </el-collapse-transition>
                </div>

                <!-- GitHub 配置 -->
                <div class="platform-config-block mt-3" :class="{ 'is-active': backupSettings.backup_target === 'github' }">
                  <div class="block-header">
                    <span class="block-title">GitHub 接口配置</span>
                    <el-switch v-model="backupSettings.backup_github_enabled" active-text="启用" />
                  </div>
                  <el-collapse-transition>
                    <div v-show="backupSettings.backup_github_enabled" class="block-body">
                      <el-form-item label="Personal Token">
                        <el-input v-model="backupSettings.backup_github_token" type="password" show-password placeholder="需具备 repo 权限" />
                      </el-form-item>
                      <div class="backup-fields-row">
                        <el-form-item label="仓库所有者">
                          <el-input v-model="backupSettings.backup_github_owner" placeholder="moneyfly1" />
                        </el-form-item>
                        <el-form-item label="仓库名称">
                          <el-input v-model="backupSettings.backup_github_repo" placeholder="backup" />
                        </el-form-item>
                      </div>
                      <el-button type="success" plain @click="testGitHubConnection" :loading="testingStates.github" :disabled="!backupSettings.backup_github_token" size="small">测试连通性</el-button>
                    </div>
                  </el-collapse-transition>
                </div>
              </el-form>
            </div>

            <!-- 右列：自动化与执行 -->
            <div class="notification-panel">
              <div class="panel-header">
                <h3>自动化与执行</h3>
              </div>

              <el-form :model="backupSettings" label-position="top" class="compact-form">
                <div class="switch-card mb-3">
                  <div class="switch-card-content">
                    <span class="title">开启自动定时备份</span>
                    <span class="desc">按照设定间隔自动打包上传。</span>
                  </div>
                  <el-switch v-model="backupSettings.backup_auto_enabled" />
                </div>

                <el-collapse-transition>
                  <el-form-item label="执行间隔" v-show="backupSettings.backup_auto_enabled">
                    <div style="display: flex; align-items: center; gap: 8px;">
                      <el-input-number v-model="backupSettings.backup_auto_interval" :min="1" size="small" style="width: 120px;" />
                      <span class="unit-text">小时</span>
                    </div>
                    <div class="form-tip">推荐 12 或 24 小时。</div>
                  </el-form-item>
                </el-collapse-transition>

                <div class="settings-section-title text-sm">手动备份</div>
                <el-button type="primary" plain @click="createManualBackup" :loading="creatingBackup" :class="{ 'full-width': isMobile }">立即打包并上传</el-button>

                <div v-if="uploadStatus || uploadTaskId" class="mt-3">
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
                          :stroke-width="6"
                        />
                        <div style="margin-top: 4px; font-size: 12px; color: #909399;">
                          {{ uploadStatus?.message || '正在准备上传...' }}
                          <span v-if="uploadStatus?.file_size"> | {{ (uploadStatus.file_size / 1024 / 1024).toFixed(2) }} MB</span>
                        </div>
                      </div>
                      <div v-else>
                        <div>{{ uploadStatus.message }}</div>
                        <div v-if="uploadStatus.error" style="margin-top: 4px; font-size: 12px; color: #f56c6c;">{{ uploadStatus.error }}</div>
                      </div>
                    </template>
                  </el-alert>
                </div>

                <div class="mt-3">
                  <el-button type="primary" @click="saveBackupSettings" :class="{ 'full-width': isMobile }">保存所有备份配置</el-button>
                </div>
              </el-form>
            </div>
          </div>
        </el-tab-pane>

        <!-- ==================== 协议过滤 ==================== -->
        <el-tab-pane label="协议过滤" name="protocol-filter">
          <div class="notification-layout">
            <div class="notification-panel">
              <div class="panel-header">
                <h3>Clash / Clash Meta</h3>
                <el-tag type="info" size="small" effect="plain">订阅环境</el-tag>
              </div>
              <el-checkbox-group v-model="protocolFilterSettings.clash_protocols" class="protocol-checkbox-group">
                <el-checkbox v-for="p in allProtocols" :key="'clash-'+p" :label="p" border>{{ p }}</el-checkbox>
              </el-checkbox-group>
            </div>

            <div class="notification-panel">
              <div class="panel-header">
                <h3>通用 Base64</h3>
                <el-tag type="info" size="small" effect="plain">Shadowrocket / V2ray</el-tag>
              </div>
              <el-checkbox-group v-model="protocolFilterSettings.universal_protocols" class="protocol-checkbox-group">
                <el-checkbox v-for="p in allProtocols" :key="'uni-'+p" :label="p" border>{{ p }}</el-checkbox>
              </el-checkbox-group>
            </div>
          </div>
          <div class="mt-3">
            <el-button type="primary" @click="saveProtocolFilterSettings" :class="{ 'full-width': isMobile }">保存协议过滤规则</el-button>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script>
import { ref, reactive, onMounted, onBeforeUnmount, computed } from 'vue'
import { ElMessage, ElMessageBox } from '@/utils/elementPlusServices'
import { Check, Plus, Refresh, Message, Bell } from '@element-plus/icons-vue'
import { useApi, adminAPI } from '@/utils/api'
import { useThemeStore } from '@/store/theme'
import { useMobile } from '@/composables/useMobile'

const ALL_PROTOCOLS = [
  'vmess', 'vless', 'trojan', 'ss', 'ssr', 'hysteria', 'hysteria2',
  'tuic', 'anytls', 'socks', 'socks5', 'http', 'wireguard'
]

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

const customerMethodSwitches = [
  { label: '邮件通知', key: 'email_notifications' }
]
const customerEventSwitches = [
  { label: '新用户注册', base: 'user_registered', legacyKey: 'new_user_notifications', channels: { email: 'user_registered_email_notifications' } },
  { label: '订单支付成功', base: 'order_paid', legacyKey: 'new_order_notifications', channels: { email: 'order_paid_email_notifications' } },
  { label: '充值到账', base: 'recharge_paid', legacyKey: 'recharge_success_notifications', channels: { email: 'recharge_paid_email_notifications' } },
  { label: '订阅创建', base: 'subscription_created', legacyKey: 'subscription_created_notifications', channels: { email: 'subscription_created_email_notifications' } },
  { label: '订阅发送', base: 'subscription_sent', legacyKey: 'subscription_sent_notifications', channels: { email: 'subscription_sent_email_notifications' } },
  { label: '订阅重置', base: 'subscription_reset', legacyKey: 'subscription_reset_notifications', channels: { email: 'subscription_reset_email_notifications' } },
  { label: '订阅到期', base: 'subscription_expiry', legacyKey: 'subscription_expiry_notifications', channels: { email: 'subscription_expiry_email_notifications' } },
  { label: '工单回复', base: 'ticket_reply', legacyKey: 'ticket_reply_notifications', channels: { email: 'ticket_reply_email_notifications' } },
  { label: '密码修改', base: 'password_changed', legacyKey: 'password_changed_notifications', channels: { email: 'password_changed_email_notifications' } },
  { label: '密码重置', base: 'password_reset', legacyKey: 'password_reset_notifications', channels: { email: 'password_reset_email_notifications' } },
  { label: '异常登录', base: 'abnormal_login', legacyKey: 'abnormal_login_notifications', channels: { email: 'abnormal_login_email_notifications' } }
]

const adminNotificationEvents = [
  { label: '新用户注册', key: 'admin_notify_user_registered' },
  { label: '管理员创建用户', key: 'admin_notify_user_created' },
  { label: '密码重置', key: 'admin_notify_password_reset' },
  { label: '密码修改', key: 'admin_notify_password_changed' },
  { label: '订单支付成功', key: 'admin_notify_order_paid' },
  { label: '充值到账', key: 'admin_notify_recharge_paid' },
  { label: '订阅创建', key: 'admin_notify_subscription_created' },
  { label: '发送订阅', key: 'admin_notify_subscription_sent' },
  { label: '重置订阅', key: 'admin_notify_subscription_reset' },
  { label: '订阅到期', key: 'admin_notify_subscription_expired' },
  { label: '用户提交工单', key: 'admin_notify_ticket_created' },
  { label: '工单新回复', key: 'admin_notify_ticket_replied' },
  { label: '异常登录', key: 'admin_notify_abnormal_login' }
].map(item => ({
  ...item,
  channels: { email: `${item.key}_email`, telegram: `${item.key}_telegram`, bark: `${item.key}_bark` }
}))

const adminLegacyEventKeys = adminNotificationEvents.map(item => item.key)
const adminEventChannelKeys = adminNotificationEvents.flatMap(item => Object.values(item.channels))
const customerEventLegacyKeys = customerEventSwitches.map(item => item.legacyKey)
const customerEventChannelKeys = customerEventSwitches.flatMap(item => Object.values(item.channels))

const customerNotificationDefaults = {
  email_notifications: true,
  subscription_expiry_reminder_cooldown_hours: 24,
  subscription_expiry_reminder_daily_limit: 1,
  ...Object.fromEntries(customerEventLegacyKeys.map(key => [key, true])),
  ...Object.fromEntries(customerEventChannelKeys.map(key => [key, key.includes('_email_')]))
}

const adminNotificationDefaults = {
  admin_notification_enabled: false,
  admin_email_notification: false,
  admin_notification_email: '',
  admin_telegram_notification: false,
  admin_telegram_bot_token: '',
  admin_telegram_chat_id: '',
  admin_bark_notification: false,
  admin_bark_server_url: 'https://api.day.app',
  admin_bark_device_key: '',
  admin_abnormal_login_alert_enabled: true,
  ...Object.fromEntries(adminLegacyEventKeys.map(key => [key, false])),
  ...Object.fromEntries(adminEventChannelKeys.map(key => [key, false]))
}

const toBool = (val) => val === true || val === 'true' || val === '1'
const hasOwn = (obj, key) => Object.prototype.hasOwnProperty.call(obj || {}, key)

export default {
  name: 'AdminSettings',
  components: { Check, Plus, Refresh, Message, Bell },
  setup() {
    const api = useApi()
    const isMobile = useMobile()
    const themeStore = useThemeStore()
    const activeTab = ref('general')
    const generalFormRef = ref()
    const uploadUrl = '/api/v1/admin/upload'
    const pageLoading = ref(false)
    const savingCurrent = ref(false)
    const testingStates = reactive({
      email: false, telegram: false, bark: false, gitee: false, github: false
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

    // Forms should stack vertically on mobile for better UX
    const formLayout = computed(() => ({
      labelWidth: isMobile.value ? 'auto' : '150px',
      labelPosition: isMobile.value ? 'top' : 'right'
    }))
    const settingsTabPosition = computed(() => (isMobile.value ? 'top' : 'left'))

    const generalSettings = reactive({
      site_name: '', site_description: '', domain_name: '', site_logo: '',
      default_theme: 'default', support_qq: '', support_email: '', unified_auth_enabled: false
    })
    const registrationSettings = reactive({
      registration_enabled: true, email_verification_required: true,
      min_password_length: 8, invite_code_required: false,
      default_subscription_device_limit: 3, default_subscription_duration_months: 1
    })
    const inviteSettings = reactive({
      inviter_reward: 0, invitee_reward: 0, min_order_amount: 0, new_user_only: true
    })
    const notificationSettings = reactive({ ...customerNotificationDefaults })
    const securitySettings = reactive({
      login_fail_limit: 5, login_lock_time: 30, session_timeout: 120,
      abnormal_login_alert_enabled: true
    })
    const themeSettings = reactive({
      default_theme: 'light', allow_user_theme: true, available_themes: THEME_OPTIONS.map(t => t.value)
    })
    const adminNotificationSettings = reactive({ ...adminNotificationDefaults })
    const announcementSettings = reactive({ announcement_enabled: false, announcement_content: '' })
    const nodeHealthSettings = reactive({
      check_interval: 30, max_latency: 3000, test_timeout: 5, test_url: 'https://ping.pe'
    })
    const backupSettings = reactive({
      backup_target: 'gitee',
      backup_gitee_enabled: false, backup_gitee_token: '', backup_gitee_owner: 'moneyfly', backup_gitee_repo: 'backup',
      backup_github_enabled: false, backup_github_token: '', backup_github_owner: 'moneyfly1', backup_github_repo: 'backup',
      backup_auto_enabled: false, backup_auto_interval: 24
    })
    const protocolFilterSettings = reactive({
      clash_protocols: [...ALL_PROTOCOLS], universal_protocols: [...ALL_PROTOCOLS]
    })
    
    const generalRules = { site_name: [{ required: true, message: '请输入网站名称', trigger: 'blur' }] }

    const formatFileSize = (bytes) => {
      if (!bytes) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    }
    
    const loadGeoIPStatus = async () => {
      try {
        const res = await api.get('/admin/settings/geoip/status')
        geoipStatus.value = res.data?.data || res.data || {}
      } catch (e) { console.error('GeoIP Status Error', e) }
    }

    const loadSettings = async () => {
      try {
        pageLoading.value = true
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
        if (data.invite) {
          Object.assign(inviteSettings, data.invite)
          inviteSettings.inviter_reward = Number(inviteSettings.inviter_reward) || 0
          inviteSettings.invitee_reward = Number(inviteSettings.invitee_reward) || 0
          inviteSettings.min_order_amount = Number(inviteSettings.min_order_amount) || 0
          inviteSettings.new_user_only = toBool(inviteSettings.new_user_only)
        }
        if (data.notification) {
          Object.keys(customerNotificationDefaults).forEach(key => {
            if (data.notification[key] === undefined || data.notification[key] === null) {
              notificationSettings[key] = customerNotificationDefaults[key]
            } else if (typeof customerNotificationDefaults[key] === 'boolean') {
              notificationSettings[key] = toBool(data.notification[key])
            } else {
              notificationSettings[key] = data.notification[key]
            }
          })
          const cooldown = parseInt(notificationSettings.subscription_expiry_reminder_cooldown_hours, 10)
          notificationSettings.subscription_expiry_reminder_cooldown_hours = Number.isNaN(cooldown) ? 24 : cooldown
          const dailyLimit = parseInt(notificationSettings.subscription_expiry_reminder_daily_limit, 10)
          notificationSettings.subscription_expiry_reminder_daily_limit = Number.isNaN(dailyLimit) ? 1 : dailyLimit
          applyCustomerNotificationFallbacks(data.notification)
          syncCustomerLegacyNotificationKeys()
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
          Object.keys(adminNotificationDefaults).forEach(key => {
            if (data.admin_notification[key] === undefined || data.admin_notification[key] === null) {
              adminNotificationSettings[key] = adminNotificationDefaults[key]
            } else if (stringFields.includes(key)) {
              adminNotificationSettings[key] = String(data.admin_notification[key] || '')
            } else {
              adminNotificationSettings[key] = toBool(data.admin_notification[key])
            }
          })
          Object.keys(data.admin_notification).forEach(key => {
            if (!(key in adminNotificationSettings)) {
              adminNotificationSettings[key] = stringFields.includes(key) ? String(data.admin_notification[key] || '') : toBool(data.admin_notification[key])
            }
          })
          applyAdminNotificationFallbacks(data.admin_notification)
          syncAdminLegacyNotificationKeys()
        }
        if (data.backup) {
          Object.assign(backupSettings, data.backup)
          backupSettings.backup_target = backupSettings.backup_target || 'gitee'
          backupSettings.backup_gitee_enabled = toBool(backupSettings.backup_gitee_enabled)
          backupSettings.backup_github_enabled = toBool(backupSettings.backup_github_enabled)
          backupSettings.backup_auto_enabled = toBool(backupSettings.backup_auto_enabled)
          backupSettings.backup_auto_interval = parseInt(backupSettings.backup_auto_interval) || 24
        }
        if (data.protocol_filter) {
          const pf = data.protocol_filter
          if (pf.clash_protocols) {
            try { protocolFilterSettings.clash_protocols = typeof pf.clash_protocols === 'string' ? JSON.parse(pf.clash_protocols) : pf.clash_protocols }
            catch { protocolFilterSettings.clash_protocols = [...ALL_PROTOCOLS] }
          }
          if (pf.universal_protocols) {
            try { protocolFilterSettings.universal_protocols = typeof pf.universal_protocols === 'string' ? JSON.parse(pf.universal_protocols) : pf.universal_protocols }
            catch { protocolFilterSettings.universal_protocols = [...ALL_PROTOCOLS] }
          }
        }
      } catch (error) {
        ElMessage.error('加载设置失败: ' + (error.response?.data?.message || error.message))
      } finally {
        pageLoading.value = false
      }
    }

    const refreshSettings = async () => {
      await Promise.all([ loadSettings(), loadGeoIPStatus() ])
      ElMessage.success('设置已刷新')
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
        ElMessage.error(error.response?.data?.message || error.message || '保存失败')
        return false
      }
    }

    const syncCustomerLegacyNotificationKeys = () => {
      customerEventSwitches.forEach(item => {
        notificationSettings[item.legacyKey] = Boolean(notificationSettings[item.channels.email])
      })
    }
    const applyCustomerNotificationFallbacks = (source) => {
      customerEventSwitches.forEach(item => {
        const legacyEnabled = toBool(source[item.legacyKey])
        Object.entries(item.channels).forEach(([channel, key]) => {
          if (!hasOwn(source, key)) notificationSettings[key] = channel === 'email' ? legacyEnabled : false
        })
      })
    }
    const syncAdminLegacyNotificationKeys = () => {
      adminNotificationEvents.forEach(item => {
        adminNotificationSettings[item.key] = Boolean(
          adminNotificationSettings[item.channels.email] || adminNotificationSettings[item.channels.telegram] || adminNotificationSettings[item.channels.bark]
        )
      })
    }
    const applyAdminNotificationFallbacks = (source) => {
      adminNotificationEvents.forEach(item => {
        const legacyEnabled = toBool(source[item.key])
        Object.entries(item.channels).forEach(([, key]) => {
          if (!hasOwn(source, key)) adminNotificationSettings[key] = legacyEnabled
        })
      })
    }

    const saveGeneralSettings = async () => {
      const data = { ...generalSettings, unified_auth_enabled: generalSettings.unified_auth_enabled }
      const success = await handleSave(() => api.put('/admin/settings/general', data), '基本设置保存成功', generalFormRef.value)
      if (success) await loadSettings()
    }
    const saveRegistrationSettings = () => handleSave(() => api.put('/admin/settings/registration', registrationSettings), '注册设置保存成功')
    const saveInviteSettings = () => handleSave(() => api.put('/admin/settings/invite', inviteSettings), '邀请设置保存成功')
    const saveNotificationSettings = () => {
      syncCustomerLegacyNotificationKeys()
      return handleSave(() => api.put('/admin/settings/notification', notificationSettings), '通知设置保存成功')
    }
    const saveSecuritySettings = () => handleSave(() => api.put('/admin/settings/security', securitySettings), '安全设置保存成功')
    const saveAnnouncementSettings = () => handleSave(() => api.put('/admin/settings/announcement', announcementSettings), '公告设置保存成功')
    const saveBackupSettings = () => handleSave(() => api.put('/admin/settings/backup', backupSettings), '备份设置保存成功')
    const saveProtocolFilterSettings = () => handleSave(() => api.put('/admin/settings/protocol-filter', protocolFilterSettings), '协议过滤设置保存成功')
    const saveThemeSettings = async () => {
      const success = await handleSave(() => api.put('/admin/settings/theme', themeSettings), '主题设置保存成功')
      if (success && themeSettings.default_theme) await themeStore.setTheme(themeSettings.default_theme)
    }
    const saveNodeHealthSettings = () => {
      const data = {
        node_health_check_interval: String(nodeHealthSettings.check_interval), node_max_latency: String(nodeHealthSettings.max_latency),
        node_test_timeout: String(nodeHealthSettings.test_timeout), test_url: nodeHealthSettings.test_url
      }
      return handleSave(() => api.put('/admin/settings/node_health', data), '监控配置保存成功')
    }
    const saveAdminNotificationSettings = () => {
      syncAdminLegacyNotificationKeys()
      const data = {}
      for (const [key, val] of Object.entries(adminNotificationSettings)) data[key] = typeof val === 'boolean' ? (val ? 'true' : 'false') : (val || '')
      return handleSave(() => adminAPI.updateAdminNotificationSettings(data), '管理员告警设置保存成功')
    }

    const currentTabSaveMap = {
      general: saveGeneralSettings, registration: saveRegistrationSettings, invite: saveInviteSettings,
      notification: async () => (await saveNotificationSettings() && await saveAdminNotificationSettings()),
      announcement: saveAnnouncementSettings, theme: saveThemeSettings, 'node-health': saveNodeHealthSettings,
      security: saveSecuritySettings, backup: saveBackupSettings, 'protocol-filter': saveProtocolFilterSettings
    }

    const saveCurrentTab = async () => {
      const save = currentTabSaveMap[activeTab.value]
      if (!save) return
      savingCurrent.value = true
      try { await save() } finally { savingCurrent.value = false }
    }

    const updateGeoIPDatabase = async () => {
      geoipUpdating.value = true
      await handleSave(() => api.post('/admin/settings/geoip/update', { type: geoipDatabaseType.value }), `${geoipDatabaseType.value === 'dbip' ? 'DB-IP' : 'GeoLite2'} 数据库下载成功`)
      await loadGeoIPStatus()
      geoipUpdating.value = false
    }
    const switchDatabase = async (path) => {
      switchingDatabase.value = true
      await handleSave(() => api.post('/admin/settings/geoip/switch', { path }), '数据库切换成功')
      await loadGeoIPStatus()
      switchingDatabase.value = false
    }
    const flushCache = async () => {
      try {
        await ElMessageBox.confirm('确定要清除所有缓存吗？此操作不可撤销。', '警告', { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' })
        cacheClearing.value = true
        const res = await api.post('/admin/settings/cache/flush')
        if (res.data.success) ElMessage.success('缓存已清除')
        else ElMessage.error(res.data.message || '清除失败')
      } catch (e) {
        if (e !== 'cancel') ElMessage.error('清除失败: ' + (e.response?.data?.message || e.message))
      } finally {
        cacheClearing.value = false
      }
    }

    const testNotification = async (type) => {
      const apiMap = {
        email: { api: adminAPI.testAdminEmailNotification, msg: '邮件测试已发送' },
        telegram: { api: adminAPI.testAdminTelegramNotification, msg: 'Telegram 测试已发送' },
        bark: { api: adminAPI.testAdminBarkNotification, msg: 'Bark 测试已发送' }
      }
      if (!apiMap[type]) return
      testingStates[type] = true
      try {
        const res = await apiMap[type].api()
        if (res.data.success) ElMessage.success(apiMap[type].msg)
        else ElMessage.error(res.data.message || '测试失败')
      } catch (e) { ElMessage.error('测试失败: ' + (e.response?.data?.message || e.message)) }
      finally { testingStates[type] = false }
    }

    const testGiteeConnection = async () => {
      const { backup_gitee_token: token, backup_gitee_owner: owner, backup_gitee_repo: repo } = backupSettings
      if (!token || !owner || !repo) return ElMessage.error('请填写完整的 Gitee 配置')
      testingStates.gitee = true
      try {
        await api.put('/admin/settings/backup', backupSettings)
        const res = await api.post('/admin/backup/test-gitee', { token, owner, repo })
        if (res.data?.success !== false) ElMessage.success('Gitee 连接成功！' + (res.data?.data?.message || ''))
        else ElMessage.error(res.data?.message || '测试失败')
      } catch (e) { ElMessage.error('测试失败: ' + (e.response?.data?.message || e.message)) }
      finally { testingStates.gitee = false }
    }
    const testGitHubConnection = async () => {
      const { backup_github_token: token, backup_github_owner: owner, backup_github_repo: repo } = backupSettings
      if (!token || !owner || !repo) return ElMessage.error('请填写完整的 GitHub 配置')
      testingStates.github = true
      try {
        await api.put('/admin/settings/backup', backupSettings)
        const res = await api.post('/admin/backup/test-github', { token, owner, repo })
        if (res.data?.success !== false) ElMessage.success('GitHub 连接成功！' + (res.data?.data?.message || ''))
        else ElMessage.error(res.data?.message || '测试失败')
      } catch (e) { ElMessage.error('测试失败: ' + (e.response?.data?.message || e.message)) }
      finally { testingStates.github = false }
    }

    const checkUploadStatus = async (taskId, target) => {
      try {
        const res = await api.get(`/admin/backup/upload-status/${taskId}?target=${target || 'gitee'}`)
        const status = res.data?.data || res.data
        if (status) {
          uploadStatus.value = status
          if (status.status === 'success' || status.status === 'failed') {
            if (uploadStatusInterval.value) { clearInterval(uploadStatusInterval.value); uploadStatusInterval.value = null }
            if (status.status === 'success') ElMessage.success(status.message || '上传成功')
            else ElMessage.error(status.message || status.error || '上传失败')
          }
        }
      } catch (e) { console.error('查询上传状态失败:', e) }
    }

    const startStatusPolling = (taskId, target) => {
      if (uploadStatusInterval.value) clearInterval(uploadStatusInterval.value)
      checkUploadStatus(taskId, target)
      uploadStatusInterval.value = setInterval(() => checkUploadStatus(taskId, target), 2000)
    }
    const stopStatusPolling = () => {
      if (uploadStatusInterval.value) { clearInterval(uploadStatusInterval.value); uploadStatusInterval.value = null }
      uploadStatus.value = null; uploadTaskId.value = null
    }

    const createManualBackup = async () => {
      creatingBackup.value = true; uploadStatus.value = null; uploadTaskId.value = null
      try {
        await api.put('/admin/settings/backup', backupSettings)
        const res = await api.post('/admin/backup', {}, { timeout: 60000 })
        if (res.data?.success !== false) {
          const d = res.data.data || res.data
          let msg = '打包成功！'
          if (d.filename) msg += ` ${d.filename}`
          
          const uploadInfo = d.github || d.gitee || {}
          if (uploadInfo.async && uploadInfo.task_id) {
            uploadTaskId.value = uploadInfo.task_id
            uploadTarget.value = uploadInfo.target || (d.github ? 'github' : 'gitee')
            uploadStatus.value = { status: 'uploading', progress: 0, message: '准备上传...', start_time: new Date().toISOString(), file_name: d.filename || '', file_size: d.size || 0 }
            ElMessage.success(msg + ' | 后台上传中...')
            startStatusPolling(uploadInfo.task_id, uploadTarget.value)
          } else if (d.github?.uploaded || d.gitee?.uploaded) {
            ElMessage.success(msg + ` | 已传至 ${d.github ? 'GitHub' : 'Gitee'}`)
          } else if (d.github?.error || d.gitee?.error) {
            ElMessage.warning(msg + ` | 上传失败: ${d.github?.error || d.gitee?.error}`)
          } else {
            ElMessage.success(msg)
          }
        } else ElMessage.error(res.data?.message || '打包失败')
      } catch (e) {
        if (e.message?.includes('timeout')) ElMessage.warning('请求超时，后台可能仍在进行任务')
        else ElMessage.error('执行失败: ' + (e.response?.data?.message || e.message))
      } finally { creatingBackup.value = false }
    }

    const handleLogoSuccess = (res) => {
      const url = res?.data?.url || res?.url
      if (res?.success || url) { generalSettings.site_logo = url || ''; ElMessage.success('Logo 上传成功') }
      else ElMessage.error('Logo 上传失败')
    }
    const beforeLogoUpload = (file) => {
      if (!file.type.startsWith('image/')) return ElMessage.error('仅支持图片文件!') && false
      if (file.size / 1024 / 1024 >= 2) return ElMessage.error('图片不能超过 2MB!') && false
      return true
    }

    onMounted(() => Promise.all([loadSettings(), loadGeoIPStatus()]))
    onBeforeUnmount(() => stopStatusPolling())

    return {
      activeTab, isMobile, formLayout, settingsTabPosition, pageLoading, savingCurrent, Refresh, Check,
      generalSettings, generalRules, generalFormRef, registrationSettings, inviteSettings, notificationSettings, securitySettings,
      themeSettings, adminNotificationSettings, announcementSettings, nodeHealthSettings, backupSettings,
      uploadUrl, themeOptions: THEME_OPTIONS,
      customerMethodSwitches, customerEventSwitches, adminNotificationEvents,
      testingStates, geoipStatus, geoipUpdating, geoipDatabaseType, switchingDatabase, creatingBackup, cacheClearing,
      uploadStatus, uploadTaskId, stopStatusPolling,
      saveGeneralSettings, saveRegistrationSettings, saveInviteSettings, saveNotificationSettings, saveSecuritySettings, saveThemeSettings, saveAnnouncementSettings,
      saveNodeHealthSettings, saveAdminNotificationSettings, saveBackupSettings, saveProtocolFilterSettings, saveCurrentTab, refreshSettings, protocolFilterSettings, allProtocols: ALL_PROTOCOLS,
      testNotification, testGiteeConnection, testGitHubConnection, createManualBackup, updateGeoIPDatabase, switchDatabase, flushCache, handleLogoSuccess, beforeLogoUpload, formatFileSize
    }
  }
}
</script>

<style scoped>
/* ========== 全局布局与色彩规划 ========== */
.admin-settings {
  padding: 16px;
  background-color: var(--el-bg-color-page);
  min-height: calc(100vh - 84px);
}

.settings-page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.header-copy h1 {
  font-size: 22px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin: 0 0 6px 0;
}

.header-copy p {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  margin: 0;
}

/* ========== Tabs 与主体表单容器 ========== */
.settings-shell {
  border-radius: 12px;
  border: 1px solid var(--el-border-color-light);
}

.settings-tabs :deep(.el-tabs__header.is-left) {
  width: 200px;
  background: var(--el-fill-color-light);
  margin-right: 0;
  border-right: 1px solid var(--el-border-color-light);
  padding: 12px 0;
}

.settings-tabs :deep(.el-tabs__item) {
  font-size: 14px;
  font-weight: 500;
  height: 48px;
  line-height: 48px;
  padding: 0 24px;
  text-align: left;
  justify-content: flex-start;
  color: var(--el-text-color-regular);
}
.settings-tabs :deep(.el-tabs__item.is-active) {
  background: var(--el-bg-color);
  color: var(--el-color-primary);
}
.settings-tabs :deep(.el-tabs__active-bar.is-left) { left: 0; right: auto; width: 3px; }
.settings-tabs :deep(.el-tabs__content) { padding: 20px; min-height: 600px; }

/* ========== 表单元素标准规范 ========== */
.settings-form { max-width: 900px; }
.settings-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin: 16px 0 10px;
  padding-left: 10px;
  border-left: 4px solid var(--el-color-primary);
  line-height: 1;
}
.settings-section-title:first-child { margin-top: 0; }

.input-short { width: 100%; max-width: 240px; }
.input-medium { width: 100%; max-width: 380px; }
.input-large { width: 100%; max-width: 600px; }
.input-full { width: 100%; }

.unit-text {
  margin-left: 12px;
  color: var(--el-text-color-secondary);
}

.form-tip {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
  margin-top: 4px;
}
.mt-2 { margin-top: 8px; }
.mt-3 { margin-top: 12px; }
.mt-4 { margin-top: 16px; }
.mt-5 { margin-top: 24px; }
.mb-2 { margin-bottom: 8px; }
.mb-3 { margin-bottom: 12px; }
.mb-4 { margin-bottom: 16px; }
.mb-5 { margin-bottom: 24px; }
.text-danger { color: var(--el-color-danger); }
.text-success { color: var(--el-color-success); }
.text-sm { font-size: 14px; }

/* ========== 开关卡片样式 (Switch Cards) ========== */
.switch-group {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}
.switch-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  background: var(--el-fill-color-blank);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  transition: all 0.2s;
}
.switch-card:hover { border-color: var(--el-color-primary-light-5); background: var(--el-color-primary-light-9); }
.switch-card.master-switch { border-color: var(--el-color-primary); background: var(--el-color-primary-light-9); }

.switch-card-content { display: flex; flex-direction: column; min-width: 0; gap: 2px; padding-right: 12px; }
.switch-card .title { font-weight: 600; font-size: 13px; color: var(--el-text-color-primary); }
.switch-card .desc { font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.3; }

/* ========== 单面板布局 (公告、主题等) ========== */
.single-panel-wrapper { max-width: 720px; }
.theme-top-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  align-items: end;
  margin-bottom: 4px;
}
.theme-top-row .switch-card { height: 100%; }
.theme-checkbox-group { display: flex; flex-wrap: wrap; gap: 8px; }
.theme-checkbox-group :deep(.el-checkbox) { margin-right: 0; background: var(--el-bg-color); }
.backup-fields-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.inline-fields-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 12px;
}
.inline-fields-row :deep(.el-form-item) { margin-bottom: 0; }

/* ========== 通知布局与矩阵表 ========== */
.notification-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  align-items: start;
}
.notification-panel {
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
  padding: 16px;
}
.notification-panel :deep(.el-form-item) { margin-bottom: 12px; }
.notification-panel :deep(.el-form-item__label) { padding-bottom: 4px; }
.panel-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.panel-header h3 { margin: 0; font-size: 15px; font-weight: 600; display: flex; align-items: center; gap: 6px; }
.panel-header.warning-header h3 { color: var(--el-color-warning); }

.header-tags { display: flex; gap: 8px; flex-wrap: wrap; }
.matrix-table { border-radius: 6px; overflow: hidden; }
.matrix-table :deep(th.el-table__cell) { background-color: var(--el-fill-color-dark) !important; color: var(--el-text-color-primary); font-weight: 600; }

.sub-config-box {
  background: var(--el-bg-color);
  padding: 12px; border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}
.expiry-config-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  background: var(--el-bg-color);
  padding: 12px;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}
.expiry-config-row :deep(.el-form-item) { margin-bottom: 0; }

/* ========== 管理员渠道配置卡 ========== */
.admin-channels-list { display: flex; flex-direction: column; gap: 12px; }
.channel-config-card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.3s;
}
.channel-config-card.is-disabled { opacity: 0.8; background: var(--el-fill-color-lighter); }
.channel-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 16px;
  background: var(--el-fill-color-lighter);
}
.channel-title { font-weight: 600; font-size: 14px; color: var(--el-text-color-primary); }
.channel-body { padding: 16px; border-top: 1px solid var(--el-border-color-lighter); }

/* ========== 备份配置 Radio 与卡片块 ========== */
.radio-block-group { display: flex; gap: 10px; flex-wrap: wrap; }
.radio-block-group.radio-block-vertical { flex-direction: column; }
.radio-block-group.radio-block-vertical :deep(.el-radio) { width: 100%; box-sizing: border-box; }
.radio-block-group :deep(.el-radio) {
  margin: 0; padding: 8px 14px 8px 8px; height: auto;
  border: 1px solid var(--el-border-color); border-radius: 8px;
  background: var(--el-bg-color); transition: all 0.2s;
}
.radio-block-group :deep(.el-radio.is-checked) {
  border-color: var(--el-color-primary); background: var(--el-color-primary-light-9);
}
.radio-content { display: flex; flex-direction: column; margin-left: 8px; }
.radio-title { font-weight: 600; color: var(--el-text-color-primary); font-size: 13px; margin-bottom: 2px; display: flex; align-items: center; gap: 6px;}
.radio-desc { font-size: 12px; color: var(--el-text-color-secondary); }

.platform-config-block {
  border: 1px solid var(--el-border-color);
  border-radius: 10px;
  overflow: hidden;
  opacity: 0.6;
  transition: opacity 0.3s, box-shadow 0.3s;
}
.platform-config-block.is-active { opacity: 1; box-shadow: var(--el-box-shadow-light); }
.platform-config-block .block-header {
  padding: 16px 20px; background: var(--el-fill-color-light);
  display: flex; justify-content: space-between; align-items: center;
}
.platform-config-block .block-title { font-weight: 600; font-size: 15px; }
.platform-config-block .block-body { padding: 20px; background: var(--el-bg-color); }

/* ========== 杂项 (Checkbox, Upload等) ========== */
.checkbox-panel { background: var(--el-fill-color-light); padding: 20px; border-radius: 8px; border: 1px solid var(--el-border-color-lighter); }
.protocol-checkbox-group { display: flex; flex-wrap: wrap; gap: 12px; }
.protocol-checkbox-group :deep(.el-checkbox) { margin-right: 0; background: var(--el-bg-color); }

.avatar-uploader .avatar { width: 56px; height: 56px; display: block; object-fit: cover; border-radius: 6px;}
.avatar-uploader :deep(.el-upload) {
  border: 1px dashed var(--el-border-color); border-radius: 6px;
  cursor: pointer; overflow: hidden; transition: all 0.2s;
}
.avatar-uploader :deep(.el-upload:hover) { border-color: var(--el-color-primary); background: var(--el-color-primary-light-9); }
.avatar-uploader-icon { width: 56px; height: 56px; color: var(--el-text-color-secondary); font-size: 18px; line-height: 56px; text-align: center; }

.theme-color-block { display: inline-block; width: 14px; height: 14px; margin-right: 8px; border-radius: 4px; vertical-align: middle; border: 1px solid rgba(0,0,0,0.1); }
.theme-color-block-sm { display: inline-block; width: 16px; height: 16px; border-radius: 4px; border: 1px solid rgba(0,0,0,0.1); }
.flex-align-center { display: inline-flex; align-items: center; gap: 8px; }

/* ========== 移动端响应式 (≤ 768px) ========== */
@media screen and (max-width: 1024px) {
  .notification-layout { grid-template-columns: 1fr; }
}

@media screen and (max-width: 768px) {
  .admin-settings { padding: 10px; }
  .settings-page-header { flex-direction: column; align-items: flex-start; gap: 12px; }
  .settings-page-header .header-actions { width: 100%; display: flex; gap: 10px; }
  .settings-page-header .header-actions .el-button { flex: 1; margin: 0; }
  
  .settings-tabs :deep(.el-tabs__header.is-top) { background: var(--el-bg-color); padding: 0 4px; }
  .settings-tabs :deep(.el-tabs__item.is-top) { font-size: 14px; padding: 0 16px; }
  .settings-tabs :deep(.el-tabs__content) { padding: 16px 12px; min-height: auto; }
  
  .input-short, .input-medium, .input-large, .settings-form :deep(.el-input), .settings-form :deep(.el-select) { max-width: 100%; }
  .switch-group { grid-template-columns: 1fr; }
  
  .radio-block-group { flex-direction: column; }
  .radio-block-group :deep(.el-radio) { width: 100%; box-sizing: border-box; }
  
  .notification-panel { padding: 12px 10px; border-radius: 8px; }
  .panel-header { flex-direction: column; align-items: flex-start; gap: 8px; }
  .theme-checkbox-group :deep(.el-checkbox) { width: calc(50% - 4px); min-width: 0; }
  .theme-top-row { grid-template-columns: 1fr; }
  .expiry-config-row { grid-template-columns: 1fr; }
  .inline-fields-row { grid-template-columns: 1fr; }
  .backup-fields-row { grid-template-columns: 1fr; }
  
  .settings-section-title { font-size: 15px; margin: 24px 0 16px; }
  .platform-config-block .block-header { padding: 14px 12px; }
  .platform-config-block .block-body { padding: 16px 12px; }
}
</style>