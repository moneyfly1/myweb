const GITHUB_PROXY_PREFIXES = [
  'https://ghproxy.com/https://github.com',
  'https://ghproxy.net/https://github.com',
  'https://github.com' // 备用，直接访问
]
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
  'clash-verge': {
    name: 'Clash Verge',
    repo: 'clash-verge-rev/clash-verge-rev',
    platforms: {
      windows: {
        x64: { pattern: /(windows|win).*x64|.*x64.*\.(exe|msi)$/i, installer: true },
        arm64: { pattern: /(windows|win).*arm64|arm64.*\.(exe|msi)$/i, installer: true }
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
export function detectSystem() {
  const userAgent = navigator.userAgent.toLowerCase()
  const platform = navigator.platform.toLowerCase()
  let os = 'unknown'
  let arch = 'unknown'
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
  if (os === 'windows') {
    if (navigator.userAgent.includes('ARM64') || navigator.userAgent.includes('arm64')) {
      arch = 'arm64'
    } else if (navigator.userAgent.includes('WOW64') || navigator.userAgent.includes('x64')) {
      arch = 'x64'
    } else {
      arch = 'x32'
    }
  } else if (os === 'macos') {
    const hardwareConcurrency = navigator.hardwareConcurrency || 0
    if (navigator.userAgent.includes('Intel') && !navigator.userAgent.includes('Apple')) {
      arch = 'intel'
    } else if (navigator.userAgent.includes('Apple') || navigator.userAgent.includes('Silicon') || navigator.userAgent.includes('ARM')) {
      arch = 'apple'
    } else {
      if (hardwareConcurrency >= 8) {
        arch = 'apple'
      } else {
        arch = 'intel'
      }
    }
  } else if (os === 'linux') {
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
export function addGitHubProxy(url) {
  if (!url || !url.includes('github.com')) {
    return url
  }
  if (url.includes('ghproxy.com') || url.includes('ghproxy.net')) {
    return url
  }
  const proxyPrefix = GITHUB_PROXY_PREFIXES[0] // https://ghproxy.com/https://github.com
  return url.replace('https://github.com', proxyPrefix)
}
export async function getGitHubDownloadUrl(repo, os, arch, configKey = null) {
  try {
    let config = configKey ? CLIENT_CONFIGS[configKey] : null
    if (!config) {
      config = Object.values(CLIENT_CONFIGS).find(c => c.repo === repo)
    }
    if (!config) {
      throw new Error(`未找到仓库配置: ${repo}`)
    }
    const apiUrl = addGitHubProxy(`https://api.github.com/repos/${repo}/releases/latest`)
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
    const platformConfig = config.platforms[os]
    if (!platformConfig) {
      throw new Error(`不支持的操作系统: ${os}`)
    }
    const archConfig = platformConfig[arch]
    if (!archConfig) {
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
    let asset = data.assets.find(asset => {
      return archConfig.pattern.test(asset.name)
    })
    if (!asset) {
      const fallbackAsset = data.assets.find(asset => {
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
    const downloadUrl = addGitHubProxy(asset.browser_download_url)
    return downloadUrl
  } catch (error) {
    console.error('获取 GitHub 下载链接失败:', error)
    return addGitHubProxy(`https://github.com/${repo}/releases/latest`)
  }
}
export async function getClientDownloadUrl(clientKey) {
  const { os, arch } = detectSystem()
  const clientMap = {
    'clash-party': { repo: 'mihomo-party-org/clash-party', name: 'Clash Party', configKey: 'clash-party' },
    'clash-verge': { repo: 'clash-verge-rev/clash-verge-rev', name: 'Clash Verge', configKey: 'clash-verge' },
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
  if (os === 'android') {
    try {
      const apiUrl = addGitHubProxy(`https://api.github.com/repos/${client.repo}/releases/latest`)
      const response = await fetch(apiUrl)
      if (response.ok) {
        const data = await response.json()
        let apkAsset = data.assets.find(asset => 
          asset.name.includes('arm64-v8a') && asset.name.endsWith('.apk')
        )
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
    return addGitHubProxy(`https://github.com/${client.repo}/releases/latest`)
  }
  return await getGitHubDownloadUrl(client.repo, os, arch, client.configKey)
}
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
