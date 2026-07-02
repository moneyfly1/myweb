<template>
  <div class="list-container admin-coupons">
    <div class="page-header">
      <h1>优惠券管理</h1>
      <el-button type="primary" @click="showCreateDialog = true" class="create-btn">
        <el-icon><Plus /></el-icon>
        <span class="desktop-only">创建优惠券</span>
      </el-button>
    </div>
    <div class="mobile-action-bar" v-if="isMobile">
      <div class="mobile-search-section">
        <div class="search-input-wrapper">
          <el-input
            v-model="filters.keyword"
            placeholder="搜索优惠券码或名称"
            class="mobile-search-input"
            clearable
            @keyup.enter="searchCoupons"
          />
          <el-button 
            @click="searchCoupons" 
            class="search-button-inside"
            type="default"
            plain
          >
            <el-icon><Search /></el-icon>
          </el-button>
        </div>
      </div>
      <div class="mobile-filter-buttons">
        <el-button
          size="small"
          :type="showFilterDrawer ? 'primary' : 'default'"
          plain
          @click="showFilterDrawer = true"
        >
          <el-icon><Filter /></el-icon>
          筛选
        </el-button>
        <el-button size="small" type="default" plain @click="resetFilters">
          <el-icon><Refresh /></el-icon>
          重置
        </el-button>
      </div>
    </div>
    <div class="filter-bar desktop-only">
      <el-input
        v-model="filters.keyword"
        placeholder="搜索优惠券码或名称"
        class="filter-keyword"
        clearable
        @clear="searchCoupons"
      />
      <el-select v-model="filters.status" placeholder="状态筛选" clearable class="filter-select">
        <el-option label="有效" value="active" />
        <el-option label="无效" value="inactive" />
        <el-option label="已过期" value="expired" />
      </el-select>
      <el-select v-model="filters.type" placeholder="类型筛选" clearable class="filter-select">
        <el-option label="折扣" value="discount" />
        <el-option label="固定金额" value="fixed" />
        <el-option label="赠送天数" value="free_days" />
      </el-select>
      <el-button type="primary" @click="searchCoupons">
        <el-icon><Search /></el-icon>
        搜索
      </el-button>
      <el-button @click="resetFilters">
        <el-icon><Refresh /></el-icon>
        重置
      </el-button>
    </div>
    <AppDrawer
      v-model="showFilterDrawer"
      title="筛选条件"
      size="400px"
      mobile-size="85%"
      class="coupon-filter-drawer"
    >
      <div class="filter-drawer-content">
        <el-form label-width="100px">
          <el-form-item label="状态">
            <el-select v-model="filters.status" placeholder="选择状态" clearable class="full-width-control">
              <el-option label="有效" value="active" />
              <el-option label="无效" value="inactive" />
              <el-option label="已过期" value="expired" />
            </el-select>
          </el-form-item>
          <el-form-item label="类型">
            <el-select v-model="filters.type" placeholder="选择类型" clearable class="full-width-control">
              <el-option label="折扣" value="discount" />
              <el-option label="固定金额" value="fixed" />
              <el-option label="赠送天数" value="free_days" />
            </el-select>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <FormActionBar
          cancel-text="重置"
          submit-text="应用"
          :sticky="false"
          @cancel="resetFilters"
          @submit="applyFilters"
        />
      </template>
    </AppDrawer>
    <ResponsiveDataView
      :data="coupons"
      :loading="loading"
      :fields="mobileCouponFields"
      title-field="code"
      empty-title="暂无优惠券数据"
      empty-description="创建优惠券后可在这里统一管理有效期、类型和使用情况"
    >
      <template #table>
        <el-table :data="coupons" v-loading="loading" class="coupons-table" stripe border empty-text=" ">
          <template #empty>
            <EmptyState
              title="暂无优惠券数据"
              description="创建优惠券后可在这里统一管理有效期、类型和使用情况"
            />
          </template>
          <el-table-column prop="code" label="优惠券码" width="150" />
          <el-table-column prop="name" label="名称" />
          <el-table-column prop="type" label="类型" width="100">
            <template #default="{ row }">
              <el-tag>{{ getTypeText(row.type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="discount_value" label="优惠值" width="120">
            <template #default="{ row }">
              {{ formatDiscountValue(row) }}
            </template>
          </el-table-column>
          <el-table-column prop="valid_until" label="有效期至" width="180">
            <template #default="{ row }">
              {{ formatTime(row.valid_until) }}
            </template>
          </el-table-column>
          <el-table-column prop="used_quantity" label="使用情况" width="120">
            <template #default="{ row }">
              {{ row.used_quantity }} / {{ row.total_quantity || '∞' }}
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusTagType(row.status)">{{ getStatusText(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200">
            <template #default="{ row }">
              <el-button size="small" @click="editCoupon(row)">编辑</el-button>
              <el-button size="small" type="danger" @click="deleteCoupon(row.id)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>
      <template #header="{ item }">
        <div class="coupon-card-header">
          <div class="coupon-code">{{ item.code }}</div>
          <el-tag :type="getStatusTagType(item.status)" size="small">
            {{ getStatusText(item.status) }}
          </el-tag>
        </div>
        <div class="coupon-card-name">{{ item.name }}</div>
      </template>
      <template #field-type="{ item }">
        <el-tag size="small">{{ getTypeText(item.type) }}</el-tag>
      </template>
      <template #field-discount_value="{ item }">
        <span class="coupon-value-highlight">{{ formatDiscountValue(item) }}</span>
      </template>
      <template #field-valid_until="{ item }">
        {{ formatTime(item.valid_until) }}
      </template>
      <template #field-usage="{ item }">
        {{ item.used_quantity }} / {{ item.total_quantity || '∞' }}
      </template>
      <template #actions="{ item }">
        <el-button size="small" @click="editCoupon(item)" class="mobile-action-btn">编辑</el-button>
        <el-button size="small" type="danger" @click="deleteCoupon(item.id)" class="mobile-action-btn">删除</el-button>
      </template>
    </ResponsiveDataView>
    <PaginationBar
      v-model:current-page="pagination.page"
      v-model:page-size="pagination.size"
      :page-sizes="[10, 20, 50, 100]"
      :total="pagination.total"
      @size-change="handlePageSizeChange"
      @current-change="loadCoupons"
    />
    <AppDrawer
      v-model="showCreateDialog"
      :title="editingCoupon ? '编辑优惠券' : '创建优惠券'"
      size="500px"
      mobile-size="100%"
      :loading="saving"
      class="coupon-form-drawer"
    >
      <el-form 
        :model="couponForm" 
        :rules="couponRules" 
        ref="couponFormRef" 
        :label-width="isMobile ? '0' : '120px'"
        :label-position="isMobile ? 'top' : 'right'"
      >
        <el-form-item label="优惠券码" prop="code" v-if="!editingCoupon">
          <template v-if="isMobile">
            <div class="mobile-label">优惠券码</div>
          </template>
          <el-input v-model="couponForm.code" placeholder="留空自动生成" />
        </el-form-item>
        <el-form-item label="名称" prop="name">
          <template v-if="isMobile">
            <div class="mobile-label">名称 <span class="required">*</span></div>
          </template>
          <el-input v-model="couponForm.name" placeholder="请输入优惠券名称" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <template v-if="isMobile">
            <div class="mobile-label">描述</div>
          </template>
          <el-input v-model="couponForm.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <template v-if="isMobile">
            <div class="mobile-label">类型 <span class="required">*</span></div>
          </template>
          <el-select v-model="couponForm.type" placeholder="请选择类型" class="full-width-control">
            <el-option label="折扣（百分比）" value="discount" />
            <el-option label="固定金额减免" value="fixed" />
            <el-option label="赠送天数" value="free_days" />
          </el-select>
        </el-form-item>
        <el-form-item label="优惠值" prop="discount_value">
          <template v-if="isMobile">
            <div class="mobile-label">优惠值 <span class="required">*</span></div>
          </template>
          <div class="discount-value-wrapper">
            <el-input-number
              v-model="couponForm.discount_value"
              :min="0"
              :precision="2"
              class="full-width-control"
            />
            <span v-if="couponForm.type === 'discount'" class="discount-unit">%</span>
            <span v-else-if="couponForm.type === 'fixed'" class="discount-unit">元</span>
            <span v-else class="discount-unit">天</span>
          </div>
        </el-form-item>
        <el-form-item label="最低消费" prop="min_amount" v-if="couponForm.type !== 'free_days'">
          <template v-if="isMobile">
            <div class="mobile-label">最低消费</div>
          </template>
          <el-input-number
            v-model="couponForm.min_amount"
            :min="0"
            :precision="2"
            class="full-width-control"
          />
        </el-form-item>
        <el-form-item label="最大折扣" prop="max_discount" v-if="couponForm.type === 'discount'">
          <template v-if="isMobile">
            <div class="mobile-label">最大折扣</div>
          </template>
          <el-input-number
            v-model="couponForm.max_discount"
            :min="0"
            :precision="2"
            class="full-width-control"
          />
        </el-form-item>
        <el-form-item label="生效时间" prop="valid_from">
          <template v-if="isMobile">
            <div class="mobile-label">生效时间 <span class="required">*</span></div>
          </template>
          <el-date-picker
            v-model="couponForm.valid_from"
            type="datetime"
            placeholder="选择生效时间"
            class="full-width-control"
            :teleported="isMobile"
            :popper-class="isMobile ? 'mobile-date-picker-popper' : ''"
          />
        </el-form-item>
        <el-form-item label="失效时间" prop="valid_until">
          <template v-if="isMobile">
            <div class="mobile-label">失效时间 <span class="required">*</span></div>
          </template>
          <el-date-picker
            v-model="couponForm.valid_until"
            type="datetime"
            placeholder="选择失效时间"
            class="full-width-control"
            :teleported="isMobile"
            :popper-class="isMobile ? 'mobile-date-picker-popper' : ''"
          />
        </el-form-item>
        <el-form-item label="总数量" prop="total_quantity">
          <template v-if="isMobile">
            <div class="mobile-label">总数量</div>
          </template>
          <el-input-number
            v-model="couponForm.total_quantity"
            :min="1"
            placeholder="留空表示无限制"
            class="full-width-control"
          />
        </el-form-item>
        <el-form-item label="每用户限用" prop="max_uses_per_user">
          <template v-if="isMobile">
            <div class="mobile-label">每用户限用</div>
          </template>
          <el-input-number
            v-model="couponForm.max_uses_per_user"
            :min="1"
            class="full-width-control"
          />
        </el-form-item>
        <el-form-item label="适用套餐" prop="applicable_packages">
          <template v-if="isMobile">
            <div class="mobile-label">适用套餐</div>
          </template>
          <el-select
            v-model="couponForm.applicable_packages"
            multiple
            placeholder="留空表示所有套餐"
            class="full-width-control"
          >
            <el-option label="自定义套餐" value="custom_package" />
            <el-option
              v-for="pkg in packageOptions"
              :key="pkg.id"
              :label="pkg.name"
              :value="String(pkg.id)"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <FormActionBar
          :loading="saving"
          :sticky="false"
          @cancel="showCreateDialog = false"
          @submit="saveCoupon"
        />
      </template>
    </AppDrawer>
  </div>
</template>
<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { Plus, Search, Filter, Refresh } from '@element-plus/icons-vue'
import { couponAPI, packageAPI } from '@/utils/api'
import { useMobile } from '@/composables/useMobile'
import AppDrawer from '@/components/AppDrawer.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import EmptyState from '@/components/EmptyState.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import { confirmDelete } from '@/utils/confirmAction'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
import { formatTime as formatTimeUtil } from '@/utils/date'
dayjs.extend(timezone)
const loading = ref(false)
const saving = ref(false)
const coupons = ref([])
const packageOptions = ref([])
const showCreateDialog = ref(false)
const showFilterDrawer = ref(false)
const editingCoupon = ref(null)
const couponFormRef = ref(null)
const isMobile = useMobile()
const filters = reactive({
  keyword: '',
  status: '',
  type: ''
})
const pagination = reactive({
  page: 1,
  size: 10,
  total: 0
})
const mobileCouponFields = computed(() => [
  { key: 'type', label: '类型' },
  { key: 'discount_value', label: '优惠值' },
  { key: 'valid_until', label: '有效期至' },
  { key: 'usage', label: '使用情况' }
])
const couponForm = reactive({
  code: '',
  name: '',
  description: '',
  type: 'discount',
  discount_value: 0,
  min_amount: 0,
  max_discount: null,
  valid_from: null,
  valid_until: null,
  total_quantity: null,
  max_uses_per_user: 1,
  applicable_packages: []
})
const couponRules = {
  name: [{ required: true, message: '请输入优惠券名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
  discount_value: [{ required: true, message: '请输入优惠值', trigger: 'blur' }],
  valid_from: [{ required: true, message: '请选择生效时间', trigger: 'change' }],
  valid_until: [{ required: true, message: '请选择失效时间', trigger: 'change' }]
}
const loadCoupons = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      size: pagination.size
    }
    if (filters.keyword && filters.keyword.trim()) params.keyword = filters.keyword.trim()
    if (filters.status && filters.status.trim()) params.status = filters.status.trim()
    if (filters.type && filters.type.trim()) params.type = filters.type.trim()
    const response = await couponAPI.getAllCoupons(params)
    if (response.data && response.data.success) {
      coupons.value = response.data.data?.coupons || []
      pagination.total = response.data.data?.total || 0
    } else {
      ElMessage.error(response.data?.message || '加载优惠券列表失败')
    }
  } catch (error) {
    const errorMsg = error.response?.data?.message || error.message || '加载优惠券列表失败'
    ElMessage.error(errorMsg)
  } finally {
    loading.value = false
  }
}
const searchCoupons = () => {
  pagination.page = 1
  loadCoupons()
}
const handlePageSizeChange = () => {
  pagination.page = 1
  loadCoupons()
}
const loadPackages = async () => {
  try {
    const response = await packageAPI.getPackages({ size: 200 })
    const data = response.data?.data || response.data || {}
    const list = Array.isArray(data.packages)
      ? data.packages
      : Array.isArray(data)
        ? data
        : []
    packageOptions.value = list.map(pkg => ({
      id: pkg.id,
      name: pkg.name || `套餐 ${pkg.id}`
    })).filter(pkg => pkg.id)
  } catch (error) {
    packageOptions.value = []
  }
}
const parseApplicablePackages = (value) => {
  if (!value) return []
  if (Array.isArray(value)) return value.map(item => String(item))
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed) return []
    try {
      const parsed = JSON.parse(trimmed)
      if (Array.isArray(parsed)) return parsed.map(item => String(item))
    } catch (error) {}
    return trimmed.split(',').map(item => item.trim()).filter(Boolean)
  }
  return []
}
const saveCoupon = async () => {
  if (!couponFormRef.value) return
  await couponFormRef.value.validate(async (valid) => {
    if (valid) {
      saving.value = true
      try {
        const formData = { ...couponForm }
        if (formData.valid_from) {
          formData.valid_from = dayjs(formData.valid_from).tz('Asia/Shanghai').format('YYYY-MM-DDTHH:mm:ss')
        }
        if (formData.valid_until) {
          formData.valid_until = dayjs(formData.valid_until).tz('Asia/Shanghai').format('YYYY-MM-DDTHH:mm:ss')
        }
        if (!formData.code || formData.code.trim() === '') {
          delete formData.code // 让后端自动生成
        }
        if (formData.min_amount === 0 || formData.min_amount === null) {
          formData.min_amount = 0
        }
        if (formData.max_discount === null || formData.max_discount === undefined) {
          delete formData.max_discount
        }
        if (formData.max_discount === 0) {
          delete formData.max_discount
        }
        if (formData.total_quantity === null || formData.total_quantity === undefined) {
          delete formData.total_quantity
        }
        if (!formData.max_uses_per_user || formData.max_uses_per_user === 0) {
          formData.max_uses_per_user = 1 // 默认值
        }
        if (formData.applicable_packages) {
          if (Array.isArray(formData.applicable_packages)) {
            formData.applicable_packages = formData.applicable_packages.join(',')
          }
        } else {
          formData.applicable_packages = ''
        }
        if (formData.type === 'free_days') {
          formData.min_amount = 0
          delete formData.max_discount
        }
        let response
        if (editingCoupon.value) {
          response = await couponAPI.updateCoupon(editingCoupon.value.id, formData)
        } else {
          response = await couponAPI.createCoupon(formData)
        }
        if (response?.data?.success) {
          ElMessage.success(editingCoupon.value ? '优惠券更新成功' : '优惠券创建成功')
          showCreateDialog.value = false
          resetForm()
          await loadCoupons()
        } else {
          throw new Error(response?.data?.message || '操作失败')
        }
      } catch (error) {
        const errorMsg = error.response?.data?.message || error.message || '操作失败'
        ElMessage.error(editingCoupon.value ? `更新失败: ${errorMsg}` : `创建失败: ${errorMsg}`)
        console.error('优惠券操作失败:', error)
      } finally {
        saving.value = false
      }
    }
  })
}
const editCoupon = (coupon) => {
  editingCoupon.value = coupon
  Object.assign(couponForm, {
    name: coupon.name,
    description: coupon.description || '',
    type: coupon.type,
    discount_value: coupon.discount_value,
    min_amount: coupon.min_amount || 0,
    max_discount: coupon.max_discount,
    valid_from: coupon.valid_from ? dayjs(coupon.valid_from).tz('Asia/Shanghai').toDate() : null,
    valid_until: coupon.valid_until ? dayjs(coupon.valid_until).tz('Asia/Shanghai').toDate() : null,
    total_quantity: coupon.total_quantity,
    max_uses_per_user: coupon.max_uses_per_user,
    applicable_packages: parseApplicablePackages(coupon.applicable_packages)
  })
  showCreateDialog.value = true
}
const deleteCoupon = async (couponId) => {
  try {
    await confirmDelete('优惠券', 1, {
      message: '确定要删除此优惠券吗？'
    })
    const response = await couponAPI.deleteCoupon(couponId)
    if (response.data.success) {
      ElMessage.success('删除成功')
      loadCoupons()
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}
const resetForm = () => {
  editingCoupon.value = null
  Object.assign(couponForm, {
    code: '',
    name: '',
    description: '',
    type: 'discount',
    discount_value: 0,
    min_amount: 0,
    max_discount: null,
    valid_from: null,
    valid_until: null,
    total_quantity: null,
    max_uses_per_user: 1,
    applicable_packages: []
  })
}
const formatDiscountValue = (row) => {
  if (row.type === 'discount') {
    return `${row.discount_value}%`
  } else if (row.type === 'fixed') {
    return `¥${row.discount_value}`
  } else {
    return `${row.discount_value}天`
  }
}
const getTypeText = (type) => {
  const map = {
    discount: '折扣',
    fixed: '固定金额',
    free_days: '赠送天数'
  }
  return map[type] || type
}
const getStatusText = (status) => {
  const map = {
    active: '有效',
    inactive: '无效',
    expired: '已过期'
  }
  return map[status] || status
}
const getStatusTagType = (status) => {
  const map = {
    active: 'success',
    inactive: 'info',
    expired: 'danger'
  }
  return map[status] || ''
}
const formatTime = (timeStr) => {
  return formatTimeUtil(timeStr) || '-'
}
const resetFilters = () => {
  filters.keyword = ''
  filters.status = ''
  filters.type = ''
  pagination.page = 1
  showFilterDrawer.value = false
  loadCoupons()
}
const applyFilters = () => {
  pagination.page = 1
  showFilterDrawer.value = false
  loadCoupons()
}
onMounted(() => {
  loadCoupons()
  loadPackages()
})
</script>
<style scoped lang="scss">
.admin-coupons {
  padding: 20px;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;

  h1 {
    margin: 0;
    color: #303133;
    font-size: 22px;
    font-weight: 600;
    line-height: 1.3;
  }
}
.filter-bar {
  display: grid;
  grid-template-columns: minmax(220px, 1.2fr) repeat(2, minmax(150px, 0.8fr)) max-content max-content;
  align-items: end;
  gap: 12px;
  margin-bottom: 20px;
  padding: 14px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 8px;
}

.filter-keyword {
  width: 100%;
  min-width: 0;
}

.filter-select {
  width: 100%;
  min-width: 0;
}

.coupons-table,
.full-width-control {
  width: 100%;
}
.admin-coupons :deep(.el-input-number) {
  width: 100%;
}
.admin-coupons :deep(.el-date-editor) {
  width: 100%;
}
@media (max-width: 768px) {
  .admin-coupons {
    padding: 12px;
  }
  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 16px;
    :is(h1) {
      font-size: 20px;
      margin: 0;
    }
    .create-btn {
      width: 100%;
      height: 44px;
    }
  }
  .filter-bar.desktop-only {
    display: none;
  }
  .filter-drawer-content {
    padding: 20px 0;
  }
  .mobile-label {
    display: block;
    margin-bottom: 8px;
    color: #606266;
    font-size: 14px;
    font-weight: 600;

    .required {
      color: #f56c6c;
      margin-left: 2px;
    }
  }
  .discount-value-wrapper {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;

    .el-input-number {
      flex: 1;
    }

    .discount-unit {
      font-size: 14px;
      color: #909399;
      min-width: 30px;
      flex-shrink: 0;
    }
  }
  :deep(.mobile-date-picker-popper) {
    .el-picker-panel {
      width: 95vw;
      max-width: 400px;
    }
    .el-date-picker__header {
      padding: 12px 16px;
    }
    .el-picker-panel__content {
      padding: 8px;
    }
  }
}
.mobile-action-btn {
  width: 100%;
  min-height: 44px;
  margin: 0;
  font-size: 14px;
  border-radius: 6px;
  font-weight: 500;
  touch-action: manipulation;
}
.coupon-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  min-width: 0;
  margin-bottom: 8px;
}
.coupon-code {
  flex: 1;
  min-width: 0;
  font-weight: 700;
  font-size: 15px;
  color: #303133;
  font-family: 'Courier New', monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.coupon-card-name {
  font-size: 14px;
  font-weight: 500;
  color: #606266;
  line-height: 1.45;
  word-break: break-word;
}
.coupon-value-highlight {
  color: #f56c6c;
  font-weight: 600;
}
.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}
@media (min-width: 769px) {
  .mobile-action-bar {
    display: none !important;
  }
}
</style>
