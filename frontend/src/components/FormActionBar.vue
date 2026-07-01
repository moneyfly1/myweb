<template>
  <div
    class="form-action-bar"
    :class="{ 'is-sticky': sticky, 'is-single-action': !showSubmit || !showCancel }"
  >
    <div v-if="$slots.left" class="form-action-bar__meta">
      <slot name="left"></slot>
    </div>
    <div class="form-action-bar__actions">
      <slot>
        <el-button v-if="showCancel" :disabled="loading" @click="$emit('cancel')">{{ cancelText }}</el-button>
        <el-button
          v-if="showSubmit"
          type="primary"
          :loading="loading"
          :disabled="disabled"
          @click="$emit('submit')"
        >
          {{ submitText }}
        </el-button>
      </slot>
    </div>
  </div>
</template>

<script setup>
defineProps({
  loading: {
    type: Boolean,
    default: false,
  },
  sticky: {
    type: Boolean,
    default: true,
  },
  cancelText: {
    type: String,
    default: '取消',
  },
  submitText: {
    type: String,
    default: '保存',
  },
  showCancel: {
    type: Boolean,
    default: true,
  },
  showSubmit: {
    type: Boolean,
    default: true,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
})

defineEmits(['cancel', 'submit'])
</script>

<style scoped lang="scss">
.form-action-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 0 0;
  background: var(--el-bg-color, #fff);
  min-width: 0;

  &.is-sticky {
    position: sticky;
    bottom: 0;
    z-index: 2;
    padding: 12px 0 calc(12px + env(safe-area-inset-bottom));
    border-top: 1px solid var(--theme-border, #dcdfe6);
  }
}

.form-action-bar__meta {
  min-width: 0;
  flex: 1;
  color: #606266;
  font-size: 13px;
  line-height: 1.5;
}

.form-action-bar__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  flex: 0 0 auto;
  min-width: 0;
  max-width: 100%;
  flex-wrap: wrap;

  :deep(.el-button) {
    margin-left: 0;
    min-width: 88px;
    touch-action: manipulation;
    white-space: normal;
  }
}

.form-action-bar.is-single-action {
  .form-action-bar__actions {
    justify-content: flex-end;
  }
}

@media (max-width: 768px) {
  .form-action-bar {
    align-items: stretch;
    flex-direction: column;
  }

  .form-action-bar__meta {
    font-size: 12px;
  }

  .form-action-bar__actions {
    width: 100%;

    :deep(.el-button) {
      flex: 1;
      min-width: 0;
      min-height: 44px;
      padding-inline: 12px;
    }
  }

  .form-action-bar.is-single-action {
    .form-action-bar__actions {
      justify-content: flex-end;

      :deep(.el-button) {
        flex: 0 1 auto;
        min-width: 96px;
      }
    }
  }
}
</style>
