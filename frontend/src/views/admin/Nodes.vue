<template>
  <div class="list-container admin-nodes">
    <el-card class="list-card" shadow="never">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <span class="title-text">节点管理</span>
            <el-tag v-if="pagination.total" type="info" round size="small" class="count-tag">{{ pagination.total }}</el-tag>
          </div>
          <div class="header-actions" v-if="!isMobile">
            <el-button type="primary" @click="handleAdd">
              <el-icon><Plus /></el-icon>添加节点
            </el-button>
            <el-button type="warning" plain @click="openSelfHostDrawer">
              <el-icon><Promotion /></el-icon>自建节点
            </el-button>
            <el-button type="info" plain @click="openSelfHostListView">
              <el-icon><Monitor /></el-icon>自建节点列表
            </el-button>
            <el-button type="success" @click="batchTest" :loading="testing" :disabled="!selectedNodes.length">
              <el-icon><Connection /></el-icon>批量测试
            </el-button>
            <el-button type="danger" @click="batchDelete" :loading="deleting" :disabled="!selectedNodes.length">
              <el-icon><Delete /></el-icon>批量删除
            </el-button>
            <el-button @click="loadNodes" :loading="loading">
              <el-icon><Refresh /></el-icon>刷新
            </el-button>
          </div>
          <div class="header-actions mobile" v-else>
            <el-button type="primary" circle @click="handleAdd" size="small">
              <el-icon><Plus /></el-icon>
            </el-button>
            <el-button type="warning" plain circle @click="openSelfHostDrawer" size="small" title="自建节点">
              <el-icon><Promotion /></el-icon>
            </el-button>
            <el-dropdown trigger="click" @command="handleCommand">
              <el-button circle size="small">
                <el-icon><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="refresh" :icon="Refresh">刷新列表</el-dropdown-item>
                  <el-dropdown-item command="selfhost" :icon="Promotion">自建节点</el-dropdown-item>
                  <el-dropdown-item command="selfhost-list" :icon="Monitor">自建节点列表</el-dropdown-item>
                  <el-dropdown-item command="test" :icon="Connection" :disabled="!selectedNodes.length">批量测试</el-dropdown-item>
                  <el-dropdown-item command="delete" :icon="Delete" :disabled="!selectedNodes.length" divided class="danger-menu-item">批量删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </template>
      <div class="filter-wrapper">
        <div class="filter-grid">
          <el-select v-model="filters.status" placeholder="状态" clearable @change="applyNodeFilters">
            <el-option label="全部状态" value="" />
            <el-option label="在线" value="online" />
            <el-option label="离线" value="offline" />
            <el-option label="超时" value="timeout" />
          </el-select>
          <el-select v-model="filters.is_active" placeholder="激活" clearable @change="applyNodeFilters">
            <el-option label="全部" value="" />
            <el-option label="已激活" value="true" />
            <el-option label="已禁用" value="false" />
          </el-select>
          <el-select v-model="filters.is_manual" placeholder="来源" clearable @change="applyNodeFilters">
            <el-option label="所有来源" value="" />
            <el-option label="手动添加" value="true" />
            <el-option label="自动采集" value="false" />
          </el-select>
          <el-select v-model="filters.region" placeholder="地区" clearable @change="applyNodeFilters">
            <el-option label="所有地区" value="" />
            <el-option v-for="r in regions" :key="r" :label="r" :value="r" />
          </el-select>
          <el-select v-model="filters.type" placeholder="类型" clearable @change="applyNodeFilters">
            <el-option label="所有类型" value="" />
            <el-option v-for="t in allNodeTypes" :key="t" :label="t" :value="t" />
          </el-select>
          <div class="search-box">
            <el-input
              v-model="searchKeyword"
              placeholder="搜索节点名称、服务器地址或域名..."
              clearable
              @input="debouncedApplyNodeFilters"
              @clear="applyNodeFilters"
              @keyup.enter="applyNodeFilters"
            >
              <template #prefix><el-icon><Search /></el-icon></template>
            </el-input>
          </div>
          <div class="filter-actions">
            <el-button type="primary" @click="applyNodeFilters">
              <el-icon><Search /></el-icon>
              搜索
            </el-button>
            <el-button @click="resetNodeFilters">
              <el-icon><Refresh /></el-icon>
              重置
            </el-button>
          </div>
        </div>
      </div>
      <div class="content-view" v-loading="loading">
        <div class="mobile-selection-bar" v-if="isMobile && nodes.length > 0">
          <el-checkbox
            v-model="isAllSelected"
            :indeterminate="isIndeterminate"
            @change="toggleMobileSelectAll"
          >全选 ({{ selectedNodes.length }})</el-checkbox>
        </div>
        <ResponsiveDataView
          :data="nodes"
          :fields="mobileNodeFields"
          :loading="loading"
          title-field="name"
          empty-title="暂无节点数据"
          empty-description="添加节点或调整筛选条件后可在这里查看节点列表"
        >
          <template #table>
            <el-table
              :data="nodes"
              stripe
              border
              @selection-change="handleSelectionChange"
              class="desktop-table"
              ref="tableRef"
            >
              <el-table-column type="selection" width="50" />
              <el-table-column prop="name" label="节点名称" min-width="180" show-overflow-tooltip />
              <el-table-column prop="region" label="地区" width="100" />
              <el-table-column label="来源" width="90">
                <template #default="{ row }">
                  <el-tag v-if="row.self_hosted" type="danger" size="small" effect="light">自建</el-tag>
                  <el-tag v-else :type="row.is_manual ? 'warning' : 'success'" size="small" effect="light">{{ row.is_manual ? '手动' : '采集' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="订阅#" width="70">
                <template #default="{ row }">
                  <span v-if="!row.is_manual && row.source_index">#{{ row.source_index }}</span>
                  <span v-else class="text-placeholder">-</span>
                </template>
              </el-table-column>
              <el-table-column prop="type" label="类型" width="90">
                <template #default="{ row }">
                  <el-tag effect="plain" size="small">{{ row.type?.toUpperCase() }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="100">
                <template #default="{ row }">
                  <el-badge :is-dot="true" :type="getStatusType(row.status)" class="status-badge">
                    <span>{{ getStatusText(row.status) }}</span>
                  </el-badge>
                </template>
              </el-table-column>
              <el-table-column label="激活" width="80">
                <template #default="{ row }">
                  <el-switch v-model="row.is_active" @change="toggleNodeStatus(row)" size="small" />
                </template>
              </el-table-column>
              <el-table-column label="延迟" width="100">
                <template #default="{ row }">
                  <span :class="getLatencyClass(row.latency)">{{ formatLatency(row.latency) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="180" fixed="right">
                <template #default="{ row }">
                  <el-button-group>
                    <el-button size="small" @click="testNode(row)" :loading="row.testing" :icon="Connection" title="测试" />
                    <el-button size="small" type="primary" @click="editNode(row)" :icon="Edit" title="编辑" />
                    <el-button size="small" type="danger" @click="deleteNode(row)" :icon="Delete" title="删除" />
                  </el-button-group>
                </template>
              </el-table-column>
            </el-table>
            <EmptyState
              v-if="!loading && nodes.length === 0"
              class="desktop-empty-state"
              title="暂无节点数据"
              description="添加节点或调整筛选条件后可在这里查看节点列表"
            />
          </template>
          <template #header="{ item }">
            <div class="mobile-node-header">
              <el-checkbox
                :model-value="isSelected(item)"
                @change="(val) => handleMobileSelect(item, val)"
                class="mobile-node-checkbox"
              />
              <div class="mobile-node-title" :title="item.name">{{ item.name || '-' }}</div>
              <el-tag size="small" :type="getStatusType(item.status)" effect="light">
                {{ getStatusText(item.status) }}
              </el-tag>
            </div>
          </template>
          <template #field-type="{ item }">
            <el-tag effect="plain" size="small">{{ item.type?.toUpperCase() || '-' }}</el-tag>
          </template>
          <template #field-source="{ item }">
            <el-tag v-if="item.self_hosted" type="danger" size="small" effect="light">自建</el-tag>
            <el-tag v-else :type="item.is_manual ? 'warning' : 'success'" size="small" effect="light">
              {{ item.is_manual ? '手动' : '采集' }}
              <span v-if="!item.is_manual && item.source_index"> #{{ item.source_index }}</span>
            </el-tag>
          </template>
          <template #field-latency="{ item }">
            <span :class="getLatencyClass(item.latency)">{{ formatLatency(item.latency) }}</span>
          </template>
          <template #actions="{ item }">
            <div class="mobile-node-actions">
              <el-switch
                v-model="item.is_active"
                @change="toggleNodeStatus(item)"
                size="small"
                inline-prompt
                active-text="开"
                inactive-text="关"
              />
              <div class="mobile-node-buttons">
                <el-button size="small" text bg @click="testNode(item)" :loading="item.testing">测试</el-button>
                <el-button size="small" text bg type="primary" @click="editNode(item)">编辑</el-button>
                <el-button size="small" text bg type="danger" @click="deleteNode(item)">删除</el-button>
              </div>
            </div>
          </template>
          <template #empty>
            <EmptyState
              title="暂无节点数据"
              description="添加节点或调整筛选条件后可在这里查看节点列表"
            />
          </template>
        </ResponsiveDataView>
      </div>
      <div class="pagination-wrapper">
        <PaginationBar
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.size"
          :total="pagination.total"
          background
          @current-change="loadNodes"
          @size-change="loadNodes"
        />
      </div>
    </el-card>
    <AppDrawer
      v-model="showAddDialog"
      :title="editingNode ? '编辑节点' : '添加节点'"
      size="600px"
      mobile-size="100%"
      :loading="saving || parsing"
      class="node-form-drawer"
    >
      <div class="dialog-scroll-content">
        <el-tabs v-model="addNodeTab" v-if="!editingNode" class="compact-tabs">
          <el-tab-pane label="链接导入" name="link">
            <div class="import-section">
              <el-alert title="支持 vmess, vless, trojan, ss, ssr 等链接批量导入" type="info" :closable="false" show-icon />
              <el-input
                v-model="nodeLinkInput"
                type="textarea"
                :rows="isMobile ? 8 : 6"
                placeholder="请粘贴节点链接，每行一个..."
                class="link-textarea"
              />
              <div class="parsed-preview" v-if="parsedNode">
                <div class="preview-title">解析预览</div>
                <div class="preview-row"><span>名称:</span> {{ parsedNode.name }}</div>
                <div class="preview-row"><span>地址:</span> {{ parsedNode.server }}:{{ parsedNode.port }}</div>
              </div>
            </div>
          </el-tab-pane>
          <el-tab-pane label="订阅地址导入" name="subscription">
            <div class="import-section">
              <el-alert title="粘贴订阅链接，系统自动拉取并解析（支持 base64 订阅、Clash YAML、节点链接列表等格式）" type="info" :closable="false" show-icon />
              <el-input
                v-model="subUrlInput"
                placeholder="请输入订阅链接，如 https://example.com/sub?token=xxx"
                class="subscription-url-input"
                clearable
              />
              <div class="subscription-tip">
                导入前请确认订阅链接可正常访问；解析出的节点将自动添加为节点。
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>
        <el-form 
          v-if="editingNode" 
          :model="nodeForm" 
          :label-position="isMobile ? 'top' : 'right'" 
          label-width="80px"
          class="node-form"
        >
          <el-row :gutter="12">
            <el-col :span="isMobile ? 24 : 12">
              <el-form-item label="名称" required>
                <el-input v-model="nodeForm.name" placeholder="节点别名" />
              </el-form-item>
            </el-col>
            <el-col :span="isMobile ? 24 : 12">
              <el-form-item label="地区" required>
                <el-input v-model="nodeForm.region" placeholder="如: 香港" />
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="类型" required>
                <el-select v-model="nodeForm.type" placeholder="选择节点类型" class="full-width-control">
                  <el-option-group label="代理协议">
                    <el-option label="VMess" value="vmess" />
                    <el-option label="VLESS" value="vless" />
                    <el-option label="Trojan" value="trojan" />
                    <el-option label="Shadowsocks (SS)" value="ss" />
                    <el-option label="ShadowsocksR (SSR)" value="ssr" />
                  </el-option-group>
                  <el-option-group label="现代协议">
                    <el-option label="Hysteria" value="hysteria" />
                    <el-option label="Hysteria2" value="hysteria2" />
                    <el-option label="TUIC" value="tuic" />
                    <el-option label="Naive" value="naive" />
                    <el-option label="AnyTLS" value="anytls" />
                  </el-option-group>
                  <el-option-group label="其他协议">
                    <el-option label="SOCKS" value="socks" />
                    <el-option label="SOCKS5" value="socks5" />
                    <el-option label="HTTP" value="http" />
                    <el-option label="HTTPS" value="https" />
                    <el-option label="WireGuard (WG)" value="wg" />
                  </el-option-group>
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="配置">
                <el-input 
                  v-model="nodeForm.config" 
                  type="textarea" 
                  :rows="6" 
                  placeholder='{"server":"1.2.3.4", "port":443, ...}' 
                  class="code-input"
                />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="推荐">
                <el-switch v-model="nodeForm.is_recommended" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="激活">
                <el-switch v-model="nodeForm.is_active" />
              </el-form-item>
            </el-col>
          </el-row>
          <div v-if="editingNode" class="link-generator">
            <div class="link-label">节点链接</div>
            <div class="link-box" v-if="nodeLink">
              <div class="link-text">{{ nodeLink }}</div>
              <el-button class="link-copy-btn" type="primary" link @click="copyNodeLink">
                <el-icon><DocumentCopy /></el-icon>
              </el-button>
            </div>
            <div v-else class="link-loading">加载中...</div>
          </div>
        </el-form>
      </div>
      <template #footer>
        <FormActionBar :loading="saving || parsing">
          <el-button :disabled="saving || parsing" @click="showAddDialog = false">取消</el-button>
          <template v-if="!editingNode && addNodeTab === 'link'">
            <el-button type="warning" plain @click="parseNodeLink" :loading="parsing">仅解析预览</el-button>
            <el-button type="primary" @click="batchImportLinks" :loading="saving" :disabled="!nodeLinkInput">批量导入</el-button>
          </template>
          <template v-else-if="!editingNode && addNodeTab === 'subscription'">
            <el-button type="primary" @click="importSubscription" :loading="importingSubscription" :disabled="!subUrlInput">导入订阅</el-button>
          </template>
          <el-button v-else type="primary" @click="saveNode" :loading="saving">保存节点</el-button>
        </FormActionBar>
      </template>
    </AppDrawer>
    <AppDrawer
      v-model="showSelfHostDrawer"
      title="自建节点"
      size="640px"
      mobile-size="100%"
      class="selfhost-drawer"
      @closed="stopSelfHostPolling"
    >
      <div class="dialog-scroll-content">
        <!-- 第一步：填写节点信息 -->
        <div v-if="!selfHostInfo" class="selfhost-section">
          <el-alert
            title="在您的 VPS / 服务器上执行一条命令，即可自动部署 sing-box 节点并回传到本面板。无需手动填写服务器地址、端口和密钥。"
            type="info"
            :closable="false"
            show-icon
            class="selfhost-alert"
          />
          <el-form :model="selfHostForm" label-position="top" class="node-form">
            <el-form-item label="节点名称" required>
              <el-input v-model="selfHostForm.name" placeholder="如: 我的香港 VPS" maxlength="50" show-word-limit />
            </el-form-item>
            <el-form-item label="协议" required>
              <el-select v-model="selfHostForm.protocol" class="full-width-control">
                <el-option label="VLESS + WebSocket（推荐，兼容性最好）" value="vless-ws" />
                <el-option label="VMess + WebSocket" value="vmess-ws" />
                <el-option label="VLESS + Reality（防封锁强，需服务器能访问外网）" value="vless-reality" />
                <el-option label="Trojan + WebSocket" value="trojan-ws" />
                <el-option label="Shadowsocks" value="ss" />
              </el-select>
            </el-form-item>
            <el-form-item label="监听端口" v-if="false">
              <el-input v-model="selfHostForm.port" placeholder="443" />
            </el-form-item>
          </el-form>
          <div class="selfhost-tip">
            <el-icon><InfoFilled /></el-icon>
            <span>脚本会自动检测系统架构、下载 sing-box（内置多镜像源）、生成随机密钥并启动服务，安装完成后节点自动出现在列表中。</span>
          </div>
        </div>
        <!-- 第二步：显示安装命令 -->
        <div v-else class="selfhost-section">
          <div class="selfhost-status-row">
            <span class="selfhost-status-label">安装状态</span>
            <el-tag :type="selfHostStatusType" effect="light" size="default">{{ selfHostStatusText }}</el-tag>
            <el-tag type="info" effect="plain" size="small" v-if="selfHostInfo.protocol_display">{{ selfHostInfo.protocol_display }}</el-tag>
          </div>
          <!-- 节点详情卡片（状态/协议/流量/心跳） -->
          <div class="selfhost-detail-grid" v-if="selfHostInfo">
            <div class="selfhost-detail-item">
              <span class="detail-label">节点名称</span>
              <span class="detail-value">{{ selfHostInfo.name || '-' }}</span>
            </div>
            <div class="selfhost-detail-item">
              <span class="detail-label">协议</span>
              <span class="detail-value">{{ selfHostInfo.protocol_display || '-' }}</span>
            </div>
            <div class="selfhost-detail-item">
              <span class="detail-label">上行流量</span>
              <span class="detail-value">{{ formatBytes(selfHostInfo.traffic_up || 0) }}</span>
            </div>
            <div class="selfhost-detail-item">
              <span class="detail-label">下行流量</span>
              <span class="detail-value">{{ formatBytes(selfHostInfo.traffic_down || 0) }}</span>
            </div>
            <div class="selfhost-detail-item">
              <span class="detail-label">最近心跳</span>
              <span class="detail-value">{{ formatTime(selfHostInfo.last_heartbeat_at) }}</span>
            </div>
            <div class="selfhost-detail-item">
              <span class="detail-label">流量统计时间</span>
              <span class="detail-value">{{ formatTime(selfHostInfo.traffic_updated_at) }}</span>
            </div>
          </div>
          <div class="selfhost-install-title">复制以下命令到您的服务器执行（需要 root 权限）：</div>
          <div class="selfhost-cmd-box">
            <div class="selfhost-cmd-text">{{ selfHostInfo.install_cmd }}</div>
            <el-button class="selfhost-copy-btn" type="primary" link @click="copyInstallCmd">
              <el-icon><DocumentCopy /></el-icon>
            </el-button>
          </div>
          <el-alert
            v-if="selfHostStatus === 'pending'"
            title="等待服务器执行安装命令… 节点回传后此处会自动变为在线状态。"
            type="warning"
            :closable="false"
            show-icon
          />
          <el-alert
            v-else-if="selfHostStatus === 'online'"
            :title="`节点已上线！地址: ${selfHostInfo.link || '已配置'}`"
            type="success"
            :closable="false"
            show-icon
          />
          <el-alert
            v-else-if="selfHostStatus === 'offline'"
            title="节点已离线（心跳超时）。请检查服务器上的 sing-box 与心跳服务是否正常运行。"
            type="error"
            :closable="false"
            show-icon
          />
          <el-alert
            v-else-if="selfHostStatus === 'expired' || selfHostStatus === 'canceled'"
            title="安装令牌已失效。请关闭此窗口后重新创建一个自建节点。"
            type="error"
            :closable="false"
            show-icon
          />
          <div class="selfhost-actions">
            <el-button @click="closeSelfHostDrawer">关闭</el-button>
            <el-button v-if="selfHostStatus === 'online'" type="primary" @click="refreshSelfHostStatus">刷新状态</el-button>
          </div>
        </div>
      </div>
      <template #footer>
        <template v-if="!selfHostInfo">
          <FormActionBar :loading="creatingSelfHost">
            <el-button :disabled="creatingSelfHost" @click="showSelfHostDrawer = false">取消</el-button>
            <el-button type="primary" :loading="creatingSelfHost" :disabled="!selfHostForm.name || !selfHostForm.protocol" @click="createSelfHostNode">
              生成安装命令
            </el-button>
          </FormActionBar>
        </template>
      </template>
    </AppDrawer>
    <!-- 自建节点列表视图 -->
    <AppDrawer
      v-model="showSelfHostList"
      title="自建节点列表"
      size="720px"
      mobile-size="100%"
      class="selfhost-list-drawer"
    >
      <div class="dialog-scroll-content">
        <div v-loading="selfHostListLoading" class="selfhost-list-body">
          <el-empty v-if="!selfHostListLoading && selfHostList.length === 0" description="暂无自建节点，点击右上角「自建节点」创建" />
          <div v-for="n in selfHostList" :key="n.id" class="selfhost-list-card">
            <div class="selfhost-card-header">
              <span class="selfhost-card-name">{{ n.name || '-' }}</span>
              <el-tag :type="selfHostStatusTypeMap[n.status]" effect="light" size="small">{{ selfHostStatusTextMap[n.status] || n.status }}</el-tag>
            </div>
            <div class="selfhost-card-grid">
              <div class="selfhost-card-item">
                <span class="detail-label">协议</span>
                <span class="detail-value">{{ n.protocol_display || n.protocol || '-' }}</span>
              </div>
              <div class="selfhost-card-item">
                <span class="detail-label">上行流量</span>
                <span class="detail-value">{{ formatBytes(n.traffic_up) }}</span>
              </div>
              <div class="selfhost-card-item">
                <span class="detail-label">下行流量</span>
                <span class="detail-value">{{ formatBytes(n.traffic_down) }}</span>
              </div>
              <div class="selfhost-card-item">
                <span class="detail-label">最近心跳</span>
                <span class="detail-value">{{ formatTime(n.last_heartbeat_at) }}</span>
              </div>
              <div class="selfhost-card-item">
                <span class="detail-label">创建时间</span>
                <span class="detail-value">{{ formatTime(n.created_at) }}</span>
              </div>
              <div class="selfhost-card-item">
                <span class="detail-label">状态</span>
                <span class="detail-value">{{ selfHostStatusTextMap[n.status] || n.status }}</span>
              </div>
            </div>
            <div class="selfhost-card-actions">
              <el-button size="small" @click="openSelfHostFromList(n)">详情</el-button>
              <el-button size="small" type="primary" @click="refreshSelfHostList">刷新</el-button>
            </div>
          </div>
        </div>
      </div>
    </AppDrawer>
  </div>
</template>
<script>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { 
  Plus, Refresh, Search, Connection, Delete, 
  DocumentCopy, Edit, MoreFilled, Promotion, InfoFilled, Monitor 
} from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
import { formatDateTimeSafe } from '@/utils/date'
import { formatFileSize } from '@/utils/format'
import { copyToClipboard } from '@/utils/textSelection'
import AppDrawer from '@/components/AppDrawer.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import EmptyState from '@/components/EmptyState.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import { confirmDelete } from '@/utils/confirmAction'
import { useMobile } from '@/composables/useMobile'
import { debounce } from '@/composables/useDebounce'
import { NODE_STATUS_MAP, getNodeStatusType, getNodeStatusText } from '@/utils/statusMaps'
export default {
  name: 'AdminNodes',
  components: { 
    Plus, Refresh, Search, Connection, Delete, 
    DocumentCopy, Edit, MoreFilled, Promotion, InfoFilled, Monitor, AppDrawer, FormActionBar, PaginationBar, EmptyState, ResponsiveDataView
  },
  setup() {
    // 所有支持的节点类型（完整列表）
    const allNodeTypes = [
      'vmess', 'vless', 'trojan', 'ss', 'ssr',
      'hysteria', 'hysteria2', 'tuic', 'naive', 'anytls',
      'socks', 'socks5', 'http', 'https', 'wg', 'wireguard'
    ]
    
    const isMobile = useMobile()
    const tableRef = ref(null)
    const loading = ref(false)
    const testing = ref(false)
    const deleting = ref(false)
    const saving = ref(false)
    const parsing = ref(false)
    const nodes = ref([])
    const selectedNodes = ref([])
    const showAddDialog = ref(false)
    const editingNode = ref(null)
    const searchKeyword = ref('')
    const regions = ref([])
    const types = ref([])
    const addNodeTab = ref('link')
    const nodeLinkInput = ref('')
    const nodeLinkValue = ref('')
    const parsedNode = ref(null)
    const filters = reactive({ status: '', is_active: '', is_manual: '', region: '', type: '' })
    const pagination = reactive({ page: 1, size: 10, total: 0 })
    const mobileNodeFields = computed(() => [
      { key: 'region', label: '地区' },
      { key: 'type', label: '类型' },
      { key: 'source', label: '来源' },
      { key: 'latency', label: '延迟' }
    ])
    const nodeForm = reactive({
      name: '', region: '', type: 'vmess', config: '',
      description: '', is_recommended: false, is_active: true
    })
    const loadNodes = async () => {
      loading.value = true
      try {
        const params = {
          page: pagination.page,
          size: pagination.size,
          ...filters,
          search: searchKeyword.value,
          _t: Date.now() // 缓存穿透参数，避免浏览器/代理返回旧列表
        }
        Object.keys(params).forEach(k => !params[k] && delete params[k])
        const res = await adminAPI.getAdminNodes(params)
        if (res.data?.success) {
          const raw = res.data.data
          const list = Array.isArray(raw) ? raw : (raw.nodes || raw.data || [])
          nodes.value = list.map(n => ({ ...n, testing: false }))
          pagination.total = raw.total || list.length
          const rSet = new Set(), tSet = new Set(allNodeTypes)
          list.forEach(n => { 
            if(n.region) rSet.add(n.region)
            if(n.type) tSet.add(n.type) // 补充任何新的协议类型
          })
          regions.value = Array.from(rSet).sort()
          types.value = Array.from(tSet).sort()
        }
      } catch (err) {
        ElMessage.error('加载失败: ' + err.message)
      } finally {
        loading.value = false
      }
    }
    const applyNodeFilters = () => {
      pagination.page = 1
      loadNodes()
    }
    // 搜索输入实时生效，无需再次点击搜索按钮（500ms 防抖）
    const debouncedApplyNodeFilters = debounce(applyNodeFilters, 500)
    const resetNodeFilters = () => {
      Object.assign(filters, { status: '', is_active: '', is_manual: '', region: '', type: '' })
      searchKeyword.value = ''
      pagination.page = 1
      loadNodes()
    }
    const handleMobileSelect = (node, checked) => {
      if (checked) {
        if (!selectedNodes.value.find(n => n.id === node.id)) {
          selectedNodes.value.push(node)
        }
      } else {
        selectedNodes.value = selectedNodes.value.filter(n => n.id !== node.id)
      }
    }
    const isSelected = (node) => selectedNodes.value.some(n => n.id === node.id)
    const isAllSelected = computed({
      get: () => nodes.value.length > 0 && selectedNodes.value.length === nodes.value.length,
      set: (val) => toggleMobileSelectAll(val)
    })
    const isIndeterminate = computed(() => {
      return selectedNodes.value.length > 0 && selectedNodes.value.length < nodes.value.length
    })
    const toggleMobileSelectAll = (val) => {
      selectedNodes.value = val ? [...nodes.value] : []
    }
    const handleSelectionChange = (val) => selectedNodes.value = val
    const handleAdd = () => {
      resetForm()
      subUrlInput.value = ''
      showAddDialog.value = true
    }
    const handleCommand = (cmd) => {
      const actions = {
        refresh: loadNodes,
        test: batchTest,
        delete: batchDelete,
        selfhost: openSelfHostDrawer,
        'selfhost-list': openSelfHostListView
      }
      actions[cmd] && actions[cmd]()
    }
    const editNode = async (node) => {
      editingNode.value = node
      Object.assign(nodeForm, {
        name: node.name || '',
        region: node.region || '',
        type: node.type || 'vmess',
        config: typeof node.config === 'object' ? JSON.stringify(node.config, null, 2) : (node.config || ''),
        description: node.description || '',
        is_recommended: !!node.is_recommended,
        is_active: node.is_active !== false
      })
      nodeLinkValue.value = ''
      showAddDialog.value = true
      // 异步获取真实节点链接
      try {
        const res = await adminAPI.getNodeLink(node.id)
        if (res.data?.success && res.data.data?.link) {
          nodeLinkValue.value = res.data.data.link
        }
      } catch (e) {
        console.warn('获取节点链接失败', e)
      }
    }
    const saveNode = async () => {
      if (!nodeForm.name || !nodeForm.region) return ElMessage.warning('请填写必填项')
      saving.value = true
      try {
        const payload = { ...nodeForm }
        const res = editingNode.value 
          ? await adminAPI.updateNode(editingNode.value.id, payload)
          : await adminAPI.createNode(payload)
        if (res.data.success) {
          ElMessage.success('保存成功')
          showAddDialog.value = false
          loadNodes()
        }
      } catch (err) {
        ElMessage.error('保存失败: ' + err.message)
      } finally {
        saving.value = false
      }
    }
    const deleteNode = async (node) => {
      try {
        await confirmDelete('节点', 1, {
          message: `确认删除节点 "${node.name}"?`
        })
        await adminAPI.deleteNode(node.id)
        ElMessage.success('删除成功')
        // 先从本地列表立即移除，再向服务端重新拉取校准，保证页面马上不显示已删除节点
        nodes.value = nodes.value.filter(n => n.id !== node.id)
        if (pagination.total > 0) pagination.total -= 1
        selectedNodes.value = selectedNodes.value.filter(n => n.id !== node.id)
        tableRef.value?.clearSelection()
        loadNodes()
      } catch (error) {
        if (error !== 'cancel') ElMessage.error('删除节点失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const subUrlInput = ref('')
    const importingSubscription = ref(false)
    const importSubscription = async () => {
      const url = (subUrlInput.value || '').trim()
      if (!url) {
        ElMessage.warning('请输入订阅链接')
        return
      }
      importingSubscription.value = true
      try {
        const res = await adminAPI.importNodeSubscription(url)
        const data = res.data?.data ?? {}
        const imported = data.imported ?? 0
        const total = data.total ?? imported
        if (imported > 0) {
          ElMessage.success(`订阅解析出 ${total} 个节点，成功导入 ${imported} 个`)
          showAddDialog.value = false
          loadNodes()
        } else {
          ElMessage.warning(res.data?.message || '订阅中没有解析到节点')
        }
      } catch (e) {
        if (e.code === 'ECONNABORTED' || (e.message && e.message.includes('timeout'))) {
          ElMessage.error('订阅导入超时：订阅源响应过慢，请检查链接是否可访问后重试')
        } else {
          ElMessage.error('导入订阅失败: ' + (e.response?.data?.message || e.message))
        }
      } finally {
        importingSubscription.value = false
      }
    }

    // ==================== 自建节点 ====================
    const showSelfHostDrawer = ref(false)
    const creatingSelfHost = ref(false)
    const selfHostInfo = ref(null)
    const selfHostForm = reactive({ name: '', protocol: 'vless-ws', port: 443 })
    let selfHostPollTimer = null

    const openSelfHostDrawer = () => {
      selfHostInfo.value = null
      showSelfHostDrawer.value = true
    }
    const closeSelfHostDrawer = () => {
      showSelfHostDrawer.value = false
      stopSelfHostPolling()
      loadNodes()
    }
    const createSelfHostNode = async () => {
      const name = (selfHostForm.name || '').trim()
      if (!name) {
        ElMessage.warning('请填写节点名称')
        return
      }
      creatingSelfHost.value = true
      try {
        const res = await adminAPI.createSelfHostNode({
          name,
          protocol: selfHostForm.protocol
        })
        if (res.data?.success) {
          selfHostInfo.value = res.data.data
          ElMessage.success('安装命令已生成，请复制到您的服务器执行')
          startSelfHostPolling(res.data.data.node?.id)
        } else {
          ElMessage.error(res.data?.message || '创建自建节点失败')
        }
      } catch (e) {
        ElMessage.error('创建失败: ' + (e.response?.data?.message || e.message))
      } finally {
        creatingSelfHost.value = false
      }
    }
    const startSelfHostPolling = (nodeId) => {
      stopSelfHostPolling()
      if (!nodeId) return
      selfHostPollTimer = setInterval(async () => {
        try {
          const res = await adminAPI.getSelfHostNodeStatus(nodeId)
          if (res.data?.success) {
            selfHostInfo.value = { ...selfHostInfo.value, ...res.data.data }
            if (res.data.data?.status === 'online') {
              stopSelfHostPolling()
              loadNodes()
            }
          }
        } catch (e) {
          // 轮询失败静默，下次继续
        }
      }, 3000)
    }
    const stopSelfHostPolling = () => {
      if (selfHostPollTimer) {
        clearInterval(selfHostPollTimer)
        selfHostPollTimer = null
      }
    }
    const refreshSelfHostStatus = async () => {
      if (!selfHostInfo.value?.id) return
      try {
        const res = await adminAPI.getSelfHostNodeStatus(selfHostInfo.value.id)
        if (res.data?.success) {
          selfHostInfo.value = { ...selfHostInfo.value, ...res.data.data }
          if (res.data.data?.status === 'online') {
            stopSelfHostPolling()
            loadNodes()
          }
        }
      } catch (e) {
        ElMessage.error('刷新状态失败')
      }
    }
    const copyInstallCmd = () => {
      if (selfHostInfo.value?.install_cmd) {
        copyToClipboard(selfHostInfo.value.install_cmd, '安装命令已复制')
      }
    }
    const selfHostStatus = computed(() => selfHostInfo.value?.status || 'pending')
    const selfHostStatusText = computed(() => ({
      pending: '等待安装',
      online: '在线',
      offline: '离线',
      expired: '已过期',
      canceled: '已取消'
    }[selfHostStatus.value] || '未知'))
    const selfHostStatusType = computed(() => ({
      pending: 'warning',
      online: 'success',
      offline: 'danger',
      expired: 'info',
      canceled: 'info'
    }[selfHostStatus.value] || 'info'))
    const formatTime = (t) => {
      return formatDateTimeSafe(t, 'YYYY-MM-DD HH:mm:ss', '-')
    }
    const formatBytes = (bytes) => {
      return formatFileSize(bytes)
    }

    // ==================== 自建节点列表视图 ====================
    const showSelfHostList = ref(false)
    const selfHostList = ref([])
    const selfHostListLoading = ref(false)
    const selfHostStatusTypeMap = {
      pending: 'warning', online: 'success', offline: 'danger', expired: 'info', canceled: 'info'
    }
    const selfHostStatusTextMap = {
      pending: '等待安装', online: '在线', offline: '离线', expired: '已过期', canceled: '已取消'
    }
    const loadSelfHostList = async () => {
      selfHostListLoading.value = true
      try {
        const res = await adminAPI.getSelfHostNodes()
        if (res.data?.success) {
          selfHostList.value = res.data.data?.list || []
        }
      } catch (e) {
        ElMessage.error('加载自建节点列表失败: ' + (e.response?.data?.message || e.message))
      } finally {
        selfHostListLoading.value = false
      }
    }
    const openSelfHostListView = () => {
      showSelfHostList.value = true
      loadSelfHostList()
    }
    const refreshSelfHostList = () => {
      loadSelfHostList()
    }
    const openSelfHostFromList = (n) => {
      showSelfHostList.value = false
      selfHostInfo.value = n
      selfHostForm.name = n.name || ''
      showSelfHostDrawer.value = true
      startSelfHostPolling(n.id)
    }

    const batchTest = async () => {
      testing.value = true
      try {
        await adminAPI.batchTestNodes(selectedNodes.value.map(n => n.id))
        ElMessage.success('批量测试请求已发送')
        setTimeout(loadNodes, 1000) // 稍作延迟刷新
      } catch (err) {
        ElMessage.error('测试失败')
      } finally {
        testing.value = false
      }
    }
    const batchDelete = async () => {
      try {
        await confirmDelete('节点', selectedNodes.value.length, {
          message: `确认删除选中的 ${selectedNodes.value.length} 个节点?`
        })
        deleting.value = true
        const deletedIds = selectedNodes.value.map(n => n.id)
        const res = await adminAPI.batchDeleteNodes(deletedIds)
        const deletedCount = res.data?.data?.deleted_count ?? 0
        if (!res.data?.success) {
          throw new Error(res.data?.message || '批量删除失败')
        }
        if (deletedCount > 0) {
          ElMessage.success(`批量删除成功，已删除 ${deletedCount} 个节点`)
        } else {
          // 节点实际已不存在（页面显示的是旧数据），直接刷新列表即可
          ElMessage.info(res.data?.message || '所选节点已不存在，已刷新列表')
        }
        // 先从本地列表立即移除所选节点，再向服务端重新拉取校准，
        // 避免因浏览器/代理缓存导致重新加载后仍显示旧节点
        nodes.value = nodes.value.filter(n => !deletedIds.includes(n.id))
        pagination.total = Math.max(0, pagination.total - deletedIds.length)
        selectedNodes.value = [] // 重置选中
        tableRef.value?.clearSelection()
        await loadNodes()
      } catch (error) {
        if (error === 'cancel') return
        const msg = error.response?.data?.message || error.message
        // 旧版后端返回 404"未删除任何节点"/"节点不存在"说明所选节点实际已删除
        // （页面显示的是旧数据），此时直接从本地移除并刷新列表即可，不再提示失败
        if (msg && (msg.includes('未删除任何节点') || msg.includes('节点不存在'))) {
          const staleIds = selectedNodes.value.map(n => n.id)
          nodes.value = nodes.value.filter(n => !staleIds.includes(n.id))
          pagination.total = Math.max(0, pagination.total - staleIds.length)
          selectedNodes.value = []
          tableRef.value?.clearSelection()
          ElMessage.info('所选节点已不存在，已刷新列表')
          await loadNodes()
          return
        }
        ElMessage.error('批量删除失败: ' + msg)
      } finally {
        deleting.value = false
      }
    }
    const testNode = async (node) => {
      node.testing = true
      try {
        const res = await adminAPI.testNode(node.id)
        if (res.data.success) {
          node.latency = res.data.data.latency
          node.status = res.data.data.status
          ElMessage.success(`延迟: ${node.latency}ms`)
        }
      } catch {
        ElMessage.error('测试失败')
      } finally {
        node.testing = false
      }
    }
    const toggleNodeStatus = async (node) => {
      try {
        await adminAPI.updateNode(node.id, { is_active: node.is_active })
        ElMessage.success(node.is_active ? '已启用' : '已禁用')
      } catch {
        node.is_active = !node.is_active
        ElMessage.error('状态更新失败')
      }
    }
    const parseNodeLink = async () => {
      const link = nodeLinkInput.value.split('\n')[0].trim()
      if (!link) return ElMessage.warning('请输入链接')
      parsing.value = true
      try {
        const res = await adminAPI.createNode({ node_link: link, preview: true })
        if (res.data.success) parsedNode.value = res.data.data
      } finally { parsing.value = false }
    }
    const batchImportLinks = async () => {
      const links = nodeLinkInput.value.split('\n').map(l => l.trim()).filter(Boolean)
      if (!links.length) return
      saving.value = true
      try {
        const res = await adminAPI.importNodeLinks(links)
        const imported = res.data.data?.imported ?? res.data.imported ?? 0
        const skipped = res.data.data?.skipped ?? res.data.skipped ?? 0
        const failed = res.data.data?.failed ?? res.data.failed ?? 0
        const errors = res.data.data?.errors ?? []
        if (imported > 0) {
          ElMessage.success(`导入成功 ${imported} 个${skipped ? `，跳过重复 ${skipped} 个` : ''}${failed ? `，失败 ${failed} 个` : ''}`)
          showAddDialog.value = false
          loadNodes()
        } else if (failed > 0) {
          // 全部失败时保留弹窗，方便用户修改链接后重试
          ElMessage.error(`导入失败 ${failed} 个：${errors[0] || '未知原因'}`)
        } else {
          ElMessage.warning(`所选节点均已存在（重复），已跳过 ${skipped} 个`)
          showAddDialog.value = false
          loadNodes()
        }
      } catch(e) {
        ElMessage.error('导入出错')
      } finally { saving.value = false }
    }
    const resetForm = () => {
      editingNode.value = null
      Object.assign(nodeForm, { name: '', region: '', type: 'vmess', config: '', description: '', is_recommended: false, is_active: true })
      nodeLinkInput.value = ''
      nodeLinkValue.value = ''
      parsedNode.value = null
    }
    const getStatusType = (s) => NODE_STATUS_MAP[s] ? getNodeStatusType(s) : 'info'
    const getStatusText = (s) => NODE_STATUS_MAP[s] ? getNodeStatusText(s) : '未知'
    const formatLatency = (l) => l > 0 ? `${l}ms` : '-'
    const getLatencyClass = (l) => l <= 0 ? '' : l < 200 ? 'text-green' : l < 500 ? 'text-orange' : 'text-red'
    const nodeLink = computed(() => {
      if (!editingNode.value) return ''
      return nodeLinkValue.value || ''
    })
    const copyNodeLink = () => {
      copyToClipboard(nodeLink.value, '复制成功')
    }
    onMounted(() => {
      loadNodes()
    })
    return {
      isMobile, tableRef, loading, testing, deleting, saving, parsing,
      nodes, selectedNodes, showAddDialog, editingNode,
      filters, pagination, nodeForm, regions, types, allNodeTypes,
      searchKeyword, addNodeTab, nodeLinkInput, parsedNode, subUrlInput, importingSubscription, importSubscription, mobileNodeFields,
      loadNodes, applyNodeFilters, debouncedApplyNodeFilters, resetNodeFilters, handleSelectionChange, handleMobileSelect,
      handleAdd, handleCommand, editNode, saveNode, deleteNode,
      batchTest, batchDelete, testNode, toggleNodeStatus,
      parseNodeLink, batchImportLinks, copyNodeLink, nodeLink,
      getStatusType, getStatusText, getLatencyClass, formatLatency,
      isSelected, isAllSelected, isIndeterminate, toggleMobileSelectAll,
      showSelfHostDrawer, creatingSelfHost, selfHostInfo, selfHostForm,
      openSelfHostDrawer, closeSelfHostDrawer, createSelfHostNode,
      copyInstallCmd, refreshSelfHostStatus, stopSelfHostPolling,
      selfHostStatus, selfHostStatusText, selfHostStatusType, formatTime, formatBytes,
      showSelfHostList, selfHostList, selfHostListLoading,
      selfHostStatusTypeMap, selfHostStatusTextMap,
      openSelfHostListView, refreshSelfHostList, openSelfHostFromList,
      Plus, Refresh, Search, Connection, Delete, DocumentCopy, Edit, MoreFilled, Promotion, InfoFilled, Monitor
    }
  }
}
</script>
<style scoped>
.admin-nodes {
  padding: 12px;
}
@media (max-width: 768px) {
  .admin-nodes {
    padding: 10px;
  }
}
.list-card {
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}
.danger-menu-item {
  color: var(--el-color-danger);
}
.text-placeholder {
  color: var(--el-text-color-placeholder);
}
.full-width-control {
  width: 100%;
}
@media (max-width: 768px) {
  .search-box {
    grid-column: 1 / -1;
  }
}
.desktop-table .status-badge {
  margin-top: 4px;
}
.text-green { color: var(--el-color-success); font-weight: 500; }
.text-orange { color: var(--el-color-warning); }
.text-red { color: var(--el-color-danger); }
.mobile-selection-bar {
  display: none;
  padding: 0 4px 10px;
}
.mobile-node-header {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.mobile-node-title {
  font-weight: 600;
  font-size: 15px;
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: clip;
  text-overflow: ellipsis;
}
.mobile-node-checkbox {
  margin-right: 0;
  height: auto;
}
.mobile-node-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
}
.mobile-node-buttons {
  display: flex;
  gap: 4px;
  min-width: 0;
  flex-wrap: wrap;
  justify-content: flex-end;
}
/* 移动端：开关一行、测试/编辑/删除三按钮一行均分，避免纵向堆叠错乱 */
@media (max-width: 768px) {
  .mobile-node-actions {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
  .mobile-node-buttons {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
    width: 100%;
  }
  .mobile-node-buttons .el-button {
    margin: 0 !important;
    width: 100%;
    padding: 8px 4px;
    font-size: 13px;
  }
}
.dialog-scroll-content {
  min-width: 0;
}
@media (max-width: 768px) {
  .mobile-selection-bar {
    display: block;
  }
  .mobile-node-actions {
    align-items: flex-start;
  }
  .link-generator {
    margin-top: 10px;
  }
  .link-box {
    padding: 6px;
  }
  .link-text {
    font-size: 11px;
    max-height: 120px;
    overflow-y: auto;
  }
}
.link-generator {
  background: var(--el-fill-color-light);
  padding: 10px;
  border-radius: 4px;
  margin-top: 10px;
}
.link-label {
  font-size: 13px;
  color: var(--el-text-color-regular);
  margin-bottom: 6px;
}
.link-box {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  background: #fff;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  padding: 8px;
}
.link-text {
  flex: 1;
  min-width: 0;
  font-family: monospace;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
  line-height: 1.5;
  user-select: all;
}
.link-copy-btn {
  flex-shrink: 0;
}
.link-loading {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  padding: 4px 0;
}
/* ===== 自建节点 ===== */
.selfhost-alert {
  margin-bottom: 16px;
}
.selfhost-tip {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin-top: 12px;
  padding: 10px 12px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}
.selfhost-tip .el-icon {
  margin-top: 3px;
  flex-shrink: 0;
}
.selfhost-status-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}
.selfhost-status-label {
  font-weight: 600;
  font-size: 14px;
}
/* 自建节点详情网格 */
.selfhost-detail-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 14px;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}
.selfhost-detail-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.detail-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.detail-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  word-break: break-all;
}
@media (max-width: 768px) {
  .selfhost-detail-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
.selfhost-install-title {
  font-size: 14px;
  color: var(--el-text-color-primary);
  margin-bottom: 8px;
}
.selfhost-cmd-box {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  background: #1e1e1e;
  border-radius: 6px;
  padding: 10px 12px;
  margin-bottom: 14px;
}
.selfhost-cmd-text {
  flex: 1;
  min-width: 0;
  font-family: monospace;
  font-size: 12px;
  color: #d4d4d4;
  word-break: break-all;
  line-height: 1.6;
  user-select: all;
}
.selfhost-copy-btn {
  flex-shrink: 0;
}
.selfhost-meta {
  margin-top: 12px;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}
.selfhost-actions {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
@media (max-width: 768px) {
  .selfhost-section {
    padding: 4px 2px;
  }
  .selfhost-cmd-text {
    font-size: 11px;
  }
}
/* ===== 自建节点列表视图 ===== */
.selfhost-list-body {
  min-height: 200px;
}
.selfhost-list-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 12px 14px;
  margin-bottom: 12px;
  background: var(--el-bg-color);
}
.selfhost-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
}
.selfhost-card-name {
  font-weight: 600;
  font-size: 15px;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.selfhost-card-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px 12px;
}
.selfhost-card-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.selfhost-card-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 10px;
}
@media (max-width: 768px) {
  .selfhost-card-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
