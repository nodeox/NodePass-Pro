package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"

	"nodepass-pro/backend/internal/domain/benefitcode"
)

// BenefitCodeCacheTestSuite 测试套件
type BenefitCodeCacheTestSuite struct {
	suite.Suite
	cache  *BenefitCodeCache
	client *redis.Client
	ctx    context.Context
}

func (s *BenefitCodeCacheTestSuite) SetupTest() {
	// 使用 Redis 测试实例
	s.client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试数据库
	})

	// 测试连接
	err := s.client.Ping(context.Background()).Err()
	if err != nil {
		s.T().Skip("Redis not available, skipping cache tests")
		return
	}

	s.cache = NewBenefitCodeCache(s.client)
	s.ctx = context.Background()

	// 清空测试数据
	s.client.FlushDB(s.ctx)
}

func (s *BenefitCodeCacheTestSuite) TearDownTest() {
	if s.client != nil {
		s.client.FlushDB(s.ctx)
		s.client.Close()
	}
}

func (s *BenefitCodeCacheTestSuite) TestSetAndGetCode() {
	code := &benefitcode.BenefitCode{
		ID:           1,
		Code:         "NP-TEST-CODE-0001",
		VIPLevel:     1,
		DurationDays: 30,
		Status:       benefitcode.BenefitCodeStatusUnused,
		IsEnabled:    true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 设置缓存
	err := s.cache.SetCode(s.ctx, code)
	s.NoError(err)

	// 获取缓存
	cached, err := s.cache.GetCode(s.ctx, "NP-TEST-CODE-0001")
	s.NoError(err)
	s.NotNil(cached)
	s.Equal(code.Code, cached.Code)
	s.Equal(code.VIPLevel, cached.VIPLevel)
}

func (s *BenefitCodeCacheTestSuite) TestGetCodeNotFound() {
	cached, err := s.cache.GetCode(s.ctx, "NOT-EXIST")
	s.NoError(err)
	s.Nil(cached)
}

func (s *BenefitCodeCacheTestSuite) TestDeleteCode() {
	code := &benefitcode.BenefitCode{
		ID:           1,
		Code:         "NP-DELETE-TEST",
		VIPLevel:     1,
		DurationDays: 30,
		Status:       benefitcode.BenefitCodeStatusUnused,
		IsEnabled:    true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 设置缓存
	s.cache.SetCode(s.ctx, code)

	// 删除缓存
	err := s.cache.DeleteCode(s.ctx, "NP-DELETE-TEST")
	s.NoError(err)

	// 验证已删除
	cached, _ := s.cache.GetCode(s.ctx, "NP-DELETE-TEST")
	s.Nil(cached)
}

func (s *BenefitCodeCacheTestSuite) TestSetAndGetCodeList() {
	codes := []*benefitcode.BenefitCode{
		{
			ID:           1,
			Code:         "NP-LIST-0001",
			VIPLevel:     1,
			DurationDays: 30,
			Status:       benefitcode.BenefitCodeStatusUnused,
			IsEnabled:    true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:           2,
			Code:         "NP-LIST-0002",
			VIPLevel:     1,
			DurationDays: 30,
			Status:       benefitcode.BenefitCodeStatusUnused,
			IsEnabled:    true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	// 设置列表缓存
	err := s.cache.SetCodeList(s.ctx, "test-list", codes)
	s.NoError(err)

	// 获取列表缓存
	cached, err := s.cache.GetCodeList(s.ctx, "test-list")
	s.NoError(err)
	s.NotNil(cached)
	s.Len(cached, 2)
}

func (s *BenefitCodeCacheTestSuite) TestGetCodeListNotFound() {
	cached, err := s.cache.GetCodeList(s.ctx, "not-exist")
	s.NoError(err)
	s.Nil(cached)
}

func (s *BenefitCodeCacheTestSuite) TestDeleteCodeList() {
	codes := []*benefitcode.BenefitCode{
		{ID: 1, Code: "NP-LIST-0001", VIPLevel: 1, DurationDays: 30,
			Status: benefitcode.BenefitCodeStatusUnused, IsEnabled: true,
			CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	// 设置列表缓存
	s.cache.SetCodeList(s.ctx, "test-list", codes)

	// 删除列表缓存
	err := s.cache.DeleteCodeList(s.ctx, "test-list")
	s.NoError(err)

	// 验证已删除
	cached, _ := s.cache.GetCodeList(s.ctx, "test-list")
	s.Nil(cached)
}

func (s *BenefitCodeCacheTestSuite) TestInvalidateAllLists() {
	codes := []*benefitcode.BenefitCode{
		{ID: 1, Code: "NP-LIST-0001", VIPLevel: 1, DurationDays: 30,
			Status: benefitcode.BenefitCodeStatusUnused, IsEnabled: true,
			CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	// 设置多个列表缓存
	s.cache.SetCodeList(s.ctx, "list-1", codes)
	s.cache.SetCodeList(s.ctx, "list-2", codes)

	// 清除所有列表缓存
	err := s.cache.InvalidateAllLists(s.ctx)
	s.NoError(err)

	// 验证都已删除
	cached1, _ := s.cache.GetCodeList(s.ctx, "list-1")
	s.Nil(cached1)
	cached2, _ := s.cache.GetCodeList(s.ctx, "list-2")
	s.Nil(cached2)
}

func (s *BenefitCodeCacheTestSuite) TestMarkCodeAsUsed() {
	err := s.cache.MarkCodeAsUsed(s.ctx, "NP-TEST-CODE", 100)
	s.NoError(err)

	// 验证已标记
	used, err := s.cache.IsCodeUsed(s.ctx, "NP-TEST-CODE")
	s.NoError(err)
	s.True(used)

	// 获取使用用户 ID
	userID, err := s.cache.GetUsedByUserID(s.ctx, "NP-TEST-CODE")
	s.NoError(err)
	s.Equal(uint(100), userID)
}

func (s *BenefitCodeCacheTestSuite) TestIsCodeUsedNotFound() {
	used, err := s.cache.IsCodeUsed(s.ctx, "NOT-EXIST")
	s.NoError(err)
	s.False(used)
}

func (s *BenefitCodeCacheTestSuite) TestGetUsedByUserIDNotFound() {
	userID, err := s.cache.GetUsedByUserID(s.ctx, "NOT-EXIST")
	s.NoError(err)
	s.Equal(uint(0), userID)
}

func TestBenefitCodeCacheTestSuite(t *testing.T) {
	suite.Run(t, new(BenefitCodeCacheTestSuite))
}
