#!/bin/bash

# 数据库检查和修复脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}错误: 请使用 root 用户运行此脚本${NC}"
    exit 1
fi

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}数据库检查和修复${NC}"
echo -e "${BLUE}========================================${NC}"

# 查找项目目录
PROJECT_DIR=""
for dir in "/var/www/sspanel" "/www/wwwroot/board.moneyfly.club"; do
    if [ -d "$dir" ] && [ -f "$dir/config/.config.php" ]; then
        PROJECT_DIR="$dir"
        break
    fi
done

if [ -z "$PROJECT_DIR" ]; then
    echo -e "${RED}错误: 未找到项目目录${NC}"
    exit 1
fi

echo -e "${GREEN}项目目录: $PROJECT_DIR${NC}"
cd "$PROJECT_DIR"

# 检查配置文件
if [ ! -f "config/.config.php" ]; then
    echo -e "${RED}错误: config/.config.php 不存在${NC}"
    exit 1
fi

echo -e "${YELLOW}步骤 1: 检查数据库配置...${NC}"

# 读取数据库配置
DB_HOST=$(grep -E "^\s*['\"]db_host['\"]" config/.config.php | head -1 | sed "s/.*=>\s*['\"]\(.*\)['\"].*/\1/" | tr -d "',\"")
DB_NAME=$(grep -E "^\s*['\"]db_database['\"]" config/.config.php | head -1 | sed "s/.*=>\s*['\"]\(.*\)['\"].*/\1/" | tr -d "',\"")
DB_USER=$(grep -E "^\s*['\"]db_username['\"]" config/.config.php | head -1 | sed "s/.*=>\s*['\"]\(.*\)['\"].*/\1/" | tr -d "',\"")
DB_PASS=$(grep -E "^\s*['\"]db_password['\"]" config/.config.php | head -1 | sed "s/.*=>\s*['\"]\(.*\)['\"].*/\1/" | tr -d "',\"")

if [ -z "$DB_HOST" ]; then
    DB_HOST="localhost"
fi

echo -e "数据库主机: ${DB_HOST:-未设置}"
echo -e "数据库名: ${DB_NAME:-未设置}"
echo -e "数据库用户: ${DB_USER:-未设置}"
echo -e "数据库密码: ${DB_PASS:+已设置}"

# 测试数据库连接
echo -e "${YELLOW}步骤 2: 测试数据库连接...${NC}"

if [ -z "$DB_NAME" ] || [ -z "$DB_USER" ] || [ -z "$DB_PASS" ]; then
    echo -e "${RED}✗ 数据库配置不完整${NC}"
    echo -e "${YELLOW}请检查 config/.config.php 文件${NC}"
    exit 1
fi

# 尝试连接数据库
if command -v mysql &> /dev/null; then
    if mysql -h"$DB_HOST" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "SELECT 1;" >/dev/null 2>&1; then
        echo -e "${GREEN}✓ 数据库连接成功${NC}"
    else
        echo -e "${RED}✗ 数据库连接失败${NC}"
        echo -e "${YELLOW}请检查:${NC}"
        echo "  1. 数据库服务是否运行"
        echo "  2. 数据库用户名和密码是否正确"
        echo "  3. 数据库是否存在"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠ mysql 客户端未安装，跳过连接测试${NC}"
fi

# 检查数据库表
echo -e "${YELLOW}步骤 3: 检查数据库表...${NC}"

if command -v mysql &> /dev/null; then
    TABLE_COUNT=$(mysql -h"$DB_HOST" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "SHOW TABLES;" 2>/dev/null | wc -l)
    if [ "$TABLE_COUNT" -gt 1 ]; then
        echo -e "${GREEN}✓ 数据库表已存在 ($((TABLE_COUNT-1)) 个表)${NC}"
    else
        echo -e "${YELLOW}⚠ 数据库表可能未初始化${NC}"
        echo -e "${YELLOW}正在运行数据库迁移...${NC}"
        
        if [ -f "vendor/autoload.php" ]; then
            php xcat Migration new
            php xcat Migration latest
            php xcat Tool importSetting
            echo -e "${GREEN}✓ 数据库迁移完成${NC}"
        else
            echo -e "${RED}✗ vendor/autoload.php 不存在，请先运行 composer install${NC}"
            exit 1
        fi
    fi
fi

# 检查用户表
echo -e "${YELLOW}步骤 4: 检查用户表...${NC}"

if command -v mysql &> /dev/null; then
    USER_COUNT=$(mysql -h"$DB_HOST" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "SELECT COUNT(*) FROM user;" 2>/dev/null | tail -1)
    if [ ! -z "$USER_COUNT" ] && [ "$USER_COUNT" -gt 0 ]; then
        echo -e "${GREEN}✓ 用户表中有 $USER_COUNT 个用户${NC}"
        
        # 检查是否有管理员
        ADMIN_COUNT=$(mysql -h"$DB_HOST" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "SELECT COUNT(*) FROM user WHERE is_admin = 1;" 2>/dev/null | tail -1)
        if [ ! -z "$ADMIN_COUNT" ] && [ "$ADMIN_COUNT" -gt 0 ]; then
            echo -e "${GREEN}✓ 已有 $ADMIN_COUNT 个管理员账户${NC}"
        else
            echo -e "${YELLOW}⚠ 没有管理员账户，需要创建${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ 用户表为空或不存在${NC}"
    fi
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}数据库检查完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}如果数据库连接正常，可以运行创建管理员脚本:${NC}"
echo "  ./create_admin.sh"
