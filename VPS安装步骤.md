# 🚀 VPS 一键安装步骤

## ✅ 代码已同步到 GitHub

您的代码已成功同步到：**https://github.com/moneyfly1/myweb**

---

## 📋 在 VPS 上一键安装

### 完整步骤

#### 1. SSH 登录 VPS

```bash
ssh root@your-vps-ip
# 输入密码
```

#### 2. 下载安装脚本

```bash
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
```

如果 `wget` 不可用，使用 `curl`：

```bash
curl -O https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
```

#### 3. 添加执行权限

```bash
chmod +x install.sh
```

#### 4. 运行安装脚本

```bash
sudo ./install.sh
```

#### 5. 按照提示输入

**输入域名：**
```
请输入您的域名 (例如: example.com): your-domain.com
```

**配置 SSL（推荐选择 Y）：**
```
是否现在配置 SSL 证书? (Y/n): Y
```

#### 6. 等待安装完成

脚本会自动完成：
- ✅ 安装 Nginx、PHP 8.2+、MariaDB、Redis
- ✅ 安装 Composer
- ✅ 从 GitHub 克隆项目
- ✅ 安装 PHP 依赖
- ✅ 自动生成数据库密码和密钥
- ✅ 配置项目
- ✅ 设置文件权限
- ✅ 配置 Nginx
- ✅ 初始化数据库（包括节点采集功能）
- ✅ 创建管理员账户
- ✅ 配置定时任务
- ✅ 配置 SSL 证书

**安装时间：约 10-20 分钟**

#### 7. 保存重要信息

安装完成后，脚本会显示：
- 🔑 **数据库密码** - 请务必保存
- 🔑 **Cookie 加密密钥** - 请务必保存
- 🔑 **WebAPI 密钥** - 请务必保存

#### 8. 访问网站

安装完成后，访问：
- 🌐 **网站**: `https://your-domain.com`
- 🔐 **管理后台**: `https://your-domain.com/auth/login`
- 📊 **节点采集**: `https://your-domain.com/admin/node/collector`

---

## 🎯 配置节点采集功能

### 步骤 1: 登录管理后台

1. 访问 `https://your-domain.com/auth/login`
2. 使用安装时创建的管理员账户登录

### 步骤 2: 配置节点采集

1. 点击左侧菜单 **节点管理** → **节点采集**
2. 在"采集源 URL 列表"中添加节点源（每行一个）：
   ```
   https://example.com/subscribe1
   https://example.com/subscribe2
   ```
3. 设置"更新间隔"（建议 3600 秒，即 1 小时）
4. 启用"定时采集"（可选）
5. 添加"过滤关键词"（可选）
6. 点击 **保存配置**

### 步骤 3: 测试采集

1. 点击 **立即采集** 按钮
2. 查看采集日志确认是否成功
3. 返回 **节点管理** → **节点列表** 查看采集到的节点

---

## 📝 完整命令示例

```bash
# ===== 在 VPS 上执行 =====

# 1. SSH 登录（在本地终端）
ssh root@your-vps-ip

# 2. 下载安装脚本
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
chmod +x install.sh

# 3. 运行安装脚本
sudo ./install.sh

# 按照提示输入：
# - 域名: your-domain.com
# - SSL 配置: Y

# 4. 等待安装完成（10-20 分钟）

# 5. 访问网站
# https://your-domain.com/auth/login
```

---

## ⚠️ 注意事项

### 安装前

1. **确保 VPS 资源充足**
   - 至少 1GB RAM
   - 至少 10GB 存储空间

2. **确保域名已解析**
   - 将域名 A 记录指向 VPS IP
   - 等待 DNS 生效

3. **确保防火墙开放端口**
   ```bash
   # Ubuntu/Debian
   ufw allow 22/tcp
   ufw allow 80/tcp
   ufw allow 443/tcp
   
   # CentOS
   firewall-cmd --permanent --add-service=ssh
   firewall-cmd --permanent --add-service=http
   firewall-cmd --permanent --add-service=https
   firewall-cmd --reload
   ```

### 安装后

1. **立即保存密码和密钥**
   - 数据库密码
   - Cookie 密钥
   - WebAPI 密钥

2. **修改管理员密码**
   - 登录后立即修改

3. **配置节点采集**
   - 添加采集源 URL
   - 测试采集功能

---

## 🐛 故障排查

### 安装脚本失败

```bash
# 查看错误信息
# 检查网络连接
ping github.com

# 检查系统版本
cat /etc/os-release
```

### 网站无法访问

```bash
# 检查服务状态
systemctl status nginx
systemctl status php8.2-fpm  # 或 php-fpm
systemctl status mariadb
systemctl status redis

# 检查 Nginx 配置
nginx -t

# 查看错误日志
tail -f /var/log/nginx/sspanel_error.log
```

### 数据库连接失败

```bash
# 检查数据库服务
systemctl status mariadb

# 测试数据库连接
mysql -u sspanel -p -h 127.0.0.1 sspanel
# 输入安装时显示的数据库密码
```

---

## 📚 相关文档

- [INSTALL.md](INSTALL.md) - 详细安装指南
- [DEPLOY.md](DEPLOY.md) - VPS 部署完整指南
- [节点采集功能部署说明.md](节点采集功能部署说明.md) - 节点采集功能说明

---

## ✅ 安装检查清单

- [ ] VPS 资源充足（1GB+ RAM）
- [ ] 域名已解析到 VPS IP
- [ ] 防火墙已开放端口
- [ ] 已下载 install.sh
- [ ] 已运行安装脚本
- [ ] 已保存密码和密钥
- [ ] 网站可以访问
- [ ] 管理后台可以登录
- [ ] 节点采集功能已配置

---

**祝安装顺利！** 🎉
