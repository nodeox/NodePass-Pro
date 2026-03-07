#!/usr/bin/env bash
set -euo pipefail

REPO_URL="https://github.com/nodeox/NodePass-Pro.git"
BRANCH="main"
INSTALL_DIR="/opt/NodePass-Pro"

log_info() {
  echo "[INFO] $*"
}

log_error() {
  echo "[ERROR] $*" >&2
}

usage() {
  cat <<'EOF'
NodePass Pro 远程一键部署引导脚本

用法:
  bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh) [deploy 参数]

支持的引导参数:
  --install-dir <目录>     代码目录（默认: /opt/NodePass-Pro）
  --repo <地址>            仓库地址（默认: https://github.com/nodeox/NodePass-Pro.git）
  --branch <分支>          分支名（默认: main）
  -h, --help               显示帮助

其余参数会透传给 scripts/deploy.sh，例如:
  --with-caddy --domain panel.example.com --email admin@example.com
  --down
EOF
}

require_command() {
  local command_name="$1"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    log_error "缺少依赖命令: $command_name"
    exit 1
  fi
}

parse_args() {
  DEPLOY_ARGS=()
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
      -h|--help)
        usage
        exit 0
        ;;
      *)
        DEPLOY_ARGS+=("$1")
        shift
        ;;
    esac
  done
}

prepare_repo() {
  if [[ -d "${INSTALL_DIR}/.git" ]]; then
    log_info "检测到已有仓库，开始更新: ${INSTALL_DIR}"
    git -C "${INSTALL_DIR}" fetch origin "${BRANCH}"
    git -C "${INSTALL_DIR}" checkout "${BRANCH}"
    git -C "${INSTALL_DIR}" reset --hard "origin/${BRANCH}"
  else
    log_info "开始克隆仓库到: ${INSTALL_DIR}"
    rm -rf "${INSTALL_DIR}"
    git clone --branch "${BRANCH}" --depth 1 "${REPO_URL}" "${INSTALL_DIR}"
  fi
}

main() {
  parse_args "$@"
  require_command git
  require_command docker

  if ! docker info >/dev/null 2>&1; then
    log_error "Docker 未运行，请先启动 Docker。"
    exit 1
  fi

  prepare_repo

  local deploy_script="${INSTALL_DIR}/scripts/deploy.sh"
  if [[ ! -x "${deploy_script}" ]]; then
    log_error "未找到部署脚本: ${deploy_script}"
    exit 1
  fi

  log_info "执行部署脚本: ${deploy_script} ${DEPLOY_ARGS[*]:-}"
  (cd "${INSTALL_DIR}" && "${deploy_script}" "${DEPLOY_ARGS[@]}")
}

main "$@"
