package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// VIPCache VIP 缓存
type VIPCache struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewVIPCache 创建 VIP 缓存
func NewVIPCache(client *redis.Client) *VIPCache {
	return &VIPCache{
		client: client,
		prefix: "vip:",
		ttl:    10 * time.Minute, // VIP 信息缓存 10 分钟
	}
}

// ========== VIP 等级缓存 ==========

// SetLevel 缓存 VIP 等级
func (c *VIPCache) SetLevel(ctx context.Context, level int, data map[string]interface{}) error {
	key := fmt.Sprintf("%slevel:%d", c.prefix, level)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, jsonData, c.ttl).Err()
}

// GetLevel 获取 VIP 等级
func (c *VIPCache) GetLevel(ctx context.Context, level int) (map[string]interface{}, error) {
	key := fmt.Sprintf("%slevel:%d", c.prefix, level)
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // 缓存未命中
	}
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteLevel 删除 VIP 等级缓存
func (c *VIPCache) DeleteLevel(ctx context.Context, level int) error {
	key := fmt.Sprintf("%slevel:%d", c.prefix, level)
	return c.client.Del(ctx, key).Err()
}

// SetLevelByID 通过 ID 缓存 VIP 等级
func (c *VIPCache) SetLevelByID(ctx context.Context, id uint, data map[string]interface{}) error {
	key := fmt.Sprintf("%slevel_id:%d", c.prefix, id)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, jsonData, c.ttl).Err()
}

// GetLevelByID 通过 ID 获取 VIP 等级
func (c *VIPCache) GetLevelByID(ctx context.Context, id uint) (map[string]interface{}, error) {
	key := fmt.Sprintf("%slevel_id:%d", c.prefix, id)
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteLevelByID 通过 ID 删除 VIP 等级缓存
func (c *VIPCache) DeleteLevelByID(ctx context.Context, id uint) error {
	key := fmt.Sprintf("%slevel_id:%d", c.prefix, id)
	return c.client.Del(ctx, key).Err()
}

// SetAllLevels 缓存所有 VIP 等级列表
func (c *VIPCache) SetAllLevels(ctx context.Context, levels []map[string]interface{}) error {
	key := fmt.Sprintf("%sall_levels", c.prefix)
	jsonData, err := json.Marshal(levels)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, jsonData, c.ttl).Err()
}

// GetAllLevels 获取所有 VIP 等级列表
func (c *VIPCache) GetAllLevels(ctx context.Context) ([]map[string]interface{}, error) {
	key := fmt.Sprintf("%sall_levels", c.prefix)
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// InvalidateAllLevels 使所有等级列表缓存失效
func (c *VIPCache) InvalidateAllLevels(ctx context.Context) error {
	key := fmt.Sprintf("%sall_levels", c.prefix)
	return c.client.Del(ctx, key).Err()
}

// ========== 用户 VIP 状态缓存 ==========

// SetUserVIP 缓存用户 VIP 状态
func (c *VIPCache) SetUserVIP(ctx context.Context, userID uint, vipLevel int, expiresAt *time.Time) error {
	key := fmt.Sprintf("%suser:%d", c.prefix, userID)

	data := map[string]interface{}{
		"vip_level": vipLevel,
	}
	if expiresAt != nil {
		data["expires_at"] = expiresAt.Unix()
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, jsonData, c.ttl).Err()
}

// GetUserVIP 获取用户 VIP 状态
func (c *VIPCache) GetUserVIP(ctx context.Context, userID uint) (vipLevel int, expiresAt *time.Time, err error) {
	key := fmt.Sprintf("%suser:%d", c.prefix, userID)
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil, nil // 缓存未命中
	}
	if err != nil {
		return 0, nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return 0, nil, err
	}

	vipLevel = int(result["vip_level"].(float64))

	if expiresAtUnix, ok := result["expires_at"].(float64); ok {
		t := time.Unix(int64(expiresAtUnix), 0)
		expiresAt = &t
	}

	return vipLevel, expiresAt, nil
}

// DeleteUserVIP 删除用户 VIP 状态缓存
func (c *VIPCache) DeleteUserVIP(ctx context.Context, userID uint) error {
	key := fmt.Sprintf("%suser:%d", c.prefix, userID)
	return c.client.Del(ctx, key).Err()
}

// IsUserVIPActive 检查用户 VIP 是否激活（快速检查）
func (c *VIPCache) IsUserVIPActive(ctx context.Context, userID uint) (bool, error) {
	vipLevel, expiresAt, err := c.GetUserVIP(ctx, userID)
	if err != nil {
		return false, err
	}

	if vipLevel == 0 {
		return false, nil // 免费用户
	}

	if expiresAt == nil {
		return true, nil // 永久 VIP
	}

	return time.Now().Before(*expiresAt), nil
}

// ========== VIP 权益缓存 ==========

// SetUserVIPBenefits 缓存用户 VIP 权益
func (c *VIPCache) SetUserVIPBenefits(ctx context.Context, userID uint, benefits map[string]interface{}) error {
	key := fmt.Sprintf("%suser_benefits:%d", c.prefix, userID)
	jsonData, err := json.Marshal(benefits)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, jsonData, c.ttl).Err()
}

// GetUserVIPBenefits 获取用户 VIP 权益
func (c *VIPCache) GetUserVIPBenefits(ctx context.Context, userID uint) (map[string]interface{}, error) {
	key := fmt.Sprintf("%suser_benefits:%d", c.prefix, userID)
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteUserVIPBenefits 删除用户 VIP 权益缓存
func (c *VIPCache) DeleteUserVIPBenefits(ctx context.Context, userID uint) error {
	key := fmt.Sprintf("%suser_benefits:%d", c.prefix, userID)
	return c.client.Del(ctx, key).Err()
}

// ========== VIP 升级记录缓存 ==========

// AddUpgradeRecord 添加升级记录（用于防重放）
func (c *VIPCache) AddUpgradeRecord(ctx context.Context, userID uint, recordID string, ttl time.Duration) error {
	key := fmt.Sprintf("%supgrade_record:%d:%s", c.prefix, userID, recordID)
	return c.client.Set(ctx, key, "1", ttl).Err()
}

// HasUpgradeRecord 检查升级记录是否存在
func (c *VIPCache) HasUpgradeRecord(ctx context.Context, userID uint, recordID string) (bool, error) {
	key := fmt.Sprintf("%supgrade_record:%d:%s", c.prefix, userID, recordID)
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// ========== VIP 统计缓存 ==========

// IncrementVIPUserCount 增加 VIP 用户数量
func (c *VIPCache) IncrementVIPUserCount(ctx context.Context, level int) error {
	key := fmt.Sprintf("%slevel_user_count:%d", c.prefix, level)
	return c.client.Incr(ctx, key).Err()
}

// DecrementVIPUserCount 减少 VIP 用户数量
func (c *VIPCache) DecrementVIPUserCount(ctx context.Context, level int) error {
	key := fmt.Sprintf("%slevel_user_count:%d", c.prefix, level)
	return c.client.Decr(ctx, key).Err()
}

// GetVIPUserCount 获取 VIP 用户数量
func (c *VIPCache) GetVIPUserCount(ctx context.Context, level int) (int64, error) {
	key := fmt.Sprintf("%slevel_user_count:%d", c.prefix, level)
	count, err := c.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

// SetVIPUserCount 设置 VIP 用户数量
func (c *VIPCache) SetVIPUserCount(ctx context.Context, level int, count int64) error {
	key := fmt.Sprintf("%slevel_user_count:%d", c.prefix, level)
	return c.client.Set(ctx, key, count, 0).Err() // 不过期
}

// ========== 批量操作 ==========

// InvalidateUserVIPCache 使用户所有 VIP 相关缓存失效
func (c *VIPCache) InvalidateUserVIPCache(ctx context.Context, userID uint) error {
	keys := []string{
		fmt.Sprintf("%suser:%d", c.prefix, userID),
		fmt.Sprintf("%suser_benefits:%d", c.prefix, userID),
	}

	for _, key := range keys {
		if err := c.client.Del(ctx, key).Err(); err != nil {
			// 继续删除其他缓存
			continue
		}
	}

	return nil
}

// InvalidateAllVIPCache 使所有 VIP 缓存失效
func (c *VIPCache) InvalidateAllVIPCache(ctx context.Context) error {
	pattern := fmt.Sprintf("%s*", c.prefix)
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			continue
		}
	}

	return iter.Err()
}
