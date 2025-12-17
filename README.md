# CBoard Go 版本 - 订阅管理系统

## 📖 系统简介

**CBoard** 是一个现代化的订阅管理系统，用于管理 VPN/代理服务的订阅、用户、订单、支付等业务。本版本使用 Go 语言重写，相比 Python 版本可以节省 **70-90% 的内存占用**。

### 🎯 核心特性

- 🚀 **高性能**: 内存占用仅 35-95 MB（Python 版本 300-850 MB）
- ⚡ **快速启动**: 毫秒级启动时间
- 🔒 **安全可靠**: JWT 认证、密码加密、SQL 注入防护
- 📦 **功能完整**: 包含所有核心业务功能
- 🎨 **现代化前端**: Vue 3 + Element Plus，响应式设计
- 🐳 **易于部署**: 支持宝塔面板一键安装，单一可执行文件

---

## 🚀 宝塔面板一键安装

### 前置条件

- ✅ 已安装宝塔面板（建议版本 7.0+）
- ✅ 服务器系统：Ubuntu 18.04+ / Debian 10+ / CentOS 7+
- ✅ 服务器配置：至少 1 核心 CPU + 512 MB 内存 + 10 GB 磁盘
- ✅ 已绑定域名（用于 SSL 证书）

### 安装步骤

#### 1. 上传项目文件

通过宝塔面板文件管理器或 SSH 将项目文件上传到服务器：

```bash
# 方式一：通过 Git 克隆
cd /www/wwwroot
git clone <repository-url> cboard
cd cboard

# 方式二：通过 SCP 上传（在本地执行）
scp -r /path/to/goweb/* root@your-server:/www/wwwroot/cboard/
```

#### 2. 运行安装脚本

通过 SSH 连接到服务器，执行：

```bash
cd /www/wwwroot/cboard

# 添加执行权限
chmod +x install.sh

# 运行安装脚本（需要 root 权限）
sudo ./install.sh
```

#### 3. 配置安装参数

安装脚本会提示您输入以下信息：

- **项目目录**：默认 `/www/wwwroot/dy.moneyfly.top`，可按需修改
- **域名**：输入您的域名（如：`example.com`）
- **管理员邮箱**：用于创建管理员账户
- **管理员密码**：设置管理员密码

#### 4. 选择安装选项

安装脚本提供以下功能：

```
==========================================
       CBoard Go 终极管理面板
==========================================
  1. 一键全自动部署 (SSL + 反代)
  2. 创建/重置管理员账号
  3. 强制重启服务 (杀进程后重启)
  4. 深度清理系统缓存
  5. 解锁管理员账户
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
- ✅ 编译后端服务
- ✅ 配置 Nginx 反向代理
- ✅ 申请 SSL 证书（Let's Encrypt）
- ✅ 创建 systemd 服务
- ✅ 启动服务

#### 5. 验证安装

安装完成后，访问您的域名：

- **前端界面**: `https://yourdomain.com`
- **健康检查**: `https://yourdomain.com/health`
- **API 文档**: `https://yourdomain.com/api/v1/...`

---

## 🛠️ 管理脚本使用说明

### 常用操作

#### 创建/重置管理员账号

```bash
sudo ./install.sh
# 选择 2
```

#### 重启服务

```bash
sudo ./install.sh
# 选择 8（标准重启）或 3（强制重启）
```

#### 查看服务状态

```bash
sudo ./install.sh
# 选择 6
```

#### 查看实时日志

```bash
sudo ./install.sh
# 选择 7
```

#### 停止服务

```bash
sudo ./install.sh
# 选择 9
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

## 📋 系统要求

### 最低配置要求

- **CPU**: 1 核心（推荐 2 核心+）
- **内存**: 512 MB（推荐 1 GB+）
- **磁盘**: 10 GB（推荐 20 GB+）
- **操作系统**: Ubuntu 18.04+ / Debian 10+ / CentOS 7+

### 软件要求

- **Go 语言**: 1.21+（安装脚本会自动安装）
- **数据库**: SQLite（默认，无需安装）或 MySQL 5.7+ / PostgreSQL 12+
- **Web 服务器**: Nginx（宝塔面板自带）

---

## ⚙️ 配置说明

### 环境变量配置

项目配置文件位于 `.env`，主要配置项：

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
```

### Nginx 配置

安装脚本会自动配置 Nginx，如需手动调整：

1. 登录宝塔面板
2. **网站** → 找到您的网站 → **设置** → **配置文件**
3. 修改配置后点击 **保存** → **重载配置**

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

---

## 📊 功能列表

### ✅ 已完成功能

- [x] 用户认证（注册、登录、JWT）
- [x] 用户管理（CRUD、权限）
- [x] 订阅管理
- [x] 订单管理
- [x] 套餐管理
- [x] 支付集成（支付宝、微信等）
- [x] 节点管理
- [x] 优惠券系统
- [x] 通知系统
- [x] 工单系统
- [x] 设备管理
- [x] 邀请码系统
- [x] 充值管理
- [x] 配置管理
- [x] 统计功能
- [x] 邮件服务
- [x] 短信服务
- [x] 前端界面（Vue 3 + Element Plus）

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

---

## 🏗️ 项目结构

```
goweb/
├── cmd/server/main.go          # 主入口
├── internal/
│   ├── api/                    # API 层
│   ├── core/                   # 核心模块
│   ├── models/                 # 数据模型
│   ├── services/               # 业务服务
│   ├── middleware/             # 中间件
│   └── utils/                  # 工具函数
├── frontend/                   # Vue 3 前端
│   ├── src/                    # 前端源代码
│   └── dist/                   # 构建后的文件
├── bin/                        # 编译后的可执行文件
├── scripts/                    # 工具脚本
├── .env                        # 环境变量配置
├── install.sh                  # 宝塔面板安装脚本
├── cboard.db                   # SQLite 数据库
└── README.md                   # 本文件
```

---

## 📖 API 文档

启动服务器后，主要 API 端点：

- `GET /health` - 健康检查
- `POST /api/v1/auth/register` - 注册
- `POST /api/v1/auth/login` - 登录
- `POST /api/v1/auth/refresh` - 刷新令牌
- `GET /api/v1/users/me` - 获取当前用户
- `GET /api/v1/subscriptions` - 获取订阅列表
- `GET /subscribe/:url` - 获取订阅配置（Clash）

完整 API 列表请查看代码中的路由定义：`internal/api/router/router.go`

---

## 🔧 技术栈

### 后端
- **Web 框架**: Gin
- **ORM**: GORM
- **数据库**: SQLite/MySQL/PostgreSQL
- **认证**: JWT
- **配置**: Viper

### 前端
- **框架**: Vue 3
- **UI 库**: Element Plus
- **构建工具**: Vite
- **状态管理**: Pinia
- **路由**: Vue Router

---

## 📞 技术支持

如遇到问题，请：

1. 查看日志文件：`/www/wwwroot/cboard/uploads/logs/app.log`
2. 查看服务日志：`journalctl -u cboard -f`
3. 检查系统资源：`htop` 或 `free -h`
4. 检查网络连接：`curl http://127.0.0.1:8000/health`

---

## 📄 许可证

本项目采用 MIT 许可证。

---

**最后更新**: 2024-12-15  
**版本**: v1.0.0  
**状态**: ✅ 生产就绪
