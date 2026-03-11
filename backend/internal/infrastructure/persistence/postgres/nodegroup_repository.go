package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"

	"nodepass-pro/backend/internal/domain/nodegroup"
	"nodepass-pro/backend/internal/models"
)

// NodeGroupRepository PostgreSQL 节点组仓储实现
type NodeGroupRepository struct {
	db *gorm.DB
}

// NewNodeGroupRepository 创建仓储实例
func NewNodeGroupRepository(db *gorm.DB) nodegroup.Repository {
	return &NodeGroupRepository{db: db}
}

// Create 创建节点组
func (r *NodeGroupRepository) Create(ctx context.Context, group *nodegroup.NodeGroup) error {
	model := r.toModel(group)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	group.ID = model.ID
	return nil
}

// FindByID 根据 ID 查找节点组
func (r *NodeGroupRepository) FindByID(ctx context.Context, id uint) (*nodegroup.NodeGroup, error) {
	var model models.NodeGroup
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nodegroup.ErrNodeGroupNotFound
		}
		return nil, err
	}
	return r.toDomain(&model)
}

// Update 更新节点组
func (r *NodeGroupRepository) Update(ctx context.Context, group *nodegroup.NodeGroup) error {
	model := r.toModel(group)
	result := r.db.WithContext(ctx).Model(&models.NodeGroup{}).
		Where("id = ?", group.ID).
		Updates(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nodegroup.ErrNodeGroupNotFound
	}
	return nil
}

// Delete 删除节点组
func (r *NodeGroupRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.NodeGroup{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nodegroup.ErrNodeGroupNotFound
	}
	return nil
}

// FindByUserID 根据用户 ID 查找节点组
func (r *NodeGroupRepository) FindByUserID(ctx context.Context, userID uint) ([]*nodegroup.NodeGroup, error) {
	var modelList []models.NodeGroup
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&modelList).Error; err != nil {
		return nil, err
	}
	return r.toDomainList(modelList)
}

// FindByType 根据类型查找节点组
func (r *NodeGroupRepository) FindByType(ctx context.Context, groupType nodegroup.NodeGroupType) ([]*nodegroup.NodeGroup, error) {
	var modelList []models.NodeGroup
	if err := r.db.WithContext(ctx).
		Where("type = ?", groupType).
		Find(&modelList).Error; err != nil {
		return nil, err
	}
	return r.toDomainList(modelList)
}

// FindByUserIDAndType 根据用户 ID 和类型查找节点组
func (r *NodeGroupRepository) FindByUserIDAndType(ctx context.Context, userID uint, groupType nodegroup.NodeGroupType) ([]*nodegroup.NodeGroup, error) {
	var modelList []models.NodeGroup
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND type = ?", userID, groupType).
		Find(&modelList).Error; err != nil {
		return nil, err
	}
	return r.toDomainList(modelList)
}

// List 列表查询
func (r *NodeGroupRepository) List(ctx context.Context, filter nodegroup.ListFilter) ([]*nodegroup.NodeGroup, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.NodeGroup{})

	// 应用过滤条件
	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.EnabledOnly {
		query = query.Where("is_enabled = ?", true)
	}
	if filter.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var modelList []models.NodeGroup
	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Offset(offset).Limit(filter.PageSize).Find(&modelList).Error; err != nil {
		return nil, 0, err
	}

	groups, err := r.toDomainList(modelList)
	return groups, total, err
}

// CountByUserID 统计用户的节点组数量
func (r *NodeGroupRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.NodeGroup{}).
		Where("user_id = ?", userID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// UpdateStats 更新节点组统计
func (r *NodeGroupRepository) UpdateStats(ctx context.Context, stats *nodegroup.NodeGroupStats) error {
	model := &models.NodeGroupStats{
		NodeGroupID:      stats.NodeGroupID,
		TotalNodes:       stats.TotalNodes,
		OnlineNodes:      stats.OnlineNodes,
		TotalTrafficIn:   stats.TotalTrafficIn,
		TotalTrafficOut:  stats.TotalTrafficOut,
		TotalConnections: stats.TotalConnections,
		UpdatedAt:        stats.UpdatedAt,
	}

	return r.db.WithContext(ctx).
		Where("node_group_id = ?", stats.NodeGroupID).
		Assign(model).
		FirstOrCreate(&models.NodeGroupStats{NodeGroupID: stats.NodeGroupID}).Error
}

// GetStats 获取节点组统计
func (r *NodeGroupRepository) GetStats(ctx context.Context, groupID uint) (*nodegroup.NodeGroupStats, error) {
	var model models.NodeGroupStats
	if err := r.db.WithContext(ctx).
		Where("node_group_id = ?", groupID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 返回空统计
			return &nodegroup.NodeGroupStats{
				NodeGroupID: groupID,
			}, nil
		}
		return nil, err
	}

	return &nodegroup.NodeGroupStats{
		NodeGroupID:      model.NodeGroupID,
		TotalNodes:       model.TotalNodes,
		OnlineNodes:      model.OnlineNodes,
		TotalTrafficIn:   model.TotalTrafficIn,
		TotalTrafficOut:  model.TotalTrafficOut,
		TotalConnections: model.TotalConnections,
		UpdatedAt:        model.UpdatedAt,
	}, nil
}

// toModel 转换为数据库模型
func (r *NodeGroupRepository) toModel(group *nodegroup.NodeGroup) *models.NodeGroup {
	configJSON, _ := json.Marshal(group.Config)
	var desc *string
	if group.Description != "" {
		desc = &group.Description
	}
	return &models.NodeGroup{
		ID:          group.ID,
		UserID:      group.UserID,
		Name:        group.Name,
		Type:        models.NodeGroupType(group.Type),
		Description: desc,
		IsEnabled:   group.IsEnabled,
		Config:      string(configJSON),
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
	}
}

// toDomain 转换为领域实体
func (r *NodeGroupRepository) toDomain(model *models.NodeGroup) (*nodegroup.NodeGroup, error) {
	var config nodegroup.NodeGroupConfig
	if err := json.Unmarshal([]byte(model.Config), &config); err != nil {
		return nil, err
	}

	desc := ""
	if model.Description != nil {
		desc = *model.Description
	}

	return &nodegroup.NodeGroup{
		ID:          model.ID,
		UserID:      model.UserID,
		Name:        model.Name,
		Type:        nodegroup.NodeGroupType(model.Type),
		Description: desc,
		IsEnabled:   model.IsEnabled,
		Config:      config,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}, nil
}

// toDomainList 转换为领域实体列表
func (r *NodeGroupRepository) toDomainList(modelList []models.NodeGroup) ([]*nodegroup.NodeGroup, error) {
	groups := make([]*nodegroup.NodeGroup, 0, len(modelList))
	for i := range modelList {
		group, err := r.toDomain(&modelList[i])
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}
