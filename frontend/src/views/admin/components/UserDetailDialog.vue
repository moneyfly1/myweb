<template>
  <AppDrawer
    :model-value="visible"
    @update:model-value="$emit('update:visible', $event)"
    :title="`用户详情 - ${user?.user_info?.username || user?.username || user?.user_info?.email || user?.email || ''}`"
    size="780px"
    mobile-size="100%"
    class="user-detail-drawer"
    close-on-click-modal
  >
    <div v-if="user" class="drawer-content">
      <!-- 用户基本信息 (始终可见) -->
      <el-descriptions :column="isMobile ? 1 : 2" border size="small">
        <el-descriptions-item label="用户ID">{{ user.user_info?.id || user.id }}</el-descriptions-item>
        <el-descriptions-item label="用户名">{{ user.user_info?.username || user.username }}</el-descriptions-item>
        <el-descriptions-item label="邮箱">{{ user.user_info?.email || user.email }}</el-descriptions-item>
        <el-descriptions-item label="账户余额">
          <span class="balance-highlight">¥{{ ((user.user_info?.balance || user.balance || 0)).toFixed(2) }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="getStatusType(user.user_info?.is_active !== false ? 'active' : 'inactive')" size="small">
            {{ getStatusText(user.user_info?.is_active !== false ? 'active' : 'inactive') }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="用户等级">
          <el-tag v-if="user.user_info?.is_admin" type="danger" size="small">管理员</el-tag>
          <el-tag v-else-if="user.user_info?.is_verified" type="success" size="small">已验证</el-tag>
          <el-tag v-else type="info" size="small">普通用户</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="注册时间">{{ formatDate(user.user_info?.created_at || user.created_at) }}</el-descriptions-item>
        <el-descriptions-item label="最后登录">{{ formatDate(user.user_info?.last_login || user.last_login) || '从未登录' }}</el-descriptions-item>
      </el-descriptions>

      <!-- 订阅信息分隔线 -->
      <el-divider content-position="left">订阅信息</el-divider>

      <!-- 订阅信息 (始终可见) -->
      <div v-if="user.subscriptions && user.subscriptions.length > 0">
        <div v-for="(sub, index) in user.subscriptions" :key="sub.id" class="subscription-section">
          <el-descriptions :column="isMobile ? 1 : 2" border size="small">
            <el-descriptions-item label="套餐名称">{{ sub.package_name || '未知套餐' }}</el-descriptions-item>
            <el-descriptions-item label="订阅状态">
              <el-tag :type="sub.is_active ? 'success' : 'danger'" size="small">
                {{ sub.is_active ? '活跃' : '未激活' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="设备数量">
              {{ sub.current_devices || 0 }} / {{ sub.device_limit || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="到期时间">
              <span :class="{ 'expired-text': sub.is_expired }">
                {{ sub.expire_time || '未设置' }}
                <span v-if="sub.days_until_expire !== undefined && !sub.is_expired">
                  (剩余 {{ sub.days_until_expire }} 天)
                </span>
                <span v-if="sub.is_expired" class="expired-badge">已过期</span>
              </span>
            </el-descriptions-item>
          </el-descriptions>

          <!-- 订阅链接 -->
          <div class="url-section">
            <div class="protocol-exclude-panel" v-if="protocolOptions.length">
              <div class="protocol-exclude-header">
                <div>
                  <div class="exclude-title">协议排除</div>
                  <div class="exclude-subtitle">
                    {{ getExcludedProtocols(sub).length ? `已排除 ${getExcludedProtocols(sub).length} 种协议` : '默认遵循后台系统协议过滤' }}
                  </div>
                </div>
                <el-button
                  text
                  type="primary"
                  size="small"
                  :disabled="!getExcludedProtocols(sub).length"
                  @click="clearExcludedProtocols(sub)"
                >
                  清空
                </el-button>
              </div>
              <el-checkbox-group
                :model-value="getExcludedProtocols(sub)"
                class="protocol-checkboxes"
                @change="value => setExcludedProtocols(sub, value)"
              >
                <el-checkbox-button
                  v-for="protocol in protocolOptions"
                  :key="protocol.value"
                  :label="protocol.value"
                >
                  {{ protocol.label }}
                </el-checkbox-button>
              </el-checkbox-group>
            </div>
            <div class="url-item" v-if="sub.universal_url || sub.subscription_url">
              <div class="url-header">
                <span class="url-label">通用订阅 (V2Ray/Shadowrocket):</span>
                <el-button
                  size="small"
                  :icon="CopyDocument"
                  @click="copyToClipboard(getSubscriptionUrlWithExclude(sub, sub.universal_url || sub.subscription_url))"
                  :disabled="!sub.universal_url && !sub.subscription_url"
                >
                  复制
                </el-button>
              </div>
              <button
                type="button"
                class="url-code url-copy"
                @click="copyToClipboard(getSubscriptionUrlWithExclude(sub, sub.universal_url || sub.subscription_url))"
                :disabled="!sub.universal_url && !sub.subscription_url"
                :title="`点击复制: ${getSubscriptionUrlWithExclude(sub, sub.universal_url || sub.subscription_url) || ''}`"
              >
                {{ getSubscriptionUrlWithExclude(sub, sub.universal_url || sub.subscription_url) || '无' }}
              </button>
            </div>
            <div class="url-item" v-if="sub.clash_url">
              <div class="url-header">
                <span class="url-label">Clash / Clash Meta:</span>
                <el-button
                  size="small"
                  :icon="CopyDocument"
                  @click="copyToClipboard(getSubscriptionUrlWithExclude(sub, getTypedSubscriptionUrl(sub, 'clash')))"
                  :disabled="!getTypedSubscriptionUrl(sub, 'clash')"
                >
                  复制
                </el-button>
              </div>
              <button
                type="button"
                class="url-code url-copy"
                @click="copyToClipboard(getSubscriptionUrlWithExclude(sub, getTypedSubscriptionUrl(sub, 'clash')))"
                :disabled="!getTypedSubscriptionUrl(sub, 'clash')"
                :title="`点击复制: ${getSubscriptionUrlWithExclude(sub, getTypedSubscriptionUrl(sub, 'clash')) || ''}`"
              >
                {{ getSubscriptionUrlWithExclude(sub, getTypedSubscriptionUrl(sub, 'clash')) || '无' }}
              </button>
            </div>
            <div class="url-item" v-if="!sub.universal_url && !sub.subscription_url && !sub.clash_url">
              <div class="url-header">
                <span class="url-label">订阅地址:</span>
              </div>
              <code class="url-code">无</code>
            </div>
            <el-collapse v-if="hasMoreSubscriptionUrls(sub)" class="more-urls-collapse">
              <el-collapse-item title="更多订阅地址" :name="`more-${sub.id || index}`">
                <div
                  v-for="client in getMoreSubscriptionUrls(sub)"
                  :key="client.type"
                  class="url-item"
                >
                  <div class="url-header">
                    <span class="url-label">{{ client.label }}:</span>
                    <el-button size="small" :icon="CopyDocument" @click="copyToClipboard(client.url)">复制</el-button>
                  </div>
                  <button
                    type="button"
                    class="url-code url-copy"
                    @click="copyToClipboard(client.url)"
                    :title="`点击复制: ${client.url}`"
                  >
                    {{ client.url }}
                  </button>
                </div>
              </el-collapse-item>
            </el-collapse>
          </div>

          <el-divider v-if="index < user.subscriptions.length - 1" />
        </div>
      </div>
      <EmptyState
        v-else
        title="暂无订阅信息"
        description="该用户当前没有可展示的订阅。"
        :icon-size="48"
        class="detail-empty-state"
      />

      <!-- 记录信息分隔线 -->
      <el-divider content-position="left">记录信息</el-divider>

      <!-- 底部记录 Tabs -->
      <el-tabs v-model="activeTab" class="records-tabs">
        <!-- 订单记录 Tab -->
        <el-tab-pane label="订单记录" name="orders">
          <el-table
            v-if="orderRecords && orderRecords.length > 0"
            :data="orderRecords"
            size="small"
            max-height="240"
            class="data-table"
          >
            <el-table-column prop="order_no" label="订单号" min-width="180" show-overflow-tooltip />
            <el-table-column prop="amount" label="金额" width="100">
              <template #default="scope">
                <span class="amount-text">¥{{ scope.row.amount }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="100">
              <template #default="scope">
                <el-tag :type="getStatusType(scope.row.status)" size="small">
                  {{ getStatusText(scope.row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="160">
              <template #default="scope">
                {{ formatDateTime(scope.row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
          <EmptyState
            v-else
            title="暂无订单记录"
            description="该用户当前没有订单记录。"
            :icon-size="48"
            class="detail-empty-state"
          />
        </el-tab-pane>

        <!-- 设备记录 Tab -->
        <el-tab-pane label="设备记录" name="devices">
          <div class="devices-section">
            <div class="devices-actions">
              <el-button
                size="small"
                :icon="RefreshRight"
                @click="loadDevices"
                :loading="loadingDevices"
              >
                刷新设备
              </el-button>
              <span v-if="devices.length > 0" class="device-count-tip">
                共 {{ devices.length }} 台设备在线
              </span>
            </div>
            <el-table
              v-if="devices.length > 0"
              :data="devices"
              size="small"
              max-height="300"
              class="data-table"
              v-loading="loadingDevices"
            >
              <el-table-column prop="device_name" label="设备名称" min-width="120" show-overflow-tooltip />
              <el-table-column prop="device_type" label="类型" width="80">
                <template #default="scope">
                  <el-tag :type="getDeviceTypeColor(scope.row.device_type)" size="small">
                    {{ getDeviceTypeName(scope.row.device_type) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="ip_address" label="IP地址" width="130" show-overflow-tooltip />
              <el-table-column prop="location" label="归属地" min-width="100" show-overflow-tooltip>
                <template #default="scope">
                  {{ displayLocation(scope.row.location) }}
                </template>
              </el-table-column>
              <el-table-column prop="last_access" label="最后访问" width="160">
                <template #default="scope">
                  {{ formatDateTime(scope.row.last_access || scope.row.last_seen) }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="80" fixed="right">
                <template #default="scope">
                  <el-button
                    type="danger"
                    size="small"
                    :icon="Delete"
                    :loading="deletingDeviceId === scope.row.id"
                    @click="deleteDevice(scope.row)"
                    plain
                  >
                    删除
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
            <EmptyState
              v-else-if="!loadingDevices"
              title="暂无在线设备"
              description="该用户当前没有在线设备。"
              :icon-size="48"
              class="detail-empty-state"
            />
            <div v-if="uaRecords && uaRecords.length > 0" class="ua-records-section">
              <el-divider content-position="left" class="compact-divider">UA访问记录</el-divider>
              <el-table
                :data="uaRecords"
                size="small"
                max-height="200"
                class="data-table"
              >
                <el-table-column prop="device_name" label="设备名称" min-width="120" show-overflow-tooltip />
                <el-table-column prop="device_type" label="类型" width="80">
                  <template #default="scope">
                    <el-tag :type="getDeviceTypeColor(scope.row.device_type)" size="small">
                      {{ getDeviceTypeName(scope.row.device_type) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="ip_address" label="IP地址" width="130" />
                <el-table-column prop="location" label="位置" min-width="100" show-overflow-tooltip>
                  <template #default="scope">
                    {{ displayLocation(scope.row.location) }}
                  </template>
                </el-table-column>
                <el-table-column prop="last_access" label="最后访问" width="160">
                  <template #default="scope">
                    {{ formatDateTime(scope.row.last_access) }}
                  </template>
                </el-table-column>
                <el-table-column prop="access_count" label="访问次数" width="90" />
              </el-table>
            </div>
          </div>
        </el-tab-pane>

        <!-- 登录历史 Tab -->
        <el-tab-pane label="登录历史" name="login">
          <el-table
            v-if="loginHistory && loginHistory.length > 0"
            :data="loginHistory"
            size="small"
            max-height="240"
            class="data-table"
          >
            <el-table-column prop="login_time" label="登录时间" width="160">
              <template #default="scope">
                {{ formatDateTime(scope.row.login_time) }}
              </template>
            </el-table-column>
            <el-table-column prop="ip_address" label="IP地址" width="130" />
            <el-table-column prop="location" label="位置" min-width="120" show-overflow-tooltip>
              <template #default="scope">
                {{ displayLocation(scope.row.location) }}
              </template>
            </el-table-column>
            <el-table-column prop="user_agent" label="User Agent" min-width="200" show-overflow-tooltip />
            <el-table-column prop="login_status" label="登录状态" width="100">
              <template #default="scope">
                <el-tag :type="scope.row.login_status === 'success' ? 'success' : 'danger'" size="small">
                  {{ scope.row.login_status === 'success' ? '成功' : '失败' }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
          <EmptyState
            v-else
            title="暂无登录历史"
            description="该用户当前没有登录历史。"
            :icon-size="48"
            class="detail-empty-state"
          />
        </el-tab-pane>

        <!-- 重置记录 Tab -->
        <el-tab-pane label="重置记录" name="resets">
          <div v-if="subscriptionResets && subscriptionResets.length > 0" class="table-responsive">
            <el-table
              :data="subscriptionResets"
              size="small"
              max-height="240"
              class="data-table"
            >
              <el-table-column prop="reset_by" label="重置人" width="100" />
              <el-table-column prop="reset_type" label="重置类型" width="110">
                <template #default="scope">
                  {{ getResetTypeText(scope.row.reset_type) }}
                </template>
              </el-table-column>
              <el-table-column prop="reason" label="原因" min-width="120" show-overflow-tooltip />
              <el-table-column label="旧订阅URL" min-width="150" show-overflow-tooltip>
                <template #default="scope">
                  <code class="url-code-small">{{ scope.row.old_subscription_url }}</code>
                </template>
              </el-table-column>
              <el-table-column label="新订阅URL" min-width="150" show-overflow-tooltip>
                <template #default="scope">
                  <code class="url-code-small">{{ scope.row.new_subscription_url }}</code>
                </template>
              </el-table-column>
              <el-table-column label="设备数变化" width="110">
                <template #default="scope">
                  {{ scope.row.device_count_before }} → {{ scope.row.device_count_after }}
                </template>
              </el-table-column>
              <el-table-column prop="created_at" label="重置时间" width="160">
                <template #default="scope">
                  {{ formatDateTime(scope.row.created_at) }}
                </template>
              </el-table-column>
            </el-table>
          </div>
          <EmptyState
            v-else
            title="暂无重置记录"
            description="该用户当前没有订阅重置记录。"
            :icon-size="48"
            class="detail-empty-state"
          />
        </el-tab-pane>

        <!-- 充值记录 Tab -->
        <el-tab-pane label="充值记录" name="recharge">
          <el-table
            v-if="rechargeRecords && rechargeRecords.length > 0"
            :data="rechargeRecords"
            size="small"
            max-height="240"
            class="data-table"
          >
            <el-table-column prop="order_no" label="订单号" min-width="180" show-overflow-tooltip />
            <el-table-column prop="amount" label="金额" width="100">
              <template #default="scope">
                <span class="amount-text positive">+¥{{ scope.row.amount }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="payment_method" label="支付方式" width="100">
              <template #default="scope">
                {{ getPaymentMethodText(scope.row.payment_method) }}
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="100">
              <template #default="scope">
                <el-tag :type="getStatusType(scope.row.status)" size="small">
                  {{ getStatusText(scope.row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="160">
              <template #default="scope">
                {{ formatDateTime(scope.row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
          <EmptyState
            v-else
            title="暂无充值记录"
            description="该用户当前没有充值记录。"
            :icon-size="48"
            class="detail-empty-state"
          />
        </el-tab-pane>

        <!-- 签到日志 Tab -->
        <el-tab-pane label="签到日志" name="checkins">
          <div class="checkin-actions">
            <el-button
              size="small"
              :icon="RefreshRight"
              @click="loadCheckinLogs"
              :loading="loadingCheckins"
            >
              刷新
            </el-button>
            <el-button
              type="success"
              size="small"
              @click="exportCheckinLogs"
              :loading="exportingCheckins"
            >
              导出签到日志
            </el-button>
          </div>
          <el-table
            v-if="checkinLogs && checkinLogs.length > 0"
            :data="checkinLogs"
            size="small"
            max-height="240"
            class="data-table"
            v-loading="loadingCheckins"
          >
            <el-table-column prop="created_at" label="签到时间" width="180">
              <template #default="scope">
                {{ formatDateTime(scope.row.created_at) }}
              </template>
            </el-table-column>
            <el-table-column prop="amount" label="奖励金额" width="140">
              <template #default="scope">
                <span class="amount-text positive">+¥{{ Number(scope.row.amount || 0).toFixed(2) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="备注" min-width="180">
              <template #default>
                每日签到奖励
              </template>
            </el-table-column>
          </el-table>
          <EmptyState
            v-else-if="!loadingCheckins"
            title="暂无签到日志"
            description="该用户当前没有签到日志。"
            :icon-size="48"
            class="detail-empty-state"
          />
          <div class="checkin-pagination">
            <PaginationBar
              v-model:current-page="checkinPagination.page"
              v-model:page-size="checkinPagination.size"
              :total="checkinPagination.total"
              @size-change="handleCheckinSizeChange"
              @current-change="handleCheckinPageChange"
            />
          </div>
        </el-tab-pane>

        <!-- 专线节点 Tab -->
        <el-tab-pane label="专线节点" name="custom-nodes">
          <div class="custom-nodes-section">
            <div class="line-mode-panel">
              <div class="line-mode-heading">
                <div>
                  <div class="line-mode-title">线路模式</div>
                  <div class="line-mode-meta">切换模式不会删除已分配的专线节点。</div>
                </div>
                <el-tag :type="getLineModeTagType(lineModeForm)" effect="plain" size="small">
                  {{ getLineModeText(lineModeForm) }}
                </el-tag>
              </div>
              <el-radio-group
                v-model="lineModeForm"
                size="small"
                class="line-mode-control"
                :disabled="savingLineMode"
                @change="updateLineMode"
              >
                <el-radio-button label="normal">普通线路</el-radio-button>
                <el-radio-button label="both" :disabled="!hasAssignedCustomNodes">专线 + 普通线路</el-radio-button>
                <el-radio-button label="special_only" :disabled="!hasAssignedCustomNodes">仅专线</el-radio-button>
              </el-radio-group>
            </div>
            <div class="custom-nodes-actions">
              <el-button
                type="primary"
                size="small"
                :icon="Plus"
                @click="openAssignDialog"
              >
                分配专线节点
              </el-button>
              <el-button
                type="danger"
                size="small"
                :icon="Delete"
                :disabled="selectedCustomNodes.length === 0"
                :loading="batchUnassigning"
                @click="batchUnassignSelectedNodes"
              >
                批量取消分配{{ selectedCustomNodes.length > 0 ? ` (${selectedCustomNodes.length})` : '' }}
              </el-button>
              <el-button
                type="danger"
                size="small"
                plain
                :disabled="!(customNodes && customNodes.length > 0)"
                :loading="clearingNodes"
                @click="clearAllCustomNodes"
              >
                清空专线节点
              </el-button>
              <el-button
                size="small"
                :icon="RefreshRight"
                @click="loadUserCustomNodes"
                :loading="loadingNodes"
              >
                刷新
              </el-button>
            </div>

            <el-table
              v-if="customNodes && customNodes.length > 0"
              :data="customNodes"
              size="small"
              max-height="240"
              class="data-table"
              @selection-change="handleCustomNodeSelectionChange"
            >
              <el-table-column type="selection" width="40" />
              <el-table-column prop="node_name" label="节点名称" min-width="150" />
              <el-table-column prop="node_address" label="节点地址" min-width="200" show-overflow-tooltip />
              <el-table-column label="专线到期" width="160">
                <template #default="scope">
                  {{ formatDateTime(scope.row.special_node_expires_at) || '跟随订阅' }}
                </template>
              </el-table-column>
              <el-table-column prop="assigned_at" label="分配时间" width="160">
                <template #default="scope">
                  {{ formatDateTime(scope.row.assigned_at) }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="100" fixed="right">
                <template #default="scope">
                  <el-button
                    type="danger"
                    size="small"
                    link
                    @click="unassignNode(scope.row.node_id)"
                  >
                    取消分配
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
            <EmptyState
              v-else
              title="暂无专线节点"
              description="该用户当前没有分配专线节点。"
              :icon-size="48"
              class="detail-empty-state"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- 分配专线节点对话框 -->
    <AppDialog
      v-model="showAssignDialog"
      title="分配专线节点"
      width="720px"
      mobile-width="94%"
      :loading="assigning"
      class="assign-node-dialog"
    >
      <div class="assign-dialog-content">
        <div class="assign-summary">
          <div class="assign-summary-item">
            <span>已选用户</span>
            <strong>{{ selectedUserIds.length }}</strong>
          </div>
          <div class="assign-summary-item">
            <span>已选节点</span>
            <strong>{{ selectedNodeIds.length }}</strong>
          </div>
          <div class="assign-summary-item">
            <span>到期时间</span>
            <strong>{{ assignExpiresAt || '跟随订阅' }}</strong>
          </div>
        </div>

        <div class="assign-section-card">
          <div class="assign-section-header">
            <div>
              <div class="section-title">选择用户</div>
              <div class="section-desc">默认包含当前详情用户，也可以搜索追加其他用户。</div>
            </div>
          </div>
          <div class="search-input-group">
            <el-input
              v-model="userSearchKeyword"
              placeholder="搜索用户名、邮箱或备注"
              clearable
              @keyup.enter="handleUserSearch"
              @clear="handleUserSearchClear"
            >
              <template #append>
                <el-button @click="handleUserSearch" :loading="searchingUsers">
                  <el-icon><Search /></el-icon>
                  <span class="search-button-text">搜索</span>
                </el-button>
              </template>
            </el-input>
          </div>
          <div v-if="hasUserSearched && userSearchKeyword && searchedUsers.length > 0" class="search-result-tip">
            找到 {{ searchedUsers.length }} 个用户
          </div>
          <div v-else-if="hasUserSearched && userSearchKeyword && !searchingUsers" class="search-result-tip empty">
            未找到匹配的用户
          </div>
          <el-select
            v-model="selectedUserIds"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            placeholder="请选择用户"
            class="form-control"
            no-data-text="请先搜索用户"
            @change="handleSelectedUsersChange"
          >
            <el-option
              v-for="userItem in assignUserOptions"
              :key="userItem.id"
              :label="formatUserOption(userItem)"
              :value="userItem.id"
            />
          </el-select>
        </div>

        <div class="assign-section-card">
          <div class="assign-section-header">
            <div>
              <div class="section-title">选择专线节点</div>
              <div class="section-desc">支持按节点名称、显示名称或地址搜索，搜索后可多选。</div>
            </div>
          </div>
          <div class="search-input-group">
            <el-input
              v-model="nodeSearchKeyword"
              placeholder="搜索节点名称、显示名称或地址"
              clearable
              @keyup.enter="handleNodeSearch"
              @clear="handleNodeSearchClear"
            >
              <template #append>
                <el-button @click="handleNodeSearch" :loading="searchingNodes">
                  <el-icon><Search /></el-icon>
                  <span class="search-button-text">搜索</span>
                </el-button>
              </template>
            </el-input>
          </div>
          <div v-if="hasNodeSearched && nodeSearchKeyword && searchedNodes.length > 0" class="search-result-tip">
            找到 {{ searchedNodes.length }} 个节点
          </div>
          <div v-else-if="hasNodeSearched && nodeSearchKeyword && !searchingNodes" class="search-result-tip empty">
            未找到匹配的节点
          </div>
          <el-select
            v-model="selectedNodeIds"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            placeholder="请选择专线节点"
            class="form-control"
            no-data-text="请先搜索专线节点"
            @change="handleSelectedNodesChange"
          >
            <el-option
              v-for="node in assignNodeOptions"
              :key="node.id"
              :label="formatNodeOption(node)"
              :value="node.id"
            />
          </el-select>
        </div>

        <el-form label-position="top" class="assign-options-form">
          <el-form-item label="专线到期时间">
            <el-date-picker
              v-model="assignExpiresAt"
              type="datetime"
              placeholder="不填则跟随用户订阅"
              class="form-control"
              value-format="YYYY-MM-DDTHH:mm:ssZ"
              :default-time="assignDefaultTime"
              clearable
            />
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

        <div class="form-tip">
          提示：提交后会把所选专线节点分配给所选用户，重复分配会由系统自动跳过。
        </div>
      </div>

      <template #footer>
        <FormActionBar
          :loading="assigning"
          :disabled="assignButtonDisabled"
          submit-text="确认分配"
          @cancel="showAssignDialog = false"
          @submit="assignNode"
        />
      </template>
    </AppDialog>
  </AppDrawer>
</template>

<script>
import { adminAPI } from '@/utils/api'
import { formatDate as formatDateUtil } from '@/utils/date'
import { ElMessage } from '@/utils/elementPlusServices'
import PaginationBar from '@/components/PaginationBar.vue'
import AppDrawer from '@/components/AppDrawer.vue'
import AppDialog from '@/components/AppDialog.vue'
import EmptyState from '@/components/EmptyState.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import { confirmDelete, confirmWarning, confirmDanger } from '@/utils/confirmAction'
import {
  Wallet,
  ShoppingCart,
  Clock,
  Connection,
  Plus,
  Search,
  RefreshRight,
  CopyDocument,
  Monitor,
  Delete
} from '@element-plus/icons-vue'

const COUNTRY_NAME_ZH = {
  CN: '中国',
  HK: '中国香港',
  MO: '中国澳门',
  TW: '中国台湾',
  US: '美国',
  JP: '日本',
  KR: '韩国',
  SG: '新加坡',
  GB: '英国',
  UK: '英国',
  DE: '德国',
  FR: '法国',
  CA: '加拿大',
  AU: '澳大利亚',
  RU: '俄罗斯',
  IN: '印度',
  TH: '泰国',
  VN: '越南',
  MY: '马来西亚',
  ID: '印度尼西亚',
  PH: '菲律宾',
  BR: '巴西',
  TR: '土耳其',
  NL: '荷兰',
  ES: '西班牙',
  IT: '意大利',
  SE: '瑞典',
  CH: '瑞士',
  PL: '波兰',
  ZA: '南非',
  AE: '阿联酋',
  SA: '沙特阿拉伯',
  IR: '伊朗'
}

const COUNTRY_ALIASES_ZH = {
  china: '中国',
  'hong kong': '中国香港',
  macao: '中国澳门',
  macau: '中国澳门',
  taiwan: '中国台湾',
  'united states': '美国',
  usa: '美国',
  'united kingdom': '英国',
  japan: '日本',
  korea: '韩国',
  'south korea': '韩国',
  singapore: '新加坡',
  germany: '德国',
  france: '法国',
  canada: '加拿大',
  australia: '澳大利亚',
  russia: '俄罗斯',
  india: '印度',
  thailand: '泰国',
  vietnam: '越南',
  malaysia: '马来西亚',
  indonesia: '印度尼西亚',
  philippines: '菲律宾',
  brazil: '巴西',
  turkey: '土耳其',
  netherlands: '荷兰',
  spain: '西班牙',
  italy: '意大利',
  sweden: '瑞典',
  switzerland: '瑞士',
  poland: '波兰',
  'south africa': '南非',
  'united arab emirates': '阿联酋',
  'saudi arabia': '沙特阿拉伯',
  iran: '伊朗'
}

export default {
  name: 'UserDetailDialog',
  components: {
    Wallet,
    ShoppingCart,
    Clock,
    Connection,
    Plus,
    Search,
    RefreshRight,
    CopyDocument,
    Monitor,
    Delete,
    PaginationBar,
    AppDrawer,
    AppDialog,
    EmptyState,
    FormActionBar
  },
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    user: {
      type: Object,
      default: () => null
    },
    isMobile: {
      type: Boolean,
      default: false
    },
    initialTab: {
      type: String,
      default: 'orders'
    }
  },
  emits: ['update:visible', 'custom-nodes-updated'],
  data() {
    return {
      activeTab: this.initialTab,
      customNodes: [],
      selectedCustomNodes: [],
      batchUnassigning: false,
      clearingNodes: false,
      searchedUsers: [],
      selectedUserIds: [],
      selectedUserCache: [],
      userSearchKeyword: '',
      searchedNodes: [],
      selectedNodeIds: [],
      selectedNodeCache: [],
      nodeSearchKeyword: '',
      showAssignDialog: false,
      assigning: false,
      searchingUsers: false,
      searchingNodes: false,
      hasUserSearched: false,
      hasNodeSearched: false,
      loadingNodes: false,
      lineModeForm: 'normal',
      savingLineMode: false,
      assignSubscriptionType: 'both',
      assignDeviceLimitMode: 'system',
      assignExpiresAt: '',
      assignDefaultTime: new Date(2000, 1, 1, 23, 59, 59),
      devices: [],
      loadingDevices: false,
      deletingDeviceId: null,
      subscriptionExcludedProtocols: {},
      protocolOptions: [
        { label: 'AnyTLS', value: 'anytls' },
        { label: 'VMess', value: 'vmess' },
        { label: 'VLESS', value: 'vless' },
        { label: 'Trojan', value: 'trojan' },
        { label: 'Shadowsocks', value: 'ss' },
        { label: 'Hysteria2', value: 'hysteria2' },
        { label: 'TUIC', value: 'tuic' },
        { label: 'SOCKS', value: 'socks' },
        { label: 'HTTP', value: 'http' }
      ],
      checkinLogs: [],
      checkinLoaded: false,
      loadingCheckins: false,
      exportingCheckins: false,
      checkinPagination: {
        page: 1,
        size: 10,
        total: 0
      }
    }
  },
  computed: {
    rechargeRecords() {
      return this.user?.recharge_records || []
    },
    orderRecords() {
      return this.user?.orders || []
    },
    subscriptionResets() {
      return this.user?.subscription_resets || []
    },
    uaRecords() {
      return this.user?.ua_records || []
    },
    loginHistory() {
      return this.user?.login_history || []
    },
    currentUserId() {
      return this.user?.user_info?.id || this.user?.id
    },
    currentUserOption() {
      if (!this.currentUserId) return null
      return {
        id: this.currentUserId,
        username: this.user?.user_info?.username || this.user?.username || '',
        email: this.user?.user_info?.email || this.user?.email || '',
        notes: this.user?.user_info?.notes || this.user?.notes || ''
      }
    },
    assignUserOptions() {
      return this.uniqueById([
        ...this.selectedUserCache,
        ...this.searchedUsers,
        ...(this.currentUserOption ? [this.currentUserOption] : [])
      ])
    },
    assignNodeOptions() {
      return this.uniqueById([
        ...this.selectedNodeCache,
        ...this.searchedNodes
      ])
    },
    assignButtonDisabled() {
      return !this.selectedUserIds.length || !this.selectedNodeIds.length
    },
    hasAssignedCustomNodes() {
      const info = this.user?.user_info || this.user || {}
      return (this.customNodes && this.customNodes.length > 0) || Boolean(info.is_special_node_user) || Number(info.custom_node_count || 0) > 0
    }
  },
  watch: {
    visible(val, oldVal) {
      if (val && !oldVal && this.user) {
        this.resetDialogState()
      }
    },
    // 抽屉已打开时切换用户：重置所有按用户缓存的状态，避免展示上一个用户的数据
    user(newUser, oldUser) {
      if (newUser && (!oldUser || oldUser.id !== newUser.id || (oldUser.user_info?.id || oldUser.id) !== (newUser.user_info?.id || newUser.id))) {
        if (this.visible) {
          this.resetDialogState()
        }
      }
    },
    activeTab(val) {
      if (val === 'custom-nodes' && this.customNodes.length === 0) {
        this.loadUserCustomNodes()
      } else if (val === 'devices' && this.devices.length === 0 && !this.loadingDevices) {
        this.loadDevices()
      } else if (val === 'checkins' && !this.checkinLoaded && !this.loadingCheckins) {
        this.loadCheckinLogs()
      }
    }
  },
  beforeUnmount() {
    this._unmounted = true
  },
  methods: {
    // 重置所有按用户缓存的状态（打开或切换用户时调用）
    resetDialogState() {
      this.activeTab = this.initialTab
      this.devices = []
      this.customNodes = []
      this.selectedCustomNodes = []
      this.checkinLogs = []
      this.checkinLoaded = false
      this.subscriptionExcludedProtocols = {}
      this.checkinPagination.page = 1
      this.checkinPagination.size = 20
      this.checkinPagination.total = 0
      this.syncLineModeForm()
      if (this.activeTab === 'devices') {
        this.loadDevices()
      } else if (this.activeTab === 'custom-nodes') {
        this.assignSubscriptionType = 'both'
        this.assignDeviceLimitMode = 'system'
        this.loadUserCustomNodes()
      } else if (this.activeTab === 'checkins') {
        this.loadCheckinLogs()
      }
    },
    getDeviceTypeName(type) {
      const map = {
        mobile: '手机',
        desktop: '电脑',
        tablet: '平板',
        router: '路由器',
        tv_box: '电视盒子',
        server: '服务器',
        unknown: '未知'
      }
      return map[type] || type || '未知'
    },
    getDeviceTypeColor(type) {
      const map = {
        mobile: 'primary',
        desktop: 'success',
        tablet: 'warning',
        router: '',
        tv_box: 'danger',
        server: 'info',
        unknown: 'info'
      }
      return map[type] || 'info'
    },
    displayLocation(loc) {
      if (!loc) return '-'
      const country = this.getLocationCountry(loc)
      return country || '-'
    },
    getLocationCountry(loc) {
      if (!loc) return ''
      if (typeof loc === 'object') {
        return this.getChineseCountryName(loc.country_code || loc.countryCode, loc.country || loc.country_name || loc.countryName)
      }
      const text = String(loc).trim()
      if (!text) return ''
      try {
        const parsed = JSON.parse(text)
        if (parsed && typeof parsed === 'object') {
          return this.getChineseCountryName(parsed.country_code || parsed.countryCode, parsed.country || parsed.country_name || parsed.countryName)
        }
      } catch (e) {
        // Plain text location; parse below.
      }
      const country = text.includes(',') ? text.split(',')[0].trim() : text
      return this.getChineseCountryName('', country)
    },
    getChineseCountryName(countryCode, countryName) {
      const code = String(countryCode || '').trim().toUpperCase()
      if (code && COUNTRY_NAME_ZH[code]) return COUNTRY_NAME_ZH[code]
      const name = String(countryName || '').trim()
      if (!name) return ''
      const alias = COUNTRY_ALIASES_ZH[name.toLowerCase()]
      if (alias) return alias
      return name
    },
    formatDate(date) {
      if (!date) return ''
      return formatDateUtil(date)
    },
    formatDateTime(date) {
      if (!date) return ''
      return formatDateUtil(date)
    },
    hasMoreSubscriptionUrls(sub) {
      return this.getMoreSubscriptionUrls(sub).length > 0
    },
    getMoreSubscriptionUrls(sub) {
      return [
        { label: 'Stash', type: 'stash' },
        { label: 'Surge', type: 'surge' },
        { label: 'Quantumult X', type: 'quantumultx' },
        { label: 'Loon', type: 'loon' },
        { label: 'Sing-Box', type: 'singbox' },
        { label: 'Shadowrocket', type: 'shadowrocket' }
      ].map(client => ({
        ...client,
        url: this.getSubscriptionUrlWithExclude(sub, this.getTypedSubscriptionUrl(sub, client.type))
      })).filter(client => client.url)
    },
    getSubscriptionExcludeKey(sub) {
      return String(sub?.id || sub?.subscription_url || sub?.universal_url || sub?.clash_url || 'default')
    },
    getExcludedProtocols(sub) {
      return this.subscriptionExcludedProtocols[this.getSubscriptionExcludeKey(sub)] || []
    },
    setExcludedProtocols(sub, value) {
      const key = this.getSubscriptionExcludeKey(sub)
      this.subscriptionExcludedProtocols = {
        ...this.subscriptionExcludedProtocols,
        [key]: Array.isArray(value) ? value : []
      }
    },
    clearExcludedProtocols(sub) {
      this.setExcludedProtocols(sub, [])
    },
    getSubscriptionUrlWithExclude(sub, url) {
      if (!url) return ''
      const excluded = this.getExcludedProtocols(sub)
      if (!excluded.length) return url
      try {
        const parsed = new URL(url, window.location.origin)
        parsed.searchParams.set('exclude', excluded.join(','))
        return parsed.toString()
      } catch (e) {
        const separator = url.includes('?') ? '&' : '?'
        return `${url}${separator}exclude=${encodeURIComponent(excluded.join(','))}`
      }
    },
    getTypedSubscriptionUrl(sub, type) {
      if (!sub) return ''
      const field = `${type}_url`
      if (sub[field]) return sub[field]
      const token = sub.subscription_url
      const base = sub.universal_url || sub.clash_url || ''
      if (base) {
        try {
          const url = new URL(base, window.location.origin)
          url.searchParams.set('type', type)
          return url.toString()
        } catch (e) {
          const separator = base.includes('?') ? '&' : '?'
          return `${base}${separator}type=${encodeURIComponent(type)}`
        }
      }
      if (!token) return ''
      return `${window.location.origin}/api/v1/client/subscribe?token=${encodeURIComponent(token)}&type=${encodeURIComponent(type)}`
    },
    getStatusType(status) {
      const statusMap = {
        active: 'success',
        inactive: 'info',
        paid: 'success',
        pending: 'warning',
        cancelled: 'info',
        refunded: 'danger',
        expired: 'danger',
        success: 'success',
        failed: 'danger'
      }
      return statusMap[status] || 'info'
    },
    getStatusText(status) {
      const statusMap = {
        active: '活跃',
        inactive: '未激活',
        paid: '已支付',
        pending: '待支付',
        cancelled: '已取消',
        refunded: '已退款',
        expired: '已过期',
        success: '成功',
        failed: '失败'
      }
      return statusMap[status] || status
    },
    getPaymentMethodText(method) {
      const methodMap = {
        alipay: '支付宝',
        wechat: '微信支付',
        balance: '余额',
        card: '银行卡',
        other: '其他'
      }
      return methodMap[method] || method || '未知'
    },
    getResetTypeText(type) {
      const typeMap = {
        admin_reset: '管理员重置',
        user_reset: '用户重置',
        admin_batch_reset: '批量重置'
      }
      return typeMap[type] || type
    },
    async copyToClipboard(text) {
      if (!text) {
        ElMessage.warning('无可复制内容')
        return
      }
      try {
        await navigator.clipboard.writeText(text)
        ElMessage.success('复制成功')
      } catch (err) {
        ElMessage.error('复制失败')
      }
    },
    getCurrentUserId() {
      return this.user?.user_info?.id || this.user?.id
    },
    uniqueById(items) {
      const seen = new Set()
      const result = []
      for (const item of items || []) {
        if (!item || item.id === undefined || item.id === null || seen.has(item.id)) {
          continue
        }
        seen.add(item.id)
        result.push(item)
      }
      return result
    },
    formatUserOption(user) {
      if (!user) return ''
      const username = user.username || '未命名用户'
      return user.email ? `${username} (${user.email})` : username
    },
    getNodeAddress(node) {
      if (!node) return ''
      const address = node.address || node.node_address || node.domain || ''
      if (address && node.port && node.port !== 443 && !String(address).includes(':')) {
        return `${address}:${node.port}`
      }
      return address
    },
    formatNodeOption(node) {
      if (!node) return ''
      const name = node.display_name || node.name || node.node_name || '未命名节点'
      const address = this.getNodeAddress(node)
      return address ? `${name} (${address})` : name
    },
    getAssignedNodeIdSet() {
      return new Set((this.customNodes || []).map(n => n.node_id || n.id).filter(Boolean))
    },
    resetAssignForm() {
      this.userSearchKeyword = ''
      this.searchedUsers = []
      this.selectedUserCache = this.currentUserOption ? [this.currentUserOption] : []
      this.selectedUserIds = this.currentUserId ? [this.currentUserId] : []
      this.nodeSearchKeyword = ''
      this.searchedNodes = []
      this.hasUserSearched = false
      this.hasNodeSearched = false
      this.selectedNodeCache = []
      this.selectedNodeIds = []
      this.assignSubscriptionType = 'both'
      this.assignDeviceLimitMode = 'system'
      this.assignExpiresAt = ''
    },
    getUserLineMode() {
      const info = this.user?.user_info || this.user || {}
      if (info.special_node_subscription_type === 'special_only') return 'special_only'
      if (info.special_node_subscription_type === 'both' && this.hasAssignedCustomNodes) return 'both'
      return 'normal'
    },
    syncLineModeForm() {
      this.lineModeForm = this.getUserLineMode()
    },
    getLineModeText(mode) {
      if (mode === 'special_only') return '仅专线'
      if (mode === 'both') return '专线+普通'
      return '普通线路'
    },
    getLineModeTagType(mode) {
      if (mode === 'special_only') return 'danger'
      if (mode === 'both') return 'warning'
      return 'info'
    },
    async updateLineMode(mode) {
      const userId = this.currentUserId
      if (!userId) {
        this.syncLineModeForm()
        ElMessage.error('用户ID不存在')
        return
      }
      if (mode !== 'normal' && !this.hasAssignedCustomNodes) {
        this.syncLineModeForm()
        ElMessage.warning('请先给用户分配专线节点')
        return
      }
      if (mode === this.getUserLineMode()) return
      this.savingLineMode = true
      try {
        await adminAPI.updateUser(userId, { special_node_subscription_type: mode })
        const info = this.user?.user_info || this.user
        if (info) {
          info.special_node_subscription_type = mode
        }
        this.lineModeForm = mode
        ElMessage.success('线路模式已更新')
        this.$emit('custom-nodes-updated', {
          userIds: [userId],
          hasCustomNodes: this.hasAssignedCustomNodes,
          subscriptionType: mode
        })
      } catch (error) {
        this.syncLineModeForm()
        ElMessage.error('线路模式更新失败: ' + (error.response?.data?.message || error.message))
      } finally {
        this.savingLineMode = false
      }
    },
    async openAssignDialog() {
      this.resetAssignForm()
      this.showAssignDialog = true
      await this.handleNodeSearch({ silent: true })
    },
    handleSelectedUsersChange(ids) {
      const optionMap = new Map(this.assignUserOptions.map(item => [item.id, item]))
      this.selectedUserCache = ids.map(id => optionMap.get(id)).filter(Boolean)
    },
    handleSelectedNodesChange(ids) {
      const optionMap = new Map(this.assignNodeOptions.map(item => [item.id, item]))
      this.selectedNodeCache = ids.map(id => optionMap.get(id)).filter(Boolean)
    },
    async loadCheckinLogs() {
      const userId = this.getCurrentUserId()
      if (!userId) {
        this.checkinLogs = []
        this.checkinPagination.total = 0
        return
      }
      this.loadingCheckins = true
      try {
        const params = {
          page: this.checkinPagination.page,
          size: this.checkinPagination.size
        }
        const response = await adminAPI.getUserCheckinLogs(userId, params)
        if (response?.data?.success) {
          const data = response.data.data || {}
          this.checkinLogs = data.logs || []
          this.checkinPagination.total = data.total || 0
          this.checkinLoaded = true
        } else {
          this.checkinLogs = []
          this.checkinPagination.total = 0
          ElMessage.error(response?.data?.message || '加载签到日志失败')
        }
      } catch (error) {
        this.checkinLogs = []
        this.checkinPagination.total = 0
        ElMessage.error('加载签到日志失败: ' + (error.response?.data?.message || error.message))
      } finally {
        this.loadingCheckins = false
      }
    },
    handleCheckinSizeChange(size) {
      this.checkinPagination.size = size
      this.checkinPagination.page = 1
      this.loadCheckinLogs()
    },
    handleCheckinPageChange(page) {
      this.checkinPagination.page = page
      this.loadCheckinLogs()
    },
    async exportCheckinLogs() {
      const userId = this.getCurrentUserId()
      if (!userId) {
        ElMessage.warning('用户ID不存在')
        return
      }
      this.exportingCheckins = true
      try {
        const response = await adminAPI.exportUserCheckinLogs(userId, {})
        if (response?.data instanceof Blob) {
          const url = window.URL.createObjectURL(response.data)
          const a = document.createElement('a')
          a.href = url
          a.download = `user_${userId}_checkin_logs_${new Date().toISOString().split('T')[0]}.csv`
          document.body.appendChild(a)
          a.click()
          document.body.removeChild(a)
          window.URL.revokeObjectURL(url)
          ElMessage.success('签到日志导出成功')
          return
        }
        ElMessage.error('导出失败：响应格式不正确')
      } catch (error) {
        if (error.response?.data instanceof Blob) {
          try {
            const text = await error.response.data.text()
            const errData = JSON.parse(text)
            ElMessage.error(errData.message || '导出签到日志失败')
          } catch (e) {
            ElMessage.error('导出签到日志失败')
          }
        } else {
          ElMessage.error('导出签到日志失败: ' + (error.response?.data?.message || error.message))
        }
      } finally {
        this.exportingCheckins = false
      }
    },
    async loadDevices() {
      if (this._unmounted) return
      const userId = this.user?.user_info?.id || this.user?.id
      if (!userId) {
        this.devices = []
        return
      }
      const subscriptions = this.user?.subscriptions || []
      if (subscriptions.length === 0) {
        this.devices = []
        return
      }
      this.loadingDevices = true
      try {
        const subIds = subscriptions
          .map(sub => sub.id || sub.subscription_id)
          .filter(Boolean)
        const parseDevices = (response, subId) => {
          if (!response || !response.data) return []
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
          return devices.map(device => ({
            id: device.id,
            device_name: device.device_name || device.name || '未知设备',
            device_type: device.device_type || device.type || 'unknown',
            ip_address: device.ip_address || device.ip || '-',
            location: device.location || '',
            last_seen: device.last_seen || device.last_access || null,
            last_access: device.last_access || device.last_seen || null,
            access_count: device.access_count || 0,
            is_active: device.is_active !== false,
            user_agent: device.user_agent || '',
            software_name: device.software_name || '',
            subscription_id: subId
          }))
        }
        // 限制并发数为5，避免大量订阅时同时发起过多请求
        const CONCURRENCY = 5
        const allDevices = []
        for (let i = 0; i < subIds.length; i += CONCURRENCY) {
          if (this._unmounted) return
          const batch = subIds.slice(i, i + CONCURRENCY)
          const results = await Promise.all(
            batch.map(subId =>
              adminAPI.getSubscriptionDevices(subId)
                .then(response => parseDevices(response, subId))
                .catch(() => [])
            )
          )
          allDevices.push(...results.flat())
        }
        if (!this._unmounted) {
          this.devices = allDevices
        }
      } catch (error) {
        console.error('加载设备列表失败:', error)
        this.devices = []
      } finally {
        this.loadingDevices = false
      }
    },
    async deleteDevice(device) {
      try {
        await confirmDelete(
          `确定要删除设备 "${device.device_name || '未知设备'}" 吗？删除后该设备将无法继续使用订阅。`,
          '确认删除'
        )
        this.deletingDeviceId = device.id
        const response = await adminAPI.removeDevice(device.id)
        if (response.data && response.data.success) {
          ElMessage.success('设备删除成功')
          await this.loadDevices()
        } else {
          throw new Error(response.data?.message || '删除设备失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('删除设备失败: ' + (error.response?.data?.message || error.message))
        }
      } finally {
        this.deletingDeviceId = null
      }
    },
    async loadUserCustomNodes() {
      if (!this.user?.user_info?.id && !this.user?.id) {
        return
      }
      this.loadingNodes = true
      try {
        const userId = this.user.user_info?.id || this.user.id
        const response = await adminAPI.getUserCustomNodes(userId)
        if (response.data && response.data.success) {
          this.customNodes = response.data.data || []
          this.syncLineModeForm()
        } else {
          this.customNodes = []
          this.syncLineModeForm()
        }
      } catch (error) {
        console.error('加载专线节点失败:', error)
        this.customNodes = []
        this.syncLineModeForm()
      } finally {
        this.loadingNodes = false
        this.selectedCustomNodes = []
      }
    },
    async handleUserSearch() {
      const keyword = this.userSearchKeyword.trim()
      if (!keyword) {
        ElMessage.warning('请输入用户搜索关键词')
        return
      }
      this.hasUserSearched = true
      this.searchingUsers = true
      try {
        const response = await adminAPI.getUsers({ keyword, page: 1, size: 50 })
        if (response.data && response.data.success) {
          const users = response.data.data?.users || response.data.data?.data || response.data.data || []
          this.searchedUsers = this.uniqueById(users)
          this.handleSelectedUsersChange(this.selectedUserIds)
          if (this.searchedUsers.length === 0) {
            ElMessage.info('未找到匹配的用户')
          }
        } else {
          ElMessage.error(response.data?.message || '搜索用户失败')
        }
      } catch (error) {
        console.error('搜索用户失败:', error)
        ElMessage.error('搜索用户失败: ' + (error.response?.data?.message || error.message))
      } finally {
        this.searchingUsers = false
      }
    },
    handleUserSearchClear() {
      this.userSearchKeyword = ''
      this.searchedUsers = []
      this.hasUserSearched = false
      this.handleSelectedUsersChange(this.selectedUserIds)
    },
    async handleNodeSearch(options = {}) {
      const keyword = this.nodeSearchKeyword.trim()
      this.hasNodeSearched = Boolean(keyword)
      this.searchingNodes = true
      try {
        const params = { is_active: 'true', page: 1, size: 100 }
        if (keyword) {
          params.search = keyword
        }
        const response = await adminAPI.getCustomNodes(params)
        if (response.data && response.data.success) {
          const allNodes = response.data.data?.data || response.data.data?.nodes || response.data.data || []
          const assignedIds = this.getAssignedNodeIdSet()
          this.searchedNodes = this.uniqueById(allNodes).filter(node => !assignedIds.has(node.id))
          this.handleSelectedNodesChange(this.selectedNodeIds)
          if (keyword && this.searchedNodes.length === 0 && !options.silent) {
            ElMessage.info('未找到匹配的节点')
          }
        } else {
          ElMessage.error(response.data?.message || '搜索节点失败')
        }
      } catch (error) {
        console.error('搜索节点失败:', error)
        ElMessage.error('搜索节点失败: ' + (error.response?.data?.message || error.message))
      } finally {
        this.searchingNodes = false
      }
    },
    handleNodeSearchClear() {
      this.nodeSearchKeyword = ''
      this.searchedNodes = []
      this.hasNodeSearched = false
      this.handleSelectedNodesChange(this.selectedNodeIds)
    },
    async assignNode() {
      if (!this.selectedNodeIds.length) {
        ElMessage.warning('请选择要分配的专线节点')
        return
      }
      if (!this.selectedUserIds.length) {
        ElMessage.warning('请选择要分配的用户')
        return
      }
      if (!this.currentUserId) {
        ElMessage.error('当前用户ID不存在')
        return
      }
      this.assigning = true
      try {
        const extraData = {
          subscription_type: this.assignSubscriptionType,
          unlimited_devices: this.assignDeviceLimitMode === 'unlimited',
          expires_at: this.assignExpiresAt || null
        }
        const response = await adminAPI.batchAssignCustomNodes(this.selectedNodeIds, this.selectedUserIds, extraData)
        if (response.data && response.data.success) {
          const affectedUserIds = [...this.selectedUserIds]
          const subscriptionType = this.assignSubscriptionType
          ElMessage.success(response.data.message || '分配成功')
          this.showAssignDialog = false
          this.resetAssignForm()
          await this.loadUserCustomNodes()
          const info = this.user?.user_info || this.user
          if (info && affectedUserIds.includes(this.currentUserId)) {
            info.special_node_subscription_type = subscriptionType
            info.custom_node_count = this.customNodes.length
            info.is_special_node_user = this.customNodes.length > 0
            this.lineModeForm = subscriptionType
          }
          this.$emit('custom-nodes-updated', {
            userIds: affectedUserIds,
            hasCustomNodes: true,
            subscriptionType
          })
        } else {
          ElMessage.error(response.data?.message || '分配失败')
        }
      } catch (error) {
        console.error('分配节点失败:', error)
        ElMessage.error('分配节点失败: ' + (error.response?.data?.message || error.message))
      } finally {
        this.assigning = false
      }
    },
    async unassignNode(nodeId) {
      const userId = this.user.user_info?.id || this.user.id
      if (!userId || !nodeId) {
        ElMessage.error('参数错误')
        return
      }
      try {
        // 如果是用户最后一个专线节点，弹出确认提示
        const isLastNode = this.customNodes.length === 1
        if (isLastNode) {
          await confirmWarning(
            '这是该用户的最后一个专线节点。取消后用户将无法访问任何专线节点，系统将自动恢复其普通线路访问。\n\n确认取消分配？',
            '取消分配专线节点'
          )
        }
        const response = await adminAPI.unassignCustomNodeFromUser(userId, nodeId)
        if (response.data && response.data.success) {
          ElMessage.success('取消分配成功')
          await this.loadUserCustomNodes()
          const info = this.user?.user_info || this.user
          if (info) {
            info.custom_node_count = this.customNodes.length
            info.is_special_node_user = this.customNodes.length > 0
            if (this.customNodes.length === 0) {
              info.special_node_subscription_type = 'normal'
              this.lineModeForm = 'normal'
            }
          }
          this.$emit('custom-nodes-updated', {
            userIds: [userId],
            hasCustomNodes: this.customNodes.length > 0,
            subscriptionType: this.customNodes.length > 0 ? this.getUserLineMode() : 'normal'
          })
        } else {
          ElMessage.error(response.data?.message || '取消分配失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          console.error('取消分配失败:', error)
          ElMessage.error('取消分配失败: ' + (error.response?.data?.message || error.message))
        }
      }
    },
    handleCustomNodeSelectionChange(selection) {
      this.selectedCustomNodes = selection || []
    },
    getSelectedNodeIds() {
      return this.selectedCustomNodes
        .map(node => node.node_id || node.id)
        .filter(Boolean)
    },
    async batchUnassignSelectedNodes() {
      const userId = this.user.user_info?.id || this.user.id
      const nodeIds = this.getSelectedNodeIds()
      if (!userId || nodeIds.length === 0) {
        ElMessage.warning('请先勾选要取消分配的专线节点')
        return
      }
      try {
        await confirmWarning(
          `确定要取消分配选中的 ${nodeIds.length} 个专线节点吗？取消后该用户将无法继续使用这些节点。`,
          {
            title: '批量取消分配',
            confirmButtonText: '确定取消'
          }
        )
      } catch {
        return
      }
      this.batchUnassigning = true
      try {
        const response = await adminAPI.batchUnassignCustomNodes(nodeIds, [userId])
        if (response.data && response.data.success) {
          ElMessage.success(response.data.message || `成功取消 ${nodeIds.length} 个节点分配`)
          this.handleCustomNodesCleared(userId)
        } else {
          ElMessage.error(response.data?.message || '批量取消分配失败')
        }
      } catch (error) {
        console.error('批量取消分配失败:', error)
        ElMessage.error('批量取消分配失败: ' + (error.response?.data?.message || error.message))
      } finally {
        this.batchUnassigning = false
      }
    },
    async clearAllCustomNodes() {
      const userId = this.user.user_info?.id || this.user.id
      const nodeIds = (this.customNodes || []).map(node => node.node_id || node.id).filter(Boolean)
      if (!userId || nodeIds.length === 0) {
        ElMessage.warning('该用户当前没有已分配的专线节点')
        return
      }
      try {
        await confirmDanger(
          `确定要清空该用户的全部 ${nodeIds.length} 个专线节点吗？清空后用户将无法访问任何专线节点，系统会自动恢复其普通线路访问。`,
          {
            title: '清空专线节点',
            confirmButtonText: '确定清空'
          }
        )
      } catch {
        return
      }
      this.clearingNodes = true
      try {
        const response = await adminAPI.batchUnassignCustomNodes(nodeIds, [userId])
        if (response.data && response.data.success) {
          ElMessage.success(response.data.message || `已清空 ${nodeIds.length} 个专线节点`)
          this.handleCustomNodesCleared(userId)
        } else {
          ElMessage.error(response.data?.message || '清空专线节点失败')
        }
      } catch (error) {
        console.error('清空专线节点失败:', error)
        ElMessage.error('清空专线节点失败: ' + (error.response?.data?.message || error.message))
      } finally {
        this.clearingNodes = false
      }
    },
    async handleCustomNodesCleared(userId) {
      await this.loadUserCustomNodes()
      const info = this.user?.user_info || this.user
      if (info) {
        info.custom_node_count = this.customNodes.length
        info.is_special_node_user = this.customNodes.length > 0
        if (this.customNodes.length === 0) {
          info.special_node_subscription_type = 'normal'
          this.lineModeForm = 'normal'
        }
      }
      this.$emit('custom-nodes-updated', {
        userIds: [userId],
        hasCustomNodes: this.customNodes.length > 0,
        subscriptionType: this.customNodes.length > 0 ? this.getUserLineMode() : 'normal'
      })
    }
  }
}
</script>

<style lang="scss" scoped>
.user-detail-drawer {
  .drawer-content {
    padding: 0;
  }

  .balance-highlight {
    font-weight: 600;
    color: #409eff;
    font-size: 14px;
  }

  .data-table,
  .form-control {
    width: 100%;
  }

  .detail-empty-state {
    min-height: 168px;
    padding: 28px 16px;
  }

  .compact-divider {
    margin: 16px 0 12px;
  }

  .expired-text {
    color: #f56c6c;
  }

  .expired-badge {
    color: #f56c6c;
    font-weight: 600;
    margin-left: 4px;
  }

  .subscription-section {
    margin-bottom: 20px;

    &:last-child {
      margin-bottom: 0;
    }
  }

  .url-section {
    margin-top: 12px;
    padding: 12px;
    background: #f5f7fa;
    border: 1px solid #e4e7ed;
    border-radius: 6px;
    display: flex;
    flex-direction: column;
    gap: 12px;

    .more-urls-collapse {
      margin-top: 4px;
      border: none;
      background: transparent;

      :deep(.el-collapse-item__header) {
        font-size: 13px;
        color: #409eff;
        background: transparent;
        border: none;
        padding: 6px 0;
        font-weight: 500;
      }
      :deep(.el-collapse-item__wrap) {
        background: transparent;
        border: none;
      }
      :deep(.el-collapse-item__content) {
        padding: 8px 0 0 0;
        display: flex;
        flex-direction: column;
        gap: 12px;
      }
    }
  }

  .protocol-exclude-panel {
    padding: 12px;
    border: 1px solid #e4e7ed;
    border-radius: 6px;
    background: #fff;

    .protocol-exclude-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 10px;
      margin-bottom: 10px;
    }

    .exclude-title {
      color: #303133;
      font-size: 13px;
      font-weight: 600;
      line-height: 1.4;
    }

    .exclude-subtitle {
      margin-top: 2px;
      color: #909399;
      font-size: 12px;
      line-height: 1.4;
    }

    .protocol-checkboxes {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;

      :deep(.el-checkbox-button__inner) {
        border: 1px solid #dcdfe6;
        border-radius: 6px;
        box-shadow: none;
        padding: 7px 10px;
        line-height: 1;
      }

      :deep(.el-checkbox-button:first-child .el-checkbox-button__inner),
      :deep(.el-checkbox-button:last-child .el-checkbox-button__inner) {
        border-radius: 6px;
      }
    }
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
      font-family: 'Courier New', Courier, monospace;
      font-size: 12px;
      color: #303133;
      background: #fff;
      padding: 8px 12px;
      border-radius: 3px;
      border: 1px solid #dcdfe6;
      word-break: break-all;
      line-height: 1.6;
      user-select: all;
      display: block;
    }

    .url-copy {
      width: 100%;
      text-align: left;
      cursor: pointer;
      appearance: none;

      &:hover:not(:disabled) {
        color: #409eff;
        border-color: #409eff;
        background: #f0f7ff;
      }

      &:disabled {
        cursor: default;
        color: #909399;
      }
    }
  }

  .records-tabs {
    margin-top: 20px;

    .el-table {
      font-size: 13px;
    }

    .amount-text {
      font-weight: 600;

      &.positive {
        color: #67c23a;
      }
    }

    .url-code-small {
      font-family: 'Courier New', Courier, monospace;
      font-size: 11px;
      color: #606266;
    }
  }

  .table-responsive {
    width: 100%;
    overflow-x: auto;
  }

  .custom-nodes-section {
    .line-mode-panel {
      margin-bottom: 14px;
      padding: 12px;
      border: 1px solid var(--el-border-color-light);
      border-radius: 6px;
      background: var(--el-fill-color-lighter);
    }

    .line-mode-heading {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      margin-bottom: 10px;
    }

    .line-mode-title {
      font-size: 13px;
      font-weight: 600;
      color: var(--el-text-color-primary);
      line-height: 1.3;
    }

    .line-mode-meta {
      margin-top: 2px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
      line-height: 1.3;
    }

    .line-mode-control {
      width: 100%;
      display: flex;
      flex-wrap: wrap;
      gap: 6px;

      :deep(.el-radio-button) {
        margin-right: 0;
      }
    }

    .custom-nodes-actions {
      margin-bottom: 15px;
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
    }
  }

  .devices-section {
    .devices-actions {
      margin-bottom: 12px;
      display: flex;
      align-items: center;
      gap: 10px;
    }

    .device-count-tip {
      font-size: 12px;
      color: #909399;
    }

    .ua-records-section {
      margin-top: 8px;
    }
  }

  .checkin-actions {
    margin-bottom: 12px;
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .checkin-pagination {
    margin-top: 12px;
    display: flex;
    justify-content: flex-end;
  }

  .node-search-section {
    margin-bottom: 15px;

    .search-input-group {
      display: flex;
      align-items: center;
      gap: 10px;
      margin-bottom: 8px;
    }

    .search-result-tip {
      font-size: 12px;
      color: #909399;
      margin-top: 5px;
      padding: 5px 0;

      &.empty {
        color: #f56c6c;
      }
    }
  }

  .form-tip {
    font-size: 12px;
    color: #909399;
    margin-top: 8px;
    line-height: 1.5;
  }

  .toggle-row {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
  }
  .toggle-hint {
    font-size: 11px;
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
    .drawer-content {
      padding: 0;
    }

    .balance-highlight {
      font-size: 13px;
    }

    .url-section {
      padding: 6px;
      gap: 6px;
    }

    .url-item {
      .url-header {
        .url-label {
          font-size: 11px;
        }
      }
      .url-code {
        font-size: 10px;
        padding: 4px 6px;
      }
    }

    .el-table {
      font-size: 11px;
    }

    :deep(.el-descriptions) {
      .el-descriptions__body {
        .el-descriptions__table {
          .el-descriptions__cell {
            padding: 4px 6px;
          }
          .el-descriptions__label {
            font-size: 11px;
            width: 62px;
            min-width: 62px;
            word-break: keep-all;
          }
          .el-descriptions__content {
            font-size: 11px;
            word-break: break-all;
          }
        }
      }
    }

    :deep(.el-tabs__item) {
      font-size: 12px;
      padding: 0 6px;
    }

    :deep(.el-divider__text) {
      font-size: 12px;
      padding: 0 6px;
    }

    :deep(.el-divider) {
      margin: 10px 0;
    }

    .subscription-section {
      margin-bottom: 8px;
    }

    .records-tabs {
      margin-top: 8px;
    }

    .custom-nodes-section {
      .custom-nodes-actions {
        margin-bottom: 10px;
        gap: 8px;
        flex-wrap: wrap;

        :deep(.el-button) {
          min-height: 44px;
          touch-action: manipulation;
        }
      }
    }

    .devices-section {
      .devices-actions {
        margin-bottom: 8px;
        gap: 8px;
        flex-wrap: wrap;

        :deep(.el-button) {
          min-height: 44px;
          touch-action: manipulation;
        }
      }
      .device-count-tip {
        font-size: 11px;
      }
    }

    .el-button {
      font-size: 12px;
      padding: 4px 8px;
      min-height: 44px;
      touch-action: manipulation;
    }

    .assign-option-group :deep(.el-radio-button__inner) {
      min-height: 44px;
      justify-content: center;
      touch-action: manipulation;
    }
  }
}

.assign-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 14px;
  max-height: 68vh;
  overflow-y: auto;
  padding-right: 4px;
}

.assign-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.assign-summary-item {
  min-width: 0;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-extra-light);

  span {
    display: block;
    margin-bottom: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  strong {
    display: block;
    overflow: hidden;
    color: var(--el-text-color-primary);
    font-size: 15px;
    font-weight: 700;
    line-height: 1.35;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.assign-section-card {
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.assign-section-header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.section-title {
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 700;
}

.section-desc {
  margin-top: 3px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.45;
}

.assign-dialog-content .search-input-group {
  margin-bottom: 8px;
}

.assign-dialog-content .search-input-group :deep(.el-input-group__append) {
  padding: 0;
}

.assign-dialog-content .search-input-group :deep(.el-input-group__append .el-button) {
  min-width: 82px;
}

.search-button-text {
  margin-left: 4px;
}

.assign-dialog-content .search-result-tip {
  margin: 5px 0 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;

  &.empty {
    color: var(--el-color-danger);
  }
}

.assign-options-form {
  padding: 14px 14px 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}

.assign-dialog-content .toggle-hint {
  margin-top: 6px;
  color: var(--el-text-color-secondary);
  font-size: 11px;
  line-height: 1.5;
}

.assign-dialog-content .assign-option-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.assign-dialog-content .assign-option-group :deep(.el-radio-button__inner) {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  border-left: 1px solid var(--el-border-color) !important;
  border-radius: 6px !important;
  touch-action: manipulation;
}

.assign-dialog-content .form-tip {
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

@media (max-width: 768px) {
  .assign-dialog-content {
    max-height: 70vh;
    padding-right: 0;
  }

  .assign-summary {
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .assign-section-card,
  .assign-options-form {
    padding: 12px;
  }

  .assign-dialog-content .search-input-group :deep(.el-input-group__append .el-button) {
    min-width: 68px;
    min-height: 44px;
    padding: 0 10px;
    touch-action: manipulation;
  }

  .assign-dialog-content .assign-option-group {
    display: grid;
    grid-template-columns: 1fr;
    width: 100%;
  }

  .assign-dialog-content .assign-option-group :deep(.el-radio-button),
  .assign-dialog-content .assign-option-group :deep(.el-radio-button__inner) {
    width: 100%;
  }

  .assign-dialog-content .assign-option-group :deep(.el-radio-button__inner) {
    min-height: 44px;
    justify-content: center;
    touch-action: manipulation;
  }
}
</style>
