#!/bin/bash
# 遇到错误不立即退出，允许重试
set +e

cd "$(dirname "$0")"

echo "=========================================="
echo "🚀 启动 CBoard Go 服务"
echo "=========================================="
echo ""

# 设置 Go 路径
export PATH="/opt/homebrew/bin:$PATH"

# 检查 Go
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未找到 Go 命令"
    exit 1
fi

echo "✅ Go 版本: $(go version)"
echo ""

# 确保 .env 存在
if [ ! -f .env ]; then
    echo "创建 .env 文件..."
    cat > .env << 'ENVEOF'
HOST=0.0.0.0
PORT=8000
DEBUG=true
DATABASE_URL=sqlite:///./cboard.db
SECRET_KEY=change-me-to-a-strong-random-32-bytes-minimum-length
BACKEND_CORS_ORIGINS=http://localhost:5173,http://localhost:3000,http://localhost:8080
PROJECT_NAME=CBoard Go
VERSION=1.0.0
API_V1_STR=/api/v1
SMTP_HOST=smtp.qq.com
SMTP_PORT=587
SMTP_USERNAME=your-email@qq.com
SMTP_PASSWORD=your-smtp-password
SMTP_FROM_EMAIL=your-email@qq.com
SMTP_FROM_NAME=CBoard Modern
SMTP_ENCRYPTION=tls
UPLOAD_DIR=uploads
MAX_FILE_SIZE=10485760
DISABLE_SCHEDULE_TASKS=false
ENVEOF
    echo "✅ .env 已创建"
fi

# 修复依赖
echo "修复 Go 依赖..."
echo "  1. 设置 Go 代理（直接模式）..."
export GOPROXY=direct
export GOSUMDB=sum.golang.google.cn
echo "   GOPROXY=$GOPROXY"

echo "  2. 下载所有依赖..."
go mod download 2>&1 || true

echo "  3. 整理依赖..."
go mod tidy 2>&1 || true

# 验证 go.sum
if [ ! -f go.sum ]; then
    echo "⚠️  go.sum 文件未生成，尝试强制生成..."
    go mod tidy 2>&1 || true
    
    if [ ! -f go.sum ]; then
        echo "❌ 错误: 无法生成 go.sum 文件"
        echo "请检查网络连接或手动运行: go mod tidy"
        exit 1
    fi
fi

SUM_LINES=$(wc -l < go.sum)
if [ $SUM_LINES -lt 100 ]; then
    echo "⚠️  go.sum 文件可能不完整 ($SUM_LINES 行)，尝试补充..."
    go mod download 2>&1 || true
    go mod tidy 2>&1 || true
    SUM_LINES=$(wc -l < go.sum)
fi
echo "✅ go.sum 已生成 ($SUM_LINES 行)"

# 编译
echo ""
echo "编译服务器..."
if go build -o bin/server ./cmd/server/main.go 2>&1; then
    echo "✅ 编译成功"
else
    echo "❌ 编译失败！"
    echo "尝试再次修复依赖..."
    go mod download 2>&1 || true
    go mod tidy 2>&1 || true
    
    if ! go build -o bin/server ./cmd/server/main.go 2>&1; then
        echo "❌ 编译仍然失败！"
        echo "请检查错误信息或手动运行:"
        echo "  go mod tidy"
        echo "  go build -o bin/server ./cmd/server/main.go"
        exit 1
    fi
    echo "✅ 编译成功（修复后）"
fi

# 停止旧进程
echo ""
echo "停止旧进程..."
pkill -f "bin/server" 2>&1 || true
pkill -f "vite" 2>&1 || true
sleep 2

# 启动后端
echo ""
echo "启动后端服务器 (端口 8000)..."
./bin/server > server.log 2>&1 &
BACKEND_PID=$!
echo $BACKEND_PID > server.pid

sleep 10

# 检查后端
echo ""
echo "检查后端服务器..."
if ps -p $BACKEND_PID > /dev/null 2>&1; then
    echo "✅ 后端进程运行中 (PID: $BACKEND_PID)"
    
    # 测试健康检查
    for i in {1..5}; do
        if HEALTH=$(curl -s http://localhost:8000/health 2>&1); then
            if echo "$HEALTH" | grep -q "healthy"; then
                echo "✅ 后端健康检查通过: $HEALTH"
                break
            fi
        fi
        if [ $i -eq 5 ]; then
            echo "⚠️  后端可能还在启动中..."
            echo "最近日志:"
            tail -30 server.log
        else
            sleep 2
        fi
    done
else
    echo "❌ 后端启动失败！"
    echo "错误日志:"
    cat server.log
    exit 1
fi

# 检查数据库
echo ""
echo "检查数据库..."
if [ -f cboard.db ]; then
    echo "✅ 数据库文件已创建: $(ls -lh cboard.db | awk '{print $5}')"
else
    echo "⚠️  数据库文件未创建（可能首次运行）"
fi

# 启动前端
echo ""
echo "启动前端服务器 (端口 5173)..."
cd frontend

# 检查并修复前端依赖
echo "检查前端依赖..."
NEED_INSTALL=false

# 检查 node_modules 是否存在
if [ ! -d node_modules ]; then
    echo "  node_modules 不存在，需要安装"
    NEED_INSTALL=true
elif [ ! -f node_modules/.bin/vite ]; then
    echo "  vite 可执行文件不存在，需要重新安装"
    NEED_INSTALL=true
elif [ ! -d node_modules/vite ]; then
    echo "  vite 模块不存在，需要重新安装"
    NEED_INSTALL=true
else
    # 检查 vite 版本是否匹配
    INSTALLED_VITE=$(cat package.json | grep '"vite"' | head -1 | sed 's/.*"vite": *"\([^"]*\)".*/\1/')
    if [ -z "$INSTALLED_VITE" ]; then
        INSTALLED_VITE="^5.0.0"
    fi
    echo "  检查 vite 版本: $INSTALLED_VITE"
    
    # 如果 package.json 中 vite 是 5.x，但安装的是 4.x，需要重新安装
    if echo "$INSTALLED_VITE" | grep -q "^5"; then
        VITE_VERSION=$(node -e "console.log(require('./node_modules/vite/package.json').version)" 2>/dev/null || echo "")
        if [ -n "$VITE_VERSION" ] && echo "$VITE_VERSION" | grep -q "^4"; then
            echo "  vite 版本不匹配（已安装: $VITE_VERSION，需要: $INSTALLED_VITE），需要重新安装"
            NEED_INSTALL=true
        fi
    fi
fi

if [ "$NEED_INSTALL" = true ]; then
    echo "清理并重新安装前端依赖..."
    rm -rf node_modules package-lock.json 2>&1 || true
    npm cache clean --force 2>&1 || true
    echo "  正在安装依赖（这可能需要几分钟）..."
    
    # 尝试安装
    if npm install --legacy-peer-deps 2>&1 | tee /tmp/npm_install.log | tail -30; then
        echo "✅ 依赖安装完成"
    else
        echo "⚠️  标准安装失败，尝试使用 --force..."
        npm install --force 2>&1 | tail -30 || true
    fi
    
    # 验证安装
    if [ ! -f node_modules/.bin/vite ]; then
        echo "❌ vite 可执行文件仍未找到，尝试直接安装 vite..."
        npm install vite@latest --legacy-peer-deps --save-dev 2>&1 | tail -20 || true
    fi
    
    # 最终验证
    if [ ! -f node_modules/.bin/vite ]; then
        echo "❌ 错误: 无法安装 vite"
        echo "请手动运行: cd frontend && npm install --legacy-peer-deps"
        exit 1
    fi
else
    echo "✅ 前端依赖已就绪"
fi

# 启动前端开发服务器
npm run dev > ../frontend.log 2>&1 &
FRONTEND_PID=$!
echo $FRONTEND_PID > ../frontend.pid
cd ..

sleep 10

# 检查前端
echo ""
echo "检查前端服务器..."
if ps -p $FRONTEND_PID > /dev/null 2>&1; then
    echo "✅ 前端进程运行中 (PID: $FRONTEND_PID)"
    
    # 测试前端
    for i in {1..5}; do
        if FRONTEND=$(curl -s http://localhost:5173 2>&1 | head -1); then
            if [ -n "$FRONTEND" ]; then
                echo "✅ 前端响应正常"
                break
            fi
        fi
        if [ $i -eq 5 ]; then
            echo "⚠️  前端可能还在启动中..."
            echo "最近日志:"
            tail -30 frontend.log
        else
            sleep 2
        fi
    done
else
    echo "❌ 前端启动失败！"
    echo "错误日志:"
    cat frontend.log
fi

echo ""
echo "=========================================="
echo "✅ 启动完成！"
echo ""
echo "📍 后端 API: http://localhost:8000"
echo "📍 前端界面: http://localhost:5173"
echo "📍 健康检查: http://localhost:8000/health"
echo ""
echo "查看日志:"
echo "  tail -f server.log"
echo "  tail -f frontend.log"
echo ""
echo "停止服务:"
echo "  kill $BACKEND_PID  # 停止后端"
echo "  kill $FRONTEND_PID  # 停止前端"
echo "=========================================="
