<template>
  <div class="admin-payment-config">
    <el-card>
      <template #header>
        <div class="header-content">
          <span>支付配置管理</span>
          <div class="header-actions desktop-only">
            <el-button type="warning" @click="openBulkDialog">
              <el-icon><Operation /></el-icon>
              批量操作
            </el-button>
            <el-button type="primary" @click="openAddDialog">
              <el-icon><Plus /></el-icon>
              添加支付配置
            </el-button>
          </div>
          <div class="header-actions mobile-only">
            <el-button type="primary" @click="openAddDialog" size="small">
              <el-icon><Plus /></el-icon>
              添加
            </el-button>
            <el-dropdown @command="handleMobileAction" trigger="click">
              <el-button type="default" size="small">
                <el-icon><Operation /></el-icon>
                更多
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="bulk">
                    <el-icon><Operation /></el-icon>
                    批量操作
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </template>
      
      <div class="batch-actions" v-if="selectedConfigs.length > 0">
        <div class="batch-info">
          <span>已选择 {{ selectedConfigs.length }} 个配置</span>
        </div>
        <div class="batch-buttons">
          <el-button type="success" @click="handleBatchAction('enable')" :loading="batchOperating">
            <el-icon><Check /></el-icon>
            批量启用
          </el-button>
          <el-button type="warning" @click="handleBatchAction('disable')" :loading="batchOperating">
            <el-icon><Close /></el-icon>
            批量禁用
          </el-button>
          <el-button type="danger" @click="handleBatchAction('delete')" :loading="batchOperating">
            <el-icon><Delete /></el-icon>
            批量删除
          </el-button>
          <el-button @click="clearSelection">
            <el-icon><Close /></el-icon>
            取消选择
          </el-button>
        </div>
      </div>

      <div class="table-wrapper desktop-only">
        <el-table 
          ref="tableRef"
          :data="paymentConfigs" 
          style="width: 100%" 
          v-loading="loading" 
          :empty-text="paymentConfigs.length === 0 ? '暂无支付配置' : '暂无数据'"
          @selection-change="handleSelectionChange"
        >
          <el-table-column type="selection" width="50" />
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="pay_type" label="支付类型" width="120">
            <template #default="scope">
              <el-tag :type="getPaymentTypeConfig(scope.row.pay_type).tag">
                {{ getPaymentTypeConfig(scope.row.pay_type).label }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="app_id" label="应用ID/商户ID" min-width="200">
            <template #default="scope">
              <span v-if="scope.row.app_id">{{ scope.row.app_id }}</span>
              <span v-else-if="scope.row.config_json?.yipay_pid">
                {{ scope.row.config_json.yipay_pid }} ({{ getPaymentTypeConfig(scope.row.pay_type).label }})
              </span>
              <span v-else class="text-muted">未配置</span>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="120" align="center">
            <template #default="scope">
              <el-switch
                v-model="scope.row.status"
                :active-value="1"
                :inactive-value="0"
                @change="(val) => toggleStatus(scope.row, val)"
              />
              <span class="status-text">
                {{ scope.row.status === 1 ? '已启用' : '已禁用' }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="180" />
          <el-table-column label="操作" width="180" align="center">
            <template #default="scope">
              <el-button size="small" type="primary" @click="editConfig(scope.row)">
                编辑
              </el-button>
              <el-button size="small" type="danger" @click="deleteConfig(scope.row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="mobile-card-list mobile-only" v-if="paymentConfigs.length > 0">
        <div v-for="config in paymentConfigs" :key="config.id" class="mobile-card">
          <div class="card-row">
            <span class="label">ID</span>
            <span class="value">#{{ config.id }}</span>
          </div>
          <div class="card-row">
            <span class="label">支付类型</span>
            <span class="value">
              <el-tag :type="getPaymentTypeConfig(config.pay_type).tag">
                {{ getPaymentTypeConfig(config.pay_type).label }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">应用ID</span>
            <span class="value">
              <span v-if="config.app_id">{{ config.app_id }}</span>
              <span v-else-if="config.config_json?.yipay_pid">
                {{ config.config_json.yipay_pid }}
              </span>
              <span v-else class="text-muted">未配置</span>
            </span>
          </div>
          <div class="card-row">
            <span class="label">状态</span>
            <span class="value">
              <el-switch
                v-model="config.status"
                :active-value="1"
                :inactive-value="0"
                @change="(val) => toggleStatus(config, val)"
              />
              <span class="status-text">
                {{ config.status === 1 ? '已启用' : '已禁用' }}
              </span>
            </span>
          </div>
          <div class="card-row">
            <span class="label">创建时间</span>
            <span class="value">{{ config.created_at || '-' }}</span>
          </div>
          <div class="card-actions">
            <el-button size="small" type="primary" @click="editConfig(config)">
              <el-icon><Edit /></el-icon> 编辑
            </el-button>
            <el-button size="small" type="danger" @click="deleteConfig(config)">
              <el-icon><Delete /></el-icon> 删除
            </el-button>
          </div>
        </div>
      </div>

      <div class="mobile-card-list mobile-only" v-if="paymentConfigs.length === 0 && !loading">
        <div class="empty-state">
          <el-empty description="暂无支付配置，请点击右上角【添加】按钮添加" :image-size="80" />
        </div>
      </div>
    </el-card>

    <el-dialog
      v-model="showAddDialog"
      :title="editingConfig ? '编辑支付配置' : '添加支付配置'"
      :width="isMobile ? '95%' : '600px'"
      :class="isMobile ? 'mobile-dialog' : ''"
    >
      <el-form :model="configForm" :label-width="isMobile ? '0' : '120px'" :label-position="isMobile ? 'top' : 'right'">
        <el-form-item label="支付类型">
          <template v-if="isMobile"><div class="mobile-label">支付类型</div></template>
          <el-select 
            v-model="configForm.pay_type" 
            placeholder="选择支付类型"
            style="width: 100%"
            :teleported="isMobile"
            :popper-class="isMobile ? 'mobile-select-popper' : ''"
          >
            <el-option-group label="官方支付">
              <el-option label="支付宝" value="alipay" />
              <el-option label="微信支付" value="wechat" />
            </el-option-group>
            <el-option-group label="第三方支付网关">
              <el-option label="易支付（统一配置，推荐）" value="yipay" />
              <el-option label="易支付-支付宝（兼容旧配置）" value="yipay_alipay" />
              <el-option label="易支付-微信（兼容旧配置）" value="yipay_wxpay" />
              <el-option label="易支付-QQ钱包（兼容旧配置）" value="yipay_qqpay" />
            </el-option-group>
          </el-select>
        </el-form-item>

        <template v-if="['alipay', 'wechat'].includes(configForm.pay_type)">
          <el-form-item label="应用ID">
            <template v-if="isMobile"><div class="mobile-label">应用ID</div></template>
            <el-input v-model="configForm.app_id" placeholder="请输入应用ID" style="width: 100%" />
            <div class="form-tip" v-if="configForm.pay_type === 'alipay'">
              <div>请确保已签约“当面付”产品并应用已上线。</div>
              <div>回调地址需同时在支付宝后台的应用网关中配置。</div>
            </div>
          </el-form-item>
        </template>

        <template v-if="configForm.pay_type.startsWith('yipay')">
          <el-form-item label="商户ID">
            <template v-if="isMobile"><div class="mobile-label">商户ID</div></template>
            <el-input v-model="configForm.app_id" placeholder="请输入易支付商户ID (pid)" style="width: 100%" />
          </el-form-item>
        </template>

        <template v-if="['yipay', 'yipay_alipay', 'yipay_wxpay', 'yipay_qqpay'].includes(configForm.pay_type)">
           <el-form-item label="商户密钥">
             <template v-if="isMobile"><div class="mobile-label">商户密钥</div></template>
             <el-input 
               v-model="configForm.merchant_private_key" 
               type="password"
               show-password
               placeholder="请输入易支付商户密钥 (key)" 
               style="width: 100%"
             />
           </el-form-item>
        </template>

        <template v-if="configForm.pay_type === 'yipay'">
          <el-form-item label="网关地址">
            <template v-if="isMobile"><div class="mobile-label">网关地址</div></template>
            <el-input v-model="configForm.yipay_gateway_url" placeholder="请输入易支付网关地址" style="width: 100%" />
            <div class="form-tip">填写官网地址，系统自动拼接 API 路径。</div>
          </el-form-item>
          
          <el-form-item label="签名方式">
            <template v-if="isMobile"><div class="mobile-label">签名方式</div></template>
            <el-select v-model="configForm.yipay_sign_type" style="width: 100%" :teleported="!isMobile">
              <el-option label="MD5签名" value="MD5" />
              <el-option label="RSA签名" value="RSA" />
              <el-option label="MD5+RSA签名" value="MD5+RSA" />
            </el-select>
          </el-form-item>

          <template v-if="['RSA', 'MD5+RSA'].includes(configForm.yipay_sign_type)">
            <el-form-item label="平台公钥">
              <template v-if="isMobile"><div class="mobile-label">平台公钥</div></template>
              <el-input v-model="configForm.yipay_platform_public_key" type="textarea" :rows="4" placeholder="易支付平台提供的公钥" />
            </el-form-item>
            <el-form-item label="商户私钥">
              <template v-if="isMobile"><div class="mobile-label">商户私钥</div></template>
              <el-input v-model="configForm.yipay_merchant_private_key" type="textarea" :rows="4" placeholder="您生成的商户RSA私钥" />
            </el-form-item>
          </template>

          <el-form-item label="支持支付方式">
            <template v-if="isMobile"><div class="mobile-label">支持支付方式</div></template>
            <el-checkbox-group v-model="configForm.yipay_supported_types">
              <el-checkbox label="alipay">支付宝</el-checkbox>
              <el-checkbox label="wxpay">微信支付</el-checkbox>
              <el-checkbox label="qqpay">QQ钱包</el-checkbox>
            </el-checkbox-group>
          </el-form-item>
        </template>

        <template v-if="['yipay_alipay', 'yipay_wxpay'].includes(configForm.pay_type)">
          <el-alert title="建议使用“易支付（统一配置）”以获得更好的兼容性" type="warning" :closable="false" style="margin-bottom: 20px;" />
          
          <el-form-item label="商户ID">
            <el-input v-model="configForm.yipay_pid" placeholder="请输入易支付商户ID" />
          </el-form-item>
          
          <el-form-item label="签名类型">
             <el-select v-model="configForm.yipay_sign_type" style="width: 100%" :teleported="!isMobile">
               <el-option label="RSA签名" value="RSA" />
               <el-option label="MD5签名" value="MD5" />
             </el-select>
          </el-form-item>

          <template v-if="configForm.yipay_sign_type === 'RSA'">
            <el-form-item label="商户私钥">
              <el-input v-model="configForm.yipay_private_key" type="textarea" :rows="4" placeholder="易支付商户私钥" />
            </el-form-item>
            <el-form-item label="平台公钥">
              <el-input v-model="configForm.yipay_public_key" type="textarea" :rows="4" placeholder="易支付平台公钥" />
            </el-form-item>
          </template>

          <template v-if="configForm.yipay_sign_type === 'MD5'">
            <el-form-item label="MD5密钥">
              <el-input v-model="configForm.yipay_md5_key" type="password" show-password />
            </el-form-item>
          </template>

          <el-form-item label="网关地址">
            <el-input v-model="configForm.yipay_gateway_url" placeholder="例如：https://pay.example.com" />
          </el-form-item>
        </template>

        <template v-if="configForm.pay_type === 'alipay'">
          <el-form-item label="支付宝公钥">
            <template v-if="isMobile"><div class="mobile-label">支付宝公钥</div></template>
            <el-input v-model="configForm.alipay_public_key" type="textarea" :rows="6" placeholder="请输入支付宝公钥" />
          </el-form-item>
          <el-form-item label="商户私钥">
            <template v-if="isMobile"><div class="mobile-label">商户私钥</div></template>
            <el-input v-model="configForm.merchant_private_key" type="textarea" :rows="6" placeholder="请输入应用私钥" />
          </el-form-item>
          <el-form-item label="支付宝网关">
             <template v-if="isMobile"><div class="mobile-label">支付宝网关</div></template>
             <el-input v-model="configForm.alipay_gateway" placeholder="默认: https://openapi.alipay.com/gateway.do" />
          </el-form-item>
        </template>

        <template v-if="configForm.pay_type === 'wechat'">
          <el-form-item label="商户号">
            <el-input v-model="configForm.wechat_mch_id" />
          </el-form-item>
          <el-form-item label="API密钥">
            <el-input v-model="configForm.wechat_api_key" />
          </el-form-item>
        </template>

        <el-form-item label="同步回调地址">
          <template v-if="isMobile"><div class="mobile-label">同步回调地址</div></template>
          <el-input v-model="configForm.return_url" placeholder="支付完成后跳转的地址" />
          <div class="form-tip">例如：{{ baseUrl }}/api/v1/payment/success</div>
        </el-form-item>

        <el-form-item label="异步回调地址">
          <template v-if="isMobile"><div class="mobile-label">异步回调地址</div></template>
          <el-input v-model="configForm.notify_url" placeholder="支付状态通知地址" />
          <div class="form-tip">
            例如：{{ baseUrl }}/api/v1/payment/notify/{{ configForm.pay_type === 'alipay' ? 'alipay' : 'yipay' }}
          </div>
        </el-form-item>

        <el-form-item label="状态">
          <template v-if="isMobile"><div class="mobile-label">状态</div></template>
          <el-select v-model="configForm.status" style="width: 100%" :teleported="!isMobile">
            <el-option label="启用" :value="1" />
            <el-option label="禁用" :value="0" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer-buttons" :class="{ 'mobile-footer': isMobile }">
          <el-button @click="showAddDialog = false" :class="{ 'mobile-action-btn': isMobile }">取消</el-button>
          <el-button type="primary" @click="saveConfig" :loading="saving" :class="{ 'mobile-action-btn': isMobile }">
            {{ editingConfig ? '更新' : '创建' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showBulkOperationsDialog"
      title="批量操作"
      :width="isMobile ? '95%' : '500px'"
      :class="isMobile ? 'mobile-dialog' : ''"
    >
      <div v-if="selectedConfigs.length === 0" class="no-selection">
        <el-alert title="请先选择要操作的配置" type="warning" :closable="false" show-icon />
      </div>
      <div v-else>
        <el-alert :title="`已选择 ${selectedConfigs.length} 个配置`" type="info" :closable="false" show-icon style="margin-bottom: 20px;" />
        <div class="bulk-actions-list">
          <el-button type="success" @click="handleBatchAction('enable')" :loading="batchOperating" class="bulk-btn">
            <el-icon><Check /></el-icon> 批量启用
          </el-button>
          <el-button type="warning" @click="handleBatchAction('disable')" :loading="batchOperating" class="bulk-btn">
            <el-icon><Close /></el-icon> 批量禁用
          </el-button>
          <el-button type="danger" @click="handleBatchAction('delete')" :loading="batchOperating" class="bulk-btn">
            <el-icon><Delete /></el-icon> 批量删除
          </el-button>
        </div>
      </div>
      <template #footer>
        <el-button @click="showBulkOperationsDialog = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Operation, Plus, Edit, Delete, Check, Close, Loading } from '@element-plus/icons-vue'
import { paymentAPI } from '@/utils/api'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
dayjs.extend(timezone)

const PAYMENT_TYPES = {
  'alipay': { label: '支付宝', tag: 'success' },
  'wechat': { label: '微信支付', tag: 'primary' },
  'yipay': { label: '易支付', tag: 'warning' },
  'yipay_alipay': { label: '易支付-支付宝', tag: 'warning' },
  'yipay_wxpay': { label: '易支付-微信', tag: 'warning' },
  'yipay_qqpay': { label: '易支付-QQ钱包', tag: 'warning' }
}

const DEFAULT_FORM_STATE = {
  pay_type: '',
  app_id: '',
  merchant_private_key: '',
  alipay_public_key: '',
  alipay_gateway: 'https://openapi.alipay.com/gateway.do',
  wechat_mch_id: '',
  wechat_api_key: '',
  yipay_gateway_url: '',
  yipay_sign_type: 'MD5',
  yipay_platform_public_key: '',
  yipay_merchant_private_key: '',
  yipay_supported_types: ['alipay', 'wxpay'],
  yipay_type: 'alipay',
  yipay_pid: '',
  yipay_private_key: '',
  yipay_public_key: '',
  yipay_md5_key: '',
  return_url: '',
  notify_url: '',
  status: 1,
  sort_order: 0
}

const utils = {
  formatValue: (value) => {
    if (value === null || value === undefined) return ''
    if (typeof value === 'string') return value
    if (typeof value === 'object' && value.String !== undefined) {
      return value.Valid ? value.String : ''
    }
    return String(value)
  },
  safeParseJSON: (str) => {
    try {
      return typeof str === 'string' ? (str ? JSON.parse(str) : {}) : (str || {})
    } catch {
      return {}
    }
  },
  handleApiError: (error, defaultMsg) => {
    let msg = defaultMsg
    if (error.isNetworkError) msg = '网络连接失败'
    else if (error.isTimeoutError) msg = '请求超时'
    else if (error.response?.data?.detail) msg = error.response.data.detail
    else if (error.message && !error.message.includes('cancel')) msg = error.message
    ElMessage.error(msg)
  }
}

export default {
  name: 'AdminPaymentConfig',
  components: { Operation, Plus, Edit, Delete, Check, Close, Loading },
  setup() {
    const loading = ref(false)
    const saving = ref(false)
    const paymentConfigs = ref([])
    const showAddDialog = ref(false)
    const showBulkOperationsDialog = ref(false)
    const editingConfig = ref(null)
    const isMobile = ref(false)
    const selectedConfigs = ref([])
    const batchOperating = ref(false)
    const tableRef = ref(null)
    const configForm = reactive({ ...DEFAULT_FORM_STATE })

    const baseUrl = computed(() => typeof window !== 'undefined' ? window.location.origin : '')

    const checkMobile = () => isMobile.value = window.innerWidth <= 768

    const getPaymentTypeConfig = (type) => PAYMENT_TYPES[type] || { label: type, tag: 'info' }

    const loadPaymentConfigs = async () => {
      loading.value = true
      try {
        const response = await paymentAPI.getPaymentConfigs({ page: 1, size: 100 })
        const rawItems = response?.data?.data?.items || response?.data?.items || response?.data || []
        
        paymentConfigs.value = (Array.isArray(rawItems) ? rawItems : []).map(config => ({
          ...config,
          status: [1, true, '1'].includes(config.status) ? 1 : 0,
          app_id: utils.formatValue(config.app_id),
          merchant_private_key: utils.formatValue(config.merchant_private_key),
          alipay_public_key: utils.formatValue(config.alipay_public_key),
          wechat_mch_id: utils.formatValue(config.wechat_mch_id),
          wechat_api_key: utils.formatValue(config.wechat_api_key),
          return_url: utils.formatValue(config.return_url),
          notify_url: utils.formatValue(config.notify_url),
          config_json: utils.safeParseJSON(config.config_json)
        }))
      } catch (error) {
        utils.handleApiError(error, '加载支付配置失败')
        paymentConfigs.value = []
      } finally {
        loading.value = false
      }
    }

    const buildRequestData = () => {
      const data = {
        pay_type: configForm.pay_type,
        status: configForm.status,
        return_url: configForm.return_url || '',
        notify_url: configForm.notify_url || '',
        sort_order: configForm.sort_order || 0
      }

      if (configForm.pay_type === 'alipay') {
        Object.assign(data, {
          app_id: configForm.app_id || '',
          merchant_private_key: configForm.merchant_private_key || '',
          alipay_public_key: configForm.alipay_public_key || '',
          config_json: {
            gateway_url: configForm.alipay_gateway || 'https://openapi.alipay.com/gateway.do',
            is_production: !(configForm.alipay_gateway || '').includes('alipaydev.com')
          }
        })
      } else if (configForm.pay_type === 'wechat') {
        Object.assign(data, {
          app_id: configForm.app_id,
          wechat_app_id: configForm.app_id,
          wechat_mch_id: configForm.wechat_mch_id,
          wechat_api_key: configForm.wechat_api_key
        })
      } else if (configForm.pay_type === 'yipay') {
        const gatewayUrl = (configForm.yipay_gateway_url || '').trim().replace(/\/$/, '')
        data.app_id = configForm.app_id || ''
        
        if (configForm.yipay_sign_type === 'MD5') {
          data.merchant_private_key = configForm.merchant_private_key || ''
        } else {
          data.merchant_private_key = configForm.merchant_private_key || ''
          data.alipay_public_key = configForm.yipay_platform_public_key || ''
        }

        data.config_json = {
          gateway_url: gatewayUrl,
          api_url: `${gatewayUrl}/mapi.php`,
          sign_type: configForm.yipay_sign_type,
          platform_public_key: configForm.yipay_platform_public_key || '',
          merchant_private_key: configForm.yipay_merchant_private_key || '',
          supported_types: configForm.yipay_supported_types
        }
      } else if (['yipay_alipay', 'yipay_wxpay', 'yipay_qqpay'].includes(configForm.pay_type)) {
        data.config_json = {
          yipay_type: configForm.pay_type === 'yipay_alipay' ? 'alipay' : 'wxpay',
          yipay_sign_type: configForm.yipay_sign_type,
          yipay_pid: configForm.yipay_pid,
          yipay_private_key: configForm.yipay_private_key || '',
          yipay_public_key: configForm.yipay_public_key || '',
          yipay_gateway: configForm.yipay_gateway_url || '',
          yipay_md5_key: configForm.yipay_md5_key || ''
        }
      }
      return data
    }

    const saveConfig = async () => {
      saving.value = true
      try {
        if (configForm.pay_type === 'yipay' && !configForm.yipay_gateway_url) {
          throw new Error('请填写易支付网关地址')
        }
        
        const requestData = buildRequestData()
        
        if (editingConfig.value) {
          await paymentAPI.updatePaymentConfig(editingConfig.value.id, requestData)
          ElMessage.success('更新成功')
        } else {
          await paymentAPI.createPaymentConfig(requestData)
          ElMessage.success('创建成功')
        }
        
        showAddDialog.value = false
        resetConfigForm()
        loadPaymentConfigs()
      } catch (error) {
        utils.handleApiError(error, '保存失败')
      } finally {
        saving.value = false
      }
    }

    const editConfig = (config) => {
      editingConfig.value = config
      const json = config.config_json || {}
      
      // Reset form then overlay data
      Object.assign(configForm, DEFAULT_FORM_STATE)
      
      const commonData = {
        pay_type: config.pay_type,
        app_id: config.app_id || json.app_id || '',
        merchant_private_key: config.merchant_private_key || json.merchant_private_key || '',
        return_url: config.return_url || '',
        notify_url: config.notify_url || '',
        status: config.status,
        sort_order: config.sort_order || 0
      }

      const specificData = {}
      if (config.pay_type === 'alipay') {
        specificData.alipay_public_key = config.alipay_public_key || json.alipay_public_key
        specificData.alipay_gateway = json.gateway_url || config.alipay_gateway
      } else if (config.pay_type === 'wechat') {
        specificData.wechat_mch_id = config.wechat_mch_id
        specificData.wechat_api_key = config.wechat_api_key
      } else if (config.pay_type === 'yipay') {
        specificData.yipay_gateway_url = json.gateway_url || (json.api_url ? json.api_url.replace('/mapi.php', '') : '')
        specificData.yipay_sign_type = json.sign_type || 'MD5'
        specificData.yipay_platform_public_key = json.platform_public_key || config.alipay_public_key
        specificData.yipay_merchant_private_key = json.merchant_private_key
        specificData.yipay_supported_types = json.supported_types || ['alipay', 'wxpay']
      } else {
        specificData.yipay_type = json.yipay_type
        specificData.yipay_pid = json.yipay_pid
        specificData.yipay_private_key = json.yipay_private_key
        specificData.yipay_public_key = json.yipay_public_key
        specificData.yipay_gateway_url = json.yipay_gateway
        specificData.yipay_md5_key = json.yipay_md5_key
        specificData.yipay_sign_type = json.yipay_sign_type || 'RSA'
      }

      Object.assign(configForm, commonData, specificData)
      showAddDialog.value = true
    }

    const deleteConfig = async (config) => {
      try {
        await ElMessageBox.confirm(`确定要删除 ${getPaymentTypeConfig(config.pay_type).label} 配置吗？`, '确认删除', { type: 'warning' })
        await paymentAPI.deletePaymentConfig(config.id)
        ElMessage.success('删除成功')
        loadPaymentConfigs()
      } catch (error) {
        if (error !== 'cancel') utils.handleApiError(error, '删除失败')
      }
    }

    const toggleStatus = async (config, newValue) => {
      const originalStatus = config.status
      const targetStatus = newValue !== undefined ? newValue : config.status
      try {
        const response = await paymentAPI.updatePaymentConfig(config.id, { status: targetStatus })
        config.status = response.data?.status ?? targetStatus
        ElMessage.success(targetStatus === 1 ? '已启用' : '已禁用')
      } catch (error) {
        config.status = originalStatus
        utils.handleApiError(error, '状态更新失败')
      }
    }

    const handleBatchAction = async (action) => {
      if (!selectedConfigs.value.length) return
      
      const actionMap = {
        'enable': { title: '批量启用', api: paymentAPI.bulkEnablePaymentConfigs },
        'disable': { title: '批量禁用', api: paymentAPI.bulkDisablePaymentConfigs },
        'delete': { title: '批量删除', api: paymentAPI.bulkDeletePaymentConfigs, type: 'error' }
      }
      
      const conf = actionMap[action]
      try {
        await ElMessageBox.confirm(
          `确定要${conf.title} ${selectedConfigs.value.length} 个配置吗？`, 
          conf.title, 
          { type: conf.type || 'warning' }
        )
        batchOperating.value = true
        await conf.api(selectedConfigs.value.map(c => c.id))
        ElMessage.success(`${conf.title}成功`)
        clearSelection()
        showBulkOperationsDialog.value = false
        loadPaymentConfigs()
      } catch (error) {
        if (error !== 'cancel') utils.handleApiError(error, '批量操作失败')
      } finally {
        batchOperating.value = false
      }
    }

    const resetConfigForm = () => {
      Object.assign(configForm, DEFAULT_FORM_STATE)
      editingConfig.value = null
    }

    const openAddDialog = () => {
      resetConfigForm()
      showAddDialog.value = true
    }
    
    const openBulkDialog = () => showBulkOperationsDialog.value = true

    const handleSelectionChange = (selection) => selectedConfigs.value = selection
    const clearSelection = () => {
      selectedConfigs.value = []
      tableRef.value?.clearSelection()
    }
    const handleMobileAction = (cmd) => { if (cmd === 'bulk') openBulkDialog() }

    onMounted(() => {
      checkMobile()
      window.addEventListener('resize', checkMobile)
      loadPaymentConfigs()
    })

    onUnmounted(() => window.removeEventListener('resize', checkMobile))

    return {
      baseUrl,
      loading,
      saving,
      paymentConfigs,
      showAddDialog,
      showBulkOperationsDialog,
      editingConfig,
      configForm,
      selectedConfigs,
      batchOperating,
      isMobile,
      tableRef,
      saveConfig,
      editConfig,
      deleteConfig,
      toggleStatus,
      getPaymentTypeConfig,
      handleMobileAction,
      handleSelectionChange,
      clearSelection,
      handleBatchAction,
      openAddDialog,
      openBulkDialog
    }
  }
}
</script>

<style scoped>
.admin-payment-config { padding: 20px; }
.header-content { display: flex; justify-content: space-between; align-items: center; }
.header-actions { display: flex; gap: 10px; }
.text-muted { color: #909399; font-style: italic; }
.status-text { margin-left: 8px; font-size: 12px; color: #909399; }
.form-tip { font-size: 12px; color: #909399; margin-top: 4px; line-height: 1.5; }
.desktop-only { @media (max-width: 768px) { display: none !important; } }
.mobile-only { display: none; @media (max-width: 768px) { display: block; } }
.bulk-btn { width: 100%; margin-bottom: 10px; margin-left: 0 !important; }

/* 样式复用与清理 */
:deep(.el-input__wrapper), :deep(.el-textarea__inner) {
  border-radius: 0; box-shadow: none; border: 1px solid #dcdfe6; background-color: #fff;
}
:deep(.el-input__wrapper:hover), :deep(.el-input__wrapper.is-focus) {
  border-color: #409EFF;
}

@media (max-width: 768px) {
  .admin-payment-config { padding: 10px; }
  .header-content { flex-direction: column; gap: 12px; }
  .header-actions { width: 100%; }
  .header-actions .el-button { flex: 1; }
  
  .mobile-card-list { display: flex; flex-direction: column; gap: 12px; }
  .mobile-card { background: #fff; border: 1px solid #e4e7ed; border-radius: 8px; padding: 16px; }
  .card-row { display: flex; justify-content: space-between; padding: 10px 0; border-bottom: 1px solid #f0f0f0; }
  .card-row:last-of-type { border-bottom: none; }
  .card-actions { display: flex; gap: 12px; margin-top: 12px; border-top: 1px solid #f0f0f0; padding-top: 12px; }
  .card-actions .el-button { flex: 1; }
  
  .mobile-dialog :deep(.el-dialog__body) { padding: 15px !important; }
  .mobile-label { font-size: 14px; font-weight: 600; margin-bottom: 8px; color: #606266; }
  .mobile-footer { flex-direction: column; }
  .mobile-footer .el-button { width: 100%; margin: 0; }
}
</style>