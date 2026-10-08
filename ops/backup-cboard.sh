#!/bin/bash
# CBoard 备份脚本：数据库一致性快照 + 关键配置归档（本地轮转）
#
# 设计要点：
#   - 用 SQLite 的 VACUUM INTO 做快照：在线一致、不锁库、不需要停机（实测 183MB/0.5s）。
#   - 每天一份日备份（保留 7 份），每周日额外留一份周备份（保留 4 份）。
#   - 配置卷（.env / nginx 站点配置 / systemd 单元 / crontab）体积很小，每天一并归档，
#     它是「换台机器恢复」时最容易漏掉、也最不可再生的东西。
#   - 只读源数据，不改动线上任何文件（除自身输出目录）。
#
# 安装（root）：
#   crontab -e 加入：
#     20 4 * * *  /www/wwwroot/dy.moneyfly.top/ops/backup-cboard.sh >> /var/log/cboard-backup.log 2>&1
#
# 异地：见 ops/RESTORE.md「把备份送出这台机器」一节。
set -u

SITE=${SITE_DIR:-/www/wwwroot/dy.moneyfly.top}
DEST=${BACKUP_DIR:-/www/backup/cboard}
KEEP_DAILY=${KEEP_DAILY:-7}
KEEP_WEEKLY=${KEEP_WEEKLY:-4}
STAMP=$(date +%Y%m%d-%H%M%S)
DAY=$(date +%u)

log() { echo "[$(date '+%F %T')] $*"; }

mkdir -p "$DEST/daily" "$DEST/weekly" "$DEST/config" || exit 1

# ---------- ① 数据库一致性快照 ----------
DB="$SITE/cboard.db"
[ -f "$DB" ] || { log "找不到数据库 $DB"; exit 1; }

TMP="$DEST/daily/cboard-$STAMP.db"
if ! sqlite3 "$DB" "VACUUM INTO '$TMP';" 2>/dev/null; then
  log "!! VACUUM INTO 失败（数据库可能被独占或磁盘满）"
  exit 1
fi
if ! gzip -9 "$TMP"; then
  log "!! gzip 失败"; rm -f "$TMP"; exit 1
fi
GZ="$TMP.gz"

# 校验：能完整解压、能打开、关键表可读、记录数与线上同量级
if ! gzip -t "$GZ" 2>/dev/null; then
  log "!! 备份压缩包损坏：$GZ"; exit 1
fi
CHECK=$(gzip -dc "$GZ" 2>/dev/null | sqlite3 "file:/dev/stdin?mode=ro" \
        "select (select count(*) from users)||'/'||(select count(*) from orders)||'/'||(select count(*) from subscriptions);" 2>/dev/null || echo "")
LIVE=$(sqlite3 "$DB" "select (select count(*) from users)||'/'||(select count(*) from orders)||'/'||(select count(*) from subscriptions);" 2>/dev/null || echo "")
if [ -z "$CHECK" ]; then
  log "!! 备份无法读取校验（保留文件待人工确认）：$GZ"
elif [ "$CHECK" != "$LIVE" ]; then
  log "!! 备份记录数与线上不一致（备份=$CHECK 线上=$LIVE）：$GZ"
else
  log "备份成功 $(du -h "$GZ" | cut -f1) 记录数 $CHECK（线上一致）→ $GZ"
fi

# 周日额外留一份周备份
if [ "$DAY" = "7" ]; then
  cp -p "$GZ" "$DEST/weekly/cboard-week-$STAMP.db.gz"
  log "已生成周备份 cboard-week-$STAMP.db.gz"
fi

# ---------- ② 关键配置归档（换机恢复必需）----------
CFG="$DEST/config/cboard-config-$STAMP.tar.gz"
TARLIST=()
[ -f "$SITE/.env" ] && TARLIST+=(".env")
[ -d /www/server/panel/vhost/nginx ] && TARLIST+=(vhost-nginx)
[ -d /etc/nginx/conf.d ] && TARLIST+=(etc-nginx-conf.d)
[ -f /etc/systemd/system/cboard.service ] && TARLIST+=(cboard.service)
[ -f /etc/systemd/system/cboard-v2.service ] && TARLIST+=(cboard-v2.service)
[ -d /etc/letsencrypt/renewal ] && TARLIST+=(letsencrypt-renewal)
if [ "${#TARLIST[@]}" -gt 0 ]; then
  ( cd / && tar czf "$CFG" \
      --transform 's#^#root/#' \
      "$SITE/.env" \
      /www/server/panel/vhost/nginx \
      /etc/nginx/conf.d \
      /etc/systemd/system/cboard.service \
      /etc/systemd/system/cboard-v2.service \
      /etc/letsencrypt/renewal \
      2>/dev/null )
  crontab -l >"$DEST/config/crontab-$STAMP.txt" 2>/dev/null || true
  log "配置归档 → $CFG（另存 crontab-$STAMP.txt）"
fi

# ---------- ③ 轮转 ----------
prune() { # $1=目录 $2=保留份数 $3=匹配前缀
  local dir=$1 keep=$2
  ls -1t "$dir"/cboard-*.gz 2>/dev/null | tail -n +$((keep + 1)) | while read -r f; do rm -f "$f"; done
}
prune "$DEST/daily" "$KEEP_DAILY"
prune "$DEST/weekly" "$KEEP_WEEKLY"
ls -1t "$DEST/config"/cboard-config-*.tar.gz 2>/dev/null | tail -n +8 | while read -r f; do rm -f "$f"; done
ls -1t "$DEST/config"/crontab-*.txt 2>/dev/null | tail -n +8 | while read -r f; do rm -f "$f"; done

log "当前保留：日备份 $(ls -1 "$DEST/daily"/cboard-*.gz 2>/dev/null | wc -l) 份、周备份 $(ls -1 "$DEST/weekly"/cboard-*.gz 2>/dev/null | wc -l) 份、占用 $(du -sh "$DEST" 2>/dev/null | cut -f1)"
exit 0
