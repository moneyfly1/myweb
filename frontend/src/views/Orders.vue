<template>
  <div class="list-container orders-container">
    <div class="stats-row" style="margin-top: 0;">
      <div class="stat-card">
        <div class="stat-number">{{ orderStats.total }}</div>
        <div class="stat-label">总订单数</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ orderStats.pending }}</div>
        <div class="stat-label">待支付</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ orderStats.paid }}</div>
        <div class="stat-label">已支付</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ formatAmount(orderStats.totalAmount) }}</div>
        <div class="stat-label">总金额(元)</div>
      </div>
    </div>
    <el-card class="filter-card">
      <div class="filter-desktop">
        <el-row :gutter="16" align="middle">
          <el-col :span="5">
            <el-select v-model="filters.status" placeholder="订单状态" clearable>
              <el-option label="全部状态" value="" />
              <el-option label="待支付" value="pending" />
              <el-option label="已支付" value="paid" />
              <el-option label="已取消" value="cancelled" />
              <el-option label="支付失败" value="failed" />
            </el-select>
          </el-col>
          <el-col :span="5">
            <el-select v-model="filters.payment_method" placeholder="支付方式" clearable>
              <el-option label="全部方式" value="" />
              <el-option label="支付宝" value="alipay" />
              <el-option label="微信支付" value="wechat" />
            </el-select>
          </el-col>
          <el-col :span="10">
            <el-date-picker
              v-model="filters.date_range"
              type="daterange"
              range-separator="至"
              start-placeholder="开始日期"
              end-placeholder="结束日期"
              format="YYYY-MM-DD"
              value-format="YYYY-MM-DD"
              style="width: 100%"
            />
          </el-col>
          <el-col :span="4" class="filter-buttons-col">
            <el-button type="primary" @click="applyFilters" size="default">筛选</el-button>
            <el-button @click="resetFilters" size="default">重置</el-button>
          </el-col>
        </el-row>
      </div>
      <div class="filter-mobile">
        <div class="filter-row">
          <el-select 
            v-model="filters.status" 
            placeholder="订单状态" 
            clearable
            class="filter-select"
          >
            <el-option label="全部状态" value="" />
            <el-option label="待支付" value="pending" />
            <el-option label="已支付" value="paid" />
            <el-option label="已取消" value="cancelled" />
            <el-option label="支付失败" value="failed" />
          </el-select>
        </div>
        <div class="filter-row">
          <el-select 
            v-model="filters.payment_method" 
            placeholder="支付方式" 
            clearable
            class="filter-select"
          >
            <el-option label="全部方式" value="" />
            <el-option label="支付宝" value="alipay" />
            <el-option label="微信支付" value="wechat" />
          </el-select>
        </div>
        <div class="filter-row">
          <el-date-picker
            v-model="filters.date_range"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
            class="filter-date"
          />
        </div>
        <div class="filter-actions">
          <el-button 
            type="primary" 
            @click="applyFilters" 
            class="filter-btn"
            size="default"
          >
            筛选
          </el-button>
          <el-button 
            @click="resetFilters" 
            class="filter-btn"
            size="default"
          >
            重置
          </el-button>
        </div>
      </div>
    </el-card>
    <el-card class="list-card orders-list">
      <template #header>
        <div class="card-header">
          <span>订单记录</span>
          <el-button type="primary" @click="refreshOrders">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </template>
      <el-tabs v-model="activeTab" @tab-change="handleTabChange" class="records-tabs">
        <el-tab-pane label="全部记录" name="all">
          <template #label>
            <span><el-icon><Wallet /></el-icon> 全部记录</span>
          </template>
        </el-tab-pane>
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
      <div class="table-wrapper">
        <el-table 
          ref="orderTableRef"
          :data="displayRecords" 
          style="width: 100%"
          v-loading="isLoading || isLoadingRecharges"
          :empty-text="emptyText"
          stripe
          border
          @header-dragend="handleOrderColumnResize"
        >
          <el-table-column prop="record_type" label="类型" :width="orderColumnWidths.record_type" resizable>
            <template #default="scope">
              <el-tag 
                :type="scope.row.record_type === 'recharge' ? 'success' : 'primary'"
                size="small"
              >
                {{ scope.row.record_type === 'recharge' ? '充值' : '订单' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="display_no" label="订单号" :width="orderColumnWidths.display_no" resizable>
            <template #default="scope">
              <el-tag size="small" type="info">{{ scope.row.display_no }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="package_name" label="套餐名称/类型" :min-width="orderColumnWidths.package_name" resizable>
            <template #default="scope">
              {{ scope.row.package_name || (scope.row.record_type === 'recharge' ? '账户充值' : '-') }}
            </template>
          </el-table-column>
          <el-table-column prop="display_amount" label="金额" :width="orderColumnWidths.display_amount" resizable>
            <template #default="scope">
              <span 
                class="amount" 
                :class="{ 'positive': scope.row.record_type === 'recharge', 'negative': scope.row.record_type === 'order' }"
              >
                {{ scope.row.record_type === 'recharge' ? '+' : '-' }}¥{{ scope.row.display_amount }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="payment_method" label="支付方式" :width="orderColumnWidths.payment_method" resizable>
            <template #default="scope">
              <el-tag 
                :type="getPaymentMethodType(scope.row.payment_method)"
                size="small"
              >
                {{ getPaymentMethodText(scope.row.payment_method) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" :width="orderColumnWidths.status" resizable>
            <template #default="scope">
              <el-tag 
                :type="getOrderStatusType(scope.row.status)"
                size="small"
              >
                {{ getOrderStatusText(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" :width="orderColumnWidths.created_at" resizable>
            <template #default="scope">
              {{ formatDateTime(scope.row.created_at) }}
            </template>
          </el-table-column>
          <el-table-column prop="paid_at" label="支付时间" :width="orderColumnWidths.paid_at" resizable>
            <template #default="scope">
              {{ (scope.row.paid_at || scope.row.payment_time) ? formatDateTime(scope.row.paid_at || scope.row.payment_time) : '-' }}
            </template>
          </el-table-column>
          <el-table-column label="操作" :width="orderColumnWidths.actions" fixed="right" resizable>
            <template #default="scope">
              <div class="action-buttons" v-if="scope.row.record_type === 'order'">
                <el-button 
                  v-if="scope.row.status === 'pending'"
                  size="small" 
                  type="primary"
                  @click.stop="payOrder(scope.row)"
                  :disabled="false"
                >
                  立即支付
                </el-button>
                <el-button 
                  v-if="scope.row.status === 'pending'"
                  size="small" 
                  @click.stop="cancelOrder(scope.row)"
                  :disabled="false"
                >
                  取消订单
                </el-button>
                <el-button 
                  v-if="scope.row.status === 'paid'"
                  size="small" 
                  type="success"
                  @click.stop="viewOrderDetail(scope.row)"
                  :disabled="false"
                >
                  查看详情
                </el-button>
              </div>
              <div class="action-buttons" v-else-if="scope.row.record_type === 'recharge'">
                <el-button 
                  v-if="scope.row.status === 'pending'"
                  size="small" 
                  type="primary"
                  @click.stop="payRecharge(scope.row)"
                  :disabled="false"
                >
                  立即支付
                </el-button>
                <el-button 
                  v-if="scope.row.status === 'pending'"
                  size="small" 
                  @click.stop="cancelRecharge(scope.row)"
                  :disabled="false"
                >
                  取消充值
                </el-button>
              </div>
              <span v-else style="color: #909399; font-size: 12px;">-</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <div class="mobile-card-list" v-if="displayRecords.length > 0">
        <div 
          v-for="record in displayRecords" 
          :key="record.id || record.order_no"
          class="mobile-card"
          :class="{ 'recharge-card': record.record_type === 'recharge', 'order-card': record.record_type === 'order' }"
        >
          <div class="card-row">
            <span class="label">类型</span>
            <span class="value">
              <el-tag 
                :type="record.record_type === 'recharge' ? 'success' : 'primary'"
                size="small"
              >
                {{ record.record_type === 'recharge' ? '充值' : '订单' }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">订单号</span>
            <span class="value">
              <el-tag size="small" type="info">{{ record.display_no || record.order_no }}</el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">{{ record.record_type === 'recharge' ? '充值类型' : '套餐名称' }}</span>
            <span class="value">{{ record.package_name || (record.record_type === 'recharge' ? '账户充值' : '-') }}</span>
          </div>
          <div class="card-row">
            <span class="label">金额</span>
            <span 
              class="value amount" 
              :class="{ 'positive': record.record_type === 'recharge', 'negative': record.record_type === 'order' }"
            >
              {{ record.record_type === 'recharge' ? '+' : '-' }}¥{{ record.display_amount || record.amount }}
            </span>
          </div>
          <div class="card-row">
            <span class="label">支付方式</span>
            <span class="value">
              <el-tag 
                :type="getPaymentMethodType(record.payment_method)"
                size="small"
              >
                {{ getPaymentMethodText(record.payment_method) }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">状态</span>
            <span class="value">
              <el-tag 
                :type="getOrderStatusType(record.status)"
                size="small"
              >
                {{ getOrderStatusText(record.status) }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">创建时间</span>
            <span class="value">{{ formatDateTime(record.created_at) }}</span>
          </div>
          <div class="card-row" v-if="record.paid_at || record.payment_time">
            <span class="label">支付时间</span>
            <span class="value">{{ formatDateTime(record.paid_at || record.payment_time) }}</span>
          </div>
          <div class="card-actions" v-if="record.record_type === 'order'">
            <el-button 
              v-if="record.status === 'pending'"
              size="small" 
              type="primary"
              @click.stop="payOrder(record)"
              :disabled="false"
            >
              立即支付
            </el-button>
            <el-button 
              v-if="record.status === 'pending'"
              size="small" 
              @click.stop="cancelOrder(record)"
              :disabled="false"
            >
              取消订单
            </el-button>
            <el-button 
              v-if="record.status === 'paid'"
              size="small" 
              type="success"
              @click.stop="viewOrderDetail(record)"
              :disabled="false"
            >
              查看详情
            </el-button>
          </div>
          <div class="card-actions" v-else-if="record.record_type === 'recharge'">
            <el-button 
              v-if="record.status === 'pending'"
              size="small" 
              type="primary"
              @click.stop="payRecharge(record)"
              :disabled="false"
            >
              立即支付
            </el-button>
            <el-button 
              v-if="record.status === 'pending'"
              size="small" 
              @click.stop="cancelRecharge(record)"
              :disabled="false"
            >
              取消充值
            </el-button>
          </div>
        </div>
      </div>
      <div class="mobile-card-list" v-if="displayRecords.length === 0 && !isLoading && !isLoadingRecharges">
        <div class="empty-state">
          <i class="el-icon-document"></i>
          <p>{{ emptyText }}</p>
        </div>
      </div>
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.current"
          v-model:page-size="pagination.size"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
    <el-dialog
      v-model="detailDialogVisible"
      title="订单详情"
      :width="isMobile ? '90%' : '600px'"
      class="order-detail-dialog"
    >
      <div v-if="selectedOrder" class="order-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="订单号">{{ selectedOrder.order_no }}</el-descriptions-item>
          <el-descriptions-item label="套餐名称">{{ selectedOrder.package_name }}</el-descriptions-item>
          <el-descriptions-item label="订单金额">¥{{ selectedOrder.amount }}</el-descriptions-item>
          <el-descriptions-item label="支付方式">{{ getPaymentMethodText(selectedOrder.payment_method) }}</el-descriptions-item>
          <el-descriptions-item label="订单状态">
            <el-tag :type="getOrderStatusType(selectedOrder.status)">
              {{ getOrderStatusText(selectedOrder.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDateTime(selectedOrder.created_at) }}</el-descriptions-item>
          <el-descriptions-item v-if="selectedOrder.paid_at" label="支付时间" :span="2">
            {{ formatDateTime(selectedOrder.paid_at) }}
          </el-descriptions-item>
          <el-descriptions-item v-if="selectedOrder.payment_id" label="支付ID" :span="2">
            {{ selectedOrder.payment_id }}
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-dialog>
    <el-dialog
      v-model="paymentQRVisible"
      title="扫码支付"
      :width="isMobile ? '90%' : '500px'"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      class="payment-qr-dialog"
    >
      <div class="payment-qr-container">
        <div class="order-info">
          <h3>订单信息</h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="订单号">{{ selectedOrder?.order_no }}</el-descriptions-item>
            <el-descriptions-item label="套餐名称">{{ selectedOrder?.package_name }}</el-descriptions-item>
            <el-descriptions-item label="支付金额">
              <span class="amount">¥{{ parseFloat(selectedOrder?.amount || 0).toFixed(2) }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="支付方式">
              <el-tag type="primary">{{ getPaymentMethodName(paymentQRCode) }}</el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </div>
        <div class="qr-code-wrapper">
          <div v-if="paymentQRCode" class="qr-code">
            <img 
              :src="paymentQRCode.startsWith('data:') ? paymentQRCode : (paymentQRCode + '?t=' + Date.now())" 
              alt="支付二维码" 
              :title="getPaymentMethodName(paymentQRCode) + '二维码'"
              @error="onImageError"
              @load="onImageLoad"
            />
          </div>
          <div v-else class="qr-loading">
            <el-icon class="is-loading"><Loading /></el-icon>
            <p>正在生成二维码...</p>
          </div>
        </div>
        <div class="payment-tips">
          <el-alert
            title="支付提示"
            type="info"
            :closable="false"
            show-icon
          >
            <template #default>
              <p>1. 请使用{{ getPaymentMethodName(paymentQRCode) }}扫描上方二维码</p>
              <p>2. 确认订单信息无误后完成支付</p>
              <p>3. 支付完成后请勿关闭此窗口，系统将自动检测支付状态</p>
            </template>
          </el-alert>
        </div>
        <div class="payment-actions">
          <el-button 
            v-if="isMobile && paymentUrl && (selectedOrder?.payment_method === 'alipay' || selectedOrder?.payment_method_name === 'alipay' || paymentUrl.includes('alipay'))"
            type="success"
            size="large"
            @click="openAlipayApp"
            style="width: 100%; margin-bottom: 10px;"
          >
            <el-icon style="margin-right: 5px;"><Wallet /></el-icon>
            跳转到支付宝支付
          </el-button>
          <el-button 
            @click="checkPaymentStatus" 
            :loading="isCheckingPayment"
            type="primary"
            size="large"
            :style="isMobile ? 'width: 100%; margin-bottom: 10px;' : ''"
          >
            检查支付状态
          </el-button>
          <el-button 
            @click="closePaymentQR"
            size="large"
            :style="isMobile ? 'width: 100%;' : ''"
          >
            关闭
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>
<script>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Loading, Refresh, Wallet, ShoppingCart } from '@element-plus/icons-vue'
import { useApi, rechargeAPI, paymentAPI } from '@/utils/api'
import { formatDateTime } from '@/utils/date'
export default {
  name: 'Orders',
  components: {
    Loading,
    Refresh,
    Wallet,
    ShoppingCart
  },
  setup() {
    const api = useApi()
    const orders = ref([])
    const recharges = ref([])
    const allRecords = ref([])  // 合并的订单和充值记录
    const activeTab = ref('all')  // 'all', 'orders', 'recharges'
    const isLoading = ref(false)
    const isLoadingRecharges = ref(false)
    const orderStats = ref({
      total: 0,
      pending: 0,
      paid: 0,
      totalAmount: 0
    })
    const filters = reactive({
      status: '',
      payment_method: '',
      date_range: []
    })
    const pagination = reactive({
      current: 1,
      size: 20,
      total: 0
    })
    const detailDialogVisible = ref(false)
    const paymentQRVisible = ref(false)
    const selectedOrder = ref(null)
    const paymentQRCode = ref('')
    const paymentUrl = ref('')  // 存储原始支付URL，用于跳转支付宝App
    const isCheckingPayment = ref(false)
    let paymentStatusCheckInterval = null
    let paymentVisibilityHandler = null
    let paymentFocusHandler = null
    let paymentTimeoutId = null
    const orderTableRef = ref(null)
    const ORDER_TABLE_STORAGE_KEY = 'user_orders_table_settings'
    const orderColumnWidths = reactive({
      record_type: 80,
      display_no: 180,
      package_name: 160,
      display_amount: 120,
      payment_method: 120,
      status: 120,
      created_at: 180,
      paid_at: 180,
      actions: 200
    })
    const loadOrderTableSettings = () => {
      try {
        const saved = localStorage.getItem(ORDER_TABLE_STORAGE_KEY)
        if (saved) {
          const s = JSON.parse(saved)
          if (s.columnWidths) Object.assign(orderColumnWidths, s.columnWidths)
        }
      } catch (e) {
        console.warn('加载订单表设置失败:', e)
      }
    }
    const saveOrderTableSettings = () => {
      try {
        localStorage.setItem(ORDER_TABLE_STORAGE_KEY, JSON.stringify({ columnWidths: { ...orderColumnWidths } }))
      } catch (e) {
        console.warn('保存订单表设置失败:', e)
      }
    }
    const ORDER_COLUMN_KEYS = ['record_type', 'display_no', 'package_name', 'display_amount', 'payment_method', 'status', 'created_at', 'paid_at', 'actions']
    let orderResizeTimer = null
    const handleOrderColumnResize = () => {
      if (orderResizeTimer) clearTimeout(orderResizeTimer)
      orderResizeTimer = setTimeout(() => {
        if (orderTableRef.value && orderTableRef.value.$el) {
          const cells = orderTableRef.value.$el.querySelectorAll('.el-table__header-wrapper thead th')
          cells.forEach((cell, index) => {
            if (ORDER_COLUMN_KEYS[index] && cell.offsetWidth > 0) orderColumnWidths[ORDER_COLUMN_KEYS[index]] = cell.offsetWidth
          })
          saveOrderTableSettings()
        }
      }, 300)
    }
    const windowWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1920)
    const isMobile = computed(() => {
      return windowWidth.value <= 768
    })
    const handleResize = () => {
      if (typeof window !== 'undefined') {
        windowWidth.value = window.innerWidth
      }
    }
    const emptyText = computed(() => {
      if (isLoading.value) return '加载中...'
      if (filters.status || filters.payment_method || filters.date_range.length > 0) {
        return '没有找到符合条件的订单'
      }
      return '暂无订单记录'
    })
    const loadRecharges = async () => {
      try {
        isLoadingRecharges.value = true
        const params = {
          page: pagination.current,
          size: pagination.size
        }
        if (filters.date_range && filters.date_range.length === 2) {
          params.start_date = filters.date_range[0]
          params.end_date = filters.date_range[1]
        }
        const response = await rechargeAPI.getRecharges(params)
        if (response.data && response.data.success) {
          const data = response.data.data
          const rechargeList = Array.isArray(data) ? data : []
          recharges.value = rechargeList.map(recharge => ({
            ...recharge,
            id: recharge.id,
            order_no: recharge.order_no,
            amount: recharge.amount,
            status: recharge.status || 'pending',
            payment_method: recharge.payment_method || 'alipay',
            created_at: recharge.created_at,
            paid_at: recharge.paid_at
          }))
        } else {
          recharges.value = []
        }
      } catch (error) {
        recharges.value = []
      } finally {
        isLoadingRecharges.value = false
      }
    }
    const mergeRecords = () => {
      const merged = []
      orders.value.forEach(order => {
        merged.push({
          ...order,
          record_type: 'order',
          display_no: order.order_no,
          display_amount: order.amount,
          display_type: '消费'
        })
      })
      recharges.value.forEach(recharge => {
        let paymentMethod = recharge.payment_method || 'alipay'
        if (paymentMethod && typeof paymentMethod === 'object') {
          if (paymentMethod.String) {
            paymentMethod = paymentMethod.String
          } else if (paymentMethod.payment_method) {
            paymentMethod = paymentMethod.payment_method
          } else {
            const values = Object.values(paymentMethod).filter(v => typeof v === 'string' && v.length > 0)
            paymentMethod = values.length > 0 ? values[0] : 'alipay'
          }
        }
        merged.push({
          ...recharge,
          record_type: 'recharge',
          display_no: recharge.order_no,
          display_amount: recharge.amount,
          display_type: '充值',
          package_name: '账户充值',
          payment_method: paymentMethod,
          status: recharge.status
        })
      })
      merged.sort((a, b) => {
        const timeA = new Date(a.created_at || a.paid_at || 0).getTime()
        const timeB = new Date(b.created_at || b.paid_at || 0).getTime()
        return timeB - timeA
      })
      allRecords.value = merged
    }
    const formatOrderRecord = (order) => {
      return {
        ...order,
        record_type: 'order',
        display_no: order.order_no || '-',
        display_amount: order.amount || '0.00',
        package_name: order.package_name || '-',
        display_type: '消费',
        id: order.id,
        order_no: order.order_no,
        amount: order.amount,
        status: order.status,
        payment_method: order.payment_method || order.payment_method_name,
        payment_method_id: order.payment_method_id, // 保留支付方式ID
        created_at: order.created_at,
        paid_at: order.paid_at || order.payment_time
      }
    }
    const formatRechargeRecord = (recharge) => {
      let paymentMethod = recharge.payment_method || 'alipay'
      if (paymentMethod && typeof paymentMethod === 'object') {
        if (paymentMethod.String) {
          paymentMethod = paymentMethod.String
        } else if (paymentMethod.payment_method) {
          paymentMethod = paymentMethod.payment_method
        } else {
          const values = Object.values(paymentMethod).filter(v => typeof v === 'string' && v.length > 0)
          paymentMethod = values.length > 0 ? values[0] : 'alipay'
        }
      }
      return {
        ...recharge,
        record_type: 'recharge',
        display_no: recharge.order_no || '-',
        display_amount: recharge.amount || '0.00',
        package_name: '账户充值',
        payment_method: paymentMethod,
        status: recharge.status || 'pending',
        display_type: '充值',
        id: recharge.id,
        order_no: recharge.order_no,
        amount: recharge.amount,
        status: recharge.status || 'pending',
        payment_method: recharge.payment_method || 'alipay',
        created_at: recharge.created_at,
        paid_at: recharge.paid_at
      }
    }
    const displayRecords = computed(() => {
      if (activeTab.value === 'orders') {
        return orders.value.map(formatOrderRecord)
      } else if (activeTab.value === 'recharges') {
        return recharges.value.map(formatRechargeRecord)
      } else {
        return allRecords.value
      }
    })
    const handleTabChange = (tabName) => {
      if (tabName === 'recharges') {
        if (recharges.value.length === 0) {
          loadRecharges()
        }
      } else if (tabName === 'orders') {
        if (orders.value.length === 0) {
          loadOrders()
        }
      } else if (tabName === 'all') {
        if (recharges.value.length === 0) {
          loadRecharges().then(() => {
            mergeRecords()
          })
        } else {
          mergeRecords()
        }
      }
    }
    const loadOrders = async () => {
      try {
        isLoading.value = true
        const params = {
          page: pagination.current,
          size: pagination.size,
          ...filters
        }
        if (filters.date_range && filters.date_range.length === 2) {
          params.start_date = filters.date_range[0]
          params.end_date = filters.date_range[1]
        }
        const response = await api.get('/orders/', { params })
        const orderList = response.data.data?.orders || []
        orders.value = orderList.map(order => ({
          ...order,
          order_no: order.order_no,
          amount: order.amount,
          package_name: order.package_name,
          status: order.status,
          payment_method: order.payment_method || order.payment_method_name,
          payment_method_id: order.payment_method_id, // 保留支付方式ID
          created_at: order.created_at,
          paid_at: order.paid_at || order.payment_time
        }))
        pagination.total = response.data.data?.total || response.data.total || 0
        await loadOrderStats()
        if (activeTab.value === 'all') {
          await loadRecharges()
          mergeRecords()
        }
      } catch (error) {
        ElMessage.error('加载订单列表失败: ' + (error.response?.data?.message || error.message))
        } finally {
        isLoading.value = false
      }
    }
    const loadOrderStats = async () => {
      try {
        const response = await api.get('/orders/stats')
        let statsData = null
        if (response && response.data) {
          if (response.data.success !== false && response.data.data) {
            statsData = response.data.data
          } else if (response.data.total !== undefined || response.data.total_orders !== undefined) {
            statsData = response.data
          } else if (response.data.data && typeof response.data.data === 'object') {
            statsData = response.data.data
          }
        }
        if (statsData) {
          const getValue = (primary, ...alternatives) => {
            if (primary !== undefined && primary !== null) return primary
            for (const alt of alternatives) {
              if (alt !== undefined && alt !== null) return alt
            }
            return 0
          }
          const toNumber = (val) => {
            const num = Number(val)
            return isNaN(num) ? 0 : num
          }
          const total = getValue(statsData.total, statsData.total_orders, statsData.totalOrders)
          const pending = getValue(statsData.pending, statsData.pending_orders, statsData.pendingOrders)
          const paid = getValue(statsData.paid, statsData.paid_orders, statsData.paidOrders)
          const cancelled = getValue(statsData.cancelled, statsData.cancelled_orders, statsData.cancelledOrders)
          const totalAmount = getValue(statsData.totalAmount, statsData.total_amount)
          orderStats.value = {
            total: toNumber(total),
            pending: toNumber(pending),
            paid: toNumber(paid),
            cancelled: toNumber(cancelled),
            totalAmount: toNumber(totalAmount)
          }
        } else {
          orderStats.value = {
            total: 0,
            pending: 0,
            paid: 0,
            cancelled: 0,
            totalAmount: 0
          }
        }
      } catch (error) {
        orderStats.value = {
          total: 0,
          pending: 0,
          paid: 0,
          cancelled: 0,
          totalAmount: 0
        }
      }
    }
    const applyFilters = () => {
      pagination.current = 1
      loadOrders()
    }
    const resetFilters = () => {
      filters.status = ''
      filters.payment_method = ''
      filters.date_range = []
      pagination.current = 1
      loadOrders()
    }
    const refreshOrders = async () => {
      await loadOrders() // loadOrders 内部会调用 loadOrderStats
      if (activeTab.value === 'all' || activeTab.value === 'recharges') {
        await loadRecharges()
        if (activeTab.value === 'all') {
          mergeRecords()
        }
      }
      await loadOrderStats()
    }
    const handleSizeChange = (size) => {
      pagination.size = size
      pagination.current = 1
      loadOrders()
    }
    const handleCurrentChange = (page) => {
      pagination.current = page
      loadOrders()
    }
    const payOrder = async (order) => {
      try {
        const orderNo = order.order_no || order.display_no
        if (!orderNo) {
          ElMessage.error('订单号不存在，无法支付')
          return
        }
        let paymentMethodId = order.payment_method_id
        if (!paymentMethodId && order.payment_method) {
          try {
            const paymentMethodsResponse = await paymentAPI.getPaymentMethods()
            const paymentMethods = paymentMethodsResponse.data?.data || paymentMethodsResponse.data || []
            const paymentMethodMap = {
              'alipay': 'alipay',
              '支付宝': 'alipay',
              'wechat': 'wechat',
              '微信支付': 'wechat',
              'weixin': 'wechat'
            }
            const methodKey = paymentMethodMap[order.payment_method] || order.payment_method
            const matchedMethod = paymentMethods.find(m => 
              m.key === methodKey || 
              m.name === order.payment_method ||
              m.pay_type === methodKey
            )
            if (matchedMethod) {
              paymentMethodId = matchedMethod.id
            } else if (paymentMethods.length > 0) {
              paymentMethodId = paymentMethods[0].id
            }
          } catch (error) {
          }
        }
        if (!paymentMethodId) {
          ElMessage.error('无法确定支付方式，请刷新页面后重试')
          return
        }
        const response = await api.post(`/orders/${orderNo}/pay`, {
          payment_method_id: paymentMethodId
        })
        if (response.data && response.data.success !== false) {
          const paymentUrl = response.data.data?.payment_url || response.data.data?.payment_qr_code
          if (paymentUrl) {
            await showPaymentQR({
              ...order,
              order_no: orderNo,
              payment_method_id: paymentMethodId
            }, paymentUrl)
          } else {
            const errorMsg = response.data.message || response.data.detail || '支付链接生成失败'
            ElMessage.error(errorMsg)
          }
        } else {
          const errorMsg = response.data?.message || response.data?.detail || '创建支付订单失败'
          ElMessage.error(errorMsg)
        }
      } catch (error) {
        const errorMsg = error.response?.data?.detail || 
                        error.response?.data?.message || 
                        error.message || 
                        '创建支付订单失败，请重试'
        ElMessage.error(errorMsg)
      }
    }
    const openAlipayApp = () => {
      if (!paymentUrl.value) {
        ElMessage.error('支付链接不存在')
        return
      }
      const alipayAppUrl = `alipays://platformapi/startapp?saId=10000007&qrcode=${encodeURIComponent(paymentUrl.value)}`
      try {
        const handleVisibilityChange = async () => {
          if (document.visibilityState === 'visible' && paymentQRVisible.value) {
            await checkPaymentStatus()
            document.removeEventListener('visibilitychange', handleVisibilityChange)
          }
        }
        document.addEventListener('visibilitychange', handleVisibilityChange)
        const handleFocus = async () => {
          if (paymentQRVisible.value) {
            await checkPaymentStatus()
            window.removeEventListener('focus', handleFocus)
          }
        }
        window.addEventListener('focus', handleFocus)
        window.location.href = alipayAppUrl
        setTimeout(() => {
          ElMessage.info('如果未跳转到支付宝，请使用支付宝扫描上方二维码完成支付')
        }, 3000)
      } catch (error) {
        ElMessage.error('跳转失败，请使用支付宝扫描二维码完成支付')
      }
    }
    const showPaymentQR = async (order, url) => {
      if (!url) {
        ElMessage.error('支付链接生成失败，请重试')
        return
      }
      paymentUrl.value = url
      selectedOrder.value = order
      const paymentMethod = order.payment_method_name || order.payment_method || 'alipay'
      if (paymentMethod === 'alipay') {
        if (url.startsWith('http://') || url.startsWith('https://')) {
          try {
            const QRCode = await import('qrcode')
            const qrCodeDataURL = await QRCode.toDataURL(url, {
              width: 256,
              margin: 2,
              color: {
                dark: '#000000',
                light: '#FFFFFF'
              },
              errorCorrectionLevel: 'M'
            })
            paymentQRCode.value = qrCodeDataURL
          } catch (error) {
            ElMessage.error('生成二维码失败，请刷新页面重试')
            return
          }
        } else {
          ElMessage.error('支付宝二维码格式错误，请联系管理员检查配置')
          return
        }
      } else {
        if (url.startsWith('http://') || url.startsWith('https://')) {
          try {
            const QRCode = await import('qrcode')
            const qrCodeDataURL = await QRCode.toDataURL(url, {
              width: 256,
              margin: 2,
              color: {
                dark: '#000000',
                light: '#FFFFFF'
              },
              errorCorrectionLevel: 'M'
            })
            paymentQRCode.value = qrCodeDataURL
          } catch (error) {
            ElMessage.error('生成二维码失败，请刷新页面重试')
            return
          }
        } else {
          try {
            const QRCode = await import('qrcode')
            const qrCodeDataURL = await QRCode.toDataURL(url, {
              width: 256,
              margin: 2,
              color: {
                dark: '#000000',
                light: '#FFFFFF'
              },
              errorCorrectionLevel: 'M'
            })
            paymentQRCode.value = qrCodeDataURL
          } catch (error) {
            ElMessage.error('生成二维码失败，请刷新页面重试')
            return
          }
        }
      }
      selectedOrder.value = {
        ...order,
        order_no: order.order_no || order.display_no,
        package_name: order.package_name || '账户充值',
        amount: order.amount || order.display_amount,
        payment_method: order.payment_method || order.payment_method_name || 'alipay'
      }
      paymentQRVisible.value = true
      await new Promise(resolve => setTimeout(resolve, 200))
      startPaymentStatusCheck()
    }
    const getPaymentMethodName = (paymentUrl) => {
      if (paymentUrl.includes('qr.alipay.com')) {
        return '支付宝'
      } else if (paymentUrl.includes('weixin') || paymentUrl.includes('wechat')) {
        return '微信'
      }
      return '支付'
    }
    const onImageLoad = () => {
      }
    const onImageError = async (event) => {
      if (paymentQRCode.value && paymentQRCode.value.startsWith('data:')) {
        ElMessage.warning('二维码显示异常，正在重新生成...')
        if (selectedOrder.value) {
          try {
            let paymentMethodId = selectedOrder.value.payment_method_id
            if (!paymentMethodId) {
              try {
                const paymentMethodsResponse = await paymentAPI.getPaymentMethods()
                const paymentMethods = paymentMethodsResponse.data?.data || paymentMethodsResponse.data || []
                const paymentMethodMap = {
                  'alipay': 'alipay',
                  '支付宝': 'alipay',
                  'wechat': 'wechat',
                  '微信支付': 'wechat',
                  'weixin': 'wechat'
                }
                const methodKey = paymentMethodMap[selectedOrder.value.payment_method] || selectedOrder.value.payment_method
                const matchedMethod = paymentMethods.find(m => 
                  m.key === methodKey || 
                  m.name === selectedOrder.value.payment_method ||
                  m.pay_type === methodKey
                )
                if (matchedMethod) {
                  paymentMethodId = matchedMethod.id
                } else if (paymentMethods.length > 0) {
                  paymentMethodId = paymentMethods[0].id
                }
              } catch (error) {
              }
            }
            if (!paymentMethodId) {
              ElMessage.error('无法确定支付方式，请刷新页面后重试')
              return
            }
            const response = await api.post(`/orders/${selectedOrder.value.order_no}/pay`, {
              payment_method_id: paymentMethodId
            })
            const paymentUrl = response.data.data?.payment_url || response.data.data?.payment_qr_code
            if (paymentUrl) {
              const QRCode = await import('qrcode')
              const qrCodeDataURL = await QRCode.toDataURL(paymentUrl, {
                width: 256,
                margin: 2,
                color: {
                  dark: '#000000',
                  light: '#FFFFFF'
                },
                errorCorrectionLevel: 'M'
              })
              paymentQRCode.value = qrCodeDataURL
              event.target.src = qrCodeDataURL
              }
          } catch (error) {
            ElMessage.error('二维码生成失败，请刷新页面后重试')
          }
        }
      }
    }
    const startPaymentStatusCheck = () => {
      // 清理之前的资源
      if (paymentStatusCheckInterval) {
        clearInterval(paymentStatusCheckInterval)
        paymentStatusCheckInterval = null
      }
      if (paymentVisibilityHandler) {
        document.removeEventListener('visibilitychange', paymentVisibilityHandler)
        paymentVisibilityHandler = null
      }
      if (paymentFocusHandler) {
        window.removeEventListener('focus', paymentFocusHandler)
        paymentFocusHandler = null
      }
      if (paymentTimeoutId) {
        clearTimeout(paymentTimeoutId)
        paymentTimeoutId = null
      }

      // 立即检查一次
      checkPaymentStatus()

      // 启动定时检查
      paymentStatusCheckInterval = setInterval(async () => {
        await checkPaymentStatus()
      }, 2000)

      // 添加页面可见性监听
      paymentVisibilityHandler = async () => {
        if (document.visibilityState === 'visible' && paymentQRVisible.value) {
          await checkPaymentStatus()
        }
      }
      document.addEventListener('visibilitychange', paymentVisibilityHandler)

      // 添加窗口焦点监听
      paymentFocusHandler = async () => {
        if (paymentQRVisible.value) {
          await checkPaymentStatus()
        }
      }
      window.addEventListener('focus', paymentFocusHandler)

      // 30分钟后自动清理
      paymentTimeoutId = setTimeout(() => {
        if (paymentStatusCheckInterval) {
          clearInterval(paymentStatusCheckInterval)
          paymentStatusCheckInterval = null
        }
        if (paymentVisibilityHandler) {
          document.removeEventListener('visibilitychange', paymentVisibilityHandler)
          paymentVisibilityHandler = null
        }
        if (paymentFocusHandler) {
          window.removeEventListener('focus', paymentFocusHandler)
          paymentFocusHandler = null
        }
      }, 30 * 60 * 1000)
    }
    const checkPaymentStatus = async () => {
      if (!selectedOrder.value) return
      try {
        isCheckingPayment.value = true
        const isRecharge = selectedOrder.value.record_type === 'recharge' || selectedOrder.value.id && !selectedOrder.value.order_no
        if (isRecharge) {
          const response = await rechargeAPI.getRechargeDetail(selectedOrder.value.id)
          const rechargeData = response.data.data
          if (rechargeData.status === 'paid') {
            if (paymentStatusCheckInterval) {
              clearInterval(paymentStatusCheckInterval)
              paymentStatusCheckInterval = null
            }
            paymentQRVisible.value = false
            ElMessage.success('支付成功！')
            await loadRecharges()
            if (activeTab.value === 'all') {
              mergeRecords()
            }
          } else if (rechargeData.status === 'cancelled') {
            if (paymentStatusCheckInterval) {
              clearInterval(paymentStatusCheckInterval)
              paymentStatusCheckInterval = null
            }
            paymentQRVisible.value = false
            ElMessage.info('充值订单已取消')
            await loadRecharges()
            if (activeTab.value === 'all') {
              mergeRecords()
            }
          }
        } else {
          if (!selectedOrder.value.order_no) {
            return
          }
          const response = await api.get(`/orders/${selectedOrder.value.order_no}/status`)
          if (!response || !response.data || response.data.success === false) {
            return
          }
          const orderData = response.data.data
          if (!orderData) {
            return
          }
          if (orderData.status === 'paid') {
            if (paymentStatusCheckInterval) {
              clearInterval(paymentStatusCheckInterval)
              paymentStatusCheckInterval = null
            }
            paymentQRVisible.value = false
            ElMessage.success('支付成功！')
            loadOrders()
            if (activeTab.value === 'all') {
              await loadRecharges()
              mergeRecords()
            }
          } else if (orderData.status === 'cancelled') {
            if (paymentStatusCheckInterval) {
              clearInterval(paymentStatusCheckInterval)
              paymentStatusCheckInterval = null
            }
            paymentQRVisible.value = false
            ElMessage.info('订单已取消')
            loadOrders()
            if (activeTab.value === 'all') {
              await loadRecharges()
              mergeRecords()
            }
          }
        }
      } catch (error) {
        } finally {
        isCheckingPayment.value = false
      }
    }
    const closePaymentQR = () => {
      if (paymentStatusCheckInterval) {
        clearInterval(paymentStatusCheckInterval)
        paymentStatusCheckInterval = null
      }
      if (paymentVisibilityHandler) {
        document.removeEventListener('visibilitychange', paymentVisibilityHandler)
        paymentVisibilityHandler = null
      }
      if (paymentFocusHandler) {
        window.removeEventListener('focus', paymentFocusHandler)
        paymentFocusHandler = null
      }
      if (paymentTimeoutId) {
        clearTimeout(paymentTimeoutId)
        paymentTimeoutId = null
      }
      paymentQRVisible.value = false
      paymentQRCode.value = ''
      selectedOrder.value = null
    }
    const cancelOrder = async (order) => {
      try {
        await ElMessageBox.confirm(
          '确定要取消这个订单吗？取消后无法恢复。',
          '确认取消',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        await api.post(`/orders/${order.order_no}/cancel`)
        ElMessage.success('订单已取消')
        loadOrders()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('取消订单失败')
          }
      }
    }
    const payRecharge = async (recharge) => {
      try {
        const response = await rechargeAPI.getRechargeDetail(recharge.id)
        if (response.data && response.data.success) {
          const rechargeData = response.data.data
          const paymentUrl = rechargeData.payment_url || rechargeData.payment_qr_code
          if (paymentUrl) {
            showPaymentQR({
              ...recharge,
              order_no: recharge.order_no,
              package_name: '账户充值',
              amount: recharge.amount
            }, paymentUrl)
          } else {
            ElMessage.warning('支付链接不存在，正在重新生成...')
            const errorMsg = response.data.message || '支付链接生成失败'
            ElMessage.error(errorMsg)
          }
        } else {
          const errorMsg = response.data?.message || response.data?.detail || '获取充值详情失败'
          ElMessage.error(errorMsg)
        }
      } catch (error) {
        const errorMsg = error.response?.data?.detail || 
                        error.response?.data?.message || 
                        error.message || 
                        '获取充值支付链接失败，请重试'
        ElMessage.error(errorMsg)
      }
    }
    const cancelRecharge = async (recharge) => {
      try {
        const rechargeId = recharge.id
        if (!rechargeId) {
          ElMessage.error('充值记录ID不存在，无法取消')
          return
        }
        await ElMessageBox.confirm(
          '确定要取消这个充值订单吗？取消后无法恢复。',
          '确认取消',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        await rechargeAPI.cancelRecharge(rechargeId)
        ElMessage.success('充值订单已取消')
        await loadRecharges()
        if (activeTab.value === 'all') {
          mergeRecords()
        }
        if (activeTab.value === 'orders') {
          await loadOrders()
        }
      } catch (error) {
        if (error !== 'cancel') {
          const errorMsg = error.response?.data?.detail || 
                          error.response?.data?.message || 
                          '取消充值订单失败'
          ElMessage.error(errorMsg)
          }
      }
    }
    const viewOrderDetail = (order) => {
      try {
        if (!order) {
          ElMessage.error('订单信息不存在')
          return
        }
        if (!order.order_no && !order.display_no) {
          ElMessage.error('订单号不存在')
          return
        }
        selectedOrder.value = {
          ...order,
          order_no: order.order_no || order.display_no,
          package_name: order.package_name || '-',
          amount: order.amount || order.display_amount,
          payment_method: order.payment_method || order.payment_method_name,
          status: order.status,
          created_at: order.created_at,
          paid_at: order.paid_at || order.payment_time
        }
        detailDialogVisible.value = true
        } catch (error) {
        ElMessage.error('查看订单详情失败')
      }
    }
    const getOrderStatusType = (status) => {
      const statusMap = {
        pending: 'warning',
        paid: 'success',
        cancelled: 'info',
        failed: 'danger'
      }
      return statusMap[status] || 'info'
    }
    const getOrderStatusText = (status) => {
      const statusMap = {
        pending: '待支付',
        paid: '已支付',
        cancelled: '已取消',
        failed: '支付失败'
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
    const getPaymentMethodType = (method) => {
      const typeMap = {
        alipay: 'primary',
        wechat: 'success',
        balance: 'warning',
        mixed: 'info'
      }
      return typeMap[method] || 'info'
    }
    const formatAmount = (amount) => {
      if (amount === null || amount === undefined || amount === '') return '0.00'
      const num = typeof amount === 'string' ? parseFloat(amount) : amount
      if (isNaN(num)) return '0.00'
      return num.toFixed(2)
    }
    onMounted(async () => {
      loadOrderTableSettings()
      await loadOrderStats()
      await loadOrders()
      if (activeTab.value === 'all') {
        await loadRecharges()
        mergeRecords()
      }
      if (typeof window !== 'undefined') {
        windowWidth.value = window.innerWidth
        window.addEventListener('resize', handleResize)
      }
    })
    onUnmounted(() => {
      // 清理 resize 监听器
      if (typeof window !== 'undefined') {
        window.removeEventListener('resize', handleResize)
      }
      // 清理支付状态检查相关资源
      if (paymentStatusCheckInterval) {
        clearInterval(paymentStatusCheckInterval)
        paymentStatusCheckInterval = null
      }
      if (paymentVisibilityHandler) {
        document.removeEventListener('visibilitychange', paymentVisibilityHandler)
        paymentVisibilityHandler = null
      }
      if (paymentFocusHandler) {
        window.removeEventListener('focus', paymentFocusHandler)
        paymentFocusHandler = null
      }
      if (paymentTimeoutId) {
        clearTimeout(paymentTimeoutId)
        paymentTimeoutId = null
      }
    })
    return {
      orders,
      recharges,
      allRecords,
      displayRecords,
      activeTab,
      isLoading,
      isLoadingRecharges,
      orderStats,
      filters,
      pagination,
      detailDialogVisible,
      paymentQRVisible,
      selectedOrder,
      paymentQRCode,
      paymentUrl,
      openAlipayApp,
      isCheckingPayment,
      emptyText,
      loadOrders,
      loadRecharges,
      mergeRecords,
      handleTabChange,
      loadOrderStats,
      applyFilters,
      resetFilters,
      refreshOrders,
      handleSizeChange,
      handleCurrentChange,
      payOrder,
      showPaymentQR,
      checkPaymentStatus,
      closePaymentQR,
      onImageLoad,
      onImageError,
      cancelOrder,
      payRecharge,
      cancelRecharge,
      viewOrderDetail,
      getOrderStatusType,
      getOrderStatusText,
      getPaymentMethodType,
      getPaymentMethodText,
      formatAmount,
      getPaymentMethodName,
      formatDateTime,
      isMobile,
      orderTableRef,
      orderColumnWidths,
      handleOrderColumnResize
    }
  }
}
</script>
<style scoped lang="scss">
@use '@/styles/list-common.scss';
.amount {
  color: #f56c6c;
  font-weight: bold;
  &.positive {
    color: #67c23a;
  }
  &.negative {
    color: #f56c6c;
  }
}
.action-buttons {
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  gap: 8px;
  align-items: center;
  position: relative;
  z-index: 100;
  min-height: 40px;
  white-space: nowrap;
  .el-button {
    position: relative;
    z-index: 101;
    pointer-events: auto !important;
    cursor: pointer !important;
    min-height: 32px;
    padding: 8px 12px;
    line-height: 1;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    vertical-align: middle;
    flex-shrink: 0;
    white-space: nowrap;
    :deep(*) {
      pointer-events: none;
    }
    &::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      z-index: 1;
      pointer-events: auto;
    }
    :deep(span) {
      position: relative;
      z-index: 2;
      white-space: nowrap;
    }
    &:hover {
      z-index: 102;
    }
  }
}
:deep(.el-table__fixed-right) {
  .action-buttons {
    z-index: 100;
    .el-button {
      z-index: 101;
    }
  }
}
:deep(.el-table__fixed) {
  .action-buttons {
    z-index: 100;
    .el-button {
      z-index: 101;
    }
  }
}
:deep(.el-table__body-wrapper) {
  .el-table__cell {
    .action-buttons {
      position: relative;
      z-index: 100;
      .el-button {
        position: relative;
        z-index: 101;
      }
    }
  }
}
.records-tabs {
  margin-bottom: 20px;
  :deep(.el-tabs__header) {
    margin-bottom: 0;
  }
  :deep(.el-tabs__item) {
    font-size: 14px;
    padding: 0 20px;
    .el-icon {
      margin-right: 5px;
    }
  }
}
.mobile-card {
  &.recharge-card {
    border-left: 4px solid #67c23a;
  }
  &.order-card {
    border-left: 4px solid #409eff;
  }
  .card-actions {
    display: flex;
    gap: 8px;
    margin-top: 12px;
    position: relative;
    z-index: 10;
    .el-button {
      position: relative;
      z-index: 11;
      pointer-events: auto !important;
      cursor: pointer !important;
      flex: 1;
      min-height: 44px; /* 增加最小高度，方便手机端点击 */
      padding: 10px 15px;
      line-height: 1.5;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      touch-action: manipulation; /* 优化移动端触摸 */
      -webkit-tap-highlight-color: rgba(0, 0, 0, 0.1); /* 添加点击反馈 */
      :deep(*) {
        pointer-events: auto; /* 改为auto，允许点击 */
      }
      &:hover, &:active {
        z-index: 12;
      }
    }
  }
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
  }
}
.order-detail {
  padding: 20px 0;
}
.dialog-footer {
  text-align: right;
  .el-button {
    margin-left: 10px;
  }
}
.payment-qr-container {
  text-align: center;
}
.payment-qr-container .order-info {
  margin-bottom: 20px;
}
.payment-qr-container .order-info h3 {
  margin-bottom: 15px;
  color: #303133;
  font-size: 16px;
}
.payment-qr-container .amount {
  color: #f56c6c;
  font-size: 18px;
  font-weight: bold;
}
.qr-code-wrapper {
  margin: 20px 0;
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 200px;
}
.qr-code img {
  max-width: 200px;
  max-height: 200px;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
}
.qr-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  color: #909399;
}
.qr-loading .el-icon {
  font-size: 24px;
  margin-bottom: 10px;
}
.payment-tips {
  margin: 20px 0;
}
.payment-actions {
  margin-top: 20px;
}
.payment-actions .el-button {
  margin: 0 10px;
}
.order-detail-dialog,
.payment-qr-dialog {
  :deep(.el-dialog) {
    @media (max-width: 768px) {
      width: 90% !important;
      margin: 5vh auto !important;
      max-height: 90vh;
      overflow-y: auto;
    }
  }
  :deep(.el-dialog__body) {
    @media (max-width: 768px) {
      padding: 15px !important;
      max-height: calc(90vh - 120px);
      overflow-y: auto;
    }
  }
  :deep(.el-dialog__header) {
    @media (max-width: 768px) {
      padding: 15px !important;
    }
  }
  :deep(.el-dialog__footer) {
    @media (max-width: 768px) {
      padding: 15px !important;
    }
  }
}
.order-detail {
  @media (max-width: 768px) {
    padding: 10px 0;
    :deep(.el-descriptions) {
      font-size: 14px;
    }
    :deep(.el-descriptions__label) {
      width: 35% !important;
      font-size: 13px;
    }
    :deep(.el-descriptions__content) {
      width: 65% !important;
      font-size: 13px;
    }
  }
}
.payment-qr-container {
  @media (max-width: 768px) {
    .order-info {
      margin-bottom: 15px;
      :is(h3) {
        font-size: 14px;
        margin-bottom: 10px;
      }
      :deep(.el-descriptions) {
        font-size: 13px;
      }
      :deep(.el-descriptions__label) {
        width: 40% !important;
        font-size: 12px;
      }
      :deep(.el-descriptions__content) {
        width: 60% !important;
        font-size: 12px;
      }
    }
    .qr-code-wrapper {
      margin: 15px 0;
      min-height: 180px;
      .qr-code img {
        max-width: 180px;
        max-height: 180px;
      }
    }
    .payment-tips {
      margin: 15px 0;
      :deep(.el-alert) {
        font-size: 12px;
        padding: 10px;
      }
    }
    .payment-actions {
      margin-top: 15px;
      display: flex;
      flex-direction: column;
      gap: 10px;
      .el-button {
        width: 100%;
        margin: 0 !important;
        min-height: 44px;
        font-size: 16px;
      }
    }
  }
}
.filter-card {
  padding: 12px;
  margin-bottom: 12px;
  @media (max-width: 768px) {
    padding: 10px;
    margin-bottom: 10px;
  }
}
.filter-desktop {
  display: block;
  @media (max-width: 768px) {
    display: none;
  }
}
.filter-mobile {
  display: none;
  @media (max-width: 768px) {
    display: block;
  }
  .filter-row {
    margin-bottom: 6px;
    &:last-of-type {
      margin-bottom: 8px;
    }
    .filter-select,
    .filter-date {
      width: 100%;
      :deep(.el-input__wrapper) {
        border-radius: 6px;
        min-height: 36px;
        padding: 0 10px;
        font-size: 13px;
        box-shadow: none;
        border: 1px solid #dcdfe6;
        background-color: #fff;
        &:hover {
          border-color: #c0c4cc;
        }
        &.is-focus {
          border-color: #409eff;
        }
      }
      :deep(.el-input__inner) {
        font-size: 13px;
        height: 36px;
        line-height: 36px;
        color: #606266;
        &::placeholder {
          color: #c0c4cc;
          font-size: 13px;
        }
      }
      :deep(.el-input__suffix) {
        .el-input__suffix-inner {
          .el-icon {
            font-size: 14px;
            color: #c0c4cc;
          }
        }
      }
    }
    .filter-date {
      :deep(.el-range-input) {
        font-size: 13px;
        color: #606266;
        &::placeholder {
          color: #c0c4cc;
          font-size: 13px;
        }
      }
      :deep(.el-range-separator) {
        font-size: 13px;
        color: #606266;
        padding: 0 4px;
      }
    }
  }
  .filter-actions {
    display: flex;
    gap: 6px;
    margin-top: 6px;
    .filter-btn {
      flex: 1;
      min-height: 36px;
      font-size: 13px;
      border-radius: 6px;
      font-weight: 500;
      margin: 0;
      padding: 8px 12px;
      border: 1px solid;
      transition: all 0.2s;
      &:first-child {
        flex: 1.3;
        background-color: #409eff;
        border-color: #409eff;
        color: #fff;
        &:active {
          background-color: #3a8ee6;
          border-color: #3a8ee6;
          transform: scale(0.98);
        }
      }
      &:last-child {
        background-color: #fff;
        border-color: #dcdfe6;
        color: #606266;
        &:active {
          background-color: #f5f7fa;
          border-color: #c0c4cc;
          transform: scale(0.98);
        }
      }
    }
  }
}
.stats-row {
  @media (max-width: 480px) {
    grid-template-columns: 1fr !important;
    .stat-card {
      .stat-number {
        font-size: 1.75rem;
      }
      .stat-label {
        font-size: 0.85rem;
      }
    }
  }
}
.pagination {
  @media (max-width: 768px) {
    :deep(.el-pagination) {
      .el-pagination__sizes,
      .el-pagination__jump {
        display: none;
      }
      .el-pagination__total {
        display: none;
      }
      .btn-prev,
      .btn-next {
        padding: 8px 12px;
        min-width: 40px;
        min-height: 40px;
      }
      .number {
        min-width: 36px;
        height: 36px;
        line-height: 36px;
        font-size: 14px;
      }
    }
  }
}
</style> 