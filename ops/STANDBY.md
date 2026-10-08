# 迁移 / 切换到另一台 VPS（无缝转移方案）

本文回答：**主 VPS 万一不能用，能不能切到另一台（配置更低的）VPS 上、几乎不断服？**
结论：**能**。实测本面板极轻，配合「每分钟 DB 快照同步 + 切换脚本」，可做到
**中断约 0.5–2 分钟、数据零丢失**。

---

## 一、这台站点的真实资源需求（实测，非估算）

| 项目 | 实测值 | 说明 |
|---|---|---|
| 后端内存 | **354 MB** | Go 服务 1 个进程 |
| nginx 内存 | **103 MB** | 静态前端 + 反代 |
| CPU | 2 核，负载常年在 0.0x | 1 核足够 |
| **必须迁移的数据** | **约 300 MB** | 数据库 282 MB + `uploads/config`(8 MB) + `uploads/tickets`(1.3 MB) + `uploads/repo_sync`(6 MB) |
| 可丢弃的数据 | 1.7 GB | `uploads/logs`(820 MB) + `uploads/backups`(919 MB) 都是本机日志/旧备份，备机不需要 |
| **月出流量** | **约 9 GB** | 订阅拉取 7206 次/天 × 平均约 40 KB，主力域名 `dy.moneyfly.club` |
| 构建需求 | Go 编译峰值约 1 GB 内存 | 小机器建议**本地交叉编译**后只传二进制，别在备机上编译 |

> 换算：**1 核 / 1 GB 内存 / 10 GB 磁盘 / 100 GB 月流量** 的低配 VPS 即可承接。
> 如果只有 512 MB 内存，也能跑，但要关掉编译、并确认没有其他站点抢内存。

---

## 二、为什么可以「无缝」

关键前提（当前架构天然满足）：

1. 客户端订阅、支付回调、节点回传**全部走同一个域名**（`dy.moneyfly.club` / `dy.moneyfly.top`，
   DNS 在 Cloudflare）→ **换 origin 只需改 DNS/回源，客户端无需改任何配置**；
2. `.env` 里的 `SECRET_KEY` 一旦保持一致，**用户不会被迫重新登录**，token 继续有效；
3. 所有可再生的东西（代码、前端产物、证书）都能重建，真正要同步的只有约 300 MB。

---

## 三、三个档位（按你能接受的停机时间选）

| 档位 | 做法 | 中断时间 | 数据丢失 | 工程量 |
|---|---|---|---|---|
| **L1 冷备** | 只保证备份可用（已有），出事手工按 `ops/RESTORE.md` 恢复 | 30–60 分钟 | 最多 1 天（备份周期） | 已完成 |
| **L2 温备 + 秒级切换（推荐）** | 备机常驻同版本程序与 nginx，`ops/standby-sync.sh` 每分钟把一致性快照推过去；切换时停主站 → 最后一次同步 → 起备站 → Cloudflare 改回源 | **约 0.5–2 分钟** | **0**（切换前做最后一次同步） | 中，1 台备机 |
| **L3 双活零中断** | 数据库从 SQLite 换成 MySQL/PostgreSQL 主从 + Cloudflare Load Balancing（付费）双活 | 0（自动切换） | 0 | 大，需改数据层 + 付费 LB |

⚠️ **SQLite 的限制**：数据库是单文件、单写者，**两台机器不能同时写**。所以 L2 的「无缝」
= 「秒级切换 + 零丢数据」，而不是「两台同时在线」。想要真正双活只能走 L3。

L3 的迁移成本提示：项目用 GORM，换 MySQL 主要是改连接配置与少量 SQLite 特有语法
（`VACUUM`、`INSERT OR REPLACE` 等），但要全量回归测试 + 数据迁移 + 回滚方案，
建议在主备方案稳定运行后，作为第二阶段项目单独做。

---

## 四、L2 落地步骤

### 1. 备机准备（一次性）

```bash
# 备机：装运行环境（不装编译环境也行，用下面第 2 步的方式传二进制）
apt update && apt install -y nginx sqlite3 curl git

# 目录与代码
mkdir -p /www/wwwroot && cd /www/wwwroot
git clone git@github-moneyfly004:moneyfly004/myweb.git dy.moneyfly.top
cd dy.moneyfly.top

# 恢复「必须迁移的数据」（从主站备份或直接 rsync）
#   主站执行： rsync -avz /www/wwwroot/dy.moneyfly.top/cboard.db* 备机:/www/wwwroot/dy.moneyfly.top/
#   主站执行： rsync -avz ops/backup-cboard.sh  # 或直接同步 uploads/{config,tickets,repo_sync}
#   主站执行： scp /www/wwwroot/dy.moneyfly.top/.env 备机:.../.env      # SECRET_KEY 必须一致
#   主站执行： scp /etc/systemd/system/cboard.service 备机:/etc/systemd/system/

# 二进制：主站编译后直接传（避免小机器编译 OOM）
#   主站执行： go build -o server ./cmd/server/main.go && scp server 备机:/www/wwwroot/dy.moneyfly.top/
# 前端产物：主站构建后打包传过去
#   主站执行： cd frontend && npx vite build && tar czf /tmp/dist.tgz dist && scp /tmp/dist.tgz 备机:/tmp/
#   备机执行： tar xzf /tmp/dist.tgz -C /www/wwwroot/dy.moneyfly.top/frontend/

systemctl daemon-reload && systemctl enable cboard
```

### 2. 证书

备机同样需要 `dy.moneyfly.club` / `dy.moneyfly.top` 的证书：

- **推荐**：Cloudflare **Origin Certificate**（15 年有效、不用续期），主备共用同一张；
- 或备机用 `certbot` 自行签发（注意备机 80 端口要能通过 HTTP-01 校验，或改 DNS-01）。

### 3. 进入「温备待命」状态

```bash
# 备机：先不对外提供服务，只保持数据新鲜
systemctl stop cboard 2>/dev/null || true
# 主站：安装同步（每分钟一次）
crontab -e
#   * * * * *  /www/wwwroot/dy.moneyfly.top/ops/standby-sync.sh <备机IP或别名> >> /var/log/cboard-standby-sync.log 2>&1
```

### 4. 切换（出事时执行）

```bash
# ① 主站（如果还能登录）：做最后一次同步，确保零丢失
/www/wwwroot/dy.moneyfly.top/ops/standby-sync.sh <备机> --final

# ② 备机：把同步来的快照就位并启动
cd /www/wwwroot/dy.moneyfly.top
systemctl stop cboard
mv cboard.db.standby cboard.db && chown www:www cboard.db
sqlite3 cboard.db "PRAGMA quick_check;"        # 必须 ok
systemctl start cboard
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8000/api/v1/health

# ③ Cloudflare：把 dy.moneyfly.club / dy.moneyfly.top 的回源指向备机 IP
#    （Cloudflare 面板改 A 记录；用「DNS only」先验证，再打开小黄云）
```

`ops/switch-to-standby.sh` 把 ①②③ 里能自动化的部分都封好了（含回滚提示）。

### 5. 回滚

备机起来的版本、`.env`、DB 都与主站一致 → 想切回主站，把 Cloudflare 回源改回主站 IP 即可
（主站数据比备机新的话，先把备机 `cboard.db` 拉回主站，避免丢切换期间的写入）。

---

## 五、需要你提供（我才能把备机真正做到「待命」）

1. 那台 VPS 的 **IP 或 SSH 别名**，以及我能登录的方式（我这边尝试 `104.168.30.106` 被拒了）；
2. 它的**规格**：内存 / 磁盘 / 月流量额度 / 系统版本（尤其**流量额度**，见第一节：需要 ≥10 GB/月）；
3. 是否同一 Cloudflare 账号（决定切换是改 DNS 还是只能换 NS）；
4. 备机是否允许长期常驻（有些试用机有期限，注意别把「备机」放到会到期的机器上）。

---

## 六、日常运行成本

| 项目 | 主站 | 备机 |
|---|---|---|
| 内存 | 已有 | 空闲常驻约 60–100 MB（服务停着，只跑 nginx/同步） |
| 流量 | 约 9 GB/月 | 同步约 24 MB/分钟？→ 实际按**变化量**算：DB 快照 24 MB × 1440 次/天 ≈ 35 GB/月 ✗ 太贵 |
| 优化 | — | 改为**每 10 分钟**同步（3.5 GB/月）+ 切换前做一次 `--final`；或只同步白天高频时段 |

> 上面这一行是本方案唯一需要权衡的点：**同步频率越高，备机流量越贵**。
> 默认建议：**每 10 分钟**同步 + 切换时 `--final` 补一次（数据零丢失、日常流量约 3.5 GB/月）。
