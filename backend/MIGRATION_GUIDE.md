# 旧代码迁移指南

## 概述

本文档提供从旧 MVC 架构迁移到新 DDD 架构的详细指南。

---

## 已重构模块

### 1. User 模块

#### 旧代码位置
- Model: `internal/models/user.go`
- Service: `internal/services/user_admin_service.go`
- Handler: `internal/handlers/user_admin_handler.go`

#### 新代码位置
- Domain: `internal/domain/user/entity.go`
- Application Commands: `internal/application/user/commands/create_user.go`
- Application Queries: `internal/application/user/queries/get_user.go`
- Repository: `internal/infrastructure/persistence/postgres/user_repository.go`
- Cache: `internal/infrastructure/cache/user_cache.go`

#### 迁移示例

**旧代码:**
```go
// 使用 Service
userService := services.NewUserAdminService(db)
user, err := userService.GetUserByID(userID)
```

**新代码:**
```go
// 使用 Container 和 Application 层
container := container.NewContainer(db, redisClient)
query := &userQueries.GetUserQuery{UserID: userID}
result, err := container.GetUserHandler.Handle(ctx, query)
```

---

### 2. Auth 模块

#### 旧代码位置
- Model: `internal/models/user.go` (部分)
- Service: `internal/services/auth_service.go`
- Handler: `internal/handlers/auth_handler.go`

#### 新代码位置
- Domain: `internal/domain/auth/entity.go`
- Application Commands:
  - `internal/application/auth/commands/login.go`
  - `internal/application/auth/commands/register.go`
  - `internal/application/auth/commands/change_password.go`
  - `internal/application/auth/commands/refresh_token.go`
- Application Queries: `internal/application/auth/queries/get_user.go`
- Repository: `internal/infrastructure/persistence/postgres/auth/auth_repository.go`
- Cache: `internal/infrastructure/cache/auth_cache.go`

#### 迁移示例

**旧代码:**
```go
// 登录
authService := services.NewAuthService(db)
result, err := authService.Login(ctx, &services.LoginRequest{
    Email:    "user@example.com",
    Password: "password",
})
```

**新代码:**
```go
// 登录
container := container.NewContainer(db, redisClient)
cmd := &authCommands.LoginCommand{
    Email:    "user@example.com",
    Password: "password",
}
result, err := container.LoginHandler.Handle(ctx, cmd)
```

---

### 3. VIP 模块

#### 旧代码位置
- Model: `internal/models/vip_level.go`
- Service: `internal/services/vip_service.go`
- Handler: `internal/handlers/vip_handler.go`

#### 新代码位置
- Domain: `internal/domain/vip/entity.go`
- Application Commands:
  - `internal/application/vip/commands/create_level.go`
  - `internal/application/vip/commands/upgrade_user.go`
- Application Queries:
  - `internal/application/vip/queries/list_levels.go`
  - `internal/application/vip/queries/get_my_level.go`
- Repository: `internal/infrastructure/persistence/postgres/vip/vip_repository.go`
- Cache: `internal/infrastructure/cache/vip_cache.go`

#### 迁移示例

**旧代码:**
```go
// 获取 VIP 等级列表
vipService := services.NewVIPService(db)
levels, err := vipService.ListLevels()
```

**新代码:**
```go
// 获取 VIP 等级列表
container := container.NewContainer(db, redisClient)
query := &vipQueries.ListLevelsQuery{}
result, err := container.ListLevelsHandler.Handle(ctx, query)
```

---

### 4. Node 模块

#### 旧代码位置
- Model: `internal/models/node.go`
- Service: `internal/services/node_instance_service.go`

#### 新代码位置
- Domain: `internal/domain/node/entity.go`
- Application Commands: `internal/application/node/commands/heartbeat.go`
- Application Queries:
  - `internal/application/node/queries/get_node.go`
  - `internal/application/node/queries/list_nodes.go`
  - `internal/application/node/queries/get_online_nodes.go`
- Repository: `internal/infrastructure/persistence/postgres/node_repository.go`
- Cache: `internal/infrastructure/cache/node_cache.go`

---

### 5. Traffic 模块

#### 旧代码位置
- Model: `internal/models/traffic_record.go`
- Service: `internal/services/traffic_service.go`

#### 新代码位置
- Domain: `internal/domain/traffic/entity.go`
- Application Commands:
  - `internal/application/traffic/commands/record_traffic.go`
  - `internal/application/traffic/commands/flush_traffic.go`
- Application Queries:
  - `internal/application/traffic/queries/get_user_traffic.go`
  - `internal/application/traffic/queries/get_tunnel_traffic.go`
- Repository: `internal/infrastructure/persistence/postgres/traffic_repository.go`
- Cache: `internal/infrastructure/cache/traffic_counter.go`

---

## 通用迁移模式

### 1. 依赖注入容器

**旧代码:**
```go
// 直接创建 Service
userService := services.NewUserAdminService(db)
authService := services.NewAuthService(db)
```

**新代码:**
```go
// 使用容器统一管理依赖
container := container.NewContainer(db, redisClient)

// 所有 Handler 都通过容器获取
loginHandler := container.LoginHandler
getUserHandler := container.GetUserHandler
```

### 2. HTTP Handler 集成

**旧代码:**
```go
// Gin Handler
func (h *AuthHandler) Login(c *gin.Context) {
    var req services.LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    result, err := h.authService.Login(c.Request.Context(), &req)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, result)
}
```

**新代码:**
```go
// Gin Handler 调用 Application 层
func LoginHandler(container *container.Container) gin.HandlerFunc {
    return func(c *gin.Context) {
        var cmd authCommands.LoginCommand
        if err := c.ShouldBindJSON(&cmd); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        result, err := container.LoginHandler.Handle(c.Request.Context(), &cmd)
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }

        c.JSON(200, result)
    }
}
```

### 3. 缓存使用

**旧代码:**
```go
// 直接操作数据库
user, err := db.First(&models.User{}, userID).Error
```

**新代码:**
```go
// 优先使用缓存
query := &userQueries.GetUserQuery{UserID: userID}
result, err := container.GetUserHandler.Handle(ctx, query)
// Handler 内部会先查缓存，缓存未命中再查数据库
```

---

## 迁移检查清单

### 代码迁移
- [ ] 识别旧代码中的业务逻辑
- [ ] 将业务逻辑迁移到 Domain 层
- [ ] 创建 Application 层的 Commands 和 Queries
- [ ] 实现 Repository 接口
- [ ] 添加 Cache 层
- [ ] 更新 HTTP Handler 调用新代码
- [ ] 编写单元测试

### 测试验证
- [ ] 单元测试通过
- [ ] 集成测试通过
- [ ] 性能测试通过
- [ ] 缓存命中率符合预期

### 清理工作
- [ ] 标记旧代码为 @Deprecated
- [ ] 添加迁移指南注释
- [ ] 确认无新代码依赖旧代码
- [ ] 删除旧代码（最后一步）

---

## 常见问题

### Q1: 如何处理复杂的业务逻辑？

**A:** 将复杂逻辑拆分到多个 Domain Service 或 Application Service 中，保持单一职责原则。

### Q2: 缓存何时失效？

**A:**
- 写操作（Create/Update/Delete）时同步更新缓存
- 使用 TTL 机制自动过期
- 提供手动失效接口

### Q3: 如何处理事务？

**A:** 在 Application 层的 Command Handler 中使用数据库事务：
```go
func (h *CreateUserHandler) Handle(ctx context.Context, cmd *CreateUserCommand) error {
    return h.db.Transaction(func(tx *gorm.DB) error {
        // 业务逻辑
        return nil
    })
}
```

### Q4: 旧代码何时删除？

**A:**
1. 确认所有调用方已迁移到新代码
2. 在生产环境运行至少 2 周无问题
3. 备份旧代码到 Git 分支
4. 删除旧代码

---

## 联系方式

如有疑问，请查看：
- 架构文档: `REDIS_POSTGRES_ARCHITECTURE.md`
- 重构指南: `REFACTORING_GUIDE.md`
- 重构路线图: `REFACTORING_ROADMAP.md`

---

**最后更新**: 2026-03-11
**版本**: v1.0
