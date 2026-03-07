package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 定义 nodeclient 运行所需配置。
type Config struct {
	HubURL                string `mapstructure:"hub_url"`
	NodeToken             string `mapstructure:"node_token"`
	CachePath             string `mapstructure:"cache_path"`
	LogPath               string `mapstructure:"log_path"`
	HeartbeatInterval     int    `mapstructure:"heartbeat_interval"`
	ConfigCheckInterval   int    `mapstructure:"config_check_interval"`
	TrafficReportInterval int    `mapstructure:"traffic_report_interval"`
}

// CLIOverrides 定义命令行覆盖配置项。
type CLIOverrides struct {
	HubURL string
	Token  string
}

// Load 使用 Viper 加载配置文件并应用命令行覆盖参数。
func Load(configPath string, overrides CLIOverrides) (*Config, error) {
	path := strings.TrimSpace(configPath)
	if path == "" {
		path = "configs/config.yaml"
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.SetDefault("heartbeat_interval", 30)
	v.SetDefault("config_check_interval", 60)
	v.SetDefault("traffic_report_interval", 60)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if hubURL := strings.TrimSpace(overrides.HubURL); hubURL != "" {
		cfg.HubURL = hubURL
	}
	if token := strings.TrimSpace(overrides.Token); token != "" {
		cfg.NodeToken = token
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate 校验配置必填项和基础范围。
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("配置不能为空")
	}
	if strings.TrimSpace(c.HubURL) == "" {
		return fmt.Errorf("hub_url 不能为空")
	}
	if strings.TrimSpace(c.NodeToken) == "" {
		return fmt.Errorf("node_token 不能为空")
	}
	if strings.TrimSpace(c.CachePath) == "" {
		return fmt.Errorf("cache_path 不能为空")
	}
	if c.HeartbeatInterval <= 0 {
		return fmt.Errorf("heartbeat_interval 必须大于 0")
	}
	if c.ConfigCheckInterval <= 0 {
		return fmt.Errorf("config_check_interval 必须大于 0")
	}
	if c.TrafficReportInterval <= 0 {
		return fmt.Errorf("traffic_report_interval 必须大于 0")
	}
	return nil
}
