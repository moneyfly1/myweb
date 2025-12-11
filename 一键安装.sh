#!/bin/bash

# SSPanel-UIM 一键安装脚本（简化版）
# 适用于 VPS 快速安装
# 所有代码将下载到当前目录

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

# 使用当前工作目录作为安装目录
CURRENT_DIR=$(pwd)
echo -e "${GREEN}当前目录: $CURRENT_DIR${NC}"
echo -e "${GREEN}所有代码将下载到此目录${NC}"

# 下载安装脚本
echo -e "${YELLOW}正在下载安装脚本...${NC}"
cd "$CURRENT_DIR"
wget -q https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh -O install.sh
chmod +x install.sh

# 运行安装脚本
bash install.sh
