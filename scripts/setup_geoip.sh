#!/bin/bash

echo "=========================================="
echo "  GeoIP 数据库快速设置"
echo "=========================================="
echo ""
echo "正在下载 DB-IP City Lite 数据库..."
echo "（推荐使用，中国城市数据最详细）"
echo ""

go run scripts/download_dbip.go

if [ $? -eq 0 ]; then
    echo ""
    echo "=========================================="
    echo "  ✅ 设置完成！"
    echo "=========================================="
    echo ""
    echo "数据库已下载并准备就绪。"
    echo ""
    echo "现在您可以："
    echo "  1. 重启应用以加载新数据库"
    echo "  2. 或通过后台管理界面查看状态"
    echo ""
    echo "测试数据库："
    echo "  bash scripts/test_geoip.sh"
    echo ""
    echo "查看文档："
    echo "  docs/GEOIP_DATABASES.md"
    echo "  docs/GEOIP_UPGRADE.md"
    echo ""
else
    echo ""
    echo "=========================================="
    echo "  ❌ 下载失败"
    echo "=========================================="
    echo ""
    echo "请检查网络连接或手动下载："
    echo "  https://db-ip.com/db/download/ip-to-city-lite"
    echo ""
    echo "或使用 GeoLite2："
    echo "  go run scripts/download_geoip.go"
    echo ""
fi
