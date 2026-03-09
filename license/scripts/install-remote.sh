#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/nodeox/NodePass-Pro.git}"
BRANCH="${BRANCH:-main}"
INSTALL_DIR="${INSTALL_DIR:-/opt/NodePass-Pro}"
PANEL_PORT="${PANEL_PORT:-8088}"
PANEL_DOMAIN="${PANEL_DOMAIN:-}"
ACME_EMAIL="${ACME_EMAIL:-}"
BOOTSTRAP_ADMIN_USERNAME="${BOOTSTRAP_ADMIN_USERNAME:-admin}"
BOOTSTRAP_ADMIN_EMAIL="${BOOTSTRAP_ADMIN_EMAIL:-admin@example.com}"
BOOTSTRAP_ADMIN_PASSWORD="${BOOTSTRAP_ADMIN_PASSWORD:-}"

is_tty() {
  [[ -t 0 ]]
}

trim() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "$value"
}

normalize_domain() {
  local raw
  raw="$(trim "$1")"
  if [[ -z "$raw" ]]; then
    printf '%s' ""
    return
  fi
  raw="${raw#http://}"
  raw="${raw#https://}"
  raw="${raw%%/*}"
  raw="${raw%%:*}"
  printf '%s' "$raw"
}

read_with_default() {
  local prompt="$1"
  local default="$2"
  local input=""
  read -r -p "${prompt} [${default}]: " input
  input="$(trim "$input")"
  if [[ -z "$input" ]]; then
    printf '%s' "$default"
  else
    printf '%s' "$input"
  fi
}

prompt_password() {
  local pass1=""
  local pass2=""
  while true; do
    read -r -s -p "设置管理员密码（至少 8 位）: " pass1
    echo
    read -r -s -p "再次输入管理员密码: " pass2
    echo
    if [[ "$pass1" != "$pass2" ]]; then
      echo "两次输入不一致，请重试。"
      continue
    fi
    if [[ "${#pass1}" -lt 8 ]]; then
      echo "密码长度不能小于 8 位，请重试。"
      continue
    fi
    BOOTSTRAP_ADMIN_PASSWORD="$pass1"
    return
  done
}

if is_tty; then
  echo "========================================="
  echo " NodePass License 交互式部署向导"
  echo "========================================="
  INSTALL_DIR="$(read_with_default "安装目录" "$INSTALL_DIR")"
  BRANCH="$(read_with_default "仓库分支" "$BRANCH")"
  PANEL_PORT="$(read_with_default "面板端口" "$PANEL_PORT")"

  read -r -p "绑定域名（可选，留空则仅端口访问）: " raw_domain
  PANEL_DOMAIN="$(normalize_domain "${raw_domain:-$PANEL_DOMAIN}")"
  if [[ -n "${PANEL_DOMAIN}" ]]; then
    read -r -p "证书邮箱（可选，建议填写，用于证书到期通知）: " raw_acme_email
    raw_acme_email="$(trim "${raw_acme_email:-}")"
    if [[ -n "${raw_acme_email}" ]]; then
      ACME_EMAIL="${raw_acme_email}"
    fi
  fi

  if [[ -z "$BOOTSTRAP_ADMIN_PASSWORD" ]]; then
    prompt_password
  fi
else
  if [[ -n "${PANEL_DOMAIN}" ]]; then
    PANEL_DOMAIN="$(normalize_domain "$PANEL_DOMAIN")"
  fi
  if [[ -z "${BOOTSTRAP_ADMIN_PASSWORD}" ]]; then
    echo "错误：非交互模式下必须传入 BOOTSTRAP_ADMIN_PASSWORD。"
    echo "示例：BOOTSTRAP_ADMIN_PASSWORD='YourStrongPassword' bash install-remote.sh"
    exit 1
  fi
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "错误：未安装 docker。"
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "错误：未安装 docker compose v2。"
  exit 1
fi

if [[ ! -d "${INSTALL_DIR}/.git" ]]; then
  git clone "${REPO_URL}" "${INSTALL_DIR}"
fi

cd "${INSTALL_DIR}"
git fetch origin "${BRANCH}"
git checkout "${BRANCH}"
git pull --ff-only origin "${BRANCH}"

cd license

if [[ -n "${JWT_SECRET:-}" ]]; then
  JWT_SECRET_VALUE="${JWT_SECRET}"
elif command -v openssl >/dev/null 2>&1; then
  JWT_SECRET_VALUE="$(openssl rand -hex 32)"
else
  JWT_SECRET_VALUE="$(cat /proc/sys/kernel/random/uuid)$(cat /proc/sys/kernel/random/uuid)"
fi

cat > .env.docker <<EOF
PANEL_PORT=${PANEL_PORT}
PANEL_DOMAIN=${PANEL_DOMAIN}
JWT_SECRET=${JWT_SECRET_VALUE}
JWT_EXPIRE_HOURS=${JWT_EXPIRE_HOURS:-24}

BOOTSTRAP_ADMIN_USERNAME=${BOOTSTRAP_ADMIN_USERNAME}
BOOTSTRAP_ADMIN_EMAIL=${BOOTSTRAP_ADMIN_EMAIL}
BOOTSTRAP_ADMIN_PASSWORD=${BOOTSTRAP_ADMIN_PASSWORD}
BOOTSTRAP_RESET_ADMIN_PASSWORD=${BOOTSTRAP_RESET_ADMIN_PASSWORD:-false}

GIN_MODE=${GIN_MODE:-release}
DB_DRIVER=${DB_DRIVER:-sqlite}
DB_DSN=${DB_DSN:-/app/data/license-unified.db}
RELEASE_UPLOAD_DIR=${RELEASE_UPLOAD_DIR:-/app/uploads/releases}

PAYMENT_CALLBACK_STRICT=${PAYMENT_CALLBACK_STRICT:-false}
PAYMENT_CALLBACK_TOLERANCE_SECONDS=${PAYMENT_CALLBACK_TOLERANCE_SECONDS:-300}
PAYMENT_CALLBACK_SECRET_DEFAULT=${PAYMENT_CALLBACK_SECRET_DEFAULT:-}
PAYMENT_CALLBACK_SECRET_MANUAL=${PAYMENT_CALLBACK_SECRET_MANUAL:-}
PAYMENT_CALLBACK_SECRET_ALIPAY=${PAYMENT_CALLBACK_SECRET_ALIPAY:-}
PAYMENT_CALLBACK_SECRET_WECHAT=${PAYMENT_CALLBACK_SECRET_WECHAT:-}

BACKEND_IMAGE=${BACKEND_IMAGE:-nodepass/license-backend:latest}
FRONTEND_IMAGE=${FRONTEND_IMAGE:-nodepass/license-frontend:latest}
EOF

mkdir -p data uploads/releases deploy

compose_args=(-f docker-compose.yml)

if [[ -n "${PANEL_DOMAIN}" ]]; then
  caddy_global=""
  if [[ -n "${ACME_EMAIL}" ]]; then
    caddy_global=$(cat <<EOF
{
  email ${ACME_EMAIL}
}

EOF
)
  fi
  cat > deploy/Caddyfile <<EOF
${caddy_global}${PANEL_DOMAIN} {
  encode gzip
  reverse_proxy license-frontend:80
}
EOF
  mkdir -p deploy/caddy_data deploy/caddy_config
  compose_args+=(-f docker-compose.domain.yml)
fi

docker compose "${compose_args[@]}" up -d --build
docker compose "${compose_args[@]}" ps

echo
echo "部署完成。"
if [[ -n "${PANEL_DOMAIN}" ]]; then
  echo "访问地址: https://${PANEL_DOMAIN}"
  echo "请确认域名 A 记录已解析到当前服务器，并已放行 80/443 端口。"
else
  echo "访问地址: http://<服务器IP>:${PANEL_PORT}"
fi
echo "管理员用户名: ${BOOTSTRAP_ADMIN_USERNAME}"
