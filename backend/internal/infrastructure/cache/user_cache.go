package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	
	"github.com/redis/go-redis/v9"
)

// UserCache 用户缓存
type UserCache struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewUserCache 创建用户缓存
func NewUserCache(client *redis.Client) *UserCache {
	return &UserCache{
		client: client,
		prefix: "user:",
		ttl:    5 * time.Minute,
	}
}

// Get 获取用户缓存
func (c *UserCache) Get(ctx context.Context, userID uint) (map[string]interface{}, error) {
	key := fmt.Sprintf("%s%d", c.prefix, userID)
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // 缓存未命中
	}
	if err != nil {
		return nil, err
	}
	
	var user map[string]interface{}
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, err
	}
	return user, nil
}

// Set 设置用户缓存
func (c *UserCache) Set(ctx context.Context, userID uint, user map[string]interface{}) error {
	key := fmt.Sprintf("%s%d", c.prefix, userID)
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, c.ttl).Err()
}

// Delete 删除用户缓存
func (c *UserCache) Delete(ctx context.Context, userID uint) error {
	key := fmt.Sprintf("%s%d", c.prefix, userID)
	return c.client.Del(ctx, key).Err()
}

// GetByEmail 通过邮箱获取用户 ID（二级索引）
func (c *UserCache) GetByEmail(ctx context.Context, email string) (uint, error) {
	key := fmt.Sprintf("%semail:%s", c.prefix, email)
	userID, err := c.client.Get(ctx, key).Uint64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return uint(userID), nil
}

// SetEmailIndex 设置邮箱索引
func (c *UserCache) SetEmailIndex(ctx context.Context, email string, userID uint) error {
	key := fmt.Sprintf("%semail:%s", c.prefix, email)
	return c.client.Set(ctx, key, userID, c.ttl).Err()
}

// IncrementTraffic 增加流量使用（原子操作）
func (c *UserCache) IncrementTraffic(ctx context.Context, userID uint, amount int64) (int64, error) {
	key := fmt.Sprintf("%s%d:traffic", c.prefix, userID)
	return c.client.IncrBy(ctx, key, amount).Result()
}

// GetTraffic 获取流量使用
func (c *UserCache) GetTraffic(ctx context.Context, userID uint) (int64, error) {
	key := fmt.Sprintf("%s%d:traffic", c.prefix, userID)
	return c.client.Get(ctx, key).Int64()
}

// ResetTraffic 重置流量
func (c *UserCache) ResetTraffic(ctx context.Context, userID uint) error {
	key := fmt.Sprintf("%s%d:traffic", c.prefix, userID)
	return c.client.Del(ctx, key).Err()
}
