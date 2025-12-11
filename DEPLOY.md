# VPS 部署指南

本指南将帮助您在 VPS 上部署带有节点采集功能的 SSPanel-UIM。

## 📋 前置要求

### 系统要求
- **操作系统**: Debian 11+ / Ubuntu 20.04+ / CentOS 8+
- **PHP**: 8.2 或更高版本
- **数据库**: MariaDB 10.11+ / MySQL 8.0+
- **缓存**: Redis 7.0+
- **Web 服务器**: Nginx (推荐) 或 Apache
- **其他**: Git, Composer

### 服务器配置
- **CPU**: 至少 1 核心（推荐 2 核心）
- **内存**: 至少 1GB（推荐 2GB）
- **存储**: 至少 10GB（推荐 20GB SSD）

## 🚀 快速部署

### 1. 安装基础环境

#### Debian/Ubuntu

```bash
# 更新系统
apt update && apt upgrade -y

# 安装基础工具
apt install -y curl wget git unzip

# 安装 PHP 8.2
apt install -y php8.2 php8.2-fpm php8.2-cli php8.2-common \
    php8.2-mysql php8.2-zip php8.2-gd php8.2-mbstring \
    php8.2-curl php8.2-xml php8.2-bcmath php8.2-redis \
    php8.2-yaml php8.2-opcache

# 安装 MariaDB
apt install -y mariadb-server mariadb-client

# 安装 Redis
apt install -y redis-server

# 安装 Nginx
apt install -y nginx

# 安装 Composer
curl -sS https://getcomposer.org/installer | php
mv composer.phar /usr/local/bin/composer
chmod +x /usr/local/bin/composer
```

#### CentOS/RHEL

```bash
# 更新系统
yum update -y

# 安装 EPEL 和 Remi 仓库
yum install -y epel-release
yum install -y https://rpms.remirepo.net/enterprise/remi-release-8.rpm

# 安装 PHP 8.2
yum install -y php82 php82-php-fpm php82-php-cli \
    php82-php-mysqlnd php82-php-zip php82-php-gd \
    php82-php-mbstring php82-php-curl php82-php-xml \
    php82-php-bcmath php82-php-redis php82-php-yaml \
    php82-php-opcache

# 安装 MariaDB
yum install -y mariadb-server mariadb

# 安装 Redis
yum install -y redis

# 安装 Nginx
yum install -y nginx

# 安装 Composer
curl -sS https://getcomposer.org/installer | php
mv composer.phar /usr/local/bin/composer
chmod +x /usr/local/bin/composer
```

### 2. 配置数据库

```bash
# 启动 MariaDB
systemctl start mariadb
systemctl enable mariadb

# 安全配置（设置 root 密码）
mysql_secure_installation

# 创建数据库和用户
mysql -u root -p <<EOF
CREATE DATABASE sspanel CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'sspanel'@'localhost' IDENTIFIED BY 'your_strong_password';
GRANT ALL PRIVILEGES ON sspanel.* TO 'sspanel'@'localhost';
FLUSH PRIVILEGES;
EXIT;
EOF
```

### 3. 配置 Redis

```bash
# 启动 Redis
systemctl start redis
systemctl enable redis

# 如果需要设置密码（可选）
# 编辑 /etc/redis/redis.conf
# 找到 # requirepass foobared
# 取消注释并设置密码: requirepass your_redis_password
# 然后重启: systemctl restart redis
```

### 4. 克隆项目

```bash
# 创建项目目录
mkdir -p /var/www
cd /var/www

# 从 GitHub 克隆（替换为您的仓库地址）
git clone https://github.com/your-username/SSPanel-UIM.git sspanel
cd sspanel

# 或者如果您 fork 了原项目
# git clone https://github.com/Anankke/SSPanel-UIM.git sspanel
# cd sspanel
# git remote add upstream https://github.com/your-username/SSPanel-UIM.git
```

### 5. 安装依赖

```bash
cd /var/www/sspanel

# 安装 PHP 依赖
composer install --no-dev --optimize-autoloader
```

### 6. 配置项目

```bash
# 复制配置文件
cp config/.config.example.php config/.config.php
cp config/appprofile.example.php config/appprofile.php

# 编辑配置文件
nano config/.config.php
```

**重要配置项**（必须修改）：
```php
// 基本设置
$_ENV['key'] = 'your_random_string_here';  // 随机字符串，用于 Cookie 加密
$_ENV['baseUrl'] = 'https://your-domain.com';  // 您的域名

// WebAPI
$_ENV['muKey'] = 'your_random_string_here';  // 随机字符串，用于节点通信

// 数据库设置
$_ENV['db_host'] = 'localhost';
$_ENV['db_database'] = 'sspanel';
$_ENV['db_username'] = 'sspanel';
$_ENV['db_password'] = 'your_strong_password';

// Redis 设置（如果设置了密码）
$_ENV['redis_host'] = '127.0.0.1';
$_ENV['redis_port'] = 6379;
$_ENV['redis_password'] = 'your_redis_password';  // 如果设置了密码
```

### 7. 设置文件权限

```bash
cd /var/www/sspanel

# 设置目录权限
chown -R www-data:www-data /var/www/sspanel
chmod -R 755 /var/www/sspanel

# 设置存储目录权限
chmod -R 777 storage/
chmod -R 777 public/clients/
```

### 8. 运行数据库迁移

```bash
cd /var/www/sspanel

# 运行迁移（包括节点采集功能的迁移）
php xcat Migration
```

### 9. 创建管理员账户

```bash
cd /var/www/sspanel

# 创建管理员（会提示输入邮箱和密码）
php xcat User createAdmin
```

### 10. 配置 Nginx

创建 Nginx 配置文件：

```bash
nano /etc/nginx/sites-available/sspanel
```

添加以下内容（替换 `your-domain.com` 为您的域名）：

```nginx
server {
    listen 80;
    listen [::]:80;
    server_name your-domain.com;
    
    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name your-domain.com;
    
    root /var/www/sspanel/public;
    index index.php;
    
    # SSL 证书配置（使用 Let's Encrypt）
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    
    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    
    # 日志
    access_log /var/log/nginx/sspanel_access.log;
    error_log /var/log/nginx/sspanel_error.log;
    
    # 主配置
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }
    
    # PHP 处理
    location ~ \.php$ {
        fastcgi_pass unix:/var/run/php/php8.2-fpm.sock;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
        
        # 安全设置
        fastcgi_param HTTP_PROXY "";
        fastcgi_hide_header X-Powered-By;
    }
    
    # 禁止访问隐藏文件
    location ~ /\. {
        deny all;
    }
    
    # 禁止访问配置文件
    location ~ /config/ {
        deny all;
    }
    
    # 静态文件缓存
    location ~* \.(jpg|jpeg|png|gif|ico|css|js|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

启用站点：

```bash
# 创建符号链接
ln -s /etc/nginx/sites-available/sspanel /etc/nginx/sites-enabled/

# 测试配置
nginx -t

# 重启 Nginx
systemctl restart nginx
```

### 11. 配置 SSL 证书（Let's Encrypt）

```bash
# 安装 Certbot
apt install -y certbot python3-certbot-nginx

# 获取证书（会自动配置 Nginx）
certbot --nginx -d your-domain.com

# 自动续期
certbot renew --dry-run
```

### 12. 配置定时任务

```bash
# 编辑 crontab
crontab -e

# 添加以下行（每 5 分钟执行一次）
*/5 * * * * cd /var/www/sspanel && /usr/bin/php xcat Cron >> /dev/null 2>&1
```

### 13. 配置防火墙

```bash
# UFW (Ubuntu)
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable

# firewalld (CentOS)
firewall-cmd --permanent --add-service=ssh
firewall-cmd --permanent --add-service=http
firewall-cmd --permanent --add-service=https
firewall-cmd --reload
```

## 🎯 节点采集功能配置

部署完成后，配置节点采集功能：

1. **登录管理后台**
   - 访问 `https://your-domain.com/auth/login`
   - 使用创建的管理员账户登录

2. **配置节点采集**
   - 进入 **节点管理** → **节点采集**
   - 添加采集源 URL（每行一个）
   - 设置采集间隔（建议 3600 秒）
   - 启用定时采集（可选）
   - 点击 **保存配置**

3. **测试采集**
   - 点击 **立即采集** 按钮
   - 查看采集日志确认是否成功
   - 返回节点列表查看采集到的节点

## 🔄 更新项目

### 从 GitHub 更新

```bash
cd /var/www/sspanel

# 拉取最新代码
git pull origin master

# 更新依赖
composer install --no-dev --optimize-autoloader

# 运行迁移（如果有新的迁移）
php xcat Migration

# 清除缓存
rm -rf storage/framework/smarty/cache/*
rm -rf storage/framework/smarty/compile/*
```

## 🛠️ 故障排查

### 常见问题

1. **502 Bad Gateway**
   - 检查 PHP-FPM 是否运行: `systemctl status php8.2-fpm`
   - 检查 Nginx 配置中的 PHP-FPM socket 路径

2. **数据库连接失败**
   - 检查数据库服务是否运行: `systemctl status mariadb`
   - 检查配置文件中的数据库凭据

3. **权限错误**
   - 检查文件权限: `ls -la /var/www/sspanel`
   - 确保 `storage/` 目录可写

4. **节点采集失败**
   - 检查采集源 URL 是否可访问
   - 查看采集日志了解具体错误
   - 确认网络连接正常

### 查看日志

```bash
# Nginx 错误日志
tail -f /var/log/nginx/sspanel_error.log

# PHP-FPM 日志
tail -f /var/log/php8.2-fpm.log

# 系统日志
journalctl -u nginx -f
journalctl -u php8.2-fpm -f
```

## 📚 更多信息

- 项目文档: https://docs.sspanel.io
- GitHub: https://github.com/Anankke/SSPanel-UIM
- 节点采集功能说明: 查看 `节点采集功能部署说明.md`

## 🔒 安全建议

1. **定期更新系统**
   ```bash
   apt update && apt upgrade -y
   ```

2. **使用强密码**
   - 数据库密码
   - Redis 密码（如果设置）
   - 管理员账户密码

3. **配置防火墙**
   - 只开放必要的端口
   - 使用 fail2ban 防止暴力破解

4. **定期备份**
   ```bash
   # 备份数据库
   mysqldump -u sspanel -p sspanel > backup_$(date +%Y%m%d).sql
   
   # 备份配置文件
   tar -czf config_backup_$(date +%Y%m%d).tar.gz config/
   ```

5. **监控系统**
   - 监控服务器资源使用
   - 监控网站访问日志
   - 设置告警通知

---

**祝部署顺利！** 🚀
