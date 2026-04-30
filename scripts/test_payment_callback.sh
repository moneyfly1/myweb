#!/bin/bash

# 支付回调测试脚本
# 用于测试码支付和易支付的回调功能

set -e

# 颜色定义
RED='\033[0:31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== 支付回调测试工具 ===${NC}"
echo ""

# 配置信息（从截图获取）
PID="REDACTED_PID"
KEY="REDACTED_MERCHANT_KEY"

# 获取测试参数
echo "请输入您的网站地址（例如: http://localhost:8080 或 https://example.com）:"
read BASE_URL

if [ -z "$BASE_URL" ]; then
    echo -e "${RED}错误: 网站地址不能为空${NC}"
    exit 1
fi

# 移除末尾的斜杠
BASE_URL="${BASE_URL%/}"

echo ""
echo "请选择测试类型:"
echo "1) 码支付 (Codepay)"
echo "2) 易支付 (Yipay)"
read -p "请输入选项 (1 或 2): " PAYMENT_TYPE

if [ "$PAYMENT_TYPE" == "1" ]; then
    NOTIFY_URL="${BASE_URL}/api/v1/payment/notify/codepay"
    TYPE_NAME="码支付"
elif [ "$PAYMENT_TYPE" == "2" ]; then
    NOTIFY_URL="${BASE_URL}/api/v1/payment/notify/yipay"
    TYPE_NAME="易支付"
else
    echo -e "${RED}错误: 无效的选项${NC}"
    exit 1
fi

echo ""
echo "请输入测试订单号（留空则自动生成）:"
read ORDER_NO

if [ -z "$ORDER_NO" ]; then
    ORDER_NO="TEST$(date +%Y%m%d%H%M%S)"
    echo -e "${YELLOW}自动生成订单号: $ORDER_NO${NC}"
fi

echo ""
echo "请输入测试金额（默认: 0.01）:"
read AMOUNT

if [ -z "$AMOUNT" ]; then
    AMOUNT="0.01"
fi

echo ""
echo "请选择支付方式:"
echo "1) 支付宝 (alipay)"
echo "2) 微信 (wxpay)"
read -p "请输入选项 (1 或 2): " PAY_METHOD

if [ "$PAY_METHOD" == "1" ]; then
    PAY_TYPE="alipay"
elif [ "$PAY_METHOD" == "2" ]; then
    PAY_TYPE="wxpay"
else
    echo -e "${RED}错误: 无效的选项${NC}"
    exit 1
fi

# 生成交易号
TRADE_NO="PLATFORM$(date +%Y%m%d%H%M%S)"

# 构建签名参数
SIGN_PARAMS="money=${AMOUNT}&name=测试订单&out_trade_no=${ORDER_NO}&pid=${PID}&trade_no=${TRADE_NO}&trade_status=TRADE_SUCCESS&type=${PAY_TYPE}${KEY}"

# 计算 MD5 签名
SIGN=$(echo -n "$SIGN_PARAMS" | md5 -q)

echo ""
echo -e "${GREEN}=== 测试信息 ===${NC}"
echo "支付类型: $TYPE_NAME"
echo "回调地址: $NOTIFY_URL"
echo "订单号: $ORDER_NO"
echo "交易号: $TRADE_NO"
echo "金额: $AMOUNT"
echo "支付方式: $PAY_TYPE"
echo "签名字符串: ${SIGN_PARAMS:0:50}..."
echo "签名: $SIGN"
echo ""

# 发送回调请求
echo -e "${YELLOW}正在发送回调请求...${NC}"
echo ""

RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$NOTIFY_URL" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "pid=$PID" \
  -d "trade_no=$TRADE_NO" \
  -d "out_trade_no=$ORDER_NO" \
  -d "type=$PAY_TYPE" \
  -d "name=测试订单" \
  -d "money=$AMOUNT" \
  -d "trade_status=TRADE_SUCCESS" \
  -d "sign=$SIGN" \
  -d "sign_type=MD5")

# 分离响应体和状态码
HTTP_BODY=$(echo "$RESPONSE" | sed -e 's/HTTP_CODE\:.*//g')
HTTP_CODE=$(echo "$RESPONSE" | tr -d '\n' | sed -e 's/.*HTTP_CODE://')

echo -e "${GREEN}=== 响应结果 ===${NC}"
echo "HTTP 状态码: $HTTP_CODE"
echo "响应内容: $HTTP_BODY"
echo ""

# 判断结果
if [ "$HTTP_CODE" == "200" ] && [ "$HTTP_BODY" == "success" ]; then
    echo -e "${GREEN}✓ 回调测试成功！${NC}"
    echo ""
    echo "请检查以下内容:"
    echo "1. 订单状态是否已更新为 'paid'"
    echo "2. 用户套餐是否已开通"
    echo "3. 设备数量是否已更新"
    echo "4. 充值金额是否已到账"
elif [ "$HTTP_CODE" == "200" ] && [ "$HTTP_BODY" == "fail" ]; then
    echo -e "${RED}✗ 回调失败: 签名验证失败${NC}"
    echo ""
    echo "可能的原因:"
    echo "1. 商户ID或密钥配置错误"
    echo "2. 签名算法不匹配"
    echo "3. 参数格式错误"
else
    echo -e "${RED}✗ 回调失败${NC}"
    echo ""
    echo "可能的原因:"
    echo "1. 回调地址不可访问"
    echo "2. 服务未启动"
    echo "3. 路由配置错误"
fi

echo ""
echo -e "${YELLOW}=== 查看日志 ===${NC}"
echo "请查看应用日志以获取详细信息:"
echo "tail -f logs/app.log | grep -i payment"
echo ""

# 提供 curl 命令供手动测试
echo -e "${YELLOW}=== 手动测试命令 ===${NC}"
echo "您可以使用以下命令手动测试:"
echo ""
echo "curl -X POST '$NOTIFY_URL' \\"
echo "  -d 'pid=$PID' \\"
echo "  -d 'trade_no=$TRADE_NO' \\"
echo "  -d 'out_trade_no=$ORDER_NO' \\"
echo "  -d 'type=$PAY_TYPE' \\"
echo "  -d 'name=测试订单' \\"
echo "  -d 'money=$AMOUNT' \\"
echo "  -d 'trade_status=TRADE_SUCCESS' \\"
echo "  -d 'sign=$SIGN' \\"
echo "  -d 'sign_type=MD5'"
echo ""
