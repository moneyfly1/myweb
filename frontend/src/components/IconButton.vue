<template>
  <el-tooltip
    v-if="tooltip"
    :content="tooltip"
    placement="top"
    :show-after="300"
    :disabled="disabled || loading"
  >
    <el-button
      v-bind="$attrs"
      class="icon-button"
      :aria-label="ariaLabel || tooltip"
      :circle="circle"
      :loading="loading"
      :disabled="disabled || loading"
      @click="handleClick"
    >
      <el-icon v-if="icon && !loading">
        <component :is="icon" />
      </el-icon>
      <slot />
    </el-button>
  </el-tooltip>
  <el-button
    v-else
    v-bind="$attrs"
    class="icon-button"
    :aria-label="ariaLabel"
    :circle="circle"
    :loading="loading"
    :disabled="disabled || loading"
    @click="handleClick"
  >
    <el-icon v-if="icon && !loading">
      <component :is="icon" />
    </el-icon>
    <slot />
  </el-button>
</template>

<script setup>
const props = defineProps({
  icon: {
    type: [Object, Function, String],
    default: null,
  },
  tooltip: {
    type: String,
    default: '',
  },
  ariaLabel: {
    type: String,
    default: '',
  },
  circle: {
    type: Boolean,
    default: true,
  },
  loading: {
    type: Boolean,
    default: false,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['click'])

const handleClick = (event) => {
  if (props.loading || props.disabled) return
  emit('click', event)
}
</script>

<style scoped>
.icon-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  min-height: 32px;
  max-width: 100%;
  vertical-align: middle;
  touch-action: manipulation;
}

.icon-button :deep(.el-icon) {
  flex: 0 0 auto;
}

.icon-button :deep(span) {
  min-width: 0;
  overflow-wrap: anywhere;
}

@media (max-width: 768px) {
  .icon-button {
    min-width: 44px;
    min-height: 44px;
  }
}
</style>
