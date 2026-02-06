<template>
  <div class="list-container admin-orders">
    <el-card class="list-card" shadow="never">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <span class="title-text">订单管理</span>
            <el-tag v-if="total" type="info" round size="small" class="count-tag">{{ total }}</el-tag>
          </div>

          <!-- 桌面端顶部操作 -->
          <div class="header-actions" v-if="!isMobile">
            <el-button type="success" @click="exportOrders">
              <el-icon><Download /></el-icon>导出订单
            </el-button>
            <el-button type="info" @click="showStatisticsDialog = true">
              <el-icon><DataAnalysis /></el-icon>订单统计
            </el-button>
          </div>

          <!-- 移动端顶部操作 -->
          <div class="header-actions mobile" v-else>
            <el-dropdown trigger="click" @command="handleCommand">
              <el-button circle size="small">
                <el-icon><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="refresh" :icon="Refresh">刷新列表</el-dropdown-item>
                  <el-dropdown-item command="export" :icon="Download">导出订单</el-dropdown-item>
                  <el-dropdown-item command="stats" :icon="DataAnalysis">查看统计</el-dropdown-item>
                  <el-dropdown-item command="bulk_paid" :disabled="!selectedOrders.length" :icon="Check">批量标记已付</el-dropdown-item>
                  <el-dropdown-item command="bulk_cancel" :disabled="!selectedOrders.length" :icon="Close">批量取消</el-dropdown-item>
                  <el-dropdown-item command="bulk_delete" :disabled="!selectedOrders.length" :icon="Delete" divided style="color: var(--el-color-danger)">批量删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </template>

      <!-- 统一的响应式筛选栏 -->
      <div class="filter-wrapper">
        <div class="filter-grid">
          <div class="tab-select-wrapper">
            <el-radio-group v-model="activeTab" @change="handleTabChange" size="default">
              <el-radio-button label="orders">订单记录</el-radio-button>
              <el-radio-button label="recharges">充值记录</el-radio-button>
            </el-radio-group>
          </div>
          <el-select v-model="searchForm.status" placeholder="所有状态" clearable @change="searchOrders" class="status-select">
            <el-option label="全部" value="" />
            <el-option label="待支付" value="pending" />
            <el-option label="已支付" value="paid" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
          <div class="search-box">
            <el-input 
              v-model="searchForm.keyword" 
              placeholder="搜索订单号/邮箱/用户名..." 
              clearable
              @keyup.enter="searchOrders"
            >
              <template #prefix><el-icon><Search /></el-icon></template>
            </el-input>
          </div>
        </div>
      </div>

      <!-- 批量操作栏 (桌面端显示，移动端在下拉菜单) -->
      <div v-if="selectedOrders.length > 0 && !isMobile" class="batch-actions-bar">
        <span class="batch-tip">已选择 {{ selectedOrders.length }} 项</span>
        <div class="batch-btns">
          <el-button type="success" link @click="bulkMarkAsPaid" :loading="bulkLoading">批量已付</el-button>
          <el-divider direction="vertical" />
          <el-button type="warning" link @click="bulkCancel" :loading="bulkLoading">批量取消</el-button>
          <el-divider direction="vertical" />
          <el-button type="danger" link @click="bulkDelete" :loading="bulkLoading">批量删除</el-button>
        </div>
      </div>

      <!-- 内容展示区 -->
      <div class="content-view" v-loading="loading">
        <!-- 桌面端表格 -->
        <el-table 
          v-if="!isMobile"
          :data="activeTab === 'orders' ? allRecords : recharges" 
          style="width: 100%" 
          stripe
          @selection-change="handleSelectionChange"
          class="desktop-table"
        >
          <el-table-column type="selection" width="50" v-if="activeTab === 'orders'" />
          <el-table-column prop="order_no" label="订单号" min-width="160" show-overflow-tooltip />
          <el-table-column label="用户" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">
              {{ row.user?.email || row.user?.username || '-' }}
            </template>
          </el-table-column>
          <el-table-column :label="activeTab === 'orders' ? '内容' : '类型'" min-width="140">
            <template #default="{ row }">
              <el-tag v-if="row.record_type === 'recharge' || activeTab === 'recharges'" type="success" size="small" effect="plain">账户充值</el-tag>
              <span v-else>{{ row.package_name }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="amount" label="金额" width="120">
            <template #default="{ row }">
              <span :class="isPositiveAmount(row) ? 'amount-plus' : 'amount-normal'">
                {{ isPositiveAmount(row) ? '+' : '' }}¥{{ formatMoney(row.amount) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)" size="small" effect="light">
                {{ getStatusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="160">
            <template #default="{ row }">
              <span class="text-xs">{{ formatDateTime(row.created_at) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="180" fixed="right" v-if="activeTab === 'orders'">
            <template #default="{ row }">
              <div class="table-actions" v-if="row.record_type === 'order'">
                <el-button size="small" @click="viewOrder(row)">查看</el-button>
                <el-dropdown trigger="click" style="margin-left: 8px">
                  <el-button size="small" icon="MoreFilled" circle />
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item v-if="row.status === 'pending'" @click="markAsPaid(row)" :icon="Check">标记已付</el-dropdown-item>
                      <el-dropdown-item v-if="row.status === 'pending'" @click="cancelOrder(row)" :icon="Close">取消订单</el-dropdown-item>
                      <el-dropdown-item @click="deleteOrder(row)" :icon="Delete" style="color: var(--el-color-danger)" divided>删除</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <!-- 移动端卡片列表 -->
        <div v-else class="mobile-list">
          <!-- 全选控制栏 -->
          <div class="mobile-selection-bar" v-if="currentList.length > 0 && activeTab === 'orders'">
            <el-checkbox 
              v-model="isAllSelected" 
              :indeterminate="isIndeterminate" 
              @change="toggleMobileSelectAll"
            >全选 ({{ selectedOrders.length }})</el-checkbox>
          </div>

          <div v-for="item in currentList" :key="item.id || item.order_no" class="order-card">
            <div class="card-header-row">
              <el-checkbox 
                v-if="activeTab === 'orders'"
                :model-value="isSelected(item)" 
                @change="(val) => handleMobileSelect(item, val)"
                class="card-checkbox" 
              />
              <div class="order-no">{{ item.order_no }}</div>
              <el-tag size="small" :type="getStatusType(item.status)" effect="light">{{ getStatusText(item.status) }}</el-tag>
            </div>

            <div class="card-body" @click="viewOrder(item)">
              <div class="info-row">
                <span class="label">用户</span>
                <span class="value">{{ item.user?.email || '-' }}</span>
              </div>
              <div class="info-row">
                <span class="label">内容</span>
                <span class="value">
                   <el-tag v-if="item.record_type === 'recharge' || activeTab === 'recharges'" type="success" size="small" effect="plain">充值</el-tag>
                   <span v-else>{{ item.package_name }}</span>
                </span>
              </div>
              <div class="info-row">
                <span class="label">金额</span>
                <span class="value amount-value" :class="isPositiveAmount(item) ? 'amount-plus' : ''">
                  {{ isPositiveAmount(item) ? '+' : '' }}¥{{ formatMoney(item.amount) }}
                </span>
              </div>
              <div class="info-row">
                <span class="label">时间</span>
                <span class="value text-xs">{{ formatDateTime(item.created_at) }}</span>
              </div>
            </div>

            <div class="card-footer" v-if="activeTab === 'orders' && item.record_type === 'order'">
               <el-button size="small" text bg @click="viewOrder(item)">详情</el-button>
               <el-button v-if="item.status === 'pending'" size="small" text bg type="success" @click="markAsPaid(item)">已付</el-button>
               <el-dropdown trigger="click">
                  <el-button size="small" text bg>更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item v-if="item.status === 'pending'" @click="cancelOrder(item)" :icon="Close">取消</el-dropdown-item>
                      <el-dropdown-item @click="deleteOrder(item)" :icon="Delete" style="color: var(--el-color-danger)">删除</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
            </div>
          </div>
          <el-empty v-if="currentList.length === 0" description="暂无记录" />
        </div>
      </div>

      <!-- 分页器 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="activeTab === 'recharges' ? rechargeTotal : total"
          :layout="isMobile ? 'prev, pager, next' : 'total, sizes, prev, pager, next, jumper'"
          :pager-count="isMobile ? 5 : 7"
          background
          @current-change="handleCurrentChange"
          @size-change="handleSizeChange"
        />
      </div>
    </el-card>

    <!-- 订单详情弹窗 -->
    <el-dialog 
      v-model="showOrderDialog" 
      title="订单详情" 
      :width="isMobile ? '95%' : '600px'"
      class="responsive-dialog"
      destroy-on-close
    >
      <div class="order-detail-container">
        <div class="detail-group">
          <div class="detail-item full">
            <span class="label">订单号</span>
            <span class="value copy-able">{{ selectedOrder.order_no }}</span>
          </div>
          <div class="detail-item">
            <span class="label">用户</span>
            <span class="value">{{ selectedOrder.user?.email || '-' }}</span>
          </div>
          <div class="detail-item">
            <span class="label">金额</span>
            <span class="value price">¥{{ formatMoney(selectedOrder.amount) }}</span>
          </div>
          <div class="detail-item">
            <span class="label">状态</span>
            <span class="value">
              <el-tag :type="getStatusType(selectedOrder.status)">{{ getStatusText(selectedOrder.status) }}</el-tag>
            </span>
          </div>
          <div class="detail-item">
             <span class="label">支付方式</span>
             <span class="value">{{ selectedOrder.payment_method || '-' }}</span>
          </div>
          <div class="detail-item full">
            <span class="label">创建时间</span>
            <span class="value">{{ formatDateTime(selectedOrder.created_at) }}</span>
          </div>
          <div class="detail-item full" v-if="selectedOrder.payment_time">
            <span class="label">支付时间</span>
            <span class="value">{{ formatDateTime(selectedOrder.payment_time) }}</span>
          </div>
        </div>
        
        <div v-if="selectedOrder.payment_proof" class="proof-section">
          <div class="section-title">支付凭证</div>
          <div class="proof-img-box">
             <el-image 
               :src="selectedOrder.payment_proof" 
               :preview-src-list="[selectedOrder.payment_proof]"
               fit="contain"
             />
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 统计弹窗 -->
    <el-dialog v-model="showStatisticsDialog" title="订单统计" :width="isMobile ? '95%' : '500px'" class="responsive-dialog">
      <div class="stats-grid">
        <div class="stat-item">
          <div class="num">{{ statistics.totalOrders }}</div>
          <div class="txt">总订单</div>
        </div>
        <div class="stat-item">
          <div class="num warning">{{ statistics.pendingOrders }}</div>
          <div class="txt">待支付</div>
        </div>
        <div class="stat-item">
          <div class="num success">{{ statistics.paidOrders }}</div>
          <div class="txt">已支付</div>
        </div>
        <div class="stat-item">
          <div class="num primary">¥{{ formatMoney(statistics.totalRevenue) }}</div>
          <div class="txt">总收入</div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Download, DataAnalysis, View, Check, Close, Search, 
  Refresh, Delete, MoreFilled, ArrowDown
} from '@element-plus/icons-vue'
import { useApi, adminAPI } from '@/utils/api'
import { formatDateTime as formatDateTimeUtil } from '@/utils/date'

export default {
  name: 'AdminOrders',
  components: {
    Download, DataAnalysis, View, Check, Close, Search, 
    Refresh, Delete, MoreFilled, ArrowDown
  },
  setup() {
    const route = useRoute()
    const api = useApi()
    
    // 状态
    const isMobile = ref(false)
    const loading = ref(false)
    const bulkLoading = ref(false)
    const activeTab = ref('orders')
    
    // 数据
    const orders = ref([]) // 纯订单
    const recharges = ref([]) // 充值记录
    const allRecords = ref([]) // 混合记录
    const selectedOrders = ref([])
    const selectedOrder = ref({})
    
    // 分页
    const currentPage = ref(1)
    const pageSize = ref(20)
    const total = ref(0)
    const rechargeTotal = ref(0)

    // 搜索与筛选
    const searchForm = reactive({ keyword: '', status: '' })

    // 弹窗
    const showOrderDialog = ref(false)
    const showStatisticsDialog = ref(false)
    const statistics = reactive({ totalOrders: 0, pendingOrders: 0, paidOrders: 0, totalRevenue: 0 })

    const checkMobile = () => { isMobile.value = window.innerWidth <= 768 }

    // --- 加载逻辑 ---
    const loadData = async () => {
      loading.value = true
      try {
        if (activeTab.value === 'recharges') {
          await loadRecharges()
        } else {
          await loadOrders()
        }
      } finally {
        loading.value = false
      }
    }

    const loadOrders = async () => {
      try {
        const params = {
          skip: (currentPage.value - 1) * pageSize.value,
          limit: pageSize.value,
          search: searchForm.keyword,
          status: searchForm.status,
          include_recharges: 'true'
        }
        // 清理空参数
        Object.keys(params).forEach(k => !params[k] && delete params[k])
        
        const res = await api.get('/admin/orders', { params })
        const list = res.data.data?.orders || []
        
        allRecords.value = list
        orders.value = list.filter(r => r.record_type === 'order')
        recharges.value = list.filter(r => r.record_type === 'recharge') // 这里的recharges仅用于混合显示时的过滤，不覆盖recharges tab的数据
        total.value = res.data.data?.total || 0
      } catch (e) {
        ElMessage.error('加载订单失败')
        allRecords.value = []
      }
    }

    const loadRecharges = async () => {
      try {
        const params = {
          page: currentPage.value,
          size: pageSize.value,
          keyword: searchForm.keyword,
          status: searchForm.status
        }
        const res = await adminAPI.getAdminRechargeRecords(params)
        // 兼容多种返回格式
        const data = res.data.data || res.data
        const list = Array.isArray(data) ? data : (data.recharges || [])
        recharges.value = list
        rechargeTotal.value = Number(data.total) || list.length
      } catch (e) {
        ElMessage.error('加载充值记录失败')
        recharges.value = []
      }
    }

    // --- 交互逻辑 ---
    const handleTabChange = () => {
      currentPage.value = 1
      selectedOrders.value = [] // 切换标签清空选中
      loadData()
    }
    const searchOrders = () => {
      currentPage.value = 1
      loadData()
    }
    const handleSizeChange = (val) => {
      pageSize.value = val
      loadData()
    }
    const handleCurrentChange = (val) => {
      currentPage.value = val
      loadData()
    }

    // --- 移动端选择 ---
    const currentList = computed(() => activeTab.value === 'orders' ? allRecords.value : recharges.value)
    
    const handleMobileSelect = (item, checked) => {
      if (checked) {
        if (!selectedOrders.value.find(i => i.id === item.id)) selectedOrders.value.push(item)
      } else {
        selectedOrders.value = selectedOrders.value.filter(i => i.id !== item.id)
      }
    }
    const isSelected = (item) => selectedOrders.value.some(i => i.id === item.id)
    const isAllSelected = computed({
      get: () => currentList.value.length > 0 && selectedOrders.value.length === currentList.value.length,
      set: (val) => toggleMobileSelectAll(val)
    })
    const isIndeterminate = computed(() => selectedOrders.value.length > 0 && selectedOrders.value.length < currentList.value.length)
    const toggleMobileSelectAll = (val) => selectedOrders.value = val ? [...currentList.value] : []
    const handleSelectionChange = (val) => selectedOrders.value = val

    // --- 订单操作 ---
    const handleCommand = (cmd) => {
      if (cmd === 'refresh') loadData()
      if (cmd === 'export') exportOrders()
      if (cmd === 'stats') showStatisticsDialog.value = true
      if (cmd === 'bulk_paid') bulkMarkAsPaid()
      if (cmd === 'bulk_cancel') bulkCancel()
      if (cmd === 'bulk_delete') bulkDelete()
    }

    const viewOrder = (order) => {
      selectedOrder.value = order
      showOrderDialog.value = true
    }

    const markAsPaid = async (order) => {
      try {
        await ElMessageBox.confirm('确认标记为已支付?', '提示')
        await api.put(`/admin/orders/${order.id}`, { status: 'paid' })
        ElMessage.success('操作成功')
        loadData()
      } catch {}
    }

    const cancelOrder = async (order) => {
      try {
        await ElMessageBox.confirm('确认取消此订单?', '警告', { type: 'warning' })
        await api.put(`/admin/orders/${order.id}`, { status: 'cancelled' })
        ElMessage.success('订单已取消')
        loadData()
      } catch {}
    }

    const deleteOrder = async (order) => {
      try {
        await ElMessageBox.confirm('删除后不可恢复，确认删除?', '危险操作', { type: 'error' })
        await api.delete(`/admin/orders/${order.id}`)
        ElMessage.success('已删除')
        loadData()
      } catch {}
    }

    // --- 批量操作 ---
    const bulkAction = async (actionName, apiPath, confirmMsg) => {
      if (!selectedOrders.value.length) return
      try {
        await ElMessageBox.confirm(confirmMsg, '批量操作', { type: 'warning' })
        bulkLoading.value = true
        const ids = selectedOrders.value.map(o => o.id)
        await api.post(apiPath, { order_ids: ids })
        ElMessage.success(`${actionName}成功`)
        selectedOrders.value = []
        loadData()
      } catch (e) {
        if (e !== 'cancel') ElMessage.error(`${actionName}失败`)
      } finally {
        bulkLoading.value = false
      }
    }

    const bulkMarkAsPaid = () => bulkAction('批量标记已付', '/admin/orders/bulk-mark-paid', `确认将 ${selectedOrders.value.length} 个订单标记为已付?`)
    const bulkCancel = () => bulkAction('批量取消', '/admin/orders/bulk-cancel', `确认取消 ${selectedOrders.value.length} 个订单?`)
    const bulkDelete = () => bulkAction('批量删除', '/admin/orders/batch-delete', `确认删除 ${selectedOrders.value.length} 个订单? 此操作不可恢复!`)

    const exportOrders = async () => {
      try {
        const params = { search: searchForm.keyword, status: searchForm.status }
        const res = await api.get('/admin/orders/export', { responseType: 'blob', params })
        const url = window.URL.createObjectURL(new Blob([res.data]))
        const link = document.createElement('a')
        link.href = url
        link.setAttribute('download', `orders_${Date.now()}.csv`)
        link.click()
        ElMessage.success('导出成功')
      } catch { ElMessage.error('导出失败') }
    }

    const loadStatistics = async () => {
      try {
        const res = await api.get('/admin/orders/statistics')
        if (res.data.success) Object.assign(statistics, res.data.data)
      } catch {}
    }

    // --- 辅助函数 ---
    const formatMoney = (val) => Number(val || 0).toFixed(2)
    const formatDateTime = (t) => formatDateTimeUtil(t) || '-'
    const getStatusType = (s) => ({ pending: 'warning', paid: 'success', cancelled: 'info' }[s] || 'info')
    const getStatusText = (s) => ({ pending: '待支付', paid: '已支付', cancelled: '已取消' }[s] || s)
    const isPositiveAmount = (row) => activeTab.value === 'recharges' || row.record_type === 'recharge'

    onMounted(() => {
      checkMobile()
      window.addEventListener('resize', checkMobile)
      if (route.query.search) searchForm.keyword = route.query.search
      loadData()
      loadStatistics()
    })
    
    onUnmounted(() => window.removeEventListener('resize', checkMobile))

    return {
      isMobile, loading, bulkLoading, activeTab,
      orders, recharges, allRecords, currentList,
      selectedOrders, selectedOrder,
      currentPage, pageSize, total, rechargeTotal,
      searchForm, statistics,
      showOrderDialog, showStatisticsDialog,
      loadData, handleTabChange, searchOrders, handleSizeChange, handleCurrentChange,
      handleCommand, viewOrder, markAsPaid, cancelOrder, deleteOrder,
      handleSelectionChange, handleMobileSelect, toggleMobileSelectAll,
      isAllSelected, isIndeterminate, isSelected,
      bulkMarkAsPaid, bulkCancel, bulkDelete, exportOrders,
      formatMoney, formatDateTime, getStatusType, getStatusText, isPositiveAmount,
      Download, DataAnalysis, View, Check, Close, Search, Refresh, Delete, MoreFilled, ArrowDown
    }
  }
}
</script>

<style scoped>
.admin-orders {
  max-width: 1400px;
  margin: 0 auto;
  padding: 16px;
}

@media (max-width: 768px) {
  .admin-orders {
    padding: 8px;
  }
}

.list-card {
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}

/* 头部 */
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.title-text {
  font-size: 16px;
  font-weight: 600;
  margin-right: 8px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

/* 筛选区 */
.filter-wrapper {
  background: var(--el-fill-color-light);
  padding: 16px;
  border-radius: 6px;
  margin-bottom: 16px;
}

.filter-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.tab-select-wrapper {
  margin-right: auto;
}

.status-select {
  width: 120px;
}

.search-box {
  min-width: 200px;
}

@media (max-width: 768px) {
  .filter-wrapper {
    padding: 12px;
  }
  .filter-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }
  .tab-select-wrapper {
    grid-column: 1 / -1;
    display: flex;
    justify-content: center;
    margin-bottom: 4px;
    width: 100%;
  }
  :deep(.el-radio-group) {
    display: flex;
    width: 100%;
  }
  :deep(.el-radio-button) {
    flex: 1;
  }
  :deep(.el-radio-button__inner) {
    width: 100%;
  }
  .status-select {
    width: 100%;
  }
  .search-box {
    grid-column: 1 / -1;
  }
}

/* 批量操作条 */
.batch-actions-bar {
  display: flex;
  align-items: center;
  background: var(--el-color-primary-light-9);
  padding: 8px 16px;
  border-radius: 4px;
  margin-bottom: 16px;
}

.batch-tip {
  font-size: 13px;
  color: var(--el-color-primary);
  margin-right: auto;
}

/* 桌面端表格 */
.amount-plus {
  color: #67c23a;
  font-weight: 600;
}
.amount-normal {
  font-weight: 500;
}
.text-xs {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

/* 移动端卡片列表 */
.mobile-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mobile-selection-bar {
  padding: 0 4px;
  margin-bottom: 4px;
}

.order-card {
  background: #fff;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 12px;
  box-shadow: 0 1px 2px rgba(0,0,0,0.02);
}

.card-header-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 8px;
  border-bottom: 1px dashed var(--el-border-color-lighter);
  margin-bottom: 8px;
}

.card-checkbox { margin-right: 0; }

.order-no {
  flex: 1;
  font-family: monospace;
  font-weight: 600;
  font-size: 13px;
  color: var(--el-text-color-primary);
}

.card-body {
  margin-bottom: 12px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 4px;
  font-size: 13px;
}

.info-row .label {
  color: var(--el-text-color-secondary);
}

.info-row .value {
  color: var(--el-text-color-regular);
  text-align: right;
  max-width: 70%;
  word-break: break-all;
}

.card-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}

/* 详情弹窗 */
.responsive-dialog :deep(.el-dialog__body) {
  padding: 15px 20px;
}

@media (max-width: 768px) {
  .responsive-dialog :deep(.el-dialog__body) {
    padding: 12px;
  }
}

.detail-group {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  background: var(--el-fill-color-light);
  padding: 16px;
  border-radius: 8px;
}

.detail-item {
  display: flex;
  flex-direction: column;
}

.detail-item.full {
  grid-column: span 2;
}

.detail-item .label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 2px;
}

.detail-item .value {
  font-size: 14px;
  font-weight: 500;
  word-break: break-all;
}

.detail-item .price {
  font-size: 16px;
  font-weight: bold;
}

.proof-section {
  margin-top: 20px;
}

.section-title {
  font-weight: 600;
  margin-bottom: 10px;
}

.proof-img-box {
  text-align: center;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 8px;
}

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.stat-item {
  background: var(--el-fill-color-light);
  padding: 16px;
  border-radius: 8px;
  text-align: center;
}

.stat-item .num {
  font-size: 20px;
  font-weight: bold;
  margin-bottom: 4px;
}

.stat-item .num.warning { color: var(--el-color-warning); }
.stat-item .num.success { color: var(--el-color-success); }
.stat-item .num.primary { color: var(--el-color-primary); }

.stat-item .txt {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.pagination-wrapper {
  margin-top: 20px;
  display: flex;
  justify-content: center;
}
</style>