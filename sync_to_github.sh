#!/bin/bash

# 同步项目到 GitHub 仓库脚本
# ⚠️ 警告：此脚本会强制推送，删除原项目的所有代码！

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${RED}========================================${NC}"
echo -e "${RED}⚠️  警告：此操作会删除 GitHub 上的所有原代码！${NC}"
echo -e "${RED}========================================${NC}"
echo ""
read -p "您确定要继续吗？输入 'YES' 确认: " CONFIRM

if [ "$CONFIRM" != "YES" ]; then
    echo -e "${YELLOW}操作已取消${NC}"
    exit 0
fi

# 项目目录
PROJECT_DIR="/Users/apple/Downloads/SSPanel-UIM-master"
cd "$PROJECT_DIR"

# 检查敏感文件
echo -e "${YELLOW}检查敏感文件...${NC}"
SENSITIVE_FILES=$(git ls-files 2>/dev/null | grep -E "(\.config\.php|appprofile\.php|\.env)" || true)
if [ ! -z "$SENSITIVE_FILES" ]; then
    echo -e "${RED}发现敏感文件，请先处理:${NC}"
    echo "$SENSITIVE_FILES"
    exit 1
fi

# 初始化 Git（如果还没有）
if [ ! -d ".git" ]; then
    echo -e "${YELLOW}初始化 Git 仓库...${NC}"
    git init
fi

# 配置远程仓库
echo -e "${YELLOW}配置远程仓库...${NC}"
git remote remove origin 2>/dev/null || true
git remote add origin https://github.com/moneyfly1/myweb.git

# 添加所有文件
echo -e "${YELLOW}添加文件...${NC}"
git add .

# 检查要提交的文件
echo -e "${YELLOW}要提交的文件:${NC}"
git status --short

# 提交
echo -e "${YELLOW}提交更改...${NC}"
git commit -m "feat: Add node collector feature to SSPanel-UIM

Features:
- Automatic node collection from external URLs
- Support multiple protocols (vmess, ss, trojan, vless, ssr, hysteria2, tuic)
- Admin interface for collector configuration
- Scheduled collection tasks
- Collection logs and status monitoring

Changes:
- Add NodeCollector service classes (NodeFetcher, NodeParser, NodeCollectorService)
- Add database migration for collector fields
- Extend NodeController with collector methods
- Add collector configuration page (collector.tpl)
- Update node list to show source (index.tpl)
- Add scheduled collection in Cron
- Add installation script (install.sh)
- Add deployment documentation (DEPLOY.md, GITHUB_SYNC_GUIDE.md)

Based on SSPanel-UIM with node collector enhancement."

# 强制推送（⚠️ 会删除原代码）
echo -e "${RED}准备强制推送到 GitHub...${NC}"
read -p "最后确认：这将删除原项目的所有代码！继续? (y/N): " FINAL_CONFIRM

if [[ "$FINAL_CONFIRM" == "y" || "$FINAL_CONFIRM" == "Y" ]]; then
    git branch -M main
    git push -u origin main --force
    echo -e "${GREEN}✅ 代码已同步到 GitHub！${NC}"
    echo -e "${GREEN}仓库地址: https://github.com/moneyfly1/myweb${NC}"
else
    echo -e "${YELLOW}操作已取消${NC}"
    exit 0
fi
