<template>
  <div class="list-container admin-subscriptions">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>订阅列表</span>
          <!-- 桌面端操作按钮 -->
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

      <!-- 移动端智能操作栏 -->
      <div class="mobile-action-bar">
        <!-- 搜索栏（移动端优先显示） -->
        <div class="mobile-search-section">
          <div class="search-input-wrapper">
            <el-input
              v-model="searchForm.keyword"
              placeholder="输入QQ或订阅地址查询"
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

        <!-- 快速操作按钮组 -->
        <div class="mobile-quick-actions">
          <!-- 排序快捷按钮 -->
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

          <!-- 功能操作按钮组 -->
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

      <!-- 桌面端搜索栏 -->
      <el-form :inline="true" :model="searchForm" class="search-form desktop-only">
        <el-form-item label="搜索">
          <el-input 
            v-model="searchForm.keyword" 
            placeholder="输入QQ或订阅地址进行搜索"
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

      <!-- 桌面端表格 -->
      <div class="table-wrapper">
        <el-table 
          :data="subscriptions" 
          style="width: 100%" 
          v-loading="loading"
          @selection-change="handleSelectionChange"
          row-key="id"
          stripe
        >
        <!-- 选择列 -->
        <el-table-column type="selection" width="55" />
        
        <!-- QQ号码/邮箱列 -->
        <el-table-column 
          v-if="visibleColumns.includes('qq')" 
          label="QQ号码" 
          width="140" 
          fixed="left"
        >
          <template #default="scope">
            <div class="qq-info">
              <div class="qq-number">{{ scope.row.user?.email || scope.row.user?.username || '未知' }}</div>
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
        
        <!-- 结束时间列 -->
        <el-table-column 
          v-if="visibleColumns.includes('expire_time')" 
          label="结束时间" 
          width="160"
        >
          <template #default="scope">
            <div class="expire-time-section">
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
        
        <!-- 二维码列 -->
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
                <img :src="generateQRCode(scope.row)" alt="QR Code" />
              </div>
              <el-text v-else type="info" size="small">无订阅</el-text>
            </div>
          </template>
        </el-table-column>
        
        <!-- 通用订阅列 -->
        <el-table-column 
          v-if="visibleColumns.includes('universal_url')" 
          label="通用订阅" 
          width="180"
        >
          <template #default="scope">
            <div class="subscription-link">
              <el-link 
                v-if="scope.row.universal_url" 
                @click="copyToClipboard(scope.row.universal_url)"
                type="primary"
                class="link-text copy-link"
                :title="'点击复制: ' + scope.row.universal_url"
              >
                {{ scope.row.universal_url }}
              </el-link>
              <el-text v-else type="info" size="small">未配置</el-text>
            </div>
          </template>
        </el-table-column>
        
        <!-- 猫咪订阅列 -->
        <el-table-column 
          v-if="visibleColumns.includes('clash_url')" 
          label="猫咪订阅" 
          width="180"
        >
          <template #default="scope">
            <div class="subscription-link">
              <el-link 
                v-if="scope.row.clash_url" 
                @click="copyToClipboard(scope.row.clash_url)"
                type="primary"
                class="link-text copy-link"
                :title="'点击复制: ' + scope.row.clash_url"
              >
                {{ scope.row.clash_url }}
              </el-link>
              <el-text v-else type="info" size="small">未配置</el-text>
            </div>
          </template>
        </el-table-column>
        
        <!-- 添加时间列 -->
        <el-table-column 
          v-if="visibleColumns.includes('created_at')" 
          label="添加时间" 
          width="160"
        >
          <template #default="scope">
            <div class="created-time">
              {{ formatDate(scope.row.created_at) }}
            </div>
          </template>
        </el-table-column>
        
        <!-- 通用订阅次数列 -->
        <el-table-column 
          v-if="visibleColumns.includes('apple_count')" 
          label="通用订阅次数" 
          width="110" 
          align="center"
        >
          <template #default="scope">
            <el-tooltip content="订阅通用订阅的次数" placement="top">
              <el-tag type="info" size="small">{{ scope.row.apple_count || 0 }}</el-tag>
            </el-tooltip>
          </template>
        </el-table-column>
        
        <!-- 猫咪订阅次数列 -->
        <el-table-column 
          v-if="visibleColumns.includes('clash_count')" 
          label="猫咪订阅次数" 
          width="110" 
          align="center"
        >
          <template #default="scope">
            <el-tooltip content="订阅猫咪订阅的次数" placement="top">
              <el-tag type="warning" size="small">{{ scope.row.clash_count || 0 }}</el-tag>
            </el-tooltip>
          </template>
        </el-table-column>
        
        <!-- 在线列 -->
        <el-table-column 
          v-if="visibleColumns.includes('online_devices')" 
          label="在线" 
          width="70" 
          align="center"
        >
          <template #default="scope">
            <el-tooltip content="当前在线设备数" placement="top">
              <el-tag type="success" size="small">{{ scope.row.online_devices || 0 }}</el-tag>
            </el-tooltip>
          </template>
        </el-table-column>
        
        <!-- 最大设备数列 -->
        <el-table-column 
          v-if="visibleColumns.includes('device_limit')" 
          label="最大设备数" 
          width="130"
          sortable="custom"
          :sort-orders="['descending', 'ascending']"
          @sort-change="handleDeviceLimitSort"
        >
          <template #default="scope">
            <div class="device-limit-section">
              <el-input-number
                v-model="scope.row.device_limit"
                :min="0"
                :max="999"
                size="small"
                @change="updateDeviceLimit(scope.row)"
                class="device-limit-input"
              />
              <div class="quick-device-buttons">
                <el-button size="small" @click="addDeviceLimit(scope.row, 5)">+5</el-button>
                <el-button size="small" @click="addDeviceLimit(scope.row, 10)">+10</el-button>
                <el-button size="small" @click="addDeviceLimit(scope.row, 15)">+15</el-button>
              </div>
            </div>
          </template>
        </el-table-column>
        
        <!-- 操作列 -->
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

      <!-- 移动端卡片式列表 -->
      <div class="mobile-card-list" v-if="subscriptions.length > 0">
        <div 
          v-for="subscription in subscriptions" 
          :key="subscription.id"
          class="mobile-card"
        >
          <div class="card-row">
            <span class="label">用户信息</span>
            <span class="value">
              <div style="display: flex; align-items: center; gap: 8px;">
                <el-avatar :size="32" :src="subscription.user?.avatar">
                  {{ subscription.user?.username?.charAt(0)?.toUpperCase() || 'U' }}
                </el-avatar>
                <div>
                  <div style="font-weight: 600; color: #303133;">
                    {{ subscription.user?.email || subscription.user?.username || subscription.email || subscription.username || '未知用户' }}
                  </div>
                  <div style="font-size: 0.85rem; color: #999;">
                    ID: #{{ subscription.user?.id || subscription.user_id || subscription.id }}
                    <el-tag v-if="subscription.user?.deleted" type="danger" size="small" style="margin-left: 8px;">用户已删除</el-tag>
                  </div>
                </div>
              </div>
            </span>
          </div>
          <div class="card-row">
            <span class="label">订阅状态</span>
            <span class="value">
              <el-tag :type="getSubscriptionStatusType(subscription.status)" size="small">
                {{ getSubscriptionStatusText(subscription.status) }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">到期时间</span>
            <span class="value">
              <div class="expire-time-section">
                <el-date-picker
                  v-model="subscription.expire_time"
                  type="date"
                  placeholder="选择日期"
                  format="YYYY-MM-DD"
                  value-format="YYYY-MM-DD"
                  size="small"
                  style="width: 100%; margin-bottom: 8px;"
                  @change="updateExpireTime(subscription)"
                  clearable
                  teleported
                  popper-class="mobile-subscription-date-picker-popper"
                />
                <div class="quick-time-buttons">
                  <el-button 
                    size="small" 
                    @click="addTime(subscription, 30)"
                    plain
                  >
                    +1个月
                  </el-button>
                  <el-button 
                    size="small" 
                    @click="addTime(subscription, 180)"
                    plain
                  >
                    +半年
                  </el-button>
                  <el-button 
                    size="small" 
                    @click="addTime(subscription, 365)"
                    plain
                  >
                    +1年
                  </el-button>
                </div>
              </div>
            </span>
          </div>
          <div class="card-row">
            <span class="label">设备使用</span>
            <span class="value">
              <el-tooltip content="当前在线设备数" placement="top">
                <el-tag type="success" size="small">{{ subscription.online_devices || 0 }}</el-tag>
              </el-tooltip>
              <span style="margin: 0 4px;">/</span>
              <el-input-number
                v-model="subscription.device_limit"
                :min="0"
                :max="999"
                size="small"
                style="width: 100px; flex-shrink: 0;"
                @change="updateDeviceLimit(subscription)"
                controls-position="right"
              />
            </span>
          </div>
          <div class="card-row">
            <span class="label">设备统计</span>
            <span class="value">
              <el-tooltip content="订阅通用订阅的次数" placement="top">
                <el-tag type="info" size="small" style="margin-right: 4px;">
                  通用订阅: {{ subscription.apple_count || 0 }}
                </el-tag>
              </el-tooltip>
              <el-tooltip content="订阅猫咪订阅的次数" placement="top">
                <el-tag type="warning" size="small" style="margin-right: 4px;">
                  猫咪订阅: {{ subscription.clash_count || 0 }}
                </el-tag>
              </el-tooltip>
            </span>
          </div>
          <!-- 订阅二维码区域 - 移动端显示 -->
          <div class="subscription-qrcode-section" v-if="subscription.subscription_url || subscription.universal_url">
            <div class="qrcode-card">
              <div class="qrcode-header">
                <el-icon style="color: #409eff; margin-right: 6px; font-size: 18px;"><Link /></el-icon>
                <span class="qrcode-title">Shadowrocket 订阅二维码</span>
              </div>
              <div class="qrcode-content">
                <div class="qrcode-wrapper">
                  <img 
                    :src="generateQRCode(subscription)" 
                    alt="订阅二维码" 
                    class="qrcode-image"
                  />
                </div>
                <div class="qrcode-info" v-if="subscription.expire_time">
                  <div class="expiry-info">
                    <el-icon style="color: #f56c6c; margin-right: 4px;"><Clock /></el-icon>
                    <span class="expiry-label">失效日期：</span>
                    <span class="expiry-date">{{ formatExpireDate(subscription.expire_time) }}</span>
                  </div>
                </div>
                <div class="qrcode-actions">
                  <el-button 
                    type="primary" 
                    size="default"
                    @click="importToShadowrocket(subscription)"
                    class="import-btn"
                  >
                    <el-icon style="margin-right: 6px;"><Download /></el-icon>
                    一键导入 Shadowrocket
                  </el-button>
                  <el-button 
                    type="default" 
                    size="default"
                    @click="showQRCode(subscription)"
                    class="view-btn"
                  >
                    <el-icon style="margin-right: 6px;"><View /></el-icon>
                    查看大图
                  </el-button>
                </div>
              </div>
            </div>
          </div>
          
          <!-- 订阅地址区域 - 独立卡片样式 -->
          <div class="subscription-urls-section" v-if="subscription.universal_url || subscription.clash_url">
            <div class="subscription-url-card" v-if="subscription.universal_url">
              <div class="url-header">
                <el-icon style="color: #409eff; margin-right: 6px;"><Link /></el-icon>
                <span class="url-type">通用订阅</span>
              </div>
              <div class="url-content">
                <div class="url-text" :title="subscription.universal_url">
                  {{ truncateUrl(subscription.universal_url) }}
                </div>
                <el-button 
                  type="primary" 
                  size="small"
                  @click="copyToClipboard(subscription.universal_url)"
                  class="copy-url-btn"
                >
                  <el-icon><DocumentCopy /></el-icon>
                  复制
                </el-button>
              </div>
            </div>
            <div class="subscription-url-card" v-if="subscription.clash_url">
              <div class="url-header">
                <el-icon style="color: #f56c6c; margin-right: 6px;"><Link /></el-icon>
                <span class="url-type">猫咪订阅</span>
              </div>
              <div class="url-content">
                <div class="url-text" :title="subscription.clash_url">
                  {{ truncateUrl(subscription.clash_url) }}
                </div>
                <el-button 
                  type="danger" 
                  size="small"
                  @click="copyToClipboard(subscription.clash_url)"
                  class="copy-url-btn"
                >
                  <el-icon><DocumentCopy /></el-icon>
                  复制
                </el-button>
              </div>
            </div>
          </div>
          <div class="card-actions">
            <el-button 
              size="small" 
              type="success" 
              @click="showUserDetails(subscription)"
            >
              <el-icon><View /></el-icon>
              详情
            </el-button>
            <el-button 
              size="small" 
              type="primary" 
              @click="goToUserBackend(subscription)"
            >
              <el-icon><User /></el-icon>
              后台
            </el-button>
            <el-button 
              size="small" 
              type="warning" 
              @click="resetSubscription(subscription)"
            >
              <el-icon><Refresh /></el-icon>
              重置
            </el-button>
            <el-button 
              size="small" 
              :type="subscription.is_active ? 'danger' : 'success'"
              @click="toggleSubscriptionStatus(subscription)"
            >
              <el-icon><Switch /></el-icon>
              {{ subscription.is_active ? '禁用' : '启用' }}
            </el-button>
            <el-button 
              size="small" 
              type="info" 
              @click="sendSubscriptionEmail(subscription)"
            >
              <el-icon><Message /></el-icon>
              邮件
            </el-button>
            <el-button 
              size="small" 
              type="danger" 
              @click="clearUserDevices(subscription)"
            >
              <el-icon><Delete /></el-icon>
              清理
            </el-button>
          </div>
        </div>
      </div>

      <!-- 移动端空状态 -->
      <div class="mobile-card-list" v-if="subscriptions.length === 0 && !loading">
        <div class="empty-state">
          <i class="el-icon-document"></i>
          <p>暂无订阅记录</p>
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

    <!-- 用户详情对话框 -->
    <el-dialog 
      v-model="showUserDetailDialog" 
      title="用户详细信息" 
      :width="isMobile ? '95%' : '1000px'"
      :close-on-click-modal="false"
      :fullscreen="isMobile"
      class="user-detail-dialog"
    >
      <div v-if="selectedUser" class="user-detail-content">
        <!-- 基本信息 -->
        <el-card class="detail-section">
          <template #header>
            <h4>基本信息</h4>
          </template>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="用户ID">{{ selectedUser.user?.id }}</el-descriptions-item>
            <el-descriptions-item label="用户名">{{ selectedUser.user?.username }}</el-descriptions-item>
            <el-descriptions-item label="邮箱">{{ selectedUser.user?.email }}</el-descriptions-item>
            <el-descriptions-item label="注册时间">{{ formatDate(selectedUser.user?.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="最后登录">{{ formatDate(selectedUser.user?.last_login) || '从未登录' }}</el-descriptions-item>
            <el-descriptions-item label="激活状态">
              <el-tag :type="selectedUser.user?.is_active ? 'success' : 'danger'">
                {{ selectedUser.user?.is_active ? '已激活' : '未激活' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="邮箱验证">
              <el-tag :type="selectedUser.user?.is_verified ? 'success' : 'warning'">
                {{ selectedUser.user?.is_verified ? '已验证' : '未验证' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="管理员权限">
              <el-tag :type="selectedUser.user?.is_admin ? 'danger' : 'info'">
                {{ selectedUser.user?.is_admin ? '是' : '否' }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>

        <!-- 订阅信息 -->
        <el-card class="detail-section">
          <template #header>
            <h4>订阅信息</h4>
          </template>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="订阅状态">
              <el-tag :type="getSubscriptionStatusType(selectedUser.status)">
                {{ getSubscriptionStatusText(selectedUser.status) }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="到期时间">{{ formatDate(selectedUser.expire_time) }}</el-descriptions-item>
            <el-descriptions-item label="设备限制">{{ selectedUser.device_limit }}</el-descriptions-item>
            <el-descriptions-item label="在线设备">
              <el-tooltip content="当前在线设备数" placement="top">
                <span>{{ selectedUser.online_devices || 0 }}</span>
              </el-tooltip>
            </el-descriptions-item>
            <el-descriptions-item label="通用订阅次数">
              <el-tooltip content="订阅通用订阅（V2Ray/SSR）的次数" placement="top">
                <span>{{ selectedUser.v2ray_count || selectedUser.apple_count || 0 }}</span>
              </el-tooltip>
            </el-descriptions-item>
            <el-descriptions-item label="猫咪订阅次数">
              <el-tooltip content="订阅猫咪订阅（Clash）的次数" placement="top">
                <span>{{ selectedUser.clash_count || 0 }}</span>
              </el-tooltip>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>

        <!-- 订阅地址 -->
        <el-card class="detail-section">
          <template #header>
            <h4>订阅地址</h4>
          </template>
          <div class="subscription-urls">
            <div class="url-item">
              <label>通用订阅地址:</label>
              <el-input :value="selectedUser.universal_url" readonly>
                <template #append>
                  <el-button @click="copyToClipboard(selectedUser.universal_url)">复制</el-button>
                </template>
              </el-input>
            </div>
            <div class="url-item">
              <label>猫咪订阅地址:</label>
              <el-input v-model="selectedUser.clash_url" readonly>
                <template #append>
                  <el-button @click="copyToClipboard(selectedUser.clash_url)">复制</el-button>
                </template>
              </el-input>
            </div>
          </div>
        </el-card>

        <!-- 设备管理 -->
        <el-card class="detail-section">
          <template #header>
            <div class="device-header">
              <h4>设备管理</h4>
              <div class="device-stats">
                <el-tag type="info">在线设备: {{ selectedUser.online_devices || 0 }}/{{ selectedUser.device_limit || 0 }}</el-tag>
                <el-button 
                  type="primary" 
                  size="small" 
                  @click="loadUserDevices"
                  :loading="loadingDevices"
                >
                  刷新设备列表
                </el-button>
              </div>
            </div>
          </template>
          
          <div class="device-table-wrapper">
            <el-table 
              :data="userDevices" 
              size="small" 
              v-loading="loadingDevices"
              empty-text="暂无设备记录"
              class="device-table"
            >
            <el-table-column prop="device_name" label="设备名称" min-width="250">
              <template #default="scope">
                <div class="device-info">
                  <el-icon><Monitor /></el-icon>
                  <div class="device-name-details">
                    <div class="device-main-name">
                      <span class="device-name-text">{{ scope.row.device_name || '未知设备' }}</span>
                      <el-tag v-if="scope.row.software_name" type="info" size="small" style="margin-left: 8px;">
                        {{ scope.row.software_name }}{{ scope.row.software_version ? ' ' + scope.row.software_version : '' }}
                      </el-tag>
                    </div>
                    <div v-if="scope.row.device_model" class="device-model-info">
                      <el-tag type="success" size="small" style="margin-top: 4px;">
                        {{ scope.row.device_model }}{{ scope.row.device_brand && scope.row.device_brand !== 'Apple' ? ' (' + scope.row.device_brand + ')' : '' }}
                      </el-tag>
                    </div>
                  </div>
                </div>
              </template>
            </el-table-column>
            
            <el-table-column prop="device_type" label="设备类型" width="100">
              <template #default="scope">
                <el-tag v-if="scope.row.device_type && scope.row.device_type !== 'unknown'" 
                        :type="getDeviceTypeTag(scope.row.device_type)" 
                        size="small">
                  {{ getDeviceTypeText(scope.row.device_type) }}
                </el-tag>
                <span v-else style="color: #909399; font-size: 12px;">-</span>
              </template>
            </el-table-column>
            
            <el-table-column prop="ip_address" label="IP地址" width="120" />
            
            <el-table-column prop="os_name" label="操作系统" width="180">
              <template #default="scope">
                <div class="os-info">
                  <div class="os-name">{{ scope.row.os_name || '-' }}</div>
                  <div v-if="scope.row.os_version" class="os-version">
                    <el-tag type="primary" size="small" style="margin-top: 4px;">
                      {{ scope.row.os_version }}
                    </el-tag>
                  </div>
                </div>
              </template>
            </el-table-column>
            
            <el-table-column prop="last_seen" label="最后在线" width="150">
              <template #default="scope">
                <span>{{ formatDate(scope.row.last_seen || scope.row.last_access) || '从未在线' }}</span>
              </template>
            </el-table-column>
            
            <el-table-column prop="access_count" label="访问次数" width="100">
              <template #default="scope">
                <el-tag type="info" size="small">{{ scope.row.access_count || 0 }}</el-tag>
              </template>
            </el-table-column>
            
            <el-table-column prop="is_active" label="状态" width="80">
              <template #default="scope">
                <el-tag :type="scope.row.is_active ? 'success' : 'danger'" size="small">
                  {{ scope.row.is_active ? '活跃' : '离线' }}
                </el-tag>
              </template>
            </el-table-column>
            
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="scope">
                <el-button 
                  type="danger" 
                  size="small" 
                  @click="deleteDevice(scope.row)"
                  :loading="deletingDevice === scope.row.id"
                >
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          </div>
          
          <div v-if="userDevices.length === 0 && !loadingDevices" class="empty-devices">
            <el-empty description="暂无设备记录">
              <el-button type="primary" @click="loadUserDevices">刷新设备列表</el-button>
            </el-empty>
          </div>
        </el-card>

        <!-- UA记录 -->
        <el-card class="detail-section">
          <template #header>
            <h4>UA记录</h4>
          </template>
          <el-table :data="selectedUser.ua_records || []" size="small" empty-text="暂无UA记录">
            <el-table-column prop="user_agent" label="User Agent" min-width="200">
              <template #default="scope">
                <el-tooltip :content="scope.row.user_agent || '未知'" placement="top">
                  <span class="user-agent-text">{{ truncateUserAgent(scope.row.user_agent) }}</span>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column prop="device_type" label="设备类型" width="100">
              <template #default="scope">
                <el-tag v-if="scope.row.device_type && scope.row.device_type !== 'unknown'" 
                        :type="getDeviceTypeTag(scope.row.device_type)" 
                        size="small">
                  {{ getDeviceTypeText(scope.row.device_type) }}
                </el-tag>
                <span v-else style="color: #909399; font-size: 12px;">-</span>
              </template>
            </el-table-column>
            <el-table-column prop="device_name" label="设备名称" width="180">
              <template #default="scope">
                <span>{{ scope.row.device_name || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="首次访问" width="160">
              <template #default="scope">
                <span>{{ formatDate(scope.row.created_at) || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="last_access" label="最后访问" width="160">
              <template #default="scope">
                <span>{{ formatDate(scope.row.last_access) || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="ip_address" label="IP地址" width="120">
              <template #default="scope">
                <span>{{ scope.row.ip_address || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="access_count" label="访问次数" width="100">
              <template #default="scope">
                <span>{{ scope.row.access_count || 0 }}</span>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <!-- 订阅重置记录 -->
        <el-card class="detail-section" v-if="selectedUser.user && selectedUser.user.subscription_resets && selectedUser.user.subscription_resets.length > 0">
          <template #header>
            <h4>
              <el-icon><Refresh /></el-icon>
              订阅重置记录
            </h4>
          </template>
          <el-table :data="selectedUser.user.subscription_resets || []" size="small" empty-text="暂无重置记录" border>
            <el-table-column prop="created_at" label="重置时间" width="180">
              <template #default="scope">
                <span>{{ formatDate(scope.row.created_at) || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="reset_type" label="重置类型" width="120">
              <template #default="scope">
                <el-tag :type="getResetTypeTag(scope.row.reset_type)" size="small">
                  {{ getResetTypeText(scope.row.reset_type) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="reset_by" label="操作人" width="100">
              <template #default="scope">
                <el-tag :type="getResetByTag(scope.row.reset_by)" size="small">
                  {{ getResetByText(scope.row.reset_by) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="old_subscription_url" label="旧订阅地址" min-width="200">
              <template #default="scope">
                <div class="subscription-url-cell">
                  <el-input 
                    v-model="scope.row.old_subscription_url" 
                    readonly 
                    size="small"
                    class="url-input"
                  >
                    <template #append>
                      <el-button 
                        size="small" 
                        @click="copyToClipboard(scope.row.old_subscription_url)"
                        title="复制旧订阅地址"
                      >
                        <el-icon><DocumentCopy /></el-icon>
                      </el-button>
                    </template>
                  </el-input>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="new_subscription_url" label="新订阅地址" min-width="200">
              <template #default="scope">
                <div class="subscription-url-cell">
                  <el-input 
                    v-model="scope.row.new_subscription_url" 
                    readonly 
                    size="small"
                    class="url-input"
                  >
                    <template #append>
                      <el-button 
                        size="small" 
                        @click="copyToClipboard(scope.row.new_subscription_url)"
                        title="复制新订阅地址"
                      >
                        <el-icon><DocumentCopy /></el-icon>
                      </el-button>
                    </template>
                  </el-input>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="device_count_before" label="重置前设备数" width="120" align="center">
              <template #default="scope">
                <el-tag type="info" size="small">{{ scope.row.device_count_before || 0 }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="device_count_after" label="重置后设备数" width="120" align="center">
              <template #default="scope">
                <el-tag type="success" size="small">{{ scope.row.device_count_after || 0 }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="reason" label="重置原因" min-width="150">
              <template #default="scope">
                <span>{{ scope.row.reason || '-' }}</span>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </div>
    </el-dialog>

    <!-- 二维码放大对话框 -->
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

    <!-- 列设置对话框 -->
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
            <el-checkbox label="qq">QQ号码</el-checkbox>
            <el-checkbox label="expire_time">结束时间</el-checkbox>
            <el-checkbox label="qr_code">二维码</el-checkbox>
          </div>
          <div class="checkbox-row">
            <el-checkbox label="universal_url">通用订阅</el-checkbox>
            <el-checkbox label="clash_url">猫咪订阅</el-checkbox>
            <el-checkbox label="created_at">添加时间</el-checkbox>
          </div>
          <div class="checkbox-row">
            <el-checkbox label="apple_count">通用订阅次数</el-checkbox>
            <el-checkbox label="clash_count">猫咪订阅次数</el-checkbox>
            <el-checkbox label="online_devices">在线</el-checkbox>
          </div>
          <div class="checkbox-row">
            <el-checkbox label="device_limit">最大设备数</el-checkbox>
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
import { ref, reactive, onMounted, onUnmounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Download, Delete, Setting, Apple, Monitor, ArrowDown, View, Refresh, HomeFilled,
  Search, Filter, Clock, Sort, Operation, Link, DocumentCopy, User, Message, Switch
} from '@element-plus/icons-vue'
import '@/styles/list-common.scss'
import { adminAPI } from '@/utils/api'
import { secureStorage } from '@/utils/secureStorage'
import { formatDateTime, formatDate as formatDateUtil, formatTime as formatTimeUtil } from '@/utils/date'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
dayjs.extend(timezone)

export default {
  name: 'AdminSubscriptions',
  components: {
    Download, Delete, Setting, Apple, Monitor, ArrowDown, View, Refresh, HomeFilled,
    Search, Clock, Sort, Operation, Link, DocumentCopy, User, Message, Switch
  },
  setup() {
    const loading = ref(false)
    const subscriptions = ref([])
    const selectedSubscriptions = ref([])
    const currentPage = ref(1)
    const pageSize = ref(20)
    const total = ref(0)
    const searchQuery = ref('')
    const currentSort = ref('add_time_desc')
    
    // 搜索表单（与其他列表保持一致）
    const searchForm = reactive({
      keyword: '',
      status: ''
    })
    const showUserDetailDialog = ref(false)
    const showQRDialog = ref(false)
    const showColumnSettings = ref(false)
    const selectedUser = ref(null)
    const currentQRCode = ref('')
    const visibleColumns = ref([
      'qq', 'expire_time', 'qr_code', 'universal_url', 'clash_url', 
      'created_at', 'apple_count', 'clash_count', 'online_devices', 
      'device_limit', 'actions'
    ])
    
    // 设备管理相关
    const userDevices = ref([])
    const loadingDevices = ref(false)
    const deletingDevice = ref(null)

    // 计算当前排序文本
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

    // 加载订阅列表
    const loadSubscriptions = async () => {
      loading.value = true
      try {
        // 同步 searchForm.keyword 到 searchQuery（向后兼容）
        if (searchForm.keyword && !searchQuery.value) {
          searchQuery.value = searchForm.keyword
        }
        
        const params = {
          page: currentPage.value,
          size: pageSize.value,
          search: searchForm.keyword || searchQuery.value,
          sort: currentSort.value
        }
        
        // 添加状态筛选
        if (searchForm.status) {
          params.status = searchForm.status
        }
        
        const response = await adminAPI.getSubscriptions(params)
        if (response.data?.success !== false) {
          const subscriptionList = response.data?.data?.subscriptions || []
          // 确保 is_active 是布尔值
          subscriptions.value = subscriptionList.map(sub => ({
            ...sub,
            is_active: sub.is_active === true || sub.is_active === 1 || sub.is_active === '1'
          }))
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

    // 搜索订阅
    const searchSubscriptions = () => {
      // 同步 searchForm.keyword 到 searchQuery（向后兼容）
      searchQuery.value = searchForm.keyword
      currentPage.value = 1
      loadSubscriptions()
    }
    
    // 重置搜索
    const resetSearch = () => {
      searchForm.keyword = ''
      searchForm.status = ''
      searchQuery.value = ''
      currentPage.value = 1
      loadSubscriptions()
    }
    
    // 处理状态筛选
    const handleStatusFilter = (status) => {
      searchForm.status = status
      currentPage.value = 1
      loadSubscriptions()
    }
    
    // 获取状态筛选文本
    const getStatusFilterText = () => {
      const statusMap = {
        '': '状态筛选',
        'active': '活跃',
        'expired': '已过期'
      }
      return statusMap[searchForm.status] || '状态筛选'
    }

    // 处理排序命令
    const handleSortCommand = (command) => {
      currentSort.value = command
      loadSubscriptions()
    }

    // 清除排序
    const clearSort = () => {
      currentSort.value = 'add_time_desc'
      loadSubscriptions()
    }

    // 处理移动端操作命令
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

    // 更新到期时间 - 立即生效，不需要确认
    const updateExpireTime = async (subscription) => {
      if (!subscription || !subscription.id) return
      
      try {
        await adminAPI.updateSubscription(subscription.id, {
          expire_time: subscription.expire_time
        })
        ElMessage.success('到期时间更新成功')
        // 不重新加载整个列表，只更新当前项
        // loadSubscriptions()
      } catch (error) {
        ElMessage.error('更新到期时间失败: ' + (error.response?.data?.message || error.message))
        // 如果失败，尝试重新加载列表以恢复原始值
        loadSubscriptions()
      }
    }

    // 添加时间 - 立即生效
    const addTime = async (subscription, days) => {
      if (!subscription || !subscription.id) return
      
      try {
        // 如果没有当前到期时间，使用今天作为基准（北京时间）
        let baseDate = subscription.expire_time 
          ? dayjs(subscription.expire_time).tz('Asia/Shanghai')
          : dayjs().tz('Asia/Shanghai')
        
        // 如果日期无效，使用今天（北京时间）
        if (!baseDate.isValid()) {
          baseDate = dayjs().tz('Asia/Shanghai')
        }
        
        const newDate = baseDate.add(days, 'day')
        subscription.expire_time = newDate.format('YYYY-MM-DD')
        
        // 立即调用更新函数
        await updateExpireTime(subscription)
      } catch (error) {
        ElMessage.error('添加时间失败: ' + (error.response?.data?.message || error.message))
        // 如果失败，尝试重新加载列表以恢复原始值
        loadSubscriptions()
      }
    }

    // 更新设备限制 - 立即生效，不需要确认
    const updateDeviceLimit = async (subscription) => {
      if (!subscription || !subscription.id) return
      
      try {
        await adminAPI.updateSubscription(subscription.id, {
          device_limit: subscription.device_limit
        })
        ElMessage.success('设备限制更新成功')
        // 触发自定义事件，通知用户列表刷新
        window.dispatchEvent(new CustomEvent('subscription-device-limit-updated', {
          detail: { subscriptionId: subscription.id, deviceLimit: subscription.device_limit }
        }))
        // 不重新加载整个列表，只更新当前项
        // loadSubscriptions()
      } catch (error) {
        ElMessage.error('更新设备限制失败: ' + (error.response?.data?.message || error.message))
        // 如果失败，尝试重新加载列表以恢复原始值
        loadSubscriptions()
      }
    }

    // 添加设备限制
    const addDeviceLimit = async (subscription, count) => {
      try {
        subscription.device_limit = (subscription.device_limit || 0) + count
        await updateDeviceLimit(subscription)
      } catch (error) {
        ElMessage.error('添加设备限制失败')
        }
    }

    // 生成二维码
    const generateQRCode = (subscription) => {
      if (!subscription) return ''
      
      // 优先使用后端返回的qrcodeUrl（如果存在）
      if (subscription.qrcodeUrl) {
        // 后端已经生成了完整的sub://链接，直接使用
        const qrData = subscription.qrcodeUrl
        // 移动端使用更大的尺寸（400x400），桌面端使用200x200
        const isMobile = window.innerWidth <= 768
        const qrSize = isMobile ? '400x400' : '200x200'
        return `https://api.qrserver.com/v1/create-qr-code/?size=${qrSize}&data=${encodeURIComponent(qrData)}&ecc=M&margin=10`
      }
      
      // 如果没有后端提供的qrcodeUrl，则前端生成
      let qrData = ''
      
      if (subscription.universal_url) {
        // 使用通用订阅URL，生成sub://格式（包含到期时间）
        const universalUrl = subscription.universal_url
        // Base64编码订阅URL
        const encodedUrl = btoa(universalUrl)
        // 格式化到期时间用于Shadowrocket显示（作为订阅名称）
        let expiryDisplayName = ''
        if (subscription.expire_time) {
          const expireDate = dayjs(subscription.expire_time).tz('Asia/Shanghai')
          expiryDisplayName = `到期时间${expireDate.format('YYYY-MM-DD')}`
        } else {
          expiryDisplayName = subscription.subscription_url || '订阅'
        }
        // 生成sub://格式的链接，hash部分是Shadowrocket显示的订阅名称
        qrData = `sub://${encodedUrl}#${encodeURIComponent(expiryDisplayName)}`
      } else if (subscription.subscription_url) {
        // 生成sub://格式的订阅链接
        const baseUrl = window.location.origin
        const subscriptionUrl = `${baseUrl}/api/v1/subscriptions/ssr/${subscription.subscription_url}`
        
        // Base64编码订阅URL
        const encodedUrl = btoa(subscriptionUrl)
        
        // 格式化到期时间用于Shadowrocket显示（作为订阅名称）
        let expiryDisplayName = ''
        if (subscription.expire_time) {
          const expireDate = new Date(subscription.expire_time)
          // 格式化为：到期时间YYYY-MM-DD (用于Shadowrocket显示的订阅名称)
          const year = expireDate.getFullYear()
          const month = String(expireDate.getMonth() + 1).padStart(2, '0')
          const day = String(expireDate.getDate()).padStart(2, '0')
          expiryDisplayName = `到期时间${year}-${month}-${day}`
        } else {
          // 如果没有到期时间，使用订阅密钥作为备用
          expiryDisplayName = subscription.subscription_url
        }
        
        // 生成sub://格式的链接，hash部分是Shadowrocket显示的订阅名称
        qrData = `sub://${encodedUrl}#${encodeURIComponent(expiryDisplayName)}`
      } else {
        return ''
      }
      
      // 生成二维码，移动端使用更大的尺寸（400x400），桌面端使用200x200
      const isMobile = window.innerWidth <= 768
      const qrSize = isMobile ? '400x400' : '200x200'
      return `https://api.qrserver.com/v1/create-qr-code/?size=${qrSize}&data=${encodeURIComponent(qrData)}&ecc=M&margin=10`
    }

    // 显示二维码
    const showQRCode = (subscription) => {
      if (subscription.subscription_url || subscription.universal_url) {
        currentQRCode.value = generateQRCode(subscription)
        showQRDialog.value = true
      }
    }

    // 下载二维码
    const downloadQRCode = () => {
      const link = document.createElement('a')
      link.href = currentQRCode.value
      link.download = 'subscription-qr.png'
      link.click()
    }

    // 格式化失效日期
    const formatExpireDate = (expireTime) => {
      if (!expireTime) return '未设置'
      const date = dayjs(expireTime).tz('Asia/Shanghai')
      if (!date.isValid()) return '未设置'
      const year = date.year()
      const month = String(date.month() + 1).padStart(2, '0')
      const day = String(date.date()).padStart(2, '0')
      return `${year}-${month}-${day}`
    }

    // 一键导入到 Shadowrocket
    const importToShadowrocket = (subscription) => {
      if (!subscription.subscription_url && !subscription.universal_url) {
        ElMessage.warning('该订阅没有可用的订阅地址')
        return
      }

      // 生成订阅URL
      let subscriptionUrl = ''
      if (subscription.qrcodeUrl) {
        // 从 qrcodeUrl 中提取 sub:// 链接
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

      // 尝试直接打开 Shadowrocket 的 sub:// 链接
      try {
        // 生成 sub:// 链接
        const encodedUrl = btoa(subscriptionUrl)
        let expiryDisplayName = ''
        if (subscription.expire_time) {
          expiryDisplayName = formatExpireDate(subscription.expire_time)
        } else {
          expiryDisplayName = '订阅'
        }
        const subLink = `sub://${encodedUrl}#${encodeURIComponent(expiryDisplayName)}`
        
        // 尝试打开 Shadowrocket
        window.location.href = subLink
        
        // 如果无法打开，则复制链接到剪贴板
        setTimeout(() => {
          copyToClipboard(subscriptionUrl)
          ElMessage.success('已复制订阅链接到剪贴板，请在 Shadowrocket 中手动添加')
        }, 500)
      } catch (error) {
        // 如果失败，复制链接
        copyToClipboard(subscriptionUrl)
        ElMessage.success('已复制订阅链接到剪贴板，请在 Shadowrocket 中手动添加')
      }
    }

    // 显示用户详情
    const showUserDetails = async (subscription) => {
      try {
        // 使用正确的API端点获取用户详情
        const userResponse = await adminAPI.getUserDetails(subscription.user.id)
        
        if (userResponse.data && userResponse.data.success) {
          const userData = userResponse.data.data
          selectedUser.value = {
            ...subscription,
            // 保留订阅的订阅次数统计
            apple_count: subscription.apple_count || userData?.apple_count || 0,
            clash_count: subscription.clash_count || userData?.clash_count || 0,
            v2ray_count: subscription.apple_count || userData?.v2ray_count || userData?.apple_count || 0,
            user: {
              ...(userData?.user_info || userData),
              subscription_resets: userData?.subscription_resets || []
            },
            ua_records: userData?.ua_records || []
          }
          showUserDetailDialog.value = true
          // 自动加载用户设备列表
          await loadUserDevices()
        } else {
          throw new Error(userResponse.data?.message || '获取用户详情失败')
        }
      } catch (error) {
        ElMessage.error('加载用户详情失败: ' + (error.response?.data?.message || error.message))
        }
    }

    // 加载用户设备列表
    const loadUserDevices = async () => {
      // 使用订阅ID获取设备列表，而不是用户ID
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
          
          // 更新在线设备数量（根据实际加载的设备数量）
          if (selectedUser.value) {
            selectedUser.value.online_devices = userDevices.value.length
            selectedUser.value.current_devices = userDevices.value.length
          }
        } else {
          userDevices.value = []
          ElMessage.warning('获取设备列表失败: 响应格式不正确')
          // 如果获取失败，也更新在线设备数量为0
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
        // 如果出错，也更新在线设备数量为0
        if (selectedUser.value) {
          selectedUser.value.online_devices = 0
          selectedUser.value.current_devices = 0
        }
      } finally {
        loadingDevices.value = false
      }
    }

    // 删除设备
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
        // 使用删除设备的API端点
        const response = await adminAPI.removeDevice(device.id)
        
        if (response.data && response.data.success) {
          ElMessage.success('设备删除成功')
          // 重新加载设备列表
          await loadUserDevices()
          // 重新加载订阅列表以更新设备计数
          await loadSubscriptions()
          // 更新当前选中用户的设备计数
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

    // 获取设备类型标签样式
    // 获取重置类型标签
    const getResetTypeTag = (type) => {
      const typeMap = {
        'manual': 'primary',
        'automatic': 'info',
        'admin': 'warning',
        'system': 'success'
      }
      return typeMap[type] || 'info'
    }

    // 获取重置类型文本
    const getResetTypeText = (type) => {
      const typeMap = {
        'manual': '手动重置',
        'automatic': '自动重置',
        'admin': '管理员重置',
        'system': '系统重置'
      }
      return typeMap[type] || type || '未知'
    }

    // 获取操作人标签
    const getResetByTag = (by) => {
      const byMap = {
        'user': 'primary',
        'admin': 'warning',
        'system': 'success'
      }
      return byMap[by] || 'info'
    }

    // 获取操作人文本
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

    // 获取设备类型文本
    const getDeviceTypeText = (type) => {
      const typeMap = {
        'mobile': '手机',
        'desktop': '电脑',
        'tablet': '平板',
        'server': '服务器'
      }
      return typeMap[type] || type || '未知'
    }

    // 复制到剪贴板
    const copyToClipboard = async (text) => {
      if (!text) {
        ElMessage.warning('没有可复制的内容')
        return
      }
      
      try {
        await navigator.clipboard.writeText(text)
        ElMessage.success('订阅链接已复制到剪贴板')
      } catch (error) {
        // 降级方案：使用传统的复制方法
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

    // 进入用户后台
    const goToUserBackend = async (subscription) => {
      try {
        const userId = subscription.user?.id || subscription.user_id
        if (!userId || userId === 0) {
          ElMessage.warning('无法进入：用户信息不存在或已被删除')
          return
        }

        const userName = subscription.user?.username || subscription.user?.email || subscription.username || subscription.email || '未知用户'
        await ElMessageBox.confirm(
          `确定要以用户 ${userName} 的身份登录吗？将在新标签页中打开用户后台。`,
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

        if (!response.data.data.token || !response.data.data.user) {
          ElMessage.error('登录失败：服务器返回数据不完整')
          return
        }
        
        // 保存管理员信息到 localStorage（用于返回管理员后台）
        const adminToken = secureStorage.get('admin_token')
        const adminUser = secureStorage.get('admin_user')
        
        const userToken = response.data.data.token
        const userData = response.data.data.user

        // 在 sessionKey 中也包含管理员信息，以便在新标签页中恢复
        const sessionKey = `user_login_${Date.now()}`
        const sessionData = {
          token: userToken,
          user: userData,
          timestamp: Date.now()
        }
        
        // 如果有管理员信息，也保存到 sessionData 中
        if (adminToken && adminUser) {
          sessionData.adminToken = adminToken
          sessionData.adminUser = typeof adminUser === 'string' ? adminUser : JSON.stringify(adminUser)
        }
        
        sessionStorage.setItem(sessionKey, JSON.stringify(sessionData))

        const dashboardUrl = window.location.origin + '/dashboard'
        const finalUrl = `${dashboardUrl}?sessionKey=${sessionKey}`

        const newWindow = window.open(finalUrl, '_blank')

        if (newWindow) {
          ElMessage.success('正在新标签页中打开用户后台...')
        } else {
          ElMessage.error('无法打开新标签页，请检查浏览器弹窗设置')
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

    // 重置订阅
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
        
        // 如果当前正在查看该用户的详情，重新加载设备列表
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

    // 发送订阅邮件（添加防重复点击机制）
    const sendingEmailMap = new Map()
    const sendSubscriptionEmail = async (subscription) => {
      const userId = subscription.user?.id || subscription.user_id
      if (!userId || userId === 0) {
        ElMessage.warning('无法发送：用户信息不存在或已被删除')
        return
      }

      // 防止重复点击
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
        // 3秒后移除标记，允许再次发送
        setTimeout(() => {
          sendingEmailMap.delete(userId)
        }, 3000)
      }
    }

    // 切换订阅状态
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

    // 删除用户
    const deleteUser = async (subscription) => {
      // 检查用户ID是否有效
      const userId = subscription.user?.id || subscription.user_id
      if (!userId || userId === 0) {
        ElMessage.warning('无法删除：用户信息不存在或已被删除')
        return
      }

      // 检查用户是否已删除
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

    // 清理用户设备
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
        
        // 重新加载订阅列表以更新设备计数
        await loadSubscriptions()
        
        // 如果当前正在查看该用户的详情，重新加载设备列表
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

    // 清理所有设备
    const clearAllDevices = async () => {
      try {
        await ElMessageBox.confirm('确定要清理所有用户的设备吗？这将清除所有设备记录。', '确认清理', {
          type: 'warning'
        })
        
        // 获取所有订阅的ID
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

    // 导出订阅
    const exportSubscriptions = async () => {
      try {
        const response = await adminAPI.exportSubscriptions()
        // 检查响应是否为Blob
        let blob = null
        if (response.data instanceof Blob) {
          blob = response.data
        } else if (response.data && typeof response.data === 'object' && response.data.data) {
          // axios可能包装了Blob
          blob = response.data.data
        }
        
        if (blob instanceof Blob) {
          // 从响应头获取文件名
          const contentDisposition = response.headers['content-disposition'] || response.headers['Content-Disposition']
          // 使用北京时间生成文件名
          const beijingDate = dayjs().tz('Asia/Shanghai')
          let filename = `subscriptions_export_${beijingDate.format('YYYYMMDD')}.csv`
          
          if (contentDisposition) {
            // 优先解析 RFC 5987 格式: filename*=UTF-8''filename
            let filenameMatch = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i)
            if (filenameMatch && filenameMatch.length > 1) {
              filename = decodeURIComponent(filenameMatch[1])
            } else {
              // 解析标准格式: filename="filename"
              filenameMatch = contentDisposition.match(/filename=['"]?([^'";]+)['"]?/i)
              if (filenameMatch && filenameMatch.length > 1) {
                filename = decodeURIComponent(filenameMatch[1])
              }
            }
          }
          
          // 创建下载链接
          const url = window.URL.createObjectURL(blob)
          const link = document.createElement('a')
          link.style.display = 'none'
          link.href = url
          link.download = filename
          
          // 添加到DOM并触发点击
          document.body.appendChild(link)
          // 使用requestAnimationFrame确保DOM已更新
          requestAnimationFrame(() => {
            link.click()
            // 延迟清理
            setTimeout(() => {
              document.body.removeChild(link)
              window.URL.revokeObjectURL(url)
              }, 1000)
          })
          
          ElMessage.success('订阅数据导出成功，文件下载已开始')
        } else {
          // 如果不是Blob，可能是JSON错误响应
          ElMessage.error('导出失败：响应格式不正确，收到的是：' + (typeof response.data))
        }
      } catch (error) {
        // 处理错误响应
        if (error.response) {
          // 如果是Blob类型的错误响应，尝试读取错误信息
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
            // 如果是JSON错误响应
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

    // 显示通用订阅统计
    const showAppleStats = () => {
      ElMessage.info('通用订阅统计功能待实现')
    }

    // 显示在线统计
    const showOnlineStats = () => {
      ElMessage.info('在线设备统计功能待实现')
    }

    // 移除设备

    const truncateText = (text, maxLength) => {
      if (!text) return ''
      return text.length > maxLength ? text.substring(0, maxLength) + '...' : text
    }

    const truncateUserAgent = (userAgent) => {
      if (!userAgent) return '未知'
      return userAgent.length > 50 ? userAgent.substring(0, 50) + '...' : userAgent
    }

    const formatTime = (time) => {
      // 使用统一的北京时间格式化函数
      return formatTimeUtil(time) || '未知'
    }

    // 截断URL用于显示
    const truncateUrl = (url) => {
      if (!url) return ''
      // 移动端显示完整URL，桌面端可以截断
      const isMobile = window.innerWidth <= 768
      if (isMobile) {
        // 移动端显示完整URL，不截断
        return url
      }
      // 桌面端：如果URL长度超过60个字符，显示前40个和后15个字符
      if (url.length > 60) {
        return url.substring(0, 40) + '...' + url.substring(url.length - 15)
      }
      return url
    }

    // 获取订阅状态类型
    const getSubscriptionStatusType = (status) => {
      const statusMap = {
        'active': 'success',
        'inactive': 'info',
        'expired': 'danger',
        'paused': 'warning'
      }
      return statusMap[status] || 'info'
    }

    // 获取订阅状态文本
    const getSubscriptionStatusText = (status) => {
      const statusMap = {
        'active': '活跃',
        'inactive': '未激活',
        'expired': '已过期',
        'paused': '已暂停'
      }
      return statusMap[status] || '未知'
    }

    // 格式化日期 - 使用统一的北京时间格式化函数
    const formatDate = (date) => {
      return formatDateUtil(date)
    }

    // 处理选择变化
    const handleSelectionChange = (selection) => {
      selectedSubscriptions.value = selection
    }

    // 处理页面大小变化
    const handleSizeChange = (val) => {
      pageSize.value = val
      loadSubscriptions()
    }

    // 处理当前页变化
    const handleCurrentChange = (val) => {
      currentPage.value = val
      loadSubscriptions()
    }

    // 排序相关方法
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

    const handleDeviceLimitSort = ({ column, prop, order }) => {
      if (order === 'descending') {
        currentSort.value = 'device_limit_desc'
      } else if (order === 'ascending') {
        currentSort.value = 'device_limit_asc'
      } else {
        currentSort.value = 'add_time_desc' // 默认排序
      }
      loadSubscriptions()
    }

    // 列设置相关方法
    const selectAllColumns = () => {
      visibleColumns.value = [
        'qq', 'expire_time', 'qr_code', 'universal_url', 'clash_url', 
        'created_at', 'apple_count', 'clash_count', 'online_devices', 
        'device_limit', 'actions'
      ]
    }

    const clearAllColumns = () => {
      // 至少保留一列，建议保留QQ号码和操作列
      visibleColumns.value = ['qq', 'actions']
    }

    const resetToDefault = () => {
      visibleColumns.value = [
        'qq', 'expire_time', 'qr_code', 'universal_url', 'clash_url', 
        'created_at', 'apple_count', 'clash_count', 'online_devices', 
        'device_limit', 'actions'
      ]
    }

    // 响应式移动端检测
    const isMobile = computed(() => {
      if (typeof window === 'undefined') return false
      return window.innerWidth <= 768
    })

    // 监听窗口大小变化
    const handleResize = () => {
      // 触发响应式更新
    }

    // 组件挂载时加载数据
    onMounted(() => {
      loadSubscriptions()
      window.addEventListener('resize', handleResize)
    })

    // 组件卸载时移除监听
    onUnmounted(() => {
      window.removeEventListener('resize', handleResize)
    })

    return {
      isMobile,
      loading,
      subscriptions,
      selectedSubscriptions,
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
      handleSizeChange,
      handleCurrentChange,
      selectAllColumns,
      clearAllColumns,
      resetToDefault,
      sortByApple,
      sortByOnline,
      sortByCreatedTime,
      handleDeviceLimitSort,
      truncateUserAgent,
      formatTime,
      handleActionCommand,
      truncateUrl
    }
  }
}
</script>

<style scoped lang="scss">
@use '@/styles/list-common.scss';

.admin-subscriptions {
  // 使用 list-container 的样式，确保宽度和其他列表一致
  // 宽度由 list-common.scss 统一管理，这里不需要额外设置
  // 继承父级样式
  @extend .list-container;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
  
  &.desktop-only {
    @media (max-width: 768px) {
      display: none !important;
    }
  }
}

.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}

// 移动端智能操作栏
// mobile-action-bar 和 mobile-search-section 样式已统一在 list-common.scss 中定义
// 这里不再重复定义，使用统一样式
.mobile-action-bar {
  .mobile-quick-actions {
    display: flex;
    flex-direction: column;
    gap: 10px;
    
    .quick-sort-buttons {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
      
      .el-button {
        flex: 1;
        min-width: 0;
        font-size: 0.85rem;
        padding: 8px 12px;
        min-height: 36px;
        
        :deep(.el-icon) {
          margin-right: 4px;
          font-size: 14px;
        }
      }
    }
    
    .action-buttons-group {
      display: flex;
      justify-content: flex-end;
      
      .el-button {
        font-size: 0.85rem;
        padding: 8px 16px;
        min-height: 36px;
        
        :deep(.el-icon) {
          margin-right: 4px;
          font-size: 14px;
        }
      }
    }
  }
  
  .mobile-sort-info {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 12px;
    padding: 8px 12px;
    background: white;
    border-radius: 6px;
    font-size: 0.85rem;
    
    .sort-label {
      color: #666;
      font-weight: 500;
    }
    
    .sort-value {
      color: #303133;
      flex: 1;
    }
  }
}

// 搜索表单样式已由 list-common.scss 统一管理
// 确保宽度和其他列表一致
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
  gap: 8px;
}

.qq-number {
  font-weight: 500;
  color: #303133;
}

.detail-btn {
  width: 100%;
}

.expire-time-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
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

.pagination {
  margin-top: 20px;
  text-align: right;
}

.user-detail-content {
  max-height: 70vh;
  overflow-y: auto;
  
  @media (max-width: 768px) {
    max-height: calc(100vh - 120px);
    padding: 0;
  }
}

/* 用户详情对话框样式优化 */
.user-detail-dialog {
  :deep(.el-dialog) {
    @media (max-width: 768px) {
      margin: 0 !important;
      height: 100vh;
      max-height: 100vh;
      border-radius: 0;
      
      .el-dialog__body {
        padding: 12px;
        max-height: calc(100vh - 120px);
        overflow-y: auto;
      }
      
      .el-dialog__header {
        padding: 16px;
        position: sticky;
        top: 0;
        background: white;
        z-index: 10;
        border-bottom: 1px solid #ebeef5;
      }
    }
  }
}

/* 设备管理样式 */
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
    
    label {
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

.url-item label {
  font-weight: 500;
  color: #606266;
}

/* 设备表格移动端优化 */
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

/* 响应式设计 */
// 响应式样式已由 list-common.scss 统一管理
// 移动端特定样式
@media (max-width: 768px) {
  .qr-code img {
    width: 40px;
    height: 40px;
  }
  
  // 移动端卡片中的日期选择器和数字输入框样式
  .mobile-card-list {
    .mobile-card {
      // 移动端卡片操作按钮样式 - 三个一排，分两排
      .card-actions {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 8px;
        width: 100%;
        margin-top: 12px;
        
        .el-button {
          width: 100% !important;
          height: 40px !important;
          font-size: 14px !important;
          font-weight: 500 !important;
          margin: 0 !important;
          padding: 0 8px !important;
          white-space: nowrap !important;
          overflow: hidden !important;
          text-overflow: ellipsis !important;
          
          :deep(.el-icon) {
            margin-right: 4px;
            font-size: 14px;
          }
        }
      }
      
      .card-row {
        .value {
          // 日期选择器样式
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
          
          // 数字输入框样式
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
          
          // 确保日期选择器和数字输入框在同一行显示良好
          display: flex;
          align-items: center;
          justify-content: flex-end;
          gap: 8px;
          flex-wrap: wrap;
          
          // 到期时间区域样式
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
  
  // 移动端订阅日期选择器弹出层样式
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

/* 列设置对话框样式 */
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

// 移动端订阅二维码区域样式
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
        
        // 确保按钮内容居中
        :deep(.el-icon) {
          margin-right: 6px;
          flex-shrink: 0;
        }
        
        // 确保按钮文字和图标都居中
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

// 移动端订阅地址区域样式
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
  
  // 猫咪订阅特殊样式
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
