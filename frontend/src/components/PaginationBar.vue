<template>
  <nav
    v-if="total > 0"
    class="pagination-bar"
    :class="{ 'pagination-bar--mobile': isMobile }"
    aria-label="分页导航"
  >
    <!--
      移动端：由本组件自己渲染「共 N 条 + 每页条数」。
      为什么不复用 Element 内部的 sizes/jump 插槽：
      它们是 el-pagination 的 flex 子项，和页码挤在同一条 flex 流里，
      在窄屏上必然出现「条数选择居中、页码靠右、跳页又掉到下一行」的错乱。
      拆成独立一行后：第一行=统计+每页条数（居中），第二行=页码（居中），
      上下两行各自独立、左右都不再互相拉扯。
      桌面端不渲染这一行（改用 Element 的 total/sizes，保持原样）。
    -->
    <div v-if="isMobile" class="pagination-bar__meta">
      <span class="pagination-bar__summary" aria-live="polite">共 {{ total }} 条</span>
      <el-select
        v-model="sizeProxy"
        class="pagination-bar__size"
        size="small"
        aria-label="每页显示条数"
      >
        <el-option
          v-for="s in pageSizes"
          :key="s"
          :label="`${s} 条/页`"
          :value="s"
        />
      </el-select>
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
  // 移动端只保留「上一页 / 页码 / 下一页」一行：
  // total 与 sizes 已由上面的 __meta 行承担，jumper 在手机上输入体验差且会多占一行。
  mobileLayout: {
    type: String,
    default: 'prev, pager, next',
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

// 移动端「每页条数」下拉：与父组件的 v-model:page-size / @size-change 完全兼容
const sizeProxy = computed({
  get: () => props.pageSize,
  set: (size) => {
    if (size === props.pageSize) return
    emit('update:pageSize', size)
    handleSizeChange(size)
  },
})

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

/* ===== 移动端分页：两行、各自居中、互不干扰 =====
   移动端唯一开关是 nav 上的 .pagination-bar--mobile（由 useMobile(768) 控制），
   不再用媒体查询，避免 CSS 与 JS 两套断点；同时多一个类也保证
   页面级 `.xxx :deep(.el-pagination)` 覆盖（如 flex-wrap:wrap）压不过这里。 */
.pagination-bar--mobile {
  /* 每页条数下拉的宽度集中在这里，断点只改变量 */
  --pb-size-w: 118px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  margin-top: 12px;
}

.pagination-bar--mobile .pagination-bar__meta {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: 100%;
  min-width: 0;
}

.pagination-bar--mobile .pagination-bar__summary {
  flex-shrink: 0;
  color: var(--theme-text-secondary, #909399);
  font-size: 12px;
  line-height: 1.4;
  white-space: nowrap;
}

.pagination-bar--mobile .pagination-bar__size {
  flex: 0 0 auto;
  /* 必须 !important：实测 /login-history、/tickets 等页面存在
     `.el-select { width: 100% !important }`（表单全宽规则）会把本组件自带的
     每页条数下拉撑成整行，导致 meta 行比容器还宽、左侧"共 N 条"被挤出屏幕。 */
  width: var(--pb-size-w) !important;
  max-width: var(--pb-size-w) !important;
  min-width: 0 !important;
}

/* 第二行：页码。nowrap + 居中，杜绝"页码靠右 / 跳页独占一行"的错位 */
.pagination-bar--mobile :deep(.el-pagination) {
  display: flex;
  /* !important：SystemLogs 等页面级样式写了 `.el-pagination{flex-wrap:wrap}`，
     实测会把「… 最后一页」拆到第二行，形成上下错乱。 */
  flex-wrap: nowrap !important;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 0;
  margin: 0;
  padding: 0;
  gap: 6px;
  overflow: visible;

  /* Element 内置的 total / sizes / jump 在移动端不参与布局（见 mobileLayout） */
  .el-pagination__total,
  .el-pagination__sizes,
  .el-pagination__jump {
    display: none;
  }
}

.pagination-bar--mobile :deep(.btn-prev),
.pagination-bar--mobile :deep(.btn-next),
.pagination-bar--mobile :deep(.el-pager li) {
  flex-shrink: 0;
  box-sizing: border-box;
  min-width: 36px;
  width: 36px;
  height: 36px;
  /* 必须显式压掉全局移动端规则里的 min-height:40px，
     否则 prev/next 会比页码高 8px，一行里出现两种高度（看着就是"错乱"）。 */
  min-height: 36px;
  line-height: 34px;
  margin: 0;
  padding: 0;
  font-size: 13px;
  border-radius: 8px;
}

.pagination-bar--mobile :deep(.el-pager) {
  flex: 0 1 auto;
  min-width: 0;
  display: flex;
  /* 关键：全局样式里有 .el-pager{flex-wrap:wrap}，
     实测在 320px 会把「… 153」拆到第二行、形成 3 行错位。
     这里强制单行；万一真放不下则横向滚动，也绝不换行。 */
  flex-wrap: nowrap !important;
  align-items: center;
  justify-content: center;
  gap: 5px;
  overflow-x: auto;
  overflow-y: hidden;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;

  &::-webkit-scrollbar {
    display: none;
  }

  li {
    margin: 0;
  }
}

/* 超窄屏（≤340px）：只收紧尺寸、结构不变，避免再次错行 */
@media (max-width: 360px) {
  .pagination-bar--mobile {
    --pb-size-w: 106px;
  }

  .pagination-bar--mobile :deep(.el-pager) {
    gap: 4px;
  }

  .pagination-bar--mobile :deep(.btn-prev),
  .pagination-bar--mobile :deep(.btn-next),
  .pagination-bar--mobile :deep(.el-pager li) {
    min-width: 32px;
    width: 32px;
    height: 32px;
    min-height: 32px;
    line-height: 30px;
  }
}

/* 超窄屏（≤330px，如 iPhone SE 320px）：再收紧一档，仍保持单行 */
@media (max-width: 330px) {
  .pagination-bar--mobile :deep(.el-pager) {
    gap: 3px;
  }

  .pagination-bar--mobile :deep(.btn-prev),
  .pagination-bar--mobile :deep(.btn-next),
  .pagination-bar--mobile :deep(.el-pager li) {
    min-width: 29px;
    width: 29px;
    height: 30px;
    min-height: 30px;
    line-height: 28px;
    font-size: 12px;
  }
}
</style>
