<template>
  <div class="mobile-log-fields">
    <slot></slot>
  </div>
</template>

<script setup>
</script>

<style scoped lang="scss">
.mobile-log-fields {
  min-width: 0;
  max-width: 100%;
}

:deep(.mobile-log-title) {
  color: var(--el-text-color-primary);
  font-size: 15px;
  font-weight: 600;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

:deep(.mobile-log-subtitle) {
  margin-top: 3px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

:deep(.mobile-log-field) {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--theme-border, #f0f0f0);
  font-size: 14px;
  min-width: 0;
}

:deep(.mobile-log-field:last-child) {
  border-bottom: none;
}

:deep(.mobile-log-field.field-full) {
  flex-direction: column;
  gap: 4px;
}

:deep(.mobile-log-label) {
  flex: 0 0 72px;
  color: var(--el-text-color-secondary, #909399);
  line-height: 1.45;
  overflow-wrap: anywhere;
}

:deep(.mobile-log-value) {
  flex: 1;
  min-width: 0;
  color: var(--el-text-color-primary);
  line-height: 1.45;
  text-align: right;
  overflow-wrap: anywhere;
}

:deep(.mobile-log-wrap) {
  text-align: left;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

@media (max-width: 380px) {
  :deep(.mobile-log-field) {
    flex-direction: column;
    gap: 4px;
  }

  :deep(.mobile-log-label) {
    flex-basis: auto;
  }

  :deep(.mobile-log-value) {
    text-align: left;
  }
}
</style>
