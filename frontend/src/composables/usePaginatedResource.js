import { computed, reactive, ref } from 'vue'
import { debounce } from './useDebounce'
import { ElMessage } from '@/utils/elementPlusServices'

const getByPath = (source, path) => {
  if (!path) return undefined
  return path.split('.').reduce((value, key) => value?.[key], source)
}

const firstDefinedPath = (source, paths) => {
  for (const path of paths) {
    const value = getByPath(source, path)
    if (value !== undefined) return value
  }
  return undefined
}

export function usePaginatedResource(fetchFn, options = {}) {
  const {
    pageSize: initialPageSize = 10,
    pageParam = 'page',
    pageSizeParam = 'page_size',
    listPaths = ['data.data.list', 'data.data.items', 'data.data.records', 'data.data.logs', 'data.list', 'data.items', 'data.records', 'data.logs', 'list', 'items', 'records', 'logs'],
    totalPaths = ['data.data.total', 'data.total', 'total'],
    searchParam = 'keyword',
    searchDelay = 400,
    defaultFilters = {},
    autoLoad = true,
    mapParams,
    normalizeResponse,
    showError = true,
  } = options

  const loading = ref(false)
  const error = ref(null)
  const list = ref([])
  const total = ref(0)
  const currentPage = ref(1)
  const pageSize = ref(initialPageSize)
  const searchKeyword = ref('')
  const filters = reactive({ ...defaultFilters })
  let requestId = 0

  const hasData = computed(() => list.value.length > 0)
  const isEmpty = computed(() => !loading.value && !error.value && list.value.length === 0)

  const buildParams = () => {
    const params = {
      [pageParam]: currentPage.value,
      [pageSizeParam]: pageSize.value,
      ...filters,
    }
    if (searchParam && searchKeyword.value) params[searchParam] = searchKeyword.value
    return typeof mapParams === 'function' ? mapParams(params) : params
  }

  const loadData = async (resetPage = false) => {
    if (resetPage) currentPage.value = 1
    const currentRequest = ++requestId
    loading.value = true
    error.value = null

    try {
      const response = await fetchFn(buildParams())
      if (currentRequest !== requestId) return response

      if (typeof normalizeResponse === 'function') {
        const normalized = normalizeResponse(response)
        list.value = normalized.list || []
        total.value = Number(normalized.total || 0)
      } else {
        const nextList = firstDefinedPath(response, listPaths)
        const nextTotal = firstDefinedPath(response, totalPaths)
        list.value = Array.isArray(nextList) ? nextList : []
        total.value = Number(nextTotal ?? list.value.length ?? 0)
      }
      return response
    } catch (err) {
      if (currentRequest !== requestId) return undefined
      error.value = err
      list.value = []
      total.value = 0
      if (showError) ElMessage.error(err?.response?.data?.message || err?.message || '加载数据失败')
      return undefined
    } finally {
      if (currentRequest === requestId) loading.value = false
    }
  }

  const debouncedLoad = debounce(() => loadData(true), searchDelay)

  const search = (keyword) => {
    searchKeyword.value = keyword
    debouncedLoad()
  }

  const resetFilters = () => {
    searchKeyword.value = ''
    Object.keys(filters).forEach((key) => {
      filters[key] = defaultFilters[key] ?? ''
    })
    loadData(true)
  }

  const handlePageChange = (page) => {
    currentPage.value = page
    loadData()
  }

  const handleSizeChange = (size) => {
    pageSize.value = size
    currentPage.value = 1
    loadData()
  }

  if (autoLoad) loadData()

  return {
    loading,
    error,
    list,
    total,
    currentPage,
    pageSize,
    searchKeyword,
    filters,
    hasData,
    isEmpty,
    loadData,
    search,
    resetFilters,
    handlePageChange,
    handleSizeChange,
    refresh: loadData,
  }
}
