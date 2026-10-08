#!/bin/bash
# CBoard 服务器自愈守护（幂等，可反复执行）
#
# 用途：解决「重启后网站能不能自己起来」以及「运行中进程挂了没人管」的问题。
#   - 宝塔 nginx（1.28）没有 systemd 自启单元，系统自带的 nginx.service 又因配置
#     （http2 on; 需要 1.25+）解析失败；因此这里每分钟兜底一次：443 没监听就拉起来。
#   - cboard.service 虽有 Restart=always，仍兜底一次失败/卡死状态。
#   - 附带本机 HTTPS 自检与磁盘余量告警（只记录，不改配置）。
#
# 安装（root）：
#   crontab -e 加入：
#     * * * * *  flock -n /var/lock/cboard-selfheal.lock /www/wwwroot/dy.moneyfly.top/ops/server-selfheal.sh
#     @reboot    flock -n /var/lock/cboard-selfheal.lock /www/wwwroot/dy.moneyfly.top/ops/server-selfheal.sh
#
# 只做「没起来就拉起来」，不修改任何配置、不碰数据库。
set -u

LOG=${SELFHEAL_LOG:-/var/log/cboard-selfheal.log}
NGINX_BIN=${NGINX_BIN:-/www/server/nginx/sbin/nginx}
SERVICE=${CBOARD_SERVICE:-cboard}
SITE_URL=${SELFHEAL_URL:-https://127.0.0.1/}

log() { echo "[$(date '+%F %T')] $*" >>"$LOG" 2>/dev/null || true; }

# 日志自截断，避免无限增长（保留最近约 2000 行）
if [ -f "$LOG" ] && [ "$(stat -c%s "$LOG" 2>/dev/null || echo 0)" -gt 5242880 ]; then
  tail -2000 "$LOG" >"$LOG.tmp" 2>/dev/null && mv "$LOG.tmp" "$LOG"
fi

# ① nginx：443 未监听时尝试拉起（先自检配置，避免把坏配置反复拉起刷屏）
if ! ss -ltn 2>/dev/null | grep -q ':443 '; then
  if "$NGINX_BIN" -t >/dev/null 2>&1; then
    if "$NGINX_BIN" >/dev/null 2>&1; then
      log "检测到 443 未监听 → 已启动宝塔 nginx"
    else
      log "!! 443 未监听且 nginx 启动失败（需人工处理）"
    fi
  else
    log "!! 443 未监听，且 nginx -t 配置自检失败 → 未启动（需人工处理）"
  fi
fi

# ② 面板后端服务：不在 active 状态就重启（systemd 的 Restart=always 之外的兜底）
if command -v systemctl >/dev/null 2>&1 && ! systemctl is-active --quiet "$SERVICE"; then
  if systemctl restart "$SERVICE" >/dev/null 2>&1; then
    log "检测到 $SERVICE 未运行 → 已重启"
  else
    log "!! $SERVICE 未运行且重启失败"
  fi
fi

# ③ 本机 HTTPS 自检（失败只记录；连续失败说明链路有问题，便于事后定位）
code=$(curl -s -o /dev/null -k -m 8 -w '%{http_code}' "$SITE_URL" 2>/dev/null || echo 000)
if [ "$code" != "200" ]; then
  log "!! 本机自检未通过：HTTP $code ($SITE_URL)"
fi

# ④ 磁盘余量告警（≥90% 记录，避免写满后数据库损坏）
use=$(df --output=pcent / 2>/dev/null | tail -1 | tr -dc '0-9')
if [ -n "${use:-}" ] && [ "$use" -ge 90 ]; then
  log "!! 根分区已用 ${use}%，请及时清理（数据库写满会导致损坏）"
fi

exit 0
