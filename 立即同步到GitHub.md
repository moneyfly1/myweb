# 🚀 立即同步到 GitHub - 操作步骤

## ⚠️ 重要提示

**此操作会删除 GitHub 仓库 `moneyfly1/myweb` 中的所有原代码！**

如果您想保留原代码，请先：
1. 在 GitHub 上创建新分支备份
2. 或者 Fork 原仓库

## 📋 快速操作步骤

### 步骤 1: 检查敏感信息

```bash
cd /Users/apple/Downloads/SSPanel-UIM-master

# 检查是否有敏感配置文件
ls -la config/.config.php config/appprofile.php 2>/dev/null
# 如果文件存在，确保它们在 .gitignore 中
```

### 步骤 2: 运行同步脚本

```bash
cd /Users/apple/Downloads/SSPanel-UIM-master
./sync_to_github.sh
```

脚本会：
1. ✅ 检查敏感文件
2. ✅ 初始化 Git 仓库
3. ✅ 配置远程仓库
4. ✅ 添加所有文件
5. ✅ 提交更改
6. ⚠️ 强制推送到 GitHub（删除原代码）

### 步骤 3: 手动操作（如果脚本失败）

如果脚本执行失败，可以手动执行：

```bash
cd /Users/apple/Downloads/SSPanel-UIM-master

# 1. 初始化 Git
git init

# 2. 配置远程仓库
git remote add origin https://github.com/moneyfly1/myweb.git

# 3. 添加所有文件
git add .

# 4. 检查要提交的文件（确认没有敏感文件）
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

## ✅ 同步后验证

1. 访问 https://github.com/moneyfly1/myweb
2. 确认所有文件都已上传
3. 确认 README.md 显示正确
4. 确认 install.sh 文件存在

## 🎯 在 VPS 上使用

同步完成后，在 VPS 上可以这样安装：

```bash
# 下载安装脚本
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
chmod +x install.sh
sudo ./install.sh
```

或者手动安装：

```bash
# 克隆项目
git clone https://github.com/moneyfly1/myweb.git sspanel
cd sspanel

# 安装依赖
composer install --no-dev --optimize-autoloader

# 配置项目
cp config/.config.example.php config/.config.php
# 编辑 config/.config.php

# 运行迁移
php xcat Migration

# 创建管理员
php xcat Tool createAdmin
```

## 📝 安装脚本说明

`install.sh` 脚本基于 [SSPanel-UIM 官方安装指南](https://docs.sspanel.io/docs/installation/manual-install) 创建，会自动：

1. 检测操作系统（Debian/Ubuntu/CentOS）
2. 安装所有必需软件（Nginx, PHP, MariaDB, Redis）
3. 从您的 GitHub 仓库克隆项目
4. 自动配置数据库和密钥
5. 配置 Nginx
6. 初始化数据库
7. 配置定时任务
8. 配置 SSL 证书（可选）

## 🔒 安全提醒

1. **不要提交敏感信息**
   - config/.config.php（已在 .gitignore）
   - config/appprofile.php（已在 .gitignore）
   - .env 文件（已在 .gitignore）

2. **使用强密码**
   - 数据库密码
   - 管理员账户密码
   - Cookie 加密密钥

3. **定期更新**
   ```bash
   cd /var/www/sspanel
   git pull origin main
   composer install --no-dev --optimize-autoloader
   php xcat Migration
   ```

## ❓ 常见问题

### Q: 如何保留原项目代码？

A: 在推送前创建新分支：
```bash
git push -u origin main:backup-original
# 然后再推送新代码
git push -u origin main --force
```

### Q: 如何撤销强制推送？

A: 如果误操作，可以恢复（如果有备份）：
```bash
git reset --hard <previous-commit>
git push origin main --force
```

### Q: 安装脚本失败怎么办？

A: 参考 [DEPLOY.md](DEPLOY.md) 手动安装。

---

**准备好后，运行 `./sync_to_github.sh` 即可！** 🚀
