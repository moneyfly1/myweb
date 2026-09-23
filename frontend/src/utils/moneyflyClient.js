/**
 * MoneyFly 自研客户端（官方推荐）—— 唯一真相源
 *
 * 平台定义、配置键、系统识别、下载链接解析集中在此，
 * 供 用户仪表盘 / 帮助中心 / 软件教程 / 后台配置页 共用，
 * 避免同一份平台列表在多处复制粘贴后失配。
 *
 * 配置存储：system_configs(category='software')，由后台"配置管理 → 软件下载配置"维护，
 * 复用现有 UpdateSoftwareConfig 接口，无需任何数据库变更。
 */
import { detectSystem, resolvePanDownloadUrl } from '@/utils/githubDownload'

export const MONEYFLY_BRAND = {
  name: 'MoneyFly',
  badge: '官方自研',
  tagline: '官方自研客户端 · 登录即用 · 一键连接',
  intro: 'MoneyFly 是本站自主研发的官方客户端，专为本站线路优化，界面简洁、连接稳定。用本站账号登录后点一下「连接」就能用 —— 不需要导入订阅、不需要复制任何链接，推荐优先选择。',
  // 用户端展示的亮点（帮助中心 / 客户端中心共用）
  highlights: [
    { title: '官方自研', desc: '与本站线路深度适配，节点延迟与稳定性表现更好' },
    { title: '登录即用', desc: '用本站账号登录后点「连接」即可，无需导入订阅或填写服务器' },
    { title: '自动同步', desc: '节点增删、套餐变更自动同步，无需反复手动更新' },
    { title: '多端支持', desc: '支持 Windows、macOS（Apple 芯片 / Intel）与 Android' },
  ],
}

// MONEYFLY_PLATFORMS 顺序即界面展示顺序：Windows → macOS(Apple) → macOS(Intel) → Android
const MONEYFLY_PLATFORMS = [
  {
    key: 'windows',
    configKey: 'moneyfly_windows_url',
    os: 'windows',
    label: 'Windows',
    archLabel: '',
    tag: '',
    hint: 'Windows 10 / 11（64 位）',
  },
  {
    key: 'macos_arm',
    configKey: 'moneyfly_macos_arm_url',
    os: 'macos',
    arch: 'apple',
    label: 'macOS',
    archLabel: 'Apple 芯片',
    tag: 'ARM',
    hint: 'M 系列芯片（M1/M2/M3/M4）',
  },
  {
    key: 'macos_intel',
    configKey: 'moneyfly_macos_url',
    os: 'macos',
    arch: 'intel',
    label: 'macOS',
    archLabel: 'Intel 芯片',
    tag: 'x64',
    hint: 'Intel 处理器',
  },
  {
    key: 'android',
    configKey: 'moneyfly_android_url',
    os: 'android',
    label: 'Android',
    archLabel: '',
    tag: '',
    hint: 'Android 8.0 及以上',
  },
]

// readMoneyflyConfig 从 software-config 原始配置解析出 MoneyFly 客户端展示模型。
// enabled 默认开启（字段缺失时视为开启），仅当显式配置为 false/0/off 才关闭。
export function readMoneyflyConfig(softwareConfig = {}) {
  const cfg = softwareConfig || {}
  const enabledRaw = cfg.moneyfly_enabled
  const enabled = !(
    enabledRaw === false ||
    enabledRaw === 0 ||
    ['false', '0', 'off', 'no'].includes(String(enabledRaw ?? '').trim().toLowerCase())
  )

  const items = MONEYFLY_PLATFORMS.map(platform => {
    const url = String(cfg[platform.configKey] ?? '').trim()
    return { ...platform, url, configured: !!url }
  })

  return {
    enabled,
    version: String(cfg.moneyfly_version ?? '').trim(),
    note: String(cfg.moneyfly_note ?? '').trim(),
    items,
    configuredCount: items.filter(i => i.configured).length,
    hasAny: items.some(i => i.configured),
  }
}

// isMoneyflyVisible 判断是否需要在用户端展示 MoneyFly 推荐区：
// 未启用、或一个下载地址都没配置时整体隐藏，避免出现点不动的空按钮。
export function isMoneyflyVisible(moneyfly) {
  return !!moneyfly && moneyfly.enabled && moneyfly.hasAny
}

// detectMoneyflyPlatform 识别访问者当前设备对应的平台定义，无法识别时返回 null。
function detectMoneyflyPlatform() {
  let detected
  try {
    detected = detectSystem()
  } catch {
    return null
  }
  return (
    MONEYFLY_PLATFORMS.find(p => p.os === detected.os && (!p.arch || p.arch === detected.arch)) || null
  )
}

// detectMoneyflyPlatformKey 识别访问者当前设备应下载哪个包，用于"推荐给你的设备"高亮。
// 返回 configKey（如 moneyfly_windows_url）；无法识别时返回 ''（此时不做高亮）。
export function detectMoneyflyPlatformKey() {
  const platform = detectMoneyflyPlatform()
  return platform ? platform.configKey : ''
}

// resolveMoneyflyUrl 把配置值转成可打开的地址：
// pan://<key> 走后端网盘/加速直链解析，普通 http(s) 原样返回。
export function resolveMoneyflyUrl(url) {
  const value = String(url ?? '').trim()
  if (!value) return ''
  return resolvePanDownloadUrl(value)
}
