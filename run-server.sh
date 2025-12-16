#!/bin/bash

cd "$(dirname "$0")"

echo "🚀 启动 CBoard Go 服务器..."
echo ""

# 设置 Go 路径
if [ -d "/opt/homebrew/bin" ]; then
    export PATH="/opt/homebrew/bin:$PATH"
fi

# 检查 Go
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未找到 Go 命令"
    exit 1
fi

echo "✅ Go 版本: $(go version)"
echo ""

# 确保 .env 文件存在
if [ ! -f .env ]; then
    echo "创建 .env 文件..."
    cat > .env << 'ENVEOF'
# 服务器配置
HOST=0.0.0.0
PORT=8000
DEBUG=true

# 数据库配置（SQLite）
DATABASE_URL=sqlite:///./cboard.db

# JWT 配置
SECRET_KEY=change-me-to-a-strong-random-32-bytes-minimum-length

# CORS 配置
BACKEND_CORS_ORIGINS=http://localhost:5173,http://localhost:3000,http://localhost:8080

# 项目配置
PROJECT_NAME=CBoard Go
VERSION=1.0.0
API_V1_STR=/api/v1

# 邮件配置（可选）
SMTP_HOST=smtp.qq.com
SMTP_PORT=587
SMTP_USERNAME=your-email@qq.com
SMTP_PASSWORD=your-smtp-password
SMTP_FROM_EMAIL=your-email@qq.com
SMTP_FROM_NAME=CBoard Modern
SMTP_ENCRYPTION=tls

# 上传目录
UPLOAD_DIR=uploads
MAX_FILE_SIZE=10485760

# 定时任务
DISABLE_SCHEDULE_TASKS=false
ENVEOF
    echo "✅ .env 文件已创建"
fi

# 修复依赖
echo "修复依赖..."
go mod download 2>&1
go mod tidy 2>&1

if [ ! -f go.sum ]; then
    echo "❌ 警告: go.sum 文件未生成，但继续尝试..."
fi

# 编译服务器
echo "编译服务器..."
go build -o bin/server ./cmd/server/main.go 2>&1

if [ ! -f bin/server ]; then
    echo "❌ 编译失败！"
    exit 1
fi

echo "✅ 编译成功"
echo ""

# 停止旧进程
pkill -f "bin/server" 2>&1
sleep 1

# 停止旧进程
pkill -f "bin/server" 2>&1
sleep 1

# 启动服务器
echo "启动服务器..."
./bin/server > server.log 2>&1 &
SERVER_PID=$!
echo $SERVER_PID > server.pid

sleep 8

# 检查服务器状态
if ps -p $SERVER_PID > /dev/null 2>&1; then
    echo "✅ 服务器已启动 (PID: $SERVER_PID)"
    echo ""
    echo "📍 服务器地址: http://localhost:8000"
    echo "📍 健康检查: http://localhost:8000/health"
    echo "📍 API 地址: http://localhost:8000/api/v1"
    echo ""
    
    # 测试健康检查
    sleep 3
    HEALTH_RESPONSE=$(curl -s http://localhost:8000/health 2>&1)
    if [ $? -eq 0 ] && [ -n "$HEALTH_RESPONSE" ]; then
        echo "✅ 服务器运行正常！"
        echo "$HEALTH_RESPONSE"
        echo ""
    else
        echo "⏳ 服务器可能还在启动中..."
        echo "查看日志: tail -f server.log"
        echo ""
        echo "最近日志:"
        tail -20 server.log
    fi
    
    echo ""
    echo "数据库文件:"
    ls -lh cboard.db 2>&1 || echo "  数据库将在首次运行时创建"
    echo ""
    echo "停止服务器: kill $SERVER_PID"
else
    echo "❌ 服务器启动失败！"
    echo "查看日志:"
    tail -50 server.log
    exit 1
fi

