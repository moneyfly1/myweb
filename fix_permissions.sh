#!/bin/bash

# 修复文件权限脚本

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
echo -e "${BLUE}修复文件权限${NC}"
echo -e "${BLUE}========================================${NC}"

# 查找项目目录（优先使用当前目录）
PROJECT_DIR=""
CURRENT_DIR=$(pwd)

# 首先检查当前目录
if [ -d "$CURRENT_DIR" ] && [ -f "$CURRENT_DIR/config/.config.php" ]; then
    PROJECT_DIR="$CURRENT_DIR"
    echo -e "${GREEN}使用当前目录: $PROJECT_DIR${NC}"
else
    # 尝试查找常见目录
    for dir in "/var/www/sspanel" "/www/wwwroot"*; do
        if [ -d "$dir" ] && [ -f "$dir/config/.config.php" ]; then
            PROJECT_DIR="$dir"
            break
        fi
    done
    
    if [ -z "$PROJECT_DIR" ]; then
        read -p "请输入项目目录路径: " PROJECT_DIR
        if [ ! -d "$PROJECT_DIR" ]; then
            echo -e "${RED}错误: 目录不存在: $PROJECT_DIR${NC}"
            exit 1
        fi
    fi
fi

echo -e "${GREEN}项目目录: $PROJECT_DIR${NC}"
cd "$PROJECT_DIR"

# 检测 Web 用户
if id "www-data" &>/dev/null; then
    WEB_USER="www-data"
elif id "nginx" &>/dev/null; then
    WEB_USER="nginx"
elif id "apache" &>/dev/null; then
    WEB_USER="apache"
else
    WEB_USER="www-data"
fi

echo -e "${GREEN}Web 用户: $WEB_USER${NC}"

# 创建必要的目录
echo -e "${YELLOW}步骤 1: 创建必要的目录...${NC}"
mkdir -p storage/framework/smarty/cache
mkdir -p storage/framework/smarty/compile
mkdir -p storage/framework/twig/cache
mkdir -p storage/logs
mkdir -p public/clients
mkdir -p uploads/avatars
mkdir -p uploads/config

echo -e "${GREEN}✓ 目录创建完成${NC}"

# 设置文件所有者
echo -e "${YELLOW}步骤 2: 设置文件所有者...${NC}"
chown -R ${WEB_USER}:${WEB_USER} "$PROJECT_DIR"
echo -e "${GREEN}✓ 文件所有者设置完成${NC}"

# 设置目录权限
echo -e "${YELLOW}步骤 3: 设置目录权限...${NC}"
find "$PROJECT_DIR" -type d -exec chmod 755 {} \;
echo -e "${GREEN}✓ 目录权限设置完成${NC}"

# 设置文件权限
echo -e "${YELLOW}步骤 4: 设置文件权限...${NC}"
find "$PROJECT_DIR" -type f -exec chmod 644 {} \;
echo -e "${GREEN}✓ 文件权限设置完成${NC}"

# 设置特殊目录权限
echo -e "${YELLOW}步骤 5: 设置特殊目录权限...${NC}"
chmod -R 777 "$PROJECT_DIR/storage"
chmod -R 777 "$PROJECT_DIR/uploads"
chmod 775 "$PROJECT_DIR/public/clients" 2>/dev/null || true
echo -e "${GREEN}✓ 特殊目录权限设置完成${NC}"

# 设置配置文件权限
echo -e "${YELLOW}步骤 6: 设置配置文件权限...${NC}"
if [ -f "$PROJECT_DIR/config/.config.php" ]; then
    chmod 664 "$PROJECT_DIR/config/.config.php"
    echo -e "${GREEN}✓ .config.php 权限已设置${NC}"
fi
if [ -f "$PROJECT_DIR/config/appprofile.php" ]; then
    chmod 664 "$PROJECT_DIR/config/appprofile.php"
    echo -e "${GREEN}✓ appprofile.php 权限已设置${NC}"
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}权限修复完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}验证权限:${NC}"
ls -ld "$PROJECT_DIR/storage"
ls -ld "$PROJECT_DIR/uploads"
ls -ld "$PROJECT_DIR/public/clients" 2>/dev/null || echo "public/clients 目录不存在（可选）"
