<template>
  <span
    class="price-display"
    :class="{ 'is-strong': strong, 'is-muted': muted }"
    :title="showTitle ? formatted : undefined"
  >
    {{ formatted }}
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { formatMoney } from '@/utils/format'

const props = defineProps({
  value: {
    type: [Number, String],
    default: 0,
  },
  prefix: {
    type: String,
    default: '¥',
  },
  strong: {
    type: Boolean,
    default: false,
  },
  muted: {
    type: Boolean,
    default: false,
  },
  showTitle: {
    type: Boolean,
    default: true,
  },
})

const formatted = computed(() => formatMoney(props.value, { prefix: props.prefix }))
</script>

<style scoped>
.price-display {
  display: inline-block;
  max-width: 100%;
  min-width: 0;
  color: var(--el-text-color-primary, #303133);
  font-variant-numeric: tabular-nums;
  line-height: 1.3;
  overflow-wrap: anywhere;
}

.price-display.is-strong {
  color: var(--theme-danger, #f56c6c);
  font-weight: 700;
}

.price-display.is-muted {
  color: var(--el-text-color-secondary, #909399);
  font-weight: 400;
}
</style>
