package heartbeat

import (
	domainconfig "nodepass-pro/nodeclient/internal/domain/config"
	"nodepass-pro/nodeclient/internal/infra/heartbeat"
	"nodepass-pro/nodeclient/internal/infra/logger"
)

// Coordinator 负责心跳服务的协调和管理。
type Coordinator interface {
	Start()
	Stop()
	IsOnline() bool
	SetConfigUpdateHandler(handler func(*domainconfig.NodeConfig, int))
	SetMetricsProvider(provider heartbeat.RuntimeMetricsProvider)
	SetCurrentConfigVersion(version int)
	Report() error
}

type coordinator struct {
	heartbeat *heartbeat.HeartbeatService
	logger    *logger.Logger
}

// NewCoordinator 创建心跳协调器。
func NewCoordinator(heartbeat *heartbeat.HeartbeatService, logger *logger.Logger) Coordinator {
	return &coordinator{
		heartbeat: heartbeat,
		logger:    logger,
	}
}

// Start 启动心跳服务。
func (c *coordinator) Start() {
	c.heartbeat.Start()
}

// Stop 停止心跳服务。
func (c *coordinator) Stop() {
	c.heartbeat.Stop()
}

// IsOnline 返回心跳服务的在线状态。
func (c *coordinator) IsOnline() bool {
	return c.heartbeat.IsOnline()
}

// SetConfigUpdateHandler 设置配置更新回调。
func (c *coordinator) SetConfigUpdateHandler(handler func(*domainconfig.NodeConfig, int)) {
	c.heartbeat.SetConfigUpdateHandler(handler)
}

// SetMetricsProvider 设置指标提供者。
func (c *coordinator) SetMetricsProvider(provider heartbeat.RuntimeMetricsProvider) {
	c.heartbeat.SetMetricsProvider(provider)
}

// SetCurrentConfigVersion 设置当前配置版本。
func (c *coordinator) SetCurrentConfigVersion(version int) {
	c.heartbeat.SetCurrentConfigVersion(version)
}

// Report 执行一次心跳上报。
func (c *coordinator) Report() error {
	return c.heartbeat.Report()
}
