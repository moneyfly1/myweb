import { ref, reactive, computed } from 'vue'
import { debounce } from './useDebounce'

/**
 * 列表管理 Composable
 * @param {Function} fetchFn - 获取列表数据的函数
 * @param {Object} options - 配置选项
 * @returns {Object} - 列表管理相关的状态和方法
 */
export function useListManagement(fetchFn, options = {}) {
  const {
    pageSize: initialPageSize = 20,
    searchDelay = 500,
    autoLoad = true
  } = options

  // 状态
  const loading = ref(false)
  const list = ref([])
  const total = ref(0)
  const currentPage = ref(1)
  const pageSize = ref(initialPageSize)
  const searchKeyword = ref('')
  const filters = reactive({})

  // 计算属性
  const hasData = computed(() => list.value.length > 0)
  const isEmpty = computed(() => !loading.value && list.value.length === 0)

  // 加载数据
  const loadData = async (resetPage = false) => {
    if (resetPage) {
      currentPage.value = 1
    }

    loading.value = true
    try {
      const params = {
        page: currentPage.value,
        page_size: pageSize.value,
        keyword: searchKeyword.value,
        ...filters
      }

      const response = await fetchFn(params)
      const data = response?.data?.data ?? response?.data ?? {}

      list.value = data.list || data.logs || data.records || data.items || []
      total.value = data.total ?? 0
    } catch (error) {
      console.error('加载数据失败:', error)
      list.value = []
      total.value = 0
      throw error
    } finally {
      loading.value = false
    }
  }

  // 防抖搜索
  const debouncedLoad = debounce(() => loadData(true), searchDelay)

  // 搜索
  const search = (keyword) => {
    searchKeyword.value = keyword
    debouncedLoad()
  }

  // 重置筛选
  const resetFilters = () => {
    searchKeyword.value = ''
    Object.keys(filters).forEach(key => {
      filters[key] = ''
    })
    loadData(true)
  }

  // 分页变化
  const handlePageChange = (page) => {
    currentPage.value = page
    loadData()
  }

  // 每页数量变化
  const handleSizeChange = (size) => {
    pageSize.value = size
    currentPage.value = 1
    loadData()
  }

  // 刷新
  const refresh = () => {
    loadData()
  }

  // 自动加载
  if (autoLoad) {
    loadData()
  }

  return {
    // 状态
    loading,
    list,
    total,
    currentPage,
    pageSize,
    searchKeyword,
    filters,
    hasData,
    isEmpty,

    // 方法
    loadData,
    search,
    resetFilters,
    handlePageChange,
    handleSizeChange,
    refresh
  }
}
