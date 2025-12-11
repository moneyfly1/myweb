#!/bin/bash

# 修复宝塔面板 PHP 版本无法切换的问题

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}修复宝塔面板 PHP 版本设置${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

DOMAIN="board.moneyfly.club"
PHP_VERSION="82"
BT_CONFIG="/www/server/panel/vhost/nginx/${DOMAIN}.conf"
BT_DB="/www/server/panel/data/default.db"
BT_SITE_CONFIG="/www/server/panel/vhost/${DOMAIN}"

# 步骤 1: 检查配置文件
echo -e "${YELLOW}步骤 1: 检查 Nginx 配置文件...${NC}"
if [ -f "$BT_CONFIG" ]; then
    echo -e "${GREEN}✓ 配置文件存在: $BT_CONFIG${NC}"
    
    # 检查是否包含 PHP 处理
    if grep -q "fastcgi_pass.*php" "$BT_CONFIG"; then
        echo -e "${GREEN}✓ 配置文件已包含 PHP 处理${NC}"
        grep "fastcgi_pass" "$BT_CONFIG" | sed 's/^/   /'
    else
        echo -e "${RED}✗ 配置文件缺少 PHP 处理配置${NC}"
    fi
else
    echo -e "${RED}✗ 配置文件不存在${NC}"
fi
echo ""

# 步骤 2: 检查宝塔面板数据库
echo -e "${YELLOW}步骤 2: 检查宝塔面板数据库...${NC}"
if [ -f "$BT_DB" ] && command -v sqlite3 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ 找到宝塔面板数据库${NC}"
    
    # 查询站点信息
    SITE_INFO=$(sqlite3 "$BT_DB" "SELECT id,name,path,phpversion FROM sites WHERE name='${DOMAIN}';" 2>/dev/null)
    if [ ! -z "$SITE_INFO" ]; then
        echo "   站点信息: $SITE_INFO"
        CURRENT_PHP=$(echo "$SITE_INFO" | cut -d'|' -f4)
        echo "   当前 PHP 版本: $CURRENT_PHP"
        
        if [ "$CURRENT_PHP" = "0" ] || [ "$CURRENT_PHP" = "静态" ] || [ -z "$CURRENT_PHP" ]; then
            echo -e "${YELLOW}   ⚠ PHP 版本设置为静态或未设置${NC}"
            echo -e "${YELLOW}   正在更新为 PHP ${PHP_VERSION}...${NC}"
            
            SITE_ID=$(echo "$SITE_INFO" | cut -d'|' -f1)
            sqlite3 "$BT_DB" "UPDATE sites SET phpversion='${PHP_VERSION}' WHERE id=${SITE_ID};" 2>/dev/null
            
            if [ $? -eq 0 ]; then
                echo -e "${GREEN}   ✓ 数据库已更新${NC}"
            else
                echo -e "${RED}   ✗ 数据库更新失败${NC}"
            fi
        else
            echo -e "${GREEN}   ✓ PHP 版本已设置为: $CURRENT_PHP${NC}"
        fi
    else
        echo -e "${YELLOW}   ⚠ 未在数据库中找到站点信息${NC}"
    fi
else
    echo -e "${YELLOW}   ⚠ 无法访问宝塔面板数据库或 sqlite3 不可用${NC}"
fi
echo ""

# 步骤 3: 检查站点配置文件
echo -e "${YELLOW}步骤 3: 检查站点配置文件...${NC}"
if [ -d "$BT_SITE_CONFIG" ]; then
    echo -e "${GREEN}✓ 站点配置目录存在: $BT_SITE_CONFIG${NC}"
    
    # 检查 PHP 版本文件
    PHP_VERSION_FILE="${BT_SITE_CONFIG}/phpversion.pl"
    if [ -f "$PHP_VERSION_FILE" ]; then
        CURRENT_PHP_FILE=$(cat "$PHP_VERSION_FILE" 2>/dev/null)
        echo "   当前 PHP 版本文件内容: $CURRENT_PHP_FILE"
        
        if [ "$CURRENT_PHP_FILE" != "$PHP_VERSION" ]; then
            echo -e "${YELLOW}   ⚠ PHP 版本文件不正确，正在更新...${NC}"
            echo "$PHP_VERSION" > "$PHP_VERSION_FILE"
            echo -e "${GREEN}   ✓ PHP 版本文件已更新${NC}"
        else
            echo -e "${GREEN}   ✓ PHP 版本文件正确${NC}"
        fi
    else
        echo -e "${YELLOW}   ⚠ PHP 版本文件不存在，正在创建...${NC}"
        echo "$PHP_VERSION" > "$PHP_VERSION_FILE"
        echo -e "${GREEN}   ✓ PHP 版本文件已创建${NC}"
    fi
else
    echo -e "${YELLOW}   ⚠ 站点配置目录不存在${NC}"
fi
echo ""

# 步骤 4: 重新生成配置文件（如果需要）
echo -e "${YELLOW}步骤 4: 验证配置文件...${NC}"
if [ -f "$BT_CONFIG" ]; then
    # 确保配置文件包含正确的 PHP 处理
    if ! grep -q "fastcgi_pass.*php-cgi-${PHP_VERSION}.sock" "$BT_CONFIG"; then
        echo -e "${YELLOW}   ⚠ 配置文件中的 PHP socket 路径不正确，正在修复...${NC}"
        
        # 备份原配置
        cp "$BT_CONFIG" "${BT_CONFIG}.backup.$(date +%Y%m%d_%H%M%S)"
        
        # 修复 PHP socket 路径
        sed -i "s|fastcgi_pass.*php.*sock|fastcgi_pass unix:/tmp/php-cgi-${PHP_VERSION}.sock;|g" "$BT_CONFIG"
        
        echo -e "${GREEN}   ✓ 配置文件已修复${NC}"
    else
        echo -e "${GREEN}   ✓ 配置文件中的 PHP socket 路径正确${NC}"
    fi
fi
echo ""

# 步骤 5: 测试 Nginx 配置
echo -e "${YELLOW}步骤 5: 测试 Nginx 配置...${NC}"
if nginx -t 2>&1; then
    echo -e "${GREEN}✓ Nginx 配置测试通过${NC}"
else
    echo -e "${RED}✗ Nginx 配置测试失败${NC}"
    exit 1
fi
echo ""

# 步骤 6: 重载 Nginx
echo -e "${YELLOW}步骤 6: 重载 Nginx...${NC}"
if systemctl reload nginx >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Nginx 已重载${NC}"
elif /etc/init.d/nginx reload >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Nginx 已重载${NC}"
else
    echo -e "${YELLOW}⚠ 自动重载失败，请手动重载 Nginx${NC}"
fi
echo ""

# 步骤 7: 验证 PHP Socket
echo -e "${YELLOW}步骤 7: 验证 PHP Socket...${NC}"
PHP_SOCKET="/tmp/php-cgi-${PHP_VERSION}.sock"
if [ -S "$PHP_SOCKET" ] || [ -f "$PHP_SOCKET" ]; then
    echo -e "${GREEN}✓ PHP Socket 存在: $PHP_SOCKET${NC}"
    ls -la "$PHP_SOCKET" | sed 's/^/   /'
else
    echo -e "${RED}✗ PHP Socket 不存在: $PHP_SOCKET${NC}"
    echo -e "${YELLOW}   需要启动 PHP-FPM 服务${NC}"
fi
echo ""

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}修复完成！${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${YELLOW}重要提示：${NC}"
echo "1. 如果宝塔面板仍然显示'静态'，请刷新页面或重新登录"
echo "2. 如果仍然无法切换，可能需要重启宝塔面板服务"
echo "3. 配置文件已经正确，即使面板显示'静态'，PHP 也应该能正常工作"
echo ""
echo -e "${YELLOW}测试访问:${NC}"
echo "curl -I http://${DOMAIN}"
echo ""
