#!/bin/bash

# 检查 Nginx 实际生效的配置

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

DOMAIN="board.moneyfly.club"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}检查 Nginx 实际生效的配置${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

echo -e "${YELLOW}1. 检查所有包含该域名的配置文件:${NC}"
find /www/server/panel/vhost/nginx /etc/nginx -name "*.conf" 2>/dev/null | xargs grep -l "$DOMAIN" 2>/dev/null | while read config; do
    echo "   配置文件: $config"
    echo "   内容:"
    grep -A 5 "server_name.*$DOMAIN" "$config" | sed 's/^/      /'
    echo ""
done

echo -e "${YELLOW}2. 检查 Nginx 实际加载的配置:${NC}"
if command -v nginx >/dev/null 2>&1; then
    echo "   使用 nginx -T 检查实际配置..."
    nginx -T 2>/dev/null | grep -A 30 "server_name.*$DOMAIN" | head -40 | sed 's/^/   /'
else
    echo -e "${RED}   nginx 命令不可用${NC}"
fi

echo ""
echo -e "${YELLOW}3. 检查宝塔面板站点设置:${NC}"
echo "   请登录宝塔面板，检查以下设置："
echo "   1. 网站 → board.moneyfly.club → 设置"
echo "   2. 网站目录：应该设置为 /www/wwwroot/board.moneyfly.club/public"
echo "   3. 运行目录：留空或设置为 /"
echo "   4. PHP 版本：选择 PHP 8.2"
echo ""

echo -e "${YELLOW}4. 检查配置文件语法:${NC}"
if nginx -t 2>&1 | sed 's/^/   /'; then
    echo -e "${GREEN}   ✓ 配置语法正确${NC}"
else
    echo -e "${RED}   ✗ 配置语法有误${NC}"
fi
