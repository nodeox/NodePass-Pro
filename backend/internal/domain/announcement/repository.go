package announcement

import "context"

// Repository 公告仓储接口
type Repository interface {
	// Create 创建公告
	Create(ctx context.Context, announcement *Announcement) error

	// FindByID 根据 ID 查找公告
	FindByID(ctx context.Context, id uint) (*Announcement, error)

	// Update 更新公告
	Update(ctx context.Context, announcement *Announcement) error

	// Delete 删除公告
	Delete(ctx context.Context, id uint) error

	// ListAll 列出所有公告
	ListAll(ctx context.Context) ([]*Announcement, error)

	// ListEnabled 列出启用的公告
	ListEnabled(ctx context.Context) ([]*Announcement, error)
}
