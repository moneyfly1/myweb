#!/bin/bash
# ============================================
# CBoard Go VPS 一键安装脚本（全自动版）
# ============================================
# 功能：在 VPS 上全自动安装环境和部署系统
# 支持：Ubuntu 18.04+ / Debian 10+ / CentOS 7+
# 说明：非宝塔面板环境，全自动安装，自动处理所有问题
# ============================================

set +e  # 遇到错误不立即退出，允许重试

# 错误处理函数
handle_error() {
    local line=$1
    local command=$2
    error "脚本执行出错！"
    error "错误位置: 第 $line 行"
    error "执行的命令: $command"
    error ""
    error "请检查以下内容："
    error "1. 查看安装日志: tail -f $LOG_FILE"
    error "2. 检查网络连接: ping -c 3 8.8.8.8"
    error "3. 检查磁盘空间: df -h"
    error "4. 检查系统资源: free -h"
    error ""
    error "如果问题持续，请提供以下信息："
    error "- 操作系统版本: cat /etc/os-release"
    error "- 错误日志: tail -50 $LOG_FILE"
    exit 1
}

# 设置错误陷阱
trap 'handle_error $LINENO "$BASH_COMMAND"' ERR

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
# 是否国内 IP（1 = 中国大陆，0 = 海外），后续用于决定是否启用国内加速镜像
IS_CHINA_IP=0

# 记录日志
exec > >(tee -a "$LOG_FILE")
exec 2>&1

# 显示日志文件位置（在 main 函数开始时显示）

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

# --- 检测是否安装阿里云盾（安骑士/aegis）---
is_aliyun_aegis_installed() {
    [ -d /usr/local/aegis ] && return 0
    systemctl list-units --type=service --all 2>/dev/null | grep -qiE 'aegis|aliyundun|aliyun\.service' && return 0
    [ -f /etc/init.d/aegis ] && [ -x /etc/init.d/aegis ] && return 0
    ps aux 2>/dev/null | grep -v grep | grep -qE 'aegis|AliYunDun|aliyun-service|aliyundun' && return 0
    return 1
}

# --- 全自动卸载阿里云盾（参考 https://gist.github.com/zhouyanyu/3ec73fe8f0b77ffd6d56a4cc2aa18c32 ）---
uninstall_aliyun_aegis() {
    step "开始全自动卸载阿里云盾（安骑士）..."
    
    # 1. 官方卸载脚本
    local uninstall_sh="/tmp/aegis_uninstall.sh"
    local quartz_sh="/tmp/quartz_uninstall.sh"
    
    for url in "http://update.aegis.aliyun.com/download/uninstall.sh" "http://update2.aegis.aliyun.com/download/uninstall.sh"; do
        if wget -q -O "$uninstall_sh" "$url" 2>/dev/null && [ -s "$uninstall_sh" ]; then
            chmod +x "$uninstall_sh"
            log "执行官方卸载脚本..."
            "$uninstall_sh" 2>/dev/null || true
            break
        fi
    done
    
    for url in "http://update.aegis.aliyun.com/download/quartz_uninstall.sh" "http://update2.aegis.aliyun.com/download/quartz_uninstall.sh"; do
        if wget -q -O "$quartz_sh" "$url" 2>/dev/null && [ -s "$quartz_sh" ]; then
            chmod +x "$quartz_sh"
            log "执行 quartz 卸载脚本..."
            "$quartz_sh" 2>/dev/null || true
            break
        fi
    done
    
    # 2. 删除残留文件与进程
    log "清理云盾进程与残留文件..."
    pkill -9 aliyun-service 2>/dev/null || true
    service aegis stop 2>/dev/null || true
    systemctl stop aegis 2>/dev/null || true
    sleep 1
    rm -f /etc/init.d/agentwatch /usr/sbin/aliyun-service
    rm -rf /usr/local/aegis*
    chkconfig --del aegis 2>/dev/null || true
    
    # 3. 屏蔽云盾 IP（防止自动重装）
    if command -v iptables &> /dev/null; then
        log "屏蔽阿里云盾 IP..."
        for rule in \
            "140.205.201.0/28" "140.205.201.16/29" "140.205.201.32/28" \
            "140.205.225.192/29" "140.205.225.200/30" "140.205.225.184/29" \
            "140.205.225.183/32" "140.205.225.206/32" "140.205.225.205/32" \
            "140.205.225.195/32" "140.205.225.204/32"; do
            iptables -I INPUT -s "$rule" -j DROP 2>/dev/null || true
        done
    fi
    
    rm -f "$uninstall_sh" "$quartz_sh"
    
    if is_aliyun_aegis_installed; then
        warn "卸载执行完毕，但仍有云盾相关进程或文件，请手动检查: ps aux | grep -E 'aliyun|AliYunDun'"
    else
        log "✅ 阿里云盾已卸载并清理完毕"
    fi
}

# --- 安装流程中：若检测到云盾则让用户选择（卸载 / 继续 / 退出）---
check_aliyun_aegis() {
    step "检测阿里云盾（安骑士）..."
    if ! is_aliyun_aegis_installed; then
        log "未检测到阿里云盾，跳过。"
        return 0
    fi
    echo ""
    warn "检测到当前环境已安装「阿里云盾」（安骑士 / aegis），可能占用资源并干扰安装。"
    echo ""
    echo -e "  ${GREEN}1)${NC} 先自动卸载云盾，再继续安装"
    echo -e "  ${YELLOW}2)${NC} 不卸载，直接继续安装（不推荐）"
    echo -e "  ${RED}3)${NC} 退出"
    echo ""
    read -p "请选择 [1/2/3]: " aegis_choice
    case "$aegis_choice" in
        1) uninstall_aliyun_aegis ;;
        2) warn "将不卸载云盾继续安装。" ;;
        3) log "已退出"; exit 0 ;;
        *) warn "无效选择，将不卸载继续安装。" ;;
    esac
    echo ""
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

detect_server_region() {
    step "检测服务器所在区域..."
    
    local country
    country="$(curl -s --max-time 3 https://ipinfo.io/country 2>/dev/null || echo "")"
    if [ -z "$country" ]; then
        country="$(curl -s --max-time 3 https://ifconfig.co/country-iso 2>/dev/null || echo "")"
    fi
    country="$(echo "$country" | tr -d ' \r\n')"
    
    if [ "$country" = "CN" ]; then
        IS_CHINA_IP=1
        log "检测到服务器位于中国大陆 (country=CN)，将启用国内加速镜像。"
    elif [ -n "$country" ]; then
        IS_CHINA_IP=0
        log "检测到服务器国家代码: $country，视为海外环境，不启用国内加速镜像。"
    else
        IS_CHINA_IP=0
        warn "无法根据 IP 检测服务器所在国家，将按海外环境处理（不强制使用国内镜像）。"
    fi
}

setup_go_proxy() {
    step "配置 Go 代理（按地域选择）..."
    
    # 海外环境：使用官方默认代理，避免走国内镜像反而变慢
    if [ "$IS_CHINA_IP" -ne 1 ]; then
        export GOPROXY=https://proxy.golang.org,direct
        export GOSUMDB=sum.golang.org
        log "检测为海外环境，使用官方 Go 代理: $GOPROXY"
        return 0
    fi
    
    # 国内环境：使用国内镜像加速
    export GOPROXY=https://goproxy.cn,direct
    export GOSUMDB=sum.golang.google.cn
    
    # 持久化配置（仅在国内环境写入）
    if ! grep -q "GOPROXY" /etc/profile; then
        cat >> /etc/profile << 'EOF'
# Go 代理配置（国内环境）
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn
EOF
    fi
    
    # 如果 goproxy.cn 不可用，尝试其他镜像
    if ! curl -s --max-time 3 https://goproxy.cn > /dev/null 2>&1; then
        warn "goproxy.cn 不可用，尝试使用阿里云镜像..."
        export GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
    fi
    
    log "✅ Go 代理已配置: $GOPROXY"
}

setup_npm_mirror() {
    step "配置 npm 镜像（按地域选择）..."
    
    # 海外环境：保持 npm 默认源（registry.npmjs.org），不强制改为国内镜像
    if [ "$IS_CHINA_IP" -ne 1 ]; then
        log "检测为海外环境，保留 npm 默认源（registry.npmjs.org），不使用国内镜像。"
        return 0
    fi
    
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
        
        # 安装基础工具（包含 tar，后续解压 Go / Node 等必须）
        log "安装基础工具..."
        DEBIAN_FRONTEND=noninteractive apt-get install -y \
            curl wget git build-essential sqlite3 libsqlite3-dev \
            ca-certificates gnupg lsb-release tar || {
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
        log "更新包列表..."
        (command -v dnf &>/dev/null && dnf makecache -q || yum makecache -q) || {
            warn "包列表更新失败，尝试继续..."
        }
        
        # 安装基础工具（包含 tar，后续解压 Go / Node 等必须）
        log "安装基础工具..."
        yum install -y curl wget git gcc gcc-c++ make sqlite sqlite-devel \
            ca-certificates tar || {
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
        
        # 检查并安装 Certbot（不同发行版/源上包名可能不同，逐个尝试）
        if ! command -v certbot &> /dev/null; then
            log "安装 Certbot..."
            # 尝试启用 EPEL（CentOS 7/8 常见）
            if ! rpm -qa 2>/dev/null | grep -qi "epel-release"; then
                (command -v dnf &>/dev/null && dnf install -y epel-release || yum install -y epel-release) 2>/dev/null || true
            fi
            local certbot_installed=0
            if command -v dnf &> /dev/null; then
                dnf install -y certbot python3-certbot-nginx 2>/dev/null && certbot_installed=1 || true
                if [ "$certbot_installed" -eq 0 ]; then
                    dnf install -y certbot certbot-nginx 2>/dev/null && certbot_installed=1 || true
                fi
            else
                yum install -y certbot python3-certbot-nginx 2>/dev/null && certbot_installed=1 || true
                if [ "$certbot_installed" -eq 0 ]; then
                    yum install -y certbot certbot-nginx 2>/dev/null && certbot_installed=1 || true
                fi
            fi
            if [ "$certbot_installed" -eq 1 ] && command -v certbot &> /dev/null; then
                log "✅ Certbot 安装成功"
            else
                warn "Certbot 安装失败，当前环境将仅自动配置 HTTP，SSL 证书需要后续手动安装"
            fi
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
            }
        else
            warn "NodeSource 脚本执行失败，尝试二进制安装..."
            install_nodejs_binary
        fi
    elif [[ "$OS" == "centos" ]] || [[ "$OS" == "rhel" ]] || [[ "$OS" == "rocky" ]]; then
        # 尝试使用 NodeSource 安装
        if curl -fsSL https://rpm.nodesource.com/setup_${NODE_VERSION}.x | bash -; then
            (command -v dnf &>/dev/null && dnf install -y nodejs || yum install -y nodejs) || {
                warn "NodeSource 安装失败，尝试二进制安装..."
                install_nodejs_binary
            }
        else
            warn "NodeSource 脚本执行失败，尝试二进制安装..."
            install_nodejs_binary
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
    
    # 尝试多个 GitHub 镜像（直连 + 国内镜像，超时 120 秒）
    local github_urls=(
        "https://github.com/moneyfly1/myweb.git"
        "https://gitclone.com/github.com/moneyfly1/myweb.git"
        "https://mirror.ghproxy.com/https://github.com/moneyfly1/myweb.git"
        "https://ghproxy.com/https://github.com/moneyfly1/myweb.git"
        "https://ghproxy.net/https://github.com/moneyfly1/myweb.git"
    )
    
    local clone_success=false
    for url in "${github_urls[@]}"; do
        log "尝试从 $url 克隆..."
        if timeout 120 git clone --depth 1 "$url" "$(basename "$PROJECT_DIR")" 2>&1; then
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

# --- 检查并创建 Swap 空间（低内存优化）---
setup_swap() {
    # 检查内存大小
    local total_mem_kb
    total_mem_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}')
    local total_mem_gb=$((total_mem_kb / 1024 / 1024))
    
    # 如果内存小于 3GB（含 1G/2G 小内存 VPS），创建 swap，避免构建时卡死
    if [ "$total_mem_gb" -lt 3 ]; then
        step "检测到低内存环境 (${total_mem_gb}GB)，创建 Swap 空间以保障构建..."
        
        # 检查是否已有 swap
        if swapon --show 2>/dev/null | grep -q "swapfile"; then
            log "✅ Swap 已存在，跳过创建"
            return 0
        fi
        
        # 检查磁盘空间（至少需要 2GB 用于 swap）
        local available_space_gb
        available_space_gb=$(df -BG / | tail -1 | awk '{print $4}' | sed 's/G//')
        
        if [ "$available_space_gb" -lt 2 ]; then
            warn "磁盘空间不足，无法创建 Swap（需要至少 2GB）"
            return 1
        fi
        
        # 创建 2GB swap 文件
        log "创建 2GB Swap 文件（这可能需要几分钟）..."
        if dd if=/dev/zero of=/swapfile bs=1M count=2048 2>/dev/null; then
            chmod 600 /swapfile
            mkswap /swapfile 2>/dev/null
            swapon /swapfile 2>/dev/null
            
            # 永久启用
            if ! grep -q "/swapfile" /etc/fstab; then
                echo "/swapfile none swap sw 0 0" >> /etc/fstab
            fi
            
            log "✅ Swap 创建成功 (2GB)"
            return 0
        else
            warn "Swap 创建失败，继续尝试编译（可能会很慢）"
            return 1
        fi
    else
        log "内存充足 (${total_mem_gb}GB)，无需创建 Swap"
        return 0
    fi
}

# 获取物理内存 MB（用于低内存时限制并发）
get_total_mem_mb() {
    grep MemTotal /proc/meminfo | awk '{print $2}' | awk '{print int($1/1024)}'
}

# --- 构建项目 ---
build_project() {
    step "构建项目..."
    
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; exit 1; }
    
    # 检查后端入口文件是否存在（必须存在才能编译）
    if [ ! -f "cmd/server/main.go" ]; then
        error "未找到后端入口文件: cmd/server/main.go"
        error "请确认仓库已包含主程序（cmd/server/main.go），或从完整源码部署。"
        error "当前目录: $(pwd)"
        exit 1
    fi
    
    # 检查并创建 Swap（低内存优化）
    setup_swap
    
    # 确保 Go 在 PATH 中
    export PATH=$PATH:/usr/local/go/bin
    
    # 构建后端
    step "编译后端程序..."
    
    # 配置 Go 代理
    setup_go_proxy
    
    # 设置 Go 编译优化选项（减少内存占用，避免 2G 机子构建时 CPU 100% 卡死）
    export GOGC=100   # 降低 GC 频率
    export GOMAXPROCS=1  # 单核编译，降低内存与 CPU 峰值
    # 必须启用 CGO：项目使用 go-sqlite3，禁用会导致运行时 "Binary was compiled with CGO_ENABLED=0" 错误
    export CGO_ENABLED=1
    
    # 下载依赖（带超时和重试机制）
    log "下载 Go 依赖..."
    local retry_count=0
    local max_retries=3
    local download_timeout=600  # 10分钟超时
    
    while [ $retry_count -lt $max_retries ]; do
        # 清理 Go 模块缓存（释放空间）
        if [ $retry_count -eq 0 ]; then
            go clean -modcache 2>/dev/null || true
        fi
        
        if timeout $download_timeout go mod download 2>&1; then
            log "✅ Go 依赖下载成功"
            break
        else
            retry_count=$((retry_count + 1))
            if [ $retry_count -lt $max_retries ]; then
                warn "依赖下载失败，重试 $retry_count/$max_retries..."
                sleep 5
            else
                error "Go 依赖下载超时或失败（可能是内存不足）"
                error ""
                error "建议解决方案："
                error "1. 增加服务器内存到至少 1GB"
                error "2. 或手动编译：cd $PROJECT_DIR && go build -o server ./cmd/server/main.go"
                exit 1
            fi
        fi
    done
    
    # 整理依赖
    go mod tidy 2>&1 || warn "依赖整理失败，尝试继续..."
    
    # 编译（使用低内存优化选项）
    log "开始编译（低内存优化模式）..."
    log "这可能需要 10-30 分钟，请耐心等待..."
    
    # 使用 timeout 防止无限卡住
    local build_timeout=1800  # 30分钟超时
    
    # 编译命令（使用正确的 ldflags 格式）
    # -ldflags="-s -w" 去除符号表和调试信息，减小二进制大小
    # -trimpath 去除路径信息
    if timeout $build_timeout go build -ldflags="-s -w" -trimpath -o server ./cmd/server/main.go 2>&1; then
        log "✅ 后端编译成功"
        
        # 检查编译产物
        if [ -f "server" ]; then
            local server_size
            server_size=$(du -h server | cut -f1)
            log "编译产物大小: $server_size"
        fi
    else
        local exit_code=$?
        if [ $exit_code -eq 124 ]; then
            error "编译超时（超过 30 分钟）"
            error "这通常是因为内存不足导致的"
        else
            error "后端编译失败"
        fi
        
        error ""
        error "建议解决方案："
        error "1. 增加服务器内存到至少 1GB"
        error "2. 手动编译（在内存更大的机器上编译后上传）："
        error "   cd $PROJECT_DIR"
        error "   go build -ldflags='-s -w' -o server ./cmd/server/main.go"
        error "3. 或使用预编译版本（如果有）"
        exit 1
    fi
    
    # 构建前端
    step "构建前端..."
    cd frontend || { error "前端目录不存在"; exit 1; }
    
    # 配置 npm 镜像
    setup_npm_mirror
    
    # 小内存 VPS：限制 Node 内存，避免 npm install / npm run build 时 OOM 卡死
    local mem_mb
    mem_mb=$(get_total_mem_mb 2>/dev/null || echo "2048")
    if [ "$mem_mb" -lt 3072 ]; then
        local node_heap
        if   [ "$mem_mb" -le 512 ];  then node_heap=256   # 极小内存机器（512M 及以下）
        elif [ "$mem_mb" -le 1024 ]; then node_heap=512   # 1G 以内
        else                           node_heap=1024  # 1G~3G 之间
        fi
        export NODE_OPTIONS="${NODE_OPTIONS:+$NODE_OPTIONS }--max-old-space-size=${node_heap}"
        log "低内存模式: 限制 Node 堆内存为 ${node_heap}MB (总内存 ${mem_mb}MB)"
    fi
    
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
        if go run scripts/admin_tool.go 2>&1; then
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
                warn "运行命令: cd $PROJECT_DIR && go run scripts/admin_tool.go"
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
    
    # ---------- SSL 证书：申请 + 若已有证书则同样启用 HTTPS ----------
    step "申请 SSL 证书..."
    check_domain_resolution
    
    mkdir -p "${PROJECT_DIR}/.well-known/acme-challenge"
    chmod -R 755 "${PROJECT_DIR}/.well-known"
    
    # 尝试申请（可能因 CAA/限速等失败；也可能证书已存在且未到期而跳过）
    certbot certonly --webroot -w "${PROJECT_DIR}" -d "$DOMAIN" \
        --non-interactive \
        --agree-tos \
        --email "$ADMIN_EMAIL" \
        2>&1 || true
    
    local ssl_cert="/etc/letsencrypt/live/${DOMAIN}/fullchain.pem"
    local ssl_key="/etc/letsencrypt/live/${DOMAIN}/privkey.pem"
    
    # 只要证书文件存在就为 Nginx 启用 HTTPS（含：本次申请成功、或之前已手动申请过）
    if [ -f "$ssl_cert" ] && [ -f "$ssl_key" ]; then
        log "✅ 使用证书: $ssl_cert"
        step "为 Nginx 配置 HTTPS..."
        cat > "$nginx_conf" << NGINXSSL
server {
    listen 80;
    listen 443 ssl;
    server_name ${DOMAIN};
    ssl_certificate ${ssl_cert};
    ssl_certificate_key ${ssl_key};
    root ${PROJECT_DIR}/frontend/dist;
    client_max_body_size 10M;
    location /.well-known/acme-challenge/ {
        root ${PROJECT_DIR};
        allow all;
    }
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
    location / {
        try_files \$uri \$uri/ /index.html;
        add_header Cache-Control "no-cache, no-store, must-revalidate";
    }
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)\$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
NGINXSSL
        if nginx -t 2>&1; then
            systemctl reload nginx 2>/dev/null || systemctl restart nginx
            log "✅ HTTPS 已启用，可通过 https://${DOMAIN} 访问"
        else
            warn "Nginx 配置测试未通过，请检查: $nginx_conf"
        fi
    else
        warn "未检测到有效 SSL 证书，当前仅支持 HTTP 访问"
        warn "若 Certbot 报错含 \"CAA record ... prevents issuance\"，请在域名 DNS 中为该域名添加 CAA 允许 Let's Encrypt，或删除 CAA 记录后重试。"
        warn "手动申请证书: certbot certonly --webroot -w ${PROJECT_DIR} -d $DOMAIN --agree-tos -m $ADMIN_EMAIL"
        warn "申请成功后，在 Nginx 站点配置中增加 listen 443 ssl 及 ssl_certificate/ssl_certificate_key 后执行: nginx -t && systemctl reload nginx"
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

# --- 一键安装流程（菜单选项 1 调用）---
run_full_install() {
    clear
    echo -e "${GREEN}"
    echo "=========================================="
    echo "    CBoard Go VPS 一键安装"
    echo "    全自动安装，自动处理所有问题"
    echo "=========================================="
    echo -e "${NC}"
    
    log "安装日志文件: $LOG_FILE"
    echo ""
    
    # 1. 基础检查
    check_root
    detect_os
    check_aliyun_aegis
    check_network
    detect_server_region
    
    # 显示系统资源信息
    log "系统资源信息："
    local total_mem
    total_mem=$(free -h | grep Mem | awk '{print $2}')
    local available_mem
    available_mem=$(free -h | grep Mem | awk '{print $7}')
    local disk_space
    disk_space=$(df -h / | tail -1 | awk '{print $4}')
    log "  总内存: $total_mem (可用: $available_mem)"
    log "  磁盘空间: $disk_space"
    
    # 内存警告
    local total_mem_kb
    total_mem_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}')
    local total_mem_gb=$((total_mem_kb / 1024 / 1024))
    if [ "$total_mem_gb" -lt 1 ]; then
        warn "⚠️  检测到低内存环境 (${total_mem_gb}GB)"
        warn "   编译过程可能会很慢，脚本将自动创建 Swap 空间"
        warn "   建议：如果可能，请升级到至少 1GB 内存"
        echo ""
        read -p "是否继续安装? (y/n): " -t 10 continue_install || continue_install="y"
        if [ "$continue_install" != "y" ] && [ "$continue_install" != "Y" ]; then
            log "安装已取消"
            exit 0
        fi
    fi
    
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
    log "若无法访问网站，请检查云服务器安全组/防火墙是否放行 80、443 端口。"
    log "安装日志已保存到: $LOG_FILE"
    echo ""
}

# --- 菜单用项目目录（未安装时默认 /opt/cboard）---
get_menu_project_dir() {
    if [ -n "$PROJECT_DIR" ] && [ -d "$PROJECT_DIR" ]; then
        echo "$PROJECT_DIR"
    elif [ -d "/opt/cboard" ]; then
        echo "/opt/cboard"
    else
        echo "/opt/cboard"
    fi
}

# --- 创建/重置管理员账号（菜单项）---
menu_manage_admin() {
    local pd
    pd="$(get_menu_project_dir)"
    if [ ! -d "$pd" ]; then
        error "项目目录不存在: $pd，请先执行一键安装"
        return 1
    fi
    if ! command -v go &>/dev/null; then
        error "未找到 Go 命令，请先执行一键安装"
        return 1
    fi
    if [ ! -f "$pd/scripts/admin_tool.go" ]; then
        error "未找到 scripts/admin_tool.go，请确认项目已完整部署"
        return 1
    fi
    log "创建/重置管理员账户（项目目录: $pd）..."
    read -r -p "管理员用户名 (留空默认 admin): " admin_username
    admin_username="${admin_username:-admin}"
    read -r -p "管理员邮箱 (留空默认 admin@example.com): " admin_email
    admin_email="${admin_email:-admin@example.com}"
    if [[ ! "$admin_email" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
        error "邮箱格式不正确"
        return 1
    fi
    read -r -sp "管理员密码 (至少6位): " admin_pass
    echo ""
    if [ -z "$admin_pass" ] || [ ${#admin_pass} -lt 6 ]; then
        error "密码至少6位"
        return 1
    fi
    export PATH="${PATH}:/usr/local/go/bin"
    export ADMIN_USERNAME="$admin_username"
    export ADMIN_EMAIL="$admin_email"
    export ADMIN_PASSWORD="$admin_pass"
    cd "$pd" || return 1
    if go run scripts/admin_tool.go 2>&1; then
        log "✅ 管理员账户已创建/重置（用户名: $admin_username）"
    else
        error "创建失败，请检查上方错误信息"
        return 1
    fi
}

# --- 强制重启服务（杀进程后重启，菜单项）---
menu_force_kill_restart() {
    if ! systemctl list-unit-files 2>/dev/null | grep -q "cboard.service"; then
        warn "未找到 cboard 服务，请先执行一键安装"
        return 1
    fi
    log "强制停止相关进程..."
    pkill -9 -f "$(get_menu_project_dir)/server" 2>/dev/null || true
    systemctl stop cboard 2>/dev/null || true
    sleep 2
    if pgrep -f "cboard|server" >/dev/null 2>&1; then
        pkill -9 -f "server" 2>/dev/null || true
        sleep 1
    fi
    log "✅ 进程已清理，正在启动服务..."
    if systemctl start cboard; then
        sleep 2
        if systemctl is-active --quiet cboard; then
            log "✅ CBoard 服务已成功重启"
        else
            error "服务未运行，请查看: journalctl -u cboard -n 50"
        fi
    else
        error "启动失败"
    fi
}

# --- 深度清理系统缓存（菜单项）---
menu_deep_clean() {
    local pd
    pd="$(get_menu_project_dir)"
    if [ ! -d "$pd" ]; then
        warn "项目目录不存在: $pd，跳过"
        return 0
    fi
    log "深度清理系统缓存（项目目录: $pd）..."
    [ -d "$pd/frontend/dist" ] && rm -rf "$pd/frontend/dist" && log "已清理 frontend/dist"
    [ -d "$pd/logs" ] && rm -rf "$pd/logs"/* 2>/dev/null && log "已清理 logs"
    local tmp_count; tmp_count=$(find "$pd" -name "*.tmp" 2>/dev/null | wc -l)
    [ "$tmp_count" -gt 0 ] && find "$pd" -name "*.tmp" -delete 2>/dev/null && log "已清理 $tmp_count 个临时文件"
    [ -f "$pd/server" ] && rm -f "$pd/server" && log "已删除 server 可执行文件"
    log "✅ 缓存清理完毕"
}

# --- 解锁用户/管理员账户（菜单项）---
menu_unlock_user() {
    local pd
    pd="$(get_menu_project_dir)"
    if [ ! -d "$pd" ] || [ ! -f "$pd/scripts/unlock_user.go" ]; then
        error "项目或 scripts/unlock_user.go 不存在，请先执行一键安装"
        return 1
    fi
    if ! command -v go &>/dev/null; then
        error "未找到 Go 命令"
        return 1
    fi
    read -r -p "请输入要解锁的用户名或邮箱: " identifier
    if [ -z "$identifier" ]; then
        error "用户名或邮箱不能为空"
        return 1
    fi
    export PATH="${PATH}:/usr/local/go/bin"
    cd "$pd" || return 1
    if go run scripts/unlock_user.go "$identifier" 2>&1; then
        log "✅ 账户 $identifier 已解锁"
    else
        error "解锁失败，请检查用户名/邮箱及上方错误信息"
        return 1
    fi
}

# --- 查看实时服务日志（菜单项）---
menu_show_logs() {
    if ! systemctl list-unit-files 2>/dev/null | grep -q "cboard.service"; then
        error "服务 cboard 不存在，请先部署"
        return 1
    fi
    log "实时日志 (Ctrl+C 退出):"
    journalctl -u cboard -n 50 -f
}

# --- 交互式主菜单（参考 install.sh，功能对齐）---
show_menu() {
    clear
    echo -e "${BLUE}=========================================="
    echo -e "       CBoard Go VPS 安装/管理脚本"
    echo -e "==========================================${NC}"
    echo -e "  ${GREEN}1.${NC} 一键安装 CBoard（VPS 全自动部署）"
    echo -e "  ${GREEN}2.${NC} 卸载阿里云盾（安骑士）"
    echo -e "  ${GREEN}3.${NC} 创建/重置管理员账号"
    echo -e "  ${GREEN}4.${NC} 强制重启服务（杀进程后重启）"
    echo -e "  ${GREEN}5.${NC} 深度清理系统缓存"
    echo -e "  ${GREEN}6.${NC} 解锁用户/管理员账户"
    echo -e "  ${CYAN}7.${NC} 重启 Nginx"
    echo -e "  ${CYAN}8.${NC} 重启 CBoard 服务"
    echo -e "  ${CYAN}9.${NC} 查看 CBoard 服务状态"
    echo -e "  ${CYAN}10.${NC} 查看实时服务日志"
    echo -e "  ${CYAN}11.${NC} 停止 CBoard 服务"
    echo -e "  ${RED}0.${NC} 退出"
    echo -e "${BLUE}==========================================${NC}"
    read -r -p "请选择操作 [0-11]: " choice
}

main() {
    check_root || exit 1
    while true; do
        show_menu
        case "$choice" in
            1) run_full_install ;;
            2)
                if is_aliyun_aegis_installed; then
                    uninstall_aliyun_aegis
                else
                    log "未检测到阿里云盾，无需卸载。"
                fi
                ;;
            3) menu_manage_admin ;;
            4) menu_force_kill_restart ;;
            5) menu_deep_clean ;;
            6) menu_unlock_user ;;
            7)
                if systemctl restart nginx 2>/dev/null; then
                    log "✅ Nginx 已重启"
                else
                    (command -v nginx &>/dev/null && nginx -s reload) || warn "Nginx 重启失败或未安装"
                fi
                ;;
            8)
                if systemctl list-unit-files 2>/dev/null | grep -q "cboard.service"; then
                    if systemctl restart cboard; then
                        log "✅ CBoard 服务已重启"
                    else
                        error "CBoard 重启失败，请查看: journalctl -u cboard -n 30"
                    fi
                else
                    warn "未找到 cboard 服务，请先执行一键安装"
                fi
                ;;
            9)
                if systemctl list-unit-files 2>/dev/null | grep -q "cboard.service"; then
                    systemctl status cboard --no-pager -l
                else
                    warn "未找到 cboard 服务，请先执行一键安装"
                fi
                ;;
            10) menu_show_logs ;;
            11)
                if systemctl list-unit-files 2>/dev/null | grep -q "cboard.service"; then
                    if systemctl stop cboard; then
                        log "✅ CBoard 服务已停止"
                    else
                        error "停止失败"
                    fi
                else
                    warn "未找到 cboard 服务"
                fi
                ;;
            0) log "已退出"; exit 0 ;;
            *) warn "无效选择，请重新输入" ;;
        esac
        echo ""
        read -r -p "按回车键返回菜单..." temp
    done
}

# 运行主函数
main
