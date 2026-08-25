<template>
  <div class="list-container admin-selfhost-nodes">
    <el-card class="list-card" shadow="never">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <span class="title-text">自建节点</span>
            <el-tag v-if="selfHostNodes.length" type="info" round size="small" class="count-tag">{{ selfHostNodes.length }}</el-tag>
          </div>
          <div class="header-actions" v-if="!isMobile">
            <el-button type="primary" @click="openVpsDialog">
              <el-icon><Promotion /></el-icon>VPS自动搭建
            </el-button>
            <el-button type="warning" plain @click="openManualDialog">
              <el-icon><DocumentCopy /></el-icon>手动搭建
            </el-button>
            <el-dropdown trigger="click" @command="onBatchCommand" :disabled="!selectedSelfHost.length">
              <el-button type="danger" plain :disabled="!selectedSelfHost.length">
                <el-icon><Operation /></el-icon>批量操作<el-icon><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="reset">批量重置UUID</el-dropdown-item>
                  <el-dropdown-item command="change-password">批量改密码</el-dropdown-item>
                  <el-dropdown-item command="change-port">批量改端口（随机）</el-dropdown-item>
                  <el-dropdown-item command="traffic-limit" divided>批量设置流量配额</el-dropdown-item>
                  <el-dropdown-item command="reset-traffic">批量清零流量</el-dropdown-item>
                  <el-dropdown-item command="enable" divided>批量启用</el-dropdown-item>
                  <el-dropdown-item command="disable">批量禁用</el-dropdown-item>
                  <el-dropdown-item command="delete" divided class="danger-menu-item">批量删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-button @click="loadSelfHostNodes" :loading="selfHostLoading">
              <el-icon><Refresh /></el-icon>刷新
            </el-button>
          </div>
          <div class="header-actions mobile" v-else>
            <el-button type="primary" circle @click="openVpsDialog" size="small" title="VPS自动搭建">
              <el-icon><Promotion /></el-icon>
            </el-button>
            <el-button type="warning" plain circle @click="openManualDialog" size="small" title="手动搭建">
              <el-icon><DocumentCopy /></el-icon>
            </el-button>
            <el-button circle @click="loadSelfHostNodes" size="small" :loading="selfHostLoading">
              <el-icon><Refresh /></el-icon>
            </el-button>
          </div>
        </div>
      </template>
      <div class="batch-toolbar" v-if="selfHostNodes.length">
        <el-checkbox
          :model-value="selfHostNodes.length > 0 && selectedSelfHost.length === selfHostNodes.length"
          :indeterminate="selectedSelfHost.length > 0 && selectedSelfHost.length < selfHostNodes.length"
          @change="toggleSelectAll"
        >全选 ({{ selectedSelfHost.length }}/{{ selfHostNodes.length }})</el-checkbox>
        <el-button v-if="selectedSelfHost.length" size="small" text bg type="danger" @click="onBatchCommand('delete')">删除所选</el-button>
        <el-button v-if="selectedSelfHost.length" size="small" text bg type="primary" @click="onBatchCommand('change-port')">随机改端口</el-button>
        <el-button v-if="selectedSelfHost.length" size="small" text bg @click="onBatchCommand('reset-traffic')">清零流量</el-button>
      </div>
      <div class="content-view" v-loading="selfHostLoading">
        <el-alert
          title="自建节点 = 在您的 VPS 上自动部署代理节点。方式一：填写 VPS 的 IP/SSH端口/root密码 全自动搭建；方式二：复制安装命令到 VPS 手动执行。节点通过心跳维护在线状态，支持远程重置/改密码/改端口/重装。"
          type="info"
          :closable="false"
          show-icon
          class="selfhost-guide"
        />
        <el-empty v-if="!selfHostLoading && selfHostNodes.length === 0" description="暂无自建节点，点击右上角「VPS自动搭建」创建" :image-size="100" />
        <div v-for="n in selfHostNodes" :key="n.id" class="selfhost-node-card">
          <div class="selfhost-node-head">
            <div class="selfhost-node-title">
              <el-checkbox :model-value="isSelected(n)" @change="(v) => toggleSelect(n, v)" class="node-checkbox" />
              <span class="selfhost-node-name">{{ n.name || '-' }}</span>
              <el-tag size="small" :type="selfHostStatusTypeMap[n.status] || 'info'" effect="light">{{ selfHostStatusMap[n.status] || n.status }}</el-tag>
              <el-tag v-if="n.is_active" type="success" size="small" effect="plain">已启用</el-tag>
              <el-tag v-else type="danger" size="small" effect="plain">已屏蔽</el-tag>
            </div>
            <el-dropdown trigger="click" @command="(cmd) => onSelfHostManage(n, cmd)">
              <el-button size="small" text bg :loading="managingSelfHostId === n.id">
                <el-icon><Setting /></el-icon>管理
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <template v-if="n.ssh_host">
                    <el-dropdown-item command="status">查看远程状态</el-dropdown-item>
                    <el-dropdown-item command="reset" divided>重置节点（重新生成UUID）</el-dropdown-item>
                    <el-dropdown-item command="change-password">更改密码</el-dropdown-item>
                    <el-dropdown-item command="change-port">更改端口</el-dropdown-item>
                    <el-dropdown-item command="reinstall" divided class="danger-menu-item">重新搭建</el-dropdown-item>
                    <el-dropdown-item divided command="traffic-limit">设置流量配额</el-dropdown-item>
                    <el-dropdown-item command="reset-traffic">清零流量</el-dropdown-item>
                    <el-dropdown-item command="update-ssh">更新SSH凭据</el-dropdown-item>
                  </template>
                  <template v-else>
                    <el-dropdown-item command="update-ssh">配置SSH凭据（转为远程管理）</el-dropdown-item>
                    <el-dropdown-item command="traffic-limit" divided>设置流量配额</el-dropdown-item>
                    <el-dropdown-item command="reset-traffic">清零流量</el-dropdown-item>
                  </template>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
          <div class="selfhost-node-grid">
            <div class="selfhost-node-item">
              <span class="detail-label">协议</span>
              <span class="detail-value">{{ n.protocol_display || n.protocol || '-' }}</span>
            </div>
            <div class="selfhost-node-item">
              <span class="detail-label">服务器</span>
              <span class="detail-value">{{ n.ssh_host || n.domain || '-' }}<template v-if="n.ssh_port">:{{ n.ssh_port }}</template></span>
            </div>
            <div class="selfhost-node-item">
              <span class="detail-label">监听端口</span>
              <span class="detail-value">{{ n.port || '-' }}</span>
            </div>
            <div class="selfhost-node-item">
              <span class="detail-label">上行流量</span>
              <span class="detail-value">{{ formatBytes(n.traffic_up) }}</span>
            </div>
            <div class="selfhost-node-item">
              <span class="detail-label">下行流量</span>
              <span class="detail-value">{{ formatBytes(n.traffic_down) }}</span>
            </div>
            <div class="selfhost-node-item">
              <span class="detail-label">最近心跳</span>
              <span class="detail-value">{{ formatTime(n.last_heartbeat_at) }}</span>
            </div>
            <div class="selfhost-node-item">
              <span class="detail-label">SSH</span>
              <span class="detail-value">{{ n.ssh_host ? n.ssh_user + '@' + n.ssh_host + ':' + (n.ssh_port || 22) : '手动模式' }}</span>
            </div>
            <div class="selfhost-node-item">
              <span class="detail-label">创建时间</span>
              <span class="detail-value">{{ formatTime(n.created_at) }}</span>
            </div>
          </div>
          <!-- 流量配额（节点级） -->
          <div v-if="n.traffic_limit_enabled" class="traffic-limit-bar">
            <div class="traffic-limit-head">
              <span class="detail-label">节点流量配额</span>
              <span class="detail-value">{{ formatBytes(n.traffic_up + n.traffic_down) }} / {{ formatBytes(n.traffic_limit_bytes) }}</span>
            </div>
            <el-progress
              :percentage="trafficPercent(n)"
              :color="trafficColor(n)"
              :stroke-width="8"
            />
            <div class="traffic-limit-tip">已用 {{ trafficPercent(n) }}%，超过 100% 将自动屏蔽</div>
          </div>
          <!-- 分配客户（客户独享节点配额） -->
          <div v-if="n.assignments && n.assignments.length" class="assignments-section">
            <div class="assignments-head">
              <span class="detail-label">分配客户</span>
              <el-button size="small" text bg @click="loadAssignments(n)">刷新</el-button>
            </div>
            <div v-for="a in n.assignments" :key="a.user_id" class="assignment-row">
              <div class="assignment-user">
                <span class="assignment-name">{{ a.username }}</span>
                <span class="assignment-email">{{ a.email }}</span>
              </div>
              <div class="assignment-quota">
                <template v-if="a.traffic_limit_enabled">
                  <el-progress
                    :percentage="assignmentPercent(a)"
                    :color="trafficColor(a)"
                    :stroke-width="6"
                    class="assignment-progress"
                  />
                  <span class="assignment-quota-text">{{ formatBytes(a.traffic_used) }} / {{ formatBytes(a.traffic_limit_bytes) }}</span>
                </template>
                <span v-else class="assignment-noquota">无限流量</span>
              </div>
              <el-button size="small" link type="primary" @click="setAssignmentQuota(n, a)">设配额</el-button>
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <!-- VPS 自动搭建弹窗 -->
    <el-dialog
      v-model="showVpsDialog"
      :title="vpsMode === 'domain' ? '域名多协议全自动搭建' : 'VPS 全自动搭建'"
      width="560px"
      class="selfhost-dialog"
      :close-on-click-modal="false"
    >
      <el-alert
        :title="vpsMode === 'domain'
          ? '填写 VPS 信息 + 域名（域名需已解析到 VPS）。系统将自动申请 TLS 证书并部署多协议节点（VLESS+WS+TLS / VLESS+Reality / Trojan+WS+TLS / Shadowsocks），全部回传面板。'
          : '填写 VPS 的 IP/域名、SSH 端口与 root 密码，系统将 SSH 全自动部署 sing-box 并回传节点，无需手动执行命令。'"
        :type="vpsMode === 'domain' ? 'warning' : 'success'"
        :closable="false"
        show-icon
        class="mb-3"
      />
      <div class="saved-vps-section mb-3" v-if="savedVpsList.length > 0">
        <div class="saved-vps-head">
          <span class="saved-vps-label"><el-icon><Connection /></el-icon> 已搭建过的 VPS</span>
          <el-button size="small" text bg @click="loadSavedVps">
            <el-icon><Refresh /></el-icon>刷新
          </el-button>
        </div>
        <el-select
          v-model="selectedSavedVps"
          placeholder="选择已保存的 VPS，自动填充 SSH 信息（免输密码）"
          filterable
          clearable
          class="full-width-control"
          @change="applySavedVps"
        >
          <el-option
            v-for="v in savedVpsList"
            :key="v.key"
            :label="`${v.ssh_host}${v.ssh_port !== 22 ? ':' + v.ssh_port : ''}（${v.node_name || '历史节点'}${v.has_password ? ' · 已存密码' : ''}）`"
            :value="v.key"
          />
        </el-select>
        <div class="form-tip" v-if="selectedSavedVps">
          已选择「{{ selectedVpsName }}」，SSH 信息已自动填充{{ savedSelectedHasPassword ? '，密码使用已保存的凭据' : '，请手动输入密码' }}
        </div>
      </div>

      <div class="vps-mode-switch mb-3">
        <el-radio-group v-model="vpsMode">
          <el-radio-button value="single">单协议搭建</el-radio-button>
          <el-radio-button value="domain">域名多协议搭建</el-radio-button>
        </el-radio-group>
      </div>
      <el-form :model="vpsForm" label-position="top" class="vps-form">
        <el-form-item label="节点名称" required>
          <el-input v-model="vpsForm.name" placeholder="如: 我的东京VPS" maxlength="50" />
        </el-form-item>
        <template v-if="vpsMode === 'domain'">
          <el-form-item label="域名（需已解析到本 VPS）">
            <el-input v-model="vpsForm.domain" placeholder="如: node.example.com" />
            <div class="form-tip">留空则跳过 TLS 证书（只部署无域名协议）</div>
          </el-form-item>
          <el-form-item label="证书邮箱（acme）">
            <el-input v-model="vpsForm.email" placeholder="如: admin@example.com" />
          </el-form-item>
          <el-form-item label="选择协议（可多选）">
            <el-checkbox-group v-model="vpsForm.protocols">
              <el-checkbox value="vless-ws">VLESS + WS + TLS</el-checkbox>
              <el-checkbox value="vmess-ws">VMess + WS + TLS</el-checkbox>
              <el-checkbox value="vless-reality">VLESS + Reality</el-checkbox>
              <el-checkbox value="vless-reality-grpc">VLESS + Reality + gRPC</el-checkbox>
              <el-checkbox value="vless-reality-xhttp">VLESS + Reality + XHTTP</el-checkbox>
              <el-checkbox value="vless-grpc-tls">VLESS + gRPC + TLS</el-checkbox>
              <el-checkbox value="trojan-tcp-tls">Trojan + TCP + TLS</el-checkbox>
              <el-checkbox value="trojan-ws">Trojan + WS + TLS</el-checkbox>
              <el-checkbox value="trojan-grpc-tls">Trojan + gRPC + TLS</el-checkbox>
              <el-checkbox value="hysteria2">Hysteria2</el-checkbox>
              <el-checkbox value="tuic">TUIC</el-checkbox>
              <el-checkbox value="ss">Shadowsocks</el-checkbox>
            </el-checkbox-group>
          </el-form-item>
        </template>
        <el-form-item v-else label="协议" required>
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
        部署过程全自动：连接 SSH → 下载 sing-box → 申请证书（域名模式）→ 生成多协议 → 启动服务 → 批量回传。密码使用 SECRET_KEY 加密保存，仅用于节点管理。
      </div>
      <template #footer>
        <el-button :disabled="deployingVPS" @click="showVpsDialog = false">取消</el-button>
        <el-button type="primary" :loading="deployingVPS" :disabled="!vpsForm.name || !vpsForm.ssh_host || (!vpsForm.ssh_pass && !savedSelectedHasPassword)" @click="deploySelfHostVPSNode">
          一键自动搭建
        </el-button>
      </template>
    </el-dialog>

    <!-- 手动搭建弹窗 -->
    <el-dialog
      v-model="showManualDialog"
      title="手动搭建"
      width="560px"
      class="selfhost-dialog"
      :close-on-click-modal="false"
    >
      <template v-if="!manualNode">
        <el-alert
          title="生成一条安装命令，复制到您的 VPS 上执行（需要 root 权限）。脚本自动下载 sing-box、生成节点并回传。"
          type="info"
          :closable="false"
          show-icon
          class="mb-3"
        />
        <el-form :model="manualForm" label-position="top" class="vps-form">
          <el-form-item label="节点名称" required>
            <el-input v-model="manualForm.name" placeholder="如: 我的香港VPS" maxlength="50" />
          </el-form-item>
          <el-form-item label="协议" required>
            <el-select v-model="manualForm.protocol" class="full-width-control">
              <el-option label="VLESS + WebSocket（推荐）" value="vless-ws" />
              <el-option label="VMess + WebSocket" value="vmess-ws" />
              <el-option label="VLESS + Reality（防封锁强）" value="vless-reality" />
              <el-option label="Trojan + WebSocket" value="trojan-ws" />
              <el-option label="Shadowsocks" value="ss" />
            </el-select>
          </el-form-item>
        </el-form>
      </template>
      <template v-else>
        <el-alert
          title="复制以下命令到您的服务器执行（需要 root 权限）："
          type="warning"
          :closable="false"
          show-icon
          class="mb-3"
        />
        <div class="selfhost-cmd-box">
          <div class="selfhost-cmd-text">{{ manualNode.install_cmd }}</div>
          <el-button class="selfhost-copy-btn" type="primary" link @click="copyManualCmd">
            <el-icon><DocumentCopy /></el-icon>
          </el-button>
        </div>
        <div class="subscription-tip">
          执行完成后节点会自动回传并显示在列表中。安装令牌 30 分钟内有效。
        </div>
      </template>
      <template #footer>
        <el-button @click="showManualDialog = false">关闭</el-button>
        <el-button v-if="!manualNode" type="primary" :loading="creatingManual" :disabled="!manualForm.name" @click="createManualNode">
          生成安装命令
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from '@/utils/elementPlusServices'
import { Promotion, DocumentCopy, Refresh, Setting, Operation, ArrowDown, Connection } from '@element-plus/icons-vue'
import { adminAPI } from '@/utils/api'
import { confirmAction } from '@/utils/confirmAction'
import { useMobile } from '@/composables/useMobile'

const isMobile = useMobile()
const selfHostNodes = ref([])
const selfHostLoading = ref(false)

const selfHostStatusMap = { pending: '等待安装', online: '在线', offline: '离线', expired: '已过期', canceled: '已取消' }
const selfHostStatusTypeMap = { pending: 'warning', online: 'success', offline: 'danger', expired: 'info', canceled: 'info' }

const formatBytes = (b) => {
  if (b === undefined || b === null || isNaN(b)) return '-'
  if (b === 0) return '0 B'
  const u = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(b) / Math.log(1024))
  return (b / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 2) + ' ' + u[i]
}
const formatTime = (t) => {
  if (!t) return '-'
  const d = new Date(t)
  if (isNaN(d.getTime())) return '-'
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

const loadSelfHostNodes = async () => {
  selfHostLoading.value = true
  try {
    const res = await adminAPI.getSelfHostNodes()
    if (res.data?.success) {
      selfHostNodes.value = res.data.data?.list || []
      // 并行加载每个节点的分配客户
      selfHostNodes.value.forEach(n => loadAssignments(n))
    }
  } catch (e) {
    console.warn('加载自建节点失败', e)
  } finally {
    selfHostLoading.value = false
  }
}

// ===== 批量选择与操作 =====
const selectedSelfHost = ref([])
const isSelected = (n) => selectedSelfHost.value.some(x => x.id === n.id)
const toggleSelect = (n, v) => {
  if (v) {
    if (!isSelected(n)) selectedSelfHost.value.push(n)
  } else {
    selectedSelfHost.value = selectedSelfHost.value.filter(x => x.id !== n.id)
  }
}
const toggleSelectAll = (v) => {
  selectedSelfHost.value = v ? [...selfHostNodes.value] : []
}
const batchRunning = ref(false)
const onBatchCommand = async (action) => {
  if (!selectedSelfHost.value.length) return ElMessage.warning('请先选择节点')
  const ids = selectedSelfHost.value.map(n => n.id)
  const names = selectedSelfHost.value.slice(0, 3).map(n => n.name).join('、') + (selectedSelfHost.value.length > 3 ? ` 等${selectedSelfHost.value.length}个` : '')

  if (action === 'delete') {
    const ok = await confirmAction(`确认删除选中的 ${ids.length} 个自建节点「${names}」？此操作不可恢复。`)
    if (!ok) return
  } else if (action === 'change-password') {
    const { value } = await ElMessageBox.prompt('输入新的 UUID（密码），将应用到所有选中节点', '批量改密码', {
      inputPattern: /^[0-9a-fA-F-]{36}$/,
      inputErrorMessage: '请输入合法的 UUID（36位）'
    }).catch(() => ({}))
    if (!value) return
    await doBatch({ node_ids: ids, action, new_pass: value })
    return
  } else if (action === 'traffic-limit') {
    const { value } = await ElMessageBox.prompt(`输入流量配额（GB）应用到 ${ids.length} 个节点，0 关闭`, '批量设置配额', {
      inputValue: '100', inputPattern: /^\d+$/, inputErrorMessage: '请输入数字'
    }).catch(() => ({}))
    if (value === undefined) return
    const gb = parseInt(value, 10)
    await doBatch({ node_ids: ids, action, enabled: gb > 0, limit_bytes: gb * 1073741824 })
    return
  } else if (['reset', 'change-port', 'reset-traffic', 'enable', 'disable'].includes(action)) {
    const tip = { reset: '重置UUID', 'change-port': '随机改端口', 'reset-traffic': '清零流量', enable: '启用', disable: '禁用' }[action]
    const ok = await confirmAction(`确认对 ${ids.length} 个节点执行「${tip}」？`)
    if (!ok) return
    await doBatch({ node_ids: ids, action })
    return
  }
  await doBatch({ node_ids: ids, action })
}
const doBatch = async (payload) => {
  batchRunning.value = true
  try {
    const res = await adminAPI.selfHostBatchManage(payload)
    if (res.data?.success) {
      ElMessage.success(res.data.message || '批量操作完成')
      const failed = (res.data.data?.results || []).filter(r => !r.success)
      if (failed.length) {
        ElMessage.warning(`${failed.length} 个失败: ${failed[0].message || ''}`)
      }
      selectedSelfHost.value = []
      loadSelfHostNodes()
    } else {
      ElMessage.error(res.data?.message || '批量操作失败')
    }
  } catch (e) {
    ElMessage.error('批量操作失败: ' + (e.response?.data?.message || e.message))
  } finally {
    batchRunning.value = false
  }
}

// ===== 分配客户与配额 =====
const loadAssignments = async (node) => {
  try {
    const res = await adminAPI.getCustomNodeUsers(node.id)
    if (res.data?.success) {
      node.assignments = res.data.data || []
    }
  } catch (e) {
    console.warn('加载分配客户失败', e)
  }
}
const assignmentPercent = (a) => {
  if (!a.traffic_limit_bytes) return 0
  const used = a.traffic_used || 0
  return Math.min(100, Math.round((used / a.traffic_limit_bytes) * 100))
}
const setAssignmentQuota = async (node, a) => {
  const { value } = await ElMessageBox.prompt(
    `为客户「${a.username}」设置节点流量配额（GB），输入 0 关闭。当前已用: ${formatBytes(a.traffic_used || 0)}`,
    '设置客户配额', {
      inputValue: a.traffic_limit_bytes ? String(Math.round(a.traffic_limit_bytes / 1073741824)) : '100',
      inputPattern: /^\d+$/,
      inputErrorMessage: '请输入数字（GB）'
    }
  ).catch(() => ({}))
  if (value === undefined) return
  const gb = parseInt(value, 10)
  try {
    await adminAPI.updateUserCustomNodeQuota(node.id, a.id, gb > 0
      ? { enabled: true, limit_bytes: gb * 1073741824 }
      : { enabled: false, limit_bytes: 0 })
    ElMessage.success('客户配额已更新')
    loadAssignments(node)
    loadSelfHostNodes()
  } catch (e) {
    ElMessage.error('设置失败: ' + (e.response?.data?.message || e.message))
  }
}

// ===== VPS 自动搭建 =====
const showVpsDialog = ref(false)
const deployingVPS = ref(false)
const vpsMode = ref('single')
const vpsReuseNodeId = ref(0) // >0 表示复用该节点记录重装（同一 VPS 二次部署时由后端占用提示触发）
const vpsForm = reactive({
  name: '', protocol: 'vless-ws', protocols: ['vless-ws', 'vless-reality', 'trojan-ws', 'ss'],
  domain: '', email: '', ssh_host: '', ssh_port: 22, ssh_user: 'root', ssh_pass: ''
})
// 已保存的 VPS（从历史自建节点去重提取，用于二次搭建免输 SSH 信息）
const savedVpsList = ref([])
const selectedSavedVps = ref('')
const savedSelectedHasPassword = ref(false)
const savedVpsNodes = ref([]) // 原始节点列表（含 ssh_password_enc 存在性判断）
const selectedVpsName = computed(() => {
  const v = savedVpsList.value.find(x => x.key === selectedSavedVps.value)
  return v ? v.node_name || v.ssh_host : ''
})
// 已保存 VPS 的 key → node_id（有加密密码的节点，用于免密部署）
const savedVpsNodeIdMap = ref({})

const loadSavedVps = async () => {
  try {
    const res = await adminAPI.getSelfHostNodes()
    const nodes = res.data?.data?.list || res.data?.data || []
    savedVpsNodes.value = nodes
    // 按 ssh_host+ssh_port 去重，保留最近一个节点作为代表
    const byKey = new Map()
    for (const n of nodes) {
      if (!n.ssh_host) continue
      const key = `${n.ssh_host}:${n.ssh_port || 22}`
      if (!byKey.has(key) || (n.created_at || '') > (byKey.get(key).created_at || '')) {
        byKey.set(key, n)
      }
    }
    savedVpsList.value = [...byKey.entries()].map(([key, n]) => ({
      key,
      ssh_host: n.ssh_host,
      ssh_port: n.ssh_port || 22,
      ssh_user: n.ssh_user || 'root',
      node_name: n.name,
      node_id: n.id,
      has_password: true // 自建节点只要有 ssh_host 就有加密密码（后端保存的）
    }))
    // node_id → 该 VPS 任一有密码的节点 id（供 saved_ssh_id 使用）
    const idMap = {}
    for (const n of nodes) {
      if (!n.ssh_host) continue
      const key = `${n.ssh_host}:${n.ssh_port || 22}`
      if (!(key in idMap)) idMap[key] = n.id
    }
    savedVpsNodeIdMap.value = idMap
  } catch (e) {
    console.warn('加载已保存 VPS 失败', e)
  }
}

const applySavedVps = (key) => {
  if (!key) {
    savedSelectedHasPassword.value = false
    return
  }
  const v = savedVpsList.value.find(x => x.key === key)
  if (!v) return
  vpsForm.ssh_host = v.ssh_host
  vpsForm.ssh_port = v.ssh_port
  vpsForm.ssh_user = v.ssh_user
  vpsForm.ssh_pass = '' // 密码留空，后端用已保存的加密密码
  savedSelectedHasPassword.value = true
  // 若该 VPS 已有节点，直接标记复用（二次搭建=重装，避免幽灵节点）
  const nodeId = savedVpsNodeIdMap.value[key]
  if (nodeId) vpsReuseNodeId.value = nodeId
}

const openVpsDialog = () => {
  Object.assign(vpsForm, { name: '', domain: '', email: '', ssh_host: '', ssh_pass: '' })
  vpsReuseNodeId.value = 0
  selectedSavedVps.value = ''
  savedSelectedHasPassword.value = false
  showVpsDialog.value = true
  loadSavedVps()
}
const deploySelfHostVPSNode = async () => {
  deployingVPS.value = true
  try {
    const payload = { name: vpsForm.name, ssh_host: vpsForm.ssh_host, ssh_port: vpsForm.ssh_port, ssh_user: vpsForm.ssh_user, ssh_pass: vpsForm.ssh_pass }
    // 使用已保存 VPS 的加密密码（未手动输入密码时）：传 saved_ssh_id，后端解密连接
    if (!payload.ssh_pass && selectedSavedVps.value) {
      const nodeId = savedVpsNodeIdMap.value[selectedSavedVps.value]
      if (nodeId) {
        payload.saved_ssh_id = nodeId
        payload.ssh_pass = ''
      }
    }
    // 复用模式：携带 reuse_node_id，后端复用原节点记录重装（不新建，避免幽灵节点）
    if (vpsReuseNodeId.value) payload.reuse_node_id = vpsReuseNodeId.value
    if (vpsMode.value === 'domain') {
      payload.domain = vpsForm.domain
      payload.email = vpsForm.email
      payload.protocols = vpsForm.protocols
    } else {
      payload.protocol = vpsForm.protocol
    }
    const res = vpsMode.value === 'domain'
      ? await adminAPI.deploySelfHostVPSDomain(payload)
      : await adminAPI.deploySelfHostVPS(payload)
    if (res.data?.success) {
      ElMessage.success(res.data.message || 'VPS 自动搭建完成')
      showVpsDialog.value = false
      vpsReuseNodeId.value = 0
      selectedSavedVps.value = ''
      loadSelfHostNodes()
    } else {
      ElMessage.error(res.data?.message || '搭建失败')
    }
  } catch (e) {
    const resp = e.response?.data
    // 该 VPS 已部署过节点：提示并让管理员确认复用（覆盖旧节点重装）
    if (resp?.code === 'vps_occupied' && resp?.data?.existing_node_id) {
      const exId = resp.data.existing_node_id
      const exName = resp.data.existing_node_name || '旧节点'
      try {
        await ElMessageBox.confirm(
          `该 VPS 已部署自建节点「${exName}」（#${exId}）。\n\n继续将覆盖它的配置并使其失效（旧节点会消失）。\n是否复用该节点重新搭建？`,
          '检测到同一 VPS 已有节点',
          { confirmButtonText: '复用并重新搭建', cancelButtonText: '取消', type: 'warning', confirmButtonClass: 'danger-confirm' }
        )
        vpsReuseNodeId.value = exId
        await deploySelfHostVPSNode()
        return
      } catch (confirmErr) {
        // 用户取消
      }
    } else {
      ElMessage.error('搭建失败: ' + (resp?.message || e.message))
    }
  } finally {
    deployingVPS.value = false
  }
}

// ===== 手动搭建 =====
const showManualDialog = ref(false)
const creatingManual = ref(false)
const manualNode = ref(null)
const manualForm = reactive({ name: '', protocol: 'vless-ws' })
const openManualDialog = () => {
  manualNode.value = null
  showManualDialog.value = true
}
const createManualNode = async () => {
  creatingManual.value = true
  try {
    const res = await adminAPI.createSelfHostNode({ name: manualForm.name, protocol: manualForm.protocol })
    if (res.data?.success) {
      manualNode.value = res.data.data
      loadSelfHostNodes()
    } else {
      ElMessage.error(res.data?.message || '创建失败')
    }
  } catch (e) {
    ElMessage.error('创建失败: ' + (e.response?.data?.message || e.message))
  } finally {
    creatingManual.value = false
  }
}
const copyManualCmd = () => {
  if (manualNode.value?.install_cmd) {
    navigator.clipboard.writeText(manualNode.value.install_cmd)
    ElMessage.success('安装命令已复制')
  }
}

// ===== 批量管理 =====
const managingSelfHostId = ref(null)
const selfHostManage = async (node, action, extra = {}) => {
  managingSelfHostId.value = node.id
  try {
    let res
    if (action === 'traffic-limit') {
      res = await adminAPI.selfHostTrafficLimit(node.id, extra)
    } else {
      res = await adminAPI.selfHostManage(node.id, action, extra)
    }
    if (res.data?.success) {
      ElMessage.success(res.data.message || '操作成功')
      loadSelfHostNodes()
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
  if (action === 'status') {
    const res = await adminAPI.selfHostManage(node.id, 'status').catch(() => null)
    if (res?.data?.success) {
      ElMessageBox.alert(`<pre style="white-space:pre-wrap;font-size:12px">${res.data.data?.status || '无输出'}</pre>`, `节点「${node.name}」远程状态`, { dangerouslyUseHTMLString: true })
    }
  } else if (action === 'reset') {
    const ok = await confirmAction(`确认重置节点「${node.name}」的凭据（重新生成 UUID）？`)
    if (!ok) return
    await selfHostManage(node, 'reset')
  } else if (action === 'change-password') {
    const { value } = await ElMessageBox.prompt('输入新的 UUID（密码）', '更改节点密码', {
      inputPattern: /^[0-9a-fA-F-]{36}$/,
      inputErrorMessage: '请输入合法的 UUID（36位，含横线）'
    }).catch(() => ({}))
    if (!value) return
    await selfHostManage(node, 'change-password', { new_pass: value })
  } else if (action === 'change-port') {
    // 多协议节点：自动随机分配互不重复端口（后端处理）；单协议节点：输入新端口
    if (node.deploy_mode === 'multi') {
      const ok = await confirmAction(`节点「${node.name}」是多协议节点，确认为其所有协议随机分配新的互不重复端口？`)
      if (!ok) return
      await selfHostManage(node, 'change-port', { new_port: 0 })
    } else {
      const { value } = await ElMessageBox.prompt('输入新的监听端口', '更改端口', {
        inputPattern: /^\d+$/,
        inputErrorMessage: '端口需在 1-65535'
      }).catch(() => ({}))
      if (!value) return
      const port = parseInt(value, 10)
      if (port < 1 || port > 65535) return ElMessage.warning('端口需在 1-65535')
      await selfHostManage(node, 'change-port', { new_port: port })
    }
  } else if (action === 'reinstall') {
    const ok = await confirmAction(`确认在 VPS「${node.ssh_host || node.name}」上重新搭建节点？会重新安装 sing-box。`)
    if (!ok) return
    await selfHostManage(node, 'reinstall')
  } else if (action === 'traffic-limit') {
    const { value } = await ElMessageBox.prompt(
      `输入流量配额（GB），输入 0 或取消关闭配额。当前已用: ${formatBytes(node.traffic_up + node.traffic_down)}`,
      '设置流量配额', {
        inputValue: node.traffic_limit_bytes ? String(Math.round(node.traffic_limit_bytes / 1073741824)) : '100',
        inputPattern: /^\d+$/,
        inputErrorMessage: '请输入数字（GB）'
      }
    ).catch(() => ({}))
    if (value === undefined) return
    const gb = parseInt(value, 10)
    if (gb > 0) {
      await selfHostManage(node, 'traffic-limit', { enabled: true, limit_bytes: gb * 1073741824 })
    } else {
      await selfHostManage(node, 'traffic-limit', { enabled: false, limit_bytes: 0 })
    }
  } else if (action === 'update-ssh') {
    const { value } = await ElMessageBox.prompt(
      '输入新的 VPS root 密码（或首次为手动节点配置 SSH 凭据）', '更新SSH凭据', {
        inputType: 'password',
        inputErrorMessage: '密码不能为空'
      }
    ).catch(() => ({}))
    if (!value) return
    managingSelfHostId.value = node.id
    try {
      await adminAPI.selfHostUpdateSSH(node.id, { ssh_pass: value, ssh_host: node.ssh_host || '', ssh_port: node.ssh_port || 22, ssh_user: node.ssh_user || 'root' })
      ElMessage.success('SSH 凭据已更新')
      loadSelfHostNodes()
    } catch (e) {
      ElMessage.error('更新失败: ' + (e.response?.data?.message || e.message))
    } finally {
      managingSelfHostId.value = null
    }
  } else if (action === 'reset-traffic') {
    const ok = await confirmAction(`确认清零节点「${node.name}」的已用流量？`)
    if (!ok) return
    managingSelfHostId.value = node.id
    try {
      await adminAPI.selfHostResetTraffic(node.id)
      ElMessage.success('流量已清零')
      loadSelfHostNodes()
    } catch (e) {
      ElMessage.error('清零失败: ' + (e.response?.data?.message || e.message))
    } finally {
      managingSelfHostId.value = null
    }
  }
}
const trafficPercent = (n) => {
  if (!n.traffic_limit_bytes) return 0
  const used = (n.traffic_up || 0) + (n.traffic_down || 0)
  return Math.min(100, Math.round((used / n.traffic_limit_bytes) * 100))
}
const trafficColor = (n) => {
  const p = trafficPercent(n)
  if (p >= 90) return '#f56c6c'
  if (p >= 60) return '#e6a23c'
  return '#67c23a'
}

onMounted(() => {
  loadSelfHostNodes()
})
</script>

<style scoped>
.admin-selfhost-nodes {
  padding: 12px;
}
@media (max-width: 768px) {
  .admin-selfhost-nodes {
    padding: 10px;
  }
}
.list-card {
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}
.selfhost-guide {
  margin-bottom: 14px;
}
/* 批量操作工具栏 */
.batch-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  margin-bottom: 12px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  flex-wrap: wrap;
}
.node-checkbox {
  margin-right: 2px;
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
  flex-wrap: wrap;
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
  grid-template-columns: repeat(4, minmax(0, 1fr));
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
  margin-top: 4px;
}
.inline-fields-row {
  display: flex;
  gap: 12px;
}
.inline-fields-row .flex-1 {
  flex: 1;
}
.input-full {
  width: 100%;
}
.full-width-control {
  width: 100%;
}
.subscription-tip {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin-top: 8px;
  padding: 8px 10px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}
.mb-3 {
  margin-bottom: 12px;
}
.vps-mode-switch {
  margin-bottom: 4px;
}
.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}
.saved-vps-section {
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 10px 12px;
}
.saved-vps-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.saved-vps-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.traffic-limit-bar {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed var(--el-border-color-lighter);
}
.traffic-limit-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}
.traffic-limit-tip {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  margin-top: 4px;
}
.assignments-section {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed var(--el-border-color-lighter);
}
.assignments-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}
.assignment-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 0;
}
.assignment-user {
  display: flex;
  flex-direction: column;
  min-width: 0;
  width: 160px;
}
.assignment-name {
  font-weight: 600;
  font-size: 13px;
}
.assignment-email {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.assignment-quota {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.assignment-progress {
  flex: 1;
  min-width: 60px;
}
.assignment-quota-text {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.assignment-noquota {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}
.selfhost-cmd-box {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  background: #1e1e1e;
  border-radius: 6px;
  padding: 10px 12px;
  margin-bottom: 12px;
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
.danger-menu-item {
  color: var(--el-color-danger);
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
