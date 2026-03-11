# 集成指南：如何使用新架构

## 概述

本指南展示如何将新的 DDD 架构集成到现有的 Gin 路由中。

## 架构层次

```
HTTP Request
    ↓
Handler (接口层)
    ↓
Command/Query Handler (应用层)
    ↓
Repository + Cache (基础设施层)
    ↓
PostgreSQL + Redis
```

## 依赖注入容器

### 1. 创建容器

```go
// internal/infrastructure/container/container.go
package container

import (
	"nodepass-pro/backend/internal/application/user/commands"
	"nodepass-pro/backend/internal/application/user/queries"
	nodeCommands "nodepass-pro/backend/internal/application/node/commands"
	nodeQueries "nodepass-pro/backend/internal/application/node/queries"
	"nodepass-pro/backend/internal/domain/user"
	"nodepass-pro/backend/internal/domain/node"
	"nodepass-pro/backend/internal/infrastructure/cache"
	"nodepass-pro/backend/internal/infrastructure/persistence/postgres"
	
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Container 依赖注入容器
type Container struct {
	// 数据库
	DB          *gorm.DB
	RedisClient *redis.Client
	
	// 仓储
	UserRepo     user.Repository
	NodeRepo     node.InstanceRepository
	
	// 缓存
	UserCache       *cache.UserCache
	NodeCache       *cache.NodeCache
	TrafficCounter  *cache.TrafficCounter
	HeartbeatBuffer *cache.HeartbeatBuffer
	
	// 用户模块处理器
	CreateUserHandler *commands.CreateUserHandler
	GetUserHandler    *queries.GetUserHandler
	
	// 节点模块处理器
	HeartbeatHandler     *nodeCommands.HeartbeatHandler
	GetNodeHandler       *nodeQueries.GetNodeHandler
	ListNodesHandler     *nodeQueries.ListNodesHandler
	GetOnlineNodesHandler *nodeQueries.GetOnlineNodesHandler
}

// NewContainer 创建容器
func NewContainer(db *gorm.DB, redisClient *redis.Client) *Container {
	// 初始化仓储
	userRepo := postgres.NewUserRepository(db)
	nodeRepo := postgres.NewNodeInstanceRepository(db)
	
	// 初始化缓存
	userCache := cache.NewUserCache(redisClient)
	nodeCache := cache.NewNodeCache(redisClient)
	trafficCounter := cache.NewTrafficCounter(redisClient)
	heartbeatBuffer := cache.NewHeartbeatBuffer(redisClient)
	
	// 初始化用户模块处理器
	createUserHandler := commands.NewCreateUserHandler(userRepo, userCache)
	getUserHandler := queries.NewGetUserHandler(userRepo, userCache)
	
	// 初始化节点模块处理器
	heartbeatHandler := nodeCommands.NewHeartbeatHandler(nodeRepo, nodeCache, heartbeatBuffer)
	getNodeHandler := nodeQueries.NewGetNodeHandler(nodeRepo, nodeCache)
	listNodesHandler := nodeQueries.NewListNodesHandler(nodeRepo, nodeCache)
	getOnlineNodesHandler := nodeQueries.NewGetOnlineNodesHandler(nodeRepo, nodeCache)
	
	return &Container{
		DB:                    db,
		RedisClient:           redisClient,
		UserRepo:              userRepo,
		NodeRepo:              nodeRepo,
		UserCache:             userCache,
		NodeCache:             nodeCache,
		TrafficCounter:        trafficCounter,
		HeartbeatBuffer:       heartbeatBuffer,
		CreateUserHandler:     createUserHandler,
		GetUserHandler:        getUserHandler,
		HeartbeatHandler:      heartbeatHandler,
		GetNodeHandler:        getNodeHandler,
		ListNodesHandler:      listNodesHandler,
		GetOnlineNodesHandler: getOnlineNodesHandler,
	}
}
```

### 2. 在 main.go 中初始化

```go
// cmd/server/main.go
func main() {
	// ... 初始化数据库和 Redis ...
	
	// 创建依赖注入容器
	container := container.NewContainer(database.DB, redisClient)
	
	// 设置路由
	router := setupRouter(container)
	
	// ... 启动服务器 ...
}
```

## Handler 集成示例

### 1. 用户模块 Handler

```go
// internal/interfaces/http/handlers/user_handler.go
package handlers

import (
	"nodepass-pro/backend/internal/application/user/commands"
	"nodepass-pro/backend/internal/application/user/queries"
	"nodepass-pro/backend/internal/utils"
	
	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	createUserHandler *commands.CreateUserHandler
	getUserHandler    *queries.GetUserHandler
}

// NewUserHandler 创建用户处理器
func NewUserHandler(
	createUserHandler *commands.CreateUserHandler,
	getUserHandler *queries.GetUserHandler,
) *UserHandler {
	return &UserHandler{
		createUserHandler: createUserHandler,
		getUserHandler:    getUserHandler,
	}
}

// Create 创建用户
func (h *UserHandler) Create(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Role     string `json:"role"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err)
		return
	}
	
	// 执行命令
	cmd := commands.CreateUserCommand{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	}
	
	result, err := h.createUserHandler.Handle(c.Request.Context(), cmd)
	if err != nil {
		utils.Error(c, err)
		return
	}
	
	utils.Success(c, gin.H{
		"id":       result.User.ID,
		"username": result.User.Username,
		"email":    result.User.Email,
		"role":     result.User.Role,
	})
}

// Get 获取用户
func (h *UserHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id") // 从中间件获取
	
	// 执行查询
	query := queries.GetUserQuery{UserID: userID}
	result, err := h.getUserHandler.Handle(c.Request.Context(), query)
	if err != nil {
		utils.Error(c, err)
		return
	}
	
	utils.Success(c, gin.H{
		"id":       result.User.ID,
		"username": result.User.Username,
		"email":    result.User.Email,
		"role":     result.User.Role,
		"status":   result.User.Status,
	})
}
```

### 2. 节点模块 Handler

```go
// internal/interfaces/http/handlers/node_handler.go
package handlers

import (
	"nodepass-pro/backend/internal/application/node/commands"
	"nodepass-pro/backend/internal/application/node/queries"
	"nodepass-pro/backend/internal/utils"
	
	"github.com/gin-gonic/gin"
)

// NodeHandler 节点处理器
type NodeHandler struct {
	heartbeatHandler      *commands.HeartbeatHandler
	getNodeHandler        *queries.GetNodeHandler
	listNodesHandler      *queries.ListNodesHandler
	getOnlineNodesHandler *queries.GetOnlineNodesHandler
}

// NewNodeHandler 创建节点处理器
func NewNodeHandler(
	heartbeatHandler *commands.HeartbeatHandler,
	getNodeHandler *queries.GetNodeHandler,
	listNodesHandler *queries.ListNodesHandler,
	getOnlineNodesHandler *queries.GetOnlineNodesHandler,
) *NodeHandler {
	return &NodeHandler{
		heartbeatHandler:      heartbeatHandler,
		getNodeHandler:        getNodeHandler,
		listNodesHandler:      listNodesHandler,
		getOnlineNodesHandler: getOnlineNodesHandler,
	}
}

// Heartbeat 心跳上报
func (h *NodeHandler) Heartbeat(c *gin.Context) {
	var req struct {
		NodeID        string  `json:"node_id" binding:"required"`
		CPUUsage      float64 `json:"cpu_usage"`
		MemoryUsage   float64 `json:"memory_usage"`
		DiskUsage     float64 `json:"disk_usage"`
		TrafficIn     int64   `json:"traffic_in"`
		TrafficOut    int64   `json:"traffic_out"`
		ActiveRules   int     `json:"active_rules"`
		ConfigVersion int     `json:"config_version"`
		ClientVersion string  `json:"client_version"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err)
		return
	}
	
	// 执行命令
	cmd := commands.HeartbeatCommand{
		NodeID:        req.NodeID,
		CPUUsage:      req.CPUUsage,
		MemoryUsage:   req.MemoryUsage,
		DiskUsage:     req.DiskUsage,
		TrafficIn:     req.TrafficIn,
		TrafficOut:    req.TrafficOut,
		ActiveRules:   req.ActiveRules,
		ConfigVersion: req.ConfigVersion,
		ClientVersion: req.ClientVersion,
	}
	
	result, err := h.heartbeatHandler.Handle(c.Request.Context(), cmd)
	if err != nil {
		utils.Error(c, err)
		return
	}
	
	utils.Success(c, gin.H{
		"config_updated":     result.ConfigUpdated,
		"new_config_version": result.NewConfigVersion,
	})
}

// ListNodes 列表查询
func (h *NodeHandler) ListNodes(c *gin.Context) {
	var req struct {
		GroupID    uint   `form:"group_id"`
		Status     string `form:"status"`
		OnlineOnly bool   `form:"online_only"`
		Page       int    `form:"page"`
		PageSize   int    `form:"page_size"`
	}
	
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequest(c, err)
		return
	}
	
	// 执行查询
	query := queries.ListNodesQuery{
		GroupID:    req.GroupID,
		Status:     req.Status,
		OnlineOnly: req.OnlineOnly,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}
	
	result, err := h.listNodesHandler.Handle(c.Request.Context(), query)
	if err != nil {
		utils.Error(c, err)
		return
	}
	
	utils.Success(c, gin.H{
		"nodes": result.Nodes,
		"total": result.Total,
	})
}

// GetOnlineNodes 获取在线节点
func (h *NodeHandler) GetOnlineNodes(c *gin.Context) {
	query := queries.GetOnlineNodesQuery{}
	
	result, err := h.getOnlineNodesHandler.Handle(c.Request.Context(), query)
	if err != nil {
		utils.Error(c, err)
		return
	}
	
	utils.Success(c, gin.H{
		"nodes": result.Nodes,
		"count": result.Count,
	})
}
```

## 路由配置

```go
// cmd/server/main.go
func setupRouter(container *container.Container) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())
	
	// 创建 Handler
	userHandler := handlers.NewUserHandler(
		container.CreateUserHandler,
		container.GetUserHandler,
	)
	
	nodeHandler := handlers.NewNodeHandler(
		container.HeartbeatHandler,
		container.GetNodeHandler,
		container.ListNodesHandler,
		container.GetOnlineNodesHandler,
	)
	
	// API 路由
	api := r.Group("/api/v1")
	{
		// 用户路由
		users := api.Group("/users")
		{
			users.POST("", userHandler.Create)
			users.GET("/me", middleware.Auth(), userHandler.Get)
		}
		
		// 节点路由
		nodes := api.Group("/nodes")
		{
			nodes.POST("/heartbeat", nodeHandler.Heartbeat)
			nodes.GET("", middleware.Auth(), nodeHandler.ListNodes)
			nodes.GET("/online", middleware.Auth(), nodeHandler.GetOnlineNodes)
		}
	}
	
	return r
}
```

## 定时任务集成

```go
// cmd/server/main.go
func startCronTasks(container *container.Container) *cron.Cron {
	scheduler := cron.New(cron.WithSeconds())
	
	// 心跳刷新任务（每分钟）
	scheduler.AddFunc("0 * * * * *", func() {
		ctx := context.Background()
		if err := container.HeartbeatHandler.FlushHeartbeats(ctx); err != nil {
			log.Printf("心跳刷新失败: %v", err)
		}
	})
	
	// 离线节点检测（每 30 秒）
	scheduler.AddFunc("*/30 * * * * *", func() {
		ctx := context.Background()
		affected, err := container.NodeRepo.MarkOfflineByTimeout(ctx, 3*time.Minute)
		if err != nil {
			log.Printf("离线节点检测失败: %v", err)
		} else if affected > 0 {
			log.Printf("标记 %d 个节点为离线", affected)
		}
	})
	
	scheduler.Start()
	return scheduler
}
```

## 迁移策略

### 渐进式迁移

1. **保留旧 Handler**：不要立即删除
2. **新旧并存**：新功能使用新架构，旧功能保持不变
3. **逐步迁移**：每次迁移一个模块
4. **充分测试**：确保功能正常后再删除旧代码

### 示例：迁移用户登录

```go
// 旧代码（保留）
func (h *OldAuthHandler) Login(c *gin.Context) {
	// 直接操作数据库
	var user models.User
	database.DB.Where("email = ?", req.Email).First(&user)
	// ...
}

// 新代码（并存）
func (h *NewAuthHandler) LoginV2(c *gin.Context) {
	// 使用新架构
	cmd := commands.LoginCommand{
		Email:    req.Email,
		Password: req.Password,
	}
	result, err := h.loginHandler.Handle(c.Request.Context(), cmd)
	// ...
}

// 路由配置
api.POST("/auth/login", oldAuthHandler.Login)      // 旧接口
api.POST("/auth/login/v2", newAuthHandler.LoginV2) // 新接口
```

## 测试示例

```go
// internal/application/user/commands/create_user_test.go
package commands_test

import (
	"context"
	"testing"
	
	"nodepass-pro/backend/internal/application/user/commands"
	"nodepass-pro/backend/internal/domain/user"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository 模拟仓储
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

// 测试创建用户
func TestCreateUser(t *testing.T) {
	// 准备
	mockRepo := new(MockUserRepository)
	mockCache := &cache.UserCache{} // 简化，实际应该也 mock
	
	handler := commands.NewCreateUserHandler(mockRepo, mockCache)
	
	// 模拟：邮箱不存在
	mockRepo.On("FindByEmail", mock.Anything, "test@example.com").
		Return(nil, user.ErrUserNotFound)
	
	// 模拟：创建成功
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).
		Return(nil)
	
	// 执行
	cmd := commands.CreateUserCommand{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     "user",
	}
	
	result, err := handler.Handle(context.Background(), cmd)
	
	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotZero(t, result.User.ID)
	
	mockRepo.AssertExpectations(t)
}
```

## 性能监控

```go
// 添加性能监控中间件
func PerformanceMonitor() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		c.Next()
		
		duration := time.Since(start)
		
		// 记录慢请求
		if duration > 100*time.Millisecond {
			log.Printf("慢请求: %s %s 耗时 %v", 
				c.Request.Method, 
				c.Request.URL.Path, 
				duration)
		}
	}
}
```

## 常见问题

### Q1: 如何处理事务？

```go
func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUserCommand) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		// 在事务中执行多个操作
		if err := h.userRepo.Create(ctx, user); err != nil {
			return err
		}
		if err := h.quotaRepo.Create(ctx, quota); err != nil {
			return err
		}
		return nil
	})
}
```

### Q2: 如何处理缓存失效？

```go
// 更新时清除缓存
func (h *UpdateUserHandler) Handle(ctx context.Context, cmd UpdateUserCommand) error {
	if err := h.userRepo.Update(ctx, user); err != nil {
		return err
	}
	
	// 清除缓存
	h.userCache.Delete(ctx, user.ID)
	
	return nil
}
```

### Q3: 如何处理并发？

```go
// 使用分布式锁
func (h *UpdateUserHandler) Handle(ctx context.Context, cmd UpdateUserCommand) error {
	lock := cache.NewDistributedLock(h.redisClient, fmt.Sprintf("user:%d", cmd.UserID), 30*time.Second)
	
	if err := lock.Lock(ctx); err != nil {
		return err
	}
	defer lock.Unlock(ctx)
	
	// 执行更新操作
	// ...
}
```

## 下一步

1. 阅读 `QUICK_START.md` 启动开发环境
2. 查看 `REFACTORING_PROGRESS.md` 了解进度
3. 参考本文档集成到现有代码
4. 编写单元测试
5. 性能测试和优化

---

**需要帮助？** 查看其他文档或提 Issue！
