<template>
  <div class="list-container admin-statistics">
      <el-row :gutter="20" class="stats-cards">
        <el-col :xs="24" :sm="12" :md="6">
          <el-card class="stat-card">
            <div class="stat-content">
              <div class="stat-icon users">
                <el-icon><User /></el-icon>
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
                <el-icon><Connection /></el-icon>
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
                <el-icon><ShoppingCart /></el-icon>
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
                <el-icon><Money /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-number">¥{{ formatMoney(statistics.totalRevenue) }}</div>
                <div class="stat-label">总收入</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
      <el-tabs v-model="activeTab" type="border-card" class="statistics-tabs">
        <el-tab-pane label="用户统计" name="users">
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
      <el-row :gutter="20" class="detailed-stats">
        <el-col :xs="24" :sm="24" :md="12">
          <el-card>
            <template #header>
              <div class="card-header">
                <h3>用户统计</h3>
              </div>
            </template>
            <div class="desktop-only">
              <el-table :data="userStats" class="full-width-table" v-if="userStats.length > 0">
                <el-table-column prop="label" label="统计项" />
                <el-table-column prop="value" label="数值" />
                <el-table-column prop="percentage" label="占比">
                  <template #default="{ row }">
                    <el-progress
                      :percentage="Math.round(row.percentage * 10) / 10"
                      :color="row.color"
                      :show-text="false"
                    />
                    <span class="percentage-text">{{ Math.round(row.percentage * 10) / 10 }}%</span>
                  </template>
                </el-table-column>
              </el-table>
              <EmptyState
                v-else
                title="暂无用户统计"
                description="当前筛选范围内还没有可展示的用户统计数据。"
                :icon-size="56"
                class="compact-empty-state"
              />
            </div>
            <div v-if="userStats.length > 0" class="mobile-stats-list mobile-only">
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
                    :percentage="Math.round(stat.percentage * 10) / 10"
                    :color="stat.color"
                    :show-text="false"
                  />
                  <span class="stat-item-percentage">{{ Math.round(stat.percentage * 10) / 10 }}%</span>
                </div>
              </div>
            </div>
            <EmptyState
              v-else
              title="暂无用户统计"
              description="当前筛选范围内还没有可展示的用户统计数据。"
              :icon-size="48"
              class="compact-empty-state mobile-empty-state mobile-only"
            />
          </el-card>
        </el-col>
        <el-col :xs="24" :sm="24" :md="12">
          <el-card>
            <template #header>
              <div class="card-header">
                <h3>订阅统计</h3>
              </div>
            </template>
            <div class="desktop-only">
              <el-table :data="subscriptionStats" class="full-width-table" v-if="subscriptionStats.length > 0">
                <el-table-column prop="label" label="统计项" />
                <el-table-column prop="value" label="数值" />
                <el-table-column prop="percentage" label="占比">
                  <template #default="{ row }">
                    <el-progress
                      :percentage="Math.round(row.percentage * 10) / 10"
                      :color="row.color"
                      :show-text="false"
                    />
                    <span class="percentage-text">{{ Math.round(row.percentage * 10) / 10 }}%</span>
                  </template>
                </el-table-column>
              </el-table>
              <EmptyState
                v-else
                title="暂无订阅统计"
                description="当前筛选范围内还没有可展示的订阅统计数据。"
                :icon-size="56"
                class="compact-empty-state"
              />
            </div>
            <div v-if="subscriptionStats.length > 0" class="mobile-stats-list mobile-only">
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
                    :percentage="Math.round(stat.percentage * 10) / 10"
                    :color="stat.color"
                    :show-text="false"
                  />
                  <span class="stat-item-percentage">{{ Math.round(stat.percentage * 10) / 10 }}%</span>
                </div>
              </div>
            </div>
            <EmptyState
              v-else
              title="暂无订阅统计"
              description="当前筛选范围内还没有可展示的订阅统计数据。"
              :icon-size="48"
              class="compact-empty-state mobile-empty-state mobile-only"
            />
          </el-card>
        </el-col>
      </el-row>
        </el-tab-pane>
        <el-tab-pane label="地区分析" name="regions">
          <el-card class="region-analysis-card">
            <template #header>
              <div class="card-header region-card-header-bar">
                <h3>用户地区分布</h3>
                <el-button type="primary" size="small" @click="loadRegionStats" :loading="loadingRegions">
                  <el-icon class="button-icon"><Refresh /></el-icon>
                  刷新数据
                </el-button>
              </div>
            </template>
            <LoadingState
              v-if="loadingRegions"
              text="正在加载地区数据..."
              class="region-loading-state"
            />
            <div v-else>
              <el-row :gutter="20">
						<el-col :xs="24" :sm="24" :md="12">
							<div class="region-chart-wrapper">
								<h4 class="region-chart-title" v-if="regionStats.length > 0">用户地区分布图</h4>
									<EmptyState
										v-if="regionStats.length === 0"
										title="暂无地区分布数据"
										description="请确保已启用 GeoIP 数据库并记录用户登录位置。"
										:icon-size="48"
										class="region-empty-state"
									/>
								<div v-else class="region-chart-container">
									<canvas ref="regionChart" class="chart-canvas"></canvas>
								</div>
							</div>
						</el-col>
						<el-col :xs="24" :sm="24" :md="12">
							<div class="desktop-only">
								<div class="region-stats-table">
									<h4 class="region-list-title">地区统计列表</h4>
									<el-table :data="regionStats" stripe class="full-width-table" :empty-text="'暂无地区数据'" max-height="400" :default-sort="{ prop: 'userCount', order: 'descending' }">
										<el-table-column prop="country" label="国家" min-width="100" sortable>
											<template #default="{ row }">
												<el-tag type="success" size="small">{{ row.country || '-' }}</el-tag>
											</template>
										</el-table-column>
										<el-table-column prop="city" label="城市" min-width="100" sortable>
											<template #default="{ row }">
												<span>{{ row.city || '-' }}</span>
											</template>
										</el-table-column>
										<el-table-column prop="userCount" label="用户数" width="100" align="right" sortable>
											<template #default="{ row }">
												<el-tag type="primary" size="small">{{ row.userCount || 0 }}</el-tag>
											</template>
										</el-table-column>
										<el-table-column prop="percentage" label="占比" width="100" align="right" sortable :sort-method="(a, b) => parseFloat(a.percentage || 0) - parseFloat(b.percentage || 0)">
											<template #default="{ row }">
												<span class="regular-text">{{ row.percentage || '0.0' }}%</span>
											</template>
										</el-table-column>
										<el-table-column prop="loginCount" label="登录次数" width="120" align="right" sortable>
											<template #default="{ row }">
												<span class="secondary-text">{{ row.loginCount || 0 }}</span>
											</template>
										</el-table-column>
										<el-table-column prop="lastLogin" label="最后登录" width="180" align="right" sortable>
											<template #default="{ row }">
												<span class="table-time-text">{{ row.lastLogin }}</span>
											</template>
										</el-table-column>
									</el-table>
								</div>
							</div>
							<div class="mobile-only">
								<div class="region-stats-list">
									<h4 class="region-list-title">地区统计列表</h4>
									<div v-if="regionStats.length === 0" class="region-list-empty">
									<EmptyState
										title="暂无地区数据"
										description="启用 GeoIP 数据库并记录用户登录位置后，这里会显示地区统计。"
										:icon-size="48"
										class="compact-empty-state mobile-empty-state"
									/>
									</div>
									<div v-else class="region-card-list">
										<div v-for="(stat, index) in regionStats" :key="index" class="region-card-item">
											<div class="region-card-header">
												<el-tag type="info" size="small" class="region-card-tag">
													{{ stat.region || '未知' }}
												</el-tag>
												<el-tag type="primary" size="small" class="region-card-tag">
													{{ stat.userCount || 0 }} 人
												</el-tag>
											</div>
											<div class="region-card-body">
												<div class="region-card-stat">
													<span class="stat-label">占比：</span>
													<span class="stat-value">{{ stat.percentage || '0.0' }}%</span>
												</div>
												<div class="region-card-stat">
													<span class="stat-label">登录：</span>
													<span class="stat-value">{{ stat.loginCount || 0 }}</span>
												</div>
												<div class="region-card-stat">
													<span class="stat-label">最后登录：</span>
													<span class="stat-value stat-time-value">{{ stat.lastLogin }}</span>
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
		</el-tab-pane>
	</el-tabs>
</div>
</template>
<script>
import { ref, reactive, onMounted, nextTick, watch } from 'vue'
import { Chart, registerables } from 'chart.js'
import { Refresh, User, Connection, ShoppingCart, Money } from '@element-plus/icons-vue'
import { statisticsAPI } from '@/utils/api'
import EmptyState from '@/components/EmptyState.vue'
import LoadingState from '@/components/LoadingState.vue'
Chart.register(...registerables)
export default {
	name: 'AdminStatistics',
	components: {
			Refresh,
			User,
			Connection,
			ShoppingCart,
			Money,
			EmptyState,
			LoadingState
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
		const regionStats = ref([])
		const fetchStatistics = async () => {
			try {
				const response = await statisticsAPI.getStatistics()
				if (response.data && response.data.success && response.data.data) {
					const data = response.data.data
					if (data.overview) {
						Object.assign(statistics, data.overview)
					} else {
						statistics.totalUsers = Number(data.total_users || data.totalUsers) || 0
						statistics.activeSubscriptions = Number(data.active_subscriptions || data.activeSubscriptions) || 0
						statistics.totalOrders = Number(data.total_orders || data.totalOrders) || 0
						statistics.totalRevenue = Number(data.total_revenue || data.totalRevenue) || 0
					}
					if (data.userStats && Array.isArray(data.userStats)) {
						userStats.value = data.userStats.map(stat => ({
							label: stat.name || stat.label,
							value: Number(stat.value) || 0,
							percentage: Number(stat.percentage) || 0,
							color: '#409eff'
						}))
					} else {
						userStats.value = []
					}
					if (data.subscriptionStats && Array.isArray(data.subscriptionStats)) {
						subscriptionStats.value = data.subscriptionStats.map(stat => ({
							label: stat.name || stat.label,
							value: Number(stat.value) || 0,
							percentage: Number(stat.percentage) || 0,
							color: '#67c23a'
						}))
					} else {
						subscriptionStats.value = []
					}
				}
			} catch (error) {
				console.error(error)
			}
		}
		const initUserChart = async () => {
			try {
				const response = await statisticsAPI.getUserTrend()
				if (!response.data || !response.data.success || !response.data.data) {
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
				console.error(error)
			}
		}
		const initRevenueChart = async () => {
			try {
				const response = await statisticsAPI.getRevenueTrend()
				if (!response.data || !response.data.success || !response.data.data) {
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
				console.error(error)
			}
		}
		const formatMoney = (value) => {
			if (value === null || value === undefined || value === '') return '0.00'
			const num = typeof value === 'string' ? parseFloat(value) : value
			if (isNaN(num)) return '0.00'
			return num.toFixed(2)
		}
		const loadRegionStats = async () => {
			try {
				loadingRegions.value = true
				const response = await statisticsAPI.getRegionStats()
				if (response.data && response.data.success && response.data.data) {
					const data = response.data.data
					regionStats.value = data.regions || []
					await nextTick()
					let attempts = 0
					const tryInit = () => {
						if (regionChart.value && regionStats.value.length > 0) {
							initRegionChart()
						} else if (attempts < 10) {
							attempts++
							setTimeout(tryInit, 200)
						}
					}
					tryInit()
				} else {
					regionStats.value = []
				}
			} catch (error) {
				regionStats.value = []
			} finally {
				loadingRegions.value = false
			}
		}
		let regionChartInstance = null
		const initRegionChart = () => {
			if (!regionChart.value || regionStats.value.length === 0) return
			try {
				if (regionChartInstance) {
					regionChartInstance.destroy()
					regionChartInstance = null
				}
				const ctx = regionChart.value.getContext('2d')
				if (ctx) {
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
									position: 'bottom',
									labels: {
										padding: isMobile ? 8 : 12,
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
				console.error(error)
			}
		}
		watch(activeTab, (newTab) => {
			if (newTab === 'regions' && regionStats.value.length === 0 && !loadingRegions.value) {
				loadRegionStats()
			}
		})
		onMounted(() => {
			fetchStatistics()
			initUserChart()
			initRevenueChart()
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
			regionStats,
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
  .statistics-tabs {
    margin-top: 20px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    box-shadow: none;
    overflow: hidden;
  }
  .statistics-tabs :deep(.el-tabs__header) {
    background: var(--el-fill-color-extra-light);
  }
  .statistics-tabs :deep(.el-tabs__content) {
    padding: 16px;
  }
  .charts-section :deep(.el-card),
  .detailed-stats :deep(.el-card),
  .region-analysis-card {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    box-shadow: none;
  }
  .charts-section :deep(.el-card__header),
  .detailed-stats :deep(.el-card__header),
  .region-analysis-card :deep(.el-card__header) {
    padding: 14px 16px;
    background: var(--el-fill-color-extra-light);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  .full-width-table {
    width: 100%;
  }
  .percentage-text {
    margin-left: 10px;
  }
  .regular-text {
    color: #606266;
  }
  .secondary-text {
    color: #909399;
  }
  .table-time-text {
    color: #909399;
    font-size: 12px;
  }
  .stat-card {
    height: 120px;
    border: 1px solid #ebeef5;
    border-radius: 8px;
    box-shadow: none;
    transition: border-color 0.2s ease, background-color 0.2s ease;
  }
  .stat-card:hover {
    border-color: var(--el-border-color);
    background: #fbfdff;
  }
  .stat-card :deep(.el-card__body) {
    height: 100%;
  }
  .stat-content {
    display: flex;
    align-items: center;
    height: 100%;
    min-width: 0;
  }
  .stat-icon {
    width: 60px;
    height: 60px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: 20px;
    flex: 0 0 60px;
    color: var(--stat-icon-color, #409eff);
    background: var(--stat-icon-bg, #ecf5ff);
  }
  .stat-icon :deep(.el-icon) {
    font-size: 24px;
  }
  .stat-icon.users {
    --stat-icon-color: #2563eb;
    --stat-icon-bg: #eff6ff;
  }
  .stat-icon.subscriptions {
    --stat-icon-color: #7c3aed;
    --stat-icon-bg: #f5f3ff;
  }
  .stat-icon.orders {
    --stat-icon-color: #0891b2;
    --stat-icon-bg: #ecfeff;
  }
  .stat-icon.revenue {
    --stat-icon-color: #16a34a;
    --stat-icon-bg: #f0fdf4;
  }
  .stat-info {
    flex: 1;
    min-width: 0;
  }
  .stat-number {
    font-size: 28px;
    font-weight: 700;
    color: #303133;
    margin-bottom: 5px;
    line-height: 1.2;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .stat-label {
    color: #909399;
    font-size: 13px;
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
  .region-analysis-card {
    margin-bottom: 20px;
  }
  .region-card-header-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .region-card-header-bar :deep(.el-button) {
    min-height: 44px;
    touch-action: manipulation;
  }
  .button-icon {
    margin-right: 4px;
  }
  .region-list-empty {
    color: #909399;
    text-align: center;
  }
  .region-loading-state {
    padding: 40px;
  }
  .region-chart-title {
    margin: 0 0 15px;
    color: #303133;
    font-size: 16px;
    text-align: center;
  }
  .region-empty-state {
    padding: 60px;
  }
  .region-chart-container {
    width: 100%;
    height: 500px;
    position: relative;
    padding: 20px;
  }
  .chart-canvas {
    width: 100%;
    height: 100%;
  }
  @media (max-width: 768px) {
    .region-chart-container {
      height: 450px;
      padding: 10px;
    }
  }
  .region-stats-table {
    padding: 10px 0;
  }
  .region-list-title {
    margin: 0 0 15px;
    color: #303133;
    font-size: 16px;
    font-weight: 600;
  }
  .region-stats-table h4 {
    font-weight: 600;
    color: #303133;
    padding-bottom: 10px;
    border-bottom: 1px solid #ebeef5;
    margin-bottom: 15px;
  }
  .region-stats-list {
    padding: 10px 0;
  }
  .region-list-empty {
    padding: 40px;
  }
  .compact-empty-state {
    min-height: 180px;
    padding: 32px 16px;
  }
  .mobile-empty-state {
    min-height: 160px;
    padding: 28px 14px;
  }
  .region-card-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .region-card-item {
    padding: 14px;
    background: #fff;
    border-radius: 8px;
    border: 1px solid #ebeef5;
    transition: border-color 0.2s ease, background-color 0.2s ease;
  }
  .region-card-item:hover {
    background: #fafcff;
    border-color: #c6e2ff;
  }
  .region-card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    flex-wrap: wrap;
    gap: 8px;
  }
  .region-card-tag {
    padding: 4px 12px;
    font-size: 14px;
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
  .region-card-stat .stat-time-value {
    font-size: 12px;
  }
  .region-details-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .region-detail-card {
    padding: 14px;
    background: #ffffff;
    border-radius: 8px;
    border: 1px solid #ebeef5;
    transition: border-color 0.2s ease, background-color 0.2s ease;
  }
  .region-detail-card:hover {
    background: #fafcff;
    border-color: #c6e2ff;
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
    .statistics-tabs {
      margin-top: 12px;
    }
    .statistics-tabs :deep(.el-tabs__content) {
      padding: 12px;
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
      flex-direction: column;
      align-items: stretch;
    }
    .region-card-header-bar {
      align-items: stretch;
      flex-direction: column;
    }
    .region-card-header-bar :deep(.el-button) {
      width: 100%;
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
    :deep(.el-timeline-item) {
      padding-bottom: 16px;
    }
    :deep(.el-timeline-item__timestamp) {
      font-size: 12px;
      color: #909399;
      margin-bottom: 8px;
    }
  }
</style>
