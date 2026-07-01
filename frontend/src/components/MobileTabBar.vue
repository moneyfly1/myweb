<template>
  <div class="mobile-tab-bar">
    <button
      v-for="item in tabs"
      :key="item.value"
      type="button"
      class="mobile-tab-bar__item"
      :class="{ active: item.value === modelValue }"
      :aria-pressed="item.value === modelValue"
      :title="item.label"
      @click="$emit('update:modelValue', item.value)"
    >
      <el-icon v-if="item.icon"><component :is="item.icon" /></el-icon>
      <span>{{ item.label }}</span>
      <em v-if="item.badge !== undefined">{{ item.badge }}</em>
    </button>
  </div>
</template>

<script setup>
defineProps({
  modelValue: {
    type: [String, Number],
    default: '',
  },
  tabs: {
    type: Array,
    default: () => [],
  },
})

defineEmits(['update:modelValue'])
</script>

<style scoped lang="scss">
.mobile-tab-bar {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(96px, 1fr);
  gap: 6px;
  padding: 4px;
  max-width: 100%;
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
  background: var(--el-fill-color-light, #f4f6f8);
  border: 1px solid var(--theme-border, #e5e7eb);
  border-radius: 8px;
  scrollbar-width: thin;
  -webkit-overflow-scrolling: touch;
}

.mobile-tab-bar__item {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  min-width: 0;
  min-height: 44px;
  padding: 0 8px;
  color: var(--el-text-color-regular, #606266);
  font-size: 13px;
  background: transparent;
  border: 0;
  border-radius: 6px;
  transition: background-color 0.16s ease, color 0.16s ease;
  touch-action: manipulation;

  span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  em {
    min-width: 18px;
    height: 18px;
    padding: 0 5px;
    color: var(--el-text-color-regular, #606266);
    font-size: 11px;
    font-style: normal;
    line-height: 18px;
    background: var(--card-bg, #fff);
    border-radius: 9px;
  }

  &.active {
    color: var(--theme-primary, #409eff);
    background: var(--card-bg, #fff);
    outline: 1px solid var(--el-color-primary-light-7, #d9ecff);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary, #409eff);
    outline-offset: 1px;
  }
}

@media (max-width: 480px) {
  .mobile-tab-bar {
    grid-auto-columns: minmax(104px, 1fr);
  }
}
</style>
