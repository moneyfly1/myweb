# CBoard — Self-hosted Subscription Panel (Go + Vue 3)

One command to install, on a **bare VPS or a BaoTa panel**; it finishes by printing your login URL and
admin credentials (after verifying the login once). SQLite needs no external service — Redis / MySQL /
PostgreSQL are optional.

> Only two scripts: **`install.sh`** (the only implementation) and **`bt-deploy.sh`** (BaoTa entry point,
> identical behaviour). Detailed guides: [bare VPS](docs/部署/VPS部署教程-一键脚本.md),
> [BaoTa panel](docs/部署/宝塔部署教程.md), [install troubleshooting](docs/故障排查/安装问题排查指南.md).

---

## 1. Install (three steps)

### Option A — Bare VPS (recommended)

```bash
# 1) Drop the scripts in place (directory name = your domain)
mkdir -p /www/wwwroot/your-domain.com && cd /www/wwwroot/your-domain.com
curl -fsSLO https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
curl -fsSLO https://raw.githubusercontent.com/moneyfly1/myweb/main/bt-deploy.sh

# 2) Run it (root required)
sudo bash install.sh

# 3) Type 1 in the menu and wait
```

Menu 1 does everything: installs dependencies (base packages / gcc / Go 1.25 / Node 22 / Nginx / certbot),
clones the code, writes `.env` (random `SECRET_KEY` + admin credentials), builds backend and frontend,
registers the systemd service, writes the Nginx config, issues the HTTPS certificate, then prints the
login details. **Re-running is safe** (it backs up the database and the previous binary first).

When it finishes you get this (already verified with a real login):

```text
================== 登录信息 ==================
  Frontend login:  https://your-domain.com/login
  Admin login:     https://your-domain.com/admin/login
  Admin user:      admin
  Admin password:  xxxxxxxxxxxxxxxx
==============================================
```

### Option B — BaoTa panel

```bash
# 1) Panel -> App Store -> install Nginx (compiled from source; 25-40 min on 2 vCPU is normal)
# 2) Panel -> Websites -> Add site: domain = your domain, root /www/wwwroot/your-domain.com, PHP = static
#    The directory name MUST equal the domain
# 3) In the site directory, remove placeholders and fetch the scripts
cd /www/wwwroot/your-domain.com
rm -f index.html 404.html .user.ini
curl -fsSLO https://raw.githubusercontent.com/moneyfly1/myweb/main/install.sh
curl -fsSLO https://raw.githubusercontent.com/moneyfly1/myweb/main/bt-deploy.sh

# 4) Deploy
bash bt-deploy.sh        # choose menu 1
```

**Certificates — pick ONE, never both:**

| Mode | How | Who renews |
|---|---|---|
| Script-managed (default) | just run the steps above | `certbot.timer`; **do not** click Apply/Renew in the panel |
| Panel-managed | `CERT_MANAGER=panel bash bt-deploy.sh`, then Panel -> Website -> SSL -> Let's Encrypt -> Apply | the panel; run **menu 16** after a domain change or a panel config rewrite |

### Option C — Docker (optional)

```bash
cp .env.example .env        # set SECRET_KEY (openssl rand -hex 32) and ADMIN_PASSWORD
docker compose up -d --build
curl -fsS http://127.0.0.1:8000/health
```

- Verified in a container: backend build, app running, admin login, and data persistence across a restart.
- **Not yet verified end-to-end:** the in-container frontend build and a full `docker compose up` (the test box
  cannot pull `node:22-alpine`). For production, prefer Options A / B.
- The runtime image **must include `tzdata`** (this repo's Dockerfile does); slim images must copy
  `/usr/share/zoneinfo`, otherwise startup fails with `Invalid _loc: Asia/Shanghai`.

---

## 2. After install — only three things to configure

| Step | What to do |
|---|---|
| **1. Log in** | Use the printed URL and credentials. To pin the password: set `ADMIN_PASSWORD` in `.env` → menu 8 (the app re-applies it on every start), or run **menu 2** |
| **2. Domain & HTTPS** | Already handled. If you change the domain later: rename the project directory (or set `DOMAIN`) and run **menu 16** |
| **3. Optional extras** | SMTP mail, Redis cache (**menu 12** — installs Redis itself, with automatic fallback), payments, notifications |

> Everything else — users, plans, orders, nodes, subscriptions, tickets, invites, coupons, announcements,
> themes, registration settings — is configured **by clicking in the admin panel**; no file editing needed.

---

## 3. Menu cheat-sheet

| Key | Action | Key | Action |
|---|---|---|---|
| `1` | Full auto deploy (use this first) | `9` | Stop service |
| `2` | Create / reset admin (verifies login) | `10` | Renew certificate (manual) |
| `3` | Force restart (project processes only) | `11` | **Sync from GitHub, rebuild, restart** |
| `4` | Deep cache clean (keeps dist + binary) | `12` | Configure Redis (installs Redis) |
| `5` | Unlock a locked user | `13` | Roll back to previous build |
| `6` | Service status | `14` | Rebuild + restart only (no Nginx change) |
| `7` | Live log (`server.log`) | `15` | Full uninstall (with residual scan) |
| `8` | Restart service | `16` | **Self-check & auto-repair** |
| | | `0` | Exit |

---

## 4. Settings (environment variables)

File: `<project>/.env` — apply changes with **menu 8**.

| Variable | Default | Notes |
|---|---|---|
| `HOST` / `PORT` | `127.0.0.1` / `8000` | Keep loopback; Nginx proxies to it |
| `DATABASE_URL` | `sqlite:////www/wwwroot/<domain>/cboard.db` | Keep the **absolute** path |
| `SECRET_KEY` | random 64 chars | Session signing key — rotate if leaked |
| `ADMIN_USERNAME` / `ADMIN_EMAIL` / `ADMIN_PASSWORD` | `admin` / `admin@<domain>` / random 16 | Password is re-applied on every start |
| `TRUSTED_PROXIES` | empty | Set to `127.0.0.1` behind Nginx / CDN |
| `PANEL_PUBLIC_URL` | empty | Required when using self-hosted nodes |
| `REDIS_ADDR` / `REDIS_PASSWORD` | empty | Written by menu 12 |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USERNAME` / `SMTP_PASSWORD` | empty | Mail (also configurable in the panel) |
| `DISABLE_SCHEDULE_TASKS` | `false` | `true` disables scheduled jobs |
| `DB_BACKUP_DIR` / `DB_BACKUP_KEEP` | `/www/backup/cboard` / `10` | Backup location and retention |

---

## 5. Upgrade / backup / rollback

| Task | How |
|---|---|
| Upgrade to latest code | **Menu 11** (backs up DB → builds → atomic swap → restart + health check) |
| Roll back | **Menu 13** (`server.bak.*` + database backup) |
| Automatic backups | `/www/backup/cboard/pre-upgrade-*.db.gz` (last 10 by default) |
| Rebuild only | **Menu 14** |
| Uninstall | **Menu 15** (service/config/renewal tasks + residual scan; backup in `/root/cboard-uninstall-<ts>/`) |

---

## 6. Database backup

- The script backs up automatically before every upgrade (see above).
- Manual (SQLite): `sqlite3 cboard.db ".backup '/root/cboard-$(date +%F).db'"`, plus copy `uploads/`.
- Docker: back up the host `./data` and `./uploads` directories.
- Never `cp` the live `cboard.db` while the service runs — use `.backup` or stop the service first.

---

## 7. Requirements

| Item | Requirement |
|---|---|
| OS | Debian / Ubuntu / CentOS etc., root access |
| Sizing | 1 vCPU / 1 GB works; 2 vCPU / 2 GB+ recommended (Go + Vite builds) |
| Ports | 80 and 443 open (cloud security group **and** host firewall; the script does not touch firewalls) |
| Domain | A record pointing at the server; **project directory name = domain** |

---

## More documentation

| Document | Contents |
|---|---|
| [Bare-VPS one-click guide](docs/部署/VPS部署教程-一键脚本.md) | Full install.sh walkthrough and menus |
| [BaoTa panel guide](docs/部署/宝塔部署教程.md) | Panel install, site setup, both certificate modes |
| [Install troubleshooting](docs/故障排查/安装问题排查指南.md) | Install failures + day-2 operations FAQ |
| [API reference](docs/接口/API文档.md) | Endpoint list |
| [Documentation index](docs/README.md) | Everything else: mail, payments, notifications, themes, operations |

## License

See [LICENSE](LICENSE).
