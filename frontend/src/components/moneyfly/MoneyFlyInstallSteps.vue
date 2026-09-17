<template>
  <div v-if="visible && platform" class="moneyfly-steps">
    <div class="moneyfly-steps-title">
      {{ brand.name }} 安装步骤
      <el-tag size="small" type="success" effect="plain">官方自研</el-tag>
    </div>
    <p class="moneyfly-steps-hint">{{ platform.hint }}</p>
    <ol class="moneyfly-steps-list">
      <li v-for="(step, idx) in platform.steps" :key="idx">{{ step }}</li>
    </ol>
    <div v-if="showUsage" class="moneyfly-steps-usage">
      <div class="moneyfly-steps-usage-title">导入订阅</div>
      <ol class="moneyfly-steps-list">
        <li v-for="(step, idx) in usageSteps" :key="idx">{{ step }}</li>
      </ol>
    </div>
  </div>
</template>

<script setup>
/**
 * MoneyFlyInstallSteps —— MoneyFly 自研客户端安装步骤（按平台）
 * 文案来自 components/moneyfly/moneyflyGuide.js，与帮助中心共用同一份内容。
 */
import { computed } from 'vue'
import { MONEYFLY_BRAND, isMoneyflyVisible, readMoneyflyConfig } from '@/utils/moneyflyClient'
import { MONEYFLY_INSTALL_STEPS, MONEYFLY_USAGE_STEPS } from '@/components/moneyfly/moneyflyGuide'

const props = defineProps({
  // platform：windows / macos / android
  platform: {
    type: String,
    required: true,
  },
  // showUsage：是否附带"导入订阅"步骤（同一页面只需出现一次，默认关闭）
  showUsage: {
    type: Boolean,
    default: false,
  },
  // softwareConfig：传入后与下载面板同样门控——未启用或未配置地址时不展示教程，
  // 避免出现"有教程但下不了软件"的矛盾
  softwareConfig: {
    type: Object,
    default: null,
  },
})

const brand = MONEYFLY_BRAND
const platform = computed(() => MONEYFLY_INSTALL_STEPS.find(p => p.key === props.platform) || null)
const visible = computed(() => {
  if (!props.softwareConfig) return true // 未传配置时由调用方自行门控
  return isMoneyflyVisible(readMoneyflyConfig(props.softwareConfig))
})
const usageSteps = MONEYFLY_USAGE_STEPS
</script>

<style scoped>
.moneyfly-steps {
  margin: 12px 0 16px;
  padding: 14px 16px;
  border: 1px dashed var(--el-color-primary-light-5);
  border-radius: 8px;
  background: var(--el-color-primary-light-9);
}
.moneyfly-steps-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14.5px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.moneyfly-steps-hint {
  margin: 6px 0 8px;
  font-size: 12.5px;
  color: var(--el-text-color-secondary);
}
.moneyfly-steps-list {
  margin: 0;
  padding-left: 20px;
  font-size: 13.5px;
  line-height: 1.9;
  color: var(--el-text-color-primary);
}
.moneyfly-steps-list li {
  margin-bottom: 4px;
}
.moneyfly-steps-usage {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-color-primary-light-7);
}
.moneyfly-steps-usage-title {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 6px;
}
</style>
