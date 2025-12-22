// 地理位置工具函数

/**
 * 解析位置信息（从JSON字符串或逗号分隔字符串）
 * @param {string} locationStr - 位置字符串
 * @returns {Object} {country, city, region}
 */
export function parseLocation(locationStr) {
  if (!locationStr) {
    return { country: '', city: '', region: '' }
  }

  try {
    // 尝试解析JSON格式
    const locationData = JSON.parse(locationStr)
    return {
      country: locationData.country || '',
      city: locationData.city || '',
      region: locationData.region || '',
      countryCode: locationData.country_code || ''
    }
  } catch (e) {
    // 如果不是JSON，尝试解析逗号分隔格式
    if (locationStr.includes(',')) {
      const parts = locationStr.split(',').map(s => s.trim())
      return {
        country: parts[0] || '',
        city: parts[1] || '',
        region: parts[0] || '',
        countryCode: ''
      }
    }
    // 如果都不匹配，直接作为国家
    return {
      country: locationStr.trim(),
      city: '',
      region: locationStr.trim(),
      countryCode: ''
    }
  }
}

/**
 * 格式化位置显示（国家, 城市）
 * @param {string} locationStr - 位置字符串
 * @returns {string} 格式化后的位置字符串
 */
export function formatLocation(locationStr) {
  const location = parseLocation(locationStr)
  if (!location.country) {
    return ''
  }
  if (location.city) {
    return `${location.country}, ${location.city}`
  }
  return location.country
}

/**
 * 获取位置标签（用于显示）
 * @param {string} locationStr - 位置字符串
 * @param {string} ipAddress - IP地址（可选，用于提示）
 * @returns {Object} {text, tooltip}
 */
export function getLocationTag(locationStr, ipAddress = '') {
  const location = parseLocation(locationStr)
  if (!location.country) {
    return {
      text: '',
      tooltip: ipAddress || '未知位置'
    }
  }
  
  let text = location.country
  if (location.city) {
    text = `${location.country}, ${location.city}`
  }
  
  return {
    text,
    tooltip: ipAddress ? `${text} (${ipAddress})` : text
  }
}

