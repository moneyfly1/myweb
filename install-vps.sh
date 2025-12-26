#!/bin/bash
# ============================================
# CBoard Go VPS 一键安装脚本（全自动版）
# ============================================
# 功能：在 VPS 上全自动安装环境和部署系统
# 支持：Ubuntu 18.04+ / Debian 10+ / CentOS 7+
# 说明：非宝塔面板环境，全自动安装，自动处理所有问题
# ============================================

set +e  # 遇到错误不立即退出，允许重试

# --- 颜色定义 ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# --- 日志函数 ---
log() { echo -e "${GREEN}[$(date +'%H:%M:%S')] $1${NC}"; }
warn() { echo -e "${YELLOW}[WARN] $1${NC}"; }
error() { echo -e "${RED}[ERROR] $1${NC}"; }
step() { echo -e "${BLUE}[STEP] $1${NC}"; }

# --- 配置变量 ---
GO_VERSION="1.21.5"
NODE_VERSION="18"
BACKEND_PORT="8000"
LOG_FILE="/tmp/cboard_install_$(date +%Y%m%d_%H%M%S).log"

# 记录日志
exec > >(tee -a "$LOG_FILE")
exec 2>&1

# --- 基础检查 ---
check_root() {
    if [[ "$EUID" -ne 0 ]]; then
        error "请使用 root 用户运行此脚本"
        echo "使用方法: sudo bash install-vps.sh"
        exit 1
    fi
}

detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
        log "检测到操作系统: $OS $OS_VERSION"
    else
        error "无法检测操作系统，请确保系统为 Ubuntu/Debian/CentOS"
        exit 1
    fi
}

# --- 网络检测和代理设置 ---
check_network() {
    step "检测网络连接..."
    
    # 检测是否能访问外网
    if ! curl -s --max-time 5 https://www.google.com > /dev/null 2>&1 && \
       ! curl -s --max-time 5 https://www.baidu.com > /dev/null 2>&1; then
        warn "网络连接可能有问题，但继续尝试安装..."
    else
        log "✅ 网络连接正常"
    fi
}

setup_go_proxy() {
    step "配置 Go 代理（国内加速）..."
    
    # 设置 Go 代理
    export GOPROXY=https://goproxy.cn,direct
    export GOSUMDB=sum.golang.google.cn
    
    # 持久化配置
    if ! grep -q "GOPROXY" /etc/profile; then
        cat >> /etc/profile << 'EOF'
# Go 代理配置
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn
EOF
    fi
    
    # 如果 goproxy.cn 不可用，尝试其他镜像
    if ! curl -s --max-time 3 https://goproxy.cn > /dev/null 2>&1; then
        warn "goproxy.cn 不可用，尝试使用其他镜像..."
        export GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
    fi
    
    log "✅ Go 代理已配置: $GOPROXY"
}

setup_npm_mirror() {
    step "配置 npm 镜像（国内加速）..."
    
    # 尝试多个镜像源
    local mirrors=(
        "https://registry.npmmirror.com"
        "https://registry.npm.taobao.org"
        "https://registry.npmjs.org"
    )
    
    for mirror in "${mirrors[@]}"; do
        if curl -s --max-time 3 "$mirror" > /dev/null 2>&1; then
            npm config set registry "$mirror"
            log "✅ npm 镜像已设置: $mirror"
            return 0
        fi
    done
    
    # 如果都不可用，使用默认
    npm config set registry "https://registry.npmmirror.com"
    warn "使用默认 npm 镜像: https://registry.npmmirror.com"
}

# --- 安装系统依赖 ---
install_system_deps() {
    step "安装系统依赖..."
    
    # 更新包管理器
    if [[ "$OS" == "ubuntu" ]] || [[ "$OS" == "debian" ]]; then
        log "更新 apt 包列表..."
        apt-get update -qq || {
            warn "apt-get update 失败，尝试继续..."
        }
        
        # 安装基础工具
        log "安装基础工具..."
        DEBIAN_FRONTEND=noninteractive apt-get install -y \
            curl wget git build-essential sqlite3 \
            ca-certificates gnupg lsb-release || {
            error "基础工具安装失败"
            exit 1
        }
        
        # 检查并安装 Nginx
        if ! command -v nginx &> /dev/null; then
            log "安装 Nginx..."
            DEBIAN_FRONTEND=noninteractive apt-get install -y nginx || {
                error "Nginx 安装失败"
                exit 1
            }
            systemctl enable nginx
            systemctl start nginx
        else
            log "✅ Nginx 已安装"
        fi
        
        # 检查并安装 Certbot
        if ! command -v certbot &> /dev/null; then
            log "安装 Certbot..."
            DEBIAN_FRONTEND=noninteractive apt-get install -y \
                certbot python3-certbot-nginx || {
                warn "Certbot 安装失败，SSL 证书申请可能失败"
            }
        else
            log "✅ Certbot 已安装"
        fi
        
    elif [[ "$OS" == "centos" ]] || [[ "$OS" == "rhel" ]] || [[ "$OS" == "rocky" ]]; then
        log "更新 yum 包列表..."
        yum makecache fast -q || {
            warn "yum makecache 失败，尝试继续..."
        }
        
        # 安装基础工具
        log "安装基础工具..."
        yum install -y curl wget git gcc gcc-c++ make sqlite \
            ca-certificates || {
            error "基础工具安装失败"
            exit 1
        }
        
        # 检查并安装 Nginx
        if ! command -v nginx &> /dev/null; then
            log "安装 Nginx..."
            # CentOS 7 需要 EPEL
            if [[ "$OS_VERSION" == "7" ]]; then
                yum install -y epel-release
            fi
            yum install -y nginx || {
                error "Nginx 安装失败"
                exit 1
            }
            systemctl enable nginx
            systemctl start nginx
        else
            log "✅ Nginx 已安装"
        fi
        
        # 检查并安装 Certbot
        if ! command -v certbot &> /dev/null; then
            log "安装 Certbot..."
            yum install -y certbot python3-certbot-nginx || {
                warn "Certbot 安装失败，SSL 证书申请可能失败"
            }
        else
            log "✅ Certbot 已安装"
        fi
    else
        error "不支持的操作系统: $OS"
        exit 1
    fi
    
    # 配置防火墙（如果存在）
    configure_firewall
    
    log "✅ 系统依赖安装完成"
}

# --- 配置防火墙 ---
configure_firewall() {
    step "配置防火墙..."
    
    # 检查 firewalld
    if systemctl is-active --quiet firewalld 2>/dev/null; then
        log "配置 firewalld..."
        firewall-cmd --permanent --add-service=http 2>/dev/null
        firewall-cmd --permanent --add-service=https 2>/dev/null
        firewall-cmd --reload 2>/dev/null
        log "✅ firewalld 已配置"
    fi
    
    # 检查 ufw
    if command -v ufw &> /dev/null; then
        log "配置 ufw..."
        ufw allow 80/tcp 2>/dev/null
        ufw allow 443/tcp 2>/dev/null
        log "✅ ufw 已配置"
    fi
    
    # 检查 iptables
    if command -v iptables &> /dev/null && ! systemctl is-active --quiet firewalld 2>/dev/null; then
        log "配置 iptables..."
        iptables -I INPUT -p tcp --dport 80 -j ACCEPT 2>/dev/null
        iptables -I INPUT -p tcp --dport 443 -j ACCEPT 2>/dev/null
        # 保存规则（如果支持）
        if command -v iptables-save &> /dev/null; then
            iptables-save > /etc/iptables.rules 2>/dev/null || true
        fi
        log "✅ iptables 已配置"
    fi
}

# --- 安装 Go 环境 ---
install_go() {
    if command -v go &> /dev/null; then
        GO_VER=$(go version | awk '{print $3}' | sed 's/go//')
        log "Go 已安装，版本: $GO_VER"
        # 检查版本是否符合要求
        local major=$(echo "$GO_VER" | cut -d. -f1)
        local minor=$(echo "$GO_VER" | cut -d. -f2)
        if [[ "$major" -gt 1 ]] || [[ "$major" -eq 1 && "$minor" -ge 21 ]]; then
            setup_go_proxy
            return 0
        else
            warn "Go 版本过低 ($GO_VER)，需要升级..."
        fi
    fi
    
    step "安装 Go $GO_VERSION..."
    
    # 检测架构
    local arch=$(uname -m)
    local go_arch="amd64"
    case $arch in
        x86_64) go_arch="amd64" ;;
        aarch64|arm64) go_arch="arm64" ;;
        armv7l) go_arch="armv6l" ;;
        *) error "不支持的架构: $arch"; exit 1 ;;
    esac
    
    cd /tmp || exit 1
    
    # 尝试多个下载源
    local download_urls=(
        "https://go.dev/dl/go${GO_VERSION}.linux-${go_arch}.tar.gz"
        "https://golang.google.cn/dl/go${GO_VERSION}.linux-${go_arch}.tar.gz"
        "https://mirrors.aliyun.com/golang/go${GO_VERSION}.linux-${go_arch}.tar.gz"
    )
    
    local download_success=false
    for url in "${download_urls[@]}"; do
        log "尝试从 $url 下载..."
        if wget -q --timeout=30 --tries=3 "$url" -O "go${GO_VERSION}.linux-${go_arch}.tar.gz"; then
            download_success=true
            break
        else
            warn "下载失败，尝试下一个源..."
        fi
    done
    
    if [ "$download_success" = false ]; then
        error "Go 下载失败，请检查网络连接"
        exit 1
    fi
    
    # 安装 Go
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "go${GO_VERSION}.linux-${go_arch}.tar.gz" || {
        error "Go 解压失败"
        exit 1
    }
    rm -f "go${GO_VERSION}.linux-${go_arch}.tar.gz"
    
    # 添加到 PATH
    export PATH=$PATH:/usr/local/go/bin
    if ! grep -q "/usr/local/go/bin" /etc/profile; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    fi
    
    # 配置 Go 代理
    setup_go_proxy
    
    # 验证安装
    if command -v go &> /dev/null; then
        log "✅ Go 安装成功: $(go version)"
    else
        error "Go 安装失败"
        exit 1
    fi
    
    cd - > /dev/null
}

# --- 安装 Node.js 环境 ---
install_nodejs() {
    if command -v node &> /dev/null; then
        NODE_VER=$(node -v | sed 's/v//')
        local major=$(echo "$NODE_VER" | cut -d. -f1)
        if [[ "$major" -ge 16 ]]; then
            log "Node.js 已安装，版本: $NODE_VER"
            setup_npm_mirror
            return 0
        else
            warn "Node.js 版本过低 ($NODE_VER)，需要升级..."
        fi
    fi
    
    step "安装 Node.js $NODE_VERSION..."
    
    # 配置 npm 镜像
    setup_npm_mirror
    
    if [[ "$OS" == "ubuntu" ]] || [[ "$OS" == "debian" ]]; then
        # 尝试使用 NodeSource 安装
        if curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash -; then
            DEBIAN_FRONTEND=noninteractive apt-get install -y nodejs || {
                warn "NodeSource 安装失败，尝试二进制安装..."
                install_nodejs_binary
                return 0
            }
        else
            warn "NodeSource 脚本执行失败，尝试二进制安装..."
            install_nodejs_binary
            return 0
        fi
    elif [[ "$OS" == "centos" ]] || [[ "$OS" == "rhel" ]] || [[ "$OS" == "rocky" ]]; then
        # 尝试使用 NodeSource 安装
        if curl -fsSL https://rpm.nodesource.com/setup_${NODE_VERSION}.x | bash -; then
            yum install -y nodejs || {
                warn "NodeSource 安装失败，尝试二进制安装..."
                install_nodejs_binary
                return 0
            }
        else
            warn "NodeSource 脚本执行失败，尝试二进制安装..."
            install_nodejs_binary
            return 0
        fi
    fi
    
    # 验证安装
    if command -v node &> /dev/null; then
        log "✅ Node.js 安装成功: $(node -v)"
        log "✅ npm 版本: $(npm -v)"
        setup_npm_mirror
    else
        error "Node.js 安装失败"
        exit 1
    fi
}

# --- 二进制安装 Node.js（备用方案）---
install_nodejs_binary() {
    step "使用二进制方式安装 Node.js..."
    
    local arch=$(uname -m)
    local node_arch="x64"
    case $arch in
        x86_64) node_arch="x64" ;;
        aarch64|arm64) node_arch="arm64" ;;
        armv7l) node_arch="armv7l" ;;
        *) error "不支持的架构: $arch"; return 1 ;;
    esac
    
    local ver="18.20.4"
    local tar="node-v${ver}-linux-${node_arch}.tar.xz"
    local dir="/usr/local/nodejs18"
    
    cd /tmp || return 1
    
    # 尝试多个下载源
    local download_urls=(
        "https://nodejs.org/dist/v${ver}/${tar}"
        "https://npmmirror.com/mirrors/node/v${ver}/${tar}"
        "https://mirrors.huaweicloud.com/nodejs/v${ver}/${tar}"
    )
    
    local download_success=false
    for url in "${download_urls[@]}"; do
        log "尝试从 $url 下载..."
        if wget -q --timeout=30 --tries=3 "$url" -O "$tar"; then
            download_success=true
            break
        else
            warn "下载失败，尝试下一个源..."
        fi
    done
    
    if [ "$download_success" = false ]; then
        error "Node.js 下载失败"
        return 1
    fi
    
    rm -rf "$dir" "node-v${ver}-linux-${node_arch}"
    tar -xf "$tar" || {
        error "Node.js 解压失败"
        return 1
    }
    mv "node-v${ver}-linux-${node_arch}" "$dir"
    rm -f "$tar"
    
    # 添加到 PATH
    export PATH=$PATH:$dir/bin
    if ! grep -q "$dir/bin" /etc/profile; then
        echo "export PATH=\$PATH:$dir/bin" >> /etc/profile
    fi
    
    cd - > /dev/null
    return 0
}

# --- 检查端口占用 ---
check_port() {
    local port=$1
    if command -v netstat &> /dev/null; then
        if netstat -tuln | grep -q ":$port "; then
            return 1  # 端口被占用
        fi
    elif command -v ss &> /dev/null; then
        if ss -tuln | grep -q ":$port "; then
            return 1  # 端口被占用
        fi
    fi
    return 0  # 端口可用
}

# --- 检查域名解析 ---
check_domain_resolution() {
    step "检查域名解析..."
    
    local domain_ip
    domain_ip=$(dig +short "$DOMAIN" | tail -n1 2>/dev/null || \
                nslookup "$DOMAIN" 2>/dev/null | grep -A1 "Name:" | tail -n1 | awk '{print $2}' || \
                getent hosts "$DOMAIN" 2>/dev/null | awk '{print $1}' | head -n1)
    
    if [[ -z "$domain_ip" ]]; then
        warn "无法解析域名 $DOMAIN，请确保域名已正确配置 DNS"
        warn "继续安装，但 SSL 证书申请可能会失败"
        return 1
    fi
    
    local server_ip
    server_ip=$(curl -s ifconfig.me 2>/dev/null || \
                curl -s ip.sb 2>/dev/null || \
                curl -s icanhazip.com 2>/dev/null || \
                hostname -I | awk '{print $1}')
    
    if [[ -z "$server_ip" ]]; then
        warn "无法获取服务器 IP，跳过域名解析验证"
        return 0
    fi
    
    if [[ "$domain_ip" == "$server_ip" ]]; then
        log "✅ 域名解析正确: $DOMAIN -> $domain_ip"
        return 0
    else
        warn "域名解析可能不正确: $DOMAIN -> $domain_ip (服务器 IP: $server_ip)"
        warn "请确保域名已正确解析到服务器 IP"
        return 1
    fi
}

# --- 获取用户输入 ---
get_user_input() {
    clear
    echo -e "${BLUE}=========================================="
    echo -e "       CBoard Go VPS 一键安装"
    echo -e "==========================================${NC}"
    echo ""
    
    # 输入域名
    while true; do
        read -p "请输入您的域名 (例如: example.com): " DOMAIN
        if [[ -z "$DOMAIN" ]]; then
            warn "域名不能为空，请重新输入"
            continue
        fi
        if [[ ! "$DOMAIN" =~ ^[a-zA-Z0-9][a-zA-Z0-9.-]{0,61}[a-zA-Z0-9]?\.[a-zA-Z]{2,}$ ]]; then
            warn "域名格式不正确，请重新输入"
            continue
        fi
        break
    done
    
    # 输入项目目录
    read -p "请输入项目安装目录 (默认: /opt/cboard): " PROJECT_DIR
    if [[ -z "$PROJECT_DIR" ]]; then
        PROJECT_DIR="/opt/cboard"
    fi
    
    # 输入管理员信息
    echo ""
    echo -e "${CYAN}请输入管理员账户信息:${NC}"
    
    read -p "管理员用户名 (默认: admin): " ADMIN_USERNAME
    if [[ -z "$ADMIN_USERNAME" ]]; then
        ADMIN_USERNAME="admin"
    fi
    
    while true; do
        read -p "管理员邮箱: " ADMIN_EMAIL
        if [[ -z "$ADMIN_EMAIL" ]]; then
            warn "邮箱不能为空，请重新输入"
            continue
        fi
        if [[ ! "$ADMIN_EMAIL" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
            warn "邮箱格式不正确，请重新输入"
            continue
        fi
        break
    done
    
    while true; do
        read -sp "管理员密码 (至少6位): " ADMIN_PASSWORD
        echo ""
        if [[ -z "$ADMIN_PASSWORD" ]]; then
            warn "密码不能为空，请重新输入"
            continue
        fi
        if [[ ${#ADMIN_PASSWORD} -lt 6 ]]; then
            warn "密码长度至少6位，请重新输入"
            continue
        fi
        read -sp "请再次输入密码确认: " ADMIN_PASSWORD_CONFIRM
        echo ""
        if [[ "$ADMIN_PASSWORD" != "$ADMIN_PASSWORD_CONFIRM" ]]; then
            warn "两次输入的密码不一致，请重新输入"
            continue
        fi
        break
    done
    
    log "配置信息确认:"
    log "  域名: $DOMAIN"
    log "  项目目录: $PROJECT_DIR"
    log "  管理员用户名: $ADMIN_USERNAME"
    log "  管理员邮箱: $ADMIN_EMAIL"
    echo ""
    read -p "确认开始安装? (y/n): " CONFIRM
    if [[ "$CONFIRM" != "y" ]] && [[ "$CONFIRM" != "Y" ]]; then
        log "安装已取消"
        exit 0
    fi
}

# --- 下载代码 ---
download_code() {
    step "下载项目代码..."
    
    if [ -d "$PROJECT_DIR" ]; then
        warn "项目目录已存在: $PROJECT_DIR"
        read -p "是否删除现有目录并重新下载? (y/n): " RECREATE
        if [[ "$RECREATE" == "y" ]] || [[ "$RECREATE" == "Y" ]]; then
            rm -rf "$PROJECT_DIR"
        else
            log "使用现有目录"
            return 0
        fi
    fi
    
    mkdir -p "$(dirname "$PROJECT_DIR")"
    cd "$(dirname "$PROJECT_DIR")" || exit 1
    
    # 尝试多个 GitHub 镜像
    local github_urls=(
        "https://github.com/moneyfly1/myweb.git"
        "https://ghproxy.com/https://github.com/moneyfly1/myweb.git"
        "https://ghproxy.net/https://github.com/moneyfly1/myweb.git"
    )
    
    local clone_success=false
    for url in "${github_urls[@]}"; do
        log "尝试从 $url 克隆..."
        if timeout 60 git clone "$url" "$(basename "$PROJECT_DIR")" 2>&1; then
            clone_success=true
            break
        else
            warn "克隆失败，尝试下一个源..."
        fi
    done
    
    if [ "$clone_success" = false ]; then
        error "代码下载失败，请检查网络连接和 Git 配置"
        exit 1
    fi
    
    log "✅ 代码下载完成"
}

# --- 构建项目 ---
build_project() {
    step "构建项目..."
    
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; exit 1; }
    
    # 确保 Go 在 PATH 中
    export PATH=$PATH:/usr/local/go/bin
    
    # 构建后端
    step "编译后端程序..."
    
    # 配置 Go 代理
    setup_go_proxy
    
    # 下载依赖（重试机制）
    local retry_count=0
    local max_retries=3
    while [ $retry_count -lt $max_retries ]; do
        if go mod download 2>&1; then
            log "✅ Go 依赖下载成功"
            break
        else
            retry_count=$((retry_count + 1))
            if [ $retry_count -lt $max_retries ]; then
                warn "依赖下载失败，重试 $retry_count/$max_retries..."
                sleep 2
            else
                warn "依赖下载失败，尝试继续..."
            fi
        fi
    done
    
    # 整理依赖
    go mod tidy 2>&1 || warn "依赖整理失败，尝试继续..."
    
    # 编译
    if go build -o server ./cmd/server/main.go 2>&1; then
        log "✅ 后端编译成功"
    else
        error "后端编译失败，请查看错误信息"
        exit 1
    fi
    
    # 构建前端
    step "构建前端..."
    cd frontend || { error "前端目录不存在"; exit 1; }
    
    # 配置 npm 镜像
    setup_npm_mirror
    
    # 安装依赖（重试机制）
    if [ ! -d "node_modules" ] || [ ! -f "node_modules/.bin/vite" ]; then
        log "安装前端依赖..."
        
        # 清理缓存
        npm cache clean --force 2>/dev/null || true
        
        # 设置环境变量
        export PUPPETEER_SKIP_DOWNLOAD=true
        export PUPPETEER_SKIP_CHROMIUM_DOWNLOAD=true
        
        # 重试安装
        retry_count=0
        while [ $retry_count -lt $max_retries ]; do
            if npm install --legacy-peer-deps 2>&1; then
                log "✅ 前端依赖安装成功"
                break
            else
                retry_count=$((retry_count + 1))
                if [ $retry_count -lt $max_retries ]; then
                    warn "依赖安装失败，重试 $retry_count/$max_retries..."
                    # 尝试切换镜像
                    if [ $retry_count -eq 2 ]; then
                        npm config set registry https://registry.npmjs.org/
                    fi
                    sleep 3
                else
                    error "前端依赖安装失败"
                    exit 1
                fi
            fi
        done
    else
        log "✅ 前端依赖已存在，跳过安装"
    fi
    
    # 构建
    if npm run build 2>&1; then
        log "✅ 前端构建成功"
    else
        error "前端构建失败"
        exit 1
    fi
    
    cd ..
}

# --- 创建环境配置文件 ---
create_env_file() {
    step "创建环境配置文件..."
    
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; exit 1; }
    
    # 创建必要的目录
    mkdir -p uploads/logs
    mkdir -p uploads/files
    chmod -R 755 uploads
    
    if [ ! -f ".env" ]; then
        # 生成随机密钥
        local secret_key
        if command -v openssl &> /dev/null; then
            secret_key=$(openssl rand -hex 32)
        else
            secret_key=$(head -c 32 /dev/urandom | base64 | tr -d "=+/" | cut -c1-32)
        fi
        
        cat > .env << EOF
# 服务器配置
HOST=127.0.0.1
PORT=${BACKEND_PORT}

# 数据库配置
DATABASE_URL=sqlite:///./cboard.db

# JWT 配置（已自动生成随机密钥）
SECRET_KEY=${secret_key}

# CORS 配置
BACKEND_CORS_ORIGINS=https://${DOMAIN},http://${DOMAIN}

# 调试模式
DEBUG=false

# 上传目录
UPLOAD_DIR=uploads
MAX_FILE_SIZE=10485760
EOF
        log "✅ 环境配置文件已创建"
    else
        warn ".env 文件已存在，跳过创建"
    fi
    
    # 确保数据库文件权限
    if [ -f "cboard.db" ]; then
        chmod 666 cboard.db 2>/dev/null || true
    fi
}

# --- 创建管理员账户 ---
create_admin_account() {
    step "创建管理员账户..."
    
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; exit 1; }
    
    export PATH=$PATH:/usr/local/go/bin
    export ADMIN_USERNAME="$ADMIN_USERNAME"
    export ADMIN_EMAIL="$ADMIN_EMAIL"
    export ADMIN_PASSWORD="$ADMIN_PASSWORD"
    
    # 重试机制
    local retry_count=0
    local max_retries=3
    while [ $retry_count -lt $max_retries ]; do
        if go run scripts/create_admin.go 2>&1; then
            log "✅ 管理员账户创建成功"
            log "  用户名: $ADMIN_USERNAME"
            log "  邮箱: $ADMIN_EMAIL"
            return 0
        else
            retry_count=$((retry_count + 1))
            if [ $retry_count -lt $max_retries ]; then
                warn "管理员账户创建失败，重试 $retry_count/$max_retries..."
                sleep 2
            else
                warn "管理员账户创建失败，您可以稍后手动创建"
                warn "运行命令: cd $PROJECT_DIR && go run scripts/create_admin.go"
                return 1
            fi
        fi
    done
}

# --- 配置 Nginx ---
configure_nginx() {
    step "配置 Nginx..."
    
    # 检查 Nginx 是否运行
    if ! systemctl is-active --quiet nginx 2>/dev/null; then
        log "启动 Nginx..."
        systemctl start nginx || {
            error "Nginx 启动失败"
            exit 1
        }
    fi
    
    local nginx_conf="/etc/nginx/sites-available/cboard"
    if [[ "$OS" == "centos" ]] || [[ "$OS" == "rhel" ]] || [[ "$OS" == "rocky" ]]; then
        nginx_conf="/etc/nginx/conf.d/cboard.conf"
    fi
    
    # 创建 HTTP 配置
    mkdir -p "$(dirname "$nginx_conf")"
    cat > "$nginx_conf" << EOF
server {
    listen 80;
    server_name ${DOMAIN};
    root ${PROJECT_DIR}/frontend/dist;
    
    client_max_body_size 10M;
    
    # Let's Encrypt 验证
    location /.well-known/acme-challenge/ {
        root ${PROJECT_DIR};
        allow all;
    }
    
    # API 代理
    location /api/ {
        proxy_pass http://127.0.0.1:${BACKEND_PORT};
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
    
    # 前端静态文件
    location / {
        try_files \$uri \$uri/ /index.html;
        add_header Cache-Control "no-cache, no-store, must-revalidate";
    }
    
    # 静态资源缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
EOF
    
    # 启用配置（Ubuntu/Debian）
    if [[ "$OS" == "ubuntu" ]] || [[ "$OS" == "debian" ]]; then
        if [ ! -L "/etc/nginx/sites-enabled/cboard" ]; then
            ln -s "$nginx_conf" /etc/nginx/sites-enabled/cboard 2>/dev/null || true
        fi
        # 删除默认配置（如果存在）
        rm -f /etc/nginx/sites-enabled/default
    fi
    
    # 测试 Nginx 配置
    if nginx -t 2>&1; then
        systemctl reload nginx || systemctl restart nginx
        log "✅ Nginx 配置完成"
    else
        error "Nginx 配置错误，请检查配置文件"
        exit 1
    fi
    
    # 等待 Nginx 启动
    sleep 2
    
    # 申请 SSL 证书
    step "申请 SSL 证书..."
    
    # 检查域名解析
    check_domain_resolution
    
    # 检查端口 80 是否可访问
    if ! check_port 80; then
        warn "端口 80 被占用，SSL 证书申请可能失败"
    fi
    
    # 尝试申请证书
    if certbot --nginx -d "$DOMAIN" \
        --non-interactive \
        --agree-tos \
        --email "$ADMIN_EMAIL" \
        --quiet 2>&1; then
        log "✅ SSL 证书申请成功"
    else
        warn "SSL 证书申请失败，将使用 HTTP"
        warn "您可以稍后手动申请: certbot --nginx -d $DOMAIN"
    fi
}

# --- 创建 systemd 服务 ---
create_systemd_service() {
    step "创建 systemd 服务..."
    
    local service_file="/etc/systemd/system/cboard.service"
    
    # 获取 Go 路径
    local go_path="/usr/local/go/bin"
    if [ ! -d "$go_path" ]; then
        go_path=$(dirname "$(which go)" 2>/dev/null || echo "/usr/local/go/bin")
    fi
    
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
Environment="PATH=${go_path}:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
Environment="GOPROXY=https://goproxy.cn,direct"
Environment="GOSUMDB=sum.golang.google.cn"

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    log "✅ systemd 服务文件已创建"
}

# --- 启动服务 ---
start_service() {
    step "启动服务..."
    
    # 检查端口是否被占用
    if ! check_port $BACKEND_PORT; then
        warn "端口 $BACKEND_PORT 被占用，尝试停止占用进程..."
        if command -v lsof &> /dev/null; then
            lsof -ti:$BACKEND_PORT | xargs kill -9 2>/dev/null || true
        elif command -v fuser &> /dev/null; then
            fuser -k $BACKEND_PORT/tcp 2>/dev/null || true
        fi
        sleep 2
    fi
    
    # 确保文件权限正确
    chmod +x "$PROJECT_DIR/server" 2>/dev/null || true
    chmod 666 "$PROJECT_DIR/cboard.db" 2>/dev/null || true
    
    systemctl enable cboard
    systemctl start cboard
    
    sleep 5
    
    # 检查服务状态
    local retry_count=0
    while [ $retry_count -lt 5 ]; do
        if systemctl is-active --quiet cboard; then
            log "✅ 服务已成功启动"
            
            # 健康检查
            sleep 2
            if curl -s http://127.0.0.1:$BACKEND_PORT/health > /dev/null 2>&1; then
                log "✅ 服务健康检查通过"
            else
                warn "服务已启动，但健康检查失败，请检查日志"
            fi
            
            return 0
        else
            retry_count=$((retry_count + 1))
            if [ $retry_count -lt 5 ]; then
                warn "服务启动中，等待 $retry_count/5..."
                sleep 2
            fi
        fi
    done
    
    error "服务启动失败，请查看日志: journalctl -u cboard -n 50"
    error "或查看应用日志: tail -f $PROJECT_DIR/server.log"
    exit 1
}

# --- 主安装流程 ---
main() {
    clear
    echo -e "${GREEN}"
    echo "=========================================="
    echo "    CBoard Go VPS 一键安装脚本"
    echo "    全自动安装，自动处理所有问题"
    echo "=========================================="
    echo -e "${NC}"
    
    log "安装日志文件: $LOG_FILE"
    echo ""
    
    # 1. 基础检查
    check_root
    detect_os
    check_network
    
    # 2. 获取用户输入
    get_user_input
    
    # 3. 安装系统依赖（包括 Nginx）
    install_system_deps
    
    # 4. 安装 Go 环境
    install_go
    
    # 5. 安装 Node.js 环境
    install_nodejs
    
    # 6. 下载代码
    download_code
    
    # 7. 创建环境配置
    create_env_file
    
    # 8. 构建项目
    build_project
    
    # 9. 创建管理员账户
    create_admin_account
    
    # 10. 创建 systemd 服务
    create_systemd_service
    
    # 11. 配置 Nginx
    configure_nginx
    
    # 12. 启动服务
    start_service
    
    # 完成
    clear
    echo -e "${GREEN}"
    echo "=========================================="
    echo "    安装完成！"
    echo "=========================================="
    echo -e "${NC}"
    log "访问地址: https://${DOMAIN} (或 http://${DOMAIN})"
    log "管理员登录: https://${DOMAIN}/admin/login"
    log "管理员用户名: $ADMIN_USERNAME"
    log "管理员邮箱: $ADMIN_EMAIL"
    echo ""
    log "常用命令:"
    log "  查看服务状态: systemctl status cboard"
    log "  查看服务日志: journalctl -u cboard -f"
    log "  查看应用日志: tail -f $PROJECT_DIR/server.log"
    log "  重启服务: systemctl restart cboard"
    log "  停止服务: systemctl stop cboard"
    echo ""
    log "安装日志已保存到: $LOG_FILE"
    echo ""
}

# 运行主函数
main
