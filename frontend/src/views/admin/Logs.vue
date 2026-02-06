<template>
  <div class="logs-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <h2>日志管理</h2>
          <p>查看和管理各类系统日志</p>
        </div>
      </template>

      <!-- 日志分类标签页 -->
      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
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
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import RegistrationLogs from './logs/RegistrationLogs.vue'
import SubscriptionLogs from './logs/SubscriptionLogs.vue'
import BalanceLogs from './logs/BalanceLogs.vue'
import CommissionLogs from './logs/CommissionLogs.vue'
import SubscriptionResetLogs from './logs/SubscriptionResetLogs.vue'
import EmailLogs from './logs/EmailLogs.vue'

const activeTab = ref('registration')

const handleTabChange = (tabName) => {
  activeTab.value = tabName
}
</script>

<style scoped>
.logs-container {
  padding: 20px;
}

.card-header h2 {
  margin: 0;
  color: #333;
}

.card-header p {
  margin: 5px 0 0 0;
  color: #666;
  font-size: 14px;
}
</style>
