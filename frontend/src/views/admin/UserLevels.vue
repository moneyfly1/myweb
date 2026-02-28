<template>
  <div class="user-levels-admin">
    <el-card>
      <template #header>
        <div class="card-header">
          <span class="card-title">用户等级管理</span>
          <div class="header-actions desktop-only">
            <el-select 
              v-model="statusFilter" 
              placeholder="状态筛选" 
              clearable 
              style="width: 150px;"
              @change="loadLevels"
            >
              <el-option label="全部" value="all" />
              <el-option label="启用" :value="true" />
              <el-option label="禁用" :value="false" />
            </el-select>
            <el-button type="primary" @click="showAddDialog" :icon="Plus" class="add-button">添加等级</el-button>
          </div>
        </div>
      </template>
      <div class="mobile-action-bar">
        <div class="mobile-filter-buttons">
          <el-button
            size="small"
            :type="statusFilter !== 'all' && statusFilter !== undefined && statusFilter !== null && statusFilter !== '' ? 'primary' : 'default'"
            plain
            @click="showStatusFilterDrawer = true"
          >
            <el-icon><Filter /></el-icon>
            {{ getStatusFilterText() }}
          </el-button>
          <el-button size="small" type="default" plain @click="resetStatusFilter">
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
        </div>
        <div class="mobile-action-buttons">
          <el-button 
            type="primary" 
            @click="showAddDialog"
            class="mobile-action-btn"
          >
            <el-icon><Plus /></el-icon>
            添加等级
          </el-button>
        </div>
      </div>
      <el-drawer
        v-model="showStatusFilterDrawer"
        title="状态筛选"
        :size="isMobile ? '85%' : '400px'"
        direction="rtl"
      >
        <div class="filter-drawer-content">
          <el-form label-width="100px">
            <el-form-item label="状态">
              <el-select 
                v-model="statusFilter" 
                placeholder="选择状态" 
                clearable 
                style="width: 100%;"
                @change="applyStatusFilter"
              >
                <el-option label="全部" value="all" />
                <el-option label="启用" :value="true" />
                <el-option label="禁用" :value="false" />
              </el-select>
            </el-form-item>
          </el-form>
          <div class="filter-drawer-actions">
            <el-button @click="resetStatusFilter" class="mobile-action-btn">重置</el-button>
            <el-button type="primary" @click="applyStatusFilter" class="mobile-action-btn">应用</el-button>
          </div>
        </div>
      </el-drawer>
      <el-table 
        :data="levels" 
        v-loading="loading"
        border
        style="width: 100%"
      >
        <el-table-column prop="level_name" label="等级名称" width="150">
          <template #default="scope">
            <div style="display: flex; align-items: center; gap: 8px;">
              <div 
                v-if="scope.row.color" 
                :style="{ 
                  width: '16px', 
                  height: '16px', 
                  borderRadius: '50%', 
                  backgroundColor: scope.row.color 
                }"
              ></div>
              <span :style="{ color: scope.row.color || '#333' }">{{ scope.row.level_name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="level_order" label="排序" width="80" align="center" />
        <el-table-column prop="min_consumption" label="最低消费" width="120" align="right">
          <template #default="scope">
            ¥{{ scope.row.min_consumption.toFixed(2) }}
          </template>
        </el-table-column>
        <el-table-column prop="discount_rate" label="折扣率" width="100" align="center">
          <template #default="scope">
            <el-tag :type="scope.row.discount_rate < 1 ? 'success' : 'info'">
              {{ (scope.row.discount_rate * 10).toFixed(1) }}折
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="user_count" label="用户数" width="100" align="center" />
        <el-table-column prop="is_active" label="状态" width="80" align="center">
          <template #default="scope">
            <el-tag :type="scope.row.is_active ? 'success' : 'danger'">
              {{ scope.row.is_active ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" align="center" fixed="right">
          <template #default="scope">
            <el-button type="primary" size="small" @click="editLevel(scope.row)">编辑</el-button>
            <el-button type="danger" size="small" @click="deleteLevel(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    <el-card class="usage-guide-card" style="margin-top: 20px;">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px;">
          <el-icon><InfoFilled /></el-icon>
          <span>用户等级系统使用说明</span>
        </div>
      </template>
      <div class="usage-guide-content">
        <div class="guide-section">
          <h4>📋 功能说明</h4>
          <ul>
            <li><strong>自动升级：</strong>用户累计消费达到等级要求时，系统会自动升级用户等级</li>
            <li><strong>等级折扣：</strong>不同等级享受不同的套餐折扣（如VIP 9折，100元套餐只需支付90元）</li>
            <li><strong>折扣叠加：</strong>等级折扣和优惠券折扣可以叠加使用，享受更多优惠</li>
            <li><strong>升级进度：</strong>用户可以在个人中心查看距离下一级的消费进度</li>
          </ul>
        </div>
        <div class="guide-section">
          <h4>👤 客户端显示位置</h4>
          <ul>
            <li><strong>用户仪表盘：</strong>在首页顶部显示当前等级（带颜色标识）</li>
            <li><strong>升级进度条：</strong>显示距离下一级还需消费的金额和进度百分比</li>
            <li><strong>订单支付：</strong>创建订单时自动应用等级折扣</li>
          </ul>
        </div>
        <div class="guide-section">
          <h4>⚙️ 配置建议</h4>
          <ul>
            <li><strong>等级排序：</strong>数字越小等级越高（1为最高等级）</li>
            <li><strong>最低消费：</strong>建议从低到高递增设置（如：0元、100元、500元）</li>
            <li><strong>折扣率：</strong>0.9表示9折（100元套餐只需支付90元），0.95表示95折，1.0表示无折扣</li>
            <li><strong>折扣计算：</strong>购买套餐时自动应用等级折扣，用户可清楚看到节省的金额</li>
          </ul>
        </div>
        <div class="guide-section">
          <h4>💡 使用示例</h4>
          <div class="example-box">
            <p><strong>示例配置：</strong></p>
            <ul>
              <li>普通会员：排序10，最低消费0元，折扣1.0（无折扣）</li>
              <li>VIP会员：排序5，最低消费100元，折扣0.95（95折，100元套餐只需支付95元）</li>
              <li>超级VIP：排序2，最低消费500元，折扣0.9（9折，100元套餐只需支付90元）</li>
            </ul>
            <p style="margin-top: 10px; color: #909399; font-size: 12px;">
              💡 用户累计消费达到100元时，自动从"普通会员"升级到"VIP会员"，享受95折优惠。购买套餐时系统会自动计算并显示折扣金额，提醒用户如何获取优惠价格。
            </p>
          </div>
        </div>
      </div>
    </el-card>
    <el-drawer
      v-model="showDialog"
      :title="editingLevel ? '编辑等级' : '添加等级'"
      :size="isMobile ? '100%' : '500px'"
      direction="rtl"
      :class="{ 'mobile-dialog': isMobile }"
    >
      <el-form 
        :model="levelForm" 
        :label-width="isMobile ? '0' : '120px'"
        :label-position="isMobile ? 'top' : 'right'"
        ref="levelFormRef"
      >
        <el-form-item label="等级名称" prop="level_name" :rules="[{ required: true, message: '请输入等级名称' }]">
          <template v-if="isMobile">
            <div class="mobile-label">等级名称 <span class="required">*</span></div>
          </template>
          <el-input v-model="levelForm.level_name" placeholder="如：VIP、超级VIP、钻石会员" />
        </el-form-item>
        <el-form-item label="等级排序" prop="level_order" :rules="[{ required: true, message: '请输入等级排序' }]">
          <template v-if="isMobile">
            <div class="mobile-label">等级排序 <span class="required">*</span></div>
          </template>
          <el-input-number 
            v-model="levelForm.level_order" 
            :min="1" 
            :max="100"
            placeholder="数字越小等级越高"
            style="width: 100%"
          />
          <div class="form-tip">数字越小，等级越高（1为最高等级）</div>
        </el-form-item>
        <el-form-item label="最低消费" prop="min_consumption" :rules="[{ required: true, message: '请输入最低消费' }]">
          <template v-if="isMobile">
            <div class="mobile-label">最低消费 <span class="required">*</span></div>
          </template>
          <el-input-number 
            v-model="levelForm.min_consumption" 
            :min="0" 
            :precision="2"
            placeholder="累计消费达到此金额可升级"
            style="width: 100%"
          />
          <div class="form-tip">用户累计消费达到此金额可升级到此等级（元）</div>
        </el-form-item>
        <el-form-item label="折扣率" prop="discount_rate">
          <template v-if="isMobile">
            <div class="mobile-label">折扣率</div>
          </template>
          <el-input-number 
            v-model="levelForm.discount_rate" 
            :min="0.1" 
            :max="1" 
            :step="0.05"
            :precision="2"
            placeholder="0.9表示9折"
            style="width: 100%"
          />
          <div class="form-tip">0.9表示9折，1.0表示无折扣</div>
        </el-form-item>
        <el-form-item label="等级颜色" prop="color">
          <template v-if="isMobile">
            <div class="mobile-label">等级颜色</div>
          </template>
          <el-color-picker v-model="levelForm.color" />
          <div class="form-tip">用于前端显示等级的颜色</div>
        </el-form-item>
        <el-form-item label="图标URL" prop="icon_url">
          <template v-if="isMobile">
            <div class="mobile-label">图标URL</div>
          </template>
          <el-input v-model="levelForm.icon_url" placeholder="等级图标URL（可选）" />
        </el-form-item>
        <el-form-item label="权益说明" prop="benefits">
          <template v-if="isMobile">
            <div class="mobile-label">权益说明</div>
          </template>
          <el-input 
            v-model="levelForm.benefits" 
            type="textarea" 
            :rows="4"
            placeholder='JSON格式，如：{"priority_support": true, "exclusive_nodes": true}'
            class="rectangular-input"
          />
        </el-form-item>
        <el-form-item label="是否启用" prop="is_active">
          <template v-if="isMobile">
            <div class="mobile-label">是否启用</div>
          </template>
          <el-switch v-model="levelForm.is_active" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer-buttons" :class="{ 'mobile-footer': isMobile }">
          <el-button @click="showDialog = false" :class="{ 'mobile-action-btn': isMobile }">取消</el-button>
          <el-button type="primary" @click="saveLevel" :loading="saving" :class="{ 'mobile-action-btn': isMobile }">保存</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>
<script setup>
import { ref, reactive, onMounted, onUnmounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, InfoFilled, Filter, Refresh } from '@element-plus/icons-vue'
import { userLevelAPI } from '@/utils/api'
const loading = ref(false)
const saving = ref(false)
const levels = ref([])
const showDialog = ref(false)
const editingLevel = ref(null)
const levelFormRef = ref(null)
const statusFilter = ref('all')
const isMobile = ref(window.innerWidth <= 768)
const showStatusFilterDrawer = ref(false)
const levelForm = reactive({
  level_name: '',
  level_order: 1,
  min_consumption: 0,
  discount_rate: 1.0,
  color: '#409eff',
  icon_url: '',
  benefits: '',
  is_active: true
})
const loadLevels = async () => {
  loading.value = true
  try {
    const filterValue = (statusFilter.value === 'all' || statusFilter.value === undefined || statusFilter.value === null || statusFilter.value === '') ? undefined : statusFilter.value
    const response = await userLevelAPI.getAllLevels(undefined, filterValue)
    if (process.env.NODE_ENV === 'development') {
    }
    let levelList = []
    if (response?.data) {
      if (response.data.data && response.data.data.levels) {
        levelList = response.data.data.levels
      } 
      else if (response.data.success && Array.isArray(response.data.data)) {
        levelList = response.data.data
      }
      else if (Array.isArray(response.data)) {
        levelList = response.data
      }
      else if (response.data.levels) {
        levelList = response.data.levels
      }
    }
    levels.value = levelList.map(level => ({
      ...level,
      is_active: level.is_active === true || level.is_active === 1 || level.is_active === '1'
    }))
  } catch (error) {
    if (process.env.NODE_ENV === 'development') {
      console.error('加载等级列表失败:', error)
    }
    const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '未知错误'
    ElMessage.error('加载等级列表失败: ' + errorMsg)
    levels.value = []
  } finally {
    loading.value = false
  }
}
const showAddDialog = () => {
  editingLevel.value = null
  resetForm()
  showDialog.value = true
}
const editLevel = (level) => {
  editingLevel.value = level
  let isActiveValue = level.is_active
  if (typeof isActiveValue === 'number') {
    isActiveValue = isActiveValue !== 0
  } else if (typeof isActiveValue === 'string') {
    isActiveValue = isActiveValue === 'true' || isActiveValue === '1'
  } else if (isActiveValue === null || isActiveValue === undefined) {
    isActiveValue = true // 默认启用
  }
  let iconUrl = ''
  if (level.icon_url) {
    if (typeof level.icon_url === 'string') {
      iconUrl = level.icon_url
    } else if (typeof level.icon_url === 'object' && level.icon_url !== null) {
      iconUrl = level.icon_url.String || level.icon_url.string || ''
    }
  }
  let benefits = ''
  if (level.benefits) {
    if (typeof level.benefits === 'string') {
      benefits = level.benefits
    } else if (typeof level.benefits === 'object' && level.benefits !== null) {
      benefits = level.benefits.String || level.benefits.string || ''
    }
  }
  Object.assign(levelForm, {
    level_name: level.level_name,
    level_order: level.level_order,
    min_consumption: level.min_consumption,
    discount_rate: level.discount_rate,
    color: level.color || '#409eff',
    icon_url: iconUrl,
    benefits: benefits,
    is_active: Boolean(isActiveValue)
  })
  showDialog.value = true
}
const resetForm = () => {
  Object.assign(levelForm, {
    level_name: '',
    level_order: 1,
    min_consumption: 0,
    discount_rate: 1.0,
    color: '#409eff',
    icon_url: '',
    benefits: '',
    is_active: true
  })
  if (levelFormRef.value) {
    levelFormRef.value.clearValidate()
  }
}
const saveLevel = async () => {
  if (!levelFormRef.value) return
  try {
    await levelFormRef.value.validate()
    saving.value = true
    const isActiveValue = Boolean(levelForm.is_active)
    const iconUrl = typeof levelForm.icon_url === 'string' ? levelForm.icon_url : ''
    const benefits = typeof levelForm.benefits === 'string' ? levelForm.benefits : ''
    const data = {
      level_name: levelForm.level_name,
      level_order: levelForm.level_order,
      min_consumption: levelForm.min_consumption,
      discount_rate: levelForm.discount_rate,
      color: levelForm.color,
      icon_url: iconUrl,  // 传递空字符串以清空字段，传递字符串以更新字段
      benefits: benefits, // 传递空字符串以清空字段，传递字符串以更新字段
      is_active: isActiveValue
    }
    if (process.env.NODE_ENV === 'development') {
    }
    let response
    if (editingLevel.value) {
      response = await userLevelAPI.updateLevel(editingLevel.value.id, data)
      if (response?.data?.success) {
        ElMessage.success('等级更新成功')
      } else {
        throw new Error(response?.data?.message || '更新失败')
      }
    } else {
      response = await userLevelAPI.createLevel(data)
      if (response?.data?.success) {
        ElMessage.success('等级创建成功')
      } else {
        throw new Error(response?.data?.message || '创建失败')
      }
    }
    showDialog.value = false
    await loadLevels()
  } catch (error) {
    if (error !== false) { // 表单验证失败会返回false
      if (process.env.NODE_ENV === 'development') {
        console.error('保存等级失败:', error)
        console.error('错误详情:', error.response?.data)
      }
      ElMessage.error('保存失败: ' + (error.response?.data?.message || error.message))
    }
  } finally {
    saving.value = false
  }
}
const deleteLevel = async (level) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除等级 "${level.level_name}" 吗？${level.user_count > 0 ? `（仍有 ${level.user_count} 个用户使用此等级）` : ''}`,
      '确认删除',
      { type: 'warning' }
    )
    await userLevelAPI.deleteLevel(level.id)
    ElMessage.success('删除成功')
    await loadLevels()
  } catch (error) {
    if (error !== 'cancel') {
      if (process.env.NODE_ENV === 'development') {
        console.error('删除等级失败:', error)
      }
      ElMessage.error('删除失败: ' + (error.response?.data?.message || error.message))
    }
  }
}
const getStatusFilterText = () => {
  if (statusFilter.value === true) return '启用'
  if (statusFilter.value === false) return '禁用'
  return '状态'
}
const resetStatusFilter = () => {
  statusFilter.value = 'all'
  showStatusFilterDrawer.value = false
  loadLevels()
}
const applyStatusFilter = () => {
  showStatusFilterDrawer.value = false
  loadLevels()
}
const handleResize = () => {
  isMobile.value = window.innerWidth <= 768
}
onMounted(() => {
  loadLevels()
  window.addEventListener('resize', handleResize)
})
onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>
<style scoped>
.user-levels-admin {
  padding: 20px;
}
.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  border: 1px solid #dcdfe6 !important;
  box-shadow: none !important;
  background: transparent !important;
  padding: 0 !important;
}
:deep(.el-input__wrapper:hover) {
  border-color: #c0c4cc !important;
}
:deep(.el-input__wrapper.is-focus) {
  border-color: #409eff !important;
  box-shadow: none !important;
}
:deep(.el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background: transparent !important;
  padding: 0 11px !important;
  height: 32px !important;
  line-height: 32px !important;
}
:deep(.el-textarea__inner) {
  border-radius: 0 !important;
  border: 1px solid #dcdfe6 !important;
  box-shadow: none !important;
  background: transparent !important;
}
:deep(.el-textarea__inner:hover) {
  border-color: #c0c4cc !important;
}
:deep(.el-textarea__inner:focus) {
  border-color: #409eff !important;
  box-shadow: none !important;
}
:deep(.el-input-number) {
  border-radius: 0 !important;
}
:deep(.el-input-number .el-input__wrapper) {
  border-radius: 0 !important;
  border: 1px solid #dcdfe6 !important;
  box-shadow: none !important;
  background: transparent !important;
  padding: 0 !important;
}
:deep(.el-input-number .el-input__wrapper:hover) {
  border-color: #c0c4cc !important;
}
:deep(.el-input-number .el-input__wrapper.is-focus) {
  border-color: #409eff !important;
  box-shadow: none !important;
}
:deep(.el-input-number .el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background: transparent !important;
  padding: 0 11px !important;
  height: 32px !important;
  line-height: 32px !important;
}
:deep(.el-select .el-input__wrapper) {
  border-radius: 0 !important;
  border: 1px solid #dcdfe6 !important;
  box-shadow: none !important;
  background: transparent !important;
  padding: 0 !important;
}
:deep(.el-select .el-input__wrapper:hover) {
  border-color: #c0c4cc !important;
}
:deep(.el-select .el-input__wrapper.is-focus) {
  border-color: #409eff !important;
  box-shadow: none !important;
}
:deep(.el-select .el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background: transparent !important;
  padding: 0 11px !important;
  height: 32px !important;
  line-height: 32px !important;
}
:deep(.el-dialog .el-input__wrapper) {
  border-radius: 0 !important;
  border: 1px solid #dcdfe6 !important;
  box-shadow: none !important;
  background: transparent !important;
  padding: 0 !important;
}
:deep(.el-dialog .el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background: transparent !important;
  padding: 0 11px !important;
  height: 32px !important;
  line-height: 32px !important;
}
:deep(.el-dialog .el-textarea__inner) {
  border-radius: 0 !important;
  border: 1px solid #dcdfe6 !important;
  box-shadow: none !important;
  background: transparent !important;
}
:deep(.el-dialog .el-input-number .el-input__wrapper) {
  border-radius: 0 !important;
  border: 1px solid #dcdfe6 !important;
  box-shadow: none !important;
  background: transparent !important;
  padding: 0 !important;
}
:deep(.el-dialog .el-input-number .el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background: transparent !important;
  padding: 0 11px !important;
  height: 32px !important;
  line-height: 32px !important;
}
:deep(.el-dialog .el-select .el-input__wrapper) {
  border-radius: 0 !important;
  border: 1px solid #dcdfe6 !important;
  box-shadow: none !important;
  background: transparent !important;
  padding: 0 !important;
}
:deep(.el-dialog .el-select .el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background: transparent !important;
  padding: 0 11px !important;
  height: 32px !important;
  line-height: 32px !important;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}
.card-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}
.add-button {
  flex-shrink: 0;
}
.usage-guide-card {
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
}
.usage-guide-content {
  line-height: 1.8;
}
.guide-section {
  margin-bottom: 20px;
  padding: 15px;
  background: white;
  border-radius: 8px;
  border-left: 4px solid #409eff;
}
.guide-section h4 {
  margin: 0 0 12px 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
}
.guide-section ul {
  margin: 0;
  padding-left: 20px;
}
.guide-section li {
  margin-bottom: 8px;
  color: #606266;
  font-size: 14px;
}
.guide-section li strong {
  color: #303133;
}
.example-box {
  background: #f5f7fa;
  padding: 15px;
  border-radius: 6px;
  margin-top: 10px;
}
.example-box p {
  margin: 0 0 10px 0;
  color: #303133;
  font-size: 14px;
}
.example-box ul {
  margin: 0;
  padding-left: 20px;
}
.example-box li {
  margin-bottom: 6px;
  color: #606266;
  font-size: 13px;
}
.mobile-action-bar {
  display: none;
  padding: 16px;
  box-sizing: border-box;
  background: #f5f7fa;
  border-radius: 8px;
  margin-bottom: 16px;
}
.mobile-filter-buttons {
  display: flex;
  flex-direction: row;
  gap: 10px;
  align-items: stretch;
  width: 100%;
  box-sizing: border-box;
  flex-wrap: nowrap;
  margin-bottom: 12px;
}
.mobile-filter-buttons .el-button {
  flex: 1;
  height: 40px;
  font-size: 14px;
  border-radius: 6px;
}
.mobile-action-buttons {
  width: 100%;
}
.mobile-action-btn {
  width: 100%;
  height: 44px;
  margin: 0;
  font-size: 16px;
  border-radius: 6px;
  font-weight: 500;
}
.filter-drawer-content {
  padding: 20px 0;
}
.filter-drawer-actions {
  display: flex;
  gap: 12px;
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid #f0f0f0;
}
.filter-drawer-actions .mobile-action-btn {
  flex: 1;
  height: 44px;
}
.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}
@media (max-width: 768px) {
  .user-levels-admin {
    padding: 10px;
  }
  .mobile-action-bar {
    display: block !important;
  }
  .usage-guide-card {
    :deep(.el-card__body) {
      padding: 15px;
    }
  }
  .guide-section {
    padding: 12px;
    margin-bottom: 15px;
    :is(h4) {
      font-size: 14px;
      margin-bottom: 10px;
    }
    :is(ul) {
      padding-left: 18px;
    }
    :is(li) {
      font-size: 13px;
      margin-bottom: 6px;
    }
  }
  :deep(.el-table) {
    font-size: 12px;
    .el-table__cell {
      padding: 8px 4px;
      word-break: break-word;
    }
    .el-table__header th {
      padding: 8px 4px;
      font-size: 12px;
      font-weight: 600;
    }
    .el-table__body-wrapper {
      overflow-x: auto;
      -webkit-overflow-scrolling: touch;
    }
    .el-table__cell:nth-child(1) { min-width: 100px; } /* 等级名称 */
    .el-table__cell:nth-child(2) { min-width: 60px; }  /* 排序 */
    .el-table__cell:nth-child(3) { min-width: 90px; }  /* 最低消费 */
    .el-table__cell:nth-child(4) { min-width: 70px; }  /* 折扣率 */
    .el-table__cell:nth-child(5) { min-width: 70px; }  /* 用户数 */
    .el-table__cell:nth-child(6) { min-width: 60px; }  /* 状态 */
    .el-table__cell:nth-child(7) { min-width: 120px; } /* 操作 */
  }
  :deep(.el-button) {
    padding: 6px 10px;
    font-size: 12px;
    & + .el-button {
      margin-left: 5px;
    }
  }
  .level-form-dialog {
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
      margin: 5vh auto !important;
      .el-dialog__body {
        padding: 15px;
        max-height: 70vh;
        overflow-y: auto;
      }
    }
  }
  .level-form-dialog {
    :deep(.el-form) {
      .el-form-item {
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
      .el-input,
      .el-input-number,
      .el-select,
      .el-textarea {
        width: 100% !important;
      }
      .el-input__wrapper,
      .el-textarea__inner {
        min-height: 40px;
        font-size: 16px; /* 防止iOS自动缩放 */
      }
      .el-input__inner {
        font-size: 16px !important; /* 防止iOS自动缩放 */
        min-height: 40px;
      }
      .form-tip {
        font-size: 12px;
        margin-top: 5px;
        color: #909399;
        line-height: 1.4;
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
  .card-header {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }
  .card-title {
    font-size: 15px;
    font-weight: 600;
    text-align: center;
  }
  .mobile-action-bar {
    padding: 12px;
  }
  .mobile-filter-buttons {
    margin-bottom: 10px;
  }
  .mobile-filter-buttons .el-button {
    height: 38px;
    font-size: 13px;
  }
  :deep(.el-card__header) {
    padding: 15px;
    font-size: 14px;
  }
}
@media (max-width: 480px) {
  .user-levels-admin {
    padding: 5px;
  }
  .card-header {
    gap: 10px;
  }
  .card-title {
    font-size: 14px;
  }
  .mobile-action-bar {
    padding: 10px;
  }
  .mobile-filter-buttons .el-button {
    height: 36px;
    font-size: 12px;
  }
  .mobile-action-btn {
    height: 42px;
    font-size: 15px;
  }
  .usage-guide-card {
    :deep(.el-card__body) {
      padding: 12px;
    }
  }
  .guide-section {
    padding: 10px;
    margin-bottom: 12px;
    :is(h4) {
      font-size: 13px;
    }
    :is(li) {
      font-size: 12px;
    }
  }
  :deep(.el-table) {
    font-size: 11px;
    .el-table__cell {
      padding: 6px 2px;
    }
    .el-table__header th {
      padding: 6px 2px;
      font-size: 11px;
    }
  }
  :deep(.el-button) {
    padding: 5px 8px;
    font-size: 11px;
  }
  :deep(.el-dialog) {
    width: 98% !important;
    margin: 2vh auto !important;
    .el-dialog__body {
      padding: 12px;
    }
  }
}
@media (min-width: 769px) {
  .mobile-action-bar {
    display: none !important;
  }
}
</style>
