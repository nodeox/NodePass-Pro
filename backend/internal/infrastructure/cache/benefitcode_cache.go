package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/domain/benefitcode"

	"github.com/redis/go-redis/v9"
)

const (
	benefitCodeCachePrefix      = "benefitcode:"
	benefitCodeListCachePrefix  = "benefitcode:list:"
	benefitCodeUsedCachePrefix  = "benefitcode:used:" // 防重放攻击
	benefitCodeCacheTTL         = 30 * time.Minute
	benefitCodeListCacheTTL     = 10 * time.Minute
	benefitCodeUsedCacheTTL     = 24 * time.Hour // 已使用的权益码缓存 24 小时
)

// BenefitCodeCache 权益码缓存
type BenefitCodeCache struct {
	client *redis.Client
}

// NewBenefitCodeCache 创建权益码缓存
func NewBenefitCodeCache(client *redis.Client) *BenefitCodeCache {
	return &BenefitCodeCache{
		client: client,
	}
}

// SetCode 设置权益码缓存
func (c *BenefitCodeCache) SetCode(ctx context.Context, code *benefitcode.BenefitCode) error {
	key := fmt.Sprintf("%s%s", benefitCodeCachePrefix, code.Code)
	data, err := json.Marshal(code)
	if err != nil {
		return fmt.Errorf("序列化权益码失败: %w", err)
	}
	return c.client.Set(ctx, key, data, benefitCodeCacheTTL).Err()
}

// GetCode 获取权益码缓存
func (c *BenefitCodeCache) GetCode(ctx context.Context, code string) (*benefitcode.BenefitCode, error) {
	key := fmt.Sprintf("%s%s", benefitCodeCachePrefix, code)
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, fmt.Errorf("获取权益码缓存失败: %w", err)
	}

	var benefitCode benefitcode.BenefitCode
	if err := json.Unmarshal([]byte(data), &benefitCode); err != nil {
		return nil, fmt.Errorf("反序列化权益码失败: %w", err)
	}

	return &benefitCode, nil
}

// DeleteCode 删除权益码缓存
func (c *BenefitCodeCache) DeleteCode(ctx context.Context, code string) error {
	key := fmt.Sprintf("%s%s", benefitCodeCachePrefix, code)
	return c.client.Del(ctx, key).Err()
}

// SetCodeList 设置权益码列表缓存
func (c *BenefitCodeCache) SetCodeList(ctx context.Context, cacheKey string, codes []*benefitcode.BenefitCode) error {
	key := fmt.Sprintf("%s%s", benefitCodeListCachePrefix, cacheKey)
	data, err := json.Marshal(codes)
	if err != nil {
		return fmt.Errorf("序列化权益码列表失败: %w", err)
	}
	return c.client.Set(ctx, key, data, benefitCodeListCacheTTL).Err()
}

// GetCodeList 获取权益码列表缓存
func (c *BenefitCodeCache) GetCodeList(ctx context.Context, cacheKey string) ([]*benefitcode.BenefitCode, error) {
	key := fmt.Sprintf("%s%s", benefitCodeListCachePrefix, cacheKey)
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, fmt.Errorf("获取权益码列表缓存失败: %w", err)
	}

	var codes []*benefitcode.BenefitCode
	if err := json.Unmarshal([]byte(data), &codes); err != nil {
		return nil, fmt.Errorf("反序列化权益码列表失败: %w", err)
	}

	return codes, nil
}

// DeleteCodeList 删除权益码列表缓存
func (c *BenefitCodeCache) DeleteCodeList(ctx context.Context, cacheKey string) error {
	key := fmt.Sprintf("%s%s", benefitCodeListCachePrefix, cacheKey)
	return c.client.Del(ctx, key).Err()
}

// InvalidateAllLists 清除所有列表缓存
func (c *BenefitCodeCache) InvalidateAllLists(ctx context.Context) error {
	pattern := fmt.Sprintf("%s*", benefitCodeListCachePrefix)
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()

	keys := make([]string, 0)
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("扫描缓存键失败: %w", err)
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}

	return nil
}

// MarkCodeAsUsed 标记权益码已使用（防重放攻击）
func (c *BenefitCodeCache) MarkCodeAsUsed(ctx context.Context, code string, userID uint) error {
	key := fmt.Sprintf("%s%s", benefitCodeUsedCachePrefix, code)
	return c.client.Set(ctx, key, userID, benefitCodeUsedCacheTTL).Err()
}

// IsCodeUsed 检查权益码是否已使用（防重放攻击）
func (c *BenefitCodeCache) IsCodeUsed(ctx context.Context, code string) (bool, error) {
	key := fmt.Sprintf("%s%s", benefitCodeUsedCachePrefix, code)
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("检查权益码使用状态失败: %w", err)
	}
	return exists > 0, nil
}

// GetUsedByUserID 获取使用该权益码的用户 ID
func (c *BenefitCodeCache) GetUsedByUserID(ctx context.Context, code string) (uint, error) {
	key := fmt.Sprintf("%s%s", benefitCodeUsedCachePrefix, code)
	userID, err := c.client.Get(ctx, key).Uint64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, fmt.Errorf("获取使用用户 ID 失败: %w", err)
	}
	return uint(userID), nil
}
