<template>
  <el-dialog 
    :model-value="visible" 
    @update:model-value="$emit('update:visible', $event)"
    title="用户详情" 
    :width="isMobile ? '95%' : '1000px'"
    class="user-detail-dialog"
    :close-on-click-modal="false"
  >
    <div v-if="user" class="user-detail-content">
      <!-- 用户基本信息 -->
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
      
      <!-- 统计信息 -->
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
      
      <!-- 用户订阅列表 -->
      <div class="user-subscriptions" v-if="user.subscriptions && user.subscriptions.length">
        <h4 class="section-title">
          <el-icon><Connection /></el-icon>
          订阅列表
        </h4>
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
      
      <!-- 余额变动记录（充值记录 + 消费记录） -->
      <div class="balance-records-section" id="balance-records-section">
        <h4 class="section-title">
          <el-icon><Wallet /></el-icon>
          余额变动记录
        </h4>
        
        <!-- 使用标签页区分充值和消费 -->
        <el-tabs v-model="activeBalanceTab" class="balance-tabs">
          <!-- 充值记录 -->
          <el-tab-pane label="充值记录" name="recharge">
            <div class="records-list" v-if="user.recharge_records && user.recharge_records.length">
              <div 
                v-for="record in user.recharge_records" 
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
            <div class="records-list" v-if="user.orders && user.orders.length">
              <div 
                v-for="order in user.orders" 
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
      
      <!-- 专线节点分配 -->
      <div class="custom-nodes-section">
        <h4 class="section-title">
          <el-icon><Connection /></el-icon>
          专线节点分配
        </h4>
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
            <el-table-column label="操作" width="100">
              <template #default="scope">
                <el-button
                  type="danger"
                  size="small"
                  link
                  @click="unassignNode(scope.row.id)"
                >
                  取消分配
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
        <el-empty v-else description="该用户暂无分配的专线节点" :image-size="100" />
      </div>
      
      <!-- 最近活动 -->
      <div class="user-activities" v-if="user.recent_activities && user.recent_activities.length">
        <h4 class="section-title">
          <el-icon><Clock /></el-icon>
          最近活动
        </h4>
        <div class="table-responsive">
          <el-table :data="user.recent_activities" size="small" style="width: 100%">
            <el-table-column prop="activity_type" label="活动类型" width="120" />
            <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
            <el-table-column prop="ip_address" label="IP地址" width="120" />
            <el-table-column prop="created_at" label="时间" width="180" />
          </el-table>
        </div>
      </div>
    </div>

    <!-- 分配专线节点对话框 -->
    <el-dialog
      v-model="showAssignDialog"
      title="分配专线节点"
      width="600px"
    >
      <el-form label-width="120px">
        <el-form-item label="选择节点">
          <el-select
            v-model="selectedNodeId"
            placeholder="请选择要分配的专线节点"
            filterable
            style="width: 100%"
          >
            <el-option
              v-for="node in availableNodes"
              :key="node.id"
              :label="`${node.name} (${node.protocol})`"
              :value="node.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAssignDialog = false">取消</el-button>
        <el-button type="primary" @click="assignNode" :loading="assigning">确定</el-button>
      </template>
    </el-dialog>
  </el-dialog>
</template>

<script>
import { ref, watch, onMounted } from 'vue'
import { Wallet, ShoppingCart, Clock, Connection, Plus } from '@element-plus/icons-vue'
import { formatDate as formatDateUtil } from '@/utils/date'
import { ElMessage } from 'element-plus'
import { adminAPI } from '@/utils/api'

export default {
  name: 'UserDetailDialog',
  components: {
    Wallet, ShoppingCart, Clock, Connection, Plus
  },
  props: {
    visible: Boolean,
    user: Object,
    isMobile: Boolean,
    initialTab: {
      type: String,
      default: 'recharge'
    }
  },
  emits: ['update:visible'],
  setup(props) {
    const activeBalanceTab = ref('recharge')
    const customNodes = ref([])
    const availableNodes = ref([])
    const showAssignDialog = ref(false)
    const selectedNodeId = ref(null)
    const assigning = ref(false)

    watch(() => props.initialTab, (newVal) => {
      if (newVal) activeBalanceTab.value = newVal
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
      try {
        const response = await adminAPI.getCustomNodes({ is_active: 'true' })
        if (response.data && response.data.success) {
          const allNodes = response.data.data || []
          const assignedIds = customNodes.value.map(n => n.id)
          availableNodes.value = allNodes.filter(n => !assignedIds.includes(n.id))
        }
      } catch (error) {
        console.error('加载可用节点失败:', error)
        availableNodes.value = []
      }
    }

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

    return {
      activeBalanceTab,
      customNodes,
      availableNodes,
      showAssignDialog,
      selectedNodeId,
      assigning,
      assignNode,
      unassignNode,
      formatDate,
      getStatusType,
      getStatusText
    }
  }
}
</script>

<style scoped lang="scss">
// 提取自 Users.vue
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

.user-stats {
  margin: 20px 0;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
  
  :is(h4) {
    margin-bottom: 15px;
    color: #606266;
  }
  
  @media (max-width: 768px) {
    :deep(.el-col) {
      margin-bottom: 15px;
    }
  }
}

.user-subscriptions,
.user-orders,
.user-activities {
  margin-top: 20px;
}

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

.balance-highlight {
  font-size: 16px;
  font-weight: 700;
  color: #409eff;
}

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
  margin-top: 20px;
}

.custom-nodes-actions {
  margin-bottom: 15px;
}
</style>

