#!/bin/bash

# 单元测试运行脚本

set -e

echo "=========================================="
echo "NodePass-Pro Backend 单元测试"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查 Redis 是否运行
echo "检查 Redis 连接..."
if ! redis-cli -h localhost -p 6379 ping > /dev/null 2>&1; then
    echo -e "${RED}错误: Redis 未运行，请先启动 Redis${NC}"
    echo "提示: docker-compose -f docker-compose.dev.yml up -d redis"
    exit 1
fi
echo -e "${GREEN}✓ Redis 连接正常${NC}"
echo ""

# 清理测试数据库
echo "清理测试数据库..."
redis-cli -h localhost -p 6379 -n 15 FLUSHDB > /dev/null
echo -e "${GREEN}✓ 测试数据库已清理${NC}"
echo ""

# 运行测试
echo "=========================================="
echo "运行单元测试"
echo "=========================================="
echo ""

# 设置测试环境变量
export GO_ENV=test
export REDIS_ADDR=localhost:6379
export REDIS_DB=15

# 测试选项
COVERAGE=false
VERBOSE=false
PACKAGE=""

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -p|--package)
            PACKAGE="$2"
            shift 2
            ;;
        *)
            echo "未知参数: $1"
            exit 1
            ;;
    esac
done

# 构建测试命令
TEST_CMD="go test"

if [ "$VERBOSE" = true ]; then
    TEST_CMD="$TEST_CMD -v"
fi

if [ "$COVERAGE" = true ]; then
    TEST_CMD="$TEST_CMD -coverprofile=coverage.out -covermode=atomic"
fi

if [ -n "$PACKAGE" ]; then
    TEST_CMD="$TEST_CMD $PACKAGE"
else
    TEST_CMD="$TEST_CMD ./..."
fi

# 运行测试
echo "执行命令: $TEST_CMD"
echo ""

if eval $TEST_CMD; then
    echo ""
    echo -e "${GREEN}=========================================="
    echo "✓ 所有测试通过"
    echo "==========================================${NC}"

    # 显示覆盖率
    if [ "$COVERAGE" = true ]; then
        echo ""
        echo "生成覆盖率报告..."
        go tool cover -func=coverage.out | tail -1

        # 生成 HTML 报告
        go tool cover -html=coverage.out -o coverage.html
        echo -e "${GREEN}✓ 覆盖率报告已生成: coverage.html${NC}"
    fi

    exit 0
else
    echo ""
    echo -e "${RED}=========================================="
    echo "✗ 测试失败"
    echo "==========================================${NC}"
    exit 1
fi
