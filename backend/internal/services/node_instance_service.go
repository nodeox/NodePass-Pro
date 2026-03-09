package services

import (
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/license"
	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/utils"

	"gorm.io/gorm"
)

// NodeInstanceService 节点实例服务。
type NodeInstanceService struct {
	db           *gorm.DB
	nodeGroupSvc *NodeGroupService
	licenseMgr   *license.Manager
}

// NewNodeInstanceService 创建节点实例服务。
func NewNodeInstanceService(db *gorm.DB, licenseMgr *license.Manager) *NodeInstanceService {
	return &NodeInstanceService{
		db:           db,
		nodeGroupSvc: NewNodeGroupService(db),
		licenseMgr:   licenseMgr,
	}
}

// UpdateNodeInstanceRequest 更新节点实例请求。
type UpdateNodeInstanceRequest struct {
	Name      *string `json:"name"`
	Host      *string `json:"host"`
	Port      *int    `json:"port"`
	Status    *string `json:"status"`
	IsEnabled *bool   `json:"is_enabled"`
}

type HeartbeatSystemInfoPayload struct {
	CPUUsage     *float64 `json:"cpu_usage"`
	MemoryUsage  *float64 `json:"memory_usage"`
	DiskUsage    *float64 `json:"disk_usage"`
	BandwidthIn  *int64   `json:"bandwidth_in"`
	BandwidthOut *int64   `json:"bandwidth_out"`
	Connections  *int     `json:"connections"`
}

type HeartbeatTrafficStatsPayload struct {
	TrafficIn         *int64 `json:"traffic_in"`
	TrafficOut        *int64 `json:"traffic_out"`
	ActiveConnections *int   `json:"active_connections"`
}

// HeartbeatRequest 节点实例心跳请求。
type HeartbeatRequest struct {
	NodeID        string `json:"node_id" binding:"required"`
	Token         string `json:"token" binding:"required"`
	ClientVersion string `json:"client_version" binding:"required"`
	// CurrentConfigVersion 表示节点当前应用的配置版本（用于差量下发）。
	CurrentConfigVersion int                           `json:"current_config_version"`
	ConnectionAddress    *string                       `json:"connection_address"`
	Status               *string                       `json:"status"`
	CPUUsage             *float64                      `json:"cpu_usage"`
	MemoryUsage          *float64                      `json:"memory_usage"`
	DiskUsage            *float64                      `json:"disk_usage"`
	BandwidthIn          *int64                        `json:"bandwidth_in"`
	BandwidthOut         *int64                        `json:"bandwidth_out"`
	Connections          *int                          `json:"connections"`
	SystemInfo           *HeartbeatSystemInfoPayload   `json:"system_info"`
	TrafficStats         *HeartbeatTrafficStatsPayload `json:"traffic_stats"`
}

// HeartbeatResponse 节点实例心跳响应。
type HeartbeatResponse struct {
	InstanceID       uint                      `json:"instance_id"`
	NodeID           string                    `json:"node_id"`
	Status           models.NodeInstanceStatus `json:"status"`
	ConfigVersion    int                       `json:"config_version"`
	ConfigUpdated    bool                      `json:"config_updated"`
	NewConfigVersion int                       `json:"new_config_version"`
	Config           *NodeConfig               `json:"config,omitempty"`
	LastHeartbeatAt  *time.Time                `json:"last_heartbeat_at"`
}

// Get 获取节点实例详情。
func (s *NodeInstanceService) Get(userID uint, id uint) (*models.NodeInstance, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("node instance service 未初始化")
	}
	if id == 0 {
		return nil, fmt.Errorf("%w: 节点实例 ID 无效", ErrInvalidParams)
	}

	query := s.db.Model(&models.NodeInstance{}).Preload("NodeGroup")
	if userID > 0 {
		query = query.Joins("JOIN node_groups ON node_groups.id = node_instances.node_group_id").
			Where("node_groups.user_id = ?", userID)
	}

	var instance models.NodeInstance
	if err := query.First(&instance, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("%w: 节点实例不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询节点实例失败: %w", err)
	}
	return &instance, nil
}

// Update 更新节点实例。
func (s *NodeInstanceService) Update(userID uint, id uint, req *UpdateNodeInstanceRequest) (*models.NodeInstance, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	instance, err := s.Get(userID, id)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	shouldBumpConfigVersion := false
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name 不能为空", ErrInvalidParams)
		}
		if len(name) > 100 {
			return nil, fmt.Errorf("%w: name 长度不能超过 100", ErrInvalidParams)
		}
		updates["name"] = name
	}

	if req.Host != nil {
		shouldBumpConfigVersion = true
		host := strings.TrimSpace(*req.Host)
		if host == "" {
			updates["host"] = nil
		} else {
			if err = utils.ValidateHost(host); err != nil {
				return nil, fmt.Errorf("%w: host 无效: %v", ErrInvalidParams, err)
			}
			updates["host"] = host
		}
	}

	if req.Port != nil {
		shouldBumpConfigVersion = true
		if *req.Port <= 0 {
			updates["port"] = nil
		} else {
			if err = utils.ValidatePort(*req.Port); err != nil {
				return nil, fmt.Errorf("%w: port 无效: %v", ErrInvalidParams, err)
			}
			updates["port"] = *req.Port
		}
	}

	if req.Status != nil {
		shouldBumpConfigVersion = true
		status := models.NodeInstanceStatus(strings.ToLower(strings.TrimSpace(*req.Status)))
		switch status {
		case models.NodeInstanceStatusOnline, models.NodeInstanceStatusOffline, models.NodeInstanceStatusMaintain:
			updates["status"] = status
		default:
			return nil, fmt.Errorf("%w: status 仅支持 online/offline/maintain", ErrInvalidParams)
		}
	}

	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
		shouldBumpConfigVersion = true
		if !*req.IsEnabled {
			updates["status"] = models.NodeInstanceStatusOffline
			updates["last_heartbeat_at"] = nil
		}
	}

	if len(updates) == 0 {
		return instance, nil
	}

	if err = s.db.Model(&models.NodeInstance{}).Where("id = ?", instance.ID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新节点实例失败: %w", err)
	}

	if err = s.nodeGroupSvc.recalculateGroupStats(instance.NodeGroupID); err != nil {
		return nil, err
	}
	if shouldBumpConfigVersion {
		if err = s.nodeGroupSvc.BumpConfigVersionForGroupAndDependents([]uint{instance.NodeGroupID}); err != nil {
			return nil, err
		}
	}

	return s.Get(userID, instance.ID)
}

// Delete 删除节点实例。
func (s *NodeInstanceService) Delete(userID uint, id uint) error {
	return s.nodeGroupSvc.DeleteNodeInstance(userID, id)
}

// Restart 重启节点实例。
func (s *NodeInstanceService) Restart(userID uint, id uint) (*models.NodeInstance, error) {
	return s.nodeGroupSvc.RestartNodeInstance(userID, id)
}

// Heartbeat 处理节点实例心跳。
func (s *NodeInstanceService) Heartbeat(req *HeartbeatRequest) (*HeartbeatResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	normalized := normalizeNodeInstanceHeartbeatRequest(req)
	clientVersion := strings.TrimSpace(normalized.ClientVersion)
	if clientVersion == "" {
		return nil, fmt.Errorf("%w: client_version 不能为空", ErrInvalidParams)
	}
	if s.licenseMgr != nil {
		if err := s.licenseMgr.ValidateNodeclientVersion(clientVersion); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrForbidden, err)
		}
	}

	instance, err := s.nodeGroupSvc.HandleNodeInstanceHeartbeat(&NodeInstanceHeartbeatRequest{
		NodeID:            normalized.NodeID,
		Token:             normalized.Token,
		ConnectionAddress: normalized.ConnectionAddress,
		Status:            normalized.Status,
		CPUUsage:          normalized.CPUUsage,
		MemoryUsage:       normalized.MemoryUsage,
		DiskUsage:         normalized.DiskUsage,
		BandwidthIn:       normalized.BandwidthIn,
		BandwidthOut:      normalized.BandwidthOut,
		Connections:       normalized.Connections,
	})
	if err != nil {
		return nil, err
	}

	nodeCfg, err := s.nodeGroupSvc.GenerateNodeConfigForInstance(instance.ID)
	if err != nil {
		return nil, err
	}

	configUpdated := normalized.CurrentConfigVersion != instance.ConfigVersion
	// current_config_version < 0 视为首次启动，强制下发一次完整配置。
	if normalized.CurrentConfigVersion < 0 {
		configUpdated = true
	}

	return &HeartbeatResponse{
		InstanceID:       instance.ID,
		NodeID:           instance.NodeID,
		Status:           instance.Status,
		ConfigVersion:    instance.ConfigVersion,
		ConfigUpdated:    configUpdated,
		NewConfigVersion: instance.ConfigVersion,
		Config: func() *NodeConfig {
			if configUpdated {
				return nodeCfg
			}
			return nil
		}(),
		LastHeartbeatAt: instance.LastHeartbeatAt,
	}, nil
}

func normalizeNodeInstanceHeartbeatRequest(req *HeartbeatRequest) *HeartbeatRequest {
	if req == nil {
		return nil
	}

	result := *req

	if req.SystemInfo != nil {
		if result.CPUUsage == nil {
			result.CPUUsage = req.SystemInfo.CPUUsage
		}
		if result.MemoryUsage == nil {
			result.MemoryUsage = req.SystemInfo.MemoryUsage
		}
		if result.DiskUsage == nil {
			result.DiskUsage = req.SystemInfo.DiskUsage
		}
		if result.BandwidthIn == nil {
			result.BandwidthIn = req.SystemInfo.BandwidthIn
		}
		if result.BandwidthOut == nil {
			result.BandwidthOut = req.SystemInfo.BandwidthOut
		}
		if result.Connections == nil {
			result.Connections = req.SystemInfo.Connections
		}
	}

	if req.TrafficStats != nil {
		if result.BandwidthIn == nil {
			result.BandwidthIn = req.TrafficStats.TrafficIn
		}
		if result.BandwidthOut == nil {
			result.BandwidthOut = req.TrafficStats.TrafficOut
		}
		if result.Connections == nil {
			result.Connections = req.TrafficStats.ActiveConnections
		}
	}

	return &result
}
