#!/bin/bash
# 从备份导入数据库到本机站点（默认只演练，加 --apply 才真正执行）
#
# 用途：把「老站自动推到 GitHub 的备份」或任意一份 backup_db_*.zip 导入当前站点。
#
# 为什么不能直接 cp 覆盖：
#   1) SQLite 是 WAL 模式，库旁边有 cboard.db-wal / cboard.db-shm；只覆盖主库文件会让
#      SQLite 把**旧 WAL** 应用到新库上 → 直接损坏（本服务器 10-02 那批 cboard.db.corrupt.*
#      大概率就是这么来的）；
#   2) 必须在服务停止时替换，否则写入会丢/串；
#   3) 备份里的 domain_name / 订阅域名等设置是老站的值，覆盖后新站会"认错家门"
#      （本脚本会自动改回你指定的站点域名）；
#   4) 运维口子：Redis 里还有旧缓存，替换后必须 FLUSH，否则页面读到老数据。
#
# 用法：
#   ops/import-db.sh <backup_db_*.zip 路径> [--apply] [--site-domain speedora.top] [--use-backup-env]
#   ops/import-db.sh --from-github latest [--apply] [--site-domain speedora.top]
#
# 不做的事：不导入 uploads/（工单附件、站点 logo 不在数据库备份里，需另外同步）。
set -u

SITE=${SITE_DIR:-$(cd "$(dirname "$0")/.." && pwd)}
DB="$SITE/cboard.db"
BACKUP_DIR=${BACKUP_DIR:-/www/backup/cboard}
APPLY=0
FROM_GITHUB=""
SRC_ZIP=""
SITE_DOMAIN=""
USE_BACKUP_ENV=0
STAMP=$(date +%Y%m%d-%H%M%S)

while [ $# -gt 0 ]; do
  case "$1" in
    --apply) APPLY=1 ;;
    --from-github) shift; FROM_GITHUB="${1:-latest}" ;;
    --site-domain) shift; SITE_DOMAIN="${1:-}" ;;
    --use-backup-env) USE_BACKUP_ENV=1 ;;
    -*) echo "未知参数: $1" >&2; exit 2 ;;
    *) SRC_ZIP="$1" ;;
  esac
  shift
done

log() { echo "[$(date '+%F %T')] $*"; }
step() { echo; echo "=== $* ==="; }
run() { if [ "$APPLY" = "1" ]; then eval "$@"; else echo "  [演练] $*"; fi; }

[ "$APPLY" = "1" ] || log "演练模式（不会改动任何数据）；确认无误后加 --apply 真正执行"

# ---------- ① 准备源备份 ----------
step "① 准备源备份"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

if [ -n "$FROM_GITHUB" ]; then
  TOKEN=$(sqlite3 "$DB" "select value from system_configs where key='backup_github_token' limit 1;" 2>/dev/null)
  OWNER=$(sqlite3 "$DB" "select value from system_configs where key='backup_github_owner' limit 1;" 2>/dev/null)
  REPO=$(sqlite3 "$DB" "select value from system_configs where key='backup_github_repo' limit 1;" 2>/dev/null)
  [ -n "$TOKEN" ] && [ -n "$OWNER" ] && [ -n "$REPO" ] || { log "!! 数据库里没有 GitHub 备份配置（token/owner/repo）"; exit 1; }
  log "从 GitHub $OWNER/$REPO 取最新备份（token 只在服务器本地使用）"
  LATEST=$(curl -s -H "Authorization: token $TOKEN" \
    "https://api.github.com/repos/$OWNER/$REPO/git/trees/main?recursive=1" \
    | python3 -c "
import sys, json
d = json.load(sys.stdin)
files = [x['path'] for x in (d.get('tree') or []) if x.get('path','').endswith('.zip')]
print(sorted(files)[-1] if files else '')
")
  [ -n "$LATEST" ] || { log "!! 仓库里找不到 zip 备份"; exit 1; }
  log "最新备份: $LATEST"
  curl -sL -H "Authorization: token $TOKEN" \
    "https://api.github.com/repos/$OWNER/$REPO/contents/$LATEST?ref=main" \
    -H "Accept: application/vnd.github.raw" -o "$TMPDIR/src.zip" || { log "!! 下载失败"; exit 1; }
  SRC_ZIP="$TMPDIR/src.zip"
fi

[ -n "$SRC_ZIP" ] && [ -f "$SRC_ZIP" ] || { echo "用法: $0 <backup_db_*.zip> 或 --from-github latest [--apply]" >&2; exit 2; }
log "源文件: $SRC_ZIP（$(du -h "$SRC_ZIP" | cut -f1)）"

# ---------- ② 解出并校验源数据库（先验证，后替换） ----------
step "② 解出并校验源数据库"
unzip -o -q "$SRC_ZIP" cboard.db -d "$TMPDIR" || { log "!! zip 里没有 cboard.db"; exit 1; }
SRC_DB="$TMPDIR/cboard.db"
[ -f "$SRC_DB" ] || { log "!! 解压失败"; exit 1; }

QC=$(sqlite3 "$SRC_DB" "PRAGMA quick_check;" 2>/dev/null | head -1)
log "源库 quick_check: ${QC:-空}"
[ "$QC" = "ok" ] || { log "!! 源库完整性检查未通过，已中止（不会覆盖现有数据）"; exit 1; }
SRC_COUNT=$(sqlite3 "$SRC_DB" "select 'users='||(select count(*) from users)||' orders='||(select count(*) from orders)||' subs='||(select count(*) from subscriptions);" 2>/dev/null)
CUR_COUNT=$(sqlite3 "$DB" "select 'users='||(select count(*) from users)||' orders='||(select count(*) from orders)||' subs='||(select count(*) from subscriptions);" 2>/dev/null)
log "源库记录数: $SRC_COUNT"
log "当前站点记录数: $CUR_COUNT"
log "（备份是清洗版，7 天前的日志表会被裁掉：audit_logs/login_history/subscription_logs 等，业务数据完整）"
if [ -f "$TMPDIR/.env" ]; then log "备份内含 .env（SECRET_KEY 可与库保持一致）"; fi

# ---------- ③ 回滚快照（当前库） ----------
step "③ 为当前库做回滚快照"
ROLLBACK="$BACKUP_DIR/pre-import-$STAMP.db.gz"
run "mkdir -p '$BACKUP_DIR'"
run "sqlite3 '$DB' \"VACUUM INTO '$TMPDIR/rollback.db';\" && gzip -9 -c '$TMPDIR/rollback.db' > '$ROLLBACK'"
log "回滚副本将写入: $ROLLBACK"

# ---------- ④ 停服 → 替换 → 清 WAL ----------
step "④ 停服、替换数据库、清理 WAL/SHM"
run "systemctl stop cboard"
run "cp -a '$DB' '$DB.pre-import-$STAMP'"
run "install -m 640 '$SRC_DB' '$DB'"
# 关键：删掉旧 WAL/SHM，否则 SQLite 会把旧日志应用到新库上，直接损坏
run "rm -f '$DB-wal' '$DB-shm'"
run "chown root:root '$DB' 2>/dev/null || true"
log "已删除 cboard.db-wal / cboard.db-shm（这一步不能省）"

# ---------- ⑤ 可选：用备份里的 .env ----------
if [ "$USE_BACKUP_ENV" = "1" ] && [ -f "$TMPDIR/.env" ]; then
  step "⑤ 用备份里的 .env 覆盖（保证 SECRET_KEY 与库同源）"
  run "cp -a '$SITE/.env' '$SITE/.env.pre-import-$STAMP'"
  run "install -m 600 '$TMPDIR/.env' '$SITE/.env'"
else
  step "⑤ 保留当前 .env（默认）"
  log "如需让 SECRET_KEY 与来源库同源，加 --use-backup-env"
fi

# ---------- ⑥ 恢复本站域名设置（否则会认成老站的域名） ----------
step "⑥ 恢复站点域名设置"
if [ -n "$SITE_DOMAIN" ]; then
  run "sqlite3 '$DB' \"update system_configs set value='$SITE_DOMAIN' where key='domain_name';\""
  run "sqlite3 '$DB' \"update system_configs set value='https://$SITE_DOMAIN' where key='subscription_domain';\""
  run "sqlite3 '$DB' \"update system_configs set value='' where key='subscription_backup_domains';\""
  log "已把 domain_name / subscription_domain 改回 $SITE_DOMAIN，并清空备用订阅域名（老站的域名不该留在这台机器上）"
  log "提示：订阅域名池的其它记录在「系统设置 → 订阅域名池」里重跑一次「一键配置/修复」即可"
else
  log "!! 未指定 --site-domain：导入后 domain_name 会变成来源站的域名"
  log "   建议：$0 <zip> --apply --site-domain <你的域名>"
fi

# ---------- ⑦ 起服 + 清缓存 + 验收 ----------
step "⑦ 启动服务、清理缓存、验收"
run "systemctl start cboard"
if [ "$APPLY" = "1" ]; then
  sleep 6
  log "cboard: $(systemctl is-active cboard)"
  log "quick_check: $(sqlite3 "$DB" 'PRAGMA quick_check;' | head -1)"
  log "记录数: $(sqlite3 "$DB" "select 'users='||(select count(*) from users)||' orders='||(select count(*) from orders)||' subs='||(select count(*) from subscriptions);")"
  log "健康检查: $(curl -s -m 8 -o /dev/null -w '%{http_code}' http://127.0.0.1:8000/health)"
  if command -v redis-cli >/dev/null 2>&1; then
    redis-cli FLUSHALL >/dev/null 2>&1 && log "Redis 缓存已清空（避免读到旧数据）"
  fi
  log "站点域名: $(sqlite3 "$DB" "select value from system_configs where key='domain_name';")"
else
  log "（演练结束。真正执行会：停服 → 备份当前库 → 替换 → 删 WAL → 恢复域名 → 起服 → 清 Redis → 打印验收）"
fi

echo
log "完成。回滚方法：停服后把 $DB.pre-import-$STAMP 换回 cboard.db，并删除 -wal/-shm 再起服。"
