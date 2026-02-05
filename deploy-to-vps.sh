#!/bin/bash
# 本地一键部署到VPS脚本
# 自动上传修复文件并执行修复

set -e

VPS_HOST="dy.moneyfly.top"
VPS_USER="root"
VPS_DIR="/www/wwwroot/dy.moneyfly.top"

echo "=========================================="
echo "🚀 一键部署到VPS"
echo "=========================================="
echo ""
echo "VPS地址: ${VPS_HOST}"
echo "VPS用户: ${VPS_USER}"
echo "VPS目录: ${VPS_DIR}"
echo ""

# 确认部署
read -p "确认部署到VPS？(y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ 部署已取消"
    exit 1
fi

echo ""
echo "1️⃣  检查本地构建产物..."
if [ ! -d "frontend/dist" ]; then
    echo "⚠️  未找到构建产物，开始构建..."
    cd frontend
    npm run build
    cd ..
fi
echo "✅ 构建产物检查完成"

echo ""
echo "2️⃣  上传源文件到VPS..."
echo "   上传 global.scss..."
scp frontend/src/styles/global.scss ${VPS_USER}@${VPS_HOST}:${VPS_DIR}/frontend/src/styles/

echo "   上传 Orders.vue..."
scp frontend/src/views/Orders.vue ${VPS_USER}@${VPS_HOST}:${VPS_DIR}/frontend/src/views/

echo "   上传 admin/Orders.vue..."
scp frontend/src/views/admin/Orders.vue ${VPS_USER}@${VPS_HOST}:${VPS_DIR}/frontend/src/views/admin/

echo "✅ 源文件上传完成"

echo ""
echo "3️⃣  上传修复脚本..."
scp vps-fix-orders.sh ${VPS_USER}@${VPS_HOST}:${VPS_DIR}/
echo "✅ 修复脚本上传完成"

echo ""
echo "4️⃣  在VPS上执行修复..."
ssh ${VPS_USER}@${VPS_HOST} << ENDSSH
cd ${VPS_DIR}
chmod +x vps-fix-orders.sh
./vps-fix-orders.sh
ENDSSH

if [ $? -eq 0 ]; then
    echo ""
    echo "=========================================="
    echo "✅ 部署成功！"
    echo "=========================================="
    echo ""
    echo "📋 重要提醒："
    echo ""
    echo "1. 清除浏览器缓存："
    echo "   Ctrl+Shift+Delete → 清除'缓存的图片和文件'"
    echo ""
    echo "2. 强制刷新页面："
    echo "   Ctrl+F5 或 Ctrl+Shift+R"
    echo ""
    echo "3. 清除Cloudflare缓存（如使用）："
    echo "   登录Cloudflare → dy.moneyfly.top → 缓存 → 清除所有"
    echo ""
    echo "4. 验证修复："
    echo "   访问: https://dy.moneyfly.top/admin/orders"
    echo "   应该看到'用户邮箱'列和正确的统计数据"
    echo ""
    echo "=========================================="
else
    echo ""
    echo "❌ 部署失败！请检查错误信息"
    exit 1
fi
