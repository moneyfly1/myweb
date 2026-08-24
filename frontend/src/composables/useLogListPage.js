import { ref, computed, onMounted, onUnmounted } from 'vue'
import { debounce } from '@/composables/useDebounce'
import { useMobile } from '@/composables/useMobile'
import { ElMessage, ElMessageBox } from '@/utils/elementPlusServices'

/**
 * 日志列表页公共逻辑（7 个日志页此前各自复制实现约 80% 同构代码）：
 * - loading/list/total/page/pageSize 状态
 * - fetch / debouncedFetch / resetFilter / onSizeChange
 * - 桌面端/移动端分页布局（sizes + jumper 全能力）
 * - 可选批量清理（clearAction 传入后返回 runCleanup）
 *
 * 用法：
 *   const { list, loading, total, page, pageSize, filter, isMobile,
 *           paginationLayout, fetchLogs, debouncedFetchLogs, resetFilter, onSizeChange,
 *           clearing, runCleanup } =
 *     useLogListPage({ fetcher, defaultFilter, extraFilterKeys, clearAction, clearLabel })
 *   - fetcher(params) 必须返回 Promise，成功时返回 { list, total }（内部已解析 res.data.data）
 *   - filter 为 ref，各页面可继续扩展字段
 *   - extraFilterKeys: 除 keyword/timeRange 外需要透传到请求参数的 filter 键名数组
 *   - clearAction: 可选，返回 Promise 的清理函数（如 () => adminAPI.cleanupData('balance_logs')）
 *   - clearLabel: 清理确认文案中的数据类型名
 */
export function useLogListPage({ fetcher, defaultFilter = () => ({ keyword: '', timeRange: null }), extraFilterKeys = [], clearAction = null, clearLabel = '' }) {
  const loading = ref(false)
  const list = ref([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(20)
  const filter = ref(defaultFilter())
  const isMobile = useMobile()
  // 统一分页能力：每页条数选择（sizes）+ 页码跳转（jumper），桌面与移动端一致
  const paginationLayout = computed(() => (isMobile.value ? 'sizes, prev, pager, next, jumper' : 'total, sizes, prev, pager, next, jumper'))
  const clearing = ref(false)

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

  // 批量清理（可选）：调用 clearAction 后回到第一页并刷新
  async function runCleanup() {
    if (!clearAction) return
    try {
      await ElMessageBox.confirm(
        `确定要清空${clearLabel || '该类型'}数据吗？此操作不可恢复！`,
        '清理确认',
        { type: 'warning', confirmButtonText: '确认清理', cancelButtonText: '取消', confirmButtonClass: 'el-button--danger' }
      )
    } catch {
      return
    }
    clearing.value = true
    try {
      const res = await clearAction()
      const count = res?.data?.data?.deleted_count ?? 0
      ElMessage.success(`已清理 ${count} 条${clearLabel || ''}`)
      page.value = 1
      await fetchLogs()
    } catch (e) {
      ElMessage.error(e?.response?.data?.message || '清理失败')
    } finally {
      clearing.value = false
    }
  }

  onMounted(() => { fetchLogs() })
  onUnmounted(() => {})

  return {
    loading, list, total, page, pageSize, filter, isMobile, paginationLayout, clearing,
    fetchLogs, debouncedFetchLogs, resetFilter, onSizeChange, runCleanup
  }
}
