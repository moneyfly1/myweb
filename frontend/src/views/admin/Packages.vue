<template>
  <div class="list-container packages-admin-container">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span class="header-title">套餐列表</span>
          <div class="header-actions">
            <el-button @click="showCustomPackageDialog" class="custom-package-btn" :icon="Setting">
              <span class="btn-text">自定义套餐设置</span>
            </el-button>
            <el-button type="primary" @click="showAddDialog" class="add-package-btn" :icon="Plus">
              <span class="btn-text">添加套餐</span>
            </el-button>
          </div>
        </div>
      </template>
      <div class="search-section list-filters desktop-only">
        <el-form :inline="true" :model="searchForm" class="list-filter-form">
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
            <el-button type="primary" @click="handleSearch" :icon="Search">
              搜索
            </el-button>
            <el-button @click="resetSearch" :icon="Refresh">
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
      <ResponsiveDataView
        :data="packages"
        :fields="mobilePackageFields"
        :loading="loading"
        title-field="name"
        empty-title="暂无套餐数据"
        empty-description="可添加新套餐或调整筛选条件"
      >
        <template #table>
          <div class="table-wrapper">
            <el-table
              :data="packages"
              v-loading="loading"
              class="packages-table"
              stripe
              border
            >
              <template #empty>
                <EmptyState
                  title="暂无套餐数据"
                  description="可添加新套餐或调整筛选条件"
                  action-text="添加套餐"
                  :loading="loading"
                  @action="showAddDialog"
                />
              </template>
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
        </template>
        <template #header="{ item }">
          <div class="mobile-package-header">
            <span>{{ item.name }}</span>
            <el-tag :type="item.is_active ? 'success' : 'danger'" size="small">
              {{ item.is_active ? '启用' : '禁用' }}
            </el-tag>
          </div>
        </template>
        <template #empty>
          <EmptyState
            title="暂无套餐数据"
            description="可添加新套餐或调整筛选条件"
            action-text="添加套餐"
            :loading="loading"
            @action="showAddDialog"
          />
        </template>
        <template #actions="{ item }">
          <el-button
            type="primary"
            size="small"
            @click="editPackage(item)"
          >
            编辑
          </el-button>
          <el-button
            type="danger"
            size="small"
            @click="deletePackage(item.id)"
          >
            删除
          </el-button>
        </template>
      </ResponsiveDataView>
      <PaginationBar
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.size"
        :page-sizes="[10, 20, 50, 100]"
        :total="pagination.total"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </el-card>
    <AppDrawer
      v-model="dialogVisible"
      :title="isEdit ? '编辑套餐' : '添加套餐'"
      size="500px"
      mobile-size="100%"
      :loading="submitLoading"
      class="package-drawer"
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
            class="full-width-control"
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
            class="full-width-control"
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
            class="full-width-control"
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
          <el-select v-model="form.is_active" placeholder="选择状态" class="full-width-control">
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
          <div class="form-tip">
            <span v-if="!isDescriptionManuallyEdited">描述将根据价格、时长、设备数量自动生成</span>
            <span v-else>已手动编辑，将使用您输入的描述</span>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <FormActionBar
          :loading="submitLoading"
          :submit-text="isEdit ? '更新' : '添加'"
          :sticky="false"
          @cancel="dialogVisible = false"
          @submit="handleSubmit"
        />
      </template>
    </AppDrawer>

    <!-- 自定义套餐设置对话框 -->
    <AppDrawer
      v-model="customPackageDialogVisible"
      title="自定义套餐设置"
      size="600px"
      mobile-size="100%"
      :loading="customPackageLoading"
      class="package-drawer custom-package-drawer"
    >
      <el-form
        ref="customPackageFormRef"
        :model="customPackageForm"
        :label-width="isMobile ? '0' : '160px'"
        :label-position="isMobile ? 'top' : 'right'"
      >
        <el-form-item :label="isMobile ? '' : '启用自定义套餐'" prop="enabled">
          <template v-if="isMobile">
            <div class="mobile-label">启用自定义套餐</div>
          </template>
          <el-switch v-model="customPackageForm.enabled" :size="isMobile ? 'large' : 'default'" />
          <div class="form-tip">启用后，用户可以自定义设备数量和购买月数</div>
        </el-form-item>

        <el-form-item :label="isMobile ? '' : '每设备每年价格 (元)'" prop="price_per_device_year">
          <template v-if="isMobile">
            <div class="mobile-label">每设备每年价格 (元) <span class="required">*</span></div>
          </template>
          <el-input-number
            v-model="customPackageForm.price_per_device_year"
            :min="0"
            :precision="2"
            :step="1"
            placeholder="例如: 40.00"
            :size="isMobile ? 'large' : 'default'"
            class="full-width-control"
          />
          <div class="form-tip">例如 40 表示每台设备每年 40 元</div>
        </el-form-item>

        <el-form-item :label="isMobile ? '' : '最少设备数'" prop="min_devices">
          <template v-if="isMobile">
            <div class="mobile-label">最少设备数 <span class="required">*</span></div>
          </template>
          <el-input-number
            v-model="customPackageForm.min_devices"
            :min="1"
            :precision="0"
            placeholder="例如: 5"
            :size="isMobile ? 'large' : 'default'"
            class="full-width-control"
          />
        </el-form-item>

        <el-form-item :label="isMobile ? '' : '最多设备数'" prop="max_devices">
          <template v-if="isMobile">
            <div class="mobile-label">最多设备数 <span class="required">*</span></div>
          </template>
          <el-input-number
            v-model="customPackageForm.max_devices"
            :min="1"
            :precision="0"
            placeholder="例如: 100"
            :size="isMobile ? 'large' : 'default'"
            class="full-width-control"
          />
        </el-form-item>

        <el-form-item :label="isMobile ? '' : '最少购买月数'" prop="min_months">
          <template v-if="isMobile">
            <div class="mobile-label">最少购买月数 <span class="required">*</span></div>
          </template>
          <el-input-number
            v-model="customPackageForm.min_months"
            :min="1"
            :precision="0"
            placeholder="例如: 6"
            :size="isMobile ? 'large' : 'default'"
            class="full-width-control"
          />
        </el-form-item>

        <el-form-item :label="isMobile ? '' : '时长折扣配置'" prop="duration_discounts">
          <template v-if="isMobile">
            <div class="mobile-label">时长折扣配置</div>
          </template>
          <div class="discount-config">
            <div
              v-for="(discount, index) in customPackageForm.duration_discounts"
              :key="index"
              class="discount-item"
            >
              <div class="discount-input-group">
                <el-input-number
                  v-model="discount.months"
                  :min="1"
                  :precision="0"
                  placeholder="月数"
                  :size="isMobile ? 'default' : 'default'"
                  class="discount-number-input"
                />
                <span class="discount-separator">个月</span>
              </div>
              <div class="discount-input-group">
                <el-input-number
                  v-model="discount.discount"
                  :min="0"
                  :max="100"
                  :precision="1"
                  placeholder="折扣"
                  :size="isMobile ? 'default' : 'default'"
                  class="discount-number-input"
                />
                <span class="discount-separator">% 优惠</span>
              </div>
              <el-button
                type="danger"
                :size="isMobile ? 'default' : 'small'"
                @click="removeDiscount(index)"
                :icon="Delete"
                class="discount-delete-btn"
              >
                删除
              </el-button>
            </div>
            <el-button
              type="primary"
              :size="isMobile ? 'default' : 'small'"
              @click="addDiscount"
              :icon="Plus"
              class="discount-add-btn"
            >
              添加折扣
            </el-button>
          </div>
          <div class="form-tip">例如: 购买12个月打10%折扣，购买24个月打20%折扣</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <FormActionBar
          :loading="customPackageLoading"
          submit-text="保存设置"
          :sticky="false"
          @cancel="customPackageDialogVisible = false"
          @submit="saveCustomPackageSettings"
        />
      </template>
    </AppDrawer>
  </div>
</template>
<script>
import { computed, ref, reactive, onMounted, watch } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { Plus, HomeFilled, Search, Refresh, Setting, Delete } from '@element-plus/icons-vue'
import { adminAPI, configAPI } from '@/utils/api'
import { useMobile } from '@/composables/useMobile'
import AppDrawer from '@/components/AppDrawer.vue'
import EmptyState from '@/components/EmptyState.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import { confirmDelete } from '@/utils/confirmAction'
export default {
  name: 'AdminPackages',
  components: {
    Plus,
    HomeFilled,
    Search,
    Refresh,
    Setting,
    Delete,
    AppDrawer,
    EmptyState,
    FormActionBar,
    PaginationBar,
    ResponsiveDataView
  },
  setup() {
    const loading = ref(false)
    const submitLoading = ref(false)
    const dialogVisible = ref(false)
    const isEdit = ref(false)
    const formRef = ref()
    const packages = ref([])
    const isMobile = useMobile()

    // 自定义套餐相关
    const customPackageDialogVisible = ref(false)
    const customPackageLoading = ref(false)
    const customPackageFormRef = ref()
    const customPackageForm = reactive({
      enabled: false,
      price_per_device_year: 40,
      min_devices: 5,
      max_devices: 100,
      min_months: 6,
      duration_discounts: [
        { months: 6, discount: 0 },
        { months: 12, discount: 10 },
        { months: 24, discount: 20 }
      ]
    })

    const searchForm = reactive({
      name: '',
      status: ''
    })
    const pagination = reactive({
      page: 1,
      size: 10,
      total: 0
    })
    const mobilePackageFields = computed(() => [
      { key: 'id', label: 'ID', formatter: value => `#${value}` },
      { key: 'price', label: '价格', formatter: value => `¥${value}` },
      { key: 'duration_days', label: '时长', formatter: value => `${value} 天` },
      { key: 'device_limit', label: '设备限制', formatter: value => value },
      {
        key: 'is_recommended',
        label: '推荐',
        type: 'tag',
        tagType: value => value ? 'success' : 'info',
        formatter: value => value ? '是' : '否'
      },
      {
        key: 'is_active',
        label: '状态',
        type: 'tag',
        tagType: value => value ? 'success' : 'danger',
        formatter: value => value ? '启用' : '禁用'
      }
    ])
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
        await confirmDelete('套餐', 1, {
          message: '确定要删除这个套餐吗？'
        })
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

    // 自定义套餐相关方法
    const showCustomPackageDialog = async () => {
      await loadCustomPackageSettings()
      customPackageDialogVisible.value = true
    }

    const loadCustomPackageSettings = async () => {
      try {
        const response = await configAPI.getSystemConfigs({ category: 'custom_package' })
        if (response.data && response.data.success) {
          const configs = response.data.data || []
          const configMap = {}
          configs.forEach(config => {
            configMap[config.key] = config.value
          })

          // 填充表单
          customPackageForm.enabled = configMap.custom_package_enabled === 'true'
          customPackageForm.price_per_device_year = parseFloat(configMap.custom_package_price_per_device_year || 40)
          customPackageForm.min_devices = parseInt(configMap.custom_package_min_devices || 5)
          customPackageForm.max_devices = parseInt(configMap.custom_package_max_devices || 100)
          customPackageForm.min_months = parseInt(configMap.custom_package_min_months || 6)

          // 解析折扣配置
          if (configMap.custom_package_duration_discounts) {
            try {
              const discounts = JSON.parse(configMap.custom_package_duration_discounts)
              if (Array.isArray(discounts) && discounts.length > 0) {
                customPackageForm.duration_discounts = discounts
              }
            } catch (e) {
              console.error('解析折扣配置失败:', e)
            }
          }
        }
      } catch (error) {
        console.error('加载自定义套餐设置失败:', error)
      }
    }

    const saveCustomPackageSettings = async () => {
      customPackageLoading.value = true
      try {
        const settings = {
          custom_package_enabled: customPackageForm.enabled.toString(),
          custom_package_price_per_device_year: customPackageForm.price_per_device_year.toString(),
          custom_package_min_devices: customPackageForm.min_devices.toString(),
          custom_package_max_devices: customPackageForm.max_devices.toString(),
          custom_package_min_months: customPackageForm.min_months.toString(),
          custom_package_duration_discounts: JSON.stringify(customPackageForm.duration_discounts)
        }

        // 批量更新配置
        for (const [key, value] of Object.entries(settings)) {
          await configAPI.updateSystemConfig(key, {
            key: key,
            value: value,
            category: 'custom_package',
            type: key === 'custom_package_enabled' ? 'boolean' :
                  key === 'custom_package_duration_discounts' ? 'json' : 'string',
            display_name: getDisplayName(key),
            is_public: true
          })
        }

        ElMessage.success('自定义套餐设置保存成功')
        customPackageDialogVisible.value = false
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '保存失败')
      } finally {
        customPackageLoading.value = false
      }
    }

    const getDisplayName = (key) => {
      const displayNames = {
        custom_package_enabled: '启用自定义套餐',
        custom_package_price_per_device_year: '每设备每年价格',
        custom_package_min_devices: '最少设备数',
        custom_package_max_devices: '最多设备数',
        custom_package_min_months: '最少购买月数',
        custom_package_duration_discounts: '时长折扣配置'
      }
      return displayNames[key] || key
    }

    const addDiscount = () => {
      customPackageForm.duration_discounts.push({ months: 6, discount: 0 })
    }

    const removeDiscount = (index) => {
      if (customPackageForm.duration_discounts.length > 1) {
        customPackageForm.duration_discounts.splice(index, 1)
      } else {
        ElMessage.warning('至少保留一个折扣配置')
      }
    }

    onMounted(() => {
      fetchPackages()
    })
    return {
      isMobile,
      loading,
      submitLoading,
      dialogVisible,
      isEdit,
      formRef,
      packages,
      mobilePackageFields,
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
      handleDescriptionInput,
      // 自定义套餐相关
      customPackageDialogVisible,
      customPackageLoading,
      customPackageFormRef,
      customPackageForm,
      showCustomPackageDialog,
      saveCustomPackageSettings,
      addDiscount,
      removeDiscount
    }
  }
}
</script>
<style scoped lang="scss">
.packages-admin-container {
  padding: 20px;
}

.packages-table,
.full-width-control {
  width: 100%;
}

.status-select-item {
  min-width: 0;
}

.status-select {
  width: 100%;
  min-width: 0;
}

.mobile-package-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
  span {
    min-width: 0;
    word-break: break-word;
  }
}
.card-header {
  .header-actions {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;

    .el-button {
      .btn-text {
        margin-left: 4px;
      }
    }
  }
}
.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 5px;
  line-height: 1.5;
}
.discount-config {
  width: 100%;

  .discount-item {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 10px;
    flex-wrap: wrap;

    .discount-input-group {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .discount-separator {
      color: #606266;
      font-size: 14px;
      white-space: nowrap;
    }

    .discount-delete-btn {
      flex-shrink: 0;
    }
  }

  .discount-add-btn {
    width: auto;
    margin-top: 10px;
  }

  .discount-number-input {
    width: 120px;
  }
}

@media (max-width: 768px) {
  .packages-admin-container {
    padding: 12px;
  }
  .card-header {
    .header-title {
      font-size: 1.125rem;
    }

    .header-actions {
      .custom-package-btn {
        order: 2;
      }

      .add-package-btn {
        order: 1;
      }
    }
  }
  .discount-config {
    .discount-item {
      flex-direction: column;
      align-items: stretch;
      gap: 12px;
      padding: 16px;
      background: #f5f7fa;
      border-radius: 8px;
      margin-bottom: 12px;

      .discount-input-group {
        display: flex;
        align-items: center;
        gap: 8px;
        width: 100%;

        .el-input-number,
        .discount-number-input {
          flex: 1;
          width: 100%;
        }

        .discount-separator {
          flex-shrink: 0;
          font-size: 14px;
          color: #606266;
        }
      }

      .discount-delete-btn {
        width: 100%;
        min-height: 44px;
        font-size: 15px;
        touch-action: manipulation;
      }

      :deep(.el-input-number) {
        width: 100% !important;

        .el-input__wrapper {
          width: 100%;
          padding: 10px 12px;
        }
      }
    }

    .discount-add-btn {
      width: 100%;
      min-height: 44px;
      font-size: 15px;
      margin-top: 10px;
      touch-action: manipulation;
    }
  }

  .package-drawer {
    :deep(.el-form) {
      .el-form-item {
        margin-bottom: 20px;

        .el-form-item__content {
          margin-left: 0 !important;
        }
      }

      .el-input-number {
        width: 100%;

        .el-input__wrapper {
          padding: 10px 12px;
          min-height: 44px;
        }

        .el-input__inner {
          font-size: 16px !important;
        }

        .el-input-number__decrease,
        .el-input-number__increase {
          width: 36px;
          height: 44px;
        }
      }

      .el-switch {
        height: 28px;

        &.is-checked .el-switch__core {
          background-color: #409eff;
        }
      }
    }

    .form-tip {
      font-size: 13px;
      color: #909399;
      margin-top: 6px;
      line-height: 1.6;
    }

    .mobile-label {
      font-size: 15px;
      font-weight: 600;
      color: #303133;
      margin-bottom: 8px;
      display: block;
      line-height: 1.5;

      .required {
        color: #f56c6c;
        margin-left: 2px;
      }
    }
  }

  .search-section.desktop-only {
    display: none;
  }
  .packages-admin-container :deep(.mobile-card-actions .el-button) {
    min-height: 44px;
    touch-action: manipulation;
  }
}
@media (min-width: 769px) {
  .mobile-search-section {
    display: none !important;
  }
}
.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}
.packages-admin-container :deep(.el-input__wrapper) {
  border-radius: 6px !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
}
.packages-admin-container :deep(.el-input__inner) {
  border-radius: 6px !important;
  border: none !important;
  box-shadow: none !important;
  background-color: transparent !important;
}
.packages-admin-container :deep(.el-input__wrapper:hover) {
  border-color: #c0c4cc !important;
  box-shadow: none !important;
}
.packages-admin-container :deep(.el-input__wrapper.is-focus) {
  border-color: #1677ff !important;
  box-shadow: none !important;
}
</style> 
