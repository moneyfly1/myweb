# 🚀 CBoard — Modern Subscription Management System

> **A high-performance, secure, and feature-complete airport panel (subscription management system) built with Go + Vue 3.**
> Designed for VPN / proxy service providers who want the full operations chain — user management, subscription delivery, payments, orders, self-hosted nodes, analytics, and marketing — in one deployable package that runs in as little as **35–95 MB of memory**.

[中文](README_zh.md) | English

---

## 📖 Table of Contents

- [Introduction & Design Philosophy](#-introduction--design-philosophy)
- [Key Features](#-key-features)
- [Technology Stack](#-technology-stack)
- [System Requirements](#-system-requirements)
- [Feature List](#-feature-list)
- [Core Design Deep-dive](#-core-design-deep-dive)
- [Installation Guide](#-installation-guide)
- [Admin Account Management](#-admin-account-management)
- [Configuration Guide](#-configuration-guide)
- [Database Backup](#-database-backup)
- [Troubleshooting](#-troubleshooting)
- [License](#-license)

---

## 💡 Introduction & Design Philosophy

### What is CBoard?

**CBoard** is a modern subscription management system — commonly called an *airport panel* — purpose-built for VPN/proxy service providers. It covers the **entire airport operation chain**: user registration and authentication, subscription link generation, device management, packages, orders, coupons, balance recharging, multi-gateway payments, node management (including SSH self-deployed nodes), tickets, invites, marketing promotions, statistics, and analytics.

It is a **complete rewrite in Go** of the classic PHP-era airport panels (in the lineage of XBoard / V2Board / SSPanel), delivering the same feature set at a fraction of the resource cost.

### Project Origin

CBoard started from a very concrete personal need:

> You hold several proxy service subscriptions with unused traffic, and you want to **share them securely with friends** — but you also want to **prevent abuse**.

The core problem chain that shaped the product:

1. Purchased proxy packages typically do not limit device count, but uncontrolled sharing quickly leaks the resource.
2. Friends might download node information and use it independently, permanently losing control over the subscription.
3. A naive shared link is a single point of failure — if one friend leaks it, everyone's access is compromised.

**The CBoard solution:**

- 🔐 **Device limits** — each subscriber is bound to a fixed number of devices (via device fingerprinting), so the subscription cannot be forwarded indefinitely.
- 🔄 **Periodic subscription resets** — subscription addresses are automatically reset on a schedule (recommended daily or every two days), so a leaked link becomes worthless within hours.
- 🧲 **Auto-aggregation** — multiple upstream subscriptions are automatically collected, aggregated, and re-served as one clean subscription link, hiding the underlying upstream addresses.

**Result:** friends get a single, always-current, device-bound link — and every attempt to hoard or redistribute the resource is technically neutralized.

### Design Philosophy: Four Pillars

| Pillar | What it means in practice |
|---|---|
| ⚡ **Performance** | Go + Gin backend. **35–95 MB memory** (vs. 300–850 MB for Python panels), millisecond startup, optional Redis cache, goroutine-based async processing. |
| 🔒 **Security** | JWT access/refresh tokens with blacklist, bcrypt password hashing, login rate limiting with brute-force lockout, sensitive-field masking, atomic verification codes, strict CORS policy, production-mode configuration enforcement. |
| 🧩 **Completeness** | The full airport operation chain out of the box: 16 user pages + 26 admin pages, 371 API routes, 9+ client formats, 10+ payment gateways, 15 self-hosted protocols. |
| 🐳 **Deployability** | SQLite by default (zero external dependencies), MySQL/PostgreSQL optional, Redis optional, one-command BaoTa panel installer, one-command VPS installer, and a production-ready two-stage Docker build. |

### Design Principles

- **Defaults that work**: SQLite + no Redis must run correctly out of the box; Redis is a pure accelerator, never a hard dependency.
- **Safe failure**: a fresh database is loudly announced at startup (never silently rebuilt over existing data), and corrupted SQLite files are auto-recovered from recent backups.
- **Client-aware delivery**: the panel knows what each client can actually parse, and never sends a protocol a client cannot handle.
- **Operator ergonomics**: batch operations, filters, login-as, one-click node deployment, and scriptable admin tooling are first-class, not afterthoughts.

---

## ✨ Key Features

| Category | Highlights |
|---|---|
| 🚀 **High Performance** | Go/Gin backend, 35–95 MB RAM, SQLite/MySQL/PostgreSQL, optional Redis cache (50–100× faster subscription generation on cache hits), async goroutine mail queue. |
| 🔒 **Security** | JWT access + refresh tokens, token blacklist, bcrypt hashing, login rate limit + brute-force lockout, sensitive data masking, atomic verification codes, CORS whitelist enforcement, production-mode secret validation. |
| 📡 **Multi-client Subscription** | Auto client detection via User-Agent: Clash YAML, Clash Meta, Stash, Surge, Loon, Quantumult X, Sing-box JSON, Shadowrocket, v2rayN. Version-aware protocol filtering per client. |
| 🖥️ **Self-hosted Nodes (signature feature)** | SSH auto-deploy sing-box, 15 protocols, 30s heartbeat / 3 min timeout, remote management (reset UUID, change password/port, reinstall, traffic quota), acme.sh auto certificate renewal. |
| 🧾 **Packages, Orders & Coupons** | Full package CRUD, order lifecycle, coupon engine, balance system, device upgrade pricing, recharge, mixed payment (balance + gateway). |
| 💳 **Multi-payment** | Alipay, WeChat Pay, Yipay, Codepay, Apple Pay, Stripe, PayPal, USDT — domestic and international gateways. |
| 👥 **User Management** | Registration/login, JWT refresh, email verification, forgot password, invite codes, device limits, login history, user levels with discounts, batch operations, login-as. |
| 📊 **Analytics** | Dashboard, DAU/WAU/MAU, retention, churn prediction, revenue statistics, GeoIP-based user analytics. |
| 🎫 **Operations** | Ticket system, knowledge base, announcements, email queue with per-email detail, audit logs, system logs, backup & repo sync (GitHub/Gitee), node health monitoring. |
| 🎁 **Engagement** | Daily check-in with random rewards (0.1–1 CNY), promotions (flash sales, new-user offers, member days). |
| 🐳 **Deployability** | BaoTa panel one-click script, bare-VPS one-click script, official multi-stage Docker image, systemd integration, Nginx/HTTPS automation. |
| 🎨 **Modern Frontend** | Vue 3 + Element Plus + Vite + Pinia + ECharts, fully responsive with drawer components, dark-friendly theming. |

---

## 🏗️ Technology Stack

### Backend

| Layer | Technology | Notes |
|---|---|---|
| Language | **Go 1.21+** (built with Go 1.24 toolchain) | Compiled, statically-linkable binary |
| Web framework | [Gin](https://github.com/gin-gonic/gin) | High-performance HTTP framework |
| ORM | [GORM](https://gorm.io/) | Parameterized queries (SQL-injection safe) |
| Database | **SQLite (default)** / MySQL 5.7+ / PostgreSQL 12+ | Driver auto-selected from `DATABASE_URL` |
| Cache | **Redis (optional)** | Hot-data accelerator; gracefully disabled when absent |
| Auth | JWT (HS256) + refresh tokens + blacklist | Access token default 1h, refresh token 30d |
| Config | [Viper](https://github.com/spf13/viper) | `.env` file + real environment variables |
| Background | goroutines + in-process queue | Async email delivery, notifications, scheduled tasks |

### Frontend

| Layer | Technology |
|---|---|
| Framework | Vue 3 (Composition API) |
| UI library | Element Plus |
| Build tool | Vite |
| State | Pinia |
| Router | Vue Router 4 |
| Charts | ECharts |

### Deployment Targets

| Target | Support |
|---|---|
| Bare VPS (Ubuntu/Debian/CentOS) | ✅ `install-vps.sh` one-click script |
| BaoTa (宝塔) Panel | ✅ `install.sh` one-click script |
| Docker / docker-compose | ✅ Official two-stage Dockerfile + compose file |
| Reverse proxy | ✅ Nginx (script-configured) / any proxy in front of port 8000 |

---

## 📋 System Requirements

### Bare-metal / VPS / BaoTa Panel

| Resource | Minimum | Recommended |
|---|---|---|
| CPU | 1 core | 2+ cores |
| Memory | 512 MB | 1 GB+ |
| Disk | 10 GB | 20 GB+ |
| OS | Ubuntu 18.04+ / Debian 10+ / CentOS 7+ | Latest LTS |
| Domain | — | Bound to server IP (required for HTTPS) |
| Open ports | 80, 443 | 80, 443 (+ 8000 if not behind a proxy) |

### Software Prerequisites

| Component | Requirement | Notes |
|---|---|---|
| Go | 1.21+ | Auto-installed by install scripts |
| Node.js | 16+ | Only needed to build the frontend |
| Nginx | Any recent version | Auto-installed/configurated by scripts |
| Database | SQLite (built-in) **or** MySQL/PostgreSQL | SQLite needs no installation |
| Redis | Optional (highly recommended for production) | Auto-configured by install scripts |

### Docker

| Resource | Requirement |
|---|---|
| Docker Engine | 20.10+ |
| Docker Compose | v2 (`docker compose` plugin) |
| Memory | ≥ 512 MB free |
| Disk | ≥ 1 GB free for the image + data |

---

## 📊 Feature List

### 👤 User-side Features (16 pages)

| Page | Features |
|---|---|
| **Dashboard** | Overview of subscription status, traffic usage, device count, recent orders, announcements; **daily check-in** with random reward (0.1–1 CNY) |
| **Subscription** | View subscription URL, traffic/expiry info, **subscription reset**, **send subscription to email**, **convert unused subscription to balance**, copy Clash/sing-box links |
| **Devices** | Device list with **device fingerprinting** (UA + unique fingerprint), add/remove devices, enforce device limit, online status |
| **Packages** | Browse available packages, pricing, features, purchase flow |
| **Orders** | Order list, status tracking, cancel, pay with balance or gateway, mixed payment |
| **Nodes** | View node list, region grouping, latency status, copy individual node links |
| **Help** | Quick start guides, Clash series tutorials |
| **Profile** | Personal info, avatar, password change, notification preferences |
| **LoginHistory** | Full login history with IP, location (GeoIP), device, result |
| **Tickets** | Open/track support tickets with the operator team |
| **Invites** | Invite codes, invite link, reward tracking (invite commission) |
| **Knowledge** | Searchable knowledge base articles |
| **UserSettings** | Theme preference, language, security settings, logout everywhere |
| **PaymentReturn** | Payment callback landing page (order status shown instantly) |
| **UnifiedAuth** | Unified login/register page: login, register, forgot password, email verification, invite-code binding, social/passwordless flows |
| **Daily Check-in** | (Dashboard module) random daily reward, streak-friendly |

### User Core Capabilities

- ✅ Register / login / JWT refresh / forgot password (email) / email verification / invite code registration
- ✅ Subscription URL generation for all major clients with UA auto-detection
- ✅ Subscription reset (manual or scheduled) — anti-leak mechanism
- ✅ Device limit management with device fingerprinting
- ✅ Email the subscription link to yourself
- ✅ Convert remaining subscription value to account balance
- ✅ Purchase packages, apply coupons, pay by balance or third-party gateway
- ✅ View node list with region grouping and latency

### 🛠️ Admin-side Features (26 pages)

| Page | Features |
|---|---|
| **Dashboard** | Real-time overview: users, orders, revenue, active subscriptions, system status |
| **Users** | Filter/search users, edit, disable, **batch operations**, **reset password**, **login as user**, **send email**, view check-in logs, GeoIP info |
| **AbnormalUsers** | Flagged/abnormal accounts (locked, inactive, unusual login patterns) for review |
| **Nodes** | Regular node CRUD, node collection from upstream subscription URLs, manual import (link / Clash config / manual entry), **batch test**, **deduplication** (Type:Server:Port), region grouping |
| **CustomNodes** | Custom/manual node definitions with arbitrary protocol templates |
| **SelfHostNodes** | **Self-hosted node management**: SSH deploy, protocol selection, heartbeat/status, remote management (reset UUID, change password/port, reinstall, traffic quota), auto cert renewal |
| **Subscriptions** | All user subscriptions, search, reset, device management, expiry extension |
| **Orders** | Full order lifecycle, status, cancellation, **CSV/Excel export**, bulk operations |
| **Packages** | Package CRUD, pricing, features, display order, activation state |
| **PaymentConfig** | Gateway configuration: Alipay, WeChat Pay, Yipay, Codepay, Apple Pay, **Stripe, PayPal, USDT** |
| **Settings** | General / registration / notification / announcement / security / theme / invite / admin-notify / node-health / backup / repo-sync / protocol-filter / GeoIP settings |
| **Config** | System configuration key-value management |
| **Statistics** | User/order/revenue/subscription statistics, traffic statistics, GeoIP distribution |
| **Analytics** | DAU/WAU/MAU, retention analysis, churn prediction |
| **EmailQueue** | Outbound email queue: pending/sent/failed, retry |
| **EmailDetail** | Per-email detail inspection (to, subject, body, error) |
| **Logs** | Application logs viewer |
| **SystemLogs** | Audit trail of admin operations |
| **Coupons** | Coupon CRUD, discount types, validity, usage limits, batch generation |
| **Tickets** | Support ticket inbox, reply, close, user lookup |
| **Invites** | Invite-code management, commission rules, reward records |
| **UserLevels** | User level definitions, upgrade rules, per-level discounts |
| **Knowledge** | Knowledge-base article CRUD, categories |
| **Promotions** | Marketing campaigns: flash sales, new-user offers, member days |
| **Profile** | Admin profile, password, 2FA-ready security |
| **ConfigUpdate** | Apply configuration updates / migrations to existing deployments |

### Admin Core Capabilities

- ✅ User management: filter/edit/disable/**batch**/reset password/**login-as**/send email/check-in logs
- ✅ Node management: regular / dedicated / custom / **self-hosted** / batch test / import
- ✅ Self-hosted nodes: SSH auto-deploy sing-box / 15 protocols / heartbeat / remote management / traffic quota / auto cert renewal
- ✅ Orders, packages, coupons, tickets, invites, user levels: full CRUD + batch operations + export
- ✅ Payments: Alipay / WeChat Pay / Yipay / Codepay / Apple Pay / **Stripe / PayPal / USDT**
- ✅ Settings: general / registration / notification / announcement / security / theme / invite / admin-notify / node-health / backup / repo-sync / protocol-filter / GeoIP
- ✅ Statistics / Analytics / Logs / Email queue / Marketing / Backup / Monitoring

### 🌍 Cross-cutting Capabilities

| Capability | Detail |
|---|---|
| 💳 Payment gateways | Alipay, WeChat Pay, Yipay (Alipay/WeChat/QQ Pay), Codepay, Apple Pay, Stripe, PayPal, USDT, balance, mixed payment |
| 🔔 Notification channels | SMTP email (customer + admin), Telegram Bot, Bark iOS push |
| 🖥️ Node types | Regular (collected/imported), dedicated, custom, self-hosted (SSH) |
| 🧠 Scheduled tasks | Subscription resets, node health checks, backup, repo sync, email queue draining, statistics refresh (can be disabled via `DISABLE_SCHEDULE_TASKS`) |
| 🌐 GeoIP | GeoLite2-City MMDB (auto-download), per-user/node region attribution, region-aware subscription delivery |

---

## 🔬 Core Design Deep-dive

### 6.1 Subscription Delivery Engine 📡

The subscription endpoint is the heart of the panel. It is **client-aware**, **version-aware**, **region-aware**, and **cache-accelerated**.

#### Client auto-detection (User-Agent)

When a client requests the subscription URL, CBoard reads the `User-Agent` header and routes to the correct generator:

| User-Agent | Output format |
|---|---|
| Clash (legacy) | Clash YAML |
| Clash Meta / Mihomo | Clash YAML (meta extensions) |
| Stash | Clash YAML (Stash-flavored) |
| Surge | Surge config |
| Loon | Loon config |
| Quantumult X | Quantumult X config |
| sing-box | sing-box JSON |
| Shadowrocket | Shadowrocket-compatible config |
| v2rayN / v2rayNG | v2rayN share links |
| Unknown / browser | Default format (configurable) |

#### Version-based protocol filtering

Different clients support different protocol sets. Sending an unsupported protocol to a client breaks the whole subscription, so CBoard **filters protocols by detected client capability**:

| Client | Protocols delivered | Rationale |
|---|---|---|
| Legacy Clash | SS, VMess only | Old Clash cannot parse Reality/Hysteria2/TUIC etc. |
| Clash Meta / Mihomo | Full set (VLESS+Reality, Hysteria2, TUIC, AnyTLS, SS, VMess, …) | Meta supports modern protocols |
| sing-box | Full set (sing-box JSON) | Native support for all modern protocols |
| Shadowrocket | Depends on **build number** (e.g. `Shadowrocket/1744`): build ≥ 1744 gets Reality-capable set, older builds get a conservative set | Reality support landed at a specific build |

The capability parser (`client_capability`) extracts `(clientType, version)` from the UA — including pure build numbers like `Shadowrocket/1744` — and applies per-client minimum-version gates before emitting any node.

#### Delivery parameters

- **`exclude`** — exclude specific node IDs from the delivered config, e.g. `?exclude=12,37`. Useful when a client cannot handle a particular node or a node is temporarily down.
- **`&filter=` routing** — filter/routing parameter for fine-grained delivery control (node groups, regions, protocols).
- **DB protocol whitelist** — the admin can globally enable/disable protocols in Settings → Protocol Filter; disabled protocols are never delivered regardless of client capability.
- **IP-region-based delivery** — combined with GeoIP, the panel can tailor the delivered node set to the client's region (e.g. exclude nodes that are blocked or undesirable in the requester's country).

#### Device binding & anti-abuse

- Each subscription is bound to a **device fingerprint**; the device limit (default 3, configurable via `DEVICE_LIMIT_DEFAULT` or per package) caps how many devices may consume the subscription.
- **Subscription resets** rotate the subscription token/URL on a schedule, invalidating leaked links.
- The generated config is cached by Redis key `subscription:config:{token}:{format}` (TTL 1–10 min), dropping generation time from **200–500 ms to 10–50 ms** on cache hits. Cache is invalidated on subscription expiry, admin edits, purchases, device changes, and node updates.

### 6.2 Self-hosted Nodes (Signature Feature) 🖥️

CBoard can **deploy and manage nodes entirely from the panel over SSH** — no manual server configuration required.

#### Architecture

```
┌──────────────┐   SSH (deploy/control)   ┌──────────────────┐
│   CBoard     │ ───────────────────────► │   Target VPS     │
│   Panel      │                          │  ┌────────────┐  │
│              │ ◄── heartbeat (30s) ───  │  │ sing-box   │  │
│              │                          │  │ + agent    │  │
│              │                          │  │ + systemd  │  │
└──────────────┘                          └───────────────┘  │
                                          └──────────────────┘
```

1. Admin enters the target VPS SSH credentials (key or password) and picks protocols.
2. The panel generates an install script, pushes it over SSH, and deploys **sing-box** with a systemd service plus a **heartbeat agent** (`cboard-heartbeat`).
3. The agent reports to `POST /api/v1/agent/heartbeat` every **30 seconds**; if the panel hears nothing for **3 minutes**, the node is marked **offline**.
4. Admin can issue remote management commands over SSH at any time.

#### Supported protocols (15)

VLESS + WebSocket (WS) · VLESS + Reality · VLESS + Reality + Vision · VLESS + Reality + gRPC · Hysteria2 · TUIC (v5) · AnyTLS · Shadowsocks (SS) · SS + AEAD · VMess + WS · VMess + TCP · VLESS + TCP · Trojan · (plus sing-box variations of the above) — all behind a **single shared user UUID** per node, so one subscription works across every protocol on that node.

#### Remote management operations

| Operation | What it does |
|---|---|
| 🔄 Reset UUID | Regenerates the node's user UUID (kicks all current sessions) |
| 🔑 Change password | Rotates the node credentials |
| 🔌 Change port | Moves the inbound listener to a new port |
| ♻️ Reinstall | Re-deploys sing-box from scratch on the target VPS |
| 📊 Traffic quota | Set a quota; the agent **auto-blocks** traffic when the quota is exhausted |
| 🔐 Auto cert renewal | Installs **acme.sh** and renews TLS certificates automatically |

#### Why self-hosted nodes matter

- **Full control** — no dependency on third-party node providers or upstream collection sources.
- **One-click lifecycle** — deploy, monitor, reconfigure, and decommission entirely from the admin panel.
- **Cheap & fast** — sing-box is a single lightweight binary; 15 protocols share one port/UUID, minimizing firewall surface.

### 6.3 Security Model 🔒

| Layer | Implementation |
|---|---|
| **Authentication** | JWT **access token** (HS256, default 1h, configurable via `JWT_EXPIRE_HOURS`) + **refresh token** (default 30d via `REFRESH_TOKEN_EXPIRE_DAYS`); refresh rotation and **token blacklist** for logout/revocation |
| **Passwords** | **bcrypt** hashing (cost-adaptive, `$2a/$2b/$2y`); the admin tool verifies hash format on creation |
| **Brute-force defense** | Per-IP + per-account **login rate limiting** with progressive lockout; locked accounts can be unlocked with `scripts/unlock_user.go` (admin account is force-unlocked on restart when `ADMIN_PASSWORD` is set) |
| **Data masking** | Sensitive fields (tokens, secrets, partial emails/phones) are **masked** in API responses and logs |
| **Verification codes** | **Atomic** email verification codes — single-use, expiry-checked, race-safe (no double-redemption) |
| **CORS/CSRF** | CORS origins are an explicit **whitelist**; wildcard `*` and `null` are **rejected at startup** (`validateConfig`); JWT-in-header authentication mitigates CSRF |
| **SQL injection** | All queries go through GORM parameterization |
| **Secrets enforcement** | In `ENV=production`/`prod`, startup **fails loudly** if `SECRET_KEY` is weak (< 32 bytes, placeholder, or low-entropy) or DB passwords are empty/known defaults |
| **Path traversal guard** | GeoIP/upload paths are validated with safe path joins |
| **Data-loss guard** | A fresh SQLite database (file missing) triggers a **loud startup warning** so operators never silently "rebuild" over existing data |

### 6.4 Performance ⚡

- **Memory footprint: 35–95 MB** (vs. 300–850 MB for Python-based panels) — comfortably fits on a 512 MB VPS.
- **Millisecond startup**; health check at `/health`.
- **Redis cache layers** (optional; auto-disabled when Redis is absent):

| Data | Cache key pattern | TTL | Gain |
|---|---|---|---|
| Subscription config | `subscription:config:{token}:{format}` | 1–10 min | ⭐⭐⭐⭐⭐ 200–500 ms → 10–50 ms |
| Package list | `packages:list:active` | 30 min | ⭐⭐⭐ |
| Announcements | `announcements:list:active` | 10 min | ⭐⭐⭐ |
| System config | `system:config:{category}` | 1 h | ⭐⭐⭐ |
| Payment methods | `payment:methods:active` | 1 h | ⭐⭐⭐ |
| Knowledge base | `knowledge:*` | 1 h | ⭐⭐⭐ |
| Statistics | `statistics:{key}` | 30 s–5 min | ⭐⭐ |

- **Async processing**: email delivery, notifications, and other side effects run on **goroutines / an in-process queue** — HTTP handlers never block on SMTP.
- **Scheduler**: subscription resets, node health checks, backups, and repo sync run as background scheduled tasks (toggleable via `DISABLE_SCHEDULE_TASKS`).
- **Low-end tuning**: `OPTIMIZE_FOR_LOW_END=true` by default; worker pool size via `WORKERS`.

---

## 🚀 Installation Guide

CBoard offers **three officially supported installation methods**:

| Method | Best for | Script | Effort |
|---|---|---|---|
| 🐳 **Docker** | Any Linux/macOS/Windows with Docker; isolated, reproducible, easy upgrades | `docker compose` | Low |
| 🖥️ **Bare VPS** | New VPS without a panel; full automation incl. Nginx + HTTPS | `install-vps.sh` | Low (one command) |
| 🪟 **BaoTa (宝塔) Panel** | Servers already running BaoTa Panel | `install.sh` | Low (menu-driven) |

> ⚠️ **All methods require root/sudo access.** For production, always bind a domain and enable HTTPS.

---

### 🐳 Method 3: Docker (Recommended for Isolated Deployments) — *Most Detailed*

This is the officially supported container deployment. It uses a **three-stage Dockerfile** (backend build + frontend build + runtime) and a **docker-compose.yml** with bind-mounted **directories** so your data (SQLite + WAL logs + uploads) lives on the host.

#### Prerequisites

```bash
docker --version            # Docker 20.10+
docker compose version      # Compose v2 plugin
```

Verify ports are free:

```bash
ss -tlnp | grep 8000 || echo "port 8000 is free"
```

#### Step ① — Clone the repository

```bash
git clone https://github.com/moneyfly1/myweb.git cboard
cd cboard
```

> In mainland China, if GitHub is slow, mirror the clone (e.g. `git clone https://gitee.com/...`) or download the source archive and extract it into `cboard/`.

#### Step ② — Configure `.env` (critical!)

Copy the sample template and edit it:

```bash
cp .env.example .env
vim .env
```

**MUST change (build/startup will FAIL otherwise):**

| Variable | Why | Example |
|----------|-----|---------|
| `SECRET_KEY` | ⚠️ `docker-compose.yml` uses `${SECRET_KEY:?}` — compose **refuses to start** if unset; weak keys are also rejected in production mode | output of `openssl rand -hex 32` |
| `ADMIN_PASSWORD` | Used to auto-create the admin at first boot. Defaults to `admin123` (insecure) if unset | your strong password (≥ 6 chars) |

**Recommended (optional):**

```env
ADMIN_USERNAME=admin                # override default username
ADMIN_EMAIL=admin@example.com       # override default email
SMTP_HOST=smtp.example.com          # email notifications (optional)
SMTP_PORT=587
SMTP_USERNAME=no-reply@example.com
SMTP_PASSWORD=your-smtp-password
PANEL_PUBLIC_URL=https://your-domain.com  # REQUIRED if you use self-hosted nodes (agent callback)
TRUSTED_PROXIES=127.0.0.1,::1             # REQUIRED behind Nginx/Cloudflare
```

> **Leave as-is** (already correct for Docker): `HOST=0.0.0.0`, `PORT=8000`, `DATABASE_URL=sqlite:///./data/cboard.db`, `DEBUG=false`.

#### Step ③ — Build & start

```bash
docker compose up -d --build
```

First start runs a **three-stage build** (backend + frontend + runtime image, see below), takes ~1–5 minutes.

```bash
docker compose ps          # status should be "Up"
docker compose logs -f app # watch startup logs
```

Success looks like:

```
服务器启动在 0.0.0.0:8000
管理员账号已自动创建 / 管理员账号已就绪
```

#### Step ④ — Admin account

**Method A: auto-created via env vars (recommended)**

Set `ADMIN_PASSWORD` in `.env` before first boot (Step ②). The container auto-creates the admin (username/email from `ADMIN_USERNAME`/`ADMIN_EMAIL`, default `admin`) on a fresh database. The password is **re-verified on every restart** — even if the admin gets locked out, a restart unlocks it.

If `ADMIN_PASSWORD` is unset, a **random password** is printed in the startup log:

```bash
docker compose logs app | grep "初始密码"
```

**Method B: inspect inside the container**

```bash
docker compose exec app sh
ls -la /root/data/   # confirm cboard.db exists
```

> The runtime image is minimal Alpine without Go toolchain — create/reset the admin via env vars (Method A); data lives on the host mount.

#### Step ⑤ — Access

| Entry | URL |
|-------|-----|
| User frontend | `http://SERVER_IP:8000` |
| Admin panel | `http://SERVER_IP:8000/admin/login` |
| Health check | `http://SERVER_IP:8000/health` |

> For production, put Nginx/Caddy in front with HTTPS (expose `80/443`, keep `8000` internal).

#### 📦 Data persistence

All data lives in **host-mounted directories** — deleting/recreating the container keeps data:

| Mount | Host path | Container path | Contents |
|-------|-----------|----------------|----------|
| SQLite data dir | `./data` | `/root/data` | cboard.db + WAL/SHM logs |
| Uploads | `./uploads` | `/root/uploads` | avatars, attachments, backups, logs |

> ⚠️ Mount **directories**, not a single `.db` file: SQLite writes `cboard.db-shm`/`cboard.db-wal` (WAL mode), and Docker creates a **directory** when the host file doesn't exist, breaking first boot — fixed by directory mounts.

**Backup = copy these two directories:**

```bash
cp -r data /backup/data-$(date +%F)
cp -r uploads /backup/uploads-$(date +%F)
```

#### 🐳 Docker image build (Dockerfile)

Three stages (backend compile + frontend build + runtime):

```dockerfile
# Stage 1: backend build (golang:1.24-alpine)
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache gcc musl-dev          # for SQLite cgo driver
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o cboard-go cmd/server/main.go

# Stage 2: frontend build (node:20-alpine — Vite 7 needs Node 20+)
FROM node:20-alpine AS frontend-builder
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install --legacy-peer-deps
COPY frontend/ .
RUN npm run build

# Stage 3: runtime (alpine)
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /root/
COPY --from=builder /app/cboard-go .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
EXPOSE 8000
CMD ["./cboard-go"]
```

**Design notes:**

- 🏗️ **Three-stage build**: backend + frontend separated; runtime only ships the binary + dist → minimal image.
- ⚡ **Frontend MUST be built**: backend serves static files from `./frontend/dist`; without it the UI 404s (a defect in the old Dockerfile — fixed).
- 🟢 **Node 20+**: the frontend uses Vite 7; Node 18 fails the build (fixed).
- ⏰ **Timezone**: runtime image ships `TZ=Asia/Shanghai`.
- 🔐 **Certs**: `ca-certificates` for HTTPS outbound (SMTP, payment callbacks, GitHub backup).
- 🗜️ **Trimmed**: `-trimpath -ldflags="-s -w"` shrinks the binary.

#### 🗄️ Optional: switch to MySQL

Default is SQLite (zero-config). For high-concurrency production, switch to MySQL:

**① Edit `docker-compose.yml` — uncomment the mysql service and change the app DATABASE_URL:**

```yaml
services:
  app:
    build: .
    ports:
      - "8000:8000"
    environment:
      - DATABASE_URL=mysql://cboard_user:cboard_password@mysql:3306/cboard_db?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai
      - SECRET_KEY=${SECRET_KEY:?set in .env}
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

**② Set `DATABASE_URL` to the MySQL DSN:**

```env
DATABASE_URL=mysql://cboard_user:cboard_password@mysql:3306/cboard_db?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai
```

> Containers reach MySQL by service name `mysql` (compose network); from the host use `127.0.0.1:3306`.

**③ Restart:**

```bash
docker compose up -d --build
```

**💡 Migrate from SQLite to MySQL** (host Go toolchain):

```bash
go run ./cmd/migrate -sqlite ./data/cboard.db -mysql "cboard_user:cboard_password@tcp(127.0.0.1:3306)/cboard_db?charset=utf8mb4&parseTime=True&loc=Local"
```

> The migration script only **reads** the SQLite source and only **writes** the MySQL target — back up the SQLite file first.

#### 🔧 Docker troubleshooting

| Problem | Solution |
|---------|----------|
| **Port in use** (`bind: address already in use`) | `lsof -i :8000`; kill the process or remap (e.g. `"8001:8000"`) in `docker-compose.yml` |
| **Container refuses to start: `SECRET_KEY` unset** | Set a strong key in `.env`: `SECRET_KEY=$(openssl rand -hex 32)`, then `docker compose up -d` |
| **Frontend 404** | Confirm the image ships dist: `docker compose exec app ls /root/frontend/dist/index.html`; rebuild with `--build` if using an old image |
| **File permission issues** (DB read-only / uploads fail) | `chmod -R 755 data uploads` on host; `chown -R 1000:1000` if needed |
| **Wrong timezone** | Runtime image ships `TZ=Asia/Shanghai`; if you customise the Dockerfile, install `tzdata` and `ENV TZ=Asia/Shanghai` |
| **DB "reset" (data gone)** | Check cwd and `DATABASE_URL`: SQLite relative path is based on container workdir `/root/`; confirm mount matches (`./data:/root/data`) |
| **Slow build / dependency fetch failure** | `export GOPROXY=https://goproxy.cn,direct` before build; npm `--registry=https://registry.npmmirror.com` |
| **Redis errors at startup** | Ignore — Redis auto-disables and the app degrades gracefully |
| **`.env` changes not applied** | Edit host `.env`, then `docker compose up -d` (re-reads env and recreates the container) |

---

### 🖥️ Method 1: Bare VPS (No Panel) — `install-vps.sh`

For Ubuntu/Debian/CentOS without any panel. The script automates: dependency install → code pull → Go/Node.js install → backend compile → frontend build → `.env` generation → Nginx + Let's Encrypt SSL → systemd service → start.

```bash
curl -sL https://raw.githubusercontent.com/moneyfly1/myweb/main/install-vps.sh -o install-vps.sh
sudo bash install-vps.sh
```

Follow the prompts (domain, project directory default `/opt/cboard`, admin username/email/password). Verify:

```bash
systemctl status cboard
curl -s https://yourdomain.com/health
```

Management:

```bash
systemctl start|stop|restart|status cboard
journalctl -u cboard -f            # service logs
tail -f /opt/cboard/server.log     # app logs
```

### 🪟 Method 2: BaoTa (宝塔) Panel — `install.sh`

1. In BaoTa: **Website → Add Site** → bind your domain (PHP type: "Pure Static"). Note the site root (e.g. `/www/wwwroot/example.com`).
2. Place the code in the site root:

```bash
cd /www/wwwroot/example.com
rm -f index.html
git clone https://github.com/moneyfly1/myweb.git .
```

3. Run the installer and choose **option 1** (One-Click Full Auto Deployment):

```bash
chmod +x install.sh
sudo ./install.sh
```

The script installs Go/Node.js, compiles the backend, builds the frontend, configures Nginx reverse proxy, applies Let's Encrypt SSL, registers the systemd service, and starts it.

4. Verify: `https://yourdomain.com` (user), `https://yourdomain.com/admin/login` (admin), `https://yourdomain.com/health`.

> If GitHub cloning fails (mainland China), place the code in the site directory manually and re-run `install.sh`, answering **n** to "Delete and re-download?".

---

## 🎯 Where Admin & Domain Are Configured (all 3 methods)

| Method | Domain configured | Admin configured | When |
|--------|-------------------|------------------|------|
| **BaoTa (`install.sh`)** | **In BaoTa when creating the site** (domain = site directory name; the script derives it via `basename PROJECT_DIR`) | Run the script and choose **menu 2** "Create/Reset Admin Account" (interactive username/email/password) | After deployment, anytime |
| **Bare VPS (`install-vps.sh`)** | **Prompted during the script run** ("域名 (如 example.com)") | **Prompted during the script run** (username/email/password) | During installation |
| **Docker** | `.env` → `PANEL_PUBLIC_URL` (only needed for self-hosted node callbacks; not required for pure subscription) | `.env` → `ADMIN_USERNAME` / `ADMIN_EMAIL` / `ADMIN_PASSWORD`, auto-created at first boot | Before startup, in `.env` |

**Key notes:**

- **BaoTa**: the domain is NOT set in `.env` — it's determined when you create the site in BaoTa (e.g. site dir `/www/wwwroot/your-domain`); the script uses that name for Nginx. Manage the admin via script menu 2.
- **Docker**: the admin comes entirely from `.env` env vars and is auto-created at first boot; every restart re-verifies the password (auto-unlocks a locked-out admin).
- **Bare VPS**: domain and admin are entered interactively during `install-vps.sh`, one shot.

---

## 👤 Admin Account Management

### Creation methods

| Method | Command / Config | When |
|---|---|---|
| **Install script** | `sudo ./install.sh` → menu option 2 | BaoTa / VPS installs |
| **Env bootstrap** | `ADMIN_USERNAME` / `ADMIN_EMAIL` / `ADMIN_PASSWORD` in `.env` | Auto-created on first server boot; re-ensured (password + unlock) on every restart |
| **Admin tool (create/update)** | `go run scripts/admin_tool.go` | Existing deployment; env vars optional (defaults: `admin` / `admin@example.com` / `admin123`) |
| **Docker** | `ADMIN_PASSWORD` in `.env` (auto) — or `docker compose exec app ./cboard-admin` | See Docker section above |

### Reset / unlock

| Operation | Command |
|---|---|
| Reset admin password | `go run scripts/admin_tool.go 'NewStrongPassword123!'` |
| Unlock a locked account | `go run scripts/unlock_user.go admin` (username) or `go run scripts/unlock_user.go user@example.com` (email) |
| Force password + unlock (non-interactive) | Set `ADMIN_PASSWORD` in `.env`, then restart the service/container |

> ℹ️ `ensureDefaultAdmin()` runs at every startup: if the admin does not exist it is created; if `ADMIN_PASSWORD` is set it guarantees the password matches and the account is active/verified/unlocked — a practical self-healing guard against lockouts.

---

## ⚙️ Configuration Guide

Configuration lives in **`.env`** (Viper reads the file, and real environment variables take precedence — `os.Getenv` overrides `.env` values for keys read directly, e.g. `JWT_SECRET_KEY`).

### Core / Server

| Variable | Default | Description |
|---|---|---|
| `HOST` | `0.0.0.0` | Listen address |
| `PORT` | `8000` | Listen port |
| `DEBUG` | `false` | Gin debug mode (verbose SQL logging) |
| `BASE_URL` | *(empty)* | Public base URL (used to build absolute links) |
| `PROJECT_NAME` | `CBoard Modern` | Display name |
| `VERSION` | `1.0.0` | Display version |
| `API_V1_STR` | `/api/v1` | API prefix |
| `WORKERS` | `4` | Worker pool size |
| `OPTIMIZE_FOR_LOW_END` | `true` | Low-end hardware tuning |
| `DISABLE_SCHEDULE_TASKS` | `false` | Disable background scheduled tasks |
| `TRUSTED_PROXIES` | *(empty)* | Comma-separated trusted proxy IPs for `GetRealClientIP` |
| `ENV` | *(empty)* | `production`/`prod` enables strict validation (strong secrets required) |

### Security / JWT 🔑

| Variable | Default | Description |
|---|---|---|
| `SECRET_KEY` | *(generated if empty)* | **MUST be ≥ 32 bytes, random, non-placeholder** in production |
| `JWT_SECRET_KEY` | *(overrides SECRET_KEY)* | Alias with highest priority |
| `JWT_ALGORITHM` | `HS256` | Signing algorithm |
| `JWT_EXPIRE_HOURS` | `1` | Access token lifetime (hours) |
| `REFRESH_TOKEN_EXPIRE_DAYS` | `30` | Refresh token lifetime (days) |
| `BACKEND_CORS_ORIGINS` | localhost list | Comma-separated CORS whitelist — **no `*`/`null` allowed** |

### Database 🗄️

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `sqlite:///./data/cboard.db` | SQLite (Docker default, data in `./data`); containing `mysql` = MySQL; containing `postgresql` = PostgreSQL |
| `USE_MYSQL` | *(empty)* | `true` forces MySQL driver |
| `USE_POSTGRES` | *(empty)* | `true` forces PostgreSQL driver |
| `MYSQL_HOST` / `MYSQL_PORT` | `localhost` / `3306` | MySQL connection |
| `MYSQL_USER` / `MYSQL_PASSWORD` / `MYSQL_DATABASE` | `cboard_user` / — / `cboard_db` | MySQL credentials (strong password required in production) |
| `POSTGRES_SERVER` / `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | `localhost` / `postgres` / — / `cboard` | PostgreSQL connection |

### Redis ⚡ (Optional)

| Variable | Default | Description |
|---|---|---|
| `REDIS_ADDR` | *(empty)* | e.g. `localhost:6379`; empty ⇒ caching disabled gracefully |
| `REDIS_PASSWORD` | *(empty)* | Redis password if set |
| `REDIS_DB` | `0` | Redis logical database |

Quick Redis via Docker: `docker run -d --name redis -p 6379:6379 redis:alpine`

### Email / SMTP ✉️

| Variable | Default | Description |
|---|---|---|
| `SMTP_HOST` | *(empty)* | SMTP server |
| `SMTP_PORT` | `587` | SMTP port |
| `SMTP_USERNAME` / `SMTP_PASSWORD` | *(empty)* | SMTP credentials |
| `SMTP_FROM_EMAIL` / `SMTP_FROM_NAME` | *(empty)* / `CBoard Modern` | Sender identity |
| `SMTP_ENCRYPTION` | `tls` | `tls`/`ssl` enables TLS; anything else = plaintext |

### Admin bootstrap 👤

| Variable | Default | Description |
|---|---|---|
| `ADMIN_USERNAME` | `admin` | Auto-created admin username |
| `ADMIN_EMAIL` | `admin@example.com` | Auto-created admin email |
| `ADMIN_PASSWORD` | `admin123` (dev) / required (prod) | Auto-created/reset admin password (≥ 6 chars) |

### Uploads / Misc

| Variable | Default | Description |
|---|---|---|
| `UPLOAD_DIR` | `uploads` | Upload/asset/log directory |
| `MAX_FILE_SIZE` | `10485760` | Max upload size in bytes (10 MB) |
| `SUBSCRIPTION_URL_PREFIX` | *(empty)* | Custom prefix for subscription URLs |
| `DEVICE_LIMIT_DEFAULT` | `3` | Default device limit per user |
| `DEVICE_UPGRADE_PRICE_PER_MONTH` / `_PER_YEAR` / `_BASE_DEVICES` | `10` / `200` / `5` | Device-upgrade pricing model |
| `GEOIP_DB_PATH` | `./GeoLite2-City.mmdb` | GeoIP MMDB path (auto-downloaded if missing) |

> 💳 **Payment gateway credentials** (Alipay app id/keys, notify/return URLs) are configured in the admin panel (**PaymentConfig** page) and stored in the database; legacy bootstrap env vars (`ALIPAY_APP_ID`, `ALIPAY_PRIVATE_KEY`, `ALIPAY_PUBLIC_KEY`, `ALIPAY_NOTIFY_URL`, `ALIPAY_RETURN_URL`) are also honored.

---

## 🗄️ Database Backup

### SQLite (default)

**Cold copy (while stopped)** — always consistent:

```bash
cd /opt/cboard            # or your project dir
systemctl stop cboard
cp cboard.db "cboard.db.backup.$(date +%Y%m%d_%H%M%S)"
systemctl start cboard
```

**Hot backup (service running)** — use the SQLite online backup API so the WAL journal is included safely:

```bash
sqlite3 cboard.db ".backup '/backup/cboard_$(date +%Y%m%d).db'"
```

**Automated backup script (cron):**

```bash
#!/bin/bash
# /etc/cron.daily/cboard-backup
cd /opt/cboard
BACKUP_DIR="/backup/cboard"
mkdir -p "$BACKUP_DIR"
sqlite3 cboard.db ".backup '$BACKUP_DIR/cboard_$(date +%Y%m%d_%H%M%S).db'"
find "$BACKUP_DIR" -name "cboard_*.db" -mtime +7 -delete   # keep 7 days
```

> 💡 **Docker:** the SQLite file is bind-mounted at `./cboard.db` on the host — back it up there exactly as above (plus `uploads/`). Stop the container first for a clean copy: `docker compose stop app`.

> 🛟 **Self-healing:** CBoard performs a SQLite integrity self-check at startup; if the file is corrupt, it **automatically restores from the most recent backup** instead of failing to boot (recovery logs tell you which backup was used).

### MySQL

```bash
docker compose exec mysql mysqldump -u root -p cboard_db > cboard_$(date +%Y%m%d).sql
# or, for bare-metal:
mysqldump -u cboard_user -p cboard_db > cboard_$(date +%Y%m%d).sql
```

### Panel-managed backup & repo sync

The admin panel provides **Backup settings** (Settings → Backup): scheduled automatic backups plus **GitHub / Gitee repo sync** — push backups to a private repository automatically. Configure tokens and a schedule in the admin UI.

### What to back up

| Artifact | Location | Required? |
|---|---|---|
| SQLite database | `./cboard.db` (or MySQL dump) | ✅ Yes |
| Uploads / assets | `./uploads/` | ✅ Yes (avatars, GeoIP, logs) |
| `.env` | project root | ✅ Yes (secrets — store securely!) |

---

## 🔧 Troubleshooting

### General

| Symptom | Check / Fix |
|---|---|
| Service won't start | `journalctl -u cboard -f`; verify `.env`, port 8000 free (`ss -tlnp \| grep 8000`), disk space |
| 502 Bad Gateway (behind Nginx) | `systemctl status cboard`; confirm `proxy_pass http://127.0.0.1:8000`; check `netstat -tlnp \| grep 8000` |
| "⚠️ 未找到现有数据库文件，即将创建【全新】数据库" | `DATABASE_URL` points to a path where no DB exists — **do not restart blindly**; restore the real file or fix the path |
| Admin password lost | `go run scripts/admin_tool.go 'NewPassword123!'`, or set `ADMIN_PASSWORD` in `.env` and restart |
| Account locked after failed logins | `go run scripts/unlock_user.go <username-or-email>` |
| Redis connection failed | `systemctl status redis`; `redis-cli ping` (expect `PONG`); ensure `REDIS_ADDR`/`REDIS_PASSWORD` match |
| SSL certificate failed | Domain must resolve to the server; port 80 must be open for Let's Encrypt |

### Docker-specific

| Symptom | Check / Fix |
|---|---|
| **Port conflict** — `Error starting userland proxy: listen tcp 0.0.0.0:8000: bind: address already in use` | Something already uses 8000. Change the host port mapping in `docker-compose.yml`, e.g. `"8001:8000"`, then `docker compose up -d`; access `http://SERVER:8001` |
| **Permission denied on `cboard.db`** | The container writes as `root`; if the host file is owned by another user, `sudo chown -R root:root cboard.db uploads` (or `chmod 660`). The startup log will say if it cannot open the DB |
| **Timezone wrong in logs** | The image already sets `ENV TZ=Asia/Shanghai`; for other zones add `TZ=Your/Zone` to the app `environment:` (or `-e TZ=UTC`) |
| Container keeps restarting | `docker compose logs app`; typical causes: weak `SECRET_KEY` in `ENV=production`, DB file not writable, MySQL not ready yet (add `depends_on`/retry when using MySQL) |
| **Fresh DB created unexpectedly in Docker** | Working directory inside the container is `/root/` and `DATABASE_URL=sqlite:///./data/cboard.db` resolves to `/root/data/cboard.db` — which is the bind mount `./data`. If the host dir is missing/renamed, a new DB appears; restore from backup |
| MySQL "connection refused" at startup | MySQL container still booting; wait, or add `depends_on: mysql` + healthcheck; verify `MYSQL_HOST=mysql` (service name), not `localhost` |
| WAL files growing (`cboard.db-wal/-shm`) | Normal SQLite WAL behavior; checkpointed automatically. Back up with `.backup` (hot) or after `docker compose stop app` |
| Image build fails on `go mod download` | Network issue pulling modules — retry, or set `GOPROXY=https://goproxy.cn,direct` as a build arg |

### Log locations

| Environment | Paths |
|---|---|
| systemd (VPS/BaoTa) | `journalctl -u cboard -f`, `server.log` in project dir, `uploads/logs/app.log` |
| Docker | `docker compose logs -f app`; app log inside container at `/root/uploads/logs/app.log` |

---

## 📄 License

This project is licensed under the **MIT License**.

---

**Version**: v1.2.0 · **Status**: ✅ Production Ready (SQLite + optional Redis/MySQL) · **Last updated**: 2026-03-05

*CBoard — built for people who share what they have, and protect what they share.* 🛡️
