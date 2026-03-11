package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// TrafficService 流量管理服务。
//
// Deprecated: 此服务已被重构为 DDD 架构。
// 新代码请使用以下模块：
//   - Commands: internal/application/traffic/commands (RecordTrafficHandler, FlushTrafficHandler)
//   - Queries: internal/application/traffic/queries (GetUserTrafficHandler, GetTunnelTrafficHandler)
//   - Repository: internal/infrastructure/persistence/postgres/traffic_repository.go
//   - Cache: internal/infrastructure/cache/traffic_counter.go
// 通过依赖注入容器获取: container.RecordTrafficHandler, container.GetUserTrafficHandler 等
// 此服务将在所有旧代码迁移完成后删除。

// QuotaResult 用户配额信息。
type QuotaResult struct {
	TrafficQuota int64   `json:"traffic_quota"`
	TrafficUsed  int64   `json:"traffic_used"`
	UsagePercent float64 `json:"usage_percent"`
	IsOverLimit  bool    `json:"is_over_limit"`
}

// UsageResult 用户时段使用汇总。
type UsageResult struct {
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	TotalTrafficIn  int64     `json:"total_traffic_in"`
	TotalTrafficOut int64     `json:"total_traffic_out"`
	TotalCalculated int64     `json:"total_calculated_traffic"`
	RecordCount     int64     `json:"record_count"`
}

// TrafficRecordFilters 流量记录筛选条件。
type TrafficRecordFilters struct {
	RuleID    *uint
	NodeID    *uint
	StartTime *time.Time
	EndTime   *time.Time
	Page      int
	PageSize  int
}

// TrafficRecordListResult 流量记录分页结果。
type TrafficRecordListResult struct {
	List     []models.TrafficRecord `json:"list"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

// TrafficService 流量统计与配额管理服务。
type TrafficService struct {
	db *gorm.DB
}

// NewTrafficService 创建流量服务实例。
func NewTrafficService(db *gorm.DB) *TrafficService {
	return &TrafficService{db: db}
}

// GetQuota 获取用户流量配额信息。
func (s *TrafficService) GetQuota(userID uint) (*QuotaResult, error) {
	user, err := s.getUserByID(userID)
	if err != nil {
		return nil, err
	}

	usagePercent := 0.0
	if user.TrafficQuota > 0 {
		usagePercent = float64(user.TrafficUsed) / float64(user.TrafficQuota) * 100
	}
	if usagePercent < 0 {
		usagePercent = 0
	}

	isOverLimit := user.TrafficUsed >= user.TrafficQuota

	return &QuotaResult{
		TrafficQuota: user.TrafficQuota,
		TrafficUsed:  user.TrafficUsed,
		UsagePercent: usagePercent,
		IsOverLimit:  isOverLimit,
	}, nil
}

// GetUsage 获取指定时间范围内流量汇总。
func (s *TrafficService) GetUsage(userID uint, startTime time.Time, endTime time.Time) (*UsageResult, error) {
	if userID == 0 {
		return nil, fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}
	if !endTime.After(startTime) {
		return nil, fmt.Errorf("%w: 时间范围无效", ErrInvalidParams)
	}

	type aggregateResult struct {
		TotalTrafficIn  int64
		TotalTrafficOut int64
		TotalCalculated int64
		RecordCount     int64
	}

	var aggregate aggregateResult
	if err := s.db.Model(&models.TrafficRecord{}).
		Select(
			"COALESCE(SUM(traffic_in), 0) AS total_traffic_in, "+
				"COALESCE(SUM(traffic_out), 0) AS total_traffic_out, "+
				"COALESCE(SUM(calculated_traffic), 0) AS total_calculated, "+
				"COUNT(1) AS record_count",
		).
		Where("user_id = ? AND hour >= ? AND hour <= ?", userID, startTime.UTC(), endTime.UTC()).
		Scan(&aggregate).Error; err != nil {
		return nil, fmt.Errorf("查询流量汇总失败: %w", err)
	}

	return &UsageResult{
		StartTime:       startTime.UTC(),
		EndTime:         endTime.UTC(),
		TotalTrafficIn:  aggregate.TotalTrafficIn,
		TotalTrafficOut: aggregate.TotalTrafficOut,
		TotalCalculated: aggregate.TotalCalculated,
		RecordCount:     aggregate.RecordCount,
	}, nil
}

// GetRecords 获取流量明细记录。
func (s *TrafficService) GetRecords(userID uint, filters TrafficRecordFilters) (*TrafficRecordListResult, error) {
	if userID == 0 {
		return nil, fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}

	page := filters.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filters.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	query := s.db.Model(&models.TrafficRecord{}).
		Preload("Rule").
		Preload("Node").
		Where("user_id = ?", userID)

	if filters.RuleID != nil {
		query = query.Where("rule_id = ?", *filters.RuleID)
	}
	if filters.NodeID != nil {
		query = query.Where("node_id = ?", *filters.NodeID)
	}
	if filters.StartTime != nil {
		query = query.Where("hour >= ?", filters.StartTime.UTC())
	}
	if filters.EndTime != nil {
		query = query.Where("hour <= ?", filters.EndTime.UTC())
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询流量记录总数失败: %w", err)
	}

	list := make([]models.TrafficRecord, 0, pageSize)
	if err := query.Order("hour DESC,id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询流量记录失败: %w", err)
	}

	return &TrafficRecordListResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// CheckQuota 校验用户是否超出配额，超限则暂停规则并标记用户状态。
func (s *TrafficService) CheckQuota(userID uint) (bool, error) {
	user, err := s.getUserByID(userID)
	if err != nil {
		return false, err
	}
	if user.TrafficUsed < user.TrafficQuota {
		return false, nil
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return false, fmt.Errorf("开启事务失败: %w", tx.Error)
	}

	runningRules := make([]models.Rule, 0)
	if err = tx.Model(&models.Rule{}).
		Where("user_id = ? AND status = ?", userID, "running").
		Find(&runningRules).Error; err != nil {
		tx.Rollback()
		return false, fmt.Errorf("查询运行中规则失败: %w", err)
	}

	if len(runningRules) > 0 {
		if err = tx.Model(&models.Rule{}).
			Where("user_id = ? AND status = ?", userID, "running").
			Updates(map[string]interface{}{
				"status":      "paused",
				"sync_status": "pending",
			}).Error; err != nil {
			tx.Rollback()
			return false, fmt.Errorf("暂停超限规则失败: %w", err)
		}

		entryNodeIDs := make(map[uint]struct{})
		for _, rule := range runningRules {
			entryNodeIDs[rule.EntryNodeID] = struct{}{}
		}
		for entryNodeID := range entryNodeIDs {
			if err = tx.Model(&models.Node{}).
				Where("id = ?", entryNodeID).
				UpdateColumn("config_version", gorm.Expr("config_version + ?", 1)).Error; err != nil {
				tx.Rollback()
				return false, fmt.Errorf("更新节点配置版本失败: %w", err)
			}
		}
	}

	if err = tx.Model(&models.User{}).
		Where("id = ?", userID).
		Update("status", "overlimit").Error; err != nil {
		tx.Rollback()
		return false, fmt.Errorf("更新用户状态失败: %w", err)
	}

	if err = tx.Commit().Error; err != nil {
		return false, fmt.Errorf("提交事务失败: %w", err)
	}
	return true, nil
}

// ResetQuota 管理员重置用户流量配额使用量。
func (s *TrafficService) ResetQuota(adminUserID uint, targetUserID uint) error {
	if adminUserID == 0 || targetUserID == 0 {
		return fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}

	admin, err := s.getUserByID(adminUserID)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(admin.Role), "admin") {
		return fmt.Errorf("%w: 仅管理员可重置配额", ErrForbidden)
	}

	if _, err = s.getUserByID(targetUserID); err != nil {
		return err
	}

	if err = s.db.Model(&models.User{}).
		Where("id = ?", targetUserID).
		Updates(map[string]interface{}{
			"traffic_used": 0,
			"status":       "normal",
		}).Error; err != nil {
		return fmt.Errorf("重置用户配额失败: %w", err)
	}

	return nil
}

// UpdateQuota 管理员更新用户流量配额。
func (s *TrafficService) UpdateQuota(adminUserID uint, targetUserID uint, trafficQuota int64) error {
	if adminUserID == 0 || targetUserID == 0 {
		return fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}
	if trafficQuota < 0 {
		return fmt.Errorf("%w: 流量配额不能为负数", ErrInvalidParams)
	}

	admin, err := s.getUserByID(adminUserID)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(admin.Role), "admin") {
		return fmt.Errorf("%w: 仅管理员可更新配额", ErrForbidden)
	}

	if _, err = s.getUserByID(targetUserID); err != nil {
		return err
	}

	if err = s.db.Model(&models.User{}).
		Where("id = ?", targetUserID).
		Update("traffic_quota", trafficQuota).Error; err != nil {
		return fmt.Errorf("更新用户流量配额失败: %w", err)
	}

	return nil
}

// MonthlyReset 每月重置配额并恢复超限用户规则。
func (s *TrafficService) MonthlyReset() error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %w", tx.Error)
	}

	overLimitUsers := make([]models.User, 0)
	if err := tx.Model(&models.User{}).
		Where("status = ?", "overlimit").
		Find(&overLimitUsers).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("查询超限用户失败: %w", err)
	}

	if err := tx.Model(&models.User{}).
		Where("1 = 1").
		Update("traffic_used", 0).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("重置流量使用量失败: %w", err)
	}

	if err := tx.Model(&models.User{}).
		Where("status = ?", "overlimit").
		Update("status", "normal").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("恢复用户状态失败: %w", err)
	}

	if len(overLimitUsers) > 0 {
		userIDs := make([]uint, 0, len(overLimitUsers))
		for _, user := range overLimitUsers {
			userIDs = append(userIDs, user.ID)
		}

		pausedRules := make([]models.Rule, 0)
		if err := tx.Model(&models.Rule{}).
			Where("user_id IN ? AND status = ?", userIDs, "paused").
			Find(&pausedRules).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("查询暂停规则失败: %w", err)
		}

		if len(pausedRules) > 0 {
			if err := tx.Model(&models.Rule{}).
				Where("user_id IN ? AND status = ?", userIDs, "paused").
				Updates(map[string]interface{}{
					"status":      "running",
					"sync_status": "pending",
				}).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("恢复暂停规则失败: %w", err)
			}

			entryNodeIDs := make(map[uint]struct{})
			for _, rule := range pausedRules {
				entryNodeIDs[rule.EntryNodeID] = struct{}{}
			}
			for nodeID := range entryNodeIDs {
				if err := tx.Model(&models.Node{}).
					Where("id = ?", nodeID).
					UpdateColumn("config_version", gorm.Expr("config_version + ?", 1)).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("更新节点配置版本失败: %w", err)
				}
			}
		}
	}

	return tx.Commit().Error
}

func (s *TrafficService) upsertTrafficRecord(
	tx *gorm.DB,
	userID uint,
	ruleID uint,
	nodeID uint,
	hour time.Time,
	trafficIn int64,
	trafficOut int64,
	vipMultiplier float64,
	nodeMultiplier float64,
	finalMultiplier float64,
	calculatedTraffic int64,
) error {
	var existing models.TrafficRecord
	err := tx.Model(&models.TrafficRecord{}).
		Where("user_id = ? AND rule_id = ? AND node_id = ? AND hour = ?", userID, ruleID, nodeID, hour).
		First(&existing).Error
	if err == nil {
		return tx.Model(&models.TrafficRecord{}).
			Where("id = ?", existing.ID).
			Updates(map[string]interface{}{
				"traffic_in":         gorm.Expr("traffic_in + ?", trafficIn),
				"traffic_out":        gorm.Expr("traffic_out + ?", trafficOut),
				"calculated_traffic": gorm.Expr("calculated_traffic + ?", calculatedTraffic),
				"vip_multiplier":     vipMultiplier,
				"node_multiplier":    nodeMultiplier,
				"final_multiplier":   finalMultiplier,
			}).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询流量聚合记录失败: %w", err)
	}

	ruleIDCopy := ruleID
	nodeIDCopy := nodeID
	record := &models.TrafficRecord{
		UserID:            userID,
		RuleID:            &ruleIDCopy,
		NodeID:            &nodeIDCopy,
		TrafficIn:         trafficIn,
		TrafficOut:        trafficOut,
		VipMultiplier:     vipMultiplier,
		NodeMultiplier:    nodeMultiplier,
		FinalMultiplier:   finalMultiplier,
		CalculatedTraffic: calculatedTraffic,
		Hour:              hour,
	}
	if err = tx.Create(record).Error; err != nil {
		return fmt.Errorf("创建流量记录失败: %w", err)
	}
	return nil
}

func (s *TrafficService) getVIPMultiplier(tx *gorm.DB, vipLevel int) (float64, error) {
	if vipLevel <= 0 {
		return 1.0, nil
	}

	var vip models.VIPLevel
	if err := tx.Where("level = ?", vipLevel).First(&vip).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 1.0, nil
		}
		return 0, fmt.Errorf("查询 VIP 倍率失败: %w", err)
	}
	if vip.TrafficMultiplier <= 0 {
		return 1.0, nil
	}
	return vip.TrafficMultiplier, nil
}

func (s *TrafficService) getUserByID(userID uint) (*models.User, error) {
	if userID == 0 {
		return nil, fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 用户不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}
