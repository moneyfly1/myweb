<template>
  <div class="list-container profile">
    <div class="breadcrumb">首页 / 个人资料</div>
    <div class="page-header">
      <div class="page-title">
        <h1>个人资料</h1>
      </div>
      <div class="actions">
        <el-button @click="viewLoginHistory">
          登录历史
        </el-button>
        <router-link to="/settings">
          <el-button type="primary">
            编辑资料
          </el-button>
        </router-link>
      </div>
    </div>

    <div class="profile-grid">
      <div class="card profile-summary-card">
        <div class="card-body">
          <img v-if="userInfo.avatar" :src="userInfo.avatar" class="profile-avatar-img" alt="头像" />
          <div v-else class="profile-avatar">{{ displayName.slice(0, 2).toUpperCase() }}</div>
          <h2>{{ displayName }}</h2>
          <el-tag :type="getAccountStatusType(userInfo)">{{ getAccountStatusText(userInfo) }}</el-tag>
          <div class="item-meta">{{ getProfileMetaText() }}</div>
          <div class="summary-list profile-status-list">
            <div class="summary-row">
              <span>邮箱状态</span>
              <strong>
                <el-tag :type="userInfo.is_verified ? 'success' : 'warning'" size="small">
                  {{ userInfo.is_verified ? '已验证' : '未验证' }}
                </el-tag>
              </strong>
            </div>
            <div class="summary-row">
              <span>订阅状态</span>
              <strong>{{ getSubscriptionStatusText() }}</strong>
            </div>
            <div class="summary-row">
              <span>最近登录</span>
              <strong>{{ formatTime(userInfo.last_login) }}</strong>
            </div>
          </div>
        </div>
      </div>

      <div class="section-stack">
        <div class="card profile-card">
          <div class="card-header">
            <div>
              <h2 class="card-title">
                <el-icon class="card-header-icon"><User /></el-icon>
                账户信息
              </h2>
            </div>
          </div>
          <div class="card-body">
            <div class="table-wrap">
              <table class="table profile-info-table">
                <tbody>
                  <tr>
                    <th>用户名</th>
                    <td>{{ profileForm.username || '未设置用户名' }}</td>
                  </tr>
                  <tr>
                    <th>邮箱</th>
                    <td>
                      <span>{{ profileForm.email || '未设置邮箱' }}</span>
                      <router-link to="/settings">
                        <el-button size="small">修改邮箱</el-button>
                      </router-link>
                    </td>
                  </tr>
                  <tr>
                    <th>昵称</th>
                    <td>{{ profileForm.nickname || '未设置昵称' }}</td>
                  </tr>
                  <tr>
                    <th>头像</th>
                    <td>
                      <router-link to="/settings">
                        <el-button>修改头像</el-button>
                      </router-link>
                    </td>
                  </tr>
                  <tr>
                    <th>最近登录</th>
                    <td>{{ formatTime(userInfo.last_login) }}</td>
                  </tr>
                  <tr>
                    <th>注册时间</th>
                    <td>{{ formatTime(userInfo.created_at) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <div class="card password-card">
          <div class="card-header">
            <div>
              <h2 class="card-title">
                <el-icon class="card-header-icon"><Lock /></el-icon>
                安全设置
              </h2>
            </div>
          </div>
          <div class="card-body">
            <el-form
              ref="passwordFormRef"
              :model="passwordForm"
              :rules="passwordRules"
              :label-width="0"
              class="password-inline-form"
            >
              <div class="form-row password-form-row">
                <el-form-item prop="oldPassword">
                  <el-input v-model="passwordForm.oldPassword" type="password" show-password placeholder="当前密码" />
                </el-form-item>
                <el-form-item prop="newPassword">
                  <el-input v-model="passwordForm.newPassword" type="password" show-password placeholder="新密码" />
                </el-form-item>
                <el-form-item prop="confirmPassword">
                  <el-input v-model="passwordForm.confirmPassword" type="password" show-password placeholder="确认密码" />
                </el-form-item>
              </div>
            </el-form>
            <div class="notice password-notice">
              密码修改成功后会清空当前输入。
            </div>
            <div class="button-row password-actions">
              <el-button type="primary" :loading="passwordLoading" @click="changePassword">
                保存密码
              </el-button>
              <el-button @click="viewLoginHistory">登录历史</el-button>
            </div>
          </div>
        </div>

      </div>
    </div>
    <AppDialog
      v-model="loginHistoryDialogVisible"
      title="登录历史"
      width="900px"
      mobile-width="94%"
      class="login-history-dialog"
    >
      <LoadingState
        v-if="loginHistoryLoading"
        text="正在加载登录历史..."
        :size="32"
        class="loading-container"
      />
      <template v-else-if="loginHistory.length > 0">
        <el-table
          :data="loginHistory"
          stripe
          class="login-history-table desktop-login-history"
          max-height="400"
        >
          <el-table-column prop="login_time" label="登录时间" width="180">
            <template #default="scope">
              {{ formatTime(scope.row.login_time) }}
            </template>
          </el-table-column>
          <el-table-column prop="ip_address" label="IP地址/地区" width="180">
            <template #default="scope">
              <div class="login-location-stack">
                <el-tag type="info" size="small">{{ scope.row.ip_address || '未知' }}</el-tag>
                <el-tag
                  v-if="getLocationText(scope.row.location, scope.row.ip_address)"
                  type="success"
                  size="small"
                >
                  {{ getLocationText(scope.row.location, scope.row.ip_address) }}
                </el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="user_agent" label="设备信息" min-width="150">
            <template #default="scope">
              <el-tooltip :content="scope.row.user_agent" placement="top">
                <span class="user-agent-text">
                  {{ getDeviceInfo(scope.row.user_agent) }}
                </span>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column prop="login_status" label="状态" width="80">
            <template #default="scope">
              <el-tag :type="scope.row.login_status === 'success' ? 'success' : 'danger'" size="small">
                {{ scope.row.login_status === 'success' ? '成功' : '失败' }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
        <div class="mobile-login-history">
          <div
            v-for="(item, index) in loginHistory"
            :key="`${item.login_time}-${item.ip_address}-${index}`"
            class="login-history-card"
          >
            <div class="login-history-card__header">
              <span>{{ formatTime(item.login_time) }}</span>
              <el-tag :type="item.login_status === 'success' ? 'success' : 'danger'" size="small">
                {{ item.login_status === 'success' ? '成功' : '失败' }}
              </el-tag>
            </div>
            <div class="login-history-card__row">
              <span>IP地址</span>
              <strong>{{ item.ip_address || '未知' }}</strong>
            </div>
            <div v-if="getLocationText(item.location, item.ip_address)" class="login-history-card__row">
              <span>地区</span>
              <strong>{{ getLocationText(item.location, item.ip_address) }}</strong>
            </div>
            <div class="login-history-card__row">
              <span>设备</span>
              <strong>{{ getDeviceInfo(item.user_agent) }}</strong>
            </div>
          </div>
        </div>
      </template>
      <EmptyState
        v-else
        title="暂无登录记录"
        description="当前账户还没有可展示的登录历史。"
        :icon-size="52"
        class="profile-empty-state"
      />
      <template #footer>
        <FormActionBar
          :sticky="false"
          :show-submit="false"
          cancel-text="关闭"
          @cancel="loginHistoryDialogVisible = false"
        />
      </template>
    </AppDialog>
  </div>
</template>
<script>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/store/auth'
import { userAPI, subscriptionAPI, authAPI, api } from '@/utils/api'
import { formatDateTimeSafe, getLocationText as getLocationTextUtil } from '@/utils/date'
import { getUserStatusType, getUserStatusText, getSubscriptionStatusText as getSubscriptionStatusTextShared } from '@/utils/statusMaps'
import { getDeviceTypeFromUA as getDeviceInfo } from '@/utils/device'
import { Lock, User } from '@element-plus/icons-vue'
import AppDialog from '@/components/AppDialog.vue'
import EmptyState from '@/components/EmptyState.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import LoadingState from '@/components/LoadingState.vue'
import { useMobile } from '@/composables/useMobile'
export default {
  name: 'Profile',
  components: { AppDialog, EmptyState, FormActionBar, LoadingState, Lock, User },
  setup() {
    const router = useRouter()
    const authStore = useAuthStore()
    const isMobile = useMobile()
    const passwordLoading = ref(false)
    const emailLoading = ref(false)
    const profileFormRef = ref(null)
    const passwordFormRef = ref(null)
    const loginHistoryDialogVisible = ref(false)
    const loginHistoryLoading = ref(false)
    const loginHistory = ref([])
    const userInfo = ref({
      username: '',
      email: '',
      nickname: '',
      avatar: '',
      is_verified: false,
      last_login: null,
      created_at: null,
      status: 'active'
    })
    const subscriptionInfo = ref(null)
    const profileForm = reactive({
      username: '',
      email: '',
      nickname: '',
      avatar: ''
    })
    const displayName = computed(() => (
      profileForm.nickname || profileForm.username || userInfo.value.nickname || userInfo.value.username || '用户'
    ))
    const passwordForm = reactive({
      oldPassword: '',
      newPassword: '',
      confirmPassword: ''
    })
    const initUserInfo = () => {
      const authUser = authStore.user
      if (authUser) {
        userInfo.value.username = authUser.username || ''
        userInfo.value.email = authUser.email || ''
        userInfo.value.nickname = authUser.nickname || ''
        userInfo.value.avatar = authUser.avatar || authUser.avatar_url || ''
        profileForm.username = userInfo.value.username
        profileForm.email = userInfo.value.email
        profileForm.nickname = userInfo.value.nickname
        profileForm.avatar = userInfo.value.avatar
        }
    }
    const profileRules = {
      username: [
        { required: true, message: '请输入用户名', trigger: 'blur' }
      ],
      email: [
        { required: true, message: '请输入邮箱', trigger: 'blur' },
        { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
      ]
    }
    const passwordRules = {
      oldPassword: [
        { required: true, message: '请输入当前密码', trigger: 'blur' }
      ],
      newPassword: [
        { required: true, message: '请输入新密码', trigger: 'blur' },
        { min: 6, message: '密码长度不能少于6位', trigger: 'blur' },
        {
          validator: (rule, value, callback) => {
            if (value && passwordForm.oldPassword && value === passwordForm.oldPassword) {
              callback(new Error('新密码不能与当前密码相同'))
            } else {
              callback()
            }
          },
          trigger: 'blur'
        }
      ],
      confirmPassword: [
        { required: true, message: '请确认新密码', trigger: 'blur' },
        {
          validator: (rule, value, callback) => {
            if (value !== passwordForm.newPassword) {
              callback(new Error('两次输入的密码不一致'))
            } else {
              callback()
            }
          },
          trigger: 'blur'
        }
      ]
    }
    const fetchUserInfo = async () => {
      try {
        const response = await api.get('/users/me')
        let data = null
        if (response && response.data) {
          if (response.data.success && response.data.data) {
            data = response.data.data
          } else if (response.data.data) {
            data = response.data.data
          } else if (response.data) {
            data = response.data
          }
        }
        if (data) {
          userInfo.value = {
            username: data.username || '',
            email: data.email || '',
            nickname: data.nickname || '',
            avatar: data.avatar || data.avatar_url || '',
            is_verified: data.is_verified !== undefined ? data.is_verified : false,
            last_login: data.last_login || data.lastLogin || data.last_login_time || null,
            created_at: data.created_at || data.createdAt || null,
            status: data.is_active !== undefined ? (data.is_active ? 'active' : 'inactive') : 'active'
          }
          profileForm.username = userInfo.value.username || ''
          profileForm.email = userInfo.value.email || ''
          profileForm.nickname = userInfo.value.nickname || ''
          profileForm.avatar = userInfo.value.avatar || ''
        } else {
          const authUser = authStore.user
          if (authUser) {
            userInfo.value.username = authUser.username || ''
            userInfo.value.email = authUser.email || ''
            userInfo.value.nickname = authUser.nickname || ''
            userInfo.value.avatar = authUser.avatar || authUser.avatar_url || ''
            profileForm.username = userInfo.value.username
            profileForm.email = userInfo.value.email
            profileForm.nickname = userInfo.value.nickname
            profileForm.avatar = userInfo.value.avatar
          } else {
            ElMessage.error('获取用户信息失败：无法解析响应数据')
          }
        }
      } catch (error) {
        const authUser = authStore.user
        if (authUser) {
          userInfo.value.username = authUser.username || ''
          userInfo.value.email = authUser.email || ''
          userInfo.value.nickname = authUser.nickname || ''
          userInfo.value.avatar = authUser.avatar || authUser.avatar_url || ''
          profileForm.username = userInfo.value.username
          profileForm.email = userInfo.value.email
          profileForm.nickname = userInfo.value.nickname
          profileForm.avatar = userInfo.value.avatar
        } else {
          ElMessage.error(`获取用户信息失败: ${error.response?.data?.message || error.message || '未知错误'}`)
        }
      }
    }
    const fetchSubscriptionInfo = async () => {
      try {
        const response = await subscriptionAPI.getUserSubscription()
        if (response.data && response.data.success) {
          const data = response.data.data
          subscriptionInfo.value = {
            remainingDays: data.remainingDays || data.remaining_days || 0,
            expiryDate: data.expiryDate || data.expiry_date || '未设置',
            currentDevices: data.currentDevices || data.current_devices || data.online_devices || 0,
            maxDevices: data.maxDevices || data.max_devices || data.device_limit || 0,
            isExpiring: data.isExpiring || data.is_expiring || false,
            status: data.status || 'expired'
          }
          } else {
          subscriptionInfo.value = {
            remainingDays: 0,
            expiryDate: '未订阅',
            currentDevices: 0,
            maxDevices: 0,
            isExpiring: false,
            status: 'expired'
          }
        }
      } catch (error) {
        subscriptionInfo.value = {
          remainingDays: 0,
          expiryDate: '未订阅',
          currentDevices: 0,
          maxDevices: 0,
          isExpiring: false,
          status: 'expired'
        }
      }
    }
    const changePassword = async () => {
      if (!passwordFormRef.value) {
        ElMessage.error('表单引用未初始化')
        return
      }
      if (passwordForm.newPassword && passwordForm.oldPassword && 
          passwordForm.newPassword === passwordForm.oldPassword) {
        ElMessage.error('新密码不能与当前密码相同')
        return
      }
      try {
        await passwordFormRef.value.validate()
      } catch (error) {
        return
      }
      if (passwordLoading.value) {
        return
      }
      passwordLoading.value = true
      try {
        const response = await userAPI.changePassword({
          current_password: passwordForm.oldPassword,
          new_password: passwordForm.newPassword
        })
        if (response.data && response.data.success) {
          ElMessage.success(response.data.message || '密码修改成功')
          passwordForm.oldPassword = ''
          passwordForm.newPassword = ''
          passwordForm.confirmPassword = ''
          if (passwordFormRef.value) {
            passwordFormRef.value.resetFields()
          }
        } else {
          ElMessage.error(response.data?.message || '密码修改失败：响应格式错误')
        }
      } catch (error) {
        const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '未知错误'
        ElMessage.error(`密码修改失败: ${errorMsg}`)
      } finally {
        passwordLoading.value = false
      }
    }
    const fetchLoginHistory = async () => {
      loginHistoryLoading.value = true
      try {
        const response = await userAPI.getLoginHistory()
        if (response.data && response.data.success) {
          const data = response.data.data
          if (Array.isArray(data)) {
            loginHistory.value = data.map(item => ({
              login_time: item.login_time || '',
              ip_address: item.ip_address || '',
              location: item.location || '',
              country: item.country || '',
              city: item.city || '',
              user_agent: item.user_agent || '',
              login_status: item.login_status || 'success'
            }))
          } else if (data.logins && Array.isArray(data.logins)) {
            loginHistory.value = data.logins.map(item => ({
              login_time: item.login_time || '',
              ip_address: item.ip_address || '',
              location: item.location || '',
              country: item.country || '',
              city: item.city || '',
              user_agent: item.user_agent || '',
              login_status: item.login_status || 'success'
            }))
          } else {
            loginHistory.value = []
          }
        } else {
          ElMessage.error('获取登录历史失败：响应格式错误')
        }
      } catch (error) {
        ElMessage.error(`获取登录历史失败: ${error.message || '未知错误'}`)
      } finally {
        loginHistoryLoading.value = false
      }
    }
    const viewLoginHistory = () => {
      loginHistoryDialogVisible.value = true
      fetchLoginHistory()
    }
    const getLocationText = (location, ipAddress) => {
      return getLocationTextUtil(location, ipAddress, { pendingText: '解析中...' })
    }
    const formatTime = (time) => {
      if (!time || time === 'null' || time === 'None') {
        return '未知'
      }
      return formatDateTimeSafe(time, 'YYYY-MM-DD HH:mm:ss', '未知')
    }
    const getAccountStatusType = (userInfo) => {
      if (!userInfo || !userInfo.status) return 'info'
      return getUserStatusType(userInfo.status)
    }
    const getAccountStatusText = (userInfo) => {
      if (!userInfo || !userInfo.status) return '未知'
      return getUserStatusText(userInfo.status)
    }
    const getProfileMetaText = () => {
      const parts = []
      if (profileForm.username) {
        parts.push(`@${profileForm.username}`)
      }
      if (userInfo.value.email) {
        parts.push(userInfo.value.email)
      }
      return parts.join(' · ')
    }
    const getSubscriptionStatusText = () => {
      const status = subscriptionInfo.value?.status
      if (!status || status === 'expired') return '未订阅'
      return getSubscriptionStatusTextShared(status)
    }
    onMounted(() => {
      initUserInfo()
      fetchUserInfo()
      fetchSubscriptionInfo()
    })
    return {
      userInfo,
      subscriptionInfo,
      profileForm,
      displayName,
      passwordForm,
      profileFormRef,
      passwordFormRef,
      profileRules,
      passwordRules,
      passwordLoading,
      emailLoading,
      isMobile,
      loginHistoryDialogVisible,
      loginHistoryLoading,
      loginHistory,
      changePassword,
      viewLoginHistory,
      fetchLoginHistory,
      getDeviceInfo,
      getProfileMetaText,
      getSubscriptionStatusText,
      formatTime,
      getLocationText,
      getAccountStatusType,
      getAccountStatusText
    }
  }
}
</script>
<style scoped>
.profile {
  padding: 0;
  max-width: none;
  margin: 0;
  width: 100%;
}
.breadcrumb {
  margin-bottom: 12px;
  color: #606266;
  font-size: 13px;
  line-height: 1.4;
}
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
  padding: 16px;
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
}
.page-header h1 {
  margin: 0;
  color: #303133;
  font-size: 22px;
  line-height: 1.25;
  font-weight: 700;
}
.page-header p {
  margin: 6px 0 0;
  color: #606266;
  line-height: 1.5;
}
.actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}
.profile-grid {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 14px;
  align-items: start;
}
.section-stack {
  display: grid;
  gap: 14px;
}
.card {
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  overflow: hidden;
}
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid #ebeef5;
  min-width: 0;
}
.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  color: #303133;
  font-size: 16px;
  font-weight: 700;
}
.card-sub,
.item-meta {
  margin-top: 4px;
  color: #909399;
  font-size: 13px;
  line-height: 1.45;
}
.card-body {
  padding: 16px;
}
.profile-summary-card {
  text-align: center;
}
.profile-avatar {
  width: 72px;
  height: 72px;
  margin: 0 auto 12px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  background: #ecf5ff;
  color: #409eff;
  font-size: 22px;
  font-weight: 800;
}
.profile-avatar-img {
  width: 72px;
  height: 72px;
  margin: 0 auto 12px;
  display: block;
  border-radius: 8px;
  object-fit: cover;
  border: 1px solid #dcdfe6;
}
.profile-summary-card h2 {
  margin: 0 0 8px;
  color: #303133;
  font-size: 20px;
  line-height: 1.3;
}
.profile-status-list {
  margin-top: 16px;
}
.summary-list {
  margin-top: 14px;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  overflow: hidden;
  text-align: left;
}
.summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-bottom: 1px solid #ebeef5;
  color: #606266;
  font-size: 13px;
}
.summary-row:last-child {
  border-bottom: none;
}
.summary-row strong {
  min-width: 0;
  color: #303133;
  font-weight: 600;
  text-align: right;
  word-break: break-word;
}
.table-wrap {
  width: 100%;
  overflow-x: auto;
}
.table {
  width: 100%;
  border-collapse: collapse;
  background: #fff;
  font-size: 14px;
}
.table th,
.table td {
  padding: 12px;
  border-bottom: 1px solid #ebeef5;
  text-align: left;
  vertical-align: middle;
}
.table th {
  width: 150px;
  color: #606266;
  background: #f5f7fa;
  font-weight: 700;
}
.table tr:last-child th,
.table tr:last-child td {
  border-bottom: 0;
}
.profile-info-table td {
  color: #303133;
}
.profile-info-table td > span {
  display: inline-block;
  margin-right: 10px;
}
.form-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}
.password-inline-form :deep(.el-form-item) {
  margin-bottom: 0;
}
.notice {
  padding: 12px 14px;
  border: 1px solid #faecd8;
  border-radius: 6px;
  background: #fdf6ec;
  color: #b88230;
  line-height: 1.55;
}
.password-actions {
  margin-top: 12px;
}
.button-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.login-history-table {
  width: 100%;
}
.mobile-login-history {
  display: none;
}
.login-history-card {
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  padding: 12px;
}
.login-history-card + .login-history-card {
  margin-top: 10px;
}
.login-history-card__header,
.login-history-card__row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.login-history-card__header {
  margin-bottom: 10px;
  color: var(--theme-text, #303133);
  font-size: 14px;
  font-weight: 600;
}
.login-history-card__row {
  padding: 7px 0;
  border-top: 1px solid #f0f2f5;
  font-size: 13px;
}
.login-history-card__row span {
  flex-shrink: 0;
  color: #909399;
}
.login-history-card__row strong {
  min-width: 0;
  color: #303133;
  font-weight: 500;
  text-align: right;
  word-break: break-word;
}
.login-location-stack {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.profile-empty-state {
  min-height: 220px;
  padding: 36px 16px;
}
.profile-card,
.password-card,
.security-card,
.subscription-card {
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  :deep(.el-card__header) {
    padding: 12px 16px;
    font-size: 0.9375rem;
  }
  :deep(.el-card__body) {
    padding: 12px 16px;
  }
  @media (max-width: 768px) {
    :deep(.el-card__header) {
      padding: 10px 12px;
      font-size: 0.875rem;
    }
    :deep(.el-card__body) {
      padding: 10px 12px;
    }
  }
}
.card-header {
  font-weight: 600;
}
.card-header-icon,
.security-title-icon {
  flex-shrink: 0;
  color: #409eff;
  font-size: 16px;
}
.form-tip {
  font-size: 0.8125rem;
  color: #666;
  margin-top: 0.375rem;
  @media (max-width: 768px) {
    font-size: 0.75rem;
    margin-top: 0.25rem;
  }
}
.mobile-label {
  display: block;
  width: 100%;
  font-size: 14px;
  font-weight: 600;
  color: #333;
  margin-bottom: 8px;
  padding: 0;
}
.security-items {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  @media (max-width: 768px) {
    gap: 0.5rem;
  }
}
.security-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem;
  background: #f8f9fa;
  border-radius: 6px;
  @media (max-width: 768px) {
    padding: 0.625rem;
  }
}
.security-info {
  flex: 1;
}
.security-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
  color: #333;
  margin-bottom: 0.25rem;
  font-size: 0.9375rem;
  @media (max-width: 768px) {
    font-size: 0.875rem;
  }
}
.security-desc {
  color: #666;
  font-size: 0.8125rem;
  @media (max-width: 768px) {
    font-size: 0.75rem;
  }
}
.security-action {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  @media (max-width: 768px) {
    gap: 0.5rem;
  }
}
.subscription-info {
  margin-bottom: 14px;
}
.subscription-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-start;
  @media (max-width: 768px) {
    gap: 0.5rem;
  }
}
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    padding: 14px;
  }
  .actions,
  .actions .el-button,
  .actions a {
    width: 100%;
  }
  .profile-grid {
    grid-template-columns: 1fr;
  }
  .form-row {
    grid-template-columns: 1fr;
  }
  .table th {
    width: 112px;
  }
  .profile-card,
  .password-card,
  .security-card,
  .subscription-card {
    border-radius: 8px;
    margin: 0;
    border-left: 1px solid #dcdfe6;
    border-right: 1px solid #dcdfe6;
  }
  .profile-card:first-child {
    margin-top: 0;
  }
  .el-form {
    .el-form-item {
      margin-bottom: 18px;
      .el-form-item__label {
        font-size: 14px;
        margin-bottom: 8px;
        width: 100% !important;
        text-align: left;
        padding: 0;
        line-height: 1.5;
        font-weight: 600;
        color: #333;
      }
      .el-form-item__content {
        width: 100%;
          .el-input,
          .el-select {
            width: 100%;
            :deep(.el-input__wrapper) {
              min-height: 44px;
              border-radius: 8px;
              transition: border-color 0.16s ease, box-shadow 0.16s ease;
            }
            :deep(.el-input__inner) {
              font-size: 15px;
              min-height: 42px;
              line-height: 42px;
              padding: 0 12px;
            }
          }
      }
    }
  }
  .mobile-label {
    font-size: 14px;
    font-weight: 600;
    color: #333;
    margin-bottom: 10px;
    display: block;
  }
  .form-tip {
    font-size: 13px;
    margin-top: 8px;
    color: #999;
    padding-left: 4px;
  }
  .profile-form,
  .password-form {
    :deep(.el-form-item) {
      .el-form-item__label {
        width: 100% !important;
        margin-bottom: 10px;
        padding: 0;
      }
      .el-form-item__content {
        width: 100%;
        margin-left: 0 !important;
      }
    }
  }
  @media (max-width: 768px) {
    .profile-form,
    .password-form {
      :deep(.el-form-item) {
        display: flex;
        flex-direction: column;
        align-items: stretch;
        .el-form-item__label {
          order: 1;
          width: 100% !important;
          margin-bottom: 10px;
          padding: 0;
          text-align: left;
        }
        .el-form-item__content {
          order: 2;
          width: 100% !important;
          margin-left: 0 !important;
          flex: 1;
        }
      }
    }
    .mobile-label {
      display: block;
      width: 100%;
      margin-bottom: 10px;
      font-size: 14px;
      font-weight: 600;
      color: #333;
    }
  }
  .el-button {
    border-radius: 8px;
    min-height: 44px;
    padding: 10px 16px;
    font-size: 15px;
    touch-action: manipulation;
  }
  .security-items {
    gap: 0.5rem;
  }
  .security-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
    padding: 0.625rem;
    .security-info {
      width: 100%;
      .security-title {
        font-size: 0.875rem;
        margin-bottom: 4px;
      }
      .security-desc {
        font-size: 0.75rem;
      }
    }
      .security-action {
        width: 100%;
        justify-content: flex-start;
        flex-wrap: wrap;
        gap: 6px;
      .el-tag {
        margin-right: 0;
      }
      .el-button {
        width: 100%;
        margin: 0;
      }
    }
  }
  .subscription-info {
    grid-template-columns: 1fr;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
    .info-item {
      padding: 0.625rem;
      border-bottom: 1px solid #f0f0f0;
      &:last-child {
        border-bottom: none;
      }
      .label {
        font-size: 0.8125rem;
        display: block;
        margin-bottom: 3px;
      }
      .value {
        font-size: 0.875rem;
        display: block;
      }
    }
  }
  .subscription-actions {
    flex-direction: column;
    gap: 0.5rem;
    margin-top: 0;
    .el-button {
      width: 100%;
      margin: 0;
      height: 44px;
      border-radius: 8px;
      font-size: 0.875rem;
    }
  }
  .submit-btn {
    width: 100%;
    height: 44px;
    border-radius: 8px;
    font-size: 0.875rem;
  }
  .login-history-dialog {
    .loading-container {
      padding: 20px;
    }
    .user-agent-text {
      display: inline-block;
      max-width: 150px;
      overflow: clip;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }
  @media (max-width: 768px) {
    .login-history-dialog {
      .desktop-login-history {
        display: none;
      }

      .mobile-login-history {
        display: block;
      }
    }
  }
}
@media (max-width: 480px) {
  .profile-card,
  .password-card,
  .security-card,
  .subscription-card {
    :deep(.el-card__header) {
      padding: 8px 10px;
    }
    :deep(.el-card__body) {
      padding: 8px 10px;
    }
  }
}
.profile :deep(.el-input-group__prepend),
.profile :deep(.el-input-group__append) {
  background-color: #f5f7fa;
}
.profile :deep(.el-input-group__prepend) {
  border-right: 1px solid #dcdfe6;
}
.profile :deep(.el-input-group__append) {
  border-left: 1px solid #dcdfe6;
}
</style> 
