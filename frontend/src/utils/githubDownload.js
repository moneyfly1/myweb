/**
 * GitHub 下载工具
 * 自动识别用户系统并获取对应的下载链接
 */

// GitHub 加速前缀（可配置多个，按优先级使用）
const GITHUB_PROXY_PREFIXES = [
  'https://ghproxy.com/https://github.com',
  'https://ghproxy.net/https://github.com',
  'https://github.com' // 备用，直接访问
]

// 客户端软件配置
const CLIENT_CONFIGS = {
  'clash-party': {
    name: 'Clash Party',
    repo: 'mihomo-party-org/clash-party',
    platforms: {
      windows: {
        x64: { pattern: /windows.*64|win64|\.exe$/i, installer: true },
        x32: { pattern: /windows.*32|win32|x32.*\.exe$/i, installer: true },
        arm64: { pattern: /windows.*arm64|win.*arm64|arm64.*\.exe$/i, installer: true }
      },
      macos: {
        intel: { pattern: /(intel|x64|amd64).*\.(pkg|dmg)$/i, installer: true },
        apple: { pattern: /(apple|silicon|m\d+|arm64|aarch64).*\.(pkg|dmg)$/i, installer: true }
      },
      linux: {
        x64: { pattern: /linux.*x64|amd64.*\.(deb|rpm|AppImage)$/i, installer: true },
        arm64: { pattern: /linux.*arm64|aarch64.*\.(deb|rpm|AppImage)$/i, installer: true }
      }
    }
  },
  'clash-verge-rev': {
    name: 'Clash Verge Rev',
    repo: 'clash-verge-rev/clash-verge-rev',
    platforms: {
      windows: {
        x64: { pattern: /(windows|win).*x64|.*x64.*setup|.*x64.*\.exe$/i, installer: true },
        arm64: { pattern: /(windows|win).*arm64|arm64.*\.exe$/i, installer: true }
      },
      macos: {
        intel: { pattern: /(intel|x64|amd64|_x64).*\.dmg$/i, installer: true },
        apple: { pattern: /(apple|silicon|m\d+|arm64|aarch64|_aarch64).*\.dmg$/i, installer: true }
      },
      linux: {
        x64: { pattern: /linux.*x64|amd64.*\.(deb|rpm|AppImage)$/i, installer: true },
        arm64: { pattern: /linux.*arm64|aarch64.*\.(deb|rpm|AppImage)$/i, installer: true }
      }
    }
  },
  'sparkle': {
    name: 'Sparkle',
    repo: 'xishang0128/sparkle',
    platforms: {
      windows: {
        x64: { pattern: /(windows|win).*x64|.*x64.*\.exe$/i, installer: true },
        arm64: { pattern: /(windows|win).*arm64|arm64.*\.exe$/i, installer: true }
      },
      macos: {
        intel: { pattern: /(intel|x64|amd64).*\.dmg$/i, installer: true },
        apple: { pattern: /(apple|silicon|m\d+|arm64|aarch64).*\.dmg$/i, installer: true }
      },
      linux: {
        x64: { pattern: /linux.*x64|amd64.*\.(deb|rpm|AppImage)$/i, installer: true },
        arm64: { pattern: /linux.*arm64|aarch64.*\.(deb|rpm|AppImage)$/i, installer: true }
      }
    }
  },
  'hiddify-app': {
    name: 'Hiddify',
    repo: 'hiddify/hiddify-app',
    platforms: {
      windows: {
        x64: { pattern: /(windows|win).*x64|.*x64.*\.exe$/i, installer: true },
        arm64: { pattern: /(windows|win).*arm64|arm64.*\.exe$/i, installer: true }
      },
      android: {
        universal: { pattern: /android.*\.apk|\.apk$/i, installer: true }
      },
      macos: {
        intel: { pattern: /(intel|x64|amd64).*\.dmg$/i, installer: true },
        apple: { pattern: /(apple|silicon|m\d+|arm64|aarch64).*\.dmg$/i, installer: true }
      },
      linux: {
        x64: { pattern: /linux.*x64|amd64.*\.(deb|rpm|AppImage)$/i, installer: true },
        arm64: { pattern: /linux.*arm64|aarch64.*\.(deb|rpm|AppImage)$/i, installer: true }
      }
    }
  },
  'FlClash': {
    name: 'FlClash',
    repo: 'chen08209/FlClash',
    platforms: {
      windows: {
        x64: { pattern: /(windows|win).*x64|.*x64.*\.exe$/i, installer: true },
        arm64: { pattern: /(windows|win).*arm64|arm64.*\.exe$/i, installer: true }
      },
      macos: {
        intel: { pattern: /(intel|x64|amd64).*\.dmg$/i, installer: true },
        apple: { pattern: /(apple|silicon|m\d+|arm64|aarch64).*\.dmg$/i, installer: true }
      },
      android: {
        universal: { pattern: /android.*arm64.*v8a|arm64.*v8a.*\.apk|android.*\.apk$/i, installer: true }
      },
      linux: {
        x64: { pattern: /linux.*x64|amd64.*\.(deb|rpm|AppImage)$/i, installer: true },
        arm64: { pattern: /linux.*arm64|aarch64.*\.(deb|rpm|AppImage)$/i, installer: true }
      }
    }
  },
  'v2rayNG': {
    name: 'V2rayNG',
    repo: '2dust/v2rayNG',
    platforms: {
      android: {
        universal: { pattern: /android.*\.apk|\.apk$/i, installer: true }
      }
    }
  },
  'v2rayN': {
    name: 'V2rayN',
    repo: '2dust/v2rayN',
    platforms: {
      windows: {
        x64: { pattern: /windows.*64|win64|.*64.*\.zip$/i, installer: false },
        x32: { pattern: /windows.*32|win32|x32.*\.zip$/i, installer: false }
      },
      macos: {
        apple: { pattern: /macos.*arm64|mac.*arm64|arm64.*\.dmg$/i, installer: true },
        intel: { pattern: /macos.*intel|mac.*intel|intel.*\.dmg$/i, installer: true }
      }
    }
  }
}

/**
 * 检测用户操作系统和架构
 */
export function detectSystem() {
  const userAgent = navigator.userAgent.toLowerCase()
  const platform = navigator.platform.toLowerCase()
  
  let os = 'unknown'
  let arch = 'unknown'
  
  // 检测操作系统
  if (userAgent.includes('win') || platform.includes('win')) {
    os = 'windows'
  } else if (userAgent.includes('mac') || platform.includes('mac')) {
    os = 'macos'
  } else if (userAgent.includes('linux') || platform.includes('linux')) {
    os = 'linux'
  } else if (userAgent.includes('android')) {
    os = 'android'
  } else if (userAgent.includes('iphone') || userAgent.includes('ipad')) {
    os = 'ios'
  }
  
  // 检测架构
  if (os === 'windows') {
    // Windows 架构检测
    if (navigator.userAgent.includes('ARM64') || navigator.userAgent.includes('arm64')) {
      arch = 'arm64'
    } else if (navigator.userAgent.includes('WOW64') || navigator.userAgent.includes('x64')) {
      arch = 'x64'
    } else {
      arch = 'x32'
    }
  } else if (os === 'macos') {
    // macOS 架构检测
    // 注意：Apple Silicon Mac 也可能显示为 MacIntel，需要通过其他方式检测
    // 检查 CPU 核心数（Apple Silicon 通常是 8 核或更多，Intel 通常是 4 核）
    // 或者检查硬件并发数
    const hardwareConcurrency = navigator.hardwareConcurrency || 0
    
    // 如果明确包含 Intel，则认为是 Intel
    if (navigator.userAgent.includes('Intel') && !navigator.userAgent.includes('Apple')) {
      arch = 'intel'
    } else if (navigator.userAgent.includes('Apple') || navigator.userAgent.includes('Silicon') || navigator.userAgent.includes('ARM')) {
      arch = 'apple'
    } else {
      // 无法确定时，优先使用 Apple Silicon（因为新 Mac 大多是 Apple Silicon）
      // 但也可以通过硬件并发数判断（Apple Silicon 通常 >= 8）
      if (hardwareConcurrency >= 8) {
        arch = 'apple'
      } else {
        arch = 'intel'
      }
    }
  } else if (os === 'linux') {
    // Linux 架构检测
    if (navigator.userAgent.includes('arm64') || navigator.userAgent.includes('aarch64')) {
      arch = 'arm64'
    } else {
      arch = 'x64'
    }
  } else if (os === 'android') {
    arch = 'universal'
  }
  
  return { os, arch }
}

/**
 * 添加 GitHub 加速前缀
 */
export function addGitHubProxy(url) {
  if (!url || !url.includes('github.com')) {
    return url
  }
  
  // 如果已经是加速链接，直接返回
  if (url.includes('ghproxy.com') || url.includes('ghproxy.net')) {
    return url
  }
  
  // 优先使用 ghproxy.com，如果失败会自动降级
  // 对于下载链接，使用加速前缀
  const proxyPrefix = GITHUB_PROXY_PREFIXES[0] // https://ghproxy.com/https://github.com
  return url.replace('https://github.com', proxyPrefix)
}

/**
 * 获取 GitHub Release 最新版本的下载链接
 * @param {string} repo - 仓库路径，格式：owner/repo
 * @param {string} os - 操作系统：windows, macos, linux, android, ios
 * @param {string} arch - 架构：x64, x32, arm64, intel, apple, universal
 * @returns {Promise<string>} 下载链接
 */
export async function getGitHubDownloadUrl(repo, os, arch, configKey = null) {
  try {
    // 如果提供了 configKey，直接使用
    let config = configKey ? CLIENT_CONFIGS[configKey] : null
    // 如果没有找到，尝试通过 repo 查找
    if (!config) {
      config = Object.values(CLIENT_CONFIGS).find(c => c.repo === repo)
    }
    if (!config) {
      throw new Error(`未找到仓库配置: ${repo}`)
    }
    
    // 构建 API URL（使用加速前缀）
    // 注意：API 请求也使用加速，但某些代理可能不支持 API，如果失败会自动降级
    const apiUrl = addGitHubProxy(`https://api.github.com/repos/${repo}/releases/latest`)
    console.log('请求 GitHub API:', apiUrl)
    
    // 获取最新版本信息（添加超时和错误处理）
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 10000) // 10秒超时
    
    let response
    try {
      response = await fetch(apiUrl, { 
        signal: controller.signal,
        headers: {
          'Accept': 'application/vnd.github.v3+json'
        }
      })
      clearTimeout(timeoutId)
    } catch (fetchError) {
      clearTimeout(timeoutId)
      if (fetchError.name === 'AbortError') {
        throw new Error('请求超时，请稍后重试')
      }
      throw fetchError
    }
    
    if (!response.ok) {
      throw new Error(`获取发布信息失败: ${response.status}`)
    }
    
    const data = await response.json()
    
    // 查找匹配的下载链接
    const platformConfig = config.platforms[os]
    if (!platformConfig) {
      throw new Error(`不支持的操作系统: ${os}`)
    }
    
    const archConfig = platformConfig[arch]
    if (!archConfig) {
      // 如果找不到精确匹配，尝试使用第一个可用的架构
      const firstArch = Object.keys(platformConfig)[0]
      if (firstArch) {
        const fallbackConfig = platformConfig[firstArch]
        const asset = data.assets.find(asset => fallbackConfig.pattern.test(asset.name))
        if (asset) {
          return addGitHubProxy(asset.browser_download_url)
        }
      }
      throw new Error(`不支持的架构: ${arch}`)
    }
    
    // 查找匹配的 asset（按优先级排序）
    let asset = data.assets.find(asset => {
      return archConfig.pattern.test(asset.name)
    })
    
    if (!asset) {
      // 如果找不到精确匹配，尝试查找任何匹配平台的 asset
      const fallbackAsset = data.assets.find(asset => {
        // 简单的文件名匹配
        const name = asset.name.toLowerCase()
        if (os === 'windows' && name.includes('.exe')) return true
        if (os === 'windows' && name.includes('.zip')) return true
        if (os === 'macos' && (name.includes('.dmg') || name.includes('.pkg'))) return true
        if (os === 'linux' && (name.includes('.deb') || name.includes('.rpm') || name.includes('.appimage'))) return true
        if (os === 'android' && name.includes('.apk')) return true
        return false
      })
      if (fallbackAsset) {
        asset = fallbackAsset
      } else {
        throw new Error(`未找到匹配的下载文件`)
      }
    }
    
    // 返回加速后的下载链接
    const downloadUrl = addGitHubProxy(asset.browser_download_url)
    console.log('匹配到下载文件:', asset.name, '下载链接:', downloadUrl)
    return downloadUrl
  } catch (error) {
    console.error('获取 GitHub 下载链接失败:', error)
    // 返回仓库 releases 页面作为备用
    return addGitHubProxy(`https://github.com/${repo}/releases/latest`)
  }
}

/**
 * 获取客户端下载链接（自动识别系统）
 * @param {string} clientKey - 客户端标识
 * @returns {Promise<string>} 下载链接
 */
export async function getClientDownloadUrl(clientKey) {
  const { os, arch } = detectSystem()
  
  // 客户端映射
  const clientMap = {
    'clash-party': { repo: 'mihomo-party-org/clash-party', name: 'Clash Party', configKey: 'clash-party' },
    'clash-verge': { repo: 'clash-verge-rev/clash-verge-rev', name: 'Clash Verge Rev', configKey: 'clash-verge-rev' },
    'sparkle': { repo: 'xishang0128/sparkle', name: 'Sparkle', configKey: 'sparkle' },
    'hiddify': { repo: 'hiddify/hiddify-app', name: 'Hiddify', configKey: 'hiddify-app' },
    'flclash': { repo: 'chen08209/FlClash', name: 'FlClash', configKey: 'FlClash' },
    'v2rayng': { repo: '2dust/v2rayNG', name: 'V2rayNG', configKey: 'v2rayNG' },
    'v2rayn': { repo: '2dust/v2rayN', name: 'V2rayN', configKey: 'v2rayN' }
  }
  
  const client = clientMap[clientKey]
  if (!client) {
    throw new Error(`未知的客户端: ${clientKey}`)
  }
  
  // Android 使用通用链接（优先 arm64-v8a，然后是其他 APK）
  if (os === 'android') {
    try {
      const apiUrl = addGitHubProxy(`https://api.github.com/repos/${client.repo}/releases/latest`)
      const response = await fetch(apiUrl)
      if (response.ok) {
        const data = await response.json()
        // 优先查找 arm64-v8a 版本
        let apkAsset = data.assets.find(asset => 
          asset.name.includes('arm64-v8a') && asset.name.endsWith('.apk')
        )
        // 如果没有找到，查找任何 APK
        if (!apkAsset) {
          apkAsset = data.assets.find(asset => asset.name.endsWith('.apk'))
        }
        if (apkAsset) {
          return addGitHubProxy(apkAsset.browser_download_url)
        }
      }
    } catch (error) {
      console.error('获取 Android 下载链接失败:', error)
    }
    // 备用：返回 releases 页面
    return addGitHubProxy(`https://github.com/${client.repo}/releases/latest`)
  }
  
  // 其他平台使用自动识别，传递 configKey 以便精确匹配
  return await getGitHubDownloadUrl(client.repo, os, arch, client.configKey)
}

/**
 * 获取客户端 GitHub Releases 页面链接（带加速）
 */
export function getClientReleasesUrl(clientKey) {
  const clientMap = {
    'clash-party': 'mihomo-party-org/clash-party',
    'clash-verge': 'clash-verge-rev/clash-verge-rev',
    'sparkle': 'xishang0128/sparkle',
    'hiddify': 'hiddify/hiddify-app',
    'flclash': 'chen08209/FlClash',
    'v2rayng': '2dust/v2rayNG',
    'v2rayn': '2dust/v2rayN'
  }
  
  const repo = clientMap[clientKey]
  if (!repo) {
    return null
  }
  
  return addGitHubProxy(`https://github.com/${repo}/releases/latest`)
}

