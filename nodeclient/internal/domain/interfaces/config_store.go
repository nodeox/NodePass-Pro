package interfaces

import "nodepass-pro/nodeclient/internal/domain/config"

// ConfigStore 定义配置缓存存取接口。
type ConfigStore interface {
	Load() (*config.NodeConfig, error)
	Save(*config.NodeConfig) error
	Exists() bool
	GetVersion() int
}
