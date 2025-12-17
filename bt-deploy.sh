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

generate_nginx_config() {
    log_step "生成 Nginx 配置..."
    local conf="/tmp/cboard_nginx_${DOMAIN}.conf"
    cat > "$conf" << EOF
server {
    listen 80;
    server_name ${DOMAIN};
    root ${PROJECT_DIR}/frontend/dist;
    index index.html;

    location /.well-known/ { root ${PROJECT_DIR}; allow all; }
    location / { try_files \$uri \$uri/ /index.html; }
    location = /index.html { add_header Cache-Control "no-cache"; try_files \$uri /index.html; }
    
    location /api/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }
    
    location ~* \.(js|css|png|jpg|gif|ico|svg)$ { expires 1y; }
    access_log /www/wwwlogs/${DOMAIN}.log;
    error_log /www/wwwlogs/${DOMAIN}.error.log;
}
EOF
    log_info "✅ 配置已生成: $conf"
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
    
    log_info "🚀 部署完成! 访问: http://$DOMAIN"
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
    echo " 0. 退出"
    echo "=========================================="
    read -r -p "请选择 [0-12]: " choice
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
            0) exit 0 ;;
            *) log_error "无效选项"; sleep 1 ;;
        esac
    done
}

main "$@"
