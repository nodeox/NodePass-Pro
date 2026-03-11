package benefitcode

import "context"

// Repository 权益码仓储接口
type Repository interface {
	// Create 创建权益码
	Create(ctx context.Context, code *BenefitCode) error

	// BatchCreate 批量创建权益码
	BatchCreate(ctx context.Context, codes []*BenefitCode) error

	// FindByID 根据 ID 查找
	FindByID(ctx context.Context, id uint) (*BenefitCode, error)

	// FindByCode 根据 Code 查找
	FindByCode(ctx context.Context, code string) (*BenefitCode, error)

	// Update 更新权益码
	Update(ctx context.Context, code *BenefitCode) error

	// Delete 删除权益码
	Delete(ctx context.Context, id uint) error

	// BatchDelete 批量删除权益码
	BatchDelete(ctx context.Context, ids []uint) (int64, error)

	// List 列表查询
	List(ctx context.Context, filter ListFilter) ([]*BenefitCode, int64, error)

	// CountByStatus 按状态统计
	CountByStatus(ctx context.Context, status BenefitCodeStatus) (int64, error)

	// FindExpiredCodes 查找过期的权益码
	FindExpiredCodes(ctx context.Context, limit int) ([]*BenefitCode, error)
}

// ListFilter 列表过滤器
type ListFilter struct {
	Status   BenefitCodeStatus
	VIPLevel *int
	UsedBy   *uint
	Page     int
	PageSize int
}
