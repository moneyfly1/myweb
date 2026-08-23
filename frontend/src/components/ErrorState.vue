<template>
  <div class="error-state" role="alert" aria-live="assertive">
    <div class="error-icon">
      <el-icon :size="iconSize">
        <CircleClose />
      </el-icon>
    </div>
    <div class="error-title">{{ title || '加载失败' }}</div>
    <div v-if="message" class="error-message">{{ message }}</div>
    <div class="error-actions">
      <el-button
        type="primary"
        :loading="retrying"
        @click="handleRetry"
      >
        {{ retrying ? '重试中...' : '重试' }}
      </el-button>
      <el-button v-if="showBack" @click="handleBack">
        返回
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { CircleClose } from '@element-plus/icons-vue'

const props = defineProps({
  // 错误标题
  title: {
    type: String,
    default: '加载失败'
  },
  // 错误消息
  message: {
    type: String,
    default: ''
  },
  // 图标大小
  iconSize: {
    type: Number,
    default: 80
  },
  // 是否显示返回按钮
  showBack: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['retry'])
const router = useRouter()
const retrying = ref(false)
let retryResetTimer = null

const clearRetryResetTimer = () => {
  if (retryResetTimer) {
    clearTimeout(retryResetTimer)
    retryResetTimer = null
  }
}

// 处理重试
const handleRetry = () => {
  retrying.value = true
  // Vue3 的 emit 不返回 Promise，无法 await 父组件完成。
  // 触发重试后：若父组件在 1.5s 内未通过事件复位，自动复位避免按钮卡死。
  emit('retry')
  clearRetryResetTimer()
  retryResetTimer = setTimeout(() => {
    retrying.value = false
    retryResetTimer = null
  }, 1500)
}

// 供父组件在加载完成时复位重试按钮（可选，配合 @retry 使用）
const resetRetrying = () => {
  retrying.value = false
  clearRetryResetTimer()
}

defineExpose({ resetRetrying })

// 处理返回
const handleBack = () => {
  router.back()
}

onUnmounted(() => {
  clearRetryResetTimer()
})
</script>

<style scoped>
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
  min-height: 300px;
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
  overscroll-behavior: contain;
}

.error-icon {
  margin-bottom: 20px;
  color: var(--el-color-danger, #f56c6c);
  flex: 0 0 auto;
}

.error-title {
  font-size: 18px;
  font-weight: 500;
  color: var(--el-text-color-primary, #303133);
  margin-bottom: 12px;
  max-width: min(460px, 100%);
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.error-message {
  font-size: 14px;
  color: var(--el-text-color-secondary, #909399);
  margin-bottom: 24px;
  max-width: 500px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.error-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
  flex-wrap: wrap;
  max-width: 100%;

  :deep(.el-button) {
    margin-left: 0;
    white-space: normal;
    min-height: 36px;
    touch-action: manipulation;
  }
}

@media (max-width: 768px) {
  .error-state {
    padding: 40px 16px;
    min-height: 200px;
  }

  .error-icon {
    margin-bottom: 16px;
  }

  .error-title {
    font-size: 16px;
    margin-bottom: 10px;
  }

  .error-message {
    font-size: 13px;
    margin-bottom: 20px;
  }

  .error-actions {
    flex-direction: column;
    width: 100%;
    gap: 10px;
  }

  .error-actions :deep(.el-button) {
    width: 100%;
    min-height: 44px;
  }
}

@media (max-width: 420px) {
  .error-state {
    padding: 32px 12px;
    min-height: 180px;
  }
}

</style>
