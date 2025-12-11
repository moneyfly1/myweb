# VPS 一键安装命令

## 方法 1: 使用一键安装脚本（推荐）

在 VPS 上执行以下命令：

```bash
# 下载并运行一键安装脚本
bash <(curl -sSL https://raw.githubusercontent.com/moneyfly1/myweb/main/一键安装.sh)
```

或者：

```bash
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/一键安装.sh
chmod +x 一键安装.sh
./一键安装.sh
```

## 方法 2: 直接运行完整安装脚本

```bash
# 下载安装脚本
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
chmod +x install.sh

# 运行安装脚本
./install.sh
```

安装过程中会提示：
1. 选择安装目录（默认：/www/wwwroot/board.moneyfly.club）
2. 输入域名（例如：board.moneyfly.club）

## 方法 3: 指定安装目录（宝塔面板）

如果您想直接安装到 `/www/wwwroot/board.moneyfly.club`：

```bash
# 下载安装脚本
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
chmod +x install.sh

# 设置环境变量并运行
export INSTALL_DIR="/www/wwwroot/board.moneyfly.club"
export DOMAIN="board.moneyfly.club"
./install.sh
```

## 安装后操作

### 1. 创建管理员账号

```bash
cd /www/wwwroot/board.moneyfly.club
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/create_admin.sh
chmod +x create_admin.sh
./create_admin.sh
```

### 2. 如果遇到问题，运行修复脚本

```bash
# 修复后台访问
cd /www/wwwroot/board.moneyfly.club
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/fix_backend_access.sh
chmod +x fix_backend_access.sh
./fix_backend_access.sh

# 检查数据库
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/check_database.sh
chmod +x check_database.sh
./check_database.sh
```

## 完整安装流程

```bash
# 1. 下载并运行一键安装
bash <(curl -sSL https://raw.githubusercontent.com/moneyfly1/myweb/main/一键安装.sh)

# 2. 等待安装完成（会自动完成所有步骤）

# 3. 创建管理员账号
cd /www/wwwroot/board.moneyfly.club
wget https://raw.githubusercontent.com/moneyfly1/myweb/main/create_admin.sh
chmod +x create_admin.sh
./create_admin.sh

# 4. 访问后台
# http://board.moneyfly.club/auth/login
```

## 安装目录说明

- **宝塔面板**: `/www/wwwroot/board.moneyfly.club`
- **标准安装**: `/var/www/sspanel`

脚本会自动检测并适配不同的安装环境。

## 注意事项

1. **需要 root 权限**: 确保使用 root 用户运行
2. **域名解析**: 确保域名已正确解析到 VPS IP
3. **防火墙**: 确保 80 和 443 端口已开放
4. **数据库密码**: 安装过程中会生成随机密码，请保存

## 如果安装失败

1. **查看错误日志**: 检查脚本输出的错误信息
2. **运行修复脚本**: 根据错误类型运行相应的修复脚本
3. **检查系统要求**: 
   - PHP 8.2+
   - Nginx
   - MariaDB/MySQL
   - Redis
   - Composer 2.1+

---

**提示**: 如果使用宝塔面板，建议在面板中先安装 PHP、Nginx、MySQL，然后运行安装脚本。
