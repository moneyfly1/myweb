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
| 🐳 **易于部署** | 支持宝塔面板、无宝塔 VPS 一键脚本、Docker Compose 三种部署方式，开箱即用 |

---

## ✨ 核心特性

### 核心能力

- 🚀 **极致性能**：内存占用仅 35-95 MB，Go 并发模型 + 异步 goroutine 通知，高并发下依然稳定
- ⚡ **快速启动**：毫秒级冷启动，SQLite 默认零配置即可运行
- 🔒 **企业级安全**：JWT + 刷新令牌 + 黑名单、bcrypt 加密、登录限流、暴力破解检测、敏感字段脱敏、验证码原子消费、CORS/CSRF 防护
- 📦 **功能完整**：371 个 API 路由，覆盖用户端 16 个页面 + 管理端 26 个页面
- 🎨 **现代化前端**：Vue 3 + Element Plus 响应式设计，深色 / 浅色主题，移动端抽屉全屏适配
- 🐳 **三种部署**：宝塔面板脚本 / 无宝塔 VPS 一键脚本 / Docker Compose

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
| Nginx | 任意版本 | 宝塔环境由面板提供；无宝塔时由 `install-vps.sh` 自动安装 |
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
| 📊 Dashboard | 全局概览、关键指标、收入 / 用户趋势 |
| 👥 Users | 用户筛选 / 编辑 / 禁用 / 批量操作、重置密码、**登录为用户**、发送邮件、签到记录 |
| 🚨 AbnormalUsers | 异常用户识别（恶意注册、异常流量、滥用嫌疑） |
| 🌍 Nodes | 节点 CRUD、节点采集、批量导入、批量测速、健康监控、去重（Type:Server:Port）、分组 |
| ⚡ CustomNodes | 专线节点创建（链接导入 / 手动填写）、分配 / 取消分配、到期管理、测速 |
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
- ✅ 专线节点：链接导入、分配与取消分配、独立到期时间
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

| 方式 | 适用场景 | 安装工具 | 难度 |
|------|----------|----------|------|
| 🐳 **Docker** | 任意 Linux 服务器（推荐生产使用） | `docker compose up -d` | ⭐ |
| 🖥️ **无宝塔 VPS** | 纯 VPS、未装宝塔 | `install-vps.sh` 一键脚本 | ⭐⭐ |
| 🧱 **宝塔面板** | 已安装宝塔面板 | `install.sh` + 面板建站 | ⭐⭐ |

---

### 🐳 方式一：Docker 部署（推荐）

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
| **后端起不来，日志报 Redis 错误** | 无需处理——Redis 连不上会自动禁用缓存并降级运行，功能不受影响 |
| **想改 .env 后生效** | 修改宿主机 `.env` 后执行 `docker compose up -d`（会重新读取环境变量并重建容器） |

---

### 🖥️ 方式二：无宝塔（纯 VPS）一键部署

**适用**：Ubuntu / Debian / CentOS，未安装宝塔。脚本自动安装 Nginx、Go、Node.js、Certbot 并完成全流程部署。

#### 前置条件

| 项 | 要求 |
|----|------|
| 系统 | Ubuntu 18.04+ / Debian 10+ / CentOS 7+ |
| 配置 | 至少 1 核 CPU、512 MB 内存、10 GB 磁盘 |
| 域名 | 已绑定且 DNS 解析到本机 IP |
| 端口 | 80、443 已开放 |

#### 安装步骤

```bash
# 1. 下载并运行脚本（需 root）
curl -sL https://raw.githubusercontent.com/moneyfly1/myweb/main/install-vps.sh -o install-vps.sh
sudo bash install-vps.sh

# 2. 按提示输入域名、项目目录（默认 /opt/cboard）、管理员信息
# 3. 脚本自动完成：装依赖 → 拉代码 → 装 Go/Node → 编译后端/构建前端
#    → 生成 .env → 配 Nginx → 申请 SSL → 创建 systemd 服务并启动
```

#### 验证

- 前端：`https://你的域名`
- 管理后台：`https://你的域名/admin/login`
- 健康检查：`https://你的域名/health`

#### 安装后管理

```bash
systemctl start/stop/restart/status cboard   # 服务管理
journalctl -u cboard -f                       # 实时日志
tail -f /opt/cboard/server.log                # 应用日志
# 修改配置：编辑 /opt/cboard/.env 后 systemctl restart cboard
```

> ⚠️ 国内网络 GitHub 克隆失败时：先把代码手动放入安装目录（如 `/opt/cboard`），重新运行脚本并在「是否删除并重新下载」时选 **n**。

---

### 🧱 方式三：宝塔面板部署

**适用**：已安装宝塔面板的服务器，先建站再跑脚本。

#### 安装步骤

1. **宝塔建站**：登录宝塔 → 网站 → 添加站点 → 填写域名，根目录如 `/www/wwwroot/example.com`（PHP 选纯静态即可，无需建数据库）；
2. **放入代码**（任选其一）：
   ```bash
   # SSH 方式
   cd /www/wwwroot/example.com && rm -f index.html
   git clone https://github.com/moneyfly1/myweb.git .
   ```
   或使用宝塔文件管理器 / 本地上传（SCP）；
3. **运行安装脚本**：
   ```bash
   cd /www/wwwroot/example.com
   chmod +x install.sh
   sudo ./install.sh
   ```
4. 按提示输入项目目录、域名、管理员用户名/邮箱/密码，首次安装选菜单 **1（一键全自动部署）**；
5. 脚本自动：装 Go/Node → 编译后端/构建前端 → 配置 Nginx 反代 → 申请 SSL → 创建 systemd 服务并启动。

#### 安装后管理

| 操作 | 方法 |
|------|------|
| 重启服务 | `systemctl restart cboard` 或项目目录 `sudo ./install.sh` 选 8 |
| 查看日志 | `journalctl -u cboard -f` 或 `tail -f 项目目录/server.log` |
| Nginx | 宝塔 → 网站 → 站点 → 设置 → 配置文件（脚本已写入反代） |
| 防火墙 | 宝塔「安全」中放行 80、443 |
| 创建/重置管理员 | `sudo ./install.sh` 选 2 |

---

## 🎯 三种部署方式：管理员与域名配置时机对比

| 部署方式 | 域名在哪配置 | 管理员在哪配置 | 配置时机 |
|---------|-------------|---------------|---------|
| **宝塔（install.sh）** | **宝塔添加站点时**（域名 = 站点目录名，脚本自动取 `basename 项目目录`） | 运行脚本后选**菜单 2**「创建/重置管理员账号」交互填写（用户名/邮箱/密码） | 部署完成后随时可改 |
| **无宝塔（install-vps.sh）** | **运行脚本时交互输入**（提示"域名 (如 example.com)"） | **运行脚本时交互输入**（脚本提示填写管理员用户名/邮箱/密码） | 安装过程中 |
| **Docker** | `.env` 中 `PANEL_PUBLIC_URL`（仅自建节点回传需要；纯订阅无需域名） | `.env` 中 `ADMIN_USERNAME`/`ADMIN_EMAIL`/`ADMIN_PASSWORD`，首次启动自动创建 | 启动前配置 `.env` |

**关键说明：**

- **宝塔**：域名不是填在 .env 里的，而是**宝塔建站时确定**（如站点目录 `/www/wwwroot/你的域名`），脚本用目录名做域名配置 Nginx。管理员用脚本菜单 2 管理。
- **Docker**：管理员完全通过 `.env` 环境变量注入，首次启动自动创建；之后每次重启校验密码（管理员被锁也能自动解锁）。
- **无宝塔**：域名和管理员都在 `install-vps.sh` 运行过程中交互输入，一步到位。

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
go run scripts/admin_tool.go                # 交互式创建（默认 admin/admin123，生产勿用）
go run scripts/admin_tool.go "新密码"        # 直接重置管理员密码

# 生产推荐：环境变量方式
export ADMIN_USERNAME="admin"
export ADMIN_EMAIL="admin@your-domain.com"
export ADMIN_PASSWORD="YourStrongPassword123!"
go run scripts/admin_tool.go
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
go run scripts/unlock_user.go admin
go run scripts/unlock_user.go admin@your-domain.com

# 解锁普通用户
go run scripts/unlock_user.go user@your-domain.com
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

快速启动 Redis：

```bash
docker run -d --name redis -p 6379:6379 redis:alpine
```

### Nginx 反代参考

```nginx
server {
    listen 80;
    server_name yourdomain.com;
    # 生产请配置 443 + SSL

    location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        # 如需要 SSE 实时日志：
        proxy_buffering off;
        proxy_read_timeout 3600s;
    }
}
```

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
| **管理员无法登录** | 密码错误 / 账户被锁 / IP 被限流 | `go run scripts/admin_tool.go "新密码"` 重置；`go run scripts/unlock_user.go admin` 解锁；IP 限流等待 15 分钟 |
| **邮件发送失败** | SMTP 配置错误 / 端口被墙 | 检查 SMTP_HOST/PORT/USERNAME/PASSWORD；QQ 邮箱需使用**授权码**而非登录密码；25 端口常被运营商屏蔽，改用 465/587 |
| **GeoIP 功能未生效** | mmdb 文件缺失 | 系统会自动下载 `GeoLite2-City.mmdb`；手动下载：`https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb` |
| **支付回调失败** | 回调地址错误 / CORS | 确认支付平台配置的回调 URL 指向 `/api/v1/payment/...` 回调端点；检查 DEBUG 日志 |
| **订阅无法更新** | 客户端 UA 未知 / 订阅过期 | 检查节点订阅 URL 是否有效；UA 未知时返回通用格式；确认订阅未过期 |
| **节点全部离线** | 心跳超时 / 自建节点脚本问题 | 自建节点心跳 30s/超时 3min；检查节点服务器 sing-box 进程与 `cboard-heartbeat` 服务 |
| **Docker 端口冲突** | 8000 被占用 | 修改 `docker-compose.yml` 端口映射为 `"8001:8000"` |

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
