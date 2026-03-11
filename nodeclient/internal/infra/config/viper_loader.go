package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

var warnLogger = log.New(os.Stdout, "[config] ", log.LstdFlags)

// SetWarnLogger 设置配置模块警告日志输出（测试或嵌入场景可覆写）。
func SetWarnLogger(logger *log.Logger) {
	if logger == nil {
		return
	}
	warnLogger = logger
}

// Config 定义 nodeclient 运行所需配置。
type Config struct {
	// 节点组配置
	GroupID     uint   `json:"group_id" yaml:"group_id" mapstructure:"group_id"`
	NodeID      string `json:"node_id" yaml:"node_id" mapstructure:"node_id"`
	ServiceName string `json:"service_name" yaml:"service_name" mapstructure:"service_name"`

	// 网络配置
	ConnectionAddress string `json:"connection_address" yaml:"connection_address" mapstructure:"connection_address"`
	ExitNetwork       string `json:"exit_network" yaml:"exit_network" mapstructure:"exit_network"`

	// 运行配置
	DebugMode bool `json:"debug_mode" yaml:"debug_mode" mapstructure:"debug_mode"`
	AutoStart bool `json:"auto_start" yaml:"auto_start" mapstructure:"auto_start"`

	// 面板连接
	HubURL string `json:"hub_url" yaml:"hub_url" mapstructure:"hub_url"`

	// 授权与版本统一校验（可选）
	LicenseEnabled   bool   `json:"license_enabled,omitempty" yaml:"license_enabled,omitempty" mapstructure:"license_enabled"`
	LicenseVerifyURL string `json:"license_verify_url,omitempty" yaml:"license_verify_url,omitempty" mapstructure:"license_verify_url"`
	LicenseKey       string `json:"license_key,omitempty" yaml:"license_key,omitempty" mapstructure:"license_key"`
	LicenseMachineID string `json:"license_machine_id,omitempty" yaml:"license_machine_id,omitempty" mapstructure:"license_machine_id"`
	LicenseProduct   string `json:"license_product,omitempty" yaml:"license_product,omitempty" mapstructure:"license_product"`
	LicenseChannel   string `json:"license_channel,omitempty" yaml:"license_channel,omitempty" mapstructure:"license_channel"`
	LicenseTimeout   int    `json:"license_timeout,omitempty" yaml:"license_timeout,omitempty" mapstructure:"license_timeout"`
	LicenseFailOpen  bool   `json:"license_fail_open,omitempty" yaml:"license_fail_open,omitempty" mapstructure:"license_fail_open"`

	// 缓存路径
	CachePath string `json:"cache_path" yaml:"cache_path" mapstructure:"cache_path"`

	// 兼容旧字段（保留，避免现有逻辑中断）
	NodeToken             string `json:"node_token,omitempty" yaml:"node_token,omitempty" mapstructure:"node_token"`
	NodeRole              string `json:"node_role,omitempty" yaml:"node_role,omitempty" mapstructure:"node_role"`
	ConnectHost           string `json:"connect_host,omitempty" yaml:"connect_host,omitempty" mapstructure:"connect_host"`
	Debug                 bool   `json:"debug,omitempty" yaml:"debug,omitempty" mapstructure:"debug"`
	EgressInterface       string `json:"egress_interface,omitempty" yaml:"egress_interface,omitempty" mapstructure:"egress_interface"`
	LogPath               string `json:"log_path,omitempty" yaml:"log_path,omitempty" mapstructure:"log_path"`
	HeartbeatInterval     int    `json:"heartbeat_interval,omitempty" yaml:"heartbeat_interval,omitempty" mapstructure:"heartbeat_interval"`
	ConfigCheckInterval   int    `json:"config_check_interval,omitempty" yaml:"config_check_interval,omitempty" mapstructure:"config_check_interval"`
	TrafficReportInterval int    `json:"traffic_report_interval,omitempty" yaml:"traffic_report_interval,omitempty" mapstructure:"traffic_report_interval"`
}

// CLIOverrides 定义命令行覆盖配置项。
type CLIOverrides struct {
	HubURL      string
	NodeID      string
	GroupID     uint
	ServiceName string
	Token       string
}

// Load 使用 Viper 加载配置文件并应用命令行覆盖参数。
func Load(configPath string, overrides CLIOverrides) (*Config, error) {
	path := strings.TrimSpace(configPath)
	if path == "" {
		path = "/etc/nodeclient/config.yaml"
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// 默认值（优先级：CLI > 配置文件 > 默认值）
	v.SetDefault("cache_path", "/var/lib/nodeclient/config_cache.json")
	v.SetDefault("auto_start", true)
	v.SetDefault("debug_mode", false)
	v.SetDefault("connection_address", "auto")
	v.SetDefault("service_name", "nodeclient")
	v.SetDefault("heartbeat_interval", 30)
	v.SetDefault("config_check_interval", 60)
	v.SetDefault("traffic_report_interval", 60)
	v.SetDefault("license_enabled", false)
	v.SetDefault("license_product", "nodeclient")
	v.SetDefault("license_channel", "stable")
	v.SetDefault("license_timeout", 10)
	v.SetDefault("license_fail_open", false)

	if err := v.ReadInConfig(); err != nil {
		// 配置文件不存在时，仅使用默认值 + CLI 覆盖。
		var notFoundErr viper.ConfigFileNotFoundError
		if !errors.As(err, &notFoundErr) && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if hubURL := strings.TrimSpace(overrides.HubURL); hubURL != "" {
		cfg.HubURL = hubURL
	}
	if nodeID := strings.TrimSpace(overrides.NodeID); nodeID != "" {
		cfg.NodeID = nodeID
	}
	if overrides.GroupID > 0 {
		cfg.GroupID = overrides.GroupID
	}
	if serviceName := strings.TrimSpace(overrides.ServiceName); serviceName != "" {
		cfg.ServiceName = serviceName
	}
	if token := strings.TrimSpace(overrides.Token); token != "" {
		cfg.NodeToken = token
	}

	// 新旧字段对齐，尽量兼容已有逻辑
	if strings.TrimSpace(cfg.ConnectHost) == "" {
		cfg.ConnectHost = strings.TrimSpace(cfg.ConnectionAddress)
	}
	if strings.TrimSpace(cfg.ConnectionAddress) == "" {
		cfg.ConnectionAddress = strings.TrimSpace(cfg.ConnectHost)
	}
	if cfg.DebugMode {
		cfg.Debug = true
	}
	if cfg.Debug {
		cfg.DebugMode = true
	}

	if strings.TrimSpace(cfg.CachePath) == "" {
		cfg.CachePath = "/var/lib/nodeclient/config_cache.json"
	}
	if strings.TrimSpace(cfg.LicenseProduct) == "" {
		cfg.LicenseProduct = "nodeclient"
	}
	if strings.TrimSpace(cfg.LicenseChannel) == "" {
		cfg.LicenseChannel = "stable"
	}
	if cfg.LicenseTimeout <= 0 {
		cfg.LicenseTimeout = 10
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
	if strings.TrimSpace(c.NodeID) == "" {
		return fmt.Errorf("node_id 不能为空")
	}
	// group_id 为 0 时给出警告，但不阻止启动（兼容旧配置）
	if c.GroupID == 0 {
		warnLogger.Printf("[WARN] group_id 未配置或为 0，节点可能无法正常工作。请在配置文件中设置 group_id")
	}
	if strings.TrimSpace(c.ServiceName) == "" {
		return fmt.Errorf("service_name 不能为空")
	}
	if strings.TrimSpace(c.NodeToken) == "" {
		return fmt.Errorf("node_token 不能为空")
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
