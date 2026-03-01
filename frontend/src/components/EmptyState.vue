<template>
  <div class="empty-state">
    <div class="empty-icon">
      <el-icon :size="iconSize">
        <component :is="iconComponent" />
      </el-icon>
    </div>
    <div class="empty-title">{{ title }}</div>
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
import { computed } from 'vue'
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
  return props.actionText || !!emit.action
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
  padding: 60px 20px;
  text-align: center;
  min-height: 300px;
}

.empty-icon {
  margin-bottom: 20px;
  color: #c0c4cc;
}

.empty-icon :deep(.el-icon) {
  transition: transform 0.3s;
}

.empty-state:hover .empty-icon :deep(.el-icon) {
  transform: scale(1.1);
}

.empty-title {
  font-size: 16px;
  font-weight: 500;
  color: #606266;
  margin-bottom: 8px;
}

.empty-description {
  font-size: 14px;
  color: #909399;
  margin-bottom: 20px;
  max-width: 400px;
  line-height: 1.6;
}

.empty-action {
  margin-top: 12px;
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
    padding: 40px 16px;
    min-height: 240px;
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
}
</style>
