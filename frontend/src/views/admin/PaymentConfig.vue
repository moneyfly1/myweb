<template>
  <div class="admin-payment-config">
    <el-card>
      <template #header>
        <div class="header-content">
          <span>支付配置管理</span>
          <div class="header-actions desktop-only">
            <el-button type="warning" @click="showBulkOperationsDialog = true">
              <el-icon><Operation /></el-icon>
              批量操作
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
          <template v-if="isMobile">
            <div class="mobile-label">支付类型</div>
          </template>
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
              <el-option label="易支付-支付宝（不推荐，仅兼容旧配置）" value="yipay_alipay" />
              <el-option label="易支付-微信（不推荐，仅兼容旧配置）" value="yipay_wxpay" />
              <el-option label="易支付-QQ钱包（不推荐，仅兼容旧配置）" value="yipay_qqpay" />
            </el-option-group>
          </el-select>
        </el-form-item>

        <el-form-item label="应用ID" v-if="configForm.pay_type === 'alipay' || configForm.pay_type === 'wechat'">
          <template v-if="isMobile">
            <div class="mobile-label">应用ID</div>
          </template>
          <el-input v-model="configForm.app_id" placeholder="请输入应用ID" style="width: 100%" />
          <div class="form-tip" v-if="configForm.pay_type === 'alipay'">
            <strong>⚠️ 重要提示：使用支付宝支付前必须完成以下步骤：</strong><br>
            <strong>第一步：签约产品（必须）</strong><br>
            1. 登录 <a href="https://open.alipay.com" target="_blank">支付宝开放平台</a><br>
            2. 进入"控制台" → "应用管理" → 选择您的应用<br>
            3. 在"产品列表"中，找到"当面付"产品<br>
            4. 点击"签约"，完成产品签约（可能需要企业认证）<br>
            5. 等待签约审核通过（通常需要1-3个工作日）<br><br>
            <strong>第二步：应用上线（必须）</strong><br>
            1. 在应用管理页面，确保应用状态为"已上线"<br>
            2. 如果应用状态是"开发中"，需要提交审核并上线<br>
            3. 只有已上线的应用才能使用支付接口<br><br>
            <strong>第三步：配置密钥（必须）</strong><br>
            1. 完成应用私钥和支付宝公钥的配置（见下方说明）<br>
            2. 确保应用公钥已上传到支付宝后台<br><br>
            <strong>第四步：配置回调地址（必须）</strong><br>
            1. 在下方"异步回调地址"中填写：<span style="color: #409EFF;">{{ baseUrl }}/api/v1/payment/notify/alipay</span><br>
            2. 在下方"同步回调地址"中填写：<span style="color: #409EFF;">{{ baseUrl }}/api/v1/payment/success</span><br>
            3. <strong>重要：</strong>登录支付宝开放平台 → 开发设置 → 应用网关，填写相同的异步回调地址<br>
            4. <strong>重要：</strong>登录支付宝开放平台 → 开发设置 → 收单异步通知，配置通知地址<br><br>
            <strong>⚠️ 关于权限不足（40006错误）：</strong><br>
            • <strong>权限不足的主要原因：</strong>未签约"当面付"产品或应用未上线<br>
            • <strong>回调地址未配置不会导致权限不足，但会导致无法接收支付回调</strong><br>
            • 必须先完成产品签约和应用上线，然后配置回调地址<br><br>
            <strong>常见错误：</strong><br>
            • <strong>40006 / insufficient-isv-permissions：</strong>未签约产品或应用未上线（与回调地址无关）<br>
            • <strong>40004：</strong>AppID和私钥不匹配，或应用公钥未正确配置<br>
            • <strong>40001：</strong>签名错误，检查私钥格式是否正确<br>
            • <strong>无法接收回调：</strong>应用网关未配置或回调地址不正确
          </div>
        </el-form-item>
        <!-- 易支付统一配置（yipay类型） -->
        <el-form-item label="商户ID" v-if="configForm.pay_type === 'yipay' || configForm.pay_type === 'yipay_alipay' || configForm.pay_type === 'yipay_wxpay' || configForm.pay_type === 'yipay_qqpay'">
          <template v-if="isMobile">
            <div class="mobile-label">商户ID</div>
          </template>
          <el-input v-model="configForm.app_id" placeholder="请输入易支付商户ID (pid)" style="width: 100%" />
          <div class="form-tip">在易支付商户后台->个人资料->API信息中查看商户ID (pid)</div>
        </el-form-item>

        <el-form-item label="商户密钥" v-if="configForm.pay_type === 'yipay' || configForm.pay_type === 'yipay_alipay' || configForm.pay_type === 'yipay_wxpay' || configForm.pay_type === 'yipay_qqpay'">
          <template v-if="isMobile">
            <div class="mobile-label">商户密钥</div>
          </template>
          <el-input 
            v-model="configForm.merchant_private_key" 
            type="password"
            show-password
            placeholder="请输入易支付商户密钥 (key)" 
            style="width: 100%"
          />
          <div class="form-tip">在易支付商户后台->个人资料->API信息中查看商户密钥 (key)，用于MD5签名</div>
        </el-form-item>

        <el-form-item label="网关地址" v-if="configForm.pay_type === 'yipay'">
          <template v-if="isMobile">
            <div class="mobile-label">网关地址</div>
          </template>
          <el-input v-model="configForm.yipay_gateway_url" placeholder="请输入易支付网关地址" style="width: 100%" />
          <div class="form-tip">填写易支付官网地址（系统会自动拼接API路径 /mapi.php）</div>
        </el-form-item>

        <el-form-item label="签名方式" v-if="configForm.pay_type === 'yipay'">
          <template v-if="isMobile">
            <div class="mobile-label">签名方式</div>
          </template>
          <el-select 
            v-model="configForm.yipay_sign_type" 
            placeholder="选择签名方式"
            style="width: 100%"
            :teleported="!isMobile"
          >
            <el-option label="MD5签名" value="MD5" />
            <el-option label="RSA签名" value="RSA" />
            <el-option label="MD5+RSA签名（推荐）" value="MD5+RSA" />
          </el-select>
          <div class="form-tip">选择签名方式：MD5（使用MD5密钥）、RSA（使用RSA密钥对）或MD5+RSA（推荐，更安全）</div>
        </el-form-item>

        <el-form-item label="平台公钥（RSA签名）" v-if="configForm.pay_type === 'yipay' && (configForm.yipay_sign_type === 'RSA' || configForm.yipay_sign_type === 'MD5+RSA')">
          <template v-if="isMobile">
            <div class="mobile-label">平台公钥</div>
          </template>
          <el-input
            v-model="configForm.yipay_platform_public_key"
            type="textarea"
            :rows="isMobile ? 6 : 4"
            placeholder="请输入易支付平台公钥（从易支付后台复制）"
            style="width: 100%"
          />
          <div class="form-tip">
            <strong>重要：</strong>这是易支付平台提供的公钥，用于验证易支付回调通知的签名。<br>
            在易支付后台->个人资料->API信息->RSA密钥->平台公钥中复制并粘贴到这里。
          </div>
        </el-form-item>

        <el-form-item label="商户私钥（RSA签名）" v-if="configForm.pay_type === 'yipay' && (configForm.yipay_sign_type === 'RSA' || configForm.yipay_sign_type === 'MD5+RSA')">
          <template v-if="isMobile">
            <div class="mobile-label">商户私钥</div>
          </template>
          <el-input
            v-model="configForm.yipay_merchant_private_key"
            type="textarea"
            :rows="isMobile ? 6 : 4"
            placeholder="请输入商户RSA私钥（您自己生成的）"
            style="width: 100%"
          />
          <div class="form-tip">
            <strong>重要：</strong>这是您自己生成的RSA私钥，用于签名发送给易支付的请求。<br>
            1. 使用OpenSSL或在线工具生成RSA密钥对<br>
            2. 将生成的公钥填写到易支付后台->个人资料->API信息->RSA密钥->商户公钥<br>
            3. 将生成的私钥填写到这里（请妥善保管，不要泄露）
          </div>
        </el-form-item>

        <el-form-item label="支持的支付方式" v-if="configForm.pay_type === 'yipay'">
          <template v-if="isMobile">
            <div class="mobile-label">支持的支付方式</div>
          </template>
          <el-checkbox-group v-model="configForm.yipay_supported_types">
            <el-checkbox label="alipay">支付宝</el-checkbox>
            <el-checkbox label="wxpay">微信支付</el-checkbox>
            <el-checkbox label="qqpay">QQ钱包</el-checkbox>
          </el-checkbox-group>
          <div class="form-tip">选择易支付平台支持哪些支付方式，客户可以在这些方式中选择</div>
        </el-form-item>

        <!-- 易支付独立类型配置（兼容旧配置，不推荐使用） -->
        <el-alert
          v-if="configForm.pay_type === 'yipay_alipay' || configForm.pay_type === 'yipay_wxpay' || configForm.pay_type === 'yipay_qqpay'"
          title="不推荐使用单独配置"
          type="warning"
          :closable="false"
          show-icon
          style="margin-bottom: 20px;"
        >
          <template #default>
            <div>
              <p><strong>建议使用"易支付（统一配置）"：</strong></p>
              <ul style="margin: 10px 0; padding-left: 20px;">
                <li>统一配置可以同时支持支付宝、微信、QQ钱包等多种支付方式</li>
                <li>配置更简单，只需配置一次即可使用所有支付方式</li>
                <li>管理更方便，无需为每种支付方式单独配置</li>
              </ul>
              <p style="margin-top: 10px; color: #909399;">
                单独配置类型仅用于兼容旧配置，新配置请使用"易支付（统一配置）"
              </p>
            </div>
          </template>
        </el-alert>
        
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

        <el-form-item label="网关地址" v-if="configForm.pay_type === 'yipay' || configForm.pay_type === 'yipay_alipay' || configForm.pay_type === 'yipay_wxpay'">
          <el-input 
            v-model="configForm.yipay_gateway_url" 
            placeholder="请输入易支付网关地址，例如：https://fhymw.com 或 https://pay.yi-zhifu.cn" 
            style="width: 100%"
            @blur="detectYipayPlatform"
          />
          <div class="form-tip">
            <div>系统会自动识别易支付平台类型，支持的平台包括：</div>
            <div style="margin-top: 5px;">
              • fhymw.com • yi-zhifu.cn • ezfp.cn • myzfw.com • 8-pay.cn • epay.hehanwang.com • wx8g.com
            </div>
            <div style="margin-top: 5px; color: #67C23A;">
              ✓ 系统会自动根据网关地址识别平台，无需手动选择
            </div>
            <div style="margin-top: 5px;">
              系统会自动拼接API路径：/mapi.php（API接口支付）或 /submit.php（页面跳转支付）
            </div>
          </div>
        </el-form-item>
        <el-form-item label="支付宝公钥" v-if="configForm.pay_type === 'alipay'">
          <el-input
            v-model="configForm.alipay_public_key"
            type="textarea"
            :rows="isMobile ? 8 : 6"
            placeholder="请输入支付宝公钥（支持以下格式）：&#10;1. 完整PEM格式（推荐）：&#10;-----BEGIN PUBLIC KEY-----&#10;MIGfMA0GCSqGSIb3...&#10;-----END PUBLIC KEY-----&#10;&#10;2. 仅公钥内容（系统会自动格式化）：&#10;MIGfMA0GCSqGSIb3..."
            style="width: 100%"
          />
          <div class="form-tip">
            <strong>支付宝公钥获取步骤：</strong><br>
            1. 登录支付宝开放平台：<a href="https://open.alipay.com" target="_blank">https://open.alipay.com</a><br>
            2. 进入"控制台" → "应用管理" → 选择您的应用<br>
            3. 在"接口加签方式"中，点击"查看"或"下载"<br>
            4. 复制"支付宝公钥"（不是应用公钥！）<br>
            5. 粘贴到此处（支持完整PEM格式或仅公钥内容）<br>
            <strong>注意：</strong>这是支付宝提供的公钥，用于验证支付宝回调签名，不是您自己生成的应用公钥
          </div>
        </el-form-item>

        <el-form-item label="商户私钥" v-if="configForm.pay_type === 'alipay'">
          <el-input
            v-model="configForm.merchant_private_key"
            type="textarea"
            :rows="isMobile ? 8 : 6"
            placeholder="请输入商户私钥（支持以下格式）：&#10;1. 完整PEM格式（推荐）：&#10;-----BEGIN RSA PRIVATE KEY-----&#10;MIIEpAIBAAKCAQEA...&#10;-----END RSA PRIVATE KEY-----&#10;&#10;2. 仅私钥内容（系统会自动格式化）：&#10;MIIEpAIBAAKCAQEA..."
            style="width: 100%"
          />
          <div class="form-tip">
            <strong>应用私钥获取完整步骤：</strong><br>
            <strong>第一步：生成密钥对</strong><br>
            1. 下载"支付宝开发平台开发助手"：<a href="https://opendocs.alipay.com/common/02kkv7" target="_blank">下载链接</a><br>
            2. 打开开发助手，选择"RSA2(SHA256)密钥"<br>
            3. 点击"生成密钥"，生成密钥对（私钥和公钥）<br>
            4. <strong>保存私钥文件</strong>（通常是 rsa_private_key.pem 或 rsa_private_key_pkcs8.pem）<br><br>
            <strong>第二步：上传应用公钥到支付宝</strong><br>
            1. 登录支付宝开放平台：<a href="https://open.alipay.com" target="_blank">https://open.alipay.com</a><br>
            2. 进入"控制台" → "应用管理" → 选择您的应用<br>
            3. 在"接口加签方式"中，点击"设置"<br>
            4. 选择"公钥"模式，粘贴"应用公钥"（从开发助手复制的公钥）<br>
            5. 保存设置<br><br>
            <strong>第三步：配置私钥到系统</strong><br>
            1. 打开保存的私钥文件（.pem文件）<br>
            2. 复制完整内容（包括 BEGIN 和 END 标记）<br>
            3. 粘贴到此处（支持以下格式）：<br>
            &nbsp;&nbsp;• <strong>完整PEM格式（推荐）：</strong>包含 BEGIN 和 END 标记<br>
            &nbsp;&nbsp;• <strong>简化格式：</strong>仅私钥内容（系统会自动格式化）<br>
            &nbsp;&nbsp;• <strong>支持格式：</strong>PKCS1 或 PKCS8 格式的 RSA2 私钥（2048位）<br><br>
            <strong>重要提示：</strong><br>
            • 应用私钥：您自己生成的私钥，用于签名请求（配置在此处）<br>
            • 应用公钥：从私钥生成的公钥，需要上传到支付宝后台<br>
            • 支付宝公钥：支付宝提供的公钥，用于验证回调（配置在"支付宝公钥"字段）
          </div>
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

        <el-form-item label="同步回调地址" v-if="configForm.pay_type === 'alipay'">
          <el-input v-model="configForm.return_url" placeholder="请输入同步回调地址" style="width: 100%" />
          <div class="form-tip">
            <strong>用途：</strong>支付完成后，用户浏览器跳转的地址（用于显示支付结果页面）<br>
            <strong>填写示例：</strong><span style="color: #409EFF;">{{ baseUrl }}/api/v1/payment/success</span><br>
            <strong>注意：</strong>必须是公网可访问的HTTPS地址（生产环境）或HTTP地址（沙箱环境）
          </div>
        </el-form-item>

        <el-form-item label="异步回调地址" v-if="configForm.pay_type === 'alipay'">
          <el-input v-model="configForm.notify_url" placeholder="请输入异步回调地址" style="width: 100%" />
          <div class="form-tip">
            <strong>用途：</strong>支付完成后，支付宝服务器主动通知您的服务器的地址（用于更新订单状态）<br>
            <strong>填写示例：</strong><span style="color: #409EFF;">{{ baseUrl }}/api/v1/payment/notify/alipay</span><br>
            <strong>注意：</strong>必须是公网可访问的HTTPS地址（生产环境）或HTTP地址（沙箱环境）<br>
            <strong>⚠️ 重要：</strong>此地址需要同时在支付宝开放平台的"开发设置" → "应用网关"中配置，否则无法接收回调通知
          </div>
        </el-form-item>

        <el-form-item label="同步回调地址" v-if="configForm.pay_type !== 'alipay'">
          <el-input v-model="configForm.return_url" placeholder="请输入同步回调地址" style="width: 100%" />
          <div class="form-tip">支付完成后跳转的地址</div>
        </el-form-item>

        <el-form-item label="异步回调地址" v-if="configForm.pay_type !== 'alipay'">
          <el-input v-model="configForm.notify_url" placeholder="请输入异步回调地址" style="width: 100%" />
          <div class="form-tip" v-if="configForm.pay_type === 'yipay' || configForm.pay_type === 'yipay_alipay' || configForm.pay_type === 'yipay_wxpay' || configForm.pay_type === 'yipay_qqpay'">
            <strong>易支付回调地址配置：</strong><br>
            <strong>用途：</strong>支付完成后，易支付服务器主动通知您的服务器的地址（用于更新订单状态和开通套餐）<br>
            <strong>填写示例：</strong><span style="color: #409EFF;">{{ baseUrl }}/api/v1/payment/notify/yipay</span><br>
            <strong>⚠️ 重要：</strong><br>
            1. 必须是公网可访问的HTTPS地址（生产环境）<br>
            2. 如果留空，系统会根据"系统设置"中的域名自动生成<br>
            3. 建议手动填写，确保回调地址正确（格式：https://您的域名/api/v1/payment/notify/yipay）<br>
            4. 此地址需要在易支付商户后台的"接口设置"中配置，否则无法接收回调通知
          </div>
          <div class="form-tip" v-else>
            支付完成后服务器通知的地址
          </div>
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
        <div class="dialog-footer-buttons" :class="{ 'mobile-footer': isMobile }">
          <el-button @click="showAddDialog = false" :class="{ 'mobile-action-btn': isMobile }">取消</el-button>
          <el-button type="primary" @click="saveConfig" :loading="saving" :class="{ 'mobile-action-btn': isMobile }">
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
      // 易支付配置（统一类型yipay）
      yipay_gateway_url: '',
      yipay_sign_type: 'MD5',
      yipay_platform_public_key: '', // 平台公钥（易支付提供）
      yipay_merchant_private_key: '', // 商户私钥（自己生成）
      yipay_supported_types: ['alipay', 'wxpay'], // 支持的支付方式列表
      // 易支付配置（独立类型，兼容旧配置）
      yipay_type: 'alipay',  // 支付类型：alipay 或 wxpay
      yipay_sign_type: 'RSA',  // 签名类型：RSA 或 MD5
      yipay_pid: '',
      yipay_private_key: '',
      yipay_public_key: '',
      yipay_gateway: 'https://pay.yi-zhifu.cn/',
      yipay_md5_key: '',
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
        // 确保 status 是数字类型（1 或 0），并处理 sql.NullString 格式的字段
        paymentConfigs.value = configList.map(config => {
          // 处理 sql.NullString 格式的字段（可能是 {String: "...", Valid: true} 或直接字符串）
          const extractValue = (value) => {
            if (value === null || value === undefined) return ''
            if (typeof value === 'string') return value
            if (typeof value === 'object' && value.String !== undefined) {
              return value.Valid ? value.String : ''
            }
            return String(value)
          }

          return {
            ...config,
            status: config.status === 1 || config.status === true || config.status === '1' ? 1 : 0,
            // 提取 sql.NullString 字段的值
            app_id: extractValue(config.app_id),
            merchant_private_key: extractValue(config.merchant_private_key),
            alipay_public_key: extractValue(config.alipay_public_key),
            wechat_app_id: extractValue(config.wechat_app_id),
            wechat_mch_id: extractValue(config.wechat_mch_id),
            wechat_api_key: extractValue(config.wechat_api_key),
            account_number: extractValue(config.account_number),
            wallet_address: extractValue(config.wallet_address),
            return_url: extractValue(config.return_url),
            notify_url: extractValue(config.notify_url),
            // 解析 config_json（可能是字符串或对象）
            config_json: typeof config.config_json === 'string' 
              ? (config.config_json ? JSON.parse(config.config_json) : {})
              : (config.config_json || {})
          }
        })
        
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

    const detectYipayPlatform = () => {
      if (!configForm.yipay_gateway_url) return
      
      const gatewayUrl = configForm.yipay_gateway_url.trim().toLowerCase()
      const knownPlatforms = [
        { domain: 'fhymw.com', name: 'FH易支付' },
        { domain: 'yi-zhifu.cn', name: '易支付官方' },
        { domain: 'ezfp.cn', name: 'EZ易支付' },
        { domain: 'myzfw.com', name: 'MY易支付' },
        { domain: '8-pay.cn', name: '8-Pay' },
        { domain: 'epay.hehanwang.com', name: '易支付' },
        { domain: 'wx8g.com', name: '易支付' }
      ]
      
      for (const platform of knownPlatforms) {
        if (gatewayUrl.includes(platform.domain)) {
          console.log(`检测到易支付平台: ${platform.name} (${platform.domain})`)
          return
        }
      }
      
      console.log('未识别的易支付平台，将使用标准适配器')
    }

    const saveConfig = async () => {
      saving.value = true
      try {
        // 构建请求数据
        const requestData = {
          pay_type: configForm.pay_type,
          status: configForm.status,
          return_url: configForm.return_url || '',
          notify_url: configForm.notify_url || '',
          sort_order: configForm.sort_order || 0
        }

        // 根据支付类型添加特定配置
        if (configForm.pay_type === 'alipay') {
          // 确保所有字段都被发送，即使为空字符串（后端需要指针类型才能更新）
          requestData.app_id = configForm.app_id !== undefined ? configForm.app_id : ''
          requestData.merchant_private_key = configForm.merchant_private_key !== undefined ? configForm.merchant_private_key : ''
          requestData.alipay_public_key = configForm.alipay_public_key !== undefined ? configForm.alipay_public_key : ''
          // 将 gateway_url 保存到 config_json 中（统一使用 gateway_url 键名）
          // 同时保存 is_production 标志（根据网关地址判断）
          const gateway = configForm.alipay_gateway || 'https://openapi.alipay.com/gateway.do'
          const isProduction = !gateway.includes('alipaydev.com')
          requestData.config_json = {
            gateway_url: gateway,
            is_production: isProduction
          }
        } else if (configForm.pay_type === 'wechat') {
          requestData.app_id = configForm.app_id
          requestData.wechat_app_id = configForm.app_id
          requestData.wechat_mch_id = configForm.wechat_mch_id
          requestData.wechat_api_key = configForm.wechat_api_key
        } else if (configForm.pay_type === 'yipay') {
          // 易支付统一配置
          requestData.app_id = configForm.app_id || '' // 商户ID (pid)
          // 根据签名方式存储不同的密钥
          if (configForm.yipay_sign_type === 'MD5') {
            requestData.merchant_private_key = configForm.merchant_private_key || '' // MD5密钥 (key)
          } else if (configForm.yipay_sign_type === 'RSA' || configForm.yipay_sign_type === 'MD5+RSA') {
            // RSA签名时，MD5密钥仍然需要（用于MD5+RSA方式）
            requestData.merchant_private_key = configForm.merchant_private_key || ''
            // 平台公钥存储在AlipayPublicKey字段（用于验证回调）
            requestData.alipay_public_key = configForm.yipay_platform_public_key || ''
          }
          if (!configForm.yipay_gateway_url || !configForm.yipay_gateway_url.trim()) {
            ElMessage.error('请填写易支付网关地址')
            saving.value = false
            return
          }
          const gatewayUrl = configForm.yipay_gateway_url.trim().replace(/\/$/, '')
          
          detectYipayPlatform()
          
          requestData.config_json = {
            gateway_url: gatewayUrl,
            api_url: `${gatewayUrl}/mapi.php`,
            sign_type: configForm.yipay_sign_type || 'MD5',
            platform_public_key: configForm.yipay_platform_public_key || '',
            merchant_private_key: configForm.yipay_merchant_private_key || '',
            supported_types: configForm.yipay_supported_types || ['alipay', 'wxpay']
          }
          
          console.log('保存易支付配置:', JSON.stringify(requestData.config_json, null, 2))
        } else if (configForm.pay_type === 'yipay_alipay' || configForm.pay_type === 'yipay_wxpay') {
          // 易支付配置保存到config_json（兼容旧配置）
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
        }

        if (editingConfig.value) {
          console.log('更新支付配置:', { id: editingConfig.value.id, requestData })
          await paymentAPI.updatePaymentConfig(editingConfig.value.id, requestData)
          ElMessage.success('支付配置更新成功')
        } else {
          console.log('创建支付配置:', requestData)
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
        wechat_mch_id: config.wechat_mch_id || '',
        wechat_api_key: config.wechat_api_key || '',
        // 易支付配置（统一类型yipay）
        yipay_gateway_url: configData.gateway_url || (configData.api_url ? configData.api_url.replace('/mapi.php', '').replace('/openapi/pay/create', '').replace(/\/$/, '') : ''),
        yipay_sign_type: configData.sign_type || 'MD5',
        yipay_platform_public_key: configData.platform_public_key || config.alipay_public_key || '',
        yipay_merchant_private_key: configData.merchant_private_key || '',
        yipay_supported_types: configData.supported_types || ['alipay', 'wxpay'],
        // 易支付配置（独立类型，兼容旧配置）
        yipay_type: configData.yipay_type || 'alipay',
        yipay_pid: configData.yipay_pid || '',
        yipay_private_key: configData.yipay_private_key || '',
        yipay_public_key: configData.yipay_public_key || '',
        yipay_gateway: configData.yipay_gateway || 'https://pay.yi-zhifu.cn/',
        yipay_md5_key: configData.yipay_md5_key || '',
        return_url: config.return_url || '',
        notify_url: config.notify_url || '',
        status: config.status !== undefined ? config.status : 1,
        sort_order: config.sort_order || 0
      })
      
      // 确保所有字段都有值（避免 undefined）
      Object.keys(configForm).forEach(key => {
        if (configForm[key] === undefined) {
          configForm[key] = ''
        }
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
        // 易支付配置（统一类型yipay）
        yipay_gateway_url: '',
        yipay_sign_type: 'MD5',
        yipay_platform_public_key: '',
        yipay_merchant_private_key: '',
        yipay_supported_types: ['alipay', 'wxpay'],
        // 易支付配置（独立类型，兼容旧配置）
        yipay_type: 'alipay',
        yipay_pid: '',
        yipay_private_key: '',
        yipay_public_key: '',
        yipay_gateway: 'https://pay.yi-zhifu.cn/',
        yipay_md5_key: '',
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
        'yipay_qqpay': '易支付-QQ钱包'
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
        'yipay_qqpay': 'warning'
      }
      return typeMap[type] || 'info'
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

    // 获取当前网站的基础URL
    const baseUrl = computed(() => {
      if (typeof window !== 'undefined') {
        return window.location.origin
      }
      return 'https://your-domain.com' // 默认值，实际不会使用
    })

    const handleMobileAction = (command) => {
      switch (command) {
        case 'bulk':
          showBulkOperationsDialog.value = true
          break
      }
    }

    onMounted(() => {
      checkMobile()
      window.addEventListener('resize', checkMobile)
      loadPaymentConfigs()
    })

    onUnmounted(() => {
      window.removeEventListener('resize', checkMobile)
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
      loadPaymentConfigs,
      saveConfig,
      editConfig,
      deleteConfig,
      toggleStatus,
      resetConfigForm,
      getTypeText,
      getTypeTagType,
      handleMobileAction,
      handleSelectionChange,
      clearSelection,
      batchEnableConfigs,
      batchDisableConfigs,
      batchDeleteConfigs,
      detectYipayPlatform,
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
      margin: 2vh auto !important;
      max-height: 96vh;
      border-radius: 8px;
      display: flex;
      flex-direction: column;
    }

    :deep(.el-dialog__header) {
      padding: 15px 15px 10px;
      flex-shrink: 0;
      border-bottom: 1px solid #ebeef5;
      
      .el-dialog__title {
        font-size: 18px;
        font-weight: 600;
      }
      
      .el-dialog__headerbtn {
        top: 8px;
        right: 8px;
        width: 32px;
        height: 32px;
        
        .el-dialog__close {
          font-size: 18px;
        }
      }
    }

    :deep(.el-dialog__body) {
      padding: 15px !important;
      flex: 1;
      overflow-y: auto;
      -webkit-overflow-scrolling: touch;
      max-height: calc(96vh - 140px);
    }
    
    :deep(.el-dialog__footer) {
      padding: 10px 15px 15px;
      flex-shrink: 0;
      border-top: 1px solid #ebeef5;
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
      display: none; /* 移动端隐藏默认标签 */
    }
    
    .mobile-label {
      font-size: 14px;
      font-weight: 600;
      color: #606266;
      margin-bottom: 8px;
      display: block;
      
      .required {
        color: #f56c6c;
        margin-left: 2px;
      }
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
      justify-content: flex-end;
      gap: 10px;
      
      &.mobile-footer {
        flex-direction: column;
        gap: 10px;
        
        .mobile-action-btn,
        .el-button {
          width: 100% !important;
          min-height: 48px !important;
          font-size: 16px !important;
          font-weight: 500 !important;
          margin: 0 !important;
          border-radius: 8px;
          -webkit-tap-highlight-color: rgba(0,0,0,0.1);
        }
      }
      
      .mobile-action-btn,
      .el-button {
        width: 100% !important;
        min-height: 48px !important;
        font-size: 16px !important;
        font-weight: 500 !important;
        margin: 0 !important;
        border-radius: 8px;
        -webkit-tap-highlight-color: rgba(0,0,0,0.1);
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