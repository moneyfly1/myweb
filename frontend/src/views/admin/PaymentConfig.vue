<template>
  <div class="list-container admin-payment-config">
    <el-card>
      <template #header>
        <div class="header-content">
          <span>支付配置管理</span>
          <div class="header-actions desktop-only">
            <el-radio-group v-model="viewMode" size="small" class="view-mode-group">
              <el-radio-button label="table">表格</el-radio-button>
              <el-radio-button label="grid">方格</el-radio-button>
            </el-radio-group>
            <template v-if="viewMode === 'grid'">
              <el-radio-group v-model="gridOrientation" size="small" class="grid-orientation-group">
                <el-radio-button label="horizontal">横向</el-radio-button>
                <el-radio-button label="vertical">纵向</el-radio-button>
              </el-radio-group>
              <template v-if="gridOrientation === 'horizontal'">
                <el-select v-model="gridColumns" size="small" class="grid-columns-select">
                  <el-option label="2列" :value="2" />
                  <el-option label="3列" :value="3" />
                  <el-option label="4列" :value="4" />
                  <el-option label="5列" :value="5" />
                  <el-option label="6列" :value="6" />
                </el-select>
              </template>
              <template v-else>
                <el-radio-group v-model="gridSize" size="small" class="grid-size-group">
                  <el-radio-button label="small">窄</el-radio-button>
                  <el-radio-button label="medium">中</el-radio-button>
                  <el-radio-button label="large">宽</el-radio-button>
                </el-radio-group>
              </template>
            </template>
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
      <ResponsiveDataView
        class="payment-config-data-view"
        :data="paymentConfigs"
        :fields="mobilePaymentConfigFields"
        :loading="loading"
        title-field="id"
        empty-title="暂无支付配置"
        empty-description="可添加支付宝、微信、易支付或码支付配置"
      >
        <template #table>
          <div class="table-wrapper desktop-only" v-if="viewMode === 'table'">
            <el-table
              ref="tableRef"
              :data="paymentConfigs"
              class="payment-config-table"
              v-loading="loading"
              stripe
              border
              :empty-text="paymentConfigs.length === 0 ? '暂无支付配置' : '暂无数据'"
              @selection-change="handleSelectionChange"
              @header-dragend="handleColumnResize"
            >
              <el-table-column type="selection" :width="columnWidths.selection" resizable />
              <el-table-column prop="id" label="ID" :width="columnWidths.id" resizable />
              <el-table-column prop="pay_type" label="支付类型" :width="columnWidths.pay_type" resizable>
                <template #default="scope">
                  <el-tag :type="getPaymentTypeConfig(scope.row.pay_type).tag">
                    {{ getPaymentTypeConfig(scope.row.pay_type).label }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="app_id" label="应用ID/商户ID" :min-width="columnWidths.app_id" resizable show-overflow-tooltip>
                <template #default="scope">
                  <span v-if="scope.row.app_id">{{ scope.row.app_id }}</span>
                  <span v-else-if="scope.row.config_json?.yipay_pid">
                    {{ scope.row.config_json.yipay_pid }} ({{ getPaymentTypeConfig(scope.row.pay_type).label }})
                  </span>
                  <span v-else class="text-muted">未配置</span>
                </template>
              </el-table-column>
              <el-table-column prop="status" label="状态" :width="columnWidths.status" resizable align="center">
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
              <el-table-column prop="created_at" label="创建时间" :width="columnWidths.created_at" resizable />
              <el-table-column label="操作" :width="columnWidths.actions" resizable align="center">
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
          <div v-if="viewMode === 'grid' && !isMobile" class="desktop-grid-view" :class="[
            gridOrientation === 'horizontal' ? 'grid-horizontal' : 'grid-vertical',
            gridOrientation === 'vertical' ? 'grid-size-' + gridSize : '',
            'grid-cols-' + gridColumns
          ]" v-loading="loading">
            <template v-if="paymentConfigs.length === 0">
              <EmptyState
                class="grid-empty"
                title="暂无支付配置"
                description="可添加支付宝、微信、易支付或码支付配置"
                action-text="添加支付配置"
                :loading="loading"
                @action="openAddDialog"
              />
            </template>
            <template v-else>
              <div
                v-for="config in paymentConfigs"
                :key="config.id"
                class="grid-config-card"
                :class="{ 'is-selected': isSelected(config) }"
              >
                <div class="gcc-header">
                  <el-checkbox
                    :model-value="isSelected(config)"
                    @change="(val) => handleGridSelect(config, val)"
                    class="gcc-checkbox"
                  />
                  <span class="gcc-title">#{{ config.id }}</span>
                  <el-tag :type="getPaymentTypeConfig(config.pay_type).tag" size="small" effect="dark">
                    {{ getPaymentTypeConfig(config.pay_type).label }}
                  </el-tag>
                </div>
                <div class="gcc-body">
                  <div class="gcc-row">
                    <span class="label">应用ID/商户ID</span>
                    <span class="value">
                      <span v-if="config.app_id">{{ config.app_id }}</span>
                      <span v-else-if="config.config_json?.yipay_pid">{{ config.config_json.yipay_pid }}</span>
                      <span v-else class="text-muted">未配置</span>
                    </span>
                  </div>
                  <div class="gcc-row">
                    <span class="label">创建时间</span>
                    <span class="value text-xs">{{ config.created_at || '-' }}</span>
                  </div>
                </div>
                <div class="gcc-footer">
                  <el-switch
                    v-model="config.status"
                    :active-value="1"
                    :inactive-value="0"
                    @change="(val) => toggleStatus(config, val)"
                    size="small"
                    inline-prompt
                    active-text="启用"
                    inactive-text="禁用"
                  />
                  <div class="gcc-actions">
                    <el-button size="small" type="primary" @click="editConfig(config)">编辑</el-button>
                    <el-button size="small" type="danger" @click="deleteConfig(config)">删除</el-button>
                  </div>
                </div>
              </div>
            </template>
          </div>
        </template>
        <template #header="{ item }">
          <div class="mobile-payment-header">
            <span>#{{ item.id }}</span>
            <el-tag :type="getPaymentTypeConfig(item.pay_type).tag" size="small">
              {{ getPaymentTypeConfig(item.pay_type).label }}
            </el-tag>
          </div>
        </template>
        <template #field-status="{ item }">
          <div class="mobile-status-control">
            <el-switch
              v-model="item.status"
              :active-value="1"
              :inactive-value="0"
              @change="(val) => toggleStatus(item, val)"
            />
            <span class="status-text">
              {{ item.status === 1 ? '已启用' : '已禁用' }}
            </span>
          </div>
        </template>
        <template #empty>
          <EmptyState
            title="暂无支付配置"
            description="可添加支付宝、微信、易支付或码支付配置"
            action-text="添加支付配置"
            :loading="loading"
            @action="openAddDialog"
          />
        </template>
        <template #actions="{ item }">
          <el-button size="small" type="primary" @click="editConfig(item)">
            <el-icon><Edit /></el-icon> 编辑
          </el-button>
          <el-button size="small" type="danger" @click="deleteConfig(item)">
            <el-icon><Delete /></el-icon> 删除
          </el-button>
        </template>
      </ResponsiveDataView>
    </el-card>
    <AppDrawer
      v-model="showAddDialog"
      :title="editingConfig ? '编辑支付配置' : '添加支付配置'"
      size="500px"
      mobile-size="100%"
      :loading="saving"
      class="payment-config-drawer"
    >
      <el-form :model="configForm" :label-width="isMobile ? '0' : '120px'" :label-position="isMobile ? 'top' : 'right'">
        <el-form-item label="支付类型">
          <template v-if="isMobile"><div class="mobile-label">支付类型</div></template>
          <el-select 
            v-model="configForm.pay_type" 
            placeholder="选择支付类型"
            class="full-width-control"
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
            <el-option-group label="码支付">
              <el-option label="码支付（统一配置，推荐）" value="codepay" />
              <el-option label="码支付-支付宝" value="codepay_alipay" />
              <el-option label="码支付-微信" value="codepay_wxpay" />
            </el-option-group>
          </el-select>
        </el-form-item>
        <template v-if="['alipay', 'wechat'].includes(configForm.pay_type)">
          <el-form-item label="应用ID">
            <template v-if="isMobile"><div class="mobile-label">应用ID</div></template>
            <el-input v-model="configForm.app_id" placeholder="请输入应用ID" class="full-width-control" />
            <div class="form-tip" v-if="configForm.pay_type === 'alipay'">
              <div>请确保已签约“当面付”产品并应用已上线。</div>
              <div>回调地址需同时在支付宝后台的应用网关中配置。</div>
            </div>
          </el-form-item>
        </template>
        <template v-if="configForm.pay_type.startsWith('yipay')">
          <el-form-item label="商户ID">
            <template v-if="isMobile"><div class="mobile-label">商户ID</div></template>
            <el-input v-model="configForm.app_id" placeholder="请输入易支付商户ID (pid)" class="full-width-control" />
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
               class="full-width-control"
             />
           </el-form-item>
        </template>
        <template v-if="configForm.pay_type === 'yipay'">
          <el-form-item label="网关地址">
            <template v-if="isMobile"><div class="mobile-label">网关地址</div></template>
            <el-input v-model="configForm.yipay_gateway_url" placeholder="请输入易支付网关地址" class="full-width-control" />
            <div class="form-tip">填写官网地址，系统自动拼接 API 路径。</div>
          </el-form-item>
          <el-form-item label="签名方式">
            <template v-if="isMobile"><div class="mobile-label">签名方式</div></template>
            <el-select v-model="configForm.yipay_sign_type" class="full-width-control" :teleported="!isMobile">
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
          <el-alert title="建议使用“易支付（统一配置）”以获得更好的兼容性" type="warning" :closable="false" class="form-alert" />
          <el-form-item label="商户ID">
            <el-input v-model="configForm.yipay_pid" placeholder="请输入易支付商户ID" />
          </el-form-item>
          <el-form-item label="签名类型">
             <el-select v-model="configForm.yipay_sign_type" class="full-width-control" :teleported="!isMobile">
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
        <template v-if="configForm.pay_type === 'codepay'">
          <el-form-item label="商户ID">
            <template v-if="isMobile"><div class="mobile-label">商户ID</div></template>
            <el-input v-model="configForm.app_id" placeholder="请输入码支付商户ID (pid)" class="full-width-control" />
          </el-form-item>
          <el-form-item label="商户密钥">
            <template v-if="isMobile"><div class="mobile-label">商户密钥</div></template>
            <el-input
              v-model="configForm.merchant_private_key"
              type="password"
              show-password
              placeholder="请输入码支付商户密钥 (key)"
              class="full-width-control"
            />
          </el-form-item>
          <el-form-item label="网关地址">
            <template v-if="isMobile"><div class="mobile-label">网关地址</div></template>
            <el-input v-model="configForm.codepay_gateway_url" placeholder="请输入码支付网关地址" class="full-width-control" />
            <div class="form-tip">填写官网地址，系统自动拼接 API 路径（/xpay/epay/mapi.php）。</div>
          </el-form-item>
          <el-form-item label="支持支付方式">
            <template v-if="isMobile"><div class="mobile-label">支持支付方式</div></template>
            <el-checkbox-group v-model="configForm.codepay_supported_types">
              <el-checkbox label="alipay">支付宝</el-checkbox>
              <el-checkbox label="wxpay">微信支付</el-checkbox>
            </el-checkbox-group>
          </el-form-item>
        </template>
        <template v-if="['codepay_alipay', 'codepay_wxpay'].includes(configForm.pay_type)">
          <el-form-item label="商户ID">
            <el-input v-model="configForm.app_id" placeholder="请输入码支付商户ID (pid)" />
          </el-form-item>
          <el-form-item label="商户密钥">
            <el-input v-model="configForm.merchant_private_key" type="password" show-password placeholder="请输入码支付商户密钥 (key)" />
          </el-form-item>
          <el-form-item label="网关地址">
            <el-input v-model="configForm.codepay_gateway_url" placeholder="例如：https://pay.example.com" />
            <div class="form-tip">填写官网地址，系统自动拼接 API 路径。</div>
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
            例如：{{ baseUrl }}/api/v1/payment/notify/{{ configForm.pay_type === 'alipay' ? 'alipay' : configForm.pay_type.startsWith('codepay') ? 'codepay' : 'yipay' }}
          </div>
        </el-form-item>
        <el-form-item label="状态">
          <template v-if="isMobile"><div class="mobile-label">状态</div></template>
          <el-select v-model="configForm.status" class="full-width-control" :teleported="!isMobile">
            <el-option label="启用" :value="1" />
            <el-option label="禁用" :value="0" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <FormActionBar
          :loading="saving"
          :submit-text="editingConfig ? '更新' : '创建'"
          :sticky="false"
          @cancel="showAddDialog = false"
          @submit="saveConfig"
        />
      </template>
    </AppDrawer>
    <AppDialog
      v-model="showBulkOperationsDialog"
      title="批量操作"
      width="500px"
      mobile-width="94%"
      :loading="batchOperating"
    >
      <div v-if="selectedConfigs.length === 0" class="no-selection">
        <el-alert title="请先选择要操作的配置" type="warning" :closable="false" show-icon />
      </div>
      <div v-else>
        <el-alert :title="`已选择 ${selectedConfigs.length} 个配置`" type="info" :closable="false" show-icon class="bulk-selection-alert" />
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
        <FormActionBar
          submit-text="关闭"
          :show-cancel="false"
          @submit="showBulkOperationsDialog = false"
        />
      </template>
    </AppDialog>
  </div>
</template>
<script>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { Operation, Plus, Edit, Delete, Check, Close, Loading } from '@element-plus/icons-vue'
import { paymentAPI } from '@/utils/api'
import { usePersistentTableColumns } from '@/composables/usePersistentTableColumns'
import { useMobile } from '@/composables/useMobile'
import { confirmWarning, confirmDelete } from '@/utils/confirmAction'
import AppDialog from '@/components/AppDialog.vue'
import AppDrawer from '@/components/AppDrawer.vue'
import EmptyState from '@/components/EmptyState.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
dayjs.extend(timezone)
const PAYMENT_TYPES = {
  'alipay': { label: '支付宝', tag: 'success' },
  'wechat': { label: '微信支付', tag: 'primary' },
  'yipay': { label: '易支付', tag: 'warning' },
  'yipay_alipay': { label: '易支付-支付宝', tag: 'warning' },
  'yipay_wxpay': { label: '易支付-微信', tag: 'warning' },
  'yipay_qqpay': { label: '易支付-QQ钱包', tag: 'warning' },
  'codepay': { label: '码支付', tag: 'danger' },
  'codepay_alipay': { label: '码支付-支付宝', tag: 'danger' },
  'codepay_wxpay': { label: '码支付-微信', tag: 'danger' }
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
  codepay_gateway_url: '',
  codepay_supported_types: ['alipay', 'wxpay'],
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
  components: { Operation, Plus, Edit, Delete, Check, Close, Loading, AppDialog, AppDrawer, EmptyState, FormActionBar, ResponsiveDataView },
  setup() {
    const loading = ref(false)
    const saving = ref(false)
    const paymentConfigs = ref([])
    const showAddDialog = ref(false)
    const showBulkOperationsDialog = ref(false)
    const editingConfig = ref(null)
    const isMobile = useMobile()
    const viewMode = ref('table') // 'table' | 'grid'
    const gridOrientation = ref('horizontal') // 'horizontal' | 'vertical'
    const gridColumns = ref(3) // 2-6 columns for horizontal
    const gridSize = ref('medium') // 'small' | 'medium' | 'large' for vertical
    const selectedConfigs = ref([])
    const batchOperating = ref(false)
    const tableRef = ref(null)
    const configForm = reactive({ ...DEFAULT_FORM_STATE })
    
    const PAYMENT_CONFIG_COLUMN_KEYS = ['selection', 'id', 'pay_type', 'app_id', 'status', 'created_at', 'actions']
    const { columnWidths, handleColumnResize } = usePersistentTableColumns(
      'paymentConfig_table_column_widths',
      {
        selection: 50,
        id: 80,
        pay_type: 120,
        app_id: 200,
        status: 120,
        created_at: 180,
        actions: 180
      },
      PAYMENT_CONFIG_COLUMN_KEYS
    )

    // 从 localStorage 加载视图设置
    const STORAGE_KEY = 'paymentConfig_table_settings'
    const loadSettings = () => {
      try {
        const saved = localStorage.getItem(STORAGE_KEY)
        if (saved) {
          const settings = JSON.parse(saved)
          if (settings.viewMode) viewMode.value = settings.viewMode
          if (settings.gridOrientation) gridOrientation.value = settings.gridOrientation
          if (settings.gridColumns) gridColumns.value = settings.gridColumns
          if (settings.gridSize) gridSize.value = settings.gridSize
        }
      } catch (e) {
        console.warn('加载设置失败:', e)
      }
    }
    
    // 保存视图设置到 localStorage
    const saveSettings = () => {
      try {
        const settings = {
          viewMode: viewMode.value,
          gridOrientation: gridOrientation.value,
          gridColumns: gridColumns.value,
          gridSize: gridSize.value
        }
        localStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
      } catch (e) {
        console.warn('保存设置失败:', e)
      }
    }
    
    // 网格视图选择处理
    const handleGridSelect = (config, checked) => {
      if (checked) {
        if (!selectedConfigs.value.find(c => c.id === config.id)) {
          selectedConfigs.value.push(config)
        }
      } else {
        selectedConfigs.value = selectedConfigs.value.filter(c => c.id !== config.id)
      }
    }
    
    const isSelected = (config) => selectedConfigs.value.some(c => c.id === config.id)
    const baseUrl = computed(() => typeof window !== 'undefined' ? window.location.origin : '')
    const getPaymentTypeConfig = (type) => PAYMENT_TYPES[type] || { label: type, tag: 'info' }
    const getDisplayAppId = (config) => {
      if (config.app_id) return config.app_id
      if (config.config_json?.yipay_pid) return config.config_json.yipay_pid
      return ''
    }
    const mobilePaymentConfigFields = computed(() => [
      {
        key: 'pay_type',
        label: '支付类型',
        type: 'tag',
        tagType: value => getPaymentTypeConfig(value).tag,
        formatter: value => getPaymentTypeConfig(value).label
      },
      {
        key: 'app_id',
        label: '应用ID',
        formatter: (_value, item) => getDisplayAppId(item) || '未配置'
      },
      {
        key: 'status',
        label: '状态'
      },
      {
        key: 'created_at',
        label: '创建时间',
        formatter: value => value || '-'
      }
    ])
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
      } else if (configForm.pay_type === 'codepay') {
        const gatewayUrl = (configForm.codepay_gateway_url || '').trim().replace(/\/$/, '')
        const apiUrl = gatewayUrl.endsWith('/xpay/epay') ? `${gatewayUrl}/mapi.php` : `${gatewayUrl}/xpay/epay/mapi.php`
        data.app_id = configForm.app_id || ''
        data.merchant_private_key = configForm.merchant_private_key || ''
        data.config_json = {
          gateway_url: gatewayUrl,
          api_url: apiUrl,
          supported_types: configForm.codepay_supported_types
        }
      } else if (['codepay_alipay', 'codepay_wxpay'].includes(configForm.pay_type)) {
        const gatewayUrl = (configForm.codepay_gateway_url || '').trim().replace(/\/$/, '')
        const apiUrl = gatewayUrl.endsWith('/xpay/epay') ? `${gatewayUrl}/mapi.php` : `${gatewayUrl}/xpay/epay/mapi.php`
        data.app_id = configForm.app_id || ''
        data.merchant_private_key = configForm.merchant_private_key || ''
        data.config_json = {
          gateway_url: gatewayUrl,
          api_url: apiUrl
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
        if (configForm.pay_type === 'codepay' && !configForm.codepay_gateway_url) {
          throw new Error('请填写码支付网关地址')
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
      } else if (config.pay_type === 'codepay') {
        specificData.codepay_gateway_url = json.gateway_url || (json.api_url ? json.api_url.replace('/xpay/epay/mapi.php', '') : '')
        specificData.codepay_supported_types = json.supported_types || ['alipay', 'wxpay']
      } else if (['codepay_alipay', 'codepay_wxpay'].includes(config.pay_type)) {
        specificData.codepay_gateway_url = json.gateway_url || (json.api_url ? json.api_url.replace('/xpay/epay/mapi.php', '') : '')
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
        await confirmDelete('支付配置', 1, {
          message: `确定要删除 ${getPaymentTypeConfig(config.pay_type).label} 配置吗？删除后该支付方式将不可用。`
        })
        await paymentAPI.deletePaymentConfig(config.id)
        ElMessage.success('删除成功')
        loadPaymentConfigs()
      } catch (error) {
        if (error !== 'cancel') utils.handleApiError(error, '删除失败')
      }
    }
    const toggleStatus = async (config, newValue) => {
      const targetStatus = newValue !== undefined ? newValue : config.status
      const originalStatus = targetStatus === 1 ? 0 : 1
      try {
        if (targetStatus === 0) {
          await confirmWarning('确定要禁用该支付配置吗？禁用后用户将无法继续使用该支付方式发起支付。', {
            title: '确认禁用支付配置',
            confirmButtonText: '确认禁用'
          })
        }
        const response = await paymentAPI.updatePaymentConfig(config.id, { status: targetStatus })
        config.status = response.data?.status ?? targetStatus
        ElMessage.success(targetStatus === 1 ? '已启用' : '已禁用')
      } catch (error) {
        config.status = originalStatus
        if (error === 'cancel' || error === 'close') return
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
        if (action === 'delete') {
          await confirmDelete('支付配置', selectedConfigs.value.length, {
            title: conf.title
          })
        } else {
          await confirmWarning(`确定要${conf.title} ${selectedConfigs.value.length} 个配置吗？`, {
            title: conf.title,
            confirmButtonText: `确认${action === 'enable' ? '启用' : '禁用'}`
          })
        }
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
    
    // 监听视图模式和网格设置变化，自动保存
    watch([viewMode, gridOrientation, gridColumns, gridSize], () => {
      saveSettings()
    })
    
    onMounted(() => {
      loadSettings() // 先加载保存的设置
      loadPaymentConfigs()
    })
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
      viewMode,
      gridOrientation,
      gridColumns,
      gridSize,
      columnWidths,
      tableRef,
      saveConfig,
      editConfig,
      deleteConfig,
      toggleStatus,
      getPaymentTypeConfig,
      handleMobileAction,
      handleSelectionChange,
      handleGridSelect,
      isSelected,
      mobilePaymentConfigFields,
      handleColumnResize,
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
.view-mode-group { margin-right: 8px; }
.grid-orientation-group { margin-right: 8px; }
.grid-size-group { margin-right: 8px; }
.grid-columns-select {
  width: 90px;
  margin-right: 8px;
}
.payment-config-table,
.full-width-control {
  width: 100%;
}
.form-alert,
.bulk-selection-alert {
  margin-bottom: 20px;
}
.text-muted { color: #909399; font-style: italic; }
.status-text { margin-left: 8px; font-size: 12px; color: #909399; }
.form-tip { font-size: 12px; color: #909399; margin-top: 4px; line-height: 1.5; }
.desktop-only { @media (max-width: 768px) { display: none !important; } }
.mobile-only { display: none; @media (max-width: 768px) { display: block; } }
.bulk-btn { width: 100%; margin-bottom: 10px; margin-left: 0 !important; }
.mobile-payment-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
  font-size: 15px;
  font-weight: 600;
}
.mobile-status-control {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  min-width: 0;
}

/* 桌面端方格视图（可调大小和方向） */
.desktop-grid-view {
  display: grid;
  gap: 16px;
  min-height: 120px;
}
/* 横向布局：固定列数 */
.desktop-grid-view.grid-horizontal.grid-cols-2 {
  grid-template-columns: repeat(2, 1fr);
}
.desktop-grid-view.grid-horizontal.grid-cols-3 {
  grid-template-columns: repeat(3, 1fr);
}
.desktop-grid-view.grid-horizontal.grid-cols-4 {
  grid-template-columns: repeat(4, 1fr);
}
.desktop-grid-view.grid-horizontal.grid-cols-5 {
  grid-template-columns: repeat(5, 1fr);
}
.desktop-grid-view.grid-horizontal.grid-cols-6 {
  grid-template-columns: repeat(6, 1fr);
}
/* 纵向布局：单列，可调宽度 */
.desktop-grid-view.grid-vertical {
  grid-template-columns: 1fr;
  max-width: 100%;
}
.desktop-grid-view.grid-vertical.grid-size-small {
  max-width: 400px;
  margin: 0 auto;
}
.desktop-grid-view.grid-vertical.grid-size-medium {
  max-width: 600px;
  margin: 0 auto;
}
.desktop-grid-view.grid-vertical.grid-size-large {
  max-width: 800px;
  margin: 0 auto;
}
.grid-empty {
  grid-column: 1 / -1;
}
.grid-config-card {
  background: #fff;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  transition: border-color 0.2s, background-color 0.2s;
}
.grid-config-card:hover {
  border-color: var(--el-border-color);
  background: #fbfdff;
}
.grid-config-card.is-selected {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9, #ecf5ff);
}
.grid-config-card .gcc-header {
  padding: 12px 16px;
  background: #f8f9fa;
  border-bottom: 1px solid #ebeef5;
  display: flex;
  align-items: center;
  gap: 8px;
}
.grid-config-card .gcc-checkbox { margin-right: 0; flex-shrink: 0; }
.grid-config-card .gcc-title {
  flex: 1;
  font-weight: 600;
  font-size: 14px;
  color: var(--el-text-color-primary);
}
.grid-config-card .gcc-body {
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
}
.grid-config-card .gcc-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
}
.grid-config-card .gcc-row .label {
  color: var(--el-text-color-secondary);
  margin-right: 8px;
}
.grid-config-card .gcc-row .value {
  font-weight: 500;
  word-break: break-all;
  text-align: right;
}
.grid-config-card .gcc-footer {
  padding: 10px 16px;
  border-top: 1px solid #f0f2f5;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-wrap: wrap;
}
.grid-config-card .gcc-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}
.admin-payment-config :deep(.el-input__wrapper), .admin-payment-config :deep(.el-textarea__inner) {
  border-radius: 0; box-shadow: none; border: 1px solid #dcdfe6; background-color: #fff;
}
.admin-payment-config :deep(.el-input__wrapper:hover), .admin-payment-config :deep(.el-input__wrapper.is-focus) {
  border-color: #409EFF;
}
@media (max-width: 768px) {
  .admin-payment-config { padding: 10px; }
  .header-content { flex-direction: column; gap: 12px; }
  .header-actions { width: 100%; }
  .header-actions .el-button { flex: 1; }
  .mobile-label { font-size: 14px; font-weight: 600; margin-bottom: 8px; color: #606266; }
}
</style>
