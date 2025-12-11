#!/bin/bash

# 诊断 Nginx 无法重启的问题

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}诊断 Nginx 无法重启问题${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 步骤 1: 检查 Nginx 配置语法
echo -e "${YELLOW}步骤 1: 检查 Nginx 配置语法...${NC}"
if nginx -t 2>&1; then
    echo -e "${GREEN}✓ Nginx 配置语法正确${NC}"
else
    echo -e "${RED}✗ Nginx 配置语法有误！${NC}"
    echo -e "${YELLOW}   这是导致无法重启的主要原因${NC}"
    echo ""
    echo -e "${YELLOW}详细错误信息：${NC}"
    nginx -t 2>&1 | sed 's/^/   /'
    exit 1
fi
echo ""

# 步骤 2: 检查 Nginx 进程
echo -e "${YELLOW}步骤 2: 检查 Nginx 进程...${NC}"
NGINX_PIDS=$(pgrep -x nginx)
if [ ! -z "$NGINX_PIDS" ]; then
    echo -e "${YELLOW}发现 Nginx 进程：${NC}"
    ps aux | grep nginx | grep -v grep | sed 's/^/   /'
    echo ""
    echo -e "${YELLOW}尝试停止 Nginx...${NC}"
    systemctl stop nginx 2>/dev/null || /etc/init.d/nginx stop 2>/dev/null || killall nginx 2>/dev/null
    sleep 2
else
    echo -e "${GREEN}✓ 没有运行中的 Nginx 进程${NC}"
fi
echo ""

# 步骤 3: 检查端口占用
echo -e "${YELLOW}步骤 3: 检查端口占用...${NC}"
PORTS=(80 443)
for port in "${PORTS[@]}"; do
    OCCUPIED=$(netstat -tuln | grep ":${port} " || lsof -i :${port} 2>/dev/null)
    if [ ! -z "$OCCUPIED" ]; then
        echo -e "${YELLOW}端口 ${port} 被占用：${NC}"
        echo "$OCCUPIED" | sed 's/^/   /'
        PID=$(echo "$OCCUPIED" | awk '{print $NF}' | grep -E "^[0-9]+" | head -1 | cut -d'/' -f1)
        if [ ! -z "$PID" ]; then
            PROCESS=$(ps -p $PID -o comm= 2>/dev/null || echo "未知")
            echo -e "${YELLOW}   占用进程: $PROCESS (PID: $PID)${NC}"
        fi
    else
        echo -e "${GREEN}✓ 端口 ${port} 未被占用${NC}"
    fi
done
echo ""

# 步骤 4: 检查 Nginx 错误日志
echo -e "${YELLOW}步骤 4: 检查 Nginx 错误日志（最近 30 行）...${NC}"
ERROR_LOGS=(
    "/var/log/nginx/error.log"
    "/www/server/nginx/logs/error.log"
)

for log in "${ERROR_LOGS[@]}"; do
    if [ -f "$log" ]; then
        echo "   日志文件: $log"
        tail -30 "$log" | sed 's/^/   /'
        echo ""
        break
    fi
done

# 步骤 5: 检查配置文件
echo -e "${YELLOW}步骤 5: 检查配置文件...${NC}"
CONFIG_FILES=(
    "/etc/nginx/nginx.conf"
    "/www/server/nginx/conf/nginx.conf"
)

for config in "${CONFIG_FILES[@]}"; do
    if [ -f "$config" ]; then
        echo "   配置文件: $config"
        
        # 检查 include 指令
        INCLUDES=$(grep -E "^\s*include\s+" "$config" | grep -v "#")
        if [ ! -z "$INCLUDES" ]; then
            echo -e "${YELLOW}   包含的配置文件：${NC}"
            echo "$INCLUDES" | sed 's/^/      /'
            
            # 检查是否有不存在的配置文件
            while IFS= read -r include_line; do
                include_path=$(echo "$include_line" | awk '{print $2}' | tr -d ';')
                # 处理通配符
                if [[ "$include_path" == *"*"* ]]; then
                    # 通配符路径，检查目录是否存在
                    dir_path=$(dirname "$include_path")
                    if [ ! -d "$dir_path" ]; then
                        echo -e "${RED}      ✗ 目录不存在: $dir_path${NC}"
                    fi
                else
                    # 普通路径
                    if [ ! -f "$include_path" ] && [ ! -d "$include_path" ]; then
                        echo -e "${RED}      ✗ 文件不存在: $include_path${NC}"
                    fi
                fi
            done <<< "$INCLUDES"
        fi
    fi
done
echo ""

# 步骤 6: 检查宝塔面板站点配置
echo -e "${YELLOW}步骤 6: 检查宝塔面板站点配置...${NC}"
BT_CONFIG_DIR="/www/server/panel/vhost/nginx"
if [ -d "$BT_CONFIG_DIR" ]; then
    CONFIG_COUNT=$(find "$BT_CONFIG_DIR" -name "*.conf" -type f 2>/dev/null | wc -l)
    echo "   配置文件数量: $CONFIG_COUNT"
    
    if [ "$CONFIG_COUNT" -gt 0 ]; then
        echo -e "${YELLOW}   检查配置文件语法...${NC}"
        for config in "$BT_CONFIG_DIR"/*.conf; do
            if [ -f "$config" ]; then
                # 检查是否有语法错误
                if ! nginx -t -c "$config" >/dev/null 2>&1; then
                    echo -e "${RED}   ✗ 配置文件有语法错误: $(basename $config)${NC}"
                fi
            fi
        done
    fi
else
    echo -e "${YELLOW}   ⚠ 宝塔面板配置目录不存在${NC}"
fi
echo ""

# 步骤 7: 尝试启动 Nginx
echo -e "${YELLOW}步骤 7: 尝试启动 Nginx...${NC}"
echo -e "${YELLOW}   使用 systemctl...${NC}"
if systemctl start nginx 2>&1; then
    sleep 2
    if systemctl is-active --quiet nginx; then
        echo -e "${GREEN}   ✓ Nginx 启动成功${NC}"
    else
        echo -e "${RED}   ✗ Nginx 启动失败${NC}"
        systemctl status nginx --no-pager | head -20 | sed 's/^/      /'
    fi
else
    echo -e "${YELLOW}   systemctl 启动失败，尝试其他方法...${NC}"
    
    # 尝试直接启动
    echo -e "${YELLOW}   直接启动 nginx...${NC}"
    if nginx 2>&1; then
        sleep 2
        if pgrep -x nginx > /dev/null; then
            echo -e "${GREEN}   ✓ Nginx 启动成功${NC}"
        else
            echo -e "${RED}   ✗ Nginx 启动失败${NC}"
        fi
    else
        echo -e "${RED}   ✗ Nginx 启动失败${NC}"
        echo -e "${YELLOW}   错误信息：${NC}"
        nginx 2>&1 | sed 's/^/      /'
    fi
fi
echo ""

# 步骤 8: 提供修复建议
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}修复建议${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

if ! nginx -t >/dev/null 2>&1; then
    echo -e "${RED}1. 修复配置文件语法错误（最重要）${NC}"
    echo "   nginx -t  # 查看详细错误"
    echo ""
fi

if [ ! -z "$NGINX_PIDS" ]; then
    echo -e "${YELLOW}2. 强制停止所有 Nginx 进程${NC}"
    echo "   killall -9 nginx"
    echo "   systemctl stop nginx"
    echo ""
fi

echo -e "${YELLOW}3. 检查并修复配置文件${NC}"
echo "   - 检查所有 include 的配置文件是否存在"
echo "   - 检查配置文件语法"
echo "   - 检查是否有残留的无效配置"
echo ""

echo -e "${YELLOW}4. 清理并重新启动${NC}"
echo "   # 停止 Nginx"
echo "   systemctl stop nginx"
echo "   killall -9 nginx 2>/dev/null"
echo ""
echo "   # 测试配置"
echo "   nginx -t"
echo ""
echo "   # 启动 Nginx"
echo "   systemctl start nginx"
echo ""
