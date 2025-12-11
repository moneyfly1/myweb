#!/bin/bash

# 修复宝塔面板 Nginx 显示失败和网站显示默认页面的问题

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}修复宝塔面板 Nginx 和网站配置${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

DOMAIN="board.moneyfly.club"
INSTALL_DIR="/www/wwwroot/board.moneyfly.club"
BT_CONFIG="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
NGINX_MAIN_CONF="/etc/nginx/nginx.conf"

# 步骤 1: 检查 Nginx 实际状态
echo -e "${YELLOW}步骤 1: 检查 Nginx 实际状态...${NC}"
if pgrep -x nginx > /dev/null; then
    echo -e "${GREEN}✓ Nginx 进程正在运行${NC}"
    ps aux | grep nginx | grep -v grep | head -3 | sed 's/^/   /'
else
    echo -e "${RED}✗ Nginx 进程未运行${NC}"
    echo -e "${YELLOW}正在启动 Nginx...${NC}"
    systemctl start nginx || /etc/init.d/nginx start
    sleep 2
fi
echo ""

# 步骤 2: 检查站点配置文件
echo -e "${YELLOW}步骤 2: 检查站点配置文件...${NC}"
if [ -f "$BT_CONFIG" ]; then
    echo -e "${GREEN}✓ 站点配置文件存在: $BT_CONFIG${NC}"
    
    # 检查 root 路径
    ROOT_PATH=$(grep -E "^\s*root\s+" "$BT_CONFIG" | head -1 | awk '{print $2}' | tr -d ';' | sed 's|/$||')
    echo "   Root 路径: $ROOT_PATH"
    
    if [ "$ROOT_PATH" != "${INSTALL_DIR}/public" ]; then
        echo -e "${RED}✗ Root 路径不正确！${NC}"
        echo -e "${YELLOW}   期望: ${INSTALL_DIR}/public${NC}"
        echo -e "${YELLOW}   实际: $ROOT_PATH${NC}"
    else
        echo -e "${GREEN}   ✓ Root 路径正确${NC}"
    fi
    
    # 检查 server_name
    SERVER_NAME=$(grep -E "^\s*server_name\s+" "$BT_CONFIG" | head -1 | awk '{print $2}' | tr -d ';')
    echo "   Server Name: $SERVER_NAME"
    
    # 检查 PHP 配置
    PHP_CONFIG=$(grep -E "fastcgi_pass" "$BT_CONFIG" | head -1)
    if [ ! -z "$PHP_CONFIG" ]; then
        echo -e "${GREEN}   ✓ PHP 配置存在${NC}"
        echo "   $PHP_CONFIG" | sed 's/^/   /'
    else
        echo -e "${RED}   ✗ PHP 配置不存在！${NC}"
    fi
else
    echo -e "${RED}✗ 站点配置文件不存在: $BT_CONFIG${NC}"
fi
echo ""

# 步骤 3: 检查 Nginx 主配置是否包含站点配置
echo -e "${YELLOW}步骤 3: 检查 Nginx 主配置...${NC}"
if [ -f "$NGINX_MAIN_CONF" ]; then
    # 检查是否包含宝塔面板配置目录
    if grep -q "/www/server/panel/vhost/nginx" "$NGINX_MAIN_CONF"; then
        echo -e "${GREEN}✓ Nginx 主配置包含宝塔面板配置目录${NC}"
        grep "/www/server/panel/vhost/nginx" "$NGINX_MAIN_CONF" | sed 's/^/   /'
    else
        echo -e "${YELLOW}⚠ Nginx 主配置可能未包含宝塔面板配置目录${NC}"
        echo -e "${YELLOW}   检查 include 指令...${NC}"
        grep -E "include\s+" "$NGINX_MAIN_CONF" | grep -v "#" | sed 's/^/   /'
    fi
else
    echo -e "${RED}✗ Nginx 主配置文件不存在: $NGINX_MAIN_CONF${NC}"
fi
echo ""

# 步骤 4: 检查实际加载的配置
echo -e "${YELLOW}步骤 4: 检查 Nginx 实际加载的配置...${NC}"
if command -v nginx >/dev/null 2>&1; then
    echo "   使用 nginx -T 检查实际配置..."
    ACTIVE_CONFIG=$(nginx -T 2>/dev/null | grep -A 20 "server_name.*${DOMAIN}" | head -25)
    if [ ! -z "$ACTIVE_CONFIG" ]; then
        echo -e "${GREEN}   ✓ 找到站点配置${NC}"
        echo "$ACTIVE_CONFIG" | grep -E "(server_name|root|fastcgi_pass)" | sed 's/^/      /'
    else
        echo -e "${RED}   ✗ 未找到站点配置！${NC}"
        echo -e "${YELLOW}   这说明 Nginx 没有加载站点配置文件${NC}"
        echo -e "${YELLOW}   可能正在使用默认配置${NC}"
    fi
else
    echo -e "${RED}   ✗ nginx 命令不可用${NC}"
fi
echo ""

# 步骤 5: 检查默认站点配置
echo -e "${YELLOW}步骤 5: 检查默认站点配置...${NC}"
DEFAULT_CONFIGS=(
    "/etc/nginx/sites-enabled/default"
    "/etc/nginx/conf.d/default.conf"
    "/www/server/nginx/conf/nginx.conf"
)

for config in "${DEFAULT_CONFIGS[@]}"; do
    if [ -f "$config" ]; then
        echo "   找到配置文件: $config"
        if grep -q "Welcome to nginx" "$config" 2>/dev/null || grep -q "default_server" "$config" 2>/dev/null; then
            echo -e "${YELLOW}   ⚠ 这可能是默认配置，可能覆盖站点配置${NC}"
        fi
    fi
done
echo ""

# 步骤 6: 修复建议
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}修复建议${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

echo -e "${YELLOW}问题分析：${NC}"
echo "1. 网站显示 Nginx 默认页面，说明 Nginx 在运行但未加载站点配置"
echo "2. 宝塔面板显示 Nginx 失败，可能是状态检测问题"
echo "3. PHP 版本显示为'静态'，需要改为 PHP 8.2"
echo ""

echo -e "${YELLOW}修复步骤：${NC}"
echo ""
echo -e "${GREEN}步骤 1: 在宝塔面板中启动 Nginx${NC}"
echo "   - 点击 Nginx 旁边的'启动'按钮"
echo "   - 或运行: systemctl start nginx"
echo ""
echo -e "${GREEN}步骤 2: 修改 PHP 版本（重要！）${NC}"
echo "   - 网站 → board.moneyfly.club → 设置 → PHP"
echo "   - 将 PHP 版本从'静态'改为'PHP 8.2'"
echo "   - 保存"
echo ""
echo -e "${GREEN}步骤 3: 检查站点配置是否正确加载${NC}"
echo "   - 在宝塔面板中：网站 → board.moneyfly.club → 设置 → 配置文件"
echo "   - 确认 root 路径为: /www/wwwroot/board.moneyfly.club/public"
echo "   - 确认 fastcgi_pass 指向: unix:/tmp/php-cgi-82.sock"
echo ""
echo -e "${GREEN}步骤 4: 重载 Nginx${NC}"
echo "   - 在宝塔面板中点击 Nginx 的'重载'按钮"
echo "   - 或运行: systemctl reload nginx"
echo ""

echo -e "${YELLOW}如果仍然显示默认页面，请检查：${NC}"
echo "1. Nginx 主配置文件是否包含宝塔面板配置目录"
echo "2. 是否有其他默认站点配置覆盖了站点配置"
echo "3. 站点配置文件的权限是否正确"
echo ""
