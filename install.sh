#!/bin/bash
# ============================================
# CBoard Go 终极管理脚本 (部署 + 运维 + 修复)
# ============================================

set +e

# 禁用 Git 自动分页，防止更新文件过多时卡在 less 界面等待按键（解决更新卡住问题）
export GIT_PAGER=cat 

# --- 基础配置 (自动检测) ---
# 脚本所在目录即为项目目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="${SCRIPT_DIR}"
# 自动从目录名提取域名（/www/wwwroot/xxx.com → xxx.com）
DOMAIN="$(basename "$PROJECT_DIR")"
GITHUB_REPO="https://github.com/moneyfly1/myweb.git"
LOG_FILE="/var/log/cboard_install.log"

# 运行环境硬性要求（与本仓库实际依赖对齐，见 go.mod / frontend/package.json）
MIN_GO_VERSION="1.25.0"   # go.mod: go 1.25.0
MIN_NODE_MAJOR=20         # vite 7 要求 node ^20.19.0 || >=22.12.0
MIN_NODE_MINOR=19
GO_VERSION="${GO_VERSION:-1.25.0}"
NODE_VERSION="${NODE_VERSION:-22}"

# 数据库备份目录（升级前自动备份，保留最近 N 份）
DB_BACKUP_DIR="${DB_BACKUP_DIR:-/www/backup/cboard}"
DB_BACKUP_KEEP="${DB_BACKUP_KEEP:-10}"

# --- 颜色定义 ---
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; CYAN='\033[0;36m'; NC='\033[0m'

# --- 辅助函数 ---
# 所有输出同时追加到 LOG_FILE（原来是纯屏幕输出，出问题后没有任何记录可查）
_LOG_DIR="$(dirname "$LOG_FILE")"
[[ -d "$_LOG_DIR" ]] || mkdir -p "$_LOG_DIR" 2>/dev/null
_log_to_file() {
    [[ -w "$_LOG_DIR" || -w "$LOG_FILE" ]] || return 0
    printf '%s %s\n' "$(date +'%Y-%m-%d %H:%M:%S')" "$1" >> "$LOG_FILE" 2>/dev/null
}
log()   { echo -e "${GREEN}[$(date +'%H:%M:%S')] $1${NC}"; _log_to_file "[INFO] $1"; }
warn()  { echo -e "${YELLOW}[WARN] $1${NC}"; _log_to_file "[WARN] $1"; }
error() { echo -e "${RED}[ERROR] $1${NC}"; _log_to_file "[ERROR] $1"; }
step()  { echo -e "${BLUE}[STEP] $1${NC}"; _log_to_file "[STEP] $1"; }

# 域名合法性校验：域名会拼进文件路径、nginx 配置与 .env，必须挡住 / 与 .. 等字符
validate_domain() {
    local d="$1"
    if [[ ! "$d" =~ ^[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?$ ]]; then
        error "域名格式不合法: '$d'（只允许字母/数字/点/短横线，且不能以 . 或 - 开头结尾）"
        return 1
    fi
    return 0
}

# 校验版本号：ver_ge 1.25.0 1.24.9 → 0/1
ver_ge() {
    local a="${1#v}" b="${2#v}"
    [[ "$(printf '%s\n%s\n' "$a" "$b" | sort -V | head -1)" == "$b" ]]
}

# --- 带超时的 Redis 重启函数 ---
restart_redis_with_timeout() {
    log "正在重启 Redis 服务..."

    # 检测 Redis 服务名称
    local svc_name=""
    if systemctl list-units --type=service --all 2>/dev/null | grep -q "redis-server"; then
        svc_name="redis-server"
    elif systemctl list-units --type=service --all 2>/dev/null | grep -q "redis\.service"; then
        svc_name="redis"
    fi

    if [ -z "$svc_name" ]; then
        warn "⚠️  未检测到 Redis 服务，跳过重启"
        return 1
    fi

    # 先优雅停止（给 20 秒持久化时间），再启动
    timeout 20 systemctl stop "$svc_name" 2>/dev/null
    sleep 1
    timeout 10 systemctl start "$svc_name" 2>/dev/null

    sleep 2
    # 带密码时 redis-cli 必须带 -a，否则永远「连不上」而误报 Redis 故障
    local auth=""
    if [[ -n "${REDIS_PASSWORD:-}" ]]; then
        auth="$REDIS_PASSWORD"
    elif [[ -f "${PROJECT_DIR}/.env" ]]; then
        auth="$(grep -E '^REDIS_PASSWORD=' "${PROJECT_DIR}/.env" 2>/dev/null | head -1 | cut -d'=' -f2)"
    fi
    local -a auth_args=()
    [[ -n "$auth" ]] && auth_args=(-a "$auth" --no-auth-warning)
    if redis-cli "${auth_args[@]}" ping &> /dev/null; then
        log "✅ Redis 服务已重启并运行正常"
        return 0
    else
        warn "⚠️  Redis 重启后无法连接，请手动检查: systemctl status $svc_name"
        return 1
    fi
}

# --- Redis 缓存配置函数 ---
configure_redis_cache() {
    log "========================================="
    log "Redis 缓存配置（可选，大幅提升性能）"
    log "========================================="
    echo ""
    echo -e "${CYAN}Redis 缓存可以将 GeoIP 查询速度提升 50-100 倍！${NC}"
    echo -e "${CYAN}首次查询: 200-500ms → 缓存命中: 10-50ms${NC}"
    echo ""

    read -r -p "是否启用 Redis 缓存？(y/n，默认: y): " enable_redis
    enable_redis=${enable_redis:-y}

    if [[ "$enable_redis" != "y" && "$enable_redis" != "Y" ]]; then
        log "跳过 Redis 配置（系统仍可正常运行）"
        return 0
    fi

    # 检查 Redis 是否已安装
    if command -v redis-cli &> /dev/null; then
        log "检测到 Redis 已安装"

        # 测试 Redis 连接
        if redis-cli ping &> /dev/null; then
            log "✅ Redis 服务运行正常"
            REDIS_ADDR="localhost:6379"
        else
            warn "Redis 已安装但未运行，正在启动..."
            systemctl start redis-server 2>/dev/null || systemctl start redis 2>/dev/null || service redis-server start 2>/dev/null || service redis start 2>/dev/null
            sleep 2
            if redis-cli ping &> /dev/null; then
                log "✅ Redis 服务已启动"
                REDIS_ADDR="localhost:6379"
            else
                warn "Redis 启动失败，将跳过缓存配置"
                return 0
            fi
        fi
    else
        log "Redis 未安装，正在自动安装..."
        echo ""
        echo -e "${YELLOW}选择安装方式：${NC}"
        echo "1) Docker 安装（推荐，快速简单）"
        echo "2) 系统包管理器安装（apt/yum）"
        echo "3) 跳过安装（稍后手动安装）"
        read -r -p "请选择 (1-3，默认: 1): " install_method
        install_method=${install_method:-1}

        case $install_method in
            1)
                # Docker 安装
                if command -v docker &> /dev/null; then
                    log "使用 Docker 安装 Redis（只绑定 127.0.0.1，避免无密码 Redis 暴露到公网）..."
                    docker run -d --name redis --restart=always -p 127.0.0.1:6379:6379 redis:alpine
                    sleep 3
                    if docker ps | grep -q redis; then
                        log "✅ Redis 容器已启动"
                        REDIS_ADDR="localhost:6379"
                    else
                        error "Redis 容器启动失败"
                        return 0
                    fi
                else
                    error "Docker 未安装，请先安装 Docker 或选择其他安装方式"
                    return 0
                fi
                ;;
            2)
                # 系统包管理器安装
                if command -v apt-get &> /dev/null; then
                    log "使用 apt 安装 Redis..."
                    apt-get update && apt-get install -y redis-server
                    systemctl enable redis-server
                    systemctl start redis-server
                elif command -v yum &> /dev/null; then
                    log "使用 yum 安装 Redis..."
                    yum install -y redis
                    systemctl enable redis
                    systemctl start redis
                else
                    error "不支持的系统，请手动安装 Redis"
                    return 0
                fi
                sleep 2
                if redis-cli ping &> /dev/null; then
                    log "✅ Redis 安装成功"
                    REDIS_ADDR="localhost:6379"
                else
                    error "Redis 安装失败"
                    return 0
                fi
                ;;
            3)
                log "跳过 Redis 安装"
                return 0
                ;;
            *)
                warn "无效选择，跳过 Redis 安装"
                return 0
                ;;
        esac
    fi

    # 询问 Redis 密码
    read -r -p "Redis 是否设置了密码？(y/n，默认: n): " has_password
    has_password=${has_password:-n}

    REDIS_PASSWORD=""
    if [[ "$has_password" == "y" || "$has_password" == "Y" ]]; then
        read -r -s -p "请输入 Redis 密码: " REDIS_PASSWORD
        echo ""
    fi

    # 创建或更新 .env 文件
    local env_file="${PROJECT_DIR}/.env"
    log "正在配置环境变量..."

    # 备份现有 .env 文件
    if [[ -f "$env_file" ]]; then
        cp "$env_file" "${env_file}.backup.$(date +%Y%m%d_%H%M%S)"
        log "已备份现有配置文件"
    fi

    # 移除旧的 Redis 配置（如果存在）
    if [[ -f "$env_file" ]]; then
        sed -i '/^REDIS_ADDR=/d' "$env_file"
        sed -i '/^REDIS_PASSWORD=/d' "$env_file"
        sed -i '/^# Redis 配置/d' "$env_file"
    fi

    # 添加新的 Redis 配置
    {
        echo ""
        echo "# Redis 配置（GeoIP 缓存加速）"
        echo "REDIS_ADDR=${REDIS_ADDR}"
        if [[ -n "$REDIS_PASSWORD" ]]; then
            echo "REDIS_PASSWORD=${REDIS_PASSWORD}"
        fi
    } >> "$env_file"

    log "✅ Redis 配置已保存到 .env 文件"

    # 固定数据库路径为绝对路径：避免因启动目录变化而静默新建数据库导致"数据丢失"
    # （相对路径 ./cboard.db 依赖进程工作目录，换目录启动会生成全新空库）
    if ! grep -q "^DATABASE_URL=" "$env_file" 2>/dev/null; then
        echo "DATABASE_URL=sqlite:///${PROJECT_DIR}/cboard.db" >> "$env_file"
        log "已写入固定数据库路径: ${PROJECT_DIR}/cboard.db"
    else
        log "检测到 .env 已配置 DATABASE_URL，保持原有配置"
    fi

    # 兼容旧单元：早期版本生成的 unit 里没有 EnvironmentFile（靠 os.Getenv 的键会静默失效）。
    # 新模板已直接写入，这里只处理历史遗留单元，且插入一次即可。
    local service_file="/etc/systemd/system/cboard.service"
    if [[ -f "$service_file" ]] && ! grep -q "EnvironmentFile" "$service_file"; then
        cp "$service_file" "${service_file}.backup.$(date +%Y%m%d_%H%M%S)"
        sed -i "0,/^Environment=/s|^Environment=\(.*\)$|Environment=\1\nEnvironmentFile=-${PROJECT_DIR}/.env|" "$service_file"
        systemctl daemon-reload
        log "✅ 已为旧 systemd 单元补上 EnvironmentFile"
    fi

    echo ""
    log "========================================="
    log "Redis 缓存配置完成！"
    log "========================================="
    echo -e "${GREEN}预期性能提升：${NC}"
    echo -e "  • 列表加载: ${YELLOW}10-50秒 → 50-200ms${NC}"
    echo -e "  • 首次查询: ${YELLOW}200-500ms${NC}"
    echo -e "  • 缓存命中: ${YELLOW}10-50ms (80-90% 命中率)${NC}"
    echo ""
}

# --- 检查并更新 Redis 配置（用于更新代码后）---
check_and_update_redis_config() {
    local non_interactive="${1:-false}"
    local env_file="${PROJECT_DIR}/.env"

    # 检查是否已配置 Redis
    if [[ -f "$env_file" ]] && grep -q "^REDIS_ADDR=" "$env_file"; then
        log "检测到已配置 Redis 缓存"

        # 测试 Redis 连接
        local redis_addr=$(grep "^REDIS_ADDR=" "$env_file" | cut -d'=' -f2)
        local redis_host=$(echo "$redis_addr" | cut -d':' -f1)
        local redis_port=$(echo "$redis_addr" | cut -d':' -f2)

        local auth
        auth="$(grep -E '^REDIS_PASSWORD=' "$env_file" 2>/dev/null | head -1 | cut -d'=' -f2)"
        local -a auth_args=()
        [[ -n "$auth" ]] && auth_args=(-a "$auth" --no-auth-warning)
        if command -v redis-cli &> /dev/null; then
            if redis-cli "${auth_args[@]}" -h "$redis_host" -p "$redis_port" ping &> /dev/null; then
                log "✅ Redis 连接正常"
                return 0
            else
                warn "Redis 连接失败，请检查 Redis 服务状态"
                if [[ "$non_interactive" == "true" ]]; then
                    log "同步模式下跳过交互式 Redis 重配置（可在主菜单选择 12 手动配置）"
                else
                    read -r -p "是否重新配置 Redis？(y/n，默认: n): " reconfig
                    if [[ "$reconfig" == "y" || "$reconfig" == "Y" ]]; then
                        configure_redis_cache
                    fi
                fi
            fi
        else
            warn "Redis 客户端未安装"
        fi
    else
        log "检测到代码已更新，包含 Redis 缓存优化功能"
        echo ""
        echo -e "${CYAN}新功能：Redis 缓存可将 GeoIP 查询速度提升 50-100 倍！${NC}"
        if [[ "$non_interactive" == "true" ]]; then
            log "同步模式下跳过交互式 Redis 配置（可在主菜单选择 12 手动配置）"
        else
            read -r -p "是否现在配置 Redis 缓存？(y/n，默认: y): " config_now
            config_now=${config_now:-y}

            if [[ "$config_now" == "y" || "$config_now" == "Y" ]]; then
                configure_redis_cache
            else
                log "跳过 Redis 配置（可稍后运行脚本选择 '配置 Redis 缓存' 选项）"
            fi
        fi
    fi
}

# --- 1. 核心部署逻辑 ---

# 统一保障 .env：缺失时创建；存在时补齐关键项并把相对数据库路径改成绝对路径。
# 为什么必须绝对路径：SQLite 相对路径锚定的是「可执行文件所在目录」，
# go run 跑运维 CLI 时会落到 Go 构建缓存里（用完即删）→ 管理员密码「重置成功」但生产库没变。
ensure_env_file() {
    local env_file="${PROJECT_DIR}/.env"
    local db_abs="${PROJECT_DIR}/cboard.db"
    if [[ ! -f "$env_file" ]]; then
        step "创建 .env 配置文件..."
        local secret
        secret="$(openssl rand -hex 32 2>/dev/null)"
        if [[ -z "$secret" ]]; then
            error "openssl 不可用，无法生成 SECRET_KEY（请先安装 openssl）"
            return 1
        fi
        umask 077
        cat > "$env_file" << EOF
HOST=127.0.0.1
PORT=8000
DEBUG=false
DATABASE_URL=sqlite:///${db_abs}
SECRET_KEY=${secret}
BACKEND_CORS_ORIGINS=https://${DOMAIN}
PROJECT_NAME=CBoard Go
VERSION=1.0.0
API_V1_STR=/api/v1
UPLOAD_DIR=uploads
MAX_FILE_SIZE=10485760
DISABLE_SCHEDULE_TASKS=false
EOF
        log "✅ .env 已创建（HOST=127.0.0.1，仅 nginx 可访问后端）"
    else
        cp "$env_file" "${env_file}.backup.$(date +%Y%m%d_%H%M%S)" 2>/dev/null
        local cur_db
        cur_db="$(grep -E '^DATABASE_URL=' "$env_file" | head -1 | cut -d'=' -f2-)"
        if [[ -z "$cur_db" ]]; then
            echo "DATABASE_URL=sqlite:///${db_abs}" >> "$env_file"
            log "已补写 DATABASE_URL（绝对路径）"
        elif [[ "$cur_db" == sqlite:///* ]] && [[ "${cur_db#sqlite:///}" != /* ]]; then
            # 相对路径（sqlite:///./cboard.db）→ 改成绝对路径
            sed -i "s|^DATABASE_URL=.*|DATABASE_URL=sqlite:///${db_abs}|" "$env_file"
            log "已将相对 DATABASE_URL 修正为绝对路径: ${db_abs}"
        fi
        if ! grep -q '^HOST=' "$env_file"; then
            echo "HOST=127.0.0.1" >> "$env_file"
            log "已补写 HOST=127.0.0.1（避免后端端口直接暴露到公网）"
        elif grep -qE '^HOST=(0\.0\.0\.0|::)$' "$env_file"; then
            warn "检测到 HOST=0.0.0.0：后端端口 $(env_port) 将对公网开放（绕过 nginx/TLS）。"
            warn "   建议改为 HOST=127.0.0.1 并只通过 nginx 访问；若必须直连请自行加防火墙规则。"
        fi
    fi
    chmod 600 "$env_file" 2>/dev/null
    # 备份文件同样收权限并只保留最近 5 份（备份里含 SECRET_KEY）
    chmod 600 "${env_file}".backup.* 2>/dev/null
    ls -1t "${env_file}".backup.* 2>/dev/null | tail -n +6 | xargs -r rm -f
    return 0
}

# 构建后端：先构建到 server.new，健康校验通过后再原子替换，并保留上一版便于回滚。
build_backend() {
    step "编译 Go 程序（CGO_ENABLED=1，SQLite 驱动需要 cgo）..."
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; return 1; }
    if ! command -v cc >/dev/null 2>&1 && ! command -v gcc >/dev/null 2>&1; then
        warn "未检测到 C 编译器，正在尝试安装编译依赖..."
        install_build_deps
    fi
    export CGO_ENABLED=1
    if ! go mod download; then
        warn "go mod download 失败，继续尝试构建（可能是代理/网络问题）"
    fi
    if ! CGO_ENABLED=1 go build -o server.new ./cmd/server/main.go; then
        error "Go 程序编译失败（若提示 requires go >= ${MIN_GO_VERSION}，请先升级 Go）"
        rm -f server.new
        return 1
    fi
    local ver
    ver="$(./server.new --version 2>/dev/null | head -1)"
    [[ -n "$ver" ]] && log "新二进制自检: $ver"
    if [[ -f server ]]; then
        mv -f server "server.bak.$(date +%Y%m%d_%H%M%S)"
    fi
    mv -f server.new server
    chmod +x server
    log "✅ Go 程序编译完成（旧版本已备份为 server.bak.*）"
    return 0
}

# 安装编译依赖（首次部署在最小化系统上必需；否则 CGO 会被静默关闭 → 二进制能编译但一碰 SQLite 就死）
install_build_deps() {
    if command -v apt-get >/dev/null 2>&1; then
        DEBIAN_FRONTEND=noninteractive apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y build-essential
    elif command -v yum >/dev/null 2>&1; then
        yum install -y gcc gcc-c++ make
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y gcc gcc-c++ make
    fi
    command -v cc >/dev/null 2>&1 || command -v gcc >/dev/null 2>&1
}

# Node 保障：本脚本过去完全不检查 Node（新机器上直接 npm: command not found），
# 而前端用的是 vite 7，要求 node ^20.19.0 || >=22.12.0
ensure_node() {
    if command -v node >/dev/null 2>&1; then
        local major minor
        major="$(node -v | sed 's/^v//' | cut -d. -f1)"
        minor="$(node -v | sed 's/^v//' | cut -d. -f2)"
        if [[ "$major" -gt "$MIN_NODE_MAJOR" ]] || { [[ "$major" -eq "$MIN_NODE_MAJOR" ]] && [[ "$minor" -ge "$MIN_NODE_MINOR" ]]; }; then
            log "✅ Node.js $(node -v) 满足前端构建要求（>= v${MIN_NODE_MAJOR}.${MIN_NODE_MINOR}）"
            return 0
        fi
        warn "Node.js $(node -v) 过低，vite 7 需要 >= v${MIN_NODE_MAJOR}.${MIN_NODE_MINOR}，将安装新版"
    else
        step "未检测到 Node.js，正在安装（前端构建需要）..."
    fi
    install_node_binary
}

install_node_binary() {
    local arch node_arch ver="22.12.0"
    arch="$(uname -m)"
    case "$arch" in
        x86_64) node_arch="x64";;
        aarch64|arm64) node_arch="arm64";;
        *) error "不支持的架构: $arch，请手动安装 Node.js >= v${MIN_NODE_MAJOR}.${MIN_NODE_MINOR}"; return 1;;
    esac
    local tmp; tmp="$(mktemp -d)"
    local tar="node-v${ver}-linux-${node_arch}.tar.xz"
    if ! wget -q "https://nodejs.org/dist/v${ver}/${tar}" -O "${tmp}/${tar}"; then
        error "下载 Node.js 失败（请检查网络或手动安装 v${ver}）"
        rm -rf "$tmp"
        return 1
    fi
    if tar -xJf "${tmp}/${tar}" -C "$tmp"; then
        rm -rf /usr/local/nodejs
        mv "${tmp}/node-v${ver}-linux-${node_arch}" /usr/local/nodejs
        ln -sf /usr/local/nodejs/bin/node /usr/local/bin/node
        ln -sf /usr/local/nodejs/bin/npm /usr/local/bin/npm
        ln -sf /usr/local/nodejs/bin/npx /usr/local/bin/npx
        log "✅ Node.js 已安装: $(node -v)"
    else
        error "Node.js 解压失败"
        rm -rf "$tmp"
        return 1
    fi
    rm -rf "$tmp"
    return 0
}

# 构建前端（依赖变更时强制重装：node_modules 存在但 package.json 变了会导致构建失败）
build_frontend() {
    step "构建前端..."
    cd "${PROJECT_DIR}/frontend" || { error "前端目录不存在"; return 1; }
    ensure_node || return 1
    local need_install="no"
    if [[ ! -d node_modules ]]; then
        need_install="yes"
    elif [[ package.json -nt node_modules/.package-lock.json ]] || [[ package-lock.json -nt node_modules/.package-lock.json ]]; then
        need_install="yes"
    fi
    if [[ "$need_install" == "yes" ]]; then
        log "安装/更新前端依赖..."
        if [[ -f package-lock.json ]]; then
            npm ci --no-audit --no-fund || npm install --legacy-peer-deps --no-audit || { error "前端依赖安装失败"; return 1; }
        else
            npm install --legacy-peer-deps --no-audit || { error "前端依赖安装失败"; return 1; }
        fi
    fi
    if npm run build; then
        cd "$PROJECT_DIR" || true
        log "✅ 前端构建成功"
        return 0
    fi
    cd "$PROJECT_DIR" || true
    error "前端构建失败（请确认 Node.js >= v${MIN_NODE_MAJOR}.${MIN_NODE_MINOR}）"
    return 1
}

# systemd 单元：显式写 EnvironmentFile（原来是 Redis 函数顺手 sed 注入，跳过 Redis 就完全不注入）
write_systemd_unit() {
    local service_file="/etc/systemd/system/cboard.service"
    step "写入 systemd 服务..."
    if [[ -f "$service_file" ]]; then
        cp "$service_file" "${service_file}.backup.$(date +%Y%m%d_%H%M%S)"
    fi
    cat > "$service_file" << EOF
[Unit]
Description=CBoard Go Service
After=network.target

[Service]
Type=simple
# 注意：本服务需要写 nginx 站点配置 / 申请证书（域名池「一键配置」），因此保持 root；
# 若改成普通用户，请在 nginx vhost 目录与 ACME webroot 上另行授权。
User=root
WorkingDirectory=${PROJECT_DIR}
ExecStart=${PROJECT_DIR}/server
Restart=always
RestartSec=5
StandardOutput=append:${PROJECT_DIR}/server.log
StandardError=append:${PROJECT_DIR}/server.log
EnvironmentFile=-${PROJECT_DIR}/.env
Environment="PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
Environment="TZ=Asia/Shanghai"
LimitNOFILE=65536
NoNewPrivileges=yes

[Install]
WantedBy=multi-user.target
EOF
    chmod 600 "${PROJECT_DIR}/server.log" 2>/dev/null
    systemctl daemon-reload
    log "✅ systemd 服务已写入（含 EnvironmentFile / LimitNOFILE）"
}

# 为 server.log 配置 logrotate（原来无限增长，可能写满磁盘）
ensure_logrotate() {
    local f=/etc/logrotate.d/cboard
    [[ -f "$f" ]] && return 0
    cat > "$f" << EOF
${PROJECT_DIR}/server.log {
    daily
    rotate 7
    size 20M
    missingok
    notifempty
    copytruncate
}
EOF
    log "已配置日志轮转: $f"
}

# 启动并做业务健康检查（systemctl is-active 只说明进程在，不说明服务可用）
start_and_verify_service() {
    local port; port="$(env_port)"
    systemctl enable cboard >/dev/null 2>&1
    systemctl restart cboard
    local i
    for i in $(seq 1 20); do
        sleep 1
        if curl -fsS --max-time 3 "http://127.0.0.1:${port}/health" >/dev/null 2>&1; then
            log "✅ 服务启动成功并通过健康检查（http://127.0.0.1:${port}/health）"
            return 0
        fi
    done
    error "服务启动后健康检查失败（20 秒内 /health 不通），最近日志："
    tail -n 20 "${PROJECT_DIR}/server.log" 2>/dev/null
    return 1
}

# 升级前备份数据库（脚本过去完全不备份；启动会 AutoMigrate 改 schema，失败无回滚点）
backup_database() {
    local db; db="$(detect_db_path)"
    [[ -n "$db" && -f "$db" ]] || { log "未发现现有数据库，跳过备份"; return 0; }
    mkdir -p "$DB_BACKUP_DIR"
    local out="${DB_BACKUP_DIR}/pre-upgrade-$(date +%Y%m%d_%H%M%S).db"
    if command -v sqlite3 >/dev/null 2>&1; then
        sqlite3 "$db" ".backup '${out}'" 2>/dev/null || cp -f "$db" "$out"
    else
        cp -f "$db" "$out"
    fi
    gzip -f "$out" 2>/dev/null || true
    log "✅ 数据库已备份: ${out}.gz"
    ls -1t "${DB_BACKUP_DIR}"/pre-upgrade-*.db.gz 2>/dev/null | tail -n +$((DB_BACKUP_KEEP+1)) | xargs -r rm -f
    return 0
}

# 安全调用运维 CLI（admin_tool / unlock_user）：
#   1) 显式导出绝对 DATABASE_URL —— 否则相对路径会被锚定到 go run 的构建缓存目录（用完即删），
#      表现为「提示创建成功，但生产库根本没变」；
#   2) 目标库必须已存在，避免静默新建空库。
run_cli_tool() {
    local tool="$1"; shift
    local db; db="$(detect_db_path)"
    if [[ -z "$db" || ! -f "$db" ]]; then
        error "数据库文件不存在（DATABASE_URL 解析结果: '${db:-空}'），已中止以避免写入错误位置"
        return 1
    fi
    local bin
    bin="$(mktemp -d)/$(basename "$tool")"
    if ! go build -o "$bin" "./scripts/${tool}"; then
        error "编译 scripts/${tool} 失败"
        return 1
    fi
    DATABASE_URL="sqlite:///${db}" "$bin" "$@"
    local rc=$?
    rm -rf "$(dirname "$bin")"
    return $rc
}

# 清 Redis 缓存：只删本项目的键前缀，不再 FLUSHDB（同机别的站点可能共用这个 Redis）
redis_clean_app_cache() {
    command -v redis-cli >/dev/null 2>&1 || return 0
    local addr host port auth
    addr="$(grep -E '^REDIS_ADDR=' "${PROJECT_DIR}/.env" 2>/dev/null | head -1 | cut -d'=' -f2)"
    [[ -n "$addr" ]] || return 0
    host="${addr%%:*}"; port="${addr##*:}"
    auth="$(grep -E '^REDIS_PASSWORD=' "${PROJECT_DIR}/.env" 2>/dev/null | head -1 | cut -d'=' -f2)"
    local -a auth_args=()
    [[ -n "$auth" ]] && auth_args=(-a "$auth" --no-auth-warning)
    local prefix total=0
    for prefix in user: system: statistics: subscription: nodes: geoip: revenue_chart: device: order: ; do
        local n
        n="$(redis-cli "${auth_args[@]}" -h "$host" -p "$port" --scan --pattern "${prefix}*" 2>/dev/null | head -500 | tr '\n' ' ' | xargs -r redis-cli "${auth_args[@]}" -h "$host" -p "$port" DEL 2>/dev/null | tail -1)"
        [[ "$n" =~ ^[0-9]+$ ]] && total=$((total+n))
    done
    log "已清理本项目缓存键 ${total} 个（不再 FLUSHDB 整库）"
}

# 回滚到上一个版本（对应应用报错提示里的「回滚到升级前版本」）
rollback_to_previous() {
    step "回滚到升级前版本..."
    local latest
    latest="$(ls -1t "${PROJECT_DIR}"/server.bak.* 2>/dev/null | head -1)"
    if [[ -z "$latest" ]]; then
        error "没有找到备份二进制（${PROJECT_DIR}/server.bak.*），无法回滚"
        return 1
    fi
    local db_backup
    db_backup="$(ls -1t "${DB_BACKUP_DIR}"/pre-upgrade-*.db.gz 2>/dev/null | head -1)"
    log "将回滚二进制: $latest"
    [[ -n "$db_backup" ]] && log "可用数据库备份: $db_backup（如需一并回滚请手动解压替换）"
    read -r -p "确认回滚？(yes/no): " ok
    [[ "$ok" == "yes" ]] || { log "已取消"; return 0; }
    stop_app_processes
    cp -f server "server.bak.failed.$(date +%Y%m%d_%H%M%S)" 2>/dev/null
    mv -f "$latest" "$PROJECT_DIR/server"
    chmod +x "$PROJECT_DIR/server"
    start_and_verify_service
}

# 探测「真正在跑」的 Nginx：二进制、站点配置目录、测试与重载命令、pid 文件。
#
# 为什么必须探测：宝塔机上是 /www/server/nginx/sbin/nginx（pid 在 /www/server/nginx/logs/，
# 且发行版 nginx 的 systemd 单元通常 enabled 但 failed）；纯 VPS 上是 /usr/sbin/nginx +
# /etc/nginx/conf.d。写死任一种都会出现「脚本说重载成功、线上其实没生效」或
# 「kill 掉唯一在跑的 nginx，整机站点全挂」。
NGINX_BIN=""
NGINX_VHOST_DIR=""
NGINX_PID_FILE=""
NGINX_TEST_CMD=""
NGINX_RELOAD_CMD=""

detect_nginx_layout() {
    # 1) 二进制：优先正在跑的进程（/proc 里的真实路径），其次宝塔路径，最后 PATH
    local running=""
    running="$(pgrep -x nginx 2>/dev/null | head -1)"
    if [[ -n "$running" ]] && [[ -r "/proc/$running/exe" ]]; then
        NGINX_BIN="$(readlink -f "/proc/$running/exe" 2>/dev/null)"
    fi
    [[ -x "$NGINX_BIN" ]] || { [[ -x /www/server/nginx/sbin/nginx ]] && NGINX_BIN="/www/server/nginx/sbin/nginx"; }
    [[ -x "$NGINX_BIN" ]] || NGINX_BIN="$(command -v nginx 2>/dev/null)"

    # 2) pid 文件：跟着二进制走，不要去动 /run/nginx.pid（宝塔机上那是个 0 字节残留文件，
    #    过去据此 pkill nginx，会把整机 16 个站点一起打死）
    if [[ "$NGINX_BIN" == /www/server/nginx/* ]]; then
        NGINX_PID_FILE="/www/server/nginx/logs/nginx.pid"
    elif [[ -f /run/nginx.pid ]]; then
        NGINX_PID_FILE="/run/nginx.pid"
    fi

    # 3) 站点配置目录：环境变量 > 宝塔目录（仅当确实在跑宝塔 nginx）> 系统 nginx 目录
    if [[ -n "$NGINX_VHOST_DIR_OVERRIDE" ]]; then
        NGINX_VHOST_DIR="$NGINX_VHOST_DIR_OVERRIDE"
    elif [[ "$NGINX_BIN" == /www/server/nginx/* ]] && [[ -d /www/server/panel/vhost/nginx ]]; then
        NGINX_VHOST_DIR="/www/server/panel/vhost/nginx"
    elif [[ -d /etc/nginx/conf.d ]]; then
        NGINX_VHOST_DIR="/etc/nginx/conf.d"
    elif [[ -d /etc/nginx/sites-enabled ]]; then
        NGINX_VHOST_DIR="/etc/nginx/sites-enabled"
    elif [[ -d /www/server/panel/vhost/nginx ]]; then
        NGINX_VHOST_DIR="/www/server/panel/vhost/nginx"
    else
        NGINX_VHOST_DIR="/etc/nginx/conf.d"
    fi

    # 4) 测试与重载：优先用探测到的二进制；宝塔机 systemd 单元不可用时退回 init.d
    if [[ -x "$NGINX_BIN" ]]; then
        NGINX_TEST_CMD="$NGINX_BIN -t"
        NGINX_RELOAD_CMD="$NGINX_BIN -s reload"
    else
        NGINX_TEST_CMD="nginx -t"
        NGINX_RELOAD_CMD="nginx -s reload"
    fi
    [[ "$NGINX_BIN" == /www/server/nginx/* ]] && [[ -x /etc/init.d/nginx ]] && NGINX_RELOAD_CMD="/etc/init.d/nginx reload"

    log "Nginx 环境: bin=${NGINX_BIN:-未找到} vhost_dir=${NGINX_VHOST_DIR} pid=${NGINX_PID_FILE:-未找到}"
}

site_conf_path() {
    echo "${NGINX_VHOST_DIR}/${DOMAIN}.conf"
}

# 站点配置文件路径（旧函数名保留，供历史调用点使用）
bt_site_conf_path() { site_conf_path; }

# 统一的站点配置渲染：只有这一份模板，避免「首版有 ACME、最终版没有」这类分支漂移。
# 用法：render_site_config https|http [证书目录]
render_site_config() {
    local mode="$1" cert_root="${2:-}"
    local app_port; app_port="$(env_port)"
    local conf; conf="$(site_conf_path)"
    local acme_block="    location ^~ /.well-known/acme-challenge/ {
        root ${PROJECT_DIR};
        default_type text/plain;
        try_files \$uri =404;
    }
"
    local proxy_common="        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection \"upgrade\";
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;"
    mkdir -p "$(dirname "$conf")"
    if [[ -f "$conf" ]]; then
        cp "$conf" "${conf}.backup.$(date +%Y%m%d_%H%M%S)"
    fi
    if [[ "$mode" == "https" && -n "$cert_root" ]]; then
        cat > "$conf" << EOF
# 由 CBoard 安装脚本生成（可重复生成，覆盖前会自动备份为 .backup.<时间戳>）
server {
    listen 80;
    server_name ${DOMAIN};
${acme_block}    location / { return 301 https://\$host\$request_uri; }
}
server {
    listen 443 ssl;
    http2 on;
    server_name ${DOMAIN};
    ssl_certificate ${cert_root}/fullchain.pem;
    ssl_certificate_key ${cert_root}/privkey.pem;
    client_max_body_size 16m;
    root ${PROJECT_DIR}/frontend/dist;
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
${acme_block}
    location /api/ {
        proxy_pass http://127.0.0.1:${app_port};
${proxy_common}
    }
    # 工单附件等 uploads/ 静态文件：必须转发给 Go 服务，
    # 否则会被下面的 SPA fallback 吞掉 → 返回 index.html（HTTP 200，比 404 更难查）
    location /uploads/ {
        proxy_pass http://127.0.0.1:${app_port};
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        expires -1;
    }
    location /repo-sync/ {
        proxy_pass http://127.0.0.1:${app_port};
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
    location /assets/ { expires 1y; add_header Cache-Control "public, immutable"; }
    # index.html 不缓存：否则浏览器启发式缓存会让人看到旧版本页面（引用已删除的 hash 资源 → 白屏）
    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        expires -1;
    }
    location / { try_files \$uri \$uri/ /index.html; }
}
EOF
    else
        cat > "$conf" << EOF
# 由 CBoard 安装脚本生成（HTTP 模式；申请到证书后重跑会自动切到 HTTPS）
server {
    listen 80;
    server_name ${DOMAIN};
    client_max_body_size 16m;
    root ${PROJECT_DIR}/frontend/dist;
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
${acme_block}
    location /api/ {
        proxy_pass http://127.0.0.1:${app_port};
${proxy_common}
    }
    location /uploads/ {
        proxy_pass http://127.0.0.1:${app_port};
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        expires -1;
    }
    location /repo-sync/ {
        proxy_pass http://127.0.0.1:${app_port};
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
    location /assets/ { expires 1y; add_header Cache-Control "public, immutable"; }
    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        expires -1;
    }
    location / { try_files \$uri \$uri/ /index.html; }
}
EOF
    fi
    log "站点配置已写入: $conf"
}

# 从 .env 读取后端端口（默认 8000），保证 nginx 反代与 .env 一致
env_port() {
    local p
    p="$(grep -E '^PORT=' "${PROJECT_DIR}/.env" 2>/dev/null | head -1 | cut -d'=' -f2 | tr -d '[:space:]')"
    [[ "$p" =~ ^[0-9]+$ ]] || p=8000
    echo "$p"
}

# 写配置后必须先 nginx -t；通过才 reload，失败则回滚到最近一次备份
nginx_apply_or_rollback() {
    local conf; conf="$(site_conf_path)"
    if $NGINX_TEST_CMD >/dev/null 2>&1; then
        $NGINX_RELOAD_CMD >/dev/null 2>&1 && log "✅ Nginx 已重载" || warn "Nginx 重载命令执行失败: $NGINX_RELOAD_CMD"
        return 0
    fi
    error "Nginx 配置检测失败（$NGINX_TEST_CMD），已回滚本次修改"
    $NGINX_TEST_CMD 2>&1 | tail -5
    local last_backup
    last_backup="$(ls -1t "${conf}".backup.* 2>/dev/null | head -1)"
    if [[ -n "$last_backup" ]]; then
        cp "$last_backup" "$conf"
        log "已回滚站点配置: $conf ← $last_backup"
        $NGINX_RELOAD_CMD >/dev/null 2>&1
    fi
    return 1
}

reload_nginx_force() {
    log "正在重载 Nginx..."
    if [[ -z "$NGINX_BIN" ]]; then
        detect_nginx_layout
    fi
    $NGINX_TEST_CMD >/dev/null 2>&1 || { warn "nginx -t 未通过，跳过重载"; $NGINX_TEST_CMD 2>&1 | tail -3; return 1; }
    $NGINX_RELOAD_CMD >/dev/null 2>&1 && log "✅ Nginx 已重载（$NGINX_RELOAD_CMD）" || {
        warn "重载失败，尝试 systemctl / init.d"
        systemctl reload nginx 2>/dev/null || /etc/init.d/nginx reload 2>/dev/null || warn "Nginx 重载全部失败，请手动检查"
    }
}

# 只清理 cboard 自己的进程：绝不使用 pkill -f "server|node"
# （宝塔机上 /www/server/... 全是宝塔生态，-f 匹配会连面板、MySQL、nginx、php-fpm 一起杀掉）
stop_app_processes() {
    systemctl stop cboard 2>/dev/null
    # 只匹配「本项目目录下的 server / server.new」：允许后面带参数（用 ( |$) 而非 $，
    # 避免因为启动参数导致匹配不到），同时绝不会命中 /www/server/... 这类路径
    pkill -9 -f "^${PROJECT_DIR}/server( |$)" 2>/dev/null
    pkill -9 -f "^${PROJECT_DIR}/server\.new( |$)" 2>/dev/null
    sleep 1
    return 0
}

full_deploy() {
    CURRENT_STAGE="部署（full_deploy）"
    log "开始全自动部署流程..."

    # 0. 运行环境探测与预检
    validate_domain "$DOMAIN" || exit 1
    detect_nginx_layout
    ensure_env_file || exit 1
    if ! preflight_environment; then
        error "环境预检未通过，部署已中止（现有服务不受影响）"
        return 1
    fi

    # 1. 检查项目文件
    if [ ! -f "$PROJECT_DIR/go.mod" ] || [ ! -f "$PROJECT_DIR/cmd/server/main.go" ]; then
        error "项目目录缺少源码（$PROJECT_DIR/go.mod 或 cmd/server/main.go），请先执行菜单 11 从 GitHub 同步"
        return 1
    fi
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; exit 1; }

    # 2. 备份现有数据库（首次部署时会跳过）
    backup_database

    # 3. 编译后端 + 构建前端
    build_backend || exit 1
    build_frontend || exit 1

    # 4. systemd 单元 + 日志轮转
    write_systemd_unit
    ensure_logrotate

    # 5. Nginx：先 HTTP（放行 ACME），申请证书后再切 HTTPS
    render_site_config http
    nginx_apply_or_rollback || warn "Nginx 配置未通过检测，请检查后再继续"

    # 6. SSL 证书（缺少 certbot 时先安装；失败要能看到原因）
    step "申请 SSL 证书..."
    ensure_certbot_autorenew
    if command -v certbot >/dev/null 2>&1; then
        if ! certbot certonly --webroot -w "${PROJECT_DIR}" -d "${DOMAIN}" \
                --email "admin@${DOMAIN}" --agree-tos --non-interactive \
                --keep-until-expiring 2>&1 | tail -5; then
            warn "SSL 证书申请失败（上方为 certbot 输出），本次按 HTTP 部署"
        fi
    else
        warn "certbot 不可用，本次按 HTTP 部署（可稍后执行菜单 10 续期/签发）"
    fi
    local cert_dir; cert_dir="$(find_cert_dir)"
    if [[ -n "$cert_dir" ]]; then
        render_site_config https "$cert_dir"
        nginx_apply_or_rollback || warn "Nginx HTTPS 配置未通过检测"
        setup_cert_auto_renew_hook
    else
        warn "未找到证书目录，保持 HTTP 配置"
    fi

    # 7. Redis（可选）
    configure_redis_cache
    restart_redis_with_timeout

    # 8. 启动 + 健康检查
    start_and_verify_service || {
        error "服务未能通过健康检查，请查看 ${PROJECT_DIR}/server.log"
        return 1
    }

    log "部署完成！日志文件: $LOG_FILE"
    log "服务状态: systemctl status cboard"
    log "查看日志: tail -n 200 -f ${PROJECT_DIR}/server.log"
    echo ""
    echo -e "${GREEN}========== 访问信息 ==========${NC}"
    if [[ -n "$cert_dir" ]]; then
        echo -e "  前端地址:     https://${DOMAIN}"
        echo -e "  管理员后台:   https://${DOMAIN}/admin"
    else
        echo -e "  前端地址:     http://${DOMAIN}（本次未启用 HTTPS）"
        echo -e "  管理员后台:   http://${DOMAIN}/admin"
    fi
    echo -e "${GREEN}======================================${NC}"
}

# 证书目录：优先直接判定 <live>/<domain>，避免 find 通配命中 xxx-0001 之类的旧 lineage
find_cert_dir() {
    local live="${LETSENCRYPT_LIVE_DIR:-/etc/letsencrypt/live}"
    if [[ -f "${live}/${DOMAIN}/fullchain.pem" ]]; then
        echo "${live}/${DOMAIN}"
        return 0
    fi
    local found
    found="$(find "$live" -maxdepth 1 -type d -name "${DOMAIN}*" 2>/dev/null | head -1)"
    [[ -n "$found" && -f "${found}/fullchain.pem" ]] && { echo "$found"; return 0; }
    echo ""
}

# --- 2. 运维管理功能 ---

manage_admin() {
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; exit 1; }
    log "创建/重置管理员账户..."
    
    # 检查 Go 环境
    if ! command -v go &> /dev/null; then
        error "未找到 Go 命令，请先安装 Go"
        return 1
    fi
    
    # 检查脚本文件是否存在
    if [[ ! -f "./scripts/admin_tool/main.go" ]]; then
        error "脚本文件不存在: ./scripts/admin_tool/main.go"
        return 1
    fi
    
    # 输入用户名
    read -r -p "请输入管理员用户名 (留空使用默认: admin): " admin_username
    if [[ -z "$admin_username" ]]; then
        admin_username="admin"
        log "使用默认用户名: admin"
    fi
    
    # 输入邮箱
    read -r -p "请输入管理员邮箱 (留空使用默认: admin@your-domain.com): " admin_email
    if [[ -z "$admin_email" ]]; then
        admin_email="admin@your-domain.com"
        log "使用默认邮箱: admin@your-domain.com"
    fi
    
    # 验证邮箱格式
    if [[ ! "$admin_email" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
        error "邮箱格式不正确，请重新输入"
        return 1
    fi
    
    # 输入密码（不回显；留空则自动生成强随机密码，不再使用弱默认值 admin123）
    local generated="no"
    read -r -s -p "请输入管理员密码（留空自动生成强随机密码）: " admin_pass
    echo ""
    if [[ -z "$admin_pass" ]]; then
        admin_pass="$(openssl rand -base64 12 2>/dev/null | tr -d '=+/' )"
        generated="yes"
        warn "已自动生成随机密码，请立即记录（脚本结束不会再次显示）"
    fi
    
    # 验证密码长度
    if [[ ${#admin_pass} -lt 6 ]]; then
        error "密码长度至少6位，请重新输入"
        return 1
    fi
    
    # 设置环境变量并执行脚本
    export ADMIN_USERNAME="$admin_username"
    export ADMIN_EMAIL="$admin_email"
    export ADMIN_PASSWORD="$admin_pass"
    
    log "正在执行创建管理员账户脚本（数据库: $(detect_db_path)）..."
    if ADMIN_USERNAME="$admin_username" ADMIN_EMAIL="$admin_email" ADMIN_PASSWORD="$admin_pass" run_cli_tool admin_tool 2>&1; then
        log "✅ 管理员账户已创建/重置"
        log "用户名: $admin_username"
        log "邮箱: $admin_email"
        [[ "$generated" == "yes" ]] && log "本次生成的密码: ${admin_pass}（请立即保存，之后不再显示）"
    else
        error "管理员账户创建/重置失败，请检查上方错误信息"
        return 1
    fi
}

# 升级环境预检：Go 版本 / 磁盘空间。任何一项不满足立即中止（不影响现有服务）。
preflight_environment() {
    local failed=0

    if ! command -v go &>/dev/null; then
        error "未安装 Go，无法构建。请先安装 Go ${MIN_GO_VERSION}+ 再升级。"
        return 1
    fi
    local go_ver
    go_ver="$(go version 2>/dev/null | awk '{print $3}' | tr -d 'go')"
    if ver_ge "$go_ver" "$MIN_GO_VERSION"; then
        log "✅ Go 版本: $go_ver (满足 ≥${MIN_GO_VERSION})"
    else
        warn "Go 版本偏低（当前 $go_ver，go.mod 要求 ${MIN_GO_VERSION}）"
        if [[ "${GOTOOLCHAIN:-auto}" != "local" ]]; then
            log "已启用 GOTOOLCHAIN=auto，构建时 Go 会自动下载匹配的工具链（需能访问 proxy.golang.org）"
        else
            error "GOTOOLCHAIN=local 且 Go 版本过低，无法构建"
            failed=1
        fi
    fi

    if command -v node &>/dev/null; then
        local node_major node_minor
        node_major="$(node -v | sed 's/^v//' | cut -d. -f1)"
        node_minor="$(node -v | sed 's/^v//' | cut -d. -f2)"
        if [[ "$node_major" -gt "$MIN_NODE_MAJOR" ]] || { [[ "$node_major" -eq "$MIN_NODE_MAJOR" ]] && [[ "$node_minor" -ge "$MIN_NODE_MINOR" ]]; }; then
            log "✅ Node.js $(node -v) (满足 ≥v${MIN_NODE_MAJOR}.${MIN_NODE_MINOR})"
        else
            warn "Node.js $(node -v) 低于 vite 7 要求（≥v${MIN_NODE_MAJOR}.${MIN_NODE_MINOR}），构建前端时会自动安装新版"
        fi
    else
        warn "未检测到 Node.js，构建前端时会自动安装 v22.12.0"
    fi

    local free_kb
    free_kb="$(df -Pk "$PROJECT_DIR" 2>/dev/null | awk 'NR==2 {print $4}')"
    if [[ -z "$free_kb" ]]; then
        warn "无法读取磁盘剩余空间（df 不可用或无权限），跳过该项检查"
    elif [[ "$free_kb" -lt 1048576 ]]; then
        error "磁盘剩余空间不足 1GB（当前 $((free_kb/1024))MB），构建与备份可能失败。请清理空间后重试。"
        failed=1
    else
        log "✅ 磁盘空间: $((free_kb/1024))MB 可用"
    fi

    # 残留进程检查（防止库被锁导致迁移失败）
    if pgrep -f "^${PROJECT_DIR}/server( |$)" >/dev/null 2>&1; then
        warn "检测到残留 server 进程，将在停止阶段一并清理。"
    fi

    if [[ "$failed" == "1" ]]; then
        error "环境预检未通过，升级已中止（现有服务不受影响）"
        return 1
    fi
    log "✅ 环境预检通过"
    return 0
}


# 从 .env 推导 SQLite 数据库文件路径
detect_db_path() {
    local db_url
    db_url="$(grep "^DATABASE_URL=" "${PROJECT_DIR}/.env" 2>/dev/null | head -1 | cut -d'=' -f2-)"
    if [[ "$db_url" == sqlite* ]]; then
        local p="${db_url#sqlite:///}"
        p="${p#./}"
        if [[ "$p" != /* ]]; then p="${PROJECT_DIR}/${p}"; fi
        echo "$p"
    else
        echo ""
    fi
}

# ===== 数据库安全守卫 =====
# 原则：install.sh 不做任何数据库备份，也绝不把 GitHub 上的任何数据库文件带到生产环境。
# db_guard_pre  ：同步前检查仓库没有跟踪数据库文件，并记录生产库指纹（仅内存，不产生文件）。
# db_guard_post ：同步后校验生产库原封未动；任何异常立即中止，不进入构建/重启。
DB_GUARD_PATH=""
DB_GUARD_HASH=""
DB_GUARD_EXISTS="no"

db_guard_pre() {
    # 1) 仓库绝不允许跟踪任何数据库文件：
    #    一旦有人误把 cboard.db 提交进 GitHub，git reset --hard 会用 GitHub 的库覆盖生产库，
    #    这里直接拒绝同步。
    local tracked_db
    tracked_db="$(git ls-files | grep -E '\.(db|sqlite|sqlite3)(\.|$)' || true)"
    if [[ -n "$tracked_db" ]]; then
        error "【数据库安全】仓库中检测到被跟踪的数据库文件，禁止同步（可能用 GitHub 的库覆盖生产库）："
        echo "$tracked_db" | sed 's/^/    /'
        return 1
    fi
    # 2) 记录生产库指纹（md5 只做比对，不留备份文件）
    DB_GUARD_PATH="$(detect_db_path)"
    DB_GUARD_HASH=""
    DB_GUARD_EXISTS="no"
    if [[ -n "$DB_GUARD_PATH" ]]; then
        if [[ -f "$DB_GUARD_PATH" ]]; then
            DB_GUARD_EXISTS="yes"
            DB_GUARD_HASH="$(md5sum "$DB_GUARD_PATH" 2>/dev/null | cut -d' ' -f1)"
        fi
        log "数据库指纹已记录: $DB_GUARD_PATH"
    fi
    return 0
}

db_guard_post() {
    [[ -z "$DB_GUARD_PATH" ]] && return 0
    if [[ "$DB_GUARD_EXISTS" == "yes" ]]; then
        if [[ ! -f "$DB_GUARD_PATH" ]]; then
            error "【数据库安全】生产数据库文件丢失！同步操作可能误删了数据库，已中止升级（服务未重启，原进程仍在运行）。"
            return 1
        fi
        local now_hash
        now_hash="$(md5sum "$DB_GUARD_PATH" 2>/dev/null | cut -d' ' -f1)"
        if [[ -n "$now_hash" && "$now_hash" != "$DB_GUARD_HASH" ]]; then
            error "【数据库安全】生产数据库文件内容发生变化！同步操作可能覆盖了数据库，已中止升级（服务未重启）。"
            return 1
        fi
        log "✅ 数据库完整性校验通过（git 同步未触碰生产库）"
    else
        # 同步前不存在（首次部署场景），同步后也不允许出现来自仓库的库文件
        if [[ -f "$DB_GUARD_PATH" ]]; then
            error "【数据库安全】同步后出现了数据库文件（$DB_GUARD_PATH），可能来自 GitHub，已中止升级。"
            return 1
        fi
    fi
    return 0
}

force_kill() {
    # 注意：这里只能用「精确到本项目」的匹配。
    # 历史实现是 pkill -9 server / node，甚至 pkill -9 -f "server|node"，
    # 而 -f 匹配整条命令行：宝塔机上 /www/server/nginx/sbin/nginx、/www/server/mysql/bin/mysqld、
    # /www/server/panel/pyenv/bin/python3 /www/server/panel/BT-Panel 全部命中
    # —— 等于一键打死面板 + MySQL + nginx + php-fpm。
    log "强制停止 CBoard 进程..."
    stop_app_processes

    if pgrep -f "^${PROJECT_DIR}/server( |$)" > /dev/null 2>&1; then
        warn "仍有残留进程，再次清理..."
        pkill -9 -f "^${PROJECT_DIR}/server( |$)" 2>/dev/null
        sleep 1
    fi

    # 重启 Redis 服务确保缓存清除（使用带超时的函数避免卡住）
    restart_redis_with_timeout

    log "✅ 进程已清理（未触碰其它站点/服务）"
}

deep_clean() {
    # 重要：不再删除 frontend/dist 与 ./server。
    # 历史实现会删掉这两者（nginx 的 root 与 systemd 的 ExecStart 都指向它们），
    # 结果「清理缓存」直接把站点清成白屏、之后任何重启都起不来。
    log "正在清理缓存（保留可运行产物；如需重建请用菜单 14）..."

    redis_clean_app_cache
    restart_redis_with_timeout

    # 只清应用自己的日志内容（保留文件，避免 systemd 追加写句柄异常）
    if [[ -f "$PROJECT_DIR/server.log" ]]; then
        : > "$PROJECT_DIR/server.log"
        log "已清空 server.log"
    fi

    local tmp_count
    tmp_count=$(find "$PROJECT_DIR" -name "*.tmp" 2>/dev/null | wc -l)
    if [[ $tmp_count -gt 0 ]]; then
        find "$PROJECT_DIR" -name "*.tmp" -delete 2>/dev/null
        log "已清理 $tmp_count 个临时文件"
    else
        log "未找到临时文件"
    fi

    if command -v go >/dev/null 2>&1; then
        go clean -cache >/dev/null 2>&1 && log "已清理 Go 编译缓存（未清空模块缓存，离线环境仍可构建）"
    fi

    log "✅ 缓存清理完毕（前端 dist 与后端二进制均已保留）"
}

# 只重建并重启（菜单 14）：不改 nginx、不改 unit，避免「只想重启」却把配置覆盖了
rebuild_and_restart() {
    step "重新构建并重启服务..."
    backup_database
    build_backend || return 1
    build_frontend || return 1
    start_and_verify_service
}

# 探测「真正在跑」的 Nginx 重载方式。
#
# 为什么必须探测：宝塔装的 Nginx 跑在 /www/server/nginx/sbin/nginx，而 apt 版 nginx 的
# systemd 单元在宝塔机器上往往是坏的（起不来，也不是对外服务的那个进程）。续期钩子若
# 写死 `systemctl reload nginx`，自动续期成功后新证书不会被加载 —— 客户在证书到期后
# 就会看到「证书无效」，而服务器上一切看起来都正常。线上就是这么踩过一次。
detect_nginx_reload_cmd() {
    if [[ -x /www/server/nginx/sbin/nginx ]]; then
        echo "/www/server/nginx/sbin/nginx -s reload"
    elif command -v systemctl >/dev/null 2>&1 && systemctl list-unit-files 2>/dev/null | grep -q '^nginx\.service'; then
        echo "systemctl reload nginx"
    elif [[ -x /etc/init.d/nginx ]]; then
        echo "/etc/init.d/nginx reload"
    else
        echo "nginx -s reload"
    fi
}

# 配置证书续期后自动重载 Nginx（供 certbot 自动续期时调用）
#
# 与旧实现的两点区别：
#   1) 重载方式按环境探测，并在运行时依次兜底（宝塔 / systemd / init.d / PATH）；
#   2) 内容不同就替换 —— 旧版只在「文件不存在」时创建，写错过一次之后重跑安装脚本
#      也修不回来（历史钩子里写死的 systemctl reload nginx 会一直生效，新证书永不加载）。
setup_cert_auto_renew_hook() {
    local hook_dir="/etc/letsencrypt/renewal-hooks/deploy"
    local hook_file="$hook_dir/reload-nginx.sh"
    local reload_cmd tmp
    reload_cmd="$(detect_nginx_reload_cmd)"
    mkdir -p "$hook_dir"
    tmp="$(mktemp)"
    cat > "$tmp" <<HOOK
#!/bin/bash
# certbot 续期成功后重载 Nginx 以加载新证书（由 CBoard 安装脚本生成，可重复生成）
for c in \\
  "${reload_cmd}" \\
  "/www/server/nginx/sbin/nginx -s reload" \\
  "systemctl reload nginx" \\
  "/etc/init.d/nginx reload" \\
  "nginx -s reload"; do
    if \$c >/dev/null 2>&1; then
        logger -t cboard-certbot "nginx reloaded via: \$c"
        exit 0
    fi
done
logger -t cboard-certbot "WARN: nginx reload 全部失败，新证书可能未生效"
exit 0
HOOK
    chmod +x "$tmp"
    if [[ ! -f "$hook_file" ]] || ! cmp -s "$tmp" "$hook_file"; then
        mv "$tmp" "$hook_file"
        chmod +x "$hook_file"
        log "已配置证书自动续期钩子（重载方式: $reload_cmd）"
    else
        rm -f "$tmp"
        log "证书自动续期钩子已是最新，跳过"
    fi
}

# 确保 certbot 已安装且「定时自动续期」处于开启状态
ensure_certbot_autorenew() {
    if ! command -v certbot >/dev/null 2>&1; then
        log "未检测到 certbot，尝试安装..."
        if command -v apt-get >/dev/null 2>&1; then
            apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y certbot
        elif command -v yum >/dev/null 2>&1; then
            yum install -y certbot
        fi
    fi
    if ! command -v certbot >/dev/null 2>&1; then
        warn "certbot 未安装成功，请手动安装后再续期（apt install certbot）"
        return 1
    fi
    # 定时续期：优先用发行版自带的 systemd timer，否则退回 cron
    if command -v systemctl >/dev/null 2>&1 && systemctl list-unit-files 2>/dev/null | grep -q '^certbot\.timer'; then
        systemctl enable certbot.timer >/dev/null 2>&1 || true
        systemctl start certbot.timer >/dev/null 2>&1 || true
        log "证书自动续期: certbot.timer 已启用"
    elif [[ -d /etc/cron.d ]]; then
        cat > /etc/cron.d/certbot-renew <<'CRON'
# CBoard: 每 12 小时检查一次证书续期（与官方 certbot 包的做法一致）
0 */12 * * * root certbot -q renew
CRON
        chmod 644 /etc/cron.d/certbot-renew
        log "证书自动续期: 已写入 /etc/cron.d/certbot-renew"
    else
        warn "未找到可用的定时机制，请自行配置 certbot 定时续期"
    fi
    setup_cert_auto_renew_hook
    return 0
}

# 在第一个 `location / {` 之前插入一段配置（幂等：已存在则不动）
insert_block_before_spa() {
    local conf="$1" marker="$2" block="$3"
    grep -q "$marker" "$conf" && return 0
    local tmp; tmp="$(mktemp)"
    awk -v block="$block" '
        !inserted && $0 ~ /^[[:space:]]*location[[:space:]]+\/[[:space:]]*\{/ { print block; inserted = 1 }
        { print }
    ' "$conf" > "$tmp" && mv "$tmp" "$conf"
}

# 幂等补齐站点配置里「必须有但历史上被漏掉」的几块：
# /uploads/ 反代（否则附件被 SPA fallback 吞掉返回 HTML）、index.html 不缓存、
# /assets/ 长缓存、client_max_body_size。只插入缺失的块，不动用户其它内容。
ensure_site_conf_blocks() {
    local conf; conf="$(site_conf_path)"
    [ -f "$conf" ] || { warn "未找到 Nginx 站点配置: $conf"; return 0; }
    local port; port="$(env_port)"
    local changed="no"

    if ! grep -q "client_max_body_size" "$conf"; then
        cp "$conf" "${conf}.backup.$(date +%Y%m%d_%H%M%S)"
        local tmp; tmp="$(mktemp)"
        awk -v line="    client_max_body_size 16m;" '
            !inserted && /^[[:space:]]*server[[:space:]]*\{/ { print; print line; inserted = 1; next }
            { print }
        ' "$conf" > "$tmp" && mv "$tmp" "$conf"
        changed="yes"
    fi

    local uploads_block
    uploads_block="$(cat << BLK
    location /uploads/ {
        proxy_pass http://127.0.0.1:${port};
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        expires -1;
    }
BLK
)"
    if ! grep -q "location /uploads/" "$conf"; then
        insert_block_before_spa "$conf" "location /uploads/" "$uploads_block" && changed="yes"
    fi

    if ! grep -q "location = /index.html" "$conf"; then
        insert_block_before_spa "$conf" "location = /index.html" '    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        expires -1;
    }' && changed="yes"
    fi

    if ! grep -q "location /assets/" "$conf"; then
        insert_block_before_spa "$conf" "location /assets/" '    location /assets/ { expires 1y; add_header Cache-Control "public, immutable"; }' && changed="yes"
    fi

    if ! grep -q "location \.well-known/acme-challenge" "$conf"; then
        warn "站点配置缺少 ACME challenge 放行段（/.well-known/acme-challenge/），证书自动续期可能失败"
        warn "   若需要自动续期，请手动在 80 端口 server 块内加入：location ^~ /.well-known/acme-challenge/ { root ${PROJECT_DIR}; }"
    fi

    if [[ "$changed" == "yes" ]]; then
        log "已补齐站点配置缺失片段"
        nginx_apply_or_rollback || warn "补齐后的配置未通过检测，已回滚"
    fi
    return 0
}

# 站点配置刷新：
#   - 本脚本生成的配置（带生成标记）→ 直接用当前模板重渲染（模板已包含全部必需片段）
#   - 手工/面板维护的配置 → 只做幂等补块，绝不整文件覆盖
refresh_site_config() {
    local conf; conf="$(site_conf_path)"
    if [[ -f "$conf" ]] && grep -q "由 CBoard 安装脚本生成" "$conf"; then
        local cert_dir; cert_dir="$(find_cert_dir)"
        if [[ -n "$cert_dir" ]]; then
            render_site_config https "$cert_dir"
        else
            render_site_config http
        fi
        nginx_apply_or_rollback || warn "站点配置未通过 nginx -t，已回滚"
    else
        ensure_site_conf_blocks
    fi
}

ensure_repo_sync_nginx() {
    local bt_path; bt_path="$(site_conf_path)"
    [ -f "$bt_path" ] || { warn "未找到 Nginx 站点配置: $bt_path"; return 0; }

    if grep -q "location /repo-sync/" "$bt_path"; then
        return 0
    fi

    log "检测到 Nginx 配置缺少 /repo-sync/ 转发，正在自动修复..."
    cp "$bt_path" "${bt_path}.backup.$(date +%Y%m%d_%H%M%S)"
    python3 - "$bt_path" <<'PY'
import sys
from pathlib import Path

path = Path(sys.argv[1])
text = path.read_text()
if 'location /repo-sync/' in text:
    sys.exit(0)

repo_block = '''    location /repo-sync/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
'''

lines = text.split('\n')
out = []
inserted = False
for line in lines:
    stripped = line.strip()
    if stripped.startswith('location / '):
        out.append(repo_block.rstrip('\n'))
        out.append(line)
        inserted = True
    else:
        out.append(line)

if not inserted:
    sys.exit(1)
path.write_text('\n'.join(out))
PY
    if [ $? -eq 0 ]; then
        log "✅ /repo-sync/ 转发已添加"
        nginx_apply_or_rollback || warn "配置未通过 nginx -t，已回滚"
    else
        warn "自动修改失败，请手动在站点配置中 location / 之前添加 /repo-sync/ 反向代理到 127.0.0.1:$(env_port)"
    fi
}

renew_cert() {
    log "证书续期（Let's Encrypt）..."
    if ! command -v certbot &>/dev/null; then
        error "未安装 certbot，请先执行「一键全自动部署」或安装 certbot"
        return 1
    fi

    detect_nginx_layout
    local conf; conf="$(site_conf_path)"
    local challenge_dir="${PROJECT_DIR}/.well-known/acme-challenge"
    mkdir -p "$challenge_dir"

    if [[ -f "$conf" ]]; then
        if ! grep -q "\\.well-known/acme-challenge" "$conf"; then
            warn "当前 Nginx 配置未放行 ACME challenge，尝试幂等补块..."
            ensure_site_conf_blocks
        fi
    fi

    # 先做一次配置检测，避免带着坏配置去续期
    if ! $NGINX_TEST_CMD >/dev/null 2>&1; then
        error "Nginx 配置检测失败（$NGINX_TEST_CMD），已中止续期："
        $NGINX_TEST_CMD 2>&1 | tail -5
        return 1
    fi

    ensure_certbot_autorenew
    local reload_cmd
    reload_cmd="$(detect_nginx_reload_cmd)"
    if certbot renew --quiet --deploy-hook "$reload_cmd"; then
        log "证书续期检查完成（未到期则不会更新）；若已续期，已按 $reload_cmd 重载 Nginx"
    else
        warn "certbot renew 执行异常，请检查: certbot certificates"
    fi
    $NGINX_RELOAD_CMD >/dev/null 2>&1 || true
    return 0
}

unlock_user() {
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; exit 1; }
    log "解锁用户账户（支持管理员和普通用户）..."
    
    # 检查 Go 环境
    if ! command -v go &> /dev/null; then
        error "未找到 Go 命令，请先安装 Go"
        return 1
    fi
    
    # 检查脚本文件是否存在
    if [[ ! -f "./scripts/unlock_user/main.go" ]]; then
        error "脚本文件不存在: ./scripts/unlock_user/main.go"
        return 1
    fi
    
    read -r -p "请输入要解锁的用户名或邮箱: " identifier
    if [[ -z "$identifier" ]]; then
        error "用户名或邮箱不能为空"
        return 1
    fi
    
    log "正在解锁账户: $identifier（数据库: $(detect_db_path)）"
    if run_cli_tool unlock_user "$identifier" 2>&1; then
        log "✅ 账户 $identifier 已解锁"
    else
        error "解锁失败，请检查："
        error "  1. 用户名或邮箱是否正确"
        error "  2. 数据库连接是否正常"
        error "  3. 查看上方错误信息"
        return 1
    fi
}

sync_from_github() {
    CURRENT_STAGE="升级同步（sync_from_github）"
    log "开始从 GitHub 同步代码..."

    # 环境预检：Go 版本 / 磁盘空间，不满足直接中止（现有服务不受影响）
    if ! preflight_environment; then
        return 1
    fi

    if [ ! -d "$PROJECT_DIR" ]; then
        error "项目目录不存在: $PROJECT_DIR"
        return 1
    fi
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; return 1; }

    # ===== 数据库安全守卫（前置）：拒绝含数据库文件的仓库 + 记录生产库指纹 =====
    if ! db_guard_pre; then
        error "数据库安全守卫未通过，同步已中止（现有服务与数据库不受影响）"
        return 1
    fi

    # 检查是否是 git 仓库，不是则自动初始化
    if [ ! -d ".git" ]; then
        log "项目目录不是 Git 仓库，正在初始化..."
        git init
        git remote add origin "$GITHUB_REPO"
        git fetch origin || { error "拉取代码失败，请检查网络"; return 1; }
        git checkout -b main
        git reset --hard origin/main
        # 首次初始化后同样校验生产库未被触碰
        if ! db_guard_post; then
            return 1
        fi
        log "✅ Git 仓库初始化完成"
    fi

    # 拉取最新代码（强制同步，完全镜像 GitHub）
    log "正在拉取最新代码..."
    local branch
    branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null)"
    if [[ -z "$branch" || "$branch" == "HEAD" ]]; then
        error "当前仓库处于 detached HEAD 状态，无法自动更新。请先 git checkout main"
        return 1
    fi
    log "当前分支: $branch"

    # fetch 失败必须中止：历史实现忽略退出码，失败时 diff 得 0 行 →
    # 打印「代码已是最新」并继续用旧代码构建（静默假成功）
    if ! git fetch origin 2>&1 | tail -3; then
        error "git fetch 失败（请检查网络/凭据），已中止升级"
        return 1
    fi
    if ! git rev-parse --verify --quiet "origin/$branch" >/dev/null; then
        error "远端分支 origin/$branch 不存在，已中止升级"
        return 1
    fi
    local changed_files
    changed_files=$(git --no-pager diff --name-only HEAD "origin/$branch" 2>/dev/null | wc -l)
    if [ "$changed_files" -gt 0 ]; then
        log "检测到 $changed_files 个文件有更新："
        git --no-pager diff --name-status HEAD "origin/$branch"

        # 强制同步：完全匹配 GitHub 状态，删除多余文件
        # （*.db / .env / uploads 已在 .gitignore；这里再用 -e 显式排除，双保险，
        #   即使 .gitignore 缺失也不会删到数据库/配置文件）
        git reset --hard "origin/$branch"
        # 先列出将被删除的未跟踪文件（避免静默删掉运维自建脚本/私钥等），再做清理
        local to_delete
        to_delete="$(git clean -nd -e '*.db' -e '*.db-*' -e '*.sqlite' -e '*.sqlite3' -e '.env' -e '.env.*' -e 'uploads/' -e 'server' -e 'server.bak.*' -e 'server.log' -e 'frontend/dist/' 2>/dev/null | head -20)"
        if [[ -n "$to_delete" ]]; then
            log "以下未跟踪文件将被清理（数据库/配置/产物均已排除）："
            echo "$to_delete" | sed 's/^/    /'
        fi
        git clean -fd -e '*.db' -e '*.db-*' -e '*.sqlite' -e '*.sqlite3' -e '.env' -e '.env.*' -e 'uploads/' -e 'server' -e 'server.bak.*' -e 'server.log' -e 'frontend/dist/'
        # ===== 数据库安全守卫（后置）：校验生产库原封未动，异常立即中止 =====
        if ! db_guard_post; then
            error "数据库完整性校验未通过，已中止升级（服务未重启，原进程仍在运行）"
            return 1
        fi
        log "✅ 代码同步成功（已完全匹配 GitHub）"
    else
        log "代码已是最新，跳过拉取，继续构建和重启..."
    fi

    # 升级前备份数据库（启动会 AutoMigrate 改 schema，失败要能回滚）
    backup_database

    # 编译后端（原子替换 + 备份旧版本）
    if ! command -v go &> /dev/null; then
        error "未找到 Go 命令，请先安装 Go"
        return 1
    fi
    log "Go 版本: $(go version)"
    build_backend || return 1

    # 构建前端（依赖变更时自动重装）
    build_frontend || return 1

    # 保障 .env 与 unit（升级路径也要保证 EnvironmentFile/绝对库路径存在）
    ensure_env_file
    if [[ -f /etc/systemd/system/cboard.service ]] && ! grep -q "EnvironmentFile" /etc/systemd/system/cboard.service; then
        log "检测到 systemd 单元缺少 EnvironmentFile，正在重写为标准模板..."
        write_systemd_unit
    fi

    # 检查并更新 Redis 配置（同步模式不阻塞等待输入）
    log "检查 Redis 配置状态（非交互模式）..."
    check_and_update_redis_config "true"

    # 清除本项目 Redis 缓存（代码更新后必须清除；只删本项目键前缀）
    log "正在清除本项目 Redis 缓存并重启服务..."
    redis_clean_app_cache
    restart_redis_with_timeout

    # 刷新 nginx 配置：本脚本生成的配置按模板重渲染，手工配置只做幂等补块
    refresh_site_config
    ensure_site_conf_blocks

    # 重启服务（带业务健康检查，不再只看 is-active）
    log "正在重启服务..."
    if systemctl list-unit-files | grep -q "cboard.service"; then
        ensure_env_file
        stop_app_processes
        ensure_repo_sync_nginx
        if start_and_verify_service; then
            log "✅ 服务已成功重启，同步完成！"
        else
            error "服务重启后健康检查失败。可执行菜单 13 回滚到升级前版本，或查看 ${PROJECT_DIR}/server.log"
        fi
    else
        warn "服务 cboard 不存在，跳过重启。请先执行全自动部署。"
    fi
}

show_logs() {
    # 服务单元把 stdout 追加到 server.log（不是 journald），
    # 所以必须看文件；过去这里跑 journalctl，用户只能看到 systemd 的启停两行。
    local logf="${PROJECT_DIR}/server.log"
    if [[ ! -f "$logf" ]]; then
        if systemctl list-unit-files 2>/dev/null | grep -q "cboard.service"; then
            warn "未找到 ${logf}，回退到 journalctl"
            journalctl -u cboard -n 50 -f
            return 0
        fi
        error "服务 cboard 不存在且无日志文件，请先部署"
        return 1
    fi
    log "展示最近 50 行日志（文件: $logf，Ctrl+C 退出）:"
    tail -n 50 -f "$logf"
}

# --- 3. 交互式菜单 ---

show_menu() {
    clear
    echo -e "${BLUE}=========================================="
    echo -e "       CBoard Go 终极管理面板"
    echo -e "==========================================${NC}"
    echo -e "  ${GREEN}1.${NC} 一键全自动部署 (SSL + 反代)"
    echo -e "  ${GREEN}2.${NC} 创建/重置管理员账号"
    echo -e "  ${GREEN}3.${NC} 强制重启服务 (杀进程后重启)"
    echo -e "  ${GREEN}4.${NC} 深度清理系统缓存"
    echo -e "  ${GREEN}5.${NC} 解锁用户账户（支持管理员和普通用户）"
    echo -e "------------------------------------------"
    echo -e "  ${CYAN}6.${NC} 查看服务运行状态"
    echo -e "  ${CYAN}7.${NC} 查看实时服务日志"
    echo -e "  ${CYAN}8.${NC} 标准重启服务 (Systemd)"
    echo -e "  ${CYAN}9.${NC} 停止服务"
    echo -e "  ${CYAN}10.${NC} 证书续期（手动续期，自动续期由 certbot 定时任务完成）"
    echo -e "  ${CYAN}11.${NC} 从 GitHub 同步代码并重新构建"
    echo -e "  ${YELLOW}12.${NC} 配置 Redis 缓存（性能优化）"
    echo -e "  ${YELLOW}13.${NC} 回滚到升级前版本（二进制/数据库备份）"
    echo -e "  ${YELLOW}14.${NC} 只重新构建并重启（不改 nginx / unit）"
    echo -e "  ${RED}0.${NC} 退出脚本"
    echo -e "${BLUE}==========================================${NC}"
    # 每次先清空 choice：bash 的 read 在 EOF 时不修改变量，
    # 若不重置，管道/关闭 stdin 的场景会一直重复执行上一次的选择
    choice=""
    if ! read -r -p "请选择操作 [0-14]: " choice; then
        log "检测到标准输入已关闭（非交互执行），退出。"
        exit 0
    fi
    if [[ -z "$choice" ]]; then
        log "未收到输入，退出。"
        exit 0
    fi
}

# --- 主程序循环 ---
main() {
    while true; do
        show_menu
        # 标准输入关闭（如管道/SSH 非交互执行）时退出，避免死循环占满 CPU
        if [[ -z "$choice" ]]; then
            log "检测到非交互输入，自动退出。"
            exit 0
        fi
        case $choice in
            1) full_deploy ;;
            2) manage_admin ;;
            3)
                force_kill
                sleep 1
                # 重启 Redis 服务（使用带超时的函数避免卡住）
                restart_redis_with_timeout
                start_and_verify_service || error "服务启动失败，请查看 ${PROJECT_DIR}/server.log"
                ;;
            4) deep_clean ;;
            5) unlock_user ;;
            6) 
                if systemctl list-unit-files | grep -q "cboard.service"; then
                    systemctl status cboard --no-pager
                else
                    error "服务 cboard 不存在，请先部署"
                fi
                ;;
            7) show_logs ;;
            8)
                if systemctl list-unit-files | grep -q "cboard.service"; then
                    log "正在重启服务..."
                    stop_app_processes
                    restart_redis_with_timeout
                    start_and_verify_service || error "服务重启后健康检查失败，请查看 ${PROJECT_DIR}/server.log"
                else
                    error "服务 cboard 不存在，请先部署"
                fi
                ;;
            9) 
                if systemctl list-unit-files | grep -q "cboard.service"; then
                    if systemctl stop cboard; then
                        log "✅ 服务已停止"
                    else
                        error "服务停止失败"
                    fi
                else
                    error "服务 cboard 不存在"
                fi
                ;;
            10) renew_cert ;;
            11) sync_from_github ;;
            13) rollback_to_previous ;;
            14) rebuild_and_restart ;;
            12)
                configure_redis_cache
                # 重启服务以应用新配置（EOF 时视为不重启）
                if systemctl list-unit-files | grep -q "cboard.service"; then
                    local restart_now=""
                    if read -r -p "是否立即重启服务以应用配置？(y/n，默认: y): " restart_now; then
                        restart_now=${restart_now:-y}
                    else
                        restart_now="n"
                    fi
                    if [[ "$restart_now" == "y" || "$restart_now" == "Y" ]]; then
                        start_and_verify_service || error "服务重启失败"
                    fi
                fi
                ;;
            0) exit 0 ;;
            *) error "无效选择，请重新输入" ;;
        esac
        # 返回菜单的等待同样要判 EOF，否则 stdin 关闭时会空转占满 CPU
        local _wait=""
        read -r -p "按回车键返回菜单..." _wait || { log "标准输入已关闭，退出。"; exit 0; }
    done
}

# 只有「直接执行」时才做 root 检查、抢锁并进菜单；
# 被 source 时只加载函数，便于自动化测试（CI/沙箱）复用这些函数。
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    [[ "$EUID" -ne 0 ]] && { echo "请使用 root 运行"; exit 1; }

    # 单实例锁：避免两个实例同时做 git reset --hard / 构建 / 覆盖 nginx 配置
    exec 9>/var/lock/cboard-install.lock 2>/dev/null
    if command -v flock >/dev/null 2>&1; then
        if ! flock -n 9; then
            echo "另一个 install.sh 正在运行，已退出（如确认无其它实例，可删除 /var/lock/cboard-install.lock）"
            exit 1
        fi
    fi

    # 中断提示：set +e 下 Ctrl-C 可能停在「代码已更新但二进制未更新」之类的中间态
    trap 'echo; echo "已中断。当前阶段: ${CURRENT_STAGE:-未知}。请重新运行菜单 11 或 14 完成构建/重启。"' INT TERM

    main
fi