<template>
  <el-dialog
    v-model="visible"
    class="app-dialog"
    :title="title"
    :width="computedWidth"
    :close-on-click-modal="closeOnClickModal && !loading"
    :close-on-press-escape="!loading"
    :before-close="handleBeforeClose"
    destroy-on-close
    append-to-body
    @open="$emit('open')"
    @opened="$emit('opened')"
    @closed="$emit('closed')"
  >
    <div class="app-dialog__body">
      <slot></slot>
    </div>
    <template v-if="$slots.footer" #footer>
      <slot name="footer"></slot>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed } from 'vue'
import { useMobile } from '@/composables/useMobile'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  title: {
    type: String,
    default: '',
  },
  width: {
    type: String,
    default: '560px',
  },
  mobileWidth: {
    type: String,
    default: '92%',
  },
  loading: {
    type: Boolean,
    default: false,
  },
  closeOnClickModal: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['update:modelValue', 'close-blocked', 'open', 'opened', 'closed'])
const isMobile = useMobile()

const visible = computed({
  get: () => props.modelValue,
  set: value => emit('update:modelValue', value),
})

const computedWidth = computed(() => isMobile.value ? props.mobileWidth : props.width)

const handleBeforeClose = (done) => {
  if (props.loading) {
    emit('close-blocked')
    return
  }
  done()
}
</script>

<style scoped>
.app-dialog__body {
  min-width: 0;
}

:deep(.app-dialog) {
  display: flex;
  flex-direction: column;
  max-width: calc(100dvw - 32px);
  max-height: calc(100vh - 12vh);
  border-radius: var(--border-radius, 8px);
}

:deep(.app-dialog .el-dialog__header) {
  padding: 18px 20px 12px;
  margin-right: 0;
  border-bottom: 1px solid #ebeef5;
}

:deep(.app-dialog .el-dialog__title) {
  color: var(--theme-text, #303133);
  font-size: 16px;
  font-weight: 600;
  line-height: 1.4;
}

:deep(.app-dialog .el-dialog__body) {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  padding: 18px 20px;
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
}

:deep(.app-dialog .el-dialog__footer) {
  flex: 0 0 auto;
  padding: 12px 20px 16px;
  border-top: 1px solid #ebeef5;
  background: var(--el-bg-color, #fff);
}

:deep(.app-dialog .el-dialog__footer > *) {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

:deep(.app-dialog .el-dialog__footer .el-button) {
  margin-left: 0;
}

@media (max-width: 768px) {
  :deep(.app-dialog) {
    max-width: calc(100dvw - 24px);
    max-height: calc(100dvh - 12vh);
    margin-top: 6vh;
  }

  :deep(.app-dialog .el-dialog__header) {
    padding: 16px 16px 10px;
  }

  :deep(.app-dialog .el-dialog__body) {
    padding: 14px 16px;
  }

  :deep(.app-dialog .el-dialog__footer) {
    padding: 12px 16px max(14px, env(safe-area-inset-bottom));
  }

  :deep(.app-dialog .el-dialog__footer > *) {
    width: 100%;
  }

  :deep(.app-dialog .el-dialog__footer .el-button) {
    flex: 1;
    min-height: 44px;
  }
}
</style>
