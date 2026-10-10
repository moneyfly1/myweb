# CBoard —— 自建订阅面板（Go + Vue 3）

一条命令装完：**纯 VPS 或宝塔面板都能装**，装完直接把登录地址和管理员密码打印给你（并实测登录一次）。
SQLite 零外部依赖，Redis / MySQL / PostgreSQL 都是可选。

> 只有两个脚本：**`install.sh`（唯一实现）**、**`bt-deploy.sh`（宝塔入口，行为完全等同）**。
> 详细教程见 [`docs/`](docs/README.md)：[纯 VPS 一键脚本](docs/部署/VPS部署教程-一键脚本.md)、[宝塔面板](docs/部署/宝塔部署教程.md)、[安装排错](docs/故障排查/安装问题排查指南.md)。

---

## 一、安装（三步）

### 方式 A：纯 VPS（推荐）

```bash
# ① 放脚本（目录名 = 你的域名，脚本就用目录名当域名）
mkdir -p /www/wwwroot/你的域名 && cd /www/wwwroot/你的域名
curl -fsSLO https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
curl -fsSLO https://raw.githubusercontent.com/moneyfly1/myweb/main/bt-deploy.sh

# ② 运行（需要 root）
sudo bash install.sh

# ③ 在菜单里输入 1，然后等它跑完
```

菜单 1 会自动完成全部事情：装依赖（基础包 / gcc / Go 1.25 / Node 22 / Nginx / certbot）→ 拉代码 →
写 `.env`（随机 `SECRET_KEY` + 管理员账号密码）→ 编译前后端 → 注册 systemd 服务 → 写 Nginx 配置并
签发 HTTPS 证书 → 打印登录信息。**重复执行是安全的**（会先备份数据库与旧二进制）。

装完屏幕上会给你这些（并已经用这组账号密码真实登录验证过）：

```text
================== 登录信息 ==================
  前台登录:     https://你的域名/login
  管理员登录:   https://你的域名/admin/login
  管理员账号:   admin
  管理员密码:   xxxxxxxxxxxxxxxx
==============================================
```

### 方式 B：宝塔面板

```bash
# ① 面板 → 软件商店 → 安装 Nginx（源码编译，2 核约 25–40 分钟，正常）
# ② 面板 → 网站 → 添加站点：域名填你的域名，根目录 /www/wwwroot/你的域名，PHP 选「纯静态」
#    ⚠️ 目录名必须等于域名
# ③ 进站点目录，清掉占位文件并放脚本
cd /www/wwwroot/你的域名
rm -f index.html 404.html .user.ini
curl -fsSLO https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
curl -fsSLO https://raw.githubusercontent.com/moneyfly1/myweb/main/bt-deploy.sh

# ④ 一键部署
bash bt-deploy.sh        # 菜单选 1
```

**证书二选一，不要混用**：

| 模式 | 怎么用 | 续期由谁负责 |
|---|---|---|
| 脚本管（默认） | 直接跑上面的步骤 | `certbot.timer` 自动续期；**面板里不要点「申请/续签」** |
| 面板管 | `CERT_MANAGER=panel bash bt-deploy.sh`，再去 面板 → 网站 → SSL → Let's Encrypt → 申请 | 面板自动续期；换域名或面板重写配置后跑一次**菜单 16** |

### 方式 C：Docker（可选）

```bash
cp .env.example .env        # 至少填 SECRET_KEY (openssl rand -hex 32) 和 ADMIN_PASSWORD
docker compose up -d --build
curl -fsS http://127.0.0.1:8000/health
```

- 已实测：后端编译、容器内运行、后台登录、重启后数据持久化都正常。
- **尚未端到端验证**：容器内的前端构建与完整 `docker compose up`（验证机网络拉不动 `node:22-alpine`）。
  所以对生产环境**优先用方式 A / B**。
- 运行时镜像**必须含 `tzdata`**（本仓库 Dockerfile 已装）；精简镜像要自己把 `/usr/share/zoneinfo` 拷进去，否则启动即报 `Invalid _loc: Asia/Shanghai`。

---

## 二、安装后要设置的（只有 3 件）

| 步骤 | 怎么做 |
|---|---|
| **1. 登录后台** | 用上面打印的地址与账号密码。想固定密码：改 `.env` 的 `ADMIN_PASSWORD` → 菜单 8 重启（应用每次启动都按它重置）；或直接跑**菜单 2** 重置 |
| **2. 域名与 HTTPS** | 脚本自动处理好了。以后换域名：把项目目录改名成新域名（或设 `DOMAIN`）后跑**菜单 16** 自动补齐配置 |
| **3. 可选增强** | 邮件 SMTP、Redis 缓存（**菜单 12**，会自动装 Redis，装不上自动换另一种装法）、支付、各类通知 |

> 其余功能——用户、套餐、订单、节点、订阅、工单、邀请码、优惠券、公告、主题、注册设置等，
> **全部在管理后台点一下就能配**，不用改服务器文件。

---

## 三、脚本菜单速查

| 键 | 作用 | 键 | 作用 |
|---|---|---|---|
| `1` | 一键全自动部署（首次安装用这个） | `9` | 停止服务 |
| `2` | 创建 / 重置管理员账号（建完实测登录） | `10` | 证书续期（手动） |
| `3` | 强制重启（只清本项目进程） | `11` | **从 GitHub 同步升级并重建** |
| `4` | 深度清理缓存（不删前端产物与二进制） | `12` | 配置 Redis 缓存（自动装 Redis） |
| `5` | 解锁被锁定的用户 | `13` | 回滚到升级前版本 |
| `6` | 查看服务状态 | `14` | 只重建 + 重启（不动 Nginx） |
| `7` | 查看实时日志 | `15` | 完全卸载（服务/配置/残留扫描） |
| `8` | 重启服务 | `16` | **自检并自动修复**（配置被面板重写后用） |
| | | `0` | 退出 |

---

## 四、常用设置（环境变量）

文件：`项目目录/.env`（改完执行**菜单 8** 生效）

| 变量 | 默认 | 说明 |
|---|---|---|
| `HOST` / `PORT` | `127.0.0.1` / `8000` | 后端监听地址；保持 `127.0.0.1`，由 Nginx 反代 |
| `DATABASE_URL` | `sqlite:////www/wwwroot/<域名>/cboard.db` | 固定**绝对路径**，不要改成相对路径 |
| `SECRET_KEY` | 随机 64 位 | 登录态签名密钥，泄露要立刻更换 |
| `ADMIN_USERNAME` / `ADMIN_EMAIL` / `ADMIN_PASSWORD` | `admin` / `admin@<域名>` / 随机 16 位 | 管理员账号；密码每次启动都会按 `ADMIN_PASSWORD` 重置 |
| `TRUSTED_PROXIES` | 空 | 放在 Nginx / CDN 后面时填 `127.0.0.1` |
| `PANEL_PUBLIC_URL` | 空 | 有自建节点回传时必须填 |
| `REDIS_ADDR` / `REDIS_PASSWORD` | 空 | 用菜单 12 自动写入 |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USERNAME` / `SMTP_PASSWORD` | 空 | 邮件通知（也可在后台配置） |
| `DISABLE_SCHEDULE_TASKS` | `false` | 设为 `true` 关闭定时任务 |
| `DB_BACKUP_DIR` / `DB_BACKUP_KEEP` | `/www/backup/cboard` / `10` | 备份目录与保留份数 |

---

## 五、升级 / 备份 / 回滚

| 操作 | 做法 |
|---|---|
| 升级到最新代码 | **菜单 11**（自动：备份数据库 → 构建前后端 → 原子替换 → 重启 + 健康检查） |
| 回滚 | **菜单 13**（用 `server.bak.*` 与数据库备份） |
| 自动备份位置 | `/www/backup/cboard/pre-upgrade-*.db.gz`（默认保留最近 10 份） |
| 只重新编译 | **菜单 14** |
| 卸载 | **菜单 15**（服务/站点配置/续期任务 + 残留扫描；配置会先备份到 `/root/cboard-uninstall-<时间戳>/`） |

---

## 六、数据库备份

- 脚本在每次升级前会自动备份（见上表）。
- 手动备份（SQLite）：`sqlite3 cboard.db ".backup '/root/cboard-$(date +%F).db'"`，连 `uploads/` 一起拷走即可。
- Docker 部署：备份宿主的 `./data` 与 `./uploads` 目录。
- 数据库文件是 `项目目录/cboard.db`（**不要**在服务运行时直接 `cp`，用 `.backup` 或先停服务）。

---

## 七、系统要求

| 项 | 要求 |
|---|---|
| 系统 | Debian / Ubuntu / CentOS 等，root 权限 |
| 配置 | 1 核 1G 可跑；建议 2 核 2G 以上（编译 Go + Vite 要内存） |
| 端口 | 80、443 放行（云安全组 + 系统防火墙；脚本不改防火墙） |
| 域名 | A 记录指向本机；**项目目录名 = 域名** |

---

## 更多文档

| 文档 | 内容 |
|---|---|
| [纯 VPS 一键脚本教程](docs/部署/VPS部署教程-一键脚本.md) | install.sh 全流程、菜单详解 |
| [宝塔面板部署教程](docs/部署/宝塔部署教程.md) | 面板安装、建站、证书两种模式、常见坑 |
| [安装问题排查指南](docs/故障排查/安装问题排查指南.md) | 安装失败逐项排查 + 运维常见问题 |
| [API 文档](docs/接口/API文档.md) | 接口清单 |
| [文档总索引](docs/README.md) | 邮件、支付、通知、主题等全部配置说明与文档 |

## 许可证

本项目采用 **MIT 许可证**，全文见 [LICENSE](LICENSE)。
