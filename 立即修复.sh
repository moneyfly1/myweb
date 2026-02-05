#!/bin/bash
# 最简单的一键修复脚本

echo "🚀 开始修复VPS订单列表问题..."
echo ""
echo "VPS: dy.moneyfly.top"
echo "目录: /www/wwwroot/dy.moneyfly.top"
echo ""

# 上传修复后的源文件
echo "1. 上传修复文件..."
scp frontend/src/styles/global.scss root@dy.moneyfly.top:/www/wwwroot/dy.moneyfly.top/frontend/src/styles/ && \
scp frontend/src/views/Orders.vue root@dy.moneyfly.top:/www/wwwroot/dy.moneyfly.top/frontend/src/views/ && \
scp frontend/src/views/admin/Orders.vue root@dy.moneyfly.top:/www/wwwroot/dy.moneyfly.top/frontend/src/views/admin/ && \
echo "✅ 文件上传完成" || { echo "❌ 上传失败"; exit 1; }

# 在VPS上执行修复
echo ""
echo "2. 在VPS上重新构建..."
ssh root@dy.moneyfly.top << 'ENDSSH'
cd /www/wwwroot/dy.moneyfly.top/frontend
echo "   清除缓存..."
rm -rf dist node_modules/.cache node_modules/.vite .vite
npm cache clean --force
echo "   开始构建..."
npm run build
echo "   重启Nginx..."
nginx -t && nginx -s reload 2>/dev/null || sudo nginx -t && sudo nginx -s reload
echo "✅ VPS修复完成"
ENDSSH

if [ $? -eq 0 ]; then
    echo ""
    echo "=========================================="
    echo "✅ 修复成功！"
    echo "=========================================="
    echo ""
    echo "⚠️  重要：请立即执行以下操作"
    echo ""
    echo "1. 清除浏览器缓存："
    echo "   Ctrl+Shift+Delete → 清除'缓存的图片和文件'"
    echo ""
    echo "2. 访问测试："
    echo "   https://dy.moneyfly.top/admin/orders"
    echo ""
    echo "3. 验证修复成功的标志："
    echo "   ✓ 表格有'用户邮箱'列"
    echo "   ✓ 统计数据不再全是0"
    echo "   ✓ 手机端显示卡片式列表"
    echo ""
else
    echo "❌ 修复失败，请检查错误信息"
    exit 1
fi
