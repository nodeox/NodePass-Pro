package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/utils"

	"gorm.io/gorm"
)

// TunnelService 节点组隧道服务。
type TunnelService struct {
	db *gorm.DB
}

// NewTunnelService 创建隧道服务。
func NewTunnelService(db *gorm.DB) *TunnelService {
	return &TunnelService{db: db}
}

// CreateTunnelRequest 创建隧道请求。
type CreateTunnelRequest struct {
	Name         string               `json:"name" binding:"required"`
	Description  *string              `json:"description"`
	EntryGroupID uint                 `json:"entry_group_id" binding:"required"`
	ExitGroupID  *uint                `json:"exit_group_id"`
	Protocol     string               `json:"protocol" binding:"required"`
	ListenHost   *string              `json:"listen_host"`
	ListenPort   *int                 `json:"listen_port"`
	RemoteHost   string               `json:"remote_host"`
	RemotePort   int                  `json:"remote_port"`
	Config       *models.TunnelConfig `json:"config"`
}

// UpdateTunnelRequest 更新隧道请求。
type UpdateTunnelRequest struct {
	Name         *string              `json:"name"`
	Description  *string              `json:"description"`
	Protocol     *string              `json:"protocol"`
	ListenHost   *string              `json:"listen_host"`
	ListenPort   *int                 `json:"listen_port"`
	RemoteHost   *string              `json:"remote_host"`
	RemotePort   *int                 `json:"remote_port"`
	EntryGroupID *uint                `json:"entry_group_id"`
	ExitGroupID  *uint                `json:"exit_group_id"`
	Config       *models.TunnelConfig `json:"config"`
}

// ListTunnelParams 隧道列表参数。
type ListTunnelParams struct {
	Status   *string `json:"status"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

// Create 创建隧道。
func (s *TunnelService) Create(userID uint, req *CreateTunnelRequest) (*models.Tunnel, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("tunnel service 未初始化")
	}
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name 不能为空", ErrInvalidParams)
	}
	if len(name) > 100 {
		return nil, fmt.Errorf("%w: name 长度不能超过 100", ErrInvalidParams)
	}

	remoteHost := strings.TrimSpace(req.RemoteHost)
	if remoteHost == "" {
		return nil, fmt.Errorf("%w: remote_host 不能为空", ErrInvalidParams)
	}
	if err := utils.ValidateHost(remoteHost); err != nil {
		return nil, fmt.Errorf("%w: remote_host 无效: %v", ErrInvalidParams, err)
	}

	remotePort := req.RemotePort
	if err := utils.ValidatePort(remotePort); err != nil {
		return nil, fmt.Errorf("%w: remote_port 无效: %v", ErrInvalidParams, err)
	}

	var listenPort int
	if req.ListenPort != nil {
		listenPort = *req.ListenPort
		if listenPort > 0 {
			if err := utils.ValidatePort(listenPort); err != nil {
				return nil, fmt.Errorf("%w: listen_port 无效: %v", ErrInvalidParams, err)
			}
		}
	}

	if req.EntryGroupID == 0 {
		return nil, fmt.Errorf("%w: entry_group_id 不能为空", ErrInvalidParams)
	}

	// 出口节点组可选（支持不带出口节点组模式）
	var exitGroupID *uint
	if req.ExitGroupID != nil && *req.ExitGroupID > 0 {
		exitGroupID = req.ExitGroupID
	}

	protocol, err := tunnelNormalizeProtocol(req.Protocol)
	if err != nil {
		return nil, err
	}

	entryGroup, err := s.getTunnelGroup(userID, req.EntryGroupID, models.NodeGroupTypeEntry, true)
	if err != nil {
		return nil, err
	}

	var exitGroup *models.NodeGroup
	if exitGroupID != nil {
		exitGroup, err = s.getTunnelGroup(userID, *exitGroupID, models.NodeGroupTypeExit, true)
		if err != nil {
			return nil, err
		}

		if err = s.validateTunnelRelation(entryGroup.ID, exitGroup.ID); err != nil {
			return nil, err
		}
		if err = s.validateTunnelProtocol(entryGroup, exitGroup, protocol); err != nil {
			return nil, err
		}
	} else {
		// 不带出口节点组模式，检查入口节点组配置
		entryConfig, err := entryGroup.GetConfig()
		if err != nil {
			return nil, fmt.Errorf("获取入口节点组配置失败: %w", err)
		}
		if entryConfig.EntryConfig != nil && entryConfig.EntryConfig.RequireExitGroup {
			return nil, fmt.Errorf("%w: 该入口节点组要求配置出口节点组", ErrInvalidParams)
		}
	}
	if listenPort > 0 {
		if err = s.checkTunnelPortConflict(0, entryGroup.ID, listenPort); err != nil {
			return nil, err
		}
	}

	// 处理监听地址
	listenHost := "0.0.0.0"
	if req.ListenHost != nil {
		listenHost = strings.TrimSpace(*req.ListenHost)
		if listenHost == "" {
			listenHost = "0.0.0.0"
		}
	}

	// 处理配置
	var tunnelConfig *models.TunnelConfig
	if req.Config != nil {
		tunnelConfig = req.Config
	}

	// 设置默认配置
	if tunnelConfig == nil {
		tunnelConfig = &models.TunnelConfig{
			LoadBalanceStrategy: models.LoadBalanceRoundRobin,
			IPType:              "auto",
			ForwardTargets:      []models.ForwardTarget{},
		}
	}

	// 验证配置
	if err := validateTunnelConfig(tunnelConfig); err != nil {
		return nil, err
	}

	tunnel := &models.Tunnel{
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

	if err = tunnel.SetConfig(tunnelConfig); err != nil {
		return nil, fmt.Errorf("设置隧道配置失败: %w", err)
	}

	if err = s.db.Create(tunnel).Error; err != nil {
		return nil, fmt.Errorf("创建隧道失败: %w", err)
	}

	return s.Get(userID, tunnel.ID)
}

// List 列出隧道。
func (s *TunnelService) List(userID uint, params *ListTunnelParams) ([]models.Tunnel, int64, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("tunnel service 未初始化")
	}

	page := 1
	pageSize := 20
	if params != nil {
		if params.Page > 0 {
			page = params.Page
		}
		if params.PageSize > 0 {
			pageSize = params.PageSize
		}
	}
	if pageSize > 200 {
		pageSize = 200
	}

	query := s.db.Model(&models.Tunnel{}).
		Preload("EntryGroup").
		Preload("ExitGroup")
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if params != nil && params.Status != nil {
		status := strings.TrimSpace(*params.Status)
		if status != "" {
			query = query.Where("status = ?", status)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询隧道总数失败: %w", err)
	}

	list := make([]models.Tunnel, 0, pageSize)
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询隧道列表失败: %w", err)
	}

	return list, total, nil
}

// Get 获取隧道详情。
func (s *TunnelService) Get(userID uint, id uint) (*models.Tunnel, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("tunnel service 未初始化")
	}
	if id == 0 {
		return nil, fmt.Errorf("%w: 隧道 ID 无效", ErrInvalidParams)
	}

	query := s.db.Model(&models.Tunnel{}).
		Preload("EntryGroup").
		Preload("ExitGroup")
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	var tunnel models.Tunnel
	if err := query.First(&tunnel, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 隧道不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询隧道失败: %w", err)
	}

	return &tunnel, nil
}

// Update 更新隧道。
func (s *TunnelService) Update(userID uint, id uint, req *UpdateTunnelRequest) (*models.Tunnel, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	current, err := s.Get(userID, id)
	if err != nil {
		return nil, err
	}

	nextName := current.Name
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name 不能为空", ErrInvalidParams)
		}
		if len(name) > 100 {
			return nil, fmt.Errorf("%w: name 长度不能超过 100", ErrInvalidParams)
		}
		nextName = name
	}

	nextProtocol := current.Protocol
	if req.Protocol != nil {
		nextProtocol, err = tunnelNormalizeProtocol(*req.Protocol)
		if err != nil {
			return nil, err
		}
	}

	nextRemoteHost := current.RemoteHost
	if req.RemoteHost != nil {
		host := strings.TrimSpace(*req.RemoteHost)
		if host == "" {
			return nil, fmt.Errorf("%w: remote_host 不能为空", ErrInvalidParams)
		}
		if err = utils.ValidateHost(host); err != nil {
			return nil, fmt.Errorf("%w: remote_host 无效: %v", ErrInvalidParams, err)
		}
		nextRemoteHost = host
	}

	nextRemotePort := current.RemotePort
	if req.RemotePort != nil {
		if err = utils.ValidatePort(*req.RemotePort); err != nil {
			return nil, fmt.Errorf("%w: remote_port 无效: %v", ErrInvalidParams, err)
		}
		nextRemotePort = *req.RemotePort
	}

	nextListenPort := current.ListenPort
	if req.ListenPort != nil {
		nextListenPort = *req.ListenPort
		if nextListenPort > 0 {
			if err = utils.ValidatePort(nextListenPort); err != nil {
				return nil, fmt.Errorf("%w: listen_port 无效: %v", ErrInvalidParams, err)
			}
		}
	}
	nextListenHost := current.ListenHost
	if req.ListenHost != nil {
		listenHost := strings.TrimSpace(*req.ListenHost)
		if listenHost == "" {
			listenHost = "0.0.0.0"
		}
		nextListenHost = listenHost
	}

	nextEntryGroupID := current.EntryGroupID
	if req.EntryGroupID != nil {
		if *req.EntryGroupID == 0 {
			return nil, fmt.Errorf("%w: entry_group_id 无效", ErrInvalidParams)
		}
		nextEntryGroupID = *req.EntryGroupID
	}

	// 处理出口节点组（支持直连模式，可为空）
	var nextExitGroupID *uint
	if current.ExitGroupID != nil {
		nextExitGroupID = current.ExitGroupID
	}
	if req.ExitGroupID != nil {
		if *req.ExitGroupID == 0 {
			// 允许设置为 0 来清除出口节点组（切换到直连模式）
			nextExitGroupID = nil
		} else {
			nextExitGroupID = req.ExitGroupID
		}
	}

	entryGroup, err := s.getTunnelGroup(userID, nextEntryGroupID, models.NodeGroupTypeEntry, true)
	if err != nil {
		return nil, err
	}

	var exitGroup *models.NodeGroup
	if nextExitGroupID != nil && *nextExitGroupID > 0 {
		exitGroup, err = s.getTunnelGroup(userID, *nextExitGroupID, models.NodeGroupTypeExit, true)
		if err != nil {
			return nil, err
		}

		if err = s.validateTunnelRelation(entryGroup.ID, exitGroup.ID); err != nil {
			return nil, err
		}
		if err = s.validateTunnelProtocol(entryGroup, exitGroup, nextProtocol); err != nil {
			return nil, err
		}
	} else {
		// 直连模式，检查入口节点组是否支持
		entryConfig, err := entryGroup.GetConfig()
		if err != nil {
			return nil, fmt.Errorf("获取入口节点组配置失败: %w", err)
		}
		if entryConfig.EntryConfig != nil && entryConfig.EntryConfig.RequireExitGroup {
			return nil, fmt.Errorf("%w: 该入口节点组要求配置出口节点组", ErrInvalidParams)
		}
	}
	if nextListenPort > 0 {
		if err = s.checkTunnelPortConflict(current.ID, nextEntryGroupID, nextListenPort); err != nil {
			return nil, err
		}
	}

	updates := map[string]interface{}{
		"name":           nextName,
		"entry_group_id": nextEntryGroupID,
		"protocol":       nextProtocol,
		"listen_host":    nextListenHost,
		"remote_host":    nextRemoteHost,
		"remote_port":    nextRemotePort,
		"listen_port":    nextListenPort,
	}

	// 处理 exit_group_id（可能为 nil）
	if nextExitGroupID != nil {
		updates["exit_group_id"] = *nextExitGroupID
	} else {
		updates["exit_group_id"] = nil
	}

	if req.Config != nil {
		if err = validateTunnelConfig(req.Config); err != nil {
			return nil, err
		}
		raw, marshalErr := json.Marshal(req.Config)
		if marshalErr != nil {
			return nil, fmt.Errorf("序列化隧道配置失败: %w", marshalErr)
		}
		updates["config_json"] = string(raw)
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("开启事务失败: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err = tx.Model(&models.Tunnel{}).Where("id = ?", current.ID).Updates(updates).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新隧道失败: %w", err)
	}

	if current.Status == models.TunnelStatusRunning {
		groupIDs := make([]uint, 0, 4)
		groupIDs = append(groupIDs, current.EntryGroupID, nextEntryGroupID)
		if current.ExitGroupID != nil && *current.ExitGroupID > 0 {
			groupIDs = append(groupIDs, *current.ExitGroupID)
		}
		if nextExitGroupID != nil && *nextExitGroupID > 0 {
			groupIDs = append(groupIDs, *nextExitGroupID)
		}

		notifyIDs, collectErr := s.collectAllInstanceIDsByGroups(tx, groupIDs)
		if collectErr != nil {
			tx.Rollback()
			return nil, collectErr
		}
		if len(notifyIDs) > 0 {
			if err = tx.Model(&models.NodeInstance{}).Where("id IN ?", notifyIDs).
				Update("config_version", gorm.Expr("config_version + 1")).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("通知节点隧道配置更新失败: %w", err)
			}
		}
	}

	if err = tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return s.Get(userID, current.ID)
}

// Delete 删除隧道。
func (s *TunnelService) Delete(userID uint, id uint) error {
	tunnel, err := s.Get(userID, id)
	if err != nil {
		return err
	}

	if tunnel.Status == models.TunnelStatusRunning {
		if err = s.Stop(userID, tunnel.ID); err != nil {
			return err
		}
	}

	if err = s.db.Delete(&models.Tunnel{}, tunnel.ID).Error; err != nil {
		return fmt.Errorf("删除隧道失败: %w", err)
	}
	return nil
}

// Start 启动隧道。
func (s *TunnelService) Start(userID uint, id uint) error {
	tunnel, err := s.Get(userID, id)
	if err != nil {
		return err
	}
	if tunnel.Status == models.TunnelStatusRunning {
		return nil
	}

	entryGroup, err := s.getTunnelGroup(userID, tunnel.EntryGroupID, models.NodeGroupTypeEntry, true)
	if err != nil {
		return err
	}
	var exitGroup *models.NodeGroup

	// 检查是否为直连模式
	if tunnel.ExitGroupID == nil || *tunnel.ExitGroupID == 0 {
		// 直连模式，验证入口节点组是否支持
		entryConfig, err := entryGroup.GetConfig()
		if err != nil {
			return fmt.Errorf("获取入口节点组配置失败: %w", err)
		}
		if entryConfig.EntryConfig != nil && entryConfig.EntryConfig.RequireExitGroup {
			return fmt.Errorf("%w: 该入口节点组要求配置出口节点组，无法以直连模式启动", ErrInvalidParams)
		}
	} else {
		// 非直连模式，验证出口节点组
		exitGroup, err = s.getTunnelGroup(userID, *tunnel.ExitGroupID, models.NodeGroupTypeExit, true)
		if err != nil {
			return err
		}

		if err = s.validateTunnelRelation(entryGroup.ID, exitGroup.ID); err != nil {
			return err
		}
		if err = s.validateTunnelProtocol(entryGroup, exitGroup, tunnel.Protocol); err != nil {
			return err
		}
	}

	entryOnline, err := s.listOnlineInstances(entryGroup.ID)
	if err != nil {
		return err
	}
	if len(entryOnline) == 0 {
		return fmt.Errorf("%w: 入口组没有在线节点实例", ErrConflict)
	}

	var selectedExit *models.NodeInstance
	if exitGroup != nil {
		exitOnline, listErr := s.listOnlineInstances(exitGroup.ID)
		if listErr != nil {
			return listErr
		}
		if len(exitOnline) == 0 {
			return fmt.Errorf("%w: 出口组没有在线节点实例", ErrConflict)
		}

		exitCfg, cfgErr := exitGroup.GetConfig()
		if cfgErr != nil {
			return fmt.Errorf("解析出口组配置失败: %w", cfgErr)
		}
		strategy := ""
		if exitCfg.ExitConfig != nil {
			strategy = strings.TrimSpace(exitCfg.ExitConfig.LoadBalanceStrategy)
		}
		if strategy == "" {
			strategy = string(models.LoadBalanceRoundRobin)
		}

		selectedExit, err = selectExitInstanceForTunnel(strategy, exitOnline, uint(time.Now().UnixNano()))
		if err != nil {
			return err
		}
		if _, _, endpointErr := resolveNodeInstanceEndpoint(selectedExit, tunnel.ListenPort); endpointErr != nil {
			return fmt.Errorf("%w: 出口节点实例地址信息不完整: %v", ErrInvalidParams, endpointErr)
		}
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err = tx.Model(&models.Tunnel{}).Where("id = ?", tunnel.ID).Updates(map[string]interface{}{
		"status":     models.TunnelStatusRunning,
		"updated_at": time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新隧道状态失败: %w", err)
	}

	affectedGroupIDs := []uint{entryGroup.ID}
	if exitGroup != nil {
		affectedGroupIDs = append(affectedGroupIDs, exitGroup.ID)
	}
	idsToNotify, collectErr := s.collectAllInstanceIDsByGroups(tx, affectedGroupIDs)
	if collectErr != nil {
		tx.Rollback()
		return collectErr
	}
	if len(idsToNotify) > 0 {
		if err = tx.Model(&models.NodeInstance{}).Where("id IN ?", idsToNotify).
			Update("config_version", gorm.Expr("config_version + 1")).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("下发隧道配置失败: %w", err)
		}
	}

	if err = tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// Stop 停止隧道。
func (s *TunnelService) Stop(userID uint, id uint) error {
	tunnel, err := s.Get(userID, id)
	if err != nil {
		return err
	}
	if tunnel.Status == models.TunnelStatusStopped {
		return nil
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err = tx.Model(&models.Tunnel{}).Where("id = ?", tunnel.ID).Updates(map[string]interface{}{
		"status":     models.TunnelStatusStopped,
		"updated_at": time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("停止隧道失败: %w", err)
	}

	affectedGroupIDs := []uint{tunnel.EntryGroupID}
	if tunnel.ExitGroupID != nil && *tunnel.ExitGroupID > 0 {
		affectedGroupIDs = append(affectedGroupIDs, *tunnel.ExitGroupID)
	}
	idsToNotify, collectErr := s.collectAllInstanceIDsByGroups(tx, affectedGroupIDs)
	if collectErr != nil {
		tx.Rollback()
		return collectErr
	}
	if len(idsToNotify) > 0 {
		if err = tx.Model(&models.NodeInstance{}).Where("id IN ?", idsToNotify).
			Update("config_version", gorm.Expr("config_version + 1")).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("通知节点移除隧道规则失败: %w", err)
		}
	}

	if err = tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

func (s *TunnelService) getTunnelGroup(userID uint, groupID uint, expectedType models.NodeGroupType, mustEnabled bool) (*models.NodeGroup, error) {
	if groupID == 0 {
		return nil, fmt.Errorf("%w: 节点组 ID 无效", ErrInvalidParams)
	}

	query := s.db.Model(&models.NodeGroup{})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	var group models.NodeGroup
	if err := query.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 节点组不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询节点组失败: %w", err)
	}

	if expectedType != "" && group.Type != expectedType {
		return nil, fmt.Errorf("%w: 节点组类型不匹配，期望 %s", ErrInvalidParams, expectedType)
	}
	if mustEnabled && !group.IsEnabled {
		return nil, fmt.Errorf("%w: 节点组已禁用", ErrForbidden)
	}

	return &group, nil
}

func (s *TunnelService) validateTunnelRelation(entryGroupID uint, exitGroupID uint) error {
	var relation models.NodeGroupRelation
	if err := s.db.Where("entry_group_id = ? AND exit_group_id = ?", entryGroupID, exitGroupID).
		First(&relation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: 入口组与出口组未建立关联", ErrConflict)
		}
		return fmt.Errorf("查询节点组关联失败: %w", err)
	}
	if !relation.IsEnabled {
		return fmt.Errorf("%w: 节点组关联已禁用", ErrForbidden)
	}
	return nil
}

func (s *TunnelService) validateTunnelProtocol(entryGroup *models.NodeGroup, exitGroup *models.NodeGroup, protocol string) error {
	entryAllowed, err := tunnelAllowedProtocols(entryGroup)
	if err != nil {
		return err
	}
	exitAllowed, err := tunnelAllowedProtocols(exitGroup)
	if err != nil {
		return err
	}

	if _, ok := entryAllowed[protocol]; !ok {
		return fmt.Errorf("%w: 协议 %s 不在入口组允许范围", ErrInvalidParams, protocol)
	}
	if _, ok := exitAllowed[protocol]; !ok {
		return fmt.Errorf("%w: 协议 %s 不在出口组允许范围", ErrInvalidParams, protocol)
	}
	return nil
}

func (s *TunnelService) checkTunnelPortConflict(excludeID uint, entryGroupID uint, listenPort int) error {
	query := s.db.Model(&models.Tunnel{}).
		Where("entry_group_id = ? AND listen_port = ?", entryGroupID, listenPort)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return fmt.Errorf("检查监听端口冲突失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("%w: 同一入口组下 listen_port 已被占用", ErrConflict)
	}
	return nil
}

func (s *TunnelService) listOnlineInstances(groupID uint) ([]models.NodeInstance, error) {
	list := make([]models.NodeInstance, 0)
	if err := s.db.Model(&models.NodeInstance{}).
		Where("node_group_id = ? AND status = ? AND is_enabled = ?", groupID, models.NodeInstanceStatusOnline, true).
		Order("id ASC").
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询在线节点实例失败: %w", err)
	}
	return list, nil
}

func tunnelNormalizeProtocol(raw string) (string, error) {
	protocol := strings.ToLower(strings.TrimSpace(raw))
	switch protocol {
	case "tcp", "udp", "ws", "wss", "tls", "quic":
		return protocol, nil
	default:
		return "", fmt.Errorf("%w: protocol 不支持", ErrInvalidParams)
	}
}

func tunnelAllowedProtocols(group *models.NodeGroup) (map[string]struct{}, error) {
	if group == nil {
		return nil, fmt.Errorf("%w: 节点组为空", ErrInvalidParams)
	}

	cfg, err := group.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("解析节点组配置失败: %w", err)
	}

	result := make(map[string]struct{})
	for _, p := range cfg.AllowedProtocols {
		normalized := strings.ToLower(strings.TrimSpace(p))
		if normalized == "" {
			continue
		}
		result[normalized] = struct{}{}
	}

	if len(result) == 0 {
		result["tcp"] = struct{}{}
		result["udp"] = struct{}{}
	}
	return result, nil
}

func selectExitInstanceForTunnel(strategy string, list []models.NodeInstance, roundRobinSeed uint) (*models.NodeInstance, error) {
	if len(list) == 0 {
		return nil, fmt.Errorf("%w: 没有可用的出口节点实例", ErrConflict)
	}

	sorted := make([]models.NodeInstance, len(list))
	copy(sorted, list)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

	s := strings.ToLower(strings.TrimSpace(strategy))
	switch s {
	case "", string(models.LoadBalanceRoundRobin):
		if roundRobinSeed == 0 {
			return &sorted[0], nil
		}
		idx := int(roundRobinSeed % uint(len(sorted)))
		return &sorted[idx], nil
	case string(models.LoadBalanceLeastConnections):
		best := sorted[0]
		bestConn := tunnelExtractConnections(best.TrafficStats)
		for i := 1; i < len(sorted); i++ {
			conn := tunnelExtractConnections(sorted[i].TrafficStats)
			if conn < bestConn {
				best = sorted[i]
				bestConn = conn
			}
		}
		return &best, nil
	case string(models.LoadBalanceRandom):
		seeded := rand.New(rand.NewSource(time.Now().UnixNano()))
		chosen := sorted[seeded.Intn(len(sorted))]
		return &chosen, nil
	default:
		return nil, fmt.Errorf("%w: 不支持的负载均衡策略 %s", ErrInvalidParams, strategy)
	}
}

func tunnelExtractConnections(statsJSON string) int64 {
	raw := strings.TrimSpace(statsJSON)
	if raw == "" {
		return 0
	}
	payload := map[string]interface{}{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return 0
	}
	val, ok := payload["connections"]
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}

func collectInstanceIDs(instances []models.NodeInstance) []uint {
	ids := make([]uint, 0, len(instances))
	for _, item := range instances {
		ids = append(ids, item.ID)
	}
	return ids
}

func (s *TunnelService) collectAllInstanceIDsByGroups(tx *gorm.DB, groupIDs []uint) ([]uint, error) {
	uniqueGroupIDs := tunnelUniqueUint(groupIDs)
	if len(uniqueGroupIDs) == 0 {
		return []uint{}, nil
	}

	instances := make([]models.NodeInstance, 0)
	if err := tx.Model(&models.NodeInstance{}).
		Select("id").
		Where("node_group_id IN ?", uniqueGroupIDs).
		Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("查询节点实例失败: %w", err)
	}

	return tunnelUniqueUint(collectInstanceIDs(instances)), nil
}

// validateTunnelConfig 验证隧道配置。
func validateTunnelConfig(cfg *models.TunnelConfig) error {
	if cfg == nil {
		return nil
	}

	// 验证负载均衡策略
	validStrategies := map[models.LoadBalanceStrategy]bool{
		models.LoadBalanceRoundRobin:       true,
		models.LoadBalanceLeastConnections: true,
		models.LoadBalanceRandom:           true,
		models.LoadBalanceFailover:         true,
		models.LoadBalanceHash:             true,
		models.LoadBalanceLatency:          true,
	}
	if cfg.LoadBalanceStrategy != "" && !validStrategies[cfg.LoadBalanceStrategy] {
		return fmt.Errorf("%w: 不支持的负载均衡策略: %s", ErrInvalidParams, cfg.LoadBalanceStrategy)
	}

	// 验证IP类型
	if cfg.IPType != "" && cfg.IPType != "ipv4" && cfg.IPType != "ipv6" && cfg.IPType != "auto" {
		return fmt.Errorf("%w: ip_type 仅支持 ipv4/ipv6/auto", ErrInvalidParams)
	}

	// 验证转发目标
	for i, target := range cfg.ForwardTargets {
		if err := utils.ValidateHost(target.Host); err != nil {
			return fmt.Errorf("%w: forward_targets[%d].host 无效: %v", ErrInvalidParams, i, err)
		}
		if err := utils.ValidatePort(target.Port); err != nil {
			return fmt.Errorf("%w: forward_targets[%d].port 无效: %v", ErrInvalidParams, i, err)
		}
		if target.Weight < 0 {
			return fmt.Errorf("%w: forward_targets[%d].weight 不能为负数", ErrInvalidParams, i)
		}
	}

	// 验证协议配置
	if cfg.ProtocolConfig != nil {
		if err := validateProtocolConfig(cfg.ProtocolConfig); err != nil {
			return err
		}
	}

	return nil
}

// validateProtocolConfig 验证协议特定配置。
func validateProtocolConfig(cfg *models.ProtocolConfig) error {
	if cfg == nil {
		return nil
	}

	// 验证 TCP 配置
	if cfg.KeepaliveInterval != nil && (*cfg.KeepaliveInterval < 1 || *cfg.KeepaliveInterval > 300) {
		return fmt.Errorf("%w: keepalive_interval 必须在 1-300 秒之间", ErrInvalidParams)
	}
	if cfg.ConnectTimeout != nil && (*cfg.ConnectTimeout < 1 || *cfg.ConnectTimeout > 60) {
		return fmt.Errorf("%w: connect_timeout 必须在 1-60 秒之间", ErrInvalidParams)
	}
	if cfg.ReadTimeout != nil && (*cfg.ReadTimeout < 1 || *cfg.ReadTimeout > 300) {
		return fmt.Errorf("%w: read_timeout 必须在 1-300 秒之间", ErrInvalidParams)
	}

	// 验证 UDP 配置
	if cfg.BufferSize != nil && (*cfg.BufferSize < 1024 || *cfg.BufferSize > 65536) {
		return fmt.Errorf("%w: buffer_size 必须在 1024-65536 字节之间", ErrInvalidParams)
	}
	if cfg.SessionTimeout != nil && (*cfg.SessionTimeout < 10 || *cfg.SessionTimeout > 600) {
		return fmt.Errorf("%w: session_timeout 必须在 10-600 秒之间", ErrInvalidParams)
	}

	// 验证 WebSocket 配置
	if cfg.WSPath != nil && *cfg.WSPath != "" && !strings.HasPrefix(*cfg.WSPath, "/") {
		return fmt.Errorf("%w: ws_path 必须以 / 开头", ErrInvalidParams)
	}
	if cfg.PingInterval != nil && (*cfg.PingInterval < 5 || *cfg.PingInterval > 300) {
		return fmt.Errorf("%w: ping_interval 必须在 5-300 秒之间", ErrInvalidParams)
	}
	if cfg.MaxMessageSize != nil && (*cfg.MaxMessageSize < 1 || *cfg.MaxMessageSize > 10240) {
		return fmt.Errorf("%w: max_message_size 必须在 1-10240 KB 之间", ErrInvalidParams)
	}

	// 验证 TLS 配置
	if cfg.TLSVersion != nil {
		validVersions := map[string]bool{
			"tls1.0": true,
			"tls1.1": true,
			"tls1.2": true,
			"tls1.3": true,
		}
		if !validVersions[*cfg.TLSVersion] {
			return fmt.Errorf("%w: tls_version 必须是 tls1.0/tls1.1/tls1.2/tls1.3", ErrInvalidParams)
		}
	}

	// 验证 QUIC 配置
	if cfg.MaxStreams != nil && (*cfg.MaxStreams < 1 || *cfg.MaxStreams > 1000) {
		return fmt.Errorf("%w: max_streams 必须在 1-1000 之间", ErrInvalidParams)
	}
	if cfg.InitialWindow != nil && (*cfg.InitialWindow < 16 || *cfg.InitialWindow > 1024) {
		return fmt.Errorf("%w: initial_window 必须在 16-1024 KB 之间", ErrInvalidParams)
	}
	if cfg.IdleTimeout != nil && (*cfg.IdleTimeout < 10 || *cfg.IdleTimeout > 600) {
		return fmt.Errorf("%w: idle_timeout 必须在 10-600 秒之间", ErrInvalidParams)
	}

	return nil
}

func tunnelUniqueUint(values []uint) []uint {
	if len(values) == 0 {
		return values
	}
	set := make(map[uint]struct{}, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		set[value] = struct{}{}
	}
	result := make([]uint, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
