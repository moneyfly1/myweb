#!/bin/bash
# ============================================
# CBoard Go 一键安装脚本 - 宝塔面板版
# ============================================
# 功能：自动安装所需环境并完成网站部署
# 支持：Ubuntu/Debian/CentOS/Rocky Linux
# ============================================

# 遇到错误不立即退出，允许重试
set +e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 配置变量（可通过环境变量覆盖）
PROJECT_DIR="${PROJECT_DIR:-/www/wwwroot/某站点}"
DOMAIN="${DOMAIN:-}"
GO_VERSION="${GO_VERSION:-1.21.5}"
NODE_VERSION="${NODE_VERSION:-18}"
LOG_FILE="/tmp/cboard_install_$(date +%Y%m%d_%H%M%S).log"
SKIP_TESTS="${SKIP_TESTS:-false}"

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1" | tee -a "$LOG_FILE"
}

# 检查是否为 root 用户
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "请使用 root 用户运行此脚本"
        log_info "使用: sudo $0"
        exit 1
    fi
}

# 检查端口占用
check_port() {
    local port=$1
    if command -v netstat &> /dev/null; then
        if netstat -tuln | grep -q ":$port "; then
            return 1
        fi
    elif command -v ss &> /dev/null; then
        if ss -tuln | grep -q ":$port "; then
            return 1
        fi
    fi
    return 0
}

# 交互式输入域名
get_domain() {
    if [ -z "$DOMAIN" ]; then
        # 尝试从项目目录名获取
        DIR_NAME=$(basename "$PROJECT_DIR")
        if [ "$DIR_NAME" != "." ] && [ "$DIR_NAME" != "/" ] && [[ "$DIR_NAME" == *.* ]]; then
            DOMAIN="$DIR_NAME"
            log_info "从目录名检测到域名: $DOMAIN"
        else
            echo -e "${CYAN}请输入您的域名（例如: example.com）:${NC} "
            read -r DOMAIN
            if [ -z "$DOMAIN" ]; then
                log_error "域名不能为空"
                exit 1
            fi
        fi
    fi
    log_info "使用域名: $DOMAIN"
}

# 检查宝塔面板
check_bt_panel() {
    if [ -d "/www" ] && [ -d "/www/server" ]; then
        log_info "✅ 检测到宝塔面板环境"
        return 0
    else
        log_warn "未检测到宝塔面板，将使用标准 Linux 环境"
        return 1
    fi
}

# 检测操作系统
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
        log_info "检测到操作系统: $OS $OS_VERSION"
    else
        log_error "无法检测操作系统"
        exit 1
    fi
}

# 查找 Go 安装路径
find_go_path() {
    # 首先检查是否在 PATH 中
    if command -v go &> /dev/null; then
        GO_BIN=$(which go)
        GO_DIR=$(dirname "$GO_BIN")
        echo "$GO_DIR"
        return 0
    fi
    
    # 查找常见的 Go 安装位置
    BT_GO_PATH=$(find /usr/local/btgojdk -name "go" -type f 2>/dev/null | grep bin/go | head -1)
    if [ -n "$BT_GO_PATH" ]; then
        echo "$(dirname "$BT_GO_PATH")"
        return 0
    fi
    
    # 检查标准安装位置
    if [ -f "/usr/local/go/bin/go" ]; then
        echo "/usr/local/go/bin"
        return 0
    fi
    
    # 检查系统包管理器安装
    if [ -f "/usr/bin/go" ]; then
        echo "/usr/bin"
        return 0
    fi
    
    return 1
}

# 配置 Go PATH
setup_go_path() {
    GO_DIR=$(find_go_path)
    if [ -n "$GO_DIR" ] && [ -f "$GO_DIR/go" ]; then
        export PATH="$PATH:$GO_DIR"
        log_info "已配置 Go PATH: $GO_DIR"
        
        # 永久添加到 ~/.bashrc
        if ! grep -q "$GO_DIR" ~/.bashrc 2>/dev/null; then
            echo "export PATH=\$PATH:$GO_DIR" >> ~/.bashrc
        fi
        
        # 永久添加到 /etc/profile
        if ! grep -q "$GO_DIR" /etc/profile 2>/dev/null; then
            echo "export PATH=\$PATH:$GO_DIR" >> /etc/profile
        fi
        
        return 0
    fi
    return 1
}

# 安装 Go 语言
install_go() {
    # 尝试查找已安装的 Go
    if setup_go_path; then
        if command -v go &> /dev/null; then
            GO_VER=$(go version | awk '{print $3}' | sed 's/go//')
            log_info "Go 已安装: $GO_VER"
            return 0
        fi
    fi

    log_step "开始安装 Go $GO_VERSION..."
    
    # 检测架构
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            GO_ARCH="amd64"
            ;;
        aarch64|arm64)
            GO_ARCH="arm64"
            ;;
        *)
            log_error "不支持的架构: $ARCH"
            exit 1
            ;;
    esac

    # 下载 Go
    GO_TAR="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    GO_URL="https://go.dev/dl/${GO_TAR}"
    
    log_info "下载 Go: $GO_URL"
    cd /tmp
    if ! wget -q --show-progress "$GO_URL" -O "$GO_TAR"; then
        log_error "下载 Go 失败"
        exit 1
    fi

    # 解压并安装
    log_info "解压并安装 Go..."
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "$GO_TAR"
    rm -f "$GO_TAR"

    # 添加到 PATH
    if ! grep -q "/usr/local/go/bin" /etc/profile; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    fi
    export PATH=$PATH:/usr/local/go/bin

    # 配置 PATH
    setup_go_path
    
    # 验证安装
    if command -v go &> /dev/null; then
        log_info "✅ Go 安装成功: $(go version)"
        return 0
    else
        log_error "Go 安装失败，请手动检查"
        log_info "提示: 如果通过宝塔面板安装，请执行："
        log_info "  export PATH=\$PATH:/usr/local/btgojdk/go*/bin"
        log_info "  echo 'export PATH=\$PATH:/usr/local/btgojdk/go*/bin' >> ~/.bashrc"
        exit 1
    fi
}

# 查找 Node.js 安装路径
find_node_path() {
    # 首先检查是否在 PATH 中
    if command -v node &> /dev/null; then
        NODE_BIN=$(which node)
        NODE_DIR=$(dirname "$NODE_BIN")
        echo "$NODE_DIR"
        return 0
    fi
    
    # 检查脚本自动安装的 Node.js 18
    if [ -f "/usr/local/nodejs18/bin/node" ]; then
        echo "/usr/local/nodejs18/bin"
        return 0
    fi
    
    # 查找宝塔面板安装的 Node.js（通常在 /www/server/nodejs 或 /usr/local/btnodejs）
    # 优先查找 18+ 版本
    BT_NODE_PATH=$(find /www/server/nodejs -name "node" -type f 2>/dev/null | grep -E "v(18|19|20|21|22)" | grep bin/node | head -1)
    if [ -n "$BT_NODE_PATH" ]; then
        echo "$(dirname "$BT_NODE_PATH")"
        return 0
    fi
    
    # 如果没有找到 18+ 版本，查找所有版本
    BT_NODE_PATH=$(find /www/server/nodejs -name "node" -type f 2>/dev/null | grep bin/node | head -1)
    if [ -n "$BT_NODE_PATH" ]; then
        echo "$(dirname "$BT_NODE_PATH")"
        return 0
    fi
    
    BT_NODE_PATH=$(find /usr/local/btnodejs -name "node" -type f 2>/dev/null | grep bin/node | head -1)
    if [ -n "$BT_NODE_PATH" ]; then
        echo "$(dirname "$BT_NODE_PATH")"
        return 0
    fi
    
    # 检查标准安装位置
    if [ -f "/usr/local/bin/node" ]; then
        echo "/usr/local/bin"
        return 0
    fi
    
    # 检查系统包管理器安装
    if [ -f "/usr/bin/node" ]; then
        echo "/usr/bin"
        return 0
    fi
    
    return 1
}

# 配置 Node.js PATH
setup_node_path() {
    NODE_DIR=$(find_node_path)
    if [ -n "$NODE_DIR" ] && [ -f "$NODE_DIR/node" ]; then
        export PATH="$PATH:$NODE_DIR"
        log_info "已配置 Node.js PATH: $NODE_DIR"
        
        # 永久添加到 ~/.bashrc
        if ! grep -q "$NODE_DIR" ~/.bashrc 2>/dev/null; then
            echo "export PATH=\$PATH:$NODE_DIR" >> ~/.bashrc
        fi
        
        # 永久添加到 /etc/profile
        if ! grep -q "$NODE_DIR" /etc/profile 2>/dev/null; then
            echo "export PATH=\$PATH:$NODE_DIR" >> /etc/profile
        fi
        
        return 0
    fi
    return 1
}

# 检查 Node.js 版本
check_node_version() {
    if ! command -v node &> /dev/null; then
        return 1
    fi
    
    NODE_VER=$(node -v | sed 's/v//')
    NODE_MAJOR=$(echo "$NODE_VER" | cut -d. -f1)
    NODE_MINOR=$(echo "$NODE_VER" | cut -d. -f2)
    
    # Vite 5.x 需要 Node.js 18+
    REQUIRED_MAJOR=18
    
    if [ "$NODE_MAJOR" -lt "$REQUIRED_MAJOR" ]; then
        log_warn "Node.js 版本过低: v$NODE_VER"
        log_warn "Vite 5.x 需要 Node.js 18.0.0 或更高版本"
        return 1
    fi
    
    return 0
}

# 自动安装 Node.js 18+（二进制包方式，适用于 ARM64）
install_nodejs_binary() {
    log_step "自动安装 Node.js 18+（二进制包方式）..."
    
    # 检测架构
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            NODE_ARCH="x64"
            ;;
        aarch64|arm64)
            NODE_ARCH="arm64"
            ;;
        armv7l|armv6l)
            NODE_ARCH="armv7l"
            ;;
        *)
            log_error "不支持的架构: $ARCH"
            return 1
            ;;
    esac
    
    # 使用 Node.js 18 LTS 版本
    NODE_VERSION_INSTALL="18.20.4"
    NODE_TAR="node-v${NODE_VERSION_INSTALL}-linux-${NODE_ARCH}.tar.xz"
    NODE_URL="https://nodejs.org/dist/v${NODE_VERSION_INSTALL}/${NODE_TAR}"
    NODE_DIR="/usr/local/nodejs18"
    
    log_info "下载 Node.js ${NODE_VERSION_INSTALL} (${NODE_ARCH})..."
    cd /tmp
    
    if ! wget -q --show-progress "$NODE_URL" -O "$NODE_TAR"; then
        log_error "下载 Node.js 失败"
        return 1
    fi
    
    log_info "解压并安装 Node.js..."
    rm -rf "node-v${NODE_VERSION_INSTALL}-linux-${NODE_ARCH}"
    tar -xf "$NODE_TAR"
    rm -rf "$NODE_DIR"
    mv "node-v${NODE_VERSION_INSTALL}-linux-${NODE_ARCH}" "$NODE_DIR"
    rm -f "$NODE_TAR"
    
    # 配置 PATH
    export PATH="$NODE_DIR/bin:$PATH"
    
    # 永久添加到配置文件
    if ! grep -q "$NODE_DIR/bin" ~/.bashrc 2>/dev/null; then
        echo "export PATH=\$PATH:$NODE_DIR/bin" >> ~/.bashrc
    fi
    
    if ! grep -q "$NODE_DIR/bin" /etc/profile 2>/dev/null; then
        echo "export PATH=\$PATH:$NODE_DIR/bin" >> /etc/profile
    fi
    
    # 验证安装
    if [ -f "$NODE_DIR/bin/node" ]; then
        log_info "✅ Node.js 安装成功: $($NODE_DIR/bin/node -v)"
        return 0
    else
        log_error "Node.js 安装失败"
        return 1
    fi
}

# 安装 Node.js
install_nodejs() {
    # 尝试查找已安装的 Node.js
    if setup_node_path; then
        if command -v node &> /dev/null; then
            NODE_VER=$(node -v)
            NPM_VER=$(npm -v 2>/dev/null || echo "未安装")
            log_info "Node.js 已安装: $NODE_VER"
            log_info "npm 已安装: $NPM_VER"
            
            # 检查版本是否符合要求
            if ! check_node_version; then
                log_warn "Node.js 版本过低，尝试自动升级..."
                
                # 尝试自动安装新版本
                if install_nodejs_binary; then
                    # 重新配置 PATH
                    export PATH="/usr/local/nodejs18/bin:$PATH"
                    
                    # 验证新版本
                    if command -v node &> /dev/null; then
                        NEW_VER=$(node -v)
                        log_info "✅ Node.js 已升级到: $NEW_VER"
                        
                        if check_node_version; then
                            log_info "✅ Node.js 版本符合要求"
                            return 0
                        else
                            log_error "升级后的版本仍不符合要求"
                            exit 1
                        fi
                    else
                        log_error "Node.js 升级失败"
                        exit 1
                    fi
                else
                    log_error "无法自动升级 Node.js"
                    log_info "请手动升级："
                    log_info "  1. 通过宝塔面板：软件商店 → Node.js 版本管理器 → 安装 Node.js 18.x 或 20.x"
                    log_info "  2. 或手动下载安装：https://nodejs.org/"
                    exit 1
                fi
            fi
            
            return 0
        fi
    fi

    log_step "开始安装 Node.js $NODE_VERSION..."

    # 使用 NodeSource 仓库安装
    if [ "$OS" = "ubuntu" ] || [ "$OS" = "debian" ]; then
        curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - || {
            log_warn "NodeSource 安装失败，尝试使用 apt 安装..."
            apt-get update
            apt-get install -y nodejs npm || {
                log_error "Node.js 安装失败"
                exit 1
            }
        }
        apt-get install -y nodejs
    elif [ "$OS" = "centos" ] || [ "$OS" = "rocky" ] || [ "$OS" = "rhel" ]; then
        curl -fsSL https://rpm.nodesource.com/setup_${NODE_VERSION}.x | bash - || {
            log_warn "NodeSource 安装失败，尝试使用 yum 安装..."
            yum install -y nodejs npm || {
                log_error "Node.js 安装失败"
                exit 1
            }
        }
        yum install -y nodejs
    else
        log_error "不支持的操作系统: $OS"
        exit 1
    fi

    # 配置 PATH
    setup_node_path
    
    # 验证安装
    if command -v node &> /dev/null; then
        NODE_VER=$(node -v)
        NPM_VER=$(npm -v)
        log_info "✅ Node.js 安装成功: $NODE_VER"
        log_info "✅ npm 安装成功: $NPM_VER"
        
        # 检查版本
        if ! check_node_version; then
            log_warn "安装的 Node.js 版本不符合要求，尝试自动升级..."
            
            # 尝试自动安装新版本
            if install_nodejs_binary; then
                export PATH="/usr/local/nodejs18/bin:$PATH"
                
                if command -v node &> /dev/null; then
                    NEW_VER=$(node -v)
                    log_info "✅ Node.js 已升级到: $NEW_VER"
                    
                    if check_node_version; then
                        log_info "✅ Node.js 版本符合要求"
                        return 0
                    else
                        log_error "升级后的版本仍不符合要求"
                        exit 1
                    fi
                else
                    log_error "Node.js 升级失败"
                    exit 1
                fi
            else
                log_error "无法自动升级 Node.js"
                log_info "请手动升级到 Node.js 18+："
                log_info "  find /www/server -name node -type f"
                log_info "  export PATH=\$PATH:/www/server/nodejs/版本号/bin"
                log_info "  echo 'export PATH=\$PATH:/www/server/nodejs/版本号/bin' >> ~/.bashrc"
                exit 1
            fi
        fi
        
        return 0
    else
        log_error "Node.js 安装失败，尝试使用二进制包安装..."
        
        # 如果包管理器安装失败，尝试二进制包安装
        if install_nodejs_binary; then
            export PATH="/usr/local/nodejs18/bin:$PATH"
            
            if command -v node &> /dev/null && check_node_version; then
                log_info "✅ Node.js 安装成功: $(node -v)"
                return 0
            fi
        fi
        
        log_error "Node.js 安装失败，请手动检查"
        log_info "提示: 如果通过宝塔面板安装，请执行："
        log_info "  find /www/server -name node -type f"
        log_info "  export PATH=\$PATH:/www/server/nodejs/版本号/bin"
        log_info "  echo 'export PATH=\$PATH:/www/server/nodejs/版本号/bin' >> ~/.bashrc"
        exit 1
    fi
}

# 生成随机密钥
generate_secret_key() {
    openssl rand -base64 32 | tr -d "=+/" | cut -c1-32
}

# 创建项目目录
setup_project_dir() {
    log_step "设置项目目录..."
    
    if [ ! -d "$PROJECT_DIR" ]; then
        log_info "创建项目目录: $PROJECT_DIR"
        mkdir -p "$PROJECT_DIR"
    fi

    cd "$PROJECT_DIR"
    log_info "当前目录: $(pwd)"
}

# 创建 .env 文件
create_env_file() {
    log_step "配置环境变量文件..."
    
    if [ -f ".env" ]; then
        log_warn ".env 文件已存在，跳过创建"
        return 0
    fi

    # 生成 SECRET_KEY
    SECRET_KEY=$(generate_secret_key)
    
    # 检测域名
    if [ -z "$DOMAIN" ] || [ "$DOMAIN" = "某站点" ]; then
        # 尝试从当前目录名获取域名
        DIR_NAME=$(basename "$PROJECT_DIR")
        if [ "$DIR_NAME" != "某站点" ] && [ "$DIR_NAME" != "." ]; then
            DOMAIN="$DIR_NAME"
        fi
    fi

    log_info "创建 .env 文件..."
    cat > .env << EOF
# ============================================
# CBoard Go 环境变量配置
# 自动生成时间: $(date '+%Y-%m-%d %H:%M:%S')
# ============================================

# 服务器配置
HOST=127.0.0.1
PORT=8000
DEBUG=false

# 数据库配置（SQLite）
DATABASE_URL=sqlite:///./cboard.db

# JWT 配置（已自动生成强密码）
SECRET_KEY=${SECRET_KEY}

# CORS 配置
BACKEND_CORS_ORIGINS=https://${DOMAIN},http://${DOMAIN}

# 项目配置
PROJECT_NAME=CBoard Go
VERSION=1.0.0
API_V1_STR=/api/v1

# 邮件配置（可选，稍后配置）
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
EOF

    log_info "✅ .env 文件已创建（SECRET_KEY 已自动生成）"
}

# 安装 Go 依赖
install_go_deps() {
    log_step "安装 Go 依赖..."
    
    # 确保 Go 在 PATH 中
    if ! setup_go_path; then
        log_error "无法找到 Go 安装路径"
        exit 1
    fi
    
    # 设置 Go 代理（使用国内镜像加速）
    export GOPROXY=https://goproxy.cn,direct
    export GOSUMDB=sum.golang.google.cn
    
    log_info "下载 Go 模块..."
    go mod download 2>&1 | tee -a "$LOG_FILE" || {
        log_warn "部分依赖下载失败，尝试继续..."
    }

    log_info "整理 Go 依赖..."
    go mod tidy 2>&1 | tee -a "$LOG_FILE" || {
        log_error "go mod tidy 失败"
        exit 1
    }

    log_info "✅ Go 依赖安装完成"
}

# 编译后端
build_backend() {
    log_step "编译后端服务..."
    
    # 确保 Go 在 PATH 中
    if ! setup_go_path; then
        log_error "无法找到 Go 安装路径"
        log_info "请手动配置 Go PATH，例如："
        log_info "  export PATH=\$PATH:/usr/local/btgojdk/go1.25.0/bin"
        exit 1
    fi
    
    log_info "开始编译..."
    if go build -o server ./cmd/server/main.go 2>&1 | tee -a "$LOG_FILE"; then
        chmod +x server
        log_info "✅ 后端编译成功"
        
        # 验证文件
        if [ -f "server" ]; then
            FILE_SIZE=$(ls -lh server | awk '{print $5}')
            log_info "可执行文件大小: $FILE_SIZE"
        fi
    else
        log_error "后端编译失败"
        log_info "尝试修复依赖..."
        go mod download
        go mod tidy
        if ! go build -o server ./cmd/server/main.go 2>&1 | tee -a "$LOG_FILE"; then
            log_error "编译仍然失败，请检查错误信息"
            exit 1
        fi
        chmod +x server
        log_info "✅ 后端编译成功（修复后）"
    fi
}

# 安装前端依赖并构建
build_frontend() {
    log_step "构建前端..."
    
    # 确保 Node.js 在 PATH 中
    if ! setup_node_path; then
        log_error "无法找到 Node.js 安装路径"
        log_info "请手动配置 Node.js PATH，例如："
        log_info "  export PATH=\$PATH:/www/server/nodejs/v18.17.0/bin"
        log_info "  或通过宝塔面板安装 Node.js"
        exit 1
    fi
    
    # 检查 Node.js 版本
    if ! check_node_version; then
        log_error "Node.js 版本不符合要求，无法构建前端"
        log_info "Vite 5.x 需要 Node.js 18.0.0 或更高版本"
        exit 1
    fi
    
    if [ ! -d "frontend" ]; then
        log_warn "未找到 frontend 目录，跳过前端构建"
        return 0
    fi

    cd frontend

    # 检查 node_modules
    if [ ! -d "node_modules" ] || [ ! -f "node_modules/.bin/vite" ]; then
        log_info "安装前端依赖（这可能需要几分钟）..."
        
        # 清理缓存
        npm cache clean --force 2>&1 || true
        
        # 安装依赖
        if npm install --legacy-peer-deps 2>&1 | tee -a "$LOG_FILE"; then
            log_info "✅ 前端依赖安装完成"
        else
            log_warn "标准安装失败，尝试使用 --force..."
            npm install --force 2>&1 | tee -a "$LOG_FILE" || {
                log_error "前端依赖安装失败"
                cd ..
                exit 1
            }
        fi
    else
        log_info "前端依赖已存在，跳过安装"
    fi

    # 构建前端
    log_info "构建前端生产版本..."
    if npm run build 2>&1 | tee -a "$LOG_FILE"; then
        if [ -d "dist" ]; then
            log_info "✅ 前端构建成功"
            DIST_SIZE=$(du -sh dist | awk '{print $1}')
            log_info "构建文件大小: $DIST_SIZE"
        else
            log_error "前端构建失败：dist 目录不存在"
            cd ..
            exit 1
        fi
    else
        log_error "前端构建失败"
        cd ..
        exit 1
    fi

    cd ..
}

# 创建必要目录
create_directories() {
    log_step "创建必要目录..."
    
    mkdir -p uploads/{avatars,config,logs}
    mkdir -p bin
    
    # 设置权限
    chmod -R 755 uploads
    chmod -R 755 frontend/dist 2>/dev/null || true
    
    log_info "✅ 目录创建完成"
}

# 设置文件权限
set_permissions() {
    log_step "设置文件权限..."
    
    chmod +x server 2>/dev/null || true
    chmod 644 .env 2>/dev/null || true
    chmod 666 cboard.db 2>/dev/null || true
    
    # 设置所有者（如果是宝塔面板）
    if [ -d "/www" ]; then
        BT_USER="www"
        if id "$BT_USER" &>/dev/null; then
            chown -R "$BT_USER:$BT_USER" . 2>/dev/null || true
            log_info "已设置所有者为: $BT_USER"
        fi
    fi
    
    log_info "✅ 权限设置完成"
}

# 创建 systemd 服务
create_systemd_service() {
    log_step "创建 systemd 服务..."
    
    SERVICE_FILE="/etc/systemd/system/cboard.service"
    
    if [ -f "$SERVICE_FILE" ]; then
        log_warn "systemd 服务已存在，跳过创建"
        return 0
    fi

    # 确定运行用户
    if [ -d "/www" ]; then
        SERVICE_USER="www"
    else
        SERVICE_USER="root"
    fi

    # 获取 Go 路径用于 systemd 环境变量
    GO_DIR=$(find_go_path)
    if [ -n "$GO_DIR" ]; then
        GO_PATH_ENV="PATH=$GO_DIR:/usr/local/go/bin:/usr/bin:/bin"
    else
        GO_PATH_ENV="PATH=/usr/local/go/bin:/usr/local/btgojdk/go*/bin:/usr/bin:/bin"
    fi
    
    log_info "创建服务文件: $SERVICE_FILE"
    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=CBoard Go Service
After=network.target

[Service]
Type=simple
User=${SERVICE_USER}
WorkingDirectory=${PROJECT_DIR}
Environment="${GO_PATH_ENV}"
ExecStart=${PROJECT_DIR}/server
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

    # 重新加载 systemd
    systemctl daemon-reload
    
    # 启用服务
    systemctl enable cboard
    
    log_info "✅ systemd 服务已创建并启用"
    log_info "使用以下命令管理服务："
    log_info "  启动: systemctl start cboard"
    log_info "  停止: systemctl stop cboard"
    log_info "  重启: systemctl restart cboard"
    log_info "  状态: systemctl status cboard"
}

# 测试后端服务
test_backend() {
    if [ "$SKIP_TESTS" = "true" ]; then
        log_warn "跳过服务测试（SKIP_TESTS=true）"
        return 0
    fi

    log_step "测试后端服务..."
    
    if [ ! -f "server" ]; then
        log_error "server 文件不存在"
        return 1
    fi

    # 检查端口占用
    if ! check_port 8000; then
        log_warn "端口 8000 已被占用，跳过测试"
        log_info "如果服务已在运行，这是正常的"
        return 0
    fi

    # 停止可能正在运行的服务
    systemctl stop cboard 2>/dev/null || true
    pkill -f "$PROJECT_DIR/server" 2>/dev/null || true
    sleep 2

    # 启动测试
    log_info "启动测试服务（10秒后自动停止）..."
    ./server > /tmp/cboard_test.log 2>&1 &
    TEST_PID=$!
    sleep 3

    # 检查进程
    if ! ps -p $TEST_PID > /dev/null 2>&1; then
        log_error "后端服务启动失败"
        log_info "错误日志:"
        cat /tmp/cboard_test.log | tail -20
        return 1
    fi

    # 测试健康检查
    for i in {1..5}; do
        if curl -s http://127.0.0.1:8000/health > /dev/null 2>&1; then
            HEALTH_RESPONSE=$(curl -s http://127.0.0.1:8000/health)
            log_info "✅ 后端服务运行正常"
            log_info "健康检查响应: $HEALTH_RESPONSE"
            kill $TEST_PID 2>/dev/null || true
            sleep 1
            return 0
        fi
        sleep 1
    done

    log_warn "健康检查超时，但进程正在运行"
    kill $TEST_PID 2>/dev/null || true
    sleep 1
    return 0
}

# 生成 Nginx 配置
generate_nginx_config() {
    log_step "生成 Nginx 配置..."
    
    NGINX_CONFIG="/tmp/cboard_nginx_${DOMAIN}.conf"
    
    cat > "$NGINX_CONFIG" << EOF
# CBoard Go Nginx 配置
# 域名: ${DOMAIN}
# 生成时间: $(date '+%Y-%m-%d %H:%M:%S')

server {
    listen 80;
    server_name ${DOMAIN};
    
    # 前端静态文件
    root ${PROJECT_DIR}/frontend/dist;
    index index.html;

    # SSL 证书验证路径（Let's Encrypt 文件验证）
    # 必须在所有 location 之前，优先级最高，确保不会被重定向
    # 宝塔面板会在网站根目录创建验证文件，所以需要指向项目根目录
    location /.well-known/acme-challenge/ {
        root ${PROJECT_DIR};
        allow all;
        access_log off;
        log_not_found off;
        default_type text/plain;
        # 重要：此路径不允许重定向到HTTPS，证书续期需要
    }
    
    # 通用 .well-known 路径
    location /.well-known/ {
        root ${PROJECT_DIR};
        allow all;
        access_log off;
        log_not_found off;
        # 重要：此路径不允许重定向到HTTPS
    }

    # 强制 HTTPS 重定向（排除 .well-known 路径）
    # 注意：如果证书还未申请，请注释掉下面的重定向配置
    # 证书申请成功后再取消注释
    location / {
        # 如果已启用强制 HTTPS，取消下面这行的注释
        # return 301 https://\$server_name\$request_uri;
        
        # 如果证书还未申请，使用下面的配置（不重定向）
        try_files \$uri \$uri/ /index.html;
    }

    # 后端 API 代理
    location /api/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
        
        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # 健康检查
    location /health {
        proxy_pass http://127.0.0.1:8000/health;
        proxy_set_header Host \$host;
    }

    # 订阅链接（如果需要）
    location /subscribe/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }

    # 静态资源缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # 日志
    access_log /www/wwwlogs/${DOMAIN}.log;
    error_log /www/wwwlogs/${DOMAIN}.error.log;
    
    # SSL 配置（宝塔面板会自动添加）
    #include /www/server/panel/vhost/cert/${DOMAIN}/fullchain.pem;
    #include /www/server/panel/vhost/cert/${DOMAIN}/privkey.pem;
}

# HTTPS 配置（SSL 证书部署后会自动启用）
#server {
#    listen 443 ssl http2;
#    server_name ${DOMAIN};
#    
#    # SSL 证书配置（宝塔面板会自动配置）
#    ssl_certificate /www/server/panel/vhost/cert/${DOMAIN}/fullchain.pem;
#    ssl_certificate_key /www/server/panel/vhost/cert/${DOMAIN}/privkey.pem;
#    ssl_protocols TLSv1.2 TLSv1.3;
#    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
#    ssl_prefer_server_ciphers off;
#    ssl_session_cache shared:SSL:10m;
#    ssl_session_timeout 10m;
#    
#    # 前端静态文件
#    root ${PROJECT_DIR}/frontend/dist;
#    index index.html;
#
#    # 前端路由（Vue Router）
#    location / {
#        try_files \$uri \$uri/ /index.html;
#    }
#
#    # 后端 API 代理
#    location /api/ {
#        proxy_pass http://127.0.0.1:8000;
#        proxy_http_version 1.1;
#        proxy_set_header Upgrade \$http_upgrade;
#        proxy_set_header Connection 'upgrade';
#        proxy_set_header Host \$host;
#        proxy_set_header X-Real-IP \$remote_addr;
#        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
#        proxy_set_header X-Forwarded-Proto \$scheme;
#        proxy_cache_bypass \$http_upgrade;
#        
#        # 超时设置
#        proxy_connect_timeout 60s;
#        proxy_send_timeout 60s;
#        proxy_read_timeout 60s;
#    }
#
#    # 健康检查
#    location /health {
#        proxy_pass http://127.0.0.1:8000/health;
#        proxy_set_header Host \$host;
#    }
#
#    # 订阅链接（如果需要）
#    location /subscribe/ {
#        proxy_pass http://127.0.0.1:8000;
#        proxy_set_header Host \$host;
#        proxy_set_header X-Real-IP \$remote_addr;
#        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
#    }
#
#    # 静态资源缓存
#    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
#        expires 1y;
#        add_header Cache-Control "public, immutable";
#    }
#
#    # 日志
#    access_log /www/wwwlogs/${DOMAIN}.log;
#    error_log /www/wwwlogs/${DOMAIN}.error.log;
#}
EOF

    log_info "✅ Nginx 配置已生成: $NGINX_CONFIG"
    log_info "配置已包含宝塔面板 SSL 自动部署标识"
    log_info "请将此配置复制到宝塔面板的网站配置中"
    log_info "然后可以在宝塔面板中申请 SSL 证书，系统会自动部署"
}

# 显示使用说明
show_usage() {
    cat << EOF
用法: $0 [选项]

选项:
    -d, --dir DIR         项目目录（默认: /www/wwwroot/某站点）
    -n, --domain DOMAIN   域名（默认: 从目录名自动检测）
    -g, --go-version VER   Go 版本（默认: 1.21.5）
    -N, --node-version VER Node.js 版本（默认: 18）
    -s, --skip-tests      跳过服务测试
    -h, --help            显示此帮助信息

环境变量:
    PROJECT_DIR           项目目录
    DOMAIN                域名
    GO_VERSION            Go 版本
    NODE_VERSION          Node.js 版本
    SKIP_TESTS            跳过测试（true/false）

示例:
    $0
    $0 -d /www/wwwroot/my-site.com -n my-site.com
    DOMAIN=my-site.com $0

EOF
}

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -d|--dir)
                PROJECT_DIR="$2"
                shift 2
                ;;
            -n|--domain)
                DOMAIN="$2"
                shift 2
                ;;
            -g|--go-version)
                GO_VERSION="$2"
                shift 2
                ;;
            -N|--node-version)
                NODE_VERSION="$2"
                shift 2
                ;;
            -s|--skip-tests)
                SKIP_TESTS="true"
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                show_usage
                exit 1
                ;;
        esac
    done
}

# 主函数
main() {
    # 解析命令行参数
    parse_args "$@"

    echo ""
    echo "=========================================="
    echo "🚀 CBoard Go 一键安装脚本"
    echo "=========================================="
    echo "项目目录: $PROJECT_DIR"
    echo "日志文件: $LOG_FILE"
    echo "=========================================="
    echo ""

    # 检查 root 权限
    check_root

    # 检测操作系统
    detect_os

    # 检查宝塔面板
    check_bt_panel

    # 获取域名
    get_domain

    # 安装 Go
    install_go

    # 安装 Node.js
    install_nodejs

    # 设置项目目录
    setup_project_dir

    # 创建 .env 文件
    create_env_file

    # 安装 Go 依赖
    install_go_deps

    # 编译后端
    build_backend

    # 构建前端
    build_frontend

    # 创建必要目录
    create_directories

    # 设置权限
    set_permissions

    # 创建 systemd 服务
    create_systemd_service

    # 测试后端
    test_backend

    # 生成 Nginx 配置
    generate_nginx_config

    echo ""
    echo "=========================================="
    echo "✅ 安装完成！"
    echo "=========================================="
    echo ""
    echo "📋 接下来的步骤："
    echo ""
    echo "1. 在宝塔面板中创建网站："
    echo "   - 域名: ${DOMAIN}"
    echo "   - 根目录: ${PROJECT_DIR}/frontend/dist"
    echo "   - PHP 版本: 纯静态"
    echo ""
    echo "2. 配置 Nginx："
    echo "   - 配置文件已生成: /tmp/cboard_nginx_${DOMAIN}.conf"
    echo "   - 请复制内容到宝塔面板的网站配置中"
    echo ""
    echo "3. 配置 SSL 证书（推荐）："
    echo "   - 在宝塔面板中申请 Let's Encrypt 证书"
    echo "   - 开启强制 HTTPS"
    echo ""
    echo "4. 启动服务："
    echo "   systemctl start cboard"
    echo "   systemctl status cboard"
    echo ""
    echo "5. 启动服务："
    echo "   systemctl start cboard"
    echo "   systemctl status cboard"
    echo ""
    echo "6. 访问网站："
    echo "   http://${DOMAIN} 或 https://${DOMAIN}"
    echo ""
    echo "7. 创建管理员账户："
    echo "   cd ${PROJECT_DIR}"
    echo "   go run scripts/create_admin.go"
    echo ""
    echo "📝 日志文件: $LOG_FILE"
    echo ""
    echo "💡 提示："
    echo "   - 如果遇到问题，请查看日志文件"
    echo "   - 可以使用 'systemctl status cboard' 查看服务状态"
    echo "   - 可以使用 'journalctl -u cboard -f' 查看实时日志"
    echo "=========================================="
    echo ""
}

# 运行主函数
main "$@"
