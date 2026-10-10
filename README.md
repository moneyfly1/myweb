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
- [Nginx Reverse-Proxy Template (final)](#-nginx-reverse-proxy-template-final--includes-attachment-forwarding)
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
| Bare VPS (Ubuntu/Debian/CentOS) | ✅ `install.sh` one-click script (auto-detects the environment) |
| BaoTa (宝塔) Panel | ✅ `install.sh`, or the BaoTa entry point `bt-deploy.sh` (thin wrapper, identical behaviour) |
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
| **Dashboard** | Real-time overview: users, orders, revenue, active subscriptions, **live user-activity feed** (registrations / subscription resets / orders / recharges, 60s polling), 7-day expiring customers |
| **Users** | Filter/search users, edit, disable, **batch operations**, **reset password**, **login as user**, **send email**, view check-in logs, GeoIP info |
| **AbnormalUsers** | Flagged/abnormal accounts (locked, inactive, unusual login patterns) for review |
| **Nodes** | Regular node CRUD, node collection from upstream subscription URLs, manual import (link / Clash config / manual entry), **batch test**, **deduplication** (Type:Server:Port), region grouping |
| **CustomNodes** | Custom node definitions: link import / **subscription-URL import (auto-parse)** / manual creation, **subscription refresh & replacement (incremental update that preserves user assignments)**, assign/unassign, expiry management, latency test |
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
- ✅ Node management: regular / dedicated / custom (link & **subscription-URL import**, refresh & replace with assignment preservation) / **self-hosted** / batch test / import
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
| 🖥️ Node types | Regular (collected/imported), dedicated, custom (link / subscription-URL import), self-hosted (SSH) |
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
| **Brute-force defense** | Per-IP + per-account **login rate limiting** with progressive lockout; locked accounts can be unlocked with `scripts/unlock_user` (admin account is force-unlocked on restart when `ADMIN_PASSWORD` is set) |
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
| 🖥️ **One-Click Script** | Bare VPS **or** BaoTa panel — it detects the environment | `install.sh` (BaoTa entry: `bt-deploy.sh`) | Low (one command) |
| 🪟 **BaoTa panel site + script** | You want the panel to own the site/certificates | panel "Add Site" + `install.sh` / `bt-deploy.sh` | Low (menu-driven) |

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

### 🖥️ Method 1: One-Click Script — `install.sh` (bare VPS **or** BaoTa)

**This is the only maintained installer.** It auto-detects the environment (BaoTa nginx + panel vhost dir, or system nginx), and on a clean machine it bootstraps everything it needs: git/sqlite3, gcc, Go 1.25.0, Node 22.12.0, Nginx and certbot.

#### Which of the three scripts should I use?

| Script | Status | Purpose |
|--------|--------|---------|
| **`install.sh`** | ✅ **The only implementation — use this** | Deploy + operate + self-repair + full uninstall; works on bare VPS and BaoTa |
| **`bt-deploy.sh`** | ✅ BaoTa entry point (thin wrapper) | Checks the BaoTa environment and certificate policy, then hands over to `install.sh` — behaviour is identical |

#### Prerequisites

| Item | Requirement |
|------|-------------|
| OS | Debian 10+ / Ubuntu 18.04+ / CentOS 7+ / Rocky / AlmaLinux (Debian 12 verified end-to-end) |
| Privileges | root |
| Resources | ≥1 vCPU, ≥1 GB RAM, ≥10 GB disk |
| Domain | A record pointing at this server. **The project directory name must equal the domain**, e.g. `/www/wwwroot/pingzen.top` |
| Ports | 80 and 443 open (cloud security group **and** host firewall; the script does not modify firewalls) |

#### Quick start (three commands)

```bash
git clone https://github.com/moneyfly1/myweb.git /www/wwwroot/your-domain.com
cd /www/wwwroot/your-domain.com
sudo bash install.sh        # then choose menu option 1, then answer the Redis prompt
```

Non-interactive (the script is safe on EOF — it never busy-loops):

```bash
printf '1\n' | sudo bash install.sh     # 1 = full auto deploy
```

> **The deploy never blocks on the Redis question**: Redis setup runs in "auto" mode during deployment —
> non-interactive runs skip it, interactive runs wait at most 20 seconds and then continue.
> Enable it later any time with **menu 12** (GeoIP lookups get 50–100x faster) — menu 12 takes care of
> installing Redis for you: **Docker** (Docker itself is installed automatically when missing) or
> **apt/yum packages**, and if one route fails to bring Redis up the script automatically tries the other
> (verified case: a host where apt's `redis-server` dies with `libjemalloc.so.2: failed to map segment`
> — the script silently switches to Docker and finishes). Redis listens on `127.0.0.1` only.

#### What the script does

| Stage | Details |
|-------|---------|
| Bootstrap | base packages (git/sqlite3/wget/curl) → gcc (required by CGO/SQLite) → **Go 1.25.0** → **Node 22.12.0** → **Nginx** (reuses BaoTa's nginx when present) |
| Source & env | clones the repo if the directory is empty; writes `.env` with `HOST=127.0.0.1`, an **absolute** database path, a random 64-char `SECRET_KEY`, mode 600 |
| Build | backend built with `CGO_ENABLED=1` into `server.new`, health-checked, then swapped in (previous binary kept as `server.bak.<ts>`); frontend installed from lockfile and built with vite |
| Service | systemd unit with `EnvironmentFile` / `LimitNOFILE` / `NoNewPrivileges`, then a **business health check** (`/health`, not just `is-active`) |
| Nginx | one template: `/api/`, `/uploads/` (attachments — otherwise the SPA fallback returns HTML), `/repo-sync/`, `/assets/` long cache, `index.html` no-cache, `client_max_body_size 16m`, ACME challenge location; `nginx -t` first, auto-switch http2 syntax or **precisely roll back this change** on failure |
| Certificates | reuse first (certbot → BaoTa panel → acme.sh), request only if none; installs the renewal reload hook and timer |
| Finishing | logrotate for `server.log`; database backed up to `/www/backup/cboard` before every upgrade; prints the **login URLs + admin username/password** (with a live login check) |

#### Menu reference (0–16)

`1` full auto deploy · `2` create/reset admin (verifies login immediately) · `3` force restart (project processes only) · `4` deep cache clean (keeps `dist` and the binary) · `5` unlock user · `6` service status · `7` live log (`server.log`) · `8` restart · `9` stop · `10` renew certificate · `11` sync from GitHub & rebuild · `12` configure Redis (auto-installs Docker, falls back to apt/yum, never blocks) · `13` roll back to previous build · `14` rebuild & restart only · `15` full uninstall (with residual scan) · `16` self-check & auto-repair · `0` exit

#### Certificates vs. BaoTa's own SSL

They **do** conflict if both manage the same domain: two ACME clients re-issuing the same host (rate limits), different file names (`fullchain.cer` vs `fullchain.pem`), and the panel rewriting the vhost. The script handles it: it probes **certbot → BaoTa (`/www/server/panel/vhost/cert/<domain>`) → acme.sh (`/root/.acme.sh/<domain>`)** and **reuses an existing BaoTa certificate, skipping certbot**; it renders the right file names per source; it warns when two certificate sets coexist; menu 16 re-adds any config fragments the panel overwrote. Set `CERT_MANAGER=panel` to let the panel own issuance/renewal (apply in the panel, then re-run the script or menu 16 to switch the site to HTTPS), or keep the default and only *view* certificates in the panel.

#### Login details are printed after install

Menu 1 (and menu 2 / menu 8) prints the front-end and admin login URLs, the admin username/e-mail, and the password.
The password is generated by the script and written to `ADMIN_PASSWORD` in `.env`, and the app **resets that admin's
password to it on every start** — so the printed password keeps working after restarts (unlike the app's built-in
"random password printed once in `server.log`" behaviour).

```text
================== 登录信息 ==================
  前台登录:     https://your-domain.com/login
  管理员登录:   https://your-domain.com/admin/login
  管理员账号:   admin
  管理员密码:   <16-char random>
==============================================
✅ 已实测：用上面这个账号密码登录成功
```

To use your own password: set `ADMIN_PASSWORD` in `.env` and restart (menu 8), or use menu 2 (interactive reset with a live login check).

#### Verify

```bash
curl -I https://your-domain.com                 # 200 + valid certificate
curl https://your-domain.com/api/v1/packages    # API proxy works
systemctl status cboard
tail -f /www/wwwroot/your-domain.com/server.log
```

### 🪟 Method 2: BaoTa (宝塔) Panel — `bt-deploy.sh`

**Use when** you have (or want) the BaoTa panel and want the panel to own the site, certificates and logs.

> 📖 **Full step-by-step guide** (panel installation, nginx compile wait, both certificate modes, verified pitfalls): [`docs/部署/宝塔部署教程.md`](docs/部署/宝塔部署教程.md)

#### Quick steps

```bash
# 1) Install the BaoTa panel if you don't have it yet.
#    ⚠️ Always pipe `yes`: BT's own installer busy-loops at 99% CPU when stdin is closed (verified).
wget -O /root/bt_install.sh https://download.bt.cn/install/install_lts.sh && yes | bash /root/bt_install.sh
cat /tmp/btpanel-install.log          # panel URL / username / password

# 2) Panel → App Store → install Nginx (compiled from source; 25–40 min on 2 vCPU is normal)

# 3) Panel → Websites → Add site: domain = your domain, root kept at /www/wwwroot/<domain>, PHP = static
#    ⚠️ The directory name MUST equal the domain (the script derives the domain from it)

# 4) In the site directory, download just the two scripts (the script clones the repo itself)
cd /www/wwwroot/your-domain.com
rm -f index.html 404.html .user.ini
curl -fsSLO https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
curl -fsSLO https://raw.githubusercontent.com/moneyfly1/myweb/main/bt-deploy.sh

# 5) One-click deploy (choose menu 1; answer n if you have no Redis)
bash bt-deploy.sh
```

#### Certificates — pick ONE (never both)

| Mode | How | Renewal owner |
|------|-----|---------------|
| **A. Panel-managed (recommended)** | `CERT_MANAGER=panel bash bt-deploy.sh` (hands SSL back to the panel) → panel → Website → SSL → Let's Encrypt → Apply | The **panel** renews automatically; the script only reuses |
| **B. Script-managed (default)** | Just run step 5 | `certbot.timer`; do **not** click Apply/Renew in the panel (it has no order for that cert) — use menu 10 instead |

Why mode A needs the hand-over: BaoTa checks whether `ssl_certificate` already exists in the config; if it does it
assumes SSL is already on and **skips deploying its own certificate** — the panel then reports success but
`vhost/cert/<domain>/` stays empty and later renewals have nothing to renew. `CERT_MANAGER=panel` clears the
script-written certificate section so the panel can fully take over.

#### Login details are printed when the deploy finishes

```text
================== 登录信息 ==================
  前台登录:     https://your-domain.com/login
  管理员登录:   https://your-domain.com/admin/login
  管理员账号:   admin
  管理员密码:   <16-char random>
==============================================
✅ 已实测：用上面这个账号密码登录成功
```

The password is written to `ADMIN_PASSWORD` in `.env` and the app resets that admin's password to it on every
start, so it keeps working after restarts. Change it in `.env` + menu 8, or use menu 2.

#### BaoTa-specific notes

| Topic | Detail |
|-------|--------|
| Config ownership | The script uses **merge mode**: the panel's `#SSL-START` / `#CERT-APPLY-CHECK` markers and includes are preserved; only app-level fragments are injected (`/api/`, `/uploads/`, SPA fallback, `index.html` no-cache, ACME location, `client_max_body_size`), and `root` is pointed at `frontend/dist` |
| Panel rewrites the vhost | "Save site settings" / "renew SSL" in the panel rewrites the file → run **menu 16** to re-inject (verified: injections and 443 restored, panel markers intact) |
| `nginx -t` | The script validates the **running** BT nginx (`/www/server/nginx/sbin/nginx -t`); BT nginx is managed by `/etc/init.d/nginx`, so the script detects it with `pgrep` and never starts a second nginx on port 80 |
| Certificates | Panel certs live in `/www/server/panel/vhost/cert/<domain>/`; the script detects them (source = baota) and **skips certbot**; coexistence of two certificate sets triggers a warning |
| File permissions | BT nginx runs as `www`, so the script makes `frontend/dist`, `uploads`, `.well-known` readable and chowns them (otherwise 403 / ACME validation failures) |
| Panel site record | The script does not register a site in the panel — create the site in the panel (step 3) if you want it managed there |
| Firewall | Open 80/443 in the panel's firewall (the script does not modify firewalls) |

#### Post-install operations

| Task | How |
|------|-----|
| Everything | `bash bt-deploy.sh` or `bash install.sh` (menus 2/3/4/11/13/15/16 …) |
| Upgrade | Menu **11** (sync from GitHub & rebuild: DB backed up first, atomic binary swap, health check) |
| Rollback | Menu **13** |
| Self-check & repair | Menu **16** (re-inject fragments after the panel rewrites the config) |
| Full uninstall | Menu **15** (with residual scan) |
| Logs | Menu 7, or `tail -f /www/wwwroot/your-domain.com/server.log` |

## 🎯 Where Admin & Domain Are Configured (all 3 methods)

| Method | Domain configured | Admin configured | When |
|--------|-------------------|------------------|------|
| **One-click script (`install.sh` / `bt-deploy.sh`)** | Derived from the **project directory name** (e.g. `/www/wwwroot/example.com`); override with `DOMAIN=example.com` | Run the script, choose **menu 2** (username/email/password; login is verified immediately) | After deployment, anytime |
| **BaoTa panel site + script** | Determined **when adding the site in the panel** | Same (menu 2) | After deployment, anytime |
| **Docker** | `.env` → `PANEL_PUBLIC_URL` (only needed for self-hosted node callbacks; not required for pure subscription) | `.env` → `ADMIN_USERNAME` / `ADMIN_EMAIL` / `ADMIN_PASSWORD`, auto-created at first boot | Before startup, in `.env` |

**Key notes:**

- **BaoTa**: the domain is NOT set in `.env` — it's determined when you create the site in BaoTa (e.g. site dir `/www/wwwroot/your-domain`); the script uses that name for Nginx. Manage the admin via script menu 2.
- **Docker**: the admin comes entirely from `.env` env vars and is auto-created at first boot; every restart re-verifies the password (auto-unlocks a locked-out admin).
- **One-click script**: the domain comes from the project directory name (`/www/wwwroot/<domain>`), so clone into a correctly named directory.

---

## 👤 Admin Account Management

### Creation methods

| Method | Command / Config | When |
|---|---|---|
| **Install script** | `sudo ./install.sh` → menu option 2 | BaoTa / VPS installs |
| **Env bootstrap** | `ADMIN_USERNAME` / `ADMIN_EMAIL` / `ADMIN_PASSWORD` in `.env` | Auto-created on first server boot; re-ensured (password + unlock) on every restart |
| **Admin tool (create/update)** | `go run scripts/admin_tool` | Existing deployment; env vars optional (defaults: `admin` / `admin@example.com` / `admin123`) |
| **Docker** | `ADMIN_PASSWORD` in `.env` (auto) — or `docker compose exec app ./cboard-admin` | See Docker section above |

### Reset / unlock

| Operation | Command |
|---|---|
| Reset admin password | `go run scripts/admin_tool 'NewStrongPassword123!'` |
| Unlock a locked account | `go run scripts/unlock_user admin` (username) or `go run scripts/unlock_user user@example.com` (email) |
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

Three ways to get Redis (menu 12 does all of this for you):

| Route | What the script does |
|---|---|
| Docker (default) | installs Docker if missing (`apt/dnf/yum docker.io`, falling back to `get.docker.com`), then runs the `redis` container with `--restart=always -p 127.0.0.1:6379:6379` (never exposed publicly) |
| System packages | `apt-get install redis-server redis-tools` (or `dnf`/`yum install redis`), `systemctl enable --now`, then waits for `PING` |
| Skip | `.env` is left untouched and the app keeps running without cache |

Manual equivalent: `docker run -d --name redis --restart=always -p 127.0.0.1:6379:6379 redis:alpine`

> Both install routes verify `PING` before declaring success, and each one **falls back to the other** if it
> fails — the deployment itself is never aborted by a Redis problem. Re-running menu 12 is idempotent
> (a marked block in `.env` is replaced in place, no duplicated lines, existing container reused).

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

## 🌐 Nginx Reverse-Proxy Template (final — includes attachment forwarding)

> ⚠️ Use the **complete template below** in production. If you omit
> `location /uploads/`, requests for uploaded attachments fall through to the
> SPA fallback and return `index.html` — the browser then fails to open images
> as HTML, and if Cloudflare/CDN sits in front it may even cache that wrong
> HTML *as the image content*.

```nginx
# Reverse proxy CBoard (Go backend on 127.0.0.1:8000; static assets served by nginx)
server {
    listen 80;
    server_name yourdomain.com;
    # For production add 443 + SSL (certbot / panel-issued)

    # Upload limit must exceed the backend per-file cap (default 30 MB)
    client_max_body_size 50m;

    root /path/to/cboard/frontend/dist;   # frontend build output
    index index.html;

    # ① Backend API → Go
    location /api/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        # For SSE live logs / long-lived connections:
        proxy_buffering off;
        proxy_read_timeout 3600s;
    }

    # ② Uploaded attachments (ticket images/videos/documents, avatars, …) → Go's /uploads static service.
    #    CRITICAL: don't let the generic `location /` try_files swallow this —
    #    it would return index.html for image URLs!
    #    no-cache: attachments may be replaced/deleted; avoids stale CDN/browser copies.
    location /uploads/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        expires -1;
    }

    # ③ Hashed frontend assets (long cache) + SPA fallback
    location /assets/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

**If you also run Cloudflare/CDN in front of Nginx**: after fixing Nginx,
historical attachment URLs that were cached with the wrong HTML response may
still be served from the CDN edge. **Purge the CDN cache** once from the panel.
The app already appends `?raw=1` to attachment URLs as an independent cache
key, so new URLs are never affected by stale dirty caches.

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
| Admin password lost | `go run scripts/admin_tool 'NewPassword123!'`, or set `ADMIN_PASSWORD` in `.env` and restart |
| Account locked after failed logins | `go run scripts/unlock_user <username-or-email>` |
| Redis connection failed | `systemctl status redis`; `redis-cli ping` (expect `PONG`); ensure `REDIS_ADDR`/`REDIS_PASSWORD` match |
| Upgrade says "site config is missing the ACME challenge passthrough" | Fixed: the old check grepped `location \.well-known/acme-challenge` while the real config says `location ^~ /.well-known/acme-challenge/` — the `^~` modifier made it a false positive on every upgrade. It now accepts `^~` / `=` / `~` / bare paths, and if the block is genuinely missing it is **added automatically** (inside the marked block, idempotent) |
| BaoTa "force HTTPS" and certificate renewal | A real failure mode: BT's redirect is a server-level `if + rewrite` that runs **before** location matching, so no `location` can exempt it and `http://domain/.well-known/acme-challenge/<token>` always 301s. While the cert is valid LE follows the redirect; once it expires the renewal can fail. The script installs `renewal-hooks/{pre,post}` hooks that temporarily comment the redirect out, **poll until the change is really live**, and restore the file byte-for-byte afterwards (immediate rollback if `nginx -t` fails). Verified with `certbot renew --dry-run` |
| Vhost filename differs from the domain (e.g. `speedora.conf` serving `speedora.top`) | The script now **adopts the config file by `server_name`** (skipping `.bak/.backup` copies) instead of creating a second file that produces duplicate `server_name` blocks |
| Redis won't install or won't start | Run **menu 12** — both routes (Docker / apt-yum) are tried automatically and each falls back to the other, so the deploy never stops on Redis. Debug manually with `journalctl -u redis-server -n 20` or `docker logs redis`; re-running menu 12 is idempotent (existing container reused, `.env` block replaced in place) |
| Docker route says it needs Docker | It doesn't any more: menu 12 → `1` installs Docker itself (`apt/dnf/yum docker.io`, then `get.docker.com`), starts the daemon and only then runs the `redis` container |
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
