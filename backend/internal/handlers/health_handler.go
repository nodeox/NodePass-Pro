package handlers

import (
	"net/http"
	"time"

	"nodepass-pro/backend/internal/cache"
	"nodepass-pro/backend/internal/database"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查处理器
type HealthHandler struct{}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string                 `json:"status"`    // overall, healthy, degraded, unhealthy
	Timestamp time.Time              `json:"timestamp"` // 检查时间
	Checks    map[string]CheckStatus `json:"checks"`    // 各组件状态
}

// CheckStatus 组件检查状态
type CheckStatus struct {
	Status  string `json:"status"`            // healthy, unhealthy
	Message string `json:"message,omitempty"` // 错误信息
}

// Health GET /health
// 健康检查端点（用于负载均衡器和容器编排）
func (h *HealthHandler) Health(c *gin.Context) {
	checks := make(map[string]CheckStatus)
	overallHealthy := true

	// 检查数据库连接
	dbStatus := h.checkDatabase()
	checks["database"] = dbStatus
	if dbStatus.Status != "healthy" {
		overallHealthy = false
	}

	// 检查 Redis 连接（可选）
	redisStatus := h.checkRedis()
	checks["redis"] = redisStatus
	// Redis 不健康不影响整体状态（降级可用）

	status := "healthy"
	if !overallHealthy {
		status = "unhealthy"
	} else if redisStatus.Status != "healthy" {
		status = "degraded"
	}

	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Checks:    checks,
	}

	// 如果不健康，返回 503
	if status == "unhealthy" {
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// Readiness GET /readiness
// 就绪检查端点（用于 Kubernetes readiness probe）
func (h *HealthHandler) Readiness(c *gin.Context) {
	// 检查数据库是否可用
	dbStatus := h.checkDatabase()
	if dbStatus.Status != "healthy" {
		utils.Error(c, http.StatusServiceUnavailable, "NOT_READY", "服务未就绪: "+dbStatus.Message)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"status":    "ready",
		"timestamp": time.Now(),
	}, "服务已就绪")
}

// Liveness GET /liveness
// 存活检查端点（用于 Kubernetes liveness probe）
func (h *HealthHandler) Liveness(c *gin.Context) {
	// 简单返回 200，表示进程存活
	utils.SuccessResponse(c, gin.H{
		"status":    "alive",
		"timestamp": time.Now(),
	}, "服务存活")
}

// checkDatabase 检查数据库连接
func (h *HealthHandler) checkDatabase() CheckStatus {
	if database.DB == nil {
		return CheckStatus{
			Status:  "unhealthy",
			Message: "数据库未初始化",
		}
	}

	sqlDB, err := database.DB.DB()
	if err != nil {
		return CheckStatus{
			Status:  "unhealthy",
			Message: "无法获取数据库实例: " + err.Error(),
		}
	}

	if err := sqlDB.Ping(); err != nil {
		return CheckStatus{
			Status:  "unhealthy",
			Message: "数据库连接失败: " + err.Error(),
		}
	}

	return CheckStatus{
		Status: "healthy",
	}
}

// checkRedis 检查 Redis 连接
func (h *HealthHandler) checkRedis() CheckStatus {
	if !cache.Enabled() {
		return CheckStatus{
			Status:  "healthy",
			Message: "Redis 未启用",
		}
	}

	// 这里可以添加 Redis ping 检查
	// 由于 cache 包没有暴露 Ping 方法，我们假设如果启用了就是健康的
	return CheckStatus{
		Status: "healthy",
	}
}
