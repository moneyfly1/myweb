<template>
  <div class="audit-log-list">
    <div class="filter-bar desktop-only">
      <el-input v-model="filter.keyword" placeholder="搜索操作描述/邮箱" clearable style="width: 220px" @keyup.enter="fetch" />
      <el-date-picker v-model="filter.timeRange" type="datetimerange" range-separator="至" start-placeholder="开始" end-placeholder="结束" value-format="YYYY-MM-DD HH:mm:ss" style="width: 340px" />
      <el-button type="primary" @click="fetch" :loading="loading">搜索</el-button>
      <el-button @click="resetFilter">重置</el-button>
    </div>
    <div class="table-wrapper desktop-only">
      <el-table v-loading="loading" :data="list" stripe border>
        <el-table-column prop="created_at" label="时间" width="170" />
        <el-table-column label="操作类型" width="140">
          <template #default="{ row }">{{ getActionTypeText(row.action_type) }}</template>
        </el-table-column>
        <el-table-column label="用户" width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <template v-if="getTargetUser(row)">
              {{ getTargetUser(row) }}
            </template>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作描述" min-width="280" show-overflow-tooltip>
          <template #default="{ row }">{{ row.action_description || '-' }}</template>
        </el-table-column>
        <el-table-column label="变更前" width="180" show-overflow-tooltip>
          <template #default="{ row }"><div class="audit-data" v-html="fmtAuditData(row.before_data)"></div></template>
        </el-table-column>
        <el-table-column label="变更后" width="180" show-overflow-tooltip>
          <template #default="{ row }"><div class="audit-data" v-html="fmtAuditData(row.after_data)"></div></template>
        </el-table-column>
        <el-table-column prop="ip_address" label="IP" width="135" show-overflow-tooltip />
      </el-table>
    </div>
    <div class="mobile-only mobile-card-list">
      <div v-loading="loading" class="mobile-list-inner">
        <div v-for="row in list" :key="row.id" class="mobile-log-card">
          <div class="mobile-card-row"><span class="mobile-label">时间</span><span class="mobile-value">{{ row.created_at }}</span></div>
          <div class="mobile-card-row"><span class="mobile-label">操作</span><span class="mobile-value">{{ getActionTypeText(row.action_type) }}</span></div>
          <div class="mobile-card-row"><span class="mobile-label">用户</span><span class="mobile-value">{{ getTargetUser(row) || '-' }}</span></div>
          <div class="mobile-card-row"><span class="mobile-label">描述</span><span class="mobile-value mobile-value-wrap">{{ row.action_description || '-' }}</span></div>
          <div class="mobile-card-row" v-if="row.before_data"><span class="mobile-label">变更前</span><span class="mobile-value mobile-value-wrap" v-html="fmtAuditData(row.before_data)"></span></div>
          <div class="mobile-card-row" v-if="row.after_data"><span class="mobile-label">变更后</span><span class="mobile-value mobile-value-wrap" v-html="fmtAuditData(row.after_data)"></span></div>
        </div>
        <el-empty v-if="list.length === 0 && !loading" description="暂无操作记录" />
      </div>
    </div>
    <el-pagination
      v-model:current-page="page"
      :page-size="pageSize"
      :total="total"
      layout="total, prev, pager, next"
      @current-change="fetch"
      @size-change="(s) => { pageSize = s; page = 1; fetch() }"
      :page-sizes="[10, 20, 50]"
      class="pagination"
    />
  </div>
</template>
<script setup>
import { ref, onMounted } from 'vue'
import { adminAPI } from '@/utils/api'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const filter = ref({ keyword: '', timeRange: null })

const ACTION_TYPE_MAP = {
  send_subscription_email: '发送订阅邮件', extend_subscription: '延长订阅', reset_subscription: '重置订阅',
  clear_user_devices: '清理用户设备', reset_user_subscription: '重置用户订阅',
  batch_update_subscriptions_status: '批量更新订阅状态', batch_reset_subscriptions: '批量重置订阅',
  batch_send_subscription_email: '批量发送订阅邮件', batch_send_sub_email: '批量发送订阅邮件',
  update_user_status: '更新用户状态', batch_enable_users: '批量启用用户', batch_disable_users: '批量禁用用户',
  batch_delete_users: '批量删除用户', reset_password: '重置密码', update_settings: '更新设置',
  login_as_user: '模拟登录', batch_delete_subscriptions: '批量删除订阅',
  refund_order: '退款', delete_order: '删除订单', update_order: '更新订单',
  batch_enable_subscriptions: '批量启用订阅', batch_disable_subscriptions: '批量禁用订阅',
  batch_delete_orders: '批量删除订单', unlock_user_login: '解除登录限制',
  update_email_config: '更新邮件配置', update_announcement: '更新公告',
  create_user: '创建用户', update_user: '更新用户信息', delete_user: '删除用户',
  create_invite_code: '创建邀请码', update_invite_code: '更新邀请码', delete_invite_code: '删除邀请码',
  create_package: '创建套餐', update_package: '更新套餐', delete_package: '删除套餐',
  create_ticket: '创建工单', update_ticket_status: '更新工单状态', close_ticket: '关闭工单',
  create_promotion: '创建营销活动', update_promotion: '更新营销活动', delete_promotion: '删除营销活动',
  create_backup: '创建备份', delete_backup: '删除备份',
  delete_device: '删除设备', admin_delete_device: '管理员删除设备', batch_delete_devices: '批量删除设备',
  batch_clear_devices: '批量清理设备', clear_config_update_logs: '清理配置更新日志',
  update_subscription: '更新订阅', update_device_limit: '修改设备数量', update_expire_time: '修改到期时间',
  activate_subscription: '启用订阅', deactivate_subscription: '停用订阅',
  update_config_update_config: '更新配置更新设置',
  start_config_update: '启动配置更新', stop_config_update: '停止配置更新',
  update_system_config: '更新系统配置', update_system_config_batch: '批量更新系统配置', create_system_config: '创建系统配置',
  update_geoip_database: '更新GeoIP数据库', switch_geoip_database: '切换GeoIP数据库', flush_cache: '清除缓存',
  create_custom_node: '创建专线节点', update_custom_node: '更新专线节点', delete_custom_node: '删除专线节点',
  import_custom_node_links: '导入专线节点链接', batch_delete_custom_nodes: '批量删除专线节点',
  batch_assign_custom_nodes: '批量分配专线节点',
  create_admin_notification: '创建通知', delete_admin_notification: '删除通知',
  send_email: '发送邮件', login: '用户登录', change_password: '修改密码',
  batch_send_expire_reminder: '批量发送到期提醒',
  batch_delete_users: '批量删除用户', update_user: '更新用户信息', delete_user: '删除用户',
  batch_delete_subscriptions: '批量删除订阅', update_admin_order: '更新订单', refund_admin_order: '退款',
  delete_admin_order: '删除订单', bulk_mark_orders_paid: '批量标记已付', bulk_cancel_orders: '批量取消订单',
  batch_delete_orders: '批量删除订单', create_custom_order: '创建自定义订单',
  create_package: '创建套餐', update_package: '更新套餐', delete_package: '删除套餐',
  create_coupon: '创建优惠券', update_coupon: '更新优惠券', delete_coupon: '删除优惠券',
  create_invite_code: '创建邀请码', update_invite_code: '更新邀请码', delete_invite_code: '删除邀请码',
  batch_delete_invite_codes: '批量删除邀请码', batch_delete_invite_relations: '批量删除邀请关系',
  create_node: '创建节点', update_node: '更新节点', delete_node: '删除节点',
  import_node_links: '导入节点链接', batch_delete_nodes: '批量删除节点', import_from_clash: 'Clash导入节点',
  create_backup: '创建备份', delete_backup: '删除备份',
  admin_delete_device: '删除设备', batch_delete_devices: '批量删除设备', batch_clear_devices: '批量清理设备',
  clear_config_update_logs: '清理配置更新日志', start_config_update: '启动配置更新', stop_config_update: '停止配置更新',
  update_config_update_config: '更新配置更新设置',
  create_user_level: '创建用户等级', update_user_level: '更新用户等级', delete_user_level: '删除用户等级',
  update_system_config: '更新系统配置', update_system_config_batch: '批量更新系统配置', create_system_config: '创建系统配置',
  update_geoip_database: '更新GeoIP数据库', switch_geoip_database: '切换GeoIP数据库', flush_cache: '清除缓存',
  create_custom_node: '创建专线节点', update_custom_node: '更新专线节点', delete_custom_node: '删除专线节点',
  import_custom_node_links: '导入专线节点链接', batch_delete_custom_nodes: '批量删除专线节点',
  batch_assign_custom_nodes: '批量分配专线节点', create_promotion: '创建营销活动', update_promotion: '更新营销活动', delete_promotion: '删除营销活动',
  create_admin_notification: '创建通知', delete_admin_notification: '删除通知',
  update_admin_notification: '更新通知', send_email: '发送邮件', login: '用户登录', change_password: '修改密码',
  reset_password: '重置密码', update_email_config: '更新邮件配置',
  create_payment_config: '创建支付配置', update_payment_config: '更新支付配置', delete_payment_config: '删除支付配置',
  bulk_delete_payment_configs: '批量删除支付配置', bulk_enable_payment_configs: '批量启用支付配置', bulk_disable_payment_configs: '批量禁用支付配置',
  update_ticket_status: '更新工单状态', close_ticket: '关闭工单', create_ticket: '创建工单',
  create_knowledge_category: '创建知识库分类', update_knowledge_category: '更新知识库分类', delete_knowledge_category: '删除知识库分类',
  create_knowledge_article: '创建知识库文章', update_knowledge_article: '更新知识库文章', delete_knowledge_article: '删除知识库文章',
  clear_audit_logs: '清空审计日志', delete_email_from_queue: '删除邮件', clear_email_queue: '清空邮件队列',
  update_single_config: '更新配置', clear_logs: '清理日志',
}

function getTargetUser(row) {
  // 从 before_data 提取目标用户
  if (row.before_data) {
    try {
      const obj = JSON.parse(row.before_data)
      if (obj.target_username || obj.target_email) {
        const name = obj.target_username || ''
        const mail = obj.target_email || ''
        return mail ? (name ? `${name} (${mail})` : mail) : name
      }
    } catch {}
  }
  // 从 after_data 提取
  if (row.after_data) {
    try {
      const obj = JSON.parse(row.after_data)
      if (obj.target_username || obj.target_email) {
        const name = obj.target_username || ''
        const mail = obj.target_email || ''
        return mail ? (name ? `${name} (${mail})` : mail) : name
      }
    } catch {}
  }
  // 回退: 从描述中提取用户名
  const desc = row.action_description || ''
  const m = desc.match(/用户\s*(\S+)\s*\(/)
  if (m) return m[1]
  return ''
}

const FIELD_LABEL_MAP = {
  target_user_id: '目标用户ID', target_username: '用户名', target_email: '邮箱',
  subscription_url: '订阅地址', old_subscription_url: '旧订阅地址', new_subscription_url: '新订阅地址',
  device_limit: '设备数', extend_days: '延长天数', old_expire_time: '原到期时间', new_expire_time: '新到期时间',
  old_device_limit: '原设备数', new_device_limit: '新设备数',
  is_active: '启用', is_verified: '已验证', is_admin: '管理员', status: '状态',
  user_ids: '用户列表', subscription_ids: '订阅列表', count: '数量', total: '总数',
  success_count: '成功', fail_count: '失败', subscription_count: '订阅数', device_count: '设备数',
  details: '详情', targets: '目标', users: '用户',
  email: '邮箱', username: '用户名',
}

function getActionTypeText(type) {
  if (!type) return '-'
  if (ACTION_TYPE_MAP[type]) return ACTION_TYPE_MAP[type]
  if (type.startsWith('security_')) return '安全事件'
  if (type.startsWith('business_')) return ''
  if (type.startsWith('scheduler_')) return ''
  if (type === 'system_error') return ''
  return type
}

function fieldLabel(key) { return FIELD_LABEL_MAP[key] || key }

function fmtAuditData(v) {
  if (!v) return '-'
  try {
    const obj = JSON.parse(v)
    if (Array.isArray(obj)) {
      if (obj.every(x => typeof x === 'object' && x !== null)) return `共 ${obj.length} 条`
      return obj.join(', ')
    }
    if (typeof obj === 'object' && obj !== null) {
      const lines = []
      for (const [k, val] of Object.entries(obj)) {
        if (val === null || val === undefined) continue
        if (typeof val === 'object' && !Array.isArray(val)) continue
        if (Array.isArray(val)) { lines.push(`${fieldLabel(k)}: ${val.length}条`); continue }
        const display = typeof val === 'boolean' ? (val ? '是' : '否') : String(val)
        lines.push(`${fieldLabel(k)}: ${display}`)
      }
      return lines.slice(0, 8).join('<br>') + (lines.length > 8 ? '<br>...' : '') || '-'
    }
    return String(obj)
  } catch {
    return v.length > 200 ? v.substring(0, 200) + '...' : v
  }
}

async function fetch() {
  loading.value = true
  try {
    const params = { page: page.value, page_size: pageSize.value }
    if (filter.value.keyword) params.keyword = filter.value.keyword
    if (filter.value.timeRange && filter.value.timeRange.length === 2) {
      params.start_time = filter.value.timeRange[0]
      params.end_time = filter.value.timeRange[1]
    }
    const res = await adminAPI.getAuditLogs(params)
    const data = res?.data?.data ?? res?.data ?? {}
    const raw = data.logs || []
    list.value = raw.filter(row => {
      const t = row.action_type || ''
      return !t.startsWith('scheduler_') && t !== 'system_error' && !t.startsWith('business_') && !t.startsWith('security_')
    })
    total.value = data.total || 0
  } catch {
    list.value = []
  } finally {
    loading.value = false
  }
}

function resetFilter() {
  filter.value = { keyword: '', timeRange: null }
  page.value = 1
  fetch()
}

onMounted(() => fetch())
</script>
<style scoped>
.audit-log-list { padding: 0; }
.filter-bar { display: flex; gap: 10px; flex-wrap: wrap; align-items: center; margin-bottom: 16px; }
.pagination { margin-top: 16px; display: flex; justify-content: flex-end; }
.mobile-log-card { background: #fff; border-radius: 8px; padding: 12px; margin-bottom: 10px; box-shadow: 0 1px 4px rgba(0,0,0,.08); }
.mobile-card-row { display: flex; justify-content: space-between; padding: 4px 0; font-size: 13px; }
.mobile-label { color: #909399; flex-shrink: 0; margin-right: 12px; }
.mobile-value { text-align: right; word-break: break-all; }
.mobile-value-wrap { max-width: 65%; }
.audit-data { font-size: 12px; line-height: 1.6; word-break: break-all; }
.text-muted { color: #909399; font-size: 11px; }
</style>
