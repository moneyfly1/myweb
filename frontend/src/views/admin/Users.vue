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
    <el-dialog 
      v-model="showAddUserDialog" 
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
          <el-input-number 
            v-model="userForm.device_limit" 
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
            :default-time="[new Date(2000, 1, 1, 23, 59, 59)]"
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
        <el-form-item label="邮箱验证" v-if="editingUser">
          <el-switch 
            v-model="userForm.is_verified" 
            active-text="已验证"
            inactive-text="未验证"
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
          <el-button @click="showAddUserDialog = false" class="mobile-action-btn">取消</el-button>
          <el-button type="primary" @click="saveUser" :loading="saving" class="mobile-action-btn">
            {{ editingUser ? '更新' : '创建' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 用户详情对话框 -->
    <el-dialog 
      v-model="showUserDialog" 
      title="用户详情" 
      :width="dialogWidth"
      class="user-detail-dialog"
      :close-on-click-modal="false"
    >
      <div v-if="selectedUser" class="user-detail-content">
        <!-- 用户基本信息 -->
        <el-descriptions :column="2" border>
          <el-descriptions-item label="用户ID">{{ selectedUser.user_info?.id || selectedUser.id }}</el-descriptions-item>
          <el-descriptions-item label="邮箱">{{ selectedUser.user_info?.email || selectedUser.email }}</el-descriptions-item>
          <el-descriptions-item label="用户名">{{ selectedUser.user_info?.username || selectedUser.username }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusType(selectedUser.user_info?.is_active !== false ? 'active' : 'inactive')">
              {{ getStatusText(selectedUser.user_info?.is_active !== false ? 'active' : 'inactive') }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="账户余额">
            <span class="balance-highlight">¥{{ ((selectedUser.user_info?.balance || selectedUser.balance || 0)).toFixed(2) }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="注册时间">{{ formatDate(selectedUser.user_info?.created_at || selectedUser.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="最后登录">{{ formatDate(selectedUser.user_info?.last_login || selectedUser.last_login) || '从未登录' }}</el-descriptions-item>
          <el-descriptions-item label="订阅数量">{{ selectedUser.statistics?.total_subscriptions || selectedUser.subscription_count || 0 }}</el-descriptions-item>
        </el-descriptions>
        
        <!-- 统计信息 -->
        <div class="user-stats" v-if="selectedUser.statistics">
          <h4>统计信息</h4>
          <el-row :gutter="20">
            <el-col :span="6">
              <el-statistic title="总消费" :value="selectedUser.statistics.total_spent" prefix="¥" />
            </el-col>
            <el-col :span="6">
              <el-statistic title="重置次数" :value="selectedUser.statistics.total_resets" />
            </el-col>
            <el-col :span="6">
              <el-statistic title="近30天重置" :value="selectedUser.statistics.recent_resets_30d" />
            </el-col>
            <el-col :span="6">
              <el-statistic title="订阅数量" :value="selectedUser.statistics.total_subscriptions" />
            </el-col>
          </el-row>
        </div>
        
        <!-- 用户订阅列表 -->
        <div class="user-subscriptions" v-if="selectedUser.subscriptions && selectedUser.subscriptions.length">
          <h4 class="section-title">
            <el-icon><Connection /></el-icon>
            订阅列表
          </h4>
          <div class="table-responsive">
            <el-table :data="selectedUser.subscriptions" size="small" style="width: 100%">
              <el-table-column prop="id" label="订阅ID" width="80" />
              <el-table-column prop="subscription_url" label="订阅地址" min-width="200" show-overflow-tooltip />
              <el-table-column prop="device_limit" label="设备限制" width="100" />
              <el-table-column prop="current_devices" label="当前设备" width="100" />
              <el-table-column prop="is_active" label="状态" width="100">
                <template #default="scope">
                  <el-tag :type="scope.row.is_active ? 'success' : 'danger'" size="small">
                    {{ scope.row.is_active ? '活跃' : '未激活' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="expire_time" label="到期时间" width="180" />
            </el-table>
          </div>
        </div>
        
        <!-- 余额变动记录（充值记录 + 消费记录） -->
        <div class="balance-records-section" :id="balanceRecordsSectionId">
          <h4 class="section-title">
            <el-icon><Wallet /></el-icon>
            余额变动记录
          </h4>
          
          <!-- 使用标签页区分充值和消费 -->
          <el-tabs v-model="activeBalanceTab" class="balance-tabs">
            <!-- 充值记录 -->
            <el-tab-pane label="充值记录" name="recharge">
              <div class="records-list" v-if="selectedUser.recharge_records && selectedUser.recharge_records.length">
                <div 
                  v-for="record in selectedUser.recharge_records" 
                  :key="record.id || record.order_no"
                  class="record-item recharge-item"
                >
                  <div class="record-header">
                    <div class="record-type">
                      <el-icon class="type-icon recharge-icon"><Plus /></el-icon>
                      <span class="type-text">充值</span>
                    </div>
                    <div class="record-amount positive">
                      +¥{{ record.amount }}
                    </div>
                  </div>
                  <div class="record-body">
                    <div class="record-info-row">
                      <span class="info-label">订单号：</span>
                      <span class="info-value">{{ record.order_no }}</span>
                    </div>
                    <div class="record-info-row">
                      <span class="info-label">支付方式：</span>
                      <span class="info-value">{{ record.payment_method || '未知' }}</span>
                    </div>
                    <div class="record-info-row">
                      <span class="info-label">状态：</span>
                      <el-tag 
                        :type="record.status === 'paid' ? 'success' : (record.status === 'pending' ? 'warning' : 'danger')" 
                        size="small"
                      >
                        {{ record.status === 'paid' ? '已支付' : (record.status === 'pending' ? '待支付' : (record.status === 'cancelled' ? '已取消' : '失败')) }}
                      </el-tag>
                    </div>
                    <div class="record-info-row">
                      <span class="info-label">IP地址：</span>
                      <span class="info-value">{{ record.ip_address || '未知' }}</span>
                    </div>
                    <div class="record-info-row">
                      <span class="info-label">创建时间：</span>
                      <span class="info-value">{{ record.created_at || '未知' }}</span>
                    </div>
                    <div class="record-info-row" v-if="record.paid_at">
                      <span class="info-label">支付时间：</span>
                      <span class="info-value">{{ record.paid_at }}</span>
                    </div>
                  </div>
                </div>
              </div>
              <el-empty v-else description="暂无充值记录" :image-size="100" />
            </el-tab-pane>
            
            <!-- 消费记录（订单） -->
            <el-tab-pane label="消费记录" name="consumption">
              <div class="records-list" v-if="selectedUser.orders && selectedUser.orders.length">
                <div 
                  v-for="order in selectedUser.orders" 
                  :key="order.id || order.order_no"
                  class="record-item consumption-item"
                >
                  <div class="record-header">
                    <div class="record-type">
                      <el-icon class="type-icon consumption-icon"><ShoppingCart /></el-icon>
                      <span class="type-text">消费</span>
                    </div>
                    <div class="record-amount negative">
                      -¥{{ order.amount }}
                    </div>
                  </div>
                  <div class="record-body">
                    <div class="record-info-row">
                      <span class="info-label">订单号：</span>
                      <span class="info-value">{{ order.order_no }}</span>
                    </div>
                    <div class="record-info-row">
                      <span class="info-label">套餐：</span>
                      <span class="info-value">{{ order.package_name || '未知' }}</span>
                    </div>
                    <div class="record-info-row">
                      <span class="info-label">支付方式：</span>
                      <span class="info-value">{{ order.payment_method || order.payment_method_name || '未知' }}</span>
                    </div>
                    <div class="record-info-row">
                      <span class="info-label">状态：</span>
                      <el-tag 
                        :type="order.status === 'paid' ? 'success' : (order.status === 'pending' ? 'warning' : 'danger')" 
                        size="small"
                      >
                        {{ order.status === 'paid' ? '已支付' : (order.status === 'pending' ? '待支付' : '已取消') }}
                      </el-tag>
                    </div>
                    <div class="record-info-row">
                      <span class="info-label">创建时间：</span>
                      <span class="info-value">{{ order.created_at || '未知' }}</span>
                    </div>
                    <div class="record-info-row" v-if="order.payment_time">
                      <span class="info-label">支付时间：</span>
                      <span class="info-value">{{ order.payment_time }}</span>
                    </div>
                  </div>
                </div>
              </div>
              <el-empty v-else description="暂无消费记录" :image-size="100" />
            </el-tab-pane>
          </el-tabs>
        </div>
        
        <!-- 最近活动 -->
        <div class="user-activities" v-if="selectedUser.recent_activities && selectedUser.recent_activities.length">
          <h4 class="section-title">
            <el-icon><Clock /></el-icon>
            最近活动
          </h4>
          <div class="table-responsive">
            <el-table :data="selectedUser.recent_activities" size="small" style="width: 100%">
              <el-table-column prop="activity_type" label="活动类型" width="120" />
              <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
              <el-table-column prop="ip_address" label="IP地址" width="120" />
              <el-table-column prop="created_at" label="时间" width="180" />
            </el-table>
          </div>
        </div>
      </div>
    </el-dialog>

  </div>
</template>

<script>
import { ref, reactive, onMounted, onUnmounted, computed, nextTick, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Plus, Edit, Delete, View, Search, Refresh, 
  Switch, Key, Close, HomeFilled, Filter,
  Wallet, ShoppingCart, Clock, Connection,
  Unlock
} from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
import { secureStorage } from '@/utils/secureStorage'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
import { formatDate as formatDateUtil } from '@/utils/date'
dayjs.extend(timezone)

export default {
  name: 'AdminUsers',
  components: {
    Plus, Edit, Delete, View, Search, Refresh, 
    Switch, Key, Close, HomeFilled, Filter,
    Wallet, ShoppingCart, Clock, Connection
  },
  setup() {
    const api = adminAPI
    const loading = ref(false)
    const saving = ref(false)
    const batchDeleting = ref(false)
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
    const balanceRecordsSectionId = 'balance-records-section'
    const isMobile = ref(window.innerWidth <= 768)
    
    // 响应式对话框宽度
    const dialogWidth = computed(() => {
      if (isMobile.value) {
        return '95%'
      } else if (window.innerWidth <= 1024) {
        return '90%'
      }
      return '1000px'
    })
    
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
      device_limit: 5, // 默认5个设备
      expire_time: getDefaultExpireTime(), // 默认一年后到期
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
        
        // 检查响应是否成功
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
          selectedUser.value = response.data.data
          showUserDialog.value = true
          // 重置标签页到充值记录
          activeBalanceTab.value = 'recharge'
        } else if (response && response.success) {
          selectedUser.value = response.data
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
      await viewUserDetails(userId)
      // 等待对话框打开后，滚动到余额记录部分
      await nextTick()
      setTimeout(() => {
        const element = document.getElementById(balanceRecordsSectionId)
        if (element) {
          element.scrollIntoView({ behavior: 'smooth', block: 'start' })
        }
      }, 300)
    }

    const editUser = (user) => {
      editingUser.value = user
      // 确保状态映射正确：后端返回的 status 可能是 "active" 或 "disabled"
      // 前端状态选择器有 "active"、"inactive"、"disabled"
      // 优先使用 user.status，如果不存在则根据 is_active 判断
      let status = user.status
      if (!status) {
        // 如果没有 status 字段，根据 is_active 判断
        status = user.is_active ? 'active' : 'inactive'
      } else if (status === 'disabled') {
        // 后端返回 "disabled" 时，前端使用 "inactive" 或 "disabled"
        // 为了保持一致性，如果 is_active 为 false，使用 "inactive"
        status = user.is_active ? 'active' : 'inactive'
      }
      Object.assign(userForm, {
        email: user.email,
        username: user.username,
        status: status,
        is_admin: Boolean(user.is_admin),  // 确保是布尔值
        is_verified: Boolean(user.is_verified),  // 确保正确读取邮箱验证状态
        note: user.note || ''
      })
      showAddUserDialog.value = true
    }

    const saveUser = async () => {
      try {
        await userFormRef.value.validate()
        saving.value = true
        
        if (editingUser.value) {
          // 转换数据格式以匹配后端API期望
          // 确保 is_active 正确转换：只有 status === 'active' 时才为 true
          const isActive = userForm.status === 'active'
          // 确保 is_verified 是明确的布尔值
          const isVerified = Boolean(userForm.is_verified)
          const userData = {
            username: userForm.username,
            email: userForm.email,
            is_active: isActive,
            is_verified: isVerified,
            is_admin: userForm.is_admin
          }
          await api.updateUser(editingUser.value.id, userData)
          ElMessage.success('用户更新成功')
        } else {
          // 转换数据格式以匹配后端API期望
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
          const response = await api.createUser(userData)
          if (response.data && response.data.success === false) {
            // 后端返回了明确的错误信息
            ElMessage.error(response.data.message || '用户创建失败')
            return
          }
          ElMessage.success('用户创建成功')
        }
        
        showAddUserDialog.value = false
        editingUser.value = null
        resetUserForm()
        loadUsers()
      } catch (error) {
        // 提取详细的错误信息
        let errorMessage = '操作失败'
        if (error.response) {
          // 后端返回的错误
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
        console.error('保存用户失败:', error)
      } finally {
        saving.value = false
      }
    }

    const deleteUser = async (user) => {
      try {
        await ElMessageBox.confirm(
          `确定要删除用户 "${user.username}" 吗？此操作不可恢复。`, 
          '确认删除', 
          { type: 'warning' }
        )
        await adminAPI.deleteUser(user.id)
        ElMessage.success('用户删除成功')
        loadUsers()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(`删除失败: ${error.response?.data?.message || error.message}`)
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

    const resetUserForm = () => {
      Object.assign(userForm, {
        email: '',
        username: '',
        password: '',
        status: 'active',
        device_limit: 5, // 重置为默认5个设备
        expire_time: getDefaultExpireTime(), // 重置为默认一年后到期
        is_admin: false,
        is_verified: false,
        note: ''
      })
      userFormRef.value?.resetFields()
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
        
        // 后端返回格式: {success: true, message: "...", data: {token: "...", user: {...}}}
        if (!response.data || !response.data.data || !response.data.data.token || !response.data.data.user) {
          ElMessage.error('登录失败：服务器返回数据不完整')
          return
        }
        
        ElMessage.success('登录成功，正在跳转...')
        
        // 保存管理员信息到 secureStorage（用于返回管理员后台）
        const adminToken = secureStorage.get('admin_token')
        const adminUser = secureStorage.get('admin_user')
        
        // 通过URL参数传递token和用户信息，在新标签页中打开用户后台
        const userToken = response.data.data.token
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
      userForm,
      userFormRef,
      userRules,
      activeBalanceTab,
      balanceRecordsSectionId,
      dialogWidth,
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
      saveUser,
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
      isDeviceOverlimit,
      // 导出图标组件供模板使用
      Search,
      Unlock
    }
  }
}
</script>

<style scoped lang="scss">
@use '@/styles/list-common.scss';

.admin-users {
  // 使用 list-container 的样式
  
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
  
  // 优化桌面端筛选框宽度
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

// mobile-filter-buttons 样式已统一在 list-common.scss 中定义
// 但为了确保与订单管理页面一致，这里添加相同的样式定义
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
          box-sizing: border-box !important;
          min-width: 0 !important;
          
          :deep(.el-input__wrapper) {
            border-radius: 10px !important;
            padding-left: 14px !important;
            padding-right: 60px !important; // 为搜索按钮留出空间
            background: rgba(255, 255, 255, 0.98) !important;
            box-shadow: 0 3px 10px rgba(0, 0, 0, 0.12) !important;
            border: 2px solid rgba(255, 255, 255, 0.4) !important;
            min-height: 48px !important;
            
            &:hover {
              background: #ffffff !important;
              border-color: rgba(255, 255, 255, 0.6) !important;
              box-shadow: 0 4px 14px rgba(0, 0, 0, 0.18) !important;
            }
            
            &.is-focus {
              background: #ffffff !important;
              border-color: #ffffff !important;
              box-shadow: 0 6px 20px rgba(0, 0, 0, 0.25) !important;
            }
          }
          
          :deep(.el-input__inner) {
            color: #1e293b !important;
            font-size: 0.95rem !important;
            font-weight: 500 !important;
            
            &::placeholder {
              color: #94a3b8 !important;
              font-weight: 400 !important;
            }
          }
        }
        
        .search-button-inside {
          position: absolute !important;
          right: 4px !important;
          top: 50% !important;
          transform: translateY(-50%) !important;
          background: rgba(255, 255, 255, 0.98) !important;
          border: 2px solid rgba(255, 255, 255, 0.4) !important;
          color: #667eea !important;
          border-radius: 8px !important;
          font-weight: 600 !important;
          box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1) !important;
          padding: 0 !important;
          height: 40px !important;
          width: 40px !important;
          min-width: 40px !important;
          max-width: 40px !important;
          transition: all 0.2s ease !important;
          display: flex !important;
          align-items: center !important;
          justify-content: center !important;
          box-sizing: border-box !important;
          z-index: 10 !important;
          
          &:hover {
            background: #ffffff !important;
            border-color: #ffffff !important;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15) !important;
            color: #5568d3 !important;
          }
          
          &:active {
            transform: translateY(-50%) scale(0.96) !important;
          }
          
          .el-icon {
            font-size: 18px !important;
            margin: 0 !important;
          }
        }
      }
    }
    
    .mobile-filter-buttons {
      display: flex !important;
      flex-direction: row !important;
      gap: 10px !important;
      align-items: stretch !important;
      width: 100% !important;
      box-sizing: border-box !important;
      flex-wrap: nowrap !important;
      
      .el-dropdown {
        flex: 1 !important;
        min-width: 0 !important;
        max-width: none !important;
        box-sizing: border-box !important;
        
        .el-button {
          width: 100% !important;
          background: rgba(255, 255, 255, 0.98) !important;
          border: 2px solid rgba(255, 255, 255, 0.4) !important;
          color: #667eea !important;
          font-weight: 600 !important;
          border-radius: 10px !important;
          box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1) !important;
          padding: 10px 12px !important;
          min-height: 44px !important;
          height: 44px !important;
          transition: all 0.2s ease !important;
          box-sizing: border-box !important;
          white-space: nowrap !important;
          overflow: hidden !important;
          text-overflow: ellipsis !important;
          
          &:hover {
            background: #ffffff !important;
            border-color: #ffffff !important;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15) !important;
          }
          
          &.el-button--primary {
            background: rgba(255, 255, 255, 0.98) !important;
            border-color: rgba(255, 255, 255, 0.6) !important;
            color: #667eea !important;
          }
          
          .el-icon {
            margin-right: 6px;
            font-size: 16px;
            flex-shrink: 0;
          }
        }
      }
      
      .el-button {
        flex: 1 !important;
        min-width: 0 !important;
        max-width: none !important;
        background: rgba(255, 255, 255, 0.98) !important;
        border: 2px solid rgba(255, 255, 255, 0.4) !important;
        color: #667eea !important;
        font-weight: 600 !important;
        border-radius: 10px !important;
        box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1) !important;
        padding: 10px 12px !important;
        min-height: 44px !important;
        height: 44px !important;
        transition: all 0.2s ease !important;
        box-sizing: border-box !important;
        white-space: nowrap !important;
        
        &:hover {
          background: #ffffff !important;
          border-color: #ffffff !important;
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15) !important;
        }
        
        &:active {
          transform: scale(0.96) !important;
        }
        
        .el-icon {
          margin-right: 6px;
          font-size: 16px;
          flex-shrink: 0;
        }
      }
    }
  }
}

// 移动端时间选择器 - 使用两个独立的日期选择器，更紧凑
.mobile-date-picker-section {
  margin-bottom: 14px;
  width: 100%;
  box-sizing: border-box;
  
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
        background: rgba(255, 255, 255, 0.98);
        box-shadow: 0 3px 10px rgba(0, 0, 0, 0.12);
        border: 2px solid rgba(255, 255, 255, 0.4);
        padding: 10px 14px;
        width: 100%;
        box-sizing: border-box;
        transition: all 0.3s ease;
        
        &:hover {
          border-color: rgba(102, 126, 234, 0.6);
          box-shadow: 0 4px 14px rgba(102, 126, 234, 0.25);
        }
        
        &.is-focus {
          border-color: #667eea;
          box-shadow: 0 6px 20px rgba(102, 126, 234, 0.35);
        }
      }
      
      :deep(.el-input__inner) {
        font-size: 0.9rem;
        color: #1e293b;
        width: 100%;
        font-weight: 500;
        
        &::placeholder {
          color: #94a3b8;
          font-weight: 400;
        }
      }
      
      :deep(.el-input__prefix) {
        color: #667eea;
        font-size: 18px;
      }
    }
    
    .date-separator {
      flex-shrink: 0;
      font-size: 0.95rem;
      color: rgba(255, 255, 255, 0.9);
      padding: 0 6px;
      font-weight: 600;
      text-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
    }
  }
}

// 移动端日期选择器弹出层样式 - 单个日期选择器，更紧凑
:deep(.mobile-date-picker-popper) {
  @media (max-width: 768px) {
    position: fixed !important;
    top: auto !important;
    bottom: 0 !important;
    left: 0 !important;
    right: 0 !important;
    width: 100% !important;
    max-width: 100% !important;
    border-radius: 16px 16px 0 0 !important;
    box-shadow: 0 -4px 20px rgba(0, 0, 0, 0.15) !important;
    z-index: 2000 !important;
    
    .el-picker__popper {
      width: 100% !important;
      max-width: 100% !important;
    }
    
    .el-date-picker {
      width: 100% !important;
      max-width: 100% !important;
      
      .el-picker-panel {
        width: 100% !important;
        max-width: 100% !important;
        border: none !important;
        box-shadow: none !important;
      }
      
      .el-picker-panel__body {
        width: 100% !important;
        max-width: 100% !important;
        padding: 16px !important;
      }
      
      .el-picker-panel__content {
        width: 100% !important;
        max-width: 100% !important;
      }
    }
  }
}

.mobile-action-buttons {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 12px;
  width: 100%;
  box-sizing: border-box;
  
  .mobile-action-btn,
  .el-button {
    width: 100%;
    height: 44px;
    margin: 0;
    font-size: 16px;
    border-radius: 6px;
    font-weight: 500;
  }
}

// mobile-action-bar 样式已统一在 list-common.scss 中定义
// 这里不再重复定义，使用统一样式
.mobile-action-bar {
  .mobile-date-picker-section {
    width: 100%;
    box-sizing: border-box;
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
}

.batch-buttons {
  display: flex;
  gap: 10px;
}

.user-email {
  display: flex;
  align-items: center;
  gap: 8px;
  
  .el-avatar {
    flex-shrink: 0;
  }
}

.email-info {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.email {
  font-weight: 500;
  color: #303133;
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 2px;
  
  .clickable-text {
    padding: 0;
    font-size: 13px;
  }
}

.username {
  font-size: 11px;
  color: #909399;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  
  .clickable-text {
    padding: 0;
    font-size: 11px;
  }
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
  border: 1px solid #e4e7ed;
}

.device-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  min-width: 40px;
}

.device-icon {
  font-size: 16px;
  margin-bottom: 2px;
}

.online-icon {
  color: #67c23a;
}

.total-icon {
  color: #409eff;
}

.device-count {
  font-weight: bold;
  font-size: 14px;
  line-height: 1;
}

.device-label {
  font-size: 10px;
  color: #909399;
  line-height: 1;
}

.device-separator {
  font-size: 14px;
  color: #c0c4cc;
  font-weight: bold;
}

/* 设备超限警报样式 */
.device-overlimit-alert {
  background: #fef0f0 !important;
  border: 2px solid #f56c6c !important;
  border-radius: 6px;
  animation: device-overlimit-flash 1.5s infinite;
  box-shadow: 0 0 10px rgba(245, 108, 108, 0.3);
}

.device-overlimit-count {
  color: #f56c6c !important;
  font-weight: bold !important;
  font-size: 16px !important;
  animation: device-overlimit-flash 1.5s infinite;
}

@keyframes device-overlimit-flash {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.8;
    transform: scale(1.05);
  }
}

.device-limit {
  margin-top: 2px;
  padding: 2px 6px;
  background: #ecf5ff;
  border-radius: 4px;
  border: 1px solid #d9ecff;
}

.subscription-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.subscription-status {
  display: flex;
  justify-content: center;
}

.no-subscription {
  display: flex;
  justify-content: center;
}

.expire-info {
  margin-top: 2px;
  text-align: center;
  padding: 2px 6px;
  background: #f0f9ff;
  border-radius: 4px;
  border: 1px solid #e1f5fe;
}

.expire-time-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.expire-date {
  font-size: 12px;
  color: #606266;
  font-weight: 500;
}

.expire-countdown {
  padding: 2px 6px;
  border-radius: 4px;
  background: #f0f9ff;
  border: 1px solid #e1f5fe;
}

.no-expire {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
}

.stat-card {
  text-align: center;
  padding: 20px;
}

.stat-number {
  font-size: 24px;
  font-weight: bold;
  color: #409eff;
  margin-bottom: 8px;
}

.stat-label {
  font-size: 14px;
  color: #666;
}

.table-wrapper {
  width: 100%;
  overflow-x: auto;
  
  :deep(.el-table) {
    min-width: 1400px;
  }
  
  :deep(.el-table__body-wrapper) {
    overflow-x: auto;
  }
}

.device-info {
  .device-stats {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    font-size: 12px;
    
    .device-item {
      display: flex;
      align-items: center;
      gap: 2px;
      
      .device-icon {
        font-size: 14px;
      }
      
      .device-count {
        font-weight: 600;
        font-size: 13px;
      }
      
      .device-label {
        display: none;
      }
    }
    
    .device-separator {
      margin: 0 2px;
      color: #999;
    }
  }
}

.action-buttons {
  display: flex;
  flex-direction: column;
  gap: 4px;
  
  .button-row {
    display: flex;
    gap: 4px;
    
    .el-button {
      flex: 1;
      padding: 5px 8px;
      font-size: 12px;
      
      .el-icon {
        font-size: 14px;
      }
    }
  }
}

/* 响应式设计 */
@media (max-width: 1600px) {
  .table-wrapper {
    :deep(.el-table) {
      min-width: 1200px;
    }
  }
}

@media (max-width: 1400px) {
  .table-wrapper {
    :deep(.el-table) {
      min-width: 1100px;
    }
  }
  
  .device-stats {
    flex-direction: column;
    gap: 2px;
    
    .device-separator {
      display: none;
    }
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
    
    :deep(.el-form-item__label) {
      font-size: 14px;
      padding-bottom: 8px;
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
  
  // 表单提示文字样式
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
  
  .device-info {
    gap: 4px;
  }
  
  .device-stats {
    padding: 2px 4px;
  }
  
  .device-icon {
    font-size: 14px;
  }
  
  .device-count {
    font-size: 12px;
  }
  
  .device-label {
    font-size: 9px;
  }
}

/* 桌面端隐藏移动端元素 */
@media (min-width: 769px) {
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

.pagination {
  margin-top: 20px;
  text-align: right;
}

.user-stats {
  margin: 20px 0;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
}

.user-stats h4 {
  margin-bottom: 15px;
  color: #606266;
}

.user-subscriptions,
.user-orders,
.user-activities {
  margin-top: 20px;
}

.user-subscriptions h4,
.user-orders h4,
.user-activities h4 {
  margin-bottom: 15px;
  color: #606266;
  border-bottom: 1px solid #ebeef5;
  padding-bottom: 8px;
}

/* 余额链接样式 */
.balance-link {
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.balance-link:hover {
  color: #66b1ff !important;
  text-decoration: underline;
}

/* 余额高亮 */
.balance-highlight {
  font-size: 16px;
  font-weight: 700;
  color: #409eff;
}

/* 用户详情对话框样式 */
.user-detail-dialog {
  :deep(.el-dialog__body) {
    max-height: 80vh;
    overflow-y: auto;
    padding: 20px;
  }
  
  @media (max-width: 768px) {
    :deep(.el-dialog) {
      width: 95% !important;
      margin: 5vh auto !important;
    }
    
    :deep(.el-dialog__body) {
      padding: 15px;
    }
  }
}

/* 章节标题样式 */
.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 20px 0 15px 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
  border-bottom: 2px solid #409eff;
  padding-bottom: 8px;
  
  .el-icon {
    font-size: 18px;
    color: #409eff;
  }
}

/* 余额变动记录区域 */
.balance-records-section {
  margin-top: 20px;
}

.balance-tabs {
  margin-top: 15px;
  
  :deep(.el-tabs__header) {
    margin-bottom: 15px;
  }
  
  :deep(.el-tabs__item) {
    font-size: 14px;
    padding: 0 20px;
  }
}

/* 记录列表样式 */
.records-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.record-item {
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 16px;
  transition: all 0.3s;
  
  &:hover {
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
    border-color: #409eff;
  }
  
  &.recharge-item {
    border-left: 4px solid #67c23a;
  }
  
  &.consumption-item {
    border-left: 4px solid #f56c6c;
  }
}

.record-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.record-type {
  display: flex;
  align-items: center;
  gap: 8px;
  
  .type-icon {
    font-size: 18px;
    
    &.recharge-icon {
      color: #67c23a;
    }
    
    &.consumption-icon {
      color: #f56c6c;
    }
  }
  
  .type-text {
    font-size: 14px;
    font-weight: 600;
    color: #303133;
  }
}

.record-amount {
  font-size: 18px;
  font-weight: 700;
  
  &.positive {
    color: #67c23a;
  }
  
  &.negative {
    color: #f56c6c;
  }
}

.record-body {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 10px;
  
  @media (max-width: 768px) {
    grid-template-columns: 1fr;
  }
}

.record-info-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 13px;
  
  .info-label {
    color: #909399;
    font-weight: 500;
    white-space: nowrap;
    min-width: 80px;
  }
  
  .info-value {
    color: #303133;
    word-break: break-all;
    flex: 1;
  }
}

/* 表格响应式容器 */
.table-responsive {
  width: 100%;
  overflow-x: auto;
  
  @media (max-width: 768px) {
    .el-table {
      font-size: 12px;
    }
  }
}

/* 统计信息响应式 */
.user-stats {
  margin-top: 20px;
  
  @media (max-width: 768px) {
    :deep(.el-col) {
      margin-bottom: 15px;
    }
  }
}

:deep(.el-table .el-table__row:hover) {
  background-color: #f5f7fa;
}

:deep(.el-button + .el-button) {
  margin-left: 8px;
}

.action-buttons {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.button-row {
  display: flex;
  gap: 6px;
  justify-content: center;
}

.action-buttons .el-button {
  margin: 0;
  padding: 6px 12px;
  font-size: 12px;
  min-width: 60px;
  flex: 1;
}

.statistics-content {
  padding: 20px 0;
}

.stat-card {
  text-align: center;
  padding: 20px;
}

.stat-number {
  font-size: 2rem;
  font-weight: bold;
  color: #409eff;
  margin-bottom: 10px;
}

.stat-label {
  color: #606266;
  font-size: 14px;
}

.subscription-management {
  padding: 20px 0;
}

.subscription-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.subscription-header h4 {
  margin: 0;
  color: #303133;
}

.chart-container {
  margin-top: 30px;
}

.chart-container h4 {
  margin-bottom: 15px;
  color: #606266;
}

/* 设备管理样式 */
.device-management {
  padding: 0;
}

.device-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 15px;
  border-bottom: 1px solid #ebeef5;
}

.device-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.device-name i {
  font-size: 16px;
  color: #409eff;
}

.ip-address {
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: #606266;
}

.user-agent {
  font-size: 12px;
  color: #909399;
  cursor: help;
}

/* 设备管理页面样式 - 复制自普通用户设备管理页面 */
.devices-container {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 2rem;
  text-align: center;
}

.page-header h3 {
  color: #1677ff;
  font-size: 1.5rem;
  margin-bottom: 0.5rem;
}

.page-header :is(p) {
  color: #666;
  font-size: 1rem;
}

.stats-card {
  margin-bottom: 2rem;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

.stats-content {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 2rem;
  padding: 1rem 0;
}

.stat-item {
  text-align: center;
}

.stat-number {
  font-size: 2.5rem;
  font-weight: bold;
  color: #1677ff;
  margin-bottom: 0.5rem;
}

.stat-label {
  color: #666;
  font-size: 0.9rem;
}

.devices-card,
.chart-card {
  margin-bottom: 2rem;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  font-weight: 600;
}

.device-name {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.device-name i {
  font-size: 1.2rem;
  color: #1677ff;
}

.ip-address {
  font-family: 'Courier New', monospace;
  color: #666;
}

.user-agent {
  color: #666;
  font-size: 0.9rem;
}

.chart-container {
  padding: 1rem 0;
}

.chart-item {
  display: flex;
  align-items: center;
  margin-bottom: 1rem;
  gap: 1rem;
}

.chart-label {
  width: 100px;
  font-weight: 500;
  color: #333;
}

.chart-bar {
  flex: 1;
  height: 20px;
  background: #f0f0f0;
  border-radius: 10px;
  overflow: hidden;
}

.chart-fill {
  height: 100%;
  background: linear-gradient(90deg, #1677ff, #4096ff);
  border-radius: 10px;
  transition: width 0.3s ease;
}

.chart-count {
  width: 60px;
  text-align: right;
  font-weight: 600;
  color: #1677ff;
}

@media (max-width: 768px) {
  .devices-container {
    padding: 10px;
  }
  
  .stats-content {
    grid-template-columns: repeat(2, 1fr);
    gap: 1rem;
  }
  
  .stat-number {
    font-size: 2rem;
  }
  
  .chart-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.5rem;
  }
  
  .chart-label {
    width: auto;
  }
  
  .chart-bar {
    width: 100%;
  }
  
  .chart-count {
    width: auto;
  }
}

.device-header h4 {
  margin: 0;
  color: #303133;
  font-size: 16px;
}

.device-actions {
  display: flex;
  gap: 10px;
}

.no-devices {
  text-align: center;
  padding: 40px 0;
}

.clickable-text {
  color: #409eff !important;
  text-decoration: none;
  padding: 0 !important;
  font-size: inherit !important;
}

.clickable-text:hover {
  color: #66b1ff !important;
  text-decoration: underline;
}

/* 移除所有输入框的圆角和阴影效果，设置为简单长方形 */
:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
}

:deep(.el-select .el-input__wrapper) {
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