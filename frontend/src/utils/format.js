export function formatMoney(value, options = {}) {
  const { prefix = '¥', empty = '0.00' } = options
  if (value === null || value === undefined || value === '') return `${prefix}${empty}`
  const number = Number(value)
  if (Number.isNaN(number)) return `${prefix}${empty}`
  return `${prefix}${number.toFixed(2)}`
}

export function formatFileSize(bytes, empty = '-') {
  const number = Number(bytes)
  if (!Number.isFinite(number) || number < 0) return empty
  if (number === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(number) / Math.log(1024)), units.length - 1)
  return `${(number / Math.pow(1024, index)).toFixed(index === 0 ? 0 : 2)} ${units[index]}`
}

/**
 * unwrapList 从任意接口响应中取出列表数组 —— 全站唯一的解包入口。
 *
 * 背景：后端历史上列表字段名多达 15 种（list/items/logs/attempts/subscriptions/
 * records/orders/users/emails/tickets/coupons/relations/invite_codes/recharges…），
 * 分页元数据也有 size/page_size、total_pages/pages 两套命名，前端因此散落了
 * 100 多处 `data.logs || data.list || data.items` 之类的兜底解析。
 *
 * 现在后端已统一返回标准字段 list（同时保留旧字段名向后兼容），前端统一走本函数：
 * 1) 优先标准字段 list
 * 2) 兼容历史字段名（legacyKeys 或内置候选）
 * 3) 裸数组响应
 * 4) 都没有则返回 []
 *
 * @param {*} response 接口响应（axios response 或已解包的 data 均可）
 * @param {...string} legacyKeys 额外的历史字段名（一般不需要传）
 * @returns {Array}
 */
const LIST_KEYS = [
  'list', 'items', 'logs', 'attempts', 'records', 'rows',
  'subscriptions', 'orders', 'users', 'emails', 'tickets',
  'coupons', 'relations', 'invite_codes', 'recharges', 'categories'
]

export function unwrapList(response, ...legacyKeys) {
  if (response === null || response === undefined) return []

  // 逐层剥开 axios 的 data 与业务信封 data
  const candidates = []
  let node = response
  for (let depth = 0; depth < 3 && node && typeof node === 'object' && !Array.isArray(node); depth += 1) {
    candidates.push(node)
    if (node.data && typeof node.data === 'object') {
      node = node.data
    } else {
      break
    }
  }
  if (Array.isArray(node)) candidates.push({ list: node })

  const keys = [...(legacyKeys || []), ...LIST_KEYS]
  for (const candidate of candidates) {
    for (const key of keys) {
      const value = candidate?.[key]
      if (Array.isArray(value)) return value
    }
  }
  return []
}
