#!/bin/bash

# NodePass 版本管理部署脚本
# 自动构建、部署并上报版本信息

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 获取版本信息
get_version_info() {
    export VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    export GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    export GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    export BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

    log_info "Version Information:"
    echo "  Version:    $VERSION"
    echo "  Git Commit: $GIT_COMMIT"
    echo "  Git Branch: $GIT_BRANCH"
    echo "  Build Time: $BUILD_TIME"
    echo ""
}

# 构建 Docker 镜像
build_images() {
    log_info "Building Docker images with version info..."

    docker-compose -f docker-compose.version.yml build \
        --build-arg VERSION="$VERSION" \
        --build-arg GIT_COMMIT="$GIT_COMMIT" \
        --build-arg GIT_BRANCH="$GIT_BRANCH" \
        --build-arg BUILD_TIME="$BUILD_TIME"

    log_info "Docker images built successfully"
}

# 启动服务
start_services() {
    log_info "Starting services..."

    docker-compose -f docker-compose.version.yml up -d

    log_info "Services started successfully"
}

# 等待服务就绪
wait_for_services() {
    log_info "Waiting for services to be ready..."

    local max_attempts=30
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if curl -s http://localhost:8090/health > /dev/null 2>&1; then
            log_info "License Center is ready"
            return 0
        fi

        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done

    log_error "Services failed to start within timeout"
    return 1
}

# 验证版本信息
verify_versions() {
    log_info "Verifying version information..."

    # 等待几秒让版本上报完成
    sleep 5

    # 获取系统版本信息
    local response=$(curl -s http://localhost:8090/api/v1/versions/system)

    if echo "$response" | grep -q "\"success\":true"; then
        log_info "Version information verified successfully"
        echo "$response" | python3 -m json.tool 2>/dev/null || echo "$response"
    else
        log_warn "Failed to verify version information"
        echo "$response"
    fi
}

# 显示服务状态
show_status() {
    log_info "Service Status:"
    docker-compose -f docker-compose.version.yml ps

    echo ""
    log_info "Access URLs:"
    echo "  License Center: http://localhost:8090"
    echo "  Backend API:    http://localhost:8080"
    echo "  Frontend:       http://localhost:3000"
    echo ""
    log_info "Version Management: http://localhost:3000/versions"
}

# 停止服务
stop_services() {
    log_info "Stopping services..."
    docker-compose -f docker-compose.version.yml down
    log_info "Services stopped"
}

# 清理
cleanup() {
    log_info "Cleaning up..."
    docker-compose -f docker-compose.version.yml down -v
    log_info "Cleanup complete"
}

# 显示日志
show_logs() {
    local service=$1
    if [ -z "$service" ]; then
        docker-compose -f docker-compose.version.yml logs -f
    else
        docker-compose -f docker-compose.version.yml logs -f "$service"
    fi
}

# 主函数
main() {
    local command=${1:-deploy}

    case $command in
        deploy)
            log_info "Starting deployment..."
            get_version_info
            build_images
            start_services
            wait_for_services
            verify_versions
            show_status
            log_info "Deployment complete!"
            ;;

        build)
            get_version_info
            build_images
            ;;

        start)
            start_services
            wait_for_services
            show_status
            ;;

        stop)
            stop_services
            ;;

        restart)
            stop_services
            sleep 2
            start_services
            wait_for_services
            show_status
            ;;

        status)
            show_status
            ;;

        verify)
            verify_versions
            ;;

        logs)
            show_logs "$2"
            ;;

        clean)
            cleanup
            ;;

        *)
            echo "Usage: $0 {deploy|build|start|stop|restart|status|verify|logs|clean}"
            echo ""
            echo "Commands:"
            echo "  deploy   - Build and deploy all services (default)"
            echo "  build    - Build Docker images with version info"
            echo "  start    - Start services"
            echo "  stop     - Stop services"
            echo "  restart  - Restart services"
            echo "  status   - Show service status"
            echo "  verify   - Verify version information"
            echo "  logs     - Show logs (optionally specify service name)"
            echo "  clean    - Stop and remove all containers and volumes"
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@"
