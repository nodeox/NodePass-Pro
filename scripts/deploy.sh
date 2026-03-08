#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"

WITH_CADDY=false
LEGACY_DOMAIN=""
FRONTEND_DOMAIN=""
BACKEND_DOMAIN=""
EMAIL=""
DOWN=false
NO_BUILD=false
CADDY_HTTP_PORT=80
CADDY_HTTPS_PORT=443
AUTO_BUILD_NODECLIENT=true
PANEL_VERSION="${PANEL_VERSION:-}"
BACKEND_VERSION="${BACKEND_VERSION:-}"
FRONTEND_VERSION="${FRONTEND_VERSION:-}"
NODECLIENT_VERSION="${NODECLIENT_VERSION:-}"
LICENSE_KEY="${LICENSE_KEY:-${NODEPASS_LICENSE_KEY:-}}"
LICENSE_VERIFY_URL="${LICENSE_VERIFY_URL:-https://license.nodepass.pro/api/v1/license/verify}"
LICENSE_MACHINE_ID="${LICENSE_MACHINE_ID:-}"
LICENSE_ACTION="${LICENSE_ACTION:-install}"
LICENSE_VERIFIED="${LICENSE_VERIFIED:-false}"
BACKEND_LICENSE_ENABLED="${BACKEND_LICENSE_ENABLED:-}"
BACKEND_LICENSE_VERIFY_INTERVAL="${BACKEND_LICENSE_VERIFY_INTERVAL:-300}"
BACKEND_LICENSE_FAIL_OPEN="${BACKEND_LICENSE_FAIL_OPEN:-false}"
BACKEND_LICENSE_OFFLINE_GRACE_SECONDS="${BACKEND_LICENSE_OFFLINE_GRACE_SECONDS:-600}"
BACKEND_LICENSE_DOMAIN="${BACKEND_LICENSE_DOMAIN:-}"
BACKEND_LICENSE_SITE_URL="${BACKEND_LICENSE_SITE_URL:-}"

log_info() {
  echo "[INFO] $*"
}

log_warn() {
  echo "[WARN] $*"
}

log_error() {
  echo "[ERROR] $*" >&2
}

read_version_file() {
  local version_file="$1"
  local default_value="$2"
  if [[ -f "$version_file" ]]; then
    local value
    value="$(tr -d '[:space:]' <"$version_file")"
    if [[ -n "$value" ]]; then
      echo "$value"
      return
    fi
  fi
  echo "$default_value"
}

detect_machine_id() {
  if [[ -n "$LICENSE_MACHINE_ID" ]]; then
    echo "$LICENSE_MACHINE_ID"
    return
  fi
  if [[ -f /etc/machine-id ]]; then
    tr -d '[:space:]' </etc/machine-id
    return
  fi
  if [[ -f /var/lib/dbus/machine-id ]]; then
    tr -d '[:space:]' </var/lib/dbus/machine-id
    return
  fi
  hostname
}

load_versions() {
  if [[ -z "$PANEL_VERSION" ]]; then
    PANEL_VERSION="$(read_version_file "${ROOT_DIR}/VERSION" "dev")"
  fi
  if [[ -z "$BACKEND_VERSION" ]]; then
    BACKEND_VERSION="$(read_version_file "${ROOT_DIR}/backend/VERSION" "$PANEL_VERSION")"
  fi
  if [[ -z "$FRONTEND_VERSION" ]]; then
    FRONTEND_VERSION="$(read_version_file "${ROOT_DIR}/frontend/VERSION" "$PANEL_VERSION")"
  fi
  if [[ -z "$NODECLIENT_VERSION" ]]; then
    NODECLIENT_VERSION="$(read_version_file "${ROOT_DIR}/nodeclient/VERSION" "$PANEL_VERSION")"
  fi
}

print_version_summary() {
  log_info "当前版本: panel=${PANEL_VERSION}, backend=${BACKEND_VERSION}, frontend=${FRONTEND_VERSION}, nodeclient=${NODECLIENT_VERSION}"
}

usage() {
  cat <<'EOF'
NodePass Pro 一键部署脚本

用法:
  ./scripts/deploy.sh [选项]

选项:
  --with-caddy                    启用 Caddy 反向代理（可选）
  --domain <域名>                 兼容参数，等价于 --frontend-domain
  --frontend-domain <域名>        前端访问域名（启用 Caddy 时必填）
  --backend-domain <域名>         后端访问域名（启用 Caddy 时可选）
  --email <邮箱>                  Caddy ACME 邮箱（可选）
  --caddy-http-port <端口>        Caddy HTTP 端口，默认 80
  --caddy-https-port <端口>       Caddy HTTPS 端口，默认 443
  --no-build                      启动时不执行镜像构建
  --skip-nodeclient-build         跳过 nodeclient 二进制自动构建
  --license-key <授权码>          授权码（非 down 模式必填）
  --license-server <URL>          授权验证接口（默认: https://license.nodepass.pro/api/v1/license/verify）
  --machine-id <ID>               指定机器标识（可选，默认自动检测）
  --license-domain <域名>         运行时授权域名（启用运行时授权时建议设置）
  --license-site-url <URL>        运行时授权站点地址（可选）
  --down                          停止并移除当前部署
  -h, --help                      显示帮助

环境变量:
  BACKEND_CONFIG_FILE             挂载到 backend 容器的配置文件路径
                                  默认: ./backend/configs/config.docker.yaml
  FRONTEND_BIND                   前端绑定地址，默认: 127.0.0.1:5173
  AUTO_BUILD_NODECLIENT           是否自动构建 nodeclient 下载包（默认: true）
  BACKEND_VERSION                 覆盖后端构建版本（默认读取 backend/VERSION）
  FRONTEND_VERSION                覆盖前端构建版本（默认读取 frontend/VERSION）
  LICENSE_KEY                     授权码（可替代 --license-key）
  LICENSE_VERIFY_URL              授权验证接口（可替代 --license-server）
  BACKEND_LICENSE_DOMAIN          运行时授权域名（启用运行时授权时建议设置）
  BACKEND_LICENSE_SITE_URL        运行时授权站点地址（可选）

示例:
  ./scripts/deploy.sh --license-key NP-XXXX-XXXX
  ./scripts/deploy.sh --license-key NP-XXXX-XXXX --license-domain panel.example.com --license-site-url https://panel.example.com
  ./scripts/deploy.sh --license-key NP-XXXX-XXXX --with-caddy --frontend-domain panel.example.com --email admin@example.com
  ./scripts/deploy.sh --license-key NP-XXXX-XXXX --with-caddy --frontend-domain panel.example.com --backend-domain api.example.com
  ./scripts/deploy.sh --skip-nodeclient-build
  BACKEND_CONFIG_FILE=./backend/configs/config.runtime.yaml ./scripts/deploy.sh --with-caddy --frontend-domain panel.example.com
EOF
}

sanitize_domain() {
  local input="$1"
  input="${input#http://}"
  input="${input#https://}"
  input="${input%%/*}"
  echo "$input"
}

is_local_domain() {
  local host="${1,,}"
  if [[ -z "$host" ]]; then
    return 0
  fi
  if [[ "$host" == "localhost" || "$host" == "::1" || "$host" == "0.0.0.0" ]]; then
    return 0
  fi
  if [[ "$host" == 127.* ]]; then
    return 0
  fi
  if [[ "$host" == *.local || "$host" == *.test ]]; then
    return 0
  fi
  return 1
}

is_usable_license_domain() {
  local host
  host="$(sanitize_domain "$1")"
  if [[ -z "$host" || "$host" == \*.* ]]; then
    return 1
  fi
  if is_local_domain "$host"; then
    return 1
  fi
  return 0
}

resolve_license_runtime_settings() {
  local frontend_domain="$1"
  local backend_domain="$2"

  if [[ -n "$BACKEND_LICENSE_DOMAIN" ]]; then
    BACKEND_LICENSE_DOMAIN="$(sanitize_domain "$BACKEND_LICENSE_DOMAIN")"
  elif [[ -n "$backend_domain" ]]; then
    BACKEND_LICENSE_DOMAIN="$backend_domain"
  elif [[ -n "$frontend_domain" ]]; then
    BACKEND_LICENSE_DOMAIN="$frontend_domain"
  fi

  if [[ -n "$BACKEND_LICENSE_DOMAIN" && -z "$BACKEND_LICENSE_SITE_URL" ]]; then
    local scheme="https"
    if [[ "$WITH_CADDY" != true ]]; then
      scheme="http"
    fi
    BACKEND_LICENSE_SITE_URL="${scheme}://${BACKEND_LICENSE_DOMAIN}"
  fi
}

require_command() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    log_error "缺少命令: $cmd"
    exit 1
  fi
}

build_backend_block() {
  local backend_domain="$1"

  if [[ -z "$backend_domain" ]]; then
    echo ""
    return
  fi

  cat <<EOF
${backend_domain} {
    encode zstd gzip

    reverse_proxy backend:8080

    log {
        output stdout
        format console
    }
}
EOF
}

generate_caddyfile() {
  local frontend_domain="$1"
  local backend_domain="$2"
  local email="$3"
  local template_file="${ROOT_DIR}/deploy/caddy/Caddyfile.template"
  local output_file="${ROOT_DIR}/deploy/caddy/Caddyfile"
  local email_block=""
  local backend_block=""

  if [[ -n "$email" ]]; then
    email_block="    email ${email}"
  fi

  backend_block="$(build_backend_block "$backend_domain")"

  if [[ ! -f "$template_file" ]]; then
    log_error "未找到模板文件: $template_file"
    exit 1
  fi

  awk \
    -v frontend_domain="$frontend_domain" \
    -v email_block="$email_block" \
    -v backend_block="$backend_block" \
    '
      {
        gsub(/__FRONTEND_DOMAIN__/, frontend_domain)
        gsub(/__EMAIL_BLOCK__/, email_block)
        if ($0 == "__BACKEND_BLOCK__") {
          print backend_block
        } else {
          print
        }
      }
    ' "$template_file" >"$output_file"

  log_info "已生成 Caddy 配置: $output_file"
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --with-caddy)
        WITH_CADDY=true
        shift
        ;;
      --domain)
        LEGACY_DOMAIN="${2:-}"
        shift 2
        ;;
      --frontend-domain)
        FRONTEND_DOMAIN="${2:-}"
        shift 2
        ;;
      --backend-domain)
        BACKEND_DOMAIN="${2:-}"
        shift 2
        ;;
      --email)
        EMAIL="${2:-}"
        shift 2
        ;;
      --caddy-http-port)
        CADDY_HTTP_PORT="${2:-}"
        shift 2
        ;;
      --caddy-https-port)
        CADDY_HTTPS_PORT="${2:-}"
        shift 2
        ;;
      --down)
        DOWN=true
        shift
        ;;
      --no-build)
        NO_BUILD=true
        shift
        ;;
      --skip-nodeclient-build)
        AUTO_BUILD_NODECLIENT=false
        shift
        ;;
      --license-key)
        LICENSE_KEY="${2:-}"
        shift 2
        ;;
      --license-server)
        LICENSE_VERIFY_URL="${2:-}"
        shift 2
        ;;
      --machine-id)
        LICENSE_MACHINE_ID="${2:-}"
        shift 2
        ;;
      --license-domain)
        BACKEND_LICENSE_DOMAIN="${2:-}"
        shift 2
        ;;
      --license-site-url)
        BACKEND_LICENSE_SITE_URL="${2:-}"
        shift 2
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        log_error "未知参数: $1"
        usage
        exit 1
        ;;
    esac
  done
}

verify_license_or_exit() {
  if [[ "$LICENSE_VERIFIED" == "true" ]]; then
    log_info "检测到上游已完成授权校验，跳过重复验证。"
    return
  fi

  if [[ -z "$LICENSE_KEY" ]]; then
    log_error "缺少授权码，请使用 --license-key 或环境变量 LICENSE_KEY/NODEPASS_LICENSE_KEY。"
    exit 1
  fi

  local verify_script="${ROOT_DIR}/scripts/license-verify.py"
  if [[ ! -f "$verify_script" ]]; then
    log_error "未找到授权验证脚本: $verify_script"
    exit 1
  fi

  require_command python3
  local machine_id
  machine_id="$(detect_machine_id)"

  log_info "开始授权校验..."
  local verify_output
  if ! verify_output="$(python3 "$verify_script" \
    --verify-url "$LICENSE_VERIFY_URL" \
    --license-key "$LICENSE_KEY" \
    --machine-id "$machine_id" \
    --action "$LICENSE_ACTION" \
    --panel-version "$PANEL_VERSION" \
    --backend-version "$BACKEND_VERSION" \
    --frontend-version "$FRONTEND_VERSION" \
    --nodeclient-version "$NODECLIENT_VERSION" \
    --timeout 20 2>&1)"; then
    log_error "授权校验失败: $verify_output"
    exit 1
  fi

  local license_id license_plan
  license_id="$(python3 -c 'import json,sys; d=json.loads(sys.argv[1]); print(d.get("license_id",""))' "$verify_output" 2>/dev/null || true)"
  license_plan="$(python3 -c 'import json,sys; d=json.loads(sys.argv[1]); print(d.get("plan",""))' "$verify_output" 2>/dev/null || true)"
  log_info "授权校验通过。license_id=${license_id:-unknown}, plan=${license_plan:-unknown}"
}

run_compose() {
  BACKEND_CONFIG_FILE="${BACKEND_CONFIG_FILE:-./backend/configs/config.docker.yaml}" \
    FRONTEND_BIND="${FRONTEND_BIND:-127.0.0.1:5173}" \
    BACKEND_VERSION="${BACKEND_VERSION}" \
    FRONTEND_VERSION="${FRONTEND_VERSION}" \
    PANEL_VERSION="${PANEL_VERSION}" \
    NODECLIENT_VERSION="${NODECLIENT_VERSION}" \
    LICENSE_KEY="${LICENSE_KEY}" \
    LICENSE_VERIFY_URL="${LICENSE_VERIFY_URL}" \
    LICENSE_MACHINE_ID="${LICENSE_MACHINE_ID}" \
    BACKEND_LICENSE_ENABLED="${BACKEND_LICENSE_ENABLED}" \
    BACKEND_LICENSE_VERIFY_INTERVAL="${BACKEND_LICENSE_VERIFY_INTERVAL}" \
    BACKEND_LICENSE_FAIL_OPEN="${BACKEND_LICENSE_FAIL_OPEN}" \
    BACKEND_LICENSE_OFFLINE_GRACE_SECONDS="${BACKEND_LICENSE_OFFLINE_GRACE_SECONDS}" \
    BACKEND_LICENSE_DOMAIN="${BACKEND_LICENSE_DOMAIN}" \
    BACKEND_LICENSE_SITE_URL="${BACKEND_LICENSE_SITE_URL}" \
    CADDY_HTTP_PORT="$CADDY_HTTP_PORT" \
    CADDY_HTTPS_PORT="$CADDY_HTTPS_PORT" \
    "$@"
}

build_nodeclient_downloads_if_needed() {
  local build_script="${ROOT_DIR}/scripts/build-nodeclient-downloads.sh"
  local auto_build="${AUTO_BUILD_NODECLIENT:-true}"
  if [[ "${auto_build}" != "true" ]]; then
    log_info "已跳过 nodeclient 构建（AUTO_BUILD_NODECLIENT=false）。"
    return
  fi
  if [[ ! -x "$build_script" ]]; then
    log_error "未找到可执行构建脚本: $build_script"
    exit 1
  fi
  log_info "开始自动构建 nodeclient 下载包..."
  "$build_script"
}

main() {
  parse_args "$@"
  load_versions
  print_version_summary
  require_command docker

  if ! docker info >/dev/null 2>&1; then
    log_error "Docker 未运行或当前用户无权限访问 Docker。"
    exit 1
  fi

  local compose_files=("-f" "${ROOT_DIR}/docker-compose.yml")
  local sanitized_frontend_domain=""
  local sanitized_backend_domain=""

  if [[ "$WITH_CADDY" == true ]]; then
    if [[ -z "$FRONTEND_DOMAIN" && -n "$LEGACY_DOMAIN" ]]; then
      FRONTEND_DOMAIN="$LEGACY_DOMAIN"
    fi

    sanitized_frontend_domain="$(sanitize_domain "$FRONTEND_DOMAIN")"
    sanitized_backend_domain="$(sanitize_domain "$BACKEND_DOMAIN")"

    if [[ -z "$sanitized_frontend_domain" ]]; then
      log_error "启用 Caddy 时必须通过 --frontend-domain（或 --domain）提供前端域名。"
      exit 1
    fi

    generate_caddyfile "$sanitized_frontend_domain" "$sanitized_backend_domain" "$EMAIL"
    compose_files+=("-f" "${ROOT_DIR}/docker-compose.caddy.yml")
  fi

  local compose_cmd=(docker compose "${compose_files[@]}")

  if [[ "$DOWN" == true ]]; then
    log_info "开始停止服务..."
    run_compose "${compose_cmd[@]}" down
    log_info "服务已停止。"
    exit 0
  fi

  verify_license_or_exit
  resolve_license_runtime_settings "$sanitized_frontend_domain" "$sanitized_backend_domain"

  if [[ -z "${BACKEND_LICENSE_ENABLED}" ]]; then
    if is_usable_license_domain "$BACKEND_LICENSE_DOMAIN"; then
      BACKEND_LICENSE_ENABLED="true"
      log_info "检测到可用域名 ${BACKEND_LICENSE_DOMAIN}，默认启用运行时授权。"
    else
      BACKEND_LICENSE_ENABLED="false"
      log_warn "未检测到可用生产域名，默认关闭运行时授权（避免业务接口被误拦截）。"
    fi
  fi

  if [[ "${BACKEND_LICENSE_ENABLED,,}" == "true" ]]; then
    if ! is_usable_license_domain "$BACKEND_LICENSE_DOMAIN"; then
      log_error "BACKEND_LICENSE_ENABLED=true，但未提供可用域名。请设置 --license-domain 或 BACKEND_LICENSE_DOMAIN。"
      exit 1
    fi
    if [[ -z "$BACKEND_LICENSE_SITE_URL" ]]; then
      BACKEND_LICENSE_SITE_URL="https://${BACKEND_LICENSE_DOMAIN}"
    fi
    log_info "运行时授权域名: ${BACKEND_LICENSE_DOMAIN}"
    log_info "运行时授权站点: ${BACKEND_LICENSE_SITE_URL}"
  fi

  local up_args=("-d")
  if [[ "$NO_BUILD" == false ]]; then
    up_args+=("--build")
  fi

  build_nodeclient_downloads_if_needed

  log_info "开始部署服务..."
  run_compose "${compose_cmd[@]}" up "${up_args[@]}"

  log_info "部署完成，当前服务状态:"
  run_compose "${compose_cmd[@]}" ps

  if [[ "$WITH_CADDY" == true ]]; then
    log_info "前端访问地址: https://${sanitized_frontend_domain}"
    if [[ -n "$sanitized_backend_domain" ]]; then
      log_info "后端访问地址: https://${sanitized_backend_domain}"
    else
      log_info "后端访问路径: https://${sanitized_frontend_domain}/api/v1"
    fi
    log_warn "请确认域名 DNS 已解析到当前服务器。"
  else
    log_info "前端访问地址: http://${FRONTEND_BIND:-127.0.0.1:5173}"
  fi
}

main "$@"
