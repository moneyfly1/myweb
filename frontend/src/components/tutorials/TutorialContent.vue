<template>
  <div class="tutorial-content">
    <template v-if="state.status === 'ready'">
      <div v-if="state.summary" class="tutorial-summary">{{ state.summary }}</div>
      <div class="tutorial-html" v-html="sanitizedContent"></div>
    </template>

    <div v-else-if="state.status === 'loading'" class="tutorial-placeholder">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>正在加载教程…</span>
    </div>

    <div v-else class="tutorial-placeholder tutorial-missing">
      <el-icon><InfoFilled /></el-icon>
      <span>该客户端的教程正在整理中。如需帮助，可在「工单中心」联系我们。</span>
    </div>
  </div>
</template>

<script setup>
/**
 * TutorialContent —— 知识库教程正文渲染
 *
 * 教程正文统一来自知识库（后台可维护），经 sanitize 后渲染，
 * 避免在多个页面重复写 v-html 与降级逻辑。
 */
import { computed } from 'vue'
import { InfoFilled, Loading } from '@element-plus/icons-vue'
import { sanitizeArticleHtml } from '@/utils/sanitizeHtml'

const props = defineProps({
  // state: { status: 'loading' | 'ready' | 'missing', summary, content }
  state: {
    type: Object,
    default: () => ({ status: 'loading' }),
  },
})

const sanitizedContent = computed(() =>
  props.state?.status === 'ready' ? sanitizeArticleHtml(props.state.content || '') : ''
)
</script>

<style scoped>
.tutorial-summary {
  margin-bottom: 10px;
  padding: 8px 12px;
  border-left: 3px solid var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-radius: 4px;
  font-size: 13px;
  color: var(--el-text-color-regular);
}
.tutorial-html :deep(h2) {
  margin: 16px 0 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.tutorial-html :deep(h2:first-child) {
  margin-top: 0;
}
.tutorial-html :deep(h3) {
  margin: 12px 0 6px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.tutorial-html :deep(p) {
  margin: 6px 0;
  font-size: 13.5px;
  line-height: 1.85;
  color: var(--el-text-color-regular);
}
.tutorial-html :deep(ol),
.tutorial-html :deep(ul) {
  margin: 6px 0;
  padding-left: 20px;
  font-size: 13.5px;
  line-height: 1.9;
  color: var(--el-text-color-regular);
}
.tutorial-html :deep(li) {
  margin-bottom: 3px;
}
.tutorial-html :deep(code) {
  padding: 1px 5px;
  border-radius: 3px;
  background: var(--el-fill-color-light);
  font-size: 12.5px;
}
.tutorial-placeholder {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.tutorial-missing {
  color: var(--el-text-color-placeholder);
}
</style>
