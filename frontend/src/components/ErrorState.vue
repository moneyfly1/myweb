<template>
  <div class="error-state">
    <div class="error-icon">
      <el-icon :size="iconSize" color="#f56c6c">
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
import { ref } from 'vue'
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

// 处理重试
const handleRetry = async () => {
  retrying.value = true
  try {
    await emit('retry')
  } finally {
    // 延迟重置状态，避免闪烁
    setTimeout(() => {
      retrying.value = false
    }, 500)
  }
}

// 处理返回
const handleBack = () => {
  router.back()
}
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
}

.error-icon {
  margin-bottom: 20px;
  animation: shake 0.5s;
}

@keyframes shake {
  0%, 100% {
    transform: translateX(0);
  }
  25% {
    transform: translateX(-10px);
  }
  75% {
    transform: translateX(10px);
  }
}

.error-title {
  font-size: 18px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 12px;
}

.error-message {
  font-size: 14px;
  color: #909399;
  margin-bottom: 24px;
  max-width: 500px;
  line-height: 1.6;
  word-break: break-word;
}

.error-actions {
  display: flex;
  gap: 12px;
}

@media (max-width: 768px) {
  .error-state {
    padding: 40px 16px;
    min-height: 240px;
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
</style>
