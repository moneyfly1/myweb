<template>
  <div class="responsive-data-view" :class="{ 'is-loading': loading, 'has-error': !!error }">
    <div class="responsive-data-view__table">
      <slot name="table"></slot>
    </div>
    <MobileCardList
      class="responsive-data-view__cards"
      :data="data"
      :fields="fields"
      :id-field="idField"
      :title-field="titleField"
      :loading="loading"
      :error="error"
      :empty-title="emptyTitle"
      :empty-description="emptyDescription"
      :selectable="selectable"
      :selected-rows="selectedRows"
      :is-row-selectable="isRowSelectable"
      @retry="$emit('retry')"
      @selection-change="$emit('selection-change', $event)"
    >
      <template
        v-for="(_, slotName) in forwardedSlots"
        :key="slotName"
        #[slotName]="slotProps"
      >
        <slot :name="slotName" v-bind="slotProps || {}"></slot>
      </template>
    </MobileCardList>
  </div>
</template>

<script setup>
import { computed, useSlots } from 'vue'
import MobileCardList from './MobileCardList.vue'

defineProps({
  data: {
    type: Array,
    default: () => [],
  },
  fields: {
    type: Array,
    default: () => [],
  },
  idField: {
    type: String,
    default: 'id',
  },
  titleField: {
    type: String,
    default: 'name',
  },
  loading: {
    type: Boolean,
    default: false,
  },
  error: {
    type: [Boolean, Object, String],
    default: false,
  },
  emptyTitle: {
    type: String,
    default: '暂无数据',
  },
  emptyDescription: {
    type: String,
    default: '',
  },
  // 移动端多选：透传给 MobileCardList（桌面端仍由 el-table 的 type="selection" 负责）
  selectable: {
    type: Boolean,
    default: false,
  },
  selectedRows: {
    type: Array,
    default: () => [],
  },
  isRowSelectable: {
    type: Function,
    default: null,
  },
})

defineEmits(['retry', 'selection-change'])

const slots = useSlots()
const forwardedSlots = computed(() => {
  const { table, ...mobileSlots } = slots
  return mobileSlots
})
</script>

<style scoped>
.responsive-data-view {
  width: 100%;
  min-width: 0;
}

.responsive-data-view__table {
  min-width: 0;
}

.responsive-data-view__cards {
  display: none;
  min-width: 0;
}

@media (max-width: 768px) {
  .responsive-data-view {
    display: block;
  }

  .responsive-data-view__table {
    display: none;
  }

  .responsive-data-view__cards {
    display: flex;
  }
}
</style>
