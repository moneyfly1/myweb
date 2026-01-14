<template>
  <div class="list-container orders-container">

    <!-- 订单统计 -->
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

    <!-- 筛选和搜索 -->
    <el-card class="filter-card">
      <!-- 桌面端布局 -->
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
      
      <!-- 移动端紧凑布局 -->
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

    <!-- 订单列表 -->
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

      <!-- 标签页切换 -->
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

      <!-- 桌面端表格 -->
      <div class="table-wrapper">
        <el-table 
          :data="displayRecords" 
          style="width: 100%"
          v-loading="isLoading || isLoadingRecharges"
          :empty-text="emptyText"
          stripe
        >
          <el-table-column prop="record_type" label="类型" width="80">
            <template #default="scope">
              <el-tag 
                :type="scope.row.record_type === 'recharge' ? 'success' : 'primary'"
                size="small"
              >
                {{ scope.row.record_type === 'recharge' ? '充值' : '订单' }}
              </el-tag>
            </template>
          </el-table-column>
          
          <el-table-column prop="display_no" label="订单号" width="180">
            <template #default="scope">
              <el-tag size="small" type="info">{{ scope.row.display_no }}</el-tag>
            </template>
          </el-table-column>
          
          <el-table-column prop="package_name" label="套餐名称/类型">
            <template #default="scope">
              {{ scope.row.package_name || (scope.row.record_type === 'recharge' ? '账户充值' : '-') }}
            </template>
          </el-table-column>
          
          <el-table-column prop="display_amount" label="金额" width="120">
            <template #default="scope">
              <span 
                class="amount" 
                :class="{ 'positive': scope.row.record_type === 'recharge', 'negative': scope.row.record_type === 'order' }"
              >
                {{ scope.row.record_type === 'recharge' ? '+' : '-' }}¥{{ scope.row.display_amount }}
              </span>
            </template>
          </el-table-column>
          
          <el-table-column prop="payment_method" label="支付方式" width="120">
            <template #default="scope">
              <el-tag 
                :type="getPaymentMethodType(scope.row.payment_method)"
                size="small"
              >
                {{ getPaymentMethodText(scope.row.payment_method) }}
              </el-tag>
            </template>
          </el-table-column>
          
          <el-table-column prop="status" label="状态" width="120">
            <template #default="scope">
              <el-tag 
                :type="getOrderStatusType(scope.row.status)"
                size="small"
              >
                {{ getOrderStatusText(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          
          <el-table-column prop="created_at" label="创建时间" width="180">
            <template #default="scope">
              {{ formatDateTime(scope.row.created_at) }}
            </template>
          </el-table-column>
          
          <el-table-column prop="paid_at" label="支付时间" width="180">
            <template #default="scope">
              {{ (scope.row.paid_at || scope.row.payment_time) ? formatDateTime(scope.row.paid_at || scope.row.payment_time) : '-' }}
            </template>
          </el-table-column>
          
          <el-table-column label="操作" width="200" fixed="right">
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

      <!-- 移动端卡片式列表 -->
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

      <!-- 移动端空状态 -->
      <div class="mobile-card-list" v-if="displayRecords.length === 0 && !isLoading && !isLoadingRecharges">
        <div class="empty-state">
          <i class="el-icon-document"></i>
          <p>{{ emptyText }}</p>
        </div>
      </div>

      <!-- 分页 -->
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

    <!-- 订单详情对话框 -->
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

    <!-- 支付二维码对话框 -->
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
    
    // 响应式数据
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
    
    // 筛选条件
    const filters = reactive({
      status: '',
      payment_method: '',
      date_range: []
    })
    
    // 分页
    const pagination = reactive({
      current: 1,
      size: 20,
      total: 0
    })
    
    // 对话框状态
    const detailDialogVisible = ref(false)
    const paymentQRVisible = ref(false)
    const selectedOrder = ref(null)
    const paymentQRCode = ref('')
    const paymentUrl = ref('')  // 存储原始支付URL，用于跳转支付宝App
    const isCheckingPayment = ref(false)
    let paymentStatusCheckInterval = null

    // 移动端检测
    const windowWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1920)
    const isMobile = computed(() => {
      return windowWidth.value <= 768
    })
    
    // 监听窗口大小变化
    const handleResize = () => {
      if (typeof window !== 'undefined') {
        windowWidth.value = window.innerWidth
      }
    }
    
    // 计算属性
    const emptyText = computed(() => {
      if (isLoading.value) return '加载中...'
      if (filters.status || filters.payment_method || filters.date_range.length > 0) {
        return '没有找到符合条件的订单'
      }
      return '暂无订单记录'
    })
    
    // 加载充值记录
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
          // 确保充值记录有正确的字段
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
    
    // 合并订单和充值记录
    const mergeRecords = () => {
      const merged = []
      
      // 添加订单记录（标记类型）
      orders.value.forEach(order => {
        merged.push({
          ...order,
          record_type: 'order',
          display_no: order.order_no,
          display_amount: order.amount,
          display_type: '消费'
        })
      })
      
      // 添加充值记录（标记类型）
      recharges.value.forEach(recharge => {
        // 处理支付方式字段，可能是对象格式
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
      
      // 按创建时间倒序排序
      merged.sort((a, b) => {
        const timeA = new Date(a.created_at || a.paid_at || 0).getTime()
        const timeB = new Date(b.created_at || b.paid_at || 0).getTime()
        return timeB - timeA
      })
      
      allRecords.value = merged
    }
    
    // 格式化订单记录
    const formatOrderRecord = (order) => {
      return {
        ...order,
        record_type: 'order',
        display_no: order.order_no || '-',
        display_amount: order.amount || '0.00',
        package_name: order.package_name || '-',
        display_type: '消费',
        // 确保保留所有必要字段
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
    
    // 格式化充值记录
    const formatRechargeRecord = (recharge) => {
      // 处理支付方式字段，可能是对象格式
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
        // 确保保留所有必要字段
        id: recharge.id,
        order_no: recharge.order_no,
        amount: recharge.amount,
        status: recharge.status || 'pending',
        payment_method: recharge.payment_method || 'alipay',
        created_at: recharge.created_at,
        paid_at: recharge.paid_at
      }
    }
    
    // 计算当前显示的记录
    const displayRecords = computed(() => {
      if (activeTab.value === 'orders') {
        // 格式化订单记录
        return orders.value.map(formatOrderRecord)
      } else if (activeTab.value === 'recharges') {
        // 格式化充值记录
        return recharges.value.map(formatRechargeRecord)
      } else {
        return allRecords.value
      }
    })
    
    // 标签页切换处理
    const handleTabChange = (tabName) => {
      if (tabName === 'recharges') {
        if (recharges.value.length === 0) {
          loadRecharges()
        }
      } else if (tabName === 'orders') {
        // 切换到订单记录时，确保订单已加载
        if (orders.value.length === 0) {
          loadOrders()
        }
      } else if (tabName === 'all') {
        // 切换到全部记录时，确保数据已合并
        if (recharges.value.length === 0) {
          loadRecharges().then(() => {
            mergeRecords()
          })
        } else {
          mergeRecords()
        }
      }
    }
    
    // 方法
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
        // 确保订单有正确的字段
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
        
        // 更新统计信息
        await loadOrderStats()
        
        // 如果当前在"全部记录"标签页，合并记录
        if (activeTab.value === 'all') {
          await loadRecharges()
          mergeRecords()
        }
        
      } catch (error) {
        ElMessage.error('加载订单列表失败')
        } finally {
        isLoading.value = false
      }
    }
    
    const loadOrderStats = async () => {
      try {
        const response = await api.get('/orders/stats')
        
        if (response.data && response.data.success && response.data.data) {
          const statsData = response.data.data
          orderStats.value = {
            total: statsData.total || 0,
            pending: statsData.pending || 0,
            paid: statsData.paid || 0,
            cancelled: statsData.cancelled || 0,
            totalAmount: statsData.totalAmount || 0
          }
        } else {
          // 设置默认值
          orderStats.value = {
            total: 0,
            pending: 0,
            paid: 0,
            cancelled: 0,
            totalAmount: 0
          }
        }
      } catch (error) {
        // 设置默认值
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
      await loadOrders()
      if (activeTab.value === 'all' || activeTab.value === 'recharges') {
        await loadRecharges()
        if (activeTab.value === 'all') {
          mergeRecords()
        }
      }
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
        
        // 确保有 order_no
        const orderNo = order.order_no || order.display_no
        if (!orderNo) {
          ElMessage.error('订单号不存在，无法支付')
          return
        }
        
        // 获取支付方式ID
        let paymentMethodId = order.payment_method_id
        
        // 如果订单中没有 payment_method_id，尝试根据 payment_method 查找
        if (!paymentMethodId && order.payment_method) {
          try {
            const paymentMethodsResponse = await paymentAPI.getPaymentMethods()
            const paymentMethods = paymentMethodsResponse.data?.data || paymentMethodsResponse.data || []
            
            // 根据支付方式名称或类型查找对应的ID
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
              // 如果没有找到匹配的，使用第一个可用的支付方式
              paymentMethodId = paymentMethods[0].id
            }
          } catch (error) {
            console.warn('获取支付方式列表失败:', error)
          }
        }
        
        // 如果仍然没有 payment_method_id，提示错误
        if (!paymentMethodId) {
          ElMessage.error('无法确定支付方式，请刷新页面后重试')
          return
        }
        
        // 调用立即支付API，传递 payment_method_id
        const response = await api.post(`/orders/${orderNo}/pay`, {
          payment_method_id: paymentMethodId
        })
        
        // 检查响应结构
        if (response.data && response.data.success !== false) {
          // 检查是否有支付URL
          const paymentUrl = response.data.data?.payment_url || response.data.data?.payment_qr_code
          
          if (paymentUrl) {
            // 显示支付二维码
            await showPaymentQR({
              ...order,
              order_no: orderNo,
              payment_method_id: paymentMethodId
            }, paymentUrl)
          } else {
            // 检查是否有错误信息
            const errorMsg = response.data.message || response.data.detail || '支付链接生成失败'
            ElMessage.error(errorMsg)
          }
        } else {
          // 响应表明失败
          const errorMsg = response.data?.message || response.data?.detail || '创建支付订单失败'
          ElMessage.error(errorMsg)
        }
        
      } catch (error) {
        // 提取错误信息
        const errorMsg = error.response?.data?.detail || 
                        error.response?.data?.message || 
                        error.message || 
                        '创建支付订单失败，请重试'
        ElMessage.error(errorMsg)
      }
    }
    
    // 跳转到支付宝App
    const openAlipayApp = () => {
      if (!paymentUrl.value) {
        ElMessage.error('支付链接不存在')
        return
      }
      
      // 生成支付宝App跳转链接
      // 支付宝App的URL Scheme格式：alipays://platformapi/startapp?saId=10000007&qrcode=支付URL
      const alipayAppUrl = `alipays://platformapi/startapp?saId=10000007&qrcode=${encodeURIComponent(paymentUrl.value)}`
      
      try {
        // 添加页面可见性监听，当用户从支付宝返回时立即检查支付状态
        const handleVisibilityChange = async () => {
          if (document.visibilityState === 'visible' && paymentQRVisible.value) {
            // 用户返回页面，立即检查支付状态
            await checkPaymentStatus()
            // 移除监听器
            document.removeEventListener('visibilitychange', handleVisibilityChange)
          }
        }
        document.addEventListener('visibilitychange', handleVisibilityChange)
        
        // 添加页面焦点监听
        const handleFocus = async () => {
          if (paymentQRVisible.value) {
            await checkPaymentStatus()
            window.removeEventListener('focus', handleFocus)
          }
        }
        window.addEventListener('focus', handleFocus)
        
        // 尝试打开支付宝App
        window.location.href = alipayAppUrl
        
        // 如果3秒后还在当前页面，说明可能没有安装支付宝App，提示用户
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
      
      // 保存原始支付URL，用于跳转支付宝App
      paymentUrl.value = url
      
      selectedOrder.value = order
      
      // 支付宝支付：使用qrcode库将支付宝URL生成为二维码图片
      const paymentMethod = order.payment_method_name || order.payment_method || 'alipay'
      
      if (paymentMethod === 'alipay') {
        // 支付宝返回的是URL（如 https://qr.alipay.com/xxx），需要在前端生成二维码图片
        if (url.startsWith('http://') || url.startsWith('https://')) {
          try {
            // 动态导入qrcode库
            const QRCode = await import('qrcode')
            // 将支付宝URL生成为base64格式的二维码图片
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
        // 非支付宝支付方式，使用qrcode库生成二维码
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
          // 直接是字符串，也使用qrcode库生成
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
      
      // 确保 selectedOrder 有正确的字段
      selectedOrder.value = {
        ...order,
        order_no: order.order_no || order.display_no,
        package_name: order.package_name || '账户充值',
        amount: order.amount || order.display_amount,
        payment_method: order.payment_method || order.payment_method_name || 'alipay'
      }
      
      // 显示二维码对话框
      paymentQRVisible.value = true
      
      // 等待一下确保对话框已渲染
      await new Promise(resolve => setTimeout(resolve, 200))
      
      // 开始检查支付状态
      startPaymentStatusCheck()
    }
    
    // 获取支付方式名称
    const getPaymentMethodName = (paymentUrl) => {
      if (paymentUrl.includes('qr.alipay.com')) {
        return '支付宝'
      } else if (paymentUrl.includes('weixin') || paymentUrl.includes('wechat')) {
        return '微信'
      }
      return '支付'
    }
    
    // 图片加载成功
    const onImageLoad = () => {
      }
    
    // 图片加载失败
    const onImageError = async (event) => {
      if (paymentQRCode.value && paymentQRCode.value.startsWith('data:')) {
        ElMessage.warning('二维码显示异常，正在重新生成...')
        
        // 从订单信息中重新获取支付URL并生成二维码
        if (selectedOrder.value) {
          try {
            // 获取支付方式ID
            let paymentMethodId = selectedOrder.value.payment_method_id
            
            // 如果没有 payment_method_id，尝试获取
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
                console.warn('获取支付方式列表失败:', error)
              }
            }
            
            if (!paymentMethodId) {
              ElMessage.error('无法确定支付方式，请刷新页面后重试')
              return
            }
            
            // 重新调用支付API获取支付URL，传递 payment_method_id
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
      // 清除之前的检查
      if (paymentStatusCheckInterval) {
        clearInterval(paymentStatusCheckInterval)
        paymentStatusCheckInterval = null
      }
      
      // 立即检查一次支付状态
      checkPaymentStatus()
      
      // 每2秒检查一次支付状态（提高检查频率）
      paymentStatusCheckInterval = setInterval(async () => {
        await checkPaymentStatus()
      }, 2000)
      
      // 添加页面可见性监听，当用户从其他应用返回时立即检查
      const handleVisibilityChange = async () => {
        if (document.visibilityState === 'visible' && paymentQRVisible.value) {
          // 用户返回页面，立即检查支付状态
          await checkPaymentStatus()
        }
      }
      document.addEventListener('visibilitychange', handleVisibilityChange)
      
      // 添加页面焦点监听
      const handleFocus = async () => {
        if (paymentQRVisible.value) {
          await checkPaymentStatus()
        }
      }
      window.addEventListener('focus', handleFocus)
      
      // 30分钟后停止检查
      setTimeout(() => {
        if (paymentStatusCheckInterval) {
          clearInterval(paymentStatusCheckInterval)
          paymentStatusCheckInterval = null
        }
        document.removeEventListener('visibilitychange', handleVisibilityChange)
        window.removeEventListener('focus', handleFocus)
      }, 30 * 60 * 1000)
    }
    
    const checkPaymentStatus = async () => {
      if (!selectedOrder.value) return
      
      try {
        isCheckingPayment.value = true
        
        // 判断是订单还是充值记录
        const isRecharge = selectedOrder.value.record_type === 'recharge' || selectedOrder.value.id && !selectedOrder.value.order_no
        
        if (isRecharge) {
          // 检查充值记录状态
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
            if (process.env.NODE_ENV === 'development') {
              console.log('检查支付状态：订单号不存在', selectedOrder.value)
            }
            return
          }

          const response = await api.get(`/orders/${selectedOrder.value.order_no}/status`)

          if (process.env.NODE_ENV === 'development') {
            console.log('订单状态检查响应:', {
              order_no: selectedOrder.value.order_no,
              response: response.data,
              status: response.data?.data?.status
            })
          }

          if (!response || !response.data) {
            if (process.env.NODE_ENV === 'development') {
              console.warn('订单状态检查：响应格式错误', response)
            }
            return
          }

          if (response.data.success === false) {
            if (process.env.NODE_ENV === 'development') {
              console.warn('订单状态检查：API返回失败', response.data.message)
            }
            return
          }

          const orderData = response.data.data
          if (!orderData) {
            if (process.env.NODE_ENV === 'development') {
              console.warn('订单状态检查：订单数据不存在', response.data)
            }
            return
          }

          if (process.env.NODE_ENV === 'development') {
            console.log('当前订单状态:', orderData.status, '订单号:', orderData.order_no)
          }

          if (orderData.status === 'paid') {
            if (process.env.NODE_ENV === 'development') {
              console.log('✅ 订单支付成功，开始处理...')
            }
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
            if (process.env.NODE_ENV === 'development') {
              console.log('订单已取消')
            }
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
          } else {
            console.log('订单状态:', orderData.status, '继续等待...')
          }
        }
        
      } catch (error) {
        } finally {
        isCheckingPayment.value = false
      }
    }
    
    const closePaymentQR = () => {
      // 清除支付状态检查定时器
      if (paymentStatusCheckInterval) {
        clearInterval(paymentStatusCheckInterval)
        paymentStatusCheckInterval = null
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
    
    // 充值记录支付
    const payRecharge = async (recharge) => {
      try {
        // 获取充值记录详情（包含支付URL）
        const response = await rechargeAPI.getRechargeDetail(recharge.id)
        if (response.data && response.data.success) {
          const rechargeData = response.data.data
          const paymentUrl = rechargeData.payment_url || rechargeData.payment_qr_code
          
          if (paymentUrl) {
            // 显示支付二维码
            showPaymentQR({
              ...recharge,
              order_no: recharge.order_no,
              package_name: '账户充值',
              amount: recharge.amount
            }, paymentUrl)
          } else {
            // 如果充值记录没有支付URL，尝试重新创建支付链接
            ElMessage.warning('支付链接不存在，正在重新生成...')
            // 这里可以调用后端API重新生成支付链接
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
    
    // 取消充值
    const cancelRecharge = async (recharge) => {
      try {
        // 确保有 id
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
        // 如果当前在订单标签页，也需要刷新订单（因为可能影响余额）
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
        // 确保订单对象有必要的字段
        if (!order) {
          ElMessage.error('订单信息不存在')
          return
        }
        
        // 确保有 order_no
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

    // 工具方法
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
      // 处理可能的对象格式（如 {"String": "alipay", "Valid": true}）
      let methodStr = method
      if (method && typeof method === 'object') {
        if (method.String) {
          methodStr = method.String
        } else if (method.payment_method) {
          methodStr = method.payment_method
        } else {
          // 尝试从对象中提取字符串值
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
    
    // 格式化金额（保留2位小数）
    const formatAmount = (amount) => {
      if (amount === null || amount === undefined || amount === '') return '0.00'
      const num = typeof amount === 'string' ? parseFloat(amount) : amount
      if (isNaN(num)) return '0.00'
      return num.toFixed(2)
    }
    
    // 生命周期
    onMounted(async () => {
      await loadOrders()
      // 如果默认显示全部记录，加载充值记录
      if (activeTab.value === 'all') {
        await loadRecharges()
        mergeRecords()
      }
      // 初始化窗口大小
      if (typeof window !== 'undefined') {
        windowWidth.value = window.innerWidth
        window.addEventListener('resize', handleResize)
      }
    })
    
    onUnmounted(() => {
      // 清理窗口大小监听
      if (typeof window !== 'undefined') {
        window.removeEventListener('resize', handleResize)
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
      isMobile
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

/* 操作按钮样式 */
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
    
    // 确保按钮内部所有元素都可以点击
    :deep(*) {
      pointer-events: none;
    }
    
    // 确保按钮本身可以点击 - 覆盖整个按钮区域
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
    
    // 确保按钮文字在伪元素之上
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

/* 修复表格固定列中的按钮点击问题 */
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

/* 确保表格单元格中的按钮可以正常点击 */
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

/* 标签页样式 */
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

/* 移动端卡片样式 */
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
      
      // 确保按钮内部所有元素都可以点击
      :deep(*) {
        pointer-events: auto; /* 改为auto，允许点击 */
      }
      
      // 移除伪元素，避免干扰点击
      
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

/* 支付二维码样式 */
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

/* 手机端对话框优化 */
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

/* 手机端订单详情优化 */
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

/* 手机端支付二维码优化 */
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

/* 筛选区域优化 */
.filter-card {
  padding: 12px;
  margin-bottom: 12px;
  
  @media (max-width: 768px) {
    padding: 10px;
    margin-bottom: 10px;
  }
}

/* 桌面端筛选布局 */
.filter-desktop {
  display: block;
  
  @media (max-width: 768px) {
    display: none;
  }
}

/* 移动端筛选布局 */
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

/* 手机端统计卡片优化 */
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

/* 手机端分页优化 */
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