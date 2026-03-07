#!/bin/bash
# NodePass-Pro 集成测试脚本

set -e

# 配置
API_BASE_URL="${API_BASE_URL:-http://localhost:8080/api/v1}"
ADMIN_TOKEN="${ADMIN_TOKEN:-}"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."

    if ! command -v curl &> /dev/null; then
        log_error "curl 未安装"
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        log_warn "jq 未安装，建议安装以获得更好的输出格式"
    fi

    log_info "依赖检查完成"
}

# 测试API连接
test_api_connection() {
    log_info "测试API连接..."

    response=$(curl -s -w "\n%{http_code}" "${API_BASE_URL}/../health")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" != "200" ]; then
        log_error "API连接失败，HTTP状态码: $http_code"
        exit 1
    fi

    log_info "API连接成功"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
}

# 测试用户登录
test_login() {
    log_info "测试用户登录..."

    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}' \
        "${API_BASE_URL}/auth/login")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" != "200" ]; then
        log_error "登录失败，HTTP状态码: $http_code"
        echo "$body"
        exit 1
    fi

    # 提取token
    ADMIN_TOKEN=$(echo "$body" | jq -r '.data.token' 2>/dev/null)

    if [ -z "$ADMIN_TOKEN" ] || [ "$ADMIN_TOKEN" = "null" ]; then
        log_error "无法获取token"
        exit 1
    fi

    log_info "登录成功，token: ${ADMIN_TOKEN:0:20}..."
}

# 测试创建入口节点组
test_create_entry_group() {
    log_info "测试创建入口节点组..."

    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d '{
            "name": "测试入口组",
            "type": "entry",
            "description": "集成测试创建的入口节点组",
            "config": {
                "allowed_protocols": ["tcp", "udp"],
                "port_range": {"start": 10000, "end": 20000},
                "entry_config": {
                    "require_exit_group": false,
                    "traffic_multiplier": 1.0,
                    "dns_load_balance": true
                }
            }
        }' \
        "${API_BASE_URL}/node-groups")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" != "200" ]; then
        log_error "创建入口节点组失败，HTTP状态码: $http_code"
        echo "$body"
        return 1
    fi

    ENTRY_GROUP_ID=$(echo "$body" | jq -r '.data.id' 2>/dev/null)
    log_info "入口节点组创建成功，ID: $ENTRY_GROUP_ID"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
}

# 测试创建出口节点组
test_create_exit_group() {
    log_info "测试创建出口节点组..."

    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d '{
            "name": "测试出口组",
            "type": "exit",
            "description": "集成测试创建的出口节点组",
            "config": {
                "allowed_protocols": ["tcp", "udp"],
                "exit_config": {
                    "load_balance_strategy": "round_robin",
                    "health_check_interval": 30,
                    "health_check_timeout": 5
                }
            }
        }' \
        "${API_BASE_URL}/node-groups")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" != "200" ]; then
        log_error "创建出口节点组失败，HTTP状态码: $http_code"
        echo "$body"
        return 1
    fi

    EXIT_GROUP_ID=$(echo "$body" | jq -r '.data.id' 2>/dev/null)
    log_info "出口节点组创建成功，ID: $EXIT_GROUP_ID"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
}

# 测试列出节点组
test_list_node_groups() {
    log_info "测试列出节点组..."

    response=$(curl -s -w "\n%{http_code}" -X GET \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        "${API_BASE_URL}/node-groups?page=1&pageSize=10")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" != "200" ]; then
        log_error "列出节点组失败，HTTP状态码: $http_code"
        echo "$body"
        return 1
    fi

    count=$(echo "$body" | jq -r '.data.total' 2>/dev/null)
    log_info "节点组列表获取成功，共 $count 个节点组"
    echo "$body" | jq '.data.list[] | {id, name, type, is_enabled}' 2>/dev/null || echo "$body"
}

# 测试生成部署命令
test_generate_deploy_command() {
    log_info "测试生成部署命令..."

    if [ -z "$ENTRY_GROUP_ID" ]; then
        log_warn "入口节点组ID为空，跳过测试"
        return 0
    fi

    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d '{
            "service_name": "test-node-1",
            "debug_mode": true
        }' \
        "${API_BASE_URL}/node-groups/${ENTRY_GROUP_ID}/nodes")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" != "200" ]; then
        log_error "生成部署命令失败，HTTP状态码: $http_code"
        echo "$body"
        return 1
    fi

    log_info "部署命令生成成功"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
}

# 测试创建隧道（直连模式）
test_create_tunnel_direct() {
    log_info "测试创建隧道（直连模式）..."

    if [ -z "$ENTRY_GROUP_ID" ]; then
        log_warn "入口节点组ID为空，跳过测试"
        return 0
    fi

    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d "{
            \"name\": \"测试隧道-直连\",
            \"description\": \"集成测试创建的直连隧道\",
            \"protocol\": \"tcp\",
            \"entry_group_id\": $ENTRY_GROUP_ID,
            \"listen_host\": \"0.0.0.0\",
            \"listen_port\": 18080,
            \"remote_host\": \"127.0.0.1\",
            \"remote_port\": 8080,
            \"config\": {
                \"load_balance_strategy\": \"round_robin\",
                \"ip_type\": \"auto\",
                \"enable_proxy_protocol\": false,
                \"forward_targets\": []
            }
        }" \
        "${API_BASE_URL}/tunnels")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" != "200" ]; then
        log_error "创建直连隧道失败，HTTP状态码: $http_code"
        echo "$body"
        return 1
    fi

    TUNNEL_ID=$(echo "$body" | jq -r '.data.id' 2>/dev/null)
    log_info "直连隧道创建成功，ID: $TUNNEL_ID"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
}

# 测试创建隧道（带出口节点组）
test_create_tunnel_with_exit() {
    log_info "测试创建隧道（带出口节点组）..."

    if [ -z "$ENTRY_GROUP_ID" ] || [ -z "$EXIT_GROUP_ID" ]; then
        log_warn "节点组ID为空，跳过测试"
        return 0
    fi

    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d "{
            \"name\": \"测试隧道-带出口\",
            \"description\": \"集成测试创建的带出口节点组的隧道\",
            \"protocol\": \"tcp\",
            \"entry_group_id\": $ENTRY_GROUP_ID,
            \"exit_group_id\": $EXIT_GROUP_ID,
            \"listen_host\": \"0.0.0.0\",
            \"listen_port\": 18081,
            \"remote_host\": \"192.168.1.100\",
            \"remote_port\": 22,
            \"config\": {
                \"load_balance_strategy\": \"least_connections\",
                \"ip_type\": \"auto\",
                \"enable_proxy_protocol\": true,
                \"forward_targets\": [
                    {\"host\": \"192.168.1.100\", \"port\": 22, \"weight\": 10},
                    {\"host\": \"192.168.1.101\", \"port\": 22, \"weight\": 5}
                ]
            }
        }" \
        "${API_BASE_URL}/tunnels")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" != "200" ]; then
        log_error "创建带出口隧道失败，HTTP状态码: $http_code"
        echo "$body"
        return 1
    fi

    log_info "带出口隧道创建成功"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
}

# 测试列出隧道
test_list_tunnels() {
    log_info "测试列出隧道..."

    response=$(curl -s -w "\n%{http_code}" -X GET \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        "${API_BASE_URL}/tunnels?page=1&pageSize=10")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" != "200" ]; then
        log_error "列出隧道失败，HTTP状态码: $http_code"
        echo "$body"
        return 1
    fi

    count=$(echo "$body" | jq -r '.data.total' 2>/dev/null)
    log_info "隧道列表获取成功，共 $count 个隧道"
    echo "$body" | jq '.data.items[] | {id, name, protocol, status}' 2>/dev/null || echo "$body"
}

# 主测试流程
main() {
    log_info "开始NodePass-Pro集成测试..."
    echo ""

    check_dependencies
    echo ""

    test_api_connection
    echo ""

    test_login
    echo ""

    test_create_entry_group
    echo ""

    test_create_exit_group
    echo ""

    test_list_node_groups
    echo ""

    test_generate_deploy_command
    echo ""

    test_create_tunnel_direct
    echo ""

    test_create_tunnel_with_exit
    echo ""

    test_list_tunnels
    echo ""

    log_info "所有测试完成！"
}

# 运行测试
main "$@"
