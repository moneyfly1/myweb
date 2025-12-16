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

# 停止通过 PID 文件记录的进程
if [ -f server.pid ]; then
    OLD_PID=$(cat server.pid 2>/dev/null || echo "")
    if [ -n "$OLD_PID" ] && ps -p "$OLD_PID" > /dev/null 2>&1; then
        echo "  停止旧后端进程 (PID: $OLD_PID)..."
        kill "$OLD_PID" 2>&1 || true
        sleep 2
        if ps -p "$OLD_PID" > /dev/null 2>&1; then
            echo "  强制停止进程..."
            kill -9 "$OLD_PID" 2>&1 || true
            sleep 1
        fi
    fi
    rm -f server.pid
fi

# 停止所有匹配的进程
pkill -f "bin/server" 2>&1 || true
pkill -f "vite" 2>&1 || true
sleep 2

# 检查并释放端口 8000
echo "检查端口 8000..."
PORT_8000_PID=""
if command -v lsof &> /dev/null; then
    PORT_8000_PID=$(lsof -ti:8000 2>/dev/null | head -1)
elif command -v fuser &> /dev/null; then
    PORT_8000_PID=$(fuser 8000/tcp 2>/dev/null | awk '{print $1}' | head -1)
elif command -v netstat &> /dev/null; then
    PORT_8000_PID=$(netstat -tlnp 2>/dev/null | grep ":8000 " | awk '{print $7}' | cut -d'/' -f1 | head -1)
    [ "$PORT_8000_PID" = "-" ] && PORT_8000_PID=""
fi

if [ -n "$PORT_8000_PID" ] && [ "$PORT_8000_PID" != "$$" ]; then
    echo "  发现端口 8000 被占用 (PID: $PORT_8000_PID)，正在释放..."
    kill "$PORT_8000_PID" 2>&1 || true
    sleep 2
    if ps -p "$PORT_8000_PID" > /dev/null 2>&1; then
        echo "  强制终止进程..."
        kill -9 "$PORT_8000_PID" 2>&1 || true
        sleep 1
    fi
fi

# 检查并释放端口 5173
echo "检查端口 5173..."
PORT_5173_PID=""
if command -v lsof &> /dev/null; then
    PORT_5173_PID=$(lsof -ti:5173 2>/dev/null | head -1)
elif command -v fuser &> /dev/null; then
    PORT_5173_PID=$(fuser 5173/tcp 2>/dev/null | awk '{print $1}' | head -1)
elif command -v netstat &> /dev/null; then
    PORT_5173_PID=$(netstat -tlnp 2>/dev/null | grep ":5173 " | awk '{print $7}' | cut -d'/' -f1 | head -1)
    [ "$PORT_5173_PID" = "-" ] && PORT_5173_PID=""
fi

if [ -n "$PORT_5173_PID" ] && [ "$PORT_5173_PID" != "$$" ]; then
    echo "  发现端口 5173 被占用 (PID: $PORT_5173_PID)，正在释放..."
    kill "$PORT_5173_PID" 2>&1 || true
    sleep 2
    if ps -p "$PORT_5173_PID" > /dev/null 2>&1; then
        echo "  强制终止进程..."
        kill -9 "$PORT_5173_PID" 2>&1 || true
        sleep 1
    fi
fi

sleep 1

# 启动后端
echo ""
echo "启动后端服务器 (端口 8000)..."

# 再次检查端口是否已释放
if command -v lsof &> /dev/null; then
    if lsof -ti:8000 &>/dev/null; then
        echo "❌ 错误: 端口 8000 仍被占用，无法启动后端"
        echo "请手动检查: lsof -i:8000"
        exit 1
    fi
fi

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

# 检查 Node.js 版本
if command -v node &> /dev/null; then
    NODE_VER=$(node -v | sed 's/v//')
    NODE_MAJOR=$(echo "$NODE_VER" | cut -d. -f1)
    echo "  Node.js 版本: v$NODE_VER"
    
    if [ "$NODE_MAJOR" -lt 18 ]; then
        echo "⚠️  警告: Node.js 版本过低 (v$NODE_VER)，建议使用 Node.js 18+"
        echo "  如果遇到问题，请升级 Node.js: https://nodejs.org/"
    fi
else
    echo "❌ 错误: 未找到 Node.js"
    echo "请先安装 Node.js 18+"
    exit 1
fi

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
    # 检查 vite 版本是否匹配 package.json
    PACKAGE_VITE=$(cat package.json | grep '"vite"' | head -1 | sed 's/.*"vite": *"\([^"]*\)".*/\1/' | sed 's/\^//' | sed 's/~//')
    if [ -z "$PACKAGE_VITE" ]; then
        PACKAGE_VITE="4.5.0"
    fi
    
    # 获取已安装的 vite 版本
    INSTALLED_VITE=$(node -e "try { console.log(require('./node_modules/vite/package.json').version); } catch(e) { console.log(''); }" 2>/dev/null || echo "")
    
    if [ -z "$INSTALLED_VITE" ]; then
        echo "  无法读取已安装的 vite 版本，需要重新安装"
        NEED_INSTALL=true
    else
        echo "  package.json 要求: vite $PACKAGE_VITE"
        echo "  已安装版本: vite $INSTALLED_VITE"
        
        # 检查主版本号是否匹配（4.x vs 5.x）
        PACKAGE_MAJOR=$(echo "$PACKAGE_VITE" | cut -d. -f1)
        INSTALLED_MAJOR=$(echo "$INSTALLED_VITE" | cut -d. -f1)
        
        if [ "$PACKAGE_MAJOR" != "$INSTALLED_MAJOR" ]; then
            echo "  vite 主版本不匹配（已安装: $INSTALLED_MAJOR.x，需要: $PACKAGE_MAJOR.x），需要重新安装"
            NEED_INSTALL=true
        fi
    fi
fi

if [ "$NEED_INSTALL" = true ]; then
    echo "清理并重新安装前端依赖..."
    rm -rf node_modules package-lock.json 2>&1 || true
    npm cache clean --force 2>&1 || true
    
    # 强制设置正确的 npm 镜像（清除可能存在的错误配置）
    echo "  配置 npm 镜像源..."
    npm config delete registry 2>&1 || true
    npm config set registry https://registry.npmmirror.com 2>&1 || true
    
    # 验证镜像配置
    CURRENT_REGISTRY=$(npm config get registry 2>/dev/null || echo "")
    echo "  当前 npm 镜像: $CURRENT_REGISTRY"
    
    # 如果镜像配置不正确，再次设置
    if [ -z "$CURRENT_REGISTRY" ] || echo "$CURRENT_REGISTRY" | grep -qv "npmmirror\|npmjs"; then
        echo "  重新设置镜像源..."
        npm config set registry https://registry.npmmirror.com 2>&1 || true
    fi
    
    echo "  正在安装依赖（这可能需要几分钟）..."
    
    # 尝试安装（最多重试3次）
    INSTALL_SUCCESS=false
    for attempt in 1 2 3; do
        if [ $attempt -gt 1 ]; then
            echo "  第 $attempt 次尝试安装..."
            sleep 2
            
            # 如果失败，尝试切换到官方源
            if [ $attempt -eq 2 ]; then
                echo "  尝试切换到官方 npm 源..."
                npm config set registry https://registry.npmjs.org/ 2>&1 || true
            elif [ $attempt -eq 3 ]; then
                echo "  再次尝试使用淘宝镜像..."
                npm config set registry https://registry.npmmirror.com 2>&1 || true
            fi
        fi
        
        if npm install --legacy-peer-deps 2>&1 | tee /tmp/npm_install.log | tail -30; then
            INSTALL_SUCCESS=true
            echo "✅ 依赖安装完成"
            break
        else
            echo "⚠️  安装失败，错误信息:"
            tail -20 /tmp/npm_install.log 2>/dev/null || true
            
            # 检查是否是镜像问题
            if grep -q "404\|Not Found\|mirrors.tuna" /tmp/npm_install.log 2>/dev/null; then
                echo "  检测到镜像问题，清理缓存并切换镜像..."
                npm cache clean --force 2>&1 || true
                if [ $attempt -eq 1 ]; then
                    npm config set registry https://registry.npmjs.org/ 2>&1 || true
                else
                    npm config set registry https://registry.npmmirror.com 2>&1 || true
                fi
            fi
        fi
    done
    
    # 如果标准安装失败，尝试使用 --force
    if [ "$INSTALL_SUCCESS" = false ]; then
        echo "⚠️  标准安装失败，尝试使用 --force..."
        if npm install --force 2>&1 | tail -30; then
            INSTALL_SUCCESS=true
        fi
    fi
    
    # 验证安装
    if [ ! -f node_modules/.bin/vite ]; then
        echo "❌ vite 可执行文件仍未找到，尝试直接安装 vite..."
        # 确保使用正确的镜像
        npm config set registry https://registry.npmmirror.com 2>&1 || true
        npm install vite@4.5.0 --legacy-peer-deps --save-dev 2>&1 | tail -20 || true
    fi
    
    # 最终验证
    if [ ! -f node_modules/.bin/vite ]; then
        echo "❌ 错误: 无法安装 vite"
        echo ""
        echo "可能的解决方案:"
        echo "1. 检查网络连接"
        echo "2. 尝试使用国内镜像:"
        echo "   npm config set registry https://registry.npmmirror.com"
        echo "   npm install --legacy-peer-deps"
        echo "3. 手动运行: cd frontend && npm install --legacy-peer-deps"
        exit 1
    fi
    
    # 验证 vite 版本
    FINAL_VITE=$(node -e "try { console.log(require('./node_modules/vite/package.json').version); } catch(e) { console.log(''); }" 2>/dev/null || echo "")
    if [ -n "$FINAL_VITE" ]; then
        echo "✅ vite 已安装: $FINAL_VITE"
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
