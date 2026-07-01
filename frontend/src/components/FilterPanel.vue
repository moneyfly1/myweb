<template>
  <section class="filter-panel" aria-label="筛选条件">
    <div class="filter-panel__body">
      <slot></slot>
    </div>
    <div v-if="$slots.actions" class="filter-panel__actions">
      <slot name="actions"></slot>
    </div>
  </section>
</template>

<style scoped lang="scss">
.filter-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  background: var(--card-bg, #fff);
  border: 1px solid var(--theme-border, #ebeef5);
  border-radius: var(--border-radius, 8px);
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
}

.filter-panel__body {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 10px;
  flex: 1;
  min-width: 0;

  :deep(.el-input),
  :deep(.el-select),
  :deep(.el-date-editor),
  :deep(.el-cascader),
  :deep(.el-input-number),
  :deep(.el-autocomplete),
  :deep(.el-form-item) {
    width: 100%;
    min-width: 0;
  }

  :deep(.el-form-item) {
    margin-bottom: 0;
  }

  :deep(.el-form-item__content) {
    min-width: 0;
  }

  :deep(.el-input__wrapper),
  :deep(.el-select__wrapper) {
    min-width: 0;
  }

  :deep(.el-date-editor.el-input__wrapper),
  :deep(.el-date-editor .el-input__wrapper) {
    min-width: 0;
  }

  :deep(.el-date-editor--daterange),
  :deep(.el-date-editor--datetimerange) {
    max-width: 100%;
  }

  :deep(.el-range-input) {
    min-width: 0;
  }
}

.filter-panel__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
  flex-shrink: 0;
  min-width: 0;
  max-width: 100%;

  :deep(.el-button) {
    margin-left: 0;
    min-height: 36px;
    min-width: 72px;
    touch-action: manipulation;
    white-space: normal;
  }

  :deep(.el-button + .el-button) {
    margin-left: 0;
  }
}

@media (max-width: 768px) {
  .filter-panel {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-panel__body {
    grid-template-columns: 1fr;
  }

  .filter-panel__actions {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    width: 100%;
    min-width: 0;

    :deep(> *) {
      width: 100%;
      min-width: 0;
      max-width: 100%;
    }

    &:has(> :nth-child(1):last-child) {
      grid-template-columns: 1fr;
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
      min-height: 44px;
      min-width: 0;
      margin-left: 0;
    }
  }
}

@media (max-width: 420px) {
  .filter-panel {
    gap: 10px;
    padding: 10px;
  }

  .filter-panel__body {
    gap: 8px;
  }

  .filter-panel__actions {
    gap: 8px;
  }
}

@media (max-width: 360px) {
  .filter-panel__actions {
    grid-template-columns: 1fr;

    &:has(> :nth-child(3):last-child) :deep(> :nth-child(3)),
    &:has(> :nth-child(5):last-child) :deep(> :nth-child(4)) {
      grid-column: auto;
    }
  }
}
</style>
