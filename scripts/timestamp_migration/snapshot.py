#!/usr/bin/env python3
"""迁移前后基线快照：用于确认转换只改"表示"、不改"真实时刻"，且无行丢失。

对比原则
--------
* 行数类指标必须完全一致（迁移只改时间文本，不该增删行）。
* "到期订阅"等时间相关清单会变化，这是预期的（原先跨偏移的行被判错），
  因此这里同时输出"真实时刻归一化后"的一致视图供人工核对。
"""
import json
import sys

import paramiko

HOST, USER, PWD = "132.226.1.44", "root", "Sikeming001@"
DB_DIR = "/www/wwwroot/dy.moneyfly.top"


def sql(c, q, t=180):
    cmd = 'cd %s && sqlite3 -readonly cboard.db "%s"' % (DB_DIR, q)
    _, o, e = c.exec_command(cmd, timeout=t)
    out = o.read().decode().strip()
    err = e.read().decode().strip()
    if err and not out:
        return "ERR: " + err[:120]
    return out


def snapshot():
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(HOST, 22, USER, PWD, timeout=20, allow_agent=False, look_for_keys=False)
    try:
        snap = {}
        # 1) 各表行数
        for t in ["users", "orders", "subscriptions", "devices", "audit_logs", "recharge_records",
                  "knowledge_articles", "invite_relations", "balance_logs", "custom_nodes"]:
            snap["rows:" + t] = sql(c, "select count(*) from %s" % t)
        # 2) 关键业务指标（金额/状态，不应受时间迁移影响）
        snap["sum_orders_amount"] = sql(c, "select round(coalesce(sum(amount),0),2) from orders where status='paid'")
        snap["sum_balance"] = sql(c, "select round(coalesce(sum(balance),0),2) from users")
        snap["paid_orders"] = sql(c, "select count(*) from orders where status='paid'")
        # 3) 时间相关视图（迁移后会有合理变化，用于人工核对）
        snap["users_today_str"] = sql(c, "select count(*) from users where created_at >= date('now','+8 hours')")
        snap["orders_today_str"] = sql(c, "select count(*) from orders where created_at >= date('now','+8 hours')")
        snap["expiring_7d_str"] = sql(
            c, "select count(*) from subscriptions where is_active=1 and expire_time between datetime('now','+8 hours') and datetime('now','+8 hours','+7 days')")
        # 4) 抽样：订单 payment_time 与 updated_at 的一致性（迁移后应更一致）
        snap["order_sample"] = sql(c, "select id, amount, status, created_at, updated_at, payment_time from orders order by id desc limit 3")
        snap["user_sample"] = sql(c, "select id, username, created_at from users order by id limit 3")
        return snap
    finally:
        c.close()


if __name__ == "__main__":
    snap = snapshot()
    out = sys.argv[1] if len(sys.argv) > 1 else "/tmp/snapshot.json"
    json.dump(snap, open(out, "w"), ensure_ascii=False, indent=1)
    for k, v in snap.items():
        if not k.endswith("_sample"):
            print("%-24s %s" % (k, v))
