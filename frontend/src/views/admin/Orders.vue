<template>
  <div class="list-container admin-orders">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>订单列表</span>
          <div class="header-actions">
            <div class="bulk-actions" v-if="selectedOrders.length > 0">
              <span class="selected-count">已选择 {{ selectedOrders.length }} 个订单</span>
              <el-button 
                type="success" 
                size="small"
                @click="bulkMarkAsPaid"
                :disabled="bulkLoading"
              >
                <el-icon><Check /></el-icon>
                批量标记已付
              </el-button>
              <el-button 
                type="warning" 
                size="small"
                @click="bulkCancel"
                :disabled="bulkLoading"
              >
                <el-icon><Close /></el-icon>
                批量取消
              </el-button>
              <el-button 
                type="danger" 
                size="small"
                @click="bulkDelete"
                :disabled="bulkLoading"
              >
                <el-icon><Delete /></el-icon>
                批量删除
              </el-button>
            </div>
            <div class="normal-actions" v-else>
              <el-button type="success" @click="exportOrders">
                <el-icon><Download /></el-icon>
                导出订单
              </el-button>
              <el-button type="info" @click="showStatisticsDialog = true">
                <el-icon><DataAnalysis /></el-icon>
                订单统计
              </el-button>
            </div>
          </div>
        </div>
      </template>
      <div class="mobile-action-bar">
        <div class="mobile-search-section">
          <div class="search-input-wrapper">
            <el-input 
              v-model="searchForm.keyword" 
              placeholder="搜索订单号、时间戳、用户邮箱或用户名"
              class="mobile-search-input"
              clearable
              @keyup.enter="searchOrders"
            />
            <el-button 
              @click="searchOrders" 
              class="search-button-inside"
              type="default"
              plain
            >
              <el-icon><Search /></el-icon>
            </el-button>
          </div>
        </div>
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
                <el-dropdown-item command="pending">待支付</el-dropdown-item>
                <el-dropdown-item command="paid">已支付</el-dropdown-item>
                <el-dropdown-item command="cancelled">已取消</el-dropdown-item>
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
      </div>
      <el-form :inline="true" :model="searchForm" class="search-form">
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
            <el-icon><Search /></el-icon>
            搜索
          </el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>

      <!-- 标签页切换 -->
      <el-tabs v-model="activeTab" @tab-change="handleTabChange" class="records-tabs">
        <el-tab-pane label="订单记录" name="orders">
          <template #label>
            <span><el-icon><ShoppingCart /></el-icon> 订单记录</span>
          </template>
        </el-tab-pane>
        <el-tab-pane label="充值记录" name="recharges">
          <template #label>
            <span><el-icon><Wallet /></el-icon> 充值记录</span>
          </template>
        </el-tab-pane>
      </el-tabs>

      <!-- 桌面端表格 -->
      <div class="table-wrapper">
        <el-table 
          :data="activeTab === 'orders' ? allRecords : recharges" 
          style="width: 100%" 
          v-loading="loading" 
          stripe
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
                <el-icon><View /></el-icon>
                查看
              </el-button>
              <el-button 
                size="small" 
                type="success" 
                @click="markAsPaid(scope.row)"
                v-if="scope.row.status === 'pending'"
                class="action-btn"
              >
                <el-icon><Check /></el-icon>
                标记已付
              </el-button>
              <el-button 
                size="small" 
                type="danger" 
                @click="deleteOrder(scope.row)"
                class="action-btn"
              >
                <el-icon><Delete /></el-icon>
                删除
              </el-button>
              <el-button 
                size="small" 
                type="danger" 
                @click="cancelOrder(scope.row)"
                v-if="scope.row.status === 'pending'"
                class="action-btn"
              >
                <el-icon><Close /></el-icon>
                取消
              </el-button>
            </div>
            <span v-else class="text-muted">充值记录</span>
          </template>
        </el-table-column>
      </el-table>
      </div>

      <!-- 移动端卡片式列表 -->
      <div class="mobile-card-list" v-if="(activeTab === 'orders' && allRecords.length > 0) || (activeTab === 'recharges' && recharges.length > 0)">
        <div 
          v-for="item in (activeTab === 'orders' ? allRecords : recharges)" 
          :key="item.id || item.order_no"
          class="mobile-card"
        >
          <div class="card-row">
            <span class="label">订单号</span>
            <span class="value">{{ item.order_no }}</span>
          </div>
          <div class="card-row">
            <span class="label">用户邮箱</span>
            <span class="value">{{ item.user?.email || '-' }}</span>
          </div>
          <div class="card-row">
            <span class="label">{{ activeTab === 'orders' ? '套餐名称/类型' : '类型' }}</span>
            <span class="value">
              <span v-if="activeTab === 'orders'">
                <el-tag v-if="item.record_type === 'recharge'" type="success" size="small">充值</el-tag>
                <span v-else>{{ item.package_name || '-' }}</span>
              </span>
              <span v-else>账户充值</span>
            </span>
          </div>
          <div class="card-row">
            <span class="label">金额</span>
            <span class="value" :class="(activeTab === 'recharges' || item.record_type === 'recharge') ? 'positive-amount' : ''">
              {{ (activeTab === 'recharges' || item.record_type === 'recharge') ? '+' : '' }}¥{{ formatMoney(item.amount) }}
            </span>
          </div>
          <div class="card-row">
            <span class="label">支付方式</span>
            <span class="value">{{ item.payment_method || '-' }}</span>
          </div>
          <div class="card-row">
            <span class="label">状态</span>
            <span class="value">
              <el-tag :type="getStatusType(item.status)">
                {{ getStatusText(item.status) }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">创建时间</span>
            <span class="value">{{ item.created_at }}</span>
          </div>
          <div class="card-row" v-if="(activeTab === 'orders' && item.payment_time) || (activeTab === 'recharges' && item.paid_at)">
            <span class="label">支付时间</span>
            <span class="value">{{ (activeTab === 'orders' ? item.payment_time : item.paid_at) || '-' }}</span>
          </div>
          <div class="card-actions" v-if="activeTab === 'orders' && item.record_type === 'order'">
            <el-button size="small" @click="viewOrder(item)" class="action-btn">
              <el-icon><View /></el-icon>
              查看
            </el-button>
            <el-button 
              v-if="item.status === 'pending'"
              size="small" 
              type="success" 
              @click="markAsPaid(item)"
              class="action-btn"
            >
              <el-icon><Check /></el-icon>
              标记已付
            </el-button>
            <el-button 
              size="small" 
              type="danger" 
              @click="deleteOrder(item)"
              class="action-btn"
            >
              <el-icon><Delete /></el-icon>
              删除
            </el-button>
            <el-button 
              v-if="item.status === 'pending'"
              size="small" 
              type="danger" 
              @click="cancelOrder(item)"
              class="action-btn"
            >
              <el-icon><Close /></el-icon>
              取消
            </el-button>
          </div>
          <div v-else-if="activeTab === 'orders' && item.record_type === 'recharge'" class="text-muted" style="padding: 8px;">
            充值记录
          </div>
        </div>
      </div>

      <!-- 移动端空状态 -->
      <div class="mobile-card-list" v-if="((activeTab === 'orders' && allRecords.length === 0) || (activeTab === 'recharges' && recharges.length === 0)) && !loading">
        <div class="empty-state">
          <i :class="activeTab === 'orders' ? 'el-icon-shopping-cart-2' : 'el-icon-wallet'"></i>
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
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 订单详情对话框 -->
    <el-dialog 
      v-model="showOrderDialog" 
      title="订单详情" 
      :width="isMobile ? '95%' : '600px'"
      class="order-detail-dialog"
      :close-on-click-modal="false"
    >
      <div class="order-detail-content">
        <!-- 移动端卡片式布局 -->
        <div class="mobile-order-detail" v-if="isMobile">
          <div class="detail-card">
            <div class="detail-row">
              <span class="detail-label">订单号</span>
              <span class="detail-value">{{ selectedOrder.order_no }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">用户邮箱</span>
              <span class="detail-value">{{ selectedOrder.user?.email || '-' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">套餐名称</span>
              <span class="detail-value">{{ selectedOrder.package_name }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">金额</span>
              <span class="detail-value highlight">¥{{ formatMoney(selectedOrder.amount) }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">支付方式</span>
              <span class="detail-value">{{ selectedOrder.payment_method }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">状态</span>
              <span class="detail-value">
                <el-tag :type="getStatusType(selectedOrder.status)" size="small">
                  {{ getStatusText(selectedOrder.status) }}
                </el-tag>
              </span>
            </div>
            <div class="detail-row">
              <span class="detail-label">创建时间</span>
              <span class="detail-value">{{ formatDateTime(selectedOrder.created_at) }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">支付时间</span>
              <span class="detail-value">{{ selectedOrder.payment_time ? formatDateTime(selectedOrder.payment_time) : '-' }}</span>
            </div>
          </div>
          
          <div v-if="selectedOrder.payment_proof" class="payment-proof-section">
            <div class="section-title">支付凭证</div>
            <div class="proof-image-wrapper">
              <img :src="selectedOrder.payment_proof" class="proof-image" />
            </div>
          </div>
        </div>
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
          <img :src="selectedOrder.payment_proof" style="max-width: 100%;" />
        </div>
      </div>
    </el-dialog>
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
  </div>
</template>

<script>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Download, Operation, DataAnalysis, View, Check, Money, Close, Search, HomeFilled,
  Filter, Refresh, Delete, Wallet, ShoppingCart
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
    Filter, Refresh, Delete, Wallet, ShoppingCart
  },
  setup() {
    const route = useRoute()
    const api = useApi()
    const loading = ref(false)
    const orders = ref([])
    const recharges = ref([]) // 充值记录
    const allRecords = ref([]) // 合并的订单和充值记录（用于"订单记录"标签页）
    const activeTab = ref('orders') // 标签页：orders-订单，recharges-充值
    const currentPage = ref(1)
    const pageSize = ref(20)
    const total = ref(0)
    const rechargeTotal = ref(0) // 充值记录总数
    const showOrderDialog = ref(false)
    const showStatisticsDialog = ref(false)
    const selectedOrder = ref({})
    const selectedOrders = ref([])
    const bulkLoading = ref(false)
    const isMobile = ref(window.innerWidth <= 768)
    
    // 监听窗口大小变化
    const handleResize = () => {
      isMobile.value = window.innerWidth <= 768
    }
    
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

    const loadOrders = async () => {
      loading.value = true
      try {
        const params = {
          skip: (currentPage.value - 1) * pageSize.value,
          limit: pageSize.value
        }
        
        // 添加搜索参数
        if (searchForm.keyword) {
          params.search = searchForm.keyword
        }
        if (searchForm.status) {
          params.status = searchForm.status
        }
        
        if (activeTab.value === 'orders') {
          params.include_recharges = 'true'
        }
        
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
        const errorMsg = error.response?.data?.message || error.message || '加载订单列表失败'
        ElMessage.error(errorMsg)
        // 确保即使出错也清空数据，避免显示旧数据
        orders.value = []
        total.value = 0
        allRecords.value = []
        recharges.value = []
      } finally {
        loading.value = false
      }
    }

    const loadRecharges = async () => {
      loading.value = true
      try {
        if (!adminAPI || typeof adminAPI.getAdminRechargeRecords !== 'function') {
          ElMessage.error('加载充值记录失败: API 函数未定义')
          return
        }
        
        const params = {
          page: currentPage.value,
          size: pageSize.value
        }
        
        if (searchForm.keyword) {
          params.keyword = searchForm.keyword
        }
        if (searchForm.status) {
          params.status = searchForm.status
        }
        
        const response = await adminAPI.getAdminRechargeRecords(params)
        
        if (response && response.data) {
          let data = null
          
          if (response.data.success !== false && response.data.data) {
            data = response.data.data
          } else if (response.data.recharges !== undefined) {
            data = response.data
          } else if (response.data.data && Array.isArray(response.data.data)) {
            data = { recharges: response.data.data, total: response.data.total || 0 }
          } else if (response.data.success === undefined && response.data.recharges === undefined) {
            data = response.data
          }
          
          if (data && (data.recharges !== undefined || Array.isArray(data))) {
            recharges.value = Array.isArray(data.recharges) ? data.recharges : (Array.isArray(data) ? data : [])
            rechargeTotal.value = Number(data.total) || 0
          } else {
            recharges.value = []
            rechargeTotal.value = 0
          }
        } else {
          recharges.value = []
          rechargeTotal.value = 0
        }
      } catch (error) {
        const errorStatus = error.response?.status
        const errorMsg = error.response?.data?.message || error.message || ''
        
        if (errorStatus === 404 || errorMsg.includes('不存在')) {
          recharges.value = []
          rechargeTotal.value = 0
        } else {
          ElMessage.error('加载充值记录失败: ' + errorMsg)
          recharges.value = []
          rechargeTotal.value = 0
        }
      } finally {
        loading.value = false
      }
    }

    // 标签页切换处理
    const handleTabChange = (tabName) => {
      currentPage.value = 1
      if (tabName === 'recharges') {
        loadRecharges()
      } else {
        // "订单记录"标签页：后端会同时返回订单和充值记录
        loadOrders()
      }
    }

    const searchOrders = () => {
      currentPage.value = 1
      if (activeTab.value === 'recharges') {
        loadRecharges()
      } else {
        // "订单记录"标签页：后端会同时返回订单和充值记录
        loadOrders()
      }
    }

    const resetSearch = () => {
      Object.assign(searchForm, { 
        keyword: '', 
        status: '' 
      })
      currentPage.value = 1
      if (activeTab.value === 'recharges') {
        loadRecharges()
      } else {
        // "订单记录"标签页：后端会同时返回订单和充值记录
        loadOrders()
      }
    }

    // 处理状态筛选
    const handleStatusFilter = (command) => {
      searchForm.status = command
      searchOrders()
    }

    // 获取状态筛选文本
    const getStatusFilterText = () => {
      const statusMap = {
        '': '全部状态',
        'pending': '待支付',
        'paid': '已支付',
        'cancelled': '已取消',
      }
      return statusMap[searchForm.status] || '全部状态'
    }

    const handleSizeChange = (val) => {
      pageSize.value = val
      currentPage.value = 1
      if (activeTab.value === 'recharges') {
        loadRecharges()
      } else {
        loadOrders()
      }
    }

    const handleCurrentChange = (val) => {
      currentPage.value = val
      if (activeTab.value === 'recharges') {
        loadRecharges()
      } else {
        loadOrders()
      }
    }

    const viewOrder = (order) => {
      selectedOrder.value = order
      showOrderDialog.value = true
    }

    const markAsPaid = async (order) => {
      try {
        await ElMessageBox.confirm('确定要将此订单标记为已支付吗？', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        
        const response = await api.put(`/admin/orders/${order.id}`, { status: 'paid' })
        
        if (response.data.success) {
          ElMessage.success('订单状态更新成功')
          if (activeTab.value === 'recharges') {
            await loadRecharges()
          } else {
            await loadOrders()
          }
        } else {
          ElMessage.error(response.data.message || '操作失败')
        }
      } catch (error) {
        if (error !== 'cancel') {
          const errorMsg = error.response?.data?.message || error.message || '操作失败'
          ElMessage.error(`操作失败: ${errorMsg}`)
        }
      }
    }

    const cancelOrder = async (order) => {
      try {
        await ElMessageBox.confirm('确定要取消此订单吗？', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        await api.put(`/admin/orders/${order.id}`, { status: 'cancelled' })
        ElMessage.success('订单已取消')
        if (activeTab.value === 'recharges') {
          loadRecharges()
        } else {
          loadOrders()
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('操作失败')
        }
      }
    }

    const deleteOrder = async (order) => {
      try {
        await ElMessageBox.confirm('确定要删除此订单吗？删除后无法恢复。', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        await api.delete(`/admin/orders/${order.id}`)
        ElMessage.success('订单删除成功')
        if (activeTab.value === 'recharges') {
          loadRecharges()
        } else {
          loadOrders()
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('删除失败')
          }
      }
    }
    
    const handleSelectionChange = (selection) => {
      selectedOrders.value = selection
    }

    const exportOrders = async () => {
      try {
        // 构建查询参数
        const params = {}
        if (searchForm.keyword) {
          params.search = searchForm.keyword
        }
        if (searchForm.status) {
          params.status = searchForm.status
        }
        
        const response = await api.get('/admin/orders/export', { 
          responseType: 'blob',
          params: params
        })
        
        // 从响应头获取文件名，如果没有则使用默认名称
        const contentDisposition = response.headers['content-disposition']
        // 使用北京时间生成文件名
        const beijingDate = dayjs().tz('Asia/Shanghai')
        let filename = `orders_export_${beijingDate.format('YYYYMMDD')}.csv`
        
        if (contentDisposition) {
          const filenameMatch = contentDisposition.match(/filename\*=UTF-8''(.+)/)
          if (filenameMatch) {
            filename = decodeURIComponent(filenameMatch[1])
          }
        }
        
        const url = window.URL.createObjectURL(new Blob([response.data], { type: 'text/csv;charset=utf-8' }))
        const link = document.createElement('a')
        link.href = url
        link.setAttribute('download', filename)
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        window.URL.revokeObjectURL(url)
        
        ElMessage.success('订单数据导出成功（CSV格式，可用Excel打开）')
      } catch (error) {
        const errorMsg = error.response?.data?.message || error.message || '导出失败'
        ElMessage.error(`导出失败: ${errorMsg}`)
      }
    }

    // 批量标记已付
    const bulkMarkAsPaid = async () => {
      if (selectedOrders.value.length === 0) {
        ElMessage.warning('请先选择要操作的订单')
        return
      }

      try {
        await ElMessageBox.confirm(`确定要将选中的 ${selectedOrders.value.length} 个订单标记为已支付吗？`, '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        
        bulkLoading.value = true
        const orderIds = selectedOrders.value.map(order => order.id)
        await api.post('/admin/orders/bulk-mark-paid', {
          order_ids: orderIds
        })
        ElMessage.success('批量标记已付成功')
        selectedOrders.value = []
        if (activeTab.value === 'recharges') {
          await loadRecharges()
        } else {
          await loadOrders()
        }
      } catch (error) {
        if (error !== 'cancel') {
          const errorMsg = error.response?.data?.message || error.message || '批量操作失败'
          ElMessage.error(errorMsg)
        }
      } finally {
        bulkLoading.value = false
      }
    }

    // 批量取消
    const bulkCancel = async () => {
      if (selectedOrders.value.length === 0) {
        ElMessage.warning('请先选择要操作的订单')
        return
      }

      try {
        await ElMessageBox.confirm(`确定要取消选中的 ${selectedOrders.value.length} 个订单吗？`, '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        
        bulkLoading.value = true
        const orderIds = selectedOrders.value.map(order => order.id)
        await api.post('/admin/orders/bulk-cancel', {
          order_ids: orderIds
        })
        ElMessage.success('批量取消成功')
        selectedOrders.value = []
        if (activeTab.value === 'recharges') {
          await loadRecharges()
        } else {
          await loadOrders()
        }
      } catch (error) {
        if (error !== 'cancel') {
          const errorMsg = error.response?.data?.message || error.message || '批量操作失败'
          ElMessage.error(errorMsg)
        }
      } finally {
        bulkLoading.value = false
      }
    }

    // 批量删除
    const bulkDelete = async () => {
      if (selectedOrders.value.length === 0) {
        ElMessage.warning('请先选择要操作的订单')
        return
      }

      try {
        await ElMessageBox.confirm(`确定要删除选中的 ${selectedOrders.value.length} 个订单吗？删除后无法恢复。`, '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        
        bulkLoading.value = true
        const orderIds = selectedOrders.value.map(order => order.id)
        await api.post('/admin/orders/batch-delete', {
          order_ids: orderIds
        })
        ElMessage.success('批量删除成功')
        selectedOrders.value = []
        if (activeTab.value === 'recharges') {
          await loadRecharges()
        } else {
          await loadOrders()
        }
      } catch (error) {
        if (error !== 'cancel') {
          const errorMsg = error.response?.data?.message || error.message || '批量操作失败'
          ElMessage.error(errorMsg)
        }
      } finally {
        bulkLoading.value = false
      }
    }

    const loadStatistics = async () => {
      try {
        const response = await api.get('/admin/orders/statistics')
        
        if (response.data && response.data.success && response.data.data) {
          const statsData = response.data.data
          statistics.totalOrders = statsData.total_orders || 0
          statistics.pendingOrders = statsData.pending_orders || 0
          statistics.paidOrders = statsData.paid_orders || 0
          statistics.totalRevenue = statsData.total_revenue || 0
        }
      } catch (error) {
        // 统计失败不影响主要功能，只记录错误
      }
    }

    const getStatusType = (status) => {
      const statusMap = {
        'pending': 'warning',
        'paid': 'success',
        'cancelled': 'danger',
      }
      return statusMap[status] || 'info'
    }

    const getStatusText = (status) => {
      const statusMap = {
        'pending': '待支付',
        'paid': '已支付',
        'cancelled': '已取消',
      }
      return statusMap[status] || status
    }
    
    const formatDateTime = (dateString) => {
      // 使用统一的工具函数，确保使用北京时间
      return formatDateTimeUtil(dateString) || '-'
    }

    const formatMoney = (value) => {
      if (value === null || value === undefined || value === '') return '0.00'
      const num = typeof value === 'string' ? parseFloat(value) : value
      if (isNaN(num)) return '0.00'
      return num.toFixed(2)
    }

    onMounted(() => {
      // 检查 URL 参数中是否有搜索关键词
      if (route.query.search) {
        const searchParam = String(route.query.search).trim()
        if (searchParam) {
          searchForm.keyword = searchParam
          currentPage.value = 1
        }
      }
      window.addEventListener('resize', handleResize)
      // "订单记录"标签页：后端会同时返回订单和充值记录
      loadOrders()
      loadStatistics()
    })
    
    onUnmounted(() => {
      window.removeEventListener('resize', handleResize)
    })

    return {
      loading,
      orders,
      recharges,
      allRecords,
      activeTab,
      currentPage,
      pageSize,
      total,
      rechargeTotal,
      searchForm,
      showOrderDialog,
      selectedOrder,
      searchOrders,
      resetSearch,
      handleStatusFilter,
      getStatusFilterText,
      handleSizeChange,
      handleCurrentChange,
      viewOrder,
      markAsPaid,
      cancelOrder,
      deleteOrder,
      handleSelectionChange,
      selectedOrders,
      exportOrders,
      bulkMarkAsPaid,
      bulkCancel,
      bulkDelete,
      loadStatistics,
      loadRecharges,
      handleTabChange,
      getStatusType,
      getStatusText,
      formatDateTime,
      formatMoney,
      isMobile,
      // 新增的响应式变量
      showStatisticsDialog,
      bulkLoading,
      statistics
    }
  }
}
</script>

<style scoped lang="scss">
@use '@/styles/list-common.scss';

// admin-orders 使用 list-container 的样式，无需额外定义

.positive-amount {
  color: #67c23a;
  font-weight: 600;
}

.records-tabs {
  margin-bottom: 20px;
}

/* 批量操作按钮组样式 */
.bulk-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  
  .selected-count {
    color: #409eff;
    font-weight: 600;
    font-size: 14px;
    margin-right: 8px;
  }
  
  .el-button {
    margin: 0;
  }
}

.normal-actions {
  display: flex;
  gap: 10px;
  align-items: center;
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
  
  // 移动端优化显示
  @media (max-width: 768px) {
    padding: 15px;
    
    :deep(.el-form-item) {
      margin-bottom: 10px;
      width: 100%;
    }
    
    :deep(.el-input) {
      width: 100% !important;
    }
    
    :deep(.el-select) {
      width: 100% !important;
    }
    
    :deep(.el-form-item__content) {
      width: 100%;
    }
  }
}

.pagination {
  margin-top: 20px;
  text-align: right;
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

/* 操作按钮网格布局 - 2x2排列 */
.action-buttons-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
  width: 100%;
  min-width: 200px;
  
  .action-btn {
    width: 100%;
    min-width: 0;
    padding: 8px 12px;
    font-size: 12px;
    white-space: nowrap;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    
    :deep(.el-icon) {
      font-size: 14px;
    }
    
    // 如果只有3个按钮，第3个按钮占满整行
    &:nth-child(3):last-child {
      grid-column: 1 / -1;
    }
  }
  
  // 如果只有2个按钮，让它们各占一列
  &:has(.action-btn:nth-child(2):last-child) {
    grid-template-columns: repeat(2, 1fr);
  }
  
  // 如果只有1个按钮，让它占满整行
  &:has(.action-btn:nth-child(1):last-child) {
    grid-template-columns: 1fr;
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

// 订单详情弹窗样式
.order-detail-dialog {
  :deep(.el-dialog) {
    @media (max-width: 768px) {
      margin: 5vh auto !important;
      border-radius: 16px;
      overflow: hidden;
    }
  }
  
  :deep(.el-dialog__header) {
    @media (max-width: 768px) {
      padding: 16px 20px;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: #ffffff;
      border-bottom: none;
      
      .el-dialog__title {
        font-size: 18px;
        font-weight: 600;
        color: #ffffff;
      }
      
      .el-dialog__headerbtn {
        .el-dialog__close {
          color: #ffffff;
          font-size: 20px;
          
          &:hover {
            color: rgba(255, 255, 255, 0.8);
          }
        }
      }
    }
  }
  
  :deep(.el-dialog__body) {
    @media (max-width: 768px) {
      padding: 0;
    }
  }
}

.order-detail-content {
  @media (max-width: 768px) {
    padding: 0;
  }
}

// 移动端订单详情卡片样式
.mobile-order-detail {
  padding: 0;
  
  .detail-card {
    background: #ffffff;
    border-radius: 0;
    padding: 20px;
    
    .detail-row {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 14px 0;
      border-bottom: 1px solid #f0f0f0;
      
      &:last-child {
        border-bottom: none;
      }
      
      .detail-label {
        font-size: 14px;
        font-weight: 600;
        color: #606266;
        min-width: 90px;
        flex-shrink: 0;
      }
      
      .detail-value {
        font-size: 14px;
        color: #303133;
        text-align: right;
        flex: 1;
        word-break: break-all;
        
        &.highlight {
          font-size: 18px;
          font-weight: 700;
          color: #f56c6c;
        }
      }
    }
  }
  
  .payment-proof-section {
    margin-top: 20px;
    padding: 20px;
    background: #f8f9fa;
    border-top: 1px solid #e4e7ed;
    
    .section-title {
      font-size: 16px;
      font-weight: 600;
      color: #303133;
      margin-bottom: 16px;
      padding-bottom: 12px;
      border-bottom: 2px solid #e4e7ed;
    }
    
    .proof-image-wrapper {
      display: flex;
      justify-content: center;
      align-items: center;
      background: #ffffff;
      border-radius: 12px;
      padding: 12px;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
      
      .proof-image {
        max-width: 100%;
        max-height: 400px;
        border-radius: 8px;
        object-fit: contain;
      }
    }
  }
}

// 手机端订单管理页面特定样式优化
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
</style> 