<template>
  <div class="qr-code-display" :class="{ 'has-caption': $slots.default || description }">
    <img v-if="src" :src="src" :alt="alt" loading="lazy" decoding="async" />
    <el-skeleton v-else animated>
      <template #template>
        <el-skeleton-item variant="image" class="qr-code-display__skeleton" />
      </template>
    </el-skeleton>
    <div v-if="$slots.default || description" class="qr-code-display__caption">
      <slot>{{ description }}</slot>
    </div>
  </div>
</template>

<script setup>
defineProps({
  src: {
    type: String,
    default: '',
  },
  alt: {
    type: String,
    default: '二维码',
  },
  description: {
    type: String,
    default: '',
  },
})
</script>

<style scoped>
.qr-code-display {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  max-width: 100%;
  min-width: 0;
}

.qr-code-display img,
.qr-code-display__skeleton {
  width: min(180px, 100%);
  aspect-ratio: 1;
  height: auto;
  border: 1px solid var(--theme-border, #ebeef5);
  border-radius: 8px;
  box-sizing: border-box;
  background: var(--card-bg, #fff);
  object-fit: contain;
}

.qr-code-display__caption {
  color: var(--el-text-color-regular, #606266);
  font-size: 13px;
  line-height: 1.5;
  text-align: center;
  max-width: min(260px, 100%);
  overflow-wrap: anywhere;
}

@media (max-width: 768px) {
  .qr-code-display {
    width: 100%;
  }

  .qr-code-display img,
  .qr-code-display__skeleton {
    width: min(180px, 60vw);
  }
}
</style>
