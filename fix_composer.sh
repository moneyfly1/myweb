#!/bin/bash

# Composer putenv 修复脚本

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
echo -e "${BLUE}Composer putenv 修复脚本${NC}"
echo -e "${BLUE}========================================${NC}"

# 检查 putenv 函数
echo -e "${YELLOW}检查 putenv 函数...${NC}"
if php -r "if (!function_exists('putenv')) { exit(1); }" 2>/dev/null; then
    echo -e "${GREEN}✓ putenv 函数可用${NC}"
    
    # 测试 Composer
    if composer --version >/dev/null 2>&1; then
        COMPOSER_VER=$(composer --version 2>/dev/null | head -n 1 | grep -oP 'version \K[0-9.]+' || echo "unknown")
        echo -e "${GREEN}✓ Composer 正常工作，版本: $COMPOSER_VER${NC}"
        exit 0
    else
        echo -e "${YELLOW}⚠ Composer 无法运行，可能需要重新安装${NC}"
    fi
else
    echo -e "${RED}✗ putenv 函数不可用${NC}"
fi

# 获取 PHP 配置文件
PHP_INI=$(php --ini | grep "Loaded Configuration File" | awk '{print $4}')
echo -e "${GREEN}PHP 配置文件: $PHP_INI${NC}"

# 检查并修复 disable_functions
echo -e "${YELLOW}检查 disable_functions 配置...${NC}"

FIXED=0

# 修复主配置文件
if [ -f "$PHP_INI" ]; then
    if grep -q "disable_functions.*putenv" "$PHP_INI" 2>/dev/null; then
        echo -e "${YELLOW}  在 $PHP_INI 中发现 putenv，正在移除...${NC}"
        
        # 备份
        cp "$PHP_INI" "${PHP_INI}.bak.$(date +%Y%m%d_%H%M%S)"
        
        # 移除 putenv（处理各种格式）
        sed -i 's/disable_functions = \(.*\)putenv\(.*\)/disable_functions = \1\2/' "$PHP_INI" 2>/dev/null || true
        sed -i 's/disable_functions = \(.*\),putenv\(.*\)/disable_functions = \1\2/' "$PHP_INI" 2>/dev/null || true
        sed -i 's/disable_functions = putenv,\(.*\)/disable_functions = \1/' "$PHP_INI" 2>/dev/null || true
        sed -i 's/disable_functions = putenv$/;disable_functions = /' "$PHP_INI" 2>/dev/null || true
        
        FIXED=1
        echo -e "${GREEN}  ✓ 已从 $PHP_INI 中移除 putenv${NC}"
    fi
fi

# 如果使用宝塔面板，检查其他配置文件
if [[ "$PHP_INI" == *"/www/server/php"* ]]; then
    PHP_VERSION_DIR=$(echo "$PHP_INI" | grep -oP '/www/server/php/\K[0-9]+')
    
    # 检查 php.ini
    FPM_INI="/www/server/php/${PHP_VERSION_DIR}/etc/php.ini"
    if [ -f "$FPM_INI" ] && [ "$FPM_INI" != "$PHP_INI" ]; then
        if grep -q "disable_functions.*putenv" "$FPM_INI" 2>/dev/null; then
            echo -e "${YELLOW}  在 $FPM_INI 中发现 putenv，正在移除...${NC}"
            cp "$FPM_INI" "${FPM_INI}.bak.$(date +%Y%m%d_%H%M%S)"
            sed -i 's/disable_functions = \(.*\)putenv\(.*\)/disable_functions = \1\2/' "$FPM_INI" 2>/dev/null || true
            sed -i 's/disable_functions = \(.*\),putenv\(.*\)/disable_functions = \1\2/' "$FPM_INI" 2>/dev/null || true
            sed -i 's/disable_functions = putenv,\(.*\)/disable_functions = \1/' "$FPM_INI" 2>/dev/null || true
            sed -i 's/disable_functions = putenv$/;disable_functions = /' "$FPM_INI" 2>/dev/null || true
            FIXED=1
            echo -e "${GREEN}  ✓ 已从 $FPM_INI 中移除 putenv${NC}"
        fi
    fi
    
    # 检查 php-cli.ini
    CLI_INI="/www/server/php/${PHP_VERSION_DIR}/etc/php-cli.ini"
    if [ -f "$CLI_INI" ] && [ "$CLI_INI" != "$PHP_INI" ]; then
        if grep -q "disable_functions.*putenv" "$CLI_INI" 2>/dev/null; then
            echo -e "${YELLOW}  在 $CLI_INI 中发现 putenv，正在移除...${NC}"
            cp "$CLI_INI" "${CLI_INI}.bak.$(date +%Y%m%d_%H%M%S)"
            sed -i 's/disable_functions = \(.*\)putenv\(.*\)/disable_functions = \1\2/' "$CLI_INI" 2>/dev/null || true
            sed -i 's/disable_functions = \(.*\),putenv\(.*\)/disable_functions = \1\2/' "$CLI_INI" 2>/dev/null || true
            sed -i 's/disable_functions = putenv,\(.*\)/disable_functions = \1/' "$CLI_INI" 2>/dev/null || true
            sed -i 's/disable_functions = putenv$/;disable_functions = /' "$CLI_INI" 2>/dev/null || true
            FIXED=1
            echo -e "${GREEN}  ✓ 已从 $CLI_INI 中移除 putenv${NC}"
        fi
    fi
fi

if [ $FIXED -eq 0 ]; then
    echo -e "${YELLOW}⚠ 未在配置文件中发现 putenv，可能在其他位置被禁用${NC}"
    echo -e "${YELLOW}请手动检查 PHP 配置文件${NC}"
fi

# 重启 PHP-FPM
if [[ "$PHP_INI" == *"/www/server/php"* ]]; then
    PHP_VERSION_DIR=$(echo "$PHP_INI" | grep -oP '/www/server/php/\K[0-9]+')
    FPM_SERVICE="php-fpm-${PHP_VERSION_DIR}"
    
    if systemctl list-units --type=service | grep -q "$FPM_SERVICE" 2>/dev/null; then
        echo -e "${YELLOW}重启 PHP-FPM 服务...${NC}"
        systemctl restart "$FPM_SERVICE" 2>/dev/null || /etc/init.d/php-fpm-${PHP_VERSION_DIR} restart 2>/dev/null || true
        sleep 2
    fi
fi

# 重新检查
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}验证修复...${NC}"
echo -e "${BLUE}========================================${NC}"

if php -r "if (!function_exists('putenv')) { exit(1); }" 2>/dev/null; then
    echo -e "${GREEN}✓ putenv 函数现在可用${NC}"
    
    if composer --version >/dev/null 2>&1; then
        COMPOSER_VER=$(composer --version 2>/dev/null | head -n 1 | grep -oP 'version \K[0-9.]+' || echo "unknown")
        echo -e "${GREEN}✓ Composer 正常工作，版本: $COMPOSER_VER${NC}"
        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}修复完成！${NC}"
        exit 0
    else
        echo -e "${YELLOW}⚠ Composer 仍然无法运行，可能需要重新安装${NC}"
        echo -e "${YELLOW}运行: rm -f /usr/local/bin/composer && 重新安装 Composer${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ putenv 仍然不可用${NC}"
    echo -e "${YELLOW}请手动检查 PHP 配置文件${NC}"
    exit 1
fi
