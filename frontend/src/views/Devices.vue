<template>
  <div class="list-container devices-container">
    <div class="breadcrumb">首页 / 设备管理</div>
    <div class="page-header">
      <div class="page-title">
        <h1>设备管理</h1>
      </div>
      <div class="actions">
        <el-button type="success" @click="openUpgradeDrawer" :loading="upgradeSubscriptionLoading">
          升级设备数量
        </el-button>
        <el-button @click="refreshDevices" :loading="loading">
          刷新
        </el-button>
      </div>
    </div>
    <div class="stats-row devices-stats-row grid cols-4">
      <div class="stat-card">
        <div class="stat-icon">D</div>
        <div>
          <div class="stat-value">{{ deviceStats.total }}</div>
          <div class="stat-label">当前设备</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">O</div>
        <div>
          <div class="stat-value">{{ deviceStats.online }}</div>
          <div class="stat-label">在线设备</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">+</div>
        <div>
          <div class="stat-value">可升级</div>
          <div class="stat-label">设备数量</div>
        </div>
      </div>
      <div class="stat-card device-type-stat-card">
        <div class="stat-icon">T</div>
        <div class="device-type-stat-content">
          <div class="device-type-stat-title">设备类型统计</div>
          <div class="device-type-stat-list">
            <span>移动 {{ deviceStats.mobile }}</span>
            <span>桌面 {{ deviceStats.desktop }}</span>
            <span>其他 {{ otherDeviceCount }}</span>
          </div>
        </div>
      </div>
    </div>
    <div class="card list-filter-card devices-filter-card">
      <div class="card-body devices-filter-body">
        <el-form :inline="true" :model="filters" class="devices-filter-form list-filter-form">
          <el-form-item label="关键词">
            <el-input
              v-model="filters.keyword"
              placeholder="设备名、系统、IP、备注"
              clearable
              class="devices-keyword-input"
              @keyup.enter="applyDeviceFilters"
              @clear="applyDeviceFilters"
            />
          </el-form-item>
          <el-form-item label="设备类型">
            <el-select v-model="filters.device_type" placeholder="全部类型" clearable class="devices-filter-select" @change="applyDeviceFilters">
              <el-option label="手机" value="mobile" />
              <el-option label="电脑" value="desktop" />
              <el-option label="平板" value="tablet" />
              <el-option label="路由器" value="router" />
              <el-option label="电视盒子" value="tv_box" />
              <el-option label="服务器" value="server" />
              <el-option label="未知" value="unknown" />
            </el-select>
          </el-form-item>
          <el-form-item label="在线状态">
            <el-select v-model="filters.online_status" placeholder="全部状态" clearable class="devices-filter-select" @change="applyDeviceFilters">
              <el-option label="在线" value="online" />
              <el-option label="离线" value="offline" />
            </el-select>
          </el-form-item>
          <el-form-item class="devices-filter-actions">
            <el-button type="primary" @click="applyDeviceFilters">筛选</el-button>
            <el-button :disabled="!hasActiveFilters" @click="resetDeviceFilters">重置</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>
    <el-card class="list-card devices-card">
      <template #header>
        <div class="card-header">
          <span>
            <el-icon class="card-header-icon"><Monitor /></el-icon>
            设备列表
          </span>
          <el-button 
            size="small" 
            @click="refreshDevices"
            :loading="loading"
          >
            刷新
          </el-button>
        </div>
      </template>
      <ResponsiveDataView
        :data="displayedDevices"
        :fields="mobileDeviceFields"
        :loading="loading"
        title-field="device_name"
        empty-title="暂无设备记录"
        empty-description="刷新后可查看最近连接过的设备"
      >
        <template #table>
          <div class="table-wrapper">
            <el-table
              ref="deviceTableRef"
              :data="displayedDevices"
              v-loading="loading"
              class="devices-table"
              stripe
              border
              @header-dragend="handleDeviceColumnResize"
            >
              <template #empty>
                <EmptyState
                  title="暂无设备记录"
                  description="刷新后可查看最近连接过的设备"
                  action-text="刷新设备列表"
                  :loading="loading"
                  @action="refreshDevices"
                />
              </template>
              <el-table-column prop="device_name" label="设备名称" :min-width="columnWidths.device_name" resizable>
                <template #default="{ row }">
                  <div class="device-name">
                    <el-icon class="device-type-icon">
                      <component :is="getDeviceIcon(row.device_type)" />
                    </el-icon>
                    <div class="device-name-details">
                      <div class="device-main-name">
                        <span class="device-name-text">{{ row.device_name || '未知设备' }}</span>
                        <el-tag v-if="row.software_name" type="info" size="small" class="software-tag">
                          {{ row.software_name }}{{ row.software_version ? ' ' + row.software_version : '' }}
                        </el-tag>
                      </div>
                      <div v-if="row.device_model" class="device-model-info">
                        <el-tag type="success" size="small" class="stacked-tag">
                          {{ row.device_model }}{{ row.device_brand && row.device_brand !== 'Apple' ? ' (' + row.device_brand + ')' : '' }}
                        </el-tag>
                      </div>
                    </div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="device_type" label="设备类型" :width="columnWidths.device_type" resizable>
                <template #default="{ row }">
                  <el-tag :type="getDeviceTypeColor(row.device_type)">
                    {{ getDeviceTypeName(row.device_type) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="os_name" label="操作系统" :width="columnWidths.os_name" resizable>
                <template #default="{ row }">
                  <div class="os-info">
                    <div class="os-name">{{ row.os_name || '-' }}</div>
                    <div v-if="row.os_version" class="os-version">
                      <el-tag type="primary" size="small" class="stacked-tag">
                        {{ row.os_version }}
                      </el-tag>
                    </div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="ip_address" label="IP地址" :width="columnWidths.ip_address" resizable>
                <template #default="{ row }">
                  <div class="ip-location-cell">
                    <span class="ip-address">{{ row.ip_address || '-' }}</span>
                    <el-tag v-if="row.location" type="info" size="small" class="location-tag">
                      <el-icon><Location /></el-icon>
                      {{ formatLocation(row.location) }}
                    </el-tag>
                    <span v-else class="no-location-text">位置信息不可用</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="last_access" label="最后访问" :width="columnWidths.last_access" resizable>
                <template #default="{ row }">
                  <span>{{ formatTime(row.last_access) }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="user_agent" label="User Agent" :min-width="columnWidths.user_agent" :width="columnWidths.user_agent" resizable>
                <template #default="{ row }">
                  <el-tooltip :content="row.user_agent" placement="top">
                    <span class="user-agent">{{ truncateUserAgent(row.user_agent) }}</span>
                  </el-tooltip>
                </template>
              </el-table-column>
              <el-table-column prop="remark" label="备注" :width="columnWidths.remark" resizable>
                <template #default="{ row }">
                  <InlineEditableText
                    :value="row.remark || ''"
                    empty-text="点击添加备注"
                    placeholder="输入设备备注，回车或失焦保存"
                    :maxlength="200"
                    :loading="!!row.savingRemark"
                    @save="value => saveRemark(row, value)"
                  />
                </template>
              </el-table-column>
              <el-table-column label="操作" :width="columnWidths.actions" fixed="right" resizable>
                <template #default="{ row }">
                  <div class="action-buttons">
                    <el-button
                      type="danger"
                      size="small"
                      @click="removeDevice(row.id)"
                      :loading="row.removing"
                    >
                      移除
                    </el-button>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </template>
        <template #header="{ item }">
          <div class="mobile-device-header">
              <div class="device-name-details">
                <div class="device-main-name">
                <el-icon class="device-type-icon">
                  <component :is="getDeviceIcon(item.device_type)" />
                </el-icon>
                <span class="device-name-text">{{ item.device_name || '未知设备' }}</span>
                <el-tag v-if="item.software_name" type="info" size="small" class="software-tag">
                  {{ item.software_name }}{{ item.software_version ? ' ' + item.software_version : '' }}
                </el-tag>
              </div>
              <div v-if="item.device_model" class="device-model-info">
                <el-tag type="success" size="small" class="stacked-tag">
                  {{ item.device_model }}{{ item.device_brand && item.device_brand !== 'Apple' ? ' (' + item.device_brand + ')' : '' }}
                </el-tag>
              </div>
            </div>
          </div>
        </template>
        <template #field-ip_address="{ item }">
          <div class="ip-location-cell mobile-ip-cell">
            <span class="ip-address">{{ item.ip_address || '-' }}</span>
            <el-tag v-if="item.location" type="info" size="small">
              <el-icon><Location /></el-icon>
              {{ formatLocation(item.location) }}
            </el-tag>
            <span v-else class="no-location">位置信息不可用</span>
          </div>
        </template>
        <template #field-remark="{ item }">
          <InlineEditableText
            :value="item.remark || ''"
            empty-text="点击添加备注"
            placeholder="输入设备备注，回车或失焦保存"
            :maxlength="200"
            :loading="!!item.savingRemark"
            @save="value => saveRemark(item, value)"
          />
        </template>
        <template #empty>
          <EmptyState
            title="暂无设备记录"
            description="刷新后可查看最近连接过的设备"
            action-text="刷新设备列表"
            :loading="loading"
            @action="refreshDevices"
          />
        </template>
        <template #actions="{ item }">
          <el-button 
            type="danger" 
            size="small" 
            @click="removeDevice(item.id)"
            :loading="item.removing"
          >
            移除
          </el-button>
        </template>
      </ResponsiveDataView>
      <PaginationBar
        v-if="displayTotal > pageSize"
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :total="displayTotal"
        :page-sizes="[10, 20, 50, 100]"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </el-card>
    <UpgradeDevicesDrawer
      v-model="showUpgradeDrawer"
      :subscription="upgradeSubscription"
      :on-success="handleUpgradeSuccess"
      @success="refreshDevices"
    />
  </div>
</template>
<script>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import {
  Box,
  Cellphone,
  Connection,
  Iphone,
  Location,
  Monitor,
  QuestionFilled,
  VideoCamera
} from '@element-plus/icons-vue'
import { subscriptionAPI } from '@/utils/api'
import { formatDateTime as formatTimeUtil } from '@/utils/date'
import { formatLocation } from '@/utils/date'
import { confirmWarning } from '@/utils/confirmAction'
import { usePersistentTableColumns } from '@/composables/usePersistentTableColumns'
import EmptyState from '@/components/EmptyState.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import UpgradeDevicesDrawer from '@/components/UpgradeDevicesDrawer.vue'
import InlineEditableText from '@/components/InlineEditableText.vue'
import dayjs from 'dayjs'
import timezone from 'dayjs/plugin/timezone'
dayjs.extend(timezone)
export default {
  name: 'Devices',
  components: {
    EmptyState,
    ResponsiveDataView,
    Box,
    Cellphone,
    Connection,
    Iphone,
    Location,
    Monitor,
    QuestionFilled,
    VideoCamera,
    PaginationBar,
    UpgradeDevicesDrawer,
    InlineEditableText
  },
  setup() {
    const loading = ref(false)
    const devices = ref([])
    const deviceTableRef = ref(null)
    const currentPage = ref(1)
    const pageSize = ref(10)
    const total = ref(0)
    const filters = reactive({
      keyword: '',
      device_type: '',
      online_status: ''
    })
    const showUpgradeDrawer = ref(false)
    const upgradeSubscription = ref(null)
    const upgradeSubscriptionLoading = ref(false)
    const DEVICES_TABLE_STORAGE_KEY = 'user_devices_table_settings'
    const defaultColumnWidths = {
      device_name: 220,
      device_type: 120,
      os_name: 180,
      ip_address: 280,
      last_access: 180,
      user_agent: 200,
      remark: 160,
      actions: 120
    }
    const DEVICE_COLUMN_KEYS = ['device_name', 'device_type', 'os_name', 'ip_address', 'last_access', 'user_agent', 'remark', 'actions']
    const { columnWidths, handleColumnResize: handleDeviceColumnResize } = usePersistentTableColumns(
      DEVICES_TABLE_STORAGE_KEY,
      defaultColumnWidths,
      DEVICE_COLUMN_KEYS
    )
    const deviceStats = reactive({
      total: 0,
      online: 0,
      mobile: 0,
      desktop: 0
    })
    const otherDeviceCount = computed(() => Math.max(
      0,
      deviceStats.total - deviceStats.mobile - deviceStats.desktop
    ))
    const hasActiveFilters = computed(() => Boolean(filters.keyword || filters.device_type || filters.online_status))
    const filteredDevices = computed(() => {
      const keyword = String(filters.keyword || '').trim().toLowerCase()
      return devices.value.filter(device => {
        if (filters.device_type && device.device_type !== filters.device_type) {
          return false
        }
        if (filters.online_status) {
          const online = isOnline(device.last_access)
          if (filters.online_status === 'online' && !online) return false
          if (filters.online_status === 'offline' && online) return false
        }
        if (keyword) {
          const haystack = [
            device.device_name,
            device.device_model,
            device.device_brand,
            device.software_name,
            device.software_version,
            device.os_name,
            device.os_version,
            device.ip_address,
            device.location,
            device.user_agent,
            device.remark
          ].filter(Boolean).join(' ').toLowerCase()
          return haystack.includes(keyword)
        }
        return true
      })
    })
    const paginatedFilteredDevices = computed(() => {
      const start = (currentPage.value - 1) * pageSize.value
      const end = start + pageSize.value
      return filteredDevices.value.slice(start, end)
    })
    const displayedDevices = computed(() => hasActiveFilters.value ? paginatedFilteredDevices.value : devices.value)
    const displayTotal = computed(() => hasActiveFilters.value ? filteredDevices.value.length : total.value)
    const mobileDeviceFields = computed(() => [
      {
        key: 'device_type',
        label: '设备类型',
        type: 'tag',
        tagType: value => getDeviceTypeColor(value),
        formatter: value => value && value !== 'unknown' ? getDeviceTypeName(value) : '-'
      },
      { key: 'os_name', label: '操作系统', formatter: (_value, row) => [row.os_name, row.os_version].filter(Boolean).join(' ') || '-' },
      { key: 'ip_address', label: 'IP地址', fullWidth: true },
      { key: 'last_access', label: '最后访问', formatter: value => formatTime(value) },
      { key: 'user_agent', label: 'User Agent', formatter: value => truncateUserAgent(value), fullWidth: true },
      { key: 'remark', label: '备注', fullWidth: true }
    ])
    const fetchDevices = async () => {
      loading.value = true
      try {
        const response = await subscriptionAPI.getDevices({
          page: currentPage.value,
          size: pageSize.value
        })
        if (response && response.data) {
          const responseData = response.data
          if (responseData.success === false) {
            const errorMsg = responseData.message || '获取设备列表失败'
            ElMessage.error(errorMsg)
            devices.value = []
            total.value = 0
          } else if (responseData.data) {
            if (responseData.data.devices && Array.isArray(responseData.data.devices)) {
              devices.value = responseData.data.devices
              total.value = responseData.data.total || 0
              updateDeviceStats(responseData.data)
            } else if (Array.isArray(responseData.data)) {
              devices.value = responseData.data
              total.value = 0
              updateDeviceStats()
            } else {
              devices.value = []
              total.value = 0
              updateDeviceStats()
            }
          } else if (Array.isArray(responseData)) {
            devices.value = responseData
            total.value = 0
            updateDeviceStats()
          } else {
            devices.value = []
            total.value = 0
            updateDeviceStats()
          }
        } else {
          devices.value = []
          total.value = 0
          updateDeviceStats()
        }
      } catch (error) {
        console.error('获取设备列表错误:', error)
        const errorMsg = error.response?.data?.message || error.response?.data?.detail || error.message || '未知错误'
        ElMessage.error('获取设备列表失败: ' + errorMsg)
        devices.value = []
        total.value = 0
        updateDeviceStats()
      } finally {
        loading.value = false
      }
    }
    const updateDeviceStats = (data) => {
      if (data && typeof data.total === 'number') {
        // 使用后端返回的全局统计数据
        deviceStats.total = data.total || 0
        deviceStats.online = data.total_online || 0
        deviceStats.mobile = data.total_mobile || 0
        deviceStats.desktop = data.total_desktop || 0
      } else {
        // 兼容旧格式：基于当前页设备计算
        deviceStats.total = devices.value.length
        deviceStats.online = devices.value.filter(d => isOnline(d.last_access)).length
        deviceStats.mobile = devices.value.filter(d => d.device_type === 'mobile').length
        deviceStats.desktop = devices.value.filter(d => d.device_type === 'desktop').length
      }
    }
    const handleSizeChange = (val) => {
      pageSize.value = val
      currentPage.value = 1
      if (!hasActiveFilters.value) {
        fetchDevices()
      }
    }
    const handleCurrentChange = (val) => {
      currentPage.value = val
      if (!hasActiveFilters.value) {
        fetchDevices()
      }
    }
    const refreshDevices = () => {
      currentPage.value = 1
      fetchDevices()
    }
    const applyDeviceFilters = () => {
      currentPage.value = 1
    }
    const resetDeviceFilters = () => {
      filters.keyword = ''
      filters.device_type = ''
      filters.online_status = ''
      currentPage.value = 1
    }
    const normalizeUpgradeSubscription = (data) => ({
      device_limit: data?.device_limit || data?.total_devices || data?.maxDevices || 0,
      maxDevices: data?.device_limit || data?.total_devices || data?.maxDevices || 0,
      expire_time: data?.expire_time || data?.expiryDate || data?.expires_at,
      expiryDate: data?.expire_time || data?.expiryDate || data?.expires_at
    })
    const loadUpgradeSubscription = async () => {
      upgradeSubscriptionLoading.value = true
      try {
        const response = await subscriptionAPI.getUserSubscription()
        const data = response?.data?.data || response?.data || null
        if (!data || response?.data?.success === false) {
          upgradeSubscription.value = null
          return null
        }
        upgradeSubscription.value = normalizeUpgradeSubscription(data)
        return upgradeSubscription.value
      } catch (error) {
        upgradeSubscription.value = null
        return null
      } finally {
        upgradeSubscriptionLoading.value = false
      }
    }
    const openUpgradeDrawer = async () => {
      const subscription = upgradeSubscription.value || await loadUpgradeSubscription()
      if (!subscription?.device_limit && !subscription?.maxDevices) {
        ElMessage.warning('当前没有可升级的订阅，请先购买套餐')
        return
      }
      showUpgradeDrawer.value = true
    }
    const handleUpgradeSuccess = async () => {
      await Promise.all([fetchDevices(), loadUpgradeSubscription()])
    }
    const removeDevice = async (deviceId) => {
      try {
        await confirmWarning('确定要移除这个设备吗？移除后该设备将需要重新连接并重新获取订阅。', {
          title: '确认移除设备',
          confirmButtonText: '确认移除'
        })
        const device = devices.value.find(d => d.id === deviceId)
        if (device) {
          device.removing = true
        }
        await subscriptionAPI.removeDevice(deviceId)
        ElMessage.success('设备移除成功')
        await fetchDevices()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('移除设备失败: ' + (error.response?.data?.message || error.message))
        }
      }
    }
    const getDeviceIcon = (deviceType) => {
      const icons = {
        mobile: Cellphone,
        desktop: Monitor,
        tablet: Iphone,
        router: Connection,
        tv_box: VideoCamera,
        server: Box,
        unknown: QuestionFilled
      }
      return icons[deviceType] || icons.unknown
    }
    const getDeviceTypeName = (deviceType) => {
      const names = {
        mobile: '手机',
        desktop: '电脑',
        tablet: '平板',
        router: '路由器',
        tv_box: '电视盒子',
        server: '服务器',
        unknown: '未知'
      }
      return names[deviceType] || '未知'
    }
    const getDeviceTypeColor = (deviceType) => {
      const colors = {
        mobile: 'primary',
        desktop: 'success',
        tablet: 'warning',
        router: '',
        tv_box: 'danger',
        server: 'info',
        unknown: 'info'
      }
      return colors[deviceType] || colors.unknown
    }
    const formatTime = (time) => {
      return formatTimeUtil(time) || '未知'
    }
    const truncateUserAgent = (ua) => {
      if (!ua) return '未知'
      return ua.length > 50 ? ua.substring(0, 50) + '...' : ua
    }
    const isOnline = (lastAccess) => {
      if (!lastAccess) return false
      try {
        const lastTime = dayjs(lastAccess).tz('Asia/Shanghai')
        const now = dayjs().tz('Asia/Shanghai')
        const diffHours = now.diff(lastTime, 'hour')
        return diffHours < 24
      } catch (e) {
        return false
      }
    }
    const saveRemark = async (device, value) => {
      const newRemark = String(value || '').trim()
      const oldRemark = device.remark || ''
      if (newRemark === oldRemark) {
        return
      }
      const deviceId = device.id || device.device_id || device.deviceId
      if (!deviceId) {
        ElMessage.error('更新备注失败: 缺少设备ID')
        return
      }
      try {
        device.savingRemark = true
        const response = await subscriptionAPI.updateDeviceRemark(deviceId, newRemark)
        if (response?.data?.success === false) {
          throw new Error(response.data.message || '后端拒绝更新备注')
        }
        device.remark = newRemark || ''
        ElMessage.success('备注已更新')
      } catch (error) {
        device.remark = oldRemark
        ElMessage.error('更新备注失败: ' + (error.response?.data?.message || error.response?.data?.detail || error.message))
        return
      } finally {
        device.savingRemark = false
      }
    }
    onMounted(() => {
      fetchDevices()
      loadUpgradeSubscription()
    })
    return {
      loading,
      devices,
      deviceStats,
      otherDeviceCount,
      filters,
      hasActiveFilters,
      filteredDevices,
      displayedDevices,
      displayTotal,
      mobileDeviceFields,
      currentPage,
      pageSize,
      total,
      fetchDevices,
      refreshDevices,
      applyDeviceFilters,
      resetDeviceFilters,
      removeDevice,
      handleSizeChange,
      handleCurrentChange,
      getDeviceIcon,
      getDeviceTypeName,
      getDeviceTypeColor,
      formatTime,
      truncateUserAgent,
      formatLocation,
      deviceTableRef,
      columnWidths,
      handleDeviceColumnResize,
      showUpgradeDrawer,
      upgradeSubscription,
      upgradeSubscriptionLoading,
      openUpgradeDrawer,
      handleUpgradeSuccess,
      saveRemark
    }
  }
}
</script>
<style scoped lang="scss">
.devices-table {
  width: 100%;
}
.devices-filter-card {
  margin-bottom: 14px;
}
.devices-filter-body {
  padding: 16px;
}
.devices-filter-form {
  display: grid;
  grid-template-columns: minmax(220px, 1.2fr) repeat(2, minmax(150px, 0.8fr)) minmax(150px, max-content);
  align-items: end;
  gap: 12px;
  width: 100%;

  :deep(.el-form-item) {
    margin: 0;
    min-width: 0;
  }

  :deep(.el-form-item__label) {
    color: #606266;
    font-weight: 600;
  }
}
.devices-keyword-input {
  width: 100%;
  min-width: 0;
}
.devices-filter-select {
  width: 100%;
  min-width: 0;
}
.devices-filter-actions {
  justify-self: end;

  :deep(.el-form-item__content) {
    display: flex;
    flex-wrap: nowrap;
    gap: 8px;
  }
}
@media (max-width: 1100px) {
  .devices-filter-form {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .devices-filter-actions {
    justify-self: start;
  }
}
.grid {
  display: grid;
  gap: 14px;
}
.grid.cols-2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.devices-container .grid.cols-4 {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}
.card {
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  background: #fff;
  overflow: hidden;
}
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid #ebeef5;
  color: #303133;
  font-weight: 800;
}
.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  color: #303133;
  font-size: 16px;
  font-weight: 700;
}
.card-body {
  padding: 16px;
}
.summary-list {
  display: grid;
  gap: 8px;
}
.summary-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 8px;
  border-bottom: 1px dashed #ebeef5;
  color: #606266;
}
.summary-row:last-child {
  padding-bottom: 0;
  border-bottom: 0;
}
.summary-row strong {
  color: #303133;
  text-align: right;
}
.button-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.device-type-stat-card {
  align-items: flex-start;
}
.device-type-stat-content {
  min-width: 0;
}
.device-type-stat-title {
  color: #303133;
  font-size: 18px;
  font-weight: 800;
  line-height: 1.25;
}
.device-type-stat-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 6px;
  color: #606266;
  font-size: 12px;
  line-height: 1.4;
}
.device-type-stat-list span {
  padding: 2px 6px;
  border-radius: 999px;
  background: #f5f7fa;
  white-space: nowrap;
}
.devices-table :deep(.inline-editable-text__display) {
  border-color: #dcdfe6;
  background: #f8fafc;
}
.devices-table :deep(.inline-editable-text__display .el-icon) {
  opacity: 1;
}
.software-tag,
.location-tag {
  margin-left: 8px;
}
.stacked-tag {
  margin-top: 4px;
}
.device-name {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
}
.card-header-icon,
.device-type-icon {
  flex-shrink: 0;
  color: var(--primary-color);
}
.device-type-icon {
  font-size: 1.2rem;
  margin-top: 2px;
}
.device-name-details {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.device-main-name {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  word-break: break-all; /* 防止长名称溢出 */
}
.device-name-text {
  font-weight: 500;
  color: #303133;
}
.device-model-info {
  display: flex;
  align-items: center;
}
.os-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.os-name {
  font-weight: 500;
  color: #303133;
}
.os-version {
  display: flex;
  align-items: center;
}
.ip-address {
  font-family: 'Courier New', monospace;
  color: #666;
  font-size: 0.9rem;
}
.ip-location-cell {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  .ip-address {
    font-family: 'Courier New', monospace;
    color: #303133;
    font-size: 13px;
    font-weight: 500;
    padding: 2px 0;
  }
  :deep(.el-tag) {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    margin-left: 0;
    .el-icon {
      font-size: 12px;
    }
  }
  .no-location-text {
    font-size: 12px;
    color: var(--el-text-color-secondary, #6b7280);
    font-style: italic;
    margin-left: 8px;
  }
  .no-location {
    font-size: 12px;
    color: var(--el-text-color-secondary, #6b7280);
    font-style: italic;
  }
}
.user-agent {
  color: #666;
  font-size: 0.9rem;
}
.mobile-device-header {
  min-width: 0;
  .device-main-name {
    font-size: 15px;
    line-height: 1.45;
    .device-type-icon {
      font-size: 16px;
      margin-top: 0;
    }
  }
}
.mobile-ip-cell {
  align-items: flex-start;
  .ip-address {
    padding: 6px 10px;
    border: 1px solid #e5e7eb;
    border-radius: 6px;
    background: #f5f7fa;
  }
}
.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
  &.mobile-pagination {
    display: none;
  }
}
@media (max-width: 768px) {
  .pagination {
    justify-content: center;
    &.mobile-pagination {
      display: flex;
    }
  }
  .devices-container {
    padding: 10px;
  }
  .devices-filter-form {
    display: grid;
    grid-template-columns: 1fr;
    align-items: stretch;
  }
  .devices-keyword-input,
  .devices-filter-select,
  .devices-filter-actions,
  .devices-filter-actions :deep(.el-form-item__content),
  .devices-filter-actions :deep(.el-button) {
    width: 100%;
  }
  .stats-row {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
    margin-bottom: 12px;
    .stat-card {
      padding: 12px;
      .stat-label {
        font-size: 0.75rem;
      }
    }
  }
  .grid.cols-2 {
    grid-template-columns: 1fr;
  }
  .devices-container .grid.cols-4 {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .devices-card {
    :deep(.el-card__header) {
      padding: 12px;
      .card-header {
        flex-direction: column;
        align-items: flex-start;
        gap: 12px;
        .el-button {
          width: 100%;
          min-height: 44px;
          font-size: 16px;
        }
      }
    }
    :deep(.el-card__body) {
      padding: 12px;
    }
  }
  .table-wrapper {
    display: none;
  }
  .mobile-device-header {
    .device-name-details {
      gap: 8px;
    }
    .software-tag {
      margin-left: 0;
    }
  }
  .devices-container :deep(.mobile-card-actions .el-button) {
    min-height: 44px;
    touch-action: manipulation;
  }
  .devices-container :deep(.el-dialog) {
    width: 92% !important;
    margin: 4vh auto !important;
    max-height: calc(100dvh - 8vh);
  }
  .devices-container :deep(.el-dialog__body) {
    padding: 15px !important;
    max-height: calc(100dvh - 8vh - 124px);
    overflow-y: auto;
  }
  .devices-container :deep(.el-dialog__footer) {
    padding: 12px 15px max(14px, env(safe-area-inset-bottom)) !important;
    .el-button {
      width: 100%;
      margin: 0 0 10px 0 !important;
      min-height: 44px;
      font-size: 16px;
      &:last-child {
        margin-bottom: 0;
      }
    }
  }
}
</style> 
