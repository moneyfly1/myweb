# 无宝塔 VPS 部署教程（使用 install-vps.sh）

适用于**新 VPS**，已绑定域名（如 `board.moneyfly.top`），**未安装宝塔面板**，通过 GitHub 代码 + 一键脚本完成部署。

---

## 一、前置准备

1. **VPS**：Ubuntu 18.04+ / Debian 10+ / CentOS 7+，建议 1GB 内存以上。
2. **域名**：已购买并解析到本机：
   - 在域名服务商处添加 **A 记录**：`board.moneyfly.top` → 你的 VPS 公网 IP。
   - 等待 DNS 生效（可用 `ping board.moneyfly.top` 检查）。
3. **SSH**：能使用 root 或 sudo 登录 VPS。

---

## 二、部署方式（二选一）

### 方式 A：直接下载并运行安装脚本（推荐）

脚本会自动从 GitHub 拉取代码，无需先克隆仓库。

```bash
# 1. SSH 登录 VPS 后，下载安装脚本
curl -sL https://raw.githubusercontent.com/moneyfly1/myweb/main/install-vps.sh -o install-vps.sh

# 2. 使用 root 运行（非 root 请用 sudo）
sudo bash install-vps.sh
```

按提示输入：

- **域名**：`board.moneyfly.top`
- **项目安装目录**：直接回车即用默认 `/opt/cboard`
- **管理员用户名 / 邮箱 / 密码**：按需填写

确认后脚本会自动：安装 Nginx、Go、Node.js → 从 GitHub 克隆代码 → 构建前后端 → 配置 Nginx、systemd、可选 HTTPS（Certbot）→ 创建管理员并启动服务。

---

### 方式 B：先克隆仓库再运行脚本

适合希望先拿到完整代码、或当前环境拉取 GitHub 较慢时使用。

```bash
# 1. 安装 git（若未安装）
# Ubuntu/Debian:
sudo apt-get update && sudo apt-get install -y git

# CentOS:
# sudo yum install -y git

# 2. 克隆仓库到安装目录（默认 /opt/cboard）
sudo mkdir -p /opt
sudo git clone https://github.com/moneyfly1/myweb.git /opt/cboard
cd /opt/cboard

# 3. 运行安装脚本
sudo bash install-vps.sh
```

按提示输入：

- **域名**：`board.moneyfly.top`
- **项目安装目录**：输入 `/opt/cboard`（与上面克隆路径一致）
- 若提示「项目目录已存在，是否删除并重新下载？」选 **n**，使用当前已克隆的代码
- **管理员用户名 / 邮箱 / 密码**：按需填写

脚本会安装环境、构建、配置 Nginx/systemd、创建管理员并启动服务（不会重复克隆）。

---

## 三、安装过程中会做什么

脚本会依次执行（无需宝塔）：

1. 检查系统、网络
2. 安装 Nginx、Go、Node.js 等依赖
3. 从 GitHub 拉取代码（方式 A）或使用已有目录（方式 B）
4. 生成 `.env` 并构建后端 + 前端
5. 创建管理员账号
6. 配置 systemd 服务（`cboard.service`）
7. 配置 Nginx（反向代理 + 静态资源）
8. 可选：Certbot 申请 HTTPS（按提示选择）
9. 启动并设置开机自启

---

## 四、安装完成后

- **访问地址**：`https://board.moneyfly.top` 或 `http://board.moneyfly.top`（若未配置 HTTPS）
- **管理后台**：`https://board.moneyfly.top/admin/login`，使用安装时设置的管理员账号登录。

常用命令：

```bash
# 查看服务状态
sudo systemctl status cboard

# 查看实时日志
sudo journalctl -u cboard -f
# 或应用日志
sudo tail -f /opt/cboard/server.log

# 重启服务
sudo systemctl restart cboard
```

---

## 五、常见问题

| 情况 | 处理 |
|------|------|
| 脚本报错 | 查看安装日志：`tail -100 /tmp/cboard_install_*.log` |
| GitHub 克隆失败 | 脚本会自动尝试 GitHub 镜像；若仍失败，可用方式 B 在本地或代理环境克隆后上传到 VPS 再运行脚本 |
| 域名打不开 | 检查 DNS 是否解析到本机：`ping board.moneyfly.top`；检查 Nginx：`sudo nginx -t`、`sudo systemctl status nginx` |
| 想用其他域名 | 重新运行脚本时输入新域名，或手动改 Nginx 配置与 `.env` 中的域名后重载 Nginx、重启 cboard |

---

## 六、简要流程回顾

1. 域名 A 记录指向 VPS IP。
2. SSH 登录 VPS，执行 **方式 A** 或 **方式 B**。
3. 按提示输入域名 `board.moneyfly.top`、安装目录、管理员信息。
4. 安装结束后访问 `https://board.moneyfly.top` 与 `/admin/login`。

无需安装宝塔，一条脚本即可完成从 GitHub 到线上运行的部署。
