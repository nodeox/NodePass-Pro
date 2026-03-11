package commands

import (
	"context"

	"nodepass-pro/backend/internal/domain/tunneltemplate"

	"github.com/stretchr/testify/mock"
)

// MockRepository 模拟仓储
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, template *tunneltemplate.TunnelTemplate) error {
	args := m.Called(ctx, template)
	if args.Get(0) != nil {
		template.ID = 1
	}
	return args.Error(0)
}

func (m *MockRepository) FindByID(ctx context.Context, id uint) (*tunneltemplate.TunnelTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tunneltemplate.TunnelTemplate), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, template *tunneltemplate.TunnelTemplate) error {
	args := m.Called(ctx, template)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) List(ctx context.Context, filter tunneltemplate.ListFilter) ([]*tunneltemplate.TunnelTemplate, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*tunneltemplate.TunnelTemplate), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) FindByUserAndName(ctx context.Context, userID uint, name string) (*tunneltemplate.TunnelTemplate, error) {
	args := m.Called(ctx, userID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tunneltemplate.TunnelTemplate), args.Error(1)
}

func (m *MockRepository) IncrementUsageCount(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
