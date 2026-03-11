# Redis + PostgreSQL 架构方案总结

## 一、技术选型

### 为什么选择 PostgreSQL？

✅ **JSONB 支持**：原生支持 JSON 数据类型，性能优异  
✅ **时序数据优化**：TimescaleDB 扩展，完美支持流量、心跳等时序数据  
✅ **高并发**：MVCC 机制，读写不阻塞  
✅ **扩展性强**：支持分区表、物化视图、全文搜索  
✅ **开源生态**：完全开源，无商业限制  

### 为什么选择 Redis？

✅ **高性能**：内存存储，微秒级延迟  
✅ **丰富数据结构**：String、Hash、List、Set、Sorted Set  
✅ **原子操作**：INCR、INCRBY 等原子命令，完美支持计数器  
✅ **过期机制**：自动清理过期数据  
✅ **Pub/Sub**：支持消息发布订阅  

## 二、架构设计

### 分层架构

```
┌─────────────────────────────────────────┐
│         Interfaces Layer                │
│    (HTTP Handlers + WebSocket)          │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│       Application Layer                 │
│  (Commands + Queries + Use Cases)       │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│          Domain Layer                   │
│   (Entities + Repositories + Services)  │
└──────────────┬──────────────────────────┘
               │
       ┌───────┴────────┐
       │                │
┌──────▼─────┐   ┌─────▼──────┐
│   Redis    │   │ PostgreSQL │
│   Cache    │   │    DB      │
└────────────┘   └────────────┘
```

### 数据流设计

#### 1. 用户查询流程（读操作）

```
用户请求 → Handler → Query Handler
                         ↓
                    查询 Redis 缓存
                         ↓
                   缓存命中？
                    ↙     ↘
                  是       否
                  ↓        ↓
              返回数据   查询 PostgreSQL
                         ↓
                     写入 Redis
                         ↓
                     返回数据
```

#### 2. 用户创建流程（写操作）

```
用户请求 → Handler → Command Handler
                         ↓
                   验证业务规则
                         ↓
                  写入 PostgreSQL
                         ↓
                   写入 Redis 缓存
                         ↓
                     返回结果
```

#### 3. 节点心跳流程（高频写入）

```
节点心跳 → Handler → 写入 Redis List
                         ↓
                   更新在线状态（TTL 3min）
                         ↓
                     立即返回
                         
定时任务（每分钟）
    ↓
批量从 Redis 弹出数据
    ↓
批量写入 PostgreSQL
```

#### 4. 流量统计流程（实时计数）

```
流量上报 → Handler → Redis INCRBY（原子操作）
                         ↓
                     立即返回
                         
定时任务（每小时）
    ↓
读取 Redis 计数器
    ↓
批量更新 PostgreSQL
    ↓
重置 Redis 计数器
```

## 三、核心组件

### 1. 用户缓存（UserCache）

**功能：**
- 用户信息缓存（5 分钟 TTL）
- 邮箱索引（快速查找）
- 流量计数器（原子递增）

**使用场景：**
- 用户登录验证
- 权限检查
- 个人信息查询

### 2. 心跳缓冲区（HeartbeatBuffer）

**功能：**
- Redis List 缓冲心跳数据
- 节点在线状态管理（TTL 机制）
- 批量弹出数据

**使用场景：**
- 节点心跳接收（高频写入）
- 在线状态检测
- 定时批量持久化

### 3. 流量计数器（TrafficCounter）

**功能：**
- 用户流量原子递增
- 隧道流量统计
- 批量查询

**使用场景：**
- 实时流量统计
- 流量配额检查
- 定期同步到数据库

### 4. 分布式锁（DistributedLock）

**功能：**
- 基于 Redis 的分布式锁
- 自动过期机制
- Lua 脚本保证原子性

**使用场景：**
- 防止并发冲突
- 定时任务互斥
- 资源竞争控制

## 四、性能优化

### PostgreSQL 优化

#### 1. TimescaleDB 时序优化

```sql
-- 安装扩展
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- 转换为超表（自动分区）
SELECT create_hypertable('traffic_records', 'created_at', 
    chunk_time_interval => INTERVAL '1 day');

-- 自动压缩（节省 90% 存储）
ALTER TABLE traffic_records SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'user_id'
);

-- 自动删除旧数据
SELECT add_retention_policy('traffic_records', INTERVAL '90 days');
```

#### 2. 索引优化

```sql
-- 复合索引（覆盖常用查询）
CREATE INDEX idx_traffic_user_time ON traffic_records(user_id, created_at DESC);

-- 部分索引（只索引活跃数据）
CREATE INDEX idx_active_users ON users(id) WHERE status = 'normal';

-- JSONB GIN 索引
CREATE INDEX idx_node_config_gin ON node_configs USING GIN (config);

-- 表达式索引
CREATE INDEX idx_users_email_lower ON users(LOWER(email));
```

#### 3. 连接池配置

```yaml
database:
  max_idle_conns: 25      # 空闲连接数
  max_open_conns: 100     # 最大连接数
  conn_max_lifetime: 300  # 连接最大生命周期（秒）
  conn_max_idle_time: 600 # 空闲连接最大生命周期（秒）
```

### Redis 优化

#### 1. 内存优化

```bash
# 启用 LRU 淘汰策略
maxmemory 2gb
maxmemory-policy allkeys-lru

# 启用压缩
hash-max-ziplist-entries 512
hash-max-ziplist-value 64
```

#### 2. 持久化配置

```bash
# RDB + AOF 混合持久化
save 900 1
save 300 10
save 60 10000

appendonly yes
appendfsync everysec
```

#### 3. 连接池配置

```yaml
redis:
  pool_size: 50          # 连接池大小
  min_idle_conns: 10     # 最小空闲连接
  max_retries: 3         # 最大重试次数
  dial_timeout: 5s       # 连接超时
  read_timeout: 3s       # 读超时
  write_timeout: 3s      # 写超时
```

## 五、监控与运维

### 关键指标

#### Redis 监控

```bash
# 内存使用率
used_memory / maxmemory

# 缓存命中率
keyspace_hits / (keyspace_hits + keyspace_misses)

# 连接数
connected_clients

# 慢查询
SLOWLOG GET 10
```

#### PostgreSQL 监控

```sql
-- 慢查询（需要 pg_stat_statements 扩展）
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;

-- 索引使用率
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0;

-- 表膨胀
SELECT schemaname, tablename, 
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename))
FROM pg_tables
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

### 告警规则

```yaml
alerts:
  - name: redis_memory_high
    condition: used_memory > 1.8GB
    action: 清理过期键、增加内存

  - name: redis_hit_rate_low
    condition: hit_rate < 80%
    action: 检查缓存策略、增加 TTL

  - name: postgres_slow_query
    condition: query_time > 1s
    action: 优化 SQL、添加索引

  - name: postgres_connection_high
    condition: connections > 80
    action: 检查连接泄漏、增加连接池
```

## 六、迁移步骤

### 第 1 周：基础设施

- [x] 创建领域层结构
- [x] 定义 Repository 接口
- [x] 实现 PostgreSQL Repository
- [x] 实现 Redis Cache 层
- [ ] 编写单元测试

### 第 2-3 周：核心模块重构

- [ ] 重构用户模块
- [ ] 重构节点模块
- [ ] 重构隧道模块
- [ ] 优化心跳处理
- [ ] 优化流量统计

### 第 4 周：数据库优化

- [ ] 安装 TimescaleDB
- [ ] 创建索引
- [ ] 配置分区表
- [ ] 性能测试
- [ ] 压力测试

### 第 5 周：上线准备

- [ ] 集成测试
- [ ] 灰度发布
- [ ] 监控配置
- [ ] 文档更新
- [ ] 团队培训

## 七、预期收益

### 性能提升

| 指标 | 优化前 | 优化后 | 提升 |
|------|--------|--------|------|
| 心跳 TPS | 100 | 1000+ | 10x |
| 流量统计延迟 | 100ms | 5ms | 20x |
| 用户查询延迟 | 50ms | 5ms | 10x |
| 数据库 CPU | 60% | 20% | 3x |
| 缓存命中率 | 0% | 85% | ∞ |

### 成本优化

- **数据库成本**：降低 60%（减少查询压力）
- **存储成本**：降低 50%（TimescaleDB 压缩）
- **运维成本**：降低 40%（自动化清理）

### 可维护性

- ✅ 清晰的分层架构
- ✅ 领域逻辑集中管理
- ✅ 易于测试和扩展
- ✅ 支持水平扩展

## 八、风险与应对

### 风险 1：缓存一致性

**问题：** Redis 和 PostgreSQL 数据不一致

**应对：**
- 使用 Write-Through 模式（写数据库后立即更新缓存）
- 设置合理的 TTL（5 分钟）
- 提供手动刷新缓存接口

### 风险 2：Redis 故障

**问题：** Redis 宕机导致服务不可用

**应对：**
- Redis 哨兵模式（高可用）
- 降级策略（直接查数据库）
- 熔断机制

### 风险 3：数据迁移

**问题：** 旧数据迁移到新架构

**应对：**
- 双写模式（同时写旧表和新表）
- 数据校验脚本
- 灰度发布

## 九、下一步行动

1. **立即开始**：创建基础设施层代码
2. **本周完成**：用户模块重构示例
3. **下周开始**：节点模块重构
4. **两周后**：性能测试和优化

需要我帮你开始实施吗？我可以：
- 完成用户模块的完整重构
- 编写单元测试
- 创建迁移脚本
- 配置 Docker Compose 环境
