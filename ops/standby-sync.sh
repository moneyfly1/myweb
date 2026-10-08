#!/bin/bash
# 把主站的一致性数据库快照推送到备机（L2 温备用）
#
# 用法：
#   ops/standby-sync.sh <user@备机>               # 常规同步（建议 cron 每 10 分钟）
#   ops/standby-sync.sh <user@备机> --final       # 切换前最后一次：额外同步 .env 与关键上传目录
#
# 设计：
#   - 源库只用 VACUUM INTO 读快照（0.5s、不锁库，不影响线上）；
#   - gzip -9 后约 24 MB，比 rsync 全量更省备机流量；
#   - 传输后**在备机原地校验**（quick_check + 记录数比对），校验通过才原子改名就位，
#     避免切换时才发现备份是坏的；
#   - 用 flock 防止上一次没跑完就重入。
set -u

TARGET=${1:-}
MODE=${2:-}
if [ -z "$TARGET" ]; then
  echo "用法: $0 <user@备机> [--final]" >&2
  exit 2
fi

SITE=${SITE_DIR:-/www/wwwroot/dy.moneyfly.top}
KEY=${STANDBY_SSH_KEY:-/root/.ssh/id_ed25519}
LOCK=/var/lock/cboard-standby-sync.lock
STAMP=$(date +%Y%m%d-%H%M%S)

log() { echo "[$(date '+%F %T')] $*"; }

exec 9>"$LOCK" || exit 1
if ! flock -n 9; then
  log "上一次同步仍在进行，跳过本次"
  exit 0
fi

# ---------- ① 主站：一致性快照 ----------
SRC="$SITE/cboard.db"
[ -f "$SRC" ] || { log "找不到数据库 $SRC"; exit 1; }

TMP="/tmp/cboard-standby-$STAMP.db"
if ! sqlite3 "$SRC" "VACUUM INTO '$TMP';" 2>/dev/null; then
  log "!! VACUUM INTO 失败"; exit 1
fi
if ! gzip -9 "$TMP"; then
  log "!! 压缩失败"; rm -f "$TMP"; exit 1
fi
GZ="$TMP.gz"
LIVE=$(sqlite3 "$SRC" "select (select count(*) from users)||'/'||(select count(*) from orders);" 2>/dev/null || echo "?")

# ---------- ② 传输（带重试） ----------
SSH_OPTS=(-i "$KEY" -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=15)
ok=0
for attempt in 1 2 3; do
  if scp -q "${SSH_OPTS[@]}" "$GZ" "$TARGET:/tmp/cboard-standby.db.gz.new"; then ok=1; break; fi
  log "第 $attempt 次传输失败，2 秒后重试"
  sleep 2
done
if [ "$ok" != "1" ]; then
  log "!! 传输失败（备机不可达？）: $TARGET"
  rm -f "$TMP" "$GZ"; exit 1
fi
rm -f "$TMP" "$GZ"

# ---------- ③ 备机：校验后原子就位 ----------
REMOTE_SITE=${REMOTE_SITE:-/www/wwwroot/dy.moneyfly.top}
read -r -d '' REMOTE_SCRIPT <<EOF
set -e
cd "$REMOTE_SITE" || exit 1
gunzip -c /tmp/cboard-standby.db.gz.new > cboard.db.standby.tmp
chk=\$(sqlite3 cboard.db.standby.tmp "PRAGMA quick_check;" 2>/dev/null | head -1)
[ "\$chk" = "ok" ] || { echo "VERIFY_FAIL:\$chk"; rm -f cboard.db.standby.tmp /tmp/cboard-standby.db.gz.new; exit 1; }
cnt=\$(sqlite3 cboard.db.standby.tmp "select (select count(*) from users)||'/'||(select count(*) from orders);")
mv cboard.db.standby.tmp cboard.db.standby
chown www:www cboard.db.standby 2>/dev/null || true
rm -f /tmp/cboard-standby.db.gz.new
echo "OK:\$cnt"
EOF

RESULT=$(ssh "${SSH_OPTS[@]}" "$TARGET" "$REMOTE_SCRIPT" 2>&1 | tail -3)
if echo "$RESULT" | grep -q "^OK:"; then
  REMOTE_CNT=$(echo "$RESULT" | sed -n 's/^OK://p')
  if [ "$REMOTE_CNT" = "$LIVE" ]; then
    log "同步成功 备机记录数 $REMOTE_CNT（与主站一致）→ $TARGET"
  else
    log "!! 备机记录数 $REMOTE_CNT ≠ 主站 $LIVE（已就位但请人工确认）"
  fi
else
  log "!! 备机校验失败: $RESULT"
  exit 1
fi

# ---------- ④ --final：同步配置文件与关键上传 ----------
if [ "$MODE" = "--final" ]; then
  PKG="/tmp/cboard-assets-$STAMP.tar.gz"
  ( cd "$SITE" && tar czf "$PKG" .env uploads/config uploads/tickets uploads/repo_sync 2>/dev/null )
  if [ -f "$PKG" ] && scp -q "${SSH_OPTS[@]}" "$PKG" "$TARGET:/tmp/cboard-assets.tar.gz"; then
    ssh "${SSH_OPTS[@]}" "$TARGET" "cd '$REMOTE_SITE' && tar xzf /tmp/cboard-assets.tar.gz && rm -f /tmp/cboard-assets.tar.gz" \
      && log "--final：已同步 .env 与 uploads/{config,tickets,repo_sync}"
  else
    log "!! --final 资源同步失败"
  fi
  rm -f "$PKG"
fi

exit 0
