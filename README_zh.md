# CBoard - 现代化订阅管理系统

[English](README.md) | 中文

---

## 📖 系统简介

**CBoard** 是一个现代化的高性能订阅管理系统，专为 VPN/代理服务提供商设计。使用 Go 语言构建，相比 Python 版本可节省 **70-90% 的内存占用**，同时保持完整的功能特性。

### 💡 项目初衷

本项目最初是为了解决一个实际需求：**将多个机场的订阅资源安全地分享给朋友，同时防止资源被滥用**。

**核心场景：**
- 拥有多个机场订阅，流量用不完，希望分享给外贸朋友使用
- 购买的机场套餐通常不限制设备数量，但需要控制分享范围
- 通过设备数量限制来防止朋友将订阅再次分享给他人
- 但存在一个问题：朋友可能将节点信息下载下来单独使用，这样就失去了控制

**解决方案：**
- 系统支持定期重置订阅地址（建议每天或每两天重置一次）
- 自动采集和聚合多个机场的订阅地址
- 生成新的聚合订阅链接分享给朋友
- 通过设备数量限制和订阅地址定期更新，有效防止资源被滥用

**适用场景：**
- 个人分享：将机场资源分享给少量朋友，控制使用范围
- 小规模商业化：如果朋友较多，也可以进行小规模商业化运营

这种设计既保证了资源的有效利用，又通过技术手段防止了资源的滥用和泄露。

### 🎯 核心特性

- 🚀 **高性能**: 内存占用仅 35-95 MB（Python 版本 300-850 MB）
- ⚡ **快速启动**: 毫秒级启动时间
- 🔒 **安全可靠**: JWT 认证、密码加密、SQL 注入防护
- 📦 **功能完整**: 包含所有核心业务功能
- 🎨 **现代化前端**: Vue 3 + Element Plus，响应式设计
- 🐳 **易于部署**: 支持无宝塔 VPS 一键脚本（install-vps.sh）与宝塔面板脚本（install.sh）
- 💳 **多支付方式**: 支持支付宝、微信支付、PayPal、Apple Pay、易支付
- 👥 **用户管理**: 完整的用户系统，包含等级、邀请、奖励
- 📊 **数据分析**: 全面的统计和监控功能
- 🎫 **工单系统**: 内置客户支持系统
- ⚙️ **Clash 配置**: 专业的 Clash 订阅配置系统，支持 16 个代理组和 3376 条分流规则

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
- **Node.js**: 16+（用于前端构建，安装脚本会自动安装）
- **Nginx**：宝塔环境由面板提供；无宝塔时由 `install-vps.sh` 自动安装
- **数据库**: SQLite（默认，无需安装）或 MySQL/PostgreSQL

---

## 🚀 安装指南

### 安装方式选择

本系统提供两套安装方式，请根据您的服务器环境选择：

| 环境 | 使用脚本 | 项目目录（常见） | 说明 |
|------|----------|------------------|------|
| **无宝塔面板**（纯 VPS） | `install-vps.sh` | `/opt/cboard` | 全自动安装 Go、Node.js、Nginx、Certbot，从 GitHub 拉代码并部署。**适合新 VPS 或未安装宝塔的服务器。** |
| **有宝塔面板** | `install.sh` | `/www/wwwroot/你的域名` | 在宝塔已创建的网站目录下部署，依赖宝塔的 Nginx 与站点。需先在宝塔中「添加站点」，再在该目录放入代码并运行脚本。 |

- **无宝塔**：详见下方 [无宝塔面板安装（纯 VPS）](#无宝塔面板安装纯-vps)，或直接查看 **[VPS 部署教程（无宝塔）](./docs/VPS部署教程-无宝塔.md)**。
- **有宝塔**：详见下方 [宝塔面板安装](#宝塔面板安装)。

---

## 一、无宝塔面板安装（纯 VPS）

适用于**未安装宝塔面板**的 VPS（Ubuntu / Debian / CentOS）。脚本会自动安装 Nginx、Go、Node.js、Certbot 并完成部署。

### 快速开始（无宝塔）

```bash
# 下载并运行安装脚本（需 root）
curl -sL https://raw.githubusercontent.com/moneyfly1/myweb/main/install-vps.sh -o install-vps.sh
sudo bash install-vps.sh
```

按提示输入**域名**、**项目目录**（默认 `/opt/cboard`）、**管理员账号**即可。若 GitHub 克隆失败（如国内网络），请先手动将代码放到安装目录再重新运行脚本，在「是否删除并重新下载」时选 **n**。详见：[VPS 部署教程 - 克隆失败时如何继续](./docs/VPS部署教程-无宝塔.md#克隆失败时如何继续代码下载失败)。

### 前置条件（无宝塔）

- ✅ 服务器系统：Ubuntu 18.04+ / Debian 10+ / CentOS 7+
- ✅ 服务器配置：至少 1 核心 CPU + 512 MB 内存 + 10 GB 磁盘
- ✅ 已绑定域名（用于 SSL 证书）
- ✅ 已配置域名 DNS 解析到服务器 IP
- ✅ 服务器已开放 80 和 443 端口

### 一键安装步骤

#### 步骤 1：下载并运行安装脚本

通过 SSH 连接到 VPS 后执行（或使用上方「快速开始」中的命令）：

```bash
curl -sL https://raw.githubusercontent.com/moneyfly1/myweb/main/install-vps.sh -o install-vps.sh
sudo bash install-vps.sh
```

**说明**：脚本会从 GitHub 下载代码并安装 Go、Node.js、Nginx、Certbot 等；若克隆失败，请按 [克隆失败时如何继续](./docs/VPS部署教程-无宝塔.md#克隆失败时如何继续代码下载失败) 操作后再重新运行脚本。

#### 步骤 2：按提示输入信息

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

#### 步骤 3：自动安装过程

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

#### 步骤 4：验证安装

安装完成后，访问您的域名：

- **前端界面**: `https://yourdomain.com`
- **管理员登录**: `https://yourdomain.com/admin/login`
- **健康检查**: `https://yourdomain.com/health`
- **API 接口**: `https://yourdomain.com/api/v1/...`

### 安装后管理（无宝塔）

以下路径以默认安装目录 `/opt/cboard` 为例；若安装时修改过目录，请替换为实际路径。

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
export ADMIN_EMAIL="admin@your-domain.com"
export ADMIN_PASSWORD="your-new-password"
go run scripts/admin_tool.go
```

### 注意事项（无宝塔）

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

## 二、宝塔面板安装

适用于**已安装宝塔面板**的服务器。需先在宝塔中「添加站点」，在网站根目录放入代码，再运行 `install.sh` 完成编译与 Nginx 配置。

### 前置条件（宝塔）

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
- **管理员邮箱**：输入管理员邮箱（如：`admin@your-domain.com`）
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

#### 步骤 6：验证安装（宝塔）

安装完成后，访问您的域名：

- **前端界面**: `https://yourdomain.com`
- **管理员登录**: `https://yourdomain.com/admin/login`
- **健康检查**: `https://yourdomain.com/health`
- **API 接口**: `https://yourdomain.com/api/v1/...`

### 安装后配置（宝塔）

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

以下命令中的**项目目录**请按实际部署方式替换：
- **无宝塔安装**：一般为 `/opt/cboard`
- **宝塔安装**：一般为 `/www/wwwroot/你的域名`（如 `/www/wwwroot/example.com`）

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
export ADMIN_EMAIL="admin@your-domain.com"
export ADMIN_PASSWORD="YourStrongPassword123!"

# 运行创建脚本
go run scripts/admin_tool.go
```

**说明**：
- 如果未设置环境变量，脚本会使用默认值（用户名：`admin`，邮箱：`admin@your-domain.com`，密码：`admin123`）
- 如果管理员账户已存在，脚本会更新该账户的信息
- 生产环境建议通过环境变量设置强密码

#### 方法三：使用 Go 脚本（交互式）

```bash
# 进入项目目录
cd /www/wwwroot/example.com

# 直接运行脚本（会使用默认值或提示输入）
go run scripts/admin_tool.go
```

### 修改管理员密码

如果忘记管理员密码，可以通过以下方式重置：

```bash
# 进入项目目录
cd /www/wwwroot/example.com

# 运行密码修改脚本（替换为您的实际密码）
go run scripts/admin_tool.go YourNewPassword123!

# 示例
go run scripts/admin_tool.go Sikeming001@
```

**说明**：
- 密码长度至少 6 位
- 脚本会自动查找管理员账户（用户名或邮箱为 `admin` 或配置的邮箱）
- 如果找不到管理员账户，请先创建账户

### 解锁用户账户

如果账户因多次登录失败被锁定，可以通过以下方式解锁：

```bash
# 进入项目目录
cd /www/wwwroot/example.com

# 解锁管理员账户（使用用户名）
go run scripts/unlock_user.go admin

# 或使用邮箱解锁
go run scripts/unlock_user.go admin@your-domain.com

# 解锁普通用户账户
go run scripts/unlock_user.go user@your-domain.com
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
A: 使用 `go run scripts/admin_tool.go <新密码>` 重置密码。

**Q: 管理员账户被锁定了怎么办？**
A: 使用 `go run scripts/unlock_user.go admin` 解锁账户。

**Q: 如何创建多个管理员账户？**
A: 目前系统只支持一个管理员账户。如果需要多个管理员，可以创建普通用户并赋予相应权限（需要修改代码）。

**Q: 安装时没有创建管理员账户怎么办？**
A: 运行 `go run scripts/admin_tool.go` 创建管理员账户。

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
- [x] Clash 配置模板系统（16 个代理组，3376 条规则）
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

# 邮件配置（可选，请按实际 SMTP 服务填写）
SMTP_HOST=
SMTP_PORT=587
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM_EMAIL=

# 调试模式
DEBUG=false
```

### Nginx 配置

- **宝塔**：安装脚本会自动写入站点 Nginx 配置；手动调整请登录宝塔 → **网站** → 对应站点 → **设置** → **配置文件** → 修改后保存并重载。
- **无宝塔**：脚本会生成 `/etc/nginx/sites-available/cboard`（或 CentOS 下 `/etc/nginx/conf.d/cboard.conf`）；修改后执行 `nginx -t` 与 `systemctl reload nginx`。

---

## 🛠️ 管理脚本使用说明

### 宝塔环境：使用 install.sh 菜单

在**宝塔安装**的服务器上，可在项目目录下运行 `./install.sh`，通过菜单操作：

| 操作 | 步骤 |
|------|------|
| 创建/重置管理员账号 | 运行 `sudo ./install.sh` → 选择 **2** |
| 重启服务 | 运行 `sudo ./install.sh` → 选择 **8**（标准）或 **3**（强制） |
| 查看服务状态 | 运行 `sudo ./install.sh` → 选择 **6** |
| 查看实时日志 | 运行 `sudo ./install.sh` → 选择 **7** |
| 停止服务 | 运行 `sudo ./install.sh` → 选择 **9** |

### 无宝塔环境：使用 systemd（推荐）

**无宝塔**安装后没有 `install.sh` 的交互菜单，请直接使用 systemd。以下命令**两种部署方式均可使用**：

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

### 自动备份（宝塔）

在**宝塔面板**中配置定时任务：

1. **计划任务** → **添加计划任务**
2. **任务类型**：Shell 脚本
3. **任务名称**：CBoard 数据库备份
4. **执行周期**：每天 0 点 2 分
5. **脚本内容**（请将 `PROJECT_DIR` 改为实际路径：宝塔一般为 `/www/wwwroot/你的域名`，无宝塔一般为 `/opt/cboard`）：
```bash
#!/bin/bash
PROJECT_DIR="/www/wwwroot/你的域名"   # 或 /opt/cboard
BACKUP_DIR="/www/backup/cboard"
mkdir -p $BACKUP_DIR
cp "$PROJECT_DIR/cboard.db" "$BACKUP_DIR/cboard_$(date +%Y%m%d_%H%M%S).db"
find $BACKUP_DIR -name "cboard_*.db" -mtime +7 -delete
```

### 手动备份

```bash
# 进入项目目录（宝塔示例：/www/wwwroot/example.com，无宝塔示例：/opt/cboard）
cd /opt/cboard
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

# 查看应用日志（无宝塔示例：/opt/cboard，宝塔示例：/www/wwwroot/你的域名）
tail -f /opt/cboard/server.log
```

**常见原因**：
- 端口被占用：检查 8000 端口是否被其他程序占用
- 权限问题：确保项目目录权限正确
- 配置文件错误：检查项目目录下的 `.env` 文件

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
# 进入项目目录（宝塔：/www/wwwroot/你的域名，无宝塔：/opt/cboard）
cd /opt/cboard
chmod 666 cboard.db
# 宝塔下若以 www 运行，可执行：chown www:www cboard.db
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
│   ├── admin_tool.go           # 创建/更新管理员账号和密码
│   └── unlock_user.go          # 解锁用户账号（支持管理员和普通用户）
├── .env                        # 环境变量
├── install.sh                  # 宝塔面板安装脚本（有宝塔时使用）
├── install-vps.sh              # 无宝塔 VPS 一键安装脚本（纯 VPS 使用）
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

## 📚 文档

### 部署与故障排查

| 文档 | 说明 |
|------|------|
| [文档目录](./docs/README.md) | 完整文档索引（功能 + 配置 + 部署） |
| [VPS 部署教程（无宝塔）](./docs/VPS部署教程-无宝塔.md) | 纯 VPS 一键部署步骤 |
| [安装问题排查指南](./docs/故障排查/安装问题排查指南.md) | 常见安装问题与解决方法 |

### 功能说明

| 文档 | 说明 |
|------|------|
| [列表功能索引](./docs/功能/列表功能索引.md) | 所有列表功能索引与导航 |
| [文档目录](./docs/README.md#中文) | 用户/订阅/订单/节点/工单/设备/登录历史/异常用户/数据分析等说明 |

### 后台与配置说明

#### 支付配置
- [支付宝配置说明](./docs/配置/支付宝配置说明.md) - 支付宝开放平台与系统后台配置
- [易支付配置指南](./docs/配置/易支付配置指南.md) - 易支付配置与使用

#### 通知配置
- [邮件服务器配置说明](./docs/配置/邮件服务器配置说明.md) - SMTP 邮件服务器配置
- [Telegram 通知配置说明](./docs/配置/Telegram通知配置说明.md) - Telegram Bot 通知配置
- [Bark 通知配置说明](./docs/配置/Bark通知配置说明.md) - Bark iOS 推送通知配置
- [客户通知设置说明](./docs/配置/客户通知设置说明.md) - 客户邮件通知开关配置

#### 系统设置
- [基本设置说明](./docs/配置/基本设置说明.md) - 网站信息、Logo、域名、GeoIP 等
- [注册设置说明](./docs/配置/注册设置说明.md) - 注册流程、密码要求、新用户默认订阅
- [安全设置与限制说明](./docs/配置/安全设置与限制说明.md) - 登录失败限制、锁定、IP 白名单、解锁方法
- [主题设置说明](./docs/配置/主题设置说明.md) - 默认主题、用户自定义权限、可用主题
- [公告管理说明](./docs/配置/公告管理说明.md) - 登录公告弹窗配置

#### 节点与备份
- [采集地址配置说明](./docs/配置/采集地址配置说明.md) - 节点采集地址（订阅 URL）配置
- [节点健康检查设置说明](./docs/配置/节点健康检查设置说明.md) - 节点自动检查间隔、延迟阈值等
- [备份设置说明](./docs/配置/备份设置说明.md) - 自动备份、Gitee/GitHub 备份配置
- [GitHub 配置说明](./docs/配置/GitHub配置说明.md) - 备份到 GitHub 的详细配置
- [Gitee 配置说明](./docs/配置/Gitee配置说明.md) - 备份到 Gitee 的详细配置

---

## 📞 技术支持

如遇到问题：

1. **查看日志**（项目目录：宝塔一般为 `/www/wwwroot/你的域名`，无宝塔一般为 `/opt/cboard`）：
   - 应用日志：`项目目录/server.log` 或 `项目目录/uploads/logs/app.log`
   - 服务日志：`journalctl -u cboard -f`

2. **检查系统状态**：
   - 系统资源：`htop` 或 `free -h`
   - 健康检查：`curl http://127.0.0.1:8000/health`
   - 服务状态：`systemctl status cboard`

3. **参考文档**：[安装问题排查指南](./docs/故障排查/安装问题排查指南.md)

---

## 📄 许可证

本项目采用 MIT 许可证。

---

---

## 🆕 最新更新

---

**最后更新**: 2026-02-10  
**版本**: v1.1.0  
**状态**: ✅ 生产就绪

