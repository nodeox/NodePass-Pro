# Backend 重构指南：Redis + PostgreSQL 架构

## 架构概览

```
┌─────────────────────────────────────────────────────────┐
│                   Application Layer                      │
│              (Handlers + Use Cases)                      │
└────────────────────┬────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
┌───────▼────────┐      ┌────────▼────────┐
│  Redis Cache   │      │   PostgreSQL    │
│                │      │                 │
│ • 会话管理     │      │ • 持久化存储    │
│ • 限流计数     │      │ • 事务处理      │
│ • 热点数据     │      │ • 复杂查询      │
│ • 分布式锁     │      │ • JSONB 支持    │
│ • 消息队列     │      │ • 时序优化      │
└────────────────┘      └─────────────────┘
```

## 数据分层策略

### 1. 缓存层（Redis）

**用户会话**
- Key: `session:{token}`
- TTL: 24h
- 数据: 用户 ID、角色、权限

**用户信息缓存**
- Key: `user:{id}`
- TTL: 5min
- 数据: 用户基本信息（热点数据）

**流量计数器**
- Key: `traffic:user:{id}:in/out`
- TTL: 7d
- 操作: INCRBY（原子递增）

**节点在线状态**
- Key: `node:online:{node_id}`
- TTL: 3min（心跳间隔 + 容错时间）
- 数据: 1（在线标记）

**限流计数**
- Key: `ratelimit:{ip}:{endpoint}`
- TTL: 1min
- 操作: INCR + EXPIRE

**分布式锁**
- Key: `lock:{resource}`
- TTL: 30s
- 数据: 唯一标识符

### 2. 持久化层（PostgreSQL）

**用户数据**
- 表: `users`
- 索引: email, username, status
- 特性: 唯一约束、外键关联

**节点数据**
- 表: `node_instances`
- 索引: group_id, status, last_heartbeat_at
- 特性: 复合索引优化查询

**流量记录**
- 表: `traffic_records`
- 分区: 按月分区
- 索引: user_id + created_at
- 优化: TimescaleDB 超表

**配置数据**
- 表: `node_configs`
- 类型: JSONB
- 索引: GIN 索引支持 JSON 查询

## 重构步骤

### 阶段 1：建立基础设施层（1-2 周）

#### 1.1 创建领域层

```bash
internal/domain/
├── user/
│   ├── entity.go       # 用户实体
│   ├── repository.go   # 仓储接口
│   └── service.go      # 领域服务
├── node/
│   ├── entity.go
│   ├── repository.go
│   └── service.go
└── tunnel/
    ├── entity.go
    ├── repository.go
    └── service.go
```

#### 1.2 实现基础设施层

```bash
internal/infrastructure/
├── persistence/
│   └── postgres/
│       ├── user_repository.go
│       ├── node_repository.go
│       └── tunnel_repository.go
└── cache/
    ├── user_cache.go
    ├── traffic_counter.go
    ├── heartbeat_buffer.go
    └── distributed_lock.go
```

### 阶段 2：重构核心模块（2-3 周）

#### 2.1 用户模块重构

**Before（旧代码）:**
```go
// handlers/auth_handler.go
func (h *AuthHandler) Login(c *gin.Context) {
    // 直接操作数据库
    var user models.User
    database.DB.Where("email = ?", req.Email).First(&user)
    // ...
}
```

**After（新代码）:**
```go
// application/user/commands/login.go
type LoginCommand struct {
    Email    string
    Password string
}

type LoginHandler struct {
    userRepo  user.Repository
    userCache *cache.UserCache
}

func (h *LoginHandler) Handle(ctx context.Context, cmd LoginCommand) (*LoginResult, error) {
    // 1. 查询用户（先查缓存）
    u, err := h.userRepo.FindByEmail(ctx, cmd.Email)
    if err != nil {
        return nil, err
    }
    
    // 2. 验证密码
    if !u.VerifyPassword(cmd.Password) {
        return nil, user.ErrInvalidPassword
    }
    
    // 3. 更新缓存
    h.userCache.Set(ctx, u.ID, toCache(u))
    
    return &LoginResult{User: u}, nil
}
```

#### 2.2 节点心跳优化

**策略：Redis 缓冲 + 批量写入**

```go
// 心跳接收（写入 Redis）
func (h *NodeHandler) Heartbeat(c *gin.Context) {
    data := &cache.HeartbeatData{
        NodeID:    req.NodeID,
        Status:    "online",
        CPUUsage:  req.CPUUsage,
        Timestamp: time.Now(),
    }
    
    // 推入 Redis 缓冲区
    h.heartbeatBuffer.Push(c.Request.Context(), data)
    
    // 更新在线状态（3 分钟 TTL）
    h.heartbeatBuffer.SetNodeOnline(c.Request.Context(), req.NodeID, 3*time.Minute)
}

// 定时任务（批量写入 PostgreSQL）
func (s *NodeService) FlushHeartbeats(ctx context.Context) error {
    nodes, _ := s.heartbeatBuffer.GetAllOnlineNodes(ctx)
    
    for _, nodeID := range nodes {
        // 批量弹出 100 条
        data, _ := s.heartbeatBuffer.PopBatch(ctx, nodeID, 100)
        
        // 批量插入数据库
        s.nodeRepo.BatchInsertHeartbeats(ctx, data)
    }
    return nil
}
```

#### 2.3 流量统计优化

**策略：Redis 实时计数 + 定期同步**

```go
// 流量上报（Redis 原子递增）
func (h *TrafficHandler) Report(c *gin.Context) {
    // Redis 原子递增
    h.trafficCounter.IncrementUserTraffic(
        c.Request.Context(),
        userID,
        req.InBytes,
        req.OutBytes,
    )
}

// 定时同步到 PostgreSQL（每小时）
func (s *TrafficService) SyncToDatabase(ctx context.Context) error {
    // 获取所有用户的流量
    userIDs := s.getAllActiveUserIDs(ctx)
    traffic := s.trafficCounter.BatchGetUserTraffic(ctx, userIDs)
    
    // 批量更新数据库
    for userID, data := range traffic {
        s.userRepo.UpdateTraffic(ctx, userID, data.In, data.Out)
        
        // 重置 Redis 计数器
        s.trafficCounter.ResetUserTraffic(ctx, userID)
    }
    return nil
}
```

### 阶段 3：PostgreSQL 优化（1-2 周）

#### 3.1 安装 TimescaleDB

```sql
-- 安装扩展
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- 转换为超表
SELECT create_hypertable('traffic_records', 'created_at', 
    chunk_time_interval => INTERVAL '1 day');

-- 自动压缩
ALTER TABLE traffic_records SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'user_id'
);

-- 自动删除旧数据
SELECT add_retention_policy('traffic_records', INTERVAL '90 days');
```

#### 3.2 索引优化

```sql
-- 用户表
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status) WHERE status = 'normal';

-- 节点表
CREATE INDEX idx_nodes_group_status ON node_instances(group_id, status);
CREATE INDEX idx_nodes_heartbeat ON node_instances(last_heartbeat_at DESC);

-- 流量表（时序优化）
CREATE INDEX idx_traffic_user_time ON traffic_records(user_id, created_at DESC);

-- JSONB 索引
CREATE INDEX idx_node_config_gin ON node_configs USING GIN (config);
```

#### 3.3 分区表

```sql
-- 按月分区
CREATE TABLE traffic_records (
    id BIGSERIAL,
    user_id INT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    traffic_in BIGINT,
    traffic_out BIGINT
) PARTITION BY RANGE (created_at);

-- 创建分区
CREATE TABLE traffic_records_2024_01 PARTITION OF traffic_records
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE traffic_records_2024_02 PARTITION OF traffic_records
FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
```

## 性能优化建议

### 1. 连接池配置

```yaml
database:
  type: postgres
  max_idle_conns: 25
  max_open_conns: 100
  conn_max_lifetime: 300  # 5 分钟
  conn_max_idle_time: 600 # 10 分钟

redis:
  pool_size: 50
  min_idle_conns: 10
```

### 2. 缓存策略

**Cache-Aside 模式**
```go
func (s *UserService) GetUser(ctx context.Context, id uint) (*User, error) {
    // 1. 查缓存
    if cached, err := s.cache.Get(ctx, id); err == nil && cached != nil {
        return cached, nil
    }
    
    // 2. 查数据库
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // 3. 写缓存
    s.cache.Set(ctx, id, user)
    
    return user, nil
}
```

**Write-Through 模式**
```go
func (s *UserService) UpdateUser(ctx context.Context, user *User) error {
    // 1. 更新数据库
    if err := s.repo.Update(ctx, user); err != nil {
        return err
    }
    
    // 2. 更新缓存
    s.cache.Set(ctx, user.ID, user)
    
    return nil
}
```

### 3. 批量操作

```go
// 批量插入（使用事务）
func (r *UserRepository) BatchCreate(ctx context.Context, users []*User) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        return tx.CreateInBatches(users, 100).Error
    })
}

// 批量查询（IN 查询）
func (r *UserRepository) FindByIDs(ctx context.Context, ids []uint) ([]*User, error) {
    var users []*User
    err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error
    return users, err
}
```

## 监控指标

### Redis 监控

```bash
# 内存使用
redis-cli INFO memory

# 命中率
redis-cli INFO stats | grep keyspace

# 慢查询
redis-cli SLOWLOG GET 10
```

### PostgreSQL 监控

```sql
-- 慢查询
SELECT * FROM pg_stat_statements 
ORDER BY mean_exec_time DESC 
LIMIT 10;

-- 索引使用率
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
ORDER BY idx_scan;

-- 表大小
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

## 迁移清单

- [ ] 创建领域层结构
- [ ] 实现 Repository 接口
- [ ] 实现 Redis 缓存层
- [ ] 重构用户模块
- [ ] 重构节点模块
- [ ] 重构隧道模块
- [ ] 优化心跳处理
- [ ] 优化流量统计
- [ ] 安装 TimescaleDB
- [ ] 创建索引
- [ ] 配置分区表
- [ ] 编写单元测试
- [ ] 性能测试
- [ ] 文档更新

## 预期收益

1. **性能提升**
   - 心跳处理：从同步写入改为异步批量，TPS 提升 10x
   - 流量统计：Redis 原子操作，延迟降低 90%
   - 用户查询：缓存命中率 80%+，响应时间 < 10ms

2. **可维护性**
   - 清晰的分层架构
   - 领域逻辑集中管理
   - 易于测试和扩展

3. **可扩展性**
   - 支持水平扩展
   - 读写分离友好
   - 微服务化准备

4. **成本优化**
   - 数据库压力降低 60%
   - 存储成本降低（自动压缩和清理）
   - 运维成本降低
