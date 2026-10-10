# 一键脚本部署教程（纯 VPS / 宝塔通用 · `install.sh`）

> 适用：Debian 10+ / Ubuntu 18.04+ / CentOS 7+ / Rocky / AlmaLinux，**纯 VPS 或已装宝塔面板都可以**。
> `install.sh` 会自动识别环境（宝塔就用宝塔的 nginx 与站点目录，没装就装系统 nginx）。

## 0. 三个脚本用哪个（结论先给）

| 脚本 | 说明 |
|------|------|
| **`install.sh`** | ✅ **唯一实现，就用它**：部署 + 运维 + 自检修复 + 完全卸载 |
| **`bt-deploy.sh`** | ✅ 宝塔入口（64 行薄封装）：检查宝塔环境并提示证书策略，然后转交 `install.sh`，行为完全一致 |
| ~~`install-vps.sh`~~ | ❌ 已删除。它固定 Go 1.21.5 / Node 18，与 `go.mod`(Go 1.25) 和 vite 7(Node ≥20.19) 冲突，会在构建前端时失败 |

## 一、前置准备

| 项 | 要求 |
|----|------|
| 系统 | Debian 10+ / Ubuntu 18.04+ / CentOS 7+ / Rocky / AlmaLinux |
| 权限 | root（脚本会安装软件、写 systemd 与 nginx 配置） |
| 配置 | ≥1 核 CPU、≥1 GB 内存（2 GB 更稳，默认会本地构建前端）、≥10 GB 磁盘 |
| 域名 | 已解析到本机（A 记录） |
| 端口 | 80、443 放行（云安全组 **和** 系统防火墙都要放；脚本不改防火墙） |

> ⚠️ **目录名必须等于域名**：脚本用「项目目录名」当域名（例：`/www/wwwroot/example.com`），
> 也可用 `DOMAIN=example.com` 覆盖。

## 二、部署步骤

```bash
# 1) 克隆到「域名同名目录」
git clone https://github.com/moneyfly1/myweb.git /www/wwwroot/你的域名
cd /www/wwwroot/你的域名

# 2) 运行脚本（root）
sudo bash install.sh

# 3) 菜单里输入 1（一键全自动部署），按提示回答 Redis 相关提问
```

宝塔机器上也可以用宝塔入口（等价）：

```bash
sudo bash bt-deploy.sh      # 会先提示宝塔环境/证书策略，再转交 install.sh
```

非交互 / 自动化场景（脚本对 EOF 安全，不会空转）：

```bash
printf '1\nn\n' | sudo bash install.sh     # 1=全自动部署，n=不配置 Redis
```

## 三、脚本会自动做什么

| 阶段 | 内容 |
|------|------|
| ① 自举依赖 | 基础包（git/sqlite3/wget/curl）→ gcc（SQLite 的 CGO 必需）→ **Go 1.25.0** → **Node 22.12.0** → **Nginx**（已装宝塔则直接用宝塔 nginx） |
| ② 源码与环境 | 源码缺失时自动 `git clone`；生成 `.env`（`HOST=127.0.0.1`、数据库**绝对路径**、随机 64 位 `SECRET_KEY`、权限 600） |
| ③ 构建 | 后端 `CGO_ENABLED=1` 编译到 `server.new` → 校验 → 替换（旧版存 `server.bak.<时间戳>` 供回滚）；前端按 lockfile 安装依赖并构建 |
| ④ 服务 | 写 systemd 单元（`EnvironmentFile`/`LimitNOFILE`/`NoNewPrivileges`）→ 启动 → **业务健康检查**（`/health`，不是只看 `is-active`） |
| ⑤ Nginx | 写入/合并站点配置：`/api/`、`/uploads/`、`/repo-sync/`、`/assets/` 长缓存、`index.html` 不缓存、`client_max_body_size 16m`、ACME 放行段；写完先 `nginx -t`，失败自动换 http2 写法或精确回滚 |
| ⑥ 证书 | 复用优先（certbot → 宝塔面板 → acme.sh），没有才申请；配置续期重载钩子与定时任务 |
| ⑦ 收尾 | 日志轮转（logrotate）、每次升级前自动备份数据库到 `/www/backup/cboard` |

**宝塔环境**下采用「合并模式」：保留面板的 `#SSL-START`/`#CERT-APPLY-CHECK` 标记与 include，
只注入本应用必需的片段（面板重写配置后，跑菜单 16 会自动补回）。

## 四、安装后验证

```bash
curl -I https://你的域名                    # 200 + 证书有效
curl https://你的域名/api/v1/packages       # API 反代正常
curl http://127.0.0.1:8000/health           # 后端健康（本机）
systemctl status cboard                     # 服务运行中
tail -f /www/wwwroot/你的域名/server.log     # 应用日志（菜单 7 同效）
```

登录管理后台：`https://你的域名/admin` → 首次请用菜单 2 创建/重置管理员
（脚本会用你设置的口令**实测登录一次**，确认真的能登进去）。

## 五、日常运维（`sudo bash install.sh` 菜单）

| 项 | 功能 |
|----|------|
| 1 | 一键全自动部署（重复执行安全：幂等，会先备份数据库与旧二进制） |
| 2 | 创建/重置管理员（密码不回显；留空自动生成强随机密码；建完自动实测登录） |
| 3 | 强制重启服务（只清理本项目进程，不动 nginx/mysql/redis） |
| 4 | 深度清理缓存（清 Redis 本项目键 + 清空 server.log + Go 编译缓存；**不删** dist 与二进制） |
| 5 | 解锁用户账户（用户名或邮箱） |
| 6 / 7 | 服务状态 / 实时日志（`tail -f server.log`） |
| 8 / 9 | 标准重启 / 停止服务 |
| 10 | 证书续期（自动续期由 certbot.timer 或宝塔面板负责） |
| **11** | **从 GitHub 同步并重建**（升级入口：fetch 失败会报错而不是假装"已是最新"；先备份库再原子构建） |
| 12 | 配置 Redis 缓存（可选；只监听 127.0.0.1，按本项目键前缀清缓存而非 FLUSHDB） |
| **13** | **回滚到升级前版本**（用 `server.bak.*`；数据库备份在 `/www/backup/cboard`） |
| 14 | 只重新构建并重启（不动 nginx/unit） |
| **15** | **完全卸载**（删服务/站点配置/logrotate/续期钩子与 cron；配置副本存 `/root/cboard-uninstall-<时间戳>/`；最后残留扫描） |
| **16** | **自检并自动修复**（缺配置片段、nginx -t 不过、服务不健康、续期任务缺失、权限不对等自动补齐） |

## 六、环境变量开关

| 变量 | 默认 | 作用 |
|------|------|------|
| `CERT_MANAGER` | `auto` | `auto`：复用已有证书，没有则 certbot 签发；`panel`：证书交给宝塔面板（脚本不写 SSL，提示你到面板申请）；`certbot`：强制 certbot |
| `DOMAIN` / `PROJECT_DIR` | 目录名 / 脚本目录 | 覆盖域名与项目目录 |
| `GO_VERSION` / `NODE_VERSION` | `1.25.0` / `22` | 自举安装的版本 |
| `DB_BACKUP_DIR` / `DB_BACKUP_KEEP` | `/www/backup/cboard` / `10` | 升级前数据库备份位置与保留份数 |

## 七、常见问题

| 现象 | 处理 |
|------|------|
| 服务起不来 | `tail -n 100 server.log`；再跑菜单 16 自检修复 |
| 页面白屏 / 还是旧版本 | `index.html` 已配置不缓存；仍异常按 `Ctrl+Shift+R`，并跑菜单 16 |
| 附件/图片打不开 | 检查配置是否含 `location /uploads/`（菜单 16 会自动补） |
| HTTPS 没生效但证书已签 | 多为 nginx 版本与 http2 写法不匹配，菜单 16 会自动换写法修复 |
| 上传大文件 413 | 配置里 `client_max_body_size 16m`；要更大就改它或调 `.env` 的 `MAX_FILE_SIZE` |
| 域名打不开 | 先确认 DNS 指向本机、80/443 安全组已放行 |
| 构建前端失败 | 确认 Node ≥ 20.19（脚本会自己装 22.12.0）；离线环境请先联网或换 npm 镜像 |
| 想彻底重来 | 菜单 15 完全卸载 → 重新克隆 → 菜单 1 |
