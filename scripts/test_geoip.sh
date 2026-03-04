#!/bin/bash

echo "=========================================="
echo "  GeoIP 数据库测试工具"
echo "=========================================="
echo ""

# 检查现有数据库
echo "检查现有数据库文件..."
echo ""

found_db=false

if [ -f "dbip-city-lite.mmdb" ]; then
    size=$(ls -lh dbip-city-lite.mmdb | awk '{print $5}')
    echo "✅ DB-IP City Lite: dbip-city-lite.mmdb ($size)"
    found_db=true
fi

if [ -f "GeoLite2-City.mmdb" ]; then
    size=$(ls -lh GeoLite2-City.mmdb | awk '{print $5}')
    echo "✅ GeoLite2 City: GeoLite2-City.mmdb ($size)"
    found_db=true
fi

if [ -f "IP2LOCATION-LITE-DB11.BIN" ]; then
    size=$(ls -lh IP2LOCATION-LITE-DB11.BIN | awk '{print $5}')
    echo "✅ IP2Location LITE (IPv4): IP2LOCATION-LITE-DB11.BIN ($size)"
    found_db=true
fi

if [ -f "IP2LOCATION-LITE-DB11.IPV6.BIN" ]; then
    size=$(ls -lh IP2LOCATION-LITE-DB11.IPV6.BIN | awk '{print $5}')
    echo "✅ IP2Location LITE (IPv6): IP2LOCATION-LITE-DB11.IPV6.BIN ($size)"
    found_db=true
fi

echo ""

if [ "$found_db" = false ]; then
    echo "❌ 未找到任何 GeoIP 数据库文件"
    echo ""
    echo "请选择下载方式："
    echo ""
    echo "1. DB-IP City Lite（推荐，中国数据详细）"
    echo "   go run scripts/download_dbip.go"
    echo ""
    echo "2. GeoLite2 City（MaxMind）"
    echo "   go run scripts/download_geoip.go"
    echo ""
    echo "3. IP2Location LITE（需要注册）"
    echo "   go run scripts/download_ip2location.go"
    echo ""
    exit 1
fi

# 创建测试程序
echo "运行测试..."
echo ""

go run scripts/test_geoip_quick.go

echo ""
echo "=========================================="
echo "如需查看详细文档，请参考："
echo "  docs/GEOIP_DATABASES.md"
echo "=========================================="
