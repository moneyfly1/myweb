import { ElMessageBox } from '@/utils/elementPlusServices'

const defaultDangerOptions = {
  confirmButtonText: '确认执行',
  cancelButtonText: '取消',
  closeOnClickModal: false,
  closeOnPressEscape: true,
  dangerouslyUseHTMLString: false,
}

export function confirmAction(message, title = '确认操作', options = {}) {
  return ElMessageBox.confirm(message, title, {
    ...defaultDangerOptions,
    type: options.type || 'warning',
    confirmButtonText: options.confirmButtonText || defaultDangerOptions.confirmButtonText,
    cancelButtonText: options.cancelButtonText || defaultDangerOptions.cancelButtonText,
    closeOnClickModal: options.closeOnClickModal ?? defaultDangerOptions.closeOnClickModal,
    closeOnPressEscape: options.closeOnPressEscape ?? defaultDangerOptions.closeOnPressEscape,
    dangerouslyUseHTMLString: options.dangerouslyUseHTMLString ?? defaultDangerOptions.dangerouslyUseHTMLString,
    ...options,
  })
}

export function confirmDanger(message, options = {}) {
  return confirmAction(message, options.title || '危险操作确认', {
    type: 'error',
    confirmButtonText: '确认执行',
    ...options,
  })
}

export function confirmWarning(message, options = {}) {
  return confirmAction(message, options.title || '确认操作', {
    type: 'warning',
    confirmButtonText: '确认操作',
    ...options,
  })
}

export function confirmDelete(entityName = '数据', count = 1, options = {}) {
  const message = options.message || (count > 1
    ? `确定删除选中的 ${count} 条${entityName}吗？删除后不可恢复。`
    : `确定删除该${entityName}吗？删除后不可恢复。`)

  return confirmDanger(message, {
    title: options.title || '确认删除',
    confirmButtonText: '确认删除',
    ...options,
  })
}

export function confirmReset(entityName = '数据', options = {}) {
  const message = options.message || `确定重置${entityName}吗？重置后原有状态可能立即失效。`

  return confirmDanger(message, {
    title: options.title || '确认重置',
    confirmButtonText: '确认重置',
    ...options,
  })
}

export function confirmClear(entityName = '数据', options = {}) {
  const message = options.message || `确定清理${entityName}吗？该操作会立即生效。`

  return confirmWarning(message, {
    title: options.title || '确认清理',
    confirmButtonText: '确认清理',
    ...options,
  })
}
