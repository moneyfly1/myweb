#!/bin/bash

# 专线节点完整流程测试脚本
# 使用方法: ./scripts/test_custom_node.sh

set -e

# 配置
BASE_URL="${BASE_URL:-http://localhost:8080/api/v1}"
ADMIN_TOKEN="${ADMIN_TOKEN:-}"  # 需要先获取管理员token

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    command -v curl >/dev/null 2>&1 || { log_error "需要安装 curl"; exit 1; }
    command -v jq >/dev/null 2>&1 || { log_warn "建议安装 jq 以便更好地查看JSON响应"; }
    log_info "依赖检查完成"
}

# 获取管理员token（需要先登录）
get_admin_token() {
    if [ -z "$ADMIN_TOKEN" ]; then
        log_warn "请先设置 ADMIN_TOKEN 环境变量"
        log_info "或者手动登录后获取token"
        read -p "请输入管理员token: " ADMIN_TOKEN
    fi
}

# API调用函数
api_call() {
    local method=$1
    local endpoint=$2
    local data=$3
    
    if [ -z "$data" ]; then
        curl -s -X "$method" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $ADMIN_TOKEN" \
            "${BASE_URL}${endpoint}"
    else
        curl -s -X "$method" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $ADMIN_TOKEN" \
            -d "$data" \
            "${BASE_URL}${endpoint}"
    fi
}

# 测试1: 保存系统设置
test_save_settings() {
    log_info "测试1: 保存专线节点系统设置..."
    
    local configs=(
        '{"key":"cloudflare_api_key","value":"<CLOUDFLARE_API_KEY_REMOVED>","category":"custom_node","type":"string","display_name":"Cloudflare API Key"}'
        '{"key":"cloudflare_email","value":"<CLOUDFLARE_EMAIL_REMOVED>","category":"custom_node","type":"string","display_name":"Cloudflare邮箱"}'
        '{"key":"cert_email","value":"<CERT_EMAIL_REMOVED>","category":"custom_node","type":"string","display_name":"证书申请邮箱"}'
    )
    
    for config in "${configs[@]}"; do
        local response=$(api_call "PUT" "/admin/configs/$(echo $config | jq -r '.key')" "$config")
        if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
            log_info "✓ 配置 $(echo $config | jq -r '.key') 保存成功"
        else
            # 尝试创建
            local create_response=$(api_call "POST" "/admin/configs" "$config")
            if echo "$create_response" | jq -e '.success' >/dev/null 2>&1; then
                log_info "✓ 配置 $(echo $config | jq -r '.key') 创建成功"
            else
                log_error "✗ 配置 $(echo $config | jq -r '.key') 保存失败"
                echo "$create_response" | jq '.' 2>/dev/null || echo "$create_response"
            fi
        fi
    done
}

# 测试2: 获取服务器列表
test_get_servers() {
    log_info "测试2: 获取服务器列表..."
    
    local response=$(api_call "GET" "/admin/servers")
    if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        local server_count=$(echo "$response" | jq '.data | length')
        log_info "✓ 获取服务器列表成功，共 $server_count 台服务器"
        echo "$response" | jq '.data[] | {id, name, host, status}' 2>/dev/null || echo "$response"
        return 0
    else
        log_error "✗ 获取服务器列表失败"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        return 1
    fi
}

# 测试3: 测试服务器连接
test_server_connection() {
    local server_id=$1
    log_info "测试3: 测试服务器连接 (ID: $server_id)..."
    
    local response=$(api_call "POST" "/admin/servers/$server_id/test" "{}")
    if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log_info "✓ 服务器连接测试成功"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        return 0
    else
        log_error "✗ 服务器连接测试失败"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        return 1
    fi
}

# 测试4: 自动设置XrayR
test_auto_setup_xrayr() {
    local server_id=$1
    log_info "测试4: 自动设置XrayR (服务器ID: $server_id)..."
    log_warn "这可能需要几分钟时间，请耐心等待..."
    
    local response=$(api_call "POST" "/admin/servers/$server_id/xrayr/auto-setup" '{"api_port":"10086"}')
    if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log_info "✓ XrayR自动设置已开始"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        log_info "等待30秒后检查状态..."
        sleep 30
        return 0
    else
        log_error "✗ XrayR自动设置启动失败"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        return 1
    fi
}

# 测试5: 检查XrayR安装状态
test_check_xrayr() {
    local server_id=$1
    log_info "测试5: 检查XrayR安装状态 (服务器ID: $server_id)..."
    
    local response=$(api_call "POST" "/admin/servers/$server_id/xrayr/check" "{}")
    if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        local installed=$(echo "$response" | jq -r '.data.installed // false')
        if [ "$installed" = "true" ]; then
            log_info "✓ XrayR已安装"
        else
            log_warn "✗ XrayR未安装或安装中"
        fi
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        return 0
    else
        log_error "✗ 检查XrayR状态失败"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        return 1
    fi
}

# 测试6: 获取XrayR配置
test_get_xrayr_config() {
    local server_id=$1
    log_info "测试6: 获取XrayR配置 (服务器ID: $server_id)..."
    
    local response=$(api_call "POST" "/admin/servers/$server_id/xrayr/config" "{}")
    if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log_info "✓ 获取XrayR配置成功"
        echo "$response" | jq '.data' 2>/dev/null || echo "$response"
        return 0
    else
        log_error "✗ 获取XrayR配置失败"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        return 1
    fi
}

# 测试7: 创建专线节点
test_create_custom_node() {
    log_info "测试7: 创建测试专线节点..."
    
    # 生成随机UUID和端口
    local uuid=$(uuidgen 2>/dev/null || cat /proc/sys/kernel/random/uuid 2>/dev/null || openssl rand -hex 16)
    local port=$((10000 + RANDOM % 55535))
    
    local node_data=$(cat <<EOF
{
    "server_id": $1,
    "name": "测试节点-$(date +%s)",
    "protocol": "vmess",
    "domain": "",
    "port": $port,
    "uuid": "$uuid",
    "network": "tcp",
    "security": "none",
    "traffic_limit": 10,
    "expire_time": null,
    "follow_user_expire": false
}
EOF
)
    
    local response=$(api_call "POST" "/admin/custom-nodes" "$node_data")
    if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log_info "✓ 专线节点创建请求已提交"
        local node_id=$(echo "$response" | jq -r '.data.id // empty')
        if [ -n "$node_id" ]; then
            log_info "节点ID: $node_id"
            echo "$node_id"
        fi
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        return 0
    else
        log_error "✗ 创建专线节点失败"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        return 1
    fi
}

# 测试8: 获取专线节点列表
test_get_custom_nodes() {
    log_info "测试8: 获取专线节点列表..."
    
    local response=$(api_call "GET" "/admin/custom-nodes")
    if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        local node_count=$(echo "$response" | jq '.data | length')
        log_info "✓ 获取专线节点列表成功，共 $node_count 个节点"
        echo "$response" | jq '.data[] | {id, name, protocol, status}' 2>/dev/null || echo "$response"
        return 0
    else
        log_error "✗ 获取专线节点列表失败"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        return 1
    fi
}

# 主测试流程
main() {
    log_info "=========================================="
    log_info "专线节点完整流程测试"
    log_info "=========================================="
    echo ""
    
    check_dependencies
    get_admin_token
    
    if [ -z "$ADMIN_TOKEN" ]; then
        log_error "未设置管理员token，无法继续测试"
        exit 1
    fi
    
    echo ""
    log_info "开始测试..."
    echo ""
    
    # 测试1: 保存系统设置
    test_save_settings
    echo ""
    
    # 测试2: 获取服务器列表
    if test_get_servers; then
        echo ""
        read -p "请输入要测试的服务器ID (直接回车跳过服务器测试): " server_id
        
        if [ -n "$server_id" ]; then
            # 测试3: 测试服务器连接
            test_server_connection "$server_id"
            echo ""
            
            # 询问是否自动设置XrayR
            read -p "是否自动设置XrayR? (y/n): " setup_xrayr
            if [ "$setup_xrayr" = "y" ] || [ "$setup_xrayr" = "Y" ]; then
                # 测试4: 自动设置XrayR
                test_auto_setup_xrayr "$server_id"
                echo ""
                
                # 测试5: 检查XrayR状态
                test_check_xrayr "$server_id"
                echo ""
                
                # 测试6: 获取XrayR配置
                test_get_xrayr_config "$server_id"
                echo ""
                
                # 询问是否创建测试节点
                read -p "是否创建测试节点? (y/n): " create_node
                if [ "$create_node" = "y" ] || [ "$create_node" = "Y" ]; then
                    # 测试7: 创建专线节点
                    test_create_custom_node "$server_id"
                    echo ""
                    
                    # 等待节点创建
                    log_info "等待节点创建完成（30秒）..."
                    sleep 30
                    
                    # 测试8: 获取节点列表
                    test_get_custom_nodes
                fi
            fi
        fi
    else
        log_warn "无法获取服务器列表，跳过服务器相关测试"
    fi
    
    echo ""
    log_info "=========================================="
    log_info "测试完成"
    log_info "=========================================="
}

# 运行主函数
main


