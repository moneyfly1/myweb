<template>
  <el-upload
    v-bind="$attrs"
    class="upload-field"
    :disabled="disabled || loading"
  >
    <slot>
      <el-button
        class="upload-field__button"
        :loading="loading"
        :disabled="disabled || loading"
        :aria-busy="loading ? 'true' : undefined"
      >
        {{ text }}
      </el-button>
    </slot>
    <template v-if="$slots.tip" #tip>
      <slot name="tip"></slot>
    </template>
  </el-upload>
</template>

<script setup>
defineProps({
  text: {
    type: String,
    default: '选择文件',
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
</script>

<style scoped>
.upload-field {
  max-width: 100%;
  min-width: 0;
}

.upload-field :deep(.el-upload) {
  max-width: 100%;
}

.upload-field__button {
  max-width: 100%;
  min-height: 36px;
  touch-action: manipulation;
  white-space: normal;
}

.upload-field__button :deep(span) {
  min-width: 0;
  overflow-wrap: anywhere;
}

.upload-field :deep(.el-upload__tip) {
  max-width: 100%;
  overflow-wrap: anywhere;
  line-height: 1.5;
}

@media (max-width: 768px) {
  .upload-field,
  .upload-field :deep(.el-upload) {
    width: 100%;
  }

  .upload-field__button {
    width: 100%;
    min-height: 44px;
  }
}
</style>
