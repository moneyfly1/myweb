# Nodes 表结构和订阅配置生成说明

## 1. Nodes 表结构

### 数据库表：`nodes`

| 字段名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| `id` | uint | 主键，自增 | 1 |
| `name` | varchar(100) | 节点名称 | "香港-01" |
| `region` | varchar(50) | 地区（自动提取） | "香港" |
| `type` | varchar(20) | 节点类型 | "vmess", "vless", "trojan", "ss", "ssr" |
| `status` | varchar(20) | 在线状态 | "online", "offline" |
| `load` | float64 | 负载 | 0.0 |
| `speed` | float64 | 速度 | 0.0 |
| `uptime` | int | 运行时间（秒） | 0 |
| `latency` | int | 延迟（毫秒） | 0 |
| `description` | text | 描述（可选） | NULL |
| `config` | text | **节点完整配置（JSON格式）** | `{"type":"vmess","server":"1.2.3.4",...}` |
| `is_recommended` | bool | 是否推荐 | false |
| `is_active` | bool | 是否激活 | true |
| `last_test` | datetime | 最后测试时间 | NULL |
| `last_update` | datetime | 最后更新时间 | 2025-01-16 10:00:00 |
| `created_at` | datetime | 创建时间 | 2025-01-16 10:00:00 |
| `updated_at` | datetime | 更新时间 | 2025-01-16 10:00:00 |

### 关键字段说明

#### `config` 字段（最重要）
- **类型**：TEXT（JSON 字符串）
- **内容**：完整的 `ProxyNode` 结构序列化后的 JSON
- **示例**：
```json
{
  "name": "香港-01",
  "type": "vmess",
  "server": "1.2.3.4",
  "port": 443,
  "uuid": "00000000-0000-4000-8000-000000000009",
  "network": "ws",
  "tls": true,
  "udp": true,
  "options": {
    "ws-opts": {
      "path": "/path",
      "headers": {
        "Host": "example.com"
      }
    }
  }
}
```

#### `region` 字段
- **自动提取**：从节点名称中识别地区关键词
- **支持的地区**：香港、台湾、日本、韩国、新加坡、美国、英国等40+个地区
- **识别方式**：匹配节点名称中的关键词（中英文都支持）

## 2. 节点数据流向

### 节点采集流程

```
节点采集任务 (RunUpdateTask)
    ↓
从 URL 获取节点链接
    ↓
解析节点链接 → ProxyNode 结构
    ↓
生成配置文件
    ├─→ 保存到 ./uploads/config/clash.yaml (覆盖)
    └─→ 保存到 system_configs 表 (覆盖)
    ↓
导入到 nodes 表
    ├─→ 提取节点名称 → name 字段
    ├─→ 提取地区信息 → region 字段
    ├─→ 节点类型 → type 字段
    ├─→ 序列化 ProxyNode → config 字段（JSON）
    └─→ 设置 is_active = true
```

### 订阅配置生成流程

```
客户订阅请求 (GenerateClashConfig)
    ↓
验证订阅有效性
    ↓
优先从 nodes 表获取节点
    ├─→ 查询条件：is_active = true
    ├─→ 从 config 字段解析 JSON → ProxyNode
    └─→ 使用 name 字段作为节点名称
    ↓
如果数据库中没有节点
    └─→ 从 URL 实时获取（兼容旧逻辑）
    ↓
生成 Clash YAML 配置
    ├─→ 使用 nodeToYAML 转换每个节点
    ├─→ 生成代理组配置
    └─→ 生成规则配置
    ↓
返回给客户端
```

## 3. 订阅配置生成逻辑

### 代码位置
`internal/services/config_update/config_update.go` → `GenerateClashConfig`

### 详细步骤

1. **验证订阅**
   ```go
   - 检查订阅是否存在
   - 检查订阅是否激活 (is_active = true)
   - 检查订阅状态 (status = "active")
   - 检查订阅是否过期 (expire_time > now)
   ```

2. **从数据库获取节点**（优先）
   ```go
   dbNodes := db.Where("is_active = ?", true).Find(&nodes)
   
   for each node:
     - 从 config 字段解析 JSON
     - json.Unmarshal(config, &proxyNode)
     - 使用 dbNode.Name 作为节点名称
     - 添加到 proxies 列表
   ```

3. **生成 Clash 配置**
   ```go
   generateClashYAML(proxies)
     - 写入基础配置（端口、模式等）
     - 转换每个节点为 YAML 格式
     - 生成代理组（节点选择、自动选择、失败切换）
     - 生成规则（直连、代理）
   ```

## 4. 节点配置示例

### VMess 节点
```json
{
  "name": "香港-01",
  "type": "vmess",
  "server": "1.2.3.4",
  "port": 443,
  "uuid": "00000000-0000-4000-8000-000000000009",
  "network": "ws",
  "tls": true,
  "udp": true,
  "options": {
    "ws-opts": {
      "path": "/path",
      "headers": {
        "Host": "example.com"
      }
    }
  }
}
```

### VLESS 节点
```json
{
  "name": "台湾-01",
  "type": "vless",
  "server": "5.6.7.8",
  "port": 443,
  "uuid": "00000000-0000-4000-8000-000000000056",
  "network": "grpc",
  "tls": true,
  "udp": true,
  "options": {
    "grpc-opts": {
      "grpc-service-name": "service"
    }
  }
}
```

### Trojan 节点
```json
{
  "name": "日本-01",
  "type": "trojan",
  "server": "203.0.113.254",
  "port": 443,
  "password": "password123",
  "network": "tcp",
  "tls": true,
  "udp": true,
  "options": {
    "skip-cert-verify": false
  }
}
```

## 5. 订阅配置能否正确获取节点信息？

### ✅ 可以正确获取

**原因：**

1. **数据完整性**
   - `config` 字段存储了完整的 `ProxyNode` 结构
   - 包含所有必要的连接信息（server, port, uuid/password, network, tls等）
   - 包含所有高级配置（ws-opts, grpc-opts等）

2. **序列化/反序列化匹配**
   - 存储时：`json.Marshal(proxyNode)` → 保存到 `config` 字段
   - 读取时：`json.Unmarshal(config, &proxyNode)` → 恢复 `ProxyNode` 结构
   - Go 的 JSON 包会自动处理字段映射

3. **节点名称处理**
   - 使用数据库中的 `name` 字段，确保名称一致性
   - 支持管理员手动修改节点名称

4. **容错机制**
   - 如果数据库中没有节点，自动回退到从 URL 获取
   - 如果某个节点的 config 解析失败，会跳过该节点，不影响其他节点

### 测试建议

1. **验证节点导入**
   ```sql
   SELECT id, name, region, type, is_active, 
          LENGTH(config) as config_length
   FROM nodes
   WHERE is_active = 1
   LIMIT 10;
   ```

2. **验证配置解析**
   ```sql
   SELECT id, name, config
   FROM nodes
   WHERE is_active = 1
   LIMIT 1;
   ```
   检查 `config` 字段是否为有效的 JSON

3. **测试订阅生成**
   - 访问订阅 URL
   - 检查返回的 YAML 配置
   - 验证节点信息是否完整

## 6. 注意事项

1. **节点更新**
   - 每次采集任务会更新已存在节点的 `config` 字段
   - 如果节点名称和类型相同，会更新而不是创建新记录

2. **地区识别**
   - 地区信息从节点名称自动提取
   - 如果无法识别，默认为"未知"
   - 支持中英文关键词识别

3. **节点激活**
   - 只有 `is_active = true` 的节点才会出现在订阅配置中
   - 管理员可以手动禁用某些节点

4. **性能优化**
   - 从数据库读取比从 URL 获取快得多
   - 减少对节点源 URL 的依赖
   - 提高订阅配置生成的稳定性

