# 安装指南

## 🚀 一键安装（推荐）

### 使用安装脚本

```bash
# 下载安装脚本
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
chmod +x install.sh
sudo ./install.sh
```

安装脚本会自动完成：
1. 检测操作系统
2. 安装 Nginx、PHP 8.2+、MariaDB、Redis
3. 安装 Composer
4. 克隆项目并安装依赖
5. 配置数据库
6. 配置 Nginx
7. 初始化数据库
8. 配置定时任务
9. 配置 SSL 证书（可选）

### 安装过程

脚本会提示您输入：
- **域名**：您的网站域名（如 example.com）
- **SSL 配置**：是否现在配置 SSL 证书

安装完成后，您将获得：
- 数据库密码（请保存）
- Cookie 加密密钥
- WebAPI 密钥

## 📋 系统要求

### 硬件要求
- **最低配置**：1 核 CPU、1GB RAM、10GB 存储
- **推荐配置**：2 核 CPU、2GB RAM、20GB SSD

### 软件要求
- **操作系统**：
  - Debian 12 (推荐)
  - Ubuntu 20.04/22.04 LTS
  - CentOS 8+/RHEL 8+
- **软件版本**：
  - PHP 8.2+（推荐 8.4）
  - MariaDB 10.11+（推荐 11.8 LTS）
  - Redis 7.0+
  - Nginx 1.24+

## 🔧 手动安装

如果您想手动安装，请参考 [DEPLOY.md](DEPLOY.md) 获取详细步骤。

## ⚙️ 安装后配置

### 1. 登录管理后台

访问：`https://your-domain.com/auth/login`

使用安装时创建的管理员账户登录。

### 2. 配置节点采集功能

1. 进入 **节点管理** → **节点采集**
2. 添加采集源 URL（每行一个）
3. 设置采集间隔（建议 3600 秒）
4. 启用定时采集（可选）
5. 点击 **保存配置**
6. 点击 **立即采集** 测试功能

### 3. 配置其他功能

- **邮件服务**：进入"设置"→"邮件"配置邮件发送
- **支付网关**：进入"设置"→"支付"配置支付方式
- **其他设置**：根据需要进行配置

## 🔄 更新项目

```bash
cd /var/www/sspanel
git pull origin main
composer install --no-dev --optimize-autoloader
php xcat Migration
```

## 🐛 故障排查

### 常见问题

1. **502 Bad Gateway**
   - 检查 PHP-FPM 是否运行：`systemctl status php8.2-fpm`
   - 检查 Nginx 配置中的 PHP-FPM socket 路径

2. **数据库连接失败**
   - 检查数据库服务：`systemctl status mariadb`
   - 验证配置文件中的数据库信息

3. **权限错误**
   - 重置权限：
     ```bash
     chown -R www-data:www-data /var/www/sspanel
     chmod -R 777 /var/www/sspanel/storage
     ```

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

# 定时任务日志
tail -f /var/log/sspanel-cron.log
```

## 📚 更多帮助

- [DEPLOY.md](DEPLOY.md) - 详细部署指南
- [节点采集功能部署说明.md](节点采集功能部署说明.md) - 节点采集功能说明
- [GITHUB_SYNC_GUIDE.md](GITHUB_SYNC_GUIDE.md) - GitHub 同步指南

## 🔒 安全建议

1. **定期更新系统**
   ```bash
   apt update && apt upgrade -y
   ```

2. **使用强密码**
   - 数据库密码
   - 管理员账户密码
   - Redis 密码（如果设置）

3. **配置防火墙**
   ```bash
   ufw allow 22/tcp
   ufw allow 80/tcp
   ufw allow 443/tcp
   ufw enable
   ```

4. **定期备份**
   ```bash
   # 备份数据库
   mysqldump -u sspanel -p sspanel > backup_$(date +%Y%m%d).sql
   ```

---

**祝安装顺利！** 🎉
