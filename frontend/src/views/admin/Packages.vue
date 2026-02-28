<template>
  <div class="list-container packages-admin-container">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>套餐列表</span>
          <el-button type="primary" @click="showAddDialog" class="add-package-btn">
            <el-icon><Plus /></el-icon>
            <span class="desktop-only">添加套餐</span>
          </el-button>
        </div>
      </template>
      <div class="search-section desktop-only">
        <el-form :inline="true" :model="searchForm">
          <el-form-item label="套餐名称">
            <el-input
              v-model="searchForm.name"
              placeholder="搜索套餐名称"
              clearable
            />
          </el-form-item>
          <el-form-item label="状态" class="status-select-item">
            <el-select v-model="searchForm.status" placeholder="选择状态" clearable class="status-select">
              <el-option label="启用" value="active" />
              <el-option label="禁用" value="inactive" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSearch">
              <i class="el-icon-search"></i>
              搜索
            </el-button>
            <el-button @click="resetSearch">
              <i class="el-icon-refresh"></i>
              重置
            </el-button>
          </el-form-item>
        </el-form>
      </div>
      <div class="mobile-action-bar">
        <div class="mobile-search-section">
          <div class="search-input-wrapper">
            <el-input
              v-model="searchForm.name"
              placeholder="搜索套餐名称"
              clearable
              class="mobile-search-input"
              @keyup.enter="handleSearch"
            />
            <el-button 
              @click="handleSearch" 
              class="search-button-inside"
              type="default"
              plain
            >
              <el-icon><Search /></el-icon>
            </el-button>
          </div>
        </div>
        <div class="mobile-filter-buttons">
          <el-select 
            v-model="searchForm.status" 
            placeholder="选择状态" 
            clearable
            class="mobile-status-select"
          >
            <el-option label="启用" value="active" />
            <el-option label="禁用" value="inactive" />
          </el-select>
          <el-button 
            @click="resetSearch" 
            type="default"
            plain
          >
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
        </div>
      </div>
      <div class="table-wrapper desktop-only">
        <el-table
          :data="packages"
          v-loading="loading"
          style="width: 100%"
          stripe
          border
        >
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="套餐名称" />
        <el-table-column prop="price" label="价格">
          <template #default="{ row }">
            ¥{{ row.price }}
          </template>
        </el-table-column>
        <el-table-column prop="duration_days" label="时长">
          <template #default="{ row }">
            {{ row.duration_days }} 天
          </template>
        </el-table-column>
        <el-table-column prop="device_limit" label="设备限制" />
        <el-table-column prop="is_recommended" label="推荐">
          <template #default="{ row }">
            <el-tag :type="row.is_recommended ? 'success' : 'info'">
              {{ row.is_recommended ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="is_active" label="状态">
          <template #default="{ row }">
            <el-tag :type="row.is_active ? 'success' : 'danger'">
              {{ row.is_active ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-button
                type="primary"
                size="small"
                @click="editPackage(row)"
              >
                编辑
              </el-button>
              <el-button
                type="danger"
                size="small"
                @click="deletePackage(row.id)"
              >
                删除
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      </div>
      <div class="mobile-card-list" v-if="packages.length > 0 && isMobile">
        <div 
          v-for="pkg in packages" 
          :key="pkg.id"
          class="mobile-card"
        >
          <div class="card-row">
            <span class="label">ID</span>
            <span class="value">#{{ pkg.id }}</span>
          </div>
          <div class="card-row">
            <span class="label">套餐名称</span>
            <span class="value">{{ pkg.name }}</span>
          </div>
          <div class="card-row">
            <span class="label">价格</span>
            <span class="value">¥{{ pkg.price }}</span>
          </div>
          <div class="card-row">
            <span class="label">时长</span>
            <span class="value">{{ pkg.duration_days }} 天</span>
          </div>
          <div class="card-row">
            <span class="label">设备限制</span>
            <span class="value">{{ pkg.device_limit }}</span>
          </div>
          <div class="card-row">
            <span class="label">推荐</span>
            <span class="value">
              <el-tag :type="pkg.is_recommended ? 'success' : 'info'">
                {{ pkg.is_recommended ? '是' : '否' }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">状态</span>
            <span class="value">
              <el-tag :type="pkg.is_active ? 'success' : 'danger'">
                {{ pkg.is_active ? '启用' : '禁用' }}
              </el-tag>
            </span>
          </div>
          <div class="card-actions">
            <el-button
              type="primary"
              @click="editPackage(pkg)"
              class="mobile-action-btn"
            >
              编辑
            </el-button>
            <el-button
              type="danger"
              @click="deletePackage(pkg.id)"
              class="mobile-action-btn"
            >
              删除
            </el-button>
          </div>
        </div>
      </div>
      <div class="mobile-card-list" v-if="packages.length === 0 && !loading && isMobile">
        <div class="empty-state">
          <i class="el-icon-goods"></i>
          <p>暂无套餐数据</p>
        </div>
      </div>
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.size"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
    <el-drawer
      v-model="dialogVisible"
      :title="isEdit ? '编辑套餐' : '添加套餐'"
      :size="isMobile ? '100%' : '500px'"
      direction="rtl"
      :class="{ 'mobile-dialog': isMobile }"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        :label-width="isMobile ? '0' : '100px'"
        :label-position="isMobile ? 'top' : 'right'"
      >
        <el-form-item label="套餐名称" prop="name">
          <template v-if="isMobile">
            <div class="mobile-label">套餐名称 <span class="required">*</span></div>
          </template>
          <el-input v-model="form.name" placeholder="请输入套餐名称" />
        </el-form-item>
        <el-form-item label="价格" prop="price">
          <template v-if="isMobile">
            <div class="mobile-label">价格 <span class="required">*</span></div>
          </template>
          <el-input-number
            v-model="form.price"
            :min="0"
            :precision="2"
            :step="0.01"
            placeholder="请输入价格"
            @change="autoGenerateDescription"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="时长(天)" prop="duration_days">
          <template v-if="isMobile">
            <div class="mobile-label">时长(天) <span class="required">*</span></div>
          </template>
          <el-input-number
            v-model="form.duration_days"
            :min="1"
            :precision="0"
            placeholder="请输入时长"
            @change="autoGenerateDescription"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="设备限制" prop="device_limit">
          <template v-if="isMobile">
            <div class="mobile-label">设备限制 <span class="required">*</span></div>
          </template>
          <el-input-number
            v-model="form.device_limit"
            :min="0"
            :precision="0"
            placeholder="请输入设备限制（0表示不限制）"
            @change="autoGenerateDescription"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="推荐套餐" prop="is_recommended">
          <template v-if="isMobile">
            <div class="mobile-label">推荐套餐</div>
          </template>
          <el-switch v-model="form.is_recommended" />
        </el-form-item>
        <el-form-item label="状态" prop="is_active">
          <template v-if="isMobile">
            <div class="mobile-label">状态 <span class="required">*</span></div>
          </template>
          <el-select v-model="form.is_active" placeholder="选择状态" style="width: 100%">
            <el-option label="启用" :value="true" />
            <el-option label="禁用" :value="false" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <template v-if="isMobile">
            <div class="mobile-label">描述</div>
          </template>
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="3"
            placeholder="自动生成描述，或手动输入自定义描述"
            @input="handleDescriptionInput"
          />
          <div style="margin-top: 5px; font-size: 12px; color: #909399;">
            <span v-if="!isDescriptionManuallyEdited">描述将根据价格、时长、设备数量自动生成</span>
            <span v-else>已手动编辑，将使用您输入的描述</span>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer-buttons" :class="{ 'mobile-footer': isMobile }">
          <el-button @click="dialogVisible = false" :class="{ 'mobile-action-btn': isMobile }">取消</el-button>
          <el-button
            type="primary"
            @click="handleSubmit"
            :loading="submitLoading"
            :class="{ 'mobile-action-btn': isMobile }">
            {{ isEdit ? '更新' : '添加' }}
          </el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>
<script>
import { ref, reactive, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, HomeFilled, Search, Refresh } from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
export default {
  name: 'AdminPackages',
  components: {
    Plus,
    HomeFilled,
    Search,
    Refresh
  },
  setup() {
    const loading = ref(false)
    const submitLoading = ref(false)
    const dialogVisible = ref(false)
    const isEdit = ref(false)
    const formRef = ref()
    const packages = ref([])
    const isMobile = ref(window.innerWidth <= 768)
    const searchForm = reactive({
      name: '',
      status: ''
    })
    const pagination = reactive({
      page: 1,
      size: 20,
      total: 0
    })
    const form = reactive({
      id: null,
      name: '',
      price: 0,
      duration_days: 30,
      device_limit: 1,
      sort_order: 0,
      is_recommended: false,
      is_active: true,
      description: ''
    })
    const isDescriptionManuallyEdited = ref(false)
    const autoGeneratedDescription = ref('')
    const rules = {
      name: [
        { required: true, message: '请输入套餐名称', trigger: 'blur' }
      ],
      price: [
        { required: true, message: '请输入价格', trigger: 'blur' }
      ],
      duration_days: [
        { required: true, message: '请输入时长', trigger: 'blur' }
      ],
      device_limit: [
        { required: true, message: '请输入设备限制', trigger: 'blur' }
      ],
      is_active: [
        { required: true, message: '请选择状态', trigger: 'change' }
      ]
    }
    const fetchPackages = async () => {
      loading.value = true
      try {
        const params = {
          page: pagination.page,
          size: pagination.size,
          ...searchForm
        }
        const response = await adminAPI.getPackages(params)
        const packageList = response.data.data?.packages || []
        packages.value = packageList.map(pkg => ({
          ...pkg,
          is_active: pkg.is_active === true || pkg.is_active === 1 || pkg.is_active === '1',
          is_recommended: pkg.is_recommended === true || pkg.is_recommended === 1 || pkg.is_recommended === '1'
        }))
        pagination.total = response.data.data?.total || response.data.total || 0
      } catch (error) {
        ElMessage.error('获取套餐列表失败')
        } finally {
        loading.value = false
      }
    }
    const handleSearch = () => {
      pagination.page = 1
      fetchPackages()
    }
    const resetSearch = () => {
      Object.assign(searchForm, {
        name: '',
        status: ''
      })
      pagination.page = 1
      fetchPackages()
    }
    const handleSizeChange = (size) => {
      pagination.size = size
      pagination.page = 1
      fetchPackages()
    }
    const handleCurrentChange = (page) => {
      pagination.page = page
      fetchPackages()
    }
    const showAddDialog = () => {
      isEdit.value = false
      resetForm()
      dialogVisible.value = true
    }
    const editPackage = (packageData) => {
      isEdit.value = true
      let descriptionValue = ''
      if (packageData.description !== null && packageData.description !== undefined) {
        if (typeof packageData.description === 'object' && packageData.description !== null) {
          descriptionValue = packageData.description.String || packageData.description.string || ''
        } else {
          descriptionValue = String(packageData.description)
        }
      }
      const data = {
        ...packageData,
        is_active: packageData.is_active === true || packageData.is_active === 1 || packageData.is_active === '1',
        is_recommended: packageData.is_recommended === true || packageData.is_recommended === 1 || packageData.is_recommended === '1',
        description: descriptionValue
      }
      Object.assign(form, data)
      const autoGeneratedKeywords = ['解锁流媒体', '无限流量', '高速稳定节点', '7×24小时技术支持', '支持售后']
      const isAutoGenerated = descriptionValue && autoGeneratedKeywords.some(keyword => descriptionValue.includes(keyword))
      if (isAutoGenerated) {
        isDescriptionManuallyEdited.value = false
        autoGeneratedDescription.value = descriptionValue
      } else {
        isDescriptionManuallyEdited.value = true
        autoGeneratedDescription.value = ''
      }
      dialogVisible.value = true
    }
    const autoGenerateDescription = () => {
      if (isDescriptionManuallyEdited.value) {
        return
      }
      const features = []
      if (form.duration_days >= 365) {
        features.push(`有效期 ${Math.floor(form.duration_days / 365)} 年`)
      } else if (form.duration_days >= 30) {
        features.push(`有效期 ${Math.floor(form.duration_days / 30)} 个月`)
      } else {
        features.push(`有效期 ${form.duration_days} 天`)
      }
      if (form.device_limit === 0) {
        features.push('支持无限设备')
      } else if (form.device_limit === 1) {
        features.push('支持 1 个设备')
      } else {
        features.push(`支持 ${form.device_limit} 个设备`)
      }
      features.push('解锁流媒体')
      features.push('无限流量')
      features.push('高速稳定节点')
      features.push('7×24小时技术支持')
      features.push('支持售后')
      if (form.price > 0) {
        features.push(`价格 ¥${form.price.toFixed(2)}`)
      }
      const generatedDescription = features.join(' | ')
      autoGeneratedDescription.value = generatedDescription
      form.description = generatedDescription
    }
    const handleDescriptionInput = (value) => {
      if (value !== autoGeneratedDescription.value) {
        isDescriptionManuallyEdited.value = true
      }
    }
    const resetForm = () => {
      Object.assign(form, {
        id: null,
        name: '',
        price: 0,
        duration_days: 30,
        device_limit: 1,
        sort_order: 0,
        is_recommended: false,
        is_active: true,
        description: ''
      })
      isDescriptionManuallyEdited.value = false
      autoGeneratedDescription.value = ''
      if (formRef.value) {
        formRef.value.resetFields()
      }
      setTimeout(() => {
        autoGenerateDescription()
      }, 100)
    }
    const handleSubmit = async () => {
      if (!formRef.value) return
      try {
        await formRef.value.validate()
        submitLoading.value = true
        if (isEdit.value) {
          if (!form.id) {
            ElMessage.error('套餐ID缺失，无法更新')
            return
          }
          const updateData = {
            name: form.name,
            description: form.description !== undefined && form.description !== null ? String(form.description) : '',
            price: form.price,
            duration_days: form.duration_days,
            device_limit: form.device_limit,
            is_active: form.is_active,
            is_recommended: form.is_recommended !== undefined ? form.is_recommended : false
          }
          if (form.sort_order !== undefined) {
            updateData.sort_order = form.sort_order || 0
          }
          await adminAPI.updatePackage(form.id, updateData)
          ElMessage.success('套餐更新成功')
        } else {
          await adminAPI.createPackage(form)
          ElMessage.success('套餐添加成功')
        }
        dialogVisible.value = false
        fetchPackages()
      } catch (error) {
        if (error.response?.data?.message) {
          ElMessage.error(error.response.data.message)
        } else {
          ElMessage.error('操作失败')
        }
      } finally {
        submitLoading.value = false
      }
    }
    const deletePackage = async (id) => {
      try {
        await ElMessageBox.confirm(
          '确定要删除这个套餐吗？',
          '确认删除',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        const response = await adminAPI.deletePackage(id)
        if (response.data && response.data.success !== false) {
          ElMessage.success(response.data.message || '套餐删除成功')
          const index = packages.value.findIndex(pkg => pkg.id === id)
          if (index !== -1) {
            packages.value.splice(index, 1)
            if (packages.value.length === 0 && pagination.page > 1) {
              pagination.page--
            }
          }
          await fetchPackages()
        } else {
          const errorMsg = response.data?.message || '删除失败'
          ElMessage.error(errorMsg)
        }
      } catch (error) {
        if (error !== 'cancel') {
          const errorMsg = error.response?.data?.message || error.message || '删除失败'
          ElMessage.error(errorMsg)
        }
      }
    }
    const handleResize = () => {
      isMobile.value = window.innerWidth <= 768
    }
    onMounted(() => {
      fetchPackages()
      window.addEventListener('resize', handleResize)
    })
    onUnmounted(() => {
      window.removeEventListener('resize', handleResize)
    })
    return {
      isMobile,
      loading,
      submitLoading,
      dialogVisible,
      isEdit,
      formRef,
      packages,
      searchForm,
      pagination,
      form,
      rules,
      isDescriptionManuallyEdited,
      handleSearch,
      resetSearch,
      handleSizeChange,
      handleCurrentChange,
      showAddDialog,
      editPackage,
      handleSubmit,
      deletePackage,
      autoGenerateDescription,
      handleDescriptionInput
    }
  }
}
</script>
<style scoped lang="scss">
@use '@/styles/list-common.scss';
.packages-admin-container {
  padding: 20px;
}
.empty-state {
  text-align: center;
  padding: 3rem 1rem;
  color: #999;
  :is(i) {
    font-size: 3rem;
    margin-bottom: 1rem;
    display: block;
  }
  :is(p) {
    font-size: 0.9rem;
    margin: 0;
    line-height: 1.5;
  }
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.card-header h2 {
  margin: 0;
  color: #333;
  font-size: 1.5rem;
}
.search-section {
  margin-bottom: 20px;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
  :deep(.status-select-item) {
    .status-select {
      min-width: 180px;
      width: 180px;
      .el-input__wrapper {
        width: 100%;
      }
    }
  }
}
.pagination-section {
  margin-top: 20px;
  text-align: right;
}
@media (max-width: 768px) {
  .packages-admin-container {
    padding: 12px;
  }
  .card-header {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
    .add-package-btn {
      width: 100%;
      height: 44px;
      font-size: 16px;
    }
  }
  .search-section.desktop-only {
    display: none;
  }
  .mobile-card-list {
    margin-top: 16px;
    .mobile-card {
      background: #fff;
      border-radius: 8px;
      padding: 16px;
      margin-bottom: 12px;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      .card-row {
        display: flex;
        align-items: center;
        margin-bottom: 12px;
        padding-bottom: 12px;
        border-bottom: 1px solid #f0f0f0;
        &:last-of-type {
          border-bottom: none;
          margin-bottom: 0;
          padding-bottom: 0;
        }
        .label {
          flex: 0 0 90px;
          font-size: 14px;
          color: #666;
          font-weight: 500;
        }
        .value {
          flex: 1;
          font-size: 14px;
          color: #333;
          word-break: break-word;
        }
      }
      .card-actions {
        display: flex;
        gap: 8px;
        margin-top: 12px;
        padding-top: 12px;
        border-top: 1px solid #f0f0f0;
        .mobile-action-btn {
          flex: 1;
          height: 44px;
          font-size: 16px;
          margin: 0;
        }
      }
    }
    .empty-state {
      padding: 40px 20px;
      text-align: center;
    }
  }
  .package-form-dialog {
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
    :deep(.el-dialog__body) {
      padding: 16px;
      max-height: calc(100vh - 200px);
      overflow-y: auto;
      -webkit-overflow-scrolling: touch;
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
  }
}
@media (min-width: 769px) {
  .mobile-search-section,
  .mobile-card-list {
    display: none !important;
  }
}
.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}
:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
}
:deep(.el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background-color: transparent !important;
}
:deep(.el-input__wrapper:hover) {
  border-color: #c0c4cc !important;
  box-shadow: none !important;
}
:deep(.el-input__wrapper.is-focus) {
  border-color: #1677ff !important;
  box-shadow: none !important;
}
</style> 