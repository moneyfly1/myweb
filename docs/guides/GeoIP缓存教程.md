# GeoIP 缓存优化方案

## 概述

本方案通过 Redis 缓存 + 前端按需加载，解决 GeoIP 查询性能问题。

## 性能对比

### 优化前
- 设备列表（100条）：10-50秒
- 充值记录列表：5-20秒
- 用户详情页：8-30秒

### 优化后
- 列表加载：50-200ms（不查询 GeoIP）
- 首次查询位置：200-500ms（查询 + 写缓存）
- 缓存命中：10-50ms（直接从 Redis 读取）
- 缓存命中率：80-90%

## 部署步骤

### 1. 启动 Redis（可选）

#### 方式一：Docker（推荐）
```bash
docker run -d --name redis -p 6379:6379 redis:alpine
```

#### 方式二：本地安装
```bash
# macOS
brew install redis
brew services start redis

# Ubuntu/Debian
sudo apt-get install redis-server
sudo systemctl start redis

# CentOS/RHEL
sudo yum install redis
sudo systemctl start redis
```

### 2. 配置环境变量

复制示例文件：
```bash
cp .env.redis.example .env
```

编辑 `.env` 文件：
```bash
REDIS_ADDR=localhost:6379
# REDIS_PASSWORD=your_password  # 如果有密码
```

### 3. 重启服务

```bash
# 停止旧服务
pkill cboard

# 启动新服务
./cboard
```

启动日志会显示：
```
Redis 连接成功
Redis 缓存已启用，GeoIP 查询将使用缓存加速
```

如果 Redis 连接失败，会显示：
```
Redis 初始化失败（缓存功能已禁用）
```
**注意**：即使 Redis 失败，系统仍然可以正常工作，只是没有缓存加速。

## API 使用

### 1. 查询单个 IP
```bash
GET /api/v1/geoip/lookup?ip=8.8.8.8
```

响应：
```json
{
  "success": true,
  "data": {
    "ip": "8.8.8.8",
    "location": "美国, 加利福尼亚"
  }
}
```

### 2. 批量查询
```bash
POST /api/v1/geoip/batch-lookup
Content-Type: application/json

{
  "ips": ["8.8.8.8", "1.1.1.1", "114.114.114.114"]
}
```

响应：
```json
{
  "success": true,
  "data": {
    "results": [
      {"ip": "8.8.8.8", "location": "美国, 加利福尼亚"},
      {"ip": "1.1.1.1", "location": "澳大利亚"},
      {"ip": "114.114.114.114", "location": "中国"}
    ],
    "total": 3
  }
}
```

## 前端集成

### 方式一：点击查看（推荐）

```vue
<template>
  <el-table :data="devices">
    <el-table-column prop="ip_address" label="IP地址">
      <template #default="{ row }">
        {{ row.ip_address }}
        <el-button
          v-if="row.ip_address && row.ip_address !== '-'"
          type="text"
          size="small"
          @click="showLocation(row)"
          :loading="row.locationLoading"
        >
          {{ row.location || '查看位置' }}
        </el-button>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup>
import { ref } from 'vue'
import { geoipAPI } from '@/utils/api'

const devices = ref([])

const showLocation = async (row) => {
  if (row.location) return

  row.locationLoading = true
  try {
    const response = await geoipAPI.lookup(row.ip_address)
    if (response.data && response.data.success) {
      row.location = response.data.data.location || '未知'
    }
  } catch (error) {
    row.location = '查询失败'
  } finally {
    row.locationLoading = false
  }
}
</script>
```

### 方式二：鼠标悬停（更好的体验）

```vue
<template>
  <el-tooltip :content="row.location || '加载中...'" placement="top">
    <span
      @mouseenter="loadLocationOnHover(row)"
      style="cursor: pointer; color: #409eff;"
    >
      {{ row.ip_address }}
    </span>
  </el-tooltip>
</template>

<script setup>
const loadLocationOnHover = async (row) => {
  if (row.location || row.locationLoading) return

  row.locationLoading = true
  try {
    const response = await geoipAPI.lookup(row.ip_address)
    if (response.data && response.data.success) {
      row.location = response.data.data.location || '未知'
    }
  } catch (error) {
    row.location = '未知'
  } finally {
    row.locationLoading = false
  }
}
</script>
```

### 方式三：批量加载（适用于需要显示所有位置）

```vue
<script setup>
const loadAllLocations = async () => {
  const ips = devices.value
    .map(d => d.ip_address)
    .filter(ip => ip && ip !== '-')

  if (ips.length === 0) return

  try {
    const response = await geoipAPI.batchLookup(ips)
    if (response.data && response.data.success) {
      const locationMap = {}
      response.data.data.results.forEach(item => {
        locationMap[item.ip] = item.location
      })

      devices.value.forEach(device => {
        if (device.ip_address && locationMap[device.ip_address]) {
          device.location = locationMap[device.ip_address]
        }
      })
    }
  } catch (error) {
    console.error('批量查询失败:', error)
  }
}

// 在数据加载完成后调用
onMounted(() => {
  loadDevices().then(() => {
    loadAllLocations()
  })
})
</script>
```

## 缓存策略

- **缓存时间**：24小时
- **缓存键格式**：`geoip:{ip_address}`
- **缓存值**：地理位置字符串或 "NULL"（表示无法解析）
- **自动清理**：Redis 自动过期

## 监控与维护

### 查看 Redis 状态
```bash
redis-cli info stats
```

### 查看缓存命中率
```bash
redis-cli info stats | grep keyspace_hits
redis-cli info stats | grep keyspace_misses
```

### 清理所有 GeoIP 缓存
```bash
redis-cli KEYS "geoip:*" | xargs redis-cli DEL
```

### 查看缓存大小
```bash
redis-cli DBSIZE
redis-cli INFO memory
```

## 故障排查

### Redis 连接失败
1. 检查 Redis 是否运行：`redis-cli ping`
2. 检查端口是否开放：`telnet localhost 6379`
3. 检查环境变量配置

### 缓存不生效
1. 检查启动日志是否显示 "Redis 缓存已启用"
2. 检查 Redis 内存是否充足
3. 使用 `redis-cli MONITOR` 查看实时命令

### 性能仍然慢
1. 检查缓存命中率（应该 > 80%）
2. 检查 Redis 网络延迟
3. 考虑使用本地 Redis 而不是远程

## 成本估算

- **内存占用**：1万个 IP ≈ 1MB
- **Redis 服务器**：最小配置即可（512MB 内存）
- **网络流量**：几乎可以忽略

## 注意事项

1. **Redis 是可选的**：即使没有 Redis，系统仍然可以正常工作
2. **缓存过期**：24小时后自动过期，确保数据不会太旧
3. **内存管理**：Redis 会自动清理过期键，无需手动维护
4. **安全性**：建议在生产环境设置 Redis 密码

## 进一步优化

如果还需要更高性能，可以考虑：

1. **增加缓存时间**：改为 7 天或 30 天
2. **预热缓存**：启动时批量查询常见 IP
3. **使用 CDN**：Cloudflare Workers 提供免费的地理位置信息
4. **数据库缓存**：将 IP 位置存入数据库表
