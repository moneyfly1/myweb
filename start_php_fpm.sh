#!/bin/bash

# 启动 PHP-FPM 服务脚本

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}启动 PHP-FPM 服务${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

PHP_VERSION="82"
PHP_SOCKET="/tmp/php-cgi-${PHP_VERSION}.sock"

# 方法 1: 使用 systemctl
echo -e "${YELLOW}方法 1: 尝试使用 systemctl 启动...${NC}"
if systemctl start php-fpm-${PHP_VERSION} 2>/dev/null; then
    echo -e "${GREEN}✓ 使用 systemctl 启动成功${NC}"
    systemctl status php-fpm-${PHP_VERSION} --no-pager | head -10
    exit 0
fi

# 方法 2: 使用宝塔面板的 PHP-FPM
echo -e "${YELLOW}方法 2: 尝试使用宝塔面板 PHP-FPM 启动...${NC}"
PHP_FPM_BIN="/www/server/php/${PHP_VERSION}/sbin/php-fpm"

if [ -f "$PHP_FPM_BIN" ]; then
    echo -e "${GREEN}找到 PHP-FPM: $PHP_FPM_BIN${NC}"
    
    # 检查配置文件
    PHP_FPM_CONF="/www/server/php/${PHP_VERSION}/etc/php-fpm.conf"
    if [ -f "$PHP_FPM_CONF" ]; then
        echo -e "${GREEN}找到配置文件: $PHP_FPM_CONF${NC}"
        
        # 检查是否已在运行
        if pgrep -f "$PHP_FPM_BIN" > /dev/null; then
            echo -e "${YELLOW}PHP-FPM 已在运行${NC}"
        else
            # 启动 PHP-FPM
            echo -e "${YELLOW}正在启动 PHP-FPM...${NC}"
            $PHP_FPM_BIN start 2>&1
            
            if [ $? -eq 0 ]; then
                echo -e "${GREEN}✓ PHP-FPM 启动成功${NC}"
            else
                echo -e "${RED}✗ PHP-FPM 启动失败${NC}"
                echo -e "${YELLOW}尝试查看错误信息...${NC}"
                $PHP_FPM_BIN -t 2>&1
            fi
        fi
    else
        echo -e "${RED}✗ 配置文件不存在: $PHP_FPM_CONF${NC}"
    fi
else
    echo -e "${RED}✗ PHP-FPM 可执行文件不存在: $PHP_FPM_BIN${NC}"
fi

# 检查 socket 是否创建
echo ""
echo -e "${YELLOW}检查 PHP Socket...${NC}"
sleep 2

if [ -S "$PHP_SOCKET" ] || [ -f "$PHP_SOCKET" ]; then
    echo -e "${GREEN}✓ PHP Socket 已创建: $PHP_SOCKET${NC}"
    ls -la "$PHP_SOCKET"
else
    echo -e "${RED}✗ PHP Socket 未创建: $PHP_SOCKET${NC}"
    echo -e "${YELLOW}请检查 PHP-FPM 配置中的 listen 设置${NC}"
    
    # 检查配置文件中的 listen 设置
    if [ -f "$PHP_FPM_CONF" ]; then
        echo -e "${YELLOW}配置文件中的 listen 设置:${NC}"
        grep -E "^\s*listen\s*=" "$PHP_FPM_CONF" | head -3 | sed 's/^/   /'
    fi
fi

# 检查进程
echo ""
echo -e "${YELLOW}检查 PHP-FPM 进程...${NC}"
PHP_PROCESSES=$(pgrep -f "php-fpm.*${PHP_VERSION}" | wc -l)
if [ "$PHP_PROCESSES" -gt 0 ]; then
    echo -e "${GREEN}✓ 找到 $PHP_PROCESSES 个 PHP-FPM 进程${NC}"
    ps aux | grep "php-fpm.*${PHP_VERSION}" | grep -v grep | head -5 | sed 's/^/   /'
else
    echo -e "${RED}✗ 未找到 PHP-FPM 进程${NC}"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}完成${NC}"
echo -e "${BLUE}========================================${NC}"
