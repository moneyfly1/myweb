# CBoard - 现代化订阅管理系统

[English](README.md) | 中文

---

## 📖 系统简介

**CBoard** 是一个现代化的高性能订阅管理系统，专为 VPN/代理服务提供商设计。使用 Go 语言构建，相比 Python 版本可节省 **70-90% 的内存占用**，同时保持完整的功能特性。

### 🎯 核心特性

- 🚀 **高性能**: 内存占用仅 35-95 MB（Python 版本 300-850 MB）
- ⚡ **快速启动**: 毫秒级启动时间
- 🔒 **安全可靠**: JWT 认证、密码加密、SQL 注入防护
- 📦 **功能完整**: 包含所有核心业务功能
- 🎨 **现代化前端**: Vue 3 + Element Plus，响应式设计
- 🐳 **易于部署**: 支持宝塔面板一键安装，单一可执行文件
- 💳 **多支付方式**: 支持支付宝、微信支付、PayPal、Apple Pay
- 👥 **用户管理**: 完整的用户系统，包含等级、邀请、奖励
- 📊 **数据分析**: 全面的统计和监控功能
- 🎫 **工单系统**: 内置客户支持系统

---

## 🏗️ 技术栈

### 后端
- **Web 框架**: [Gin](https://github.com/gin-gonic/gin) - 高性能 HTTP Web 框架
- **ORM**: [GORM](https://gorm.io/) - Go 语言优秀的 ORM 库
- **数据库**: SQLite（默认）/ MySQL 5.7+ / PostgreSQL 12+
- **认证**: JWT（JSON Web Tokens）
- **配置管理**: Viper
- **编程语言**: Go 1.21+

### 前端
- **框架**: Vue 3（组合式 API）
- **UI 库**: Element Plus
- **构建工具**: Vite
- **状态管理**: Pinia
- **路由**: Vue Router 4

---

## 📋 系统要求

### 最低配置要求
- **CPU**: 1 核心（推荐 2 核心+）
- **内存**: 512 MB（推荐 1 GB+）
- **磁盘**: 10 GB（推荐 20 GB+）
- **操作系统**: Ubuntu 18.04+ / Debian 10+ / CentOS 7+

### 软件要求
- **Go**: 1.21+（安装脚本会自动安装）
- **Node.js**: 16+（用于前端构建）
- **Nginx**: （宝塔面板自带）
- **数据库**: SQLite（默认，无需安装）或 MySQL/PostgreSQL

---

## 🚀 安装指南

### 快速开始 - VPS 一键安装（推荐）

最简单的方式是直接在 VPS 上运行一键安装脚本：

```bash
# 1. 下载安装脚本
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install-vps.sh

# 2. 添加执行权限
chmod +x install-vps.sh

# 3. 运行安装脚本（需要 root 权限）
#    脚本会自动下载代码、安装所有依赖、配置环境
sudo bash install-vps.sh
```

**GitHub 仓库地址**: https://github.com/moneyfly1/myweb

**一键安装脚本**: `install-vps.sh` - 全自动安装，自动处理所有环境问题

---

## 🚀 VPS 一键安装（非宝塔面板）

### 前置条件

- ✅ 服务器系统：Ubuntu 18.04+ / Debian 10+ / CentOS 7+
- ✅ 服务器配置：至少 1 核心 CPU + 512 MB 内存 + 10 GB 磁盘
- ✅ 已绑定域名（用于 SSL 证书）
- ✅ 已配置域名 DNS 解析到服务器 IP
- ✅ 服务器已开放 80 和 443 端口

### 一键安装步骤

#### 步骤 1：下载并运行安装脚本

通过 SSH 连接到您的 VPS 服务器，然后执行：

```bash
# 下载安装脚本
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install-vps.sh

# 添加执行权限
chmod +x install-vps.sh

# 直接运行安装脚本（脚本会自动下载代码）
sudo bash install-vps.sh
```

**重要说明**：
- 脚本会自动从 GitHub 下载项目代码，无需手动克隆仓库
- 脚本会自动安装所有依赖（Go、Node.js、Nginx、Certbot 等）
- 脚本会自动配置网络代理和镜像源
- 脚本会自动处理防火墙、端口、域名解析等问题
- 只需按提示输入域名和管理员信息即可

#### 步骤 3：按提示输入信息

安装脚本会依次提示您输入以下信息：

1. **域名**：输入您的域名（例如：`example.com`）
   - 必须输入，格式需正确
   - 确保域名已正确解析到服务器 IP

2. **项目安装目录**：输入项目安装路径（默认：`/opt/cboard`）
   - 可直接按回车使用默认路径
   - 或输入自定义路径

3. **管理员用户名**：输入管理员用户名（默认：`admin`）
   - 可直接按回车使用默认值
   - 或输入自定义用户名

4. **管理员邮箱**：输入管理员邮箱（必填）
   - 必须输入有效的邮箱地址
   - 用于接收系统通知和 SSL 证书申请

5. **管理员密码**：输入管理员密码（必填）
   - 密码长度至少 6 位
   - 需要输入两次确认

#### 步骤 4：自动安装过程

确认信息后，脚本会自动完成以下操作：

- ✅ 检测操作系统类型
- ✅ 安装系统依赖（curl, wget, git, nginx, certbot 等）
- ✅ 自动安装 Go 语言环境（1.21.5）
- ✅ 自动安装 Node.js 环境（18.x）
- ✅ 从 GitHub 下载项目代码
- ✅ 创建环境配置文件（`.env`）
- ✅ 编译后端程序
- ✅ 构建前端项目
- ✅ 创建管理员账户
- ✅ 配置 Nginx 反向代理
- ✅ 申请 SSL 证书（Let's Encrypt）
- ✅ 创建 systemd 服务
- ✅ 启动服务

#### 步骤 5：验证安装

安装完成后，访问您的域名：

- **前端界面**: `https://yourdomain.com`
- **管理员登录**: `https://yourdomain.com/admin/login`
- **健康检查**: `https://yourdomain.com/health`
- **API 接口**: `https://yourdomain.com/api/v1/...`

### 安装后管理

#### 常用命令

```bash
# 查看服务状态
systemctl status cboard

# 查看服务日志
journalctl -u cboard -f

# 重启服务
systemctl restart cboard

# 停止服务
systemctl stop cboard

# 启动服务
systemctl start cboard
```

#### 查看应用日志

```bash
# 查看应用日志文件
tail -f /opt/cboard/server.log

# 或查看 systemd 日志
journalctl -u cboard -n 100
```

#### 修改配置

配置文件位置：`/opt/cboard/.env`

修改后需要重启服务：

```bash
systemctl restart cboard
```

### 故障排除

#### 1. SSL 证书申请失败

**可能原因**：
- 域名未正确解析到服务器 IP
- 80 端口未开放
- 防火墙阻止了 Let's Encrypt 验证

**解决方法**：
```bash
# 检查域名解析
nslookup yourdomain.com

# 检查端口是否开放
netstat -tlnp | grep :80

# 手动申请证书
certbot --nginx -d yourdomain.com
```

#### 2. 服务无法启动

**检查日志**：
```bash
journalctl -u cboard -n 50
```

**常见原因**：
- 端口被占用（默认 8000）
- 配置文件错误
- 数据库权限问题

#### 3. 前端无法访问后端 API

**检查 Nginx 配置**：
```bash
# 查看 Nginx 配置
cat /etc/nginx/sites-available/cboard
# 或 CentOS
cat /etc/nginx/conf.d/cboard.conf

# 测试 Nginx 配置
nginx -t

# 重载 Nginx
systemctl reload nginx
```

#### 4. 忘记管理员密码

```bash
cd /opt/cboard
export ADMIN_USERNAME="admin"
export ADMIN_EMAIL="your-email@example.com"
export ADMIN_PASSWORD="your-new-password"
go run scripts/create_admin.go
```

### 注意事项

1. **首次安装**：
   - 确保服务器有足够的磁盘空间（至少 2GB）
   - 确保网络连接正常，可以访问 GitHub
   - 安装过程可能需要 5-10 分钟

2. **安全建议**：
   - 安装后立即修改默认密码
   - 定期更新系统和依赖
   - 配置防火墙规则
   - 定期备份数据库

3. **性能优化**：
   - 对于高流量场景，建议使用 MySQL/PostgreSQL
   - 可以配置 Nginx 缓存静态文件
   - 监控服务器资源使用情况

---

## 🚀 宝塔面板一键安装

### 前置条件

- ✅ 已安装宝塔面板（建议版本 7.0+）
- ✅ 服务器系统：Ubuntu 18.04+ / Debian 10+ / CentOS 7+
- ✅ 服务器配置：至少 1 核心 CPU + 512 MB 内存 + 10 GB 磁盘
- ✅ 已绑定域名（用于 SSL 证书）

### 详细安装步骤

#### 步骤 1：在宝塔面板创建网站

1. **登录宝塔面板**
   - 访问 `http://your-server-ip:8888`（或您的宝塔面板地址）
   - 使用您的宝塔账号登录

2. **创建网站**
   - 点击左侧菜单 **网站** → **添加站点**
   - 填写以下信息：
     - **域名**：输入您的域名（如：`example.com`）
     - **备注**：可填写项目名称（如：CBoard）
     - **根目录**：系统会自动生成，通常为 `/www/wwwroot/example.com`
     - **FTP**：不创建（可选）
     - **数据库**：不创建（可选，系统使用 SQLite）
     - **PHP 版本**：纯静态（或任意版本，不影响）
   - 点击 **提交** 完成网站创建

3. **记录网站目录路径**
   - 创建完成后，记录下网站根目录路径（如：`/www/wwwroot/example.com`）
   - 后续步骤将在此目录中部署代码

#### 步骤 2：下载代码到网站目录

**方式一：通过 SSH 克隆（推荐）**

```bash
# 1. 通过 SSH 连接到服务器
ssh root@your-server-ip

# 2. 进入刚创建的网站目录（替换为您的实际路径）
cd /www/wwwroot/example.com

# 3. 删除默认的 index.html（如果存在）
rm -f index.html

# 4. 从 GitHub 克隆项目代码
git clone https://github.com/moneyfly1/myweb.git .

# 5. 验证文件是否正确下载
ls -la
# 应该能看到 install.sh、go.mod、frontend 等文件和目录
```

**方式二：通过宝塔面板文件管理器**

1. 登录宝塔面板
2. 进入 **文件** → 导航到 `/www/wwwroot/example.com`
3. 删除默认的 `index.html` 文件（如果存在）
4. 点击 **终端** 按钮，打开终端
5. 在终端中执行：
   ```bash
   git clone https://github.com/moneyfly1/myweb.git .
   ```
6. 验证文件是否正确下载

**方式三：通过 SCP 上传（从本地机器）**

```bash
# 在本地机器执行（替换为您的实际路径）
scp -r /path/to/goweb/* root@your-server:/www/wwwroot/example.com/
```

#### 步骤 3：运行安装脚本

代码下载完成后，运行安装脚本：

```bash
# 1. 确保在网站目录中（替换为您的实际路径）
cd /www/wwwroot/example.com

# 2. 添加安装脚本执行权限
chmod +x install.sh

# 3. 运行安装脚本（需要 root 权限）
sudo ./install.sh
```

#### 步骤 4：配置安装参数

安装脚本会提示您输入以下信息：

- **项目目录**：默认会检测当前目录，直接按回车确认即可
- **域名**：输入您的域名（如：`example.com`）
- **管理员用户名**：输入管理员用户名（默认：`admin`）
- **管理员邮箱**：输入管理员邮箱（如：`admin@example.com`）
- **管理员密码**：设置管理员密码（建议使用强密码）

#### 步骤 5：选择安装选项

安装脚本会显示以下菜单：

```
==========================================
       CBoard Go 终极管理面板
==========================================
  1. 一键全自动部署 (SSL + 反代)
  2. 创建/重置管理员账号
  3. 强制重启服务 (杀进程后重启)
  4. 深度清理系统缓存
  5. 解锁用户账户
------------------------------------------
  6. 查看服务运行状态
  7. 查看实时服务日志
  8. 标准重启服务 (Systemd)
  9. 停止服务
  0. 退出脚本
==========================================
```

**首次安装请选择 `1`**，脚本会自动完成：
- ✅ 安装 Go 语言环境（如未安装）
- ✅ 安装 Node.js 环境（如未安装）
- ✅ 编译后端服务
- ✅ 构建前端
- ✅ 配置 Nginx 反向代理
- ✅ 申请 SSL 证书（Let's Encrypt）
- ✅ 创建 systemd 服务
- ✅ 启动服务

#### 步骤 6：验证安装

安装完成后，访问您的域名：

- **前端界面**: `https://yourdomain.com`
- **管理员登录**: `https://yourdomain.com/admin/login`
- **健康检查**: `https://yourdomain.com/health`
- **API 接口**: `https://yourdomain.com/api/v1/...`

### 安装后配置

#### 配置 Nginx（如果需要）

安装脚本会自动配置 Nginx，但您也可以手动检查：

1. 登录宝塔面板
2. 进入 **网站** → 找到您的网站 → 点击 **设置**
3. 进入 **配置文件** 标签
4. 确认反向代理配置是否正确（脚本已自动配置）

#### 配置防火墙

确保以下端口已开放：
- **80**：HTTP
- **443**：HTTPS
- **后端端口**：默认 8080（仅内网访问，不需要对外开放）

在宝塔面板中：
1. 进入 **安全** → **防火墙**
2. 确保 80 和 443 端口已开放

---

## 👤 管理员账户管理

### 创建管理员账户

管理员账户可以在安装过程中创建，也可以后续单独创建。

#### 方法一：使用安装脚本（推荐）

```bash
# 进入项目目录（替换为您的实际路径）
cd /www/wwwroot/example.com

# 运行安装脚本
sudo ./install.sh

# 选择选项 2: 创建/重置管理员账号
# 然后按照提示输入：
# - 管理员用户名（默认：admin）
# - 管理员邮箱
# - 管理员密码
```

#### 方法二：使用 Go 脚本（通过环境变量）

```bash
# 进入项目目录
cd /www/wwwroot/example.com

# 设置环境变量并运行脚本
export ADMIN_USERNAME="admin"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="YourStrongPassword123!"

# 运行创建脚本
go run scripts/create_admin.go
```

**说明**：
- 如果未设置环境变量，脚本会使用默认值（用户名：`admin`，邮箱：`admin@example.com`，密码：`admin123`）
- 如果管理员账户已存在，脚本会更新该账户的信息
- 生产环境建议通过环境变量设置强密码

#### 方法三：使用 Go 脚本（交互式）

```bash
# 进入项目目录
cd /www/wwwroot/example.com

# 直接运行脚本（会使用默认值或提示输入）
go run scripts/create_admin.go
```

### 修改管理员密码

如果忘记管理员密码，可以通过以下方式重置：

```bash
# 进入项目目录
cd /www/wwwroot/example.com

# 运行密码修改脚本（替换为您的实际密码）
go run scripts/update_admin_password.go YourNewPassword123!

# 示例
go run scripts/update_admin_password.go Sikeming001@
```

**说明**：
- 密码长度至少 6 位
- 脚本会自动查找管理员账户（用户名或邮箱为 `admin` 或 `admin@example.com`）
- 如果找不到管理员账户，请先创建账户

### 解锁用户账户

如果账户因多次登录失败被锁定，可以通过以下方式解锁：

```bash
# 进入项目目录
cd /www/wwwroot/example.com

# 解锁管理员账户（使用用户名）
go run scripts/unlock_user.go admin

# 或使用邮箱解锁
go run scripts/unlock_user.go admin@example.com

# 解锁普通用户账户
go run scripts/unlock_user.go user@example.com
```

**说明**：
- 脚本支持使用用户名或邮箱解锁
- 可以解锁管理员账户和普通用户账户
- 解锁操作会：
  - 清除所有登录失败记录
  - 设置账户为激活状态（`IsActive=true`）
  - 设置账户为已验证状态（`IsVerified=true`）

**注意事项**：
- 如果仍然无法登录，可能是 IP 地址被速率限制器锁定
- 速率限制器基于 IP 地址，锁定时间为 15 分钟
- 解决方案：
  - 等待 15 分钟后重试
  - 更换 IP 地址（使用 VPN 或移动网络）
  - 重启服务器以清除内存中的速率限制记录

### 管理员登录

1. **访问管理员登录页面**
   - 地址：`https://yourdomain.com/admin/login`
   - 或：`https://yourdomain.com/#/admin/login`

2. **输入登录凭据**
   - **用户名**：您创建的管理员用户名（默认：`admin`）
   - **密码**：您设置的管理员密码
   - 支持使用用户名或邮箱登录

3. **登录后功能**
   - 进入管理员后台
   - 可以访问所有管理功能

### 管理员权限

管理员拥有以下完整权限：

- **用户管理**：创建、编辑、删除、查看用户，批量操作
- **订阅管理**：创建、编辑、删除订阅，批量操作，到期提醒
- **订单管理**：查看、处理订单，订单导出
- **套餐管理**：创建、编辑、删除套餐，定价管理
- **节点管理**：添加、编辑、删除节点，批量导入，节点测试
- **支付配置**：配置支付宝、微信支付、PayPal 等
- **系统配置**：系统设置、通知设置、邮件配置
- **统计和监控**：数据统计、地区分析、用户分析
- **工单管理**：处理用户工单，回复工单
- **设备管理**：查看用户设备，管理设备限制
- **邀请码管理**：生成、管理邀请码
- **日志管理**：查看系统日志、登录历史、操作日志

### 常见问题

**Q: 忘记管理员密码怎么办？**
A: 使用 `go run scripts/update_admin_password.go <新密码>` 重置密码。

**Q: 管理员账户被锁定了怎么办？**
A: 使用 `go run scripts/unlock_user.go admin` 解锁账户。

**Q: 如何创建多个管理员账户？**
A: 目前系统只支持一个管理员账户。如果需要多个管理员，可以创建普通用户并赋予相应权限（需要修改代码）。

**Q: 安装时没有创建管理员账户怎么办？**
A: 运行 `go run scripts/create_admin.go` 创建管理员账户。

**Q: 如何验证管理员账户是否创建成功？**
A: 尝试登录管理员后台，或检查数据库中的 `users` 表，查看 `is_admin` 字段为 `true` 的记录。

---

## 📊 功能列表

### ✅ 核心功能

#### 用户管理
- [x] 用户注册和登录
- [x] JWT 认证
- [x] 邮箱密码重置
- [x] 邮箱验证
- [x] 用户资料管理
- [x] 登录历史记录
- [x] 用户活动日志
- [x] 用户等级系统（含折扣）
- [x] 账户安全（支持 2FA）

#### 订阅管理
- [x] 订阅创建和续费
- [x] 设备数量限制管理
- [x] 到期时间控制
- [x] 订阅重置
- [x] 多种订阅类型
- [x] 订阅链接生成（Clash/V2Ray 格式）
- [x] 设备管理（添加、删除、查看）
- [x] 在线设备追踪
- [x] 设备指纹识别和 UA 检测

#### 订单管理
- [x] 订单创建和处理
- [x] 套餐订单
- [x] 设备升级订单
- [x] 订单取消
- [x] 订单状态追踪
- [x] 订单历史
- [x] 订单导出（CSV/Excel）
- [x] 批量操作

#### 支付集成
- [x] 支付宝集成
- [x] 微信支付集成
- [x] PayPal 集成
- [x] Apple Pay 集成
- [x] 余额支付
- [x] 混合支付（余额 + 第三方）
- [x] 支付回调处理
- [x] 支付交易追踪
- [x] 充值管理

#### 套餐管理
- [x] 套餐 CRUD 操作
- [x] 套餐定价
- [x] 套餐启用/停用
- [x] 套餐功能配置
- [x] 套餐显示顺序

#### 优惠券系统
- [x] 优惠券创建和管理
- [x] 折扣券（百分比）
- [x] 固定金额券
- [x] 优惠券代码验证
- [x] 优惠券使用追踪
- [x] 优惠券过期管理

#### 邀请系统
- [x] 邀请码生成
- [x] 邀请关系追踪
- [x] 邀请人奖励
- [x] 被邀请人奖励
- [x] 最低订单金额要求
- [x] 仅新用户奖励
- [x] 奖励自动分配

#### 节点管理
- [x] 节点 CRUD 操作
- [x] 节点健康监控
- [x] 节点状态追踪
- [x] 自定义节点支持
- [x] 节点分组
- [x] 节点订阅集成

#### 专线节点系统
- [x] 服务器管理（SSH 连接）
- [x] 自动节点部署（通过 XrayR API）
- [x] Cloudflare DNS 和证书自动化
- [x] 流量控制
- [x] 到期时间管理
- [x] 用户专属节点分配

#### 设备管理
- [x] 设备识别和指纹识别
- [x] 设备数量限制执行
- [x] 设备删除
- [x] 设备信息追踪（UA、IP 等）
- [x] 在线设备监控
- [x] 批量设备操作

#### 通知系统
- [x] 邮件通知
- [x] 站内通知
- [x] 通知模板
- [x] 通知偏好设置
- [x] 通知历史

#### 工单系统
- [x] 工单创建
- [x] 工单回复
- [x] 工单状态管理
- [x] 工单附件
- [x] 工单分配
- [x] 工单优先级

#### 统计和监控
- [x] 仪表盘统计
- [x] 用户统计
- [x] 订单统计
- [x] 收入统计
- [x] 订阅统计
- [x] 系统日志
- [x] 审计日志
- [x] 实时监控

#### 系统配置
- [x] 系统设置管理
- [x] 支付配置
- [x] 邮件配置
- [x] 短信配置
- [x] 安全设置
- [x] 功能开关
- [x] 公告管理

#### 备份和恢复
- [x] 数据库备份
- [x] 配置备份
- [x] 自动备份调度
- [x] 备份文件管理

---

## ⚙️ 配置说明

### 环境变量

主配置文件：`.env`

```env
# 服务器配置
HOST=127.0.0.1          # 只监听本地，通过 Nginx 反向代理
PORT=8000               # 后端服务端口

# 数据库配置（SQLite）
DATABASE_URL=sqlite:///./cboard.db

# JWT 配置（生产环境必须修改！）
SECRET_KEY=your-secret-key-here-change-in-production-min-32-chars

# CORS 配置（替换为您的域名）
BACKEND_CORS_ORIGINS=https://yourdomain.com,http://yourdomain.com

# 邮件配置（可选）
SMTP_HOST=smtp.qq.com
SMTP_PORT=587
SMTP_USERNAME=your-email@qq.com
SMTP_PASSWORD=your-smtp-password
SMTP_FROM_EMAIL=your-email@qq.com

# 调试模式
DEBUG=false
```

### Nginx 配置

安装脚本会自动配置 Nginx。如需手动调整：

1. 登录宝塔面板
2. **网站** → 找到您的网站 → **设置** → **配置文件**
3. 修改配置 → **保存** → **重载配置**

---

## 🛠️ 管理脚本使用说明

### 常用操作

#### 创建/重置管理员账号
```bash
sudo ./install.sh
# 选择选项 2
```

#### 重启服务
```bash
sudo ./install.sh
# 选择选项 8（标准重启）或 3（强制重启）
```

#### 查看服务状态
```bash
sudo ./install.sh
# 选择选项 6
```

#### 查看实时日志
```bash
sudo ./install.sh
# 选择选项 7
```

#### 停止服务
```bash
sudo ./install.sh
# 选择选项 9
```

### 手动管理命令

如果不想使用管理脚本，也可以直接使用 systemd 命令：

```bash
# 启动服务
systemctl start cboard

# 停止服务
systemctl stop cboard

# 重启服务
systemctl restart cboard

# 查看状态
systemctl status cboard

# 查看日志
journalctl -u cboard -f

# 设置开机自启
systemctl enable cboard
```

---

## 🔒 安全建议

1. **生产环境必须设置强密码**
   - `SECRET_KEY` 至少 32 位随机字符串
   - 管理员密码使用强密码

2. **使用 HTTPS**
   - 安装脚本会自动配置 SSL 证书
   - 确保强制 HTTPS 已开启

3. **配置 CORS**
   - 生产环境必须明确指定允许的域名
   - 不要使用通配符 `*`

4. **数据库安全**
   - 定期备份数据库
   - 使用 SQLite 时确保文件权限正确

5. **系统安全**
   - 定期更新系统和依赖
   - 配置防火墙规则
   - 使用强密码策略

---

## 📝 数据库备份

### 自动备份（推荐）

在宝塔面板中配置定时任务：

1. **计划任务** → **添加计划任务**
2. **任务类型**：Shell 脚本
3. **任务名称**：CBoard 数据库备份
4. **执行周期**：每天 0 点 2 分
5. **脚本内容**：
```bash
#!/bin/bash
cd /www/wwwroot/cboard
BACKUP_DIR="/www/backup/cboard"
mkdir -p $BACKUP_DIR
cp cboard.db $BACKUP_DIR/cboard_$(date +%Y%m%d_%H%M%S).db
# 保留最近 7 天的备份
find $BACKUP_DIR -name "cboard_*.db" -mtime +7 -delete
```

### 手动备份

```bash
cd /www/wwwroot/cboard
cp cboard.db cboard.db.backup.$(date +%Y%m%d_%H%M%S)
```

### 通过 API 备份

系统还提供备份 API 接口（仅管理员）：
- `POST /api/v1/admin/backup/create` - 创建备份

---

## 🔧 常见问题

### 1. 服务无法启动

**检查日志**：
```bash
# 查看服务日志
journalctl -u cboard -f

# 查看应用日志
tail -f /www/wwwroot/cboard/uploads/logs/app.log
```

**常见原因**：
- 端口被占用：检查 8000 端口是否被其他程序占用
- 权限问题：确保项目目录权限正确
- 配置文件错误：检查 `.env` 文件配置

### 2. 502 Bad Gateway

- 检查后端服务是否运行：`systemctl status cboard`
- 检查端口是否正确：`netstat -tlnp | grep 8000`
- 检查 Nginx 配置中的 `proxy_pass` 地址

### 3. SSL 证书申请失败

- 确保域名已正确解析到服务器 IP
- 确保 80 端口已开放
- 检查防火墙设置

### 4. 数据库权限错误

```bash
cd /www/wwwroot/cboard
chmod 666 cboard.db
chown www:www cboard.db
```

### 5. 前端无法访问后端 API

- 检查 `.env` 中的 `BACKEND_CORS_ORIGINS` 是否包含您的域名
- 检查 Nginx 配置中的 `/api/` 代理是否正确

### 6. 管理员登录问题

- 使用安装脚本重置管理员密码（选项 2）
- 检查管理员账号状态：`go run scripts/unlock_user.go <用户名或邮箱>`
- 解锁用户账号（支持管理员和普通用户）：`go run scripts/unlock_user.go <用户名或邮箱>`

---

## 📖 API 文档

启动服务器后，主要 API 端点：

### 认证
- `POST /api/v1/auth/register` - 用户注册
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/refresh` - 刷新令牌
- `POST /api/v1/auth/logout` - 用户登出

### 用户
- `GET /api/v1/users/me` - 获取当前用户
- `PUT /api/v1/users/me` - 更新用户资料
- `GET /api/v1/users/login-history` - 获取登录历史

### 订阅
- `GET /api/v1/subscriptions` - 获取订阅列表
- `GET /api/v1/subscriptions/:id` - 获取订阅详情
- `GET /subscribe/:url` - 获取订阅配置（Clash/V2Ray）

### 订单
- `GET /api/v1/orders` - 获取订单列表
- `POST /api/v1/orders` - 创建订单
- `GET /api/v1/orders/:id` - 获取订单详情
- `POST /api/v1/orders/:id/cancel` - 取消订单

### 套餐
- `GET /api/v1/packages` - 获取套餐列表
- `GET /api/v1/packages/:id` - 获取套餐详情

### 支付
- `POST /api/v1/payment/notify/:method` - 支付回调
- `GET /api/v1/payment/status/:orderNo` - 获取支付状态

### 管理员 API
所有管理员 API 需要管理员认证，前缀为 `/api/v1/admin/`

完整 API 列表请查看：`internal/api/router/router.go`

---

## 🏗️ 项目结构

```
goweb/
├── cmd/server/main.go          # 主入口
├── internal/
│   ├── api/                    # API 层
│   │   ├── handlers/           # 请求处理器
│   │   └── router/             # 路由定义
│   ├── core/                   # 核心模块
│   │   ├── auth/               # 认证
│   │   ├── config/             # 配置
│   │   └── database/           # 数据库
│   ├── models/                 # 数据模型
│   ├── services/               # 业务服务
│   ├── middleware/             # 中间件
│   └── utils/                  # 工具函数
├── frontend/                   # Vue 3 前端
│   ├── src/                    # 前端源代码
│   │   ├── views/              # 页面组件
│   │   ├── components/         # 可复用组件
│   │   ├── router/             # 前端路由
│   │   └── store/              # 状态管理
│   └── dist/                   # 构建后的文件
├── scripts/                    # 工具脚本
│   ├── create_admin.go         # 创建管理员账号
│   └── unlock_user.go          # 解锁用户账号（支持管理员和普通用户）
├── .env                        # 环境变量
├── install.sh                  # 宝塔面板安装脚本
├── cboard.db                   # SQLite 数据库
├── README.md                   # 英文版本文档
└── README_zh.md                # 中文版本文档（本文件）
```

---

## ⚠️ 重要注意事项

1. **首次设置**
   - 安装后，立即更改默认管理员密码
   - 更新 `.env` 文件中的 `SECRET_KEY`
   - 配置邮件设置以支持密码重置和通知

2. **数据库**
   - 默认使用 SQLite（无需安装）
   - 高流量生产环境建议使用 MySQL 或 PostgreSQL
   - 定期备份至关重要

3. **安全**
   - 永远不要将 `.env` 文件提交到版本控制
   - 所有账户使用强密码
   - 生产环境启用 HTTPS
   - 定期更新依赖

4. **性能**
   - 高流量场景建议使用 MySQL/PostgreSQL
   - 为静态文件启用 Nginx 缓存
   - 定期监控服务器资源

5. **更新**
   - 更新前始终备份数据库
   - 先在测试环境测试更新
   - 更新前查看更新日志

---

## 📚 功能文档

系统提供了详细的功能说明文档，帮助您更好地理解和使用各项功能：

### 📋 列表功能文档

- **[列表功能索引](./docs/list_functions_index.md)** - 所有列表功能的索引和快速查找

#### 管理员端列表功能

- **[用户列表管理](./docs/user_list_management.md)** - 用户信息管理、搜索、筛选、批量操作
- **[订阅列表管理](./docs/subscription_list_management.md)** - 订阅管理、设备数量限制原理详解
- **[节点列表管理](./docs/node_management.md)** - 节点采集、导入、管理、地区识别
- **[订单列表管理](./docs/order_management.md)** - 订单处理、支付管理、订单统计
- **[工单列表管理](./docs/ticket_management.md)** - 工单处理、回复、状态管理
- **[异常用户管理](./docs/abnormal_users_management.md)** - 异常用户识别和处理
- **[数据分析](./docs/statistics_analysis.md)** - 数据统计、地区分析、趋势分析

#### 用户端列表功能

- **[设备管理](./docs/device_management.md)** - 设备查看、删除、设备数量限制原理
- **[登录历史管理](./docs/login_history_management.md)** - 登录记录、地区信息、安全监控

### 🔧 技术文档

- **[专线节点实现文档](./docs/custom_node_implementation.md)** - 专线节点系统的完整实现说明
- **[GeoIP集成说明](./docs/GeoIP集成说明.md)** - GeoIP数据库集成和使用说明

### 📖 核心功能原理

#### 设备数量限制原理

设备数量限制是订阅管理的核心功能，详细原理请参考：

- **[订阅列表管理 - 设备数量限制原理](./docs/subscription_list_management.md#设备数量限制原理)**
- **[设备管理 - 设备数量限制原理](./docs/device_management.md#设备数量限制原理)**

**关键概念：**
- 设备识别：基于 User-Agent 和 IP 地址生成设备哈希
- 设备漫游：同一设备在不同网络环境下自动识别
- 限制机制：设备未超限/达到限制/超过限制的不同处理策略
- 优先级策略：采用"最近使用优先"策略，长时间未使用的设备自动淘汰

---

## 📞 技术支持

如遇到问题：

1. 查看日志文件：`/www/wwwroot/cboard/uploads/logs/app.log`
2. 查看服务日志：`journalctl -u cboard -f`
3. 检查系统资源：`htop` 或 `free -h`
4. 检查网络连接：`curl http://127.0.0.1:8000/health`
5. 查看本文档和故障排除部分
6. 参考 [功能文档](#-功能文档) 了解详细功能说明

---

## 📄 许可证

本项目采用 MIT 许可证。

---

**最后更新**: 2024-12-22  
**版本**: v1.0.0  
**状态**: ✅ 生产就绪

