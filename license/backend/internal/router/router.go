package router

import (
	"net/http"

	"nodepass-license-unified/internal/handlers"
	"nodepass-license-unified/internal/middleware"
	"nodepass-license-unified/internal/services"
	"nodepass-license-unified/internal/utils"

	"github.com/gin-gonic/gin"
)

// Setup 初始化路由。
func Setup(authHandler *handlers.AuthHandler, unifiedHandler *handlers.UnifiedHandler, enhancedHandler *handlers.LicenseEnhancedHandler, commercialHandler *handlers.CommercialHandler, authService *services.AuthService) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/", func(c *gin.Context) {
		utils.Success(c, gin.H{
			"name":    "NodePass License Unified",
			"version": "1.0.0",
			"health":  "/health",
			"api":     "/api/v1",
		}, "service is running")
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", authHandler.Login)
		api.POST("/verify", unifiedHandler.Verify)
		api.POST("/commercial/payments/callback/:channel", commercialHandler.PaymentCallback)
	}

	admin := api.Group("")
	admin.Use(middleware.Auth(authService))
	{
		admin.GET("/auth/me", authHandler.Me)

		admin.GET("/dashboard", unifiedHandler.Dashboard)

		admin.GET("/plans", unifiedHandler.ListPlans)
		admin.POST("/plans", unifiedHandler.CreatePlan)
		admin.PUT("/plans/:id", unifiedHandler.UpdatePlan)
		admin.POST("/plans/:id/clone", unifiedHandler.ClonePlan)
		admin.DELETE("/plans/:id", unifiedHandler.DeletePlan)

		admin.POST("/licenses/generate", unifiedHandler.GenerateLicenses)
		admin.GET("/licenses", unifiedHandler.ListLicenses)
		admin.GET("/licenses/:id", unifiedHandler.GetLicense)
		admin.PUT("/licenses/:id", unifiedHandler.UpdateLicense)
		admin.DELETE("/licenses/:id", unifiedHandler.DeleteLicense)
		admin.POST("/licenses/:id/revoke", unifiedHandler.RevokeLicense)
		admin.POST("/licenses/:id/restore", unifiedHandler.RestoreLicense)
		admin.GET("/licenses/:id/activations", unifiedHandler.ListActivations)
		admin.DELETE("/licenses/:id/activations/:activation_id", unifiedHandler.UnbindActivation)
		admin.DELETE("/licenses/:id/activations", unifiedHandler.ClearActivations)

		// 增强功能 - 导入导出
		admin.POST("/licenses/advanced/export", enhancedHandler.ExportCSV)

		// 增强功能 - 批量操作
		admin.POST("/licenses/advanced/renew", enhancedHandler.RenewLicenses)
		admin.POST("/licenses/advanced/clone", enhancedHandler.CloneLicense)
		admin.POST("/licenses/batch/delete", unifiedHandler.BatchDeleteLicenses)
		admin.POST("/licenses/batch/update", enhancedHandler.BatchUpdate)
		admin.POST("/licenses/batch/revoke", enhancedHandler.BatchRevoke)
		admin.POST("/licenses/batch/restore", enhancedHandler.BatchRestore)

		// 增强功能 - 报告分析
		admin.GET("/licenses/advanced/:id/usage-report", enhancedHandler.GetUsageReport)
		admin.GET("/licenses/advanced/customer-report", enhancedHandler.GetCustomerReport)

		admin.GET("/releases", unifiedHandler.ListReleases)
		admin.GET("/releases/recycle", unifiedHandler.ListDeletedReleases)
		admin.GET("/version-sync/configs", unifiedHandler.ListVersionSyncConfigs)
		admin.GET("/version-sync/config", unifiedHandler.GetVersionSyncConfig)
		admin.PUT("/version-sync/config", unifiedHandler.UpdateVersionSyncConfig)
		admin.POST("/version-sync/manual", unifiedHandler.ManualSyncVersionMirror)
		admin.POST("/releases", unifiedHandler.CreateRelease)
		admin.PUT("/releases/:id", unifiedHandler.UpdateRelease)
		admin.POST("/releases/upload", unifiedHandler.UploadRelease)
		admin.PUT("/releases/:id/file", unifiedHandler.ReplaceReleasePackage)
		admin.GET("/releases/:id/file", unifiedHandler.DownloadReleaseFile)
		admin.DELETE("/releases/:id", unifiedHandler.DeleteRelease)
		admin.POST("/releases/:id/restore", unifiedHandler.RestoreRelease)
		admin.DELETE("/releases/:id/purge", unifiedHandler.PurgeRelease)

		admin.GET("/version-policies", unifiedHandler.ListVersionPolicies)
		admin.POST("/version-policies", unifiedHandler.CreateVersionPolicy)
		admin.PUT("/version-policies/:id", unifiedHandler.UpdateVersionPolicy)
		admin.DELETE("/version-policies/:id", unifiedHandler.DeleteVersionPolicy)

		admin.GET("/verify-logs", unifiedHandler.ListVerifyLogs)

		// 商业化能力（试用/订单）
		admin.POST("/commercial/trials/issue", commercialHandler.IssueTrial)
		admin.POST("/commercial/orders/renew", commercialHandler.CreateRenewOrder)
		admin.POST("/commercial/orders/upgrade", commercialHandler.CreateUpgradeOrder)
		admin.POST("/commercial/orders/transfer", commercialHandler.CreateTransferOrder)
		admin.GET("/commercial/orders", commercialHandler.ListOrders)
		admin.GET("/commercial/orders/:id", commercialHandler.GetOrder)
		admin.POST("/commercial/orders/:id/mark-paid", commercialHandler.MarkOrderPaid)
	}

	return r
}
