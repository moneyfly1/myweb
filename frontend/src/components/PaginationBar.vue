<template>
  <nav
    v-if="total > 0"
    class="pagination-bar"
    aria-label="分页导航"
  >
    <div class="pagination-bar__summary" aria-live="polite">
      共 {{ total }} 条
    </div>
    <el-pagination
      :current-page="currentPage"
      :page-size="pageSize"
      :page-sizes="pageSizes"
      :total="total"
      :layout="computedLayout"
      :pager-count="computedPagerCount"
      background
      @update:current-page="$emit('update:currentPage', $event)"
      @update:page-size="$emit('update:pageSize', $event)"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
    />
  </nav>
</template>

<script setup>
import { computed } from 'vue'
import { useMobile } from '@/composables/useMobile'

const props = defineProps({
  currentPage: {
    type: Number,
    default: 1,
  },
  pageSize: {
    type: Number,
    default: 10,
  },
  total: {
    type: Number,
    default: 0,
  },
  pageSizes: {
    type: Array,
    default: () => [10, 20, 50, 100],
  },
  layout: {
    type: String,
    default: 'total, sizes, prev, pager, next, jumper',
  },
  mobileLayout: {
    type: String,
    default: 'sizes, prev, pager, next, jumper',
  },
  pagerCount: {
    type: Number,
    default: 7,
  },
  mobilePagerCount: {
    type: Number,
    default: 5,
  },
})

const emit = defineEmits(['update:currentPage', 'update:pageSize', 'size-change', 'current-change', 'change'])

const isMobile = useMobile()
const computedLayout = computed(() => isMobile.value ? props.mobileLayout : props.layout)
const computedPagerCount = computed(() => isMobile.value ? props.mobilePagerCount : props.pagerCount)

const handleSizeChange = (size) => {
  emit('size-change', size)
  emit('change', { page: props.currentPage, pageSize: size })
}

const handleCurrentChange = (page) => {
  emit('current-change', page)
  emit('change', { page, pageSize: props.pageSize })
}
</script>

<style scoped lang="scss">
.pagination-bar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
}

.pagination-bar__summary {
  display: none;
  flex-shrink: 0;
  color: var(--theme-text-secondary, #909399);
  font-size: 13px;
  line-height: 1.4;
  white-space: nowrap;
}

.pagination-bar :deep(.el-pagination) {
  max-width: 100%;
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
  min-height: 36px;
  padding-bottom: 2px;
  scrollbar-width: thin;

  .el-pagination__total,
  .el-pagination__jump {
    white-space: nowrap;
  }

  .btn-prev,
  .btn-next,
  .el-pager li {
    flex-shrink: 0;
  }
}

@media (max-width: 768px) {
  .pagination-bar {
    justify-content: space-between;
    margin-top: 12px;
  }

  .pagination-bar__summary {
    display: block;
  }

  .pagination-bar :deep(.el-pagination) {
    justify-content: flex-end;
    width: auto;
    min-height: 44px;
  }

  .pagination-bar :deep(.el-pager) {
    flex: 0 1 auto;
    min-width: 0;
  }

  .pagination-bar :deep(.btn-prev),
  .pagination-bar :deep(.btn-next),
  .pagination-bar :deep(.el-pager li) {
    min-width: 44px;
    height: 44px;
    line-height: 44px;
    margin: 0 2px;
  }
}

@media (max-width: 420px) {
  .pagination-bar {
    align-items: stretch;
    flex-direction: column;
    gap: 8px;
  }

  .pagination-bar__summary {
    text-align: center;
  }

  .pagination-bar :deep(.el-pagination) {
    justify-content: center;
    width: 100%;
  }
}

@media (max-width: 360px) {
  .pagination-bar :deep(.el-pager li:not(.is-active)) {
    display: none;
  }

  .pagination-bar :deep(.el-pager li.is-active) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 44px;
  }
}
</style>
