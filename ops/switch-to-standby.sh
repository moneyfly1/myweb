#!/bin/bash
# 从主站切换到备机（L2 温备），或在备机上执行「提升为主站」的动作
#
# 用法（在主站执行）：
#   ops/switch-to-standby.sh <user@备机>             # 最后一次同步 + 提示你去 Cloudflare 改回源
#   ops/switch-to-standby.sh <user@备机> --promote   # 额外：在备机上就位数据库并启动服务
#
# 说明：域名不变（Cloudflare 只改回源 IP），因此客户端订阅、支付回调、节点回传都无需改动。
#       本脚本故意把「改 DNS」这一步留给人来做，并在最后打印验收清单。
set -u

TARGET=${1:-}
DO_PROMOTE=${2:-}
if [ -z "$TARGET" ]; then
  echo "用法: $0 <user@备机> [--promote]" >&2
  exit 2
fi

SITE=${SITE_DIR:-/www/wwwroot/dy.moneyfly.top}
KEY=${STANDBY_SSH_KEY:-/root/.ssh/id_ed25519}
SSH_OPTS=(-i "$KEY" -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=15)
REMOTE_SITE=${REMOTE_SITE:-/www/wwwroot/dy.moneyfly.top}

step() { echo; echo "=== $* ==="; }

step "① 切换前最后一次同步（保证零丢数据）"
"$SITE/ops/standby-sync.sh" "$TARGET" --final || { echo "!! 同步失败，先别切换，排查备机连通性"; exit 1; }

step "② 主站写入会在这之后停止 —— 确认要切换再继续"
echo "  切到备机后，主站若还活着请把 cboard 服务停掉，避免两边同时写导致数据分叉："
echo "    systemctl stop cboard"
echo "  本脚本不自动停主站服务（保留现场，方便快速回滚）。"

if [ "$DO_PROMOTE" = "--promote" ]; then
  step "③ 在备机上就位数据库并启动服务"
  ssh "${SSH_OPTS[@]}" "$TARGET" "set -e
    cd '$REMOTE_SITE'
    [ -f cboard.db.standby ] || { echo '  !! 备机上没有 cboard.db.standby'; exit 1; }
    systemctl stop cboard 2>/dev/null || true
    cp -a cboard.db cboard.db.pre-cutover.\$(date +%Y%m%d-%H%M%S) 2>/dev/null || true
    mv cboard.db.standby cboard.db
    chown www:www cboard.db 2>/dev/null || true
    echo -n '  quick_check: '; sqlite3 cboard.db 'PRAGMA quick_check;' | head -1
    systemctl start cboard
    sleep 3
    echo -n '  cboard: '; systemctl is-active cboard
    echo -n '  本机自检: '; curl -s -o /dev/null -k -m 8 -w '%{http_code}\n' https://127.0.0.1/ || true
    echo -n '  API: '; curl -s -o /dev/null -m 8 -w '%{http_code}\n' http://127.0.0.1:8000/api/v1/health || true
  "
fi

step "④ 手动一步：Cloudflare 把回源指向备机"
cat <<'EOF'
  Cloudflare 控制台 → DNS：
    - dy.moneyfly.club   （客户端订阅主力域名）A 记录 → 备机 IP
    - dy.moneyfly.top
    - 其它订阅用域名（如 sub.moneyfly.dpdns.org）如有同源记录一并改
  建议先切成「DNS only（灰云）」验证证书与访客可达，再切回「Proxied（小黄云）」。

  如果备机是 Cloudflare Origin 证书：证书与主站一致，可直接开小黄云；
  如果是 certbot 签的：确认备机 80 端口可被 Let's Encrypt 校验。
EOF

step "⑤ 验收清单"
cat <<'EOF'
  [ ] 官网首页 200，能登录（老账号密码有效 → SECRET_KEY 一致）
  [ ] 后台「用户列表」记录数与主站一致
  [ ] 客户端订阅可拉取：
      curl -A clash-verge/v2.0.0 -o /dev/null -w '%{http_code}\n' https://dy.moneyfly.club/api/v1/subscribe/<token>
  [ ] 专线节点/自建节点仍在线（节点回传走域名，无需改）
  [ ] 支付下单 → 回调地址仍是域名，验证一笔小额
  [ ] 通知/邮件能发出（系统设置里的密钥来自 .env，已随 --final 同步）

  回滚：把 Cloudflare 回源改回主站 IP 即可。
  注意：若备机已产生新写入，回滚前先把备机 cboard.db 拉回主站，避免丢数据。
EOF
