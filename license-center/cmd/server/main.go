package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"nodepass-license-center/internal/config"
	"nodepass-license-center/internal/database"
	"nodepass-license-center/internal/handlers"
	"nodepass-license-center/internal/middleware"
	"nodepass-license-center/internal/services"
	"nodepass-license-center/internal/utils"

	"github.com/gin-gonic/gin"
)

var appVersion = "0.1.0"

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

	authService := services.NewAuthService(db, cfg.JWT.Secret, cfg.JWT.ExpireHours)
	licenseService := services.NewLicenseService(db)

	if affected, expireErr := licenseService.ExpireOverdueLicenses(); expireErr != nil {
		log.Printf("启动时过期清理失败: %v", expireErr)
	} else if affected > 0 {
		log.Printf("启动时已标记过期授权: %d", affected)
	}

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

	authHandler := handlers.NewAuthHandler(authService)
	licenseHandler := handlers.NewLicenseHandler(licenseService)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.GET("/", func(c *gin.Context) {
		utils.Success(c, gin.H{
			"name":    "NodePass License Center",
			"version": appVersion,
			"health":  "/health",
			"api":     "/api/v1",
		}, "service is running")
	})

	r.GET("/health", func(c *gin.Context) {
		utils.Success(c, gin.H{"status": "ok", "version": appVersion}, "ok")
	})

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", authHandler.Login)
		api.POST("/license/verify", licenseHandler.Verify)
	}

	admin := api.Group("")
	admin.Use(middleware.AdminAuth(authService))
	{
		admin.GET("/auth/me", authHandler.Me)

		admin.GET("/plans", licenseHandler.ListPlans)
		admin.POST("/plans", licenseHandler.CreatePlan)
		admin.PUT("/plans/:id", licenseHandler.UpdatePlan)
		admin.DELETE("/plans/:id", licenseHandler.DeletePlan)

		admin.POST("/licenses/generate", licenseHandler.GenerateLicenses)
		admin.GET("/licenses", licenseHandler.ListLicenses)
		admin.GET("/licenses/:id", licenseHandler.GetLicense)
		admin.PUT("/licenses/:id", licenseHandler.UpdateLicense)
		admin.DELETE("/licenses/:id", licenseHandler.DeleteLicense)
		admin.POST("/licenses/:id/revoke", licenseHandler.RevokeLicense)
		admin.POST("/licenses/:id/restore", licenseHandler.RestoreLicense)
		admin.GET("/licenses/:id/activations", licenseHandler.ListActivations)
		admin.DELETE("/licenses/:id/activations/:activationId", licenseHandler.UnbindActivation)

		admin.GET("/verify-logs", licenseHandler.ListVerifyLogs)
		admin.GET("/stats", licenseHandler.Stats)
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
