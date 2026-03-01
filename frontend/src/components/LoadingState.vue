<template>
  <div class="loading-state" :class="{ 'loading-fullscreen': fullscreen }">
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
  color: #409eff;
  animation: rotate 1.5s linear infinite;
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
  color: #606266;
}

@media (max-width: 768px) {
  .loading-state {
    padding: 30px 16px;
    min-height: 160px;
  }

  .loading-text {
    font-size: 13px;
    margin-top: 12px;
  }
}
</style>
