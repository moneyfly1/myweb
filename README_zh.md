# 🚀 CBoard - 现代化订阅管理系统

> **CBoard**（CBoard-Go）是一个为 VPN / 代理服务商设计的高性能、现代化订阅管理系统（机场面板）。
> Go 后端 + Vue 3 前端，内存占用仅 **35 - 95 MB**，提供 **371 个 API 路由**，覆盖从注册、订阅分发、设备管理、支付订单到自建节点运维的**全链路运营能力**。

---

## 📖 目录

- [💡 系统简介与设计理念](#-系统简介与设计理念)
- [✨ 核心特性](#-核心特性)
- [🏗️ 技术栈](#️-技术栈)
- [📋 系统要求](#-系统要求)
- [📊 功能清单](#-功能清单)
- [🔬 核心设计详解](#-核心设计详解)
- [🚀 安装指南](#-安装指南)
- [👤 管理员账户管理](#-管理员账户管理)
- [⚙️ 配置说明](#️-配置说明)
- [💾 数据库备份](#-数据库备份)
- [🔧 故障排查](#-故障排查)
- [📄 许可证](#-许可证)

---

## 💡 系统简介与设计理念

### 项目初衷

CBoard 最初源于一个非常实际的个人需求：**将多个机场的订阅资源安全地分享给朋友，同时防止资源被滥用**。

**典型场景：**

- 🧑‍🤝‍🧑 手头有多个机场订阅、流量用不完，希望分享给外贸 / 出海办公的朋友使用；
- 🎫 购买的机场套餐通常不限制设备数量，但需要**控制分享范围**；
- 🚫 通过**设备数量限制**来防止朋友将订阅再次转发给他人；
- ⚠️ 但仅限制设备还不够——朋友可能把节点信息下载下来单独使用，彻底脱离控制。

**CBoard 的解决方案：**

- 🔄 **定期重置订阅地址**（建议每天或每两天重置一次），让"下载节点单独使用"失去意义；
- 🧬 **自动采集与聚合**多个机场的订阅地址，生成统一的聚合订阅链接；
- 📱 **设备指纹 + 设备数量限制**，杜绝订阅二次扩散；
- 🎯 既能保证资源被有效利用，又通过技术手段防止资源滥用与泄露。

### 设计理念

| 理念 | 说明 |
|------|------|
| ⚡ **高性能** | 采用 Go 语言构建，内存占用仅 35-95 MB（同类 Python 面板通常需要 300-850 MB），毫秒级启动 |
| 🔒 **安全可靠** | JWT 双令牌认证、bcrypt 密码加密、登录限流与暴力破解检测、敏感字段脱敏、CORS/CSRF 防护 |
| 🧩 **功能完整** | 覆盖机场运营全链路：注册认证、订阅分发、设备管控、套餐订单、支付、工单、邀请、营销、统计 |
| 🐳 **易于部署** | 一套一键脚本（纯 VPS / 宝塔通用，自动识别环境）+ Docker Compose，开箱即用 |

---

## ✨ 核心特性

### 核心能力

- 🚀 **极致性能**：内存占用仅 35-95 MB，Go 并发模型 + 异步 goroutine 通知，高并发下依然稳定
- ⚡ **快速启动**：毫秒级冷启动，SQLite 默认零配置即可运行
- 🔒 **企业级安全**：JWT + 刷新令牌 + 黑名单、bcrypt 加密、登录限流、暴力破解检测、敏感字段脱敏、验证码原子消费、CORS/CSRF 防护
- 📦 **功能完整**：371 个 API 路由，覆盖用户端 16 个页面 + 管理端 26 个页面
- 🎨 **现代化前端**：Vue 3 + Element Plus 响应式设计，深色 / 浅色主题，移动端抽屉全屏适配
- 🐳 **多种部署**：一键脚本（`install.sh`，纯 VPS 与宝塔通用；宝塔入口 `bt-deploy.sh`）/ Docker Compose

### 业务能力

- 🌐 **智能订阅分发**：按客户端 UA 自动识别 9 大类客户端并分发对应格式（Clash / Clash Meta / Stash / Surge / Loon / QuantumultX / Sing-box / Shadowrocket / v2rayN）
- 🧠 **协议版本感知**：按客户端版本过滤新协议——老版 Clash 只推 SS/VMess，Clash Meta 与 sing-box 全量推送，Shadowrocket 按构建号判断
- 🖥️ **自建节点（特色）**：SSH 全自动部署 sing-box，支持 15 种协议，30s 心跳 + 3min 超时离线判定，远程管理（重置 UUID / 改密码 / 改端口 / 重装 / 流量配额），证书自动续期
- 💳 **多支付方式**：支付宝、微信支付、易支付、码支付、Apple Pay、**Stripe / PayPal / USDT（国际支付）**
- 📊 **数据分析**：DAU/WAU/MAU 统计、留存分析、流失预警、收入统计、节点健康监控
- 🎫 **运营套件**：工单系统、知识库、邀请返利、优惠券、每日签到、营销活动（限时抢购 / 新用户优惠 / 会员日）
- 🧬 **节点采集**：从订阅地址自动采集节点、Clash 配置导入、节点去重（Type:Server:Port）、分组与批量测速
- 💾 **自动备份**：数据库自动备份并推送至 GitHub / Gitee，升级前只读预检工具

---

## 🏗️ 技术栈

### 后端（Go）

| 组件 | 技术 | 说明 |
|------|------|------|
| 语言 | **Go 1.21+** | 高性能、低内存、天然并发 |
| Web 框架 | [Gin](https://github.com/gin-gonic/gin) | 高性能 HTTP 框架 |
| ORM | [GORM](https://gorm.io/) | 功能强大的 ORM 库 |
| 数据库 | **SQLite**（默认）/ MySQL 5.7+ / PostgreSQL 12+ | 多数据库即插即用 |
| 认证 | JWT（JSON Web Tokens） | 双令牌 + 黑名单 |
| 配置 | [Viper](https://github.com/spf13/viper) | 环境变量 / .env 文件 |
| 缓存 | **Redis（可选）** | 支付回调、订阅缓存、GeoIP 查询加速、任务队列 |

### 前端（Vue 3）

| 组件 | 技术 |
|------|------|
| 框架 | Vue 3（组合式 API） |
| UI 库 | Element Plus |
| 构建工具 | Vite |
| 状态管理 | Pinia |
| 路由 | Vue Router 4 |
| 图表 | ECharts |

---

## 📋 系统要求

### 最低配置

| 资源 | 最低要求 | 推荐 |
|------|----------|------|
| CPU | 1 核 | 2 核+ |
| 内存 | 512 MB | 1 GB+（面板本体仅需 35-95 MB） |
| 磁盘 | 10 GB | 20 GB+ |
| 操作系统 | Ubuntu 18.04+ / Debian 10+ / CentOS 7+ | 任意主流 Linux 发行版 |

### 软件要求

| 组件 | 要求 | 说明 |
|------|------|------|
| Go | 1.21+ | 安装脚本会自动安装 |
| Node.js | 16+ | 仅前端构建需要，安装脚本会自动安装 |
| Nginx | 任意版本 | 宝塔环境用面板的 nginx；纯 VPS 由 `install.sh` 自动安装（http2 写法按版本自适应） |
| 数据库 | SQLite（默认，零配置）或 MySQL / PostgreSQL | 高流量生产建议 MySQL / PostgreSQL |
| Redis | 可选 | 不配置自动禁用缓存，功能不受影响 |

---

## 📊 功能清单

### 🧑‍💻 用户端（16 个页面）

| 页面 | 核心功能 |
|------|----------|
| 📊 Dashboard | 账户概览、订阅状态、流量统计、公告、快捷入口 |
| 🔗 Subscription | 订阅 URL 生成 / 复制 / 二维码、重置订阅、发送订阅邮件、转换余额、多客户端订阅分发 |
| 📱 Devices | 设备列表、设备指纹识别、设备数量限制、添加 / 删除设备、在线设备追踪 |
| 📦 Packages | 套餐展示、购买、续费、升级 |
| 🧾 Orders | 订单创建 / 取消 / 支付 / 历史查询 |
| 🌍 Nodes | 节点列表、分组（按地区）、延迟测速、专线节点展示 |
| ❓ Help | 帮助中心、使用教程、常见问题 |
| 👤 Profile | 个人资料编辑、头像、签名 |
| 🕐 LoginHistory | 登录历史、登录设备、异常登录提醒 |
| 🎫 Tickets | 工单创建 / 回复 / 状态跟踪 / 附件 |
| 🎁 Invites | 邀请码生成、邀请关系、邀请奖励 |
| 📚 Knowledge | 知识库文章浏览、Clash 系列教程 |
| ⚙️ UserSettings | 账户安全（改密 / 邮箱）、通知设置、主题偏好 |
| 💳 PaymentReturn | 支付回调结果页 |
| 🔐 UnifiedAuth | 统一登录 / 注册页（支持邮箱验证码、邀请码注册、找回密码） |
| 🎰 签到 | 每日签到随机奖励（0.1-1 元），提升用户活跃度 |

#### 认证体系

- ✅ 注册 / 登录（用户名或邮箱）
- ✅ JWT 双令牌（访问令牌 + 刷新令牌）与刷新机制
- ✅ 找回密码（邮箱验证码 / 重置链接）
- ✅ 邮箱验证码注册、邀请码注册
- ✅ 登录限流与暴力破解检测

#### 订阅体系

- ✅ 订阅 URL 生成（UUID 订阅标识）
- ✅ 设备管理（设备指纹识别，防多设备共享）
- ✅ 订阅重置（防止节点信息被剥离单独使用）
- ✅ 发送订阅到邮箱（SMTP）
- ✅ 订阅余额转换（流量/余额互转）

#### 多客户端订阅分发

| 客户端 | 分发格式 | 说明 |
|--------|----------|------|
| Clash | YAML | 16 个代理组 + 3376 条分流规则 |
| Clash Meta | YAML | 支持新协议（Reality / Hysteria2 / TUIC 等） |
| Stash | YAML | Apple 生态 Clash 客户端 |
| Surge | 专用配置 | Apple 生态 |
| Loon | 专用配置 | Apple 生态 |
| QuantumultX | 专用配置 | Apple 生态 |
| sing-box | JSON | 新一代全平台客户端 |
| Shadowrocket | 专用配置 | iOS，按构建号判断能力 |
| v2rayN | 专用格式 | Windows 主流客户端 |

> 客户端类型通过 **User-Agent（UA）自动识别**，无需用户手动选择；也支持 `&filter=` 参数做路由过滤。

### 🛠️ 管理端（26 个页面）

| 页面 | 核心功能 |
|------|----------|
| 📊 Dashboard | 全局概览、关键指标、收入 / 用户趋势、**实时动态流**（用户注册/重置订阅/下单/充值等，60s 轮询）、7 天内到期客户 |
| 👥 Users | 用户筛选 / 编辑 / 禁用 / 批量操作、重置密码、**登录为用户**、发送邮件、签到记录 |
| 🚨 AbnormalUsers | 异常用户识别（恶意注册、异常流量、滥用嫌疑） |
| 🌍 Nodes | 节点 CRUD、节点采集、批量导入、批量测速、健康监控、去重（Type:Server:Port）、分组 |
| ⚡ CustomNodes | 专线节点：链接导入 / **订阅导入（URL 拉取解析）** / 手动创建、**订阅更新与更换（增量更新、保留用户分配）**、分配 / 取消分配、到期管理、测速 |
| 🖥️ SelfHostNodes | **自建节点**：SSH 全自动部署 sing-box、15 种协议、心跳维护、远程管理、证书续期 |
| 🔗 Subscriptions | 订阅管理、批量操作、到期提醒、订阅统计 |
| 🧾 Orders | 订单查看 / 处理 / 导出（CSV/Excel）、批量操作、状态追踪 |
| 📦 Packages | 套餐 CRUD、定价、启用 / 停用、显示顺序 |
| 💳 PaymentConfig | 支付宝 / 微信 / 易支付 / 码支付 / Apple Pay / **Stripe / PayPal / USDT** 支付配置 |
| ⚙️ Settings | 系统设置：通用 / 注册 / 通知 / 公告 / 安全 / 主题 / 邀请 / 管理员通知 / 节点健康 / 备份 / 仓库同步 / 协议过滤 / GeoIP |
| 🧩 Config | 高级配置管理 |
| 📈 Statistics | 用户统计、订单统计、收入统计、订阅统计 |
| 📊 Analytics | DAU/WAU/MAU、留存分析、流失预警、地区分析 |
| 📧 EmailQueue | 邮件队列查看、重试、状态管理 |
| 📄 EmailDetail | 邮件详情、模板变量、发送记录 |
| 📜 Logs | 应用日志查看 |
| 🗄️ SystemLogs | 系统日志、审计日志 |
| 🎟️ Coupons | 优惠券 CRUD：折扣券 / 固定金额券、验证、使用追踪、过期管理 |
| 🎫 Tickets | 工单处理、回复、分配、优先级、附件 |
| 🎁 Invites | 邀请码生成、邀请关系、奖励规则 |
| 👑 UserLevels | 用户等级管理（含折扣体系） |
| 📚 Knowledge | 知识库文章 CRUD、分类、教程维护 |
| 🎉 Promotions | 营销活动：限时抢购、新用户优惠、会员日 |
| 👤 Profile | 管理员个人资料 |
| 🔄 ConfigUpdate | 配置热更新 |

#### 用户管理能力

- ✅ 筛选（用户名 / 邮箱 / 状态 / 等级 / 注册时间）
- ✅ 编辑 / 禁用 / 启用 / 批量操作
- ✅ 重置密码
- ✅ **登录为用户**（模拟登录，方便排查用户问题）
- ✅ 发送邮件（验证码、通知）
- ✅ 签到记录查询

#### 节点管理能力

- ✅ 普通节点：采集 / 手动导入 / CRUD / 批量测速 / 健康检查
- ✅ 专线节点：链接导入 / **订阅导入（URL 拉取解析）**、**订阅更新与更换（增量更新、保留用户分配）**、分配与取消分配、独立到期时间
- ✅ 自建节点：SSH 部署、远程管理、心跳监控（详见核心设计详解）

#### 自建节点（CBoard 特色功能）

| 能力 | 说明 |
|------|------|
| 🔑 SSH 全自动部署 | 一键远程安装 sing-box 并生成节点 |
| 🧬 15 种协议 | VLESS+WS、Reality 系列、Hysteria2、TUIC、AnyTLS、SS 等 |
| 💓 心跳维护 | 30s 心跳上报，3min 无心跳判离线 |
| 🔁 多协议共享 UUID | 同一节点多协议共享同一 UUID，客户端配置简单 |
| 🚦 流量配额 | 流量超限自动屏蔽，续费后自动恢复 |
| 🔐 远程管理 | 重置 UUID、改密码、改端口、重装、流量配额调整 |
| 🏅 证书续期 | acme.sh 证书自动续期，无需人工干预 |

---

## 🔬 核心设计详解

### 1️⃣ 订阅分发设计

CBoard 的订阅分发是整个系统的核心，设计上兼顾**兼容性**与**先进性**：

#### UA 自动识别与格式分发

- 系统读取订阅请求的 **User-Agent**，自动识别客户端类型并返回对应格式（Clash YAML / Clash Meta / Stash / Surge / Loon / QuantumultX / sing-box JSON / Shadowrocket / v2rayN）；
- 未知客户端返回通用格式，保证最大兼容性。

#### 按客户端版本过滤新协议

不同客户端对新协议的支持差异巨大，盲目全推会导致老客户端无法解析：

| 客户端 | 协议推送策略 |
|--------|--------------|
| 老版 Clash | 只推 SS / VMess 等经典协议 |
| Clash Meta / sing-box | 全量推送（Reality、Hysteria2、TUIC 等） |
| Shadowrocket | 按构建号判断能力，渐进推送 |
| 其他客户端 | 按协议白名单过滤 |

#### 多层过滤体系

- **exclude 参数过滤**：URL 中指定 `&exclude=协议名` 排除指定协议；
- **DB 协议白名单**：管理端在「协议过滤」中配置允许下发的协议全集；
- **按 IP 地区分发**：结合 GeoIP（GeoLite2-City.mmdb）按用户出口 IP 地区差异化分发节点；
- **&filter= 路由过滤**：订阅链接携带 filter 参数，按节点分组 / 地区筛选下发。

#### 聚合订阅

- 自动采集多个机场的订阅地址，聚合去重（基于 Type:Server:Port）后统一分发；
- 定期重置订阅地址 + 设备数限制，从机制上防止订阅被剥离滥用。

### 2️⃣ 自建节点设计

自建节点（SelfHostNodes）是 CBoard 的特色能力，把「买 VPS 自己搭节点」的运维成本降到最低：

#### 部署流程

1. 管理端录入 VPS 的 SSH 连接信息（IP / 端口 / 用户名 / 密码或密钥）；
2. 系统通过 SSH 上传安装脚本，**自动安装 sing-box** 并生成节点配置；
3. 探测公网 IP，构造节点链接**回传面板**并注册节点；
4. 节点上启动**后台心跳守护进程**，周期性向面板上报状态。

#### 心跳与离线判定

| 参数 | 值 | 说明 |
|------|-----|------|
| 心跳间隔 | 30 秒 | 节点脚本上报间隔 |
| 心跳超时 | 3 分钟 | 超时即判定节点离线 |
| 安装令牌有效期 | 30 分钟 | 防止安装令牌被滥用 |

#### 协议支持（15 种）

VLESS + WS、VLESS + Reality（Reality / Reality Vision / gRPC Reality）、Hysteria2、TUIC、AnyTLS、Shadowsocks 等，同一节点**多协议共享同一 UUID**，客户端侧配置极简。

#### 资源管控

- **流量配额**：节点流量超限自动屏蔽，防止超卖与滥用；
- **证书自动续期**：内置 acme.sh 集成，证书到期自动续签；
- **远程管理**：重置 UUID、修改密码、修改端口、重装系统，全部后台一键完成。

### 3️⃣ 安全设计

| 安全机制 | 实现 |
|----------|------|
| 🔑 认证 | JWT 访问令牌 + 刷新令牌，刷新令牌支持黑名单吊销 |
| 🔐 密码 | bcrypt 加盐哈希，不存明文 |
| 🚦 登录限流 | 基于 IP 的速率限制器，连续失败锁定（默认 15 分钟） |
| 🛡️ 暴力破解检测 | 失败次数累计，自动锁定账户 / IP |
| 🙈 敏感字段脱敏 | 密码、令牌、支付密钥等敏感字段一律脱敏输出 |
| 🎫 验证码 | 邮箱验证码原子消费，防止重放与并发抢兑 |
| 🌐 CORS/CSRF | 白名单式 CORS 配置（`BACKEND_CORS_ORIGINS`）+ CSRF 防护中间件 |
| 🧹 路径安全 | GeoIP 路径、上传路径做路径遍历防护（`safePathJoin`） |
| 🪵 可信代理 | `TRUSTED_PROXIES` 配置，确保真实客户端 IP 获取正确 |

#### 数据库保护

- 启动时若检测到 SQLite 文件不存在（即将创建全新库），会**大声告警**，提示检查 `DATABASE_URL` 路径，避免"重启后数据消失"的误操作；
- 管理员账户每次启动时校验 `ADMIN_PASSWORD`，确保固定密码可用、账户始终处于激活状态（即使被锁定，重启后也能登录）。

### 4️⃣ 性能设计

| 设计 | 说明 |
|------|------|
| 💾 低内存 | Go 运行时 + 单体架构，内存占用 35-95 MB |
| 🧠 Redis 缓存（可选） | 支付回调、订阅数据、GeoIP 查询缓存加速；不配置时自动降级为直查，功能无损 |
| 🔄 任务队列 | 基于 Redis 的任务队列 + worker，异步消费耗时任务 |
| ⚡ 异步通知 | goroutine 异步发送邮件 / Telegram / Bark 通知，不阻塞主流程 |
| 🗄️ 数据库索引 | 提供性能索引 SQL（`docs/sql/performance_indexes.sql`），高流量场景可手动启用 |
| 📊 调度器 | 定时任务（签到结算、节点健康检查、订阅重置、备份）由调度器统一管理，可 `DISABLE_SCHEDULE_TASKS` 关闭 |

---

## 🚀 安装指南

CBoard 提供 **三种部署方式**，按环境选择：

| 方式 | 适用场景 | 安装工具 | 验证状态 |
|------|----------|----------|----------|
| 🖥️ **方式一：一键脚本**（推荐） | 纯 VPS（也兼容宝塔）——自动识别环境、自动装齐 Go/Node/Nginx/证书 | `install.sh`；宝塔入口 `bt-deploy.sh` | ✅ **已在生产实测**：3 台服务器跑通「全自动部署 / 同步升级 / 自检修复」全路径，退出码 0 |
| 🧱 **方式二：宝塔面板建站 + 脚本** | 想在面板里统一管理站点、证书与日志 | 面板「添加站点」+ `bt-deploy.sh` | ✅ **已在生产实测**（宝塔 nginx 1.28，合并模式注入配置） |
| 🐳 **方式三：Docker**（可选） | 需要隔离/可复现部署，且机器拉镜像顺畅 | `docker compose up -d --build` | ⚠️ **构建链已修好、部分验证**——详见该章节「实测说明」 |

> 只有两个脚本需要关心：**`install.sh`（唯一实现）** 与 **`bt-deploy.sh`（宝塔入口，薄封装，行为等同 install.sh）**。（历史上的 `install-vps.sh` 已删除：它固定 Go 1.21.5 / Node 18，与 `go.mod` 和 vite 7 的要求冲突）

---

### 🖥️ 方式一：一键脚本部署（`install.sh`，纯 VPS 与宝塔通用）— 推荐

**这是本项目唯一维护的安装/运维脚本。**它会自动识别环境：装了宝塔就用宝塔的 nginx 与站点目录，没装就装系统 nginx；干净机器上会自己把 git/gcc/Go/Node/Nginx/certbot 装齐。

#### 三个脚本到底用哪个？

| 脚本 | 状态 | 用它做什么 |
|------|------|-----------|
| **`install.sh`** | ✅ **唯一实现，就用它** | 部署 + 运维 + 自检修复 + 完全卸载，纯 VPS 与宝塔都适用 |
| **`bt-deploy.sh`** | ✅ 宝塔入口（薄封装） | 只做宝塔环境检查与证书策略提示，然后转交 `install.sh`，**行为与 install.sh 完全一致** |

> 为什么 `bt-deploy.sh` 只剩一层封装：它以前是 `install.sh` 的完整拷贝，两份代码各自演进后开始漂移（宝塔版曾缺 CGO/Node 版本校验/`nginx -t`/`/uploads` 反代/ACME 放行段，而 install.sh 曾不会自举 Go/Node/Nginx）。现在只保留一套实现，两个入口不会再有差异。

#### 前置条件

| 项 | 要求 |
|----|------|
| 系统 | Debian 10+ / Ubuntu 18.04+ / CentOS 7+ / Rocky / AlmaLinux（实测 Debian 12 全新机器全程通过） |
| 权限 | root（脚本会装软件、写 `/etc/systemd`、写 nginx 配置） |
| 配置 | ≥1 核、≥1 GB 内存（默认编译前端；1 GB 也能跑，2 GB 更稳）、≥10 GB 磁盘 |
| 域名 | 已解析到本机 IP（A 记录）。**目录名必须等于域名**，例如 `/www/wwwroot/pingzen.top` |
| 端口 | 80、443 放行（云厂商安全组 + 系统防火墙都要放；脚本不代改防火墙） |

#### 快速开始（三条命令）

```bash
# 1) 克隆到「域名同名目录」（脚本用目录名当域名）
git clone https://github.com/moneyfly1/myweb.git /www/wwwroot/你的域名
cd /www/wwwroot/你的域名

# 2) 运行脚本（root）
sudo bash install.sh

# 3) 在菜单里输入 1（一键全自动部署），然后按提示回答 Redis 相关提问
```

非交互/自动化场景（脚本对 EOF 安全，不会空转）：

```bash
printf '1\n' | sudo bash install.sh     # 1=全自动部署
```

> **部署过程不会因 Redis 提问卡住**：Redis 配置在部署流程中是「自动模式」——
> 非交互环境直接跳过，交互环境最多等 20 秒后按"跳过"继续；
> 想启用随时用**菜单 12**（GeoIP 查询可提速 50–100 倍）——菜单 12 会自己把 Redis 装好：
> **Docker 方式**（机器上没有 Docker 时脚本会自动安装 Docker）或**系统包方式**（apt/yum）；
> 其中一种装不上、起不来，脚本会**自动改用另一种**，绝不停下（实测案例：某机器 apt 装的
> `redis-server` 报 `libjemalloc.so.2: failed to map segment` 起不来，脚本自动切 Docker 并跑通）。
> Redis 只监听 `127.0.0.1`。

#### 脚本自动做了什么

| 阶段 | 内容 |
|------|------|
| ① 自举依赖 | 基础包（git/sqlite3/wget/curl）→ gcc（SQLite 的 CGO 必需）→ **Go 1.25.0** → **Node 22.12.0** → **Nginx**（已装宝塔则直接用宝塔 nginx） |
| ② 源码与环境 | 源码缺失时自动 `git clone`；生成 `.env`（`HOST=127.0.0.1`、数据库**绝对路径**、`SECRET_KEY` 随机 64 位、权限 600），并**自动生成管理员账号密码写进 `.env`**（`ADMIN_USERNAME`/`ADMIN_EMAIL`/`ADMIN_PASSWORD`） |
| ③ 构建 | 后端 `CGO_ENABLED=1` 编译到 `server.new` → 校验 → 替换（旧版存为 `server.bak.<时间戳>` 供回滚）；前端按 lockfile 安装依赖并 `vite build` |
| ④ 服务 | 写 systemd 单元（`EnvironmentFile`/`LimitNOFILE`/`NoNewPrivileges`）→ 启服务 → **业务健康检查**（`/health`，不是只看 `is-active`） |
| ⑤ Nginx | 写入统一点站模板：`/api/`、`/uploads/`（附件反代，避免被 SPA fallback 吞成 HTML）、`/repo-sync/`、`/assets/` 长缓存、`index.html` 不缓存、`client_max_body_size 16m`、ACME 放行段；写完先 `nginx -t`，失败自动换 http2 写法或**精确回滚本次修改** |
| ⑥ 证书 | 复用优先（certbot → 宝塔 → acme.sh），没有才申请；配置续期重载钩子与定时任务 |
| ⑦ 收尾 | 日志轮转（logrotate）、升级前自动备份数据库到 `/www/backup/cboard`，并打印**登录地址 + 管理员账号密码**（含一次真实登录验证） |

#### 菜单说明（0–16）

| 项 | 功能 | 说明 |
|----|------|------|
| **1** | 一键全自动部署 | 首次安装走这个；重复执行是安全的（幂等，会先备份数据库与旧二进制） |
| 2 | 创建/重置管理员账号 | 密码不回显、留空自动生成强随机密码；**建完会自动用该口令打一次登录接口实测**，确认真的能登进去 |
| 3 | 强制重启服务 | 只清理本项目进程（`^<项目目录>/server`），**不会**误杀 nginx/mysql/redis |
| 4 | 深度清理缓存 | 清 Redis 本项目键前缀 + 清空 `server.log` + 清 Go 编译缓存；**不删** `frontend/dist` 与 `server`（历史上删了会导致站点白屏且重启失败） |
| 5 | 解锁用户账户 | 支持用户名或邮箱 |
| 6 | 查看服务状态 | `systemctl status cboard` |
| 7 | 查看实时日志 | `tail -f server.log`（单元把日志写文件，`journalctl` 里只有 systemd 启停两行） |
| 8 | 标准重启服务 | 重启 + 健康检查 |
| 9 | 停止服务 | |
| 10 | 证书续期 | 手动续期（自动续期由 certbot.timer 或宝塔面板负责） |
| **11** | 从 GitHub 同步并重建 | 升级入口：`git fetch`（失败会报错而不是假装"已是最新"）→ 备份数据库 → 原子构建 → 重启 + 健康检查 |
| **12** | 配置 Redis 缓存 | 可选；只监听 `127.0.0.1`，清缓存按本项目键前缀删除而不是 `FLUSHDB`。选 Docker 方式时会**自动安装 Docker**（装不上自动降级系统包安装，反之亦然），重复执行幂等 |
| **13** | 回滚到升级前版本 | 用 `server.bak.*` 回滚二进制；数据库备份在 `/www/backup/cboard/pre-upgrade-*.db.gz` |
| 14 | 只重新构建并重启 | 不动 nginx/unit，适合"只想重编译" |
| **15** | 完全卸载 | 删服务/站点配置/宝塔扩展目录/logrotate/续期钩子与 cron（配置副本存 `/root/cboard-uninstall-<时间戳>/`），项目目录/数据库/软件包分别询问，最后做**残留扫描** |
| **16** | 自检并自动修复 | 见下方「自检自动修复」 |
| 0 | 退出 | |

#### 环境变量开关

| 变量 | 默认 | 作用 |
|------|------|------|
| `CERT_MANAGER` | `auto` | 证书归属：`auto` 复用已有证书、没有则 certbot 签发；`panel` 交给宝塔面板（脚本不主动签，面板申请后重跑菜单 1/16 自动接入 HTTPS）；`certbot` 强制 certbot |
| `DOMAIN` / `PROJECT_DIR` | 目录名 / 脚本所在目录 | 覆盖域名与项目目录 |
| `GO_VERSION` / `NODE_VERSION` | `1.25.0` / `22` | 自举安装的版本 |
| `DB_BACKUP_DIR` / `DB_BACKUP_KEEP` | `/www/backup/cboard` / `10` | 升级前数据库备份位置与保留份数 |
| `LETSENCRYPT_LIVE_DIR` | `/etc/letsencrypt/live` | certbot 证书目录 |

#### 证书：和宝塔面板的 SSL 会冲突吗？

**会，但脚本已经规避。**冲突点与处理：

| 冲突 | 脚本处理 |
|------|---------|
| 两个 ACME 客户端给同一域名重复签发（面板/acme.sh 与 certbot），更容易触发 Let's Encrypt 重复证书速率限制 | `find_cert_dir` 按 **certbot → 宝塔 `/www/server/panel/vhost/cert/<域名>` → acme.sh `/root/.acme.sh/<域名>`** 顺序探测；只要宝塔已有证书就**复用并跳过 certbot**，日志会写明"已跳过 certbot 申请" |
| 证书文件名不同（acme.sh 是 `fullchain.cer`，certbot/宝塔是 `fullchain.pem`） | 按探测来源渲染文件名，不硬编码 |
| 面板"保存设置/续签 SSL"会重写站点配置，冲掉脚本加的片段 | 重跑**菜单 16** 自动补回缺的片段，并做"有证书就必须是 HTTPS"的状态对齐 |
| 面板管理证书时脚本又塞一套 certbot 定时任务 | 自检只在**证书来源是 certbot** 时才补 `certbot.timer`/续期钩子 |
| 两套证书并存 | `cert_conflict_check` 明确告警，并指出站点当前实际用的是哪一套 |

**结论**：想省事就用面板管证书（`CERT_MANAGER=panel`，或在面板「网站 → SSL」申请后重跑脚本）；用脚本管证书（默认）就**别在面板点"申请/续签"**，只查看。

#### 自检自动修复（菜单 16）

用于"部署/运维中出了问题，让脚本自己修"，会检查并在可能时自动修复：

`.env` 关键项 → 数据库是否存在（**只报告不自动新建**，避免修出空库）→ 必要目录（uploads/ACME webroot/站点目录）→ 后端二进制与前端产物（缺则重建）→ systemd 单元（缺或丢 `EnvironmentFile` 则重写）→ 站点配置必需片段 → `nginx -t`（不过就换 http2 写法重试）→ nginx 进程 → logrotate → 证书与续期钩子/定时任务 → 目标状态对齐（有证书就得是 HTTPS）→ 服务健康（不健康则备份库 → 重建 → 重启）

最后打印「已自动修复 N 项 / 需要人工处理 N 项」。**真机演练**：故意破坏 7 处（删配置片段、写错 http2、删 unit 的 EnvironmentFile、停 nginx、删 logrotate 与续期钩子、删前端产物、把 HTTPS 退回 HTTP），只跑菜单 16 → 7 项全部自动修复，外部 HTTPS 恢复 200。

#### 安装完成后会直接显示登录信息

菜单 1 跑完会打印（菜单 2 建号后、菜单 8 重启后同样会打印）：

```text
================== 登录信息 ==================
  前台地址:     https://你的域名
  前台登录:     https://你的域名/login
  管理后台:     https://你的域名/admin
  管理员登录:   https://你的域名/admin/login
----------------------------------------------
  管理员账号:   admin
  管理员邮箱:   admin@你的域名
  管理员密码:   <脚本生成的 16 位强随机密码>
  （密码由 .env 的 ADMIN_PASSWORD 固定：应用每次启动都会按它重置，重启后依然可用；
    如需修改：改 .env 后菜单 8 重启，或用菜单 2 重置）
==============================================
✅ 已实测：用上面这个账号密码登录成功
```

**密码为什么"永远能显示"**：脚本把生成的密码写进 `.env` 的 `ADMIN_PASSWORD`，
而应用每次启动都会把该管理员的密码重置成这个值（即使账号被锁定，重启后也能登录）。
对比：应用自带的默认行为是"首次启动生成随机密码、只在 `server.log` 打印一次"，日志被清理/轮转后就找不回来了。

> 想换成自己的密码：编辑 `.env` 的 `ADMIN_PASSWORD=你的强密码` → 菜单 8 重启；或用**菜单 2** 交互式重置（会当场实测登录）。

#### 验证安装结果

```bash
curl -I https://你的域名                      # 200 + 证书有效
curl https://你的域名/api/v1/packages         # API 反代正常
curl http://127.0.0.1:8000/health             # 后端健康（本机）
systemctl status cboard                       # 服务运行中
tail -f /www/wwwroot/你的域名/server.log       # 应用日志
```

#### 故障排查

| 现象 | 处理 |
|------|------|
| 服务起不来 | `tail -n 100 server.log`；再跑**菜单 16** 自检修复 |
| 页面白屏 / 还是旧版本 | 浏览器缓存：配置里 `index.html` 已 no-cache；仍异常就 `Ctrl+Shift+R`，并跑菜单 16（会补齐配置片段） |
| 附件/图片打不开 | 检查配置是否含 `location /uploads/`（菜单 16 会自动补） |
| HTTPS 没生效但证书已签 | 多半是 nginx 版本与 http2 写法不匹配，菜单 16 会自动换写法修复 |
| 上传大文件 413 | 配置里 `client_max_body_size 16m`；要更大就改配置或调 `.env` 的 `MAX_FILE_SIZE` |
| 域名打不开 | 先确认 DNS 指向本机、80/443 安全组已放行 |
| 想彻底重来 | 菜单 15 完全卸载 → 重新克隆 → 菜单 1 |

---

### 🧱 方式二：宝塔面板部署（推荐宝塔用户）

**适用**：已装或准备装宝塔面板，希望在面板里管理站点、证书与日志。

> 📖 **完整逐步教程（含宝塔安装、nginx 编译等待、证书两种模式、常见坑）**：[`docs/部署/宝塔部署教程.md`](docs/部署/宝塔部署教程.md)

#### 快速步骤

```bash
# ① 还没有宝塔面板的先装（⚠️ 必须喂 yes：宝塔脚本在 stdin 关闭时会 99% CPU 空转）
wget -O /root/bt_install.sh https://download.bt.cn/install/install_lts.sh && yes | bash /root/bt_install.sh
cat /tmp/btpanel-install.log            # 记下面板地址/账号/密码

# ② 面板 → 软件商店 → 安装 Nginx（源码编译，2 核约 25–40 分钟，属正常）
# ③ 面板 → 网站 → 添加站点：域名填你的域名，根目录保持 /www/wwwroot/你的域名，PHP 选「纯静态」
#    ⚠️ 目录名必须等于域名（脚本用目录名当域名）

# ④ 进站点目录，只下载两个脚本（代码由脚本自己拉，支持非空目录）
cd /www/wwwroot/你的域名
rm -f index.html 404.html .user.ini
curl -fsSLO https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
curl -fsSLO https://raw.githubusercontent.com/moneyfly1/myweb/main/bt-deploy.sh

# ⑤ 一键部署（菜单选 1；没有 Redis 就答 n）；非交互：printf '1\nn\n' | bash bt-deploy.sh
bash bt-deploy.sh
```

#### 证书二选一（别混用）

| 方案 | 做法 | 续期由谁负责 |
|------|------|-------------|
| **A. 面板管（推荐）** | `CERT_MANAGER=panel bash bt-deploy.sh`（把 SSL 交还面板）→ 面板「网站 → SSL → Let's Encrypt」申请 | **面板自动续签**；脚本只复用、不重复签发 |
| **B. 脚本管（默认）** | 直接按上面 ⑤ 跑 | `certbot.timer` 自动续期；此时**别在面板点申请/续签**（面板没有该证书订单），需要时用菜单 10 |

> 为什么 A 要先"交还"：宝塔部署证书前会检查配置里是否已有 `ssl_certificate`，只要存在就认为"已开启 SSL"而跳过写它自己的证书 → 面板申请"成功"却没有证书、后续续签也找不到订单（表现为续签失败）。脚本的 `CERT_MANAGER=panel` 会清掉脚本写的证书段，让面板完整接管。

#### 部署结束会打印登录信息

```text
================== 登录信息 ==================
  前台登录:     https://你的域名/login
  管理员登录:   https://你的域名/admin/login
  管理员账号:   admin
  管理员密码:   <16 位强随机>
==============================================
✅ 已实测：用上面这个账号密码登录成功
```

密码写进 `.env` 的 `ADMIN_PASSWORD`，应用每次启动都会按它重置 → **重启后依然可用**；想换：改 `.env` 后菜单 8，或用菜单 2 重置。

#### 宝塔环境注意事项

| 事项 | 说明 |
|------|------|
| 站点配置归属 | 脚本采用**合并模式**：保留面板的 `#SSL-START`/`#CERT-APPLY-CHECK` 标记与 include，只注入 `/api/`、`/uploads/`、SPA 回退、`index.html` 不缓存、ACME 放行段等必需片段 |
| 面板重写配置后 | 面板「保存设置 / 续签 SSL」会重写 vhost → 跑**菜单 16** 自动补回（实测：注入块与 443 全恢复，面板标记完好） |
| `nginx -t` | 校验的是**正在运行的**宝塔 nginx（`/www/server/nginx/sbin/nginx -t`）；宝塔 nginx 由 `/etc/init.d/nginx` 管理，脚本用 `pgrep` 判运行、不会误启系统 nginx 抢 80 |
| 证书复用 | 面板证书在 `/www/server/panel/vhost/cert/<域名>/`，脚本识别来源为"宝塔面板"并**跳过 certbot**；两套并存会告警 |
| 前端产物权限 | 宝塔 nginx 以 `www` 运行 → 脚本自动把 `frontend/dist`、`uploads`、`.well-known` 设为可读并 chown（否则 403 / ACME 校验失败） |
| 面板站点记录 | 脚本不在面板注册站点；要在面板管理站点就必须按 ③ 在面板建站 |
| 防火墙 | 宝塔「安全」里放行 80/443（脚本不改防火墙） |

#### 安装后管理

| 操作 | 方法 |
|------|------|
| 全套运维 | `bash bt-deploy.sh` 或 `bash install.sh`（菜单 2/3/4/11/13/15/16 等） |
| 升级 | 菜单 **11**（从 GitHub 同步并重建：先备份数据库 → 原子替换二进制 → 重启 + 健康检查） |
| 回滚 | 菜单 **13** |
| 自检修复 | 菜单 **16**（面板重写配置后用它补回片段） |
| 完全卸载 | 菜单 **15**（含残留扫描） |
| 查看日志 | 菜单 7，或 `tail -f /www/wwwroot/你的域名/server.log` |

---

### 🐳 方式三：Docker 部署（可选）

> **实测说明（2026-10-10，干净 Debian 12 + Docker 20.10.24 上真实构建）。**
> 之前仓库里这份 Dockerfile **根本构建不起来**，第 5 步就失败：
> `go: go.mod requires go >= 1.25.0 (running go 1.24.13; GOTOOLCHAIN=local)`
> （基础镜像 `golang:1.24-alpine` 与 `go.mod` 要求的 `go 1.25.0` 不匹配）。共修复 3 处：
> 1. 基础镜像升到 `golang:1.25-alpine`（与 `go.mod` 一致）；
> 2. alpine/musl 下 `go-sqlite3` 必须加 `CGO_CFLAGS="-D_LARGEFILE64_SOURCE"`，否则报
>    `'pread64' undeclared / unknown type name 'off64_t'`；
> 3. 前端基础镜像 `node:20-alpine` → `node:22-alpine`（Vite 7 要求 `^20.19.0 || >=22.12.0`），
>    并新增 `GOPROXY` / `NPM_REGISTRY` 两个 build-arg（慢网络可用国内源）、`HEALTHCHECK`，
>    去掉弱默认管理员密码（`admin123` → 必须显式设置）。
>
**已在容器内验证的项：**

| 验证项 | 结果 |
|---|---|
| 用 Dockerfile 里**完全相同**的命令编译后端（`golang:1.25-alpine` + `apk add gcc musl-dev` + `CGO_CFLAGS=-D_LARGEFILE64_SOURCE`） | ✅ 产出 41MB 二进制 |
| 应用在容器里运行（`HOST=0.0.0.0`，挂载 `./data`、`./uploads`、`frontend/dist`） | ✅ `/health` 200、`/` 200（`<title>CBoard Modern</title>`）、`/admin/login` 200、`/api/v1/packages` 200 |
| 容器内管理员登录 | ✅ `POST /api/v1/auth/login-json` → 200 + `access_token` |
| 重启容器后数据仍在（等价 `./data` 目录挂载） | ✅ SQLite 与 WAL 落在宿主目录，重启后仍可登录 |

**尚未端到端验证**：容器内的**前端构建**与完整 `docker compose up`——验证机拉不动
`node:22-alpine`（Docker Hub 大层停滞、常见镜像站被 Cloudflare 拦）。
在补齐这一步之前，请把 Docker 当**可选项**，优先用**方式一/方式二**（两者均已生产实测）。

> ⚠️ **运行时镜像必须包含 `tzdata`。** 应用会无条件给 SQLite DSN 拼 `_loc=Asia%2FShanghai`
> （`internal/core/database/database.go`），镜像里没有 IANA 时区库就会启动即失败：
> `数据库初始化失败: Invalid _loc: Asia/Shanghai: unknown time zone Asia/Shanghai`（实测复现）。
> 本仓库 Dockerfile 的运行时阶段已安装 `tzdata`；如果你要精简镜像（distroless/scratch），
> 记得把 `/usr/share/zoneinfo` 复制进去。

Docker 部署是**最干净、最可复现**的方式：一条命令构建并启动，数据通过卷持久化，升级只需重新构建镜像。

#### ① 前置条件

| 项 | 要求 |
|----|------|
| Docker | 20.10+ |
| Docker Compose | v2（`docker compose` 子命令） |
| 操作系统 | 任意支持 Docker 的 Linux 发行版 |
| 端口 | 8000（应用端口）需放行 |

```bash
# 验证环境
docker --version
docker compose version
```

#### ② 克隆代码

```bash
git clone https://github.com/moneyfly1/myweb.git cboard
cd cboard
```

#### ③ 配置 .env（关键！）

```bash
cp .env.example .env
vim .env
```

**必须修改的变量（不改会导致启动失败或安全隐患）：**

| 变量 | 必改原因 | 示例 |
|------|---------|------|
| `SECRET_KEY` | ⚠️ 必须改为强随机串！docker-compose 中 `${SECRET_KEY:?}` 未设置会**直接报错拒绝启动**；弱密钥在生产模式也会被拒绝 | `openssl rand -hex 32` 的输出 |
| `ADMIN_PASSWORD` | 首次启动自动创建管理员时使用。不设置则默认 `admin123`（不安全） | 你的强密码（至少 6 位） |

**推荐一并设置（可选但建议）：**

```env
ADMIN_USERNAME=admin                # 可选，覆盖默认用户名
ADMIN_EMAIL=admin@example.com       # 可选，覆盖默认邮箱
SMTP_HOST=smtp.qq.com               # 邮件服务（验证码/通知），可选
SMTP_PORT=587
SMTP_USERNAME=your-email@qq.com
SMTP_PASSWORD=your-smtp-password
PANEL_PUBLIC_URL=https://your-domain.com  # 有自建节点时必配（节点回传地址）
TRUSTED_PROXIES=127.0.0.1,::1             # 部署在 Nginx/Cloudflare 后必配
```

> **不需要修改的变量**：`HOST`（Docker 内必须 0.0.0.0，已默认）、`PORT`（与 compose 映射一致，已默认）、`DATABASE_URL`（默认 SQLite 到挂载目录，已默认）、`DEBUG`（默认 false 生产安全）。

#### ④ 启动服务

```bash
docker compose up -d --build
```

首次启动执行**三阶段构建**（后端 + 前端 + 运行镜像，见下方「Docker 镜像构建说明」），耗时约 1-5 分钟。

验证启动状态：

```bash
docker compose ps          # 查看容器状态（应为 Up）
docker compose logs -f app # 查看启动日志
```

看到以下日志即启动成功：

```
服务器启动在 0.0.0.0:8000
管理员账号已自动创建 / 管理员账号已就绪
```

#### ⑤ 创建管理员

**方式 A：环境变量自动创建（推荐）**

首次启动前在 `.env` 中设置 `ADMIN_PASSWORD`（见第 ③ 步）。容器首次启动检测到全新数据库时，会自动以该密码创建管理员（用户名/邮箱由 `ADMIN_USERNAME`/`ADMIN_EMAIL` 指定，默认 `admin`）。**之后每次重启都会校验该密码**，即使管理员被锁定，重启后也能恢复登录。

若未设置 `ADMIN_PASSWORD`，系统生成**随机密码**并打印在启动日志中：

```bash
docker compose logs app | grep "初始密码"
```

**方式 B：进入容器查看**

```bash
docker compose exec app sh
ls -la /root/data/   # 确认 cboard.db 已生成
```

> 说明：运行镜像为精简 Alpine，不包含 Go 工具链，创建管理员请用环境变量（方式 A），数据保存在挂载目录。

#### ⑥ 访问系统

| 入口 | 地址 |
|------|------|
| 用户前台 | `http://服务器IP:8000` |
| 管理后台 | `http://服务器IP:8000/admin/login` |
| 健康检查 | `http://服务器IP:8000/health` |

> 生产环境建议前置 Nginx / Caddy 反向代理并配置 HTTPS（`80/443` 对外，`8000` 仅内网监听）。

#### 📦 数据持久化说明

Docker 部署的数据**全部保存在宿主机挂载目录**中，删除 / 重建容器不影响数据：

| 卷挂载 | 宿主机路径 | 容器内路径 | 内容 |
|--------|-----------|-----------|------|
| SQLite 数据目录 | `./data` | `/root/data` | 业务数据（cboard.db 及 WAL 日志） |
| 上传目录 | `./uploads` | `/root/uploads` | 头像、附件、备份文件、日志 |

> ⚠️ 挂载**目录**而非单个 `.db` 文件：SQLite 运行时生成 `cboard.db-shm`/`cboard.db-wal`（WAL 模式），且首次启动若宿主机无文件，Docker 单文件挂载会创建**目录**导致启动失败——本配置已改为目录挂载规避此坑。

**备份时只需复制这两个目录：**

```bash
cp -r data /backup/data-$(date +%F)
cp -r uploads /backup/uploads-$(date +%F)
```

#### 🐳 Docker 镜像构建说明（Dockerfile）

三阶段构建（后端编译 + 前端构建 + 运行镜像）：

```dockerfile
# 阶段 1：后端构建（golang:1.24-alpine）
FROM golang:1.24-alpine AS builder
# gcc/musl-dev 满足 SQLite cgo 驱动
RUN apk add --no-cache gcc musl-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o cboard-go cmd/server/main.go

# 阶段 2：前端构建（node:20-alpine，Vite 7 需要 Node 20+）
FROM node:20-alpine AS frontend-builder
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install --legacy-peer-deps
COPY frontend/ .
RUN npm run build

# 阶段 3：运行镜像（alpine）
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /root/
COPY --from=builder /app/cboard-go .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
EXPOSE 8000
CMD ["./cboard-go"]
```

**设计要点：**

- 🏗️ **三阶段构建**：后端编译 + 前端构建分离，运行阶段只拷贝二进制和前端产物，镜像体积最小化；
- ⚡ **前端必须构建**：后端从 `./frontend/dist` 提供静态文件，缺失会导致前端 404（旧版 Dockerfile 的缺陷，已修复）；
- 🟢 **Node 20+**：前端使用 Vite 7，Node 18 会构建失败（已修复）；
- ⏰ **时区**：运行镜像预设 `TZ=Asia/Shanghai`，保证日志与业务时间正确；
- 🔐 **证书**：安装 `ca-certificates`，保证 HTTPS 出站（SMTP、支付回调、GitHub 备份）正常；
- 🗜️ **裁剪**：`-trimpath -ldflags="-s -w"` 去除调试信息，进一步减小体积。

#### 🗄️ 可选：使用 MySQL

默认使用 SQLite（零配置、单文件）。高并发生产环境可切换 MySQL：

**① 编辑 `docker-compose.yml`，取消 MySQL 服务注释并修改 app 的 DATABASE_URL：**

```yaml
services:
  app:
    build: .
    ports:
      - "8000:8000"
    environment:
      - DATABASE_URL=mysql://cboard_user:cboard_password@mysql:3306/cboard_db?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai
      - SECRET_KEY=${SECRET_KEY:?请在 .env 中设置 SECRET_KEY}
    volumes:
      - ./data:/root/data
      - ./uploads:/root/uploads
    depends_on:
      - mysql
    restart: unless-stopped

  mysql:
    image: mysql:8.0
    command: --default-authentication-plugin=mysql_native_password
    environment:
      - MYSQL_ROOT_PASSWORD=rootpassword
      - MYSQL_DATABASE=cboard_db
      - MYSQL_USER=cboard_user
      - MYSQL_PASSWORD=cboard_password
    volumes:
      - mysql_data:/var/lib/mysql
    ports:
      - "3306:3306"

volumes:
  mysql_data:
```

**② 修改 `DATABASE_URL` 为 MySQL 连接串：**

```env
DATABASE_URL=mysql://cboard_user:cboard_password@mysql:3306/cboard_db?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai
```

> 连接串格式：`mysql://用户名:密码@主机:3306/库名?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai`
> 容器间通信使用服务名 `mysql`（compose 内部网络），宿主机访问用 `127.0.0.1:3306`。

**③ 重启：**

```bash
docker compose up -d --build
```

**💡 从 SQLite 迁移到 MySQL：** 项目提供官方迁移脚本（宿主机 Go 环境）：

```bash
go run ./cmd/migrate -sqlite ./data/cboard.db -mysql "cboard_user:cboard_password@tcp(127.0.0.1:3306)/cboard_db?charset=utf8mb4&parseTime=True&loc=Local"
```

> 迁移脚本**只读源库（SQLite）**、只写目标库（MySQL），迁移前请先备份 SQLite 文件。

#### 🔧 Docker 常见问题

| 问题 | 解决方案 |
|------|----------|
| **端口被占用**（`bind: address already in use`） | ① `lsof -i :8000` 找到占用进程；② 杀掉进程或修改 `docker-compose.yml` 的映射端口（如 `"8001:8000"`） |
| **容器启动报 `SECRET_KEY` 未设置** | 在 `.env` 中设置强密钥：`SECRET_KEY=$(openssl rand -hex 32)`，再 `docker compose up -d` |
| **前端页面 404** | 确认镜像已包含前端：`docker compose exec app ls /root/frontend/dist/index.html`；旧镜像需重新 `--build` |
| **容器内文件权限问题**（数据库只读 / 上传失败） | ① 宿主机执行 `chmod -R 755 data uploads`；② 检查挂载目录属主，必要时 `chown -R 1000:1000` |
| **时区不正确** | 运行镜像已内置 `TZ=Asia/Shanghai`；如自定义 Dockerfile，需安装 `tzdata` 并设置 `ENV TZ=Asia/Shanghai` |
| **数据库被"重置"了（数据消失）** | 检查启动目录与 `DATABASE_URL`：SQLite 相对路径基于容器工作目录 `/root/`，确认卷挂载路径与 `DATABASE_URL` 一致（应为 `./data:/root/data`） |
| **构建缓慢 / 拉取依赖失败** | 配置 Go 代理：构建命令前加 `export GOPROXY=https://goproxy.cn,direct`，或修改 Dockerfile 中 `go mod download` 前添加该环境变量；npm 可用 `--registry=https://registry.npmmirror.com` |
| **容器启动即退出，报 `Invalid _loc: Asia/Shanghai: unknown time zone`** | 运行时镜像缺少 IANA 时区库——保留 `tzdata`（或把 `/usr/share/zoneinfo` 拷进镜像）。应用会给 SQLite DSN 拼 `_loc=Asia%2FShanghai` |
| **后端起不来，日志报 Redis 错误** | 无需处理——Redis 连不上会自动禁用缓存并降级运行，功能不受影响 |
| **想改 .env 后生效** | 修改宿主机 `.env` 后执行 `docker compose up -d`（会重新读取环境变量并重建容器） |

---

## 🎯 各部署方式的域名与管理员配置时机

| 部署方式 | 域名在哪配置 | 管理员在哪配置 | 配置时机 |
|---------|-------------|---------------|---------|
| **一键脚本（`install.sh` / `bt-deploy.sh`）** | 由**项目目录名**决定（如 `/www/wwwroot/example.com`），也可用 `DOMAIN=example.com` 覆盖 | 跑脚本后选**菜单 2**（用户名/邮箱/密码交互填写，建完自动实测登录） | 部署后随时可改 |
| **宝塔面板建站 + 脚本部署** | **面板添加站点时**确定域名与站点目录，脚本读取目录名 | 同上（菜单 2） | 部署后随时可改 |
| **Docker** | `.env` 的 `PANEL_PUBLIC_URL`（仅自建节点回传需要；纯订阅无需域名） | `.env` 的 `ADMIN_USERNAME`/`ADMIN_EMAIL`/`ADMIN_PASSWORD`，首次启动自动创建 | 启动前配 `.env` |

**关键说明：**

- **一键脚本**：域名取自项目目录名（`/www/wwwroot/<域名>`），所以克隆时目录名要写对；宝塔环境下会优先使用宝塔的站点目录与 nginx。
- **Docker**：管理员完全通过 `.env` 注入，首次启动自动创建；不设 `ADMIN_PASSWORD` 会生成随机密码并打印在 `server.log`。
- **证书**：脚本默认自己用 certbot 管；若你更习惯面板，设 `CERT_MANAGER=panel`，在面板「网站 → SSL」申请后重跑脚本即可自动接入 HTTPS。

---

## 👤 管理员账户管理

### 创建管理员账户

**系统首次启动自动创建**（三种部署方式通用）：

- 全新数据库启动时自动创建管理员（默认用户名 `admin`、邮箱 `admin@example.com`）；
- 若 `.env` 设置了 `ADMIN_PASSWORD`：使用该固定密码创建（推荐，可预测）；
- 若未设置：生成 **16 位随机密码**并打印在启动日志 `server.log` 的「初始密码」中，**仅显示一次**，请立即保存并登录修改。

**手动创建 / 重置（宿主机 Go 环境）：**

```bash
cd /项目目录
go run scripts/admin_tool                # 交互式创建（默认 admin/admin123，生产勿用）
go run scripts/admin_tool "新密码"        # 直接重置管理员密码

# 生产推荐：环境变量方式
export ADMIN_USERNAME="admin"
export ADMIN_EMAIL="admin@your-domain.com"
export ADMIN_PASSWORD="YourStrongPassword123!"
go run scripts/admin_tool
```

> 若管理员已存在，脚本会更新该账户信息；密码长度至少 6 位。

### 让管理员密码"固定不变"（推荐）

在 `.env` 中设置环境变量，**重新部署 / 换数据库都会使用这个密码**，不再随机：

```env
ADMIN_PASSWORD=你的强密码
# ADMIN_USERNAME=admin
# ADMIN_EMAIL=admin@example.com
```

> 注意：`ADMIN_PASSWORD` 在管理员已存在时每次启动都会校验并重置为该密码，保证固定密码始终可用；同时确保管理员账户始终处于激活 / 已验证状态（被锁定重启后也能登录）。

### 解锁被锁定的账户

账户因多次登录失败被锁定（或 IP 被限流）时：

```bash
# 解锁管理员（用户名或邮箱）
go run scripts/unlock_user admin
go run scripts/unlock_user admin@your-domain.com

# 解锁普通用户
go run scripts/unlock_user user@your-domain.com
```

解锁操作会：清除所有登录失败记录、设置账户为激活状态（`IsActive=true`）、设置账户为已验证状态（`IsVerified=true`）。

> 若仍无法登录，可能是 **IP 被速率限制器锁定**（基于 IP，锁定 15 分钟）：等待 15 分钟、更换 IP（VPN / 移动网络）、或重启服务器清空内存中的限流记录。

### 升级前数据库预检（升级不再"赌"）

担心旧数据库与新代码不匹配导致升级失败？升级前先用**只读预检工具**检查数据库兼容性（不修改任何数据）：

```bash
# 1. 先复制一份生产数据库作为副本（切勿直接指向生产库）
cp /项目目录/cboard.db /root/preflight.db

# 2. 运行预检（指向副本，只读检查）
cd /项目目录
go run ./scripts/db_preflight /root/preflight.db
```

输出会逐项列出：核心表是否齐全、金额单位是否需要分→元迁移（含条数与换算预览）、是否有重复邀请关系、旧版节点表是否将重建、缺失列等，并给出结论：

- ✅ **可直接升级**
- ⚠️ **可升级（升级时自动处理）**
- ❌ **有阻塞问题**

预检通过后建议先在副本上"试跑"新版本确认，再正式升级（升级脚本会自动备份数据库到 `uploads/backups/upgrade_pre_<时间戳>.db`，失败可回滚）。

### 管理员权限一览

- 👥 用户管理：创建 / 编辑 / 删除 / 禁用 / 批量操作 / 重置密码 / 登录为用户 / 发邮件
- 🔗 订阅管理：CRUD / 批量 / 到期提醒
- 🧾 订单管理：查看 / 处理 / 导出
- 📦 套餐管理：CRUD / 定价
- 🌍 节点管理：采集 / 导入 / 测速 / 健康监控 / 自建节点
- 💳 支付配置：支付宝 / 微信 / 易支付 / 码支付 / Apple Pay / Stripe / PayPal / USDT
- ⚙️ 系统配置：通用 / 注册 / 通知 / 公告 / 安全 / 主题 / 备份 / 协议过滤 / GeoIP
- 📈 统计监控：数据统计 / 地区分析 / 用户分析
- 🎫 工单管理：处理 / 回复 / 分配
- 📱 设备管理：查看 / 限制管理
- 🎁 邀请码管理：生成 / 管理
- 📜 日志管理：系统日志 / 登录历史 / 操作日志

---

## ⚙️ 配置说明

### 环境变量总表

主配置文件：`.env`（Viper 加载，环境变量优先级更高）。

| 变量 | 必填 | 默认值 | 说明 |
|------|:----:|--------|------|
| `HOST` | 否 | `127.0.0.1` | 监听地址；Docker 内必须为 `0.0.0.0` |
| `PORT` | 否 | `8000` | 服务端口 |
| `DEBUG` | 否 | `false` | 调试模式（生产必须 false） |
| `DATABASE_URL` | 否 | `sqlite:///./data/cboard.db` | 数据库连接串：SQLite（Docker 默认，数据在 `./data`）或 `mysql://user:pass@host:3306/db?charset=utf8mb4&parseTime=True&loc=Local` |
| `SECRET_KEY` | **是** | 无 | **JWT 签名密钥，生产必须改为 32 位以上随机字符串** |
| `BACKEND_CORS_ORIGINS` | 否 | localhost 列表 | CORS 白名单，逗号分隔，生产替换为你的域名 |
| `PROJECT_NAME` | 否 | `CBoard Go` | 项目名称（邮件署名等） |
| `VERSION` | 否 | `1.0.0` | 版本号 |
| `API_V1_STR` | 否 | `/api/v1` | API 前缀 |
| `ADMIN_PASSWORD` | 否 | 随机生成 | 管理员固定密码（首次创建 & 每次启动校验重置） |
| `ADMIN_USERNAME` | 否 | `admin` | 默认管理员用户名 |
| `ADMIN_EMAIL` | 否 | `admin@example.com` | 默认管理员邮箱 |
| `SMTP_HOST` | 否 | 空 | SMTP 服务器地址（邮件功能需要） |
| `SMTP_PORT` | 否 | `587` | SMTP 端口 |
| `SMTP_USERNAME` | 否 | 空 | SMTP 账号 |
| `SMTP_PASSWORD` | 否 | 空 | SMTP 密码 / 授权码 |
| `SMTP_FROM_EMAIL` | 否 | 空 | 发件邮箱 |
| `SMTP_FROM_NAME` | 否 | `CBoard Modern` | 发件人名称 |
| `SMTP_ENCRYPTION` | 否 | `tls` | 加密方式（tls / none 等） |
| `UPLOAD_DIR` | 否 | `uploads` | 上传目录（头像 / 附件 / 备份 / 日志） |
| `MAX_FILE_SIZE` | 否 | `10485760` | 上传文件大小上限（字节，默认 10 MB） |
| `DISABLE_SCHEDULE_TASKS` | 否 | `false` | 是否禁用定时任务 |
| `TRUSTED_PROXIES` | 否 | 空 | 可信反向代理 IP 列表（保证真实客户端 IP） |
| `GEOIP_DB_PATH` | 否 | `./GeoLite2-City.mmdb` | GeoIP 数据库路径（不存在时自动下载） |
| `REDIS_ADDR` | 否 | 空 | Redis 地址（如 `localhost:6379`），不配则禁用缓存 |
| `REDIS_PASSWORD` | 否 | 空 | Redis 密码（可选） |

### Redis 可选配置

```env
# 配置后大幅提升 GeoIP 查询等缓存性能；不配置则自动禁用缓存，功能不受影响
REDIS_ADDR=localhost:6379
# REDIS_PASSWORD=your_password_here
```

三种启用方式（菜单 12 全部自动完成）：

| 方式 | 脚本实际做的事 |
|---|---|
| Docker（默认） | 没装 Docker 就自动装（`apt/dnf/yum docker.io`，失败回落 `get.docker.com`），再以 `--restart=always -p 127.0.0.1:6379:6379` 起名为 `redis` 的容器（**不暴露公网**） |
| 系统包 | `apt-get install redis-server redis-tools`（或 `dnf`/`yum install redis`）→ `systemctl enable --now` → 轮询 `PING` 确认真的起来了 |
| 跳过 | `.env` 不动，应用继续无缓存运行 |

手动等价命令：

```bash
docker run -d --name redis --restart=always -p 127.0.0.1:6379:6379 redis:alpine
```

> 两种方式都以 `PING` 通为准判定成功，任一种失败会**自动换另一种重试**，不会中断部署；
> 重复执行菜单 12 是幂等的（`.env` 里用带标记块整块替换，不会堆积重复行/空行，已有容器直接复用）。

### Nginx 反代参考

> ⚠️ **最终模板（含附件上传转发）**。生产环境请使用下面的完整配置——
> 若省略 `location /uploads/`，附件请求会落到 SPA fallback 返回 `index.html`，
> 浏览器将图片当作 HTML 打不开（若前置了 Cloudflare 还会把错误 HTML 缓存为图片内容）。

```nginx
# 反向代理 CBoard（Go 后端监听 127.0.0.1:8000，前端静态由 nginx 直接服务）
server {
    listen 80;
    server_name yourdomain.com;
    # 生产请配置 443 + SSL（证书可用 certbot / 宝塔签发）

    # 附件上传上限必须大于后端单文件上限（默认 30MB）
    client_max_body_size 50m;

    root /path/to/cboard/frontend/dist;   # 前端构建产物目录
    index index.html;

    # ① 后端 API 一律转发给 Go
    location /api/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        # 如需要 SSE 实时日志 / 长连接：
        proxy_buffering off;
        proxy_read_timeout 3600s;
    }

    # ② 上传的附件（工单图片/视频/文档、头像等）转发给 Go 的 /uploads 静态服务。
    #    关键：不要用下面「location / 的 try_files」兜住它——那会让图片请求返回 index.html！
    #    no-cache：附件可更新/删除，避免浏览器或 Cloudflare 缓存旧文件。
    location /uploads/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        expires -1;
    }

    # ③ 前端静态资源（带 hash 的 assets 可长缓存；其余回 index.html 走 SPA）
    location /assets/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

**若你在 Nginx 前还挂了 Cloudflare 等 CDN**：修复 Nginx 后，历史期间被 CDN
缓存的「错误 HTML 附件」仍会挡在旧 URL 前，请到 CDN 面板 **Purge Cache** 清理一次；
应用代码已对附件 URL 追加 `?raw=1` 独立缓存键，新 URL 不受历史脏缓存影响。

---

## 💾 数据库备份

### 自动备份（推荐）

系统支持**自动备份数据库并上传到 GitHub / Gitee**：

1. 管理后台 → **系统设置 → 备份设置**；
2. 配置备份计划与仓库（GitHub / Gitee）；
3. 系统按计划自动备份并推送远程仓库，实现异地容灾。

相关文档：`docs/配置/备份设置说明.md`、`docs/配置/GitHub配置说明.md`、`docs/配置/Gitee配置说明.md`

### 手动备份

```bash
# 方式一：后台触发（管理端操作）
POST /api/v1/admin/backup

# 方式二：直接复制数据库文件（Docker / 非 Docker 通用）
cp cboard.db cboard-$(date +%F).db
tar czf backup-$(date +%F).tar.gz cboard.db uploads
```

### 升级保护

- 升级前务必先备份（或使用 `scripts/db_preflight` 只读预检）；
- 升级脚本会自动备份数据库到 `uploads/backups/upgrade_pre_<时间戳>.db`，失败可回滚。

---

## 🔧 故障排查

### 常见问题速查

| 症状 | 可能原因 | 解决方案 |
|------|----------|----------|
| **服务无法启动** | 端口被占用 / 配置错误 | `lsof -i :8000` 查占用；检查 `.env` 语法与 `DATABASE_URL` 路径；查看日志 `journalctl -u cboard -f` 或 `docker compose logs app` |
| **502 Bad Gateway** | 后端未启动 / 端口不匹配 | 确认 8000 端口进程存在；检查 Nginx `proxy_pass` 端口与 `.env` 的 `PORT` 一致 |
| **数据库"数据消失"** | 启动目录 / `DATABASE_URL` 变化导致连到新库 | 启动日志出现「⚠️ 即将创建全新数据库」即为信号；核对路径与卷挂载；旧文件未被删除，找到后改回路径即可 |
| **脚本显示的邮箱登录报 401** | 已修：账号/邮箱以**数据库**为准（应用只在首次创建管理员时用 `.env` 的 `ADMIN_USERNAME`/`ADMIN_EMAIL`，后台改过之后 `.env` 不会跟着变）。脚本现在会打印实际生效的账号，并在与 `.env` 不一致时标注两者；密码仍以 `.env` 的 `ADMIN_PASSWORD` 为准（应用每次启动都会按它重置） |
| **管理员无法登录** | 密码错误 / 账户被锁 / IP 被限流 | `go run scripts/admin_tool "新密码"` 重置；`go run scripts/unlock_user admin` 解锁；IP 限流等待 15 分钟 |
| **邮件发送失败** | SMTP 配置错误 / 端口被墙 | 检查 SMTP_HOST/PORT/USERNAME/PASSWORD；QQ 邮箱需使用**授权码**而非登录密码；25 端口常被运营商屏蔽，改用 465/587 |
| **GeoIP 功能未生效** | mmdb 文件缺失 | 系统会自动下载 `GeoLite2-City.mmdb`；手动下载：`https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb` |
| **支付回调失败** | 回调地址错误 / CORS | 确认支付平台配置的回调 URL 指向 `/api/v1/payment/...` 回调端点；检查 DEBUG 日志 |
| **订阅无法更新** | 客户端 UA 未知 / 订阅过期 | 检查节点订阅 URL 是否有效；UA 未知时返回通用格式；确认订阅未过期 |
| **节点全部离线** | 心跳超时 / 自建节点脚本问题 | 自建节点心跳 30s/超时 3min；检查节点服务器 sing-box 进程与 `cboard-heartbeat` 服务 |
| **Docker 端口冲突** | 8000 被占用 | 修改 `docker-compose.yml` 端口映射为 `"8001:8000"` |
| **升级时提示「缺少 ACME challenge 放行段」** | 已修：旧版检查写的是 `grep "location \.well-known/acme-challenge"`，而实际配置是 `location ^~ /.well-known/acme-challenge/`（带 `^~` 修饰符）→ 明明有放行段也误报。现在认 `^~`/`=`/`~`/裸路径所有写法，真缺时会**自动补上**（写进注入块内，幂等） |
| **宝塔开启了「强制 HTTPS」，续期会不会失败** | 会（这是真实的失败点）：宝塔的强制跳转是一段 server 级 `if + rewrite`，执行在 location 匹配**之前**，任何 location 都挡不住 → `http://域名/.well-known/acme-challenge/<token>` 一律 301。证书有效时 LE 跟随跳转能过；证书过期/不可用时跟随 HTTPS 就失败。脚本会自动装一对 `renewal-hooks/{pre,post}` 钩子：续期前临时注释掉该跳转并**轮询确认已生效**，续期后原样恢复（逐字节还原，`nginx -t` 失败立即回滚）。实测 `certbot renew --dry-run` 通过 |
| **站点配置文件名和域名不一致**（如 `speedora.conf` 服务 `speedora.top`） | 脚本会按 `server_name` **认领真正在服务该域名的配置文件**（跳过 `.bak/.backup` 副本），不再另建一个同名文件造成两个 server_name 相同的 server 块 |
| **Redis 装不上 / 起不来** | 环境限制（如 apt 版 redis-server 因 `libjemalloc.so.2` 映射失败起不来）/ 未装 Docker | 直接跑**菜单 12**：Docker 与系统包两条路会自动互相兜底，脚本还会打印 `journalctl -u redis-server` 的真实报错；两条都失败也只警告、不中断部署（网站照常运行）。手动排查：`journalctl -u redis-server -n 20`、`docker logs redis` |

### 日志位置

| 日志 | 路径 |
|------|------|
| 应用日志 | `项目目录/server.log` 或 `项目目录/uploads/logs/app.log` |
| 服务日志 | `journalctl -u cboard -f`（非 Docker） |
| 容器日志 | `docker compose logs -f app`（Docker） |

### 健康检查

```bash
curl http://127.0.0.1:8000/health
# 返回 OK 即服务正常
```

---

## 📄 许可证

本项目采用 **MIT 许可证**。

```
MIT License

Copyright (c) 2024 CBoard

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

---

## 🙏 致谢

- [Gin](https://github.com/gin-gonic/gin) - 高性能 Web 框架
- [GORM](https://gorm.io/) - Go ORM 库
- [Vue 3](https://vuejs.org/) / [Element Plus](https://element-plus.org/) - 现代化前端
- [sing-box](https://github.com/SagerNet/sing-box) - 自建节点核心代理
- [GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) - 地理位置数据库

---

**最后更新**：2025  
**版本**：v1.x  
**状态**：✅ 生产就绪
