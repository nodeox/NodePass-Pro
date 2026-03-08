package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisNonceStore 使用 Redis 做全局 Nonce 去重，适用于多实例部署。
type RedisNonceStore struct {
	client *redis.Client
	prefix string
}

// NewRedisNonceStore 创建 Redis Nonce 存储。
func NewRedisNonceStore(client *redis.Client, prefix string) *RedisNonceStore {
	return &RedisNonceStore{
		client: client,
		prefix: prefix,
	}
}

// AddIfAbsent 原子添加 Nonce（SET NX + TTL）。
func (s *RedisNonceStore) AddIfAbsent(nonce string, ttl time.Duration) (bool, error) {
	if s == nil || s.client == nil {
		return false, fmt.Errorf("redis nonce store 未初始化")
	}
	key := strings.TrimSpace(s.prefix) + strings.TrimSpace(nonce)
	return s.client.SetNX(context.Background(), key, "1", ttl).Result()
}
