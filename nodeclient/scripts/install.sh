#!/bin/bash
# NodePass Client 一键安装脚本

set -Eeuo pipefail

SERVICE_NAME="nodeclient"
NODE_ID=""
GROUP_ID=""
NODE_ROLE="both"
CONNECT_HOST=""
DEBUG_MODE="false"
EGRESS_INTERFACE=""

HUB_URL=""
TOKEN=""
INSTALL_DIR="/opt/nodeclient"
UNINSTALL="false"
UPGRADE="false"
SHOW_VERSION="false"
OS_NAME=""
ARCH_NAME=""

BASE_CONFIG_DIR="/etc/nodeclient"
BASE_CACHE_DIR="/var/lib/nodeclient"
BASE_LOG_DIR="/var/log/nodeclient"
LEGACY_CONFIG_FILE="/etc/nodeclient/config.yaml"

SERVICE_FILE=""
CONFIG_DIR=""
CONFIG_FILE=""
CACHE_DIR=""
LOG_DIR=""

usage() {
  cat <<'USAGE'
NodePass Client 一键安装/升级/卸载脚本

安装:
  install.sh --hub-url <url> --node-id <uuid> --group-id <id> --token <token> [--service-name <name>] [--connect-host <ip|domain>] [--debug] [--egress-interface <iface>] [--install-dir <dir>]

升级:
  install.sh --upgrade [--service-name <name>] [--hub-url <url>] [--install-dir <dir>]

卸载:
  install.sh --uninstall [--service-name <name>] [--install-dir <dir>]

查看版本:
  install.sh --version [--install-dir <dir>]

参数:
  --hub-url <url>          面板地址 (安装时必填)
  --node-id <uuid>         节点唯一 ID (安装时必填)
  --group-id <id>          节点组 ID (安装时必填)
  --token <token>          节点认证 Token (安装时必填)
  --service-name <name>    systemd 服务名，默认 nodeclient
  --connect-host <value>   连接 IP/域名，默认自动探测公网 IPv4/IPv6
  --debug                  开启调试模式
  --egress-interface <if>  指定出口网卡（可选）
  --install-dir <dir>      安装目录 (默认: /opt/nodeclient)
  --upgrade                升级 nodeclient（二进制与服务）
  --uninstall              卸载指定服务实例
  --version                查看已安装版本
  -h, --help               显示帮助
USAGE
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

refresh_paths() {
  SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
  CONFIG_DIR="${BASE_CONFIG_DIR}/${SERVICE_NAME}"
  CONFIG_FILE="${CONFIG_DIR}/config.yaml"
  CACHE_DIR="${BASE_CACHE_DIR}/${SERVICE_NAME}"
  LOG_DIR="${BASE_LOG_DIR}/${SERVICE_NAME}"
}

validate_service_name() {
  if [[ ! "${SERVICE_NAME}" =~ ^[a-zA-Z0-9_.@-]{1,64}$ ]]; then
    fail "--service-name 格式无效，仅支持字母数字及 _.@-，长度 1-64"
  fi
}

validate_node_id() {
  if [[ -z "${NODE_ID}" ]]; then
    return 0
  fi

  if [[ ! "${NODE_ID}" =~ ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$ ]]; then
    fail "--node-id 必须是合法 UUID"
  fi
}

validate_node_role() {
  case "${NODE_ROLE}" in
    entry|exit|both)
      return 0
      ;;
    *)
      fail "--node-role 仅支持 entry / exit / both"
      ;;
  esac
}

validate_connect_host() {
  if [[ -z "${CONNECT_HOST}" ]]; then
    return 0
  fi

  if [[ "${CONNECT_HOST}" =~ [[:space:]] ]]; then
    fail "--connect-host 不能包含空白字符"
  fi
}

validate_egress_interface() {
  if [[ -z "${EGRESS_INTERFACE}" ]]; then
    return 0
  fi

  if [[ ! "${EGRESS_INTERFACE}" =~ ^[a-zA-Z0-9_.:@-]{1,64}$ ]]; then
    fail "--egress-interface 格式无效，仅支持字母数字及 _.:@-，长度 1-64"
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
  local file="${2:-${CONFIG_FILE}}"

  if [[ ! -f "${file}" ]]; then
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
  ' "${file}"
}

get_binary_version() {
  if [[ -x "${INSTALL_DIR}/nodeclient" ]]; then
    "${INSTALL_DIR}/nodeclient" --version 2>/dev/null || echo "unknown"
    return
  fi
  echo "not-installed"
}

generate_node_id() {
  if [[ -n "${NODE_ID}" ]]; then
    return
  fi

  if command -v uuidgen >/dev/null 2>&1; then
    NODE_ID="$(uuidgen | tr '[:upper:]' '[:lower:]')"
    return
  fi

  if [[ -r /proc/sys/kernel/random/uuid ]]; then
    NODE_ID="$(cat /proc/sys/kernel/random/uuid)"
    return
  fi

  fail "无法自动生成 UUID，请手动传入 --node-id"
}

collect_public_hosts() {
  local hosts=() value

  for endpoint in "https://api.ipify.org" "https://api64.ipify.org"; do
    value="$(curl -fsS --max-time 3 "${endpoint}" 2>/dev/null || true)"
    value="${value//$'\r'/}"
    value="${value//$'\n'/}"

    if [[ -z "${value}" ]]; then
      continue
    fi

    if [[ "${value}" =~ [[:space:]] ]]; then
      continue
    fi

    local duplicated="false"
    local existing
    for existing in "${hosts[@]:-}"; do
      if [[ "${existing}" == "${value}" ]]; then
        duplicated="true"
        break
      fi
    done
    if [[ "${duplicated}" == "false" ]]; then
      hosts+=("${value}")
    fi
  done

  if [[ ${#hosts[@]} -eq 0 ]]; then
    value="$(hostname -I 2>/dev/null | awk '{print $1}' || true)"
    if [[ -n "${value}" ]]; then
      hosts+=("${value}")
    fi
  fi

  printf '%s\n' "${hosts[@]:-}"
}

resolve_connect_host() {
  if [[ -n "${CONNECT_HOST}" ]]; then
    return
  fi

  local detected=()
  local item
  while IFS= read -r item; do
    [[ -n "${item}" ]] || continue
    detected+=("${item}")
  done < <(collect_public_hosts)

  if [[ ${#detected[@]} -eq 0 ]]; then
    log_warn "未探测到公网地址，connect_host 将留空"
    return
  fi

  if [[ ${#detected[@]} -gt 1 ]]; then
    log_warn "检测到多个公网地址: ${detected[*]}，默认使用第一个，可通过 --connect-host 指定"
  fi

  CONNECT_HOST="${detected[0]}"
  log_info "自动选择 connect_host: ${CONNECT_HOST}"
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
      --service-name)
        [[ $# -ge 2 ]] || fail "--service-name 缺少参数"
        SERVICE_NAME="$2"
        shift 2
        ;;
      --node-id)
        [[ $# -ge 2 ]] || fail "--node-id 缺少参数"
        NODE_ID="$2"
        shift 2
        ;;
      --group-id)
        [[ $# -ge 2 ]] || fail "--group-id 缺少参数"
        GROUP_ID="$2"
        shift 2
        ;;
      --node-role)
        [[ $# -ge 2 ]] || fail "--node-role 缺少参数"
        NODE_ROLE="$2"
        shift 2
        ;;
      --connect-host)
        [[ $# -ge 2 ]] || fail "--connect-host 缺少参数"
        CONNECT_HOST="$2"
        shift 2
        ;;
      --debug)
        DEBUG_MODE="true"
        shift
        ;;
      --egress-interface)
        [[ $# -ge 2 ]] || fail "--egress-interface 缺少参数"
        EGRESS_INTERFACE="$2"
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

  cat > "${CONFIG_FILE}" <<CONFIG
hub_url: "${HUB_URL}"
service_name: "${SERVICE_NAME}"
node_id: "${NODE_ID}"
group_id: ${GROUP_ID}
node_token: "${TOKEN}"
connection_address: "${CONNECT_HOST}"
cache_path: "${CACHE_DIR}/config.json"
heartbeat_interval: 30
config_check_interval: 60
traffic_report_interval: 60
debug_mode: ${DEBUG_MODE}
auto_start: true
CONFIG

  if [[ -n "${EGRESS_INTERFACE}" ]]; then
    echo "egress_interface: \"${EGRESS_INTERFACE}\"" >> "${CONFIG_FILE}"
  fi

  run_step "同步运行目录配置" mkdir -p "${INSTALL_DIR}/configs"
  run_step "复制运行目录配置" cp "${CONFIG_FILE}" "${INSTALL_DIR}/configs/${SERVICE_NAME}.yaml"
}

write_service_file() {
  cat > "${SERVICE_FILE}" <<SERVICE
[Unit]
Description=NodePass Client Agent (${SERVICE_NAME})
After=network.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/nodeclient --config ${CONFIG_FILE}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
SERVICE
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
  [[ -n "${NODE_ID}" ]] || fail "--node-id 为必填参数"
  [[ -n "${GROUP_ID}" ]] || fail "--group-id 为必填参数"
  [[ -n "${TOKEN}" ]] || fail "--token 为必填参数"

  detect_platform

  require_cmd curl
  require_cmd sha256sum
  require_cmd install
  require_cmd systemctl

  validate_node_id
  resolve_connect_host

  download_and_install_binary
  write_config
  write_service_file
  enable_and_start_service
  show_status

  echo
  echo "安装完成!"
  echo "服务名称: ${SERVICE_NAME}"
  echo "节点 ID: ${NODE_ID}"
  echo "节点角色: ${NODE_ROLE}"
  echo "连接地址: ${CONNECT_HOST:-未指定}"
  echo "配置文件: ${CONFIG_FILE}"
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
    HUB_URL="$(read_config_value "hub_url" "${CONFIG_FILE}" || true)"
  fi
  if [[ -z "${HUB_URL}" ]]; then
    HUB_URL="$(read_config_value "hub_url" "${LEGACY_CONFIG_FILE}" || true)"
  fi
  [[ -n "${HUB_URL}" ]] || fail "升级模式下需要 --hub-url，或已存在 ${CONFIG_FILE}"

  download_and_install_binary
  run_step "重载 systemd 配置" systemctl daemon-reload

  if systemctl list-unit-files | grep -q "^${SERVICE_NAME}\\.service"; then
    run_step "重启服务" systemctl restart "${SERVICE_NAME}"
    show_status
  else
    log_warn "服务 ${SERVICE_NAME} 不存在，仅完成二进制升级"
  fi

  new_version="$(get_binary_version)"
  log_info "升级完成: ${old_version} -> ${new_version}"
}

do_uninstall() {
  require_cmd systemctl

  if systemctl list-unit-files | grep -q "^${SERVICE_NAME}\\.service"; then
    if systemctl is-enabled "${SERVICE_NAME}" >/dev/null 2>&1; then
      run_step "关闭开机自启" systemctl disable "${SERVICE_NAME}"
    fi
    if systemctl is-active "${SERVICE_NAME}" >/dev/null 2>&1; then
      run_step "停止服务" systemctl stop "${SERVICE_NAME}"
    fi
  fi

  if [[ -f "${SERVICE_FILE}" ]]; then
    run_step "删除服务文件" rm -f "${SERVICE_FILE}"
    run_step "重载 systemd 配置" systemctl daemon-reload
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
  if [[ -f "${INSTALL_DIR}/configs/${SERVICE_NAME}.yaml" ]]; then
    run_step "删除运行目录配置" rm -f "${INSTALL_DIR}/configs/${SERVICE_NAME}.yaml"
  fi

  echo "卸载完成。已移除服务实例 ${SERVICE_NAME}。"
  echo "说明: 二进制文件保留在 ${INSTALL_DIR}/nodeclient，避免影响其他实例。"
}

main() {
  parse_args "$@"
  check_root

  validate_service_name
  validate_node_role
  validate_connect_host
  validate_egress_interface
  refresh_paths

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
