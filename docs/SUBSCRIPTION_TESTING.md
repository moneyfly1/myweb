# 订阅功能测试指南

## 设备限制逻辑说明

**重要规则**：
- 如果套餐允许 5 个设备，那么第 1-5 个设备可以正常获取节点
- 只有第 6 个及以后的设备（超过限制）才会返回错误或提醒信息
- 已存在的设备（通过 device_hash 识别）再次订阅时，不会触发设备限制检查

## 测试场景

### 场景 1：正常用户（有效期内，设备在限制内）

**前提条件**：
- 订阅状态：`is_active = true`, `status = "active"`
- 到期时间：未来日期
- 设备数量：< `device_limit`（例如：3/5）

**预期结果**：
- ✅ 返回完整的节点配置
- ✅ 包含信息节点：
  - 📢 网站域名
  - ⏰ 到期时间
  - 💬 售后QQ: 3219904322
- ✅ 包含所有可用节点

**测试步骤**：
```bash
# 使用 Clash 订阅
curl -H "User-Agent: ClashForWindows/1.0.0" \
  "http://localhost:8000/api/v1/subscriptions/clash/{subscription_url}"

# 使用 V2Ray 订阅
curl -H "User-Agent: V2RayNG/1.0.0" \
  "http://localhost:8000/api/v1/subscriptions/v2ray/{subscription_url}"

# 使用 SSR 订阅
curl -H "User-Agent: ShadowsocksR/1.0.0" \
  "http://localhost:8000/api/v1/subscriptions/ssr/{subscription_url}"
```

### 场景 2：到期用户

**前提条件**：
- 订阅状态：`is_active = true`, `status = "active"`
- 到期时间：过去日期（`expire_time < now()`）
- 设备数量：在限制内

**预期结果**：
- ✅ 返回节点配置（不阻止，但添加提醒）
- ✅ 包含提醒节点：`⚠️ 订阅已过期，请及时续费！`
- ✅ 包含所有信息节点

**测试步骤**：
```sql
-- 设置订阅为已过期
UPDATE subscriptions 
SET expire_time = NOW() - INTERVAL '1 day'
WHERE subscription_url = '{subscription_url}';
```

然后执行订阅请求。

### 场景 3：设备超限用户（但未到期）

**前提条件**：
- 订阅状态：`is_active = true`, `status = "active"`
- 到期时间：未来日期
- 设备数量：> `device_limit`（例如：6/5）

**预期结果**：
- ✅ 已存在的设备（前 5 个）：正常返回节点配置
- ✅ 包含提醒节点：`⚠️ 设备超限！当前 6/5，请删除多余设备`
- ❌ 新设备（第 6 个及以后）：返回错误或提醒配置

**测试步骤**：
```sql
-- 查看当前设备数量
SELECT COUNT(*) FROM devices 
WHERE subscription_id = {subscription_id} AND is_active = true;

-- 确保设备数量超过限制
-- 例如：device_limit = 5，当前有 6 个设备
```

**测试第 5 个设备（在限制内）**：
```bash
# 使用第 5 个设备的 User-Agent
curl -H "User-Agent: Device5/1.0.0" \
  "http://localhost:8000/api/v1/subscriptions/clash/{subscription_url}"
```
预期：✅ 正常返回节点

**测试第 6 个设备（超过限制）**：
```bash
# 使用新设备的 User-Agent（不同的设备指纹）
curl -H "User-Agent: Device6/1.0.0" \
  "http://localhost:8000/api/v1/subscriptions/clash/{subscription_url}"
```
预期：❌ 返回提醒配置或错误信息

### 场景 4：到期且设备超限用户

**前提条件**：
- 订阅状态：`is_active = true`, `status = "active"`
- 到期时间：过去日期
- 设备数量：> `device_limit`

**预期结果**：
- ✅ 已存在的设备：返回配置，包含两个提醒节点
- ❌ 新设备：返回错误或提醒配置
- ✅ 提醒节点：
  - `⚠️ 订阅已过期，请及时续费！`
  - `⚠️ 设备超限！当前 X/Y，请删除多余设备`

### 场景 5：订阅失效用户

**前提条件**：
- 订阅状态：`is_active = false` 或 `status != "active"`
- 到期时间：任意
- 设备数量：任意

**预期结果**：
- ✅ 返回配置（不阻止）
- ✅ 包含提醒节点：`⚠️ 订阅已失效，请联系客服！`

## 设备限制检查逻辑

### 关键代码位置

**`internal/api/handlers/subscription_config.go`**：

```go
// 检查当前设备数量
var currentDeviceCount int64
db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&currentDeviceCount)

// 检查这个设备是否是新设备
deviceHash := deviceManager.GenerateDeviceHash(userAgent, ipAddress, "")
var existingDevice models.Device
isNewDevice := db.Where("device_hash = ? AND subscription_id = ?", deviceHash, subscription.ID).First(&existingDevice).Error != nil

// 如果是新设备，检查是否会超过限制
if isNewDevice && int(currentDeviceCount) >= subscription.DeviceLimit {
    // 设备超过限制，返回提醒配置
    // ...
}
```

### 逻辑说明

1. **设备识别**：通过 `device_hash`（基于 User-Agent 和 IP 地址生成）识别设备
2. **新设备判断**：如果数据库中不存在该 `device_hash`，则为新设备
3. **限制检查**：
   - 如果是**已存在的设备**：直接允许，不检查限制
   - 如果是**新设备**：
     - 当前设备数量 < 限制：允许，记录设备访问
     - 当前设备数量 >= 限制：拒绝，返回提醒配置

### 示例

假设 `device_limit = 5`：

| 当前设备数 | 设备类型 | 结果 |
|----------|---------|------|
| 4 | 新设备 | ✅ 允许（记录后变成 5 个） |
| 5 | 新设备 | ❌ 拒绝（已经是 5 个，不能再添加） |
| 5 | 已存在设备 | ✅ 允许（不增加设备数） |
| 6 | 新设备 | ❌ 拒绝 |
| 6 | 已存在设备 | ✅ 允许 |

## 测试脚本

### 创建测试用户和订阅

```sql
-- 1. 创建测试用户
INSERT INTO users (username, email, password_hash, role, created_at, updated_at)
VALUES ('test_user', 'test@example.com', '$2a$10$...', 'user', NOW(), NOW());

-- 2. 创建订阅（5 个设备限制，30 天后到期）
INSERT INTO subscriptions (user_id, subscription_url, device_limit, current_devices, is_active, status, expire_time, created_at, updated_at)
VALUES (
  (SELECT id FROM users WHERE username = 'test_user'),
  'test_subscription_url_123',
  5,
  0,
  true,
  'active',
  NOW() + INTERVAL '30 days',
  NOW(),
  NOW()
);
```

### 测试不同设备

```bash
# 设备 1-5：应该都能正常获取节点
for i in {1..5}; do
  echo "Testing device $i"
  curl -H "User-Agent: TestDevice$i/1.0.0" \
    -H "X-Forwarded-For: 192.168.1.$i" \
    "http://localhost:8000/api/v1/subscriptions/clash/test_subscription_url_123" \
    > device_$i.yaml
done

# 设备 6：应该返回错误或提醒
echo "Testing device 6 (should fail)"
curl -H "User-Agent: TestDevice6/1.0.0" \
  -H "X-Forwarded-For: 192.168.1.6" \
  "http://localhost:8000/api/v1/subscriptions/clash/test_subscription_url_123"
```

## 验证要点

1. **设备限制正确性**：
   - ✅ 第 5 个设备（在限制内）能正常获取节点
   - ❌ 第 6 个设备（超过限制）返回错误或提醒

2. **信息节点完整性**：
   - ✅ 网站域名节点
   - ✅ 到期时间节点
   - ✅ 售后QQ节点

3. **提醒节点正确性**：
   - ✅ 到期提醒（如果已过期）
   - ✅ 设备超限提醒（如果超限）
   - ✅ 订阅失效提醒（如果失效）

4. **不同格式支持**：
   - ✅ Clash 格式（YAML）
   - ✅ V2Ray 格式（Base64 编码的节点链接）
   - ✅ SSR 格式（Base64 编码的节点链接）

## 注意事项

1. **设备识别**：相同 User-Agent 和 IP 地址会被识别为同一设备
2. **设备计数**：只有 `is_active = true` 的设备才计入设备数量
3. **设备限制**：等于限制的设备是允许的，只有超过限制才算超限
4. **已存在设备**：已存在的设备再次订阅时，不会触发设备限制检查

