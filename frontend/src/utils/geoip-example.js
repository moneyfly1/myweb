// GeoIP 查询 API 封装
export const geoipAPI = {
  // 查询单个 IP 地理位置
  lookup: (ip) => api.get(`/geoip/lookup?ip=${ip}`),

  // 批量查询 IP 地理位置
  batchLookup: (ips) => api.post('/geoip/batch-lookup', { ips })
}

// 使用示例：

// 1. 在设备列表中添加"查看位置"功能
// <template>
//   <el-table :data="devices">
//     <el-table-column prop="ip_address" label="IP地址">
//       <template #default="{ row }">
//         {{ row.ip_address }}
//         <el-button
//           v-if="row.ip_address && row.ip_address !== '-'"
//           type="text"
//           size="small"
//           @click="showLocation(row)"
//           :loading="row.locationLoading"
//         >
//           {{ row.location || '查看位置' }}
//         </el-button>
//       </template>
//     </el-table-column>
//   </el-table>
// </template>

// <script setup>
// import { ref } from 'vue'
// import { geoipAPI } from '@/utils/api'

// const devices = ref([])

// const showLocation = async (row) => {
//   if (row.location) return // 已经查询过了
//
//   row.locationLoading = true
//   try {
//     const response = await geoipAPI.lookup(row.ip_address)
//     if (response.data && response.data.success) {
//       row.location = response.data.data.location || '未知'
//     }
//   } catch (error) {
//     console.error('查询位置失败:', error)
//     row.location = '查询失败'
//   } finally {
//     row.locationLoading = false
//   }
// }
// </script>

// 2. 鼠标悬停自动查询（更好的用户体验）
// <el-tooltip :content="row.location || '加载中...'" placement="top">
//   <span
//     @mouseenter="loadLocationOnHover(row)"
//     style="cursor: pointer; color: #409eff;"
//   >
//     {{ row.ip_address }}
//   </span>
// </el-tooltip>

// const loadLocationOnHover = async (row) => {
//   if (row.location || row.locationLoading) return
//
//   row.locationLoading = true
//   try {
//     const response = await geoipAPI.lookup(row.ip_address)
//     if (response.data && response.data.success) {
//       row.location = response.data.data.location || '未知'
//     }
//   } catch (error) {
//     row.location = '未知'
//   } finally {
//     row.locationLoading = false
//   }
// }

// 3. 批量查询（适用于需要显示所有位置的场景）
// const loadAllLocations = async () => {
//   const ips = devices.value
//     .map(d => d.ip_address)
//     .filter(ip => ip && ip !== '-')
//
//   if (ips.length === 0) return
//
//   try {
//     const response = await geoipAPI.batchLookup(ips)
//     if (response.data && response.data.success) {
//       const locationMap = {}
//       response.data.data.results.forEach(item => {
//         locationMap[item.ip] = item.location
//       })
//
//       devices.value.forEach(device => {
//         if (device.ip_address && locationMap[device.ip_address]) {
//           device.location = locationMap[device.ip_address]
//         }
//       })
//     }
//   } catch (error) {
//     console.error('批量查询位置失败:', error)
//   }
// }
