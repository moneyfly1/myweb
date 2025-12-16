#!/bin/bash

# 订阅地址测试脚本
# 用于测试订阅地址是否能正常获取节点信息

DOMAIN="${1:-dy.moneyfly.top}"
SUBSCRIPTION_URL="${2}"

if [ -z "$SUBSCRIPTION_URL" ]; then
    echo "❌ 错误: 请提供订阅URL"
    echo "用法: $0 [域名] <订阅URL>"
    echo "示例: $0 dy.moneyfly.top abc123xyz"
    exit 1
fi

TIMESTAMP=$(date +%s)
BASE_URL="https://${DOMAIN}"

echo "=========================================="
echo "🧪 测试订阅地址获取节点信息"
echo "=========================================="
echo "域名: $DOMAIN"
echo "订阅URL: $SUBSCRIPTION_URL"
echo "时间戳: $TIMESTAMP"
echo ""

# 测试 Clash 订阅
echo "📋 测试 Clash 订阅格式..."
CLASH_URL="${BASE_URL}/api/v1/subscriptions/clash/${SUBSCRIPTION_URL}?t=${TIMESTAMP}"
echo "URL: $CLASH_URL"
echo ""

CLASH_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -H "User-Agent: ClashForWindows/1.0.0" "$CLASH_URL")
CLASH_HTTP_CODE=$(echo "$CLASH_RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
CLASH_BODY=$(echo "$CLASH_RESPONSE" | sed '/HTTP_CODE:/d')

if [ "$CLASH_HTTP_CODE" = "200" ]; then
    echo "✅ Clash 订阅成功 (HTTP $CLASH_HTTP_CODE)"
    # 检查是否包含节点信息
    if echo "$CLASH_BODY" | grep -q "proxies:"; then
        NODE_COUNT=$(echo "$CLASH_BODY" | grep -c "  - name:" || echo "0")
        echo "   📊 发现 $NODE_COUNT 个节点"
        
        # 检查信息节点
        if echo "$CLASH_BODY" | grep -q "📢 网站域名"; then
            echo "   ✅ 包含网站域名信息节点"
        fi
        if echo "$CLASH_BODY" | grep -q "⏰ 到期时间"; then
            echo "   ✅ 包含到期时间信息节点"
        fi
        if echo "$CLASH_BODY" | grep -q "💬 售后QQ"; then
            echo "   ✅ 包含售后QQ信息节点"
        fi
        
        # 显示前3个节点名称
        echo "   前3个节点:"
        echo "$CLASH_BODY" | grep "  - name:" | head -3 | sed 's/^/      /'
    else
        echo "   ⚠️  响应中未找到节点信息"
        echo "   响应内容:"
        echo "$CLASH_BODY" | head -10 | sed 's/^/      /'
    fi
else
    echo "❌ Clash 订阅失败 (HTTP $CLASH_HTTP_CODE)"
    echo "   错误信息:"
    echo "$CLASH_BODY" | head -10 | sed 's/^/      /'
fi

echo ""
echo "=========================================="

# 测试 V2Ray 订阅
echo "📋 测试 V2Ray 订阅格式..."
V2RAY_URL="${BASE_URL}/api/v1/subscriptions/v2ray/${SUBSCRIPTION_URL}?t=${TIMESTAMP}"
echo "URL: $V2RAY_URL"
echo ""

V2RAY_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -H "User-Agent: V2RayNG/1.0.0" "$V2RAY_URL")
V2RAY_HTTP_CODE=$(echo "$V2RAY_RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
V2RAY_BODY=$(echo "$V2RAY_RESPONSE" | sed '/HTTP_CODE:/d')

if [ "$V2RAY_HTTP_CODE" = "200" ]; then
    echo "✅ V2Ray 订阅成功 (HTTP $V2RAY_HTTP_CODE)"
    # V2Ray 格式是 Base64 编码的
    DECODED=$(echo "$V2RAY_BODY" | base64 -d 2>/dev/null || echo "")
    if [ -n "$DECODED" ]; then
        NODE_COUNT=$(echo "$DECODED" | grep -c "vmess://\|vless://\|trojan://\|ss://" || echo "0")
        echo "   📊 发现 $NODE_COUNT 个节点链接"
    else
        echo "   ⚠️  无法解码 Base64 响应"
    fi
else
    echo "❌ V2Ray 订阅失败 (HTTP $V2RAY_HTTP_CODE)"
    echo "   错误信息:"
    echo "$V2RAY_BODY" | head -10 | sed 's/^/      /'
fi

echo ""
echo "=========================================="

# 测试 SSR 订阅
echo "📋 测试 SSR 订阅格式..."
SSR_URL="${BASE_URL}/api/v1/subscriptions/ssr/${SUBSCRIPTION_URL}?t=${TIMESTAMP}"
echo "URL: $SSR_URL"
echo ""

SSR_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -H "User-Agent: ShadowsocksR/1.0.0" "$SSR_URL")
SSR_HTTP_CODE=$(echo "$SSR_RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
SSR_BODY=$(echo "$SSR_RESPONSE" | sed '/HTTP_CODE:/d')

if [ "$SSR_HTTP_CODE" = "200" ]; then
    echo "✅ SSR 订阅成功 (HTTP $SSR_HTTP_CODE)"
    # SSR 格式也是 Base64 编码的
    DECODED=$(echo "$SSR_BODY" | base64 -d 2>/dev/null || echo "")
    if [ -n "$DECODED" ]; then
        NODE_COUNT=$(echo "$DECODED" | grep -c "ssr://" || echo "0")
        echo "   📊 发现 $NODE_COUNT 个节点链接"
    else
        echo "   ⚠️  无法解码 Base64 响应"
    fi
else
    echo "❌ SSR 订阅失败 (HTTP $SSR_HTTP_CODE)"
    echo "   错误信息:"
    echo "$SSR_BODY" | head -10 | sed 's/^/      /'
fi

echo ""
echo "=========================================="
echo "✅ 测试完成"
echo "=========================================="

