package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"nodepass-pro/backend/internal/domain/benefitcode"
	"nodepass-pro/backend/internal/models"
)

// BenefitCodeRepositoryTestSuite 测试套件
type BenefitCodeRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo *BenefitCodeRepository
	ctx  context.Context
}

func (s *BenefitCodeRepositoryTestSuite) SetupTest() {
	// 使用 SQLite 内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	s.NoError(err)

	// 自动迁移
	err = db.AutoMigrate(&models.BenefitCode{})
	s.NoError(err)

	s.db = db
	s.repo = NewBenefitCodeRepository(db)
	s.ctx = context.Background()
}

func (s *BenefitCodeRepositoryTestSuite) TearDownTest() {
	sqlDB, _ := s.db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}
}

func (s *BenefitCodeRepositoryTestSuite) TestCreate() {
	now := time.Now()
	code := &benefitcode.BenefitCode{
		Code:         "NP-TEST-CODE-0001",
		VIPLevel:     1,
		DurationDays: 30,
		Status:       benefitcode.BenefitCodeStatusUnused,
		IsEnabled:    true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err := s.repo.Create(s.ctx, code)
	s.NoError(err)
	s.NotZero(code.ID)
}

func (s *BenefitCodeRepositoryTestSuite) TestCreateDuplicate() {
	now := time.Now()
	code1 := &benefitcode.BenefitCode{
		Code:         "NP-DUPLICATE",
		VIPLevel:     1,
		DurationDays: 30,
		Status:       benefitcode.BenefitCodeStatusUnused,
		IsEnabled:    true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err := s.repo.Create(s.ctx, code1)
	s.NoError(err)

	// 尝试创建重复的权益码
	code2 := &benefitcode.BenefitCode{
		Code:         "NP-DUPLICATE",
		VIPLevel:     2,
		DurationDays: 60,
		Status:       benefitcode.BenefitCodeStatusUnused,
		IsEnabled:    true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = s.repo.Create(s.ctx, code2)
	s.Error(err)
	s.Equal(benefitcode.ErrBenefitCodeAlreadyExists, err)
}

func (s *BenefitCodeRepositoryTestSuite) TestBatchCreate() {
	now := time.Now()
	codes := []*benefitcode.BenefitCode{
		{
			Code:         "NP-BATCH-0001",
			VIPLevel:     1,
			DurationDays: 30,
			Status:       benefitcode.BenefitCodeStatusUnused,
			IsEnabled:    true,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			Code:         "NP-BATCH-0002",
			VIPLevel:     1,
			DurationDays: 30,
			Status:       benefitcode.BenefitCodeStatusUnused,
			IsEnabled:    true,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	err := s.repo.BatchCreate(s.ctx, codes)
	s.NoError(err)
	s.NotZero(codes[0].ID)
	s.NotZero(codes[1].ID)
}

func (s *BenefitCodeRepositoryTestSuite) TestFindByID() {
	now := time.Now()
	code := &benefitcode.BenefitCode{
		Code:         "NP-FIND-BY-ID",
		VIPLevel:     1,
		DurationDays: 30,
		Status:       benefitcode.BenefitCodeStatusUnused,
		IsEnabled:    true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.repo.Create(s.ctx, code)

	found, err := s.repo.FindByID(s.ctx, code.ID)
	s.NoError(err)
	s.NotNil(found)
	s.Equal(code.Code, found.Code)
}

func (s *BenefitCodeRepositoryTestSuite) TestFindByIDNotFound() {
	found, err := s.repo.FindByID(s.ctx, 999)
	s.Error(err)
	s.Nil(found)
	s.Equal(benefitcode.ErrBenefitCodeNotFound, err)
}

func (s *BenefitCodeRepositoryTestSuite) TestFindByCode() {
	now := time.Now()
	code := &benefitcode.BenefitCode{
		Code:         "NP-FIND-BY-CODE",
		VIPLevel:     1,
		DurationDays: 30,
		Status:       benefitcode.BenefitCodeStatusUnused,
		IsEnabled:    true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.repo.Create(s.ctx, code)

	found, err := s.repo.FindByCode(s.ctx, "NP-FIND-BY-CODE")
	s.NoError(err)
	s.NotNil(found)
	s.Equal(code.ID, found.ID)
}

func (s *BenefitCodeRepositoryTestSuite) TestFindByCodeNotFound() {
	found, err := s.repo.FindByCode(s.ctx, "NOT-EXIST")
	s.Error(err)
	s.Nil(found)
	s.Equal(benefitcode.ErrBenefitCodeNotFound, err)
}

func (s *BenefitCodeRepositoryTestSuite) TestUpdate() {
	now := time.Now()
	code := &benefitcode.BenefitCode{
		Code:         "NP-UPDATE-TEST",
		VIPLevel:     1,
		DurationDays: 30,
		Status:       benefitcode.BenefitCodeStatusUnused,
		IsEnabled:    true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.repo.Create(s.ctx, code)

	// 更新状态
	userID := uint(100)
	code.Status = benefitcode.BenefitCodeStatusUsed
	code.UsedBy = &userID
	code.UsedAt = &now

	err := s.repo.Update(s.ctx, code)
	s.NoError(err)

	// 验证更新
	updated, _ := s.repo.FindByID(s.ctx, code.ID)
	s.Equal(benefitcode.BenefitCodeStatusUsed, updated.Status)
	s.NotNil(updated.UsedBy)
	s.Equal(uint(100), *updated.UsedBy)
}

func (s *BenefitCodeRepositoryTestSuite) TestDelete() {
	now := time.Now()
	code := &benefitcode.BenefitCode{
		Code:         "NP-DELETE-TEST",
		VIPLevel:     1,
		DurationDays: 30,
		Status:       benefitcode.BenefitCodeStatusUnused,
		IsEnabled:    true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.repo.Create(s.ctx, code)

	err := s.repo.Delete(s.ctx, code.ID)
	s.NoError(err)

	// 验证已删除
	_, err = s.repo.FindByID(s.ctx, code.ID)
	s.Error(err)
	s.Equal(benefitcode.ErrBenefitCodeNotFound, err)
}

func (s *BenefitCodeRepositoryTestSuite) TestDeleteNotFound() {
	err := s.repo.Delete(s.ctx, 999)
	s.Error(err)
	s.Equal(benefitcode.ErrBenefitCodeNotFound, err)
}

func (s *BenefitCodeRepositoryTestSuite) TestBatchDelete() {
	now := time.Now()
	for i := 1; i <= 5; i++ {
		s.repo.Create(s.ctx, &benefitcode.BenefitCode{
			Code:         "NP-BATCH-DEL-" + string(rune('0'+i)),
			VIPLevel:     1,
			DurationDays: 30,
			Status:       benefitcode.BenefitCodeStatusUnused,
			IsEnabled:    true,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	deleted, err := s.repo.BatchDelete(s.ctx, []uint{1, 2, 3})
	s.NoError(err)
	s.Equal(int64(3), deleted)
}

func (s *BenefitCodeRepositoryTestSuite) TestList() {
	now := time.Now()
	for i := 1; i <= 10; i++ {
		status := benefitcode.BenefitCodeStatusUnused
		if i > 5 {
			status = benefitcode.BenefitCodeStatusUsed
		}
		s.repo.Create(s.ctx, &benefitcode.BenefitCode{
			Code:         "NP-LIST-" + string(rune('0'+i)),
			VIPLevel:     1,
			DurationDays: 30,
			Status:       status,
			IsEnabled:    true,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	codes, total, err := s.repo.List(s.ctx, benefitcode.ListFilter{
		Page:     1,
		PageSize: 20,
	})
	s.NoError(err)
	s.GreaterOrEqual(len(codes), 10)
	s.GreaterOrEqual(int(total), 10)
}

func (s *BenefitCodeRepositoryTestSuite) TestListByStatus() {
	now := time.Now()
	for i := 1; i <= 10; i++ {
		status := benefitcode.BenefitCodeStatusUnused
		if i > 5 {
			status = benefitcode.BenefitCodeStatusUsed
		}
		s.repo.Create(s.ctx, &benefitcode.BenefitCode{
			Code:         "NP-STATUS-" + string(rune('0'+i)),
			VIPLevel:     1,
			DurationDays: 30,
			Status:       status,
			IsEnabled:    true,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	codes, total, err := s.repo.List(s.ctx, benefitcode.ListFilter{
		Status:   benefitcode.BenefitCodeStatusUnused,
		Page:     1,
		PageSize: 20,
	})
	s.NoError(err)
	s.GreaterOrEqual(len(codes), 5)
	s.GreaterOrEqual(int(total), 5)
	for _, code := range codes {
		s.Equal(benefitcode.BenefitCodeStatusUnused, code.Status)
	}
}

func (s *BenefitCodeRepositoryTestSuite) TestCountByStatus() {
	now := time.Now()
	for i := 1; i <= 10; i++ {
		status := benefitcode.BenefitCodeStatusUnused
		if i > 7 {
			status = benefitcode.BenefitCodeStatusUsed
		}
		s.repo.Create(s.ctx, &benefitcode.BenefitCode{
			Code:         "NP-COUNT-" + string(rune('0'+i)),
			VIPLevel:     1,
			DurationDays: 30,
			Status:       status,
			IsEnabled:    true,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	unusedCount, err := s.repo.CountByStatus(s.ctx, benefitcode.BenefitCodeStatusUnused)
	s.NoError(err)
	s.GreaterOrEqual(int(unusedCount), 7)

	usedCount, err := s.repo.CountByStatus(s.ctx, benefitcode.BenefitCodeStatusUsed)
	s.NoError(err)
	s.GreaterOrEqual(int(usedCount), 3)
}

func (s *BenefitCodeRepositoryTestSuite) TestFindExpiredCodes() {
	now := time.Now()
	expiredTime := now.Add(-24 * time.Hour)

	// 创建过期的权益码
	for i := 1; i <= 3; i++ {
		s.repo.Create(s.ctx, &benefitcode.BenefitCode{
			Code:         "NP-EXPIRED-" + string(rune('0'+i)),
			VIPLevel:     1,
			DurationDays: 30,
			Status:       benefitcode.BenefitCodeStatusUnused,
			IsEnabled:    true,
			ExpiresAt:    &expiredTime,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	// 创建未过期的权益码
	futureTime := now.Add(24 * time.Hour)
	s.repo.Create(s.ctx, &benefitcode.BenefitCode{
		Code:         "NP-VALID-CODE",
		VIPLevel:     1,
		DurationDays: 30,
		Status:       benefitcode.BenefitCodeStatusUnused,
		IsEnabled:    true,
		ExpiresAt:    &futureTime,
		CreatedAt:    now,
		UpdatedAt:    now,
	})

	codes, err := s.repo.FindExpiredCodes(s.ctx, 10)
	s.NoError(err)
	s.GreaterOrEqual(len(codes), 3)
	for _, code := range codes {
		s.True(code.IsExpired())
	}
}

func TestBenefitCodeRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(BenefitCodeRepositoryTestSuite))
}
