/**
 * API 响应缓存工具
 * 用于缓存频繁调用的 API 响应，减少重复请求
 */

class ApiCache {
  constructor() {
    this.cache = new Map()
    this.expiryTimes = new Map()
  }

  /**
   * 生成缓存键
   */
  generateKey(url, params = {}) {
    const paramStr = JSON.stringify(params)
    return `${url}:${paramStr}`
  }

  /**
   * 设置缓存
   * @param {string} key - 缓存键
   * @param {any} data - 缓存数据
   * @param {number} ttl - 过期时间（毫秒）
   */
  set(key, data, ttl = 300000) { // 默认5分钟
    this.cache.set(key, data)
    this.expiryTimes.set(key, Date.now() + ttl)
  }

  /**
   * 获取缓存
   * @param {string} key - 缓存键
   * @returns {any|null} 缓存数据或 null
   */
  get(key) {
    const expiryTime = this.expiryTimes.get(key)

    // 检查是否过期
    if (!expiryTime || Date.now() > expiryTime) {
      this.delete(key)
      return null
    }

    return this.cache.get(key)
  }

  /**
   * 删除缓存
   */
  delete(key) {
    this.cache.delete(key)
    this.expiryTimes.delete(key)
  }

  /**
   * 清空所有缓存
   */
  clear() {
    this.cache.clear()
    this.expiryTimes.clear()
  }

  /**
   * 清理过期缓存
   */
  cleanup() {
    const now = Date.now()
    for (const [key, expiryTime] of this.expiryTimes.entries()) {
      if (now > expiryTime) {
        this.delete(key)
      }
    }
  }

  /**
   * 包装 API 调用，自动处理缓存
   * @param {string} key - 缓存键
   * @param {Function} apiCall - API 调用函数
   * @param {number} ttl - 过期时间（毫秒）
   * @returns {Promise} API 响应
   */
  async wrap(key, apiCall, ttl = 300000) {
    // 尝试从缓存获取
    const cached = this.get(key)
    if (cached !== null) {
      return Promise.resolve(cached)
    }

    // 调用 API 并缓存结果
    try {
      const result = await apiCall()
      this.set(key, result, ttl)
      return result
    } catch (error) {
      throw error
    }
  }
}

// 创建全局缓存实例
export const apiCache = new ApiCache()

// 定期清理过期缓存（每5分钟）
if (typeof window !== 'undefined') {
  setInterval(() => {
    apiCache.cleanup()
  }, 300000)
}

export default apiCache
