#!/bin/bash

# 修复 Nginx 配置未加载的问题

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}修复 Nginx 配置加载问题${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

DOMAIN="board.moneyfly.club"
BT_CONFIG="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
NGINX_MAIN_CONF="/etc/nginx/nginx.conf"
NGINX_BT_CONF="/www/server/nginx/conf/nginx.conf"

# 步骤 1: 检查站点配置是否被加载
echo -e "${YELLOW}步骤 1: 检查站点配置是否被 Nginx 加载...${NC}"
if command -v nginx >/dev/null 2>&1; then
    ACTIVE_CONFIG=$(nginx -T 2>/dev/null | grep -A 30 "server_name.*${DOMAIN}")
    if [ ! -z "$ACTIVE_CONFIG" ]; then
        echo -e "${GREEN}✓ 站点配置已被加载${NC}"
        echo "$ACTIVE_CONFIG" | grep -E "(server_name|root|fastcgi_pass)" | sed 's/^/   /'
    else
        echo -e "${RED}✗ 站点配置未被加载！${NC}"
        echo -e "${YELLOW}   这是问题的根源 - Nginx 没有加载站点配置${NC}"
    fi
else
    echo -e "${RED}✗ nginx 命令不可用${NC}"
fi
echo ""

# 步骤 2: 检查 Nginx 主配置文件
echo -e "${YELLOW}步骤 2: 检查 Nginx 主配置文件...${NC}"
CONFIG_FILES=("$NGINX_MAIN_CONF" "$NGINX_BT_CONF")

for config_file in "${CONFIG_FILES[@]}"; do
    if [ -f "$config_file" ]; then
        echo "   检查: $config_file"
        
        # 检查是否包含宝塔面板配置目录
        if grep -q "/www/server/panel/vhost/nginx" "$config_file"; then
            echo -e "${GREEN}   ✓ 包含宝塔面板配置目录${NC}"
            grep "/www/server/panel/vhost/nginx" "$config_file" | grep -v "#" | sed 's/^/      /'
        else
            echo -e "${YELLOW}   ⚠ 未包含宝塔面板配置目录${NC}"
        fi
        
        # 检查是否有默认站点配置
        if grep -q "default_server" "$config_file"; then
            echo -e "${YELLOW}   ⚠ 发现 default_server 配置，可能覆盖站点配置${NC}"
            grep "default_server" "$config_file" | sed 's/^/      /'
        fi
    fi
done
echo ""

# 步骤 3: 检查默认站点配置
echo -e "${YELLOW}步骤 3: 检查默认站点配置...${NC}"
DEFAULT_CONFIGS=(
    "/etc/nginx/sites-enabled/default"
    "/etc/nginx/conf.d/default.conf"
    "/www/server/nginx/conf/vhost/default.conf"
)

for config in "${DEFAULT_CONFIGS[@]}"; do
    if [ -f "$config" ]; then
        echo "   找到: $config"
        if grep -q "default_server\|listen.*80" "$config"; then
            echo -e "${YELLOW}   ⚠ 这可能是默认配置，可能覆盖站点配置${NC}"
            echo -e "${YELLOW}   建议：禁用或修改此配置${NC}"
        fi
    fi
done
echo ""

# 步骤 4: 修复 Nginx 主配置（如果需要）
echo -e "${YELLOW}步骤 4: 修复 Nginx 主配置...${NC}"
if [ -f "$NGINX_BT_CONF" ]; then
    if ! grep -q "/www/server/panel/vhost/nginx" "$NGINX_BT_CONF"; then
        echo -e "${YELLOW}   ⚠ 宝塔面板 Nginx 配置缺少站点配置目录，正在添加...${NC}"
        
        # 备份
        cp "$NGINX_BT_CONF" "${NGINX_BT_CONF}.backup.$(date +%Y%m%d_%H%M%S)"
        
        # 查找 http 块
        if grep -q "^\s*include\s.*vhost.*nginx" "$NGINX_BT_CONF"; then
            echo -e "${GREEN}   ✓ 配置已存在（可能被注释）${NC}"
        else
            # 在 http 块末尾添加 include
            sed -i '/^}/i\    include /www/server/panel/vhost/nginx/*.conf;' "$NGINX_BT_CONF"
            echo -e "${GREEN}   ✓ 已添加站点配置目录${NC}"
        fi
    else
        echo -e "${GREEN}   ✓ 配置已包含站点配置目录${NC}"
    fi
fi
echo ""

# 步骤 5: 确保站点配置文件正确
echo -e "${YELLOW}步骤 5: 验证站点配置文件...${NC}"
if [ -f "$BT_CONFIG" ]; then
    # 检查 root 路径
    ROOT_PATH=$(grep -E "^\s*root\s+" "$BT_CONFIG" | head -1 | awk '{print $2}' | tr -d ';' | sed 's|/$||')
    if [ "$ROOT_PATH" = "/www/wwwroot/board.moneyfly.club/public" ]; then
        echo -e "${GREEN}   ✓ Root 路径正确${NC}"
    else
        echo -e "${RED}   ✗ Root 路径不正确: $ROOT_PATH${NC}"
    fi
    
    # 检查 PHP 配置
    if grep -q "fastcgi_pass.*php-cgi-82.sock" "$BT_CONFIG"; then
        echo -e "${GREEN}   ✓ PHP 配置正确${NC}"
    else
        echo -e "${RED}   ✗ PHP 配置不正确${NC}"
    fi
else
    echo -e "${RED}   ✗ 站点配置文件不存在${NC}"
fi
echo ""

# 步骤 6: 测试并重载
echo -e "${YELLOW}步骤 6: 测试配置并重载 Nginx...${NC}"
if nginx -t 2>&1; then
    echo -e "${GREEN}✓ Nginx 配置测试通过${NC}"
    
    # 重载 Nginx
    if systemctl reload nginx >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Nginx 已重载${NC}"
    elif /etc/init.d/nginx reload >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Nginx 已重载${NC}"
    else
        echo -e "${YELLOW}⚠ 自动重载失败，请手动重载${NC}"
    fi
else
    echo -e "${RED}✗ Nginx 配置测试失败${NC}"
    exit 1
fi
echo ""

# 步骤 7: 再次验证配置是否加载
echo -e "${YELLOW}步骤 7: 再次验证配置是否加载...${NC}"
sleep 2
ACTIVE_CONFIG=$(nginx -T 2>/dev/null | grep -A 30 "server_name.*${DOMAIN}")
if [ ! -z "$ACTIVE_CONFIG" ]; then
    echo -e "${GREEN}✓ 站点配置现在已被加载${NC}"
    echo "$ACTIVE_CONFIG" | head -15 | sed 's/^/   /'
else
    echo -e "${RED}✗ 站点配置仍然未被加载${NC}"
    echo -e "${YELLOW}   可能需要重启 Nginx 或检查配置路径${NC}"
fi
echo ""

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}修复完成！${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${YELLOW}测试命令：${NC}"
echo "curl -I http://${DOMAIN}/auth/login"
echo "curl http://${DOMAIN}/test.php"
echo ""
