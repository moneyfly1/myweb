<template>
  <div class="list-container help-container">
    <div class="breadcrumb">首页 / 帮助中心</div>
    <div class="page-header">
      <div class="page-title">
        <h1>帮助中心</h1>
        <p>常见问题自助查询、常用功能入口与客服联系方式。软件下载与安装教程请前往「客户端中心」。</p>
      </div>
    </div>

    <div class="help-layout">
      <aside class="help-nav-card">
        <button
          v-for="section in sections"
          :key="section.id"
          type="button"
          class="help-nav-button"
          @click="scrollToSection(section.id)"
        >
          <el-icon><component :is="section.icon" /></el-icon>
          <span>{{ section.title }}</span>
        </button>
      </aside>

      <main class="help-content">
        <!-- 1. 自研客户端推荐 -->
        <el-card v-if="moneyflyVisible" class="moneyfly-help-card help-section-card" id="moneyfly">
          <template #header>
            <div class="card-header">
              {{ moneyflyBrand.name }} 自研客户端
              <el-tag type="success" effect="dark" size="small">{{ moneyflyBrand.badge }}</el-tag>
              <el-tag type="danger" effect="plain" size="small">推荐优先使用</el-tag>
            </div>
          </template>
          <p class="moneyfly-help-summary">{{ moneyflyDescription }}</p>
          <MoneyFlyDownloadPanel :software-config="softwareConfig" plain hide-head />
          <div class="help-actions">
            <el-button type="primary" size="small" @click="goClientTutorial('moneyfly')">
              查看 {{ moneyflyBrand.name }} 使用教程
            </el-button>
            <el-button size="small" plain @click="goSubscription">获取订阅地址</el-button>
          </div>
        </el-card>

        <!-- 2. 快速入口 -->
        <el-card class="quick-card help-section-card" id="quick">
          <template #header>
            <div class="card-header">常用功能入口</div>
          </template>
          <div class="quick-grid">
            <button
              v-for="entry in quickEntries"
              :key="entry.path"
              type="button"
              class="quick-item"
              @click="safeNavigate(entry.path)"
            >
              <el-icon class="quick-icon"><component :is="entry.icon" /></el-icon>
              <div class="quick-text">
                <div class="quick-title">{{ entry.title }}</div>
                <div class="quick-desc">{{ entry.desc }}</div>
              </div>
            </button>
          </div>
        </el-card>

        <!-- 3. 高频问题（来自知识库） -->
        <el-card class="faq-card help-section-card" id="faq">
          <template #header>
            <div class="card-header">
              常见问题
              <el-button type="primary" link size="small" @click="goKnowledge">查看全部</el-button>
            </div>
          </template>
          <div v-if="faqLoading" class="faq-loading">
            <el-icon class="is-loading"><Loading /></el-icon>
            <span>正在加载…</span>
          </div>
          <el-collapse v-else-if="faqArticles.length" v-model="activeFAQ">
            <el-collapse-item
              v-for="article in faqArticles"
              :key="article.id"
              :title="article.title"
              :name="String(article.id)"
            >
              <div class="faq-content" v-html="sanitizeHtml(article.content)"></div>
            </el-collapse-item>
          </el-collapse>
          <el-empty v-else description="暂无常见问题内容" :image-size="70" />
          <div v-if="!faqLoading && faqArticles.length" class="faq-more">
            <el-button type="primary" link size="small" @click="goKnowledge">
              还有疑问？到知识库搜索 →
            </el-button>
          </div>
        </el-card>

        <!-- 4. 联系我们 -->
        <el-card class="contact-card help-section-card" id="contact">
          <template #header>
            <div class="card-header">联系我们</div>
          </template>
          <div class="contact-info">
            <div class="contact-item" v-if="contactEmail">
              <el-icon class="contact-icon"><Message /></el-icon>
              <div class="contact-details">
                <h4>售后邮箱</h4>
                <p>{{ contactEmail }}</p>
              </div>
            </div>
            <div class="contact-item" v-if="contactQQ">
              <el-icon class="contact-icon"><ChatDotRound /></el-icon>
              <div class="contact-details">
                <h4>售后联系方式</h4>
                <p>{{ contactQQ }}</p>
              </div>
            </div>
            <div v-if="contactHours" class="contact-item">
              <el-icon class="contact-icon"><Clock /></el-icon>
              <div class="contact-details">
                <h4>服务时间</h4>
                <p>{{ contactHours }}</p>
              </div>
            </div>
          </div>
          <div class="help-actions">
            <el-button type="primary" size="small" @click="safeNavigate('/tickets')">
              提交工单
            </el-button>
          </div>
        </el-card>
      </main>
    </div>
  </div>
</template>

<script setup>
/**
 * 帮助中心（路由 /help）—— 自助服务台
 *
 * 定位调整：这里只做「高频问答 + 快捷入口 + 客服联系方式」。
 * 客户端下载与安装教程统一收敛到「客户端中心」(/tutorials)，
 * 长文内容统一收敛到「知识库」(/knowledge)，避免同一份教程在三个页面重复维护。
 *
 * 兼容：旧深链 /help?client=xxx 会自动跳转到 /tutorials?client=xxx
 * （仪表盘与历史收藏里存在该链接）。
 */
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Cellphone,
  ChatDotRound,
  Clock,
  Document,
  Download,
  Iphone,
  Loading,
  Message,
  Money,
  QuestionFilled,
  Service,
  Star,
  Ticket,
  Wallet
} from '@element-plus/icons-vue'
import { sanitizeArticleHtml } from '@/utils/sanitizeHtml'
import { safeNavigate } from '@/utils/safeOpen'
import api, { cachedAPI } from '@/utils/api'
import { loadArticles } from '@/utils/knowledge'
import { MONEYFLY_BRAND, isMoneyflyVisible, readMoneyflyConfig } from '@/utils/moneyflyClient'
import MoneyFlyDownloadPanel from '@/components/moneyfly/MoneyFlyDownloadPanel.vue'

const route = useRoute()
const router = useRouter()

const sanitizeHtml = sanitizeArticleHtml
const softwareConfig = ref({})
const contactEmail = ref('')
const contactQQ = ref('')
// 客服服务时间：由后台「系统设置 → 服务时间」配置，未配置时不显示该行
const contactHours = ref('')

const moneyflyBrand = MONEYFLY_BRAND
const moneyflyConfig = computed(() => readMoneyflyConfig(softwareConfig.value || {}))
const moneyflyVisible = computed(() => isMoneyflyVisible(moneyflyConfig.value))
const moneyflyDescription = computed(() => moneyflyConfig.value.note || moneyflyBrand.intro)

const sections = computed(() => {
  const list = []
  if (moneyflyVisible.value) {
    list.push({ id: 'moneyfly', title: `${MONEYFLY_BRAND.name} 自研客户端`, icon: Star })
  }
  list.push(
    { id: 'quick', title: '常用功能入口', icon: Document },
    { id: 'faq', title: '常见问题', icon: QuestionFilled },
    { id: 'contact', title: '联系我们', icon: Service }
  )
  return list
})

// 常用功能入口：把用户最常找的页面集中到一个地方，减少到处翻菜单
const quickEntries = [
  { path: '/subscription', title: '订阅管理', desc: '查看订阅地址、二维码与设备数', icon: Wallet },
  { path: '/devices', title: '设备管理', desc: '查看在线设备、移除不用的设备', icon: Cellphone },
  { path: '/tutorials', title: '客户端中心', desc: '软件下载与安装教程', icon: Download },
  { path: '/knowledge', title: '知识库', desc: '使用指南、进阶教程与政策说明', icon: QuestionFilled },
  { path: '/tickets', title: '工单中心', desc: '遇到问题提交工单给我们', icon: Ticket },
  { path: '/packages', title: '套餐购买', desc: '购买或续费套餐、升级设备数', icon: Money }
]

const activeFAQ = ref([])
const faqArticles = ref([])
const faqLoading = ref(true)

async function loadFaq() {
  faqLoading.value = true
  try {
    // 常见问题统一来自知识库「常见问题」分类（后台可维护，改文案无需重新部署）
    const { items } = await loadArticles({ categoryName: '常见问题', pageSize: 8 })
    faqArticles.value = items
  } catch {
    faqArticles.value = []
  } finally {
    faqLoading.value = false
  }
}

const scrollToSection = (sectionId) => {
  const element = document.getElementById(sectionId)
  if (element) element.scrollIntoView({ behavior: 'smooth' })
}

const goSubscription = () => router.push('/subscription')
const goKnowledge = () => router.push('/knowledge')
const goClientTutorial = (clientId) => router.push({ path: '/tutorials', query: { client: clientId } })

// 旧契约：/help?client=xxx 跳到客户端中心对应教程
function redirectLegacyClientQuery() {
  if (route.query.client) {
    router.replace({ path: '/tutorials', query: { client: route.query.client } })
  }
}

onMounted(async () => {
  redirectLegacyClientQuery()

  const [configResult, settingsResult] = await Promise.allSettled([
    cachedAPI.getSoftwareConfig(),
    api.get('/settings/public-settings')
  ])

  if (configResult.status === 'fulfilled') {
    const data = configResult.value?.data
    if (data?.success !== false) softwareConfig.value = data?.data || {}
  }

  if (settingsResult?.status === 'fulfilled' && settingsResult.value?.data) {
    const settings = settingsResult.value.data.data || settingsResult.value.data || {}
    if (settings.support_email) contactEmail.value = String(settings.support_email).trim()
    if (settings.support_qq) contactQQ.value = String(settings.support_qq).trim()
    if (settings.support_hours) contactHours.value = String(settings.support_hours).trim()
  }

  await loadFaq()
})

watch(() => route.query.client, () => redirectLegacyClientQuery())
</script>

<style scoped lang="scss">
.help-container {
  padding: 0;
  max-width: none;
  margin: 0;
  width: 100%;
  @media (max-width: 768px) {
    padding-top: 0 !important;
    margin-top: 0 !important;
  }
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
  text-align: left;
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 1px 6px rgba(0, 0, 0, 0.05);
}
.page-title h1 {
  margin: 0;
  font-size: 20px;
  color: #303133;
}
.page-title p {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: #909399;
}
.help-layout {
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 16px;
  align-items: start;
}
.help-nav-card {
  position: sticky;
  top: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px;
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 1px 6px rgba(0, 0, 0, 0.05);
}
.help-nav-button {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 9px 10px;
  font-size: 13.5px;
  color: #303133;
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.2s, color 0.2s;
}
.help-nav-button:hover {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}
.help-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.help-section-card {
  border-radius: 10px;
}
.help-section-card .card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}
.help-section-card .card-header .el-button {
  margin-left: auto;
}
.moneyfly-help-card {
  border: 1px solid var(--el-color-primary-light-5);
  background: linear-gradient(180deg, var(--el-color-primary-light-9), transparent 55%);
}
.moneyfly-help-summary {
  margin: 0 0 14px;
  font-size: 14px;
  line-height: 1.8;
  color: #303133;
}
.help-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 14px;
}
.quick-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 12px;
}
.quick-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 14px;
  text-align: left;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  cursor: pointer;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.quick-item:hover {
  border-color: var(--el-color-primary-light-5);
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.06);
}
.quick-icon {
  font-size: 18px;
  color: var(--el-color-primary);
  margin-top: 2px;
}
.quick-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}
.quick-desc {
  margin-top: 3px;
  font-size: 12.5px;
  line-height: 1.6;
  color: #909399;
}
.faq-loading {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 0;
  font-size: 13px;
  color: #909399;
}
.faq-content {
  font-size: 13.5px;
  line-height: 1.85;
  color: #303133;
}
.faq-content :deep(h2),
.faq-content :deep(h3) {
  margin: 10px 0 6px;
  font-size: 14px;
}
.faq-content :deep(ol),
.faq-content :deep(ul) {
  margin: 6px 0;
  padding-left: 20px;
}
.faq-more {
  margin-top: 10px;
  text-align: right;
}
.contact-info {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}
.contact-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 14px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
}
.contact-icon {
  font-size: 18px;
  color: var(--el-color-primary);
  margin-top: 2px;
}
.contact-details h4 {
  margin: 0 0 4px;
  font-size: 13.5px;
  color: #303133;
}
.contact-details p {
  margin: 0;
  font-size: 13px;
  color: #606266;
  word-break: break-all;
}

@media (max-width: 768px) {
  .help-layout {
    grid-template-columns: 1fr;
  }
  .help-nav-card {
    position: static;
    flex-direction: row;
    flex-wrap: wrap;
  }
  .help-nav-button {
    width: auto;
  }
  .quick-grid,
  .contact-info {
    grid-template-columns: 1fr;
  }
}
</style>
