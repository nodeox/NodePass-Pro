package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupVIPCacheTest(t *testing.T) (*VIPCache, context.Context) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试数据库
	})

	ctx := context.Background()

	// 清空测试数据库
	err := client.FlushDB(ctx).Err()
	assert.NoError(t, err)

	cache := NewVIPCache(client)
	return cache, ctx
}

// ========== VIP 等级缓存测试 ==========

func TestVIPCache_SetAndGetLevel(t *testing.T) {
	cache, ctx := setupVIPCacheTest(t)

	level := 1
	data := map[string]interface{}{
		"id":            1,
		"level":         1,
		"name":          "VIP 1",
		"traffic_quota": 100000000,
	}

	// 设置等级
	err := cache.SetLevel(ctx, level, data)
	assert.NoError(t, err)

	// 获取等级
	result, err := cache.GetLevel(ctx, level)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "VIP 1", result["name"])
}

func TestVIPCache_GetLevel_NotFound(t *testing.T) {
	cache, ctx := setupVIPCacheTest(t)

	result, err := cache.GetLevel(ctx, 999)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestVIPCache_DeleteLevel(t *testing.T) {
	cache, ctx := setupVIPCacheTest(t)

	level := 2
	data := map[string]interface{}{
		"id":    2,
		"level": 2,
		"name":  "VIP 2",
	}

	// 设置等级
	err := cache.SetLevel(ctx, level, data)
	assert.NoError(t, err)

	// 删除等级
	err = cache.DeleteLevel(ctx, level)
	assert.NoError(t, err)

	// 验证已删除
	result, err := cache.GetLevel(ctx, level)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestVIPCache_LevelByID(t *testing.T) {
	cache, ctx := setupVIPCacheTest(t)

	id := uint(10)
	data := map[string]interface{}{
		"id":    10,
		"level": 3,
		"name":  "VIP 3",
	}

	// 设置等级
	err := cache.SetLevelByID(ctx, id, data)
	assert.NoError(t, err)

	// 获取等级
	result, err := cache.GetLevelByID(ctx, id)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(3), result["level"])

	// 删除等级
	err = cache.DeleteLevelByID(ctx, id)
	assert.NoError(t, err)

	// 验证已删除
	result, err = cache.GetLevelByID(ctx, id)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestVIPCache_AllLevels(t *testing.T) {
	cache, ctx := setupVIPCacheTest(t)

	levels := []map[string]interface{}{
		{"id": 1, "level": 0, "name": "Free"},
		{"id": 2, "level": 1, "name": "VIP 1"},
		{"id": 3, "level": 2, "name": "VIP 2"},
	}

	// 设置所有等级
	err := cache.SetAllLevels(ctx, levels)
	assert.NoError(t, err)

	// 获取所有等级
	result, err := cache.GetAllLevels(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 3)

	// 使缓存失效
	err = cache.InvalidateAllLevels(ctx)
	assert.NoError(t, err)

	// 验证已删除
	result, err = cache.GetAllLevels(ctx)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// ========== 用户 VIP 状态缓存测试 ==========

func TestVIPCache_UserVIP(t *testing.T) {
	cache, ctx := setupVIPCacheTest(t)

	userID := uint(100)
	vipLevel := 2
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	// 设置用户 VIP
	err := cache.SetUserVIP(ctx, userID, vipLevel, &expiresAt)
	assert.NoError(t, err)

	// 获取用户 VIP
	gotLevel, gotExpiresAt, err := cache.GetUserVIP(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, vipLevel, gotLevel)
	assert.NotNil(t, gotExpiresAt)
	assert.WithinDuration(t, expiresAt, *gotExpiresAt, time.Second)

	// 删除用户 VIP
	err = cache.DeleteUserVIP(ctx, userID)
	assert.NoError(t, err)

	// 验证已删除
	gotLevel, gotExpiresAt, err = cache.GetUserVIP(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 0, gotLevel)
	assert.Nil(t, gotExpiresAt)
}

func TestVIPCache_UserVIP_Permanent(t *testing.T) {
	cache, ctx := setupVIPCacheTest(t)

	userID := uint(200)
	vipLevel := 3

	// 设置永久 VIP（无过期时间）
	err := cache.SetUserVIP(ctx, userID, vipLevel, nil)
	assert.NoError(t, err)

	// 获取用户 VIP
	gotLevel, gotExpiresAt, err := cache.GetUserVIP(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, vipLevel, gotLevel)
	assert.Nil(t, gotExpiresAt)
}

func TestVIPCache_IsUserVIPActive(t *testing.T) {
	cache, ctx := setupVIPCacheTest(t)

	// 测试激活的 VIP
	userID1 := uint(300)
	expiresAt1 := time.Now().Add(30 * 24 * time.Hour)
	err := cache.SetUserVIP(ctx, userID1, 1, &expiresAt1)
	assert.NoError(t, err)

	active, err := cache.IsUserVIPActive(ctx, userID1)
	assert.NoError(t, err)
	assert.True(t, active)

	// 测试过期的 VIP
	userID2 := uint(400)
	expiresAt2 := time.Now().Add(-1 * time.Hour)
	err = cache.SetUserVIP(ctx, userID2, 1, &expiresAt2)
	assert.NoError(t, err)

	active, err = cache.IsUserVIPActive(ctx, userID2)
	assert.NoError(t, err)
	assert.False(t, active)

	// 测试永久 VIP
	userID3 := uint(500)
	err = cache.SetUserVIP(ctx, userID3, 2, nil)
	assert.NoError(t, err)

	active, err = cache.IsUserVIPActive(ctx, userID3)
	assert.NoError(t, err)
	assert.True(t, active)

	// 测试免费用户
	userID4 := uint(600)
	err = cache.SetUserVIP(ctx, userID4, 0, nil)
	assert.NoError(t, err)

	active, err = cache.IsUserVIPActive(ctx, userID4)
	assert.NoError(t, err)
	assert.False(t, active)
}

// ========== VIP 权益缓存测试 ==========

func TestVIPCache_UserVIPBenefits(t *testing.T) {
	cache, ctx := setupVIPCacheTest(t)

	userID := uint(700)
	benefits := map[string]interface{}{
		"traffic_quota": 100000000,
		"max_rules":     10,
		"max_bandwidth": 1000,
	}

	// 设置权益
	err := cache.SetUserVIPBenefits(ctx, userID, benefits)
	assert.NoError(t, err)

	// 获取权益
	result, err := cache.GetUserVIPBenefits(ctx, userID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(100000000), result["traffic_quota"])

	// 删除权益
	err = cache.DeleteUserVIPBenefits(ctx, userID)
	assert.NoError(t, err)

	// 验证已删除
	result, err = cache.GetUserVIPBenefits(ctx, userID)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// ========== VIP 升级记录测试 ==========

func TestVIPCache_UpgradeRecord(t *testing.T) {
	cache, ctx := setupVIPCacheTest(t)

	userID := uint(800)
	recordID := "upgrade_123"
	ttl := 1 * time.Hour

	// 添加升级记录
	err := cache.AddUpgradeRecord(ctx, userID, recordID, ttl)
	assert.NoError(t, err)

	// 检查记录存在
	exists, err := cache.HasUpgradeRecord(ctx, userID, recordID)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 检查不存在的记录
	exists, err = cache.HasUpgradeRecord(ctx, userID, "non_existent")
	assert.NoError(t, err)
	assert.False(t, exists)
}

// ========== VIP 统计缓存测试 ==========

func TestVIPCache_VIPUserCount(t *testing.T) {
	cache, ctx := setupVIPCacheTest(t)

	level := 1

	// 设置用户数量
	err := cache.SetVIPUserCount(ctx, level, 100)
	assert.NoError(t, err)

	// 获取用户数量
	count, err := cache.GetVIPUserCount(ctx, level)
	assert.NoError(t, err)
	assert.Equal(t, int64(100), count)

	// 增加用户数量
	err = cache.IncrementVIPUserCount(ctx, level)
	assert.NoError(t, err)

	count, err = cache.GetVIPUserCount(ctx, level)
	assert.NoError(t, err)
	assert.Equal(t, int64(101), count)

	// 减少用户数量
	err = cache.DecrementVIPUserCount(ctx, level)
	assert.NoError(t, err)

	count, err = cache.GetVIPUserCount(ctx, level)
	assert.NoError(t, err)
	assert.Equal(t, int64(100), count)
}

// ========== 批量操作测试 ==========

func TestVIPCache_InvalidateUserVIPCache(t *testing.T) {
	cache, ctx := setupVIPCacheTest(t)

	userID := uint(900)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	// 设置多个缓存
	err := cache.SetUserVIP(ctx, userID, 1, &expiresAt)
	assert.NoError(t, err)

	benefits := map[string]interface{}{"traffic_quota": 100000000}
	err = cache.SetUserVIPBenefits(ctx, userID, benefits)
	assert.NoError(t, err)

	// 使所有用户 VIP 缓存失效
	err = cache.InvalidateUserVIPCache(ctx, userID)
	assert.NoError(t, err)

	// 验证已删除
	level, _, err := cache.GetUserVIP(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 0, level)

	result, err := cache.GetUserVIPBenefits(ctx, userID)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestVIPCache_InvalidateAllVIPCache(t *testing.T) {
	cache, ctx := setupVIPCacheTest(t)

	// 设置多个缓存
	err := cache.SetLevel(ctx, 1, map[string]interface{}{"name": "VIP 1"})
	assert.NoError(t, err)

	err = cache.SetUserVIP(ctx, 100, 1, nil)
	assert.NoError(t, err)

	// 使所有 VIP 缓存失效
	err = cache.InvalidateAllVIPCache(ctx)
	assert.NoError(t, err)

	// 验证已删除
	result, err := cache.GetLevel(ctx, 1)
	assert.NoError(t, err)
	assert.Nil(t, result)

	level, _, err := cache.GetUserVIP(ctx, 100)
	assert.NoError(t, err)
	assert.Equal(t, 0, level)
}
