#!/bin/bash
# ============================================
# CBoard Go 终极管理脚本 (部署 + 运维 + 修复)
# ============================================

set +e

# --- 基础配置 (自动检测) ---
# 脚本所在目录即为项目目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="${SCRIPT_DIR}"
# 自动从目录名提取域名（/www/wwwroot/xxx.com → xxx.com）
DOMAIN="$(basename "$PROJECT_DIR")"
GITHUB_REPO="https://github.com/moneyfly1/myweb.git"
LOG_FILE="/tmp/cboard_admin.log"

# --- 颜色定义 ---
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; CYAN='\033[0;36m'; NC='\033[0m'

# --- 辅助函数 ---
log() { echo -e "${GREEN}[$(date +'%H:%M:%S')] $1${NC}"; }
warn() { echo -e "${YELLOW}[WARN] $1${NC}"; }
error() { echo -e "${RED}[ERROR] $1${NC}"; }

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
            systemctl start redis 2>/dev/null || service redis start 2>/dev/null
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
                    log "使用 Docker 安装 Redis..."
                    docker run -d --name redis --restart=always -p 6379:6379 redis:alpine
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

    # 更新 systemd 服务文件以加载环境变量
    local service_file="/etc/systemd/system/cboard.service"
    if [[ -f "$service_file" ]]; then
        if ! grep -q "EnvironmentFile" "$service_file"; then
            sed -i "/^Environment=/a EnvironmentFile=-${PROJECT_DIR}/.env" "$service_file"
            systemctl daemon-reload
            log "✅ systemd 服务已更新"
        fi
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

        if command -v redis-cli &> /dev/null; then
            if redis-cli -h "$redis_host" -p "$redis_port" ping &> /dev/null; then
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

# --- 1. 核心部署逻辑 (融合之前的完美版) ---

reload_nginx_force() {
    log "正在配置 Nginx..."
    if [[ -f "/run/nginx.pid" ]] && [[ ! -s "/run/nginx.pid" ]]; then
        rm -f /run/nginx.pid && pkill -9 nginx 2>/dev/null
    fi
    systemctl restart nginx || /etc/init.d/nginx restart || nginx
}

full_deploy() {
    log "开始全自动部署流程..."
    
    # 1. 检查并进入项目目录
    if [ ! -d "$PROJECT_DIR" ]; then
        error "项目目录不存在: $PROJECT_DIR"
        exit 1
    fi
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; exit 1; }
    
    # 2. 检查 Go 环境
    if ! command -v go &> /dev/null; then
        error "未找到 Go 命令，请先安装 Go"
        exit 1
    fi
    log "Go 版本: $(go version)"
    
    # 3. 下载和整理 Go 依赖
    log "正在下载 Go 依赖..."
    if ! go mod download; then
        warn "依赖下载失败，尝试继续..."
    fi
    if ! go mod tidy; then
        warn "依赖整理失败，尝试继续..."
    fi
    
    # 4. 编译 Go 程序
    log "正在编译 Go 程序..."
    if go build -o server ./cmd/server/main.go; then
        log "✅ Go 程序编译成功"
    else
        error "Go 程序编译失败"
        exit 1
    fi
    
    # 5. 构建前端
    log "正在构建前端..."
    cd frontend || { error "前端目录不存在"; exit 1; }
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
    
    # 6. 创建 systemd 服务文件
    log "正在创建 systemd 服务..."
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
    
    # 7. 生成HTTP配置
    local bt_path="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
    mkdir -p "$(dirname "$bt_path")"
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

    # 8. 申请SSL
    log "正在申请 SSL 证书..."
    certbot certonly --webroot -w "${PROJECT_DIR}" -d "${DOMAIN}" --email "admin@${DOMAIN}" --agree-tos --non-interactive --quiet 2>/dev/null || {
        warn "SSL 证书申请失败，继续使用 HTTP 配置"
    }
    
    # 8. 生成最终配置
    local cert_root=$(find /etc/letsencrypt/live -name "*${DOMAIN}*" -type d | head -n 1)
    if [ -n "$cert_root" ] && [ -f "$cert_root/fullchain.pem" ]; then
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
        setup_cert_auto_renew_hook
    else
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

    # 7. 配置 Redis 缓存（可选）
    configure_redis_cache

    # 8. 启动服务
    log "正在启动服务..."
    systemctl enable cboard
    systemctl restart cboard
    
    # 10. 检查服务状态
    sleep 3
    if systemctl is-active --quiet cboard; then
        log "✅ 服务已成功启动"
        systemctl status cboard --no-pager -l
    else
        error "服务启动失败，请查看日志: journalctl -u cboard -n 50"
        exit 1
    fi
    
    log "部署完成！"
    log "服务状态: systemctl status cboard"
    log "查看日志: journalctl -u cboard -f"
}

# --- 2. 运维管理功能 (参考原脚本融合) ---

manage_admin() {
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; exit 1; }
    log "创建/重置管理员账户..."
    
    # 检查 Go 环境
    if ! command -v go &> /dev/null; then
        error "未找到 Go 命令，请先安装 Go"
        return 1
    fi
    
    # 检查脚本文件是否存在
    if [[ ! -f "scripts/admin_tool.go" ]]; then
        error "脚本文件不存在: scripts/admin_tool.go"
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
    
    # 输入密码
    read -r -p "请输入管理员密码 (留空使用默认密码 admin123): " admin_pass
    if [[ -z "$admin_pass" ]]; then
        warn "使用默认密码 admin123（仅开发环境）"
        admin_pass="admin123"
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
    
    log "正在执行创建管理员账户脚本..."
    if go run scripts/admin_tool.go 2>&1; then
        log "✅ 管理员账户已创建/重置"
        log "用户名: $admin_username"
        log "邮箱: $admin_email"
    else
        error "管理员账户创建/重置失败，请检查错误信息"
        return 1
    fi
}

force_kill() {
    log "强制停止所有相关进程..."
    pkill -9 server 2>/dev/null
    pkill -9 node 2>/dev/null
    systemctl stop cboard 2>/dev/null
    
    # 等待进程完全停止
    sleep 2
    
    # 检查是否还有残留进程
    if pgrep -f "server|node" > /dev/null 2>&1; then
        warn "仍有残留进程，再次清理..."
        pkill -9 -f "server|node" 2>/dev/null
        sleep 1
    fi
    
    log "✅ 进程已全部清理"
}

deep_clean() {
    log "正在清理深度缓存..."

    # 清理 Redis 缓存
    if command -v redis-cli &> /dev/null; then
        local redis_addr=$(grep "^REDIS_ADDR=" "${PROJECT_DIR}/.env" 2>/dev/null | cut -d'=' -f2)
        if [[ -n "$redis_addr" ]]; then
            local redis_host=$(echo "$redis_addr" | cut -d':' -f1)
            local redis_port=$(echo "$redis_addr" | cut -d':' -f2)
            if redis-cli -h "$redis_host" -p "$redis_port" FLUSHDB &> /dev/null; then
                log "✅ Redis 缓存已清空"
            else
                warn "Redis 缓存清除失败（可能未连接）"
            fi
        fi
    fi

    # 清理前端构建文件
    if [[ -d "$PROJECT_DIR/frontend/dist" ]]; then
        rm -rf "$PROJECT_DIR/frontend/dist"
        log "已清理前端构建文件"
    else
        log "前端构建目录不存在，跳过"
    fi

    # 清理日志文件
    if [[ -d "$PROJECT_DIR/logs" ]]; then
        rm -rf "$PROJECT_DIR/logs"/* 2>/dev/null
        log "已清理日志文件"
    else
        log "日志目录不存在，跳过"
    fi

    # 清理临时文件
    local tmp_count=$(find "$PROJECT_DIR" -name "*.tmp" 2>/dev/null | wc -l)
    if [[ $tmp_count -gt 0 ]]; then
        find "$PROJECT_DIR" -name "*.tmp" -delete 2>/dev/null
        log "已清理 $tmp_count 个临时文件"
    else
        log "未找到临时文件"
    fi

    # 清理 Go 构建缓存
    if [[ -f "$PROJECT_DIR/server" ]]; then
        rm -f "$PROJECT_DIR/server"
        log "已清理 Go 可执行文件"
    fi

    log "✅ 缓存清理完毕"
}

# 配置证书续期后自动重载 Nginx（供 certbot 自动续期时调用）
setup_cert_auto_renew_hook() {
    local hook_dir="/etc/letsencrypt/renewal-hooks/deploy"
    local hook_file="$hook_dir/reload-nginx.sh"
    mkdir -p "$hook_dir"
    if [[ ! -x "$hook_file" ]]; then
        cat > "$hook_file" <<'HOOK'
#!/bin/bash
# certbot 续期成功后自动执行，重载 Nginx 以加载新证书
systemctl reload nginx 2>/dev/null || /etc/init.d/nginx reload 2>/dev/null || true
HOOK
        chmod +x "$hook_file"
        log "已配置证书自动续期钩子: 续期后将自动重载 Nginx"
    fi
}

renew_cert() {
    log "证书续期（Let's Encrypt）..."
    if ! command -v certbot &>/dev/null; then
        error "未安装 certbot，请先执行「一键全自动部署」或安装 certbot"
        return 1
    fi
    setup_cert_auto_renew_hook
    if certbot renew --quiet --deploy-hook "systemctl reload nginx 2>/dev/null || /etc/init.d/nginx reload 2>/dev/null"; then
        log "证书续期检查完成（未到期则不会更新）；若已续期，Nginx 已重载"
    else
        warn "certbot renew 执行异常，请检查: certbot certificates"
        certbot renew --no-quiet 2>&1 | tail -20
    fi
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
    if [[ ! -f "scripts/unlock_user.go" ]]; then
        error "脚本文件不存在: scripts/unlock_user.go"
        return 1
    fi
    
    read -r -p "请输入要解锁的用户名或邮箱: " identifier
    if [[ -z "$identifier" ]]; then
        error "用户名或邮箱不能为空"
        return 1
    fi
    
    log "正在解锁账户: $identifier"
    if go run scripts/unlock_user.go "$identifier" 2>&1; then
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
    log "开始从 GitHub 同步代码..."

    if [ ! -d "$PROJECT_DIR" ]; then
        error "项目目录不存在: $PROJECT_DIR"
        return 1
    fi
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; return 1; }

    # 检查是否是 git 仓库，不是则自动初始化
    if [ ! -d ".git" ]; then
        log "项目目录不是 Git 仓库，正在初始化..."
        git init
        git remote add origin "$GITHUB_REPO"
        git fetch origin || { error "拉取代码失败，请检查网络"; return 1; }
        git checkout -b main
        git reset --hard origin/main
        log "✅ Git 仓库初始化完成"
    fi

    # 拉取最新代码（增量更新）
    log "正在拉取最新代码..."
    local branch=$(git rev-parse --abbrev-ref HEAD)
    log "当前分支: $branch"

    # 显示即将更新的文件
    git fetch origin
    local changed_files=$(git diff --name-only HEAD "origin/$branch" | wc -l)
    if [ "$changed_files" -gt 0 ]; then
        log "检测到 $changed_files 个文件有更新："
        git diff --name-status HEAD "origin/$branch"

        # 增量拉取（保留本地未提交的修改）
        if ! git pull origin "$branch"; then
            warn "自动合并失败，尝试保存本地修改后重新拉取..."

            # 仅在存在已跟踪文件改动时保存（untracked 文件不影响 pull）
            if ! git diff --quiet || ! git diff --cached --quiet; then
                if git stash push -m "自动保存 - $(date +'%Y-%m-%d %H:%M:%S')"; then
                    log "本地修改已保存到 stash"
                else
                    error "保存本地修改失败"
                    return 1
                fi
            else
                log "未检测到需要 stash 的已跟踪修改，继续拉取..."
            fi

            # 重新拉取
            if git pull origin "$branch"; then
                log "代码拉取成功，尝试恢复本地修改..."

                # 仅在有 stash 项时恢复
                if git stash list | grep -q "自动保存 - "; then
                    if git stash pop; then
                        log "✅ 本地修改已恢复"
                    else
                        warn "本地修改恢复失败（可能有冲突），已保存在 stash 中"
                        warn "请手动执行: git stash list 查看，git stash pop 恢复"
                    fi
                else
                    log "没有需要恢复的 stash 修改"
                fi
            else
                error "代码同步失败"
                return 1
            fi
        fi
        log "✅ 代码同步成功"
    else
        log "代码已是最新，跳过拉取，继续构建和重启..."
    fi

    # 编译后端
    log "正在编译 Go 程序..."
    if ! command -v go &> /dev/null; then
        error "未找到 Go 命令，请先安装 Go"
        return 1
    fi
    log "Go 版本: $(go version)"
    if ! go mod download; then
        warn "go mod download 失败，继续尝试构建..."
    fi
    if ! go mod tidy; then
        warn "go mod tidy 失败，继续尝试构建..."
    fi
    if go build -o server ./cmd/server/main.go; then
        log "✅ Go 程序编译成功"
    else
        error "Go 程序编译失败"
        return 1
    fi

    # 构建前端
    log "正在构建前端..."
    cd frontend || { error "前端目录不存在"; return 1; }
    if [ ! -d "node_modules" ]; then
        log "安装前端依赖..."
        npm install --legacy-peer-deps || { error "前端依赖安装失败"; return 1; }
    fi
    if npm run build; then
        log "✅ 前端构建成功"
    else
        error "前端构建失败"
        return 1
    fi
    cd ..

    # 检查并更新 Redis 配置（同步模式不阻塞等待输入）
    log "检查 Redis 配置状态（非交互模式）..."
    check_and_update_redis_config "true"

    # 清除 Redis 缓存（代码更新后必须清除）
    log "正在清除 Redis 缓存..."
    if command -v redis-cli &> /dev/null; then
        local redis_addr=$(grep "^REDIS_ADDR=" "${PROJECT_DIR}/.env" 2>/dev/null | cut -d'=' -f2)
        if [[ -n "$redis_addr" ]]; then
            local redis_host=$(echo "$redis_addr" | cut -d':' -f1)
            local redis_port=$(echo "$redis_addr" | cut -d':' -f2)
            if redis-cli -h "$redis_host" -p "$redis_port" FLUSHDB &> /dev/null; then
                log "✅ Redis 缓存已清空（避免旧数据）"
            fi
        fi
    fi

    # 重启服务
    log "正在重启服务..."
    if systemctl list-unit-files | grep -q "cboard.service"; then
        systemctl stop cboard 2>/dev/null
        pkill -9 -f "${PROJECT_DIR}/server" 2>/dev/null
        sleep 1
        if systemctl start cboard; then
            sleep 2
            if systemctl is-active --quiet cboard; then
                log "✅ 服务已成功重启，同步完成！"
            else
                error "服务重启后未运行，请查看日志: journalctl -u cboard -n 50"
            fi
        else
            error "服务启动失败"
        fi
    else
        warn "服务 cboard 不存在，跳过重启。请先执行全自动部署。"
    fi
}

show_logs() {
    # 检查服务是否存在
    if ! systemctl list-unit-files | grep -q "cboard.service"; then
        error "服务 cboard 不存在，请先部署"
        return 1
    fi
    
    log "展示最近 50 行日志 (Ctrl+C 退出):"
    journalctl -u cboard -n 50 -f
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
    echo -e "  ${RED}0.${NC} 退出脚本"
    echo -e "${BLUE}==========================================${NC}"
    read -r -p "请选择操作 [0-12]: " choice
}

# --- 主程序循环 ---
main() {
    while true; do
        show_menu
        case $choice in
            1) full_deploy ;;
            2) manage_admin ;;
            3) 
                force_kill
                sleep 1
                if systemctl start cboard; then
                    sleep 2
                    if systemctl is-active --quiet cboard; then
                        log "✅ 服务已成功重启"
                    else
                        error "服务启动失败，请查看日志: journalctl -u cboard -n 50"
                    fi
                else
                    error "服务启动失败"
                fi
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
                    log "正在强制停止服务..."
                    systemctl stop cboard 2>/dev/null
                    pkill -9 -f "${PROJECT_DIR}/server" 2>/dev/null
                    sleep 1
                    if pgrep -f "${PROJECT_DIR}/server" > /dev/null 2>&1; then
                        error "仍有残留进程，尝试再次强制终止..."
                        pkill -9 -f "${PROJECT_DIR}/server" 2>/dev/null
                        sleep 1
                    fi
                    log "正在启动服务..."
                    if systemctl start cboard; then
                        sleep 2
                        if systemctl is-active --quiet cboard; then
                            log "✅ 服务已成功重启"
                        else
                            error "服务重启后未运行，请查看日志: journalctl -u cboard -n 50"
                        fi
                    else
                        error "服务启动失败"
                    fi
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
            12)
                configure_redis_cache
                # 重启服务以应用新配置
                if systemctl list-unit-files | grep -q "cboard.service"; then
                    read -r -p "是否立即重启服务以应用配置？(y/n，默认: y): " restart_now
                    restart_now=${restart_now:-y}
                    if [[ "$restart_now" == "y" || "$restart_now" == "Y" ]]; then
                        systemctl restart cboard
                        sleep 2
                        if systemctl is-active --quiet cboard; then
                            log "✅ 服务已重启，Redis 缓存已生效"
                        else
                            error "服务重启失败"
                        fi
                    fi
                fi
                ;;
            0) exit 0 ;;
            *) error "无效选择，请重新输入" ;;
        esac
        read -r -p "按回车键返回菜单..." temp
    done
}

# 运行检查
[[ "$EUID" -ne 0 ]] && { echo "请使用 root 运行"; exit 1; }
main
