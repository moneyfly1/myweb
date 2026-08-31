<template>
  <div class="list-container help-container">
    <div class="breadcrumb">首页 / 帮助中心</div>
    <div class="page-header">
      <div class="page-title">
        <h1>帮助中心</h1>
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
      <el-card class="clients-card help-section-card" id="clients">
        <template #header>
          <div class="card-header">
            客户端下载
          </div>
        </template>
        <div class="help-client-grid">
          <div
            v-for="client in clients"
            :key="client.id"
            class="client-row"
          >
            <div class="client-title">
              <div class="client-name-line">
                <el-icon class="client-icon"><component :is="getClientIcon(client.icon)" /></el-icon>
                <span>{{ client.name }}</span>
              </div>
              <div class="client-tags">
                <el-tag
                  v-for="platform in client.platforms"
                  :key="platform"
                  size="small"
                >
                  {{ platform }}
                </el-tag>
                <el-tag v-if="clientVersion(client)" type="success" size="small">
                  v{{ clientVersion(client) }}
                </el-tag>
              </div>
              <div class="item-meta">{{ client.description }}</div>
            </div>
            <div class="button-row client-actions">
              <el-dropdown
                v-if="hasMacArchChoice(client)"
                trigger="click"
                @command="(key) => downloadClient(client, key)"
              >
                <el-button type="primary" size="small">
                  下载
                  <el-icon><ArrowDown /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item :command="macArmKey(client)">
                      <span class="client-download-option">
                        {{ client.name }}（Apple 芯片）
                        <el-tag size="small" type="success" effect="plain">ARM</el-tag>
                      </span>
                    </el-dropdown-item>
                    <el-dropdown-item :command="macIntelKey(client)">
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
                @click="downloadClient(client)"
              >
                下载
              </el-button>
              <el-button
                size="small"
                @click="openClientGuide(client.id)"
              >
                教程
              </el-button>
            </div>
          </div>
        </div>
      </el-card>
      <el-card class="guide-card help-section-card" id="guide">
        <template #header>
          <div class="card-header">
            使用指南
          </div>
        </template>
        <el-collapse v-model="activeNames">
          <el-collapse-item 
            v-for="guide in guides" 
            :key="guide.id"
            :title="guide.title"
            :name="guide.id"
          >
            <div class="guide-content" v-html="guide.content"></div>
          </el-collapse-item>
        </el-collapse>
      </el-card>
      <el-card class="faq-card help-section-card" id="faq">
        <template #header>
          <div class="card-header">
            常见问题
          </div>
        </template>
        <el-collapse v-model="activeFAQ">
          <el-collapse-item 
            v-for="faq in faqs" 
            :key="faq.id"
            :title="faq.question"
            :name="faq.id"
          >
            <div class="faq-content" v-html="faq.answer"></div>
          </el-collapse-item>
        </el-collapse>
      </el-card>
      <el-card class="client-guides-card help-section-card" id="client-guides">
        <template #header>
          <div class="card-header">
            客户端安装教程
          </div>
        </template>
        <el-collapse v-model="activeClientGuides">
          <el-collapse-item
            v-for="client in clients"
            :key="client.id"
            :name="client.id"
            :id="`client-guide-${client.id}`"
          >
            <template #title>
              <span class="client-guide-title">{{ client.name }}</span>
            </template>
            <div class="client-guide-actions">
              <el-button type="primary" size="small" @click.stop="downloadClient(client)">下载 {{ client.name }}</el-button>
            </div>
            <div class="guide-content" v-html="sanitizeHtml(client.guide)"></div>
          </el-collapse-item>
        </el-collapse>
      </el-card>
      <el-card class="contact-card help-section-card" id="contact">
        <template #header>
          <div class="card-header">
            联系我们
          </div>
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
          <div class="contact-item">
            <el-icon class="contact-icon"><Clock /></el-icon>
            <div class="contact-details">
              <h4>服务时间</h4>
              <p>周一至周日 9:00-22:00</p>
            </div>
          </div>
        </div>
      </el-card>
      </main>
    </div>
  </div>
</template>
<script>
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Cellphone,
  ChatDotRound,
  Clock,
  Document,
  Download,
  Guide,
  Iphone,
  Menu,
  Message,
  Monitor,
  QuestionFilled,
  Service
} from '@element-plus/icons-vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { safeOpen } from '@/utils/safeOpen'
import { sanitizeBasicHtml } from '@/utils/sanitizeHtml'
import { resolvePanDownloadUrl, pickConfiguredUrl } from '@/utils/githubDownload'
import { cachedAPI } from '@/utils/api'
export default {
  name: 'Help',
  setup() {
    const route = useRoute()
    const router = useRouter()
    const sanitizeHtml = sanitizeBasicHtml
    const activeNames = ref([])
    const activeFAQ = ref([])
    const activeClientGuides = ref([])
    const contactEmail = ref('')
    const contactQQ = ref('')
    const softwareConfig = ref({})
    const loadContactInfo = async () => {
      try {
        const api = (await import('@/utils/api')).default
        const response = await api.get('/settings/public-settings')
        if (response && response.data) {
          let settings = null
          if (response.data.success !== false) {
            settings = response.data.data || response.data
          } else {
            settings = response.data
          }
          if (settings) {
            if (settings.support_email !== undefined && settings.support_email !== null) {
              const email = String(settings.support_email).trim()
              if (email !== '') {
                contactEmail.value = email
              }
            }
            if (settings.support_qq !== undefined && settings.support_qq !== null) {
              const qq = String(settings.support_qq).trim()
              if (qq !== '') {
                contactQQ.value = qq
              }
            }
          }
        }
      } catch (error) {
        console.error('获取联系信息失败:', error)
      }
    }
    const sections = [
      { id: 'clients', title: '客户端下载', icon: Download },
      { id: 'guide', title: '使用指南', icon: Guide },
      { id: 'faq', title: '常见问题', icon: QuestionFilled },
      { id: 'client-guides', title: '安装教程', icon: Document },
      { id: 'contact', title: '联系我们', icon: Service }
    ]
    const guides = [
      {
        id: 'guide-1',
        title: '注册与登录',
        content: `
          <div>
            <ol>
              <li>使用邮箱注册账户，按页面提示完成验证。</li>
              <li>登录后进入仪表盘查看余额、订阅、设备和订单入口。</li>
              <li>忘记密码时在登录页发起找回，按邮件提示重置密码。</li>
            </ol>
          </div>
        `
      },
      {
        id: 'guide-2',
        title: '购买套餐',
        content: `
          <div>
            <ol>
              <li>进入套餐购买页面，选择普通套餐或自定义套餐。</li>
              <li>确认设备数量、周期、优惠券和支付方式。</li>
              <li>支付完成后，套餐订单会进入订单记录统一管理。</li>
            </ol>
          </div>
        `
      },
      {
        id: 'guide-3',
        title: '导入订阅',
        content: `
          <div>
            <ol>
              <li>进入订阅管理，复制对应客户端的订阅地址。</li>
              <li>在客户端中选择从 URL 导入或扫描二维码。</li>
              <li>需要隐藏某些协议时，在订阅管理中使用协议排除。</li>
            </ol>
          </div>
        `
      },
      {
        id: 'guide-4',
        title: '设备管理',
        content: `
          <div>
            <ol>
              <li>进入设备管理查看当前设备、可用设备和最近访问记录。</li>
              <li>移除不再使用的设备，释放可用设备数量。</li>
              <li>设备数量不足时，可从设备管理或仪表盘升级设备数量。</li>
            </ol>
          </div>
        `
      },
      {
        id: 'guide-5',
        title: '充值与订单',
        content: `
          <div>
            <ol>
              <li>在仪表盘点击充值，选择金额和支付方式。</li>
              <li>余额充值、套餐购买、设备升级都会进入订单记录。</li>
              <li>待支付订单可在订单记录中继续支付或取消。</li>
            </ol>
          </div>
        `
      }
    ]
    const faqs = [
      {
        id: 'faq-1',
        question: '订阅地址可以分享给他人吗？',
        answer: `
          <div><p>不建议分享。订阅地址和账户权益绑定，分享后可能占用设备数量。发现泄露时，请在订阅管理中重置订阅地址。</p></div>
        `
      },
      {
        id: 'faq-2',
        question: '订单在哪里查看？',
        answer: `
          <div><p>套餐购买、升级设备数量和账户充值都会进入订单记录。订单记录中可查看状态、金额、时间，并对待支付订单继续支付。</p></div>
        `
      },
      {
        id: 'faq-3',
        question: '设备数量不够怎么办？',
        answer: `
          <div><p>可以先在设备管理中移除不用的设备；仍不够时，点击升级设备数量并完成支付。</p></div>
        `
      },
      {
        id: 'faq-4',
        question: '支持哪些客户端？',
        answer: `
          <div><p>页面提供 Clash for Windows、Clash Verge、Hiddify、V2rayN、V2rayNG、Shadowrocket、FlClash、Clash Part 等客户端的下载和教程入口。</p></div>
        `
      },
      {
        id: 'faq-5',
        question: '如何修改密码或邮箱？',
        answer: `
          <div><p>进入个人资料或用户设置，按页面表单填写当前密码、新密码或邮箱验证码，保存后系统会同步账号状态。</p></div>
        `
      }
    ]
    const baseClients = [
      {
        id: 'clash-windows',
        aliases: ['clash_windows', 'clash-for-windows'],
        name: 'Clash for Windows',
        description: 'Windows 平台 Clash 客户端',
        icon: 'desktop',
        platforms: ['Windows'],
        githubKey: null,
        downloadKeys: ['clash_windows_url'],
        osKeys: { windows: ['clash_windows_url'] },
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div>
            <ol>
              <li>下载并安装 Clash for Windows。</li>
              <li>复制订阅管理中的对应订阅地址。</li>
              <li>在 Clash for Windows 中选择从 URL 导入或添加订阅。</li>
              <li>保存配置后启用连接，需要更新时刷新订阅。</li>
            </ol>
          </div>
        `
      },
      {
        id: 'clash-party',
        name: 'Clash Part',
        description: 'Windows/Mac/Linux 平台功能强大的 Clash 客户端',
        icon: 'desktop',
        platforms: ['Windows', 'Mac', 'Linux'],
        githubKey: 'clash-party',
        downloadKeys: ['clash_party_windows_url', 'clash_party_macos_arm_url', 'clash_party_macos_url', 'clash_windows_url'],
        osKeys: { windows: ['clash_party_windows_url'], macos: ['clash_party_macos_arm_url', 'clash_party_macos_url'] },
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div>
            <ol>
              <li>下载并安装 Clash Part。</li>
              <li>复制订阅管理中的对应订阅地址。</li>
              <li>在 Clash Part 中选择从 URL 导入或添加订阅。</li>
              <li>保存配置后启用连接，需要更新时刷新订阅。</li>
            </ol>
          </div>
        `
      },
      {
        id: 'clash-verge',
        aliases: ['clash-verge-rev'],
        name: 'Clash Verge',
        description: 'Windows/Mac/Linux平台优秀的代理客户端，界面现代化',
        icon: 'desktop',
        platforms: ['Windows', 'Mac', 'Linux'],
        githubKey: 'clash-verge',
        downloadKeys: ['clash_verge_windows_url', 'clash_verge_macos_arm_url', 'clash_verge_macos_url'],
        osKeys: { windows: ['clash_verge_windows_url'], macos: ['clash_verge_macos_arm_url', 'clash_verge_macos_url'] },
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div>
            <ol>
              <li>下载并安装 Clash Verge。</li>
              <li>复制订阅管理中的对应订阅地址。</li>
              <li>在 Clash Verge 中选择从 URL 导入或添加订阅。</li>
              <li>保存配置后启用连接，需要更新时刷新订阅。</li>
            </ol>
          </div>
        `
      },
      {
        id: 'clash-meta',
        aliases: ['clash-android'],
        name: 'Clash Meta',
        description: 'Android 平台代理客户端',
        icon: 'android',
        platforms: ['Android'],
        githubKey: null,
        downloadKeys: ['clash_android_url'],
        osKeys: { android: ['clash_android_url'] },
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div>
            <ol>
              <li>下载并安装 Clash Meta。</li>
              <li>复制订阅管理中的对应订阅地址。</li>
              <li>在 Clash Meta 中选择从 URL 导入或添加订阅。</li>
              <li>保存配置后启用连接，需要更新时刷新订阅。</li>
            </ol>
          </div>
        `
      },
      {
        id: 'hiddify',
        name: 'Hiddify',
        description: '跨平台代理客户端，支持Windows/Mac/Android',
        icon: 'desktop',
        platforms: ['Windows', 'Mac', 'Android'],
        githubKey: 'hiddify',
        downloadKeys: ['hiddify_windows_url', 'hiddify_android_url', 'hiddify_macos_arm_url', 'hiddify_macos_url'],
        osKeys: { windows: ['hiddify_windows_url'], android: ['hiddify_android_url'], macos: ['hiddify_macos_arm_url', 'hiddify_macos_url'] },
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div>
            <ol>
              <li>下载并安装 Hiddify。</li>
              <li>复制订阅管理中的对应订阅地址。</li>
              <li>在 Hiddify 中选择从 URL 导入或添加订阅。</li>
              <li>保存配置后启用连接，需要更新时刷新订阅。</li>
            </ol>
          </div>
        `
      },
      {
        id: 'flclash',
        name: 'FlClash',
        description: 'Windows/Mac/Android平台Flutter开发的代理客户端',
        icon: 'desktop',
        platforms: ['Windows', 'Mac', 'Android'],
        githubKey: 'flclash',
        downloadKeys: ['flash_windows_url', 'flash_android_url', 'flash_macos_arm_url', 'flash_macos_url'],
        osKeys: { windows: ['flash_windows_url'], android: ['flash_android_url'], macos: ['flash_macos_arm_url', 'flash_macos_url'] },
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div>
            <ol>
              <li>下载并安装 FlClash。</li>
              <li>复制订阅管理中的对应订阅地址。</li>
              <li>在 FlClash 中选择从 URL 导入或添加订阅。</li>
              <li>保存配置后启用连接，需要更新时刷新订阅。</li>
            </ol>
          </div>
        `
      },
      {
        id: 'v2rayn',
        name: 'V2rayN',
        description: 'Windows平台轻量级代理客户端，资源占用低',
        icon: 'desktop',
        platforms: ['Windows'],
        githubKey: 'v2rayn',
        downloadKeys: ['v2rayn_url', 'v2rayn_macos_arm_url', 'v2rayn_macos_url'],
        osKeys: { windows: ['v2rayn_url'], macos: ['v2rayn_macos_arm_url', 'v2rayn_macos_url'] },
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div>
            <ol>
              <li>下载并安装 V2rayN。</li>
              <li>复制订阅管理中的对应订阅地址。</li>
              <li>在 V2rayN 中选择从 URL 导入或添加订阅。</li>
              <li>保存配置后启用连接，需要更新时刷新订阅。</li>
            </ol>
          </div>
        `
      },
      {
        id: 'v2rayng',
        name: 'V2rayNG',
        description: 'Android平台轻量级代理客户端，简单易用',
        icon: 'android',
        platforms: ['Android'],
        githubKey: 'v2rayng',
        downloadKeys: ['v2rayng_url'],
        osKeys: { android: ['v2rayng_url'] },
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div>
            <ol>
              <li>下载并安装 V2rayNG。</li>
              <li>复制订阅管理中的对应订阅地址。</li>
              <li>在 V2rayNG 中选择从 URL 导入或添加订阅。</li>
              <li>保存配置后启用连接，需要更新时刷新订阅。</li>
            </ol>
          </div>
        `
      },
      {
        id: 'shadowrocket',
        name: 'Shadowrocket',
        description: 'iOS平台优秀的代理客户端，界面简洁，操作便捷',
        icon: 'ios',
        platforms: ['iOS'],
        githubKey: null,
        downloadKeys: ['shadowrocket_url'],
        osKeys: { ios: ['shadowrocket_url'] },
        downloadUrl: 'https://apps.apple.com/app/shadowrocket/id932747118',
        guideUrl: '#',
        guide: `
          <div>
            <ol>
              <li>下载并安装 Shadowrocket。</li>
              <li>复制订阅管理中的对应订阅地址。</li>
              <li>在 Shadowrocket 中选择从 URL 导入或添加订阅。</li>
              <li>保存配置后启用连接，需要更新时刷新订阅。</li>
            </ol>
          </div>
        `
      }
    ]
    const clientIconMap = {
      desktop: Monitor,
      android: Cellphone,
      mobile: Cellphone,
      ios: Iphone
    }
    const getClientIcon = (icon) => clientIconMap[icon] || Monitor
    const clients = computed(() => baseClients)
    const clientAliasMap = computed(() => {
      const map = new Map()
      clients.value.forEach(client => {
        map.set(client.id, client.id)
        ;(client.aliases || []).forEach(alias => map.set(alias, client.id))
      })
      return map
    })
    const normalizeClientId = (clientId) => clientAliasMap.value.get(String(clientId || '').trim()) || ''
    const getConfiguredDownloadUrl = (client) => {
      return pickConfiguredUrl(client.downloadKeys, softwareConfig.value || {}, null, client.osKeys)
    }
    // hasMacArchChoice 判断客户端是否有 macOS 双架构（Apple 芯片 / Intel）可拆分
    const hasMacArchChoice = (client) => {
      const macKeys = (client.osKeys?.macos) || []
      return macKeys.some(k => /arm/i.test(k)) && macKeys.some(k => !/arm/i.test(k))
    }
    // macArmKey 返回 macOS Apple 芯片下载配置键（优先 *_macos_arm_url）
    const macArmKey = (client) => {
      const macKeys = (client.osKeys?.macos) || []
      return macKeys.find(k => /arm/i.test(k)) || ''
    }
    // macIntelKey 返回 macOS Intel 下载配置键（优先 *_macos_url）
    const macIntelKey = (client) => {
      const macKeys = (client.osKeys?.macos) || []
      return macKeys.find(k => !/arm/i.test(k)) || ''
    }
    const softwareVersions = ref({})
    const clientVersion = (client) => {
      const key = (client.downloadKeys || []).find(k => {
        const v = softwareConfig.value?.[k]
        return v && String(v).trim()
      })
      return key ? softwareVersions.value[key] || '' : ''
    }
    const openClientGuide = async (clientId, { updateUrl = true } = {}) => {
      const normalizedId = normalizeClientId(clientId)
      if (!normalizedId) return
      if (!activeClientGuides.value.includes(normalizedId)) {
        activeClientGuides.value = [...activeClientGuides.value, normalizedId]
      }
      if (updateUrl && route.query.client !== normalizedId) {
        router.replace({ path: '/help', query: { ...route.query, client: normalizedId } })
      }
      await nextTick()
      const element = document.getElementById(`client-guide-${normalizedId}`) || document.getElementById('client-guides')
      if (element) {
        element.scrollIntoView({ behavior: 'smooth', block: 'start' })
      }
    }
    const applyRouteClientGuide = () => {
      const clientId = route.query.client || (route.hash ? route.hash.replace(/^#/, '') : '')
      if (clientId) {
        openClientGuide(clientId, { updateUrl: false })
      }
    }
    const loadSoftwareConfig = async () => {
      try {
        const response = await cachedAPI.getSoftwareConfig()
        if (response.data?.success !== false) {
          softwareConfig.value = response.data?.data || response.data || {}
        }
      } catch (error) {
        softwareConfig.value = {}
      }
      try {
        const vRes = await fetch('/api/v1/software/versions')
        const vData = await vRes.json()
        const map = {}
        ;(vData?.data?.list || []).forEach(item => {
          map[item.key] = item.version
        })
        softwareVersions.value = map
      } catch (error) {
        // 版本信息可选，失败静默
      }
    }
    const scrollToSection = (sectionId) => {
      const element = document.getElementById(sectionId)
      if (element) {
        element.scrollIntoView({ behavior: 'smooth' })
      }
    }
    const downloadClient = async (client, archKey = null) => {
      try {
        // archKey 非空 = 用户显式选择架构（如 *_macos_arm_url / *_macos_url），只查该键
        const configuredUrl = archKey
          ? String(softwareConfig.value?.[archKey] || '').trim()
          : getConfiguredDownloadUrl(client)
        if (configuredUrl) {
          safeOpen(resolvePanDownloadUrl(configuredUrl))
          ElMessage.success('已打开下载页面')
          return
        }
        if (client.name === 'Shadowrocket') {
          safeOpen(client.downloadUrl || 'https://apps.apple.com/app/shadowrocket/id932747118')
          return
        }
        if (client.githubKey) {
          ElMessage.info('正在获取最新下载链接...')
          const { getClientDownloadUrl, getClientReleasesUrl } = await import('@/utils/githubDownload')
          try {
            const forcedArch = archKey && /arm/i.test(archKey) ? 'apple' : null
            const downloadUrl = await getClientDownloadUrl(client.githubKey, softwareConfig.value || {}, forcedArch)
            const isAccelerated = downloadUrl.includes('ghproxy.com') || downloadUrl.includes('ghproxy.net')
            const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
            if (isMobile) {
              safeOpen(downloadUrl)
              ElMessage.success(isAccelerated ? '已打开下载页面（已启用加速）' : '已打开下载页面')
            } else {
              const link = document.createElement('a')
              link.href = downloadUrl
              link.download = '' // 让浏览器自动识别文件名
              link.target = '_blank'
              link.rel = 'noopener noreferrer'
              link.style.display = 'none'
              document.body.appendChild(link)
              link.click()
              setTimeout(() => {
                if (document.body.contains(link)) {
                  document.body.removeChild(link)
                }
              }, 200)
              ElMessage.success(isAccelerated ? '开始下载（已启用加速）...' : '开始下载...')
            }
          } catch (error) {
            console.error('获取下载链接失败:', error)
            ElMessage.error('获取下载链接失败: ' + (error.message || '未知错误'))
            try {
              const { getClientReleasesUrl } = await import('@/utils/githubDownload')
              const releasesUrl = getClientReleasesUrl(client.githubKey)
              if (releasesUrl) {
                setTimeout(() => {
                  safeOpen(releasesUrl)
                  ElMessage.warning('已打开发布页面，请手动选择下载')
                }, 1000)
              } else {
                ElMessage.error('无法获取下载链接')
              }
            } catch (err) {
              console.error('获取发布页面失败:', err)
              ElMessage.error('无法获取下载链接，请稍后重试')
            }
          }
        } else if (client.downloadUrl) {
          const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
          if (isMobile) {
            safeOpen(client.downloadUrl)
            ElMessage.success('已打开下载页面')
          } else {
            const link = document.createElement('a')
            link.href = client.downloadUrl
            link.download = ''
            link.target = '_blank'
            link.rel = 'noopener noreferrer'
            link.style.display = 'none'
            document.body.appendChild(link)
            link.click()
            setTimeout(() => {
              if (document.body.contains(link)) {
                document.body.removeChild(link)
              }
            }, 200)
            ElMessage.success('开始下载...')
          }
        } else {
          ElMessage.error('下载链接未配置')
        }
      } catch (error) {
        console.error('下载失败:', error)
        ElMessage.error('下载失败: ' + (error.message || '请稍后重试'))
      }
    }
    onMounted(async () => {
      await Promise.all([loadContactInfo(), loadSoftwareConfig()])
      applyRouteClientGuide()
    })
    watch(() => [route.query.client, route.hash], () => {
      applyRouteClientGuide()
    })
    const sanitizedGuides = computed(() => {
      return guides.map(guide => ({
        ...guide,
        content: sanitizeHtml(guide.content)
      }))
    })
    const sanitizedFaqs = computed(() => {
      return faqs.map(faq => ({
        ...faq,
        answer: sanitizeHtml(faq.answer)
      }))
    })
    return {
      activeNames,
      activeFAQ,
      sections,
      guides: sanitizedGuides,
      faqs: sanitizedFaqs,
      clients,
      contactEmail,
      contactQQ,
      activeClientGuides,
      scrollToSection,
      downloadClient,
      openClientGuide,
      getClientIcon,
      clientVersion,
      sanitizeHtml
    }
  }
}
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
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  @media (max-width: 768px) {
    margin-top: 0 !important;
    margin-bottom: 0.75rem;
  }
}
.page-header :is(h1) {
  margin: 0;
  color: #303133;
  font-size: 22px;
  line-height: 1.25;
  @media (max-width: 768px) {
    font-size: 1.25rem;
  }
}
.page-header :is(p) {
  margin: 6px 0 0;
  color: #606266;
  line-height: 1.5;
  @media (max-width: 768px) {
    font-size: 0.8125rem;
  }
}
.help-content {
  display: grid;
  gap: 14px;
  min-width: 0;
  @media (max-width: 768px) {
    gap: 0.75rem;
  }
}






.guide-card,
.faq-card,
.clients-card,
.client-guides-card,
.contact-card {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  :deep(.el-card__header) {
    padding: 14px 16px;
    border-bottom: 1px solid #ebeef5;
    font-size: 16px;
    font-weight: 700;
  }
  :deep(.el-card__body) {
    padding: 16px;
  }
  @media (max-width: 768px) {
    :deep(.el-card__header) {
      padding: 10px 12px;
      font-size: 0.875rem;
    }
    :deep(.el-card__body) {
      padding: 10px 12px;
    }
  }
}

.guide-content,
.faq-content {
  color: #333;
  font-size: 0.875rem;
  line-height: 1.7;
}

.guide-content :deep(p),
.faq-content :deep(p) {
  margin: 0.75rem 0;
  color: #374151;
  line-height: 1.7;
}

.guide-content :deep(p:has(> strong:only-child)),
.faq-content :deep(p:has(> strong:only-child)) {
  margin-top: 0.25rem;
  margin-bottom: 0.625rem;
}

.guide-content :deep(p:has(> strong:first-child):not(:has(+ ol)):not(:has(+ ul))),
.faq-content :deep(p:has(> strong:first-child):not(:has(+ ol)):not(:has(+ ul))) {
  padding: 10px 12px;
  background: #f0f9ff;
  border-left: 3px solid #1677ff;
  border-radius: 6px;
}

.guide-content :deep(p:has(> strong:first-child):has(ul)),
.faq-content :deep(p:has(> strong:first-child):has(ul)) {
  padding: 10px 12px;
  background: #f8fafc;
  border-left: 3px solid #94a3b8;
  border-radius: 6px;
}

.guide-content :deep(h3),
.faq-content :deep(h3) {
  margin: 0 0 14px;
  color: #409eff;
  font-size: 1.1rem;
  font-weight: 700;
  line-height: 1.35;
}

.guide-content :deep(h4),
.faq-content :deep(h4) {
  margin: 18px 0 10px;
  color: #1f2937;
  font-size: 0.98rem;
  font-weight: 700;
  line-height: 1.4;
}

.guide-content :deep(strong),
.faq-content :deep(strong) {
  color: #1f2937;
  font-weight: 700;
}

.guide-content :deep(pre),
.faq-content :deep(pre) {
  margin: 0.75rem 0;
  padding: 10px 12px;
  overflow-x: auto;
  background: #f5f7fa;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.85em;
}

.guide-content :deep(code),
.faq-content :deep(code) {
  padding: 2px 4px;
  background: #f0f2f5;
  border-radius: 4px;
  color: #b45309;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}
.client-guide-title {
  color: #1f2937;
  font-weight: 600;
}

.client-guide-actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 12px;
}

.guide-content :deep(ol),
.faq-content :deep(ol) {
  padding-left: 1.25rem;
  margin: 0.5rem 0;
}
.guide-content :deep(ul),
.faq-content :deep(ul) {
  padding-left: 1.25rem;
  margin: 0.5rem 0;
}
.guide-content :deep(li),
.faq-content :deep(li) {
  margin-bottom: 0.375rem;
  font-size: 0.875rem;
  line-height: 1.5;
}




.client-icon {
  width: 42px;
  height: 42px;
  border-radius: 8px;
  background: #ecf5ff;
  color: #409eff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  @media (max-width: 768px) {
    width: 40px;
    height: 40px;
  }
}

.contact-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}






.client-actions {
  display: flex;
  flex-direction: row;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
  @media (max-width: 768px) {
    gap: 0.25rem;
  }
}
.contact-info {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.75rem;
  @media (max-width: 768px) {
    gap: 0.5rem;
  }
}
.contact-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  background: #f8f9fa;
  border-radius: 6px;
  @media (max-width: 768px) {
    padding: 0.625rem;
    gap: 0.5rem;
  }
}
.contact-icon {
  font-size: 1.5rem;
  color: #409eff;
  width: 40px;
  height: 40px;
  text-align: center;
  @media (max-width: 768px) {
    font-size: 1.25rem;
    width: 40px;
    height: 40px;
  }
}
.contact-details h4 {
  margin: 0 0 0.25rem 0;
  color: #333;
  font-weight: 600;
  font-size: 0.9375rem;
  @media (max-width: 768px) {
    font-size: 0.875rem;
  }
}
.contact-details :is(p) {
  margin: 0;
  color: #666;
  font-size: 0.8125rem;
  @media (max-width: 768px) {
    font-size: 0.75rem;
  }
}
@media (max-width: 768px) {
  .help-container {
    padding: 12px;
  }
  .page-header {
    margin-bottom: 1rem;
    padding: 0 4px;
    :is(h1) {
      font-size: 1.5rem;
      margin-bottom: 0.5rem;
    }
    :is(p) {
      font-size: 0.875rem;
      line-height: 1.5;
    }
  }
  .help-content {
    gap: 1rem;
  }
  .nav-card,
  .guide-card,
  .faq-card,
  .clients-card,
  .client-guides-card,
  .contact-card {
    margin-bottom: 0.75rem;
    border-radius: 8px;
    border: 1px solid #dcdfe6;
    :deep(.el-card__header) {
      padding: 14px 16px;
      font-size: 0.9375rem;
      border-bottom: 1px solid #f0f0f0;
    }
    :deep(.el-card__body) {
      padding: 16px;
    }
  }
  .card-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 0.9375rem;
    font-weight: 600;
  }


  .guide-content,
  .faq-content {
    font-size: 0.875rem;
    line-height: 1.7;

    :deep(ol),
    :deep(ul) {
      padding-left: 1.25rem;
      margin: 0.75rem 0;
    }

    :deep(li) {
      margin-bottom: 0.5rem;
      line-height: 1.6;
    }

    :deep(p) {
      margin: 0.75rem 0;
      line-height: 1.7;
    }

    :deep(h3),
    :deep(h4) {
      font-size: 1rem;
      margin-top: 1rem;
      margin-bottom: 0.75rem;
    }

    :deep(pre) {
      background: #f5f7fa;
      padding: 10px;
      border-radius: 6px;
      overflow-x: auto;
      font-family: monospace;
      font-size: 0.85em;
      border: 1px solid #dcdfe6;
      margin: 0.75rem 0;
    }

    :deep(code) {
      background: #f0f2f5;
      padding: 2px 4px;
      border-radius: 4px;
      color: #e6a23c;
      font-family: monospace;
    }
  }


  .client-icon {
    width: 44px;
    margin-bottom: 0;
    margin-right: 12px;
    flex-shrink: 0;
    display: flex;
    justify-content: center;
    align-items: center;
    height: 44px; /* 保持图标区域正方形 */
  }


  .client-actions {
    flex-direction: column; /* 按钮垂直排列 */
    justify-content: center;
    gap: 8px;
    width: auto;
    margin-top: 0;
    margin-left: 8px;
    flex-shrink: 0;
    :deep(.el-button) {
      flex: none;
      width: 70px;
      padding: 8px 0;
      font-size: 0.8125rem;
      border-radius: 6px;
      margin: 0;
      min-height: 44px;
      touch-action: manipulation;
      &:active {
        background: var(--el-color-primary-light-9, #ecf5ff);
      }
    }
  }
  .contact-info {
    grid-template-columns: 1fr;
    gap: 12px;
  }
  .contact-item {
    flex-direction: column;
    text-align: center;
    padding: 16px;
    border-radius: 8px;
    background: #ffffff;
    border: 1px solid #dcdfe6;
    .contact-icon {
      font-size: 2rem;
      width: auto;
      height: auto;
      margin-bottom: 12px;
    }
  }
  .contact-details {
    width: 100%;
    :is(h4) {
      font-size: 1rem;
      margin-bottom: 8px;
      font-weight: 600;
    }
    :is(p) {
      font-size: 0.875rem;
      line-height: 1.5;
      color: #666;
    }
  }
  .help-container :deep(.el-collapse) {
    border: none;
    .el-collapse-item {
      border-bottom: 1px solid #f0f0f0;
      margin-bottom: 8px;
      &:last-child {
        border-bottom: none;
        margin-bottom: 0;
      }
    }
    .el-collapse-item__header {
      padding: 12px 0;
      font-size: 0.9375rem;
      font-weight: 500;
      color: #333;
    }
    .el-collapse-item__content {
      padding: 12px 0 16px 0;
    }
  }
}
.help-container :deep(.client-guide-dialog) {
  @media (max-width: 768px) {
    .el-dialog {
      width: 92% !important;
      margin: 4vh auto !important;
      max-height: calc(100dvh - 8vh);
    }
    .el-dialog__header {
      padding: 16px;
      border-bottom: 1px solid #f0f0f0;
      .el-dialog__title {
        font-size: 1.125rem;
        font-weight: 600;
      }
    }
    .el-dialog__body {
      max-height: calc(100dvh - 8vh - 124px);
      overflow-y: auto;
      padding: 16px;
      -webkit-overflow-scrolling: touch;
    }
    .el-dialog__footer {
      padding: 12px 16px max(14px, env(safe-area-inset-bottom));
      border-top: 1px solid #f0f0f0;
      .el-button {
        min-height: 44px;
        padding: 10px 20px;
        font-size: 0.875rem;
      }
    }
  }
  .guide-dialog-content {
    line-height: 1.8;
    color: #333;
    :is(h3) {
      color: #409eff;
      margin-bottom: 15px;
      font-size: 1.25rem;
      font-weight: 600;
    }
    :is(h4) {
      color: #333;
      margin-top: 20px;
      margin-bottom: 10px;
      font-size: 1.1rem;
      font-weight: 600;
    }
    :is(ol), :is(ul) {
      padding-left: 1.5rem;
      margin: 10px 0;
      :is(li) {
        margin-bottom: 8px;
        line-height: 1.6;
      }
    }
    :is(p) {
      margin: 10px 0;
    }
    @media (max-width: 768px) {
      font-size: 0.875rem;
      line-height: 1.7;
      :is(h3) {
        font-size: 1.125rem;
        margin-bottom: 12px;
      }
      :is(h4) {
        font-size: 1rem;
        margin-top: 16px;
        margin-bottom: 8px;
      }
      :is(ol), :is(ul) {
        padding-left: 1.5rem;
        margin: 8px 0;
        :is(li) {
          margin-bottom: 6px;
          line-height: 1.6;
        }
      }
      :is(p) {
        margin: 8px 0;
        line-height: 1.7;
      }
    }
  }
}

.help-container {
  .help-quick-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 14px;
    margin-bottom: 0;
  }

  .guide-card,
  .faq-card,
  .clients-card,
  .client-guides-card,
  .contact-card {
    border: 1px solid #dcdfe6 !important;
    border-radius: 8px !important;
    background: #fff !important;
    box-shadow: none !important;
    overflow: hidden;

    :deep(.el-card__header) {
      padding: 14px 16px !important;
      border-bottom: 1px solid #ebeef5 !important;
      background: #fff !important;
    }

    :deep(.el-card__body) {
      padding: 16px !important;
    }
  }

  .card-header {
    display: flex !important;
    align-items: center !important;
    justify-content: space-between !important;
    gap: 12px !important;
    color: #303133 !important;
    font-size: 16px !important;
    font-weight: 700 !important;
    line-height: 1.3 !important;
  }

  .help-client-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0 16px;
  }

  .client-row {
    display: grid !important;
    grid-template-columns: minmax(0, 1fr) auto !important;
    align-items: center !important;
    gap: 12px !important;
    padding: 14px 0 !important;
    border: 0 !important;
    border-bottom: 1px solid #ebeef5 !important;
    border-radius: 0 !important;
    background: #fff !important;
    box-shadow: none !important;
  }

  .client-row:nth-last-child(-n + 2) {
    border-bottom: 0 !important;
  }

  .client-title {
    min-width: 0;
    color: #303133;
    font-size: 15px;
    font-weight: 700;
    line-height: 1.45;
  }

  .client-tags {
    display: inline-flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-left: 8px;
    vertical-align: middle;
  }

  .item-meta {
    margin-top: 4px;
    color: #909399;
    font-size: 13px;
    font-weight: 400;
    line-height: 1.45;
  }

  .client-actions {
    display: flex !important;
    flex-wrap: nowrap !important;
    gap: 8px !important;
    justify-content: flex-end !important;
  }

  .client-actions .el-button {
    min-width: 58px;
    margin-left: 0 !important;
  }

  .el-collapse {
    border-top: 0;
    border-bottom: 0;
  }

  .el-collapse-item:last-child {
    :deep(.el-collapse-item__wrap) {
      border-bottom: 0;
    }
  }
}

@media (max-width: 768px) {
  .help-container {
    .help-quick-grid,
    .help-client-grid {
      grid-template-columns: 1fr !important;
    }

    .client-row {
      grid-template-columns: 1fr !important;
      align-items: stretch !important;
      gap: 10px !important;
      padding: 14px 12px !important;
      overflow: hidden !important;
    }

    .client-row:nth-last-child(-n + 2) {
      border-bottom: 1px solid #ebeef5 !important;
    }

    .client-row:last-child {
      border-bottom: 0 !important;
    }

    .client-actions,
    .client-actions .el-button {
      width: 100% !important;
    }

    .client-actions {
      display: grid !important;
      grid-template-columns: repeat(2, minmax(0, 1fr)) !important;
      gap: 8px !important;
    }

    .client-actions .el-button {
      min-width: 0 !important;
      margin: 0 !important;
    }
  }
}

/* Final help center layout */
.help-layout {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  gap: 14px;
  align-items: start;
}
.help-nav-card {
  position: sticky;
  top: 76px;
  display: grid;
  gap: 8px;
  padding: 10px;
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
}
.help-nav-button {
  width: 100%;
  min-height: 42px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  color: #606266;
  font: inherit;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.help-nav-button:hover {
  background: #f5f7fa;
  border-color: #ebeef5;
  color: #409eff;
}

.help-section-card {
  scroll-margin-top: 86px;
}
.help-section-card :deep(.el-card__header) {
  padding: 14px 16px !important;
  background: #f5f7fa !important;
}
.help-section-card :deep(.el-card__body) {
  padding: 0 !important;
}
.help-section-card .card-header {
  color: #303133 !important;
  font-size: 16px !important;
  font-weight: 700 !important;
}
.help-client-grid {
  display: grid !important;
  grid-template-columns: repeat(2, minmax(0, 1fr)) !important;
  gap: 0 !important;
  min-width: 0;
  max-width: 100%;
}
.client-row {
  display: grid !important;
  grid-template-columns: minmax(0, 1fr) auto !important;
  align-items: center !important;
  gap: 12px !important;
  padding: 16px !important;
  min-width: 0 !important;
  max-width: 100% !important;
  border: 0 !important;
  border-right: 1px solid #ebeef5 !important;
  border-bottom: 1px solid #ebeef5 !important;
  background: #fff !important;
}
.client-title {
  min-width: 0 !important;
  max-width: 100% !important;
}
.client-row:nth-child(2n) {
  border-right: 0 !important;
}
.client-row:nth-last-child(-n + 2) {
  border-bottom: 0 !important;
}
.client-name-line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: #303133;
  font-size: 15px;
  font-weight: 700;
}
.client-name-line span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.client-icon {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 8px;
  background: #ecf5ff;
  color: #409eff;
}
.client-tags {
  display: flex !important;
  flex-wrap: wrap;
  gap: 6px;
  margin: 8px 0 0 !important;
}
.client-row .item-meta {
  margin-top: 6px;
  color: #909399;
  font-size: 13px;
  line-height: 1.45;
}
.client-actions {
  display: flex !important;
  flex-direction: column;
  gap: 8px !important;
  min-width: 0 !important;
}
.client-actions .el-button {
  width: 72px;
  margin: 0 !important;
}
.guide-card :deep(.el-collapse),
.faq-card :deep(.el-collapse),
.client-guides-card :deep(.el-collapse) {
  border: 0;
}
.guide-card :deep(.el-collapse-item__header),
.faq-card :deep(.el-collapse-item__header),
.client-guides-card :deep(.el-collapse-item__header) {
  height: auto;
  min-height: 48px;
  padding: 0 16px;
  color: #303133;
  font-weight: 600;
  line-height: 1.45;
}
.guide-card :deep(.el-collapse-item__content),
.faq-card :deep(.el-collapse-item__content),
.client-guides-card :deep(.el-collapse-item__content) {
  padding: 0 16px 16px;
}
.client-guide-actions {
  justify-content: flex-start;
}
.contact-info {
  grid-template-columns: repeat(3, minmax(0, 1fr)) !important;
  gap: 0 !important;
}
.contact-item {
  padding: 16px !important;
  border-right: 1px solid #ebeef5;
  border-radius: 0 !important;
  background: #fff !important;
}
.contact-item:last-child {
  border-right: 0;
}
@media (max-width: 980px) {
  .help-layout {
    grid-template-columns: 1fr;
  }
  .help-nav-card {
    position: static;
    grid-template-columns: repeat(5, minmax(0, 1fr));
  }
  .help-nav-button {
    justify-content: center;
    padding: 0 8px;
    text-align: center;
  }
}
@media (max-width: 768px) {
  .help-nav-card {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .help-client-grid {
    grid-template-columns: 1fr !important;
  }
  .client-row,
  .client-row:nth-child(2n),
  .client-row:nth-last-child(-n + 2) {
    grid-template-columns: 1fr !important;
    width: 100% !important;
    min-width: 0 !important;
    max-width: 100% !important;
    padding: 14px 12px !important;
    overflow: hidden !important;
    border-right: 0 !important;
    border-bottom: 1px solid #ebeef5 !important;
  }
  .client-row:last-child {
    border-bottom: 0 !important;
  }
  .client-actions {
    display: grid !important;
    grid-template-columns: repeat(2, minmax(0, 1fr)) !important;
    width: 100% !important;
  }
  .client-actions .el-button {
    width: auto !important;
    min-width: 0 !important;
    margin: 0 !important;
  }
  .contact-info {
    grid-template-columns: 1fr !important;
  }
  .contact-item {
    border-right: 0;
    border-bottom: 1px solid #ebeef5;
  }
  .contact-item:last-child {
    border-bottom: 0;
  }
}

/* 客户端下载下拉：macOS 架构选项 */
.client-download-option {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}
.client-download-option .el-tag {
  margin-left: 2px;
}
</style>
