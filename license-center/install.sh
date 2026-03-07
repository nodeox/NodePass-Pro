#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/nodeox/NodePass-License-Center.git}"
BRANCH="${BRANCH:-main}"
INSTALL_DIR="${INSTALL_DIR:-/opt/nodepass-license-center}"
ACTION="install" # install / upgrade / uninstall

usage() {
  cat <<'USAGE'
NodePass License Center 一键脚本

用法:
  bash install.sh [--install|--upgrade|--uninstall] [--repo <url>] [--branch <branch>] [--install-dir <dir>]
USAGE
}

log_info() { echo "[INFO] $*"; }
log_error() { echo "[ERROR] $*" >&2; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || { log_error "缺少依赖命令: $1"; exit 1; }
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

ensure_env() {
  require_cmd git
  require_cmd docker
  if ! docker compose version >/dev/null 2>&1; then
    log_error "缺少 docker compose"
    exit 1
  fi
}

prepare_repo() {
  if [[ -d "${INSTALL_DIR}/.git" ]]; then
    log_info "更新仓库: ${INSTALL_DIR}"
    git -C "${INSTALL_DIR}" fetch origin "${BRANCH}"
    git -C "${INSTALL_DIR}" checkout "${BRANCH}"
    git -C "${INSTALL_DIR}" reset --hard "origin/${BRANCH}"
  else
    log_info "克隆仓库到: ${INSTALL_DIR}"
    rm -rf "${INSTALL_DIR}"
    mkdir -p "$(dirname "${INSTALL_DIR}")"
    git clone --branch "${BRANCH}" --depth 1 "${REPO_URL}" "${INSTALL_DIR}"
  fi
}

do_uninstall() {
  if [[ ! -d "${INSTALL_DIR}" ]]; then
    log_info "目录不存在，跳过卸载"
    exit 0
  fi
  if [[ -f "${INSTALL_DIR}/scripts/deploy.sh" ]]; then
    (cd "${INSTALL_DIR}" && ./scripts/deploy.sh --down) || true
  fi
  rm -rf "${INSTALL_DIR}"
  log_info "卸载完成"
}

main() {
  parse_args "$@"

  if [[ "$ACTION" == "uninstall" ]]; then
    do_uninstall
    exit 0
  fi

  ensure_env
  prepare_repo

  if [[ "$ACTION" == "upgrade" ]]; then
    log_info "执行升级部署"
  else
    log_info "执行安装部署"
  fi

  (cd "${INSTALL_DIR}" && ./scripts/deploy.sh)
  log_info "完成: http://127.0.0.1:8090"
}

main "$@"
