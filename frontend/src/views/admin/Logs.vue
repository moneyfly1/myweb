<template>
  <div class="list-container admin-logs">
    <el-card>
      <template #header>
        <div class="card-header">
          <h2>日志管理</h2>
          <p>查看和管理各类系统日志</p>
        </div>
      </template>
      <!-- 全局搜索：一个搜索框把这个用户的**全部**日志一次搜出来（不再逐个 Tab 搜） -->
      <GlobalLogSearch v-model:searching="searching" />

      <!-- 看全局搜索结果时隐藏分类 Tab（重置后恢复），避免"搜索完回不去" -->
      <el-tabs v-if="!searching" v-model="activeTab" class="logs-tabs-host tabs-wrap-mobile">
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


defineOptions({ name: 'AdminLogs' })

/// 是否正在看全局搜索结果（GlobalLogSearch 通过 v-model:searching 同步）
const searching = ref(false)

import { ref } from 'vue'
import GlobalLogSearch from './logs/GlobalLogSearch.vue'
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
/* 说明：原先这里强制「7 个 tab 单行横向滚动」（nav-wrap overflow-x:auto + nav min-width:max-content）。
   这些 scoped 规则的权重高于全局的 .tabs-wrap-mobile，把移动端「换行胶囊」覆盖掉了 ——
   实测 390px 下 tab 条宽 599px、被裁在 264px 的容器里，后面的「管理员操作日志」点不到。
   现在移动端统一由全局 .tabs-wrap-mobile 处理（换行、全部可见），桌面端用 Element Plus 默认行为。 */
</style>
