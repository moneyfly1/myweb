<template>
  <el-dialog 
    :model-value="visible" 
    @update:model-value="$emit('update:visible', $event)"
    :title="editingUser ? '编辑用户' : '添加用户'"
    :width="isMobile ? '95%' : '600px'"
    :close-on-click-modal="!isMobile"
    class="user-form-dialog"
  >
    <el-form :model="userForm" :rules="userRules" ref="userFormRef" :label-width="isMobile ? '90px' : '100px'">
      <el-form-item label="邮箱" prop="email">
        <el-input v-model="userForm.email" placeholder="请输入邮箱" />
      </el-form-item>
      <el-form-item label="用户名" prop="username">
        <el-input v-model="userForm.username" placeholder="请输入用户名" />
      </el-form-item>
      <el-form-item label="密码" prop="password" v-if="!editingUser">
        <el-input v-model="userForm.password" type="password" placeholder="请输入密码" show-password />
      </el-form-item>
      <el-form-item label="状态" prop="status">
        <el-select v-model="userForm.status" placeholder="选择状态" style="width: 100%">
          <el-option label="活跃" value="active" />
          <el-option label="待激活" value="inactive" />
          <el-option label="禁用" value="disabled" />
        </el-select>
      </el-form-item>
      <el-form-item label="最大设备数" prop="device_limit" v-if="!editingUser">
        <el-input 
          v-model.number="userForm.device_limit" 
          type="number"
          :min="0" 
          :max="100" 
          placeholder="请输入最大设备数量"
          style="width: 100%"
        />
        <div class="form-item-hint">允许用户同时使用的最大设备数量（0表示不限制）</div>
      </el-form-item>
      <el-form-item label="到期时间" prop="expire_time" v-if="!editingUser">
        <el-date-picker
          v-model="userForm.expire_time"
          type="datetime"
          placeholder="选择到期时间"
          format="YYYY-MM-DD HH:mm:ss"
          value-format="YYYY-MM-DDTHH:mm:ss"
          style="width: 100%"
          :teleported="isMobile"
          :popper-class="isMobile ? 'mobile-date-picker-popper' : ''"
          :default-time="defaultTime"
        />
        <div class="form-item-hint">订阅的到期时间，到期后用户将无法使用服务</div>
      </el-form-item>
      <el-form-item label="管理员权限" v-if="editingUser">
        <el-switch 
          v-model="userForm.is_admin" 
          active-text="是管理员"
          inactive-text="普通用户"
        />
      </el-form-item>
      <el-form-item label="备注" prop="note">
        <el-input 
          v-model="userForm.note" 
          type="textarea" 
          :rows="3"
          placeholder="请输入备注信息"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <div class="dialog-footer-buttons">
        <el-button @click="$emit('update:visible', false)" class="mobile-action-btn">取消</el-button>
        <el-button type="primary" @click="saveUser" :loading="saving" class="mobile-action-btn">
          {{ editingUser ? '更新' : '创建' }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script>
import { ref, reactive, watch, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { adminAPI } from '@/utils/api'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
dayjs.extend(timezone)

export default {
  name: 'UserFormDialog',
  props: {
    visible: Boolean,
    editingUser: Object,
    isMobile: Boolean
  },
  emits: ['update:visible', 'success'],
  setup(props, { emit }) {
    const userFormRef = ref()
    const saving = ref(false)
    const defaultTime = ref(new Date(2000, 1, 1, 23, 59, 59))

    // 计算默认到期时间（一年后，使用北京时间）
    const getDefaultExpireTime = () => {
      const now = dayjs().tz('Asia/Shanghai')
      const oneYearLater = now.add(1, 'year')
      return oneYearLater.format('YYYY-MM-DDTHH:mm:ss')
    }

    const userForm = reactive({
      email: '',
      username: '',
      password: '',
      status: 'active',
      device_limit: 5,
      expire_time: getDefaultExpireTime(),
      is_admin: false,
      is_verified: false,
      note: ''
    })

    const userRules = {
      email: [
        { required: true, message: '请输入邮箱', trigger: 'blur' },
        { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
      ],
      username: [
        { required: true, message: '请输入用户名', trigger: 'blur' },
        { min: 2, max: 20, message: '用户名长度在2到20个字符', trigger: 'blur' }
      ],
      password: [
        { required: true, message: '请输入密码', trigger: 'blur' },
        { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
      ],
      status: [
        { required: true, message: '请选择状态', trigger: 'change' }
      ],
      device_limit: [
        { required: true, message: '请输入最大设备数量', trigger: 'blur' },
        { type: 'number', min: 0, max: 100, message: '设备数量应在0-100之间（0表示不限制）', trigger: 'blur' }
      ],
      expire_time: [
        { required: true, message: '请选择到期时间', trigger: 'change' }
      ]
    }

    const resetUserForm = () => {
      Object.assign(userForm, {
        email: '',
        username: '',
        password: '',
        status: 'active',
        device_limit: 5,
        expire_time: getDefaultExpireTime(),
        is_admin: false,
        is_verified: false,
        note: ''
      })
      if (userFormRef.value) {
        userFormRef.value.resetFields()
      }
    }

    // 监听编辑用户变化
    watch(() => props.editingUser, (user) => {
      if (user) {
        let status = user.status
        if (!status) {
          status = user.is_active ? 'active' : 'inactive'
        } else if (status === 'disabled') {
          status = user.is_active ? 'active' : 'inactive'
        }
        
        Object.assign(userForm, {
          email: user.email,
          username: user.username,
          status: status,
          is_admin: Boolean(user.is_admin),
          is_verified: Boolean(user.is_verified),
          note: user.note || '',
          // 编辑模式下不显示这些字段，但为了防止验证错误，保持原值或默认值
          password: '',
          device_limit: 5,
          expire_time: getDefaultExpireTime()
        })
      } else {
        resetUserForm()
      }
    }, { immediate: true })

    const saveUser = async () => {
      try {
        await userFormRef.value.validate()
        saving.value = true
        
        if (props.editingUser) {
          const isActive = userForm.status === 'active'
          const isVerified = Boolean(userForm.is_verified)
          const userData = {
            username: userForm.username,
            email: userForm.email,
            is_active: isActive,
            is_verified: isVerified,
            is_admin: userForm.is_admin
          }
          await adminAPI.updateUser(props.editingUser.id, userData)
          ElMessage.success('用户更新成功')
        } else {
          const userData = {
            username: userForm.username,
            email: userForm.email,
            password: userForm.password,
            is_active: userForm.status === 'active',
            is_admin: false,
            is_verified: false,
            device_limit: userForm.device_limit || 5,
            expire_time: userForm.expire_time || getDefaultExpireTime()
          }
          const response = await adminAPI.createUser(userData)
          if (response.data && response.data.success === false) {
            ElMessage.error(response.data.message || '用户创建失败')
            saving.value = false
            return
          }
          ElMessage.success('用户创建成功')
        }
        
        emit('success')
        emit('update:visible', false)
      } catch (error) {
        let errorMessage = '操作失败'
        if (error.response) {
          const data = error.response.data
          if (data) {
            if (data.message) {
              errorMessage = data.message
            } else if (data.detail) {
              errorMessage = data.detail
            } else if (typeof data === 'string') {
              errorMessage = data
            }
          }
        } else if (error.message) {
          errorMessage = error.message
        }
        ElMessage.error(errorMessage)
      } finally {
        saving.value = false
      }
    }

    return {
      userForm,
      userRules,
      userFormRef,
      saving,
      saveUser,
      defaultTime
    }
  }
}
</script>

<style scoped lang="scss">
// 提取自 Users.vue
.user-form-dialog {
  :deep(.el-dialog__body) {
    padding: 16px;
    max-height: calc(100vh - 200px);
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
  }
  
  :deep(.el-form-item) {
    margin-bottom: 20px;
  }
  
  /* 移除所有输入框的圆角和阴影效果，设置为简单长方形 - 完全移除嵌套边框 */
  /* 注意：需要覆盖全局样式 global.scss 中给 el-input__inner 添加的边框 */
  
  /* 基础输入框 - 外层包装器（唯一有边框的元素） */
  :deep(.el-input__wrapper) {
    border-radius: 0 !important;
    box-shadow: none !important;
    border: 1px solid #dcdfe6 !important;
    background-color: #ffffff !important;
    padding: 0 !important;
  }
  
  /* Select 下拉框的外层包装器 */
  :deep(.el-select .el-input__wrapper) {
    border-radius: 0 !important;
    box-shadow: none !important;
    border: 1px solid #dcdfe6 !important;
    background-color: #ffffff !important;
    padding: 0 !important;
  }
  
  /* 日期选择器的外层包装器 */
  :deep(.el-date-editor .el-input__wrapper) {
    border-radius: 0 !important;
    box-shadow: none !important;
    border: 1px solid #dcdfe6 !important;
    background-color: #ffffff !important;
    padding: 0 !important;
  }
  
  /* 内层输入框 - 完全移除边框、圆角、背景，只保留内边距 */
  /* 必须覆盖 global.scss 中的全局样式 */
  :deep(.el-input .el-input__inner),
  :deep(.el-input__inner),
  :deep(.el-input__inner input),
  :deep(.el-input__inner textarea),
  :deep(.el-select .el-input__inner),
  :deep(.el-date-editor .el-input__inner) {
    border-radius: 0 !important;
    border: none !important;
    border-width: 0 !important;
    outline: none !important;
    box-shadow: none !important;
    background-color: transparent !important;
    background: transparent !important;
    padding: 0 11px !important;
  }
  
  /* 确保所有嵌套的 input 元素都没有边框 */
  :deep(.el-input__wrapper input),
  :deep(.el-input__wrapper textarea),
  :deep(.el-input input),
  :deep(.el-input textarea),
  :deep(.el-select input),
  :deep(.el-date-editor input) {
    border: none !important;
    border-width: 0 !important;
    outline: none !important;
    border-radius: 0 !important;
    background: transparent !important;
    background-color: transparent !important;
    box-shadow: none !important;
  }
  
  /* 移除数字输入框的上下箭头 */
  :deep(.el-input__inner::-webkit-inner-spin-button),
  :deep(.el-input__inner::-webkit-outer-spin-button) {
    -webkit-appearance: none;
    margin: 0;
  }
  
  :deep(.el-input__inner[type="number"]) {
    -moz-appearance: textfield;
    appearance: textfield;
  }
  
  /* 移除前缀和后缀的背景和边框 */
  :deep(.el-input__prefix),
  :deep(.el-input__suffix) {
    background-color: transparent !important;
    background: transparent !important;
    border: none !important;
  }
  
  /* 移除前缀和后缀内部元素的边框 */
  :deep(.el-input__prefix *),
  :deep(.el-input__suffix *) {
    border: none !important;
    background: transparent !important;
  }
  
  /* 文本域 - 只有外层有边框 */
  :deep(.el-textarea__inner) {
    border-radius: 0 !important;
    border: 1px solid #dcdfe6 !important;
    box-shadow: none !important;
    outline: none !important;
  }
  
  /* 悬停和聚焦状态 */
  :deep(.el-input__wrapper:hover) {
    border-color: #c0c4cc !important;
    box-shadow: none !important;
  }
  
  :deep(.el-input__wrapper.is-focus) {
    border-color: #1677ff !important;
    box-shadow: none !important;
  }
  
  /* 确保聚焦时内层元素没有边框 */
  :deep(.el-input__wrapper.is-focus .el-input__inner),
  :deep(.el-input__wrapper.is-focus input),
  :deep(.el-input__wrapper.is-focus textarea) {
    border: none !important;
    outline: none !important;
    box-shadow: none !important;
  }
  
  // 手机端优化
  @media (max-width: 768px) {
    :deep(.el-dialog__body) {
      padding: 12px;
      max-height: calc(100vh - 120px);
    }
    
    :deep(.el-form-item) {
      margin-bottom: 16px;
    }
    
    :deep(.el-form-item__label) {
      font-size: 13px;
      padding-bottom: 6px;
      width: 90px !important;
    }
    
    :deep(.el-form-item__content) {
      margin-left: 90px !important;
    }
    
    :deep(.el-input),
    :deep(.el-select),
    :deep(.el-date-editor),
    :deep(.el-input-number) {
      width: 100%;
    }
  }
}

.form-item-hint {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
  
  @media (max-width: 768px) {
    font-size: 11px;
    margin-top: 3px;
  }
}

// 手机端日期选择器优化
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

.dialog-footer-buttons {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  
  @media (max-width: 768px) {
    .mobile-action-btn {
      flex: 1;
      height: 40px;
    }
  }
}
</style>

