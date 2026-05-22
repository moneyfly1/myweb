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
            <el-option label="支付失败" value="failed" />
            <el-option label="已过期" value="expired" />
            <el-option label="已退款" value="refunded" />
          </el-select>
          <el-button @click="resetSearch" icon="Refresh" circle class="mobile-reset-btn"></el-button>
        </div>
      </div>

      <!-- 电脑端搜索表单 (保持不变) -->
      <el-form :inline="true" :model="searchForm" class="search-form list-filter-form" v-else>
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
            <el-option label="支付失败" value="failed" />
            <el-option label="已过期" value="expired" />
            <el-option label="已退款" value="refunded" />
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
          <el-table-column type="selection" width="55" v-if="activeTab === 'orders'" :selectable="isOrderSelectable" />
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
          <el-table-column label="操作" width="280" fixed="right">
            <template #default="scope">
              <div class="action-buttons-grid" v-if="!isRechargeRecord(scope.row)">
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
                  type="warning" 
                  @click="refundOrder(scope.row)"
                  v-if="scope.row.status === 'paid' && canRefundOrder(scope.row)"
                  class="action-btn"
                >
                  <el-icon><Money /></el-icon> 退款
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
              <el-button v-else size="small" @click="viewOrder(scope.row)" class="action-btn">
                <el-icon><View /></el-icon> 查看充值
              </el-button>
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
          <div class="mc-footer">
            <el-button-group class="mc-actions">
              <el-button size="small" @click="viewOrder(item)">
                 详情
              </el-button>
              <el-button 
                v-if="item.status === 'paid' && canRefundOrder(item)"
                size="small" 
                type="warning" 
                plain
                @click="refundOrder(item)"
              >
                退款
              </el-button>
              <el-button 
                v-if="!isRechargeRecord(item) && item.status === 'pending'"
                size="small" 
                type="success" 
                plain
                @click="markAsPaid(item)"
              >
                已付
              </el-button>
              <el-button 
                v-if="!isRechargeRecord(item) && item.status === 'pending'"
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
                v-if="!isRechargeRecord(item)"
                @click="deleteOrder(item)"
              />
            </el-button-group>
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
      :title="detailTitle"
      :size="isMobile ? '94%' : '720px'"
      direction="rtl"
      class="order-detail-drawer"
      :lock-scroll="false"
    >
      <div class="order-detail-content" v-if="selectedOrder && selectedOrder.order_no">
        <div class="detail-hero" :class="{ 'is-recharge': isRechargeRecord(selectedOrder) }">
          <div class="hero-main">
            <div class="hero-kicker">{{ detailKindLabel(selectedOrder) }}</div>
            <div class="hero-title">{{ detailPrimaryName(selectedOrder) }}</div>
            <div class="hero-order-no">{{ selectedOrder.order_no }}</div>
          </div>
          <div class="hero-side">
            <div class="hero-amount" :class="{ 'is-plus': isRechargeRecord(selectedOrder) }">
              {{ isRechargeRecord(selectedOrder) ? '+' : '' }}¥{{ formatMoney(selectedOrder.amount) }}
            </div>
            <el-tag :type="getStatusType(selectedOrder.status)" effect="dark">
              {{ getStatusText(selectedOrder.status) }}
            </el-tag>
          </div>
        </div>

        <div class="detail-section">
          <div class="detail-section-title">基础信息</div>
          <div class="detail-grid">
            <div class="detail-field">
              <span class="field-label">客户邮箱</span>
              <span class="field-value">{{ selectedOrder.user?.email || '-' }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">客户账号</span>
              <span class="field-value">{{ selectedOrder.user?.username || '-' }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">创建时间</span>
              <span class="field-value">{{ formatDateTime(selectedOrder.created_at) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">{{ isRechargeRecord(selectedOrder) ? '到账时间' : '支付时间' }}</span>
              <span class="field-value">{{ detailPaidTime(selectedOrder) }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section">
          <div class="detail-section-title">支付信息</div>
          <div class="detail-grid">
            <div class="detail-field">
              <span class="field-label">支付方式</span>
              <span class="field-value">{{ selectedOrder.payment_method || '-' }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">交易流水</span>
              <span class="field-value mono">{{ selectedOrder.payment_transaction_id || '-' }}</span>
            </div>
            <div class="detail-field" v-if="!isRechargeRecord(selectedOrder)">
              <span class="field-label">订单原价</span>
              <span class="field-value">¥{{ formatMoney(orderOriginalAmount(selectedOrder)) }}</span>
            </div>
            <div class="detail-field" v-if="!isRechargeRecord(selectedOrder)">
              <span class="field-label">优惠金额</span>
              <span class="field-value discount">-¥{{ formatMoney(selectedOrder.discount_amount || 0) }}</span>
            </div>
            <div class="detail-field" v-if="isRechargeRecord(selectedOrder)">
              <span class="field-label">充值金额</span>
              <span class="field-value positive">+¥{{ formatMoney(selectedOrder.amount) }}</span>
            </div>
            <div class="detail-field" v-if="isRechargeRecord(selectedOrder)">
              <span class="field-label">充值 IP</span>
              <span class="field-value">{{ selectedOrder.ip_address || '-' }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section" v-if="!isRechargeRecord(selectedOrder)">
          <div class="detail-section-title">金额拆分</div>
          <div class="detail-grid">
            <div class="detail-field">
              <span class="field-label">订单标价</span>
              <span class="field-value">{{ moneyField(selectedOrder.base_amount, selectedOrder.order_amount, orderOriginalAmount(selectedOrder)) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">优惠合计</span>
              <span class="field-value discount">-{{ moneyField(selectedOrder.discount_amount, 0) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">余额抵扣</span>
              <span class="field-value discount">-{{ moneyField(extraValue(selectedOrder, 'balance_used'), 0) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">折后应付</span>
              <span class="field-value">{{ moneyField(extraValue(selectedOrder, 'payable_amount'), selectedOrder.final_amount, selectedOrder.amount) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">第三方实付</span>
              <span class="field-value positive">{{ moneyField(selectedOrder.final_amount, selectedOrder.amount) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">订单入账金额</span>
              <span class="field-value positive">{{ moneyField(selectedOrder.amount) }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section" v-if="!isRechargeRecord(selectedOrder)">
          <div class="detail-section-title">套餐与订单参数</div>
          <div class="detail-grid">
            <div class="detail-field">
              <span class="field-label">订单类型</span>
              <span class="field-value">{{ packageTypeText(selectedOrder) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">套餐 ID</span>
              <span class="field-value mono">{{ fieldOrDash(selectedOrder.package_id || extraValue(selectedOrder, 'package_id')) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">原套餐 ID</span>
              <span class="field-value mono">{{ fieldOrDash(extraValue(selectedOrder, 'old_package_id')) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">付款后套餐 ID</span>
              <span class="field-value mono">{{ fieldOrDash(extraValue(selectedOrder, 'new_package_id') || extraValue(selectedOrder, 'package_id') || selectedOrder.package_id) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">购买设备数</span>
              <span class="field-value">{{ deviceCountText(selectedOrder) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">购买月数</span>
              <span class="field-value">{{ monthCountText(selectedOrder) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">套餐基础天数</span>
              <span class="field-value">{{ daysText(extraValue(selectedOrder, 'package_duration_days')) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">本次开通天数</span>
              <span class="field-value accent">{{ daysText(extraValue(selectedOrder, 'duration_days') || extraValue(selectedOrder, 'additional_days')) }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section" v-if="hasDiscountOrBalance(selectedOrder)">
          <div class="detail-section-title">优惠与余额</div>
          <div class="detail-grid">
            <div class="detail-field">
              <span class="field-label">优惠券 ID</span>
              <span class="field-value mono">{{ fieldOrDash(selectedOrder.coupon_id) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">优惠券赠送天数</span>
              <span class="field-value">{{ daysText(extraValue(selectedOrder, 'coupon_free_days')) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">会员等级优惠</span>
              <span class="field-value discount">-{{ moneyField(extraValue(selectedOrder, 'level_discount'), 0) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">等级折扣率</span>
              <span class="field-value">{{ percentField(extraValue(selectedOrder, 'level_discount_rate')) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">自定义时长折扣</span>
              <span class="field-value">{{ percentField(extraValue(selectedOrder, 'discount_percent')) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">余额已扣除</span>
              <span class="field-value">{{ boolText(extraValue(selectedOrder, 'balance_deducted')) }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section" v-if="isPackageLikeOrder(selectedOrder)">
          <div class="detail-section-title">套餐开通变化</div>
          <div class="change-panel">
            <div class="change-card">
              <span class="change-label">原到期时间</span>
              <strong>{{ extraValue(selectedOrder, 'old_expire_time') || '未记录' }}</strong>
            </div>
            <div class="change-arrow">→</div>
            <div class="change-card is-new">
              <span class="change-label">付款后到期时间</span>
              <strong>{{ extraValue(selectedOrder, 'new_expire_time') || '未记录' }}</strong>
            </div>
          </div>
          <div class="detail-grid compact">
            <div class="detail-field">
              <span class="field-label">本次购买天数</span>
              <span class="field-value accent">{{ detailAddedDays(selectedOrder) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">到期净变化</span>
              <span class="field-value accent">{{ expireDeltaText(selectedOrder) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">开通方式</span>
              <span class="field-value">{{ activationModeText(extraValue(selectedOrder, 'activation_mode')) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">付款前已有订阅</span>
              <span class="field-value">{{ boolText(extraValue(selectedOrder, 'had_existing_subscription')) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">原设备数</span>
              <span class="field-value">{{ numberOrDash(extraValue(selectedOrder, 'old_device_limit')) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">付款后设备数</span>
              <span class="field-value">{{ numberOrDash(extraValue(selectedOrder, 'new_device_limit')) }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section" v-if="isDeviceUpgradeOrder(selectedOrder)">
          <div class="detail-section-title">设备升级变化</div>
          <div class="change-panel">
            <div class="change-card">
              <span class="change-label">原设备数</span>
              <strong>{{ numberOrZero(extraValue(selectedOrder, 'old_device_limit')) }} 台</strong>
            </div>
            <div class="change-arrow">→</div>
            <div class="change-card is-new">
              <span class="change-label">升级后设备数</span>
              <strong>{{ numberOrZero(extraValue(selectedOrder, 'new_device_limit')) }} 台</strong>
            </div>
          </div>
          <div class="detail-grid compact">
            <div class="detail-field">
              <span class="field-label">增加设备</span>
              <span class="field-value accent">+{{ numberOrZero(extraValue(selectedOrder, 'additional_devices')) }} 台</span>
            </div>
            <div class="detail-field">
              <span class="field-label">增加天数</span>
              <span class="field-value accent">+{{ numberOrZero(extraValue(selectedOrder, 'additional_days')) }} 天</span>
            </div>
            <div class="detail-field">
              <span class="field-label">原到期时间</span>
              <span class="field-value">{{ extraValue(selectedOrder, 'old_expire_time') || '未记录' }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">升级后到期时间</span>
              <span class="field-value">{{ extraValue(selectedOrder, 'new_expire_time') || '未记录' }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">到期净变化</span>
              <span class="field-value accent">{{ expireDeltaText(selectedOrder) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">实际履约时间</span>
              <span class="field-value">{{ formatDateTime(selectedOrder.fulfilled_at) }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section" v-if="isRechargeRecord(selectedOrder)">
          <div class="detail-section-title">充值到账信息</div>
          <div class="detail-grid">
            <div class="detail-field">
              <span class="field-label">充值状态</span>
              <span class="field-value">{{ getStatusText(selectedOrder.status) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">更新时间</span>
              <span class="field-value">{{ formatDateTime(selectedOrder.updated_at) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">充值记录 ID</span>
              <span class="field-value mono">{{ selectedOrder.id }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">用户 ID</span>
              <span class="field-value mono">{{ selectedOrder.user_id }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">支付二维码</span>
              <span class="field-value mono">{{ qrSummary(selectedOrder.payment_qr_code, selectedOrder.payment_url) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">支付链接</span>
              <span class="field-value mono">{{ selectedOrder.payment_url || '-' }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">充值 IP</span>
              <span class="field-value">{{ selectedOrder.ip_address || '-' }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">IP 归属地</span>
              <span class="field-value">{{ selectedOrder.location || '-' }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">用户代理</span>
              <span class="field-value">{{ selectedOrder.user_agent || '-' }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section">
          <div class="detail-section-title">履约与审计</div>
          <div class="detail-grid">
            <div class="detail-field">
              <span class="field-label">{{ isRechargeRecord(selectedOrder) ? '充值 ID' : '订单 ID' }}</span>
              <span class="field-value mono">{{ selectedOrder.id }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">用户 ID</span>
              <span class="field-value mono">{{ selectedOrder.user_id }}</span>
            </div>
            <div class="detail-field" v-if="!isRechargeRecord(selectedOrder)">
              <span class="field-label">支付方式 ID</span>
              <span class="field-value mono">{{ fieldOrDash(selectedOrder.payment_method_id) }}</span>
            </div>
            <div class="detail-field" v-if="!isRechargeRecord(selectedOrder)">
              <span class="field-label">优惠券 ID</span>
              <span class="field-value mono">{{ fieldOrDash(selectedOrder.coupon_id) }}</span>
            </div>
            <div class="detail-field" v-if="!isRechargeRecord(selectedOrder)">
              <span class="field-label">订单支付过期时间</span>
              <span class="field-value">{{ formatDateTime(selectedOrder.expire_time) }}</span>
            </div>
            <div class="detail-field" v-if="!isRechargeRecord(selectedOrder)">
              <span class="field-label">实际履约时间</span>
              <span class="field-value">{{ formatDateTime(selectedOrder.fulfilled_at) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">创建时间</span>
              <span class="field-value">{{ formatDateTime(selectedOrder.created_at) }}</span>
            </div>
            <div class="detail-field">
              <span class="field-label">最后更新时间</span>
              <span class="field-value">{{ formatDateTime(selectedOrder.updated_at) }}</span>
            </div>
            <div class="detail-field" v-if="!isRechargeRecord(selectedOrder)">
              <span class="field-label">履约状态</span>
              <span class="field-value">{{ fulfillmentText(selectedOrder) }}</span>
            </div>
            <div class="detail-field" v-if="!isRechargeRecord(selectedOrder)">
              <span class="field-label">开通方式</span>
              <span class="field-value">{{ activationModeText(extraValue(selectedOrder, 'activation_mode')) }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section" v-if="extraEntries(selectedOrder).length">
          <div class="detail-section-title">扩展数据快照</div>
          <div class="extra-data-list">
            <div
              v-for="entry in extraEntries(selectedOrder)"
              :key="entry.key"
              class="extra-data-row"
            >
              <span class="extra-data-label">{{ entry.label }}</span>
              <span class="extra-data-value">{{ entry.value }}</span>
            </div>
          </div>
        </div>

        <div v-if="selectedOrder.payment_proof" class="payment-proof-section">
          <div class="detail-section-title">支付凭证</div>
          <img :src="selectedOrder.payment_proof" class="proof-image" @click="previewImage(selectedOrder.payment_proof)" />
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
              <div class="stat-number">{{ statistics.cancelledOrders }}</div>
              <div class="stat-label">已取消</div>
            </el-card>
          </el-col>
        </el-row>
        <el-row :gutter="20" style="margin-top: 20px;">
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
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from '@/utils/elementPlusServices'
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
    const pageSize = ref(10)
    const total = ref(0)
    const rechargeTotal = ref(0)
    
    const showOrderDialog = ref(false)
    const showStatisticsDialog = ref(false)
    const showImageViewer = ref(false)
    const imageViewerUrl = ref('')
    
    const selectedOrder = ref({})
    const selectedOrders = ref([])
    const bulkLoading = ref(false)
    const detailTitle = computed(() => isRechargeRecord(selectedOrder.value) ? '充值详情' : '订单详情')
    const renderedExtraKeys = new Set([
      'type',
      'duration_days',
      'additional_days',
      'old_device_limit',
      'new_device_limit',
      'old_expire_time',
      'new_expire_time',
      'old_expire_time_rfc3339',
      'new_expire_time_rfc3339',
      'activation_mode',
      'had_existing_subscription',
      'package_id',
      'package_name',
      'package_duration_days',
      'duration_months',
      'new_package_id',
      'old_package_id',
      'devices',
      'months',
      'discount_percent',
      'additional_devices',
      'balance_used',
      'balance_deducted',
      'level_discount',
      'level_discount_rate',
      'payable_amount',
      'coupon_free_days'
    ])
    const extraLabelMap = {
      type: '订单类型',
      duration_days: '开通天数',
      additional_days: '增加天数',
      old_device_limit: '原设备数',
      new_device_limit: '付款后设备数',
      old_expire_time: '原到期时间',
      new_expire_time: '付款后到期时间',
      old_expire_time_rfc3339: '原到期时间原始值',
      new_expire_time_rfc3339: '付款后到期时间原始值',
      activation_mode: '开通方式',
      had_existing_subscription: '付款前已有订阅',
      package_id: '套餐 ID',
      package_name: '套餐名称',
      package_duration_days: '套餐基础天数',
      duration_months: '购买月数',
      new_package_id: '付款后套餐 ID',
      old_package_id: '原套餐 ID',
      devices: '自定义设备数',
      months: '自定义月数',
      discount_percent: '自定义时长折扣',
      additional_devices: '增加设备',
      balance_used: '余额抵扣',
      balance_deducted: '余额已扣除',
      level_discount: '会员等级优惠',
      level_discount_rate: '等级折扣率',
      payable_amount: '折后应付',
      coupon_free_days: '优惠券赠送天数'
    }
    
    const isMobile = ref(window.innerWidth <= 768)
    const searchForm = reactive({
      keyword: '',
      status: ''
    })
    const statistics = reactive({
      totalOrders: 0,
      pendingOrders: 0,
      paidOrders: 0,
      cancelledOrders: 0,
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

    const isRechargeRecord = (record) => {
      return record?.record_type === 'recharge' || String(record?.order_no || '').startsWith('RCH')
    }

    const orderExtra = (order) => order?.extra_data || {}

    const extraValue = (order, key) => {
      const value = orderExtra(order)[key]
      return value === undefined || value === null || value === '' ? '' : value
    }

    const numberOrZero = (value) => {
      const num = Number(value)
      return Number.isFinite(num) ? num : 0
    }

    const numberOrDash = (value) => {
      const num = Number(value)
      return Number.isFinite(num) ? num : '-'
    }

    const isDeviceUpgradeOrder = (order) => {
      return !isRechargeRecord(order) && extraValue(order, 'type') === 'device_upgrade'
    }

    const isCustomPackageOrder = (order) => {
      return !isRechargeRecord(order) && extraValue(order, 'type') === 'custom_package'
    }

    const isPackageLikeOrder = (order) => {
      if (!order || isRechargeRecord(order) || isDeviceUpgradeOrder(order)) return false
      return true
    }

    const detailKindLabel = (order) => {
      if (isRechargeRecord(order)) return '账户充值'
      if (isDeviceUpgradeOrder(order)) return '设备升级'
      if (isCustomPackageOrder(order)) return '自定义套餐'
      return '套餐订单'
    }

    const detailPrimaryName = (order) => {
      if (isRechargeRecord(order)) return '账户余额充值'
      return order?.package_name || detailKindLabel(order)
    }

    const detailPaidTime = (order) => {
      return formatDateTime(isRechargeRecord(order) ? order?.paid_at : order?.payment_time)
    }

    const detailAddedDays = (order) => {
      const days = numberOrZero(extraValue(order, 'additional_days') || extraValue(order, 'duration_days'))
      return days > 0 ? `+${days} 天` : '未记录'
    }

    const hasValue = (value) => value !== undefined && value !== null && value !== ''

    const fieldOrDash = (...values) => {
      const value = values.find(hasValue)
      return hasValue(value) ? String(value) : '-'
    }

    const moneyField = (...values) => {
      const value = values.find(hasValue)
      return `¥${formatMoney(value || 0)}`
    }

    const percentField = (value) => {
      if (!hasValue(value)) return '-'
      const num = Number(value)
      if (!Number.isFinite(num)) return String(value)
      const percent = num > 0 && num <= 1 ? num * 100 : num
      return `${percent.toFixed(percent % 1 === 0 ? 0 : 2)}%`
    }

    const boolText = (value) => {
      if (value === true || value === 'true' || value === 1 || value === '1') return '是'
      if (value === false || value === 'false' || value === 0 || value === '0') return '否'
      return '未记录'
    }

    const daysText = (value) => {
      const days = numberOrZero(value)
      return days > 0 ? `${days} 天` : '-'
    }

    const deviceCountText = (order) => {
      const devices = extraValue(order, 'devices') || extraValue(order, 'new_device_limit') || order?.package?.device_limit
      const count = numberOrZero(devices)
      return count > 0 ? `${count} 台` : '-'
    }

    const monthCountText = (order) => {
      const months = extraValue(order, 'months') || extraValue(order, 'duration_months')
      const count = numberOrZero(months)
      return count > 0 ? `${count} 个月` : '-'
    }

    const packageTypeText = (order) => {
      if (isDeviceUpgradeOrder(order)) return '设备升级'
      if (isCustomPackageOrder(order)) return '自定义套餐'
      return '固定套餐'
    }

    const toComparableDate = (value) => {
      if (!hasValue(value)) return null
      if (value instanceof Date) return Number.isNaN(value.getTime()) ? null : value
      const raw = String(value)
      const normalized = /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(raw)
        ? raw.replace(' ', 'T') + '+08:00'
        : raw
      const date = new Date(normalized)
      return Number.isNaN(date.getTime()) ? null : date
    }

    const expireDeltaText = (order) => {
      const oldDate = toComparableDate(extraValue(order, 'old_expire_time_rfc3339') || extraValue(order, 'old_expire_time'))
      const newDate = toComparableDate(extraValue(order, 'new_expire_time_rfc3339') || extraValue(order, 'new_expire_time'))
      if (!newDate) return '未记录'
      if (!oldDate) return detailAddedDays(order)
      const days = Math.round((newDate.getTime() - oldDate.getTime()) / 86400000)
      if (days > 0) return `+${days} 天`
      if (days < 0) return `${days} 天`
      return '0 天'
    }

    const hasDiscountOrBalance = (order) => {
      if (!order || isRechargeRecord(order)) return false
      return [
        order.coupon_id,
        order.discount_amount,
        extraValue(order, 'coupon_free_days'),
        extraValue(order, 'balance_used'),
        extraValue(order, 'level_discount'),
        extraValue(order, 'level_discount_rate'),
        extraValue(order, 'discount_percent'),
        extraValue(order, 'balance_deducted')
      ].some(hasValue)
    }

    const qrSummary = (qrCode, paymentUrl) => {
      if (!qrCode) return '-'
      if (qrCode === paymentUrl) return '同支付链接'
      return String(qrCode)
    }

    const fulfillmentText = (order) => {
      if (!order) return '-'
      if (order.status !== 'paid') return '未支付未履约'
      return order.fulfilled_at ? '已开通/已履约' : '已支付但未记录履约时间'
    }

    const formatExtraValue = (key, value) => {
      if (value === undefined || value === null || value === '') return '-'
      if (key.includes('amount') || key.includes('price') || key.includes('balance') || key.includes('discount')) {
        if (!key.includes('percent') && !key.includes('rate')) return moneyField(value)
      }
      if (key.includes('rate') || key.includes('percent')) return percentField(value)
      if (key.includes('days')) return daysText(value)
      if (typeof value === 'boolean') return boolText(value)
      if (typeof value === 'object') return JSON.stringify(value)
      return String(value)
    }

    const extraEntries = (order) => {
      return Object.entries(orderExtra(order))
        .filter(([key, value]) => !renderedExtraKeys.has(key) && hasValue(value))
        .map(([key, value]) => ({
          key,
          label: extraLabelMap[key] || key,
          value: formatExtraValue(key, value)
        }))
    }

    const activationModeText = (mode) => ({
      create: '新建订阅',
      extend: '续期叠加',
      reactivate: '过期重开',
      replace: '替换开通'
    }[mode] || '未记录')

    const orderOriginalAmount = (order) => {
      const discount = Number(order?.discount_amount || 0)
      const paid = Number(order?.amount || 0)
      return discount > 0 ? paid + discount : paid
    }

    const canRefundOrder = (order) => {
      if (!order || isRechargeRecord(order) || order.status !== 'paid') return false
      const method = String(order.payment_method || '').toLowerCase()
      return method.includes('yipay') || method.includes('易支付')
    }

    const refundOrder = (order) => {
      confirmAction('确定要退款此订单吗？退款后会回退订阅/设备数量和相关优惠占用。', async () => {
        await api.post(`/admin/orders/${order.id}/refund`)
      })
    }

    // Bulk Actions
    const handleSelectionChange = (selection) => {
      selectedOrders.value = selection.filter(item => item.record_type !== 'recharge')
    }

    const isOrderSelectable = (row) => row.record_type !== 'recharge'

    const handleBulkAction = async (actionType, apiPath, confirmMsg) => {
      if (selectedOrders.value.length === 0) return
      
      try {
        await ElMessageBox.confirm(confirmMsg, '提示', { type: 'warning' })
        bulkLoading.value = true
        const orderIds = selectedOrders.value
          .filter(o => o.record_type !== 'recharge')
          .map(o => o.id)
        if (orderIds.length === 0) {
          ElMessage.warning('请选择订单记录，充值记录不能执行此操作')
          return
        }
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
            cancelledOrders: s.cancelled || s.cancelled_orders || 0,
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
      'cancelled': 'info',
      'failed': 'danger',
      'expired': 'info',
      'refunded': 'warning'
    }[status] || 'info')

    const getStatusText = (status) => ({
      'pending': '待支付',
      'paid': '已支付',
      'cancelled': '已取消',
      'failed': '支付失败',
      'expired': '已过期',
      'refunded': '已退款'
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
      isOrderSelectable, isRechargeRecord,
      viewOrder, previewImage, markAsPaid, cancelOrder, deleteOrder, refundOrder, canRefundOrder,
      exportOrders, bulkMarkAsPaid, bulkCancel, bulkDelete,
      
      // Utils
      detailTitle, getStatusType, getStatusText, formatDateTime, formatMoney,
      orderExtra, extraValue, numberOrZero, numberOrDash,
      isDeviceUpgradeOrder, isCustomPackageOrder, isPackageLikeOrder,
      detailKindLabel, detailPrimaryName, detailPaidTime, detailAddedDays,
      activationModeText, orderOriginalAmount, moneyField, percentField, boolText,
      fieldOrDash, daysText, deviceCountText, monthCountText, packageTypeText,
      expireDeltaText, hasDiscountOrBalance, qrSummary, fulfillmentText, extraEntries
    }
  }
}
</script>

<style scoped lang="scss">
// 通用样式
.positive-amount { color: #67c23a; font-weight: 600; }
.text-muted { color: #909399; font-size: 12px; }
.ml-1 { margin-left: 4px; }

.selected-count { color: #409eff; font-weight: 600; font-size: 14px; }
.action-buttons-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 8px; }

.order-detail-content {
  padding: 0 2px 24px;
}

.detail-hero {
  display: flex;
  justify-content: space-between;
  gap: 20px;
  padding: 20px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: linear-gradient(135deg, #f8fafc 0%, #eef6ff 100%);
  margin-bottom: 18px;

  &.is-recharge {
    background: linear-gradient(135deg, #f7fdf8 0%, #ecf8f0 100%);
  }
}

.hero-main {
  min-width: 0;
}

.hero-kicker {
  font-size: 12px;
  color: #64748b;
  font-weight: 700;
  letter-spacing: 0;
  margin-bottom: 8px;
}

.hero-title {
  font-size: 18px;
  font-weight: 700;
  color: #111827;
  line-height: 1.35;
  word-break: break-word;
}

.hero-order-no {
  margin-top: 8px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  color: #6b7280;
  word-break: break-all;
}

.hero-side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 10px;
  flex-shrink: 0;
}

.hero-amount {
  font-size: 24px;
  font-weight: 800;
  color: #111827;

  &.is-plus {
    color: #16a34a;
  }
}

.detail-section {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  padding: 16px;
  margin-bottom: 14px;
}

.detail-section-title {
  font-size: 14px;
  font-weight: 700;
  color: #1f2937;
  margin-bottom: 14px;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;

  &.compact {
    margin-top: 14px;
  }
}

.detail-field {
  min-width: 0;
  padding: 10px 12px;
  border-radius: 6px;
  background: #f9fafb;
  border: 1px solid #f1f5f9;
}

.field-label {
  display: block;
  color: #6b7280;
  font-size: 12px;
  margin-bottom: 6px;
}

.field-value {
  color: #111827;
  font-size: 14px;
  font-weight: 600;
  word-break: break-word;

  &.mono {
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 12px;
  }

  &.discount {
    color: #dc2626;
  }

  &.positive {
    color: #16a34a;
  }

  &.accent {
    color: #2563eb;
  }
}

.change-panel {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
  align-items: stretch;
  gap: 10px;
}

.change-card {
  min-width: 0;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
  background: #f8fafc;
  padding: 14px;

  &.is-new {
    border-color: #bfdbfe;
    background: #eff6ff;
  }

  strong {
    display: block;
    color: #111827;
    font-size: 14px;
    line-height: 1.45;
    word-break: break-word;
  }
}

.change-label {
  display: block;
  color: #64748b;
  font-size: 12px;
  margin-bottom: 6px;
}

.change-arrow {
  display: flex;
  align-items: center;
  color: #64748b;
  font-weight: 700;
}

.extra-data-list {
  display: grid;
  gap: 8px;
}

.extra-data-row {
  display: grid;
  grid-template-columns: minmax(120px, 0.45fr) minmax(0, 1fr);
  gap: 12px;
  align-items: start;
  padding: 10px 12px;
  border: 1px solid #f1f5f9;
  border-radius: 6px;
  background: #f9fafb;
}

.extra-data-label {
  color: #6b7280;
  font-size: 12px;
  line-height: 1.5;
}

.extra-data-value {
  min-width: 0;
  color: #111827;
  font-size: 13px;
  font-weight: 600;
  line-height: 1.5;
  word-break: break-word;
}

.payment-proof-section {
  margin-top: 18px;
}

.proof-image {
  width: 100%;
  max-height: 420px;
  object-fit: contain;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
  cursor: pointer;
  background: #f8fafc;
}

@media (max-width: 768px) {
  .detail-hero {
    flex-direction: column;
    gap: 14px;
    padding: 16px;
  }

  .hero-side {
    align-items: flex-start;
  }

  .hero-amount {
    font-size: 22px;
  }

  .detail-grid,
  .change-panel {
    grid-template-columns: 1fr;
  }

  .extra-data-row {
    grid-template-columns: 1fr;
    gap: 4px;
  }

  .change-arrow {
    justify-content: center;
    transform: rotate(90deg);
  }

  .mobile-card-optimized {
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
