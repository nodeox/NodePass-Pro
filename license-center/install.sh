#!/usr/bin/env bash
set -euo pipefail

# ============================================================================
# NodePass License Center 一键部署脚本 v0.3.0
# ============================================================================

REPO_URL="${REPO_URL:-https://github.com/nodeox/NodePass-Pro.git}"
BRANCH="${BRANCH:-main}"
INSTALL_DIR="${INSTALL_DIR:-/opt/nodepass-license-center}"
PROJECT_SUBDIR="${PROJECT_SUBDIR:-license-center}"
ACTION="install" # install / upgrade / uninstall
BACKUP_DIR="${BACKUP_DIR:-${INSTALL_DIR%/}-backups}"
LAST_CONFIG_BACKUP=""
GENERATED_CONFIG_WARNING=false

# 镜像相关配置
USE_PREBUILT_IMAGE=true
IMAGE_URL=""
IMAGE_FILE=""
IMAGE_NAME="ghcr.io/nodeox/license-center"
IMAGE_VERSION="latest"

SUDO_CMD=""
PKG_MANAGER=""
RUN_DEPLOY_AS_SUDO=false
SKIP_HEALTH_CHECK=false

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

usage() {
  cat <<'USAGE'
NodePass License Center 一键部署脚本 v0.3.0

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
  --use-image                  使用预构建镜像（默认）
  --build-source               使用源码构建（覆盖默认镜像部署）
  --image-url <url>            镜像下载地址
  --image-file <file>          本地镜像文件路径
  --image-name <name>          镜像名称（默认: ghcr.io/nodeox/license-center）
  --image-version <ver>        镜像版本（默认: latest）
  -h, --help                   显示帮助

示例:
  # 安装（默认镜像部署）
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

远程一键安装:
  bash <(curl -fsSL "https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license-center/install.sh?t=$(date +%s)") --install

版本: v0.3.0
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
          print "  " key ": \"" value "\""
          updated=1
        }
        in_section=0
      }

      if (in_section && $0 ~ "^[[:space:]]*" key ":[[:space:]]*") {
        print "  " key ": \"" value "\""
        updated=1
        next
      }

      print $0
    }
    END {
      if (in_section && !updated) {
        print "  " key ": \"" value "\""
        updated=1
      }
      if (!updated) {
        if (!section_seen) {
          print section ":"
        }
        print "  " key ": \"" value "\""
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

  ensure_file_readable "$config_file"
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

  log_info "✓ 依赖检查完成"
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
    gunzip -c "$image_file" | docker load
  else
    docker load -i "$image_file"
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
    docker pull "${IMAGE_NAME}:${IMAGE_VERSION}"
  fi

  log_info "✓ 镜像准备完成"
}

prepare_config_only() {
  log_step "准备配置文件..."

  prepare_install_dir

  # 只下载必要的配置文件
  local project_dir="${INSTALL_DIR}/${PROJECT_SUBDIR}"
  run_root mkdir -p "$project_dir"
  run_root mkdir -p "${project_dir}/configs"
  run_root mkdir -p "${project_dir}/scripts"

  # 下载 docker-compose.prod.yml
  log_info "下载 docker-compose 配置..."
  local compose_url="https://raw.githubusercontent.com/nodeox/NodePass-Pro/${BRANCH}/${PROJECT_SUBDIR}/docker-compose.prod.yml"
  if command -v curl >/dev/null 2>&1; then
    run_root curl -fsSL "$compose_url" -o "${project_dir}/docker-compose.yml"
  else
    run_root wget -q "$compose_url" -O "${project_dir}/docker-compose.yml"
  fi

  # 下载配置文件模板（升级时保留用户现有配置）
  if [[ -f "${project_dir}/configs/config.yaml" ]]; then
    log_info "检测到已有配置，跳过覆盖: ${project_dir}/configs/config.yaml"
  else
    log_info "下载配置文件模板..."
    local config_url="https://raw.githubusercontent.com/nodeox/NodePass-Pro/${BRANCH}/${PROJECT_SUBDIR}/configs/config.yaml"
    if command -v curl >/dev/null 2>&1; then
      run_root curl -fsSL "$config_url" -o "${project_dir}/configs/config.yaml"
    else
      run_root wget -q "$config_url" -O "${project_dir}/configs/config.yaml"
    fi
  fi
  ensure_file_readable "${project_dir}/configs/config.yaml"

  # 下载部署脚本（镜像模式仍需通过 deploy.sh 统一执行 up/down/logs）
  log_info "下载部署脚本..."
  local deploy_url="https://raw.githubusercontent.com/nodeox/NodePass-Pro/${BRANCH}/${PROJECT_SUBDIR}/scripts/deploy.sh"
  if command -v curl >/dev/null 2>&1; then
    run_root curl -fsSL "$deploy_url" -o "${project_dir}/scripts/deploy.sh"
  else
    run_root wget -q "$deploy_url" -O "${project_dir}/scripts/deploy.sh"
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
APP_PORT=8090
BUILD_VERSION=${IMAGE_VERSION}
GIN_MODE=release
IMAGE_NAME=${IMAGE_NAME}
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
  ensure_file_readable "$target"
  log_info "已恢复用户配置: $target"
}

run_deploy() {
  local project_dir="$1"

  if [[ -z "$project_dir" ]]; then
    log_error "未找到部署脚本，请检查 --repo 与 --project-subdir 是否正确。"
    exit 1
  fi

  log_step "开始部署服务..."

  if [[ "$RUN_DEPLOY_AS_SUDO" == true ]]; then
    (cd "$project_dir" && run_root ./scripts/deploy.sh)
  else
    (cd "$project_dir" && ./scripts/deploy.sh)
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

  if [[ "$RUN_DEPLOY_AS_SUDO" == true ]]; then
    (cd "$project_dir" && run_root ./scripts/deploy.sh --down) || true
  else
    (cd "$project_dir" && ./scripts/deploy.sh --down) || true
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
  log_info "查看日志命令: cd ${INSTALL_DIR}/${PROJECT_SUBDIR} && ./scripts/deploy.sh --logs"
  return 1
}

show_success_info() {
  local app_port="$1"

  cat <<EOF

${GREEN}╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║  🎉 NodePass License Center 部署成功！                         ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝${NC}

${BLUE}📍 访问地址:${NC}
  • 健康检查: http://127.0.0.1:${app_port}/health
  • 管理面板: http://127.0.0.1:${app_port}/console
  • API 文档: http://127.0.0.1:${app_port}/api/v1

${BLUE}🔐 管理员账号:${NC}
  • 首次初始化会使用配置文件中的 admin.username / admin.password
  • 请确保已在 ${INSTALL_DIR}/${PROJECT_SUBDIR}/configs/config.yaml 设置强密码

${BLUE}📚 功能特性:${NC}
  • ✅ 授权码管理（生成、吊销、转移）
  • ✅ 域名绑定（防止多站点共享）
  • ✅ 套餐管理（版本限制、机器数量）
  • ✅ 监控告警（实时统计、趋势分析）
  • ✅ Webhook 通知（事件推送）
  • ✅ 标签管理（授权码分类）
  • ✅ 安全增强（限流、签名、IP 白名单）

${BLUE}🔧 常用命令:${NC}
  • 查看日志: docker compose -f ${INSTALL_DIR}/${PROJECT_SUBDIR}/docker-compose.yml logs -f
  • 重启服务: docker compose -f ${INSTALL_DIR}/${PROJECT_SUBDIR}/docker-compose.yml restart
  • 停止服务: docker compose -f ${INSTALL_DIR}/${PROJECT_SUBDIR}/docker-compose.yml down
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

  cat <<EOF

${GREEN}╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║  ✨ NodePass License Center 升级成功！                         ║
║                                                                ║
╚══════════════════════════════��═════════════════════════════════╝${NC}

${BLUE}📍 访问地址:${NC}
  • 管理面板: http://127.0.0.1:${app_port}/console

${BLUE}🆕 v0.3.0 新功能:${NC}
  • ✨ 优化的 Docker 多阶段构建（前后端一体化）
  • ✨ 增强的部署脚本（支持更多操作）
  • ✨ Makefile 快捷命令支持
  • ✨ 完善的环境变量配置
  • ✨ 改进的日志和监控
  • ✨ 数据库备份恢复功能

${BLUE}⚠️  重要提示:${NC}
  • 配置文件已备份，请检查新配置项
  • 数据库已自动迁移
  • 新增 .env 文件支持环境变量配置
  • 建议查看更新日志了解详细变更

${BLUE}🔧 新增命令:${NC}
  • make help     - 查看所有可用命令
  • make status   - 查看服务状态
  • make logs     - 查看实时日志
  • make backup-db - 备份数据库

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
║              License Center v0.3.0                             ║
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
