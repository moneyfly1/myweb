# CBoard - Modern Subscription Management System

[中文](README_zh.md) | English

---

## 📖 Overview

**CBoard** is a modern, high-performance subscription management system designed for VPN/proxy service providers. Built with Go language, it offers **70-90% memory reduction** compared to Python-based solutions while maintaining full feature parity.

### 💡 Project Origin

This project was initially created to solve a practical need: **securely sharing subscription resources from multiple proxy services with friends while preventing resource abuse**.

**Core Scenario:**
- You have multiple proxy service subscriptions with unused traffic, and want to share them with friends (e.g., for international trade)
- The purchased proxy service packages typically don't limit device count, but you need to control the sharing scope
- Use device count limits to prevent friends from sharing subscriptions with others
- However, there's a problem: friends might download node information and use it separately, losing control

**Solution:**
- The system supports periodic subscription address resets (recommended daily or every two days)
- Automatically collects and aggregates subscription addresses from multiple proxy services
- Generates new aggregated subscription links to share with friends
- Effectively prevents resource abuse through device limits and periodic subscription address updates

**Use Cases:**
- Personal Sharing: Share proxy resources with a small group of friends while controlling usage scope
- Small-scale Commercialization: If you have many friends, you can also operate on a small commercial scale

This design ensures effective resource utilization while preventing resource abuse and leakage through technical means.

### 🎯 Key Features

#### Core Features
- 🚀 **High Performance**: Memory usage only 35-95 MB (vs 300-850 MB in Python version)
- ⚡ **Fast Startup**: Millisecond-level startup time
- 🔒 **Secure**: JWT authentication, password encryption, SQL injection protection
- 📦 **Feature Complete**: All core business functions included
- 🎨 **Modern Frontend**: Vue 3 + Element Plus, responsive design
- 🐳 **Easy Deployment**: One-click VPS script (`install-vps.sh`) or BT Panel script (`install.sh`)
- ⚡ **Redis Cache**: Optional Redis caching for 50-100x performance boost on hot data

#### Business Features
- 💳 **Multi-Payment**: Alipay, WeChat Pay, PayPal, Apple Pay, Yipay
- 👥 **User Management**: Complete user system with levels, invites, and rewards
- 📊 **Analytics Dashboard**: DAU/WAU/MAU statistics, retention analysis, churn prediction
- 🎫 **Ticket System**: Built-in customer support system
- 📚 **Knowledge Base**: Complete help documentation with Clash series tutorials
- 🎁 **Daily Check-in**: Random rewards (0.1-1 CNY) for user engagement
- 🎉 **Promotions**: Flexible marketing campaigns (flash sales, new user offers, member days)
- 📱 **Mobile Optimized**: Full responsive design with drawer components
- ⚙️ **Clash Config**: Professional Clash subscription system with 16 proxy groups and 3376 routing rules

---

## 🏗️ Technology Stack

### Backend
- **Web Framework**: [Gin](https://github.com/gin-gonic/gin) - High-performance HTTP web framework
- **ORM**: [GORM](https://gorm.io/) - The fantastic ORM library for Go
- **Database**: SQLite (default) / MySQL 5.7+ / PostgreSQL 12+
- **Cache**: Redis (optional, highly recommended for production)
- **Authentication**: JWT (JSON Web Tokens)
- **Configuration**: Viper
- **Language**: Go 1.21+

### Frontend
- **Framework**: Vue 3 (Composition API)
- **UI Library**: Element Plus
- **Build Tool**: Vite
- **State Management**: Pinia
- **Router**: Vue Router 4

---

## 📋 System Requirements

### Minimum Requirements
- **CPU**: 1 core (2+ cores recommended)
- **Memory**: 512 MB (1 GB+ recommended)
- **Disk**: 10 GB (20 GB+ recommended)
- **OS**: Ubuntu 18.04+ / Debian 10+ / CentOS 7+

### Software Requirements
- **Go**: 1.21+ (auto-installed by install script)
- **Node.js**: 16+ (for frontend build)
- **Nginx**: (included with BT Panel or auto-installed by VPS script)
- **Database**: SQLite (default, no installation needed) or MySQL/PostgreSQL
- **Redis**: Optional but highly recommended for production (auto-configured by install script)

---

## 🚀 Installation

### Installation Methods Overview

| Item | Method 1: VPS (No BT Panel) | Method 2: BT Panel |
|------|----------------------------|-------------------|
| **Use Case** | New VPS, no BT Panel | Server with BT Panel installed |
| **Script** | `install-vps.sh` | `install.sh` |
| **Project Dir** | Default `/opt/cboard` (customizable) | BT site root, e.g., `/www/wwwroot/yourdomain.com` |
| **Setup** | Script auto-installs Nginx, Go, Node.js, Certbot | Create site in BT Panel first, then place code in site directory |
| **Automation** | One command to download and run, follow prompts | Create site → Place code → Run script → Select menu option 1 |

**How to Choose:**

- **No BT Panel**: Use **Method 1**. SSH into VPS and run one command to start installation. Script auto-installs all dependencies and deploys.
- **Have BT Panel**: Use **Method 2**. Create a site in BT Panel for your domain, place code in site directory, then run `install.sh` and select "One-Click Full Auto Deployment".

---

### Method 1: VPS Installation (No BT Panel) - Using install-vps.sh

**For**: Ubuntu / Debian / CentOS without BT Panel. Script auto-installs Nginx, Go, Node.js, Certbot, and completes deployment.

#### Prerequisites

| Item | Requirement |
|------|-------------|
| OS | Ubuntu 18.04+ / Debian 10+ / CentOS 7+ |
| Specs | At least 1 core CPU, 512 MB RAM, 10 GB disk |
| Domain | Bound and DNS resolved to server IP |
| Ports | 80, 443 open |

#### Installation Steps

**1. Download and Run Script (requires root)**

```bash
curl -sL https://raw.githubusercontent.com/moneyfly1/myweb/main/install-vps.sh -o install-vps.sh
sudo bash install-vps.sh
```

**2. Follow Prompts**

- **Domain**: e.g., `example.com` (must be resolved)
- **Project Directory**: Press Enter for default `/opt/cboard`, or enter custom path
- **Admin Username / Email / Password**: Fill as needed (email required)

**3. Automatic Execution**

Script will: Install dependencies → Pull code from GitHub → Install Go, Node.js → Compile backend, build frontend → Generate `.env` → Configure Nginx, apply SSL → Create systemd service and start.

**4. Verify**

- Frontend: `https://yourdomain.com`
- Admin Panel: `https://yourdomain.com/admin/login`
- Health Check: `https://yourdomain.com/health`

**If GitHub Clone Fails in China**: Manually place code in installation directory (e.g., `/opt/cboard`), then re-run script and select **n** when asked "Delete and re-download?". See: [VPS Deployment Guide - Continue When Clone Fails](./docs/VPS部署教程-无宝塔.md#克隆失败时如何继续代码下载失败).

#### Post-Installation Management (No BT Panel)

- **Project Directory**: Default `/opt/cboard` (replace with actual path if changed during installation)
- **Service**: `systemctl start/stop/restart/status cboard`
- **Logs**: `journalctl -u cboard -f` or `tail -f /opt/cboard/server.log`
- **Configuration**: Edit `/opt/cboard/.env` then run `systemctl restart cboard`

For more troubleshooting, see [Installation Troubleshooting Guide](./docs/故障排查/安装问题排查指南.md) or [VPS Deployment Guide (No BT Panel)](./docs/VPS部署教程-无宝塔.md).

---

### Method 2: BT Panel Installation - Using install.sh

**For**: Servers with BT Panel installed. Create a site in BT Panel for your domain first, place code in site root directory, then run `install.sh` to complete compilation and Nginx configuration.

#### Prerequisites

| Item | Requirement |
|------|-------------|
| BT Panel | Installed (version 7.0+ recommended) |
| OS | Ubuntu 18.04+ / Debian 10+ / CentOS 7+ |
| Specs | At least 1 core CPU, 512 MB RAM, 10 GB disk |
| Domain | Site created in BT Panel and bound |

#### Installation Steps

**1. Create Website in BT Panel**

- Login to BT Panel → **Website** → **Add Site**
- Enter domain (e.g., `example.com`), root directory typically `/www/wwwroot/example.com`
- No need to create FTP/database, PHP select "Pure Static" or any
- Note the **site root directory**, code will be placed here

**2. Place Code in Site Directory**

Choose one method:

- **SSH**: `cd /www/wwwroot/example.com` → `rm -f index.html` → `git clone https://github.com/moneyfly1/myweb.git .`
- **BT File Manager**: Navigate to directory, open terminal and run `git clone https://github.com/moneyfly1/myweb.git .`
- **Local Upload**: Use SCP to upload project files to directory

Verify directory contains `install.sh`, `go.mod`, `frontend`, etc.

**3. Run Installation Script**

```bash
cd /www/wwwroot/example.com   # Replace with your site directory
chmod +x install.sh
sudo ./install.sh
```

**4. Follow Prompts**

- **Project Directory**: Script auto-detects current directory, press Enter to confirm
- **Domain**: Enter your domain (e.g., `example.com`)
- **Admin Username**: Enter admin username (default: `admin`)
- **Admin Email**: Enter admin email (e.g., `admin@example.com`)
- **Admin Password**: Set admin password (recommend strong password)

**5. Select Installation Option**

The script will display a menu. **For first-time installation, select `1`**. The script will automatically:
- ✅ Install Go language environment (if not installed)
- ✅ Install Node.js environment (if not installed)
- ✅ Compile backend service
- ✅ Build frontend
- ✅ Configure Nginx reverse proxy
- ✅ Apply for SSL certificate (Let's Encrypt)
- ✅ Create systemd service
- ✅ Start service

**6. Verify Installation**

- Frontend: `https://yourdomain.com`
- Admin Panel: `https://yourdomain.com/admin/login`
- Health Check: `https://yourdomain.com/health`

#### Post-Installation Management (BT Panel)

- **Project Directory**: Your BT site directory (e.g., `/www/wwwroot/example.com`)
- **Service**: `systemctl start/stop/restart/status cboard`
- **Logs**: `journalctl -u cboard -f` or `tail -f /www/wwwroot/example.com/server.log`
- **Configuration**: Edit `.env` in site directory, then restart service
- **Nginx Config**: BT Panel → Website → Settings → Configuration File

---

## ⚡ Redis Cache Configuration (Optional but Recommended)

### Why Redis Cache?

Redis caching can boost performance by **50-100x** for frequently accessed data:

- **Subscription Config Generation**: 200-500ms → 10-50ms (cache hit)
- **Package List**: Database query → Instant cache response
- **System Config**: Reduces database load significantly
- **User Info**: Faster profile loading

### Cache Strategy

| Data Type | Cache Key | TTL | Performance Gain |
|-----------|-----------|-----|------------------|
| Subscription Config | `subscription:config:{token}:{format}` | 1-10 min | ⭐⭐⭐⭐⭐ Very High |
| Package List | `packages:list:active` | 30 min | ⭐⭐⭐ High |
| Announcements | `announcements:list:active` | 10 min | ⭐⭐⭐ High |
| System Config | `system:config:{category}` | 1 hour | ⭐⭐⭐ High |
| Payment Methods | `payment:methods:active` | 1 hour | ⭐⭐⭐ High |
| Knowledge Base | `knowledge:*` | 1 hour | ⭐⭐⭐ High |
| Statistics | `statistics:{key}` | 30s-5min | ⭐⭐ Medium |

### Cache Invalidation

The system automatically clears cache when data changes:

- **Subscription Expiry**: Clears config cache when subscription expires
- **Admin Updates**: Clears cache when admin modifies subscriptions/users
- **User Purchase/Renewal**: Clears user subscription cache after payment
- **Device Management**: Clears cache when devices are cleared
- **Node Changes**: Clears all subscription configs when nodes are updated

### Enable Redis Cache

**During Installation:**

The installation script will prompt:

```
是否启用 Redis 缓存？(y/n，默认: y):
```

Press Enter or type `y` to enable. The script will:
- Auto-install Redis if not present
- Configure Redis connection in `.env`
- Start Redis service

**Manual Configuration:**

Edit `.env` file:

```env
# Redis Configuration (Optional but Recommended)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

Then restart service:

```bash
systemctl restart cboard
```

### Verify Redis is Working

```bash
# Check Redis service
systemctl status redis

# Test Redis connection
redis-cli ping
# Should return: PONG

# View cache keys
redis-cli keys "subscription:config:*"
redis-cli keys "packages:*"
```

---

## 👤 Administrator Account Management

### Create Administrator Account

Administrator account can be created during installation or separately afterwards.

#### Method 1: Using Installation Script (Recommended)

```bash
# Navigate to project directory
cd /www/wwwroot/example.com  # Or /opt/cboard for VPS installation

# Run installation script
sudo ./install.sh

# Select option 2: Create/Reset Admin Account
# Then follow prompts to enter:
# - Admin username (default: admin)
# - Admin email
# - Admin password
```

#### Method 2: Using Go Script (via Environment Variables)

```bash
# Navigate to project directory
cd /www/wwwroot/example.com  # Or /opt/cboard

# Set environment variables and run script
export ADMIN_USERNAME="admin"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="YourStrongPassword123!"

# Run creation script
go run scripts/admin_tool.go
```

**Notes**:
- If environment variables are not set, script will use default values (username: `admin`, email: `admin@example.com`, password: `admin123`)
- If admin account already exists, script will update the account information
- Production environment should set strong password via environment variables

### Update Administrator Password

If you forget the admin password, you can reset it using:

```bash
# Navigate to project directory
cd /www/wwwroot/example.com  # Or /opt/cboard

# Run password update script (replace with your actual password)
go run scripts/admin_tool.go YourNewPassword123!
```

### Unlock User Account

If account is locked due to multiple failed login attempts, unlock using:

```bash
# Navigate to project directory
cd /www/wwwroot/example.com  # Or /opt/cboard

# Unlock admin account (using username)
go run scripts/unlock_user.go admin

# Or unlock using email
go run scripts/unlock_user.go admin@example.com

# Unlock regular user account
go run scripts/unlock_user.go user@example.com
```

---

## 📊 Feature List

### ✅ Core Features

#### User Management
- [x] User registration and login
- [x] JWT authentication
- [x] Password reset via email
- [x] Email verification
- [x] User profile management
- [x] Login history tracking
- [x] User activity logging
- [x] User level system with discounts
- [x] Account security (2FA ready)

#### Subscription Management
- [x] Subscription creation and renewal
- [x] Device limit management
- [x] Expiration time control
- [x] Subscription reset
- [x] Multiple subscription types
- [x] Subscription URL generation (Clash/V2Ray format)
- [x] Clash config template (proxy groups and rules)
- [x] Device management (add, remove, view)
- [x] Online device tracking
- [x] Device fingerprinting and UA detection
- [x] **Redis cache for subscription configs (10-50ms response time)**

#### Order Management
- [x] Order creation and processing
- [x] Package orders
- [x] Device upgrade orders
- [x] Order cancellation
- [x] Order status tracking
- [x] Order history
- [x] Order export (CSV/Excel)
- [x] Bulk operations

#### Payment Integration
- [x] Alipay integration
- [x] WeChat Pay integration
- [x] Yipay integration (supports Alipay, WeChat Pay, QQ Pay)
- [x] Apple Pay integration
- [x] Balance payment
- [x] Mixed payment (balance + third-party)
- [x] Payment callback handling
- [x] Payment transaction tracking
- [x] Recharge management

#### Package Management
- [x] Package CRUD operations
- [x] Package pricing
- [x] Package activation/deactivation
- [x] Package features configuration
- [x] Package display order
- [x] **Redis cache for package list (instant loading)**

#### Node Management
- [x] Node collection (auto-collect from subscription URLs)
- [x] Manual node import (link import, manual entry, Clash config import)
- [x] Node CRUD operations
- [x] Node health monitoring and auto-check
- [x] Node status tracking (online/offline/timeout)
- [x] Node speed test (single/batch)
- [x] Node deduplication (based on Type:Server:Port)
- [x] Node grouping (by region)
- [x] Node subscription integration
- [x] **Auto cache invalidation when nodes change**

#### Notification System
- [x] Email notifications (SMTP configuration)
- [x] Telegram Bot notifications (admin notifications)
- [x] Bark iOS push notifications (admin notifications)
- [x] Customer email notifications (subscription expiry reminders, new user registration, new orders, etc.)
- [x] Admin notifications (order payment, user registration, subscription reset, subscription expiry, etc.)

#### Statistics & Monitoring
- [x] Dashboard statistics
- [x] User statistics
- [x] Order statistics
- [x] Revenue statistics
- [x] Subscription statistics
- [x] System logs
- [x] Audit logs
- [x] Real-time monitoring
- [x] **Redis cache for statistics (30s-5min TTL)**

---

## ⚙️ Configuration

### Environment Variables

Main configuration file: `.env`

```env
# Server Configuration
HOST=127.0.0.1          # Listen on localhost only, via Nginx reverse proxy
PORT=8000               # Backend service port

# Database Configuration (SQLite)
DATABASE_URL=sqlite:///./cboard.db

# Redis Configuration (Optional but Recommended for Production)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT Configuration (MUST CHANGE IN PRODUCTION!)
SECRET_KEY=your-secret-key-here-change-in-production-min-32-chars

# CORS Configuration (replace with your domain)
BACKEND_CORS_ORIGINS=https://yourdomain.com,http://yourdomain.com

# Email Configuration (Optional, use your SMTP provider)
SMTP_HOST=
SMTP_PORT=587
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM_EMAIL=

# Debug Mode
DEBUG=false
```

---

## 🛠️ Management Script Usage

### Common Operations

#### Create/Reset Admin Account
```bash
sudo ./install.sh
# Select option 2
```

#### Restart Service
```bash
sudo ./install.sh
# Select option 8 (standard restart) or 3 (force restart)
```

#### View Service Status
```bash
sudo ./install.sh
# Select option 6
```

#### View Real-time Logs
```bash
sudo ./install.sh
# Select option 7
```

### Manual Management Commands

If you prefer not to use the management script, you can use systemd commands directly:

```bash
# Start service
systemctl start cboard

# Stop service
systemctl stop cboard

# Restart service
systemctl restart cboard

# View status
systemctl status cboard

# View logs
journalctl -u cboard -f

# Enable auto-start on boot
systemctl enable cboard
```

---

## 🔒 Security Recommendations

1. **Strong Passwords in Production**
   - `SECRET_KEY` must be at least 32 characters random string
   - Use strong passwords for admin accounts

2. **Use HTTPS**
   - Installation script automatically configures SSL certificate
   - Ensure HTTPS enforcement is enabled

3. **Configure CORS**
   - Production environment must explicitly specify allowed domains
   - Do not use wildcard `*`

4. **Database Security**
   - Regular database backups
   - Ensure correct file permissions when using SQLite

5. **Redis Security**
   - Set Redis password in production
   - Bind Redis to localhost only
   - Use firewall to restrict Redis port access

6. **System Security**
   - Regularly update system and dependencies
   - Configure firewall rules
   - Use strong password policies

---

## 📝 Database Backup

### Automatic Backup (Recommended)

Configure scheduled task in BT Panel or cron:

```bash
#!/bin/bash
cd /www/wwwroot/cboard  # Or /opt/cboard
BACKUP_DIR="/www/backup/cboard"
mkdir -p $BACKUP_DIR
cp cboard.db $BACKUP_DIR/cboard_$(date +%Y%m%d_%H%M%S).db
# Keep backups from last 7 days
find $BACKUP_DIR -name "cboard_*.db" -mtime +7 -delete
```

### Manual Backup

```bash
cd /www/wwwroot/cboard  # Or /opt/cboard
cp cboard.db cboard.db.backup.$(date +%Y%m%d_%H%M%S)
```

---

## 🔧 Troubleshooting

### 1. Service Cannot Start

**Check logs**:
```bash
# View service logs
journalctl -u cboard -f

# View application logs
tail -f /www/wwwroot/cboard/server.log  # Or /opt/cboard/server.log
```

**Common causes**:
- Port occupied: Check if port 8000 is used by another program
- Permission issues: Ensure project directory permissions are correct
- Configuration errors: Check `.env` file configuration

### 2. 502 Bad Gateway

- Check if backend service is running: `systemctl status cboard`
- Check if port is correct: `netstat -tlnp | grep 8000`
- Check `proxy_pass` address in Nginx configuration

### 3. Redis Connection Failed

```bash
# Check Redis service
systemctl status redis

# Start Redis
systemctl start redis

# Test connection
redis-cli ping
```

### 4. SSL Certificate Application Failed

- Ensure domain is correctly resolved to server IP
- Ensure port 80 is open
- Check firewall settings

---

## 📚 Documentation

### Quick Start & Migration
| Document | Description |
|----------|-------------|
| [Quick Start Guide](./docs/migration/QUICK_START.md) | Quick start guide for new features |
| [Migration Guide](./docs/migration/MIGRATION_GUIDE.md) | Migrate new features to existing database |
| [Production Ready Report](./docs/reports/PRODUCTION_READY_REPORT.md) | Latest production deployment report |

### Deployment & Troubleshooting

| Document | Description |
|----------|-------------|
| [Docs Index](./docs/README.md) | Full doc index (features, config, deploy) |
| [VPS Deployment (No BT Panel)](./docs/VPS部署教程-无宝塔.md) | One-click VPS deployment |
| [Installation Troubleshooting](./docs/故障排查/安装问题排查指南.md) | Common installation issues and solutions |

### Configuration Documentation

#### Payment Configuration
- [Alipay configuration](./docs/配置/支付宝配置说明.md) - Alipay backend and panel config
- [Yipay guide](./docs/配置/易支付配置指南.md) - Yipay setup and usage

#### Notification Configuration
- [SMTP / Email](./docs/配置/邮件服务器配置说明.md) - Configure mail server
- [Telegram notifications](./docs/配置/Telegram通知配置说明.md) - Telegram Bot notification setup
- [Bark notifications](./docs/配置/Bark通知配置说明.md) - Bark iOS push notification setup
- [Customer notifications](./docs/配置/客户通知设置说明.md) - Customer email notification settings

#### System Settings
- [Basic settings](./docs/配置/基本设置说明.md) - Site info, logo, domain, GeoIP
- [Registration settings](./docs/配置/注册设置说明.md) - Registration flow, password requirements
- [Security & limits](./docs/配置/安全设置与限制说明.md) - Login fail limit, lock, IP whitelist, unlock
- [Theme settings](./docs/配置/主题设置说明.md) - Default theme, user customization
- [Announcement management](./docs/配置/公告管理说明.md) - Login announcement popup

#### Node & Backup
- [Node collection URLs](./docs/配置/采集地址配置说明.md) - Configure node collection addresses
- [Node health check](./docs/配置/节点健康检查设置说明.md) - Auto check interval, latency threshold
- [Backup settings](./docs/配置/备份设置说明.md) - Auto backup, Gitee/GitHub backup
- [GitHub configuration](./docs/配置/GitHub配置说明.md) - Detailed GitHub backup setup
- [Gitee configuration](./docs/配置/Gitee配置说明.md) - Detailed Gitee backup setup

---

## 📞 Support

If you encounter issues:

1. **Logs** (project dir: BT usually `/www/wwwroot/your-domain`, non-BT `/opt/cboard`):
   - App log: `project_dir/server.log` or `project_dir/uploads/logs/app.log`
   - Service log: `journalctl -u cboard -f`

2. **System check**:
   - Resources: `htop` or `free -h`
   - Health: `curl http://127.0.0.1:8000/health`
   - Service: `systemctl status cboard`
   - Redis: `redis-cli ping`

3. **Docs**: [Installation Troubleshooting](./docs/故障排查/安装问题排查指南.md)

---

## 📄 License

This project is licensed under the MIT License.

---

**Last Updated**: 2026-03-05
**Version**: v1.2.0
**Status**: ✅ Production Ready with Redis Cache Optimization
