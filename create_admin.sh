#!/bin/bash

# 创建管理员账号脚本

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
echo -e "${BLUE}创建管理员账号${NC}"
echo -e "${BLUE}========================================${NC}"

# 检查项目目录
if [ ! -d "/var/www/sspanel" ]; then
    echo -e "${RED}错误: 项目目录 /var/www/sspanel 不存在${NC}"
    exit 1
fi

cd /var/www/sspanel

# 检查 vendor 目录
if [ ! -f "vendor/autoload.php" ]; then
    echo -e "${RED}错误: vendor/autoload.php 不存在，请先运行 composer install${NC}"
    exit 1
fi

# 检查配置文件
if [ ! -f "config/.config.php" ]; then
    echo -e "${RED}错误: config/.config.php 不存在，请先配置项目${NC}"
    exit 1
fi

# 检查数据库连接
echo -e "${YELLOW}正在检查数据库连接...${NC}"

# 读取数据库配置
DB_HOST=$(grep -E "^\s*['\"]db_host['\"]" config/.config.php | head -1 | sed "s/.*=>\s*['\"]\(.*\)['\"].*/\1/" | tr -d "',\"" || echo "localhost")
DB_NAME=$(grep -E "^\s*['\"]db_database['\"]" config/.config.php | head -1 | sed "s/.*=>\s*['\"]\(.*\)['\"].*/\1/" | tr -d "',\"")
DB_USER=$(grep -E "^\s*['\"]db_username['\"]" config/.config.php | head -1 | sed "s/.*=>\s*['\"]\(.*\)['\"].*/\1/" | tr -d "',\"")
DB_PASS=$(grep -E "^\s*['\"]db_password['\"]" config/.config.php | head -1 | sed "s/.*=>\s*['\"]\(.*\)['\"].*/\1/" | tr -d "',\"")

if [ -z "$DB_NAME" ] || [ -z "$DB_USER" ] || [ -z "$DB_PASS" ]; then
    echo -e "${RED}错误: 数据库配置不完整${NC}"
    echo -e "${YELLOW}请检查 config/.config.php 文件${NC}"
    exit 1
fi

# 测试数据库连接
if command -v mysql &> /dev/null; then
    if ! mysql -h"$DB_HOST" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "SELECT 1;" >/dev/null 2>&1; then
        echo -e "${RED}错误: 无法连接到数据库${NC}"
        echo -e "${YELLOW}请检查:${NC}"
        echo "  1. 数据库服务是否运行: systemctl status mariadb"
        echo "  2. 数据库配置是否正确"
        echo "  3. 数据库是否存在"
        exit 1
    fi
    echo -e "${GREEN}✓ 数据库连接正常${NC}"
fi

# 检查是否需要运行数据库迁移
echo -e "${YELLOW}正在检查数据库表...${NC}"
if command -v mysql &> /dev/null; then
    TABLE_COUNT=$(mysql -h"$DB_HOST" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "SHOW TABLES;" 2>/dev/null | wc -l)
    if [ "$TABLE_COUNT" -le 1 ]; then
        echo -e "${YELLOW}数据库表未初始化，正在运行迁移...${NC}"
        php xcat Migration new
        php xcat Migration latest
        php xcat Tool importSetting
        echo -e "${GREEN}✓ 数据库迁移完成${NC}"
    fi
fi

echo ""
echo -e "${YELLOW}正在创建管理员账号...${NC}"
echo -e "${YELLOW}请按照提示输入管理员信息${NC}"
echo ""

# 运行创建管理员命令
php xcat Tool createAdmin

# 检查是否创建成功
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 管理员账号创建成功${NC}"
else
    echo -e "${RED}✗ 管理员账号创建失败${NC}"
    echo -e "${YELLOW}可能的原因:${NC}"
    echo "  1. 数据库连接失败"
    echo "  2. 数据库表未初始化"
    echo "  3. 配置文件错误"
    echo ""
    echo -e "${YELLOW}请运行数据库检查脚本:${NC}"
    echo "  ./check_database.sh"
    exit 1
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}管理员账号创建完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}请使用刚才创建的管理员账号登录后台${NC}"
