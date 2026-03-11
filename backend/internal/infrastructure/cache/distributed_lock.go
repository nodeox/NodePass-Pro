package cache

import (
	"context"
	"errors"
	"time"
	
	"github.com/redis/go-redis/v9"
)

var ErrLockFailed = errors.New("获取锁失败")

// DistributedLock 分布式锁
type DistributedLock struct {
	client *redis.Client
	key    string
	value  string
	ttl    time.Duration
}

// NewDistributedLock 创建分布式锁
func NewDistributedLock(client *redis.Client, key string, ttl time.Duration) *DistributedLock {
	return &DistributedLock{
		client: client,
		key:    "lock:" + key,
		value:  generateLockValue(),
		ttl:    ttl,
	}
}

// Lock 获取锁
func (l *DistributedLock) Lock(ctx context.Context) error {
	success, err := l.client.SetNX(ctx, l.key, l.value, l.ttl).Result()
	if err != nil {
		return err
	}
	if !success {
		return ErrLockFailed
	}
	return nil
}

// TryLock 尝试获取锁（带重试）
func (l *DistributedLock) TryLock(ctx context.Context, retries int, retryDelay time.Duration) error {
	for i := 0; i < retries; i++ {
		if err := l.Lock(ctx); err == nil {
			return nil
		}
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay):
			continue
		}
	}
	return ErrLockFailed
}

// Unlock 释放锁（使用 Lua 脚本保证原子性）
func (l *DistributedLock) Unlock(ctx context.Context) error {
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	
	result, err := l.client.Eval(ctx, script, []string{l.key}, l.value).Result()
	if err != nil {
		return err
	}
	
	if result.(int64) == 0 {
		return errors.New("锁已被其他进程持有")
	}
	
	return nil
}

// Extend 延长锁的过期时间
func (l *DistributedLock) Extend(ctx context.Context, ttl time.Duration) error {
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("expire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`
	
	result, err := l.client.Eval(ctx, script, []string{l.key}, l.value, int(ttl.Seconds())).Result()
	if err != nil {
		return err
	}
	
	if result.(int64) == 0 {
		return errors.New("锁已被其他进程持有")
	}
	
	return nil
}

// generateLockValue 生成锁的唯一值
func generateLockValue() string {
	return time.Now().Format("20060102150405.000000")
}

// WithLock 使用锁执行函数
func WithLock(ctx context.Context, client *redis.Client, key string, ttl time.Duration, fn func() error) error {
	lock := NewDistributedLock(client, key, ttl)
	
	if err := lock.Lock(ctx); err != nil {
		return err
	}
	defer lock.Unlock(ctx)
	
	return fn()
}
