<template>
  <div class="prose-content">
    <slot></slot>
  </div>
</template>

<style scoped lang="scss">
.prose-content {
  color: var(--theme-text, #303133);
  font-size: 14px;
  line-height: 1.75;
  max-width: 100%;
  min-width: 0;
  overflow-wrap: anywhere;

  :deep(p) {
    margin: 0 0 12px;
  }

  :deep(h1),
  :deep(h2),
  :deep(h3) {
    margin: 20px 0 10px;
    color: var(--el-text-color-primary, #1f2937);
    line-height: 1.35;
    overflow-wrap: anywhere;
  }

  :deep(pre) {
    overflow: auto;
    padding: 12px;
    background: #0f172a;
    color: #e5e7eb;
    border-radius: 8px;
    -webkit-overflow-scrolling: touch;
  }

  :deep(code) {
    font-size: 0.92em;
    overflow-wrap: anywhere;
  }

  :deep(img) {
    max-width: 100%;
    height: auto;
    border-radius: 8px;
  }

  :deep(table) {
    display: block;
    max-width: 100%;
    overflow-x: auto;
    border-collapse: collapse;
    -webkit-overflow-scrolling: touch;
  }

  :deep(a) {
    color: var(--theme-primary, #409eff);
    overflow-wrap: anywhere;
  }
}
</style>
