#!/bin/bash

# 修复 Nginx 404 问题脚本

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}修复 Nginx 404 问题${NC}"
echo -e "${BLUE}========================================${NC}"

INSTALL_DIR=$(pwd)
DOMAIN="board.moneyfly.club"

# 检测 PHP socket
PHP_VER=$(php -v | head -n 1 | cut -d " " -f 2 | cut -d "." -f 1,2)
PHP_VERSION_DIR=$(php --ini | grep "Loaded Configuration File" | grep -oP '/www/server/php/\K[0-9]+' || echo "82")
PHP_SOCKET="/tmp/php-cgi-${PHP_VERSION_DIR}.sock"

echo -e "${YELLOW}安装目录: $INSTALL_DIR${NC}"
echo -e "${YELLOW}域名: $DOMAIN${NC}"
echo -e "${YELLOW}PHP Socket: $PHP_SOCKET${NC}"
echo ""

# 检查宝塔面板配置文件
BT_CONFIG="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
SSPANEL_CONFIG="/www/server/panel/vhost/nginx/sspanel.conf"

echo -e "${YELLOW}步骤 1: 检查现有配置...${NC}"

# 如果存在宝塔面板的配置文件，备份并修复
if [ -f "$BT_CONFIG" ]; then
    echo -e "${GREEN}找到宝塔面板配置文件: $BT_CONFIG${NC}"
    echo -e "${YELLOW}备份原配置...${NC}"
    cp "$BT_CONFIG" "${BT_CONFIG}.backup.$(date +%Y%m%d_%H%M%S)"
    
    echo -e "${YELLOW}修复宝塔面板配置...${NC}"
    cat > "$BT_CONFIG" <<EOF
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN};
    
    root ${INSTALL_DIR}/public;
    index index.php index.html;
    
    # 日志配置
    access_log /www/wwwlogs/${DOMAIN}.log;
    error_log /www/wwwlogs/${DOMAIN}.error.log;
    
    # 主 location 块
    location / {
        try_files \$uri \$uri/ /index.php?\$query_string;
    }
    
    # PHP 处理
    location ~ \.php\$ {
        try_files \$uri =404;
        fastcgi_pass unix:${PHP_SOCKET};
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME \$document_root\$fastcgi_script_name;
        include fastcgi_params;
        fastcgi_param HTTP_PROXY "";
        fastcgi_hide_header X-Powered-By;
        fastcgi_read_timeout 300;
    }
    
    # 禁止访问隐藏文件
    location ~ /\.(?!well-known).* {
        deny all;
    }
    
    # 禁止访问配置目录
    location ~ /config/ {
        deny all;
    }
    
    # 静态文件缓存
    location ~* \.(jpg|jpeg|png|gif|ico|css|js|svg|woff|woff2|ttf|eot)\$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
EOF
    echo -e "${GREEN}✓ 宝塔面板配置已修复${NC}"
fi

# 删除重复的 sspanel.conf（如果存在）
if [ -f "$SSPANEL_CONFIG" ] && [ "$SSPANEL_CONFIG" != "$BT_CONFIG" ]; then
    echo -e "${YELLOW}删除重复的配置文件: $SSPANEL_CONFIG${NC}"
    mv "$SSPANEL_CONFIG" "${SSPANEL_CONFIG}.backup.$(date +%Y%m%d_%H%M%S)"
    echo -e "${GREEN}✓ 已备份并删除重复配置${NC}"
fi

# 测试 Nginx 配置
echo -e "${YELLOW}步骤 2: 测试 Nginx 配置...${NC}"
if nginx -t 2>&1; then
    echo -e "${GREEN}✓ Nginx 配置测试通过${NC}"
else
    echo -e "${RED}✗ Nginx 配置测试失败${NC}"
    exit 1
fi

# 重载 Nginx
echo -e "${YELLOW}步骤 3: 重载 Nginx...${NC}"
if systemctl reload nginx >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Nginx 已重载${NC}"
elif /etc/init.d/nginx reload >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Nginx 已重载${NC}"
else
    echo -e "${YELLOW}尝试重启 Nginx...${NC}"
    if systemctl restart nginx >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Nginx 已重启${NC}"
    elif /etc/init.d/nginx restart >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Nginx 已重启${NC}"
    else
        echo -e "${RED}✗ Nginx 重载/重启失败，请手动执行: systemctl restart nginx${NC}"
    fi
fi

# 检查文件权限
echo -e "${YELLOW}步骤 4: 检查文件权限...${NC}"
if [ -d "$INSTALL_DIR/public" ]; then
    chown -R www-data:www-data "$INSTALL_DIR/public" 2>/dev/null || chown -R www:www "$INSTALL_DIR/public" 2>/dev/null
    chmod -R 755 "$INSTALL_DIR/public" 2>/dev/null
    echo -e "${GREEN}✓ 文件权限已设置${NC}"
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}修复完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}请测试访问: http://${DOMAIN}${NC}"
echo -e "${YELLOW}如果仍有问题，请检查:${NC}"
echo -e "${YELLOW}  1. 宝塔面板中站点配置是否正确${NC}"
echo -e "${YELLOW}  2. PHP 版本是否选择正确（PHP 8.2）${NC}"
echo -e "${YELLOW}  3. 站点根目录是否指向: ${INSTALL_DIR}/public${NC}"
