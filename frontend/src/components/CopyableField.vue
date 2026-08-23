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

  // 优先使用 Clipboard API；非安全上下文（http/iframe）时降级为 execCommand
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(copyValue.value)
      ElMessage.success('已复制')
      return
    }
  } catch (error) {
    console.warn('Clipboard API 复制失败，尝试降级:', error)
  }

  // 降级：隐藏 textarea + execCommand（兼容 http 与旧浏览器）
  try {
    const textarea = document.createElement('textarea')
    textarea.value = copyValue.value
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    textarea.style.pointerEvents = 'none'
    document.body.appendChild(textarea)
    textarea.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(textarea)
    if (ok) {
      ElMessage.success('已复制')
    } else {
      ElMessage.info('请长按手动复制：' + copyValue.value)
    }
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
