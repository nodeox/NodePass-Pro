package systemconfig

import "context"

// Repository 系统配置仓储接口
type Repository interface {
	// FindByKey 根据键查找配置
	FindByKey(ctx context.Context, key string) (*SystemConfig, error)

	// FindAll 查找所有配置
	FindAll(ctx context.Context) ([]*SystemConfig, error)

	// Upsert 创建或更新配置
	Upsert(ctx context.Context, config *SystemConfig) error

	// Delete 删除配置
	Delete(ctx context.Context, key string) error

	// GetAllAsMap 获取所有配置作为 map
	GetAllAsMap(ctx context.Context) (map[string]string, error)
}
