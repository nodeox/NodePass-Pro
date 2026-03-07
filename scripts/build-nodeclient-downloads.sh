#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"

NODECLIENT_DIR="${ROOT_DIR}/nodeclient"
OUTPUT_DIR="${ROOT_DIR}/deploy/nodeclient/downloads"
GO_IMAGE="${GO_IMAGE:-golang:1.21-bookworm}"

log_info() {
  echo "[INFO] $*"
}

log_error() {
  echo "[ERROR] $*" >&2
}

checksum_file() {
  local target_file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    (
      cd "$OUTPUT_DIR"
      sha256sum "$(basename "$target_file")" >"$(basename "$target_file").sha256"
    )
    return
  fi

  if command -v shasum >/dev/null 2>&1; then
    (
      cd "$OUTPUT_DIR"
      shasum -a 256 "$(basename "$target_file")" >"$(basename "$target_file").sha256"
    )
    return
  fi

  log_error "未找到 sha256sum/shasum，无法生成校验文件。"
  exit 1
}

build_with_local_go() {
  local goos="$1"
  local goarch="$2"
  local output_file="$3"

  (
    cd "$NODECLIENT_DIR"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
      go build -trimpath -ldflags="-s -w" -o "$output_file" ./cmd/client
  )
}

build_with_docker_go() {
  local goos="$1"
  local goarch="$2"
  local output_file="$3"

  docker run --rm \
    -v "${ROOT_DIR}:/workspace" \
    -w /workspace/nodeclient \
    "$GO_IMAGE" \
    /bin/sh -lc "if [ -x /usr/local/go/bin/go ]; then GO_BIN=/usr/local/go/bin/go; elif command -v go >/dev/null 2>&1; then GO_BIN=\$(command -v go); else echo '[ERROR] 容器内未找到 go 可执行文件' >&2; exit 1; fi; \"\${GO_BIN}\" version >/dev/null && \"\${GO_BIN}\" mod download && CGO_ENABLED=0 GOOS=${goos} GOARCH=${goarch} \"\${GO_BIN}\" build -trimpath -ldflags='-s -w' -o ${output_file} ./cmd/client"
}

build_target() {
  local goos="$1"
  local goarch="$2"
  local output_file="${OUTPUT_DIR}/nodeclient-${goos}-${goarch}"

  log_info "构建 nodeclient: ${goos}/${goarch}"
  if command -v go >/dev/null 2>&1; then
    build_with_local_go "$goos" "$goarch" "$output_file"
  else
    log_info "本机未检测到 Go，使用容器镜像构建: ${GO_IMAGE}"
    build_with_docker_go "$goos" "$goarch" "/workspace/deploy/nodeclient/downloads/nodeclient-${goos}-${goarch}"
  fi

  chmod +x "$output_file"
  checksum_file "$output_file"
}

main() {
  if [[ ! -f "${NODECLIENT_DIR}/go.mod" ]]; then
    log_error "未找到 nodeclient 模块目录: ${NODECLIENT_DIR}"
    exit 1
  fi

  mkdir -p "$OUTPUT_DIR"

  build_target "linux" "amd64"
  build_target "linux" "arm64"

  log_info "构建完成，输出目录: ${OUTPUT_DIR}"
  ls -lh "$OUTPUT_DIR" | sed 's/^/[INFO] /'
}

main "$@"
