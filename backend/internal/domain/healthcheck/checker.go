package healthcheck

import "context"

// Checker 健康检查器接口
type Checker interface {
	// Check 执行健康检查
	Check(ctx context.Context, nodeInstanceID uint, config *HealthCheck) (*HealthRecord, error)
}
