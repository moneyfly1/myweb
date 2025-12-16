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

      <!-- 移动端卡片式列表 -->
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

      <!-- 移动端空状态 -->
      <div class="mobile-card-list" v-if="packages.length === 0 && !loading && isMobile">
        <div class="empty-state">
          <i class="el-icon-goods"></i>
          <p>暂无套餐数据</p>
        </div>
      </div>

      <!-- 分页 -->
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

    <!-- 添加/编辑套餐对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑套餐' : '添加套餐'"
      :width="isMobile ? '95%' : '600px'"
      class="package-form-dialog"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        :label-width="isMobile ? '90px' : '100px'"
      >
        <el-form-item label="套餐名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入套餐名称" />
        </el-form-item>
        
        <el-form-item label="价格" prop="price">
          <el-input-number
            v-model="form.price"
            :min="0"
            :precision="2"
            :step="0.01"
            placeholder="请输入价格"
          />
        </el-form-item>
        
        <el-form-item label="时长(天)" prop="duration_days">
          <el-input-number
            v-model="form.duration_days"
            :min="1"
            :precision="0"
            placeholder="请输入时长"
          />
        </el-form-item>
        
        <el-form-item label="设备限制" prop="device_limit">
          <el-input-number
            v-model="form.device_limit"
            :min="0"
            :precision="0"
            placeholder="请输入设备限制（0表示不限制）"
          />
        </el-form-item>
        
        <el-form-item label="推荐套餐" prop="is_recommended">
          <el-switch v-model="form.is_recommended" />
        </el-form-item>
        
        <el-form-item label="状态" prop="is_active">
          <el-select v-model="form.is_active" placeholder="选择状态">
            <el-option label="启用" :value="true" />
            <el-option label="禁用" :value="false" />
          </el-select>
        </el-form-item>
        
        <el-form-item label="描述" prop="description">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="3"
            placeholder="请输入套餐描述"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer-buttons">
          <el-button @click="dialogVisible = false" class="mobile-action-btn">取消</el-button>
          <el-button
            type="primary"
            @click="handleSubmit"
            :loading="submitLoading"
           class="mobile-action-btn">
            {{ isEdit ? '更新' : '添加' }}
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
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
      name: '',
      price: 0,
      duration_days: 30,
      device_limit: 1,
      is_recommended: false,
      is_active: true,
      description: ''
    })

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

    // 获取套餐列表
    const fetchPackages = async () => {
      loading.value = true
      try {
        const params = {
          page: pagination.page,
          size: pagination.size,
          ...searchForm
        }
        const response = await adminAPI.getPackages(params)
        const packageList = response.data.data?.packages || response.data.items || []
        // 确保 is_active 和 is_recommended 是布尔值
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

    // 搜索
    const handleSearch = () => {
      pagination.page = 1
      fetchPackages()
    }

    // 重置搜索
    const resetSearch = () => {
      Object.assign(searchForm, {
        name: '',
        status: ''
      })
      pagination.page = 1
      fetchPackages()
    }

    // 分页处理
    const handleSizeChange = (size) => {
      pagination.size = size
      pagination.page = 1
      fetchPackages()
    }

    const handleCurrentChange = (page) => {
      pagination.page = page
      fetchPackages()
    }

    // 显示添加对话框
    const showAddDialog = () => {
      isEdit.value = false
      resetForm()
      dialogVisible.value = true
    }

    // 编辑套餐
    const editPackage = (packageData) => {
      isEdit.value = true
      // 确保 is_active 和 is_recommended 是布尔值
      const data = {
        ...packageData,
        is_active: packageData.is_active === true || packageData.is_active === 1 || packageData.is_active === '1',
        is_recommended: packageData.is_recommended === true || packageData.is_recommended === 1 || packageData.is_recommended === '1'
      }
      Object.assign(form, data)
      dialogVisible.value = true
    }

    // 重置表单
    const resetForm = () => {
      Object.assign(form, {
        name: '',
        price: 0,
        duration_days: 30,
        device_limit: 1,
        is_recommended: false,
        is_active: true,
        description: ''
      })
      if (formRef.value) {
        formRef.value.resetFields()
      }
    }

    // 提交表单
    const handleSubmit = async () => {
      if (!formRef.value) return

      try {
        await formRef.value.validate()
        submitLoading.value = true

        if (isEdit.value) {
          // 构建更新数据，排除 id 字段，确保所有字段都有值
          const updateData = {
            name: form.name,
            description: form.description || '',
            price: form.price,
            duration_days: form.duration_days,
            device_limit: form.device_limit,
            is_active: form.is_active,
            is_recommended: form.is_recommended !== undefined ? form.is_recommended : false
          }
          // 如果表单中有这些字段，也包含它们
          if (form.bandwidth_limit !== undefined) {
            updateData.bandwidth_limit = form.bandwidth_limit || null
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

    // 删除套餐
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
        
        // 检查响应是否成功
        if (response.data && response.data.success !== false) {
          ElMessage.success(response.data.message || '套餐删除成功')
          // 立即从列表中移除，避免等待刷新
          const index = packages.value.findIndex(pkg => pkg.id === id)
          if (index !== -1) {
            packages.value.splice(index, 1)
            // 如果删除后当前页没有数据了，且不是第一页，则返回上一页
            if (packages.value.length === 0 && pagination.page > 1) {
              pagination.page--
            }
          }
          // 刷新列表以确保数据同步
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
      handleSearch,
      resetSearch,
      handleSizeChange,
      handleCurrentChange,
      showAddDialog,
      editPackage,
      handleSubmit,
      deletePackage
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
  
  // 状态筛选框加长
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

/* 移动端样式 */
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
  
  // mobile-action-bar 和 mobile-search-section 样式已统一在 list-common.scss 中定义
  // 这里不再重复定义，使用统一样式
  
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
    :deep(.el-dialog__body) {
      padding: 16px;
      max-height: calc(100vh - 200px);
      overflow-y: auto;
      -webkit-overflow-scrolling: touch;
    }
    
    :deep(.el-form-item) {
      margin-bottom: 20px;
    }
    
    :deep(.el-form-item__label) {
      font-size: 14px;
      padding-bottom: 8px;
    }
  }
}

/* 桌面端隐藏移动端元素 */
@media (min-width: 769px) {
  .mobile-search-section,
  .mobile-card-list {
    display: none !important;
  }
}

/* 移动端隐藏桌面端元素 */
.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}

/* 移除所有输入框的圆角和阴影效果，设置为简单长方形 */
/* 但保留手机端搜索框的样式 */
:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
}

// mobile-action-bar 样式已统一在 list-common.scss 中定义
// 这里不再重复定义，使用统一样式

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