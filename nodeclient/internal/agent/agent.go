package agent

import (
	"nodepass-pro/nodeclient/internal/app/manager"
	"nodepass-pro/nodeclient/internal/infra/config"
)

var clientVersion = "1.0.1"

// Version 返回节点客户端版本号。
func Version() string {
	return clientVersion
}

// Agent 定义节点客户端核心控制器（薄包装层，委托给 app/manager）。
type Agent struct {
	manager manager.Manager
}

// NewAgent 创建 Agent。
func NewAgent(cfg *config.Config) *Agent {
	return &Agent{
		manager: manager.NewManager(cfg, clientVersion),
	}
}

// Start 启动 Agent 并阻塞等待退出信号。
func (a *Agent) Start() error {
	return a.manager.Start()
}

// Shutdown 停止所有服务并释放资源。
func (a *Agent) Shutdown() {
	a.manager.Shutdown()
}

