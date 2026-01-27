<template>
  <div class="config-admin-container">
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
    </el-card>
  </div>
</template>

<script>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { configAPI, softwareConfigAPI } from '@/utils/api'

export default {
  name: 'AdminConfig',
  setup() {
    const activeTab = ref('software')
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
          // 确保 smtp_port 是数字类型
          smtp_port: typeof emailForm.smtp_port === 'number' ? emailForm.smtp_port : Number(emailForm.smtp_port) || 587,
          email_username: emailForm.email_username,
          // 安全修复：如果密码是掩码（******），不发送给后端，让后端保持原值
          // 只有用户真正修改了密码（不是掩码）时才发送
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
          // 确保 smtp_port 是数字类型，ElInputNumber 需要 Number 或 Null
          const port = configData.smtp_port
          emailForm.smtp_port = port ? Number(port) : 587
          emailForm.email_username = configData.email_username || configData.smtp_username || ''
          // 安全修复：密码字段不直接显示真实值，使用掩码或空字符串
          // 如果后端返回了密码，使用掩码显示；否则留空，让用户重新输入
          if (configData.email_password || configData.smtp_password) {
            // 如果后端返回了掩码（如******），直接使用；否则使用掩码
            const passwordValue = configData.email_password || configData.smtp_password
            if (passwordValue && passwordValue.length > 0 && !passwordValue.startsWith('*')) {
              // 后端返回了真实密码，使用掩码保护
              emailForm.email_password = '******'
            } else {
              // 后端返回了掩码或空值
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

    onMounted(() => {
      loadEmailConfig()
      loadSoftwareConfig()
    })

    return {
      activeTab,
      emailLoading,
      softwareLoading,
      emailForm,
      softwareForm,
      saveSoftwareConfig,
      loadSoftwareConfig,
      saveEmailConfig,
      loadEmailConfig
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
