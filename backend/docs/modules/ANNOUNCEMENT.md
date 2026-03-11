# 公告模块文档

## 概述

公告模块负责系统公告的发布和管理，支持时间范围控制、多种公告类型和启用/禁用机制。

## 功能特性

- ✅ 公告创建、更新、删除
- ✅ 时间范围控制（开始时间、结束时间）
- ✅ 多种公告类型（info、warning、error、success）
- ✅ 启用/禁用机制
- ✅ 自动过滤过期公告
- ✅ WebSocket 实时推送（可选）

## 架构设计

### 领域层

```go
// Announcement 公告实体
type Announcement struct {
    ID        uint
    Title     string              // 公告标题
    Content   string              // 公告内容
    Type      AnnouncementType    // 公告类型
    IsEnabled bool                // 是否启用
    StartTime *time.Time          // 开始时间（可选）
    EndTime   *time.Time          // 结束时间（可选）
    CreatedAt time.Time
    UpdatedAt time.Time
}

// AnnouncementType 公告类型
type AnnouncementType string

const (
    AnnouncementTypeInfo    AnnouncementType = "info"
    AnnouncementTypeWarning AnnouncementType = "warning"
    AnnouncementTypeError   AnnouncementType = "error"
    AnnouncementTypeSuccess AnnouncementType = "success"
)
```

### 业务规则

#### 1. 时间范围控制

公告的有效性由以下规则决定：

```go
func (a *Announcement) IsActive() bool {
    // 1. 必须启用
    if !a.IsEnabled {
        return false
    }

    now := time.Now()

    // 2. 检查开始时间
    if a.StartTime != nil && now.Before(*a.StartTime) {
        return false
    }

    // 3. 检查结束时间
    if a.EndTime != nil && now.After(*a.EndTime) {
        return false
    }

    return true
}
```

#### 2. 验证规则

- **标题**: 1-200 字符
- **内容**: 1-5000 字符
- **类型**: 必须是 info、warning、error、success 之一
- **时间范围**: 如果同时设置开始和结束时间，结束时间必须晚于开始时间

### 应用层

#### Commands (命令)

**CreateAnnouncementCommand** - 创建公告
```go
type CreateAnnouncementCommand struct {
    Title     string
    Content   string
    Type      string
    IsEnabled bool
    StartTime *time.Time
    EndTime   *time.Time
}
```

**UpdateAnnouncementCommand** - 更新公告
```go
type UpdateAnnouncementCommand struct {
    ID        uint
    Title     *string
    Content   *string
    Type      *string
    IsEnabled *bool
    StartTime *time.Time
    EndTime   *time.Time
}
```

**DeleteAnnouncementCommand** - 删除公告
```go
type DeleteAnnouncementCommand struct {
    ID uint
}
```

#### Queries (查询)

**GetAnnouncementQuery** - 获取单个公告
```go
type GetAnnouncementQuery struct {
    ID uint
}
```

**ListAnnouncementsQuery** - 列出公告
```go
type ListAnnouncementsQuery struct {
    OnlyEnabled bool  // 只返回启用的公告
    OnlyActive  bool  // 只返回有效期内的公告
}
```

### 基础设施层

#### PostgreSQL 仓储

```go
type AnnouncementRepository struct {
    db *gorm.DB
}

// ListEnabled 列出启用且在有效期内的公告
func (r *AnnouncementRepository) ListEnabled(ctx context.Context) ([]*Announcement, error) {
    var announcements []*Announcement
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

## 使用示例

### 1. 创建公告

```go
// 创建一个立即生效的公告
cmd := commands.CreateAnnouncementCommand{
    Title:     "系统维护通知",
    Content:   "系统将于今晚 22:00-24:00 进行维护",
    Type:      "warning",
    IsEnabled: true,
}

handler := commands.NewCreateAnnouncementHandler(repo)
announcement, err := handler.Handle(ctx, cmd)
```

### 2. 创建定时公告

```go
// 创建一个定时生效的公告
startTime := time.Now().Add(24 * time.Hour)
endTime := startTime.Add(7 * 24 * time.Hour)

cmd := commands.CreateAnnouncementCommand{
    Title:     "新功能上线",
    Content:   "新版本将于下周上线",
    Type:      "info",
    IsEnabled: true,
    StartTime: &startTime,
    EndTime:   &endTime,
}

announcement, err := handler.Handle(ctx, cmd)
```

### 3. 更新公告

```go
// 禁用公告
disabled := false
cmd := commands.UpdateAnnouncementCommand{
    ID:        1,
    IsEnabled: &disabled,
}

handler := commands.NewUpdateAnnouncementHandler(repo)
err := handler.Handle(ctx, cmd)
```

### 4. 查询有效公告

```go
// 查询所有有效的公告
query := queries.ListAnnouncementsQuery{
    OnlyEnabled: true,
    OnlyActive:  true,
}

handler := queries.NewListAnnouncementsHandler(repo)
announcements, err := handler.Handle(ctx, query)
```

## API 接口

详见 [公告 API 文档](../api/ANNOUNCEMENT_API.md)

## 数据库表结构

```sql
CREATE TABLE announcements (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    type VARCHAR(20) NOT NULL,
    is_enabled BOOLEAN DEFAULT true,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_announcements_enabled ON announcements(is_enabled);
CREATE INDEX idx_announcements_time ON announcements(start_time, end_time);
CREATE INDEX idx_announcements_created ON announcements(created_at DESC);
```

## 扩展功能

### 1. WebSocket 实时推送

当创建或更新公告时，可以通过 WebSocket 实时推送给在线用户：

```go
// 创建公告后推送
announcement, err := handler.Handle(ctx, cmd)
if err == nil {
    websocket.Broadcast("announcement:created", announcement)
}
```

### 2. 公告阅读状态

可以扩展添加用户阅读状态跟踪：

```sql
CREATE TABLE announcement_reads (
    id SERIAL PRIMARY KEY,
    announcement_id INT NOT NULL,
    user_id INT NOT NULL,
    read_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(announcement_id, user_id)
);
```

### 3. 公告优先级

可以添加优先级字段，控制公告显示顺序：

```go
type Announcement struct {
    // ... 其他字段
    Priority int  // 优先级：1-5，数字越大优先级越高
}
```

## 测试

### 单元测试

```go
func TestAnnouncement_IsActive(t *testing.T) {
    now := time.Now()
    future := now.Add(24 * time.Hour)
    past := now.Add(-24 * time.Hour)

    tests := []struct {
        name       string
        ann        *Announcement
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
                StartTime: &past,
                EndTime:   &future,
            },
            wantActive: false,
        },
        {
            name: "未到开始时间",
            ann: &Announcement{
                IsEnabled: true,
                StartTime: &future,
            },
            wantActive: false,
        },
        {
            name: "已过结束时间",
            ann: &Announcement{
                IsEnabled: true,
                EndTime:   &past,
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

## 最佳实践

### 1. 时间范围设置

- **临时公告**: 设置开始和结束时间
- **长期公告**: 只设置开始时间，不设置结束时间
- **定时公告**: 提前创建，设置未来的开始时间

### 2. 公告类型选择

- **info**: 一般信息通知
- **warning**: 警告信息（如维护通知）
- **error**: 错误或紧急通知
- **success**: 成功或好消息

### 3. 内容编写

- 标题简洁明了，不超过 50 字
- 内容结构清晰，使用 Markdown 格式
- 重要信息使用加粗或高亮
- 提供必要的链接和联系方式

## 常见问题

### Q: 如何实现公告置顶？

A: 可以添加 `is_pinned` 字段，查询时优先返回置顶公告：

```go
ORDER BY is_pinned DESC, created_at DESC
```

### Q: 如何实现公告分类？

A: 可以添加 `category` 字段，支持按分类筛选：

```go
type Announcement struct {
    // ... 其他字段
    Category string  // 如：system、feature、maintenance
}
```

### Q: 如何防止公告被频繁修改？

A: 可以添加版本控制或修改历史记录：

```sql
CREATE TABLE announcement_history (
    id SERIAL PRIMARY KEY,
    announcement_id INT NOT NULL,
    title VARCHAR(200),
    content TEXT,
    modified_by INT,
    modified_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## 相关文档

- [公告 API 文档](../api/ANNOUNCEMENT_API.md)
- [DDD 架构文档](../architecture/DDD_ARCHITECTURE.md)
- [开发指南](../guides/DEVELOPMENT_GUIDE.md)
