package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"nodepass-panel/backend/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// NodeConfig 下发给节点客户端的配置结构。
type NodeConfig struct {
	ConfigVersion int          `json:"config_version"`
	Rules         []RuleConfig `json:"rules"`
	Settings      Settings     `json:"settings"`
}

// HostPort 主机与端口。
type HostPort struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// RuleConfig 单条规则配置。
type RuleConfig struct {
	RuleID   int       `json:"rule_id"`
	Mode     string    `json:"mode"`
	Listen   HostPort  `json:"listen"`
	ExitNode *HostPort `json:"exit_node,omitempty"`
	Target   HostPort  `json:"target"`
	Protocol string    `json:"protocol"`
}

// Settings 节点运行配置。
type Settings struct {
	HeartbeatInterval   int `json:"heartbeat_interval"`
	ConfigCheckInterval int `json:"config_check_interval"`
}

// RegisterResult 节点注册返回。
type RegisterResult struct {
	NodeID uint        `json:"node_id"`
	Config *NodeConfig `json:"config"`
}

// HeartbeatSystemInfo 心跳上报的系统信息。
type HeartbeatSystemInfo struct {
	CPUUsage     *float64 `json:"cpu_usage"`
	MemoryUsage  *float64 `json:"memory_usage"`
	DiskUsage    *float64 `json:"disk_usage"`
	BandwidthIn  *int64   `json:"bandwidth_in"`
	BandwidthOut *int64   `json:"bandwidth_out"`
	Connections  *int     `json:"connections"`
}

// RuleRuntimeStatus 心跳上报的规则实例状态。
type RuleRuntimeStatus struct {
	RuleID      uint   `json:"rule_id"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	TrafficIn   *int64 `json:"traffic_in"`
	TrafficOut  *int64 `json:"traffic_out"`
	Connections *int   `json:"connections"`
	InstanceID  string `json:"instance_id"`
	ReportedAt  string `json:"reported_at"`
}

// HeartbeatResult 心跳处理返回。
type HeartbeatResult struct {
	NodeID        uint        `json:"node_id"`
	ConfigUpdated bool        `json:"config_updated"`
	Config        *NodeConfig `json:"config,omitempty"`
}

// TrafficReportRecord 流量上报项。
type TrafficReportRecord struct {
	RuleID            *uint      `json:"rule_id"`
	NodeID            *uint      `json:"node_id"`
	TrafficIn         int64      `json:"traffic_in"`
	TrafficOut        int64      `json:"traffic_out"`
	VipMultiplier     *float64   `json:"vip_multiplier"`
	NodeMultiplier    *float64   `json:"node_multiplier"`
	FinalMultiplier   *float64   `json:"final_multiplier"`
	CalculatedTraffic *int64     `json:"calculated_traffic"`
	Hour              *time.Time `json:"hour"`
}

// ConfigDistributionService 配置下发服务。
type ConfigDistributionService struct {
	db *gorm.DB
}

// NewConfigDistributionService 创建配置下发服务。
func NewConfigDistributionService(db *gorm.DB) *ConfigDistributionService {
	return &ConfigDistributionService{db: db}
}

// GenerateNodeConfig 生成节点最新配置。
func (s *ConfigDistributionService) GenerateNodeConfig(nodeID uint) (*NodeConfig, error) {
	if nodeID == 0 {
		return nil, fmt.Errorf("%w: 节点 ID 无效", ErrInvalidParams)
	}

	node, err := s.getNodeByID(nodeID)
	if err != nil {
		return nil, err
	}

	rules := make([]models.Rule, 0)
	if err = s.db.Model(&models.Rule{}).
		Preload("ExitNode").
		Where("entry_node_id = ? AND status = ?", nodeID, "running").
		Order("id ASC").
		Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("查询节点运行规则失败: %w", err)
	}

	configRules := make([]RuleConfig, 0, len(rules))
	for _, rule := range rules {
		ruleConfig := RuleConfig{
			RuleID: int(rule.ID),
			Mode:   rule.Mode,
			Listen: HostPort{
				Host: rule.ListenHost,
				Port: rule.ListenPort,
			},
			Target: HostPort{
				Host: rule.TargetHost,
				Port: rule.TargetPort,
			},
			Protocol: rule.Protocol,
		}

		if strings.EqualFold(strings.TrimSpace(rule.Mode), "tunnel") {
			var exitNode *models.Node
			if rule.ExitNode != nil {
				exitNode = rule.ExitNode
			} else if rule.ExitNodeID != nil {
				exitNode, err = s.getNodeByID(*rule.ExitNodeID)
				if err != nil {
					return nil, fmt.Errorf("读取 tunnel 出口节点失败(rule_id=%d): %w", rule.ID, err)
				}
			}
			if exitNode == nil {
				return nil, fmt.Errorf("%w: 隧道规则缺少出口节点(rule_id=%d)", ErrInvalidParams, rule.ID)
			}
			ruleConfig.ExitNode = &HostPort{
				Host: exitNode.Host,
				Port: exitNode.Port,
			}
		}

		configRules = append(configRules, ruleConfig)
	}

	settings, err := s.loadSettings()
	if err != nil {
		return nil, err
	}

	return &NodeConfig{
		ConfigVersion: node.ConfigVersion,
		Rules:         configRules,
		Settings:      settings,
	}, nil
}

// HandleNodeRegister 处理节点注册并返回初始配置。
func (s *ConfigDistributionService) HandleNodeRegister(token string, hostname string, version string) (*RegisterResult, error) {
	node, err := s.AuthenticateNode(token)
	if err != nil {
		return nil, err
	}
	_ = version

	updates := map[string]interface{}{
		"status":            "online",
		"last_heartbeat_at": time.Now(),
	}
	trimmedHostname := strings.TrimSpace(hostname)
	if trimmedHostname != "" {
		updates["host"] = trimmedHostname
	}

	if err = s.db.Model(&models.Node{}).Where("id = ?", node.ID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新节点注册状态失败: %w", err)
	}

	config, err := s.GenerateNodeConfig(node.ID)
	if err != nil {
		return nil, err
	}

	return &RegisterResult{
		NodeID: node.ID,
		Config: config,
	}, nil
}

// HandleHeartbeat 处理节点心跳并在配置更新时返回新配置。
func (s *ConfigDistributionService) HandleHeartbeat(
	token string,
	currentConfigVersion int,
	systemInfo *HeartbeatSystemInfo,
	rulesStatus []RuleRuntimeStatus,
) (*HeartbeatResult, error) {
	node, err := s.AuthenticateNode(token)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":            "online",
		"last_heartbeat_at": now,
	}
	if systemInfo != nil {
		if systemInfo.CPUUsage != nil {
			updates["cpu_usage"] = *systemInfo.CPUUsage
		}
		if systemInfo.MemoryUsage != nil {
			updates["memory_usage"] = *systemInfo.MemoryUsage
		}
		if systemInfo.DiskUsage != nil {
			updates["disk_usage"] = *systemInfo.DiskUsage
		}
		if systemInfo.BandwidthIn != nil {
			updates["bandwidth_in"] = *systemInfo.BandwidthIn
		}
		if systemInfo.BandwidthOut != nil {
			updates["bandwidth_out"] = *systemInfo.BandwidthOut
		}
		if systemInfo.Connections != nil {
			updates["connections"] = *systemInfo.Connections
		}
	}

	if err = s.db.Model(&models.Node{}).Where("id = ?", node.ID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新节点心跳失败: %w", err)
	}

	for _, status := range rulesStatus {
		if status.RuleID == 0 {
			continue
		}

		payload := map[string]interface{}{
			"status":      strings.TrimSpace(status.Status),
			"message":     strings.TrimSpace(status.Message),
			"instance_id": strings.TrimSpace(status.InstanceID),
			"reported_at": now.Format(time.RFC3339),
		}
		if strings.TrimSpace(status.ReportedAt) != "" {
			payload["reported_at"] = strings.TrimSpace(status.ReportedAt)
		}
		if status.TrafficIn != nil {
			payload["traffic_in"] = *status.TrafficIn
		}
		if status.TrafficOut != nil {
			payload["traffic_out"] = *status.TrafficOut
		}
		if status.Connections != nil {
			payload["connections"] = *status.Connections
		}

		instanceStatusBytes, _ := json.Marshal(payload)
		instanceStatusText := string(instanceStatusBytes)

		ruleUpdates := map[string]interface{}{
			"instance_status": &instanceStatusText,
		}
		if status.TrafficIn != nil {
			ruleUpdates["traffic_in"] = *status.TrafficIn
		}
		if status.TrafficOut != nil {
			ruleUpdates["traffic_out"] = *status.TrafficOut
		}
		if status.Connections != nil {
			ruleUpdates["connections"] = *status.Connections
		}
		if strings.TrimSpace(status.InstanceID) != "" {
			instanceID := strings.TrimSpace(status.InstanceID)
			ruleUpdates["instance_id"] = &instanceID
		}

		if err = s.db.Model(&models.Rule{}).
			Where("id = ? AND entry_node_id = ?", status.RuleID, node.ID).
			Updates(ruleUpdates).Error; err != nil {
			return nil, fmt.Errorf("更新规则实例状态失败(rule_id=%d): %w", status.RuleID, err)
		}
	}

	freshNode, err := s.getNodeByID(node.ID)
	if err != nil {
		return nil, err
	}

	result := &HeartbeatResult{
		NodeID:        node.ID,
		ConfigUpdated: false,
	}
	if currentConfigVersion < freshNode.ConfigVersion {
		config, cfgErr := s.GenerateNodeConfig(node.ID)
		if cfgErr != nil {
			return nil, cfgErr
		}
		result.ConfigUpdated = true
		result.Config = config
	}

	return result, nil
}

// HandleConfigPull 主动拉取最新配置。
func (s *ConfigDistributionService) HandleConfigPull(nodeID uint) (*NodeConfig, error) {
	return s.GenerateNodeConfig(nodeID)
}

// HandleTrafficReport 处理节点流量上报。
func (s *ConfigDistributionService) HandleTrafficReport(token string, records []TrafficReportRecord) (int, error) {
	node, err := s.AuthenticateNode(token)
	if err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return 0, nil
	}

	accepted := 0
	for _, record := range records {
		reportNodeID := node.ID
		if record.NodeID != nil {
			reportNodeID = *record.NodeID
		}
		if reportNodeID != node.ID {
			return accepted, fmt.Errorf("%w: 上报节点与 token 不匹配", ErrForbidden)
		}

		hour := time.Now().UTC().Truncate(time.Hour)
		if record.Hour != nil {
			hour = record.Hour.UTC().Truncate(time.Hour)
		}

		vipMultiplier := 1.0
		if record.VipMultiplier != nil {
			vipMultiplier = *record.VipMultiplier
		}
		nodeMultiplier := 1.0
		if record.NodeMultiplier != nil {
			nodeMultiplier = *record.NodeMultiplier
		}
		finalMultiplier := 1.0
		if record.FinalMultiplier != nil {
			finalMultiplier = *record.FinalMultiplier
		}
		calculatedTraffic := record.TrafficIn + record.TrafficOut
		if record.CalculatedTraffic != nil {
			calculatedTraffic = *record.CalculatedTraffic
		}

		trafficRecord := &models.TrafficRecord{
			UserID:            node.UserID,
			RuleID:            record.RuleID,
			NodeID:            &reportNodeID,
			TrafficIn:         record.TrafficIn,
			TrafficOut:        record.TrafficOut,
			VipMultiplier:     vipMultiplier,
			NodeMultiplier:    nodeMultiplier,
			FinalMultiplier:   finalMultiplier,
			CalculatedTraffic: calculatedTraffic,
			Hour:              hour,
		}
		if err = s.db.Create(trafficRecord).Error; err != nil {
			return accepted, fmt.Errorf("写入流量记录失败: %w", err)
		}

		if record.RuleID != nil && *record.RuleID > 0 {
			if err = s.db.Model(&models.Rule{}).
				Where("id = ? AND entry_node_id = ?", *record.RuleID, node.ID).
				Updates(map[string]interface{}{
					"traffic_in":  gorm.Expr("traffic_in + ?", record.TrafficIn),
					"traffic_out": gorm.Expr("traffic_out + ?", record.TrafficOut),
				}).Error; err != nil {
				return accepted, fmt.Errorf("更新规则流量失败(rule_id=%d): %w", *record.RuleID, err)
			}
		}

		accepted++
	}

	return accepted, nil
}

// AuthenticateNode 通过 node token 认证节点身份。
func (s *ConfigDistributionService) AuthenticateNode(token string) (*models.Node, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("%w: 节点 token 不能为空", ErrUnauthorized)
	}

	tokenHash := hashNodeToken(token)
	var node models.Node
	err := s.db.Where("token_hash = ?", tokenHash).First(&node).Error
	if err == nil {
		return &node, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("校验节点 token 失败: %w", err)
	}

	// 兼容 bcrypt 历史 token_hash。
	candidates := make([]models.Node, 0)
	if err = s.db.Where("token_hash LIKE ?", "$2%").Find(&candidates).Error; err != nil {
		return nil, fmt.Errorf("校验节点 token 失败: %w", err)
	}
	for _, candidate := range candidates {
		if bcrypt.CompareHashAndPassword([]byte(candidate.TokenHash), []byte(token)) == nil {
			return &candidate, nil
		}
	}

	return nil, fmt.Errorf("%w: 节点 token 无效", ErrUnauthorized)
}

func (s *ConfigDistributionService) loadSettings() (Settings, error) {
	settings := Settings{
		HeartbeatInterval:   30,
		ConfigCheckInterval: 30,
	}

	systemConfigs := make([]models.SystemConfig, 0)
	if err := s.db.Model(&models.SystemConfig{}).
		Where("key IN ?", []string{"heartbeat_interval", "config_check_interval"}).
		Find(&systemConfigs).Error; err != nil {
		return settings, fmt.Errorf("读取系统配置失败: %w", err)
	}

	for _, item := range systemConfigs {
		if item.Value == nil {
			continue
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(*item.Value))
		if err != nil || parsed <= 0 {
			continue
		}
		switch strings.TrimSpace(item.Key) {
		case "heartbeat_interval":
			settings.HeartbeatInterval = parsed
		case "config_check_interval":
			settings.ConfigCheckInterval = parsed
		}
	}

	return settings, nil
}

func hashNodeToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *ConfigDistributionService) getNodeByID(nodeID uint) (*models.Node, error) {
	var node models.Node
	if err := s.db.First(&node, nodeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 节点不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询节点失败: %w", err)
	}
	return &node, nil
}
