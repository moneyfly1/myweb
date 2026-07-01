import { reactive, toRaw } from 'vue'

export function usePersistentTableColumns(storageKey, defaultWidths = {}, columnKeys = []) {
  const columnWidths = reactive({ ...defaultWidths })

  const allowedKeys = new Set(columnKeys.length ? columnKeys : Object.keys(defaultWidths))

  const loadColumnWidths = () => {
    if (!storageKey || typeof localStorage === 'undefined') return
    try {
      const saved = JSON.parse(localStorage.getItem(storageKey) || '{}')
      const savedWidths = saved.columnWidths || saved
      Object.entries(savedWidths || {}).forEach(([key, width]) => {
        if ((!allowedKeys.size || allowedKeys.has(key)) && Number(width) > 0) {
          columnWidths[key] = Number(width)
        }
      })
    } catch (error) {
      console.warn('读取表格列宽失败:', error)
    }
  }

  const saveColumnWidths = () => {
    if (!storageKey || typeof localStorage === 'undefined') return
    try {
      localStorage.setItem(storageKey, JSON.stringify({ columnWidths: { ...toRaw(columnWidths) } }))
    } catch (error) {
      console.warn('保存表格列宽失败:', error)
    }
  }

  const handleColumnResize = (newWidth, _oldWidth, column) => {
    const key = column?.property || column?.columnKey || column?.label
    if (!key || (allowedKeys.size && !allowedKeys.has(key))) return
    columnWidths[key] = Number(newWidth)
    saveColumnWidths()
  }

  const resetColumnWidths = () => {
    Object.keys(columnWidths).forEach((key) => {
      delete columnWidths[key]
    })
    Object.assign(columnWidths, defaultWidths)
    saveColumnWidths()
  }

  loadColumnWidths()

  return {
    columnWidths,
    loadColumnWidths,
    saveColumnWidths,
    handleColumnResize,
    resetColumnWidths,
  }
}
