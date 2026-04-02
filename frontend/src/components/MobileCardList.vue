<template>
  <div class="mobile-card-list">
    <div 
      v-for="(item, index) in data" 
      :key="item[idField] || index" 
      class="mobile-card"
    >
      <!-- 卡片头部 -->
      <div class="mobile-card-header" v-if="$slots.header || titleField">
        <slot name="header" :item="item" :index="index">
          <div class="card-title">{{ item[titleField] }}</div>
        </slot>
      </div>
      
      <!-- 卡片内容 -->
      <div class="mobile-card-body">
        <slot :item="item" :index="index">
          <div 
            v-for="field in fields" 
            :key="field.key" 
            class="card-field"
            :class="{ 'field-full': field.fullWidth }"
          >
            <span class="field-label">{{ field.label }}</span>
            <span class="field-value">
              <slot :name="`field-${field.key}`" :item="item" :value="item[field.key]">
                <template v-if="field.type === 'tag'">
                  <el-tag 
                    :type="field.tagType ? field.tagType(item[field.key]) : 'info'" 
                    size="small"
                  >
                    {{ field.formatter ? field.formatter(item[field.key], item) : item[field.key] }}
                  </el-tag>
                </template>
                <template v-else-if="field.type === 'date'">
                  {{ formatDate(item[field.key], field.format) }}
                </template>
                <template v-else-if="field.type === 'money'">
                  ¥{{ formatMoney(item[field.key]) }}
                </template>
                <template v-else>
                  {{ field.formatter ? field.formatter(item[field.key], item) : item[field.key] }}
                </template>
              </slot>
            </span>
          </div>
        </slot>
      </div>
      
      <!-- 卡片操作按钮 -->
      <div class="mobile-card-actions" v-if="$slots.actions">
        <slot name="actions" :item="item" :index="index"></slot>
      </div>
    </div>
    
    <!-- 空状态 -->
    <div v-if="data.length === 0" class="mobile-card-empty">
      <slot name="empty">
        <el-empty description="暂无数据" />
      </slot>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import dayjs from 'dayjs'

const props = defineProps({
  // 数据列表
  data: {
    type: Array,
    required: true,
    default: () => []
  },
  // ID字段名
  idField: {
    type: String,
    default: 'id'
  },
  // 标题字段名
  titleField: {
    type: String,
    default: 'name'
  },
  // 字段配置
  fields: {
    type: Array,
    default: () => []
  },
  // 日期格式
  dateFormat: {
    type: String,
    default: 'YYYY-MM-DD HH:mm:ss'
  }
})

const formatDate = (date, format) => {
  if (!date) return '-'
  return dayjs(date).format(format || props.dateFormat)
}

const formatMoney = (value) => {
  if (value === null || value === undefined) return '0.00'
  return Number(value).toFixed(2)
}
</script>

<style scoped lang="scss">
.mobile-card-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 4px;
}

.mobile-card {
  background: var(--card-bg, #fff);
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  overflow: hidden;
  transition: all 0.3s ease;
  
  &:active {
    transform: scale(0.98);
  }
}

.mobile-card-header {
  padding: 12px 16px;
  background: linear-gradient(135deg, var(--theme-primary, #409EFF) 0%, lighten(#409EFF, 10%) 100%);
  color: white;
  
  .card-title {
    font-size: 15px;
    font-weight: 600;
    line-height: 1.4;
  }
}

.mobile-card-body {
  padding: 12px 16px;
}

.card-field {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 8px 0;
  border-bottom: 1px solid var(--theme-border, #f0f0f0);
  
  &:last-child {
    border-bottom: none;
  }
  
  &.field-full {
    flex-direction: column;
    gap: 4px;
  }
}

.field-label {
  font-size: 13px;
  color: #909399;
  flex-shrink: 0;
  margin-right: 12px;
}

.field-value {
  font-size: 14px;
  color: var(--theme-text, #303133);
  text-align: right;
  flex: 1;
  word-break: break-all;
}

.mobile-card-actions {
  padding: 12px 16px;
  border-top: 1px solid var(--theme-border, #f0f0f0);
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  
  :deep(.el-button) {
    flex: 1;
    min-width: 0;
  }
}

.mobile-card-empty {
  padding: 40px 0;
}
</style>
