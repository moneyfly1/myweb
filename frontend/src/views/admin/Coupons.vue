<template>
  <div class="coupons-container">
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
            @keyup.enter="loadCoupons"
          />
          <el-button 
            @click="loadCoupons" 
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
        style="width: 200px"
        clearable
        @clear="loadCoupons"
      />
      <el-select v-model="filters.status" placeholder="状态筛选" clearable style="width: 150px">
        <el-option label="有效" value="active" />
        <el-option label="无效" value="inactive" />
        <el-option label="已过期" value="expired" />
      </el-select>
      <el-select v-model="filters.type" placeholder="类型筛选" clearable style="width: 150px">
        <el-option label="折扣" value="discount" />
        <el-option label="固定金额" value="fixed" />
        <el-option label="赠送天数" value="free_days" />
      </el-select>
      <el-button @click="loadCoupons">搜索</el-button>
    </div>
    <el-drawer
      v-model="showFilterDrawer"
      title="筛选条件"
      :size="isMobile ? '85%' : '400px'"
      direction="rtl"
    >
      <div class="filter-drawer-content">
        <el-form label-width="100px">
          <el-form-item label="状态">
            <el-select v-model="filters.status" placeholder="选择状态" clearable style="width: 100%">
              <el-option label="有效" value="active" />
              <el-option label="无效" value="inactive" />
              <el-option label="已过期" value="expired" />
            </el-select>
          </el-form-item>
          <el-form-item label="类型">
            <el-select v-model="filters.type" placeholder="选择类型" clearable style="width: 100%">
              <el-option label="折扣" value="discount" />
              <el-option label="固定金额" value="fixed" />
              <el-option label="赠送天数" value="free_days" />
            </el-select>
          </el-form-item>
        </el-form>
        <div class="filter-drawer-actions">
          <el-button @click="resetFilters" class="mobile-action-btn">重置</el-button>
          <el-button type="primary" @click="applyFilters" class="mobile-action-btn">应用</el-button>
        </div>
      </div>
    </el-drawer>
    <el-table :data="coupons" v-loading="loading" class="desktop-only" style="width: 100%">
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
      <el-table-column prop="valid_until" label="有效期至" width="180" />
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
    <div class="mobile-coupons-list" v-if="isMobile" v-loading="loading">
      <div
        v-for="coupon in coupons"
        :key="coupon.id"
        class="mobile-coupon-card"
      >
        <div class="coupon-card-header">
          <div class="coupon-code">{{ coupon.code }}</div>
          <el-tag :type="getStatusTagType(coupon.status)" size="small">{{ getStatusText(coupon.status) }}</el-tag>
        </div>
        <div class="coupon-card-name">{{ coupon.name }}</div>
        <div class="coupon-card-info">
          <div class="info-row">
            <span class="info-label">类型：</span>
            <el-tag size="small">{{ getTypeText(coupon.type) }}</el-tag>
          </div>
          <div class="info-row">
            <span class="info-label">优惠值：</span>
            <span class="info-value highlight">{{ formatDiscountValue(coupon) }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">有效期至：</span>
            <span class="info-value">{{ formatTime(coupon.valid_until) }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">使用情况：</span>
            <span class="info-value">{{ coupon.used_quantity }} / {{ coupon.total_quantity || '∞' }}</span>
          </div>
        </div>
        <div class="coupon-card-actions">
          <el-button size="small" @click.stop="editCoupon(coupon)" class="mobile-action-btn">编辑</el-button>
          <el-button size="small" type="danger" @click.stop="deleteCoupon(coupon.id)" class="mobile-action-btn">删除</el-button>
        </div>
      </div>
      <div v-if="coupons.length === 0" class="empty-state">
        <el-empty description="暂无优惠券数据" />
      </div>
    </div>
    <el-pagination
      v-model:current-page="pagination.page"
      v-model:page-size="pagination.size"
      :total="pagination.total"
      :page-sizes="[10, 20, 50, 100]"
      layout="total, sizes, prev, pager, next, jumper"
      @size-change="loadCoupons"
      @current-change="loadCoupons"
      style="margin-top: 20px; justify-content: center"
    />
    <el-dialog
      v-model="showCreateDialog"
      :title="editingCoupon ? '编辑优惠券' : '创建优惠券'"
      :width="isMobile ? '95%' : '600px'"
      :close-on-click-modal="!isMobile"
      class="coupon-form-dialog"
      :class="{ 'mobile-dialog': isMobile }"
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
          <el-select v-model="couponForm.type" placeholder="请选择类型" style="width: 100%">
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
              style="width: 100%"
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
            style="width: 100%"
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
            style="width: 100%"
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
            style="width: 100%"
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
            style="width: 100%"
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
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="每用户限用" prop="max_uses_per_user">
          <template v-if="isMobile">
            <div class="mobile-label">每用户限用</div>
          </template>
          <el-input-number
            v-model="couponForm.max_uses_per_user"
            :min="1"
            style="width: 100%"
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
            style="width: 100%"
          >
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer-buttons" :class="{ 'mobile-footer': isMobile }">
          <el-button @click="showCreateDialog = false" :class="{ 'mobile-action-btn': isMobile }">取消</el-button>
          <el-button type="primary" @click="saveCoupon" :loading="saving" :class="{ 'mobile-action-btn': isMobile }">保存</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search, Filter, Refresh } from '@element-plus/icons-vue'
import { couponAPI } from '@/utils/api'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
import { formatTime as formatTimeUtil } from '@/utils/date'
dayjs.extend(timezone)

const loading = ref(false)
const saving = ref(false)
const coupons = ref([])
const showCreateDialog = ref(false)
const showFilterDrawer = ref(false)
const editingCoupon = ref(null)
const couponFormRef = ref(null)
const isMobile = ref(window.innerWidth <= 768)

const filters = reactive({
  keyword: '',
  status: '',
  type: ''
})

const pagination = reactive({
  page: 1,
  size: 20,
  total: 0
})

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

const saveCoupon = async () => {
  if (!couponFormRef.value) return
  await couponFormRef.value.validate(async (valid) => {
    if (valid) {
      saving.value = true
      try {
        const formData = { ...couponForm }
        // 转换日期格式（使用北京时间）
        if (formData.valid_from) {
          formData.valid_from = dayjs(formData.valid_from).tz('Asia/Shanghai').format('YYYY-MM-DDTHH:mm:ss')
        }
        if (formData.valid_until) {
          formData.valid_until = dayjs(formData.valid_until).tz('Asia/Shanghai').format('YYYY-MM-DDTHH:mm:ss')
        }
        
        // 清理空值字段
        if (!formData.code || formData.code.trim() === '') {
          delete formData.code // 让后端自动生成
        }
        if (formData.min_amount === 0 || formData.min_amount === null) {
          formData.min_amount = 0
        }
        if (formData.max_discount === null || formData.max_discount === undefined) {
          delete formData.max_discount
        }
        if (formData.total_quantity === null || formData.total_quantity === undefined) {
          delete formData.total_quantity
        }
        if (!formData.max_uses_per_user || formData.max_uses_per_user === 0) {
          formData.max_uses_per_user = 1 // 默认值
        }
        // 处理 applicable_packages：如果是数组，转换为字符串；如果为空，设置为空字符串
        if (formData.applicable_packages) {
          if (Array.isArray(formData.applicable_packages)) {
            formData.applicable_packages = formData.applicable_packages.join(',')
          }
        } else {
          formData.applicable_packages = ''
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
    applicable_packages: coupon.applicable_packages || []
  })
  showCreateDialog.value = true
}

const deleteCoupon = async (couponId) => {
  try {
    await ElMessageBox.confirm('确定要删除此优惠券吗？', '提示', {
      type: 'warning'
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
  // 使用统一的北京时间格式化函数
  return formatTimeUtil(timeStr) || '-'
}

const resetFilters = () => {
  filters.keyword = ''
  filters.status = ''
  filters.type = ''
  showFilterDrawer.value = false
  loadCoupons()
}

const applyFilters = () => {
  showFilterDrawer.value = false
  loadCoupons()
}

const handleResize = () => {
  isMobile.value = window.innerWidth <= 768
}

onMounted(() => {
  loadCoupons()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped lang="scss">
.coupons-container {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.filter-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
}

:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
  pointer-events: auto !important;
}

:deep(.el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background-color: transparent !important;
  pointer-events: auto !important;
}

:deep(.el-input__wrapper:hover) {
  border-color: #c0c4cc !important;
  box-shadow: none !important;
  background-color: #ffffff !important;
}

:deep(.el-input__wrapper.is-focus) {
  border-color: #1677ff !important;
  box-shadow: none !important;
  background-color: #ffffff !important;
}

:deep(.el-input__wrapper.is-focus:hover) {
  background-color: #ffffff !important;
}

:deep(.el-input__wrapper > *) {
  background-color: transparent !important;
  background: transparent !important;
}

:deep(.el-textarea__inner) {
  border-radius: 0 !important;
  border: 1px solid #dcdfe6 !important;
  box-shadow: none !important;
  background-color: #ffffff !important;
}

:deep(.el-textarea__inner:hover) {
  border-color: #c0c4cc !important;
}

:deep(.el-textarea__inner:focus) {
  border-color: #1677ff !important;
  box-shadow: none !important;
}

:deep(.el-select .el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
  pointer-events: auto !important;
}

:deep(.el-input-number) {
  width: 100%;
}

:deep(.el-input-number .el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
  pointer-events: auto !important;
}

:deep(.el-date-editor) {
  width: 100%;
}

:deep(.el-date-editor .el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
  pointer-events: auto !important;
}

/* 移动端样式 */
@media (max-width: 768px) {
  .coupons-container {
    padding: 12px;
  }

  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 16px;
    
    h1 {
      font-size: 20px;
      margin: 0;
    }
    
    .create-btn {
      width: 100%;
      height: 44px;
    }
  }

  // mobile-action-bar 样式已统一在 list-common.scss 中定义
  // 这里不再重复定义，使用统一样式

  .filter-bar.desktop-only {
    display: none;
  }

  .mobile-coupons-list {
    margin-top: 16px;
    
    .mobile-coupon-card {
      background: #fff;
      border-radius: 8px;
      padding: 16px;
      margin-bottom: 12px;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      
      .coupon-card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;
        
        .coupon-code {
          font-weight: bold;
          font-size: 16px;
          color: #333;
          font-family: 'Courier New', monospace;
        }
      }
      
      .coupon-card-name {
        font-size: 15px;
        font-weight: 500;
        color: #333;
        margin-bottom: 12px;
        line-height: 1.4;
      }
      
      .coupon-card-info {
        margin-bottom: 12px;
        
        .info-row {
          display: flex;
          align-items: center;
          margin-bottom: 8px;
          font-size: 14px;
          
          .info-label {
            color: #666;
            min-width: 80px;
          }
          
          .info-value {
            color: #333;
            flex: 1;
            
            &.highlight {
              color: #f56c6c;
              font-weight: 600;
              font-size: 16px;
            }
          }
        }
      }
      
      .coupon-card-actions {
        display: flex;
        gap: 8px;
        padding-top: 12px;
        border-top: 1px solid #f0f0f0;
        
        .mobile-action-btn {
          flex: 1;
          height: 40px;
        }
      }
    }
    
    .empty-state {
      padding: 40px 20px;
      text-align: center;
    }
  }

  .filter-drawer-content {
    padding: 20px 0;
    
    .filter-drawer-actions {
      display: flex;
      gap: 12px;
      margin-top: 24px;
      padding-top: 20px;
      border-top: 1px solid #f0f0f0;
      
      .mobile-action-btn {
        flex: 1;
        height: 44px;
      }
    }
  }

  .coupon-form-dialog {
    &.mobile-dialog {
      :deep(.el-dialog) {
        width: 95% !important;
        margin: 2vh auto !important;
        max-height: 96vh;
        border-radius: 8px;
        display: flex;
        flex-direction: column;
      }
      
      :deep(.el-dialog__header) {
        padding: 15px 15px 10px;
        flex-shrink: 0;
        border-bottom: 1px solid #ebeef5;
        
        .el-dialog__title {
          font-size: 18px;
          font-weight: 600;
        }
        
        .el-dialog__headerbtn {
          top: 8px;
          right: 8px;
          width: 32px;
          height: 32px;
          
          .el-dialog__close {
            font-size: 18px;
          }
        }
      }
      
      :deep(.el-dialog__body) {
        padding: 15px !important;
        flex: 1;
        overflow-y: auto;
        -webkit-overflow-scrolling: touch;
        max-height: calc(96vh - 140px);
      }
      
      :deep(.el-dialog__footer) {
        padding: 10px 15px 15px;
        flex-shrink: 0;
        border-top: 1px solid #ebeef5;
      }
    }
    
    :deep(.el-dialog) {
      width: 95% !important;
      margin: 1vh auto !important;
      max-height: 98vh;
      border-radius: 12px;
      display: flex;
      flex-direction: column;
    }
    
    :deep(.el-dialog__header) {
      padding: 16px 16px 12px;
      flex-shrink: 0;
      border-bottom: 1px solid #ebeef5;
      
      .el-dialog__title {
        font-size: 18px;
        font-weight: 600;
      }
      
      .el-dialog__headerbtn {
        top: 12px;
        right: 12px;
        width: 36px;
        height: 36px;
      }
    }
    
    :deep(.el-dialog__body) {
      padding: 16px !important;
      flex: 1;
      overflow-y: auto;
      -webkit-overflow-scrolling: touch;
      max-height: calc(98vh - 140px);
    }
    
    :deep(.el-dialog__footer) {
      padding: 12px 16px 16px;
      flex-shrink: 0;
      border-top: 1px solid #ebeef5;
      display: flex;
      flex-direction: column;
      gap: 10px;
    }
    
    :deep(.el-form-item) {
      margin-bottom: 18px;
      
      .el-form-item__label {
        display: none; /* 移动端隐藏默认标签 */
      }
      
      .el-form-item__content {
        margin-left: 0 !important;
        width: 100%;
      }
    }
    
    .mobile-label {
      font-size: 14px;
      font-weight: 600;
      color: #606266;
      margin-bottom: 8px;
      display: block;
      
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
    
    :deep(.el-input),
    :deep(.el-select),
    :deep(.el-textarea),
    :deep(.el-input-number) {
      width: 100%;
      
      .el-input__wrapper,
      .el-textarea__inner {
        min-height: 40px;
        font-size: 16px; /* 防止iOS自动缩放 */
      }
      
      .el-input__inner {
        font-size: 16px !important; /* 防止iOS自动缩放 */
        min-height: 40px;
        padding: 0 12px;
      }
    }
    
    :deep(.el-textarea .el-textarea__inner) {
      min-height: 100px;
      padding: 12px;
      line-height: 1.6;
      font-size: 16px; /* 防止iOS自动缩放 */
    }
    
    :deep(.el-input-number) {
      .el-input__wrapper {
        min-height: 40px;
      }
    }
  }

  .dialog-footer-buttons {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    
    &.mobile-footer {
      flex-direction: column;
      gap: 10px;
      
      .mobile-action-btn {
        width: 100%;
        min-height: 48px;
        font-size: 16px;
        font-weight: 500;
        margin: 0 !important;
        border-radius: 8px;
        -webkit-tap-highlight-color: rgba(0,0,0,0.1);
      }
    }
    
    .mobile-action-btn {
      width: 100%;
      min-height: 48px;
      font-size: 16px;
      font-weight: 500;
      margin: 0 !important;
      border-radius: 8px;
      -webkit-tap-highlight-color: rgba(0,0,0,0.1);
    }
  }
  
  /* 移动端日期选择器优化 */
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
  height: 44px;
  margin: 0;
  font-size: 16px;
  border-radius: 6px;
  font-weight: 500;
}

.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}

@media (min-width: 769px) {
  .mobile-action-bar,
  .mobile-coupons-list {
    display: none !important;
  }
}
</style>

