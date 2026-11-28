<template>
  <span class="pay-method-icon" :class="kind" :title="label">
    <!-- 支付宝：蓝色方块 + 白色“支” -->
    <svg v-if="kind === 'alipay'" viewBox="0 0 48 48" aria-hidden="true">
      <rect x="2" y="2" width="44" height="44" rx="12" fill="#1677ff" />
      <text x="24" y="33" text-anchor="middle" font-size="26" font-weight="700" fill="#fff" font-family="PingFang SC, Microsoft YaHei, sans-serif">支</text>
    </svg>
    <!-- 微信：绿色方块 + 白色气泡 -->
    <svg v-else viewBox="0 0 48 48" aria-hidden="true">
      <rect x="2" y="2" width="44" height="44" rx="12" fill="#07c160" />
      <path d="M18 13c-5.5 0-10 3.7-10 8.3 0 2.6 1.4 4.9 3.6 6.4l-.9 3.1 3.6-1.8c1.2.3 2.4.5 3.7.5 5.5 0 10-3.7 10-8.3S23.5 13 18 13z" fill="#fff" />
      <path d="M31 20c-4.9 0-8.9 3.3-8.9 7.4 0 2.3 1.2 4.4 3.2 5.7l-.8 2.8 3.2-1.6c1 .3 2.1.4 3.3.4 4.9 0 8.9-3.3 8.9-7.4S35.9 20 31 20z" fill="#fff" />
      <circle cx="15" cy="20" r="1.6" fill="#07c160" />
      <circle cx="21" cy="20" r="1.6" fill="#07c160" />
      <circle cx="28.5" cy="26.5" r="1.3" fill="#07c160" />
      <circle cx="33.5" cy="26.5" r="1.3" fill="#07c160" />
    </svg>
  </span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  method: { type: String, default: '' }
})

const kind = computed(() => {
  const m = String(props.method || '').toLowerCase()
  if (m.includes('wx') || m.includes('wechat') || m.includes('weixin') || m.includes('微信')) return 'wechat'
  return 'alipay'
})
const label = computed(() => (kind.value === 'wechat' ? '微信支付' : '支付宝'))
</script>

<style scoped>
.pay-method-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  vertical-align: -4px;
  margin-right: 6px;
}
.pay-method-icon svg {
  width: 100%;
  height: 100%;
  display: block;
}
</style>
