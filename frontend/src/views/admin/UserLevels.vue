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
              <el-option label="全部" :value="null" />
              <el-option label="启用" :value="true" />
              <el-option label="禁用" :value="false" />
            </el-select>
            <el-button type="primary" @click="showAddDialog" :icon="Plus" class="add-button">添加等级</el-button>
          </div>
        </div>
      </template>
      
      <!-- 移动端操作栏 -->
      <div class="mobile-action-bar">
        <div class="mobile-filter-buttons">
          <el-button
            size="small"
            :type="statusFilter !== null ? 'primary' : 'default'"
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
      
      <!-- 移动端状态筛选抽屉 -->
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
                <el-option label="全部" :value="null" />
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

      <!-- 等级列表 -->
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
        <!-- 已删除设备限制列 -->
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

    <!-- 使用说明卡片 - 放在底部 -->
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

    <!-- 添加/编辑对话框 -->
    <el-dialog
      v-model="showDialog"
      :title="editingLevel ? '编辑等级' : '添加等级'"
      width="600px"
    >
      <el-form :model="levelForm" label-width="120px" ref="levelFormRef">
        <el-form-item label="等级名称" prop="level_name" :rules="[{ required: true, message: '请输入等级名称' }]">
          <el-input v-model="levelForm.level_name" placeholder="如：VIP、超级VIP、钻石会员" />
        </el-form-item>
        <el-form-item label="等级排序" prop="level_order" :rules="[{ required: true, message: '请输入等级排序' }]">
          <el-input-number 
            v-model="levelForm.level_order" 
            :min="1" 
            :max="100"
            placeholder="数字越小等级越高"
          />
          <div class="form-tip">数字越小，等级越高（1为最高等级）</div>
        </el-form-item>
        <el-form-item label="最低消费" prop="min_consumption" :rules="[{ required: true, message: '请输入最低消费' }]">
          <el-input-number 
            v-model="levelForm.min_consumption" 
            :min="0" 
            :precision="2"
            placeholder="累计消费达到此金额可升级"
          />
          <div class="form-tip">用户累计消费达到此金额可升级到此等级（元）</div>
        </el-form-item>
        <el-form-item label="折扣率" prop="discount_rate">
          <el-input-number 
            v-model="levelForm.discount_rate" 
            :min="0.1" 
            :max="1" 
            :step="0.05"
            :precision="2"
            placeholder="0.9表示9折"
          />
          <div class="form-tip">0.9表示9折，1.0表示无折扣</div>
        </el-form-item>
        <!-- 已删除设备限制功能，等级仅用于折扣优惠 -->
        <el-form-item label="等级颜色" prop="color">
          <el-color-picker v-model="levelForm.color" />
          <div class="form-tip">用于前端显示等级的颜色</div>
        </el-form-item>
        <el-form-item label="图标URL" prop="icon_url">
          <el-input v-model="levelForm.icon_url" placeholder="等级图标URL（可选）" />
        </el-form-item>
        <el-form-item label="权益说明" prop="benefits">
          <el-input 
            v-model="levelForm.benefits" 
            type="textarea" 
            :rows="4"
            placeholder='JSON格式，如：{"priority_support": true, "exclusive_nodes": true}'
            class="rectangular-input"
          />
        </el-form-item>
        <el-form-item label="是否启用" prop="is_active">
          <el-switch v-model="levelForm.is_active" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" @click="saveLevel" :loading="saving">保存</el-button>
      </template>
    </el-dialog>
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
const statusFilter = ref(null)
const isMobile = ref(window.innerWidth <= 768)
const showStatusFilterDrawer = ref(false)

const levelForm = reactive({
  level_name: '',
  level_order: 1,
  min_consumption: 0,
  discount_rate: 1.0,
  // device_limit 已删除，等级仅用于折扣优惠
  color: '#409eff',
  icon_url: '',
  benefits: '',
  is_active: true
})

const loadLevels = async () => {
  loading.value = true
  try {
    // 传递状态筛选参数
    const response = await userLevelAPI.getAllLevels(undefined, statusFilter.value)
    console.log('等级列表API响应:', response)
    // 处理多种可能的响应格式
    let levelList = []
    if (response?.data) {
      // 标准格式：{ success: true, data: { levels: [...] } }
      if (response.data.data && response.data.data.levels) {
        levelList = response.data.data.levels
      } 
      // 直接返回数组格式
      else if (Array.isArray(response.data)) {
        levelList = response.data
      }
      // 其他格式
      else if (response.data.levels) {
        levelList = response.data.levels
      }
    }
    // 确保 is_active 是布尔值
    levels.value = levelList.map(level => ({
      ...level,
      is_active: level.is_active === true || level.is_active === 1 || level.is_active === '1'
    }))
  } catch (error) {
    console.error('加载等级列表失败:', error)
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
  // 确保 is_active 是布尔值（处理可能的 0/1 或字符串格式）
  let isActiveValue = level.is_active
  if (typeof isActiveValue === 'number') {
    isActiveValue = isActiveValue !== 0
  } else if (typeof isActiveValue === 'string') {
    isActiveValue = isActiveValue === 'true' || isActiveValue === '1'
  } else if (isActiveValue === null || isActiveValue === undefined) {
    isActiveValue = true // 默认启用
  }
  
  Object.assign(levelForm, {
    level_name: level.level_name,
    level_order: level.level_order,
    min_consumption: level.min_consumption,
    discount_rate: level.discount_rate,
    // device_limit 已删除
    color: level.color || '#409eff',
    icon_url: level.icon_url || '',
    benefits: level.benefits || '',
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
    // device_limit 已删除，等级仅用于折扣优惠
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
    
    // 确保 is_active 是布尔值
    const isActiveValue = Boolean(levelForm.is_active)
    
    const data = {
      level_name: levelForm.level_name,
      level_order: levelForm.level_order,
      min_consumption: levelForm.min_consumption,
      discount_rate: levelForm.discount_rate,
      // device_limit 已删除
      color: levelForm.color,
      icon_url: levelForm.icon_url || null,
      benefits: levelForm.benefits || null,
      is_active: isActiveValue
    }
    
    console.log('保存等级数据:', data)
    console.log('is_active 值:', isActiveValue, '类型:', typeof isActiveValue)
    
    if (editingLevel.value) {
      const response = await userLevelAPI.updateLevel(editingLevel.value.id, data)
      console.log('更新等级响应:', response)
      ElMessage.success('等级更新成功')
    } else {
      await userLevelAPI.createLevel(data)
      ElMessage.success('等级创建成功')
    }
    
    showDialog.value = false
    await loadLevels()
  } catch (error) {
    if (error !== false) { // 表单验证失败会返回false
      console.error('保存等级失败:', error)
      console.error('错误详情:', error.response?.data)
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
      console.error('删除等级失败:', error)
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
  statusFilter.value = null
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

/* 去掉输入框内部的叠加输入框，只保留外部方形框 */
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

/* 确保对话框中的所有输入框都去掉内部叠加框，只保留外部方形框 */
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

/* 卡片头部样式 */
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

/* 使用说明卡片样式 */
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

/* 移动端操作栏样式 */
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

/* 手机端响应式样式 */
@media (max-width: 768px) {
  .user-levels-admin {
    padding: 10px;
  }
  
  .mobile-action-bar {
    display: block !important;
  }

  /* 使用说明卡片优化 */
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

  /* 表格优化 */
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
    
    /* 隐藏部分列在手机端 */
    .el-table__body-wrapper {
      overflow-x: auto;
      -webkit-overflow-scrolling: touch;
    }
    
    /* 调整列宽 */
    .el-table__cell:nth-child(1) { min-width: 100px; } /* 等级名称 */
    .el-table__cell:nth-child(2) { min-width: 60px; }  /* 排序 */
    .el-table__cell:nth-child(3) { min-width: 90px; }  /* 最低消费 */
    .el-table__cell:nth-child(4) { min-width: 70px; }  /* 折扣率 */
    .el-table__cell:nth-child(5) { min-width: 70px; }  /* 用户数 */
    .el-table__cell:nth-child(6) { min-width: 60px; }  /* 状态 */
    .el-table__cell:nth-child(7) { min-width: 120px; } /* 操作 */
  }

  /* 操作按钮优化 */
  :deep(.el-button) {
    padding: 6px 10px;
    font-size: 12px;
    
    & + .el-button {
      margin-left: 5px;
    }
  }

  /* 对话框优化 */
  :deep(.el-dialog) {
    width: 95% !important;
    margin: 5vh auto !important;
    
    .el-dialog__body {
      padding: 15px;
      max-height: 70vh;
      overflow-y: auto;
    }
  }

  /* 表单优化 */
  :deep(.el-form) {
    .el-form-item {
      margin-bottom: 18px;
      
      .el-form-item__label {
        width: 100% !important;
        text-align: left;
        margin-bottom: 8px;
        padding: 0;
        font-size: 14px;
        line-height: 1.5;
      }
      
      .el-form-item__content {
        margin-left: 0 !important;
        width: 100%;
      }
    }
    
    .el-input,
    .el-input-number,
    .el-select,
    .el-textarea {
      width: 100% !important;
    }
    
    .form-tip {
      font-size: 12px;
      margin-top: 5px;
      color: #909399;
    }
  }

  /* 卡片头部优化 */
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

  /* 卡片头部进一步优化 */
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

  /* 使用说明卡片进一步优化 */
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

  /* 表格进一步优化 */
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

  /* 操作按钮进一步优化 */
  :deep(.el-button) {
    padding: 5px 8px;
    font-size: 11px;
  }

  /* 对话框进一步优化 */
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

