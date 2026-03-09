package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"nodepass-license-center/internal/cache"
	"nodepass-license-center/internal/config"
	"nodepass-license-center/internal/database"
	"nodepass-license-center/internal/handlers"
	"nodepass-license-center/internal/middleware"
	"nodepass-license-center/internal/services"
	"nodepass-license-center/internal/utils"
	"nodepass-license-center/internal/web"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

var appVersion = "1.0.0"

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(configFile)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Init(cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 初始化 Redis 缓存（可选）
	var redisCache *cache.RedisCache
	if cfg.Redis.Enabled {
		redisCache, err = cache.NewRedisCache(
			fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
			cfg.Redis.Password,
			cfg.Redis.DB,
			cfg.Redis.Prefix,
		)
		if err != nil {
			log.Printf("Redis 连接失败，将不使用缓存: %v", err)
		} else {
			log.Printf("Redis 缓存已启用")
		}
	}

	// 初始化服务
	authService := services.NewAuthService(db, cfg.JWT.Secret, cfg.JWT.ExpireHours)
	webhookService := services.NewWebhookService(db)
	alertService := services.NewAlertService(db, redisCache, webhookService)
	domainBindingService := services.NewDomainBindingService(db, webhookService, alertService)
	licenseService := services.NewLicenseService(db, domainBindingService)
	extensionService := services.NewExtensionService(db, webhookService)
	monitoringService := services.NewMonitoringService(db, redisCache)
	licenseTemplateService := services.NewLicenseTemplateService(db)
	licenseGroupService := services.NewLicenseGroupService(db)
	licenseEnhancedService := services.NewLicenseEnhancedService(db)

	// 启动时过期清理
	if affected, expireErr := licenseService.ExpireOverdueLicenses(); expireErr != nil {
		log.Printf("启动时过期清理失败: %v", expireErr)
	} else if affected > 0 {
		log.Printf("启动时已标记过期授权: %d", affected)
	}

	// 定时任务：过期清理
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			affected, expireErr := licenseService.ExpireOverdueLicenses()
			if expireErr != nil {
				log.Printf("定时过期清理失败: %v", expireErr)
				continue
			}
			if affected > 0 {
				log.Printf("定时标记过期授权: %d", affected)
			}
		}
	}()

	// 定时任务：告警检查
	if cfg.Monitoring.Alert.Enabled {
		go func() {
			interval := time.Duration(cfg.Monitoring.Alert.CheckInterval) * time.Second
			if interval < 60*time.Second {
				interval = 60 * time.Second
			}
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for range ticker.C {
				// 检查即将过期的授权码
				if err := alertService.CheckExpiringLicenses(cfg.Monitoring.Alert.ExpiringDays); err != nil {
					log.Printf("检查过期告警失败: %v", err)
				}

				// 检查配额超限
				if err := alertService.CheckQuotaExceeded(); err != nil {
					log.Printf("检查配额告警失败: %v", err)
				}
			}
		}()
		log.Printf("告警监控已启用，检查间隔: %d 秒", cfg.Monitoring.Alert.CheckInterval)
	}

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(authService)
	licenseHandler := handlers.NewLicenseHandler(licenseService)
	extensionHandler := handlers.NewExtensionHandler(extensionService, webhookService)
	monitoringHandler := handlers.NewMonitoringHandler(monitoringService, alertService)
	domainBindingHandler := handlers.NewDomainBindingHandler(domainBindingService)
	versionHandler := handlers.NewVersionHandler(db)
	licenseTemplateHandler := handlers.NewLicenseTemplateHandler(licenseTemplateService)
	licenseGroupHandler := handlers.NewLicenseGroupHandler(licenseGroupService)
	licenseEnhancedHandler := handlers.NewLicenseEnhancedHandler(licenseEnhancedService)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// 限流中间件
	if cfg.Security.RateLimit.Enabled {
		limiter := middleware.NewRateLimiter(
			rate.Limit(cfg.Security.RateLimit.RequestsPerSecond),
			cfg.Security.RateLimit.Burst,
		)
		limiter.CleanupOldLimiters()
		r.Use(middleware.RateLimitMiddleware(limiter))
		log.Printf("限流已启用: %d req/s, burst: %d", cfg.Security.RateLimit.RequestsPerSecond, cfg.Security.RateLimit.Burst)
	}

	// IP 白名单中间件
	if cfg.Security.IPWhitelist.Enabled {
		ipConfig, err := middleware.NewIPWhitelistConfig(
			cfg.Security.IPWhitelist.AllowedIPs,
			cfg.Security.IPWhitelist.AllowedCIDRs,
		)
		if err != nil {
			log.Fatalf("IP 白名单配置错误: %v", err)
		}
		ipConfig.SkipPaths = []string{"/health", "/"}
		r.Use(middleware.IPWhitelistMiddleware(ipConfig))
		log.Printf("IP 白名单已启用")
	}

	r.GET("/console", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", web.ConsoleIndexHTML)
	})
	r.GET("/console/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", web.ConsoleIndexHTML)
	})

	r.GET("/", func(c *gin.Context) {
		utils.Success(c, gin.H{
			"name":    "NodePass License Center",
			"version": appVersion,
			"health":  "/health",
			"api":     "/api/v1",
			"console": "/console",
		}, "service is running")
	})

	r.GET("/health", func(c *gin.Context) {
		utils.Success(c, gin.H{"status": "ok", "version": appVersion}, "ok")
	})

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", authHandler.Login)
		if cfg.Security.Signature.Enabled {
			var nonceStore middleware.NonceStore = middleware.NewMemoryNonceStore()
			if redisCache != nil {
				nonceStore = middleware.NewRedisNonceStore(
					redisCache.GetClient(),
					fmt.Sprintf("%ssignature:nonce:", cfg.Redis.Prefix),
				)
				log.Printf("签名防重放使用 Redis Nonce 存储")
			} else {
				log.Printf("签名防重放使用内存 Nonce 存储（单实例）")
			}
			apiSigned := api.Group("")
			apiSigned.Use(middleware.SignatureMiddleware(middleware.SignatureConfig{
				Secret:     cfg.Security.Signature.Secret,
				TimeWindow: cfg.Security.Signature.TimeWindow,
				NonceStore: nonceStore,
			}))
			apiSigned.POST("/license/verify", licenseHandler.Verify)
			log.Printf("签名校验已启用")
		} else {
			api.POST("/license/verify", licenseHandler.Verify)
		}
	}

	admin := api.Group("")
	admin.Use(middleware.AdminAuth(authService))
	{
		// 认证
		admin.GET("/auth/me", authHandler.Me)

		// 套餐管理
		admin.GET("/plans", licenseHandler.ListPlans)
		admin.POST("/plans", licenseHandler.CreatePlan)
		admin.PUT("/plans/:id", licenseHandler.UpdatePlan)
		admin.DELETE("/plans/:id", licenseHandler.DeletePlan)

		// 授权码管理
		admin.POST("/licenses/generate", licenseHandler.GenerateLicenses)
		admin.GET("/licenses", licenseHandler.ListLicenses)

		// 授权码增强功能 - 必须在 :id 路由之前注册
		admin.POST("/licenses/batch/update-enhanced", licenseEnhancedHandler.BatchUpdate)
		admin.POST("/licenses/batch/transfer", licenseEnhancedHandler.BatchTransfer)
		admin.POST("/licenses/batch/revoke-enhanced", licenseEnhancedHandler.BatchRevoke)
		admin.POST("/licenses/batch/restore-enhanced", licenseEnhancedHandler.BatchRestore)
		admin.POST("/licenses/batch/delete-enhanced", licenseEnhancedHandler.BatchDelete)
		admin.POST("/licenses/search/advanced", licenseEnhancedHandler.AdvancedSearch)
		admin.POST("/licenses/search/save", licenseEnhancedHandler.SaveSearch)
		admin.GET("/licenses/search/saved", licenseEnhancedHandler.ListSavedSearches)
		admin.GET("/licenses/search/saved/:id", licenseEnhancedHandler.GetSavedSearch)
		admin.DELETE("/licenses/search/saved/:id", licenseEnhancedHandler.DeleteSavedSearch)
		admin.GET("/licenses/statistics", licenseEnhancedHandler.GetStatistics)
		admin.GET("/licenses/expiring", licenseEnhancedHandler.GetExpiringLicenses)

		admin.GET("/licenses/:id", licenseHandler.GetLicense)
		admin.PUT("/licenses/:id", licenseHandler.UpdateLicense)
		admin.DELETE("/licenses/:id", licenseHandler.DeleteLicense)
		admin.POST("/licenses/:id/revoke", licenseHandler.RevokeLicense)
		admin.POST("/licenses/:id/restore", licenseHandler.RestoreLicense)
		admin.GET("/licenses/:id/activations", licenseHandler.ListActivations)
		admin.DELETE("/licenses/:id/activations/:activationId", licenseHandler.UnbindActivation)
		admin.GET("/licenses/:id/groups", licenseGroupHandler.GetLicenseGroups)

		// 授权码扩展功能
		admin.POST("/licenses/:id/transfer", extensionHandler.TransferLicense)
		admin.GET("/licenses/:id/tags", extensionHandler.GetLicenseTags)
		admin.POST("/licenses/:id/tags", extensionHandler.AddTagsToLicense)
		admin.DELETE("/licenses/:id/tags", extensionHandler.RemoveTagsFromLicense)
		admin.POST("/licenses/batch/update", extensionHandler.BatchUpdateLicenses)
		admin.POST("/licenses/batch/revoke", extensionHandler.BatchRevokeLicenses)
		admin.POST("/licenses/batch/restore", extensionHandler.BatchRestoreLicenses)
		admin.POST("/licenses/batch/delete", extensionHandler.BatchDeleteLicenses)

		// 标签管理
		admin.GET("/tags", extensionHandler.ListTags)
		admin.POST("/tags", extensionHandler.CreateTag)
		admin.PUT("/tags/:id", extensionHandler.UpdateTag)
		admin.DELETE("/tags/:id", extensionHandler.DeleteTag)

		// Webhook 管理
		admin.GET("/webhooks", extensionHandler.ListWebhooks)
		admin.POST("/webhooks", extensionHandler.CreateWebhook)
		admin.PUT("/webhooks/:id", extensionHandler.UpdateWebhook)
		admin.DELETE("/webhooks/:id", extensionHandler.DeleteWebhook)
		admin.GET("/webhook-logs", extensionHandler.ListWebhookLogs)

		// 监控与统计
		admin.GET("/dashboard", monitoringHandler.GetDashboard)
		admin.GET("/verify-trend", monitoringHandler.GetVerifyTrend)
		admin.GET("/top-customers", monitoringHandler.GetTopCustomers)
		admin.GET("/stats", licenseHandler.Stats)

		// 告警管理
		admin.GET("/alerts", monitoringHandler.ListAlerts)
		admin.POST("/alerts/:id/read", monitoringHandler.MarkAlertRead)
		admin.POST("/alerts/read-all", monitoringHandler.MarkAllAlertsRead)
		admin.DELETE("/alerts/:id", monitoringHandler.DeleteAlert)
		admin.GET("/alert-stats", monitoringHandler.GetAlertStats)

		// 日志查询
		admin.GET("/verify-logs", licenseHandler.ListVerifyLogs)

		// 版本管理
		admin.GET("/versions/system", versionHandler.GetSystemVersionInfo)
		admin.GET("/versions/components/:component", versionHandler.GetComponentVersion)
		admin.POST("/versions/components", versionHandler.UpdateComponentVersion)
		admin.GET("/versions/components/:component/history", versionHandler.ListComponentVersions)
		admin.GET("/versions/compatibility/:version", versionHandler.CheckCompatibility)
		admin.GET("/versions/compatibility", versionHandler.ListCompatibilityConfigs)
		admin.POST("/versions/compatibility", versionHandler.CreateCompatibilityConfig)

		// 授权码模板管理
		admin.GET("/license-templates", licenseTemplateHandler.ListTemplates)
		admin.GET("/license-templates/:id", licenseTemplateHandler.GetTemplate)
		admin.POST("/license-templates", licenseTemplateHandler.CreateTemplate)
		admin.PUT("/license-templates/:id", licenseTemplateHandler.UpdateTemplate)
		admin.DELETE("/license-templates/:id", licenseTemplateHandler.DeleteTemplate)
		admin.POST("/license-templates/generate", licenseTemplateHandler.GenerateFromTemplate)
		admin.POST("/license-templates/:id/toggle", licenseTemplateHandler.ToggleTemplate)

		// 授权码分组管理
		admin.GET("/license-groups", licenseGroupHandler.ListGroups)
		admin.GET("/license-groups/:id", licenseGroupHandler.GetGroup)
		admin.POST("/license-groups", licenseGroupHandler.CreateGroup)
		admin.PUT("/license-groups/:id", licenseGroupHandler.UpdateGroup)
		admin.DELETE("/license-groups/:id", licenseGroupHandler.DeleteGroup)
		admin.POST("/license-groups/:id/licenses", licenseGroupHandler.AddLicensesToGroup)
		admin.DELETE("/license-groups/:id/licenses", licenseGroupHandler.RemoveLicensesFromGroup)
		admin.GET("/license-groups/:id/licenses", licenseGroupHandler.GetGroupLicenses)
		admin.GET("/license-groups/:id/stats", licenseGroupHandler.GetGroupStats)

		// 域名绑定管理
		admin.POST("/licenses/:id/domain/change", domainBindingHandler.ChangeDomain)
		admin.POST("/licenses/:id/domain/unbind", domainBindingHandler.UnbindDomain)
		admin.POST("/licenses/:id/domain/lock", domainBindingHandler.LockDomain)
		admin.GET("/licenses/:id/domain/history", domainBindingHandler.GetBindingHistory)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("license-center 已启动: :%s version=%s", cfg.Server.Port, appVersion)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
