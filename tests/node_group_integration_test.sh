#!/usr/bin/env bash

set -u
set -o pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
TOKEN="${TOKEN:-}"

PASS_COUNT=0
FAIL_COUNT=0
TOTAL_COUNT=0

ENTRY_GROUP_ID=""
EXIT_GROUP_ID=""
NODE_ID=""
TUNNEL_ID=""

RESPONSE_CODE=""
RESPONSE_BODY=""

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "[FATAL] 缺少依赖命令: $cmd"
    exit 1
  fi
}

print_response() {
  echo "  HTTP: ${RESPONSE_CODE}"
  echo "  响应:"
  echo "${RESPONSE_BODY}" | sed 's/^/    /'
}

api_request() {
  local method="$1"
  local path="$2"
  local data="${3:-}"
  local auth_mode="${4:-auth}"

  local tmp
  tmp="$(mktemp)"

  local -a cmd
  cmd=(curl -sS -X "$method" "${BASE_URL}${path}" -o "$tmp" -w "%{http_code}" -H "Content-Type: application/json")

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

assert_http_code() {
  local expected="$1"
  if [[ "$RESPONSE_CODE" != "$expected" ]]; then
    echo "  断言失败: 期望 HTTP ${expected}，实际 ${RESPONSE_CODE}"
    return 1
  fi
  return 0
}

assert_success_true() {
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

step1_create_entry_group() {
  local payload
  payload='{
    "name": "测试入口组",
    "type": "entry",
    "config": {
      "allowed_protocols": ["tcp", "udp"],
      "port_range": {"start": 10000, "end": 20000},
      "entry_config": {
        "require_exit_group": true,
        "traffic_multiplier": 1.0,
        "dns_load_balance": false
      }
    }
  }'

  api_request "POST" "/api/v1/node-groups" "$payload" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }

  assert_http_code "201" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }

  local group_id
  local group_type
  group_id="$(json_read '.data.id // empty')"
  group_type="$(json_read '.data.type // empty')"

  if [[ -z "$group_id" || ! "$group_id" =~ ^[0-9]+$ ]]; then
    echo "  断言失败: data.id 不存在或无效"
    print_response
    return 1
  fi

  if [[ "$group_type" != "entry" ]]; then
    echo "  断言失败: data.type != entry"
    print_response
    return 1
  fi

  ENTRY_GROUP_ID="$group_id"
  echo "  ENTRY_GROUP_ID=${ENTRY_GROUP_ID}"
  return 0
}

step2_create_exit_group() {
  local payload
  payload='{
    "name": "测试出口组",
    "type": "exit",
    "config": {
      "allowed_protocols": ["tcp", "udp"],
      "port_range": {"start": 10000, "end": 20000},
      "exit_config": {
        "load_balance_strategy": "round_robin",
        "health_check_interval": 30,
        "health_check_timeout": 5
      }
    }
  }'

  api_request "POST" "/api/v1/node-groups" "$payload" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }

  assert_http_code "201" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }

  local group_id
  local group_type
  group_id="$(json_read '.data.id // empty')"
  group_type="$(json_read '.data.type // empty')"

  if [[ -z "$group_id" || ! "$group_id" =~ ^[0-9]+$ ]]; then
    echo "  断言失败: data.id 不存在或无效"
    print_response
    return 1
  fi

  if [[ "$group_type" != "exit" ]]; then
    echo "  断言失败: data.type != exit"
    print_response
    return 1
  fi

  EXIT_GROUP_ID="$group_id"
  echo "  EXIT_GROUP_ID=${EXIT_GROUP_ID}"
  return 0
}

step3_list_node_groups() {
  api_request "GET" "/api/v1/node-groups?page=1&page_size=50" "" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }

  assert_http_code "200" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }

  local count
  count="$(json_read '((.data.items // .data.list // []) | length)')"
  if [[ -z "$count" || ! "$count" =~ ^[0-9]+$ ]]; then
    echo "  断言失败: 无法解析列表长度"
    print_response
    return 1
  fi

  if (( count < 2 )); then
    echo "  断言失败: items 长度 < 2（实际 ${count}）"
    print_response
    return 1
  fi

  echo "  当前节点组数量=${count}"
  return 0
}

step4_get_entry_group_detail() {
  if [[ -z "$ENTRY_GROUP_ID" ]]; then
    echo "  前置失败: ENTRY_GROUP_ID 为空"
    return 1
  fi

  api_request "GET" "/api/v1/node-groups/${ENTRY_GROUP_ID}" "" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }

  assert_http_code "200" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }

  local name
  name="$(json_read '.data.name // empty')"
  if [[ "$name" != "测试入口组" ]]; then
    echo "  断言失败: data.name != 测试入口组（实际: ${name}）"
    print_response
    return 1
  fi

  return 0
}

step5_update_entry_group() {
  if [[ -z "$ENTRY_GROUP_ID" ]]; then
    echo "  前置失败: ENTRY_GROUP_ID 为空"
    return 1
  fi

  local payload='{"description":"更新后的描述"}'

  api_request "PUT" "/api/v1/node-groups/${ENTRY_GROUP_ID}" "$payload" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }

  assert_http_code "200" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }
  return 0
}

step6_generate_deploy_command() {
  if [[ -z "$ENTRY_GROUP_ID" ]]; then
    echo "  前置失败: ENTRY_GROUP_ID 为空"
    return 1
  fi

  local payload='{"service_name":"test-node-1","debug_mode":true}'

  api_request "POST" "/api/v1/node-groups/${ENTRY_GROUP_ID}/generate-deploy-command" "$payload" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }

  assert_http_code "200" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }

  local node_id
  local command
  node_id="$(json_read '.data.node_id // empty')"
  command="$(json_read '.data.command // empty')"

  if [[ -z "$node_id" ]]; then
    echo "  断言失败: data.node_id 不存在"
    print_response
    return 1
  fi

  if [[ -z "$command" || "$command" != *"hub-url"* ]]; then
    echo "  断言失败: data.command 不包含 hub-url"
    print_response
    return 1
  fi

  NODE_ID="$node_id"
  echo "  NODE_ID=${NODE_ID}"
  return 0
}

step7_list_nodes() {
  if [[ -z "$ENTRY_GROUP_ID" ]]; then
    echo "  前置失败: ENTRY_GROUP_ID 为空"
    return 1
  fi

  api_request "GET" "/api/v1/node-groups/${ENTRY_GROUP_ID}/nodes" "" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }

  assert_http_code "200" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }

  local count
  count="$(json_read 'if (.data|type)=="array" then (.data|length) else ((.data.items // .data.list // []) | length) end')"

  if [[ -z "$count" || ! "$count" =~ ^[0-9]+$ ]]; then
    echo "  断言失败: 无法解析节点列表长度"
    print_response
    return 1
  fi

  if (( count < 1 )); then
    echo "  断言失败: 节点列表为空"
    print_response
    return 1
  fi

  echo "  节点实例数量=${count}"
  return 0
}

step8_report_heartbeat() {
  if [[ -z "$NODE_ID" ]]; then
    echo "  前置失败: NODE_ID 为空"
    return 1
  fi

  local payload
  payload="$(cat <<JSON
{
  "node_id": "${NODE_ID}",
  "cpu_usage": 23.5,
  "memory_usage": 45.7,
  "disk_usage": 60.2,
  "bandwidth_in": 123456,
  "bandwidth_out": 654321,
  "connections": 12,
  "system_info": {
    "cpu_usage": 23.5,
    "memory_usage": 45.7,
    "disk_usage": 60.2,
    "bandwidth_in": 123456,
    "bandwidth_out": 654321,
    "connections": 12
  },
  "traffic_stats": {
    "traffic_in": 1234567,
    "traffic_out": 7654321,
    "active_connections": 12
  }
}
JSON
)"

  api_request "POST" "/api/v1/node-instances/heartbeat" "$payload" "noauth" || {
    echo "  请求失败"
    print_response
    return 1
  }

  assert_http_code "200" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }
  return 0
}

step9_toggle_group_status() {
  if [[ -z "$ENTRY_GROUP_ID" ]]; then
    echo "  前置失败: ENTRY_GROUP_ID 为空"
    return 1
  fi

  api_request "GET" "/api/v1/node-groups/${ENTRY_GROUP_ID}" "" "auth" || {
    echo "  请求失败（获取当前状态）"
    print_response
    return 1
  }
  assert_http_code "200" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }

  local before
  before="$(json_read '.data.is_enabled')"

  api_request "POST" "/api/v1/node-groups/${ENTRY_GROUP_ID}/toggle" "{}" "auth" || {
    echo "  请求失败（切换状态）"
    print_response
    return 1
  }

  assert_http_code "200" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }

  local after
  after="$(json_read '.data.is_enabled')"

  if [[ "$before" == "$after" || -z "$before" || -z "$after" ]]; then
    echo "  断言失败: is_enabled 未发生变化（before=${before}, after=${after}）"
    print_response
    return 1
  fi

  echo "  is_enabled: ${before} -> ${after}"
  return 0
}

step10_create_tunnel() {
  if [[ -z "$ENTRY_GROUP_ID" || -z "$EXIT_GROUP_ID" ]]; then
    echo "  前置失败: ENTRY_GROUP_ID 或 EXIT_GROUP_ID 为空"
    return 1
  fi

  local listen_port
  listen_port=$((22000 + (RANDOM % 2000)))

  local payload
  payload="$(cat <<JSON
{
  "name": "测试隧道",
  "entry_group_id": ${ENTRY_GROUP_ID},
  "exit_group_id": ${EXIT_GROUP_ID},
  "protocol": "tcp",
  "remote_host": "192.168.1.100",
  "remote_port": 22,
  "listen_port": ${listen_port}
}
JSON
)"

  api_request "POST" "/api/v1/tunnels" "$payload" "auth" || {
    echo "  请求失败"
    print_response
    return 1
  }

  assert_http_code "201" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }

  local tunnel_id
  tunnel_id="$(json_read '.data.id // empty')"
  if [[ -z "$tunnel_id" || ! "$tunnel_id" =~ ^[0-9]+$ ]]; then
    echo "  断言失败: data.id 不存在或无效"
    print_response
    return 1
  fi

  TUNNEL_ID="$tunnel_id"
  echo "  TUNNEL_ID=${TUNNEL_ID}"
  return 0
}

step11_cleanup_resources() {
  local ok=0

  if [[ -n "$TUNNEL_ID" ]]; then
    api_request "DELETE" "/api/v1/tunnels/${TUNNEL_ID}" "" "auth" || {
      echo "  删除隧道请求失败"
      print_response
      ok=1
    }
    if [[ "$RESPONSE_CODE" != "200" ]]; then
      echo "  删除隧道断言失败: HTTP=${RESPONSE_CODE}"
      print_response
      ok=1
    fi
  else
    echo "  提示: TUNNEL_ID 为空，跳过删除隧道"
  fi

  if [[ -n "$EXIT_GROUP_ID" ]]; then
    api_request "DELETE" "/api/v1/node-groups/${EXIT_GROUP_ID}" "" "auth" || {
      echo "  删除出口组请求失败"
      print_response
      ok=1
    }
    if [[ "$RESPONSE_CODE" != "200" ]]; then
      echo "  删除出口组断言失败: HTTP=${RESPONSE_CODE}"
      print_response
      ok=1
    fi
  else
    echo "  提示: EXIT_GROUP_ID 为空，跳过删除出口组"
  fi

  if [[ -n "$ENTRY_GROUP_ID" ]]; then
    api_request "DELETE" "/api/v1/node-groups/${ENTRY_GROUP_ID}" "" "auth" || {
      echo "  删除入口组请求失败"
      print_response
      ok=1
    }
    if [[ "$RESPONSE_CODE" != "200" ]]; then
      echo "  删除入口组断言失败: HTTP=${RESPONSE_CODE}"
      print_response
      ok=1
    fi
  else
    echo "  提示: ENTRY_GROUP_ID 为空，跳过删除入口组"
  fi

  if [[ "$ok" -ne 0 ]]; then
    return 1
  fi
  return 0
}

main() {
  require_cmd curl
  require_cmd jq

  if [[ -z "$TOKEN" ]]; then
    echo "[FATAL] 缺少 TOKEN 环境变量，请先导出 JWT Token。"
    echo "示例: export TOKEN=eyJ..."
    exit 1
  fi

  echo "== NodeGroup 集成测试 =="
  echo "BASE_URL=${BASE_URL}"
  echo

  run_step "创建入口节点组" step1_create_entry_group
  run_step "创建出口节点组" step2_create_exit_group
  run_step "获取节点组列表" step3_list_node_groups
  run_step "获取入口组详情" step4_get_entry_group_detail
  run_step "更新入口组" step5_update_entry_group
  run_step "生成部署命令" step6_generate_deploy_command
  run_step "获取节点列表" step7_list_nodes
  run_step "模拟心跳上报" step8_report_heartbeat
  run_step "切换节点组状态" step9_toggle_group_status
  run_step "创建隧道" step10_create_tunnel
  run_step "清理资源（删除隧道、节点组）" step11_cleanup_resources

  echo "========== 测试汇总 =========="
  echo "总计: ${TOTAL_COUNT}"
  echo "通过: ${PASS_COUNT}"
  echo "失败: ${FAIL_COUNT}"

  if [[ "$FAIL_COUNT" -gt 0 ]]; then
    exit 1
  fi
  exit 0
}

main "$@"
