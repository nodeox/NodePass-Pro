package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/domain/role"

	"github.com/redis/go-redis/v9"
)

const (
	roleCachePrefix           = "role:"
	roleCodeCachePrefix       = "role:code:"
	rolePermissionCachePrefix = "role:permission:"
	roleCacheTTL              = 30 * time.Minute
	rolePermissionCacheTTL    = 30 * time.Minute
)

// RoleCache 角色缓存
type RoleCache struct {
	client *redis.Client
}

// NewRoleCache 创建角色缓存
func NewRoleCache(client *redis.Client) *RoleCache {
	return &RoleCache{client: client}
}

// SetRole 设置角色缓存
func (c *RoleCache) SetRole(ctx context.Context, r *role.Role) error {
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("序列化角色失败: %w", err)
	}

	// 按 ID 缓存
	key := fmt.Sprintf("%s%d", roleCachePrefix, r.ID)
	if err := c.client.Set(ctx, key, data, roleCacheTTL).Err(); err != nil {
		return err
	}

	// 按 Code 缓存
	codeKey := fmt.Sprintf("%s%s", roleCodeCachePrefix, r.Code)
	return c.client.Set(ctx, codeKey, data, roleCacheTTL).Err()
}

// GetRole 获取角色缓存（按 ID）
func (c *RoleCache) GetRole(ctx context.Context, id uint) (*role.Role, error) {
	key := fmt.Sprintf("%s%d", roleCachePrefix, id)
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("获取角色缓存失败: %w", err)
	}

	var r role.Role
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		return nil, fmt.Errorf("反序列化角色失败: %w", err)
	}

	return &r, nil
}

// GetRoleByCode 获取角色缓存（按 Code）
func (c *RoleCache) GetRoleByCode(ctx context.Context, code string) (*role.Role, error) {
	key := fmt.Sprintf("%s%s", roleCodeCachePrefix, code)
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("获取角色缓存失败: %w", err)
	}

	var r role.Role
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		return nil, fmt.Errorf("反序列化角色失败: %w", err)
	}

	return &r, nil
}

// DeleteRole 删除角色缓存
func (c *RoleCache) DeleteRole(ctx context.Context, id uint, code string) error {
	keys := []string{
		fmt.Sprintf("%s%d", roleCachePrefix, id),
		fmt.Sprintf("%s%s", roleCodeCachePrefix, code),
	}
	return c.client.Del(ctx, keys...).Err()
}

// SetPermissionCheck 设置权限检查缓存
func (c *RoleCache) SetPermissionCheck(ctx context.Context, roleCode, permissionCode string, hasPermission bool) error {
	key := fmt.Sprintf("%s%s:%s", rolePermissionCachePrefix, roleCode, permissionCode)
	value := "0"
	if hasPermission {
		value = "1"
	}
	return c.client.Set(ctx, key, value, rolePermissionCacheTTL).Err()
}

// GetPermissionCheck 获取权限检查缓存
func (c *RoleCache) GetPermissionCheck(ctx context.Context, roleCode, permissionCode string) (*bool, error) {
	key := fmt.Sprintf("%s%s:%s", rolePermissionCachePrefix, roleCode, permissionCode)
	value, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("获取权限检查缓存失败: %w", err)
	}

	hasPermission := value == "1"
	return &hasPermission, nil
}

// InvalidateRolePermissions 清除角色的所有权限缓存
func (c *RoleCache) InvalidateRolePermissions(ctx context.Context, roleCode string) error {
	pattern := fmt.Sprintf("%s%s:*", rolePermissionCachePrefix, roleCode)
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()

	keys := make([]string, 0)
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("扫描权限缓存失败: %w", err)
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}

	return nil
}
