#!/bin/bash

# 数据库迁移脚本
# 用于执行节点分组功能的数据库迁移

set -e  # 遇到错误立即退出

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-nodepass_panel}"
MIGRATION_UP_FILES=("migrations/0001_create_node_groups.up.sql" "migrations/0002_update_tunnels.up.sql")
MIGRATION_DOWN_FILES=("migrations/0002_update_tunnels.down.sql" "migrations/0001_create_node_groups.down.sql")

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

# 检查 psql 是否安装
check_psql() {
    if ! command -v psql &> /dev/null; then
        log_error "psql 未安装，请先安装 PostgreSQL 客户端"
        exit 1
    fi
    log_info "psql 已安装"
}

# 检查数据库连接
check_db_connection() {
    log_info "检查数据库连接..."
    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1" > /dev/null 2>&1; then
        log_info "数据库连接成功"
        return 0
    else
        log_error "无法连接到数据库"
        log_error "请检查数据库配置: $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"
        exit 1
    fi
}

# 备份数据库
backup_database() {
    log_info "备份数据库..."
    BACKUP_FILE="backup_$(date +%Y%m%d_%H%M%S).sql"

    if PGPASSWORD=$DB_PASSWORD pg_dump -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME > "$BACKUP_FILE"; then
        log_info "数据库备份成功: $BACKUP_FILE"
        echo "$BACKUP_FILE"
    else
        log_error "数据库备份失败"
        exit 1
    fi
}

# 执行迁移
run_migration() {
    local migration_file=$1
    log_info "执行迁移: $migration_file"

    if [ ! -f "$migration_file" ]; then
        log_error "迁移文件不存在: $migration_file"
        exit 1
    fi

    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$migration_file"; then
        log_info "迁移执行成功"
        return 0
    else
        log_error "迁移执行失败"
        return 1
    fi
}

# 验证迁移
verify_migration() {
    log_info "验证迁移结果..."

    # 检查表是否创建
    local tables=("node_groups" "node_instances" "node_group_relations" "node_group_stats" "tunnels")

    for table in "${tables[@]}"; do
        if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\dt $table" | grep -q "$table"; then
            log_info "✓ 表 $table 已创建"
        else
            log_error "✗ 表 $table 未创建"
            return 1
        fi
    done

    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\df update_node_group_stats" | grep -q "update_node_group_stats"; then
        log_info "✓ 函数 update_node_group_stats 已创建"
    else
        log_error "✗ 函数 update_node_group_stats 未创建"
        return 1
    fi

    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1 FROM information_schema.columns WHERE table_name='tunnels' AND column_name='listen_host';" | grep -q "1"; then
        log_info "✓ tunnels.listen_host 字段已存在"
    else
        log_error "✗ tunnels.listen_host 字段不存在"
        return 1
    fi

    log_info "迁移验证通过"
    return 0
}

# 回滚迁移
rollback_migration() {
    local backup_file=$1
    log_warn "开始回滚迁移..."

    if [ -z "$backup_file" ] || [ ! -f "$backup_file" ]; then
        log_error "备份文件不存在，无法回滚"
        exit 1
    fi

    # 先执行 down 迁移
    log_info "执行 down 迁移..."
    for migration in "${MIGRATION_DOWN_FILES[@]}"; do
        run_migration "$migration"
    done

    # 恢复备份
    log_info "恢复数据库备份..."
    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME < "$backup_file"; then
        log_info "数据库恢复成功"
    else
        log_error "数据库恢复失败"
        exit 1
    fi
}

# 主函数
main() {
    local action=${1:-up}

    echo "========================================"
    echo "  NodePass Pro 数据库迁移工具"
    echo "========================================"
    echo ""

    # 检查环境
    check_psql
    check_db_connection

    case $action in
        up)
            log_info "开始执行 UP 迁移..."

            # 备份数据库
            BACKUP_FILE=$(backup_database)

            # 执行迁移
            migration_ok="true"
            for migration in "${MIGRATION_UP_FILES[@]}"; do
                if ! run_migration "$migration"; then
                    migration_ok="false"
                    break
                fi
            done
            if [[ "${migration_ok}" == "true" ]]; then
                # 验证迁移
                if verify_migration; then
                    log_info "✓ 迁移完成！"
                    log_info "备份文件: $BACKUP_FILE"
                    echo ""
                    log_info "如需回滚，请运行: $0 rollback $BACKUP_FILE"
                else
                    log_error "迁移验证失败"
                    log_warn "是否回滚？(y/n)"
                    read -r response
                    if [[ "$response" =~ ^[Yy]$ ]]; then
                        rollback_migration "$BACKUP_FILE"
                    fi
                    exit 1
                fi
            else
                log_error "迁移执行失败"
                log_warn "是否回滚？(y/n)"
                read -r response
                if [[ "$response" =~ ^[Yy]$ ]]; then
                    rollback_migration "$BACKUP_FILE"
                fi
                exit 1
            fi
            ;;

        down)
            log_warn "开始执行 DOWN 迁移（删除所有节点分组相关表）..."
            log_warn "此操作将删除所有节点分组数据，是否继续？(y/n)"
            read -r response
            if [[ ! "$response" =~ ^[Yy]$ ]]; then
                log_info "已取消"
                exit 0
            fi

            # 备份数据库
            BACKUP_FILE=$(backup_database)

            # 执行 down 迁移
            migration_ok="true"
            for migration in "${MIGRATION_DOWN_FILES[@]}"; do
                if ! run_migration "$migration"; then
                    migration_ok="false"
                    break
                fi
            done
            if [[ "${migration_ok}" == "true" ]]; then
                log_info "✓ DOWN 迁移完成"
                log_info "备份文件: $BACKUP_FILE"
            else
                log_error "DOWN 迁移失败"
                exit 1
            fi
            ;;

        rollback)
            local backup_file=$2
            if [ -z "$backup_file" ]; then
                log_error "请提供备份文件路径"
                echo "用法: $0 rollback <backup_file>"
                exit 1
            fi
            rollback_migration "$backup_file"
            ;;

        verify)
            verify_migration
            ;;

        *)
            echo "用法: $0 {up|down|rollback|verify}"
            echo ""
            echo "命令:"
            echo "  up       - 执行迁移（创建表）"
            echo "  down     - 回滚迁移（删除表）"
            echo "  rollback - 从备份恢复"
            echo "  verify   - 验证迁移结果"
            echo ""
            echo "环境变量:"
            echo "  DB_HOST     - 数据库主机 (默认: localhost)"
            echo "  DB_PORT     - 数据库端口 (默认: 5432)"
            echo "  DB_USER     - 数据库用户 (默认: postgres)"
            echo "  DB_PASSWORD - 数据库密码 (默认: postgres)"
            echo "  DB_NAME     - 数据库名称 (默认: nodepass_panel)"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
