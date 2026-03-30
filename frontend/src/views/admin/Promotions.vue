<template>
  <div class="list-container">
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
        <el-select v-model="filter.type" placeholder="活动类型" clearable style="width: 150px" @change="loadData">
          <el-option label="限时抢购" value="flash_sale" />
          <el-option label="新用户优惠" value="new_user" />
          <el-option label="召回活动" value="recall" />
          <el-option label="会员日" value="member_day" />
        </el-select>
        <el-select v-model="filter.is_active" placeholder="状态" clearable style="width: 120px" @change="loadData">
          <el-option label="启用" :value="true" />
          <el-option label="禁用" :value="false" />
        </el-select>
      </div>

      <el-table :data="promotions" v-loading="loading" stripe border>
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
              <div style="color: #909399">至</div>
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

      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.page_size"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadData"
          @current-change="loadData"
        />
      </div>
    </el-card>

    <!-- 活动抽屉 -->
    <el-drawer
      v-model="drawerVisible"
      :title="form.id ? '编辑活动' : '新建活动'"
      :size="isMobile ? '100%' : '600px'"
      direction="rtl"
      :lock-scroll="false"
    >
      <el-form :model="form" label-width="100px" :rules="rules" ref="formRef">
        <el-form-item label="活动名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入活动名称" />
        </el-form-item>

        <el-form-item label="活动类型" prop="type">
          <el-select v-model="form.type" placeholder="请选择活动类型" style="width: 100%">
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
            style="width: 200px"
          />
          <span style="margin-left: 12px; color: #909399">
            {{ getDiscountUnit() }}
          </span>
        </el-form-item>

        <el-form-item label="最低消费" v-if="form.discount_type !== 'free_days'">
          <el-input-number v-model="form.min_amount" :min="0" :precision="2" style="width: 200px" />
          <span style="margin-left: 12px; color: #909399">元（0表示无限制）</span>
        </el-form-item>

        <el-form-item label="最高优惠" v-if="form.discount_type === 'percentage'">
          <el-input-number v-model="form.max_discount" :min="0" :precision="2" style="width: 200px" />
          <span style="margin-left: 12px; color: #909399">元（0表示无限制）</span>
        </el-form-item>

        <el-form-item label="开始时间" prop="start_time">
          <el-date-picker
            v-model="form.start_time"
            type="datetime"
            placeholder="选择开始时间"
            style="width: 100%"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
          />
        </el-form-item>

        <el-form-item label="结束时间" prop="end_time">
          <el-date-picker
            v-model="form.end_time"
            type="datetime"
            placeholder="选择结束时间"
            style="width: 100%"
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
          <span style="margin-left: 12px; color: #909399">
            {{ form.is_active ? '启用' : '禁用' }}
          </span>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="drawer-footer">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="save">保存</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { promotionAPI } from '@/utils/api'

const isMobile = ref(window.innerWidth <= 768)
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
  page_size: 20,
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

const loadData = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.page_size
    }
    if (filter.type) params.type = filter.type
    if (filter.is_active !== null) params.is_active = filter.is_active

    const res = await promotionAPI.getAll(params)
    const data = res.data?.data || {}
    promotions.value = data.list || []
    pagination.total = data.total || 0
  } catch (e) {
    ElMessage.error('加载数据失败')
  } finally {
    loading.value = false
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
  await ElMessageBox.confirm('确定删除该活动？', '确认删除', {
    type: 'warning'
  })
  try {
    await promotionAPI.remove(id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '删除失败')
  }
}

const handleResize = () => {
  isMobile.value = window.innerWidth <= 768
}

onMounted(() => {
  loadData()
  window.addEventListener('resize', handleResize)
})
</script>

<style scoped>
.list-container {
  padding: 0;
}

.list-card {
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header span {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.filter-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.time-range {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
}

.pagination-wrapper {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
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

/* 移动端适配 */
@media (max-width: 768px) {
  .card-header {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .filter-bar {
    flex-direction: column;
  }

  .filter-bar .el-select {
    width: 100% !important;
  }

  :deep(.el-table) {
    font-size: 12px;
  }

  :deep(.el-table .el-button) {
    padding: 5px 8px;
    font-size: 12px;
  }

  .time-range {
    font-size: 12px;
  }

  .pagination-wrapper {
    justify-content: center;
  }

  :deep(.el-pagination) {
    flex-wrap: wrap;
    justify-content: center;
  }
}
</style>
