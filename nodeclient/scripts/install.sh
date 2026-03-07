#!/bin/bash
# NodePass Client 一键安装脚本

set -Eeuo pipefail

SERVICE_NAME="nodeclient"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
CONFIG_DIR="/etc/nodeclient"
CONFIG_FILE="${CONFIG_DIR}/config.yaml"
CACHE_DIR="/var/lib/nodeclient"
LOG_DIR="/var/log/nodeclient"

HUB_URL=""
TOKEN=""
INSTALL_DIR="/opt/nodeclient"
UNINSTALL="false"
UPGRADE="false"
SHOW_VERSION="false"
OS_NAME=""
ARCH_NAME=""

usage() {
  cat <<'EOF'
NodePass Client 一键安装/卸载脚本

安装:
  install.sh --hub-url <url> --token <token> [--install-dir <dir>]

升级:
  install.sh --upgrade [--hub-url <url>] [--install-dir <dir>]

卸载:
  install.sh --uninstall [--install-dir <dir>]

查看版本:
  install.sh --version [--install-dir <dir>]

参数:
  --hub-url <url>      面板地址 (安装时必填)
  --token <token>      节点 Token (安装时必填)
  --install-dir <dir>  安装目录 (默认: /opt/nodeclient)
  --upgrade            升级 nodeclient（二进制与服务）
  --uninstall          卸载 nodeclient
  --version            查看已安装版本
  -h, --help           显示帮助
EOF
}

log_info() {
  echo "[INFO] $*"
}

log_warn() {
  echo "[WARN] $*" >&2
}

log_error() {
  echo "[ERROR] $*" >&2
}

fail() {
  log_error "$*"
  exit 1
}

require_cmd() {
  local cmd="$1"
  command -v "$cmd" >/dev/null 2>&1 || fail "缺少命令: $cmd"
}

run_step() {
  local desc="$1"
  shift
  if "$@"; then
    return 0
  fi
  fail "${desc} 失败"
}

check_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    fail "请使用 root 用户执行该脚本"
  fi
}

detect_platform() {
  local raw_os raw_arch
  raw_os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  raw_arch="$(uname -m)"

  if [[ "${raw_os}" != "linux" ]]; then
    fail "仅支持 Linux 系统，当前系统: ${raw_os}"
  fi
  OS_NAME="${raw_os}"

  case "${raw_arch}" in
    x86_64|amd64)
      ARCH_NAME="amd64"
      ;;
    aarch64|arm64)
      ARCH_NAME="arm64"
      ;;
    *)
      fail "不支持的架构: ${raw_arch} (仅支持 amd64/arm64)"
      ;;
  esac
}

read_config_value() {
  local key="$1"
  if [[ ! -f "${CONFIG_FILE}" ]]; then
    return 1
  fi
  awk -F': ' -v key="$key" '
    $1 == key {
      value = $2
      gsub(/^"/, "", value)
      gsub(/"$/, "", value)
      print value
      exit
    }
  ' "${CONFIG_FILE}"
}

get_binary_version() {
  if [[ -x "${INSTALL_DIR}/nodeclient" ]]; then
    "${INSTALL_DIR}/nodeclient" --version 2>/dev/null || echo "unknown"
    return
  fi
  echo "not-installed"
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --hub-url)
        [[ $# -ge 2 ]] || fail "--hub-url 缺少参数"
        HUB_URL="$2"
        shift 2
        ;;
      --token)
        [[ $# -ge 2 ]] || fail "--token 缺少参数"
        TOKEN="$2"
        shift 2
        ;;
      --install-dir)
        [[ $# -ge 2 ]] || fail "--install-dir 缺少参数"
        INSTALL_DIR="$2"
        shift 2
        ;;
      --uninstall)
        UNINSTALL="true"
        shift
        ;;
      --upgrade)
        UPGRADE="true"
        shift
        ;;
      --version)
        SHOW_VERSION="true"
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        fail "未知参数: $1"
        ;;
    esac
  done
}

download_and_install_binary() {
  local base_url binary_url checksum_url tmp_dir tmp_bin tmp_sha expected_sha actual_sha

  base_url="${HUB_URL%/}"
  binary_url="${base_url}/downloads/nodeclient-${OS_NAME}-${ARCH_NAME}"
  checksum_url="${binary_url}.sha256"

  tmp_dir="$(mktemp -d)"
  tmp_bin="${tmp_dir}/nodeclient"
  tmp_sha="${tmp_dir}/nodeclient.sha256"
  trap 'rm -rf "${tmp_dir}"' RETURN

  log_info "下载二进制: ${binary_url}"
  run_step "下载 nodeclient 二进制" curl -fsSL "${binary_url}" -o "${tmp_bin}"

  log_info "下载校验文件: ${checksum_url}"
  run_step "下载 sha256 校验文件" curl -fsSL "${checksum_url}" -o "${tmp_sha}"

  expected_sha="$(awk 'NR==1 {print $1}' "${tmp_sha}")"
  [[ -n "${expected_sha}" ]] || fail "解析 sha256 校验值失败"

  actual_sha="$(sha256sum "${tmp_bin}" | awk '{print $1}')"
  [[ "${actual_sha}" == "${expected_sha}" ]] || fail "sha256 校验失败: expected=${expected_sha}, actual=${actual_sha}"

  run_step "创建安装目录" mkdir -p "${INSTALL_DIR}"
  run_step "安装 nodeclient 二进制" install -m 0755 "${tmp_bin}" "${INSTALL_DIR}/nodeclient"
}

write_config() {
  run_step "创建配置目录" mkdir -p "${CONFIG_DIR}"
  run_step "创建缓存目录" mkdir -p "${CACHE_DIR}"
  run_step "创建日志目录" mkdir -p "${LOG_DIR}"

  cat > "${CONFIG_FILE}" <<EOF
hub_url: "${HUB_URL}"
node_token: "${TOKEN}"
cache_path: "${CACHE_DIR}/config.json"
log_path: "${LOG_DIR}/"
EOF

  run_step "同步运行目录配置" mkdir -p "${INSTALL_DIR}/configs"
  run_step "复制运行目录配置" cp "${CONFIG_FILE}" "${INSTALL_DIR}/configs/config.yaml"
}

write_service_file() {
  cat > "${SERVICE_FILE}" <<EOF
[Unit]
Description=NodePass Client Agent
After=network.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/nodeclient --config ${CONFIG_FILE}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
}

enable_and_start_service() {
  run_step "重载 systemd 配置" systemctl daemon-reload
  run_step "设置开机自启" systemctl enable "${SERVICE_NAME}"
  run_step "启动服务" systemctl start "${SERVICE_NAME}"
}

show_status() {
  run_step "检查服务状态" systemctl --no-pager status "${SERVICE_NAME}"
}

do_install() {
  [[ -n "${HUB_URL}" ]] || fail "--hub-url 为必填参数"
  [[ -n "${TOKEN}" ]] || fail "--token 为必填参数"

  detect_platform

  require_cmd curl
  require_cmd sha256sum
  require_cmd install
  require_cmd systemctl

  download_and_install_binary
  write_config
  write_service_file
  enable_and_start_service
  show_status

  echo "安装完成! 节点将自动注册到面板。"
}

do_upgrade() {
  detect_platform
  require_cmd curl
  require_cmd sha256sum
  require_cmd install
  require_cmd systemctl

  local old_version new_version
  old_version="$(get_binary_version)"

  if [[ -z "${HUB_URL}" ]]; then
    HUB_URL="$(read_config_value "hub_url" || true)"
  fi
  [[ -n "${HUB_URL}" ]] || fail "升级模式下需要 --hub-url，或已存在 ${CONFIG_FILE}"

  download_and_install_binary
  run_step "重载 systemd 配置" systemctl daemon-reload
  run_step "重启服务" systemctl restart "${SERVICE_NAME}"

  new_version="$(get_binary_version)"
  log_info "升级完成: ${old_version} -> ${new_version}"
  show_status
}

do_uninstall() {
  require_cmd systemctl

  if [[ -f "${SERVICE_FILE}" ]]; then
    if systemctl list-unit-files | grep -q "^${SERVICE_NAME}\.service"; then
      if systemctl is-enabled "${SERVICE_NAME}" >/dev/null 2>&1; then
        run_step "关闭开机自启" systemctl disable "${SERVICE_NAME}"
      fi
      if systemctl is-active "${SERVICE_NAME}" >/dev/null 2>&1; then
        run_step "停止服务" systemctl stop "${SERVICE_NAME}"
      fi
    fi

    run_step "删除服务文件" rm -f "${SERVICE_FILE}"
    run_step "重载 systemd 配置" systemctl daemon-reload
  else
    log_warn "未检测到服务文件: ${SERVICE_FILE}"
  fi

  if [[ -d "${INSTALL_DIR}" ]]; then
    run_step "删除安装目录" rm -rf "${INSTALL_DIR}"
  fi
  if [[ -d "${CONFIG_DIR}" ]]; then
    run_step "删除配置目录" rm -rf "${CONFIG_DIR}"
  fi
  if [[ -d "${CACHE_DIR}" ]]; then
    run_step "删除缓存目录" rm -rf "${CACHE_DIR}"
  fi
  if [[ -d "${LOG_DIR}" ]]; then
    run_step "删除日志目录" rm -rf "${LOG_DIR}"
  fi

  echo "卸载完成。"
}

main() {
  parse_args "$@"
  check_root

  if [[ "${SHOW_VERSION}" == "true" ]]; then
    echo "nodeclient version: $(get_binary_version)"
    exit 0
  fi

  if [[ "${UNINSTALL}" == "true" ]]; then
    do_uninstall
    exit 0
  fi

  if [[ "${UPGRADE}" == "true" ]]; then
    do_upgrade
    exit 0
  fi

  do_install
}

main "$@"
