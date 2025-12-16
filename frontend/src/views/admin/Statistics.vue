<template>
    <div class="statistics-admin-container">
      <!-- 统计卡片 -->
      <el-row :gutter="20" class="stats-cards">
        <el-col :xs="24" :sm="12" :md="6">
          <el-card class="stat-card">
            <div class="stat-content">
              <div class="stat-icon users">
                <i class="el-icon-user"></i>
              </div>
              <div class="stat-info">
                <div class="stat-number">{{ statistics.totalUsers }}</div>
                <div class="stat-label">总用户数</div>
              </div>
            </div>
          </el-card>
        </el-col>
        
        <el-col :xs="24" :sm="12" :md="6">
          <el-card class="stat-card">
            <div class="stat-content">
              <div class="stat-icon subscriptions">
                <i class="el-icon-connection"></i>
              </div>
              <div class="stat-info">
                <div class="stat-number">{{ statistics.activeSubscriptions }}</div>
                <div class="stat-label">活跃订阅</div>
              </div>
            </div>
          </el-card>
        </el-col>
        
        <el-col :xs="24" :sm="12" :md="6">
          <el-card class="stat-card">
            <div class="stat-content">
              <div class="stat-icon orders">
                <i class="el-icon-shopping-cart-2"></i>
              </div>
              <div class="stat-info">
                <div class="stat-number">{{ statistics.totalOrders }}</div>
                <div class="stat-label">总订单数</div>
              </div>
            </div>
          </el-card>
        </el-col>
        
        <el-col :xs="24" :sm="12" :md="6">
          <el-card class="stat-card">
            <div class="stat-content">
              <div class="stat-icon revenue">
                <i class="el-icon-money"></i>
              </div>
              <div class="stat-info">
                <div class="stat-number">¥{{ formatMoney(statistics.totalRevenue) }}</div>
                <div class="stat-label">总收入</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
  
      <!-- 图表区域 -->
      <el-row :gutter="20" class="charts-section">
        <el-col :xs="24" :sm="24" :md="12">
          <el-card>
            <template #header>
              <div class="card-header">
                <h3>用户注册趋势</h3>
              </div>
            </template>
            <div class="chart-container">
              <canvas ref="userChart"></canvas>
            </div>
          </el-card>
        </el-col>
        
        <el-col :xs="24" :sm="24" :md="12">
          <el-card>
            <template #header>
              <div class="card-header">
                <h3>收入统计</h3>
              </div>
            </template>
            <div class="chart-container">
              <canvas ref="revenueChart"></canvas>
            </div>
          </el-card>
        </el-col>
      </el-row>
  
      <!-- 详细统计 -->
      <el-row :gutter="20" class="detailed-stats">
        <el-col :xs="24" :sm="24" :md="12">
          <el-card>
            <template #header>
              <div class="card-header">
                <h3>用户统计</h3>
              </div>
            </template>
            
            <!-- 桌面端表格 -->
            <div class="desktop-only">
              <el-table :data="userStats" style="width: 100%">
                <el-table-column prop="label" label="统计项" />
                <el-table-column prop="value" label="数值" />
                <el-table-column prop="percentage" label="占比">
                  <template #default="{ row }">
                    <el-progress
                      :percentage="row.percentage"
                      :color="row.color"
                      :show-text="false"
                    />
                    <span style="margin-left: 10px">{{ row.percentage }}%</span>
                  </template>
                </el-table-column>
              </el-table>
            </div>
            
            <!-- 移动端卡片列表 -->
            <div class="mobile-stats-list mobile-only">
              <div 
                v-for="stat in userStats" 
                :key="stat.label"
                class="mobile-stat-item"
              >
                <div class="stat-item-header">
                  <span class="stat-item-label">{{ stat.label }}</span>
                  <span class="stat-item-value">{{ stat.value }}</span>
                </div>
                <div class="stat-item-progress">
                  <el-progress
                    :percentage="stat.percentage"
                    :color="stat.color"
                    :show-text="false"
                  />
                  <span class="stat-item-percentage">{{ stat.percentage }}%</span>
                </div>
              </div>
            </div>
          </el-card>
        </el-col>
        
        <el-col :xs="24" :sm="24" :md="12">
          <el-card>
            <template #header>
              <div class="card-header">
                <h3>订阅统计</h3>
              </div>
            </template>
            
            <!-- 桌面端表格 -->
            <div class="desktop-only">
              <el-table :data="subscriptionStats" style="width: 100%">
                <el-table-column prop="label" label="统计项" />
                <el-table-column prop="value" label="数值" />
                <el-table-column prop="percentage" label="占比">
                  <template #default="{ row }">
                    <el-progress
                      :percentage="row.percentage"
                      :color="row.color"
                      :show-text="false"
                    />
                    <span style="margin-left: 10px">{{ row.percentage }}%</span>
                  </template>
                </el-table-column>
              </el-table>
            </div>
            
            <!-- 移动端卡片列表 -->
            <div class="mobile-stats-list mobile-only">
              <div 
                v-for="stat in subscriptionStats" 
                :key="stat.label"
                class="mobile-stat-item"
              >
                <div class="stat-item-header">
                  <span class="stat-item-label">{{ stat.label }}</span>
                  <span class="stat-item-value">{{ stat.value }}</span>
                </div>
                <div class="stat-item-progress">
                  <el-progress
                    :percentage="stat.percentage"
                    :color="stat.color"
                    :show-text="false"
                  />
                  <span class="stat-item-percentage">{{ stat.percentage }}%</span>
                </div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
  
      <!-- 最近活动 -->
      <el-card class="recent-activities">
        <template #header>
          <div class="card-header">
            <h3>最近活动</h3>
          </div>
        </template>
        
        <el-timeline>
          <el-timeline-item
            v-for="activity in recentActivities"
            :key="activity.id"
            :timestamp="activity.time"
            :type="activity.type"
          >
            <div class="activity-content">
              <div class="activity-title">{{ activity.title }}</div>
              <div class="activity-description">{{ activity.description }}</div>
            </div>
          </el-timeline-item>
        </el-timeline>
      </el-card>
    </div>
  </template>
  
  <script>
  import { ref, reactive, onMounted } from 'vue'
  import { Chart, registerables } from 'chart.js'
  import { statisticsAPI } from '@/utils/api'
  
  Chart.register(...registerables)
  
  export default {
    name: 'AdminStatistics',
    setup() {
      const userChart = ref(null)
      const revenueChart = ref(null)
      
      const statistics = reactive({
        totalUsers: 0,
        activeSubscriptions: 0,
        totalOrders: 0,
        totalRevenue: 0
      })
  
      const userStats = ref([])
      const subscriptionStats = ref([])
      const recentActivities = ref([])
  
      // 获取统计数据
      const fetchStatistics = async () => {
        try {
          const response = await statisticsAPI.getStatistics()
          if (response.data && response.data.data) {
            const data = response.data.data
            // 更新概览数据
            if (data.overview) {
              Object.assign(statistics, data.overview)
              }
            
            // 更新用户统计
            if (data.userStats) {
              userStats.value = data.userStats.map(stat => ({
                label: stat.name,
                value: stat.value,
                percentage: stat.percentage,
                color: '#409eff'
              }))
              }
            
            // 更新订阅统计
            if (data.subscriptionStats) {
              subscriptionStats.value = data.subscriptionStats.map(stat => ({
                label: stat.name,
                value: stat.value,
                percentage: stat.percentage,
                color: '#67c23a'
              }))
              }
            
            // 更新最近活动
            if (data.recentActivities) {
              // 将活动类型映射到 Element Plus 有效的类型值
              const mapActivityType = (activityType, status) => {
                // 如果类型已经是有效的 Element Plus 类型，直接返回
                const validTypes = ['primary', 'success', 'warning', 'danger', 'info', '']
                if (validTypes.includes(activityType)) {
                  return activityType
                }
                // 根据订单状态映射类型
                if (status === 'paid') return 'success'
                if (status === 'pending') return 'warning'
                if (status === 'cancelled' || status === 'refunded') return 'danger'
                return 'primary'
              }
              
              recentActivities.value = data.recentActivities.map(activity => ({
                id: activity.id,
                type: mapActivityType(activity.type, activity.status),
                title: activity.description,
                description: `金额: ¥${formatMoney(activity.amount)} | 状态: ${activity.status}`,
                time: activity.time
              }))
              }
          } else {
            }
        } catch (error) {
          }
      }
  
      // 初始化用户注册趋势图表
      const initUserChart = async () => {
        try {
          const response = await statisticsAPI.getUserTrend()
          const ctx = userChart.value.getContext('2d')
          
          new Chart(ctx, {
            type: 'line',
            data: {
              labels: response.data.labels,
              datasets: [{
                label: '新用户注册',
                data: response.data.data,
                borderColor: '#409eff',
                backgroundColor: 'rgba(64, 158, 255, 0.1)',
                tension: 0.4
              }]
            },
            options: {
              responsive: true,
              maintainAspectRatio: false,
              plugins: {
                legend: {
                  display: false
                }
              },
              scales: {
                y: {
                  beginAtZero: true
                }
              }
            }
          })
        } catch (error) {
          }
      }
  
      // 初始化收入统计图表
      const initRevenueChart = async () => {
        try {
          const response = await statisticsAPI.getRevenueTrend()
          const ctx = revenueChart.value.getContext('2d')
          
          new Chart(ctx, {
            type: 'bar',
            data: {
              labels: response.data.labels,
              datasets: [{
                label: '收入',
                data: response.data.data,
                backgroundColor: '#67c23a',
                borderColor: '#67c23a',
                borderWidth: 1
              }]
            },
            options: {
              responsive: true,
              maintainAspectRatio: false,
              plugins: {
                legend: {
                  display: false
                }
              },
              scales: {
                y: {
                  beginAtZero: true
                }
              }
            }
          })
        } catch (error) {
          }
      }
  
      const formatMoney = (value) => {
        if (value === null || value === undefined || value === '') return '0.00'
        const num = typeof value === 'string' ? parseFloat(value) : value
        if (isNaN(num)) return '0.00'
        return num.toFixed(2)
      }

      onMounted(() => {
        fetchStatistics()
        initUserChart()
        initRevenueChart()
      })

      return {
        userChart,
        revenueChart,
        statistics,
        userStats,
        subscriptionStats,
        recentActivities,
        formatMoney
      }
    }
  }
  </script>
  
  <style scoped>
  .statistics-admin-container {
    padding: 20px;
  }
  
  .stats-cards {
    margin-bottom: 20px;
  }
  
  .stat-card {
    height: 120px;
  }
  
  .stat-content {
    display: flex;
    align-items: center;
    height: 100%;
  }
  
  .stat-icon {
    width: 60px;
    height: 60px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: 20px;
  }
  
  .stat-icon i {
    font-size: 24px;
    color: white;
  }
  
  .stat-icon.users {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  }
  
  .stat-icon.subscriptions {
    background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  }
  
  .stat-icon.orders {
    background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  }
  
  .stat-icon.revenue {
    background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
  }
  
  .stat-info {
    flex: 1;
  }
  
  .stat-number {
    font-size: 2rem;
    font-weight: 700;
    color: #333;
    margin-bottom: 5px;
  }
  
  .stat-label {
    color: #666;
    font-size: 0.9rem;
  }
  
  .charts-section {
    margin-bottom: 20px;
  }
  
  .chart-container {
    height: 300px;
    position: relative;
  }
  
  .card-header h3 {
    margin: 0;
    color: #333;
    font-size: 1.2rem;
  }
  
  .detailed-stats {
    margin-bottom: 20px;
  }
  
  .recent-activities {
    margin-bottom: 20px;
  }
  
  .activity-content {
    padding: 10px 0;
  }
  
  .activity-title {
    font-weight: 600;
    color: #333;
    margin-bottom: 5px;
  }
  
  .activity-description {
    color: #666;
    font-size: 0.9rem;
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
  }

  @media (max-width: 768px) {
    .statistics-admin-container {
      padding: 10px;
    }
    
    .stats-cards {
      margin-bottom: 16px;
      
      .el-col {
        margin-bottom: 12px;
      }
    }
    
    .stat-card {
      height: auto;
      min-height: 100px;
    }
    
    .stat-content {
      padding: 12px;
      flex-direction: row;
      align-items: center;
    }
    
    .stat-icon {
      width: 50px;
      height: 50px;
      margin-right: 16px;
      flex-shrink: 0;
      
      :is(i) {
        font-size: 20px;
      }
    }
    
    .stat-info {
      flex: 1;
      min-width: 0;
    }
    
    .stat-number {
      font-size: 1.8rem;
      font-weight: 700;
      margin-bottom: 4px;
      word-break: break-all;
    }
    
    .stat-label {
      font-size: 14px;
      color: #666;
    }
    
    .charts-section {
      margin-bottom: 16px;
      
      .el-col {
        margin-bottom: 16px;
      }
    }
    
    .chart-container {
      height: 280px;
      padding: 8px;
    }
    
    .card-header {
      padding: 12px 0;
      
      :is(h3) {
        font-size: 16px;
        font-weight: 600;
        margin: 0;
      }
    }
    
    .detailed-stats {
      margin-bottom: 16px;
      
      .el-col {
        margin-bottom: 16px;
      }
    }
    
    /* 移动端统计列表 */
    .mobile-stats-list {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    
    .mobile-stat-item {
      padding: 12px;
      background: #f8f9fa;
      border-radius: 8px;
      border: 1px solid #e9ecef;
    }
    
    .stat-item-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 10px;
      
      .stat-item-label {
        font-weight: 600;
        color: #606266;
        font-size: 14px;
      }
      
      .stat-item-value {
        font-weight: 700;
        color: #303133;
        font-size: 16px;
      }
    }
    
    .stat-item-progress {
      display: flex;
      align-items: center;
      gap: 10px;
      
      .el-progress {
        flex: 1;
      }
      
      .stat-item-percentage {
        font-size: 14px;
        color: #606266;
        min-width: 45px;
        text-align: right;
      }
    }
    
    .recent-activities {
      margin-bottom: 16px;
    }
    
    .activity-content {
      padding: 8px 0;
    }
    
    .activity-title {
      font-size: 14px;
      font-weight: 600;
      margin-bottom: 6px;
    }
    
    .activity-description {
      font-size: 13px;
      color: #666;
      line-height: 1.5;
    }
    
    /* 时间线优化 */
    :deep(.el-timeline-item) {
      padding-bottom: 16px;
    }
    
    :deep(.el-timeline-item__timestamp) {
      font-size: 12px;
      color: #909399;
      margin-bottom: 8px;
    }
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
</style>