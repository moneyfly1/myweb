#!/usr/bin/env python3
"""扫描 VPS 上 cboard.db 的时间列，统计需要归一化的规模。

判定规则（正则精确匹配，避免把"无偏移但带秒"误判为有偏移）：
  含偏移   ：以 [+-]HH:MM 结尾
  无偏移   ：有值但不是以 [+-]HH:MM 结尾（SQLite datetime('now') 等写入的 UTC 墙钟）
  非+08:00 ：有值但不以 +08:00 结尾
"""
import json
import sys

import paramiko

HOST, USER, PWD = "132.226.1.44", "root", "Sikeming001@"
DB_DIR = "/www/wwwroot/dy.moneyfly.top"


def connect():
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(HOST, 22, USER, PWD, timeout=20, allow_agent=False, look_for_keys=False)
    return c


def run(c, q, t=120):
    cmd = 'cd %s && sqlite3 -readonly cboard.db "%s"' % (DB_DIR, q)
    _, o, e = c.exec_command(cmd, timeout=t)
    return o.read().decode().strip() or e.read().decode().strip()


def main():
    pairs = json.load(open(sys.argv[1]))
    c = connect()
    rows = []
    try:
        for t, col in pairs:
            q = (
                "select count(*), "
                "sum(case when {c} glob '*[+-][0-9][0-9]:[0-9][0-9]' then 1 else 0 end), "
                "sum(case when {c} is not null and {c}<>'' and {c} not glob '*[+-][0-9][0-9]:[0-9][0-9]' then 1 else 0 end), "
                "sum(case when {c} is not null and {c}<>'' and {c} not like '%+08:00' then 1 else 0 end) "
                "from {t} where {c} is not null and {c}<>''"
            ).format(c=col, t=t)
            r = run(c, q)
            if "|" in r and "Error" not in r:
                tot, withoff, without, nonbj = r.split("|")
                without, nonbj = int(without or 0), int(nonbj or 0)
                if without > 0 or nonbj > 0:
                    rows.append((t, col, int(tot or 0), int(withoff or 0), without, nonbj))
    finally:
        c.close()

    print("%-46s %8s %8s %8s %10s" % ("表.列", "有值", "含偏移", "无偏移", "非+08:00"))
    tw = tn = 0
    for t, col, tot, w, wo, nb in sorted(rows, key=lambda x: -(x[4] + x[5])):
        print("%-46s %8d %8d %8d %10d" % (t + "." + col, tot, w, wo, nb))
        tw += wo
        tn += nb
    print()
    print("合计：无偏移 %d 行；非 +08:00 %d 行" % (tw, tn))
    json.dump(
        [{"table": t, "column": col, "total": tot, "with_offset": w, "without_offset": wo, "non_beijing": nb}
         for t, col, tot, w, wo, nb in rows],
        open("/tmp/need_fix.json", "w"), ensure_ascii=False, indent=1,
    )


if __name__ == "__main__":
    main()
