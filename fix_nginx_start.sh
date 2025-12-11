#!/bin/bash

# 修复 Nginx 启动失败和 PHP 版本切换问题

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}修复 Nginx 启动失败和 PHP 版本问题${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 步骤 1: 检查 Nginx 配置语法
echo -e "${YELLOW}步骤 1: 检查 Nginx 配置语法...${NC}"
if nginx -t 2>&1; then
    echo -e "${GREEN}✓ Nginx 配置语法正确${NC}"
else
    echo -e "${RED}✗ Nginx 配置语法有误${NC}"
    echo -e "${YELLOW}请查看上面的错误信息并修复${NC}"
    echo ""
fi

# 步骤 2: 检查 Nginx 进程
echo -e "${YELLOW}步骤 2: 检查 Nginx 进程...${NC}"
if pgrep -x nginx > /dev/null; then
    echo -e "${GREEN}✓ Nginx 正在运行${NC}"
    echo "   进程信息:"
    ps aux | grep nginx | grep -v grep | sed 's/^/   /'
else
    echo -e "${RED}✗ Nginx 未运行${NC}"
fi
echo ""

# 步骤 3: 检查端口占用
echo -e "${YELLOW}步骤 3: 检查端口占用...${NC}"
if netstat -tuln | grep -q ":80 "; then
    echo -e "${YELLOW}端口 80 已被占用:${NC}"
    netstat -tuln | grep ":80 " | sed 's/^/   /'
    PID=$(netstat -tuln | grep ":80 " | awk '{print $7}' | cut -d'/' -f1 | head -1)
    if [ ! -z "$PID" ]; then
        echo -e "${YELLOW}   占用进程: $(ps -p $PID -o comm= 2>/dev/null || echo '未知') (PID: $PID)${NC}"
    fi
else
    echo -e "${GREEN}✓ 端口 80 未被占用${NC}"
fi
echo ""

# 步骤 4: 检查 Nginx 错误日志
echo -e "${YELLOW}步骤 4: 检查 Nginx 错误日志（最近 20 行）...${NC}"
NGINX_ERROR_LOGS=(
    "/var/log/nginx/error.log"
    "/www/server/nginx/logs/error.log"
    "/www/wwwlogs/board.moneyfly.club.error.log"
)

for log in "${NGINX_ERROR_LOGS[@]}"; do
    if [ -f "$log" ]; then
        echo "   日志文件: $log"
        tail -20 "$log" | sed 's/^/   /'
        echo ""
        break
    fi
done

# 步骤 5: 检查 PHP-FPM 状态
echo -e "${YELLOW}步骤 5: 检查 PHP-FPM 状态...${NC}"
PHP_VERSIONS=(82 83 84 80 81)
FOUND_PHP=0

for ver in "${PHP_VERSIONS[@]}"; do
    # 检查宝塔面板的 PHP-FPM
    if systemctl status php-fpm-${ver} >/dev/null 2>&1 || [ -f "/www/server/php/${ver}/sbin/php-fpm" ]; then
        echo "   检测到 PHP ${ver}"
        FOUND_PHP=1
        
        # 检查 socket
        SOCKET="/tmp/php-cgi-${ver}.sock"
        if [ -S "$SOCKET" ] || [ -f "$SOCKET" ]; then
            echo -e "${GREEN}   ✓ Socket 存在: $SOCKET${NC}"
        else
            echo -e "${YELLOW}   ⚠ Socket 不存在: $SOCKET${NC}"
        fi
        
        # 检查服务状态
        if systemctl is-active --quiet php-fpm-${ver} 2>/dev/null; then
            echo -e "${GREEN}   ✓ PHP-FPM ${ver} 服务正在运行${NC}"
        elif [ -f "/www/server/php/${ver}/sbin/php-fpm" ]; then
            echo -e "${YELLOW}   ⚠ PHP-FPM ${ver} 服务可能未通过 systemctl 管理${NC}"
        fi
    fi
done

if [ $FOUND_PHP -eq 0 ]; then
    echo -e "${RED}   ✗ 未找到 PHP-FPM 服务${NC}"
fi
echo ""

# 步骤 6: 检查宝塔面板配置文件
echo -e "${YELLOW}步骤 6: 检查宝塔面板配置文件...${NC}"
DOMAIN="board.moneyfly.club"
BT_CONFIG="/www/server/panel/vhost/nginx/${DOMAIN}.conf"

if [ -f "$BT_CONFIG" ]; then
    echo "   配置文件: $BT_CONFIG"
    
    # 检查 root 路径
    ROOT_PATH=$(grep -E "^\s*root\s+" "$BT_CONFIG" | head -1 | awk '{print $2}' | tr -d ';')
    echo "   Root 路径: $ROOT_PATH"
    
    if [ ! -d "$ROOT_PATH" ]; then
        echo -e "${RED}   ✗ Root 目录不存在！${NC}"
    else
        echo -e "${GREEN}   ✓ Root 目录存在${NC}"
    fi
    
    # 检查 PHP socket 配置
    PHP_SOCKET_CONFIG=$(grep -E "fastcgi_pass\s+unix:" "$BT_CONFIG" | head -1 | awk '{print $2}' | tr -d ';')
    PHP_SOCKET_PATH=$(echo "$PHP_SOCKET_CONFIG" | sed 's|unix:||')
    echo "   PHP Socket 配置: $PHP_SOCKET_CONFIG"
    echo "   PHP Socket 路径: $PHP_SOCKET_PATH"
    
    if [ ! -z "$PHP_SOCKET_PATH" ]; then
        if [ -S "$PHP_SOCKET_PATH" ] || [ -f "$PHP_SOCKET_PATH" ]; then
            echo -e "${GREEN}   ✓ PHP Socket 存在${NC}"
        else
            echo -e "${RED}   ✗ PHP Socket 不存在！${NC}"
            echo -e "${YELLOW}   这可能是导致网站无法访问的原因${NC}"
            echo -e "${YELLOW}   需要启动 PHP-FPM 服务${NC}"
        fi
    fi
else
    echo -e "${RED}   ✗ 配置文件不存在: $BT_CONFIG${NC}"
fi
echo ""

# 步骤 7: 提供修复建议
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}修复建议${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

echo -e "${YELLOW}如果 Nginx 启动失败，请尝试：${NC}"
echo "1. 检查配置文件语法: nginx -t"
echo "2. 查看详细错误: tail -50 /var/log/nginx/error.log"
echo "3. 如果端口被占用，停止占用进程或修改 Nginx 监听端口"
echo "4. 如果 PHP Socket 不存在，启动对应的 PHP-FPM 服务"
echo ""

echo -e "${YELLOW}如果 PHP 版本无法切换，请尝试：${NC}"
echo "1. 在宝塔面板中：网站 → board.moneyfly.club → 设置 → PHP"
echo "2. 确保已安装对应的 PHP 版本"
echo "3. 重启 PHP-FPM 服务: systemctl restart php-fpm-82"
echo "4. 如果宝塔面板无法切换，手动编辑配置文件:"
echo "   nano /www/server/panel/vhost/nginx/board.moneyfly.club.conf"
echo "   修改 fastcgi_pass 中的 socket 路径"
echo ""

echo -e "${YELLOW}快速修复命令：${NC}"
echo "# 启动 PHP-FPM 8.2"
echo "systemctl start php-fpm-82"
echo "# 或"
echo "/www/server/php/82/sbin/php-fpm start"
echo ""
echo "# 测试 Nginx 配置"
echo "nginx -t"
echo ""
echo "# 启动 Nginx"
echo "systemctl start nginx"
echo "# 或"
echo "/etc/init.d/nginx start"
echo ""
