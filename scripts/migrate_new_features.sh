#!/bin/bash
# 数据库迁移脚本 - 将新增功能迁移到现有数据库
# 使用方法: ./migrate_new_features.sh your_database.db

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查参数
if [ $# -eq 0 ]; then
    echo -e "${RED}错误: 请提供数据库文件路径${NC}"
    echo "使用方法: $0 <数据库文件路径>"
    echo "示例: $0 /path/to/your/cboard.db"
    exit 1
fi

DB_FILE="$1"

# 检查数据库文件是否存在
if [ ! -f "$DB_FILE" ]; then
    echo -e "${RED}错误: 数据库文件不存在: $DB_FILE${NC}"
    exit 1
fi

echo -e "${GREEN}=== 开始迁移新增功能到数据库 ===${NC}"
echo "目标数据库: $DB_FILE"
echo ""

# 备份数据库
BACKUP_FILE="${DB_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
echo -e "${YELLOW}1. 备份数据库...${NC}"
cp "$DB_FILE" "$BACKUP_FILE"
echo -e "${GREEN}   ✓ 备份完成: $BACKUP_FILE${NC}"
echo ""

# 检查表是否已存在
echo -e "${YELLOW}2. 检查现有表结构...${NC}"
EXISTING_TABLES=$(sqlite3 "$DB_FILE" "SELECT name FROM sqlite_master WHERE type='table' AND (name='checkin_records' OR name='knowledge_categories' OR name='knowledge_articles' OR name='promotions');")

if [ ! -z "$EXISTING_TABLES" ]; then
    echo -e "${YELLOW}   警告: 以下表已存在:${NC}"
    echo "$EXISTING_TABLES"
    echo ""
    read -p "是否要删除现有表并重新创建? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}   删除现有表...${NC}"
        sqlite3 "$DB_FILE" "DROP TABLE IF EXISTS checkin_records;"
        sqlite3 "$DB_FILE" "DROP TABLE IF EXISTS knowledge_articles;"
        sqlite3 "$DB_FILE" "DROP TABLE IF EXISTS knowledge_categories;"
        sqlite3 "$DB_FILE" "DROP TABLE IF EXISTS promotions;"
        echo -e "${GREEN}   ✓ 已删除现有表${NC}"
    else
        echo -e "${YELLOW}   跳过表创建步骤${NC}"
    fi
fi
echo ""

# 创建新表
echo -e "${YELLOW}3. 创建新表结构...${NC}"

# 创建签到记录表
sqlite3 "$DB_FILE" <<EOF
CREATE TABLE IF NOT EXISTS checkin_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_checkin_user_id ON checkin_records(user_id);
CREATE INDEX IF NOT EXISTS idx_checkin_created_at ON checkin_records(created_at);
EOF
echo -e "${GREEN}   ✓ checkin_records 表创建完成${NC}"

# 创建知识库分类表
sqlite3 "$DB_FILE" <<EOF
CREATE TABLE IF NOT EXISTS knowledge_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100) NOT NULL,
    icon VARCHAR(50),
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
EOF
echo -e "${GREEN}   ✓ knowledge_categories 表创建完成${NC}"

# 创建知识库文章表
sqlite3 "$DB_FILE" <<EOF
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
CREATE INDEX IF NOT EXISTS idx_knowledge_category_id ON knowledge_articles(category_id);
EOF
echo -e "${GREEN}   ✓ knowledge_articles 表创建完成${NC}"

# 创建营销活动表
sqlite3 "$DB_FILE" <<EOF
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
CREATE INDEX IF NOT EXISTS idx_promotion_type ON promotions(type);
EOF
echo -e "${GREEN}   ✓ promotions 表创建完成${NC}"
echo ""

# 检查users表是否需要添加telegram字段
echo -e "${YELLOW}4. 检查并更新 users 表...${NC}"
TELEGRAM_ID_EXISTS=$(sqlite3 "$DB_FILE" "SELECT COUNT(*) FROM pragma_table_info('users') WHERE name='telegram_id';")
if [ "$TELEGRAM_ID_EXISTS" = "0" ]; then
    echo -e "${YELLOW}   添加 telegram_id 字段...${NC}"
    sqlite3 "$DB_FILE" "ALTER TABLE users ADD COLUMN telegram_id INTEGER;"
    sqlite3 "$DB_FILE" "CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id);"
    echo -e "${GREEN}   ✓ telegram_id 字段添加完成${NC}"
else
    echo -e "${GREEN}   ✓ telegram_id 字段已存在${NC}"
fi

TELEGRAM_USERNAME_EXISTS=$(sqlite3 "$DB_FILE" "SELECT COUNT(*) FROM pragma_table_info('users') WHERE name='telegram_username';")
if [ "$TELEGRAM_USERNAME_EXISTS" = "0" ]; then
    echo -e "${YELLOW}   添加 telegram_username 字段...${NC}"
    sqlite3 "$DB_FILE" "ALTER TABLE users ADD COLUMN telegram_username VARCHAR(100);"
    echo -e "${GREEN}   ✓ telegram_username 字段添加完成${NC}"
else
    echo -e "${GREEN}   ✓ telegram_username 字段已存在${NC}"
fi
echo ""

# 导入初始数据
echo -e "${YELLOW}5. 导入知识库初始数据...${NC}"
read -p "是否要导入知识库初始数据? (Y/n): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Nn]$ ]]; then
    # 检查init_knowledge.sql是否存在
    if [ -f "scripts/init_knowledge.sql" ]; then
        sqlite3 "$DB_FILE" < scripts/init_knowledge.sql
        echo -e "${GREEN}   ✓ 知识库初始数据导入完成${NC}"
    else
        echo -e "${YELLOW}   警告: scripts/init_knowledge.sql 文件不存在，跳过数据导入${NC}"
    fi
else
    echo -e "${YELLOW}   跳过数据导入${NC}"
fi
echo ""

# 更新教程内容
echo -e "${YELLOW}6. 更新客户端教程...${NC}"
read -p "是否要更新客户端教程为Clash系列? (Y/n): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Nn]$ ]]; then
    if [ -f "scripts/update_knowledge_tutorials.sql" ]; then
        sqlite3 "$DB_FILE" < scripts/update_knowledge_tutorials.sql
        echo -e "${GREEN}   ✓ 教程更新完成${NC}"
    else
        echo -e "${YELLOW}   警告: scripts/update_knowledge_tutorials.sql 文件不存在，跳过教程更新${NC}"
    fi
else
    echo -e "${YELLOW}   跳过教程更新${NC}"
fi
echo ""

# 验证迁移结果
echo -e "${YELLOW}7. 验证迁移结果...${NC}"
echo "表统计:"
echo "  - checkin_records: $(sqlite3 "$DB_FILE" "SELECT COUNT(*) FROM checkin_records;") 条记录"
echo "  - knowledge_categories: $(sqlite3 "$DB_FILE" "SELECT COUNT(*) FROM knowledge_categories;") 条记录"
echo "  - knowledge_articles: $(sqlite3 "$DB_FILE" "SELECT COUNT(*) FROM knowledge_articles;") 条记录"
echo "  - promotions: $(sqlite3 "$DB_FILE" "SELECT COUNT(*) FROM promotions;") 条记录"
echo ""

# 完成
echo -e "${GREEN}=== 迁移完成 ===${NC}"
echo ""
echo "备份文件: $BACKUP_FILE"
echo "如果迁移出现问题，可以使用以下命令恢复:"
echo "  cp $BACKUP_FILE $DB_FILE"
echo ""
echo -e "${GREEN}下一步:${NC}"
echo "1. 重启后端服务: go run cmd/server/main.go"
echo "2. 访问前端页面测试新功能"
echo "3. 检查日志确认无错误"
echo ""
