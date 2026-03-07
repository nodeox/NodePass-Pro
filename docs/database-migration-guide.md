# 数据库迁移文档 - 节点分组功能

## 📋 概述

本迁移为 NodePass-Pro 添加完整的节点分组管理功能，包括：
- 节点组管理（入口组/出口组）
- 节点实例管理
- 节点组关联
- 隧道管理（重构）
- 自动统计和触发器

---

## 📁 文件清单

```
backend/
├── migrations/
│   ├── 000X_create_node_groups.up.sql    # UP 迁移（创建表）
│   ├── 000X_create_node_groups.down.sql  # DOWN 迁移（删除表）
│   └── test_data.sql                      # 测试数据
├── scripts/
│   └── migrate-node-groups.sh             # 迁移执行脚本
└── internal/
    ├── models/
    │   └── node_group.go                  # 数据模型
    └── services/
        └── node_group_service.go          # 业务逻辑
```

---

## 🗄️ 数据库变更

### 新增表

1. **node_groups** - 节点组表
   - 主键: `id`
   - 外键: `user_id` → `users(id)`
   - 索引: `user_id`, `type`, `enabled`
   - 唯一索引: `(user_id, name)`

2. **node_instances** - 节点实例表
   - 主键: `id`
   - 外键: `group_id` → `node_groups(id)`, `user_id` → `users(id)`
   - 索引: `group_id`, `user_id`, `status`, `node_id`
   - 唯一索引: `node_id`, `(user_id, service_name)`

3. **node_group_relations** - 节点组关联表
   - 主键: `(entry_group_id, exit_group_id)`
   - 外键: `entry_group_id` → `node_groups(id)`, `exit_group_id` → `node_groups(id)`
   - 索引: `entry_group_id`, `exit_group_id`

4. **node_group_stats** - 节点组统计表
   - 主键: `group_id`
   - 外键: `group_id` → `node_groups(id)`

5. **tunnels** - 隧道表（重构）
   - 主键: `id`
   - 外键: `user_id` → `users(id)`, `entry_group_id` → `node_groups(id)`, `exit_group_id` → `node_groups(id)`
   - 索引: `user_id`, `entry_group_id`, `exit_group_id`, `status`

### 新增视图

1. **node_groups_with_stats** - 节点组详细信息视图
   - 包含节点组基本信息和统计数据

2. **tunnels_with_groups** - 隧道详细信息视图
   - 包含隧道信息和关联的节点组信息

### 新增函数

1. **update_updated_at_column()** - 自动更新 updated_at 字段
2. **update_node_group_stats()** - 自动更新节点组统计
3. **get_available_ports(group_id)** - 获取可用端口列表
4. **validate_node_group_config(type, config)** - 验证节点组配置

### 新增触发器

1. **trigger_node_groups_updated_at** - 自动更新 node_groups.updated_at
2. **trigger_node_instances_updated_at** - 自动更新 node_instances.updated_at
3. **trigger_tunnels_updated_at** - 自动更新 tunnels.updated_at
4. **trigger_update_node_group_stats** - 自动更新节点组统计

---

## 🚀 执行迁移

### 方式一：使用迁移脚本（推荐）

```bash
cd backend

# 执行迁移
./scripts/migrate-node-groups.sh up

# 验证迁移
./scripts/migrate-node-groups.sh verify

# 如需回滚
./scripts/migrate-node-groups.sh rollback <backup_file>
```

### 方式二：手动执行

```bash
# 1. 备份数据库
pg_dump -h localhost -U postgres -d nodepass_panel > backup.sql

# 2. 执行迁移
psql -h localhost -U postgres -d nodepass_panel -f migrations/000X_create_node_groups.up.sql

# 3. 验证
psql -h localhost -U postgres -d nodepass_panel -c "\dt node_*"
psql -h localhost -U postgres -d nodepass_panel -c "\dv"
psql -h localhost -U postgres -d nodepass_panel -c "\df"

# 4. 如需回滚
psql -h localhost -U postgres -d nodepass_panel -f migrations/000X_create_node_groups.down.sql
```

### 方式三：使用 Docker Compose

```bash
# 进入数据库容器
docker compose exec postgres psql -U postgres -d nodepass_panel

# 在 psql 中执行
\i /path/to/migrations/000X_create_node_groups.up.sql
```

---

## 🧪 测试迁移

### 1. 插入测试数据

```bash
psql -h localhost -U postgres -d nodepass_panel -f migrations/test_data.sql
```

### 2. 验证数据

```sql
-- 查看节点组（包含统计）
SELECT * FROM node_groups_with_stats;

-- 查看节点实例
SELECT
    ni.id,
    ni.node_id,
    ni.service_name,
    ni.status,
    ng.name AS group_name,
    ng.type AS group_type
FROM node_instances ni
JOIN node_groups ng ON ni.group_id = ng.id;

-- 查看隧道（包含节点组信息）
SELECT * FROM tunnels_with_groups;

-- 查看节点组统计
SELECT
    ng.name,
    ng.type,
    ngs.node_count,
    ngs.online_count,
    ngs.total_traffic,
    ngs.active_connections
FROM node_groups ng
JOIN node_group_stats ngs ON ng.id = ngs.group_id;

-- 测试获取可用端口函数
SELECT * FROM get_available_ports(1) LIMIT 10;

-- 测试配置验证函数
SELECT * FROM validate_node_group_config(
    'entry',
    '{"allowed_protocols": ["tcp"], "entry_config": {"traffic_multiplier": 1.0}}'::jsonb
);
```

### 3. 测试触发器

```sql
-- 测试统计自动更新
-- 插入新节点实例
INSERT INTO node_instances (
    group_id, user_id, node_id, service_name,
    connection_address, status
)
VALUES (
    1, 1, 'test-node-' || gen_random_uuid()::text,
    'test-service', '127.0.0.1', 'online'
);

-- 查看统计是否自动更新
SELECT * FROM node_group_stats WHERE group_id = 1;

-- 测试 updated_at 自动更新
UPDATE node_groups SET name = 'Updated Name' WHERE id = 1;
SELECT id, name, updated_at FROM node_groups WHERE id = 1;
```

---

## 📊 性能优化

### 索引说明

所有关键查询字段都已添加索引：

```sql
-- 节点组
CREATE INDEX idx_node_groups_user ON node_groups(user_id);
CREATE INDEX idx_node_groups_type ON node_groups(type);
CREATE INDEX idx_node_groups_enabled ON node_groups(enabled);

-- 节点实例
CREATE INDEX idx_node_instances_group ON node_instances(group_id);
CREATE INDEX idx_node_instances_user ON node_instances(user_id);
CREATE INDEX idx_node_instances_status ON node_instances(status);

-- 隧道
CREATE INDEX idx_tunnels_user ON tunnels(user_id);
CREATE INDEX idx_tunnels_entry_group ON tunnels(entry_group_id);
CREATE INDEX idx_tunnels_exit_group ON tunnels(exit_group_id);
CREATE INDEX idx_tunnels_status ON tunnels(status);
```

### 查询优化建议

1. **使用视图**：优先使用 `node_groups_with_stats` 和 `tunnels_with_groups` 视图
2. **分页查询**：大量数据时使用 LIMIT 和 OFFSET
3. **避免 N+1**：使用 JOIN 而不是多次查询
4. **使用 EXPLAIN**：分析慢查询

```sql
-- 示例：分析查询性能
EXPLAIN ANALYZE
SELECT * FROM node_groups_with_stats
WHERE user_id = 1 AND type = 'entry';
```

---

## ⚠️ 注意事项

### 1. 数据备份

**在执行迁移前务必备份数据库！**

```bash
# 完整备份
pg_dump -h localhost -U postgres -d nodepass_panel > backup_$(date +%Y%m%d_%H%M%S).sql

# 仅备份数据（不含结构）
pg_dump -h localhost -U postgres -d nodepass_panel --data-only > data_backup.sql
```

### 2. 停机时间

- 预计迁移时间：< 1 分钟（小型数据库）
- 建议在低峰期执行
- 如有大量数据，建议先在测试环境验证

### 3. 兼容性

- PostgreSQL 版本要求：>= 12
- 使用了 JSONB 类型，需要 PostgreSQL 9.4+
- 使用了 `gen_random_uuid()`，需要 PostgreSQL 13+ 或安装 `pgcrypto` 扩展

### 4. 权限要求

执行迁移的数据库用户需要以下权限：

```sql
-- 检查权限
SELECT
    grantee,
    privilege_type
FROM information_schema.role_table_grants
WHERE table_schema = 'public';

-- 如需授权
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO nodepass_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO nodepass_user;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO nodepass_user;
```

---

## 🔄 回滚计划

### 自动回滚

使用迁移脚本会自动创建备份：

```bash
./scripts/migrate-node-groups.sh rollback backup_20240307_120000.sql
```

### 手动回滚

```bash
# 1. 执行 down 迁移
psql -h localhost -U postgres -d nodepass_panel \
  -f migrations/000X_create_node_groups.down.sql

# 2. 恢复备份（如果需要）
psql -h localhost -U postgres -d nodepass_panel < backup.sql
```

### 部分回滚

如果只需要删除特定表：

```sql
-- 删除隧道表
DROP TABLE IF EXISTS tunnels;

-- 删除节点实例表
DROP TABLE IF EXISTS node_instances;

-- 删除节点组表
DROP TABLE IF EXISTS node_groups;
```

---

## 📈 监控和维护

### 1. 监控统计更新

```sql
-- 检查统计更新时间
SELECT
    ng.name,
    ngs.updated_at,
    EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - ngs.updated_at)) AS seconds_ago
FROM node_groups ng
JOIN node_group_stats ngs ON ng.id = ngs.group_id;
```

### 2. 清理孤立数据

```sql
-- 查找没有节点实例的节点组
SELECT ng.id, ng.name, ng.type
FROM node_groups ng
LEFT JOIN node_instances ni ON ng.id = ni.group_id
WHERE ni.id IS NULL;

-- 查找离线超过 7 天的节点
SELECT *
FROM node_instances
WHERE status = 'offline'
  AND last_heartbeat < CURRENT_TIMESTAMP - INTERVAL '7 days';
```

### 3. 性能监控

```sql
-- 查看表大小
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE tablename LIKE 'node_%' OR tablename = 'tunnels'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- 查看索引使用情况
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE tablename LIKE 'node_%' OR tablename = 'tunnels'
ORDER BY idx_scan DESC;
```

---

## ✅ 验收清单

迁移完成后，请验证以下项目：

- [ ] 所有表已创建（5 个表）
- [ ] 所有视图已创建（2 个视图）
- [ ] 所有函数已创建（4 个函数）
- [ ] 所有触发器已创建（4 个触发器）
- [ ] 所有索引已创建
- [ ] 测试数据插入成功
- [ ] 统计自动更新正常工作
- [ ] 查询性能符合预期
- [ ] 备份文件已保存

---

## 🆘 故障排除

### 问题 1：迁移执行失败

**错误**：`ERROR: relation "users" does not exist`

**解决**：确保基础表（users）已存在

```sql
-- 检查 users 表
SELECT * FROM users LIMIT 1;
```

### 问题 2：权限不足

**错误**：`ERROR: permission denied for table users`

**解决**：授予必要权限

```sql
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO current_user;
```

### 问题 3：触发器不工作

**错误**：统计数据未自动更新

**解决**：手动触发统计更新

```sql
-- 手动更新所有节点组统计
INSERT INTO node_group_stats (group_id, node_count, online_count, total_traffic, active_connections)
SELECT
    ng.id,
    COUNT(ni.id),
    COUNT(ni.id) FILTER (WHERE ni.status = 'online'),
    COALESCE(SUM((ni.traffic_stats->>'total_in')::BIGINT + (ni.traffic_stats->>'total_out')::BIGINT), 0),
    COALESCE(SUM((ni.traffic_stats->>'connections')::INTEGER), 0)
FROM node_groups ng
LEFT JOIN node_instances ni ON ng.id = ni.group_id
GROUP BY ng.id
ON CONFLICT (group_id) DO UPDATE SET
    node_count = EXCLUDED.node_count,
    online_count = EXCLUDED.online_count,
    total_traffic = EXCLUDED.total_traffic,
    active_connections = EXCLUDED.active_connections,
    updated_at = CURRENT_TIMESTAMP;
```

### 问题 4：JSONB 查询慢

**解决**：添加 GIN 索引

```sql
-- 为 config 字段添加 GIN 索引
CREATE INDEX idx_node_groups_config ON node_groups USING GIN (config);
CREATE INDEX idx_node_instances_system_info ON node_instances USING GIN (system_info);
```

---

## 📞 支持

如有问题，请：
1. 查看日志文件
2. 检查数据库连接
3. 验证权限设置
4. 查阅本文档的故障排除部分

---

**迁移准备完成！** 🎉

执行命令开始迁移：
```bash
cd backend
./scripts/migrate-node-groups.sh up
```
