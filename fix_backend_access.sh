#!/bin/bash

# 修复后台访问问题脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}错误: 请使用 root 用户运行此脚本${NC}"
    exit 1
fi

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}修复后台访问问题${NC}"
echo -e "${BLUE}========================================${NC}"

# 1. 检查项目目录
echo -e "${YELLOW}步骤 1: 检查项目目录...${NC}"
if [ ! -d "/var/www/sspanel" ]; then
    echo -e "${RED}✗ 项目目录不存在: /var/www/sspanel${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 项目目录存在${NC}"

# 2. 检查 Nginx 配置
echo -e "${YELLOW}步骤 2: 检查 Nginx 配置...${NC}"

# 查找 Nginx 配置文件
NGINX_CONFIG=""
for config in "/etc/nginx/conf.d/sspanel.conf" "/www/server/panel/vhost/nginx/board.moneyfly.club.conf" "/www/server/nginx/conf/vhost/board.moneyfly.club.conf"; do
    if [ -f "$config" ]; then
        NGINX_CONFIG="$config"
        break
    fi
done

if [ -z "$NGINX_CONFIG" ]; then
    echo -e "${YELLOW}⚠ 未找到 Nginx 配置文件，正在创建...${NC}"
    
    # 获取域名
    read -p "请输入您的域名 (例如: board.moneyfly.club): " DOMAIN
    if [ -z "$DOMAIN" ]; then
        DOMAIN="board.moneyfly.club"
    fi
    
    # 检测 PHP socket
    PHP_INI=$(php --ini | grep "Loaded Configuration File" | awk '{print $4}')
    PHP_VER=$(php -v | head -n 1 | cut -d " " -f 2 | cut -d "." -f 1,2)
    PHP_MAJOR=$(echo $PHP_VER | cut -d "." -f 1)
    PHP_MINOR=$(echo $PHP_VER | cut -d "." -f 2)
    
    if [[ "$PHP_INI" == *"/www/server/php"* ]]; then
        PHP_VERSION_DIR=$(echo "$PHP_INI" | grep -oP '/www/server/php/\K[0-9]+')
        PHP_SOCKET="/tmp/php-cgi-${PHP_VERSION_DIR}.sock"
        CONFIG_DIR="/etc/nginx/conf.d"
    else
        PHP_SOCKET="/run/php/php${PHP_MAJOR}.${PHP_MINOR}-fpm.sock"
        CONFIG_DIR="/etc/nginx/conf.d"
    fi
    
    mkdir -p "$CONFIG_DIR"
    
    cat > ${CONFIG_DIR}/sspanel.conf <<EOF
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN};
    
    root /var/www/sspanel/public;
    index index.php;
    
    location / {
        try_files \$uri /index.php?\$query_string;
    }
    
    location ~ \.php\$ {
        fastcgi_pass unix:${PHP_SOCKET};
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME \$document_root\$fastcgi_script_name;
        include fastcgi_params;
        fastcgi_param HTTP_PROXY "";
        fastcgi_hide_header X-Powered-By;
    }
    
    location ~ /\.(?!well-known).* {
        deny all;
    }
    
    location ~ /config/ {
        deny all;
    }
}
EOF
    
    NGINX_CONFIG="${CONFIG_DIR}/sspanel.conf"
    echo -e "${GREEN}✓ 已创建 Nginx 配置文件: $NGINX_CONFIG${NC}"
else
    echo -e "${GREEN}✓ 找到 Nginx 配置文件: $NGINX_CONFIG${NC}"
fi

# 检查配置是否正确
if grep -q "/var/www/sspanel/public" "$NGINX_CONFIG" 2>/dev/null; then
    echo -e "${GREEN}✓ 配置指向正确的目录${NC}"
else
    echo -e "${YELLOW}⚠ 配置可能不正确，请检查 root 路径${NC}"
fi

# 3. 测试 Nginx 配置
echo -e "${YELLOW}步骤 3: 测试 Nginx 配置...${NC}"
if nginx -t >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Nginx 配置测试通过${NC}"
else
    echo -e "${RED}✗ Nginx 配置测试失败${NC}"
    nginx -t
    exit 1
fi

# 4. 检查文件权限
echo -e "${YELLOW}步骤 4: 检查文件权限...${NC}"
chown -R www-data:www-data /var/www/sspanel 2>/dev/null || chown -R nginx:nginx /var/www/sspanel 2>/dev/null || true
chmod -R 755 /var/www/sspanel
chmod -R 777 /var/www/sspanel/storage
echo -e "${GREEN}✓ 文件权限已设置${NC}"

# 5. 重启 Nginx
echo -e "${YELLOW}步骤 5: 重启 Nginx...${NC}"
if systemctl reload nginx >/dev/null 2>&1 || systemctl restart nginx >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Nginx 已重启${NC}"
else
    /etc/init.d/nginx restart >/dev/null 2>&1 || nginx -s reload >/dev/null 2>&1 || true
    echo -e "${GREEN}✓ Nginx 已重启${NC}"
fi

# 6. 检查数据库初始化
echo -e "${YELLOW}步骤 6: 检查数据库初始化...${NC}"
cd /var/www/sspanel

if [ ! -f "vendor/autoload.php" ]; then
    echo -e "${RED}✗ vendor/autoload.php 不存在，请先运行 composer install${NC}"
    exit 1
fi

# 检查是否需要运行数据库迁移
echo -e "${YELLOW}正在检查数据库状态...${NC}"
php xcat Migration latest >/dev/null 2>&1 || {
    echo -e "${YELLOW}正在运行数据库迁移...${NC}"
    php xcat Migration new
    php xcat Migration latest
    php xcat Tool importSetting
}

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}修复完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}下一步操作:${NC}"
echo "1. 访问: http://board.moneyfly.club"
echo "2. 如果仍然 404，请检查:"
echo "   - Nginx 配置文件: $NGINX_CONFIG"
echo "   - 项目目录: /var/www/sspanel/public"
echo "   - PHP-FPM 是否运行"
echo ""
echo -e "${YELLOW}创建管理员账号:${NC}"
echo "运行: ./create_admin.sh"
