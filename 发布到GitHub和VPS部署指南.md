# 发布到 GitHub 和 VPS 部署完整指南

## 📋 概述

本文档列出了将项目发布到 GitHub 并在 VPS 上部署所需的所有更改和步骤。

## ✅ 已完成的准备工作

### 1. 文件更新
- ✅ 更新了 `.gitignore` - 确保敏感文件不会被提交
- ✅ 更新了 `README.md` - 添加了节点采集功能说明
- ✅ 创建了 `DEPLOY.md` - 详细的 VPS 部署指南
- ✅ 创建了 `deploy.sh` - 自动化部署脚本
- ✅ 创建了 `GITHUB_RELEASE_CHECKLIST.md` - GitHub 发布检查清单
- ✅ 创建了 `PRE_RELEASE.md` - 发布前检查清单

### 2. 代码完整性
- ✅ 所有核心功能已实现
- ✅ 数据库迁移文件已创建
- ✅ 路由配置完整
- ✅ 前端页面完整

## 🔧 发布到 GitHub 前需要做的更改

### 1. 检查敏感信息（⚠️ 最重要）

```bash
# 在项目根目录执行
cd /Users/apple/Downloads/SSPanel-UIM-master

# 检查是否有敏感配置文件被跟踪
git ls-files | grep -E "(\.config\.php|appprofile\.php|\.env)"

# 应该返回空，如果有文件，需要从 git 中移除：
# git rm --cached config/.config.php
```

### 2. 检查代码中的硬编码信息

```bash
# 搜索可能的硬编码密码或密钥
grep -r "password.*=" --include="*.php" --exclude-dir=vendor --exclude-dir=storage .
grep -r "api_key.*=" --include="*.php" --exclude-dir=vendor --exclude-dir=storage .
```

### 3. 确认 .gitignore 配置

确认 `.gitignore` 包含以下内容（已更新）：
- `config/.config.php`
- `config/appprofile.php`
- `.env`
- `vendor/`
- `storage/` 下的缓存文件

### 4. 可选：决定是否提交文档文件

以下文档文件可以选择性提交或删除：
- `节点采集改造大纲.md` - 开发文档（可选）
- `实施确认.md` - 开发文档（可选）
- `实施完成总结.md` - 开发文档（可选）

如果不想提交，可以在 `.gitignore` 中添加：
```
# 节点采集功能相关文档
节点采集改造大纲.md
实施确认.md
实施完成总结.md
```

## 🚀 发布到 GitHub 的步骤

### 步骤 1: 初始化 Git 仓库（如果还没有）

```bash
cd /Users/apple/Downloads/SSPanel-UIM-master

# 如果还没有初始化
git init

# 添加远程仓库（替换为您的仓库地址）
git remote add origin https://github.com/your-username/SSPanel-UIM.git
```

### 步骤 2: 添加文件并提交

```bash
# 添加所有文件
git add .

# 检查要提交的文件（确认没有敏感文件）
git status

# 提交
git commit -m "feat: Add node collector feature

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
- Add scheduled collection in Cron"
```

### 步骤 3: 推送到 GitHub

```bash
# 推送到主分支
git branch -M main
git push -u origin main

# 或者推送到其他分支
git push -u origin main:dev
```

### 步骤 4: 创建 Release（可选）

1. 在 GitHub 网页上进入仓库
2. 点击 "Releases" → "Create a new release"
3. 填写版本号（如 `v1.0.0`）
4. 填写发布说明
5. 发布

## 🖥️ VPS 部署步骤

### 方法 1: 使用自动化脚本（推荐）

```bash
# 1. 上传 deploy.sh 到 VPS
scp deploy.sh root@your-vps-ip:/root/

# 2. SSH 登录 VPS
ssh root@your-vps-ip

# 3. 运行部署脚本
chmod +x deploy.sh
./deploy.sh

# 4. 按照脚本提示完成配置
```

### 方法 2: 手动部署（参考 DEPLOY.md）

详细步骤请查看 `DEPLOY.md` 文件，包括：
1. 安装基础环境（PHP, MySQL, Redis, Nginx）
2. 配置数据库
3. 克隆项目
4. 安装依赖
5. 配置项目
6. 配置 Nginx
7. 配置 SSL
8. 配置定时任务

### 快速部署命令

```bash
# 在 VPS 上执行
cd /var/www
git clone https://github.com/your-username/SSPanel-UIM.git sspanel
cd sspanel
composer install --no-dev --optimize-autoloader
cp config/.config.example.php config/.config.php
# 编辑 config/.config.php
nano config/.config.php
# 运行迁移
php xcat Migration
# 创建管理员
php xcat User createAdmin
```

## 📝 部署后配置

### 1. 配置节点采集功能

1. 登录管理后台：`https://your-domain.com/auth/login`
2. 进入 **节点管理** → **节点采集**
3. 添加采集源 URL
4. 配置采集间隔
5. 启用定时采集（可选）
6. 点击 **立即采集** 测试

### 2. 配置定时任务

```bash
# 编辑 crontab
crontab -e

# 添加以下行（每 5 分钟执行一次）
*/5 * * * * cd /var/www/sspanel && /usr/bin/php xcat Cron >> /dev/null 2>&1
```

### 3. 配置 Nginx（如果还没配置）

参考 `DEPLOY.md` 中的 Nginx 配置示例。

### 4. 配置 SSL 证书

```bash
# 安装 Certbot
apt install -y certbot python3-certbot-nginx

# 获取证书
certbot --nginx -d your-domain.com
```

## 🔒 安全建议

### 1. 服务器安全
- 使用强密码
- 配置防火墙
- 定期更新系统
- 使用 SSH 密钥认证

### 2. 应用安全
- 修改所有默认密钥
- 使用 HTTPS
- 定期备份数据库
- 监控日志

### 3. 数据库安全
- 使用强密码
- 限制数据库用户权限
- 定期备份

## 📚 相关文档

- **部署指南**: `DEPLOY.md`
- **节点采集功能说明**: `节点采集功能部署说明.md`
- **发布检查清单**: `GITHUB_RELEASE_CHECKLIST.md`
- **发布前检查**: `PRE_RELEASE.md`

## ❓ 常见问题

### Q: 如何更新项目？

A: 
```bash
cd /var/www/sspanel
git pull origin main
composer install --no-dev --optimize-autoloader
php xcat Migration
```

### Q: 如何备份？

A:
```bash
# 备份数据库
mysqldump -u sspanel -p sspanel > backup_$(date +%Y%m%d).sql

# 备份配置文件
tar -czf config_backup_$(date +%Y%m%d).tar.gz config/
```

### Q: 如何回滚？

A:
```bash
cd /var/www/sspanel
git log  # 查看提交历史
git checkout <commit-hash>  # 回滚到指定提交
php xcat Migration  # 运行迁移
```

## ✅ 检查清单

发布前确认：
- [ ] 所有敏感信息已移除
- [ ] .gitignore 配置正确
- [ ] 代码语法正确
- [ ] 功能测试通过
- [ ] 文档完整
- [ ] README 已更新

部署后确认：
- [ ] 网站可以访问
- [ ] 管理后台可以登录
- [ ] 节点采集功能正常
- [ ] 定时任务正常运行
- [ ] SSL 证书配置正确

---

**祝发布和部署顺利！** 🚀
