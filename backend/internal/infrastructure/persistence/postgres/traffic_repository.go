package postgres

import (
	"context"
	"time"
	
	"nodepass-pro/backend/internal/domain/traffic"
	"nodepass-pro/backend/internal/models"
	
	"gorm.io/gorm"
)

// TrafficRecordRepository PostgreSQL 流量记录仓储实现
type TrafficRecordRepository struct {
	db *gorm.DB
}

// NewTrafficRecordRepository 创建流量记录仓储
func NewTrafficRecordRepository(db *gorm.DB) traffic.RecordRepository {
	return &TrafficRecordRepository{db: db}
}

// Create 创建流量记录
func (r *TrafficRecordRepository) Create(ctx context.Context, record *traffic.TrafficRecord) error {
	model := toTrafficRecordModel(record)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	record.ID = model.ID
	record.CreatedAt = model.CreatedAt
	return nil
}

// BatchCreate 批量创建流量记录
func (r *TrafficRecordRepository) BatchCreate(ctx context.Context, records []*traffic.TrafficRecord) error {
	if len(records) == 0 {
		return nil
	}
	
	models := make([]*models.TrafficRecord, len(records))
	for i, record := range records {
		models[i] = toTrafficRecordModel(record)
	}
	
	// 批量插入（每次 100 条）
	return r.db.WithContext(ctx).CreateInBatches(models, 100).Error
}

// FindByID 根据 ID 查找流量记录
func (r *TrafficRecordRepository) FindByID(ctx context.Context, id uint) (*traffic.TrafficRecord, error) {
	var model models.TrafficRecord
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		return nil, err
	}
	return toTrafficRecordEntity(&model), nil
}

// FindByUserID 根据用户 ID 查找流量记录
func (r *TrafficRecordRepository) FindByUserID(ctx context.Context, userID uint, start, end time.Time) ([]*traffic.TrafficRecord, error) {
	var models []models.TrafficRecord
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	
	if !start.IsZero() {
		query = query.Where("recorded_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("recorded_at <= ?", end)
	}
	
	if err := query.Order("recorded_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	
	records := make([]*traffic.TrafficRecord, len(models))
	for i, m := range models {
		records[i] = toTrafficRecordEntity(&m)
	}
	return records, nil
}

// FindByTunnelID 根据隧道 ID 查找流量记录
func (r *TrafficRecordRepository) FindByTunnelID(ctx context.Context, tunnelID uint, start, end time.Time) ([]*traffic.TrafficRecord, error) {
	var models []models.TrafficRecord
	query := r.db.WithContext(ctx).Where("tunnel_id = ?", tunnelID)
	
	if !start.IsZero() {
		query = query.Where("recorded_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("recorded_at <= ?", end)
	}
	
	if err := query.Order("recorded_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	
	records := make([]*traffic.TrafficRecord, len(models))
	for i, m := range models {
		records[i] = toTrafficRecordEntity(&m)
	}
	return records, nil
}

// List 列表查询
func (r *TrafficRecordRepository) List(ctx context.Context, filter traffic.RecordListFilter) ([]*traffic.TrafficRecord, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.TrafficRecord{})
	
	// 过滤条件
	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.TunnelID > 0 {
		query = query.Where("tunnel_id = ?", filter.TunnelID)
	}
	if !filter.StartTime.IsZero() {
		query = query.Where("recorded_at >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query = query.Where("recorded_at <= ?", filter.EndTime)
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
	
	var models []models.TrafficRecord
	if err := query.Offset(offset).Limit(filter.PageSize).Order("recorded_at DESC").Find(&models).Error; err != nil {
		return nil, 0, err
	}
	
	records := make([]*traffic.TrafficRecord, len(models))
	for i, m := range models {
		records[i] = toTrafficRecordEntity(&m)
	}
	
	return records, total, nil
}

// SumByUserID 统计用户流量
func (r *TrafficRecordRepository) SumByUserID(ctx context.Context, userID uint, start, end time.Time) (int64, int64, error) {
	var result struct {
		TotalIn  int64
		TotalOut int64
	}
	
	query := r.db.WithContext(ctx).Model(&models.TrafficRecord{}).
		Select("COALESCE(SUM(traffic_in), 0) as total_in, COALESCE(SUM(traffic_out), 0) as total_out").
		Where("user_id = ?", userID)
	
	if !start.IsZero() {
		query = query.Where("recorded_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("recorded_at <= ?", end)
	}
	
	if err := query.Scan(&result).Error; err != nil {
		return 0, 0, err
	}
	
	return result.TotalIn, result.TotalOut, nil
}

// SumByTunnelID 统计隧道流量
func (r *TrafficRecordRepository) SumByTunnelID(ctx context.Context, tunnelID uint, start, end time.Time) (int64, int64, error) {
	var result struct {
		TotalIn  int64
		TotalOut int64
	}
	
	query := r.db.WithContext(ctx).Model(&models.TrafficRecord{}).
		Select("COALESCE(SUM(traffic_in), 0) as total_in, COALESCE(SUM(traffic_out), 0) as total_out").
		Where("tunnel_id = ?", tunnelID)
	
	if !start.IsZero() {
		query = query.Where("recorded_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("recorded_at <= ?", end)
	}
	
	if err := query.Scan(&result).Error; err != nil {
		return 0, 0, err
	}
	
	return result.TotalIn, result.TotalOut, nil
}

// DeleteOldRecords 删除旧记录
func (r *TrafficRecordRepository) DeleteOldRecords(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("recorded_at < ?", before).Delete(&models.TrafficRecord{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// toTrafficRecordEntity 模型转实体
func toTrafficRecordEntity(m *models.TrafficRecord) *traffic.TrafficRecord {
	return &traffic.TrafficRecord{
		ID:         m.ID,
		UserID:     m.UserID,
		TunnelID:   m.TunnelID,
		TrafficIn:  m.TrafficIn,
		TrafficOut: m.TrafficOut,
		RecordedAt: m.RecordedAt,
		CreatedAt:  m.CreatedAt,
	}
}

// toTrafficRecordModel 实体转模型
func toTrafficRecordModel(r *traffic.TrafficRecord) *models.TrafficRecord {
	return &models.TrafficRecord{
		ID:         r.ID,
		UserID:     r.UserID,
		TunnelID:   r.TunnelID,
		TrafficIn:  r.TrafficIn,
		TrafficOut: r.TrafficOut,
		RecordedAt: r.RecordedAt,
		CreatedAt:  r.CreatedAt,
	}
}
