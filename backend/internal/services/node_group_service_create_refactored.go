//go:build refactor_experiment
// +build refactor_experiment

package services

import (
	"fmt"
	"strings"

	"nodepass-pro/backend/internal/models"

	"go.uber.org/zap"
)

// validateNodeGroupCreateRequest 验证创建节点组请求
func (s *NodeGroupService) validateNodeGroupCreateRequest(userID uint, req *CreateNodeGroupRequest) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("node group service 未初始化")
	}
	if userID == 0 {
		return fmt.Errorf("%w: user_id 无效", ErrInvalidParams)
	}
	if req == nil {
		return fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}
	return nil
}

// validateNodeGroupName 验证节点组名称
func validateNodeGroupName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: name 不能为空", ErrInvalidParams)
	}
	if len(name) > 100 {
		return "", fmt.Errorf("%w: name 长度不能超过 100", ErrInvalidParams)
	}
	return name, nil
}

// validateNodeGroupType 验证节点组类型
func validateNodeGroupType(groupType models.NodeGroupType) error {
	if groupType != models.NodeGroupTypeEntry && groupType != models.NodeGroupTypeExit {
		return fmt.Errorf("%w: type 仅支持 entry/exit", ErrInvalidParams)
	}
	return nil
}

// checkNodeGroupNameExists 检查节点组名称是否已存在
func (s *NodeGroupService) checkNodeGroupNameExists(userID uint, name string) error {
	var exists int64
	if err := s.db.Model(&models.NodeGroup{}).
		Where("user_id = ? AND name = ?", userID, name).
		Count(&exists).Error; err != nil {
		return fmt.Errorf("检查节点组重名失败: %w", err)
	}

	if exists > 0 {
		return fmt.Errorf("%w: 节点组名称已存在", ErrConflict)
	}

	return nil
}

// prepareNodeGroupConfig 准备节点组配置
func prepareNodeGroupConfig(config *models.NodeGroupConfig, groupType models.NodeGroupType) (*models.NodeGroupConfig, error) {
	cfg := normalizeNodeGroupConfig(config, groupType)
	if err := validateNodeGroupConfig(cfg, groupType); err != nil {
		return nil, err
	}
	return cfg, nil
}

// buildNodeGroup 构建节点组对象
func buildNodeGroup(userID uint, name string, groupType models.NodeGroupType, description *string, config *models.NodeGroupConfig) (*models.NodeGroup, error) {
	group := &models.NodeGroup{
		UserID:      userID,
		Name:        name,
		Type:        groupType,
		Description: normalizeOptionalStringNG(description),
		IsEnabled:   true,
	}

	if err := group.SetConfig(config); err != nil {
		return nil, fmt.Errorf("序列化节点组配置失败: %w", err)
	}

	return group, nil
}

// createNodeGroupStats 创建节点组统计
func createNodeGroupStats(groupID uint) *models.NodeGroupStats {
	return &models.NodeGroupStats{
		NodeGroupID:      groupID,
		TotalNodes:       0,
		OnlineNodes:      0,
		TotalTrafficIn:   0,
		TotalTrafficOut:  0,
		TotalConnections: 0,
	}
}

// executeNodeGroupCreateTransaction 执行创建节点组事务
func (s *NodeGroupService) executeNodeGroupCreateTransaction(group *models.NodeGroup) error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			zap.L().Error("事务执行 panic",
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	// 创建节点组
	if err := tx.Create(group).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("创建节点组失败: %w", err)
	}

	// 创建统计记录
	stats := createNodeGroupStats(group.ID)
	if err := tx.Create(stats).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("创建节点组统计失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// Create 创建节点组（重构后）
func (s *NodeGroupService) Create(userID uint, req *CreateNodeGroupRequest) (*models.NodeGroup, error) {
	// 1. 验证服务和请求
	if err := s.validateNodeGroupCreateRequest(userID, req); err != nil {
		return nil, err
	}

	// 2. 验证名称
	name, err := validateNodeGroupName(req.Name)
	if err != nil {
		return nil, err
	}

	// 3. 验证类型
	if err := validateNodeGroupType(req.Type); err != nil {
		return nil, err
	}

	// 4. 准备配置
	config, err := prepareNodeGroupConfig(req.Config, req.Type)
	if err != nil {
		return nil, err
	}

	// 5. 检查名称是否已存在
	if err := s.checkNodeGroupNameExists(userID, name); err != nil {
		return nil, err
	}

	// 6. 构建节点组对象
	group, err := buildNodeGroup(userID, name, req.Type, req.Description, config)
	if err != nil {
		return nil, err
	}

	// 7. 执行事务创建
	if err := s.executeNodeGroupCreateTransaction(group); err != nil {
		return nil, err
	}

	// 8. 返回完整的节点组信息
	return s.Get(userID, group.ID)
}
