# CBoard API 文档

本文档根据 `internal/api/router/router.go` 整理，列出主要接口的请求方法与路径。除注明「无需认证」或「公开」外，均需在请求头携带有效 JWT：`Authorization: Bearer <token>`。管理员接口需登录且当前用户为管理员。

**基础路径**：`/api/v1`（下文路径均相对此前缀）。  
**健康检查**（无需认证）：`GET /health` — 返回服务状态与版本。

---

## 认证（/auth）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /auth/register | 用户注册 |
| POST | /auth/login | 用户登录 |
| POST | /auth/login-json | 用户登录（JSON  body） |
| POST | /auth/refresh | 刷新令牌 |
| POST | /auth/logout | 登出（需认证） |
| POST | /auth/verification/send | 发送验证码 |
| POST | /auth/verification/verify | 校验验证码 |
| POST | /auth/forgot-password | 忘记密码 |
| POST | /auth/reset-password | 通过验证码重置密码 |

---

## 支付回调（无需 CSRF，可被第三方调用）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST/GET | /payment/notify/:type | 支付异步/同步回调，:type 为支付方式 |

---

## 用户（/users，需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /users/me | 获取当前用户信息 |
| PUT | /users/me | 更新当前用户资料 |
| GET | /users/dashboard-info | 用户仪表盘信息 |
| POST | /users/change-password | 修改密码 |
| PUT | /users/preferences | 更新偏好设置 |
| GET | /users/notification-settings | 通知设置 |
| PUT | /users/notification-settings | 更新通知设置 |
| GET | /users/privacy-settings | 隐私设置 |
| PUT | /users/privacy-settings | 更新隐私设置 |
| GET | /users/my-level | 当前用户等级 |
| GET | /users/theme | 用户主题 |
| PUT | /users/theme | 更新用户主题 |
| GET | /users/login-history | 登录历史 |
| GET | /users/activities | 用户活动 |
| GET | /users/subscription-resets | 订阅重置记录 |
| GET | /users/devices | 当前用户设备列表 |

---

## 订阅（/subscriptions，需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /subscriptions | 订阅列表 |
| GET | /subscriptions/:id | 订阅详情 |
| POST | /subscriptions | 创建订阅 |
| GET | /subscriptions/user-subscription | 当前用户订阅 |
| GET | /subscriptions/devices | 当前订阅设备 |
| POST | /subscriptions/reset-subscription | 用户自助重置订阅 |
| POST | /subscriptions/send-subscription-email | 发送订阅邮件 |
| POST | /subscriptions/convert-to-balance | 订阅转余额 |
| DELETE | /subscriptions/devices/:id | 删除设备 |

---

## 订阅链接（公开，无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /subscribe/:url | 获取订阅配置（Clash/V2Ray 等），:url 为加密参数 |
| GET | /subscriptions/clash/:url | Clash 订阅 |
| GET | /subscriptions/universal/:url | 通用订阅 |
| GET | /client/subscribe | XBoard 兼容：客户端订阅 |

---

## 订单（/orders，需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /orders | 订单列表 |
| POST | /orders | 创建订单 |
| POST | /orders/upgrade-devices | 设备升级订单 |
| GET | /orders/stats | 订单统计 |
| POST | /orders/:orderNo/pay | 发起支付 |
| POST | /orders/:orderNo/cancel | 取消订单 |
| GET | /orders/:orderNo/status | 订单支付状态 |
| GET | /orders/id/:id | 按 ID 获取订单 |

---

## 套餐（/packages，公开读）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /packages | 套餐列表 |
| GET | /packages/:id | 套餐详情 |

---

## 支付（/payment，需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /payment/methods | 可用支付方式 |
| POST | /payment | 创建支付（下单后获取支付链接） |
| GET | /payment/status/:id | 查询支付状态 |
| GET | /payment-methods/active | 当前启用的支付方式（可无需认证） |

---

## 节点（/nodes）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /nodes | 节点列表（可选认证，未登录则仅公开节点） |
| GET | /nodes/stats | 节点统计 |
| GET | /nodes/:id | 节点详情 |
| POST | /nodes/:id/test | 测速（需认证） |
| POST | /nodes/batch-test | 批量测速（需认证） |
| POST | /nodes/import-from-clash | 从 Clash 配置导入（需认证） |

---

## 优惠券（/coupons）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /coupons | 可用优惠券列表 |
| GET | /coupons/:code | 按码查询优惠券 |
| POST | /coupons/verify | 校验优惠券 |
| GET | /coupons/my | 我的优惠券（需认证） |
| GET | /coupons/admin | 管理端优惠券列表（管理员） |
| GET | /coupons/admin/:id | 管理端优惠券详情（管理员） |
| POST | /coupons/admin | 创建优惠券（管理员） |
| PUT | /coupons/admin/:id | 更新优惠券（管理员） |
| DELETE | /coupons/admin/:id | 删除优惠券（管理员） |

---

## 通知（/notifications，需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /notifications | 通知列表 |
| GET | /notifications/unread-count | 未读数量 |
| PUT | /notifications/:id/read | 标记已读 |
| PUT | /notifications/read-all | 全部已读 |
| DELETE | /notifications/:id | 删除通知 |
| GET | /notifications/user-notifications | 用户通知列表 |
| GET | /notifications/admin/notifications | 管理端通知列表（管理员） |
| POST | /notifications/admin/notifications | 创建通知（管理员） |
| PUT | /notifications/admin/notifications/:id | 更新通知（管理员） |
| DELETE | /notifications/admin/notifications/:id | 删除通知（管理员） |

---

## 工单（/tickets，需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /tickets | 我的工单列表 |
| GET | /tickets/unread-count | 未读回复数 |
| GET | /tickets/:id | 工单详情 |
| POST | /tickets | 创建工单 |
| POST | /tickets/:id/reply | 回复工单 |
| POST | /tickets/:id/replies | 回复工单（同上） |
| PUT | /tickets/:id | 关闭工单 |
| GET | /tickets/admin/all | 全部工单（管理员） |
| GET | /tickets/admin/statistics | 工单统计（管理员） |
| GET | /tickets/admin/:id | 工单详情（管理员） |
| PUT | /tickets/admin/:id | 更新工单状态/分配（管理员） |

---

## 设备（/devices，需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /devices | 当前用户设备列表 |
| DELETE | /devices/:id | 删除设备 |

---

## 邀请（/invites）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /invites/validate/:code | 校验邀请码（可无需认证） |
| GET | /invites | 邀请码列表（需认证） |
| POST | /invites | 创建邀请码（需认证） |
| GET | /invites/stats | 邀请统计（需认证） |
| GET | /invites/reward-settings | 邀请奖励设置（需认证） |
| GET | /invites/my-codes | 我的邀请码（需认证） |
| PUT | /invites/:id | 更新邀请码（需认证） |
| DELETE | /invites/:id | 删除邀请码（需认证） |

---

## 充值（/recharge，需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /recharge | 充值记录 |
| GET | /recharge/status/:orderNo | 按订单号查充值状态 |
| GET | /recharge/admin | 管理端充值记录（管理员） |
| GET | /recharge/:id | 单条充值记录 |
| POST | /recharge | 创建充值订单 |
| POST | /recharge/:id/cancel | 取消充值 |

---

## 配置与设置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /config | 系统配置列表（需认证） |
| GET | /config/:key | 按 key 获取配置（需认证） |
| GET | /software-config | 软件配置（可公开） |
| GET | /mobile-config | 移动端配置（可公开） |
| PUT | /software-config | 更新软件配置（管理员） |
| GET | /settings/public-settings | 公开设置（可无需认证） |

---

## 统计（/statistics，需管理员）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /statistics | 统计概览 |
| GET | /statistics/revenue | 收入图表 |
| GET | /statistics/users | 用户统计 |
| GET | /statistics/user-trend | 用户趋势 |
| GET | /statistics/revenue-trend | 收入趋势 |
| GET | /statistics/regions | 地区统计 |

---

## 管理员（/admin，需管理员认证）

以下为管理员接口，前缀均为 `/api/v1/admin`。

### 仪表盘与概览
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admin/dashboard | 仪表盘 |
| GET | /admin/stats | 同上 |
| GET | /admin/users/recent | 最近用户 |
| GET | /admin/orders/recent | 最近订单 |
| GET | /admin/users/abnormal | 异常用户列表 |
| POST | /admin/users/abnormal/:id/mark-normal | 标记为正常 |

### 用户管理
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admin/users | 用户列表 |
| POST | /admin/users | 创建用户 |
| GET | /admin/users/:id | 用户详情 |
| GET | /admin/users/:id/details | 用户详细信息 |
| PUT | /admin/users/:id | 更新用户 |
| PUT | /admin/users/:id/status | 更新用户状态 |
| POST | /admin/users/:id/unlock-login | 解锁登录 |
| DELETE | /admin/users/:id | 删除用户 |
| POST | /admin/users/:id/reset-password | 重置密码 |
| POST | /admin/users/:id/login-as | 以该用户身份登录 |
| POST | /admin/users/batch-delete | 批量删除用户 |
| POST | /admin/users/batch-enable | 批量启用 |
| POST | /admin/users/batch-disable | 批量禁用 |
| POST | /admin/users/batch-send-subscription-email | 批量发送订阅邮件 |
| POST | /admin/users/batch-expire-reminder | 批量到期提醒 |

### 订单管理
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admin/orders | 订单列表 |
| PUT | /admin/orders/:id | 更新订单 |
| POST | /admin/orders/:id/refund | 退款 |
| DELETE | /admin/orders/:id | 删除订单 |
| GET | /admin/orders/export | 导出订单 |
| GET | /admin/orders/statistics | 订单统计 |
| POST | /admin/orders/bulk-mark-paid | 批量标记已支付 |
| POST | /admin/orders/bulk-cancel | 批量取消 |
| POST | /admin/orders/batch-delete | 批量删除订单 |

### 套餐管理
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admin/packages | 套餐列表 |
| POST | /admin/packages | 创建套餐 |
| PUT | /admin/packages/:id | 更新套餐 |
| DELETE | /admin/packages/:id | 删除套餐 |

### 节点管理
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admin/nodes | 节点列表 |
| GET | /admin/nodes/stats | 节点统计 |
| POST | /admin/nodes | 创建节点 |
| POST | /admin/nodes/import-links | 链接批量导入 |
| PUT | /admin/nodes/:id | 更新节点 |
| DELETE | /admin/nodes/:id | 删除节点 |
| POST | /admin/nodes/:id/test | 节点测速 |
| POST | /admin/nodes/batch-test | 批量测速 |
| POST | /admin/nodes/batch-delete | 批量删除节点 |
| POST | /admin/nodes/import-from-file | 从文件导入 |

### 专线节点（custom-nodes）
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admin/custom-nodes | 专线节点列表 |
| GET | /admin/custom-nodes/:id/users | 节点已分配用户 |
| POST | /admin/custom-nodes | 创建专线节点 |
| POST | /admin/custom-nodes/import-links | 批量导入专线链接 |
| POST | /admin/custom-nodes/batch-delete | 批量删除 |
| POST | /admin/custom-nodes/batch-assign | 批量分配用户 |
| POST | /admin/custom-nodes/batch-test | 批量测速 |
| POST | /admin/custom-nodes/:id/test | 单节点测速 |
| GET | /admin/custom-nodes/:id/link | 获取节点链接 |
| PUT | /admin/custom-nodes/:id | 更新专线节点 |
| DELETE | /admin/custom-nodes/:id | 删除专线节点 |
| GET | /admin/users/:id/custom-nodes | 用户已分配专线 |
| POST | /admin/users/:id/custom-nodes | 为用户分配专线 |
| DELETE | /admin/users/:id/custom-nodes/:node_id | 取消分配专线 |

### 工单与设备
| 方法 | 路径 | 说明 |
|------|------|------|
| PUT | /admin/tickets/:id/status | 更新工单状态（同 PUT /tickets/admin/:id） |
| GET | /admin/devices/stats | 设备统计 |
| DELETE | /admin/devices/:id | 删除设备（管理端） |
| POST | /admin/devices/batch-delete | 批量删除设备 |

### 订阅管理
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admin/subscriptions | 订阅列表 |
| PUT | /admin/subscriptions/:id | 更新订阅 |
| POST | /admin/subscriptions/:id/reset | 重置订阅 |
| POST | /admin/subscriptions/:id/extend | 延长订阅 |
| GET | /admin/subscriptions/:id/devices | 订阅设备列表 |
| POST | /admin/subscriptions/user/:id/reset-all | 重置用户全部订阅 |
| POST | /admin/subscriptions/user/:id/send-email | 发送订阅邮件 |
| DELETE | /admin/subscriptions/user/:id/delete-all | 清空用户设备 |
| GET | /admin/subscriptions/export | 导出订阅 |
| POST | /admin/subscriptions/batch-clear-devices | 批量清空设备 |
| POST | /admin/subscriptions/batch-delete | 批量删除订阅 |
| POST | /admin/subscriptions/batch-enable | 批量启用 |
| POST | /admin/subscriptions/batch-disable | 批量禁用 |
| POST | /admin/subscriptions/batch-reset | 批量重置 |
| POST | /admin/subscriptions/batch-send-email | 批量发送邮件 |
| GET | /admin/subscriptions/expiring | 即将到期订阅 |

### 系统设置
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admin/settings | 后台设置总览 |
| PUT | /admin/settings/general | 基本设置 |
| PUT | /admin/settings/registration | 注册设置 |
| PUT | /admin/settings/notification | 通知设置 |
| PUT | /admin/settings/announcement | 公告设置 |
| PUT | /admin/settings/security | 安全设置 |
| PUT | /admin/settings/theme | 主题设置 |
| PUT | /admin/settings/invite | 邀请设置 |
| PUT | /admin/settings/admin-notification | 管理员通知设置 |
| POST | /admin/settings/admin-notification/test/email | 测试邮件通知 |
| POST | /admin/settings/admin-notification/test/telegram | 测试 Telegram |
| POST | /admin/settings/admin-notification/test/bark | 测试 Bark |
| PUT | /admin/settings/node_health | 节点健康检查设置 |
| PUT | /admin/settings/backup | 备份设置 |
| GET | /admin/settings/geoip/status | GeoIP 状态 |
| POST | /admin/settings/geoip/update | 更新 GeoIP 库 |

### 采集与配置更新
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admin/config-update/status | 采集任务状态 |
| GET | /admin/config-update/config | 采集配置 |
| PUT | /admin/config-update/config | 更新采集配置 |
| POST | /admin/config-update/start | 启动采集 |
| POST | /admin/config-update/stop | 停止采集 |
| POST | /admin/config-update/test | 测试采集 |
| GET | /admin/config-update/files | 采集文件列表 |
| GET | /admin/config-update/logs | 采集日志 |
| POST | /admin/config-update/logs/clear | 清空采集日志 |
| POST | /admin/config-update | 更新订阅配置（兼容） |

### 邀请与等级
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admin/invites | 邀请码列表 |
| GET | /admin/invite-relations | 邀请关系 |
| GET | /admin/invite-statistics | 邀请统计 |
| GET | /admin/user-levels | 用户等级列表 |
| POST | /admin/user-levels | 创建等级 |
| PUT | /admin/user-levels/:id | 更新等级 |

### 邮件与配置
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admin/email-queue | 邮件队列 |
| GET | /admin/email-queue/statistics | 队列统计 |
| GET | /admin/email-queue/:id | 单条邮件详情 |
| DELETE | /admin/email-queue/:id | 从队列删除 |
| POST | /admin/email-queue/:id/retry | 重试发送 |
| POST | /admin/email-queue/clear | 清空队列 |
| GET | /admin/email-config | 邮件配置 |
| POST | /admin/email-config | 更新邮件配置 |
| GET | /admin/configs | 系统配置列表 |
| POST | /admin/configs | 新增配置 |
| PUT | /admin/configs/:key | 更新配置 |

### 支付配置与上传
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admin/payment-config | 支付配置列表 |
| POST | /admin/payment-config | 创建支付配置 |
| PUT | /admin/payment-config/:id | 更新支付配置 |
| POST | /admin/upload | 上传文件 |

### 备份
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /admin/backup | 创建备份并上传（如 GitHub/Gitee） |
| GET | /admin/backups | 备份列表 |
| POST | /admin/backup/test-gitee | 测试 Gitee 连接 |
| POST | /admin/backup/test-github | 测试 GitHub 连接 |
| GET | /admin/backup/upload-status/:taskId | 上传任务状态 |

### 监控与日志
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admin/monitoring/system | 系统信息 |
| GET | /admin/monitoring/database | 数据库统计 |
| GET | /admin/logs/audit | 审计日志 |
| GET | /admin/logs/login-attempts | 登录尝试日志 |
| GET | /admin/system-logs | 系统日志 |
| GET | /admin/logs-stats | 日志统计 |
| GET | /admin/export-logs | 导出日志 |
| POST | /admin/clear-logs | 清空日志 |
| GET | /admin/logs/registration | 注册日志 |
| GET | /admin/logs/subscription | 订阅日志 |
| GET | /admin/logs/balance | 余额日志 |
| GET | /admin/logs/commission | 佣金日志 |
| GET | /admin/logs/subscription-reset | 订阅重置日志 |
| GET | /admin/logs/email | 邮件日志 |

### 管理员个人
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admin/profile | 管理员资料 |
| PUT | /admin/profile | 更新资料 |
| POST | /admin/change-password | 修改密码 |
| GET | /admin/login-history | 登录历史 |
| GET | /admin/security-settings | 安全设置 |
| PUT | /admin/security-settings | 更新安全设置 |
| GET | /admin/notification-settings | 通知设置 |
| PUT | /admin/notification-settings | 更新通知设置 |

---

## 兼容与其它

- **XBoard 兼容**（需认证）：`GET /api/v1/user/info`、`GET /api/v1/user/subscribe`、`GET /api/v1/client/subscribe` 等，见路由中的 `xboardCompat` 与 `subscribePublic` 分组。

完整路由定义以代码为准：`internal/api/router/router.go`。
