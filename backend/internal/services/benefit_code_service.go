package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"nodepass-panel/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const benefitCodeCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// BenefitCodeService 权益码服务。
type BenefitCodeService struct {
	db         *gorm.DB
	vipService *VIPService
}

// BenefitCodeListFilters 权益码查询过滤条件。
type BenefitCodeListFilters struct {
	Status   string
	VIPLevel *int
	Page     int
	PageSize int
}

// BenefitCodeListResult 权益码分页结果。
type BenefitCodeListResult struct {
	List     []models.BenefitCode `json:"list"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

// RedeemResult 兑换结果。
type RedeemResult struct {
	Code         string     `json:"code"`
	AppliedLevel int        `json:"applied_level"`
	VIPExpiresAt *time.Time `json:"vip_expires_at"`
}

// NewBenefitCodeService 创建权益码服务实例。
func NewBenefitCodeService(db *gorm.DB) *BenefitCodeService {
	return &BenefitCodeService{
		db:         db,
		vipService: NewVIPService(db),
	}
}

// Generate 生成批量权益码（管理员）。
func (s *BenefitCodeService) Generate(
	adminID uint,
	vipLevel int,
	durationDays int,
	count int,
	expiresAt *time.Time,
) ([]models.BenefitCode, error) {
	if _, err := ensureAdminUser(s.db, adminID); err != nil {
		return nil, err
	}
	if count <= 0 {
		return nil, fmt.Errorf("%w: count 必须大于 0", ErrInvalidParams)
	}
	if count > 1000 {
		return nil, fmt.Errorf("%w: 单次最多生成 1000 个权益码", ErrInvalidParams)
	}
	if durationDays <= 0 {
		return nil, fmt.Errorf("%w: duration_days 必须大于 0", ErrInvalidParams)
	}
	if _, err := s.vipService.getLevelByLevel(vipLevel); err != nil {
		return nil, err
	}

	now := time.Now()
	if expiresAt != nil && expiresAt.Before(now) {
		return nil, fmt.Errorf("%w: expires_at 不能早于当前时间", ErrInvalidParams)
	}

	codes := make([]models.BenefitCode, 0, count)
	seen := make(map[string]struct{}, count)
	for len(codes) < count {
		code, err := generateBenefitCode()
		if err != nil {
			return nil, fmt.Errorf("生成权益码失败: %w", err)
		}
		if _, exists := seen[code]; exists {
			continue
		}
		seen[code] = struct{}{}

		record := models.BenefitCode{
			Code:         code,
			VipLevel:     vipLevel,
			DurationDays: durationDays,
			Status:       "unused",
			IsEnabled:    true,
			ExpiresAt:    expiresAt,
		}
		if err = s.db.Create(&record).Error; err != nil {
			if isDuplicateKeyError(err) {
				delete(seen, code)
				continue
			}
			return nil, fmt.Errorf("保存权益码失败: %w", err)
		}

		codes = append(codes, record)
	}

	return codes, nil
}

// List 查询权益码列表（支持过滤与分页）。
func (s *BenefitCodeService) List(filters BenefitCodeListFilters) (*BenefitCodeListResult, error) {
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

	query := s.db.Model(&models.BenefitCode{})
	if status := strings.TrimSpace(filters.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if filters.VIPLevel != nil {
		query = query.Where("vip_level = ?", *filters.VIPLevel)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询权益码总数失败: %w", err)
	}

	list := make([]models.BenefitCode, 0, pageSize)
	if err := query.Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询权益码列表失败: %w", err)
	}

	return &BenefitCodeListResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Redeem 兑换权益码并应用 VIP 权益。
func (s *BenefitCodeService) Redeem(userID uint, code string) (*RedeemResult, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, fmt.Errorf("%w: code 不能为空", ErrInvalidParams)
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("开启事务失败: %w", tx.Error)
	}

	var benefitCode models.BenefitCode
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("code = ?", code).
		First(&benefitCode).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 权益码不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询权益码失败: %w", err)
	}

	if !benefitCode.IsEnabled {
		tx.Rollback()
		return nil, fmt.Errorf("%w: 权益码已禁用", ErrForbidden)
	}
	if !strings.EqualFold(strings.TrimSpace(benefitCode.Status), "unused") {
		tx.Rollback()
		return nil, fmt.Errorf("%w: 权益码已使用", ErrConflict)
	}
	now := time.Now()
	if benefitCode.ExpiresAt != nil && benefitCode.ExpiresAt.Before(now) {
		tx.Rollback()
		return nil, fmt.Errorf("%w: 权益码已过期", ErrForbidden)
	}

	var user models.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&user, userID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 用户不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	appliedLevel := benefitCode.VipLevel
	if user.VipLevel > appliedLevel {
		appliedLevel = user.VipLevel
	}
	levelDetail, err := s.vipService.getLevelByLevel(appliedLevel)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	baseTime := now
	if user.VipExpiresAt != nil && user.VipExpiresAt.After(now) {
		baseTime = *user.VipExpiresAt
	}
	expireAt := baseTime.AddDate(0, 0, benefitCode.DurationDays)

	userUpdates := buildUserVIPUpdates(levelDetail, &expireAt)
	if err = tx.Model(&models.User{}).Where("id = ?", user.ID).Updates(userUpdates).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新用户 VIP 权益失败: %w", err)
	}

	if err = tx.Model(&models.BenefitCode{}).
		Where("id = ?", benefitCode.ID).
		Updates(map[string]interface{}{
			"status":  "used",
			"used_by": user.ID,
			"used_at": now,
		}).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新权益码状态失败: %w", err)
	}

	if err = tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return &RedeemResult{
		Code:         benefitCode.Code,
		AppliedLevel: appliedLevel,
		VIPExpiresAt: &expireAt,
	}, nil
}

// BatchDelete 批量删除权益码（管理员）。
func (s *BenefitCodeService) BatchDelete(adminID uint, ids []uint) (int64, error) {
	if _, err := ensureAdminUser(s.db, adminID); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, fmt.Errorf("%w: ids 不能为空", ErrInvalidParams)
	}

	result := s.db.Where("id IN ?", ids).Delete(&models.BenefitCode{})
	if result.Error != nil {
		return 0, fmt.Errorf("批量删除权益码失败: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func generateBenefitCode() (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	chars := make([]byte, 12)
	for i, b := range raw {
		chars[i] = benefitCodeCharset[int(b)%len(benefitCodeCharset)]
	}
	return fmt.Sprintf("NP-%s-%s-%s", string(chars[0:4]), string(chars[4:8]), string(chars[8:12])), nil
}
