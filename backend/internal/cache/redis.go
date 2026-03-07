package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"nodepass-panel/backend/internal/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	mu       sync.RWMutex
	client   *redis.Client
	redisCfg *config.RedisConfig
)

// Init 初始化 Redis 缓存连接。
func Init(cfg *config.RedisConfig) error {
	mu.Lock()
	defer mu.Unlock()

	redisCfg = cfg
	if cfg == nil || !cfg.Enabled {
		zap.L().Info("Redis 缓存未启用")
		client = nil
		return nil
	}

	options := &redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	}
	client = redis.NewClient(options)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client = nil
		return fmt.Errorf("连接 Redis 失败: %w", err)
	}

	zap.L().Info("Redis 缓存初始化成功",
		zap.String("addr", cfg.Addr),
		zap.Int("db", cfg.DB),
	)
	return nil
}

// Close 关闭 Redis 连接。
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if client == nil {
		return nil
	}
	if err := client.Close(); err != nil {
		return err
	}
	client = nil
	return nil
}

// Enabled 返回 Redis 缓存是否可用。
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return client != nil && redisCfg != nil && redisCfg.Enabled
}

// DefaultTTL 返回默认缓存有效期。
func DefaultTTL() time.Duration {
	mu.RLock()
	defer mu.RUnlock()
	if redisCfg == nil || redisCfg.DefaultTTL <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(redisCfg.DefaultTTL) * time.Second
}

// BuildKey 按配置前缀拼接缓存键。
func BuildKey(key string) string {
	mu.RLock()
	defer mu.RUnlock()

	base := strings.TrimSpace(key)
	if redisCfg == nil || strings.TrimSpace(redisCfg.KeyPrefix) == "" {
		return base
	}
	return strings.TrimSuffix(redisCfg.KeyPrefix, ":") + ":" + base
}

// GetJSON 按键读取 JSON 缓存。
func GetJSON(ctx context.Context, key string, out any) (bool, error) {
	mu.RLock()
	c := client
	mu.RUnlock()

	if c == nil {
		return false, nil
	}

	raw, err := c.Get(ctx, BuildKey(key)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return false, err
	}
	return true, nil
}

// SetJSON 写入 JSON 缓存。
func SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	mu.RLock()
	c := client
	mu.RUnlock()

	if c == nil {
		return nil
	}

	if ttl <= 0 {
		ttl = DefaultTTL()
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.Set(ctx, BuildKey(key), raw, ttl).Err()
}

// Delete 删除指定缓存键。
func Delete(ctx context.Context, keys ...string) error {
	mu.RLock()
	c := client
	mu.RUnlock()

	if c == nil || len(keys) == 0 {
		return nil
	}

	targets := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		targets = append(targets, BuildKey(key))
	}
	if len(targets) == 0 {
		return nil
	}
	return c.Del(ctx, targets...).Err()
}
