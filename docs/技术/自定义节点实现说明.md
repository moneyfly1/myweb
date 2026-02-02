# 专线节点定制功能实现文档

## 功能概述

本功能实现了完整的专线节点定制系统，包括：
1. 服务器管理
2. 自动节点搭建（通过XrayR API）
3. Cloudflare域名和证书自动配置
4. 流量控制和到期时间管理
5. 用户专线节点分配
6. 订阅中自动显示专线节点

## 数据模型

### 1. Server（服务器）
- 存储服务器SSH连接信息
- 支持服务器状态管理

### 2. CustomNode（专线节点）
- 节点基本信息（名称、协议、域名等）
- 流量控制（流量限制、已使用流量）
- 到期时间控制（可设置独立到期时间或遵循用户订阅到期时间）
- 证书信息（证书路径、到期时间）

### 3. UserCustomNode（用户-专线节点关联）
- 管理用户与专线节点的分配关系

### 4. CustomNodeTrafficLog（流量日志）
- 记录节点的流量使用情况

## 核心功能

### 1. 服务器管理
- **添加服务器**：填写服务器地址、端口、用户名、密码
- **测试连接**：验证服务器SSH连接
- **服务器列表**：查看所有服务器

### 2. 自动节点搭建
创建专线节点时，系统会：
1. 在Cloudflare中创建DNS记录（如果提供了域名）
2. 自动申请Let's Encrypt证书（使用acme.sh）
3. 通过XrayR API创建节点
4. 保存节点配置到数据库

### 3. 流量控制
- 可以为每个节点设置流量限制
- 系统会检查节点流量是否超限
- 超限的节点不会出现在用户订阅中

### 4. 到期时间控制
- **独立到期时间**：节点有自己的到期时间
- **遵循用户到期时间**：节点到期时间跟随用户订阅到期时间
- 过期的节点不会出现在用户订阅中

### 5. 用户分配
- 管理员可以为用户分配专线节点
- 用户可以同时看到普通线路和专线节点
- 专线节点在订阅中显示为"专线定制-xxx"

## 系统配置

需要在系统设置中配置以下参数（category: `custom_node`）：

1. **XrayR配置**
   - `xrayr_api_url`: XrayR API地址
   - `xrayr_api_key`: XrayR API密钥

2. **Cloudflare配置**
   - `cloudflare_api_key`: Cloudflare API Key（或使用API Token）
   - `cloudflare_email`: Cloudflare账号邮箱
   - `cloudflare_api_token`: Cloudflare API Token（优先使用，如果设置了则不需要API Key和Email）

3. **证书配置**
   - `cert_email`: Let's Encrypt证书申请邮箱

## API接口

### 服务器管理
```
GET    /api/v1/admin/servers              # 获取服务器列表
POST   /api/v1/admin/servers               # 创建服务器
PUT    /api/v1/admin/servers/:id           # 更新服务器
DELETE /api/v1/admin/servers/:id           # 删除服务器
POST   /api/v1/admin/servers/:id/test      # 测试服务器连接
```

### 专线节点管理
```
GET    /api/v1/admin/custom-nodes          # 获取专线节点列表
POST   /api/v1/admin/custom-nodes          # 创建专线节点（自动搭建）
PUT    /api/v1/admin/custom-nodes/:id      # 更新专线节点
DELETE /api/v1/admin/custom-nodes/:id      # 删除专线节点
GET    /api/v1/admin/custom-nodes/:id/traffic  # 获取节点流量统计
```

### 用户分配
```
GET    /api/v1/admin/users/:id/custom-nodes           # 获取用户的专线节点
POST   /api/v1/admin/users/:id/custom-nodes           # 为用户分配专线节点
DELETE /api/v1/admin/users/:id/custom-nodes/:node_id  # 取消分配
```

## 使用流程

### 1. 配置系统参数
在系统设置中配置XrayR、Cloudflare和证书相关参数。

### 2. 添加服务器
在服务器管理页面添加VPS服务器信息。

### 3. 创建专线节点
1. 选择服务器
2. 选择协议类型（vmess、vless、trojan等）
3. 填写域名（可选，如果填写会自动配置DNS和证书）
4. 设置流量限制和到期时间
5. 系统自动搭建节点

### 4. 分配节点给用户
在用户管理页面，为用户分配专线节点。

### 5. 用户订阅
用户订阅时会自动包含：
- 普通线路节点
- 分配给该用户的专线节点（显示为"专线定制-xxx"）

## 注意事项

1. **证书申请**：需要服务器上安装acme.sh，系统会调用acme.sh申请证书
2. **XrayR配置**：确保XrayR已正确配置API访问
3. **Cloudflare DNS**：域名需要在Cloudflare中管理
4. **流量统计**：需要定期从XrayR同步流量数据（可通过定时任务实现）
5. **到期检查**：系统会在生成订阅时自动检查节点到期时间和流量限制

## 后续优化建议

1. **定时任务**：
   - 定期同步XrayR节点流量
   - 定期检查证书到期时间并自动续期
   - 定期检查节点状态

2. **前端界面**：
   - 服务器管理页面
   - 专线节点管理页面
   - 用户分配界面

3. **监控告警**：
   - 节点流量接近限制时告警
   - 证书即将到期时告警
   - 节点状态异常时告警

4. **批量操作**：
   - 批量创建节点
   - 批量分配节点
   - 批量更新节点配置


