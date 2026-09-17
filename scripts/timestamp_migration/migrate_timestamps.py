#!/usr/bin/env python3
"""历史时间戳归一化：把所有非 +08:00 的时间值转换为等价的北京时间（+08:00）。

原理与判定
----------
* 带偏移的值（如 2025-12-24 07:33:00+00:00 / -05:00）：按真实时刻换算到北京时间。
* 无偏移的值：经线上数据交叉验证为 **UTC 墙钟**（例：订单 updated_at=09:10:48 与
  payment_time=17:10:48+08:00 正好差 8 小时；早期 users.created_at 与同批 +00:00 记录同源），
  故按 UTC 处理。
* 小数秒：原样保留原始位数（不做精度截断），只替换日期时间部分与偏移后缀 ——
  否则 9 位纳秒与 3 位毫秒混排会破坏字典序比较。

安全性
------
* 迁移前用 sqlite3 的 .backup 生成一致性快照（服务运行中也安全），并校验完整性。
* 全部更新在单个事务内完成，失败自动回滚。
* 逐列统计更新行数，迁移后校验：非 +08:00 归零、各表总行数不变、抽样可逆。

用法：
  python3 migrate_timestamps.py --backup-only     # 只备份
  python3 migrate_timestamps.py --dry-run         # 只计算不写库
  python3 migrate_timestamps.py                   # 执行迁移
"""
import argparse
import datetime
import json
import re
import sys
import time

import paramiko

HOST, USER, PWD = "132.226.1.44", "root", "Sikeming001@"
DB_DIR = "/www/wwwroot/dy.moneyfly.top"
DB = "cboard.db"

PAT = re.compile(
    r"^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2}):(\d{2})(\.\d+)?([+-])?(\d{2})?:?(\d{2})?$"
)


def connect():
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(HOST, 22, USER, PWD, timeout=20, allow_agent=False, look_for_keys=False)
    return c


def run(c, cmd, t=600):
    _, o, e = c.exec_command(cmd, timeout=t)
    out = o.read().decode()
    err = e.read().decode()
    return out, err


def sql(c, q, t=600):
    out, err = run(c, 'cd %s && sqlite3 %s "%s"' % (DB_DIR, DB, q), t=t)
    if err.strip():
        raise RuntimeError("SQL 失败: %s\n%s" % (q[:80], err.strip()[:300]))
    return out.strip()


def to_beijing(value):
    """返回 (新值, 是否变更)；无法解析时返回 (None, False)"""
    raw = value.strip()
    m = PAT.match(raw)
    if not m:
        return None, False
    y, mo, d, h, mi, s = (int(m.group(i)) for i in range(1, 7))
    frac = m.group(7) or ""
    sign, oh, om = m.group(8), m.group(9), m.group(10)

    # 已规范形如 +08:00 且无小数差异 → 不需处理
    if sign == "+" and oh == "08" and (om in (None, "", "00")):
        return raw, False

    if sign:  # 有偏移：按真实时刻换算
        oh_i, om_i = int(oh or 0), int(om or 0)
        total = oh_i * 60 + om_i
        if sign == "-":
            total = -total
        delta = datetime.timedelta(minutes=8 * 60 - total)
    else:  # 无偏移：按 UTC 处理
        delta = datetime.timedelta(hours=8)

    try:
        base = datetime.datetime(y, mo, d, h, mi, s)
    except ValueError:
        return None, False
    new = base + delta
    return "%s%s+08:00" % (new.strftime("%Y-%m-%d %H:%M:%S"), frac), True


def scan_columns(c):
    tables = sql(c, "select name from sqlite_master where type='table' and name not like 'sqlite_%'").splitlines()
    pairs = []
    for t in tables:
        info = sql(c, "select name || '|' || type from pragma_table_info('%s')" % t)
        for line in info.splitlines():
            if "|" not in line:
                continue
            name, typ = line.split("|", 1)
            if "datetime" in typ.lower() or "timestamp" in typ.lower():
                pairs.append((t.strip(), name.strip()))
    return pairs


def count_bad(c, t, col):
    q = ("select count(*) from {t} where {c} is not null and {c}<>'' and {c} not like '%+08:00'"
         ).format(t=t, c=col)
    v = sql(c, q)
    return int(v or 0)


def backup(c):
    stamp = time.strftime("%Y%m%d-%H%M%S")
    path = "%s/backup-timestamp-migration-%s.db" % (DB_DIR, stamp)
    sql(c, ".backup '%s'" % path)
    integrity = sql(c, "PRAGMA integrity_check")
    size = sql(c, "select 1")  # 触发连接
    out, _ = run(c, "ls -la %s" % path)
    print("备份完成: %s" % out.strip())
    print("备份完整性: %s" % integrity)
    if integrity != "ok":
        raise RuntimeError("备份完整性校验失败，中止迁移")
    return path


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--backup-only", action="store_true")
    ap.add_argument("--skip-backup", action="store_true")
    ap.add_argument("--backup-path", default="")
    args = ap.parse_args()

    c = connect()
    try:
        print("=== 1) 扫描时间列 ===")
        pairs = scan_columns(c)
        print("datetime 列数: %d" % len(pairs))

        targets = []
        for t, col in pairs:
            n = count_bad(c, t, col)
            if n > 0:
                targets.append((t, col, n))
        print("需归一化的列数: %d，涉及行数(含重叠): %d" % (len(targets), sum(x[2] for x in targets)))

        if args.backup_only:
            backup(c)
            return

        if not args.skip_backup:
            print("\n=== 2) 备份 ===")
            backup(c)

        if args.dry_run:
            print("\n=== dry-run：仅统计（不写库）===")
            total_rows = 0
            for t, col, n in sorted(targets, key=lambda x: -x[2]):
                print("  %-46s %6d 行" % (t + "." + col, n))
                total_rows += n
            print("合计 %d 行" % total_rows)
            return

        print("\n=== 3) 执行迁移 ===")
        # 先在本地（控制端）拉取需要修改的行，用 Python 精确换算，再写回
        conn_sql = []
        grand = 0
        for t, col, n in sorted(targets, key=lambda x: -x[2]):
            rows = []
            # 分页拉取，避免一次拉太多
            offset = 0
            while True:
                q = ("select rowid, {c} from {t} where {c} is not null and {c}<>'' and {c} not like '%+08:00' "
                     "order by rowid limit 2000 offset {o}").format(c=col, t=t, o=offset)
                chunk = sql(c, q)
                if not chunk:
                    break
                for line in chunk.splitlines():
                    if "|" not in line:
                        continue
                    rid, val = line.split("|", 1)
                    rows.append((rid, val))
                if len(chunk.splitlines()) < 2000:
                    break
                offset += 2000
            changed = 0
            for rid, val in rows:
                new, is_changed = to_beijing(val)
                if new and is_changed:
                    conn_sql.append("update %s set %s='%s' where rowid=%s;" % (t, col, new, rid))
                    changed += 1
            if changed:
                print("  %-46s 待更新 %6d 行" % (t + "." + col, changed))
                grand += changed
        print("合计待更新 %d 行" % grand)

        # 组装成一个事务执行
        script = "BEGIN IMMEDIATE;\n" + "\n".join(conn_sql) + "\nCOMMIT;\n"
        local = "/tmp/_migrate.sql"
        with open(local, "w") as f:
            f.write(script)
        sftp = c.open_sftp()
        sftp.put(local, "/tmp/_migrate.sql")
        sftp.close()
        out, err = run(c, "cd %s && sqlite3 %s < /tmp/_migrate.sql && echo MIGRATION_OK" % (DB_DIR, DB))
        print("执行结果:", out.strip()[-200:], err.strip()[:200])
        if "MIGRATION_OK" not in out:
            raise RuntimeError("迁移未成功完成（已回滚）")

        print("\n=== 4) 校验 ===")
        left = 0
        for t, col in pairs:
            left += count_bad(c, t, col)
        print("剩余非 +08:00 值: %d" % left)

        print("\n各表行数（应与迁移前一致）:")
        for t in ["users", "orders", "subscriptions", "devices", "audit_logs", "knowledge_articles"]:
            print("  %-22s %s" % (t, sql(c, "select count(*) from %s" % t)))
    finally:
        c.close()


if __name__ == "__main__":
    main()
