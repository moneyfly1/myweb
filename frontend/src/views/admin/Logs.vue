<template>
  <div class="list-container admin-logs">
    <el-card>
      <template #header>
        <div class="card-header">
          <h2>日志管理</h2>
          <p>查看和管理各类系统日志</p>
        </div>
      </template>
      <!-- 移动端：7 个 tab 横排 ≈654px（390px 屏幕只露 3 个）→ 下拉导航 -->
      <MobileTabSelect v-model="activeTab" :tabs="logTabs" />
      <el-tabs v-model="activeTab" class="logs-tabs-host hide-tabs-mobile">
        <el-tab-pane label="注册日志" name="registration">
          <RegistrationLogs />
        </el-tab-pane>
        <el-tab-pane label="订阅日志" name="subscription">
          <SubscriptionLogs />
        </el-tab-pane>
        <el-tab-pane label="余额日志" name="balance">
          <BalanceLogs />
        </el-tab-pane>
        <el-tab-pane label="佣金日志" name="commission">
          <CommissionLogs />
        </el-tab-pane>
        <el-tab-pane label="重置订阅日志" name="subscription-reset">
          <SubscriptionResetLogs />
        </el-tab-pane>
        <el-tab-pane label="邮件日志" name="email">
          <EmailLogs />
        </el-tab-pane>
        <el-tab-pane label="管理员操作日志" name="audit">
          <AuditLogs />
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>
<script setup>
import MobileTabSelect from '@/components/MobileTabSelect.vue'

// 与下方 el-tab-pane 的 name/label 一一对应（新增 tab 时两处都要加）
const logTabs = [
  { name: 'registration', label: '注册日志' },
  { name: 'subscription', label: '订阅日志' },
  { name: 'balance', label: '余额日志' },
  { name: 'commission', label: '佣金日志' },
  { name: 'subscription-reset', label: '重置订阅日志' },
  { name: 'email', label: '邮件日志' },
  { name: 'audit', label: '管理员操作日志' }
]

defineOptions({ name: 'AdminLogs' })

import { ref } from 'vue'
import RegistrationLogs from './logs/RegistrationLogs.vue'
import SubscriptionLogs from './logs/SubscriptionLogs.vue'
import BalanceLogs from './logs/BalanceLogs.vue'
import CommissionLogs from './logs/CommissionLogs.vue'
import SubscriptionResetLogs from './logs/SubscriptionResetLogs.vue'
import EmailLogs from './logs/EmailLogs.vue'
import AuditLogs from './logs/AuditLogs.vue'
const activeTab = ref('registration')
</script>

<style scoped>
/* 移动端：7 个日志 tab 超宽时横向滚动，确保能切到后面的 tab（如管理员操作日志） */
.logs-tabs-host :deep(.el-tabs__nav-wrap) {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}
.logs-tabs-host :deep(.el-tabs__nav-wrap::after) {
  display: none;
}
.logs-tabs-host :deep(.el-tabs__nav) {
  min-width: max-content;
}
.logs-tabs-host :deep(.el-tabs__item) {
  white-space: nowrap;
}
</style>
