<template>
  <div class="list-container admin-custom-nodes">
    <el-card class="list-card" shadow="never">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <span class="title-text">专线节点管理</span>
            <el-tag v-if="pagination.total" type="info" round size="small" class="count-tag">{{ pagination.total }}</el-tag>
          </div>
          <div class="header-actions" v-if="!isMobile">
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
            <el-button type="primary" @click="openCreateNodeDialog">
              <el-icon><Plus /></el-icon>创建节点
            </el-button>
            <el-button @click="loadCustomNodes" :loading="loading">
              <el-icon><Refresh /></el-icon>刷新
            </el-button>
          </div>
          <div class="header-actions mobile" v-else>
            <el-button type="primary" circle @click="openCreateNodeDialog" size="small">
              <el-icon><Plus /></el-icon>
            </el-button>
            <el-dropdown trigger="click" @command="handleCommand">
              <el-button circle size="small">
                <el-icon><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="refresh" :icon="Refresh">刷新列表</el-dropdown-item>
                  <el-dropdown-item command="batch_test" :disabled="!selectedNodes.length" :icon="Connection">批量测速</el-dropdown-item>
                  <el-dropdown-item command="batch_assign" :disabled="!selectedNodes.length" :icon="User">批量分配</el-dropdown-item>
                  <el-dropdown-item command="batch_unassign" :disabled="!selectedNodes.length" :icon="Close">批量取消分配</el-dropdown-item>
                  <el-dropdown-item command="migrate_assignments" :disabled="selectedNodes.length !== 1" :icon="Connection">迁移分配</el-dropdown-item>
                  <el-dropdown-item command="batch_delete" :disabled="!selectedNodes.length" :icon="Delete" divided class="danger-menu-item">批量删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </template>
      <div class="filter-wrapper">
        <div class="filter-grid">
          <el-select v-model="filters.status" placeholder="状态" clearable @change="handleFilterChange">
            <el-option label="全部状态" value="" />
            <el-option label="活跃" value="active" />
            <el-option label="非活跃" value="inactive" />
            <el-option label="错误" value="error" />
          </el-select>
          <el-select v-model="filters.protocol" placeholder="节点类型" clearable @change="handleFilterChange">
            <el-option label="全部类型" value="" />
            <el-option-group v-for="group in nodeTypeGroups" :key="group.label" :label="group.label">
              <el-option
                v-for="type in group.options"
                :key="type.value"
                :label="type.label"
                :value="type.value"
              />
            </el-option-group>
          </el-select>
          <el-select v-model="filters.is_active" placeholder="激活" clearable @change="handleFilterChange">
            <el-option label="全部" value="" />
            <el-option label="已激活" value="true" />
            <el-option label="已禁用" value="false" />
          </el-select>
          <el-select v-model="filters.source" placeholder="来源" clearable @change="handleFilterChange">
            <el-option label="全部来源" value="" />
            <el-option label="手动添加" value="manual" />
            <el-option label="链接导入" value="link" />
            <el-option label="订阅导入" value="subscription" />
          </el-select>
          <div class="search-box">
            <el-input
              v-model="searchKeyword"
              placeholder="搜索名称/域名/用户..."
              clearable
              @input="debouncedSearch"
              @clear="handleFilterChange"
              @keyup.enter="handleFilterChange"
            >
              <template #prefix><el-icon><Search /></el-icon></template>
            </el-input>
          </div>
          <div class="filter-actions">
            <el-button type="primary" @click="handleFilterChange">
              <el-icon><Search /></el-icon>
              搜索
            </el-button>
            <el-button @click="resetFilters">
              <el-icon><Refresh /></el-icon>
              重置
            </el-button>
          </div>
        </div>
      </div>
      <div v-if="selectedNodes.length > 0 && !isMobile" class="batch-actions-bar">
        <span class="batch-tip">已选择 {{ selectedNodes.length }} 个节点</span>
        <div class="batch-btns">
          <el-button type="success" link @click="batchTest" :loading="batchTesting">批量测速</el-button>
          <el-divider direction="vertical" />
          <el-button type="primary" link @click="handleBatchAssignClick">批量分配</el-button>
          <el-divider direction="vertical" />
          <el-button type="warning" link @click="batchUnassign" :loading="batchUnassigning">批量取消</el-button>
          <el-divider direction="vertical" />
          <el-button type="info" link @click="openMigrateDialog(selectedNodes[0])" :disabled="selectedNodes.length !== 1">迁移分配</el-button>
          <el-divider direction="vertical" />
          <el-button type="danger" link @click="batchDelete" :loading="batchDeleting">批量删除</el-button>
        </div>
      </div>
      <div class="content-view" v-loading="loading">
        <el-table
          v-if="!isMobile && viewMode === 'table'"
          :data="customNodes"
          stripe
          border
          @selection-change="handleSelectionChange"
          @header-dragend="handleColumnResize"
          row-key="id"
          class="desktop-table"
          ref="tableRef"
        >
          <el-table-column type="selection" column-key="selection" :width="columnWidths.selection" resizable />
          <el-table-column prop="name" label="名称" :min-width="columnWidths.name" resizable show-overflow-tooltip />
          <el-table-column prop="display_name" label="显示名称" :min-width="columnWidths.display_name" resizable show-overflow-tooltip>
            <template #default="{ row }">
              <span :class="row.display_name ? '' : 'text-secondary'">
                {{ row.display_name || row.name || '-' }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="来源" :width="90" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="getSourceTagType(row.source)" effect="plain">{{ getSourceText(row.source) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="protocol" label="协议" :width="columnWidths.protocol" resizable>
            <template #default="{ row }">
              <el-tag size="small" effect="plain">{{ getProtocolLabel(row.protocol) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="domain" column-key="server_ip" label="服务器IP" :min-width="columnWidths.server_ip" resizable show-overflow-tooltip>
            <template #default="{ row }">
              <span :class="getNodeServer(row) ? 'server-address' : 'text-secondary'">
                {{ getNodeServer(row) || '-' }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="状态" column-key="status" :width="columnWidths.status" resizable>
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)" size="small" effect="light">
                {{ getStatusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="激活" column-key="is_active" :width="columnWidths.is_active" resizable>
            <template #default="{ row }">
              <el-switch v-model="row.is_active" @change="toggleNodeStatus(row)" size="small" />
            </template>
          </el-table-column>
          <el-table-column label="到期" column-key="expire_time" :width="columnWidths.expire_time" resizable>
            <template #default="{ row }">
              <span class="text-xs">{{ formatExpire(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" column-key="actions" :width="columnWidths.actions" fixed="right" resizable>
            <template #default="{ row }">
              <div class="table-actions">
                <el-button size="small" @click="testNode(row)" :loading="row.testing">测试</el-button>
                <el-button size="small" type="success" plain @click="viewLink(row)">链接</el-button>
                <el-button size="small" type="warning" plain @click="assignSingleNode(row)">分配</el-button>
                <el-button size="small" type="info" plain @click="openMigrateDialog(row)">迁移</el-button>
                <el-button size="small" type="primary" plain @click="editNode(row)" :icon="Edit">编辑</el-button>
                <el-button size="small" type="danger" plain @click="deleteNode(row)" :icon="Delete">删除</el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>
        <div v-if="!isMobile && viewMode === 'grid'" class="desktop-grid-view" :class="[
          gridOrientation === 'horizontal' ? 'grid-horizontal' : 'grid-vertical',
          gridOrientation === 'vertical' ? 'grid-size-' + gridSize : '',
          'grid-cols-' + gridColumns
        ]">
          <template v-if="customNodes.length === 0">
            <EmptyState
              class="grid-empty"
              title="暂无专线节点"
              description="创建专线节点或调整筛选条件后可在这里查看"
            />
          </template>
          <template v-else>
            <div
              v-for="node in customNodes"
              :key="node.id"
              class="grid-node-card"
              :class="{ 'is-selected': isSelected(node) }"
            >
              <div class="gnc-header">
                <el-checkbox
                  :model-value="isSelected(node)"
                  @change="(val) => handleGridSelect(node, val)"
                  class="gnc-checkbox"
                />
                <span class="gnc-title" :title="node.name">{{ node.name }}</span>
                <el-tag size="small" :type="getSourceTagType(node.source)" effect="plain">{{ getSourceText(node.source) }}</el-tag>
                <el-tag :type="getStatusType(node.status)" size="small" effect="dark">
                  {{ getStatusText(node.status) }}
                </el-tag>
              </div>
              <div class="gnc-body">
                <div class="gnc-row">
                  <span class="label">协议</span>
                  <span class="value">{{ getProtocolLabel(node.protocol) }}</span>
                </div>
                <div class="gnc-row">
                  <span class="label">服务器IP</span>
                  <span class="value">{{ getNodeServer(node) || '-' }}</span>
                </div>
                <div class="gnc-row">
                  <span class="label">端口</span>
                  <span class="value">{{ node.port || '-' }}</span>
                </div>
                <div class="gnc-row">
                  <span class="label">到期</span>
                  <span class="value text-xs">{{ formatExpire(node) }}</span>
                </div>
              </div>
              <div class="gnc-footer">
                <el-switch
                  v-model="node.is_active"
                  @change="toggleNodeStatus(node)"
                  size="small"
                  inline-prompt
                  active-text="开"
                  inactive-text="关"
                />
                <div class="gnc-actions">
                  <el-button size="small" @click="testNode(node)" :loading="node.testing">测试</el-button>
                  <el-button size="small" type="success" plain @click="viewLink(node)">链接</el-button>
                  <el-button size="small" type="warning" plain @click="assignSingleNode(node)">分配</el-button>
                  <el-button size="small" type="info" plain @click="openMigrateDialog(node)">迁移</el-button>
                  <el-button size="small" type="primary" plain @click="editNode(node)" :icon="Edit">编辑</el-button>
                  <el-button size="small" type="danger" plain @click="deleteNode(node)" :icon="Delete">删除</el-button>
                </div>
              </div>
            </div>
          </template>
        </div>
        <div v-if="isMobile" class="mobile-list">
          <div class="mobile-selection-bar" v-if="customNodes.length > 0">
            <el-checkbox 
              v-model="isAllSelected" 
              :indeterminate="isIndeterminate" 
              @change="toggleMobileSelectAll"
            >全选 ({{ selectedNodes.length }})</el-checkbox>
          </div>
          <div v-for="node in customNodes" :key="node.id" class="node-card">
            <div class="card-header-row">
              <el-checkbox 
                :model-value="isSelected(node)" 
                @change="(val) => handleMobileSelect(node, val)"
                class="card-checkbox" 
              />
              <div class="node-title">{{ node.name }}</div>
              <el-tag size="small" :type="getStatusType(node.status)" effect="light">{{ getStatusText(node.status) }}</el-tag>
            </div>
            <div class="card-info-grid">
              <div class="info-item">
                <span class="label">来源</span>
                <span class="value"><el-tag size="small" :type="getSourceTagType(node.source)" effect="plain">{{ getSourceText(node.source) }}</el-tag></span>
              </div>
              <div class="info-item">
                <span class="label">协议</span>
                <span class="value">{{ getProtocolLabel(node.protocol) }}</span>
              </div>
              <div class="info-item">
                <span class="label">服务器IP</span>
                <span class="value">{{ getNodeServer(node) || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="label">端口</span>
                <span class="value">{{ node.port }}</span>
              </div>
              <div class="info-item full-width">
                <span class="label">到期</span>
                <span class="value">{{ formatExpire(node) }}</span>
              </div>
            </div>
            <div class="card-actions-row">
              <div class="left-actions">
                <el-switch 
                  v-model="node.is_active" 
                  @change="toggleNodeStatus(node)" 
                  size="small"
                  inline-prompt
                  active-text="开"
                  inactive-text="关"
                />
              </div>
              <div class="right-buttons">
                <el-button size="small" text bg @click="testNode(node)" :loading="node.testing">测试</el-button>
                <el-button size="small" text bg type="warning" @click="assignSingleNode(node)">分配</el-button>
                <el-dropdown trigger="click">
                  <el-button size="small" text bg>更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item @click="viewLink(node)" :icon="Link">链接</el-dropdown-item>
                      <el-dropdown-item @click="openMigrateDialog(node)" :icon="Connection">迁移分配</el-dropdown-item>
                      <el-dropdown-item @click="editNode(node)" :icon="Edit">编辑</el-dropdown-item>
                      <el-dropdown-item @click="deleteNode(node)" :icon="Delete" class="danger-menu-item">删除</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
            </div>
          </div>
          <EmptyState
            v-if="customNodes.length === 0"
            title="暂无专线节点"
            description="创建专线节点或调整筛选条件后可在这里查看"
          />
        </div>
      </div>
      <div class="pagination-wrapper">
        <PaginationBar
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.size"
          :total="pagination.total"
          background
          @current-change="loadCustomNodes"
          @size-change="loadCustomNodes"
        />
      </div>
    </el-card>
    <AppDrawer
      v-model="showAddDialog"
      :title="editingNode ? '编辑专线节点' : '添加专线节点'"
      size="600px"
      mobile-size="100%"
      :loading="saving || parsing"
    >
      <div class="dialog-scroll-content">
        <el-tabs v-model="addNodeTab" v-if="!editingNode" class="compact-tabs">
          <el-tab-pane label="链接导入" name="link">
            <div class="import-section">
              <el-alert title="支持 vmess, vless, trojan, ss, hysteria2 等链接批量导入" type="info" :closable="false" show-icon />
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
          <el-tab-pane label="订阅导入" name="subscription">
            <div class="import-section">
              <el-alert title="粘贴订阅链接，系统自动拉取并解析（支持 base64 订阅、Clash YAML、节点链接列表等格式）" type="info" :closable="false" show-icon />
              <!-- 已导入订阅：更新 / 删除 -->
              <div v-if="subscriptionList.length > 0" class="subscription-manage-section">
                <div class="sub-section-label">已导入的订阅（{{ subscriptionList.length }}）</div>
                <div
                  v-for="sub in subscriptionList"
                  :key="sub.url"
                  class="subscription-item"
                >
                  <div class="subscription-item-main">
                    <div class="subscription-item-url" :title="sub.url">
                      <template v-if="isLegacySubscription(sub.url)">历史订阅导入节点（旧版数据，无原始订阅地址）</template>
                      <template v-else>{{ sub.url }}</template>
                    </div>
                    <div class="subscription-item-count">共 {{ sub.node_count }} 个节点</div>
                  </div>
                  <div class="subscription-item-actions">
                    <template v-if="isLegacySubscription(sub.url)">
                      <el-tooltip content="历史导入节点无原始订阅地址，可通过下方「替换模式」导入新订阅整体替换" placement="top">
                        <el-button size="small" type="warning" plain @click="handleReplaceLegacy(sub)">
                          用新订阅替换
                        </el-button>
                      </el-tooltip>
                    </template>
                    <template v-else>
                      <el-button size="small" type="primary" plain @click="handleUpdateSubscription(sub)">
                        更新此订阅
                      </el-button>
                    </template>
                    <el-button size="small" type="danger" plain @click="handleDeleteSubscription(sub)">
                      删除
                    </el-button>
                  </div>
                </div>
                <el-divider />
              </div>
              <!-- 导入新订阅 / 更换订阅地址 -->
              <div class="sub-section-label">导入新订阅或更换订阅地址</div>
              <el-input
                v-model="subUrlInput"
                placeholder="请输入订阅链接，如 https://example.com/sub?token=xxx"
                class="subscription-url-input"
                clearable
              />
              <div class="sub-replace-row">
                <el-checkbox v-model="subReplaceMode" class="sub-replace-checkbox">
                  <span class="sub-replace-label">替换模式</span>
                </el-checkbox>
                <span class="subscription-tip">
                  {{ subReplaceMode ? '将删除该订阅地址下原有节点，按最新订阅内容重建（适用于订阅地址更换或内容更新）' : '仅追加新节点，已存在的节点自动跳过（推荐用于订阅内容增量同步）' }}
                </span>
              </div>
              <div class="subscription-tip">
                导入前请确认订阅链接可正常访问；解析出的节点将自动添加为专线节点。
              </div>
            </div>
          </el-tab-pane>
          <el-tab-pane label="VPS自动搭建" name="vps">
            <div class="import-section">
              <el-alert
                title="填写 VPS 的 IP/域名、SSH 端口与 root 密码，系统将 SSH 全自动部署 sing-box 节点并回传，无需手动执行命令。"
                type="success"
                :closable="false"
                show-icon
              />
              <el-form :model="vpsForm" label-position="top" class="vps-form">
                <el-form-item label="节点名称" required>
                  <el-input v-model="vpsForm.name" placeholder="如: 我的东京VPS" maxlength="50" />
                </el-form-item>
                <el-form-item label="协议" required>
                  <el-select v-model="vpsForm.protocol" class="full-width-control">
                    <el-option label="VLESS + WebSocket（推荐）" value="vless-ws" />
                    <el-option label="VMess + WebSocket" value="vmess-ws" />
                    <el-option label="VLESS + Reality（防封锁强）" value="vless-reality" />
                    <el-option label="Trojan + WebSocket" value="trojan-ws" />
                    <el-option label="Shadowsocks" value="ss" />
                  </el-select>
                </el-form-item>
                <el-form-item label="VPS IP / 域名" required>
                  <el-input v-model="vpsForm.ssh_host" placeholder="如: 113.20.13.206 或 vps.example.com" />
                </el-form-item>
                <div class="inline-fields-row">
                  <el-form-item label="SSH 端口" class="flex-1">
                    <el-input-number v-model="vpsForm.ssh_port" :min="1" :max="65535" controls-position="right" class="input-full" />
                  </el-form-item>
                  <el-form-item label="SSH 用户" class="flex-1">
                    <el-input v-model="vpsForm.ssh_user" placeholder="root" />
                  </el-form-item>
                </div>
                <el-form-item label="root 密码" required>
                  <el-input v-model="vpsForm.ssh_pass" type="password" show-password placeholder="VPS root 密码（AES 加密存储，用于后续远程管理）" />
                </el-form-item>
              </el-form>
              <div class="subscription-tip">
                部署过程全自动：连接 SSH → 下载 sing-box → 生成节点 → 启动服务 → 自动回传。密码使用 SECRET_KEY 加密保存，仅用于节点管理。
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>
        <el-form 
          v-if="editingNode" 
          :model="nodeForm" 
          :rules="rules" 
          ref="nodeFormRef"
          :label-position="isMobile ? 'top' : 'right'" 
          label-width="100px"
          class="node-form"
        >
          <el-form-item label="节点名称" prop="name">
            <el-input v-model="nodeForm.name" placeholder="请输入节点名称" />
          </el-form-item>
          <el-form-item label="显示名称" prop="display_name">
            <el-input v-model="nodeForm.display_name" placeholder="客户端显示的名称 (可选)" />
          </el-form-item>
          <el-form-item label="节点类型" prop="protocol">
            <el-select v-model="nodeForm.protocol" placeholder="选择节点类型" class="full-width-control">
              <el-option-group v-for="group in nodeTypeGroups" :key="group.label" :label="group.label">
                <el-option
                  v-for="type in group.options"
                  :key="type.value"
                  :label="type.label"
                  :value="type.value"
                />
              </el-option-group>
            </el-select>
          </el-form-item>
          <template v-if="editingNode">
            <el-form-item label="配置(JSON)" prop="config">
              <el-input 
                v-model="nodeForm.config" 
                type="textarea" 
                :rows="6" 
                placeholder='{"server":"1.2.3.4", "port":443, ...}'
                class="code-input"
              />
            </el-form-item>
          </template>
          <el-form-item label="到期时间" prop="expire_time">
             <el-date-picker
                v-model="nodeForm.expire_time"
                type="datetime"
                placeholder="永久有效"
                class="full-width-control"
                format="YYYY-MM-DD HH:mm"
                value-format="YYYY-MM-DDTHH:mm:ssZ"
              />
          </el-form-item>
          <el-form-item label="跟随用户" prop="follow_user_expire">
            <el-switch v-model="nodeForm.follow_user_expire" />
            <span class="form-tip-text">节点到期时间将与被分配用户的订阅时间同步</span>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <FormActionBar
          v-if="!editingNode && addNodeTab === 'link'"
          :loading="saving"
          :disabled="!nodeLinkInput"
          submit-text="批量导入"
          cancel-text="取消"
          @cancel="showAddDialog = false"
          @submit="batchImportLinks"
        >
          <template #left>
            <el-button type="warning" plain @click="parseNodeLink" :loading="parsing">仅解析</el-button>
          </template>
        </FormActionBar>
        <FormActionBar
          v-if="!editingNode && addNodeTab === 'subscription'"
          :loading="importingSubscription"
          :disabled="!subUrlInput"
          submit-text="导入订阅"
          cancel-text="取消"
          @cancel="showAddDialog = false"
          @submit="importSubscription"
        />
        <FormActionBar
          v-if="!editingNode && addNodeTab === 'vps'"
          :loading="deployingVPS"
          :disabled="!vpsForm.name || !vpsForm.ssh_host || !vpsForm.ssh_pass"
          submit-text="一键自动搭建"
          cancel-text="取消"
          @cancel="showAddDialog = false"
          @submit="deploySelfHostVPSNode"
        />
        <FormActionBar
          v-if="editingNode"
          :loading="saving"
          submit-text="保存"
          @cancel="showAddDialog = false"
          @submit="saveNode"
        />
      </template>
    </AppDrawer>
    <AppDialog
      v-model="showLinkDialog"
      title="节点链接"
      width="500px"
      mobile-width="92%"
      :loading="testingFromLink"
    >
      <div v-if="nodeLink" class="link-view-content">
        <el-input
          v-model="nodeLink.link"
          type="textarea"
          :rows="5"
          readonly
          class="code-input"
        />
      </div>
      <template #footer>
        <FormActionBar
          :loading="testingFromLink"
          submit-text="测试连接"
          cancel-text="复制链接"
          @cancel="copyLink"
          @submit="testNodeFromLink"
        />
      </template>
    </AppDialog>
    <AppDrawer
      v-model="showAssignDialog"
      :title="assignMode === 'single' ? '分配节点' : '批量分配'"
      size="760px"
      mobile-size="100%"
      class="assign-drawer"
      :loading="batchAssigning || batchUnassigning"
    >
      <div class="dialog-scroll-content">
        <div v-if="assignMode === 'single'" class="assigned-section">
          <div class="section-header">
            <span>已分配用户</span>
            <el-button type="danger" link size="small" :disabled="!assignedUsers.length" @click="batchUnassignAssignedUsers">
              批量取消
            </el-button>
          </div>
          <el-table 
            v-if="!isMobile" 
            :data="assignedUsers" 
            size="small" 
            empty-text="暂无分配"
            class="assigned-users-table"
          >
            <el-table-column prop="username" label="用户" />
            <el-table-column prop="email" label="邮箱" show-overflow-tooltip />
            <el-table-column label="到期">
              <template #default="{ row }">
                <span :class="{'text-danger': isExpired(row.special_node_expires_at)}">
                  {{ formatTime(row.special_node_expires_at) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="80">
              <template #default="{ row }">
                <el-button type="danger" link size="small" @click="handleUnassign(row)">移除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div v-else class="mobile-assigned-list">
             <div v-for="u in assignedUsers" :key="u.id" class="mini-user-card">
               <div class="u-info">
                 <div class="u-name">{{ u.username }}</div>
                 <div class="u-time" :class="{'text-danger': isExpired(u.special_node_expires_at)}">
                   {{ formatTime(u.special_node_expires_at) }}
                 </div>
               </div>
               <el-button type="danger" circle size="small" icon="Close" @click="handleUnassign(u)" />
             </div>
             <EmptyState
               v-if="!assignedUsers.length"
               title="暂无分配"
               description="该专线节点还没有分配给任何用户"
               :icon-size="56"
               class="compact-empty-state"
             />
          </div>
        </div>
        <div class="assign-form">
          <div class="section-header">新增分配</div>
          <div class="search-user-row">
            <el-input 
              v-model="userSearchKeyword" 
              placeholder="搜索用户名/邮箱..." 
              clearable
              @keyup.enter="handleUserSearch"
            >
              <template #append>
                <el-button @click="handleUserSearch" :loading="loadingUsers">
                  <el-icon><Search /></el-icon>
                  <span class="search-button-text">搜索</span>
                </el-button>
              </template>
            </el-input>
          </div>
          <el-select
            v-model="selectedUserIds"
            multiple
            placeholder="请从搜索结果中选择用户"
            class="user-select"
            no-data-text="请先搜索"
          >
            <el-option
              v-for="user in searchedUsers"
              :key="user.id"
              :label="`${user.username} (${user.email})`"
              :value="user.id"
            />
          </el-select>
          <el-form label-position="top" size="small">
             <el-form-item label="节点显示模式">
               <el-radio-group v-model="assignExtraData.subscription_type" class="option-radio-group">
                 <el-radio-button label="both">专线 + 普通节点</el-radio-button>
                 <el-radio-button label="special_only">仅专线</el-radio-button>
               </el-radio-group>
               <div class="toggle-desc">{{ subscriptionTypeDesc }}</div>
             </el-form-item>
             <el-form-item label="设备数量限制">
               <el-radio-group v-model="assignExtraData.device_limit_mode" class="option-radio-group">
                 <el-radio-button label="system">跟随系统</el-radio-button>
                 <el-radio-button label="unlimited">不限制</el-radio-button>
               </el-radio-group>
               <div class="toggle-desc">{{ deviceLimitDesc }}</div>
             </el-form-item>
             <el-form-item label="专线到期 (可选)">
                <el-date-picker
                  v-model="assignExtraData.expires_at"
                  type="datetime"
                  placeholder="默认跟随用户订阅"
                  class="full-width-control"
                  value-format="YYYY-MM-DDTHH:mm:ssZ"
                />
             </el-form-item>
          </el-form>
        </div>
      </div>
      <template #footer>
        <FormActionBar
          :loading="batchAssigning"
          :disabled="!selectedUserIds.length"
          submit-text="确定分配"
          @cancel="showAssignDialog = false"
          @submit="handleAssign"
        />
      </template>
    </AppDrawer>
    <AppDrawer
      v-model="showMigrateDialog"
      title="迁移分配"
      size="520px"
      mobile-size="100%"
      class="assign-drawer"
      :loading="migratingAssignments"
    >
      <div class="dialog-scroll-content">
        <div class="migrate-summary" v-if="migratingNode">
          <div class="summary-label">源专线节点</div>
          <div class="summary-title">{{ migratingNode.name }}</div>
          <div class="summary-meta">会把该节点当前已分配用户迁移到新的专线节点。</div>
        </div>
        <el-form label-position="top" size="small">
          <el-form-item label="目标专线节点">
            <el-select
              v-model="migrateTargetNodeId"
              filterable
              placeholder="选择新的专线节点"
              class="full-width-control"
            >
              <el-option
                v-for="node in migrateTargetNodes"
                :key="node.id"
                :label="node.name"
                :value="node.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-checkbox v-model="deactivateSourceAfterMigrate">迁移后停用源节点</el-checkbox>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <FormActionBar
          :loading="migratingAssignments"
          :disabled="!migrateTargetNodeId"
          submit-text="确认迁移"
          @cancel="showMigrateDialog = false"
          @submit="migrateAssignments"
        />
      </template>
    </AppDrawer>
  </div>
</template>
<script>
import { ref, reactive, onMounted, computed, watch, onActivated} from 'vue'
import { ElMessage, ElMessageBox } from '@/utils/elementPlusServices'
import { 
  Plus, Refresh, Search, Connection, Delete, 
  DocumentCopy, Edit, MoreFilled, User, Link,
  ArrowDown, Close, Setting
} from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
import { formatDateTimeSafe } from '@/utils/date'
import { formatFileSize } from '@/utils/format'
import { copyToClipboard } from '@/utils/textSelection'
import { confirmDelete, confirmWarning, confirmAction } from '@/utils/confirmAction'
import { usePersistentTableColumns } from '@/composables/usePersistentTableColumns'
import PaginationBar from '@/components/PaginationBar.vue'
import AppDrawer from '@/components/AppDrawer.vue'
import AppDialog from '@/components/AppDialog.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import EmptyState from '@/components/EmptyState.vue'
import { useMobile } from '@/composables/useMobile'
import { debounce } from '@/composables/useDebounce'
import { getCustomNodeStatusType, getCustomNodeStatusText } from '@/utils/statusMaps'
export default {
  name: 'AdminCustomNodes',
  components: { 
    Plus, Refresh, Search, Connection, Delete, 
    DocumentCopy, Edit, MoreFilled, User, Link, ArrowDown, Close,
    PaginationBar,
    AppDrawer,
    AppDialog,
    FormActionBar,
    EmptyState
  },
  setup() {
    const isMobile = useMobile()
    const viewMode = ref('table') // 'table' | 'grid'
    const gridOrientation = ref('horizontal') // 'horizontal' | 'vertical'
    const gridColumns = ref(3) // 2-6 columns for horizontal
    const gridSize = ref('medium') // 'small' | 'medium' | 'large' for vertical
    const tableRef = ref(null)
    const loading = ref(false)
    
    const defaultColumnWidths = {
      selection: 50,
      name: 140,
      display_name: 120,
      protocol: 100,
      server_ip: 160,
      status: 100,
      is_active: 80,
      expire_time: 150,
      actions: 440  // 增加操作列宽度以容纳更多按钮
    }
    const columnKeys = Object.keys(defaultColumnWidths)
    const { columnWidths, handleColumnResize } = usePersistentTableColumns(
      'customNodes_table_column_widths',
      defaultColumnWidths,
      columnKeys
    )
    
    // 从 localStorage 加载设置
    const STORAGE_KEY = 'customNodes_table_settings'
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
    
    // 保存设置到 localStorage
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
    
    const saving = ref(false)
    const parsing = ref(false)
    const importingSubscription = ref(false)
    const subUrlInput = ref('')
    const customNodes = ref([])
    const selectedNodes = ref([])
    const showAddDialog = ref(false)
    const showLinkDialog = ref(false)
    const showAssignDialog = ref(false)
    const addNodeTab = ref('link')
    const nodeTypeGroups = [
      {
        label: '代理协议',
        options: [
          { label: 'VMess', value: 'vmess' },
          { label: 'VLESS', value: 'vless' },
          { label: 'Trojan', value: 'trojan' },
          { label: 'Shadowsocks (SS)', value: 'ss' },
          { label: 'ShadowsocksR (SSR)', value: 'ssr' }
        ]
      },
      {
        label: '现代协议',
        options: [
          { label: 'Hysteria', value: 'hysteria' },
          { label: 'Hysteria2', value: 'hysteria2' },
          { label: 'TUIC', value: 'tuic' },
          { label: 'Naive', value: 'naive' },
          { label: 'AnyTLS', value: 'anytls' }
        ]
      },
      {
        label: '其他协议',
        options: [
          { label: 'SOCKS', value: 'socks' },
          { label: 'SOCKS5', value: 'socks5' },
          { label: 'HTTP', value: 'http' },
          { label: 'HTTPS', value: 'https' },
          { label: 'WireGuard (WG)', value: 'wireguard' }
        ]
      }
    ]
    const nodeTypeLabels = nodeTypeGroups.reduce((acc, group) => {
      group.options.forEach(option => { acc[option.value] = option.label })
      return acc
    }, {})
    const searchKeyword = ref('')
    const filters = reactive({ status: '', protocol: '', is_active: '', source: '' })
    const pagination = reactive({ page: 1, size: 10, total: 0 })
    const nodeFormRef = ref(null)
    const nodeForm = reactive({
      name: '', display_name: '', protocol: 'vmess', config: '', 
      expire_time: null, follow_user_expire: false
    })
    const nodeLinkInput = ref('')
    const parsedNode = ref(null)
    const nodeLink = ref(null)
    const testingFromLink = ref(false)
    const assignMode = ref('single') // single | batch
    const assigningNode = ref(null)
    const assignedUsers = ref([])
    const userSearchKeyword = ref('')
    const searchedUsers = ref([])
    const selectedUserIds = ref([])
    const loadingUsers = ref(false)
    const batchAssigning = ref(false)
    const assignExtraData = reactive({
      subscription_type: 'both',
      device_limit_mode: 'system',
      expires_at: null
    })
    const subscriptionTypeDesc = computed(() => (
      assignExtraData.subscription_type === 'special_only'
        ? '用户订阅里只显示已分配的专线节点。'
        : '用户订阅里同时显示普通节点和已分配专线节点。'
    ))
    const deviceLimitDesc = computed(() => (
      assignExtraData.device_limit_mode === 'unlimited'
        ? '专线用户不受设备数量限制。'
        : '专线用户仍按系统套餐设备数限制。'
    ))
    const batchTesting = ref(false)
    const batchDeleting = ref(false)
    const batchUnassigning = ref(false)
    const showMigrateDialog = ref(false)
    const migratingNode = ref(null)
    const migrateTargetNodeId = ref(null)
    const migrateTargetNodes = ref([])
    const deactivateSourceAfterMigrate = ref(true)
    const migratingAssignments = ref(false)
    const rules = {
      name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
      protocol: [{ required: true, message: '请选择节点类型', trigger: 'change' }],
      config: [{ required: true, message: '请输入配置', trigger: 'blur' }]
    }
    const getProtocolLabel = (protocol) => nodeTypeLabels[protocol] || protocol || '-'
    const getSourceText = (source) => ({
      manual: '手动添加',
      link: '链接导入',
      subscription: '订阅导入'
    }[source] || '手动添加')
    const getSourceTagType = (source) => ({
      manual: 'info',
      link: 'primary',
      subscription: 'success'
    }[source] || 'info')
    const parseConfigValue = (config) => {
      if (!config) return null
      if (typeof config === 'object') return config
      try {
        return JSON.parse(config)
      } catch {
        return null
      }
    }
    const getNodeServer = (node) => {
      if (!node) return ''
      if (node.domain) return node.domain
      const config = parseConfigValue(node.config)
      return config?.server || config?.Server || config?.add || ''
    }
    const loadCustomNodes = async () => {
      loading.value = true
      try {
        const params = {
          page: pagination.page, size: pagination.size,
          ...filters, search: searchKeyword.value,
          _t: Date.now() // 缓存穿透参数，避免浏览器/代理返回旧列表
        }
        for (const key in params) { if (!params[key]) delete params[key] }
        const res = await adminAPI.getCustomNodes(params)
        if (res.data?.success !== false) {
          const raw = res.data.data
          const list = Array.isArray(raw) ? raw : (raw.data || [])
          customNodes.value = list.map(n => ({...n, testing: false}))
          pagination.total = raw.total || list.length
          // 删除后当前页可能超出总页数（如删空最后一页），自动回退到有效页并重新加载
          const maxPage = Math.max(1, Math.ceil(pagination.total / pagination.size))
          if (pagination.page > maxPage) {
            pagination.page = maxPage
            return loadCustomNodes()
          }
        } else {
          customNodes.value = []
        }
      } catch (e) {
        ElMessage.error('加载失败')
      } finally {
        loading.value = false
      }
    }
    const handleFilterChange = () => {
      pagination.page = 1
      loadCustomNodes()
    }
    // 搜索输入实时生效，无需再点击搜索按钮（500ms 防抖）
    const debouncedSearch = debounce(handleFilterChange, 500)
    const resetFilters = () => {
      Object.assign(filters, { status: '', protocol: '', is_active: '', source: '' })
      searchKeyword.value = ''
      pagination.page = 1
      loadCustomNodes()
    }
    const handleMobileSelect = (node, checked) => {
      if (checked) {
        if (!selectedNodes.value.find(n => n.id === node.id)) selectedNodes.value.push(node)
      } else {
        selectedNodes.value = selectedNodes.value.filter(n => n.id !== node.id)
      }
    }
    const handleGridSelect = (node, checked) => {
      handleMobileSelect(node, checked)
    }
    const isSelected = (node) => selectedNodes.value.some(n => n.id === node.id)
    const isAllSelected = computed({
      get: () => customNodes.value.length > 0 && selectedNodes.value.length === customNodes.value.length,
      set: (val) => selectedNodes.value = val ? [...customNodes.value] : []
    })
    const isIndeterminate = computed(() => selectedNodes.value.length > 0 && selectedNodes.value.length < customNodes.value.length)
    const toggleMobileSelectAll = (val) => selectedNodes.value = val ? [...customNodes.value] : []
    const handleSelectionChange = (val) => selectedNodes.value = val
    const editingNode = ref(null)
    const resetNodeForm = () => {
      Object.assign(nodeForm, {
        name: '',
        display_name: '',
        protocol: 'vmess',
        config: '',
        expire_time: null,
        follow_user_expire: false
      })
    }
    const openCreateNodeDialog = () => {
      editingNode.value = null
      addNodeTab.value = 'link'
      nodeLinkInput.value = ''
      subUrlInput.value = ''
      subReplaceMode.value = false
      parsedNode.value = null
      Object.assign(vpsForm, { name: '', ssh_host: '', ssh_pass: '' })
      resetNodeForm()
      loadSubscriptionList()
      showAddDialog.value = true
    }
    const handleNodeDrawerClosed = () => {
      editingNode.value = null
      resetNodeForm()
      nodeLinkInput.value = ''
      subUrlInput.value = ''
      parsedNode.value = null
    }
    const editNode = (node) => {
      editingNode.value = node
      Object.assign(nodeForm, {
        name: node.name,
        display_name: node.display_name,
        protocol: node.protocol,
        config: typeof node.config === 'object' ? JSON.stringify(node.config) : node.config,
        expire_time: node.expire_time,
        follow_user_expire: node.follow_user_expire
      })
      showAddDialog.value = true
    }
    const saveNode = async () => {
      if (!nodeFormRef.value) return
      await nodeFormRef.value.validate(async (valid) => {
        if (!valid) return
        saving.value = true
        try {
          const payload = { ...nodeForm }
          if (editingNode.value) await adminAPI.updateCustomNode(editingNode.value.id, payload)
          else await adminAPI.createCustomNode(payload)
          ElMessage.success('保存成功')
          showAddDialog.value = false
          loadCustomNodes()
        } catch (e) {
          ElMessage.error('保存失败: ' + e.message)
        } finally {
          saving.value = false
        }
      })
    }
    const deleteNode = async (node) => {
      try {
        // 检查受影响用户，生成更明确的提示
        let warningMsg = `确认删除 "${node.name}"?`
        try {
          const usersRes = await adminAPI.getCustomNodeUsers(node.id)
          if (usersRes?.data?.success) {
            const users = usersRes.data.data || []
            if (users.length > 0) {
              const specialOnlyUsers = users.filter(u => u.special_node_subscription_type === 'special_only')
              if (specialOnlyUsers.length > 0) {
                const names = specialOnlyUsers.map(u => u.username).join('、')
                warningMsg = `该节点有 ${users.length} 个用户，其中 ${specialOnlyUsers.length} 个开启了「仅专线显示」：${names}。\n\n删除后系统将自动恢复其普通线路访问。\n\n确认删除 "${node.name}"?`
              } else {
                warningMsg = `该节点有 ${users.length} 个用户正在使用。\n\n删除后若用户无其他专线节点，系统将自动恢复其普通线路访问。\n\n确认删除 "${node.name}"?`
              }
            }
          }
        } catch (e) { /* 获取用户列表失败，使用默认提示 */ }
        await confirmDelete('专线节点', 1, {
          message: warningMsg,
          title: '删除专线节点'
        })
        await adminAPI.deleteCustomNode(node.id)
        ElMessage.success('已删除')
        // 先从本地列表立即移除，再向服务端重新拉取校准，保证页面马上不显示已删除节点
        customNodes.value = customNodes.value.filter(n => n.id !== node.id)
        if (pagination.total > 0) pagination.total -= 1
        selectedNodes.value = selectedNodes.value.filter(n => n.id !== node.id)
        tableRef.value?.clearSelection()
        loadCustomNodes()
      } catch (error) {
        if (error !== 'cancel') ElMessage.error('删除节点失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const toggleNodeStatus = async (node) => {
      try {
        await adminAPI.updateCustomNode(node.id, { is_active: node.is_active })
        ElMessage.success(node.is_active ? '已启用' : '已禁用')
      } catch {
        node.is_active = !node.is_active
        ElMessage.error('操作失败')
      }
    }
    const handleCommand = (cmd) => {
      if (cmd === 'refresh') loadCustomNodes()
      if (cmd === 'batch_test') batchTest()
      if (cmd === 'batch_assign') handleBatchAssignClick()
      if (cmd === 'batch_unassign') batchUnassign()
      if (cmd === 'migrate_assignments' && selectedNodes.value.length === 1) openMigrateDialog(selectedNodes.value[0])
      if (cmd === 'batch_delete') batchDelete()
    }
    const batchTest = async () => {
      if (!selectedNodes.value.length) return
      batchTesting.value = true
      try {
        await adminAPI.batchTestCustomNodes(selectedNodes.value.map(n => n.id))
        ElMessage.success('批量测试请求已发送')
        setTimeout(loadCustomNodes, 1000)
      } catch { ElMessage.error('测试请求失败') }
      finally { batchTesting.value = false }
    }
    const batchDelete = async () => {
      if (!selectedNodes.value.length) return
      try {
        batchDeleting.value = true
        // 检查受影响用户：一次批量请求拿全部节点的用户，避免逐个串行请求导致确认框延迟弹出
        let warningMsg = `确认删除选中的 ${selectedNodes.value.length} 个节点?`
        try {
          const usersRes = await adminAPI.batchGetCustomNodeUsers(selectedNodes.value.map(n => n.id))
          if (usersRes?.data?.success) {
            const nodeUsers = usersRes.data.data || {}
            let totalUsers = 0
            const specialOnlyUsers = []
            for (const node of selectedNodes.value) {
              const users = nodeUsers[node.id] || []
              totalUsers += users.length
              users.filter(u => u.special_node_subscription_type === 'special_only').forEach(u => {
                if (!specialOnlyUsers.find(s => s.id === u.id)) specialOnlyUsers.push(u)
              })
            }
            if (totalUsers > 0) {
              if (specialOnlyUsers.length > 0) {
                const names = specialOnlyUsers.slice(0, 5).map(u => u.username).join('、')
                const more = specialOnlyUsers.length > 5 ? `等${specialOnlyUsers.length}人` : ''
                warningMsg = `选中节点共有 ${totalUsers} 个用户，其中 ${specialOnlyUsers.length} 个开启了「仅专线显示」：${names}${more}。\n\n删除后系统将自动恢复其普通线路访问。\n\n确认删除 ${selectedNodes.value.length} 个节点?`
              } else {
                warningMsg = `选中节点共有 ${totalUsers} 个用户正在使用。\n\n删除后若用户无其他专线节点，系统将自动恢复其普通线路访问。\n\n确认删除 ${selectedNodes.value.length} 个节点?`
              }
            }
          }
        } catch (e) { /* 获取用户列表失败，使用默认提示 */ }
        await confirmDelete('专线节点', selectedNodes.value.length, {
          message: warningMsg,
          title: '批量删除专线节点'
        })
        const deletedIds = selectedNodes.value.map(n => n.id)
        await adminAPI.batchDeleteCustomNodes(deletedIds)
        ElMessage.success('批量删除成功')
        // 先从本地列表立即移除已删除节点，再向服务端重新拉取校准，
        // 避免因浏览器/代理缓存导致重新加载后仍显示旧节点
        customNodes.value = customNodes.value.filter(n => !deletedIds.includes(n.id))
        pagination.total = Math.max(0, pagination.total - deletedIds.length)
        selectedNodes.value = []
        tableRef.value?.clearSelection()
        loadCustomNodes()
      } catch (error) {
        if (error !== 'cancel') ElMessage.error('批量删除失败: ' + (error.response?.data?.message || error.message))
      } finally { batchDeleting.value = false }
    }
    const batchUnassign = async () => {
      if (!selectedNodes.value.length) return
      try {
        let totalUsers = 0
        const affectedUsers = []
        // 一次批量请求拿全部节点的用户，避免逐个串行请求导致确认框延迟弹出
        const usersRes = await adminAPI.batchGetCustomNodeUsers(selectedNodes.value.map(n => n.id))
        if (usersRes?.data?.success) {
          const nodeUsers = usersRes.data.data || {}
          for (const node of selectedNodes.value) {
            const users = nodeUsers[node.id] || []
            totalUsers += users.length
            users.forEach(u => {
              if (!affectedUsers.find(item => item.id === u.id)) affectedUsers.push(u)
            })
          }
        }
        if (!totalUsers) {
          ElMessage.info('选中的节点暂无分配用户')
          return
        }
        const warningMsg = `将取消 ${selectedNodes.value.length} 个专线节点的分配关系，影响 ${affectedUsers.length} 个用户。\n\n如果某个用户取消后没有其他专线节点，系统会自动恢复普通线路访问。\n\n确认继续？`
        await confirmWarning(warningMsg, {
          title: '批量取消分配',
          confirmButtonText: '确认取消分配'
        })
        batchUnassigning.value = true
        const res = await adminAPI.batchUnassignCustomNodes(selectedNodes.value.map(n => n.id))
        ElMessage.success(res.data?.message || '批量取消成功')
        if (assigningNode.value) loadAssignedUsers(assigningNode.value.id)
      } catch (error) {
        if (error !== 'cancel') ElMessage.error('批量取消失败: ' + (error.response?.data?.message || error.message))
      } finally {
        batchUnassigning.value = false
      }
    }
    const parseNodeLink = async () => {
      const link = nodeLinkInput.value.split('\n')[0].trim()
      if (!link) return
      parsing.value = true
      try {
        const res = await adminAPI.createCustomNode({ node_link: link, preview: true })
        if (res.data.success) {
           const data = res.data.data
           parsedNode.value = { 
             name: data.name, 
             server: typeof data.config === 'object' ? data.config.server : 'unknown',
             port: typeof data.config === 'object' ? data.config.port : 'unknown'
           }
        }
      } finally { parsing.value = false }
    }
    const batchImportLinks = async () => {
      const links = nodeLinkInput.value.split('\n').map(l=>l.trim()).filter(Boolean)
      if (!links.length) return
      saving.value = true
      try {
        const res = await adminAPI.importCustomNodeLinks(links)
        const imported = res.data.data?.imported ?? res.data.imported ?? 0
        const errorCount = res.data.data?.error_count ?? res.data.error_count ?? 0
        const errors = res.data.data?.errors ?? []
        if (imported > 0) {
          ElMessage.success(`导入成功: ${imported} 个${errorCount ? `，失败 ${errorCount} 个` : ''}`)
          showAddDialog.value = false
          loadCustomNodes()
        } else if (errorCount > 0) {
          // 全部失败时保留弹窗，方便用户修改链接后重试
          ElMessage.error(`导入失败 ${errorCount} 个：${errors[0] || '未知原因'}`)
        } else {
          ElMessage.warning('没有解析到可导入的节点')
          showAddDialog.value = false
        }
      } catch { ElMessage.error('导入失败') }
      finally { saving.value = false }
    }
    const importSubscription = async () => {
      const url = (subUrlInput.value || '').trim()
      if (!url) {
        ElMessage.warning('请输入订阅链接')
        return
      }
      importingSubscription.value = true
      try {
        // 替换模式：更新该订阅地址下原有节点
        const res = subReplaceMode.value
          ? await adminAPI.updateCustomNodeSubscription(url)
          : await adminAPI.importCustomNodeSubscription(url)
        const data = res.data?.data ?? {}
        const imported = data.imported ?? 0
        const updated = data.updated ?? 0
        const removed = data.removed ?? 0
        const kept = data.kept ?? 0
        const total = data.total ?? imported
        if (subReplaceMode.value) {
          if (imported > 0 || updated > 0 || kept > 0) {
            ElMessage.success(res.data?.message || `订阅更新完成：新增 ${imported} 个，更新 ${updated} 个，保留 ${kept} 个`)
            showAddDialog.value = false
            loadCustomNodes()
            loadSubscriptionList()
          } else {
            ElMessage.warning(res.data?.message || '订阅中没有解析到可更新的节点')
          }
        } else if (imported > 0) {
          ElMessage.success(`订阅解析出 ${total} 个节点，成功导入 ${imported} 个`)
          showAddDialog.value = false
          loadCustomNodes()
          loadSubscriptionList()
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
    // ==================== 已导入订阅管理（更新 / 删除） ====================
    const subscriptionList = ref([])
    const subReplaceMode = ref(false)
    // isLegacySubscription 判断是否为历史导入节点（旧版数据无原始订阅地址，用占位 URL 标记）
    const isLegacySubscription = (url) => {
      return typeof url === 'string' && url.startsWith('legacy://')
    }
    const loadSubscriptionList = async () => {
      try {
        const res = await adminAPI.getCustomNodeSubscriptions()
        if (res.data?.success) {
          subscriptionList.value = res.data.data?.list || []
        }
      } catch (e) {
        console.warn('加载已导入订阅失败', e)
      }
    }
    // handleReplaceLegacy 历史订阅节点：引导用户输入新订阅地址 + 替换模式，整体替换旧节点
    const handleReplaceLegacy = (sub) => {
      addNodeTab.value = 'subscription'
      subReplaceMode.value = true
      subUrlInput.value = ''
      ElMessage.info('请在下方输入新的订阅链接，将以「替换模式」导入，整体替换历史节点')
    }
    const handleUpdateSubscription = async (sub) => {
      try {
        await ElMessageBox.confirm(
          `将重新拉取订阅并替换该订阅下的全部节点（当前 ${sub.node_count} 个旧节点会被移除，按最新内容重建）。确定继续？`,
          '更新订阅',
          { type: 'warning', confirmButtonText: '更新', cancelButtonText: '取消' }
        )
      } catch {
        return
      }
      importingSubscription.value = true
      try {
        const res = await adminAPI.updateCustomNodeSubscription(sub.url, false)
        const data = res.data?.data ?? {}
        const imported = data.imported ?? 0
        const updated = data.updated ?? 0
        const kept = data.kept ?? 0
        if (res.data?.success && (imported > 0 || updated > 0 || kept > 0)) {
          ElMessage.success(res.data.message || '订阅更新完成')
          loadCustomNodes()
          loadSubscriptionList()
        } else {
          ElMessage.warning(res.data?.message || '订阅中没有解析到可更新的节点')
        }
      } catch (e) {
        ElMessage.error('更新订阅失败: ' + (e.response?.data?.message || e.message))
      } finally {
        importingSubscription.value = false
      }
    }
    const handleDeleteSubscription = async (sub) => {
      try {
        await ElMessageBox.confirm(
          `将删除该订阅导入的全部 ${sub.node_count} 个专线节点（含已分配用户的关联）。确定继续？`,
          '删除订阅节点',
          { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
        )
      } catch {
        return
      }
      try {
        const res = await adminAPI.deleteCustomNodeSubscription(sub.url)
        if (res.data?.success) {
          ElMessage.success(res.data.message || '删除成功')
          loadCustomNodes()
          loadSubscriptionList()
        } else {
          ElMessage.error(res.data?.message || '删除失败')
        }
      } catch (e) {
        ElMessage.error('删除订阅节点失败: ' + (e.response?.data?.message || e.message))
      }
    }
    // ==================== VPS 自动搭建 ====================
    const deployingVPS = ref(false)
    const vpsForm = reactive({
      name: '', protocol: 'vless-ws', ssh_host: '', ssh_port: 22, ssh_user: 'root', ssh_pass: ''
    })
    const deploySelfHostVPSNode = async () => {
      deployingVPS.value = true
      try {
        const res = await adminAPI.deploySelfHostVPS({ ...vpsForm })
        if (res.data?.success) {
          ElMessage.success(res.data.message || 'VPS 自动搭建完成')
          showAddDialog.value = false
          Object.assign(vpsForm, { name: '', ssh_host: '', ssh_pass: '' })
          loadCustomNodes()
          loadSelfHostNodes()
        } else {
          ElMessage.error(res.data?.message || '搭建失败')
        }
      } catch (e) {
        ElMessage.error('搭建失败: ' + (e.response?.data?.message || e.message))
      } finally {
        deployingVPS.value = false
      }
    }
    // ==================== 自建节点列表（专线列表下方） ====================
    const selfHostNodes = ref([])
    const selfHostLoading = ref(false)
    const loadSelfHostNodes = async () => {
      selfHostLoading.value = true
      try {
        const res = await adminAPI.getSelfHostNodes()
        if (res.data?.success) {
          selfHostNodes.value = res.data.data?.list || []
        }
      } catch (e) {
        console.warn('加载自建节点失败', e)
      } finally {
        selfHostLoading.value = false
      }
    }
    const managingSelfHostId = ref(null)
    const selfHostManage = async (node, action, extra = {}) => {
      managingSelfHostId.value = node.id
      try {
        const res = await adminAPI.selfHostManage(node.id, action, extra)
        if (res.data?.success) {
          ElMessage.success(res.data.message || '操作成功')
          if (action === 'change-port') node.port = extra.new_port
          loadSelfHostNodes()
          loadCustomNodes()
        } else {
          ElMessage.error(res.data?.message || '操作失败')
        }
      } catch (e) {
        ElMessage.error('操作失败: ' + (e.response?.data?.message || e.message))
      } finally {
        managingSelfHostId.value = null
      }
    }
    const onSelfHostManage = async (node, action) => {
      if (action === 'reset') {
        const confirmed = await confirmAction(`确认重置节点「${node.name}」的凭据（重新生成 UUID）？重置后需更新订阅。`)
        if (!confirmed) return
        await selfHostManage(node, 'reset')
      } else if (action === 'change-password') {
        const { value } = await ElMessageBox.prompt('输入新的 UUID（密码）', '更改节点密码', {
          inputPattern: /^[0-9a-fA-F-]{36}$/,
          inputErrorMessage: '请输入合法的 UUID（36位，含横线）'
        }).catch(() => ({}))
        if (!value) return
        await selfHostManage(node, 'change-password', { new_pass: value })
      } else if (action === 'change-port') {
        const { value } = await ElMessageBox.prompt('输入新的监听端口', '更改端口', {
          inputPattern: /^\d+$/,
          inputValidator: (v) => (v >= 1 && v <= 65535) ? true : '端口需在 1-65535',
          inputErrorMessage: '端口需在 1-65535'
        }).catch(() => ({}))
        if (!value) return
        await selfHostManage(node, 'change-port', { new_port: parseInt(value, 10) })
      } else if (action === 'reinstall') {
        const confirmed = await confirmAction(`确认在 VPS「${node.ssh_host || node.name}」上重新搭建节点？会重新安装 sing-box。`)
        if (!confirmed) return
        await selfHostManage(node, 'reinstall')
      } else if (action === 'status') {
        const res = await adminAPI.selfHostManage(node.id, 'status').catch(() => null)
        if (res?.data?.success) {
          ElMessageBox.alert(`<pre style="white-space:pre-wrap;font-size:12px">${res.data.data?.status || '无输出'}</pre>`, `节点「${node.name}」远程状态`, { dangerouslyUseHTMLString: true })
        }
      }
    }
    const selfHostStatusMap = { pending: '等待安装', online: '在线', offline: '离线', expired: '已过期', canceled: '已取消' }
    const selfHostStatusTypeMap = { pending: 'warning', online: 'success', offline: 'danger', expired: 'info', canceled: 'info' }
    const formatBytes2 = (b) => {
      return formatFileSize(b)
    }
    const formatTime2 = (t) => {
      return formatDateTimeSafe(t, 'YYYY-MM-DD HH:mm', '-')
    }
    const viewLink = async (node) => {
      try {
        const res = await adminAPI.getCustomNodeLink(node.id)
        if (res.data.success) {
          nodeLink.value = res.data.data
          showLinkDialog.value = true
        }
      } catch { ElMessage.error('获取链接失败') }
    }
    const copyLink = () => {
      if (nodeLink.value?.link) {
        copyToClipboard(nodeLink.value.link, '已复制')
      }
    }
    const assignSingleNode = (node) => {
      assignMode.value = 'single'
      assigningNode.value = node
      loadAssignedUsers(node.id)
      openAssignDialog()
    }
    const handleBatchAssignClick = () => {
      if (!selectedNodes.value.length) return
      assignMode.value = 'batch'
      assigningNode.value = null
      openAssignDialog()
    }
    const openAssignDialog = () => {
      selectedUserIds.value = []
      userSearchKeyword.value = ''
      searchedUsers.value = []
      showAssignDialog.value = true
    }
    const handleUserSearch = async () => {
      if (!userSearchKeyword.value) return
      loadingUsers.value = true
      try {
        const res = await adminAPI.getUsers({ keyword: userSearchKeyword.value, page: 1, size: 50 })
        searchedUsers.value = res.data.data?.users || []
      } finally { loadingUsers.value = false }
    }
    const handleAssign = async () => {
      batchAssigning.value = true
      try {
        const nodeIds = assignMode.value === 'single' ? [assigningNode.value.id] : selectedNodes.value.map(n => n.id)
        const payload = {
          subscription_type: assignExtraData.subscription_type,
          unlimited_devices: assignExtraData.device_limit_mode === 'unlimited',
          expires_at: assignExtraData.expires_at
        }
        await adminAPI.batchAssignCustomNodes(nodeIds, selectedUserIds.value, payload)
        ElMessage.success('分配成功')
        showAssignDialog.value = false
        if (assignMode.value === 'single') loadAssignedUsers(assigningNode.value.id)
      } catch (e) { ElMessage.error('分配失败: ' + e.message) }
      finally { batchAssigning.value = false }
    }
    const loadAssignedUsers = async (nodeId) => {
      try {
        const res = await adminAPI.getCustomNodeUsers(nodeId)
        assignedUsers.value = res.data.data || []
      } catch (error) {
        ElMessage.error('加载用户列表失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const batchUnassignAssignedUsers = async () => {
      if (!assigningNode.value || !assignedUsers.value.length) return
      try {
        await confirmWarning(`确认取消 ${assignedUsers.value.length} 个用户与「${assigningNode.value.name}」的分配关系？`, {
          title: '批量取消分配',
          confirmButtonText: '确认取消'
        })
        batchUnassigning.value = true
        await adminAPI.batchUnassignCustomNodes([assigningNode.value.id], assignedUsers.value.map(u => u.id))
        ElMessage.success('已批量取消分配')
        await loadAssignedUsers(assigningNode.value.id)
      } catch (error) {
        if (error !== 'cancel') ElMessage.error('批量取消失败: ' + (error.response?.data?.message || error.message))
      } finally {
        batchUnassigning.value = false
      }
    }
    const handleUnassign = async (user) => {
      try {
        await adminAPI.unassignCustomNodeFromUser(user.id, assigningNode.value.id)
        ElMessage.success('已移除')
        loadAssignedUsers(assigningNode.value.id)
      } catch (error) {
        ElMessage.error('移除用户失败: ' + (error.response?.data?.message || error.message))
      }
    }
    const loadMigrateTargetNodes = async (sourceNodeId) => {
      try {
        const res = await adminAPI.getCustomNodes({ page: 1, size: 1000, is_active: 'true' })
        const raw = res.data?.data
        const list = Array.isArray(raw) ? raw : (raw?.data || raw?.nodes || [])
        migrateTargetNodes.value = list.filter(node => node.id !== sourceNodeId)
      } catch (error) {
        migrateTargetNodes.value = customNodes.value.filter(node => node.id !== sourceNodeId)
      }
    }
    const openMigrateDialog = async (node) => {
      if (!node) return
      migratingNode.value = node
      migrateTargetNodeId.value = null
      deactivateSourceAfterMigrate.value = true
      showMigrateDialog.value = true
      await loadMigrateTargetNodes(node.id)
    }
    const migrateAssignments = async () => {
      if (!migratingNode.value || !migrateTargetNodeId.value) return
      const target = migrateTargetNodes.value.find(node => node.id === migrateTargetNodeId.value)
      try {
        await confirmWarning(`确认把「${migratingNode.value.name}」上的用户迁移到「${target?.name || '目标节点'}」？`, {
          title: '迁移专线分配',
          confirmButtonText: '确认迁移'
        })
        migratingAssignments.value = true
        const res = await adminAPI.migrateCustomNodeAssignments(
          migratingNode.value.id,
          migrateTargetNodeId.value,
          { deactivate_source: deactivateSourceAfterMigrate.value }
        )
        ElMessage.success(res.data?.message || '迁移完成')
        showMigrateDialog.value = false
        selectedNodes.value = []
        await loadCustomNodes()
      } catch (error) {
        if (error !== 'cancel') ElMessage.error('迁移失败: ' + (error.response?.data?.message || error.message))
      } finally {
        migratingAssignments.value = false
      }
    }
    const getStatusType = (s) => getCustomNodeStatusType(s)
    const getStatusText = (s) => getCustomNodeStatusText(s)
    const formatExpire = (row) => row.follow_user_expire ? '跟随用户' : formatDateTimeSafe(row.expire_time, 'YYYY-MM-DD HH:mm:ss', '永久')
    const formatTime = (t) => formatDateTimeSafe(t, 'YYYY-MM-DD HH:mm:ss', '跟随订阅')
    const isExpired = (t) => t && new Date(t) < new Date()
    const testNode = async (node) => {
       node.testing = true
       try {
         const res = await adminAPI.testCustomNode(node.id)
         ElMessage.success(`延迟: ${res.data.data.latency}ms`)
         node.status = res.data.data.status
       } catch { ElMessage.error('测试失败') }
       finally { node.testing = false }
    }
    const testNodeFromLink = async () => {
      testingFromLink.value = true
      try {
        const link = nodeLinkInput.value.split('\n')[0].trim()
        if (!link) {
          ElMessage.warning('请先输入节点链接')
          return
        }
        // 链接尚未保存为节点，无法按 ID 测速：先解析确认格式，再引导保存后测试
        if (!parsedNode.value) {
          await parseNodeLink()
          if (!parsedNode.value) {
            ElMessage.error('节点链接解析失败，无法测试')
            return
          }
          ElMessage.info('节点解析成功，请点击"保存"创建节点后，在节点列表执行测速')
          return
        }
        ElMessage.info('节点解析成功，请点击"保存"创建节点后，在节点列表执行测速')
      } finally { testingFromLink.value = false }
    }
    // 监听视图模式和网格设置变化，自动保存
    watch([viewMode, gridOrientation, gridColumns, gridSize], () => {
      saveSettings()
    })
    watch(showAddDialog, (visible) => {
      if (!visible) handleNodeDrawerClosed()
    })

    onMounted(() => {
      loadSettings() // 先加载保存的设置
      loadCustomNodes()
      loadSelfHostNodes()
      loadSubscriptionList()
    })

    // keep-alive 激活时刷新数据（避免显示缓存旧数据）
    onActivated(() => {
      loadCustomNodes()
      loadSubscriptionList()
    })
    return {
      isMobile, viewMode, gridOrientation, gridColumns, gridSize, tableRef, columnWidths, loading, saving, parsing, customNodes, selectedNodes,
      handleColumnResize,
      showAddDialog, showLinkDialog, showAssignDialog, addNodeTab,
      searchKeyword, filters, pagination, nodeForm, nodeFormRef, rules,
      nodeTypeGroups,
      nodeLinkInput, parsedNode, nodeLink, testingFromLink, subUrlInput, importingSubscription,
      subReplaceMode, subscriptionList, handleUpdateSubscription, handleDeleteSubscription, handleReplaceLegacy, isLegacySubscription,
      deployingVPS, vpsForm, deploySelfHostVPSNode,
      selfHostNodes, selfHostLoading, loadSelfHostNodes, onSelfHostManage, managingSelfHostId,
      selfHostStatusMap, selfHostStatusTypeMap, formatBytes2, formatTime2,
      assignMode, assignedUsers, userSearchKeyword, searchedUsers, selectedUserIds,
      loadingUsers, batchAssigning, assignExtraData, subscriptionTypeDesc, deviceLimitDesc,
      batchTesting, batchDeleting, batchUnassigning,
      showMigrateDialog, migratingNode, migrateTargetNodeId, migrateTargetNodes,
      deactivateSourceAfterMigrate, migratingAssignments,
      loadCustomNodes, handleFilterChange, debouncedSearch, resetFilters, handleSelectionChange, handleMobileSelect, handleGridSelect,
      handleCommand, openCreateNodeDialog, handleNodeDrawerClosed, editNode, saveNode, deleteNode, toggleNodeStatus,
      batchTest, batchDelete, batchUnassign, parseNodeLink, batchImportLinks, importSubscription, viewLink, copyLink,
      testNode, testNodeFromLink, assignSingleNode, handleBatchAssignClick, handleAssign,
      handleUserSearch, handleUnassign, batchUnassignAssignedUsers, openMigrateDialog, migrateAssignments,
      getStatusType, getStatusText, getProtocolLabel, getSourceText, getSourceTagType, getNodeServer, formatExpire, formatTime, isExpired,
      isSelected, isAllSelected, isIndeterminate, toggleMobileSelectAll,
      Delete, Edit, Link, Refresh, Connection, User,
      editingNode
    }
  }
}
</script>
<style scoped>
.admin-custom-nodes {
  padding: 12px;
}
@media (max-width: 768px) {
  .admin-custom-nodes {
    padding: 10px;
  }
}
.list-card {
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}
@media (max-width: 768px) {
  .search-box {
    grid-column: 1 / -1;
  }
}
.view-mode-group { margin-right: 8px; }
.grid-orientation-group { margin-right: 8px; }
.grid-size-group { margin-right: 8px; }
.grid-columns-select {
  width: 90px;
  margin-right: 8px;
}
.full-width-control {
  width: 100%;
}
.danger-menu-item {
  color: var(--el-color-danger);
}
.assigned-users-table {
  margin-bottom: 15px;
}
.user-select {
  width: 100%;
  margin: 10px 0;
}
.batch-tip {
  font-size: 13px;
  color: var(--el-color-primary);
  margin-right: auto;
}
.batch-actions-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  margin-bottom: 12px;
  background: var(--el-fill-color-extra-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}
.batch-btns {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
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
  padding: 40px 0;
}
.compact-empty-state {
  min-height: 140px;
  padding: 16px 0;
}
.grid-node-card {
  background: #fff;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  transition: background-color 0.2s, border-color 0.2s;
}
.grid-node-card:hover {
  border-color: var(--el-border-color);
  background: #fbfdff;
}
.grid-node-card.is-selected {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.grid-node-card .gnc-header {
  padding: 12px 16px;
  background: #f8f9fa;
  border-bottom: 1px solid #ebeef5;
  display: flex;
  align-items: center;
  gap: 8px;
}
.grid-node-card .gnc-checkbox { margin-right: 0; flex-shrink: 0; }
.grid-node-card .gnc-title {
  flex: 1;
  font-weight: 600;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.grid-node-card .gnc-body {
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
}
.grid-node-card .gnc-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
}
.grid-node-card .gnc-row .label {
  color: var(--el-text-color-secondary);
  margin-right: 8px;
}
.grid-node-card .gnc-row .value {
  font-weight: 500;
  word-break: break-all;
  text-align: right;
}
.grid-node-card .gnc-footer {
  padding: 10px 16px;
  border-top: 1px solid #f0f2f5;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-wrap: wrap;
}
.table-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.grid-node-card .gnc-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.mobile-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.mobile-selection-bar {
  padding: 0 4px;
  margin-bottom: 4px;
}
.node-card {
  background: #fff;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 12px;
}
.card-header-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px dashed var(--el-border-color-lighter);
}
.node-title {
  font-weight: 600;
  font-size: 15px;
  flex: 1;
  overflow: clip;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.card-checkbox { margin-right: 0; }
.card-info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-bottom: 12px;
}
.info-item {
  display: flex;
  flex-direction: column;
  background: var(--el-fill-color-extra-light);
  padding: 6px;
  border-radius: 4px;
}
.info-item.full-width {
  grid-column: 1 / -1;
}
.info-item .label {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.info-item .value {
  font-size: 13px;
  font-weight: 500;
  word-break: break-all;
}
.card-actions-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-top: 1px solid var(--el-border-color-lighter);
  padding-top: 8px;
}
.right-buttons {
  display: flex;
  gap: 8px;
}
.code-input :deep(.el-textarea__inner) {
  font-family: monospace;
  font-size: 12px;
  background-color: var(--el-fill-color-darker);
  color: var(--el-text-color-primary);
}
.mini-user-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--el-fill-color-light);
  padding: 8px 12px;
  border-radius: 6px;
  margin-bottom: 8px;
}
.u-name { font-weight: 500; font-size: 14px; }
.u-time { font-size: 12px; color: var(--el-text-color-secondary); }
.text-danger { color: var(--el-color-danger); }
.text-secondary { color: var(--el-text-color-secondary); font-size: 12px; }
.text-xs { font-size: 12px; }
.toggle-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.toggle-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 6px;
  line-height: 1.5;
}
.option-radio-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.option-radio-group :deep(.el-radio-button__inner) {
  min-height: 44px;
  border-radius: 6px !important;
  border-left: 1px solid var(--el-border-color) !important;
  display: inline-flex;
  align-items: center;
  touch-action: manipulation;
}
.search-button-text {
  margin-left: 4px;
}
.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-weight: 600;
  margin-bottom: 10px;
}
.assigned-section {
  margin-bottom: 18px;
}
.assign-form {
  padding-top: 4px;
}
.search-user-row :deep(.el-button) {
  min-width: 76px;
}
.migrate-summary {
  padding: 14px;
  margin-bottom: 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-extra-light);
}
.summary-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}
.summary-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.summary-meta {
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}
.dialog-scroll-content {
  max-height: 70vh;
  overflow-y: auto;
  padding-right: 4px;
}
@media (max-width: 768px) {
  .batch-actions-bar {
    display: none;
  }
  .dialog-scroll-content {
    max-height: none;
  }
  .assign-form .el-select {
    width: 100%;
  }
  .option-radio-group {
    display: grid;
    grid-template-columns: 1fr;
    width: 100%;
  }
  .option-radio-group :deep(.el-radio-button) {
    width: 100%;
  }
  .option-radio-group :deep(.el-radio-button__inner) {
    width: 100%;
    min-height: 44px;
    justify-content: center;
  }
  .card-actions-row {
    align-items: stretch;
    gap: 8px;
    flex-wrap: wrap;
  }
  .right-buttons {
    flex: 1;
    justify-content: flex-end;
    flex-wrap: wrap;
  }
  .right-buttons .el-button {
    min-height: 44px;
    touch-action: manipulation;
  }
  .card-actions-row :deep(.el-button) {
    min-height: 44px;
    touch-action: manipulation;
  }
  .search-user-row :deep(.el-button) {
    min-height: 44px;
    touch-action: manipulation;
  }
}

.import-section {
  display: flex;
  flex-direction: column;
  gap: 12px;

  .link-textarea {
    width: 100%;
  }

  .subscription-url-input {
    width: 100%;
  }

  .subscription-tip {
    font-size: 12px;
    color: #909399;
    line-height: 1.5;
  }

  /* 已导入订阅管理 */
  .subscription-manage-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 240px;
    overflow-y: auto;
    padding: 4px;
  }

  .sub-section-label {
    font-size: 13px;
    font-weight: 600;
    color: #303133;
  }

  .subscription-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 8px 10px;
    border: 1px solid #e4e7ed;
    border-radius: 6px;
    background: #f5f7fa;
  }

  .subscription-item-main {
    flex: 1;
    min-width: 0;
  }

  .subscription-item-url {
    font-size: 12px;
    color: #606266;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 100%;
  }

  .subscription-item-count {
    font-size: 12px;
    color: #909399;
    margin-top: 2px;
  }

  .subscription-item-actions {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
  }

  .sub-replace-row {
    display: flex;
    align-items: flex-start;
    gap: 8px;
  }

  .sub-replace-checkbox {
    flex-shrink: 0;
    margin-top: 1px;
  }

  .sub-replace-label {
    font-weight: 500;
    color: #303133;
  }

  .parsed-preview {
    padding: 12px;
    border: 1px solid #e4e7ed;
    border-radius: 6px;
    background: #f5f7fa;

    .preview-title {
      font-size: 13px;
      font-weight: 600;
      color: #303133;
      margin-bottom: 6px;
    }

    .preview-row {
      font-size: 12px;
      color: #606266;
      line-height: 1.8;
      word-break: break-all;

      span {
        color: #909399;
        margin-right: 4px;
      }
    }
  }
}
/* ==================== 自建节点分区 ==================== */
.selfhost-section-card {
  margin-top: 14px;
}
.selfhost-section-body {
  min-height: 60px;
}
.selfhost-node-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 12px 14px;
  margin-bottom: 12px;
  background: var(--el-bg-color);
}
.selfhost-node-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
}
.selfhost-node-title {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.selfhost-node-name {
  font-weight: 600;
  font-size: 15px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.selfhost-node-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px 12px;
}
.selfhost-node-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.detail-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.detail-value {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  word-break: break-all;
}
.vps-form {
  margin-top: 8px;
}
.inline-fields-row {
  display: flex;
  gap: 12px;
}
.inline-fields-row .flex-1 {
  flex: 1;
}
@media (max-width: 768px) {
  .selfhost-node-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .inline-fields-row {
    flex-direction: column;
    gap: 0;
  }
}
</style>
