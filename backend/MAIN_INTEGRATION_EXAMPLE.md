# main.go 集成示例

本文档展示如何将新架构集成到 `cmd/server/main.go` 中。

## 完整的 main.go 示例

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/database"
	"nodepass-pro/backend/internal/infrastructure/container"
	"nodepass-pro/backend/internal/interfaces/http/handlers"
	"nodepass-pro/backend/internal/middleware"
	"nodepass-pro/backend/internal/utils"
	"nodepass-pro/backend/internal/version"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		panic(err)
	}

	// 2. 初始化日志
	logger, err := initLogger(cfg.Server.Mode)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	// 3. 初始化数据库
	db, err := database.InitDB(&cfg.Database)
	if err != nil {
		zap.L().Fatal("数据库初始化失败", zap.Error(err))
	}
	defer database.Close()

	// 4. 初始化 Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	// 测试 Redis 连接
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		zap.L().Fatal("Redis 连接失败", zap.Error(err))
	}

	// 5. 创建依赖注入容器
	appContainer := container.NewContainer(db, redisClient)
	defer appContainer.Close()

	zap.L().Info("依赖注入容器初始化完成")

	// 6. 设置路由
	router := setupRouter(appContainer)

	// 7. 启动定时任务
	scheduler := startCronTasks(appContainer)
	defer scheduler.Stop()

	// 8. 启动 HTTP 服务器
	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	go func() {
		zap.L().Info("服务启动",
			zap.String("addr", server.Addr),
			zap.String("mode", cfg.Server.Mode),
			zap.String("version", version.Version),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Fatal("服务启动失败", zap.Error(err))
		}
	}()

	// 9. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zap.L().Info("收到退出信号，开始优雅关闭")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		zap.L().Error("服务关闭失败", zap.Error(err))
	}
	zap.L().Info("服务已停止")
}

// setupRouter 设置路由
func setupRouter(c *container.Container) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.SecurityHeaders())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		utils.Success(c, gin.H{"status": "ok"})
	})

	// API 路由
	api := r.Group("/api/v1")
	{
		// ==================== 用户路由（新架构） ====================
		userHandler := handlers.NewExampleUserHandler(
			c.CreateUserHandler,
			c.GetUserHandler,
		)
		
		users := api.Group("/users")
		{
			users.POST("", userHandler.CreateUser)
			users.GET("/me", middleware.Auth(), userHandler.GetUser)
		}

		// ==================== 节点路由（新架构） ====================
		nodes := api.Group("/nodes")
		{
			// 心跳上报（无需认证）
			nodes.POST("/heartbeat", func(ctx *gin.Context) {
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

				if err := ctx.ShouldBindJSON(&req); err != nil {
					utils.BadRequest(ctx, err)
					return
				}

				// 使用容器中的 HeartbeatHandler
				cmd := nodeCommands.HeartbeatCommand{
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

				result, err := c.HeartbeatHandler.Handle(ctx.Request.Context(), cmd)
				if err != nil {
					utils.Error(ctx, err)
					return
				}

				utils.Success(ctx, gin.H{
					"config_updated":     result.ConfigUpdated,
					"new_config_version": result.NewConfigVersion,
				})
			})

			// 节点列表（需要认证）
			nodes.GET("", middleware.Auth(), func(ctx *gin.Context) {
				var req struct {
					GroupID    uint   `form:"group_id"`
					Status     string `form:"status"`
					OnlineOnly bool   `form:"online_only"`
					Page       int    `form:"page"`
					PageSize   int    `form:"page_size"`
				}

				if err := ctx.ShouldBindQuery(&req); err != nil {
					utils.BadRequest(ctx, err)
					return
				}

				query := nodeQueries.ListNodesQuery{
					GroupID:    req.GroupID,
					Status:     req.Status,
					OnlineOnly: req.OnlineOnly,
					Page:       req.Page,
					PageSize:   req.PageSize,
				}

				result, err := c.ListNodesHandler.Handle(ctx.Request.Context(), query)
				if err != nil {
					utils.Error(ctx, err)
					return
				}

				utils.Success(ctx, gin.H{
					"nodes": result.Nodes,
					"total": result.Total,
				})
			})

			// 在线节点
			nodes.GET("/online", middleware.Auth(), func(ctx *gin.Context) {
				query := nodeQueries.GetOnlineNodesQuery{}
				result, err := c.GetOnlineNodesHandler.Handle(ctx.Request.Context(), query)
				if err != nil {
					utils.Error(ctx, err)
					return
				}

				utils.Success(ctx, gin.H{
					"nodes": result.Nodes,
					"count": result.Count,
				})
			})
		}

		// ==================== 隧道路由（新架构） ====================
		tunnels := api.Group("/tunnels")
		tunnels.Use(middleware.Auth())
		{
			// 创建隧道
			tunnels.POST("", func(ctx *gin.Context) {
				userID := ctx.GetUint("user_id")
				
				var req struct {
					Name        string `json:"name" binding:"required"`
					Description string `json:"description"`
					Protocol    string `json:"protocol" binding:"required"`
					Mode        string `json:"mode" binding:"required"`
					ListenHost  string `json:"listen_host"`
					ListenPort  int    `json:"listen_port" binding:"required"`
					TargetHost  string `json:"target_host" binding:"required"`
					TargetPort  int    `json:"target_port" binding:"required"`
					EntryNodeID uint   `json:"entry_node_id"`
					ExitNodeID  uint   `json:"exit_node_id"`
				}

				if err := ctx.ShouldBindJSON(&req); err != nil {
					utils.BadRequest(ctx, err)
					return
				}

				cmd := tunnelCommands.CreateTunnelCommand{
					UserID:      userID,
					Name:        req.Name,
					Description: req.Description,
					Protocol:    req.Protocol,
					Mode:        req.Mode,
					ListenHost:  req.ListenHost,
					ListenPort:  req.ListenPort,
					TargetHost:  req.TargetHost,
					TargetPort:  req.TargetPort,
					EntryNodeID: req.EntryNodeID,
					ExitNodeID:  req.ExitNodeID,
				}

				result, err := c.CreateTunnelHandler.Handle(ctx.Request.Context(), cmd)
				if err != nil {
					utils.Error(ctx, err)
					return
				}

				utils.Success(ctx, result.Tunnel)
			})

			// 启动隧道
			tunnels.POST("/:id/start", func(ctx *gin.Context) {
				userID := ctx.GetUint("user_id")
				tunnelID := ctx.GetUint("id")

				cmd := tunnelCommands.StartTunnelCommand{
					TunnelID: tunnelID,
					UserID:   userID,
				}

				_, err := c.StartTunnelHandler.Handle(ctx.Request.Context(), cmd)
				if err != nil {
					utils.Error(ctx, err)
					return
				}

				utils.Success(ctx, gin.H{"message": "隧道已启动"})
			})

			// 停止隧道
			tunnels.POST("/:id/stop", func(ctx *gin.Context) {
				userID := ctx.GetUint("user_id")
				tunnelID := ctx.GetUint("id")

				cmd := tunnelCommands.StopTunnelCommand{
					TunnelID: tunnelID,
					UserID:   userID,
				}

				_, err := c.StopTunnelHandler.Handle(ctx.Request.Context(), cmd)
				if err != nil {
					utils.Error(ctx, err)
					return
				}

				utils.Success(ctx, gin.H{"message": "隧道已停止"})
			})
		}

		// ==================== 流量路由（新架构） ====================
		traffic := api.Group("/traffic")
		traffic.Use(middleware.Auth())
		{
			// 获取用户流量统计
			traffic.GET("/stats", func(ctx *gin.Context) {
				userID := ctx.GetUint("user_id")

				query := trafficQueries.GetUserTrafficQuery{
					UserID: userID,
				}

				result, err := c.GetUserTrafficHandler.Handle(ctx.Request.Context(), query)
				if err != nil {
					utils.Error(ctx, err)
					return
				}

				utils.Success(ctx, result)
			})
		}
	}

	return r
}

// startCronTasks 启动定时任务
func startCronTasks(c *container.Container) *cron.Cron {
	scheduler := cron.New(cron.WithSeconds())

	// 心跳刷新任务（每分钟）
	scheduler.AddFunc("0 * * * * *", func() {
		ctx := context.Background()
		if err := c.HeartbeatHandler.FlushHeartbeats(ctx); err != nil {
			zap.L().Error("心跳刷新失败", zap.Error(err))
		} else {
			zap.L().Debug("心跳刷新完成")
		}
	})

	// 离线节点检测（每 30 秒）
	scheduler.AddFunc("*/30 * * * * *", func() {
		ctx := context.Background()
		affected, err := c.NodeInstanceRepo.MarkOfflineByTimeout(ctx, 3*time.Minute)
		if err != nil {
			zap.L().Error("离线节点检测失败", zap.Error(err))
		} else if affected > 0 {
			zap.L().Info("标记离线节点", zap.Int64("count", affected))
		}
	})

	// 流量刷新任务（每小时）
	scheduler.AddFunc("0 0 * * * *", func() {
		ctx := context.Background()
		result, err := c.FlushTrafficHandler.Handle(ctx, trafficCommands.FlushTrafficCommand{})
		if err != nil {
			zap.L().Error("流量刷新失败", zap.Error(err))
		} else {
			zap.L().Info("流量刷新完成",
				zap.Int("users", result.FlushedUsers),
				zap.Int("tunnels", result.FlushedTunnels))
		}
	})

	scheduler.Start()
	zap.L().Info("定时任务调度器已启动")
	return scheduler
}

// initLogger 初始化日志
func initLogger(mode string) (*zap.Logger, error) {
	if mode == gin.ReleaseMode {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
```

## 关键点说明

### 1. 容器初始化

```go
// 创建容器
appContainer := container.NewContainer(db, redisClient)
defer appContainer.Close()
```

容器会自动初始化所有依赖：
- 仓储层（PostgreSQL）
- 缓存层（Redis）
- 命令处理器
- 查询处理器

### 2. Handler 使用

```go
// 从容器获取处理器
userHandler := handlers.NewExampleUserHandler(
    c.CreateUserHandler,
    c.GetUserHandler,
)
```

### 3. 内联 Handler

对于简单的路由，可以直接在路由中使用容器的处理器：

```go
nodes.POST("/heartbeat", func(ctx *gin.Context) {
    // 绑定请求
    var req HeartbeatRequest
    ctx.ShouldBindJSON(&req)
    
    // 构建命令
    cmd := nodeCommands.HeartbeatCommand{...}
    
    // 执行命令
    result, err := c.HeartbeatHandler.Handle(ctx.Request.Context(), cmd)
    
    // 返回结果
    utils.Success(ctx, result)
})
```

### 4. 定时任务集成

```go
// 使用容器中的处理器
scheduler.AddFunc("0 * * * * *", func() {
    c.HeartbeatHandler.FlushHeartbeats(ctx)
})
```

## 渐进式迁移策略

### 阶段 1：新旧并存

```go
// 旧路由（保留）
api.POST("/auth/login", oldAuthHandler.Login)

// 新路由（并存）
api.POST("/auth/login/v2", newAuthHandler.LoginV2)
```

### 阶段 2：逐步替换

1. 先迁移读操作（查询）
2. 再迁移写操作（命令）
3. 最后删除旧代码

### 阶段 3：完全切换

```go
// 只保留新路由
api.POST("/auth/login", newAuthHandler.LoginV2)
```

## 下一步

1. 复制此示例到 `cmd/server/main.go`
2. 根据实际情况调整路由
3. 逐步迁移现有 Handler
4. 编写单元测试
5. 性能测试

---

**参考文档**:
- INTEGRATION_GUIDE.md - 详细集成指南
- QUICK_START.md - 快速开始
- REFACTORING_PROGRESS.md - 进度跟踪
