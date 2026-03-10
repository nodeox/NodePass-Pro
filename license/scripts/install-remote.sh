#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/nodeox/NodePass-Pro.git}"
BRANCH="${BRANCH:-main}"
INSTALL_DIR="${INSTALL_DIR:-/opt/NodePass-Pro}"
PANEL_PORT="${PANEL_PORT:-8088}"
PANEL_DOMAIN="${PANEL_DOMAIN:-}"
ACME_EMAIL="${ACME_EMAIL:-}"
DEPLOY_WITH_BUILD="${DEPLOY_WITH_BUILD:-false}"
AUTO_BUILD_FALLBACK="${AUTO_BUILD_FALLBACK:-true}"
BOOTSTRAP_ADMIN_USERNAME="${BOOTSTRAP_ADMIN_USERNAME:-admin}"
BOOTSTRAP_ADMIN_EMAIL="${BOOTSTRAP_ADMIN_EMAIL:-admin@example.com}"
BOOTSTRAP_ADMIN_PASSWORD="${BOOTSTRAP_ADMIN_PASSWORD:-}"

SUDO_BIN=""
PKG_MANAGER=""
DOCKER_USE_SUDO=0

command_exists() {
  command -v "$1" >/dev/null 2>&1
}

log_info() {
  echo "[INFO] $*"
}

log_warn() {
  echo "[WARN] $*"
}

log_error() {
  echo "[ERROR] $*"
}

is_tty() {
  [[ -t 0 ]]
}

init_privilege() {
  if [[ "$(id -u)" -eq 0 ]]; then
    SUDO_BIN=""
    return
  fi
  if ! command_exists sudo; then
    log_error "当前用户不是 root 且未安装 sudo，无法自动安装依赖。请先安装 sudo 或切换 root。"
    exit 1
  fi
  SUDO_BIN="sudo"
}

run_privileged() {
  if [[ -n "${SUDO_BIN}" ]]; then
    "${SUDO_BIN}" "$@"
  else
    "$@"
  fi
}

detect_package_manager() {
  if command_exists apt-get; then
    PKG_MANAGER="apt-get"
  elif command_exists dnf; then
    PKG_MANAGER="dnf"
  elif command_exists yum; then
    PKG_MANAGER="yum"
  elif command_exists apk; then
    PKG_MANAGER="apk"
  else
    PKG_MANAGER=""
  fi
}

install_packages() {
  local pkgs=("$@")
  if [[ ${#pkgs[@]} -eq 0 ]]; then
    return
  fi
  if [[ -z "${PKG_MANAGER}" ]]; then
    log_error "无法识别包管理器，不能自动安装: ${pkgs[*]}"
    exit 1
  fi

  log_info "自动安装依赖: ${pkgs[*]}"
  case "${PKG_MANAGER}" in
    apt-get)
      run_privileged apt-get update -y
      run_privileged apt-get install -y "${pkgs[@]}"
      ;;
    dnf)
      run_privileged dnf install -y "${pkgs[@]}"
      ;;
    yum)
      run_privileged yum install -y "${pkgs[@]}"
      ;;
    apk)
      run_privileged apk add --no-cache "${pkgs[@]}"
      ;;
  esac
}

ensure_base_tools() {
  local missing=()
  if ! command_exists git; then
    missing+=("git")
  fi
  if ! command_exists curl && ! command_exists wget; then
    missing+=("curl")
  fi
  if ! command_exists ca-certificates; then
    # 仅用于部分发行版补证书链，命令不存在时忽略。
    :
  fi
  install_packages "${missing[@]}"
}

ensure_docker_installed() {
  if command_exists docker && docker compose version >/dev/null 2>&1; then
    log_info "检测到 Docker 与 Docker Compose。"
    return
  fi

  log_info "未检测到可用 Docker 环境，开始自动安装..."
  ensure_base_tools

  local install_script="/tmp/get-docker.sh"
  if command_exists curl; then
    curl -fsSL https://get.docker.com -o "${install_script}"
  elif command_exists wget; then
    wget -qO "${install_script}" https://get.docker.com
  else
    log_error "无法下载 Docker 安装脚本（缺少 curl/wget）。"
    exit 1
  fi

  run_privileged sh "${install_script}"
  rm -f "${install_script}"

  if command_exists systemctl; then
    run_privileged systemctl enable --now docker || true
  elif command_exists service; then
    run_privileged service docker start || true
  fi

  if ! command_exists docker; then
    log_error "Docker 自动安装后仍不可用，请手动检查。"
    exit 1
  fi
}

prepare_docker_access() {
  if docker info >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    DOCKER_USE_SUDO=0
    return
  fi
  if command_exists sudo && sudo docker info >/dev/null 2>&1 && sudo docker compose version >/dev/null 2>&1; then
    DOCKER_USE_SUDO=1
    if [[ -n "${SUDO_BIN}" ]] && ! id -nG "${USER}" | grep -qw docker; then
      run_privileged usermod -aG docker "${USER}" || true
      log_warn "已尝试将用户 ${USER} 加入 docker 组；本次脚本继续使用 sudo docker。"
    fi
    return
  fi
  log_error "Docker 已安装但当前用户无法使用（docker info 失败）。"
  exit 1
}

docker_compose() {
  if [[ "${DOCKER_USE_SUDO}" -eq 1 ]]; then
    sudo docker compose "$@"
  else
    docker compose "$@"
  fi
}

configure_firewall_ports() {
  local ports=("$@")
  if [[ ${#ports[@]} -eq 0 ]]; then
    return
  fi

  if command_exists ufw; then
    log_info "检测到 ufw，自动放行端口: ${ports[*]}"
    local p
    for p in "${ports[@]}"; do
      run_privileged ufw allow "${p}/tcp" >/dev/null 2>&1 || true
    done
    return
  fi

  if command_exists firewall-cmd; then
    if ! command_exists systemctl || run_privileged systemctl is-active --quiet firewalld; then
      log_info "检测到 firewalld，自动放行端口: ${ports[*]}"
      local p
      for p in "${ports[@]}"; do
        run_privileged firewall-cmd --permanent --add-port="${p}/tcp" >/dev/null 2>&1 || true
      done
      run_privileged firewall-cmd --reload >/dev/null 2>&1 || true
      return
    fi
  fi

  log_warn "未检测到 ufw/firewalld，未自动配置防火墙。请自行放行端口: ${ports[*]}"
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

if ! [[ "${PANEL_PORT}" =~ ^[0-9]+$ ]] || [[ "${PANEL_PORT}" -lt 1 ]] || [[ "${PANEL_PORT}" -gt 65535 ]]; then
  log_error "面板端口无效: ${PANEL_PORT}"
  exit 1
fi

init_privilege
detect_package_manager
ensure_base_tools
ensure_docker_installed
prepare_docker_access

if [[ ! -d "${INSTALL_DIR}/.git" ]]; then
  log_info "克隆仓库到 ${INSTALL_DIR}"
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

if [[ -n "${PANEL_DOMAIN}" ]]; then
  configure_firewall_ports 80 443
else
  configure_firewall_ports "${PANEL_PORT}"
fi

if [[ "${DEPLOY_WITH_BUILD}" == "true" ]]; then
  log_warn "当前为源码编译部署模式（DEPLOY_WITH_BUILD=true）。"
  docker_compose "${compose_args[@]}" up -d --build
else
  log_info "当前为镜像部署模式（默认）。"
  log_info "正在拉取镜像: ${BACKEND_IMAGE:-nodepass/license-backend:latest} / ${FRONTEND_IMAGE:-nodepass/license-frontend:latest}"
  if docker_compose "${compose_args[@]}" pull; then
    docker_compose "${compose_args[@]}" up -d --no-build
  else
    if [[ "${AUTO_BUILD_FALLBACK}" == "true" ]]; then
      log_warn "镜像拉取失败，自动回退到源码编译部署（AUTO_BUILD_FALLBACK=true）。"
      docker_compose "${compose_args[@]}" up -d --build
    else
      log_error "镜像拉取失败，且已禁用自动回退（AUTO_BUILD_FALLBACK=false）。"
      log_error "请检查镜像仓库权限，或设置 DEPLOY_WITH_BUILD=true 进行源码编译部署。"
      exit 1
    fi
  fi
fi
docker_compose "${compose_args[@]}" ps

echo
echo "部署完成。"
if [[ -n "${PANEL_DOMAIN}" ]]; then
  echo "访问地址: https://${PANEL_DOMAIN}"
  echo "请确认域名 A 记录已解析到当前服务器，并已放行 80/443 端口。"
else
  echo "访问地址: http://<服务器IP>:${PANEL_PORT}"
fi
echo "管理员用户名: ${BOOTSTRAP_ADMIN_USERNAME}"
