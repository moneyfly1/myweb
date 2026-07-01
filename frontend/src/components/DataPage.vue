<template>
  <div class="data-page">
    <section v-if="title || description || $slots.actions" class="page-header data-page__header">
      <div class="page-title data-page__heading">
        <h1 v-if="title">{{ title }}</h1>
        <p v-if="description">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="data-page__actions">
        <slot name="actions"></slot>
      </div>
    </section>

    <section v-if="$slots.stats" class="data-page__stats">
      <slot name="stats"></slot>
    </section>

    <section v-if="$slots.filters" class="data-page__filters">
      <slot name="filters"></slot>
    </section>

    <section class="data-page__content">
      <slot></slot>
    </section>
  </div>
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
})
</script>

<style scoped lang="scss">
.data-page {
  width: 100%;
  min-width: 0;
}

.data-page__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
  padding: 16px;
  background: var(--card-bg, #fff);
  border: 1px solid var(--theme-border, #dcdfe6);
  border-radius: var(--border-radius, 8px);
  box-sizing: border-box;
  min-width: 0;
}

.data-page__heading {
  min-width: 0;
  flex: 1 1 auto;

  h1 {
    margin: 0;
    color: var(--theme-text, #303133);
    font-size: 22px;
    line-height: 1.25;
    overflow-wrap: anywhere;
  }

  p {
    margin: 6px 0 0;
    color: var(--el-text-color-regular, #606266);
    font-size: 14px;
    line-height: 1.5;
    max-width: 720px;
    overflow-wrap: anywhere;
  }
}

.data-page__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
  flex: 0 0 auto;
  max-width: 100%;

  :deep(.el-button) {
    margin-left: 0;
    min-height: 36px;
    touch-action: manipulation;
    white-space: normal;
  }
}

.data-page__stats,
.data-page__filters {
  margin-bottom: 12px;
  min-width: 0;
}

.data-page__content {
  min-width: 0;
}

@media (max-width: 768px) {
  .data-page__header {
    flex-direction: column;
    padding: 12px;
  }

  .data-page__heading h1 {
    font-size: 18px;
  }

  .data-page__actions {
    width: 100%;

    :deep(.el-button) {
      flex: 1;
      min-width: 0;
      min-height: 44px;
      margin-left: 0;
    }
  }
}
</style>
