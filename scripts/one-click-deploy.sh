#!/usr/bin/env bash
set -euo pipefail

# NodePass Pro 一键部署脚本（默认镜像部署）
# 用法示例：
#   bash scripts/one-click-deploy.sh --license-key NP-XXXX-XXXX --domain panel.example.com --email admin@example.com

INSTALL_SCRIPT_URL="${INSTALL_SCRIPT_URL:-https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh}"
INSTALL_DIR="${INSTALL_DIR:-/opt/NodePass-Pro}"

LICENSE_KEY="${LICENSE_KEY:-}"
DOMAIN="${DOMAIN:-}"
EMAIL="${EMAIL:-}"
BACKEND_DOMAIN="${BACKEND_DOMAIN:-}"
FRONTEND_BIND="${FRONTEND_BIND:-127.0.0.1:5173}"

ADMIN_USERNAME="${ADMIN_USERNAME:-}"
ADMIN_EMAIL="${ADMIN_EMAIL:-}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"

WITH_SOURCE="${WITH_SOURCE:-false}"
BUILD_IMAGE="${BUILD_IMAGE:-false}"
BUILD_NODECLIENT="${BUILD_NODECLIENT:-false}"

log_info() {
  echo "[INFO] $*"
}

log_error() {
  echo "[ERROR] $*" >&2
}

usage() {
  cat <<'USAGE'
NodePass Pro 一键部署脚本

必填参数：
  --license-key <授权码>
  --domain <面板域名>

可选参数：
  --email <邮箱>                 Caddy 证书邮箱
  --backend-domain <域名>        独立后端域名（可选）
  --install-dir <目录>           安装目录（默认 /opt/NodePass-Pro）
  --frontend-bind <地址>         不启用 Caddy 时前端绑定地址（默认 127.0.0.1:5173）
  --admin-username <用户名>      部署后初始化管理员
  --admin-email <邮箱>
  --admin-password <密码>
  --with-source                  保留源码部署
  --build-image                  本地构建镜像
  --build-nodeclient             构建 nodeclient 下载包（建议配合 --with-source）
  -h, --help                     查看帮助

环境变量（可替代参数）：
  LICENSE_KEY DOMAIN EMAIL BACKEND_DOMAIN
  ADMIN_USERNAME ADMIN_EMAIL ADMIN_PASSWORD
  INSTALL_DIR FRONTEND_BIND WITH_SOURCE BUILD_IMAGE BUILD_NODECLIENT
USAGE
}

sanitize_domain() {
  local value="$1"
  value="${value#http://}"
  value="${value#https://}"
  value="${value%%/*}"
  echo "$value"
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --license-key)
        LICENSE_KEY="${2:-}"
        shift 2
        ;;
      --domain)
        DOMAIN="${2:-}"
        shift 2
        ;;
      --email)
        EMAIL="${2:-}"
        shift 2
        ;;
      --backend-domain)
        BACKEND_DOMAIN="${2:-}"
        shift 2
        ;;
      --install-dir)
        INSTALL_DIR="${2:-}"
        shift 2
        ;;
      --frontend-bind)
        FRONTEND_BIND="${2:-}"
        shift 2
        ;;
      --admin-username)
        ADMIN_USERNAME="${2:-}"
        shift 2
        ;;
      --admin-email)
        ADMIN_EMAIL="${2:-}"
        shift 2
        ;;
      --admin-password)
        ADMIN_PASSWORD="${2:-}"
        shift 2
        ;;
      --with-source)
        WITH_SOURCE="true"
        shift
        ;;
      --build-image)
        BUILD_IMAGE="true"
        shift
        ;;
      --build-nodeclient)
        BUILD_NODECLIENT="true"
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

prompt_if_empty() {
  local var_name="$1"
  local prompt_text="$2"
  local current_value="${!var_name}"
  if [[ -n "$current_value" ]]; then
    return
  fi
  if [[ -t 0 ]]; then
    read -r -p "$prompt_text" current_value
    printf -v "$var_name" '%s' "$current_value"
  fi
}

generate_jwt_secret_if_missing() {
  if [[ -n "${JWT_SECRET:-}" ]]; then
    return
  fi

  if command -v openssl >/dev/null 2>&1; then
    JWT_SECRET="$(openssl rand -base64 48 | tr -d '\n')"
  else
    JWT_SECRET="$(LC_ALL=C tr -dc 'A-Za-z0-9!@#$%^&*()_+{}[]:;<>,.?/~-' </dev/urandom | head -c 64)"
  fi
  export JWT_SECRET
  log_info "已自动生成 JWT_SECRET"
}

main() {
  parse_args "$@"

  prompt_if_empty LICENSE_KEY "请输入授权码: "
  prompt_if_empty DOMAIN "请输入面板域名: "

  DOMAIN="$(sanitize_domain "$DOMAIN")"
  BACKEND_DOMAIN="$(sanitize_domain "$BACKEND_DOMAIN")"

  if [[ -z "$LICENSE_KEY" ]]; then
    log_error "授权码不能为空（--license-key）"
    exit 1
  fi
  if [[ -z "$DOMAIN" ]]; then
    log_error "域名不能为空（--domain）"
    exit 1
  fi

  if ! command -v curl >/dev/null 2>&1; then
    log_error "缺少 curl，请先安装"
    exit 1
  fi

  generate_jwt_secret_if_missing

  local install_script
  install_script="$(mktemp)"
  trap 'rm -f "$install_script"' EXIT

  log_info "下载部署引导脚本..."
  curl -fsSL "$INSTALL_SCRIPT_URL" -o "$install_script"

  local args=(
    --non-interactive
    --install
    --install-dir "$INSTALL_DIR"
    --license-key "$LICENSE_KEY"
    --license-domain "$DOMAIN"
    --license-site-url "https://$DOMAIN"
    --with-caddy
    --frontend-domain "$DOMAIN"
  )

  if [[ -n "$EMAIL" ]]; then
    args+=(--email "$EMAIL")
  fi

  if [[ -n "$BACKEND_DOMAIN" ]]; then
    args+=(--backend-domain "$BACKEND_DOMAIN")
  fi

  if [[ -n "$ADMIN_USERNAME" ]]; then
    args+=(--admin-username "$ADMIN_USERNAME")
  fi
  if [[ -n "$ADMIN_EMAIL" ]]; then
    args+=(--admin-email "$ADMIN_EMAIL")
  fi
  if [[ -n "$ADMIN_PASSWORD" ]]; then
    args+=(--admin-password "$ADMIN_PASSWORD")
  fi

  if [[ "$WITH_SOURCE" == "true" ]]; then
    args+=(--with-source)
  fi
  if [[ "$BUILD_IMAGE" == "true" ]]; then
    args+=(--build-image)
  fi
  if [[ "$BUILD_NODECLIENT" == "true" ]]; then
    args+=(--build-nodeclient)
  fi

  log_info "开始一键部署..."
  FRONTEND_BIND="$FRONTEND_BIND" bash "$install_script" "${args[@]}"

  log_info "部署完成。"
  log_info "访问地址: https://$DOMAIN"
}

main "$@"
