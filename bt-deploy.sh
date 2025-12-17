#!/bin/bash
# ============================================
# CBoard Go 一键安装脚本 - 宝塔面板版
# ============================================
# 功能：自动安装所需环境并完成网站部署
# 支持：Ubuntu/Debian/CentOS/Rocky Linux
# ============================================

set +e

# --- 颜色定义 ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# --- 配置变量 ---
PROJECT_DIR="${PROJECT_DIR:-/www/wwwroot/dy.moneyfly.top}"
DOMAIN="${DOMAIN:-}"
GO_VERSION="${GO_VERSION:-1.21.5}"
NODE_VERSION="${NODE_VERSION:-18}"
LOG_FILE="/tmp/cboard_install_$(date +%Y%m%d_%H%M%S).log"
SKIP_TESTS="${SKIP_TESTS:-false}"
BT_API_URL="${BT_API_URL:-http://132.226.1.44:30632}"
BT_API_KEY="${BT_API_KEY:-FBH2US6j6VtY0NhIVcMW0bQKKwREIivR}"
AUTO_SSL="${AUTO_SSL:-false}"
AUTO_PROXY="${AUTO_PROXY:-false}"
SSL_METHOD="${SSL_METHOD:-bt}"  # bt: 使用宝塔API, certbot: 直接使用certbot

# --- 日志函数 ---
log() { echo -e "${2}[${3}]${NC} $1" | tee -a "$LOG_FILE"; }
log_info() { log "$1" "$GREEN" "INFO"; }
log_warn() { log "$1" "$YELLOW" "WARN"; }
log_error() { log "$1" "$RED" "ERROR"; }
log_step() { log "$1" "$BLUE" "STEP"; }

# --- 基础检查与工具 ---
check_root() {
    [[ "$EUID" -ne 0 ]] && { log_error "请使用 root 用户运行: sudo $0"; exit 1; }
}

check_port() {
    local port=$1
    if command -v netstat &>/dev/null; then
        netstat -tuln | grep -q ":$port " && return 1
    elif command -v ss &>/dev/null; then
        ss -tuln | grep -q ":$port " && return 1
    fi
    return 0
}

detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID; OS_VERSION=$VERSION_ID
        log_info "检测到操作系统: $OS $OS_VERSION"
    else
        log_error "无法检测操作系统"; exit 1
    fi
}

check_bt_panel() {
    if [ -d "/www/server" ]; then
        log_info "✅ 检测到宝塔面板环境"
        return 0
    else
        log_warn "未检测到宝塔面板，使用标准 Linux 环境"
        return 1
    fi
}

persist_path() {
    local dir="$1"
    [[ -z "$dir" ]] && return
    export PATH="$PATH:$dir"
    for f in ~/.bashrc /etc/profile; do
        grep -q "$dir" "$f" 2>/dev/null || echo "export PATH=\$PATH:$dir" >> "$f"
    done
}

# --- Go 环境 ---
find_go_path() {
    if command -v go &>/dev/null; then dirname "$(which go)"; return 0; fi
    local bt_go; bt_go=$(find /usr/local/btgojdk -name "go" -type f 2>/dev/null | grep bin/go | head -1)
    [[ -n "$bt_go" ]] && { dirname "$bt_go"; return 0; }
    [[ -f "/usr/local/go/bin/go" ]] && { echo "/usr/local/go/bin"; return 0; }
    [[ -f "/usr/bin/go" ]] && { echo "/usr/bin"; return 0; }
    return 1
}

setup_go_env() {
    local go_dir; go_dir=$(find_go_path)
    if [[ -n "$go_dir" ]] && [[ -f "$go_dir/go" ]]; then
        persist_path "$go_dir"
        log_info "Go 环境已配置: $go_dir"
        return 0
    fi
    return 1
}

install_go() {
    setup_go_env && command -v go &>/dev/null && { log_info "Go 已安装: $(go version)"; return 0; }
    
    log_step "安装 Go $GO_VERSION..."
    local arch; arch=$(uname -m)
    case $arch in x86_64) arch="amd64";; aarch64|arm64) arch="arm64";; *) log_error "不支持架构: $arch"; exit 1;; esac
    
    local tar="go${GO_VERSION}.linux-${arch}.tar.gz"
    cd /tmp || exit
    wget -q --show-progress "https://go.dev/dl/${tar}" -O "$tar" || { log_error "下载 Go 失败"; exit 1; }
    
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "$tar" && rm -f "$tar"
    
    persist_path "/usr/local/go/bin"
    setup_go_env
    
    command -v go &>/dev/null && log_info "✅ Go 安装成功" || { log_error "Go 安装失败"; exit 1; }
}

# --- Node.js 环境 ---
find_node_path() {
    command -v node &>/dev/null && { dirname "$(which node)"; return 0; }
    [[ -f "/usr/local/nodejs18/bin/node" ]] && { echo "/usr/local/nodejs18/bin"; return 0; }
    
    local bt_node; bt_node=$(find /www/server/nodejs -name "node" -type f 2>/dev/null | grep -E "v(18|19|20|21|22)" | grep bin/node | head -1)
    [[ -n "$bt_node" ]] && { dirname "$bt_node"; return 0; }
    
    bt_node=$(find /usr/local/btnodejs -name "node" -type f 2>/dev/null | grep bin/node | head -1)
    [[ -n "$bt_node" ]] && { dirname "$bt_node"; return 0; }
    
    [[ -f "/usr/local/bin/node" ]] && { echo "/usr/local/bin"; return 0; }
    [[ -f "/usr/bin/node" ]] && { echo "/usr/bin"; return 0; }
    return 1
}

setup_node_env() {
    local node_dir; node_dir=$(find_node_path)
    if [[ -n "$node_dir" ]] && [[ -f "$node_dir/node" ]]; then
        persist_path "$node_dir"
        log_info "Node.js 环境已配置: $node_dir"
        return 0
    fi
    return 1
}

check_node_version() {
    command -v node &>/dev/null || return 1
    local ver; ver=$(node -v | sed 's/v//')
    [[ $(echo "$ver" | cut -d. -f1) -ge 18 ]] || { log_warn "Node.js 版本过低: v$ver (需 >= 18)"; return 1; }
    return 0
}

install_nodejs_binary() {
    log_step "安装 Node.js 18+ (二进制)..."
    local arch; arch=$(uname -m)
    local node_arch
    case $arch in x86_64) node_arch="x64";; aarch64|arm64) node_arch="arm64";; armv7l) node_arch="armv7l";; *) log_error "不支持架构"; return 1;; esac
    
    local ver="18.20.4"
    local tar="node-v${ver}-linux-${node_arch}.tar.xz"
    local dir="/usr/local/nodejs18"
    
    local cwd=$(pwd)
    cd /tmp || exit
    wget -q --show-progress "https://nodejs.org/dist/v${ver}/${tar}" -O "$tar" || { cd "$cwd"; return 1; }
    
    rm -rf "$dir" "node-v${ver}-linux-${node_arch}"
    tar -xf "$tar"
    mv "node-v${ver}-linux-${node_arch}" "$dir"
    rm -f "$tar"
    
    cd "$cwd"
    persist_path "$dir/bin"
    return 0
}

install_nodejs() {
    if setup_node_env && command -v node &>/dev/null; then
        check_node_version && { log_info "Node.js 已安装且版本符合要求"; return 0; }
        log_warn "尝试升级 Node.js..."
    fi

    if install_nodejs_binary; then
        setup_node_env
        check_node_version && { log_info "✅ Node.js 升级/安装成功"; return 0; }
    fi
    
    log_step "尝试使用包管理器安装 Node.js..."
    if [[ "$OS" == "ubuntu" ]] || [[ "$OS" == "debian" ]]; then
        curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash -
        apt-get install -y nodejs
    elif [[ "$OS" == "centos" ]] || [[ "$OS" == "rocky" ]]; then
        curl -fsSL https://rpm.nodesource.com/setup_${NODE_VERSION}.x | bash -
        yum install -y nodejs
    fi
    
    setup_node_env
    check_node_version && { log_info "✅ Node.js 安装成功"; return 0; }
    
    log_error "Node.js 安装失败，请手动安装 Node.js 18+"
    exit 1
}

# --- 项目设置 ---
setup_project_dir() {
    [[ ! -d "$PROJECT_DIR" ]] && mkdir -p "$PROJECT_DIR"
    cd "$PROJECT_DIR" || exit 1
    log_info "工作目录: $PROJECT_DIR"
}

get_domain() {
    [[ -n "$DOMAIN" ]] && { log_info "使用域名: $DOMAIN"; return; }
    local dir_name; dir_name=$(basename "$PROJECT_DIR")
    if [[ "$dir_name" != "." && "$dir_name" != "/" && "$dir_name" == *.* ]]; then
        DOMAIN="$dir_name"
        log_info "自动检测域名: $DOMAIN"
    else
        read -r -p "请输入域名 (如 example.com): " DOMAIN
        [[ -z "$DOMAIN" ]] && { log_error "域名不能为空"; exit 1; }
    fi
}

create_env_file() {
    [[ -f ".env" ]] && { log_warn ".env 已存在"; return 0; }
    log_step "生成 .env 文件..."
    local secret; secret=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)
    cat > .env << EOF
# Generated by install script
HOST=127.0.0.1
PORT=8000
DEBUG=false
DATABASE_URL=sqlite:///./cboard.db
SECRET_KEY=${secret}
BACKEND_CORS_ORIGINS=https://${DOMAIN},http://${DOMAIN}
PROJECT_NAME=CBoard Go
VERSION=1.0.0
API_V1_STR=/api/v1
UPLOAD_DIR=uploads
MAX_FILE_SIZE=10485760
DISABLE_SCHEDULE_TASKS=false
EOF
    log_info "✅ .env 创建完成"
}

create_directories() {
    mkdir -p uploads/{avatars,config,logs} bin
    chmod -R 755 uploads
    [[ -d "frontend/dist" ]] && chmod -R 755 frontend/dist
}

set_permissions() {
    log_step "设置权限..."
    chmod +x server 2>/dev/null
    chmod 644 .env 2>/dev/null
    chmod 666 cboard.db 2>/dev/null
    if [[ -d "/www" ]] && id "www" &>/dev/null; then
        chown -R "www:www" . 2>/dev/null
        log_info "所有者已设为 www"
    fi
}

# --- 构建流程 ---
install_go_deps() {
    cd "$PROJECT_DIR" || exit 1
    log_step "安装 Go 依赖..."
    setup_go_env || { log_error "Go 未找到"; exit 1; }
    export GOPROXY=https://goproxy.cn,direct
    export GOSUMDB=sum.golang.google.cn
    go mod download && go mod tidy || { log_error "依赖安装失败"; exit 1; }
    log_info "✅ Go 依赖完成"
}

build_backend() {
    log_step "编译后端..."
    setup_go_env
    go clean -cache 2>/dev/null
    
    # 优化：使用 nice 降低优先级，-p 1 限制并发数为 1，防止 CPU 爆满
    log_info "正在使用低资源模式编译 (防止 CPU 占用过高)..."
    
    if nice -n 19 go build -p 1 -o server ./cmd/server/main.go; then
        chmod +x server
        log_info "✅ 后端编译成功: $(ls -lh server | awk '{print $5}')"
    else
        log_warn "编译失败，尝试修复依赖..."
        go mod tidy
        nice -n 19 go build -p 1 -o server ./cmd/server/main.go || { log_error "后端编译最终失败"; exit 1; }
        chmod +x server
        log_info "✅ 后端编译成功 (修复后)"
    fi
}

init_database() {
    log_step "初始化数据库..."
    if [[ -f "cboard.db" ]]; then
        log_info "数据库已存在，跳过初始化"
        return 0
    fi

    setup_go_env
    
    local tmp_go="./init_db_temp.go"
    cat > "$tmp_go" << 'EOF'
package main
import ("fmt"; "log"; "cboard-go/internal/core/config"; "cboard-go/internal/core/database")
func main() {
    if _, err := config.LoadConfig(); err != nil { log.Fatalf("Config err: %v", err) }
    if err := database.InitDatabase(); err != nil { log.Fatalf("DB Init err: %v", err) }
    if err := database.AutoMigrate(); err != nil { log.Fatalf("Migrate err: %v", err) }
    fmt.Println("DB Init Success")
}
EOF
    if go run "$tmp_go"; then
        log_info "✅ 数据库初始化成功"
        rm -f "$tmp_go"
        create_admin_account
    else
        log_error "数据库初始化失败"
        rm -f "$tmp_go"
        exit 1
    fi
}

create_admin_account() {
    log_step "创建管理员..."
    setup_go_env
    local pwd; pwd=$(openssl rand -base64 16 | tr -d "=+/" | cut -c1-16)
    if ADMIN_PASSWORD="$pwd" go run scripts/create_admin.go; then
        log_info "✅ 管理员创建成功\n账号: admin / admin@example.com\n密码: $pwd"
        log_warn "⚠️  请立即登录修改密码！"
    else
        log_error "管理员创建失败"
    fi
}

build_frontend() {
    log_step "构建前端..."
    setup_node_env || { log_error "Node.js 未找到"; exit 1; }
    check_node_version || { log_error "Node.js 版本不足"; exit 1; }
    [[ ! -d "frontend" ]] && { log_warn "frontend 目录不存在，跳过"; return 0; }
    
    cd frontend || return
    rm -rf dist node_modules/.cache .vite .npm
    
    if [[ ! -d "node_modules" ]]; then
        log_info "安装前端依赖..."
        npm install --legacy-peer-deps || npm install --force || { log_error "npm install 失败"; cd ..; exit 1; }
    fi
    
    log_info "编译前端..."
    # 优化：使用 nice 降低优先级
    nice -n 19 npm run build || { log_error "npm run build 失败"; cd ..; exit 1; }
    [[ -d "dist" ]] && log_info "✅ 前端构建成功"
    cd ..
}

# --- 服务管理 ---
create_systemd_service() {
    log_step "配置 Systemd 服务..."
    local svc="/etc/systemd/system/cboard.service"
    [[ -f "$svc" ]] && { log_warn "服务文件已存在"; return 0; }
    
    local user="root"
    [[ -d "/www" ]] && user="www"
    
    local go_path; go_path=$(find_go_path)
    local env_path="PATH=$go_path:/usr/local/go/bin:/usr/bin:/bin"
    
    cat > "$svc" << EOF
[Unit]
Description=CBoard Go Service
After=network.target

[Service]
Type=simple
User=${user}
WorkingDirectory=${PROJECT_DIR}
Environment="${env_path}"
ExecStart=${PROJECT_DIR}/server
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable cboard
    log_info "✅ 服务已创建并启用"
}

manage_service() {
    local action=$1
    local force=$2
    
    case $action in
        start) systemctl start cboard ;;
        stop) systemctl stop cboard ;;
        restart)
            log_step "重启服务..."
            if [[ "$force" == "force" ]]; then
                log_info "强制杀死进程..."
                pkill -9 -f "$PROJECT_DIR/server" 2>/dev/null
                kill_port 8000
            fi
            systemctl restart cboard
            sleep 2
            if systemctl is-active --quiet cboard; then
                log_info "✅ 服务运行中"
                systemctl status cboard --no-pager -l | head -n 10
            else
                log_error "❌ 服务启动失败"
                journalctl -u cboard -n 20 --no-pager
            fi
            
            command -v nginx &>/dev/null && nginx -s reload 2>/dev/null
            ;;
        status)
            systemctl status cboard --no-pager -l
            check_port 8000 && log_warn "端口 8000 未占用" || log_info "端口 8000 正常"
            ;;
        logs)
            journalctl -u cboard -n 50 --no-pager
            ;;
    esac
}

kill_port() {
    local port=$1
    local pids
    if command -v lsof &>/dev/null; then pids=$(lsof -ti:$port); else pids=$(lsof -t -i:$port 2>/dev/null); fi
    [[ -z "$pids" ]] && command -v netstat &>/dev/null && pids=$(netstat -tlnp | grep ":$port " | awk '{print $7}' | cut -d'/' -f1)
    
    if [[ -n "$pids" ]]; then
        log_info "释放端口 $port (PID: $pids)..."
        kill -9 $pids 2>/dev/null
    fi
}

test_backend() {
    [[ "$SKIP_TESTS" == "true" ]] && return
    log_step "测试服务..."
    [[ ! -f "server" ]] && { log_error "server 文件丢失"; return; }
    ! check_port 8000 && { log_warn "端口占用，跳过测试"; return; }
    
    ./server > /tmp/test.log 2>&1 &
    local pid=$!
    sleep 5
    if curl -s http://127.0.0.1:8000/health >/dev/null; then
        log_info "✅ 健康检查通过"
    else
        log_error "❌ 服务响应失败"
        tail -n 10 /tmp/test.log
    fi
    kill $pid 2>/dev/null
}

# 检查网站配置是否正确（用于SSL和反向代理）
check_website_config() {
    local conf_file="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
    
    [[ ! -f "$conf_file" ]] && {
        log_warn "⚠️  网站配置文件不存在: $conf_file"
        log_warn "   请在宝塔面板中先创建网站"
        return 1
    }
    
    local errors=0
    local warnings=0
    
    log_step "检查网站配置: $conf_file"
    
    # 检查SSL标识
    if ! grep -q "#error_page 404/404.html;" "$conf_file" 2>/dev/null; then
        log_warn "⚠️  缺少SSL自动部署标识: #error_page 404/404.html;"
        log_warn "   这会导致SSL证书申请后无法自动部署"
        ((warnings++))
    else
        log_info "✅ SSL自动部署标识存在"
    fi
    
    # 检查扩展配置include
    if ! grep -q "include.*extension.*${DOMAIN}" "$conf_file" 2>/dev/null; then
        log_warn "⚠️  缺少扩展配置include语句"
        log_warn "   这会导致反向代理配置无法生效"
        log_warn "   需要添加: include /www/server/panel/vhost/nginx/extension/${DOMAIN}/*.conf;"
        ((warnings++))
    else
        log_info "✅ 扩展配置include存在"
    fi
    
    # 检查SSL验证路径
    if ! grep -q "location.*\.well-known/acme-challenge" "$conf_file" 2>/dev/null; then
        log_error "❌ 缺少SSL证书验证路径: location /.well-known/acme-challenge/"
        log_error "   这会导致SSL证书申请失败"
        log_error "   需要在配置文件中添加此location块，root指向: ${PROJECT_DIR}"
        ((errors++))
    else
        # 检查root是否正确指向项目根目录
        if grep -A 5 "location.*\.well-known/acme-challenge" "$conf_file" | grep -q "root.*${PROJECT_DIR}[^/]"; then
            log_info "✅ SSL验证路径配置正确（指向项目根目录）"
        else
            log_warn "⚠️  SSL验证路径的root可能不正确"
            log_warn "   应该指向项目根目录: ${PROJECT_DIR}"
            log_warn "   而不是: ${PROJECT_DIR}/frontend/dist"
            ((warnings++))
        fi
    fi
    
    # 检查nginx配置语法
    if command -v nginx &>/dev/null; then
        if nginx -t 2>/dev/null; then
            log_info "✅ Nginx配置语法正确"
        else
            log_error "❌ Nginx配置语法错误"
            log_error "   请运行: nginx -t 查看详细错误"
            ((errors++))
        fi
    fi
    
    if [[ $errors -gt 0 ]]; then
        log_error "❌ 发现 $errors 个错误，$warnings 个警告"
        return 1
    elif [[ $warnings -gt 0 ]]; then
        log_warn "⚠️  发现 $warnings 个警告，建议修复"
        return 0
    else
        log_info "✅ 配置检查通过"
        return 0
    fi
}

# 检查反向代理配置
check_proxy_config() {
    local ext_dir="/www/server/panel/vhost/nginx/extension/${DOMAIN}"
    local proxy_conf="${ext_dir}/proxy.conf"
    
    log_step "检查反向代理配置..."
    
    if [[ ! -d "$ext_dir" ]]; then
        log_warn "⚠️  扩展配置目录不存在: $ext_dir"
        log_warn "   反向代理配置可能未设置"
        log_info "   提示：反向代理会在宝塔面板设置后自动创建此目录"
        return 1
    fi
    
    if [[ ! -f "$proxy_conf" ]]; then
        log_warn "⚠️  反向代理配置文件不存在: $proxy_conf"
        log_warn "   请在宝塔面板中设置反向代理："
        log_info "   1. 网站 → 设置 → 反向代理"
        log_info "   2. 添加反向代理：目标URL: http://127.0.0.1:8000, 位置: /api/"
        return 1
    fi
    
    # 检查反向代理配置内容
    if grep -q "proxy_pass.*127.0.0.1:8000" "$proxy_conf" 2>/dev/null || \
       grep -q "proxy_pass.*localhost:8000" "$proxy_conf" 2>/dev/null; then
        log_info "✅ 反向代理配置存在且指向正确端口 (8000)"
        
        # 检查location路径
        if grep -q "location.*/api/" "$proxy_conf" 2>/dev/null; then
            log_info "✅ 反向代理路径配置正确 (/api/)"
        else
            log_warn "⚠️  反向代理路径可能不正确，应该包含: location /api/"
        fi
        
        return 0
    else
        log_warn "⚠️  反向代理配置可能不正确"
        log_warn "   应该指向: http://127.0.0.1:8000"
        log_warn "   当前配置内容："
        grep "proxy_pass" "$proxy_conf" 2>/dev/null | head -1 || log_warn "   未找到proxy_pass配置"
        return 1
    fi
}

# --- 宝塔面板API功能 ---
# 获取宝塔面板API密钥
get_bt_api_key() {
    if [[ -n "$BT_API_KEY" ]]; then
        log_info "使用环境变量中的宝塔API密钥"
        return 0
    fi
    
    # 尝试从宝塔面板配置文件中读取
    local bt_config="/www/server/panel/data/api.json"
    if [[ -f "$bt_config" ]]; then
        BT_API_KEY=$(grep -o '"key":"[^"]*' "$bt_config" 2>/dev/null | cut -d'"' -f4 | head -1)
        if [[ -n "$BT_API_KEY" ]]; then
            log_info "从宝塔面板配置文件中读取API密钥"
            return 0
        fi
    fi
    
    # 提示用户输入
    log_warn "未找到宝塔面板API密钥"
    log_info "获取方式："
    log_info "  1. 登录宝塔面板"
    log_info "  2. 面板设置 → API接口"
    log_info "  3. 开启API接口并复制API密钥"
    read -r -p "请输入宝塔面板API密钥（留空跳过自动配置）: " BT_API_KEY
    
    if [[ -z "$BT_API_KEY" ]]; then
        log_warn "未提供API密钥，将跳过自动配置"
        return 1
    fi
    
    return 0
}

# 调用宝塔面板API
bt_api_call() {
    local action="$1"
    local data="$2"
    
    [[ -z "$BT_API_KEY" ]] && { log_error "宝塔API密钥未设置"; return 1; }
    
    local url="${BT_API_URL}/$action"
    local timestamp=$(date +%s)
    
    # 计算token（兼容md5和md5sum）
    local token
    if command -v md5sum &>/dev/null; then
        token=$(echo -n "${BT_API_KEY}${timestamp}" | md5sum | cut -d' ' -f1)
    elif command -v md5 &>/dev/null; then
        token=$(echo -n "${BT_API_KEY}${timestamp}" | md5 | cut -d' ' -f1)
    else
        log_error "未找到md5或md5sum命令，无法计算API token"
        return 1
    fi
    
    local response
    local json_data
    if [[ -n "$data" ]]; then
        json_data="{\"request_token\":\"$token\",\"request_time\":$timestamp,$data}"
    else
        json_data="{\"request_token\":\"$token\",\"request_time\":$timestamp}"
    fi
    
    # 测试API连接
    local test_response
    test_response=$(curl -s -m 5 -X POST "$url" \
        -H "Content-Type: application/json" \
        -d "$json_data" 2>&1)
    local curl_exit=$?
    
    if [[ $curl_exit -ne 0 ]] || [[ -z "$test_response" ]]; then
        log_warn "⚠️  API调用失败 (curl退出码: $curl_exit)"
        log_warn "   API地址: $url"
        log_warn "   可能原因："
        log_warn "   1. 宝塔面板API接口未开启或地址不正确"
        log_warn "   2. 网络连接问题"
        log_warn "   3. API密钥错误"
        log_info "   将使用备用方法（直接操作配置文件）"
        return 1
    fi
    
    response="$test_response"
    
    # 检查响应
    if echo "$response" | grep -q '"status":true'; then
        echo "$response"
        return 0
    else
        # 尝试解析错误信息
        local error_msg=$(echo "$response" | grep -o '"msg":"[^"]*' | cut -d'"' -f4 || echo "未知错误")
        log_error "API调用失败: $error_msg"
        log_error "完整响应: $response"
        return 1
    fi
}

# 检查网站是否存在
check_website_exists() {
    log_step "检查网站是否存在..."
    
    # 方法1：检查nginx配置文件是否存在（最可靠）
    local nginx_conf="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
    if [[ -f "$nginx_conf" ]]; then
        log_info "✅ 网站配置文件存在: $nginx_conf"
        log_info "✅ 网站已存在: $DOMAIN"
        return 0
    fi
    
    # 方法2：尝试通过API检查
    if [[ -n "$BT_API_KEY" ]]; then
        log_info "通过API检查网站列表..."
        local response
        response=$(bt_api_call "site?action=get_site_list" "")
        
        if [[ $? -eq 0 ]]; then
            # 尝试多种匹配方式
            if echo "$response" | grep -q "\"${DOMAIN}\"" || \
               echo "$response" | grep -q "'${DOMAIN}'" || \
               echo "$response" | grep -q "${DOMAIN}"; then
                log_info "✅ 网站已存在: $DOMAIN (通过API确认)"
                return 0
            else
                log_warn "API返回的网站列表中没有找到: $DOMAIN"
                log_info "API响应预览: $(echo "$response" | head -c 200)"
            fi
        else
            log_warn "API调用失败，使用备用方法检查"
        fi
    fi
    
    # 方法3：检查其他可能的配置文件位置
    local alt_conf="/www/server/panel/vhost/nginx/${DOMAIN}_80.conf"
    if [[ -f "$alt_conf" ]]; then
        log_info "✅ 网站配置文件存在: $alt_conf"
        log_info "✅ 网站已存在: $DOMAIN"
        return 0
    fi
    
    # 如果所有方法都失败，尝试自动创建配置文件
    log_warn "⚠️  未找到网站配置文件: $nginx_conf"
    log_info "正在自动创建基础配置文件..."
    
    # 创建配置文件目录
    mkdir -p "$(dirname "$nginx_conf")"
    
    # 创建基础nginx配置
    cat > "$nginx_conf" << EOF
server {
    listen 80;
    server_name ${DOMAIN};
    
    root ${PROJECT_DIR}/frontend/dist;
    index index.html;
    
    # ⚠️ 重要：宝塔面板SSL自动部署标识（必须保留，不能删除！）
    #error_page 404/404.html;
    
    # ⚠️ 重要：包含宝塔面板的扩展配置（必须保留，不能注释！）
    include /www/server/panel/vhost/nginx/extension/${DOMAIN}/*.conf;

    # ⚠️ 重要：SSL证书验证路径（必须在最前面，优先级最高）
    location /.well-known/acme-challenge/ {
        root ${PROJECT_DIR};
        allow all;
        access_log off;
        log_not_found off;
        default_type text/plain;
    }
    
    # 通用 .well-known 路径（用于SSL验证）
    location /.well-known/ {
        root ${PROJECT_DIR};
        allow all;
        access_log off;
        log_not_found off;
        default_type text/plain;
    }

    # 前端路由（Vue Router）
    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # 禁止 index.html 缓存
    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        add_header Pragma "no-cache";
        add_header Expires "0";
        try_files \$uri /index.html;
    }

    # 静态资源缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
        access_log off;
    }

    # 日志
    access_log /www/wwwlogs/${DOMAIN}.log;
    error_log /www/wwwlogs/${DOMAIN}.error.log;
}
EOF
    
    # 测试配置
    if nginx -t 2>/dev/null; then
        nginx -s reload 2>/dev/null
        log_info "✅ 基础配置文件已创建: $nginx_conf"
        log_info "✅ 网站配置已就绪"
        return 0
    else
        log_error "❌ 配置文件创建失败，Nginx语法错误"
        rm -f "$nginx_conf"
        log_error "请先在宝塔面板中创建网站"
        return 1
    fi
}

# 自动申请SSL证书
auto_apply_ssl() {
    log_step "自动申请SSL证书..."
    
    # 检查网站是否存在（如果不存在会自动创建配置文件）
    if ! check_website_exists; then
        log_error "网站配置创建失败，无法申请SSL证书"
        return 1
    fi
    
    # 确保配置文件存在
    local nginx_conf="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
    if [[ ! -f "$nginx_conf" ]]; then
        log_error "网站配置文件不存在: $nginx_conf"
        return 1
    fi
    
    # 检查是否已有证书（检查多个可能的位置）
    local cert_file="/www/server/panel/vhost/cert/${DOMAIN}/fullchain.pem"
    local certbot_cert="/etc/letsencrypt/live/${DOMAIN}/fullchain.pem"
    
    if [[ -f "$cert_file" ]] || [[ -f "$certbot_cert" ]]; then
        log_info "✅ SSL证书已存在"
        if [[ -f "$cert_file" ]]; then
            log_info "   证书位置: $cert_file"
        else
            log_info "   证书位置: $certbot_cert"
        fi
        read -r -p "是否重新申请证书？(yes/no，默认no): " reapply
        reapply=${reapply:-no}
        if [[ "$reapply" != "yes" ]]; then
            log_info "跳过证书申请，使用现有证书"
            return 0
        fi
    fi
    
    log_info "正在申请Let's Encrypt SSL证书（文件验证）..."
    
    local response
    response=$(bt_api_call "site?action=apply_cert" "\"domain\":\"${DOMAIN}\",\"type\":\"lets\",\"auth_type\":\"http\"")
    
    if [[ $? -eq 0 ]]; then
        log_info "✅ SSL证书申请已提交"
        log_info "   等待证书申请完成（通常需要1-2分钟）..."
        
        # 等待证书申请完成
        local max_wait=120
        local waited=0
        while [[ $waited -lt $max_wait ]]; do
            sleep 5
            waited=$((waited + 5))
            
            if [[ -f "$cert_file" ]]; then
                log_info "✅ SSL证书申请成功！"
                return 0
            fi
            
            log_info "   等待中... (${waited}/${max_wait}秒)"
        done
        
        if [[ -f "$cert_file" ]]; then
            log_info "✅ SSL证书申请成功！"
            return 0
        else
            log_warn "⚠️  SSL证书申请可能还在进行中，请稍后检查"
            log_warn "   证书文件路径: $cert_file"
            return 1
        fi
    else
        log_error "❌ SSL证书申请失败"
        return 1
    fi
}

# 自动设置反向代理
auto_setup_proxy() {
    log_step "自动设置反向代理..."
    
    # 检查网站是否存在（如果不存在会自动创建）
    if ! check_website_exists; then
        log_error "网站配置创建失败，无法设置反向代理"
        return 1
    fi
    
    # 确保配置文件存在
    local nginx_conf="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
    if [[ ! -f "$nginx_conf" ]]; then
        log_error "网站配置文件不存在: $nginx_conf"
        return 1
    fi
    
    log_info "正在设置反向代理: /api/ -> http://127.0.0.1:8000"
    
    local response
    response=$(bt_api_call "site?action=CreateProxy" "\"sitename\":\"${DOMAIN}\",\"cache\":0,\"proxyname\":\"api\",\"proxydir\":\"/api/\",\"proxydomain\":\"http://127.0.0.1:8000\",\"advanced\":0,\"savename\":\"api\",\"subdomain\":\"\",\"todomain\":\"\",\"type\":0")
    
    if [[ $? -eq 0 ]]; then
        log_info "✅ 反向代理设置成功！"
        
        # 验证配置
        sleep 2
        if check_proxy_config; then
            log_info "✅ 反向代理配置验证通过"
            return 0
        else
            log_warn "⚠️  反向代理已设置，但配置验证失败，请手动检查"
            return 1
        fi
    else
        log_error "❌ 反向代理设置失败"
        log_warn "   请手动在宝塔面板中设置反向代理"
        return 1
    fi
}

# --- 方案二：直接使用certbot申请证书（不使用宝塔API）---
# 安装certbot
install_certbot() {
    if command -v certbot &>/dev/null; then
        log_info "✅ certbot 已安装: $(certbot --version 2>&1 | head -1)"
        return 0
    fi
    
    log_step "安装 certbot..."
    
    local install_success=false
    
    if [[ "$OS" == "ubuntu" ]] || [[ "$OS" == "debian" ]]; then
        # 方法1：使用apt安装
        log_info "尝试使用apt安装certbot..."
        if apt-get update -qq && apt-get install -y certbot python3-certbot-nginx 2>&1; then
            if command -v certbot &>/dev/null; then
                install_success=true
            fi
        fi
        
        # 方法2：如果apt失败，尝试使用snap
        if [[ "$install_success" == false ]] && command -v snap &>/dev/null; then
            log_info "尝试使用snap安装certbot..."
            if snap install --classic certbot 2>&1; then
                if [[ -f /snap/bin/certbot ]]; then
                    ln -sf /snap/bin/certbot /usr/bin/certbot 2>/dev/null
                    if command -v certbot &>/dev/null; then
                        install_success=true
                    fi
                fi
            fi
        fi
        
        # 方法3：使用pip安装（如果前两种方法都失败）
        if [[ "$install_success" == false ]] && command -v pip3 &>/dev/null; then
            log_info "尝试使用pip3安装certbot..."
            if pip3 install certbot 2>&1; then
                if command -v certbot &>/dev/null; then
                    install_success=true
                fi
            fi
        fi
        
        # 方法4：直接下载certbot-auto（已弃用但可能有用）
        if [[ "$install_success" == false ]]; then
            log_info "尝试下载certbot-auto..."
            if wget -q https://dl.eff.org/certbot-auto -O /usr/local/bin/certbot-auto 2>/dev/null; then
                chmod +x /usr/local/bin/certbot-auto
                ln -sf /usr/local/bin/certbot-auto /usr/bin/certbot 2>/dev/null
                if command -v certbot &>/dev/null; then
                    install_success=true
                fi
            fi
        fi
        
    elif [[ "$OS" == "centos" ]] || [[ "$OS" == "rocky" ]]; then
        # CentOS/Rocky Linux安装方法
        log_info "尝试安装epel-release..."
        yum install -y epel-release 2>&1
        
        log_info "尝试使用yum安装certbot..."
        if yum install -y certbot python3-certbot-nginx 2>&1; then
            if command -v certbot &>/dev/null; then
                install_success=true
            fi
        fi
        
        # 如果yum失败，尝试使用pip
        if [[ "$install_success" == false ]] && command -v pip3 &>/dev/null; then
            log_info "尝试使用pip3安装certbot..."
            if pip3 install certbot 2>&1; then
                if command -v certbot &>/dev/null; then
                    install_success=true
                fi
            fi
        fi
    fi
    
    if [[ "$install_success" == true ]] && command -v certbot &>/dev/null; then
        log_info "✅ certbot 安装成功: $(certbot --version 2>&1 | head -1)"
        return 0
    else
        log_error "❌ certbot 安装失败"
        log_error ""
        log_error "请手动安装certbot，方法如下："
        log_error ""
        if [[ "$OS" == "ubuntu" ]] || [[ "$OS" == "debian" ]]; then
            log_error "Ubuntu/Debian:"
            log_error "  sudo apt-get update"
            log_error "  sudo apt-get install -y certbot python3-certbot-nginx"
            log_error ""
            log_error "或使用snap:"
            log_error "  sudo snap install --classic certbot"
        elif [[ "$OS" == "centos" ]] || [[ "$OS" == "rocky" ]]; then
            log_error "CentOS/Rocky Linux:"
            log_error "  sudo yum install -y epel-release"
            log_error "  sudo yum install -y certbot python3-certbot-nginx"
        fi
        log_error ""
        log_error "安装完成后，可以："
        log_error "  1. 重新运行此脚本"
        log_error "  2. 或使用宝塔面板API方式（选项1）申请SSL证书"
        return 1
    fi
}

# 停止nginx（申请证书需要80端口）
stop_nginx_for_ssl() {
    log_step "停止Nginx以释放80端口..."
    
    if command -v systemctl &>/dev/null; then
        if systemctl is-active --quiet nginx 2>/dev/null; then
            systemctl stop nginx
            log_info "✅ Nginx已停止"
            return 0
        fi
    elif command -v service &>/dev/null; then
        if service nginx status &>/dev/null; then
            service nginx stop
            log_info "✅ Nginx已停止"
            return 0
        fi
    fi
    
    # 尝试直接kill进程
    if pgrep -x nginx &>/dev/null; then
        pkill -9 nginx
        sleep 1
        log_info "✅ Nginx进程已终止"
        return 0
    fi
    
    log_info "Nginx未运行，80端口可用"
    return 0
}

# 启动nginx
start_nginx() {
    log_step "启动Nginx..."
    
    if command -v systemctl &>/dev/null; then
        systemctl start nginx
        systemctl enable nginx 2>/dev/null
    elif command -v service &>/dev/null; then
        service nginx start
    else
        # 尝试直接启动
        if command -v nginx &>/dev/null; then
            nginx
        fi
    fi
    
    sleep 2
    if pgrep -x nginx &>/dev/null; then
        log_info "✅ Nginx已启动"
        return 0
    else
        log_warn "⚠️  Nginx启动失败，请手动启动"
        return 1
    fi
}

# 检查SSL证书状态
check_ssl_status() {
    local cert_dir="/etc/letsencrypt/live/${DOMAIN}"
    local bt_cert_dir="/www/server/panel/vhost/cert/${DOMAIN}"
    
    # 检查certbot证书
    if [[ -f "${cert_dir}/fullchain.pem" ]] && [[ -f "${cert_dir}/privkey.pem" ]]; then
        log_info "✅ 发现certbot证书: ${cert_dir}"
        return 0
    fi
    
    # 检查宝塔证书
    if [[ -f "${bt_cert_dir}/fullchain.pem" ]] && [[ -f "${bt_cert_dir}/privkey.pem" ]]; then
        log_info "✅ 发现宝塔证书: ${bt_cert_dir}"
        return 0
    fi
    
    return 1
}

# 直接使用certbot申请证书
certbot_apply_ssl() {
    log_step "使用certbot申请SSL证书..."
    
    # 检查是否已有证书
    if check_ssl_status; then
        log_info "✅ SSL证书已存在"
        read -r -p "是否重新申请证书？(yes/no，默认no): " reapply
        reapply=${reapply:-no}
        if [[ "$reapply" != "yes" ]]; then
            log_info "跳过证书申请，使用现有证书"
            # 确保nginx配置了SSL
            certbot_configure_nginx_ssl
            return 0
        fi
    fi
    
    # 安装certbot
    if ! install_certbot; then
        log_error "无法安装certbot"
        log_info ""
        log_info "您可以选择："
        log_info "  1. 手动安装certbot后重新运行此脚本"
        log_info "  2. 使用宝塔面板API方式（选项1）申请SSL证书"
        log_info "  3. 跳过SSL证书申请，只配置反向代理"
        log_info ""
        read -r -p "是否跳过SSL证书申请，只配置反向代理？(yes/no，默认no): " skip_ssl
        skip_ssl=${skip_ssl:-no}
        if [[ "$skip_ssl" == "yes" ]]; then
            log_info "跳过SSL证书申请，继续配置反向代理"
            return 2  # 返回特殊代码，表示跳过但不失败
        else
            return 1
        fi
    fi
    
    # 检查网站配置文件是否存在
    local nginx_conf="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
    if [[ ! -f "$nginx_conf" ]]; then
        log_warn "⚠️  网站配置文件不存在: $nginx_conf"
        log_info "正在创建基础配置文件..."
        
        # 创建基础nginx配置
        mkdir -p "$(dirname "$nginx_conf")"
        cat > "$nginx_conf" << EOF
server {
    listen 80;
    server_name ${DOMAIN};
    
    root ${PROJECT_DIR}/frontend/dist;
    index index.html;
    
    # SSL证书验证路径
    location /.well-known/acme-challenge/ {
        root ${PROJECT_DIR};
        allow all;
        access_log off;
        log_not_found off;
        default_type text/plain;
    }
    
    location /.well-known/ {
        root ${PROJECT_DIR};
        allow all;
        access_log off;
        log_not_found off;
        default_type text/plain;
    }
    
    location / {
        try_files \$uri \$uri/ /index.html;
    }
    
    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        add_header Pragma "no-cache";
        add_header Expires "0";
        try_files \$uri /index.html;
    }
    
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
        access_log off;
    }
    
    access_log /www/wwwlogs/${DOMAIN}.log;
    error_log /www/wwwlogs/${DOMAIN}.error.log;
}
EOF
        
        # 测试配置
        if nginx -t 2>/dev/null; then
            nginx -s reload 2>/dev/null
            log_info "✅ 基础配置文件已创建"
        else
            log_error "❌ 配置文件创建失败，请手动创建网站"
            rm -f "$nginx_conf"
            return 1
        fi
    else
        log_info "✅ 网站配置文件已存在"
    fi
    
    # 确保配置文件中包含SSL验证路径
    if ! grep -q "location.*\.well-known/acme-challenge" "$nginx_conf" 2>/dev/null; then
        log_warn "配置文件中缺少SSL验证路径，正在添加..."
        # 在server块中添加验证路径（在location /之前）
        local temp_conf="${nginx_conf}.tmp"
        local in_server=false
        local added=false
        
        while IFS= read -r line || [[ -n "$line" ]]; do
            if [[ "$line" =~ ^[[:space:]]*server[[:space:]]*\{ ]]; then
                in_server=true
                echo "$line" >> "$temp_conf"
            elif [[ "$in_server" == true ]] && [[ "$line" =~ ^[[:space:]]*location[[:space:]]+/[[:space:]]*\{ ]] && [[ "$added" == false ]]; then
                # 在location /之前添加验证路径
                cat >> "$temp_conf" << 'EOF'
    # SSL证书验证路径
    location /.well-known/acme-challenge/ {
        root /www/wwwroot;
        allow all;
        access_log off;
        log_not_found off;
        default_type text/plain;
    }
EOF
                added=true
                echo "$line" >> "$temp_conf"
            else
                echo "$line" >> "$temp_conf"
            fi
        done < "$nginx_conf"
        
        if [[ "$added" == true ]]; then
            mv "$temp_conf" "$nginx_conf"
            log_info "✅ 已添加SSL验证路径配置"
        else
            rm -f "$temp_conf"
            log_warn "⚠️  无法自动添加验证路径，请手动添加"
        fi
    fi
    
    # 停止nginx释放80端口
    stop_nginx_for_ssl
    
    # 申请证书
    log_info "正在申请Let's Encrypt证书..."
    log_info "域名: $DOMAIN"
    log_info "验证方式: standalone（需要临时停止nginx）"
    
    # 使用standalone方式申请（更可靠）
    if certbot certonly --standalone \
        -d "$DOMAIN" \
        --non-interactive \
        --agree-tos \
        --email "admin@${DOMAIN}" \
        --preferred-challenges http \
        --quiet 2>&1; then
        log_info "✅ SSL证书申请成功！"
        
        # 启动nginx
        start_nginx
        
        # 配置nginx使用SSL证书
        certbot_configure_nginx_ssl
        
        return 0
    else
        log_error "❌ SSL证书申请失败"
        log_info "尝试使用webroot方式..."
        
        # 如果standalone失败，尝试webroot方式
        # 确保验证目录存在
        mkdir -p "${PROJECT_DIR}/.well-known/acme-challenge"
        chmod -R 755 "${PROJECT_DIR}/.well-known"
        
        # 启动nginx（webroot方式需要nginx运行）
        start_nginx
        sleep 2
        
        if certbot certonly --webroot \
            -w "${PROJECT_DIR}" \
            -d "$DOMAIN" \
            --non-interactive \
            --agree-tos \
            --email "admin@${DOMAIN}" \
            --quiet 2>&1; then
            log_info "✅ SSL证书申请成功（webroot方式）！"
            certbot_configure_nginx_ssl
            return 0
        else
            log_error "❌ SSL证书申请失败（两种方式都失败）"
            log_error "请检查："
            log_error "  1. 域名是否正确解析到服务器"
            log_error "  2. 80端口是否可访问"
            log_error "  3. 防火墙是否开放80端口"
            return 1
        fi
    fi
}

# 配置nginx使用SSL证书
certbot_configure_nginx_ssl() {
    log_step "配置Nginx使用SSL证书..."
    
    local nginx_conf="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
    if [[ ! -f "$nginx_conf" ]]; then
        log_error "网站配置文件不存在"
        return 1
    fi
    
    # 检查是否已配置SSL
    if grep -q "listen 443 ssl" "$nginx_conf" 2>/dev/null; then
        log_info "✅ HTTPS server块已存在"
        
        # 检查证书路径是否正确
        local cert_in_use
        if grep -q "ssl_certificate.*letsencrypt" "$nginx_conf" 2>/dev/null || \
           grep -q "ssl_certificate.*/etc/letsencrypt" "$nginx_conf" 2>/dev/null; then
            log_info "✅ 证书路径配置正确（certbot证书）"
            return 0
        elif grep -q "ssl_certificate" "$nginx_conf" 2>/dev/null; then
            log_warn "⚠️  检测到其他证书配置，可能需要更新证书路径"
            read -r -p "是否更新为certbot证书？(yes/no，默认no): " update_cert
            update_cert=${update_cert:-no}
            if [[ "$update_cert" != "yes" ]]; then
                return 0
            fi
        else
            log_warn "⚠️  HTTPS配置存在但证书路径未找到，将更新"
        fi
    fi
    
    # 添加HTTPS server块
    local cert_dir="/etc/letsencrypt/live/${DOMAIN}"
    local cert_file="${cert_dir}/fullchain.pem"
    local key_file="${cert_dir}/privkey.pem"
    
    if [[ ! -f "$cert_file" ]] || [[ ! -f "$key_file" ]]; then
        log_error "证书文件不存在"
        return 1
    fi
    
    # 更准确地检测HTTP server块的结束位置
    # 使用括号计数来准确找到server块的结束
    local temp_conf="${nginx_conf}.ssl.tmp"
    local brace_count=0
    local in_http_server=false
    local http_server_start_line=0
    local http_block_done=false
    
    # 首先找到HTTP server块的开始和结束行号
    local line_num=0
    local http_start=0
    local http_end=0
    
    while IFS= read -r line || [[ -n "$line" ]]; do
        line_num=$((line_num + 1))
        
        # 检测HTTP server块开始（listen 80且没有443）
        if [[ "$line" =~ ^[[:space:]]*server[[:space:]]*\{ ]] && [[ $http_start -eq 0 ]]; then
            # 检查接下来的几行是否有listen 80
            local check_lines
            check_lines=$(sed -n "${line_num},$((line_num + 5))p" "$nginx_conf" 2>/dev/null)
            if echo "$check_lines" | grep -q "listen[[:space:]]\+80[^0-9]" && \
               ! echo "$check_lines" | grep -q "listen[[:space:]]\+443"; then
                http_start=$line_num
                brace_count=1
            fi
        fi
        
        # 如果在HTTP server块内，计算括号
        if [[ $http_start -gt 0 ]] && [[ $http_end -eq 0 ]]; then
            # 计算开括号和闭括号
            local open_braces
            local close_braces
            open_braces=$(echo "$line" | grep -o '{' | wc -l)
            close_braces=$(echo "$line" | grep -o '}' | wc -l)
            brace_count=$((brace_count + open_braces - close_braces))
            
            # 如果括号计数归零，说明server块结束
            if [[ $brace_count -eq 0 ]]; then
                http_end=$line_num
                break
            fi
        fi
    done < "$nginx_conf"
    
    # 如果找到了HTTP server块，在它之后添加HTTPS server块
    if [[ $http_start -gt 0 ]] && [[ $http_end -gt 0 ]]; then
        # 复制HTTP server块之前的内容
        if [[ $http_start -gt 1 ]]; then
            sed -n "1,$((http_start - 1))p" "$nginx_conf" > "$temp_conf" 2>/dev/null
        else
            > "$temp_conf"
        fi
        
        # 复制HTTP server块内容
        sed -n "${http_start},${http_end}p" "$nginx_conf" >> "$temp_conf" 2>/dev/null
        
        # 添加HTTP到HTTPS重定向（在HTTP server块内，在第一个location之前）
        if ! grep -q "return 301 https" "$temp_conf" 2>/dev/null && \
           ! grep -q "if.*request_uri.*well-known" "$temp_conf" 2>/dev/null; then
            # 使用sed在server_name之后插入重定向
            local temp_redirect="${temp_conf}.redirect"
            if sed "/server_name.*${DOMAIN}/a\\
    # HTTP到HTTPS重定向（保留.well-known用于证书续期）\\
    if (\$request_uri !~* ^/.well-known/) {\\
        return 301 https://\$server_name\$request_uri;\\
    }" "$temp_conf" > "$temp_redirect" 2>/dev/null; then
                mv "$temp_redirect" "$temp_conf"
                log_info "✅ 已添加HTTP到HTTPS重定向"
            else
                rm -f "$temp_redirect"
                log_warn "⚠️  无法自动添加HTTP到HTTPS重定向，将在后续步骤中处理"
            fi
        fi
        
        # 在HTTP server块之后添加HTTPS server块
        cat >> "$temp_conf" << EOF

# HTTPS server block - 自动生成
server {
    listen 443 ssl http2;
    server_name ${DOMAIN};
    
    ssl_certificate ${cert_file};
    ssl_certificate_key ${key_file};
    
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE:ECDH:AES:HIGH:!NULL:!aNULL:!MD5:!ADH:!RC4;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    root ${PROJECT_DIR}/frontend/dist;
    index index.html;
    
    include /www/server/panel/vhost/nginx/extension/${DOMAIN}/*.conf;
    
    location /.well-known/acme-challenge/ {
        root ${PROJECT_DIR};
        allow all;
        access_log off;
        log_not_found off;
        default_type text/plain;
    }
    
    location / {
        try_files \$uri \$uri/ /index.html;
    }
    
    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        add_header Pragma "no-cache";
        add_header Expires "0";
        try_files \$uri /index.html;
    }
    
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
        access_log off;
    }
    
    access_log /www/wwwlogs/${DOMAIN}.log;
    error_log /www/wwwlogs/${DOMAIN}.error.log;
}
EOF
        
        # 复制HTTP server块之后的内容（如果有）
        local total_lines
        total_lines=$(wc -l < "$nginx_conf" 2>/dev/null || echo "0")
        if [[ $http_end -lt $total_lines ]]; then
            sed -n "$((http_end + 1)),\$p" "$nginx_conf" >> "$temp_conf" 2>/dev/null
        fi
        
        http_block_done=true
    else
        # 如果无法准确检测，使用备用方法：在文件末尾添加
        log_warn "⚠️  无法准确检测HTTP server块位置，将在文件末尾添加HTTPS配置"
        cp "$nginx_conf" "$temp_conf"
        cat >> "$temp_conf" << EOF

# HTTPS server block - 自动生成
server {
    listen 443 ssl http2;
    server_name ${DOMAIN};
    
    ssl_certificate ${cert_file};
    ssl_certificate_key ${key_file};
    
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE:ECDH:AES:HIGH:!NULL:!aNULL:!MD5:!ADH:!RC4;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    root ${PROJECT_DIR}/frontend/dist;
    index index.html;
    
    include /www/server/panel/vhost/nginx/extension/${DOMAIN}/*.conf;
    
    location /.well-known/acme-challenge/ {
        root ${PROJECT_DIR};
        allow all;
        access_log off;
        log_not_found off;
        default_type text/plain;
    }
    
    location / {
        try_files \$uri \$uri/ /index.html;
    }
    
    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        add_header Pragma "no-cache";
        add_header Expires "0";
        try_files \$uri /index.html;
    }
    
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
        access_log off;
    }
    
    access_log /www/wwwlogs/${DOMAIN}.log;
    error_log /www/wwwlogs/${DOMAIN}.error.log;
}
EOF
        http_block_done=true
    fi
    
    # 检查重定向是否已添加（如果还没有，在HTTP server块中添加）
    if [[ "$http_block_done" == true ]] && ! grep -q "return 301 https" "$temp_conf" 2>/dev/null && ! grep -q "if.*request_uri.*well-known" "$temp_conf" 2>/dev/null; then
        # 在HTTP server块中查找server_name并插入重定向
        local temp_redirect="${temp_conf}.redirect"
        if sed "/listen[[:space:]]\+80[^0-9]/,/^[[:space:]]*}/ {
            /server_name.*${DOMAIN}/a\\
    # HTTP到HTTPS重定向（保留.well-known用于证书续期）\\
    if (\$request_uri !~* ^/.well-known/) {\\
        return 301 https://\$server_name\$request_uri;\\
    }
}" "$temp_conf" > "$temp_redirect" 2>/dev/null; then
            mv "$temp_redirect" "$temp_conf"
            log_info "✅ 已添加HTTP到HTTPS重定向（备用方法）"
        else
            rm -f "$temp_redirect"
            log_warn "⚠️  无法自动添加HTTP到HTTPS重定向，请手动配置"
        fi
    elif grep -q "return 301 https" "$temp_conf" 2>/dev/null || grep -q "if.*request_uri.*well-known" "$temp_conf" 2>/dev/null; then
        log_info "✅ HTTP到HTTPS重定向已存在"
    fi
    
    # 备份原配置并应用新配置
    local backup_file="${nginx_conf}.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$nginx_conf" "$backup_file"
    mv "$temp_conf" "$nginx_conf"
    
    # 测试配置
    log_info "验证Nginx配置..."
    if nginx -t 2>/dev/null; then
        log_info "✅ Nginx配置语法正确"
        nginx -s reload 2>/dev/null
        sleep 2
        
        # 验证HTTPS是否可访问
        log_info "验证HTTPS访问..."
        if curl -k -s -o /dev/null -w "%{http_code}" "https://${DOMAIN}" | grep -q "200\|301\|302"; then
            log_info "✅ HTTPS访问正常"
        else
            log_warn "⚠️  HTTPS访问测试失败，但配置已应用"
            log_warn "   请手动访问 https://${DOMAIN} 验证"
        fi
        
        # 验证HTTP重定向
        log_info "验证HTTP到HTTPS重定向..."
        local redirect_code
        redirect_code=$(curl -s -o /dev/null -w "%{http_code}" "http://${DOMAIN}" 2>/dev/null)
        if [[ "$redirect_code" == "301" ]] || [[ "$redirect_code" == "302" ]]; then
            log_info "✅ HTTP到HTTPS重定向正常 (HTTP $redirect_code)"
        else
            log_warn "⚠️  HTTP重定向可能未生效 (HTTP $redirect_code)"
        fi
        
        log_info "✅ SSL配置已应用并验证"
        return 0
    else
        log_error "❌ Nginx配置错误，已恢复备份"
        mv "$backup_file" "$nginx_conf"
        nginx -t 2>&1 | head -10
        return 1
    fi
}

# 直接配置nginx反向代理（不使用宝塔API）
certbot_setup_proxy() {
    log_step "配置Nginx反向代理..."
    
    local ext_dir="/www/server/panel/vhost/nginx/extension/${DOMAIN}"
    local proxy_conf="${ext_dir}/proxy.conf"
    
    # 创建扩展配置目录
    mkdir -p "$ext_dir"
    
    # 检查是否已存在配置
    if [[ -f "$proxy_conf" ]]; then
        if grep -q "proxy_pass.*127.0.0.1:8000" "$proxy_conf" 2>/dev/null; then
            log_info "✅ 反向代理配置已存在且正确"
            # 确保主配置文件包含扩展配置
            ensure_proxy_include
            return 0
        else
            log_warn "⚠️  发现现有的反向代理配置，但目标地址不同"
            log_info "当前配置内容："
            grep "proxy_pass" "$proxy_conf" 2>/dev/null | head -1 || log_info "   未找到proxy_pass"
            read -r -p "是否覆盖现有配置？(yes/no，默认yes): " overwrite
            overwrite=${overwrite:-yes}
            if [[ "$overwrite" != "yes" ]]; then
                log_info "保留现有配置"
                ensure_proxy_include
                return 0
            fi
        fi
    fi
    
    # 生成反向代理配置
    cat > "$proxy_conf" << 'EOF'
# 反向代理配置 - 自动生成
location /api/ {
    proxy_pass http://127.0.0.1:8000;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection 'upgrade';
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_cache_bypass $http_upgrade;
    
    proxy_connect_timeout 60s;
    proxy_send_timeout 60s;
    proxy_read_timeout 60s;
    
    client_max_body_size 10M;
}
EOF
    
    chmod 644 "$proxy_conf"
    
    # 确保主配置文件包含扩展配置（HTTP和HTTPS都需要）
    ensure_proxy_include
    
    # 验证并重载nginx
    log_info "验证Nginx配置..."
    if nginx -t 2>/dev/null; then
        nginx -s reload 2>/dev/null
        sleep 2
        
        # 验证反向代理是否工作（通过HTTPS）
        log_info "验证反向代理配置..."
        local proxy_test
        proxy_test=$(curl -k -s -o /dev/null -w "%{http_code}" "https://${DOMAIN}/api/v1/health" 2>/dev/null || echo "000")
        if [[ "$proxy_test" == "200" ]] || [[ "$proxy_test" == "404" ]] || [[ "$proxy_test" == "401" ]]; then
            log_info "✅ 反向代理配置成功！(测试响应: HTTP $proxy_test)"
        else
            log_warn "⚠️  反向代理测试响应异常 (HTTP $proxy_test)"
            log_warn "   请检查后端服务是否运行在 127.0.0.1:8000"
        fi
        
        log_info "✅ 反向代理配置已应用"
        return 0
    else
        log_error "❌ Nginx配置错误"
        log_error "请运行: nginx -t 查看详细错误"
        nginx -t 2>&1 | head -10
        return 1
    fi
}

# 确保主配置文件包含扩展配置（HTTP和HTTPS server块都需要）
ensure_proxy_include() {
    local nginx_conf="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
    if [[ ! -f "$nginx_conf" ]]; then
        return 1
    fi
    
    local include_line="include /www/server/panel/vhost/nginx/extension/${DOMAIN}/*.conf;"
    local needs_update=false
    
    # 检查HTTP server块
    if ! grep -A 20 "listen 80" "$nginx_conf" 2>/dev/null | grep -q "include.*extension.*${DOMAIN}" 2>/dev/null; then
        log_info "在HTTP server块中添加扩展配置include..."
        # 在server_name之后添加
        if grep -q "server_name.*${DOMAIN}" "$nginx_conf" 2>/dev/null; then
            sed -i "/server_name.*${DOMAIN}/a\    ${include_line}" "$nginx_conf" 2>/dev/null
            needs_update=true
        fi
    fi
    
    # 检查HTTPS server块
    if grep -q "listen 443 ssl" "$nginx_conf" 2>/dev/null; then
        if ! grep -A 20 "listen 443 ssl" "$nginx_conf" 2>/dev/null | grep -q "include.*extension.*${DOMAIN}" 2>/dev/null; then
            log_info "在HTTPS server块中添加扩展配置include..."
            # 在HTTPS server块的server_name之后添加
            local https_server_name_line
            https_server_name_line=$(grep -n "listen 443 ssl" "$nginx_conf" | head -1 | cut -d: -f1)
            if [[ -n "$https_server_name_line" ]]; then
                # 找到HTTPS server块中的server_name行
                local line_num=$((https_server_name_line + 5))  # 通常在listen后几行
                sed -i "${line_num}a\    ${include_line}" "$nginx_conf" 2>/dev/null || {
                    # 如果失败，尝试在ssl_certificate之后添加
                    sed -i "/ssl_certificate_key/a\    ${include_line}" "$nginx_conf" 2>/dev/null
                }
                needs_update=true
            fi
        fi
    fi
    
    if [[ "$needs_update" == true ]]; then
        log_info "✅ 已更新配置文件，添加扩展配置include"
        # 验证配置
        if nginx -t 2>/dev/null; then
            nginx -s reload 2>/dev/null
            log_info "✅ Nginx配置已重载"
        else
            log_error "❌ 配置更新后Nginx语法错误"
            nginx -t 2>&1 | head -10
        fi
    fi
}

# 最终验证HTTPS访问
verify_https_access() {
    log_step "最终验证HTTPS访问..."
    
    log_info "测试HTTPS连接..."
    local https_code
    https_code=$(curl -k -s -o /dev/null -w "%{http_code}" "https://${DOMAIN}" 2>/dev/null || echo "000")
    
    if [[ "$https_code" == "200" ]] || [[ "$https_code" == "301" ]] || [[ "$https_code" == "302" ]]; then
        log_info "✅ HTTPS访问正常 (HTTP $https_code)"
        
        # 测试反向代理
        log_info "测试反向代理..."
        local api_code
        api_code=$(curl -k -s -o /dev/null -w "%{http_code}" "https://${DOMAIN}/api/v1/health" 2>/dev/null || echo "000")
        if [[ "$api_code" == "200" ]] || [[ "$api_code" == "404" ]] || [[ "$api_code" == "401" ]]; then
            log_info "✅ 反向代理工作正常 (HTTP $api_code)"
        else
            log_warn "⚠️  反向代理测试异常 (HTTP $api_code)，但HTTPS已正常工作"
        fi
        
        log_info ""
        log_info "═══════════════════════════════════════════════════════════"
        log_info "🎉 配置完成！网站已可通过HTTPS访问"
        log_info "═══════════════════════════════════════════════════════════"
        log_info "   HTTPS地址: https://${DOMAIN}"
        log_info "   HTTP会自动重定向到HTTPS"
        log_info "   反向代理: https://${DOMAIN}/api/ -> http://127.0.0.1:8000"
        log_info "═══════════════════════════════════════════════════════════"
        return 0
    else
        log_error "❌ HTTPS访问失败 (HTTP $https_code)"
        log_error "请检查："
        log_error "  1. SSL证书是否正确配置"
        log_error "  2. Nginx配置是否正确"
        log_error "  3. 防火墙是否开放443端口"
        log_error "  4. 运行: nginx -t 检查配置"
        return 1
    fi
}

# 自动配置SSL和反向代理（菜单选项）
auto_config_ssl_proxy() {
    log_step "自动配置SSL证书和反向代理..."
    
    if ! check_bt_panel; then
        log_error "未检测到宝塔面板环境"
        return 1
    fi
    
    log_info ""
    log_info "请选择配置方式："
    log_info "  1. 使用宝塔面板API（需要API密钥）"
    log_info "  2. 直接使用certbot（不需要API密钥）"
    log_info ""
    read -r -p "请选择 [1/2，默认2]: " method_choice
    method_choice=${method_choice:-2}
    
    if [[ "$method_choice" == "1" ]]; then
        # 方案一：使用宝塔API
        if ! get_bt_api_key; then
            log_error "无法获取宝塔API密钥，请手动配置"
            return 1
        fi
        
        # 检查网站是否存在（如果不存在会自动创建配置文件）
        if ! check_website_exists; then
            log_error "网站配置创建失败，无法继续配置"
            return 1
        fi
        
        # 确保配置文件存在
        local nginx_conf="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
        if [[ ! -f "$nginx_conf" ]]; then
            log_error "网站配置文件不存在: $nginx_conf"
            log_error "请先在宝塔面板中创建网站，或检查域名是否正确"
            return 1
        fi
        
        log_info ""
        log_info "开始使用宝塔API配置..."
        log_info ""
        
        read -r -p "是否自动申请SSL证书？(yes/no，默认yes): " auto_ssl_confirm
        auto_ssl_confirm=${auto_ssl_confirm:-yes}
        if [[ "$auto_ssl_confirm" == "yes" ]]; then
            auto_apply_ssl
            log_info ""
        fi
        
        read -r -p "是否自动设置反向代理？(yes/no，默认yes): " auto_proxy_confirm
        auto_proxy_confirm=${auto_proxy_confirm:-yes}
        if [[ "$auto_proxy_confirm" == "yes" ]]; then
            auto_setup_proxy
            log_info ""
        fi
    else
        # 方案二：直接使用certbot
        log_info ""
        log_info "开始使用certbot直接配置..."
        log_info ""
        
        # 检查网站是否存在（会创建配置文件如果不存在）
        if check_website_exists || [[ -f "/www/server/panel/vhost/nginx/${DOMAIN}.conf" ]]; then
            # 网站存在或用户选择继续
            :
        else
            log_error "无法继续配置"
            return 1
        fi
        
        read -r -p "是否自动申请SSL证书？(yes/no，默认yes): " auto_ssl_confirm
        auto_ssl_confirm=${auto_ssl_confirm:-yes}
        if [[ "$auto_ssl_confirm" == "yes" ]]; then
            certbot_apply_ssl
            log_info ""
        fi
        
        read -r -p "是否自动设置反向代理？(yes/no，默认yes): " auto_proxy_confirm
        auto_proxy_confirm=${auto_proxy_confirm:-yes}
        if [[ "$auto_proxy_confirm" == "yes" ]]; then
            certbot_setup_proxy
            log_info ""
        fi
        
        # 最终验证HTTPS访问
        log_info ""
        verify_https_access
    fi
    
    log_info ""
    log_info "建议运行菜单选项 14 检查配置是否正确"
}

generate_nginx_config() {
    log_step "生成 Nginx 配置参考文件..."
    local conf="/tmp/cboard_nginx_${DOMAIN}.conf"
    
    # 生成网站主配置文件参考（专为SSL证书申请和反向代理优化）
    # 注意：此配置确保SSL证书申请和反向代理设置不会出错
    cat > "$conf" << EOF
server {
    listen 80;
    server_name ${DOMAIN};
    
    # 前端静态文件目录
    root ${PROJECT_DIR}/frontend/dist;
    index index.html;
    
    # ⚠️ 重要：宝塔面板SSL自动部署标识（必须保留，不能删除！）
    # 宝塔面板通过这个标识来确定SSL配置的添加位置
    # 如果缺少此标识，SSL证书申请成功后无法自动部署
    #error_page 404/404.html;
    
    # ⚠️ 重要：包含宝塔面板的扩展配置（必须保留，不能注释！）
    # 宝塔面板通过这个include来添加反向代理等配置
    # 如果缺少此include，反向代理配置无法生效
    include /www/server/panel/vhost/nginx/extension/${DOMAIN}/*.conf;

    # ⚠️ 重要：SSL证书验证路径（必须在最前面，优先级最高）
    # 宝塔面板会在项目根目录创建验证文件，所以root必须指向项目根目录
    # 如果指向 frontend/dist，SSL证书申请会失败
    # 此location必须在 location / 之前，确保优先级更高
    location /.well-known/acme-challenge/ {
        root ${PROJECT_DIR};
        allow all;
        access_log off;
        log_not_found off;
        default_type text/plain;
    }
    
    # 通用 .well-known 路径（用于SSL验证和续期）
    location /.well-known/ {
        root ${PROJECT_DIR};
        allow all;
        access_log off;
        log_not_found off;
        default_type text/plain;
    }

    # 前端路由（Vue Router）
    # 注意：此location必须在 .well-known 之后
    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # 禁止 index.html 缓存
    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        add_header Pragma "no-cache";
        add_header Expires "0";
        try_files \$uri /index.html;
    }

    # 静态资源缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
        access_log off;
    }

    # 日志
    access_log /www/wwwlogs/${DOMAIN}.log;
    error_log /www/wwwlogs/${DOMAIN}.error.log;
}
EOF
    log_info "✅ 网站配置参考文件已生成: $conf"
    log_info ""
    log_info "📋 配置要点（确保SSL和反向代理正常工作）："
    log_info "   1. 必须包含: #error_page 404/404.html; (SSL自动部署标识)"
    log_info "   2. 必须包含: include /www/server/panel/vhost/nginx/extension/${DOMAIN}/*.conf; (反向代理支持)"
    log_info "   3. SSL验证路径root必须指向: ${PROJECT_DIR} (项目根目录)"
    log_info "   4. .well-known location必须在 location / 之前"
}

manage_cache() {
    log_step "清除缓存..."
    local deep=$1
    rm -rf "$PROJECT_DIR/.cache" "/tmp/cboard_cache"
    
    if [[ "$deep" == "deep" ]]; then
        log_info "执行深度清理..."
        npm cache clean --force 2>/dev/null
        command -v go &>/dev/null && go clean -cache -modcache -i -r 2>/dev/null
        rm -rf "$PROJECT_DIR/frontend/dist" "$PROJECT_DIR/frontend/node_modules/.cache"
    fi
    log_info "✅ 缓存清理完成"
}

unlock_admin() {
    log_step "解锁管理员..."
    setup_go_env
    read -r -p "输入用户名 (默认 admin): " user
    user=${user:-admin}
    go run unlock_admin.go "$user"
}

show_db_info() {
    [[ -f "cboard.db" ]] && {
        log_info "DB大小: $(du -sh cboard.db | awk '{print $1}')"
        setup_go_env && go run scripts/check_admin.go 2>/dev/null
    } || log_warn "数据库不存在"
}

# 检查配置文件是否包含当前域名
check_config_contains_domain() {
    local file="$1"
    local domain="$2"
    
    [[ ! -f "$file" ]] && return 1
    
    # 检查文件中是否包含当前域名
    if grep -q "${domain}" "$file" 2>/dev/null; then
        return 0
    fi
    
    return 1
}

delete_all_configs() {
    log_step "卸载网站 - 删除所有相关配置..."
    
    # 获取域名（如果未设置）
    [[ -z "$DOMAIN" ]] && get_domain
    
    log_warn "⚠️  警告：此操作将完全卸载网站，删除以下所有配置："
    log_info "1. 临时配置文件: /tmp/cboard_*_${DOMAIN}.conf"
    log_info "2. 宝塔面板网站配置: /www/server/panel/vhost/nginx/${DOMAIN}.conf"
    log_info "3. 宝塔面板扩展配置目录: /www/server/panel/vhost/nginx/extension/${DOMAIN}/"
    log_info "4. 宝塔面板 Apache 配置: /www/server/panel/vhost/apache/${DOMAIN}.conf"
    log_info "5. Systemd 服务: /etc/systemd/system/cboard.service"
    log_info "6. 项目配置文件: ${PROJECT_DIR}/.env"
    log_info "7. 日志文件: /www/wwwlogs/${DOMAIN}*.log"
    log_info "8. 检查并清理共享配置文件中的相关配置"
    
    read -r -p "确认完全卸载？(yes/no): " confirm
    [[ "$confirm" != "yes" ]] && { log_info "已取消"; return 0; }
    
    # 停止并禁用服务
    log_step "停止并禁用服务..."
    systemctl stop cboard 2>/dev/null
    systemctl disable cboard 2>/dev/null
    pkill -9 -f "${PROJECT_DIR}/server" 2>/dev/null
    kill_port 8000
    
    local deleted_count=0
    
    # 删除所有临时配置文件
    log_step "删除临时配置文件..."
    for tmp_file in /tmp/cboard_nginx_${DOMAIN}.conf /tmp/cboard_proxy_${DOMAIN}.conf; do
        if [[ -f "$tmp_file" ]]; then
            rm -f "$tmp_file"
            log_info "✅ 已删除: $tmp_file"
            ((deleted_count++))
        fi
    done
    
    # 删除宝塔面板网站配置文件（Nginx）
    log_step "删除宝塔面板网站配置..."
    local bt_conf="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
    if [[ -f "$bt_conf" ]]; then
        rm -f "$bt_conf"
        log_info "✅ 已删除宝塔面板网站配置: $bt_conf"
        ((deleted_count++))
        # 重载 Nginx
        command -v nginx &>/dev/null && nginx -s reload 2>/dev/null && log_info "✅ Nginx 已重载"
    fi
    
    # 删除宝塔面板扩展配置目录（强制删除，包括所有子文件和子目录）
    log_step "删除扩展配置目录..."
    local ext_dir="/www/server/panel/vhost/nginx/extension/${DOMAIN}"
    if [[ -d "$ext_dir" ]] || [[ -e "$ext_dir" ]]; then
        log_info "正在删除扩展配置目录: $ext_dir"
        # 先列出目录内容
        if [[ -d "$ext_dir" ]]; then
            local files_count=$(find "$ext_dir" -type f 2>/dev/null | wc -l)
            local dirs_count=$(find "$ext_dir" -type d 2>/dev/null | wc -l)
            log_info "   发现 $files_count 个文件，$dirs_count 个目录"
            # 列出所有文件
            if [[ $files_count -gt 0 ]]; then
                log_info "   文件列表："
                find "$ext_dir" -type f 2>/dev/null | while read -r file; do
                    log_info "     - $file"
                done
            fi
        fi
        # 强制删除目录及其所有内容（使用多种方法确保删除）
        rm -rf "$ext_dir" 2>/dev/null
        # 如果还存在，尝试使用 find 删除
        if [[ -d "$ext_dir" ]]; then
            find "$ext_dir" -delete 2>/dev/null
            rm -rf "$ext_dir" 2>/dev/null
        fi
        # 验证是否删除成功
        sleep 0.5  # 等待文件系统同步
        if [[ ! -d "$ext_dir" ]] && [[ ! -e "$ext_dir" ]]; then
            log_info "✅ 已删除扩展配置目录: $ext_dir"
            ((deleted_count++))
        else
            log_error "❌ 扩展配置目录删除失败: $ext_dir"
            log_warn "   请手动检查并删除该目录: rm -rf $ext_dir"
            log_warn "   或检查文件权限: ls -la $(dirname "$ext_dir")"
        fi
    else
        log_info "扩展配置目录不存在: $ext_dir"
    fi
    
    # 删除宝塔面板 Apache 配置文件（如果存在）
    log_step "删除 Apache 配置..."
    local apache_conf="/www/server/panel/vhost/apache/${DOMAIN}.conf"
    if [[ -f "$apache_conf" ]]; then
        rm -f "$apache_conf"
        log_info "✅ 已删除 Apache 配置: $apache_conf"
        ((deleted_count++))
        command -v apachectl &>/dev/null && apachectl graceful 2>/dev/null
    fi
    
    # 删除 Systemd 服务文件
    log_step "删除 Systemd 服务..."
    local svc="/etc/systemd/system/cboard.service"
    if [[ -f "$svc" ]]; then
        systemctl daemon-reload 2>/dev/null
        rm -f "$svc"
        systemctl daemon-reload 2>/dev/null
        log_info "✅ 已删除 Systemd 服务文件: $svc"
        ((deleted_count++))
    fi
    
    # 检查共享配置文件（仅提示，不自动删除，避免误删其他网站配置）
    log_step "检查共享配置文件..."
    local shared_configs=(
        "/www/server/panel/vhost/nginx/default.conf"
        "/etc/nginx/conf.d/default.conf"
    )
    
    local found_shared=false
    for shared_conf in "${shared_configs[@]}"; do
        if [[ -f "$shared_conf" ]] && check_config_contains_domain "$shared_conf" "$DOMAIN"; then
            log_warn "⚠️  发现共享配置文件包含当前域名: $shared_conf"
            log_warn "   为避免误删其他网站配置，请手动检查并清理"
            found_shared=true
        fi
    done
    
    if [[ "$found_shared" == true ]]; then
        log_warn "⚠️  请手动检查共享配置文件，确保只删除属于 ${DOMAIN} 的配置"
        log_warn "   删除后记得重载 Nginx: nginx -s reload"
    fi
    
    # 询问是否删除项目 .env 文件
    if [[ -f "${PROJECT_DIR}/.env" ]]; then
        read -r -p "是否删除项目 .env 文件？(yes/no): " del_env
        if [[ "$del_env" == "yes" ]]; then
            rm -f "${PROJECT_DIR}/.env"
            log_info "✅ 已删除 .env 文件: ${PROJECT_DIR}/.env"
            ((deleted_count++))
        else
            log_info "已保留 .env 文件"
        fi
    fi
    
    # 询问是否删除日志文件
    read -r -p "是否删除网站日志文件？(yes/no): " del_logs
    if [[ "$del_logs" == "yes" ]]; then
        local log_files=(
            "/www/wwwlogs/${DOMAIN}.log"
            "/www/wwwlogs/${DOMAIN}.error.log"
            "/www/wwwlogs/${DOMAIN}_access.log"
            "/www/wwwlogs/${DOMAIN}_error.log"
        )
        for log_file in "${log_files[@]}"; do
            if [[ -f "$log_file" ]]; then
                rm -f "$log_file"
                log_info "✅ 已删除日志文件: $log_file"
                ((deleted_count++))
            fi
        done
    fi
    
    log_info "✅ 卸载完成，共删除 $deleted_count 个配置文件/目录"
    log_warn "⚠️  注意："
    log_warn "   1. 如果网站仍在宝塔面板中，请手动在宝塔面板中删除网站"
    log_warn "   2. 项目文件目录 ${PROJECT_DIR} 未被删除，如需完全清理请手动删除"
    log_warn "   3. 数据库文件 ${PROJECT_DIR}/cboard.db 未被删除，如需清理请手动删除"
    
    # 再次检查扩展配置目录是否还存在
    if [[ -d "$ext_dir" ]] || [[ -e "$ext_dir" ]]; then
        log_error "❌ 扩展配置目录仍然存在: $ext_dir"
        log_warn "请手动检查并删除该目录"
    fi
}

# --- 菜单与入口 ---
full_build() {
    check_root
    detect_os
    check_bt_panel
    get_domain
    setup_project_dir
    create_env_file
    
    install_go
    install_nodejs
    manage_cache deep
    
    install_go_deps
    build_backend
    init_database
    
    build_frontend
    create_directories
    set_permissions
    create_systemd_service
    
    test_backend
    generate_nginx_config
    manage_service restart
    manage_cache
    
    log_info "🚀 部署完成!"
    log_info ""
    
    # 询问是否自动配置SSL和反向代理
    if check_bt_panel; then
        log_info "═══════════════════════════════════════════════════════════"
        log_info "🤖 自动配置SSL证书和反向代理"
        log_info "═══════════════════════════════════════════════════════════"
    log_info ""
        log_info "脚本可以自动完成以下配置："
        log_info "  1. 申请SSL证书（Let's Encrypt）"
        log_info "  2. 设置反向代理（/api/ -> http://127.0.0.1:8000）"
    log_info ""
        log_warn "⚠️  前提条件："
        log_warn "  - 必须在宝塔面板中先创建网站（域名: $DOMAIN）"
        log_warn "  - 域名必须已正确解析到服务器"
    log_info ""
        read -r -p "是否使用自动配置？(yes/no，默认yes): " auto_config
        auto_config=${auto_config:-yes}
        
        if [[ "$auto_config" == "yes" ]]; then
    log_info ""
            log_info "请选择配置方式："
            log_info "  1. 使用宝塔面板API（已配置API密钥）"
            log_info "  2. 直接使用certbot（不需要API密钥，推荐）"
    log_info ""
            read -r -p "请选择 [1/2，默认2]: " method_choice
            method_choice=${method_choice:-2}
            
            log_info ""
            log_info "开始自动配置..."
            log_info ""
            
            if [[ "$method_choice" == "1" ]]; then
                # 方案一：使用宝塔API
                if ! get_bt_api_key; then
                    log_warn "无法获取API密钥，切换到certbot方式"
                    method_choice=2
                fi
            fi
            
            if [[ "$method_choice" == "1" ]]; then
                # 检查网站是否存在
                if check_website_exists; then
                    # 申请SSL证书
                    log_info "正在申请SSL证书..."
                    auto_apply_ssl
                    log_info ""
                    
                    # 设置反向代理
                    log_info "正在设置反向代理..."
                    auto_setup_proxy
                    log_info ""
                    
                    # 最终验证HTTPS访问
                    log_info ""
                    verify_https_access
                    
                    log_info "✅ 自动配置完成！"
                else
                    log_error "网站配置创建失败"
                    log_error "请检查："
                    log_error "  1. 项目目录是否正确: ${PROJECT_DIR}"
                    log_error "  2. 域名是否正确: ${DOMAIN}"
                    log_error "  3. 是否有权限创建配置文件"
                    log_info "或者先在宝塔面板中手动创建网站，然后运行脚本菜单选项 15 进行自动配置"
                fi
            else
                # 方案二：直接使用certbot
                # 检查网站是否存在（如果不存在会自动创建配置文件）
                if ! check_website_exists; then
                    log_error "网站配置创建失败，无法继续配置"
                    log_error "请检查："
                    log_error "  1. 项目目录是否正确: ${PROJECT_DIR}"
                    log_error "  2. 域名是否正确: ${DOMAIN}"
                    log_error "  3. 是否有权限创建配置文件"
                else
                    # 确保配置文件存在
                    local nginx_conf="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
                    if [[ ! -f "$nginx_conf" ]]; then
                        log_error "网站配置文件不存在: $nginx_conf"
                    else
                        # 申请SSL证书
                        log_info "正在使用certbot申请SSL证书..."
                        certbot_apply_ssl
                        local ssl_result=$?
                        if [[ $ssl_result -eq 2 ]]; then
                            # SSL申请被跳过，继续配置反向代理
                            log_info "SSL证书申请已跳过，继续配置反向代理..."
                        elif [[ $ssl_result -ne 0 ]]; then
                            log_warn "⚠️  SSL证书申请失败，但将继续配置反向代理"
                        fi
                        log_info ""
                        
                        # 设置反向代理
                        log_info "正在设置反向代理..."
                        certbot_setup_proxy
                        log_info ""
                        
                        # 最终验证（如果SSL配置成功）
                        if [[ $ssl_result -eq 0 ]]; then
                            log_info ""
                            verify_https_access
                        else
                            log_info ""
                            log_warn "⚠️  SSL证书未配置，网站只能通过HTTP访问"
                            log_info "   建议："
                            log_info "   1. 手动安装certbot后重新运行此脚本"
                            log_info "   2. 或使用宝塔面板手动申请SSL证书"
                            log_info "   3. 或使用宝塔面板API方式（选项1）申请SSL证书"
                        fi
                        
                        log_info "✅ 自动配置完成！"
                    fi
                fi
            fi
            
            log_info ""
            log_info "建议运行菜单选项 14 检查配置是否正确"
            log_info ""
            log_info "═══════════════════════════════════════════════════════════"
            log_info "📋 后续步骤（如果自动配置成功，可跳过）："
            log_info "═══════════════════════════════════════════════════════════"
        else
            log_info "跳过自动配置，请按照以下步骤手动配置"
            log_info ""
            log_info "═══════════════════════════════════════════════════════════"
            log_info "📋 后续配置步骤（请严格按照顺序执行，避免错误）："
            log_info "═══════════════════════════════════════════════════════════"
        fi
        log_info ""
    else
        log_info "═══════════════════════════════════════════════════════════"
        log_info "📋 后续配置步骤（请严格按照顺序执行，避免错误）："
        log_info "═══════════════════════════════════════════════════════════"
        log_info ""
    fi
    
    log_info "【第一步】在宝塔面板中创建网站"
    log_info "  1. 进入宝塔面板 → 网站 → 添加站点"
    log_info "  2. 域名: $DOMAIN"
    log_info "  3. 根目录: ${PROJECT_DIR}/frontend/dist"
    log_info "  4. 其他选项保持默认，点击提交"
    log_info ""
    log_info "【第二步】配置Nginx（确保SSL证书申请成功）"
    log_info "  1. 网站 → 设置 → 配置文件"
    log_info "  2. 参考文件: /tmp/cboard_nginx_${DOMAIN}.conf"
    log_info "  3. ⚠️ 必须添加以下配置（复制参考文件中的内容）："
    log_info "     - #error_page 404/404.html; (SSL自动部署标识)"
    log_info "     - include /www/server/panel/vhost/nginx/extension/${DOMAIN}/*.conf; (反向代理支持)"
    log_info "     - location /.well-known/acme-challenge/ { root ${PROJECT_DIR}; ... }"
    log_info "     - location /.well-known/ { root ${PROJECT_DIR}; ... }"
    log_info "  4. ⚠️ 重要：SSL验证路径的root必须指向项目根目录: ${PROJECT_DIR}"
    log_info "     不能指向: ${PROJECT_DIR}/frontend/dist"
    log_info "  5. 保存配置"
    log_info ""
    log_info "【第三步】申请SSL证书（必须先完成第二步）"
    log_info "  1. 网站 → 设置 → SSL"
    log_info "  2. 选择 Let's Encrypt"
    log_info "  3. 验证方式选择：文件验证"
    log_info "  4. 点击申请"
    log_info "  5. ⚠️ 等待证书申请成功（可能需要1-2分钟）"
    log_info "  6. ⚠️ 不要先开启强制HTTPS，等证书申请成功后再开启"
    log_info ""
    log_info "【第四步】设置反向代理（必须在SSL证书申请成功后）"
    log_info "  1. 网站 → 设置 → 反向代理"
    log_info "  2. 点击"添加反向代理""
    log_info "  3. 配置参数："
    log_info "     - 代理名称: api (任意名称)"
    log_info "     - 目标URL: http://127.0.0.1:8000"
    log_info "     - 发送域名: $DOMAIN"
    log_info "     - 位置: /api/"
    log_info "  4. 其他选项保持默认"
    log_info "  5. 点击提交"
    log_info "  6. ⚠️ 确保反向代理状态为"已开启""
    log_info ""
    log_info "【第五步】验证配置（可选，但强烈推荐）"
    log_info "  运行脚本菜单选项 14 检查配置是否正确"
    log_info ""
    log_info "【第六步】访问网站"
    log_info "  - HTTP: http://$DOMAIN"
    log_info "  - HTTPS: https://$DOMAIN (证书申请后)"
    log_info "  - 管理员登录: https://$DOMAIN/admin/login"
    log_info ""
    log_info "═══════════════════════════════════════════════════════════"
    log_warn "⚠️  重要提示："
    log_warn "   1. 必须严格按照顺序执行：创建网站 → 配置Nginx → 申请SSL → 设置反向代理"
    log_warn "   2. SSL验证路径的root必须指向项目根目录，否则SSL申请会失败"
    log_warn "   3. 反向代理必须在SSL证书申请成功后设置，避免配置冲突"
    log_warn "   4. 如果遇到问题，运行脚本菜单选项 14 检查配置"
    log_info "═══════════════════════════════════════════════════════════"
}

show_menu() {
    clear
    echo "=========================================="
    echo "🚀 CBoard Go 管理工具 - $PROJECT_DIR"
    echo "=========================================="
    echo " 1. 完整构建 (部署/更新)"
    echo " 2. 创建/重置管理员"
    echo " 3. 强制重启服务 (杀进程)"
    echo " 4. 深度清理缓存"
    echo " 5. 解锁管理员账户"
    echo " 6. 服务状态"
    echo " 7. 服务日志"
    echo " 8. 重启服务"
    echo " 9. 停止服务"
    echo " 10. 生成 Nginx 配置"
    echo " 11. 测试后端"
    echo " 12. 数据库信息"
    echo " 13. 删除所有配置文件"
    echo " 14. 检查网站配置 (SSL/反向代理)"
    echo " 15. 自动配置SSL和反向代理"
    echo " 0. 退出"
    echo "=========================================="
    read -r -p "请选择 [0-15]: " choice
}

main() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -d|--dir) PROJECT_DIR="$2"; shift 2 ;;
            -n|--domain) DOMAIN="$2"; shift 2 ;;
            -h|--help) echo "Usage: $0 [-d dir] [-n domain]"; exit 0 ;;
            *) shift ;;
        esac
    done

    [[ $# -gt 0 ]] && { full_build; exit 0; } # Compat: args triggers build

    while true; do
        show_menu
        case $choice in
            1) full_build; read -r -p "按回车继续..." ;;
            2) check_root; setup_project_dir; create_admin_account; read -r -p "按回车继续..." ;;
            3) check_root; setup_project_dir; manage_service restart force; read -r -p "按回车继续..." ;;
            4) check_root; setup_project_dir; manage_cache deep; read -r -p "按回车继续..." ;;
            5) check_root; setup_project_dir; unlock_admin; read -r -p "按回车继续..." ;;
            6) manage_service status; read -r -p "按回车继续..." ;;
            7) manage_service logs; read -r -p "按回车继续..." ;;
            8) check_root; manage_service restart; read -r -p "按回车继续..." ;;
            9) check_root; manage_service stop; read -r -p "按回车继续..." ;;
            10) check_root; setup_project_dir; get_domain; generate_nginx_config; read -r -p "按回车继续..." ;;
            11) check_root; setup_project_dir; test_backend; read -r -p "按回车继续..." ;;
            12) setup_project_dir; show_db_info; read -r -p "按回车继续..." ;;
            13) check_root; setup_project_dir; delete_all_configs; read -r -p "按回车继续..." ;;
            14) check_root; setup_project_dir; get_domain; check_website_config; check_proxy_config; read -r -p "按回车继续..." ;;
            15) check_root; setup_project_dir; get_domain; auto_config_ssl_proxy; read -r -p "按回车继续..." ;;
            0) exit 0 ;;
            *) log_error "无效选项"; sleep 1 ;;
        esac
    done
}

main "$@"
