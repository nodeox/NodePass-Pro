package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"nodepass-panel/backend/internal/cache"
	"nodepass-panel/backend/internal/config"
	"nodepass-panel/backend/internal/database"
	"nodepass-panel/backend/internal/handlers"
	"nodepass-panel/backend/internal/middleware"
	"nodepass-panel/backend/internal/models"
	"nodepass-panel/backend/internal/services"
	"nodepass-panel/backend/internal/utils"
	"nodepass-panel/backend/internal/version"
	panelws "nodepass-panel/backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		panic(err)
	}

	logger, err := initLogger(cfg.Server.Mode)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()
	zap.ReplaceGlobals(logger)

	gin.SetMode(cfg.Server.Mode)

	if err = cache.Init(&cfg.Redis); err != nil {
		zap.L().Fatal("Redis 初始化失败", zap.Error(err))
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			zap.L().Warn("Redis 连接关闭失败", zap.Error(closeErr))
		}
	}()

	if _, err = database.InitDB(&cfg.Database); err != nil {
		zap.L().Fatal("数据库初始化失败", zap.Error(err))
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			zap.L().Warn("数据库关闭失败", zap.Error(closeErr))
		}
	}()

	router, wsHub := setupRouter()
	taskScheduler, err := startCronTasks(wsHub)
	if err != nil {
		zap.L().Fatal("定时任务注册失败", zap.Error(err))
	}
	defer func() {
		ctx := taskScheduler.Stop()
		<-ctx.Done()
		zap.L().Info("定时任务调度器已停止")
	}()

	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	go func() {
		zap.L().Info("服务启动",
			zap.String("addr", server.Addr),
			zap.String("mode", cfg.Server.Mode),
			zap.String("db_type", cfg.Database.Type),
			zap.String("version", version.Version),
		)
		if runErr := server.ListenAndServe(); runErr != nil && !errors.Is(runErr, http.ErrServerClosed) {
			zap.L().Fatal("服务启动失败", zap.Error(runErr))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zap.L().Info("收到退出信号，开始优雅关闭")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if shutdownErr := server.Shutdown(ctx); shutdownErr != nil {
		zap.L().Error("服务关闭失败", zap.Error(shutdownErr))
	}
	zap.L().Info("服务已停止")
}

func initLogger(mode string) (*zap.Logger, error) {
	if mode == gin.ReleaseMode {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

func setupRouter() (*gin.Engine, *panelws.Hub) {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.RequestBodyLimit(10 * 1024 * 1024)) // 10MB 限制
	r.Use(middleware.RateLimit(50, 100))

	r.GET("/health", func(c *gin.Context) {
		utils.Success(c, gin.H{"status": "ok", "version": version.Version})
	})

	api := r.Group("/api/v1")
	api.GET("/ping", func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "pong", "version": version.Version})
	})

	nodeHandler := handlers.NewNodeHandler(database.DB)
	nodePairHandler := handlers.NewNodePairHandler(database.DB)
	ruleHandler := handlers.NewRuleHandler(database.DB)
	nodeAgentHandler := handlers.NewNodeAgentHandler(database.DB)
	trafficHandler := handlers.NewTrafficHandler(database.DB)
	vipHandler := handlers.NewVIPHandler(database.DB)
	benefitCodeHandler := handlers.NewBenefitCodeHandler(database.DB)
	telegramHandler := handlers.NewTelegramHandler(database.DB)
	authHandler := handlers.NewAuthHandler(database.DB)

	wsHub := panelws.NewHub()
	wsHandler := handlers.NewWebSocketHandler(wsHub)
	systemHandler := handlers.NewSystemHandler(database.DB, wsHub)
	announcementHandler := handlers.NewAnnouncementHandler(database.DB, wsHub)
	auditHandler := handlers.NewAuditHandler(database.DB)

	if cfg := config.GlobalConfig; cfg != nil {
		botToken := strings.TrimSpace(cfg.Telegram.BotToken)
		if botToken != "" {
			if err := telegramHandler.InitBot(botToken); err != nil {
				zap.L().Warn("初始化 Telegram Bot 失败", zap.Error(err))
			} else {
				zap.L().Info("Telegram Bot 初始化成功")
			}
		}
	}

	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/telegram/webhook", telegramHandler.Webhook)
	api.POST("/telegram/login", telegramHandler.Login)

	agentNodes := api.Group("/nodes")
	{
		agentNodes.POST("/register", nodeAgentHandler.Register)
		agentNodes.POST("/heartbeat", nodeAgentHandler.Heartbeat)
		agentNodes.GET("/:id/config", nodeAgentHandler.PullConfig)
		agentNodes.POST("/traffic/report", trafficHandler.ReportTraffic)
	}

	authGroup := api.Group("")
	authGroup.Use(middleware.AuthMiddleware())
	authGroup.Use(middleware.AuditLogger())
	{
		auth := authGroup.Group("/auth")
		{
			auth.GET("/me", authHandler.Me)
			auth.POST("/refresh", authHandler.Refresh)
			auth.PUT("/password", authHandler.ChangePassword)
		}

		nodes := authGroup.Group("/nodes")
		{
			nodes.POST("", nodeHandler.CreateNode)
			nodes.GET("", nodeHandler.ListNodes)
			nodes.GET("/quota", nodeHandler.GetQuota)
			nodes.GET("/:id", nodeHandler.GetNode)
			nodes.PUT("/:id", nodeHandler.UpdateNode)
			nodes.DELETE("/:id", nodeHandler.DeleteNode)
		}

		pairs := authGroup.Group("/node-pairs")
		{
			pairs.POST("", nodePairHandler.CreatePair)
			pairs.GET("", nodePairHandler.ListPairs)
			pairs.PUT("/:id", nodePairHandler.UpdatePair)
			pairs.DELETE("/:id", nodePairHandler.DeletePair)
			pairs.PUT("/:id/toggle", nodePairHandler.TogglePair)
		}

		rules := authGroup.Group("/rules")
		{
			rules.POST("", ruleHandler.CreateRule)
			rules.GET("", ruleHandler.ListRules)
			rules.GET("/:id", ruleHandler.GetRule)
			rules.PUT("/:id", ruleHandler.UpdateRule)
			rules.DELETE("/:id", ruleHandler.DeleteRule)
			rules.POST("/:id/start", ruleHandler.StartRule)
			rules.POST("/:id/stop", ruleHandler.StopRule)
			rules.POST("/:id/restart", ruleHandler.RestartRule)
		}

		traffic := authGroup.Group("/traffic")
		{
			traffic.GET("/quota", trafficHandler.GetQuota)
			traffic.GET("/usage", trafficHandler.GetUsage)
			traffic.GET("/records", trafficHandler.GetRecords)
		}

		vip := authGroup.Group("/vip")
		{
			vip.GET("/levels", vipHandler.ListLevels)
			vip.GET("/my-level", vipHandler.GetMyLevel)
		}

		benefitCodes := authGroup.Group("/benefit-codes")
		{
			benefitCodes.POST("/redeem", benefitCodeHandler.Redeem)
		}

		announcements := authGroup.Group("/announcements")
		{
			announcements.GET("", announcementHandler.List)
		}

		telegram := authGroup.Group("/telegram")
		{
			telegram.POST("/bind", telegramHandler.Bind)
			telegram.POST("/unbind", telegramHandler.Unbind)
		}
	}

	adminGroup := authGroup.Group("")
	adminGroup.Use(middleware.RequireRole("admin"))
	{
		system := adminGroup.Group("/system")
		{
			system.GET("/config", systemHandler.GetConfig)
			system.PUT("/config", systemHandler.UpdateConfig)
			system.GET("/stats", systemHandler.GetStats)
		}

		adminTraffic := adminGroup.Group("/traffic")
		{
			adminTraffic.POST("/quota/reset", trafficHandler.ResetQuota)
		}

		adminVIP := adminGroup.Group("/vip")
		{
			adminVIP.POST("/levels", vipHandler.CreateLevel)
			adminVIP.PUT("/levels/:id", vipHandler.UpdateLevel)
		}

		adminUsers := adminGroup.Group("/users")
		{
			adminUsers.POST("/:id/vip/upgrade", vipHandler.UpgradeUser)
		}

		adminBenefitCodes := adminGroup.Group("/benefit-codes")
		{
			adminBenefitCodes.POST("/generate", benefitCodeHandler.Generate)
			adminBenefitCodes.GET("", benefitCodeHandler.List)
			adminBenefitCodes.POST("/batch-delete", benefitCodeHandler.BatchDelete)
		}

		adminAnnouncements := adminGroup.Group("/announcements")
		{
			adminAnnouncements.POST("", announcementHandler.Create)
			adminAnnouncements.PUT("/:id", announcementHandler.Update)
			adminAnnouncements.DELETE("/:id", announcementHandler.Delete)
		}

		adminGroup.GET("/audit-logs", auditHandler.List)
	}

	wsRoute := r.Group("/ws")
	wsRoute.Use(middleware.AuthMiddleware())
	{
		wsRoute.GET("", wsHandler.Handle)
	}

	return r, wsHub
}

func startCronTasks(wsHub *panelws.Hub) (*cron.Cron, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	taskScheduler := cron.New(
		cron.WithSeconds(),
		cron.WithLocation(time.Local),
	)

	trafficService := services.NewTrafficService(database.DB)
	vipService := services.NewVIPService(database.DB)

	addTask := func(spec string, taskName string, job func() error) error {
		_, err := taskScheduler.AddFunc(spec, func() {
			startAt := time.Now()
			zap.L().Info("定时任务开始",
				zap.String("task", taskName),
				zap.String("spec", spec),
			)

			if runErr := job(); runErr != nil {
				zap.L().Error("定时任务执行失败",
					zap.String("task", taskName),
					zap.Error(runErr),
				)
				return
			}

			zap.L().Info("定时任务完成",
				zap.String("task", taskName),
				zap.Duration("duration", time.Since(startAt)),
			)
		})
		if err != nil {
			return fmt.Errorf("注册定时任务失败(%s): %w", taskName, err)
		}

		zap.L().Info("定时任务已注册",
			zap.String("task", taskName),
			zap.String("spec", spec),
		)
		return nil
	}

	if err := addTask("0 0 0 1 * *", "traffic.monthly_reset", func() error {
		if runErr := trafficService.MonthlyReset(); runErr != nil {
			return runErr
		}
		zap.L().Info("每月流量重置完成")
		return nil
	}); err != nil {
		return nil, err
	}

	if err := addTask("0 0 * * * *", "vip.check_expiration", func() error {
		affected, runErr := vipService.CheckExpiration()
		if runErr != nil {
			return runErr
		}
		zap.L().Info("VIP 到期检查完成", zap.Int64("degraded_users", affected))
		return nil
	}); err != nil {
		return nil, err
	}

	if err := addTask("*/30 * * * * *", "nodes.heartbeat_timeout_check", func() error {
		offlineCount, runErr := checkNodeHeartbeatTimeout(wsHub)
		if runErr != nil {
			return runErr
		}
		zap.L().Info("节点心跳超时检查完成", zap.Int("offline_nodes", offlineCount))
		return nil
	}); err != nil {
		return nil, err
	}

	if err := addTask("0 0 0 * * *", "audit.cleanup_90d", func() error {
		rows, runErr := cleanupExpiredAuditLogs()
		if runErr != nil {
			return runErr
		}
		zap.L().Info("审计日志清理完成", zap.Int64("deleted_rows", rows))
		return nil
	}); err != nil {
		return nil, err
	}

	if err := addTask("0 0 * * * *", "csrf.cleanup_expired_tokens", func() error {
		middleware.CleanupExpiredTokens()
		zap.L().Info("CSRF 令牌清理完成")
		return nil
	}); err != nil {
		return nil, err
	}

	taskScheduler.Start()
	zap.L().Info("定时任务调度器已启动")
	return taskScheduler, nil
}

func checkNodeHeartbeatTimeout(wsHub *panelws.Hub) (int, error) {
	cutoff := time.Now().Add(-3 * time.Minute)

	nodes := make([]models.Node, 0)
	if err := database.DB.Model(&models.Node{}).
		Where("status <> ? AND last_heartbeat_at IS NOT NULL AND last_heartbeat_at < ?", "offline", cutoff).
		Find(&nodes).Error; err != nil {
		return 0, fmt.Errorf("查询心跳超时节点失败: %w", err)
	}
	if len(nodes) == 0 {
		return 0, nil
	}

	ids := make([]uint, 0, len(nodes))
	for _, node := range nodes {
		ids = append(ids, node.ID)
	}

	if err := database.DB.Model(&models.Node{}).
		Where("id IN ?", ids).
		Update("status", "offline").Error; err != nil {
		return 0, fmt.Errorf("更新离线节点状态失败: %w", err)
	}

	if wsHub != nil {
		for _, node := range nodes {
			payload := map[string]interface{}{
				"node_id":           node.ID,
				"user_id":           node.UserID,
				"status":            "offline",
				"reason":            "heartbeat_timeout",
				"last_heartbeat_at": node.LastHeartbeatAt,
			}
			if err := wsHub.Broadcast(panelws.MessageTypeNodeStatusChanged, payload); err != nil {
				zap.L().Warn("推送节点离线事件失败",
					zap.Uint("node_id", node.ID),
					zap.Error(err),
				)
			}
		}
	}

	return len(nodes), nil
}

func cleanupExpiredAuditLogs() (int64, error) {
	expireAt := time.Now().AddDate(0, 0, -90)
	result := database.DB.Where("created_at < ?", expireAt).Delete(&models.AuditLog{})
	if result.Error != nil {
		return 0, fmt.Errorf("清理审计日志失败: %w", result.Error)
	}
	return result.RowsAffected, nil
}
