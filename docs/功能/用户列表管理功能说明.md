# 用户列表管理功能说明

## 功能概述

用户列表管理是管理员后台的核心功能之一，用于管理系统中所有注册用户。该功能提供了完整的用户信息查看、搜索、筛选、编辑、删除等操作。

## 访问路径

- **管理员后台** → **用户管理** → **用户列表**

## 主要功能

### 1. 用户信息展示

列表显示以下用户信息：
- **用户ID**：系统自动分配的唯一标识符
- **邮箱**：用户注册邮箱（可点击查看详情）
- **用户名**：用户注册时设置的用户名（可点击查看详情）
- **状态**：用户账户状态（活跃/待激活/禁用/设备超限）
- **余额**：用户账户余额（支持排序）
- **注册时间**：用户注册的时间
- **最后登录**：用户最后一次登录的时间
- **订阅数量**：用户拥有的订阅数量
- **操作**：可执行的操作按钮

### 2. 搜索功能

- **关键词搜索**：支持通过邮箱或用户名进行搜索
- **状态筛选**：可按用户状态筛选（全部/活跃/待激活/禁用/设备超限）
- **注册时间筛选**：可按注册时间范围筛选
- **实时搜索**：输入关键词后自动搜索或按回车键搜索

### 3. 批量操作

支持对选中的多个用户执行批量操作：
- **批量启用**：激活选中的用户账户
- **批量禁用**：禁用选中的用户账户
- **批量发送订阅邮件**：向选中用户发送订阅地址邮件
- **批量发送到期提醒**：向即将到期的用户发送提醒邮件

### 4. 单个用户操作

- **查看详情**：点击用户邮箱或用户名查看完整的用户信息
  - 基本信息（ID、邮箱、用户名、状态、余额等）
  - 统计信息（总消费、重置次数、订阅数量等）
  - 订阅列表
  - 余额变动记录（充值记录、消费记录）
  - 专线节点分配
  - 最近活动记录
- **编辑用户**：修改用户信息（邮箱、用户名、余额、状态等）
- **删除用户**：删除用户账户（会同时删除相关数据）
- **登录为用户**：以该用户身份登录系统（用于客服支持）
- **重置订阅**：重置用户的订阅地址
- **发送订阅邮件**：向用户发送订阅地址邮件

### 5. 排序功能

- **余额排序**：按用户余额升序/降序排序
- **注册时间排序**：按注册时间排序
- **最后登录排序**：按最后登录时间排序

### 6. 分页功能

- 支持自定义每页显示数量（10/20/50/100条）
- 支持页码跳转
- 显示总记录数

## 数据模型

### User 模型

```go
type User struct {
    ID                  uint
    Username            string
    Email               string
    Password            string
    Balance             float64
    IsActive            bool
    IsVerified          bool
    IsAdmin             bool
    LastLogin           sql.NullTime
    SpecialNodeExpiresAt sql.NullTime
    DataSharing         bool
    Analytics           bool
    QQ                  sql.NullString
    CreatedAt           time.Time
    UpdatedAt           time.Time
}
```

### 用户状态说明

- **active**：活跃状态，用户可正常使用系统
- **inactive**：待激活状态，用户注册但未激活
- **disabled**：禁用状态，管理员手动禁用
- **device_overlimit**：设备超限状态，用户的订阅设备数超过限制

## API 接口

### 获取用户列表

```
GET /api/v1/admin/users
```

**查询参数：**
- `page`: 页码（默认1）
- `size`: 每页数量（默认20）
- `keyword`: 搜索关键词（邮箱或用户名）
- `status`: 状态筛选（active/inactive/disabled/device_overlimit）
- `start_date`: 注册开始日期
- `end_date`: 注册结束日期

**响应示例：**
```json
{
  "success": true,
  "data": {
    "users": [
      {
        "id": 1,
        "email": "user@example.com",
        "username": "user",
        "balance": 100.00,
        "is_active": true,
        "last_login": "2025-12-22 10:00:00",
        "subscription_count": 2
      }
    ],
    "total": 100,
    "page": 1,
    "size": 20
  }
}
```

### 获取用户详情

```
GET /api/v1/admin/users/:id
```

**响应示例：**
```json
{
  "success": true,
  "data": {
    "user_info": {
      "id": 1,
      "email": "user@example.com",
      "username": "user",
      "balance": 100.00
    },
    "statistics": {
      "total_spent": 500.00,
      "total_resets": 3,
      "recent_resets_30d": 1,
      "total_subscriptions": 2
    },
    "subscriptions": [...],
    "recharge_records": [...],
    "orders": [...],
    "recent_activities": [...]
  }
}
```

### 批量操作接口

```
POST /api/v1/admin/users/batch-enable
POST /api/v1/admin/users/batch-disable
POST /api/v1/admin/users/batch-send-subscription-email
POST /api/v1/admin/users/batch-expire-reminder
```

**请求体：**
```json
{
  "user_ids": [1, 2, 3]
}
```

## 使用场景

### 场景1：查找特定用户

1. 在搜索框输入用户的邮箱或用户名
2. 点击搜索按钮或按回车键
3. 系统显示匹配的用户列表

### 场景2：查看异常用户

1. 点击状态筛选下拉框
2. 选择"设备超限"状态
3. 系统显示所有设备超限的用户
4. 可以批量发送提醒邮件或单独处理

### 场景3：批量发送到期提醒

1. 使用注册时间筛选，选择即将到期的用户
2. 勾选需要提醒的用户
3. 点击"批量发送到期提醒"按钮
4. 系统向选中用户发送到期提醒邮件

### 场景4：客服支持

1. 搜索需要帮助的用户
2. 点击"登录为用户"按钮
3. 系统以该用户身份登录，可以查看用户视角的所有信息
4. 帮助用户解决问题后退出

## 注意事项

1. **删除用户**：删除用户会同时删除该用户的所有相关数据（订阅、订单、设备等），请谨慎操作
2. **余额修改**：修改用户余额时，建议记录操作原因，以便后续审计
3. **批量操作**：批量操作会影响多个用户，请确认后再执行
4. **设备超限状态**：设备超限状态是自动计算的，当用户的订阅设备数超过限制时自动标记
5. **登录为用户**：此功能会创建新的会话，退出时需要手动退出

## 相关功能

- [订阅列表管理](./subscription_list_management.md)
- [设备管理](./device_management.md)
- [订单管理](./order_management.md)

