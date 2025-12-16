# CBoard Go 一键安装脚本使用说明

## 📋 简介

`bt-deploy.sh` 是一个全自动安装脚本，可以自动完成以下任务：

- ✅ 自动检测并安装 Go 语言（如果未安装）
- ✅ 自动检测并安装 Node.js（如果未安装）
- ✅ 自动安装 Go 依赖
- ✅ 自动编译后端服务
- ✅ 自动安装前端依赖并构建
- ✅ 自动创建配置文件（.env）
- ✅ 自动生成强密码 SECRET_KEY
- ✅ 自动创建必要目录
- ✅ 自动设置文件权限
- ✅ 自动创建 systemd 服务
- ✅ 自动生成 Nginx 配置
- ✅ 自动测试服务运行状态

## 🚀 快速开始

### 方法一：直接运行（推荐）

```bash
# 1. 进入项目目录
cd /www/wwwroot/your-domain.com

# 2. 添加执行权限
chmod +x bt-deploy.sh

# 3. 运行安装脚本
sudo ./bt-deploy.sh
```

### 方法二：指定参数运行

```bash
# 指定项目目录和域名
sudo ./bt-deploy.sh -d /www/wwwroot/my-site.com -n my-site.com
```

### 方法三：使用环境变量

```bash
# 设置环境变量后运行
export PROJECT_DIR="/www/wwwroot/my-site.com"
export DOMAIN="my-site.com"
sudo ./bt-deploy.sh
```

## 📝 命令行参数

```bash
./bt-deploy.sh [选项]

选项:
    -d, --dir DIR         项目目录（默认: /www/wwwroot/dy.moneyfly.top）
    -n, --domain DOMAIN   域名（默认: 从目录名自动检测）
    -g, --go-version VER   Go 版本（默认: 1.21.5）
    -N, --node-version VER Node.js 版本（默认: 18）
    -s, --skip-tests      跳过服务测试
    -h, --help            显示帮助信息
```

## 🔧 环境变量

可以通过环境变量覆盖默认配置：

```bash
export PROJECT_DIR="/www/wwwroot/my-site.com"
export DOMAIN="my-site.com"
export GO_VERSION="1.21.5"
export NODE_VERSION="18"
export SKIP_TESTS="true"  # 跳过测试
```

## 📋 安装步骤说明

脚本会自动执行以下步骤：

1. **环境检查**
   - 检查 root 权限
   - 检测操作系统类型
   - 检测宝塔面板环境

2. **安装 Go 语言**
   - 如果未安装，自动下载并安装 Go
   - 自动配置 PATH 环境变量

3. **安装 Node.js**
   - 如果未安装，自动安装 Node.js 和 npm
   - 支持 Ubuntu/Debian/CentOS/Rocky Linux

4. **项目配置**
   - 创建项目目录（如果不存在）
   - 自动生成 .env 配置文件
   - 自动生成强密码 SECRET_KEY

5. **编译后端**
   - 安装 Go 依赖
   - 编译后端服务

6. **构建前端**
   - 安装前端依赖
   - 构建生产版本

7. **系统配置**
   - 创建必要目录
   - 设置文件权限
   - 创建 systemd 服务

8. **测试验证**
   - 测试后端服务启动
   - 验证健康检查接口

9. **生成配置**
   - 生成 Nginx 配置文件

## 📂 安装后的目录结构

```
/www/wwwroot/your-domain.com/
├── server                  # 后端可执行文件
├── .env                    # 环境变量配置
├── cboard.db              # SQLite 数据库（首次运行后创建）
├── frontend/
│   └── dist/              # 前端构建文件
├── uploads/               # 上传目录
│   ├── avatars/          # 头像
│   ├── config/           # 配置文件
│   └── logs/             # 日志文件
└── bt-deploy.sh          # 安装脚本
```

## 🔍 安装后检查

### 1. 检查服务状态

```bash
systemctl status cboard
```

### 2. 检查后端健康

```bash
curl http://127.0.0.1:8000/health
```

应该返回：
```json
{"status":"healthy","version":"1.0.0"}
```

### 3. 查看日志

```bash
# 系统日志
journalctl -u cboard -f

# 应用日志
tail -f /www/wwwroot/your-domain.com/uploads/logs/app.log

# 安装日志
cat /tmp/cboard_install_*.log
```

## 🌐 配置 Nginx

脚本会自动生成 Nginx 配置文件，位置在：
```
/tmp/cboard_nginx_你的域名.conf
```

### 在宝塔面板中配置：

1. **创建网站**
   - 登录宝塔面板
   - 网站 → 添加站点
   - 域名：你的域名
   - 根目录：`/www/wwwroot/your-domain.com/frontend/dist`
   - PHP 版本：纯静态

2. **配置 Nginx**
   - 网站 → 设置 → 配置文件
   - 复制 `/tmp/cboard_nginx_你的域名.conf` 的内容
   - 粘贴到配置文件中
   - 保存

3. **配置 SSL 证书（推荐）**
   - 网站 → 设置 → SSL
   - 选择 Let's Encrypt
   - 申请证书
   - 开启强制 HTTPS

## 🎯 启动服务

```bash
# 启动服务
systemctl start cboard

# 停止服务
systemctl stop cboard

# 重启服务
systemctl restart cboard

# 查看状态
systemctl status cboard

# 设置开机自启（已自动设置）
systemctl enable cboard
```

## 👤 创建管理员账户

安装完成后，需要创建管理员账户：

```bash
cd /www/wwwroot/your-domain.com
go run scripts/create_admin.go
```

或使用已编译的二进制文件：

```bash
cd /www/wwwroot/your-domain.com
./server --create-admin
```

## ❓ 常见问题

### 1. 安装失败：Go 下载失败

**原因**：网络连接问题

**解决**：
```bash
# 手动下载 Go
cd /tmp
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
# 然后重新运行脚本
```

### 2. 前端构建失败

**原因**：Node.js 版本不兼容或网络问题

**解决**：
```bash
# 检查 Node.js 版本
node -v  # 应该是 16+

# 清理并重新安装
cd frontend
rm -rf node_modules package-lock.json
npm cache clean --force
npm install --legacy-peer-deps
npm run build
```

### 3. 端口 8000 被占用

**原因**：其他服务正在使用该端口

**解决**：
```bash
# 查看占用端口的进程
netstat -tlnp | grep 8000
# 或
ss -tlnp | grep 8000

# 停止占用端口的服务，或修改 .env 中的 PORT
```

### 4. 权限错误

**原因**：文件所有者不正确

**解决**：
```bash
cd /www/wwwroot/your-domain.com
chown -R www:www .
chmod +x server
chmod 666 cboard.db
```

### 5. 服务无法启动

**检查步骤**：
```bash
# 1. 查看服务状态
systemctl status cboard

# 2. 查看日志
journalctl -u cboard -n 50

# 3. 手动测试运行
cd /www/wwwroot/your-domain.com
./server
```

## 🔄 更新项目

如果需要更新项目代码：

```bash
# 1. 停止服务
systemctl stop cboard

# 2. 更新代码（如果是 Git 仓库）
git pull

# 3. 重新运行安装脚本
sudo ./bt-deploy.sh

# 4. 启动服务
systemctl start cboard
```

## 📞 获取帮助

如果遇到问题：

1. 查看安装日志：`/tmp/cboard_install_*.log`
2. 查看服务日志：`journalctl -u cboard -f`
3. 检查配置文件：`.env`
4. 查看部署指南：`宝塔面板部署指南.md`

## ✅ 安装检查清单

安装完成后，请确认：

- [ ] Go 语言已安装（`go version`）
- [ ] Node.js 已安装（`node -v`）
- [ ] 后端编译成功（`./server` 文件存在）
- [ ] 前端构建成功（`frontend/dist/` 目录存在）
- [ ] `.env` 文件已创建
- [ ] systemd 服务已创建（`systemctl status cboard`）
- [ ] 后端服务可以启动（`curl http://127.0.0.1:8000/health`）
- [ ] Nginx 配置已添加
- [ ] SSL 证书已配置（推荐）
- [ ] 管理员账户已创建

## 🎉 完成！

如果所有检查项都通过，您的网站应该可以正常访问了！

访问地址：
- HTTP: `http://your-domain.com`
- HTTPS: `https://your-domain.com`（如果配置了 SSL）

---

**最后更新**: 2024-12-15

