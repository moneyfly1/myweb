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

# 自动检测安装目录（使用当前工作目录）
CURRENT_DIR=$(pwd)
echo -e "${GREEN}当前目录: $CURRENT_DIR${NC}"

# 如果当前目录看起来像项目目录，直接使用
if [[ "$CURRENT_DIR" == *"board.moneyfly.club"* ]] || [[ "$CURRENT_DIR" == *"wwwroot"* ]]; then
    INSTALL_DIR="$CURRENT_DIR"
    echo -e "${GREEN}检测到项目目录，将安装到: $INSTALL_DIR${NC}"
else
    # 询问安装目录
    echo -e "${YELLOW}请选择安装目录:${NC}"
    echo "1. 当前目录 ($CURRENT_DIR)"
    echo "2. /www/wwwroot/board.moneyfly.club (宝塔面板)"
    echo "3. /var/www/sspanel (标准安装)"
    read -p "请选择 (1/2/3) [默认: 1]: " DIR_CHOICE
    
    case "$DIR_CHOICE" in
        2)
            INSTALL_DIR="/www/wwwroot/board.moneyfly.club"
            ;;
        3)
            INSTALL_DIR="/var/www/sspanel"
            ;;
        *)
            INSTALL_DIR="$CURRENT_DIR"
            ;;
    esac
fi

# 确保是绝对路径
if [[ "$INSTALL_DIR" != /* ]]; then
    INSTALL_DIR="$(cd "$INSTALL_DIR" && pwd)"
fi

echo -e "${GREEN}安装目录: $INSTALL_DIR${NC}"

# 询问域名
read -p "请输入您的域名 (例如: board.moneyfly.club): " DOMAIN
if [ -z "$DOMAIN" ]; then
    if [[ "$INSTALL_DIR" == *"board.moneyfly.club"* ]]; then
        DOMAIN="board.moneyfly.club"
    else
        # 尝试从目录名提取域名
        DIR_NAME=$(basename "$INSTALL_DIR")
        if [[ "$DIR_NAME" == *"."* ]]; then
            DOMAIN="$DIR_NAME"
        else
            DOMAIN="example.com"
        fi
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
