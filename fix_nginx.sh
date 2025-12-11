#!/bin/bash

# Nginx 修复脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}错误: 请使用 root 用户运行此脚本${NC}"
    exit 1
fi

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Nginx 修复脚本${NC}"
echo -e "${BLUE}========================================${NC}"

# 检查 Nginx 是否安装
if ! command -v nginx &> /dev/null; then
    echo -e "${RED}错误: Nginx 未安装${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Nginx 已安装${NC}"

# 检查 Nginx 配置文件
echo -e "${YELLOW}检查 Nginx 配置...${NC}"
if nginx -t 2>&1 | grep -q "successful"; then
    echo -e "${GREEN}✓ Nginx 配置测试通过${NC}"
else
    echo -e "${RED}✗ Nginx 配置测试失败${NC}"
    nginx -t
    exit 1
fi

# 查找 Nginx PID 文件位置
NGINX_PID_FILE=""
if [ -f "/var/run/nginx.pid" ]; then
    NGINX_PID_FILE="/var/run/nginx.pid"
elif [ -f "/run/nginx.pid" ]; then
    NGINX_PID_FILE="/run/nginx.pid"
elif [ -f "/usr/local/nginx/logs/nginx.pid" ]; then
    NGINX_PID_FILE="/usr/local/nginx/logs/nginx.pid"
fi

# 检查 Nginx 主配置文件中的 PID 设置
NGINX_CONF="/etc/nginx/nginx.conf"
if [ -f "$NGINX_CONF" ]; then
    PID_FROM_CONF=$(grep -E "^\s*pid\s+" "$NGINX_CONF" | awk '{print $2}' | tr -d ';' | head -1)
    if [ ! -z "$PID_FROM_CONF" ]; then
        NGINX_PID_FILE="$PID_FROM_CONF"
    fi
fi

echo -e "${GREEN}Nginx PID 文件: ${NGINX_PID_FILE:-未找到}${NC}"

# 检查 Nginx 进程是否运行
if pgrep -x nginx > /dev/null; then
    echo -e "${GREEN}✓ Nginx 进程正在运行${NC}"
    
    # 如果 PID 文件不存在或为空，创建它
    if [ -z "$NGINX_PID_FILE" ] || [ ! -f "$NGINX_PID_FILE" ] || [ ! -s "$NGINX_PID_FILE" ]; then
        echo -e "${YELLOW}PID 文件不存在或为空，正在创建...${NC}"
        
        # 获取主进程 PID
        MASTER_PID=$(pgrep -x nginx | head -1)
        
        if [ ! -z "$MASTER_PID" ]; then
            # 确定 PID 文件位置
            if [ -z "$NGINX_PID_FILE" ]; then
                NGINX_PID_FILE="/var/run/nginx.pid"
            fi
            
            # 确保目录存在
            PID_DIR=$(dirname "$NGINX_PID_FILE")
            mkdir -p "$PID_DIR"
            
            # 写入 PID
            echo "$MASTER_PID" > "$NGINX_PID_FILE"
            echo -e "${GREEN}✓ 已创建 PID 文件: $NGINX_PID_FILE (PID: $MASTER_PID)${NC}"
        fi
    fi
else
    echo -e "${YELLOW}⚠ Nginx 进程未运行，正在启动...${NC}"
    
    # 确保 PID 文件目录存在
    if [ ! -z "$NGINX_PID_FILE" ]; then
        PID_DIR=$(dirname "$NGINX_PID_FILE")
        mkdir -p "$PID_DIR"
    fi
    
    # 启动 Nginx
    if systemctl start nginx 2>/dev/null; then
        echo -e "${GREEN}✓ Nginx 已启动${NC}"
    elif /etc/init.d/nginx start 2>/dev/null; then
        echo -e "${GREEN}✓ Nginx 已启动${NC}"
    elif nginx 2>/dev/null; then
        echo -e "${GREEN}✓ Nginx 已启动${NC}"
    else
        echo -e "${RED}✗ Nginx 启动失败${NC}"
        nginx -t
        exit 1
    fi
    
    sleep 2
    
    # 再次检查 PID 文件
    if [ -z "$NGINX_PID_FILE" ] || [ ! -f "$NGINX_PID_FILE" ] || [ ! -s "$NGINX_PID_FILE" ]; then
        MASTER_PID=$(pgrep -x nginx | head -1)
        if [ ! -z "$MASTER_PID" ]; then
            if [ -z "$NGINX_PID_FILE" ]; then
                NGINX_PID_FILE="/var/run/nginx.pid"
            fi
            PID_DIR=$(dirname "$NGINX_PID_FILE")
            mkdir -p "$PID_DIR"
            echo "$MASTER_PID" > "$NGINX_PID_FILE"
            echo -e "${GREEN}✓ 已创建 PID 文件: $NGINX_PID_FILE${NC}"
        fi
    fi
fi

# 验证 Nginx 状态
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}验证 Nginx 状态...${NC}"
echo -e "${BLUE}========================================${NC}"

if pgrep -x nginx > /dev/null; then
    echo -e "${GREEN}✓ Nginx 进程正在运行${NC}"
    
    if [ ! -z "$NGINX_PID_FILE" ] && [ -f "$NGINX_PID_FILE" ]; then
        PID=$(cat "$NGINX_PID_FILE" 2>/dev/null)
        if [ ! -z "$PID" ] && kill -0 "$PID" 2>/dev/null; then
            echo -e "${GREEN}✓ PID 文件有效 (PID: $PID)${NC}"
        else
            echo -e "${YELLOW}⚠ PID 文件中的 PID 无效，但 Nginx 正在运行${NC}"
        fi
    fi
    
    # 测试 HTTP 连接
    if curl -s http://localhost > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Nginx HTTP 服务正常${NC}"
    else
        echo -e "${YELLOW}⚠ Nginx HTTP 服务可能有问题${NC}"
    fi
    
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}修复完成！${NC}"
    exit 0
else
    echo -e "${RED}✗ Nginx 进程未运行${NC}"
    exit 1
fi
