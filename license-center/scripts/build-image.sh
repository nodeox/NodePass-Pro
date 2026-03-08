#!/usr/bin/env bash
set -euo pipefail

# ============================================================================
# License Center 镜像构建脚本 v0.3.0
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
VERSION="${VERSION:-0.3.0}"
IMAGE_NAME="${IMAGE_NAME:-nodepass/license-center}"
REGISTRY="${REGISTRY:-}"
PLATFORM="${PLATFORM:-linux/amd64}"
OUTPUT_DIR="${OUTPUT_DIR:-./dist}"
ACTION="build"

log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_step() { echo -e "${BLUE}[STEP]${NC} $*"; }

usage() {
  cat <<'USAGE'
License Center 镜像构建脚本 v0.3.0

用法:
  ./scripts/build-image.sh [选项]

操作:
  --build           构建镜像（默认）
  --save            构建并保存为 tar 文件
  --load            从 tar 文件加载镜像
  --push            推送镜像到仓库
  --multi-arch      构建多架构镜像

选项:
  --version <ver>   镜像版本（默认: 0.3.0）
  --image <name>    镜像名称（默认: nodepass/license-center）
  --registry <url>  镜像仓库地址
  --platform <arch> 目标平台（默认: linux/amd64）
  --output <dir>    输出目录（默认: ./dist）
  --no-cache        不使用缓存构建
  -h, --help        显示帮助

示例:
  # 构建镜像
  ./scripts/build-image.sh --build

  # 构建并保存为文件
  ./scripts/build-image.sh --save

  # 从文件加载镜像
  ./scripts/build-image.sh --load

  # 推送到仓库
  ./scripts/build-image.sh --push --registry registry.example.com

  # 构建多架构镜像
  ./scripts/build-image.sh --multi-arch --platform linux/amd64,linux/arm64
USAGE
}

parse_args() {
  local no_cache=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --build)
        ACTION="build"
        shift
        ;;
      --save)
        ACTION="save"
        shift
        ;;
      --load)
        ACTION="load"
        shift
        ;;
      --push)
        ACTION="push"
        shift
        ;;
      --multi-arch)
        ACTION="multi-arch"
        shift
        ;;
      --version)
        VERSION="${2:-}"
        shift 2
        ;;
      --image)
        IMAGE_NAME="${2:-}"
        shift 2
        ;;
      --registry)
        REGISTRY="${2:-}"
        shift 2
        ;;
      --platform)
        PLATFORM="${2:-}"
        shift 2
        ;;
      --output)
        OUTPUT_DIR="${2:-}"
        shift 2
        ;;
      --no-cache)
        no_cache="--no-cache"
        shift
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

  BUILD_ARGS="$no_cache"
}

check_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    log_error "未检测到 docker，请先安装 Docker"
    exit 1
  fi
}

get_full_image_name() {
  if [[ -n "$REGISTRY" ]]; then
    echo "${REGISTRY}/${IMAGE_NAME}:${VERSION}"
  else
    echo "${IMAGE_NAME}:${VERSION}"
  fi
}

build_image() {
  local full_image=$(get_full_image_name)

  log_step "构建 Docker 镜像..."
  log_info "镜像名称: $full_image"
  log_info "目标平台: $PLATFORM"

  docker build \
    --platform "$PLATFORM" \
    --tag "$full_image" \
    --tag "${IMAGE_NAME}:latest" \
    --build-arg BUILD_VERSION="$VERSION" \
    $BUILD_ARGS \
    .

  log_info "✓ 镜像构建完成"

  # 显示镜像信息
  log_step "镜像信息:"
  docker images "$IMAGE_NAME" | head -2
}

save_image() {
  build_image

  local full_image=$(get_full_image_name)
  mkdir -p "$OUTPUT_DIR"
  local output_file="${OUTPUT_DIR}/license-center-${VERSION}.tar"

  log_step "保存镜像到文件..."
  log_info "输出文件: $output_file"

  docker save "$full_image" -o "$output_file"

  # 压缩镜像
  log_info "压缩镜像文件..."
  gzip -f "$output_file"

  local final_file="${output_file}.gz"
  local file_size=$(du -h "$final_file" | cut -f1)

  log_info "✓ 镜像已保存: $final_file"
  log_info "✓ 文件大小: $file_size"

  # 生成 SHA256 校验和
  log_info "生成校验和..."
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$final_file" > "${final_file}.sha256"
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$final_file" > "${final_file}.sha256"
  fi

  log_info "✓ 校验和文件: ${final_file}.sha256"
}

load_image() {
  local input_file="${OUTPUT_DIR}/license-center-${VERSION}.tar.gz"

  if [[ ! -f "$input_file" ]]; then
    log_error "镜像文件不存在: $input_file"
    exit 1
  fi

  log_step "从文件加载镜像..."
  log_info "输入文件: $input_file"

  # 验证校验和
  if [[ -f "${input_file}.sha256" ]]; then
    log_info "验证校验和..."
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum -c "${input_file}.sha256"
    elif command -v shasum >/dev/null 2>&1; then
      shasum -a 256 -c "${input_file}.sha256"
    fi
  fi

  # 解压并加载
  gunzip -c "$input_file" | docker load

  log_info "✓ 镜像加载完成"

  # 显示镜像信息
  log_step "镜像信息:"
  docker images "$IMAGE_NAME" | head -2
}

push_image() {
  if [[ -z "$REGISTRY" ]]; then
    log_error "请使用 --registry 指定镜像仓库地址"
    exit 1
  fi

  build_image

  local full_image=$(get_full_image_name)

  log_step "推送镜像到仓库..."
  log_info "目标仓库: $full_image"

  docker push "$full_image"

  # 同时推送 latest 标签
  if [[ -n "$REGISTRY" ]]; then
    local latest_image="${REGISTRY}/${IMAGE_NAME}:latest"
    docker tag "$full_image" "$latest_image"
    docker push "$latest_image"
  fi

  log_info "✓ 镜像推送完成"
}

build_multi_arch() {
  if [[ -z "$REGISTRY" ]]; then
    log_error "多架构构建需要推送到仓库，请使用 --registry 指定仓库地址"
    exit 1
  fi

  local full_image=$(get_full_image_name)

  log_step "构建多架构镜像..."
  log_info "镜像名称: $full_image"
  log_info "目���平台: $PLATFORM"

  # 创建并使用 buildx builder
  if ! docker buildx inspect multiarch-builder >/dev/null 2>&1; then
    log_info "创建 buildx builder..."
    docker buildx create --name multiarch-builder --use
  else
    docker buildx use multiarch-builder
  fi

  # 构建并推送多架构镜像
  docker buildx build \
    --platform "$PLATFORM" \
    --tag "$full_image" \
    --tag "${REGISTRY}/${IMAGE_NAME}:latest" \
    --build-arg BUILD_VERSION="$VERSION" \
    --push \
    $BUILD_ARGS \
    .

  log_info "✓ 多架构镜像构建并推送完成"
}

show_summary() {
  cat <<EOF

${GREEN}╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║  ✅ 镜像操作完成！                                             ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝${NC}

${BLUE}📦 镜像信息:${NC}
  • 镜像名称: $(get_full_image_name)
  • 版本号: ${VERSION}
  • 平台: ${PLATFORM}

${BLUE}🚀 使用方式:${NC}
  # 使用 docker-compose
  docker compose -f docker-compose.prod.yml up -d

  # 直接运行
  docker run -d -p 8090:8090 $(get_full_image_name)

${BLUE}📚 相关文件:${NC}
  • 镜像文件: ${OUTPUT_DIR}/license-center-${VERSION}.tar.gz
  • 校验和: ${OUTPUT_DIR}/license-center-${VERSION}.tar.gz.sha256

EOF
}

main() {
  parse_args "$@"
  check_docker

  case "$ACTION" in
    build)
      build_image
      ;;
    save)
      save_image
      ;;
    load)
      load_image
      ;;
    push)
      push_image
      ;;
    multi-arch)
      build_multi_arch
      ;;
    *)
      log_error "未知操作: $ACTION"
      usage
      exit 1
      ;;
  esac

  show_summary
}

main "$@"
