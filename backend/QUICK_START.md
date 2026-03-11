# 快速开始：Redis + PostgreSQL 重构

## 一、环境准备

### 1. 启动开发环境

```bash
# 启动 PostgreSQL + Redis + 管理界面
docker-compose -f docker-compose.dev.yml up -d

# 查看服务状态
docker-compose -f docker-compose.dev.yml ps

# 查看日志
docker-compose -f docker-compose.dev.yml logs -f
```

**服务访问：**
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`
- PgAdmin: `http://localhost:5050` (admin@nodepass.local / admin)
- Redis Commander: `http://localhost:8081`

### 2. 配置应用

```yaml
# configs/config.yaml
database:
  type: postgres
  host: localhost
  port: 5432
  user: nodepass
  password: nodepass_dev_password
  db_name: nodepass_pro
  ssl_mode: disable
  max_idle_conns: 25
  max_open_conns: 100
  conn_max_lifetime: 300
  conn_max_idle_time: 600

redis:
  enabled: true
  addr: localhost:6379
  password: ""
  db: 0
  key_prefix: "nodepass:panel"
  default_ttl: 300
```

## 二、代码结构

### 已创建的文件

```
backend/
├── internal/
│   ├── domain/                    # 领域层
│   │   └── user/
│   │       ├── entity.go          ✅ 用户实体
│   │       └── repository.go      ✅ 仓储接口
│   │
│   ├── application/               # 应用层
│   │   └── user/
│   │       ├── commands/
│   │       │   └── create_user.go ✅ 创建用户命令
│   │       └── queries/
│   │           └── get_user.go    ✅ 获取用户查询
│   │
│   └── infrastructure/            # 基础设施层
│       ├── persistence/
│       │   └── postgres/
│       │       └── user_repository.go ✅ PostgreSQL 实现
│       └── cache/
│           ├── user_cache.go          ✅ 用户缓存
│           ├── traffic_counter.go     ✅ 流量计数器
│           ├── heartbeat_buffer.go    ✅ 心跳缓冲区
│           └── distributed_lock.go    ✅ 分布式锁
│
├── docker-compose.dev.yml         ✅ 开发环境
├── scripts/
│   └── init_db.sql                ✅ 数据库初始化
├── REFACTORING_GUIDE.md           ✅ 重构指南
├── REDIS_POSTGRES_ARCHITECTURE.md ✅ 架构文档
└── QUICK_START.md                 ✅ 快速开始
```

## 三、使用示例

### 1. 创建用户（命令模式）

```go
package main

import (
    "context"
    "nodepass-pro/backend/internal/application/user/commands"
    "nodepass-pro/backend/internal/domain/user"
    "nodepass-pro/backend/internal/infrastructure/cache"
    "nodepass-pro/backend/internal/infrastructure/persistence/postgres"
)

func main() {
    // 初始化依赖
    userRepo := postgres.NewUserRepository(db)
    userCache := cache.NewUserCache(redisClient)
    
    // 创建命令处理器
    handler := commands.NewCreateUserHandler(userRepo, userCache)
    
    // 执行命令
    cmd := commands.CreateUserCommand{
        Username: "testuser",
        Email:    "test@example.com",
        Password: "password123",
        Role:     "user",
    }
    
    result, err := handler.Handle(context.Background(), cmd)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("用户创建成功: ID=%d\n", result.User.ID)
}
```

### 2. 查询用户（查询模式）

```go
package main

import (
    "context"
    "nodepass-pro/backend/internal/application/user/queries"
)

func main() {
    // 创建查询处理器
    handler := queries.NewGetUserHandler(userRepo, userCache)
    
    // 执行查询
    query := queries.GetUserQuery{UserID: 1}
    result, err := handler.Handle(context.Background(), query)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("用户: %s (%s)\n", result.User.Username, result.User.Email)
}
```

### 3. 流量统计

```go
package main

import (
    "context"
    "nodepass-pro/backend/internal/infrastructure/cache"
)

func main() {
    counter := cache.NewTrafficCounter(redisClient)
    
    // 增加流量（原子操作）
    err := counter.IncrementUserTraffic(ctx, userID, 1024*1024, 512*1024)
    
    // 查询流量
    inBytes, outBytes, err := counter.GetUserTraffic(ctx, userID)
    fmt.Printf("流量: 入=%d, 出=%d\n", inBytes, outBytes)
}
```

### 4. 心跳缓冲

```go
package main

import (
    "context"
    "time"
    "nodepass-pro/backend/internal/infrastructure/cache"
)

func main() {
    buffer := cache.NewHeartbeatBuffer(redisClient)
    
    // 推送心跳
    data := &cache.HeartbeatData{
        NodeID:      "node-001",
        Status:      "online",
        CPUUsage:    45.5,
        MemoryUsage: 60.2,
        Timestamp:   time.Now(),
    }
    buffer.Push(ctx, data)
    
    // 更新在线状态
    buffer.SetNodeOnline(ctx, "node-001", 3*time.Minute)
    
    // 批量弹出（定时任务）
    heartbeats, err := buffer.PopBatch(ctx, "node-001", 100)
    // 批量写入数据库...
}
```

### 5. 分布式锁

```go
package main

import (
    "context"
    "time"
    "nodepass-pro/backend/internal/infrastructure/cache"
)

func main() {
    // 使用锁执行函数
    err := cache.WithLock(ctx, redisClient, "user:123:update", 30*time.Second, func() error {
        // 执行需要加锁的操作
        return updateUser(123)
    })
    
    if err == cache.ErrLockFailed {
        fmt.Println("获取锁失败，其他进程正在处理")
    }
}
```

## 四、数据库优化

### 1. 应用启动后执行

```sql
-- 连接到数据库
psql -h localhost -U nodepass -d nodepass_pro

-- 创建优化索引
SELECT create_optimized_indexes();

-- 转换时序表
SELECT convert_to_hypertable();

-- 查看超表状态
SELECT * FROM timescaledb_information.hypertables;

-- 查看压缩策略
SELECT * FROM timescaledb_information.compression_settings;
```

### 2. 验证优化效果

```sql
-- 查看表大小
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- 查看索引使用情况
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan AS scans,
    pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- 查看慢查询
SELECT 
    query,
    calls,
    mean_exec_time,
    max_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;
```

## 五、测试

### 1. 单元测试

```bash
# 测试用户仓储
go test ./internal/infrastructure/persistence/postgres -v

# 测试缓存层
go test ./internal/infrastructure/cache -v

# 测试应用层
go test ./internal/application/user/... -v
```

### 2. 集成测试

```bash
# 启动测试环境
docker-compose -f docker-compose.dev.yml up -d

# 运行集成测试
go test ./internal/... -tags=integration -v
```

### 3. 性能测试

```bash
# 使用 wrk 进行压测
wrk -t4 -c100 -d30s http://localhost:8080/api/v1/users/1

# 使用 ab 进行压测
ab -n 10000 -c 100 http://localhost:8080/api/v1/users/1
```

## 六、监控

### 1. Redis 监控

```bash
# 连接 Redis
redis-cli

# 查看内存使用
INFO memory

# 查看命中率
INFO stats

# 查看慢查询
SLOWLOG GET 10

# 查看键空间
INFO keyspace
```

### 2. PostgreSQL 监控

```sql
-- 当前连接数
SELECT count(*) FROM pg_stat_activity;

-- 活跃查询
SELECT pid, usename, query, state, query_start
FROM pg_stat_activity
WHERE state != 'idle'
ORDER BY query_start;

-- 锁等待
SELECT * FROM pg_locks WHERE NOT granted;

-- 表膨胀
SELECT 
    schemaname,
    tablename,
    n_live_tup,
    n_dead_tup,
    round(n_dead_tup * 100.0 / NULLIF(n_live_tup + n_dead_tup, 0), 2) AS dead_ratio
FROM pg_stat_user_tables
WHERE n_dead_tup > 1000
ORDER BY n_dead_tup DESC;
```

## 七、常见问题

### Q1: 缓存不一致怎么办？

**A:** 使用 Write-Through 模式，更新数据库后立即更新缓存。如果发现不一致，可以：

```bash
# 清空用户缓存
redis-cli DEL "user:123"

# 批量清空
redis-cli --scan --pattern "user:*" | xargs redis-cli DEL
```

### Q2: Redis 内存不足？

**A:** 检查并清理：

```bash
# 查看内存使用
redis-cli INFO memory

# 查看大键
redis-cli --bigkeys

# 设置淘汰策略
redis-cli CONFIG SET maxmemory-policy allkeys-lru
```

### Q3: PostgreSQL 慢查询？

**A:** 优化步骤：

```sql
-- 1. 查看执行计划
EXPLAIN ANALYZE SELECT * FROM users WHERE email = 'test@example.com';

-- 2. 添加索引
CREATE INDEX idx_users_email ON users(email);

-- 3. 更新统计信息
ANALYZE users;

-- 4. 重建索引
REINDEX TABLE users;
```

## 八、下一步

1. **完成用户模块重构**：参考已创建的代码
2. **重构节点模块**：复用相同的模式
3. **重构隧道模块**：添加缓存层
4. **性能测试**：验证优化效果
5. **灰度发布**：逐步切换到新架构

## 九、参考资料

- [REFACTORING_GUIDE.md](./REFACTORING_GUIDE.md) - 详细重构指南
- [REDIS_POSTGRES_ARCHITECTURE.md](./REDIS_POSTGRES_ARCHITECTURE.md) - 架构设计文档
- [TimescaleDB 文档](https://docs.timescale.com/)
- [Redis 最佳实践](https://redis.io/docs/manual/patterns/)
- [PostgreSQL 性能优化](https://www.postgresql.org/docs/current/performance-tips.html)

需要帮助？查看文档或提 Issue！
