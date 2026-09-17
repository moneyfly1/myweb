<template>
  <div v-if="visible && items.length" class="moneyfly-download-panel" :class="{ 'is-plain': plain }">
    <div v-if="!hideHead" class="moneyfly-panel-head">
      <div class="moneyfly-panel-title">
        <span class="moneyfly-name">{{ brand.name }}</span>
        <el-tag type="success" effect="dark" size="small">{{ brand.badge }}</el-tag>
        <el-tag type="danger" effect="plain" size="small">推荐优先使用</el-tag>
        <span v-if="config.version" class="moneyfly-panel-version">v{{ config.version }}</span>
      </div>
      <p class="moneyfly-panel-intro">{{ description }}</p>
    </div>
    <div class="moneyfly-panel-grid">
      <div
        v-for="item in items"
        :key="item.key"
        class="moneyfly-panel-item"
        :class="{ 'is-recommended': item.configKey === recommendedKey }"
      >
        <div class="moneyfly-panel-item-head">
          <span class="moneyfly-platform">{{ item.label }}</span>
          <el-tag
            v-if="item.archLabel"
            size="small"
            :type="item.arch === 'apple' ? 'success' : 'info'"
            effect="plain"
          >
            {{ item.tag }}
          </el-tag>
          <el-tag
            v-if="item.configKey === recommendedKey"
            size="small"
            type="danger"
            effect="light"
          >
            推荐给你的设备
          </el-tag>
        </div>
        <div class="moneyfly-hint">
          <template v-if="item.archLabel">{{ item.archLabel }} · </template>{{ item.hint }}
        </div>
        <el-button
          type="primary"
          size="small"
          class="moneyfly-panel-btn"
          :loading="loadingKey === item.configKey"
          @click="handleDownload(item)"
        >
          <el-icon><Download /></el-icon>
          下载 {{ brand.name }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
/**
 * MoneyFlyDownloadPanel —— MoneyFly 自研客户端下载面板（用户端通用）
 * 帮助中心 / 软件教程等"用户下载软件的地方"共用此组件，
 * 平台列表与配置解析来自 utils/moneyflyClient.js，避免重复维护。
 */
import { computed, ref } from 'vue'
import { Download } from '@element-plus/icons-vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { safeOpen } from '@/utils/safeOpen'
import {
  MONEYFLY_BRAND,
  isMoneyflyVisible,
  readMoneyflyConfig,
  detectMoneyflyPlatformKey,
  resolveMoneyflyUrl,
} from '@/utils/moneyflyClient'

const props = defineProps({
  // softwareConfig：/api/v1/software-config 返回的原始配置对象
  softwareConfig: {
    type: Object,
    default: () => ({}),
  },
  // onlyPlatform：仅展示指定平台，支持逗号分隔多值
  // （如软件教程 Windows 标签传 'windows'，macOS 标签传 'macos_arm,macos_intel'）
  onlyPlatform: {
    type: String,
    default: '',
  },
  // plain：去掉外框/内边距，供已有卡片容器的页面（如仪表盘）内嵌使用
  plain: {
    type: Boolean,
    default: false,
  },
  // hideHead：隐藏组件内部的标题区（由外层卡片自己提供标题时使用）
  hideHead: {
    type: Boolean,
    default: false,
  },
})

const brand = MONEYFLY_BRAND
const loadingKey = ref('')

const config = computed(() => readMoneyflyConfig(props.softwareConfig || {}))
const visible = computed(() => isMoneyflyVisible(config.value))
const items = computed(() => {
  const list = config.value.items.filter(i => i.configured)
  const wanted = String(props.onlyPlatform || '')
    .split(',')
    .map(s => s.trim())
    .filter(Boolean)
  if (!wanted.length) return list
  return list.filter(i => wanted.includes(i.key))
})
const description = computed(() => config.value.note || brand.intro)
const recommendedKey = computed(() => detectMoneyflyPlatformKey())

const handleDownload = (item) => {
  if (!item || !item.configured) {
    ElMessage.error('该平台的下载地址尚未配置，请联系管理员')
    return
  }
  const url = resolveMoneyflyUrl(item.url)
  if (!url) {
    ElMessage.error('下载地址无效，请联系管理员')
    return
  }
  loadingKey.value = item.configKey
  try {
    safeOpen(url)
  } catch {
    ElMessage.error('打开下载地址失败，请稍后重试')
  } finally {
    loadingKey.value = ''
  }
}
</script>

<style scoped>
.moneyfly-download-panel {
  border: 1px solid var(--el-color-primary-light-5);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
  background: linear-gradient(180deg, var(--el-color-primary-light-9), transparent 70%);
}
.moneyfly-download-panel.is-plain {
  border: none;
  border-radius: 0;
  padding: 0;
  margin-bottom: 0;
  background: none;
}
.moneyfly-panel-head {
  margin-bottom: 12px;
}
.moneyfly-panel-title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.moneyfly-panel-version {
  font-size: 13px;
  font-weight: 400;
  color: var(--el-text-color-secondary);
}
.moneyfly-panel-intro {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--el-text-color-regular);
}
.moneyfly-panel-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}
.moneyfly-panel-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
}
.moneyfly-panel-item.is-recommended {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
}
.moneyfly-panel-item-head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}
.moneyfly-platform {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.moneyfly-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  min-height: 18px;
}
.moneyfly-panel-btn {
  width: 100%;
  margin-top: 2px;
}
@media (max-width: 768px) {
  .moneyfly-panel-grid {
    grid-template-columns: 1fr;
  }
}
</style>
