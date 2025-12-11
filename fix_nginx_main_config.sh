#!/bin/bash

# 修复宝塔面板 Nginx 主配置文件

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

NGINX_CONF="/www/server/nginx/conf/nginx.conf"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}修复 Nginx 主配置文件${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 备份配置文件
if [ -f "$NGINX_CONF" ]; then
    BACKUP_FILE="${NGINX_CONF}.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$NGINX_CONF" "$BACKUP_FILE"
    echo -e "${GREEN}✓ 已备份配置文件到: $BACKUP_FILE${NC}"
else
    echo -e "${RED}✗ 配置文件不存在: $NGINX_CONF${NC}"
    exit 1
fi

echo ""

# 修复 mime.types 路径
echo -e "${YELLOW}步骤 1: 修复 mime.types 路径...${NC}"

# 检查 mime.types 文件位置
if [ -f "/www/server/nginx/conf/mime.types" ]; then
    MIME_PATH="/www/server/nginx/conf/mime.types"
    echo -e "${GREEN}   找到: $MIME_PATH${NC}"
elif [ -f "/etc/nginx/mime.types" ]; then
    MIME_PATH="/etc/nginx/mime.types"
    echo -e "${GREEN}   找到: $MIME_PATH${NC}"
else
    echo -e "${YELLOW}   ⚠ 未找到 mime.types，将使用默认路径${NC}"
    MIME_PATH="/etc/nginx/mime.types"
fi

# 修复配置（移除双分号，使用正确路径）
sed -i "s|include\s*mime.types;;|include       ${MIME_PATH};|g" "$NGINX_CONF"
sed -i "s|include\s*mime.types;|include       ${MIME_PATH};|g" "$NGINX_CONF"

echo -e "${GREEN}✓ mime.types 路径已修复${NC}"
echo ""

# 确保被注释的配置保持注释状态
echo -e "${YELLOW}步骤 2: 检查被注释的配置...${NC}"
if grep -q "^#.*include.*proxy.conf" "$NGINX_CONF"; then
    echo -e "${GREEN}   ✓ proxy.conf 已正确注释${NC}"
else
    echo -e "${YELLOW}   ⚠ proxy.conf 未被注释，正在注释...${NC}"
    sed -i 's|^[^#]*include.*proxy.conf|# \t\tinclude proxy.conf;|g' "$NGINX_CONF"
fi

if grep -q "^#.*include.*enable-php.conf" "$NGINX_CONF"; then
    echo -e "${GREEN}   ✓ enable-php.conf 已正确注释${NC}"
else
    echo -e "${YELLOW}   ⚠ enable-php.conf 未被注释，正在注释...${NC}"
    sed -i 's|^[^#]*include.*enable-php.conf|#         include enable-php.conf;|g' "$NGINX_CONF"
fi
echo ""

# 测试配置
echo -e "${YELLOW}步骤 3: 测试配置...${NC}"
if nginx -t 2>&1; then
    echo -e "${GREEN}✓ Nginx 配置测试通过${NC}"
else
    echo -e "${RED}✗ Nginx 配置测试失败${NC}"
    echo -e "${YELLOW}   正在恢复备份...${NC}"
    cp "$BACKUP_FILE" "$NGINX_CONF"
    echo -e "${YELLOW}   已恢复备份，请手动检查配置${NC}"
    exit 1
fi
echo ""

# 显示修复后的关键配置
echo -e "${YELLOW}步骤 4: 显示修复后的关键配置...${NC}"
echo "   mime.types 配置:"
grep "mime.types" "$NGINX_CONF" | sed 's/^/      /'
echo ""
echo "   被注释的配置:"
grep -E "(proxy.conf|enable-php.conf)" "$NGINX_CONF" | sed 's/^/      /'
echo ""

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}修复完成！${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${YELLOW}下一步：${NC}"
echo "1. 重载 Nginx: systemctl reload nginx"
echo "2. 或重启 Nginx: systemctl restart nginx"
echo ""
