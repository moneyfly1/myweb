#!/bin/bash

# SSPanel-UIM 自动部署脚本
# 适用于 Debian/Ubuntu 系统

set -e

echo "=========================================="
echo "SSPanel-UIM 自动部署脚本"
echo "=========================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}请使用 root 用户运行此脚本${NC}"
    exit 1
fi

# 检测系统
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
    VER=$VERSION_ID
else
    echo -e "${RED}无法检测操作系统${NC}"
    exit 1
fi

echo -e "${GREEN}检测到系统: $OS $VER${NC}"

# 安装基础环境
echo -e "${YELLOW}正在安装基础环境...${NC}"
apt update && apt upgrade -y
apt install -y curl wget git unzip

# 安装 PHP 8.2
echo -e "${YELLOW}正在安装 PHP 8.2...${NC}"
apt install -y php8.2 php8.2-fpm php8.2-cli php8.2-common \
    php8.2-mysql php8.2-zip php8.2-gd php8.2-mbstring \
    php8.2-curl php8.2-xml php8.2-bcmath php8.2-redis \
    php8.2-yaml php8.2-opcache

# 安装 MariaDB
echo -e "${YELLOW}正在安装 MariaDB...${NC}"
apt install -y mariadb-server mariadb-client

# 安装 Redis
echo -e "${YELLOW}正在安装 Redis...${NC}"
apt install -y redis-server

# 安装 Nginx
echo -e "${YELLOW}正在安装 Nginx...${NC}"
apt install -y nginx

# 安装 Composer
echo -e "${YELLOW}正在安装 Composer...${NC}"
if [ ! -f /usr/local/bin/composer ]; then
    curl -sS https://getcomposer.org/installer | php
    mv composer.phar /usr/local/bin/composer
    chmod +x /usr/local/bin/composer
fi

# 配置 MariaDB
echo -e "${YELLOW}正在配置 MariaDB...${NC}"
systemctl start mariadb
systemctl enable mariadb

# 提示用户设置数据库
echo -e "${YELLOW}请设置 MariaDB root 密码并创建数据库:${NC}"
echo "运行以下命令创建数据库和用户:"
echo "mysql -u root -p"
echo "CREATE DATABASE sspanel CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
echo "CREATE USER 'sspanel'@'localhost' IDENTIFIED BY 'your_password';"
echo "GRANT ALL PRIVILEGES ON sspanel.* TO 'sspanel'@'localhost';"
echo "FLUSH PRIVILEGES;"
echo "EXIT;"
read -p "按 Enter 继续..."

# 启动服务
echo -e "${YELLOW}正在启动服务...${NC}"
systemctl start redis
systemctl enable redis
systemctl start php8.2-fpm
systemctl enable php8.2-fpm
systemctl start nginx
systemctl enable nginx

# 创建项目目录
echo -e "${YELLOW}正在创建项目目录...${NC}"
mkdir -p /var/www
cd /var/www

# 提示用户克隆项目
echo -e "${YELLOW}请手动克隆项目到 /var/www/sspanel:${NC}"
echo "git clone https://github.com/your-username/SSPanel-UIM.git sspanel"
read -p "克隆完成后按 Enter 继续..."

if [ ! -d "/var/www/sspanel" ]; then
    echo -e "${RED}项目目录不存在，请先克隆项目${NC}"
    exit 1
fi

cd /var/www/sspanel

# 安装依赖
echo -e "${YELLOW}正在安装 PHP 依赖...${NC}"
composer install --no-dev --optimize-autoloader

# 配置项目
echo -e "${YELLOW}正在配置项目...${NC}"
if [ ! -f "config/.config.php" ]; then
    cp config/.config.example.php config/.config.php
    echo -e "${YELLOW}请编辑配置文件: config/.config.php${NC}"
    echo "重要配置项:"
    echo "  - \$_ENV['key'] = '随机字符串'"
    echo "  - \$_ENV['baseUrl'] = 'https://your-domain.com'"
    echo "  - \$_ENV['muKey'] = '随机字符串'"
    echo "  - 数据库配置"
    read -p "配置完成后按 Enter 继续..."
fi

if [ ! -f "config/appprofile.php" ]; then
    cp config/appprofile.example.php config/appprofile.php
fi

# 设置权限
echo -e "${YELLOW}正在设置文件权限...${NC}"
chown -R www-data:www-data /var/www/sspanel
chmod -R 755 /var/www/sspanel
chmod -R 777 storage/
chmod -R 777 public/clients/

# 运行数据库迁移
echo -e "${YELLOW}正在运行数据库迁移...${NC}"
php xcat Migration

# 创建管理员
echo -e "${YELLOW}正在创建管理员账户...${NC}"
php xcat User createAdmin

# 配置 Nginx
echo -e "${YELLOW}请手动配置 Nginx${NC}"
echo "参考 DEPLOY.md 中的 Nginx 配置示例"
read -p "配置完成后按 Enter 继续..."

# 配置定时任务
echo -e "${YELLOW}正在配置定时任务...${NC}"
(crontab -l 2>/dev/null; echo "*/5 * * * * cd /var/www/sspanel && /usr/bin/php xcat Cron >> /dev/null 2>&1") | crontab -

echo -e "${GREEN}=========================================="
echo "部署完成！"
echo "==========================================${NC}"
echo ""
echo "下一步:"
echo "1. 配置 Nginx（参考 DEPLOY.md）"
echo "2. 配置 SSL 证书（Let's Encrypt）"
echo "3. 访问管理后台配置节点采集功能"
echo ""
echo "管理后台: https://your-domain.com/auth/login"
echo "节点采集: https://your-domain.com/admin/node/collector"
