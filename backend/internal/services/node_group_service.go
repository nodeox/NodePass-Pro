package services

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	deployServiceNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_.@-]{1,64}$`)
	allowedProtocols       = map[string]struct{}{
		"tcp":  {},
		"udp":  {},
		"ws":   {},
		"tls":  {},
		"quic": {},
	}
)

// NodeGroupService 节点组服务。
type NodeGroupService struct {
	db *gorm.DB
}

// NewNodeGroupService 创建节点组服务。
func NewNodeGroupService(db *gorm.DB) *NodeGroupService {
	return &NodeGroupService{db: db}
}

// CreateNodeGroupRequest 创建节点组请求。
type CreateNodeGroupRequest struct {
	Name        string                  `json:"name" binding:"required"`
	Type        models.NodeGroupType    `json:"type" binding:"required"`
	Description *string                 `json:"description"`
	Config      *models.NodeGroupConfig `json:"config"`
}

// UpdateNodeGroupRequest 更新节点组请求。
type UpdateNodeGroupRequest struct {
	Name        *string                 `json:"name"`
	Description *string                 `json:"description"`
	Config      *models.NodeGroupConfig `json:"config"`
	IsEnabled   *bool                   `json:"is_enabled"`
}

// ListNodeGroupParams 列表参数。
type ListNodeGroupParams struct {
	Type      *string `json:"type"`
	IsEnabled *bool   `json:"is_enabled"`
	Page      int     `json:"page"`
	PageSize  int     `json:"page_size"`
}

// DeployNodeRequest 部署节点请求。
type DeployNodeRequest struct {
	ServiceName string `json:"service_name"`
	DebugMode   bool   `json:"debug_mode"`
}

// DeployCommandResponse 部署命令响应。
type DeployCommandResponse struct {
	NodeID      string `json:"node_id"`
	Command     string `json:"command"`
	ServiceName string `json:"service_name"`
}

// AddNodeRequest 手动添加节点请求。
type AddNodeRequest struct {
	Name string `json:"name" binding:"required"`
	Host string `json:"host" binding:"required"`
	Port int    `json:"port" binding:"required"`
}

// NodeInstanceHeartbeatRequest 节点实例心跳请求。
type NodeInstanceHeartbeatRequest struct {
	NodeID            string   `json:"node_id"`
	Token             string   `json:"token"`
	ConnectionAddress *string  `json:"connection_address"`
	Status            *string  `json:"status"`
	CPUUsage          *float64 `json:"cpu_usage"`
	MemoryUsage       *float64 `json:"memory_usage"`
	DiskUsage         *float64 `json:"disk_usage"`
	BandwidthIn       *int64   `json:"bandwidth_in"`
	BandwidthOut      *int64   `json:"bandwidth_out"`
	Connections       *int     `json:"connections"`
}

// Create 创建节点组。
func (s *NodeGroupService) Create(userID uint, req *CreateNodeGroupRequest) (*models.NodeGroup, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("node group service 未初始化")
	}
	if userID == 0 {
		return nil, fmt.Errorf("%w: user_id 无效", ErrInvalidParams)
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
	if req.Type != models.NodeGroupTypeEntry && req.Type != models.NodeGroupTypeExit {
		return nil, fmt.Errorf("%w: type 仅支持 entry/exit", ErrInvalidParams)
	}

	cfg := normalizeNodeGroupConfig(req.Config, req.Type)
	if err := validateNodeGroupConfig(cfg, req.Type); err != nil {
		return nil, err
	}

	var exists int64
	if err := s.db.Model(&models.NodeGroup{}).
		Where("user_id = ? AND name = ?", userID, name).
		Count(&exists).Error; err != nil {
		return nil, fmt.Errorf("检查节点组重名失败: %w", err)
	}
	if exists > 0 {
		return nil, fmt.Errorf("%w: 节点组名称已存在", ErrConflict)
	}

	group := &models.NodeGroup{
		UserID:      userID,
		Name:        name,
		Type:        req.Type,
		Description: normalizeOptionalStringNG(req.Description),
		IsEnabled:   true,
	}
	if err := group.SetConfig(cfg); err != nil {
		return nil, fmt.Errorf("序列化节点组配置失败: %w", err)
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("开启事务失败: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			// 记录 panic 信息而非重新抛出
			zap.L().Error("事务执行 panic",
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	if err := tx.Create(group).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建节点组失败: %w", err)
	}

	stats := &models.NodeGroupStats{
		NodeGroupID:      group.ID,
		TotalNodes:       0,
		OnlineNodes:      0,
		TotalTrafficIn:   0,
		TotalTrafficOut:  0,
		TotalConnections: 0,
	}
	if err := tx.Create(stats).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建节点组统计失败: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return s.Get(userID, group.ID)
}

// List 列出节点组。
func (s *NodeGroupService) List(userID uint, params *ListNodeGroupParams) ([]models.NodeGroup, int64, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("node group service 未初始化")
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

	query := s.db.Model(&models.NodeGroup{})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if params != nil {
		if params.Type != nil {
			groupType := strings.TrimSpace(*params.Type)
			if groupType != "" {
				query = query.Where("type = ?", groupType)
			}
		}
		if params.IsEnabled != nil {
			query = query.Where("is_enabled = ?", *params.IsEnabled)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询节点组总数失败: %w", err)
	}

	list := make([]models.NodeGroup, 0, pageSize)
	if err := query.
		Preload("Stats").
		Preload("NodeInstances", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "node_group_id", "status")
		}).
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询节点组列表失败: %w", err)
	}

	return list, total, nil
}

// Get 获取节点组。
func (s *NodeGroupService) Get(userID uint, id uint) (*models.NodeGroup, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("node group service 未初始化")
	}
	if id == 0 {
		return nil, fmt.Errorf("%w: id 无效", ErrInvalidParams)
	}

	query := s.db.Model(&models.NodeGroup{}).
		Preload("Stats").
		Preload("NodeInstances")
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	var group models.NodeGroup
	if err := query.First(&group, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("%w: 节点组不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询节点组失败: %w", err)
	}

	return &group, nil
}

// Update 更新节点组。
func (s *NodeGroupService) Update(userID uint, id uint, req *UpdateNodeGroupRequest) (*models.NodeGroup, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("node group service 未初始化")
	}
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	group, err := s.Get(userID, id)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name 不能为空", ErrInvalidParams)
		}
		if len(name) > 100 {
			return nil, fmt.Errorf("%w: name 长度不能超过 100", ErrInvalidParams)
		}

		var exists int64
		if err = s.db.Model(&models.NodeGroup{}).
			Where("user_id = ? AND name = ? AND id <> ?", group.UserID, name, group.ID).
			Count(&exists).Error; err != nil {
			return nil, fmt.Errorf("检查节点组重名失败: %w", err)
		}
		if exists > 0 {
			return nil, fmt.Errorf("%w: 节点组名称已存在", ErrConflict)
		}

		updates["name"] = name
	}

	if req.Description != nil {
		updates["description"] = normalizeOptionalStringNG(req.Description)
	}

	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}

	if req.Config != nil {
		cfg := normalizeNodeGroupConfig(req.Config, group.Type)
		if err = validateNodeGroupConfig(cfg, group.Type); err != nil {
			return nil, err
		}

		serialized := &models.NodeGroup{}
		if err = serialized.SetConfig(cfg); err != nil {
			return nil, fmt.Errorf("序列化节点组配置失败: %w", err)
		}
		updates["config"] = serialized.Config
	}

	if len(updates) == 0 {
		return group, nil
	}

	if err = s.db.Model(&models.NodeGroup{}).Where("id = ?", group.ID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新节点组失败: %w", err)
	}

	return s.Get(userID, id)
}

// Delete 删除节点组。
func (s *NodeGroupService) Delete(userID uint, id uint) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("node group service 未初始化")
	}

	group, err := s.Get(userID, id)
	if err != nil {
		return err
	}

	var tunnelCount int64
	if err = s.db.Model(&models.Tunnel{}).
		Where("entry_group_id = ? OR exit_group_id = ?", group.ID, group.ID).
		Count(&tunnelCount).Error; err != nil {
		return fmt.Errorf("检查关联隧道失败: %w", err)
	}
	if tunnelCount > 0 {
		return fmt.Errorf("%w: 节点组存在关联隧道，不能删除", ErrConflict)
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

	if err = tx.Where("node_group_id = ?", group.ID).Delete(&models.NodeInstance{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除节点实例失败: %w", err)
	}
	if err = tx.Where("node_group_id = ?", group.ID).Delete(&models.NodeGroupStats{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除节点组统计失败: %w", err)
	}
	if err = tx.Where("entry_group_id = ? OR exit_group_id = ?", group.ID, group.ID).Delete(&models.NodeGroupRelation{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除节点组关联失败: %w", err)
	}
	if err = tx.Delete(&models.NodeGroup{}, group.ID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除节点组失败: %w", err)
	}

	if err = tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// Toggle 切换启用状态。
func (s *NodeGroupService) Toggle(userID uint, id uint) (*models.NodeGroup, error) {
	group, err := s.Get(userID, id)
	if err != nil {
		return nil, err
	}

	nextEnabled := !group.IsEnabled
	if err = s.db.Model(&models.NodeGroup{}).Where("id = ?", group.ID).Update("is_enabled", nextEnabled).Error; err != nil {
		return nil, fmt.Errorf("切换节点组状态失败: %w", err)
	}

	return s.Get(userID, id)
}

// GetStats 获取节点组统计。
func (s *NodeGroupService) GetStats(userID uint, id uint) (*models.NodeGroupStats, error) {
	group, err := s.Get(userID, id)
	if err != nil {
		return nil, err
	}

	if group.Stats != nil {
		return group.Stats, nil
	}

	stats := &models.NodeGroupStats{}
	if err = s.db.Where("node_group_id = ?", group.ID).First(stats).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &models.NodeGroupStats{NodeGroupID: group.ID}, nil
		}
		return nil, fmt.Errorf("查询节点组统计失败: %w", err)
	}
	return stats, nil
}

// GenerateDeployCommand 生成部署命令并创建离线节点实例。
func (s *NodeGroupService) GenerateDeployCommand(userID uint, groupID uint, req *DeployNodeRequest) (*DeployCommandResponse, error) {
	group, err := s.Get(userID, groupID)
	if err != nil {
		return nil, err
	}
	if !group.IsEnabled {
		return nil, fmt.Errorf("%w: 节点组已禁用", ErrForbidden)
	}

	if req == nil {
		req = &DeployNodeRequest{}
	}

	nodeID := uuid.NewString()
	authToken, err := generateAuthToken()
	if err != nil {
		return nil, err
	}
	serviceName := strings.TrimSpace(req.ServiceName)
	if serviceName == "" {
		serviceName = fmt.Sprintf("nodeclient-%d-%s", group.ID, nodeID[:8])
	}
	if !deployServiceNameRegex.MatchString(serviceName) {
		return nil, fmt.Errorf("%w: service_name 格式无效，仅支持字母数字和 _.@-", ErrInvalidParams)
	}

	instance := &models.NodeInstance{
		NodeGroupID:   group.ID,
		NodeID:        nodeID,
		AuthTokenHash: hashTokenSHA256(authToken),
		Name:          serviceName,
		Status:        models.NodeInstanceStatusOffline,
		IsEnabled:     true,
		SystemInfo:    "{}",
		TrafficStats:  "{}",
		ConfigVersion: 0,
	}
	if err = s.db.Create(instance).Error; err != nil {
		return nil, fmt.Errorf("创建节点实例失败: %w", err)
	}

	if err = s.recalculateGroupStats(group.ID); err != nil {
		return nil, err
	}

	hubURL := resolveHubURL("")
	scriptURL := strings.TrimRight(hubURL, "/") + "/nodeclient-install.sh"

	var cmd strings.Builder
	cmd.WriteString("bash <(curl -fsSL ")
	cmd.WriteString(shellEscapeNG(scriptURL))
	cmd.WriteString(")")
	cmd.WriteString(" --hub-url ")
	cmd.WriteString(shellEscapeNG(hubURL))
	cmd.WriteString(" --node-id ")
	cmd.WriteString(shellEscapeNG(nodeID))
	cmd.WriteString(" --group-id ")
	cmd.WriteString(fmt.Sprintf("%d", group.ID))
	cmd.WriteString(" --token ")
	cmd.WriteString(shellEscapeNG(authToken))
	cmd.WriteString(" --service-name ")
	cmd.WriteString(shellEscapeNG(serviceName))
	if req.DebugMode {
		cmd.WriteString(" --debug")
	}

	return &DeployCommandResponse{
		NodeID:      nodeID,
		Command:     cmd.String(),
		ServiceName: serviceName,
	}, nil
}

// ListNodes 列出节点组下的节点实例。
func (s *NodeGroupService) ListNodes(userID uint, groupID uint) ([]models.NodeInstance, error) {
	group, err := s.Get(userID, groupID)
	if err != nil {
		return nil, err
	}

	list := make([]models.NodeInstance, 0)
	if err = s.db.Where("node_group_id = ?", group.ID).Order("id DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询节点实例失败: %w", err)
	}

	return list, nil
}

// AddNode 手动添加节点实例。
func (s *NodeGroupService) AddNode(userID uint, groupID uint, req *AddNodeRequest) (*models.NodeInstance, error) {
	group, err := s.Get(userID, groupID)
	if err != nil {
		return nil, err
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

	host := strings.TrimSpace(req.Host)
	if err = utils.ValidateHost(host); err != nil {
		return nil, fmt.Errorf("%w: host 无效: %v", ErrInvalidParams, err)
	}
	if err = utils.ValidatePort(req.Port); err != nil {
		return nil, fmt.Errorf("%w: port 无效: %v", ErrInvalidParams, err)
	}

	nodeID := uuid.NewString()
	authToken, err := generateAuthToken()
	if err != nil {
		return nil, err
	}
	instance := &models.NodeInstance{
		NodeGroupID:   group.ID,
		NodeID:        nodeID,
		AuthTokenHash: hashTokenSHA256(authToken),
		Name:          name,
		Host:          &host,
		Port:          &req.Port,
		Status:        models.NodeInstanceStatusOffline,
		IsEnabled:     true,
		SystemInfo:    "{}",
		TrafficStats:  "{}",
		ConfigVersion: 0,
	}
	if err = s.db.Create(instance).Error; err != nil {
		return nil, fmt.Errorf("创建节点实例失败: %w", err)
	}

	if err = s.recalculateGroupStats(group.ID); err != nil {
		return nil, err
	}

	return instance, nil
}

// CreateRelation 创建节点组关联（入口组 -> 出口组）。
func (s *NodeGroupService) CreateRelation(userID uint, entryGroupID uint, exitGroupID uint) (*models.NodeGroupRelation, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("node group service 未初始化")
	}
	if userID == 0 {
		return nil, fmt.Errorf("%w: user_id 无效", ErrInvalidParams)
	}
	if entryGroupID == 0 || exitGroupID == 0 {
		return nil, fmt.Errorf("%w: entry_group_id 和 exit_group_id 必须大于 0", ErrInvalidParams)
	}

	entryGroup, err := s.Get(userID, entryGroupID)
	if err != nil {
		return nil, err
	}
	if entryGroup.Type != models.NodeGroupTypeEntry {
		return nil, fmt.Errorf("%w: entry_group_id 必须是入口组", ErrInvalidParams)
	}

	exitGroup, err := s.Get(userID, exitGroupID)
	if err != nil {
		return nil, err
	}
	if exitGroup.Type != models.NodeGroupTypeExit {
		return nil, fmt.Errorf("%w: exit_group_id 必须是出口组", ErrInvalidParams)
	}

	var count int64
	if err = s.db.Model(&models.NodeGroupRelation{}).
		Where("entry_group_id = ? AND exit_group_id = ?", entryGroupID, exitGroupID).
		Count(&count).Error; err != nil {
		return nil, fmt.Errorf("检查节点组关联重复失败: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("%w: 节点组关联已存在", ErrConflict)
	}

	relation := &models.NodeGroupRelation{
		EntryGroupID: entryGroupID,
		ExitGroupID:  exitGroupID,
		IsEnabled:    true,
	}
	if err = s.db.Create(relation).Error; err != nil {
		return nil, fmt.Errorf("创建节点组关联失败: %w", err)
	}

	created, err := s.getOwnedRelation(userID, relation.ID)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// ListRelations 列出某节点组的全部关联（作为入口组或出口组）。
func (s *NodeGroupService) ListRelations(userID uint, groupID uint) ([]models.NodeGroupRelation, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("node group service 未初始化")
	}
	if groupID == 0 {
		return nil, fmt.Errorf("%w: group_id 无效", ErrInvalidParams)
	}

	if _, err := s.Get(userID, groupID); err != nil {
		return nil, err
	}

	list := make([]models.NodeGroupRelation, 0)
	if err := s.db.Model(&models.NodeGroupRelation{}).
		Preload("EntryGroup").
		Preload("ExitGroup").
		Where("entry_group_id = ? OR exit_group_id = ?", groupID, groupID).
		Order("id DESC").
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询节点组关联失败: %w", err)
	}

	return list, nil
}

// DeleteRelation 删除节点组关联。
func (s *NodeGroupService) DeleteRelation(userID uint, relationID uint) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("node group service 未初始化")
	}
	if relationID == 0 {
		return fmt.Errorf("%w: relation_id 无效", ErrInvalidParams)
	}

	relation, err := s.getOwnedRelation(userID, relationID)
	if err != nil {
		return err
	}

	var activeTunnelCount int64
	if err = s.db.Model(&models.Tunnel{}).
		Where("entry_group_id = ? AND exit_group_id = ? AND status = ?",
			relation.EntryGroupID, relation.ExitGroupID, models.TunnelStatusRunning).
		Count(&activeTunnelCount).Error; err != nil {
		return fmt.Errorf("检查关联隧道失败: %w", err)
	}
	if activeTunnelCount > 0 {
		return fmt.Errorf("%w: 存在活跃隧道依赖该关联，无法删除", ErrConflict)
	}

	if err = s.db.Delete(&models.NodeGroupRelation{}, relation.ID).Error; err != nil {
		return fmt.Errorf("删除节点组关联失败: %w", err)
	}

	return nil
}

// ToggleRelation 切换节点组关联启用状态。
func (s *NodeGroupService) ToggleRelation(userID uint, relationID uint) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("node group service 未初始化")
	}
	if relationID == 0 {
		return fmt.Errorf("%w: relation_id 无效", ErrInvalidParams)
	}

	relation, err := s.getOwnedRelation(userID, relationID)
	if err != nil {
		return err
	}

	nextEnabled := !relation.IsEnabled
	if err = s.db.Model(&models.NodeGroupRelation{}).
		Where("id = ?", relation.ID).
		Update("is_enabled", nextEnabled).Error; err != nil {
		return fmt.Errorf("切换节点组关联状态失败: %w", err)
	}

	return nil
}

func (s *NodeGroupService) getOwnedRelation(userID uint, relationID uint) (*models.NodeGroupRelation, error) {
	if relationID == 0 {
		return nil, fmt.Errorf("%w: relation_id 无效", ErrInvalidParams)
	}

	query := s.db.Model(&models.NodeGroupRelation{}).
		Preload("EntryGroup").
		Preload("ExitGroup")
	if userID > 0 {
		query = query.Joins("JOIN node_groups entry_groups ON entry_groups.id = node_group_relations.entry_group_id").
			Where("entry_groups.user_id = ?", userID)
	}

	var relation models.NodeGroupRelation
	if err := query.First(&relation, relationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 节点组关联不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询节点组关联失败: %w", err)
	}

	if relation.EntryGroup == nil || relation.ExitGroup == nil {
		return nil, fmt.Errorf("%w: 节点组关联数据不完整", ErrNotFound)
	}
	if userID > 0 && (relation.EntryGroup.UserID != userID || relation.ExitGroup.UserID != userID) {
		return nil, fmt.Errorf("%w: 无权操作该节点组关联", ErrForbidden)
	}

	return &relation, nil
}

// GetNodeInstance 获取节点实例。
func (s *NodeGroupService) GetNodeInstance(scopeUserID uint, instanceID uint) (*models.NodeInstance, error) {
	if instanceID == 0 {
		return nil, fmt.Errorf("%w: 节点实例 ID 无效", ErrInvalidParams)
	}

	query := s.db.Model(&models.NodeInstance{}).Preload("NodeGroup")
	if scopeUserID > 0 {
		query = query.Joins("JOIN node_groups ON node_groups.id = node_instances.node_group_id").
			Where("node_groups.user_id = ?", scopeUserID)
	}

	var instance models.NodeInstance
	if err := query.First(&instance, instanceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("%w: 节点实例不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询节点实例失败: %w", err)
	}
	return &instance, nil
}

// DeleteNodeInstance 删除节点实例。
func (s *NodeGroupService) DeleteNodeInstance(scopeUserID uint, instanceID uint) error {
	instance, err := s.GetNodeInstance(scopeUserID, instanceID)
	if err != nil {
		return err
	}
	if err = s.db.Delete(&models.NodeInstance{}, instance.ID).Error; err != nil {
		return fmt.Errorf("删除节点实例失败: %w", err)
	}
	return s.recalculateGroupStats(instance.NodeGroupID)
}

// RestartNodeInstance 重启节点实例（状态置为 offline）。
func (s *NodeGroupService) RestartNodeInstance(scopeUserID uint, instanceID uint) (*models.NodeInstance, error) {
	instance, err := s.GetNodeInstance(scopeUserID, instanceID)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"status":            models.NodeInstanceStatusOffline,
		"last_heartbeat_at": nil,
	}
	if err = s.db.Model(&models.NodeInstance{}).Where("id = ?", instance.ID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("重启节点实例失败: %w", err)
	}
	if err = s.recalculateGroupStats(instance.NodeGroupID); err != nil {
		return nil, err
	}
	return s.GetNodeInstance(scopeUserID, instance.ID)
}

// HandleNodeInstanceHeartbeat 处理节点实例心跳。
func (s *NodeGroupService) HandleNodeInstanceHeartbeat(req *NodeInstanceHeartbeatRequest) (*models.NodeInstance, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}
	nodeID := strings.TrimSpace(req.NodeID)
	if nodeID == "" {
		return nil, fmt.Errorf("%w: node_id 不能为空", ErrInvalidParams)
	}

	var instance models.NodeInstance
	if err := s.db.Where("node_id = ?", nodeID).First(&instance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("%w: 节点实例不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询节点实例失败: %w", err)
	}
	token := strings.TrimSpace(req.Token)
	if token == "" {
		return nil, fmt.Errorf("%w: token 不能为空", ErrUnauthorized)
	}
	if strings.TrimSpace(instance.AuthTokenHash) == "" {
		return nil, fmt.Errorf("%w: 节点认证令牌未初始化", ErrForbidden)
	}
	tokenHash := hashTokenSHA256(token)
	if subtle.ConstantTimeCompare([]byte(tokenHash), []byte(instance.AuthTokenHash)) != 1 {
		return nil, fmt.Errorf("%w: 节点认证失败", ErrUnauthorized)
	}

	// 禁用的节点不允许心跳
	if !instance.IsEnabled {
		return nil, fmt.Errorf("%w: 节点已被禁用", ErrForbidden)
	}

	previousStatus := instance.Status
	previousHost := ""
	if instance.Host != nil {
		previousHost = strings.TrimSpace(*instance.Host)
	}
	previousPort := 0
	if instance.Port != nil {
		previousPort = *instance.Port
	}

	status := models.NodeInstanceStatusOnline
	if !instance.IsEnabled {
		// 禁用实例不允许上报为 online，避免继续参与规则调度。
		status = models.NodeInstanceStatusOffline
	} else if req.Status != nil {
		parsed := strings.ToLower(strings.TrimSpace(*req.Status))
		switch parsed {
		case string(models.NodeInstanceStatusOnline), string(models.NodeInstanceStatusOffline), string(models.NodeInstanceStatusMaintain):
			status = models.NodeInstanceStatus(parsed)
		default:
			return nil, fmt.Errorf("%w: status 无效", ErrInvalidParams)
		}
	}

	sysInfoRaw, err := buildSystemInfoJSON(req)
	if err != nil {
		return nil, err
	}
	trafficRaw, err := buildTrafficStatsJSON(req)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"status":            status,
		"last_heartbeat_at": time.Now(),
	}
	nextHost := previousHost
	nextPort := previousPort
	if host, port, ok := parseConnectionAddress(req.ConnectionAddress); ok {
		updates["host"] = host
		nextHost = host
		if port > 0 {
			updates["port"] = port
			nextPort = port
		}
	}
	if sysInfoRaw != "" {
		updates["system_info"] = sysInfoRaw
	}
	if trafficRaw != "" {
		updates["traffic_stats"] = trafficRaw
	}

	if err = s.db.Model(&models.NodeInstance{}).Where("id = ?", instance.ID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新节点心跳失败: %w", err)
	}

	if err = s.recalculateGroupStats(instance.NodeGroupID); err != nil {
		return nil, err
	}

	statusChanged := previousStatus != status
	endpointChanged := previousHost != nextHost || previousPort != nextPort
	if statusChanged || endpointChanged {
		if err = s.BumpConfigVersionForGroupAndDependents([]uint{instance.NodeGroupID}); err != nil {
			return nil, err
		}
	}

	return s.GetNodeInstance(0, instance.ID)
}

// GenerateNodeConfigForInstance 按节点实例生成最新运行配置（纯新链路）。
func (s *NodeGroupService) GenerateNodeConfigForInstance(instanceID uint) (*NodeConfig, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("node group service 未初始化")
	}
	if instanceID == 0 {
		return nil, fmt.Errorf("%w: 节点实例 ID 无效", ErrInvalidParams)
	}

	instance, err := s.GetNodeInstance(0, instanceID)
	if err != nil {
		return nil, err
	}

	settings, err := s.loadNodeRuntimeSettings()
	if err != nil {
		return nil, err
	}

	nodeConfig := &NodeConfig{
		ConfigVersion: instance.ConfigVersion,
		Rules:         make([]RuleConfig, 0),
		Settings:      settings,
	}
	if !instance.IsEnabled {
		// 禁用实例只下发空规则，确保节点端尽快撤销本地转发。
		return nodeConfig, nil
	}

	groupType := models.NodeGroupType("")
	if instance.NodeGroup != nil {
		groupType = instance.NodeGroup.Type
	} else {
		var group models.NodeGroup
		if err = s.db.Select("id", "type").First(&group, instance.NodeGroupID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("%w: 节点组不存在", ErrNotFound)
			}
			return nil, fmt.Errorf("查询节点组失败: %w", err)
		}
		groupType = group.Type
	}

	// 目前仅入口组实例负责承载转发规则；出口组用于被入口组选择，不直接下发本地监听规则。
	if groupType != models.NodeGroupTypeEntry {
		return nodeConfig, nil
	}

	tunnels := make([]models.Tunnel, 0)
	if err = s.db.Model(&models.Tunnel{}).
		Where("entry_group_id = ? AND status = ?", instance.NodeGroupID, models.TunnelStatusRunning).
		Order("id ASC").
		Find(&tunnels).Error; err != nil {
		return nil, fmt.Errorf("查询运行中隧道失败: %w", err)
	}

	for _, tunnel := range tunnels {
		if tunnel.ListenPort <= 0 {
			continue
		}

		listenHost := strings.TrimSpace(tunnel.ListenHost)
		if listenHost == "" {
			listenHost = "0.0.0.0"
		}

		protocol := strings.ToLower(strings.TrimSpace(tunnel.Protocol))
		if protocol == "" {
			protocol = "tcp"
		}

		rule := RuleConfig{
			RuleID: int(tunnel.ID),
			Mode:   "single",
			Listen: HostPort{
				Host: listenHost,
				Port: tunnel.ListenPort,
			},
			Target: HostPort{
				Host: tunnel.RemoteHost,
				Port: tunnel.RemotePort,
			},
			Protocol: protocol,
		}

		if tunnel.ExitGroupID != nil && *tunnel.ExitGroupID > 0 {
			exitOnline, endpointErr := s.listOnlineEnabledInstances(*tunnel.ExitGroupID)
			if endpointErr != nil {
				return nil, endpointErr
			}
			if len(exitOnline) == 0 {
				return nil, fmt.Errorf("%w: 隧道 %d 对应出口组无可用在线节点", ErrConflict, tunnel.ID)
			}
			strategy, strategyErr := s.resolveExitGroupLoadBalanceStrategy(*tunnel.ExitGroupID)
			if strategyErr != nil {
				return nil, strategyErr
			}
			seed := tunnel.ID + instance.ID
			if instance.ConfigVersion > 0 {
				seed += uint(instance.ConfigVersion)
			}
			selectedExit, selectErr := selectExitInstanceForTunnel(strategy, exitOnline, seed)
			if selectErr != nil {
				return nil, fmt.Errorf("%w: 隧道 %d 选择出口失败: %v", ErrConflict, tunnel.ID, selectErr)
			}
			exitHost, exitPort, resolveErr := resolveNodeInstanceEndpoint(selectedExit, tunnel.ListenPort)
			if resolveErr != nil {
				return nil, fmt.Errorf("%w: 隧道 %d 缺少可用出口地址: %v", ErrConflict, tunnel.ID, resolveErr)
			}
			rule.Mode = "tunnel"
			rule.ExitNode = &HostPort{
				Host: exitHost,
				Port: exitPort,
			}
		}

		nodeConfig.Rules = append(nodeConfig.Rules, rule)
	}

	return nodeConfig, nil
}

// MarkOfflineByHeartbeat 将超时节点实例标记为离线。
func (s *NodeGroupService) MarkOfflineByHeartbeat(timeout time.Duration) (int64, error) {
	if timeout <= 0 {
		timeout = 3 * time.Minute
	}

	cutoff := time.Now().Add(-timeout)
	instances := make([]models.NodeInstance, 0)
	if err := s.db.Model(&models.NodeInstance{}).
		Where("status <> ? AND last_heartbeat_at IS NOT NULL AND last_heartbeat_at < ?", models.NodeInstanceStatusOffline, cutoff).
		Find(&instances).Error; err != nil {
		return 0, fmt.Errorf("查询超时节点实例失败: %w", err)
	}
	if len(instances) == 0 {
		return 0, nil
	}

	ids := make([]uint, 0, len(instances))
	groupSet := make(map[uint]struct{})
	for _, item := range instances {
		ids = append(ids, item.ID)
		groupSet[item.NodeGroupID] = struct{}{}
	}

	res := s.db.Model(&models.NodeInstance{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{"status": models.NodeInstanceStatusOffline})
	if res.Error != nil {
		return 0, fmt.Errorf("更新超时节点实例状态失败: %w", res.Error)
	}

	for groupID := range groupSet {
		if err := s.recalculateGroupStats(groupID); err != nil {
			return res.RowsAffected, err
		}
	}
	affectedGroups := make([]uint, 0, len(groupSet))
	for groupID := range groupSet {
		affectedGroups = append(affectedGroups, groupID)
	}
	if err := s.BumpConfigVersionForGroupAndDependents(affectedGroups); err != nil {
		return res.RowsAffected, err
	}

	return res.RowsAffected, nil
}

func (s *NodeGroupService) loadNodeRuntimeSettings() (Settings, error) {
	settings := Settings{
		HeartbeatInterval:   30,
		ConfigCheckInterval: 60,
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

func (s *NodeGroupService) listOnlineEnabledInstances(groupID uint) ([]models.NodeInstance, error) {
	instances := make([]models.NodeInstance, 0)
	if err := s.db.Model(&models.NodeInstance{}).
		Where("node_group_id = ? AND status = ? AND is_enabled = ?", groupID, models.NodeInstanceStatusOnline, true).
		Order("id ASC").
		Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("查询在线节点实例失败: %w", err)
	}
	return instances, nil
}

func (s *NodeGroupService) resolveExitGroupLoadBalanceStrategy(exitGroupID uint) (string, error) {
	group := &models.NodeGroup{}
	if err := s.db.Model(&models.NodeGroup{}).
		Select("id", "config").
		Where("id = ?", exitGroupID).
		First(group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("%w: 出口组不存在", ErrNotFound)
		}
		return "", fmt.Errorf("查询出口组配置失败: %w", err)
	}

	cfg, err := group.GetConfig()
	if err != nil {
		return "", fmt.Errorf("解析出口组配置失败: %w", err)
	}
	if cfg != nil && cfg.ExitConfig != nil {
		strategy := strings.ToLower(strings.TrimSpace(cfg.ExitConfig.LoadBalanceStrategy))
		if strategy != "" {
			return strategy, nil
		}
	}
	return string(models.LoadBalanceRoundRobin), nil
}

func (s *NodeGroupService) recalculateGroupStats(groupID uint) error {
	if groupID == 0 {
		return nil
	}

	instances := make([]models.NodeInstance, 0)
	if err := s.db.Model(&models.NodeInstance{}).
		Select("id", "status", "traffic_stats", "system_info").
		Where("node_group_id = ?", groupID).
		Find(&instances).Error; err != nil {
		return fmt.Errorf("查询节点实例失败: %w", err)
	}

	total := int64(len(instances))
	var online int64
	var totalTrafficIn int64
	var totalTrafficOut int64
	var totalConnections int64
	for _, item := range instances {
		if item.Status == models.NodeInstanceStatusOnline {
			online++
		}
		trafficIn, trafficOut, connections := parseNodeInstanceAggregates(item.TrafficStats, item.SystemInfo)
		totalTrafficIn += trafficIn
		totalTrafficOut += trafficOut
		totalConnections += connections
	}

	stats := &models.NodeGroupStats{}
	err := s.db.Where("node_group_id = ?", groupID).First(stats).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询节点组统计失败: %w", err)
	}

	if err == gorm.ErrRecordNotFound {
		stats = &models.NodeGroupStats{
			NodeGroupID:      groupID,
			TotalNodes:       int(total),
			OnlineNodes:      int(online),
			TotalTrafficIn:   totalTrafficIn,
			TotalTrafficOut:  totalTrafficOut,
			TotalConnections: int(totalConnections),
		}
		if createErr := s.db.Create(stats).Error; createErr != nil {
			return fmt.Errorf("创建节点组统计失败: %w", createErr)
		}
		return nil
	}

	updates := map[string]interface{}{
		"total_nodes":       int(total),
		"online_nodes":      int(online),
		"total_traffic_in":  totalTrafficIn,
		"total_traffic_out": totalTrafficOut,
		"total_connections": int(totalConnections),
		"updated_at":        time.Now(),
	}
	if updateErr := s.db.Model(&models.NodeGroupStats{}).
		Where("node_group_id = ?", groupID).
		Updates(updates).Error; updateErr != nil {
		return fmt.Errorf("更新节点组统计失败: %w", updateErr)
	}

	return nil
}

// BumpConfigVersionForGroupAndDependents 推进指定节点组及其依赖入口组的配置版本。
func (s *NodeGroupService) BumpConfigVersionForGroupAndDependents(groupIDs []uint) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("node group service 未初始化")
	}
	expanded, err := s.expandGroupIDsWithDependents(groupIDs)
	if err != nil {
		return err
	}
	if len(expanded) == 0 {
		return nil
	}
	if err := s.db.Model(&models.NodeInstance{}).
		Where("node_group_id IN ?", expanded).
		Update("config_version", gorm.Expr("config_version + 1")).Error; err != nil {
		return fmt.Errorf("推进节点配置版本失败: %w", err)
	}
	return nil
}

func (s *NodeGroupService) expandGroupIDsWithDependents(groupIDs []uint) ([]uint, error) {
	ids := uniqueUint(groupIDs)
	if len(ids) == 0 {
		return ids, nil
	}

	// 当出口组节点状态变化时，关联入口组也需要重算并下发配置。
	dependentEntryGroups := make([]uint, 0)
	if err := s.db.Model(&models.Tunnel{}).
		Distinct("entry_group_id").
		Where("status = ? AND exit_group_id IN ?", models.TunnelStatusRunning, ids).
		Pluck("entry_group_id", &dependentEntryGroups).Error; err != nil {
		return nil, fmt.Errorf("查询受影响入口组失败: %w", err)
	}

	merged := append(ids, dependentEntryGroups...)
	return uniqueUint(merged), nil
}

func parseNodeInstanceAggregates(trafficStatsJSON string, systemInfoJSON string) (int64, int64, int64) {
	trafficIn := extractInt64FromJSON(trafficStatsJSON, "traffic_in", "bandwidth_in")
	trafficOut := extractInt64FromJSON(trafficStatsJSON, "traffic_out", "bandwidth_out")
	connections := extractInt64FromJSON(trafficStatsJSON, "connections", "active_connections")
	if connections == 0 {
		connections = extractInt64FromJSON(systemInfoJSON, "connections")
	}
	return trafficIn, trafficOut, connections
}

func extractInt64FromJSON(raw string, keys ...string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return 0
	}

	payload := map[string]interface{}{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return 0
	}

	for _, key := range keys {
		value, exists := payload[key]
		if !exists {
			continue
		}
		switch v := value.(type) {
		case float64:
			return int64(v)
		case float32:
			return int64(v)
		case int:
			return int64(v)
		case int32:
			return int64(v)
		case int64:
			return v
		case uint:
			return int64(v)
		case uint32:
			return int64(v)
		case uint64:
			if v > uint64(^uint64(0)>>1) {
				return int64(^uint64(0) >> 1)
			}
			return int64(v)
		case string:
			if parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
				return parsed
			}
		}
	}

	return 0
}

func normalizeNodeGroupConfig(input *models.NodeGroupConfig, groupType models.NodeGroupType) *models.NodeGroupConfig {
	cfg := &models.NodeGroupConfig{}
	if input != nil {
		*cfg = *input
	}

	if len(cfg.AllowedProtocols) == 0 {
		cfg.AllowedProtocols = []string{"tcp", "udp"}
	}

	if groupType == models.NodeGroupTypeEntry {
		if cfg.EntryConfig == nil {
			cfg.EntryConfig = &models.EntryGroupConfig{TrafficMultiplier: 1.0}
		}
		if cfg.EntryConfig.TrafficMultiplier <= 0 {
			cfg.EntryConfig.TrafficMultiplier = 1.0
		}
	}

	if groupType == models.NodeGroupTypeExit {
		if cfg.ExitConfig == nil {
			cfg.ExitConfig = &models.ExitGroupConfig{
				LoadBalanceStrategy: string(models.LoadBalanceRoundRobin),
				HealthCheckInterval: 30,
				HealthCheckTimeout:  5,
			}
		}
		if strings.TrimSpace(cfg.ExitConfig.LoadBalanceStrategy) == "" {
			cfg.ExitConfig.LoadBalanceStrategy = string(models.LoadBalanceRoundRobin)
		}
		if cfg.ExitConfig.HealthCheckInterval <= 0 {
			cfg.ExitConfig.HealthCheckInterval = 30
		}
		if cfg.ExitConfig.HealthCheckTimeout <= 0 {
			cfg.ExitConfig.HealthCheckTimeout = 5
		}
	}

	return cfg
}

func validateNodeGroupConfig(cfg *models.NodeGroupConfig, groupType models.NodeGroupType) error {
	if cfg == nil {
		return nil
	}

	if cfg.PortRange.Start > 0 {
		if err := utils.ValidatePort(cfg.PortRange.Start); err != nil {
			return fmt.Errorf("%w: port_range.start 无效: %v", ErrInvalidParams, err)
		}
	}
	if cfg.PortRange.End > 0 {
		if err := utils.ValidatePort(cfg.PortRange.End); err != nil {
			return fmt.Errorf("%w: port_range.end 无效: %v", ErrInvalidParams, err)
		}
	}
	if cfg.PortRange.Start > 0 && cfg.PortRange.End > 0 && cfg.PortRange.Start >= cfg.PortRange.End {
		return fmt.Errorf("%w: port_range.start 必须小于 port_range.end", ErrInvalidParams)
	}

	for _, protocol := range cfg.AllowedProtocols {
		normalized := strings.ToLower(strings.TrimSpace(protocol))
		if normalized == "" {
			continue
		}
		if _, ok := allowedProtocols[normalized]; !ok {
			return fmt.Errorf("%w: 不支持的协议 %s", ErrInvalidParams, protocol)
		}
	}

	if groupType == models.NodeGroupTypeEntry {
		if cfg.ExitConfig != nil {
			return fmt.Errorf("%w: entry 节点组不能配置 exit_config", ErrInvalidParams)
		}
		if cfg.EntryConfig != nil && cfg.EntryConfig.TrafficMultiplier <= 0 {
			return fmt.Errorf("%w: traffic_multiplier 必须大于 0", ErrInvalidParams)
		}
	}

	if groupType == models.NodeGroupTypeExit {
		if cfg.EntryConfig != nil {
			return fmt.Errorf("%w: exit 节点组不能配置 entry_config", ErrInvalidParams)
		}

		if cfg.ExitConfig != nil {
			strategy := strings.ToLower(strings.TrimSpace(cfg.ExitConfig.LoadBalanceStrategy))
			switch strategy {
			case string(models.LoadBalanceRoundRobin), string(models.LoadBalanceLeastConnections), string(models.LoadBalanceRandom):
			default:
				return fmt.Errorf("%w: load_balance_strategy 仅支持 round_robin/least_connections/random", ErrInvalidParams)
			}
			if cfg.ExitConfig.HealthCheckInterval <= 0 {
				return fmt.Errorf("%w: health_check_interval 必须大于 0", ErrInvalidParams)
			}
			if cfg.ExitConfig.HealthCheckTimeout <= 0 {
				return fmt.Errorf("%w: health_check_timeout 必须大于 0", ErrInvalidParams)
			}
		}
	}

	return nil
}

func normalizeOptionalStringNG(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func resolveHubURL(raw string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		candidate = strings.TrimSpace(os.Getenv("NODEPASS_HUB_URL"))
	}
	if candidate == "" {
		candidate = "http://127.0.0.1:8080"
	}
	if !strings.Contains(candidate, "://") {
		candidate = "https://" + candidate
	}
	return strings.TrimRight(candidate, "/")
}

func shellEscapeNG(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func buildSystemInfoJSON(req *NodeInstanceHeartbeatRequest) (string, error) {
	sysInfo := map[string]interface{}{}
	if req.CPUUsage != nil {
		sysInfo["cpu_usage"] = *req.CPUUsage
	}
	if req.MemoryUsage != nil {
		sysInfo["memory_usage"] = *req.MemoryUsage
	}
	if req.DiskUsage != nil {
		sysInfo["disk_usage"] = *req.DiskUsage
	}
	if len(sysInfo) == 0 {
		return "", nil
	}
	data, err := json.Marshal(sysInfo)
	if err != nil {
		return "", fmt.Errorf("序列化系统信息失败: %w", err)
	}
	return string(data), nil
}

func buildTrafficStatsJSON(req *NodeInstanceHeartbeatRequest) (string, error) {
	traffic := map[string]interface{}{}
	if req.BandwidthIn != nil {
		traffic["bandwidth_in"] = *req.BandwidthIn
	}
	if req.BandwidthOut != nil {
		traffic["bandwidth_out"] = *req.BandwidthOut
	}
	if req.Connections != nil {
		traffic["connections"] = *req.Connections
	}
	if len(traffic) == 0 {
		return "", nil
	}
	data, err := json.Marshal(traffic)
	if err != nil {
		return "", fmt.Errorf("序列化流量统计失败: %w", err)
	}
	return string(data), nil
}

func parseConnectionAddress(raw *string) (string, int, bool) {
	if raw == nil {
		return "", 0, false
	}

	candidate := strings.TrimSpace(*raw)
	if candidate == "" || strings.EqualFold(candidate, "auto") {
		return "", 0, false
	}

	parsePort := func(portRaw string) (int, bool) {
		parsed, err := strconv.Atoi(strings.TrimSpace(portRaw))
		if err != nil || parsed <= 0 || parsed > 65535 {
			return 0, false
		}
		return parsed, true
	}

	if host, portRaw, err := net.SplitHostPort(candidate); err == nil {
		host = strings.Trim(strings.TrimSpace(host), "[]")
		if host == "" {
			return "", 0, false
		}
		port, ok := parsePort(portRaw)
		if !ok {
			return host, 0, true
		}
		return host, port, true
	}

	if strings.Count(candidate, ":") == 1 {
		hostPart, portPart, ok := strings.Cut(candidate, ":")
		hostPart = strings.Trim(strings.TrimSpace(hostPart), "[]")
		if ok && hostPart != "" {
			if port, parsed := parsePort(portPart); parsed {
				return hostPart, port, true
			}
			return hostPart, 0, true
		}
	}

	hostOnly := strings.Trim(strings.TrimSpace(candidate), "[]")
	if hostOnly == "" {
		return "", 0, false
	}
	return hostOnly, 0, true
}

func resolveNodeInstanceEndpoint(instance *models.NodeInstance, fallbackPort int) (string, int, error) {
	if instance == nil {
		return "", 0, fmt.Errorf("%w: 节点实例为空", ErrInvalidParams)
	}
	if instance.Host == nil || strings.TrimSpace(*instance.Host) == "" {
		return "", 0, fmt.Errorf("%w: 节点实例缺少 host", ErrInvalidParams)
	}

	host := strings.TrimSpace(*instance.Host)
	port := 0
	if instance.Port != nil && *instance.Port > 0 {
		port = *instance.Port
	} else if fallbackPort > 0 {
		port = fallbackPort
	}
	if port <= 0 {
		return "", 0, fmt.Errorf("%w: 节点实例缺少 port", ErrInvalidParams)
	}

	return host, port, nil
}

func uniqueUint(values []uint) []uint {
	if len(values) == 0 {
		return values
	}
	set := make(map[uint]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	result := make([]uint, 0, len(set))
	for v := range set {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
