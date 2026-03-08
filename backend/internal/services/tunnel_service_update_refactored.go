//go:build refactor_experiment
// +build refactor_experiment

package services

import (
	"fmt"
	"strings"

	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/utils"
)

// updateTunnelName 更新隧道名称
func updateTunnelName(current *models.Tunnel, newName *string) (string, error) {
	if newName == nil {
		return current.Name, nil
	}

	name := strings.TrimSpace(*newName)
	if name == "" {
		return "", fmt.Errorf("%w: name 不能为空", ErrInvalidParams)
	}
	if len(name) > 100 {
		return "", fmt.Errorf("%w: name 长度不能超过 100", ErrInvalidParams)
	}

	return name, nil
}

// updateTunnelProtocol 更新隧道协议
func updateTunnelProtocol(current *models.Tunnel, newProtocol *string) (string, error) {
	if newProtocol == nil {
		return current.Protocol, nil
	}

	protocol, err := tunnelNormalizeProtocol(*newProtocol)
	if err != nil {
		return "", err
	}

	return protocol, nil
}

// updateTunnelRemoteHost 更新远程主机
func updateTunnelRemoteHost(current *models.Tunnel, newHost *string) (string, error) {
	if newHost == nil {
		return current.RemoteHost, nil
	}

	host := strings.TrimSpace(*newHost)
	if host == "" {
		return "", fmt.Errorf("%w: remote_host 不能为空", ErrInvalidParams)
	}
	if err := utils.ValidateHost(host); err != nil {
		return "", fmt.Errorf("%w: remote_host 无效: %v", ErrInvalidParams, err)
	}

	return host, nil
}

// updateTunnelRemotePort 更新远程端口
func updateTunnelRemotePort(current *models.Tunnel, newPort *int) (int, error) {
	if newPort == nil {
		return current.RemotePort, nil
	}

	if err := utils.ValidatePort(*newPort); err != nil {
		return 0, fmt.Errorf("%w: remote_port 无效: %v", ErrInvalidParams, err)
	}

	return *newPort, nil
}

// updateTunnelListenPort 更新监听端口
func updateTunnelListenPort(current *models.Tunnel, newPort *int) (int, error) {
	if newPort == nil {
		return current.ListenPort, nil
	}

	listenPort := *newPort
	if listenPort > 0 {
		if err := utils.ValidatePort(listenPort); err != nil {
			return 0, fmt.Errorf("%w: listen_port 无效: %v", ErrInvalidParams, err)
		}
	}

	return listenPort, nil
}

// updateTunnelListenHost 更新监听地址
func updateTunnelListenHost(current *models.Tunnel, newHost *string) string {
	if newHost == nil {
		return current.ListenHost
	}

	host := strings.TrimSpace(*newHost)
	if host == "" {
		return "0.0.0.0"
	}

	return host
}

// updateTunnelEntryGroupID 更新入口节点组ID
func updateTunnelEntryGroupID(current *models.Tunnel, newID *uint) (uint, error) {
	if newID == nil {
		return current.EntryGroupID, nil
	}

	if *newID == 0 {
		return 0, fmt.Errorf("%w: entry_group_id 无效", ErrInvalidParams)
	}

	return *newID, nil
}

// updateTunnelExitGroupID 更新出口节点组ID
func updateTunnelExitGroupID(current *models.Tunnel, newID *uint) *uint {
	// 保持当前值
	nextExitGroupID := current.ExitGroupID

	// 如果提供了新值
	if newID != nil {
		if *newID == 0 {
			// 允许设置为 0 来清除出口节点组（切换到直连模式）
			nextExitGroupID = nil
		} else {
			nextExitGroupID = newID
		}
	}

	return nextExitGroupID
}

// validateUpdateTunnelGroups 验证更新的节点组
func (s *TunnelService) validateUpdateTunnelGroups(userID uint, entryGroupID uint, exitGroupID *uint, protocol string) (*models.NodeGroup, *models.NodeGroup, error) {
	// 验证入口节点组
	entryGroup, err := s.getTunnelGroup(userID, entryGroupID, models.NodeGroupTypeEntry, true)
	if err != nil {
		return nil, nil, err
	}

	// 验证出口节点组
	var exitGroup *models.NodeGroup
	if exitGroupID != nil && *exitGroupID > 0 {
		exitGroup, err = s.getTunnelGroup(userID, *exitGroupID, models.NodeGroupTypeExit, true)
		if err != nil {
			return nil, nil, err
		}

		if err = s.validateTunnelRelation(entryGroup.ID, exitGroup.ID); err != nil {
			return nil, nil, err
		}

		if err = s.validateTunnelProtocol(entryGroup, exitGroup, protocol); err != nil {
			return nil, nil, err
		}
	} else {
		// 直连模式，检查入口节点组是否支持
		entryConfig, err := entryGroup.GetConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("获取入口节点组配置失败: %w", err)
		}

		if entryConfig.EntryConfig != nil && entryConfig.EntryConfig.RequireExitGroup {
			return nil, nil, fmt.Errorf("%w: 该入口节点组要求配置出口节点组", ErrInvalidParams)
		}
	}

	return entryGroup, exitGroup, nil
}

// updateTunnelConfig 更新隧道配置
func updateTunnelConfig(current *models.Tunnel, newConfig *models.TunnelConfig, protocol string) (*models.TunnelConfig, error) {
	if newConfig == nil {
		// 保持当前配置
		return current.GetConfig()
	}

	// 验证新配置
	if err := validateTunnelConfig(protocol, newConfig); err != nil {
		return nil, err
	}

	return newConfig, nil
}

// applyTunnelUpdates 应用隧道更新
func applyTunnelUpdates(tunnel *models.Tunnel, name, protocol, remoteHost, listenHost string,
	remotePort, listenPort int, entryGroupID uint, exitGroupID *uint, config *models.TunnelConfig) error {

	tunnel.Name = name
	tunnel.Protocol = protocol
	tunnel.RemoteHost = remoteHost
	tunnel.RemotePort = remotePort
	tunnel.ListenHost = listenHost
	tunnel.ListenPort = listenPort
	tunnel.EntryGroupID = entryGroupID
	tunnel.ExitGroupID = exitGroupID

	if config != nil {
		if err := tunnel.SetConfig(config); err != nil {
			return fmt.Errorf("设置隧道配置失败: %w", err)
		}
	}

	return nil
}

// Update 更新隧道（重构后）
func (s *TunnelService) Update(userID uint, id uint, req *UpdateTunnelRequest) (*models.Tunnel, error) {
	// 1. 验证请求
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	// 2. 获取当前隧道
	current, err := s.Get(userID, id)
	if err != nil {
		return nil, err
	}

	// 3. 更新各个字段
	nextName, err := updateTunnelName(current, req.Name)
	if err != nil {
		return nil, err
	}

	nextProtocol, err := updateTunnelProtocol(current, req.Protocol)
	if err != nil {
		return nil, err
	}

	nextRemoteHost, err := updateTunnelRemoteHost(current, req.RemoteHost)
	if err != nil {
		return nil, err
	}

	nextRemotePort, err := updateTunnelRemotePort(current, req.RemotePort)
	if err != nil {
		return nil, err
	}

	nextListenPort, err := updateTunnelListenPort(current, req.ListenPort)
	if err != nil {
		return nil, err
	}

	nextListenHost := updateTunnelListenHost(current, req.ListenHost)

	nextEntryGroupID, err := updateTunnelEntryGroupID(current, req.EntryGroupID)
	if err != nil {
		return nil, err
	}

	nextExitGroupID := updateTunnelExitGroupID(current, req.ExitGroupID)

	// 4. 验证节点组
	_, _, err = s.validateUpdateTunnelGroups(userID, nextEntryGroupID, nextExitGroupID, nextProtocol)
	if err != nil {
		return nil, err
	}

	// 5. 检查端口冲突
	if nextListenPort > 0 {
		if err = s.checkTunnelPortConflict(current.ID, nextEntryGroupID, nextListenPort); err != nil {
			return nil, err
		}
	}

	// 6. 更新配置
	nextConfig, err := updateTunnelConfig(current, req.Config, nextProtocol)
	if err != nil {
		return nil, err
	}

	// 7. 应用所有更新
	if err = applyTunnelUpdates(current, nextName, nextProtocol, nextRemoteHost, nextListenHost,
		nextRemotePort, nextListenPort, nextEntryGroupID, nextExitGroupID, nextConfig); err != nil {
		return nil, err
	}

	// 8. 保存到数据库
	if err = s.db.Save(current).Error; err != nil {
		return nil, fmt.Errorf("更新隧道失败: %w", err)
	}

	// 9. 返回更新后的隧道
	return s.Get(userID, current.ID)
}
