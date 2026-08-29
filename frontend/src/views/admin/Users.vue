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
              @input="debouncedSearch"
              @keyup.enter="searchUsers"
              @clear="searchUsers"
            />
            <el-button @click="searchUsers" class="search-button-inside" type="default" plain>
              <el-icon><Search /></el-icon>
            </el-button>
          </div>
        </div>
        <div class="mobile-filter-row">
          <el-dropdown @command="handleStatusFilter" trigger="click" placement="bottom-start" class="mobile-filter-dropdown">
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
                  <span class="danger-filter-option">设备超限</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-button size="small" type="default" plain @click="resetSearch" class="mobile-filter-reset-btn">
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
      <el-form :inline="true" :model="searchForm" class="search-form list-filter-form desktop-only">
        <el-form-item label="搜索">
          <el-input
            v-model="searchForm.keyword"
            placeholder="搜索邮箱、用户名或备注"
            class="keyword-search-input"
            clearable
            @input="debouncedSearch"
            @keyup.enter="searchUsers"
            @clear="searchUsers"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchForm.status" placeholder="选择状态" clearable class="status-filter-select" @change="searchUsers">
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
      <ResponsiveDataView
        class="admin-users-data"
        :data="users"
        :fields="mobileUserFields"
        :loading="loading"
        empty-title="暂无用户数据"
        empty-description="可调整筛选条件后重试"
      >
        <template #table>
          <div class="table-wrapper">
            <el-table 
              ref="tableRef"
              :data="users" 
              class="users-table"
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
                        <el-dropdown
                          trigger="click"
                          @command="mode => updateUserLineMode(scope.row, mode)"
                          :disabled="lineModeSaving[scope.row.id]"
                        >
                          <el-tag :type="getLineModeTagType(scope.row)" size="small" effect="plain" class="special-user-tag line-mode-tag">
                            {{ getLineModeTagText(scope.row) }}
                          </el-tag>
                          <template #dropdown>
                            <el-dropdown-menu>
                              <el-dropdown-item command="normal">普通线路</el-dropdown-item>
                              <el-dropdown-item command="both" :disabled="!hasAssignedCustomNodes(scope.row)">专线+普通线路</el-dropdown-item>
                              <el-dropdown-item command="special_only" :disabled="!hasAssignedCustomNodes(scope.row)">仅专线</el-dropdown-item>
                            </el-dropdown-menu>
                          </template>
                        </el-dropdown>
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
        </template>

        <template #header="{ item }">
          <div class="mobile-user-header">
            <div class="user-info-mobile">
              <el-avatar :size="28" :src="item.avatar">
                {{ item.username?.charAt(0)?.toUpperCase() }}
              </el-avatar>
              <button type="button" class="user-mobile-link" @click="viewUserDetails(item.id)">
                <div class="user-email-mobile">{{ item.email }}</div>
                <div class="user-name-mobile">
                  <span>{{ item.username }}</span>
                  <el-dropdown
                    trigger="click"
                    @command="mode => updateUserLineMode(item, mode)"
                    :disabled="lineModeSaving[item.id]"
                  >
                    <el-tag :type="getLineModeTagType(item)" size="small" effect="plain" class="special-user-tag line-mode-tag">
                      {{ getLineModeTagText(item) }}
                    </el-tag>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="normal">普通线路</el-dropdown-item>
                        <el-dropdown-item command="both" :disabled="!hasAssignedCustomNodes(item)">专线+普通线路</el-dropdown-item>
                        <el-dropdown-item command="special_only" :disabled="!hasAssignedCustomNodes(item)">仅专线</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </button>
            </div>
            <el-tag :type="getStatusType(item.status)" size="small">
              {{ getStatusText(item.status) }}
            </el-tag>
          </div>
        </template>

        <template #field-status="{ item }">
          <el-tag :type="getStatusType(item.status)" size="small">
            {{ getStatusText(item.status) }}
          </el-tag>
        </template>

        <template #field-balance="{ item }">
          <el-button type="text" class="balance-link" @click="viewUserBalance(item.id)">
            ¥{{ Number(item.balance || 0).toFixed(2) }}
          </el-button>
        </template>

        <template #field-device_info="{ item }">
          <span class="mobile-device-summary" :class="{ 'device-overlimit-count': isDeviceOverlimit(item) }">
            {{ item.online_devices || 0 }}/{{ item.subscription?.device_limit || 0 }}
          </span>
        </template>

        <template #field-subscription="{ item }">
          <div v-if="item.subscription" class="mobile-subscription-summary">
            <el-tag :type="getSubscriptionStatusType(item.subscription.status)" size="small" effect="plain">
              {{ getSubscriptionStatusText(item.subscription.status) }}
            </el-tag>
            <el-text
              v-if="item.subscription.days_until_expire !== null"
              size="small"
              :type="getExpireTextType(item.subscription)"
            >
              {{ getExpireText(item.subscription) }}
            </el-text>
          </div>
          <el-tag v-else type="info" size="small" effect="plain">无订阅</el-tag>
        </template>

        <template #field-notes="{ item }">
          <div class="notes-input-wrapper-mobile">
            <el-input
              v-model="item.notes"
              type="textarea"
              :rows="1"
              placeholder="点击输入备注"
              class="notes-input-mobile"
              @blur="saveNotes(item)"
              @input="debounceSaveNotes(item)"
              :maxlength="500"
            />
            <div v-if="item.savingNotes" class="saving-indicator-mobile">
              <el-icon class="is-loading"><Loading /></el-icon>
              <span>保存中...</span>
            </div>
            <div v-else-if="item.notesSaved" class="saved-indicator-mobile">
              <el-icon><CircleCheck /></el-icon>
              <span>已保存</span>
            </div>
          </div>
        </template>

        <template #actions="{ item }">
          <div class="mobile-user-actions">
            <div class="action-buttons-row">
              <el-button type="primary" @click="viewUserDetails(item.id)" class="mobile-action-btn">
                <el-icon><View /></el-icon>
                详情
              </el-button>
              <el-button type="primary" @click="editUser(item)" class="mobile-action-btn" plain>
                <el-icon><Edit /></el-icon>
                编辑
              </el-button>
              <el-button :type="item.status === 'active' ? 'warning' : 'success'" @click="toggleUserStatus(item)" class="mobile-action-btn">
                <el-icon><Switch /></el-icon>
                {{ item.status === 'active' ? '禁用' : '启用' }}
              </el-button>
            </div>
            <div class="action-buttons-row">
              <el-button type="info" @click="resetUserPassword(item)" class="mobile-action-btn">
                <el-icon><Key /></el-icon>
                重置密码
              </el-button>
              <el-button type="warning" @click="unlockUserLogin(item)" class="mobile-action-btn">
                <el-icon><Unlock /></el-icon>
                解除限制
              </el-button>
              <el-button type="danger" @click="deleteUser(item)" class="mobile-action-btn">
                <el-icon><Delete /></el-icon>
                删除
              </el-button>
            </div>
          </div>
        </template>

        <template #empty>
          <EmptyState
            title="暂无用户数据"
            description="可调整筛选条件后重试"
            action-text="重置筛选"
            :loading="loading"
            @action="resetSearch"
          />
        </template>
      </ResponsiveDataView>
      <PaginationBar
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </el-card>
    <!-- 添加/编辑用户抽屉 -->
    <AppDrawer
      v-model="showAddUserDialog"
      :title="editingUser ? '编辑用户' : '添加用户'"
      size="500px"
      mobile-size="100%"
      class="user-form-drawer"
      :loading="savingUser"
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
            <div class="form-mobile-label">邮箱 <span class="required">*</span></div>
          </template>
          <el-input v-model="userForm.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '用户名'" prop="username">
          <template v-if="isMobile">
            <div class="form-mobile-label">用户名 <span class="required">*</span></div>
          </template>
          <el-input v-model="userForm.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '密码'" prop="password" v-if="!editingUser">
          <template v-if="isMobile">
            <div class="form-mobile-label">密码 <span class="required">*</span></div>
          </template>
          <el-input v-model="userForm.password" type="password" placeholder="请输入密码" show-password />
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '密码'" prop="password" v-else>
          <template v-if="isMobile">
            <div class="form-mobile-label">密码</div>
          </template>
          <el-input v-model="userForm.password" type="password" placeholder="留空则不修改密码" show-password />
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '状态'" prop="status">
          <template v-if="isMobile">
            <div class="form-mobile-label">状态 <span class="required">*</span></div>
          </template>
          <el-select v-model="userForm.status" placeholder="选择状态" class="full-width-control">
            <el-option label="活跃" value="active" />
            <el-option label="待激活" value="inactive" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '最大设备数'" prop="device_limit" v-if="!editingUser">
          <template v-if="isMobile">
            <div class="form-mobile-label">最大设备数 <span class="required">*</span></div>
          </template>
          <el-input-number
            v-model="userForm.device_limit"
            :min="0"
            :max="100"
            placeholder="请输入最大设备数量"
            controls-position="right"
            class="full-width-control"
          />
          <div class="form-item-hint">允许用户同时使用的最大设备数量（0表示不限制）</div>
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '到期时间'" prop="expire_time" v-if="!editingUser">
          <template v-if="isMobile">
            <div class="form-mobile-label">到期时间 <span class="required">*</span></div>
          </template>
          <el-date-picker
            v-model="userForm.expire_time"
            type="datetime"
            placeholder="选择到期时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DDTHH:mm:ss"
            class="full-width-control"
            :teleported="isMobile"
            :default-time="defaultTime"
          />
          <div class="form-item-hint">订阅的到期时间，到期后用户将无法使用服务</div>
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '管理员权限'" v-if="editingUser">
          <template v-if="isMobile">
            <div class="form-mobile-label">管理员权限</div>
          </template>
          <el-switch
            v-model="userForm.is_admin"
            active-text="是管理员"
            inactive-text="普通用户"
          />
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '余额'" prop="balance" v-if="editingUser">
          <template v-if="isMobile">
            <div class="form-mobile-label">余额</div>
          </template>
          <el-input-number
            v-model="userForm.balance"
            :min="0"
            :precision="2"
            :step="10"
            class="full-width-control"
          />
          <div class="form-item-hint">用户账户余额（元）</div>
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '设备数量'" prop="device_limit" v-if="editingUser">
          <template v-if="isMobile">
            <div class="form-mobile-label">设备数量</div>
          </template>
          <el-input-number
            v-model="userForm.device_limit"
            :min="0"
            :max="100"
            class="full-width-control"
          />
          <div class="form-item-hint">允许用户同时使用的最大设备数量（0表示不限制）</div>
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '到期时间'" prop="expire_time" v-if="editingUser">
          <template v-if="isMobile">
            <div class="form-mobile-label">到期时间</div>
          </template>
          <el-date-picker
            v-model="userForm.expire_time"
            type="datetime"
            placeholder="选择到期时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DDTHH:mm:ss"
            class="full-width-control"
            :teleported="isMobile"
            :default-time="defaultTime"
          />
          <div class="form-item-hint">订阅的到期时间，到期后用户将无法使用服务</div>
        </el-form-item>
        <el-form-item :label="isMobile ? '' : '备注'" prop="note">
          <template v-if="isMobile">
            <div class="form-mobile-label">备注</div>
          </template>
          <el-input
            v-model="userForm.note"
            type="textarea"
            :rows="isMobile ? 2 : 3"
            placeholder="请输入备注信息"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <FormActionBar
          :loading="savingUser"
          :submit-text="editingUser ? '更新' : '创建'"
          @cancel="showAddUserDialog = false"
          @submit="saveUser"
        />
      </template>
    </AppDrawer>
    <!-- 用户详情抽屉 -->
    <UserDetailDialog
      :visible="showUserDialog"
      @update:visible="showUserDialog = $event"
      @custom-nodes-updated="handleCustomNodesUpdated"
      :user="selectedUser"
      :isMobile="isMobile"
    />

    <!-- 重置用户密码 -->
    <AppDialog
      v-model="showResetPasswordDialog"
      title="重置密码"
      width="460px"
      mobile-width="94%"
      :loading="resettingPassword"
    >
      <el-form
        ref="resetPasswordFormRef"
        :model="resetPasswordForm"
        :rules="resetPasswordRules"
        label-width="96px"
        class="reset-password-form"
      >
        <el-alert
          v-if="resetPasswordUser"
          type="warning"
          :closable="false"
          show-icon
          class="reset-password-alert"
        >
          <template #title>
            正在为用户 {{ resetPasswordUser.username }} 设置新密码
          </template>
        </el-alert>
        <el-form-item label="新密码" prop="password">
          <el-input
            v-model="resetPasswordForm.password"
            type="password"
            placeholder="请输入新密码（至少8位）"
            show-password
            autocomplete="new-password"
            @keyup.enter="submitResetUserPassword"
          />
          <div class="form-item-hint">
            需包含大小写字母、数字和特殊字符中的至少三种。
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <FormActionBar
          :loading="resettingPassword"
          cancel-text="取消"
          submit-text="确认重置"
          @cancel="closeResetPasswordDialog"
          @submit="submitResetUserPassword"
        />
      </template>
    </AppDialog>

    <!-- 分配专线节点对话框 -->
    <AppDialog
      v-model="showAssignNodeDialog"
      title="分配专线节点"
      width="500px"
      mobile-width="94%"
      :loading="assigningNode"
    >
      <div class="node-search-section">
        <div class="search-input-group">
          <el-input
            v-model="nodeSearchKeyword"
            placeholder="输入节点名称或地址搜索"
            clearable
            @clear="handleNodeSearchClear"
          />
          <el-button type="primary" @click="handleNodeSearch">搜索</el-button>
        </div>
        <div v-if="nodeSearchKeyword && searchedNodes.length > 0" class="search-result-tip">
          找到 {{ searchedNodes.length }} 个节点
        </div>
        <div v-else-if="nodeSearchKeyword && searchedNodes.length === 0" class="search-result-tip empty">
          未找到匹配的节点
        </div>
      </div>

      <el-form label-width="100px" class="assign-node-form">
        <el-form-item label="选择节点">
          <el-select
            v-model="selectedNodeId"
            placeholder="请选择要分配的节点"
            filterable
            class="full-width-control"
          >
            <el-option
              v-for="node in (nodeSearchKeyword ? searchedNodes : availableNodes)"
              :key="node.id"
              :label="`${node.name} - ${node.address || node.domain}`"
              :value="node.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="节点显示模式">
          <el-radio-group v-model="assignSubscriptionType" class="assign-option-group">
            <el-radio-button label="both">专线 + 普通节点</el-radio-button>
            <el-radio-button label="special_only">仅专线</el-radio-button>
          </el-radio-group>
          <div class="toggle-hint">
            {{ assignSubscriptionType === 'special_only' ? '用户订阅里只显示已分配的专线节点' : '用户订阅里同时显示普通节点和专线节点' }}
          </div>
        </el-form-item>
        <el-form-item label="设备数量限制">
          <el-radio-group v-model="assignDeviceLimitMode" class="assign-option-group">
            <el-radio-button label="system">跟随系统</el-radio-button>
            <el-radio-button label="unlimited">不限制</el-radio-button>
          </el-radio-group>
          <div class="toggle-hint">
            {{ assignDeviceLimitMode === 'unlimited' ? '专线用户不受设备数量限制' : '专线用户仍按系统套餐设备数限制' }}
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <FormActionBar
          :loading="assigningNode"
          submit-text="确定分配"
          @cancel="showAssignNodeDialog = false"
          @submit="assignCustomNode"
        />
      </template>
    </AppDialog>
  </div>
</template>
<script>
import { ref, reactive, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import {
  Plus, Edit, Delete, Search, Refresh, Switch, Key, Close, Filter,
  Connection, Monitor, Unlock, Check, Message, Bell, Loading, CircleCheck, View
} from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
import { formatDateTimeSafe, formatLocation } from '@/utils/date'
import { getDeviceTypeColor as getDeviceTypeTag, getDeviceTypeName as getDeviceTypeText } from '@/utils/device'
import { copyToClipboard } from '@/utils/textSelection'
import {
  getUserStatusType as getStatusType,
  getUserStatusText as getStatusText,
  getSubscriptionStatusType,
  getSubscriptionStatusText,
  getOrderStatusType,
  getOrderStatusText,
  getPaymentMethodText
} from '@/utils/statusMaps'
import { debounce } from '@/composables/useDebounce'
import { useMobile } from '@/composables/useMobile'
import { confirmDelete, confirmWarning } from '@/utils/confirmAction'
import PaginationBar from '@/components/PaginationBar.vue'
import AppDrawer from '@/components/AppDrawer.vue'
import AppDialog from '@/components/AppDialog.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import EmptyState from '@/components/EmptyState.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import UserDetailDialog from './components/UserDetailDialog.vue'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
dayjs.extend(timezone)
const STATUS_FILTER_MAP = {
  '': '状态筛选',
  'active': '活跃',
  'inactive': '待激活',
  'disabled': '禁用',
  'device_overlimit': '设备超限'
}
const normalizeBoolean = (val) => val === true || val === 1 || val === '1'
export default {
  name: 'AdminUsers',
  components: {
    UserDetailDialog,
    PaginationBar,
    AppDrawer,
    AppDialog,
    FormActionBar,
    EmptyState,
    ResponsiveDataView,
    Plus, Edit, Delete, Search, Refresh, Switch, Key, Close, Filter,
    Connection, Monitor, Unlock, Check, Message, Bell, Loading, CircleCheck, View
  },
  setup() {
    const loading = ref(false)
    const batchDeleting = ref(false)
    const batchOperating = ref(false)
    const users = ref([])
    const selectedUsers = ref([])
    const currentPage = ref(1)
    const pageSize = ref(10)
    const total = ref(0)
    const showAddUserDialog = ref(false)
    const showUserDialog = ref(false)
    const editingUser = ref(null)
    const selectedUser = ref(null)
    const activeBalanceTab = ref('recharge')
    const detailActiveTab = ref('devices')
    const userDevices = ref([])
    const loadingDevices = ref(false)
    const deletingDevice = ref(null)
    const userCustomNodes = ref([])
    const loadingCustomNodes = ref(false)
    const showAssignNodeDialog = ref(false)
    const availableNodes = ref([])
    const searchedNodes = ref([])
    const nodeSearchKeyword = ref('')
    const selectedNodeId = ref(null)
    const assigningNode = ref(false)
    const assignSubscriptionType = ref('both')
    const assignDeviceLimitMode = ref('system')
    const lineModeSaving = ref({})
    const showResetPasswordDialog = ref(false)
    const resetPasswordUser = ref(null)
    const resettingPassword = ref(false)
    const resetPasswordFormRef = ref()
    const resetPasswordForm = reactive({
      password: ''
    })
    const isMobile = useMobile()
    const defaultSort = ref({ prop: 'created_at', order: 'descending' })
    const tableRef = ref(null)
    // 用户表单相关
    const userFormRef = ref()
    const savingUser = ref(false)
    const defaultTime = ref(new Date(2000, 1, 1, 23, 59, 59))
    const getDefaultExpireTime = () => {
      return dayjs().tz('Asia/Shanghai').add(1, 'year').format('YYYY-MM-DDTHH:mm:ss')
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
      note: '',
      balance: 0
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
        {
          validator: (rule, value, callback) => {
            if (!editingUser.value && !value) {
              callback(new Error('请输入密码'))
              return
            }
            if (value && value.length < 6) {
              callback(new Error('密码长度不能少于6位'))
              return
            }
            callback()
          },
          trigger: 'blur'
        }
      ],
      status: [
        { required: true, message: '请选择状态', trigger: 'change' }
      ],
      device_limit: [
        { required: true, message: '请输入最大设备数量', trigger: 'blur' },
        { type: 'number', min: 0, max: 100, message: '设备数量应在0-100之间', trigger: 'blur' }
      ],
      expire_time: [
        { required: true, message: '请选择到期时间', trigger: 'change' }
      ]
    }
    const validateResetPassword = (value) => {
      if (!value) return '密码不能为空'
      if (value.length < 8) return '密码长度不能少于8位'

      let complexityCount = 0
      if (/[A-Z]/.test(value)) complexityCount += 1
      if (/[a-z]/.test(value)) complexityCount += 1
      if (/\d/.test(value)) complexityCount += 1
      if (/[!@#$%^&*()_+\-=[\]{}|;:,.<>?]/.test(value)) complexityCount += 1
      if (complexityCount < 3) return '密码需包含大小写字母、数字和特殊字符中的至少三种'

      const weakPasswords = [
        'password', '123456', '123456789', 'qwerty', 'abc123',
        'password123', 'admin', 'root', 'user', 'test',
        '12345678', 'password1', 'qwerty123', 'admin123'
      ]
      if (weakPasswords.includes(value.toLowerCase())) return '密码过于简单，请使用更复杂的密码'

      return true
    }
    const resetPasswordRules = {
      password: [
        {
          validator: (_rule, value, callback) => {
            const result = validateResetPassword(value)
            if (result === true) {
              callback()
              return
            }
            callback(new Error(result))
          },
          trigger: ['blur', 'change']
        }
      ]
    }
    const resetUserForm = () => {
      Object.assign(userForm, {
        email: '', username: '', password: '', status: 'active',
        device_limit: 5, expire_time: getDefaultExpireTime(),
        is_admin: false, is_verified: false, note: '', balance: 0
      })
      if (userFormRef.value) {
        userFormRef.value.resetFields()
      }
    }
    const onFormDrawerClosed = () => {
      editingUser.value = null
      resetUserForm()
    }
    watch(showAddUserDialog, (visible) => {
      if (!visible) onFormDrawerClosed()
    })
    watch(editingUser, async (user) => {
      if (user) {
        let status = user.status
        if (!status) {
          status = user.is_active ? 'active' : 'inactive'
        }

        // 基本信息
        Object.assign(userForm, {
          email: user.email, username: user.username,
          status, is_admin: Boolean(user.is_admin),
          is_verified: Boolean(user.is_verified),
          note: user.notes || '', password: '',
          balance: user.balance || 0,
          device_limit: user.subscription?.device_limit || 5,
          expire_time: user.subscription?.expire_time ? dayjs(user.subscription.expire_time).format('YYYY-MM-DDTHH:mm:ss') : ''
        })

        // 加载用户详情以获取订阅信息
        try {
          const response = await adminAPI.getUserDetails(user.id)
          const userData = response?.data?.success ? response.data.data : (response?.success ? response.data : response.data)
          if (userData && userData.subscription) {
            const subscription = userData.subscription
            userForm.device_limit = subscription.device_limit || 5
            if (subscription.expire_time) {
              userForm.expire_time = dayjs(subscription.expire_time).format('YYYY-MM-DDTHH:mm:ss')
            }
          }
        } catch (error) {
          console.error('加载用户详情失败:', error)
        }
      } else {
        resetUserForm()
      }
    }, { immediate: true })
    const saveUser = async () => {
      try {
        await userFormRef.value.validate()
        savingUser.value = true
        if (editingUser.value) {
          await adminAPI.updateUser(editingUser.value.id, {
            username: userForm.username, email: userForm.email,
            is_active: userForm.status === 'active',
            is_verified: Boolean(userForm.is_verified),
            is_admin: userForm.is_admin,
            notes: userForm.note || '',
            balance: userForm.balance,
            device_limit: userForm.device_limit,
            expire_time: userForm.expire_time,
            password: userForm.password || undefined
          })
          ElMessage.success('用户更新成功')
        } else {
          const response = await adminAPI.createUser({
            username: userForm.username, email: userForm.email,
            password: userForm.password,
            is_active: userForm.status === 'active',
            is_admin: false, is_verified: false,
            device_limit: userForm.device_limit || 5,
            expire_time: userForm.expire_time || getDefaultExpireTime(),
            notes: userForm.note || ''
          })
          if (response.data && response.data.success === false) {
            ElMessage.error(response.data.message || '用户创建失败')
            savingUser.value = false
            return
          }
          ElMessage.success('用户创建成功')
        }
        handleUserSaved()
      } catch (error) {
        if (error.response) {
          const data = error.response.data
          ElMessage.error(data?.message || data?.detail || '操作失败')
        } else if (error.message) {
          ElMessage.error(error.message)
        }
      } finally {
        savingUser.value = false
      }
    }
    const searchForm = reactive({
      keyword: '',
      status: '',
      date_range: '',
      start_date: '',
      end_date: '',
      sort: '',
      order: ''
    })
    const getStatusFilterText = () => STATUS_FILTER_MAP[searchForm.status] || '状态筛选'
    const formatDate = (date) => formatDateTimeSafe(date, 'YYYY-MM-DD HH:mm:ss', '')
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
    const hasAssignedCustomNodes = (user) => {
      return Boolean(user?.is_special_node_user) || Number(user?.custom_node_count || 0) > 0
    }
    const getUserLineMode = (user) => {
      if (user?.special_node_subscription_type === 'special_only') return 'special_only'
      if (user?.special_node_subscription_type === 'both' && hasAssignedCustomNodes(user)) return 'both'
      return 'normal'
    }
    const getLineModeTagText = (user) => {
      const mode = getUserLineMode(user)
      if (mode === 'normal') return '普通线路'
      return mode === 'special_only' ? '仅专线' : '专线+普通'
    }
    const getLineModeTagType = (user) => {
      const mode = getUserLineMode(user)
      if (mode === 'normal') return 'info'
      return mode === 'special_only' ? 'danger' : 'warning'
    }
    const updateUserLineMode = async (user, mode) => {
      if (!user?.id) return
      if (mode !== 'normal' && !hasAssignedCustomNodes(user)) {
        ElMessage.warning('请先给用户分配专线节点')
        return
      }
      if (getUserLineMode(user) === mode) return
      lineModeSaving.value[user.id] = true
      try {
        await adminAPI.updateUser(user.id, { special_node_subscription_type: mode })
        ElMessage.success('线路模式已更新')
        await loadUsers()
      } catch (error) {
        ElMessage.error(`线路模式更新失败: ${error.response?.data?.message || error.message}`)
      } finally {
        lineModeSaving.value[user.id] = false
      }
    }
    const mobileUserFields = computed(() => [
      { key: 'id', label: '用户ID', formatter: value => `#${value}` },
      { key: 'status', label: '状态' },
      { key: 'balance', label: '余额' },
      { key: 'device_info', label: '设备信息' },
      { key: 'subscription', label: '订阅状态' },
      { key: 'created_at', label: '注册时间', formatter: value => formatDate(value) },
      { key: 'notes', label: '备注', fullWidth: true }
    ])
    let resizeTimer = null
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
    let loadUsersSeq = 0
    const loadUsers = async () => {
      const seq = ++loadUsersSeq
      loading.value = true
      try {
        const params = buildSearchParams()
        const response = await adminAPI.getUsers(params)
        if (seq !== loadUsersSeq) return // 丢弃过时的响应
        if (response.data?.success && response.data?.data) {
          const responseData = response.data.data
          let userList = normalizeUserData(responseData.users || [])
          if (searchForm.status === 'device_overlimit') {
            userList = userList.filter(user => isDeviceOverlimit(user))
          }
          // 初始化备注状态，避免使用 deep watcher
          userList.forEach(user => {
            if (user.id && !originalNotes.has(user.id)) {
              originalNotes.set(user.id, user.notes || '')
            }
            if (!Object.prototype.hasOwnProperty.call(user, 'savingNotes')) {
              user.savingNotes = false
              user.notesSaved = false
            }
          })
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
        if (seq !== loadUsersSeq) return
        ElMessage.error(`加载用户列表失败: ${error.response?.data?.message || error.message}`)
        users.value = []
        total.value = 0
      } finally {
        if (seq === loadUsersSeq) {
          loading.value = false
        }
      }
    }
    const searchUsers = () => {
      currentPage.value = 1
      loadUsers()
    }
    // 创建防抖版本的搜索函数（500ms延迟）
    const debouncedSearch = debounce(searchUsers, 500)
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
    watch(() => searchForm.date_range, debounce((newVal) => {
      if (Array.isArray(newVal) && newVal.length === 2) {
        searchForm.start_date = newVal[0]
        searchForm.end_date = newVal[1]
      } else {
        searchForm.start_date = ''
        searchForm.end_date = ''
      }
      searchUsers() // 日期变化后自动搜索
    }, 300), { immediate: true })
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
    const savedIndicatorTimers = new Map()
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
        await adminAPI.updateUser(user.id, { notes: currentNotes })
        originalNotes.set(user.id, currentNotes)
        user.notesSaved = true
        if (savedIndicatorTimers.has(user.id)) {
          clearTimeout(savedIndicatorTimers.get(user.id))
        }
        savedIndicatorTimers.set(user.id, setTimeout(() => {
          user.notesSaved = false
          savedIndicatorTimers.delete(user.id)
        }, 2000))
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
    // 备注初始化已移至 loadUsers 完成后执行，不再需要 deep watcher
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
        } else {
          ElMessage.error('获取用户详情失败: ' + (response?.data?.message || response?.message || '未知错误'))
        }
      } catch (error) {
        ElMessage.error('获取用户详情失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const viewUserBalance = async (userId) => {
      activeBalanceTab.value = 'recharge'
      detailActiveTab.value = 'recharge'
      await viewUserDetails(userId)
    }
    const loadUserDevices = async () => {
      if (!selectedUser.value?.id) {
        userDevices.value = []
        return
      }
      loadingDevices.value = true
      try {
        const subscriptionId = selectedUser.value.id
        const response = await adminAPI.getSubscriptionDevices(subscriptionId)
        if (response && response.data) {
          const responseData = response.data
          let devices = []
          if (responseData.data && responseData.data.devices && Array.isArray(responseData.data.devices)) {
            devices = responseData.data.devices
          } else if (responseData.data && Array.isArray(responseData.data)) {
            devices = responseData.data
          } else if (responseData.devices && Array.isArray(responseData.devices)) {
            devices = responseData.devices
          } else if (Array.isArray(responseData)) {
            devices = responseData
          }
          userDevices.value = devices.map(device => ({
            id: device.id,
            device_name: device.device_name || device.name || '未知设备',
            device_type: device.device_type || device.type || 'unknown',
            ip_address: device.ip_address || device.ip || '-',
            location: device.location || '',
            last_seen: device.last_seen || device.last_access || null,
            last_access: device.last_access || device.last_seen || null
          }))
        } else {
          userDevices.value = []
        }
      } catch (error) {
        ElMessage.error('加载设备列表失败: ' + (error.response?.data?.message || error.message))
        userDevices.value = []
      } finally {
        loadingDevices.value = false
      }
    }
    const deleteDevice = async (device) => {
      try {
        await confirmDelete('设备', 1, {
          message: `确定要删除设备 "${device.device_name || '未知设备'}" 吗？删除后不可恢复。`
        })
        deletingDevice.value = device.id
        const response = await adminAPI.removeDevice(device.id)
        if (response.data && response.data.success) {
          ElMessage.success('设备删除成功')
          await loadUserDevices()
        } else {
          throw new Error(response.data?.message || '删除设备失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('删除设备失败: ' + (error.response?.data?.message || error.message))
        }
      } finally {
        deletingDevice.value = null
      }
    }
    const loadUserCustomNodes = async () => {
      if (!selectedUser.value?.user?.id) {
        userCustomNodes.value = []
        return
      }
      loadingCustomNodes.value = true
      try {
        const userId = selectedUser.value.user.id
        const response = await adminAPI.getUserCustomNodes(userId)
        if (response.data && response.data.success) {
          userCustomNodes.value = response.data.data || []
        } else {
          throw new Error(response.data?.message || '加载专线节点失败')
        }
      } catch (error) {
        ElMessage.error('加载专线节点失败: ' + (error.response?.data?.message || error.message))
        userCustomNodes.value = []
      } finally {
        loadingCustomNodes.value = false
      }
    }
    const loadAvailableNodes = async () => {
      try {
        const response = await adminAPI.getCustomNodes({ page: 1, page_size: 1000 })
        if (response.data && response.data.success) {
          availableNodes.value = response.data.data?.nodes || response.data.data || []
        }
      } catch (error) {
        ElMessage.error('加载可用节点失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const handleNodeSearch = async () => {
      if (!nodeSearchKeyword.value.trim()) {
        searchedNodes.value = []
        return
      }
      try {
        const response = await adminAPI.getCustomNodes({
          search: nodeSearchKeyword.value,
          page: 1,
          page_size: 100
        })
        if (response.data && response.data.success) {
          searchedNodes.value = response.data.data?.nodes || response.data.data || []
        }
      } catch (error) {
        ElMessage.error('搜索节点失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const handleNodeSearchClear = () => {
      nodeSearchKeyword.value = ''
      searchedNodes.value = []
    }
    const assignCustomNode = async () => {
      if (!selectedNodeId.value) {
        ElMessage.warning('请选择要分配的节点')
        return
      }
      if (!selectedUser.value?.user?.id) {
        ElMessage.error('用户信息不存在')
        return
      }
      assigningNode.value = true
      try {
        const userId = selectedUser.value.user.id
        const extraData = {
          subscription_type: assignSubscriptionType.value,
          unlimited_devices: assignDeviceLimitMode.value === 'unlimited'
        }
        const response = await adminAPI.assignCustomNodeToUser(userId, selectedNodeId.value, extraData)
        if (response.data && response.data.success) {
          ElMessage.success('专线节点分配成功')
          showAssignNodeDialog.value = false
          selectedNodeId.value = null
          nodeSearchKeyword.value = ''
          searchedNodes.value = []
          assignSubscriptionType.value = 'both'
          assignDeviceLimitMode.value = 'system'
          await loadUserCustomNodes()
          await loadUsers()
        } else {
          throw new Error(response.data?.message || '分配失败')
        }
      } catch (error) {
        ElMessage.error('分配专线节点失败: ' + (error.response?.data?.message || error.message))
      } finally {
        assigningNode.value = false
      }
    }
    const unassignCustomNode = async (nodeId) => {
      if (!selectedUser.value?.user?.id) {
        ElMessage.error('用户信息不存在')
        return
      }
      try {
        await confirmWarning('确定要取消分配此专线节点吗？', {
          confirmButtonText: '确定取消'
        })
        const userId = selectedUser.value.user.id
        const response = await adminAPI.unassignCustomNodeFromUser(userId, nodeId)
        if (response.data && response.data.success) {
          ElMessage.success('已取消分配')
          await loadUserCustomNodes()
          await loadUsers()
        } else {
          throw new Error(response.data?.message || '取消分配失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('取消分配失败: ' + (error.response?.data?.message || error.message))
        }
      }
    }
    const getResetTypeTag = (type) => {
      const typeMap = { 'manual': 'primary', 'automatic': 'info', 'admin': 'warning', 'system': 'success' }
      return typeMap[type] || 'info'
    }
    const getResetTypeText = (type) => {
      const typeMap = { 'manual': '手动重置', 'automatic': '自动重置', 'admin': '管理员重置', 'system': '系统重置' }
      return typeMap[type] || type || '未知'
    }
    const getResetByTag = (by) => {
      const byMap = { 'user': 'primary', 'admin': 'warning', 'system': 'success' }
      return byMap[by] || 'info'
    }
    const getResetByText = (by) => {
      const byMap = { 'user': '用户', 'admin': '管理员', 'system': '系统' }
      return byMap[by] || by || '未知'
    }
    const deleteUser = async (user) => {
      if (!user?.id) {
        ElMessage.warning('无效的用户ID，无法删除')
        return
      }
      try {
        await confirmDelete('用户', 1, {
          message: `确定要删除用户 "${user.username || user.email || '未知用户'}" 吗？删除后不可恢复。`
        })
      } catch {
        return
      }
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
      try {
        await confirmWarning(`确定要${action}用户 "${user.username}" 吗？`, {
          title: `确认${action}`,
          confirmButtonText: `确认${action}`
        })
      } catch {
        return
      }
      try {
        await adminAPI.updateUserStatus(user.id, newStatus)
        ElMessage.success(`用户${action}成功`)
        loadUsers()
      } catch (error) {
        ElMessage.error(`状态更新失败: ${error.response?.data?.message || error.message}`)
      }
    }
    const resetUserPassword = async (user) => {
      resetPasswordUser.value = user
      resetPasswordForm.password = ''
      showResetPasswordDialog.value = true
      await nextTick()
      resetPasswordFormRef.value?.clearValidate()
    }
    const closeResetPasswordDialog = () => {
      if (resettingPassword.value) return
      showResetPasswordDialog.value = false
      resetPasswordUser.value = null
      resetPasswordForm.password = ''
      resetPasswordFormRef.value?.clearValidate()
    }
    const submitResetUserPassword = async () => {
      if (!resetPasswordUser.value) return
      try {
        await resetPasswordFormRef.value?.validate()
        resettingPassword.value = true
        await adminAPI.resetUserPassword(resetPasswordUser.value.id, resetPasswordForm.password)
        ElMessage.success('密码重置成功')
        showResetPasswordDialog.value = false
        resetPasswordUser.value = null
        resetPasswordForm.password = ''
        resetPasswordFormRef.value?.clearValidate()
      } catch (error) {
        if (error?.response || error?.message) {
          ElMessage.error(`密码重置失败: ${error.response?.data?.message || error.message}`)
        }
      } finally {
        resettingPassword.value = false
      }
    }
    const unlockUserLogin = async (user) => {
      try {
        await confirmWarning(`确定要解除用户 "${user.username}" 的登录限制吗？这将清除该用户的所有登录失败记录。`, {
          title: '解除登录限制',
          confirmButtonText: '确认解除'
        })
      } catch {
        return
      }
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
    const handleCustomNodesUpdated = async () => {
      await loadUsers()
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
      try {
        await confirmDelete('用户', selectedUsers.value.length, {
          message: `确定要删除选中的 ${selectedUsers.value.length} 个用户吗？此操作将清空这些用户的所有数据（订阅、设备、日志等），且不可恢复。`,
          title: '确认批量删除'
        })
      } catch {
        return
      }
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
      try {
        await confirmWarning(`确定要禁用选中的 ${selectedUsers.value.length} 个用户吗？`, {
          title: '确认批量禁用',
          confirmButtonText: '确认禁用'
        })
      } catch {
        return
      }
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
      window.addEventListener('subscription-device-limit-updated', loadUsers)
    })
    onUnmounted(() => {
      window.removeEventListener('subscription-device-limit-updated', loadUsers)
      if (resizeTimer) clearTimeout(resizeTimer)
      saveTimers.forEach(timer => clearTimeout(timer))
      saveTimers.clear()
      savedIndicatorTimers.forEach(timer => clearTimeout(timer))
      savedIndicatorTimers.clear()
      originalNotes.clear()
      // 清理防抖函数
      if (debouncedSearch.cancel) debouncedSearch.cancel()
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
      detailActiveTab,
      userDevices,
      loadingDevices,
      deletingDevice,
      userCustomNodes,
      loadingCustomNodes,
      showAssignNodeDialog,
      availableNodes,
      searchedNodes,
      nodeSearchKeyword,
      selectedNodeId,
      assigningNode,
      assignSubscriptionType,
      assignDeviceLimitMode,
      lineModeSaving,
      // 用户表单
      userForm,
      userRules,
      userFormRef,
      savingUser,
      showResetPasswordDialog,
      resetPasswordUser,
      resettingPassword,
      resetPasswordFormRef,
      resetPasswordForm,
      resetPasswordRules,
      defaultTime,
      saveUser,
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
      loadUserDevices,
      deleteDevice,
      loadUserCustomNodes,
      loadAvailableNodes,
      handleNodeSearch,
      handleNodeSearchClear,
      assignCustomNode,
      unassignCustomNode,
      handleCustomNodesUpdated,
      getDeviceTypeTag,
      getDeviceTypeText,
      getResetTypeTag,
      getResetTypeText,
      getResetByTag,
      getResetByText,
      getOrderStatusType,
      getOrderStatusText,
      getPaymentMethodText,
      copyToClipboard,
      formatLocation,
      editUser,
      deleteUser,
      toggleUserStatus,
      getStatusType,
      getStatusText,
      formatDate,
      resetUserPassword,
      closeResetPasswordDialog,
      submitResetUserPassword,
      unlockUserLogin,
      getSubscriptionStatusType,
      getSubscriptionStatusText,
      getExpireTextType,
      getExpireText,
      getLineModeTagText,
      getLineModeTagType,
      hasAssignedCustomNodes,
      updateUserLineMode,
      handleSelectionChange,
      clearSelection,
      batchDeleteUsers,
      batchEnableUsers,
      batchDisableUsers,
      batchSendSubEmail,
      batchSendExpireReminder,
      isDeviceOverlimit,
      mobileUserFields,
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
.admin-users {
  @media (max-width: 768px) {
    width: 100% !important;
    max-width: 100% !important;
    margin: 0 !important;
    padding: 0 12px !important;
  }
}
/* 桌面端筛选表单：仅 ≥769px 生效；移动端强制隐藏（desktop-only 的 display:none
   会被本规则的特异性压过，导致状态筛选框溢出屏幕） */
@media (min-width: 769px) {
  .admin-users :deep(.search-form.list-filter-form) {
    display: grid !important;
    grid-template-columns: minmax(240px, 1.35fr) minmax(150px, 0.75fr) minmax(300px, 1.25fr) minmax(144px, max-content);
    align-items: end;
    column-gap: 16px;
    row-gap: 12px;
    width: 100%;
  }
}
@media (max-width: 768px) {
  .admin-users :deep(.search-form.list-filter-form) {
    display: none !important;
  }
}
.admin-users :deep(.search-form.list-filter-form .el-form-item) {
  min-width: 0;
  margin: 0 !important;
}
.admin-users :deep(.search-form.list-filter-form .el-form-item:last-child) {
  justify-self: end;
}
.admin-users :deep(.search-form.list-filter-form .el-form-item__content) {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-width: 0;
}
.admin-users :deep(.search-form.list-filter-form .el-form-item__label) {
  flex: 0 0 auto;
  padding-right: 8px;
}
.keyword-search-input,
.status-filter-select {
  width: 100%;
  min-width: 0;
}
.admin-users :deep(.search-form.list-filter-form .el-date-editor) {
  width: 100%;
  min-width: 0;
}
.admin-users :deep(.search-form.list-filter-form .el-date-editor .el-range-input) {
  min-width: 0;
}
.admin-users :deep(.search-form.list-filter-form .el-date-editor .el-range-separator) {
  flex: 0 0 auto;
  padding: 0 6px;
}
.admin-users :deep(.search-form.list-filter-form .el-button + .el-button) {
  margin-left: 0;
}
@media (max-width: 1440px) {
  .admin-users :deep(.search-form.list-filter-form) {
    grid-template-columns: minmax(240px, 1fr) minmax(150px, 0.7fr) minmax(280px, 1fr);
  }
  .admin-users :deep(.search-form.list-filter-form .el-form-item:last-child) {
    grid-column: 1 / -1;
    justify-self: start;
  }
}
@media (max-width: 1180px) {
  .admin-users :deep(.search-form.list-filter-form) {
    grid-template-columns: repeat(2, minmax(220px, 1fr));
  }
  .admin-users :deep(.search-form.list-filter-form .el-form-item:last-child) {
    justify-self: start;
  }
}
.full-width-control,
.users-table {
  width: 100%;
}
.reset-password-form {
  .reset-password-alert {
    margin-bottom: 16px;
  }

  .form-item-hint {
    margin-top: 6px;
    color: var(--el-text-color-secondary, #909399);
    font-size: 12px;
    line-height: 1.5;
  }

  @media (max-width: 768px) {
    :deep(.el-form-item) {
      display: block;
    }

    :deep(.el-form-item__label) {
      justify-content: flex-start;
      width: auto !important;
      margin-bottom: 6px;
      padding: 0;
      line-height: 1.4;
    }

    :deep(.el-form-item__content) {
      margin-left: 0 !important;
    }
  }
}
.danger-filter-option {
  color: var(--el-color-danger);
  font-weight: 600;
}
.mobile-filter-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(72px, 88px);
  gap: 8px;
  align-items: stretch;
  min-width: 0;
  width: 100%;
  .mobile-filter-dropdown {
    min-width: 0;
    max-width: 100%;
    :deep(.el-button) {
      width: 100%;
      min-width: 0;
      height: 34px;
      justify-content: center;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }
  .mobile-filter-reset-btn {
    width: 100%;
    min-width: 0;
    height: 34px;
    margin-left: 0;
  }
}

@media (max-width: 380px) {
  .mobile-filter-row {
    grid-template-columns: 1fr;
  }
}
.empty-state {
  text-align: center;
  padding: 3rem 1rem;
  color: var(--el-text-color-placeholder, #999);
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
  display: flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
  overflow: clip;
  text-overflow: ellipsis;
}
.special-user-tag {
  flex: 0 0 auto;
}
.line-mode-tag {
  cursor: pointer;
  user-select: none;
}
.user-info-mobile {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
  overflow: hidden;
  > :last-child {
    min-width: 0;
    overflow: hidden;
  }
}
.user-mobile-link {
  appearance: none;
  border: 0;
  background: transparent;
  padding: 0;
  margin: 0;
  display: block;
  min-width: 0;
  text-align: left;
  cursor: pointer;
}
.user-mobile-link:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 2px;
  border-radius: 4px;
}
.user-email-mobile {
  font-weight: 600;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-name-mobile {
  font-size: 12px;
  color: var(--el-text-color-placeholder, #999);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
  background: var(--el-fill-color-light, #f5f7fa);
  border-radius: 6px;
  transition: background-color 0.2s, border-color 0.2s;
  &.device-overlimit-alert {
    background: #fef0f0;
    border: 1px solid #f56c6c;
    animation: pulse-alert 2s ease-in-out infinite;
  }
}
@keyframes pulse-alert {
  0%, 100% {
    border-color: #f56c6c;
  }
  50% {
    border-color: #f8b4b4;
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
  .admin-users-data {
    margin-top: 10px;
    :deep(.field-full .field-value) {
      width: 100%;
      text-align: left;
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
  background-color: var(--el-fill-color-lighter, #fafafa) !important;
}
:deep(.notes-column .cell) {
  padding: 8px !important;
  background-color: var(--el-fill-color-lighter, #fafafa) !important;
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
  transition: border-color 0.2s, background-color 0.2s;
  background-color: #fff;
}
.notes-input :deep(.el-textarea__inner:hover) {
  border-color: #c0c4cc;
  background: #fbfdff;
}
.notes-input :deep(.el-textarea__inner:focus) {
  border-color: #409eff;
  background: #ffffff;
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
.mobile-user-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
}
.mobile-device-summary {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.mobile-subscription-summary {
  display: flex;
  align-items: flex-end;
  flex-direction: column;
  gap: 4px;
}
.mobile-user-actions {
  display: grid;
  gap: 8px;
  width: 100%;
  min-width: 0;

  .action-buttons-row {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
    width: 100%;
    min-width: 0;

    .mobile-action-btn {
      width: 100%;
      min-width: 0;
      max-width: 100%;
      min-height: 44px;
      font-size: 12px;
      margin: 0;
      padding: 6px 3px;
      white-space: normal;
      overflow-wrap: anywhere;
      line-height: 1.25;
      touch-action: manipulation;

      :deep(span) {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 3px;
        min-width: 0;
        max-width: 100%;
        white-space: normal;
        overflow-wrap: anywhere;
        line-height: 1.25;
      }

      :deep(.el-icon) {
        flex: 0 0 auto;
        margin-right: 0;
      }
    }
  }
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
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 6px 8px;
  font-size: 12px;
  line-height: 1.5;
  transition: border-color 0.2s, background-color 0.2s;
  background-color: #fff;
  min-height: 44px;
  touch-action: manipulation;
}
.notes-input-mobile :deep(.el-textarea__inner:hover) {
  border-color: #c0c4cc;
  background: #fbfdff;
}
.notes-input-mobile :deep(.el-textarea__inner:focus) {
  border-color: #409eff;
  background: #ffffff;
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

.drawer-content {
  .url-section {
    margin-top: 12px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .url-item {
    display: flex;
    flex-direction: column;
    gap: 6px;
    .url-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      .url-label {
        font-size: 13px;
        color: #606266;
        font-weight: 500;
      }
    }
    .url-code {
      font-size: 12px;
      font-family: monospace;
      background: var(--el-fill-color-light, #f5f7fa);
      padding: 8px 12px;
      border-radius: 4px;
      border: 1px solid #e4e7ed;
      word-break: break-all;
      color: #303133;
      line-height: 1.6;
      max-height: 120px;
      overflow-y: auto;
    }
  }
  .records-tabs {
    :deep(.el-tabs__header) {
      margin-bottom: 10px;
    }
  }

  @media (max-width: 768px) {
    padding: 15px 10px;

    :deep(.el-descriptions) {
      .el-descriptions__body {
        .el-descriptions__table {
          .el-descriptions__cell {
            padding: 6px 8px;
          }
          .el-descriptions__label {
            font-size: 12px;
            width: 70px;
            word-break: keep-all;
          }
          .el-descriptions__content {
            font-size: 12px;
            word-break: break-all;
          }
        }
      }
    }

    :deep(.el-divider) {
      margin: 15px 0;
      .el-divider__text {
        font-size: 13px;
        padding: 0 10px;
      }
    }

    .url-section {
      margin-top: 10px;
      gap: 10px;
    }

    .url-item {
      .url-header {
        margin-bottom: 5px;
        .url-label {
          font-size: 12px;
        }
        .el-button {
          padding: 5px 10px;
          font-size: 12px;
        }
      }
      .url-code {
        font-size: 10px;
        padding: 6px 8px;
        max-height: 80px;
        line-height: 1.4;
      }
    }

    :deep(.el-tabs__item) {
      font-size: 12px;
      padding: 0 10px;
      height: 44px;
      line-height: 44px;
      touch-action: manipulation;
    }

    :deep(.el-table) {
      font-size: 11px;
      .el-table__cell {
        padding: 4px 0;
      }
      .el-table__header th {
        padding: 6px 0;
        font-size: 11px;
      }
      .el-button {
        padding: 3px 8px;
        font-size: 11px;
      }
    }

    :deep(.el-tag) {
      font-size: 11px;
      padding: 0 6px;
      height: 20px;
      line-height: 20px;
    }
  }
}

.toggle-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.toggle-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 6px;
  line-height: 1.5;
}
.assign-option-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.assign-option-group :deep(.el-radio-button__inner) {
  min-height: 44px;
  border-radius: 6px !important;
  border-left: 1px solid var(--el-border-color) !important;
  display: inline-flex;
  align-items: center;
  touch-action: manipulation;
}
@media (max-width: 768px) {
  .assign-node-form {
    :deep(.el-form-item) {
      display: block;
      margin-bottom: 18px;
    }

    :deep(.el-form-item__label) {
      width: auto !important;
      justify-content: flex-start;
      margin-bottom: 6px;
      line-height: 1.4;
    }

    :deep(.el-form-item__content) {
      margin-left: 0 !important;
      width: 100%;
    }
  }

  .assign-option-group {
    display: grid;
    grid-template-columns: 1fr;
    width: 100%;
  }
  .assign-option-group :deep(.el-radio-button),
  .assign-option-group :deep(.el-radio-button__inner) {
    width: 100%;
  }
  .assign-option-group :deep(.el-radio-button__inner) {
    min-height: 44px;
    justify-content: center;
  }
}
.node-search-section {
  margin-bottom: 20px;

  .search-input-group {
    display: flex;
    gap: 10px;
    margin-bottom: 10px;
  }

  .search-result-tip {
    font-size: 13px;
    color: #67c23a;
    padding: 8px 12px;
    background: #f0f9ff;
    border-radius: 4px;

    &.empty {
      color: #909399;
      background: var(--el-fill-color-light, #f5f7fa);
    }
  }
}

@media (max-width: 768px) {
  .node-search-section {
    margin-bottom: 18px;

    .search-input-group {
      flex-direction: column;
      gap: 8px;
    }

    .search-input-group .el-button {
      width: 100%;
      min-height: 44px;
      margin-left: 0;
      touch-action: manipulation;
    }
  }
}

// 用户表单抽屉样式
.form-mobile-label {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 6px;
  line-height: 1.4;
  .required {
    color: #f56c6c;
    margin-left: 2px;
  }
}
.form-item-hint {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}
</style>
