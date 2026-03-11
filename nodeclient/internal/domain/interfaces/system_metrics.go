package interfaces

import (
	"context"

	"nodepass-pro/nodeclient/internal/domain/heartbeat"
)

// SystemMetricsProvider 定义系统指标采集接口。
type SystemMetricsProvider interface {
	Collect(ctx context.Context) (heartbeat.SystemInfoData, error)
}
