# 节点管理功能说明

## 功能概述

节点管理是管理员后台的核心功能，用于管理系统中所有的代理节点。节点可以通过自动采集或手动添加的方式获得，管理员可以对节点进行查看、编辑、删除、测试等操作。

## 访问路径

- **管理员后台** → **节点管理** → **节点管理**

## 主要功能

### 1. 节点信息展示

列表显示以下节点信息：
- **节点ID**：系统自动分配的唯一标识符
- **节点名称**：节点的显示名称
- **类型**：节点协议类型（VMess、VLESS、Trojan、SS、SSR、Hysteria、TUIC、Naive、Anytls）
- **服务器**：节点服务器地址（域名或IP）
- **端口**：节点服务端口
- **地区**：节点所在地区（自动识别或手动设置）
- **状态**：节点状态（启用/禁用）
- **来源**：节点来源（采集/手动添加）
- **操作**：可执行的操作按钮

### 2. 搜索和筛选

- **关键词搜索**：支持通过节点名称、服务器地址搜索
- **类型筛选**：按节点协议类型筛选
- **地区筛选**：按节点所在地区筛选
- **状态筛选**：按节点状态筛选（全部/启用/禁用）
- **来源筛选**：按节点来源筛选（全部/采集/手动）

### 3. 批量操作

支持对选中的多个节点执行批量操作：
- **批量删除**：删除选中的节点
  - 普通节点（ID <= 1000000）：直接删除
  - 专线节点（ID > 1000000）：需要特殊处理

### 4. 单个节点操作

- **编辑节点**：修改节点信息（名称、服务器、端口、地区、状态等）
- **删除节点**：删除单个节点
- **测试节点**：测试节点的连通性和速度
- **批量测试**：测试选中的多个节点

### 5. 节点添加

#### 手动添加节点

1. **单个添加**
   - 点击"添加节点"按钮
   - 输入节点链接（支持各种协议格式）
   - 系统自动解析节点信息
   - 保存节点

2. **批量添加**
   - 点击"批量导入"按钮
   - 输入多个节点链接（每行一个）
   - 系统批量解析并导入
   - 显示导入结果（成功数、跳过数、错误数）

#### 自动采集节点

1. **配置节点源**
   - 进入"配置更新"页面
   - 添加节点源URL（支持多个，每行一个）
   - 保存配置

2. **执行采集**
   - 点击"立即采集"按钮
   - 系统自动从配置的URL下载节点
   - 解析并导入节点
   - 显示采集结果

3. **定时采集**
   - 可以设置定时采集任务
   - 系统按设定时间自动采集节点

### 6. 节点排序

节点在订阅中的显示顺序：
1. **专线节点优先**：分配给用户的专线节点显示在最前面
2. **按采集顺序**：按节点采集时的顺序显示
3. **按OrderIndex排序**：系统为每个节点分配了OrderIndex，用于控制显示顺序

### 7. 节点去重

系统自动对节点进行去重：
- **去重规则**：基于 `Type:Server:Port` 组合
- **去重时机**：采集或导入时自动去重
- **处理方式**：如果节点已存在，更新节点信息而不是创建新节点

## 节点采集原理

### 采集流程

```
1. 从配置的URL下载内容
   ↓
2. Base64解码（如果需要）
   ↓
3. 提取节点链接
   - 使用正则表达式匹配各种协议格式
   - 支持的协议：vmess://, vless://, trojan://, ss://, ssr://, hysteria://, tuic://, naive://, anytls://
   ↓
4. 解析节点信息
   - 解析协议类型
   - 提取服务器、端口、认证信息等
   - 解析节点名称（从URL fragment中提取）
   ↓
5. 识别节点地区
   - 从节点名称中提取地区信息
   - 从服务器地址中提取地区信息
   - 使用地区映射表匹配
   ↓
6. 节点去重
   - 检查节点是否已存在（基于Type:Server:Port）
   - 如果存在，更新节点信息
   - 如果不存在，创建新节点
   ↓
7. 分配OrderIndex
   - 按节点源URL的顺序分配
   - 第一个URL的节点OrderIndex = 10000 + 节点索引
   - 第二个URL的节点OrderIndex = 20000 + 节点索引
   - 以此类推
   ↓
8. 保存到数据库
```

### 节点链接格式

系统支持以下节点链接格式：

1. **VMess**
   ```
   vmess://base64(JSON)
   ```

2. **VLESS**
   ```
   vless://uuid@server:port?params#name
   ```

3. **Trojan**
   ```
   trojan://password@server:port?params#name
   ```

4. **Shadowsocks (SS)**
   ```
   ss://base64(method:password)@server:port#name
   ```

5. **ShadowsocksR (SSR)**
   ```
   ssr://base64(server:port:protocol:method:obfs:base64(password)/?params)
   ```

6. **Hysteria**
   ```
   hysteria://server:port?params#name
   ```

7. **TUIC**
   ```
   tuic://uuid:password@server:port?params#name
   ```

8. **Naive**
   ```
   naive+https://username:password@server:port#name
   naive://username:password@server:port#name
   ```

9. **Anytls**
   ```
   anytls://server:port?params#name
   ```

### 地区识别原理

系统通过以下方式识别节点地区：

1. **从节点名称识别**
   - 解析节点名称中的地区关键词
   - 支持中文、英文、国家代码
   - 例如："香港01"、"HK-01"、"Hong Kong"等

2. **从服务器地址识别**
   - 解析服务器域名或IP中的地区信息
   - 例如："hk1.example.com"、"jp.example.com"等

3. **地区映射表**
   - 系统维护了完整的地区映射表
   - 支持100+国家和地区
   - 支持200+城市
   - 支持多种语言和格式

### 节点清理机制

在定时采集任务执行时：

1. **清理采集节点**
   - 系统会先清理所有 `IsManual = false` 的节点
   - 然后导入新采集的节点
   - 手动添加的节点（`IsManual = true`）不会被清理

2. **保留手动节点**
   - 手动添加的节点始终保留
   - 不会因为采集任务而被删除

## 数据模型

### Node 模型

```go
type Node struct {
    ID          uint
    Name        string
    Type        string    // 节点类型（vmess/vless/trojan/ss/ssr/hysteria/tuic/naive/anytls）
    Server      string    // 服务器地址
    Port        int       // 端口
    UUID        string    // UUID（VMess/VLESS）
    Password    string    // 密码（Trojan/SS/SSR）
    Method      string    // 加密方法（SS/SSR）
    Protocol    string    // 协议（SSR）
    Obfs        string    // 混淆（SSR）
    Region      string    // 地区
    IsActive    bool      // 是否启用
    IsManual    bool      // 是否手动添加
    OrderIndex  int       // 排序索引
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

## API 接口

### 获取节点列表

```
GET /api/v1/admin/nodes
```

**查询参数：**
- `page`: 页码
- `size`: 每页数量
- `keyword`: 搜索关键词
- `type`: 节点类型筛选
- `region`: 地区筛选
- `status`: 状态筛选
- `source`: 来源筛选（manual/collected）

### 添加节点

```
POST /api/v1/admin/nodes
```

**请求体：**
```json
{
  "node_link": "vmess://..."
}
```

### 批量导入节点

```
POST /api/v1/admin/nodes/import-links
```

**请求体：**
```json
{
  "links": "vmess://...\nvless://...\ntrojan://..."
}
```

**响应示例：**
```json
{
  "success": true,
  "data": {
    "imported": 10,
    "skipped": 2,
    "error_count": 1,
    "errors": ["节点链接格式错误: ..."]
  }
}
```

### 批量删除节点

```
POST /api/v1/admin/nodes/batch-delete
```

**请求体：**
```json
{
  "node_ids": [1, 2, 3]
}
```

## 使用场景

### 场景1：手动添加节点

1. 点击"添加节点"按钮
2. 输入节点链接（如：`vmess://...`）
3. 系统自动解析节点信息
4. 确认并保存

### 场景2：批量导入节点

1. 点击"批量导入"按钮
2. 在文本框中输入多个节点链接（每行一个）
3. 点击"导入"按钮
4. 查看导入结果（成功数、跳过数、错误数）

### 场景3：配置自动采集

1. 进入"配置更新"页面
2. 在"节点源URL"中添加订阅地址（每行一个）
3. 保存配置
4. 点击"立即采集"按钮
5. 系统自动采集并导入节点

### 场景4：节点排序调整

1. 节点按采集顺序自动排序
2. 第一个URL的节点显示在最前面
3. 专线节点始终显示在最前面
4. 可以通过编辑节点修改OrderIndex调整顺序

## 注意事项

1. **节点去重**：系统自动去重，相同Type:Server:Port的节点只会保留一个
2. **采集清理**：定时采集会清理所有采集节点，手动节点不会被清理
3. **节点格式**：确保节点链接格式正确，否则无法解析
4. **地区识别**：系统会自动识别地区，也可以手动设置
5. **专线节点**：专线节点（ID > 1000000）有特殊处理，删除时需要谨慎

## 相关功能

- [订阅列表管理](./subscription_list_management.md)
- [专线节点管理](./custom_node_implementation.md)

