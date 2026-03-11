package postgres

import (
	"context"
	"errors"
	
	"nodepass-pro/backend/internal/domain/tunnel"
	"nodepass-pro/backend/internal/models"
	
	"gorm.io/gorm"
)

// TunnelRepository PostgreSQL 隧道仓储实现
type TunnelRepository struct {
	db *gorm.DB
}

// NewTunnelRepository 创建隧道仓储
func NewTunnelRepository(db *gorm.DB) tunnel.Repository {
	return &TunnelRepository{db: db}
}

// Create 创建隧道
func (r *TunnelRepository) Create(ctx context.Context, t *tunnel.Tunnel) error {
	model := toTunnelModel(t)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	t.ID = model.ID
	t.CreatedAt = model.CreatedAt
	t.UpdatedAt = model.UpdatedAt
	return nil
}

// FindByID 根据 ID 查找隧道
func (r *TunnelRepository) FindByID(ctx context.Context, id uint) (*tunnel.Tunnel, error) {
	var model models.Tunnel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tunnel.ErrTunnelNotFound
		}
		return nil, err
	}
	return toTunnelEntity(&model), nil
}

// Update 更新隧道
func (r *TunnelRepository) Update(ctx context.Context, t *tunnel.Tunnel) error {
	model := toTunnelModel(t)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete 删除隧道
func (r *TunnelRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Tunnel{}, id).Error
}

// FindByUserID 根据用户 ID 查找隧道
func (r *TunnelRepository) FindByUserID(ctx context.Context, userID uint) ([]*tunnel.Tunnel, error) {
	var models []models.Tunnel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error; err != nil {
		return nil, err
	}
	
	tunnels := make([]*tunnel.Tunnel, len(models))
	for i, m := range models {
		tunnels[i] = toTunnelEntity(&m)
	}
	return tunnels, nil
}

// FindByIDs 批量查找隧道
func (r *TunnelRepository) FindByIDs(ctx context.Context, ids []uint) ([]*tunnel.Tunnel, error) {
	var models []models.Tunnel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&models).Error; err != nil {
		return nil, err
	}
	
	tunnels := make([]*tunnel.Tunnel, len(models))
	for i, m := range models {
		tunnels[i] = toTunnelEntity(&m)
	}
	return tunnels, nil
}

// List 列表查询
func (r *TunnelRepository) List(ctx context.Context, filter tunnel.ListFilter) ([]*tunnel.Tunnel, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Tunnel{})
	
	// 过滤条件
	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Protocol != "" {
		query = query.Where("protocol = ?", filter.Protocol)
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", keyword, keyword)
	}
	if filter.EnabledOnly {
		query = query.Where("is_enabled = ?", true)
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
	
	var models []models.Tunnel
	if err := query.Offset(offset).Limit(filter.PageSize).Order("id DESC").Find(&models).Error; err != nil {
		return nil, 0, err
	}
	
	tunnels := make([]*tunnel.Tunnel, len(models))
	for i, m := range models {
		tunnels[i] = toTunnelEntity(&m)
	}
	
	return tunnels, total, nil
}

// FindRunningTunnels 查找运行中的隧道
func (r *TunnelRepository) FindRunningTunnels(ctx context.Context) ([]*tunnel.Tunnel, error) {
	var models []models.Tunnel
	if err := r.db.WithContext(ctx).
		Where("status = ? AND is_enabled = ?", "running", true).
		Find(&models).Error; err != nil {
		return nil, err
	}
	
	tunnels := make([]*tunnel.Tunnel, len(models))
	for i, m := range models {
		tunnels[i] = toTunnelEntity(&m)
	}
	return tunnels, nil
}

// FindByPort 根据端口查找隧道
func (r *TunnelRepository) FindByPort(ctx context.Context, port int) (*tunnel.Tunnel, error) {
	var model models.Tunnel
	if err := r.db.WithContext(ctx).Where("listen_port = ?", port).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tunnel.ErrTunnelNotFound
		}
		return nil, err
	}
	return toTunnelEntity(&model), nil
}

// CountByUserID 统计用户隧道数
func (r *TunnelRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Tunnel{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// UpdateTraffic 更新流量统计
func (r *TunnelRepository) UpdateTraffic(ctx context.Context, tunnelID uint, inBytes, outBytes int64) error {
	return r.db.WithContext(ctx).Model(&models.Tunnel{}).
		Where("id = ?", tunnelID).
		Updates(map[string]interface{}{
			"traffic_in":  gorm.Expr("traffic_in + ?", inBytes),
			"traffic_out": gorm.Expr("traffic_out + ?", outBytes),
		}).Error
}

// BatchUpdateTraffic 批量更新流量统计
func (r *TunnelRepository) BatchUpdateTraffic(ctx context.Context, data map[uint]tunnel.TrafficData) error {
	if len(data) == 0 {
		return nil
	}
	
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for tunnelID, traffic := range data {
			if err := tx.Model(&models.Tunnel{}).
				Where("id = ?", tunnelID).
				Updates(map[string]interface{}{
					"traffic_in":  gorm.Expr("traffic_in + ?", traffic.InBytes),
					"traffic_out": gorm.Expr("traffic_out + ?", traffic.OutBytes),
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// toTunnelEntity 模型转实体
func toTunnelEntity(m *models.Tunnel) *tunnel.Tunnel {
	description := ""
	if m.Description != nil {
		description = *m.Description
	}

	var exitNodeID uint
	if m.ExitGroupID != nil {
		exitNodeID = *m.ExitGroupID
	}

	return &tunnel.Tunnel{
		ID:          m.ID,
		UserID:      m.UserID,
		Name:        m.Name,
		Description: description,
		Protocol:    m.Protocol,
		Mode:        "single", // 默认模式
		ListenHost:  m.ListenHost,
		ListenPort:  m.ListenPort,
		TargetHost:  m.RemoteHost,
		TargetPort:  m.RemotePort,
		EntryNodeID: m.EntryGroupID,
		ExitNodeID:  exitNodeID,
		Status:      string(m.Status),
		IsEnabled:   true, // 默认启用
		TrafficIn:   m.TrafficIn,
		TrafficOut:  m.TrafficOut,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// toTunnelModel 实体转模型
func toTunnelModel(t *tunnel.Tunnel) *models.Tunnel {
	var description *string
	if t.Description != "" {
		description = &t.Description
	}

	var exitGroupID *uint
	if t.ExitNodeID > 0 {
		exitGroupID = &t.ExitNodeID
	}

	return &models.Tunnel{
		ID:           t.ID,
		UserID:       t.UserID,
		Name:         t.Name,
		Description:  description,
		EntryGroupID: t.EntryNodeID,
		ExitGroupID:  exitGroupID,
		Protocol:     t.Protocol,
		ListenHost:   t.ListenHost,
		ListenPort:   t.ListenPort,
		RemoteHost:   t.TargetHost,
		RemotePort:   t.TargetPort,
		Status:       models.TunnelStatus(t.Status),
		TrafficIn:    t.TrafficIn,
		TrafficOut:   t.TrafficOut,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}
