# 回填 registration_logs.inviter_id

一次性数据修复工具，用于补齐历史「邀请码注册」记录里缺失的邀请人。

## 背景

注册日志（后台 → 日志管理 → 注册日志）的 **邀请人** 列取自
`registration_logs.inviter_id`，该值原本由 `users.invited_by` 推导，
但 `users.invited_by` 全仓从未被写入过（只有读取方，没有写入方），
真正的邀请关系记录在 `invite_relations` 表里。

结果：`register_source = 'invite_code'` 的历史记录 `inviter_id` 全为 NULL，
日志里"邀请人"一列从未显示过内容。

代码侧已修好（`handlers/auth.go` 的 `processInviteCode` 现在返回邀请人 ID，
注册日志直接采用 `invite_relations` 的邀请人），本工具只负责**补齐历史记录**。

## 回填规则

只改 `inviter_id` 一个字段，匹配条件：

```
registration_logs.user_id = invite_relations.invitee_id
且 status = 'success'
且 inviter_id IS NULL
```

- 匹配必须唯一（同一 `invitee_id` 有多条邀请关系时脚本拒绝执行）；
- 邀请码注册但在 `invite_relations` 中找不到对应关系的记录**保持原样**，不做猜测。

## 用法

```bash
# 预览（默认，不写盘）：打印将写入的前后对照
python3 backfill_inviter_id.py --db /path/to/cboard.db

# 执行：先备份 + 备份完整性校验，再在事务内更新，最后复核"仅目标行变化"
python3 backfill_inviter_id.py --db /path/to/cboard.db --apply
```

`--apply` 会在数据库同目录生成
`backup-inviter-backfill-<时间戳>.db`（更新前的快照，可用于回滚），
并校验 `PRAGMA integrity_check`，未通过则终止。

## 生产执行记录（2026-09-18）

```
待回填记录: 3 条（与规则匹配数 3 条一致: 是）
匹配歧义: 0 条
  日志ID 1176  danding2    用户1430 → 邀请人 807 (nio253)
  日志ID 1226  somniferum  用户1475 → 邀请人 286 (664156937)
  日志ID 1245  julia       用户1488 → 邀请人 286 (664156937)

已更新 3 行；回填前 有邀请人 0 / 空 122 / 共 122
             回填后 有邀请人 3 / 空 119 / 共 122
备份: /www/wwwroot/dy.moneyfly.top/backup-inviter-backfill-20260918-011548.db（另存 /root/ 一份）
复核: 仅目标记录变化；服务 active；integrity_check ok；用户 1474 / 订单 502 未变
```

另有 2 条 `invite_code` 注册记录在 `invite_relations` 中没有对应关系，
无法回填（脚本会跳过，不影响其余记录）。
