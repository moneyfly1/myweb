<template>
  <el-alert
    class="tip-block"
    :type="type"
    :title="title"
    :description="description"
    :closable="closable"
    show-icon
  >
    <slot></slot>
  </el-alert>
</template>

<script setup>
defineProps({
  title: {
    type: String,
    default: '',
  },
  description: {
    type: String,
    default: '',
  },
  type: {
    type: String,
    default: 'info',
  },
  closable: {
    type: Boolean,
    default: false,
  },
})
</script>

<style scoped>
.tip-block {
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
  border-radius: var(--border-radius, 8px);
}

.tip-block :deep(.el-alert__content) {
  min-width: 0;
}

.tip-block :deep(.el-alert__title),
.tip-block :deep(.el-alert__description) {
  overflow-wrap: anywhere;
}

.tip-block :deep(.el-alert__description) {
  line-height: 1.6;
}
</style>
