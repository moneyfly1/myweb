<template>
  <div class="mobile-card-list" role="list" :aria-busy="loading ? 'true' : 'false'">
    <SkeletonLoader v-if="loading" variant="list" :rows="4" :show-title="false" />
    <ErrorState
      v-else-if="error"
      :message="errorMessage"
      @retry="$emit('retry')"
    />
    <template v-else>
      <!-- 移动端多选：卡片列表原本没有任何勾选入口，导致"批量操作"在手机上不可达。
           这里提供「全选 + 已选计数 + 批量按钮插槽」，语义与桌面 el-table type="selection" 一致：
           全选只作用于**当前页**的数据（selectedRows 由父组件持有，双向同步）。 -->
      <div v-if="selectable && normalizedData.length > 0" class="mobile-card-list__selection">
        <el-checkbox
          class="mobile-card-list__select-all"
          :model-value="allSelected"
          :indeterminate="indeterminate"
          :disabled="selectableItems.length === 0"
          @change="toggleAll"
        >
          全选
        </el-checkbox>
        <span class="mobile-card-list__selection-count">
          已选 {{ selectedKeySet.size }}<template v-if="selectableItems.length !== normalizedData.length"> / {{ selectableItems.length }}</template>
        </span>
        <div class="mobile-card-list__selection-actions"><slot name="selection-actions" :selected="selectedRows" /></div>
      </div>
      <div class="mobile-card-list__items">
      <div
        v-for="(item, index) in normalizedData" 
        :key="item[idField] || index" 
        class="mobile-card"
        role="listitem"
      >
      <div class="mobile-card-header" v-if="$slots.header || hasTitleField || (selectable && rowSelectable(item))">
        <el-checkbox
          v-if="selectable && rowSelectable(item)"
          class="mobile-card-list__row-check"
          :model-value="isSelected(item)"
          @change="(val) => toggleRow(item, val)"
          @click.stop
        />
        <slot name="header" :item="item" :index="index">
          <div class="card-title" :title="getTitle(item)">{{ getTitle(item) }}</div>
        </slot>
      </div>
      
      <div class="mobile-card-body">
        <slot :item="item" :index="index">
          <div 
            v-for="field in fields" 
            :key="field.key" 
            class="card-field"
            :class="{ 'field-full': field.fullWidth }"
          >
            <span class="field-label" :title="field.label">{{ field.label }}</span>
            <span class="field-value" :title="getFieldTitle(item, field)">
              <slot :name="`field-${field.key}`" :item="item" :value="item[field.key]">
                <template v-if="field.type === 'tag'">
                  <el-tag 
                    v-if="!field.hideTagWhenEmpty || hasDisplayValue(item[field.key])"
                    :type="field.tagType ? field.tagType(item[field.key], item) : 'info'" 
                    size="small"
                    :title="getTagDisplayValue(item, field)"
                  >
                    <span class="field-tag-text">{{ getTagDisplayValue(item, field) }}</span>
                  </el-tag>
                </template>
                <template v-else-if="field.type === 'date'">
                  {{ formatDate(item[field.key], field.format) }}
                </template>
                <template v-else-if="field.type === 'money'">
                  ¥{{ formatMoney(item[field.key]) }}
                </template>
                <template v-else-if="field.type === 'copy'">
                  <CopyableField :value="getFieldCopyValue(item, field)" :empty="field.empty || '-'" />
                </template>
                <template v-else>
                  {{ formatValue(item[field.key], item, field) }}
                </template>
              </slot>
            </span>
          </div>
        </slot>
      </div>
      
      <div class="mobile-card-actions" v-if="$slots.actions">
        <slot name="actions" :item="item" :index="index"></slot>
      </div>
      </div>
    </div>
    </template>
    
    <div v-if="!loading && !error && normalizedData.length === 0" class="mobile-card-empty">
      <slot name="empty">
        <EmptyState :title="emptyTitle" :description="emptyDescription" />
      </slot>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import dayjs from 'dayjs'
import { formatDateTimeSafe } from '@/utils/date'
import { formatMoney as formatMoneyUtil } from '@/utils/format'
import SkeletonLoader from './SkeletonLoader.vue'
import ErrorState from './ErrorState.vue'
import EmptyState from './EmptyState.vue'
import CopyableField from './CopyableField.vue'

const props = defineProps({
  data: {
    type: Array,
    required: true,
    default: () => []
  },
  idField: {
    type: String,
    default: 'id'
  },
  titleField: {
    type: String,
    default: 'name'
  },
  fields: {
    type: Array,
    default: () => []
  },
  dateFormat: {
    type: String,
    default: 'YYYY-MM-DD HH:mm:ss'
  },
  loading: {
    type: Boolean,
    default: false
  },
  loadingText: {
    type: String,
    default: '加载中...'
  },
  error: {
    type: [Boolean, Object, String],
    default: false
  },
  emptyTitle: {
    type: String,
    default: '暂无数据'
  },
  emptyDescription: {
    type: String,
    default: ''
  },
  // ===== 移动端多选（可选）=====
  // selectable: 打开后卡片前出现复选框，并在列表顶部显示「全选 + 已选计数」
  selectable: {
    type: Boolean,
    default: false
  },
  // selectedRows: 父组件持有的已选行（数组）。与桌面 el-table 的 selection 语义一致，
  // 组件内部按 idField 派生勾选状态，勾选变化时通过 selection-change 回传新的行数组。
  selectedRows: {
    type: Array,
    default: () => []
  },
  // isRowSelectable: 逐行可选性判定（如订单里"已支付"不允许批量取消）
  isRowSelectable: {
    type: Function,
    default: null
  }
})

const emit = defineEmits(['retry', 'selection-change'])

const normalizedData = computed(() => Array.isArray(props.data) ? props.data : [])

// ===== 移动端多选 =====
const rowKey = (item) => String(item?.[props.idField])
const selectedKeySet = computed(() => new Set((props.selectedRows || []).map(rowKey)))
const rowSelectable = (item) => (typeof props.isRowSelectable === 'function' ? !!props.isRowSelectable(item) : true)
const selectableItems = computed(() => normalizedData.value.filter(rowSelectable))
const allSelected = computed(() => selectableItems.value.length > 0 && selectableItems.value.every(isSelected))
const indeterminate = computed(() => {
  const n = selectableItems.value.filter(isSelected).length
  return n > 0 && n < selectableItems.value.length
})
function isSelected(item) {
  return selectedKeySet.value.has(rowKey(item))
}
function emitSelection(rows) {
  emit('selection-change', rows)
}
function toggleRow(item, checked) {
  const key = rowKey(item)
  const cur = [...(props.selectedRows || [])]
  if (checked) {
    if (!cur.some(r => rowKey(r) === key)) cur.push(item)
  } else {
    const i = cur.findIndex(r => rowKey(r) === key)
    if (i >= 0) cur.splice(i, 1)
  }
  emitSelection(cur)
}
// 与 el-table 表头全选一致：只作用于当前页
function toggleAll(checked) {
  if (!checked) return emitSelection([])
  const cur = [...(props.selectedRows || [])]
  selectableItems.value.forEach((item) => {
    if (!cur.some(r => rowKey(r) === rowKey(item))) cur.push(item)
  })
  emitSelection(cur)
}

const errorMessage = computed(() => {
  if (typeof props.error === 'string') return props.error
  return props.error?.message || '数据加载失败'
})

const formatDate = (date, format) => {
  if (!date) return '-'
  const parsed = dayjs(date)
  if (!parsed.isValid()) return '-'
  return formatDateTimeSafe(date, format || props.dateFormat, '-')
}

const formatMoney = (value) => {
  return formatMoneyUtil(value, { prefix: '' })
}

const formatValue = (value, item, field) => {
  if (field.formatter) return field.formatter(value, item)
  if (value === null || value === undefined || value === '') return '-'
  return value
}

const toDisplayText = (value) => {
  if (value === null || value === undefined || value === '') return '-'
  return String(value)
}

const hasDisplayValue = (value) => value !== null && value !== undefined && value !== ''

const hasTitleField = computed(() => typeof props.titleField === 'string' && props.titleField.length > 0)

const getTitle = (item) => toDisplayText(item?.[props.titleField])

const getFieldDisplayValue = (item, field) => toDisplayText(formatValue(item?.[field.key], item, field))

const getTagDisplayValue = (item, field) => getFieldDisplayValue(item, field)

const getFieldCopyValue = (item, field) => {
  if (field.copyValue) return field.copyValue(item?.[field.key], item)
  return formatValue(item?.[field.key], item, field)
}

const getFieldTitle = (item, field) => {
  if (field.type === 'copy') return toDisplayText(getFieldCopyValue(item, field))
  if (field.type === 'date') return formatDate(item?.[field.key], field.format)
  if (field.type === 'money') return `¥${formatMoney(item?.[field.key])}`
  return getFieldDisplayValue(item, field)
}
</script>

<style scoped lang="scss">
.mobile-card-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 0;
  width: 100%;
  min-width: 0;
  max-width: 100%;
  overscroll-behavior: contain;
}

.mobile-card {
  background: var(--card-bg, #fff);
  border: 1px solid #ebeef5;
  border-radius: 8px;
  box-shadow: none;
  overflow: hidden;
  transition: border-color 0.16s ease, background-color 0.16s ease;
  min-width: 0;
  max-width: 100%;
  
  &:active {
    border-color: #c6e2ff;
    background: #f8fbff;
  }
}

.mobile-card-list__selection {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  padding: 8px 10px;
  margin-bottom: 2px;
  background: var(--card-bg, #fff);
  border: 1px solid #ebeef5;
  border-radius: 8px;
  position: sticky;
  top: 0;
  z-index: 6;
}

.mobile-card-list__select-all {
  flex: 0 0 auto;
  height: auto;
  margin-right: 0;
  font-weight: 500;
}

.mobile-card-list__selection-count {
  flex: 1 1 auto;
  min-width: 0;
  font-size: 12px;
  color: var(--theme-text-secondary, #909399);
  white-space: nowrap;
}

.mobile-card-list__selection-actions {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;

  &:empty {
    display: none;
  }
}

.mobile-card-header {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px;
  background: #f8fafc;
  border-bottom: 1px solid #ebeef5;
  color: var(--theme-text, #303133);
  min-width: 0;
  
  .card-title {
    flex: 1 1 auto;
    font-size: 15px;
    font-weight: 600;
    line-height: 1.4;
    min-width: 0;
    max-width: 100%;
    word-break: break-word;
  }

  .mobile-card-list__row-check {
    flex: 0 0 auto;
    height: auto;
    margin-right: 0;
    /* 触控热区：复选框本身太小，撑到 ~40px 便于手机点选 */
    padding: 8px 2px;
    margin: -8px 0;
  }
}

.mobile-card-body {
  padding: 10px 12px;
}

.card-field {
  display: grid;
  grid-template-columns: minmax(82px, 0.36fr) minmax(0, 1fr);
  align-items: flex-start;
  gap: 10px;
  padding: 8px 0;
  border-bottom: 1px solid var(--theme-border, #f0f0f0);
  min-width: 0;
  
  &:last-child {
    border-bottom: none;
  }
  
  &.field-full {
    grid-template-columns: 1fr;
    gap: 4px;

    .field-label {
      max-width: none;
    }

    .field-value {
      width: 100%;
      text-align: left;
      word-break: break-word;
    }
  }
}

.field-label {
  font-size: 13px;
  color: #909399;
  line-height: 1.45;
  min-width: 0;
  overflow-wrap: anywhere;
}

.field-value {
  display: inline-flex;
  justify-content: flex-end;
  align-items: flex-start;
  font-size: 14px;
  color: var(--theme-text, #303133);
  text-align: right;
  flex: 1;
  min-width: 0;
  max-width: 100%;
  word-break: break-all;
  overflow-wrap: anywhere;
  line-height: 1.45;

  :deep(.el-tag) {
    max-width: 100%;
    min-width: 0;
  }

  :deep(.el-tag__content) {
    min-width: 0;
    max-width: 100%;
  }

  :deep(.copyable-field) {
    width: 100%;
    justify-content: flex-end;
  }

  :deep(.copyable-field__value) {
    flex: 1 1 auto;
    min-width: 0;
    text-align: right;
  }
}

.field-tag-text {
  display: inline-block;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: top;
  white-space: nowrap;
}

.mobile-card-actions {
  padding: 10px 12px 12px;
  border-top: 1px solid var(--theme-border, #f0f0f0);
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  align-items: stretch;
  min-width: 0;

  :deep(> *) {
    width: 100%;
    min-width: 0;
    max-width: 100%;
  }

  :deep(> a),
  :deep(> .el-dropdown) {
    display: block;
  }

  &:has(> :nth-child(1):last-child) {
    grid-template-columns: minmax(0, 1fr);
  }

  &:has(> :nth-child(2):last-child),
  &:has(> :nth-child(3):last-child),
  &:has(> :nth-child(4):last-child) {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  &:has(> :nth-child(3):last-child) :deep(> :nth-child(3)),
  &:has(> :nth-child(5):last-child) :deep(> :nth-child(4)) {
    grid-column: span 2;
  }
  
  :deep(.el-button) {
    width: 100%;
    min-width: 0;
    max-width: 100%;
    min-height: 44px;
    margin-left: 0;
    padding-left: 6px;
    padding-right: 6px;
    white-space: normal;
    overflow-wrap: anywhere;
    word-break: keep-all;
    touch-action: manipulation;
  }

  :deep(.el-button > span) {
    min-width: 0;
    max-width: 100%;
    white-space: normal;
    overflow-wrap: anywhere;
    line-height: 1.25;
  }
}

.mobile-card-empty {
  padding: 24px 0;
  width: 100%;
  min-width: 0;
}

@media (prefers-reduced-motion: reduce) {
  .mobile-card {
    transition: none;
  }
}

@media (max-width: 420px) {
  .mobile-card-list {
    gap: 10px;
  }

  .card-field {
    gap: 8px;
    grid-template-columns: minmax(76px, 0.34fr) minmax(0, 1fr);
  }

  .mobile-card-actions :deep(.el-button) {
    min-width: 0;
  }
}

@media (max-width: 360px) {
  .card-field {
    grid-template-columns: 1fr;
    gap: 4px;
  }

  .field-value {
    justify-content: flex-start;
    text-align: left;
  }

  .field-value :deep(.copyable-field) {
    justify-content: flex-start;
  }

  .field-value :deep(.copyable-field__value) {
    text-align: left;
  }

  .mobile-card-actions :deep(.el-button) {
    padding-left: 4px;
    padding-right: 4px;
  }
}
</style>
