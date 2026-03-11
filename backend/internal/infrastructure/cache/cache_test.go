package cache_test

import (
	"context"
	"testing"
	"time"

	"nodepass-pro/backend/internal/infrastructure/cache"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// setupTestRedis 设置测试 Redis 客户端
func setupTestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试数据库
	})

	ctx := context.Background()
	// 清理测试数据
	client.FlushDB(ctx)

	return client
}

// TestNodeCache_SetOnline 测试设置节点在线
func TestNodeCache_SetOnline(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	nodeCache := cache.NewNodeCache(client)
	ctx := context.Background()

	// 设置节点在线
	err := nodeCache.SetOnline(ctx, "test-node-001", 3*time.Minute)
	assert.NoError(t, err)

	// 验证节点在线
	isOnline, err := nodeCache.IsOnline(ctx, "test-node-001")
	assert.NoError(t, err)
	assert.True(t, isOnline)

	// 验证 TTL
	ttl := client.TTL(ctx, "node:online:test-node-001")
	assert.Greater(t, ttl.Val().Seconds(), float64(0))
	assert.LessOrEqual(t, ttl.Val().Seconds(), float64(180))
}

// TestNodeCache_IsOnline_Expired 测试节点过期
func TestNodeCache_IsOnline_Expired(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	nodeCache := cache.NewNodeCache(client)
	ctx := context.Background()

	// 设置节点在线，1 秒过期
	err := nodeCache.SetOnline(ctx, "test-node-002", 1*time.Second)
	assert.NoError(t, err)

	// 等待过期
	time.Sleep(2 * time.Second)

	// 验证节点离线
	isOnline, err := nodeCache.IsOnline(ctx, "test-node-002")
	assert.NoError(t, err)
	assert.False(t, isOnline)
}

// TestNodeCache_GetAllOnlineNodes 测试获取所有在线节点
func TestNodeCache_GetAllOnlineNodes(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	nodeCache := cache.NewNodeCache(client)
	ctx := context.Background()

	// 设置多个节点在线
	nodeCache.SetOnline(ctx, "node-001", 3*time.Minute)
	nodeCache.SetOnline(ctx, "node-002", 3*time.Minute)
	nodeCache.SetOnline(ctx, "node-003", 3*time.Minute)

	// 获取所有在线节点
	nodes, err := nodeCache.GetAllOnlineNodes(ctx)
	assert.NoError(t, err)
	assert.Len(t, nodes, 3)
	assert.Contains(t, nodes, "node-001")
	assert.Contains(t, nodes, "node-002")
	assert.Contains(t, nodes, "node-003")
}

// TestNodeCache_SetNodeMetrics 测试设置节点指标
func TestNodeCache_SetNodeMetrics(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	nodeCache := cache.NewNodeCache(client)
	ctx := context.Background()

	metrics := map[string]float64{
		"cpu":    45.5,
		"memory": 60.2,
		"disk":   75.8,
	}

	// 设置指标
	err := nodeCache.SetNodeMetrics(ctx, "test-node-001", metrics)
	assert.NoError(t, err)

	// 验证指标
	result, err := client.HGetAll(ctx, "node:metrics:test-node-001").Result()
	assert.NoError(t, err)
	assert.Equal(t, "45.5", result["cpu"])
	assert.Equal(t, "60.2", result["memory"])
	assert.Equal(t, "75.8", result["disk"])
}

// TestHeartbeatBuffer_PushAndPop 测试心跳缓冲区
func TestHeartbeatBuffer_PushAndPop(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	buffer := cache.NewHeartbeatBuffer(client)
	ctx := context.Background()

	// 推送心跳数据
	data1 := &cache.HeartbeatData{
		NodeID:      "node-001",
		Status:      "online",
		CPUUsage:    50.0,
		MemoryUsage: 60.0,
		TrafficIn:   1000,
		TrafficOut:  2000,
		Timestamp:   time.Now(),
	}

	data2 := &cache.HeartbeatData{
		NodeID:      "node-001",
		Status:      "online",
		CPUUsage:    55.0,
		MemoryUsage: 65.0,
		TrafficIn:   1500,
		TrafficOut:  2500,
		Timestamp:   time.Now(),
	}

	err := buffer.Push(ctx, data1)
	assert.NoError(t, err)

	err = buffer.Push(ctx, data2)
	assert.NoError(t, err)

	// 弹出数据
	results, err := buffer.PopBatch(ctx, "node-001", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	// 验证数据顺序（FIFO）
	assert.Equal(t, 50.0, results[0].CPUUsage)
	assert.Equal(t, 55.0, results[1].CPUUsage)

	// 再次弹出应该为空
	results, err = buffer.PopBatch(ctx, "node-001", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 0)
}

// TestHeartbeatBuffer_PopBatch_Limit 测试批量弹出限制
func TestHeartbeatBuffer_PopBatch_Limit(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	buffer := cache.NewHeartbeatBuffer(client)
	ctx := context.Background()

	// 推送 5 条数据
	for i := 0; i < 5; i++ {
		data := &cache.HeartbeatData{
			NodeID:    "node-001",
			CPUUsage:  float64(i * 10),
			Timestamp: time.Now(),
		}
		buffer.Push(ctx, data)
	}

	// 只弹出 3 条
	results, err := buffer.PopBatch(ctx, "node-001", 3)
	assert.NoError(t, err)
	assert.Len(t, results, 3)

	// 剩余 2 条
	results, err = buffer.PopBatch(ctx, "node-001", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

// TestUserCache_SetAndGet 测试用户缓存
func TestUserCache_SetAndGet(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	userCache := cache.NewUserCache(client)
	ctx := context.Background()

	userData := map[string]interface{}{
		"id":            uint(1),
		"username":      "testuser",
		"email":         "test@example.com",
		"role":          "user",
		"vip_level":     1,
		"traffic_quota": int64(10737418240), // 10GB
		"traffic_used":  int64(1073741824),  // 1GB
	}

	// 设置缓存
	err := userCache.Set(ctx, 1, userData)
	assert.NoError(t, err)

	// 获取缓存
	result, err := userCache.Get(ctx, 1)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "testuser", result["username"])
	assert.Equal(t, "test@example.com", result["email"])
}

// TestUserCache_IncrementTraffic 测试增加流量
func TestUserCache_IncrementTraffic(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	userCache := cache.NewUserCache(client)
	ctx := context.Background()

	// 第一次增加流量（从 0 开始）
	newTotal, err := userCache.IncrementTraffic(ctx, 1, 500)
	assert.NoError(t, err)
	assert.Equal(t, int64(500), newTotal)

	// 再次增加
	newTotal, err = userCache.IncrementTraffic(ctx, 1, 300)
	assert.NoError(t, err)
	assert.Equal(t, int64(800), newTotal)
}

// TestUserCache_Delete 测试删除缓存
func TestUserCache_Delete(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	userCache := cache.NewUserCache(client)
	ctx := context.Background()

	userData := map[string]interface{}{
		"id":       uint(1),
		"username": "testuser",
	}

	// 设置缓存
	userCache.Set(ctx, 1, userData)

	// 验证存在
	result, err := userCache.Get(ctx, 1)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// 删除缓存
	err = userCache.Delete(ctx, 1)
	assert.NoError(t, err)

	// 验证已删除 - Get 应该返回 nil 或空结果
	result, _ = userCache.Get(ctx, 1)
	assert.Nil(t, result)
}

// TestTrafficCounter_IncrementUserTraffic 测试用户流量计数
func TestTrafficCounter_IncrementUserTraffic(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	counter := cache.NewTrafficCounter(client)
	ctx := context.Background()

	// 增加流量
	err := counter.IncrementUserTraffic(ctx, 1, 1000, 2000)
	assert.NoError(t, err)

	// 再次增加
	err = counter.IncrementUserTraffic(ctx, 1, 500, 1000)
	assert.NoError(t, err)

	// 验证总量
	inKey := "traffic:user:1:in"
	outKey := "traffic:user:1:out"

	inResult, err := client.Get(ctx, inKey).Result()
	assert.NoError(t, err)
	assert.Equal(t, "1500", inResult)

	outResult, err := client.Get(ctx, outKey).Result()
	assert.NoError(t, err)
	assert.Equal(t, "3000", outResult)
}

// TestTrafficCounter_IncrementTunnelTraffic 测试隧道流量计数
func TestTrafficCounter_IncrementTunnelTraffic(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	counter := cache.NewTrafficCounter(client)
	ctx := context.Background()

	// 增加流量
	err := counter.IncrementTunnelTraffic(ctx, 100, 5000, 10000)
	assert.NoError(t, err)

	// 验证
	inKey := "traffic:tunnel:100:in"
	outKey := "traffic:tunnel:100:out"

	inResult, err := client.Get(ctx, inKey).Result()
	assert.NoError(t, err)
	assert.Equal(t, "5000", inResult)

	outResult, err := client.Get(ctx, outKey).Result()
	assert.NoError(t, err)
	assert.Equal(t, "10000", outResult)
}

// TestDistributedLock_LockAndUnlock 测试分布式锁
func TestDistributedLock_LockAndUnlock(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	lock := cache.NewDistributedLock(client, "test-lock", 10*time.Second)
	ctx := context.Background()

	// 获取锁
	err := lock.Lock(ctx)
	assert.NoError(t, err)

	// 再次获取应该失败
	err = lock.TryLock(ctx, 1, 10*time.Millisecond)
	assert.Error(t, err)

	// 释放锁
	err = lock.Unlock(ctx)
	assert.NoError(t, err)

	// 再次获取应该成功
	err = lock.Lock(ctx)
	assert.NoError(t, err)

	lock.Unlock(ctx)
}

// TestDistributedLock_AutoExpire 测试锁自动过期
func TestDistributedLock_AutoExpire(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	lock := cache.NewDistributedLock(client, "test-lock-expire", 1*time.Second)
	ctx := context.Background()

	// 获取锁
	err := lock.Lock(ctx)
	assert.NoError(t, err)

	// 等待过期
	time.Sleep(2 * time.Second)

	// 其他进程可以获取锁
	lock2 := cache.NewDistributedLock(client, "test-lock-expire", 10*time.Second)
	err = lock2.Lock(ctx)
	assert.NoError(t, err)

	lock2.Unlock(ctx)
}
