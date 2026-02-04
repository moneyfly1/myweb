#!/bin/bash

# 高级压力测试脚本
# 使用 Apache Bench (ab) 或 wrk 进行压力测试

BASE_URL="https://dy.moneyfly.top"

echo "========================================="
echo "网站压力测试 - 高级版"
echo "目标网站: $BASE_URL"
echo "========================================="
echo ""

# 检查工具是否安装
check_tool() {
    if ! command -v $1 &> /dev/null; then
        echo "❌ $1 未安装，请先安装:"
        if [ "$1" == "ab" ]; then
            echo "   macOS: brew install httpd"
            echo "   Ubuntu: sudo apt-get install apache2-utils"
        elif [ "$1" == "wrk" ]; then
            echo "   macOS: brew install wrk"
            echo "   Ubuntu: sudo apt-get install wrk"
        fi
        return 1
    fi
    return 0
}

# 使用 Apache Bench 测试
test_with_ab() {
    echo "【使用 Apache Bench 测试】"
    echo "----------------------------------------"
    
    if ! check_tool ab; then
        return
    fi
    
    echo ""
    echo "1. 测试首页 (1000请求, 100并发)"
    ab -n 1000 -c 100 -k "$BASE_URL/" | grep -E "(Requests per second|Time per request|Failed requests|Transfer rate)"
    
    echo ""
    echo "2. 测试登录页面 (1000请求, 100并发)"
    ab -n 1000 -c 100 -k "$BASE_URL/login" | grep -E "(Requests per second|Time per request|Failed requests|Transfer rate)"
    
    echo ""
    echo "3. 测试API端点 (2000请求, 200并发)"
    ab -n 2000 -c 200 -k "$BASE_URL/api/v1/settings/public-settings" | grep -E "(Requests per second|Time per request|Failed requests|Transfer rate)"
}

# 使用 wrk 测试
test_with_wrk() {
    echo "【使用 wrk 测试】"
    echo "----------------------------------------"
    
    if ! check_tool wrk; then
        return
    fi
    
    echo ""
    echo "1. 测试首页 (30秒, 100线程, 100连接)"
    wrk -t100 -c100 -d30s --timeout 10s "$BASE_URL/" | grep -E "(Requests/sec|Transfer/sec|Latency)"
    
    echo ""
    echo "2. 测试登录页面 (30秒, 100线程, 100连接)"
    wrk -t100 -c100 -d30s --timeout 10s "$BASE_URL/login" | grep -E "(Requests/sec|Transfer/sec|Latency)"
    
    echo ""
    echo "3. 测试API端点 (60秒, 200线程, 200连接)"
    wrk -t200 -c200 -d60s --timeout 10s "$BASE_URL/api/v1/settings/public-settings" | grep -E "(Requests/sec|Transfer/sec|Latency)"
}

# 使用 curl 进行简单测试
test_with_curl() {
    echo "【使用 curl 进行响应时间测试】"
    echo "----------------------------------------"
    
    endpoints=(
        "/"
        "/login"
        "/register"
        "/api/v1/settings/public-settings"
    )
    
    for endpoint in "${endpoints[@]}"; do
        echo ""
        echo "测试: $BASE_URL$endpoint"
        total_time=0
        success=0
        
        for i in {1..10}; do
            response=$(curl -o /dev/null -s -w "%{time_total}\n" "$BASE_URL$endpoint" 2>&1)
            if [ $? -eq 0 ]; then
                total_time=$(echo "$total_time + $response" | bc)
                success=$((success + 1))
            fi
        done
        
        if [ $success -gt 0 ]; then
            avg_time=$(echo "scale=3; $total_time / $success" | bc)
            echo "  成功: $success/10, 平均响应时间: ${avg_time}s"
        else
            echo "  全部失败"
        fi
    done
}

# 主菜单
echo "请选择测试方式:"
echo "1. Apache Bench (ab)"
echo "2. wrk"
echo "3. curl (简单测试)"
echo "4. 全部执行"
echo ""
read -p "请输入选项 (1-4): " choice

case $choice in
    1)
        test_with_ab
        ;;
    2)
        test_with_wrk
        ;;
    3)
        test_with_curl
        ;;
    4)
        test_with_ab
        echo ""
        test_with_wrk
        echo ""
        test_with_curl
        ;;
    *)
        echo "无效选项"
        exit 1
        ;;
esac

echo ""
echo "========================================="
echo "测试完成！"
echo "========================================="
