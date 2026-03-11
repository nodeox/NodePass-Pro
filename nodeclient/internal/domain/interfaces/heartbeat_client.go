package interfaces

import (
	"context"

	"nodepass-pro/nodeclient/internal/domain/heartbeat"
)

// HeartbeatClient 定义心跳上报接口。
type HeartbeatClient interface {
	Report(ctx context.Context, payload heartbeat.HeartbeatPayload) (heartbeat.HeartbeatResponseData, error)
}
