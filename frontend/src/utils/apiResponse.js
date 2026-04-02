/**
 * 统一的 API 响应数据处理工具
 */

/**
 * 标准化 API 响应数据
 * 处理不同格式的响应数据，统一返回格式
 * @param {Object} response - Axios 响应对象
 * @returns {Object} 标准化后的数据
 */
export function normalizeResponse(response) {
  if (!response) return { data: {}, success: false }
  
  // 尝试多种数据结构
  const data = response?.data?.data ?? response?.data ?? {}
  const success = response?.data?.success ?? response?.status === 200 ?? true
  
  return {
    data,
    success,
    message: response?.data?.message ?? '',
    raw: response
  }
}

/**
 * 从响应中提取列表数据
 * 支持多种列表字段名
 * @param {Object} response - Axios 响应对象
 * @returns {Array} 列表数据
 */
export function extractList(response) {
  const { data } = normalizeResponse(response)
  return data.list ?? data.logs ?? data.records ?? data.items ?? data.data ?? []
}

/**
 * 从响应中提取分页信息
 * @param {Object} response - Axios 响应对象
 * @returns {Object} 分页信息 { page, size, total, pages }
 */
export function extractPagination(response) {
  const { data } = normalizeResponse(response)
  return {
    page: data.page ?? data.current ?? 1,
    size: data.size ?? data.page_size ?? data.pageSize ?? 20,
    total: data.total ?? 0,
    pages: data.pages ?? data.total_pages ?? Math.ceil((data.total ?? 0) / (data.size ?? data.page_size ?? 20))
  }
}

/**
 * 从响应中提取统计数据
 * @param {Object} response - Axios 响应对象
 * @returns {Object} 统计数据
 */
export function extractStatistics(response) {
  const { data } = normalizeResponse(response)
  return data.statistics ?? data.stats ?? data.count ?? {}
}

/**
 * 统一的分页配置
 */
export const PAGINATION_CONFIG = {
  pageSizes: [10, 20, 50, 100],
  layout: 'total, sizes, prev, pager, next, jumper',
  mobileLayout: 'total, prev, pager, next',
  defaultPageSize: 20,
  defaultPage: 1
}

/**
 * 获取分页布局（响应式）
 * @param {boolean} isMobile - 是否为移动端
 * @returns {string} 分页布局字符串
 */
export function getPaginationLayout(isMobile) {
  return isMobile ? PAGINATION_CONFIG.mobileLayout : PAGINATION_CONFIG.layout
}
