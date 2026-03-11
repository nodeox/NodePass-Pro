package commands

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"nodepass-pro/backend/internal/domain/benefitcode"
)

// MockBenefitCodeRepository Mock 仓储
type MockBenefitCodeRepository struct {
	codes map[uint]*benefitcode.BenefitCode
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

// GenerateCodesHandlerTestSuite 测试套件
type GenerateCodesHandlerTestSuite struct {
	suite.Suite
	repo    *MockBenefitCodeRepository
	handler *GenerateCodesHandler
	ctx     context.Context
}

func (s *GenerateCodesHandlerTestSuite) SetupTest() {
	s.repo = NewMockBenefitCodeRepository()
	s.handler = NewGenerateCodesHandler(s.repo)
	s.ctx = context.Background()
}

func (s *GenerateCodesHandlerTestSuite) TestGenerateCodesSuccess() {
	cmd := GenerateCodesCommand{
		AdminID:      1,
		VIPLevel:     1,
		DurationDays: 30,
		Count:        5,
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(5, result.Total)
	s.Len(result.Codes, 5)

	// 验证生成的权益码
	for _, code := range result.Codes {
		s.NotEmpty(code.Code)
		s.Equal(1, code.VIPLevel)
		s.Equal(30, code.DurationDays)
		s.Equal(benefitcode.BenefitCodeStatusUnused, code.Status)
		s.True(code.IsEnabled)
	}
}

func (s *GenerateCodesHandlerTestSuite) TestGenerateCodesInvalidCount() {
	cmd := GenerateCodesCommand{
		AdminID:      1,
		VIPLevel:     1,
		DurationDays: 30,
		Count:        0,
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Nil(result)
	s.Equal(benefitcode.ErrInvalidCount, err)
}

func (s *GenerateCodesHandlerTestSuite) TestGenerateCodesTooMany() {
	cmd := GenerateCodesCommand{
		AdminID:      1,
		VIPLevel:     1,
		DurationDays: 30,
		Count:        1001,
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Nil(result)
}

func (s *GenerateCodesHandlerTestSuite) TestGenerateCodesInvalidDuration() {
	cmd := GenerateCodesCommand{
		AdminID:      1,
		VIPLevel:     1,
		DurationDays: 0,
		Count:        5,
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Nil(result)
	s.Equal(benefitcode.ErrInvalidDuration, err)
}

func (s *GenerateCodesHandlerTestSuite) TestGenerateCodesWithExpiration() {
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	cmd := GenerateCodesCommand{
		AdminID:      1,
		VIPLevel:     2,
		DurationDays: 60,
		Count:        3,
		ExpiresAt:    &expiresAt,
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(3, result.Total)

	for _, code := range result.Codes {
		s.NotNil(code.ExpiresAt)
		s.Equal(expiresAt.Unix(), code.ExpiresAt.Unix())
	}
}

func TestGenerateCodesHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GenerateCodesHandlerTestSuite))
}

// RedeemCodeHandlerTestSuite 测试套件
type RedeemCodeHandlerTestSuite struct {
	suite.Suite
	repo    *MockBenefitCodeRepository
	handler *RedeemCodeHandler
	ctx     context.Context
}

func (s *RedeemCodeHandlerTestSuite) SetupTest() {
	s.repo = NewMockBenefitCodeRepository()
	s.handler = NewRedeemCodeHandler(s.repo)
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

func (s *RedeemCodeHandlerTestSuite) TestRedeemCodeSuccess() {
	cmd := RedeemCodeCommand{
		UserID:          100,
		Code:            "NP-TEST-CODE-0001",
		CurrentVIPLevel: 0,
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)
	s.NotNil(result)
	s.Equal("NP-TEST-CODE-0001", result.Code)
	s.Equal(1, result.AppliedLevel)
	s.Equal(30, result.DurationDays)

	// 验证权益码已标记为已使用
	code, _ := s.repo.FindByCode(s.ctx, "NP-TEST-CODE-0001")
	s.Equal(benefitcode.BenefitCodeStatusUsed, code.Status)
	s.NotNil(code.UsedBy)
	s.Equal(uint(100), *code.UsedBy)
	s.NotNil(code.UsedAt)
}

func (s *RedeemCodeHandlerTestSuite) TestRedeemCodeNotFound() {
	cmd := RedeemCodeCommand{
		UserID:          100,
		Code:            "NOT-EXIST",
		CurrentVIPLevel: 0,
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Nil(result)
	s.Equal(benefitcode.ErrBenefitCodeNotFound, err)
}

func (s *RedeemCodeHandlerTestSuite) TestRedeemCodeAlreadyUsed() {
	// 先兑换一次
	cmd := RedeemCodeCommand{
		UserID:          100,
		Code:            "NP-TEST-CODE-0001",
		CurrentVIPLevel: 0,
	}
	s.handler.Handle(s.ctx, cmd)

	// 再次兑换
	cmd2 := RedeemCodeCommand{
		UserID:          200,
		Code:            "NP-TEST-CODE-0001",
		CurrentVIPLevel: 0,
	}

	result, err := s.handler.Handle(s.ctx, cmd2)
	s.Error(err)
	s.Nil(result)
	s.Equal(benefitcode.ErrBenefitCodeAlreadyUsed, err)
}

func (s *RedeemCodeHandlerTestSuite) TestRedeemCodeExpired() {
	// 创建过期的权益码
	expiredTime := time.Now().Add(-24 * time.Hour)
	s.repo.Create(s.ctx, &benefitcode.BenefitCode{
		Code:         "NP-EXPIRED-CODE",
		VIPLevel:     1,
		DurationDays: 30,
		Status:       benefitcode.BenefitCodeStatusUnused,
		IsEnabled:    true,
		ExpiresAt:    &expiredTime,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})

	cmd := RedeemCodeCommand{
		UserID:          100,
		Code:            "NP-EXPIRED-CODE",
		CurrentVIPLevel: 0,
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Nil(result)
	s.Equal(benefitcode.ErrBenefitCodeExpired, err)
}

func (s *RedeemCodeHandlerTestSuite) TestRedeemCodeWithHigherVIPLevel() {
	cmd := RedeemCodeCommand{
		UserID:          100,
		Code:            "NP-TEST-CODE-0001",
		CurrentVIPLevel: 2, // 当前 VIP 等级高于权益码等级
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(2, result.AppliedLevel) // 应该保持当前等级
}

func TestRedeemCodeHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(RedeemCodeHandlerTestSuite))
}

// RevokeCodeHandlerTestSuite 测试套件
type RevokeCodeHandlerTestSuite struct {
	suite.Suite
	repo    *MockBenefitCodeRepository
	handler *RevokeCodeHandler
	ctx     context.Context
}

func (s *RevokeCodeHandlerTestSuite) SetupTest() {
	s.repo = NewMockBenefitCodeRepository()
	s.handler = NewRevokeCodeHandler(s.repo)
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

func (s *RevokeCodeHandlerTestSuite) TestRevokeCodeSuccess() {
	code, _ := s.repo.FindByCode(s.ctx, "NP-TEST-CODE-0001")
	cmd := RevokeCodeCommand{
		AdminID: 1,
		CodeID:  code.ID,
	}

	err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)

	// 验证权益码已撤销
	revokedCode, _ := s.repo.FindByID(s.ctx, code.ID)
	s.Equal(benefitcode.BenefitCodeStatusRevoked, revokedCode.Status)
	s.False(revokedCode.IsEnabled)
}

func (s *RevokeCodeHandlerTestSuite) TestRevokeCodeNotFound() {
	cmd := RevokeCodeCommand{
		AdminID: 1,
		CodeID:  999,
	}

	err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Equal(benefitcode.ErrBenefitCodeNotFound, err)
}

func TestRevokeCodeHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(RevokeCodeHandlerTestSuite))
}

// DeleteCodesHandlerTestSuite 测试套件
type DeleteCodesHandlerTestSuite struct {
	suite.Suite
	repo    *MockBenefitCodeRepository
	handler *DeleteCodesHandler
	ctx     context.Context
}

func (s *DeleteCodesHandlerTestSuite) SetupTest() {
	s.repo = NewMockBenefitCodeRepository()
	s.handler = NewDeleteCodesHandler(s.repo)
	s.ctx = context.Background()

	// 预创建测试权益码
	now := time.Now()
	for i := 1; i <= 5; i++ {
		code := fmt.Sprintf("NP-TEST-CODE-%04d", i)
		s.repo.Create(s.ctx, &benefitcode.BenefitCode{
			Code:         code,
			VIPLevel:     1,
			DurationDays: 30,
			Status:       benefitcode.BenefitCodeStatusUnused,
			IsEnabled:    true,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
}

func (s *DeleteCodesHandlerTestSuite) TestDeleteCodesSuccess() {
	cmd := DeleteCodesCommand{
		AdminID: 1,
		IDs:     []uint{1, 2, 3},
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(int64(3), result.Deleted)

	// 验证已删除
	_, err = s.repo.FindByID(s.ctx, 1)
	s.Error(err)
}

func (s *DeleteCodesHandlerTestSuite) TestDeleteCodesEmptyIDs() {
	cmd := DeleteCodesCommand{
		AdminID: 1,
		IDs:     []uint{},
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Nil(result)
}

func TestDeleteCodesHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteCodesHandlerTestSuite))
}
