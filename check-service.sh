#!/bin/bash
# 检查服务状态的脚本

echo "=========================================="
echo "🔍 检查 CBoard 服务状态"
echo "=========================================="
echo ""

# 1. 检查服务状态
echo "1️⃣ 服务状态："
systemctl status cboard --no-pager -l | head -20
echo ""

# 2. 查看最近的日志
echo "2️⃣ 最近的错误日志："
journalctl -u cboard -n 30 --no-pager | tail -30
echo ""

# 3. 检查服务配置文件
echo "3️⃣ 服务配置文件："
cat /etc/systemd/system/cboard.service
echo ""

# 4. 检查可执行文件
echo "4️⃣ 检查可执行文件："
PROJECT_DIR="/www/wwwroot/dy.moneyfly.top"
if [ -f "$PROJECT_DIR/server" ]; then
    echo "✅ server 文件存在"
    ls -lh "$PROJECT_DIR/server"
    file "$PROJECT_DIR/server"
    
    # 检查是否有执行权限
    if [ -x "$PROJECT_DIR/server" ]; then
        echo "✅ server 文件有执行权限"
    else
        echo "❌ server 文件没有执行权限"
        echo "   修复: chmod +x $PROJECT_DIR/server"
    fi
else
    echo "❌ server 文件不存在"
    echo "   需要编译: cd $PROJECT_DIR && go build -o server ./cmd/server/main.go"
fi
echo ""

# 5. 检查工作目录
echo "5️⃣ 检查工作目录："
if [ -d "$PROJECT_DIR" ]; then
    echo "✅ 工作目录存在: $PROJECT_DIR"
    ls -la "$PROJECT_DIR" | head -10
else
    echo "❌ 工作目录不存在: $PROJECT_DIR"
fi
echo ""

# 6. 检查端口占用
echo "6️⃣ 检查端口占用："
if command -v netstat &> /dev/null; then
    netstat -tlnp | grep -E ":(8000|8080)" || echo "   端口未被占用"
elif command -v ss &> /dev/null; then
    ss -tlnp | grep -E ":(8000|8080)" || echo "   端口未被占用"
fi
echo ""

# 7. 尝试手动运行（测试）
echo "7️⃣ 测试手动运行："
if [ -f "$PROJECT_DIR/server" ] && [ -x "$PROJECT_DIR/server" ]; then
    echo "   尝试运行服务器（5秒后自动停止）..."
    cd "$PROJECT_DIR"
    timeout 5 ./server 2>&1 | head -20 || echo "   运行测试完成"
else
    echo "   ⚠️  无法测试（server 文件不存在或没有执行权限）"
fi

echo ""
echo "=========================================="
echo "✅ 检查完成"
echo "=========================================="

