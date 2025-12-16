<template>
  <div class="config-admin-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <h2>配置管理</h2>
          <p>管理系统配置文件和节点配置</p>
        </div>
      </template>

      <el-tabs v-model="activeTab" type="border-card">
        <el-tab-pane label="系统配置" name="system">
          <el-form
            ref="systemFormRef"
            :model="systemForm"
            label-width="120px"
          >
            <el-form-item label="网站名称">
              <el-input v-model="systemForm.site_name" />
            </el-form-item>
            
            <el-form-item label="网站描述">
              <el-input
                v-model="systemForm.site_description"
                type="textarea"
                :rows="3"
              />
            </el-form-item>
            
            <el-form-item label="网站Logo">
              <el-upload
                class="avatar-uploader"
                :action="uploadUrl"
                :show-file-list="false"
                :on-success="handleLogoSuccess"
                :before-upload="beforeLogoUpload"
              >
                <img v-if="systemForm.logo_url" :src="systemForm.logo_url" class="avatar" />
                <el-icon v-else class="avatar-uploader-icon"><Plus /></el-icon>
              </el-upload>
            </el-form-item>
            
            <el-form-item label="维护模式">
              <el-switch v-model="systemForm.maintenance_mode" />
            </el-form-item>
            
            <el-form-item label="维护信息">
              <el-input
                v-model="systemForm.maintenance_message"
                type="textarea"
                :rows="3"
                :disabled="!systemForm.maintenance_mode"
              />
            </el-form-item>
            
            <el-form-item class="config-buttons-group">
              <el-button type="primary" @click="saveSystemConfig" :loading="systemLoading" class="config-action-btn">
                保存系统配置
              </el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="软件下载配置" name="software">
          <div class="config-section">
            <h3>软件下载链接配置</h3>
            <el-form
              ref="softwareFormRef"
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
                  <el-form-item label="Mihomo Part">
                    <el-input v-model="softwareForm.mihomo_windows_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="Sparkle">
                    <el-input v-model="softwareForm.sparkle_windows_url" placeholder="请输入下载链接" />
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
                  <el-form-item label="Flash">
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
                <el-col :span="12"></el-col>
              </el-row>
              
              <el-divider content-position="left">macOS 软件</el-divider>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Flash">
                    <el-input v-model="softwareForm.flash_macos_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="Mihomo Part">
                    <el-input v-model="softwareForm.mihomo_macos_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
              </el-row>
              
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Sparkle">
                    <el-input v-model="softwareForm.sparkle_macos_url" placeholder="请输入下载链接" />
                  </el-form-item>
                </el-col>
                <el-col :span="12"></el-col>
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
        </el-tab-pane>
        <el-tab-pane label="节点配置" name="nodes">
          <div class="config-section">
            <h3>Clash 配置</h3>
            <el-form>
              <el-form-item label="配置文件">
                <el-input
                  v-model="clashConfig"
                  type="textarea"
                  :rows="15"
                  placeholder="请输入Clash配置文件内容"
                />
              </el-form-item>
              <el-form-item class="config-buttons-group">
                <el-button type="primary" @click="saveClashConfig" :loading="clashLoading" class="config-action-btn">
                  保存Clash配置
                </el-button>
                <el-button @click="loadClashConfig" class="config-action-btn">
                  加载当前配置
                </el-button>
              </el-form-item>
            </el-form>
          </div>

          <div class="config-section">
            <h3>V2Ray 配置</h3>
            <el-form>
              <el-form-item label="配置文件">
                <el-input
                  v-model="v2rayConfig"
                  type="textarea"
                  :rows="15"
                  placeholder="请输入V2Ray配置文件内容"
                />
              </el-form-item>
              <el-form-item class="config-buttons-group">
                <el-button type="primary" @click="saveV2rayConfig" :loading="v2rayLoading" class="config-action-btn">
                  保存V2Ray配置
                </el-button>
                <el-button @click="loadV2rayConfig" class="config-action-btn">
                  加载当前配置
                </el-button>
              </el-form-item>
            </el-form>
          </div>

          <div class="config-section">
            <h3>Clash 失效配置</h3>
            <el-form>
              <el-form-item label="失效配置文件">
                <el-input
                  v-model="clashConfigInvalid"
                  type="textarea"
                  :rows="10"
                  placeholder="请输入Clash失效配置文件内容（用于无效用户）"
                />
              </el-form-item>
              <el-form-item class="config-buttons-group">
                <el-button type="primary" @click="saveClashConfigInvalid" :loading="clashInvalidLoading" class="config-action-btn">
                  保存Clash失效配置
                </el-button>
                <el-button @click="loadClashConfigInvalid" class="config-action-btn">
                  加载当前失效配置
                </el-button>
              </el-form-item>
            </el-form>
          </div>

          <div class="config-section">
            <h3>V2Ray 失效配置</h3>
            <el-form>
              <el-form-item label="失效配置文件">
                <el-input
                  v-model="v2rayConfigInvalid"
                  type="textarea"
                  :rows="10"
                  placeholder="请输入V2Ray失效配置文件内容（用于无效用户）"
                />
              </el-form-item>
              <el-form-item class="config-buttons-group">
                <el-button type="primary" @click="saveV2rayConfigInvalid" :loading="v2rayInvalidLoading" class="config-action-btn">
                  保存V2Ray失效配置
                </el-button>
                <el-button @click="loadV2rayConfigInvalid" class="config-action-btn">
                  加载当前失效配置
                </el-button>
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>
        <el-tab-pane label="邮件配置" name="email">
          <el-form
            ref="emailFormRef"
            :model="emailForm"
            label-width="120px"
            class="email-config-form"
          >
            <el-form-item label="SMTP服务器">
              <el-input v-model="emailForm.smtp_host" placeholder="例如: smtp.gmail.com" />
            </el-form-item>
            
            <el-form-item label="SMTP端口">
              <el-input-number v-model="emailForm.smtp_port" :min="1" :max="65535" />
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
        <el-tab-pane label="备份恢复" name="backup">
          <div class="backup-section">
            <h3>配置备份</h3>
            <el-button type="primary" @click="exportConfig">
              <i class="el-icon-download"></i>
              导出配置
            </el-button>
            
            <el-upload
              class="upload-demo"
              :auto-upload="false"
              :on-change="handleConfigImport"
              :before-upload="beforeConfigUpload"
              accept=".json"
            >
              <el-button type="success">
                <i class="el-icon-upload"></i>
                导入配置
              </el-button>
            </el-upload>
          </div>

          <div class="backup-section">
            <h3>备份历史</h3>
            <el-table :data="backupHistory" style="width: 100%">
              <el-table-column prop="filename" label="文件名" />
              <el-table-column prop="created_at" label="创建时间" />
              <el-table-column prop="size" label="文件大小" />
              <el-table-column label="操作" width="200">
                <template #default="{ row }">
                  <el-button
                    type="primary"
                    size="small"
                    @click="downloadBackup(row.filename)"
                  >
                    下载
                  </el-button>
                  <el-button
                    type="danger"
                    size="small"
                    @click="deleteBackup(row.filename)"
                  >
                    删除
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { configAPI, softwareConfigAPI } from '@/utils/api'
import { adminAPI } from '@/utils/api'
import { useRouter } from 'vue-router'

export default {
  name: 'AdminConfig',
  components: {
    Plus
  },
  setup() {
    const activeTab = ref('system')
    const systemFormRef = ref()
    const emailFormRef = ref()
    const softwareFormRef = ref()
    
    const systemLoading = ref(false)
    const clashLoading = ref(false)
    const v2rayLoading = ref(false)
    const clashInvalidLoading = ref(false)
    const v2rayInvalidLoading = ref(false)
    const emailLoading = ref(false)
    const softwareLoading = ref(false)
    
    const clashConfig = ref('')
    const v2rayConfig = ref('')
    const clashConfigInvalid = ref('')
    const v2rayConfigInvalid = ref('')
    const backupHistory = ref([])
    
    const uploadUrl = '/api/admin/upload'

    const systemForm = reactive({
      site_name: '',
      site_description: '',
      logo_url: '',
      maintenance_mode: false,
      maintenance_message: ''
    })

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
      mihomo_windows_url: '',
      sparkle_windows_url: '',
      hiddify_windows_url: '',
      flash_windows_url: '',
      clash_android_url: '',
      v2rayng_url: '',
      hiddify_android_url: '',
      flash_macos_url: '',
      mihomo_macos_url: '',
      sparkle_macos_url: '',
      shadowrocket_url: ''
    })

    const router = useRouter()
    const saveSystemConfig = async () => {
      systemLoading.value = true
      try {
        await configAPI.saveSystemConfig(systemForm)
        ElMessage.success('系统配置保存成功')
        await loadSystemConfig()
      } catch (error) {
        ElMessage.error('保存失败')
      } finally {
        systemLoading.value = false
      }
    }
    const saveClashConfig = async () => {
      clashLoading.value = true
      try {
        await configAPI.saveClashConfig(clashConfig.value)
        ElMessage.success('Clash配置保存成功')
      } catch (error) {
        ElMessage.error('保存失败')
      } finally {
        clashLoading.value = false
      }
    }
    const saveV2rayConfig = async () => {
      v2rayLoading.value = true
      try {
        await configAPI.saveV2rayConfig(v2rayConfig.value)
        ElMessage.success('V2Ray配置保存成功')
      } catch (error) {
        ElMessage.error('保存失败')
      } finally {
        v2rayLoading.value = false
      }
    }
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
          smtp_port: emailForm.smtp_port,
          email_username: emailForm.email_username,
          email_password: emailForm.email_password,
          sender_name: emailForm.sender_name,
          smtp_encryption: emailForm.smtp_encryption,
          from_email: emailForm.from_email
        }
        await configAPI.saveEmailConfig(emailConfigData)
        ElMessage.success('邮件配置保存成功')
        await loadEmailConfig()
      } catch (error) {
        ElMessage.error('保存失败')
      } finally {
        emailLoading.value = false
      }
    }
    const loadClashConfig = async () => {
      try {
        const response = await configAPI.getClashConfig()
        if (response.data && response.data.success) {
          const data = response.data.data
          clashConfig.value = typeof data === 'string' ? data : (data?.content || data || '')
        }
      } catch (error) {
        ElMessage.error('加载Clash配置失败')
      }
    }

    const loadV2rayConfig = async () => {
      try {
        const response = await configAPI.getV2rayConfig()
        if (response.data && response.data.success) {
          const data = response.data.data
          v2rayConfig.value = typeof data === 'string' ? data : (data?.content || data || '')
        }
      } catch (error) {
        ElMessage.error('加载V2Ray配置失败')
      }
    }
    const saveClashConfigInvalid = async () => {
      clashInvalidLoading.value = true
      try {
        await configAPI.saveClashConfigInvalid(clashConfigInvalid.value)
        ElMessage.success('Clash失效配置保存成功')
      } catch (error) {
        ElMessage.error('保存失败')
      } finally {
        clashInvalidLoading.value = false
      }
    }
    const saveV2rayConfigInvalid = async () => {
      v2rayInvalidLoading.value = true
      try {
        await configAPI.saveV2rayConfigInvalid(v2rayConfigInvalid.value)
        ElMessage.success('V2Ray失效配置保存成功')
      } catch (error) {
        ElMessage.error('保存失败')
      } finally {
        v2rayInvalidLoading.value = false
      }
    }
    const loadClashConfigInvalid = async () => {
      try {
        const response = await configAPI.getClashConfigInvalid()
        if (response.data && response.data.success) {
          const data = response.data.data
          clashConfigInvalid.value = typeof data === 'string' ? data : (data?.content || data || '')
        }
      } catch (error) {
        ElMessage.error('加载Clash失效配置失败')
      }
    }
    const loadV2rayConfigInvalid = async () => {
      try {
        const response = await configAPI.getV2rayConfigInvalid()
        if (response.data && response.data.success) {
          const data = response.data.data
          v2rayConfigInvalid.value = typeof data === 'string' ? data : (data?.content || data || '')
        }
      } catch (error) {
        ElMessage.error('加载V2Ray失效配置失败')
      }
    }
    const exportConfig = async () => {
      try {
        const response = await configAPI.exportConfig()
        const configData = response.data?.data || response.data
        
        if (!configData) {
          ElMessage.error('导出失败: 未获取到配置数据')
          return
        }
        
        const blob = new Blob([JSON.stringify(configData, null, 2)], {
          type: 'application/json'
        })
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        const dateStr = new Date().toISOString().split('T')[0]
        a.download = `cboard-config-${dateStr}.json`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        window.URL.revokeObjectURL(url)
        ElMessage.success('配置导出成功')
      } catch (error) {
        const errorMsg = error.response?.data?.message || error.message || '未知错误'
        ElMessage.error(`导出失败: ${errorMsg}`)
      }
    }
    const handleLogoSuccess = (response) => {
      systemForm.logo_url = response.data.url
      ElMessage.success('Logo上传成功')
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

    const handleConfigImport = async (file, fileList) => {
      try {
        const fileContent = await readFileContent(file.raw)
        const configData = JSON.parse(fileContent)
        const response = await configAPI.importConfig(configData)
        
        if (response.data && !response.data.success) {
          throw new Error(response.data.message || '导入失败')
        }
        
        ElMessage.success('配置导入成功')
        await loadSystemConfig()
        await loadEmailConfig()
        await loadSoftwareConfig()
        await loadClashConfig()
        await loadV2rayConfig()
        await loadClashConfigInvalid()
        await loadV2rayConfigInvalid()
      } catch (error) {
        const errorMsg = error.response?.data?.message || error.message || '未知错误'
        ElMessage.error(`导入配置失败: ${errorMsg}`)
      }
    }
    const readFileContent = (file) => {
      return new Promise((resolve, reject) => {
        const reader = new FileReader()
        reader.onload = (e) => resolve(e.target.result)
        reader.onerror = (e) => reject(e)
        reader.readAsText(file)
      })
    }

    const beforeConfigUpload = (file) => {
      const isJSON = file.type === 'application/json'
      if (!isJSON) {
        ElMessage.error('只能上传JSON文件!')
        return false
      }
      return true
    }

    // 加载系统配置
    const loadSystemConfig = async () => {
      try {
        const response = await configAPI.getSystemConfig()
        if (response.data && response.data.success) {
          const configData = response.data.data
          Object.assign(systemForm, configData)
        }
      } catch (error) {
        ElMessage.error('加载系统配置失败')
      }
    }
    const loadEmailConfig = async () => {
      try {
        const response = await configAPI.getEmailConfig()
        if (response.data && response.data.success) {
          const configData = response.data.data
          emailForm.smtp_host = configData.smtp_host || ''
          emailForm.smtp_port = configData.smtp_port || 587
          emailForm.email_username = configData.email_username || configData.smtp_username || ''
          emailForm.email_password = configData.email_password || configData.smtp_password || ''
          emailForm.sender_name = configData.sender_name || ''
          emailForm.smtp_encryption = configData.smtp_encryption || 'tls'
          emailForm.from_email = configData.from_email || ''
        }
      } catch (error) {
        ElMessage.error('加载邮件配置失败')
      }
    }
    const formatDate = (timestamp) => {
      const date = new Date(timestamp)
      const year = date.getFullYear()
      const month = String(date.getMonth() + 1).padStart(2, '0')
      const day = String(date.getDate()).padStart(2, '0')
      const hours = String(date.getHours()).padStart(2, '0')
      const minutes = String(date.getMinutes()).padStart(2, '0')
      return `${year}-${month}-${day} ${hours}:${minutes}`
    }

    // 获取邮件状态标签类型
    const getEmailStatusTagType = (status) => {
      switch (status) {
        case 'pending':
          return 'info'
        case 'sending':
          return 'warning'
        case 'sent':
          return 'success'
        case 'failed':
          return 'danger'
        default:
          return 'info'
      }
    }
    const getEmailStatusText = (status) => {
      switch (status) {
        case 'pending':
          return '待发送'
        case 'sending':
          return '发送中'
        case 'sent':
          return '已发送'
        case 'failed':
          return '发送失败'
        default:
          return status
      }
    }

    onMounted(() => {
      loadSystemConfig()
      loadEmailConfig()
      loadClashConfig()
      loadV2rayConfig()
      loadClashConfigInvalid()
      loadV2rayConfigInvalid()
      loadSoftwareConfig()
    })

    return {
      activeTab,
      systemFormRef,
      emailFormRef,
      systemLoading,
      clashLoading,
      v2rayLoading,
      clashInvalidLoading,
      v2rayInvalidLoading,
      emailLoading,
      softwareLoading,
      systemForm,
      emailForm,
      softwareForm,
      clashConfig,
      v2rayConfig,
      clashConfigInvalid,
      v2rayConfigInvalid,
      backupHistory,
      uploadUrl,
      saveSystemConfig,
      saveClashConfig,
      saveV2rayConfig,
      saveSoftwareConfig,
      loadSoftwareConfig,
      saveClashConfigInvalid,
      saveV2rayConfigInvalid,
      saveEmailConfig,
      loadClashConfig,
      loadV2rayConfig,
      loadClashConfigInvalid,
      loadV2rayConfigInvalid,
      exportConfig,
      handleLogoSuccess,
      beforeLogoUpload,
      handleConfigImport,
      beforeConfigUpload,
      formatDate
    }
  }
}
</script>

<style scoped>
.config-admin-container {
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

.config-section {
  margin-bottom: 30px;
}

.config-section h3 {
  color: #333;
  margin-bottom: 20px;
  font-size: 1.2rem;
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
  overflow: hidden;
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

.header-actions {
  display: flex;
  gap: 10px;
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

.pagination-wrapper {
  text-align: right;
  margin-top: 20px;
}

@media (max-width: 768px) {
  .config-admin-container {
    padding: 10px;
    width: 100%;
    box-sizing: border-box;
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
  
  /* 统一所有表单在移动端的样式 */
  :deep(.el-form) {
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
        
        /* 统一所有输入框宽度 */
        .el-input,
        .el-input-number,
        .el-select,
        .el-textarea,
        .el-input__wrapper {
          width: 100% !important;
          max-width: 100% !important;
          box-sizing: border-box;
        }
        
        /* 统一所有按钮宽度和样式（排除邮件配置按钮组） */
        .el-button:not(.email-action-btn) {
          width: 100% !important;
          min-width: 100% !important;
          box-sizing: border-box;
          margin-bottom: 10px;
          margin-right: 0 !important;
        }
        
        .el-button:not(.email-action-btn):last-child {
          margin-bottom: 0;
        }
        
        /* 确保输入框内部元素宽度一致 */
        .el-input__inner {
          width: 100% !important;
          box-sizing: border-box;
        }
        
        /* 确保输入数字组件宽度一致 */
        .el-input-number {
          width: 100% !important;
        }
        
        .el-input-number .el-input__wrapper {
          width: 100% !important;
        }
        
        /* 确保下拉框宽度一致 */
        .el-select {
          width: 100% !important;
        }
        
        .el-select .el-input__wrapper {
          width: 100% !important;
        }
        
        /* 确保文本域宽度一致 */
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
  
  /* 确保标签页内容宽度一致 */
  :deep(.el-tabs__content) {
    width: 100% !important;
    box-sizing: border-box;
  }
  
  /* 确保卡片内容宽度一致 */
  :deep(.el-card__body) {
    width: 100% !important;
    padding: 12px !important;
    box-sizing: border-box;
  }
  
  /* 确保表格在移动端宽度一致 */
  :deep(.el-table) {
    width: 100% !important;
    box-sizing: border-box;
  }
  
  /* 软件下载配置中的列布局在移动端改为单列 */
  :deep(.el-row) {
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

/* 邮件配置表单样式 */
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

/* 邮件配置按钮组样式 - 最高优先级 */
.email-buttons-group {
  width: 100% !important;
  max-width: 100% !important;
  box-sizing: border-box !important;
  margin: 0 !important;
  padding: 0 !important;
  
  /* 确保表单项本身没有额外样式 */
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
      /* 桌面端：按钮水平排列，统一宽度 */
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
      /* 移动端：按钮垂直排列，宽度100%，完全对齐 */
      flex-direction: column !important;
      width: 100% !important;
      max-width: 100% !important;
      align-items: stretch !important;
      gap: 10px !important;
    }
  }
  
  /* 移动端按钮样式 - 确保完全一致 */
  @media (max-width: 768px) {
    :deep(.el-button.email-action-btn) {
      width: 100% !important;
      min-width: 100% !important;
      max-width: 100% !important;
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
    
    /* 确保按钮内部元素不影响宽度 */
    :deep(.el-button.email-action-btn > span) {
      width: 100% !important;
      display: block !important;
      text-align: center !important;
      box-sizing: border-box !important;
    }
    
    /* 确保按钮图标和文字对齐 */
    :deep(.el-button.email-action-btn .el-icon) {
      margin-right: 8px !important;
    }
    
    /* 强制覆盖所有可能的宽度设置 */
    :deep(.el-button.email-action-btn),
    :deep(.el-button.email-action-btn *) {
      max-width: 100% !important;
    }
  }
}

/* 配置按钮组样式（Clash、V2Ray、系统配置等） */
.config-buttons-group {
  .el-form-item__content {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
    align-items: stretch;
    justify-content: flex-start;
    width: 100%;
    box-sizing: border-box;
    
    @media (min-width: 769px) {
      /* 桌面端：按钮水平排列，统一宽度 */
      .config-action-btn {
        flex: 1 1 0;
        min-width: 140px;
        max-width: 200px;
        box-sizing: border-box;
      }
    }
    
    @media (max-width: 768px) {
      /* 移动端：按钮垂直排列，宽度100% */
      flex-direction: column;
      width: 100%;
      
      .config-action-btn {
        width: 100% !important;
        min-width: 100% !important;
        max-width: 100% !important;
        margin-bottom: 10px;
        margin-right: 0 !important;
        box-sizing: border-box;
      }
      
      .config-action-btn:last-child {
        margin-bottom: 0;
      }
    }
  }
}

.payment-form .el-divider {
  margin: 30px 0 20px 0;
}

.payment-form .el-divider:first-child {
  margin-top: 0;
}

/* 移除所有输入框的圆角和阴影效果，设置为简单长方形 */
:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
}

:deep(.el-select .el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
}

:deep(.el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background-color: transparent !important;
}

:deep(.el-input__wrapper:hover) {
  border-color: #c0c4cc !important;
  box-shadow: none !important;
}

:deep(.el-input__wrapper.is-focus) {
  border-color: #1677ff !important;
  box-shadow: none !important;
}
</style> 