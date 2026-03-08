#!/usr/bin/env bash
set -euo pipefail

# ============================================================================
# NodePass License Center 一键部署脚本 v0.4.0
# ============================================================================

REPO_URL="${REPO_URL:-https://github.com/nodeox/NodePass-Pro.git}"
BRANCH="${BRANCH:-main}"
INSTALL_DIR="${INSTALL_DIR:-/opt/nodepass-license-center}"
PROJECT_SUBDIR="${PROJECT_SUBDIR:-license-center}"
ACTION="install" # install / upgrade / uninstall
BACKUP_DIR="${BACKUP_DIR:-${INSTALL_DIR%/}-backups}"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
LAST_CONFIG_BACKUP=""
GENERATED_CONFIG_WARNING=false

# 镜像相关配置（默认使用 GitHub Actions 产出的多架构镜像）
USE_PREBUILT_IMAGE=true
IMAGE_URL=""
IMAGE_FILE=""
IMAGE_NAME="ghcr.io/nodeox/license-center"
IMAGE_VERSION="main"

SUDO_CMD=""
PKG_MANAGER=""
RUN_DEPLOY_AS_SUDO=false
SKIP_HEALTH_CHECK=false
INTERACTIVE_MODE="auto" # auto / true / false

# 交互式部署参数
APP_BIND="0.0.0.0"
APP_PORT="8090"
CADDY_ENABLED=false
CADDY_DOMAIN=""
CADDY_EMAIL=""
CADDY_HTTP_PORT="80"
CADDY_HTTPS_PORT="443"
ADMIN_USERNAME=""
ADMIN_EMAIL=""
ADMIN_PASSWORD=""
JWT_SECRET=""

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

usage() {
  cat <<'USAGE'
NodePass License Center 一键部署脚本 v0.4.0（交互式）

用法:
  bash install.sh [--install|--upgrade|--uninstall] [选项]

操作:
  --install                    安装（默认）
  --upgrade                    升级到最新版本
  --uninstall                  完全卸载

选项:
  --repo <url>                 仓库地址（默认: https://github.com/nodeox/NodePass-Pro.git）
  --branch <branch>            分支（默认: main）
  --install-dir <dir>          安装目录（默认: /opt/nodepass-license-center）
  --project-subdir <name>      子目录（默认: license-center）
  --skip-health-check          跳过健康检查
  --interactive                强制交互式安装向导
  --non-interactive            关闭交互向导
  --use-image                  使用预构建镜像（默认）
  --build-source               使用源码构建（覆盖默认镜像部署）
  --image-url <url>            镜像下载地址
  --image-file <file>          本地镜像文件路径
  --image-name <name>          镜像名称（默认: ghcr.io/nodeox/license-center）
  --image-version <ver>        镜像版本（默认: main，多架构）
  --app-port <port>            服务端口（默认: 8090）
  --app-bind <addr>            服务绑定地址（默认: 0.0.0.0）
  --enable-caddy               启用域名反代 + 自动证书
  --domain <domain>            绑定域名（启用 Caddy 时必填）
  --cert-email <email>         证书邮箱（可选）
  --admin-username <name>      管理员用户名
  --admin-email <email>        管理员邮箱
  --admin-password <pwd>       管理员密码
  --jwt-secret <secret>        JWT 密钥（32 位以上）
  -h, --help                   显示帮助

示例:
  # 交互式安装（默认镜像部署）
  bash install.sh --install

  # 升级
  bash install.sh --upgrade

  # 卸载
  bash install.sh --uninstall

  # 自定义安装目录
  bash install.sh --install --install-dir /data/license-center

  # 显式使用镜像安装
  bash install.sh --install --use-image

  # 切换为源码构建安装
  bash install.sh --install --build-source

  # 从 URL 下载镜像安装
  bash install.sh --install --image-url https://example.com/license-center-latest.tar.gz

  # 从本地文件安装
  bash install.sh --install --image-file /path/to/license-center-latest.tar.gz

  # 一键启用域名 HTTPS
  bash install.sh --install --enable-caddy --domain license.example.com --cert-email ops@example.com

远程一键安装:
  bash <(curl -fsSL "https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license-center/install.sh?t=$(date +%s)") --install

版本: v0.4.0
功能: 授权管理、域名绑定、监控告警、Webhook 通知、Web 管理界面
USAGE
}

log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_step() { echo -e "${BLUE}[STEP]${NC} $*"; }

run_root() {
  if [[ -n "$SUDO_CMD" ]]; then
    "$SUDO_CMD" "$@"
  else
    "$@"
  fi
}

sanitize_domain() {
  local input="${1:-}"
  input="${input#http://}"
  input="${input#https://}"
  input="${input%%/*}"
  input="${input%%:*}"
  echo "${input,,}"
}

prompt_with_default() {
  local prompt="$1"
  local default_value="${2:-}"
  local input=""
  read -r -p "${prompt} [${default_value}]: " input
  if [[ -z "$input" ]]; then
    echo "$default_value"
  else
    echo "$input"
  fi
}

prompt_yes_no() {
  local prompt="$1"
  local default_value="${2:-y}" # y / n
  local input=""
  local show_default="y/N"
  if [[ "$default_value" == "y" ]]; then
    show_default="Y/n"
  fi

  while true; do
    read -r -p "${prompt} (${show_default}): " input
    input="${input:-$default_value}"
    case "${input,,}" in
      y|yes) return 0 ;;
      n|no) return 1 ;;
      *) echo "请输入 y 或 n" ;;
    esac
  done
}

prompt_password_with_confirm() {
  local prompt="$1"
  local allow_empty="${2:-false}"
  local password=""
  local confirm=""
  while true; do
    read -r -s -p "${prompt}: " password
    echo ""
    if [[ -z "$password" && "$allow_empty" == "true" ]]; then
      echo ""
      return
    fi
    read -r -s -p "确认${prompt}: " confirm
    echo ""
    if [[ "$password" != "$confirm" ]]; then
      echo "两次输入不一致，请重试。"
      continue
    fi
    if [[ ${#password} -lt 12 ]]; then
      echo "密码长度至少 12 位。"
      continue
    fi
    echo "$password"
    return
  done
}

port_in_use() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -iTCP:"$port" -sTCP:LISTEN -Pn >/dev/null 2>&1
    return $?
  fi
  if command -v ss >/dev/null 2>&1; then
    ss -lnt | awk '{print $4}' | grep -Eq "[.:]${port}$"
    return $?
  fi
  return 1
}

find_available_port() {
  local start_port="$1"
  local end_port=$((start_port + 200))
  local port="$start_port"
  while [[ "$port" -le "$end_port" ]]; do
    if ! port_in_use "$port"; then
      echo "$port"
      return
    fi
    port=$((port + 1))
  done
  echo "$start_port"
}

read_env_value() {
  local env_file="$1"
  local key="$2"
  if [[ ! -f "$env_file" ]]; then
    return 0
  fi
  grep -E "^${key}=" "$env_file" | tail -n 1 | cut -d'=' -f2- || true
}

upsert_env_value() {
  local env_file="$1"
  local key="$2"
  local value="$3"
  local tmp_file
  tmp_file="$(mktemp)"
  awk -v key="$key" -v value="$value" '
    BEGIN { updated=0 }
    $0 ~ ("^" key "=") {
      print key "=" value
      updated=1
      next
    }
    { print }
    END {
      if (!updated) {
        print key "=" value
      }
    }
  ' "$env_file" >"$tmp_file"
  run_root mv "$tmp_file" "$env_file"
}

should_run_interactive() {
  if [[ "$ACTION" == "uninstall" ]]; then
    return 1
  fi
  if [[ "$INTERACTIVE_MODE" == "true" ]]; then
    return 0
  fi
  if [[ "$INTERACTIVE_MODE" == "false" ]]; then
    return 1
  fi
  [[ -t 0 ]]
}

detect_host_ip() {
  if command -v ip >/dev/null 2>&1; then
    ip route get 1.1.1.1 2>/dev/null | awk '/src/ {for(i=1;i<=NF;i++) if($i=="src"){print $(i+1); exit}}'
    return
  fi
  if command -v hostname >/dev/null 2>&1; then
    hostname -I 2>/dev/null | awk '{print $1}'
    return
  fi
  echo ""
}

detect_sudo() {
  if [[ "${EUID}" -eq 0 ]]; then
    SUDO_CMD=""
    return
  fi

  if command -v sudo >/dev/null 2>&1; then
    SUDO_CMD="sudo"
  else
    SUDO_CMD=""
  fi
}

detect_pkg_manager() {
  if command -v apt-get >/dev/null 2>&1; then
    PKG_MANAGER="apt"
  elif command -v dnf >/dev/null 2>&1; then
    PKG_MANAGER="dnf"
  elif command -v yum >/dev/null 2>&1; then
    PKG_MANAGER="yum"
  elif command -v pacman >/dev/null 2>&1; then
    PKG_MANAGER="pacman"
  elif command -v zypper >/dev/null 2>&1; then
    PKG_MANAGER="zypper"
  else
    PKG_MANAGER=""
  fi
}

generate_random_hex() {
  local bytes="${1:-32}"
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex "$bytes"
    return
  fi

  if command -v xxd >/dev/null 2>&1; then
    head -c "$bytes" /dev/urandom | xxd -p -c 256
    return
  fi

  # 最后兜底：仅在极简系统中使用
  head -c "$bytes" /dev/urandom | od -An -tx1 | tr -d ' \n'
}

extract_yaml_value() {
  local file="$1"
  local section="$2"
  local key="$3"

  awk -v section="$section" -v key="$key" '
    $0 ~ "^[[:space:]]*" section ":[[:space:]]*$" { in_section=1; next }
    in_section && $0 ~ "^[^[:space:]]" { in_section=0 }
    in_section && $0 ~ "^[[:space:]]*" key ":[[:space:]]*" {
      value=$0
      sub("^[[:space:]]*" key ":[[:space:]]*", "", value)
      gsub(/^[\"\047]|[\"\047]$/, "", value)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
      print value
      exit
    }
  ' "$file"
}

update_yaml_value() {
  local file="$1"
  local section="$2"
  local key="$3"
  local value="$4"
  local tmp_file

  tmp_file="$(mktemp)"
  awk -v section="$section" -v key="$key" -v value="$value" '
    BEGIN {
      in_section=0
      section_seen=0
      updated=0
      escaped_value=value
      gsub(/\\/, "\\\\", escaped_value)
      gsub(/"/, "\\\"", escaped_value)
      gsub(/\r/, "", escaped_value)
    }
    {
      if ($0 ~ "^[[:space:]]*" section ":[[:space:]]*$") {
        section_seen=1
        in_section=1
        print $0
        next
      }

      if (in_section && $0 ~ "^[^[:space:]]") {
        if (!updated) {
          print "  " key ": \"" escaped_value "\""
          updated=1
        }
        in_section=0
      }

      if (in_section && $0 ~ "^[[:space:]]*" key ":[[:space:]]*") {
        print "  " key ": \"" escaped_value "\""
        updated=1
        next
      }

      print $0
    }
    END {
      if (in_section && !updated) {
        print "  " key ": \"" escaped_value "\""
        updated=1
      }
      if (!updated) {
        if (!section_seen) {
          print section ":"
        }
        print "  " key ": \"" escaped_value "\""
      }
    }
  ' "$file" > "$tmp_file"

  run_root mv "$tmp_file" "$file"
}

ensure_file_readable() {
  local file="$1"
  if [[ -f "$file" ]]; then
    # 容器内进程默认非 root，绑定挂载配置必须至少全局可读。
    run_root chmod 644 "$file" || true
  fi
}

ensure_dir_traversable() {
  local dir="$1"
  if [[ -d "$dir" ]]; then
    # 目录需要 x 权限才可遍历访问文件。
    run_root chmod 755 "$dir" || true
  fi
}

ensure_config_mount_permissions() {
  local config_file="$1"
  local config_dir project_dir

  config_dir="$(dirname "$config_file")"
  project_dir="$(dirname "$config_dir")"

  ensure_dir_traversable "$project_dir"
  ensure_dir_traversable "$config_dir"
  ensure_file_readable "$config_file"
}

ensure_secure_config() {
  local config_file="$1"

  if [[ ! -f "$config_file" ]]; then
    log_error "配置文件不存在: $config_file"
    exit 1
  fi

  local jwt_secret
  jwt_secret="$(extract_yaml_value "$config_file" "jwt" "secret" || true)"
  if [[ -z "$jwt_secret" || "$jwt_secret" == "change-this-license-secret" ]]; then
    local generated_jwt
    generated_jwt="$(generate_random_hex 32)"
    update_yaml_value "$config_file" "jwt" "secret" "$generated_jwt"
    GENERATED_CONFIG_WARNING=true
    log_warn "检测到 jwt.secret 未配置或弱默认值，已自动生成强随机密钥。"
  fi

  local admin_password
  admin_password="$(extract_yaml_value "$config_file" "admin" "password" || true)"
  if [[ -z "$admin_password" || "$admin_password" == "ChangeMe123!" || ${#admin_password} -lt 12 ]]; then
    local generated_admin_password
    generated_admin_password="Np!$(generate_random_hex 10)Aa1"
    update_yaml_value "$config_file" "admin" "password" "$generated_admin_password"
    GENERATED_CONFIG_WARNING=true
    log_warn "检测到 admin.password 缺失或强度不足，已自动生成强密码。"
  fi

  ensure_config_mount_permissions "$config_file"
}

install_packages() {
  if [[ -z "$PKG_MANAGER" ]]; then
    log_error "无法识别包管理器，请手动安装依赖后重试。"
    exit 1
  fi

  case "$PKG_MANAGER" in
    apt)
      run_root apt-get update -y
      run_root apt-get install -y "$@"
      ;;
    dnf)
      run_root dnf install -y "$@"
      ;;
    yum)
      run_root yum install -y "$@"
      ;;
    pacman)
      run_root pacman -Sy --noconfirm "$@"
      ;;
    zypper)
      run_root zypper install -y "$@"
      ;;
  esac
}

ensure_cmd_installable() {
  local cmd="$1"
  shift
  if command -v "$cmd" >/dev/null 2>&1; then
    return
  fi
  log_info "未检测到 $cmd，开始自动安装..."
  install_packages "$@"
}

install_docker_engine() {
  if command -v docker >/dev/null 2>&1; then
    return
  fi

  if [[ "$(uname -s)" != "Linux" ]]; then
    log_error "自动安装 Docker 仅支持 Linux，请先手动安装 Docker。"
    exit 1
  fi

  log_info "未检测到 docker，开始自动安装 Docker Engine..."
  if [[ -n "$SUDO_CMD" ]]; then
    curl -fsSL https://get.docker.com | sudo sh
  else
    curl -fsSL https://get.docker.com | sh
  fi
}

ensure_docker_service() {
  if command -v systemctl >/dev/null 2>&1; then
    run_root systemctl enable --now docker >/dev/null 2>&1 || true
  fi
}

check_system_requirements() {
  log_step "检查系统要求..."

  # 检查操作系统
  local os_type=$(uname -s)
  log_info "操作系统: $os_type"

  # 检查内存
  if command -v free >/dev/null 2>&1; then
    local total_mem=$(free -m | awk '/^Mem:/{print $2}')
    log_info "总内存: ${total_mem}MB"
    if [[ $total_mem -lt 1024 ]]; then
      log_warn "内存不足 1GB，可能影响性能"
    fi
  fi

  # 检查磁盘空间
  local available_space=$(df -BG "$INSTALL_DIR" 2>/dev/null | awk 'NR==2 {print $4}' | sed 's/G//')
  if [[ -n "$available_space" ]] && [[ $available_space -lt 5 ]]; then
    log_warn "磁盘空间不足 5GB，可能影响运行"
  fi
}

ensure_env() {
  log_step "检查并安装依赖..."

  detect_sudo
  detect_pkg_manager

  ensure_cmd_installable git git
  ensure_cmd_installable curl curl

  install_docker_engine
  ensure_docker_service

  if ! docker compose version >/dev/null 2>&1; then
    if run_root docker compose version >/dev/null 2>&1; then
      RUN_DEPLOY_AS_SUDO=true
    else
      log_error "缺少 docker compose，请安装 Docker Compose 插件后重试。"
      exit 1
    fi
  fi

  if ! docker info >/dev/null 2>&1; then
    if run_root docker info >/dev/null 2>&1; then
      RUN_DEPLOY_AS_SUDO=true
      log_warn "当前用户无 Docker 访问权限，将自动使用 sudo 执行部署。"
    else
      log_error "无法访问 Docker，请确认 Docker 已启动且当前用户具备权限。"
      exit 1
    fi
  fi

  log_info "✓ 依赖检查完成"
}

run_docker_cmd() {
  if [[ "$RUN_DEPLOY_AS_SUDO" == true ]]; then
    run_root docker "$@"
  else
    docker "$@"
  fi
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --install)
        ACTION="install"
        shift
        ;;
      --upgrade)
        ACTION="upgrade"
        shift
        ;;
      --uninstall)
        ACTION="uninstall"
        shift
        ;;
      --repo)
        REPO_URL="${2:-}"
        shift 2
        ;;
      --branch)
        BRANCH="${2:-}"
        shift 2
        ;;
      --install-dir)
        INSTALL_DIR="${2:-}"
        shift 2
        ;;
      --project-subdir)
        PROJECT_SUBDIR="${2:-}"
        shift 2
        ;;
      --skip-health-check)
        SKIP_HEALTH_CHECK=true
        shift
        ;;
      --interactive)
        INTERACTIVE_MODE="true"
        shift
        ;;
      --non-interactive)
        INTERACTIVE_MODE="false"
        shift
        ;;
      --use-image)
        USE_PREBUILT_IMAGE=true
        shift
        ;;
      --build-source|--use-source)
        USE_PREBUILT_IMAGE=false
        shift
        ;;
      --image-url)
        IMAGE_URL="${2:-}"
        USE_PREBUILT_IMAGE=true
        shift 2
        ;;
      --image-file)
        IMAGE_FILE="${2:-}"
        USE_PREBUILT_IMAGE=true
        shift 2
        ;;
      --image-name)
        IMAGE_NAME="${2:-}"
        shift 2
        ;;
      --image-version)
        IMAGE_VERSION="${2:-}"
        shift 2
        ;;
      --app-port)
        APP_PORT="${2:-}"
        shift 2
        ;;
      --app-bind)
        APP_BIND="${2:-}"
        shift 2
        ;;
      --enable-caddy)
        CADDY_ENABLED=true
        shift
        ;;
      --domain)
        CADDY_DOMAIN="${2:-}"
        CADDY_ENABLED=true
        shift 2
        ;;
      --cert-email)
        CADDY_EMAIL="${2:-}"
        CADDY_ENABLED=true
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
      --jwt-secret)
        JWT_SECRET="${2:-}"
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

load_runtime_defaults() {
  local project_dir="$1"
  local env_file="${project_dir}/.env"
  local config_file="${project_dir}/configs/config.yaml"

  local value=""
  value="$(read_env_value "$env_file" "APP_BIND")"
  if [[ -n "$value" ]]; then APP_BIND="$value"; fi
  value="$(read_env_value "$env_file" "APP_PORT")"
  if [[ -n "$value" ]]; then APP_PORT="$value"; fi
  value="$(read_env_value "$env_file" "IMAGE_NAME")"
  if [[ -n "$value" ]]; then IMAGE_NAME="$value"; fi
  value="$(read_env_value "$env_file" "BUILD_VERSION")"
  if [[ -n "$value" ]]; then IMAGE_VERSION="$value"; fi
  value="$(read_env_value "$env_file" "CADDY_DOMAIN")"
  if [[ -n "$value" ]]; then CADDY_DOMAIN="$value"; fi
  value="$(read_env_value "$env_file" "CADDY_EMAIL")"
  if [[ -n "$value" ]]; then CADDY_EMAIL="$value"; fi
  value="$(read_env_value "$env_file" "CADDY_HTTP_PORT")"
  if [[ -n "$value" ]]; then CADDY_HTTP_PORT="$value"; fi
  value="$(read_env_value "$env_file" "CADDY_HTTPS_PORT")"
  if [[ -n "$value" ]]; then CADDY_HTTPS_PORT="$value"; fi
  value="$(read_env_value "$env_file" "ENABLE_HTTPS_PROXY")"
  if [[ "${value,,}" == "true" ]]; then CADDY_ENABLED=true; fi

  if [[ -f "$config_file" ]]; then
    value="$(extract_yaml_value "$config_file" "admin" "username" || true)"
    if [[ -n "$value" ]]; then ADMIN_USERNAME="$value"; fi
    value="$(extract_yaml_value "$config_file" "admin" "email" || true)"
    if [[ -n "$value" ]]; then ADMIN_EMAIL="$value"; fi
    value="$(extract_yaml_value "$config_file" "admin" "password" || true)"
    if [[ -n "$value" ]]; then ADMIN_PASSWORD="$value"; fi
    value="$(extract_yaml_value "$config_file" "jwt" "secret" || true)"
    if [[ -n "$value" ]]; then JWT_SECRET="$value"; fi
  fi
}

auto_complete_runtime_values() {
  if [[ ! "$APP_PORT" =~ ^[0-9]+$ ]]; then
    APP_PORT="8090"
  fi
  if [[ ! "$CADDY_HTTP_PORT" =~ ^[0-9]+$ ]]; then
    CADDY_HTTP_PORT="80"
  fi
  if [[ ! "$CADDY_HTTPS_PORT" =~ ^[0-9]+$ ]]; then
    CADDY_HTTPS_PORT="443"
  fi

  CADDY_DOMAIN="$(sanitize_domain "$CADDY_DOMAIN")"
  if [[ "$CADDY_ENABLED" == true && -z "$CADDY_DOMAIN" ]]; then
    log_warn "已启用 Caddy 但未提供域名，自动回退为不启用 Caddy。"
    CADDY_ENABLED=false
  fi

  if [[ "$CADDY_ENABLED" == true ]]; then
    APP_BIND="127.0.0.1"
    if [[ -z "$CADDY_EMAIL" ]]; then
      CADDY_EMAIL="$ADMIN_EMAIL"
    fi
    if [[ "$ACTION" == "install" ]]; then
      if port_in_use "$CADDY_HTTP_PORT"; then
        local new_http_port
        new_http_port="$(find_available_port "$CADDY_HTTP_PORT")"
        log_warn "检测到 HTTP 端口 ${CADDY_HTTP_PORT} 占用，自动调整为 ${new_http_port}"
        CADDY_HTTP_PORT="$new_http_port"
      fi
      if port_in_use "$CADDY_HTTPS_PORT"; then
        local new_https_port
        new_https_port="$(find_available_port "$CADDY_HTTPS_PORT")"
        log_warn "检测到 HTTPS 端口 ${CADDY_HTTPS_PORT} 占用，自动调整为 ${new_https_port}"
        CADDY_HTTPS_PORT="$new_https_port"
      fi
    fi
  fi

  if [[ "$ACTION" == "install" ]] && port_in_use "$APP_PORT"; then
    local new_port
    new_port="$(find_available_port "$APP_PORT")"
    log_warn "检测到应用端口 ${APP_PORT} 占用，自动调整为 ${new_port}"
    APP_PORT="$new_port"
  fi

  if [[ -z "$ADMIN_USERNAME" ]]; then
    ADMIN_USERNAME="admin"
  fi
  if [[ -z "$ADMIN_EMAIL" ]]; then
    ADMIN_EMAIL="admin@license.local"
  fi
  if [[ -z "$ADMIN_PASSWORD" || "$ADMIN_PASSWORD" == "ChangeMe123!" || ${#ADMIN_PASSWORD} -lt 12 ]]; then
    ADMIN_PASSWORD="Np!$(generate_random_hex 10)Aa1"
    GENERATED_CONFIG_WARNING=true
    log_warn "管理员密码为空或弱密码，已自动生成强密码。"
  fi

  if [[ -z "$JWT_SECRET" || "$JWT_SECRET" == "change-this-license-secret" || ${#JWT_SECRET} -lt 32 ]]; then
    JWT_SECRET="$(generate_random_hex 32)"
    GENERATED_CONFIG_WARNING=true
    log_warn "JWT 密钥为空或长度不足，已自动生成 32 位随机密钥。"
  fi
}

run_interactive_wizard() {
  local host_ip
  host_ip="$(detect_host_ip)"

  echo ""
  echo "========== NodePass License Center 交互式部署 =========="
  log_info "检测到系统架构: $(uname -m)"
  if [[ -n "$host_ip" ]]; then
    log_info "检测到主机 IP: $host_ip"
  fi
  echo ""

  if prompt_yes_no "是否使用预构建多架构镜像（推荐）" "y"; then
    USE_PREBUILT_IMAGE=true
    IMAGE_NAME="$(prompt_with_default "镜像名称" "$IMAGE_NAME")"
    IMAGE_VERSION="$(prompt_with_default "镜像版本（建议 main 或 latest）" "$IMAGE_VERSION")"
  else
    USE_PREBUILT_IMAGE=false
  fi

  APP_PORT="$(prompt_with_default "服务端口" "$APP_PORT")"
  if [[ "$CADDY_ENABLED" != true ]]; then
    APP_BIND="$(prompt_with_default "服务绑定地址" "$APP_BIND")"
  fi

  ADMIN_USERNAME="$(prompt_with_default "管理员用户名" "${ADMIN_USERNAME:-admin}")"
  ADMIN_EMAIL="$(prompt_with_default "管理员邮箱" "${ADMIN_EMAIL:-admin@license.local}")"
  local input_password
  input_password="$(prompt_password_with_confirm "管理员密码（留空自动生成）" true)"
  if [[ -n "$input_password" ]]; then
    ADMIN_PASSWORD="$input_password"
  fi

  local jwt_input
  jwt_input="$(prompt_with_default "JWT 密钥（留空自动生成）" "")"
  if [[ -n "$jwt_input" ]]; then
    JWT_SECRET="$jwt_input"
  fi

  local caddy_default="n"
  if [[ "$CADDY_ENABLED" == true ]]; then
    caddy_default="y"
  fi
  if prompt_yes_no "是否启用域名绑定 + 自动 HTTPS 证书（Caddy）" "$caddy_default"; then
    CADDY_ENABLED=true
    while true; do
      CADDY_DOMAIN="$(prompt_with_default "绑定域名（必填）" "${CADDY_DOMAIN:-}")"
      CADDY_DOMAIN="$(sanitize_domain "$CADDY_DOMAIN")"
      if [[ -n "$CADDY_DOMAIN" ]]; then
        break
      fi
      echo "域名不能为空。"
    done
    CADDY_EMAIL="$(prompt_with_default "证书邮箱（建议填写）" "${CADDY_EMAIL:-$ADMIN_EMAIL}")"
    CADDY_HTTP_PORT="$(prompt_with_default "Caddy HTTP 端口" "$CADDY_HTTP_PORT")"
    CADDY_HTTPS_PORT="$(prompt_with_default "Caddy HTTPS 端口" "$CADDY_HTTPS_PORT")"
  else
    CADDY_ENABLED=false
  fi

  auto_complete_runtime_values
}

generate_https_proxy_files() {
  local project_dir="$1"
  local compose_file="${project_dir}/docker-compose.https.yml"
  local caddy_dir="${project_dir}/deploy/caddy"
  local caddy_file="${caddy_dir}/Caddyfile"

  if [[ "$CADDY_ENABLED" != true ]]; then
    run_root rm -f "$compose_file" || true
    return
  fi

  run_root mkdir -p "$caddy_dir"

  local tmp_caddy
  tmp_caddy="$(mktemp)"
  if [[ -n "$CADDY_EMAIL" ]]; then
    cat >"$tmp_caddy" <<EOF
{
    email ${CADDY_EMAIL}
}

${CADDY_DOMAIN} {
    encode gzip zstd
    reverse_proxy license-center:8090
}
EOF
  else
    cat >"$tmp_caddy" <<EOF
${CADDY_DOMAIN} {
    encode gzip zstd
    reverse_proxy license-center:8090
}
EOF
  fi
  run_root mv "$tmp_caddy" "$caddy_file"

  local tmp_compose
  tmp_compose="$(mktemp)"
  cat >"$tmp_compose" <<'EOF'
services:
  caddy:
    image: caddy:2.10-alpine
    container_name: license-center-caddy
    depends_on:
      license-center:
        condition: service_healthy
    ports:
      - "${CADDY_HTTP_PORT:-80}:80"
      - "${CADDY_HTTPS_PORT:-443}:443"
    volumes:
      - ./deploy/caddy/Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    networks:
      - license-network
    restart: unless-stopped

volumes:
  caddy_data:
  caddy_config:
EOF
  run_root mv "$tmp_compose" "$compose_file"
}

apply_runtime_settings() {
  local project_dir="$1"
  local env_file="${project_dir}/.env"
  local config_file="${project_dir}/configs/config.yaml"

  if [[ ! -f "$env_file" ]]; then
    run_root touch "$env_file"
  fi

  upsert_env_value "$env_file" "APP_BIND" "$APP_BIND"
  upsert_env_value "$env_file" "APP_PORT" "$APP_PORT"
  upsert_env_value "$env_file" "BUILD_VERSION" "$IMAGE_VERSION"
  upsert_env_value "$env_file" "IMAGE_NAME" "$IMAGE_NAME"
  upsert_env_value "$env_file" "ENABLE_HTTPS_PROXY" "$CADDY_ENABLED"
  upsert_env_value "$env_file" "CADDY_DOMAIN" "$CADDY_DOMAIN"
  upsert_env_value "$env_file" "CADDY_EMAIL" "$CADDY_EMAIL"
  upsert_env_value "$env_file" "CADDY_HTTP_PORT" "$CADDY_HTTP_PORT"
  upsert_env_value "$env_file" "CADDY_HTTPS_PORT" "$CADDY_HTTPS_PORT"
  ensure_file_readable "$env_file"

  update_yaml_value "$config_file" "jwt" "secret" "$JWT_SECRET"
  update_yaml_value "$config_file" "admin" "username" "$ADMIN_USERNAME"
  update_yaml_value "$config_file" "admin" "email" "$ADMIN_EMAIL"
  update_yaml_value "$config_file" "admin" "password" "$ADMIN_PASSWORD"
  ensure_config_mount_permissions "$config_file"

  generate_https_proxy_files "$project_dir"
}

prepare_install_dir() {
  local parent_dir
  parent_dir="$(dirname "$INSTALL_DIR")"
  run_root mkdir -p "$parent_dir"
}

download_image() {
  local url="$1"
  local output_file="$2"

  log_info "下载镜像: $url"

  if command -v wget >/dev/null 2>&1; then
    wget -O "$output_file" "$url"
  elif command -v curl >/dev/null 2>&1; then
    curl -L -o "$output_file" "$url"
  else
    log_error "需要 wget 或 curl 来下载镜像"
    exit 1
  fi
}

load_image_from_file() {
  local image_file="$1"

  log_step "加载 Docker 镜像..."
  log_info "镜像文件: $image_file"

  # 验证文件存在
  if [[ ! -f "$image_file" ]]; then
    log_error "镜像文件不存在: $image_file"
    exit 1
  fi

  # 验证校验和（如果存在）
  if [[ -f "${image_file}.sha256" ]]; then
    log_info "验证校验和..."
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum -c "${image_file}.sha256" || log_warn "校验和验证失败"
    elif command -v shasum >/dev/null 2>&1; then
      shasum -a 256 -c "${image_file}.sha256" || log_warn "校验和验证失败"
    fi
  fi

  # 加载镜像
  if [[ "$image_file" == *.gz ]]; then
    log_info "解压并加载镜像..."
    if [[ "$RUN_DEPLOY_AS_SUDO" == true ]]; then
      gunzip -c "$image_file" | run_root docker load
    else
      gunzip -c "$image_file" | docker load
    fi
  else
    run_docker_cmd load -i "$image_file"
  fi

  log_info "✓ 镜像加载完成"
}

prepare_image() {
  log_step "准备 Docker 镜像..."

  if [[ -n "$IMAGE_FILE" ]]; then
    # 使用本地镜像文件
    load_image_from_file "$IMAGE_FILE"
  elif [[ -n "$IMAGE_URL" ]]; then
    # 从 URL 下载镜像
    local temp_file="/tmp/license-center-${IMAGE_VERSION}.tar.gz"
    download_image "$IMAGE_URL" "$temp_file"
    load_image_from_file "$temp_file"
    rm -f "$temp_file"
  else
    # 从镜像仓库拉取镜像
    log_info "从镜像仓库拉取镜像: ${IMAGE_NAME}:${IMAGE_VERSION}"
    run_docker_cmd pull "${IMAGE_NAME}:${IMAGE_VERSION}"
  fi

  log_info "✓ 镜像准备完成"
}

has_local_templates() {
  [[ -f "${SCRIPT_DIR}/docker-compose.prod.yml" ]] && \
  [[ -f "${SCRIPT_DIR}/configs/config.yaml" ]] && \
  [[ -f "${SCRIPT_DIR}/scripts/deploy.sh" ]]
}

prepare_config_only() {
  log_step "准备配置文件..."

  prepare_install_dir

  # 只下载必要的配置文件
  local project_dir="${INSTALL_DIR}/${PROJECT_SUBDIR}"
  run_root mkdir -p "$project_dir"
  run_root mkdir -p "${project_dir}/configs"
  run_root mkdir -p "${project_dir}/scripts"
  ensure_dir_traversable "$project_dir"
  ensure_dir_traversable "${project_dir}/configs"
  ensure_dir_traversable "${project_dir}/scripts"

  local use_local_templates=false
  if has_local_templates; then
    use_local_templates=true
    log_info "检测到本地模板，优先使用本地文件。"
  fi

  # 准备 docker-compose 配置
  if [[ "$use_local_templates" == true ]]; then
    run_root cp "${SCRIPT_DIR}/docker-compose.prod.yml" "${project_dir}/docker-compose.yml"
  else
    log_info "下载 docker-compose 配置..."
    local compose_url="https://raw.githubusercontent.com/nodeox/NodePass-Pro/${BRANCH}/${PROJECT_SUBDIR}/docker-compose.prod.yml"
    if command -v curl >/dev/null 2>&1; then
      run_root curl -fsSL "$compose_url" -o "${project_dir}/docker-compose.yml"
    else
      run_root wget -q "$compose_url" -O "${project_dir}/docker-compose.yml"
    fi
  fi

  # 下载配置文件模板（升级时保留用户现有配置）
  if [[ -f "${project_dir}/configs/config.yaml" ]]; then
    log_info "检测到已有配置，跳过覆盖: ${project_dir}/configs/config.yaml"
  elif [[ "$use_local_templates" == true ]]; then
    run_root cp "${SCRIPT_DIR}/configs/config.yaml" "${project_dir}/configs/config.yaml"
  else
    log_info "下载配置文件模板..."
    local config_url="https://raw.githubusercontent.com/nodeox/NodePass-Pro/${BRANCH}/${PROJECT_SUBDIR}/configs/config.yaml"
    if command -v curl >/dev/null 2>&1; then
      run_root curl -fsSL "$config_url" -o "${project_dir}/configs/config.yaml"
    else
      run_root wget -q "$config_url" -O "${project_dir}/configs/config.yaml"
    fi
  fi
  ensure_config_mount_permissions "${project_dir}/configs/config.yaml"

  # 准备部署脚本
  if [[ "$use_local_templates" == true ]]; then
    run_root cp "${SCRIPT_DIR}/scripts/deploy.sh" "${project_dir}/scripts/deploy.sh"
  else
    log_info "下载部署脚本..."
    local deploy_url="https://raw.githubusercontent.com/nodeox/NodePass-Pro/${BRANCH}/${PROJECT_SUBDIR}/scripts/deploy.sh"
    if command -v curl >/dev/null 2>&1; then
      run_root curl -fsSL "$deploy_url" -o "${project_dir}/scripts/deploy.sh"
    else
      run_root wget -q "$deploy_url" -O "${project_dir}/scripts/deploy.sh"
    fi
  fi
  run_root chmod +x "${project_dir}/scripts/deploy.sh"

  # 创建 .env 文件
  if [[ ! -f "${project_dir}/.env" ]]; then
    log_info "创建环境变量文件..."
    run_root tee "${project_dir}/.env" > /dev/null <<EOF
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=nodepass_license
POSTGRES_PORT=5432
APP_BIND=${APP_BIND}
APP_PORT=${APP_PORT}
BUILD_VERSION=${IMAGE_VERSION}
GIN_MODE=release
IMAGE_NAME=${IMAGE_NAME}
ENABLE_HTTPS_PROXY=false
CADDY_DOMAIN=
CADDY_EMAIL=
CADDY_HTTP_PORT=80
CADDY_HTTPS_PORT=443
EOF
  fi

  log_info "✓ 配置准备完成"
}

prepare_repo() {
  log_step "准备代码仓库..."

  prepare_install_dir

  if [[ -d "${INSTALL_DIR}/.git" ]]; then
    log_info "更新仓库: ${INSTALL_DIR}"
    run_root git -C "${INSTALL_DIR}" fetch origin "${BRANCH}"
    run_root git -C "${INSTALL_DIR}" checkout "${BRANCH}"
    run_root git -C "${INSTALL_DIR}" reset --hard "origin/${BRANCH}"
    run_root git -C "${INSTALL_DIR}" clean -fd
  else
    log_info "克隆仓库到: ${INSTALL_DIR}"
    run_root rm -rf "${INSTALL_DIR}"
    run_root git clone --branch "${BRANCH}" --depth 1 "${REPO_URL}" "${INSTALL_DIR}"
  fi

  log_info "✓ 代码准备完成"
}

resolve_project_dir() {
  if [[ -f "${INSTALL_DIR}/${PROJECT_SUBDIR}/scripts/deploy.sh" ]]; then
    echo "${INSTALL_DIR}/${PROJECT_SUBDIR}"
    return
  fi

  if [[ -f "${INSTALL_DIR}/scripts/deploy.sh" && -f "${INSTALL_DIR}/docker-compose.yml" ]]; then
    echo "${INSTALL_DIR}"
    return
  fi

  echo ""
}

get_app_port() {
  local project_dir="$1"
  local env_file="${project_dir}/.env"
  local app_port="8090"

  if [[ -f "$env_file" ]]; then
    local value
    value="$(grep -E '^APP_PORT=' "$env_file" | tail -n 1 | cut -d'=' -f2- | tr -d '[:space:]' || true)"
    if [[ "$value" =~ ^[0-9]+$ ]]; then
      app_port="$value"
    fi
  fi

  echo "$app_port"
}

backup_config() {
  local project_dir="$1"
  local config_file="${project_dir}/configs/config.yaml"

  if [[ -z "$project_dir" ]]; then
    log_warn "未检测到项目目录，跳过配置备份。"
    return 0
  fi

  if [[ -f "$config_file" ]]; then
    run_root mkdir -p "$BACKUP_DIR"
    local backup_file="${BACKUP_DIR}/config.yaml.backup.$(date +%Y%m%d_%H%M%S)"
    log_info "备份配置文件: $backup_file"
    run_root cp "$config_file" "$backup_file"
    LAST_CONFIG_BACKUP="$backup_file"
  else
    log_warn "未找到配置文件，跳过备份: $config_file"
  fi
}

restore_config() {
  local project_dir="$1"

  if [[ -z "$project_dir" || -z "$LAST_CONFIG_BACKUP" ]]; then
    return 0
  fi
  if [[ ! -f "$LAST_CONFIG_BACKUP" ]]; then
    log_warn "备份文件不存在，跳过配置恢复: $LAST_CONFIG_BACKUP"
    return 0
  fi

  local target="${project_dir}/configs/config.yaml"
  run_root mkdir -p "$(dirname "$target")"
  run_root cp "$LAST_CONFIG_BACKUP" "$target"
  ensure_config_mount_permissions "$target"
  log_info "已恢复用户配置: $target"
}

run_deploy() {
  local project_dir="$1"

  if [[ -z "$project_dir" ]]; then
    log_error "未找到部署脚本，请检查 --repo 与 --project-subdir 是否正确。"
    exit 1
  fi

  log_step "开始部署服务..."

  local compose_args=(-f docker-compose.yml)
  if [[ "$CADDY_ENABLED" == true && -f "${project_dir}/docker-compose.https.yml" ]]; then
    compose_args+=(-f docker-compose.https.yml)
  fi

  if [[ "$RUN_DEPLOY_AS_SUDO" == true ]]; then
    if [[ "$USE_PREBUILT_IMAGE" == true ]]; then
      (cd "$project_dir" && run_root docker compose "${compose_args[@]}" pull) || true
      (cd "$project_dir" && run_root docker compose "${compose_args[@]}" up -d)
    else
      (cd "$project_dir" && run_root docker compose "${compose_args[@]}" up -d --build)
    fi
  else
    if [[ "$USE_PREBUILT_IMAGE" == true ]]; then
      (cd "$project_dir" && docker compose "${compose_args[@]}" pull) || true
      (cd "$project_dir" && docker compose "${compose_args[@]}" up -d)
    else
      (cd "$project_dir" && docker compose "${compose_args[@]}" up -d --build)
    fi
  fi

  log_info "✓ 服务部署完成"
}

run_down() {
  local project_dir="$1"

  if [[ -z "$project_dir" ]]; then
    log_warn "未检测到可卸载的项目目录，跳过停止服务。"
    return
  fi

  log_step "停止服务..."

  local compose_args=(-f docker-compose.yml)
  if [[ -f "${project_dir}/docker-compose.https.yml" ]]; then
    compose_args+=(-f docker-compose.https.yml)
  fi

  if [[ "$RUN_DEPLOY_AS_SUDO" == true ]]; then
    (cd "$project_dir" && run_root docker compose "${compose_args[@]}" down) || true
  else
    (cd "$project_dir" && docker compose "${compose_args[@]}" down) || true
  fi
}

check_service_health() {
  local project_dir="$1"

  if [[ "$SKIP_HEALTH_CHECK" == true ]]; then
    log_info "跳过健康检查"
    return 0
  fi

  log_step "检查服务健康状态..."
  local app_port
  app_port="$(get_app_port "$project_dir")"

  local max_attempts=30
  local attempt=0

  while [[ $attempt -lt $max_attempts ]]; do
    if curl -sf "http://127.0.0.1:${app_port}/health" >/dev/null 2>&1; then
      log_info "✓ 服务健康检查通过"
      return 0
    fi

    attempt=$((attempt + 1))
    sleep 2
  done

  log_warn "服务健康检查超时，请手动检查日志"
  if [[ "$CADDY_ENABLED" == true ]]; then
    log_info "查看日志命令: cd ${INSTALL_DIR}/${PROJECT_SUBDIR} && ./scripts/deploy.sh --with-https-proxy --logs"
  else
    log_info "查看日志命令: cd ${INSTALL_DIR}/${PROJECT_SUBDIR} && ./scripts/deploy.sh --logs"
  fi
  return 1
}

show_success_info() {
  local app_port="$1"
  local health_url="http://127.0.0.1:${app_port}/health"
  local console_url="http://127.0.0.1:${app_port}/console"
  local api_url="http://127.0.0.1:${app_port}/api/v1"
  local compose_cmd="docker compose -f ${INSTALL_DIR}/${PROJECT_SUBDIR}/docker-compose.yml"

  if [[ "$CADDY_ENABLED" == true && -n "$CADDY_DOMAIN" ]]; then
    console_url="https://${CADDY_DOMAIN}/console"
    api_url="https://${CADDY_DOMAIN}/api/v1"
    compose_cmd="${compose_cmd} -f ${INSTALL_DIR}/${PROJECT_SUBDIR}/docker-compose.https.yml"
  fi

  cat <<EOF

${GREEN}╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║  🎉 NodePass License Center 部署成功！                         ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝${NC}

${BLUE}📍 访问地址:${NC}
  • 健康检查: ${health_url}
  • 管理面板: ${console_url}
  • API 文档: ${api_url}

${BLUE}🔐 管理员账号:${NC}
  • 用户名: ${ADMIN_USERNAME}
  • 邮箱: ${ADMIN_EMAIL}
  • 密码: ${ADMIN_PASSWORD}

${BLUE}📚 功能特性:${NC}
  • ✅ 授权码管理（生成、吊销、转移）
  • ✅ 域名绑定（防止多站点共享）
  • ✅ 套餐管理（版本限制、机器数量）
  • ✅ 监控告警（实时统计、趋势分析）
  • ✅ Webhook 通知（事件推送）
  • ✅ 标签管理（授权码分类）
  • ✅ 安全增强（限流、签名、IP 白名单）

${BLUE}🔧 常用命令:${NC}
  • 查看日志: ${compose_cmd} logs -f
  • 重启服务: ${compose_cmd} restart
  • 停止服务: ${compose_cmd} down
  • 升级版本: bash install.sh --upgrade

${BLUE}📖 文档:${NC}
  • 完整文档: ${INSTALL_DIR}/${PROJECT_SUBDIR}/README.md
  • 架构说明: ${INSTALL_DIR}/${PROJECT_SUBDIR}/ARCHITECTURE.md
  • 域名绑定: ${INSTALL_DIR}/${PROJECT_SUBDIR}/docs/DOMAIN_BINDING.md

${BLUE}💡 提示:${NC}
  • 配置文件: ${INSTALL_DIR}/${PROJECT_SUBDIR}/configs/config.yaml
  • 修改配置后需重启服务
  • 建议启用 HTTPS 和防火墙

EOF

  if [[ "$GENERATED_CONFIG_WARNING" == true ]]; then
    cat <<EOF
${YELLOW}⚠️ 安全提示:${NC}
  • 安装过程已自动生成 jwt.secret / admin.password（因检测到空值或弱值）
  • 请立即检查并妥善保存配置文件中的密钥与管理员密码
  • 配置文件路径: ${INSTALL_DIR}/${PROJECT_SUBDIR}/configs/config.yaml

EOF
  fi
}

show_upgrade_info() {
  local app_port="$1"
  local console_url="http://127.0.0.1:${app_port}/console"

  if [[ "$CADDY_ENABLED" == true && -n "$CADDY_DOMAIN" ]]; then
    console_url="https://${CADDY_DOMAIN}/console"
  fi

  cat <<EOF

${GREEN}╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║  ✨ NodePass License Center 升级成功！                         ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝${NC}

${BLUE}📍 访问地址:${NC}
  • 管理面板: ${console_url}

${BLUE}🆕 v0.4.0 关键能力:${NC}
  • ✨ 交互式一键部署（镜像/源码可选）
  • ✨ 自动环境检测与补全（端口/JWT/管理员密码）
  • ✨ 可选域名绑定 + Caddy 自动 HTTPS 证书
  • ✨ 默认使用 GHCR 多架构镜像（main）
  • ✨ 部署脚本支持多 compose 文件叠加

${BLUE}⚠️  重要提示:${NC}
  • 升级前配置已自动备份（如存在）
  • 数据库已自动迁移
  • 请确认 .env 与 configs/config.yaml 中的新配置项

EOF

  if [[ "$GENERATED_CONFIG_WARNING" == true ]]; then
    cat <<EOF
${YELLOW}⚠️ 升级提示:${NC}
  • 检测到弱配置已自动修复（jwt.secret / admin.password）
  • 请检查并确认配置文件: ${INSTALL_DIR}/${PROJECT_SUBDIR}/configs/config.yaml

EOF
  fi
}

show_uninstall_info() {
  cat <<EOF

${GREEN}╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║  ✅ NodePass License Center 已卸载                             ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝${NC}

${YELLOW}已清理:${NC}
  • 服务容器
  • 代码文件
  • 安装目录: ${INSTALL_DIR}

${YELLOW}保留:${NC}
  • Docker 镜像（可手动删除）
  • 数据库数据（如使用外部数据库）

${BLUE}完全清理命令:${NC}
  docker system prune -a

EOF
}

do_uninstall() {
  if [[ ! -d "${INSTALL_DIR}" ]]; then
    log_info "目录不存在，跳过卸载"
    exit 0
  fi

  local project_dir
  project_dir="$(resolve_project_dir)"
  run_down "$project_dir"

  run_root rm -rf "${INSTALL_DIR}"

  show_uninstall_info
}

main() {
  echo -e "${BLUE}"
  cat <<'BANNER'
╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║   _   _           _      ____                                  ║
║  | \ | | ___   __| | ___|  _ \ __ _ ___ ___                   ║
║  |  \| |/ _ \ / _` |/ _ \ |_) / _` / __/ __|                  ║
║  | |\  | (_) | (_| |  __/  __/ (_| \__ \__ \                  ║
║  |_| \_|\___/ \__,_|\___|_|   \__,_|___/___/                  ║
║                                                                ║
║              License Center v0.4.0                             ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝
BANNER
  echo -e "${NC}"

  parse_args "$@"

  if [[ "$ACTION" == "uninstall" ]]; then
    do_uninstall
    exit 0
  fi

  check_system_requirements
  ensure_env

  if [[ "$ACTION" == "upgrade" ]]; then
    local existing_project_dir
    existing_project_dir="$(resolve_project_dir)"
    backup_config "$existing_project_dir"
  fi

  if [[ "$USE_PREBUILT_IMAGE" == true ]]; then
    log_info "使用预构建镜像模式（默认）"
    prepare_image
    prepare_config_only
  else
    log_info "使用源码构建模式（已显式指定）"
    prepare_repo
  fi

  local project_dir
  project_dir="$(resolve_project_dir)"

  if [[ "$ACTION" == "upgrade" ]]; then
    log_info "执行升级部署..."
    restore_config "$project_dir"
  else
    log_info "执行全新安装..."
  fi

  load_runtime_defaults "$project_dir"
  if should_run_interactive; then
    run_interactive_wizard
  else
    auto_complete_runtime_values
  fi

  apply_runtime_settings "$project_dir"
  ensure_secure_config "${project_dir}/configs/config.yaml"

  run_deploy "$project_dir"

  # 等待服务启动
  sleep 5
  check_service_health "$project_dir"

  local app_port
  app_port="$(get_app_port "$project_dir")"

  if [[ "$ACTION" == "upgrade" ]]; then
    show_upgrade_info "$app_port"
  else
    show_success_info "$app_port"
  fi
}

main "$@"
