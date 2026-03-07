#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"

PANEL_FILE="${ROOT_DIR}/VERSION"
BACKEND_FILE="${ROOT_DIR}/backend/VERSION"
FRONTEND_FILE="${ROOT_DIR}/frontend/VERSION"
NODECLIENT_FILE="${ROOT_DIR}/nodeclient/VERSION"

usage() {
  cat <<'EOF'
NodePass Pro 版本管理脚本

用法:
  ./scripts/version.sh show
  ./scripts/version.sh export-env
  ./scripts/version.sh set [--panel x.y.z] [--backend x.y.z] [--frontend x.y.z] [--nodeclient x.y.z]

说明:
  - show:       显示当前版本信息
  - export-env: 输出可用于 shell 的版本环境变量
  - set:        更新一个或多个版本文件
EOF
}

read_version() {
  local file="$1"
  if [[ -f "$file" ]]; then
    tr -d '[:space:]' <"$file"
  fi
}

write_version() {
  local file="$1"
  local version="$2"
  printf '%s\n' "$version" >"$file"
}

validate_version() {
  local value="$1"
  if [[ ! "$value" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
    echo "非法版本号: ${value}" >&2
    exit 1
  fi
}

show_versions() {
  local panel backend frontend nodeclient
  panel="$(read_version "$PANEL_FILE")"
  backend="$(read_version "$BACKEND_FILE")"
  frontend="$(read_version "$FRONTEND_FILE")"
  nodeclient="$(read_version "$NODECLIENT_FILE")"

  echo "panel: ${panel:-unknown}"
  echo "backend: ${backend:-unknown}"
  echo "frontend: ${frontend:-unknown}"
  echo "nodeclient: ${nodeclient:-unknown}"
}

export_env() {
  local panel backend frontend nodeclient
  panel="$(read_version "$PANEL_FILE")"
  backend="$(read_version "$BACKEND_FILE")"
  frontend="$(read_version "$FRONTEND_FILE")"
  nodeclient="$(read_version "$NODECLIENT_FILE")"

  echo "PANEL_VERSION=${panel:-dev}"
  echo "BACKEND_VERSION=${backend:-dev}"
  echo "FRONTEND_VERSION=${frontend:-dev}"
  echo "NODECLIENT_VERSION=${nodeclient:-dev}"
}

set_versions() {
  local panel=""
  local backend=""
  local frontend=""
  local nodeclient=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --panel)
        panel="${2:-}"
        shift 2
        ;;
      --backend)
        backend="${2:-}"
        shift 2
        ;;
      --frontend)
        frontend="${2:-}"
        shift 2
        ;;
      --nodeclient)
        nodeclient="${2:-}"
        shift 2
        ;;
      *)
        echo "未知参数: $1" >&2
        usage
        exit 1
        ;;
    esac
  done

  if [[ -z "$panel" && -z "$backend" && -z "$frontend" && -z "$nodeclient" ]]; then
    echo "未指定任何版本参数。" >&2
    usage
    exit 1
  fi

  if [[ -n "$panel" ]]; then
    validate_version "$panel"
    write_version "$PANEL_FILE" "$panel"
  fi
  if [[ -n "$backend" ]]; then
    validate_version "$backend"
    write_version "$BACKEND_FILE" "$backend"
  fi
  if [[ -n "$frontend" ]]; then
    validate_version "$frontend"
    write_version "$FRONTEND_FILE" "$frontend"
  fi
  if [[ -n "$nodeclient" ]]; then
    validate_version "$nodeclient"
    write_version "$NODECLIENT_FILE" "$nodeclient"
  fi
}

main() {
  local command="${1:-show}"
  case "$command" in
    show)
      show_versions
      ;;
    export-env)
      export_env
      ;;
    set)
      shift || true
      set_versions "$@"
      show_versions
      ;;
    -h|--help|help)
      usage
      ;;
    *)
      echo "未知命令: $command" >&2
      usage
      exit 1
      ;;
  esac
}

main "$@"
