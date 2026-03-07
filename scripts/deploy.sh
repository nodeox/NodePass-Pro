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

log_info() {
  echo "[INFO] $*"
}

log_warn() {
  echo "[WARN] $*"
}

log_error() {
  echo "[ERROR] $*" >&2
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
  --down                          停止并移除当前部署
  -h, --help                      显示帮助

环境变量:
  BACKEND_CONFIG_FILE             挂载到 backend 容器的配置文件路径
                                  默认: ./backend/configs/config.docker.yaml
  FRONTEND_BIND                   前端绑定地址，默认: 127.0.0.1:5173

示例:
  ./scripts/deploy.sh
  ./scripts/deploy.sh --with-caddy --frontend-domain panel.example.com --email admin@example.com
  ./scripts/deploy.sh --with-caddy --frontend-domain panel.example.com --backend-domain api.example.com
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

run_compose() {
  BACKEND_CONFIG_FILE="${BACKEND_CONFIG_FILE:-./backend/configs/config.docker.yaml}" \
    FRONTEND_BIND="${FRONTEND_BIND:-127.0.0.1:5173}" \
    CADDY_HTTP_PORT="$CADDY_HTTP_PORT" \
    CADDY_HTTPS_PORT="$CADDY_HTTPS_PORT" \
    "$@"
}

main() {
  parse_args "$@"
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

  local up_args=("-d")
  if [[ "$NO_BUILD" == false ]]; then
    up_args+=("--build")
  fi

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
