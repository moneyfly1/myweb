# 代码库清理方案

## 一、空文件夹（可直接删除）

1. **`frontend/src/composables/`** - 空文件夹
   - 状态：✅ 已确认为空
   - 建议：删除

2. **`certs/`** - 空文件夹（如果确实为空）
   - 状态：需要确认是否用于存放证书
   - 建议：如果确实为空且不需要，可以删除

## 二、测试文件（可删除或保留）

### Go 测试文件
1. **`internal/core/auth/auth_test.go`** (82 行)
   - 引用：仅在测试时使用
   - 建议：保留（测试文件）

2. **`internal/utils/utils_test.go`** (88 行)
   - 引用：仅在测试时使用
   - 建议：保留（测试文件）

3. **`internal/utils/validator_test.go`** (55 行)
   - 引用：仅在测试时使用
   - 建议：保留（测试文件）

## 三、小文件分析（< 100 行）

### 后端 Go 文件

#### 可以合并的模型文件（建议合并到 `models.go`）
1. **`internal/models/package.go`** (27 行) - Package 模型
2. **`internal/models/recharge.go`** (29 行) - RechargeRecord 模型
3. **`internal/models/audit_log.go`** (31 行) - AuditLog 模型
4. **`internal/models/node.go`** (31 行) - Node 模型
5. **`internal/models/order.go`** (34 行) - Order 模型
6. **`internal/models/device.go`** (40 行) - Device 模型
7. **`internal/models/payment.go`** (45 行) - PaymentTransaction 模型
8. **`internal/models/token_blacklist.go`** (49 行) - TokenBlacklist 模型
9. **`internal/models/subscription.go`** (51 行) - Subscription 模型
10. **`internal/models/invite.go`** (55 行) - InviteCode 模型
11. **`internal/models/config.go`** (58 行) - SystemConfig 模型
12. **`internal/models/security.go`** (58 行) - SecurityLog 模型
13. **`internal/models/notification.go`** (61 行) - Notification 模型
14. **`internal/models/custom_node.go`** (65 行) - CustomNode 模型
15. **`internal/models/user_level.go`** (71 行) - UserLevel 模型
16. **`internal/models/activity.go`** (73 行) - UserActivity 和 LoginHistory 模型
17. **`internal/models/payment_config.go`** (99 行) - PaymentConfig 模型

**建议**：这些模型文件可以合并成几个大文件：
- `models/user.go` - 用户相关（User, UserLevel, UserActivity, LoginHistory）
- `models/order.go` - 订单相关（Order, Package, RechargeRecord, PaymentTransaction, PaymentConfig）
- `models/subscription.go` - 订阅相关（Subscription, Device, CustomNode）
- `models/system.go` - 系统相关（SystemConfig, SecurityLog, AuditLog, Notification, TokenBlacklist）
- `models/invite.go` - 邀请相关（InviteCode, InviteRelation）
- `models/node.go` - 节点相关（Node）

#### 可以合并的 Handler 文件
1. **`internal/api/handlers/monitoring.go`** (55 行)
   - 功能：系统监控（GetSystemInfo, GetDatabaseStats）
   - 引用：在 router.go 中注册
   - 建议：可以合并到 `admin.go` 或 `dashboard.go`

2. **`internal/api/handlers/xboard_compat.go`** (150 行)
   - 功能：XBoard 兼容性接口
   - 引用：在 router.go 中注册
   - 建议：如果不再需要 XBoard 兼容，可以删除；否则保留

#### 可以合并的 Service 文件
1. **`internal/services/email/template.go`** (95 行)
   - 建议：可以合并到 `email.go`

2. **`internal/services/payment/applepay.go`** (93 行)
   - 建议：可以合并到 `payment.go` 主文件

### 前端文件

#### 可以合并的文件
1. **`frontend/src/views/admin/Logs.vue`** (67 行)
   - 功能：日志管理容器（仅包含标签页）
   - 引用：在 router 中使用
   - 建议：可以合并到各个日志组件中，或者保留（因为它是容器组件）

2. **`frontend/src/components/tutorials/SoftwareTutorials.vue`** (62 行)
   - 功能：教程容器（仅包含标签页）
   - 引用：在 Help.vue 中使用
   - 建议：可以合并到 Help.vue 中

3. **`frontend/src/views/NotFound.vue`** (80 行)
   - 功能：404 页面
   - 引用：在 router 中使用
   - 建议：保留（独立功能）

#### 可能未使用的文件
1. **`frontend/src/config/theme.js`** (227 行)
   - 功能：主题配置
   - 引用：未找到引用
   - 建议：检查是否被 `store/theme.js` 替代，如果是则删除

2. **`frontend/src/utils/githubDownload.js`** (406 行)
   - 功能：GitHub 下载工具
   - 引用：在 Dashboard.vue 和 Help.vue 中使用
   - 建议：保留（有实际使用）

## 四、引用次数少的文件

1. **`internal/models/activity.go`** (73 行)
   - 引用：在多个地方使用（AbnormalUsers, Statistics, User, Dashboard）
   - 建议：保留

2. **`internal/models/token_blacklist.go`** (49 行)
   - 引用：在 auth.go 和 database.go 中使用
   - 建议：保留（安全功能）

3. **`internal/api/handlers/monitoring.go`** (55 行)
   - 引用：仅在 router.go 中注册
   - 建议：可以合并到 admin.go

## 五、具体清理建议

### 优先级 1：立即删除
1. **`frontend/src/composables/`** - 空文件夹

### 优先级 2：合并小文件

#### 后端模型文件合并方案
- **方案 A（推荐）**：按功能域合并
  - `models/user.go` - 合并 User, UserLevel, UserActivity, LoginHistory
  - `models/order.go` - 合并 Order, Package, RechargeRecord, PaymentTransaction, PaymentConfig
  - `models/subscription.go` - 合并 Subscription, Device, CustomNode
  - `models/system.go` - 合并 SystemConfig, SecurityLog, AuditLog, Notification, TokenBlacklist
  - `models/invite.go` - 保留独立（已有 InviteCode, InviteRelation）
  - `models/node.go` - 保留独立
  - `models/logs.go` - 保留独立（已有多个日志模型）
  - `models/ticket.go` - 保留独立
  - `models/coupon.go` - 保留独立

- **方案 B（保守）**：只合并最小的几个文件
  - 合并 `package.go` 到 `order.go`
  - 合并 `recharge.go` 到 `order.go`
  - 合并 `token_blacklist.go` 到 `security.go` 或 `user.go`

#### Handler 文件合并
- 合并 `monitoring.go` 到 `admin.go` 或 `dashboard.go`

#### Service 文件合并
- 合并 `email/template.go` 到 `email.go`
- 合并 `payment/applepay.go` 到 `payment.go`

#### 前端文件合并
- 合并 `SoftwareTutorials.vue` 到 `Help.vue`（如果 Help.vue 是唯一使用它的地方）

### 优先级 3：检查未使用文件
1. **`frontend/src/config/theme.js`**
   - 检查是否被 `store/theme.js` 替代
   - 如果未使用，删除

2. **`internal/api/handlers/xboard_compat.go`**
   - 检查是否还需要 XBoard 兼容性
   - 如果不需要，删除

## 六、合并后的文件结构建议

### 后端 models 目录
```
models/
  ├── user.go          # User, UserLevel, UserActivity, LoginHistory
  ├── order.go         # Order, Package, RechargeRecord, PaymentTransaction, PaymentConfig
  ├── subscription.go  # Subscription, Device, CustomNode
  ├── system.go        # SystemConfig, SecurityLog, AuditLog, Notification, TokenBlacklist
  ├── invite.go        # InviteCode, InviteRelation (保留)
  ├── node.go          # Node (保留)
  ├── logs.go          # 各种日志模型 (保留)
  ├── ticket.go        # Ticket (保留)
  └── coupon.go        # Coupon (保留)
```

### 前端 components/tutorials 目录
- 如果 `SoftwareTutorials.vue` 合并到 `Help.vue`，可以保留其他教程文件（它们被 SoftwareTutorials 使用）

## 七、风险评估

### 低风险（可以安全执行）
1. 删除空文件夹 `composables/`
2. 合并 `monitoring.go` 到 `admin.go`
3. 合并 `email/template.go` 到 `email.go`
4. 合并 `payment/applepay.go` 到 `payment.go`

### 中风险（需要仔细测试）
1. 合并模型文件（需要更新所有导入）
2. 合并 `SoftwareTutorials.vue` 到 `Help.vue`
3. 删除 `config/theme.js`（如果确认未使用）

### 高风险（不建议删除）
1. 测试文件（`*_test.go`）- 保留
2. `xboard_compat.go` - 如果还在使用，保留
3. `NotFound.vue` - 必需的路由组件

## 八、执行步骤建议

1. **第一步**：删除空文件夹
2. **第二步**：合并小的 Handler 和 Service 文件
3. **第三步**：检查并删除未使用的文件（如 `config/theme.js`）
4. **第四步**：合并前端小文件（如 `SoftwareTutorials.vue`）
5. **第五步**：最后考虑合并模型文件（需要大量导入更新）

## 九、预期收益

- **减少文件数量**：约 15-20 个文件
- **提高可维护性**：相关代码集中在一起
- **减少导入复杂度**：更少的 import 语句
- **保持功能完整**：不影响现有功能
