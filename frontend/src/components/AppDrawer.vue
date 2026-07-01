<template>
  <el-drawer
    v-model="visible"
    class="app-drawer"
    :title="title"
    :size="computedSize"
    :direction="direction"
    :close-on-click-modal="closeOnClickModal && !loading"
    :close-on-press-escape="!loading"
    :before-close="handleBeforeClose"
    destroy-on-close
    append-to-body
    @open="$emit('open')"
    @opened="$emit('opened')"
    @closed="$emit('closed')"
  >
    <div class="app-drawer__body">
      <slot></slot>
    </div>
    <template v-if="$slots.footer" #footer>
      <slot name="footer"></slot>
    </template>
  </el-drawer>
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
  size: {
    type: String,
    default: '520px',
  },
  mobileSize: {
    type: String,
    default: '100%',
  },
  direction: {
    type: String,
    default: 'rtl',
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

const computedSize = computed(() => isMobile.value ? props.mobileSize : props.size)

const handleBeforeClose = (done) => {
  if (props.loading) {
    emit('close-blocked')
    return
  }
  done()
}
</script>

<style scoped>
.app-drawer__body {
  min-height: 100%;
  min-width: 0;
}

:deep(.app-drawer) {
  min-width: 0;
}

:deep(.app-drawer .el-drawer__header) {
  flex: 0 0 auto;
  align-items: center;
  padding: 18px 20px 12px;
  margin-bottom: 0;
  border-bottom: 1px solid #ebeef5;
  color: var(--theme-text, #303133);
  font-weight: 600;
}

:deep(.app-drawer .el-drawer__body) {
  flex: 1 1 auto;
  min-height: 0;
  min-width: 0;
  padding: 18px 20px;
  overflow: auto;
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
}

:deep(.app-drawer .el-drawer__footer) {
  flex: 0 0 auto;
  padding: 12px 20px 16px;
  border-top: 1px solid #ebeef5;
  background: var(--el-bg-color, #fff);
}

:deep(.app-drawer .el-drawer__footer > *) {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

:deep(.app-drawer .el-drawer__footer .el-button) {
  margin-left: 0;
}

@media (max-width: 768px) {
  :deep(.app-drawer .el-drawer__header) {
    padding: 16px 16px 10px;
  }

  :deep(.app-drawer .el-drawer__body) {
    padding: 14px 16px;
  }

  :deep(.app-drawer .el-drawer__footer) {
    padding: 12px 16px max(14px, env(safe-area-inset-bottom));
  }

  :deep(.app-drawer .el-drawer__footer > *) {
    width: 100%;
  }

  :deep(.app-drawer .el-drawer__footer .el-button) {
    flex: 1;
    min-height: 44px;
  }
}
</style>
