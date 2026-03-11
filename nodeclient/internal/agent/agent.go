package agent

import (
	"fmt"

	"nodepass-pro/nodeclient/internal/app/manager"
	"nodepass-pro/nodeclient/internal/domain/interfaces"
	"nodepass-pro/nodeclient/internal/infra/config"
	"nodepass-pro/nodeclient/internal/infra/license"
)

var clientVersion = "1.0.1"

// Version 返回节点客户端版本号。
func Version() string {
	return clientVersion
}

// Agent 定义节点客户端核心控制器（薄包装层，委托给 app/manager）。
type Agent struct {
	manager  manager.Manager
	verifier interfaces.LicenseVerifier
}

// NewAgent 创建 Agent。
func NewAgent(cfg *config.Config) *Agent {
	return &Agent{
		manager:  manager.NewManager(cfg, clientVersion),
		verifier: license.NewVerifier(cfg),
	}
}

// Start 启动 Agent 并阻塞等待退出信号。
func (a *Agent) Start() error {
	if a == nil || a.manager == nil {
		return fmt.Errorf("agent 未初始化")
	}
	if a.verifier != nil && a.verifier.Enabled() {
		if _, err := a.verifier.Verify(clientVersion); err != nil {
			return fmt.Errorf("授权校验失败: %w", err)
		}
	}
	return a.manager.Start()
}

// Shutdown 停止所有服务并释放资源。
func (a *Agent) Shutdown() {
	a.manager.Shutdown()
}
