<template>
  <div class="list-container">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>用户行为分析</span>
          <div class="header-actions">
            <el-button @click="exportData" :loading="exporting">
              <el-icon><Download /></el-icon>
              导出数据
            </el-button>
            <el-button @click="loadData" :loading="loading">
              <el-icon><Refresh /></el-icon>
              刷新数据
            </el-button>
          </div>
        </div>
      </template>

      <!-- 时间范围切换 -->
      <div class="time-range-selector">
        <el-radio-group v-model="timeRange" @change="loadData">
          <el-radio-button label="day">今日</el-radio-button>
          <el-radio-button label="month">本月</el-radio-button>
          <el-radio-button label="year">本年</el-radio-button>
        </el-radio-group>
      </div>

      <!-- 概览卡片 -->
      <el-row :gutter="16" class="stats-row">
        <el-col :xs="12" :sm="6" v-for="item in overviewCards" :key="item.label">
          <div class="stat-card-mini">
            <div class="stat-icon" :style="{ background: item.color }">
              <el-icon><component :is="item.icon" /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ item.value }}</div>
              <div class="stat-label">{{ item.label }}</div>
            </div>
          </div>
        </el-col>
      </el-row>

      <!-- 收入统计 -->
      <el-card header="收入统计" shadow="never" class="data-card" style="margin-top: 20px">
        <el-row :gutter="16">
          <el-col :xs="24" :sm="8">
            <div class="income-stat">
              <div class="income-label">{{ timeRangeLabel }}收入</div>
              <div class="income-value">¥{{ revenueStats.current || '0.00' }}</div>
              <div class="income-trend" :class="revenueTrendClass">
                <el-icon><component :is="revenueTrendIcon" /></el-icon>
                <span>{{ revenueTrendText }}</span>
              </div>
            </div>
          </el-col>
          <el-col :xs="24" :sm="8">
            <div class="income-stat">
              <div class="income-label">订单数量</div>
              <div class="income-value">{{ revenueStats.order_count || 0 }}</div>
              <div class="income-desc">{{ timeRangeLabel }}成交订单</div>
            </div>
          </el-col>
          <el-col :xs="24" :sm="8">
            <div class="income-stat">
              <div class="income-label">平均订单金额</div>
              <div class="income-value">¥{{ revenueStats.avg_order || '0.00' }}</div>
              <div class="income-desc">单笔订单均值</div>
            </div>
          </el-col>
        </el-row>
      </el-card>

      <!-- 数据图表 -->
      <el-row :gutter="16" style="margin-top: 20px">
        <el-col :xs="24" :lg="12">
          <el-card header="用户留存分析" shadow="never" class="data-card">
            <el-table :data="retention" stripe size="small" v-loading="loading">
              <el-table-column label="指标" min-width="90" align="center">
                <template #default="{ row }">{{ row.label || `第${row.day}天` }}</template>
              </el-table-column>
              <el-table-column prop="total" label="基数" width="80" align="center" />
              <el-table-column prop="retained" label="达标" width="80" align="center" />
              <el-table-column label="留存率">
                <template #default="{ row }">
                  <el-progress
                    :percentage="Math.round(row.rate)"
                    :stroke-width="14"
                    :text-inside="true"
                    :color="getProgressColor(row.rate)"
                  />
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-col>

        <el-col :xs="24" :lg="12">
          <el-card header="设备分析" shadow="never" class="data-card">
            <div v-loading="loading">
              <div v-if="deviceStats.length > 0">
                <h4 class="section-title">设备类型分布</h4>
                <el-table :data="deviceStats" stripe size="small">
                  <el-table-column label="设备类型">
                    <template #default="{ row }">
                      {{ getDeviceTypeName(row.device_type) }}
                    </template>
                  </el-table-column>
                  <el-table-column prop="count" label="数量" width="100" align="center" />
                  <el-table-column label="占比" width="120">
                    <template #default="{ row }">
                      <el-tag size="small">{{ getPercentage(row.count, totalDevices) }}%</el-tag>
                    </template>
                  </el-table-column>
                </el-table>
              </div>

              <div v-if="osStats.length > 0" style="margin-top: 20px">
                <h4 class="section-title">操作系统分布</h4>
                <el-table :data="osStats" stripe size="small">
                  <el-table-column label="系统">
                    <template #default="{ row }">
                      {{ getDeviceTypeName(row.device_type) }}
                    </template>
                  </el-table-column>
                  <el-table-column prop="count" label="数量" width="100" align="center" />
                  <el-table-column label="占比" width="120">
                    <template #default="{ row }">
                      <el-tag size="small" type="success">{{ getPercentage(row.count, totalOS) }}%</el-tag>
                    </template>
                  </el-table-column>
                </el-table>
              </div>

              <el-empty v-if="deviceStats.length === 0 && osStats.length === 0" description="暂无数据" />
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 流失预警 -->
      <el-card header="流失预警用户" shadow="never" class="data-card" style="margin-top: 20px">
        <el-table :data="churnUsers" stripe size="small" v-loading="loading">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="username" label="用户名" min-width="120" />
          <el-table-column prop="email" label="邮箱" min-width="180" show-overflow-tooltip />
          <el-table-column label="最后登录" width="160">
            <template #default="{ row }">
              <span :class="{ 'text-danger': isLongTimeNoLogin(row.last_login) }">
                {{ formatDate(row.last_login) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="订阅到期" width="160">
            <template #default="{ row }">
              <span :class="{ 'text-warning': isExpiringSoon(row.expire_time) }">
                {{ formatDate(row.expire_time) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" @click="openContactDialog(row)">联系</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="churnUsers.length === 0 && !loading" description="暂无预警用户" />
      </el-card>
    </el-card>

    <!-- 联系用户抽屉 -->
    <el-drawer
      v-model="contactDialogVisible"
      title="联系用户"
      :size="isMobile ? '100%' : '600px'"
      :close-on-click-modal="false"
      :lock-scroll="false"
    >
      <el-form :model="contactForm" label-width="100px">
        <el-form-item label="用户">
          <el-input v-model="contactForm.username" disabled />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="contactForm.email" disabled />
        </el-form-item>
        <el-form-item label="邮件类型">
          <el-select v-model="contactForm.templateName" placeholder="请选择邮件模板" @change="onTemplateChange" style="width: 100%">
            <el-option
              v-for="template in emailTemplates"
              :key="template.name"
              :label="template.label"
              :value="template.name"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="邮件主题">
          <el-input v-model="contactForm.subject" placeholder="请输入邮件主题" />
        </el-form-item>
        <el-form-item label="邮件内容">
          <el-input
            v-model="contactForm.content"
            type="textarea"
            :rows="isMobile ? 10 : 8"
            placeholder="请输入邮件内容"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="drawer-footer">
          <el-button @click="contactDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="sendEmail" :loading="sending">发送邮件</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { Refresh, User, TrendCharts, DataAnalysis, Monitor, Download, Top, Bottom } from '@element-plus/icons-vue'
import { api } from '@/utils/api'
import { useRouter } from 'vue-router'

const router = useRouter()
const loading = ref(false)
const exporting = ref(false)
const sending = ref(false)
const timeRange = ref('day')
const contactDialogVisible = ref(false)
const isMobile = ref(false)

// 检测是否为移动端
const checkMobile = () => {
  isMobile.value = window.innerWidth <= 768
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
  loadData()
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})

const userAnalytics = ref({})
const retention = ref([])
const churnUsers = ref([])
const deviceStats = ref([])
const osStats = ref([])
const revenueStats = ref({})
let loadSeq = 0
const contactForm = ref({
  userId: null,
  username: '',
  email: '',
  templateName: '',
  subject: '',
  content: ''
})

const emailTemplates = ref([
  { name: 'subscription_expiring', label: '订阅即将到期提醒' },
  { name: 'subscription_expired', label: '订阅已到期通知' },
  { name: 'user_recall', label: '流失用户召回' },
  { name: 'custom', label: '自定义邮件' }
])

const timeRangeLabel = computed(() => {
  const labels = { day: '今日', month: '本月', year: '本年' }
  return labels[timeRange.value] || '今日'
})

const overviewCards = computed(() => [
  {
    label: '日活跃(DAU)',
    value: userAnalytics.value.dau || 0,
    icon: User,
    color: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)'
  },
  {
    label: '周活跃(WAU)',
    value: userAnalytics.value.wau || 0,
    icon: TrendCharts,
    color: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)'
  },
  {
    label: '月活跃(MAU)',
    value: userAnalytics.value.mau || 0,
    icon: DataAnalysis,
    color: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)'
  },
  {
    label: '总用户数',
    value: userAnalytics.value.total_users || 0,
    icon: Monitor,
    color: 'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)'
  }
])

const revenueTrendClass = computed(() => {
  const change = revenueStats.value.change_rate || 0
  return change >= 0 ? 'trend-up' : 'trend-down'
})

const revenueTrendIcon = computed(() => {
  const change = revenueStats.value.change_rate || 0
  return change >= 0 ? Top : Bottom
})

const revenueTrendText = computed(() => {
  const change = revenueStats.value.change_rate || 0
  const prefix = change >= 0 ? '较上期增长' : '较上期下降'
  return `${prefix} ${Math.abs(change).toFixed(1)}%`
})

const totalDevices = computed(() => {
  return deviceStats.value.reduce((sum, item) => sum + item.count, 0)
})

const totalOS = computed(() => {
  return osStats.value.reduce((sum, item) => sum + item.count, 0)
})

const deviceTypeMap = {
  'mobile': '移动设备',
  'desktop': '桌面设备',
  'tablet': '平板设备',
  'unknown': '未知设备',
  'ios': 'iOS',
  'android': 'Android',
  'windows': 'Windows',
  'macos': 'macOS',
  'linux': 'Linux'
}

const getDeviceTypeName = (type) => {
  return deviceTypeMap[type?.toLowerCase()] || type || '未知'
}

const formatDate = (d) => {
  if (!d) return '-'
  const date = new Date(d)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  })
}

const getPercentage = (value, total) => {
  if (total === 0) return 0
  return ((value / total) * 100).toFixed(1)
}

const getProgressColor = (rate) => {
  if (rate >= 70) return '#67c23a'
  if (rate >= 40) return '#e6a23c'
  return '#f56c6c'
}

const isLongTimeNoLogin = (lastLogin) => {
  if (!lastLogin) return true
  const days = Math.floor((Date.now() - new Date(lastLogin).getTime()) / (1000 * 60 * 60 * 24))
  return days > 7
}

const isExpiringSoon = (expireTime) => {
  if (!expireTime) return false
  const days = Math.floor((new Date(expireTime).getTime() - Date.now()) / (1000 * 60 * 60 * 24))
  return days <= 7 && days >= 0
}


const openContactDialog = (user) => {
  contactForm.value = {
    userId: user.id,
    username: user.username,
    email: user.email,
    templateName: '',
    subject: '',
    content: ''
  }

  // 根据用户状态自动选择邮件模板
  const now = Date.now()
  const expireTime = new Date(user.expire_time).getTime()
  const daysDiff = Math.floor((expireTime - now) / (1000 * 60 * 60 * 24))

  if (daysDiff < 0) {
    // 已到期
    contactForm.value.templateName = 'subscription_expired'
  } else if (daysDiff <= 7) {
    // 即将到期
    contactForm.value.templateName = 'subscription_expiring'
  } else if (isLongTimeNoLogin(user.last_login)) {
    // 流失用户
    contactForm.value.templateName = 'user_recall'
  }

  onTemplateChange()
  contactDialogVisible.value = true
}

const onTemplateChange = async () => {
  if (!contactForm.value.templateName || contactForm.value.templateName === 'custom') {
    return
  }

  try {
    const res = await api.get(`/admin/email-templates/${contactForm.value.templateName}`)

    if (res.data?.data) {
      const template = res.data.data
      contactForm.value.subject = template.subject
      contactForm.value.content = template.content
    }
  } catch (e) {
    console.error('获取邮件模板失败:', e)
  }
}

const sendEmail = async () => {
  if (!contactForm.value.subject || !contactForm.value.content) {
    ElMessage.warning('请填写邮件主题和内容')
    return
  }

  sending.value = true
  try {
    await api.post('/admin/users/send-email', {
      user_id: contactForm.value.userId,
      email: contactForm.value.email,
      subject: contactForm.value.subject,
      content: contactForm.value.content,
      template_name: contactForm.value.templateName
    })

    ElMessage.success('邮件发送成功')
    contactDialogVisible.value = false
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '邮件发送失败')
  } finally {
    sending.value = false
  }
}

const contactUser = (user) => {
  router.push(`/admin/users?search=${user.email}`)
}

const loadData = async () => {
  const seq = ++loadSeq
  const range = timeRange.value
  loading.value = true
  try {
    const [revenueRes, uRes, rRes, cRes, dRes] = await Promise.all([
      api.get(`/admin/analytics/revenue?range=${range}`),
      api.get(`/admin/analytics/users?range=${range}`),
      api.get(`/admin/analytics/retention?range=${range}`),
      api.get(`/admin/analytics/churn?range=${range}`),
      api.get(`/admin/analytics/devices?range=${range}`)
    ])
    if (seq !== loadSeq) return

    revenueStats.value = revenueRes.data?.data || {}
    userAnalytics.value = uRes.data?.data || {}
    retention.value = rRes.data?.data || []
    churnUsers.value = cRes.data?.data || []

    const devData = dRes.data?.data || {}
    deviceStats.value = devData.device_types || []
    osStats.value = devData.os_stats || []
  } catch (e) {
    console.error('加载数据失败:', e)
    console.error('错误详情:', e.response?.data)
    ElMessage.error(e.response?.data?.message || '加载数据失败')
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

const exportData = async () => {
  exporting.value = true
  try {
    // 构建 CSV 内容
    let csvContent = '\uFEFF' // UTF-8 BOM for Excel

    // 1. 基本信息
    csvContent += '用户分析数据导出\n'
    csvContent += `导出时间,${new Date().toLocaleString('zh-CN')}\n`
    csvContent += `时间范围,${timeRangeLabel.value}\n\n`

    // 2. 收入统计
    csvContent += '收入统计\n'
    csvContent += '指标,数值\n'
    csvContent += `${timeRangeLabel.value}收入,¥${revenueStats.value.current || '0.00'}\n`
    csvContent += `订单数量,${revenueStats.value.order_count || 0}\n`
    csvContent += `平均订单金额,¥${revenueStats.value.avg_order || '0.00'}\n`
    csvContent += `较上期变化,${revenueStats.value.change_rate || 0}%\n\n`

    // 3. 用户活跃度
    csvContent += '用户活跃度统计\n'
    csvContent += '指标,数值\n'
    csvContent += `日活跃用户(DAU),${userAnalytics.value.dau || 0}\n`
    csvContent += `周活跃用户(WAU),${userAnalytics.value.wau || 0}\n`
    csvContent += `月活跃用户(MAU),${userAnalytics.value.mau || 0}\n`
    csvContent += `总用户数,${userAnalytics.value.total_users || 0}\n\n`

    // 4. 用户留存分析
    csvContent += '用户留存分析\n'
    csvContent += '指标,基数,达标,比率\n'
    retention.value.forEach(row => {
      csvContent += `${row.label || '第' + row.day + '天'},${row.total},${row.retained},${row.rate.toFixed(1)}%\n`
    })
    csvContent += '\n'

    // 5. 设备类型分布
    csvContent += '设备类型分布\n'
    csvContent += '设备类型,数量,占比\n'
    deviceStats.value.forEach(row => {
      const percentage = getPercentage(row.count, totalDevices.value)
      csvContent += `${getDeviceTypeName(row.device_type)},${row.count},${percentage}%\n`
    })
    csvContent += '\n'

    // 6. 操作系统分布
    csvContent += '操作系统分布\n'
    csvContent += '系统,数量,占比\n'
    osStats.value.forEach(row => {
      const percentage = getPercentage(row.count, totalOS.value)
      csvContent += `${getDeviceTypeName(row.device_type)},${row.count},${percentage}%\n`
    })
    csvContent += '\n'

    // 7. 流失预警用户
    csvContent += '流失预警用户\n'
    csvContent += 'ID,用户名,邮箱,最后登录,订阅到期\n'
    churnUsers.value.forEach(user => {
      csvContent += `${user.id},${user.username},${user.email},${formatDate(user.last_login)},${formatDate(user.expire_time)}\n`
    })

    // 创建并下载 CSV 文件
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `用户分析数据_${timeRangeLabel.value}_${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)

    ElMessage.success('数据导出成功')
  } catch (e) {
    ElMessage.error('导出失败')
  } finally {
    exporting.value = false
  }
}

</script>

<style scoped>
.list-container {
  padding: 0;
}

.list-card {
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.time-range-selector {
  margin-bottom: 20px;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 8px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.stats-row {
  margin: -8px;
}

.stats-row .el-col {
  padding: 8px;
}

.stat-card-mini {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  transition: all 0.3s;
}

.stat-card-mini:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 24px;
  flex-shrink: 0;
}

.stat-info {
  flex: 1;
  min-width: 0;
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: #303133;
  line-height: 1.2;
}

.stat-label {
  font-size: 13px;
  color: #909399;
  margin-top: 4px;
}

.income-stat {
  padding: 20px;
  background: #f5f7fa;
  border-radius: 8px;
  text-align: center;
}

.income-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}

.income-value {
  font-size: 28px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 8px;
}

.income-trend {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-size: 13px;
}

.income-trend.trend-up {
  color: #67c23a;
}

.income-trend.trend-down {
  color: #f56c6c;
}

.income-desc {
  font-size: 13px;
  color: #909399;
  margin-top: 8px;
}

.data-card {
  border: 1px solid #ebeef5;
  border-radius: 8px;
}

.section-title {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.text-danger {
  color: #f56c6c;
}

.text-warning {
  color: #e6a23c;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .header-actions {
    flex-direction: column;
    gap: 8px;
  }

  .header-actions .el-button {
    width: 100%;
  }

  .time-range-selector {
    padding: 12px;
  }

  .time-range-selector :deep(.el-radio-group) {
    display: flex;
    width: 100%;
  }

  .time-range-selector :deep(.el-radio-button) {
    flex: 1;
  }

  .time-range-selector :deep(.el-radio-button__inner) {
    width: 100%;
  }

  .stats-row {
    margin: -4px;
  }

  .stats-row .el-col {
    padding: 4px;
  }

  .stat-card-mini {
    padding: 16px;
    gap: 12px;
  }

  .stat-icon {
    width: 40px;
    height: 40px;
    font-size: 20px;
  }

  .stat-value {
    font-size: 20px;
  }

  .stat-label {
    font-size: 12px;
  }

  .income-stat {
    margin-bottom: 12px;
  }

  .income-value {
    font-size: 24px;
  }

  .income-label {
    font-size: 13px;
  }

  .income-trend {
    font-size: 12px;
  }

  .drawer-footer {
    flex-direction: column;
  }

  .drawer-footer .el-button {
    width: 100%;
  }

  :deep(.el-drawer__header) {
    padding: 16px;
    margin-bottom: 0;
  }

  :deep(.el-drawer__body) {
    padding: 16px;
  }

  :deep(.el-form-item__label) {
    font-size: 14px;
  }

  :deep(.el-input__inner) {
    font-size: 16px !important;
  }

  :deep(.el-textarea__inner) {
    font-size: 16px !important;
  }

  .data-card {
    margin-top: 16px;
  }

  :deep(.el-table) {
    font-size: 12px;
  }

  :deep(.el-table .el-button) {
    padding: 5px 8px;
    font-size: 12px;
  }
}
</style>
