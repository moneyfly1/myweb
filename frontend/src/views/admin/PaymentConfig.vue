<template>
  <div class="admin-payment-config">
    <el-card>
      <template #header>
        <div class="header-content">
          <span>支付配置管理</span>
          <div class="header-actions desktop-only">
            <el-button type="success" @click="exportConfigs">
              <el-icon><Download /></el-icon>
              导出配置
            </el-button>
            <el-button type="warning" @click="showBulkOperationsDialog = true">
              <el-icon><Operation /></el-icon>
              批量操作
            </el-button>
            <el-button type="info" @click="handleShowStatistics">
              <el-icon><DataAnalysis /></el-icon>
              配置统计
            </el-button>
            <el-button type="primary" @click="showAddDialog = true">
              <el-icon><Plus /></el-icon>
              添加支付配置
            </el-button>
          </div>
          <div class="header-actions mobile-only">
            <el-button type="primary" @click="showAddDialog = true" size="small">
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
                  <el-dropdown-item command="export">
                    <el-icon><Download /></el-icon>
                    导出配置
                  </el-dropdown-item>
                  <el-dropdown-item command="bulk">
                    <el-icon><Operation /></el-icon>
                    批量操作
                  </el-dropdown-item>
                  <el-dropdown-item command="statistics">
                    <el-icon><DataAnalysis /></el-icon>
                    配置统计
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </template>
      <!-- 批量操作工具栏 -->
      <div class="batch-actions" v-if="selectedConfigs.length > 0">
        <div class="batch-info">
          <span>已选择 {{ selectedConfigs.length }} 个配置</span>
        </div>
        <div class="batch-buttons">
          <el-button type="success" @click="batchEnableConfigs" :loading="batchOperating">
            <el-icon><Check /></el-icon>
            批量启用
          </el-button>
          <el-button type="warning" @click="batchDisableConfigs" :loading="batchOperating">
            <el-icon><Close /></el-icon>
            批量禁用
          </el-button>
          <el-button type="danger" @click="batchDeleteConfigs" :loading="batchOperating">
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
          :empty-text="paymentConfigs.length === 0 ? '暂无支付配置，请点击右上角【添加支付配置】按钮添加' : '暂无数据'"
          @selection-change="handleSelectionChange"
        >
          <el-table-column type="selection" width="50" />
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="pay_type" label="支付类型" width="120">
            <template #default="scope">
              <el-tag :type="getTypeTagType(scope.row.pay_type)">
                {{ getTypeText(scope.row.pay_type) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="app_id" label="应用ID/商户ID" min-width="200">
            <template #default="scope">
              <span v-if="scope.row.app_id">{{ scope.row.app_id }}</span>
              <span v-else-if="scope.row.config_json && scope.row.config_json.yipay_pid">
                {{ scope.row.config_json.yipay_pid }} ({{ getTypeText(scope.row.pay_type) }})
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
                @change="(newValue) => toggleStatus(scope.row, newValue)"
              />
              <span style="margin-left: 8px; font-size: 12px; color: #909399;">
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

      <!-- 移动端卡片式列表 -->
      <div class="mobile-card-list mobile-only" v-if="paymentConfigs.length > 0">
        <div 
          v-for="config in paymentConfigs" 
          :key="config.id"
          class="mobile-card"
        >
          <div class="card-row">
            <span class="label">ID</span>
            <span class="value">#{{ config.id }}</span>
          </div>
          <div class="card-row">
            <span class="label">支付类型</span>
            <span class="value">
              <el-tag :type="getTypeTagType(config.pay_type)">
                {{ getTypeText(config.pay_type) }}
              </el-tag>
            </span>
          </div>
          <div class="card-row">
            <span class="label">应用ID/商户ID</span>
            <span class="value">
              <span v-if="config.app_id">{{ config.app_id }}</span>
              <span v-else-if="config.config_json && config.config_json.yipay_pid">
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
                @change="(newValue) => toggleStatus(config, newValue)"
              />
              <span style="margin-left: 8px; font-size: 14px; color: #909399;">
                {{ config.status === 1 ? '已启用' : '已禁用' }}
              </span>
            </span>
          </div>
          <div class="card-row">
            <span class="label">创建时间</span>
            <span class="value">{{ config.created_at || '-' }}</span>
          </div>
          <div class="card-actions">
            <el-button 
              size="small" 
              type="primary" 
              @click="editConfig(config)"
            >
              <el-icon><Edit /></el-icon>
              编辑
            </el-button>
            <el-button 
              size="small" 
              type="danger" 
              @click="deleteConfig(config)"
            >
              <el-icon><Delete /></el-icon>
              删除
            </el-button>
          </div>
        </div>
      </div>

      <!-- 移动端空状态 -->
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
          <el-select 
            v-model="configForm.pay_type" 
            placeholder="选择支付类型"
            style="width: 100%"
            :teleported="!isMobile"
          >
            <el-option-group label="官方支付">
              <el-option label="支付宝" value="alipay" />
              <el-option label="微信支付" value="wechat" />
            </el-option-group>
            <el-option-group label="第三方支付网关">
              <el-option label="易支付-支付宝" value="yipay_alipay" />
              <el-option label="易支付-微信" value="yipay_wxpay" />
              <el-option label="码支付-支付宝" value="codepay_alipay" />
              <el-option label="码支付-微信" value="codepay_wechat" />
              <el-option label="码支付-QQ钱包" value="codepay_qq" />
            </el-option-group>
            <el-option-group label="国际支付">
              <el-option label="PayPal" value="paypal" />
              <el-option label="Stripe" value="stripe" />
            </el-option-group>
            <el-option-group label="其他">
              <el-option label="银行转账" value="bank_transfer" />
            </el-option-group>
          </el-select>
        </el-form-item>

        <el-form-item label="应用ID" v-if="configForm.pay_type === 'alipay' || configForm.pay_type === 'wechat'">
          <el-input v-model="configForm.app_id" placeholder="请输入应用ID" style="width: 100%" />
        </el-form-item>
        <el-form-item label="商户ID" v-if="configForm.pay_type === 'yipay_alipay' || configForm.pay_type === 'yipay_wxpay'">
          <el-input v-model="configForm.yipay_pid" placeholder="请输入易支付商户ID" style="width: 100%" />
          <div class="form-tip">在易支付商户后台->个人资料->API信息中查看（易支付-支付宝和易支付-微信使用相同的商户ID）</div>
        </el-form-item>

        <el-form-item label="签名类型" v-if="configForm.pay_type === 'yipay_alipay' || configForm.pay_type === 'yipay_wxpay'">
          <el-select 
            v-model="configForm.yipay_sign_type" 
            placeholder="选择签名类型"
            style="width: 100%"
            :teleported="!isMobile"
          >
            <el-option label="RSA签名（推荐）" value="RSA" />
            <el-option label="MD5签名" value="MD5" />
          </el-select>
          <div class="form-tip">选择签名类型：RSA（使用RSA私钥/公钥）或MD5（使用MD5密钥）</div>
        </el-form-item>

        <el-form-item label="商户私钥（RSA签名）" v-if="(configForm.pay_type === 'yipay_alipay' || configForm.pay_type === 'yipay_wxpay') && configForm.yipay_sign_type === 'RSA'">
          <el-input
            v-model="configForm.yipay_private_key"
            type="textarea"
            :rows="isMobile ? 6 : 4"
            placeholder="请输入易支付商户私钥"
            style="width: 100%"
          />
          <div class="form-tip">在易支付商户后台->个人资料->API信息中点击"生成商户RSA密钥对"生成（RSA签名时必填，易支付-支付宝和易支付-微信使用相同的私钥）</div>
        </el-form-item>

        <el-form-item label="平台公钥（RSA签名）" v-if="(configForm.pay_type === 'yipay_alipay' || configForm.pay_type === 'yipay_wxpay') && configForm.yipay_sign_type === 'RSA'">
          <el-input
            v-model="configForm.yipay_public_key"
            type="textarea"
            :rows="isMobile ? 6 : 4"
            placeholder="请输入易支付平台公钥"
            style="width: 100%"
          />
          <div class="form-tip">在易支付商户后台->个人资料->API信息中查看（RSA签名时必填，用于验签，易支付-支付宝和易支付-微信使用相同的公钥）</div>
        </el-form-item>

        <el-form-item label="MD5密钥（MD5签名）" v-if="(configForm.pay_type === 'yipay_alipay' || configForm.pay_type === 'yipay_wxpay') && configForm.yipay_sign_type === 'MD5'">
          <el-input 
            v-model="configForm.yipay_md5_key" 
            placeholder="请输入易支付MD5密钥" 
            style="width: 100%"
            type="password"
            show-password
          />
          <div class="form-tip">在易支付商户后台->个人资料->API信息中查看（MD5签名时必填）</div>
        </el-form-item>

        <el-form-item label="网关地址" v-if="configForm.pay_type === 'yipay_alipay' || configForm.pay_type === 'yipay_wxpay'">
          <el-input v-model="configForm.yipay_gateway" placeholder="请输入易支付网关地址" style="width: 100%" />
          <div class="form-tip">默认: https://pay.yi-zhifu.cn/（系统会自动拼接V2接口路径 /api/pay/create）</div>
        </el-form-item>
        <el-form-item label="支付宝公钥" v-if="configForm.pay_type === 'alipay'">
          <el-input
            v-model="configForm.alipay_public_key"
            type="textarea"
            :rows="isMobile ? 6 : 4"
            placeholder="请输入支付宝公钥"
            style="width: 100%"
          />
        </el-form-item>

        <el-form-item label="商户私钥" v-if="configForm.pay_type === 'alipay'">
          <el-input
            v-model="configForm.merchant_private_key"
            type="textarea"
            :rows="isMobile ? 6 : 4"
            placeholder="请输入商户私钥"
            style="width: 100%"
          />
        </el-form-item>

        <el-form-item label="支付宝网关" v-if="configForm.pay_type === 'alipay'">
          <el-input v-model="configForm.alipay_gateway" placeholder="请输入支付宝网关地址" style="width: 100%" />
          <div class="form-tip">默认: https://openapi.alipay.com/gateway.do (生产环境) 或 https://openapi.alipaydev.com/gateway.do (沙箱环境)</div>
        </el-form-item>
        <el-form-item label="商户号" v-if="configForm.pay_type === 'wechat'">
          <el-input v-model="configForm.wechat_mch_id" placeholder="请输入微信商户号" style="width: 100%" />
        </el-form-item>

        <el-form-item label="API密钥" v-if="configForm.pay_type === 'wechat'">
          <el-input v-model="configForm.wechat_api_key" placeholder="请输入微信API密钥" style="width: 100%" />
        </el-form-item>

        <!-- PayPal配置 -->
        <el-form-item label="PayPal客户端ID" v-if="configForm.pay_type === 'paypal'">
          <el-input v-model="configForm.paypal_client_id" placeholder="请输入PayPal客户端ID" style="width: 100%" />
          <div class="form-tip">在PayPal开发者后台创建应用后获取</div>
        </el-form-item>
        <el-form-item label="PayPal客户端密钥" v-if="configForm.pay_type === 'paypal'">
          <el-input v-model="configForm.paypal_secret" type="password" show-password placeholder="请输入PayPal客户端密钥" style="width: 100%" />
          <div class="form-tip">在PayPal开发者后台创建应用后获取</div>
        </el-form-item>
        <el-form-item label="PayPal模式" v-if="configForm.pay_type === 'paypal'">
          <el-select 
            v-model="configForm.paypal_mode" 
            placeholder="选择PayPal模式"
            style="width: 100%"
            :teleported="!isMobile"
          >
            <el-option label="沙箱模式（测试）" value="sandbox" />
            <el-option label="生产模式（正式）" value="live" />
          </el-select>
          <div class="form-tip">沙箱模式用于测试，生产模式用于正式环境</div>
        </el-form-item>

        <!-- Stripe配置 -->
        <el-form-item label="Stripe公钥" v-if="configForm.pay_type === 'stripe'">
          <el-input v-model="configForm.stripe_publishable_key" placeholder="请输入Stripe公钥" style="width: 100%" />
          <div class="form-tip">用于前端的公钥（Publishable Key）</div>
        </el-form-item>
        <el-form-item label="Stripe私钥" v-if="configForm.pay_type === 'stripe'">
          <el-input v-model="configForm.stripe_secret_key" type="password" show-password placeholder="请输入Stripe私钥" style="width: 100%" />
          <div class="form-tip">用于后端的私钥（Secret Key）</div>
        </el-form-item>
        <el-form-item label="Stripe Webhook密钥" v-if="configForm.pay_type === 'stripe'">
          <el-input v-model="configForm.stripe_webhook_secret" type="password" show-password placeholder="请输入Stripe Webhook签名密钥（可选）" style="width: 100%" />
          <div class="form-tip">用于验证Webhook回调的签名密钥（可选）</div>
        </el-form-item>

        <!-- 码支付配置 -->
        <el-form-item label="码支付ID" v-if="configForm.pay_type.startsWith('codepay_')">
          <el-input v-model="configForm.codepay_id" placeholder="请输入码支付商户ID" style="width: 100%" />
          <div class="form-tip">在码支付后台->商户中心查看</div>
        </el-form-item>
        <el-form-item label="码支付Token" v-if="configForm.pay_type.startsWith('codepay_')">
          <el-input v-model="configForm.codepay_token" type="password" show-password placeholder="请输入码支付通信密钥Token" style="width: 100%" />
          <div class="form-tip">在码支付后台->商户中心->API接口中查看</div>
        </el-form-item>
        <el-form-item label="码支付网关" v-if="configForm.pay_type.startsWith('codepay_')">
          <el-input v-model="configForm.codepay_gateway" placeholder="请输入码支付网关地址" style="width: 100%" />
          <div class="form-tip">默认: https://api.xiuxiu888.com/creat_order</div>
        </el-form-item>

        <!-- 银行转账配置 -->
        <el-form-item label="银行名称" v-if="configForm.pay_type === 'bank_transfer'">
          <el-input v-model="configForm.bank_name" placeholder="请输入银行名称" style="width: 100%" />
        </el-form-item>
        <el-form-item label="银行账号" v-if="configForm.pay_type === 'bank_transfer'">
          <el-input v-model="configForm.bank_account" placeholder="请输入银行账号" style="width: 100%" />
        </el-form-item>
        <el-form-item label="开户支行" v-if="configForm.pay_type === 'bank_transfer'">
          <el-input v-model="configForm.bank_branch" placeholder="请输入开户支行（可选）" style="width: 100%" />
        </el-form-item>
        <el-form-item label="账户持有人" v-if="configForm.pay_type === 'bank_transfer'">
          <el-input v-model="configForm.account_holder" placeholder="请输入账户持有人姓名" style="width: 100%" />
        </el-form-item>

        <el-form-item label="同步回调地址">
          <el-input v-model="configForm.return_url" placeholder="请输入同步回调地址" style="width: 100%" />
          <div class="form-tip">支付完成后跳转的地址</div>
        </el-form-item>

        <el-form-item label="异步回调地址">
          <el-input v-model="configForm.notify_url" placeholder="请输入异步回调地址" style="width: 100%" />
          <div class="form-tip">支付完成后服务器通知的地址</div>
        </el-form-item>

        <el-form-item label="状态">
          <el-select 
            v-model="configForm.status" 
            placeholder="选择状态"
            style="width: 100%"
            :teleported="!isMobile"
          >
            <el-option label="启用" :value="1" />
            <el-option label="禁用" :value="0" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer-buttons">
          <el-button @click="showAddDialog = false" class="mobile-action-btn">取消</el-button>
          <el-button type="primary" @click="saveConfig" :loading="saving" class="mobile-action-btn">
            {{ editingConfig ? '更新' : '创建' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 批量操作对话框 -->
    <el-dialog
      v-model="showBulkOperationsDialog"
      title="批量操作"
      :width="isMobile ? '95%' : '500px'"
      :class="isMobile ? 'mobile-dialog' : ''"
    >
      <div v-if="selectedConfigs.length === 0" class="no-selection">
        <el-alert
          title="请先选择要操作的配置"
          type="warning"
          :closable="false"
          show-icon
        />
        <div style="margin-top: 20px;">
          <p>批量操作步骤：</p>
          <ol style="padding-left: 20px; line-height: 2;">
            <li>在表格中勾选要操作的支付配置</li>
            <li>点击批量操作按钮或使用下方操作按钮</li>
            <li>选择要执行的操作（启用/禁用/删除）</li>
          </ol>
        </div>
      </div>
      <div v-else>
        <el-alert
          :title="`已选择 ${selectedConfigs.length} 个配置`"
          type="info"
          :closable="false"
          show-icon
          style="margin-bottom: 20px;"
        />
        <div class="bulk-actions-list">
          <el-button 
            type="success" 
            @click="batchEnableConfigs" 
            :loading="batchOperating"
            style="width: 100%; margin-bottom: 10px;"
          >
            <el-icon><Check /></el-icon>
            批量启用 ({{ selectedConfigs.length }})
          </el-button>
          <el-button 
            type="warning" 
            @click="batchDisableConfigs" 
            :loading="batchOperating"
            style="width: 100%; margin-bottom: 10px;"
          >
            <el-icon><Close /></el-icon>
            批量禁用 ({{ selectedConfigs.length }})
          </el-button>
          <el-button 
            type="danger" 
            @click="batchDeleteConfigs" 
            :loading="batchOperating"
            style="width: 100%;"
          >
            <el-icon><Delete /></el-icon>
            批量删除 ({{ selectedConfigs.length }})
          </el-button>
        </div>
      </div>
      <template #footer>
        <el-button @click="showBulkOperationsDialog = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 配置统计对话框 -->
    <el-dialog
      v-model="showStatisticsDialog"
      title="配置统计"
      :width="isMobile ? '95%' : '600px'"
      :class="isMobile ? 'mobile-dialog' : ''"
    >
      <div v-if="statisticsLoading" style="text-align: center; padding: 40px;">
        <el-icon class="is-loading" style="font-size: 24px;"><Loading /></el-icon>
        <p style="margin-top: 10px;">加载中...</p>
      </div>
      <div v-else-if="statisticsData">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="总配置数">
            <el-tag type="info">{{ statisticsData.total_configs || 0 }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="已启用">
            <el-tag type="success">{{ statisticsData.active_configs || 0 }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="已禁用">
            <el-tag type="warning">{{ statisticsData.inactive_configs || 0 }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>
        
        <div style="margin-top: 20px;">
          <h4 style="margin-bottom: 15px;">按支付类型统计：</h4>
          <el-table :data="typeStatsList" style="width: 100%" size="small">
            <el-table-column prop="type" label="支付类型" width="150">
              <template #default="scope">
                <el-tag :type="getTypeTagType(scope.row.type)">
                  {{ getTypeText(scope.row.type) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="count" label="已启用数量" align="center">
              <template #default="scope">
                <el-tag type="success">{{ scope.row.count }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <div v-else style="text-align: center; padding: 40px;">
        <el-empty description="暂无统计数据" />
      </div>
      <template #footer>
        <el-button type="primary" @click="loadStatistics">刷新</el-button>
        <el-button @click="showStatisticsDialog = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Download, Operation, Plus, Edit, Delete, DataAnalysis, Check, Close, Loading } from '@element-plus/icons-vue'
import { paymentAPI } from '@/utils/api'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
dayjs.extend(timezone)

export default {
  name: 'AdminPaymentConfig',
  components: { Download, Operation, Plus, Edit, Delete, DataAnalysis, Check, Close, Loading },
  setup() {
    const loading = ref(false)
    const saving = ref(false)
    const paymentConfigs = ref([])
    const showAddDialog = ref(false)
    const showBulkOperationsDialog = ref(false)
    const showStatisticsDialog = ref(false)
    const editingConfig = ref(null)
    const isMobile = ref(false)
    const selectedConfigs = ref([])
    const batchOperating = ref(false)
    const statisticsLoading = ref(false)
    const statisticsData = ref(null)
    const tableRef = ref(null)

    const checkMobile = () => {
      isMobile.value = window.innerWidth <= 768
    }

    const configForm = reactive({
      pay_type: '',
      app_id: '',
      merchant_private_key: '',
      alipay_public_key: '',
      alipay_gateway: 'https://openapi.alipay.com/gateway.do',
      // 微信支付配置
      wechat_mch_id: '',
      wechat_api_key: '',
      // 易支付配置
      yipay_type: 'alipay',  // 支付类型：alipay 或 wxpay
      yipay_sign_type: 'RSA',  // 签名类型：RSA 或 MD5
      yipay_pid: '',
      yipay_private_key: '',
      yipay_public_key: '',
      yipay_gateway: 'https://pay.yi-zhifu.cn/',
      yipay_md5_key: '',
      // PayPal配置
      paypal_client_id: '',
      paypal_secret: '',
      paypal_mode: 'sandbox',  // sandbox 或 live
      // Stripe配置
      stripe_publishable_key: '',
      stripe_secret_key: '',
      stripe_webhook_secret: '',
      // 码支付配置
      codepay_id: '',
      codepay_token: '',
      codepay_gateway: 'https://api.xiuxiu888.com/creat_order',
      // 银行转账配置
      bank_name: '',
      bank_account: '',
      bank_branch: '',
      account_holder: '',
      return_url: '',
      notify_url: '',
      status: 1,
      sort_order: 0
    })

    const loadPaymentConfigs = async () => {
      loading.value = true
      try {
        // 使用管理员API获取支付配置列表
        const response = await paymentAPI.getPaymentConfigs({
          page: 1,
          size: 100  // 获取更多配置
        })
        // 处理响应数据
        let configList = []
        if (response && response.data) {
          // 处理标准响应格式 { success: true, data: { items: [...], total: ... } }
          if (response.data.success && response.data.data) {
            if (response.data.data.items && Array.isArray(response.data.data.items)) {
              configList = response.data.data.items
            } else if (Array.isArray(response.data.data)) {
              configList = response.data.data
            }
          } 
          // 处理直接返回 items 的格式 { items: [...], total: ... }
          else if (response.data.items && Array.isArray(response.data.items)) {
            configList = response.data.items
          } 
          // 处理直接返回数组的格式 [...]
          else if (Array.isArray(response.data)) {
            configList = response.data
          }
        }
        // 确保 status 是数字类型（1 或 0）
        paymentConfigs.value = configList.map(config => ({
          ...config,
          status: config.status === 1 || config.status === true || config.status === '1' ? 1 : 0
        }))
        
        // 检查是否有易支付配置
        const yipayConfig = paymentConfigs.value.find(c => c.pay_type === 'yipay_alipay' || c.pay_type === 'yipay_wxpay')
      } catch (error) {
        // 处理不同类型的错误
        let errorMessage = '加载支付配置列表失败'
        if (error.isNetworkError) {
          errorMessage = '网络连接失败，请检查网络连接'
        } else if (error.isTimeoutError) {
          errorMessage = '请求超时，请稍后重试'
        } else if (error.response?.data?.detail) {
          errorMessage = error.response.data.detail
        } else if (error.message) {
          errorMessage = error.message
        }
        ElMessage.error(errorMessage)
        paymentConfigs.value = []
      } finally {
        loading.value = false
      }
    }

    const saveConfig = async () => {
      saving.value = true
      try {
        // 构建请求数据
        const requestData = {
          pay_type: configForm.pay_type,
          status: configForm.status,
          return_url: configForm.return_url,
          notify_url: configForm.notify_url,
          sort_order: configForm.sort_order || 0
        }

        // 根据支付类型添加特定配置
        if (configForm.pay_type === 'alipay') {
          requestData.app_id = configForm.app_id
          requestData.merchant_private_key = configForm.merchant_private_key
          requestData.alipay_public_key = configForm.alipay_public_key
          // 将 gateway_url 保存到 config_json 中（统一使用 gateway_url 键名）
          requestData.config_json = {
            gateway_url: configForm.alipay_gateway || 'https://openapi.alipay.com/gateway.do'
          }
        } else if (configForm.pay_type === 'wechat') {
          requestData.app_id = configForm.app_id
          requestData.wechat_app_id = configForm.app_id
          requestData.wechat_mch_id = configForm.wechat_mch_id
          requestData.wechat_api_key = configForm.wechat_api_key
        } else if (configForm.pay_type === 'yipay_alipay' || configForm.pay_type === 'yipay_wxpay') {
          // 易支付配置保存到config_json
          // 根据 pay_type 确定 yipay_type（调用值）
          const yipay_type = configForm.pay_type === 'yipay_alipay' ? 'alipay' : 'wxpay'
          requestData.config_json = {
            yipay_type: yipay_type,  // 调用值：alipay 或 wxpay
            yipay_sign_type: configForm.yipay_sign_type || 'RSA',  // 签名类型：RSA 或 MD5
            yipay_pid: configForm.yipay_pid,
            yipay_private_key: configForm.yipay_private_key || '',
            yipay_public_key: configForm.yipay_public_key || '',
            yipay_gateway: configForm.yipay_gateway || 'https://pay.yi-zhifu.cn/',
            yipay_md5_key: configForm.yipay_md5_key || ''
          }
        } else if (configForm.pay_type === 'paypal') {
          requestData.paypal_client_id = configForm.paypal_client_id
          requestData.paypal_secret = configForm.paypal_secret
          requestData.paypal_mode = configForm.paypal_mode || 'sandbox'
        } else if (configForm.pay_type === 'stripe') {
          requestData.stripe_publishable_key = configForm.stripe_publishable_key
          requestData.stripe_secret_key = configForm.stripe_secret_key
          requestData.stripe_webhook_secret = configForm.stripe_webhook_secret || ''
        } else if (configForm.pay_type.startsWith('codepay_')) {
          // 码支付配置保存到config_json
          const codepay_type_map = {
            'codepay_alipay': '1',
            'codepay_wechat': '3',
            'codepay_qq': '2'
          }
          requestData.config_json = {
            codepay_type: codepay_type_map[configForm.pay_type] || '1',
            codepay_id: configForm.codepay_id,
            codepay_token: configForm.codepay_token,
            codepay_gateway: configForm.codepay_gateway || 'https://api.xiuxiu888.com/creat_order'
          }
        } else if (configForm.pay_type === 'bank_transfer') {
          // 银行转账配置保存到config_json
          requestData.config_json = {
            bank_name: configForm.bank_name,
            bank_account: configForm.bank_account,
            bank_branch: configForm.bank_branch || '',
            account_holder: configForm.account_holder
          }
        }

        if (editingConfig.value) {
          await paymentAPI.updatePaymentConfig(editingConfig.value.id, requestData)
          ElMessage.success('支付配置更新成功')
        } else {
          await paymentAPI.createPaymentConfig(requestData)
          ElMessage.success('支付配置创建成功')
        }

        showAddDialog.value = false
        resetConfigForm()
        loadPaymentConfigs()
      } catch (error) {
        // 处理不同类型的错误
        let errorMessage = '操作失败'
        if (error.isNetworkError) {
          errorMessage = '网络连接失败，请检查网络连接'
        } else if (error.isTimeoutError) {
          errorMessage = '请求超时，请稍后重试'
        } else if (error.response?.data?.detail) {
          errorMessage = error.response.data.detail
        } else if (error.message) {
          errorMessage = error.message
        }
        ElMessage.error(errorMessage)
      } finally {
        saving.value = false
      }
    }

    const editConfig = (config) => {
      editingConfig.value = config
      // 从config中提取配置信息
      const configData = config.config_json || {}
      Object.assign(configForm, {
        pay_type: config.pay_type || '',
        app_id: config.app_id || configData.app_id || '',
        merchant_private_key: config.merchant_private_key || configData.merchant_private_key || '',
        alipay_public_key: config.alipay_public_key || configData.alipay_public_key || '',
        alipay_gateway: configData.gateway_url || configData.alipay_gateway || config.gateway_url || config.alipay_gateway || 'https://openapi.alipay.com/gateway.do',
        // 微信支付配置
        wechat_mch_id: config.wechat_mch_id || configData.mch_id || '',
        wechat_api_key: config.wechat_api_key || configData.api_key || '',
        // 易支付配置
        yipay_type: configData.yipay_type || 'alipay',  // 支付类型
        yipay_sign_type: configData.yipay_sign_type || 'RSA',  // 签名类型
        yipay_pid: configData.yipay_pid || '',
        yipay_private_key: configData.yipay_private_key || '',
        yipay_public_key: configData.yipay_public_key || '',
        yipay_gateway: configData.yipay_gateway || 'https://pay.yi-zhifu.cn/',
        yipay_md5_key: configData.yipay_md5_key || '',
        // PayPal配置
        paypal_client_id: config.paypal_client_id || configData.paypal_client_id || configData.client_id || '',
        paypal_secret: config.paypal_secret || configData.paypal_secret || configData.secret || '',
        paypal_mode: config.paypal_mode || configData.paypal_mode || 'sandbox',
        // Stripe配置
        stripe_publishable_key: config.stripe_publishable_key || configData.stripe_publishable_key || configData.publishable_key || '',
        stripe_secret_key: config.stripe_secret_key || configData.stripe_secret_key || configData.secret_key || '',
        stripe_webhook_secret: config.stripe_webhook_secret || configData.stripe_webhook_secret || '',
        // 码支付配置
        codepay_id: configData.codepay_id || '',
        codepay_token: configData.codepay_token || '',
        codepay_gateway: configData.codepay_gateway || 'https://api.xiuxiu888.com/creat_order',
        // 银行转账配置
        bank_name: configData.bank_name || '',
        bank_account: configData.bank_account || '',
        bank_branch: configData.bank_branch || '',
        account_holder: configData.account_holder || '',
        return_url: config.return_url || '',
        notify_url: config.notify_url || '',
        status: config.status !== undefined ? config.status : 1,
        sort_order: config.sort_order || 0
      })
      showAddDialog.value = true
    }

    const deleteConfig = async (config) => {
      try {
        await ElMessageBox.confirm(
          `确定要删除支付配置 "${config.pay_type}" 吗？`,
          '确认删除',
          { type: 'warning' }
        )
        await paymentAPI.deletePaymentConfig(config.id)
        ElMessage.success('支付配置删除成功')
        loadPaymentConfigs()
      } catch (error) {
        if (error !== 'cancel') {
          // 处理不同类型的错误
          let errorMessage = '删除失败'
          if (error.isNetworkError) {
            errorMessage = '网络连接失败，请检查网络连接'
          } else if (error.isTimeoutError) {
            errorMessage = '请求超时，请稍后重试'
          } else if (error.response?.data?.detail) {
            errorMessage = error.response.data.detail
          } else if (error.message && !error.message.includes('cancel')) {
            errorMessage = error.message
          }
          ElMessage.error(errorMessage)
        }
      }
    }

    const toggleStatus = async (config, newValue) => {
      // newValue 是 switch 组件传递的新状态值（1 或 0）
      // 如果 newValue 未传递，则从 config.status 获取（已经被 switch 更新了）
      const newStatus = newValue !== undefined ? newValue : config.status
      const oldStatus = newStatus === 1 ? 0 : 1
      
      try {
        // 使用管理员API更新支付配置状态
        const response = await paymentAPI.updatePaymentConfig(config.id, { status: newStatus })
        
        // 如果响应成功，使用返回的数据更新状态
        if (response.data && response.data.status !== undefined) {
          config.status = response.data.status
        } else {
          // 如果响应没有返回状态，使用请求的状态
          config.status = newStatus
        }
        
        ElMessage.success(`支付配置${newStatus === 1 ? '启用' : '禁用'}成功`)
        // 重新加载配置列表以确保数据同步
        await loadPaymentConfigs()
      } catch (error) {
        // 恢复原状态
        config.status = oldStatus
        ElMessage.error('状态更新失败: ' + (error.response?.data?.detail || error.message || '未知错误'))
      }
    }

    const resetConfigForm = () => {
      Object.assign(configForm, {
        pay_type: '',
        app_id: '',
        merchant_private_key: '',
        alipay_public_key: '',
        alipay_gateway: 'https://openapi.alipay.com/gateway.do',
        // 微信支付配置
        wechat_mch_id: '',
        wechat_api_key: '',
        // 易支付配置
        yipay_type: 'alipay',  // 支付类型
        yipay_sign_type: 'RSA',  // 签名类型
        yipay_pid: '',
        yipay_private_key: '',
        yipay_public_key: '',
        yipay_gateway: 'https://pay.yi-zhifu.cn/',
        yipay_md5_key: '',
        // PayPal配置
        paypal_client_id: '',
        paypal_secret: '',
        paypal_mode: 'sandbox',
        // Stripe配置
        stripe_publishable_key: '',
        stripe_secret_key: '',
        stripe_webhook_secret: '',
        // 码支付配置
        codepay_id: '',
        codepay_token: '',
        codepay_gateway: 'https://api.xiuxiu888.com/creat_order',
        // 银行转账配置
        bank_name: '',
        bank_account: '',
        bank_branch: '',
        account_holder: '',
        return_url: '',
        notify_url: '',
        status: 1,
        sort_order: 0
      })
      editingConfig.value = null
    }

    const getTypeText = (type) => {
      const typeMap = {
        'alipay': '支付宝',
        'wechat': '微信支付',
        'yipay': '易支付',
        'yipay_alipay': '易支付-支付宝',
        'yipay_wxpay': '易支付-微信',
        'codepay_alipay': '码支付-支付宝',
        'codepay_wechat': '码支付-微信',
        'codepay_qq': '码支付-QQ钱包',
        'paypal': 'PayPal',
        'stripe': 'Stripe',
        'bank_transfer': '银行转账'
      }
      return typeMap[type] || type
    }

    const getTypeTagType = (type) => {
      const typeMap = {
        'alipay': 'success',
        'wechat': 'primary',
        'yipay': 'warning',
        'yipay_alipay': 'warning',
        'yipay_wxpay': 'warning',
        'codepay_alipay': 'success',
        'codepay_wechat': 'primary',
        'codepay_qq': 'info',
        'paypal': 'warning',
        'stripe': 'success',
        'bank_transfer': 'info'
      }
      return typeMap[type] || 'info'
    }

    const exportConfigs = async () => {
      try {
        const response = await paymentAPI.exportPaymentConfigs()
        // 检查响应是否为Blob（CSV文件）
        let blob = null
        if (response.data instanceof Blob) {
          blob = response.data
        } else if (response.data && typeof response.data === 'object' && response.data.data) {
          // axios可能包装了Blob
          blob = response.data.data
        }
        
        if (blob instanceof Blob) {
          // 从响应头获取文件名
          const contentDisposition = response.headers['content-disposition'] || response.headers['Content-Disposition']
          // 使用北京时间生成文件名
          const beijingDate = dayjs().tz('Asia/Shanghai')
          let filename = `payment_configs_export_${beijingDate.format('YYYYMMDD')}.csv`
          
          if (contentDisposition) {
            // 优先解析 RFC 5987 格式: filename*=UTF-8''filename
            let filenameMatch = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i)
            if (filenameMatch && filenameMatch.length > 1) {
              filename = decodeURIComponent(filenameMatch[1])
            } else {
              // 解析标准格式: filename="filename"
              filenameMatch = contentDisposition.match(/filename=['"]?([^'";]+)['"]?/i)
              if (filenameMatch && filenameMatch.length > 1) {
                filename = decodeURIComponent(filenameMatch[1])
              }
            }
          }
          
          // 创建下载链接
          const url = window.URL.createObjectURL(blob)
          const link = document.createElement('a')
          link.style.display = 'none'
          link.href = url
          link.download = filename
          
          // 添加到DOM并触发点击
          document.body.appendChild(link)
          // 使用requestAnimationFrame确保DOM已更新
          requestAnimationFrame(() => {
            link.click()
            // 延迟清理
            setTimeout(() => {
              document.body.removeChild(link)
              window.URL.revokeObjectURL(url)
            }, 1000)
          })
          
          ElMessage.success('支付配置导出成功，文件下载已开始')
        } else {
          // 如果不是Blob，可能是JSON错误响应
          ElMessage.error('导出失败：响应格式不正确')
        }
      } catch (error) {
        // 处理错误响应
        if (error.response) {
          // 如果是Blob类型的错误响应，尝试读取错误信息
          if (error.response.data instanceof Blob) {
            error.response.data.text().then(text => {
              try {
                const errorData = JSON.parse(text)
                ElMessage.error(errorData.message || errorData.detail || '导出失败')
              } catch (e) {
                ElMessage.error('导出失败：服务器返回错误')
              }
            }).catch(() => {
              ElMessage.error('导出失败：无法读取错误信息')
            })
          } else if (error.response.data?.message || error.response.data?.detail) {
            // 如果是JSON错误响应
            ElMessage.error(error.response.data.message || error.response.data.detail || '导出失败')
          } else {
            ElMessage.error(`导出失败：${error.response.status} ${error.response.statusText}`)
          }
        } else if (error.message) {
          ElMessage.error(`导出失败：${error.message}`)
        } else {
          ElMessage.error('导出失败：未知错误')
        }
      }
    }

    const handleSelectionChange = (selection) => {
      selectedConfigs.value = selection
    }

    const clearSelection = () => {
      selectedConfigs.value = []
      // 清除表格选择
      if (tableRef.value) {
        tableRef.value.clearSelection()
      }
    }

    const batchEnableConfigs = async () => {
      if (selectedConfigs.value.length === 0) {
        ElMessage.warning('请先选择要启用的配置')
        return
      }
      try {
        await ElMessageBox.confirm(
          `确定要启用 ${selectedConfigs.value.length} 个支付配置吗？`,
          '确认批量启用',
          { type: 'warning' }
        )
        batchOperating.value = true
        const configIds = selectedConfigs.value.map(c => c.id)
        await paymentAPI.bulkEnablePaymentConfigs(configIds)
        ElMessage.success(`成功启用 ${configIds.length} 个配置`)
        clearSelection()
        showBulkOperationsDialog.value = false
        loadPaymentConfigs()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('批量启用失败: ' + (error.response?.data?.detail || error.message))
        }
      } finally {
        batchOperating.value = false
      }
    }

    const batchDisableConfigs = async () => {
      if (selectedConfigs.value.length === 0) {
        ElMessage.warning('请先选择要禁用的配置')
        return
      }
      try {
        await ElMessageBox.confirm(
          `确定要禁用 ${selectedConfigs.value.length} 个支付配置吗？`,
          '确认批量禁用',
          { type: 'warning' }
        )
        batchOperating.value = true
        const configIds = selectedConfigs.value.map(c => c.id)
        await paymentAPI.bulkDisablePaymentConfigs(configIds)
        ElMessage.success(`成功禁用 ${configIds.length} 个配置`)
        clearSelection()
        showBulkOperationsDialog.value = false
        loadPaymentConfigs()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('批量禁用失败: ' + (error.response?.data?.detail || error.message))
        }
      } finally {
        batchOperating.value = false
      }
    }

    const batchDeleteConfigs = async () => {
      if (selectedConfigs.value.length === 0) {
        ElMessage.warning('请先选择要删除的配置')
        return
      }
      try {
        await ElMessageBox.confirm(
          `确定要删除 ${selectedConfigs.value.length} 个支付配置吗？此操作不可恢复！`,
          '确认批量删除',
          { type: 'error' }
        )
        batchOperating.value = true
        const configIds = selectedConfigs.value.map(c => c.id)
        await paymentAPI.bulkDeletePaymentConfigs(configIds)
        ElMessage.success(`成功删除 ${configIds.length} 个配置`)
        clearSelection()
        showBulkOperationsDialog.value = false
        loadPaymentConfigs()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('批量删除失败: ' + (error.response?.data?.detail || error.message))
        }
      } finally {
        batchOperating.value = false
      }
    }

    const loadStatistics = async () => {
      statisticsLoading.value = true
      try {
        const response = await paymentAPI.getPaymentConfigStats()
        if (response && response.data) {
          statisticsData.value = response.data
        } else {
          statisticsData.value = null
        }
      } catch (error) {
        // 处理不同类型的错误
        let errorMessage = '加载统计数据失败'
        if (error.isNetworkError) {
          errorMessage = '网络连接失败，请检查网络连接'
        } else if (error.isTimeoutError) {
          errorMessage = '请求超时，请稍后重试'
        } else if (error.response?.data?.detail) {
          errorMessage = error.response.data.detail
        } else if (error.message) {
          errorMessage = error.message
        }
        ElMessage.error(errorMessage)
        statisticsData.value = null
      } finally {
        statisticsLoading.value = false
      }
    }

    const typeStatsList = computed(() => {
      if (!statisticsData.value || !statisticsData.value.type_stats) {
        return []
      }
      return Object.entries(statisticsData.value.type_stats).map(([type, count]) => ({
        type,
        count
      }))
    })

    const handleMobileAction = (command) => {
      switch (command) {
        case 'export':
          exportConfigs()
          break
        case 'bulk':
          showBulkOperationsDialog.value = true
          break
        case 'statistics':
          showStatisticsDialog.value = true
          loadStatistics()
          break
      }
    }

    // 监听统计对话框打开
    const watchStatisticsDialog = () => {
      if (showStatisticsDialog.value && !statisticsData.value) {
        loadStatistics()
      }
    }

    onMounted(() => {
      checkMobile()
      window.addEventListener('resize', checkMobile)
      loadPaymentConfigs()
      // 监听统计对话框
      if (showStatisticsDialog.value) {
        loadStatistics()
      }
    })

    onUnmounted(() => {
      window.removeEventListener('resize', checkMobile)
    })

    const handleShowStatistics = () => {
      showStatisticsDialog.value = true
      loadStatistics()
    }

    return {
      loading,
      saving,
      paymentConfigs,
      showAddDialog,
      showBulkOperationsDialog,
      showStatisticsDialog,
      editingConfig,
      configForm,
      selectedConfigs,
      batchOperating,
      statisticsLoading,
      statisticsData,
      typeStatsList,
      loadPaymentConfigs,
      saveConfig,
      editConfig,
      deleteConfig,
      toggleStatus,
      resetConfigForm,
      getTypeText,
      getTypeTagType,
      exportConfigs,
      handleMobileAction,
      handleSelectionChange,
      clearSelection,
      batchEnableConfigs,
      batchDisableConfigs,
      batchDeleteConfigs,
      loadStatistics,
      handleShowStatistics,
      tableRef,
      isMobile
    }
  }
}
</script>

<style scoped>
.admin-payment-config {
  padding: 20px;
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

.text-muted {
  color: #909399;
  font-style: italic;
}

:deep(.el-table .el-table__row:hover) {
  background-color: #f5f7fa;
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

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.5;
}

/* 桌面端/移动端显示控制 */
.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}

.mobile-only {
  display: none;
  
  @media (max-width: 768px) {
    display: block;
  }
  
  &.mobile-card-list {
    @media (max-width: 768px) {
      display: flex;
      flex-direction: column;
    }
  }
  
  &.header-actions {
    @media (max-width: 768px) {
      display: flex;
      gap: 8px;
    }
  }
}

/* 移动端样式 */
@media (max-width: 768px) {
  .admin-payment-config {
    padding: 10px;
  }

  .header-content {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .header-actions {
    width: 100%;
    display: flex;
    flex-direction: row;
    gap: 8px;
    
    .el-button {
      flex: 1;
      height: 40px;
      font-size: 14px;
      font-weight: 500;
    }
  }

  /* 移动端卡片列表 */
  .mobile-card-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .mobile-card {
    background: #fff;
    border: 1px solid #e4e7ed;
    border-radius: 8px;
    padding: 16px;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  }

  .card-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 0;
    border-bottom: 1px solid #f0f0f0;
    
    &:last-of-type {
      border-bottom: none;
    }
    
    .label {
      font-weight: 600;
      color: #606266;
      font-size: 14px;
      min-width: 100px;
    }
    
    .value {
      flex: 1;
      text-align: right;
      color: #303133;
      font-size: 14px;
      word-break: break-all;
    }
  }

  .card-actions {
    display: flex;
    gap: 12px;
    margin-top: 12px;
    padding-top: 12px;
    border-top: 1px solid #f0f0f0;
    
    .el-button {
      flex: 1;
      height: 44px;
      font-size: 16px;
      font-weight: 500;
      
      :deep(.el-icon) {
        margin-right: 6px;
        font-size: 16px;
      }
    }
  }

  .empty-state {
    padding: 40px 20px;
    text-align: center;
  }

  /* 移动端对话框 */
  .mobile-dialog {
    :deep(.el-dialog) {
      width: 95% !important;
      margin: 5vh auto !important;
      max-height: 90vh;
      border-radius: 12px;
    }

    :deep(.el-dialog__header) {
      padding: 16px 20px;
      border-bottom: 1px solid #f0f0f0;
    }

    :deep(.el-dialog__title) {
      font-size: 18px;
      font-weight: 600;
    }

    :deep(.el-dialog__body) {
      max-height: calc(90vh - 140px);
      overflow-y: auto;
      padding: 20px;
      -webkit-overflow-scrolling: touch;
    }

    :deep(.el-form) {
      width: 100%;
    }

    :deep(.el-form-item) {
      margin-bottom: 20px;
      display: flex;
      flex-direction: column;
    }

    :deep(.el-form-item__label) {
      width: 100% !important;
      text-align: left !important;
      margin-bottom: 8px !important;
      padding: 0 !important;
      font-weight: 600;
      font-size: 14px;
      color: #303133;
      line-height: 1.5;
    }

    :deep(.el-form-item__content) {
      margin-left: 0 !important;
      width: 100%;
    }

    :deep(.el-input),
    :deep(.el-select),
    :deep(.el-textarea) {
      width: 100%;
    }

    :deep(.el-input__wrapper),
    :deep(.el-textarea__inner) {
      width: 100%;
      font-size: 16px; /* 防止iOS自动缩放 */
    }

    :deep(.el-select .el-input__wrapper) {
      width: 100%;
    }

    :deep(.el-textarea) {
      .el-textarea__inner {
        min-height: 100px;
        resize: vertical;
      }
    }

    .form-tip {
      font-size: 12px;
      color: #909399;
      margin-top: 6px;
      line-height: 1.6;
      padding: 0 4px;
    }

    .dialog-footer-buttons {
      display: flex;
      flex-direction: column;
      gap: 12px;
      width: 100%;
      padding-top: 16px;
      border-top: 1px solid #f0f0f0;
      
      .mobile-action-btn,
      .el-button {
        width: 100% !important;
        height: 48px !important;
        font-size: 16px !important;
        font-weight: 500 !important;
        margin: 0 !important;
        border-radius: 8px;
      }
    }

    /* 优化选项组在移动端的显示 */
    :deep(.el-select-dropdown) {
      .el-select-group__title {
        font-size: 13px;
        padding: 8px 12px;
      }
      
      .el-select-group .el-select-group__wrap {
        .el-select-dropdown__item {
          padding: 10px 20px;
          font-size: 14px;
        }
      }
    }
  }
}
</style>