# DDD 架构设计文档

## 概述

NodePass-Pro 后端采用领域驱动设计 (Domain-Driven Design, DDD) 架构，结合 CQRS 模式，实现高内聚低耦合的代码结构。

## 架构分层

### 1. Domain Layer (领域层)

领域层是业务逻辑的核心，包含业务规则和领域知识。

```
internal/domain/
├── announcement/
│   ├── entity.go          # 公告实体
│   ├── repository.go      # 仓储接口
│   └── errors.go          # 领域错误
├── systemconfig/
│   ├── entity.go          # 系统配置实体
│   ├── repository.go      # 仓储接口
│   └── errors.go          # 领域错误
├── nodeperformance/
│   ├── entity.go          # 性能指标实体
│   ├── repository.go      # 仓储接口
│   └── errors.go          # 领域错误
└── nodeautomation/
    ├── entity.go          # 自动化策略实体
    ├── repository.go      # 仓储接口
    └── errors.go          # 领域错误
```

#### 领域层职责

- **Entity (实体)**: 具有唯一标识的领域对象
- **Value Object (值对象)**: 无唯一标识的不可变对象
- **Repository Interface (仓储接口)**: 定义数据访问契约
- **Domain Service (领域服务)**: 跨实体的业务逻辑
- **Domain Error (领域错误)**: 业务规则违反的错误

#### 示例：公告实体

```go
// Announcement 公告实体
type Announcement struct {
    ID        uint
    Title     string
    Content   string
    Type      AnnouncementType
    IsEnabled bool
    StartTime *time.Time
    EndTime   *time.Time
    CreatedAt time.Time
    UpdatedAt time.Time
}

// IsActive 检查公告是否在有效期内
func (a *Announcement) IsActive() bool {
    if !a.IsEnabled {
        return false
    }
    now := time.Now()
    if a.StartTime != nil && now.Before(*a.StartTime) {
        return false
    }
    if a.EndTime != nil && now.After(*a.EndTime) {
        return false
    }
    return true
}
```

### 2. Application Layer (应用层)

应用层协调领域对象完成业务用例，采用 CQRS 模式分离命令和查询。

```
internal/application/
├── announcement/
│   ├── commands/
│   │   ├── create_announcement.go
│   │   ├── update_announcement.go
│   │   └── delete_announcement.go
│   └── queries/
│       ├── get_announcement.go
│       └── list_announcements.go
├── systemconfig/
│   ├── commands/
│   │   └── upsert_config.go
│   └── queries/
│       ├── get_config.go
│       └── list_configs.go
├── nodeperformance/
│   ├── commands/
│   │   └── record_metric.go
│   └── queries/
│       ├── get_metrics.go
│       └── get_stats.go
└── nodeautomation/
    ├── commands/
    │   ├── create_policy.go
    │   └── isolate_node.go
    └── queries/
        ├── get_policy.go
        └── list_actions.go
```

#### 应用层职责

- **Commands (命令)**: 修改系统状态的操作
- **Queries (查询)**: 读取系统状态的操作
- **Use Case (用例)**: 业务流程编排
- **DTO (数据传输对象)**: 跨层数据传输

#### 示例：创建公告命令

```go
// CreateAnnouncementCommand 创建公告命令
type CreateAnnouncementCommand struct {
    Title     string
    Content   string
    Type      string
    IsEnabled bool
    StartTime *time.Time
    EndTime   *time.Time
}

// CreateAnnouncementHandler 创建公告处理器
type CreateAnnouncementHandler struct {
    repo announcement.Repository
}

// Handle 处理命令
func (h *CreateAnnouncementHandler) Handle(ctx context.Context, cmd CreateAnnouncementCommand) (*announcement.Announcement, error) {
    // 1. 验证输入
    if err := validateCommand(cmd); err != nil {
        return nil, err
    }

    // 2. 创建领域对象
    ann := announcement.NewAnnouncement(cmd.Title, cmd.Content, cmd.Type)
    ann.IsEnabled = cmd.IsEnabled
    ann.StartTime = cmd.StartTime
    ann.EndTime = cmd.EndTime

    // 3. 持久化
    if err := h.repo.Create(ctx, ann); err != nil {
        return nil, err
    }

    return ann, nil
}
```

### 3. Infrastructure Layer (基础设施层)

基础设施层提供技术实现，支持领域层和应用层。

```
internal/infrastructure/
├── persistence/
│   └── postgres/
│       ├── announcement_repository.go
│       ├── systemconfig_repository.go
│       ├── nodeperformance_repository.go
│       └── nodeautomation_repository.go
└── cache/
    ├── announcement_cache.go
    └── config_cache.go
```

#### 基础设施层职责

- **Repository Implementation (仓储实现)**: 实现领域层定义的仓储接口
- **Cache Implementation (缓存实现)**: Redis 缓存封装
- **External Service (外部服务)**: 第三方服务集成
- **Message Queue (消息队列)**: 异步消息处理

#### 示例：PostgreSQL 仓储实现

```go
// AnnouncementRepository PostgreSQL 实现
type AnnouncementRepository struct {
    db *gorm.DB
}

// Create 创建公告
func (r *AnnouncementRepository) Create(ctx context.Context, ann *announcement.Announcement) error {
    return r.db.WithContext(ctx).Create(ann).Error
}

// FindByID 根据 ID 查找
func (r *AnnouncementRepository) FindByID(ctx context.Context, id uint) (*announcement.Announcement, error) {
    var ann announcement.Announcement
    err := r.db.WithContext(ctx).First(&ann, id).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, announcement.ErrAnnouncementNotFound
        }
        return nil, err
    }
    return &ann, nil
}

// ListEnabled 列出启用的公告
func (r *AnnouncementRepository) ListEnabled(ctx context.Context) ([]*announcement.Announcement, error) {
    var announcements []*announcement.Announcement
    now := time.Now()

    err := r.db.WithContext(ctx).
        Where("is_enabled = ?", true).
        Where("(start_time IS NULL OR start_time <= ?)", now).
        Where("(end_time IS NULL OR end_time >= ?)", now).
        Order("created_at DESC").
        Find(&announcements).Error

    return announcements, err
}
```

## 依赖关系

```
┌─────────────────────────────────────┐
│     Presentation Layer (API)        │
│         (Gin Handlers)              │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│      Application Layer              │
│   (Commands + Queries)              │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│        Domain Layer                 │
│  (Entities + Repositories)          │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│    Infrastructure Layer             │
│  (PostgreSQL + Redis)               │
└─────────────────────────────────────┘
```

**依赖规则**:
- 外层依赖内层
- 内层不依赖外层
- 领域层是核心，不依赖任何层

## 设计原则

### 1. SOLID 原则

- **S (Single Responsibility)**: 每个类只有一个职责
- **O (Open/Closed)**: 对扩展开放，对修改关闭
- **L (Liskov Substitution)**: 子类可以替换父类
- **I (Interface Segregation)**: 接口隔离，不依赖不需要的接口
- **D (Dependency Inversion)**: 依赖抽象而非具体实现

### 2. DDD 战术模式

- **Aggregate (聚合)**: 一组相关对象的集合
- **Entity (实体)**: 有唯一标识的对象
- **Value Object (值对象)**: 无标识的不可变对象
- **Repository (仓储)**: 聚合的持久化抽象
- **Domain Service (领域服务)**: 跨实体的业务逻辑

### 3. CQRS 模式

- **Command (命令)**: 修改状态，无返回值或返回简单结果
- **Query (查询)**: 读取状态，不修改数据
- **分离优势**: 读写优化、扩展性、清晰职责

## 最佳实践

### 1. 领域层

```go
// ✅ 好的实践：业务逻辑在实体中
func (a *Announcement) IsActive() bool {
    if !a.IsEnabled {
        return false
    }
    // 业务规则：检查时间范围
    now := time.Now()
    if a.StartTime != nil && now.Before(*a.StartTime) {
        return false
    }
    if a.EndTime != nil && now.After(*a.EndTime) {
        return false
    }
    return true
}

// ❌ 不好的实践：业务逻辑在应用层
func (h *Handler) IsAnnouncementActive(ann *Announcement) bool {
    // 业务逻辑不应该在应用层
}
```

### 2. 应用层

```go
// ✅ 好的实践：命令处理器单一职责
type CreateAnnouncementHandler struct {
    repo announcement.Repository
}

func (h *CreateAnnouncementHandler) Handle(ctx context.Context, cmd CreateAnnouncementCommand) error {
    // 只负责创建公告
}

// ❌ 不好的实践：处理器做太多事情
type AnnouncementHandler struct {
    repo announcement.Repository
}

func (h *AnnouncementHandler) HandleEverything(ctx context.Context, action string, data interface{}) error {
    // 违反单一职责原则
}
```

### 3. 基础设施层

```go
// ✅ 好的实践：实现领域层接口
type AnnouncementRepository struct {
    db *gorm.DB
}

func (r *AnnouncementRepository) Create(ctx context.Context, ann *announcement.Announcement) error {
    return r.db.WithContext(ctx).Create(ann).Error
}

// ❌ 不好的实践：直接暴露 GORM
func (r *AnnouncementRepository) GetDB() *gorm.DB {
    return r.db // 泄露实现细节
}
```

## 测试策略

### 1. 领域层测试

```go
func TestAnnouncement_IsActive(t *testing.T) {
    now := time.Now()
    future := now.Add(24 * time.Hour)
    past := now.Add(-24 * time.Hour)

    tests := []struct {
        name      string
        ann       *Announcement
        wantActive bool
    }{
        {
            name: "启用且在时间范围内",
            ann: &Announcement{
                IsEnabled: true,
                StartTime: &past,
                EndTime:   &future,
            },
            wantActive: true,
        },
        {
            name: "未启用",
            ann: &Announcement{
                IsEnabled: false,
            },
            wantActive: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := tt.ann.IsActive()
            assert.Equal(t, tt.wantActive, got)
        })
    }
}
```

### 2. 应用层测试

```go
func TestCreateAnnouncementHandler_Handle(t *testing.T) {
    mockRepo := &MockAnnouncementRepository{}
    handler := NewCreateAnnouncementHandler(mockRepo)

    cmd := CreateAnnouncementCommand{
        Title:   "测试公告",
        Content: "测试内容",
        Type:    "info",
    }

    ann, err := handler.Handle(context.Background(), cmd)

    assert.NoError(t, err)
    assert.NotNil(t, ann)
    assert.Equal(t, "测试公告", ann.Title)
}
```

### 3. 基础设施层测试

```go
func TestAnnouncementRepository_Create(t *testing.T) {
    db := setupTestDB(t)
    repo := NewAnnouncementRepository(db)

    ann := &Announcement{
        Title:   "测试",
        Content: "内容",
        Type:    "info",
    }

    err := repo.Create(context.Background(), ann)

    assert.NoError(t, err)
    assert.NotZero(t, ann.ID)
}
```

## 总结

DDD 架构为 NodePass-Pro 提供了：

1. **清晰的分层结构** - 职责明确，易于理解
2. **高内聚低耦合** - 模块独立，易于维护
3. **业务逻辑集中** - 领域层包含核心业务规则
4. **易于测试** - 每层可独立测试
5. **易于扩展** - 符合开闭原则，扩展不修改
