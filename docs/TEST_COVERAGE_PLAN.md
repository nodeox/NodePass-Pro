# 测试覆盖率提升计划

## 📊 当前测试覆盖率状态

### 后端 (Backend)
```
模块                                    覆盖率    状态
─────────────────────────────────────────────────────
internal/cache                          0.0%     ❌ 无测试
internal/config                         0.0%     ❌ 无测试
internal/database                       0.0%     ❌ 无测试
internal/handlers                       0.0%     ❌ 无测试
internal/license                       41.5%     ⚠️  部分覆盖
internal/middleware                     3.8%     ❌ 覆盖不足
internal/models                         0.0%     ❌ 无测试
internal/services                       3.5%     ❌ 覆盖不足
internal/utils                         12.6%     ❌ 覆盖不足
internal/websocket                     21.0%     ⚠️  部分覆盖
─────────────────────────────────────────────────────
总体覆盖率                              ~8%      ❌ 严重不足
```

### 前端 (Frontend)
```
状态: ❌ 无测试文件
覆盖率: 0%
```

### 授权中心 (License Center)
```
测试文件: 1 个 (signature_redis_test.go)
覆盖率: ~15% (估算)
```

### 节点客户端 (NodeClient)
```
状态: ❌ 无测试文件
覆盖率: 0%
```

---

## 🎯 目标覆盖率

| 模块 | 当前 | 目标 | 优先级 |
|------|------|------|--------|
| **后端核心服务** | 3.5% | 75% | P0 |
| **后端中间件** | 3.8% | 80% | P0 |
| **后端处理器** | 0% | 60% | P1 |
| **前端工具函数** | 0% | 70% | P1 |
| **前端 API 客户端** | 0% | 80% | P0 |
| **授权中心** | 15% | 70% | P1 |
| **节点客户端** | 0% | 60% | P2 |

**总体目标: 70%+ 覆盖率**

---

## ✅ 已完成的测试

### 1. 中间件测试 (新增)
- ✅ `rate_limit_test.go` - 速率限制中间件测试
  - 测试用例: 11 个
  - 覆盖场景: 正常限流、突发限制、自定义 key、并发测试
  - 预计覆盖率提升: 3.8% → 65%

- ✅ `auth_test.go` - 认证中间件测试
  - 测试用例: 15 个
  - 覆盖场景: JWT 验证、WebSocket 认证、token 提取
  - 预计覆盖率提升: 3.8% → 70%

### 2. 服务层测试 (新增)
- ✅ `auth_service_test.go` - 认证服务测试
  - 测试用例: 30+ 个
  - 覆盖场景: 注册、登录、密码修改、用户查询
  - 预计覆盖率提升: 3.5% → 25%

---

## 📋 待完成的测试任务

### P0 - 高优先级 (核心功能)

#### 后端服务层测试

**1. traffic_service_test.go**
```go
// 测试用例
- TestTrafficService_GetQuota
- TestTrafficService_UpdateQuota
- TestTrafficService_RecordUsage
- TestTrafficService_MonthlyReset
- TestTrafficService_GetUsageSummary
- TestTrafficService_GetRecords
```

**2. node_group_service_test.go**
```go
// 测试用例
- TestNodeGroupService_Create
- TestNodeGroupService_Update
- TestNodeGroupService_Delete
- TestNodeGroupService_List
- TestNodeGroupService_GetStats
- TestNodeGroupService_CreateRelation
- TestNodeGroupService_MarkOfflineByHeartbeat
```

**3. vip_service_test.go**
```go
// 测试用例
- TestVIPService_GetMyLevel
- TestVIPService_UpgradeUser
- TestVIPService_CheckExpiration
- TestVIPService_ListLevels
- TestVIPService_CreateLevel
- TestVIPService_UpdateLevel
```

**4. benefit_code_service_test.go**
```go
// 测试用例
- TestBenefitCodeService_Generate
- TestBenefitCodeService_Redeem
- TestBenefitCodeService_List
- TestBenefitCodeService_BatchDelete
- TestBenefitCodeService_ValidateCode
```

#### 前端测试

**5. api.test.ts** (API 客户端)
```typescript
// 测试用例
- describe('Token 刷新机制')
  - test('自动刷新过期 token')
  - test('并发请求时只刷新一次')
  - test('刷新失败后重定向登录')

- describe('CSRF 令牌管理')
  - test('自动同步 CSRF token')
  - test('请求时自动添加 CSRF header')

- describe('错误处理')
  - test('401 错误处理')
  - test('403 错误处理')
  - test('网络错误处理')
```

**6. secureStorage.test.ts** (安全存储)
```typescript
// 测试用例
- test('存储和读取 token')
- test('清除存储')
- test('迁移旧数据')
- test('处理损坏的数据')
```

### P1 - 中优先级

#### 后端中间件测试

**7. license_test.go** (授权中间件)
```go
// 测试用例
- TestLicenseGuard_Enabled
- TestLicenseGuard_Disabled
- TestLicenseGuard_Expired
- TestLicenseGuard_SkipHealthCheck
```

**8. audit_test.go** (审计日志中间件)
```go
// 测试用例
- TestAuditLogger_RecordRequest
- TestAuditLogger_SkipReadOnlyRequests
- TestAuditLogger_RecordSensitiveOperations
```

**9. heartbeat_test.go** (心跳防重放)
```go
// 测试用例
- TestHeartbeatReplayProtection_ValidNonce
- TestHeartbeatReplayProtection_DuplicateNonce
- TestHeartbeatReplayProtection_ExpiredNonce
```

#### 后端处理器测试

**10. auth_handler_test.go**
```go
// 测试用例
- TestAuthHandler_Register
- TestAuthHandler_Login
- TestAuthHandler_LoginV2
- TestAuthHandler_RefreshTokenV2
- TestAuthHandler_Me
- TestAuthHandler_ChangePassword
```

**11. tunnel_handler_test.go**
```go
// 测试用例
- TestTunnelHandler_Create
- TestTunnelHandler_List
- TestTunnelHandler_Get
- TestTunnelHandler_Update
- TestTunnelHandler_Delete
- TestTunnelHandler_Start
- TestTunnelHandler_Stop
```

#### 前端组件测试

**12. Login.test.tsx**
```typescript
// 测试用例
- test('渲染登录表单')
- test('提交登录表单')
- test('显示验证错误')
- test('登录成功后跳转')
```

**13. useWebSocket.test.ts** (WebSocket Hook)
```typescript
// 测试用例
- test('建立 WebSocket 连接')
- test('接收消息')
- test('发送消息')
- test('自动重连')
- test('清理连接')
```

### P2 - 低优先级

**14. 授权中心测试**
- license_service_test.go
- domain_binding_service_test.go
- extension_service_test.go

**15. 节点客户端测试**
- agent_test.go
- heartbeat_test.go
- config_test.go

---

## 🛠️ 测试环境配置

### 1. Makefile (测试命令)

```makefile
# 后端测试
.PHONY: test
test:
	cd backend && go test -v -cover ./internal/...

.PHONY: test-coverage
test-coverage:
	cd backend && go test -v -coverprofile=coverage.out ./internal/...
	cd backend && go tool cover -html=coverage.out -o coverage.html

.PHONY: test-race
test-race:
	cd backend && go test -v -race ./internal/...

.PHONY: test-short
test-short:
	cd backend && go test -v -short ./internal/...

# 前端测试
.PHONY: test-frontend
test-frontend:
	cd frontend && npm test

.PHONY: test-frontend-coverage
test-frontend-coverage:
	cd frontend && npm test -- --coverage

# 集成测试
.PHONY: test-integration
test-integration:
	./tests/integration_test.sh

# 全部测试
.PHONY: test-all
test-all: test test-frontend test-integration

# 测试报告
.PHONY: test-report
test-report:
	@echo "生成测试覆盖率报告..."
	cd backend && go test -coverprofile=coverage.out ./internal/...
	cd backend && go tool cover -func=coverage.out | grep total
	cd frontend && npm test -- --coverage --coverageReporters=text-summary
```

### 2. GitHub Actions CI 配置

```yaml
# .github/workflows/test.yml
name: Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  backend-tests:
    name: Backend Tests
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: nodepass_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

      redis:
        image: redis:7-alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 6379:6379

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Cache Go modules
        uses: actions/cache@v4
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-

      - name: Run tests
        working-directory: backend
        env:
          NODEPASS_DATABASE_TYPE: postgres
          NODEPASS_DATABASE_HOST: localhost
          NODEPASS_DATABASE_PORT: 5432
          NODEPASS_DATABASE_USER: postgres
          NODEPASS_DATABASE_PASSWORD: postgres
          NODEPASS_DATABASE_DB_NAME: nodepass_test
          NODEPASS_REDIS_ADDR: localhost:6379
          NODEPASS_JWT_SECRET: test-secret-key-for-ci-testing-only-32-chars
        run: |
          go test -v -race -coverprofile=coverage.out -covermode=atomic ./internal/...

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v4
        with:
          file: ./backend/coverage.out
          flags: backend
          name: backend-coverage

  frontend-tests:
    name: Frontend Tests
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json

      - name: Install dependencies
        working-directory: frontend
        run: npm ci

      - name: Run tests
        working-directory: frontend
        run: npm test -- --coverage --watchAll=false

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v4
        with:
          file: ./frontend/coverage/coverage-final.json
          flags: frontend
          name: frontend-coverage

  integration-tests:
    name: Integration Tests
    runs-on: ubuntu-latest
    needs: [backend-tests, frontend-tests]

    steps:
      - uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Run integration tests
        run: |
          chmod +x tests/integration_test.sh
          ./tests/integration_test.sh
```

### 3. 前端测试配置

**package.json 添加测试脚本**
```json
{
  "scripts": {
    "test": "vitest",
    "test:ui": "vitest --ui",
    "test:coverage": "vitest --coverage"
  },
  "devDependencies": {
    "@testing-library/react": "^14.0.0",
    "@testing-library/jest-dom": "^6.1.5",
    "@testing-library/user-event": "^14.5.1",
    "@vitest/ui": "^1.0.4",
    "@vitest/coverage-v8": "^1.0.4",
    "vitest": "^1.0.4",
    "jsdom": "^23.0.1"
  }
}
```

**vitest.config.ts**
```typescript
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'lcov'],
      exclude: [
        'node_modules/',
        'src/test/',
        '**/*.d.ts',
        '**/*.config.*',
        '**/mockData',
        '**/*.test.{ts,tsx}',
      ],
      thresholds: {
        lines: 70,
        functions: 70,
        branches: 70,
        statements: 70,
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
```

**src/test/setup.ts**
```typescript
import '@testing-library/jest-dom'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

// 每个测试后清理
afterEach(() => {
  cleanup()
})

// Mock window.matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})
```

---

## 📈 预期覆盖率提升

### 完成 P0 任务后
```
后端服务层:     3.5%  → 75%  (+71.5%)
后端中间件:     3.8%  → 70%  (+66.2%)
前端 API 客户端: 0%   → 80%  (+80%)
前端工具函数:    0%   → 70%  (+70%)
─────────────────────────────────────
总体覆盖率:     ~8%   → 55%  (+47%)
```

### 完成 P0 + P1 任务后
```
后端处理器:     0%    → 60%  (+60%)
后端工具函数:   12.6% → 75%  (+62.4%)
前端组件:       0%    → 50%  (+50%)
授权中心:       15%   → 70%  (+55%)
─────────────────────────────────────
总体覆盖率:     ~8%   → 72%  (+64%)
```

---

## 🚀 执行计划

### 第一周 (P0 任务)
- Day 1-2: 完成中间件测试 ✅
- Day 3-4: 完成核心服务测试 (auth_service ✅, traffic_service, node_group_service)
- Day 5: 完成前端 API 客户端测试

### 第二周 (P1 任务)
- Day 1-2: 完成处理器测试
- Day 3-4: 完成前端组件测试
- Day 5: 完成授权中心测试

### 第三周 (P2 任务 + CI 集成)
- Day 1-2: 完成节点客户端测试
- Day 3: 配置 GitHub Actions CI
- Day 4: 集成 Codecov
- Day 5: 文档更新和总结

---

## 📊 测试质量标准

### 单元测试要求
- ✅ 每个公共函数至少 1 个测试用例
- ✅ 覆盖正常流程和异常流程
- ✅ 测试边界条件
- ✅ 使用表驱动测试 (table-driven tests)
- ✅ Mock 外部依赖
- ✅ 测试名称清晰描述测试场景

### 集成测试要求
- ✅ 测试完整的业务流程
- ✅ 使用真实的数据库 (测试环境)
- ✅ 测试 API 端到端
- ✅ 验证数据一致性

### 性能测试要求
- ✅ 关键路径添加 Benchmark
- ✅ 并发测试 (使用 -race 标志)
- ✅ 压力测试 (使用 testing.B)

---

## 📝 测试最佳实践

### Go 测试
```go
// ✅ 好的测试
func TestUserService_Create(t *testing.T) {
    tests := []struct {
        name        string
        input       *CreateUserRequest
        expectError bool
        errorMsg    string
    }{
        {
            name: "成功创建用户",
            input: &CreateUserRequest{
                Username: "testuser",
                Email:    "test@example.com",
            },
            expectError: false,
        },
        {
            name: "用户名为空",
            input: &CreateUserRequest{
                Username: "",
                Email:    "test@example.com",
            },
            expectError: true,
            errorMsg:    "用户名不能为空",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试逻辑
        })
    }
}
```

### TypeScript 测试
```typescript
// ✅ 好的测试
describe('API Client', () => {
  describe('Token 刷新', () => {
    it('应该自动刷新过期的 token', async () => {
      // Arrange
      const expiredToken = 'expired-token'
      const newToken = 'new-token'

      // Act
      const result = await apiClient.get('/test')

      // Assert
      expect(result.status).toBe(200)
      expect(getStoredToken()).toBe(newToken)
    })
  })
})
```

---

## 🎯 成功指标

- ✅ 总体测试覆盖率达到 70%+
- ✅ 核心模块覆盖率达到 80%+
- ✅ 所有 CI 测试通过
- ✅ 无测试失败或跳过
- ✅ 测试执行时间 < 5 分钟
- ✅ 代码审查通过率 > 95%

---

## 📚 参考资源

- [Go Testing 官方文档](https://golang.org/pkg/testing/)
- [Testify 断言库](https://github.com/stretchr/testify)
- [Vitest 文档](https://vitest.dev/)
- [React Testing Library](https://testing-library.com/react)
- [测试金字塔](https://martinfowler.com/articles/practical-test-pyramid.html)
