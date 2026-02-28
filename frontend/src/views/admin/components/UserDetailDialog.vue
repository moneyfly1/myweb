<template>
  <el-drawer
    :model-value="visible"
    @update:model-value="$emit('update:visible', $event)"
    :title="`用户详情 - ${user?.user_info?.username || user?.username || user?.user_info?.email || user?.email || ''}`"
    :size="isMobile ? '100%' : '780px'"
    direction="rtl"
    class="user-detail-drawer"
    :close-on-click-modal="false"
  >
    <div v-if="user" class="drawer-content">
      <el-tabs v-model="activeTab" class="user-tabs">
        <!-- 基本信息 Tab -->
        <el-tab-pane label="基本信息" name="basic">
          <el-descriptions :column="isMobile ? 1 : 2" border>
            <el-descriptions-item label="用户ID">{{ user.user_info?.id || user.id }}</el-descriptions-item>
            <el-descriptions-item label="邮箱">{{ user.user_info?.email || user.email }}</el-descriptions-item>
            <el-descriptions-item label="用户名">{{ user.user_info?.username || user.username }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="getStatusType(user.user_info?.is_active !== false ? 'active' : 'inactive')">
                {{ getStatusText(user.user_info?.is_active !== false ? 'active' : 'inactive') }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="账户余额">
              <span class="balance-highlight">¥{{ ((user.user_info?.balance || user.balance || 0)).toFixed(2) }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="注册时间">{{ formatDate(user.user_info?.created_at || user.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="最后登录">{{ formatDate(user.user_info?.last_login || user.last_login) || '从未登录' }}</el-descriptions-item>
            <el-descriptions-item label="订阅数量">{{ user.statistics?.total_subscriptions || user.subscription_count || 0 }}</el-descriptions-item>
          </el-descriptions>

          <div class="user-stats" v-if="user.statistics">
            <h4>统计信息</h4>
            <el-row :gutter="20">
              <el-col :span="6" :xs="12">
                <el-statistic title="总消费" :value="user.statistics.total_spent" prefix="¥" />
              </el-col>
              <el-col :span="6" :xs="12">
                <el-statistic title="重置次数" :value="user.statistics.total_resets" />
              </el-col>
              <el-col :span="6" :xs="12">
                <el-statistic title="近30天重置" :value="user.statistics.recent_resets_30d" />
              </el-col>
              <el-col :span="6" :xs="12">
                <el-statistic title="订阅数量" :value="user.statistics.total_subscriptions" />
              </el-col>
            </el-row>
          </div>
        </el-tab-pane>

        <!-- 订阅列表 Tab -->
        <el-tab-pane label="订阅列表" name="subscriptions">
          <div v-if="user.subscriptions && user.subscriptions.length">
            <div class="table-responsive">
              <el-table :data="user.subscriptions" size="small" style="width: 100%">
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
          <el-empty v-else description="暂无订阅" :image-size="100" />
        </el-tab-pane>

        <!-- 订单记录 Tab -->
        <el-tab-pane label="订单记录" name="orders">
          <div class="records-list" v-if="orderRecords && orderRecords.length > 0" :key="'orders-' + orderRecords.length">
            <div
              v-for="order in orderRecords"
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
                  <span class="info-value">{{ getPaymentMethodText(order.payment_method || order.payment_method_name) }}</span>
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
                  <span class="info-value">{{ formatDateTime(order.created_at) }}</span>
                </div>
                <div class="record-info-row" v-if="order.payment_time">
                  <span class="info-label">支付时间：</span>
                  <span class="info-value">{{ formatDateTime(order.payment_time) }}</span>
                </div>
              </div>
            </div>
          </div>
          <el-empty v-else description="暂无订单记录" :image-size="100" />
        </el-tab-pane>

        <!-- 充值记录 Tab -->
        <el-tab-pane label="充值记录" name="recharge">
          <div class="records-list" v-if="rechargeRecords && rechargeRecords.length > 0" :key="'recharge-' + rechargeRecords.length">
            <div
              v-for="record in rechargeRecords"
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
                  <span class="info-value">{{ getPaymentMethodText(record.payment_method) }}</span>
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
                <div class="record-info-row" v-if="record.location">
                  <span class="info-label">归属地：</span>
                  <span class="info-value">{{ record.location }}</span>
                </div>
                <div class="record-info-row">
                  <span class="info-label">创建时间：</span>
                  <span class="info-value">{{ formatDateTime(record.created_at) }}</span>
                </div>
                <div class="record-info-row" v-if="record.paid_at">
                  <span class="info-label">支付时间：</span>
                  <span class="info-value">{{ formatDateTime(record.paid_at) }}</span>
                </div>
              </div>
            </div>
          </div>
          <el-empty v-else description="暂无充值记录" :image-size="100" />
        </el-tab-pane>

        <!-- 重置记录 Tab -->
        <el-tab-pane label="重置记录" name="resets">
          <div class="records-list" v-if="subscriptionResets && subscriptionResets.length > 0">
            <div
              v-for="reset in subscriptionResets"
              :key="reset.id"
              class="record-item reset-item"
            >
              <div class="record-header">
                <div class="record-type">
                  <el-icon class="type-icon reset-icon"><RefreshRight /></el-icon>
                  <span class="type-text">{{ getResetTypeText(reset.reset_type) }}</span>
                </div>
                <div class="reset-time">
                  {{ formatDateTime(reset.created_at) }}
                </div>
              </div>
              <div class="record-body">
                <div class="record-info-row">
                  <span class="info-label">订阅ID：</span>
                  <span class="info-value">{{ reset.subscription_id }}</span>
                </div>
                <div class="record-info-row">
                  <span class="info-label">操作者：</span>
                  <span class="info-value">{{ reset.reset_by || '系统' }}</span>
                </div>
                <div class="record-info-row" v-if="reset.reason">
                  <span class="info-label">原因：</span>
                  <span class="info-value">{{ reset.reason }}</span>
                </div>
                <div class="record-info-row">
                  <span class="info-label">设备数量：</span>
                  <span class="info-value">{{ reset.device_count_before }} → {{ reset.device_count_after }}</span>
                </div>
                <div class="subscription-url-box old-url">
                  <div class="url-label">旧订阅地址</div>
                  <div class="url-content">
                    <span class="url-text">{{ reset.old_subscription_url }}</span>
                    <el-button
                      type="primary"
                      size="small"
                      @click="copyToClipboard(reset.old_subscription_url)"
                      :icon="DocumentCopy"
                    >
                      复制
                    </el-button>
                  </div>
                </div>
                <div class="subscription-url-box new-url">
                  <div class="url-label">新订阅地址</div>
                  <div class="url-content">
                    <span class="url-text">{{ reset.new_subscription_url }}</span>
                    <el-button
                      type="success"
                      size="small"
                      @click="copyToClipboard(reset.new_subscription_url)"
                      :icon="DocumentCopy"
                    >
                      复制
                    </el-button>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <el-empty v-else description="暂无重置记录" :image-size="100" />
        </el-tab-pane>

        <!-- 设备记录 Tab -->
        <el-tab-pane label="设备记录" name="devices">
          <div class="records-list" v-if="uaRecords && uaRecords.length > 0">
            <div
              v-for="(record, index) in uaRecords"
              :key="index"
              class="record-item device-item"
            >
              <div class="record-header">
                <div class="record-type">
                  <el-icon class="type-icon device-icon"><Monitor /></el-icon>
                  <span class="type-text">{{ record.device_name || record.device_type || '未知设备' }}</span>
                </div>
                <div class="access-count">
                  访问 {{ record.access_count }} 次
                </div>
              </div>
              <div class="record-body">
                <div class="record-info-row">
                  <span class="info-label">设备类型：</span>
                  <span class="info-value">{{ record.device_type || '未知' }}</span>
                </div>
                <div class="record-info-row">
                  <span class="info-label">IP地址：</span>
                  <span class="info-value">{{ record.ip_address || '未知' }}</span>
                </div>
                <div class="record-info-row" v-if="record.location">
                  <span class="info-label">归属地：</span>
                  <span class="info-value">{{ record.location }}</span>
                </div>
                <div class="record-info-row">
                  <span class="info-label">首次访问：</span>
                  <span class="info-value">{{ formatDateTime(record.created_at) }}</span>
                </div>
                <div class="record-info-row">
                  <span class="info-label">最后访问：</span>
                  <span class="info-value">{{ formatDateTime(record.last_access) }}</span>
                </div>
                <div class="record-info-row full-width" v-if="record.user_agent">
                  <span class="info-label">User Agent：</span>
                  <span class="info-value">{{ record.user_agent }}</span>
                </div>
              </div>
            </div>
          </div>
          <el-empty v-else description="暂无设备记录" :image-size="100" />
        </el-tab-pane>

        <!-- 登录历史 Tab -->
        <el-tab-pane label="登录历史" name="login">
          <div class="records-list" v-if="loginHistory && loginHistory.length > 0">
            <div
              v-for="record in loginHistory"
              :key="record.id"
              class="record-item login-item"
            >
              <div class="record-header">
                <div class="record-type">
                  <el-icon class="type-icon login-icon"><User /></el-icon>
                  <span class="type-text">{{ record.login_status === 'success' ? '登录成功' : '登录失败' }}</span>
                </div>
                <el-tag :type="record.login_status === 'success' ? 'success' : 'danger'" size="small">
                  {{ record.login_status === 'success' ? '成功' : '失败' }}
                </el-tag>
              </div>
              <div class="record-body">
                <div class="record-info-row">
                  <span class="info-label">登录时间：</span>
                  <span class="info-value">{{ formatDateTime(record.login_time) }}</span>
                </div>
                <div class="record-info-row">
                  <span class="info-label">IP地址：</span>
                  <span class="info-value">{{ record.ip_address || '未知' }}</span>
                </div>
                <div class="record-info-row" v-if="record.location">
                  <span class="info-label">归属地：</span>
                  <span class="info-value">{{ record.location }}</span>
                </div>
                <div class="record-info-row" v-if="record.failure_reason">
                  <span class="info-label">失败原因：</span>
                  <span class="info-value failure-reason">{{ record.failure_reason }}</span>
                </div>
                <div class="record-info-row full-width" v-if="record.user_agent">
                  <span class="info-label">User Agent：</span>
                  <span class="info-value">{{ record.user_agent }}</span>
                </div>
              </div>
            </div>
          </div>
          <el-empty v-else description="暂无登录历史" :image-size="100" />
        </el-tab-pane>

        <!-- 专线节点 Tab -->
        <el-tab-pane label="专线节点" name="custom_nodes">
          <div class="custom-nodes-section">
            <div class="custom-nodes-actions">
              <el-button type="primary" @click="showAssignDialog = true">
                <el-icon><Plus /></el-icon>
                分配专线节点
              </el-button>
            </div>
            <div class="table-responsive" v-if="customNodes && customNodes.length">
              <el-table :data="customNodes" size="small" style="width: 100%">
                <el-table-column prop="id" label="节点ID" width="80" />
                <el-table-column prop="name" label="节点名称" min-width="150" />
                <el-table-column prop="protocol" label="协议" width="100" />
                <el-table-column prop="domain" label="域名" min-width="150" />
                <el-table-column prop="port" label="端口" width="80" />
                <el-table-column label="状态" width="100">
                  <template #default="scope">
                    <el-tag :type="scope.row.is_active ? 'success' : 'danger'" size="small">
                      {{ scope.row.is_active ? '活跃' : '禁用' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="120" fixed="right">
                  <template #default="scope">
                    <el-button
                      type="danger"
                      size="small"
                      @click="unassignNode(scope.row.id)"
                    >
                      取消分配
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
            </div>
            <el-empty v-else description="暂无分配的专线节点" :image-size="100" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- 分配节点对话框 -->
    <el-dialog
      v-model="showAssignDialog"
      title="分配专线节点"
      :width="isMobile ? '95%' : '600px'"
      :close-on-click-modal="false"
    >
      <div class="node-search-section">
        <div class="search-input-group">
          <el-input
            v-model="nodeSearchKeyword"
            placeholder="输入节点名称或域名搜索"
            clearable
            @clear="handleNodeSearchClear"
            @keyup.enter="handleNodeSearch"
          >
            <template #append>
              <el-button
                :icon="Search"
                @click="handleNodeSearch"
                :loading="loadingNodes"
              >
                搜索
              </el-button>
            </template>
          </el-input>
        </div>
        <div class="search-result-tip" v-if="searchedNodes.length > 0">
          找到 {{ searchedNodes.length }} 个可分配的节点
        </div>
        <div class="search-result-tip empty" v-else-if="nodeSearchKeyword && !loadingNodes">
          未找到匹配的节点
        </div>
      </div>

      <el-form label-width="100px">
        <el-form-item label="选择节点">
          <el-select
            v-model="selectedNodeId"
            placeholder="请先搜索节点"
            style="width: 100%"
            filterable
            :disabled="searchedNodes.length === 0"
          >
            <el-option
              v-for="node in searchedNodes"
              :key="node.id"
              :label="`${node.name} (${node.domain}:${node.port})`"
              :value="node.id"
            />
          </el-select>
        </el-form-item>
        <div class="form-tip">
          提示：请先在搜索框中输入节点名称或域名进行搜索，然后从搜索结果中选择要分配的节点。
        </div>
      </el-form>

      <template #footer>
        <el-button @click="showAssignDialog = false">取消</el-button>
        <el-button
          type="primary"
          @click="assignNode"
          :loading="assigning"
          :disabled="!selectedNodeId"
        >
          确定分配
        </el-button>
      </template>
    </el-dialog>
  </el-drawer>
</template>

<script>
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Connection,
  Wallet,
  Plus,
  ShoppingCart,
  Search,
  RefreshRight,
  Monitor,
  User,
  DocumentCopy
} from '@element-plus/icons-vue'
import { formatDate as formatDateUtil } from '@/utils/date'
import { adminAPI } from '@/utils/api'

export default {
  name: 'UserDetailDrawer',
  components: {
    Connection,
    Wallet,
    Plus,
    ShoppingCart,
    Search,
    RefreshRight,
    Monitor,
    User,
    DocumentCopy
  },
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    user: {
      type: Object,
      default: null
    },
    isMobile: {
      type: Boolean,
      default: false
    },
    initialTab: {
      type: String,
      default: 'basic'
    }
  },
  emits: ['update:visible'],
  setup(props, { emit }) {
    const activeTab = ref('basic')
    const customNodes = ref([])
    const availableNodes = ref([])
    const searchedNodes = ref([])
    const nodeSearchKeyword = ref('')
    const showAssignDialog = ref(false)
    const selectedNodeId = ref(null)
    const assigning = ref(false)
    const loadingNodes = ref(false)

    const rechargeRecords = computed(() => {
      if (!props.user) return []
      return props.user.recharge_records || []
    })

    const orderRecords = computed(() => {
      if (!props.user) return []
      return props.user.orders || []
    })

    const subscriptionResets = computed(() => {
      if (!props.user) return []
      return props.user.subscription_resets || []
    })

    const uaRecords = computed(() => {
      if (!props.user) return []
      return props.user.ua_records || []
    })

    const loginHistory = computed(() => {
      if (!props.user) return []
      return props.user.login_history || []
    })

    if (process.env.NODE_ENV === 'development') {
      watch(() => props.user, (newUser) => {
      }, { immediate: true, deep: true })
    }

    const loadUserCustomNodes = async (userId) => {
      try {
        const response = await adminAPI.getUserCustomNodes(userId)
        if (response.data && response.data.success) {
          customNodes.value = response.data.data || []
        }
      } catch (error) {
        console.error('加载用户专线节点失败:', error)
        customNodes.value = []
      }
    }

    const loadAvailableNodes = async () => {
      searchedNodes.value = []
      nodeSearchKeyword.value = ''
    }

    const handleNodeSearch = async () => {
      const keyword = nodeSearchKeyword.value?.trim()
      if (!keyword) {
        ElMessage.warning('请输入节点名称或域名进行搜索')
        return
      }
      if (keyword.length < 2) {
        ElMessage.warning('搜索关键词至少需要2个字符')
        return
      }
      loadingNodes.value = true
      try {
        const params = {
          is_active: 'true',
          page: 1,
          size: 200,
          search: keyword
        }
        const response = await adminAPI.getCustomNodes(params)
        if (response.data && response.data.success) {
          const allNodes = response.data.data?.data || response.data.data || []
          const assignedIds = customNodes.value.map(n => n.id)
          const filteredNodes = allNodes.filter(n => !assignedIds.includes(n.id))
          searchedNodes.value = filteredNodes
          if (filteredNodes.length === 0) {
            ElMessage.info('未找到匹配的节点，请检查输入的节点名称或域名')
          } else {
            ElMessage.success(`找到 ${filteredNodes.length} 个匹配的节点`)
          }
        } else {
          searchedNodes.value = []
          ElMessage.error(response.data?.message || '搜索失败')
        }
      } catch (error) {
        console.error('搜索节点失败:', error)
        ElMessage.error('搜索节点失败: ' + (error.response?.data?.message || error.message))
        searchedNodes.value = []
      } finally {
        loadingNodes.value = false
      }
    }

    const handleNodeSearchClear = () => {
      nodeSearchKeyword.value = ''
      searchedNodes.value = []
      selectedNodeId.value = null
    }

    watch(() => props.initialTab, (newVal) => {
      if (newVal) activeTab.value = newVal
    }, { immediate: true })

    watch(() => props.user, async (newUser) => {
      if (newUser && newUser.user_info?.id) {
        await loadUserCustomNodes(newUser.user_info.id)
      } else if (newUser && newUser.id) {
        await loadUserCustomNodes(newUser.id)
      }
    }, { immediate: true })

    watch(() => props.visible, async (visible) => {
      if (visible) {
        await loadAvailableNodes()
      }
    })

    watch(() => showAssignDialog.value, async (visible) => {
      if (visible) {
        await loadAvailableNodes()
        selectedNodeId.value = null
      }
    })

    const assignNode = async () => {
      if (!selectedNodeId.value) {
        ElMessage.warning('请选择要分配的节点')
        return
      }
      const userId = props.user?.user_info?.id || props.user?.id
      if (!userId) {
        ElMessage.error('用户ID不存在')
        return
      }
      assigning.value = true
      try {
        const response = await adminAPI.assignCustomNodeToUser(userId, selectedNodeId.value)
        if (response.data && response.data.success) {
          ElMessage.success('分配成功')
          showAssignDialog.value = false
          selectedNodeId.value = null
          await loadUserCustomNodes(userId)
          await loadAvailableNodes()
        } else {
          ElMessage.error(response.data?.message || '分配失败')
        }
      } catch (error) {
        ElMessage.error('分配失败: ' + (error.response?.data?.message || error.message))
      } finally {
        assigning.value = false
      }
    }

    const unassignNode = async (nodeId) => {
      const userId = props.user?.user_info?.id || props.user?.id
      if (!userId) {
        ElMessage.error('用户ID不存在')
        return
      }
      try {
        const response = await adminAPI.unassignCustomNodeFromUser(userId, nodeId)
        if (response.data && response.data.success) {
          ElMessage.success('取消分配成功')
          await loadUserCustomNodes(userId)
          await loadAvailableNodes()
        } else {
          ElMessage.error(response.data?.message || '取消分配失败')
        }
      } catch (error) {
        ElMessage.error('取消分配失败: ' + (error.response?.data?.message || error.message))
      }
    }

    const formatDate = (date) => formatDateUtil(date) || ''

    const formatDateTime = (dateTime) => {
      if (!dateTime) return '未知'
      if (typeof dateTime === 'string') {
        if (dateTime.includes('T') || dateTime.includes('+')) {
          try {
            const date = new Date(dateTime)
            if (!isNaN(date.getTime())) {
              return date.toLocaleString('zh-CN', {
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit'
              }).replace(/\//g, '-')
            }
          } catch (e) {
            return formatDateUtil(dateTime) || dateTime
          }
        }
        if (dateTime.match(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/)) {
          return dateTime
        }
        return formatDateUtil(dateTime) || dateTime
      }
      if (dateTime instanceof Date) {
        return dateTime.toLocaleString('zh-CN', {
          year: 'numeric',
          month: '2-digit',
          day: '2-digit',
          hour: '2-digit',
          minute: '2-digit',
          second: '2-digit'
        }).replace(/\//g, '-')
      }
      return formatDateUtil(dateTime) || '未知'
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

    const getPaymentMethodText = (method) => {
      let methodStr = method
      if (method && typeof method === 'object') {
        if (method.String) {
          methodStr = method.String
        } else if (method.payment_method) {
          methodStr = method.payment_method
        } else {
          const values = Object.values(method).filter(v => typeof v === 'string' && v.length > 0)
          if (values.length > 0) {
            methodStr = values[0]
          } else {
            methodStr = '未知'
          }
        }
      }
      const methodMap = {
        alipay: '支付宝',
        wechat: '微信支付',
        balance: '余额支付',
        mixed: '余额+支付宝'
      }
      return methodMap[methodStr] || methodStr || '未知'
    }

    const getResetTypeText = (resetType) => {
      const typeMap = {
        'admin_reset': '管理员重置',
        'user_reset': '用户重置',
        'admin_batch_reset': '批量重置'
      }
      return typeMap[resetType] || resetType
    }

    const copyToClipboard = async (text) => {
      try {
        await navigator.clipboard.writeText(text)
        ElMessage.success('已复制到剪贴板')
      } catch (error) {
        const textarea = document.createElement('textarea')
        textarea.value = text
        textarea.style.position = 'fixed'
        textarea.style.opacity = '0'
        document.body.appendChild(textarea)
        textarea.select()
        try {
          document.execCommand('copy')
          ElMessage.success('已复制到剪贴板')
        } catch (err) {
          ElMessage.error('复制失败，请手动复制')
        }
        document.body.removeChild(textarea)
      }
    }

    return {
      activeTab,
      customNodes,
      availableNodes,
      searchedNodes,
      nodeSearchKeyword,
      showAssignDialog,
      selectedNodeId,
      assigning,
      loadingNodes,
      assignNode,
      unassignNode,
      loadAvailableNodes,
      handleNodeSearch,
      handleNodeSearchClear,
      formatDate,
      formatDateTime,
      getStatusType,
      getStatusText,
      getPaymentMethodText,
      getResetTypeText,
      copyToClipboard,
      rechargeRecords,
      orderRecords,
      subscriptionResets,
      uaRecords,
      loginHistory,
      Search,
      DocumentCopy
    }
  }
}
</script>

<style scoped lang="scss">
.user-detail-drawer {
  :deep(.el-drawer__body) {
    padding: 0;
    overflow-y: auto;
  }
}

.drawer-content {
  padding: 20px;
}

.user-tabs {
  :deep(.el-tabs__content) {
    padding-top: 15px;
  }
}

.balance-highlight {
  font-size: 16px;
  font-weight: 600;
  color: #67c23a;
}

.user-stats {
  margin-top: 20px;
  padding: 15px;
  background: #f5f7fa;
  border-radius: 8px;

  h4 {
    margin: 0 0 15px 0;
    font-size: 14px;
    color: #303133;
  }
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 15px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;

  .el-icon {
    font-size: 18px;
  }
}

.records-list {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.record-item {
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 15px;
  background: #fff;
  transition: all 0.3s;

  &:hover {
    box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  }

  &.recharge-item {
    border-left: 3px solid #67c23a;
  }

  &.consumption-item {
    border-left: 3px solid #f56c6c;
  }

  &.reset-item {
    border-left: 3px solid #409eff;
  }

  &.device-item {
    border-left: 3px solid #e6a23c;
  }

  &.login-item {
    border-left: 3px solid #909399;
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
    font-size: 20px;

    &.recharge-icon {
      color: #67c23a;
    }

    &.consumption-icon {
      color: #f56c6c;
    }

    &.reset-icon {
      color: #409eff;
    }

    &.device-icon {
      color: #e6a23c;
    }

    &.login-icon {
      color: #909399;
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

.reset-time,
.access-count {
  font-size: 13px;
  color: #909399;
  font-weight: 500;
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

  &.full-width {
    grid-column: 1 / -1;
  }

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

    &.failure-reason {
      color: #f56c6c;
      font-weight: 500;
    }
  }
}

.subscription-url-box {
  margin-top: 12px;
  padding: 12px;
  border-radius: 6px;
  grid-column: 1 / -1;

  &.old-url {
    background: #fef0f0;
    border: 1px solid #fbc4c4;
  }

  &.new-url {
    background: #f0f9ff;
    border: 1px solid #b3d8ff;
  }

  .url-label {
    font-size: 12px;
    font-weight: 600;
    color: #606266;
    margin-bottom: 8px;
  }

  .url-content {
    display: flex;
    align-items: center;
    gap: 10px;

    .url-text {
      flex: 1;
      font-size: 12px;
      color: #303133;
      word-break: break-all;
      font-family: monospace;
    }

    .el-button {
      flex-shrink: 0;
    }
  }
}

.table-responsive {
  width: 100%;
  overflow-x: auto;

  @media (max-width: 768px) {
    .el-table {
      font-size: 12px;
    }
  }
}

.custom-nodes-section {
  .custom-nodes-actions {
    margin-bottom: 15px;
  }
}

.node-search-section {
  margin-bottom: 10px;

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
</style>
