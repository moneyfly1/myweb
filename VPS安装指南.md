# SSPanel-UIM VPS 安装指南

## 📋 前提条件

在开始安装之前，请确保：

- ✅ 已购买 VPS 服务器（推荐配置：2核CPU、2GB内存、20GB SSD）
- ✅ 已获取 VPS 的 root 密码或 SSH 密钥
- ✅ 已准备好域名（可选，但生产环境强烈推荐）
- ✅ 代码已上传到 GitHub（仓库：`moneyfly1/myweb`）

---

## 🚀 方法一：一键自动安装（推荐）

这是最简单快捷的安装方式，安装脚本会自动从 GitHub 克隆代码并完成所有配置。

### 步骤 1: 连接到 VPS

```bash
# 使用 SSH 连接到您的 VPS
ssh root@your-vps-ip

# 或者使用密钥文件
ssh -i /path/to/your-key.pem root@your-vps-ip
```

### 步骤 2: 选择安装目录

**重要提示**：安装脚本会在**当前目录**安装项目，所以必须先进入您想安装的目录。

根据您的系统选择目录：

```bash
# 检查常见目录是否存在
ls -d /www/wwwroot 2>/dev/null && echo "✅ 宝塔面板目录存在" || echo "❌ 宝塔目录不存在"
ls -d /var/www 2>/dev/null && echo "✅ 标准目录存在" || echo "❌ 标准目录不存在"
ls -d /opt 2>/dev/null && echo "✅ /opt 目录存在" || echo "❌ /opt 目录不存在"
```

**推荐目录选择**：
- **宝塔面板用户**：`/www/wwwroot/sspanel`（最推荐）
- **标准 Linux 系统**：`/opt/sspanel` 或 `/home/www/sspanel`
- **测试环境**：`~/sspanel`（当前用户目录）

### 步骤 3: 下载安装脚本

```bash
# 先进入您选择的安装目录（例如：宝塔面板用户）
cd /www/wwwroot
mkdir -p sspanel
cd sspanel

# 下载安装脚本
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh

# 或者如果 wget 不可用，使用 curl
curl -O https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh

# 添加执行权限
chmod +x install.sh
```

### 步骤 4: 运行安装脚本

**重要**：安装脚本会在**当前目录**安装项目，所以先进入您想安装的目录。

```bash
# 方式 A: 宝塔面板用户（推荐使用宝塔标准目录）
cd /www/wwwroot
mkdir -p sspanel
cd sspanel
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
chmod +x install.sh
sudo ./install.sh

# 方式 B: 标准 Linux 系统（如果 /var/www 存在）
cd /var/www
sudo mkdir -p sspanel
cd sspanel
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
chmod +x install.sh
sudo ./install.sh

# 方式 C: 安装到当前用户目录（最简单，适合测试）
cd ~
mkdir sspanel
cd sspanel
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
chmod +x install.sh
sudo ./install.sh

# 方式 D: 自定义目录
cd /your/custom/path
mkdir -p sspanel
cd sspanel
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
chmod +x install.sh
sudo ./install.sh
```

**如何选择目录？**
- **宝塔面板用户**：使用 `/www/wwwroot/sspanel`（最推荐）
- **标准 Linux 系统**：使用 `/opt/sspanel` 或 `/home/www/sspanel`
- **测试环境**：可以使用 `~/sspanel` 或 `/home/username/sspanel`

**重要**：安装脚本会在**当前目录**安装，所以必须先 `cd` 到目标目录再运行脚本！

### 步骤 4: 按照提示完成安装

安装脚本会自动执行以下操作：

1. ✅ 检测操作系统
2. ✅ 安装基础工具（Git、curl、wget 等）
3. ✅ 安装 Nginx
4. ✅ 安装 PHP 8.2+ 及所有必需扩展
5. ✅ 安装 MariaDB/MySQL
6. ✅ 安装 Redis
7. ✅ 安装 Composer
8. ✅ **从 GitHub 自动克隆代码**（`https://github.com/moneyfly1/myweb.git`）
9. ✅ 安装 Composer 依赖
10. ✅ 配置数据库
11. ✅ 配置 Nginx
12. ✅ 初始化数据库
13. ✅ 创建管理员账号
14. ✅ 配置定时任务
15. ✅ 配置 SSL 证书（可选）

### 安装过程中的提示

安装脚本会询问您以下信息：

1. **域名**：输入您的网站域名（例如：`example.com`）
2. **数据库 root 密码**：如果数据库未安装，会要求设置 root 密码
3. **数据库名称**：默认 `sspanel`，可以自定义
4. **数据库用户名**：默认 `sspanel`，可以自定义
5. **数据库密码**：设置数据库用户密码
6. **管理员邮箱**：用于创建管理员账号
7. **管理员密码**：设置管理员登录密码
8. **是否配置 SSL**：选择是否现在配置 HTTPS 证书

---

## 🔧 方法二：手动安装（适合自定义需求）

如果您需要更多控制，可以手动执行安装步骤。

### 步骤 1: 连接到 VPS 并准备环境

```bash
# SSH 连接到 VPS
ssh root@your-vps-ip

# 更新系统
apt update && apt upgrade -y  # Debian/Ubuntu
# 或
dnf update -y  # CentOS/RHEL

# 安装基础工具
apt install -y curl wget git unzip  # Debian/Ubuntu
# 或
dnf install -y curl wget git unzip  # CentOS/RHEL
```

### 步骤 2: 选择目录并从 GitHub 克隆代码

**选择安装目录**（根据您的系统）：
- 宝塔面板：`/www/wwwroot/sspanel`
- 标准系统：`/opt/sspanel` 或 `/home/www/sspanel`
- 测试环境：`~/sspanel`

```bash
# 创建安装目录（根据您的系统选择）
# 宝塔面板用户：
mkdir -p /www/wwwroot/sspanel
cd /www/wwwroot/sspanel

# 或者标准 Linux 系统：
# mkdir -p /opt/sspanel
# cd /opt/sspanel

# 从 GitHub 克隆代码
git clone https://github.com/moneyfly1/myweb.git .

# 如果遇到 "dubious ownership" 错误，执行：
git config --global --add safe.directory $(pwd)
```

### 步骤 3: 运行安装脚本

```bash
# 进入项目目录
cd /var/www/sspanel

# 运行安装脚本
chmod +x install.sh
sudo ./install.sh
```

安装脚本会检测到代码已存在，会跳过克隆步骤，直接进行环境配置。

---

## 📝 安装后的配置

### 1. 访问网站

安装完成后，访问：
- **网站首页**: `https://your-domain.com`
- **管理后台**: `https://your-domain.com/auth/login`
- **节点采集管理**: `https://your-domain.com/admin/node/collector`

### 2. 登录管理后台

使用安装时创建的管理员账号登录：
- **邮箱**: 安装时输入的管理员邮箱
- **密码**: 安装时设置的管理员密码

### 3. 配置节点采集功能

1. 登录管理后台
2. 进入 **节点管理** → **节点采集**
3. 添加采集源 URL
4. 配置采集参数
5. 测试采集功能

---

## 🔍 常见问题排查

### 问题 1: 无法从 GitHub 克隆代码

**错误信息**: `fatal: could not read Username for 'https://github.com'`

**解决方案**:
```bash
# 确保使用公开仓库的 HTTPS URL
git clone https://github.com/moneyfly1/myweb.git .

# 如果是私有仓库，需要配置 GitHub Token
git clone https://YOUR_TOKEN@github.com/moneyfly1/myweb.git .
```

### 问题 2: Git "dubious ownership" 错误

**错误信息**: `fatal: detected dubious ownership in repository`

**解决方案**:
```bash
# 方法 1: 添加安全目录
git config --global --add safe.directory /var/www/sspanel

# 方法 2: 修改目录所有者
chown -R root:root /var/www/sspanel
```

### 问题 3: 安装脚本无法下载

**解决方案**:
```bash
# 检查网络连接
ping github.com

# 使用代理（如果需要）
export http_proxy=http://your-proxy:port
export https_proxy=http://your-proxy:port

# 或者手动下载安装脚本
curl -O https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
```

### 问题 4: Composer 依赖安装失败

**解决方案**:
```bash
# 检查 PHP 扩展
php -m

# 确保所有必需扩展已安装
apt install -y php8.2-{bcmath,curl,fileinfo,mbstring,mysqli,redis,xml,yaml,zip,gmp}

# 清理 Composer 缓存
composer clear-cache

# 重新安装依赖
composer install --no-dev --optimize-autoloader
```

### 问题 5: 数据库连接失败

**解决方案**:
```bash
# 检查数据库服务状态
systemctl status mariadb
# 或
systemctl status mysql

# 测试数据库连接
mysql -u root -p

# 检查数据库配置
cat /var/www/sspanel/config/.config.php
```

### 问题 6: Nginx 配置错误

**解决方案**:
```bash
# 检查 Nginx 配置语法
nginx -t

# 查看 Nginx 错误日志
tail -f /var/log/nginx/error.log

# 检查 Nginx 服务状态
systemctl status nginx
```

---

## 🔄 更新代码

如果代码有更新，需要从 GitHub 拉取最新代码：

```bash
# 进入项目目录
cd /var/www/sspanel

# 拉取最新代码
git pull origin main

# 如果遇到冲突，可以强制更新（注意：会丢失本地修改）
git fetch origin main
git reset --hard origin/main

# 更新 Composer 依赖
composer install --no-dev --optimize-autoloader

# 运行数据库迁移（如果有）
php xcat Migration latest

# 清除缓存
php xcat Tool clearCache
```

---

## 📊 系统服务管理

### 查看服务状态

```bash
# 查看所有相关服务状态
systemctl status nginx
systemctl status mariadb  # 或 mysql
systemctl status redis
systemctl status php8.2-fpm  # 根据 PHP 版本调整
```

### 重启服务

```bash
# 重启 Nginx
systemctl restart nginx

# 重启数据库
systemctl restart mariadb  # 或 mysql

# 重启 Redis
systemctl restart redis

# 重启 PHP-FPM
systemctl restart php8.2-fpm
```

### 查看日志

```bash
# Nginx 访问日志
tail -f /var/log/nginx/access.log

# Nginx 错误日志
tail -f /var/log/nginx/error.log

# PHP-FPM 日志
tail -f /var/log/php8.2-fpm.log

# 应用日志
tail -f /var/www/sspanel/storage/logs/app.log

# 定时任务日志
tail -f /var/log/sspanel-cron.log
```

---

## 🔐 安全建议

### 1. 防火墙配置

```bash
# Ubuntu/Debian (UFW)
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw enable

# CentOS/RHEL (firewalld)
firewall-cmd --permanent --add-service=ssh
firewall-cmd --permanent --add-service=http
firewall-cmd --permanent --add-service=https
firewall-cmd --reload
```

### 2. 修改默认端口（可选）

```bash
# 修改 SSH 端口（编辑 /etc/ssh/sshd_config）
Port 2222

# 重启 SSH 服务
systemctl restart sshd
```

### 3. 配置 SSL 证书

```bash
# 使用 Let's Encrypt 免费证书
certbot --nginx -d your-domain.com

# 自动续期
systemctl enable certbot-renew.timer
```

### 4. 定期备份

```bash
# 备份数据库
mysqldump -u root -p sspanel > /backup/sspanel_$(date +%Y%m%d).sql

# 备份代码和配置文件
tar -czf /backup/sspanel_$(date +%Y%m%d).tar.gz /var/www/sspanel
```

---

## 📞 获取帮助

如果遇到问题，可以：

1. 查看项目文档：`README.md`
2. 查看系统配置要求：`系统配置要求.md`
3. 检查日志文件
4. 提交 Issue 到 GitHub

---

## ✅ 安装检查清单

安装完成后，请确认：

- [ ] 网站可以正常访问（HTTP/HTTPS）
- [ ] 管理后台可以登录
- [ ] 数据库连接正常
- [ ] Redis 服务运行正常
- [ ] 定时任务已配置（`crontab -l` 查看）
- [ ] SSL 证书已配置（生产环境）
- [ ] 防火墙规则已配置
- [ ] 管理员账号已创建
- [ ] 节点采集功能可以正常使用

---

**最后更新**: 2024-12-12  
**GitHub 仓库**: https://github.com/moneyfly1/myweb
