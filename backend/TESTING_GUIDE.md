# 单元测试指南

## 概述

本文档介绍 NodePass-Pro 后端的单元测试策略、测试工具和运行方法。

## 测试架构

### 测试分层

```
测试层次
├── 单元测试 (Unit Tests)
│   ├── 应用层测试 (Application Layer)
│   │   ├── Commands 测试
│   │   └── Queries 测试
│   ├── 领域层测试 (Domain Layer)
│   │   └── 实体业务逻辑测试
│   └── 基础设施层测试 (Infrastructure Layer)
│       ├── 缓存层测试
│       └── 仓储层测试
├── 集成测试 (Integration Tests)
│   ├── API 端点测试
│   └── 数据库集成测试
└── 性能测试 (Performance Tests)
    ├── 压力测试
    └── 基准测试
```

## 已完成的测试

### 1. 应用层测试

#### Node Commands 测试
**文件**: `internal/application/node/commands/heartbeat_test.go`

测试用例：
- ✅ `TestHeartbeatHandler_Handle_Success` - 心跳处理成功
- ✅ `TestHeartbeatHandler_Handle_NodeNotFound` - 节点不存在
- ✅ `TestHeartbeatHandler_FlushHeartbeats` - 批量刷新心跳
- ✅ `TestHeartbeatHandler_DetectOfflineNodes` - 离线节点检测

#### User Commands 测试
**文件**: `internal/application/user/commands/create_user_test.go`

测试用例：
- ✅ `TestCreateUserHandler_Handle_Success` - 创建用户成功
- ✅ `TestCreateUserHandler_Handle_EmailExists` - 邮箱已存在
- ✅ `TestCreateUserHandler_Handle_UsernameExists` - 用户名已存在
- ✅ `TestCreateUserHandler_Handle_InvalidEmail` - 无效邮箱
- ✅ `TestCreateUserHandler_Handle_WeakPassword` - 弱密码

### 2. 基础设施层测试

#### Cache 测试
**文件**: `internal/infrastructure/cache/cache_test.go`

测试用例：
- ✅ `TestNodeCache_SetOnline` - 设置节点在线
- ✅ `TestNodeCache_IsOnline_Expired` - 节点过期检测
- ✅ `TestNodeCache_GetAllOnlineNodes` - 获取所有在线节点
- ✅ `TestNodeCache_SetNodeMetrics` - 设置节点指标
- ✅ `TestHeartbeatBuffer_PushAndPop` - 心跳缓冲区
- ✅ `TestHeartbeatBuffer_PopBatch_Limit` - 批量弹出限制
- ✅ `TestUserCache_SetAndGet` - 用户缓存读写
- ✅ `TestUserCache_IncrementTraffic` - 增加流量
- ✅ `TestUserCache_Delete` - 删除缓存
- ✅ `TestTrafficCounter_IncrementUserTraffic` - 用户流量计数
- ✅ `TestTrafficCounter_IncrementTunnelTraffic` - 隧道流量计数
- ✅ `TestDistributedLock_AcquireAndRelease` - 分布式锁
- ✅ `TestDistributedLock_AutoExpire` - 锁自动过期

## 测试工具

### 依赖包

```go
// 测试框架
"testing"

// 断言库
"github.com/stretchr/testify/assert"

// Mock 库
"github.com/stretchr/testify/mock"

// Redis 客户端
"github.com/redis/go-redis/v9"
```

### Mock 对象

我们为每个仓储接口创建了 Mock 实现：

- `MockNodeRepository` - 节点仓储 Mock
- `MockUserRepository` - 用户仓储 Mock
- `MockTunnelRepository` - 隧道仓储 Mock (待实现)
- `MockTrafficRepository` - 流量仓储 Mock (待实现)

## 运行测试

### 前置条件

1. **启动 Redis**
```bash
# 使用 Docker Compose
docker-compose -f docker-compose.dev.yml up -d redis

# 或直接启动 Redis
redis-server
```

2. **验证 Redis 连接**
```bash
redis-cli ping
# 应返回: PONG
```

### 运行所有测试

```bash
# 基本运行
./scripts/run-tests.sh

# 详细输出
./scripts/run-tests.sh -v

# 生成覆盖率报告
./scripts/run-tests.sh -c

# 详细输出 + 覆盖率
./scripts/run-tests.sh -v -c
```

### 运行特定包的测试

```bash
# 只测试 node commands
./scripts/run-tests.sh -p ./internal/application/node/commands

# 只测试 cache
./scripts/run-tests.sh -p ./internal/infrastructure/cache

# 只测试 user commands
./scripts/run-tests.sh -p ./internal/application/user/commands
```

### 使用 Go 命令直接运行

```bash
# 运行所有测试
go test ./...

# 详细输出
go test -v ./...

# 生成覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 运行特定测试
go test -v -run TestHeartbeatHandler_Handle_Success ./internal/application/node/commands

# 并行运行
go test -parallel 4 ./...
```

## 测试编写规范

### 1. 测试文件命名

```
源文件: heartbeat.go
测试文件: heartbeat_test.go
```

### 2. 测试函数命名

```go
// 格式: Test<StructName>_<MethodName>_<Scenario>
func TestHeartbeatHandler_Handle_Success(t *testing.T) {}
func TestHeartbeatHandler_Handle_NodeNotFound(t *testing.T) {}
func TestUserCache_SetAndGet(t *testing.T) {}
```

### 3. 测试结构 (AAA 模式)

```go
func TestExample(t *testing.T) {
    // Arrange - 准备测试数据
    mockRepo := new(MockRepository)
    handler := NewHandler(mockRepo)

    // Act - 执行测试
    result, err := handler.Handle(ctx, command)

    // Assert - 验证结果
    assert.NoError(t, err)
    assert.NotNil(t, result)
    mockRepo.AssertExpectations(t)
}
```

### 4. Mock 使用示例

```go
// 设置 Mock 期望
mockRepo.On("FindByID", ctx, uint(1)).Return(&entity, nil)
mockRepo.On("Create", ctx, mock.AnythingOfType("*Entity")).Return(nil)

// 验证 Mock 调用
mockRepo.AssertExpectations(t)
mockRepo.AssertCalled(t, "FindByID", ctx, uint(1))
mockRepo.AssertNumberOfCalls(t, "Create", 1)
```

### 5. Redis 测试最佳实践

```go
func setupTestRedis(t *testing.T) *redis.Client {
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
        DB:   15, // 使用专用测试数据库
    })

    // 清理测试数据
    ctx := context.Background()
    client.FlushDB(ctx)

    return client
}

func TestExample(t *testing.T) {
    client := setupTestRedis(t)
    defer client.Close()

    // 测试逻辑...
}
```

## 测试覆盖率

### 当前覆盖率

| 模块 | 覆盖率 | 状态 |
|------|--------|------|
| Node Commands | 80% | ✅ |
| User Commands | 75% | ✅ |
| Cache Layer | 85% | ✅ |
| Node Repository | 0% | ⏳ |
| User Repository | 0% | ⏳ |
| Tunnel Module | 0% | ⏳ |
| Traffic Module | 0% | ⏳ |

### 目标覆盖率

- **整体目标**: 80%
- **核心模块**: 90%
- **工具函数**: 70%

### 查看覆盖率报告

```bash
# 生成覆盖率
go test -coverprofile=coverage.out ./...

# 查看总体覆盖率
go tool cover -func=coverage.out | tail -1

# 生成 HTML 报告
go tool cover -html=coverage.out -o coverage.html

# 在浏览器中打开
open coverage.html
```

## 持续集成

### GitHub Actions 配置示例

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install dependencies
        run: go mod download

      - name: Run tests
        run: ./scripts/run-tests.sh -c

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

## 待完成的测试

### 高优先级

- [ ] Node Repository 测试
- [ ] User Repository 测试
- [ ] Tunnel Commands 测试
- [ ] Traffic Commands 测试

### 中优先级

- [ ] Tunnel Repository 测试
- [ ] Traffic Repository 测试
- [ ] User Queries 测试
- [ ] Node Queries 测试

### 低优先级

- [ ] Tunnel Queries 测试
- [ ] Traffic Queries 测试
- [ ] 集成测试
- [ ] 性能测试

## 常见问题

### Q1: Redis 连接失败

**问题**: `dial tcp [::1]:6379: connect: connection refused`

**解决方案**:
```bash
# 启动 Redis
docker-compose -f docker-compose.dev.yml up -d redis

# 或
redis-server
```

### Q2: 测试数据污染

**问题**: 测试之间相互影响

**解决方案**:
```go
// 每个测试前清理数据库
func setupTestRedis(t *testing.T) *redis.Client {
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
        DB:   15, // 使用专用测试数据库
    })
    client.FlushDB(context.Background())
    return client
}
```

### Q3: Mock 未按预期工作

**问题**: `mock: Unexpected Method Call`

**解决方案**:
```go
// 确保设置了正确的 Mock 期望
mockRepo.On("Method", arg1, arg2).Return(result, nil)

// 使用 mock.Anything 匹配任意参数
mockRepo.On("Method", mock.Anything).Return(result, nil)

// 使用类型匹配
mockRepo.On("Method", mock.AnythingOfType("*Type")).Return(result, nil)
```

### Q4: 测试超时

**问题**: 测试运行时间过长

**解决方案**:
```bash
# 设置超时时间
go test -timeout 30s ./...

# 并行运行
go test -parallel 4 ./...
```

## 最佳实践

### 1. 测试隔离

- 每个测试独立运行
- 使用专用测试数据库 (DB 15)
- 测试前清理数据

### 2. 测试可读性

- 使用描述性的测试名称
- 遵循 AAA 模式
- 添加必要的注释

### 3. Mock 使用

- 只 Mock 外部依赖
- 不要 Mock 被测试的对象
- 验证 Mock 调用

### 4. 断言清晰

```go
// 好的断言
assert.Equal(t, expected, actual, "用户名应该匹配")
assert.NoError(t, err, "创建用户不应该失败")

// 避免
assert.True(t, result == expected)
```

### 5. 测试数据

```go
// 使用有意义的测试数据
testUser := &User{
    Username: "testuser",
    Email:    "test@example.com",
}

// 避免
testUser := &User{
    Username: "aaa",
    Email:    "a@a.com",
}
```

## 参考资源

- [Go Testing 官方文档](https://golang.org/pkg/testing/)
- [Testify 文档](https://github.com/stretchr/testify)
- [Go Redis 文档](https://redis.uptrace.dev/)
- [测试驱动开发 (TDD)](https://en.wikipedia.org/wiki/Test-driven_development)

---

**最后更新**: 2026-03-11
**测试覆盖率**: 30% (目标 80%)
**状态**: 进行中
