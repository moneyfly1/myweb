<template>
  <div class="list-container admin-users">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>用户列表</span>
          <div class="header-actions desktop-only">
            <el-button type="primary" @click="showAddUserDialog = true">
              <el-icon><Plus /></el-icon>
              添加用户
            </el-button>
          </div>
        </div>
      </template>
      <div class="mobile-action-bar">
        <div class="mobile-search-section">
          <div class="search-input-wrapper">
            <el-input
              v-model="searchForm.keyword"
              placeholder="搜索邮箱、用户名或备注"
              class="mobile-search-input"
              clearable
              @keyup.enter="searchUsers"
            />
            <el-button @click="searchUsers" class="search-button-inside" type="default" plain>
              <el-icon><Search /></el-icon>
            </el-button>
          </div>
        </div>
        <div class="mobile-filter-buttons">
          <el-dropdown @command="handleStatusFilter" trigger="click" placement="bottom-start">
            <el-button size="small" :type="searchForm.status ? 'primary' : 'default'" plain>
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
          <el-button size="small" type="default" plain @click="resetSearch">
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
        </div>
        <div class="mobile-date-picker-section">
          <div class="date-picker-row">
            <el-date-picker
              v-model="searchForm.start_date"
              type="date"
              placeholder="开始日期"
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
        <div class="mobile-action-buttons">
          <el-button type="primary" @click="showAddUserDialog = true" class="mobile-action-btn">
            <el-icon><Plus /></el-icon>
            添加用户
          </el-button>
        </div>
      </div>
      <el-form :inline="true" :model="searchForm" class="search-form desktop-only">
        <el-form-item label="搜索">
          <el-input 
            v-model="searchForm.keyword" 
            placeholder="搜索邮箱、用户名或备注"
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
      <div class="table-wrapper desktop-only">
        <el-table 
          ref="tableRef"
          :data="users" 
          style="width: 100%" 
          v-loading="loading"
          @selection-change="handleSelectionChange"
          @sort-change="handleSortChange"
          stripe
          table-layout="auto"
          border
          :default-sort="defaultSort"
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
          <el-table-column 
            prop="balance" 
            label="余额" 
            width="100" 
            sortable="custom" 
            align="right"
            :sort-orders="['ascending', 'descending', null]"
            @sort-change="handleSortChange"
          >
            <template #default="scope">
              <el-button type="text" class="balance-link" @click="viewUserBalance(scope.row.id)">
                ¥{{ (scope.row.balance || 0).toFixed(2) }}
              </el-button>
            </template>
          </el-table-column>
          <el-table-column label="设备信息" width="120" align="center">
            <template #default="scope">
              <div class="device-info">
                <div class="device-stats" :class="{ 'device-overlimit-alert': isDeviceOverlimit(scope.row) }">
                  <el-tooltip content="已订阅设备数量" placement="top">
                    <div class="device-item online">
                      <el-icon class="device-icon online-icon"><Monitor /></el-icon>
                      <span class="device-count" :class="{ 'device-overlimit-count': isDeviceOverlimit(scope.row) }">
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
                  <el-tag :type="getSubscriptionStatusType(scope.row.subscription.status)" size="small" effect="dark">
                    {{ getSubscriptionStatusText(scope.row.subscription.status) }}
                  </el-tag>
                </div>
                <div v-if="scope.row.subscription.days_until_expire !== null" class="expire-info">
                  <el-text 
                    size="small" 
                    :type="getExpireTextType(scope.row.subscription)"
                  >
                    {{ getExpireText(scope.row.subscription) }}
                  </el-text>
                </div>
              </div>
              <div v-else class="no-subscription">
                <el-tag type="info" size="small" effect="plain">无订阅</el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="注册时间" width="180" show-overflow-tooltip sortable="custom" :sort-orders="['ascending', 'descending', null]">
            <template #default="scope">
              {{ formatDate(scope.row.created_at) }}
            </template>
          </el-table-column>
          <el-table-column prop="notes" label="备注" min-width="200" class-name="notes-column">
            <template #default="scope">
              <div class="notes-input-wrapper">
                <el-input
                  v-model="scope.row.notes"
                  type="textarea"
                  :rows="2"
                  placeholder="点击输入备注，自动保存"
                  class="notes-input"
                  @blur="saveNotes(scope.row)"
                  @input="debounceSaveNotes(scope.row)"
                  :maxlength="500"
                  show-word-limit
                />
                <div v-if="scope.row.savingNotes" class="saving-indicator">
                  <el-icon class="is-loading"><Loading /></el-icon>
                  <span>保存中...</span>
                </div>
                <div v-else-if="scope.row.notesSaved" class="saved-indicator">
                  <el-icon><CircleCheck /></el-icon>
                  <span>已保存</span>
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="到期时间" width="160" show-overflow-tooltip>
            <template #default="scope">
              <div v-if="scope.row.subscription && scope.row.subscription.expire_time" class="expire-time-info">
                <div class="expire-date">{{ formatDate(scope.row.subscription.expire_time) }}</div>
                <div class="expire-countdown">
                  <el-text size="small" :type="getExpireTextType(scope.row.subscription)">
                    {{ getExpireText(scope.row.subscription) }}
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
                  <el-button size="small" :type="scope.row.status === 'active' ? 'warning' : 'success'" @click="toggleUserStatus(scope.row)">
                    <el-icon><Switch /></el-icon>
                    {{ scope.row.status === 'active' ? '禁用' : '启用' }}
                  </el-button>
                </div>
                <div class="button-row">
                  <el-button size="small" type="info" @click="resetUserPassword(scope.row)">
                    <el-icon><Key /></el-icon>
                    重置密码
                  </el-button>
                  <el-button size="small" type="warning" @click="unlockUserLogin(scope.row)">
                    <el-icon><Unlock /></el-icon>
                    解除限制
                  </el-button>
                </div>
                <div class="button-row">
                  <el-button size="small" type="danger" @click="deleteUser(scope.row)">
                    <el-icon><Delete /></el-icon>
                    删除
                  </el-button>
                </div>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <div class="mobile-card-list" v-if="users.length > 0 && isMobile">
        <div v-for="user in users" :key="user.id" class="mobile-card">
          <div class="card-row">
            <span class="label">用户ID</span>
            <span class="value">#{{ user.id }}</span>
          </div>
          <div class="card-row">
            <span class="label">邮箱/用户名</span>
            <span class="value">
              <div class="user-info-mobile">
                <el-avatar :size="24" :src="user.avatar">
                  {{ user.username?.charAt(0)?.toUpperCase() }}
                </el-avatar>
                <div>
                  <div class="user-email-mobile">{{ user.email }}</div>
                  <div class="user-name-mobile">{{ user.username }}</div>
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
              <el-tag :type="getSubscriptionStatusType(user.subscription.status)" size="small">
                {{ getSubscriptionStatusText(user.subscription.status) }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">注册时间</span>
            <span class="value">{{ formatDate(user.created_at) }}</span>
          </div>
          <div class="card-row notes-row">
            <span class="label">备注</span>
            <div class="notes-input-wrapper-mobile">
              <el-input
                v-model="user.notes"
                type="textarea"
                :rows="2"
                placeholder="点击输入备注，自动保存"
                class="notes-input-mobile"
                @blur="saveNotes(user)"
                @input="debounceSaveNotes(user)"
                :maxlength="500"
                show-word-limit
              />
              <div v-if="user.savingNotes" class="saving-indicator-mobile">
                <el-icon class="is-loading"><Loading /></el-icon>
                <span>保存中...</span>
              </div>
              <div v-else-if="user.notesSaved" class="saved-indicator-mobile">
                <el-icon><CircleCheck /></el-icon>
                <span>已保存</span>
              </div>
            </div>
          </div>
          <div class="card-actions">
            <div class="action-buttons-row">
              <el-button type="primary" @click="editUser(user)" class="mobile-action-btn">
                <el-icon><Edit /></el-icon>
                编辑
              </el-button>
              <el-button :type="user.status === 'active' ? 'warning' : 'success'" @click="toggleUserStatus(user)" class="mobile-action-btn">
                <el-icon><Switch /></el-icon>
                {{ user.status === 'active' ? '禁用' : '启用' }}
              </el-button>
            </div>
            <div class="action-buttons-row">
              <el-button type="info" @click="resetUserPassword(user)" class="mobile-action-btn">
                <el-icon><Key /></el-icon>
                重置密码
              </el-button>
              <el-button type="warning" @click="unlockUserLogin(user)" class="mobile-action-btn">
                <el-icon><Unlock /></el-icon>
                解除限制
              </el-button>
            </div>
            <div class="action-buttons-row">
              <el-button type="danger" @click="deleteUser(user)" class="mobile-action-btn">
                <el-icon><Delete /></el-icon>
                删除
              </el-button>
            </div>
          </div>
        </div>
      </div>
      <div class="mobile-card-list" v-if="users.length === 0 && !loading && isMobile">
        <div class="empty-state">
          <i class="el-icon-user"></i>
          <p>暂无用户数据</p>
        </div>
      </div>
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
    <UserFormDialog
      v-model:visible="showAddUserDialog"
      :editingUser="editingUser"
      :isMobile="isMobile"
      @success="handleUserSaved"
    />
    <UserDetailDialog
      v-model:visible="showUserDialog"
      :user="selectedUser"
      :isMobile="isMobile"
      :initialTab="activeBalanceTab"
    />
  </div>
</template>
<script>
import { ref, reactive, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Plus, Edit, Delete, Search, Refresh, Switch, Key, Close, Filter,
  Connection, Monitor, Unlock, Check, Message, Bell, Loading, CircleCheck
} from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
import { formatDate as formatDateUtil } from '@/utils/date'
import UserFormDialog from './components/UserFormDialog.vue'
import UserDetailDialog from './components/UserDetailDialog.vue'
const STATUS_MAP = {
  active: { type: 'success', text: '活跃' },
  inactive: { type: 'warning', text: '待激活' },
  disabled: { type: 'danger', text: '禁用' }
}
const SUBSCRIPTION_STATUS_MAP = {
  active: { type: 'success', text: '活跃' },
  inactive: { type: 'info', text: '未激活' },
  expired: { type: 'danger', text: '已过期' }
}
const STATUS_FILTER_MAP = {
  '': '状态筛选',
  'active': '活跃',
  'inactive': '待激活',
  'disabled': '禁用',
  'device_overlimit': '⚠️ 设备超限'
}
const normalizeBoolean = (val) => val === true || val === 1 || val === '1'
export default {
  name: 'AdminUsers',
  components: {
    UserFormDialog,
    UserDetailDialog,
    Plus, Edit, Delete, Search, Refresh, Switch, Key, Close, Filter,
    Connection, Monitor, Unlock, Check, Message, Bell
  },
  setup() {
    const loading = ref(false)
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
    const activeBalanceTab = ref('recharge')
    const isMobile = ref(window.innerWidth <= 768)
    const defaultSort = ref({ prop: 'created_at', order: 'descending' })
    const tableRef = ref(null)
    const searchForm = reactive({
      keyword: '',
      status: '',
      date_range: '',
      start_date: '',
      end_date: '',
      sort: '',
      order: ''
    })
    const getStatusType = (status) => STATUS_MAP[status]?.type || 'info'
    const getStatusText = (status) => STATUS_MAP[status]?.text || status
    const getSubscriptionStatusType = (status) => SUBSCRIPTION_STATUS_MAP[status]?.type || 'info'
    const getSubscriptionStatusText = (status) => SUBSCRIPTION_STATUS_MAP[status]?.text || '未知'
    const getStatusFilterText = () => STATUS_FILTER_MAP[searchForm.status] || '状态筛选'
    const formatDate = (date) => formatDateUtil(date) || ''
    const isDeviceOverlimit = (user) => {
      const onlineDevices = user.online_devices || 0
      const deviceLimit = user.subscription?.device_limit || 0
      return deviceLimit > 0 && onlineDevices >= deviceLimit
    }
    const getExpireTextType = (subscription) => {
      if (subscription.is_expired) return 'danger'
      return subscription.days_until_expire <= 7 ? 'warning' : 'success'
    }
    const getExpireText = (subscription) => {
      return subscription.is_expired ? '已过期' : `${subscription.days_until_expire}天后到期`
    }
    const handleResize = () => {
      isMobile.value = window.innerWidth <= 768
    }
    const buildSearchParams = () => {
      const params = {
        page: currentPage.value,
        size: pageSize.value,
        keyword: searchForm.keyword,
        status: searchForm.status
      }
      if (searchForm.start_date && searchForm.end_date) {
        params.start_date = searchForm.start_date
        params.end_date = searchForm.end_date
      } else if (Array.isArray(searchForm.date_range) && searchForm.date_range.length === 2) {
        params.start_date = searchForm.date_range[0]
        params.end_date = searchForm.date_range[1]
      } else if (searchForm.date_range) {
        params.date_range = searchForm.date_range
      }
      if (searchForm.sort) {
        params.sort = searchForm.sort
        params.order = searchForm.order || 'asc'
      }
      return params
    }
    const normalizeUserData = (userList) => {
      return userList.map(user => ({
        ...user,
        is_active: normalizeBoolean(user.is_active),
        is_verified: normalizeBoolean(user.is_verified),
        is_admin: normalizeBoolean(user.is_admin)
      }))
    }
    const loadUsers = async () => {
      loading.value = true
      try {
        const params = buildSearchParams()
        const response = await adminAPI.getUsers(params)
        if (response.data?.success && response.data?.data) {
          const responseData = response.data.data
          let userList = normalizeUserData(responseData.users || [])
          if (searchForm.status === 'device_overlimit') {
            userList = userList.filter(user => isDeviceOverlimit(user))
          }
          users.value = userList
          total.value = searchForm.status === 'device_overlimit' ? userList.length : (responseData.total || 0)
        } else {
          users.value = []
          total.value = 0
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
    const handleStatusFilter = (command) => {
      searchForm.status = command
      searchUsers()
    }
    const handleDateRangeChange = () => {
      if (searchForm.start_date && searchForm.end_date) {
        searchForm.date_range = [searchForm.start_date, searchForm.end_date]
      } else if (!searchForm.start_date && !searchForm.end_date) {
        searchForm.date_range = ''
      }
      searchUsers()
    }
    const handleSortChange = ({ prop, order }) => {
      if (prop && order) {
        searchForm.sort = prop
        searchForm.order = order === 'ascending' ? 'asc' : 'desc'
        defaultSort.value = { prop, order }
      } else {
        searchForm.sort = ''
        searchForm.order = ''
        defaultSort.value = { prop: 'created_at', order: 'descending' }
      }
      currentPage.value = 1
      loadUsers()
    }
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
    const handleUserSaved = () => {
      showAddUserDialog.value = false
      editingUser.value = null
      loadUsers()
    }
    const saveTimers = new Map()
    const originalNotes = new Map()
    const saveNotes = async (user) => {
      if (!user || !user.id) return
      const currentNotes = user.notes || ''
      const originalNote = originalNotes.get(user.id) || ''
      if (currentNotes === originalNote) {
        user.savingNotes = false
        return
      }
      if (saveTimers.has(user.id)) {
        clearTimeout(saveTimers.get(user.id))
        saveTimers.delete(user.id)
      }
      user.savingNotes = true
      user.notesSaved = false
      try {
        await adminAPI.updateUser(user.id, { notes: currentNotes || null })
        originalNotes.set(user.id, currentNotes)
        user.notesSaved = true
        setTimeout(() => {
          user.notesSaved = false
        }, 2000)
      } catch (error) {
        ElMessage.error(`保存备注失败: ${error.response?.data?.message || error.message}`)
        user.notes = originalNote
      } finally {
        user.savingNotes = false
      }
    }
    const debounceSaveNotes = (user) => {
      if (!user || !user.id) return
      if (!originalNotes.has(user.id)) {
        originalNotes.set(user.id, user.notes || '')
      }
      if (saveTimers.has(user.id)) {
        clearTimeout(saveTimers.get(user.id))
      }
      const timer = setTimeout(() => {
        saveNotes(user)
        saveTimers.delete(user.id)
      }, 1000)
      saveTimers.set(user.id, timer)
    }
    watch(users, (newUsers) => {
      newUsers.forEach(user => {
        if (user.id && !originalNotes.has(user.id)) {
          originalNotes.set(user.id, user.notes || '')
        }
        if (!user.hasOwnProperty('savingNotes')) {
          user.savingNotes = false
          user.notesSaved = false
        }
      })
    }, { deep: true })
    const editUser = (user) => {
      editingUser.value = user
      showAddUserDialog.value = true
    }
    const viewUserDetails = async (userId) => {
      try {
        const response = await adminAPI.getUserDetails(userId)
        const userData = response?.data?.success ? response.data.data : (response?.success ? response.data : null)
        if (userData) {
          selectedUser.value = userData
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
    const handleConfirmAction = async (message, title, type = 'warning') => {
      try {
        await ElMessageBox.confirm(message, title, { type })
        return true
      } catch {
        return false
      }
    }
    const deleteUser = async (user) => {
      if (!user?.id) {
        ElMessage.warning('无效的用户ID，无法删除')
        return
      }
      const confirmed = await handleConfirmAction(
        `确定要删除用户 "${user.username || user.email || '未知用户'}" 吗？此操作不可恢复。`,
        '确认删除'
      )
      if (!confirmed) return
      try {
        await adminAPI.deleteUser(user.id)
        ElMessage.success('用户删除成功')
        loadUsers()
      } catch (error) {
        ElMessage.error(`删除失败: ${error.response?.data?.message || error.message || '删除失败'}`)
      }
    }
    const toggleUserStatus = async (user) => {
      const newStatus = user.status === 'active' ? 'disabled' : 'active'
      const action = newStatus === 'active' ? '启用' : '禁用'
      const confirmed = await handleConfirmAction(
        `确定要${action}用户 "${user.username}" 吗？`,
        `确认${action}`
      )
      if (!confirmed) return
      try {
        await adminAPI.updateUserStatus(user.id, newStatus)
        ElMessage.success(`用户${action}成功`)
        loadUsers()
      } catch (error) {
        ElMessage.error(`状态更新失败: ${error.response?.data?.message || error.message}`)
      }
    }
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
              if (!value) return '密码不能为空'
              if (value.length < 6) return '密码长度不能少于6位'
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
      const confirmed = await handleConfirmAction(
        `确定要解除用户 "${user.username}" 的登录限制吗？这将清除该用户的所有登录失败记录。`,
        '解除登录限制'
      )
      if (!confirmed) return
      try {
        const result = await adminAPI.unlockUserLogin(user.id)
        ElMessage.success(result.message || '登录限制已解除')
      } catch (error) {
        ElMessage.error(`解除限制失败: ${error.response?.data?.message || error.message}`)
      }
    }
    const handleSelectionChange = (selection) => {
      selectedUsers.value = selection
    }
    const clearSelection = () => {
      selectedUsers.value = []
    }
    const executeBatchOperation = async (operation, successMessage) => {
      if (selectedUsers.value.length === 0) {
        ElMessage.warning('请先选择用户')
        return
      }
      try {
        batchOperating.value = true
        const userIds = selectedUsers.value.map(user => user.id)
        const response = await operation(userIds)
        if (response.data?.success !== false) {
          const data = response.data?.data || {}
          const successCount = data.success_count || selectedUsers.value.length
          const failCount = data.fail_count || 0
          const message = successMessage || response.data?.message || '操作成功'
          ElMessage.success(failCount > 0 ? `${message}，成功 ${successCount} 个，失败 ${failCount} 个` : message)
          clearSelection()
          loadUsers()
        } else {
          ElMessage.error(response.data?.message || '操作失败')
        }
      } catch (error) {
        ElMessage.error(`操作失败: ${error.response?.data?.message || error.message}`)
      } finally {
        batchOperating.value = false
      }
    }
    const checkAdminUsers = (action) => {
      const adminUsers = selectedUsers.value.filter(user => user.is_admin)
      if (adminUsers.length > 0) {
        ElMessage.error(`不能${action}管理员用户`)
        return false
      }
      return true
    }
    const batchDeleteUsers = async () => {
      if (selectedUsers.value.length === 0) {
        ElMessage.warning('请先选择要删除的用户')
        return
      }
      if (!checkAdminUsers('删除')) return
      const confirmed = await handleConfirmAction(
        `确定要删除选中的 ${selectedUsers.value.length} 个用户吗？此操作将清空这些用户的所有数据（订阅、设备、日志等），且不可恢复。`,
        '确认批量删除',
        'warning'
      )
      if (!confirmed) return
      try {
        batchDeleting.value = true
        const userIds = selectedUsers.value.map(user => user.id)
        await adminAPI.batchDeleteUsers(userIds)
        ElMessage.success(`成功删除 ${selectedUsers.value.length} 个用户`)
        clearSelection()
        loadUsers()
      } catch (error) {
        ElMessage.error(`批量删除失败: ${error.response?.data?.message || error.message}`)
      } finally {
        batchDeleting.value = false
      }
    }
    const batchEnableUsers = () => {
      executeBatchOperation(
        (userIds) => adminAPI.batchEnableUsers(userIds),
        `成功启用 ${selectedUsers.value.length} 个用户`
      )
    }
    const batchDisableUsers = async () => {
      if (selectedUsers.value.length === 0) {
        ElMessage.warning('请先选择要禁用的用户')
        return
      }
      if (!checkAdminUsers('禁用')) return
      const confirmed = await handleConfirmAction(
        `确定要禁用选中的 ${selectedUsers.value.length} 个用户吗？`,
        '确认批量禁用'
      )
      if (!confirmed) return
      await executeBatchOperation(
        (userIds) => adminAPI.batchDisableUsers(userIds),
        `成功禁用 ${selectedUsers.value.length} 个用户`
      )
    }
    const batchSendSubEmail = () => {
      executeBatchOperation(
        (userIds) => adminAPI.batchSendSubEmail(userIds),
        `成功发送 ${selectedUsers.value.length} 封邮件`
      )
    }
    const batchSendExpireReminder = () => {
      executeBatchOperation(
        (userIds) => adminAPI.batchSendExpireReminder(userIds),
        `成功发送 ${selectedUsers.value.length} 封提醒邮件`
      )
    }
    onMounted(() => {
      loadUsers()
      window.addEventListener('resize', handleResize)
      window.addEventListener('subscription-device-limit-updated', loadUsers)
    })
    onUnmounted(() => {
      window.removeEventListener('resize', handleResize)
      window.removeEventListener('subscription-device-limit-updated', loadUsers)
      saveTimers.forEach(timer => clearTimeout(timer))
      saveTimers.clear()
      originalNotes.clear()
    })
    return {
      isMobile,
      loading,
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
      handleSortChange,
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
      getExpireTextType,
      getExpireText,
      handleSelectionChange,
      clearSelection,
      batchDeleteUsers,
      batchEnableUsers,
      batchDisableUsers,
      batchSendSubEmail,
      batchSendExpireReminder,
      isDeviceOverlimit,
      handleUserSaved,
      saveNotes,
      debounceSaveNotes,
      defaultSort,
      Loading,
      CircleCheck
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
    margin: 0 !important;
    padding: 0 12px !important;
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
    .el-select, .el-date-editor, .el-input {
      .el-input__wrapper {
        border: 1px solid #dcdfe6;
        box-shadow: none;
        padding: 0 11px;
      }
      .el-input__inner {
        border: none !important;
        background: transparent !important;
        box-shadow: none !important;
        outline: none !important;
      }
    }
  }
}
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
            border: 1px solid #dcdfe6 !important;
            border-radius: 10px !important;
            padding-right: 60px !important;
            min-height: 48px !important;
            box-shadow: none !important;
            background: #fff !important;
            .el-input__inner {
              border: none !important;
              box-shadow: none !important;
              background: transparent !important;
            }
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
        border: 1px solid #dcdfe6;
        box-shadow: none;
        min-height: 48px;
        border-radius: 10px;
        width: 100%;
        background: #fff;
        .el-input__inner {
          border: none;
          box-shadow: none;
          background: transparent;
        }
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
  overflow: clip;
  text-overflow: ellipsis;
}
.user-info-mobile {
  display: flex;
  align-items: center;
  gap: 8px;
}
.user-email-mobile {
  font-weight: 600;
}
.user-name-mobile {
  font-size: 0.85rem;
  color: #999;
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
.no-subscription, .no-expire {
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
.balance-link, .clickable-text {
  color: #409eff;
  cursor: pointer;
  font-weight: 600;
  &:hover {
    text-decoration: underline;
  }
}
:deep(.notes-column) {
  background-color: #fafafa !important;
}
:deep(.notes-column .cell) {
  padding: 8px !important;
  background-color: #fafafa !important;
}
.notes-input-wrapper {
  position: relative;
  width: 100%;
  padding: 4px 0;
}
.notes-input {
  width: 100%;
}
.notes-input :deep(.el-textarea__inner) {
  border: 2px solid #e4e7ed;
  border-radius: 6px;
  padding: 8px 12px;
  font-size: 13px;
  line-height: 1.5;
  transition: all 0.3s;
  background-color: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}
.notes-input :deep(.el-textarea__inner:hover) {
  border-color: #c0c4cc;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}
.notes-input :deep(.el-textarea__inner:focus) {
  border-color: #409eff;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.1);
  outline: none;
}
.notes-input :deep(.el-input__count) {
  background-color: transparent;
  color: #909399;
  font-size: 12px;
}
.saving-indicator,
.saved-indicator {
  position: absolute;
  right: 8px;
  top: 8px;
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #909399;
  pointer-events: none;
  z-index: 10;
}
.saving-indicator {
  color: #409eff;
}
.saved-indicator {
  color: #67c23a;
  animation: fadeInOut 2s ease-in-out;
}
@keyframes fadeInOut {
  0%, 100% { opacity: 0; }
  10%, 90% { opacity: 1; }
}
.saving-indicator .el-icon,
.saved-indicator .el-icon {
  font-size: 14px;
}
.notes-row {
  margin-top: 12px;
}
.notes-input-wrapper-mobile {
  position: relative;
  width: 100%;
  margin-top: 8px;
}
.notes-input-mobile {
  width: 100%;
}
.notes-input-mobile :deep(.el-textarea__inner) {
  border: 2px solid #e4e7ed;
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 14px;
  line-height: 1.6;
  transition: all 0.3s;
  background-color: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  min-height: 60px;
}
.notes-input-mobile :deep(.el-textarea__inner:hover) {
  border-color: #c0c4cc;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}
.notes-input-mobile :deep(.el-textarea__inner:focus) {
  border-color: #409eff;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.1);
  outline: none;
}
.notes-input-mobile :deep(.el-input__count) {
  background-color: transparent;
  color: #909399;
  font-size: 12px;
}
.saving-indicator-mobile,
.saved-indicator-mobile {
  position: absolute;
  right: 12px;
  top: 10px;
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #909399;
  pointer-events: none;
  z-index: 10;
  background: rgba(255, 255, 255, 0.9);
  padding: 2px 6px;
  border-radius: 4px;
}
.saving-indicator-mobile {
  color: #409eff;
}
.saved-indicator-mobile {
  color: #67c23a;
  animation: fadeInOut 2s ease-in-out;
}
.saving-indicator-mobile .el-icon,
.saved-indicator-mobile .el-icon {
  font-size: 14px;
}
</style>