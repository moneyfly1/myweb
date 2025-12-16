#!/bin/bash
# 修复清理日志功能的脚本

echo "=========================================="
echo "🔧 修复清理日志功能"
echo "=========================================="
echo ""

# 获取项目路径
PROJECT_DIR="/www/wwwroot/dy.moneyfly.top"
cd "$PROJECT_DIR" || {
    echo "❌ 错误: 无法进入项目目录: $PROJECT_DIR"
    exit 1
}

echo "📍 当前目录: $(pwd)"
echo ""

# 1. 检查后端代码
echo "1️⃣ 检查后端代码..."
if grep -q "ClearConfigUpdateLogs" internal/api/handlers/subscription_config.go 2>/dev/null; then
    echo "✅ 后端 handler 代码存在"
else
    echo "❌ 后端 handler 代码不存在，请先同步代码"
    exit 1
fi

if grep -q "config-update/logs/clear" internal/api/router/router.go 2>/dev/null; then
    echo "✅ 后端路由代码存在"
else
    echo "❌ 后端路由代码不存在，请先同步代码"
    exit 1
fi

# 2. 重新编译后端
echo ""
echo "2️⃣ 重新编译后端..."
if go build -o bin/server ./cmd/server/main.go 2>&1; then
    echo "✅ 后端编译成功"
else
    echo "❌ 后端编译失败"
    exit 1
fi

# 3. 重启后端服务器
echo ""
echo "3️⃣ 重启后端服务器..."
# 停止旧进程
pkill -f "bin/server" 2>&1 || true
sleep 2

# 启动新进程
if [ -f "bin/server" ]; then
    nohup ./bin/server > server.log 2>&1 &
    BACKEND_PID=$!
    echo $BACKEND_PID > server.pid
    sleep 3
    
    # 检查是否启动成功
    if ps -p $BACKEND_PID > /dev/null 2>&1; then
        echo "✅ 后端服务器已重启 (PID: $BACKEND_PID)"
        
        # 测试健康检查
        sleep 2
        if curl -s http://localhost:8000/health | grep -q "healthy"; then
            echo "✅ 后端健康检查通过"
        else
            echo "⚠️  后端可能还在启动中，请稍后检查"
        fi
    else
        echo "❌ 后端启动失败，查看日志: tail -f server.log"
        exit 1
    fi
else
    echo "❌ 编译后的服务器文件不存在"
    exit 1
fi

# 4. 检查前端代码
echo ""
echo "4️⃣ 检查前端代码..."
if grep -q "clearLogs.*config-update/logs/clear" frontend/src/utils/api.js 2>/dev/null; then
    echo "✅ 前端 API 代码存在"
else
    echo "❌ 前端 API 代码不存在，请先同步代码"
    exit 1
fi

# 5. 重新构建前端（生产环境）
echo ""
echo "5️⃣ 重新构建前端..."
cd frontend || {
    echo "❌ 无法进入 frontend 目录"
    exit 1
}

# 检查是否有 dist 目录（生产环境）
if [ -d "dist" ]; then
    echo "检测到 dist 目录，重新构建前端..."
    
    # 清理旧的构建文件
    rm -rf dist 2>&1 || true
    
    # 构建前端
    if npm run build 2>&1; then
        if [ -d "dist" ]; then
            echo "✅ 前端构建成功"
        else
            echo "❌ 前端构建失败：dist 目录不存在"
            exit 1
        fi
    else
        echo "❌ 前端构建失败"
        exit 1
    fi
else
    echo "⚠️  未检测到 dist 目录（可能是开发环境），跳过构建"
fi

cd ..

# 6. 测试路由
echo ""
echo "6️⃣ 测试清理日志路由..."
sleep 2
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8000/api/v1/admin/config-update/logs/clear \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer test" 2>&1)

if [ "$RESPONSE" = "401" ] || [ "$RESPONSE" = "403" ]; then
    echo "✅ 路由存在（返回认证错误是正常的，说明路由已注册）"
elif [ "$RESPONSE" = "404" ]; then
    echo "❌ 路由不存在（404），请检查："
    echo "   1. 服务器是否已重启"
    echo "   2. 代码是否已同步"
    echo "   3. 查看服务器日志: tail -f server.log"
else
    echo "⚠️  路由响应: $RESPONSE"
fi

echo ""
echo "=========================================="
echo "✅ 修复完成！"
echo ""
echo "📋 下一步操作："
echo "   1. 清除浏览器缓存（Ctrl+Shift+Delete 或 Cmd+Shift+Delete）"
echo "   2. 硬刷新页面（Ctrl+F5 或 Cmd+Shift+R）"
echo "   3. 重新测试清理日志功能"
echo ""
echo "📊 查看日志："
echo "   tail -f server.log"
echo "=========================================="

