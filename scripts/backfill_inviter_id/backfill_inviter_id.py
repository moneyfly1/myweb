#!/usr/bin/env python3
"""回填 registration_logs.inviter_id（历史"邀请码注册"记录的邀请人）

背景
----
注册日志的"邀请人"列取自 registration_logs.inviter_id，该值原本由
users.invited_by 推导；但 users.invited_by 全仓从未被写入过，
真正的邀请关系记录在 invite_relations 表，于是历史记录的 inviter_id 全是 NULL，
日志里"邀请人"一列从未显示过内容。

新版代码已改为直接从 invite_relations 取邀请人（见 handlers/auth.go
processInviteCode 的返回值），本脚本只负责把**历史记录**按同一口径补齐。

回填规则（只改 inviter_id 一个字段）
------------------------------------
    registration_logs.user_id = invite_relations.invitee_id
    且 status = 'success' 且 inviter_id IS NULL

无法匹配的记录（邀请码注册但 invite_relations 无对应关系）保持原样，
不做任何猜测。

用法
----
    python3 backfill_inviter_id.py --db /www/wwwroot/dy.moneyfly.top/cboard.db
        # 预览：打印将写入的前后对照，不写盘

    python3 backfill_inviter_id.py --db ... --apply
        # 执行：先做 .backup 备份 + integrity_check，再在事务内更新，
        #       最后复核"仅目标行发生变化"
"""

import argparse
import datetime as dt
import os
import sqlite3
import sys

SELECT_CANDIDATES = """
SELECT r.id            AS log_id,
       r.username      AS username,
       r.user_id       AS user_id,
       r.register_source AS source,
       r.inviter_id    AS old_inviter_id,
       ir.inviter_id   AS new_inviter_id,
       u.username      AS inviter_name
  FROM registration_logs r
  JOIN invite_relations ir ON ir.invitee_id = r.user_id
  LEFT JOIN users u ON u.id = ir.inviter_id
 WHERE r.status = 'success'
   AND r.inviter_id IS NULL
 ORDER BY r.id
"""

COUNT_SAME_MATCH = """
SELECT COUNT(*)
  FROM registration_logs r
  JOIN invite_relations ir ON ir.invitee_id = r.user_id
 WHERE r.status = 'success'
   AND r.inviter_id IS NULL
"""

# 匹配必须唯一：一个 invitee 只能对应一条邀请关系，否则拒绝执行
COUNT_AMBIGUOUS = """
SELECT COUNT(*) FROM (
    SELECT invitee_id FROM invite_relations GROUP BY invitee_id HAVING COUNT(*) > 1
)
"""

UPDATE_STATEMENT = """
UPDATE registration_logs
   SET inviter_id = (SELECT ir.inviter_id
                       FROM invite_relations ir
                      WHERE ir.invitee_id = registration_logs.user_id)
 WHERE status = 'success'
   AND inviter_id IS NULL
   AND EXISTS (SELECT 1 FROM invite_relations ir2
                WHERE ir2.invitee_id = registration_logs.user_id)
"""


def snapshot(conn):
    cur = conn.cursor()
    has_inviter = cur.execute(
        "SELECT COUNT(*) FROM registration_logs WHERE inviter_id IS NOT NULL"
    ).fetchone()[0]
    total = cur.execute("SELECT COUNT(*) FROM registration_logs").fetchone()[0]
    null_inviter = cur.execute(
        "SELECT COUNT(*) FROM registration_logs WHERE inviter_id IS NULL"
    ).fetchone()[0]
    return {"total": total, "has_inviter": has_inviter, "null_inviter": null_inviter}


def main():
    ap = argparse.ArgumentParser(description="回填 registration_logs.inviter_id")
    ap.add_argument("--db", required=True, help="SQLite 数据库路径")
    ap.add_argument("--apply", action="store_true", help="实际写盘（缺省仅预览）")
    args = ap.parse_args()

    if not os.path.isfile(args.db):
        print(f"数据库不存在: {args.db}", file=sys.stderr)
        return 2

    conn = sqlite3.connect(args.db, timeout=30)
    conn.row_factory = sqlite3.Row
    cur = conn.cursor()

    ambiguous = cur.execute(COUNT_AMBIGUOUS).fetchone()[0]
    expected = cur.execute(COUNT_SAME_MATCH).fetchone()[0]
    rows = cur.execute(SELECT_CANDIDATES).fetchall()

    print(f"数据库: {args.db}")
    print(f"待回填记录: {len(rows)} 条（与规则匹配数 {expected} 条一致: "
          f"{'是' if len(rows) == expected else '否'}）")
    print(f"匹配歧义（同一 invitee 多条关系）: {ambiguous} 条")
    print()
    print(f"{'日志ID':>7}  {'用户名':<12} {'用户ID':>7} {'来源':<12} "
          f"{'邀请人ID':>9}  邀请人")
    for r in rows:
        print(f"{r['log_id']:>7}  {(r['username'] or '(空)'):<12} {r['user_id']:>7} "
              f"{(r['source'] or '(空)'):<12} {r['new_inviter_id']:>9}  "
              f"{r['inviter_name'] or '(未知)'}")

    if ambiguous > 0:
        print("\n存在歧义匹配，拒绝执行。请人工确认 invite_relations 后重试。", file=sys.stderr)
        return 3
    if not rows:
        print("\n没有需要回填的记录。")
        return 0

    if not args.apply:
        print("\n[预览模式] 未写盘。确认无误后加 --apply 执行。")
        return 0

    before = snapshot(conn)

    backup = os.path.join(
        os.path.dirname(os.path.abspath(args.db)),
        f"backup-inviter-backfill-{dt.datetime.now():%Y%m%d-%H%M%S}.db",
    )
    cur.execute("VACUUM INTO ?", (backup,))
    print(f"\n已备份: {backup}")
    integrity = sqlite3.connect(backup).execute("PRAGMA integrity_check").fetchone()[0]
    print(f"备份完整性: {integrity}")
    if integrity != "ok":
        print("备份完整性检查未通过，终止。", file=sys.stderr)
        return 4

    with conn:  # 事务：全部成功或全部回滚
        cur.execute(UPDATE_STATEMENT)
        changed = cur.rowcount

    after = snapshot(conn)
    print(f"\n已更新 {changed} 行（预告 {len(rows)} 行）")
    print(f"回填前: 有邀请人 {before['has_inviter']} / 空 {before['null_inviter']} / 共 {before['total']}")
    print(f"回填后: 有邀请人 {after['has_inviter']} / 空 {after['null_inviter']} / 共 {after['total']}")

    problems = []
    if changed != len(rows):
        problems.append(f"更新行数 {changed} 与预告 {len(rows)} 不一致")
    if after["total"] != before["total"]:
        problems.append("总行数发生变化（不应发生）")
    if after["has_inviter"] != before["has_inviter"] + changed:
        problems.append("有邀请人的行数增量与更新行数不一致")

    print("\n回填后明细:")
    for r in cur.execute(
        """SELECT r.id, r.username, r.inviter_id, u.username AS inviter_name
             FROM registration_logs r LEFT JOIN users u ON u.id = r.inviter_id
            WHERE r.inviter_id IS NOT NULL ORDER BY r.id"""
    ):
        print(f"  日志ID {r[0]:>5}  {r[1]:<12} → 邀请人 {r[2]} ({r[3] or '未知'})")

    if problems:
        print("\n复核发现异常:", "; ".join(problems), file=sys.stderr)
        return 5

    print("\n复核通过：仅目标记录发生变化。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
