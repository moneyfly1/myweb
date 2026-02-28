<template>
  <div class="list-container admin-orders">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>订单列表</span>
          <!-- 电脑端操作栏保持不变 -->
          <div class="header-actions">
            <div class="bulk-actions" v-if="selectedOrders.length > 0">
              <span class="selected-count">已选择 {{ selectedOrders.length }} 个订单</span>
              <el-button type="success" size="small" @click="bulkMarkAsPaid" :disabled="bulkLoading">
                <el-icon><Check /></el-icon> 批量标记已付
              </el-button>
              <el-button type="warning" size="small" @click="bulkCancel" :disabled="bulkLoading">
                <el-icon><Close /></el-icon> 批量取消
              </el-button>
              <el-button type="danger" size="small" @click="bulkDelete" :disabled="bulkLoading">
                <el-icon><Delete /></el-icon> 批量删除
              </el-button>
            </div>
            <div class="normal-actions" v-else>
              <el-button type="success" @click="exportOrders">
                <el-icon><Download /></el-icon> 导出订单
              </el-button>
              <el-button type="info" @click="showStatisticsDialog = true">
                <el-icon><DataAnalysis /></el-icon> 订单统计
              </el-button>
            </div>
          </div>
        </div>
      </template>

      <!-- 手机端搜索/筛选栏 (优化布局) -->
      <div class="mobile-action-bar" v-if="isMobile">
        <div class="mobile-search-row">
          <el-input 
            v-model="searchForm.keyword" 
            placeholder="搜索订单..." 
            class="mobile-search-input"
            clearable
            @keyup.enter="searchOrders"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <el-button type="primary" class="mobile-search-btn" @click="searchOrders">
            搜索
          </el-button>
        </div>
        
        <div class="mobile-filter-row">
          <el-select v-model="searchForm.status" placeholder="状态筛选" @change="searchOrders" class="mobile-filter-select">
            <el-option label="全部状态" value="" />
            <el-option label="待支付" value="pending" />
            <el-option label="已支付" value="paid" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
          <el-button @click="resetSearch" icon="Refresh" circle class="mobile-reset-btn"></el-button>
        </div>
      </div>

      <!-- 电脑端搜索表单 (保持不变) -->
      <el-form :inline="true" :model="searchForm" class="search-form" v-else>
        <el-form-item label="搜索">
          <el-input 
            v-model="searchForm.keyword" 
            placeholder="输入订单号、时间戳、用户邮箱或用户名进行搜索"
            style="width: 350px;"
            clearable
            @keyup.enter="searchOrders"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchForm.status" placeholder="选择状态" style="width: 120px;">
            <el-option label="全部" value="" />
            <el-option label="待支付" value="pending" />
            <el-option label="已支付" value="paid" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="searchOrders">
            <el-icon><Search /></el-icon> 搜索
          </el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>

      <!-- 标签页切换 -->
      <el-tabs v-model="activeTab" @tab-change="handleTabChange" class="records-tabs">
        <el-tab-pane label="订单记录" name="orders">
          <template #label><span><el-icon><ShoppingCart /></el-icon> 订单记录</span></template>
        </el-tab-pane>
        <el-tab-pane label="充值记录" name="recharges">
          <template #label><span><el-icon><Wallet /></el-icon> 充值记录</span></template>
        </el-tab-pane>
      </el-tabs>

      <!-- 电脑端表格 (保持不变) -->
      <div class="table-wrapper" v-if="!isMobile">
        <el-table 
          :data="activeTab === 'orders' ? allRecords : recharges" 
          style="width: 100%" 
          v-loading="loading" 
          stripe
          border
          @selection-change="handleSelectionChange"
        >
          <el-table-column type="selection" width="55" v-if="activeTab === 'orders'" />
          <el-table-column prop="order_no" label="订单号" width="180" />
          <el-table-column label="用户邮箱">
            <template #default="scope">
              {{ activeTab === 'orders' ? (scope.row.user?.email || '-') : (scope.row.user?.email || '-') }}
            </template>
          </el-table-column>
          <el-table-column :label="activeTab === 'orders' ? '套餐名称/类型' : '类型'">
            <template #default="scope">
              <span v-if="activeTab === 'orders'">
                <el-tag v-if="scope.row.record_type === 'recharge'" type="success" size="small">充值</el-tag>
                <span v-else>{{ scope.row.package_name || '-' }}</span>
              </span>
              <span v-else>账户充值</span>
            </template>
          </el-table-column>
          <el-table-column prop="amount" label="金额">
            <template #default="scope">
              <span :class="(activeTab === 'recharges' || scope.row.record_type === 'recharge') ? 'positive-amount' : ''">
                {{ (activeTab === 'recharges' || scope.row.record_type === 'recharge') ? '+' : '' }}¥{{ formatMoney(scope.row.amount) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="payment_method" label="支付方式" />
          <el-table-column prop="status" label="状态">
            <template #default="scope">
              <el-tag :type="getStatusType(scope.row.status)">
                {{ getStatusText(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" />
          <el-table-column :label="activeTab === 'orders' ? '支付时间' : '支付时间'">
            <template #default="scope">
              {{ (activeTab === 'orders' ? scope.row.payment_time : scope.row.paid_at) || '-' }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="280" fixed="right" v-if="activeTab === 'orders'">
            <template #default="scope">
              <div class="action-buttons-grid" v-if="scope.row.record_type === 'order'">
                <el-button size="small" @click="viewOrder(scope.row)" class="action-btn">
                  <el-icon><View /></el-icon> 查看
                </el-button>
                <el-button 
                  size="small" 
                  type="success" 
                  @click="markAsPaid(scope.row)"
                  v-if="scope.row.status === 'pending'"
                  class="action-btn"
                >
                  <el-icon><Check /></el-icon> 标记已付
                </el-button>
                <el-button 
                  size="small" 
                  type="danger" 
                  @click="deleteOrder(scope.row)"
                  class="action-btn"
                >
                  <el-icon><Delete /></el-icon> 删除
                </el-button>
                <el-button 
                  size="small" 
                  type="danger" 
                  @click="cancelOrder(scope.row)"
                  v-if="scope.row.status === 'pending'"
                  class="action-btn"
                >
                  <el-icon><Close /></el-icon> 取消
                </el-button>
              </div>
              <span v-else class="text-muted">充值记录</span>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- 手机端卡片列表 (完全重构优化) -->
      <div class="mobile-card-list" v-if="isMobile && ((activeTab === 'orders' && allRecords.length > 0) || (activeTab === 'recharges' && recharges.length > 0))">
        <div 
          v-for="item in (activeTab === 'orders' ? allRecords : recharges)" 
          :key="item.id || item.order_no"
          class="mobile-card-optimized"
        >
          <!-- 卡片头部：订单号与状态 -->
          <div class="mc-header">
            <div class="mc-id">
              <span class="label">#</span>
              <span class="value">{{ item.order_no }}</span>
              <el-tag v-if="item.record_type === 'recharge'" type="success" size="small" effect="plain" class="ml-1">充值</el-tag>
            </div>
            <el-tag :type="getStatusType(item.status)" size="small" effect="dark">
              {{ getStatusText(item.status) }}
            </el-tag>
          </div>

          <!-- 卡片主体：左右布局 -->
          <div class="mc-body">
            <div class="mc-main-info">
              <div class="mc-amount" :class="{'is-plus': activeTab === 'recharges' || item.record_type === 'recharge'}">
                <span class="currency">¥</span>
                <span class="num">{{ formatMoney(item.amount) }}</span>
              </div>
              <div class="mc-title">
                {{ activeTab === 'orders' ? (item.package_name || '账户充值') : '账户充值' }}
              </div>
            </div>
            <div class="mc-sub-info">
              <div class="mc-row">
                <el-icon><User /></el-icon>
                <span class="text-truncate">{{ item.user?.email || '未知用户' }}</span>
              </div>
              <div class="mc-row">
                <el-icon><Wallet /></el-icon>
                <span>{{ item.payment_method || '未知支付' }}</span>
              </div>
              <div class="mc-row">
                <el-icon><Timer /></el-icon>
                <span>{{ formatDateTime(item.created_at) }}</span>
              </div>
            </div>
          </div>

          <!-- 卡片底部：操作区 -->
          <div class="mc-footer" v-if="activeTab === 'orders' && item.record_type === 'order'">
            <el-button-group class="mc-actions">
              <el-button size="small" @click="viewOrder(item)">
                 详情
              </el-button>
              <el-button 
                v-if="item.status === 'pending'"
                size="small" 
                type="success" 
                plain
                @click="markAsPaid(item)"
              >
                已付
              </el-button>
              <el-button 
                v-if="item.status === 'pending'"
                size="small" 
                type="warning" 
                plain
                @click="cancelOrder(item)"
              >
                取消
              </el-button>
              <el-button 
                size="small" 
                type="danger" 
                plain
                icon="Delete"
                @click="deleteOrder(item)"
              />
            </el-button-group>
          </div>
          <div class="mc-footer-info" v-else>
            <span class="text-muted">充值记录 - 自动处理</span>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div class="mobile-card-list" v-if="((activeTab === 'orders' && allRecords.length === 0) || (activeTab === 'recharges' && recharges.length === 0)) && !loading">
        <div class="empty-state">
          <el-icon class="empty-icon"><component :is="activeTab === 'orders' ? 'ShoppingCart' : 'Wallet'" /></el-icon>
          <p>{{ activeTab === 'orders' ? '暂无订单数据' : '暂无充值记录' }}</p>
        </div>
      </div>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="activeTab === 'recharges' ? rechargeTotal : total"
          :layout="isMobile ? 'prev, pager, next' : 'total, sizes, prev, pager, next, jumper'"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 详情抽屉 -->
    <el-drawer
      v-model="showOrderDialog"
      title="订单详情"
      :size="isMobile ? '100%' : '500px'"
      direction="rtl"
      class="order-detail-drawer"
    >
      <div class="order-detail-content">
        <div class="mobile-order-detail" v-if="isMobile">
           <!-- 使用新的手机端详情展示方式 -->
           <div class="detail-header-block">
              <div class="amount">¥{{ formatMoney(selectedOrder.amount) }}</div>
              <el-tag :type="getStatusType(selectedOrder.status)">{{ getStatusText(selectedOrder.status) }}</el-tag>
           </div>
           <div class="detail-list-block">
             <div class="d-item">
               <span class="label">订单号</span>
               <span class="val copyable">{{ selectedOrder.order_no }}</span>
             </div>
             <div class="d-item">
               <span class="label">用户</span>
               <span class="val">{{ selectedOrder.user?.email || '-' }}</span>
             </div>
             <div class="d-item">
               <span class="label">套餐</span>
               <span class="val">{{ selectedOrder.package_name }}</span>
             </div>
             <div class="d-item">
               <span class="label">支付方式</span>
               <span class="val">{{ selectedOrder.payment_method }}</span>
             </div>
             <div class="d-item">
               <span class="label">创建时间</span>
               <span class="val">{{ formatDateTime(selectedOrder.created_at) }}</span>
             </div>
             <div class="d-item" v-if="selectedOrder.payment_time">
               <span class="label">支付时间</span>
               <span class="val">{{ formatDateTime(selectedOrder.payment_time) }}</span>
             </div>
           </div>
           
           <div v-if="selectedOrder.payment_proof" class="payment-proof-section">
            <div class="section-title">支付凭证</div>
            <div class="proof-image-wrapper">
              <img :src="selectedOrder.payment_proof" class="proof-image" @click="previewImage(selectedOrder.payment_proof)" />
            </div>
          </div>
        </div>
        <!-- 电脑端详情保持不变 -->
        <el-descriptions :column="2" border v-else>
          <el-descriptions-item label="订单号">{{ selectedOrder.order_no }}</el-descriptions-item>
          <el-descriptions-item label="用户邮箱">{{ selectedOrder.user?.email }}</el-descriptions-item>
          <el-descriptions-item label="套餐名称">{{ selectedOrder.package_name }}</el-descriptions-item>
          <el-descriptions-item label="金额">¥{{ formatMoney(selectedOrder.amount) }}</el-descriptions-item>
          <el-descriptions-item label="支付方式">{{ selectedOrder.payment_method }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusType(selectedOrder.status)">
              {{ getStatusText(selectedOrder.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ selectedOrder.created_at }}</el-descriptions-item>
          <el-descriptions-item label="支付时间">{{ selectedOrder.payment_time || '-' }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="selectedOrder.payment_proof && !isMobile" style="margin-top: 20px;">
          <h4>支付凭证</h4>
          <img :src="selectedOrder.payment_proof" style="max-width: 100%; cursor: pointer;" />
        </div>
      </div>
    </el-drawer>
    <el-dialog v-model="showStatisticsDialog" title="订单统计" width="600px">
      <div class="statistics-content">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-card class="stat-card">
              <div class="stat-number">{{ statistics.totalOrders }}</div>
              <div class="stat-label">总订单数</div>
            </el-card>
          </el-col>
          <el-col :span="12">
            <el-card class="stat-card">
              <div class="stat-number">{{ statistics.pendingOrders }}</div>
              <div class="stat-label">待支付</div>
            </el-card>
          </el-col>
        </el-row>
        <el-row :gutter="20" style="margin-top: 20px;">
          <el-col :span="12">
            <el-card class="stat-card">
              <div class="stat-number">{{ statistics.paidOrders }}</div>
              <div class="stat-label">已支付</div>
            </el-card>
          </el-col>
          <el-col :span="12">
            <el-card class="stat-card">
              <div class="stat-number">¥{{ formatMoney(statistics.totalRevenue) }}</div>
              <div class="stat-label">总收入</div>
            </el-card>
          </el-col>
        </el-row>
      </div>
    </el-dialog>

    <!-- 图片预览 -->
    <el-image-viewer 
      v-if="showImageViewer" 
      :url-list="[imageViewerUrl]" 
      @close="showImageViewer = false" 
    />
  </div>
</template>

<script>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Download, Operation, DataAnalysis, View, Check, Money, Close, Search, HomeFilled,
  Filter, Refresh, Delete, Wallet, ShoppingCart, User, Timer
} from '@element-plus/icons-vue'
import { useApi, adminAPI } from '@/utils/api'
import { formatDateTime as formatDateTimeUtil } from '@/utils/date'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'

dayjs.extend(timezone)

export default {
  name: 'AdminOrders',
  components: {
    Download, Operation, DataAnalysis, View, Check, Money, Close, Search, HomeFilled,
    Filter, Refresh, Delete, Wallet, ShoppingCart, User, Timer
  },
  setup() {
    const route = useRoute()
    const api = useApi()
    
    // State
    const loading = ref(false)
    const orders = ref([])
    const recharges = ref([]) 
    const allRecords = ref([]) 
    const activeTab = ref('orders')
    const currentPage = ref(1)
    const pageSize = ref(20)
    const total = ref(0)
    const rechargeTotal = ref(0)
    
    const showOrderDialog = ref(false)
    const showStatisticsDialog = ref(false)
    const showImageViewer = ref(false)
    const imageViewerUrl = ref('')
    
    const selectedOrder = ref({})
    const selectedOrders = ref([])
    const bulkLoading = ref(false)
    
    const isMobile = ref(window.innerWidth <= 768)
    const searchForm = reactive({
      keyword: '',
      status: ''
    })
    const statistics = reactive({
      totalOrders: 0,
      pendingOrders: 0,
      paidOrders: 0,
      totalRevenue: 0
    })

    // Resize Handler
    const handleResize = () => {
      isMobile.value = window.innerWidth <= 768
    }

    // Data Loading Functions
    const loadOrders = async () => {
      loading.value = true
      try {
        const params = {
          skip: (currentPage.value - 1) * pageSize.value,
          limit: pageSize.value
        }
        if (searchForm.keyword) params.search = searchForm.keyword
        if (searchForm.status) params.status = searchForm.status
        if (activeTab.value === 'orders') params.include_recharges = 'true'
        
        const response = await api.get('/admin/orders', { params })
        const ordersList = response.data.data?.orders || []
        
        if (activeTab.value === 'orders') {
          allRecords.value = ordersList
          orders.value = ordersList.filter(r => r.record_type === 'order')
          recharges.value = ordersList.filter(r => r.record_type === 'recharge')
        } else {
          orders.value = ordersList
        }
        total.value = response.data.data?.total || response.data.total || 0
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '加载订单列表失败')
        allRecords.value = []
      } finally {
        loading.value = false
      }
    }

    const loadRecharges = async () => {
      loading.value = true
      try {
        const params = { page: currentPage.value, size: pageSize.value }
        if (searchForm.keyword) params.keyword = searchForm.keyword
        if (searchForm.status) params.status = searchForm.status
        
        const response = await adminAPI.getAdminRechargeRecords(params)
        const data = response?.data
        
        if (data?.success !== false && data?.data) {
          // 后端返回格式: { recharges: [...], total: ... }
          const responseData = data.data
          if (Array.isArray(responseData.recharges)) {
            recharges.value = responseData.recharges
            rechargeTotal.value = Number(responseData.total) || 0
          } else if (Array.isArray(responseData)) {
            // 兼容直接返回数组的情况
            recharges.value = responseData
            rechargeTotal.value = Number(data.total) || responseData.length
          } else {
            recharges.value = []
            rechargeTotal.value = 0
          }
        } else if (data?.recharges) {
          // 兼容其他可能的响应格式
          recharges.value = Array.isArray(data.recharges) ? data.recharges : []
          rechargeTotal.value = Number(data.total) || 0
        } else {
          recharges.value = []
          rechargeTotal.value = 0
        }
      } catch (error) {
        console.error('加载充值记录失败:', error)
        if (!error.response || error.response.status !== 404) {
          ElMessage.error('加载充值记录失败: ' + (error.response?.data?.message || error.message))
        }
        recharges.value = []
        rechargeTotal.value = 0
      } finally {
        loading.value = false
      }
    }

    // Handlers
    const handleTabChange = (tabName) => {
      currentPage.value = 1
      tabName === 'recharges' ? loadRecharges() : loadOrders()
    }

    const searchOrders = () => {
      currentPage.value = 1
      activeTab.value === 'recharges' ? loadRecharges() : loadOrders()
    }

    const resetSearch = () => {
      searchForm.keyword = ''
      searchForm.status = ''
      searchOrders()
    }

    const handleSizeChange = (val) => {
      pageSize.value = val
      searchOrders()
    }

    const handleCurrentChange = (val) => {
      currentPage.value = val
      activeTab.value === 'recharges' ? loadRecharges() : loadOrders()
    }

    // Actions
    const viewOrder = (order) => {
      selectedOrder.value = order
      showOrderDialog.value = true
    }
    
    const previewImage = (url) => {
      imageViewerUrl.value = url
      showImageViewer.value = true
    }

    const confirmAction = async (message, actionFn) => {
      try {
        await ElMessageBox.confirm(message, '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        await actionFn()
        ElMessage.success('操作成功')
        searchOrders() // Reload current view
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('操作失败')
        }
      }
    }

    const markAsPaid = (order) => {
      confirmAction('确定要将此订单标记为已支付吗？', async () => {
        await api.put(`/admin/orders/${order.id}`, { status: 'paid' })
      })
    }

    const cancelOrder = (order) => {
      confirmAction('确定要取消此订单吗？', async () => {
        await api.put(`/admin/orders/${order.id}`, { status: 'cancelled' })
      })
    }

    const deleteOrder = (order) => {
      confirmAction('确定要删除此订单吗？删除后无法恢复。', async () => {
        await api.delete(`/admin/orders/${order.id}`)
      })
    }

    // Bulk Actions
    const handleSelectionChange = (selection) => {
      selectedOrders.value = selection
    }

    const handleBulkAction = async (actionType, apiPath, confirmMsg) => {
      if (selectedOrders.value.length === 0) return
      
      try {
        await ElMessageBox.confirm(confirmMsg, '提示', { type: 'warning' })
        bulkLoading.value = true
        const orderIds = selectedOrders.value.map(o => o.id)
        await api.post(apiPath, { order_ids: orderIds })
        ElMessage.success('批量操作成功')
        selectedOrders.value = []
        searchOrders()
      } catch (error) {
        if (error !== 'cancel') ElMessage.error('批量操作失败')
      } finally {
        bulkLoading.value = false
      }
    }

    const bulkMarkAsPaid = () => handleBulkAction(
      'markPaid', 
      '/admin/orders/bulk-mark-paid', 
      `确定要将选中的 ${selectedOrders.value.length} 个订单标记为已支付吗？`
    )

    const bulkCancel = () => handleBulkAction(
      'cancel', 
      '/admin/orders/bulk-cancel', 
      `确定要取消选中的 ${selectedOrders.value.length} 个订单吗？`
    )

    const bulkDelete = () => handleBulkAction(
      'delete', 
      '/admin/orders/batch-delete', 
      `确定要删除选中的 ${selectedOrders.value.length} 个订单吗？`
    )

    // Export & Stats
    const exportOrders = async () => {
      // Keep existing implementation
       try {
        const params = { ...searchForm }
        const response = await api.get('/admin/orders/export', { 
          responseType: 'blob',
          params: params
        })
        const url = window.URL.createObjectURL(new Blob([response.data]))
        const link = document.createElement('a')
        link.href = url
        link.setAttribute('download', `orders_export_${dayjs().format('YYYYMMDD')}.csv`)
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        window.URL.revokeObjectURL(url)
        ElMessage.success('导出成功')
      } catch (error) {
        ElMessage.error('导出失败')
      }
    }

    const loadStatistics = async () => {
      try {
        const res = await api.get('/admin/orders/statistics')
        if (res.data?.data) {
          const s = res.data.data
          Object.assign(statistics, {
            totalOrders: s.total_orders || 0,
            pendingOrders: s.pending_orders || 0,
            paidOrders: s.paid_orders || 0,
            totalRevenue: s.total_revenue || 0
          })
        }
      } catch (e) {
        // 统计信息加载失败，不影响主功能
      }
    }

    // Utilities
    const getStatusType = (status) => ({
      'pending': 'warning',
      'paid': 'success',
      'cancelled': 'danger'
    }[status] || 'info')

    const getStatusText = (status) => ({
      'pending': '待支付',
      'paid': '已支付',
      'cancelled': '已取消'
    }[status] || status)

    const formatDateTime = (d) => formatDateTimeUtil(d) || '-'
    const formatMoney = (v) => {
      const n = parseFloat(v)
      return isNaN(n) ? '0.00' : n.toFixed(2)
    }

    // Lifecycle
    onMounted(() => {
      if (route.query.search) searchForm.keyword = String(route.query.search).trim()
      window.addEventListener('resize', handleResize)
      loadOrders()
      loadStatistics()
    })

    onUnmounted(() => {
      window.removeEventListener('resize', handleResize)
    })

    return {
      // State
      loading, orders, recharges, allRecords, activeTab, 
      currentPage, pageSize, total, rechargeTotal, 
      searchForm, statistics, isMobile, bulkLoading,
      showOrderDialog, showStatisticsDialog, showImageViewer, imageViewerUrl,
      selectedOrder, selectedOrders,
      
      // Actions
      searchOrders, resetSearch, handleTabChange,
      handleSizeChange, handleCurrentChange, handleSelectionChange,
      viewOrder, previewImage, markAsPaid, cancelOrder, deleteOrder,
      exportOrders, bulkMarkAsPaid, bulkCancel, bulkDelete,
      
      // Utils
      getStatusType, getStatusText, formatDateTime, formatMoney
    }
  }
}
</script>

<style scoped lang="scss">
@use '@/styles/list-common.scss';

// 通用样式
.positive-amount { color: #67c23a; font-weight: 600; }
.records-tabs { margin-bottom: 20px; }
.text-muted { color: #909399; font-size: 12px; }
.ml-1 { margin-left: 4px; }

// 电脑端样式保留
.bulk-actions, .normal-actions, .header-actions { display: flex; gap: 10px; align-items: center; }
.selected-count { color: #409eff; font-weight: 600; font-size: 14px; }
.action-buttons-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 8px; }

// 手机端优化样式
@media (max-width: 768px) {
  // 1. 搜索与筛选栏优化
  .mobile-action-bar {
    padding: 12px;
    background: #f8fafc;
    border-radius: 8px;
    margin-bottom: 16px;
    display: flex;
    flex-direction: column;
    gap: 10px;

    .mobile-search-row {
      display: flex;
      gap: 8px;
      .mobile-search-input {
        flex: 1;
        --el-input-border-radius: 20px;
      }
      .mobile-search-btn {
        border-radius: 20px;
        padding: 0 20px;
      }
    }

    .mobile-filter-row {
      display: flex;
      gap: 8px;
      align-items: center;
      .mobile-filter-select {
        flex: 1;
        :deep(.el-input__wrapper) {
          border-radius: 20px;
        }
      }
      .mobile-reset-btn {
        flex-shrink: 0;
      }
    }
  }

  // 2. 手机端卡片深度优化
  .mobile-card-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .mobile-card-optimized {
    background: white;
    border-radius: 12px;
    box-shadow: 0 2px 12px rgba(0,0,0,0.05);
    overflow: hidden;
    border: 1px solid #ebeef5;
    
    // 头部：ID与状态
    .mc-header {
      padding: 12px 16px;
      background: #f8f9fa;
      border-bottom: 1px solid #ebeef5;
      display: flex;
      justify-content: space-between;
      align-items: center;

      .mc-id {
        font-family: monospace;
        color: #606266;
        display: flex;
        align-items: center;
        .label { color: #909399; margin-right: 2px; }
        .value { font-weight: 600; }
      }
    }

    // 主体：关键信息
    .mc-body {
      padding: 16px;
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      gap: 12px;
    }

    .mc-main-info {
      flex: 1;
      .mc-amount {
        font-size: 20px;
        font-weight: 700;
        color: #303133;
        line-height: 1.2;
        margin-bottom: 4px;
        &.is-plus { color: #67c23a; }
        .currency { font-size: 14px; margin-right: 2px; }
      }
      .mc-title {
        font-size: 14px;
        color: #606266;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
      }
    }

    .mc-sub-info {
      display: flex;
      flex-direction: column;
      gap: 6px;
      align-items: flex-end;
      min-width: 100px;

      .mc-row {
        display: flex;
        align-items: center;
        gap: 4px;
        font-size: 12px;
        color: #909399;
        
        .text-truncate {
          max-width: 120px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
      }
    }

    // 底部：操作按钮
    .mc-footer {
      padding: 10px 16px;
      border-top: 1px solid #f0f2f5;
      display: flex;
      justify-content: flex-end;
      
      .mc-actions {
        width: 100%;
        display: flex;
        :deep(.el-button) {
          flex: 1;
        }
      }
    }
    .mc-footer-info {
      padding: 8px 16px;
      background: #fafafa;
      text-align: right;
      font-size: 12px;
    }
  }

  // 3. 空状态与分页
  .empty-state {
    padding: 40px 0;
    text-align: center;
    color: #909399;
    .empty-icon { font-size: 48px; margin-bottom: 10px; opacity: 0.5; }
    p { margin: 0; font-size: 14px; }
  }
  
  .pagination {
    margin-top: 20px;
    display: flex;
    justify-content: center;
    :deep(.el-pagination) {
      .el-pagination__jump { display: none; }
    }
  }

  // 4. 详情弹窗优化
  .mobile-order-detail {
    .detail-header-block {
      text-align: center;
      padding: 20px 0;
      background: #f8fafc;
      margin: -20px -20px 20px -20px; // 抵消 dialog padding
      border-bottom: 1px solid #ebeef5;
      
      .amount {
        font-size: 28px;
        font-weight: 700;
        color: #303133;
        margin-bottom: 8px;
      }
    }
    
    .detail-list-block {
      .d-item {
        display: flex;
        justify-content: space-between;
        padding: 12px 0;
        border-bottom: 1px dashed #ebeef5;
        font-size: 14px;
        &:last-child { border-bottom: none; }
        
        .label { color: #909399; }
        .val { 
          color: #303133; 
          font-weight: 500; 
          text-align: right; 
          max-width: 70%;
          word-break: break-all;
        }
      }
    }
    
    .payment-proof-section {
      margin-top: 20px;
      .section-title { font-weight: 600; margin-bottom: 10px; }
      .proof-image { width: 100%; border-radius: 8px; border: 1px solid #eee; }
    }
  }
}
</style>