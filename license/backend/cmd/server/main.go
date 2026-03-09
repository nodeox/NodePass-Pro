package main

import (
	"log"
	"strings"
	"time"

	"nodepass-license-unified/internal/config"
	"nodepass-license-unified/internal/database"
	"nodepass-license-unified/internal/handlers"
	"nodepass-license-unified/internal/router"
	"nodepass-license-unified/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	if strings.TrimSpace(cfg.Bootstrap.AdminPassword) == "" {
		log.Fatal("缺少 BOOTSTRAP_ADMIN_PASSWORD，请在环境变量或 .env 中显式设置管理员初始密码")
	}
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Init(cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	authService := services.NewAuthService(db, cfg.JWT.Secret, cfg.JWT.ExpireHours)
	unifiedService := services.NewUnifiedService(db)
	enhancedService := services.NewLicenseEnhancedService(db, unifiedService)
	callbackVerifier := services.NewPaymentCallbackVerifier(
		cfg.Payment.CallbackStrict,
		cfg.Payment.CallbackToleranceSeconds,
		map[string]string{
			"default": cfg.Payment.CallbackSecretDefault,
			"manual":  cfg.Payment.CallbackSecretManual,
			"alipay":  cfg.Payment.CallbackSecretAlipay,
			"wechat":  cfg.Payment.CallbackSecretWechat,
		},
	)
	commercialService := services.NewCommercialService(db, callbackVerifier)

	authHandler := handlers.NewAuthHandler(authService)
	unifiedHandler := handlers.NewUnifiedHandler(unifiedService)
	unifiedHandler.SetReleaseUploadDir(cfg.Storage.ReleaseUploadDir)
	enhancedHandler := handlers.NewLicenseEnhancedHandler(enhancedService)
	commercialHandler := handlers.NewCommercialHandler(commercialService)

	r := router.Setup(authHandler, unifiedHandler, enhancedHandler, commercialHandler, authService)

	go func() {
		if err := unifiedService.RunAutoVersionMirrorSync(); err != nil {
			log.Printf("启动后自动版本镜像同步失败: %v", err)
		}
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := unifiedService.RunAutoVersionMirrorSync(); err != nil {
				log.Printf("自动版本镜像同步失败: %v", err)
			}
		}
	}()

	log.Printf("license-unified 服务已启动: :%s", cfg.Server.Port)
	if err = r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
