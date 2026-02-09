import { ElMessage } from 'element-plus'
const isMobileDevice = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
export const isMobile = () => isMobileDevice
export const copyToClipboard = async (text, successMessage = '已复制到剪贴板') => {
  if (!text) {
    ElMessage.warning('没有可复制的内容')
    return false
  }
  try {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      await navigator.clipboard.writeText(text)
      ElMessage.success(successMessage)
      return true
    }
    throw new Error('fallback')
  } catch (_) {
    try {
      const ta = document.createElement('textarea')
      ta.value = text
      ta.style.cssText = 'position:fixed;top:-9999px;left:-9999px;opacity:0'
      document.body.appendChild(ta)
      if (isMobileDevice) {
        ta.contentEditable = 'true'
        ta.readOnly = false
      }
      ta.focus()
      ta.select()
      if (/iPhone|iPad|iPod/i.test(navigator.userAgent)) {
        ta.setSelectionRange(0, text.length)
      }
      const ok = document.execCommand('copy')
      document.body.removeChild(ta)
      if (ok) {
        ElMessage.success(successMessage)
        return true
      }
    } catch (e) {
      console.error('Copy failed:', e)
    }
    ElMessage.error('复制失败，请手动复制')
    return false
  }
}
let initialized = false
export function initTextSelection() {
  if (initialized) return
  initialized = true
  if (isMobileDevice) return
  let timer = null
  const scan = () => {
    document.querySelectorAll('.el-table td .cell').forEach(cell => {
      const td = cell.closest('td')
      if (!td || td.dataset.ctx === '1') return
      if (cell.querySelector('input,textarea,button,.el-button')) return
      td.dataset.ctx = '1'
      td.addEventListener('contextmenu', (e) => {
        if (e.ctrlKey) return
        const text = cell.innerText?.trim()
        if (!text) return
        e.preventDefault()
        showMenu(e, text, cell)
      })
    })
  }
  scan()
  const ob = new MutationObserver(() => {
    if (timer) clearTimeout(timer)
    timer = setTimeout(scan, 500)
  })
  ob.observe(document.body, { childList: true, subtree: true })
  window.addEventListener('popstate', scan)
}
function showMenu(e, text, cell) {
  let old = document.getElementById('_ctx')
  if (old) old.remove()
  const items = []
  items.push({ label: '📋 复制内容', fn: () => copyToClipboard(text) })
  if (text.length < 50) {
    items.push({ label: '🔍 Google 搜索', fn: () => window.open('https://www.google.com/search?q=' + encodeURIComponent(text), '_blank') })
  }
  const link = cell.querySelector('a[href]')
  const href = link ? link.getAttribute('href') : null
  if (href) {
    items.push({ label: '🔗 新标签页打开', fn: () => window.open(href, '_blank') })
  }
  const menu = document.createElement('div')
  menu.id = '_ctx'
  menu.style.cssText = 'position:fixed;background:#fff;border:1px solid #e4e7ed;box-shadow:0 2px 12px rgba(0,0,0,.1);border-radius:4px;z-index:99999;padding:5px 0;min-width:140px;font-size:13px;user-select:none'
  items.forEach(it => {
    const d = document.createElement('div')
    d.textContent = it.label
    d.style.cssText = 'padding:8px 16px;cursor:pointer;color:#606266'
    d.onmouseenter = () => d.style.background = '#ecf5ff'
    d.onmouseleave = () => d.style.background = ''
    d.onclick = (ev) => { ev.stopPropagation(); it.fn(); menu.remove() }
    menu.appendChild(d)
  })
  document.body.appendChild(menu)
  requestAnimationFrame(() => {
    let x = e.clientX, y = e.clientY
    if (x + menu.offsetWidth > window.innerWidth) x = window.innerWidth - menu.offsetWidth - 5
    if (y + menu.offsetHeight > window.innerHeight) y = window.innerHeight - menu.offsetHeight - 5
    menu.style.left = x + 'px'
    menu.style.top = y + 'px'
  })
  const close = (ev) => {
    if (!menu.contains(ev.target)) {
      menu.remove()
      document.removeEventListener('click', close)
      document.removeEventListener('contextmenu', close)
    }
  }
  setTimeout(() => {
    document.addEventListener('click', close)
    document.addEventListener('contextmenu', close)
  }, 10)
}
export function cleanupTextSelection() {
  initialized = false
}
export function useTextSelection() {
  return { copyToClipboard }
}
