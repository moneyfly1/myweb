<template>
  <div class="list-container admin-promotions">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>营销活动管理</span>
          <el-button type="primary" @click="showDrawer()">
            <el-icon><Plus /></el-icon>
            新建活动
          </el-button>
        </div>
      </template>

      <div class="filter-bar">
        <el-select v-model="filter.type" placeholder="活动类型" clearable class="filter-type" @change="loadData">
          <el-option label="限时抢购" value="flash_sale" />
          <el-option label="新用户优惠" value="new_user" />
          <el-option label="召回活动" value="recall" />
          <el-option label="会员日" value="member_day" />
        </el-select>
        <el-select v-model="filter.is_active" placeholder="状态" clearable class="filter-status" @change="loadData">
          <el-option label="启用" :value="true" />
          <el-option label="禁用" :value="false" />
        </el-select>
        <div class="filter-actions">
          <el-button type="primary" @click="loadData">搜索</el-button>
          <el-button @click="resetFilter">重置</el-button>
        </div>
      </div>

      <ResponsiveDataView
        class="promotions-data"
        :data="promotions"
        :fields="mobilePromotionFields"
        :loading="loading"
        empty-title="暂无营销活动"
        empty-description="可新建活动或调整筛选条件"
      >
        <template #table>
          <el-table :data="promotions" v-loading="loading" stripe border class="promotions-table">
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="name" label="活动名称" min-width="150" show-overflow-tooltip />
            <el-table-column prop="type" label="类型" width="120">
              <template #default="{ row }">
                <el-tag :type="getTypeTagType(row.type)" size="small">
                  {{ typeMap[row.type] || row.type }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="折扣" width="140">
              <template #default="{ row }">
                <el-tag type="danger" size="small">
                  {{ formatDiscount(row) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="活动时间" min-width="200">
              <template #default="{ row }">
                <div class="time-range">
                  <div>{{ formatDate(row.start_time) }}</div>
                  <div class="time-separator">至</div>
                  <div>{{ formatDate(row.end_time) }}</div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="getStatusType(row)" size="small">
                  {{ getStatusText(row) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="180" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="showDrawer(row)">编辑</el-button>
                <el-button size="small" type="danger" @click="remove(row.id)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </template>

        <template #header="{ item }">
          <div class="mobile-promotion-header">
            <div class="mobile-promotion-title">
              <span class="promotion-name">{{ item.name }}</span>
              <span class="promotion-id">#{{ item.id }}</span>
            </div>
            <el-tag :type="getStatusType(item)" size="small">
              {{ getStatusText(item) }}
            </el-tag>
          </div>
        </template>

        <template #field-type="{ item }">
          <el-tag :type="getTypeTagType(item.type)" size="small" effect="plain">
            {{ typeMap[item.type] || item.type }}
          </el-tag>
        </template>

        <template #field-discount="{ item }">
          <el-tag type="danger" size="small" effect="plain">
            {{ formatDiscount(item) }}
          </el-tag>
        </template>

        <template #field-time_range="{ item }">
          <div class="mobile-time-range">
            <span>{{ formatDate(item.start_time) }}</span>
            <span class="time-separator">至</span>
            <span>{{ formatDate(item.end_time) }}</span>
          </div>
        </template>

        <template #actions="{ item }">
          <div class="mobile-promotion-actions">
            <el-button size="small" @click="showDrawer(item)">编辑</el-button>
            <el-button size="small" type="danger" @click="remove(item.id)">删除</el-button>
          </div>
        </template>
      </ResponsiveDataView>

      <PaginationBar
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.page_size"
        :total="pagination.total"
        @change="loadData"
      />
    </el-card>

    <!-- 活动抽屉 -->
    <AppDrawer
      v-model="drawerVisible"
      :title="form.id ? '编辑活动' : '新建活动'"
      size="600px"
      mobile-size="100%"
      :loading="saving"
      class="promotion-form-drawer"
    >
      <el-form :model="form" label-width="100px" :rules="rules" ref="formRef">
        <el-form-item label="活动名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入活动名称" />
        </el-form-item>

        <el-form-item label="活动类型" prop="type">
          <el-select v-model="form.type" placeholder="请选择活动类型" class="form-control">
            <el-option label="限时抢购" value="flash_sale">
              <div class="option-item">
                <span>限时抢购</span>
                <span class="option-desc">短期内的特价促销</span>
              </div>
            </el-option>
            <el-option label="新用户优惠" value="new_user">
              <div class="option-item">
                <span>新用户优惠</span>
                <span class="option-desc">首次购买用户专享</span>
              </div>
            </el-option>
            <el-option label="召回活动" value="recall">
              <div class="option-item">
                <span>召回活动</span>
                <span class="option-desc">针对流失用户的优惠</span>
              </div>
            </el-option>
            <el-option label="会员日" value="member_day">
              <div class="option-item">
                <span>会员日</span>
                <span class="option-desc">定期会员专享优惠</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>

        <el-form-item label="折扣类型" prop="discount_type">
          <el-radio-group v-model="form.discount_type">
            <el-radio label="percentage">百分比折扣</el-radio>
            <el-radio label="fixed">固定减免</el-radio>
            <el-radio label="free_days">赠送天数</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="折扣值" prop="discount_value">
          <el-input-number
            v-model="form.discount_value"
            :min="0"
            :max="form.discount_type === 'percentage' ? 100 : 99999"
            :precision="form.discount_type === 'free_days' ? 0 : 2"
            :step="form.discount_type === 'percentage' ? 5 : 10"
            class="number-input"
          />
          <span class="form-hint">
            {{ getDiscountUnit() }}
          </span>
        </el-form-item>

        <el-form-item label="最低消费" v-if="form.discount_type !== 'free_days'">
          <el-input-number v-model="form.min_amount" :min="0" :precision="2" class="number-input" />
          <span class="form-hint">元（0表示无限制）</span>
        </el-form-item>

        <el-form-item label="最高优惠" v-if="form.discount_type === 'percentage'">
          <el-input-number v-model="form.max_discount" :min="0" :precision="2" class="number-input" />
          <span class="form-hint">元（0表示无限制）</span>
        </el-form-item>

        <el-form-item label="开始时间" prop="start_time">
          <el-date-picker
            v-model="form.start_time"
            type="datetime"
            placeholder="选择开始时间"
            class="form-control"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
          />
        </el-form-item>

        <el-form-item label="结束时间" prop="end_time">
          <el-date-picker
            v-model="form.end_time"
            type="datetime"
            placeholder="选择结束时间"
            class="form-control"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
          />
        </el-form-item>

        <el-form-item label="活动描述">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="4"
            placeholder="请输入活动描述（选填）"
            maxlength="500"
            show-word-limit
          />
        </el-form-item>

        <el-form-item label="启用状态">
          <el-switch v-model="form.is_active" />
          <span class="form-hint">
            {{ form.is_active ? '启用' : '禁用' }}
          </span>
        </el-form-item>
      </el-form>

      <template #footer>
        <FormActionBar
          :loading="saving"
          submit-text="保存"
          @cancel="drawerVisible = false"
          @submit="save"
        />
      </template>
    </AppDrawer>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { Plus } from '@element-plus/icons-vue'
import { promotionAPI } from '@/utils/api'
import AppDrawer from '@/components/AppDrawer.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import { confirmDelete } from '@/utils/confirmAction'

const loading = ref(false)
const saving = ref(false)
const promotions = ref([])
const drawerVisible = ref(false)
const formRef = ref()

const filter = reactive({
  type: null,
  is_active: null
})

const pagination = reactive({
  page: 1,
  page_size: 10,
  total: 0
})

const form = ref({
  name: '',
  type: 'flash_sale',
  discount_type: 'percentage',
  discount_value: 10,
  min_amount: 0,
  max_discount: 0,
  start_time: '',
  end_time: '',
  description: '',
  is_active: true
})

const rules = {
  name: [{ required: true, message: '请输入活动名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择活动类型', trigger: 'change' }],
  discount_type: [{ required: true, message: '请选择折扣类型', trigger: 'change' }],
  discount_value: [{ required: true, message: '请输入折扣值', trigger: 'blur' }],
  start_time: [{ required: true, message: '请选择开始时间', trigger: 'change' }],
  end_time: [{ required: true, message: '请选择结束时间', trigger: 'change' }]
}

const typeMap = {
  flash_sale: '限时抢购',
  new_user: '新用户优惠',
  recall: '召回活动',
  member_day: '会员日'
}

const mobilePromotionFields = [
  { key: 'type', label: '活动类型' },
  { key: 'discount', label: '折扣' },
  { key: 'time_range', label: '活动时间', fullWidth: true }
]

const formatDate = (d) => {
  if (!d) return ''
  const date = new Date(d)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const formatDiscount = (row) => {
  if (row.discount_type === 'percentage') {
    return `${row.discount_value}% 折扣`
  } else if (row.discount_type === 'fixed') {
    return `减免 ¥${row.discount_value}`
  } else if (row.discount_type === 'free_days') {
    return `赠送 ${row.discount_value} 天`
  }
  return '-'
}

const getTypeTagType = (type) => {
  const map = {
    flash_sale: 'danger',
    new_user: 'success',
    recall: 'warning',
    member_day: 'primary'
  }
  return map[type] || ''
}

const getStatusType = (row) => {
  if (!row.is_active) return 'info'
  const now = new Date()
  const start = new Date(row.start_time)
  const end = new Date(row.end_time)
  if (now < start) return 'warning'
  if (now > end) return 'info'
  return 'success'
}

const getStatusText = (row) => {
  if (!row.is_active) return '已禁用'
  const now = new Date()
  const start = new Date(row.start_time)
  const end = new Date(row.end_time)
  if (now < start) return '未开始'
  if (now > end) return '已结束'
  return '进行中'
}

const getDiscountUnit = () => {
  if (form.value.discount_type === 'percentage') return '%'
  if (form.value.discount_type === 'fixed') return '元'
  if (form.value.discount_type === 'free_days') return '天'
  return ''
}

let loadSeq = 0
const loadData = async () => {
  const seq = ++loadSeq
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.page_size
    }
    if (filter.type) params.type = filter.type
    if (filter.is_active !== null) params.is_active = filter.is_active

    const res = await promotionAPI.getAll(params)
    if (seq !== loadSeq) return
    const data = res.data?.data || {}
    promotions.value = data.list || []
    pagination.total = data.total || 0
  } catch (e) {
    if (seq !== loadSeq) return
    ElMessage.error('加载数据失败')
  } finally {
    if (seq === loadSeq) {
      loading.value = false
    }
  }
}

const showDrawer = (row) => {
  if (row) {
    form.value = {
      ...row,
      start_time: row.start_time,
      end_time: row.end_time,
      description: row.description?.String || row.description || ''
    }
  } else {
    const now = new Date()
    const tomorrow = new Date(now.getTime() + 24 * 60 * 60 * 1000)
    const nextWeek = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000)

    form.value = {
      name: '',
      type: 'flash_sale',
      discount_type: 'percentage',
      discount_value: 10,
      min_amount: 0,
      max_discount: 0,
      start_time: tomorrow.toISOString(),
      end_time: nextWeek.toISOString(),
      description: '',
      is_active: true
    }
  }
  drawerVisible.value = true
}

const save = async () => {
  if (!formRef.value) return
  await formRef.value.validate()

  // 验证时间
  if (new Date(form.value.start_time) >= new Date(form.value.end_time)) {
    ElMessage.error('结束时间必须晚于开始时间')
    return
  }

  saving.value = true
  try {
    const data = { ...form.value }
    if (typeof data.description === 'string') {
      data.description = { String: data.description, Valid: !!data.description }
    }
    // 确保时间字段为ISO 8601格式
    if (data.start_time) data.start_time = new Date(data.start_time).toISOString()
    if (data.end_time) data.end_time = new Date(data.end_time).toISOString()

    if (data.id) {
      await promotionAPI.update(data.id, data)
      ElMessage.success('更新成功')
    } else {
      await promotionAPI.create(data)
      ElMessage.success('创建成功')
    }
    drawerVisible.value = false
    loadData()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const remove = async (id) => {
  await confirmDelete('活动', 1, {
    message: '确定删除该活动吗？删除后不可恢复。',
    title: '确认删除活动'
  })
  try {
    await promotionAPI.remove(id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '删除失败')
  }
}

const resetFilter = () => {
  filter.type = null
  filter.is_active = null
  pagination.page = 1
  loadData()
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.list-container {
  padding: 0;
}

.list-card {
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}

.filter-bar {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) minmax(160px, 0.8fr) minmax(150px, max-content);
  align-items: end;
  gap: 12px;
  margin-top: 16px;
  margin-bottom: 16px;
}

.filter-actions {
  display: flex;
  justify-self: end;
  align-items: center;
  gap: 8px;

  .el-button {
    margin-left: 0;
  }
}

.filter-type {
  width: 100%;
  min-width: 0;
}

.filter-status {
  width: 100%;
  min-width: 0;
}

@media (max-width: 768px) {
  .filter-bar {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .filter-actions {
    grid-column: 1 / -1;
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    justify-self: stretch;
    width: 100%;

    .el-button {
      width: 100%;
      margin-left: 0;
    }
  }
}

.promotions-data {
  min-width: 0;
}

.promotions-table {
  width: 100%;
}

.time-range {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
}

.time-separator {
  color: #909399;
}

.form-control {
  width: 100%;
}

.number-input {
  width: 100%;
  max-width: 220px;
}

.form-hint {
  margin-left: 12px;
  color: #909399;
}

.option-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.option-desc {
  font-size: 12px;
  color: #909399;
}

.mobile-promotion-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
}

.mobile-promotion-title {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.promotion-name {
  color: var(--el-text-color-primary);
  font-size: 15px;
  font-weight: 600;
  line-height: 1.35;
  word-break: break-word;
}

.promotion-id {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.mobile-time-range {
  display: flex;
  flex-direction: column;
  gap: 3px;
  line-height: 1.45;
}

.mobile-promotion-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  width: 100%;

  .el-button {
    margin: 0;
    min-height: 44px;
    touch-action: manipulation;
  }
}

/* 移动端适配 */
@media (max-width: 768px) {
  .card-header {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .filter-bar {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .filter-bar .el-select {
    width: 100%;
  }

  .time-range {
    font-size: 12px;
  }

  .admin-promotions :deep(.el-pagination) {
    flex-wrap: wrap;
  }

  .promotion-form-drawer {
    :deep(.el-form-item) {
      align-items: flex-start;
    }

    :deep(.el-form-item__content) {
      gap: 8px;
    }

    .number-input {
      width: 100%;
    }

    .form-hint {
      display: block;
      width: 100%;
      margin-left: 0;
      line-height: 1.5;
    }
  }
}
</style>
