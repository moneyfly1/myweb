/**
 * 客户端下载统一逻辑
 *
 * 此前帮助中心、软件教程、仪表盘各写了一份"取配置地址 → 打开 / 回退 GitHub / 回退商店"
 * 的实现，行为不一致（macOS 架构处理、pan:// 解析、失败提示都各写各的）。
 * 现在统一到这里，三个页面调用同一个函数。
 */
import { ElMessage } from '@/utils/elementPlusServices'
import { safeOpen } from '@/utils/safeOpen'
import { resolvePanDownloadUrl, pickConfiguredUrl, getClientDownloadUrl, getClientReleasesUrl } from '@/utils/githubDownload'
import { detectSystem } from '@/utils/githubDownload'
import { linkCandidates } from '@/data/clientRegistry'

// findConfiguredUrl 取该客户端在当前系统/架构下已配置的下载地址（原样返回配置值）
export function findConfiguredUrl(client, { softwareConfig = {}, os, arch } = {}) {
  if (!client) return ''
  const system = os ? { os, arch } : detectSystem()
  const keys = linkCandidates(client, system.os, system.arch)
  if (!keys.length) return ''
  // 显式指定架构时只取该架构键，避免 pickConfiguredUrl 的自动架构重排
  if (arch && os) {
    const archKeys = linkCandidates(client, os, arch)
    if (archKeys.length) return pickConfiguredUrl(archKeys, softwareConfig, { os, arch })
  }
  return pickConfiguredUrl(keys, softwareConfig, system)
}

// openClientDownload 执行一次下载：
//   1) 后台配置的自定义地址（支持 pan:// 网盘间接链接）
//   2) GitHub 最新版本（仅第三方客户端有 githubKey 时）
//   3) 固定兜底地址（如 iOS 的 App Store）
//   4) 都没有 → 明确提示，不静默失败
export async function openClientDownload(client, { softwareConfig = {}, os, arch, silent = false } = {}) {
  if (!client) {
    if (!silent) ElMessage.error('未找到该客户端配置')
    return { ok: false, reason: 'unknown-client' }
  }

  const configured = findConfiguredUrl(client, { softwareConfig, os, arch })
  if (configured) {
    safeOpen(resolvePanDownloadUrl(configured))
    return { ok: true, source: 'configured', url: configured }
  }

  if (client.githubKey) {
    try {
      if (!silent) ElMessage.info('正在获取最新下载链接...')
      const url = await getClientDownloadUrl(client.githubKey, softwareConfig, arch === 'apple' ? 'apple' : null)
      if (url) {
        safeOpen(url)
        if (!silent) ElMessage.success('已打开下载页面')
        return { ok: true, source: 'github', url }
      }
    } catch {
      // 交给下面的兜底
    }
    try {
      const releasesUrl = getClientReleasesUrl(client.githubKey)
      if (releasesUrl) {
        safeOpen(releasesUrl)
        if (!silent) ElMessage.warning('已打开发布页面，请手动选择下载')
        return { ok: true, source: 'github-releases', url: releasesUrl }
      }
    } catch {
      // 继续兜底
    }
  }

  if (client.fallbackUrl) {
    safeOpen(client.fallbackUrl)
    return { ok: true, source: 'fallback', url: client.fallbackUrl }
  }

  if (!silent) ElMessage.error(`${client.name} 的下载地址尚未配置，请联系客服`)
  return { ok: false, reason: 'not-configured' }
}
