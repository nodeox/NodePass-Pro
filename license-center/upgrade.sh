#!/bin/bash

# NodePass License Center 一键升级脚本
# 用法: ./upgrade.sh [version]

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

log_success() {
    echo -e "${CYAN}[SUCCESS]${NC} $1"
}

# 配置
CONTAINER_NAME="license-center"
IMAGE_NAME="ghcr.io/nodeox/license-center"
VERSION=${1:-"latest"}
BACKUP_DIR="./backups"
DATA_DIR="./data"
CONFIG_DIR="./configs"

# 显示横幅
show_banner() {
    echo ""
    echo -e "${CYAN}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║                                                ║${NC}"
    echo -e "${CYAN}║   NodePass License Center 一键升级脚本         ║${NC}"
    echo -e "${CYAN}║                                                ║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════╝${NC}"
    echo ""
}

# 检查 Docker
check_docker() {
    log_step "检查 Docker 环境..."

    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        echo "请先安装 Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        log_error "Docker 守护进程未运行"
        exit 1
    fi

    log_info "Docker 环境检查通过 ✓"
}

# 检查当前版本
check_current_version() {
    log_step "检查当前版本..."

    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        CURRENT_VERSION=$(docker inspect ${CONTAINER_NAME} --format='{{.Config.Image}}' 2>/dev/null || echo "unknown")
        log_info "当前版本: ${CURRENT_VERSION}"

        # 检查容器状态
        CONTAINER_STATUS=$(docker inspect ${CONTAINER_NAME} --format='{{.State.Status}}' 2>/dev/null || echo "unknown")
        log_info "容器状态: ${CONTAINER_STATUS}"
    else
        log_warn "未找到现有容器，将执行全新安装"
        CURRENT_VERSION="none"
    fi
}

# 备份数据
backup_data() {
    log_step "备份数据..."

    # 创建备份目录
    BACKUP_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BACKUP_PATH="${BACKUP_DIR}/backup_${BACKUP_TIMESTAMP}"
    mkdir -p "${BACKUP_PATH}"

    # 备份数据库
    if [ -d "${DATA_DIR}" ]; then
        log_info "备份数据库..."
        cp -r "${DATA_DIR}" "${BACKUP_PATH}/data"
        log_info "数据库备份完成: ${BACKUP_PATH}/data"
    fi

    # 备份配置文件
    if [ -d "${CONFIG_DIR}" ]; then
        log_info "备份配置文件..."
        cp -r "${CONFIG_DIR}" "${BACKUP_PATH}/configs"
        log_info "配置文件备份完成: ${BACKUP_PATH}/configs"
    fi

    # 备份容器配置
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log_info "导出容器配置..."
        docker inspect ${CONTAINER_NAME} > "${BACKUP_PATH}/container_config.json"
    fi

    log_success "备份完成: ${BACKUP_PATH}"
    echo "${BACKUP_PATH}" > .last_backup
}

# 停止旧容器
stop_old_container() {
    log_step "停止旧容器..."

    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log_info "停止容器: ${CONTAINER_NAME}"
        docker stop ${CONTAINER_NAME}
        log_info "容器已停止 ✓"
    else
        log_info "容器未运行，跳过停止步骤"
    fi
}

# 删除旧容器
remove_old_container() {
    log_step "删除旧容器..."

    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log_info "删除容器: ${CONTAINER_NAME}"
        docker rm ${CONTAINER_NAME}
        log_info "容器已删除 ✓"
    else
        log_info "容器不存在，跳过删除步骤"
    fi
}

# 拉取新镜像
pull_new_image() {
    log_step "拉取新镜像..."

    NEW_IMAGE="${IMAGE_NAME}:${VERSION}"
    log_info "镜像: ${NEW_IMAGE}"

    # 拉取镜像
    if docker pull ${NEW_IMAGE}; then
        log_success "镜像拉取成功 ✓"
    else
        log_error "镜像拉取失败"
        log_warn "尝试从备份恢复..."
        rollback
        exit 1
    fi

    # 显示镜像信息
    log_info "镜像信息:"
    docker images ${IMAGE_NAME} --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}" | head -2
}

# 启动新容器
start_new_container() {
    log_step "启动新容器..."

    # 确保目录存在
    mkdir -p ${DATA_DIR}
    mkdir -p ${CONFIG_DIR}

    # 检查环境变量
    if [ -z "$JWT_SECRET" ]; then
        log_warn "JWT_SECRET 未设置，使用默认值（不推荐用于生产环境）"
        JWT_SECRET="default-jwt-secret-please-change-in-production"
    fi

    if [ -z "$ADMIN_PASSWORD" ]; then
        log_warn "ADMIN_PASSWORD 未设置，使用默认值（不推荐用于生产环境）"
        ADMIN_PASSWORD="ChangeMe123!"
    fi

    # 启动容器
    log_info "启动容器: ${CONTAINER_NAME}"
    docker run -d \
        --name ${CONTAINER_NAME} \
        -p 8090:8090 \
        -v $(pwd)/${DATA_DIR}:/app/data \
        -v $(pwd)/${CONFIG_DIR}:/app/configs \
        -e JWT_SECRET="${JWT_SECRET}" \
        -e ADMIN_PASSWORD="${ADMIN_PASSWORD}" \
        --restart unless-stopped \
        ${IMAGE_NAME}:${VERSION}

    log_success "容器已启动 ✓"
}

# 健康检查
health_check() {
    log_step "执行健康检查..."

    local max_attempts=30
    local attempt=0

    log_info "等待服务启动..."

    while [ $attempt -lt $max_attempts ]; do
        if curl -f http://localhost:8090/health > /dev/null 2>&1; then
            log_success "健康检查通过 ✓"

            # 获取版本信息
            log_info "服务版本信息:"
            curl -s http://localhost:8090/ | python3 -m json.tool 2>/dev/null || curl -s http://localhost:8090/
            return 0
        fi

        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done

    echo ""
    log_error "健康检查失败"
    log_warn "查看容器日志:"
    docker logs --tail 50 ${CONTAINER_NAME}

    log_warn "尝试从备份恢复..."
    rollback
    exit 1
}

# 回滚
rollback() {
    log_error "升级失败，开始回滚..."

    # 停止并删除新容器
    docker stop ${CONTAINER_NAME} 2>/dev/null || true
    docker rm ${CONTAINER_NAME} 2>/dev/null || true

    # 检查是否有备份
    if [ ! -f .last_backup ]; then
        log_error "未找到备份信息，无法回滚"
        exit 1
    fi

    LAST_BACKUP=$(cat .last_backup)

    if [ ! -d "${LAST_BACKUP}" ]; then
        log_error "备份目录不存在: ${LAST_BACKUP}"
        exit 1
    fi

    log_info "从备份恢复: ${LAST_BACKUP}"

    # 恢复数据
    if [ -d "${LAST_BACKUP}/data" ]; then
        rm -rf ${DATA_DIR}
        cp -r "${LAST_BACKUP}/data" ${DATA_DIR}
        log_info "数据已恢复"
    fi

    # 恢复配置
    if [ -d "${LAST_BACKUP}/configs" ]; then
        rm -rf ${CONFIG_DIR}
        cp -r "${LAST_BACKUP}/configs" ${CONFIG_DIR}
        log_info "配置已恢复"
    fi

    # 启动旧版本容器
    if [ "${CURRENT_VERSION}" != "none" ] && [ "${CURRENT_VERSION}" != "unknown" ]; then
        log_info "启动旧版本容器: ${CURRENT_VERSION}"
        docker run -d \
            --name ${CONTAINER_NAME} \
            -p 8090:8090 \
            -v $(pwd)/${DATA_DIR}:/app/data \
            -v $(pwd)/${CONFIG_DIR}:/app/configs \
            -e JWT_SECRET="${JWT_SECRET}" \
            -e ADMIN_PASSWORD="${ADMIN_PASSWORD}" \
            --restart unless-stopped \
            ${CURRENT_VERSION}

        log_success "已回滚到旧版本"
    else
        log_warn "无法回滚，未找到旧版本信息"
    fi
}

# 清理旧镜像
cleanup_old_images() {
    log_step "清理旧镜像..."

    read -p "是否清理旧的 Docker 镜像? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "清理未使用的镜像..."
        docker image prune -f
        log_success "清理完成 ✓"
    else
        log_info "跳过清理"
    fi
}

# 显示升级信息
show_upgrade_info() {
    echo ""
    log_success "╔════════════════════════════════════════════════╗"
    log_success "║          升级完成！                             ║"
    log_success "╚════════════════════════════════════════════════╝"
    echo ""
    log_info "升级信息:"
    echo "  旧版本: ${CURRENT_VERSION}"
    echo "  新版本: ${IMAGE_NAME}:${VERSION}"
    echo "  备份位置: $(cat .last_backup 2>/dev/null || echo 'N/A')"
    echo ""
    log_info "访问地址:"
    echo "  http://localhost:8090"
    echo "  http://localhost:8090/console"
    echo ""
    log_info "查看日志:"
    echo "  docker logs -f ${CONTAINER_NAME}"
    echo ""
    log_info "查看状态:"
    echo "  docker ps | grep ${CONTAINER_NAME}"
    echo ""
}

# 主函数
main() {
    show_banner

    log_info "目标版本: ${VERSION}"
    echo ""

    # 确认升级
    read -p "是否继续升级? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_warn "升级已取消"
        exit 0
    fi

    # 执行升级步骤
    check_docker
    check_current_version
    backup_data
    stop_old_container
    remove_old_container
    pull_new_image
    start_new_container
    health_check
    cleanup_old_images
    show_upgrade_info

    log_success "升级流程完成！"
}

# 捕获错误
trap 'log_error "升级过程中发生错误"; rollback; exit 1' ERR

# 运行主函数
main "$@"
