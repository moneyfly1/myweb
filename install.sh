#!/bin/bash

# SSPanel-UIM 节点采集版 - 一键安装脚本
# 基于官方安装指南优化
# 适用于 Debian 12 / Ubuntu 20.04+ / CentOS 8+

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

# 检测系统
detect_system() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        VER=$VERSION_ID
    else
        echo -e "${RED}错误: 无法检测操作系统${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}检测到系统: $OS $VER${NC}"
    
    case $OS in
        debian)
            if [ "$VER" != "12" ]; then
                echo -e "${YELLOW}警告: 推荐使用 Debian 12，当前版本: $VER${NC}"
            fi
            ;;
        ubuntu)
            if [[ "$VER" != "20.04" && "$VER" != "22.04" ]]; then
                echo -e "${YELLOW}警告: 推荐使用 Ubuntu 20.04/22.04，当前版本: $VER${NC}"
            fi
            ;;
        centos|rhel|rocky|almalinux)
            echo -e "${GREEN}检测到 CentOS/RHEL 系列${NC}"
            ;;
        *)
            echo -e "${YELLOW}警告: 未测试的系统，可能无法正常工作${NC}"
            ;;
    esac
}

# 检测软件是否已安装
check_installed() {
    local software=$1
    if command -v "$software" &> /dev/null; then
        return 0  # 已安装
    else
        return 1  # 未安装
    fi
}

# 检测 PHP 版本
check_php_version() {
    if command -v php &> /dev/null; then
        PHP_VER=$(php -v | head -n 1 | cut -d " " -f 2 | cut -d "." -f 1,2)
        MAJOR=$(echo $PHP_VER | cut -d "." -f 1)
        MINOR=$(echo $PHP_VER | cut -d "." -f 2)
        
        if [ "$MAJOR" -ge 8 ] && [ "$MINOR" -ge 2 ]; then
            return 0  # PHP 版本符合要求
        else
            return 1  # PHP 版本不符合要求
        fi
    else
        return 1  # PHP 未安装
    fi
}

# 检测服务是否运行
check_service_running() {
    local service=$1
    if systemctl is-active --quiet "$service" 2>/dev/null; then
        return 0  # 服务正在运行
    else
        return 1  # 服务未运行
    fi
}

# 检测 PHP 扩展是否已安装
check_php_extension() {
    local ext=$1
    # 首先检查 php -m 输出
    if php -m | grep -qi "^${ext}$"; then
        return 0  # 扩展已安装
    fi
    
    # 对于某些扩展，检查别名或相关扩展
    case "$ext" in
        "mysql")
            # mysql 扩展在 PHP 7.0+ 已移除，检查 mysqli 或 pdo_mysql
            if php -m | grep -qi "^mysqli$\|^pdo_mysql$"; then
                return 0
            fi
            ;;
        "opcache")
            # opcache 可能已编译但未启用，检查配置
            PHP_INI=$(php --ini | grep "Loaded Configuration File" | awk '{print $4}')
            if [ -f "$PHP_INI" ] && grep -qi "^zend_extension.*opcache\|^;zend_extension.*opcache" "$PHP_INI" 2>/dev/null; then
                # 检查是否被注释
                if ! grep -q "^;zend_extension.*opcache" "$PHP_INI" 2>/dev/null || grep -q "^zend_extension.*opcache" "$PHP_INI" 2>/dev/null; then
                    # 尝试加载 opcache
                    if php -r "if (function_exists('opcache_get_status')) exit(0); else exit(1);" 2>/dev/null; then
                        return 0
                    fi
                fi
            fi
            ;;
    esac
    
    return 1  # 扩展未安装
}

# 安装 PHP 扩展
install_php_extensions() {
    echo -e "${BLUE}检查并安装 PHP 扩展...${NC}"
    
    # 获取 PHP 版本
    PHP_VER=$(php -v | head -n 1 | cut -d " " -f 2 | cut -d "." -f 1,2)
    PHP_MAJOR=$(echo $PHP_VER | cut -d "." -f 1)
    PHP_MINOR=$(echo $PHP_VER | cut -d "." -f 2)
    
    # 必需的扩展列表（mysql 在 PHP 7.0+ 已被移除，使用 mysqli 或 pdo_mysql）
    REQUIRED_EXTENSIONS=("fileinfo" "redis" "yaml" "gmp" "bcmath" "bz2" "curl" "gd" "intl" "mbstring" "mysqli" "opcache" "soap" "xml" "zip")
    
    MISSING_EXTENSIONS=()
    
    # 检测缺失的扩展
    echo -e "${YELLOW}正在检测 PHP 扩展...${NC}"
    for ext in "${REQUIRED_EXTENSIONS[@]}"; do
        if check_php_extension "$ext"; then
            echo -e "${GREEN}  ✓ $ext${NC}"
        else
            echo -e "${YELLOW}  ✗ $ext (缺失)${NC}"
            MISSING_EXTENSIONS+=("$ext")
        fi
    done
    
    # 如果没有缺失的扩展，直接返回
    if [ ${#MISSING_EXTENSIONS[@]} -eq 0 ]; then
        echo -e "${GREEN}✓ 所有必需的 PHP 扩展已安装${NC}"
        return 0
    fi
    
    echo -e "${YELLOW}检测到缺失的 PHP 扩展: ${MISSING_EXTENSIONS[*]}${NC}"
    echo -e "${YELLOW}正在安装缺失的扩展...${NC}"
    
    # 检测 PHP 安装路径（宝塔面板或其他）
    PHP_INI=$(php --ini | grep "Loaded Configuration File" | awk '{print $4}')
    PHP_DIR=$(dirname "$PHP_INI")
    
    # 判断 PHP 安装方式
    if [[ "$PHP_INI" == *"/www/server/php"* ]]; then
        # 宝塔面板安装的 PHP
        echo -e "${YELLOW}检测到宝塔面板 PHP 安装，尝试通过宝塔方式安装扩展...${NC}"
        
        # 尝试通过 apt/yum 安装扩展
        if [[ "$OS" == "debian" || "$OS" == "ubuntu" ]]; then
            for ext in "${MISSING_EXTENSIONS[@]}"; do
                case "$ext" in
                    "fileinfo")
                        apt install -y php${PHP_MAJOR}.${PHP_MINOR}-fileinfo 2>/dev/null || echo "fileinfo 扩展需要手动启用"
                        ;;
                    "redis")
                        apt install -y php${PHP_MAJOR}.${PHP_MINOR}-redis 2>/dev/null || true
                        ;;
                    "yaml")
                        apt install -y php${PHP_MAJOR}.${PHP_MINOR}-yaml 2>/dev/null || true
                        ;;
                    "gmp")
                        apt install -y php${PHP_MAJOR}.${PHP_MINOR}-gmp 2>/dev/null || true
                        ;;
                    *)
                        apt install -y php${PHP_MAJOR}.${PHP_MINOR}-${ext} 2>/dev/null || true
                        ;;
                esac
            done
        elif [[ "$OS" == "centos" || "$OS" == "rhel" || "$OS" == "rocky" || "$OS" == "almalinux" ]]; then
            for ext in "${MISSING_EXTENSIONS[@]}"; do
                case "$ext" in
                    "redis")
                        dnf install -y php-redis 2>/dev/null || yum install -y php-redis 2>/dev/null || true
                        ;;
                    "yaml")
                        dnf install -y php-yaml 2>/dev/null || yum install -y php-yaml 2>/dev/null || true
                        ;;
                    *)
                        dnf install -y php-${ext} 2>/dev/null || yum install -y php-${ext} 2>/dev/null || true
                        ;;
                esac
            done
        fi
    else
        # 标准安装的 PHP
        if [[ "$OS" == "debian" || "$OS" == "ubuntu" ]]; then
            for ext in "${MISSING_EXTENSIONS[@]}"; do
                apt install -y php${PHP_MAJOR}.${PHP_MINOR}-${ext} 2>/dev/null || true
            done
        elif [[ "$OS" == "centos" || "$OS" == "rhel" || "$OS" == "rocky" || "$OS" == "almalinux" ]]; then
            for ext in "${MISSING_EXTENSIONS[@]}"; do
                dnf install -y php-${ext} 2>/dev/null || yum install -y php-${ext} 2>/dev/null || true
            done
        fi
    fi
    
    # 对于 fileinfo 扩展，通常已经编译在 PHP 中，只需要在 php.ini 中启用
    if [[ " ${MISSING_EXTENSIONS[@]} " =~ " fileinfo " ]]; then
        # fileinfo 在 PHP 5.3+ 已内置，无需配置
        echo -e "${GREEN}  ✓ fileinfo 已内置（PHP 5.3+）${NC}"
        MISSING_EXTENSIONS=("${MISSING_EXTENSIONS[@]/fileinfo}")
    fi
    
    # 重新检测（等待一下让系统更新）
    sleep 1
    echo -e "${YELLOW}重新检测扩展...${NC}"
    STILL_MISSING=()
    for ext in "${MISSING_EXTENSIONS[@]}"; do
        if check_php_extension "$ext"; then
            echo -e "${GREEN}  ✓ $ext 已可用${NC}"
        else
            STILL_MISSING+=("$ext")
            echo -e "${YELLOW}  ✗ $ext 仍未检测到${NC}"
        fi
    done
    
    if [ ${#STILL_MISSING[@]} -gt 0 ]; then
        echo -e "${YELLOW}⚠ 以下扩展可能需要手动检查: ${STILL_MISSING[*]}${NC}"
        echo -e "${YELLOW}提示:${NC}"
        echo -e "${YELLOW}  1. 运行 'php -m' 查看所有已加载的扩展${NC}"
        echo -e "${YELLOW}  2. 检查 PHP 配置文件: $PHP_INI${NC}"
        if [[ "$PHP_INI" == *"/www/server/php"* ]]; then
            echo -e "${YELLOW}  3. 如果使用宝塔面板，请在面板中检查扩展状态${NC}"
            PHP_VERSION_DIR=$(echo "$PHP_INI" | grep -oP '/www/server/php/\K[0-9]+')
            echo -e "${YELLOW}  4. 扩展目录: /www/server/php/${PHP_VERSION_DIR}/lib/php/extensions/${NC}"
        fi
        echo -e "${YELLOW}  5. 某些扩展（如 opcache, gmp）可能已安装但需要重启 PHP-FPM${NC}"
        
        # 尝试重启 PHP-FPM
        if [[ "$PHP_INI" == *"/www/server/php"* ]]; then
            PHP_VERSION_DIR=$(echo "$PHP_INI" | grep -oP '/www/server/php/\K[0-9]+')
            FPM_SERVICE="php-fpm-${PHP_VERSION_DIR}"
            if systemctl list-units --type=service | grep -q "$FPM_SERVICE"; then
                echo -e "${YELLOW}  正在重启 PHP-FPM 服务...${NC}"
                systemctl restart "$FPM_SERVICE" 2>/dev/null || /etc/init.d/php-fpm-${PHP_VERSION_DIR} restart 2>/dev/null || true
            fi
        fi
        
        # 最后再检测一次
        sleep 1
        FINAL_MISSING=()
        for ext in "${STILL_MISSING[@]}"; do
            if ! check_php_extension "$ext"; then
                FINAL_MISSING+=("$ext")
            fi
        done
        
        if [ ${#FINAL_MISSING[@]} -eq 0 ]; then
            echo -e "${GREEN}✓ 所有扩展在重启后已可用${NC}"
            return 0
        else
            return 1
        fi
    else
        echo -e "${GREEN}✓ 所有必需的 PHP 扩展已成功安装${NC}"
        return 0
    fi
}

# 安装基础工具
install_basic_tools() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}步骤 1: 安装基础工具${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    if [[ "$OS" == "debian" || "$OS" == "ubuntu" ]]; then
        # 清理错误的 PPA 仓库配置
        echo -e "${YELLOW}清理错误的仓库配置...${NC}"
        rm -f /etc/apt/sources.list.d/deadsnakes-ubuntu-ppa-*.list 2>/dev/null || true
        rm -f /etc/apt/sources.list.d/*questing*.list 2>/dev/null || true
        
        apt update && apt upgrade -y
        apt install -y curl wget git unzip software-properties-common \
            apt-transport-https ca-certificates gnupg2 lsb-release
        timedatectl set-timezone Asia/Shanghai
    elif [[ "$OS" == "centos" || "$OS" == "rhel" || "$OS" == "rocky" || "$OS" == "almalinux" ]]; then
        dnf update -y
        dnf install -y epel-release
        dnf install -y curl wget git unzip tar
        timedatectl set-timezone Asia/Shanghai
    fi
}

# 安装 Nginx
install_nginx() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}步骤 2: 安装 Nginx${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    # 检测 Nginx 是否已安装
    if check_installed nginx; then
        echo -e "${GREEN}✓ Nginx 已安装，版本: $(nginx -v 2>&1 | cut -d '/' -f 2)${NC}"
        if check_service_running nginx; then
            echo -e "${GREEN}✓ Nginx 服务正在运行${NC}"
        else
            echo -e "${YELLOW}启动 Nginx 服务...${NC}"
            systemctl start nginx && systemctl enable nginx
        fi
        return 0
    fi
    
    echo -e "${YELLOW}正在安装 Nginx...${NC}"
    
    if [[ "$OS" == "debian" ]]; then
        curl https://nginx.org/keys/nginx_signing.key | gpg --dearmor \
            | tee /usr/share/keyrings/nginx-archive-keyring.gpg >/dev/null
        echo "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] \
            http://nginx.org/packages/mainline/debian bookworm nginx" \
            | tee /etc/apt/sources.list.d/nginx.list
        echo -e "Package: *\nPin: origin nginx.org\nPin: release o=nginx\nPin-Priority: 900\n" \
            | tee /etc/apt/preferences.d/99nginx
        apt update && apt install -y nginx
        if [ -f /etc/nginx/nginx.conf ]; then
            sed -i 's/^user.*/user www-data;/' /etc/nginx/nginx.conf
        fi
    elif [[ "$OS" == "ubuntu" ]]; then
        # 清理错误的 PPA 仓库
        echo -e "${YELLOW}清理错误的 PPA 仓库...${NC}"
        rm -f /etc/apt/sources.list.d/deadsnakes-ubuntu-ppa-*.list 2>/dev/null || true
        rm -f /etc/apt/sources.list.d/*questing*.list 2>/dev/null || true
        
        # 添加 Nginx 仓库
        curl https://nginx.org/keys/nginx_signing.key | gpg --dearmor \
            | tee /usr/share/keyrings/nginx-archive-keyring.gpg >/dev/null
        UBUNTU_CODENAME=$(lsb_release -cs)
        echo "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] \
            http://nginx.org/packages/mainline/ubuntu ${UBUNTU_CODENAME} nginx" \
            | tee /etc/apt/sources.list.d/nginx.list
        echo -e "Package: *\nPin: origin nginx.org\nPin: release o=nginx\nPin-Priority: 900\n" \
            | tee /etc/apt/preferences.d/99nginx
        
        # 更新并安装（忽略 PPA 错误）
        apt update 2>&1 | grep -v "questing" || true
        apt install -y nginx
        
        if [ -f /etc/nginx/nginx.conf ]; then
            sed -i 's/^user.*/user www-data;/' /etc/nginx/nginx.conf
        fi
    elif [[ "$OS" == "centos" || "$OS" == "rhel" || "$OS" == "rocky" || "$OS" == "almalinux" ]]; then
        cat > /etc/yum.repos.d/nginx.repo <<EOF
[nginx-mainline]
name=nginx mainline repo
baseurl=http://nginx.org/packages/mainline/rhel/\$releasever/\$basearch/
gpgcheck=1
enabled=1
gpgkey=https://nginx.org/keys/nginx_signing.key
module_hotfixes=true
priority=9
EOF
        dnf install -y nginx
    fi
    
    # 检查 Nginx 是否安装成功
    if command -v nginx &> /dev/null; then
        systemctl start nginx && systemctl enable nginx
        echo -e "${GREEN}Nginx 安装完成${NC}"
    else
        echo -e "${RED}Nginx 安装失败，尝试使用系统仓库安装...${NC}"
        if [[ "$OS" == "ubuntu" || "$OS" == "debian" ]]; then
            apt install -y nginx
            systemctl start nginx && systemctl enable nginx
            echo -e "${GREEN}Nginx 安装完成（使用系统仓库）${NC}"
        else
            echo -e "${RED}Nginx 安装失败，请手动安装${NC}"
            exit 1
        fi
    fi
}

# 安装 PHP
install_php() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}步骤 3: 安装 PHP 8.2+${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    # 检测 PHP 是否已安装且版本符合要求
    if check_php_version; then
        PHP_VER=$(php -v | head -n 1 | cut -d " " -f 2 | cut -d "." -f 1,2)
        echo -e "${GREEN}✓ PHP 已安装，版本: $PHP_VER${NC}"
        
        # 检测并安装缺失的 PHP 扩展
        install_php_extensions
        
        # 检测 PHP-FPM 是否运行
        if [[ "$OS" == "debian" || "$OS" == "ubuntu" ]]; then
            PHP_MAJOR=$(echo $PHP_VER | cut -d "." -f 1)
            PHP_MINOR=$(echo $PHP_VER | cut -d "." -f 2)
            FPM_SERVICE="php${PHP_MAJOR}.${PHP_MINOR}-fpm"
            if systemctl list-units --type=service | grep -q "$FPM_SERVICE"; then
                if check_service_running "$FPM_SERVICE"; then
                    echo -e "${GREEN}✓ PHP-FPM 服务正在运行${NC}"
                else
                    echo -e "${YELLOW}启动 PHP-FPM 服务...${NC}"
                    systemctl start "$FPM_SERVICE" && systemctl enable "$FPM_SERVICE"
                fi
            fi
        else
            if check_service_running php-fpm; then
                echo -e "${GREEN}✓ PHP-FPM 服务正在运行${NC}"
            else
                echo -e "${YELLOW}启动 PHP-FPM 服务...${NC}"
                systemctl start php-fpm && systemctl enable php-fpm
            fi
        fi
        return 0
    elif check_installed php; then
        PHP_VER=$(php -v | head -n 1 | cut -d " " -f 2 | cut -d "." -f 1,2)
        echo -e "${YELLOW}⚠ PHP 已安装但版本过低: $PHP_VER，需要 8.2+${NC}"
        read -p "是否继续安装 PHP 8.2+? (y/N): " UPGRADE_PHP
        if [[ "$UPGRADE_PHP" != "y" && "$UPGRADE_PHP" != "Y" ]]; then
            echo -e "${YELLOW}跳过 PHP 安装${NC}"
            return 0
        fi
    fi
    
    echo -e "${YELLOW}正在安装 PHP 8.2+...${NC}"
    PHP_VERSION="8.2"
    
    if [[ "$OS" == "debian" ]]; then
        curl -sSLo /tmp/php.gpg https://packages.sury.org/php/apt.gpg
        gpg --dearmor < /tmp/php.gpg > /usr/share/keyrings/php-archive-keyring.gpg
        echo "deb [signed-by=/usr/share/keyrings/php-archive-keyring.gpg] \
            https://packages.sury.org/php/ bookworm main" > /etc/apt/sources.list.d/php.list
        apt update
        apt install -y php${PHP_VERSION}-{bcmath,bz2,cli,common,curl,fpm,gd,gmp,igbinary,intl,mbstring,mysql,opcache,readline,redis,soap,xml,yaml,zip}
        apt install -y php${PHP_VERSION}-posix php${PHP_VERSION}-sodium || true
    elif [[ "$OS" == "ubuntu" ]]; then
        # 清理错误的 PPA 仓库
        rm -f /etc/apt/sources.list.d/deadsnakes-ubuntu-ppa-*.list 2>/dev/null || true
        rm -f /etc/apt/sources.list.d/*questing*.list 2>/dev/null || true
        
        # 添加 PHP PPA
        add-apt-repository ppa:ondrej/php -y
        
        # 更新（忽略错误）
        apt update 2>&1 | grep -v "questing" || true
        
        # 安装 PHP
        apt install -y php${PHP_VERSION}-{bcmath,bz2,cli,common,curl,fpm,gd,gmp,intl,mbstring,mysql,opcache,readline,redis,soap,xml,yaml,zip}
        apt install -y php${PHP_VERSION}-posix php${PHP_VERSION}-sodium || true
    elif [[ "$OS" == "centos" || "$OS" == "rhel" || "$OS" == "rocky" || "$OS" == "almalinux" ]]; then
        dnf config-manager --set-enabled crb 2>/dev/null || dnf config-manager --set-enabled powertools 2>/dev/null || true
        dnf install -y epel-release
        dnf install -y https://rpms.remirepo.net/enterprise/remi-release-9.rpm 2>/dev/null || \
            dnf install -y https://rpms.remirepo.net/enterprise/remi-release-8.rpm
        dnf module reset php -y
        dnf module install php:remi-${PHP_VERSION} -y
        dnf install -y php-{bcmath,cli,common,fpm,gd,gmp,intl,json,mbstring,mysqlnd,opcache,pdo,pecl-redis5,pecl-yaml,process,soap,sodium,xml,zip}
    fi
    
    # 配置 PHP
    PHP_INI=$(php --ini | grep "Loaded Configuration File" | awk '{print $4}')
    if [ -f "$PHP_INI" ]; then
        sed -i 's/^max_execution_time.*/max_execution_time = 300/' "$PHP_INI"
        sed -i 's/^memory_limit.*/memory_limit = 256M/' "$PHP_INI"
        sed -i 's/^post_max_size.*/post_max_size = 50M/' "$PHP_INI"
        sed -i 's/^upload_max_filesize.*/upload_max_filesize = 50M/' "$PHP_INI"
        sed -i 's/^;date.timezone.*/date.timezone = Asia\/Shanghai/' "$PHP_INI" || \
            echo "date.timezone = Asia/Shanghai" >> "$PHP_INI"
    fi
    
    # 配置 PHP-FPM
    if [[ "$OS" == "debian" || "$OS" == "ubuntu" ]]; then
        FPM_CONF="/etc/php/${PHP_VERSION}/fpm/pool.d/www.conf"
        sed -i 's/^;listen.owner.*/listen.owner = www-data/' "$FPM_CONF"
        sed -i 's/^;listen.group.*/listen.group = www-data/' "$FPM_CONF"
        sed -i 's/^;listen.mode.*/listen.mode = 0660/' "$FPM_CONF"
        systemctl restart php${PHP_VERSION}-fpm && systemctl enable php${PHP_VERSION}-fpm
    else
        FPM_CONF="/etc/php-fpm.d/www.conf"
        sed -i 's/^user = apache/user = nginx/' "$FPM_CONF"
        sed -i 's/^group = apache/group = nginx/' "$FPM_CONF"
        sed -i 's/^listen.owner = nobody/listen.owner = nginx/' "$FPM_CONF"
        sed -i 's/^listen.group = nobody/listen.group = nginx/' "$FPM_CONF"
        systemctl start php-fpm && systemctl enable php-fpm
    fi
    
    echo -e "${GREEN}PHP ${PHP_VERSION} 安装完成${NC}"
    
    # 安装完成后，再次检查扩展
    install_php_extensions
}

# 安装 MariaDB
install_mariadb() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}步骤 4: 安装 MariaDB${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    # 检测 MariaDB/MySQL 是否已安装
    if check_installed mariadb || check_installed mysql; then
        if check_installed mariadb; then
            echo -e "${GREEN}✓ MariaDB 已安装，版本: $(mariadb --version | cut -d " " -f 3 | cut -d "," -f 1)${NC}"
        else
            echo -e "${GREEN}✓ MySQL 已安装，版本: $(mysql --version | cut -d " " -f 5 | cut -d "," -f 1)${NC}"
        fi
        
        if check_service_running mariadb || check_service_running mysql; then
            echo -e "${GREEN}✓ 数据库服务正在运行${NC}"
            echo -e "${YELLOW}请手动创建数据库和用户，或使用现有数据库${NC}"
            read -p "请输入数据库密码（如果已设置）: " EXISTING_DB_PASSWORD
            if [ ! -z "$EXISTING_DB_PASSWORD" ]; then
                DB_PASSWORD="$EXISTING_DB_PASSWORD"
                echo "$DB_PASSWORD" > /tmp/sspanel_db_password.txt
                chmod 600 /tmp/sspanel_db_password.txt
            else
                read -p "请输入新数据库密码: " DB_PASSWORD
                echo "$DB_PASSWORD" > /tmp/sspanel_db_password.txt
                chmod 600 /tmp/sspanel_db_password.txt
            fi
            return 0
        else
            echo -e "${YELLOW}启动数据库服务...${NC}"
            systemctl start mariadb 2>/dev/null || systemctl start mysql
            systemctl enable mariadb 2>/dev/null || systemctl enable mysql
        fi
    fi
    
    echo -e "${YELLOW}正在安装 MariaDB...${NC}"
    
    if [[ "$OS" == "debian" ]]; then
        mkdir -p /etc/apt/keyrings
        curl -o /etc/apt/keyrings/mariadb-keyring.pgp \
            'https://mariadb.org/mariadb_release_signing_key.pgp'
        cat > /etc/apt/sources.list.d/mariadb.sources <<EOF
X-RepoLib-Name: MariaDB
Types: deb
URIs: https://deb.mariadb.org/11.8/debian
Suites: bookworm
Components: main
Signed-By: /etc/apt/keyrings/mariadb-keyring.pgp
EOF
        apt update && apt install -y mariadb-server mariadb-client
    elif [[ "$OS" == "ubuntu" ]]; then
        curl -o /tmp/mariadb_repo_setup.sh \
            https://downloads.mariadb.com/MariaDB/mariadb_repo_setup
        bash /tmp/mariadb_repo_setup.sh --mariadb-server-version="mariadb-11.8"
        apt update && apt install -y mariadb-server mariadb-client
    elif [[ "$OS" == "centos" || "$OS" == "rhel" || "$OS" == "rocky" || "$OS" == "almalinux" ]]; then
        cat > /etc/yum.repos.d/MariaDB.repo <<EOF
[mariadb]
name = MariaDB
baseurl = https://rpm.mariadb.org/11.8/rhel/\$releasever/\$basearch
module_hotfixes = 1
gpgkey = https://rpm.mariadb.org/RPM-GPG-KEY-MariaDB
gpgcheck = 1
EOF
        dnf install -y MariaDB-server MariaDB-client
    fi
    
    systemctl start mariadb && systemctl enable mariadb
    
    # 检查数据库是否已存在
    DB_EXISTS=$(mariadb -u root -e "SHOW DATABASES LIKE 'sspanel';" 2>/dev/null | grep -c sspanel || mysql -u root -e "SHOW DATABASES LIKE 'sspanel';" 2>/dev/null | grep -c sspanel || echo "0")
    
    if [ "$DB_EXISTS" -gt 0 ]; then
        echo -e "${GREEN}✓ 数据库 sspanel 已存在${NC}"
        read -p "请输入数据库密码（如果已设置）: " EXISTING_DB_PASSWORD
        if [ ! -z "$EXISTING_DB_PASSWORD" ]; then
            DB_PASSWORD="$EXISTING_DB_PASSWORD"
        else
            # 生成新密码
            DB_PASSWORD=$(openssl rand -base64 16 | tr -d "=+/" | cut -c1-16)
            echo -e "${YELLOW}数据库密码已生成: ${DB_PASSWORD}${NC}"
        fi
    else
        # 生成数据库密码
        DB_PASSWORD=$(openssl rand -base64 16 | tr -d "=+/" | cut -c1-16)
        echo -e "${YELLOW}数据库密码已生成: ${DB_PASSWORD}${NC}"
        echo -e "${YELLOW}请保存此密码，稍后需要配置到 .config.php${NC}"
        
        # 创建数据库和用户（如果不存在）
        if command -v mariadb &> /dev/null; then
            mariadb -u root <<EOF 2>/dev/null
CREATE DATABASE IF NOT EXISTS sspanel CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS 'sspanel'@'localhost' IDENTIFIED BY '${DB_PASSWORD}';
GRANT ALL PRIVILEGES ON sspanel.* TO 'sspanel'@'localhost';
FLUSH PRIVILEGES;
EOF
        else
            mysql -u root <<EOF 2>/dev/null
CREATE DATABASE IF NOT EXISTS sspanel CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS 'sspanel'@'localhost' IDENTIFIED BY '${DB_PASSWORD}';
GRANT ALL PRIVILEGES ON sspanel.* TO 'sspanel'@'localhost';
FLUSH PRIVILEGES;
EOF
        fi
    fi
    
    echo -e "${GREEN}MariaDB 安装完成${NC}"
    echo -e "${YELLOW}数据库名: sspanel${NC}"
    echo -e "${YELLOW}数据库用户: sspanel${NC}"
    echo -e "${YELLOW}数据库密码: ${DB_PASSWORD}${NC}"
    
    # 保存密码到文件（临时）
    echo "$DB_PASSWORD" > /tmp/sspanel_db_password.txt
    chmod 600 /tmp/sspanel_db_password.txt
}

# 安装 Redis
install_redis() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}步骤 5: 安装 Redis${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    # 检测 Redis 是否已安装
    if check_installed redis-server || check_installed redis-cli; then
        echo -e "${GREEN}✓ Redis 已安装${NC}"
        if check_service_running redis-server || check_service_running redis; then
            echo -e "${GREEN}✓ Redis 服务正在运行${NC}"
        else
            echo -e "${YELLOW}启动 Redis 服务...${NC}"
            systemctl start redis-server 2>/dev/null || systemctl start redis
            systemctl enable redis-server 2>/dev/null || systemctl enable redis
        fi
        return 0
    fi
    
    echo -e "${YELLOW}正在安装 Redis...${NC}"
    
    if [[ "$OS" == "debian" ]]; then
        curl -fsSL https://packages.redis.io/gpg | gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg
        echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] \
            https://packages.redis.io/deb bookworm main" | tee /etc/apt/sources.list.d/redis.list
        apt update && apt install -y redis
    elif [[ "$OS" == "ubuntu" ]]; then
        curl -fsSL https://packages.redis.io/gpg | gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg
        echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] \
            https://packages.redis.io/deb $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/redis.list
        apt update && apt install -y redis
    elif [[ "$OS" == "centos" || "$OS" == "rhel" || "$OS" == "rocky" || "$OS" == "almalinux" ]]; then
        dnf install -y redis
    fi
    
    # 配置 Redis
    REDIS_CONF="/etc/redis/redis.conf"
    if [ -f "$REDIS_CONF" ]; then
        sed -i 's/^# bind 127.0.0.1 ::1/bind 127.0.0.1 ::1/' "$REDIS_CONF" || \
            sed -i 's/^bind 127.0.0.1/bind 127.0.0.1/' "$REDIS_CONF"
        grep -q "^maxmemory" "$REDIS_CONF" || echo "maxmemory 256mb" >> "$REDIS_CONF"
        grep -q "^maxmemory-policy" "$REDIS_CONF" || echo "maxmemory-policy allkeys-lru" >> "$REDIS_CONF"
    fi
    
    if [[ "$OS" == "debian" || "$OS" == "ubuntu" ]]; then
        systemctl restart redis-server && systemctl enable redis-server
    else
        systemctl start redis && systemctl enable redis
    fi
    
    echo -e "${GREEN}Redis 安装完成${NC}"
}

# 安装 Composer
install_composer() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}步骤 6: 安装 Composer${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    # 检测 Composer 是否已安装
    if check_installed composer; then
        echo -e "${GREEN}✓ Composer 已安装，版本: $(composer --version | cut -d " " -f 3)${NC}"
        return 0
    fi
    
    echo -e "${YELLOW}正在安装 Composer...${NC}"
    
    if [ ! -f /usr/local/bin/composer ]; then
        curl -sS https://getcomposer.org/installer -o /tmp/composer-setup.php
        HASH=$(curl -sS https://composer.github.io/installer.sig)
        php -r "if (hash_file('SHA384', '/tmp/composer-setup.php') === '$HASH') { echo 'Installer verified'; } else { echo 'Installer corrupt'; unlink('/tmp/composer-setup.php'); } echo PHP_EOL;"
        php /tmp/composer-setup.php --install-dir=/usr/local/bin --filename=composer
        rm /tmp/composer-setup.php
    fi
    
    echo -e "${GREEN}Composer 安装完成${NC}"
}

# 部署项目
deploy_project() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}步骤 7: 部署 SSPanel-UIM${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    # 获取域名
    read -p "请输入您的域名 (例如: example.com): " DOMAIN
    if [ -z "$DOMAIN" ]; then
        echo -e "${RED}错误: 域名不能为空${NC}"
        exit 1
    fi
    
    # 创建项目目录
    mkdir -p /var/www/sspanel
    cd /var/www
    
    # 检查是否已存在项目
    if [ -d "sspanel" ] && [ "$(ls -A sspanel)" ]; then
        read -p "项目目录已存在，是否删除并重新克隆? (y/N): " RECLONE
        if [[ "$RECLONE" == "y" || "$RECLONE" == "Y" ]]; then
            rm -rf sspanel
        else
            echo -e "${YELLOW}使用现有项目目录${NC}"
            cd sspanel
            git pull origin main || true
        fi
    fi
    
    if [ ! -d "sspanel" ] || [ ! "$(ls -A sspanel)" ]; then
        echo -e "${YELLOW}正在从 GitHub 克隆项目...${NC}"
        git clone https://github.com/moneyfly1/myweb.git sspanel
        cd sspanel
        git config --global --add safe.directory /var/www/sspanel
    else
        cd sspanel
    fi
    
    # 安装依赖
    echo -e "${YELLOW}正在安装 Composer 依赖...${NC}"
    if php -v | grep -q "PHP 8.4"; then
        echo -e "${YELLOW}检测到 PHP 8.4，删除 composer.lock 以优化兼容性${NC}"
        rm -f composer.lock
    fi
    
    # 在安装 Composer 依赖之前，再次检查 PHP 扩展
    echo -e "${YELLOW}检查 PHP 扩展...${NC}"
    install_php_extensions
    
    composer install --no-dev --optimize-autoloader
    
    if [ ! -f vendor/autoload.php ]; then
        echo -e "${RED}错误: Composer 依赖安装失败${NC}"
        echo -e "${YELLOW}请检查 PHP 扩展是否已正确安装${NC}"
        echo -e "${YELLOW}运行 'php -m' 查看已安装的扩展${NC}"
        exit 1
    fi
    
    # 复制配置文件
    if [ ! -f config/.config.php ]; then
        cp config/.config.example.php config/.config.php
    fi
    if [ ! -f config/appprofile.php ]; then
        cp config/appprofile.example.php config/appprofile.php
    fi
    
    # 读取数据库密码
    if [ -f /tmp/sspanel_db_password.txt ]; then
        DB_PASSWORD=$(cat /tmp/sspanel_db_password.txt)
        rm -f /tmp/sspanel_db_password.txt
    else
        read -p "请输入数据库密码: " DB_PASSWORD
    fi
    
    # 生成随机密钥
    KEY=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)
    MU_KEY=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)
    
    # 配置 .config.php
    sed -i "s|ChangeMe|${KEY}|g" config/.config.php
    sed -i "s|https://example.com|https://${DOMAIN}|g" config/.config.php
    sed -i "s|'sspanel'|'sspanel'|g" config/.config.php
    sed -i "s|'root'|'sspanel'|g" config/.config.php
    sed -i "s|'sspanel'|'${DB_PASSWORD}'|g" config/.config.php
    
    # 更新 muKey
    if grep -q "muKey.*ChangeMe" config/.config.php; then
        sed -i "s|muKey.*ChangeMe|muKey.*${MU_KEY}|g" config/.config.php
    fi
    
    echo -e "${GREEN}项目部署完成${NC}"
    echo -e "${YELLOW}配置文件已更新:${NC}"
    echo -e "  - 域名: https://${DOMAIN}"
    echo -e "  - 数据库密码: ${DB_PASSWORD}"
    echo -e "  - Cookie 密钥: ${KEY}"
    echo -e "  - WebAPI 密钥: ${MU_KEY}"
}

# 设置权限
set_permissions() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}步骤 8: 设置文件权限${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    if [[ "$OS" == "debian" || "$OS" == "ubuntu" ]]; then
        WEB_USER="www-data"
    else
        WEB_USER="nginx"
    fi
    
    chown -R ${WEB_USER}:${WEB_USER} /var/www/sspanel
    find /var/www/sspanel -type d -exec chmod 755 {} \;
    find /var/www/sspanel -type f -exec chmod 644 {} \;
    chmod -R 777 /var/www/sspanel/storage
    chmod 775 /var/www/sspanel/public/clients
    
    mkdir -p /var/www/sspanel/storage/framework/smarty/{cache,compile}
    mkdir -p /var/www/sspanel/storage/framework/twig/cache
    chmod -R 777 /var/www/sspanel/storage/framework
    
    chmod 664 /var/www/sspanel/config/.config.php
    chmod 664 /var/www/sspanel/config/appprofile.php
    
    echo -e "${GREEN}权限设置完成${NC}"
}

# 配置 Nginx
configure_nginx() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}步骤 9: 配置 Nginx${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    if [ -z "$DOMAIN" ]; then
        read -p "请输入您的域名: " DOMAIN
    fi
    
    if [[ "$OS" == "debian" || "$OS" == "ubuntu" ]]; then
        PHP_SOCKET="/run/php/php8.2-fpm.sock"
        [ ! -f "$PHP_SOCKET" ] && PHP_SOCKET="/run/php/php8.3-fpm.sock"
        [ ! -f "$PHP_SOCKET" ] && PHP_SOCKET="/run/php/php8.4-fpm.sock"
        CONFIG_DIR="/etc/nginx/conf.d"
    else
        PHP_SOCKET="/run/php-fpm/www.sock"
        CONFIG_DIR="/etc/nginx/conf.d"
    fi
    
    cat > ${CONFIG_DIR}/sspanel.conf <<'EOF'
server {
    listen 80;
    listen [::]:80;
    server_name DOMAIN_PLACEHOLDER;
    
    root /var/www/sspanel/public;
    index index.php;
    
    location / {
        try_files $uri /index.php?$query_string;
    }
    
    location ~ \.php$ {
        fastcgi_pass unix:PHP_SOCKET_PLACEHOLDER;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
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
    # 替换占位符
    sed -i "s|DOMAIN_PLACEHOLDER|${DOMAIN}|g" ${CONFIG_DIR}/sspanel.conf
    sed -i "s|PHP_SOCKET_PLACEHOLDER|${PHP_SOCKET}|g" ${CONFIG_DIR}/sspanel.conf
    
    nginx -t && systemctl reload nginx
    echo -e "${GREEN}Nginx 配置完成${NC}"
}

# 初始化数据库
init_database() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}步骤 10: 初始化数据库${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    cd /var/www/sspanel
    
    if [ ! -f vendor/autoload.php ]; then
        echo -e "${RED}错误: vendor/autoload.php 不存在，请先运行 composer install${NC}"
        exit 1
    fi
    
    echo -e "${YELLOW}正在运行数据库迁移...${NC}"
    php xcat Migration new
    php xcat Migration latest
    
    echo -e "${YELLOW}正在导入配置项...${NC}"
    php xcat Tool importSetting
    
    echo -e "${YELLOW}正在创建管理员账户...${NC}"
    php xcat Tool createAdmin
    
    echo -e "${GREEN}数据库初始化完成${NC}"
}

# 配置定时任务
configure_cron() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}步骤 11: 配置定时任务${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    CRON_JOB="*/5 * * * * /usr/bin/php /var/www/sspanel/xcat Cron >> /var/log/sspanel-cron.log 2>&1"
    
    (crontab -l 2>/dev/null | grep -v "xcat Cron"; echo "$CRON_JOB") | crontab -
    
    echo -e "${GREEN}定时任务配置完成${NC}"
    echo -e "${YELLOW}定时任务: 每 5 分钟执行一次${NC}"
}

# 配置 SSL
configure_ssl() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}步骤 12: 配置 SSL 证书 (可选)${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    read -p "是否现在配置 SSL 证书? (Y/n): " CONFIGURE_SSL
    if [[ "$CONFIGURE_SSL" != "n" && "$CONFIGURE_SSL" != "N" ]]; then
        if [[ "$OS" == "debian" || "$OS" == "ubuntu" ]]; then
            apt install -y certbot python3-certbot-nginx
        else
            dnf install -y certbot python3-certbot-nginx
        fi
        
        certbot --nginx -d ${DOMAIN}
        systemctl enable certbot-renew.timer
        
        echo -e "${GREEN}SSL 证书配置完成${NC}"
    else
        echo -e "${YELLOW}跳过 SSL 配置，稍后可以运行: certbot --nginx -d ${DOMAIN}${NC}"
    fi
}

# 主函数
main() {
    echo -e "${GREEN}"
    echo "=========================================="
    echo "SSPanel-UIM 节点采集版 - 一键安装脚本"
    echo "=========================================="
    echo -e "${NC}"
    
    detect_system
    install_basic_tools
    install_nginx
    install_php
    install_mariadb
    install_redis
    install_composer
    deploy_project
    set_permissions
    configure_nginx
    init_database
    configure_cron
    configure_ssl
    
    echo -e "${GREEN}"
    echo "=========================================="
    echo "安装完成！"
    echo "=========================================="
    echo -e "${NC}"
    echo -e "访问地址: https://${DOMAIN}"
    echo -e "管理后台: https://${DOMAIN}/auth/login"
    echo -e "节点采集: https://${DOMAIN}/admin/node/collector"
    echo ""
    echo -e "${YELLOW}下一步操作:${NC}"
    echo "1. 登录管理后台"
    echo "2. 配置节点采集功能"
    echo "3. 添加采集源 URL"
    echo "4. 测试节点采集"
    echo ""
}

# 运行主函数
main
