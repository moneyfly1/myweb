<template>
  <div class="list-container tutorial-container">
    <div class="breadcrumb">首页 / 软件教程</div>
    <div class="page-header">
      <div class="page-title">
        <h1>软件教程</h1>
        <p>对应路由 /tutorials，按 Windows、macOS、iOS、Android 分类展示软件安装和订阅导入步骤。</p>
      </div>
    </div>
    <div class="card tutorial-card">
      <el-tabs v-model="activeTab" class="tutorial-tabs">
        <el-tab-pane label="Windows" name="windows">
          <div class="client-grid tutorial-client-grid">
            <div class="client-row">
              <div>
                <div class="client-title">Clash Verge</div>
                <div class="item-meta">下载软件 → 安装 → 复制 Clash 订阅 → 导入配置</div>
              </div>
              <div class="button-row">
                <el-button
                  type="primary"
                  size="small"
                  :loading="downloadingKey === 'clash_verge_windows_url'"
                  @click="downloadClient('clash_verge_windows_url', 'clash-verge')"
                >
                  下载
                </el-button>
                <el-button size="small" @click="copySubscription('clash')">复制订阅</el-button>
              </div>
            </div>
          </div>
          <WindowsTutorials />
        </el-tab-pane>
        <el-tab-pane label="macOS" name="macos">
          <div class="client-grid tutorial-client-grid">
            <div class="client-row">
              <div>
                <div class="client-title">Clash 系列</div>
                <div class="item-meta">下载软件 → 安装 → 复制 Clash 订阅 → 导入配置</div>
              </div>
              <div class="button-row">
                <el-button
                  type="primary"
                  size="small"
                  :loading="downloadingKey === 'clash_verge_macos_url'"
                  @click="downloadClient('clash_verge_macos_url', 'clash-verge')"
                >
                  下载
                </el-button>
                <el-button size="small" @click="copySubscription('clash')">复制订阅</el-button>
              </div>
            </div>
          </div>
          <MacOSTutorials />
        </el-tab-pane>
        <el-tab-pane label="iOS" name="ios">
          <div class="client-grid tutorial-client-grid">
            <div class="client-row">
              <div>
                <div class="client-title">Shadowrocket</div>
                <div class="item-meta">App Store 安装 → 扫码 / 一键导入 → 选择节点</div>
              </div>
              <div class="button-row">
                <el-button
                  type="primary"
                  size="small"
                  :loading="downloadingKey === 'shadowrocket_url'"
                  @click="downloadClient('shadowrocket_url', null, 'https://apps.apple.com/app/shadowrocket/id932747118')"
                >
                  打开商店
                </el-button>
                <el-button size="small" @click="openSubscriptionQr">显示二维码</el-button>
              </div>
            </div>
          </div>
          <iOSTutorials />
        </el-tab-pane>
        <el-tab-pane label="Android" name="android">
          <div class="client-grid tutorial-client-grid">
            <div class="client-row">
              <div>
                <div class="client-title">Clash Meta / V2rayNG</div>
                <div class="item-meta">下载 APK → 安装 → 复制订阅 → 导入配置</div>
              </div>
              <div class="button-row">
                <el-button
                  type="primary"
                  size="small"
                  :loading="downloadingKey === 'clash_meta_android_url'"
                  @click="downloadClient('clash_meta_android_url', 'clash-verge')"
                >
                  下载
                </el-button>
                <el-button size="small" @click="copySubscription('clash')">复制订阅</el-button>
              </div>
            </div>
          </div>
          <AndroidTutorials />
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import WindowsTutorials from '@/components/tutorials/WindowsTutorials.vue'
import AndroidTutorials from '@/components/tutorials/AndroidTutorials.vue'
import MacOSTutorials from '@/components/tutorials/MacOSTutorials.vue'
import iOSTutorials from '@/components/tutorials/iOSTutorials.vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { cachedAPI } from '@/utils/api'
import { safeOpen } from '@/utils/safeOpen'
import { copyToClipboard as copyText } from '@/utils/textSelection'

const router = useRouter()
const activeTab = ref('windows')
const softwareConfig = ref({})
const subscription = ref({})
const downloadingKey = ref('')

const getResponseData = (response) => {
  if (!response?.data) return {}
  if (response.data.success === false) return {}
  return response.data.data || response.data || {}
}

const loadRuntimeData = async () => {
  const [softwareResult, subscriptionResult] = await Promise.allSettled([
    cachedAPI.getSoftwareConfig(),
    cachedAPI.getUserSubscription()
  ])
  if (softwareResult.status === 'fulfilled') {
    softwareConfig.value = getResponseData(softwareResult.value)
  }
  if (subscriptionResult.status === 'fulfilled') {
    subscription.value = getResponseData(subscriptionResult.value)
  }
}

const downloadClient = async (configKey, githubKey = null, fallbackUrl = '') => {
  if (!configKey) return
  downloadingKey.value = configKey
  try {
    const configuredUrl = String(softwareConfig.value?.[configKey] || '').trim()
    if (configuredUrl) {
      safeOpen(configuredUrl)
      ElMessage.success('已打开下载页面')
      return
    }
    if (fallbackUrl) {
      safeOpen(fallbackUrl)
      ElMessage.success('已打开下载页面')
      return
    }
    if (githubKey) {
      ElMessage.info('正在获取最新下载链接...')
      const { getClientDownloadUrl, getClientReleasesUrl } = await import('@/utils/githubDownload')
      try {
        const downloadUrl = await getClientDownloadUrl(githubKey, softwareConfig.value || {})
        safeOpen(downloadUrl)
        ElMessage.success('已打开下载页面')
      } catch (error) {
        const releasesUrl = getClientReleasesUrl(githubKey)
        if (releasesUrl) {
          safeOpen(releasesUrl)
          ElMessage.warning('已打开发布页面，请手动选择下载')
          return
        }
        throw error
      }
      return
    }
    ElMessage.error('下载链接未配置，请联系管理员')
  } catch (error) {
    console.error('下载失败:', error)
    ElMessage.error('下载失败，请稍后重试')
  } finally {
    downloadingKey.value = ''
  }
}

const copySubscription = async (type = 'universal') => {
  const url = type === 'clash'
    ? subscription.value?.clash_url || subscription.value?.universal_url
    : subscription.value?.universal_url || subscription.value?.clash_url
  await copyText(url, '订阅链接已复制')
}

const openSubscriptionQr = () => {
  router.push('/subscription')
  ElMessage.info('请在订阅管理中查看真实订阅二维码')
}

onMounted(loadRuntimeData)
</script>
<style scoped>
.tutorial-container {
  padding: 0;
}
.breadcrumb {
  margin-bottom: 12px;
  color: #606266;
  font-size: 13px;
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
.page-title h1 {
  margin: 0;
  color: #303133;
  font-size: 22px;
  line-height: 1.25;
}
.page-title p {
  margin: 6px 0 0;
  color: #606266;
  line-height: 1.5;
}
.card {
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  overflow: hidden;
}
.tutorial-tabs :deep(.el-tabs__header) {
  margin: 0;
  border-bottom: 1px solid #ebeef5;
}
.tutorial-tabs :deep(.el-tabs__nav-wrap) {
  padding: 0 16px;
}
.tutorial-tabs :deep(.el-tabs__content) {
  padding: 16px;
}
.tutorial-client-grid {
  display: grid;
  gap: 14px;
  margin-bottom: 14px;
}
.client-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 14px;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  background: #fff;
}
.client-title {
  color: #303133;
  font-weight: 700;
}
.item-meta {
  margin-top: 6px;
  color: #909399;
  font-size: 12px;
  line-height: 1.5;
}
.button-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
}
@media (max-width: 768px) {
  .tutorial-tabs :deep(.el-tabs__nav-wrap) {
    padding: 0 12px;
  }
  .tutorial-tabs :deep(.el-tabs__item) {
    padding: 0 10px;
  }
  .tutorial-tabs :deep(.el-tabs__content) {
    padding: 0 12px 12px;
  }
  .client-row {
    display: grid;
  }
  .button-row {
    justify-content: flex-start;
  }
}
</style>
