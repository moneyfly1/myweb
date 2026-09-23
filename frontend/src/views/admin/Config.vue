<template>
  <div class="list-container admin-config">
    <el-card>
  <template #header>
    <div class="card-header">
      <h2>配置管理</h2>
      <p>管理软件下载配置、软件库自动同步和邮件配置</p>
    </div>
  </template>
  <!-- 移动端：3 个较长的 tab 在 390px 下也放不下（实测溢出 54px）→ 下拉导航 -->
  <el-tabs v-model="activeTab" type="border-card" class="tabs-wrap-mobile">
    <el-tab-pane label="软件下载配置" name="software">
      <div class="config-section">
        <el-divider content-position="left">MoneyFly 自研客户端（用户端置顶推荐）</el-divider>
        <el-alert
          type="success"
          show-icon
          :closable="false"
          title="这是您自研的官方客户端，会在用户仪表盘、帮助中心、软件教程页置顶展示并标注「官方自研」推荐。留空的平台不会显示下载按钮；四个都留空时用户端整块隐藏。"
        />
        <el-form
          :model="softwareForm"
          label-width="150px"
          style="margin-top: 16px"
        >
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="Android 安装包">
                <el-input v-model="softwareForm.moneyfly_android_url" placeholder="APK 直链，或 pan://配置键" clearable />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="Windows 安装包">
                <el-input v-model="softwareForm.moneyfly_windows_url" placeholder="exe/msi 直链，或 pan://配置键" clearable />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="macOS（Apple 芯片）">
                <el-input v-model="softwareForm.moneyfly_macos_arm_url" placeholder="M 系列芯片 dmg/pkg 直链" clearable />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="macOS（Intel 芯片）">
                <el-input v-model="softwareForm.moneyfly_macos_url" placeholder="Intel 芯片 dmg/pkg 直链" clearable />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="版本号（可选）">
                <el-input v-model="softwareForm.moneyfly_version" placeholder="例如 2.1.2，展示为 v2.1.2" clearable />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="用户端展示">
                <el-switch
                  v-model="softwareForm.moneyfly_enabled"
                  active-text="展示"
                  inactive-text="隐藏"
                />
              </el-form-item>
            </el-col>
          </el-row>
          <el-form-item label="客户端说明（可选）">
            <el-input
              v-model="softwareForm.moneyfly_note"
              type="textarea"
              :rows="2"
              placeholder="留空则使用默认说明：官方自研客户端 · 一键导入订阅 · 开箱即用"
            />
          </el-form-item>
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
        <el-divider content-position="left">第三方客户端下载链接</el-divider>
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
            <el-divider content-position="left">GitHub 软件库自动同步（定时）</el-divider>
            <el-alert
              type="info"
              show-icon
              :closable="false"
              title="自动检测 v2rayN / Hiddify / Clash Verge / Clash Part / V2rayNG / Clash Meta / FlClash 在 GitHub 的最新版本；用户点击下载时，后端实时获取最新安装包并通过国内加速镜像（ghfast.top 等）302 直链给用户，国内下载快、无需网盘、无需审核。"
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
                <el-button v-if="syncReport.list.length" @click="showSyncReport" class="config-action-btn">
                  查看上次同步报告
                </el-button>
                <span v-if="syncStatusText" class="pan-test-result" style="margin-left: 8px; color: var(--el-color-primary); word-break: break-all">{{ syncStatusText }}</span>
              </el-form-item>
            </el-form>

            <h4 class="pan-mapping-title">版本对照（GitHub 最新版 ↔ 已检出版本）</h4>
            <!-- 7 列合计约 820px，390px 屏幕上只能横向滑动（实测内容超出视口 478px）→
                 窄屏改用卡片（ResponsiveDataView），宽屏保持表格。 -->
            <ResponsiveDataView
              :data="panVersions"
              :fields="panVersionMobileFields"
              id-field="name"
              title-field="name"
              empty-title="暂无版本记录"
            >
              <template #table>
            <el-table :data="panVersions" size="small" border max-height="420">
              <el-table-column prop="name" label="软件" width="130" />
              <el-table-column prop="label" label="平台/架构" width="160" />
              <el-table-column label="GitHub 版本" width="110" align="center">
                <template #default="{ row }">
                  <span v-if="row.github_version">v{{ row.github_version }}</span>
                  <el-tag v-else type="danger" size="small">获取失败</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="已检出版本" width="110" align="center">
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
              <el-table-column prop="file_name" label="安装包文件" min-width="200" show-overflow-tooltip />
            </el-table>
              </template>
              <template #header="{ item }">
                <div class="pan-version-mobile-head">
                  <span class="pan-version-name">{{ item.name }}</span>
                  <el-tag v-if="item.custom" type="warning" size="small">自定义链接</el-tag>
                  <el-tag v-else-if="item.synced" type="success" size="small">已同步</el-tag>
                  <el-tag v-else-if="item.cloud_version" type="warning" size="small">待更新</el-tag>
                  <el-tag v-else type="info" size="small">待上传</el-tag>
                </div>
              </template>
              <template #field-label="{ item }">{{ item.label }}</template>
              <template #field-github_version="{ item }">
                <span v-if="item.github_version">v{{ item.github_version }}</span>
                <el-tag v-else type="danger" size="small">获取失败</el-tag>
              </template>
              <template #field-cloud_version="{ item }">
                <span v-if="item.cloud_version">v{{ item.cloud_version }}</span>
                <span v-else class="pan-soft-key">未上传</span>
              </template>
              <template #field-file_name="{ item }">{{ item.file_name || '-' }}</template>
            </ResponsiveDataView>
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
      </el-tabs>

      <el-dialog v-model="syncReport.visible" title="上次同步报告" width="820px">
        <el-alert
          v-if="syncReport.lastRun"
          type="info"
          show-icon
          :closable="false"
          :title="`运行时间：${syncReport.lastRun.replace('T', ' ').slice(0, 19)}${syncReport.folder}`"
          style="margin-bottom: 12px"
        />
        <el-table :data="syncReport.list" size="small" max-height="420" border>
          <el-table-column prop="name" label="软件" width="110" />
          <el-table-column prop="label" label="平台/架构" width="150" />
          <el-table-column label="状态" width="90" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.status === 'ok'" type="success" size="small">成功</el-tag>
              <el-tag v-else-if="row.status === 'skip'" type="info" size="small">跳过</el-tag>
              <el-tag v-else type="danger" size="small">失败</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="file_name" label="安装包文件" min-width="170" show-overflow-tooltip />
          <el-table-column prop="message" label="说明" min-width="180" show-overflow-tooltip />
        </el-table>
        <template #footer>
          <el-button @click="syncReport.visible = false">关闭</el-button>
        </template>
      </el-dialog>
    </el-card>
  </div>
</template>
<script>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { unwrapList } from '@/utils/format'
import { configAPI, softwareConfigAPI, cloudAPI } from '@/utils/api'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
export default {
  name: 'AdminConfig',
  components: { ResponsiveDataView },
  setup() {
    const activeTab = ref('software')
    // 与下方 el-tab-pane 的 name/label 一一对应（新增 tab 时两处都要加）
        // 窄屏卡片字段（宽屏表格不受影响）
    const panVersionMobileFields = [
      { key: 'label', label: '平台/架构' },
      { key: 'github_version', label: 'GitHub 版本' },
      { key: 'cloud_version', label: '已检出版本' },
      { key: 'file_name', label: '安装包文件' }
    ]
    const emailLoading = ref(false)
    const softwareLoading = ref(false)
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
      // MoneyFly 自研客户端（官方推荐，用户端置顶）
      moneyfly_android_url: '',
      moneyfly_windows_url: '',
      moneyfly_macos_url: '',
      moneyfly_macos_arm_url: '',
      moneyfly_version: '',
      moneyfly_note: '',
      moneyfly_enabled: true,
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
    const panForm = reactive({
      sync_enabled: true,
      sync_interval_hours: 12
    })
    const panSaving = ref(false)
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
          const data = response.data.data || {}
          Object.assign(softwareForm, data)
          // 后端以字符串存储（"true"/"false"/"0"/"1"），需转回布尔，
          // 否则 el-switch 会把字符串 "false" 当成真值显示为开启。
          // 未配置过（缺失/空）时默认开启展示。
          const raw = data.moneyfly_enabled
          const isEmpty = raw === undefined || raw === null || String(raw).trim() === ''
          const isFalsy = raw === false || raw === 0 ||
            ['false', '0', 'off', 'no'].includes(String(raw).trim().toLowerCase())
          softwareForm.moneyfly_enabled = isEmpty ? true : !isFalsy
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
    const loadPanConfig = async () => {
      try {
        const response = await cloudAPI.getConfig()
        if (response.data && response.data.success) {
          const data = response.data.data || {}
          panForm.sync_enabled = data.sync_enabled !== false
          panForm.sync_interval_hours = data.sync_interval_hours || 12
        }
      } catch (error) {
        // 未配置时静默
      }
    }
    const savePanConfig = async () => {
      panSaving.value = true
      try {
        const response = await cloudAPI.saveConfig({
          sync_enabled: panForm.sync_enabled,
          sync_interval_hours: panForm.sync_interval_hours
        })
        if (response.data && response.data.success) {
          ElMessage.success('同步配置已保存')
          await loadPanConfig()
          return true
        } else {
          ElMessage.error(response.data?.message || '保存失败，请检查网络或配置')
          return false
        }
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '保存失败')
        return false
      } finally {
        panSaving.value = false
      }
    }
    const panSyncing = ref(false)
    const panVersions = ref([])
    const syncStatusText = ref('')
    const syncReport = reactive({ visible: false, list: [], folder: '', lastRun: '' })
    const showSyncReport = () => { syncReport.visible = true }
    let syncPollTimer = null
    const loadPanSyncStatus = async () => {
      try {
        const [statusRes, versionsRes] = await Promise.all([
          cloudAPI.syncStatus(),
          cloudAPI.versions()
        ])
        const status = statusRes.data?.data || {}
        if (versionsRes.data?.success) {
          panVersions.value = unwrapList(versionsRes)
        }
        if (status.running) {
          // 实时进度展示
          const p = status.progress || {}
          const done = p.done ?? 0
          const total = p.total ?? 0
          const folder = p.folder ? `（上传目录：${p.folder}）` : ''
          if (p.item && p.stage) {
            syncStatusText.value = `正在同步 ${done}/${total}：${p.item} · ${p.stage}${p.current_file ? ' · ' + p.current_file : ''}${folder}`
          } else {
            syncStatusText.value = `正在同步 ${done}/${total}${folder}，请稍候…`
          }
          clearTimeout(syncPollTimer)
          syncPollTimer = setTimeout(loadPanSyncStatus, 3000)
          return
        }
        clearTimeout(syncPollTimer)
        const folder = status.progress?.folder ? `，上传目录：${status.progress.folder}` : ''
        if (status.last_run) {
          const total = status.total_uploaded || 0
          const ok = (status.last_report || []).filter(r => r.status === 'ok').length
          const err = (status.last_report || []).filter(r => r.status === 'error').length
          syncStatusText.value = `上次检测：${status.last_run.replace('T', ' ').slice(0, 19)}，成功 ${ok} / 失败 ${err} / 跳过 ${((status.last_report || []).length - ok - err)}，发现新版本 ${total} 个${folder}`
          syncReport.list = status.last_report || []
          syncReport.folder = folder
          syncReport.lastRun = status.last_run || ''
        } else {
          syncStatusText.value = '尚未运行过同步'
        }
        await loadSoftwareConfig()
      } catch (error) {
        syncStatusText.value = '获取同步状态失败'
      }
    }
    const runPanSync = async () => {
      panSyncing.value = true
      try {
        const saved = await savePanConfig()
        if (!saved) {
          ElMessage.error('保存同步配置失败，无法执行检测')
          return
        }
        const response = await cloudAPI.sync()
        if (response.data?.success) {
          ElMessage.success('同步已开始（首次可能需下载上传多个安装包，耗时较长，页面会显示实时进度）')
          syncStatusText.value = '同步已开始，正在获取进度…'
          clearTimeout(syncPollTimer)
          syncPollTimer = setTimeout(loadPanSyncStatus, 2000)
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
      loadPanConfig()
      loadPanSyncStatus()
    })
    return {
      activeTab,
      panVersionMobileFields,
      emailLoading,
      softwareLoading,
      emailForm,
      softwareForm,
      saveSoftwareConfig,
      loadSoftwareConfig,
      saveEmailConfig,
      loadEmailConfig,
      panForm,
      panSaving,
      loadPanConfig,
      savePanConfig,
      panSyncing,
      panVersions,
      syncStatusText,
      syncReport,
      showSyncReport,
      loadPanSyncStatus,
      runPanSync
    }
  }
}
</script>
<style scoped>
.pan-version-mobile-head {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  min-width: 0;
}
.pan-version-name {
  font-weight: 600;
  word-break: break-all;
}

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
.folder-crumb {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 10px;
  min-height: 32px;
}
.folder-crumb-tag {
  cursor: pointer;
}
.token-guide {
  font-size: 13px;
  line-height: 1.9;
  color: #333;
  padding: 4px 8px;
}
.token-guide p {
  margin: 4px 0;
}
.token-guide code {
  background: var(--el-fill-color-light);
  border-radius: 3px;
  padding: 1px 5px;
  font-family: monospace;
  color: #d03050;
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
