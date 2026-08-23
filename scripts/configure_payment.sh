#!/bin/bash

# 码支付和易支付配置脚本
# 根据截图信息配置支付系统

echo "=== 码支付和易支付配置工具 ==="
echo ""

# 商户凭据不再硬编码进脚本/仓库。
# 优先级：环境变量 > 交互输入（YIPAY_PID / YIPAY_KEY 必填）。
YIPAY_PID="${YIPAY_PID:-}"
if [ -z "$YIPAY_PID" ]; then
    read -p "请输入商户ID (PID): " YIPAY_PID
fi
if [ -z "$YIPAY_PID" ]; then
    echo "错误: 商户ID不能为空（可设置环境变量 YIPAY_PID 传入）"
    exit 1
fi

YIPAY_KEY="${YIPAY_KEY:-}"
if [ -z "$YIPAY_KEY" ]; then
    read -sp "请输入商户密钥（输入不回显）: " YIPAY_KEY
    echo ""
fi
if [ -z "$YIPAY_KEY" ]; then
    echo "错误: 商户密钥不能为空（可设置环境变量 YIPAY_KEY 传入）"
    exit 1
fi

# 网关地址同样支持环境变量覆盖；如未设置则要求输入
# （不同支付平台/商户的网关地址不同，请勿使用他人截图中的地址）
YIPAY_GATEWAY="${YIPAY_GATEWAY:-}"
if [ -z "$YIPAY_GATEWAY" ]; then
    read -p "请输入支付网关地址（例如: https://pay.example.com/）: " YIPAY_GATEWAY
fi
if [ -z "$YIPAY_GATEWAY" ]; then
    echo "错误: 网关地址不能为空（可设置环境变量 YIPAY_GATEWAY 传入）"
    exit 1
fi
YIPAY_SUBMIT_URL="${YIPAY_SUBMIT_URL:-}"
if [ -z "$YIPAY_SUBMIT_URL" ]; then
    YIPAY_SUBMIT_URL="${YIPAY_GATEWAY%/}/submit.php"
fi
YIPAY_MAPI_URL="${YIPAY_MAPI_URL:-}"
if [ -z "$YIPAY_MAPI_URL" ]; then
    YIPAY_MAPI_URL="${YIPAY_GATEWAY%/}/mapi.php"
fi

# 获取当前域名（需要用户确认）
echo "请输入您的网站域名（例如: example.com 或 localhost:8080）:"
read DOMAIN

if [ -z "$DOMAIN" ]; then
    echo "错误: 域名不能为空"
    exit 1
fi

# 构建回调地址
NOTIFY_URL="https://${DOMAIN}/api/v1/payment/notify/codepay"
RETURN_URL="https://${DOMAIN}/payment/return"

# 如果是本地域名，使用 http
if [[ "$DOMAIN" == *"localhost"* ]] || [[ "$DOMAIN" == *"127.0.0.1"* ]]; then
    NOTIFY_URL="http://${DOMAIN}/api/v1/payment/notify/codepay"
    RETURN_URL="http://${DOMAIN}/payment/return"
fi

echo ""
echo "=== 配置信息 ==="
echo "商户ID (PID): $YIPAY_PID"
echo "商户密钥: ${YIPAY_KEY:0:10}..."
echo "网关地址: $YIPAY_GATEWAY"
echo "异步回调地址: $NOTIFY_URL"
echo "同步回调地址: $RETURN_URL"
echo ""

# 生成配置 JSON
CONFIG_JSON=$(cat <<EOF
{
  "gateway_url": "$YIPAY_GATEWAY",
  "api_url": "$YIPAY_MAPI_URL",
  "submit_url": "$YIPAY_SUBMIT_URL",
  "notify_url": "$NOTIFY_URL",
  "supported_types": ["alipay", "wxpay"]
}
EOF
)

echo "=== SQL 配置语句 ==="
echo ""
echo "-- 1. 配置码支付（Codepay）"
echo "INSERT INTO payment_configs (pay_type, app_id, merchant_private_key, notify_url, return_url, status, sort_order, config_json, created_at, updated_at)"
echo "VALUES ("
echo "  'codepay',"
echo "  '$YIPAY_PID',"
echo "  '$YIPAY_KEY',"
echo "  '$NOTIFY_URL',"
echo "  '$RETURN_URL',"
echo "  1,"
echo "  10,"
echo "  '$CONFIG_JSON',"
echo "  NOW(),"
echo "  NOW()"
echo ")"
echo "ON DUPLICATE KEY UPDATE"
echo "  app_id = '$YIPAY_PID',"
echo "  merchant_private_key = '$YIPAY_KEY',"
echo "  notify_url = '$NOTIFY_URL',"
echo "  return_url = '$RETURN_URL',"
echo "  config_json = '$CONFIG_JSON',"
echo "  status = 1,"
echo "  updated_at = NOW();"
echo ""

echo "-- 2. 配置易支付（Yipay）- 使用相同的商户信息"
echo "INSERT INTO payment_configs (pay_type, app_id, merchant_private_key, notify_url, return_url, status, sort_order, config_json, created_at, updated_at)"
echo "VALUES ("
echo "  'yipay',"
echo "  '$YIPAY_PID',"
echo "  '$YIPAY_KEY',"
echo "  '${NOTIFY_URL/codepay/yipay}',"
echo "  '$RETURN_URL',"
echo "  1,"
echo "  11,"
echo "  '$(echo $CONFIG_JSON | sed 's/codepay/yipay/g')',"
echo "  NOW(),"
echo "  NOW()"
echo ")"
echo "ON DUPLICATE KEY UPDATE"
echo "  app_id = '$YIPAY_PID',"
echo "  merchant_private_key = '$YIPAY_KEY',"
echo "  notify_url = '${NOTIFY_URL/codepay/yipay}',"
echo "  return_url = '$RETURN_URL',"
echo "  config_json = '$(echo $CONFIG_JSON | sed 's/codepay/yipay/g')',"
echo "  status = 1,"
echo "  updated_at = NOW();"
echo ""

echo "=== 执行说明 ==="
echo "1. 复制上面的 SQL 语句"
echo "2. 连接到您的数据库"
echo "3. 执行 SQL 语句"
echo "4. 重启您的应用服务"
echo ""
echo "=== 重要提示 ==="
echo "⚠️  如果您的网站是本地环境（localhost），码支付平台无法直接回调"
echo "    需要使用内网穿透工具（如 ngrok, frp）将本地服务暴露到公网"
echo ""
echo "⚠️  请在码支付平台后台配置以下回调地址:"
echo "    异步通知地址: $NOTIFY_URL"
echo "    同步返回地址: $RETURN_URL"
echo ""
echo "=== 测试回调 ==="
echo "您可以使用以下命令测试回调接口是否可访问:"
echo "curl -X POST $NOTIFY_URL -d 'pid=$YIPAY_PID&trade_no=TEST123&out_trade_no=ORDER123&type=alipay&name=测试&money=0.01&trade_status=TRADE_SUCCESS&sign=test'"
echo ""
