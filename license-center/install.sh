#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/nodeox/NodePass-Pro.git}"
BRANCH="${BRANCH:-main}"
INSTALL_DIR="${INSTALL_DIR:-/opt/nodepass-license-center}"
PROJECT_SUBDIR="${PROJECT_SUBDIR:-license-center}"
ACTION="install" # install / upgrade / uninstall

SUDO_CMD=""
PKG_MANAGER=""
RUN_DEPLOY_AS_SUDO=false

usage() {
  cat <<'USAGE'
NodePass License Center 一键脚本

用法:
  bash install.sh [--install|--upgrade|--uninstall] [--repo <url>] [--branch <branch>] [--install-dir <dir>] [--project-subdir <name>]

参数:
  --install                    安装（默认）
  --upgrade                    升级
  --uninstall                  卸载
  --repo <url>                 仓库地址（默认: https://github.com/nodeox/NodePass-Pro.git）
  --branch <branch>            分支（默认: main）
  --install-dir <dir>          克隆目录（默认: /opt/nodepass-license-center）
  --project-subdir <name>      License Center 子目录（默认: license-center）
  -h, --help                   显示帮助
USAGE
}

log_info() { echo "[INFO] $*"; }
log_warn() { echo "[WARN] $*"; }
log_error() { echo "[ERROR] $*" >&2; }

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

ensure_env() {
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

run_deploy() {
  local project_dir="$1"

  if [[ -z "$project_dir" ]]; then
    log_error "未找到部署脚本，请检查 --repo 与 --project-subdir 是否正确。"
    exit 1
  fi

  if [[ "$RUN_DEPLOY_AS_SUDO" == true ]]; then
    (cd "$project_dir" && run_root ./scripts/deploy.sh)
  else
    (cd "$project_dir" && ./scripts/deploy.sh)
  fi
}

run_down() {
  local project_dir="$1"

  if [[ -z "$project_dir" ]]; then
    log_warn "未检测到可卸载的项目目录，跳过停止服务。"
    return
  fi

  if [[ "$RUN_DEPLOY_AS_SUDO" == true ]]; then
    (cd "$project_dir" && run_root ./scripts/deploy.sh --down) || true
  else
    (cd "$project_dir" && ./scripts/deploy.sh --down) || true
  fi
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
  log_info "卸载完成"
}

main() {
  parse_args "$@"
  ensure_env

  if [[ "$ACTION" == "uninstall" ]]; then
    do_uninstall
    exit 0
  fi

  prepare_repo

  local project_dir
  project_dir="$(resolve_project_dir)"

  if [[ "$ACTION" == "upgrade" ]]; then
    log_info "执行升级部署"
  else
    log_info "执行安装部署"
  fi

  run_deploy "$project_dir"
  log_info "完成: http://127.0.0.1:8090/health"
  log_info "管理面板: http://127.0.0.1:8090/console"
}

main "$@"
