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
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'

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
    # 生成HTTP配置
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

    # 申请SSL
    certbot certonly --webroot -w "${PROJECT_DIR}" -d "${DOMAIN}" --email "admin@${DOMAIN}" --agree-tos --non-interactive --quiet
    
    # 生成最终配置
    local cert_root=$(find /etc/letsencrypt/live -name "*${DOMAIN}*" -type d | head -n 1)
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
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }
    location / { try_files \$uri \$uri/ /index.html; }
}
EOF
    reload_nginx_force
    log "部署完成！"
}

# --- 2. 运维管理功能 (参考原脚本融合) ---

manage_admin() {
    cd "$PROJECT_DIR"
    read -r -p "请输入新的管理员邮箱: " admin_email
    read -r -p "请输入新的管理员密码: " admin_pass
    export ADMIN_EMAIL="$admin_email"
    export ADMIN_PASSWORD="$admin_pass"
    go run scripts/create_admin.go
    log "管理员账户已创建/重置。"
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
    read -r -p "请输入要解锁的管理员用户名或邮箱: " identifier
    cd "$PROJECT_DIR"
    if [ -f "scripts/unlock_admin.go" ]; then
        go run scripts/unlock_admin.go "$identifier"
        log "管理员账户解锁操作已完成。"
    else
        error "未找到解锁脚本: scripts/unlock_admin.go"
        log "尝试使用 SQLite 直接解锁..."
        if [ -f "cboard.db" ]; then
            sqlite3 cboard.db "UPDATE users SET is_active=1, is_verified=1 WHERE email='$identifier' OR username='$identifier';"
            log "账户 $identifier 已尝试解锁。"
        else
            error "未找到数据库文件: cboard.db"
        fi
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
    echo -e "  ${GREEN}5.${NC} 解锁管理员账户"
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