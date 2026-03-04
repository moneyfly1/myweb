<template>
  <el-drawer
    :model-value="visible"
    @update:model-value="$emit('update:visible', $event)"
    :title="`用户详情 - ${user?.user_info?.username || user?.username || user?.user_info?.email || user?.email || ''}`"
    :size="isMobile ? '92%' : '780px'"
    direction="rtl"
    class="user-detail-drawer"
    :close-on-click-modal="true"
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
            <div class="url-item">
              <div class="url-header">
                <span class="url-label">通用订阅:</span>
                <el-button
                  size="small"
                  :icon="CopyDocument"
                  @click="copyToClipboard(sub.universal_url || sub.subscription_url)"
                  :disabled="!sub.universal_url && !sub.subscription_url"
                >
                  复制
                </el-button>
              </div>
              <code class="url-code">{{ sub.universal_url || sub.subscription_url || '无' }}</code>
            </div>
            <div class="url-item">
              <div class="url-header">
                <span class="url-label">Clash订阅:</span>
                <el-button
                  size="small"
                  :icon="CopyDocument"
                  @click="copyToClipboard(sub.clash_url)"
                  :disabled="!sub.clash_url"
                >
                  复制
                </el-button>
              </div>
              <code class="url-code">{{ sub.clash_url || '无' }}</code>
            </div>
          </div>

          <el-divider v-if="index < user.subscriptions.length - 1" />
        </div>
      </div>
      <el-empty v-else description="暂无订阅信息" :image-size="80" />

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
            style="width: 100%"
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
          <el-empty v-else description="暂无订单记录" :image-size="80" />
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
              style="width: 100%"
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
            <el-empty v-else-if="!loadingDevices" description="暂无在线设备" :image-size="80" />
            <div v-if="uaRecords && uaRecords.length > 0" class="ua-records-section">
              <el-divider content-position="left" style="margin: 16px 0 12px;">UA访问记录</el-divider>
              <el-table
                :data="uaRecords"
                size="small"
                max-height="200"
                style="width: 100%"
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
            style="width: 100%"
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
          <el-empty v-else description="暂无登录历史" :image-size="80" />
        </el-tab-pane>

        <!-- 重置记录 Tab -->
        <el-tab-pane label="重置记录" name="resets">
          <div v-if="subscriptionResets && subscriptionResets.length > 0" class="table-responsive">
            <el-table
              :data="subscriptionResets"
              size="small"
              max-height="240"
              style="width: 100%"
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
          <el-empty v-else description="暂无重置记录" :image-size="80" />
        </el-tab-pane>

        <!-- 充值记录 Tab -->
        <el-tab-pane label="充值记录" name="recharge">
          <el-table
            v-if="rechargeRecords && rechargeRecords.length > 0"
            :data="rechargeRecords"
            size="small"
            max-height="240"
            style="width: 100%"
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
          <el-empty v-else description="暂无充值记录" :image-size="80" />
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
            style="width: 100%"
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
          <el-empty v-else-if="!loadingCheckins" description="暂无签到日志" :image-size="80" />
          <div class="checkin-pagination">
            <el-pagination
              v-model:current-page="checkinPagination.page"
              v-model:page-size="checkinPagination.size"
              :total="checkinPagination.total"
              :page-sizes="[10, 20, 50, 100]"
              layout="total, sizes, prev, pager, next"
              small
              @size-change="handleCheckinSizeChange"
              @current-change="handleCheckinPageChange"
            />
          </div>
        </el-tab-pane>

        <!-- 专线节点 Tab -->
        <el-tab-pane label="专线节点" name="custom-nodes">
          <div class="custom-nodes-section">
            <div class="custom-nodes-actions">
              <el-button
                type="primary"
                size="small"
                :icon="Plus"
                @click="showAssignDialog = true"
              >
                分配专线节点
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
              style="width: 100%"
            >
              <el-table-column prop="node_name" label="节点名称" min-width="150" />
              <el-table-column prop="node_address" label="节点地址" min-width="200" show-overflow-tooltip />
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
            <el-empty v-else description="暂无专线节点" :image-size="80" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- 分配专线节点对话框 -->
    <el-dialog
      v-model="showAssignDialog"
      title="分配专线节点"
      width="500px"
      :close-on-click-modal="false"
      append-to-body
    >
      <div class="node-search-section">
        <div class="search-input-group">
          <el-input
            v-model="nodeSearchKeyword"
            placeholder="输入节点名称或地址搜索"
            clearable
            @clear="handleNodeSearchClear"
            :prefix-icon="Search"
          />
          <el-button type="primary" :icon="Search" @click="handleNodeSearch">
            搜索
          </el-button>
        </div>
        <div v-if="nodeSearchKeyword && searchedNodes.length > 0" class="search-result-tip">
          找到 {{ searchedNodes.length }} 个节点
        </div>
        <div v-else-if="nodeSearchKeyword && searchedNodes.length === 0" class="search-result-tip empty">
          未找到匹配的节点
        </div>
      </div>

      <el-form label-width="80px">
        <el-form-item label="选择节点">
          <el-select
            v-model="selectedNodeId"
            placeholder="请选择要分配的节点"
            filterable
            style="width: 100%"
          >
            <el-option
              v-for="node in searchedNodes"
              :key="node.id"
              :label="`${node.name} (${node.address})`"
              :value="node.id"
            />
          </el-select>
        </el-form-item>
      </el-form>

      <div class="form-tip">
        提示：专线节点分配后，用户可以在订阅中使用该节点。
      </div>

      <template #footer>
        <el-button @click="showAssignDialog = false">取消</el-button>
        <el-button
          type="primary"
          @click="assignNode"
          :loading="assigning"
          :disabled="!selectedNodeId"
        >
          确认分配
        </el-button>
      </template>
    </el-dialog>
  </el-drawer>
</template>

<script>
import { adminAPI } from '@/utils/api'
import { formatDate as formatDateUtil, formatLocation } from '@/utils/date'
import { ElMessage, ElMessageBox } from 'element-plus'
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
    Delete
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
  emits: ['update:visible'],
  data() {
    return {
      activeTab: this.initialTab,
      customNodes: [],
      searchedNodes: [],
      nodeSearchKeyword: '',
      showAssignDialog: false,
      selectedNodeId: null,
      assigning: false,
      loadingNodes: false,
      devices: [],
      loadingDevices: false,
      deletingDeviceId: null,
      checkinLogs: [],
      checkinLoaded: false,
      loadingCheckins: false,
      exportingCheckins: false,
      checkinPagination: {
        page: 1,
        size: 20,
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
    }
  },
  watch: {
    visible(val, oldVal) {
      if (val && !oldVal && this.user) {
        this.activeTab = this.initialTab
        this.devices = []
        this.customNodes = []
        this.checkinLogs = []
        this.checkinLoaded = false
        this.checkinPagination.page = 1
        this.checkinPagination.size = 20
        this.checkinPagination.total = 0
        if (this.activeTab === 'devices') {
          this.loadDevices()
        } else if (this.activeTab === 'custom-nodes') {
          this.loadUserCustomNodes()
        } else if (this.activeTab === 'checkins') {
          this.loadCheckinLogs()
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
      const result = formatLocation(loc)
      return result || loc
    },
    formatDate(date) {
      if (!date) return ''
      return formatDateUtil(date)
    },
    formatDateTime(date) {
      if (!date) return ''
      return formatDateUtil(date)
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
        await ElMessageBox.confirm(
          `确定要删除设备 "${device.device_name || '未知设备'}" 吗？删除后该设备将无法继续使用订阅。`,
          '确认删除',
          {
            confirmButtonText: '确定删除',
            cancelButtonText: '取消',
            type: 'warning',
          }
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
        } else {
          this.customNodes = []
        }
      } catch (error) {
        console.error('加载专线节点失败:', error)
        this.customNodes = []
      } finally {
        this.loadingNodes = false
      }
    },
    async handleNodeSearch() {
      if (!this.nodeSearchKeyword.trim()) {
        ElMessage.warning('请输入搜索关键词')
        return
      }
      try {
        const params = { is_active: 'true', page: 1, size: 200, search: this.nodeSearchKeyword.trim() }
        const response = await adminAPI.getCustomNodes(params)
        if (response.data && response.data.success) {
          const allNodes = response.data.data?.data || response.data.data || []
          const assignedIds = this.customNodes.map(n => n.id)
          this.searchedNodes = allNodes.filter(n => !assignedIds.includes(n.id))
          if (this.searchedNodes.length === 0) {
            ElMessage.info('未找到匹配的节点')
          }
        } else {
          ElMessage.error('搜索节点失败')
        }
      } catch (error) {
        console.error('搜索节点失败:', error)
        ElMessage.error('搜索节点失败')
      }
    },
    handleNodeSearchClear() {
      this.nodeSearchKeyword = ''
      this.searchedNodes = []
      this.selectedNodeId = null
    },
    async assignNode() {
      if (!this.selectedNodeId) {
        ElMessage.warning('请选择要分配的节点')
        return
      }
      const userId = this.user.user_info?.id || this.user.id
      if (!userId) {
        ElMessage.error('用户ID不存在')
        return
      }
      this.assigning = true
      try {
        const response = await adminAPI.assignCustomNodeToUser(userId, this.selectedNodeId)
        if (response.data && response.data.success) {
          ElMessage.success('分配成功')
          this.showAssignDialog = false
          this.selectedNodeId = null
          this.nodeSearchKeyword = ''
          this.searchedNodes = []
          await this.loadUserCustomNodes()
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
        const response = await adminAPI.unassignCustomNodeFromUser(userId, nodeId)
        if (response.data && response.data.success) {
          ElMessage.success('取消分配成功')
          await this.loadUserCustomNodes()
        } else {
          ElMessage.error(response.data?.message || '取消分配失败')
        }
      } catch (error) {
        console.error('取消分配失败:', error)
        ElMessage.error('取消分配失败: ' + (error.response?.data?.message || error.message))
      }
    }
  }
}
</script>

<style lang="scss" scoped>
.user-detail-drawer {
  .drawer-content {
    padding: 20px;
  }

  .balance-highlight {
    font-weight: 600;
    color: #409eff;
    font-size: 14px;
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
    border-radius: 4px;
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
    .custom-nodes-actions {
      margin-bottom: 15px;
      display: flex;
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

  @media (max-width: 768px) {
    .drawer-content {
      padding: 8px;
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
        gap: 6px;
      }
    }

    .devices-section {
      .devices-actions {
        margin-bottom: 8px;
        gap: 6px;
      }
      .device-count-tip {
        font-size: 11px;
      }
    }

    .el-button {
      font-size: 12px;
      padding: 4px 8px;
      min-height: 28px;
    }
  }
}
</style>
