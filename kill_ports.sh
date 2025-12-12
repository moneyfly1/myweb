#!/bin/bash

# 端口清理脚本 - 安全版本
# 用于查看和选择性杀死占用端口的进程

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}端口占用查看和清理工具${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 显示所有监听的端口
echo -e "${GREEN}当前监听的端口：${NC}"
echo ""

if command -v lsof &> /dev/null; then
    lsof -i -P -n | grep LISTEN | awk '{printf "%-10s %-10s %-20s %s\n", $2, $9, $1, $NF}' | head -30
elif command -v netstat &> /dev/null; then
    netstat -tlnp 2>/dev/null | grep LISTEN | head -30
elif command -v ss &> /dev/null; then
    ss -tlnp | grep LISTEN | head -30
fi

echo ""
echo -e "${YELLOW}请输入要清理的端口号（例如: 8000），或按 Ctrl+C 退出：${NC}"
read -p "端口号: " PORT

if [ -z "$PORT" ]; then
    echo -e "${RED}错误: 端口号不能为空${NC}"
    exit 1
fi

# 查找占用端口的进程
PIDS=""
if command -v lsof &> /dev/null; then
    PIDS=$(lsof -ti:$PORT 2>/dev/null)
elif command -v fuser &> /dev/null; then
    PIDS=$(fuser $PORT/tcp 2>/dev/null | awk '{print $1}')
fi

if [ -z "$PIDS" ]; then
    echo -e "${GREEN}端口 $PORT 未被占用${NC}"
    exit 0
fi

echo -e "${YELLOW}找到占用端口 $PORT 的进程：${NC}"
for PID in $PIDS; do
    if [ ! -z "$PID" ] && [ "$PID" != "-" ]; then
        PROCESS_INFO=$(ps -p $PID -o pid,comm,args 2>/dev/null | tail -1)
        echo -e "  PID: $PID - $PROCESS_INFO"
    fi
done

echo ""
read -p "确认要杀死这些进程? (y/N): " CONFIRM
if [[ "$CONFIRM" == "y" || "$CONFIRM" == "Y" ]]; then
    for PID in $PIDS; do
        if [ ! -z "$PID" ] && [ "$PID" != "-" ]; then
            echo -e "${YELLOW}正在杀死进程 $PID...${NC}"
            kill -9 $PID 2>/dev/null && echo -e "${GREEN}  ✓ 进程 $PID 已终止${NC}" || echo -e "${RED}  ✗ 无法终止进程 $PID${NC}"
        fi
    done
    echo -e "${GREEN}端口 $PORT 清理完成${NC}"
else
    echo -e "${YELLOW}操作已取消${NC}"
fi

