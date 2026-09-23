/**
 * 客户端注册表 —— 全站客户端信息的唯一数据源（Single Source of Truth）
 *
 * 此前客户端信息散落在 4 个文件里各自维护，新增一个客户端要改 4 处：
 *   - frontend/src/components/tutorials/tutorialsData.js（教程正文）
 *   - frontend/src/views/Help.vue（baseClients：下载键 + 教程）
 *   - frontend/src/views/Dashboard.vue（platforms + clientKeyMap）
 *   - frontend/src/views/admin/Config.vue（下载地址表单字段）
 * 现在统一到本文件：平台归属、下载配置键（含 macOS 双架构）、GitHub 兜底、
 * 教程文章绑定、推荐位、别名（深链兼容）都从这里读。
 *
 * 教程正文不在这里，而是存放在知识库（后台可维护），用 tutorialTitle 绑定。
 */

// 平台定义（顺序即界面展示顺序）
export const CLIENT_PLATFORMS = [
  { key: 'windows', label: 'Windows', icon: 'desktop' },
  { key: 'macos', label: 'macOS', icon: 'mac' },
  { key: 'android', label: 'Android', icon: 'android' },
  { key: 'ios', label: 'iOS', icon: 'ios' },
]

// links 的键格式：
//   '<os>'            该平台通用下载键（按顺序取第一个已配置的）
//   '<os>:<arch>'     指定架构（macos 的 apple=Apple 芯片 / intel=Intel 芯片）
const CLIENT_LIST = [
  {
    id: 'moneyfly',
    name: 'MoneyFly',
    official: true, // 自研官方客户端（用户端置顶推荐）
    recommended: true,
    description: '本站自研官方客户端：登录账号后点「连接」即用，无需导入订阅',
    icon: 'desktop',
    platforms: ['windows', 'macos', 'android'],
    links: {
      windows: ['moneyfly_windows_url'],
      'macos:apple': ['moneyfly_macos_arm_url'],
      'macos:intel': ['moneyfly_macos_url'],
      android: ['moneyfly_android_url'],
    },
    tutorialTitle: 'MoneyFly 使用教程',
    // 品牌信息与展示文案由 utils/moneyflyClient.js 提供（介绍语、亮点等）
    moneyfly: true,
  },
  {
    id: 'clash-verge',
    name: 'Clash Verge',
    aliases: ['clash-verge-rev', 'clash-verge-windows', 'clash-verge-macos'],
    description: 'Windows / macOS 界面现代化的 Clash 客户端，功能完善',
    icon: 'desktop',
    githubKey: 'clash-verge',
    platforms: ['windows', 'macos'],
    links: {
      windows: ['clash_verge_windows_url'],
      'macos:apple': ['clash_verge_macos_arm_url'],
      'macos:intel': ['clash_verge_macos_url'],
    },
    tutorialTitle: 'Clash Verge 使用教程',
  },
  {
    id: 'clash-party',
    aliases: ['clash-party-windows', 'clash-party-macos'],
    name: 'Clash Part',
    description: 'Windows / macOS 功能强大的 Clash 客户端',
    icon: 'desktop',
    githubKey: 'clash-party',
    platforms: ['windows', 'macos'],
    links: {
      windows: ['clash_party_windows_url'],
      'macos:apple': ['clash_party_macos_arm_url'],
      'macos:intel': ['clash_party_macos_url'],
    },
    tutorialTitle: 'Clash Part 使用教程',
  },
  {
    id: 'clash-windows',
    name: 'Clash for Windows',
    aliases: ['clash_windows', 'clash-for-windows'],
    // Windows 7 专用：本站分发的是压缩包版（Win10/11 请引导用 Clash Verge / Clash Part）
    description: 'Windows 7 专用客户端（压缩包版）：先解压，运行「一键配置.bat」，再以管理员身份运行',
    icon: 'desktop',
    githubKey: null,
    platforms: ['windows'],
    links: { windows: ['clash_windows_url'] },
    tutorialTitle: 'Clash for Windows 使用教程',
  },
  {
    id: 'v2rayn',
    aliases: ['v2rayn_windows', 'v2rayn-macos'],
    name: 'V2rayN',
    description: 'Windows / macOS 老牌 V2Ray 客户端，协议支持全面',
    icon: 'desktop',
    githubKey: 'v2rayn',
    platforms: ['windows', 'macos'],
    links: {
      windows: ['v2rayn_url'],
      'macos:apple': ['v2rayn_macos_arm_url'],
      'macos:intel': ['v2rayn_macos_url'],
    },
    tutorialTitle: 'V2rayN 使用教程',
  },
  {
    id: 'hiddify',
    aliases: ['hiddify-windows', 'hiddify-macos', 'hiddify-android'],
    name: 'Hiddify',
    description: '跨平台开源客户端，支持 Windows / macOS / Android',
    icon: 'desktop',
    githubKey: 'hiddify',
    platforms: ['windows', 'macos', 'android'],
    links: {
      windows: ['hiddify_windows_url'],
      'macos:apple': ['hiddify_macos_arm_url'],
      'macos:intel': ['hiddify_macos_url'],
      android: ['hiddify_android_url'],
    },
    tutorialTitle: 'Hiddify 使用教程',
  },
  {
    id: 'flclash',
    aliases: ['flash-windows', 'flash-macos', 'flash-android', 'flclash-windows', 'flclash-macos', 'flclash-android'],
    name: 'FlClash',
    description: '基于 Flutter 的跨平台 Clash 客户端，界面简洁',
    icon: 'desktop',
    githubKey: 'flclash',
    platforms: ['windows', 'macos', 'android'],
    links: {
      windows: ['flash_windows_url'],
      'macos:apple': ['flash_macos_arm_url'],
      'macos:intel': ['flash_macos_url'],
      android: ['flash_android_url'],
    },
    tutorialTitle: 'FlClash 使用教程',
  },
  {
    id: 'clash-meta',
    name: 'Clash Meta',
    aliases: ['clash-android', 'clash-meta-android'],
    description: 'Android 平台 Clash Meta 内核客户端',
    icon: 'android',
    githubKey: null,
    platforms: ['android'],
    links: { android: ['clash_android_url'] },
    tutorialTitle: 'Clash Meta 使用教程',
  },
  {
    id: 'v2rayng',
    aliases: ['v2rayng-android'],
    name: 'V2rayNG',
    description: 'Android 平台经典 V2Ray 客户端',
    icon: 'android',
    githubKey: 'v2rayng',
    platforms: ['android'],
    links: { android: ['v2rayng_url'] },
    tutorialTitle: 'V2rayNG 使用教程',
  },
  {
    id: 'shadowrocket',
    name: 'Shadowrocket',
    description: 'iOS 平台客户端，需在 App Store 购买下载',
    icon: 'ios',
    githubKey: null,
    platforms: ['ios'],
    links: { ios: ['shadowrocket_url'] },
    // 未配置自定义地址时的兜底跳转（App Store）
    fallbackUrl: 'https://apps.apple.com/app/shadowrocket/id932747118',
    tutorialTitle: 'Shadowrocket 使用教程',
  },
]

// ===== 查询辅助 =====

export function getClientById(id) {
  if (!id) return null
  const key = String(id).trim().toLowerCase()
  return (
    CLIENT_LIST.find(c => c.id.toLowerCase() === key) ||
    CLIENT_LIST.find(c => (c.aliases || []).some(a => a.toLowerCase() === key)) ||
    null
  )
}
// normalizeClientId 把历史别名/深链参数归一化为注册表 id（用于 /help?client=xxx 兼容）
export function normalizeClientId(id) {
  const client = getClientById(id)
  return client ? client.id : ''
}

// clientsForPlatform 取某平台支持的客户端（registry 顺序即展示顺序）
export function clientsForPlatform(platformKey) {
  return CLIENT_LIST.filter(c => c.platforms.includes(platformKey))
}

// linkCandidates 返回某客户端在某系统/架构下的下载配置键候选（架构专用优先）
export function linkCandidates(client, os, arch) {
  if (!client || !client.links) return []
  const out = []
  if (os && arch && client.links[`${os}:${arch}`]) out.push(...client.links[`${os}:${arch}`])
  if (os && client.links[os]) out.push(...client.links[os])
  // 兜底：同平台另一架构（便于用户手动选择另一版本）
  if (os) {
    Object.keys(client.links)
      .filter(k => k.startsWith(`${os}:`))
      .forEach(k => out.push(...client.links[k]))
  }
  return [...new Set(out)]
}

// macArchKeys 返回 macOS 双架构键（用于把"下载"拆成 Apple 芯片 / Intel 两个选项）
function macArchKeys(client) {
  if (!client || !client.links) return { apple: [], intel: [] }
  return {
    apple: client.links['macos:apple'] || [],
    intel: client.links['macos:intel'] || [],
  }
}

// clientSupportsArchSplit 该客户端在当前平台是否需要让用户选架构
export function clientSupportsArchSplit(client, platformKey) {
  if (platformKey !== 'macos') return false
  const { apple, intel } = macArchKeys(client)
  return apple.length > 0 && intel.length > 0
}
