#!/bin/bash
# Clash 订阅配置检查工具

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "======================================"
echo "  Clash 订阅配置检查工具"
echo "======================================"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查模板文件
echo -e "${BLUE}[1/5]${NC} 检查模板文件..."
TEMPLATE_FILE="$PROJECT_DIR/uploads/config/temp.yaml"

if [ -f "$TEMPLATE_FILE" ]; then
    echo -e "  ${GREEN}✓${NC} 模板文件存在: $TEMPLATE_FILE"
    FILE_SIZE=$(ls -lh "$TEMPLATE_FILE" | awk '{print $5}')
    echo -e "  ${GREEN}✓${NC} 文件大小: $FILE_SIZE"
else
    echo -e "  ${RED}✗${NC} 模板文件不存在！"
    exit 1
fi

echo ""

# 验证 YAML 语法
echo -e "${BLUE}[2/5]${NC} 验证 YAML 语法..."

if command -v python3 &> /dev/null; then
    if python3 -c "import yaml; yaml.safe_load(open('$TEMPLATE_FILE'))" 2>/dev/null; then
        echo -e "  ${GREEN}✓${NC} YAML 语法正确"
    else
        echo -e "  ${RED}✗${NC} YAML 语法错误！"
        exit 1
    fi
else
    echo -e "  ${YELLOW}⚠${NC} Python3 未安装，跳过语法检查"
fi

echo ""

# 统计配置信息
echo -e "${BLUE}[3/5]${NC} 统计配置信息..."

PROXY_GROUPS=$(grep -c "^  - name:" "$TEMPLATE_FILE" || true)
echo -e "  ${GREEN}✓${NC} 代理组数量: $PROXY_GROUPS 个"

RULES=$(grep -c "^  - " "$TEMPLATE_FILE" || true)
echo -e "  ${GREEN}✓${NC} 总行数（包含规则）: $RULES 行"

if grep -q "^rules:" "$TEMPLATE_FILE"; then
    echo -e "  ${GREEN}✓${NC} 规则部分存在"
else
    echo -e "  ${RED}✗${NC} 规则部分缺失！"
fi

echo ""

# 检查关键代理组
echo -e "${BLUE}[4/5]${NC} 检查关键代理组..."

KEY_GROUPS=("🚀 节点选择" "♻️ 自动选择" "🔰 故障转移" "🔮 负载均衡" "🤖 OpenAI" "🌍 国际媒体")

for group in "${KEY_GROUPS[@]}"; do
    if grep -q "name: $group" "$TEMPLATE_FILE" || grep -q "name: \"$group\"" "$TEMPLATE_FILE"; then
        echo -e "  ${GREEN}✓${NC} $group"
    else
        echo -e "  ${RED}✗${NC} $group 缺失"
    fi
done

echo ""

# 运行详细验证（如果 Python 可用）
echo -e "${BLUE}[5/5]${NC} 运行详细验证..."

if command -v python3 &> /dev/null && [ -f "$SCRIPT_DIR/verify_clash_config.py" ]; then
    echo ""
    python3 "$SCRIPT_DIR/verify_clash_config.py" "$TEMPLATE_FILE"
else
    echo -e "  ${YELLOW}⚠${NC} 详细验证脚本不可用，跳过"
fi

echo ""
echo "======================================"
echo -e "${GREEN}✓ 检查完成！${NC}"
echo "======================================"
echo ""
echo "提示："
echo "  • 模板文件路径: $TEMPLATE_FILE"
echo "  • 修改模板后请重新运行此脚本验证"
echo "  • 详细文档: docs/clash_config_template.md"
echo "  • 验证报告: 订阅配置验证报告.md"
echo ""
