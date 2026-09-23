<!--
  ClientPlatformList —— 按「用户端（平台）」分栏的客户端下载列表。

  为什么需要：仪表盘的「客户端下载与教程」此前只给一条「当前系统的快捷下载」，
  macOS 用户只能看到一句「Apple 芯片 / Intel 由客户端中心选择」—— 等于把架构选择
  推给了用户去找。这里改成与「客户端中心」一致的分端展示，并且 **macOS 直接给出
  两个按钮**（Apple 芯片 ARM / Intel x64），不用再点下拉。

  数据来源仍唯一：data/clientRegistry.js（平台归属、下载键、mac 双架构）+ utils/clientDownload.js
  （取配置地址 / pan:// 解析 / GitHub 兜底）。组件本身不硬编码任何客户端或链接。

  用法：
    <ClientPlatformList :software-config="softwareConfig" :default-platform="'macos'" />
-->
<template>
  <div class="client-platform-list">
    <!-- 平台切换（只显示有客户端的平台，避免出现空栏） -->
    <div class="cpl-platforms" role="tablist">
      <button
        v-for="platform in visiblePlatforms"
        :key="platform.key"
        type="button"
        role="tab"
        class="cpl-platform"
        :class="{ 'is-active': platform.key === activePlatform }"
        :aria-selected="platform.key === activePlatform"
        @click="activePlatform = platform.key"
      >
        {{ platform.label }}
        <span class="cpl-count">{{ clientsOf(platform.key).length }}</span>
      </button>
    </div>

    <div v-if="clients.length" class="cpl-rows">
      <div v-for="client in clients" :key="client.id" class="cpl-row">
        <div class="cpl-info">
          <div class="cpl-name">
            {{ client.name }}
            <el-tag v-if="client.official" type="success" size="small" effect="dark">官方自研</el-tag>
            <el-tag v-else-if="client.recommended" type="danger" size="small" effect="plain">推荐</el-tag>
          </div>
          <div class="cpl-desc">{{ client.description }}</div>
        </div>
        <div class="cpl-actions">
          <!-- macOS：显式区分版本（Apple 芯片 / Intel），不用下拉 -->
          <template v-if="archSplit(client)">
            <el-button size="small" type="primary" @click="download(client, 'apple')">
              Apple 芯片<el-tag size="small" type="success" effect="plain" class="cpl-arch">ARM</el-tag>
            </el-button>
            <el-button size="small" type="primary" plain @click="download(client, 'intel')">
              Intel<el-tag size="small" type="info" effect="plain" class="cpl-arch">x64</el-tag>
            </el-button>
          </template>
          <el-button v-else size="small" type="primary" @click="download(client)">
            下载
          </el-button>
          <el-button size="small" plain @click="openTutorial(client)">教程</el-button>
        </div>
      </div>
    </div>
    <el-empty v-else :description="`${platformLabel} 暂无客户端`" :image-size="60" />
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  CLIENT_PLATFORMS,
  clientsForPlatform,
  clientSupportsArchSplit,
} from '@/data/clientRegistry'
import { openClientDownload } from '@/utils/clientDownload'

const props = defineProps({
  softwareConfig: { type: Object, default: () => ({}) },
  // 默认展示哪个平台（仪表盘传「当前系统」）
  defaultPlatform: { type: String, default: 'windows' },
  // 是否包含自研客户端（客户端中心自己单独置顶展示 MoneyFly，故传 false）
  includeOfficial: { type: Boolean, default: true },
})

const router = useRouter()
const platforms = CLIENT_PLATFORMS

const clientsOf = (key) => clientsForPlatform(key).filter((c) => props.includeOfficial || !c.official)

// 只展示有客户端的平台（iOS 未配置时不会出现空栏）
const visiblePlatforms = computed(() => platforms.filter((p) => clientsOf(p.key).length > 0))

const pickDefault = () => {
  const keys = visiblePlatforms.value.map((p) => p.key)
  return keys.includes(props.defaultPlatform) ? props.defaultPlatform : (keys[0] || 'windows')
}
const activePlatform = ref(pickDefault())
const platformLabel = computed(
  () => platforms.find((p) => p.key === activePlatform.value)?.label || '该平台'
)
const clients = computed(() => clientsOf(activePlatform.value))

// macOS 双架构：Apple 芯片与 Intel 是两个不同的安装包，必须分别给按钮
const archSplit = (client) => clientSupportsArchSplit(client, activePlatform.value)

const download = async (client, arch = null) => {
  await openClientDownload(client, {
    softwareConfig: props.softwareConfig,
    os: activePlatform.value,
    arch,
  })
}

const openTutorial = (client) => {
  // 交给客户端中心：带上平台与客户端，落地即定位到对应教程
  router.push({ path: '/tutorials', query: { client: client.id, platform: activePlatform.value } })
}
</script>

<style scoped>
.client-platform-list { width: 100%; }

.cpl-platforms {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}
.cpl-platform {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.3;
  cursor: pointer;
  transition: background-color 0.16s ease, border-color 0.16s ease, color 0.16s ease;
}
.cpl-platform:hover { border-color: var(--el-color-primary-light-5); color: var(--el-color-primary); }
.cpl-platform.is-active {
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
  font-weight: 600;
}
.cpl-count {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.cpl-platform.is-active .cpl-count { color: var(--el-color-primary); }

.cpl-rows { display: flex; flex-direction: column; gap: 10px; }
.cpl-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
}
.cpl-info { min-width: 0; }
.cpl-name {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.cpl-desc {
  margin-top: 2px;
  font-size: 12px;
  line-height: 1.4;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}
.cpl-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.cpl-actions :deep(.el-button) { margin-left: 0; }
.cpl-arch { margin-left: 4px; }

/* 窄屏：按钮换行铺满，避免两个下载按钮 + 教程挤成一行溢出（实测 320px 会挤） */
@media (max-width: 768px) {
  .cpl-row { flex-direction: column; align-items: stretch; gap: 10px; }
  .cpl-actions { flex-wrap: wrap; }
  .cpl-actions :deep(.el-button) { flex: 1 1 auto; }
}
</style>
