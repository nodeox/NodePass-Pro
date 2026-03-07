package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// VIPService VIP 等级管理服务。
type VIPService struct {
	db *gorm.DB
}

// VIPLevelCreateRequest 创建 VIP 等级请求。
type VIPLevelCreateRequest struct {
	Level                   int      `json:"level" binding:"required"`
	Name                    string   `json:"name" binding:"required"`
	Description             *string  `json:"description"`
	TrafficQuota            int64    `json:"traffic_quota" binding:"required"`
	MaxRules                int      `json:"max_rules" binding:"required"`
	MaxBandwidth            int      `json:"max_bandwidth" binding:"required"`
	MaxSelfHostedEntryNodes int      `json:"max_self_hosted_entry_nodes"`
	MaxSelfHostedExitNodes  int      `json:"max_self_hosted_exit_nodes"`
	AccessibleNodeLevel     int      `json:"accessible_node_level"`
	TrafficMultiplier       float64  `json:"traffic_multiplier"`
	CustomFeatures          *string  `json:"custom_features"`
	Price                   *float64 `json:"price"`
	DurationDays            *int     `json:"duration_days"`
}

// VIPLevelUpdateRequest 更新 VIP 等级请求。
type VIPLevelUpdateRequest struct {
	Name                    *string  `json:"name"`
	Description             *string  `json:"description"`
	TrafficQuota            *int64   `json:"traffic_quota"`
	MaxRules                *int     `json:"max_rules"`
	MaxBandwidth            *int     `json:"max_bandwidth"`
	MaxSelfHostedEntryNodes *int     `json:"max_self_hosted_entry_nodes"`
	MaxSelfHostedExitNodes  *int     `json:"max_self_hosted_exit_nodes"`
	AccessibleNodeLevel     *int     `json:"accessible_node_level"`
	TrafficMultiplier       *float64 `json:"traffic_multiplier"`
	CustomFeatures          *string  `json:"custom_features"`
	Price                   *float64 `json:"price"`
	DurationDays            *int     `json:"duration_days"`
}

// MyVIPLevelResult 当前用户 VIP 信息。
type MyVIPLevelResult struct {
	UserID       uint             `json:"user_id"`
	VIPLevel     int              `json:"vip_level"`
	VIPExpiresAt *time.Time       `json:"vip_expires_at"`
	LevelDetail  *models.VIPLevel `json:"level_detail"`
}

// NewVIPService 创建 VIP 服务实例。
func NewVIPService(db *gorm.DB) *VIPService {
	return &VIPService{db: db}
}

// ListLevels 返回所有 VIP 等级及权益详情。
func (s *VIPService) ListLevels() ([]models.VIPLevel, error) {
	levels := make([]models.VIPLevel, 0)
	if err := s.db.Model(&models.VIPLevel{}).
		Order("level ASC").
		Find(&levels).Error; err != nil {
		return nil, fmt.Errorf("查询 VIP 等级失败: %w", err)
	}
	return levels, nil
}

// CreateLevel 创建 VIP 等级（管理员）。
func (s *VIPService) CreateLevel(adminID uint, req *VIPLevelCreateRequest) (*models.VIPLevel, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}
	if _, err := ensureAdminUser(s.db, adminID); err != nil {
		return nil, err
	}
	if err := validateCreateVIPLevelRequest(req); err != nil {
		return nil, err
	}

	level := &models.VIPLevel{
		Level:                   req.Level,
		Name:                    strings.TrimSpace(req.Name),
		Description:             normalizeOptionalString(req.Description),
		TrafficQuota:            req.TrafficQuota,
		MaxRules:                req.MaxRules,
		MaxBandwidth:            req.MaxBandwidth,
		MaxSelfHostedEntryNodes: req.MaxSelfHostedEntryNodes,
		MaxSelfHostedExitNodes:  req.MaxSelfHostedExitNodes,
		AccessibleNodeLevel:     normalizePositiveInt(req.AccessibleNodeLevel, 1),
		TrafficMultiplier:       normalizePositiveFloat(req.TrafficMultiplier, 1.0),
		CustomFeatures:          normalizeOptionalString(req.CustomFeatures),
		Price:                   req.Price,
		DurationDays:            req.DurationDays,
	}
	if err := s.db.Create(level).Error; err != nil {
		if isDuplicateKeyError(err) {
			return nil, fmt.Errorf("%w: VIP 等级已存在", ErrConflict)
		}
		return nil, fmt.Errorf("创建 VIP 等级失败: %w", err)
	}

	return level, nil
}

// UpdateLevel 更新 VIP 等级（管理员）。
func (s *VIPService) UpdateLevel(adminID uint, id uint, req *VIPLevelUpdateRequest) (*models.VIPLevel, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}
	if _, err := ensureAdminUser(s.db, adminID); err != nil {
		return nil, err
	}

	level, err := s.getLevelByID(id)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name 不能为空", ErrInvalidParams)
		}
		updates["name"] = name
	}
	if req.Description != nil {
		updates["description"] = normalizeOptionalString(req.Description)
	}
	if req.TrafficQuota != nil {
		if *req.TrafficQuota < 0 {
			return nil, fmt.Errorf("%w: traffic_quota 不能为负数", ErrInvalidParams)
		}
		updates["traffic_quota"] = *req.TrafficQuota
	}
	if req.MaxRules != nil {
		updates["max_rules"] = *req.MaxRules
	}
	if req.MaxBandwidth != nil {
		updates["max_bandwidth"] = *req.MaxBandwidth
	}
	if req.MaxSelfHostedEntryNodes != nil {
		if *req.MaxSelfHostedEntryNodes < 0 {
			return nil, fmt.Errorf("%w: max_self_hosted_entry_nodes 不能为负数", ErrInvalidParams)
		}
		updates["max_self_hosted_entry_nodes"] = *req.MaxSelfHostedEntryNodes
	}
	if req.MaxSelfHostedExitNodes != nil {
		if *req.MaxSelfHostedExitNodes < 0 {
			return nil, fmt.Errorf("%w: max_self_hosted_exit_nodes 不能为负数", ErrInvalidParams)
		}
		updates["max_self_hosted_exit_nodes"] = *req.MaxSelfHostedExitNodes
	}
	if req.AccessibleNodeLevel != nil {
		if *req.AccessibleNodeLevel <= 0 {
			return nil, fmt.Errorf("%w: accessible_node_level 必须大于 0", ErrInvalidParams)
		}
		updates["accessible_node_level"] = *req.AccessibleNodeLevel
	}
	if req.TrafficMultiplier != nil {
		if *req.TrafficMultiplier <= 0 {
			return nil, fmt.Errorf("%w: traffic_multiplier 必须大于 0", ErrInvalidParams)
		}
		updates["traffic_multiplier"] = *req.TrafficMultiplier
	}
	if req.CustomFeatures != nil {
		updates["custom_features"] = normalizeOptionalString(req.CustomFeatures)
	}
	if req.Price != nil {
		updates["price"] = req.Price
	}
	if req.DurationDays != nil {
		updates["duration_days"] = req.DurationDays
	}

	if len(updates) == 0 {
		return level, nil
	}

	if err = s.db.Model(&models.VIPLevel{}).Where("id = ?", level.ID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新 VIP 等级失败: %w", err)
	}
	return s.getLevelByID(level.ID)
}

// GetMyLevel 返回用户当前 VIP 等级及权益。
func (s *VIPService) GetMyLevel(userID uint) (*MyVIPLevelResult, error) {
	user, err := s.getUserByID(userID)
	if err != nil {
		return nil, err
	}

	level, err := s.getLevelByLevel(user.VipLevel)
	if err != nil {
		return nil, err
	}

	return &MyVIPLevelResult{
		UserID:       user.ID,
		VIPLevel:     user.VipLevel,
		VIPExpiresAt: user.VipExpiresAt,
		LevelDetail:  level,
	}, nil
}

// UpgradeUser 管理员升级用户 VIP。
func (s *VIPService) UpgradeUser(adminID uint, targetUserID uint, level int, durationDays int) (*models.User, error) {
	if _, err := ensureAdminUser(s.db, adminID); err != nil {
		return nil, err
	}
	if durationDays <= 0 {
		return nil, fmt.Errorf("%w: durationDays 必须大于 0", ErrInvalidParams)
	}

	user, err := s.getUserByID(targetUserID)
	if err != nil {
		return nil, err
	}
	levelDetail, err := s.getLevelByLevel(level)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expireAt := now.AddDate(0, 0, durationDays)
	if user.VipExpiresAt != nil && user.VipExpiresAt.After(now) && user.VipLevel == level {
		expireAt = user.VipExpiresAt.AddDate(0, 0, durationDays)
	}

	updates := buildUserVIPUpdates(levelDetail, &expireAt)
	if err = s.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("升级用户 VIP 失败: %w", err)
	}

	return s.getUserByID(user.ID)
}

// CheckExpiration 检查 VIP 到期并降级。
func (s *VIPService) CheckExpiration() (int64, error) {
	freeLevel, err := s.getLevelByLevel(0)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	expiredUsers := make([]models.User, 0)
	if err = s.db.Model(&models.User{}).
		Where("vip_level > ? AND vip_expires_at IS NOT NULL AND vip_expires_at < ?", 0, now).
		Find(&expiredUsers).Error; err != nil {
		return 0, fmt.Errorf("查询到期 VIP 用户失败: %w", err)
	}
	if len(expiredUsers) == 0 {
		return 0, nil
	}

	updates := buildUserVIPUpdates(freeLevel, nil)
	userIDs := make([]uint, 0, len(expiredUsers))
	for _, user := range expiredUsers {
		userIDs = append(userIDs, user.ID)
	}

	if err = s.db.Model(&models.User{}).Where("id IN ?", userIDs).Updates(updates).Error; err != nil {
		return 0, fmt.Errorf("降级到免费等级失败: %w", err)
	}

	return int64(len(userIDs)), nil
}

func validateCreateVIPLevelRequest(req *VIPLevelCreateRequest) error {
	if req.Level < 0 {
		return fmt.Errorf("%w: level 不能为负数", ErrInvalidParams)
	}
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("%w: name 不能为空", ErrInvalidParams)
	}
	if req.TrafficQuota < 0 {
		return fmt.Errorf("%w: traffic_quota 不能为负数", ErrInvalidParams)
	}
	if req.MaxSelfHostedEntryNodes < 0 || req.MaxSelfHostedExitNodes < 0 {
		return fmt.Errorf("%w: 自托管节点配额不能为负数", ErrInvalidParams)
	}
	if req.AccessibleNodeLevel < 0 {
		return fmt.Errorf("%w: accessible_node_level 不能为负数", ErrInvalidParams)
	}
	return nil
}

func buildUserVIPUpdates(level *models.VIPLevel, expireAt *time.Time) map[string]interface{} {
	updates := map[string]interface{}{
		"vip_level":                   level.Level,
		"vip_expires_at":              expireAt,
		"traffic_quota":               level.TrafficQuota,
		"max_rules":                   level.MaxRules,
		"max_bandwidth":               level.MaxBandwidth,
		"max_self_hosted_entry_nodes": level.MaxSelfHostedEntryNodes,
		"max_self_hosted_exit_nodes":  level.MaxSelfHostedExitNodes,
	}
	return updates
}

func (s *VIPService) getLevelByID(id uint) (*models.VIPLevel, error) {
	if id == 0 {
		return nil, fmt.Errorf("%w: VIP 等级 ID 无效", ErrInvalidParams)
	}
	var level models.VIPLevel
	if err := s.db.First(&level, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: VIP 等级不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询 VIP 等级失败: %w", err)
	}
	return &level, nil
}

func (s *VIPService) getLevelByLevel(levelValue int) (*models.VIPLevel, error) {
	var level models.VIPLevel
	if err := s.db.Where("level = ?", levelValue).First(&level).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: VIP 等级不存在(level=%d)", ErrNotFound, levelValue)
		}
		return nil, fmt.Errorf("查询 VIP 等级失败: %w", err)
	}
	return &level, nil
}

func (s *VIPService) getUserByID(userID uint) (*models.User, error) {
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

func normalizePositiveInt(value int, defaultValue int) int {
	if value <= 0 {
		return defaultValue
	}
	return value
}

func normalizePositiveFloat(value float64, defaultValue float64) float64 {
	if value <= 0 {
		return defaultValue
	}
	return value
}
