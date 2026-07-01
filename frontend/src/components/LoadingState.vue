<template>
  <div
    class="loading-state"
    :class="{ 'loading-fullscreen': fullscreen }"
    role="status"
    aria-live="polite"
  >
    <el-icon class="loading-icon" :size="size">
      <Loading />
    </el-icon>
    <div v-if="text" class="loading-text">{{ text }}</div>
  </div>
</template>

<script setup>
import { Loading } from '@element-plus/icons-vue'

defineProps({
  // 加载文字
  text: {
    type: String,
    default: '加载中...'
  },
  // 图标大小
  size: {
    type: Number,
    default: 40
  },
  // 是否全屏
  fullscreen: {
    type: Boolean,
    default: false
  }
})
</script>

<style scoped>
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  min-height: 200px;
  box-sizing: border-box;
  text-align: center;
  width: 100%;
  min-width: 0;
  overscroll-behavior: contain;
}

.loading-fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 255, 255, 0.9);
  z-index: 9999;
  min-height: 100vh;
}

.loading-icon {
  color: var(--el-color-primary, #409eff);
  animation: rotate 1.5s linear infinite;
  flex: 0 0 auto;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.loading-text {
  margin-top: 16px;
  font-size: 14px;
  color: var(--el-text-color-regular, #606266);
  line-height: 1.5;
  max-width: min(420px, 100%);
  overflow-wrap: anywhere;
}

@media (max-width: 768px) {
  .loading-state {
    padding: 30px 16px;
    min-height: 140px;
  }

  .loading-text {
    font-size: 13px;
    margin-top: 12px;
  }
}

@media (max-width: 420px) {
  .loading-state {
    padding: 24px 12px;
    min-height: 120px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .loading-icon {
    animation: none;
  }
}
</style>
