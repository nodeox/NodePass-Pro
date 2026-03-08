#!/usr/bin/env bash
set -euo pipefail

# ============================================================================
# License Center 部署脚本 v0.4.0
# ============================================================================

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 默认参数
ACTION="up"
BUILD_FLAG="--build"
ENV_FILE=".env"
COMPOSE_FILE="docker-compose.yml"
COMPOSE_FILES=()
ENABLE_HTTPS_PROXY="${ENABLE_HTTPS_PROXY:-false}"
HTTPS_COMPOSE_FILE="${HTTPS_COMPOSE_FILE:-docker-compose.https.yml}"

log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_step() { echo -e "${BLUE}[STEP]${NC} $*"; }

usage() {
  cat <<'USAGE'
License Center 部署脚本 v0.4.0

用法:
  ./scripts/deploy.sh [选项]

操作:
  --up              启动服务（默认）
  --down            停止并移除服务
  --restart         重启服务
  --logs            查看日志
  --status          查看服务状态
  --clean           清理所有数据（危险操作）

选项:
  --no-build        启动时不重新构建镜像
  --build-only      仅构建镜像，不启动
  --pull            拉取最新基础镜像
  --with-https-proxy 启用 HTTPS 代理编排（叠加 docker-compose.https.yml）
  --https-file <file> 指定 HTTPS 代理 compose 文件（默认: docker-compose.https.yml）
  -f, --file <file> 指定 docker-compose 文件（默认: docker-compose.yml）
  -h, --help        显示帮助

示例:
  # 构建并启动
  ./scripts/deploy.sh

  # 仅启动（不重新构建）
  ./scripts/deploy.sh --up --no-build

  # 停止服务
  ./scripts/deploy.sh --down

  # 重启服务
  ./scripts/deploy.sh --restart

  # 查看日志
  ./scripts/deploy.sh --logs

  # 清理所有数据
  ./scripts/deploy.sh --clean
USAGE
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --up)
        ACTION="up"
        shift
        ;;
      --down)
        ACTION="down"
        shift
        ;;
      --restart)
        ACTION="restart"
        shift
        ;;
      --logs)
        ACTION="logs"
        shift
        ;;
      --status)
        ACTION="status"
        shift
        ;;
      --clean)
        ACTION="clean"
        shift
        ;;
      --no-build)
        BUILD_FLAG=""
        shift
        ;;
      --build-only)
        ACTION="build"
        shift
        ;;
      --pull)
        ACTION="pull"
        shift
        ;;
      --with-https-proxy)
        ENABLE_HTTPS_PROXY="true"
        shift
        ;;
      --https-file)
        HTTPS_COMPOSE_FILE="${2:-}"
        shift 2
        ;;
      -f|--file)
        COMPOSE_FILE="${2:-}"
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

build_compose_args() {
  COMPOSE_FILES=("$COMPOSE_FILE")
  if [[ "${ENABLE_HTTPS_PROXY,,}" == "true" ]]; then
    if [[ -f "$HTTPS_COMPOSE_FILE" ]]; then
      COMPOSE_FILES+=("$HTTPS_COMPOSE_FILE")
    else
      log_warn "已启用 HTTPS 代理，但未找到 compose 文件: $HTTPS_COMPOSE_FILE"
    fi
  fi
}

run_compose() {
  local args=(docker compose)
  local compose_file
  for compose_file in "${COMPOSE_FILES[@]}"; do
    args+=(-f "$compose_file")
  done
  "${args[@]}" "$@"
}

check_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    log_error "未检测到 docker，请先安装 Docker"
    exit 1
  fi

  if ! docker compose version >/dev/null 2>&1; then
    log_error "未检测到 docker compose，请安装 Docker Compose 插件"
    exit 1
  fi
}

check_env_file() {
  if [[ ! -f "$ENV_FILE" ]]; then
    log_warn "未找到 .env 文件，创建默认配置..."
    cat > "$ENV_FILE" <<'EOF'
# PostgreSQL 配置
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=nodepass_license
POSTGRES_PORT=5432

# 应用配置
APP_BIND=0.0.0.0
APP_PORT=8090
BUILD_VERSION=main
GIN_MODE=release
IMAGE_NAME=ghcr.io/nodeox/license-center
ENABLE_HTTPS_PROXY=false
CADDY_DOMAIN=
CADDY_EMAIL=
CADDY_HTTP_PORT=80
CADDY_HTTPS_PORT=443
EOF
    log_info "已创建 $ENV_FILE，请根据需要修改配置"
  fi
}

check_config() {
  if [[ ! -f "configs/config.yaml" ]]; then
    log_error "未找到配置文件: configs/config.yaml"
    exit 1
  fi
}

build_images() {
  log_step "构建 Docker 镜像..."

  # 检查前端是否已构建
  if [[ ! -d "web-ui/dist" ]]; then
    log_info "前端未构建，将在 Docker 构建过程中自动构建"
  fi

  run_compose build --no-cache
  log_info "✓ 镜像构建完成"
}

pull_images() {
  log_step "拉取基础镜像..."
  run_compose pull
  log_info "✓ 镜像拉取完成"
}

start_services() {
  log_step "启动服务..."

  if [[ -n "$BUILD_FLAG" ]]; then
    run_compose up -d --build
  else
    run_compose up -d
  fi

  log_info "✓ 服务启动完成"

  # 等待服务健康检查
  log_step "等待服务就绪..."

  # 从 .env 文件读取端口配置
  local app_port=8090
  if [[ -f "$ENV_FILE" ]]; then
    app_port=$(grep "^APP_PORT=" "$ENV_FILE" | cut -d'=' -f2 || echo "8090")
  fi

  local max_wait=60
  local waited=0

  while [[ $waited -lt $max_wait ]]; do
    if curl -sf "http://127.0.0.1:${app_port}/health" >/dev/null 2>&1; then
      log_info "✓ 服务健康检查通过"
      show_service_info
      return 0
    fi
    sleep 2
    waited=$((waited + 2))
  done

  log_warn "服务健康检查超时，请手动检查日志"
  run_compose logs --tail=50
}

stop_services() {
  log_step "停止服务..."
  run_compose down
  log_info "✓ 服务已停止"
}

restart_services() {
  log_step "重启服务..."
  run_compose restart
  log_info "✓ 服务已重启"
  show_service_info
}

show_logs() {
  run_compose logs -f --tail=100
}

show_status() {
  log_step "服务状态:"
  run_compose ps

  echo ""
  log_step "容器健康状态:"
  run_compose ps --format json | \
    jq -r '.[] | "\(.Name): \(.State) - \(.Health)"' 2>/dev/null || \
    run_compose ps
}

clean_all() {
  log_warn "⚠️  此操作将删除所有容器、卷和数据，无法恢复！"
  read -p "确认清理所有数据？(yes/no): " confirm

  if [[ "$confirm" != "yes" ]]; then
    log_info "已取消清理操作"
    exit 0
  fi

  log_step "清理所有数据..."
  run_compose down -v --remove-orphans

  # 清理构建缓存
  if [[ -d "web-ui/dist" ]]; then
    rm -rf web-ui/dist
    log_info "✓ 已清理前端构建产物"
  fi

  log_info "✓ 清理完成"
}

show_service_info() {
  # 从 .env 文件读取端口配置
  local app_port=8090
  if [[ -f "$ENV_FILE" ]]; then
    app_port=$(grep "^APP_PORT=" "$ENV_FILE" | cut -d'=' -f2 || echo "8090")
  fi

  cat <<EOF

${GREEN}╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║  🎉 License Center 部署成功！                                  ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝${NC}

${BLUE}📍 访问地址:${NC}
  • 健康检查: http://127.0.0.1:${app_port}/health
  • 管理面板: http://127.0.0.1:${app_port}/console
  • API 文档:  http://127.0.0.1:${app_port}/api/v1

${BLUE}🔧 常用命令:${NC}
  • 查看日志: ./scripts/deploy.sh --logs
  • 查看状态: ./scripts/deploy.sh --status
  • 重启服务: ./scripts/deploy.sh --restart
  • 停止服务: ./scripts/deploy.sh --down

${BLUE}📊 容器状态:${NC}
$(run_compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "  运行中")

EOF
}

main() {
  parse_args "$@"
  build_compose_args

  check_docker
  check_env_file

  case "$ACTION" in
    up)
      check_config
      start_services
      ;;
    down)
      stop_services
      ;;
    restart)
      restart_services
      ;;
    logs)
      show_logs
      ;;
    status)
      show_status
      ;;
    build)
      build_images
      ;;
    pull)
      pull_images
      ;;
    clean)
      clean_all
      ;;
    *)
      log_error "未知操作: $ACTION"
      usage
      exit 1
      ;;
  esac
}

main "$@"
