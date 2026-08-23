import { ref, computed, onMounted, onUnmounted } from 'vue'
import { debounce } from '@/composables/useDebounce'
import { useMobile } from '@/composables/useMobile'

/**
 * 日志列表页公共逻辑（7 个日志页此前各自复制实现约 80% 同构代码）：
 * - loading/list/total/page/pageSize 状态
 * - fetch / debouncedFetch / resetFilter / onSizeChange
 * - 桌面端/移动端分页布局
 *
 * 用法：
 *   const { list, loading, total, page, pageSize, filter, isMobile,
 *           paginationLayout, fetchLogs, debouncedFetchLogs, resetFilter, onSizeChange } =
 *     useLogListPage({ fetcher, defaultFilter, extraFilterKeys })
 *   - fetcher(params) 必须返回 Promise，成功时返回 { list, total }（内部已解析 res.data.data）
 *   - filter 为 ref，各页面可继续扩展字段
 *   - extraFilterKeys: 除 keyword/timeRange 外需要透传到请求参数的 filter 键名数组
 */
export function useLogListPage({ fetcher, defaultFilter = () => ({ keyword: '', timeRange: null }), extraFilterKeys = [] }) {
  const loading = ref(false)
  const list = ref([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(20)
  const filter = ref(defaultFilter())
  const isMobile = useMobile()
  const paginationLayout = computed(() => (isMobile.value ? 'total, prev, pager, next' : 'total, prev, pager, next, sizes'))

  async function fetchLogs() {
    loading.value = true
    try {
      const params = { page: page.value, page_size: pageSize.value, ...buildFilterParams(filter.value) }
      const res = await fetcher(params)
      const data = res?.data?.data ?? res?.data ?? {}
      list.value = data.logs ?? data.list ?? []
      total.value = data.total ?? 0
    } catch (e) {
      list.value = []
      total.value = 0
    } finally {
      loading.value = false
    }
  }

  // 把 filter 对象转换为请求参数：keyword / start_time / end_time + 扩展键
  function buildFilterParams(f) {
    const params = {}
    if (f.keyword) params.keyword = f.keyword
    if (f.timeRange && f.timeRange.length === 2) {
      params.start_time = f.timeRange[0]
      params.end_time = f.timeRange[1]
    }
    for (const key of extraFilterKeys) {
      if (f[key] !== undefined && f[key] !== null && f[key] !== '') {
        params[key] = f[key]
      }
    }
    return params
  }

  const debouncedFetchLogs = debounce(fetchLogs, 500)

  function resetFilter() {
    filter.value = defaultFilter()
    page.value = 1
    fetchLogs()
  }

  function onSizeChange(size) {
    pageSize.value = size
    page.value = 1
    fetchLogs()
  }

  onMounted(() => { fetchLogs() })
  onUnmounted(() => {})

  return {
    loading, list, total, page, pageSize, filter, isMobile, paginationLayout,
    fetchLogs, debouncedFetchLogs, resetFilter, onSizeChange
  }
}
