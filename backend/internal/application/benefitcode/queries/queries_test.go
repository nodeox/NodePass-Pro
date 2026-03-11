package queries

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"nodepass-pro/backend/internal/domain/benefitcode"
)

// MockBenefitCodeRepository Mock 仓储
type MockBenefitCodeRepository struct {
	codes  map[uint]*benefitcode.BenefitCode
	byCode map[string]*benefitcode.BenefitCode
	nextID uint
}

func NewMockBenefitCodeRepository() *MockBenefitCodeRepository {
	return &MockBenefitCodeRepository{
		codes:  make(map[uint]*benefitcode.BenefitCode),
		byCode: make(map[string]*benefitcode.BenefitCode),
		nextID: 1,
	}
}

func (m *MockBenefitCodeRepository) Create(ctx context.Context, code *benefitcode.BenefitCode) error {
	if _, exists := m.byCode[code.Code]; exists {
		return benefitcode.ErrBenefitCodeAlreadyExists
	}
	code.ID = m.nextID
	m.nextID++
	m.codes[code.ID] = code
	m.byCode[code.Code] = code
	return nil
}

func (m *MockBenefitCodeRepository) BatchCreate(ctx context.Context, codes []*benefitcode.BenefitCode) error {
	for _, code := range codes {
		if err := m.Create(ctx, code); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockBenefitCodeRepository) FindByID(ctx context.Context, id uint) (*benefitcode.BenefitCode, error) {
	code, ok := m.codes[id]
	if !ok {
		return nil, benefitcode.ErrBenefitCodeNotFound
	}
	return code, nil
}

func (m *MockBenefitCodeRepository) FindByCode(ctx context.Context, code string) (*benefitcode.BenefitCode, error) {
	c, ok := m.byCode[code]
	if !ok {
		return nil, benefitcode.ErrBenefitCodeNotFound
	}
	return c, nil
}

func (m *MockBenefitCodeRepository) Update(ctx context.Context, code *benefitcode.BenefitCode) error {
	if _, ok := m.codes[code.ID]; !ok {
		return benefitcode.ErrBenefitCodeNotFound
	}
	m.codes[code.ID] = code
	m.byCode[code.Code] = code
	return nil
}

func (m *MockBenefitCodeRepository) Delete(ctx context.Context, id uint) error {
	code, ok := m.codes[id]
	if !ok {
		return benefitcode.ErrBenefitCodeNotFound
	}
	delete(m.codes, id)
	delete(m.byCode, code.Code)
	return nil
}

func (m *MockBenefitCodeRepository) BatchDelete(ctx context.Context, ids []uint) (int64, error) {
	count := int64(0)
	for _, id := range ids {
		if err := m.Delete(ctx, id); err == nil {
			count++
		}
	}
	return count, nil
}

func (m *MockBenefitCodeRepository) List(ctx context.Context, filter benefitcode.ListFilter) ([]*benefitcode.BenefitCode, int64, error) {
	var result []*benefitcode.BenefitCode
	for _, code := range m.codes {
		if filter.Status != "" && code.Status != filter.Status {
			continue
		}
		if filter.VIPLevel != nil && code.VIPLevel != *filter.VIPLevel {
			continue
		}
		if filter.UsedBy != nil && (code.UsedBy == nil || *code.UsedBy != *filter.UsedBy) {
			continue
		}
		result = append(result, code)
	}
	return result, int64(len(result)), nil
}

func (m *MockBenefitCodeRepository) CountByStatus(ctx context.Context, status benefitcode.BenefitCodeStatus) (int64, error) {
	count := int64(0)
	for _, code := range m.codes {
		if code.Status == status {
			count++
		}
	}
	return count, nil
}

func (m *MockBenefitCodeRepository) FindExpiredCodes(ctx context.Context, limit int) ([]*benefitcode.BenefitCode, error) {
	var result []*benefitcode.BenefitCode
	now := time.Now()
	for _, code := range m.codes {
		if code.ExpiresAt != nil && code.ExpiresAt.Before(now) && code.Status == benefitcode.BenefitCodeStatusUnused {
			result = append(result, code)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// GetCodeHandlerTestSuite 测试套件
type GetCodeHandlerTestSuite struct {
	suite.Suite
	repo    *MockBenefitCodeRepository
	handler *GetCodeHandler
	ctx     context.Context
}

func (s *GetCodeHandlerTestSuite) SetupTest() {
	s.repo = NewMockBenefitCodeRepository()
	s.handler = NewGetCodeHandler(s.repo)
	s.ctx = context.Background()

	// 预创建测试权益码
	now := time.Now()
	s.repo.Create(s.ctx, &benefitcode.BenefitCode{
		Code:         "NP-TEST-CODE-0001",
		VIPLevel:     1,
		DurationDays: 30,
		Status:       benefitcode.BenefitCodeStatusUnused,
		IsEnabled:    true,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}

func (s *GetCodeHandlerTestSuite) TestGetCodeSuccess() {
	query := GetCodeQuery{ID: 1}
	code, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.NotNil(code)
	s.Equal("NP-TEST-CODE-0001", code.Code)
}

func (s *GetCodeHandlerTestSuite) TestGetCodeNotFound() {
	query := GetCodeQuery{ID: 999}
	code, err := s.handler.Handle(s.ctx, query)
	s.Error(err)
	s.Nil(code)
	s.Equal(benefitcode.ErrBenefitCodeNotFound, err)
}

func TestGetCodeHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GetCodeHandlerTestSuite))
}

// ListCodesHandlerTestSuite 测试套件
type ListCodesHandlerTestSuite struct {
	suite.Suite
	repo    *MockBenefitCodeRepository
	handler *ListCodesHandler
	ctx     context.Context
}

func (s *ListCodesHandlerTestSuite) SetupTest() {
	s.repo = NewMockBenefitCodeRepository()
	s.handler = NewListCodesHandler(s.repo)
	s.ctx = context.Background()

	// 预创建测试权益码
	now := time.Now()
	for i := 1; i <= 10; i++ {
		status := benefitcode.BenefitCodeStatusUnused
		if i > 5 {
			status = benefitcode.BenefitCodeStatusUsed
		}
		code := &benefitcode.BenefitCode{
			Code:         "NP-TEST-CODE-" + string(rune('0'+i)),
			VIPLevel:     1,
			DurationDays: 30,
			Status:       status,
			IsEnabled:    true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if status == benefitcode.BenefitCodeStatusUsed {
			userID := uint(100)
			code.UsedBy = &userID
			code.UsedAt = &now
		}
		s.repo.Create(s.ctx, code)
	}
}

func (s *ListCodesHandlerTestSuite) TestListAllCodes() {
	query := ListCodesQuery{
		Page:     1,
		PageSize: 20,
	}

	result, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.NotNil(result)
	s.GreaterOrEqual(len(result.List), 10)
	s.GreaterOrEqual(int(result.Total), 10)
}

func (s *ListCodesHandlerTestSuite) TestListCodesByStatus() {
	query := ListCodesQuery{
		Status:   benefitcode.BenefitCodeStatusUnused,
		Page:     1,
		PageSize: 20,
	}

	result, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.NotNil(result)
	s.GreaterOrEqual(len(result.List), 5)
	for _, code := range result.List {
		s.Equal(benefitcode.BenefitCodeStatusUnused, code.Status)
	}
}

func (s *ListCodesHandlerTestSuite) TestListCodesByVIPLevel() {
	vipLevel := 1
	query := ListCodesQuery{
		VIPLevel: &vipLevel,
		Page:     1,
		PageSize: 20,
	}

	result, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.NotNil(result)
	s.GreaterOrEqual(len(result.List), 10)
}

func TestListCodesHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ListCodesHandlerTestSuite))
}

// ValidateCodeHandlerTestSuite 测试套件
type ValidateCodeHandlerTestSuite struct {
	suite.Suite
	repo    *MockBenefitCodeRepository
	handler *ValidateCodeHandler
	ctx     context.Context
}

func (s *ValidateCodeHandlerTestSuite) SetupTest() {
	s.repo = NewMockBenefitCodeRepository()
	s.handler = NewValidateCodeHandler(s.repo)
	s.ctx = context.Background()

	// 预创建测试权益码
	now := time.Now()
	s.repo.Create(s.ctx, &benefitcode.BenefitCode{
		Code:         "NP-VALID-CODE",
		VIPLevel:     1,
		DurationDays: 30,
		Status:       benefitcode.BenefitCodeStatusUnused,
		IsEnabled:    true,
		CreatedAt:    now,
		UpdatedAt:    now,
	})

	// 已使用的权益码
	userID := uint(100)
	s.repo.Create(s.ctx, &benefitcode.BenefitCode{
		Code:         "NP-USED-CODE",
		VIPLevel:     1,
		DurationDays: 30,
		Status:       benefitcode.BenefitCodeStatusUsed,
		IsEnabled:    true,
		UsedBy:       &userID,
		UsedAt:       &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	})

	// 过期的权益码
	expiredTime := now.Add(-24 * time.Hour)
	s.repo.Create(s.ctx, &benefitcode.BenefitCode{
		Code:         "NP-EXPIRED-CODE",
		VIPLevel:     1,
		DurationDays: 30,
		Status:       benefitcode.BenefitCodeStatusUnused,
		IsEnabled:    true,
		ExpiresAt:    &expiredTime,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}

func (s *ValidateCodeHandlerTestSuite) TestValidateCodeSuccess() {
	query := ValidateCodeQuery{Code: "NP-VALID-CODE"}
	result, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.NotNil(result)
	s.True(result.Valid)
	s.NotNil(result.Code)
}

func (s *ValidateCodeHandlerTestSuite) TestValidateCodeNotFound() {
	query := ValidateCodeQuery{Code: "NOT-EXIST"}
	result, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.NotNil(result)
	s.False(result.Valid)
	s.Equal("权益码不存在", result.ErrorMessage)
}

func (s *ValidateCodeHandlerTestSuite) TestValidateCodeAlreadyUsed() {
	query := ValidateCodeQuery{Code: "NP-USED-CODE"}
	result, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.NotNil(result)
	s.False(result.Valid)
	s.Equal("权益码已使用", result.ErrorMessage)
}

func (s *ValidateCodeHandlerTestSuite) TestValidateCodeExpired() {
	query := ValidateCodeQuery{Code: "NP-EXPIRED-CODE"}
	result, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.NotNil(result)
	s.False(result.Valid)
	s.Equal("权益码已过期", result.ErrorMessage)
}

func TestValidateCodeHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ValidateCodeHandlerTestSuite))
}

// GetStatsHandlerTestSuite 测试套件
type GetStatsHandlerTestSuite struct {
	suite.Suite
	repo    *MockBenefitCodeRepository
	handler *GetStatsHandler
	ctx     context.Context
}

func (s *GetStatsHandlerTestSuite) SetupTest() {
	s.repo = NewMockBenefitCodeRepository()
	s.handler = NewGetStatsHandler(s.repo)
	s.ctx = context.Background()

	// 预创建测试权益码
	now := time.Now()
	for i := 1; i <= 15; i++ {
		status := benefitcode.BenefitCodeStatusUnused
		if i > 10 {
			status = benefitcode.BenefitCodeStatusUsed
		} else if i > 12 {
			status = benefitcode.BenefitCodeStatusRevoked
		}
		s.repo.Create(s.ctx, &benefitcode.BenefitCode{
			Code:         "NP-TEST-" + string(rune('0'+i)),
			VIPLevel:     1,
			DurationDays: 30,
			Status:       status,
			IsEnabled:    true,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
}

func (s *GetStatsHandlerTestSuite) TestGetStats() {
	query := GetStatsQuery{}
	result, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.NotNil(result)
	s.GreaterOrEqual(result.TotalCodes, int64(15))
	s.GreaterOrEqual(result.UnusedCodes, int64(10))
}

func TestGetStatsHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GetStatsHandlerTestSuite))
}
