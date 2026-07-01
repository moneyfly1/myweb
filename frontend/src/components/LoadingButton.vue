<template>
  <el-button
    v-bind="$attrs"
    class="loading-button"
    :loading="loading"
    :disabled="disabled || loading"
    :aria-busy="loading ? 'true' : undefined"
    @click="handleClick"
  >
    <slot>{{ text }}</slot>
  </el-button>
</template>

<script setup>
const props = defineProps({
  loading: {
    type: Boolean,
    default: false,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  text: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['click'])

const handleClick = (event) => {
  if (props.loading || props.disabled) return
  emit('click', event)
}
</script>

<style scoped>
.loading-button {
  min-height: 36px;
  max-width: 100%;
  touch-action: manipulation;
  white-space: normal;
}

.loading-button :deep(span) {
  min-width: 0;
  overflow-wrap: anywhere;
}

@media (max-width: 768px) {
  .loading-button {
    min-height: 44px;
  }
}
</style>
