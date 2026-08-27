<template>
  <div class="list-container admin-config">
    <el-card>
  <template #header>
    <div class="card-header">
      <h2>配置管理</h2>
      <p>管理软件下载配置和邮件配置</p>
    </div>
  </template>
  <el-tabs v-model="activeTab" type="border-card">
    <el-tab-pane label="软件下载配置" name="software">
      <div class="config-section">
        <h3>软件下载链接配置</h3>
        <el-form
          :model="softwareForm"
          label-width="150px"
        >
              <el-divider content-position="left">Windows 软件</el-divider>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Clash for Windows">
                    <el-input v-model="softwareForm.clash_windows_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="V2rayN">
                    <el-input v-model="softwareForm.v2rayn_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Clash Part">
                    <el-input v-model="softwareForm.clash_party_windows_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="Clash Verge">
                    <el-input v-model="softwareForm.clash_verge_windows_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Hiddify">
                    <el-input v-model="softwareForm.hiddify_windows_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="FlClash">
                    <el-input v-model="softwareForm.flash_windows_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-divider content-position="left">Android 软件</el-divider>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Clash Meta">
                    <el-input v-model="softwareForm.clash_android_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="V2rayNG">
                    <el-input v-model="softwareForm.v2rayng_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Hiddify">
                    <el-input v-model="softwareForm.hiddify_android_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="FlClash">
                    <el-input v-model="softwareForm.flash_android_url" placeholder="请输入下载链接（可留空）" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-divider content-position="left">macOS 软件</el-divider>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="FlClash">
                    <el-input v-model="softwareForm.flash_macos_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="FlClash (Apple 芯片)">
                    <el-input v-model="softwareForm.flash_macos_arm_url" placeholder="Apple 芯片版本（可留空）" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Clash Part (Intel)">
                    <el-input v-model="softwareForm.clash_party_macos_url" placeholder="Intel 芯片版本" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="Clash Part (Apple 芯片)">
                    <el-input v-model="softwareForm.clash_party_macos_arm_url" placeholder="Apple 芯片版本（可留空）" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Clash Verge (Intel)">
                    <el-input v-model="softwareForm.clash_verge_macos_url" placeholder="Intel 芯片版本" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="Clash Verge (Apple 芯片)">
                    <el-input v-model="softwareForm.clash_verge_macos_arm_url" placeholder="Apple 芯片版本（可留空）" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="V2rayN (Intel)">
                    <el-input v-model="softwareForm.v2rayn_macos_url" placeholder="Intel 芯片版本（可留空）" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="V2rayN (Apple 芯片)">
                    <el-input v-model="softwareForm.v2rayn_macos_arm_url" placeholder="Apple 芯片版本（可留空）" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Hiddify (Intel)">
                    <el-input v-model="softwareForm.hiddify_macos_url" placeholder="Intel 芯片版本（可留空）" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="Hiddify (Apple 芯片)">
                    <el-input v-model="softwareForm.hiddify_macos_arm_url" placeholder="Apple 芯片版本（可留空）" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-divider content-position="left">iOS 软件</el-divider>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Shadowrocket">
                    <el-input v-model="softwareForm.shadowrocket_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
                <el-col :span="12"></el-col>
              </el-row>
              <el-form-item class="config-buttons-group">
                <el-button type="primary" @click="saveSoftwareConfig" :loading="softwareLoading" class="config-action-btn">
                  保存软件配置
                </el-button>
                <el-button @click="loadSoftwareConfig" class="config-action-btn">
                  重新加载
                </el-button>
              </el-form-item>
            </el-form>
          </div>
          <div class="config-section">
            <el-divider content-position="left">123云盘自动填充（可选）</el-divider>
            <el-alert
              type="info"
              show-icon
              :closable="false"
              title="配置 123 云盘账号后，可一键搜索云盘里的安装包并自动填入上方下载链接。动态模式（pan://）下用户每次点击下载时实时生成新链接，链接永不失效；静态模式则填入当前有效链接，失效后需再次一键填充。"
            />
            <el-form :model="panForm" label-width="150px" style="margin-top: 16px">
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="123云盘账号">
                    <el-input v-model="panForm.username" placeholder="手机号或邮箱" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="账号密码">
                    <el-input v-model="panForm.password" type="password" show-password placeholder="密码仅保存在本服务器" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="分享密码(可选)">
                    <el-input v-model="panForm.share_pwd" placeholder="生成的分享链接密码，留空为无密码" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="链接模式">
                    <el-select v-model="panForm.mode" style="width: 100%">
                      <el-option label="直链（已验证可直接下载）" value="direct" />
                    </el-select>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-form-item label="Cookie（可选）">
                <el-input
                  v-model="panForm.cookies"
                  type="textarea"
                  :rows="2"
                  placeholder="从浏览器 DevTools 复制 Cookie 粘贴到此；填写后优先于账号密码。留空则使用账号密码登录"
                />
              </el-form-item>
              <el-form-item label="Token（可选）">
                <el-input
                  v-model="panForm.token"
                  type="textarea"
                  :rows="1"
                  placeholder="浏览器 DevTools → Application → Cookies → yun.123pan.cn → 复制 jwt 的值；绕过登录不触发风控（不是 localStorage 的 authorToken）"
                />
              </el-form-item>
              <el-divider content-position="left">阿里云盘（海外服务器推荐）</el-divider>
              <el-alert
                type="info"
                show-icon
                :closable="false"
                title="阿里云盘 refresh_token 登录：不绑定 IP、无境外风控，海外服务器可用（推荐替代 123 云盘）"
                description="获取 refresh_token：浏览器登录 www.alipan.com → F12 → Application → Local Storage → https://www.alipan.com → 找到键名 token，复制其中 refresh_token 的值"
                style="margin-bottom: 8px"
              />
              <el-form-item label="阿里云盘 refresh_token">
                <el-input v-model="panForm.aliyun_refresh_token" type="textarea" :rows="2" placeholder="粘贴 refresh_token（配置后优先使用阿里云盘，同步/直链自动切换）" />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" plain :loading="aliyunTesting" @click="testAliyun">
                  测试阿里云盘连接
                </el-button>
                <span v-if="aliyunTestResult" class="pan-test-result" style="margin-left: 8px">{{ aliyunTestResult }}</span>
              </el-form-item>
              <el-form-item class="config-buttons-group">
                <el-button type="primary" @click="savePanConfig" :loading="panSaving" class="config-action-btn">
                  保存云盘配置
                </el-button>
                <el-button @click="testPanConfig" :loading="panTesting" class="config-action-btn">
                  测试连接
                </el-button>
                <el-button @click="loadPanConfig" class="config-action-btn">
                  重新加载
                </el-button>
                <span v-if="panTestResult" class="pan-test-result">{{ panTestResult }}</span>
              </el-form-item>
              <el-alert
                v-if="smsMode && smsNeedsCaptcha"
                type="warning"
                show-icon
                :closable="false"
                title="滑块验证后自动发送验证码"
                description="若下方滑块未显示：请打开 yun.123pan.cn 登录页 → 用该手机号获取验证码（浏览器内滑动）→ 把收到的验证码填到下方完成登录"
                style="margin-bottom: 8px"
              />
              <div v-if="smsMode && smsNeedsCaptcha" id="pan-captcha" style="margin-bottom: 8px; max-width: 340px"></div>
              <el-form-item v-if="smsMode" label="短信验证码">
                <el-input v-model="panSmsCode" placeholder="输入手机收到的验证码" style="max-width: 220px" />
                <el-button type="success" :loading="panSmsLogging" @click="smsLogin" class="config-action-btn" style="margin-left: 8px">
                  完成登录（自动保存 token）
                </el-button>
              </el-form-item>
              <el-form-item v-if="tokenExpiryText">
                <el-tag :type="tokenExpiryDanger ? 'danger' : 'success'" size="small">
                  {{ tokenExpiryText }}
                </el-tag>
                <span class="pan-soft-key" style="margin-left: 8px">token 过期后需重新短信登录</span>
              </el-form-item>
              <el-form-item>
                <el-button type="warning" plain :loading="panSmsSending" @click="smsSend">
                  短信验证码登录
                </el-button>
                <el-button type="success" plain :loading="qrLoading" @click="qrLogin">
                  二维码登录（手机 App 扫码）
                </el-button>
                <span v-if="smsStatus" class="pan-test-result" style="margin-left: 8px">{{ smsStatus }}</span>
              </el-form-item>
            </el-form>

            <h4 class="pan-mapping-title">软件 ↔ 云盘文件名关键词</h4>
            <p class="pan-tip">
              每个软件填一个能在云盘里搜到安装包的关键词（如 v2rayN）。<strong>留空 = 该软件不用云盘</strong>，
              继续使用上方手动填的外部链接（官网 / App Store / GitHub 等均可），一键生成时会跳过留空的行，不会覆盖外部链接。
              同一软件云盘里有多个平台安装包（exe / dmg / apk）时，用<strong>扩展名过滤</strong>精确锁定，并自动取最新版本。
            </p>
            <el-table :data="panMapRows" size="small" border>
              <el-table-column label="软件" width="200">
                <template #default="{ row }">
                  <div class="pan-soft-name">{{ row.label }}</div>
                  <div class="pan-soft-key">{{ row.key }}</div>
                </template>
              </el-table-column>
              <el-table-column label="云盘文件名关键词">
                <template #default="{ row }">
                  <el-input v-model="panMap[row.key]" size="small" placeholder="留空 = 使用外部链接" clearable />
                </template>
              </el-table-column>
              <el-table-column label="扩展名过滤" width="110">
                <template #default="{ row }">
                  <el-select v-model="panExts[row.key]" size="small" clearable placeholder="不限">
                    <el-option v-for="e in PAN_EXT_OPTIONS" :key="e" :label="e" :value="e" />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column label="当前配置" width="170">
                <template #default="{ row }">
                  <el-tag v-if="isPanDynamic(row.key)" type="success" size="small">云盘动态</el-tag>
                  <el-tag v-else-if="softwareForm[row.key]" type="info" size="small">外部链接</el-tag>
                  <el-tag v-else type="warning" size="small">未配置</el-tag>
                  <el-tooltip
                    v-if="softwareForm[row.key]"
                    :content="softwareForm[row.key]"
                    placement="top"
                    :show-after="300"
                  >
                    <div class="pan-soft-key pan-current-value">{{ shortUrl(row.key) }}</div>
                  </el-tooltip>
                </template>
              </el-table-column>
              <el-table-column label="搜索验证" width="100" align="center">
                <template #default="{ row }">
                  <el-button size="small" :loading="panTestingKey === row.key" @click="testPanKeyword(row)">测试</el-button>
                </template>
              </el-table-column>
            </el-table>

            <el-form-item class="config-buttons-group" style="margin-top: 16px">
              <el-button type="primary" @click="panRefresh(true)" :loading="panRefreshing" class="config-action-btn">
                一键生成并填充（动态 · 实时解析）
              </el-button>
              <el-button @click="panRefresh(false)" :loading="panRefreshing" class="config-action-btn">
                一键生成并填充（静态链接）
              </el-button>
            </el-form-item>
          </div>

          <div class="config-section">
            <el-divider content-position="left">GitHub 软件库自动同步（定时）</el-divider>
            <el-alert
              type="info"
              show-icon
              :closable="false"
              title="自动从 GitHub 获取 v2rayN / Hiddify / Clash Verge / Clash Part / V2rayNG / Clash Meta 的最新版本，下载安装包上传到你的 123 云盘，并自动设置直链（pan://）。仅当 GitHub 有新版本时才会下载上传，其余时间跳过。"
            />
            <el-form label-width="150px" style="margin-top: 16px">
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="启用定时同步">
                    <el-switch v-model="panForm.sync_enabled" active-text="开启" inactive-text="关闭" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="同步间隔（小时）">
                    <el-input-number v-model="panForm.sync_interval_hours" :min="1" :max="168" :precision="0" style="width: 140px" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-form-item class="config-buttons-group">
                <el-button type="primary" @click="savePanConfig" :loading="panSaving" class="config-action-btn">
                  保存定时设置
                </el-button>
                <el-button type="warning" @click="runPanSync" :loading="panSyncing" class="config-action-btn">
                  立即同步
                </el-button>
                <el-button @click="loadPanSyncStatus" class="config-action-btn">
                  刷新状态
                </el-button>
                <span v-if="syncStatusText" class="pan-test-result">{{ syncStatusText }}</span>
              </el-form-item>
            </el-form>

            <h4 class="pan-mapping-title">版本对照（GitHub 最新版 ↔ 云盘已同步版本）</h4>
            <el-table :data="panVersions" size="small" border max-height="420">
              <el-table-column prop="name" label="软件" width="130" />
              <el-table-column prop="label" label="平台/架构" width="160" />
              <el-table-column label="GitHub 版本" width="110" align="center">
                <template #default="{ row }">
                  <span v-if="row.github_version">v{{ row.github_version }}</span>
                  <el-tag v-else type="danger" size="small">获取失败</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="云盘版本" width="110" align="center">
                <template #default="{ row }">
                  <span v-if="row.cloud_version">v{{ row.cloud_version }}</span>
                  <span v-else class="pan-soft-key">未上传</span>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="110" align="center">
                <template #default="{ row }">
                  <el-tag v-if="row.custom" type="warning" size="small">自定义链接</el-tag>
                  <el-tag v-else-if="row.synced" type="success" size="small">已同步</el-tag>
                  <el-tag v-else-if="row.cloud_version" type="warning" size="small">待更新</el-tag>
                  <el-tag v-else type="info" size="small">待上传</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="file_name" label="云盘文件" min-width="200" show-overflow-tooltip />
            </el-table>
          </div>
        </el-tab-pane>
        <el-tab-pane label="邮件配置" name="email">
          <el-form
            :model="emailForm"
            label-width="120px"
            class="email-config-form"
          >
            <el-form-item label="SMTP服务器">
              <el-input v-model="emailForm.smtp_host" placeholder="例如: smtp.gmail.com" />
            </el-form-item>
            <el-form-item label="SMTP端口">
              <el-input-number 
                v-model="emailForm.smtp_port" 
                :min="1" 
                :max="65535"
                :precision="0"
                :step="1"
              />
            </el-form-item>
            <el-form-item label="邮箱账号">
              <el-input v-model="emailForm.email_username" placeholder="邮箱地址" />
            </el-form-item>
            <el-form-item label="邮箱密码">
              <el-input
                v-model="emailForm.email_password"
                type="password"
                placeholder="邮箱密码或授权码"
                show-password
              />
            </el-form-item>
            <el-form-item label="发件人名称">
              <el-input v-model="emailForm.sender_name" placeholder="发件人显示名称" />
            </el-form-item>
            <el-form-item label="加密方式">
              <el-select v-model="emailForm.smtp_encryption" placeholder="选择加密方式">
                <el-option label="TLS (推荐)" value="tls" />
                <el-option label="SSL" value="ssl" />
                <el-option label="无加密" value="none" />
              </el-select>
            </el-form-item>
            <el-form-item label="发件人邮箱">
              <el-input v-model="emailForm.from_email" placeholder="发件人邮箱地址" />
            </el-form-item>
            <el-form-item class="email-buttons-group">
              <el-button type="primary" @click="saveEmailConfig" :loading="emailLoading" class="email-action-btn">
                保存邮件配置
              </el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="订阅访问控制" name="subscriptionAccess">
          <el-form
            :model="subscriptionAccessForm"
            label-width="180px"
            class="subscription-access-form"
          >
            <el-alert
              class="subscription-access-alert"
              title="开启后，浏览器直接打开订阅地址会返回空内容；代理客户端仍可正常订阅。该判断基于 User-Agent，主要用于防止普通浏览器查看内容。"
              type="info"
              show-icon
              :closable="false"
            />
            <el-form-item label="浏览器访问返回空">
              <el-switch
                v-model="subscriptionAccessForm.block_browser_subscription_access"
                active-text="开启"
                inactive-text="关闭"
              />
            </el-form-item>
            <el-form-item class="config-buttons-group">
              <el-button
                type="primary"
                @click="saveSubscriptionAccessConfig"
                :loading="subscriptionAccessLoading"
                class="config-action-btn"
              >
                保存访问控制
              </el-button>
              <el-button @click="loadSubscriptionAccessConfig" class="config-action-btn">
                重新加载
              </el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>

      <el-dialog v-model="searchDialog.visible" :title="searchDialog.title" width="640px">
        <el-table :data="searchDialog.list" size="small" max-height="360" border>
          <el-table-column prop="file_name" label="文件名" min-width="240" show-overflow-tooltip />
          <el-table-column prop="size_text" label="大小" width="110" />
          <el-table-column prop="update_at" label="更新时间" width="170" />
        </el-table>
        <template #footer>
          <el-button @click="searchDialog.visible = false">关闭</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="qrDialog.visible" title="123云盘扫码登录" width="340px" :close-on-click-modal="false">
        <div style="text-align: center">
          <p class="pan-soft-key" style="margin-bottom: 8px">使用 123云盘 手机 App（或微信）扫码，并在手机上确认登录</p>
          <img v-if="qrDialog.qrDataUrl" :src="qrDialog.qrDataUrl" alt="登录二维码" style="width: 200px; height: 200px" />
          <div v-else class="pan-soft-key">二维码生成中...</div>
          <p class="pan-test-result" style="margin-top: 8px">{{ qrDialog.statusText }}</p>
        </div>
        <template #footer>
          <el-button @click="stopQrPolling">关闭</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="refreshReport.visible" title="填充结果" width="720px">
        <el-alert
          v-if="refreshReport.dynamic"
          type="success"
          show-icon
          :closable="false"
          title="已写入动态标记 pan://，用户点击下载时将实时获取最新链接，链接永不失效"
          style="margin-bottom: 12px"
        />
        <el-table :data="refreshReport.list" size="small" max-height="420" border>
          <el-table-column prop="key" label="配置键" width="200" show-overflow-tooltip />
          <el-table-column prop="keyword" label="关键词" width="110" />
          <el-table-column label="状态" width="90" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.ok" type="success" size="small">成功</el-tag>
              <el-tag v-else type="danger" size="small">失败</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="file_name" label="匹配文件" min-width="150" show-overflow-tooltip />
          <el-table-column prop="size_text" label="大小" width="95" />
          <el-table-column prop="error" label="错误" min-width="130" show-overflow-tooltip />
        </el-table>
        <template #footer>
          <el-button @click="refreshReport.visible = false">关闭</el-button>
        </template>
      </el-dialog>
    </el-card>
  </div>
</template>
<script>
import { ref, reactive, onMounted, nextTick } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { configAPI, softwareConfigAPI, pan123API } from '@/utils/api'
import { createQRCodeDataURL } from '@/utils/qrcode'
export default {
  name: 'AdminConfig',
  setup() {
    const activeTab = ref('software')
    const emailLoading = ref(false)
    const softwareLoading = ref(false)
    const subscriptionAccessLoading = ref(false)
    const emailForm = reactive({
      smtp_host: '',
      smtp_port: 587,
      email_username: '',
      email_password: '',
      sender_name: '',
      smtp_encryption: 'tls',
      from_email: ''
    })
    const softwareForm = reactive({
      clash_windows_url: '',
      v2rayn_url: '',
      clash_party_windows_url: '',
      clash_verge_windows_url: '',
      hiddify_windows_url: '',
      flash_windows_url: '',
      clash_android_url: '',
      v2rayng_url: '',
      hiddify_android_url: '',
      flash_macos_url: '',
      flash_macos_arm_url: '',
      flash_android_url: '',
      clash_party_macos_url: '',
      clash_party_macos_arm_url: '',
      clash_verge_macos_url: '',
      clash_verge_macos_arm_url: '',
      v2rayn_macos_url: '',
      v2rayn_macos_arm_url: '',
      hiddify_macos_url: '',
      hiddify_macos_arm_url: '',
      shadowrocket_url: ''
    })
    const subscriptionAccessForm = reactive({
      block_browser_subscription_access: false
    })
    const PAN_SOFTWARE_KEYS = [
      { key: 'clash_windows_url', label: 'Clash for Windows' },
      { key: 'v2rayn_url', label: 'V2rayN (Win)' },
      { key: 'clash_party_windows_url', label: 'Clash Part (Win)' },
      { key: 'clash_verge_windows_url', label: 'Clash Verge (Win)' },
      { key: 'hiddify_windows_url', label: 'Hiddify (Win)' },
      { key: 'flash_windows_url', label: 'FlClash (Win)' },
      { key: 'clash_android_url', label: 'Clash Meta (Android)' },
      { key: 'v2rayng_url', label: 'V2rayNG (Android)' },
      { key: 'hiddify_android_url', label: 'Hiddify (Android)' },
      { key: 'flash_android_url', label: 'FlClash (Android)' },
      { key: 'flash_macos_url', label: 'FlClash (macOS Intel)' },
      { key: 'flash_macos_arm_url', label: 'FlClash (macOS Apple)' },
      { key: 'clash_party_macos_url', label: 'Clash Part (macOS Intel)' },
      { key: 'clash_party_macos_arm_url', label: 'Clash Part (macOS Apple)' },
      { key: 'clash_verge_macos_url', label: 'Clash Verge (macOS Intel)' },
      { key: 'clash_verge_macos_arm_url', label: 'Clash Verge (macOS Apple)' },
      { key: 'v2rayn_macos_url', label: 'V2rayN (macOS Intel)' },
      { key: 'v2rayn_macos_arm_url', label: 'V2rayN (macOS Apple)' },
      { key: 'hiddify_macos_url', label: 'Hiddify (macOS Intel)' },
      { key: 'hiddify_macos_arm_url', label: 'Hiddify (macOS Apple)' },
      { key: 'shadowrocket_url', label: 'Shadowrocket (iOS)' }
    ]
    const panForm = reactive({
      username: '',
      password: '',
      share_pwd: '',
      mode: 'direct',
      cookies: '',
      token: '',
      aliyun_refresh_token: '',
      sync_enabled: true,
      sync_interval_hours: 12
    })
    const panMap = reactive({})
    const panExts = reactive({})
    const PAN_EXT_OPTIONS = ['exe', 'msi', 'dmg', 'pkg', 'apk', 'zip', '7z']
    const panMapRows = PAN_SOFTWARE_KEYS
    const panSaving = ref(false)
    const panTesting = ref(false)
    const panTestingKey = ref('')
    const panRefreshing = ref(false)
    const panTestResult = ref('')
    const searchDialog = reactive({ visible: false, title: '搜索结果', list: [] })
    const refreshReport = reactive({ visible: false, dynamic: true, list: [] })
    const saveSoftwareConfig = async () => {
      softwareLoading.value = true
      try {
        await softwareConfigAPI.updateSoftwareConfig(softwareForm)
        ElMessage.success('软件配置保存成功')
      } catch (error) {
        ElMessage.error('保存失败')
      } finally {
        softwareLoading.value = false
      }
    }
    const loadSoftwareConfig = async () => {
      try {
        const response = await softwareConfigAPI.getSoftwareConfig()
        if (response.data && response.data.success) {
          Object.assign(softwareForm, response.data.data)
        }
      } catch (error) {
        ElMessage.error('加载失败')
      }
    }
    const saveEmailConfig = async () => {
      emailLoading.value = true
      try {
        const emailConfigData = {
          smtp_host: emailForm.smtp_host,
          smtp_port: typeof emailForm.smtp_port === 'number' ? emailForm.smtp_port : Number(emailForm.smtp_port) || 587,
          email_username: emailForm.email_username,
          email_password: (emailForm.email_password && emailForm.email_password !== '******') 
            ? emailForm.email_password 
            : undefined, // 不发送掩码，让后端保持原值
          sender_name: emailForm.sender_name,
          smtp_encryption: emailForm.smtp_encryption,
          from_email: emailForm.from_email
        }
        const response = await configAPI.saveEmailConfig(emailConfigData)
        if (response.data && response.data.success) {
          ElMessage.success('邮件配置保存成功')
          await loadEmailConfig()
        } else {
          ElMessage.error(response.data?.message || '保存失败')
        }
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '保存失败')
      } finally {
        emailLoading.value = false
      }
    }
    const loadEmailConfig = async () => {
      try {
        const response = await configAPI.getEmailConfig()
        if (response.data && response.data.success) {
          const configData = response.data.data
          emailForm.smtp_host = configData.smtp_host || ''
          const port = configData.smtp_port
          emailForm.smtp_port = port ? Number(port) : 587
          emailForm.email_username = configData.email_username || configData.smtp_username || ''
          if (configData.email_password || configData.smtp_password) {
            const passwordValue = configData.email_password || configData.smtp_password
            if (passwordValue && passwordValue.length > 0 && !passwordValue.startsWith('*')) {
              emailForm.email_password = '******'
            } else {
              emailForm.email_password = passwordValue || ''
            }
          } else {
            emailForm.email_password = ''
          }
          emailForm.sender_name = configData.sender_name || ''
          emailForm.smtp_encryption = configData.smtp_encryption || 'tls'
          emailForm.from_email = configData.from_email || ''
        }
      } catch (error) {
        ElMessage.error('加载邮件配置失败')
      }
    }
    const loadSubscriptionAccessConfig = async () => {
      try {
        const response = await configAPI.getSystemConfigs({ category: 'subscription_access' })
        if (response.data && response.data.success) {
          const configList = response.data.data || []
          const configMap = {}
          configList.forEach(item => {
            configMap[item.key] = item.value
          })
          subscriptionAccessForm.block_browser_subscription_access = configMap.block_browser_subscription_access === 'true'
        }
      } catch (error) {
        ElMessage.error('加载订阅访问控制失败')
      }
    }
    const saveSubscriptionAccessConfig = async () => {
      subscriptionAccessLoading.value = true
      try {
        const response = await configAPI.updateSystemConfig('block_browser_subscription_access', {
          key: 'block_browser_subscription_access',
          value: subscriptionAccessForm.block_browser_subscription_access.toString(),
          category: 'subscription_access',
          type: 'boolean',
          display_name: '浏览器访问订阅返回空'
        })
        if (response.data && response.data.success) {
          ElMessage.success('订阅访问控制保存成功')
        } else {
          ElMessage.error(response.data?.message || '保存失败')
        }
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '保存失败')
      } finally {
        subscriptionAccessLoading.value = false
      }
    }
    const loadPanConfig = async () => {
      try {
        const response = await pan123API.getConfig()
        if (response.data && response.data.success) {
          const data = response.data.data || {}
          panForm.username = data.username || ''
          panForm.password = data.password || ''
          panForm.share_pwd = data.share_pwd || ''
          panForm.mode = data.mode || 'share'
          panForm.cookies = data.cookies || ''
          panForm.token = data.token || ''
          panForm.aliyun_refresh_token = data.aliyun_refresh_token || ''
          panForm.sync_enabled = data.sync_enabled !== false
          panForm.sync_interval_hours = data.sync_interval_hours || 12
          const daysLeft = Number(data.token_days_left || 0)
          if (data.token_expiry && !isNaN(daysLeft)) {
            tokenExpiryText.value = daysLeft >= 0 ? `Token 剩余 ${daysLeft} 天（${(data.token_expiry || '').replace('T', ' ').slice(0, 10)} 到期）` : `Token 已过期，请重新短信登录`
            tokenExpiryDanger.value = daysLeft < 7
          } else {
            tokenExpiryText.value = ''
          }
          const map = data.file_map || {}
          PAN_SOFTWARE_KEYS.forEach(({ key }) => {
            panMap[key] = map[key] || ''
          })
          // 已同步的软件自动回填关键词（取云盘文件名），避免输入框为空
          const syncedFiles = data.synced_files || {}
          PAN_SOFTWARE_KEYS.forEach(({ key }) => {
            if (!panMap[key] && syncedFiles[key]) {
              panMap[key] = syncedFiles[key]
            }
          })
          const extMap = data.file_exts || {}
          PAN_SOFTWARE_KEYS.forEach(({ key }) => {
            panExts[key] = extMap[key] || ''
          })
        }
      } catch (error) {
        // 未配置时静默
      }
    }
    // persistPanConfig 把当前表单（凭证 + 关键词 + 扩展名）保存到后端，不弹提示
    const persistPanConfig = async () => {
      try {
        const response = await pan123API.saveConfig({
          ...panForm,
          file_map: { ...panMap },
          file_exts: { ...panExts }
        })
        return !!(response.data && response.data.success)
      } catch (error) {
        return false
      }
    }
    const savePanConfig = async () => {
      panSaving.value = true
      try {
        const ok = await persistPanConfig()
        if (ok) {
          ElMessage.success('云盘配置已保存')
          await loadPanConfig()
        } else {
          ElMessage.error('保存失败，请检查网络或配置')
        }
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '保存失败')
      } finally {
        panSaving.value = false
      }
    }
    const testPanConfig = async () => {
      panTesting.value = true
      panTestResult.value = ''
      try {
        const response = await pan123API.test()
        if (response.data && response.data.success) {
          const data = response.data.data || {}
          const modeText = data.mode === 'direct' ? '直链' : '分享链接'
          panTestResult.value = `连接成功：${data.nickname || '已登录'}（当前模式：${modeText}）`
        } else {
          panTestResult.value = `失败：${response.data?.message || '未知错误'}`
        }
      } catch (error) {
        panTestResult.value = `失败：${error.response?.data?.message || error.message || '网络错误'}`
      } finally {
        panTesting.value = false
      }
    }
    const isPanDynamic = (key) => String(softwareForm[key] || '').startsWith('pan://')
    // 当前配置列：超长链接截断显示，悬停查看完整
    const shortUrl = (key) => {
      const v = String(softwareForm[key] || '').trim()
      if (v.length <= 24) return v
      return v.slice(0, 24) + '…'
    }
    // ---- 阿里云盘连接测试 ----
    const aliyunTesting = ref(false)
    const aliyunTestResult = ref('')
    const testAliyun = async () => {
      aliyunTesting.value = true
      aliyunTestResult.value = ''
      try {
        const saved = await persistPanConfig()
        if (!saved) {
          aliyunTestResult.value = '保存配置失败'
          return
        }
        const response = await pan123API.aliyunTest()
        if (response.data?.success) {
          const d = response.data.data || {}
          aliyunTestResult.value = `连接成功（根目录 ${d.root_files} 个文件）${d.refresh_token_rotated ? '，token 已自动轮换' : ''}`
        } else {
          aliyunTestResult.value = response.data?.message || '连接失败'
        }
      } catch (error) {
        aliyunTestResult.value = error.response?.data?.message || '连接失败'
      } finally {
        aliyunTesting.value = false
      }
    }
    // ---- 短信验证码登录（海外服务器风控场景）----
    const smsMode = ref(false)
    const panSmsCode = ref('')
    const panSmsSending = ref(false)
    const panSmsLogging = ref(false)
    const smsStatus = ref('')
    const smsHashCode = ref('')
    const smsTimeStamp = ref('')
    const smsNeedsCaptcha = ref(false)
    let panelCaptchaInstance = null
    // ---- 二维码扫码登录 ----
    const qrLoading = ref(false)
    const qrDialog = reactive({ visible: false, qrDataUrl: '', statusText: '等待扫码...' })
    let qrPollTimer = null
    let qrPolling = false
    const qrLogin = async () => {
      qrLoading.value = true
      qrDialog.visible = true
      qrDialog.statusText = '正在生成二维码...'
      qrDialog.qrDataUrl = ''
      try {
        const response = await pan123API.qrGenerate()
        if (response.data?.success) {
          const data = response.data.data || {}
          qrDialog.qrDataUrl = await createQRCodeDataURL(data.qr_value, { width: 200, margin: 2 })
          qrDialog.statusText = '等待扫码...'
          startQrPolling(data.uni_id)
        } else {
          qrDialog.statusText = response.data?.message || '生成二维码失败'
        }
      } catch (error) {
        qrDialog.statusText = error.response?.data?.message || '生成二维码失败'
      } finally {
        qrLoading.value = false
      }
    }
    const startQrPolling = (uniId) => {
      stopQrPolling()
      qrPolling = true
      const poll = async () => {
        if (!qrPolling) return
        try {
          const response = await pan123API.qrStatus({ uni_id: uniId })
          const status = response.data?.data?.status || 'waiting'
          if (status === 'waiting') {
            qrDialog.statusText = '等待扫码...'
          } else if (status === 'scanned') {
            qrDialog.statusText = '已扫码，请在手机上确认'
          } else if (status === 'success') {
            qrDialog.statusText = '登录成功，token 已保存'
            stopQrPolling()
            setTimeout(() => { qrDialog.visible = false }, 1200)
            ElMessage.success('登录成功，token 已自动保存')
            await loadPanConfig()
            return
          } else if (status === 'expired' || status === 'rejected') {
            qrDialog.statusText = response.data?.data?.message || '扫码已失效'
            stopQrPolling()
            return
          }
        } catch (error) {
          qrDialog.statusText = '查询状态失败，重试中...'
        }
        qrPollTimer = setTimeout(poll, 2000)
      }
      poll()
    }
    const stopQrPolling = () => {
      qrPolling = false
      if (qrPollTimer) {
        clearTimeout(qrPollTimer)
        qrPollTimer = null
      }
    }

    // 加载 123pan 同款阿里云验证码 SDK（含其内置 appKey）
    const loadCaptchaScripts = () => {
      return new Promise((resolve) => {
        if (window.initAliyunCaptcha) return resolve(true)
        const s = document.createElement('script')
        s.src = 'https://o.alicdn.com/captcha-frontend/aliyunCaptcha/AliyunCaptcha.js'
        s.onload = () => {
          const s2 = document.createElement('script')
          s2.src = 'https://statics.123957.com/static/sliderAli.js'
          s2.onload = () => resolve(true)
          s2.onerror = () => resolve(true)
          document.head.appendChild(s2)
        }
        s.onerror = () => resolve(false)
        document.head.appendChild(s)
      })
    }

    // 初始化面板内嵌滑块（场景与 123pan 登录中心一致）
    const initPanelCaptcha = async () => {
      const ok = await loadCaptchaScripts()
      if (!ok || !window.initAliyunCaptcha) {
        smsStatus.value = '滑块组件加载失败，请检查网络后重试'
        return
      }
      window.AliyunCaptchaConfig = { region: 'cn', prefix: '1gkapk' }
      await nextTick()
      try {
        window.initAliyunCaptcha({
          SceneId: 'unts8f61',
          mode: 'embed',
          element: '#pan-captcha',
          captchaVerifyCallback: async (traceless) => {
            // 滑块完成 → 带 traceless 重发验证码（服务器发码）
            try {
              const response = await pan123API.smsSend({
                username: panForm.username,
                password: panForm.password,
                traceless
              })
              if (response.data?.success) {
                const d = response.data.data || {}
                if (d.needs_captcha) {
                  smsStatus.value = '滑块验证未通过，请重新滑动'
                  return { captchaResult: false }
                }
                smsHashCode.value = d.hash_code || smsHashCode.value
                smsTimeStamp.value = d.time_stamp || ''
                smsNeedsCaptcha.value = false
                smsStatus.value = '验证码已发送，请查收短信'
                return { captchaResult: true }
              }
              smsStatus.value = response.data?.message || '发送失败'
              return { captchaResult: false }
            } catch (error) {
              smsStatus.value = error.response?.data?.message || '发送失败'
              return { captchaResult: false }
            }
          },
          getInstance: (inst) => { panelCaptchaInstance = inst || null },
          onError: (err) => { console.error('滑块初始化失败', err); smsStatus.value = '滑块初始化失败，请刷新重试' },
          slideStyle: { width: 320, height: 40 },
          language: 'cn'
        })
      } catch (error) {
        console.error('init captcha error', error)
        smsStatus.value = '滑块初始化失败，请刷新重试'
      }
    }
    const tokenExpiryText = ref('')
    const tokenExpiryDanger = ref(false)
    const smsSend = async () => {
      panSmsSending.value = true
      smsStatus.value = ''
      try {
        const response = await pan123API.smsSend({
          username: panForm.username,
          password: panForm.password
        })
        if (response.data?.success) {
          const data = response.data.data || {}
          if (data.need_sms === false) {
            smsStatus.value = '账号密码登录成功，无需短信验证'
            return
          }
          smsHashCode.value = data.hash_code || ''
          smsTimeStamp.value = data.time_stamp || ''
          smsNeedsCaptcha.value = !!data.needs_captcha
          smsMode.value = true
          smsStatus.value = response.data.message || (data.needs_captcha ? '请完成下方滑块验证' : '验证码已发送')
          if (data.needs_captcha) {
            await initPanelCaptcha()
          }
        } else {
          smsStatus.value = response.data?.message || '发送失败'
        }
      } catch (error) {
        smsStatus.value = error.response?.data?.message || '发送失败'
      } finally {
        panSmsSending.value = false
      }
    }
    const smsLogin = async () => {
      if (!panSmsCode.value.trim()) {
        ElMessage.warning('请输入验证码')
        return
      }
      panSmsLogging.value = true
      smsStatus.value = ''
      try {
        const response = await pan123API.smsLogin({
          username: panForm.username,
          password: panForm.password,
          hash_code: smsHashCode.value,
          sms_code: panSmsCode.value.trim(),
          time_stamp: smsTimeStamp.value
        })
        if (response.data?.success) {
          ElMessage.success('登录成功，token 已自动保存')
          smsMode.value = false
          smsNeedsCaptcha.value = false
          smsTimeStamp.value = ''
          panSmsCode.value = ''
          smsStatus.value = '登录成功，token 已保存'
          await loadPanConfig()
        } else {
          smsStatus.value = response.data?.message || '登录失败'
        }
      } catch (error) {
        smsStatus.value = error.response?.data?.message || '登录失败'
      } finally {
        panSmsLogging.value = false
      }
    }
    const testPanKeyword = async (row) => {
      const keyword = String(panMap[row.key] || '').trim()
      if (!keyword) {
        ElMessage.warning('请先填写关键词')
        return
      }
      panTestingKey.value = row.key
      try {
        const response = await pan123API.search({ keyword, ext: panExts[row.key] || '' })
        if (response.data && response.data.success) {
          const list = response.data.data?.list || []
          searchDialog.list = list
          searchDialog.title = `“${keyword}” 搜索结果（共 ${list.length} 个）`
          searchDialog.visible = true
          if (!list.length) {
            ElMessage.warning('未找到匹配文件，请调整关键词')
          }
        } else {
          ElMessage.error(response.data?.message || '搜索失败')
        }
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '搜索失败')
      } finally {
        panTestingKey.value = ''
      }
    }
    const panRefresh = async (dynamic) => {
      panRefreshing.value = true
      try {
        // 先把当前表格里的关键词/扩展名和凭证落库，避免刷新读到旧配置（旧配置关键词为空导致 0 个软件）
        const saved = await persistPanConfig()
        if (!saved) {
          ElMessage.error('保存云盘配置失败，无法执行一键生成')
          return
        }
        const response = await pan123API.refresh({
          dynamic,
          mode: panForm.mode,
          file_map: { ...panMap },
          file_exts: { ...panExts }
        })
        if (response.data && response.data.success) {
          const data = response.data.data || {}
          ElMessage.success(response.data.message || '填充完成')
          await loadSoftwareConfig()
          refreshReport.list = data.report || []
          refreshReport.dynamic = data.dynamic !== false
          refreshReport.visible = true
        } else {
          ElMessage.error(response.data?.message || '填充失败')
        }
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '填充失败')
      } finally {
        panRefreshing.value = false
      }
    }
    const panSyncing = ref(false)
    const panVersions = ref([])
    const syncStatusText = ref('')
    let syncPollTimer = null
    const loadPanSyncStatus = async () => {
      try {
        const [statusRes, versionsRes] = await Promise.all([
          pan123API.syncStatus(),
          pan123API.versions()
        ])
        const status = statusRes.data?.data || {}
        if (versionsRes.data?.success) {
          panVersions.value = versionsRes.data.data?.list || []
        }
        if (status.running) {
          syncStatusText.value = '同步进行中，请稍候…'
          clearTimeout(syncPollTimer)
          syncPollTimer = setTimeout(loadPanSyncStatus, 5000)
          return
        }
        clearTimeout(syncPollTimer)
        if (status.last_run) {
          const total = status.total_uploaded || 0
          syncStatusText.value = `上次运行：${status.last_run.replace('T', ' ').slice(0, 19)}，本次上传 ${total} 个文件`
        } else {
          syncStatusText.value = '尚未运行过同步'
        }
        // 同步完成后刷新版本对照与软件配置
        if (versionsRes.data?.success) {
          panVersions.value = versionsRes.data.data?.list || []
        }
        await loadSoftwareConfig()
      } catch (error) {
        syncStatusText.value = '获取同步状态失败'
      }
    }
    const runPanSync = async () => {
      panSyncing.value = true
      try {
        const saved = await persistPanConfig()
        if (!saved) {
          ElMessage.error('保存云盘配置失败，无法执行同步')
          return
        }
        const response = await pan123API.sync()
        if (response.data?.success) {
          ElMessage.success('同步已开始（首次可能需下载上传多个安装包，耗时较长）')
          syncStatusText.value = '同步进行中，请稍候…'
          clearTimeout(syncPollTimer)
          syncPollTimer = setTimeout(loadPanSyncStatus, 5000)
        } else {
          ElMessage.error(response.data?.message || '触发同步失败')
        }
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '触发同步失败')
      } finally {
        panSyncing.value = false
      }
    }
    onMounted(() => {
      loadEmailConfig()
      loadSoftwareConfig()
      loadSubscriptionAccessConfig()
      loadPanConfig()
      loadPanSyncStatus()
    })
    return {
      activeTab,
      emailLoading,
      softwareLoading,
      subscriptionAccessLoading,
      emailForm,
      softwareForm,
      subscriptionAccessForm,
      saveSoftwareConfig,
      loadSoftwareConfig,
      saveEmailConfig,
      loadEmailConfig,
      loadSubscriptionAccessConfig,
      saveSubscriptionAccessConfig,
      panForm,
      panMap,
      panExts,
      PAN_EXT_OPTIONS,
      panMapRows,
      panSaving,
      panTesting,
      panTestingKey,
      panRefreshing,
      panTestResult,
      searchDialog,
      refreshReport,
      loadPanConfig,
      savePanConfig,
      testPanConfig,
      testPanKeyword,
      isPanDynamic,
      shortUrl,
      panRefresh,
      panSyncing,
      panVersions,
      syncStatusText,
      loadPanSyncStatus,
      runPanSync,
      smsMode,
      panSmsCode,
      panSmsSending,
      panSmsLogging,
      smsStatus,
      smsNeedsCaptcha,
      tokenExpiryText,
      tokenExpiryDanger,
      smsSend,
      smsLogin,
      aliyunTesting,
      aliyunTestResult,
      testAliyun,
      qrLoading,
      qrDialog,
      qrLogin,
      stopQrPolling
    }
  }
}
</script>
<style scoped>
.config-admin-container {
  padding: 20px;
}
.admin-config > :deep(.el-card) {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  box-shadow: none;
}
.admin-config > :deep(.el-card > .el-card__header) {
  background: var(--el-fill-color-extra-light);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.admin-config :deep(.el-tabs) {
  border-radius: 8px;
  box-shadow: none;
  overflow: hidden;
}
.admin-config :deep(.el-tabs__header) {
  background: var(--el-fill-color-extra-light);
}
.admin-config :deep(.el-input__wrapper),
.admin-config :deep(.el-select .el-input__wrapper),
.admin-config :deep(.el-input-number .el-input__wrapper) {
  min-height: 44px;
  touch-action: manipulation;
}
.admin-config :deep(.el-button) {
  touch-action: manipulation;
}
.config-section {
  margin-bottom: 30px;
}
.config-section h3 {
  color: #333;
  margin-bottom: 20px;
  font-size: 1.2rem;
}
.pan-test-result {
  margin-left: 12px;
  font-size: 13px;
  color: var(--el-color-success);
}
.pan-mapping-title {
  margin: 16px 0 8px;
  color: #333;
  font-size: 1rem;
}
.pan-tip {
  margin: 0 0 10px;
  font-size: 12px;
  color: #909399;
}
.pan-soft-name {
  font-weight: 600;
}
.pan-soft-key {
  font-size: 12px;
  color: #909399;
}
.pan-current-value {
  margin-top: 2px;
  max-width: 150px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: default;
}
.avatar-uploader {
  text-align: center;
}
.avatar-uploader .avatar {
  width: 100px;
  height: 100px;
  border-radius: 6px;
}
.avatar-uploader .el-upload {
  border: 1px dashed #d9d9d9;
  border-radius: 6px;
  cursor: pointer;
  position: relative;
  overflow: clip;
  width: 100px;
  height: 100px;
  display: flex;
  align-items: center;
  justify-content: center;
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
.backup-section {
  margin-bottom: 30px;
}
.backup-section h3 {
  color: #333;
  margin-bottom: 20px;
  font-size: 1.2rem;
}
.backup-section .el-button {
  margin-right: 15px;
  margin-bottom: 15px;
}
.email-queue-section {
  padding: 20px;
}
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.section-header h3 {
  margin: 0;
  color: #333;
  font-size: 1.2rem;
}
.queue-stats {
  margin-bottom: 20px;
}
.stat-card {
  text-align: center;
}
.stat-number {
  font-size: 1.8rem;
  font-weight: bold;
  color: #333;
}
.stat-label {
  font-size: 0.9rem;
  color: #666;
  margin-top: 5px;
}
.queue-filter {
  margin-bottom: 20px;
}
@media (max-width: 768px) {
  .config-admin-container {
    padding: 10px;
    width: 100%;
    box-sizing: border-box;
  }
  .admin-config {
    padding: 10px;
  }
  .admin-config :deep(.el-tabs__content) {
    padding: 12px !important;
  }
  .admin-config :deep(.el-tabs__item) {
    min-height: 44px;
    display: inline-flex;
    align-items: center;
  }
  .admin-config :deep(.el-input__wrapper),
  .admin-config :deep(.el-select .el-input__wrapper),
  .admin-config :deep(.el-input-number .el-input__wrapper) {
    min-height: 44px;
  }
  .card-header {
    flex-direction: column;
    gap: 10px;
    align-items: flex-start;
  }
  .backup-section .el-button {
    width: 100%;
    margin-right: 0;
    margin-bottom: 10px;
    box-sizing: border-box;
  }
  .header-actions {
    flex-direction: column;
    gap: 10px;
    width: 100%;
    .el-button {
      width: 100%;
      box-sizing: border-box;
    }
  }
  .admin-config :deep(.el-form) {
    width: 100% !important;
    box-sizing: border-box;
    .el-form-item {
      width: 100% !important;
      margin-bottom: 20px;
      display: flex;
      flex-direction: column;
      box-sizing: border-box;
      .el-form-item__label {
        width: 100% !important;
        text-align: left;
        margin-bottom: 8px;
        padding: 0;
        font-weight: 600;
        color: #1e293b;
        font-size: 0.95rem;
        box-sizing: border-box;
      }
      .el-form-item__content {
        width: 100% !important;
        margin-left: 0 !important;
        box-sizing: border-box;
        .el-input,
        .el-input-number,
        .el-select,
        .el-textarea,
        .el-input__wrapper {
          width: 100% !important;
          max-width: 100% !important;
          box-sizing: border-box;
        }
        .el-button:not(.email-action-btn) {
          width: 100% !important;
          min-width: 100% !important;
          min-height: 44px;
          box-sizing: border-box;
          margin-bottom: 10px;
          margin-right: 0 !important;
        }
        .el-button:not(.email-action-btn):last-child {
          margin-bottom: 0;
        }
        .el-input__inner {
          width: 100% !important;
          box-sizing: border-box;
        }
        .el-input-number {
          width: 100% !important;
        }
        .el-input-number .el-input__wrapper {
          width: 100% !important;
        }
        .el-select {
          width: 100% !important;
        }
        .el-select .el-input__wrapper {
          width: 100% !important;
        }
        .el-textarea {
          width: 100% !important;
        }
        .el-textarea .el-textarea__inner {
          width: 100% !important;
          box-sizing: border-box;
        }
      }
    }
  }
  .admin-config :deep(.el-tabs__content) {
    width: 100% !important;
    box-sizing: border-box;
  }
  .admin-config :deep(.el-card__body) {
    width: 100% !important;
    padding: 12px !important;
    box-sizing: border-box;
  }
  .admin-config :deep(.el-table) {
    width: 100% !important;
    box-sizing: border-box;
  }
  .admin-config :deep(.el-row) {
    .el-col {
      width: 100% !important;
      max-width: 100% !important;
      flex: 0 0 100% !important;
      margin-bottom: 12px;
      box-sizing: border-box;
    }
    .el-col .el-form-item {
      margin-bottom: 0;
    }
  }
}
.email-config-form {
  @media (max-width: 768px) {
    :deep(.el-form-item.email-buttons-group) {
      width: 100% !important;
      max-width: 100% !important;
      margin: 0 !important;
      padding: 0 !important;
      display: block !important;
    }
  }
}
.email-buttons-group {
  width: 100% !important;
  max-width: 100% !important;
  box-sizing: border-box !important;
  margin: 0 !important;
  padding: 0 !important;
  :deep(.el-form-item__label) {
    display: none !important;
  }
  :deep(.el-form-item__content) {
    display: flex !important;
    gap: 0 !important;
    flex-wrap: nowrap !important;
    align-items: stretch !important;
    justify-content: flex-start !important;
    width: 100% !important;
    max-width: 100% !important;
    box-sizing: border-box !important;
    margin-left: 0 !important;
    margin-right: 0 !important;
    padding: 0 !important;
    @media (min-width: 769px) {
      flex-direction: row !important;
      gap: 10px !important;
      .email-action-btn {
        flex: 1 1 0 !important;
        min-width: 160px !important;
        max-width: 220px !important;
        box-sizing: border-box !important;
      }
    }
    @media (max-width: 768px) {
      flex-direction: column !important;
      width: 100% !important;
      max-width: 100% !important;
      align-items: stretch !important;
      gap: 10px !important;
    }
  }
  @media (max-width: 768px) {
    :deep(.el-button.email-action-btn) {
      width: 100% !important;
      min-width: 100% !important;
      max-width: 100% !important;
      min-height: 44px !important;
      display: block !important;
      box-sizing: border-box !important;
      margin: 0 !important;
      padding: 12px 20px !important;
      flex: none !important;
      align-self: stretch !important;
      border-radius: 4px !important;
      position: relative !important;
    }
    :deep(.el-button.email-action-btn:not(:last-child)) {
      margin-bottom: 10px !important;
    }
    :deep(.el-button.email-action-btn:last-child) {
      margin-bottom: 0 !important;
    }
    :deep(.el-button.email-action-btn > span) {
      width: 100% !important;
      display: block !important;
      text-align: center !important;
      box-sizing: border-box !important;
    }
    :deep(.el-button.email-action-btn .el-icon) {
      margin-right: 8px !important;
    }
    :deep(.el-button.email-action-btn),
    :deep(.el-button.email-action-btn *) {
      max-width: 100% !important;
    }
  }
}
.config-buttons-group {
  :deep(.el-form-item__content) {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
    align-items: stretch;
    justify-content: flex-start;
    width: 100%;
    box-sizing: border-box;
    @media (min-width: 769px) {
      .config-action-btn {
        flex: 1 1 0;
        min-width: 140px;
        max-width: 200px;
        box-sizing: border-box;
      }
    }
    @media (max-width: 768px) {
      flex-direction: column;
      width: 100%;
      .config-action-btn {
        width: 100% !important;
        min-width: 100% !important;
        max-width: 100% !important;
        min-height: 44px;
        margin-bottom: 10px;
        margin-right: 0 !important;
        margin-left: 0 !important; /* 覆盖 el-button+el-button 默认 margin-left:12px，保证纵向对齐 */
        box-sizing: border-box;
      }
      .config-action-btn:last-child {
        margin-bottom: 0;
      }
    }
  }
}
.subscription-access-form {
  max-width: 760px;
}
.subscription-access-alert {
  margin-bottom: 20px;
}
.payment-form .el-divider {
  margin: 30px 0 20px 0;
}
.payment-form .el-divider:first-child {
  margin-top: 0;
}
</style> 
