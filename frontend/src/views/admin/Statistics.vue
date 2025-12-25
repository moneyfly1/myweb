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
  
      <!-- 标签页：用户统计、地区分析 -->
      <el-tabs v-model="activeTab" type="border-card" class="statistics-tabs" style="margin-top: 20px;">
        <el-tab-pane label="用户统计" name="users">
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
              <el-table :data="userStats" style="width: 100%" v-if="userStats.length > 0">
                <el-table-column prop="label" label="统计项" />
                <el-table-column prop="value" label="数值" />
                <el-table-column prop="percentage" label="占比">
                  <template #default="{ row }">
                    <el-progress
                      :percentage="Math.round(row.percentage * 10) / 10"
                      :color="row.color"
                      :show-text="false"
                    />
                    <span style="margin-left: 10px">{{ Math.round(row.percentage * 10) / 10 }}%</span>
                  </template>
                </el-table-column>
              </el-table>
              <el-empty v-else description="暂无数据" />
            </div>
            
            <!-- 移动端卡片列表 -->
            <div class="mobile-stats-list mobile-only">
              <div 
                v-for="stat in userStats" 
                :key="stat.label"
                class="mobile-stat-item"
                v-if="userStats.length > 0"
              >
                <div class="stat-item-header">
                  <span class="stat-item-label">{{ stat.label }}</span>
                  <span class="stat-item-value">{{ stat.value }}</span>
                </div>
                <div class="stat-item-progress">
                  <el-progress
                    :percentage="Math.round(stat.percentage * 10) / 10"
                    :color="stat.color"
                    :show-text="false"
                  />
                  <span class="stat-item-percentage">{{ Math.round(stat.percentage * 10) / 10 }}%</span>
                </div>
              </div>
              <el-empty v-else description="暂无数据" />
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
              <el-table :data="subscriptionStats" style="width: 100%" v-if="subscriptionStats.length > 0">
                <el-table-column prop="label" label="统计项" />
                <el-table-column prop="value" label="数值" />
                <el-table-column prop="percentage" label="占比">
                  <template #default="{ row }">
                    <el-progress
                      :percentage="Math.round(row.percentage * 10) / 10"
                      :color="row.color"
                      :show-text="false"
                    />
                    <span style="margin-left: 10px">{{ Math.round(row.percentage * 10) / 10 }}%</span>
                  </template>
                </el-table-column>
              </el-table>
              <el-empty v-else description="暂无数据" />
            </div>
            
            <!-- 移动端卡片列表 -->
            <div class="mobile-stats-list mobile-only">
              <div 
                v-for="stat in subscriptionStats" 
                :key="stat.label"
                class="mobile-stat-item"
                v-if="subscriptionStats.length > 0"
              >
                <div class="stat-item-header">
                  <span class="stat-item-label">{{ stat.label }}</span>
                  <span class="stat-item-value">{{ stat.value }}</span>
                </div>
                <div class="stat-item-progress">
                  <el-progress
                    :percentage="Math.round(stat.percentage * 10) / 10"
                    :color="stat.color"
                    :show-text="false"
                  />
                  <span class="stat-item-percentage">{{ Math.round(stat.percentage * 10) / 10 }}%</span>
                </div>
              </div>
              <el-empty v-else description="暂无数据" />
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
        
        <el-timeline v-if="recentActivities.length > 0">
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
        <el-empty v-else description="暂无活动记录" />
      </el-card>
        </el-tab-pane>

        <!-- 地区分析 -->
        <el-tab-pane label="地区分析" name="regions">
          <el-card style="margin-bottom: 20px;">
            <template #header>
              <div class="card-header" style="display: flex; justify-content: space-between; align-items: center;">
                <h3 style="margin: 0;">用户地区分布</h3>
                <el-button type="primary" size="small" @click="loadRegionStats" :loading="loadingRegions">
                  <el-icon style="margin-right: 4px;"><Refresh /></el-icon>
                  刷新数据
                </el-button>
              </div>
            </template>
            
            <div v-if="loadingRegions" style="text-align: center; padding: 40px;">
              <el-icon class="is-loading" style="font-size: 32px; color: #409eff;"><Loading /></el-icon>
              <p style="margin-top: 10px; color: #909399;">正在加载地区数据...</p>
            </div>
            
            <div v-else>
              <el-row :gutter="20">
                <el-col :xs="24" :sm="24" :md="12">
                  <div class="region-chart-wrapper">
                    <div v-if="regionStats.length === 0" style="text-align: center; padding: 60px; color: #909399;">
                      <el-icon style="font-size: 48px; margin-bottom: 10px;"><DataAnalysis /></el-icon>
                      <p>暂无地区分布数据</p>
                      <p style="font-size: 12px; margin-top: 5px;">请确保已启用 GeoIP 数据库并记录用户登录位置</p>
                    </div>
                    <div v-else class="region-chart-container">
                      <canvas ref="regionChart"></canvas>
                    </div>
                  </div>
                </el-col>
                <el-col :xs="24" :sm="24" :md="12">
                  <!-- 桌面端表格 -->
                  <div class="desktop-only">
                    <div class="region-stats-table">
                      <h4 style="margin: 0 0 15px 0; font-size: 16px; color: #303133;">地区统计列表</h4>
                      <el-table 
                        :data="regionStats" 
                        stripe 
                        style="width: 100%"
                        :empty-text="'暂无地区数据'"
                        max-height="400"
                      >
                        <el-table-column prop="region" label="地区" min-width="120">
                          <template #default="{ row }">
                            <el-tag type="info" size="small">{{ row.region || '未知' }}</el-tag>
                          </template>
                        </el-table-column>
                        <el-table-column prop="userCount" label="用户数" width="100" align="right">
                          <template #default="{ row }">
                            <el-tag type="primary" size="small">{{ row.userCount || 0 }}</el-tag>
                          </template>
                        </el-table-column>
                        <el-table-column prop="percentage" label="占比" width="100" align="right">
                          <template #default="{ row }">
                            <span style="color: #606266; font-weight: 500;">{{ row.percentage || '0.0' }}%</span>
                          </template>
                        </el-table-column>
                        <el-table-column prop="loginCount" label="登录次数" width="110" align="right">
                          <template #default="{ row }">
                            <span style="color: #909399;">{{ row.loginCount || 0 }}</span>
                          </template>
                        </el-table-column>
                      </el-table>
                    </div>
                  </div>
                  
                  <!-- 移动端卡片列表 -->
                  <div class="mobile-only">
                    <div class="region-stats-list">
                      <h4 style="margin: 0 0 15px 0; font-size: 16px; color: #303133; font-weight: 600;">地区统计列表</h4>
                      <div v-if="regionStats.length === 0" style="text-align: center; padding: 40px; color: #909399;">
                        <el-empty description="暂无地区数据" :image-size="80" />
                      </div>
                      <div v-else class="region-card-list">
                        <div 
                          v-for="(stat, index) in regionStats" 
                          :key="index"
                          class="region-card-item"
                        >
                          <div class="region-card-header">
                            <el-tag type="info" size="small" style="font-size: 14px; padding: 4px 12px;">
                              {{ stat.region || '未知' }}
                            </el-tag>
                            <el-tag type="primary" size="small" style="font-size: 14px; padding: 4px 12px;">
                              {{ stat.userCount || 0 }} 人
                            </el-tag>
                          </div>
                          <div class="region-card-body">
                            <div class="region-card-stat">
                              <span class="stat-label">占比：</span>
                              <span class="stat-value">{{ stat.percentage || '0.0' }}%</span>
                            </div>
                            <div class="region-card-stat">
                              <span class="stat-label">登录次数：</span>
                              <span class="stat-value">{{ stat.loginCount || 0 }}</span>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </el-col>
              </el-row>
            </div>
          </el-card>

          <el-card>
            <template #header>
              <div class="card-header">
                <h3 style="margin: 0;">地区详细统计</h3>
              </div>
            </template>
            
            <!-- 桌面端表格 -->
            <div class="desktop-only">
              <el-table 
                :data="regionDetails" 
                stripe
                :empty-text="'暂无详细统计数据'"
                v-loading="loadingRegions"
              >
                <el-table-column prop="country" label="国家" width="140" fixed="left">
                  <template #default="{ row }">
                    <el-tag type="success" size="small" v-if="row.country">{{ row.country }}</el-tag>
                    <span v-else style="color: #c0c4cc;">-</span>
                  </template>
                </el-table-column>
                <el-table-column prop="city" label="城市" width="140">
                  <template #default="{ row }">
                    <span v-if="row.city">{{ row.city }}</span>
                    <span v-else style="color: #c0c4cc;">-</span>
                  </template>
                </el-table-column>
                <el-table-column prop="userCount" label="用户数" width="110" align="right">
                  <template #default="{ row }">
                    <el-tag type="primary" size="small">{{ row.userCount || 0 }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="loginCount" label="登录次数" width="120" align="right">
                  <template #default="{ row }">
                    <span style="color: #606266;">{{ row.loginCount || 0 }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="lastLogin" label="最后登录" width="180">
                  <template #default="{ row }">
                    <span style="color: #909399; font-size: 13px;">{{ row.lastLogin || '-' }}</span>
                  </template>
                </el-table-column>
              </el-table>
            </div>
            
            <!-- 移动端卡片列表 -->
            <div class="mobile-only">
              <div v-loading="loadingRegions" style="min-height: 200px;">
                <div v-if="regionDetails.length === 0" style="text-align: center; padding: 40px;">
                  <el-empty description="暂无详细统计数据" :image-size="80" />
                </div>
                <div v-else class="region-details-list">
                  <div 
                    v-for="(detail, index) in regionDetails" 
                    :key="index"
                    class="region-detail-card"
                  >
                    <div class="detail-card-header">
                      <div class="detail-location">
                        <el-tag type="success" size="small" v-if="detail.country" style="margin-right: 8px;">
                          {{ detail.country }}
                        </el-tag>
                        <span v-if="detail.city" class="detail-city">{{ detail.city }}</span>
                        <span v-else style="color: #c0c4cc; font-size: 13px;">-</span>
                      </div>
                    </div>
                    <div class="detail-card-body">
                      <div class="detail-stat-row">
                        <span class="detail-label">用户数：</span>
                        <el-tag type="primary" size="small">{{ detail.userCount || 0 }}</el-tag>
                      </div>
                      <div class="detail-stat-row">
                        <span class="detail-label">登录次数：</span>
                        <span class="detail-value">{{ detail.loginCount || 0 }}</span>
                      </div>
                      <div class="detail-stat-row" v-if="detail.lastLogin">
                        <span class="detail-label">最后登录：</span>
                        <span class="detail-time">{{ detail.lastLogin }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </el-card>
        </el-tab-pane>
      </el-tabs>
    </div>
  </template>
  
  <script>
  import { ref, reactive, onMounted, nextTick, watch } from 'vue'
  import { Chart, registerables } from 'chart.js'
  import { Refresh, Loading, DataAnalysis } from '@element-plus/icons-vue'
  import { statisticsAPI } from '@/utils/api'
  
  Chart.register(...registerables)
  
  export default {
    name: 'AdminStatistics',
    components: {
      Refresh,
      Loading,
      DataAnalysis
    },
    setup() {
      const userChart = ref(null)
      const revenueChart = ref(null)
      const regionChart = ref(null)
      const activeTab = ref('users')
      const loadingRegions = ref(false)
      
      const statistics = reactive({
        totalUsers: 0,
        activeSubscriptions: 0,
        totalOrders: 0,
        totalRevenue: 0
      })
  
      const userStats = ref([])
      const subscriptionStats = ref([])
      const recentActivities = ref([])
      const regionStats = ref([])
      const regionDetails = ref([])
  
      // 获取统计数据
      const fetchStatistics = async () => {
        try {
          const response = await statisticsAPI.getStatistics()
          if (response.data && response.data.success && response.data.data) {
            const data = response.data.data
            // 更新概览数据（优先使用 overview，如果没有则使用直接字段）
            if (data.overview) {
              Object.assign(statistics, data.overview)
            } else {
              // 如果没有 overview，从直接字段映射
              statistics.totalUsers = Number(data.total_users || data.totalUsers) || 0
              statistics.activeSubscriptions = Number(data.active_subscriptions || data.activeSubscriptions) || 0
              statistics.totalOrders = Number(data.total_orders || data.totalOrders) || 0
              statistics.totalRevenue = Number(data.total_revenue || data.totalRevenue) || 0
            }
            
            // 更新用户统计
            if (data.userStats && Array.isArray(data.userStats)) {
              userStats.value = data.userStats.map(stat => ({
                label: stat.name || stat.label,
                value: Number(stat.value) || 0,
                percentage: Number(stat.percentage) || 0,
                color: '#409eff'
              }))
            } else {
              // 如果没有数据，设置为空数组
              userStats.value = []
            }
            
            // 更新订阅统计
            if (data.subscriptionStats && Array.isArray(data.subscriptionStats)) {
              subscriptionStats.value = data.subscriptionStats.map(stat => ({
                label: stat.name || stat.label,
                value: Number(stat.value) || 0,
                percentage: Number(stat.percentage) || 0,
                color: '#67c23a'
              }))
            } else {
              // 如果没有数据，设置为空数组
              subscriptionStats.value = []
            }
            
            // 更新最近活动
            if (data.recentActivities && Array.isArray(data.recentActivities)) {
              recentActivities.value = data.recentActivities.map(activity => ({
                id: activity.id,
                type: activity.type || 'primary',
                title: activity.description || activity.title || '未知活动',
                description: activity.amount !== undefined 
                  ? `金额: ¥${formatMoney(activity.amount)} | 状态: ${activity.status || '未知'}`
                  : (activity.description || ''),
                time: activity.time || activity.created_at || ''
              }))
            } else {
              // 如果没有数据，设置为空数组
              recentActivities.value = []
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
          if (!response.data || !response.data.success || !response.data.data) {
            console.error('获取用户趋势数据失败')
            return
          }
          
          const chartData = response.data.data
          const labels = chartData.labels || []
          const data = chartData.data || []
          
          const ctx = userChart.value.getContext('2d')
          
          new Chart(ctx, {
            type: 'line',
            data: {
              labels: labels,
              datasets: [{
                label: '新用户注册',
                data: data,
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
          if (!response.data || !response.data.success || !response.data.data) {
            console.error('获取收入趋势数据失败')
            return
          }
          
          const chartData = response.data.data
          const labels = chartData.labels || []
          const data = chartData.data || []
          
          const ctx = revenueChart.value.getContext('2d')
          
          new Chart(ctx, {
            type: 'bar',
            data: {
              labels: labels,
              datasets: [{
                label: '收入',
                data: data,
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

      // 加载地区统计
      const loadRegionStats = async () => {
        try {
          loadingRegions.value = true
          const response = await statisticsAPI.getRegionStats()
          if (response.data && response.data.success && response.data.data) {
            const data = response.data.data
            regionStats.value = data.regions || []
            regionDetails.value = data.details || []
            
            // 等待 DOM 更新后绘制图表
            await nextTick()
            if (regionChart.value && regionStats.value.length > 0) {
              // 延迟一下确保 canvas 已渲染
              setTimeout(() => {
                initRegionChart()
              }, 100)
            }
          } else {
            regionStats.value = []
            regionDetails.value = []
          }
        } catch (error) {
          console.error('加载地区统计失败:', error)
          regionStats.value = []
          regionDetails.value = []
        } finally {
          loadingRegions.value = false
        }
      }

      // 初始化地区分布图表
      let regionChartInstance = null
      const initRegionChart = () => {
        if (!regionChart.value || regionStats.value.length === 0) return
        
        try {
          // 如果已有图表实例，先销毁
          if (regionChartInstance) {
            regionChartInstance.destroy()
            regionChartInstance = null
          }
          
          const ctx = regionChart.value.getContext('2d')
          if (ctx) {
            // 检测是否为移动端
            const isMobile = window.innerWidth <= 768
            
            regionChartInstance = new Chart(ctx, {
              type: 'doughnut',
              data: {
                labels: regionStats.value.map(r => r.region || '未知'),
                datasets: [{
                  data: regionStats.value.map(r => r.userCount || 0),
                  backgroundColor: [
                    '#409EFF', '#67C23A', '#E6A23C', '#F56C6C', '#909399',
                    '#9C27B0', '#FF9800', '#00BCD4', '#4CAF50', '#FF5722',
                    '#795548', '#607D8B', '#E91E63', '#009688', '#FFC107'
                  ]
                }]
              },
              options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                  legend: {
                    position: isMobile ? 'bottom' : 'right',
                    labels: {
                      padding: isMobile ? 8 : 15,
                      usePointStyle: true,
                      font: {
                        size: isMobile ? 10 : 12
                      },
                      boxWidth: isMobile ? 10 : 12
                    }
                  },
                  tooltip: {
                    callbacks: {
                      label: function(context) {
                        const label = context.label || ''
                        const value = context.parsed || 0
                        const total = context.dataset.data.reduce((a, b) => a + b, 0)
                        const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0.0'
                        return `${label}: ${value} 人 (${percentage}%)`
                      }
                    }
                  },
                  title: {
                    display: !isMobile,
                    text: '用户地区分布',
                    font: {
                      size: isMobile ? 14 : 16,
                      weight: 'bold'
                    },
                    padding: {
                      top: 10,
                      bottom: 20
                    }
                  }
                }
              }
            })
          }
        } catch (error) {
          console.error('初始化地区图表失败:', error)
        }
      }

      // 监听标签页切换，加载地区数据
      watch(activeTab, (newTab) => {
        if (newTab === 'regions' && regionStats.value.length === 0 && !loadingRegions.value) {
          loadRegionStats()
        }
      })

      onMounted(() => {
        fetchStatistics()
        initUserChart()
        initRevenueChart()
        // 如果默认是地区分析标签页，则加载数据
        if (activeTab.value === 'regions') {
          loadRegionStats()
        }
      })

      return {
        userChart,
        revenueChart,
        regionChart,
        activeTab,
        loadingRegions,
        statistics,
        userStats,
        subscriptionStats,
        recentActivities,
        regionStats,
        regionDetails,
        loadRegionStats,
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
  
  .stat-icon :is(i) {
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
  
  .region-chart-wrapper {
    min-height: 300px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  
  .region-chart-container {
    width: 100%;
    height: 350px;
    position: relative;
    padding: 20px;
  }
  
  @media (max-width: 768px) {
    .region-chart-container {
      height: 300px;
      padding: 10px;
    }
  }
  
  .region-stats-table {
    padding: 10px 0;
  }
  
  .region-stats-table h4 {
    font-weight: 600;
    color: #303133;
    padding-bottom: 10px;
    border-bottom: 1px solid #ebeef5;
    margin-bottom: 15px;
  }
  
  /* 移动端地区统计卡片 */
  .region-stats-list {
    padding: 10px 0;
  }
  
  .region-card-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  
  .region-card-item {
    padding: 14px;
    background: #f8f9fa;
    border-radius: 8px;
    border: 1px solid #e9ecef;
    transition: all 0.3s ease;
  }
  
  .region-card-item:hover {
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    transform: translateY(-2px);
  }
  
  .region-card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    flex-wrap: wrap;
    gap: 8px;
  }
  
  .region-card-body {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  
  .region-card-stat {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 14px;
  }
  
  .region-card-stat .stat-label {
    color: #606266;
    font-weight: 500;
  }
  
  .region-card-stat .stat-value {
    color: #303133;
    font-weight: 600;
  }
  
  /* 移动端地区详细统计卡片 */
  .region-details-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  
  .region-detail-card {
    padding: 14px;
    background: #ffffff;
    border-radius: 8px;
    border: 1px solid #e9ecef;
    transition: all 0.3s ease;
  }
  
  .region-detail-card:hover {
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    transform: translateY(-2px);
  }
  
  .detail-card-header {
    margin-bottom: 12px;
    padding-bottom: 10px;
    border-bottom: 1px solid #f0f0f0;
  }
  
  .detail-location {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 6px;
  }
  
  .detail-city {
    color: #303133;
    font-size: 14px;
    font-weight: 500;
  }
  
  .detail-card-body {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  
  .detail-stat-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 14px;
  }
  
  .detail-label {
    color: #606266;
    font-weight: 500;
  }
  
  .detail-value {
    color: #303133;
    font-weight: 600;
  }
  
  .detail-time {
    color: #909399;
    font-size: 13px;
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
    
    .region-chart-wrapper {
      min-height: 250px;
      margin-bottom: 16px;
    }
    
    .region-chart-container {
      height: 280px;
      padding: 10px;
    }
    
    .card-header {
      padding: 12px 0;
      
      :is(h3) {
        font-size: 16px;
        font-weight: 600;
        margin: 0;
      }
    }
    
    .region-stats-list h4 {
      font-size: 15px;
      margin-bottom: 12px;
      padding-bottom: 8px;
      border-bottom: 1px solid #ebeef5;
    }
    
    .region-card-item {
      padding: 12px;
    }
    
    .region-card-header {
      margin-bottom: 10px;
    }
    
    .region-card-stat {
      font-size: 13px;
    }
    
    .region-detail-card {
      padding: 12px;
    }
    
    .detail-card-header {
      margin-bottom: 10px;
      padding-bottom: 8px;
    }
    
    .detail-stat-row {
      font-size: 13px;
    }
    
    .detail-time {
      font-size: 12px;
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