<template>
  <div class="list-container admin-config">
    <el-card>
  <template #header>
    <div class="card-header">
      <h2>配置管理</h2>
      <p>管理软件下载配置、云盘自动填充和邮件配置</p>
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
            <el-divider content-position="left">阿里云盘自动填充（可选）</el-divider>
            <el-alert
              type="info"
              show-icon
              :closable="false"
              title="配置阿里云盘 refresh_token 后，定时从 GitHub 获取最新安装包上传到你的阿里云盘，并自动把上方软件下载链接填充为 pan:// 动态直链（用户每次点击下载时实时生成新链接，链接永不失效）。阿里云盘 OAuth 不绑定 IP、无境外风控，海外服务器可用。"
              description="获取 refresh_token：浏览器登录 www.alipan.com → F12 → Application → Local Storage → https://www.alipan.com → 找到键名 token，复制其中 refresh_token 的值（约 90 天有效；测试连接成功后会自动轮换并保存新 token）。"
            />
            <el-form :model="panForm" label-width="150px" style="margin-top: 16px">
              <el-form-item label="云盘上传目录">
                <el-input v-model="panForm.aliyun_folder" placeholder="如：软件下载（支持 软件/下载 多级路径；留空 = 网盘根目录）" />
                <div class="pan-tip" style="width: 100%">同步时会在该目录下自动为每个软件创建子文件夹（如 软件下载/v2rayN/），安装包按软件分别存放。</div>
              </el-form-item>
              <el-form-item label="阿里云盘 refresh_token">
                <el-input v-model="panForm.aliyun_refresh_token" type="textarea" :rows="2" placeholder="粘贴 refresh_token" />
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
                <el-button @click="loadPanConfig" class="config-action-btn">
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
              title="自动从 GitHub 获取 v2rayN / Hiddify / Clash Verge / Clash Part / V2rayNG / Clash Meta / FlClash 的最新版本，下载安装包上传到你的阿里云盘，并自动设置为动态直链（pan://）。仅当 GitHub 有新版本时才会下载上传，其余时间跳过。"
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
    </el-card>
  </div>
</template>
<script>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { configAPI, softwareConfigAPI, cloudAPI } from '@/utils/api'
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
    const panForm = reactive({
      aliyun_refresh_token: '',
      aliyun_folder: '',
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
        const response = await cloudAPI.getConfig()
        if (response.data && response.data.success) {
          const data = response.data.data || {}
          panForm.aliyun_refresh_token = data.aliyun_refresh_token || ''
          panForm.aliyun_folder = data.aliyun_folder || ''
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
          aliyun_refresh_token: panForm.aliyun_refresh_token,
          aliyun_folder: panForm.aliyun_folder,
          sync_enabled: panForm.sync_enabled,
          sync_interval_hours: panForm.sync_interval_hours
        })
        if (response.data && response.data.success) {
          ElMessage.success('云盘配置已保存')
          await loadPanConfig()
        } else {
          ElMessage.error(response.data?.message || '保存失败，请检查网络或配置')
        }
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '保存失败')
      } finally {
        panSaving.value = false
      }
    }
    // ---- 阿里云盘连接测试 ----
    const aliyunTesting = ref(false)
    const aliyunTestResult = ref('')
    const testAliyun = async () => {
      aliyunTesting.value = true
      aliyunTestResult.value = ''
      try {
        const saved = await savePanConfig()
        if (!saved) {
          aliyunTestResult.value = '保存配置失败'
          return
        }
        const response = await cloudAPI.aliyunTest()
        if (response.data?.success) {
          const d = response.data.data || {}
          aliyunTestResult.value = `连接成功（${d.folder || '根目录'} ${d.root_files} 个文件）${d.refresh_token_rotated ? '，token 已自动轮换' : ''}`
        } else {
          aliyunTestResult.value = response.data?.message || '连接失败'
        }
      } catch (error) {
        aliyunTestResult.value = error.response?.data?.message || '连接失败'
      } finally {
        aliyunTesting.value = false
      }
    }
    const panSyncing = ref(false)
    const panVersions = ref([])
    const syncStatusText = ref('')
    let syncPollTimer = null
    const loadPanSyncStatus = async () => {
      try {
        const [statusRes, versionsRes] = await Promise.all([
          cloudAPI.syncStatus(),
          cloudAPI.versions()
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
          ElMessage.error('保存云盘配置失败，无法执行同步')
          return
        }
        const response = await cloudAPI.sync()
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
      panSaving,
      loadPanConfig,
      savePanConfig,
      aliyunTesting,
      aliyunTestResult,
      testAliyun,
      panSyncing,
      panVersions,
      syncStatusText,
      loadPanSyncStatus,
      runPanSync
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
