<template>
  <div class="skeleton-page" role="status" aria-label="加载中">
    <!-- 顶部标题骨架 -->
    <div class="skeleton-title" v-if="showTitle" />
    <!-- 统计卡片行 -->
    <div v-if="variant === 'dashboard'" class="skeleton-grid">
      <div v-for="i in 4" :key="i" class="skeleton-card">
        <div class="skeleton-icon" />
        <div class="skeleton-lines">
          <div class="skeleton-line w-60" />
          <div class="skeleton-line w-40" />
        </div>
      </div>
    </div>
    <!-- 列表行骨架 -->
    <div v-if="variant === 'list'" class="skeleton-list">
      <div v-for="i in rows" :key="i" class="skeleton-row">
        <div class="skeleton-line w-80" />
        <div class="skeleton-line w-50" />
        <div class="skeleton-line w-30" />
      </div>
    </div>
    <!-- 通用卡片骨架 -->
    <div v-else-if="variant === 'card'" class="skeleton-cards">
      <div v-for="i in rows" :key="i" class="skeleton-card">
        <div class="skeleton-line w-60" />
        <div class="skeleton-line w-90" />
        <div class="skeleton-line w-70" />
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  // dashboard: 统计卡布局 / list: 列表行 / card: 卡片网格
  variant: {
    type: String,
    default: 'dashboard'
  },
  rows: {
    type: Number,
    default: 3
  },
  showTitle: {
    type: Boolean,
    default: true
  }
})
</script>

<style scoped>
.skeleton-page {
  width: 100%;
  min-width: 0;
  padding: 4px 0;
}

/* 基础骨骼块 */
.skeleton-title,
.skeleton-line,
.skeleton-icon,
.skeleton-card,
.skeleton-row {
  background: linear-gradient(90deg, #f0f2f5 25%, #e6e8eb 37%, #f0f2f5 63%);
  background-size: 400% 100%;
  animation: skeleton-loading 1.4s ease infinite;
  border-radius: 6px;
}

@keyframes skeleton-loading {
  0% { background-position: 100% 50%; }
  100% { background-position: 0 50%; }
}

.skeleton-title {
  height: 28px;
  width: 40%;
  margin-bottom: 20px;
}

/* 仪表盘统计卡 */
.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}
.skeleton-card {
  padding: 20px;
  display: flex;
  gap: 14px;
  align-items: flex-start;
  min-height: 100px;
}
.skeleton-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  flex-shrink: 0;
}
.skeleton-lines {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding-top: 4px;
}
.skeleton-line {
  height: 14px;
}
.w-30 { width: 30%; }
.w-40 { width: 40%; }
.w-50 { width: 50%; }
.w-60 { width: 60%; }
.w-70 { width: 70%; }
.w-80 { width: 80%; }
.w-90 { width: 90%; }

/* 列表骨架 */
.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.skeleton-row {
  padding: 16px 20px;
  display: flex;
  gap: 12px;
  align-items: center;
  min-height: 56px;
}
.skeleton-row .skeleton-line:first-child { flex: 2; }
.skeleton-row .skeleton-line:nth-child(2) { flex: 1.2; }
.skeleton-row .skeleton-line:nth-child(3) { flex: 0.8; }

/* 卡片网格（套餐等） */
.skeleton-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}
.skeleton-cards .skeleton-card {
  flex-direction: column;
  min-height: 160px;
}

@media (max-width: 992px) {
  .skeleton-grid { grid-template-columns: repeat(2, 1fr); }
  .skeleton-cards { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 768px) {
  .skeleton-grid { grid-template-columns: 1fr; gap: 12px; }
  .skeleton-cards { grid-template-columns: 1fr; gap: 12px; }
  .skeleton-title { width: 60%; }
  .skeleton-card { min-height: 84px; padding: 16px; }
}

@media (prefers-reduced-motion: reduce) {
  .skeleton-title,
  .skeleton-line,
  .skeleton-icon,
  .skeleton-card,
  .skeleton-row {
    animation: none;
    background: #f0f2f5;
  }
}
</style>
