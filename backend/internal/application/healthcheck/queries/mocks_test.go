package queries

import (
	"context"
	"time"

	"nodepass-pro/backend/internal/domain/healthcheck"

	"github.com/stretchr/testify/mock"
)

// MockRepository 模拟仓储
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateHealthCheck(ctx context.Context, check *healthcheck.HealthCheck) error {
	args := m.Called(ctx, check)
	if args.Get(0) != nil {
		check.ID = 1
	}
	return args.Error(0)
}

func (m *MockRepository) FindHealthCheckByID(ctx context.Context, id uint) (*healthcheck.HealthCheck, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*healthcheck.HealthCheck), args.Error(1)
}

func (m *MockRepository) FindHealthCheckByNodeInstance(ctx context.Context, nodeInstanceID uint) (*healthcheck.HealthCheck, error) {
	args := m.Called(ctx, nodeInstanceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*healthcheck.HealthCheck), args.Error(1)
}

func (m *MockRepository) UpdateHealthCheck(ctx context.Context, check *healthcheck.HealthCheck) error {
	args := m.Called(ctx, check)
	return args.Error(0)
}

func (m *MockRepository) DeleteHealthCheck(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListEnabledHealthChecks(ctx context.Context) ([]*healthcheck.HealthCheck, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*healthcheck.HealthCheck), args.Error(1)
}

func (m *MockRepository) CreateHealthRecord(ctx context.Context, record *healthcheck.HealthRecord) error {
	args := m.Called(ctx, record)
	if args.Get(0) != nil {
		record.ID = 1
	}
	return args.Error(0)
}

func (m *MockRepository) FindHealthRecordsByNodeInstance(ctx context.Context, nodeInstanceID uint, limit int) ([]*healthcheck.HealthRecord, error) {
	args := m.Called(ctx, nodeInstanceID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*healthcheck.HealthRecord), args.Error(1)
}

func (m *MockRepository) FindHealthRecordsByTimeRange(ctx context.Context, nodeInstanceID uint, startTime time.Time) ([]*healthcheck.HealthRecord, error) {
	args := m.Called(ctx, nodeInstanceID, startTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*healthcheck.HealthRecord), args.Error(1)
}

func (m *MockRepository) DeleteOldHealthRecords(ctx context.Context, cutoffTime time.Time) (int64, error) {
	args := m.Called(ctx, cutoffTime)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) CreateOrUpdateQualityScore(ctx context.Context, score *healthcheck.QualityScore) error {
	args := m.Called(ctx, score)
	return args.Error(0)
}

func (m *MockRepository) FindQualityScoreByNodeInstance(ctx context.Context, nodeInstanceID uint) (*healthcheck.QualityScore, error) {
	args := m.Called(ctx, nodeInstanceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*healthcheck.QualityScore), args.Error(1)
}

func (m *MockRepository) ListQualityScoresByUser(ctx context.Context, userID uint) ([]*healthcheck.QualityScore, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*healthcheck.QualityScore), args.Error(1)
}

// MockChecker 模拟检查器
type MockChecker struct {
	mock.Mock
}

func (m *MockChecker) Check(ctx context.Context, nodeInstanceID uint, config *healthcheck.HealthCheck) (*healthcheck.HealthRecord, error) {
	args := m.Called(ctx, nodeInstanceID, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*healthcheck.HealthRecord), args.Error(1)
}
