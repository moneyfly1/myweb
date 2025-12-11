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

echo -e "${YELLOW}正在创建管理员账号...${NC}"
echo -e "${YELLOW}请按照提示输入管理员信息${NC}"
echo ""

# 运行创建管理员命令
php xcat Tool createAdmin

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}管理员账号创建完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}请使用刚才创建的管理员账号登录后台${NC}"
