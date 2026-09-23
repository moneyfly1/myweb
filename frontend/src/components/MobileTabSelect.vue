<!--
  MobileTabSelect —— 移动端用下拉选择代替横向滚动的 tab 条。

  为什么需要：一屏放不下的 tab 条会变成横向滚动条，实测
  「系统设置」12 个 tab ≈ 1098px、「日志管理」7 个 tab ≈ 654px，390px 屏幕上一屏
  只露 2~3 个，用户得反复横滑才能找到目标页；截图里也常被当成「页面溢出」。

  用法（桌面端不受影响）：
    <el-tabs v-model="activeTab" class="hide-tabs-mobile">...</el-tabs>
    <MobileTabSelect v-model="activeTab" :tabs="myTabs" />
  其中 `hide-tabs-mobile` 是 global.scss 里的工具类，移动端隐藏 tab 条本身。

  注意：tabs 列表必须与 el-tab-pane 的 name/label 一一对应，
  否则会出现「切不过去 / 切错页」——建议在各页 onMounted 里做一次一致性自检。
-->
<template>
  <div v-if="isMobile" class="mobile-tab-select">
    <el-select
      :model-value="modelValue"
      class="mobile-tab-select__inner"
      size="default"
      @update:model-value="$emit('update:modelValue', $event)"
    >
      <el-option v-for="t in tabs" :key="t.name" :label="t.label" :value="t.name" />
    </el-select>
  </div>
</template>

<script setup>
import { useMobile } from '@/composables/useMobile'

defineProps({
  modelValue: { type: String, default: '' },
  // [{ name: 'general', label: '基本设置' }, ...]
  tabs: { type: Array, default: () => [] }
})
defineEmits(['update:modelValue'])

const isMobile = useMobile()
</script>

<style scoped>
.mobile-tab-select {
  margin-bottom: 10px;
}
.mobile-tab-select__inner {
  width: 100%;
}
</style>
