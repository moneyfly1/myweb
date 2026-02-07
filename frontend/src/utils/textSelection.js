/**
 * 文本选择、复制与增强上下文菜单工具
 * * 功能亮点：
 * 1. 统一移动端（长按）和桌面端（右键）交互
 * 2. 智能识别 Element Plus 表格、链接和卡片
 * 3. 增强右键菜单：支持复制、打开链接、Google搜索
 * 4. 边界检测：防止菜单溢出屏幕
 * 5. 性能优化：Observer 防抖与 WeakMap 内存管理
 */

import { ElMessage } from 'element-plus'
import { onMounted, onUnmounted, nextTick } from 'vue'

// ==========================================
// 常量与状态管理
// ==========================================

// 用于存储元素与清理函数的映射，使用 WeakMap 自动处理垃圾回收
const listenerMap = new WeakMap()

// 全局 MutationObserver 实例
let observer = null
let isGlobalInitialized = false

// 防抖计时器
let debounceTimer = null

// ==========================================
// 基础工具函数
// ==========================================

/**
 * 检测是否为移动设备
 */
export const isMobile = () => {
  return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
}

/**
 * 防抖函数
 */
const debounce = (fn, delay = 200) => {
  return (...args) => {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => {
      fn(...args)
    }, delay)
  }
}

/**
 * 智能提取元素的文本内容
 */
const extractText = (element) => {
  if (!element) return ''
  // 优先获取 data-copy-content (允许手动覆盖)
  if (element.dataset.copyContent) return element.dataset.copyContent
  return element.innerText?.trim() || element.textContent?.trim() || ''
}

/**
 * 智能提取元素的 URL
 * 支持 a 标签、el-link 组件及 data-url 属性
 */
const extractUrl = (element) => {
  if (!element) return null
  
  // 1. 直接检查自身或子元素的 a 标签
  const link = element.tagName === 'A' ? element : element.querySelector('a[href]')
  if (link) return link.getAttribute('href')

  // 2. 检查 el-link 或 data-url
  const elLink = element.classList.contains('el-link') ? element : element.querySelector('.el-link')
  if (elLink) {
    return elLink.getAttribute('href') || 
           elLink.getAttribute('data-url') ||
           // 简单的正则判断文本是否为链接
           (elLink.textContent?.trim().match(/^https?:\/\//) ? elLink.textContent.trim() : null)
  }
  
  // 3. 检查自身 data-url
  return element.getAttribute('data-url')
}

// ==========================================
// 核心业务逻辑
// ==========================================

/**
 * 复制文本到剪贴板 (增强版)
 */
export const copyToClipboard = async (text, successMessage = '已复制到剪贴板') => {
  if (!text) {
    ElMessage.warning('没有可复制的内容')
    return false
  }
  
  try {
    // 方案 A: 现代异步 API (仅在 HTTPS 或 localhost 下可用)
    if (navigator.clipboard && navigator.clipboard.writeText) {
      await navigator.clipboard.writeText(text)
      ElMessage.success(successMessage)
      return true
    } 
    
    // 方案 B: document.execCommand (兼容性兜底)
    throw new Error('Clipboard API unavailable, falling back')
  } catch (error) {
    // 降级处理
    const textArea = document.createElement('textarea')
    textArea.value = text
    
    // 确保元素不可见但仍属于 DOM flow 以便选中
    textArea.style.cssText = `
      position: fixed;
      top: -9999px;
      left: -9999px;
      opacity: 0;
      z-index: -1;
    `
    document.body.appendChild(textArea)
    textArea.focus()
    textArea.select()
    
    try {
      const successful = document.execCommand('copy')
      document.body.removeChild(textArea)
      if (successful) {
        ElMessage.success(successMessage)
        return true
      }
    } catch (err) {
      document.body.removeChild(textArea)
    }
    
    console.error('Copy failed:', error)
    ElMessage.error('复制失败，请手动选择复制')
    return false
  }
}

/**
 * 移动端长按逻辑
 */
export const setupLongPressCopy = (element, getTextFn) => {
  if (!element) return () => {}
  
  let longPressTimer = null
  let isLongPress = false
  let startX = 0
  let startY = 0
  
  const handleTouchStart = (e) => {
    if (e.touches.length > 1) return // 忽略多指触控
    
    isLongPress = false
    startX = e.touches[0].clientX
    startY = e.touches[0].clientY
    
    longPressTimer = setTimeout(() => {
      isLongPress = true
      const text = getTextFn()
      
      if (text) {
        // 触觉反馈 (如果有 API)
        if (navigator.vibrate) navigator.vibrate(50)
        
        copyToClipboard(text)
        
        // 视觉反馈
        const originalTransition = element.style.transition
        const originalBg = element.style.backgroundColor
        
        element.style.transition = 'background-color 0.2s'
        element.style.backgroundColor = 'var(--el-color-primary-light-9, #e6f7ff)'
        
        setTimeout(() => {
          element.style.backgroundColor = originalBg
          element.style.transition = originalTransition
        }, 300)
      }
    }, 600) // 600ms 阈值
  }
  
  const handleTouchEnd = (e) => {
    if (longPressTimer) {
      clearTimeout(longPressTimer)
      longPressTimer = null
    }
    // 如果触发了长按，阻止后续的 click 事件（防止误触链接）
    if (isLongPress && e.cancelable) {
      e.preventDefault()
    }
  }
  
  const handleTouchMove = (e) => {
    // 允许微小的移动抖动 (10px)
    const moveX = e.touches[0].clientX
    const moveY = e.touches[0].clientY
    if (Math.abs(moveX - startX) > 10 || Math.abs(moveY - startY) > 10) {
      if (longPressTimer) {
        clearTimeout(longPressTimer)
        longPressTimer = null
      }
    }
  }
  
  element.addEventListener('touchstart', handleTouchStart, { passive: false }) // passive: false 为了能 preventDefault
  element.addEventListener('touchend', handleTouchEnd)
  element.addEventListener('touchmove', handleTouchMove, { passive: true })
  
  return () => {
    element.removeEventListener('touchstart', handleTouchStart)
    element.removeEventListener('touchend', handleTouchEnd)
    element.removeEventListener('touchmove', handleTouchMove)
  }
}

/**
 * 桌面端右键菜单逻辑
 */
export const setupContextMenu = (element, getTextFn, options = {}) => {
  if (!element) return () => {}
  
  const {
    onCopy,
    onOpenInNewTab,
    url,
    getUrl,
    additionalItems = []
  } = options
  
  const handleContextMenu = (e) => {
    // 如果按下 Ctrl 键，允许原生右键菜单（方便开发者调试）
    if (e.ctrlKey) return

    e.preventDefault()
    
    // 移除旧菜单
    removeExistingMenu()
    
    const text = getTextFn()
    const currentUrl = getUrl ? getUrl() : url
    const menuItems = []
    
    // 1. 复制选项
    if (text) {
      menuItems.push({
        label: '复制内容',
        icon: '📋',
        action: () => {
          copyToClipboard(text)
          if (onCopy) onCopy(text)
        }
      })
      
      // 2. Google 搜索 (新增功能)
      if (text.length < 50) { // 只有短文本才显示搜索，避免长文本刷屏
        menuItems.push({
          label: 'Google 搜索',
          icon: '🔍',
          action: () => {
            window.open(`https://www.google.com/search?q=${encodeURIComponent(text)}`, '_blank')
          }
        })
      }
    }
    
    // 3. 打开链接选项
    if (currentUrl) {
      menuItems.push({
        label: '新标签页打开',
        icon: '🔗',
        action: () => {
          window.open(currentUrl, '_blank')
          if (onOpenInNewTab) onOpenInNewTab(currentUrl)
        }
      })
    }
    
    // 4. 自定义选项
    if (additionalItems.length > 0) {
      // 添加分隔线
      if (menuItems.length > 0) menuItems.push({ type: 'divider' })
      menuItems.push(...additionalItems)
    }
    
    if (menuItems.length === 0) return
    
    createAndShowMenu(e, menuItems)
  }
  
  element.addEventListener('contextmenu', handleContextMenu)
  return () => element.removeEventListener('contextmenu', handleContextMenu)
}

/**
 * 辅助：移除已存在的菜单
 */
const removeExistingMenu = () => {
  const existingMenu = document.getElementById('global-custom-context-menu')
  if (existingMenu) {
    document.body.removeChild(existingMenu)
  }
}

/**
 * 辅助：创建并定位菜单
 */
const createAndShowMenu = (e, items) => {
  const menu = document.createElement('div')
  menu.id = 'global-custom-context-menu'
  
  // 基础样式
  menu.style.cssText = `
    position: fixed;
    background: #fff;
    border: 1px solid #e4e7ed;
    box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
    border-radius: 4px;
    z-index: 99999;
    padding: 5px 0;
    min-width: 140px;
    font-family: inherit;
    font-size: 13px;
    user-select: none;
    opacity: 0;
    transform: scale(0.95);
    transition: opacity 0.1s, transform 0.1s;
  `

  // 构建菜单项
  items.forEach(item => {
    if (item.type === 'divider') {
      const divider = document.createElement('div')
      divider.style.cssText = 'height: 1px; background: #ebeef5; margin: 5px 0;'
      menu.appendChild(divider)
      return
    }

    const div = document.createElement('div')
    div.innerHTML = `<span style="margin-right: 8px">${item.icon || ''}</span>${item.label}`
    div.style.cssText = `
      padding: 8px 16px;
      cursor: pointer;
      color: #606266;
      display: flex;
      align-items: center;
      transition: background 0.1s;
    `
    div.addEventListener('mouseenter', () => div.style.backgroundColor = '#ecf5ff')
    div.addEventListener('mouseleave', () => div.style.backgroundColor = 'transparent')
    div.addEventListener('click', (ev) => {
      ev.stopPropagation()
      item.action()
      removeExistingMenu()
    })
    menu.appendChild(div)
  })

  document.body.appendChild(menu)

  // 智能定位 (防止溢出屏幕)
  requestAnimationFrame(() => {
    const { clientWidth: menuWidth, clientHeight: menuHeight } = menu
    const { innerWidth, innerHeight } = window
    
    let left = e.clientX
    let top = e.clientY

    // 右侧溢出检测
    if (left + menuWidth > innerWidth) {
      left = innerWidth - menuWidth - 10
    }
    // 底部溢出检测
    if (top + menuHeight > innerHeight) {
      top = innerHeight - menuHeight - 10
    }

    menu.style.left = `${left}px`
    menu.style.top = `${top}px`
    menu.style.opacity = '1'
    menu.style.transform = 'scale(1)'
  })

  // 点击外部关闭
  const closeHandler = (ev) => {
    if (!menu.contains(ev.target)) {
      removeExistingMenu()
      document.removeEventListener('click', closeHandler)
      document.removeEventListener('contextmenu', closeHandler)
      window.removeEventListener('scroll', closeHandler) // 滚动也关闭
    }
  }
  
  // 延迟绑定避免触发本次点击
  setTimeout(() => {
    document.addEventListener('click', closeHandler)
    document.addEventListener('contextmenu', closeHandler)
    window.addEventListener('scroll', closeHandler, { passive: true })
  }, 10)
}

/**
 * 统一设置函数
 */
export const setupTextSelection = (element, getTextFn, options = {}) => {
  if (!element || !getTextFn) return () => {}

  // 检查是否已经初始化过，防止重复绑定
  if (listenerMap.has(element)) {
    return listenerMap.get(element)
  }

  const cleanups = []

  if (isMobile()) {
    cleanups.push(setupLongPressCopy(element, getTextFn))
  } else {
    cleanups.push(setupContextMenu(element, getTextFn, options))
  }

  const cleanupAll = () => {
    cleanups.forEach(fn => fn())
    listenerMap.delete(element)
    delete element.dataset.textSelectionEnabled
  }

  // 标记并存储
  element.dataset.textSelectionEnabled = 'true'
  listenerMap.set(element, cleanupAll)

  return cleanupAll
}

// ==========================================
// 自动化处理逻辑
// ==========================================

/**
 * 为输入框启用粘贴优化
 */
const enableInputPaste = (input) => {
  if (input.dataset.pasteEnabled === 'true') return
  
  input.dataset.pasteEnabled = 'true'
  // 确保样式允许选择
  input.style.userSelect = 'text'
  input.style.webkitUserSelect = 'text'
  
  // 某些移动端浏览器需要 inputmode 触发更好的键盘支持
  if (isMobile() && !input.getAttribute('inputmode')) {
    input.setAttribute('inputmode', 'text')
  }
}

/**
 * 扫描并激活所有目标元素
 */
const scanAndActivate = () => {
  // 1. 处理输入框
  const inputs = document.querySelectorAll('input, textarea, .el-input__inner')
  inputs.forEach(enableInputPaste)

  // 2. 处理表格单元格 (.el-table)
  // 排除操作列：通常操作列包含 '操作' 字样或只有按钮
  const tableCells = document.querySelectorAll('.el-table td .cell')
  tableCells.forEach(cell => {
    // 性能优化：快速跳过已处理元素
    if (cell.closest('td')?.dataset?.textSelectionEnabled) return

    // 排除含有按钮、Tag但没有链接的纯操作容器
    if (cell.querySelector('button, .el-button') && !cell.querySelector('a, .el-link')) return

    const url = extractUrl(cell)
    
    // 父级 td 才是我们想绑定事件的目标（点击范围更大）
    const target = cell.closest('td')
    if (!target) return

    setupTextSelection(target, () => extractText(cell), {
      url,
      getUrl: () => extractUrl(cell) // 动态获取，防止 DOM 更新后 url 没变但 href 变了
    })
  })

  // 3. 处理移动端卡片和其他通用类
  const selectors = [
    '.mobile-card .value', 
    '.mobile-card .label',
    '.allow-copy', // 允许用户添加自定义类名
    '[data-allow-copy]'
  ]
  
  const commonElements = document.querySelectorAll(selectors.join(','))
  commonElements.forEach(el => {
    // 跳过按钮内部文本
    if (el.closest('button, .el-button, .el-tag')) return
    
    setupTextSelection(el, () => extractText(el), {
      url: extractUrl(el),
      getUrl: () => extractUrl(el)
    })
  })
}

// ==========================================
// 导出 API
// ==========================================

/**
 * 全局初始化 (在 App.vue 或 main.js 调用)
 */
export function initTextSelection() {
  if (isGlobalInitialized) return
  isGlobalInitialized = true

  // 立即执行一次
  scanAndActivate()

  // 监听 DOM 变化 (防抖)
  observer = new MutationObserver(debounce(() => {
    scanAndActivate()
  }, 300))

  observer.observe(document.body, {
    childList: true,
    subtree: true
  })

  // 路由变化监听 (Vue Router 场景)
  window.addEventListener('popstate', scanAndActivate)
}

/**
 * 全局清理
 */
export function cleanupTextSelection() {
  if (observer) {
    observer.disconnect()
    observer = null
  }
  removeExistingMenu()
  window.removeEventListener('popstate', scanAndActivate)
  isGlobalInitialized = false
  // WeakMap 会自动处理 listener 引用，无需手动遍历清理
}

/**
 * Vue Composable (组件级使用)
 */
export function useTextSelection() {
  const localCleanups = []

  // 组件内手动启用
  const enableSelectionFor = (elementOrSelector, getTextFn = null) => {
    nextTick(() => {
      let elements = []
      if (typeof elementOrSelector === 'string') {
        elements = document.querySelectorAll(elementOrSelector)
      } else if (elementOrSelector instanceof HTMLElement) {
        elements = [elementOrSelector]
      }

      elements.forEach(el => {
        const textFn = getTextFn || (() => extractText(el))
        // 传递空 options，因为组件级调用通常不需要复杂的 url 猜测，或者用户可以在 options 里传
        const cleanup = setupTextSelection(el, textFn)
        localCleanups.push(cleanup)
      })
    })
  }

  // 组件挂载时自动尝试扫描 (如果全局未开启)
  onMounted(() => {
    if (!isGlobalInitialized) {
      scanAndActivate()
    }
  })

  onUnmounted(() => {
    localCleanups.forEach(fn => fn())
    // 移除该组件创建的菜单
    removeExistingMenu()
  })

  return {
    enableSelectionFor,
    // 暴露核心函数供特定场景调用
    copyToClipboard
  }
}