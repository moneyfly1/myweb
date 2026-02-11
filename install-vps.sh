#!/bin/bash
# ============================================
# CBoard Go VPS 一键安装脚本（全自动优化版）
# ============================================
# 功能：全自动环境安装、系统部署、故障自愈
# 支持：Ubuntu 18.04+ / Debian 10+ / CentOS 7+
# ============================================

set +e # 允许错误重试，配合 trap 捕捉
export LANG=en_US.UTF-8

# --- 全局配置 ---
GO_VERSION="1.21.5"
NODE_VERSION="18"
BACKEND_PORT="8000"
LOG_FILE="/tmp/cboard_install_$(date +%Y%m%d_%H%M%S).log"
PROJECT_DIR_DEFAULT="/opt/cboard"
IS_CHINA_IP=0

# 记录日志
exec > >(tee -a "$LOG_FILE") 2>&1

# --- 基础函数 ---
log() { echo -e "\033[0;32m[INFO] $1\033[0m"; }
warn() { echo -e "\033[1;33m[WARN] $1\033[0m"; }
error() { echo -e "\033[0;31m[ERROR] $1\033[0m"; }
step() { echo -e "\033[0;34m[STEP] $1\033[0m"; }

# 带进度条执行命令（估算时间仅用于进度条动画，不影响实际执行）
run_with_progress() {
    local label="$1" est_sec="${2:-60}" tmpout tmpexit pid start elapsed pct filled empty bar i
    shift 2
    [ $# -eq 0 ] && return 1
    tmpout=$(mktemp) tmpexit=$(mktemp)
    echo 1 > "$tmpexit"
    ( "$@" > "$tmpout" 2>&1; echo $? > "$tmpexit" ) &
    pid=$!
    start=$(date +%s)
    while kill -0 "$pid" 2>/dev/null; do
        elapsed=$(($(date +%s) - start))
        [ "$est_sec" -gt 0 ] && pct=$((elapsed * 90 / est_sec)) || pct=$((elapsed / 2))
        [ "$pct" -gt 90 ] && pct=90
        filled=$((pct / 3)); [ "$filled" -gt 30 ] && filled=30
        empty=$((30 - filled))
        bar=""; for ((i=0;i<filled;i++)); do bar="${bar}#"; done; for ((i=0;i<empty;i++)); do bar="${bar} "; done
        printf "\r\033[0;34m[STEP] %s [%s] %d%%\033[0m" "$label" "$bar" "$pct"
        sleep 1
    done
    wait "$pid" 2>/dev/null
    bar=""; for ((i=0;i<30;i++)); do bar="${bar}#"; done
    printf "\r\033[0;34m[STEP] %s [%s] 100%%\033[0m\n" "$label" "$bar"
    local exitcode; exitcode=$(cat "$tmpexit")
    rm -f "$tmpexit"
    if [ "$exitcode" -ne 0 ]; then
        [ -s "$tmpout" ] && tail -n 30 "$tmpout"
        rm -f "$tmpout"
        return "$exitcode"
    fi
    rm -f "$tmpout"
    return 0
}

handle_error() {
    local line=$1 cmd=$2
    error "脚本执行出错! 位置: 第 $line 行 | 命令: $cmd"
    error "请检查: 1.日志($LOG_FILE) 2.网络 3.磁盘/内存"
    # exit 1  # 视情况是否强制退出，此处保持原逻辑不强制退出，由具体逻辑决定
}
trap 'handle_error $LINENO "$BASH_COMMAND"' ERR

check_root() { [[ "$EUID" -ne 0 ]] && { error "请使用 root 用户运行: sudo bash $0"; exit 1; }; }

# 通用下载函数 (支持多源重试)
download_file() {
    local output=$1; shift
    local urls=("$@")
    for url in "${urls[@]}"; do
        log "尝试下载: $url"
        if wget -q --timeout=30 --tries=3 "$url" -O "$output" && [ -s "$output" ]; then
            return 0
        fi
    done
    return 1
}

# --- 环境检测与准备 ---
detect_system() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID; OS_VERSION=$VERSION_ID
        log "系统检测: $OS $OS_VERSION"
    else
        error "无法检测操作系统"; exit 1
    fi
    
    # 检测架构
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) GO_ARCH="amd64"; NODE_ARCH="x64" ;;
        aarch64|arm64) GO_ARCH="arm64"; NODE_ARCH="arm64" ;;
        armv7l) GO_ARCH="armv6l"; NODE_ARCH="armv7l" ;;
        *) error "不支持的架构: $ARCH"; exit 1 ;;
    esac

    # 检测地域
    local country
    country=$(curl -s --max-time 3 https://ipinfo.io/country || curl -s --max-time 3 https://ifconfig.co/country-iso || echo "")
    if [[ "$country" =~ "CN" ]]; then
        IS_CHINA_IP=1
        log "检测为中国大陆 IP，启用加速镜像"
    else
        log "检测为海外环境 ($country)"
    fi
}

# --- 阿里云盾处理 ---
is_aegis_installed() {
    [ -d /usr/local/aegis ] || ps aux | grep -v grep | grep -qE 'aegis|AliYunDun|aliyun-service'
}

uninstall_aliyun_aegis() {
    step "开始卸载阿里云盾..."
    local script_name="/tmp/uninstall_aegis.sh"
    download_file "$script_name" "http://update.aegis.aliyun.com/download/uninstall.sh" "http://update.aegis.aliyun.com/download/quartz_uninstall.sh"
    chmod +x "$script_name" && "$script_name" 2>/dev/null
    
    # 暴力清理
    pkill -9 aliyun-service 2>/dev/null
    rm -rf /usr/local/aegis* /etc/init.d/agentwatch /usr/sbin/aliyun-service
    
    # 屏蔽 IP
    if command -v iptables &>/dev/null; then
        local ips=("140.205.201.0/28" "140.205.225.192/29" "140.205.225.184/29")
        for ip in "${ips[@]}"; do iptables -I INPUT -s "$ip" -j DROP 2>/dev/null; done
    fi
    log "阿里云盾清理完成"
}

check_aegis_interaction() {
    if is_aegis_installed; then
        warn "检测到阿里云盾，可能干扰安装。"
        echo -e "1) 卸载(推荐)  2) 保留  3) 退出"
        read -p "选择: " ch
        case "$ch" in 1) uninstall_aliyun_aegis ;; 3) exit 0 ;; *) warn "保留云盾继续安装" ;; esac
    fi
}

# --- 依赖安装 ---
install_base_deps() {
    step "安装系统基础依赖..."
    local pkgs_common="curl wget git tar sqlite3 ca-certificates nginx"
    
    if [[ "$OS" =~ (ubuntu|debian) ]]; then
        apt-get update -qq
        DEBIAN_FRONTEND=noninteractive apt-get install -y $pkgs_common build-essential libsqlite3-dev certbot python3-certbot-nginx
    elif [[ "$OS" =~ (centos|rhel|rocky) ]]; then
        [[ "$OS_VERSION" == "7" ]] && yum install -y epel-release
        (command -v dnf &>/dev/null && dnf makecache -q || yum makecache -q)
        yum install -y $pkgs_common gcc gcc-c++ make sqlite-devel certbot python3-certbot-nginx
    fi
    
    # 防火墙
    if command -v firewall-cmd &>/dev/null; then
        firewall-cmd --permanent --add-service=http --add-service=https 2>/dev/null
        firewall-cmd --reload 2>/dev/null
    elif command -v ufw &>/dev/null; then
        ufw allow 80/tcp; ufw allow 443/tcp
    fi
    systemctl enable --now nginx
}

# --- 运行时环境 ---
setup_proxy() {
    if [ "$IS_CHINA_IP" -eq 1 ]; then
        export GOPROXY=https://goproxy.cn,direct
        export GOSUMDB=sum.golang.google.cn
        npm config set registry https://registry.npmmirror.com
        grep -q "GOPROXY" /etc/profile || echo "export GOPROXY=https://goproxy.cn,direct" >> /etc/profile
    else
        export GOPROXY=https://proxy.golang.org,direct
    fi
}

install_go() {
    if command -v go &>/dev/null; then
        local ver; ver=$(go version | awk '{print $3}' | sed 's/go//')
        [[ "${ver%%.*}" -ge 1 && $(echo "$ver" | cut -d. -f2) -ge 21 ]] && { setup_proxy; return; }
    fi
    step "安装 Go $GO_VERSION..."
    
    cd /tmp
    local url="https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    [ "$IS_CHINA_IP" -eq 1 ] && url="https://mirrors.aliyun.com/golang/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    
    if download_file "go.tar.gz" "$url"; then
        rm -rf /usr/local/go && tar -C /usr/local -xzf go.tar.gz && rm go.tar.gz
        export PATH=$PATH:/usr/local/go/bin
        grep -q "/usr/local/go/bin" /etc/profile || echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
        setup_proxy
        log "Go 安装成功: $(go version)"
    else
        error "Go 下载失败"; exit 1
    fi
}

install_node() {
    command -v node &>/dev/null && [[ $(node -v | cut -d. -f1 | tr -d 'v') -ge 16 ]] && { setup_proxy; return; }
    step "安装 Node.js $NODE_VERSION..."
    
    setup_proxy # 设置 registry
    # 优先尝试二进制安装，比编译快且比脚本稳
    local url="https://nodejs.org/dist/v${NODE_VERSION}.20.4/node-v${NODE_VERSION}.20.4-linux-${NODE_ARCH}.tar.xz"
    [ "$IS_CHINA_IP" -eq 1 ] && url="https://npmmirror.com/mirrors/node/v${NODE_VERSION}.20.4/node-v${NODE_VERSION}.20.4-linux-${NODE_ARCH}.tar.xz"
    
    cd /tmp
    if download_file "node.tar.xz" "$url"; then
        mkdir -p /usr/local/nodejs
        tar -xJf node.tar.xz --strip-components=1 -C /usr/local/nodejs
        rm node.tar.xz
        export PATH=$PATH:/usr/local/nodejs/bin
        grep -q "/usr/local/nodejs/bin" /etc/profile || echo 'export PATH=$PATH:/usr/local/nodejs/bin' >> /etc/profile
        log "Node 安装成功: $(node -v)"
    else
        warn "二进制下载失败，尝试 NodeSource..."
        curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash -
        apt-get install -y nodejs || yum install -y nodejs
    fi
}

# --- 构建与部署 ---
check_swap() {
    local mem_gb=$(($(grep MemTotal /proc/meminfo | awk '{print $2}') / 1024 / 1024))
    if [ "$mem_gb" -lt 3 ] && ! grep -q "swap" /proc/swaps; then
        step "内存较小 (${mem_gb}GB)，创建 2GB Swap..."
        dd if=/dev/zero of=/swapfile bs=1M count=2048 status=none
        chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile
        grep -q "/swapfile" /etc/fstab || echo "/swapfile none swap sw 0 0" >> /etc/fstab
    fi
}

deploy_project() {
    step "部署项目..."
    # 1. 下载代码
    if [ -d "$PROJECT_DIR" ]; then
        read -p "目录已存在，是否覆盖? (y/N): " cov
        [[ "$cov" =~ [yY] ]] && rm -rf "$PROJECT_DIR" || return 0
    fi
    mkdir -p "$(dirname "$PROJECT_DIR")" && cd "$(dirname "$PROJECT_DIR")" || exit 1
    
    local mirrors=("https://github.com/moneyfly1/myweb.git" "https://gitclone.com/github.com/moneyfly1/myweb.git")
    local cloned=0
    for url in "${mirrors[@]}"; do
        if git clone --depth 1 "$url" "$(basename "$PROJECT_DIR")"; then cloned=1; break; fi
    done
    [ $cloned -eq 0 ] && { error "代码克隆失败"; exit 1; }

    cd "$PROJECT_DIR" || exit 1
    check_swap
    
    # 2. 编译后端
    export CGO_ENABLED=1 GOGC=100 GOMAXPROCS=1
    go mod download || { go env -w GOPROXY=https://goproxy.cn,direct && go mod download; }
    run_with_progress "编译后端..." 120 timeout 1800 go build -ldflags="-s -w" -trimpath -o server ./cmd/server/main.go || { error "后端编译失败"; exit 1; }
    
    # 3. 编译前端
    cd frontend || exit 1
    local mem_limit=""
    [ $(getconf _PHYS_PAGES) -lt 262144 ] && mem_limit="--max-old-space-size=512" # <1GB 内存限制
    export NODE_OPTIONS="$mem_limit" PUPPETEER_SKIP_DOWNLOAD=true
    
    npm install --legacy-peer-deps || { npm config set registry https://registry.npmjs.org/ && npm install --legacy-peer-deps; }
    run_with_progress "编译前端..." 180 npm run build || { error "前端编译失败"; exit 1; }
    cd ..
    
    # 4. 配置环境
    mkdir -p uploads/{logs,files}
    if [ ! -f .env ]; then
        local secret=$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | base64)
        cat > .env <<EOF
HOST=127.0.0.1
PORT=$BACKEND_PORT
DATABASE_URL=sqlite:///./cboard.db
SECRET_KEY=$secret
BACKEND_CORS_ORIGINS=https://$DOMAIN,http://$DOMAIN
DEBUG=false
UPLOAD_DIR=uploads
MAX_FILE_SIZE=10485760
EOF
    fi
}

create_admin() {
    step "初始化管理员..."
    export ADMIN_USERNAME="$1" ADMIN_EMAIL="$2" ADMIN_PASSWORD="$3"
    cd "$PROJECT_DIR"
    go run scripts/admin_tool.go || warn "管理员创建失败，请手动运行 scripts/admin_tool.go"
}

setup_service() {
    step "配置系统服务..."
    cat > /etc/systemd/system/cboard.service <<EOF
[Unit]
Description=CBoard Service
After=network.target
[Service]
Type=simple
User=root
WorkingDirectory=$PROJECT_DIR
ExecStart=$PROJECT_DIR/server
Restart=always
Environment="PATH=$PATH"
Environment="GOPROXY=$(go env GOPROXY)"
[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable --now cboard
}

setup_nginx() {
    step "配置 Nginx & SSL..."
    local conf="/etc/nginx/sites-enabled/cboard"
    [ -d "/etc/nginx/conf.d" ] && conf="/etc/nginx/conf.d/cboard.conf"
    mkdir -p "$(dirname "$conf")"
    
    # 基础模板
    cat > "$conf" <<EOF
server {
    listen 80; server_name $DOMAIN;
    root $PROJECT_DIR/frontend/dist;
    client_max_body_size 10M;
    location /.well-known/acme-challenge/ { root $PROJECT_DIR; allow all; }
    location /api/ {
        proxy_pass http://127.0.0.1:$BACKEND_PORT;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }
    location / { try_files \$uri \$uri/ /index.html; }
}
EOF
    [ -L /etc/nginx/sites-enabled/default ] && rm /etc/nginx/sites-enabled/default
    nginx -t && systemctl reload nginx
    
    # 申请证书
    certbot certonly --webroot -w "$PROJECT_DIR" -d "$DOMAIN" --non-interactive --agree-tos -m "$ADMIN_EMAIL" || warn "SSL 申请失败，请检查 DNS"
    
    # 若证书存在，启用 HTTPS
    if [ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]; then
        sed -i "/listen 80;/a \    listen 443 ssl;\n    ssl_certificate /etc/letsencrypt/live/$DOMAIN/fullchain.pem;\n    ssl_certificate_key /etc/letsencrypt/live/$DOMAIN/privkey.pem;" "$conf"
        systemctl reload nginx
        log "SSL 配置成功: https://$DOMAIN"
    fi
}

# --- 主逻辑 ---
run_install() {
    clear
    echo "=== CBoard 一键安装 ==="
    read -p "域名 (如 example.com): " DOMAIN
    [[ -z "$DOMAIN" ]] && { error "域名不能为空"; return; }
    read -p "安装目录 (默认 $PROJECT_DIR_DEFAULT): " PROJECT_DIR
    PROJECT_DIR=${PROJECT_DIR:-$PROJECT_DIR_DEFAULT}
    read -p "管理员用户名 (默认 admin): " A_USER
    A_USER=${A_USER:-admin}
    read -p "管理员邮箱: " A_EMAIL
    read -sp "管理员密码 (>=6位): " A_PASS; echo
    
    check_aegis_interaction
    install_base_deps
    install_go
    install_node
    deploy_project
    create_admin "$A_USER" "$A_EMAIL" "$A_PASS"
    setup_service
    setup_nginx
    
    echo "=== 安装完成 ==="
    echo "地址: https://$DOMAIN"
    echo "账号: $A_USER / $A_EMAIL"
}

# --- 菜单功能 ---
get_dir() { echo "${PROJECT_DIR:-$PROJECT_DIR_DEFAULT}"; }

menu_manage_admin() {
    local pd=$(get_dir)
    [ ! -d "$pd" ] && { error "未安装"; return; }
    read -p "新用户名: " u; read -p "新邮箱: " e; read -sp "新密码: " p; echo
    export ADMIN_USERNAME=${u:-admin} ADMIN_EMAIL=$e ADMIN_PASSWORD=$p
    cd "$pd" && go run scripts/admin_tool.go
}

menu_unlock() {
    local pd=$(get_dir)
    read -p "用户名或邮箱: " target
    cd "$pd" && go run scripts/unlock_user.go "$target"
}

menu_clean() {
    local pd=$(get_dir)
    rm -rf "$pd/frontend/dist" "$pd/logs/"* "$pd/server"
    log "缓存已清理"
}

force_restart() {
    pkill -f "server" 2>/dev/null
    systemctl restart cboard
    log "服务已强重启动"
}

# --- 仅构建前端（改前端代码后使用，无需整机重装）---
menu_build_frontend() {
    local pd=$(get_dir)
    [ ! -d "$pd" ] && { error "项目目录不存在，请先执行一键安装"; return 1; }
    [ ! -d "$pd/frontend" ] && { error "frontend 目录不存在"; return 1; }
    step "仅构建前端..."
    rm -rf "$pd/frontend/dist"
    export PATH="${PATH}:/usr/local/go/bin:/usr/local/nodejs/bin"
    local mem_limit=""
    [ $(getconf _PHYS_PAGES 2>/dev/null) -lt 262144 ] && mem_limit="--max-old-space-size=512"
    export NODE_OPTIONS="$mem_limit" PUPPETEER_SKIP_DOWNLOAD=true
    cd "$pd/frontend" || return 1
    if run_with_progress "构建前端..." 180 npm run build; then
        log "前端构建成功"
        nginx -t 2>/dev/null && systemctl reload nginx 2>/dev/null && log "已重载 Nginx，新前端已生效"
    else
        error "前端构建失败"
        return 1
    fi
}

# --- 仅构建后端（改 Go 代码后使用）---
menu_build_backend() {
    local pd=$(get_dir)
    [ ! -d "$pd" ] && { error "项目目录不存在，请先执行一键安装"; return 1; }
    [ ! -f "$pd/cmd/server/main.go" ] && { error "后端入口不存在"; return 1; }
    step "仅构建后端..."
    export PATH="${PATH}:/usr/local/go/bin"
    export CGO_ENABLED=1 GOGC=100 GOMAXPROCS=1
    cd "$pd" || return 1
    if run_with_progress "构建后端..." 120 go build -ldflags="-s -w" -trimpath -o server ./cmd/server/main.go; then
        log "后端构建成功"
        systemctl restart cboard 2>/dev/null && log "已重启 CBoard 服务"
    else
        error "后端构建失败"
        return 1
    fi
}

# --- 构建前后端并重启（改多处代码后使用）---
menu_build_all() {
    local pd=$(get_dir)
    [ ! -d "$pd" ] && { error "项目目录不存在，请先执行一键安装"; return 1; }
    step "构建后端..."
    export PATH="${PATH}:/usr/local/go/bin:/usr/local/nodejs/bin"
    export CGO_ENABLED=1 GOGC=100 GOMAXPROCS=1
    cd "$pd" || return 1
    if ! run_with_progress "构建后端..." 120 go build -ldflags="-s -w" -trimpath -o server ./cmd/server/main.go; then
        error "后端构建失败"; return 1
    fi
    log "后端构建成功"
    step "构建前端..."
    rm -rf "$pd/frontend/dist"
    local mem_limit=""
    [ $(getconf _PHYS_PAGES 2>/dev/null) -lt 262144 ] && mem_limit="--max-old-space-size=512"
    export NODE_OPTIONS="$mem_limit" PUPPETEER_SKIP_DOWNLOAD=true
    cd "$pd/frontend" || return 1
    if ! run_with_progress "构建前端..." 180 npm run build; then
        error "前端构建失败"; return 1
    fi
    log "前端构建成功"
    systemctl restart cboard 2>/dev/null && systemctl reload nginx 2>/dev/null
    log "已重启 CBoard 并重载 Nginx"
}

# --- 卸载项目（可选是否保留环境）---
menu_uninstall() {
    local pd
    pd=$(get_dir)
    
    echo ""
    warn "即将卸载 CBoard 项目（可选择性保留运行环境）。"
    read -p "项目目录 (直接回车使用 $pd): " input_dir
    pd=${input_dir:-$pd}
    
    if [ ! -d "$pd" ]; then
        warn "目录不存在: $pd，可能已卸载或路径错误"
        return 0
    fi
    
    read -p "确定要卸载该项目吗？(y/N): " confirm
    if [[ ! "$confirm" =~ ^[yY]$ ]]; then
        log "已取消"
        return 0
    fi
    
    # 1. 停止并删除 systemd 服务
    step "停止并移除 CBoard 服务..."
    systemctl stop cboard 2>/dev/null
    systemctl disable cboard 2>/dev/null
    rm -f /etc/systemd/system/cboard.service
    systemctl daemon-reload 2>/dev/null
    pkill -9 -f "$pd/server" 2>/dev/null
    log "服务已移除"
    
    # 2. 删除项目目录
    step "删除项目目录: $pd"
    rm -rf "$pd"
    log "项目目录已删除"
    
    # 3. 删除 Nginx 中本项目的配置（不卸载 Nginx 本身）
    local nginx_removed=0
    [ -f /etc/nginx/sites-enabled/cboard ] && rm -f /etc/nginx/sites-enabled/cboard && nginx_removed=1
    [ -f /etc/nginx/sites-available/cboard ] && rm -f /etc/nginx/sites-available/cboard && nginx_removed=1
    [ -f /etc/nginx/conf.d/cboard.conf ] && rm -f /etc/nginx/conf.d/cboard.conf && nginx_removed=1
    if [ "$nginx_removed" -eq 1 ]; then
        nginx -t 2>/dev/null && systemctl reload nginx 2>/dev/null && log "已移除 CBoard 的 Nginx 配置"
    fi
    
    # 4. 是否同时卸载运行环境（Go、Node）
    read -p "是否同时卸载运行环境？(删除 Go、Node，保留 Nginx) [y/N]: " rm_env
    if [[ "$rm_env" =~ ^[yY]$ ]]; then
        step "卸载运行环境..."
        [ -d /usr/local/go ] && rm -rf /usr/local/go && log "已删除 /usr/local/go"
        [ -d /usr/local/nodejs ] && rm -rf /usr/local/nodejs && log "已删除 /usr/local/nodejs"
        warn "Go/Node 已删除，当前终端 PATH 可能仍含旧路径，新开终端生效。"
    else
        log "已保留 Go、Node 环境，可继续用于其他项目或重新安装。"
    fi
    
    log "卸载完成。"
}

show_menu() {
    clear
    echo -e "\033[0;34m=== CBoard 管理脚本 ===\033[0m"
    echo "1. 一键安装 / 重装"
    echo "2. 卸载阿里云盾"
    echo "3. 创建/重置管理员"
    echo "4. 强制重启服务"
    echo "5. 清理系统缓存"
    echo "6. 解锁用户"
    echo "7. 重启 Nginx"
    echo "8. 重启 CBoard"
    echo "9. 查看状态"
    echo "10. 查看日志"
    echo "11. 停止服务"
    echo "12. 卸载项目（可选保留环境）"
    echo "--- 仅更新代码（不跑完整安装）---"
    echo "13. 仅构建前端（改前端后选此项）"
    echo "14. 仅构建后端（改 Go 后选此项）"
    echo "15. 构建前后端并重启"
    echo "0. 退出"
    read -p "请选择: " choice
}

main() {
    check_root
    detect_system
    while true; do
        show_menu
        case "$choice" in
            1) run_install ;;
            2) uninstall_aliyun_aegis ;;
            3) menu_manage_admin ;;
            4) force_restart ;;
            5) menu_clean ;;
            6) menu_unlock ;;
            7) systemctl restart nginx && log "Nginx 已重启" ;;
            8) systemctl restart cboard && log "CBoard 已重启" ;;
            9) systemctl status cboard --no-pager ;;
            10) journalctl -u cboard -n 50 -f ;;
            11) systemctl stop cboard && log "服务已停止" ;;
            12) menu_uninstall ;;
            13) menu_build_frontend ;;
            14) menu_build_backend ;;
            15) menu_build_all ;;
            0) exit 0 ;;
            *) warn "无效选项" ;;
        esac
        read -p "按回车返回..."
    done
}

main