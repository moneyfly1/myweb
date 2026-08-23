# CBoard 全项目代码审查与优化报告

> **审查范围**：`/Users/apple/Downloads/myweb` 全部 338 个源文件（Go 后端 119 个文件 ≈ 5 万行；Vue3 前端 75 个视图/组件 + JS/SCSS ≈ 7.4 万行；部署脚本与文档）。
> **审查方法**：21 个并行深度分析代理逐一通读全部文件，按"逻辑 / 安全 / 性能 / 重复 / 风格 / 架构 / 错误处理 / UX / 可维护性"九维度逐文件产出发现，主代理对 critical/high 关键结论做了二次核实（含源码对照）。
> **产出**：
> - 本报告（汇总、分级、优化路线图）
> - [AUDIT_FINDINGS_FULL.md](./AUDIT_FINDINGS_FULL.md)（972 条逐文件发现全文，含行号与修改建议）

---

## 1. 总体统计

| 严重度 | 数量 | 说明 |
|--------|------|------|
| 🔴 Critical | 9 | 资金安全 / 凭据泄露 / 构建不可用 / 整页功能失效 |
| 🟠 High | 103 | 越权、伪造 IP 绕过限流、竞态、假功能、契约断裂 |
| 🟡 Medium | 383 | 一致性、性能、边界问题 |
| 🔵 Low | 443 | 风格、死代码、可复用性 |
| ⚪ Info | 34 | 健康项/提示 |
| **合计** | **972** | 覆盖 **249+ 个代码文件**（另含文档与配置） |

按类别分布：logic 337 / security 128 / duplication 89 / performance 84 / maintainability 83 / error-handling 73 / architecture 71 / style 61 / ux 24 / other 22。

**总体评价**：项目功能覆盖完整（支付、订阅、节点、工单、营销、数据看板一应俱全），后端 SQL 基本参数化、前端 XSS 面控制良好、组件复用意识较强。但存在三类系统性顽疾：**① 安全多为"表面防御"**（IP 限流可伪造绕过、支付回调可伪造免单、密钥明文回传/入库/进日志）；**② 前端大量"假功能"**（假成功提示、假搜索、假测试、恒空列表），前后端契约未统一；**③ 复制粘贴式开发**（日志页 80% 同构、教程组件 95% 相同、CSS `:has()` 网格复制 40+ 处、API 层重复 9 个方法），以及明显版本断裂（Go 1.24 vs 部署脚本 1.21.5、Dockerfile 构建必失败）。

---

## 2. 🔴 Critical 问题（9 个，必须立即处理）

### 2.1 苹果支付回调可被任意伪造免单
**文件**：`internal/services/payment/applepay.go:91-93`（已核实）
```go
func (s *ApplePayService) VerifyNotify(params map[string]string) bool { return true }
```
`payment.go:669-673` 对 applepay 渠道直接采信该返回值进入入账流程，且无 trade_status 校验。**攻击者只需 POST `/api/v1/payment/notify` 携带 `payment_type=applepay&out_trade_no=<订单号>` 即可零成本标记任意订单已支付。**
**修复**：在实现真正的 Apple 服务端验证前，禁止启用该渠道（`VerifyNotify` 返回 false + 配置校验拒绝上线）；或补齐 App Store Server API 收据校验。

### 2.2 易支付商户 RSA 私钥明文写入日志
**文件**：`internal/services/payment/yipay.go:609`（已核实）
```go
utils.LogInfo("易支付RSA签名: 私钥长度=%d, 内容前50字符=%s", len(s.MerchantPrivateKey), s.MerchantPrivateKey[:min(50, ...)])
```
PEM 头占 26 字符，前 50 字符已含约 24 字节真实密钥材料。**立即删除该日志行**，并全库检索其他打印密钥字节的日志（codepay.go:116 同样打印签名串）。

### 2.3 真实支付商户凭据硬编码进 Git 仓库
**文件**：`scripts/configure_payment.sh:10-14`、`scripts/configure_payment.sql`（已核实）
`YIPAY_PID="REDACTED_PID"`、`YIPAY_KEY="REDACTED_MERCHANT_KEY"` 明文提交并已在版本历史中。
**修复**：① 立即在支付后台重置密钥；② `git filter-repo` 重写历史清除凭据；③ 脚本改为环境变量/交互输入。

### 2.4 Dockerfile 构建必失败（版本断裂）
**文件**：`Dockerfile:2,14`（已核实）
- `FROM golang:1.21-alpine` 与 `go.mod` 的 `go 1.24.0` 不匹配；
- `CGO_ENABLED=0` 编译引入 `mattn/go-sqlite3`（cgo 实现）的项目必然报 "cgo: C files required but cgo is disabled"。
**修复**：Go 1.24 镜像 + `CGO_ENABLED=1`（装 gcc/musl-dev），或换纯 Go 的 `modernc.org/sqlite`。

### 2.5 用户等级升降级判定方向疑似整体颠倒
**文件**：`internal/services/order/order.go:1252-1290`（已核实）
```go
if targetLevel == nil || level.LevelOrder < targetLevel.LevelOrder { ... }  // 选"最小"达标档
if currentLevel.LevelOrder < targetLevel.LevelOrder { shouldUpgrade = false } // 当前低→禁止升级
```
两个比较方向互相矛盾：若 LevelOrder 越大等级越高，则**用户永远不会升级、且会被降级到最低达标档**（资金/权益级错误）。
**修复**：统一方向——选达标档中 LevelOrder 最大者；降级守卫为 `currentLevel.LevelOrder > targetLevel.LevelOrder` 时禁止；补单测覆盖"升级"与"不降级"。

### 2.6 ConvertSubscriptionToBalance 并发双花（资金竞态）
**文件**：`internal/api/handlers/subscription.go:1147-1163`（已核实）
- `tx.Set("gorm:query_option", "FOR UPDATE")` 是 **GORM v1 语法**，项目 v1.25.5（v2）下**行锁静默失效**（同仓 payment.go:350 已正确使用 `clause.Locking`）；
- 订阅行在事务外加载、事务内不复核状态，`tx.Delete` 不检查 RowsAffected——两个并发请求可重复入账。
**修复**：事务内 `tx.Clauses(clause.Locking{Strength:"UPDATE"})` 重新锁定并复核；`balance` 用 `gorm.Expr("balance + ?", n)` 原子累加。

### 2.7 AbnormalUsers「标记正常」是纯前端假操作
**文件**：`frontend/src/views/admin/AbnormalUsers.vue:559-572`（已核实）
`markAsNormal()` 确认后**没有任何 API 调用**（`api.js` 中无 mark-normal 方法），直接 `ElMessage.success('用户已标记为正常')`。用户点击后数据毫无变化却收到成功提示。
**修复**：新增后端接口 `POST /admin/users/:id/mark-normal`（router.go:355 其实**已有该路由**！）并封装 `adminAPI.markUserNormal`，await 成功后再提示。

### 2.8 EmailDetail 整页因 `response.success` 恒假而失效
**文件**：`frontend/src/views/admin/EmailDetail.vue:230-249,258-264,280-286`（已核实）
axios 拦截器返回完整 response（api.js:335），`if (response.success)` 恒为 undefined → 加载/重试/删除**全部永远报失败**，页面只能渲染"邮件不存在"。
**修复**：统一改为 `response.data?.success` 并解包 `response.data.data`（或复用其他页的 handleResponse 模式）。

### 2.9 备份将 .env / config.yaml 打进 zip 并自动上传第三方仓库
**文件**：`internal/api/handlers/backup.go:119-196`、`internal/services/scheduler/scheduler.go:575-685`
备份 zip 内含 `.env`（SMTP 密码、SECRET_KEY 等）与 config.yaml，且自动上传到**硬编码的作者仓库** `moneyfly/backup`、`moneyfly1/backup`（backup_service.go:36-117）。备份一旦落入他人仓库，全站凭据泄露。
**修复**：备份排除密钥类文件（或加密）；远程仓库目标改为用户配置的私有仓，禁止默认推送。

---

## 3. 🟠 High 问题（103 个，按模块摘录）

### 3.1 安全：认证 / 授权 / 信任边界

| 文件 | 行 | 问题 |
|------|----|------|
| `internal/utils/network.go` | 180-258 | `GetRealClientIP` **无条件信任** `CF-Connecting-IP`/`X-Forwarded-For`/`X-Real-IP`；配合 ratelimit.go:241-295（限流键取自客户端头）→ **暴力破解/注册/验证码限流可被伪造 IP 完全绕过**。router.go:15-17 还 `SetTrustedProxies(nil)`，信任模型自相矛盾。 |
| `internal/middleware/ratelimit.go` | 241-295 | 登录/注册/验证码限流键取自可伪造头；限流状态仅存进程内存（多实例 N 倍放宽、重启清零）；`RateLimitMiddleware`/`generalRateLimiter` 是死代码。 |
| `internal/api/handlers/auth.go` | 1182-1319 | `ResetPasswordByCode` **无限流**，6 位重置验证码可暴力枚举；验证码 `Used` 读改写非原子（models/security.go:40,52-57），并发可复用同一验证码。 |
| `internal/api/handlers/node.go` | 772-902,1037-1052 | `TestNode`/`BatchTestNodes`/`ImportFromClash` 仅挂 `AuthMiddleware`（router.go:179-185），**任意登录用户**可导入节点、翻转节点状态。 |
| `internal/api/router/router.go` | 187-194 | `GET /coupons` 与 `GET /coupons/:code` **完全公开**，未登录可枚举全部有效优惠码。 |
| `internal/api/handlers/download.go` | 22-59 | 未认证的下载解析接口 = **SSRF + 开放重定向**组合漏洞（且跟随重定向、协议二次校验缺失）。 |
| `internal/api/handlers/ticket.go` | 450-464 | `GetTicket` 向普通用户泄露 `admin_notes`/`rating`。 |
| `internal/middleware/maintenance.go` | 125-158 | 维护页 siteName/message/logoURL **未转义拼入 HTML**，存储型 XSS（全站用户都会加载）。 |
| `internal/api/handlers/user.go` | 1551-1586 | `CreateUser` 把**明文密码**写入管理员通知与欢迎邮件。 |
| `internal/api/handlers/backup.go` | 119-196 | 备份含密钥并自动上传（见 Critical 2.9）。 |
| `internal/api/handlers/repo_sync.go` / router.go:37-38 | — | `/repo-sync/*filepath` 无鉴权公开文件服务 + 目录列表。 |
| `internal/services/repo_sync/repo_sync.go` | 403-432 | 同步文件公开访问、符号链接跟随（repo_sync.go:69-93）。 |
| `internal/services/git/git.go` | 484-490 | Gitee `access_token` 拼进 URL query，进代理/访问日志。 |
| `internal/services/payment/query.go` | 84-118 | 商户 key 明文放查单 URL query，响应不验签。 |
| `internal/services/payment/wechat.go` | 61-89 | 统一下单响应未校验微信签名。 |
| `internal/services/payment/yipay.go` | 529-533 | MD5+RSA 缺 rsa_sign 时**降级仅 MD5** 校验并通过。 |
| `scripts/admin_tool.go` | 118-149 | username/email **OR 查询**可接管任意用户并提升为管理员。 |
| `install-vps.sh` | 124 | 明文 HTTP 下载阿里云盾卸载脚本并以 root 执行（供应链风险）。 |
| `install.sh` | 113 | Docker Redis 无密码映射 `0.0.0.0:6379`；`FLUSHDB` 无 -a 参数（:597,900）。 |
| `frontend/src/router/index.js` | 263-268 | 路由守卫 catch **直接 next() 放行**，异常绕过鉴权。 |
| `frontend/src/views/Invites.vue` | 459-466 | 客户端可提交 `inviter_reward`/`invitee_reward`，后端非零值完全信任 → **可自邀刷奖励**。 |
| `frontend/src/views/admin/Subscriptions.vue` | 1399-1597 | "以用户身份登录"把仿冒会话令牌写入 localStorage/sessionStorage 且未限制仿冒管理员。 |
| `frontend/src/views/admin/PaymentConfig.vue` | 821-863 | 支付配置列表明文回显全部密钥。 |
| `frontend/src/views/admin/Users.vue` | 1602-1640,1725-1732 | 单行删除/禁用绕过 checkAdminUsers 保护（可自锁/越权）。 |
| `frontend/src/utils/api.js` | 173-183 | `PUBLIC_APIS` 用 **startsWith 前缀匹配**，私有端点可能被误判公开而跳过 Authorization。 |

### 3.2 逻辑：假功能 / 契约断裂 / 竞态

| 文件 | 行 | 问题 |
|------|----|------|
| `frontend/src/views/Orders.vue` | 805-813,685-691 | 充值 tab 分页失效（翻页总加载订单）；`keyword` 搜索**后端根本未实现**（假搜索）。 |
| `frontend/src/views/UserSettings.vue` | 671-696,55-66 | 邮箱修改假成功（后端忽略 email/verification_code）；头像上传 `action="#"` 无处理器，不可用。 |
| `frontend/src/views/admin/Dashboard.vue` | 448-465 | 异常客户取数路径与后端 `{users,total}` 结构不匹配，**恒为空**。 |
| `frontend/src/views/admin/CustomNodes.vue` | 1373-1378 | `testNodeFromLink` 是**空桩**，未调接口却弹"测试连接通过"。 |
| `internal/api/handlers/custom_node.go` | 931-1056 | `TestCustomNode`/`BatchTestCustomNodes` 假测试：无连通性检查就置 active 并伪造 100ms 延迟。 |
| `internal/api/handlers/custom_node.go` | 602-607,574 | 部分更新用旧列值覆盖新配置；`DisplayName != "" \|\| == ""` 恒真。 |
| `internal/api/handlers/dashboard.go` | 115-128 | 公告查询 `category="system"`，实际存 `announcement` → **公告永不显示**。 |
| `internal/api/handlers/order.go` | 1770-1841 | `UpgradeDevices` 扣款后支付链接生成失败**不退回余额**。 |
| `internal/api/handlers/subscription.go` | 209-280,1943-2023 | 订阅重置非事务（失败时 URL 已轮换）；`GetUniversalSubscription` 缺过期/停用校验。 |
| `internal/api/handlers/user.go` | 1860-1889,2310-2328 | 删除顺序错误：先删 subscriptions 再删 devices（恒空）；关联表残留（CheckinRecord/LoginAttempt 等）；`LoginAsUser` 不查 IsActive；`UpdateUserStatus` 无自我保护。 |
| `internal/api/handlers/analytics.go` | 175-180 | 硬编码 `is_active = 1`，**PostgreSQL 下直接报错**。 |
| `internal/api/handlers/coupon.go` | 424-428 | `UpdateCoupon` 恒真 else-if 导致任何更新都清空适用套餐。 |
| `internal/utils/common.go` | 185-204 | 订单号生成 **TOCTOU 竞态**：并发可拿到相同订单号（资金相关）。 |
| `internal/utils/response.go` | 175-183 | `ParsePagination` 的 limit 仅在 size==20 时生效，分页逻辑错误。 |
| `internal/core/database/database.go` | 187-229 | custom_nodes 重建忽略备份错误仍 DROP 原表，**数据丢失风险**。 |
| `internal/middleware/csrf.go` | 127-137 | 每次 GET 重新生成并覆盖 CSRF token → 多标签页/并发 GET 合法请求 403。 |
| `internal/middleware/brotli.go` | 26-37 | 压缩 writer 未转发 Flush → **全局压缩下 SSE 推送失效**；panic 时 writer 不回收。 |
| `internal/services/node_health/node_health.go` | 272-328 | 离线节点置 `is_active=false` 后**永远不被健康检查覆盖**，无法自动恢复。 |
| `internal/services/scheduler/scheduler.go` | 114-156 | 到期提醒 ±1h 窗口配合 24h ticker，**7/3/1 天提醒系统性漏发**。 |
| `internal/services/order/order.go` | 1431-1515,1517-1628 | 退款无事务；按"当前订阅"直接减时长，叠加续费后退款错误。 |
| `frontend/src/views/Packages.vue` | 486,1582-1652 | 关闭扫码对话框**不停止轮询**（3 秒空转请求）；paymentStatusTimeoutId 从不清理。 |
| `frontend/src/views/admin/ConfigUpdate.vue` | 483-512,682-688 | 组件卸载后轮询仍续订。 |
| `frontend/src/components/UpgradeDevicesDrawer.vue` | 445-464 | 嵌套 setTimeout 覆盖用户支付方式选择，可提交空 payment_method。 |
| `frontend/src/views/admin/Packages.vue` | 833-865 | 循环内串行 await 6 次请求，非原子易部分失败。 |
| `frontend/src/views/admin/Settings.vue` | 40-49,1088 | el-upload 上传 Logo 不带 Authorization，**必然 401**。 |
| `frontend/src/views/admin/Users.vue` | 1448-1457 | "分配专线节点"下拉永远为空（loadAvailableNodes 未调用）。 |
| `frontend/src/views/admin/components/UserDetailDialog.vue` | 1031-1054 | 抽屉打开时切换用户不重置状态，展示上一个用户数据。 |

### 3.3 性能

| 文件 | 行 | 问题 |
|------|----|------|
| `internal/api/handlers/node.go` | 113-165 | findNodeIDsByKey 全表加载 + 循环调用 = **O(N²) 全表扫描**。 |
| `internal/api/handlers/dashboard.go` | 299-436,652-719 | GetAbnormalUsers 多轮 GROUP BY 全表聚合 + 内存分页。 |
| `internal/api/handlers/statistics.go` | 339-466 | GetRegionStats 全表装载 audit_logs+user_activities 内存聚合 + 逐行 GeoIP。 |
| `internal/api/handlers/custom_node.go` | 404-777 | Import/BatchAssign 逐条 Create + 无事务，N+1 写入。 |
| `internal/services/device/device_manager.go` | 42-643 | UA 解析热路径内 `regexp.MustCompile` 每次调用重新编译。 |
| `internal/services/notification/notification.go` | 397-515 | `doJSONPost` 无超时默认 http.Client 且被 goroutine 调用 → 永久泄漏。 |
| `internal/services/geoip/cache.go` | 71-82 | WarmupCache 每个 IP 一个 goroutine 形成风暴。 |
| `internal/services/config_update/config_update.go` | 1267,2355 | 每次订阅请求额外 2 次 DB 查询；缓存代际导致全量失效（cache.go:119-141）。 |
| `internal/api/handlers/user.go` | 543-591 | GetUserDetails 每订阅 4 次 SystemConfig 查询（N+1）。 |
| `internal/api/handlers/auth.go` | 47-55 | getMinPasswordLength 每次请求查库。 |
| `frontend/src/utils/textSelection.js` | 53-73 | MutationObserver 监听整个 body 子树，全文档扫描。 |
| `frontend/src/views/Knowledge.vue` | 166-179 | 文章列表无分页全量拉取渲染。 |

### 3.4 模型 / 数据层

| 文件 | 行 | 问题 |
|------|----|------|
| `internal/models/payment.go` | 13-14 | **Amount 单位是"分"，Order.Amount 是"元"**，金钱核心单位不一致（全项目最高技术债）。 |
| `internal/models/order.go` | 13,22-23 | 金额 float64 decimal(10,2) + DiscountAmount default:0 使 NULL 语义失效。 |
| `internal/models/invite.go` | 33-45 | invite_relations 无 InviteeID 唯一约束 → 并发注册重复发奖。 |
| `internal/models/checkin.go` | 5-10 | 无数据库级防重复约束，同日重复签到依赖服务层。 |
| `internal/models/token_blacklist.go` | 35-45 | 黑名单判定对 DB 错误 **fail-open**（放行已登出令牌）。 |
| `internal/models/payment_config.go` | 12-25 | 支付密钥字段带 json tag 无 `json:"-"`，管理端原样回传。 |
| `internal/models/audit_log.go` | 20-23 | RequestParams 原样落库，密码/令牌明文进审计表。 |
| `internal/models/config.go` | 8-18 | SystemConfig.Value 明文存储密钥类配置。 |
| `internal/models/user.go` | 36,59 | BarkDeviceKey/Notes 暴露在 JSON 序列化。 |

### 3.5 前端复用/风格重点（详见第 5 节）

7 个日志页 ~80% 同构复制、5 个教程组件 95% 相同（1400 行可数据化约 1000 行）、AdminLayout/UserLayout 70% 重复、api.js 中 nodeAPI 与 adminAPI 重复 9 个方法、button-common 网格规则复制 6 次、list-common `:has(:nth-child)` 复制 5 类、user-client-polish 12 类 × 4 变体复制、formatDate 在 8+ 文件重复实现、状态映射与 statusMaps.js 多文件冲突。

---

## 4. 🟡 Medium 问题摘要（383 个）

- **认证链路**：`auth.go` 注册验证码在事务前标记已用；邀请奖励 goroutine 与 `db.Save(user)` 余额覆盖竞态；邀请码只校验非空不校验有效性（"需要邀请码"配置形同虚设）；登出不吊销 refresh token（黑名单写入错误被忽略）；用错误字符串匹配判断唯一约束（PostgreSQL 下返回 500）。
- **支付**：`payment.go:348-372` 只置最新 pending 交易成功，旧交易永久滞留；`alipay.go` 三个解析方法重叠、DecodeNotification 不验签易误用；`PaymentNotify` 未认证端点多轮全量 INFO 日志（日志洪水）；`yipay.go isLocalDomain` 子串匹配误伤生产域名；`wechat.go VerifyNotify` 就地 delete(params) 修改调用方 map。
- **一致性**：`order.go` GetOrder 返回原始模型 vs 列表 formatOrderData 形状分裂；refund 无幂等保护可二次退款；ExportOrders CSV 公式注入；`recharge.go` 同一资源三套响应契约；`invite.go`/`knowledge.go` sql.Null* 原样序列化泄漏后端细节；`GetAdminSettings`/`GetAdminEmailConfig` 明文返回 SMTP 密码/API Token。
- **配置**：`config.go` 生产校验只检查 SECRET_KEY 却强制同时要求 MySQL+Postgres 密码（与所选数据库无关）；`getInt` 把 0 当未设置；`JWT_ALGORITHM` 配置从未被使用；redisDB 硬编码 0。
- **数据库迁移**：database.go 迁移中 Raw/Scan/DDL 普遍不检查错误；`HasColumn` 用 Go 字段名 `FulfilledAt` 探测（MySQL/PG 恒 false）；NullInt64(0) 无法表达 NULL。
- **缓存**：`user_cache.go` 整文件零调用方（死代码）；`cache_service.go:266` ClearKnowledgeArticlesCache 清不掉按分类键；`GetSessionTimeout` 每次签发 token 2 次 DB 查询。
- **前端**：login 硬编码 `remember=true` 忽略记住我选项；theme.js/settings.js 双主题引擎键集互斥（'default' vs 'light'）；Orders 用 `new Date('YYYY-MM-DD HH:mm:ss')` Safari Invalid Date；AbnormalUsers 的 toISOString UTC 日期偏一天；Analytics CSV 未转义（公式注入）；Statistics Chart.js 无卸载清理（内存泄漏）；UserLevels 浏览器代码直接访问 `process.env.NODE_ENV`（Vite 未 define → ReferenceError）；用户端 Dashboard 设备数用 `||` 而非 `??`（0 被覆盖）。
- **服务层**：`promotion.go` 重复参与检查在事务外且无唯一约束（并发可重复领奖）；`scheduler` 自动备份含密钥；`email/templates.go` 邮件模板 `template.HTML` 直插用户数据（XSS）；`discount/coupon.go` 每用户使用次数 Count 错误被忽略（绕过限额）；`geoip.go` 两个解析函数高度重复且依赖第三方网页结构。

---

## 5. 风格与格式不一致汇总（重点：前端）

### 5.1 同一项目内重复实现且行为分叉的公共工具

| 工具 | 出现位置 | 差异 |
|------|---------|------|
| `formatDate` | AbnormalUsers/Tickets/Analytics/Profile/Knowledge/Promotions/EmailQueue/EmailDetail/Invites/UserDetailDialog 等 **10+ 处** | 有的用 toLocaleString('zh-CN')、有的用 dayjs、有的 `YYYY-MM-DD HH:mm:ss`，时区口径不一 |
| 订单/节点状态映射 | Dashboard.vue、Nodes.vue、Orders.vue、statusMaps.js | 文案冲突：inactive=待激活 vs 未激活；'completed' 订单状态缺失；颜色与 statusMaps.js 相悖 |
| 支付方式解包 | Orders.vue 4 处 | GORM NullString 解包复制 4 份 |
| 密码策略 | UserSettings.vue vs UnifiedAuth.vue | 两套规则不一致（长度/复杂度要求不同） |
| 用户数据映射 | Profile.vue 4 份 | 同一映射逻辑复制 4 遍，已开始分叉 |
| 登录历史 | Profile.vue 弹窗 vs LoginHistory.vue 页面 | 双实现，行为已分叉（'解析中...' vs '本地/内网'） |
| 主题引擎 | store/theme.js vs store/settings.js | 键集互斥：'default' vs 'light' |
| 分页参数 | admin 各页 | `size` vs `page_size` 混用；响应容器 users/tickets/list/logs/levels 各异 |

### 5.2 脚本/组件风格割裂

- **Options API vs `<script setup>`**：ConfigUpdate.vue、Profile.vue 用 `export default { setup() }`，其余视图全部 `<script setup>`。
- **缩进**：Statistics.vue 模板与 script 混用 tab/空格；Orders.vue 同一函数内 2/4/6 空格混用；Nodes.vue/Config.vue 缩进错乱。
- **时间处理**：Coupons.vue 硬编码 `Asia/Shanghai` 往返转换（非东八区管理员双偏）；Promotions.vue 新建用 toISOString 而 value-format 是 'YYYY-MM-DD HH:mm:ss'；CustomNodes.vue 用 toLocaleString 与全局 dayjs 不一致。
- **API 路径**：同一业务多条路径（`/devices` vs `/subscriptions/devices`、`/orders` vs `/orders/`、`/auth/login` vs `/auth/login-json`、`getUserOrders` vs `getOrderList`）；`/orders/` 尾斜杠每次触发 301。
- **错误处理风格**：GenerateRandomString 失败 panic，其余生成函数回退时间戳；`log.Printf` vs `utils.LogError` 混用（package.go:110）；handlers 内 4 种取用户/回错误写法。
- **命名**：camelCase 与 snake_case 混出（dashboard.go 响应同时输出两套字段；Invites.vue statistics 用 snake_case 键）；`Protocol` vs `NodeConfig.Type` 双份协议来源；`clash` vs `vless` 协议命名不一。
- **Go 错误处理**：err.Error() 字符串比较判断业务错误（checkin.go、auth.go 唯一约束）；错误文本直接外泄前端（order.go:818,1829,2189）。
- **注释**：`#nosec G117` 引用不存在的 gosec 规则（node.go、cache.go）——无效注释；中文/英文注释混用。

### 5.3 CSS 体系（最严重的风格问题）

- `user-client-polish.scss` 2235 行 god 文件 + **数百处 !important 特异性军备竞赛**，后段 "final pass" 静默覆盖前段 → 前段规则成死代码。
- 全局样式被编译 3 份：`global.scss` 经 main.js + UserLayout + AdminLayout 三处独立引入，打包膨胀约 3 倍。
- 4 个样式文件对弹窗宽度（92%/88%/94vw/min(420px,...)）、按钮布局（column vs column-reverse）、分页/空态等同一批类名互相覆盖，**胜负由 @use/import 顺序决定**。
- `:has(:nth-child)` 移动端按钮网格模式全库复制 ~40 处。
- `text-selection.css` 全局 `* { user-select: auto !important }` 破坏依赖 user-select:none 的组件。
- Dashboard.vue/Help.vue 文件尾部 300-1130 行 `!important` 覆盖瀑布。
- 主题变量三套命名并行、十六进制大小写混用、`-webkit-overflow-scrolling: touch` 废弃属性残留。

---

## 6. 可复用 / 精简建议（Top 优先级）

### 后端
1. **抽取公共"取当前用户"与响应辅助**：handlers 包内 4 种写法统一为 `getCurrentUser(c)` + 统一错误返回。
2. **日志/审计四份近似实现合一**：`internal/utils/audit.go` 的 CreateBusinessLog/Fast/Async/SimpleFast 抽一个带选项的公共函数；日志函数统一注入 `*gorm.DB`（消除全局 DB 依赖）。
3. **订单号生成器重构**：`findMax*Sequence`/`check*Exists` 四函数以表名参数合一；删除死参数 `userID` 与未使用的 `getTableName`；加唯一索引 + 乐观重试消除 TOCTOU。
4. **通知 builder 合一**：`notification/template.go` Telegram 与 Bark 两套 builder ~700 行重复 → 抽公共模板渲染层。
5. **PEM 工具合一**：`FormatPEMKey`/`FormatPEMPublicKey` 删除后者；`NormalizePrivateKey` 改用真实 x509 解析判定类型。
6. **GeoIP 解析器合一**：`GetLocationFromIPW`/`GetLocationFromPing0` 合并，减少对第三方网页结构的强依赖。
7. **AuthMiddleware/TryAuthMiddleware 抽公共 `resolveUser(c, token)`**，两中间件只定义失败策略。
8. **BatchEnable/BatchDisableUsers、applyAuditLogFilters/applySystemLogsFilters 参数化合并**。
9. **删除死代码**：`user_cache.go` 整文件、`RateLimitMiddleware`+`generalRateLimiter`、`Algorithm` 配置、`CSRFExemptMiddleware`、`PaymentConfig.GetConfig()`、`CleanOldStatuses`、`CleanExpiredTokens` 的调用缺口、`transport_opts.go`（若无引用）、`CreateOrderRequest` 结构体。
10. **金钱单位统一**：订单/充值/余额全部改为**整数分**存储，消除 float64 精度与"分/元"混用。

### 前端
1. **7 个日志页合一个配置驱动组件**：把 fetch/debouncedFetch/resetFilter/onSizeChange/paginationLayout/双端筛选抽成 `LogListPage.vue` + 每页只提供列定义与 API。
2. **5 个教程组件数据化**：客户端面板（macOS/iOS/Windows/Android/软件）用一份配置数组渲染，删除 ~1000 行重复模板；并修复"立即下载"按钮不存在、产品名 'Clash Part' vs 'Clash Party' 不一致。
3. **AdminLayout/UserLayout 抽公共布局基座**（头部/侧边栏/移动导航/主题/未读逻辑）。
4. **api.js 去重**：nodeAPI 与 adminAPI 合并（重复 9 个方法）；config-update 端点从 paymentAPI 移到 configUpdateAPI；同一业务收敛为唯一路径。
5. **抽 `usePaymentFlow` composable**：Dashboard/Packages/Orders/UpgradeDevicesDrawer 的支付宝唤起、visibility 监听、支付方式解析、轮询生命周期统一。
6. **statusMaps.js 成为唯一真相源**：所有页面删除本地状态/类型映射，改用共享 map + `getXxxText()`。
7. **formatDate/formatMoney 统一到 utils**：删除 10+ 处本地重复实现。
8. **CSS 重构**：删除 user-client-polish.scss 的 !important 军备竞赛（改为具名组件类）；`:has(:nth-child)` 网格抽一个 `@mixin`；global.scss 只在 main.js 引入一次。
9. **修复假功能**（见 High/Critical）：markAsNormal、EmailDetail 响应解包、Orders 充值分页、keyword 搜索（后端补实现或前端移除）、头像上传、邮箱修改。
10. **前后端契约统一**：后端统一响应信封（`{success,code,message,data}`），前端 api 层做一次响应归一化，页面禁止 3-5 分支"猜响应"。

---

## 7. 部署与配置（必须修正）

| 问题 | 位置 | 说明 |
|------|------|------|
| Go 版本断裂 | go.mod(1.24.0) vs Dockerfile/install.sh/install-vps.sh(1.21.x) | 全部统一到 1.24，否则本地编译与线上产物行为不一致 |
| Dockerfile 构建失败 | Dockerfile | CGO_ENABLED=0 + mattn/go-sqlite3（见 Critical 2.4） |
| SECRET_KEY 默认值公开 | docker-compose.yml:10、start.sh:32 | 未设置者全用同一公开密钥，JWT 可伪造 |
| 后端直连公网 | bt-deploy.sh:208-221、docker-compose 8000 端口 | 绕过 nginx 防护；应仅监听 127.0.0.1 或经反代 |
| pkill -9 无差别杀进程 | bt-deploy.sh:539、install-vps.sh:456、start.sh | 多站点主机上会杀死无关进程 |
| root 运行服务 | install.sh:344 | systemd 服务应以非特权用户运行 |
| 管理员密码明文回显/写 /tmp | install-vps.sh:421-429 | 日志即凭据泄露 |
| 重装删库无备份 | install-vps.sh:246-248 | rm -rf 整个项目目录含 cboard.db |
| .github/SECURITY.md 是 PocketBase 模板残留 | .github/SECURITY.md | 与本项目无关，建议替换 |
| Makefile clean 删生产库 | Makefile:17-19 | `rm -f *.db *.log` 会删生产数据 |
| .goreleaser.yaml 是 PocketBase 示例复制 | .goreleaser.yaml:3-16 | 构建目标 ./examples/base 不存在 |
| migrate/init SQL 仅 SQLite/MySQL 专属语法 | scripts/migrations/*.sql、init_knowledge.sql | 与三库支持矛盾；init_knowledge.sql 开头无条件 DELETE |
| GeoIP 自动下载无超时/大小限制 | main.go:141-171、scripts/download_*.go | 启动挂起或磁盘耗尽 |
| install.sh tee|tail 判断成败 | start.sh:346-393 | 管道退出码恒 0，重试分支永不触发 |

---

## 8. 优化路线图（建议执行顺序）

### 第一阶段：止血（安全与资金，1-3 天）
1. 修复支付安全：ApplePay 验证（C2.1）、删除私钥日志（C2.2）、重置并清理硬编码商户凭据（C2.3）、易支付签名降级（yipay.go:529）、查单验签（query.go）。
2. 修复资金竞态：ConvertSubscriptionToBalance 行锁（C2.6）、订单号 TOCTOU（common.go:185）、promotion 并发领奖、邀请关系唯一索引。
3. 修复用户等级升降级逻辑（C2.5）。
4. 修复备份泄露 .env 与默认推送作者仓库（C2.9）。
5. 修复 IP 伪造绕过限流（network.go + ratelimit.go 改用可信代理链/Redis 限流）。
6. 删除明文密钥回传接口（admin.go 支付配置、email-config、config.go 的 API Token）。

### 第二阶段：修复真实功能缺陷（3-7 天）
7. 前端假功能批量修复：markAsNormal（C2.7）、EmailDetail（C2.8）、Orders 充值分页/假搜索、UserSettings 邮箱/头像、Dashboard 异常客户、CustomNodes 假测试、公告分类 system→announcement。
8. 后端假逻辑修复：TestNode/ImportFromClash 权限、DeleteUser 顺序与残留、UpdateCustomNode 配置覆盖、ResetPasswordByCode 限流、GetTicket 泄露。
9. 修复 Dockerfile 与部署脚本版本/凭据/进程管理问题。

### 第三阶段：一致性与架构（1-2 周）
10. 前后端契约统一（响应信封、分页参数、字段命名），前端 api 层归一化。
11. 抽公共组件/composable：LogListPage、教程数据化、支付流程 composable、布局基座。
12. CSS 体系重构（删除 !important 军备竞赛、全局样式单次引入、:has() 网格抽 mixin）。
13. 后端公共逻辑抽取与死代码清理（见第 6 节后端清单）。
14. 金钱单位统一为整数分 + 全局金额类型重构。

### 第四阶段：性能与可观测（持续）
15. N+1 / O(N²) 查询修复（node.go、dashboard.go、statistics.go、user.go）。
16. 缓存体系修复（代际失效、GetSessionTimeout 缓存、WarmupCache 限流）。
17. 审计日志异步有界队列；限流/CSRF 状态迁 Redis（支持多副本）。
18. 补齐单元测试：等级升降级、订阅兑换、退款、订单号并发。

---

## 9. 亮点与良好实践（值得保留）

- SQL 普遍参数化、列表查询批量格式化避免 N+1（GetUsers/GetAdminTickets）、分页上限 100。
- 生产环境错误响应脱敏（response.go）、审计日志敏感字段 REDACTED（sensitiveFields）。
- 前端 XSS 面控制良好（{{}} 插值 + DOMPurify + 协议白名单 safeOpen/sanitizeHtml）。
- 移动端体验认真（44px 触控目标、safe-area、prefers-reduced-motion、focus-visible、ResponsiveDataView 双端渲染）。
- CSRF 双因子（cookie+header）、401 刷新队列、双角色令牌存储等前端安全设计有深度。
- 支付关键路径已有部分正确实践（payment.go 的 clause.Locking、FulfilledAt 幂等），可推广到订阅兑换等其余路径。
- `config_update_test.go` 对 YAML 1.1 事故值（953e8078 等）的回归测试质量高，值得全项目推广测试范式。

---

*报告生成时间：2026-08-24 · 覆盖 338 个文件 · 完整逐条发现见 [AUDIT_FINDINGS_FULL.md](./AUDIT_FINDINGS_FULL.md)*
