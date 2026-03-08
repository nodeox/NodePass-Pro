#!/usr/bin/env bash
set -euo pipefail

# ============================================================================
# NodePass License Center 一键部署脚本 v0.2.0
# ============================================================================

REPO_URL="${REPO_URL:-https://github.com/nodeox/NodePass-Pro.git}"
BRANCH="${BRANCH:-main}"
INSTALL_DIR="${INSTALL_DIR:-/opt/nodepass-license-center}"
PROJECT_SUBDIR="${PROJECT_SUBDIR:-license-center}"
ACTION="install" # install / upgrade / uninstall

SUDO_CMD=""
PKG_MANAGER=""
RUN_DEPLOY_AS_SUDO=false

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

usage() {
  cat <<'USAGE'
NodePass License Center 一键部署脚本 v0.2.0

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
  -h, --help                   显示帮助

示例:
  # 安装
  bash install.sh --install

  # 升级
  bash install.sh --upgrade

  # 卸载
  bash install.sh --uninstall

  # 自定义安装目录
  bash install.sh --install --install-dir /data/license-center

远程一键安装:
  bash <(curl -fsSL "https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license-center/install.sh?t=$(date +%s)") --install

版本: v0.2.0
功能: 授权管理、域名绑定、监控告警、Webhook 通知
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

prepare_repo() {
  log_step "准备代码仓库..."

  prepare_install_dir

  if [[ -d "${INSTALL_DIR}/.git" ]]; then
    log_info "更新仓库: ${INSTALL_DIR}"
    run_root git -C "${INSTALL_DIR}" fetch origin "${BRANCH}"
    run_root git -C "${INSTALL_DIR}" checkout "${BRANCH}"
    run_root git -C "${INSTALL_DIR}" reset --hard "origin/${BRANCH}"
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

backup_config() {
  local project_dir="$1"
  local config_file="${project_dir}/configs/config.yaml"

  if [[ -f "$config_file" ]]; then
    local backup_file="${config_file}.backup.$(date +%Y%m%d_%H%M%S)"
    log_info "备份配置文件: $backup_file"
    run_root cp "$config_file" "$backup_file"
  fi
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
  log_step "检查服务健康状态..."

  local max_attempts=30
  local attempt=0

  while [[ $attempt -lt $max_attempts ]]; do
    if curl -sf http://127.0.0.1:8090/health >/dev/null 2>&1; then
      log_info "✓ 服务健康检查通过"
      return 0
    fi

    attempt=$((attempt + 1))
    sleep 2
  done

  log_warn "服务健康检查超时，请手动检查"
  return 1
}

show_success_info() {
  cat <<EOF

${GREEN}╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║  🎉 NodePass License Center 部署成功！                         ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝${NC}

${BLUE}📍 访问地址:${NC}
  • 健康检查: http://127.0.0.1:8090/health
  • 管理面板: http://127.0.0.1:8090/console
  • API 文档: http://127.0.0.1:8090/api/v1

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
}

show_upgrade_info() {
  cat <<EOF

${GREEN}╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║  ✨ NodePass License Center 升级成功！                         ║
║                                                                ║
╚══════════════════════════════��═════════════════════════════════╝${NC}

${BLUE}📍 访问地址:${NC}
  • 管理面板: http://127.0.0.1:8090/console

${BLUE}🆕 v0.2.0 新功能:${NC}
  • ✨ 域名绑定功能（防止授权码多站点共享）
  • ✨ 完整的 Web 管理界面（React + TypeScript）
  • ✨ 实时监控仪表盘
  • ✨ 自动告警系统
  • ✨ Webhook 事件通知
  • ✨ 批量操作功能
  • ✨ 标签管理系统

${BLUE}⚠️  重要提示:${NC}
  • 配置文件已备份，请检查新配置项
  • 数据库已自动迁移
  • 建议查看更新日志了解详细变更

EOF
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
║              License Center v0.2.0                             ║
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

  prepare_repo

  local project_dir
  project_dir="$(resolve_project_dir)"

  if [[ "$ACTION" == "upgrade" ]]; then
    log_info "执行升级部署..."
    backup_config "$project_dir"
  else
    log_info "执行全新安装..."
  fi

  run_deploy "$project_dir"

  # 等待服务启动
  sleep 5
  check_service_health

  if [[ "$ACTION" == "upgrade" ]]; then
    show_upgrade_info
  else
    show_success_info
  fi
}

main "$@"
