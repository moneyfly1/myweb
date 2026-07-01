<template>
  <div class="inline-editable-text">
    <el-input
      v-if="editing"
      ref="inputRef"
      v-model="draft"
      size="small"
      :maxlength="maxlength"
      :placeholder="placeholder"
      :disabled="loading"
      @keyup.enter="save"
      @keyup.esc="cancel"
      @blur="save"
    />
    <button
      v-else
      type="button"
      class="inline-editable-text__display"
      :disabled="disabled"
      :title="value || emptyText"
      :aria-label="`编辑${value || emptyText}`"
      @click="startEdit"
    >
      <span>{{ value || emptyText }}</span>
      <el-icon><Edit /></el-icon>
    </button>
  </div>
</template>

<script setup>
import { nextTick, ref, watch } from 'vue'
import { Edit } from '@element-plus/icons-vue'

const props = defineProps({
  value: {
    type: String,
    default: '',
  },
  emptyText: {
    type: String,
    default: '点击编辑',
  },
  placeholder: {
    type: String,
    default: '',
  },
  maxlength: {
    type: Number,
    default: 200,
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

const emit = defineEmits(['save', 'cancel'])
const editing = ref(false)
const draft = ref(props.value)
const inputRef = ref(null)

watch(() => props.value, value => {
  if (!editing.value) draft.value = value
})

const startEdit = async () => {
  if (props.disabled) return
  draft.value = props.value
  editing.value = true
  await nextTick()
  inputRef.value?.focus?.()
}

const save = () => {
  if (!editing.value || props.loading) return
  editing.value = false
  if (draft.value !== props.value) emit('save', draft.value)
}

const cancel = () => {
  draft.value = props.value
  editing.value = false
  emit('cancel')
}
</script>

<style scoped lang="scss">
.inline-editable-text {
  width: 100%;
  min-width: 0;

  :deep(.el-input) {
    max-width: 100%;
  }
}

.inline-editable-text__display {
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  width: 100%;
  min-width: 0;
  padding: 3px 6px;
  color: var(--theme-text, #303133);
  text-align: left;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--border-radius-small, 6px);
  cursor: pointer;
  min-height: 28px;
  touch-action: manipulation;

  span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .el-icon {
    flex: 0 0 auto;
    color: var(--el-text-color-secondary, #909399);
    opacity: 0;
    transition: opacity 0.16s ease;
  }

  &:hover,
  &:focus-visible {
    border-color: var(--theme-border, #dcdfe6);
    background: var(--el-fill-color-lighter, #f8fafc);

    .el-icon {
      opacity: 1;
    }
  }

  &:disabled {
    color: var(--el-text-color-disabled, #a8abb2);
    cursor: not-allowed;
  }
}

@media (max-width: 768px) {
  .inline-editable-text__display {
    min-height: 44px;
    padding: 6px 8px;
  }

  .inline-editable-text__display .el-icon {
    opacity: 1;
  }
}
</style>
