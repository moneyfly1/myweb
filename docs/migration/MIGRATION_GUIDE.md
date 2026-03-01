# 数据库迁移指南

## 概述

本指南帮助您将新增的功能（签到系统、知识库、营销活动、用户分析）迁移到现有的数据库中。

## 新增的数据库表

### 1. checkin_records (签到记录表)
```sql
- id: 主键
- user_id: 用户ID
- amount: 奖励金额
- created_at: 签到时间
```

### 2. knowledge_categories (知识库分类表)
```sql
- id: 主键
- name: 分类名称
- icon: 图标
- sort_order: 排序
- is_active: 是否启用
- created_at, updated_at: 时间戳
```

### 3. knowledge_articles (知识库文章表)
```sql
- id: 主键
- category_id: 分类ID
- title: 标题
- content: 内容(HTML)
- summary: 摘要
- view_count: 浏览次数
- sort_order: 排序
- is_active: 是否启用
- created_at, updated_at: 时间戳
```

### 4. promotions (营销活动表)
```sql
- id: 主键
- name: 活动名称
- type: 活动类型
- discount_type: 折扣类型
- discount_value: 折扣值
- min_amount: 最低消费
- max_discount: 最高优惠
- package_ids: 适用套餐
- start_time, end_time: 活动时间
- is_active: 是否启用
- description: 活动描述
- created_at, updated_at: 时间戳
```

### 5. users 表新增字段
```sql
- telegram_id: Telegram用户ID
- telegram_username: Telegram用户名
```

---

## 迁移方法

### 方法一：使用自动迁移脚本（推荐）

#### Linux/macOS:
```bash
# 1. 进入项目目录
cd /path/to/goweb

# 2. 确保脚本有执行权限
chmod +x scripts/migrate_new_features.sh

# 3. 执行迁移（替换为你的数据库路径）
./scripts/migrate_new_features.sh /path/to/your/cboard.db

# 4. 按照提示操作
```

#### Windows:
```cmd
# 1. 进入项目目录
cd C:\path\to\goweb

# 2. 执行迁移（替换为你的数据库路径）
scripts\migrate_new_features.bat C:\path\to\your\cboard.db

# 3. 按照提示操作
```

### 方法二：手动迁移

#### 步骤1: 备份数据库
```bash
# 创建备份
cp your_database.db your_database.db.backup.$(date +%Y%m%d)
```

#### 步骤2: 创建新表
```bash
# 使用SQLite命令行工具
sqlite3 your_database.db

# 然后执行以下SQL（或直接导入SQL文件）
.read scripts/create_new_tables.sql
```

或者手动执行SQL:
```sql
-- 创建签到记录表
CREATE TABLE IF NOT EXISTS checkin_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_checkin_user_id ON checkin_records(user_id);
CREATE INDEX idx_checkin_created_at ON checkin_records(created_at);

-- 创建知识库分类表
CREATE TABLE IF NOT EXISTS knowledge_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100) NOT NULL,
    icon VARCHAR(50),
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建知识库文章表
CREATE TABLE IF NOT EXISTS knowledge_articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id INTEGER NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    summary VARCHAR(500),
    view_count INTEGER DEFAULT 0,
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (category_id) REFERENCES knowledge_categories(id)
);
CREATE INDEX idx_knowledge_category_id ON knowledge_articles(category_id);

-- 创建营销活动表
CREATE TABLE IF NOT EXISTS promotions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(30) NOT NULL,
    discount_type VARCHAR(20) NOT NULL,
    discount_value DECIMAL(10,2) NOT NULL,
    min_amount DECIMAL(10,2) DEFAULT 0,
    max_discount DECIMAL(10,2) DEFAULT 0,
    package_ids VARCHAR(500),
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    is_active BOOLEAN DEFAULT 1,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_promotion_type ON promotions(type);

-- 更新users表
ALTER TABLE users ADD COLUMN telegram_id INTEGER;
ALTER TABLE users ADD COLUMN telegram_username VARCHAR(100);
CREATE INDEX idx_users_telegram_id ON users(telegram_id);
```

#### 步骤3: 导入初始数据
```bash
sqlite3 your_database.db < scripts/init_knowledge.sql
```

#### 步骤4: 更新教程内容
```bash
sqlite3 your_database.db < scripts/update_knowledge_tutorials.sql
```

#### 步骤5: 验证迁移
```bash
sqlite3 your_database.db "SELECT name FROM sqlite_master WHERE type='table' AND (name LIKE '%knowledge%' OR name LIKE '%checkin%' OR name LIKE '%promotion%');"
```

---

## 迁移后验证

### 1. 检查表是否创建成功
```bash
sqlite3 your_database.db ".tables"
```

应该能看到:
- checkin_records
- knowledge_categories
- knowledge_articles
- promotions

### 2. 检查数据是否导入
```bash
# 检查知识库分类
sqlite3 your_database.db "SELECT COUNT(*) FROM knowledge_categories;"
# 应该返回: 5

# 检查知识库文章
sqlite3 your_database.db "SELECT COUNT(*) FROM knowledge_articles;"
# 应该返回: 14
```

### 3. 检查users表字段
```bash
sqlite3 your_database.db "PRAGMA table_info(users);" | grep telegram
```

应该能看到 telegram_id 和 telegram_username 字段。

### 4. 启动服务测试
```bash
# 启动后端
go run cmd/server/main.go

# 检查日志，确保没有数据库错误
# 访问前端页面测试新功能
```

---

## 常见问题

### Q1: 迁移脚本提示"表已存在"
**A:** 这是正常的，脚本会询问是否删除现有表。如果是首次迁移，选择 Y；如果是更新数据，选择 N。

### Q2: ALTER TABLE 失败
**A:** 可能是因为字段已存在。可以忽略这个错误，或者先检查字段是否存在：
```sql
PRAGMA table_info(users);
```

### Q3: 外键约束错误
**A:** SQLite默认不启用外键约束。如果需要启用：
```sql
PRAGMA foreign_keys = ON;
```

### Q4: 数据导入失败
**A:** 检查SQL文件编码是否为UTF-8，确保文件路径正确。

### Q5: 迁移后服务启动报错
**A:** 检查以下几点：
1. 数据库文件路径是否正确
2. 数据库文件权限是否正确
3. 查看详细错误日志
4. 如果有问题，恢复备份文件

---

## 回滚方法

如果迁移出现问题，可以恢复备份：

```bash
# Linux/macOS
cp your_database.db.backup.YYYYMMDD your_database.db

# Windows
copy your_database.db.backup.YYYYMMDD your_database.db
```

---

## 迁移检查清单

- [ ] 备份原数据库
- [ ] 创建新表结构
- [ ] 更新users表字段
- [ ] 导入知识库初始数据
- [ ] 更新客户端教程
- [ ] 验证表结构
- [ ] 验证数据完整性
- [ ] 启动服务测试
- [ ] 测试新功能
- [ ] 检查日志无错误

---

## 技术支持

如果迁移过程中遇到问题：

1. 查看迁移脚本的输出日志
2. 检查数据库备份文件是否存在
3. 查看服务器启动日志
4. 提供详细的错误信息

---

## 附录：SQL文件说明

### init_knowledge.sql
包含知识库的初始数据：
- 5个分类（新手入门、客户端教程、常见问题、进阶使用、账户相关）
- 14篇文章（涵盖各平台使用教程和常见问题）

### update_knowledge_tutorials.sql
更新客户端教程内容：
- Windows: 更新为 Clash Verge
- macOS: 更新为 ClashX Pro
- iOS: Shadowrocket（小火箭）
- Android: Clash for Android

### migrate_new_features.sh / .bat
自动化迁移脚本：
- 自动备份数据库
- 创建所有新表
- 导入初始数据
- 验证迁移结果

---

## 更新日志

### 2026-03-02
- 创建初始迁移脚本
- 添加知识库初始数据
- 更新客户端教程为Clash系列
- 添加审计日志支持
