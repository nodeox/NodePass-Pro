package config

import (
	"encoding/json"

	domainconfig "nodepass-pro/nodeclient/internal/domain/config"
)

// cloneNodeConfig 深拷贝配置对象。
func cloneNodeConfig(cfg *domainconfig.NodeConfig) *domainconfig.NodeConfig {
	if cfg == nil {
		return nil
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return cfg
	}
	result := &domainconfig.NodeConfig{}
	if err := json.Unmarshal(raw, result); err != nil {
		return cfg
	}
	return result
}
