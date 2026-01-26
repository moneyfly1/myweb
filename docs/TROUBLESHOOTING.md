# 安装问题排查指南

## 常见问题及解决方法

### 1. 脚本无法继续执行

**问题**：运行 `sudo bash install-vps.sh` 后脚本停止或报错

**排查步骤**：

```bash
# 1. 查看安装日志
tail -100 /tmp/cboard_install_*.log

# 2. 检查网络连接
ping -c 3 8.8.8.8
curl -I https://github.com

# 3. 检查磁盘空间
df -h

# 4. 检查系统资源
free -h

# 5. 检查操作系统版本
cat /etc/os-release
```

### 2. 网络连接问题

**问题**：无法下载 Go、Node.js 或 GitHub 代码

**解决方法**：

```bash
# 检查网络
ping -c 3 github.com
ping -c 3 go.dev

# 如果无法访问，可能需要配置代理
export http_proxy=http://your-proxy:port
export https_proxy=http://your-proxy:port

# 或使用镜像源（脚本会自动尝试）
```

### 3. Go 安装失败

**问题**：Go 下载或安装失败

**解决方法**：

```bash
# 手动安装 Go
cd /tmp
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
# 或使用国内镜像
wget https://golang.google.cn/dl/go1.21.5.linux-amd64.tar.gz

tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
```

### 4. Node.js 安装失败

**问题**：Node.js 安装失败

**解决方法**：

```bash
# Ubuntu/Debian
curl -fsSL https://deb.nodesource.com/setup_18.x | bash -
apt-get install -y nodejs

# CentOS
curl -fsSL https://rpm.nodesource.com/setup_18.x | bash -
yum install -y nodejs

# 或使用二进制安装
cd /tmp
wget https://nodejs.org/dist/v18.20.4/node-v18.20.4-linux-x64.tar.xz
tar -xf node-v18.20.4-linux-x64.tar.xz
mv node-v18.20.4-linux-x64 /usr/local/nodejs18
export PATH=$PATH:/usr/local/nodejs18/bin
```

### 5. Nginx 安装失败

**问题**：Nginx 无法安装或启动

**解决方法**：

```bash
# Ubuntu/Debian
apt-get update
apt-get install -y nginx

# CentOS
yum install -y nginx
systemctl enable nginx
systemctl start nginx

# 检查 Nginx 状态
systemctl status nginx
nginx -t
```

### 6. 代码下载失败

**问题**：无法从 GitHub 下载代码

**解决方法**：

```bash
# 方法1：使用 GitHub 镜像
git clone https://ghproxy.com/https://github.com/moneyfly1/myweb.git /opt/cboard

# 方法2：手动下载
cd /opt
wget https://github.com/moneyfly1/myweb/archive/refs/heads/main.zip
unzip main.zip
mv myweb-main cboard
```

### 7. 编译失败

**问题**：Go 或前端编译失败

**解决方法**：

```bash
# 检查 Go 环境
go version
go env

# 检查 Go 代理
go env GOPROXY
# 如果不可用，设置代理
go env -w GOPROXY=https://goproxy.cn,direct

# 清理并重新编译
cd /opt/cboard
go clean -modcache
go mod download
go build -o server ./cmd/server/main.go

# 前端编译
cd frontend
rm -rf node_modules
npm cache clean --force
npm install --legacy-peer-deps
npm run build
```

### 8. 服务启动失败

**问题**：systemd 服务无法启动

**解决方法**：

```bash
# 查看服务状态
systemctl status cboard

# 查看服务日志
journalctl -u cboard -n 50

# 查看应用日志
tail -f /opt/cboard/server.log

# 检查端口占用
netstat -tlnp | grep 8000
# 如果被占用，停止占用进程
lsof -ti:8000 | xargs kill -9

# 手动测试运行
cd /opt/cboard
./server
```

### 9. SSL 证书申请失败

**问题**：Let's Encrypt 证书申请失败

**解决方法**：

```bash
# 检查域名解析
nslookup yourdomain.com
dig yourdomain.com

# 检查端口 80 是否开放
netstat -tlnp | grep :80

# 手动申请证书
certbot --nginx -d yourdomain.com --email your-email@example.com

# 如果自动申请失败，可以稍后手动申请
```

### 10. 管理员账户创建失败

**问题**：无法创建管理员账户

**解决方法**：

```bash
cd /opt/cboard
export ADMIN_USERNAME="admin"
export ADMIN_EMAIL="your-email@example.com"
export ADMIN_PASSWORD="your-password"
go run scripts/create_admin.go
```

### 11. 权限问题

**问题**：文件权限错误

**解决方法**：

```bash
# 确保文件权限正确
chmod +x /opt/cboard/server
chmod 666 /opt/cboard/cboard.db
chmod -R 755 /opt/cboard/uploads

# 确保目录存在
mkdir -p /opt/cboard/uploads/logs
mkdir -p /opt/cboard/uploads/files
```

### 12. 防火墙问题

**问题**：无法访问网站

**解决方法**：

```bash
# 检查防火墙状态
systemctl status firewalld
ufw status

# 开放端口（firewalld）
firewall-cmd --permanent --add-service=http
firewall-cmd --permanent --add-service=https
firewall-cmd --reload

# 开放端口（ufw）
ufw allow 80/tcp
ufw allow 443/tcp

# 开放端口（iptables）
iptables -I INPUT -p tcp --dport 80 -j ACCEPT
iptables -I INPUT -p tcp --dport 443 -j ACCEPT
```

## 获取帮助

如果以上方法都无法解决问题，请提供以下信息：

1. **操作系统信息**：
   ```bash
   cat /etc/os-release
   uname -a
   ```

2. **错误日志**：
   ```bash
   tail -100 /tmp/cboard_install_*.log
   journalctl -u cboard -n 50
   ```

3. **系统资源**：
   ```bash
   df -h
   free -h
   ```

4. **网络状态**：
   ```bash
   ping -c 3 github.com
   curl -I https://github.com
   ```

5. **服务状态**：
   ```bash
   systemctl status cboard
   systemctl status nginx
   ```

## 重新安装

如果问题无法解决，可以尝试重新安装：

```bash
# 1. 停止服务
systemctl stop cboard

# 2. 删除项目目录（可选）
rm -rf /opt/cboard

# 3. 重新运行安装脚本
sudo bash install-vps.sh
```

