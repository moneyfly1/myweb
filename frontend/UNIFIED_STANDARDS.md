# 前端代码统一规范指南

本文档定义了前端项目的统一规范，所有新代码和代码修改都应遵循这些规范。

## 📋 目录

- [移动端检测](#移动端检测)
- [弹窗样式](#弹窗样式)
- [API 响应处理](#api-响应处理)
- [错误处理](#错误处理)
- [表单验证](#表单验证)
- [按钮样式](#按钮样式)
- [图标使用](#图标使用)
- [分页配置](#分页配置)
- [移动端卡片列表](#移动端卡片列表)

---

## 移动端检测

### 使用方式

使用统一的 `useMobile` composable：

```vue
<script setup>
import { useMobile } from '@/composables/useMobile'

const isMobile = useMobile() // 默认断点 768px
// 或自定义断点
const isMobile = useMobile(992)
</script>

<template>
  <div v-if="isMobile">移动端内容</div>
  <div v-else>桌面端内容</div>
</template>
```

### 注意事项

- ✅ 所有需要响应式的组件都应使用此 composable
- ✅ 默认断点为 768px
- ❌ 不要在各组件中重复定义 `checkMobile` 函数

---

## 弹窗样式

### 弹窗尺寸规范

| 类名 | 宽度 | 使用场景 |
|------|------|----------|
| `.dialog-xs` | 320px | 确认提示、简单信息 |
| `.dialog-sm` | 480px | 简单表单、编辑 |
| `.dialog-md` | 640px | 一般表单、详情 |
| `.dialog-lg` | 800px | 复杂表单、多列布局 |
| `.dialog-xl` | 1000px | 表格、复杂内容 |

### 使用示例

```vue
<template>
  <el-dialog
    v-model="dialogVisible"
    title="编辑用户"
    class="dialog-md"
  >
    <!-- 弹窗内容 -->
  </el-dialog>
</template>
```

### 弹窗底部按钮

```vue
<template>
  <el-dialog v-model="dialogVisible">
    <!-- 内容 -->
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </div>
    </template>
  </el-dialog>
</template>
```

### 注意事项

- ✅ 使用预定义的尺寸类
- ✅ 移动端自动适配为 92% 宽度
- ❌ 不要在组件中重复定义 `:deep(.el-dialog)` 样式

---

## API 响应处理

### 统一响应数据结构

所有 API 响应都应使用 `utils/apiResponse.js` 中的工具函数处理：

```javascript
import { extractList, extractPagination } from '@/utils/apiResponse'

// 获取列表数据
const loadData = async () => {
  try {
    const response = await api.get('/users')
    
    // 提取列表数据
    list.value = extractList(response)
    
    // 提取分页信息
    const pagination = extractPagination(response)
    total.value = pagination.total
  } catch (error) {
    // 错误处理
  }
}
```

### 支持的列表字段

工具函数会自动识别以下字段：
- `list`
- `logs`
- `records`
- `items`
- `data`

### 分页配置

```javascript
import { PAGINATION_CONFIG, getPaginationLayout } from '@/utils/apiResponse'

// 使用统一的分页配置
const pageSizes = PAGINATION_CONFIG.pageSizes // [10, 20, 50, 100]

// 响应式分页布局
const paginationLayout = computed(() => getPaginationLayout(isMobile.value))
```

---

## 错误处理

### 统一错误提示

```javascript
import { showError, showSuccess, showConfirm } from '@/utils/errorHandler'

// 显示错误
try {
  await api.post('/users', data)
  showSuccess('操作成功')
} catch (error) {
  showError(error) // 自动提取和格式化错误消息
}

// 确认操作
const confirmed = await showConfirm('确定要删除吗？', '删除确认', {
  type: 'warning',
  confirmText: '删除',
  cancelText: '取消'
})

if (confirmed) {
  // 执行删除
}
```

### 带 Loading 的异步操作

```javascript
import { withLoading } from '@/utils/errorHandler'

const loading = ref(false)

const handleSubmit = async () => {
  await withLoading(
    async () => {
      // 异步操作
      await api.post('/users', formData)
    },
    {
      loadingRef: loading,
      successMessage: '保存成功',
      errorMessage: '保存失败'
    }
  )
}
```

---

## 表单验证

### 使用预定义验证规则

```javascript
import { VALIDATION_RULES, getFormLayout } from '@/utils/formValidation'

// 表单验证规则
const formRules = {
  username: VALIDATION_RULES.username,
  email: VALIDATION_RULES.email,
  phone: VALIDATION_RULES.phone,
  password: VALIDATION_RULES.password,
  amount: VALIDATION_RULES.amount
}

// 响应式表单布局
const formLayout = computed(() => getFormLayout(isMobile.value))
```

### 表单模板

```vue
<template>
  <el-form
    :model="form"
    :rules="formRules"
    v-bind="formLayout"
    ref="formRef"
  >
    <el-form-item label="用户名" prop="username">
      <el-input v-model="form.username" />
    </el-form-item>
    
    <el-form-item label="邮箱" prop="email">
      <el-input v-model="form.email" />
    </el-form-item>
    
    <div class="form-actions">
      <el-button @click="resetForm">重置</el-button>
      <el-button type="primary" @click="submitForm">提交</el-button>
    </div>
  </el-form>
</template>
```

### 常用验证规则

| 规则 | 说明 |
|------|------|
| `VALIDATION_RULES.required` | 必填 |
| `VALIDATION_RULES.email` | 邮箱 |
| `VALIDATION_RULES.phone` | 手机号 |
| `VALIDATION_RULES.password` | 密码（最少6位） |
| `VALIDATION_RULES.username` | 用户名（3-20字符） |
| `VALIDATION_RULES.amount` | 金额 |
| `VALIDATION_RULES.verificationCode` | 验证码（6位数字） |

---

## 按钮样式

### 按钮组布局

```vue
<template>
  <!-- 操作按钮组 -->
  <div class="action-button-group">
    <el-button type="primary" size="small" :icon="IconEdit">编辑</el-button>
    <el-button type="danger" size="small" :icon="IconDelete">删除</el-button>
  </div>
  
  <!-- 页面头部操作 -->
  <div class="page-header-actions">
    <el-button type="primary" :icon="IconPlus">添加</el-button>
    <el-button :icon="IconRefresh">刷新</el-button>
  </div>
  
  <!-- 工具栏 -->
  <div class="toolbar-buttons">
    <el-button type="success">批量通过</el-button>
    <el-button type="warning">批量拒绝</el-button>
  </div>
</template>
```

### 表单操作按钮

```vue
<template>
  <div class="form-actions">
    <el-button @click="handleCancel">取消</el-button>
    <el-button type="primary" @click="handleSubmit">确定</el-button>
  </div>
</template>
```

### 注意事项

- ✅ 使用预定义的按钮组类
- ✅ 移动端自动适配为全宽垂直布局
- ✅ 按钮最小点击区域 36px（移动端友好）
- ❌ 不要使用内联样式定义按钮布局

---

## 图标使用

### 统一导入方式

```vue
<script setup>
import { 
  IconPlus, 
  IconEdit, 
  IconDelete,
  IconSearch,
  IconRefresh 
} from '@/utils/icons'
</script>

<template>
  <el-button :icon="IconPlus">添加</el-button>
  <el-button :icon="IconEdit">编辑</el-button>
  <el-button :icon="IconDelete">删除</el-button>
</template>
```

### 注意事项

- ✅ 统一从 `@/utils/icons` 导入
- ✅ 使用 `Icon` 前缀避免命名冲突
- ✅ 使用 Element Plus 图标组件
- ❌ 不要使用 Font Awesome 或旧版 `el-icon-*` 类名
- ❌ 不要在各个组件中重复导入图标

---

## 分页配置

### 统一分页组件

```vue
<template>
  <el-pagination
    v-model:current-page="pagination.page"
    v-model:page-size="pagination.size"
    :page-sizes="PAGINATION_CONFIG.pageSizes"
    :layout="paginationLayout"
    :total="pagination.total"
    @size-change="handleSizeChange"
    @current-change="handlePageChange"
  />
</template>

<script setup>
import { computed } from 'vue'
import { PAGINATION_CONFIG, getPaginationLayout } from '@/utils/apiResponse'
import { useMobile } from '@/composables/useMobile'

const isMobile = useMobile()
const paginationLayout = computed(() => getPaginationLayout(isMobile.value))
</script>
```

### 分页配置常量

```javascript
PAGINATION_CONFIG = {
  pageSizes: [10, 20, 50, 100],
  layout: 'total, sizes, prev, pager, next, jumper',
  mobileLayout: 'total, prev, pager, next',
  defaultPageSize: 20,
  defaultPage: 1
}
```

---

## 移动端卡片列表

### 使用 MobileCardList 组件

```vue
<template>
  <!-- 桌面端表格 -->
  <el-table v-if="!isMobile" :data="list">
    <el-table-column prop="name" label="名称" />
    <el-table-column prop="status" label="状态" />
    <el-table-column label="操作">
      <template #default="{ row }">
        <el-button size="small" @click="editItem(row)">编辑</el-button>
      </template>
    </el-table-column>
  </el-table>
  
  <!-- 移动端卡片列表 -->
  <MobileCardList
    v-else
    :data="list"
    title-field="name"
    :fields="[
      { key: 'status', label: '状态', type: 'tag' },
      { key: 'created_at', label: '创建时间', type: 'date' }
    ]"
  >
    <template #actions="{ item }">
      <el-button size="small" @click="editItem(item)">编辑</el-button>
      <el-button size="small" type="danger" @click="deleteItem(item)">删除</el-button>
    </template>
  </MobileCardList>
</template>

<script setup>
import MobileCardList from '@/components/MobileCardList.vue'
import { useMobile } from '@/composables/useMobile'

const isMobile = useMobile()
</script>
```

### 字段类型支持

| 类型 | 说明 | 示例 |
|------|------|------|
| `text` | 普通文本（默认） | `{ key: 'name', label: '名称' }` |
| `tag` | 标签显示 | `{ key: 'status', label: '状态', type: 'tag' }` |
| `date` | 日期格式化 | `{ key: 'created_at', label: '时间', type: 'date' }` |
| `money` | 金额格式化 | `{ key: 'price', label: '价格', type: 'money' }` |

### 自定义字段渲染

```vue
<MobileCardList :data="list" title-field="username">
  <template #field-status="{ item, value }">
    <el-tag :type="value === 1 ? 'success' : 'danger'">
      {{ value === 1 ? '启用' : '禁用' }}
    </el-tag>
  </template>
</MobileCardList>
```

---

## 🎯 快速检查清单

在提交代码前，请确认：

- [ ] 使用 `useMobile()` 而非自定义的移动端检测
- [ ] 弹窗使用预定义的尺寸类（`.dialog-md` 等）
- [ ] 使用 `extractList()` 和 `extractPagination()` 处理响应
- [ ] 使用 `showError()` 和 `showSuccess()` 显示消息
- [ ] 表单使用 `VALIDATION_RULES` 中的验证规则
- [ ] 按钮使用预定义的按钮组类
- [ ] 图标从 `@/utils/icons` 统一导入
- [ ] 分页使用 `PAGINATION_CONFIG` 配置
- [ ] 移动端使用 `MobileCardList` 组件

---

## 📝 迁移指南

### 从旧代码迁移

1. **替换移动端检测**
   ```javascript
   // 旧代码
   const isMobile = ref(false)
   const checkMobile = () => { isMobile.value = window.innerWidth <= 768 }
   onMounted(() => { checkMobile(); window.addEventListener('resize', checkMobile) })
   onUnmounted(() => { window.removeEventListener('resize', checkMobile) })
   
   // 新代码
   import { useMobile } from '@/composables/useMobile'
   const isMobile = useMobile()
   ```

2. **替换响应处理**
   ```javascript
   // 旧代码
   list.value = response.data.data.list
   total.value = response.data.data.total
   
   // 新代码
   list.value = extractList(response)
   const pagination = extractPagination(response)
   total.value = pagination.total
   ```

3. **替换错误处理**
   ```javascript
   // 旧代码
   catch (error) {
     ElMessage.error(error.response?.data?.message || error.message)
   }
   
   // 新代码
   catch (error) {
     showError(error)
   }
   ```

4. **替换弹窗样式**
   ```vue
   <!-- 旧代码 -->
   <el-dialog v-model="visible" width="600px">
   
   <!-- 新代码 -->
   <el-dialog v-model="visible" class="dialog-md">
   ```

---

## 🔧 工具文件位置

| 文件 | 路径 | 说明 |
|------|------|------|
| 移动端检测 | `@/composables/useMobile.js` | 统一的移动端检测 |
| 弹窗样式 | `@/styles/dialog-common.scss` | 公共弹窗样式 |
| 按钮样式 | `@/styles/button-common.scss` | 公共按钮样式 |
| 列表样式 | `@/styles/list-common.scss` | 公共列表样式 |
| API 响应处理 | `@/utils/apiResponse.js` | 响应数据提取 |
| 错误处理 | `@/utils/errorHandler.js` | 统一错误提示 |
| 表单验证 | `@/utils/formValidation.js` | 验证规则 |
| 图标导入 | `@/utils/icons.js` | 统一图标导入 |
| 移动端卡片 | `@/components/MobileCardList.vue` | 移动端卡片组件 |

---

## ❓ 常见问题

### Q: 为什么要统一这些规范？

A: 统一规范可以：
- 减少代码重复
- 提高可维护性
- 保证 UI 一致性
- 降低学习成本
- 提升开发效率

### Q: 旧代码需要全部重构吗？

A: 不需要立即全部重构。建议：
- 新功能严格遵循新规范
- 旧功能在修改时逐步迁移
- 优先重构高频使用的页面

### Q: 如何确保团队成员遵循规范？

A: 建议：
- 代码审查时检查规范遵循情况
- 在 PR 模板中添加规范检查项
- 定期组织规范培训

---

## 📚 相关资源

- [Element Plus 文档](https://element-plus.org/)
- [Vue 3 组合式 API](https://cn.vuejs.org/guide/extras/composition-api-faq.html)
- [响应式设计最佳实践](https://web.dev/responsive-web-design-basics/)
