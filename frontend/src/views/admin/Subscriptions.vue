<template>
  <div class="list-container admin-subscriptions">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>订阅列表</span>
          <div class="header-actions desktop-only">
            <el-button type="success" @click="exportSubscriptions">
              <el-icon><Download /></el-icon>
              导出订阅
            </el-button>
            <el-button type="warning" @click="clearAllDevices">
              <el-icon><Delete /></el-icon>
              清理设备数
            </el-button>
            <el-button type="info" @click="showColumnSettings = true">
              <el-icon><Setting /></el-icon>
              列设置
            </el-button>
            <el-button type="primary" @click="sortByApple">
              <el-icon><Apple /></el-icon>
              通用订阅
            </el-button>
            <el-button type="success" @click="sortByOnline">
              <el-icon><Monitor /></el-icon>
              在线
            </el-button>
            <el-button type="default" @click="sortByCreatedTime">
              最新↓<el-icon class="el-icon--right"><arrow-down /></el-icon>
            </el-button>
            <el-dropdown @command="handleSortCommand">
              <el-button type="default">
                更多排序<el-icon class="el-icon--right"><arrow-down /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="add_time_desc">添加时间 (降序)</el-dropdown-item>
                  <el-dropdown-item command="add_time_asc">添加时间 (升序)</el-dropdown-item>
                  <el-dropdown-item command="expire_time_desc">到期时间 (降序)</el-dropdown-item>
                  <el-dropdown-item command="expire_time_asc">到期时间 (升序)</el-dropdown-item>
                  <el-dropdown-item command="device_count_desc">设备数量 (降序)</el-dropdown-item>
                  <el-dropdown-item command="device_count_asc">设备数量 (升序)</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </template>
      <div class="mobile-action-bar">
        <div class="mobile-search-section">
          <div class="search-input-wrapper">
            <el-input
              v-model="searchForm.keyword"
              placeholder="输入QQ、订阅地址或旧订阅地址查询"
              class="mobile-search-input"
              clearable
              @keyup.enter="searchSubscriptions"
            />
            <el-button 
              @click="searchSubscriptions" 
              class="search-button-inside"
              type="default"
              plain
            >
              <el-icon><Search /></el-icon>
            </el-button>
          </div>
        </div>
        <div class="mobile-quick-actions">
          <div class="quick-sort-buttons">
            <el-button 
              size="small" 
              :type="currentSort === 'add_time_desc' ? 'primary' : 'default'"
              @click="sortByCreatedTime"
              plain
            >
              <el-icon><Clock /></el-icon>
              最新
            </el-button>
            <el-button 
              size="small" 
              :type="currentSort.includes('apple') ? 'primary' : 'default'"
              @click="sortByApple"
              plain
            >
              <el-icon><Apple /></el-icon>
              通用订阅
            </el-button>
            <el-button 
              size="small" 
              :type="currentSort.includes('online') ? 'primary' : 'default'"
              @click="sortByOnline"
              plain
            >
              <el-icon><Monitor /></el-icon>
              在线
            </el-button>
            <el-dropdown @command="handleSortCommand" trigger="click">
              <el-button size="small" type="default" plain>
                <el-icon><Sort /></el-icon>
                更多
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="add_time_desc">添加时间 (降序)</el-dropdown-item>
                  <el-dropdown-item command="add_time_asc">添加时间 (升序)</el-dropdown-item>
                  <el-dropdown-item command="expire_time_desc">到期时间 (降序)</el-dropdown-item>
                  <el-dropdown-item command="expire_time_asc">到期时间 (升序)</el-dropdown-item>
                  <el-dropdown-item command="device_count_desc">设备数量 (降序)</el-dropdown-item>
                  <el-dropdown-item command="device_count_asc">设备数量 (升序)</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
          <div class="action-buttons-group">
            <el-dropdown @command="handleActionCommand" trigger="click" placement="bottom-end">
              <el-button type="primary" size="small" plain>
                <el-icon><Operation /></el-icon>
                更多操作
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="export">
                    <el-icon><Download /></el-icon>
                    导出订阅
                  </el-dropdown-item>
                  <el-dropdown-item command="clearDevices">
                    <el-icon><Delete /></el-icon>
                    清理设备数
                  </el-dropdown-item>
                  <el-dropdown-item command="columnSettings">
                    <el-icon><Setting /></el-icon>
                    列设置
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </div>
      <el-form :inline="true" :model="searchForm" class="search-form list-filter-form desktop-only">
        <el-form-item label="搜索">
          <el-input 
            v-model="searchForm.keyword" 
            placeholder="输入QQ、订阅地址或旧订阅地址进行搜索"
            style="width: 300px;"
            clearable
            @keyup.enter="searchSubscriptions"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchForm.status" placeholder="选择状态" clearable style="width: 120px;">
            <el-option label="全部" value="" />
            <el-option label="活跃" value="active" />
            <el-option label="已过期" value="expired" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="searchSubscriptions">
            <el-icon><Search /></el-icon>
            搜索
          </el-button>
          <el-button @click="resetSearch">
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
        </el-form-item>
      </el-form>
      <div class="batch-actions" v-if="selectedSubscriptions.length > 0">
        <div class="batch-info">
          <span>已选择 {{ selectedSubscriptions.length }} 个订阅</span>
        </div>
        <div class="batch-buttons">
          <el-button type="success" @click="batchEnableSubscriptions" :loading="batchOperating">
            <el-icon><Check /></el-icon>
            批量启用
          </el-button>
          <el-button type="warning" @click="batchDisableSubscriptions" :loading="batchOperating">
            <el-icon><Close /></el-icon>
            批量禁用
          </el-button>
          <el-button type="primary" @click="batchResetSubscriptions" :loading="batchOperating">
            <el-icon><Refresh /></el-icon>
            批量重置
          </el-button>
          <el-button type="info" @click="batchSendAdminSubEmail" :loading="batchOperating">
            <el-icon><Message /></el-icon>
            发送订阅邮件
          </el-button>
          <el-button type="danger" @click="batchDeleteSubscriptions" :loading="batchOperating">
            <el-icon><Delete /></el-icon>
            批量删除
          </el-button>
          <el-button @click="clearSelection">
            <el-icon><Close /></el-icon>
            取消选择
          </el-button>
        </div>
      </div>
      <div class="table-wrapper">
        <el-table 
          ref="tableRef"
          :data="subscriptions" 
          style="width: 100%" 
          v-loading="loading"
          @selection-change="handleSelectionChange"
          @sort-change="handleSortChange"
          row-key="id"
          stripe
          border
        >
        <el-table-column type="selection" width="55" />
        <el-table-column
          v-if="visibleColumns.includes('qq')"
          label="用户"
          width="150"
          fixed="left"
        >
          <template #default="scope">
            <div class="qq-info">
              <div class="qq-email">{{ scope.row.user?.email || '未知' }}</div>
              <div class="qq-username">{{ scope.row.user?.username || '-' }}</div>
              <el-button
                size="small"
                type="success"
                @click="showUserDetails(scope.row)"
                class="detail-btn"
              >
                详情
              </el-button>
            </div>
          </template>
        </el-table-column>
        <el-table-column 
          v-if="visibleColumns.includes('expire_time')" 
          label="结束时间" 
          width="160"
          prop="expire_time"
          sortable="custom"
          :sort-orders="['descending', 'ascending', null]"
        >
          <template #default="scope">
            <div 
              class="expire-time-section"
              :class="{ 'expire-time-expired': isExpired(scope.row) }"
            >
              <el-date-picker
                v-model="scope.row.expire_time"
                type="date"
                placeholder="年/月/日"
                format="YYYY/MM/DD"
                value-format="YYYY-MM-DD"
                size="small"
                @change="updateExpireTime(scope.row)"
                class="expire-picker"
              />
              <div class="quick-buttons">
                <el-button size="small" @click="addTime(scope.row, 180)">+半年</el-button>
                <el-button size="small" @click="addTime(scope.row, 365)">+一年</el-button>
                <el-button size="small" @click="addTime(scope.row, 730)">+两年</el-button>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column 
          v-if="visibleColumns.includes('qr_code')" 
          label="二维码" 
          width="100" 
          align="center"
        >
          <template #default="scope">
            <div class="qr-code-section">
              <div 
                class="qr-code" 
                @click="showQRCode(scope.row)"
                v-if="scope.row.subscription_url || scope.row.universal_url"
              >
                <img :src="scope.row.qr_code_url" alt="QR Code" />
              </div>
              <el-text v-else type="info" size="small">无订阅</el-text>
            </div>
          </template>
        </el-table-column>
        <el-table-column
          v-if="visibleColumns.includes('sub_urls')"
          label="订阅链接"
          min-width="200"
        >
          <template #default="scope">
            <div class="sub-urls-stacked">
              <div class="sub-url-row" v-if="scope.row.universal_url">
                <span class="sub-url-label">通用</span>
                <el-link
                  @click="copyToClipboard(scope.row.universal_url)"
                  type="primary"
                  class="link-text copy-link"
                  :title="'点击复制: ' + scope.row.universal_url"
                >
                  {{ scope.row.universal_url }}
                </el-link>
              </div>
              <div class="sub-url-row" v-if="scope.row.clash_url">
                <span class="sub-url-label">Clash</span>
                <el-link
                  @click="copyToClipboard(scope.row.clash_url)"
                  type="primary"
                  class="link-text copy-link"
                  :title="'点击复制: ' + scope.row.clash_url"
                >
                  {{ scope.row.clash_url }}
                </el-link>
              </div>
              <el-text v-if="!scope.row.universal_url && !scope.row.clash_url" type="info" size="small">未配置</el-text>
            </div>
          </template>
        </el-table-column>
        <el-table-column 
          v-if="visibleColumns.includes('created_at')" 
          label="添加时间" 
          width="160"
          prop="created_at"
          sortable="custom"
          :sort-orders="['descending', 'ascending', null]"
        >
          <template #default="scope">
            <div class="created-time">
              {{ formatDate(scope.row.created_at) }}
            </div>
          </template>
        </el-table-column>
        <el-table-column
          v-if="visibleColumns.includes('sub_count')"
          label="订阅次数"
          width="110"
          align="center"
          prop="apple_count"
          sortable="custom"
          :sort-orders="['descending', 'ascending', null]"
        >
          <template #default="scope">
            <div class="sub-count-stacked">
              <div class="sub-count-row">
                <span class="sub-count-label">通用</span>
                <el-tag type="info" size="small">{{ scope.row.apple_count || 0 }}</el-tag>
              </div>
              <div class="sub-count-row">
                <span class="sub-count-label">Clash</span>
                <el-tag type="warning" size="small">{{ scope.row.clash_count || 0 }}</el-tag>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column 
          v-if="visibleColumns.includes('online_devices')" 
          label="在线" 
          width="70" 
          align="center"
          prop="online_devices"
          sortable="custom"
          :sort-orders="['descending', 'ascending', null]"
        >
          <template #default="scope">
            <el-tooltip content="当前在线设备数" placement="top">
              <el-tag type="success" size="small">{{ scope.row.online_devices || 0 }}</el-tag>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column 
          v-if="visibleColumns.includes('device_limit')" 
          label="最大设备数" 
          width="130"
          prop="device_limit"
          sortable="custom"
          :sort-orders="['descending', 'ascending', null]"
        >
          <template #default="scope">
            <div 
              class="device-limit-section"
              :class="{ 'device-limit-overlimit': isDeviceOverlimit(scope.row) }"
            >
              <el-input-number
                v-model="scope.row.device_limit"
                :min="0"
                :max="999"
                size="small"
                @change="updateDeviceLimit(scope.row)"
                class="device-limit-input"
              />
              <div class="quick-device-buttons">
                <el-button size="small" @click="addDeviceLimit(scope.row, 1)">+1</el-button>
                <el-button size="small" @click="addDeviceLimit(scope.row, 5)">+5</el-button>
                <el-button size="small" @click="addDeviceLimit(scope.row, 10)">+10</el-button>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column
          v-if="visibleColumns.includes('notes')"
          label="备注"
          min-width="180"
          class-name="notes-column"
        >
          <template #default="scope">
            <div class="notes-input-wrapper">
              <el-input
                v-model="scope.row.user_notes"
                type="textarea"
                :rows="2"
                placeholder="点击输入备注，自动保存"
                class="notes-input"
                @blur="saveSubNotes(scope.row)"
                @input="debounceSaveSubNotes(scope.row)"
                :maxlength="500"
                show-word-limit
              />
              <div v-if="scope.row.savingNotes" class="saving-indicator">
                <el-icon class="is-loading"><Loading /></el-icon>
                <span>保存中...</span>
              </div>
              <div v-else-if="scope.row.notesSaved" class="saved-indicator">
                <el-icon><CircleCheckFilled /></el-icon>
                <span>已保存</span>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column
          v-if="visibleColumns.includes('actions')"
          label="操作" 
          width="220" 
          fixed="right"
        >
          <template #default="scope">
            <div class="action-buttons">
              <div class="button-row">
                <el-button size="small" type="success" @click="goToUserBackend(scope.row)">
                  后台
                </el-button>
                <el-button size="small" type="primary" @click="resetSubscription(scope.row)">
                  重置
                </el-button>
                <el-button size="small" type="info" @click="sendSubscriptionEmail(scope.row)">
                  发送
                </el-button>
              </div>
              <div class="button-row">
                <el-button 
                  size="small" 
                  :type="scope.row.is_active ? 'warning' : 'success'"
                  @click="toggleSubscriptionStatus(scope.row)"
                >
                  {{ scope.row.is_active ? '禁用' : '启用' }}
                </el-button>
                <el-button 
                  size="small" 
                  type="danger" 
                  @click="deleteUser(scope.row)"
                  :disabled="!scope.row.user?.id || scope.row.user?.id === 0 || scope.row.user?.deleted"
                >
                  删除
                </el-button>
                <el-button size="small" type="danger" @click="clearUserDevices(scope.row)">
                  清理
                </el-button>
              </div>
            </div>
          </template>
        </el-table-column>
      </el-table>
      </div>
      <div class="mobile-card-list" v-if="subscriptions.length > 0">
        <div 
          v-for="subscription in subscriptions" 
          :key="subscription.id"
          class="mobile-card sub-card"
        >
          <div class="sub-card-header">
            <div class="sub-user-info">
              <el-avatar :size="36" :src="subscription.user?.avatar">
                {{ subscription.user?.username?.charAt(0)?.toUpperCase() || 'U' }}
              </el-avatar>
              <div class="sub-user-meta">
                <div class="sub-user-email">
                  {{ subscription.user?.email || subscription.user?.username || '未知用户' }}
                </div>
                <div class="sub-user-id">
                  ID: {{ subscription.user?.id || subscription.user_id || subscription.id }} · 
                  <el-tag :type="getSubscriptionStatusType(subscription.status)" size="small" effect="plain" style="border: none; padding: 0 4px;">
                    {{ getSubscriptionStatusText(subscription.status) }}
                  </el-tag>
                  <el-tag v-if="subscription.user?.deleted" type="danger" size="small" style="margin-left: 4px;">已删除</el-tag>
                </div>
              </div>
            </div>
            <el-button size="small" type="success" plain @click="goToUserBackend(subscription)" class="sub-goto-btn">
              进入后台
            </el-button>
          </div>
          <div class="sub-section" :class="{ 'expire-time-expired': isExpired(subscription) }">
            <div class="sub-section-row">
              <span class="sub-section-icon"><el-icon><Clock /></el-icon></span>
              <span class="sub-section-label">到期时间</span>
              <span class="sub-section-value">{{ formatDate(subscription.expire_time) || '未设置' }}</span>
            </div>
            <div class="sub-btn-row">
              <el-button size="small" plain @click="addTime(subscription, 30)">+1月</el-button>
              <el-button size="small" plain @click="addTime(subscription, 90)">+3月</el-button>
              <el-button size="small" plain @click="addTime(subscription, 180)">+半年</el-button>
              <el-button size="small" plain @click="addTime(subscription, 365)">+1年</el-button>
            </div>
            <div class="sub-date-picker-row">
              <el-date-picker
                v-model="subscription.expire_time"
                type="date"
                placeholder="选择日期"
                format="YYYY-MM-DD"
                value-format="YYYY-MM-DD"
                size="small"
                @change="updateExpireTime(subscription)"
                clearable
                teleported
                popper-class="mobile-subscription-date-picker-popper"
              />
            </div>
          </div>
          <div class="sub-section" :class="{ 'device-limit-overlimit': isDeviceOverlimit(subscription) }">
            <div class="sub-section-row">
              <span class="sub-section-icon"><el-icon><Monitor /></el-icon></span>
              <span class="sub-section-label">设备限制</span>
              <span class="sub-section-value">{{ subscription.online_devices || 0 }} / {{ subscription.device_limit || 0 }}</span>
              <el-input-number
                v-model="subscription.device_limit"
                :min="0"
                :max="999"
                size="small"
                @change="updateDeviceLimit(subscription)"
                class="device-limit-input-inline"
              />
            </div>
            <div class="sub-btn-row first-btn-row">
              <el-button size="small" type="danger" plain @click="clearUserDevices(subscription)">清理在线</el-button>
              <el-button size="small" plain @click="showUserDetails(subscription)">详情</el-button>
            </div>
            <div class="sub-btn-row second-btn-row">
              <el-button size="small" plain @click="addDeviceLimit(subscription, 1)">+1</el-button>
              <el-button size="small" plain @click="addDeviceLimit(subscription, 5)">+5</el-button>
              <el-button size="small" plain @click="addDeviceLimit(subscription, 10)">+10</el-button>
              <el-button size="small" plain @click="addDeviceLimit(subscription, 20)">+20</el-button>
            </div>
          </div>
          <div class="sub-action-grid">
            <div class="sub-action-item" @click="copyToClipboard(subscription.universal_url)">
              <div class="sub-action-icon" style="background: #ecf5ff; color: #409eff;"><el-icon><DocumentCopy /></el-icon></div>
              <span class="sub-action-text">复制通用</span>
            </div>
            <div class="sub-action-item" @click="copyToClipboard(subscription.clash_url)">
              <div class="sub-action-icon" style="background: #fdf6ec; color: #e6a23c;"><el-icon><Link /></el-icon></div>
              <span class="sub-action-text">复制Clash</span>
            </div>
            <div class="sub-action-item" @click="importToShadowrocket(subscription)">
              <div class="sub-action-icon" style="background: #f0f9eb; color: #67c23a;"><el-icon><Download /></el-icon></div>
              <span class="sub-action-text">导入小火箭</span>
            </div>
            <div class="sub-action-item" @click="showQRCode(subscription)">
              <div class="sub-action-icon" style="background: #f4f4f5; color: #909399;"><el-icon><View /></el-icon></div>
              <span class="sub-action-text">二维码</span>
            </div>
          </div>
          <div class="sub-action-grid">
            <div class="sub-action-item" @click="resetSubscription(subscription)">
              <div class="sub-action-icon" style="background: #ecf5ff; color: #409eff;"><el-icon><Refresh /></el-icon></div>
              <span class="sub-action-text">重置订阅</span>
            </div>
            <div class="sub-action-item" @click="toggleSubscriptionStatus(subscription)">
              <div class="sub-action-icon" :style="subscription.is_active ? 'background: #fef0f0; color: #f56c6c;' : 'background: #f0f9eb; color: #67c23a;'">
                <el-icon><Switch /></el-icon>
              </div>
              <span class="sub-action-text">{{ subscription.is_active ? '禁用' : '启用' }}</span>
            </div>
            <div class="sub-action-item" @click="sendSubscriptionEmail(subscription)">
              <div class="sub-action-icon" style="background: #f0f9eb; color: #67c23a;"><el-icon><Message /></el-icon></div>
              <span class="sub-action-text">发邮件</span>
            </div>
            <div class="sub-action-item" @click="deleteUser(subscription)">
              <div class="sub-action-icon" style="background: #fef0f0; color: #f56c6c;"><el-icon><Delete /></el-icon></div>
              <span class="sub-action-text">删除用户</span>
            </div>
          </div>
        </div>
      </div>
      <div class="mobile-card-list" v-if="subscriptions.length === 0 && !loading">
        <div class="empty-state">
          <i class="el-icon-document"></i>
          <p>暂无订阅记录</p>
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
    <UserDetailDialog
      :visible="showUserDetailDialog"
      @update:visible="showUserDetailDialog = $event"
      :user="selectedUser"
      :isMobile="isMobile"
    />
    <el-dialog v-model="showQRDialog" title="订阅二维码" width="400px" center>
      <div class="qr-dialog-content">
        <div class="qr-code-large">
          <img :src="currentQRCode" alt="QR Code" />
        </div>
        <div class="qr-info">
          <p>扫描二维码即可在Shadowrocket中添加订阅</p>
          <p class="qr-tip">支持V2Ray和通用订阅格式，包含到期时间信息</p>
          <el-button type="primary" @click="downloadQRCode">下载二维码</el-button>
        </div>
      </div>
    </el-dialog>
    <el-dialog v-model="showColumnSettings" title="列设置" width="600px">
      <div class="column-settings">
        <div class="settings-header">
          <p>选择要显示的列，取消勾选将隐藏对应列：</p>
          <div class="quick-actions">
            <el-button size="small" @click="selectAllColumns">全选</el-button>
            <el-button size="small" @click="clearAllColumns">全不选</el-button>
            <el-button size="small" @click="resetToDefault">恢复默认</el-button>
          </div>
        </div>
        <el-checkbox-group v-model="visibleColumns" class="column-checkboxes">
          <div class="checkbox-row">
            <el-checkbox label="qq">用户</el-checkbox>
            <el-checkbox label="expire_time">结束时间</el-checkbox>
            <el-checkbox label="qr_code">二维码</el-checkbox>
          </div>
          <div class="checkbox-row">
            <el-checkbox label="sub_urls">订阅链接</el-checkbox>
            <el-checkbox label="created_at">添加时间</el-checkbox>
            <el-checkbox label="sub_count">订阅次数</el-checkbox>
          </div>
          <div class="checkbox-row">
            <el-checkbox label="online_devices">在线</el-checkbox>
            <el-checkbox label="device_limit">最大设备数</el-checkbox>
            <el-checkbox label="notes">备注</el-checkbox>
          </div>
          <div class="checkbox-row">
            <el-checkbox label="actions">操作</el-checkbox>
          </div>
        </el-checkbox-group>
        <div class="settings-footer">
          <p class="tip">💡 提示：至少需要保留一列显示，建议保留"QQ号码"和"操作"列</p>
        </div>
      </div>
    </el-dialog>
  </div>
</template>
<script>
import { ref, reactive, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from '@/utils/elementPlusServices'
import {
  Download, Delete, Setting, Apple, Monitor, ArrowDown, View, Refresh, HomeFilled,
  Search, Filter, Clock, Sort, Operation, Link, DocumentCopy, User, Message, Switch,
  Check, Close, Loading, CircleCheckFilled
} from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
import { secureStorage } from '@/utils/api'
import { safeNavigate, safeOpen } from '@/utils/safeOpen'
import { formatLocation } from '@/utils/date'
import { formatDateTime, formatDate as formatDateUtil, formatTime as formatTimeUtil } from '@/utils/date'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
import UserDetailDialog from './components/UserDetailDialog.vue'
dayjs.extend(timezone)

const LOGIN_HANDOFF_STORAGE_PREFIX = 'cboard_login_handoff_'

export default {
  name: 'AdminSubscriptions',
  components: {
    Download, Delete, Setting, Apple, Monitor, ArrowDown, View, Refresh, HomeFilled,
    Search, Clock, Sort, Operation, Link, DocumentCopy, User, Message, Switch,
    Check, Close, Loading, CircleCheckFilled, UserDetailDialog
  },
  setup() {
    const route = useRoute()
    const loading = ref(false)
    const subscriptions = ref([])
    const selectedSubscriptions = ref([])
    const batchOperating = ref(false)
    const currentPage = ref(1)
    const pageSize = ref(10)
    const total = ref(0)
    const searchQuery = ref('')
    const currentSort = ref('add_time_desc')
    const searchForm = reactive({
      keyword: '',
      status: ''
    })
    const showUserDetailDialog = ref(false)
    const showQRDialog = ref(false)
    const showColumnSettings = ref(false)
    const selectedUser = ref(null)
    const currentQRCode = ref('')
    const COLUMN_SETTINGS_KEY = 'admin_subscriptions_visible_columns'
    const defaultVisibleColumns = [
      'qq', 'expire_time', 'qr_code', 'sub_urls',
      'created_at', 'sub_count', 'online_devices',
      'device_limit', 'notes', 'actions'
    ]
    const loadColumnSettings = () => {
      try {
        const saved = localStorage.getItem(COLUMN_SETTINGS_KEY)
        if (saved) {
          let parsed = JSON.parse(saved)
          if (parsed.includes('apple_count') || parsed.includes('clash_count')) {
            parsed = parsed.filter(col => col !== 'apple_count' && col !== 'clash_count')
            if (!parsed.includes('sub_count')) parsed.push('sub_count')
          }
          if (parsed.includes('universal_url') || parsed.includes('clash_url')) {
            parsed = parsed.filter(col => col !== 'universal_url' && col !== 'clash_url')
            if (!parsed.includes('sub_urls')) parsed.push('sub_urls')
          }
          if (!parsed.includes('notes')) parsed.push('notes')
          const validColumns = parsed.filter(col => defaultVisibleColumns.includes(col))
          if (validColumns.length > 0) {
            return validColumns
          }
        }
      } catch (error) {
      }
      return defaultVisibleColumns
    }
    const saveColumnSettings = (columns) => {
      try {
        localStorage.setItem(COLUMN_SETTINGS_KEY, JSON.stringify(columns))
      } catch (error) {
      }
    }
    const visibleColumns = ref(loadColumnSettings())
    const tableRef = ref(null)
    const userDevices = ref([])
    const loadingDevices = ref(false)
    const deletingDevice = ref(null)
    const currentSortText = computed(() => {
      const sortMap = {
        'add_time_desc': '添加时间 (降序)',
        'add_time_asc': '添加时间 (升序)',
        'expire_time_desc': '到期时间 (降序)',
        'expire_time_asc': '到期时间 (升序)',
        'device_count_desc': '设备数量 (降序)',
        'device_count_asc': '设备数量 (升序)',
        'apple_count_desc': '通用订阅次数 (降序)',
        'apple_count_asc': '通用订阅次数 (升序)',
        'online_devices_desc': '在线设备 (降序)',
        'online_devices_asc': '在线设备 (升序)',
        'device_limit_desc': '最大设备数 (降序)',
        'device_limit_asc': '最大设备数 (升序)'
      }
      return sortMap[currentSort.value] || '添加时间 (降序)'
    })
    const loadSubscriptions = async () => {
      loading.value = true
      try {
        if (searchForm.keyword && !searchQuery.value) {
          searchQuery.value = searchForm.keyword
        }
        const params = {
          page: currentPage.value,
          size: pageSize.value,
          search: searchForm.keyword || searchQuery.value,
          sort: currentSort.value
        }
        if (searchForm.status) {
          params.status = searchForm.status
        }
        const response = await adminAPI.getSubscriptions(params)
        if (response.data?.success !== false) {
          const subscriptionList = response.data?.data?.subscriptions || []
          subscriptions.value = subscriptionList.map(sub => {
            const mapped = {
              ...sub,
              is_active: sub.is_active === true || sub.is_active === 1 || sub.is_active === '1',
              device_limit: Number(sub.device_limit) || 0,
              expire_time: sub.expire_time || '',
              user_notes: sub.user?.notes || '',
              savingNotes: false,
              notesSaved: false
            }
            // 预计算二维码 URL，避免模板每次渲染重复计算
            mapped.qr_code_url = generateQRCode(mapped)
            const uid = sub.user?.id || sub.user_id
            if (uid) subOriginalNotes.set(uid, mapped.user_notes)
            return mapped
          })
          total.value = response.data?.data?.total || 0
          } else {
          ElMessage.error('加载订阅列表失败')
        }
      } catch (error) {
        ElMessage.error('加载订阅列表失败')
      } finally {
        loading.value = false
      }
    }
    const searchSubscriptions = () => {
      searchQuery.value = searchForm.keyword
      currentPage.value = 1
      loadSubscriptions()
    }
    const resetSearch = () => {
      searchForm.keyword = ''
      searchForm.status = ''
      searchQuery.value = ''
      currentPage.value = 1
      loadSubscriptions()
    }
    const handleStatusFilter = (status) => {
      searchForm.status = status
      currentPage.value = 1
      loadSubscriptions()
    }
    const getStatusFilterText = () => {
      const statusMap = {
        '': '状态筛选',
        'active': '活跃',
        'expired': '已过期'
      }
      return statusMap[searchForm.status] || '状态筛选'
    }
    const handleSortCommand = (command) => {
      currentSort.value = command
      loadSubscriptions()
    }
    const clearSort = () => {
      currentSort.value = 'add_time_desc'
      loadSubscriptions()
    }
    const handleActionCommand = (command) => {
      switch (command) {
        case 'export':
          exportSubscriptions()
          break
        case 'clearDevices':
          clearAllDevices()
          break
        case 'columnSettings':
          showColumnSettings.value = true
          break
      }
    }
    const updateExpireTime = async (subscription) => {
      if (!subscription || !subscription.id) return
      try {
        await adminAPI.updateSubscription(subscription.id, {
          expire_time: subscription.expire_time
        })
        ElMessage.success('到期时间更新成功')
      } catch (error) {
        ElMessage.error('更新到期时间失败: ' + (error.response?.data?.message || error.message))
        loadSubscriptions()
      }
    }
    const addTime = async (subscription, days) => {
      if (!subscription || !subscription.id) return
      try {
        const now = dayjs().tz('Asia/Shanghai')
        let baseDate
        if (subscription.expire_time) {
          const currentExpire = dayjs(subscription.expire_time).tz('Asia/Shanghai')
          if (currentExpire.isAfter(now)) {
            baseDate = currentExpire
          } else {
            baseDate = now
          }
        } else {
          baseDate = now
        }
        if (!baseDate.isValid()) {
          baseDate = now
        }
        const newDate = baseDate.add(days, 'day')
        subscription.expire_time = newDate.format('YYYY-MM-DD')
        await updateExpireTime(subscription)
      } catch (error) {
        ElMessage.error('添加时间失败: ' + (error.response?.data?.message || error.message))
        loadSubscriptions()
      }
    }
    const updateDeviceLimit = async (subscription) => {
      if (!subscription || !subscription.id) return
      try {
        await adminAPI.updateSubscription(subscription.id, {
          device_limit: subscription.device_limit
        })
        ElMessage.success('设备限制更新成功')
        window.dispatchEvent(new CustomEvent('subscription-device-limit-updated', {
          detail: { subscriptionId: subscription.id, deviceLimit: subscription.device_limit }
        }))
      } catch (error) {
        ElMessage.error('更新设备限制失败: ' + (error.response?.data?.message || error.message))
        loadSubscriptions()
      }
    }
    const addDeviceLimit = async (subscription, count) => {
      try {
        const newLimit = Math.max(0, (subscription.device_limit || 0) + count)
        subscription.device_limit = newLimit
        await updateDeviceLimit(subscription)
      } catch (error) {
        ElMessage.error('修改设备限制失败')
        }
    }
    const isExpired = (subscription) => {
      if (!subscription || !subscription.expire_time) return false
      const expireDate = dayjs(subscription.expire_time).tz('Asia/Shanghai')
      if (!expireDate.isValid()) return false
      return expireDate.isBefore(dayjs().tz('Asia/Shanghai'), 'day')
    }
    const isDeviceOverlimit = (subscription) => {
      const onlineDevices = subscription.online_devices || 0
      const deviceLimit = subscription.device_limit || 0
      return deviceLimit > 0 && onlineDevices >= deviceLimit
    }
    const generateQRCode = (subscription) => {
      if (!subscription) return ''
      if (subscription.qrcodeUrl) {
        const qrData = subscription.qrcodeUrl
        const isMobile = window.innerWidth <= 768
        const qrSize = isMobile ? '400x400' : '200x200'
        return `https://api.qrserver.com/v1/create-qr-code/?size=${qrSize}&data=${encodeURIComponent(qrData)}&ecc=M&margin=10`
      }
      let qrData = ''
      if (subscription.universal_url) {
        const universalUrl = subscription.universal_url
        const encodedUrl = btoa(universalUrl)
        let expiryDisplayName = ''
        if (subscription.expire_time) {
          const expireDate = dayjs(subscription.expire_time).tz('Asia/Shanghai')
          expiryDisplayName = `到期时间${expireDate.format('YYYY-MM-DD HH:mm:ss')}`
        } else {
          expiryDisplayName = subscription.subscription_url || '订阅'
        }
        qrData = `sub://${encodedUrl}#${encodeURIComponent(expiryDisplayName)}`
      } else if (subscription.subscription_url) {
        const baseUrl = window.location.origin
        const subscriptionUrl = `${baseUrl}/api/v1/subscriptions/ssr/${subscription.subscription_url}`
        const encodedUrl = btoa(subscriptionUrl)
        let expiryDisplayName = ''
        if (subscription.expire_time) {
          const expireDate = new Date(subscription.expire_time)
          const year = expireDate.getFullYear()
          const month = String(expireDate.getMonth() + 1).padStart(2, '0')
          const day = String(expireDate.getDate()).padStart(2, '0')
          expiryDisplayName = `到期时间${year}-${month}-${day}`
        } else {
          expiryDisplayName = subscription.subscription_url
        }
        qrData = `sub://${encodedUrl}#${encodeURIComponent(expiryDisplayName)}`
      } else {
        return ''
      }
      const isMobile = window.innerWidth <= 768
      const qrSize = isMobile ? '400x400' : '200x200'
      return `https://api.qrserver.com/v1/create-qr-code/?size=${qrSize}&data=${encodeURIComponent(qrData)}&ecc=M&margin=10`
    }
    const showQRCode = (subscription) => {
      if (subscription.subscription_url || subscription.universal_url) {
        currentQRCode.value = generateQRCode(subscription)
        showQRDialog.value = true
      }
    }
    const downloadQRCode = () => {
      const link = document.createElement('a')
      link.href = currentQRCode.value
      link.download = 'subscription-qr.png'
      link.click()
    }
    const formatExpireDate = (expireTime) => {
      if (!expireTime) return '未设置'
      const date = dayjs(expireTime).tz('Asia/Shanghai')
      if (!date.isValid()) return '未设置'
      const year = date.year()
      const month = String(date.month() + 1).padStart(2, '0')
      const day = String(date.date()).padStart(2, '0')
      return `${year}-${month}-${day}`
    }
    const importToShadowrocket = (subscription) => {
      if (!subscription.subscription_url && !subscription.universal_url) {
        ElMessage.warning('该订阅没有可用的订阅地址')
        return
      }
      let subscriptionUrl = ''
      if (subscription.qrcodeUrl) {
        const match = subscription.qrcodeUrl.match(/sub:\/\/([^#]+)/)
        if (match) {
          subscriptionUrl = atob(match[1])
        } else {
          subscriptionUrl = subscription.universal_url || subscription.subscription_url
        }
      } else if (subscription.universal_url) {
        subscriptionUrl = subscription.universal_url
      } else if (subscription.subscription_url) {
        const baseUrl = window.location.origin
        subscriptionUrl = `${baseUrl}/api/v1/subscriptions/ssr/${subscription.subscription_url}`
      }
      if (!subscriptionUrl) {
        ElMessage.error('无法生成订阅链接')
        return
      }
      try {
        const encodedUrl = btoa(subscriptionUrl)
        let expiryDisplayName = ''
        if (subscription.expire_time) {
          expiryDisplayName = formatExpireDate(subscription.expire_time)
        } else {
          expiryDisplayName = '订阅'
        }
        const subLink = `sub://${encodedUrl}#${encodeURIComponent(expiryDisplayName)}`
        safeNavigate(subLink, { allowAppProtocols: true })
        setTimeout(() => {
          copyToClipboard(subscriptionUrl)
          ElMessage.success('已复制订阅链接到剪贴板，请在 Shadowrocket 中手动添加')
        }, 500)
      } catch (error) {
        copyToClipboard(subscriptionUrl)
        ElMessage.success('已复制订阅链接到剪贴板，请在 Shadowrocket 中手动添加')
      }
    }
    const showUserDetails = async (subscription) => {
      try {
        const userResponse = await adminAPI.getUserDetails(subscription.user.id)
        if (userResponse.data && userResponse.data.success) {
          selectedUser.value = userResponse.data.data
          showUserDetailDialog.value = true
        } else {
          throw new Error(userResponse.data?.message || '获取用户详情失败')
        }
      } catch (error) {
        ElMessage.error('加载用户详情失败: ' + (error.response?.data?.message || error.message))
      }
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
          userDevices.value = devices.map(device => {
            return {
              id: device.id,
              device_name: device.device_name || device.name || '未知设备',
              device_type: device.device_type || device.type || 'unknown',
              ip_address: device.ip_address || device.ip || '-',
              location: device.location || '', // 添加归属地字段
              os_name: device.os_name || '-',
              os_version: device.os_version || '',
              last_seen: device.last_seen || device.last_access || null,
              last_access: device.last_access || device.last_seen || null,
              access_count: device.access_count || 0,
              is_active: device.is_active !== false,
              is_allowed: device.is_allowed !== false,
              user_agent: device.user_agent || '',
              software_name: device.software_name || '',
              software_version: device.software_version || '',
              device_model: device.device_model || '',
              device_brand: device.device_brand || '',
              created_at: device.created_at || null
            }
          })
          if (selectedUser.value) {
            selectedUser.value.online_devices = userDevices.value.length
            selectedUser.value.current_devices = userDevices.value.length
          }
        } else {
          userDevices.value = []
          ElMessage.warning('获取设备列表失败: 响应格式不正确')
          if (selectedUser.value) {
            selectedUser.value.online_devices = 0
            selectedUser.value.current_devices = 0
          }
        }
      } catch (error) {
        if (process.env.NODE_ENV === 'development') {
          console.error('加载设备列表失败:', error)
        }
        userDevices.value = []
        ElMessage.error('加载设备列表失败: ' + (error.response?.data?.message || error.message || '未知错误'))
        if (selectedUser.value) {
          selectedUser.value.online_devices = 0
          selectedUser.value.current_devices = 0
        }
      } finally {
        loadingDevices.value = false
      }
    }
    const deleteDevice = async (device) => {
      try {
        await ElMessageBox.confirm(
          `确定要删除设备 "${device.device_name || '未知设备'}" 吗？删除后该设备将无法继续使用订阅，此操作不可恢复。`,
          '确认删除',
          {
            confirmButtonText: '确定删除',
            cancelButtonText: '取消',
            type: 'warning',
          }
        )
        deletingDevice.value = device.id
        const response = await adminAPI.removeDevice(device.id)
        if (response.data && response.data.success) {
          ElMessage.success('设备删除成功')
          await loadUserDevices()
          await loadSubscriptions()
          if (selectedUser.value) {
            selectedUser.value.online_devices = (selectedUser.value.online_devices || 1) - 1
            selectedUser.value.current_devices = (selectedUser.value.current_devices || 1) - 1
          }
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
    const getResetTypeTag = (type) => {
      const typeMap = {
        'manual': 'primary',
        'automatic': 'info',
        'admin': 'warning',
        'system': 'success'
      }
      return typeMap[type] || 'info'
    }
    const getResetTypeText = (type) => {
      const typeMap = {
        'manual': '手动重置',
        'automatic': '自动重置',
        'admin': '管理员重置',
        'system': '系统重置'
      }
      return typeMap[type] || type || '未知'
    }
    const getResetByTag = (by) => {
      const byMap = {
        'user': 'primary',
        'admin': 'warning',
        'system': 'success'
      }
      return byMap[by] || 'info'
    }
    const getResetByText = (by) => {
      const byMap = {
        'user': '用户',
        'admin': '管理员',
        'system': '系统'
      }
      return byMap[by] || by || '未知'
    }
    const getDeviceTypeTag = (type) => {
      const typeMap = {
        'mobile': 'primary',
        'desktop': 'success',
        'tablet': 'warning',
        'server': 'danger'
      }
      return typeMap[type] || 'info'
    }
    const getDeviceTypeText = (type) => {
      const typeMap = {
        'mobile': '手机',
        'desktop': '电脑',
        'tablet': '平板',
        'server': '服务器'
      }
      return typeMap[type] || type || '未知'
    }
    const copyToClipboard = async (text) => {
      if (!text) {
        ElMessage.warning('没有可复制的内容')
        return
      }
      try {
        await navigator.clipboard.writeText(text)
        ElMessage.success('订阅链接已复制到剪贴板')
      } catch (error) {
        try {
          const textArea = document.createElement('textarea')
          textArea.value = text
          document.body.appendChild(textArea)
          textArea.select()
          document.execCommand('copy')
          document.body.removeChild(textArea)
          ElMessage.success('订阅链接已复制到剪贴板')
        } catch (fallbackError) {
          ElMessage.error('复制失败，请手动复制')
        }
      }
    }
    const goToUserBackend = async (subscription) => {
      try {
        const userId = subscription.user?.id || subscription.user_id
        if (!userId || userId === 0) {
          ElMessage.warning('无法进入：用户信息不存在或已被删除')
          return
        }
        const userName = subscription.user?.username || subscription.user?.email || subscription.username || subscription.email || '未知用户'
        await ElMessageBox.confirm(
          `确定要以用户 ${userName} 的身份登录吗？将跳转到用户后台。`,
          '确认登录',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'info'
          }
        )
        const response = await adminAPI.loginAsUser(userId)
        if (!response.data) {
          ElMessage.error('登录失败：服务器未返回数据')
          return
        }
        if (response.data.success === false) {
          ElMessage.error(response.data.message || '登录失败')
          return
        }
        if (!response.data.data) {
          ElMessage.error('登录失败：服务器返回数据不完整')
          return
        }
        if (!response.data.data.access_token || !response.data.data.user) {
          ElMessage.error('登录失败：服务器返回数据不完整')
          return
        }
        const adminToken = secureStorage.get('admin_token')
        const adminUser = secureStorage.get('admin_user')
        const adminRefreshToken = secureStorage.get('admin_refresh_token')
        const userToken = response.data.data.access_token
        const userRefreshToken = response.data.data.refresh_token
        const userData = response.data.data.user
        const sessionKey = `user_login_${Date.now()}`
        const sessionData = {
          token: userToken,
          refreshToken: userRefreshToken,
          user: userData,
          timestamp: Date.now()
        }
        if (adminToken && adminUser) {
          sessionData.adminToken = adminToken
          sessionData.adminUser = typeof adminUser === 'string' ? adminUser : JSON.stringify(adminUser)
          sessionData.adminRefreshToken = adminRefreshToken
        }
        const handoffPayload = JSON.stringify(sessionData)
        sessionStorage.setItem(sessionKey, handoffPayload)
        localStorage.setItem(`${LOGIN_HANDOFF_STORAGE_PREFIX}${sessionKey}`, handoffPayload)
        window.setTimeout(() => {
          localStorage.removeItem(`${LOGIN_HANDOFF_STORAGE_PREFIX}${sessionKey}`)
        }, 5 * 60 * 1000)
        const dashboardUrl = window.location.origin + '/dashboard'
        const finalUrl = `${dashboardUrl}?sessionKey=${sessionKey}`
        ElMessage.success('正在跳转到用户后台...')
        const opened = safeOpen(finalUrl)
        if (!opened) {
          safeNavigate(finalUrl)
        }
      } catch (error) {
        if (error !== 'cancel') {
          if (process.env.NODE_ENV === 'development') {
            console.error('登录失败:', error)
          }
          const errorMessage = error.response?.data?.message ||
                             error.response?.data?.detail ||
                             error.message ||
                             '登录失败'
          ElMessage.error(errorMessage)
        }
      }
    }
    const resetSubscription = async (subscription) => {
      try {
        const userId = subscription.user?.id || subscription.user_id
        if (!userId || userId === 0) {
          ElMessage.warning('无法重置：用户信息不存在或已被删除')
          return
        }
        await ElMessageBox.confirm('确定要重置该用户的订阅地址吗？重置后所有设备将无法继续使用，需要重新订阅。', '确认重置', {
          type: 'warning'
        })
        await adminAPI.resetUserSubscription(userId)
        ElMessage.success('订阅地址重置成功')
        loadSubscriptions()
        if (selectedUser.value && (selectedUser.value.user?.id === userId || selectedUser.value.user_id === userId)) {
          await loadUserDevices()
        }
      } catch (error) {
        if (error !== 'cancel') {
          const errorMsg = error.response?.data?.message || error.message || '重置订阅失败'
          ElMessage.error(`重置订阅失败: ${errorMsg}`)
        }
      }
    }
    const sendingEmailMap = new Map()
    const sendSubscriptionEmail = async (subscription) => {
      const userId = subscription.user?.id || subscription.user_id
      if (!userId || userId === 0) {
        ElMessage.warning('无法发送：用户信息不存在或已被删除')
        return
      }
      if (sendingEmailMap.has(userId)) {
        ElMessage.warning('邮件正在发送中，请勿重复点击')
        return
      }
      sendingEmailMap.set(userId, true)
      try {
        const response = await adminAPI.sendSubscriptionEmail(userId)
        if (response && response.data) {
          if (response.data.success === false) {
            ElMessage.error(response.data.message || '发送订阅邮件失败')
          } else {
            ElMessage.success(response.data.message || '订阅邮件发送成功')
          }
        } else {
          ElMessage.success('订阅邮件已加入发送队列')
        }
      } catch (error) {
        const errorMessage = error.response?.data?.message || error.response?.data?.detail || error.message || '发送订阅邮件失败'
        ElMessage.error(errorMessage)
        console.error('发送订阅邮件失败:', error)
      } finally {
        setTimeout(() => {
          sendingEmailMap.delete(userId)
        }, 3000)
      }
    }
    const toggleSubscriptionStatus = async (subscription) => {
      try {
        const newStatus = !subscription.is_active
        await adminAPI.updateSubscription(subscription.id, {
          is_active: newStatus
        })
        subscription.is_active = newStatus
        ElMessage.success(`订阅已${newStatus ? '启用' : '禁用'}`)
      } catch (error) {
        ElMessage.error('更新订阅状态失败')
        }
    }
    const deleteUser = async (subscription) => {
      const userId = subscription.user?.id || subscription.user_id
      if (!userId || userId === 0) {
        ElMessage.warning('无法删除：用户信息不存在或已被删除')
        return
      }
      if (subscription.user?.deleted) {
        ElMessage.warning('该用户已被删除，无法再次删除')
        return
      }
      try {
        await ElMessageBox.confirm(
          '确定要删除该用户吗？这将删除用户的所有信息，包括设备记录、账号信息、邮件信息、UA记录等。此操作不可恢复！',
          '确认删除',
          {
            type: 'error',
            confirmButtonText: '确定删除',
            cancelButtonText: '取消'
          }
        )
        await adminAPI.deleteUser(userId)
        ElMessage.success('用户删除成功')
        loadSubscriptions()
      } catch (error) {
        if (error !== 'cancel') {
          const errorMsg = error.response?.data?.message || error.message || '删除用户失败'
          ElMessage.error(`删除用户失败: ${errorMsg}`)
        }
      }
    }
    const clearUserDevices = async (subscription) => {
      try {
        const userId = subscription.user?.id || subscription.user_id
        if (!userId || userId === 0) {
          ElMessage.warning('无法清理：用户信息不存在或已被删除')
          return
        }
        await ElMessageBox.confirm('确定要清理该用户的在线设备吗？这将清除所有设备记录和UA记录。', '确认清理', {
          type: 'warning'
        })
        await adminAPI.clearUserDevices(userId)
        ElMessage.success('设备清理成功')
        await loadSubscriptions()
        if (selectedUser.value && (selectedUser.value.user?.id === userId || selectedUser.value.user_id === userId)) {
          await loadUserDevices()
        }
      } catch (error) {
        if (error !== 'cancel') {
          const errorMsg = error.response?.data?.message || error.message || '清理设备失败'
          ElMessage.error(`清理设备失败: ${errorMsg}`)
        }
      }
    }
    const clearAllDevices = async () => {
      try {
        await ElMessageBox.confirm('确定要清理所有用户的设备吗？这将清除所有设备记录。', '确认清理', {
          type: 'warning'
        })
        const subscriptionIds = subscriptions.value.map(sub => sub.id)
        if (subscriptionIds.length === 0) {
          ElMessage.warning('没有可清理的订阅')
          return
        }
        await adminAPI.batchClearDevices({ subscription_ids: subscriptionIds })
        ElMessage.success('批量清理设备成功')
        loadSubscriptions()
      } catch (error) {
        if (error !== 'cancel') {
          const errorMsg = error.response?.data?.message || error.message || '批量清理设备失败'
          ElMessage.error(`批量清理设备失败: ${errorMsg}`)
        }
      }
    }
    const subNotesSaveTimers = new Map()
    const subNotesSavedTimers = new Map()
    const subOriginalNotes = new Map()
    const saveSubNotes = async (row) => {
      const userId = row.user?.id || row.user_id
      if (!userId || userId === 0) return
      const currentNotes = row.user_notes || ''
      const original = subOriginalNotes.get(userId) || ''
      if (currentNotes === original) { row.savingNotes = false; return }
      if (subNotesSaveTimers.has(userId)) { clearTimeout(subNotesSaveTimers.get(userId)); subNotesSaveTimers.delete(userId) }
      row.savingNotes = true
      row.notesSaved = false
      try {
        await adminAPI.updateUser(userId, { notes: currentNotes || null })
        subOriginalNotes.set(userId, currentNotes)
        row.notesSaved = true
        if (subNotesSavedTimers.has(userId)) clearTimeout(subNotesSavedTimers.get(userId))
        subNotesSavedTimers.set(userId, setTimeout(() => { row.notesSaved = false; subNotesSavedTimers.delete(userId) }, 2000))
      } catch (error) {
        ElMessage.error(`保存备注失败: ${error.response?.data?.message || error.message}`)
        row.user_notes = original
      } finally { row.savingNotes = false }
    }
    const debounceSaveSubNotes = (row) => {
      const userId = row.user?.id || row.user_id
      if (!userId) return
      if (!subOriginalNotes.has(userId)) subOriginalNotes.set(userId, row.user_notes || '')
      if (subNotesSaveTimers.has(userId)) clearTimeout(subNotesSaveTimers.get(userId))
      subNotesSaveTimers.set(userId, setTimeout(() => { saveSubNotes(row); subNotesSaveTimers.delete(userId) }, 1000))
    }
    const exportSubscriptions = async () => {
      try {
        const response = await adminAPI.exportSubscriptions()
        let blob = null
        if (response.data instanceof Blob) {
          blob = response.data
        } else if (response.data && typeof response.data === 'object' && response.data.data) {
          blob = response.data.data
        }
        if (blob instanceof Blob) {
          const contentDisposition = response.headers['content-disposition'] || response.headers['Content-Disposition']
          const beijingDate = dayjs().tz('Asia/Shanghai')
          let filename = `subscriptions_export_${beijingDate.format('YYYYMMDD')}.csv`
          if (contentDisposition) {
            let filenameMatch = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i)
            if (filenameMatch && filenameMatch.length > 1) {
              filename = decodeURIComponent(filenameMatch[1])
            } else {
              filenameMatch = contentDisposition.match(/filename=['"]?([^'";]+)['"]?/i)
              if (filenameMatch && filenameMatch.length > 1) {
                filename = decodeURIComponent(filenameMatch[1])
              }
            }
          }
          const url = window.URL.createObjectURL(blob)
          const link = document.createElement('a')
          link.style.display = 'none'
          link.href = url
          link.download = filename
          document.body.appendChild(link)
          requestAnimationFrame(() => {
            link.click()
            setTimeout(() => {
              document.body.removeChild(link)
              window.URL.revokeObjectURL(url)
              }, 1000)
          })
          ElMessage.success('订阅数据导出成功，文件下载已开始')
        } else {
          ElMessage.error('导出失败：响应格式不正确，收到的是：' + (typeof response.data))
        }
      } catch (error) {
        if (error.response) {
          if (error.response.data instanceof Blob) {
            error.response.data.text().then(text => {
              try {
                const errorData = JSON.parse(text)
                ElMessage.error(errorData.message || errorData.detail || '导出失败')
              } catch (e) {
                ElMessage.error('导出失败：服务器返回错误')
              }
            }).catch(() => {
              ElMessage.error('导出失败：无法读取错误信息')
            })
          } else if (error.response.data?.message || error.response.data?.detail) {
            ElMessage.error(error.response.data.message || error.response.data.detail || '导出失败')
          } else {
            ElMessage.error(`导出失败：${error.response.status} ${error.response.statusText}`)
          }
        } else if (error.message) {
          ElMessage.error(`导出失败：${error.message}`)
        } else {
          ElMessage.error('导出失败：未知错误')
        }
      }
    }
    const showAppleStats = () => {
      ElMessage.info('通用订阅统计功能待实现')
    }
    const showOnlineStats = () => {
      ElMessage.info('在线设备统计功能待实现')
    }
    const truncateText = (text, maxLength) => {
      if (!text) return ''
      return text.length > maxLength ? text.substring(0, maxLength) + '...' : text
    }
    const truncateUserAgent = (userAgent) => {
      if (!userAgent) return '未知'
      return userAgent.length > 50 ? userAgent.substring(0, 50) + '...' : userAgent
    }
    const formatTime = (time) => {
      return formatTimeUtil(time) || '未知'
    }
    const truncateUrl = (url) => {
      if (!url) return ''
      const isMobile = window.innerWidth <= 768
      if (isMobile) {
        return url
      }
      if (url.length > 60) {
        return url.substring(0, 40) + '...' + url.substring(url.length - 15)
      }
      return url
    }
    const getSubscriptionStatusType = (status) => {
      const statusMap = {
        'active': 'success',
        'inactive': 'info',
        'expired': 'danger',
        'paused': 'warning'
      }
      return statusMap[status] || 'info'
    }
    const getSubscriptionStatusText = (status) => {
      const statusMap = {
        'active': '活跃',
        'inactive': '未激活',
        'expired': '已过期',
        'paused': '已暂停'
      }
      return statusMap[status] || '未知'
    }
    const formatDate = (date) => {
      return formatDateUtil(date)
    }
    const handleSelectionChange = (selection) => {
      selectedSubscriptions.value = selection
    }
    const clearSelection = () => {
      selectedSubscriptions.value = []
    }
    const batchDeleteSubscriptions = async () => {
      if (selectedSubscriptions.value.length === 0) {
        ElMessage.warning('请先选择要删除的订阅')
        return
      }
      try {
        await ElMessageBox.confirm(
          `确定要删除选中的 ${selectedSubscriptions.value.length} 个订阅吗？此操作不可恢复。`,
          '确认批量删除',
          {
            type: 'warning',
            confirmButtonText: '确定删除',
            cancelButtonText: '取消'
          }
        )
        batchOperating.value = true
        const subscriptionIds = selectedSubscriptions.value.map(sub => sub.id)
        const response = await adminAPI.batchDeleteSubscriptions(subscriptionIds)
        if (response.data?.success !== false) {
          ElMessage.success(response.data?.message || `成功删除 ${selectedSubscriptions.value.length} 个订阅`)
          clearSelection()
          loadSubscriptions()
        } else {
          ElMessage.error(response.data?.message || '批量删除失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(`批量删除失败: ${error.response?.data?.message || error.message}`)
        }
      } finally {
        batchOperating.value = false
      }
    }
    const batchEnableSubscriptions = async () => {
      if (selectedSubscriptions.value.length === 0) {
        ElMessage.warning('请先选择要启用的订阅')
        return
      }
      try {
        batchOperating.value = true
        const subscriptionIds = selectedSubscriptions.value.map(sub => sub.id)
        const response = await adminAPI.batchEnableSubscriptions(subscriptionIds)
        if (response.data?.success !== false) {
          ElMessage.success(response.data?.message || `成功启用 ${selectedSubscriptions.value.length} 个订阅`)
          clearSelection()
          loadSubscriptions()
        } else {
          ElMessage.error(response.data?.message || '批量启用失败')
        }
      } catch (error) {
        ElMessage.error(`批量启用失败: ${error.response?.data?.message || error.message}`)
      } finally {
        batchOperating.value = false
      }
    }
    const batchDisableSubscriptions = async () => {
      if (selectedSubscriptions.value.length === 0) {
        ElMessage.warning('请先选择要禁用的订阅')
        return
      }
      try {
        await ElMessageBox.confirm(
          `确定要禁用选中的 ${selectedSubscriptions.value.length} 个订阅吗？`,
          '确认批量禁用',
          {
            type: 'warning',
            confirmButtonText: '确定禁用',
            cancelButtonText: '取消'
          }
        )
        batchOperating.value = true
        const subscriptionIds = selectedSubscriptions.value.map(sub => sub.id)
        const response = await adminAPI.batchDisableSubscriptions(subscriptionIds)
        if (response.data?.success !== false) {
          ElMessage.success(response.data?.message || `成功禁用 ${selectedSubscriptions.value.length} 个订阅`)
          clearSelection()
          loadSubscriptions()
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
    const batchResetSubscriptions = async () => {
      if (selectedSubscriptions.value.length === 0) {
        ElMessage.warning('请先选择要重置的订阅')
        return
      }
      try {
        await ElMessageBox.confirm(
          `确定要重置选中的 ${selectedSubscriptions.value.length} 个订阅吗？这将生成新的订阅地址并清理所有设备。`,
          '确认批量重置',
          {
            type: 'warning',
            confirmButtonText: '确定重置',
            cancelButtonText: '取消'
          }
        )
        batchOperating.value = true
        const subscriptionIds = selectedSubscriptions.value.map(sub => sub.id)
        const response = await adminAPI.batchResetSubscriptions(subscriptionIds)
        if (response.data?.success !== false) {
          const data = response.data?.data || {}
          const successCount = data.success_count || selectedSubscriptions.value.length
          const failCount = data.fail_count || 0
          ElMessage.success(response.data?.message || `成功重置 ${successCount} 个订阅${failCount > 0 ? `，失败 ${failCount} 个` : ''}`)
          clearSelection()
          loadSubscriptions()
        } else {
          ElMessage.error(response.data?.message || '批量重置失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(`批量重置失败: ${error.response?.data?.message || error.message}`)
        }
      } finally {
        batchOperating.value = false
      }
    }
    const batchSendAdminSubEmail = async () => {
      if (selectedSubscriptions.value.length === 0) {
        ElMessage.warning('请先选择要发送邮件的订阅')
        return
      }
      try {
        batchOperating.value = true
        const subscriptionIds = selectedSubscriptions.value.map(sub => sub.id)
        const response = await adminAPI.batchSendAdminSubEmail(subscriptionIds)
        if (response.data?.success !== false) {
          const data = response.data?.data || {}
          const successCount = data.success_count || selectedSubscriptions.value.length
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
    const handleSizeChange = (val) => {
      pageSize.value = val
      loadSubscriptions()
    }
    const handleCurrentChange = (val) => {
      currentPage.value = val
      loadSubscriptions()
    }
    const sortByApple = () => {
      currentSort.value = 'apple_count_desc'
      loadSubscriptions()
    }
    const sortByOnline = () => {
      currentSort.value = 'online_devices_desc'
      loadSubscriptions()
    }
    const sortByCreatedTime = () => {
      currentSort.value = 'add_time_desc'
      loadSubscriptions()
    }
    const handleSortChange = ({ column, prop, order }) => {
      if (!order) {
        currentSort.value = 'add_time_desc'
      } else {
        const direction = order === 'descending' ? 'desc' : 'asc'
        let sortField = prop
        if (prop === 'created_at') sortField = 'add_time'
        currentSort.value = `${sortField}_${direction}`
      }
      currentPage.value = 1
      loadSubscriptions()
    }
    const selectAllColumns = () => {
      visibleColumns.value = [...defaultVisibleColumns]
    }
    const clearAllColumns = () => {
      visibleColumns.value = ['qq', 'actions']
    }
    const resetToDefault = () => {
      visibleColumns.value = [...defaultVisibleColumns]
    }
    watch(visibleColumns, (newColumns) => {
      if (newColumns.length === 0) {
        visibleColumns.value = ['qq', 'actions']
        return
      }
      saveColumnSettings(newColumns)
    }, { deep: true })
    const isMobile = ref(typeof window !== 'undefined' && window.innerWidth <= 768)
    function onResize() { isMobile.value = window.innerWidth <= 768 }
    onMounted(() => window.addEventListener('resize', onResize))
    onUnmounted(() => window.removeEventListener('resize', onResize))
    onMounted(() => {
      if (route.query.search) {
        const searchParam = String(route.query.search).trim()
        if (searchParam) {
          searchForm.keyword = searchParam
          searchQuery.value = searchParam
          currentPage.value = 1
        }
      }
      loadSubscriptions()
    })
    return {
      isMobile,
      loading,
      subscriptions,
      selectedSubscriptions,
      batchOperating,
      currentPage,
      pageSize,
      total,
      searchQuery,
      searchForm,
      currentSort,
      currentSortText,
      showUserDetailDialog,
      showQRDialog,
      showColumnSettings,
      selectedUser,
      currentQRCode,
      visibleColumns,
      userDevices,
      loadingDevices,
      deletingDevice,
      loadSubscriptions,
      searchSubscriptions,
      resetSearch,
      handleStatusFilter,
      getStatusFilterText,
      handleSortCommand,
      clearSort,
      updateExpireTime,
      addTime,
      updateDeviceLimit,
      addDeviceLimit,
      isExpired,
      isDeviceOverlimit,
      generateQRCode,
      showQRCode,
      downloadQRCode,
      formatExpireDate,
      importToShadowrocket,
      showUserDetails,
      loadUserDevices,
      deleteDevice,
      getDeviceTypeTag,
      getDeviceTypeText,
      getResetTypeTag,
      getResetTypeText,
      getResetByTag,
      getResetByText,
      copyToClipboard,
      goToUserBackend,
      resetSubscription,
      sendSubscriptionEmail,
      toggleSubscriptionStatus,
      deleteUser,
      clearUserDevices,
      clearAllDevices,
      exportSubscriptions,
      showAppleStats,
      showOnlineStats,
      getSubscriptionStatusType,
      getSubscriptionStatusText,
      formatDate,
      handleSelectionChange,
      clearSelection,
      batchDeleteSubscriptions,
      batchEnableSubscriptions,
      batchDisableSubscriptions,
      batchResetSubscriptions,
      batchSendAdminSubEmail,
      handleSizeChange,
      handleCurrentChange,
      selectAllColumns,
      clearAllColumns,
      resetToDefault,
      sortByApple,
      sortByOnline,
      sortByCreatedTime,
      handleSortChange,
      truncateUserAgent,
      formatTime,
      formatLocation,
      handleActionCommand,
      truncateUrl,
      saveSubNotes,
      debounceSaveSubNotes
    }
  }
}
</script>
<style scoped lang="scss">
.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
:deep(.el-card) {
  width: 100%;
  max-width: 100%;
}
:deep(.list-card) {
  width: 100%;
  max-width: 100%;
}
.qq-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.qq-email {
  font-size: 12px;
  font-weight: 500;
  color: #303133;
  word-break: break-all;
  line-height: 1.3;
}
.qq-username {
  font-size: 11px;
  color: #909399;
  line-height: 1.3;
}
.detail-btn {
  width: 100%;
}
.expire-time-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px;
  border-radius: 6px;
  transition: all 0.3s;
  &.expire-time-expired {
    background: #fef0f0;
    border: 1px solid #f56c6c;
    animation: pulse-alert 2s ease-in-out infinite;
  }
}
.expire-picker {
  width: 100%;
}
.quick-buttons {
  display: flex;
  gap: 4px;
  justify-content: center;
}
.quick-buttons .el-button {
  padding: 2px 6px;
  font-size: 11px;
  min-width: 0;
}
.qr-code-section {
  display: flex;
  justify-content: center;
  align-items: center;
}
.qr-code {
  cursor: pointer;
  transition: transform 0.2s;
}
.qr-code:hover {
  transform: scale(1.1);
}
.qr-code img {
  width: 50px;
  height: 50px;
  border-radius: 4px;
}
.subscription-link {
  word-break: break-all;
}
.link-text {
  font-size: 12px;
}
.copy-link {
  cursor: pointer;
  transition: all 0.3s ease;
}
.copy-link:hover {
  color: #409eff !important;
  text-decoration: underline !important;
  transform: scale(1.02);
}
.copy-link:active {
  transform: scale(0.98);
}
.device-limit-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px;
  border-radius: 6px;
  transition: all 0.3s;
  &.device-limit-overlimit {
    background: #fef0f0;
    border: 1px solid #f56c6c;
    animation: pulse-alert 2s ease-in-out infinite;
  }
}
.device-limit-input {
  width: 100%;
}
.quick-device-buttons {
  display: flex;
  gap: 4px;
  justify-content: center;
}
.quick-device-buttons .el-button {
  padding: 2px 6px;
  font-size: 11px;
  min-width: 0;
}
.quick-device-buttons-mobile {
  display: flex;
  gap: 6px;
  margin-top: 8px;
  flex-wrap: wrap;
}
.quick-device-buttons-mobile .el-button {
  flex: 1;
  min-width: 0;
  padding: 6px 8px;
  font-size: 0.75rem;
}
@keyframes pulse-alert {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(245, 108, 108, 0.4);
  }
  50% {
    box-shadow: 0 0 0 4px rgba(245, 108, 108, 0);
  }
}
.action-buttons {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.button-row {
  display: flex;
  gap: 4px;
  justify-content: center;
}
.button-row .el-button {
  padding: 3px 6px;
  font-size: 11px;
  flex: 1;
  min-width: 0;
}
.user-detail-drawer {
  :deep(.el-drawer__header) {
    margin-bottom: 0;
    padding: 16px 20px;
    border-bottom: 1px solid #ebeef5;
    font-weight: 600;
    color: #303133;
  }
  :deep(.el-drawer__body) {
    padding: 20px;
    overflow-y: auto;
    @media (max-width: 768px) {
      padding: 12px;
    }
  }
}
.device-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  @media (max-width: 768px) {
    flex-direction: column;
    align-items: flex-start;
    :is(h4) {
      width: 100%;
      margin-bottom: 8px;
    }
  }
}
.device-header h4 {
  margin: 0;
}
.device-stats {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  @media (max-width: 768px) {
    width: 100%;
    justify-content: space-between;
    .el-button {
      flex: 1;
      min-width: 120px;
    }
  }
}
.device-info {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.device-info .el-icon {
  color: #409eff;
  margin-top: 2px;
}
.device-name-details {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.device-main-name {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.device-name-text {
  font-weight: 500;
  color: #303133;
}
.device-model-info {
  display: flex;
  align-items: center;
}
.os-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.os-name {
  font-weight: 500;
  color: #303133;
}
.os-version {
  display: flex;
  align-items: center;
}
.empty-devices {
  text-align: center;
  padding: 20px;
}
.ip-location-cell {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
}
.detail-section {
  margin-bottom: 20px;
  @media (max-width: 768px) {
    margin-bottom: 16px;
    :deep(.el-card__body) {
      padding: 12px;
    }
    :deep(.el-card__header) {
      padding: 12px;
    }
    :deep(.el-descriptions) {
      .el-descriptions__table {
        .el-descriptions__label {
          width: 100px;
          font-size: 13px;
          padding: 8px 6px;
        }
        .el-descriptions__content {
          font-size: 13px;
          padding: 8px 6px;
        }
      }
    }
  }
  :is(h4) {
    margin: 0;
    color: #303133;
    @media (max-width: 768px) {
      font-size: 16px;
    }
  }
}
.subscription-urls {
  display: flex;
  flex-direction: column;
  gap: 16px;
  @media (max-width: 768px) {
    gap: 12px;
  }
}
.url-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  @media (max-width: 768px) {
    gap: 6px;
    :is(label) {
      font-size: 14px;
    }
    :deep(.el-input) {
      .el-input__wrapper {
        padding: 8px 12px;
      }
    }
    :deep(.el-button) {
      padding: 8px 16px;
      font-size: 14px;
    }
  }
}
.url-item :is(label) {
  font-weight: 500;
  color: #606266;
}
.device-table-wrapper {
  width: 100%;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  @media (max-width: 768px) {
    margin: 0 -12px;
    padding: 0 12px;
    .device-table {
      min-width: 800px;
      :deep(.el-table__cell) {
        padding: 8px 4px;
        font-size: 12px;
      }
      :deep(.el-button) {
        padding: 4px 8px;
        font-size: 12px;
      }
      :deep(.el-tag) {
        font-size: 11px;
        padding: 2px 6px;
      }
    }
  }
}
.qr-dialog-content {
  text-align: center;
}
.qr-code-large img {
  width: 250px;
  height: 250px;
  border-radius: 8px;
  margin-bottom: 16px;
}
.qr-info {
  color: #666;
}
.qr-info :is(p) {
  margin-bottom: 16px;
}
.qr-tip {
  font-size: 12px;
  color: #909399;
  margin-bottom: 16px !important;
}
@media (max-width: 768px) {
  .qr-code img {
    width: 40px;
    height: 40px;
  }
  .mobile-card-list {
    .mobile-card.sub-card {
      padding: 0 !important;
      border-radius: 14px;
      .sub-card-header {
        display: flex;
        align-items: flex-start;
        gap: 10px;
        padding: 14px 16px;
        border-bottom: 1px solid #f0f0f0;
        .sub-user-info {
          display: flex;
          align-items: flex-start;
          gap: 10px;
          flex: 1;
          min-width: 0;
        }
        .el-avatar {
          flex-shrink: 0;
          margin-top: 2px;
        }
        .sub-user-meta {
          flex: 1;
          min-width: 0;
        }
        .sub-user-email {
          font-weight: 600;
          font-size: 14px;
          color: #303133;
          word-break: break-all;
          line-height: 1.4;
        }
        .sub-user-id {
          font-size: 12px;
          color: #999;
          margin-top: 3px;
          display: flex;
          align-items: center;
          flex-wrap: wrap;
          gap: 2px;
        }
        .sub-goto-btn {
          flex-shrink: 0;
          align-self: center;
          border-radius: 20px;
          font-size: 12px;
          padding: 6px 14px !important;
          height: auto !important;
          min-height: 0 !important;
        }
      }
      .sub-section {
        padding: 12px 16px;
        border-bottom: 1px solid #f5f5f5;
        border-radius: 0;
        transition: background 0.3s;
        &.expire-time-expired {
          background: #fef0f0;
        }
        &.device-limit-overlimit {
          background: #fef0f0;
        }
        .sub-section-row {
          display: flex;
          align-items: center;
          gap: 8px;
          margin-bottom: 10px;
        }
        .sub-section-icon {
          color: #909399;
          font-size: 16px;
          display: flex;
          align-items: center;
        }
        .sub-section-label {
          font-size: 13px;
          color: #909399;
          flex-shrink: 0;
        }
        .sub-section-value {
          margin-left: auto;
          font-size: 14px;
          font-weight: 600;
          color: #303133;
          flex-shrink: 0;
        }
        .device-limit-input-inline {
          width: 70px;
          flex-shrink: 0;
          margin-left: 8px;
          :deep(.el-input__wrapper) {
            padding: 0 8px;
          }
          :deep(.el-input__inner) {
            text-align: center;
            font-size: 13px;
            font-weight: 600;
          }
        }
        .sub-btn-row {
          display: grid !important;
          grid-template-columns: repeat(4, 1fr) !important;
          gap: 6px;
          margin-bottom: 8px;
          .el-button {
            margin: 0 !important;
            padding: 0 4px !important;
            font-size: 12px !important;
            height: 32px !important;
            min-height: 0 !important;
            border-radius: 6px !important;
            width: 100% !important;
          }
          &.first-btn-row {
            grid-template-columns: 1.5fr 1fr !important;
          }
          &.second-btn-row {
            grid-template-columns: repeat(4, 1fr) !important;
          }
        }
        .sub-date-picker-row {
          width: 100%;
          :deep(.el-date-editor) {
            width: 100% !important;
            height: 32px !important;
            .el-input__wrapper {
              padding: 0 8px !important;
            }
            .el-input__inner {
              font-size: 12px !important;
              height: 30px !important;
              min-height: 0 !important;
            }
          }
        }
      }
      .sub-action-grid {
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 0;
        padding: 12px 8px;
        border-bottom: 1px solid #f5f5f5;
        &:last-child {
          border-bottom: none;
        }
        .sub-action-item {
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 6px;
          padding: 8px 4px;
          cursor: pointer;
          border-radius: 8px;
          transition: background 0.2s;
          &:active {
            background: #f5f7fa;
          }
          .sub-action-icon {
            width: 44px;
            height: 44px;
            border-radius: 12px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 20px;
            transition: transform 0.2s;
          }
          .sub-action-text {
            font-size: 11px;
            color: #606266;
            text-align: center;
            line-height: 1.3;
          }
        }
      }
    }
    .mobile-card {
      .card-row {
        padding: 12px;
        border-radius: 8px;
        transition: all 0.3s;
        &.expire-time-expired {
          background: #fef0f0;
          border: 1px solid #f56c6c;
          animation: pulse-alert 2s ease-in-out infinite;
        }
        &.device-limit-overlimit {
          background: #fef0f0;
          border: 1px solid #f56c6c;
          animation: pulse-alert 2s ease-in-out infinite;
        }
        .value {
          :deep(.el-date-picker) {
            width: 100%;
            .el-input__wrapper {
              min-height: 40px;
              border-radius: 6px;
              background: rgba(255, 255, 255, 0.95);
              box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
              border: 1px solid rgba(66, 165, 245, 0.3);
              &:hover {
                border-color: rgba(66, 165, 245, 0.5);
                box-shadow: 0 2px 6px rgba(66, 165, 245, 0.2);
              }
              &.is-focus {
                border-color: #409eff;
                box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.15);
              }
            }
            .el-input__inner {
              font-size: 0.875rem;
              color: #303133;
            }
          }
          :deep(.el-input-number) {
            width: 100px;
            flex-shrink: 0;
            .el-input__wrapper {
              min-height: 40px;
              border-radius: 6px;
              background: rgba(255, 255, 255, 0.95);
              box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
              border: 1px solid rgba(66, 165, 245, 0.3);
              &:hover {
                border-color: rgba(66, 165, 245, 0.5);
                box-shadow: 0 2px 6px rgba(66, 165, 245, 0.2);
              }
              &.is-focus {
                border-color: #409eff;
                box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.15);
              }
            }
            .el-input__inner {
              font-size: 0.875rem;
              text-align: center;
            }
          }
          display: flex;
          align-items: center;
          justify-content: flex-end;
          gap: 8px;
          flex-wrap: wrap;
          .expire-time-section {
            width: 100%;
            display: flex;
            flex-direction: column;
            gap: 8px;
            .quick-time-buttons {
              display: flex;
              gap: 6px;
              flex-wrap: wrap;
              .el-button {
                flex: 1;
                min-width: 0;
                font-size: 0.75rem;
                padding: 6px 8px;
                min-height: 32px;
                border-radius: 6px;
                &:active {
                  transform: scale(0.96);
                }
              }
            }
          }
        }
      }
    }
  }
  :deep(.mobile-subscription-date-picker-popper) {
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
        }
      }
    }
  }
}
.device-card {
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 16px;
  background-color: #fafafa;
}
.device-main-info {
  margin-bottom: 12px;
}
.device-header-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.device-software {
  display: flex;
  align-items: center;
  gap: 8px;
}
.software-tag {
  font-weight: 500;
}
.software-version {
  font-size: 12px;
  color: #606266;
}
.device-details {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-bottom: 12px;
}
.device-info-row {
  display: flex;
  align-items: center;
  font-size: 13px;
}
.info-label {
  font-weight: 500;
  color: #606266;
  margin-right: 8px;
  min-width: 80px;
}
.info-value {
  color: #303133;
  font-family: monospace;
}
.device-ua-section {
  border-top: 1px solid #e4e7ed;
  padding-top: 12px;
}
.ua-label {
  font-size: 12px;
  font-weight: 500;
  color: #606266;
  margin-bottom: 4px;
}
.ua-content {
  font-size: 11px;
  color: #909399;
  font-family: monospace;
  background-color: #f5f7fa;
  padding: 8px;
  border-radius: 4px;
  word-break: break-all;
  line-height: 1.4;
  max-height: 60px;
  overflow-y: auto;
}
.device-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
.sub-urls-stacked {
  display: flex;
  flex-direction: column;
  gap: 6px;
  .sub-url-row {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }
  .sub-url-label {
    font-size: 11px;
    color: #909399;
    flex-shrink: 0;
    min-width: 32px;
  }
  .link-text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }
}
.sub-count-stacked {
  display: flex;
  flex-direction: column;
  gap: 4px;
  .sub-count-row {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
  }
  .sub-count-label {
    font-size: 11px;
    color: #909399;
    min-width: 30px;
    text-align: right;
  }
}
.notes-input-wrapper {
  position: relative;
  .notes-input {
    :deep(.el-textarea__inner) {
      font-size: 12px;
      line-height: 1.4;
      padding: 6px 8px;
      resize: none;
    }
  }
  .saving-indicator, .saved-indicator {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 11px;
    margin-top: 2px;
    color: #909399;
  }
  .saved-indicator {
    color: #67c23a;
  }
}
:deep(.notes-column) {
  background-color: var(--el-fill-color-lighter, #fafafa) !important;
}
:deep(.notes-column .cell) {
  padding: 8px !important;
  background-color: var(--el-fill-color-lighter, #fafafa) !important;
}
.column-settings {
  .settings-header {
    margin-bottom: 20px;
    :is(p) {
      margin: 0 0 15px 0;
      color: #606266;
      font-size: 14px;
    }
    .quick-actions {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
    }
  }
  .column-checkboxes {
    .checkbox-row {
      display: flex;
      flex-wrap: wrap;
      gap: 20px;
      margin-bottom: 15px;
      .el-checkbox {
        min-width: 120px;
        margin-right: 0;
      }
    }
  }
  .settings-footer {
    margin-top: 20px;
    padding-top: 15px;
    border-top: 1px solid #ebeef5;
    .tip {
      margin: 0;
      color: #909399;
      font-size: 12px;
      line-height: 1.5;
    }
  }
}
@media (max-width: 768px) {
  .column-settings {
    .column-checkboxes .checkbox-row {
      flex-direction: column;
      gap: 10px;
      .el-checkbox {
        min-width: auto;
      }
    }
    .settings-header .quick-actions {
      flex-direction: column;
    }
  }
}
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
.subscription-qrcode-section {
  margin-top: 16px;
  margin-bottom: 16px;
  padding-top: 16px;
  border-top: 2px solid rgba(66, 165, 245, 0.3);
}
.qrcode-card {
  background: linear-gradient(135deg, rgba(66, 165, 245, 0.08) 0%, rgba(102, 126, 234, 0.08) 100%);
  border: 2px solid rgba(66, 165, 245, 0.25);
  border-radius: 16px;
  padding: 20px;
  transition: all 0.3s ease;
  &:active {
    transform: scale(0.98);
    border-color: rgba(66, 165, 245, 0.4);
  }
  .qrcode-header {
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 20px;
    font-weight: 600;
    font-size: 1rem;
    color: #1e293b;
    .qrcode-title {
      font-size: 1.05rem;
      color: #409eff;
    }
  }
  .qrcode-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    .qrcode-wrapper {
      display: flex;
      justify-content: center;
      align-items: center;
      width: 100%;
      padding: 20px;
      background: #ffffff;
      border-radius: 12px;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
      .qrcode-image {
        width: 100%;
        max-width: 320px;
        min-width: 250px;
        height: auto;
        border-radius: 8px;
        display: block;
        margin: 0 auto;
      }
    }
    .qrcode-info {
      width: 100%;
      text-align: center;
      .expiry-info {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 6px;
        padding: 12px;
        background: rgba(245, 108, 108, 0.1);
        border-radius: 8px;
        border: 1px solid rgba(245, 108, 108, 0.2);
        .expiry-label {
          font-size: 0.9rem;
          color: #606266;
          font-weight: 500;
        }
        .expiry-date {
          font-size: 0.95rem;
          color: #f56c6c;
          font-weight: 600;
        }
      }
    }
    .qrcode-actions {
      display: flex;
      flex-direction: column;
      align-items: stretch;
      gap: 12px;
      width: 100%;
      .import-btn,
      .view-btn {
        width: 100%;
        min-width: 100%;
        max-width: 100%;
        height: 48px;
        min-height: 48px;
        max-height: 48px;
        font-size: 1rem;
        font-weight: 600;
        border-radius: 10px;
        box-shadow: 0 2px 8px rgba(64, 158, 255, 0.3);
        transition: all 0.3s ease;
        display: flex;
        align-items: center;
        justify-content: center;
        margin: 0;
        padding: 0;
        box-sizing: border-box;
        &:active {
          transform: scale(0.96);
          box-shadow: 0 1px 4px rgba(64, 158, 255, 0.2);
        }
        :deep(.el-icon) {
          margin-right: 6px;
          flex-shrink: 0;
        }
        :deep(span) {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 100%;
        }
      }
      .import-btn {
        background: linear-gradient(135deg, #409eff 0%, #667eea 100%);
        border: none;
        color: #ffffff;
        &:hover {
          background: linear-gradient(135deg, #66b1ff 0%, #7c8ff0 100%);
        }
      }
      .view-btn {
        background: #ffffff;
        border: 1px solid #dcdfe6;
        color: #606266;
        &:hover {
          background: #f5f7fa;
          border-color: #c0c4cc;
          color: #409eff;
        }
      }
    }
  }
}
.subscription-urls-section {
  margin-top: 16px;
  margin-bottom: 16px;
  padding-top: 16px;
  border-top: 2px solid rgba(224, 231, 255, 0.6);
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.subscription-url-card {
  background: linear-gradient(135deg, rgba(66, 165, 245, 0.05) 0%, rgba(102, 126, 234, 0.05) 100%);
  border: 1.5px solid rgba(66, 165, 245, 0.2);
  border-radius: 10px;
  padding: 12px;
  transition: all 0.3s ease;
  &:active {
    transform: scale(0.98);
    border-color: rgba(66, 165, 245, 0.4);
  }
  .url-header {
    display: flex;
    align-items: center;
    margin-bottom: 10px;
    font-weight: 600;
    font-size: 0.9rem;
    color: #1e293b;
    .url-type {
      font-size: 0.95rem;
    }
  }
  .url-content {
    display: flex;
    flex-direction: column;
    gap: 10px;
    .url-text {
      background: rgba(255, 255, 255, 0.8);
      border: 1px solid rgba(66, 165, 245, 0.15);
      border-radius: 6px;
      padding: 10px 12px;
      font-size: 0.85rem;
      color: #1e293b;
      word-break: break-all;
      line-height: 1.6;
      font-family: 'Courier New', monospace;
      min-height: 44px;
      display: flex;
      align-items: flex-start;
      white-space: pre-wrap;
      overflow-wrap: break-word;
      max-height: 120px;
      overflow-y: auto;
    }
    .copy-url-btn {
      width: 100%;
      min-height: 44px;
      font-size: 0.9rem;
      font-weight: 600;
      border-radius: 8px;
      box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
      transition: all 0.2s ease;
      :deep(.el-icon) {
        margin-right: 6px;
        font-size: 16px;
      }
      &:active {
        transform: scale(0.96);
      }
    }
  }
  &:has(.url-header .el-icon[style*="f56c6c"]) {
    background: linear-gradient(135deg, rgba(245, 108, 108, 0.05) 0%, rgba(255, 152, 0, 0.05) 100%);
    border-color: rgba(245, 108, 108, 0.2);
    .url-content .url-text {
      border-color: rgba(245, 108, 108, 0.15);
    }
  }
}
@media (max-width: 768px) {
  .subscription-urls-section {
    margin-top: 12px;
    margin-bottom: 12px;
    padding-top: 12px;
    gap: 10px;
  }
  .subscription-url-card {
    padding: 10px;
    .url-header {
      margin-bottom: 8px;
      font-size: 0.85rem;
    }
    .url-content {
      gap: 8px;
      .url-text {
        padding: 8px 10px;
        font-size: 0.8rem;
        min-height: 40px;
      }
      .copy-url-btn {
        min-height: 40px;
        font-size: 0.85rem;
      }
    }
  }
}
</style>
