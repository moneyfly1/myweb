#!/bin/bash

# Nginx 404 问题诊断脚本

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Nginx 404 问题诊断${NC}"
echo -e "${BLUE}========================================${NC}"

# 1. 检查安装目录
INSTALL_DIR=$(pwd)
echo -e "${YELLOW}1. 检查安装目录:${NC}"
echo "   安装目录: $INSTALL_DIR"
echo "   public 目录存在: $([ -d "$INSTALL_DIR/public" ] && echo "是" || echo "否")"
echo "   public/index.php 存在: $([ -f "$INSTALL_DIR/public/index.php" ] && echo "是" || echo "否")"
echo ""

# 2. 检查 Nginx 配置
echo -e "${YELLOW}2. 检查 Nginx 配置:${NC}"
NGINX_CONFIGS=(
    "/etc/nginx/sites-enabled/sspanel.conf"
    "/etc/nginx/conf.d/sspanel.conf"
    "/www/server/panel/vhost/nginx/sspanel.conf"
    "/etc/nginx/sites-available/sspanel.conf"
)

FOUND_CONFIG=""
for config in "${NGINX_CONFIGS[@]}"; do
    if [ -f "$config" ]; then
        FOUND_CONFIG="$config"
        echo "   找到配置文件: $config"
        echo "   配置内容:"
        cat "$config" | sed 's/^/   /'
        echo ""
        break
    fi
done

if [ -z "$FOUND_CONFIG" ]; then
    echo -e "${RED}   未找到 Nginx 配置文件！${NC}"
    echo "   正在搜索所有可能的配置文件..."
    find /etc/nginx -name "*.conf" -type f 2>/dev/null | grep -E "(sspanel|board\.moneyfly)" | head -5
    echo ""
fi

# 3. 检查 Nginx 错误日志
echo -e "${YELLOW}3. 检查 Nginx 错误日志（最近 20 行）:${NC}"
NGINX_ERROR_LOGS=(
    "/var/log/nginx/error.log"
    "/www/wwwlogs/nginx_error.log"
    "/www/server/nginx/logs/error.log"
)

for log in "${NGINX_ERROR_LOGS[@]}"; do
    if [ -f "$log" ]; then
        echo "   日志文件: $log"
        tail -20 "$log" | sed 's/^/   /'
        echo ""
        break
    fi
done

# 4. 检查 Nginx 访问日志
echo -e "${YELLOW}4. 检查 Nginx 访问日志（最近 10 行）:${NC}"
NGINX_ACCESS_LOGS=(
    "/var/log/nginx/access.log"
    "/www/wwwlogs/access.log"
    "/www/server/nginx/logs/access.log"
)

for log in "${NGINX_ACCESS_LOGS[@]}"; do
    if [ -f "$log" ]; then
        echo "   日志文件: $log"
        tail -10 "$log" | sed 's/^/   /'
        echo ""
        break
    fi
done

# 5. 检查 PHP-FPM
echo -e "${YELLOW}5. 检查 PHP-FPM:${NC}"
PHP_SOCKETS=(
    "/tmp/php-cgi-82.sock"
    "/tmp/php-fpm.sock"
    "/var/run/php/php8.2-fpm.sock"
    "/var/run/php-fpm/php-fpm.sock"
    "/www/server/php/82/var/run/php-fpm.sock"
)

FOUND_SOCKET=""
for socket in "${PHP_SOCKETS[@]}"; do
    if [ -S "$socket" ] || [ -f "$socket" ]; then
        FOUND_SOCKET="$socket"
        echo "   找到 PHP socket: $socket"
        break
    fi
done

if [ -z "$FOUND_SOCKET" ]; then
    echo -e "${RED}   未找到 PHP-FPM socket！${NC}"
    echo "   正在搜索..."
    find /tmp /var/run /www/server -name "*php*.sock" 2>/dev/null | head -5
fi
echo ""

# 6. 测试 Nginx 配置
echo -e "${YELLOW}6. 测试 Nginx 配置:${NC}"
if nginx -t 2>&1 | sed 's/^/   /'; then
    echo -e "${GREEN}   Nginx 配置测试通过${NC}"
else
    echo -e "${RED}   Nginx 配置测试失败${NC}"
fi
echo ""

# 7. 检查文件权限
echo -e "${YELLOW}7. 检查文件权限:${NC}"
echo "   public 目录权限: $(stat -c '%a %U:%G' "$INSTALL_DIR/public" 2>/dev/null || echo "无法获取")"
echo "   public/index.php 权限: $(stat -c '%a %U:%G' "$INSTALL_DIR/public/index.php" 2>/dev/null || echo "无法获取")"
echo ""

# 8. 检查宝塔面板站点配置
echo -e "${YELLOW}8. 检查宝塔面板配置（如果使用）:${NC}"
if [ -d "/www/server/panel" ]; then
    echo "   检测到宝塔面板"
    echo "   站点配置目录: /www/server/panel/vhost"
    BT_CONFIGS=$(ls -la /www/server/panel/vhost/nginx/*.conf 2>/dev/null | grep -E "(sspanel|board)" | awk '{print $NF}')
    for config in $BT_CONFIGS; do
        echo "   配置文件: $config"
        echo "   配置内容:"
        cat "$config" | sed 's/^/      /'
        echo ""
    done
else
    echo "   未检测到宝塔面板"
fi
echo ""

echo -e "${GREEN}诊断完成！${NC}"
echo -e "${YELLOW}请将以上输出发送给技术支持${NC}"
