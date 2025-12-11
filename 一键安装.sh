#!/bin/bash

# SSPanel-UIM 一键安装脚本（简化版）
# 适用于 VPS 快速安装

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}"
echo "=========================================="
echo "SSPanel-UIM 节点采集版 - 一键安装"
echo "=========================================="
echo -e "${NC}"

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}错误: 请使用 root 用户运行此脚本${NC}"
    exit 1
fi

# 询问安装目录
echo -e "${YELLOW}请选择安装目录:${NC}"
echo "1. /www/wwwroot/board.moneyfly.club (宝塔面板推荐)"
echo "2. /var/www/sspanel (标准安装)"
read -p "请选择 (1/2) [默认: 1]: " DIR_CHOICE

if [ "$DIR_CHOICE" == "2" ]; then
    INSTALL_DIR="/var/www/sspanel"
else
    INSTALL_DIR="/www/wwwroot/board.moneyfly.club"
fi

echo -e "${GREEN}安装目录: $INSTALL_DIR${NC}"

# 询问域名
read -p "请输入您的域名 (例如: board.moneyfly.club): " DOMAIN
if [ -z "$DOMAIN" ]; then
    if [[ "$INSTALL_DIR" == *"board.moneyfly.club"* ]]; then
        DOMAIN="board.moneyfly.club"
    else
        DOMAIN="example.com"
    fi
fi

echo -e "${GREEN}域名: $DOMAIN${NC}"

# 下载安装脚本
echo -e "${YELLOW}正在下载安装脚本...${NC}"
cd /tmp
wget -q https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh -O install.sh
chmod +x install.sh

# 运行安装脚本（传递参数）
export INSTALL_DIR="$INSTALL_DIR"
export DOMAIN="$DOMAIN"
bash install.sh

echo ""
echo -e "${GREEN}=========================================="
echo -e "安装完成！"
echo -e "==========================================${NC}"
echo ""
echo -e "访问地址: https://${DOMAIN}"
echo -e "管理后台: https://${DOMAIN}/auth/login"
echo ""
