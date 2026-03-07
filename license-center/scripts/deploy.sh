#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'USAGE'
License Center 一键部署脚本

用法:
  ./scripts/deploy.sh [--down] [--no-build]

参数:
  --down      停止并移除服务
  --no-build  启动时不重新构建镜像
USAGE
}

DOWN=false
NO_BUILD=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --down)
      DOWN=true
      shift
      ;;
    --no-build)
      NO_BUILD=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "[ERROR] 未知参数: $1" >&2
      usage
      exit 1
      ;;
  esac
done

cd "$ROOT_DIR"

if [[ "$DOWN" == true ]]; then
  docker compose down
  echo "[INFO] 服务已停止"
  exit 0
fi

if [[ "$NO_BUILD" == true ]]; then
  docker compose up -d
else
  docker compose up -d --build
fi

echo "[INFO] 部署完成: http://127.0.0.1:8090"
