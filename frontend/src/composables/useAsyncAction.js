import { ref } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'

export function useAsyncAction(options = {}) {
  const {
    successMessage = '',
    errorMessage = '操作失败',
    showSuccess = true,
    showError = true,
  } = options

  const loading = ref(false)
  const error = ref(null)

  const run = async (action, actionOptions = {}) => {
    if (loading.value) return undefined
    loading.value = true
    error.value = null

    try {
      const result = await action()
      const message = actionOptions.successMessage ?? successMessage
      if (showSuccess && message) ElMessage.success(message)
      return result
    } catch (err) {
      error.value = err
      const message = actionOptions.errorMessage ?? err?.response?.data?.message ?? err?.message ?? errorMessage
      if (showError && message) ElMessage.error(message)
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    loading,
    error,
    run,
  }
}
