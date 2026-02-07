import { ElMessage } from 'element-plus'
import { onMounted, onUnmounted, nextTick } from 'vue'

const listenerMap = new WeakMap()
let observer = null
let isGlobalInitialized = false
let debounceTimer = null

export const isMobile = () => {
  return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
}

const debounce = (fn, delay = 200) => {
  return (...args) => {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => {
      fn(...args)
    }, delay)
  }
}

const extractText = (element) => {
  if (!element) return ''
  if (element.dataset.copyContent) return element.dataset.copyContent
  return element.innerText?.trim() || element.textContent?.trim() || ''
}

const extractUrl = (element) => {
  if (!element) return null
  
  const link = element.tagName === 'A' ? element : element.querySelector('a[href]')
  if (link) return link.getAttribute('href')

  const elLink = element.classList.contains('el-link') ? element : element.querySelector('.el-link')
  if (elLink) {
    return elLink.getAttribute('href') || 
           elLink.getAttribute('data-url') ||
           (elLink.textContent?.trim().match(/^https?:\/\//) ? elLink.textContent.trim() : null)
  }
  
  return element.getAttribute('data-url')
}

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
    
    throw new Error('Clipboard API unavailable, falling back')
  } catch (error) {
    try {
      const textArea = document.createElement('textarea')
      textArea.value = text
      
      textArea.style.cssText = `
        position: fixed;
        top: -9999px;
        left: -9999px;
        opacity: 0;
        z-index: -1;
        pointer-events: none;
      `
      document.body.appendChild(textArea)
      
      if (isMobile()) {
        textArea.contentEditable = 'true'
        textArea.readOnly = false
      }
      
      textArea.focus()
      textArea.select()
      
      if (/iPhone|iPad|iPod/i.test(navigator.userAgent)) {
        const range = document.createRange()
        range.selectNodeContents(textArea)
        const selection = window.getSelection()
        selection.removeAllRanges()
        selection.addRange(range)
        textArea.setSelectionRange(0, text.length)
      }
      
      const successful = document.execCommand('copy')
      document.body.removeChild(textArea)
      
      if (successful) {
        ElMessage.success(successMessage)
        return true
      }
    } catch (err) {
      console.error('Copy fallback failed:', err)
    }
    
    return false
  }
}

export const setupLongPressCopy = (element, getTextFn) => {
  if (!element) return () => {}
  
  element.style.webkitUserSelect = 'text'
  element.style.userSelect = 'text'
  element.style.webkitTouchCallout = 'default'
  
  return () => {}
}

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
    if (e.ctrlKey) return
    if (isMobile()) return

    e.preventDefault()
    
    removeExistingMenu()
    
    const text = getTextFn()
    const currentUrl = getUrl ? getUrl() : url
    const menuItems = []
    
    if (text) {
      menuItems.push({
        label: '复制内容',
        icon: '📋',
        action: () => {
          copyToClipboard(text)
          if (onCopy) onCopy(text)
        }
      })
      
      if (text.length < 50) {
        menuItems.push({
          label: 'Google 搜索',
          icon: '🔍',
          action: () => {
            window.open(`https://www.google.com/search?q=${encodeURIComponent(text)}`, '_blank')
          }
        })
      }
    }
    
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
    
    if (additionalItems.length > 0) {
      if (menuItems.length > 0) menuItems.push({ type: 'divider' })
      menuItems.push(...additionalItems)
    }
    
    if (menuItems.length === 0) return
    
    createAndShowMenu(e, menuItems)
  }
  
  element.addEventListener('contextmenu', handleContextMenu)
  return () => element.removeEventListener('contextmenu', handleContextMenu)
}

const removeExistingMenu = () => {
  const existingMenu = document.getElementById('global-custom-context-menu')
  if (existingMenu) {
    document.body.removeChild(existingMenu)
  }
}

const createAndShowMenu = (e, items) => {
  const menu = document.createElement('div')
  menu.id = 'global-custom-context-menu'
  
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

  requestAnimationFrame(() => {
    const { clientWidth: menuWidth, clientHeight: menuHeight } = menu
    const { innerWidth, innerHeight } = window
    
    let left = e.clientX
    let top = e.clientY

    if (left + menuWidth > innerWidth) {
      left = innerWidth - menuWidth - 10
    }
    if (top + menuHeight > innerHeight) {
      top = innerHeight - menuHeight - 10
    }

    menu.style.left = `${left}px`
    menu.style.top = `${top}px`
    menu.style.opacity = '1'
    menu.style.transform = 'scale(1)'
  })

  const closeHandler = (ev) => {
    if (!menu.contains(ev.target)) {
      removeExistingMenu()
      document.removeEventListener('click', closeHandler)
      document.removeEventListener('contextmenu', closeHandler)
      window.removeEventListener('scroll', closeHandler)
    }
  }
  
  setTimeout(() => {
    document.addEventListener('click', closeHandler)
    document.addEventListener('contextmenu', closeHandler)
    window.addEventListener('scroll', closeHandler, { passive: true })
  }, 10)
}

export const setupTextSelection = (element, getTextFn, options = {}) => {
  if (!element || !getTextFn) return () => {}

  const isInputElement = element.tagName === 'INPUT' || 
                        element.tagName === 'TEXTAREA' ||
                        element.classList.contains('el-input__inner') ||
                        element.classList.contains('el-textarea__inner') ||
                        element.closest('input, textarea, .el-input__inner, .el-textarea__inner')
  
  if (isInputElement) {
    enableInputPaste(element)
    return () => {}
  }

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

  element.dataset.textSelectionEnabled = 'true'
  listenerMap.set(element, cleanupAll)

  return cleanupAll
}

const enableInputPaste = (input) => {
  if (input.dataset.pasteEnabled === 'true') return
  
  input.dataset.pasteEnabled = 'true'
  
  input.style.userSelect = 'text'
  input.style.webkitUserSelect = 'text'
  input.style.webkitTouchCallout = 'default'
  
  if (isMobile() && !input.getAttribute('inputmode')) {
    const tagName = input.tagName.toLowerCase()
    if (tagName === 'input') {
      const type = input.getAttribute('type')
      if (!type || type === 'text' || type === 'search' || type === 'email') {
        input.setAttribute('inputmode', 'text')
      }
    } else if (tagName === 'textarea') {
      input.setAttribute('inputmode', 'text')
    }
  }
  
  const elInput = input.closest('.el-input, .el-textarea')
  if (elInput) {
    const innerInput = elInput.querySelector('input, textarea')
    if (innerInput && innerInput !== input) {
      innerInput.style.userSelect = 'text'
      innerInput.style.webkitUserSelect = 'text'
      innerInput.style.webkitTouchCallout = 'default'
    }
  }
}

const scanAndActivate = () => {
  const inputs = document.querySelectorAll('input, textarea, .el-input__inner, .el-textarea__inner')
  inputs.forEach(enableInputPaste)

  const tableCells = document.querySelectorAll('.el-table td .cell')
  tableCells.forEach(cell => {
    if (cell.closest('td')?.dataset?.textSelectionEnabled) return
    if (cell.querySelector('input, textarea, .el-input__inner, .el-textarea__inner')) return
    if (cell.querySelector('button, .el-button') && !cell.querySelector('a, .el-link')) return

    const url = extractUrl(cell)
    const target = cell.closest('td')
    if (!target) return

    setupTextSelection(target, () => extractText(cell), {
      url,
      getUrl: () => extractUrl(cell)
    })
  })

  const selectors = [
    '.mobile-card .value', 
    '.mobile-card .label',
    '.allow-copy',
    '[data-allow-copy]'
  ]
  
  const commonElements = document.querySelectorAll(selectors.join(','))
  commonElements.forEach(el => {
    if (el.closest('button, .el-button, .el-tag')) return
    
    if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || 
        el.classList.contains('el-input__inner') || 
        el.classList.contains('el-textarea__inner') ||
        el.closest('input, textarea, .el-input__inner, .el-textarea__inner')) return
    
    setupTextSelection(el, () => extractText(el), {
      url: extractUrl(el),
      getUrl: () => extractUrl(el)
    })
  })
}

export function initTextSelection() {
  if (isGlobalInitialized) return
  isGlobalInitialized = true

  scanAndActivate()

  observer = new MutationObserver(debounce(() => {
    scanAndActivate()
  }, 300))

  observer.observe(document.body, {
    childList: true,
    subtree: true
  })

  window.addEventListener('popstate', scanAndActivate)
}

export function cleanupTextSelection() {
  if (observer) {
    observer.disconnect()
    observer = null
  }
  removeExistingMenu()
  window.removeEventListener('popstate', scanAndActivate)
  isGlobalInitialized = false
}

export function useTextSelection() {
  const localCleanups = []

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
        const cleanup = setupTextSelection(el, textFn)
        localCleanups.push(cleanup)
      })
    })
  }

  onMounted(() => {
    if (!isGlobalInitialized) {
      scanAndActivate()
    }
  })

  onUnmounted(() => {
    localCleanups.forEach(fn => fn())
    removeExistingMenu()
  })

  return {
    enableSelectionFor,
    copyToClipboard
  }
}
