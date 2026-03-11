package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	
	"github.com/redis/go-redis/v9"
)

// TunnelCache 隧道缓存
type TunnelCache struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewTunnelCache 创建隧道缓存
func NewTunnelCache(client *redis.Client) *TunnelCache {
	return &TunnelCache{
		client: client,
		prefix: "tunnel:",
		ttl:    5 * time.Minute,
	}
}

// Get 获取隧道缓存
func (c *TunnelCache) Get(ctx context.Context, tunnelID uint) (map[string]interface{}, error) {
	key := fmt.Sprintf("%s%d", c.prefix, tunnelID)
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	var tunnel map[string]interface{}
	if err := json.Unmarshal([]byte(data), &tunnel); err != nil {
		return nil, err
	}
	return tunnel, nil
}

// Set 设置隧道缓存
func (c *TunnelCache) Set(ctx context.Context, tunnelID uint, tunnel map[string]interface{}) error {
	key := fmt.Sprintf("%s%d", c.prefix, tunnelID)
	data, err := json.Marshal(tunnel)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, c.ttl).Err()
}

// Delete 删除隧道缓存
func (c *TunnelCache) Delete(ctx context.Context, tunnelID uint) error {
	key := fmt.Sprintf("%s%d", c.prefix, tunnelID)
	return c.client.Del(ctx, key).Err()
}

// SetRunning 设置隧道运行状态
func (c *TunnelCache) SetRunning(ctx context.Context, tunnelID uint, running bool) error {
	key := fmt.Sprintf("%srunning:%d", c.prefix, tunnelID)
	if running {
		return c.client.Set(ctx, key, "1", 10*time.Minute).Err()
	}
	return c.client.Del(ctx, key).Err()
}

// IsRunning 检查隧道是否运行中
func (c *TunnelCache) IsRunning(ctx context.Context, tunnelID uint) (bool, error) {
	key := fmt.Sprintf("%srunning:%d", c.prefix, tunnelID)
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// IncrementTraffic 增加隧道流量（原子操作）
func (c *TunnelCache) IncrementTraffic(ctx context.Context, tunnelID uint, inBytes, outBytes int64) error {
	pipe := c.client.Pipeline()
	
	inKey := fmt.Sprintf("%s%d:traffic:in", c.prefix, tunnelID)
	outKey := fmt.Sprintf("%s%d:traffic:out", c.prefix, tunnelID)
	
	pipe.IncrBy(ctx, inKey, inBytes)
	pipe.IncrBy(ctx, outKey, outBytes)
	
	pipe.Expire(ctx, inKey, 24*time.Hour)
	pipe.Expire(ctx, outKey, 24*time.Hour)
	
	_, err := pipe.Exec(ctx)
	return err
}

// GetTraffic 获取隧道流量
func (c *TunnelCache) GetTraffic(ctx context.Context, tunnelID uint) (inBytes, outBytes int64, err error) {
	pipe := c.client.Pipeline()
	
	inKey := fmt.Sprintf("%s%d:traffic:in", c.prefix, tunnelID)
	outKey := fmt.Sprintf("%s%d:traffic:out", c.prefix, tunnelID)
	
	inCmd := pipe.Get(ctx, inKey)
	outCmd := pipe.Get(ctx, outKey)
	
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return 0, 0, err
	}
	
	inBytes, _ = inCmd.Int64()
	outBytes, _ = outCmd.Int64()
	
	return inBytes, outBytes, nil
}

// ResetTraffic 重置隧道流量
func (c *TunnelCache) ResetTraffic(ctx context.Context, tunnelID uint) error {
	inKey := fmt.Sprintf("%s%d:traffic:in", c.prefix, tunnelID)
	outKey := fmt.Sprintf("%s%d:traffic:out", c.prefix, tunnelID)
	
	pipe := c.client.Pipeline()
	pipe.Del(ctx, inKey)
	pipe.Del(ctx, outKey)
	
	_, err := pipe.Exec(ctx)
	return err
}

// CheckPortConflict 检查端口冲突
func (c *TunnelCache) CheckPortConflict(ctx context.Context, port int) (bool, error) {
	key := fmt.Sprintf("%sport:%d", c.prefix, port)
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// ReservePort 预留端口
func (c *TunnelCache) ReservePort(ctx context.Context, port int, tunnelID uint) error {
	key := fmt.Sprintf("%sport:%d", c.prefix, port)
	return c.client.Set(ctx, key, tunnelID, 24*time.Hour).Err()
}

// ReleasePort 释放端口
func (c *TunnelCache) ReleasePort(ctx context.Context, port int) error {
	key := fmt.Sprintf("%sport:%d", c.prefix, port)
	return c.client.Del(ctx, key).Err()
}
