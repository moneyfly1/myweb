<template>
  <div class="empty-state" :data-type="type" role="status" aria-live="polite">
    <div class="empty-icon">
      <el-icon :size="iconSize">
        <component :is="iconComponent" />
      </el-icon>
    </div>
    <div class="empty-title">{{ defaultTitle }}</div>
    <div v-if="description" class="empty-description">{{ description }}</div>
    <div v-if="showAction" class="empty-action">
      <el-button
        v-if="actionText"
        :type="actionType"
        :loading="loading"
        @click="handleAction"
      >
        {{ actionText }}
      </el-button>
      <slot name="action"></slot>
    </div>
  </div>
</template>

<script setup>
import { computed, useSlots } from 'vue'
import { Document, Warning, CircleClose, InfoFilled } from '@element-plus/icons-vue'

const props = defineProps({
  // 状态类型：empty(空数据), error(错误), loading(加载中), noPermission(无权限)
  type: {
    type: String,
    default: 'empty',
    validator: (value) => ['empty', 'error', 'loading', 'noPermission'].includes(value)
  },
  // 标题
  title: {
    type: String,
    default: ''
  },
  // 描述
  description: {
    type: String,
    default: ''
  },
  // 操作按钮文字
  actionText: {
    type: String,
    default: ''
  },
  // 操作按钮类型
  actionType: {
    type: String,
    default: 'primary'
  },
  // 图标大小
  iconSize: {
    type: Number,
    default: 80
  },
  // 加载状态
  loading: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['action'])
const slots = useSlots()

// 根据类型选择图标
const iconComponent = computed(() => {
  const iconMap = {
    empty: Document,
    error: CircleClose,
    loading: InfoFilled,
    noPermission: Warning
  }
  return iconMap[props.type] || Document
})

// 是否显示操作区域
const showAction = computed(() => {
  return props.actionText || !!slots.action
})

// 处理操作按钮点击
const handleAction = () => {
  emit('action')
}

// 默认标题
const defaultTitle = computed(() => {
  if (props.title) return props.title
  const titleMap = {
    empty: '暂无数据',
    error: '加载失败',
    loading: '加载中...',
    noPermission: '无权限访问'
  }
  return titleMap[props.type] || '暂无数据'
})
</script>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 20px;
  text-align: center;
  min-height: 260px;
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
  overscroll-behavior: contain;
}

.empty-icon {
  margin-bottom: 16px;
  color: var(--el-text-color-placeholder, #c0c4cc);
  flex: 0 0 auto;
}

.empty-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-regular, #606266);
  margin-bottom: 8px;
  line-height: 1.4;
  max-width: min(420px, 100%);
  overflow-wrap: anywhere;
}

.empty-description {
  font-size: 14px;
  color: var(--el-text-color-secondary, #909399);
  margin-bottom: 20px;
  max-width: 400px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.empty-action {
  margin-top: 12px;
  display: flex;
  justify-content: center;
  gap: 8px;
  flex-wrap: wrap;
  max-width: 100%;

  :deep(.el-button) {
    margin-left: 0;
    white-space: normal;
    min-height: 36px;
    touch-action: manipulation;
  }
}

/* 不同类型的颜色 */
.empty-state[data-type="error"] .empty-icon {
  color: #f56c6c;
}

.empty-state[data-type="loading"] .empty-icon {
  color: #409eff;
}

.empty-state[data-type="noPermission"] .empty-icon {
  color: #e6a23c;
}

@media (max-width: 768px) {
  .empty-state {
    padding: 32px 16px;
    min-height: 180px;
  }

  .empty-icon {
    margin-bottom: 16px;
  }

  .empty-title {
    font-size: 15px;
  }

  .empty-description {
    font-size: 13px;
    margin-bottom: 16px;
  }

  .empty-action {
    width: 100%;
  }

  .empty-action :deep(.el-button) {
    flex: 1;
    min-height: 44px;
    min-width: 0;
  }
}

@media (max-width: 420px) {
  .empty-state {
    padding: 28px 12px;
    min-height: 160px;
  }
}

</style>
