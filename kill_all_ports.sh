#!/bin/bash

# 批量端口清理脚本 - 安全版本
# ⚠️ 警告：此脚本会杀死所有非系统关键端口的进程

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 系统关键端口（这些端口不会被清理）
SYSTEM_PORTS=(22 80 443 3306 6379 53 25 587 465 993 995 143 110)

echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${RED}⚠️  警告：批量端口清理工具 ⚠️${NC}"
echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}此脚本将：${NC}"
echo -e "  • 显示所有监听的端口"
echo -e "  • 清理所有非系统关键端口的进程"
echo -e "  • 保护以下系统端口: ${SYSTEM_PORTS[*]}"
echo ""
echo -e "${RED}⚠️  注意：此操作可能会终止正在运行的服务！${NC}"
echo ""

# 显示所有端口
echo -e "${GREEN}当前所有监听的端口：${NC}"
echo ""

if command -v lsof &> /dev/null; then
    ALL_PORTS=$(lsof -i -P -n | grep LISTEN | awk '{print $9}' | sed 's/.*://' | sort -u)
    lsof -i -P -n | grep LISTEN | awk '{printf "%-10s %-10s %-20s %s\n", $2, $9, $1, $NF}' | head -50
elif command -v netstat &> /dev/null; then
    ALL_PORTS=$(netstat -tlnp 2>/dev/null | grep LISTEN | awk '{print $4}' | sed 's/.*://' | sort -u)
    netstat -tlnp 2>/dev/null | grep LISTEN | head -50
elif command -v ss &> /dev/null; then
    ALL_PORTS=$(ss -tlnp | grep LISTEN | awk '{print $4}' | sed 's/.*://' | sort -u)
    ss -tlnp | grep LISTEN | head -50
fi

echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}请选择操作：${NC}"
echo "1. 清理所有非系统端口（推荐）"
echo "2. 清理指定端口列表（用逗号分隔，如: 8000,8080,3000）"
echo "3. 仅查看，不清理"
echo "4. 退出"
echo ""
read -p "请输入选项 (1/2/3/4): " OPTION

case $OPTION in
    1)
        echo ""
        echo -e "${YELLOW}将清理所有非系统关键端口...${NC}"
        echo -e "${GREEN}受保护的系统端口: ${SYSTEM_PORTS[*]}${NC}"
        echo ""
        read -p "确认要继续? (yes/no): " CONFIRM
        
        if [[ "$CONFIRM" != "yes" ]]; then
            echo -e "${YELLOW}操作已取消${NC}"
            exit 0
        fi
        
        KILLED_COUNT=0
        SKIPPED_COUNT=0
        
        for PORT in $ALL_PORTS; do
            # 检查是否为系统端口
            IS_SYSTEM=false
            for SYS_PORT in "${SYSTEM_PORTS[@]}"; do
                if [ "$PORT" == "$SYS_PORT" ]; then
                    IS_SYSTEM=true
                    break
                fi
            done
            
            if [ "$IS_SYSTEM" = true ]; then
                echo -e "${BLUE}跳过系统端口: $PORT${NC}"
                SKIPPED_COUNT=$((SKIPPED_COUNT + 1))
                continue
            fi
            
            # 查找占用端口的进程
            if command -v lsof &> /dev/null; then
                PIDS=$(lsof -ti:$PORT 2>/dev/null)
            elif command -v fuser &> /dev/null; then
                PIDS=$(fuser $PORT/tcp 2>/dev/null | awk '{print $1}')
            fi
            
            if [ ! -z "$PIDS" ]; then
                for PID in $PIDS; do
                    if [ ! -z "$PID" ] && [ "$PID" != "-" ]; then
                        PROCESS_NAME=$(ps -p $PID -o comm= 2>/dev/null)
                        echo -e "${YELLOW}清理端口 $PORT (PID: $PID, 进程: $PROCESS_NAME)${NC}"
                        kill -9 $PID 2>/dev/null && {
                            echo -e "${GREEN}  ✓ 已终止${NC}"
                            KILLED_COUNT=$((KILLED_COUNT + 1))
                        } || echo -e "${RED}  ✗ 无法终止（可能需要权限）${NC}"
                    fi
                done
            fi
        done
        
        echo ""
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${GREEN}清理完成！${NC}"
        echo -e "${GREEN}已终止进程数: $KILLED_COUNT${NC}"
        echo -e "${GREEN}已跳过系统端口数: $SKIPPED_COUNT${NC}"
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        ;;
    2)
        echo ""
        read -p "请输入要清理的端口列表（用逗号分隔，如: 8000,8080,3000）: " PORT_LIST
        
        if [ -z "$PORT_LIST" ]; then
            echo -e "${RED}错误: 端口列表不能为空${NC}"
            exit 1
        fi
        
        echo ""
        read -p "确认要清理这些端口? (y/N): " CONFIRM
        if [[ "$CONFIRM" != "y" && "$CONFIRM" != "Y" ]]; then
            echo -e "${YELLOW}操作已取消${NC}"
            exit 0
        fi
        
        KILLED_COUNT=0
        IFS=',' read -ra PORTS <<< "$PORT_LIST"
        
        for PORT in "${PORTS[@]}"; do
            PORT=$(echo "$PORT" | tr -d ' ')
            if [ -z "$PORT" ]; then
                continue
            fi
            
            # 查找占用端口的进程
            if command -v lsof &> /dev/null; then
                PIDS=$(lsof -ti:$PORT 2>/dev/null)
            elif command -v fuser &> /dev/null; then
                PIDS=$(fuser $PORT/tcp 2>/dev/null | awk '{print $1}')
            fi
            
            if [ -z "$PIDS" ]; then
                echo -e "${BLUE}端口 $PORT 未被占用${NC}"
                continue
            fi
            
            echo -e "${YELLOW}清理端口 $PORT...${NC}"
            for PID in $PIDS; do
                if [ ! -z "$PID" ] && [ "$PID" != "-" ]; then
                    PROCESS_NAME=$(ps -p $PID -o comm= 2>/dev/null)
                    kill -9 $PID 2>/dev/null && {
                        echo -e "${GREEN}  ✓ PID $PID ($PROCESS_NAME) 已终止${NC}"
                        KILLED_COUNT=$((KILLED_COUNT + 1))
                    } || echo -e "${RED}  ✗ 无法终止 PID $PID${NC}"
                fi
            done
        done
        
        echo ""
        echo -e "${GREEN}清理完成！已终止 $KILLED_COUNT 个进程${NC}"
        ;;
    3)
        echo ""
        echo -e "${GREEN}仅查看模式，未执行任何清理操作${NC}"
        ;;
    4)
        echo -e "${GREEN}退出${NC}"
        exit 0
        ;;
    *)
        echo -e "${RED}无效选项${NC}"
        exit 1
        ;;
esac

