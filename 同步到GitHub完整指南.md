# 同步到 GitHub 完整指南

## 📋 已完成的工作

### 1. 创建的文件
- ✅ `install.sh` - 一键安装脚本（基于官方安装指南）
- ✅ `sync_to_github.sh` - GitHub 同步脚本
- ✅ `INSTALL.md` - 安装指南
- ✅ `DEPLOY.md` - VPS 部署指南
- ✅ `GITHUB_SYNC_GUIDE.md` - GitHub 同步指南
- ✅ 所有节点采集功能代码

### 2. 修改的文件
- ✅ `README.md` - 更新为节点采集版说明
- ✅ `.gitignore` - 确保敏感文件不被提交

## 🚀 快速同步步骤

### 方法 1: 使用同步脚本（推荐）

```bash
cd /Users/apple/Downloads/SSPanel-UIM-master
./sync_to_github.sh
```

脚本会：
1. 检查敏感文件
2. 初始化 Git（如果需要）
3. 配置远程仓库
4. 添加所有文件
5. 提交更改
6. 强制推送到 GitHub（会删除原代码）

### 方法 2: 手动同步

```bash
cd /Users/apple/Downloads/SSPanel-UIM-master

# 1. 检查敏感文件
git ls-files | grep -E "(\.config\.php|appprofile\.php|\.env)"
# 应该返回空

# 2. 初始化 Git（如果还没有）
git init
git remote add origin https://github.com/moneyfly1/myweb.git

# 3. 添加文件
git add .

# 4. 检查要提交的文件
git status

# 5. 提交
git commit -m "feat: Add node collector feature to SSPanel-UIM

Features:
- Automatic node collection from external URLs
- Support multiple protocols (vmess, ss, trojan, vless, ssr, hysteria2, tuic)
- Admin interface for collector configuration
- Scheduled collection tasks
- Collection logs and status monitoring

Changes:
- Add NodeCollector service classes
- Add database migration for collector fields
- Extend NodeController with collector methods
- Add collector configuration page
- Update node list to show source
- Add scheduled collection in Cron
- Add installation script (install.sh)
- Add deployment documentation"

# 6. 强制推送（⚠️ 会删除原代码）
git branch -M main
git push -u origin main --force
```

## ⚠️ 重要注意事项

### 1. 删除原项目代码

**强制推送会删除 GitHub 上的所有原代码！**

如果您想保留原项目的部分代码：
- 使用新分支：`git push -u origin main:node-collector-feature`
- 或者先备份原项目

### 2. 敏感信息检查

在推送前，务必检查：

```bash
# 检查敏感配置文件
git ls-files | grep -E "(\.config\.php|appprofile\.php|\.env)"

# 检查代码中的硬编码密码
grep -r "password.*=" --include="*.php" --exclude-dir=vendor . | grep -v "ChangeMe" | grep -v "//"
```

### 3. .gitignore 确认

确认 `.gitignore` 包含：
```
config/.config.php
config/appprofile.php
.env
vendor/
storage/framework/smarty/cache/*
storage/framework/smarty/compile/*
```

## 📝 安装脚本说明

### install.sh 功能

基于 [SSPanel-UIM 官方安装指南](https://docs.sspanel.io/docs/installation/manual-install) 创建，自动完成：

1. **系统检测** - 自动检测 Debian/Ubuntu/CentOS
2. **安装基础环境** - Nginx, PHP 8.2+, MariaDB, Redis
3. **安装 Composer** - PHP 依赖管理
4. **部署项目** - 从 GitHub 克隆项目
5. **配置项目** - 自动生成密钥和配置
6. **设置权限** - 正确的文件权限
7. **配置 Nginx** - Web 服务器配置
8. **初始化数据库** - 运行迁移和创建管理员
9. **配置定时任务** - Cron 任务设置
10. **配置 SSL** - Let's Encrypt 证书（可选）

### 使用方法

在 VPS 上执行：

```bash
# 下载脚本
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
chmod +x install.sh
sudo ./install.sh
```

## 🔧 安装脚本的修改点

相比官方安装指南，我们的脚本：

1. **自动检测系统** - 支持 Debian/Ubuntu/CentOS
2. **自动生成密码** - 数据库密码、密钥等
3. **自动配置** - 自动修改 .config.php
4. **从您的仓库克隆** - 使用 `moneyfly1/myweb`
5. **包含节点采集功能** - 自动运行数据库迁移

## 📦 项目结构

同步后的项目结构：

```
myweb/
├── install.sh                    # 一键安装脚本
├── sync_to_github.sh            # GitHub 同步脚本
├── README.md                     # 项目说明（已更新）
├── INSTALL.md                    # 安装指南
├── DEPLOY.md                     # 部署指南
├── GITHUB_SYNC_GUIDE.md         # 同步指南
├── 节点采集功能部署说明.md        # 功能说明
├── src/
│   └── Services/
│       └── NodeCollector/       # 节点采集服务
├── db/
│   └── migrations/
│       └── 2025010100-add_node_collector.php
├── resources/
│   └── views/
│       └── tabler/
│           └── admin/
│               └── node/
│                   ├── collector.tpl  # 采集配置页面
│                   └── index.tpl      # 节点列表（已修改）
└── ... (其他 SSPanel-UIM 文件)
```

## ✅ 同步前检查清单

- [ ] 检查敏感文件（应该为空）
- [ ] 检查硬编码密码（应该没有）
- [ ] 确认 .gitignore 正确
- [ ] 测试 install.sh 脚本语法
- [ ] 确认所有文档完整
- [ ] 备份原项目（如果需要）

## 🎯 同步后操作

### 1. 验证同步

访问 https://github.com/moneyfly1/myweb 确认：
- 所有文件都已上传
- 没有敏感文件泄露
- README 显示正确

### 2. 测试安装脚本

在测试 VPS 上测试安装脚本：

```bash
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
chmod +x install.sh
sudo ./install.sh
```

### 3. 创建 Release（可选）

1. 在 GitHub 上创建 Release
2. 版本号：v1.0.0
3. 添加发布说明

## 🔄 后续更新

### 更新代码

```bash
cd /var/www/sspanel
git pull origin main
composer install --no-dev --optimize-autoloader
php xcat Migration
```

### 更新安装脚本

如果修改了 install.sh，同步到 GitHub：

```bash
git add install.sh
git commit -m "update: Update installation script"
git push origin main
```

## 📚 相关文档

- [INSTALL.md](INSTALL.md) - 安装指南
- [DEPLOY.md](DEPLOY.md) - 详细部署指南
- [节点采集功能部署说明.md](节点采集功能部署说明.md) - 功能说明
- [GITHUB_SYNC_GUIDE.md](GITHUB_SYNC_GUIDE.md) - 同步指南

## ⚠️ 警告

1. **强制推送会删除原代码** - 请确认您真的想要这样做
2. **敏感信息不要提交** - 检查所有配置文件
3. **备份重要数据** - 在操作前备份

---

**完成检查后，运行 `./sync_to_github.sh` 即可同步！** 🚀
