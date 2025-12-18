#!/bin/bash
# ============================================
# CBoard Go 一键安装脚本 - 宝塔面板版（优化版）
# ============================================
# 功能：自动安装所需环境并完成网站部署
# 支持：Ubuntu/Debian/CentOS/Rocky Linux
# 优化：基于 install.sh 的正确逻辑，减少代码重复
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

# --- 日志函数 ---
log() { echo -e "${GREEN}[$(date +'%H:%M:%S')] $1${NC}" | tee -a "$LOG_FILE"; }
warn() { echo -e "${YELLOW}[WARN] $1${NC}" | tee -a "$LOG_FILE"; }
error() { echo -e "${RED}[ERROR] $1${NC}" | tee -a "$LOG_FILE"; }
step() { echo -e "${BLUE}[STEP] $1${NC}" | tee -a "$LOG_FILE"; }

# --- 基础检查 ---
check_root() {
    [[ "$EUID" -ne 0 ]] && { error "请使用 root 用户运行: sudo $0"; exit 1; }
}

detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        log "检测到操作系统: $OS $VERSION_ID"
    else
        error "无法检测操作系统"
        exit 1
    fi
}

check_bt_panel() {
    if [ -d "/www/server" ]; then
        log "✅ 检测到宝塔面板环境"
        return 0
    else
        warn "未检测到宝塔面板，使用标准 Linux 环境"
        return 1
    fi
}

# --- 环境安装（保留原有逻辑）---
persist_path() {
    local dir="$1"
    [[ -z "$dir" ]] && return
    export PATH="$PATH:$dir"
    for f in ~/.bashrc /etc/profile; do
        grep -q "$dir" "$f" 2>/dev/null || echo "export PATH=\$PATH:$dir" >> "$f"
    done
}

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
        log "Go 环境已配置: $go_dir"
        return 0
    fi
    return 1
}

install_go() {
    setup_go_env && command -v go &>/dev/null && { log "Go 已安装: $(go version)"; return 0; }
    
    step "安装 Go $GO_VERSION..."
    local arch; arch=$(uname -m)
    case $arch in x86_64) arch="amd64";; aarch64|arm64) arch="arm64";; *) error "不支持架构: $arch"; exit 1;; esac
    
    local tar="go${GO_VERSION}.linux-${arch}.tar.gz"
    cd /tmp || exit
    wget -q --show-progress "https://go.dev/dl/${tar}" -O "$tar" || { error "下载 Go 失败"; exit 1; }
    
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "$tar" && rm -f "$tar"
    
    persist_path "/usr/local/go/bin"
    setup_go_env
    
    command -v go &>/dev/null && log "✅ Go 安装成功" || { error "Go 安装失败"; exit 1; }
}

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
        log "Node.js 环境已配置: $node_dir"
        return 0
    fi
    return 1
}

check_node_version() {
    command -v node &>/dev/null || return 1
    local ver; ver=$(node -v | sed 's/v//')
    [[ $(echo "$ver" | cut -d. -f1) -ge 18 ]] || { warn "Node.js 版本过低: v$ver (需 >= 18)"; return 1; }
    return 0
}

install_nodejs_binary() {
    step "安装 Node.js 18+ (二进制)..."
    local arch; arch=$(uname -m)
    local node_arch
    case $arch in x86_64) node_arch="x64";; aarch64|arm64) node_arch="arm64";; armv7l) node_arch="armv7l";; *) error "不支持架构"; return 1;; esac
    
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
        check_node_version && { log "Node.js 已安装且版本符合要求"; return 0; }
        warn "尝试升级 Node.js..."
    fi

    if install_nodejs_binary; then
        setup_node_env
        check_node_version && { log "✅ Node.js 升级/安装成功"; return 0; }
    fi
    
    step "尝试使用包管理器安装 Node.js..."
    if [[ "$OS" == "ubuntu" ]] || [[ "$OS" == "debian" ]]; then
        curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash -
        apt-get install -y nodejs
    elif [[ "$OS" == "centos" ]] || [[ "$OS" == "rocky" ]]; then
        curl -fsSL https://rpm.nodesource.com/setup_${NODE_VERSION}.x | bash -
        yum install -y nodejs
    fi
    
    setup_node_env
    check_node_version && { log "✅ Node.js 安装成功"; return 0; } || { error "Node.js 安装失败"; exit 1; }
}

# --- 项目设置 ---
get_domain() {
    if [[ -z "$DOMAIN" ]]; then
        read -r -p "请输入域名 (例如: example.com): " DOMAIN
        [[ -z "$DOMAIN" ]] && { error "域名不能为空"; exit 1; }
    fi
    log "使用域名: $DOMAIN"
}

setup_project_dir() {
    if [[ -z "$PROJECT_DIR" ]]; then
        PROJECT_DIR="/www/wwwroot/${DOMAIN}"
    fi
    mkdir -p "$PROJECT_DIR"
    log "项目目录: $PROJECT_DIR"
}

create_env_file() {
    if [[ ! -f "${PROJECT_DIR}/.env" ]]; then
        step "创建 .env 配置文件..."
        cat > "${PROJECT_DIR}/.env" << EOF
HOST=0.0.0.0
PORT=8000
DEBUG=false
DATABASE_URL=sqlite:///${PROJECT_DIR}/cboard.db
SECRET_KEY=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)
BACKEND_CORS_ORIGINS=https://${DOMAIN}
PROJECT_NAME=CBoard Go
VERSION=1.0.0
API_V1_STR=/api/v1
UPLOAD_DIR=uploads
MAX_FILE_SIZE=10485760
DISABLE_SCHEDULE_TASKS=false
EOF
        log "✅ .env 文件已创建"
    else
        log ".env 文件已存在，跳过创建"
    fi
}

# --- 构建和部署（使用 install.sh 的正确逻辑）---
reload_nginx_force() {
    log "正在重载 Nginx..."
    if [[ -f "/run/nginx.pid" ]] && [[ ! -s "/run/nginx.pid" ]]; then
        rm -f /run/nginx.pid && pkill -9 nginx 2>/dev/null
    fi
    systemctl restart nginx || /etc/init.d/nginx restart || nginx -s reload
}

build_backend() {
    step "编译 Go 程序..."
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; exit 1; }
    
    if ! command -v go &>/dev/null; then
        error "未找到 Go 命令，请先安装 Go"
        exit 1
    fi
    
    if go build -o server ./cmd/server/main.go; then
        log "✅ Go 程序编译成功"
    else
        error "Go 程序编译失败"
        exit 1
    fi
}

build_frontend() {
    step "构建前端..."
    cd "${PROJECT_DIR}/frontend" || { error "前端目录不存在"; exit 1; }
    
    if [ ! -d "node_modules" ]; then
        log "安装前端依赖..."
        npm install --legacy-peer-deps || { error "前端依赖安装失败"; exit 1; }
    fi
    
    if npm run build; then
        log "✅ 前端构建成功"
    else
        error "前端构建失败"
        exit 1
    fi
    cd ..
}

create_systemd_service() {
    step "创建 systemd 服务..."
    local service_file="/etc/systemd/system/cboard.service"
    cat > "$service_file" << EOF
[Unit]
Description=CBoard Go Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${PROJECT_DIR}
ExecStart=${PROJECT_DIR}/server
Restart=always
RestartSec=5
StandardOutput=append:${PROJECT_DIR}/server.log
StandardError=append:${PROJECT_DIR}/server.log
Environment="PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    log "✅ systemd 服务文件已创建"
}

# --- Nginx 配置（使用 install.sh 的正确逻辑）---
configure_nginx() {
    step "配置 Nginx..."
    local bt_path="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
    mkdir -p "$(dirname "$bt_path")"
    
    # 1. 生成初始 HTTP 配置（用于 SSL 证书申请）
    cat > "$bt_path" << EOF
server {
    listen 80;
    server_name ${DOMAIN};
    root ${PROJECT_DIR}/frontend/dist;
    location /.well-known/acme-challenge/ { root ${PROJECT_DIR}; }
    location / { try_files \$uri \$uri/ /index.html; }
}
EOF
    reload_nginx_force
    log "✅ 初始 HTTP 配置已生成"

    # 2. 申请 SSL 证书
    step "申请 SSL 证书..."
    if command -v certbot &>/dev/null; then
        certbot certonly --webroot -w "${PROJECT_DIR}" -d "${DOMAIN}" --email "admin@${DOMAIN}" --agree-tos --non-interactive --quiet 2>/dev/null || {
            warn "SSL 证书申请失败，继续使用 HTTP 配置"
        }
    else
        warn "certbot 未安装，跳过 SSL 证书申请"
    fi
    
    # 3. 生成最终配置（HTTPS 或 HTTP）
    local cert_root=$(find /etc/letsencrypt/live -name "*${DOMAIN}*" -type d | head -n 1)
    if [ -n "$cert_root" ] && [ -f "$cert_root/fullchain.pem" ]; then
        # HTTPS 配置
        cat > "$bt_path" << EOF
server {
    listen 80; server_name ${DOMAIN}; return 301 https://\$host\$request_uri;
}
server {
    listen 443 ssl http2; server_name ${DOMAIN};
    ssl_certificate ${cert_root}/fullchain.pem;
    ssl_certificate_key ${cert_root}/privkey.pem;
    root ${PROJECT_DIR}/frontend/dist;
    location /api/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }
    location / { try_files \$uri \$uri/ /index.html; }
}
EOF
        log "✅ HTTPS 配置已生成"
    else
        # HTTP 配置（带反向代理）
        warn "SSL 证书未找到，使用 HTTP 配置"
        cat > "$bt_path" << EOF
server {
    listen 80;
    server_name ${DOMAIN};
    root ${PROJECT_DIR}/frontend/dist;
    location /api/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }
    location / { try_files \$uri \$uri/ /index.html; }
}
EOF
    fi
    reload_nginx_force
}

# --- 服务管理 ---
manage_service() {
    local action=$1
    case $action in
        start)
            systemctl enable cboard
            systemctl start cboard
            sleep 3
            if systemctl is-active --quiet cboard; then
                log "✅ 服务已启动"
            else
                error "服务启动失败，请查看日志: journalctl -u cboard -n 50"
                return 1
            fi
            ;;
        restart)
            systemctl restart cboard
            log "✅ 服务已重启"
            ;;
        stop)
            systemctl stop cboard
            log "✅ 服务已停止"
            ;;
        status)
            systemctl status cboard --no-pager
            ;;
    esac
}

# --- 主部署流程 ---
full_deploy() {
    log "开始全自动部署流程..."
    
    # 1. 基础检查
    check_root
    detect_os
    check_bt_panel
    get_domain
    setup_project_dir
    create_env_file
    
    # 2. 安装环境
    install_go
    install_nodejs
    
    # 3. 构建项目
    build_backend
    build_frontend
    
    # 4. 创建服务
    create_systemd_service
    
    # 5. 配置 Nginx
    configure_nginx
    
    # 6. 启动服务
    step "启动服务..."
    manage_service start
    
    log "部署完成！"
    log "服务状态: systemctl status cboard"
    log "查看日志: journalctl -u cboard -f"
}

# --- 其他功能 ---
create_admin() {
    step "创建/重置管理员账户..."
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; exit 1; }
    read -r -p "请输入管理员邮箱: " admin_email
    read -r -p "请输入管理员密码: " admin_pass
    export ADMIN_EMAIL="$admin_email"
    export ADMIN_PASSWORD="$admin_pass"
    go run scripts/create_admin.go
    log "✅ 管理员账户已创建/重置"
}

force_restart() {
    step "强制重启服务..."
    log "正在强制停止所有相关进程..."
    pkill -9 -f "${PROJECT_DIR}/server" 2>/dev/null
    pkill -9 server 2>/dev/null
    pkill -9 node 2>/dev/null
    systemctl stop cboard 2>/dev/null
    sleep 2
    log "进程已全部清理"
    
    step "重新启动服务..."
    systemctl start cboard
    sleep 3
    if systemctl is-active --quiet cboard; then
        log "✅ 服务已成功重启"
    else
        error "服务启动失败，请查看日志: journalctl -u cboard -n 50"
        return 1
    fi
}

unlock_user() {
    step "解锁用户账户..."
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; exit 1; }
    read -r -p "请输入要解锁的邮箱账号: " email
    [[ -z "$email" ]] && { error "邮箱不能为空"; return 1; }
    
    # 尝试使用服务器命令解锁
    if [[ -f "${PROJECT_DIR}/server" ]]; then
        "${PROJECT_DIR}/server" -unlock "$email" 2>/dev/null && {
            log "✅ 账户 $email 已解锁"
            return 0
        }
    fi
    
    # 如果服务器命令失败，尝试直接操作数据库
    if [[ -f "${PROJECT_DIR}/cboard.db" ]]; then
        if command -v sqlite3 &>/dev/null; then
            sqlite3 "${PROJECT_DIR}/cboard.db" "UPDATE users SET is_active=1, login_fails=0 WHERE email='$email';" 2>/dev/null && {
                log "✅ 账户 $email 已解锁（通过数据库）"
                return 0
            }
        fi
    fi
    
    warn "无法解锁账户，请检查数据库文件或手动操作"
    return 1
}

deep_clean() {
    step "深度清理缓存..."
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; exit 1; }
    
    log "正在清理前端构建文件..."
    rm -rf "${PROJECT_DIR}/frontend/dist"
    
    log "正在清理日志文件..."
    rm -rf "${PROJECT_DIR}/logs/*" 2>/dev/null
    rm -rf "${PROJECT_DIR}/*.log" 2>/dev/null
    
    log "正在清理临时文件..."
    find "$PROJECT_DIR" -name "*.tmp" -delete 2>/dev/null
    find "$PROJECT_DIR" -name ".cache" -type d -exec rm -rf {} + 2>/dev/null
    
    log "正在清理 npm 缓存..."
    if command -v npm &>/dev/null; then
        npm cache clean --force 2>/dev/null
    fi
    
    log "正在清理 Go 缓存..."
    if command -v go &>/dev/null; then
        go clean -cache -modcache -i -r 2>/dev/null
    fi
    
    log "✅ 缓存清理完毕"
}

delete_all_configs() {
    step "卸载网站 - 删除所有相关配置..."
    
    # 获取域名（如果未设置）
    [[ -z "$DOMAIN" ]] && get_domain
    
    warn "⚠️  警告：此操作将完全卸载网站，删除以下所有配置："
    log "1. 宝塔面板网站配置: /www/server/panel/vhost/nginx/${DOMAIN}.conf"
    log "2. 宝塔面板扩展配置目录: /www/server/panel/vhost/nginx/extension/${DOMAIN}/"
    log "3. 宝塔面板 Apache 配置: /www/server/panel/vhost/apache/${DOMAIN}.conf"
    log "4. Systemd 服务: /etc/systemd/system/cboard.service"
    log "5. 临时配置文件: /tmp/cboard_*_${DOMAIN}.conf"
    
    read -r -p "确认完全卸载？(yes/no): " confirm
    [[ "$confirm" != "yes" ]] && { log "已取消"; return 0; }
    
    # 停止并禁用服务
    step "停止并禁用服务..."
    systemctl stop cboard 2>/dev/null
    systemctl disable cboard 2>/dev/null
    pkill -9 -f "${PROJECT_DIR}/server" 2>/dev/null
    
    # 释放端口
    if command -v lsof &>/dev/null; then
        local port_pid=$(lsof -ti:8000 2>/dev/null | head -1)
        [[ -n "$port_pid" ]] && kill -9 "$port_pid" 2>/dev/null
    fi
    
    local deleted_count=0
    
    # 删除临时配置文件
    step "删除临时配置文件..."
    for tmp_file in /tmp/cboard_nginx_${DOMAIN}.conf /tmp/cboard_proxy_${DOMAIN}.conf; do
        if [[ -f "$tmp_file" ]]; then
            rm -f "$tmp_file"
            log "✅ 已删除: $tmp_file"
            ((deleted_count++))
        fi
    done
    
    # 删除宝塔面板网站配置文件（Nginx）
    step "删除宝塔面板网站配置..."
    local bt_conf="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
    if [[ -f "$bt_conf" ]]; then
        rm -f "$bt_conf"
        log "✅ 已删除宝塔面板网站配置: $bt_conf"
        ((deleted_count++))
        reload_nginx_force
    fi
    
    # 删除宝塔面板扩展配置目录
    step "删除扩展配置目录..."
    local ext_dir="/www/server/panel/vhost/nginx/extension/${DOMAIN}"
    if [[ -d "$ext_dir" ]] || [[ -e "$ext_dir" ]]; then
        rm -rf "$ext_dir" 2>/dev/null
        sleep 0.5
        if [[ ! -d "$ext_dir" ]] && [[ ! -e "$ext_dir" ]]; then
            log "✅ 已删除扩展配置目录: $ext_dir"
            ((deleted_count++))
        else
            warn "⚠️  扩展配置目录删除失败: $ext_dir"
            warn "   请手动删除: rm -rf $ext_dir"
        fi
    fi
    
    # 删除宝塔面板 Apache 配置文件
    step "删除 Apache 配置..."
    local apache_conf="/www/server/panel/vhost/apache/${DOMAIN}.conf"
    if [[ -f "$apache_conf" ]]; then
        rm -f "$apache_conf"
        log "✅ 已删除 Apache 配置: $apache_conf"
        ((deleted_count++))
        command -v apachectl &>/dev/null && apachectl graceful 2>/dev/null
    fi
    
    # 删除 Systemd 服务文件
    step "删除 Systemd 服务..."
    local svc="/etc/systemd/system/cboard.service"
    if [[ -f "$svc" ]]; then
        systemctl daemon-reload 2>/dev/null
        rm -f "$svc"
        systemctl daemon-reload 2>/dev/null
        log "✅ 已删除 Systemd 服务文件: $svc"
        ((deleted_count++))
    fi
    
    # 询问是否删除 .env 文件
    if [[ -f "${PROJECT_DIR}/.env" ]]; then
        read -r -p "是否删除项目 .env 文件？(yes/no): " del_env
        if [[ "$del_env" == "yes" ]]; then
            rm -f "${PROJECT_DIR}/.env"
            log "✅ 已删除 .env 文件"
            ((deleted_count++))
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
                log "✅ 已删除日志文件: $log_file"
                ((deleted_count++))
            fi
        done
    fi
    
    log "✅ 卸载完成，共删除 $deleted_count 个配置文件/目录"
    warn "⚠️  注意："
    warn "   1. 如果网站仍在宝塔面板中，请手动在宝塔面板中删除网站"
    warn "   2. 项目文件目录 ${PROJECT_DIR} 未被删除，如需完全清理请手动删除"
    warn "   3. 数据库文件 ${PROJECT_DIR}/cboard.db 未被删除，如需清理请手动删除"
}

show_logs() {
    log "展示最近 50 行日志 (Ctrl+C 退出):"
    journalctl -u cboard -n 50 -f
}

# --- 菜单 ---
show_menu() {
    clear
    echo -e "${BLUE}=========================================="
    echo -e "       CBoard Go 一键安装脚本"
    echo -e "==========================================${NC}"
    echo -e "  ${GREEN}1.${NC} 一键全自动部署"
    echo -e "  ${GREEN}2.${NC} 创建/重置管理员账号"
    echo -e "  ${GREEN}3.${NC} 强制重启服务 (杀进程后重启)"
    echo -e "  ${GREEN}4.${NC} 深度清理系统缓存"
    echo -e "  ${GREEN}5.${NC} 解锁用户账户"
    echo -e "------------------------------------------"
    echo -e "  ${CYAN}6.${NC} 查看服务运行状态"
    echo -e "  ${CYAN}7.${NC} 查看实时服务日志"
    echo -e "  ${CYAN}8.${NC} 标准重启服务 (Systemd)"
    echo -e "  ${CYAN}9.${NC} 停止服务"
    echo -e "  ${YELLOW}10.${NC} 卸载网站/删除配置"
    echo -e "  ${RED}0.${NC} 退出"
    echo -e "${BLUE}==========================================${NC}"
    read -r -p "请选择操作 [0-10]: " choice
}

# --- 主程序 ---
main() {
    while true; do
        show_menu
        case $choice in
            1) full_deploy ;;
            2) create_admin ;;
            3) force_restart ;;
            4) deep_clean ;;
            5) unlock_user ;;
            6) manage_service status ;;
            7) show_logs ;;
            8) manage_service restart ;;
            9) manage_service stop ;;
            10) delete_all_configs ;;
            0) exit 0 ;;
            *) error "无效选择，请重新输入" ;;
        esac
        read -r -p "按回车键返回菜单..." temp
    done
}

# 运行检查
[[ "$EUID" -ne 0 ]] && { echo "请使用 root 运行"; exit 1; }
main
