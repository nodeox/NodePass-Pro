package user

import (
	"context"
	"time"
)

// Repository 用户仓储接口
type Repository interface {
	// 基础 CRUD
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id uint) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error
	
	// 批量操作
	FindByIDs(ctx context.Context, ids []uint) ([]*User, error)
	List(ctx context.Context, filter ListFilter) ([]*User, int64, error)
	
	// 业务查询
	FindActiveUsers(ctx context.Context, limit int) ([]*User, error)
	CountByRole(ctx context.Context, role string) (int64, error)
	UpdateLastLogin(ctx context.Context, userID uint, loginTime time.Time) error
}

// ListFilter 列表查询过滤器
type ListFilter struct {
	Page     int
	PageSize int
	Role     string
	Status   string
	Keyword  string // 搜索关键词（用户名或邮箱）
}
