<template>
  <div class="list-container packages-container">
    <!-- 加载状态 -->
    <LoadingState v-if="isLoading" text="正在加载套餐列表..." />

    <!-- 错误状态 -->
    <ErrorState
      v-else-if="errorMessage"
      :message="errorMessage"
      @retry="loadPackages"
    />

    <!-- 空状态 -->
    <EmptyState
      v-else-if="packages.length === 0"
      type="empty"
      title="暂无可用套餐"
      description="当前没有可购买的套餐，请稍后再试"
    />

    <!-- 套餐列表 -->
    <div v-else class="packages-grid">
      <!-- 自定义套餐卡片 -->
      <el-card
        v-if="customPackageEnabled"
        class="package-card custom-package-card"
      >
        <div class="package-header">
          <h3 class="package-name">自定义套餐</h3>
          <div class="custom-badge">灵活配置</div>
        </div>
        <div class="package-price">
          <div style="display: flex; align-items: baseline; gap: 4px;">
            <span class="currency">¥</span>
            <span class="amount">{{ customPackageConfig.price_per_device_year }}</span>
            <span class="period">/设备/年</span>
          </div>
        </div>
        <div class="package-description">
          <p>根据您的需求自由选择设备数量和购买时长</p>
        </div>
        <div class="package-features">
          <ul>
            <li><i class="el-icon-check"></i>设备数：{{ customPackageConfig.min_devices }}-{{ customPackageConfig.max_devices }}个</li>
            <li><i class="el-icon-check"></i>最少购买：{{ customPackageConfig.min_months }}个月</li>
            <li><i class="el-icon-check"></i>购买越久，折扣越多</li>
            <li><i class="el-icon-check"></i>灵活配置，按需购买</li>
          </ul>
        </div>
        <div class="package-actions">
          <el-button
            type="success"
            size="large"
            @click.stop.prevent="openCustomPackageDialog"
            :loading="isProcessing"
            :disabled="isProcessing"
            style="width: 100%"
          >
            {{ isProcessing ? '处理中...' : '自定义购买' }}
          </el-button>
        </div>
      </el-card>

      <!-- 普通套餐卡片 -->
      <el-card
        v-for="pkg in packages"
        :key="pkg.id"
        class="package-card"
        :class="{ 'popular': pkg.is_popular, 'recommended': pkg.is_recommended }"
      >
        <div class="package-header">
          <h3 class="package-name">{{ pkg.name }}</h3>
          <div v-if="pkg.is_popular" class="popular-badge">热门</div>
          <div v-if="pkg.is_recommended" class="recommended-badge">推荐</div>
        </div>
        <div class="package-price">
          <div v-if="userLevel && levelDiscountRate < 1.0" style="display: flex; flex-direction: column; gap: 4px;">
            <div style="display: flex; align-items: baseline; gap: 4px;">
              <span style="text-decoration: line-through; color: #909399; font-size: 14px;">¥{{ pkg.price }}</span>
              <span class="currency">¥</span>
              <span class="amount" style="color: #f56c6c;">{{ (pkg.price * levelDiscountRate).toFixed(2) }}</span>
              <span class="period">/{{ pkg.duration_days }}天</span>
              <el-tooltip :content="`原价 ¥${pkg.price}，${userLevel.name}等级享 ${(levelDiscountRate * 10).toFixed(1)} 折优惠`" placement="top">
                <el-icon style="color:#909399;cursor:pointer;font-size:13px;"><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
            <el-tag :type="userLevel.color ? 'info' : 'success'" size="small" :style="{ backgroundColor: userLevel.color || '#67c23a', color: '#fff', border: 'none', alignSelf: 'flex-start' }">
              {{ userLevel.name }} {{ (levelDiscountRate * 10).toFixed(1) }}折
            </el-tag>
          </div>
          <div v-else style="display: flex; align-items: baseline; gap: 4px;">
            <span class="currency">¥</span>
            <span class="amount">{{ pkg.price }}</span>
            <span class="period">/{{ pkg.duration_days }}天</span>
          </div>
        </div>
        <div v-if="pkg.description && pkg.description.trim()" class="package-description">
          <p>{{ pkg.description }}</p>
        </div>
        <div v-else class="package-features">
          <ul>
            <li v-for="feature in pkg.features" :key="feature">
              <i class="el-icon-check"></i>
              {{ feature }}
            </li>
          </ul>
        </div>
        <div class="package-actions">
          <el-button 
            type="primary" 
            size="large" 
            @click.stop.prevent="selectPackage(pkg)"
            :loading="isProcessing"
            :disabled="isProcessing || !pkg || !pkg.id"
            style="width: 100%"
          >
            {{ isProcessing ? '处理中...' : '立即购买' }}
          </el-button>
        </div>
      </el-card>
    </div>
    <el-dialog
      v-model="purchaseDialogVisible"
      title="确认购买"
      :width="isMobile ? '95%' : '800px'"
      :close-on-click-modal="false"
      class="purchase-dialog"
      :class="{ 'mobile-purchase-dialog': isMobile }"
      :show-close="true"
    >
      <div class="purchase-confirm-horizontal">
        <div class="purchase-left">
          <div class="package-summary">
            <h4>订单信息</h4>
            <el-descriptions :column="1" border size="small">
              <el-descriptions-item label="套餐名称">{{ selectedPackage?.name }}</el-descriptions-item>
              <el-descriptions-item label="套餐单价">
                <span>¥{{ selectedPackage?.price }}</span>
                <span style="color: #909399; margin-left: 8px;">/{{ packageType?.type === 'monthly' ? '月' : packageType?.type === 'yearly' ? '年' : packageType?.type === 'half_yearly' ? '半年' : packageType?.type === 'quarterly' ? '季度' : `${selectedPackage?.duration_days || 30}天` }}</span>
              </el-descriptions-item>
              <el-descriptions-item label="设备限制">{{ selectedPackage?.device_limit }}个</el-descriptions-item>
            </el-descriptions>
          </div>
          <div class="duration-selection" style="margin-top: 12px;">
            <h4>购买时长</h4>
            <el-select
              v-model="selectedQuantity"
              @change="handleQuantityChange"
              style="width: 100%"
              :placeholder="durationPlaceholder"
              :size="isMobile ? 'large' : 'default'"
            >
              <el-option
                v-for="option in durationOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </el-select>
            <div class="form-hint" style="margin-top: 4px; color: #909399; font-size: 11px;">
              {{ durationHint }}
            </div>
          </div>
          <div class="coupon-section" style="margin-top: 12px; padding: 12px; background: #f5f7fa; border-radius: 4px">
            <h4 style="margin-bottom: 8px; font-size: 14px;">优惠券（可选）</h4>
            <div class="coupon-input-group">
              <el-input
                v-model="couponCode"
                placeholder="输入优惠券码"
                class="coupon-input"
                :disabled="validatingCoupon || isProcessing"
                @input="handleCouponInput"
                @focus="handleCouponFocus"
                :size="isMobile ? 'large' : 'default'"
              />
              <div class="coupon-buttons">
                <el-button
                  @click="validateCoupon"
                  :loading="validatingCoupon"
                  :disabled="!couponCode || isProcessing"
                  :size="isMobile ? 'large' : 'default'"
                >
                  验证
                </el-button>
                <el-button
                  v-if="couponCode"
                  @click="clearCoupon"
                  :disabled="isProcessing"
                  :size="isMobile ? 'large' : 'default'"
                >
                  清除
                </el-button>
              </div>
            </div>
            <div v-if="couponInfo" style="margin-top: 8px">
              <el-alert
                :title="couponInfo.message"
                :type="couponInfo.valid ? 'success' : 'error'"
                :closable="false"
                show-icon
                :effect="'plain'"
              />
              <div v-if="couponInfo.valid && couponInfo.discount_amount" style="margin-top: 6px; color: #67c23a; font-weight: bold; font-size: 13px;">
                优惠金额：¥{{ couponInfo.discount_amount.toFixed(2) }}
              </div>
            </div>
          </div>
        </div>
        <div class="purchase-right">
          <div class="price-summary">
            <h4>费用明细</h4>
            <el-descriptions :column="1" border size="small">
              <el-descriptions-item label="购买时长">
                <span>{{ durationDisplayText }}</span>
              </el-descriptions-item>
              <el-descriptions-item label="原价总计">
                <span>¥{{ totalOriginalPrice.toFixed(2) }}</span>
              </el-descriptions-item>
              <el-descriptions-item label="等级折扣" v-if="userLevel && levelDiscountRate < 1.0">
                <div class="discount-item">
                  <span class="discount-amount">
                    -¥{{ calculateLevelDiscount(totalOriginalPrice).toFixed(2) }}
                  </span>
                  <el-tag
                    :type="userLevel.color ? 'info' : 'success'"
                    size="small"
                    class="level-tag"
                    :style="{ backgroundColor: userLevel.color || '#67c23a', color: '#fff', border: 'none' }"
                  >
                    {{ userLevel.name }} {{ (levelDiscountRate * 10).toFixed(1) }}折
                  </el-tag>
                </div>
              </el-descriptions-item>
              <el-descriptions-item label="优惠券折扣" v-if="couponInfo && couponInfo.valid && couponInfo.discount_amount">
                <span class="discount-amount">-¥{{ couponInfo.discount_amount.toFixed(2) }}</span>
              </el-descriptions-item>
              <el-descriptions-item label="实付金额">
                <span class="final-amount">
                  ¥{{ finalAmount.toFixed(2) }}
                </span>
              </el-descriptions-item>
            </el-descriptions>
          </div>
          <div v-if="userLevel && levelDiscountRate < 1.0" class="level-discount-tip" style="margin-top: 12px;">
            <div class="tip-header">
              <el-icon class="tip-icon"><StarFilled /></el-icon>
              <span class="tip-title">
                您当前是 <span class="level-name-highlight" :style="{ color: userLevel.color || '#4caf50' }">{{ userLevel.name }}</span>，享受 {{ (levelDiscountRate * 10).toFixed(1) }}折优惠！
              </span>
            </div>
            <div class="tip-content">
              💡 本次购买可节省 ¥{{ calculateLevelDiscount(totalOriginalPrice).toFixed(2) }}，累计消费达到更高等级可享受更多优惠！
            </div>
          </div>
          <div v-else-if="!userLevel || levelDiscountRate >= 1.0" class="level-upgrade-tip" style="margin-top: 12px;">
            <div class="tip-header">
              <el-icon class="tip-icon upgrade-icon"><Promotion /></el-icon>
              <span class="tip-title upgrade-title">
                升级会员等级，享受更多优惠！
              </span>
            </div>
            <div class="tip-content upgrade-content">
              💡 累计消费达到一定金额即可升级会员等级，享受专属折扣优惠。立即购买即可开始累计消费！
            </div>
          </div>
          <div class="payment-method-section" style="margin-top: 12px;">
            <h4 class="payment-section-title">支付方式</h4>
            <div class="balance-info">
              <div class="balance-row">
                <span class="balance-label">账户余额：</span>
                <span class="balance-amount">¥{{ userBalance.toFixed(2) }}</span>
              </div>
            </div>
            <el-radio-group v-model="paymentMethod" @change="handlePaymentMethodChange" style="width: 100%">
              <el-radio
                label="balance"
                :disabled="userBalance < finalAmount"
                class="payment-option"
              >
                <div class="payment-option-content">
                  <span class="payment-option-label">
                    <el-icon class="payment-icon"><Wallet /></el-icon>
                    余额支付
                  </span>
                  <span v-if="userBalance >= finalAmount" class="payment-status success">（余额充足）</span>
                  <span v-else-if="userBalance > 0" class="payment-status error">
                    （余额不足，还需 ¥{{ (finalAmount - userBalance).toFixed(2) }}）
                  </span>
                  <span v-else class="payment-status disabled">（余额为0）</span>
                </div>
              </el-radio>
              <el-radio
                v-for="method in availablePaymentMethods"
                :key="method.key"
                :label="method.key"
                class="payment-option"
              >
                <div class="payment-option-content">
                  <span class="payment-option-label">
                    <el-icon class="payment-icon"><CreditCard /></el-icon>
                    {{ method.name || method.key }}
                  </span>
                </div>
              </el-radio>
              <el-radio
                v-if="availablePaymentMethods.length === 0"
                label="alipay"
                class="payment-option"
              >
                <div class="payment-option-content">
                  <span class="payment-option-label">
                    <el-icon class="payment-icon"><CreditCard /></el-icon>
                    支付宝支付
                  </span>
                </div>
              </el-radio>
            </el-radio-group>
            <div v-if="paymentMethod === 'balance' && userBalance >= finalAmount" style="margin-top: 8px; padding: 8px; background: #e1f3d8; border-radius: 4px">
              <el-alert
                title="将使用余额全额支付"
                type="success"
                :closable="false"
                show-icon
                :effect="'plain'"
              />
            </div>
          </div>
          <div class="purchase-actions" style="margin-top: 16px; padding-top: 16px; border-top: 1px solid #e4e7ed;">
            <el-button @click="purchaseDialogVisible = false" :size="isMobile ? 'large' : 'default'">取消</el-button>
            <el-button type="primary" @click="confirmPurchase" :loading="isProcessing" :size="isMobile ? 'large' : 'default'">
              确认购买
            </el-button>
          </div>
        </div>
      </div>
    </el-dialog>
    <el-dialog
      v-model="paymentQRVisible"
      title="扫码支付"
      :width="isMobile ? '92%' : '520px'"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      class="payment-qr-dialog"
    >
      <div class="payment-qr-container">
        <div class="payment-summary-card">
          <div class="summary-header">
            <div>
              <div class="summary-label">支付金额</div>
              <div class="summary-amount">¥{{ parseFloat(currentOrder?.amount || orderInfo.amount || 0).toFixed(2) }}</div>
            </div>
            <div class="summary-badge">
              {{ currentOrder?.package_name || orderInfo.packageName }}
            </div>
          </div>
          <div class="summary-meta">
            <div class="meta-item">
              <span class="meta-key">订单号</span>
              <span class="meta-value">{{ currentOrder?.order_no || orderInfo.orderNo }}</span>
            </div>
            <div class="meta-item">
              <span class="meta-key">支付方式</span>
              <span class="meta-value">{{ getPaymentMethodDisplayName(currentOrder?.payment_method || paymentMethod) }}</span>
            </div>
          </div>
        </div>

        <div class="qr-panel">
          <div class="qr-panel-header">
            <h4 v-if="isPaymentPageUrl && paymentUrl">请在页面中完成支付</h4>
            <h4 v-else>请使用支付宝扫码</h4>
            <p>支付完成后会自动刷新购买结果</p>
          </div>
          <div class="qr-code-wrapper" :class="{ 'iframe-mode': isPaymentPageUrl && paymentUrl }">
            <div v-if="isPaymentPageUrl && paymentUrl" class="payment-page-iframe">
              <iframe
                ref="paymentIframe"
                :src="paymentUrl"
                frameborder="0"
                scrolling="auto"
                style="width: 100%; min-height: 600px; border: none;"
                @load="onIframeLoad"
              ></iframe>
            </div>
            <div v-else-if="paymentQRCode" class="qr-code">
              <img
                :src="paymentQRCode.startsWith('data:') ? paymentQRCode : (paymentQRCode + '?t=' + Date.now())"
                alt="支付二维码"
                :title="getPaymentMethodDisplayName(currentOrder?.payment_method || paymentMethod) + '二维码'"
                @error="onImageError"
                @load="onImageLoad"
              />
            </div>
            <div v-else class="qr-loading">
              <el-icon class="is-loading"><Loading /></el-icon>
              <p>正在生成二维码...</p>
            </div>
          </div>
          <div class="payment-tips" v-if="!isPaymentPageUrl">
            <p class="tip-text"><el-icon><InfoFilled /></el-icon><span>请使用支付宝扫码支付</span></p>
          </div>
          <div class="payment-actions-container" v-if="isMobile && paymentUrl && (currentOrder?.payment_method === 'alipay' || paymentUrl.includes('alipay'))">
            <el-button
              type="success"
              size="default"
              class="payment-btn alipay-btn"
              @click="openAlipayApp"
              style="width: 100%;"
            >
              <el-icon class="btn-icon"><Wallet /></el-icon>
              打开支付宝App
            </el-button>
          </div>
        </div>
      </div>
    </el-dialog>
    <el-dialog
      v-model="successDialogVisible"
      title="购买成功"
      width="400px"
      :close-on-click-modal="false"
    >
      <div class="success-message">
        <el-icon class="success-icon"><CircleCheckFilled /></el-icon>
        <h3>恭喜！购买成功</h3>
        <p>您的订阅已激活，可以正常使用了。</p>
        <div class="success-actions">
          <el-button type="primary" @click="goToSubscription">查看订阅</el-button>
          <el-button @click="successDialogVisible = false">关闭</el-button>
        </div>
      </div>
    </el-dialog>

    <!-- 自定义套餐对话框 -->
    <el-dialog
      v-model="customPackageDialogVisible"
      title="自定义套餐购买"
      :width="isMobile ? '95%' : '600px'"
      :close-on-click-modal="false"
      class="custom-package-dialog"
    >
      <div class="custom-package-form">
        <el-form :model="customPackageForm" :label-width="isMobile ? '0' : '100px'" :label-position="isMobile ? 'top' : 'right'">
          <el-form-item :label="isMobile ? '' : '设备数量'">
            <div v-if="isMobile" class="mobile-form-label">设备数量</div>
            <el-input-number
              v-model="customPackageForm.devices"
              :min="customPackageConfig.min_devices"
              :max="customPackageConfig.max_devices"
              :step="1"
              @change="calculateCustomPrice"
              :size="isMobile ? 'large' : 'default'"
              style="width: 100%"
            />
            <div class="form-hint">
              可选范围：{{ customPackageConfig.min_devices }}-{{ customPackageConfig.max_devices }}个设备
            </div>
          </el-form-item>

          <el-form-item :label="isMobile ? '' : '购买月数'">
            <div v-if="isMobile" class="mobile-form-label">购买月数</div>
            <el-input-number
              v-model="customPackageForm.months"
              :min="customPackageConfig.min_months"
              :max="120"
              :step="1"
              @change="calculateCustomPrice"
              :size="isMobile ? 'large' : 'default'"
              style="width: 100%"
            />
            <div class="form-hint">
              最少购买{{ customPackageConfig.min_months }}个月，最多120个月
            </div>
          </el-form-item>

          <el-form-item :label="isMobile ? '' : '优惠券'">
            <div v-if="isMobile" class="mobile-form-label">优惠券（可选）</div>
            <div class="coupon-input-group">
              <el-input
                v-model="customPackageForm.couponCode"
                placeholder="输入优惠券码（可选）"
                @input="handleCustomCouponInput"
                :size="isMobile ? 'large' : 'default'"
              />
              <el-button
                @click="validateCustomCoupon"
                :loading="validatingCustomCoupon"
                :disabled="!customPackageForm.couponCode"
                :size="isMobile ? 'large' : 'default'"
              >
                验证
              </el-button>
            </div>
            <div v-if="customCouponInfo" style="margin-top: 8px">
              <el-alert
                :title="customCouponInfo.message"
                :type="customCouponInfo.valid ? 'success' : 'error'"
                :closable="false"
                show-icon
              />
            </div>
          </el-form-item>

          <el-divider />

          <div class="price-summary-custom">
            <h4 style="margin: 0 0 12px 0; font-size: 15px; color: #303133;">费用明细</h4>
            <el-descriptions :column="1" border size="small">
              <el-descriptions-item label="单价">
                ¥{{ customPackageConfig.price_per_device_year }}/设备/年
              </el-descriptions-item>
              <el-descriptions-item label="设备数量">
                {{ customPackageForm.devices }}个
              </el-descriptions-item>
              <el-descriptions-item label="购买时长">
                {{ customPackageForm.months }}个月
              </el-descriptions-item>
              <el-descriptions-item label="基础价格">
                ¥{{ customPackagePrice.basePrice.toFixed(2) }}
              </el-descriptions-item>
              <el-descriptions-item label="时长折扣" v-if="customPackagePrice.discountPercent > 0">
                <span class="discount-amount">
                  -¥{{ (customPackagePrice.basePrice * customPackagePrice.discountPercent / 100).toFixed(2) }}
                  ({{ customPackagePrice.discountPercent }}%折扣)
                </span>
              </el-descriptions-item>
              <el-descriptions-item label="优惠券折扣" v-if="customCouponInfo && customCouponInfo.valid && customPackagePrice.couponDiscount > 0">
                <span class="discount-amount">-¥{{ customPackagePrice.couponDiscount.toFixed(2) }}</span>
              </el-descriptions-item>
              <el-descriptions-item label="实付金额">
                <span class="final-amount">¥{{ customPackagePrice.finalPrice.toFixed(2) }}</span>
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </el-form>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="customPackageDialogVisible = false" :size="isMobile ? 'large' : 'default'">取消</el-button>
          <el-button
            type="primary"
            @click="confirmCustomPackage"
            :loading="isProcessing"
            :disabled="isProcessing"
            :size="isMobile ? 'large' : 'default'"
          >
            确认购买
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>
<script>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CircleCheckFilled, Loading, Wallet, CreditCard, Money, StarFilled, Promotion, QuestionFilled } from '@element-plus/icons-vue'
import { useApi, couponAPI, userAPI, userLevelAPI, orderAPI, parsePaymentMethods, cachedAPI } from '@/utils/api'
import EmptyState from '@/components/EmptyState.vue'
import LoadingState from '@/components/LoadingState.vue'
import ErrorState from '@/components/ErrorState.vue'
export default {
  name: 'Packages',
  components: {
    CircleCheckFilled,
    Loading,
    Wallet,
    CreditCard,
    Money,
    StarFilled,
    Promotion,
    EmptyState,
    LoadingState,
    ErrorState
  },
  setup() {
    const router = useRouter()
    const api = useApi()
    const packages = ref([])
    const isLoading = ref(false)
    const errorMessage = ref('')
    const isProcessing = ref(false)
    const purchaseDialogVisible = ref(false)
    const paymentQRVisible = ref(false)
    const successDialogVisible = ref(false)
    const selectedPackage = ref(null)
    const selectedQuantity = ref(1)
    const currentOrder = ref(null)
    const paymentQRCode = ref('')
    const paymentUrl = ref('')
    const isPaymentPageUrl = computed(() => {
      if (!paymentUrl.value) return false
      const url = String(paymentUrl.value).toLowerCase()
      return url.includes('payapi/pay/payment') ||
             url.includes('submit.php') ||
             (url.startsWith('http') && !url.includes('qrcode') && !url.includes('qr.alipay') && !url.startsWith('weixin://') && !url.startsWith('wxp://'))
    })
    const isCheckingPayment = ref(false)
    let paymentStatusCheckInterval = null
    const couponCode = ref('')
    const validatingCoupon = ref(false)
    const couponInfo = ref(null)
    const paymentMethod = ref('alipay')

    // 自定义套餐相关
    const customPackageEnabled = ref(false)
    const customPackageConfig = reactive({
      price_per_device_year: 40,
      min_devices: 5,
      max_devices: 100,
      min_months: 6,
      duration_discounts: []
    })
    const customPackageDialogVisible = ref(false)
    const customPackageForm = reactive({
      devices: 5,
      months: 6,
      couponCode: ''
    })
    const validatingCustomCoupon = ref(false)
    const customCouponInfo = ref(null)
    const customPackagePrice = reactive({
      basePrice: 0,
      discountPercent: 0,
      couponDiscount: 0,
      finalPrice: 0
    })
    const availablePaymentMethods = ref([])
    const userBalance = ref(0)
    const userLevel = ref(null)
    const levelDiscountRate = ref(1.0)
    const windowWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1920)
    const isMobile = computed(() => {
      return windowWidth.value <= 768
    })
    const handleResize = () => {
      if (typeof window !== 'undefined') {
        windowWidth.value = window.innerWidth
      }
    }
    const handleCouponInput = (value) => {
      couponCode.value = value
    }
    const handleCouponFocus = () => {
    }
    const validateCoupon = async () => {
      if (!couponCode.value || !couponCode.value.trim()) {
        ElMessage.warning('请输入优惠券码')
        return
      }
      if (!selectedPackage.value) {
        ElMessage.warning('请先选择套餐')
        return
      }
      validatingCoupon.value = true
      try {
        const totalPrice = totalOriginalPrice.value
        const levelDiscountedPrice = totalPrice * levelDiscountRate.value
        const response = await couponAPI.validateCoupon({
          code: couponCode.value.trim(),
          package_id: selectedPackage.value.id,
          amount: levelDiscountedPrice
        })
        if (response.data && response.data.success) {
          couponInfo.value = {
            valid: true,
            message: '优惠券验证成功',
            discount_amount: response.data.data?.discount_amount || 0
          }
          ElMessage.success('优惠券验证成功')
        } else {
          couponInfo.value = {
            valid: false,
            message: response.data?.message || '优惠券验证失败'
          }
          ElMessage.error(response.data?.message || '优惠券验证失败')
        }
        } catch (error) {
        const errorMsg = error.response?.data?.message || error.message || '验证优惠券失败'
        couponInfo.value = {
          valid: false,
          message: errorMsg
        }
        ElMessage.error(errorMsg)
      } finally {
        validatingCoupon.value = false
      }
    }
    const clearCoupon = () => {
      couponCode.value = ''
      couponInfo.value = null
    }
    const getPaymentMethodDisplayName = (method) => {
      if (!method) return '支付宝'
      const methodStr = String(method).toLowerCase()
      if (methodStr.includes('yipay_wxpay') || methodStr.includes('易支付-微信') || methodStr.includes('wxpay')) {
        return '微信'
      } else if (methodStr.includes('yipay_alipay') || methodStr.includes('易支付-支付宝') || methodStr.includes('alipay')) {
        return '支付宝'
      } else if (methodStr.includes('yipay_qqpay') || methodStr.includes('易支付-qq')) {
        return 'QQ钱包'
      } else if (methodStr.includes('wechat') || methodStr.includes('微信')) {
        return '微信'
      } else if (methodStr.includes('alipay') || methodStr.includes('支付宝')) {
        return '支付宝'
      }
      return '支付宝'
    }
    const orderInfo = reactive({
      orderNo: '',
      packageName: '',
      amount: 0,
      duration: 0,
      paymentUrl: ''
    })
    const calculateLevelDiscount = (price) => {
      if (!price || levelDiscountRate.value >= 1.0) return 0
      return price * (1 - levelDiscountRate.value)
    }
    const packageType = computed(() => {
      if (!selectedPackage.value) return null
      const days = selectedPackage.value.duration_days || 30
      if (days >= 28 && days <= 32) {
        return { type: 'monthly', unit: '个月', days: 30, max: 12 }
      } else if (days >= 88 && days <= 92) {
        return { type: 'quarterly', unit: '个季度', days: 90, max: 8 }
      } else if (days >= 175 && days <= 185) {
        return { type: 'half_yearly', unit: '个半年', days: 180, max: 6 }
      } else if (days >= 360 && days <= 370) {
        return { type: 'yearly', unit: '年', days: 365, max: 5 }
      } else if (days >= 720 && days <= 730) {
        return { type: 'two_yearly', unit: '个两年', days: 730, max: 3 }
      } else {
        return { type: 'custom', unit: '个周期', days: days, max: 12 }
      }
    })
    const durationOptions = computed(() => {
      if (!packageType.value) return []
      const type = packageType.value
      const options = []
      for (let i = 1; i <= type.max; i++) {
        const totalDays = type.days * i
        let label = ''
        if (type.type === 'monthly') {
          label = `${i} 个月（${totalDays} 天）`
        } else if (type.type === 'quarterly') {
          label = `${i} 个季度（${totalDays} 天）`
        } else if (type.type === 'half_yearly') {
          label = `${i} 个半年（${totalDays} 天）`
        } else if (type.type === 'yearly') {
          label = `${i} 年（${totalDays} 天）`
        } else if (type.type === 'two_yearly') {
          label = `${i} 个两年（${totalDays} 天）`
        } else {
          label = `${i} 个周期（${totalDays} 天）`
        }
        options.push({
          value: i,
          label: label
        })
      }
      return options
    })
    const durationPlaceholder = computed(() => {
      if (!packageType.value) return '请选择购买时长'
      const type = packageType.value
      if (type.type === 'monthly') return '请选择购买月数'
      if (type.type === 'quarterly') return '请选择购买季度数'
      if (type.type === 'half_yearly') return '请选择购买半年数'
      if (type.type === 'yearly') return '请选择购买年数'
      if (type.type === 'two_yearly') return '请选择购买两年数'
      return '请选择购买数量'
    })
    const durationHint = computed(() => {
      if (!packageType.value) return '选择购买数量，价格将按比例计算'
      const type = packageType.value
      if (type.type === 'monthly') return '选择购买月数，价格将按比例计算'
      if (type.type === 'quarterly') return '选择购买季度数，价格将按比例计算'
      if (type.type === 'half_yearly') return '选择购买半年数，价格将按比例计算'
      if (type.type === 'yearly') return '选择购买年数，价格将按比例计算'
      if (type.type === 'two_yearly') return '选择购买两年数，价格将按比例计算'
      return '选择购买数量，价格将按比例计算'
    })
    const durationDisplayText = computed(() => {
      if (!selectedPackage.value || !selectedQuantity.value || !packageType.value) return ''
      const type = packageType.value
      const totalDays = type.days * selectedQuantity.value
      if (type.type === 'monthly') {
        return `${selectedQuantity.value} 个月（${totalDays} 天）`
      } else if (type.type === 'quarterly') {
        return `${selectedQuantity.value} 个季度（${totalDays} 天）`
      } else if (type.type === 'half_yearly') {
        return `${selectedQuantity.value} 个半年（${totalDays} 天）`
      } else if (type.type === 'yearly') {
        return `${selectedQuantity.value} 年（${totalDays} 天）`
      } else if (type.type === 'two_yearly') {
        return `${selectedQuantity.value} 个两年（${totalDays} 天）`
      } else {
        return `${selectedQuantity.value} 个周期（${totalDays} 天）`
      }
    })
    const totalOriginalPrice = computed(() => {
      if (!selectedPackage.value || !selectedQuantity.value) return 0
      const singlePrice = parseFloat(selectedPackage.value.price) || 0
      return singlePrice * selectedQuantity.value
    })
    const totalDurationDays = computed(() => {
      if (!selectedPackage.value || !selectedQuantity.value || !packageType.value) return 0
      return packageType.value.days * selectedQuantity.value
    })
    const finalAmount = computed(() => {
      if (!selectedPackage.value || !selectedQuantity.value) return 0
      const totalPrice = totalOriginalPrice.value
      const levelDiscount = calculateLevelDiscount(totalPrice)
      const couponDiscount = (couponInfo.value && couponInfo.value.valid && couponInfo.value.discount_amount) 
        ? couponInfo.value.discount_amount 
        : 0
      return Math.max(0, totalPrice - levelDiscount - couponDiscount)
    })
    const handleQuantityChange = () => {
      if (couponCode.value && couponInfo.value && couponInfo.value.valid) {
        validateCoupon()
      }
    }
    
    // --- 核心优化：并发加载设置和套餐 ---
    const loadPackages = async () => {
      try {
        isLoading.value = true
        errorMessage.value = ''

        // 利用 Promise.all 机制，使获取公共设置和获取套餐列表这两个互相不依赖的请求并发执行，
        // 且不会因为 settings 接口失败而阻断 packages 接口。
        const settingsPromise = api.get('/settings/public-settings').catch(error => {
            console.error('加载自定义套餐配置失败:', error);
            return null;
        });
        const packagesPromise = api.get('/packages/');

        // 1. 等待并处理设置接口数据
        const settingsResponse = await settingsPromise;
        if (settingsResponse && settingsResponse.data && settingsResponse.data.data) {
          const settings = settingsResponse.data.data
          customPackageEnabled.value = settings.custom_package_enabled === true
          if (customPackageEnabled.value) {
            customPackageConfig.price_per_device_year = parseFloat(settings.custom_package_price_per_device_year || 40)
            customPackageConfig.min_devices = parseInt(settings.custom_package_min_devices || 5)
            customPackageConfig.max_devices = parseInt(settings.custom_package_max_devices || 100)
            customPackageConfig.min_months = parseInt(settings.custom_package_min_months || 6)

            // 解析折扣配置
            if (settings.custom_package_duration_discounts) {
              try {
                const discounts = typeof settings.custom_package_duration_discounts === 'string'
                  ? JSON.parse(settings.custom_package_duration_discounts)
                  : settings.custom_package_duration_discounts
                if (Array.isArray(discounts)) {
                  customPackageConfig.duration_discounts = discounts
                }
              } catch (e) {
                console.error('解析折扣配置失败:', e)
              }
            }

            // 初始化表单默认值
            customPackageForm.devices = customPackageConfig.min_devices
            customPackageForm.months = customPackageConfig.min_months
            calculateCustomPrice()
          }
        }

        // 2. 等待并处理套餐接口数据
        const response = await packagesPromise;
        let packagesList = []
        if (response && response.data) {
          const responseData = response.data
          if (responseData.data && responseData.data.packages && Array.isArray(responseData.data.packages)) {
            packagesList = responseData.data.packages
          } else if (Array.isArray(responseData.data)) {
            packagesList = responseData.data
          } else if (responseData.packages && Array.isArray(responseData.packages)) {
            packagesList = responseData.packages
          } else if (Array.isArray(responseData)) {
            packagesList = responseData
          } else if (responseData.data && typeof responseData.data === 'object' && !Array.isArray(responseData.data)) {
            if (responseData.data.id || responseData.data.name) {
              packagesList = [responseData.data]
            }
          }
        }
        if (packagesList && Array.isArray(packagesList) && packagesList.length > 0) {
          packages.value = packagesList.map(pkg => ({
            ...pkg,
            features: [
              `有效期 ${pkg.duration_days} 天`,
              `支持 ${pkg.device_limit} 个设备`,
              '7×24小时技术支持',
              '高速稳定节点'
            ],
            is_recommended: pkg.is_recommended === true || pkg.is_recommended === 1 || pkg.is_recommended === '1' || pkg.is_recommended === 'true',
            is_popular: pkg.is_popular === true || pkg.is_popular === 1 || pkg.is_popular === '1' || pkg.is_popular === 'true' || pkg.sort_order === 2
          }))
          errorMessage.value = ''
        } else {
          packages.value = []
          errorMessage.value = ''
        }
      } catch (error) {
        if (error.response?.status === 404) {
          errorMessage.value = '套餐服务暂时不可用'
        } else if (error.response?.status === 500) {
          errorMessage.value = '服务器内部错误'
        } else if (error.code === 'ECONNREFUSED') {
          errorMessage.value = '无法连接到服务器'
        } else {
          const errorMsg = error.response?.data?.detail || error.response?.data?.message || error.message || '加载套餐列表失败，请重试'
          errorMessage.value = errorMsg
        }
        packages.value = [] // 确保清空套餐列表
      } finally {
        isLoading.value = false
      }
    }
    const loadUserBalance = async () => {
      try {
        const response = await userAPI.getUserInfo()
        if (response.data && response.data.success && response.data.data) {
          userBalance.value = parseFloat(response.data.data.balance || 0)
          if (response.data.data.user_level) {
            userLevel.value = response.data.data.user_level
            levelDiscountRate.value = parseFloat(userLevel.value.discount_rate || 1.0)
          } else {
            try {
              const levelResponse = await userLevelAPI.getMyLevel()
              if (levelResponse?.data?.data?.current_level) {
                userLevel.value = levelResponse.data.data.current_level
                levelDiscountRate.value = parseFloat(userLevel.value.discount_rate || 1.0)
              }
            } catch (e) {
              // 用户等级信息加载失败，使用默认折扣率
            }
          }
        }
      } catch (error) {
        userBalance.value = 0
        userLevel.value = null
        levelDiscountRate.value = 1.0
      }
    }
    const loadPaymentMethods = async () => {
      try {
        const response = await api.get('/payment-methods/active')
        availablePaymentMethods.value = parsePaymentMethods(response)
      } catch (error) {
        availablePaymentMethods.value = [
          { key: 'alipay', name: '支付宝' },
          { key: 'yipay', name: '易支付' }
        ]
      }
    }
    const handlePaymentMethodChange = (value) => {
    }
    
    // --- 优化弹窗时的请求阻塞 ---
    const selectPackage = async (pkg) => {
      try {
        if (!pkg) {
          ElMessage.error('套餐信息错误，请刷新页面重试')
          return
        }
        if (!pkg.id) {
          ElMessage.error('套餐ID缺失，请刷新页面重试')
          return
        }
        selectedPackage.value = pkg
        selectedQuantity.value = 1
        
        // 并发获取最新余额和支付方式，减少弹窗显示前的等待时间
        await Promise.all([
          loadUserBalance(),
          loadPaymentMethods()
        ]);
        
        const finalPrice = finalAmount.value
        if (userBalance.value >= finalPrice && userBalance.value > 0) {
          paymentMethod.value = 'balance'
        } else {
          paymentMethod.value = availablePaymentMethods.value[0]?.key || 'alipay'
        }
        purchaseDialogVisible.value = true
      } catch (error) {
        ElMessage.error('选择套餐失败: ' + error.message)
      }
    }
    const confirmPurchase = async () => {
      // 如果是自定义套餐订单（已创建），直接调用支付接口
      if (currentOrder.value && currentOrder.value.order_no && selectedPackage.value && selectedPackage.value.id === 0) {
        try {
          if (isProcessing.value) return
          isProcessing.value = true

          // 查找支付方式ID
          const selectedPaymentMethod = availablePaymentMethods.value.find(
            method => method.key === paymentMethod.value || method.pay_type === paymentMethod.value
          )

          if (!selectedPaymentMethod && paymentMethod.value !== 'balance') {
            ElMessage.error('请选择有效的支付方式')
            isProcessing.value = false
            return
          }

          const payData = {
            payment_method: paymentMethod.value
          }

          // 如果不是余额支付，需要传payment_method_id
          if (paymentMethod.value !== 'balance' && selectedPaymentMethod) {
            payData.payment_method_id = selectedPaymentMethod.id
          }

          const response = await api.post(`/orders/${currentOrder.value.order_no}/pay`, payData)

          if (response.data && response.data.success) {
            const order = response.data.data || response.data

            if (order.status === 'paid') {
              purchaseDialogVisible.value = false
              ElMessage.success('支付成功！')
              successDialogVisible.value = true
              await loadPackages()
            } else if (order.payment_url || order.payment_qr_code) {
              purchaseDialogVisible.value = false
              await showPaymentQRCode(order)
            } else {
              ElMessage.error('支付链接生成失败')
            }
          }
        } catch (error) {
          ElMessage.error(error.response?.data?.message || '支付失败')
        } finally {
          isProcessing.value = false
        }
        return
      }

      // 普通套餐购买流程
      if (paymentMethod.value === 'balance' && userBalance.value < finalAmount.value) {
        ElMessage.error(`余额不足，当前余额：¥${userBalance.value.toFixed(2)}，需要：¥${finalAmount.value.toFixed(2)}`)
        return
      }
      try {
        if (isProcessing.value) {
          return
        }
        isProcessing.value = true
        const orderData = {
          package_id: selectedPackage.value.id,
          payment_method: paymentMethod.value === 'balance' ? 'balance' : paymentMethod.value,
          amount: finalAmount.value,
          currency: 'CNY',
          duration_months: selectedQuantity.value
        }
        if (couponInfo.value && couponInfo.value.valid && couponCode.value) {
          orderData.coupon_code = couponCode.value.trim()
        }
        if (paymentMethod.value === 'balance') {
          orderData.use_balance = true
        }
        const response = await api.post('/orders/', orderData, {
          timeout: 25000  // 25秒超时，与后端20秒读取超时+5秒缓冲匹配
        }).catch(error => {
          if (error.code === 'ECONNABORTED' || error.message?.includes('timeout')) {
            throw new Error('请求超时，支付宝服务响应较慢，请稍后重试或前往订单页面查看')
          } else if (error.response) {
            const errorMsg = error.response.data?.message || error.response.data?.detail || '创建订单失败'
            const headers = error.response.headers || {}
            const requiresConversion = headers['x-requires-conversion'] === 'true'
            const remainingDays = headers['x-remaining-days'] || '0'
            const remainingValue = headers['x-remaining-value'] || '0'
            if (requiresConversion) {
              const conversionError = new Error(errorMsg)
              conversionError.requiresConversion = true
              conversionError.remainingDays = remainingDays ? parseInt(remainingDays) : 0
              conversionError.remainingValue = remainingValue ? parseFloat(remainingValue) : 0
              throw conversionError
            }
            throw new Error(errorMsg)
          } else {
            throw new Error('网络连接失败，请检查网络连接后重试')
          }
        })
        let order = null
        if (response.data) {
          if (response.data.success !== false) {
            order = response.data.data || response.data
          } else {
            throw new Error(response.data.message || '创建订单失败')
          }
        } else {
          throw new Error('订单创建响应格式错误')
        }
        if (!order) {
          throw new Error('订单创建失败：未返回订单数据')
        }
        orderInfo.orderNo = order.order_no || order.orderNo || order.order_id || ''
        orderInfo.packageName = selectedPackage.value.name
        orderInfo.amount = order.amount
        orderInfo.duration = selectedPackage.value.duration_days
        const orderPaymentMethod = order.payment_method || paymentMethod.value
        order.payment_method = orderPaymentMethod
        if (order.status === 'paid') {
          purchaseDialogVisible.value = false
          ElMessage.success('购买成功！订单已支付')
          if (order.remaining_balance !== undefined) {
            userBalance.value = order.remaining_balance
          }
          successDialogVisible.value = true
          await loadPackages()
        } else if (order.payment_url || order.payment_qr_code) {
          purchaseDialogVisible.value = false
          orderInfo.orderNo = order.order_no || order.orderNo
          orderInfo.packageName = selectedPackage.value.name
          orderInfo.amount = order.amount
          orderInfo.duration = selectedPackage.value.duration_days
          orderInfo.paymentUrl = order.payment_url || order.payment_qr_code
          if (!order.payment_method) {
            order.payment_method = paymentMethod.value
          }
          const paymentMethodName = order.payment_method || paymentMethod.value
          const isYipay = paymentMethodName && (
            paymentMethodName.includes('yipay') ||
            paymentMethodName.includes('易支付')
          )
          if (isYipay) {
            const paymentUrl = order.payment_url || order.payment_qr_code
            if (paymentUrl) {
              const isInWeChat = /MicroMessenger/i.test(navigator.userAgent)
              const isWxpayMethod = paymentMethodName && (
                paymentMethodName.includes('wxpay') || 
                paymentMethodName.includes('微信')
              )
              if (isInWeChat && isWxpayMethod && (paymentUrl.startsWith('http://') || paymentUrl.startsWith('https://'))) {
                ElMessage.info('正在跳转到微信支付页面...')
                window.location.href = paymentUrl
                return
              }
              if (paymentUrl.startsWith('weixin://') || paymentUrl.startsWith('wxp://')) {
                ElMessage.info('正在唤起微信支付...')
                window.location.href = paymentUrl
                const handleVisibilityChange = async () => {
                  if (document.visibilityState === 'visible') {
                    await checkPaymentStatus()
                    document.removeEventListener('visibilitychange', handleVisibilityChange)
                  }
                }
                document.addEventListener('visibilitychange', handleVisibilityChange)
                startPaymentStatusCheck()
              } else {
                ElMessage.info('正在跳转到支付页面...')
                window.location.href = paymentUrl
              }
            } else {
              ElMessage.error('支付链接不存在')
            }
          } else {
            try {
              await showPaymentQRCode(order)
            } catch (error) {
              console.error('显示支付二维码失败:', error)
              ElMessage.error('显示支付二维码失败: ' + (error.message || '未知错误'))
            }
          }
        } else {
          const errorMsg = order.payment_error || order.note || '支付链接生成失败，可能是网络问题或支付宝配置问题'
          const orderNo = order.order_no || order.orderNo || '未知'
          ElMessageBox.confirm(
            `${errorMsg}。订单已创建成功（订单号：${orderNo}），您可以：\n\n1. 前往订单页面重新生成支付链接\n2. 稍后重试`,
            '支付链接生成失败',
            {
              confirmButtonText: '前往订单页面',
              cancelButtonText: '稍后重试',
              type: 'warning',
              distinguishCancelAndClose: true
            }
          ).then(() => {
            router.push('/orders')
          }).catch(() => {
          })
          purchaseDialogVisible.value = false
        }
      } catch (error) {
        if (error.requiresConversion) {
          const remainingDays = error.remainingDays || 0
          const remainingValue = error.remainingValue || 0
          const errorMessage = error.message || '您当前有高级套餐，无法购买低等级套餐'
          const conversionMessage = `${errorMessage}\n\n` +
            `📊 折算详情：\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `剩余天数：${remainingDays} 天\n` +
            `可折算金额：¥${remainingValue.toFixed(2)}\n\n` +
            `📐 折算公式：\n` +
            `折算金额 = 剩余天数 × (原套餐价格 ÷ 原套餐天数)\n\n` +
            `⚠️ 重要提示：\n` +
            `折算后，您的设备和时间都将清零，然后可以购买新套餐。\n` +
            `折算操作不可撤销，请谨慎操作。`
          ElMessageBox.confirm(
            conversionMessage,
            '需要折算套餐',
            {
              confirmButtonText: '立即折算',
              cancelButtonText: '取消',
              type: 'warning',
              distinguishCancelAndClose: true,
              dangerouslyUseHTMLString: false
            }
          ).then(async () => {
            try {
              isProcessing.value = true
              const { subscriptionAPI } = await import('@/utils/api')
              const response = await subscriptionAPI.convertToBalance()
              if (response.data && response.data.success) {
                const data = response.data.data || {}
                const convertedAmount = data.converted_amount || data.balance_added || remainingValue
                const dailyPrice = data.daily_price || 0
                const originalPackagePrice = data.original_package_price || 0
                const originalPackageDays = data.original_package_days || 0
                let successMessage = `套餐折算成功！\n\n`
                successMessage += `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`
                successMessage += `已返还金额：¥${convertedAmount.toFixed(2)}\n`
                if (originalPackagePrice > 0 && originalPackageDays > 0) {
                  successMessage += `原套餐价格：¥${originalPackagePrice.toFixed(2)}\n`
                  successMessage += `原套餐天数：${originalPackageDays} 天\n`
                  successMessage += `每天单价：¥${dailyPrice.toFixed(2)}\n`
                  successMessage += `剩余天数：${data.remaining_days || remainingDays} 天\n`
                  successMessage += `折算金额：¥${convertedAmount.toFixed(2)}\n`
                }
                successMessage += `当前余额：¥${data.new_balance?.toFixed(2) || '0.00'}\n`
                ElMessage.success(successMessage)
                await loadUserBalance()
                purchaseDialogVisible.value = false
                ElMessageBox.alert(
                  '套餐已折算成余额，您现在可以购买新套餐了。',
                  '折算成功',
                  {
                    confirmButtonText: '确定',
                    type: 'success'
                  }
                )
              } else {
                ElMessage.error(response.data?.message || '折算失败，请重试')
              }
            } catch (convertError) {
              const convertErrorMsg = convertError.response?.data?.message || convertError.message || '折算失败，请重试'
              ElMessage.error(convertErrorMsg)
            } finally {
              isProcessing.value = false
            }
          }).catch(() => {
          })
        } else {
          const errorMessage = error.response?.data?.detail || error.response?.data?.message || error.message || '创建订单失败，请重试'
          ElMessage.error(errorMessage)
        }
      } finally {
        isProcessing.value = false
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
    const showPaymentQRCode = async (order) => {
      try {
        const url = order.payment_url || order.payment_qr_code || orderInfo.paymentUrl
        if (!url) {
          ElMessage.error('支付链接生成失败，请重试或前往订单页面重新生成')
          return
        }
        paymentUrl.value = url
        const orderPaymentMethod = order.payment_method || paymentMethod.value
        currentOrder.value = {
          order_no: order.order_no || orderInfo.orderNo,
          amount: order.amount || orderInfo.amount,
          package_name: orderInfo.packageName || selectedPackage.value?.name,
          payment_method: orderPaymentMethod
        }
        const paymentMethodForQR = orderPaymentMethod
      try {
        if (!url || url.trim() === '') {
          ElMessage.error('支付链接为空，请联系管理员检查配置')
          return
        }
        const urlString = String(url).trim()

        // 检查是否是支付宝页面支付URL（需要跳转而不是生成二维码）
        const isAlipayPagePay = urlString.includes('alipay.com') &&
                                (urlString.includes('?') || urlString.includes('&'))

        // 检查是否是易支付页面
        const isYipayPaymentPage = urlString.includes('payApi/pay/payment') ||
                                   urlString.includes('payapi/pay/payment') ||
                                   urlString.includes('submit.php')

        if (isAlipayPagePay) {
          // 支付宝页面支付，直接跳转
          ElMessage.info('正在跳转到支付宝支付页面...')
          window.location.href = urlString
          return
        } else if (isYipayPaymentPage) {
          // 易支付页面，使用iframe
          paymentQRCode.value = ''
        } else {
          // 生成二维码
          const QRCode = await import('qrcode')
          const isMobileDevice = window.innerWidth <= 768
          const qrOptions = {
            width: isMobileDevice ? 200 : 256,
            margin: 2,
            color: {
              dark: '#000000',
              light: '#FFFFFF'
            },
            errorCorrectionLevel: 'M'
          }
          const qrCodeDataURL = await QRCode.toDataURL(urlString, qrOptions)
          paymentQRCode.value = qrCodeDataURL
        }
      } catch (error) {
        console.error('处理支付链接失败:', error)
        ElMessage.error('处理支付链接失败: ' + (error.message || '未知错误'))
        return
      }
        paymentQRVisible.value = true
        await new Promise(resolve => setTimeout(resolve, 100))
        startPaymentStatusCheck()
      } catch (error) {
        console.error('showPaymentQRCode 错误:', error)
        ElMessage.error('显示支付二维码失败: ' + (error.message || '未知错误'))
        throw error
      }
    }
    const paymentIframe = ref(null)
    const onIframeLoad = (event) => {
      const iframe = event.target
      try {
        const iframeUrl = iframe.contentWindow?.location?.href || iframe.src
        if (iframeUrl && (
          iframeUrl.includes('success') || 
          iframeUrl.includes('paid') || 
          iframeUrl.includes('支付成功') ||
          iframeUrl.includes('支付完成') ||
          iframeUrl.includes('callback') ||
          iframeUrl.includes('return')
        )) {
          setTimeout(() => {
            checkPaymentStatus()
          }, 1000)
        }
      } catch (e) {
        // 页面可见性变化处理失败，不影响主功能
      }
      const iframeCheckInterval = setInterval(() => {
        try {
          if (iframe && iframe.contentWindow) {
            const currentUrl = iframe.contentWindow.location.href
            if (currentUrl && (
              currentUrl.includes('success') || 
              currentUrl.includes('paid') || 
              currentUrl.includes('支付成功') ||
              currentUrl.includes('支付完成') ||
              currentUrl.includes('callback') ||
              currentUrl.includes('return')
            )) {
              clearInterval(iframeCheckInterval)
              setTimeout(() => {
                checkPaymentStatus()
              }, 1000)
            }
          }
        } catch (e) {
          // iframe 检查失败，不影响主功能
        }
      }, 2000)
      setTimeout(() => {
        clearInterval(iframeCheckInterval)
      }, 10000)
      startPaymentStatusCheck()
    }
    let visibilityChangeHandler = null
    let focusHandler = null
    const startPaymentStatusCheck = () => {
      if (paymentStatusCheckInterval) {
        clearInterval(paymentStatusCheckInterval)
        paymentStatusCheckInterval = null
      }
      if (visibilityChangeHandler) {
        document.removeEventListener('visibilitychange', visibilityChangeHandler)
      }
      if (focusHandler) {
        window.removeEventListener('focus', focusHandler)
      }
      checkPaymentStatus()
      paymentStatusCheckInterval = setInterval(async () => {
        await checkPaymentStatus()
      }, 5000)
      visibilityChangeHandler = async () => {
        if (document.visibilityState === 'visible' && paymentQRVisible.value) {
          await checkPaymentStatus()
        }
      }
      document.addEventListener('visibilitychange', visibilityChangeHandler)
      focusHandler = async () => {
        if (paymentQRVisible.value) {
          await checkPaymentStatus()
        }
      }
      window.addEventListener('focus', focusHandler)
      setTimeout(() => {
        if (paymentStatusCheckInterval) {
          clearInterval(paymentStatusCheckInterval)
          paymentStatusCheckInterval = null
        }
        if (visibilityChangeHandler) {
          document.removeEventListener('visibilitychange', visibilityChangeHandler)
          visibilityChangeHandler = null
        }
        if (focusHandler) {
          window.removeEventListener('focus', focusHandler)
          focusHandler = null
        }
      }, 30 * 60 * 1000)
    }
    const checkPaymentStatus = async () => {
      if (!currentOrder.value || !currentOrder.value.order_no) {
        return
      }
      if (!paymentQRVisible.value) {
        return
      }
      try {
        isCheckingPayment.value = true
        const response = await api.get(`/orders/${currentOrder.value.order_no}/status`, {
          timeout: 10000
        })
        if (process.env.NODE_ENV === 'development') {
          console.log('订单状态检查:', {
            order_no: currentOrder.value.order_no,
            response: response.data,
            status: response.data?.data?.status
          })
        }
        if (!response || !response.data) {
          return
        }
        if (response.data.success === false) {
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
          if (visibilityChangeHandler) {
            document.removeEventListener('visibilitychange', visibilityChangeHandler)
            visibilityChangeHandler = null
          }
          if (focusHandler) {
            window.removeEventListener('focus', focusHandler)
            focusHandler = null
          }
          paymentQRVisible.value = false
          successDialogVisible.value = true
          ElMessage.success('支付成功！您的订阅已激活')
          isCheckingPayment.value = false
          const refreshUserInfo = async () => {
            try {
              const userResponse = await userAPI.getUserInfo()
              if (userResponse?.data?.success) {
                userBalance.value = parseFloat(userResponse.data.data.balance || 0)
              }
            } catch (refreshError) {
              if (process.env.NODE_ENV === 'development') {
                console.error('刷新用户信息失败:', refreshError)
              }
            }
          }
          const refreshSubscription = async () => {
            try {
              const { subscriptionAPI } = await import('@/utils/api')
              const subscriptionResponse = await subscriptionAPI.getUserSubscription()
              if (subscriptionResponse?.data?.success) {
                window.dispatchEvent(new CustomEvent('subscription-updated', {
                  detail: subscriptionResponse.data.data
                }))
              }
            } catch (refreshError) {
              if (process.env.NODE_ENV === 'development') {
                console.error('刷新订阅信息失败:', refreshError)
              }
            }
          }
          Promise.all([refreshUserInfo(), refreshSubscription()]).then(() => {
            setTimeout(async () => {
              await Promise.all([refreshUserInfo(), refreshSubscription()])
            }, 500)
          })
          setTimeout(() => {
            successDialogVisible.value = false
            loadPackages()
            Promise.all([refreshUserInfo(), refreshSubscription()])
            if (router.currentRoute.value.path === '/subscription') {
              router.go(0)
            }
            if (router.currentRoute.value.path === '/dashboard') {
              router.go(0)
            }
          }, 3000)
          return
        } else if (orderData.status === 'cancelled') {
          if (paymentStatusCheckInterval) {
            clearInterval(paymentStatusCheckInterval)
            paymentStatusCheckInterval = null
          }
          if (visibilityChangeHandler) {
            document.removeEventListener('visibilitychange', visibilityChangeHandler)
            visibilityChangeHandler = null
          }
          if (focusHandler) {
            window.removeEventListener('focus', focusHandler)
            focusHandler = null
          }
          paymentQRVisible.value = false
          ElMessage.info('订单已取消')
        }
      } catch (error) {
        if (process.env.NODE_ENV === 'development') {
          console.error('检查支付状态出错:', {
            error: error,
            message: error.message,
            response: error.response?.data,
            order_no: currentOrder.value?.order_no
          })
        }
        // 超时错误已在上面的 catch 中处理
      } finally {
        isCheckingPayment.value = false
      }
    }
    const onImageLoad = () => {
    }
    const onImageError = async (event) => {
      if (paymentQRCode.value && paymentQRCode.value.startsWith('data:')) {
        ElMessage.warning('二维码显示异常，正在重新生成...')
        const paymentUrl = orderInfo.paymentUrl || currentOrder.value?.payment_url
        if (paymentUrl) {
          try {
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
          } catch (error) {
            ElMessage.error('二维码生成失败，请刷新页面重试')
          }
        } else {
          ElMessage.error('无法获取支付链接，请刷新页面重试')
        }
      } else {
        ElMessage.error('二维码加载失败，请刷新页面重试')
      }
    }
    const goToSubscription = () => {
      successDialogVisible.value = false
      router.push('/subscription')
    }
    const onPaymentSuccess = () => {
    }
    const onPaymentCancel = () => {
    }
    const onPaymentError = (error) => {
    }
    const handleSubscriptionUpdate = async (event) => {
      await loadUserBalance()
    }
    const handleUserInfoUpdate = async () => {
      await loadUserBalance()
    }
    
    // --- 核心优化：解除 onMounted 的阻塞，改用并发 ---
    onMounted(() => {
      // 不再使用 await 阻塞组件的生命周期
      loadUserBalance()
      loadPackages()
      
      if (typeof window !== 'undefined') {
        windowWidth.value = window.innerWidth
        window.addEventListener('resize', handleResize)
      }
      window.addEventListener('subscription-updated', handleSubscriptionUpdate)
      window.addEventListener('user-info-updated', handleUserInfoUpdate)
    })
    
    onUnmounted(() => {
      if (paymentStatusCheckInterval) {
        clearInterval(paymentStatusCheckInterval)
        paymentStatusCheckInterval = null
      }
      if (typeof window !== 'undefined') {
        window.removeEventListener('resize', handleResize)
      }
      window.removeEventListener('subscription-updated', handleSubscriptionUpdate)
      window.removeEventListener('user-info-updated', handleUserInfoUpdate)
    })

    // 自定义套餐相关方法
    const openCustomPackageDialog = () => {
      customPackageDialogVisible.value = true
      customPackageForm.devices = customPackageConfig.min_devices
      customPackageForm.months = customPackageConfig.min_months
      customPackageForm.couponCode = ''
      customCouponInfo.value = null
      calculateCustomPrice()
    }

    const calculateCustomPrice = () => {
      // 计算基础价格
      const basePrice = customPackageConfig.price_per_device_year *
                       customPackageForm.devices *
                       (customPackageForm.months / 12)
      customPackagePrice.basePrice = Math.round(basePrice * 100) / 100

      // 计算时长折扣
      let discountPercent = 0
      if (customPackageConfig.duration_discounts && Array.isArray(customPackageConfig.duration_discounts)) {
        for (const tier of customPackageConfig.duration_discounts) {
          if (customPackageForm.months >= tier.months && tier.discount > discountPercent) {
            discountPercent = tier.discount
          }
        }
      }
      customPackagePrice.discountPercent = discountPercent

      // 应用时长折扣
      let priceAfterDiscount = customPackagePrice.basePrice * (1 - discountPercent / 100)
      priceAfterDiscount = Math.round(priceAfterDiscount * 100) / 100

      // 应用优惠券折扣
      let couponDiscount = 0
      if (customCouponInfo.value && customCouponInfo.value.valid && customCouponInfo.value.discount_amount) {
        couponDiscount = customCouponInfo.value.discount_amount
      }
      customPackagePrice.couponDiscount = couponDiscount

      // 计算最终价格
      let finalPrice = priceAfterDiscount - couponDiscount
      if (finalPrice < 0) finalPrice = 0
      customPackagePrice.finalPrice = Math.round(finalPrice * 100) / 100
    }

    const handleCustomCouponInput = () => {
      customCouponInfo.value = null
      calculateCustomPrice()
    }

    const validateCustomCoupon = async () => {
      if (!customPackageForm.couponCode) {
        return
      }

      validatingCustomCoupon.value = true
      try {
        const response = await couponAPI.validateCoupon({
          code: customPackageForm.couponCode,
          amount: customPackagePrice.basePrice * (1 - customPackagePrice.discountPercent / 100)
        })

        if (response.data && response.data.success) {
          const data = response.data.data
          customCouponInfo.value = {
            valid: data.valid,
            message: data.message,
            discount_amount: data.discount_amount || 0
          }
          calculateCustomPrice()
        }
      } catch (error) {
        customCouponInfo.value = {
          valid: false,
          message: error.response?.data?.message || '优惠券验证失败'
        }
      } finally {
        validatingCustomCoupon.value = false
      }
    }

    const confirmCustomPackage = async () => {
      if (isProcessing.value) return

      isProcessing.value = true
      try {
        const orderData = {
          devices: customPackageForm.devices,
          months: customPackageForm.months,
          coupon_code: customPackageForm.couponCode || undefined
        }

        const response = await orderAPI.createCustomOrder(orderData)

        if (response.data && response.data.success) {
          const order = response.data.data

          // 关闭自定义套餐对话框
          customPackageDialogVisible.value = false

          // 构建虚拟套餐对象用于显示
          selectedPackage.value = {
            id: 0,
            name: order.package_name || `自定义套餐 (${customPackageForm.devices}设备/${customPackageForm.months}月)`,
            price: order.final_amount || order.amount || 0,
            duration_days: customPackageForm.months * 30,
            device_limit: customPackageForm.devices,
            description: `自定义套餐：${customPackageForm.devices}个设备，${customPackageForm.months}个月`
          }

          // 设置数量为1
          selectedQuantity.value = 1

          // 设置当前订单信息
          currentOrder.value = order

          // 清空优惠券信息
          couponCode.value = ''
          couponInfo.value = null

          // 并发加载支付方式和余额
          await Promise.all([
            loadPaymentMethods(),
            loadUserBalance()
          ])

          // 设置默认支付方式
          const finalPrice = order.final_amount || order.amount || 0
          if (userBalance.value >= finalPrice) {
            paymentMethod.value = 'balance'
          } else {
            paymentMethod.value = availablePaymentMethods.value[0]?.key || 'alipay'
          }

          // 显示支付对话框
          purchaseDialogVisible.value = true

          ElMessage.success('订单创建成功，请选择支付方式')
        }
      } catch (error) {
        ElMessage.error(error.response?.data?.message || '创建订单失败')
      } finally {
        isProcessing.value = false
      }
    }

    return {
      packages,
      isLoading,
      errorMessage,
      isProcessing,
      purchaseDialogVisible,
      paymentQRVisible,
      successDialogVisible,
      paymentQRCode,
      paymentUrl,
      isPaymentPageUrl,
      currentOrder,
      isCheckingPayment,
      showPaymentQRCode,
      checkPaymentStatus,
      openAlipayApp,
      onImageLoad,
      onImageError,
      onIframeLoad,
      selectedPackage,
      orderInfo,
      loadPackages,
      selectPackage,
      confirmPurchase,
      onPaymentSuccess,
      onPaymentCancel,
      onPaymentError,
      goToSubscription,
      couponCode,
      validatingCoupon,
      couponInfo,
      finalAmount,
      handleCouponInput,
      handleCouponFocus,
      paymentMethod,
      availablePaymentMethods,
      loadPaymentMethods,
      userBalance,
      handlePaymentMethodChange,
      loadUserBalance,
      isMobile,
      validateCoupon,
      clearCoupon,
      getPaymentMethodDisplayName,
      userLevel,
      levelDiscountRate,
      calculateLevelDiscount,
      selectedQuantity,
      packageType,
      durationOptions,
      durationPlaceholder,
      // 自定义套餐
      customPackageEnabled,
      customPackageConfig,
      customPackageDialogVisible,
      customPackageForm,
      customPackagePrice,
      validatingCustomCoupon,
      customCouponInfo,
      openCustomPackageDialog,
      calculateCustomPrice,
      handleCustomCouponInput,
      validateCustomCoupon,
      confirmCustomPackage,
      durationHint,
      durationDisplayText,
      handleQuantityChange,
      totalOriginalPrice,
      totalDurationDays
    }
  }
}
</script>
<style scoped lang="scss">
@use '@/styles/list-common.scss';
.packages-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 30px;
  margin-top: 20px;
}
.package-card {
  position: relative;
  text-align: center;
  transition: all 0.3s ease;
  border: 2px solid transparent;
}
.package-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
}
.package-card.popular {
  border-color: #409EFF;
}
.package-card.recommended {
  border-color: #67C23A;
}
.package-header {
  position: relative;
  margin-bottom: 20px;
}
.package-name {
  margin: 0;
  color: #303133;
  font-size: 20px;
  font-weight: bold;
}
.popular-badge,
.recommended-badge {
  position: absolute;
  top: -10px;
  right: -10px;
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: bold;
  color: white;
}
.popular-badge {
  background: #409EFF;
}
.recommended-badge {
  background: #67C23A;
}
.package-price {
  margin-bottom: 30px;
}
.currency {
  font-size: 18px;
  color: #909399;
  vertical-align: top;
}
.amount {
  font-size: 36px;
  font-weight: bold;
  color: #409EFF;
  margin: 0 5px;
}
.period {
  font-size: 16px;
  color: #909399;
}
.package-features {
  margin-bottom: 30px;
  text-align: left;
}
.package-features :is(ul) {
  list-style: none;
  padding: 0;
  margin: 0;
}
.package-features :is(li) {
  padding: 8px 0;
  color: #606266;
  display: flex;
  align-items: center;
}
.package-features :is(li) :is(i) {
  color: #67C23A;
  margin-right: 10px;
  font-size: 16px;
}
.package-actions {
  margin-bottom: 20px;
}
.package-actions .el-button {
  cursor: pointer;
  position: relative;
  z-index: 1;
}
.package-actions .el-button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
.purchase-confirm-horizontal {
  display: flex;
  gap: 16px;
  padding: 10px 0;
  max-height: calc(80vh - 120px);
  overflow-y: auto;
}
.purchase-left {
  flex: 1;
  min-width: 0;
}
.purchase-right {
  flex: 1;
  min-width: 0;
}
.purchase-confirm {
  padding: 10px 0;
}
.package-summary :is(h4),
.duration-selection :is(h4),
.price-summary :is(h4),
.payment-section-title {
  margin-bottom: 8px;
  margin-top: 0;
  color: #303133;
  font-size: 14px;
  font-weight: 600;
}
.amount {
  color: #f56c6c;
  font-weight: bold;
}
.purchase-actions {
  text-align: center;
  margin-top: 12px;
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}
.purchase-actions .el-button {
  margin: 0;
  min-width: 100px;
}
.success-message {
  text-align: center;
  padding: 20px 0;
}
.success-icon {
  font-size: 48px;
  color: #67C23A;
  margin-bottom: 15px;
}
.success-message h3 {
  margin: 15px 0;
  color: #303133;
}
.success-message :is(p) {
  margin-bottom: 20px;
  color: #606266;
}
.success-actions {
  margin-top: 20px;
}
.success-actions .el-button {
  margin: 0 10px;
}
.package-description {
  margin: 15px 0;
  padding: 10px;
  background: #f8f9fa;
  border-radius: 6px;
  border-left: 3px solid #409EFF;
}
.package-description :is(p) {
  margin: 0;
  color: #606266;
  font-size: 14px;
  line-height: 1.5;
}
.purchase-dialog {
  :deep(.el-dialog) {
    margin: 3vh auto !important;
    max-height: 92vh;
    overflow-y: auto;
  }
  :deep(.el-dialog__body) {
    padding: 12px 20px !important;
    max-height: calc(92vh - 100px);
    overflow-y: auto;
  }
}
.level-discount-tip {
  margin-top: 0;
  padding: 10px;
  background: linear-gradient(135deg, #e8f5e9 0%, #c8e6c9 100%);
  border-radius: 4px;
  border-left: 3px solid #4caf50;
}
.level-discount-tip .tip-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}
.level-discount-tip .tip-icon {
  color: #4caf50;
  font-size: 16px;
  flex-shrink: 0;
}
.level-discount-tip .tip-title {
  font-weight: 600;
  color: #2e7d32;
  font-size: 13px;
  line-height: 1.4;
}
.level-discount-tip .level-name-highlight {
  font-weight: bold;
}
.level-discount-tip .tip-content {
  font-size: 12px;
  color: #388e3c;
  line-height: 1.5;
  margin-top: 4px;
}
.level-upgrade-tip {
  margin-top: 0;
  padding: 10px;
  background: linear-gradient(135deg, #fff3e0 0%, #ffe0b2 100%);
  border-radius: 4px;
  border-left: 3px solid #ff9800;
}
.level-upgrade-tip .tip-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}
.level-upgrade-tip .upgrade-icon {
  color: #ff9800;
  font-size: 16px;
  flex-shrink: 0;
}
.level-upgrade-tip .upgrade-title {
  font-weight: 600;
  color: #e65100;
  font-size: 13px;
  line-height: 1.4;
}
.level-upgrade-tip .upgrade-content {
  font-size: 12px;
  color: #f57c00;
  line-height: 1.5;
  margin-top: 4px;
}
.price-summary {
  margin-top: 0;
  padding: 0;
  background: transparent;
  border-radius: 4px;
}
.price-summary .discount-item {
  display: flex;
  align-items: center;
  gap: 8px;
}
.price-summary .discount-amount {
  color: #67c23a;
  font-weight: bold;
}
.price-summary .level-tag {
  flex-shrink: 0;
}
.price-summary .final-amount {
  font-size: 18px;
  color: #f56c6c;
  font-weight: bold;
}
.payment-method-section {
  margin-top: 0;
  padding: 0;
  background: transparent;
  border-radius: 4px;
  border: 1px solid #e4e7ed;
}
.payment-option {
  width: 100%;
  margin-bottom: 10px;
  padding: 10px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  transition: all 0.3s;
  &:hover {
    border-color: #409eff;
    background-color: #f0f9ff;
  }
  &:last-child {
    margin-bottom: 0;
  }
}
.payment-option-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  font-size: 14px;
  flex-wrap: wrap;
  gap: 4px;
}
.payment-option-label {
  display: flex;
  align-items: center;
  font-weight: 500;
}
.payment-icon {
  margin-right: 6px;
  font-size: 16px;
}
.payment-status {
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  &.success {
    color: #67c23a;
  }
  &.error {
    color: #f56c6c;
  }
  &.disabled {
    color: #909399;
  }
  &.info {
    color: #409eff;
  }
}
.payment-method-section .payment-section-title {
  margin-bottom: 15px;
  margin-top: 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
}
.balance-info {
  margin-bottom: 10px;
  padding: 8px;
  background: #f5f7fa;
  border-radius: 4px;
}
.balance-info .balance-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.balance-info .balance-label {
  font-weight: 600;
  color: #606266;
  font-size: 13px;
}
.balance-info .balance-amount {
  font-size: 16px;
  color: #409eff;
  font-weight: 700;
}
.coupon-input-group {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  flex-wrap: nowrap;
}
.coupon-input {
  flex: 1;
  min-width: 0;
}
.coupon-buttons {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}
@media (max-width: 768px) {
  .purchase-dialog {
    :deep(.el-dialog) {
      width: 95% !important;
      margin: 1vh auto !important;
      max-height: 98vh;
    }
    :deep(.el-dialog__header) {
      padding: 12px 12px 8px 12px;
    }
    :deep(.el-dialog__title) {
      font-size: 15px;
      font-weight: 600;
    }
    :deep(.el-dialog__body) {
      padding: 8px 12px 12px 12px !important;
      max-height: calc(98vh - 70px);
      overflow-y: auto;
    }
  }
  .purchase-confirm-horizontal {
    flex-direction: column;
    gap: 10px;
    padding: 0;
    max-height: calc(98vh - 100px);
  }
  .purchase-left,
  .purchase-right {
    width: 100%;
  }
  .package-summary :is(h4),
  .duration-selection :is(h4),
  .price-summary :is(h4),
  .payment-section-title {
    font-size: 14px;
    margin-bottom: 8px;
    font-weight: 600;
  }
  .purchase-confirm-horizontal :deep(.el-descriptions) {
    font-size: 12px;
  }
  .purchase-confirm-horizontal :deep(.el-descriptions__label) {
    width: 30% !important;
    font-size: 12px;
    padding: 8px 6px !important;
    font-weight: 500;
  }
  .purchase-confirm-horizontal :deep(.el-descriptions__content) {
    width: 70% !important;
    font-size: 12px;
    padding: 8px 6px !important;
  }
  .purchase-confirm-horizontal :deep(.el-descriptions__table) {
    width: 100%;
  }
  .coupon-section {
    padding: 10px !important;
    margin-top: 10px !important;
    :is(h4) {
      font-size: 13px !important;
      margin-bottom: 6px !important;
    }
  }
  .coupon-input-group {
    flex-direction: column;
    gap: 8px;
  }
  .coupon-input {
    width: 100%;
  }
  .coupon-buttons {
    width: 100%;
    display: flex;
    gap: 6px;
  }
  .coupon-buttons .el-button {
    flex: 1;
    min-height: 40px;
    font-size: 14px;
  }
  .duration-selection {
    margin-top: 10px !important;
    :deep(.el-select) {
      .el-input__wrapper {
        min-height: 40px;
      }
      .el-input__inner {
        font-size: 14px;
      }
    }
  }
  .form-hint {
    font-size: 11px !important;
    margin-top: 4px !important;
  }
  .payment-method-section {
    border: none;
    padding: 0;
    margin-top: 10px !important;
    :deep(.el-radio-group) {
      width: 100%;
    }
  }
  .balance-info {
    margin-bottom: 8px;
    .balance-row {
      font-size: 13px;
    }
  }
  .payment-option {
    padding: 10px;
    margin-bottom: 8px;
    min-height: 48px;
    border: 1.5px solid #e4e7ed;
    :deep(.el-radio__label) {
      font-size: 13px;
    }
    &:active {
      background-color: #f0f9ff;
      border-color: #409eff;
    }
  }
  .payment-option-content {
    font-size: 13px;
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
  .payment-option-label {
    font-size: 13px;
    font-weight: 600;
    width: 100%;
  }
  .payment-icon {
    font-size: 16px;
    margin-right: 6px;
  }
  .payment-status {
    font-size: 12px;
    width: 100%;
    text-align: left;
    line-height: 1.3;
  }
  .balance-info {
    padding: 8px;
    margin-bottom: 8px;
    .balance-label {
      font-size: 13px;
    }
    .balance-amount {
      font-size: 16px;
    }
  }
  .level-discount-tip,
  .level-upgrade-tip {
    padding: 10px;
    margin-top: 10px !important;
    .tip-title {
      font-size: 12px;
      line-height: 1.4;
    }
    .tip-content {
      font-size: 11px;
      line-height: 1.4;
      margin-top: 4px;
    }
    .tip-icon {
      font-size: 14px;
    }
  }
  .purchase-actions {
    display: flex;
    flex-direction: row;
    gap: 8px;
    margin-top: 10px !important;
    padding-top: 10px !important;
  }
  .purchase-actions .el-button {
    flex: 1;
    min-height: 44px;
    font-size: 15px;
    font-weight: 600;
    margin: 0 !important;
  }
  .price-summary {
    .final-amount {
      font-size: 18px;
    }
    .discount-amount {
      font-size: 13px;
    }
  }
  // 优化 Alert 组件
  :deep(.el-alert) {
    padding: 8px 10px !important;
    font-size: 12px !important;
    .el-alert__title {
      font-size: 12px !important;
    }
    .el-alert__icon {
      font-size: 14px !important;
    }
  }
  .packages-grid {
    grid-template-columns: 1fr;
    gap: 16px;
    padding: 12px;
    margin-top: 12px;
  }
  .package-card {
    margin: 0;
    border-radius: 12px;
    :deep(.el-card__body) {
      padding: 16px;
    }
    .package-header {
      margin-bottom: 16px;
      .package-name {
        font-size: 18px;
        font-weight: 600;
      }
    }
    .package-price {
      margin-bottom: 16px;
      .amount {
        font-size: 28px;
      }
      .currency {
        font-size: 18px;
      }
      .period {
        font-size: 14px;
      }
    }
    .package-features {
      margin-bottom: 16px;
      text-align: left;
      ul {
        padding-left: 20px;
        li {
          font-size: 14px;
          line-height: 1.8;
          margin-bottom: 6px;
        }
      }
    }
    .package-actions {
      .el-button {
        min-height: 48px;
        font-size: 16px;
        font-weight: 600;
      }
    }
  }
  .popular-badge,
  .recommended-badge {
    font-size: 12px;
    padding: 4px 10px;
  }
  .purchase-confirm :deep(.el-descriptions) {
    font-size: 13px;
  }
  .purchase-confirm :deep(.el-descriptions__label) {
    width: 35% !important;
  }
  .purchase-confirm :deep(.el-descriptions__content) {
    width: 65% !important;
  }
  .packages-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  .package-card {
    margin: 0;
    border-radius: 12px;
    :deep(.el-card__body) {
      padding: 20px 16px;
    }
    .package-header {
      flex-direction: column;
      align-items: flex-start;
      gap: 12px;
      margin-bottom: 16px;
      .package-name {
        font-size: 1.25rem;
        margin: 0;
      }
      .popular-badge,
      .recommended-badge {
        font-size: 0.75rem;
        padding: 4px 10px;
      }
    }
    .package-price {
      margin-bottom: 20px;
      .currency {
        font-size: 1.25rem;
      }
      .amount {
        font-size: 2rem;
      }
      .period {
        font-size: 1rem;
      }
    }
    .package-features {
      margin-bottom: 20px;
      :is(ul) {
        :is(li) {
          padding: 8px 0;
          font-size: 0.875rem;
          :is(i) {
            font-size: 14px;
            margin-right: 8px;
          }
        }
      }
    }
    .package-description {
      margin-bottom: 20px;
      :is(p) {
        font-size: 0.875rem;
        line-height: 1.6;
      }
    }
    .package-button {
      width: 100%;
      padding: 14px;
      font-size: 1rem;
    }
  }
}
@media (max-width: 480px) {
  .package-card {
    .package-price {
      .amount {
        font-size: 1.75rem;
      }
    }
  }
}
:deep(.el-input__wrapper) {
  border-radius: 0 !important;
  box-shadow: none !important;
  border: 1px solid #dcdfe6 !important;
  background-color: #ffffff !important;
  pointer-events: auto !important;
}
:deep(.el-input__inner) {
  border-radius: 0 !important;
  border: none !important;
  box-shadow: none !important;
  background-color: transparent !important;
  pointer-events: auto !important;
}
:deep(.el-input__wrapper:hover) {
  border-color: #c0c4cc !important;
  box-shadow: none !important;
  background-color: #ffffff !important;
}
:deep(.el-input__wrapper.is-focus) {
  border-color: #1677ff !important;
  box-shadow: none !important;
  background-color: #ffffff !important;
}
:deep(.el-input__wrapper.is-focus:hover) {
  background-color: #ffffff !important;
}
:deep(.el-input__wrapper > *) {
  background-color: transparent !important;
  background: transparent !important;
}
:deep(.el-textarea__inner) {
  border-radius: 0 !important;
  border: 1px solid #dcdfe6 !important;
  box-shadow: none !important;
  background-color: #ffffff !important;
}
:deep(.el-textarea__inner:hover) {
  border-color: #c0c4cc !important;
}
:deep(.el-textarea__inner:focus) {
  border-color: #1677ff !important;
  box-shadow: none !important;
}
.coupon-section {
  position: relative;
  z-index: 1;
}
.coupon-section :deep(.el-input) {
  pointer-events: auto !important;
  position: relative;
  z-index: 10 !important;
}
.coupon-section :deep(.el-input__wrapper) {
  pointer-events: auto !important;
  cursor: text !important;
  position: relative;
  z-index: 10 !important;
}
.coupon-section :deep(.el-input__inner) {
  pointer-events: auto !important;
  cursor: text !important;
  position: relative;
  z-index: 10 !important;
  -webkit-user-select: auto !important;
  user-select: auto !important;
  -webkit-tap-highlight-color: transparent !important;
}
.coupon-section :deep(.el-input:not(.is-disabled)) {
  pointer-events: auto !important;
}
.coupon-section :deep(.el-input:not(.is-disabled) .el-input__wrapper) {
  pointer-events: auto !important;
  cursor: text !important;
}
.coupon-section :deep(.el-input:not(.is-disabled) .el-input__inner) {
  pointer-events: auto !important;
  cursor: text !important;
  -webkit-user-select: auto !important;
  user-select: auto !important;
}
.coupon-section :deep(.el-input.is-disabled) {
  pointer-events: none !important;
}
.coupon-section :deep(.el-input.is-disabled .el-input__wrapper) {
  pointer-events: none !important;
  cursor: not-allowed !important;
}
.coupon-section :deep(.el-input.is-disabled .el-input__inner) {
  pointer-events: none !important;
  cursor: not-allowed !important;
}
.purchase-confirm .coupon-section {
  position: relative;
  z-index: 1;
}
.purchase-confirm .coupon-section .el-input {
  position: relative;
  z-index: 2;
}
.coupon-input {
  pointer-events: auto !important;
}
.coupon-input :deep(*) {
  pointer-events: auto !important;
}
.coupon-input :deep(.el-input__wrapper) {
  pointer-events: auto !important;
}
.coupon-input :deep(.el-input__inner) {
  pointer-events: auto !important;
}
.payment-qr-dialog {
  :deep(.el-dialog) {
    border-radius: 24px;
    overflow: hidden;
    background: linear-gradient(180deg, #f8fbff 0%, #ffffff 100%);
    box-shadow: 0 24px 80px rgba(15, 23, 42, 0.22);
  }
  :deep(.el-dialog__header) {
    padding: 22px 24px 12px;
    border-bottom: 1px solid rgba(37, 99, 235, 0.08);
  }
  :deep(.el-dialog__title) {
    font-size: 24px;
    font-weight: 700;
    color: #0f172a;
    letter-spacing: 0.02em;
  }
  :deep(.el-dialog__headerbtn) {
    top: 22px;
    right: 20px;
  }
  :deep(.el-dialog__body) {
    padding: 0 24px 24px;
  }
}

.payment-qr-container {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.payment-summary-card {
  padding: 18px 20px;
  border-radius: 20px;
  background: linear-gradient(135deg, #eff6ff 0%, #f8fafc 100%);
  border: 1px solid rgba(59, 130, 246, 0.14);
}

.summary-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.summary-label {
  font-size: 13px;
  color: #64748b;
  margin-bottom: 6px;
}

.summary-amount {
  font-size: 32px;
  line-height: 1;
  font-weight: 800;
  color: #111827;
}

.summary-badge {
  flex-shrink: 0;
  padding: 8px 12px;
  border-radius: 999px;
  background: rgba(37, 99, 235, 0.1);
  color: #1d4ed8;
  font-size: 13px;
  font-weight: 600;
  max-width: 220px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.summary-meta {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.meta-item {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  font-size: 14px;
}

.meta-key {
  color: #64748b;
  flex-shrink: 0;
}

.meta-value {
  color: #0f172a;
  font-weight: 500;
  text-align: right;
  word-break: break-all;
}

.qr-panel {
  padding: 20px;
  border-radius: 24px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.8);
}

.qr-panel-header {
  text-align: center;
  margin-bottom: 18px;
}

.qr-panel-header h4 {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
}

.qr-panel-header p {
  margin: 8px 0 0;
  font-size: 13px;
  color: #64748b;
}

.qr-code-wrapper {
  display: flex;
  justify-content: center;
}

.qr-code-wrapper.iframe-mode {
  display: block;
}

.qr-code,
.qr-loading {
  width: 280px;
  min-height: 280px;
  border-radius: 24px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
  border: 1px solid #dbeafe;
  box-shadow: 0 16px 40px rgba(37, 99, 235, 0.12);
  display: flex;
  align-items: center;
  justify-content: center;
}

.qr-code img {
  width: 232px;
  height: 232px;
  display: block;
  border-radius: 16px;
  background: #fff;
}

.qr-loading {
  flex-direction: column;
  gap: 12px;
  color: #64748b;
}

.payment-page-iframe {
  width: 100%;
  min-height: 600px;
  border: 1px solid #dbeafe;
  border-radius: 20px;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 16px 40px rgba(37, 99, 235, 0.12);
}

.payment-page-iframe iframe {
  width: 100%;
  min-height: 600px;
  border: none;
  display: block;
}

.payment-tips {
  margin-top: 16px;
}

.tip-text {
  margin: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #334155;
  font-size: 14px;
  font-weight: 500;
}

.tip-text :deep(svg) {
  color: #2563eb;
}

.payment-actions-container {
  margin-top: 18px;
}

.alipay-btn {
  height: 46px;
  border-radius: 14px;
  font-weight: 600;
}

.btn-icon {
  margin-right: 6px;
}

@media (max-width: 768px) {
  .payment-qr-dialog {
    :deep(.el-dialog) {
      width: 92% !important;
      margin: 5vh auto !important;
      border-radius: 20px;
    }
    :deep(.el-dialog__header) {
      padding: 18px 18px 10px;
    }
    :deep(.el-dialog__title) {
      font-size: 20px;
    }
    :deep(.el-dialog__body) {
      padding: 0 18px 18px;
    }
  }

  .payment-summary-card,
  .qr-panel {
    padding: 16px;
    border-radius: 18px;
  }

  .summary-header,
  .meta-item {
    flex-direction: column;
    align-items: flex-start;
  }

  .summary-badge {
    max-width: 100%;
  }

  .meta-value {
    text-align: left;
  }

  .summary-amount {
    font-size: 28px;
  }

  .qr-code,
  .qr-loading {
    width: 100%;
    min-height: 252px;
  }

  .qr-code img {
    width: min(220px, calc(100% - 32px));
    height: min(220px, calc(100% - 32px));
  }

  .payment-page-iframe,
  .payment-page-iframe iframe {
    min-height: 420px;
  }
}

/* 自定义套餐卡片样式 */
.packages-container {
  padding: 20px;
}

.custom-package-card {
  border: 2px solid #67c23a !important;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  position: relative;
}

.custom-badge {
  position: absolute;
  top: 10px;
  right: 10px;
  background: linear-gradient(135deg, #67c23a 0%, #85ce61 100%);
  color: white;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: bold;
  box-shadow: 0 2px 8px rgba(103, 194, 58, 0.3);
  z-index: 1;
}

.custom-package-form {
  .form-hint {
    font-size: 12px;
    color: #909399;
    margin-top: 5px;
    line-height: 1.5;
  }

  .coupon-input-group {
    display: flex;
    gap: 10px;
    align-items: flex-start;

    .el-input {
      flex: 1;
    }

    .el-button {
      flex-shrink: 0;
    }
  }

  .price-summary-custom {
    margin-top: 20px;

    .el-descriptions {
      :deep(.el-descriptions__label) {
        font-weight: 600;
        background-color: #f5f7fa;
      }

      :deep(.el-descriptions__content) {
        font-weight: 500;
      }
    }
  }

  .discount-amount {
    color: #f56c6c;
    font-weight: bold;
  }

  .final-amount {
    color: #409eff;
    font-size: 20px;
    font-weight: bold;
  }

  .mobile-form-label {
    font-size: 15px;
    font-weight: 600;
    color: #303133;
    margin-bottom: 8px;
    line-height: 1.5;
  }
}

.custom-package-dialog {
  :deep(.el-dialog__body) {
    padding: 20px;
  }

  :deep(.el-form-item__label) {
    font-weight: 600;
    color: #303133;
  }

  :deep(.el-input-number) {
    width: 100%;

    .el-input__wrapper {
      width: 100%;
    }
  }
}

@media (max-width: 768px) {
  .packages-container {
    padding: 12px;
  }

  .packages-grid {
    gap: 16px;
    margin-top: 12px;
  }

  .custom-package-card {
    .package-header {
      padding-right: 100px;
      margin-bottom: 16px;

      .package-name {
        font-size: 18px;
        font-weight: 600;
        margin: 0;
      }
    }

    .custom-badge {
      top: 15px;
      right: 15px;
      padding: 6px 14px;
      font-size: 13px;
    }

    .package-price {
      margin: 15px 0;

      .currency {
        font-size: 18px;
      }

      .amount {
        font-size: 32px;
      }

      .period {
        font-size: 14px;
      }
    }

    .package-description {
      margin: 15px 0;

      p {
        font-size: 14px;
        line-height: 1.6;
        color: #606266;
      }
    }

    .package-features {
      margin: 15px 0;

      ul {
        padding-left: 0;
        list-style: none;

        li {
          font-size: 14px;
          padding: 8px 0;
          display: flex;
          align-items: center;
          gap: 8px;

          i {
            color: #67c23a;
            font-size: 16px;
          }
        }
      }
    }

    .package-actions {
      margin-top: 20px;

      .el-button {
        height: 48px;
        font-size: 16px;
        font-weight: 600;
        border-radius: 8px;
      }
    }
  }

  .custom-package-dialog {
    :deep(.el-dialog) {
      width: 95% !important;
      margin: 2vh auto !important;
      max-height: 96vh;
      border-radius: 12px;
    }

    :deep(.el-dialog__header) {
      padding: 16px 20px 12px;
      border-bottom: 1px solid #ebeef5;

      .el-dialog__title {
        font-size: 18px;
        font-weight: 600;
      }

      .el-dialog__headerbtn {
        top: 12px;
        right: 12px;
        width: 36px;
        height: 36px;

        .el-dialog__close {
          font-size: 20px;
        }
      }
    }

    :deep(.el-dialog__body) {
      padding: 16px 20px !important;
      max-height: calc(96vh - 140px);
      overflow-y: auto;
      -webkit-overflow-scrolling: touch;
    }

    :deep(.el-dialog__footer) {
      padding: 12px 20px 16px;
      border-top: 1px solid #ebeef5;

      .dialog-footer {
        display: flex;
        gap: 10px;

        .el-button {
          flex: 1;
          height: 44px;
          font-size: 16px;
          font-weight: 500;
          border-radius: 8px;
        }
      }
    }

    :deep(.el-form) {
      .el-form-item {
        margin-bottom: 20px;

        .el-form-item__label {
          display: block;
          text-align: left;
          padding: 0 0 8px 0;
          font-size: 15px;
          font-weight: 600;
          color: #303133;
          line-height: 1.5;
        }

        .el-form-item__content {
          margin-left: 0 !important;
        }
      }

      .el-input-number {
        width: 100%;

        .el-input__wrapper {
          padding: 12px 15px;
          font-size: 16px;
          min-height: 48px;
        }

        .el-input__inner {
          font-size: 16px !important;
          text-align: center;
        }

        .el-input-number__decrease,
        .el-input-number__increase {
          width: 40px;
          height: 48px;
          font-size: 18px;
          display: flex;
          align-items: center;
          justify-content: center;

          .el-icon {
            font-size: 18px;
          }
        }
      }

      .el-input {
        .el-input__wrapper {
          padding: 12px 15px;
          font-size: 16px;
          min-height: 48px;
        }

        .el-input__inner {
          font-size: 16px !important;
        }
      }
    }

    .form-hint {
      font-size: 13px;
      color: #909399;
      margin-top: 6px;
      line-height: 1.6;
    }

    .coupon-input-group {
      flex-direction: column;
      gap: 10px;

      .el-input {
        width: 100%;
      }

      .el-button {
        width: 100%;
        height: 44px;
        font-size: 16px;
      }
    }

    .price-summary-custom {
      margin-top: 20px;
      background: #f5f7fa;
      padding: 15px;
      border-radius: 8px;

      .el-descriptions {
        :deep(.el-descriptions__table) {
          .el-descriptions__label,
          .el-descriptions__content {
            font-size: 14px;
            padding: 10px 12px;
          }

          .el-descriptions__label {
            font-weight: 600;
            background-color: #fff;
          }

          .el-descriptions__content {
            font-weight: 500;
          }
        }
      }

      .discount-amount {
        font-size: 15px;
      }

      .final-amount {
        font-size: 22px;
      }
    }

    :deep(.el-divider) {
      margin: 20px 0;
    }

    :deep(.el-alert) {
      padding: 10px 12px;
      font-size: 13px;

      .el-alert__title {
        font-size: 13px;
        line-height: 1.5;
      }
    }
  }
}
</style>