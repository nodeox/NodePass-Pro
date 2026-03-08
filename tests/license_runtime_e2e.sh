#!/usr/bin/env bash
# NodePass Pro 运行时授权 E2E 联调（Docker）
#
# 覆盖目标：
# 1) 真实启动 license-center（镜像）并通过 API 生成授权码
# 2) 启动 backend（Docker）并开启运行时授权校验
# 3) 验证授权通过时 /api/v1/license/status=valid
# 4) 验证缺失 domain/site_url 时授权失败且业务接口被拦截

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_COMPOSE_FILE="${ROOT_DIR}/docker-compose.yml"

PROJECT_NAME="${PROJECT_NAME:-nodepass-license-e2e}"
E2E_BACKEND_PORT="${E2E_BACKEND_PORT:-18080}"
E2E_LICENSE_PORT="${E2E_LICENSE_PORT:-18090}"
E2E_DOMAIN="${E2E_DOMAIN:-panel.e2e.example.com}"
E2E_LICENSE_IMAGE="${E2E_LICENSE_IMAGE:-nodepass/license-center:e2e-local}"
E2E_LICENSE_IMAGE_SOURCE="${E2E_LICENSE_IMAGE_SOURCE:-local}" # local / pull
KEEP_STACK="${KEEP_STACK:-false}"

LICENSE_ADMIN_USERNAME="${LICENSE_ADMIN_USERNAME:-admin}"
LICENSE_ADMIN_EMAIL="${LICENSE_ADMIN_EMAIL:-admin@e2e.local}"
LICENSE_ADMIN_PASSWORD="${LICENSE_ADMIN_PASSWORD:-NodePassE2E#2026!}"

PASS_COUNT=0
FAIL_COUNT=0
TOTAL_COUNT=0

LICENSE_TOKEN=""
LICENSE_KEY=""

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

RESPONSE_CODE=""
RESPONSE_BODY=""
TMP_OVERRIDE_DIR=""
TMP_OVERRIDE_FILE=""

log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_step() { echo -e "${BLUE}[STEP]${NC} $*"; }

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    log_error "缺少依赖命令: $cmd"
    exit 1
  fi
}

compose() {
  docker compose \
    --project-name "${PROJECT_NAME}" \
    -f "${BASE_COMPOSE_FILE}" \
    -f "${TMP_OVERRIDE_FILE}" \
    "$@"
}

api_request() {
  local method="$1"
  local url="$2"
  local data="${3:-}"
  local auth_token="${4:-}"

  local tmp
  tmp="$(mktemp)"

  local -a cmd
  cmd=(curl -sS -X "$method" "$url" -o "$tmp" -w "%{http_code}" -H "Content-Type: application/json")

  if [[ -n "$auth_token" ]]; then
    cmd+=(-H "Authorization: Bearer ${auth_token}")
  fi
  if [[ -n "$data" ]]; then
    cmd+=(--data "$data")
  fi

  if ! RESPONSE_CODE="$("${cmd[@]}" 2>&1)"; then
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

assert_http() {
  local expected="$1"
  [[ "${RESPONSE_CODE}" == "${expected}" ]]
}

assert_http_not() {
  local not_expected="$1"
  [[ "${RESPONSE_CODE}" != "${not_expected}" ]]
}

assert_success_true() {
  echo "$RESPONSE_BODY" | jq -e '.success == true' >/dev/null 2>&1
}

assert_success_false() {
  echo "$RESPONSE_BODY" | jq -e '.success == false' >/dev/null 2>&1
}

print_response() {
  echo "  HTTP: ${RESPONSE_CODE}"
  echo "  响应:"
  echo "${RESPONSE_BODY}" | sed 's/^/    /'
}

run_step() {
  local title="$1"
  shift
  TOTAL_COUNT=$((TOTAL_COUNT + 1))
  echo "[${TOTAL_COUNT}] ${title}"
  if "$@"; then
    PASS_COUNT=$((PASS_COUNT + 1))
    echo -e "${GREEN}[PASS]${NC} ${title}"
  else
    FAIL_COUNT=$((FAIL_COUNT + 1))
    echo -e "${RED}[FAIL]${NC} ${title}"
  fi
  echo
}

wait_http_ok() {
  local url="$1"
  local retries="${2:-60}"
  local sleep_seconds="${3:-2}"

  local i
  for ((i=1; i<=retries; i++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$sleep_seconds"
  done
  return 1
}

cleanup() {
  if [[ "${KEEP_STACK}" == "true" ]]; then
    log_warn "KEEP_STACK=true，跳过清理。项目名: ${PROJECT_NAME}"
    return
  fi

  log_step "清理 E2E 容器与卷..."
  compose down -v --remove-orphans >/dev/null 2>&1 || true
  rm -rf "${TMP_OVERRIDE_DIR}" >/dev/null 2>&1 || true
}

prepare_override_file() {
  if TMP_OVERRIDE_DIR="$(mktemp -d "${ROOT_DIR}/tests/.license-e2e.XXXXXX" 2>/dev/null)"; then
    :
  else
    TMP_OVERRIDE_DIR="$(mktemp -d -t nodepass-license-e2e)"
  fi
  TMP_OVERRIDE_FILE="${TMP_OVERRIDE_DIR}/compose.override.yml"

  cat > "${TMP_OVERRIDE_FILE}" <<'YAML'
services:
  postgres:
    ports: []

  redis:
    ports: []

  backend:
    ports:
      - "${E2E_BACKEND_PORT:-18080}:8080"
    depends_on:
      license-center-postgres:
        condition: service_healthy
      license-center-e2e:
        condition: service_healthy

  license-center-postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: nodepass_license
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d nodepass_license"]
      interval: 10s
      timeout: 5s
      retries: 20
      start_period: 10s

  license-center-e2e:
    image: ${E2E_LICENSE_IMAGE}
    depends_on:
      license-center-postgres:
        condition: service_healthy
    ports:
      - "${E2E_LICENSE_PORT:-18090}:8090"
    volumes:
      - ./license-center/configs/config.yaml:/app/configs/config.yaml:ro
    environment:
      TZ: Asia/Shanghai
      GIN_MODE: release
      LICENSE_SERVER_PORT: "8090"
      LICENSE_SERVER_MODE: "release"
      LICENSE_DATABASE_TYPE: "postgres"
      LICENSE_DATABASE_HOST: "license-center-postgres"
      LICENSE_DATABASE_PORT: "5432"
      LICENSE_DATABASE_USER: "postgres"
      LICENSE_DATABASE_PASSWORD: "postgres"
      LICENSE_DATABASE_DB_NAME: "nodepass_license"
      LICENSE_DATABASE_DSN: "host=license-center-postgres port=5432 user=postgres password=postgres dbname=nodepass_license sslmode=disable TimeZone=UTC"
      LICENSE_JWT_SECRET: "${E2E_LICENSE_JWT_SECRET}"
      LICENSE_ADMIN_USERNAME: "${E2E_LICENSE_ADMIN_USERNAME}"
      LICENSE_ADMIN_EMAIL: "${E2E_LICENSE_ADMIN_EMAIL}"
      LICENSE_ADMIN_PASSWORD: "${E2E_LICENSE_ADMIN_PASSWORD}"
      LICENSE_REDIS_ENABLED: "false"
      LICENSE_SECURITY_SIGNATURE_ENABLED: "false"
      LICENSE_SECURITY_IP_WHITELIST_ENABLED: "false"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8090/health"]
      interval: 15s
      timeout: 5s
      retries: 20
      start_period: 20s
YAML
}

prepare_license_image() {
  case "${E2E_LICENSE_IMAGE_SOURCE}" in
    local)
      log_step "构建本地授权中心镜像: ${E2E_LICENSE_IMAGE}"
      docker build \
        --build-arg BUILD_VERSION=e2e-local \
        -t "${E2E_LICENSE_IMAGE}" \
        "${ROOT_DIR}/license-center" >/dev/null
      ;;
    pull)
      log_step "拉取授权中心镜像: ${E2E_LICENSE_IMAGE}"
      docker pull "${E2E_LICENSE_IMAGE}" >/dev/null
      ;;
    *)
      log_error "未知 E2E_LICENSE_IMAGE_SOURCE: ${E2E_LICENSE_IMAGE_SOURCE}（仅支持 local/pull）"
      exit 1
      ;;
  esac
}

step_start_license_center() {
  log_step "启动 license-center 及其数据库..."
  compose up -d license-center-postgres license-center-e2e >/dev/null
  wait_http_ok "http://127.0.0.1:${E2E_LICENSE_PORT}/health" 90 2
}

step_license_login() {
  local payload
  payload="$(cat <<JSON
{"username":"${LICENSE_ADMIN_USERNAME}","password":"${LICENSE_ADMIN_PASSWORD}"}
JSON
)"

  api_request "POST" "http://127.0.0.1:${E2E_LICENSE_PORT}/api/v1/auth/login" "$payload" || {
    print_response
    return 1
  }

  assert_http "200" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }

  LICENSE_TOKEN="$(json_read '.data.token // empty')"
  [[ -n "${LICENSE_TOKEN}" ]]
}

step_generate_license_key() {
  api_request "GET" "http://127.0.0.1:${E2E_LICENSE_PORT}/api/v1/plans" "" "${LICENSE_TOKEN}" || {
    print_response
    return 1
  }
  assert_http "200" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }

  local plan_id
  plan_id="$(json_read '.data[0].id // empty')"
  if [[ -z "${plan_id}" || ! "${plan_id}" =~ ^[0-9]+$ ]]; then
    print_response
    return 1
  fi

  local payload
  payload="$(cat <<JSON
{"plan_id":${plan_id},"customer":"NodePass E2E","count":1,"prefix":"NP"}
JSON
)"
  api_request "POST" "http://127.0.0.1:${E2E_LICENSE_PORT}/api/v1/licenses/generate" "$payload" "${LICENSE_TOKEN}" || {
    print_response
    return 1
  }
  assert_http "200" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }

  LICENSE_KEY="$(json_read '.data[0].key // empty')"
  [[ -n "${LICENSE_KEY}" ]]
}

step_start_backend_with_valid_domain() {
  log_step "启动 backend（启用运行时授权，带有效 domain）..."

  JWT_SECRET="${E2E_BACKEND_JWT_SECRET}" \
  BACKEND_LICENSE_ENABLED=true \
  LICENSE_VERIFY_URL="http://license-center-e2e:8090/api/v1/license/verify" \
  LICENSE_KEY="${LICENSE_KEY}" \
  BACKEND_LICENSE_DOMAIN="${E2E_DOMAIN}" \
  BACKEND_LICENSE_SITE_URL="https://${E2E_DOMAIN}" \
  E2E_BACKEND_PORT="${E2E_BACKEND_PORT}" \
  E2E_LICENSE_PORT="${E2E_LICENSE_PORT}" \
  compose up -d postgres redis backend >/dev/null

  wait_http_ok "http://127.0.0.1:${E2E_BACKEND_PORT}/health" 120 2
}

step_verify_runtime_license_ok() {
  api_request "GET" "http://127.0.0.1:${E2E_BACKEND_PORT}/api/v1/license/status" || {
    print_response
    return 1
  }
  assert_http "200" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }
  echo "$RESPONSE_BODY" | jq -e '.data.enabled == true and .data.valid == true' >/dev/null 2>&1
}

step_verify_business_not_blocked() {
  local payload='{"account":"nonexistent","password":"wrong-password"}'
  api_request "POST" "http://127.0.0.1:${E2E_BACKEND_PORT}/api/v1/auth/login" "$payload" || {
    print_response
    return 1
  }

  assert_http_not "403" || { print_response; return 1; }
  if echo "$RESPONSE_BODY" | jq -e '.error.code == "LICENSE_INVALID"' >/dev/null 2>&1; then
    print_response
    return 1
  fi
  return 0
}

step_restart_backend_without_domain() {
  log_step "重建 backend（故意不传 domain/site_url）验证阻断..."

  JWT_SECRET="${E2E_BACKEND_JWT_SECRET}" \
  BACKEND_LICENSE_ENABLED=true \
  LICENSE_VERIFY_URL="http://license-center-e2e:8090/api/v1/license/verify" \
  LICENSE_KEY="${LICENSE_KEY}" \
  BACKEND_LICENSE_DOMAIN="" \
  BACKEND_LICENSE_SITE_URL="" \
  E2E_BACKEND_PORT="${E2E_BACKEND_PORT}" \
  E2E_LICENSE_PORT="${E2E_LICENSE_PORT}" \
  compose up -d --force-recreate backend >/dev/null

  wait_http_ok "http://127.0.0.1:${E2E_BACKEND_PORT}/health" 120 2
}

step_verify_runtime_license_failed() {
  api_request "GET" "http://127.0.0.1:${E2E_BACKEND_PORT}/api/v1/license/status" || {
    print_response
    return 1
  }
  assert_http "200" || { print_response; return 1; }
  assert_success_true || { print_response; return 1; }
  echo "$RESPONSE_BODY" | jq -e '.data.enabled == true and .data.valid == false' >/dev/null 2>&1 || {
    print_response
    return 1
  }
  echo "$RESPONSE_BODY" | jq -e '(.data.message // "") | test("domain/site_url")' >/dev/null 2>&1 || {
    print_response
    return 1
  }
}

step_verify_business_blocked() {
  local payload='{"account":"nonexistent","password":"wrong-password"}'
  api_request "POST" "http://127.0.0.1:${E2E_BACKEND_PORT}/api/v1/auth/login" "$payload" || {
    print_response
    return 1
  }
  assert_http "403" || { print_response; return 1; }
  assert_success_false || { print_response; return 1; }
  echo "$RESPONSE_BODY" | jq -e '.error.code == "LICENSE_INVALID"' >/dev/null 2>&1 || {
    print_response
    return 1
  }
}

main() {
  require_cmd docker
  require_cmd curl
  require_cmd jq
  require_cmd openssl

  if [[ ! -f "${BASE_COMPOSE_FILE}" ]]; then
    log_error "未找到 compose 文件: ${BASE_COMPOSE_FILE}"
    exit 1
  fi

  export E2E_BACKEND_PORT
  export E2E_LICENSE_PORT
  export E2E_LICENSE_IMAGE
  export E2E_LICENSE_JWT_SECRET="${E2E_LICENSE_JWT_SECRET:-$(openssl rand -hex 24)}"
  export E2E_BACKEND_JWT_SECRET="${E2E_BACKEND_JWT_SECRET:-$(openssl rand -hex 24)}"
  export E2E_LICENSE_ADMIN_USERNAME="${LICENSE_ADMIN_USERNAME}"
  export E2E_LICENSE_ADMIN_EMAIL="${LICENSE_ADMIN_EMAIL}"
  export E2E_LICENSE_ADMIN_PASSWORD="${LICENSE_ADMIN_PASSWORD}"

  prepare_license_image
  prepare_override_file
  trap cleanup EXIT

  run_step "启动 license-center（Docker）" step_start_license_center
  run_step "license-center 管理员登录" step_license_login
  run_step "生成可用授权码" step_generate_license_key
  run_step "启动 backend（有效域名）" step_start_backend_with_valid_domain
  run_step "验证运行时授权状态为通过" step_verify_runtime_license_ok
  run_step "验证业务接口未被授权拦截" step_verify_business_not_blocked
  run_step "重启 backend（缺失域名配置）" step_restart_backend_without_domain
  run_step "验证运行时授权状态为失败" step_verify_runtime_license_failed
  run_step "验证业务接口被授权拦截" step_verify_business_blocked

  echo "=============================================="
  echo "总计: ${TOTAL_COUNT}, 通过: ${PASS_COUNT}, 失败: ${FAIL_COUNT}"
  if [[ "${FAIL_COUNT}" -gt 0 ]]; then
    exit 1
  fi
}

main "$@"
