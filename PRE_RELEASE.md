# 发布前检查清单

## 🔍 发布前必须完成的检查

### 1. 敏感信息检查 ⚠️ 重要

```bash
# 检查是否有敏感信息泄露
grep -r "password" --include="*.php" --exclude-dir=vendor --exclude-dir=storage .
grep -r "api_key" --include="*.php" --exclude-dir=vendor --exclude-dir=storage .
grep -r "secret" --include="*.php" --exclude-dir=vendor --exclude-dir=storage .
```

**必须确保：**
- ✅ 没有硬编码的密码
- ✅ 没有硬编码的 API 密钥
- ✅ 没有真实的数据库连接信息
- ✅ 所有配置都使用环境变量或配置文件

### 2. 配置文件检查

```bash
# 确认敏感配置文件不在仓库中
git ls-files | grep -E "(\.config\.php|appprofile\.php|\.env)"
```

**应该返回空**，这些文件应该在 .gitignore 中。

### 3. 代码检查

```bash
# 检查 PHP 语法错误
find . -name "*.php" -not -path "./vendor/*" -exec php -l {} \;

# 检查是否有未使用的文件
# （手动检查）
```

### 4. 数据库迁移检查

```bash
# 确认迁移文件格式正确
php xcat Migration --check
```

### 5. 功能测试清单

- [ ] 节点采集功能正常
- [ ] 节点列表显示正常
- [ ] 采集配置保存正常
- [ ] 采集日志查看正常
- [ ] 定时任务执行正常
- [ ] 手动采集功能正常

## 📝 发布步骤

### 步骤 1: 最终检查

```bash
# 1. 检查 git 状态
git status

# 2. 查看所有更改
git diff

# 3. 检查是否有大文件
find . -size +10M -not -path "./.git/*" -not -path "./vendor/*"

# 4. 检查 .gitignore
cat .gitignore
```

### 步骤 2: 提交更改

```bash
# 添加所有文件
git add .

# 提交（使用有意义的提交信息）
git commit -m "feat: Add node collector feature

- Add NodeFetcher service for fetching nodes from URLs
- Add NodeParser service for parsing multiple protocols
- Add NodeCollectorService for managing collection process
- Add database migration for collector fields
- Add admin interface for collector configuration
- Add scheduled collection task in Cron
- Support protocols: vmess, ss, trojan, vless, ssr, hysteria2, tuic"

# 推送到 GitHub
git push origin main
```

### 步骤 3: 创建 Release（可选）

1. 在 GitHub 上创建新的 Release
2. 填写版本号（如 v1.0.0）
3. 填写发布说明
4. 添加标签

```bash
# 创建标签
git tag -a v1.0.0 -m "Release v1.0.0: Node Collector Feature"
git push origin v1.0.0
```

## 🚨 常见问题

### Q: 如果误提交了敏感信息怎么办？

A: 立即处理：
1. 更改所有泄露的密码/密钥
2. 使用 `git filter-branch` 或 `git filter-repo` 从历史中删除
3. 强制推送（警告：这会重写历史）

### Q: 如何确保配置文件不被提交？

A: 检查 .gitignore：
```bash
# 应该包含：
config/.config.php
config/.config.php.bak
config/appprofile.php
.env
```

### Q: 发布后如何更新？

A: 使用标准的 git 工作流：
```bash
git pull origin main
composer install --no-dev --optimize-autoloader
php xcat Migration
```

## ✅ 发布检查表

在点击"发布"按钮前，确认：

- [ ] 所有敏感信息已移除
- [ ] 配置文件示例已更新
- [ ] 代码语法正确
- [ ] 功能测试通过
- [ ] 文档完整
- [ ] .gitignore 正确
- [ ] README 已更新
- [ ] 提交信息清晰
- [ ] 已创建备份

---

**完成所有检查后即可安全发布！** 🚀
