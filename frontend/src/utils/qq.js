// QQ 聊天唤起（管理员在后台一键打开与某个用户的 QQ 对话）
//
// 用途：管理员仪表盘「即将到期客户」、用户管理列表、用户详情里点「QQ」按钮，
// 直接唤起本机已登录的 QQ 客户端，打开与该用户的聊天窗口 —— 免去手动搜索 QQ 号。
//
// ⚠️ 限制（协议能力，不是 bug）：
//   1. 只能打开「与指定 QQ 号」的对话框，**无法预填消息内容**（PC QQ 的
//      tencent:// 协议不支持 msg 参数，网页版 wpa.qq.com 已基本失效）；
//   2. 需要本机安装并登录 QQ PC 客户端；浏览器首次会弹「是否允许打开腾讯QQ」；
//   3. 手机端无效（tencent:// 只对桌面客户端有意义），所以按钮只在桌面视图出现。
//
// QQ 号来源：用户表没有 QQ 字段，这里从邮箱反推 —— 只有「纯数字@qq.com」才算，
// 其它邮箱返回空串，调用方据此不渲染按钮（避免点了没反应）。

/// 从邮箱推断 QQ 号：1123880343@qq.com → "1123880343"；非 QQ 邮箱返回 ""
export function qqFromEmail(email) {
  const m = String(email || '').trim().match(/^(\d{5,12})@qq\.com$/i)
  return m ? m[1] : ''
}

/// 唤起 QQ 客户端打开与该 QQ 号的聊天窗口
export function openQQChat(qq) {
  const id = String(qq || '').trim()
  if (!id) return false
  try {
    // tencent:// 兼容 QQ PC 版各历史版本；mqqwpa:// 是移动/新协议，桌面端也常能用
    window.location.href = `tencent://message/?uin=${encodeURIComponent(id)}&Site=&Menu=yes`
    return true
  } catch (e) {
    console.warn('唤起 QQ 失败:', e)
    return false
  }
}

/// 复制 QQ 号（浏览器拦截外部协议时的兜底）
export async function copyQQ(qq) {
  const id = String(qq || '').trim()
  if (!id) return false
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(id)
      return true
    }
    const ta = document.createElement('textarea')
    ta.value = id
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    return ok
  } catch (e) {
    console.warn('复制 QQ 号失败:', e)
    return false
  }
}
