#!/usr/bin/env bash
# NodePass-Pro 集成测试脚本（节点组新链路）

set -u
set -o pipefail

API_BASE_URL="${API_BASE_URL:-http://localhost:8080/api/v1}"
ADMIN_ACCOUNT="${ADMIN_ACCOUNT:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"

TOKEN="${TOKEN:-}"
ENTRY_GROUP_ID=""
EXIT_GROUP_ID=""
RELATION_ID=""
NODE_ID=""
NODE_TOKEN=""
TUNNEL_ID=""

PASS_COUNT=0
FAIL_COUNT=0
TOTAL_COUNT=0

RESPONSE_CODE=""
RESPONSE_BODY=""

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() {
  echo -e "${GREEN}[INFO]${NC} $*"
}

warn() {
  echo -e "${YELLOW}[WARN]${NC} $*"
}

err() {
  echo -e "${RED}[ERROR]${NC} $*"
}

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    err "缺少依赖命令: $cmd"
    exit 1
  fi
}

api_request() {
  local method="$1"
  local path="$2"
  local data="${3:-}"
  local auth_mode="${4:-auth}"

  local tmp
  tmp="$(mktemp)"

  local -a cmd
  cmd=(curl -sS -X "$method" "${API_BASE_URL}${path}" -o "$tmp" -w "%{http_code}" -H "Content-Type: application/json")

  if [[ "$auth_mode" == "auth" ]]; then
    cmd+=(-H "Authorization: Bearer ${TOKEN}")
  fi
  if [[ -n "$data" ]]; then
    cmd+=(--data "$data")
  fi

  if ! RESPONSE_CODE="$(${cmd[@]} 2>&1)"; then
    RESPONSE_BODY="$RESPONSE_CODE"
    RESPONSE_CODE="000"
    rm -f "$tmp"
    return 1
  fi

  RESPONSE_BODY="$(cat "$tmp")"
  rm -f "$tmp"
  return 0
}

json_read() {
  local expr="$1"
  echo "$RESPONSE_BODY" | jq -r "$expr" 2>/dev/null
}

print_response() {
  echo "  HTTP: ${RESPONSE_CODE}"
  echo "  响应:"
  echo "${RESPONSE_BODY}" | sed 's/^/    /'
}

assert_http() {
  local expected="$1"
  if [[ "$RESPONSE_CODE" != "$expected" ]]; then
    echo "  断言失败: 期望 HTTP ${expected}，实际 ${RESPONSE_CODE}"
    return 1
  fi
  return 0
}

assert_success() {
  if ! echo "$RESPONSE_BODY" | jq -e '.success == true' >/dev/null 2>&1; then
    echo "  断言失败: success != true"
    return 1
  fi
  return 0
}

run_step() {
  local title="$1"
  shift

  TOTAL_COUNT=$((TOTAL_COUNT + 1))
  echo "[$TOTAL_COUNT] ${title}"
  if "$@"; then
    PASS_COUNT=$((PASS_COUNT + 1))
    echo "[PASS] ${title}"
  else
    FAIL_COUNT=$((FAIL_COUNT + 1))
    echo "[FAIL] ${title}"
  fi
  echo
}

step0_health() {
  local health_url="${API_BASE_URL%/api/v1}/health"
  RESPONSE_CODE="$(curl -sS -o /tmp/nodepass_health.$$ -w "%{http_code}" "$health_url" || true)"
  RESPONSE_BODY="$(cat /tmp/nodepass_health.$$ 2>/dev/null || true)"
  rm -f /tmp/nodepass_health.$$

  assert_http "200" || { print_response; return 1; }
  return 0
}

step1_login() {
  local payload
  payload="$(cat <<JSON
{"account":"${ADMIN_ACCOUNT}","password":"${ADMIN_PASSWORD}"}
JSON
)"

  api_request "POST" "/auth/login" "$payload" "noauth" || {
    echo "  请求失败"
    print_response
    return 1
  }

  assert_http "200" || { print_response; return 1; }
  assert_success || { print_response; return 1; }

  TOKEN="$(json_read '.data.token // empty')"
  if [[ -z "$TOKEN" ]]; then
    echo "  断言失败: token 为空"
    print_response
    return 1
  fi
  return 0
}

step2_create_entry_group() {
  local payload='{
    "name": "集成测试入口组",
    "type": "entry",
    "description": "integration_test.sh",
    "config": {
      "allowed_protocols": ["tcp", "udp"],
      "port_range": {"start": 12000, "end": 22000},
      "entry_config": {
        "require_exit_group": true,
        "traffic_multiplier": 1.0,
        "dns_load_balance": false
      }
    }
  }'

  api_request "POST" "/node-groups" "$payload" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }
  assert_http "201" || { print_response; return 1; }
  assert_success || { print_response; return 1; }

  ENTRY_GROUP_ID="$(json_read '.data.id // empty')"
  if [[ -z "$ENTRY_GROUP_ID" || ! "$ENTRY_GROUP_ID" =~ ^[0-9]+$ ]]; then
    echo "  断言失败: ENTRY_GROUP_ID 无效"
    print_response
    return 1
  fi
  return 0
}

step3_create_exit_group() {
  local payload='{
    "name": "集成测试出口组",
    "type": "exit",
    "description": "integration_test.sh",
    "config": {
      "allowed_protocols": ["tcp", "udp"],
      "port_range": {"start": 12000, "end": 22000},
      "exit_config": {
        "load_balance_strategy": "round_robin",
        "health_check_interval": 30,
        "health_check_timeout": 5
      }
    }
  }'

  api_request "POST" "/node-groups" "$payload" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }
  assert_http "201" || { print_response; return 1; }
  assert_success || { print_response; return 1; }

  EXIT_GROUP_ID="$(json_read '.data.id // empty')"
  if [[ -z "$EXIT_GROUP_ID" || ! "$EXIT_GROUP_ID" =~ ^[0-9]+$ ]]; then
    echo "  断言失败: EXIT_GROUP_ID 无效"
    print_response
    return 1
  fi
  return 0
}

step4_create_relation() {
  if [[ -z "$ENTRY_GROUP_ID" || -z "$EXIT_GROUP_ID" ]]; then
    echo "  前置失败: group id 缺失"
    return 1
  fi

  local payload="{\"exit_group_id\": ${EXIT_GROUP_ID}}"
  api_request "POST" "/node-groups/${ENTRY_GROUP_ID}/relations" "$payload" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }
  assert_http "201" || { print_response; return 1; }
  assert_success || { print_response; return 1; }

  RELATION_ID="$(json_read '.data.id // empty')"
  return 0
}

step5_list_groups() {
  api_request "GET" "/node-groups?page=1&page_size=20" "" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }
  assert_http "200" || { print_response; return 1; }
  assert_success || { print_response; return 1; }

  local count
  count="$(json_read '(.data.items // [] | length)')"
  if [[ -z "$count" || ! "$count" =~ ^[0-9]+$ ]]; then
    echo "  断言失败: 节点组数量解析失败"
    print_response
    return 1
  fi
  return 0
}

step6_generate_deploy_command() {
  if [[ -z "$ENTRY_GROUP_ID" ]]; then
    echo "  前置失败: ENTRY_GROUP_ID 为空"
    return 1
  fi

  local payload='{"service_name":"integration-node-1","debug_mode":true}'
  api_request "POST" "/node-groups/${ENTRY_GROUP_ID}/generate-deploy-command" "$payload" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }
  assert_http "200" || { print_response; return 1; }
  assert_success || { print_response; return 1; }

  NODE_ID="$(json_read '.data.node_id // empty')"
  local command
  command="$(json_read '.data.command // empty')"
  if [[ -z "$NODE_ID" || -z "$command" ]]; then
    echo "  断言失败: 部署信息不完整"
    print_response
    return 1
  fi

  NODE_TOKEN="$(echo "$command" | sed -n "s/.*--token '\\([^']*\\)'.*/\\1/p")"
  if [[ -z "$NODE_TOKEN" ]]; then
    echo "  断言失败: 未能从命令中解析 token"
    print_response
    return 1
  fi
  return 0
}

step7_list_nodes() {
  api_request "GET" "/node-groups/${ENTRY_GROUP_ID}/nodes" "" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }
  assert_http "200" || { print_response; return 1; }
  assert_success || { print_response; return 1; }
  return 0
}

step8_heartbeat() {
  if [[ -z "$NODE_ID" || -z "$NODE_TOKEN" ]]; then
    echo "  前置失败: NODE_ID/NODE_TOKEN 缺失"
    return 1
  fi

  local ts nonce payload tmp
  ts="$(date +%s)"
  nonce="it-$(date +%s%N)-${RANDOM}"
  payload="$(cat <<JSON
{
  "node_id": "${NODE_ID}",
  "token": "${NODE_TOKEN}",
  "connection_address": "203.0.113.10",
  "system_info": {
    "cpu_usage": 12.5,
    "memory_usage": 44.8,
    "disk_usage": 58.1,
    "bandwidth_in": 12345,
    "bandwidth_out": 67890,
    "connections": 6
  },
  "traffic_stats": {
    "traffic_in": 12345,
    "traffic_out": 67890,
    "active_connections": 6
  }
}
JSON
)"

  tmp="$(mktemp)"
  RESPONSE_CODE="$(curl -sS -o "$tmp" -w "%{http_code}" \
    -X POST "${API_BASE_URL}/node-instances/heartbeat" \
    -H "Content-Type: application/json" \
    -H "X-Timestamp: ${ts}" \
    -H "X-Nonce: ${nonce}" \
    --data "$payload" 2>&1 || true)"
  RESPONSE_BODY="$(cat "$tmp")"
  rm -f "$tmp"

  assert_http "200" || { print_response; return 1; }
  assert_success || { print_response; return 1; }
  return 0
}

step9_create_tunnel() {
  if [[ -z "$ENTRY_GROUP_ID" || -z "$EXIT_GROUP_ID" ]]; then
    echo "  前置失败: group id 缺失"
    return 1
  fi

  local listen_port payload
  listen_port=$((22000 + (RANDOM % 2000)))
  payload="$(cat <<JSON
{
  "name": "集成测试隧道",
  "entry_group_id": ${ENTRY_GROUP_ID},
  "exit_group_id": ${EXIT_GROUP_ID},
  "protocol": "tcp",
  "remote_host": "192.168.1.100",
  "remote_port": 22,
  "listen_port": ${listen_port}
}
JSON
)"

  api_request "POST" "/tunnels" "$payload" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }
  assert_http "201" || { print_response; return 1; }
  assert_success || { print_response; return 1; }

  TUNNEL_ID="$(json_read '.data.id // empty')"
  if [[ -z "$TUNNEL_ID" ]]; then
    echo "  断言失败: TUNNEL_ID 为空"
    print_response
    return 1
  fi
  return 0
}

step10_cleanup() {
  local failed=0

  if [[ -n "$TUNNEL_ID" ]]; then
    api_request "DELETE" "/tunnels/${TUNNEL_ID}" "" "auth" || failed=1
    [[ "$RESPONSE_CODE" == "200" ]] || failed=1
  fi

  if [[ -n "$RELATION_ID" ]]; then
    api_request "DELETE" "/node-group-relations/${RELATION_ID}" "" "auth" || failed=1
    [[ "$RESPONSE_CODE" == "200" ]] || failed=1
  fi

  if [[ -n "$EXIT_GROUP_ID" ]]; then
    api_request "DELETE" "/node-groups/${EXIT_GROUP_ID}" "" "auth" || failed=1
    [[ "$RESPONSE_CODE" == "200" ]] || failed=1
  fi

  if [[ -n "$ENTRY_GROUP_ID" ]]; then
    api_request "DELETE" "/node-groups/${ENTRY_GROUP_ID}" "" "auth" || failed=1
    [[ "$RESPONSE_CODE" == "200" ]] || failed=1
  fi

  if [[ "$failed" -ne 0 ]]; then
    print_response
    return 1
  fi
  return 0
}

main() {
  require_cmd curl
  require_cmd jq

  info "开始 NodePass-Pro 集成测试"
  run_step "健康检查" step0_health
  run_step "管理员登录（account/password）" step1_login
  run_step "创建入口节点组" step2_create_entry_group
  run_step "创建出口节点组" step3_create_exit_group
  run_step "创建节点组关联" step4_create_relation
  run_step "获取节点组列表（page_size）" step5_list_groups
  run_step "生成部署命令（新路径）" step6_generate_deploy_command
  run_step "获取节点实例列表" step7_list_nodes
  run_step "节点心跳（含防重放头）" step8_heartbeat
  run_step "创建隧道" step9_create_tunnel
  run_step "清理资源" step10_cleanup

  echo "========== 测试汇总 =========="
  echo "总计: ${TOTAL_COUNT}"
  echo "通过: ${PASS_COUNT}"
  echo "失败: ${FAIL_COUNT}"

  if [[ "$FAIL_COUNT" -gt 0 ]]; then
    exit 1
  fi
  info "全部通过"
}

main "$@"
