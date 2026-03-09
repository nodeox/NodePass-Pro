#!/bin/bash

# 版本信息注入脚本
# 用于在编译时注入版本信息

set -e

# 获取版本信息
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
GO_VERSION=$(go version | awk '{print $3}')

# 构建标志
LDFLAGS="-X 'github.com/yourusername/nodepass/pkg/version.Version=${VERSION}' \
         -X 'github.com/yourusername/nodepass/pkg/version.GitCommit=${GIT_COMMIT}' \
         -X 'github.com/yourusername/nodepass/pkg/version.GitBranch=${GIT_BRANCH}' \
         -X 'github.com/yourusername/nodepass/pkg/version.BuildTime=${BUILD_TIME}'"

echo "Building with version info:"
echo "  Version:    ${VERSION}"
echo "  Git Commit: ${GIT_COMMIT}"
echo "  Git Branch: ${GIT_BRANCH}"
echo "  Build Time: ${BUILD_TIME}"
echo "  Go Version: ${GO_VERSION}"
echo ""

# 构建
go build -ldflags "${LDFLAGS}" "$@"
