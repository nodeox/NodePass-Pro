#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"

WITH_CADDY=false
DOMAIN=""
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
  --with-caddy                 启用 Caddy 反向代理（可选）
  --domain <域名>              Caddy 站点域名（启用 Caddy 时必填）
  --email <邮箱>               Caddy ACME 邮箱（可选）
  --caddy-http-port <端口>     Caddy HTTP 端口，默认 80
  --caddy-https-port <端口>    Caddy HTTPS 端口，默认 443
  --no-build                   启动时不执行镜像构建
  --down                       停止并移除当前部署
  -h, --help                   显示帮助

示例:
  ./scripts/deploy.sh
  ./scripts/deploy.sh --with-caddy --domain panel.example.com --email admin@example.com
  ./scripts/deploy.sh --with-caddy --domain panel.example.com --caddy-http-port 8080 --caddy-https-port 8443
  ./scripts/deploy.sh --down
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

generate_caddyfile() {
  local domain="$1"
  local email="$2"
  local template_file="${ROOT_DIR}/deploy/caddy/Caddyfile.template"
  local output_file="${ROOT_DIR}/deploy/caddy/Caddyfile"
  local email_block=""

  if [[ -n "$email" ]]; then
    email_block="    email ${email}"
  fi

  if [[ ! -f "$template_file" ]]; then
    log_error "未找到模板文件: $template_file"
    exit 1
  fi

  sed \
    -e "s|__DOMAIN__|${domain}|g" \
    -e "s|__EMAIL_BLOCK__|${email_block}|g" \
    "$template_file" >"$output_file"

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
        DOMAIN="${2:-}"
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

main() {
  parse_args "$@"
  require_command docker

  if ! docker info >/dev/null 2>&1; then
    log_error "Docker 未运行，请先启动 Docker。"
    exit 1
  fi

  local compose_files=("-f" "${ROOT_DIR}/docker-compose.yml")
  if [[ "$WITH_CADDY" == true ]]; then
    local sanitized_domain
    sanitized_domain="$(sanitize_domain "$DOMAIN")"
    if [[ -z "$sanitized_domain" ]]; then
      log_error "启用 Caddy 时必须通过 --domain 提供有效域名。"
      exit 1
    fi
    DOMAIN="$sanitized_domain"
    generate_caddyfile "$DOMAIN" "$EMAIL"
    compose_files+=("-f" "${ROOT_DIR}/docker-compose.caddy.yml")
  fi

  local compose_cmd=(docker compose "${compose_files[@]}")

  if [[ "$DOWN" == true ]]; then
    log_info "开始停止服务..."
    CADDY_HTTP_PORT="$CADDY_HTTP_PORT" CADDY_HTTPS_PORT="$CADDY_HTTPS_PORT" \
      "${compose_cmd[@]}" down
    log_info "服务已停止。"
    exit 0
  fi

  local up_args=("-d")
  if [[ "$NO_BUILD" == false ]]; then
    up_args+=("--build")
  fi

  log_info "开始部署服务..."
  CADDY_HTTP_PORT="$CADDY_HTTP_PORT" CADDY_HTTPS_PORT="$CADDY_HTTPS_PORT" \
    "${compose_cmd[@]}" up "${up_args[@]}"

  log_info "部署完成，当前服务状态:"
  CADDY_HTTP_PORT="$CADDY_HTTP_PORT" CADDY_HTTPS_PORT="$CADDY_HTTPS_PORT" \
    "${compose_cmd[@]}" ps

  if [[ "$WITH_CADDY" == true ]]; then
    log_info "访问地址: https://${DOMAIN}"
    log_warn "请确认域名 DNS 已解析到当前服务器。"
  else
    log_info "访问地址: http://127.0.0.1:5173"
  fi
}

main "$@"
