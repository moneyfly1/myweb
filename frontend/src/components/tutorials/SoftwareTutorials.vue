<template>
  <div class="list-container tutorial-container">
    <div class="breadcrumb">首页 / 客户端中心</div>
    <div class="page-header">
      <div class="page-title">
        <h1>客户端中心</h1>
        <p>按系统选择客户端下载，并查看对应的安装与导入订阅教程。</p>
      </div>
    </div>

    <div class="card tutorial-card">
      <el-tabs v-model="activeTab" class="tutorial-tabs">
        <el-tab-pane
          v-for="platform in platforms"
          :key="platform.key"
          :label="platform.label"
          :name="platform.key"
        >
          <!-- 自研客户端置顶推荐（已配置下载地址时展示） -->
          <div v-if="moneyflyVisible && moneyflySupports(platform.key)" class="official-block">
            <div class="official-head">
              <span class="official-name">{{ moneyflyBrand.name }}</span>
              <el-tag type="success" effect="dark" size="small">{{ moneyflyBrand.badge }}</el-tag>
              <el-tag type="danger" effect="plain" size="small">推荐优先使用</el-tag>
              <span v-if="moneyflyConfig.version" class="official-version">v{{ moneyflyConfig.version }}</span>
            </div>
            <p class="official-intro">{{ moneyflyDescription }}</p>
            <MoneyFlyDownloadPanel
              :software-config="softwareConfig"
              plain
              hide-head
              :only-platform="moneyflyPlatformKeys(platform.key)"
            />
            <div class="official-actions">
              <el-button
                size="small"
                :loading="tutorialLoading === 'moneyfly'"
                @click="toggleTutorial('moneyfly')"
              >
                {{ isTutorialOpen('moneyfly') ? '收起教程' : '查看使用教程' }}
              </el-button>
              <router-link to="/subscription">
                <el-button size="small" type="primary" plain>获取订阅地址</el-button>
              </router-link>
            </div>
            <div v-if="isTutorialOpen('moneyfly')" class="tutorial-body">
              <TutorialContent :state="tutorialState('moneyfly')" />
            </div>
          </div>

          <!-- 第三方客户端列表（来自客户端注册表） -->
          <div class="client-grid tutorial-client-grid">
            <div v-for="client in clientsOf(platform.key)" :key="client.id" class="client-row-block">
              <div class="client-row">
                <div class="client-info">
                  <div class="client-title">{{ client.name }}</div>
                  <div class="item-meta">{{ client.description }}</div>
                </div>
                <div class="button-row">
                  <!-- macOS 双架构：拆成 Apple 芯片 / Intel 两个选项 -->
                  <el-dropdown
                    v-if="clientSupportsArchSplit(client, platform.key)"
                    trigger="click"
                    @command="(arch) => download(client, platform.key, arch)"
                  >
                    <el-button type="primary" size="small">
                      下载<el-icon><ArrowDown /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="apple">
                          <span class="client-download-option">
                            {{ client.name }}（Apple 芯片）
                            <el-tag size="small" type="success" effect="plain">ARM</el-tag>
                          </span>
                        </el-dropdown-item>
                        <el-dropdown-item command="intel">
                          <span class="client-download-option">
                            {{ client.name }}（Intel）
                            <el-tag size="small" type="info" effect="plain">x64</el-tag>
                          </span>
                        </el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                  <el-button
                    v-else
                    type="primary"
                    size="small"
                    @click="download(client, platform.key)"
                  >
                    下载
                  </el-button>
                  <el-button
                    size="small"
                    :loading="tutorialLoading === client.id"
                    @click="toggleTutorial(client.id)"
                  >
                    {{ isTutorialOpen(client.id) ? '收起教程' : '教程' }}
                  </el-button>
                </div>
              </div>
              <div v-if="isTutorialOpen(client.id)" class="tutorial-body">
                <TutorialContent :state="tutorialState(client.id)" />
              </div>
            </div>
          </div>

          <el-alert
            v-if="!clientsOf(platform.key).length && !(moneyflyVisible && moneyflySupports(platform.key))"
            type="info"
            show-icon
            :closable="false"
            title="该平台暂无客户端"
          />
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup>
/**
 * 客户端中心（路由 /tutorials）
 *
 * 全站唯一的客户端下载入口：客户端清单来自 data/clientRegistry.js（唯一数据源），
 * 教程正文来自知识库（后台可维护），本页不再硬编码任何教程文案。
 * 兼容旧深链 /tutorials?client=xxx 与 /help?client=xxx（由路由层重定向而来）。
 */
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowDown } from '@element-plus/icons-vue'
import { cachedAPI } from '@/utils/api'
import { openClientDownload } from '@/utils/clientDownload'
import {
  CLIENT_PLATFORMS,
  clientsForPlatform,
  clientSupportsArchSplit,
  getClientById,
  normalizeClientId,
} from '@/data/clientRegistry'
import { loadTutorialContent } from '@/utils/knowledge'
import { MONEYFLY_BRAND, isMoneyflyVisible, readMoneyflyConfig } from '@/utils/moneyflyClient'
import MoneyFlyDownloadPanel from '@/components/moneyfly/MoneyFlyDownloadPanel.vue'
import TutorialContent from '@/components/tutorials/TutorialContent.vue'

const route = useRoute()

const activeTab = ref('windows')
const softwareConfig = ref({})
const platforms = CLIENT_PLATFORMS

const moneyflyBrand = MONEYFLY_BRAND
const moneyflyConfig = computed(() => readMoneyflyConfig(softwareConfig.value || {}))
const moneyflyVisible = computed(() => isMoneyflyVisible(moneyflyConfig.value))
const moneyflyDescription = computed(() => moneyflyConfig.value.note || moneyflyBrand.intro)
const moneyflySupports = (platformKey) => ['windows', 'macos', 'android'].includes(platformKey)
const moneyflyPlatformKeys = (platformKey) =>
  platformKey === 'macos' ? 'macos_arm,macos_intel' : platformKey

// 客户端列表：注册表里排除自研（自研单独置顶展示）
const clientsOf = (platformKey) => clientsForPlatform(platformKey).filter(c => !c.official)

// ===== 教程正文（按需加载）=====
const tutorialOpen = ref(new Set())
const tutorialCache = ref({})
const tutorialLoading = ref('')

const isTutorialOpen = (id) => tutorialOpen.value.has(id)
const tutorialState = (id) => tutorialCache.value[id] || { status: 'loading' }

async function ensureTutorial(client) {
  if (!client || tutorialCache.value[client.id]) return
  tutorialLoading.value = client.id
  tutorialCache.value = { ...tutorialCache.value, [client.id]: { status: 'loading' } }
  try {
    const tutorial = await loadTutorialContent(client)
    tutorialCache.value = {
      ...tutorialCache.value,
      [client.id]: tutorial
        ? { status: 'ready', title: tutorial.title, summary: tutorial.summary, content: tutorial.content }
        : { status: 'missing' },
    }
  } catch {
    tutorialCache.value = { ...tutorialCache.value, [client.id]: { status: 'missing' } }
  } finally {
    tutorialLoading.value = ''
  }
}

async function toggleTutorial(id) {
  const client = getClientById(id)
  const next = new Set(tutorialOpen.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
    await ensureTutorial(client)
  }
  tutorialOpen.value = next
}

// ===== 下载 =====
const download = async (client, platformKey, arch = null) => {
  await openClientDownload(client, {
    softwareConfig: softwareConfig.value,
    os: platformKey,
    arch,
  })
}

// ===== 初始化：系统识别 + 深链 =====
function detectPlatformTab() {
  const ua = (navigator.userAgent || '').toLowerCase()
  if (ua.includes('android')) return 'android'
  if (/iphone|ipad|ipod/.test(ua)) return 'ios'
  if (ua.includes('mac os') || ua.includes('macintosh')) return 'macos'
  return 'windows'
}

function applyClientQuery() {
  const id = normalizeClientId(route.query.client)
  const client = id ? getClientById(id) : null
  if (!client) return
  activeTab.value = client.platforms[0]
  toggleTutorial(client.id)
}

onMounted(async () => {
  activeTab.value = detectPlatformTab()
  try {
    const res = await cachedAPI.getSoftwareConfig()
    if (res?.data?.success !== false) softwareConfig.value = res?.data?.data || {}
  } catch {
    softwareConfig.value = {}
  }
  applyClientQuery()
})

watch(() => route.query.client, () => applyClientQuery())
</script>

<style scoped>
.tutorial-container {
  padding-bottom: 24px;
}
.page-title p {
  margin: 6px 0 0;
  color: #909399;
  font-size: 13px;
}
.official-block {
  border: 1px solid var(--el-color-primary-light-5);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 18px;
  background: linear-gradient(180deg, var(--el-color-primary-light-9), transparent 70%);
}
.official-head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.official-version {
  font-size: 13px;
  font-weight: 400;
  color: var(--el-text-color-secondary);
}
.official-intro {
  margin: 8px 0 12px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--el-text-color-regular);
}
.official-actions {
  margin-top: 12px;
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  align-items: center;
}
.client-row-block {
  border-bottom: 1px solid #f0f2f5;
  padding: 12px 0;
}
.client-row-block:last-child {
  border-bottom: 0;
}
.client-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.client-info {
  min-width: 0;
}
.tutorial-body {
  margin-top: 12px;
  padding: 14px 16px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  background: #fafafa;
}
@media (max-width: 768px) {
  .client-row {
    align-items: flex-start;
  }
  .button-row {
    width: 100%;
  }
}
</style>
