#!/bin/bash
# ============================================================
# bt-deploy.sh —— 宝塔面板专用入口（薄封装）
# ============================================================
# 为什么只是一个"薄封装"：
#   历史上 bt-deploy.sh 是 install.sh 的一份完整拷贝，两份代码各自演进后开始漂移
#   （宝塔版曾缺少 CGO/Node 版本/nginx -t/uploads 反代/ACME 放行段等，
#    而 install.sh 曾不会自举 Go/Node/Nginx）。现在只保留**一套实现**：
#       install.sh  —— 唯一的实现，会自动识别环境（宝塔 / 系统 nginx 都能正确定位）
#       bt-deploy.sh—— 本文件，只做宝塔环境的检查与提示，然后转交 install.sh
#   这样两个入口的行为永远一致，也不会再出现"改了一个忘了另一个"。
#
# 用法：与 install.sh 完全一致
#   cd /www/wwwroot/你的域名 && bash bt-deploy.sh          # 进菜单
#   printf '1\n' | bash bt-deploy.sh                       # 直接跑「一键全自动部署」
#   CERT_MANAGER=panel bash bt-deploy.sh                   # 证书交给宝塔面板管理
# ============================================================

set +e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; CYAN='\033[0;36m'; NC='\033[0m'

# 找到真正要执行的 install.sh：优先同目录，其次当前目录（兼容脚本被放到别处执行的情况）
TARGET=""
if [[ -f "${SCRIPT_DIR}/install.sh" ]]; then
    TARGET="${SCRIPT_DIR}/install.sh"
elif [[ -f "./install.sh" ]]; then
    TARGET="./install.sh"
fi

if [[ -z "$TARGET" ]]; then
    echo -e "${RED}未找到 install.sh${NC}"
    echo "bt-deploy.sh 是 install.sh 的宝塔入口，两者必须放在同一个项目目录下（例如 /www/wwwroot/你的域名）。"
    echo "请先克隆仓库：git clone https://github.com/moneyfly1/myweb.git /www/wwwroot/你的域名"
    exit 1
fi

# 宝塔环境检查（只提示，不阻断：没有宝塔也能正常部署，install.sh 会按系统 nginx 处理）
if [[ -d /www/server/panel ]]; then
    echo -e "${GREEN}✅ 检测到宝塔面板${NC}"
    if [[ -x /www/server/nginx/sbin/nginx ]]; then
        echo -e "   nginx: /www/server/nginx/sbin/nginx（宝塔 nginx，站点配置写入 /www/server/panel/vhost/nginx/）"
    else
        echo -e "${YELLOW}   注意：宝塔 nginx 尚未安装。若要在宝塔里管理站点，请先在面板「软件商店」安装 Nginx；${NC}"
        echo -e "${YELLOW}        本次将按系统 nginx 部署（装好宝塔 nginx 后重跑本脚本即可切回宝塔环境）${NC}"
    fi
    # 宝塔面板自带 SSL 申请/续期。默认策略：已有宝塔证书就复用、不重复用 certbot 签发。
    if [[ -z "${CERT_MANAGER:-}" ]]; then
        export CERT_MANAGER="auto"
        echo -e "   证书策略: CERT_MANAGER=auto（已有宝塔证书则复用；面板「网站 → SSL」申请后重跑即可接入 HTTPS）"
    else
        echo -e "   证书策略: CERT_MANAGER=${CERT_MANAGER}（来自你的环境变量）"
    fi
else
    echo -e "${YELLOW}⚠️  未检测到宝塔面板（/www/server/panel 不存在）${NC}"
    echo -e "   本脚本将按普通 VPS 方式部署（系统 nginx + systemd）。"
    echo -e "   如果你确实想用宝塔：请先安装面板并安装 Nginx，再重跑本脚本。"
fi

echo -e "${CYAN}→ 转交 install.sh（唯一实现，两个入口行为一致）${NC}"
echo
exec bash "$TARGET" "$@"
