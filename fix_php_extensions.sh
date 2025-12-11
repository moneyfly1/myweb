#!/bin/bash

# PHP 扩展快速修复脚本
# 用于安装缺失的 PHP 扩展

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
echo -e "${BLUE}PHP 扩展修复脚本${NC}"
echo -e "${BLUE}========================================${NC}"

# 检测 PHP 版本
if ! command -v php &> /dev/null; then
    echo -e "${RED}错误: 未检测到 PHP${NC}"
    exit 1
fi

PHP_VER=$(php -v | head -n 1 | cut -d " " -f 2 | cut -d "." -f 1,2)
PHP_MAJOR=$(echo $PHP_VER | cut -d "." -f 1)
PHP_MINOR=$(echo $PHP_VER | cut -d "." -f 2)

echo -e "${GREEN}检测到 PHP 版本: $PHP_VER${NC}"

# 检测系统
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
else
    echo -e "${RED}错误: 无法检测操作系统${NC}"
    exit 1
fi

echo -e "${GREEN}检测到系统: $OS${NC}"

# 检测 PHP 安装路径
PHP_INI=$(php --ini | grep "Loaded Configuration File" | awk '{print $4}')
echo -e "${GREEN}PHP 配置文件: $PHP_INI${NC}"

# 必需的扩展列表
REQUIRED_EXTENSIONS=("fileinfo" "redis" "yaml" "gmp" "bcmath" "bz2" "curl" "gd" "intl" "mbstring" "mysql" "opcache" "soap" "xml" "zip")

MISSING_EXTENSIONS=()

# 检测缺失的扩展
echo -e "${YELLOW}正在检测缺失的扩展...${NC}"
for ext in "${REQUIRED_EXTENSIONS[@]}"; do
    if ! php -m | grep -qi "^${ext}$"; then
        MISSING_EXTENSIONS+=("$ext")
        echo -e "${RED}✗ 缺失: $ext${NC}"
    else
        echo -e "${GREEN}✓ 已安装: $ext${NC}"
    fi
done

# 如果没有缺失的扩展，直接返回
if [ ${#MISSING_EXTENSIONS[@]} -eq 0 ]; then
    echo -e "${GREEN}✓ 所有必需的 PHP 扩展已安装${NC}"
    exit 0
fi

echo -e "${YELLOW}需要安装的扩展: ${MISSING_EXTENSIONS[*]}${NC}"

# 判断 PHP 安装方式（宝塔面板或其他）
if [[ "$PHP_INI" == *"/www/server/php"* ]]; then
    echo -e "${YELLOW}检测到宝塔面板 PHP 安装${NC}"
    
    # 宝塔面板：尝试通过包管理器安装
    if [[ "$OS" == "debian" || "$OS" == "ubuntu" ]]; then
        echo -e "${YELLOW}使用 apt 安装扩展...${NC}"
        apt update
        
        for ext in "${MISSING_EXTENSIONS[@]}"; do
            case "$ext" in
                "fileinfo")
                    echo -e "${YELLOW}fileinfo 扩展通常已编译在 PHP 中，检查配置...${NC}"
                    # 检查并启用 fileinfo
                    if ! grep -q "^extension=fileinfo" "$PHP_INI" 2>/dev/null; then
                        if grep -q "^;extension=fileinfo" "$PHP_INI" 2>/dev/null; then
                            sed -i 's/^;extension=fileinfo/extension=fileinfo/' "$PHP_INI"
                            echo -e "${GREEN}✓ 已启用 fileinfo 扩展${NC}"
                        else
                            echo "extension=fileinfo" >> "$PHP_INI"
                            echo -e "${GREEN}✓ 已添加 fileinfo 扩展${NC}"
                        fi
                    fi
                    ;;
                "redis")
                    apt install -y php${PHP_MAJOR}.${PHP_MINOR}-redis 2>/dev/null && echo -e "${GREEN}✓ 已安装 redis${NC}" || echo -e "${RED}✗ redis 安装失败${NC}"
                    ;;
                "yaml")
                    apt install -y php${PHP_MAJOR}.${PHP_MINOR}-yaml 2>/dev/null && echo -e "${GREEN}✓ 已安装 yaml${NC}" || echo -e "${RED}✗ yaml 安装失败${NC}"
                    ;;
                "gmp")
                    apt install -y php${PHP_MAJOR}.${PHP_MINOR}-gmp 2>/dev/null && echo -e "${GREEN}✓ 已安装 gmp${NC}" || echo -e "${RED}✗ gmp 安装失败${NC}"
                    ;;
                *)
                    apt install -y php${PHP_MAJOR}.${PHP_MINOR}-${ext} 2>/dev/null && echo -e "${GREEN}✓ 已安装 ${ext}${NC}" || echo -e "${YELLOW}⚠ ${ext} 可能需要手动安装${NC}"
                    ;;
            esac
        done
    elif [[ "$OS" == "centos" || "$OS" == "rhel" || "$OS" == "rocky" || "$OS" == "almalinux" ]]; then
        echo -e "${YELLOW}使用 yum/dnf 安装扩展...${NC}"
        for ext in "${MISSING_EXTENSIONS[@]}"; do
            dnf install -y php-${ext} 2>/dev/null || yum install -y php-${ext} 2>/dev/null || echo -e "${YELLOW}⚠ ${ext} 安装失败，可能需要手动安装${NC}"
        done
    fi
else
    # 标准安装
    echo -e "${YELLOW}使用标准方式安装扩展...${NC}"
    if [[ "$OS" == "debian" || "$OS" == "ubuntu" ]]; then
        apt update
        for ext in "${MISSING_EXTENSIONS[@]}"; do
            apt install -y php${PHP_MAJOR}.${PHP_MINOR}-${ext} 2>/dev/null || echo -e "${YELLOW}⚠ ${ext} 安装失败${NC}"
        done
    elif [[ "$OS" == "centos" || "$OS" == "rhel" || "$OS" == "rocky" || "$OS" == "almalinux" ]]; then
        for ext in "${MISSING_EXTENSIONS[@]}"; do
            dnf install -y php-${ext} 2>/dev/null || yum install -y php-${ext} 2>/dev/null || echo -e "${YELLOW}⚠ ${ext} 安装失败${NC}"
        done
    fi
fi

# 重新检测
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}重新检测扩展...${NC}"
echo -e "${BLUE}========================================${NC}"

STILL_MISSING=()
for ext in "${MISSING_EXTENSIONS[@]}"; do
    if php -m | grep -qi "^${ext}$"; then
        echo -e "${GREEN}✓ ${ext} 已安装${NC}"
    else
        STILL_MISSING+=("$ext")
        echo -e "${RED}✗ ${ext} 仍未安装${NC}"
    fi
done

if [ ${#STILL_MISSING[@]} -gt 0 ]; then
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}以下扩展需要手动安装:${NC}"
    for ext in "${STILL_MISSING[@]}"; do
        echo -e "${YELLOW}  - $ext${NC}"
    done
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}如果使用宝塔面板，请在面板中安装这些扩展${NC}"
    echo -e "${YELLOW}PHP 配置文件: $PHP_INI${NC}"
    exit 1
else
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}✓ 所有扩展已成功安装！${NC}"
    echo -e "${GREEN}========================================${NC}"
    
    # 重启 PHP-FPM（如果存在）
    if systemctl list-units --type=service | grep -q "php.*-fpm"; then
        FPM_SERVICE=$(systemctl list-units --type=service | grep "php.*-fpm" | head -n 1 | awk '{print $1}')
        echo -e "${YELLOW}重启 PHP-FPM 服务: $FPM_SERVICE${NC}"
        systemctl restart "$FPM_SERVICE" 2>/dev/null || true
    fi
    
    exit 0
fi
