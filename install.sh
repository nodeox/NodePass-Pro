#!/usr/bin/env bash
set -euo pipefail

REPO_URL="https://github.com/nodeox/NodePass-Pro.git"
BRANCH="main"
INSTALL_DIR="/opt/NodePass-Pro"
INTERACTIVE_MODE="auto" # auto / true / false

PKG_MANAGER=""
SUDO_CMD=""
USE_SUDO_FS=false
USE_SUDO_DOCKER=false
RUN_DEPLOY_AS_SUDO=false

PASSTHROUGH_ARGS=()

WITH_CADDY=true
FRONTEND_DOMAIN=""
BACKEND_DOMAIN=""
CADDY_EMAIL=""
CADDY_HTTP_PORT="80"
CADDY_HTTPS_PORT="443"
FRONTEND_BIND="127.0.0.1:5173"

DB_MODE="internal_postgres" # internal_postgres / external_postgres / external_mysql / sqlite
DB_HOST="postgres"
DB_PORT="5432"
DB_USER="postgres"
DB_PASSWORD="postgres"
DB_NAME="nodepass_panel"
DB_SSLMODE="disable"
DB_TIMEZONE="Asia/Shanghai"
SQLITE_DSN="./data/nodepass.db"

REDIS_MODE="internal" # internal / external
REDIS_ADDR="redis:6379"
REDIS_PASSWORD=""
REDIS_DB="0"
REDIS_KEY_PREFIX="nodepass:panel"
REDIS_DEFAULT_TTL="300"

BACKEND_PORT="8080"
BACKEND_MODE="release"
JWT_SECRET=""
JWT_EXPIRE_TIME="168"

BACKEND_CONFIG_FILE_REL="./backend/configs/config.runtime.yaml"
BACKEND_CONFIG_FILE_ABS=""

CREATE_ADMIN="auto"
ADMIN_USERNAME="admin"
ADMIN_EMAIL="admin@example.com"
ADMIN_PASSWORD=""
ADMIN_CREATED="false"

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
NodePass Pro 远程一键部署引导脚本（自动检测环境 + 交互式部署）

用法:
  bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh)
  bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh) [引导参数] [deploy 参数]

引导参数:
  --install-dir <目录>      安装目录（默认: /opt/NodePass-Pro）
  --repo <地址>             仓库地址（默认: https://github.com/nodeox/NodePass-Pro.git）
  --branch <分支>           分支名（默认: main）
  --interactive             强制交互式部署
  --non-interactive         关闭交互，直接透传参数给 scripts/deploy.sh
  --admin-username <用户名> 指定管理员用户名（可配合 --non-interactive）
  --admin-email <邮箱>      指定管理员邮箱（可配合 --non-interactive）
  --admin-password <密码>   指定管理员密码（可配合 --non-interactive）
  --skip-admin              跳过管理员账号创建
  -h, --help                显示帮助

示例:
  # 交互式（推荐）
  bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh)

  # 非交互，直接透传 deploy 参数
  bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh) --non-interactive --with-caddy --frontend-domain panel.example.com --email admin@example.com

  # 非交互，部署后自动创建管理员
  bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh) --non-interactive --admin-username admin --admin-email admin@example.com --admin-password 'YourStrongPassword'

  # 停止服务
  bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh) --non-interactive --down
EOF
}

run_root() {
  if [[ -n "$SUDO_CMD" ]]; then
    "$SUDO_CMD" "$@"
  else
    "$@"
  fi
}

run_fs() {
  if [[ "$USE_SUDO_FS" == true ]]; then
    run_root "$@"
  else
    "$@"
  fi
}

require_command() {
  local command_name="$1"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    log_error "缺少命令: $command_name"
    exit 1
  fi
}

sanitize_domain() {
  local input="$1"
  input="${input#http://}"
  input="${input#https://}"
  input="${input%%/*}"
  echo "$input"
}

yaml_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

contains_passthrough_arg() {
  local target="$1"
  local value
  for value in "${PASSTHROUGH_ARGS[@]}"; do
    if [[ "$value" == "$target" ]]; then
      return 0
    fi
  done
  return 1
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

ensure_tool() {
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
  log_info "未检测到 Docker，开始自动安装..."
  if [[ "$(uname -s)" != "Linux" ]]; then
    log_error "自动安装 Docker 仅支持 Linux，请手动安装 Docker。"
    exit 1
  fi

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

ensure_docker_compose() {
  if docker compose version >/dev/null 2>&1; then
    return
  fi

  log_info "未检测到 Docker Compose 插件，尝试自动安装..."
  case "$PKG_MANAGER" in
    apt|dnf|yum)
      install_packages docker-compose-plugin || true
      ;;
    pacman)
      install_packages docker-compose || true
      ;;
    zypper)
      install_packages docker-compose || true
      ;;
  esac

  if ! docker compose version >/dev/null 2>&1; then
    log_error "Docker Compose 插件安装失败，请手动安装后重试。"
    exit 1
  fi
}

ensure_docker_access() {
  if docker info >/dev/null 2>&1; then
    USE_SUDO_DOCKER=false
    return
  fi

  if [[ -n "$SUDO_CMD" ]] && sudo docker info >/dev/null 2>&1; then
    USE_SUDO_DOCKER=true
    log_warn "当前用户无法直接访问 Docker，将使用 sudo 执行部署。"
    return
  fi

  log_error "无法访问 Docker，请确认 Docker 已启动且当前用户有权限。"
  exit 1
}

ensure_environment() {
  detect_sudo
  detect_pkg_manager
  ensure_tool git git
  ensure_tool curl curl ca-certificates
  install_docker_engine
  ensure_docker_service
  ensure_docker_compose
  ensure_docker_access
}

determine_install_dir_privilege() {
  local parent_dir
  parent_dir="$(dirname "$INSTALL_DIR")"
  if [[ -d "$INSTALL_DIR" ]]; then
    if [[ -w "$INSTALL_DIR" ]]; then
      USE_SUDO_FS=false
    elif [[ -n "$SUDO_CMD" ]]; then
      USE_SUDO_FS=true
    else
      log_error "无权限写入安装目录: $INSTALL_DIR，请使用 --install-dir 指定可写目录。"
      exit 1
    fi
  else
    if [[ -d "$parent_dir" && -w "$parent_dir" ]]; then
      USE_SUDO_FS=false
    elif [[ -n "$SUDO_CMD" ]]; then
      USE_SUDO_FS=true
    else
      log_error "无权限创建安装目录: $INSTALL_DIR，请使用 --install-dir 指定可写目录。"
      exit 1
    fi
  fi
}

prepare_repo() {
  determine_install_dir_privilege

  if run_fs test -d "${INSTALL_DIR}/.git"; then
    log_info "检测到已有仓库，开始更新: ${INSTALL_DIR}"
    run_fs git -C "${INSTALL_DIR}" fetch origin "${BRANCH}"
    run_fs git -C "${INSTALL_DIR}" checkout "${BRANCH}"
    run_fs git -C "${INSTALL_DIR}" reset --hard "origin/${BRANCH}"
  else
    log_info "开始克隆仓库到: ${INSTALL_DIR}"
    run_fs mkdir -p "$(dirname "${INSTALL_DIR}")"
    run_fs rm -rf "${INSTALL_DIR}"
    run_fs git clone --branch "${BRANCH}" --depth 1 "${REPO_URL}" "${INSTALL_DIR}"
  fi
}

prompt_with_default() {
  local prompt="$1"
  local default_value="$2"
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
  local default_value="$2" # y / n
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

generate_random_secret() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
    return
  fi
  head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n'
}

prompt_password_with_confirm() {
  local prompt="$1"
  local password=""
  local confirm=""
  while true; do
    read -r -s -p "${prompt}: " password
    echo ""
    read -r -s -p "确认${prompt}: " confirm
    echo ""

    if [[ -z "$password" ]]; then
      echo "密码不能为空。"
      continue
    fi
    if [[ "$password" != "$confirm" ]]; then
      echo "两次输入的密码不一致，请重新输入。"
      continue
    fi
    if [[ ${#password} -lt 6 ]]; then
      echo "密码长度不能少于 6 位。"
      continue
    fi
    echo "$password"
    return
  done
}

validate_admin_config() {
  if [[ "$CREATE_ADMIN" != "true" ]]; then
    return
  fi
  if [[ -z "$ADMIN_USERNAME" || -z "$ADMIN_EMAIL" || -z "$ADMIN_PASSWORD" ]]; then
    log_error "管理员信息不完整，请提供管理员用户名、邮箱和密码。"
    exit 1
  fi
}

run_interactive_wizard() {
  echo ""
  echo "========== NodePass Pro 交互式部署 =========="
  echo ""

  INSTALL_DIR="$(prompt_with_default "安装目录" "$INSTALL_DIR")"
  FRONTEND_BIND="$(prompt_with_default "前端本机监听地址（用于本地反向代理）" "$FRONTEND_BIND")"

  if prompt_yes_no "是否启用 Caddy 自动 HTTPS 反代" "y"; then
    WITH_CADDY=true
    while true; do
      FRONTEND_DOMAIN="$(prompt_with_default "前端域名（必填）" "${FRONTEND_DOMAIN:-panel.example.com}")"
      FRONTEND_DOMAIN="$(sanitize_domain "$FRONTEND_DOMAIN")"
      if [[ -n "$FRONTEND_DOMAIN" ]]; then
        break
      fi
      echo "前端域名不能为空。"
    done
    BACKEND_DOMAIN="$(prompt_with_default "后端域名（可选，留空则使用前端域名/api）" "$BACKEND_DOMAIN")"
    BACKEND_DOMAIN="$(sanitize_domain "$BACKEND_DOMAIN")"
    CADDY_EMAIL="$(prompt_with_default "Caddy 证书邮箱（可选）" "$CADDY_EMAIL")"
    CADDY_HTTP_PORT="$(prompt_with_default "Caddy HTTP 端口" "$CADDY_HTTP_PORT")"
    CADDY_HTTPS_PORT="$(prompt_with_default "Caddy HTTPS 端口" "$CADDY_HTTPS_PORT")"
  else
    WITH_CADDY=false
  fi

  echo ""
  echo "数据库类型:"
  echo "  1) 内置 PostgreSQL（默认，Docker 内置）"
  echo "  2) 外部 PostgreSQL"
  echo "  3) 外部 MySQL"
  echo "  4) SQLite（单机）"
  local db_choice
  db_choice="$(prompt_with_default "请选择数据库类型" "1")"

  case "$db_choice" in
    1)
      DB_MODE="internal_postgres"
      DB_HOST="postgres"
      DB_PORT="5432"
      DB_USER="postgres"
      DB_PASSWORD="postgres"
      DB_NAME="nodepass_panel"
      ;;
    2)
      DB_MODE="external_postgres"
      DB_HOST="$(prompt_with_default "PostgreSQL 主机" "${DB_HOST:-127.0.0.1}")"
      DB_PORT="$(prompt_with_default "PostgreSQL 端口" "${DB_PORT:-5432}")"
      DB_USER="$(prompt_with_default "PostgreSQL 用户" "${DB_USER:-postgres}")"
      read -r -p "PostgreSQL 密码: " DB_PASSWORD
      DB_NAME="$(prompt_with_default "PostgreSQL 数据库名" "${DB_NAME:-nodepass_panel}")"
      DB_SSLMODE="$(prompt_with_default "PostgreSQL sslmode" "$DB_SSLMODE")"
      DB_TIMEZONE="$(prompt_with_default "PostgreSQL TimeZone" "$DB_TIMEZONE")"
      ;;
    3)
      DB_MODE="external_mysql"
      DB_HOST="$(prompt_with_default "MySQL 主机" "${DB_HOST:-127.0.0.1}")"
      DB_PORT="$(prompt_with_default "MySQL 端口" "3306")"
      DB_USER="$(prompt_with_default "MySQL 用户" "${DB_USER:-root}")"
      read -r -p "MySQL 密码: " DB_PASSWORD
      DB_NAME="$(prompt_with_default "MySQL 数据库名" "${DB_NAME:-nodepass_panel}")"
      ;;
    4)
      DB_MODE="sqlite"
      SQLITE_DSN="$(prompt_with_default "SQLite 路径（容器内）" "$SQLITE_DSN")"
      ;;
    *)
      log_warn "未知选项，使用内置 PostgreSQL。"
      DB_MODE="internal_postgres"
      ;;
  esac

  if prompt_yes_no "Redis 使用外部服务吗" "n"; then
    REDIS_MODE="external"
    REDIS_ADDR="$(prompt_with_default "Redis 地址（host:port）" "${REDIS_ADDR:-127.0.0.1:6379}")"
    read -r -p "Redis 密码（可选）: " REDIS_PASSWORD
    REDIS_DB="$(prompt_with_default "Redis DB" "$REDIS_DB")"
  else
    REDIS_MODE="internal"
    REDIS_ADDR="redis:6379"
    REDIS_PASSWORD=""
    REDIS_DB="0"
  fi

  BACKEND_PORT="$(prompt_with_default "后端服务端口" "$BACKEND_PORT")"
  JWT_SECRET="$(prompt_with_default "JWT Secret（留空自动生成）" "$JWT_SECRET")"
  if [[ -z "$JWT_SECRET" ]]; then
    JWT_SECRET="$(generate_random_secret)"
  fi
  JWT_EXPIRE_TIME="$(prompt_with_default "JWT 过期时间（小时）" "$JWT_EXPIRE_TIME")"

  if prompt_yes_no "是否创建/更新管理员账号" "y"; then
    CREATE_ADMIN="true"
    ADMIN_USERNAME="$(prompt_with_default "管理员用户名" "$ADMIN_USERNAME")"
    ADMIN_EMAIL="$(prompt_with_default "管理员邮箱" "$ADMIN_EMAIL")"
    ADMIN_PASSWORD="$(prompt_password_with_confirm "管理员密码")"
  else
    CREATE_ADMIN="false"
  fi

  echo ""
  log_info "交互配置完成。"
}

render_backend_config() {
  local db_type=""
  local db_dsn=""

  case "$DB_MODE" in
    internal_postgres|external_postgres)
      db_type="postgres"
      db_dsn="host=${DB_HOST} port=${DB_PORT} user=${DB_USER} password=${DB_PASSWORD} dbname=${DB_NAME} sslmode=${DB_SSLMODE} TimeZone=${DB_TIMEZONE}"
      ;;
    external_mysql)
      db_type="mysql"
      db_dsn="${DB_USER}:${DB_PASSWORD}@tcp(${DB_HOST}:${DB_PORT})/${DB_NAME}?charset=utf8mb4&parseTime=True&loc=Local"
      ;;
    sqlite)
      db_type="sqlite"
      db_dsn="${SQLITE_DSN}"
      DB_HOST="localhost"
      DB_PORT="0"
      DB_USER=""
      DB_PASSWORD=""
      DB_NAME="nodepass_panel"
      ;;
    *)
      log_error "未知数据库模式: $DB_MODE"
      exit 1
      ;;
  esac

  local allowed_origins=(
    "localhost"
    "127.0.0.1"
  )
  if [[ -n "$FRONTEND_DOMAIN" ]]; then
    allowed_origins+=("https://${FRONTEND_DOMAIN}" "http://${FRONTEND_DOMAIN}")
  fi
  if [[ -n "$BACKEND_DOMAIN" ]]; then
    allowed_origins+=("https://${BACKEND_DOMAIN}" "http://${BACKEND_DOMAIN}")
  fi

  local allowed_origins_yaml=""
  local origin
  for origin in "${allowed_origins[@]}"; do
    allowed_origins_yaml+="    - \"$(yaml_escape "$origin")\"\n"
  done

  local tmp_file
  tmp_file="$(mktemp)"
  cat >"$tmp_file" <<EOF
server:
  port: "$(yaml_escape "$BACKEND_PORT")"
  mode: "$(yaml_escape "$BACKEND_MODE")"
  allowed_origins:
$(printf "%b" "$allowed_origins_yaml")

database:
  type: "$(yaml_escape "$db_type")"
  host: "$(yaml_escape "$DB_HOST")"
  port: ${DB_PORT}
  user: "$(yaml_escape "$DB_USER")"
  password: "$(yaml_escape "$DB_PASSWORD")"
  db_name: "$(yaml_escape "$DB_NAME")"
  dsn: "$(yaml_escape "$db_dsn")"

redis:
  enabled: true
  addr: "$(yaml_escape "$REDIS_ADDR")"
  password: "$(yaml_escape "$REDIS_PASSWORD")"
  db: ${REDIS_DB}
  key_prefix: "$(yaml_escape "$REDIS_KEY_PREFIX")"
  default_ttl: ${REDIS_DEFAULT_TTL}

jwt:
  secret: "$(yaml_escape "$JWT_SECRET")"
  expire_time: ${JWT_EXPIRE_TIME}

telegram:
  bot_token: ""
  bot_username: ""
  webhook_url: ""
EOF

  BACKEND_CONFIG_FILE_ABS="${INSTALL_DIR}/backend/configs/config.runtime.yaml"
  run_fs mkdir -p "${INSTALL_DIR}/backend/configs"
  run_fs cp "$tmp_file" "$BACKEND_CONFIG_FILE_ABS"
  rm -f "$tmp_file"
  log_info "已生成后端运行配置: ${BACKEND_CONFIG_FILE_ABS}"
}

build_interactive_deploy_args() {
  PASSTHROUGH_ARGS=()
  if [[ "$WITH_CADDY" == true ]]; then
    PASSTHROUGH_ARGS+=(
      "--with-caddy"
      "--frontend-domain" "$FRONTEND_DOMAIN"
      "--caddy-http-port" "$CADDY_HTTP_PORT"
      "--caddy-https-port" "$CADDY_HTTPS_PORT"
    )
    if [[ -n "$BACKEND_DOMAIN" ]]; then
      PASSTHROUGH_ARGS+=("--backend-domain" "$BACKEND_DOMAIN")
    fi
    if [[ -n "$CADDY_EMAIL" ]]; then
      PASSTHROUGH_ARGS+=("--email" "$CADDY_EMAIL")
    fi
  fi
}

run_compose_cmd() {
  local compose_args=(-f docker-compose.yml)
  if [[ "$WITH_CADDY" == true ]]; then
    compose_args+=(-f docker-compose.caddy.yml)
  fi

  if [[ "$RUN_DEPLOY_AS_SUDO" == true ]]; then
    (cd "$INSTALL_DIR" && run_root docker compose "${compose_args[@]}" "$@")
  else
    (cd "$INSTALL_DIR" && docker compose "${compose_args[@]}" "$@")
  fi
}

bootstrap_admin_account() {
  if [[ "$CREATE_ADMIN" != "true" ]]; then
    return
  fi

  validate_admin_config

  log_info "开始创建/更新管理员账号..."
  local attempt
  for attempt in 1 2 3; do
    if run_compose_cmd exec -T backend /usr/local/bin/nodepass-admin-bootstrap \
      --username "$ADMIN_USERNAME" \
      --email "$ADMIN_EMAIL" \
      --password "$ADMIN_PASSWORD"; then
      ADMIN_CREATED="true"
      log_info "管理员账号初始化成功。"
      return
    fi
    log_warn "管理员初始化失败，正在重试 (${attempt}/3)..."
    sleep 2
  done

  log_error "管理员账号初始化失败，请检查后端日志。"
  exit 1
}

print_success_summary() {
  local frontend_url="http://${FRONTEND_BIND}"
  local backend_url="http://127.0.0.1:8080/api/v1"
  local node_install=""

  if [[ "$WITH_CADDY" == true && -n "$FRONTEND_DOMAIN" ]]; then
    frontend_url="https://${FRONTEND_DOMAIN}"
    backend_url="${frontend_url}/api/v1"
    node_install="${frontend_url}/nodeclient-install.sh"
    if [[ -n "$BACKEND_DOMAIN" ]]; then
      backend_url="https://${BACKEND_DOMAIN}/api/v1"
    fi
  fi

  echo ""
  echo "================ 部署完成 ================"
  echo "安装目录: ${INSTALL_DIR}"
  echo "前端地址: ${frontend_url}"
  echo "后端 API: ${backend_url}"
  if [[ -n "$node_install" ]]; then
    echo "节点安装脚本: ${node_install}"
  fi
  if [[ "$CREATE_ADMIN" == "true" && "$ADMIN_CREATED" == "true" ]]; then
    echo "管理员账号: ${ADMIN_USERNAME}"
    echo "管理员邮箱: ${ADMIN_EMAIL}"
    echo "管理员密码: ${ADMIN_PASSWORD}"
  fi
  echo "========================================="
  echo ""
}

invoke_deploy() {
  local deploy_script="${INSTALL_DIR}/scripts/deploy.sh"
  if [[ ! -f "$deploy_script" ]]; then
    log_error "未找到部署脚本: $deploy_script"
    exit 1
  fi

  run_fs chmod +x "$deploy_script"
  RUN_DEPLOY_AS_SUDO=false
  if [[ "$USE_SUDO_DOCKER" == true || "$USE_SUDO_FS" == true ]]; then
    RUN_DEPLOY_AS_SUDO=true
  fi

  local backend_config_env="${BACKEND_CONFIG_FILE_REL}"
  if [[ -z "$BACKEND_CONFIG_FILE_ABS" ]]; then
    backend_config_env="./backend/configs/config.docker.yaml"
  fi

  log_info "开始执行部署..."
  if [[ "$RUN_DEPLOY_AS_SUDO" == true ]]; then
    (cd "$INSTALL_DIR" && run_root env \
      BACKEND_CONFIG_FILE="$backend_config_env" \
      FRONTEND_BIND="$FRONTEND_BIND" \
      "$deploy_script" "${PASSTHROUGH_ARGS[@]}")
  else
    (cd "$INSTALL_DIR" && \
      BACKEND_CONFIG_FILE="$backend_config_env" \
      FRONTEND_BIND="$FRONTEND_BIND" \
      "$deploy_script" "${PASSTHROUGH_ARGS[@]}")
  fi
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --install-dir)
        INSTALL_DIR="${2:-}"
        shift 2
        ;;
      --repo)
        REPO_URL="${2:-}"
        shift 2
        ;;
      --branch)
        BRANCH="${2:-}"
        shift 2
        ;;
      --interactive)
        INTERACTIVE_MODE="true"
        shift
        ;;
      --non-interactive)
        INTERACTIVE_MODE="false"
        shift
        ;;
      --admin-username)
        ADMIN_USERNAME="${2:-}"
        CREATE_ADMIN="true"
        shift 2
        ;;
      --admin-email)
        ADMIN_EMAIL="${2:-}"
        CREATE_ADMIN="true"
        shift 2
        ;;
      --admin-password)
        ADMIN_PASSWORD="${2:-}"
        CREATE_ADMIN="true"
        shift 2
        ;;
      --skip-admin)
        CREATE_ADMIN="false"
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        PASSTHROUGH_ARGS+=("$1")
        shift
        ;;
    esac
  done
}

should_run_interactive() {
  if [[ "$INTERACTIVE_MODE" == "true" ]]; then
    return 0
  fi
  if [[ "$INTERACTIVE_MODE" == "false" ]]; then
    return 1
  fi
  if contains_passthrough_arg "--down"; then
    return 1
  fi
  if [[ -t 0 && ${#PASSTHROUGH_ARGS[@]} -eq 0 ]]; then
    return 0
  fi
  return 1
}

main() {
  parse_args "$@"
  ensure_environment
  local interactive_enabled="false"
  local down_mode="false"

  if contains_passthrough_arg "--down"; then
    down_mode="true"
  fi

  if should_run_interactive; then
    interactive_enabled="true"
    run_interactive_wizard
  fi

  prepare_repo

  if [[ "$interactive_enabled" == "true" ]]; then
    render_backend_config
    build_interactive_deploy_args
  else
    if [[ "$CREATE_ADMIN" == "auto" ]]; then
      CREATE_ADMIN="false"
    fi
    if [[ "$CREATE_ADMIN" == "true" ]]; then
      validate_admin_config
    fi
    log_info "使用非交互模式，透传参数给 deploy.sh: ${PASSTHROUGH_ARGS[*]:-(无)}"
  fi

  invoke_deploy

  if [[ "$down_mode" == "true" ]]; then
    log_info "已完成下线操作。"
    exit 0
  fi

  bootstrap_admin_account
  print_success_summary
}

main "$@"
