<template>
  <el-drawer
    :model-value="visible"
    @update:model-value="$emit('update:visible', $event)"
    :title="editingUser ? '编辑用户' : '添加用户'"
    :size="isMobile ? '100%' : '500px'"
    direction="rtl"
    class="user-form-dialog"
  >
    <el-form 
      :model="userForm" 
      :rules="userRules" 
      ref="userFormRef" 
      :label-width="isMobile ? '0' : '100px'"
      :label-position="isMobile ? 'top' : 'right'"
    >
      <el-form-item :label="isMobile ? '' : '邮箱'" prop="email">
        <template v-if="isMobile">
          <div class="mobile-label">邮箱 <span class="required">*</span></div>
        </template>
        <div class="input-wrapper">
          <el-input v-model="userForm.email" placeholder="请输入邮箱" class="styled-input" />
        </div>
      </el-form-item>
      <el-form-item :label="isMobile ? '' : '用户名'" prop="username">
        <template v-if="isMobile">
          <div class="mobile-label">用户名 <span class="required">*</span></div>
        </template>
        <div class="input-wrapper">
          <el-input v-model="userForm.username" placeholder="请输入用户名" class="styled-input" />
        </div>
      </el-form-item>
      <el-form-item :label="isMobile ? '' : '密码'" prop="password" v-if="!editingUser">
        <template v-if="isMobile">
          <div class="mobile-label">密码 <span class="required">*</span></div>
        </template>
        <div class="input-wrapper">
          <el-input v-model="userForm.password" type="password" placeholder="请输入密码" show-password class="styled-input" />
        </div>
      </el-form-item>
      <el-form-item :label="isMobile ? '' : '状态'" prop="status">
        <template v-if="isMobile">
          <div class="mobile-label">状态 <span class="required">*</span></div>
        </template>
        <div class="input-wrapper">
          <el-select v-model="userForm.status" placeholder="选择状态" class="styled-select">
            <el-option label="活跃" value="active" />
            <el-option label="待激活" value="inactive" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </div>
      </el-form-item>
      <el-form-item :label="isMobile ? '' : '最大设备数'" prop="device_limit" v-if="!editingUser">
        <template v-if="isMobile">
          <div class="mobile-label">最大设备数 <span class="required">*</span></div>
        </template>
        <div class="input-wrapper">
          <el-input-number 
            v-model="userForm.device_limit" 
            :min="0" 
            :max="100" 
            placeholder="请输入最大设备数量"
            controls-position="right"
            class="styled-input-number"
          />
        </div>
        <div class="form-item-hint">允许用户同时使用的最大设备数量（0表示不限制）</div>
      </el-form-item>
      <el-form-item :label="isMobile ? '' : '到期时间'" prop="expire_time" v-if="!editingUser">
        <template v-if="isMobile">
          <div class="mobile-label">到期时间 <span class="required">*</span></div>
        </template>
        <div class="input-wrapper">
          <el-date-picker
            v-model="userForm.expire_time"
            type="datetime"
            placeholder="选择到期时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DDTHH:mm:ss"
            class="styled-date-picker"
            :teleported="isMobile"
            :popper-class="isMobile ? 'mobile-date-picker-popper' : ''"
            :default-time="defaultTime"
          />
        </div>
        <div class="form-item-hint">订阅的到期时间，到期后用户将无法使用服务</div>
      </el-form-item>
      <el-form-item :label="isMobile ? '' : '管理员权限'" v-if="editingUser">
        <template v-if="isMobile">
          <div class="mobile-label">管理员权限</div>
        </template>
        <el-switch 
          v-model="userForm.is_admin" 
          active-text="是管理员"
          inactive-text="普通用户"
        />
      </el-form-item>
      <el-form-item :label="isMobile ? '' : '备注'" prop="note">
        <template v-if="isMobile">
          <div class="mobile-label">备注</div>
        </template>
        <div class="input-wrapper textarea-wrapper">
          <el-input 
            v-model="userForm.note" 
            type="textarea" 
            :rows="isMobile ? 2 : 3"
            placeholder="请输入备注信息"
            class="styled-textarea"
          />
        </div>
      </el-form-item>
    </el-form>
    <template #footer>
      <div class="dialog-footer-buttons" :class="{ 'mobile-footer': isMobile }">
        <el-button @click="$emit('update:visible', false)" :class="{ 'mobile-action-btn': isMobile }">取消</el-button>
        <el-button type="primary" @click="saveUser" :loading="saving" :class="{ 'mobile-action-btn': isMobile }">
          {{ editingUser ? '更新' : '创建' }}
        </el-button>
      </div>
    </template>
  </el-drawer>
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
          note: user.notes || '',
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
            is_admin: userForm.is_admin,
            notes: userForm.note || null
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
            expire_time: userForm.expire_time || getDefaultExpireTime(),
            notes: userForm.note || ''
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
.user-form-dialog {
  * { box-sizing: border-box; }
  :deep(.el-form-item) { margin-bottom: 20px; }

  /* 参考 ConfigUpdate：只保留外层输入框，内部输入无边框。保持原有宽度与统一高度 */
  .input-wrapper {
    width: 100%;
    min-height: 32px;
    border: 1px solid #dcdfe6;
    border-radius: 6px;
    overflow: hidden;
    transition: border-color 0.2s;
    &:focus-within { border-color: #409eff; }
  }
  .textarea-wrapper {
    min-height: auto;
  }

  .styled-input,
  .styled-select,
  .styled-input-number,
  .styled-date-picker,
  .styled-textarea {
    width: 100% !important;
  }

  /* 单行输入：统一高度 32px */
  .styled-input,
  .styled-select,
  .styled-input-number,
  .styled-date-picker {
    :deep(.el-input__wrapper),
    :deep(.el-input__inner) {
      box-shadow: none !important;
      border: none !important;
      border-radius: 0 !important;
      background: transparent !important;
    }
    :deep(.el-input__wrapper) {
      padding: 0 11px;
      min-height: 32px;
      &.is-focus { box-shadow: none !important; }
    }
    :deep(.el-input-number),
    :deep(.el-input) {
      width: 100%;
    }
  }

  /* 到期时间日期选择器：强制去除内层框，仅保留外层 input-wrapper */
  .input-wrapper :deep(.el-date-editor.styled-date-picker),
  .input-wrapper :deep(.el-date-editor) {
    width: 100% !important;
  }
  .input-wrapper :deep(.el-date-editor .el-input__wrapper),
  .input-wrapper :deep(.el-date-editor .el-input__inner) {
    box-shadow: none !important;
    border: none !important;
    border-radius: 0 !important;
    background: transparent !important;
  }
  .input-wrapper :deep(.el-date-editor .el-input__wrapper) {
    padding: 0 11px;
    min-height: 32px;
  }
  .input-wrapper :deep(.el-date-editor .el-input__wrapper.is-focus),
  .input-wrapper :deep(.el-date-editor .el-input__wrapper:hover) {
    box-shadow: none !important;
  }

  .styled-input-number {
    :deep(.el-input-number__decrease),
    :deep(.el-input-number__increase) {
      border-radius: 0 !important;
    }
  }

  .styled-textarea {
    :deep(.el-textarea__inner) {
      box-shadow: none !important;
      border: none !important;
      border-radius: 0 !important;
      background: transparent !important;
      padding: 8px 11px;
    }
  }
  &.mobile-dialog {
    :deep(.el-form-item__label) {
      display: none !important;
    }
    :deep(.el-form-item__content) {
      display: flex;
      flex-direction: column;
      margin-left: 0 !important;
      align-items: stretch;
    }
  }
  @media (max-width: 768px) {
    :deep(.el-form-item) {
      margin-bottom: 12px;
    }
    :deep(.el-form-item:last-child) {
      margin-bottom: 4px;
    }
    :deep(.el-form-item__label) {
      display: none !important;
    }
    :deep(.el-form-item__content) {
      margin-left: 0 !important;
      display: flex;
      flex-direction: column;
      align-items: stretch;
    }
    .mobile-label {
      font-size: 14px;
      font-weight: 500;
      color: #303133;
      margin-bottom: 6px;
      display: block;
      line-height: 1.4;
      .required {
        color: #f56c6c;
        margin-left: 2px;
      }
    }
    .input-wrapper {
      min-height: 40px;
    }
    .styled-input :deep(.el-input__wrapper),
    .styled-select :deep(.el-input__wrapper),
    .styled-input-number :deep(.el-input__wrapper),
    .styled-date-picker :deep(.el-input__wrapper),
    .input-wrapper :deep(.el-date-editor .el-input__wrapper) {
      min-height: 40px;
      padding: 0 11px;
      font-size: 16px;
    }
    .styled-textarea :deep(.el-textarea__inner) {
      min-height: 64px;
      padding: 8px 11px;
      font-size: 16px;
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
    margin-top: 2px;
    line-height: 1.35;
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
  @media (max-width: 768px) {
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
</style>
