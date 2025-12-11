# GitHub 发布检查清单

在将项目发布到 GitHub 之前，请完成以下检查：

## ✅ 安全检查

### 1. 敏感信息检查
- [ ] 确认 `config/.config.php` 不在仓库中（已在 .gitignore）
- [ ] 确认 `config/appprofile.php` 不在仓库中（已在 .gitignore）
- [ ] 确认 `.env` 文件不在仓库中（已在 .gitignore）
- [ ] 检查代码中是否有硬编码的密码、密钥、API Token
- [ ] 检查是否有数据库连接字符串暴露
- [ ] 检查是否有 Redis 密码暴露

### 2. 配置文件检查
- [ ] `config/.config.example.php` 中的示例值都是安全的（如 'ChangeMe'）
- [ ] 所有敏感配置项都有明确的注释说明需要修改

## 📝 文档检查

### 1. README.md
- [ ] 更新了项目描述
- [ ] 添加了节点采集功能的说明
- [ ] 更新了特性列表
- [ ] 检查了所有链接是否有效

### 2. 部署文档
- [ ] `DEPLOY.md` 存在且内容完整
- [ ] `节点采集功能部署说明.md` 存在（可选，如果不想提交可以删除）
- [ ] 所有文档中的示例域名都已替换为占位符

### 3. 代码文档
- [ ] 核心功能有适当的注释
- [ ] 复杂逻辑有说明注释

## 🔧 代码检查

### 1. 代码质量
- [ ] 所有 PHP 文件符合 PSR-12 代码规范
- [ ] 没有语法错误
- [ ] 没有未使用的变量或函数
- [ ] 错误处理完善

### 2. 功能完整性
- [ ] 所有新增功能都已实现
- [ ] 数据库迁移文件正确
- [ ] 路由配置完整
- [ ] 前端页面功能正常

### 3. 兼容性
- [ ] 与现有功能兼容
- [ ] 不影响原有功能
- [ ] 向后兼容（如果有数据库变更）

## 🗂️ 文件结构检查

### 1. .gitignore
- [ ] 已更新，包含所有敏感文件
- [ ] 包含临时文件和缓存目录
- [ ] 包含 IDE 配置文件

### 2. 必要文件
- [ ] `LICENSE` 文件存在
- [ ] `CONTRIBUTING.md` 存在（如果适用）
- [ ] `SECURITY.md` 存在（如果适用）

## 🚀 发布准备

### 1. 版本信息
- [ ] 确定版本号（如 v1.0.0）
- [ ] 更新 CHANGELOG（如果有）
- [ ] 准备发布说明

### 2. 测试
- [ ] 在本地环境测试所有功能
- [ ] 测试数据库迁移
- [ ] 测试节点采集功能
- [ ] 测试定时任务

### 3. 提交前检查
```bash
# 检查是否有未提交的更改
git status

# 检查是否有敏感信息
git diff

# 检查文件大小（避免提交大文件）
find . -size +10M -not -path "./.git/*"
```

## 📦 发布步骤

### 1. 创建 GitHub 仓库
```bash
# 在 GitHub 上创建新仓库
# 然后执行：
git init
git add .
git commit -m "Initial commit: Add node collector feature"
git branch -M main
git remote add origin https://github.com/your-username/SSPanel-UIM.git
git push -u origin main
```

### 2. 创建 Release
- 在 GitHub 上创建新的 Release
- 填写版本号和发布说明
- 上传必要的文件（如果有）

### 3. 添加标签
```bash
git tag -a v1.0.0 -m "Release version 1.0.0 with node collector feature"
git push origin v1.0.0
```

## 🔐 安全建议

### 1. 仓库设置
- [ ] 设置为私有仓库（如果不想公开）
- [ ] 启用分支保护
- [ ] 设置协作者权限

### 2. 密钥管理
- [ ] 使用 GitHub Secrets 存储敏感信息（CI/CD）
- [ ] 不要在代码中硬编码密钥
- [ ] 使用环境变量管理配置

### 3. 依赖安全
- [ ] 定期更新依赖包
- [ ] 检查已知安全漏洞
- [ ] 使用 `composer audit` 检查

## 📋 发布后检查

### 1. 功能验证
- [ ] 在 VPS 上测试部署
- [ ] 验证所有功能正常
- [ ] 检查日志是否有错误

### 2. 文档更新
- [ ] 更新项目 Wiki（如果有）
- [ ] 更新相关文档链接
- [ ] 回复用户问题

## 🎯 可选操作

### 1. 添加徽章
在 README.md 中添加项目徽章：
```markdown
![License](https://img.shields.io/badge/license-MIT-blue.svg)
![PHP Version](https://img.shields.io/badge/PHP-8.2+-777BB4.svg)
```

### 2. 添加 GitHub Actions
- CI/CD 自动化测试
- 自动代码检查
- 自动部署（可选）

### 3. 添加 Issue 模板
- Bug 报告模板
- 功能请求模板

---

**完成所有检查后，即可安全发布到 GitHub！** ✅
