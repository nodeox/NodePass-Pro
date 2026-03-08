//go:build refactor_experiment
// +build refactor_experiment

package services

import (
	"fmt"
	"strings"

	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/utils"
)

// validateCreateTunnelRequest 验证创建隧道请求
func (s *TunnelService) validateCreateTunnelRequest(req *CreateTunnelRequest) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("tunnel service 未初始化")
	}
	if req == nil {
		return fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}
	return nil
}

// validateTunnelName 验证隧道名称
func validateTunnelName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: name 不能为空", ErrInvalidParams)
	}
	if len(name) > 100 {
		return "", fmt.Errorf("%w: name 长度不能超过 100", ErrInvalidParams)
	}
	return name, nil
}

// validateTunnelRemoteHost 验证远程主机
func validateTunnelRemoteHost(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("%w: remote_host 不能为空", ErrInvalidParams)
	}
	if err := utils.ValidateHost(host); err != nil {
		return "", fmt.Errorf("%w: remote_host 无效: %v", ErrInvalidParams, err)
	}
	return host, nil
}

// validateTunnelRemotePort 验证远程端口
func validateTunnelRemotePort(port int) error {
	if err := utils.ValidatePort(port); err != nil {
		return fmt.Errorf("%w: remote_port 无效: %v", ErrInvalidParams, err)
	}
	return nil
}

// validateTunnelListenPort 验证监听端口
func validateTunnelListenPort(port *int) (int, error) {
	if port == nil {
		return 0, nil
	}

	listenPort := *port
	if listenPort > 0 {
		if err := utils.ValidatePort(listenPort); err != nil {
			return 0, fmt.Errorf("%w: listen_port 无效: %v", ErrInvalidParams, err)
		}
	}
	return listenPort, nil
}

// validateTunnelEntryGroup 验证入口节点组
func (s *TunnelService) validateTunnelEntryGroup(userID, entryGroupID uint) (*models.NodeGroup, error) {
	if entryGroupID == 0 {
		return nil, fmt.Errorf("%w: entry_group_id 不能为空", ErrInvalidParams)
	}

	entryGroup, err := s.getTunnelGroup(userID, entryGroupID, models.NodeGroupTypeEntry, true)
	if err != nil {
		return nil, err
	}

	return entryGroup, nil
}

// validateTunnelExitGroup 验证出口节点组
func (s *TunnelService) validateTunnelExitGroup(userID uint, exitGroupID *uint, entryGroup *models.NodeGroup, protocol string) (*models.NodeGroup, error) {
	// 出口节点组可选
	if exitGroupID == nil || *exitGroupID == 0 {
		return s.validateNoExitGroupMode(entryGroup)
	}

	exitGroup, err := s.getTunnelGroup(userID, *exitGroupID, models.NodeGroupTypeExit, true)
	if err != nil {
		return nil, err
	}

	if err = s.validateTunnelRelation(entryGroup.ID, exitGroup.ID); err != nil {
		return nil, err
	}

	if err = s.validateTunnelProtocol(entryGroup, exitGroup, protocol); err != nil {
		return nil, err
	}

	return exitGroup, nil
}

// validateNoExitGroupMode 验证不带出口节点组模式
func (s *TunnelService) validateNoExitGroupMode(entryGroup *models.NodeGroup) (*models.NodeGroup, error) {
	entryConfig, err := entryGroup.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("获取入口节点组配置失败: %w", err)
	}

	if entryConfig.EntryConfig != nil && entryConfig.EntryConfig.RequireExitGroup {
		return nil, fmt.Errorf("%w: 该入口节点组要求配置出口节点组", ErrInvalidParams)
	}

	return nil, nil
}

// prepareTunnelListenHost 准备监听地址
func prepareTunnelListenHost(listenHost *string) string {
	if listenHost == nil {
		return "0.0.0.0"
	}

	host := strings.TrimSpace(*listenHost)
	if host == "" {
		return "0.0.0.0"
	}

	return host
}

// prepareTunnelConfig 准备隧道配置
func prepareTunnelConfig(config *models.TunnelConfig, protocol string) (*models.TunnelConfig, error) {
	// 使用提供的配置或创建默认配置
	tunnelConfig := config
	if tunnelConfig == nil {
		tunnelConfig = &models.TunnelConfig{
			LoadBalanceStrategy: models.LoadBalanceRoundRobin,
			IPType:              "auto",
			ForwardTargets:      []models.ForwardTarget{},
		}
	}

	// 验证配置
	if err := validateTunnelConfig(protocol, tunnelConfig); err != nil {
		return nil, err
	}

	return tunnelConfig, nil
}

// buildTunnel 构建隧道对象
func buildTunnel(req *CreateTunnelRequest, entryGroup *models.NodeGroup, exitGroupID *uint,
	name, remoteHost, listenHost, protocol string, listenPort, remotePort int) *models.Tunnel {

	return &models.Tunnel{
		UserID:       entryGroup.UserID,
		Name:         name,
		Description:  req.Description,
		EntryGroupID: entryGroup.ID,
		ExitGroupID:  exitGroupID,
		Protocol:     protocol,
		ListenHost:   listenHost,
		ListenPort:   listenPort,
		RemoteHost:   remoteHost,
		RemotePort:   remotePort,
		Status:       models.TunnelStatusStopped,
	}
}

// Create 创建隧道（重构后）
func (s *TunnelService) Create(userID uint, req *CreateTunnelRequest) (*models.Tunnel, error) {
	// 1. 验证服务和请求
	if err := s.validateCreateTunnelRequest(req); err != nil {
		return nil, err
	}

	// 2. 验证和准备基本参数
	name, err := validateTunnelName(req.Name)
	if err != nil {
		return nil, err
	}

	remoteHost, err := validateTunnelRemoteHost(req.RemoteHost)
	if err != nil {
		return nil, err
	}

	if err := validateTunnelRemotePort(req.RemotePort); err != nil {
		return nil, err
	}

	listenPort, err := validateTunnelListenPort(req.ListenPort)
	if err != nil {
		return nil, err
	}

	// 3. 验证协议
	protocol, err := tunnelNormalizeProtocol(req.Protocol)
	if err != nil {
		return nil, err
	}

	// 4. 验证入口节点组
	entryGroup, err := s.validateTunnelEntryGroup(userID, req.EntryGroupID)
	if err != nil {
		return nil, err
	}

	// 5. 验证出口节点组
	var exitGroupID *uint
	if req.ExitGroupID != nil && *req.ExitGroupID > 0 {
		exitGroupID = req.ExitGroupID
	}

	_, err = s.validateTunnelExitGroup(userID, exitGroupID, entryGroup, protocol)
	if err != nil {
		return nil, err
	}

	// 6. 检查端口冲突
	if listenPort > 0 {
		if err = s.checkTunnelPortConflict(0, entryGroup.ID, listenPort); err != nil {
			return nil, err
		}
	}

	// 7. 准备监听地址和配置
	listenHost := prepareTunnelListenHost(req.ListenHost)
	tunnelConfig, err := prepareTunnelConfig(req.Config, protocol)
	if err != nil {
		return nil, err
	}

	// 8. 构建隧道对象
	tunnel := buildTunnel(req, entryGroup, exitGroupID, name, remoteHost,
		listenHost, protocol, listenPort, req.RemotePort)

	if err = tunnel.SetConfig(tunnelConfig); err != nil {
		return nil, fmt.Errorf("设置隧道配置失败: %w", err)
	}

	// 9. 保存到数据库
	if err = s.db.Create(tunnel).Error; err != nil {
		return nil, fmt.Errorf("创建隧道失败: %w", err)
	}

	// 10. 返回完整的隧道信息
	return s.Get(userID, tunnel.ID)
}
