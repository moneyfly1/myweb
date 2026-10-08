# CBoard 灾难恢复手册（VPS 挂了怎么办）

面向「这台 Oracle 日本 VPS 突然不能用」的场景。目标是：**在一台全新 VPS 上，把
dy.moneyfly.top 恢复到完全一致的状态**，并尽量缩短中断时间。

---

## 一、先分清：什么丢了就没了，什么能重建

| 数据 | 位置 | 可重建？ | 说明 |
|---|---|---|---|
| **数据库** | `/www/wwwroot/dy.moneyfly.top/cboard.db`（约 282 MB） | ❌ 不可 | 用户/订单/订阅/专线节点，唯一真相源 |
| **用户上传** | `uploads/config`、`uploads/tickets`、`uploads/repo_sync` | ❌ 不可 | 站点 logo、工单附件、同步文件 |
| **密钥配置** | `.env`（SECRET_KEY / 数据库路径 / 支付与邮件密钥） | ❌ 不可 | 丢了所有人都要重新登录，支付/邮件失效 |
| **站点配置** | 宝塔 `vhost/nginx/*.conf`、systemd 单元、crontab | ❌ 不可 | 重建要重写，容易漏 |
| **证书续期状态** | `/etc/letsencrypt/renewal/` | ⚠️ 半可 | 证书可重签，但续期配置要重建 |
| 代码 | GitHub（两个远端） | ✅ 可 | `git clone` 即可 |
| 后端二进制 / 前端产物 | 服务器 | ✅ 可 | 服务器上 `go build` + `vite build` 重建 |
| TLS 证书 | Let's Encrypt | ✅ 可 | `certbot` 重新签发 |
| BBR / 内核参数 / 软件包 | 系统 | ✅ 可 | 按下方步骤重做 |

> 结论：**只要「数据库 + uploads + .env + 站点配置」这四样在手，换机就是体力活，不是灾难。**

---

## 二、把备份送出这台机器（最重要的一步）

线上已有本地备份（`ops/backup-cboard.sh`：日备份 7 份 + 周备份 4 份 + 配置归档）。
但**本地备份和 VPS 一起死**，所以必须异地。数据库 gzip 后仅约 **24 MB**，很容易送出去。

任选其一（按推荐顺序）：

| 方案 | 做法 | 代价 |
|---|---|---|
| **A. 独立私有仓库的 Release/分支**（推荐，最省事） | 新建私有仓库 `cboard-backups`，用细粒度 PAT（Contents: write）；每天把 `cboard-*.db.gz` + 配置归档推上去 | 免费；需一个 PAT |
| **B. 对象存储**（Cloudflare R2 / 阿里云 OSS / S3） | 服务器装 `rclone`，`rclone sync /www/backup/cboard remote:cboard-backup` | 24 MB/天 ≈ 10 元/年以内 |
| **C. 另一台服务器** | `ssh`/`rsync` 到第二台机器（当前 `104.168.30.106` 我这边没有密钥，需要你给） | 需要第二台机器 |
| **D. 你的电脑** | `scp -r vps-moneyfly:/www/backup/cboard ~/cboard-backup`（偶尔手动） | 免费，但不自动 |

**上线后建议**：异地备份完成后自动校验（脚本已带记录数比对：`users/orders/subscriptions` 数量必须与线上一致），并**每月做一次恢复演练**（见第五节）。

---

## 三、全新 VPS 恢复步骤（约 15–30 分钟）

假设新机器公网 IP 为 `<NEW_IP>`，域名仍用 `dy.moneyfly.top`。

### 1. 基础环境

```bash
apt update && apt install -y nginx sqlite3 curl git ca-certificates
# Go（版本对齐 go.mod，当前 1.25+）
curl -fsSL https://go.dev/dl/go1.25.0.linux-amd64.tar.gz | tar -C /usr/local -xz
export PATH=$PATH:/usr/local/go/bin
# Node（构建前端用，20+）
curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && apt install -y nodejs
# 宝塔面板（如果沿用宝塔管理 nginx/证书；也可直接用系统 nginx + certbot）
```

开启 BBR（原服务器已开）：

```bash
cat >>/etc/sysctl.conf <<'EOF'
net.core.default_qdisc=fq
net.ipv4.tcp_congestion_control=bbr
EOF
sysctl -p && sysctl net.ipv4.tcp_congestion_control   # 应输出 bbr
```

### 2. 拉代码 + 建目录

```bash
mkdir -p /www/wwwroot && cd /www/wwwroot
git clone git@github-moneyfly004:moneyfly004/myweb.git dy.moneyfly.top
cd dy.moneyfly.top
```

### 3. 恢复四样不可再生数据

```bash
# ① 数据库（备份是 gzip 过的 VACUUM INTO 快照）
gunzip -c cboard-<STAMP>.db.gz > cboard.db && chown www:www cboard.db
sqlite3 cboard.db "PRAGMA quick_check;"        # 必须输出 ok

# ② 上传目录
tar xzf uploads-backup.tar.gz -C .             # 恢复 uploads/config、uploads/tickets、uploads/repo_sync

# ③ 密钥与站点配置（ops/backup-cboard.sh 生成的配置归档）
tar xzf cboard-config-<STAMP>.tar.gz -C /
#   内含：.env、宝塔 vhost、/etc/nginx/conf.d、cboard.service、cboard-v2.service、letsencrypt renewal
```

### 4. 建服务并启动

```bash
cd /www/wwwroot/dy.moneyfly.top
go build -o server ./cmd/server/main.go         # 约 55 MB
cd frontend && npm ci && npx vite build && cd ..

install -m 644 /root/cboard.service /etc/systemd/system/cboard.service   # 配置归档里已带
systemctl daemon-reload && systemctl enable --now cboard
systemctl is-active cboard                       # 应为 active
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8000/api/v1/health
```

### 5. nginx + 证书

```bash
# 宝塔环境：把归档里的 vhost 放回 /www/server/panel/vhost/nginx/，重载
/www/server/nginx/sbin/nginx -t && /www/server/nginx/sbin/nginx -s reload
# 纯 certbot 环境：
certbot --nginx -d dy.moneyfly.top -d sub.moneyfly.dpdns.org --agree-tos -m <你的邮箱>
```

> 注意：宝塔 nginx 1.28 支持 `http2 on;`，而系统自带 nginx 1.14 不支持（线上就因此
> 有一个 enabled 但启动失败的系统 nginx unit）。新机器**只保留一套 nginx**，别两套并存。

### 6. 切换流量

Cloudflare 控制台把 `dy.moneyfly.top`（及订阅域名）的 A 记录改成 `<NEW_IP>`，或临时关闭小黄云
（DNS only）以便快速验证，确认无误后再打开。

### 7. 装回自动化

```bash
chmod +x ops/*.sh
crontab -e
# 加入：
#   * * * * *  flock -n /var/lock/cboard-selfheal.lock /www/wwwroot/dy.moneyfly.top/ops/server-selfheal.sh
#   @reboot    flock -n /var/lock/cboard-selfheal.lock /www/wwwroot/dy.moneyfly.top/ops/server-selfheal.sh
#   20 4 * * * /www/wwwroot/dy.moneyfly.top/ops/backup-cboard.sh >> /var/log/cboard-backup.log 2>&1
```

### 8. 验收清单

- [ ] `sqlite3 cboard.db "select count(*) from users;"` 与备份记录数一致
- [ ] 官网可打开、可登录（老用户密码仍有效 → 说明 `.env` 的 `SECRET_KEY` 恢复正确）
- [ ] 后台「用户列表/订单/专线节点」数据量正常
- [ ] 客户端订阅链接可拉取（`curl -A clash-verge/v2.0.0 https://dy.moneyfly.top/api/v1/subscribe/<token>`）
- [ ] 上传的 logo/工单附件可访问
- [ ] 支付回调与邮件发送配置在「系统设置」里正常显示

---

## 四、想「几乎不断服」的两种做法

1. **热备 + DNS 切换（性价比最高）**
   第二台机器上跑同一套（数据库每天同步、nginx 配置一致），Cloudflare 把 A 记录指过去即可切换。
   中断时间 = DNS 生效时间（Cloudflare 代理下通常 1–2 分钟）。日常可让备机只跑 nginx，
   主库定期同步过去（24 MB 的备份文件推送很快）。

2. **Cloudflare 兜底页 / Worker 故障转移**
   主站挂掉时，用 Cloudflare Worker 返回维护页或转发到备机，避免用户看到 Cloudflare 的 522 错误页。
   免费版可用 Worker（付费版有 Load Balancing + 健康检查自动切换）。

3. **监控告警**（先知道，再处理）
   UptimeRobot（免费）或自建 cron 每分钟探测 `https://dy.moneyfly.top/`，
   连续 3 次失败发邮件/Telegram。`ops/server-selfheal.sh` 已在本机记录自检失败日志。

---

## 五、演练（最重要，别只备份不验证）

每季度做一次，任选成本最低的形式：

1. **备份可恢复性**：把最新 `cboard-*.db.gz` 解到本地，`PRAGMA integrity_check` + 表数量比对；
2. **换机演练（推荐半年一次）**：找一台临时小机器，按第三节走一遍到「能登录后台」为止，
   记录下来卡住的地方并更新本手册；
3. **重启演练**：`reboot` 后确认自愈守护把 nginx 与服务拉起来了（这也是验证第一节隐患的唯一方法）。

---

## 六、当前已知风险与待办

| 风险 | 现状 | 处置 |
|---|---|---|
| 异地备份缺失 | 只有本机 `/www/backup/cboard` | 按第二节选一个方案接上 |
| 宝塔 nginx 无自启单元 | 已用 `ops/server-selfheal.sh`（每分钟 + @reboot）兜底 | 建议再做一次真实重启验证 |
| 系统 nginx 1.14 的 unit 处于 enabled/failed | 起因是宝塔 vhost 里的 `http2 on;` | 可 `systemctl disable nginx` 消除隐患（不影响现网） |
| 磁盘：282 MB×多份「corrupt」残留 | 约 2.0 GB 冗余（10-02 事故留下） | 确认最新备份可用后可清理 |
| 数据库曾损坏（10-02） | 现库 `quick_check` 为 ok | 保持日备份 + 磁盘余量告警 |
