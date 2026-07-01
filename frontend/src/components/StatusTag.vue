<template>
  <el-tag
    class="status-tag"
    :type="resolved.type"
    :effect="effect"
    :size="size"
    :title="showTitle ? resolved.text : undefined"
  >
    {{ resolved.text }}
  </el-tag>
</template>

<script setup>
import { computed } from 'vue'
import { getStatusConfig } from '@/utils/statusMaps'

const props = defineProps({
  value: {
    type: [String, Number, Boolean],
    default: '',
  },
  map: {
    type: Object,
    default: () => ({}),
  },
  size: {
    type: String,
    default: 'small',
  },
  effect: {
    type: String,
    default: 'light',
  },
  showTitle: {
    type: Boolean,
    default: true,
  },
})

const resolved = computed(() => getStatusConfig(props.value, props.map))
</script>

<style scoped>
.status-tag {
  max-width: 100%;
  min-width: 0;
  vertical-align: middle;
}

.status-tag :deep(.el-tag__content) {
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
