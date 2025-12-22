#!/bin/bash
# ============================================
# CBoard Go 终极管理脚本 (部署 + 运维 + 修复)
# ============================================

set +e

# --- 基础配置 ---
PROJECT_DIR="/www/wwwroot/dy.moneyfly.top"
DOMAIN="dy.moneyfly.top"
LOG_FILE="/tmp/cboard_admin.log"

# --- 颜色定义 ---
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; CYAN='\033[0;36m'; NC='\033[0m'

# --- 辅助函数 ---
log() { echo -e "${GREEN}[$(date +'%H:%M:%S')] $1${NC}"; }
warn() { echo -e "${YELLOW}[WARN] $1${NC}"; }
error() { echo -e "${RED}[ERROR] $1${NC}"; }

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
    
    # 9. 启动服务
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
    
    # 输入用户名
    read -r -p "请输入管理员用户名 (留空使用默认: admin): " admin_username
    if [[ -z "$admin_username" ]]; then
        admin_username="admin"
        log "使用默认用户名: admin"
    fi
    
    # 输入邮箱
    read -r -p "请输入管理员邮箱 (留空使用默认: admin@example.com): " admin_email
    if [[ -z "$admin_email" ]]; then
        admin_email="admin@example.com"
        log "使用默认邮箱: admin@example.com"
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
    
    if go run scripts/create_admin.go; then
        log "✅ 管理员账户已创建/重置"
        log "用户名: $admin_username"
        log "邮箱: $admin_email"
    else
        error "管理员账户创建/重置失败"
        return 1
    fi
}

force_kill() {
    log "强制停止所有相关进程..."
    pkill -9 server
    pkill -9 node
    systemctl stop cboard 2>/dev/null
    log "进程已全部清理。"
}

deep_clean() {
    log "正在清理深度缓存..."
    rm -rf "$PROJECT_DIR/frontend/dist"
    rm -rf "$PROJECT_DIR/logs/*"
    find "$PROJECT_DIR" -name "*.tmp" -delete
    log "缓存清理完毕。"
}

unlock_user() {
    cd "$PROJECT_DIR" || { error "无法进入项目目录"; exit 1; }
    log "解锁用户账户（支持管理员和普通用户）..."
    read -r -p "请输入要解锁的用户名或邮箱: " identifier
    if [[ -z "$identifier" ]]; then
        error "用户名或邮箱不能为空"
        return 1
    fi
    if go run scripts/unlock_user.go "$identifier"; then
        log "✅ 账户 $identifier 已解锁"
    else
        error "解锁失败，请检查用户名或邮箱是否正确"
    fi
}

show_logs() {
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
    echo -e "  ${RED}0.${NC} 退出脚本"
    echo -e "${BLUE}==========================================${NC}"
    read -r -p "请选择操作 [0-9]: " choice
}

# --- 主程序循环 ---
main() {
    while true; do
        show_menu
        case $choice in
            1) full_deploy ;;
            2) manage_admin ;;
            3) force_kill; systemctl start cboard; log "服务已重启" ;;
            4) deep_clean ;;
            5) unlock_user ;;
            6) systemctl status cboard --no-pager ;;
            7) show_logs ;;
            8) systemctl restart cboard; log "服务已重启" ;;
            9) systemctl stop cboard; log "服务已停止" ;;
            0) exit 0 ;;
            *) error "无效选择，请重新输入" ;;
        esac
        read -r -p "按回车键返回菜单..." temp
    done
}

# 运行检查
[[ "$EUID" -ne 0 ]] && { echo "请使用 root 运行"; exit 1; }
main