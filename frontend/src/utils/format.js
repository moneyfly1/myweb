export function formatMoney(value, options = {}) {
  const { prefix = '¥', empty = '0.00' } = options
  if (value === null || value === undefined || value === '') return `${prefix}${empty}`
  const number = Number(value)
  if (Number.isNaN(number)) return `${prefix}${empty}`
  return `${prefix}${number.toFixed(2)}`
}

/**
 * apiErrorMessage 从 axios 错误对象中取出可读的错误文案 —— 全站唯一的错误提示入口。
 *
 * 背景：前端各处的 catch / 业务失败分支里，散落着上百处手写模板：
 *   `error.response?.data?.message || error.message`
 *    `error.response?.data?.detail || error.response?.data?.message || error.message || 'xxx失败'`
 *   `'xxx失败: ' + (error.response?.data?.message || error.message)`
 * 三套字段名（message / detail / message）顺序不一、有的漏了 detail、有的漏了兜底，
 * 同一个后端错误在不同页面显示的文案因此不一致，也容易把 undefined / 空串直接抛给用户。
 *
 * 统一为一个函数后：
 * 1) 取值顺序固定：后端 message → 后端 detail → axios/JS 的 error.message → 调用方兜底文案；
 * 2) 永远不会返回 undefined / 空串（兜底默认「操作失败」），用户不会看到空白提示；
 *    但 axios 自己的通用英文文案（"Request failed with status code 500" / "Network Error" /
 *    "timeout of ..."）对用户没有信息量，这类一律退回中文兜底，避免中文界面冒出英文技术提示；
 * 3) 原来「'xxx失败: ' + 详情」的写法改为把「xxx失败」当兜底文案传入，
 *    后端有具体原因时直接显示原因，没有时才显示「xxx失败」，语义不变且不会出现「xxx失败: xxx失败」。
 *
 * @param {*} error 捕获到的错误（axios error / Error / undefined 均可）
 * @param {string} fallback 兜底文案（一般传业务动作名，如「保存失败」）
 * @returns {string}
 */
const GENERIC_AXIOS_MESSAGE = /^(Request failed with status code|Network Error|timeout of)/i

export function apiErrorMessage(error, fallback = '操作失败') {
  const data = error?.response?.data
  const serverMessage = data?.message || data?.detail
  if (serverMessage) return serverMessage
  const raw = typeof error?.message === 'string' ? error.message.trim() : ''
  if (!raw || GENERIC_AXIOS_MESSAGE.test(raw)) return fallback
  return raw
}

// formatMoneyPlain 不带货币符号的金额（模板里自己拼 ¥ 的场景用）。
// 此前每个页面各写一份 `(v) => formatMoneyUtil(v, { prefix: '' })` 薄包装，
// 四份行为完全一致，纯属重复。
export function formatMoneyPlain(value, empty = '0.00') {
  return formatMoney(value, { prefix: '', empty })
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
