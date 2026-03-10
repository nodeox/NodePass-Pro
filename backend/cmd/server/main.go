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

	"nodepass-pro/backend/internal/cache"
	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/database"
	"nodepass-pro/backend/internal/handlers"
	"nodepass-pro/backend/internal/license"
	"nodepass-pro/backend/internal/middleware"
	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"
	"nodepass-pro/backend/internal/version"
	panelws "nodepass-pro/backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

const insecureJWTSecretPlaceholder = "change-this-secret-in-production"

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

	if err = validateSecurityConfig(cfg); err != nil {
		zap.L().Fatal("关键安全配置校验失败", zap.Error(err))
	}

	gin.SetMode(cfg.Server.Mode)

	licenseManager := license.NewManager(&cfg.License, &cfg.Server)
	licenseCtx, stopLicenseCheck := context.WithCancel(context.Background())
	defer stopLicenseCheck()
	licenseManager.Start(licenseCtx)
	if licenseManager.Enabled() {
		licenseStatus := licenseManager.Status()
		if licenseStatus.Valid {
			zap.L().Info("运行时授权校验已启用", zap.String("message", licenseStatus.Message))
		} else {
			zap.L().Warn("运行时授权校验未通过，业务请求将被拦截", zap.String("message", licenseStatus.Message))
		}
	}

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

	router, _ := setupRouter(licenseManager)
	taskScheduler, err := startCronTasks()
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

func validateSecurityConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("配置为空")
	}

	secret := strings.TrimSpace(cfg.JWT.Secret)
	if secret == "" {
		return fmt.Errorf("JWT Secret 未配置，请设置 configs/config.yaml 的 jwt.secret 或 NODEPASS_JWT_SECRET")
	}
	if secret == insecureJWTSecretPlaceholder {
		return fmt.Errorf("检测到默认 JWT Secret，请修改后再启动")
	}
	if len(secret) < 64 {
		return fmt.Errorf("JWT Secret 长度不足 64 字符（当前: %d 字符），请使用更长的随机字符串以确保安全性", len(secret))
	}
	return nil
}

func setupRouter(licenseManager *license.Manager) (*gin.Engine, *panelws.Hub) {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID()) // 请求 ID 追踪（必须在最前面）
	r.Use(middleware.RequestLogger())
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.SecurityHeaders())                 // 添加安全 HTTP 头
	r.Use(middleware.RequestBodyLimit(1 * 1024 * 1024)) // 全局默认 1MB 限制
	r.Use(middleware.RateLimit(20, 50))                 // 降低全局速率限制：20 QPS，50 突发

	// 健康检查端点（不需要认证）
	healthHandler := handlers.NewHealthHandler()
	r.GET("/health", healthHandler.Health)
	r.GET("/readiness", healthHandler.Readiness)
	r.GET("/liveness", healthHandler.Liveness)

	// 旧的健康检查端点（保持兼容性，包含授权信息）
	r.GET("/health-legacy", func(c *gin.Context) {
		licenseStatus := gin.H{
			"enabled": false,
			"valid":   true,
		}
		if licenseManager != nil {
			status := licenseManager.Status()
			licenseStatus = gin.H{
				"enabled":         status.Enabled,
				"valid":           status.Valid,
				"message":         status.Message,
				"last_checked_at": status.LastCheckedAt,
				"expires_at":      status.ExpiresAt,
			}
		}
		utils.Success(c, gin.H{"status": "ok", "version": version.Version, "license": licenseStatus})
	})

	api := r.Group("/api/v1")
	api.Use(middleware.LicenseGuard(licenseManager))
	api.GET("/ping", func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "pong", "version": version.Version})
	})
	licenseHandler := handlers.NewLicenseRuntimeHandler(licenseManager)
	api.GET("/license/status", licenseHandler.GetStatus)

	nodeGroupHandler := handlers.NewNodeGroupHandler(database.DB)
	nodeInstanceHandler := handlers.NewNodeInstanceHandler(database.DB, licenseManager)
	tunnelHandler := handlers.NewTunnelHandler(database.DB)
	trafficHandler := handlers.NewTrafficHandler(database.DB)
	vipHandler := handlers.NewVIPHandler(database.DB)
	benefitCodeHandler := handlers.NewBenefitCodeHandler(database.DB)
	telegramHandler := handlers.NewTelegramHandler(database.DB)
	authHandler := handlers.NewAuthHandler(database.DB)
	userAdminHandler := handlers.NewUserAdminHandler(database.DB)
	roleAdminHandler := handlers.NewRoleAdminHandler(database.DB)
	alertHandler := handlers.NewAlertHandler(database.DB)
	alertRuleHandler := handlers.NewAlertRuleHandler(database.DB)
	notificationChannelHandler := handlers.NewNotificationChannelHandler(database.DB)
	nodeHealthHandler := handlers.NewNodeHealthHandler(database.DB)
	nodePerformanceHandler := handlers.NewNodePerformanceHandler(database.DB)

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
	api.POST("/auth/login", middleware.RateLimit(0.2, 5), authHandler.LoginDeprecated)
	api.POST("/auth/refresh", middleware.RateLimit(1, 10), authHandler.RefreshDeprecated)

	// V2 认证接口 - 支持 refresh token
	api.POST("/auth/login/v2", middleware.RateLimit(0.2, 5), authHandler.LoginV2)
	api.POST("/auth/refresh/v2", middleware.RateLimit(1, 10), authHandler.RefreshTokenV2)
	api.POST("/auth/logout", authHandler.Logout)

	// Telegram webhook - 严格的请求体限制（64KB）和签名验证
	api.POST("/telegram/webhook",
		middleware.RequestBodyLimit(64*1024), // 64KB
		middleware.TelegramWebhookAuth(),     // 签名验证
		telegramHandler.Webhook)
	api.POST("/telegram/login", middleware.RateLimit(0.5, 5), telegramHandler.Login)
	api.GET("/telegram/sso-login", middleware.RateLimit(1, 20), telegramHandler.SSOLogin)

	// 节点心跳接口 - 添加严格的速率限制和防重放保护
	api.POST("/node-instances/heartbeat",
		middleware.HeartbeatRateLimit(2, 20),   // 按 node_id 限流，回退 IP
		middleware.HeartbeatReplayProtection(), // 防重放攻击
		nodeInstanceHandler.Heartbeat)

	authGroup := api.Group("")
	authGroup.Use(middleware.AuthMiddleware())
	authGroup.Use(middleware.CSRFProtection())
	authGroup.Use(middleware.AuditLogger())
	{
		auth := authGroup.Group("/auth")
		{
			auth.GET("/me", authHandler.Me)
			auth.GET("/sessions", authHandler.ListSessions)
			auth.PUT("/password", authHandler.ChangePassword)
			auth.POST("/email/code", authHandler.SendEmailChangeCode)
			auth.PUT("/email", authHandler.ChangeEmail)
			auth.POST("/revoke-current", authHandler.RevokeCurrentSession)
			auth.DELETE("/sessions/:id", authHandler.RevokeSession)
			auth.POST("/revoke-all", authHandler.RevokeAllTokens) // 撤销所有 tokens
		}

		nodeGroups := authGroup.Group("/node-groups")
		{
			nodeGroups.POST("", nodeGroupHandler.Create)
			nodeGroups.GET("", nodeGroupHandler.List)
			nodeGroups.GET("/accessible-nodes", nodeGroupHandler.ListAccessibleNodes)
			nodeGroups.GET("/:id", nodeGroupHandler.Get)
			nodeGroups.PUT("/:id", nodeGroupHandler.Update)
			nodeGroups.DELETE("/:id", nodeGroupHandler.Delete)
			nodeGroups.POST("/:id/toggle", nodeGroupHandler.Toggle)
			nodeGroups.GET("/:id/stats", nodeGroupHandler.GetStats)
			nodeGroups.POST("/:id/generate-deploy-command", nodeGroupHandler.GenerateDeployCommand)
			nodeGroups.POST("/:id/relations", nodeGroupHandler.CreateRelation)
			nodeGroups.GET("/:id/relations", nodeGroupHandler.ListRelations)
			nodeGroups.GET("/:id/nodes", nodeGroupHandler.ListNodes)
			nodeGroups.POST("/:id/nodes", nodeGroupHandler.AddNode)
		}

		nodeGroupRelations := authGroup.Group("/node-group-relations")
		{
			nodeGroupRelations.DELETE("/:id", nodeGroupHandler.DeleteRelation)
			nodeGroupRelations.POST("/:id/toggle", nodeGroupHandler.ToggleRelation)
		}

		nodeInstances := authGroup.Group("/node-instances")
		{
			nodeInstances.GET("/:id", nodeInstanceHandler.Get)
			nodeInstances.PUT("/:id", nodeInstanceHandler.Update)
			nodeInstances.DELETE("/:id", nodeInstanceHandler.Delete)
			nodeInstances.POST("/:id/restart", nodeInstanceHandler.Restart)

			// 节点健康检查
			nodeInstances.POST("/:id/health-check", nodeHealthHandler.CreateHealthCheck)
			nodeInstances.GET("/:id/health-check", nodeHealthHandler.GetHealthCheck)
			nodeInstances.PUT("/:id/health-check", nodeHealthHandler.UpdateHealthCheck)
			nodeInstances.DELETE("/:id/health-check", nodeHealthHandler.DeleteHealthCheck)
			nodeInstances.POST("/:id/health-check/perform", nodeHealthHandler.PerformHealthCheck)
			nodeInstances.GET("/:id/quality-score", nodeHealthHandler.GetQualityScore)
			nodeInstances.GET("/:id/health-records", nodeHealthHandler.GetHealthRecords)
			nodeInstances.GET("/:id/health-stats", nodeHealthHandler.GetHealthStats)

			// 节点性能监控
			nodeInstances.POST("/:id/performance/metrics", nodePerformanceHandler.RecordMetric)
			nodeInstances.GET("/:id/performance/latest", nodePerformanceHandler.GetLatestMetric)
			nodeInstances.GET("/:id/performance/metrics", nodePerformanceHandler.GetMetrics)
			nodeInstances.GET("/:id/performance/stats", nodePerformanceHandler.GetMetricsStats)
			nodeInstances.POST("/:id/performance/alert", nodePerformanceHandler.CreateAlert)
			nodeInstances.GET("/:id/performance/alert", nodePerformanceHandler.GetAlert)
			nodeInstances.PUT("/:id/performance/alert", nodePerformanceHandler.UpdateAlert)
			nodeInstances.DELETE("/:id/performance/alert", nodePerformanceHandler.DeleteAlert)
			nodeInstances.GET("/:id/performance/alert-records", nodePerformanceHandler.GetAlertRecords)
			nodeInstances.GET("/:id/performance/summaries", nodePerformanceHandler.GetSummaries)
		}

		// 节点质量评分列表
		authGroup.GET("/node-instances/quality-scores", nodeHealthHandler.ListQualityScores)

		// 解决性能告警
		authGroup.POST("/node-instances/performance/alert-records/:alert_id/resolve", nodePerformanceHandler.ResolveAlert)

		tunnels := authGroup.Group("/tunnels")
		{
			tunnels.POST("", tunnelHandler.Create)
			tunnels.GET("", tunnelHandler.List)
			tunnels.GET("/:id", tunnelHandler.Get)
			tunnels.PUT("/:id", tunnelHandler.Update)
			tunnels.DELETE("/:id", tunnelHandler.Delete)
			tunnels.POST("/:id/start", tunnelHandler.Start)
			tunnels.POST("/:id/stop", tunnelHandler.Stop)
		}

		// 隧道模板管理
		tunnelTemplateHandler := handlers.NewTunnelTemplateHandler(database.DB)
		tunnelTemplates := authGroup.Group("/tunnel-templates")
		{
			tunnelTemplates.POST("", tunnelTemplateHandler.Create)
			tunnelTemplates.GET("", tunnelTemplateHandler.List)
			tunnelTemplates.GET("/:id", tunnelTemplateHandler.Get)
			tunnelTemplates.PUT("/:id", tunnelTemplateHandler.Update)
			tunnelTemplates.DELETE("/:id", tunnelTemplateHandler.Delete)
		}

		// 隧道导入导出
		tunnelImportExportHandler := handlers.NewTunnelImportExportHandler(database.DB)
		authGroup.POST("/tunnels/export", tunnelImportExportHandler.Export)
		authGroup.POST("/tunnels/export-all", tunnelImportExportHandler.ExportAll)
		authGroup.POST("/tunnels/import", tunnelImportExportHandler.Import)
		authGroup.POST("/tunnels/apply-template", tunnelImportExportHandler.ApplyTemplate)

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
			telegram.POST("/sso-url", telegramHandler.GenerateSSOURL)
			telegram.POST("/notify", telegramHandler.NotifySelf)
		}
	}

	adminGroup := authGroup.Group("")
	adminGroup.Use(middleware.RequireRole("admin"))
	adminGroup.Use(middleware.RateLimit(10, 20)) // 管理端点限流：10 QPS，20 突发
	{
		adminLicense := adminGroup.Group("/license")
		{
			adminLicense.PUT("/domain", licenseHandler.UpdateDomain)
		}

		system := adminGroup.Group("/system")
		{
			system.GET("/config", systemHandler.GetConfig)
			system.PUT("/config", systemHandler.UpdateConfig)
			system.GET("/stats", systemHandler.GetStats)
		}

		adminTraffic := adminGroup.Group("/traffic")
		{
			adminTraffic.POST("/quota/reset", trafficHandler.ResetQuota)
			adminTraffic.PUT("/quota/:id", trafficHandler.UpdateQuota)
		}

		adminVIP := adminGroup.Group("/vip")
		{
			adminVIP.POST("/levels", vipHandler.CreateLevel)
			adminVIP.PUT("/levels/:id", vipHandler.UpdateLevel)
		}

		adminUsers := adminGroup.Group("/users")
		{
			adminUsers.GET("", userAdminHandler.ListUsers)
			adminUsers.GET("/:id", userAdminHandler.GetUser)
			adminUsers.GET("/:id/detail", userAdminHandler.GetUserDetail)
			adminUsers.PUT("/:id/role", userAdminHandler.UpdateRole)
			adminUsers.PUT("/:id/status", userAdminHandler.UpdateStatus)
			adminUsers.POST("/:id/vip/upgrade", vipHandler.UpgradeUser)
		}

		adminRoles := adminGroup.Group("/roles")
		{
			adminRoles.GET("", roleAdminHandler.ListRoles)
			adminRoles.GET("/permissions", roleAdminHandler.ListAvailablePermissions)
			adminRoles.POST("", roleAdminHandler.CreateRole)
			adminRoles.GET("/:id", roleAdminHandler.GetRole)
			adminRoles.PUT("/:id", roleAdminHandler.UpdateRole)
			adminRoles.PUT("/:id/permissions", roleAdminHandler.UpdateRolePermissions)
			adminRoles.DELETE("/:id", roleAdminHandler.DeleteRole)
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

		// 告警管理
		alerts := adminGroup.Group("/alerts")
		{
			alerts.GET("", alertHandler.List)
			alerts.GET("/firing", alertHandler.GetFiring)
			alerts.GET("/stats", alertHandler.Stats)
			alerts.GET("/:id", alertHandler.Get)
			alerts.POST("/:id/acknowledge", alertHandler.Acknowledge)
			alerts.POST("/:id/resolve", alertHandler.Resolve)
			alerts.POST("/:id/silence", alertHandler.Silence)
		}

		// 告警规则管理
		alertRules := adminGroup.Group("/alert-rules")
		{
			alertRules.POST("", alertRuleHandler.Create)
			alertRules.GET("", alertRuleHandler.List)
			alertRules.GET("/:id", alertRuleHandler.Get)
			alertRules.PUT("/:id", alertRuleHandler.Update)
			alertRules.DELETE("/:id", alertRuleHandler.Delete)
			alertRules.POST("/:id/toggle", alertRuleHandler.Toggle)
		}

		// 通知渠道管理
		notificationChannels := adminGroup.Group("/notification-channels")
		{
			notificationChannels.POST("", notificationChannelHandler.Create)
			notificationChannels.GET("", notificationChannelHandler.List)
			notificationChannels.GET("/:id", notificationChannelHandler.Get)
			notificationChannels.PUT("/:id", notificationChannelHandler.Update)
			notificationChannels.DELETE("/:id", notificationChannelHandler.Delete)
			notificationChannels.POST("/:id/toggle", notificationChannelHandler.Toggle)
			notificationChannels.POST("/:id/test", notificationChannelHandler.Test)
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

func startCronTasks() (*cron.Cron, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	taskScheduler := cron.New(
		cron.WithSeconds(),
		cron.WithLocation(time.Local),
	)

	trafficService := services.NewTrafficService(database.DB)
	vipService := services.NewVIPService(database.DB)
	nodeGroupService := services.NewNodeGroupService(database.DB)

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

	if err := addTask("*/30 * * * * *", "node_instances.heartbeat_timeout_check", func() error {
		offlineCount, runErr := nodeGroupService.MarkOfflineByHeartbeat(3 * time.Minute)
		if runErr != nil {
			return runErr
		}
		zap.L().Info("节点实例心跳超时检查完成", zap.Int64("offline_instances", offlineCount))
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

	// 节点健康检查任务 - 每分钟执行一次
	nodeHealthService := services.NewNodeHealthService(database.DB)
	if err := addTask("0 * * * * *", "node_health.batch_check", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if runErr := nodeHealthService.BatchPerformHealthCheck(ctx); runErr != nil {
			return runErr
		}
		zap.L().Info("节点健康检查完成")
		return nil
	}); err != nil {
		return nil, err
	}

	// 清理旧的健康检查记录 - 每天凌晨 2 点执行
	if err := addTask("0 0 2 * * *", "node_health.cleanup_old_records", func() error {
		rows, runErr := nodeHealthService.CleanupOldRecords(30)
		if runErr != nil {
			return runErr
		}
		zap.L().Info("健康检查记录清理完成", zap.Int64("deleted_rows", rows))
		return nil
	}); err != nil {
		return nil, err
	}

	// 性能指标聚合任务 - 每小时执行一次
	nodePerformanceService := services.NewNodePerformanceService(database.DB)
	if err := addTask("0 0 * * * *", "node_performance.aggregate_hourly", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if runErr := nodePerformanceService.AggregateMetrics(ctx, "hourly"); runErr != nil {
			return runErr
		}
		zap.L().Info("性能指标小时聚合完成")
		return nil
	}); err != nil {
		return nil, err
	}

	// 性能指标聚合任务 - 每天凌晨 1 点执行
	if err := addTask("0 0 1 * * *", "node_performance.aggregate_daily", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if runErr := nodePerformanceService.AggregateMetrics(ctx, "daily"); runErr != nil {
			return runErr
		}
		zap.L().Info("性能指标每日聚合完成")
		return nil
	}); err != nil {
		return nil, err
	}

	// 清理旧的性能指标 - 每天凌晨 3 点执行
	if err := addTask("0 0 3 * * *", "node_performance.cleanup_old_metrics", func() error {
		rows, runErr := nodePerformanceService.CleanupOldMetrics(7)
		if runErr != nil {
			return runErr
		}
		zap.L().Info("性能指标清理完成", zap.Int64("deleted_rows", rows))
		return nil
	}); err != nil {
		return nil, err
	}

	// 清理旧的性能汇总 - 每天凌晨 3 点执行
	if err := addTask("0 0 3 * * *", "node_performance.cleanup_old_summaries", func() error {
		rows, runErr := nodePerformanceService.CleanupOldSummaries(90)
		if runErr != nil {
			return runErr
		}
		zap.L().Info("性能汇总清理完成", zap.Int64("deleted_rows", rows))
		return nil
	}); err != nil {
		return nil, err
	}

	taskScheduler.Start()
	zap.L().Info("定时任务调度器已启动")
	return taskScheduler, nil
}

func cleanupExpiredAuditLogs() (int64, error) {
	expireAt := time.Now().AddDate(0, 0, -90)
	result := database.DB.Where("created_at < ?", expireAt).Delete(&models.AuditLog{})
	if result.Error != nil {
		return 0, fmt.Errorf("清理审计日志失败: %w", result.Error)
	}
	return result.RowsAffected, nil
}
