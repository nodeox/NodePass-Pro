package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TunnelTemplate 隧道配置模板。
type TunnelTemplate struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      uint      `gorm:"column:user_id;not null;index:idx_tunnel_templates_user_id" json:"user_id"`
	Name        string    `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Description *string   `gorm:"column:description;type:text" json:"description"`
	Protocol    string    `gorm:"column:protocol;type:varchar(20);not null" json:"protocol"`
	ConfigJSON  string    `gorm:"column:config_json;type:text;not null" json:"config_json"`
	IsPublic    bool      `gorm:"column:is_public;not null;default:false;index:idx_tunnel_templates_is_public" json:"is_public"`
	UsageCount  int       `gorm:"column:usage_count;not null;default:0" json:"usage_count"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
}

// TableName 指定表名。
func (TunnelTemplate) TableName() string {
	return "tunnel_templates"
}

// TunnelTemplateConfig 模板配置内容。
type TunnelTemplateConfig struct {
	ListenHost          *string       `json:"listen_host,omitempty"`
	ListenPort          *int          `json:"listen_port,omitempty"`
	RemoteHost          string        `json:"remote_host"`
	RemotePort          int           `json:"remote_port"`
	LoadBalanceStrategy string        `json:"load_balance_strategy"`
	IPType              string        `json:"ip_type"`
	EnableProxyProtocol bool          `json:"enable_proxy_protocol"`
	ForwardTargets      []ForwardTarget `json:"forward_targets"`
	HealthCheckInterval int           `json:"health_check_interval"`
	HealthCheckTimeout  int           `json:"health_check_timeout"`
	ProtocolConfig      *ProtocolConfig `json:"protocol_config,omitempty"`
}

// GetConfig 读取并反序列化模板配置。
func (t *TunnelTemplate) GetConfig() (*TunnelTemplateConfig, error) {
	if t == nil {
		return nil, fmt.Errorf("tunnel template 不能为空")
	}

	raw := strings.TrimSpace(t.ConfigJSON)
	if raw == "" || raw == "{}" {
		return nil, fmt.Errorf("模板配置为空")
	}

	cfg := &TunnelTemplateConfig{}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("解析模板配置失败: %w", err)
	}
	return cfg, nil
}

// SetConfig 序列化并写入模板配置。
func (t *TunnelTemplate) SetConfig(cfg *TunnelTemplateConfig) error {
	if t == nil {
		return fmt.Errorf("tunnel template 不能为空")
	}
	if cfg == nil {
		return fmt.Errorf("模板配置不能为空")
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化模板配置失败: %w", err)
	}
	t.ConfigJSON = string(data)
	return nil
}

// MarshalJSON 自定义 JSON 序列化。
func (t TunnelTemplate) MarshalJSON() ([]byte, error) {
	type Alias TunnelTemplate

	var configObj interface{}
	if t.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(t.ConfigJSON), &configObj); err != nil {
			configObj = t.ConfigJSON
		}
	} else {
		configObj = map[string]interface{}{}
	}

	return json.Marshal(&struct {
		*Alias
		Config interface{} `json:"config"`
	}{
		Alias:  (*Alias)(&t),
		Config: configObj,
	})
}
