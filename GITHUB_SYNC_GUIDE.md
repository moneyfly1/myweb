# GitHub 同步指南

## ⚠️ 重要警告

**在同步到 GitHub 之前，请务必：**
1. 备份原项目代码（如果需要）
2. 确认要删除原项目的所有代码
3. 检查是否有敏感信息泄露

## 📋 同步前检查清单

### 1. 检查敏感信息

```bash
cd /Users/apple/Downloads/SSPanel-UIM-master

# 检查是否有敏感配置文件被跟踪
git ls-files | grep -E "(\.config\.php|appprofile\.php|\.env)"

# 应该返回空，如果有文件，需要移除：
# git rm --cached config/.config.php
```

### 2. 检查代码中的硬编码信息

```bash
# 搜索可能的硬编码密码或密钥
grep -r "password.*=" --include="*.php" --exclude-dir=vendor --exclude-dir=storage . | grep -v "//" | grep -v "ChangeMe"
grep -r "api_key.*=" --include="*.php" --exclude-dir=vendor --exclude-dir=storage . | grep -v "//" | grep -v "ChangeMe"
```

### 3. 确认 .gitignore 配置正确

确认 `.gitignore` 包含：
- `config/.config.php`
- `config/appprofile.php`
- `.env`
- `vendor/`
- `storage/` 下的缓存文件

## 🚀 同步步骤

### 步骤 1: 初始化 Git 仓库（如果还没有）

```bash
cd /Users/apple/Downloads/SSPanel-UIM-master

# 如果还没有初始化
git init

# 添加远程仓库
git remote add origin https://github.com/moneyfly1/myweb.git

# 或者如果已经存在，更新远程地址
git remote set-url origin https://github.com/moneyfly1/myweb.git
```

### 步骤 2: 检查当前状态

```bash
# 查看当前状态
git status

# 查看要提交的文件
git add -n .
```

### 步骤 3: 添加文件

```bash
# 添加所有文件（.gitignore 会自动排除敏感文件）
git add .

# 再次检查要提交的文件（确认没有敏感文件）
git status
```

### 步骤 4: 提交更改

```bash
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
```

### 步骤 5: 强制推送到 GitHub（⚠️ 会删除原项目代码）

```bash
# ⚠️ 警告：这会删除原项目的所有代码！
# 请确认您真的想要这样做

# 推送到 main 分支（强制覆盖）
git branch -M main
git push -u origin main --force

# 或者推送到新分支（推荐，保留原代码）
# git push -u origin main:node-collector-feature
```

## 🔄 如果只想更新部分代码（不删除原项目）

如果您想保留原项目的部分代码，可以：

### 方法 1: 创建新分支

```bash
# 创建新分支
git checkout -b node-collector-feature

# 推送新分支
git push -u origin node-collector-feature
```

### 方法 2: 合并到现有代码

```bash
# 先拉取原项目代码
git fetch origin
git checkout main
git pull origin main

# 合并新功能
git merge node-collector-feature

# 解决冲突（如果有）
# 然后推送
git push origin main
```

## 📝 同步后操作

### 1. 在 GitHub 上检查

1. 访问 https://github.com/moneyfly1/myweb
2. 确认所有文件都已上传
3. 确认没有敏感文件泄露

### 2. 创建 Release（可选）

1. 在 GitHub 上点击 "Releases"
2. 点击 "Create a new release"
3. 填写版本号（如 `v1.0.0`）
4. 填写发布说明
5. 发布

### 3. 更新 README

确保 README.md 包含：
- 项目描述
- 安装说明
- 功能特性
- 使用文档链接

## 🔒 安全建议

1. **不要提交敏感信息**
   - 配置文件（.config.php）
   - 数据库密码
   - API 密钥

2. **使用 GitHub Secrets**
   - 如果使用 CI/CD，使用 GitHub Secrets 存储敏感信息

3. **定期检查**
   - 定期检查是否有敏感信息泄露
   - 使用 GitHub 的安全扫描功能

## ❓ 常见问题

### Q: 如何撤销强制推送？

A: 如果误操作，可以：
```bash
# 恢复之前的提交（如果有备份）
git reset --hard <previous-commit-hash>
git push origin main --force
```

### Q: 如何保留原项目的某些文件？

A: 在推送前，可以：
```bash
# 从原仓库拉取文件
git fetch origin
git checkout origin/main -- path/to/file

# 然后提交
git add path/to/file
git commit -m "Keep original file"
```

### Q: 如何确保配置文件不被提交？

A: 检查 .gitignore：
```bash
# 应该包含：
config/.config.php
config/.config.php.bak
config/appprofile.php
.env
```

---

**完成所有检查后即可安全同步！** 🚀
