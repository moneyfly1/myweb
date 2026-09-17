# 历史时间戳归一化（一次性迁移，2026-09-17 已执行）

## 背景

代码侧已统一为「一律写北京时间（+08:00）」（GORM NowFunc + SQLite `_loc` + 系统 TZ），
但历史数据中仍混有三种偏移：

| 来源 | 偏移 | 说明 |
|---|---|---|
| 早期部署在 UTC 主机 | `+00:00` | `.UTC()` 代码路径与 GORM `time.Now().Local()` |
| 早期部署在美东主机 | `-05:00` | 同上，宿主机时区漂移 |
| 当前及多数记录 | `+08:00` | 显式 `GetBeijingTime()` |
| 早期部分路径 | 无偏移 | 语义为 **UTC**（已用订单 `updated_at` 与 `payment_time` 差 8 小时交叉验证） |

SQLite 的 datetime 是**文本**、按字典序比较，跨偏移行的 `WHERE created_at >= ?`、
`ORDER BY created_at`、到期判定都会出错（实测同一天注册数按字符串算是 4、按真实时刻是 3）。

## 做法

1. `scan_columns.py` 扫描 schema 中所有 datetime 列，精确统计需处理行数
   （偏移判定用 `*[+-]HH:MM` glob，避免把"无偏移但带秒"误判为有偏移）
2. `snapshot.py` 抓迁移前后基线（行数/金额/业务计数），确认只改表示、不改真实时刻
3. `migrate_timestamps.py` 执行迁移：
   - 先用 `sqlite3 .backup` 生成一致性快照并校验 `integrity_check`
   - 按真实时刻换算到 +08:00；无偏移按 UTC 处理
   - **小数秒原样保留原始位数**（9 位纳秒不做截断，否则与毫秒值混排会破坏字典序）
   - 全部更新在单个事务内（`BEGIN IMMEDIATE ... COMMIT`），失败自动回滚

## 本次结果（2026-09-17）

- 待处理：34 个时间列、**2596 行**
- 实际更新：**2595 行**（1 行为已规范形式）
- 迁移后残留非 `+08:00` 值：**0**
- 行数与金额指标前后完全一致（`users` 1474 / `orders` 500 / `subscriptions` 1473 /
  已支付订单额 65037.09 等）
- 语义核对：订单 3 `updated_at=17:10:48+08:00` 与 `payment_time=17:10:48.367+08:00` 精确对齐
  （迁移前二者相差 8 小时）；设备 `-05:00` 记录正确换算为 18:55:38（+13h），9 位小数保留

备份文件位于 VPS：`/www/wwwroot/dy.moneyfly.top/backup-timestamp-migration-<时间戳>.db`

## 第二步：小数秒精度统一（同日执行）

统一到**秒精度**（固定 25 字符 `YYYY-MM-DD HH:MM:SS+08:00`），原因：

- Go 的 sqlite 驱动按 `.999999999` 格式化并**去掉末尾零**，同一列会混有 0/3/6/9 位小数
- 与 MySQL `DATETIME(0)` 语义对齐（本项目同时支持 MySQL/PostgreSQL）
- 与展示格式 `utils.FormatBeijingTime` 一致，外部工具按固定长度解析也不会错

代码侧同步改动（否则新写入会再次漂移）：

- `timeutil.NowForDB()` = `Now().Truncate(time.Second)`；`utils.GetBeijingTime()` 委托它
- GORM `NowFunc = timeutil.NowForDB`（覆盖全部 autoCreateTime/autoUpdateTime）
- `selfhost` 服务、工单附件归档目录等绕开统一入口的 `time.Now()` 一并改为秒精度北京时间
- **耗时/延迟统计仍用 `time.Now()`**（带单调时钟），不受影响

`migrate_fraction.py` 用纯 SQL 截断（`substr(col,1,19) || substr(col,-6)`），
17 万个值在单事务内完成，无需逐行往返。

结果：需处理 79 个列、**171,946 个值**；迁移后残留含小数秒的值 **0**；
`orders.created_at` 长度分布 501/501 全部为 25 字符。

## 用法（如需在新环境复用）

```bash
python3 scan_columns.py            # 先看规模（需先导出 datetime 列清单）
python3 snapshot.py before.json    # 迁移前基线
python3 migrate_timestamps.py --dry-run
python3 migrate_timestamps.py      # 执行（自动备份）
python3 snapshot.py after.json     # 迁移后对比
python3 migrate_fraction.py --dry-run   # 小数秒精度统一（看规模）
python3 migrate_fraction.py             # 执行（自动备份）
```
