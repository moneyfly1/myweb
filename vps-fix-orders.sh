#!/bin/bash
# VPS订单列表修复脚本
# 专门修复组件加载错误和缓存问题

set -e

echo "=========================================="
echo "订单列表问题修复脚本"
echo "=========================================="

# 项目路径
PROJECT_DIR="/www/wwwroot/dy.moneyfly.top"
FRONTEND_DIR="${PROJECT_DIR}/frontend"

cd "${PROJECT_DIR}"

echo ""
echo "📊 当前状态检查..."
echo "项目目录: ${PROJECT_DIR}"
echo "前端目录: ${FRONTEND_DIR}"

if [ ! -d "${FRONTEND_DIR}" ]; then
    echo "❌ 错误: 前端目录不存在！"
    echo "请检查路径是否正确：${FRONTEND_DIR}"
    exit 1
fi

echo "✅ 目录验证通过"

echo ""
echo "1️⃣  停止前端服务..."
# 尝试多种停止方式
pkill -f "vite" 2>/dev/null || true
pkill -f "npm run dev" 2>/dev/null || true
if command -v pm2 &> /dev/null; then
    pm2 stop frontend 2>/dev/null || true
    pm2 delete frontend 2>/dev/null || true
fi
echo "✅ 服务已停止"

echo ""
echo "2️⃣  备份当前dist目录..."
cd "${FRONTEND_DIR}"
if [ -d "dist" ]; then
    BACKUP_NAME="dist_backup_$(date +%Y%m%d_%H%M%S)"
    mv dist "${BACKUP_NAME}"
    echo "✅ 备份完成: ${BACKUP_NAME}"
else
    echo "⚠️  没有找到dist目录，跳过备份"
fi

echo ""
echo "3️⃣  清除所有缓存..."
rm -rf node_modules/.cache 2>/dev/null || true
rm -rf node_modules/.vite 2>/dev/null || true
rm -rf .vite 2>/dev/null || true
rm -rf dist 2>/dev/null || true
echo "✅ 前端缓存已清除"

echo ""
echo "4️⃣  清除npm缓存..."
npm cache clean --force
echo "✅ npm缓存已清除"

echo ""
echo "5️⃣  验证路由配置..."
if grep -q "AdminOrders.*admin/Orders.vue" src/router/index.js; then
    echo "✅ 路由配置正确"
else
    echo "⚠️  警告: 路由配置可能有问题"
fi

echo ""
echo "6️⃣  重新构建前端..."
echo "   这可能需要1-2分钟..."
npm run build

if [ $? -eq 0 ]; then
    echo "✅ 构建成功！"
else
    echo "❌ 构建失败！请检查错误信息"
    exit 1
fi

echo ""
echo "7️⃣  验证构建产物..."
if [ -d "dist" ] && [ -f "dist/index.html" ]; then
    echo "✅ 构建产物验证通过"
    ls -lh dist/ | head -5
else
    echo "❌ 构建产物不完整！"
    exit 1
fi

echo ""
echo "8️⃣  清除Nginx缓存..."
if [ -d "/var/cache/nginx" ]; then
    rm -rf /var/cache/nginx/* 2>/dev/null || sudo rm -rf /var/cache/nginx/* 2>/dev/null || true
    echo "✅ Nginx缓存已清除"
fi

echo ""
echo "9️⃣  重启Nginx..."
if command -v nginx &> /dev/null; then
    nginx -t 2>/dev/null && nginx -s reload 2>/dev/null || sudo nginx -t && sudo nginx -s reload
    echo "✅ Nginx已重启"
else
    echo "⚠️  未找到Nginx命令"
fi

echo ""
echo "🔟  启动前端服务（如果需要）..."
if command -v pm2 &> /dev/null; then
    # 如果使用pm2管理前端服务
    # pm2 start npm --name "frontend" -- run dev
    # pm2 save
    echo "⚠️  如需使用pm2管理，请手动执行: pm2 start npm --name 'frontend' -- run dev"
fi

echo ""
echo "=========================================="
echo "✅ 修复完成！"
echo "=========================================="
echo ""
echo "📋 接下来请执行："
echo ""
echo "1. 清除浏览器缓存："
echo "   - 按 Ctrl+Shift+Delete"
echo "   - 选择'缓存的图片和文件'"
echo "   - 时间范围：全部时间"
echo "   - 点击'清除数据'"
echo ""
echo "2. 强制刷新页面："
echo "   - Windows: Ctrl+F5"
echo "   - Mac: Cmd+Shift+R"
echo ""
echo "3. 或使用无痕模式测试："
echo "   - Ctrl+Shift+N (Chrome)"
echo ""
echo "4. 如果使用Cloudflare CDN："
echo "   - 登录Cloudflare控制台"
echo "   - 选择 dy.moneyfly.top"
echo "   - 缓存 → 清除缓存 → 全部清除"
echo ""
echo "=========================================="
echo "验证步骤："
echo ""
echo "管理端 (https://dy.moneyfly.top/admin/orders)："
echo "  ✓ 应该看到'用户邮箱'列"
echo "  ✓ 应该显示所有用户的订单统计"
echo "  ✓ 表格应该有'导出订单'和'订单统计'按钮"
echo ""
echo "手机端（浏览器调整到375px宽度）："
echo "  ✓ 应该显示卡片式订单列表"
echo "  ✓ 订单列表不应该消失"
echo "=========================================="
