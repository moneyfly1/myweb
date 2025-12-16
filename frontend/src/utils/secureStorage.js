const SECURE_STORAGE_KEY = 'cboard_secure_'
const MAX_STORAGE_AGE = 24 * 60 * 60 * 1000
function isSecureContext() {
  return typeof window !== 'undefined' && window.isSecureContext
}
function getStorageKey(key) {
  return `${SECURE_STORAGE_KEY}${key}`
}
function getStorageItem(key) {
  try {
    const storageKey = getStorageKey(key)
    const item = sessionStorage.getItem(storageKey) || localStorage.getItem(storageKey)
    if (!item) return null
    const data = JSON.parse(item)
    if (data.expiry && Date.now() > data.expiry) {
      removeStorageItem(key)
      return null
    }
    return data.value
  } catch (error) {
    if (process.env.NODE_ENV === 'development') {
      console.error('读取存储失败:', error)
    }
    return null
  }
}
function setStorageItem(key, value, useSession = true, maxAge = MAX_STORAGE_AGE) {
  try {
    const storageKey = getStorageKey(key)
    const data = {
      value,
      expiry: Date.now() + maxAge,
      timestamp: Date.now()
    }
    const storage = useSession ? sessionStorage : localStorage
    storage.setItem(storageKey, JSON.stringify(data))
  } catch (error) {
    if (process.env.NODE_ENV === 'development') {
      console.error('写入存储失败:', error)
    }
  }
}
function removeStorageItem(key) {
  try {
    const storageKey = getStorageKey(key)
    sessionStorage.removeItem(storageKey)
    localStorage.removeItem(storageKey)
  } catch (error) {
    if (process.env.NODE_ENV === 'development') {
      console.error('删除存储失败:', error)
    }
  }
}
function clearSecureStorage() {
  try {
    const keysToRemove = []
    for (let i = 0; i < sessionStorage.length; i++) {
      const key = sessionStorage.key(i)
      if (key && key.startsWith(SECURE_STORAGE_KEY)) {
        keysToRemove.push(key)
      }
    }
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i)
      if (key && key.startsWith(SECURE_STORAGE_KEY)) {
        keysToRemove.push(key)
      }
    }
    keysToRemove.forEach(key => {
      sessionStorage.removeItem(key)
      localStorage.removeItem(key)
    })
  } catch (error) {
    if (process.env.NODE_ENV === 'development') {
      console.error('清理存储失败:', error)
    }
  }
}
export const secureStorage = {
  get: getStorageItem,
  set: setStorageItem,
  remove: removeStorageItem,
  clear: clearSecureStorage,
  isSecureContext
}

