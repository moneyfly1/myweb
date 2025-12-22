# CBoard - Modern Subscription Management System

[中文](README_zh.md) | English

---

## 📖 Overview

**CBoard** is a modern, high-performance subscription management system designed for VPN/proxy service providers. Built with Go language, it offers **70-90% memory reduction** compared to Python-based solutions while maintaining full feature parity.

### 🎯 Key Features

- 🚀 **High Performance**: Memory usage only 35-95 MB (vs 300-850 MB in Python version)
- ⚡ **Fast Startup**: Millisecond-level startup time
- 🔒 **Secure**: JWT authentication, password encryption, SQL injection protection
- 📦 **Feature Complete**: All core business functions included
- 🎨 **Modern Frontend**: Vue 3 + Element Plus, responsive design
- 🐳 **Easy Deployment**: One-click installation via BT Panel, single executable file
- 💳 **Multi-Payment**: Supports Alipay, WeChat Pay, PayPal, Apple Pay
- 👥 **User Management**: Complete user system with levels, invites, and rewards
- 📊 **Analytics**: Comprehensive statistics and monitoring
- 🎫 **Ticket System**: Built-in customer support system

---

## 🏗️ Technology Stack

### Backend
- **Web Framework**: [Gin](https://github.com/gin-gonic/gin) - High-performance HTTP web framework
- **ORM**: [GORM](https://gorm.io/) - The fantastic ORM library for Go
- **Database**: SQLite (default) / MySQL 5.7+ / PostgreSQL 12+
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
- **Nginx**: (included with BT Panel)
- **Database**: SQLite (default, no installation needed) or MySQL/PostgreSQL

---

## 🚀 Installation

### Quick Start - Clone from GitHub

The easiest way to get started is to clone the project directly from GitHub:

```bash
# Clone the repository
git clone https://github.com/moneyfly1/myweb.git cboard
cd cboard

# Make installation script executable
chmod +x install.sh

# Run installation script (requires root)
sudo ./install.sh
```

**GitHub Repository**: https://github.com/moneyfly1/myweb

---

## 🚀 Installation via BT Panel

### Prerequisites

- ✅ BT Panel installed (version 7.0+ recommended)
- ✅ Server OS: Ubuntu 18.04+ / Debian 10+ / CentOS 7+
- ✅ Server specs: At least 1 core CPU + 512 MB RAM + 10 GB disk
- ✅ Domain name bound (for SSL certificate)

### Detailed Installation Steps

#### Step 1: Create Website in BT Panel

1. **Login to BT Panel**
   - Access `http://your-server-ip:8888` (or your BT Panel address)
   - Login with your BT Panel credentials

2. **Create Website**
   - Click **Website** → **Add Site** in the left menu
   - Fill in the following information:
     - **Domain**: Enter your domain (e.g., `example.com`)
     - **Remark**: Optional project name (e.g., CBoard)
     - **Root Directory**: Auto-generated, typically `/www/wwwroot/example.com`
     - **FTP**: Don't create (optional)
     - **Database**: Don't create (optional, system uses SQLite)
     - **PHP Version**: Pure Static (or any version, doesn't matter)
   - Click **Submit** to complete website creation

3. **Record Website Directory Path**
   - After creation, record the website root directory path (e.g., `/www/wwwroot/example.com`)
   - This directory will be used for code deployment in the next steps

#### Step 2: Download Code to Website Directory

**Method 1: Clone via SSH (Recommended)**

```bash
# 1. Connect to your server via SSH
ssh root@your-server-ip

# 2. Navigate to the website directory you just created (replace with your actual path)
cd /www/wwwroot/example.com

# 3. Remove default index.html (if exists)
rm -f index.html

# 4. Clone project code from GitHub
git clone https://github.com/moneyfly1/myweb.git .

# 5. Verify files are downloaded correctly
ls -la
# You should see files and directories like install.sh, go.mod, frontend, etc.
```

**Method 2: Via BT Panel File Manager**

1. Login to BT Panel
2. Go to **File** → Navigate to `/www/wwwroot/example.com`
3. Delete the default `index.html` file (if exists)
4. Click **Terminal** button to open terminal
5. Execute in terminal:
   ```bash
   git clone https://github.com/moneyfly1/myweb.git .
   ```
6. Verify files are downloaded correctly

**Method 3: Upload via SCP (from local machine)**

```bash
# Execute on your local machine (replace with your actual path)
scp -r /path/to/goweb/* root@your-server:/www/wwwroot/example.com/
```

#### Step 3: Run Installation Script

After downloading the code, run the installation script:

```bash
# 1. Make sure you're in the website directory (replace with your actual path)
cd /www/wwwroot/example.com

# 2. Add execute permission to installation script
chmod +x install.sh

# 3. Run installation script (requires root privileges)
sudo ./install.sh
```

#### Step 4: Configure Installation Parameters

The installation script will prompt you for:

- **Project Directory**: Default detects current directory, press Enter to confirm
- **Domain Name**: Enter your domain (e.g., `example.com`)
- **Admin Username**: Enter admin username (default: `admin`)
- **Admin Email**: Enter admin email (e.g., `admin@example.com`)
- **Admin Password**: Set admin password (recommend using strong password)

#### Step 5: Select Installation Option

The installation script will display the following menu:

```
==========================================
       CBoard Go Management Panel
==========================================
  1. One-Click Full Auto Deployment (SSL + Reverse Proxy)
  2. Create/Reset Admin Account
  3. Force Restart Service (Kill process then restart)
  4. Deep Clean System Cache
  5. Unlock User Account
------------------------------------------
  6. View Service Status
  7. View Real-time Service Logs
  8. Standard Restart Service (Systemd)
  9. Stop Service
  0. Exit Script
==========================================
```

**For first-time installation, select `1`**. The script will automatically:
- ✅ Install Go language environment (if not installed)
- ✅ Install Node.js environment (if not installed)
- ✅ Compile backend service
- ✅ Build frontend
- ✅ Configure Nginx reverse proxy
- ✅ Apply for SSL certificate (Let's Encrypt)
- ✅ Create systemd service
- ✅ Start service

#### Step 6: Verify Installation

After installation, access your domain:

- **Frontend Interface**: `https://yourdomain.com`
- **Admin Login**: `https://yourdomain.com/admin/login`
- **Health Check**: `https://yourdomain.com/health`
- **API Endpoints**: `https://yourdomain.com/api/v1/...`

### Post-Installation Configuration

#### Configure Nginx (if needed)

The installation script automatically configures Nginx, but you can manually check:

1. Login to BT Panel
2. Go to **Website** → Find your website → Click **Settings**
3. Go to **Configuration File** tab
4. Verify reverse proxy configuration is correct (script has auto-configured)

#### Configure Firewall

Ensure the following ports are open:
- **80**: HTTP
- **443**: HTTPS
- **Backend Port**: Default 8080 (internal access only, no need to open externally)

In BT Panel:
1. Go to **Security** → **Firewall**
2. Ensure ports 80 and 443 are open

---

## 👤 Administrator Account Management

### Create Administrator Account

Administrator account can be created during installation or separately afterwards.

#### Method 1: Using Installation Script (Recommended)

```bash
# Navigate to project directory (replace with your actual path)
cd /www/wwwroot/example.com

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
cd /www/wwwroot/example.com

# Set environment variables and run script
export ADMIN_USERNAME="admin"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="YourStrongPassword123!"

# Run creation script
go run scripts/create_admin.go
```

**Notes**:
- If environment variables are not set, script will use default values (username: `admin`, email: `admin@example.com`, password: `admin123`)
- If admin account already exists, script will update the account information
- Production environment should set strong password via environment variables

#### Method 3: Using Go Script (Interactive)

```bash
# Navigate to project directory
cd /www/wwwroot/example.com

# Run script directly (will use defaults or prompt for input)
go run scripts/create_admin.go
```

### Update Administrator Password

If you forget the admin password, you can reset it using:

```bash
# Navigate to project directory
cd /www/wwwroot/example.com

# Run password update script (replace with your actual password)
go run scripts/update_admin_password.go YourNewPassword123!

# Example
go run scripts/update_admin_password.go Sikeming001@
```

**Notes**:
- Password must be at least 6 characters
- Script automatically finds admin account (username or email is `admin` or `admin@example.com`)
- If admin account is not found, please create account first

### Unlock User Account

If account is locked due to multiple failed login attempts, unlock using:

```bash
# Navigate to project directory
cd /www/wwwroot/example.com

# Unlock admin account (using username)
go run scripts/unlock_user.go admin

# Or unlock using email
go run scripts/unlock_user.go admin@example.com

# Unlock regular user account
go run scripts/unlock_user.go user@example.com
```

**Notes**:
- Script supports unlocking using username or email
- Can unlock both admin and regular user accounts
- Unlock operation will:
  - Clear all failed login records
  - Set account to active status (`IsActive=true`)
  - Set account to verified status (`IsVerified=true`)

**Important Notes**:
- If still unable to login, IP address may be locked by rate limiter
- Rate limiter is based on IP address, lock duration is 15 minutes
- Solutions:
  - Wait 15 minutes and retry
  - Change IP address (use VPN or mobile network)
  - Restart server to clear rate limiter records in memory

### Administrator Login

1. **Access Admin Login Page**
   - URL: `https://yourdomain.com/admin/login`
   - Or: `https://yourdomain.com/#/admin/login`

2. **Enter Login Credentials**
   - **Username**: Your created admin username (default: `admin`)
   - **Password**: Your set admin password
   - Supports login with username or email

3. **After Login**
   - Enter admin backend
   - Access all management functions

### Administrator Permissions

Administrators have full access to:

- **User Management**: Create, edit, delete, view users, bulk operations
- **Subscription Management**: Create, edit, delete subscriptions, bulk operations, expiration reminders
- **Order Management**: View, process orders, order export
- **Package Management**: Create, edit, delete packages, pricing management
- **Node Management**: Add, edit, delete nodes, bulk import, node testing
- **Payment Configuration**: Configure Alipay, WeChat Pay, PayPal, etc.
- **System Configuration**: System settings, notification settings, email configuration
- **Statistics and Monitoring**: Data statistics, region analysis, user analysis
- **Ticket Management**: Handle user tickets, reply to tickets
- **Device Management**: View user devices, manage device limits
- **Invite Code Management**: Generate, manage invite codes
- **Log Management**: View system logs, login history, operation logs

### Frequently Asked Questions

**Q: What if I forget the admin password?**
A: Use `go run scripts/update_admin_password.go <new-password>` to reset password.

**Q: What if admin account is locked?**
A: Use `go run scripts/unlock_user.go admin` to unlock account.

**Q: How to create multiple admin accounts?**
A: Currently system only supports one admin account. If multiple admins are needed, create regular users and assign permissions (requires code modification).

**Q: What if admin account was not created during installation?**
A: Run `go run scripts/create_admin.go` to create admin account.

**Q: How to verify admin account was created successfully?**
A: Try logging into admin backend, or check `users` table in database for records with `is_admin` field set to `true`.

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
- [x] Device management (add, remove, view)
- [x] Online device tracking
- [x] Device fingerprinting and UA detection

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
- [x] PayPal integration
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

#### Coupon System
- [x] Coupon creation and management
- [x] Discount coupons (percentage)
- [x] Fixed amount coupons
- [x] Coupon code validation
- [x] Coupon usage tracking
- [x] Coupon expiration management

#### Invite System
- [x] Invite code generation
- [x] Invite relationship tracking
- [x] Inviter rewards
- [x] Invitee rewards
- [x] Minimum order amount requirement
- [x] New user only rewards
- [x] Reward distribution automation

#### Node Management
- [x] Node CRUD operations
- [x] Node health monitoring
- [x] Node status tracking
- [x] Custom node support
- [x] Node grouping
- [x] Node subscription integration

#### Custom Node System
- [x] Server management (SSH connection)
- [x] Automatic node deployment (via XrayR API)
- [x] Cloudflare DNS and certificate automation
- [x] Traffic control
- [x] Expiration time management
- [x] User-specific node allocation

#### Device Management
- [x] Device recognition and fingerprinting
- [x] Device limit enforcement
- [x] Device deletion
- [x] Device information tracking (UA, IP, etc.)
- [x] Active device monitoring
- [x] Batch device operations

#### Notification System
- [x] Email notifications
- [x] In-app notifications
- [x] Notification templates
- [x] Notification preferences
- [x] Notification history

#### Ticket System
- [x] Ticket creation
- [x] Ticket replies
- [x] Ticket status management
- [x] Ticket attachments
- [x] Ticket assignment
- [x] Ticket priority levels

#### Statistics & Monitoring
- [x] Dashboard statistics
- [x] User statistics
- [x] Order statistics
- [x] Revenue statistics
- [x] Subscription statistics
- [x] System logs
- [x] Audit logs
- [x] Real-time monitoring

#### System Configuration
- [x] System settings management
- [x] Payment configuration
- [x] Email configuration
- [x] SMS configuration
- [x] Security settings
- [x] Feature toggles
- [x] Announcement management

#### Backup & Restore
- [x] Database backup
- [x] Configuration backup
- [x] Automated backup scheduling
- [x] Backup file management

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

# JWT Configuration (MUST CHANGE IN PRODUCTION!)
SECRET_KEY=your-secret-key-here-change-in-production-min-32-chars

# CORS Configuration (replace with your domain)
BACKEND_CORS_ORIGINS=https://yourdomain.com,http://yourdomain.com

# Email Configuration (Optional)
SMTP_HOST=smtp.qq.com
SMTP_PORT=587
SMTP_USERNAME=your-email@qq.com
SMTP_PASSWORD=your-smtp-password
SMTP_FROM_EMAIL=your-email@qq.com

# Debug Mode
DEBUG=false
```

### Nginx Configuration

The installation script automatically configures Nginx. To manually adjust:

1. Login to BT Panel
2. **Website** → Find your website → **Settings** → **Configuration File**
3. Modify configuration → **Save** → **Reload Configuration**

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

#### Stop Service
```bash
sudo ./install.sh
# Select option 9
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

5. **System Security**
   - Regularly update system and dependencies
   - Configure firewall rules
   - Use strong password policies

---

## 📝 Database Backup

### Automatic Backup (Recommended)

Configure scheduled task in BT Panel:

1. **Scheduled Tasks** → **Add Scheduled Task**
2. **Task Type**: Shell Script
3. **Task Name**: CBoard Database Backup
4. **Execution Cycle**: Daily at 00:02
5. **Script Content**:
```bash
#!/bin/bash
cd /www/wwwroot/cboard
BACKUP_DIR="/www/backup/cboard"
mkdir -p $BACKUP_DIR
cp cboard.db $BACKUP_DIR/cboard_$(date +%Y%m%d_%H%M%S).db
# Keep backups from last 7 days
find $BACKUP_DIR -name "cboard_*.db" -mtime +7 -delete
```

### Manual Backup

```bash
cd /www/wwwroot/cboard
cp cboard.db cboard.db.backup.$(date +%Y%m%d_%H%M%S)
```

### Backup via API

The system also provides backup API endpoint (admin only):
- `POST /api/v1/admin/backup/create` - Create backup

---

## 🔧 Troubleshooting

### 1. Service Cannot Start

**Check logs**:
```bash
# View service logs
journalctl -u cboard -f

# View application logs
tail -f /www/wwwroot/cboard/uploads/logs/app.log
```

**Common causes**:
- Port occupied: Check if port 8000 is used by another program
- Permission issues: Ensure project directory permissions are correct
- Configuration errors: Check `.env` file configuration

### 2. 502 Bad Gateway

- Check if backend service is running: `systemctl status cboard`
- Check if port is correct: `netstat -tlnp | grep 8000`
- Check `proxy_pass` address in Nginx configuration

### 3. SSL Certificate Application Failed

- Ensure domain is correctly resolved to server IP
- Ensure port 80 is open
- Check firewall settings

### 4. Database Permission Error

```bash
cd /www/wwwroot/cboard
chmod 666 cboard.db
chown www:www cboard.db
```

### 5. Frontend Cannot Access Backend API

- Check if `BACKEND_CORS_ORIGINS` in `.env` includes your domain
- Check if `/api/` proxy in Nginx configuration is correct

### 6. Admin Login Issues

- Reset admin password using installation script (option 2)
- Check admin account status: `go run scripts/check_admin.go`
- Unlock admin account: `go run scripts/unlock_admin.go`

---

## 📖 API Documentation

After starting the server, main API endpoints:

### Authentication
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Refresh token
- `POST /api/v1/auth/logout` - User logout

### User
- `GET /api/v1/users/me` - Get current user
- `PUT /api/v1/users/me` - Update user profile
- `GET /api/v1/users/login-history` - Get login history

### Subscription
- `GET /api/v1/subscriptions` - Get subscription list
- `GET /api/v1/subscriptions/:id` - Get subscription details
- `GET /subscribe/:url` - Get subscription configuration (Clash/V2Ray)

### Orders
- `GET /api/v1/orders` - Get order list
- `POST /api/v1/orders` - Create order
- `GET /api/v1/orders/:id` - Get order details
- `POST /api/v1/orders/:id/cancel` - Cancel order

### Packages
- `GET /api/v1/packages` - Get package list
- `GET /api/v1/packages/:id` - Get package details

### Payment
- `POST /api/v1/payment/notify/:method` - Payment callback
- `GET /api/v1/payment/status/:orderNo` - Get payment status

### Admin APIs
All admin APIs require admin authentication and are prefixed with `/api/v1/admin/`

For complete API list, see: `internal/api/router/router.go`

---

## 🏗️ Project Structure

```
goweb/
├── cmd/server/main.go          # Main entry point
├── internal/
│   ├── api/                    # API layer
│   │   ├── handlers/           # Request handlers
│   │   └── router/             # Route definitions
│   ├── core/                   # Core modules
│   │   ├── auth/               # Authentication
│   │   ├── config/             # Configuration
│   │   └── database/           # Database
│   ├── models/                 # Data models
│   ├── services/               # Business services
│   ├── middleware/             # Middleware
│   └── utils/                  # Utility functions
├── frontend/                   # Vue 3 frontend
│   ├── src/                    # Frontend source code
│   │   ├── views/              # Page components
│   │   ├── components/         # Reusable components
│   │   ├── router/             # Frontend routes
│   │   └── store/              # State management
│   └── dist/                   # Built files
├── scripts/                    # Utility scripts
│   ├── create_admin.go         # Create admin account
│   ├── check_admin.go          # Check admin account
│   └── unlock_admin.go        # Unlock admin account
├── .env                        # Environment variables
├── install.sh                  # BT Panel installation script
├── cboard.db                   # SQLite database
├── README.md                   # This file (English)
└── README_zh.md                # Chinese version
```

---

## ⚠️ Important Notes

1. **First-Time Setup**
   - After installation, immediately change the default admin password
   - Update `SECRET_KEY` in `.env` file
   - Configure email settings for password reset and notifications

2. **Database**
   - SQLite is used by default (no installation needed)
   - For production with high traffic, consider MySQL or PostgreSQL
   - Regular backups are essential

3. **Security**
   - Never commit `.env` file to version control
   - Use strong passwords for all accounts
   - Enable HTTPS in production
   - Regularly update dependencies

4. **Performance**
   - For high-traffic scenarios, consider using MySQL/PostgreSQL
   - Enable Nginx caching for static files
   - Monitor server resources regularly

5. **Updates**
   - Always backup database before updating
   - Test updates in staging environment first
   - Review changelog before updating

---

## 📚 Documentation

The system provides comprehensive documentation for all features:

### 📋 List Function Documentation

- **[List Functions Index](./docs/list_functions_index.md)** - Index and quick reference for all list functions

#### Admin List Functions

- **[User List Management](./docs/user_list_management.md)** - User information management, search, filtering, batch operations
- **[Subscription List Management](./docs/subscription_list_management.md)** - Subscription management, device limit principles
- **[Node List Management](./docs/node_management.md)** - Node collection, import, management, region identification
- **[Order List Management](./docs/order_management.md)** - Order processing, payment management, order statistics
- **[Ticket List Management](./docs/ticket_management.md)** - Ticket processing, replies, status management
- **[Abnormal Users Management](./docs/abnormal_users_management.md)** - Abnormal user identification and handling
- **[Statistics Analysis](./docs/statistics_analysis.md)** - Data statistics, region analysis, trend analysis

#### User List Functions

- **[Device Management](./docs/device_management.md)** - Device viewing, deletion, device limit principles
- **[Login History Management](./docs/login_history_management.md)** - Login records, region information, security monitoring

### 🔧 Technical Documentation

- **[Custom Node Implementation](./docs/custom_node_implementation.md)** - Complete implementation guide for custom node system
- **[GeoIP Integration Guide](./docs/GeoIP集成说明.md)** - GeoIP database integration and usage

### 📖 Core Function Principles

#### Device Limit Principles

Device limit is a core feature of subscription management. For detailed principles, see:

- **[Subscription List Management - Device Limit Principles](./docs/subscription_list_management.md#设备数量限制原理)**
- **[Device Management - Device Limit Principles](./docs/device_management.md#设备数量限制原理)**

**Key Concepts:**
- Device Identification: Generate device hash based on User-Agent and IP address
- Device Roaming: Automatic identification of the same device in different network environments
- Limit Mechanism: Different handling strategies for devices under limit/at limit/over limit
- Priority Strategy: "Most Recently Used First" strategy, automatically eliminates unused devices

---

## 📞 Support

If you encounter issues:

1. Check log files: `/www/wwwroot/cboard/uploads/logs/app.log`
2. Check service logs: `journalctl -u cboard -f`
3. Check system resources: `htop` or `free -h`
4. Check network connection: `curl http://127.0.0.1:8000/health`
5. Review this README and troubleshooting section
6. Refer to [Documentation](#-documentation) for detailed feature descriptions

---

## 📄 License

This project is licensed under the MIT License.

---

**Last Updated**: 2024-12-22  
**Version**: v1.0.0  
**Status**: ✅ Production Ready
