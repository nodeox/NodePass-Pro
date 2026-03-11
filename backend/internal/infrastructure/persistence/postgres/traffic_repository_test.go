package postgres

import (
	"context"
	"testing"
	"time"

	"nodepass-pro/backend/internal/domain/traffic"
	"nodepass-pro/backend/internal/models"

	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TrafficRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo traffic.RecordRepository
	ctx  context.Context
}

func (s *TrafficRepositoryTestSuite) SetupSuite() {
	// 使用内存 SQLite 数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	s.Require().NoError(err)

	// 自动迁移
	err = db.AutoMigrate(&models.TrafficRecord{})
	s.Require().NoError(err)

	s.db = db
	s.repo = NewTrafficRecordRepository(db)
	s.ctx = context.Background()
}

func (s *TrafficRepositoryTestSuite) TearDownTest() {
	// 每个测试后清理数据
	s.db.Exec("DELETE FROM traffic_records")
}

func TestTrafficRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(TrafficRepositoryTestSuite))
}

// ========== Create 测试 ==========

func (s *TrafficRepositoryTestSuite) TestCreate() {
	record := &traffic.TrafficRecord{
		UserID:     1,
		TunnelID:   10,
		TrafficIn:  1000,
		TrafficOut: 2000,
		RecordedAt: time.Now(),
	}

	err := s.repo.Create(s.ctx, record)
	s.NoError(err)
	s.NotZero(record.ID)
	s.NotZero(record.CreatedAt)
}

// ========== BatchCreate 测试 ==========

func (s *TrafficRepositoryTestSuite) TestBatchCreate() {
	records := []*traffic.TrafficRecord{
		{UserID: 1, TunnelID: 10, TrafficIn: 1000, TrafficOut: 2000, RecordedAt: time.Now()},
		{UserID: 1, TunnelID: 11, TrafficIn: 1500, TrafficOut: 2500, RecordedAt: time.Now()},
		{UserID: 2, TunnelID: 12, TrafficIn: 2000, TrafficOut: 3000, RecordedAt: time.Now()},
	}

	err := s.repo.BatchCreate(s.ctx, records)
	s.NoError(err)

	// 注意：GORM 的 CreateInBatches 不会回填 ID
	// 我们通过查询来验证记录已创建
	filter := traffic.RecordListFilter{
		Page:     1,
		PageSize: 10,
	}
	found, total, err := s.repo.List(s.ctx, filter)
	s.NoError(err)
	s.Len(found, 3)
	s.Equal(int64(3), total)
}

func (s *TrafficRepositoryTestSuite) TestBatchCreate_Empty() {
	err := s.repo.BatchCreate(s.ctx, []*traffic.TrafficRecord{})
	s.NoError(err)
}

// ========== FindByID 测试 ==========

func (s *TrafficRepositoryTestSuite) TestFindByID() {
	// 创建测试数据
	record := &traffic.TrafficRecord{
		UserID:     1,
		TunnelID:   10,
		TrafficIn:  1000,
		TrafficOut: 2000,
		RecordedAt: time.Now(),
	}
	err := s.repo.Create(s.ctx, record)
	s.Require().NoError(err)

	// 查找
	found, err := s.repo.FindByID(s.ctx, record.ID)
	s.NoError(err)
	s.NotNil(found)
	s.Equal(record.ID, found.ID)
	s.Equal(record.UserID, found.UserID)
	s.Equal(record.TrafficIn, found.TrafficIn)
}

// ========== FindByUserID 测试 ==========

func (s *TrafficRepositoryTestSuite) TestFindByUserID() {
	userID := uint(100)
	now := time.Now()

	// 创建测试数据
	for i := 0; i < 3; i++ {
		record := &traffic.TrafficRecord{
			UserID:     userID,
			TunnelID:   uint(10 + i),
			TrafficIn:  int64(1000 * (i + 1)),
			TrafficOut: int64(2000 * (i + 1)),
			RecordedAt: now.Add(time.Duration(i) * time.Hour),
		}
		err := s.repo.Create(s.ctx, record)
		s.Require().NoError(err)
	}

	// 查找
	records, err := s.repo.FindByUserID(s.ctx, userID, time.Time{}, time.Time{})
	s.NoError(err)
	s.Len(records, 3)
}

func (s *TrafficRepositoryTestSuite) TestFindByUserID_WithTimeRange() {
	userID := uint(200)
	now := time.Now()

	// 创建测试数据
	for i := 0; i < 5; i++ {
		record := &traffic.TrafficRecord{
			UserID:     userID,
			TunnelID:   uint(10 + i),
			TrafficIn:  1000,
			TrafficOut: 2000,
			RecordedAt: now.Add(time.Duration(i) * time.Hour),
		}
		err := s.repo.Create(s.ctx, record)
		s.Require().NoError(err)
	}

	// 查找指定时间范围
	start := now.Add(1 * time.Hour)
	end := now.Add(3 * time.Hour)
	records, err := s.repo.FindByUserID(s.ctx, userID, start, end)
	s.NoError(err)
	s.Len(records, 3) // 应该返回 1, 2, 3 小时的记录
}

// ========== FindByTunnelID 测试 ==========

func (s *TrafficRepositoryTestSuite) TestFindByTunnelID() {
	tunnelID := uint(50)
	now := time.Now()

	// 创建测试数据
	for i := 0; i < 3; i++ {
		record := &traffic.TrafficRecord{
			UserID:     uint(100 + i),
			TunnelID:   tunnelID,
			TrafficIn:  1000,
			TrafficOut: 2000,
			RecordedAt: now.Add(time.Duration(i) * time.Hour),
		}
		err := s.repo.Create(s.ctx, record)
		s.Require().NoError(err)
	}

	// 查找
	records, err := s.repo.FindByTunnelID(s.ctx, tunnelID, time.Time{}, time.Time{})
	s.NoError(err)
	s.Len(records, 3)
}

// ========== List 测试 ==========

func (s *TrafficRepositoryTestSuite) TestList() {
	// 创建测试数据
	for i := 0; i < 5; i++ {
		record := &traffic.TrafficRecord{
			UserID:     1,
			TunnelID:   uint(10 + i),
			TrafficIn:  1000,
			TrafficOut: 2000,
			RecordedAt: time.Now(),
		}
		err := s.repo.Create(s.ctx, record)
		s.Require().NoError(err)
	}

	// 列表查询
	filter := traffic.RecordListFilter{
		Page:     1,
		PageSize: 10,
	}
	records, total, err := s.repo.List(s.ctx, filter)
	s.NoError(err)
	s.Len(records, 5)
	s.Equal(int64(5), total)
}

func (s *TrafficRepositoryTestSuite) TestList_WithFilters() {
	now := time.Now()

	// 创建测试数据
	record1 := &traffic.TrafficRecord{
		UserID:     1,
		TunnelID:   10,
		TrafficIn:  1000,
		TrafficOut: 2000,
		RecordedAt: now,
	}
	err := s.repo.Create(s.ctx, record1)
	s.Require().NoError(err)

	record2 := &traffic.TrafficRecord{
		UserID:     2,
		TunnelID:   11,
		TrafficIn:  1500,
		TrafficOut: 2500,
		RecordedAt: now,
	}
	err = s.repo.Create(s.ctx, record2)
	s.Require().NoError(err)

	// 按用户过滤
	filter := traffic.RecordListFilter{
		UserID:   1,
		Page:     1,
		PageSize: 10,
	}
	records, total, err := s.repo.List(s.ctx, filter)
	s.NoError(err)
	s.Len(records, 1)
	s.Equal(int64(1), total)
	s.Equal(uint(1), records[0].UserID)

	// 按隧道过滤
	filter = traffic.RecordListFilter{
		TunnelID: 11,
		Page:     1,
		PageSize: 10,
	}
	records, total, err = s.repo.List(s.ctx, filter)
	s.NoError(err)
	s.Len(records, 1)
	s.Equal(int64(1), total)
	s.Equal(uint(11), records[0].TunnelID)
}

// ========== SumByUserID 测试 ==========

func (s *TrafficRepositoryTestSuite) TestSumByUserID() {
	userID := uint(300)

	// 创建测试数据
	for i := 0; i < 3; i++ {
		record := &traffic.TrafficRecord{
			UserID:     userID,
			TunnelID:   uint(10 + i),
			TrafficIn:  1000,
			TrafficOut: 2000,
			RecordedAt: time.Now(),
		}
		err := s.repo.Create(s.ctx, record)
		s.Require().NoError(err)
	}

	// 统计
	totalIn, totalOut, err := s.repo.SumByUserID(s.ctx, userID, time.Time{}, time.Time{})
	s.NoError(err)
	s.Equal(int64(3000), totalIn)
	s.Equal(int64(6000), totalOut)
}

func (s *TrafficRepositoryTestSuite) TestSumByUserID_Empty() {
	totalIn, totalOut, err := s.repo.SumByUserID(s.ctx, 99999, time.Time{}, time.Time{})
	s.NoError(err)
	s.Equal(int64(0), totalIn)
	s.Equal(int64(0), totalOut)
}

// ========== SumByTunnelID 测试 ==========

func (s *TrafficRepositoryTestSuite) TestSumByTunnelID() {
	tunnelID := uint(60)

	// 创建测试数据
	for i := 0; i < 3; i++ {
		record := &traffic.TrafficRecord{
			UserID:     uint(100 + i),
			TunnelID:   tunnelID,
			TrafficIn:  1000,
			TrafficOut: 2000,
			RecordedAt: time.Now(),
		}
		err := s.repo.Create(s.ctx, record)
		s.Require().NoError(err)
	}

	// 统计
	totalIn, totalOut, err := s.repo.SumByTunnelID(s.ctx, tunnelID, time.Time{}, time.Time{})
	s.NoError(err)
	s.Equal(int64(3000), totalIn)
	s.Equal(int64(6000), totalOut)
}

// ========== DeleteOldRecords 测试 ==========

func (s *TrafficRepositoryTestSuite) TestDeleteOldRecords() {
	now := time.Now()

	// 创建旧记录
	oldRecord := &traffic.TrafficRecord{
		UserID:     1,
		TunnelID:   10,
		TrafficIn:  1000,
		TrafficOut: 2000,
		RecordedAt: now.Add(-48 * time.Hour),
	}
	err := s.repo.Create(s.ctx, oldRecord)
	s.Require().NoError(err)

	// 创建新记录
	newRecord := &traffic.TrafficRecord{
		UserID:     1,
		TunnelID:   11,
		TrafficIn:  1500,
		TrafficOut: 2500,
		RecordedAt: now,
	}
	err = s.repo.Create(s.ctx, newRecord)
	s.Require().NoError(err)

	// 删除 24 小时前的记录
	deleted, err := s.repo.DeleteOldRecords(s.ctx, now.Add(-24*time.Hour))
	s.NoError(err)
	s.Equal(int64(1), deleted)

	// 验证新记录仍然存在
	found, err := s.repo.FindByID(s.ctx, newRecord.ID)
	s.NoError(err)
	s.NotNil(found)
}
