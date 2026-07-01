<template>
  <div class="copyable-field">
    <span class="copyable-field__value" :title="copyValue">{{ displayValue }}</span>
    <el-button
      size="small"
      link
      type="primary"
      :disabled="isEmpty"
      :title="isEmpty ? '' : '复制'"
      aria-label="复制"
      @click.stop="copy"
    >
      复制
    </el-button>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'

const props = defineProps({
  value: {
    type: [String, Number],
    default: '',
  },
  empty: {
    type: String,
    default: '-',
  },
})

const isEmpty = computed(() => props.value === null || props.value === undefined || props.value === '')
const copyValue = computed(() => isEmpty.value ? '' : String(props.value))
const displayValue = computed(() => isEmpty.value ? props.empty : copyValue.value)

const copy = async () => {
  if (isEmpty.value) return

  try {
    await navigator.clipboard.writeText(copyValue.value)
    ElMessage.success('已复制')
  } catch (error) {
    console.error('复制失败:', error)
    ElMessage.error('复制失败，请手动复制')
  }
}
</script>

<style scoped>
.copyable-field {
  display: inline-flex;
  align-items: flex-start;
  gap: 6px;
  max-width: 100%;
  min-width: 0;
  vertical-align: top;
}

.copyable-field__value {
  min-width: 0;
  overflow-wrap: anywhere;
  word-break: break-word;
  line-height: 1.45;
}

.copyable-field :deep(.el-button) {
  flex: 0 0 auto;
  min-height: 24px;
  padding-inline: 2px;
  touch-action: manipulation;
  white-space: nowrap;
}

@media (max-width: 768px) {
  .copyable-field {
    align-items: center;
    gap: 8px;
  }

  .copyable-field :deep(.el-button) {
    min-height: 44px;
    min-width: 44px;
  }
}
</style>
