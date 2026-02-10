# install-vps.sh 脚本检查说明

本文档记录对 `install-vps.sh` 的检查结果与已做修复，便于后续维护。

---

## 一、已发现并修复的问题

### 1. Node.js 安装后未统一验证（已修复）

**问题**：当通过「NodeSource 失败 → 二进制安装」路径安装 Node 时，脚本在 `install_nodejs_binary` 后直接 `return 0`，不会执行后面的「验证安装」逻辑。若二进制安装失败，脚本会误认为安装成功并继续执行，导致后续前端构建失败。

**修复**：去掉两处 `install_nodejs_binary` 后的 `return 0`，让流程统一落到最后的「验证安装」块；只有 `command -v node` 通过才视为成功，否则 `exit 1`。

### 2. 未检查后端入口文件（已修复）

**问题**：脚本直接执行 `go build -o server ./cmd/server/main.go`。若仓库中未包含 `cmd/server/main.go`（例如未提交或仅在部分分支），编译会失败且报错不够明确。

**修复**：在 `build_project()` 开头增加检查：若 `cmd/server/main.go` 不存在则报错并退出，并提示「请确认仓库已包含主程序（cmd/server/main.go）」。

**重要**：请确保 GitHub 仓库中**已提交** `cmd/server/main.go`。若当前本地/仓库没有该文件，需要先补上主程序入口再使用本脚本。

### 3. CentOS 8+ 包管理器兼容（已修复）

**问题**：原脚本使用 `yum makecache fast`。在 CentOS 8 / RHEL 8 / Rocky 8+ 上，默认包管理器为 `dnf`，且部分系统上 `yum makecache fast` 不可用或行为不同。

**修复**：
- 包列表更新：若存在 `dnf` 则执行 `dnf makecache -q`，否则 `yum makecache -q`（去掉 `fast`，提高兼容性）。
- Node 安装：在 Red Hat 系中，若存在 `dnf` 则用 `dnf install -y nodejs`，否则用 `yum install -y nodejs`。

---

## 二、脚本逻辑与依赖检查（当前无问题）

| 项目 | 说明 |
|------|------|
| **执行顺序** | 基础检查 → 用户输入 → 系统依赖 → Go → Node → 下载代码 → .env → 构建 → 管理员 → systemd → Nginx → 启动，顺序正确。 |
| **.env 与管理员** | `create_env_file` 在 `create_admin_account` 之前执行，且 `go run scripts/admin_tool.go` 在项目目录下运行，能正确读取 `.env` 和 `DATABASE_URL`。 |
| **Nginx 代理** | `location /api/` 转发到 `BACKEND_PORT`，与后端 `router` 的 `/api/v1` 前缀一致；健康检查使用 `/health`，与路由一致。 |
| **systemd** | `WorkingDirectory`、`ExecStart` 使用 `PROJECT_DIR`，不依赖 Go 运行时，仅运行已编译的 `server` 二进制。 |
| **admin_tool 环境变量** | 脚本在运行前 export `ADMIN_USERNAME`、`ADMIN_EMAIL`、`ADMIN_PASSWORD`，与 `scripts/admin_tool.go` 的 `os.Getenv` 一致。 |
| **防火墙** | 对 firewalld、ufw、iptables 的检测与放行 80/443 逻辑合理，且失败不阻塞安装。 |

---

## 三、使用前请确认

1. **仓库中必须包含 `cmd/server/main.go`**  
   脚本会在构建前检查该文件；若不存在会直接报错。若你的仓库中主程序入口路径或文件名不同，需要同步修改脚本中的检查路径和 `go build` 命令。

2. **域名 DNS**  
   安装过程中会尝试用 Certbot 申请 HTTPS，域名需已解析到当前 VPS 的 IP。若解析未生效，可先跳过证书，安装完成后再执行：  
   `certbot --nginx -d board.moneyfly.top`

3. **内存与 Swap**  
   脚本在低内存时会自动创建 2GB Swap，可减轻 OOM；若编译仍失败，建议将 VPS 内存提高到至少 1GB。

4. **日志与排错**  
   安装过程会写入 `/tmp/cboard_install_*.log`。失败时请先执行：  
   `tail -100 /tmp/cboard_install_*.log`  
   再根据报错排查（如缺少 `cmd/server/main.go`、网络、权限等）。

---

## 四、小结

- 已修复：Node 安装后统一验证、构建前检查 `cmd/server/main.go`、CentOS 8+ 包管理器与 makecache 兼容。
- 其余流程（依赖顺序、Nginx、systemd、.env、管理员创建）检查通过，无需修改。
- 使用前请确保仓库中有 `cmd/server/main.go`，并按上述说明确认域名与资源。
