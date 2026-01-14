<template>
  <div class="list-container admin-users">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>用户列表</span>
          <!-- 桌面端操作按钮 -->
          <div class="header-actions desktop-only">
            <el-button type="primary" @click="showAddUserDialog = true">
              <el-icon><Plus /></el-icon>
              添加用户
            </el-button>
          </div>
        </div>
      </template>

      <!-- 移动端智能操作栏 -->
      <div class="mobile-action-bar">
        <!-- 搜索栏（移动端优先显示） -->
        <div class="mobile-search-section">
          <div class="search-input-wrapper">
            <el-input
              v-model="searchForm.keyword"
              placeholder="输入邮箱或用户名搜索"
              class="mobile-search-input"
              clearable
              @keyup.enter="searchUsers"
            />
            <el-button 
              @click="searchUsers" 
              class="search-button-inside"
              type="default"
              plain
            >
              <el-icon><Search /></el-icon>
            </el-button>
          </div>
        </div>

        <!-- 筛选按钮组 -->
        <div class="mobile-filter-buttons">
          <el-dropdown @command="handleStatusFilter" trigger="click" placement="bottom-start">
            <el-button 
              size="small" 
              :type="searchForm.status ? 'primary' : 'default'"
              plain
            >
              <el-icon><Filter /></el-icon>
              {{ getStatusFilterText() }}
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="">全部状态</el-dropdown-item>
                <el-dropdown-item command="active">活跃</el-dropdown-item>
                <el-dropdown-item command="inactive">待激活</el-dropdown-item>
                <el-dropdown-item command="disabled">禁用</el-dropdown-item>
                <el-dropdown-item command="device_overlimit" divided>
                  <span style="color: #f56c6c; font-weight: bold;">⚠️ 设备超限</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>

          <el-button 
            size="small" 
            type="default" 
            plain
            @click="resetSearch"
          >
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
        </div>
        
        <!-- 时间选择器 - 移动端使用两个独立的日期选择器 -->
        <div class="mobile-date-picker-section">
          <div class="date-picker-row">
            <el-date-picker
              v-model="searchForm.start_date"
              type="date"
              placeholder="开始日期"
              size="default"
              format="YYYY-MM-DD"
              value-format="YYYY-MM-DD"
              class="mobile-date-picker-item"
              clearable
              @change="handleDateRangeChange"
              teleported
              popper-class="mobile-date-picker-popper"
            />
            <span class="date-separator">至</span>
            <el-date-picker
              v-model="searchForm.end_date"
              type="date"
              placeholder="结束日期"
              size="default"
              format="YYYY-MM-DD"
              value-format="YYYY-MM-DD"
              class="mobile-date-picker-item"
              clearable
              @change="handleDateRangeChange"
              teleported
              popper-class="mobile-date-picker-popper"
            />
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="mobile-action-buttons">
          <el-button 
            type="primary" 
            @click="showAddUserDialog = true"
            class="mobile-action-btn"
          >
            <el-icon><Plus /></el-icon>
            添加用户
          </el-button>
        </div>
      </div>

      <!-- 桌面端搜索栏 -->
      <el-form :inline="true" :model="searchForm" class="search-form desktop-only">
        <el-form-item label="搜索">
          <el-input 
            v-model="searchForm.keyword" 
            placeholder="输入用户邮箱或用户名进行搜索"
            style="width: 300px;"
            clearable
            @keyup.enter="searchUsers"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchForm.status" placeholder="选择状态" clearable style="width: 180px;" @change="searchUsers">
            <el-option label="全部" value="" />
            <el-option label="活跃" value="active" />
            <el-option label="待激活" value="inactive" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="注册时间">
          <el-date-picker
            v-model="searchForm.date_range"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="searchUsers">
            <el-icon><Search /></el-icon>
            搜索
          </el-button>
          <el-button @click="resetSearch">
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
        </el-form-item>
      </el-form>

      <!-- 批量操作工具栏 -->
      <div class="batch-actions" v-if="selectedUsers.length > 0">
        <div class="batch-info">
          <span>已选择 {{ selectedUsers.length }} 个用户</span>
        </div>
        <div class="batch-buttons">
          <el-button type="success" @click="batchEnableUsers" :loading="batchOperating">
            <el-icon><Check /></el-icon>
            批量启用
          </el-button>
          <el-button type="warning" @click="batchDisableUsers" :loading="batchOperating">
            <el-icon><Close /></el-icon>
            批量禁用
          </el-button>
          <el-button type="primary" @click="batchSendSubEmail" :loading="batchOperating">
            <el-icon><Message /></el-icon>
            发送订阅邮件
          </el-button>
          <el-button type="info" @click="batchSendExpireReminder" :loading="batchOperating">
            <el-icon><Bell /></el-icon>
            发送到期提醒
          </el-button>
          <el-button type="danger" @click="batchDeleteUsers" :loading="batchDeleting">
            <el-icon><Delete /></el-icon>
            批量删除
          </el-button>
          <el-button @click="clearSelection">
            <el-icon><Close /></el-icon>
            取消选择
          </el-button>
        </div>
      </div>

      <!-- 桌面端表格 -->
      <div class="table-wrapper desktop-only">
        <el-table 
          :data="users" 
          style="width: 100%" 
          v-loading="loading"
          @selection-change="handleSelectionChange"
          stripe
          table-layout="auto"
        >
        <el-table-column type="selection" width="50" />
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="email" label="邮箱" min-width="180" show-overflow-tooltip>
          <template #default="scope">
            <div class="user-email">
              <el-avatar :size="28" :src="scope.row.avatar">
                {{ scope.row.username?.charAt(0)?.toUpperCase() }}
              </el-avatar>
              <div class="email-info">
                <div class="email">
                  <el-button type="text" @click="viewUserDetails(scope.row.id)" class="clickable-text">
                    {{ scope.row.email }}
                  </el-button>
                </div>
                <div class="username">
                  <el-button type="text" @click="viewUserDetails(scope.row.id)" class="clickable-text">
                    {{ scope.row.username }}
                  </el-button>
                </div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="90">
          <template #default="scope">
            <el-tag :type="getStatusType(scope.row.status)" size="small">
              {{ getStatusText(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="balance" label="余额" width="100" sortable="custom" align="right">
          <template #default="scope">
            <el-button 
              type="text" 
              class="balance-link"
              @click="viewUserBalance(scope.row.id)"
              style="color: #409eff; font-weight: 600;"
            >
              ¥{{ (scope.row.balance || 0).toFixed(2) }}
            </el-button>
          </template>
        </el-table-column>
        <el-table-column label="设备信息" width="120" align="center">
          <template #default="scope">
            <div class="device-info">
              <div 
                class="device-stats" 
                :class="{ 'device-overlimit-alert': isDeviceOverlimit(scope.row) }"
              >
                <el-tooltip content="已订阅设备数量" placement="top">
                  <div class="device-item online">
                    <el-icon class="device-icon online-icon"><Monitor /></el-icon>
                    <span 
                      class="device-count"
                      :class="{ 'device-overlimit-count': isDeviceOverlimit(scope.row) }"
                    >
                      {{ scope.row.online_devices || 0 }}
                    </span>
                  </div>
                </el-tooltip>
                <div class="device-separator">/</div>
                <el-tooltip content="允许最大设备数量" placement="top">
                  <div class="device-item total">
                    <el-icon class="device-icon total-icon"><Connection /></el-icon>
                    <span class="device-count">{{ scope.row.subscription?.device_limit || 0 }}</span>
                  </div>
                </el-tooltip>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="订阅状态" width="130" align="center">
          <template #default="scope">
            <div v-if="scope.row.subscription" class="subscription-info">
              <div class="subscription-status">
                <el-tag 
                  :type="getSubscriptionStatusType(scope.row.subscription.status)" 
                  size="small"
                  effect="dark"
                >
                  {{ getSubscriptionStatusText(scope.row.subscription.status) }}
                </el-tag>
              </div>
              <div v-if="scope.row.subscription.days_until_expire !== null" class="expire-info">
                <el-text 
                  size="small" 
                  :type="scope.row.subscription.is_expired ? 'danger' : (scope.row.subscription.days_until_expire <= 7 ? 'warning' : 'success')"
                >
                  {{ scope.row.subscription.is_expired ? '已过期' : `${scope.row.subscription.days_until_expire}天后到期` }}
                </el-text>
              </div>
            </div>
            <div v-else class="no-subscription">
              <el-tag type="info" size="small" effect="plain">无订阅</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="注册时间" width="160" show-overflow-tooltip>
          <template #default="scope">
            {{ formatDate(scope.row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="last_login" label="最后登录" width="160" show-overflow-tooltip>
          <template #default="scope">
            {{ formatDate(scope.row.last_login) || '从未登录' }}
          </template>
        </el-table-column>
        <el-table-column label="到期时间" width="160" show-overflow-tooltip>
          <template #default="scope">
            <div v-if="scope.row.subscription && scope.row.subscription.expire_time" class="expire-time-info">
              <div class="expire-date">{{ formatDate(scope.row.subscription.expire_time) }}</div>
              <div class="expire-countdown">
                <el-text 
                  size="small" 
                  :type="scope.row.subscription.is_expired ? 'danger' : (scope.row.subscription.days_until_expire <= 7 ? 'warning' : 'success')"
                >
                  {{ scope.row.subscription.is_expired ? '已过期' : `${scope.row.subscription.days_until_expire}天后到期` }}
                </el-text>
              </div>
            </div>
            <div v-else class="no-expire">
              <el-text type="info" size="small">无订阅</el-text>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="scope">
            <div class="action-buttons">
              <div class="button-row">
                <el-button size="small" type="primary" @click="editUser(scope.row)">
                  <el-icon><Edit /></el-icon>
                  编辑
                </el-button>
                <el-button 
                  size="small" 
                  :type="scope.row.status === 'active' ? 'warning' : 'success'"
                  @click="toggleUserStatus(scope.row)"
                >
                  <el-icon><Switch /></el-icon>
                  {{ scope.row.status === 'active' ? '禁用' : '启用' }}
                </el-button>
              </div>
              <div class="button-row">
                <el-button 
                  size="small" 
                  type="info" 
                  @click="resetUserPassword(scope.row)"
                >
                  <el-icon><Key /></el-icon>
                  重置密码
                </el-button>
                <el-button 
                  size="small" 
                  type="warning" 
                  @click="unlockUserLogin(scope.row)"
                >
                  <el-icon><Unlock /></el-icon>
                  解除限制
                </el-button>
              </div>
              <div class="button-row">
                <el-button 
                  size="small" 
                  type="danger" 
                  @click="deleteUser(scope.row)"
                >
                  <el-icon><Delete /></el-icon>
                  删除
                </el-button>
              </div>
            </div>
          </template>
        </el-table-column>
      </el-table>
      </div>

      <!-- 移动端卡片式列表 -->
      <div class="mobile-card-list" v-if="users.length > 0 && isMobile">
        <div 
          v-for="user in users" 
          :key="user.id"
          class="mobile-card"
        >
          <div class="card-row">
            <span class="label">用户ID</span>
            <span class="value">#{{ user.id }}</span>
          </div>
          <div class="card-row">
            <span class="label">邮箱/用户名</span>
            <span class="value">
              <div style="display: flex; align-items: center; gap: 8px;">
                <el-avatar :size="24" :src="user.avatar">
                  {{ user.username?.charAt(0)?.toUpperCase() }}
                </el-avatar>
                <div>
                  <div style="font-weight: 600;">{{ user.email }}</div>
                  <div style="font-size: 0.85rem; color: #999;">{{ user.username }}</div>
                </div>
              </div>
            </span>
          </div>
          <div class="card-row">
            <span class="label">状态</span>
            <span class="value">
              <el-tag :type="getStatusType(user.status)" size="small">
                {{ getStatusText(user.status) }}
              </el-tag>
            </span>
          </div>
          <div class="card-row" v-if="user.subscription">
            <span class="label">订阅状态</span>
            <span class="value">
              <el-tag 
                :type="getSubscriptionStatusType(user.subscription.status)" 
                size="small"
              >
                {{ getSubscriptionStatusText(user.subscription.status) }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">注册时间</span>
            <span class="value">{{ formatDate(user.created_at) }}</span>
          </div>
          <div class="card-actions">
            <div class="action-buttons-row">
              <el-button 
                type="primary" 
                @click="editUser(user)"
                class="mobile-action-btn"
              >
                <el-icon><Edit /></el-icon>
                编辑
              </el-button>
              <el-button 
                :type="user.status === 'active' ? 'warning' : 'success'"
                @click="toggleUserStatus(user)"
                class="mobile-action-btn"
              >
                <el-icon><Switch /></el-icon>
                {{ user.status === 'active' ? '禁用' : '启用' }}
              </el-button>
            </div>
            <div class="action-buttons-row">
              <el-button 
                type="info" 
                @click="resetUserPassword(user)"
                class="mobile-action-btn"
              >
                <el-icon><Key /></el-icon>
                重置密码
              </el-button>
              <el-button 
                type="warning" 
                @click="unlockUserLogin(user)"
                class="mobile-action-btn"
              >
                <el-icon><Unlock /></el-icon>
                解除限制
              </el-button>
            </div>
            <div class="action-buttons-row">
              <el-button 
                type="danger" 
                @click="deleteUser(user)"
                class="mobile-action-btn"
              >
                <el-icon><Delete /></el-icon>
                删除
              </el-button>
            </div>
          </div>
        </div>
      </div>

      <!-- 移动端空状态 -->
      <div class="mobile-card-list" v-if="users.length === 0 && !loading && isMobile">
        <div class="empty-state">
          <i class="el-icon-user"></i>
          <p>暂无用户数据</p>
        </div>
      </div>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 添加/编辑用户对话框 -->
    <UserFormDialog
      v-model:visible="showAddUserDialog"
      :editingUser="editingUser"
      :isMobile="isMobile"
      @success="handleUserSaved"
    />

    <!-- 用户详情对话框 -->
    <UserDetailDialog
      v-model:visible="showUserDialog"
      :user="selectedUser"
      :isMobile="isMobile"
      :initialTab="activeBalanceTab"
    />

  </div>
</template>

<script>
import { ref, reactive, onMounted, onUnmounted, computed, nextTick, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Plus, Edit, Delete, View, Search, Refresh, 
  Switch, Key, Close, HomeFilled, Filter,
  Wallet, ShoppingCart, Clock, Connection,
  Unlock, Check, Message, Bell
} from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
import { secureStorage } from '@/utils/secureStorage'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
import { formatDate as formatDateUtil } from '@/utils/date'
import UserFormDialog from '@/components/admin/users/UserFormDialog.vue'
import UserDetailDialog from '@/components/admin/users/UserDetailDialog.vue'
dayjs.extend(timezone)

export default {
  name: 'AdminUsers',
  components: {
    UserFormDialog,
    UserDetailDialog,
    Plus, Edit, Delete, View, Search, Refresh, 
    Switch, Key, Close, HomeFilled, Filter,
    Wallet, ShoppingCart, Clock, Connection
  },
  setup() {
    const api = adminAPI
    const loading = ref(false)
    const saving = ref(false)
    const batchDeleting = ref(false)
    const batchOperating = ref(false)
    const users = ref([])
    const selectedUsers = ref([])
    const currentPage = ref(1)
    const pageSize = ref(20)
    const total = ref(0)
    const showAddUserDialog = ref(false)
    const showUserDialog = ref(false)
    const editingUser = ref(null)
    const selectedUser = ref(null)
    const userFormRef = ref()
    const activeBalanceTab = ref('recharge')
    const isMobile = ref(window.innerWidth <= 768)
    
    const handleResize = () => {
      isMobile.value = window.innerWidth <= 768
    }

    const searchForm = reactive({
      keyword: '',
      status: '',
      date_range: '',
      start_date: '',
      end_date: ''
    })

    const handleUserSaved = () => {
      showAddUserDialog.value = false
      editingUser.value = null
      loadUsers()
    }

    const editUser = (user) => {
      editingUser.value = user
      showAddUserDialog.value = true
    }

    // 判断设备是否超限
    const isDeviceOverlimit = (user) => {
      const onlineDevices = user.online_devices || 0
      const deviceLimit = user.subscription?.device_limit || 0
      return deviceLimit > 0 && onlineDevices >= deviceLimit
    }

    const loadUsers = async () => {
      loading.value = true
      try {
        const params = {
          page: currentPage.value,
          size: pageSize.value,
          keyword: searchForm.keyword,
          status: searchForm.status
        }
        
        // 处理日期范围参数
        // 优先使用 start_date 和 end_date，如果没有则使用 date_range
        if (searchForm.start_date && searchForm.end_date) {
          params.start_date = searchForm.start_date
          params.end_date = searchForm.end_date
        } else if (Array.isArray(searchForm.date_range) && searchForm.date_range.length === 2) {
          params.start_date = searchForm.date_range[0]
          params.end_date = searchForm.date_range[1]
        } else if (searchForm.date_range) {
          // 如果 date_range 是字符串或其他格式，尝试处理
          params.date_range = searchForm.date_range
        }
        
        const response = await api.getUsers(params)
        
        if (response.data && response.data.success && response.data.data) {
          const responseData = response.data.data
          let userList = responseData.users || []
          
          // 确保 is_active 和 is_verified 是布尔值
          userList = userList.map(user => ({
            ...user,
            is_active: user.is_active === true || user.is_active === 1 || user.is_active === '1',
            is_verified: user.is_verified === true || user.is_verified === 1 || user.is_verified === '1',
            is_admin: user.is_admin === true || user.is_admin === 1 || user.is_admin === '1'
          }))
          
          // 如果筛选设备超限用户，进行前端筛选
          if (searchForm.status === 'device_overlimit') {
            userList = userList.filter(user => isDeviceOverlimit(user))
          }
          
          users.value = userList
          total.value = searchForm.status === 'device_overlimit' ? userList.length : (responseData.total || 0)
        } else {
          users.value = []
          total.value = 0
          
          // 如果有错误消息，显示给用户
          if (response.data?.message) {
            ElMessage.error(`加载用户列表失败: ${response.data.message}`)
          }
        }
      } catch (error) {
        ElMessage.error(`加载用户列表失败: ${error.response?.data?.message || error.message}`)
        users.value = []
        total.value = 0
      } finally {
        loading.value = false
      }
    }

    const searchUsers = () => {
      currentPage.value = 1
      loadUsers()
    }

    const resetSearch = () => {
      Object.assign(searchForm, { 
        keyword: '', 
        status: '', 
        date_range: '',
        start_date: '',
        end_date: ''
      })
      searchUsers()
    }

    // 处理状态筛选
    const handleStatusFilter = (command) => {
      searchForm.status = command
      searchUsers()
    }

    // 获取状态筛选文本
    const getStatusFilterText = () => {
      const statusMap = {
        '': '状态筛选',
        'active': '活跃',
        'inactive': '待激活',
        'disabled': '禁用',
        'device_overlimit': '⚠️ 设备超限'
      }
      return statusMap[searchForm.status] || '状态筛选'
    }
    
    // 处理日期范围变更
    const handleDateRangeChange = () => {
      // 同步 start_date 和 end_date 到 date_range
      if (searchForm.start_date && searchForm.end_date) {
        searchForm.date_range = [searchForm.start_date, searchForm.end_date]
      } else if (!searchForm.start_date && !searchForm.end_date) {
        searchForm.date_range = ''
      }
      searchUsers()
    }
    
    // 监听 date_range 变化，同步到 start_date 和 end_date
    watch(() => searchForm.date_range, (newVal) => {
      if (Array.isArray(newVal) && newVal.length === 2) {
        searchForm.start_date = newVal[0]
        searchForm.end_date = newVal[1]
      } else {
        searchForm.start_date = ''
        searchForm.end_date = ''
      }
    }, { immediate: true })

    const handleSizeChange = (val) => {
      pageSize.value = val
      loadUsers()
    }

    const handleCurrentChange = (val) => {
      currentPage.value = val
      loadUsers()
    }

    const viewUserDetails = async (userId) => {
      try {
        const response = await adminAPI.getUserDetails(userId)
        
        if (response && response.data && response.data.success) {
          const userData = response.data.data
          selectedUser.value = userData
          // 调试信息（仅在开发环境）
          if (process.env.NODE_ENV === 'development') {
            console.log('Users.vue - API response:', response.data)
            console.log('Users.vue - selectedUser set to:', userData)
            console.log('Users.vue - recharge_records:', userData?.recharge_records, 'length:', userData?.recharge_records?.length)
            console.log('Users.vue - orders:', userData?.orders, 'length:', userData?.orders?.length)
          }
          showUserDialog.value = true
          activeBalanceTab.value = 'recharge'
        } else if (response && response.success) {
          selectedUser.value = response.data
          if (process.env.NODE_ENV === 'development') {
            console.log('Users.vue - selectedUser set to (alt path):', response.data)
            console.log('Users.vue - recharge_records:', response.data?.recharge_records)
            console.log('Users.vue - orders:', response.data?.orders)
          }
          showUserDialog.value = true
          activeBalanceTab.value = 'recharge'
        } else {
          ElMessage.error('获取用户详情失败: ' + (response?.data?.message || response?.message || '未知错误'))
        }
      } catch (error) {
        ElMessage.error('获取用户详情失败: ' + (error.response?.data?.message || error.message))
      }
    }
    
    const viewUserBalance = async (userId) => {
      activeBalanceTab.value = 'recharge'
      await viewUserDetails(userId)
    }


    const deleteUser = async (user) => {
      // 检查用户ID是否有效
      if (!user || !user.id || user.id === 0) {
        ElMessage.warning('无效的用户ID，无法删除')
        return
      }

      try {
        await ElMessageBox.confirm(
          `确定要删除用户 "${user.username || user.email || '未知用户'}" 吗？此操作不可恢复。`, 
          '确认删除', 
          { type: 'warning' }
        )
        await adminAPI.deleteUser(user.id)
        ElMessage.success('用户删除成功')
        loadUsers()
      } catch (error) {
        if (error !== 'cancel') {
          const errorMsg = error.response?.data?.message || error.message || '删除失败'
          ElMessage.error(`删除失败: ${errorMsg}`)
        }
      }
    }

    const toggleUserStatus = async (user) => {
      try {
        const newStatus = user.status === 'active' ? 'disabled' : 'active'
        const action = newStatus === 'active' ? '启用' : '禁用'
        
        await ElMessageBox.confirm(
          `确定要${action}用户 "${user.username}" 吗？`, 
          `确认${action}`, 
          { type: 'warning' }
        )
        
        await adminAPI.updateUserStatus(user.id, newStatus)
        ElMessage.success(`用户${action}成功`)
        loadUsers()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(`状态更新失败: ${error.response?.data?.message || error.message}`)
        }
      }
    }


    const getStatusType = (status) => {
      const statusMap = {
        'active': 'success',
        'inactive': 'warning',
        'disabled': 'danger'
      }
      return statusMap[status] || 'info'
    }

    const getStatusText = (status) => {
      const statusMap = {
        'active': '活跃',
        'inactive': '待激活',
        'disabled': '禁用'
      }
      return statusMap[status] || status
    }

    const formatDate = (date) => {
      // 使用统一的北京时间格式化函数
      return formatDateUtil(date) || ''
    }

    const loginAsUser = async (user) => {
      try {
        await ElMessageBox.confirm(
          `确定要以用户 ${user.username} 的身份登录吗？`,
          '确认登录',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'info'
          }
        )
        
        const response = await api.post(`/admin/users/${user.id}/login-as`)
        
        // 后端返回格式: {success: true, message: "...", data: {access_token: "...", user: {...}}}
        if (!response.data || !response.data.data || !response.data.data.access_token || !response.data.data.user) {
          ElMessage.error('登录失败：服务器返回数据不完整')
          return
        }
        
        ElMessage.success('登录成功，正在跳转...')
        
        // 保存管理员信息到 secureStorage（用于返回管理员后台）
        const adminToken = secureStorage.get('admin_token')
        const adminUser = secureStorage.get('admin_user')
        
        // 通过URL参数传递token和用户信息，在新标签页中打开用户后台
        const userToken = response.data.data.access_token
        const userData = response.data.data.user
        
        // 构建用户数据，如果存在管理员信息，也包含在用户数据中
        let userDataToSend = { ...userData }
        if (adminToken && adminUser) {
          // 将管理员信息编码到用户数据中（通过一个特殊字段）
          userDataToSend._adminToken = adminToken
          userDataToSend._adminUser = typeof adminUser === 'string' ? adminUser : JSON.stringify(adminUser)
        }
        
        const userDataStr = encodeURIComponent(JSON.stringify(userDataToSend))
        
        // 在新标签页中打开用户后台，通过URL参数传递认证信息
        const dashboardUrl = `/dashboard?token=${userToken}&user=${userDataStr}`
        window.open(dashboardUrl, '_blank')
        
      } catch (error) {
        if (error !== 'cancel') {
          if (process.env.NODE_ENV === 'development') {
            console.error('登录失败:', error)
          }
          ElMessage.error(error.response?.data?.message || '登录失败')
        }
      }
    }

    // 重置用户密码
    const resetUserPassword = async (user) => {
      try {
        const { value: newPassword } = await ElMessageBox.prompt(
          `为用户 ${user.username} 设置新密码`,
          '重置密码',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            inputType: 'password',
            inputPlaceholder: '请输入新密码（至少6位）',
            inputValidator: (value) => {
              if (!value) {
                return '密码不能为空'
              }
              if (value.length < 6) {
                return '密码长度不能少于6位'
              }
              return true
            }
          }
        )

        await adminAPI.resetUserPassword(user.id, newPassword)
        
        ElMessage.success('密码重置成功')
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(`密码重置失败: ${error.response?.data?.message || error.message}`)
        }
      }
    }

    const unlockUserLogin = async (user) => {
      try {
        await ElMessageBox.confirm(
          `确定要解除用户 "${user.username}" 的登录限制吗？这将清除该用户的所有登录失败记录。`,
          '解除登录限制',
          { type: 'warning' }
        )
        const result = await adminAPI.unlockUserLogin(user.id)
        ElMessage.success(result.message || '登录限制已解除')
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(`解除限制失败: ${error.response?.data?.message || error.message}`)
        }
      }
    }

    // 获取订阅状态类型
    const getSubscriptionStatusType = (status) => {
      const statusMap = {
        'active': 'success',
        'inactive': 'info',
        'expired': 'danger'
      }
      return statusMap[status] || 'info'
    }

    // 获取订阅状态文本
    const getSubscriptionStatusText = (status) => {
      const statusMap = {
        'active': '活跃',
        'inactive': '未激活',
        'expired': '已过期'
      }
      return statusMap[status] || '未知'
    }

    // 批量操作相关函数
    const handleSelectionChange = (selection) => {
      selectedUsers.value = selection
    }

    const clearSelection = () => {
      selectedUsers.value = []
      // 清除表格选择
      const table = document.querySelector('.el-table')
      if (table) {
        const checkboxes = table.querySelectorAll('input[type="checkbox"]')
        checkboxes.forEach(checkbox => {
          checkbox.checked = false
        })
      }
    }

    const batchDeleteUsers = async () => {
      if (selectedUsers.value.length === 0) {
        ElMessage.warning('请先选择要删除的用户')
        return
      }

      // 检查是否包含管理员用户
      const adminUsers = selectedUsers.value.filter(user => user.is_admin)
      if (adminUsers.length > 0) {
        ElMessage.error('不能删除管理员用户')
        return
      }

      try {
        await ElMessageBox.confirm(
          `确定要删除选中的 ${selectedUsers.value.length} 个用户吗？此操作将清空这些用户的所有数据（订阅、设备、日志等），且不可恢复。`, 
          '确认批量删除', 
          { 
            type: 'warning',
            confirmButtonText: '确定删除',
            cancelButtonText: '取消'
          }
        )

        batchDeleting.value = true
        
        // 获取要删除的用户ID列表
        const userIds = selectedUsers.value.map(user => user.id)
        
        // 调用批量删除API
        await adminAPI.batchDeleteUsers(userIds)
        
        ElMessage.success(`成功删除 ${selectedUsers.value.length} 个用户`)
        
        // 清空选择并重新加载数据
        clearSelection()
        loadUsers()
        
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(`批量删除失败: ${error.response?.data?.message || error.message}`)
        }
      } finally {
        batchDeleting.value = false
      }
    }

    const batchEnableUsers = async () => {
      if (selectedUsers.value.length === 0) {
        ElMessage.warning('请先选择要启用的用户')
        return
      }

      try {
        batchOperating.value = true
        const userIds = selectedUsers.value.map(user => user.id)
        const response = await adminAPI.batchEnableUsers(userIds)
        
        if (response.data?.success !== false) {
          ElMessage.success(response.data?.message || `成功启用 ${selectedUsers.value.length} 个用户`)
          clearSelection()
          loadUsers()
        } else {
          ElMessage.error(response.data?.message || '批量启用失败')
        }
      } catch (error) {
        ElMessage.error(`批量启用失败: ${error.response?.data?.message || error.message}`)
      } finally {
        batchOperating.value = false
      }
    }

    const batchDisableUsers = async () => {
      if (selectedUsers.value.length === 0) {
        ElMessage.warning('请先选择要禁用的用户')
        return
      }

      // 检查是否包含管理员用户
      const adminUsers = selectedUsers.value.filter(user => user.is_admin)
      if (adminUsers.length > 0) {
        ElMessage.error('不能禁用管理员用户')
        return
      }

      try {
        await ElMessageBox.confirm(
          `确定要禁用选中的 ${selectedUsers.value.length} 个用户吗？`,
          '确认批量禁用',
          {
            type: 'warning',
            confirmButtonText: '确定禁用',
            cancelButtonText: '取消'
          }
        )

        batchOperating.value = true
        const userIds = selectedUsers.value.map(user => user.id)
        const response = await adminAPI.batchDisableUsers(userIds)
        
        if (response.data?.success !== false) {
          ElMessage.success(response.data?.message || `成功禁用 ${selectedUsers.value.length} 个用户`)
          clearSelection()
          loadUsers()
        } else {
          ElMessage.error(response.data?.message || '批量禁用失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(`批量禁用失败: ${error.response?.data?.message || error.message}`)
        }
      } finally {
        batchOperating.value = false
      }
    }

    const batchSendSubEmail = async () => {
      if (selectedUsers.value.length === 0) {
        ElMessage.warning('请先选择要发送邮件的用户')
        return
      }

      try {
        batchOperating.value = true
        const userIds = selectedUsers.value.map(user => user.id)
        const response = await adminAPI.batchSendSubEmail(userIds)
        
        if (response.data?.success !== false) {
          const data = response.data?.data || {}
          const successCount = data.success_count || selectedUsers.value.length
          const failCount = data.fail_count || 0
          ElMessage.success(response.data?.message || `成功发送 ${successCount} 封邮件${failCount > 0 ? `，失败 ${failCount} 封` : ''}`)
        } else {
          ElMessage.error(response.data?.message || '批量发送邮件失败')
        }
      } catch (error) {
        ElMessage.error(`批量发送邮件失败: ${error.response?.data?.message || error.message}`)
      } finally {
        batchOperating.value = false
      }
    }

    const batchSendExpireReminder = async () => {
      if (selectedUsers.value.length === 0) {
        ElMessage.warning('请先选择要发送提醒的用户')
        return
      }

      try {
        batchOperating.value = true
        const userIds = selectedUsers.value.map(user => user.id)
        const response = await adminAPI.batchSendExpireReminder(userIds)
        
        if (response.data?.success !== false) {
          const data = response.data?.data || {}
          const successCount = data.success_count || selectedUsers.value.length
          const failCount = data.fail_count || 0
          ElMessage.success(response.data?.message || `成功发送 ${successCount} 封提醒邮件${failCount > 0 ? `，失败 ${failCount} 封` : ''}`)
        } else {
          ElMessage.error(response.data?.message || '批量发送提醒失败')
        }
      } catch (error) {
        ElMessage.error(`批量发送提醒失败: ${error.response?.data?.message || error.message}`)
      } finally {
        batchOperating.value = false
      }
    }

    onMounted(() => {
      loadUsers()
      window.addEventListener('resize', handleResize)
      // 监听订阅管理更新设备限制的事件
      window.addEventListener('subscription-device-limit-updated', loadUsers)
    })

    onUnmounted(() => {
      window.removeEventListener('resize', handleResize)
      // 移除订阅管理更新设备限制的事件监听
      window.removeEventListener('subscription-device-limit-updated', loadUsers)
    })

      return {
      isMobile,
      loading,
      saving,
      batchDeleting,
      batchOperating,
      users,
      selectedUsers,
      currentPage,
      pageSize,
      total,
      searchForm,
      showAddUserDialog,
      showUserDialog,
      editingUser,
      selectedUser,
      activeBalanceTab,
      searchUsers,
      resetSearch,
      handleStatusFilter,
      getStatusFilterText,
      handleDateRangeChange,
      handleSizeChange,
      handleCurrentChange,
      viewUserDetails,
      viewUserBalance,
      editUser,
      deleteUser,
      toggleUserStatus,
      getStatusType,
      getStatusText,
      formatDate,
      resetUserPassword,
      unlockUserLogin,
      getSubscriptionStatusType,
      getSubscriptionStatusText,
      handleSelectionChange,
      clearSelection,
      batchDeleteUsers,
      batchEnableUsers,
      batchDisableUsers,
      batchSendSubEmail,
      batchSendExpireReminder,
      isDeviceOverlimit,
      handleUserSaved,
      // 导出图标组件供模板使用
      Search,
      Unlock,
      Check,
      Message,
      Bell
    }
  }
}
</script>

<style scoped lang="scss">
@use '@/styles/list-common.scss';

.admin-users {
  @media (max-width: 768px) {
    width: 100% !important;
    max-width: 100% !important;
    margin-left: 0 !important;
    margin-right: 0 !important;
    padding-left: 12px !important;
    padding-right: 12px !important;
  }
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

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.search-form {
  margin-bottom: 20px;
  padding: 20px;
  background: #f5f7fa;
  border-radius: 8px;
  
  &.desktop-only {
    @media (max-width: 768px) {
      display: none !important;
    }
  }
  
  :deep(.el-form-item) {
    .el-select {
      min-width: 180px;
      width: 180px;
    }
    
    .el-date-editor {
      min-width: 240px;
      width: 240px;
    }
  }
}

// 移动端样式适配
@media (max-width: 768px) {
  .mobile-action-bar {
    padding: 16px !important;
    box-sizing: border-box !important;
    
    .mobile-search-section {
      margin-bottom: 12px !important;
      width: 100% !important;
      box-sizing: border-box !important;
      
      .search-input-wrapper {
        position: relative !important;
        display: flex !important;
        align-items: center !important;
        width: 100% !important;
        box-sizing: border-box !important;
        
        .mobile-search-input {
          flex: 1 !important;
          width: 100% !important;
          
          :deep(.el-input__wrapper) {
            border-radius: 10px !important;
            padding-right: 60px !important;
            min-height: 48px !important;
          }
        }
        
        .search-button-inside {
          position: absolute !important;
          right: 4px !important;
          top: 50% !important;
          transform: translateY(-50%) !important;
          height: 40px !important;
          width: 40px !important;
          min-width: 40px !important;
          display: flex !important;
          align-items: center !important;
          justify-content: center !important;
          z-index: 10 !important;
        }
      }
    }
    
    .mobile-filter-buttons {
      display: flex !important;
      gap: 10px !important;
      width: 100% !important;
      
      .el-dropdown, .el-button {
        flex: 1 !important;
        width: 100% !important;
      }
      
      .el-button {
        height: 44px !important;
        border-radius: 10px !important;
      }
    }
  }
}

.mobile-date-picker-section {
  margin-bottom: 14px;
  width: 100%;
  
  .date-picker-row {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    
    .mobile-date-picker-item {
      flex: 1;
      min-width: 0;
      
      :deep(.el-input__wrapper) {
        min-height: 48px;
        border-radius: 10px;
        width: 100%;
      }
    }
    
    .date-separator {
      flex-shrink: 0;
      padding: 0 6px;
      font-weight: 600;
      color: #606266;
    }
  }
}


.mobile-action-buttons {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
  
  .mobile-action-btn {
    width: 100%;
    height: 44px;
    font-size: 16px;
    border-radius: 6px;
  }
}

.batch-actions {
  margin: 20px 0;
  padding: 15px;
  background: #f0f9ff;
  border: 1px solid #bae6fd;
  border-radius: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  
  @media (max-width: 768px) {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }
}

.batch-info {
  font-weight: 600;
  color: #303133;
  font-size: 14px;
  
  @media (max-width: 768px) {
    text-align: center;
    font-size: 13px;
  }
}

.batch-buttons {
  display: flex;
  gap: 10px;
  
  @media (max-width: 768px) {
    width: 100%;
    
    .el-button {
      flex: 1;
    }
  }
}

.user-email {
  display: flex;
  align-items: center;
  gap: 8px;
}

.email-info {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.email, .username {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.device-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.device-stats {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  background: #f5f7fa;
  border-radius: 6px;
  transition: all 0.3s;
  
  &.device-overlimit-alert {
    background: #fef0f0;
    border: 1px solid #f56c6c;
    animation: pulse-alert 2s ease-in-out infinite;
  }
}

@keyframes pulse-alert {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(245, 108, 108, 0.4);
  }
  50% {
    box-shadow: 0 0 0 4px rgba(245, 108, 108, 0);
  }
}

.device-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.device-icon {
  font-size: 16px;
  
  &.online-icon {
    color: #67c23a;
  }
  
  &.total-icon {
    color: #409eff;
  }
}

.device-separator {
  color: #909399;
  font-weight: 600;
  padding: 0 4px;
}

.device-count {
  font-weight: 600;
  font-size: 14px;
  
  &.device-overlimit-count {
    color: #f56c6c;
    font-weight: 700;
  }
}

.subscription-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.subscription-status {
  margin-bottom: 4px;
}

.expire-info {
  font-size: 12px;
  margin-top: 4px;
}

.no-subscription {
  text-align: center;
  color: #909399;
  font-size: 12px;
}

.expire-time-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.expire-date {
  font-size: 13px;
  color: #303133;
  font-weight: 500;
}

.expire-countdown {
  font-size: 12px;
  margin-top: 2px;
}

.no-expire {
  text-align: center;
  color: #909399;
  font-size: 12px;
}

.action-buttons {
  display: flex;
  flex-direction: column;
  gap: 4px;
  
  .button-row {
    display: flex;
    gap: 4px;
    justify-content: center;
    
    .el-button {
      flex: 1;
      padding: 5px 8px;
      font-size: 12px;
    }
  }
}

.table-wrapper {
  width: 100%;
  overflow-x: auto;
  
  :deep(.el-table) {
    min-width: 1400px;
  }
}

@media (max-width: 768px) {
  .admin-users {
    padding: 12px;
  }
  
  .table-wrapper.desktop-only {
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
          color: #666;
        }
        
        .value {
          flex: 1;
          color: #333;
          word-break: break-word;
        }
      }
      
      .card-actions {
        margin-top: 12px;
        padding-top: 12px;
        border-top: 1px solid #f0f0f0;
        display: flex;
        flex-direction: column;
        gap: 8px;
        
        .action-buttons-row {
          display: flex;
          gap: 8px;
          width: 100%;
          
          .mobile-action-btn {
            flex: 1;
            height: 44px;
            font-size: 16px;
            margin: 0;
          }
        }
      }
    }
    
    .empty-state {
      padding: 40px 20px;
      text-align: center;
    }
  }
}

.balance-link {
  font-weight: 600;
  cursor: pointer;
  color: #409eff;
  
  &:hover {
    text-decoration: underline;
  }
}

.clickable-text {
  color: #409eff;
  cursor: pointer;
  
  &:hover {
    text-decoration: underline;
  }
}
</style>
