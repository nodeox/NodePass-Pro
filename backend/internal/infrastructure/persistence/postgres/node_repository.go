package postgres

import (
	"context"
	"errors"
	"time"
	
	"nodepass-pro/backend/internal/domain/node"
	"nodepass-pro/backend/internal/models"
	
	"gorm.io/gorm"
)

// NodeInstanceRepository PostgreSQL 节点实例仓储实现
type NodeInstanceRepository struct {
	db *gorm.DB
}

// NewNodeInstanceRepository 创建节点实例仓储
func NewNodeInstanceRepository(db *gorm.DB) node.InstanceRepository {
	return &NodeInstanceRepository{db: db}
}

// Create 创建节点实例
func (r *NodeInstanceRepository) Create(ctx context.Context, instance *node.NodeInstance) error {
	model := toNodeInstanceModel(instance)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	instance.ID = model.ID
	instance.CreatedAt = model.CreatedAt
	instance.UpdatedAt = model.UpdatedAt
	return nil
}

// FindByID 根据 ID 查找节点实例
func (r *NodeInstanceRepository) FindByID(ctx context.Context, id uint) (*node.NodeInstance, error) {
	var model models.NodeInstance
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, node.ErrNodeNotFound
		}
		return nil, err
	}
	return toNodeInstanceEntity(&model), nil
}

// FindByNodeID 根据 NodeID 查找节点实例
func (r *NodeInstanceRepository) FindByNodeID(ctx context.Context, nodeID string) (*node.NodeInstance, error) {
	var model models.NodeInstance
	if err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, node.ErrNodeNotFound
		}
		return nil, err
	}
	return toNodeInstanceEntity(&model), nil
}

// Update 更新节点实例
func (r *NodeInstanceRepository) Update(ctx context.Context, instance *node.NodeInstance) error {
	model := toNodeInstanceModel(instance)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete 删除节点实例
func (r *NodeInstanceRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.NodeInstance{}, id).Error
}

// FindByGroupID 根据组 ID 查找节点实例
func (r *NodeInstanceRepository) FindByGroupID(ctx context.Context, groupID uint) ([]*node.NodeInstance, error) {
	var models []models.NodeInstance
	if err := r.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&models).Error; err != nil {
		return nil, err
	}
	
	instances := make([]*node.NodeInstance, len(models))
	for i, m := range models {
		instances[i] = toNodeInstanceEntity(&m)
	}
	return instances, nil
}

// FindByIDs 批量查找节点实例
func (r *NodeInstanceRepository) FindByIDs(ctx context.Context, ids []uint) ([]*node.NodeInstance, error) {
	var models []models.NodeInstance
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&models).Error; err != nil {
		return nil, err
	}
	
	instances := make([]*node.NodeInstance, len(models))
	for i, m := range models {
		instances[i] = toNodeInstanceEntity(&m)
	}
	return instances, nil
}

// List 列表查询
func (r *NodeInstanceRepository) List(ctx context.Context, filter node.InstanceListFilter) ([]*node.NodeInstance, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.NodeInstance{})
	
	// 过滤条件
	if filter.GroupID > 0 {
		query = query.Where("group_id = ?", filter.GroupID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where("node_id LIKE ? OR service_name LIKE ?", keyword, keyword)
	}
	if filter.OnlineOnly {
		// 3 分钟内有心跳认为在线
		query = query.Where("last_heartbeat_at > ?", time.Now().Add(-3*time.Minute))
	}
	
	// 总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// 分页
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PageSize
	
	var models []models.NodeInstance
	if err := query.Offset(offset).Limit(filter.PageSize).Order("id DESC").Find(&models).Error; err != nil {
		return nil, 0, err
	}
	
	instances := make([]*node.NodeInstance, len(models))
	for i, m := range models {
		instances[i] = toNodeInstanceEntity(&m)
	}
	
	return instances, total, nil
}

// FindOnlineNodes 查找在线节点
func (r *NodeInstanceRepository) FindOnlineNodes(ctx context.Context) ([]*node.NodeInstance, error) {
	var models []models.NodeInstance
	if err := r.db.WithContext(ctx).
		Where("last_heartbeat_at > ?", time.Now().Add(-3*time.Minute)).
		Find(&models).Error; err != nil {
		return nil, err
	}
	
	instances := make([]*node.NodeInstance, len(models))
	for i, m := range models {
		instances[i] = toNodeInstanceEntity(&m)
	}
	return instances, nil
}

// FindOfflineNodes 查找离线节点
func (r *NodeInstanceRepository) FindOfflineNodes(ctx context.Context, timeout time.Duration) ([]*node.NodeInstance, error) {
	var models []models.NodeInstance
	if err := r.db.WithContext(ctx).
		Where("last_heartbeat_at < ? OR last_heartbeat_at IS NULL", time.Now().Add(-timeout)).
		Where("status != ?", "offline").
		Find(&models).Error; err != nil {
		return nil, err
	}
	
	instances := make([]*node.NodeInstance, len(models))
	for i, m := range models {
		instances[i] = toNodeInstanceEntity(&m)
	}
	return instances, nil
}

// CountByStatus 按状态统计
func (r *NodeInstanceRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeInstance{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

// UpdateHeartbeat 更新心跳信息
func (r *NodeInstanceRepository) UpdateHeartbeat(ctx context.Context, nodeID string, data *node.HeartbeatData) error {
	updates := map[string]interface{}{
		"last_heartbeat_at": data.Timestamp,
		"cpu_usage":         data.CPUUsage,
		"memory_usage":      data.MemoryUsage,
		"disk_usage":        data.DiskUsage,
		"traffic_in":        data.TrafficIn,
		"traffic_out":       data.TrafficOut,
		"active_rules":      data.ActiveRules,
		"config_version":    data.ConfigVersion,
		"client_version":    data.ClientVersion,
		"status":            "online",
	}
	
	return r.db.WithContext(ctx).Model(&models.NodeInstance{}).
		Where("node_id = ?", nodeID).
		Updates(updates).Error
}

// BatchUpdateHeartbeat 批量更新心跳信息
func (r *NodeInstanceRepository) BatchUpdateHeartbeat(ctx context.Context, data []*node.HeartbeatData) error {
	if len(data) == 0 {
		return nil
	}
	
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, hb := range data {
			updates := map[string]interface{}{
				"last_heartbeat_at": hb.Timestamp,
				"cpu_usage":         hb.CPUUsage,
				"memory_usage":      hb.MemoryUsage,
				"disk_usage":        hb.DiskUsage,
				"traffic_in":        hb.TrafficIn,
				"traffic_out":       hb.TrafficOut,
				"active_rules":      hb.ActiveRules,
				"config_version":    hb.ConfigVersion,
				"client_version":    hb.ClientVersion,
				"status":            "online",
			}
			
			if err := tx.Model(&models.NodeInstance{}).
				Where("node_id = ?", hb.NodeID).
				Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// MarkOfflineByTimeout 标记超时节点为离线
func (r *NodeInstanceRepository) MarkOfflineByTimeout(ctx context.Context, timeout time.Duration) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.NodeInstance{}).
		Where("last_heartbeat_at < ?", time.Now().Add(-timeout)).
		Where("status != ?", "offline").
		Update("status", "offline")
	
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// toNodeInstanceEntity 模型转实体
func toNodeInstanceEntity(m *models.NodeInstance) *node.NodeInstance {
	return &node.NodeInstance{
		ID:              m.ID,
		GroupID:         m.GroupID,
		NodeID:          m.NodeID,
		ServiceName:     m.ServiceName,
		Status:          m.Status,
		ConnectionAddr:  m.ConnectionAddr,
		ExitNetwork:     m.ExitNetwork,
		LastHeartbeatAt: m.LastHeartbeatAt,
		ConfigVersion:   m.ConfigVersion,
		ClientVersion:   m.ClientVersion,
		CPUUsage:        m.CPUUsage,
		MemoryUsage:     m.MemoryUsage,
		DiskUsage:       m.DiskUsage,
		TrafficIn:       m.TrafficIn,
		TrafficOut:      m.TrafficOut,
		ActiveRules:     m.ActiveRules,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

// toNodeInstanceModel 实体转模型
func toNodeInstanceModel(n *node.NodeInstance) *models.NodeInstance {
	return &models.NodeInstance{
		ID:              n.ID,
		GroupID:         n.GroupID,
		NodeID:          n.NodeID,
		ServiceName:     n.ServiceName,
		Status:          n.Status,
		ConnectionAddr:  n.ConnectionAddr,
		ExitNetwork:     n.ExitNetwork,
		LastHeartbeatAt: n.LastHeartbeatAt,
		ConfigVersion:   n.ConfigVersion,
		ClientVersion:   n.ClientVersion,
		CPUUsage:        n.CPUUsage,
		MemoryUsage:     n.MemoryUsage,
		DiskUsage:       n.DiskUsage,
		TrafficIn:       n.TrafficIn,
		TrafficOut:      n.TrafficOut,
		ActiveRules:     n.ActiveRules,
		CreatedAt:       n.CreatedAt,
		UpdatedAt:       n.UpdatedAt,
	}
}
