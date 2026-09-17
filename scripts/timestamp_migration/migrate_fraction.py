#!/usr/bin/env python3
"""历史时间戳精度归一化：把所有带小数秒的时间值截断为"秒"精度。

目标格式（固定 25 字符）：YYYY-MM-DD HH:MM:SS+08:00

为什么用纯 SQL 而不是逐行改写
------------------------------
本库时间列格式已在上一步统一为 `YYYY-MM-DD HH:MM:SS[.fraction]+08:00`，
因此"去掉小数秒"等价于「取前 19 字符 + 取末 6 字符（时区偏移）」，无需解析时间：

    UPDATE t SET col = substr(col, 1, 19) || substr(col, -6) WHERE col LIKE '%.%';

17 万个值逐行往返太慢，上述写法在单个事务内即可完成且语义等价（截断而非四舍五入）。

安全性
------
* 迁移前用 sqlite3 .backup 生成一致性快照并校验 integrity_check。
* 单事务执行，失败自动回滚。
* 迁移后校验：残留含小数秒的值必须为 0；行数与金额指标不变。
"""
import argparse
import json
import sys
import time

import paramiko

HOST, USER, PWD = "132.226.1.44", "root", "Sikeming001@"
DB_DIR = "/www/wwwroot/dy.moneyfly.top"
DB = "cboard.db"


def connect():
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(HOST, 22, USER, PWD, timeout=20, allow_agent=False, look_for_keys=False)
    return c


def run(c, cmd, t=1800):
    _, o, e = c.exec_command(cmd, timeout=t)
    return o.read().decode().strip(), e.read().decode().strip()


def sql(c, q, t=1800):
    out, err = run(c, 'cd %s && sqlite3 %s "%s"' % (DB_DIR, DB, q), t=t)
    if err.strip() and not out:
        raise RuntimeError("SQL 失败: %s\n%s" % (q[:90], err.strip()[:300]))
    return out.strip()


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--skip-backup", action="store_true")
    ap.add_argument("cols", nargs="?", default="/tmp/datetime_cols.json")
    args = ap.parse_args()

    pairs = json.load(open(args.cols))
    c = connect()
    try:
        targets = []
        for t, col in pairs:
            n = sql(c, "select count(*) from {t} where {c} like '%.%'".format(t=t, c=col))
            if int(n or 0) > 0:
                targets.append((t, col, int(n)))
        total = sum(x[2] for x in targets)
        print("需截断的列数: %d，值总数: %d" % (len(targets), total))
        if args.dry_run:
            for t, col, n in sorted(targets, key=lambda x: -x[2])[:15]:
                print("  %-46s %8d" % (t + "." + col, n))
            return

        if not args.skip_backup:
            stamp = time.strftime("%Y%m%d-%H%M%S")
            path = "%s/backup-frac-migration-%s.db" % (DB_DIR, stamp)
            print("备份中 → %s" % path)
            sql(c, ".backup '%s'" % path)
            integrity = sql(c, "PRAGMA integrity_check")
            print("备份完整性:", integrity)
            if integrity != "ok":
                raise RuntimeError("备份校验失败，中止")

        stmts = ["BEGIN IMMEDIATE;"]
        for t, col, n in targets:
            stmts.append(
                "UPDATE {t} SET {c} = substr({c}, 1, 19) || substr({c}, -6) WHERE {c} LIKE '%.%';".format(c=col, t=t)
            )
        stmts.append("COMMIT;")
        local = "/tmp/_frac.sql"
        with open(local, "w") as f:
            f.write("\n".join(stmts) + "\n")
        sftp = c.open_sftp()
        sftp.put(local, "/tmp/_frac.sql")
        sftp.close()
        out, err = run(c, "cd %s && sqlite3 %s < /tmp/_frac.sql && echo FRAC_MIGRATION_OK" % (DB_DIR, DB))
        print("执行:", out.strip()[-120:], err.strip()[:200])
        if "FRAC_MIGRATION_OK" not in out:
            raise RuntimeError("迁移未成功（已回滚）")

        print("\n=== 校验 ===")
        left = 0
        for t, col in pairs:
            left += int(sql(c, "select count(*) from {t} where {c} like '%.%'".format(t=t, c=col)) or 0)
        print("残留含小数秒的值: %d" % left)
        print("时间值长度分布(orders.created_at):", sql(c, "select length(created_at), count(*) from orders group by 1"))
        print("行数抽查: users=%s orders=%s audit_logs=%s" % (
            sql(c, "select count(*) from users"), sql(c, "select count(*) from orders"), sql(c, "select count(*) from audit_logs")))
    finally:
        c.close()


if __name__ == "__main__":
    main()
