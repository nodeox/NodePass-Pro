# 系统配置模块文档

## 概述

系统配置模块提供灵活的键值对配置管理，支持动态配置更新、批量操作和缓存优化。

## 功能特性

- ✅ 键值对配置存储
- ✅ Upsert 操作（创建或更新）
- ✅ 批量更新配置
- ✅ 配置描述说明
- ✅ Redis 缓存支持
- ✅ 配置导出/导入

## 架构设计

### 领域层

```go
// SystemConfig 系统配置实体
type SystemConfig struct {
    ID          uint
    Key         string   // 配置键
    Value       *string  // 配置值（可为空）
    Description *string  // 配置描述（可选）
    UpdatedAt   time.Time
}

// GetValueOrDefault 获取值或默认值
func (c *SystemConfig) GetValueOrDefault(defaultValue string) string {
    if c.Value == nil {
        return defaultValue
    }
    return *c.Value
}

// SetValue 设置值
func (c *SystemConfig) SetValue(value string) {
    c.Value = &value
}
```

### 业务规则

#### 1. 配置键规范

- 使用点号分隔的命名空间：`app.name`、`mail.smtp.host`
- 全小写字母，使用下划线或点号分隔
- 不超过 100 字符

#### 2. 配置值类型

虽然存储为字符串，但支持多种类型：

```go
// 字符串
config.SetValue("NodePass Pro")

// 数字
config.SetValue("8080")

// 布尔值
config.SetValue("true")

// JSON
config.SetValue(`{"host":"smtp.example.com","port":587}`)
```

### 应用层

#### Commands (命令)

**UpsertConfigCommand** - 创建或更新配置
```go
type UpsertConfigCommand struct {
    Key         string
    Value       *string
    Description *string
}
```

**BatchUpsertConfigsCommand** - 批量更新配置
```go
type BatchUpsertConfigsCommand struct {
    Configs []ConfigItem
}

type ConfigItem struct {
    Key         string
    Value       *string
    Description *string
}
```

**DeleteConfigCommand** - 删除配置
```go
type DeleteConfigCommand struct {
    Key string
}
```

#### Queries (查询)

**GetConfigQuery** - 获取单个配置
```go
type GetConfigQuery struct {
    Key string
}
```

**ListConfigsQuery** - 列出所有配置
```go
type ListConfigsQuery struct {
    Prefix string  // 按前缀筛选（可选）
}
```

**GetAllAsMapQuery** - 获取所有配置为 Map
```go
type GetAllAsMapQuery struct{}

// 返回 map[string]string
```

### 基础设施层

#### PostgreSQL 仓储

```go
type SystemConfigRepository struct {
    db *gorm.DB
}

// Upsert 创建或更新配置
func (r *SystemConfigRepository) Upsert(ctx context.Context, config *SystemConfig) error {
    return r.db.WithContext(ctx).
        Clauses(clause.OnConflict{
            Columns:   []clause.Column{{Name: "key"}},
            DoUpdates: clause.AssignmentColumns([]string{"value", "description", "updated_at"}),
        }).
        Create(config).Error
}

// GetAllAsMap 获取所有配置为 Map
func (r *SystemConfigRepository) GetAllAsMap(ctx context.Context) (map[string]string, error) {
    var configs []*SystemConfig
    if err := r.db.WithContext(ctx).Find(&configs).Error; err != nil {
        return nil, err
    }

    result := make(map[string]string)
    for _, config := range configs {
        if config.Value != nil {
            result[config.Key] = *config.Value
        }
    }
    return result, nil
}
```

#### Redis 缓存

```go
type ConfigCache struct {
    redis *redis.Client
    ttl   time.Duration
}

// Get 获取配置
func (c *ConfigCache) Get(ctx context.Context, key string) (*string, error) {
    val, err := c.redis.Get(ctx, "config:"+key).Result()
    if err == redis.Nil {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &val, nil
}

// Set 设置配置
func (c *ConfigCache) Set(ctx context.Context, key string, value string) error {
    return c.redis.Set(ctx, "config:"+key, value, c.ttl).Err()
}

// Delete 删除配置
func (c *ConfigCache) Delete(ctx context.Context, key string) error {
    return c.redis.Del(ctx, "config:"+key).Err()
}
```

## 使用示例

### 1. 创建或更新配置

```go
// 单个配置
value := "NodePass Pro"
desc := "应用名称"

cmd := commands.UpsertConfigCommand{
    Key:         "app.name",
    Value:       &value,
    Description: &desc,
}

handler := commands.NewUpsertConfigHandler(repo, cache)
config, err := handler.Handle(ctx, cmd)
```

### 2. 批量更新配置

```go
// 批量更新
smtpHost := "smtp.example.com"
smtpPort := "587"

cmd := commands.BatchUpsertConfigsCommand{
    Configs: []commands.ConfigItem{
        {
            Key:   "mail.smtp.host",
            Value: &smtpHost,
        },
        {
            Key:   "mail.smtp.port",
            Value: &smtpPort,
        },
    },
}

handler := commands.NewBatchUpsertConfigsHandler(repo, cache)
err := handler.Handle(ctx, cmd)
```

### 3. 查询配置

```go
// 获取单个配置
query := queries.GetConfigQuery{
    Key: "app.name",
}

handler := queries.NewGetConfigHandler(repo, cache)
config, err := handler.Handle(ctx, query)

// 使用默认值
appName := config.GetValueOrDefault("Default App")
```

### 4. 获取所有配置

```go
// 获取所有配置为 Map
query := queries.GetAllAsMapQuery{}

handler := queries.NewGetAllAsMapHandler(repo, cache)
configMap, err := handler.Handle(ctx, query)

// 使用配置
appName := configMap["app.name"]
smtpHost := configMap["mail.smtp.host"]
```

## API 接口

详见 [系统配置 API 文档](../api/SYSTEM_CONFIG_API.md)

## 数据库表结构

```sql
CREATE TABLE system_configs (
    id SERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,
    value TEXT,
    description TEXT,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE UNIQUE INDEX idx_system_configs_key ON system_configs(key);
CREATE INDEX idx_system_configs_updated ON system_configs(updated_at DESC);
```

## 配置分类

### 1. 应用配置

```
app.name              应用名称
app.version           应用版本
app.env               运行环境（dev/prod）
app.debug             调试模式
```

### 2. 邮件配置

```
mail.smtp.host        SMTP 服务器
mail.smtp.port        SMTP 端口
mail.smtp.username    SMTP 用户名
mail.smtp.password    SMTP 密码
mail.from.address     发件人地址
mail.from.name        发件人名称
```

### 3. 系统限制

```
system.max_users              最大用户数
system.max_nodes_per_user     每用户最大节点数
system.max_tunnels_per_user   每用户最大隧道数
system.traffic_limit_gb       流量限制（GB）
```

### 4. 功能开关

```
feature.registration_enabled  注册功能开关
feature.invite_only           仅邀请注册
feature.email_verification    邮箱验证开关
feature.two_factor_auth       双因素认证开关
```

### 5. 性能配置

```
performance.cache_ttl         缓存 TTL（秒）
performance.heartbeat_interval 心跳间隔（秒）
performance.batch_size        批量操作大小
```

## 配置管理最佳实践

### 1. 配置命名规范

```go
// ✅ 好的命名
"app.name"
"mail.smtp.host"
"feature.registration_enabled"

// ❌ 不好的命名
"AppName"
"SMTP_HOST"
"reg_enabled"
```

### 2. 配置分组

使用命名空间前缀对配置进行分组：

```
app.*           应用相关
mail.*          邮件相关
feature.*       功能开关
system.*        系统限制
performance.*   性能配置
```

### 3. 配置验证

在应用层添加配置验证：

```go
func (h *UpsertConfigHandler) Handle(ctx context.Context, cmd UpsertConfigCommand) error {
    // 验证配置键
    if !isValidConfigKey(cmd.Key) {
        return ErrInvalidConfigKey
    }

    // 验证配置值
    if err := validateConfigValue(cmd.Key, cmd.Value); err != nil {
        return err
    }

    // ... 执行 Upsert
}

func validateConfigValue(key string, value *string) error {
    if value == nil {
        return nil
    }

    switch key {
    case "mail.smtp.port":
        // 验证端口号
        port, err := strconv.Atoi(*value)
        if err != nil || port < 1 || port > 65535 {
            return ErrInvalidPort
        }
    case "feature.registration_enabled":
        // 验证布尔值
        if *value != "true" && *value != "false" {
            return ErrInvalidBoolean
        }
    }

    return nil
}
```

### 4. 配置缓存策略

```go
// Cache-Aside 模式
func (h *GetConfigHandler) Handle(ctx context.Context, query GetConfigQuery) (*SystemConfig, error) {
    // 1. 查缓存
    if cached, err := h.cache.Get(ctx, query.Key); err == nil && cached != nil {
        return &SystemConfig{Key: query.Key, Value: cached}, nil
    }

    // 2. 查数据库
    config, err := h.repo.FindByKey(ctx, query.Key)
    if err != nil {
        return nil, err
    }

    // 3. 写缓存
    if config.Value != nil {
        h.cache.Set(ctx, query.Key, *config.Value)
    }

    return config, nil
}
```

## 扩展功能

### 1. 配置历史记录

跟踪配置变更历史：

```sql
CREATE TABLE system_config_history (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(100) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_by INT,
    changed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### 2. 配置加密

敏感配置加密存储：

```go
func (c *SystemConfig) SetEncryptedValue(value string, encryptor Encryptor) error {
    encrypted, err := encryptor.Encrypt(value)
    if err != nil {
        return err
    }
    c.Value = &encrypted
    return nil
}

func (c *SystemConfig) GetDecryptedValue(encryptor Encryptor) (string, error) {
    if c.Value == nil {
        return "", nil
    }
    return encryptor.Decrypt(*c.Value)
}
```

### 3. 配置导出/导入

```go
// 导出配置为 JSON
func ExportConfigs(ctx context.Context, repo Repository) ([]byte, error) {
    configs, err := repo.FindAll(ctx)
    if err != nil {
        return nil, err
    }
    return json.Marshal(configs)
}

// 从 JSON 导入配置
func ImportConfigs(ctx context.Context, repo Repository, data []byte) error {
    var configs []*SystemConfig
    if err := json.Unmarshal(data, &configs); err != nil {
        return err
    }

    for _, config := range configs {
        if err := repo.Upsert(ctx, config); err != nil {
            return err
        }
    }
    return nil
}
```

## 测试

```go
func TestSystemConfig_GetValueOrDefault(t *testing.T) {
    tests := []struct {
        name         string
        config       *SystemConfig
        defaultValue string
        want         string
    }{
        {
            name: "有值时返回实际值",
            config: &SystemConfig{
                Key:   "app.name",
                Value: stringPtr("NodePass Pro"),
            },
            defaultValue: "Default",
            want:         "NodePass Pro",
        },
        {
            name: "无值时返回默认值",
            config: &SystemConfig{
                Key:   "app.name",
                Value: nil,
            },
            defaultValue: "Default",
            want:         "Default",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := tt.config.GetValueOrDefault(tt.defaultValue)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

## 常见问题

### Q: 如何实现配置热更新？

A: 使用 Redis Pub/Sub 或定时轮询：

```go
// 订阅配置变更
func (s *ConfigService) WatchConfigChanges(ctx context.Context) {
    pubsub := s.redis.Subscribe(ctx, "config:changed")
    defer pubsub.Close()

    for msg := range pubsub.Channel() {
        key := msg.Payload
        // 重新加载配置
        s.reloadConfig(ctx, key)
    }
}
```

### Q: 如何实现配置权限控制？

A: 添加配置级别和权限检查：

```go
type SystemConfig struct {
    // ... 其他字段
    Level string  // public, protected, private
}

func (h *GetConfigHandler) Handle(ctx context.Context, query GetConfigQuery) error {
    config, err := h.repo.FindByKey(ctx, query.Key)
    if err != nil {
        return err
    }

    // 检查权限
    if config.Level == "private" && !hasAdminRole(ctx) {
        return ErrUnauthorized
    }

    return config, nil
}
```

## 相关文档

- [系统配置 API 文档](../api/SYSTEM_CONFIG_API.md)
- [DDD 架构文档](../architecture/DDD_ARCHITECTURE.md)
- [缓存策略文档](../architecture/CACHE_STRATEGY.md)
