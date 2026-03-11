package commands

import (
	"context"
	"testing"

	"nodepass-pro/backend/internal/domain/healthcheck"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateHealthCheckHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewCreateHealthCheckHandler(repo)

	cmd := CreateHealthCheckCommand{
		NodeInstanceID: 1,
		Type:           healthcheck.CheckTypeTCP,
		Interval:       60,
		Timeout:        10,
	}

	repo.On("FindHealthCheckByNodeInstance", ctx, uint(1)).Return(nil, healthcheck.ErrHealthCheckNotFound)
	repo.On("CreateHealthCheck", ctx, mock.AnythingOfType("*healthcheck.HealthCheck")).Return(nil)

	check, err := handler.Handle(ctx, cmd)
	assert.NoError(t, err)
	assert.NotNil(t, check)
	assert.Equal(t, uint(1), check.NodeInstanceID)
	assert.Equal(t, healthcheck.CheckTypeTCP, check.Type)

	repo.AssertExpectations(t)
}

func TestCreateHealthCheckHandler_Handle_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewCreateHealthCheckHandler(repo)

	cmd := CreateHealthCheckCommand{
		NodeInstanceID: 1,
		Type:           healthcheck.CheckTypeTCP,
	}

	existing := healthcheck.NewHealthCheck(1, healthcheck.CheckTypeTCP)
	repo.On("FindHealthCheckByNodeInstance", ctx, uint(1)).Return(existing, nil)

	_, err := handler.Handle(ctx, cmd)
	assert.ErrorIs(t, err, healthcheck.ErrHealthCheckAlreadyExists)

	repo.AssertExpectations(t)
}

func TestUpdateHealthCheckHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewUpdateHealthCheckHandler(repo)

	interval := 60
	enabled := false
	cmd := UpdateHealthCheckCommand{
		NodeInstanceID: 1,
		Interval:       &interval,
		Enabled:        &enabled,
	}

	existing := healthcheck.NewHealthCheck(1, healthcheck.CheckTypeTCP)
	repo.On("FindHealthCheckByNodeInstance", ctx, uint(1)).Return(existing, nil)
	repo.On("UpdateHealthCheck", ctx, mock.AnythingOfType("*healthcheck.HealthCheck")).Return(nil)

	err := handler.Handle(ctx, cmd)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestDeleteHealthCheckHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewDeleteHealthCheckHandler(repo)

	cmd := DeleteHealthCheckCommand{
		NodeInstanceID: 1,
	}

	existing := healthcheck.NewHealthCheck(1, healthcheck.CheckTypeTCP)
	existing.ID = 1
	repo.On("FindHealthCheckByNodeInstance", ctx, uint(1)).Return(existing, nil)
	repo.On("DeleteHealthCheck", ctx, uint(1)).Return(nil)

	err := handler.Handle(ctx, cmd)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestPerformHealthCheckHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	checker := new(MockChecker)
	handler := NewPerformHealthCheckHandler(repo, checker)

	cmd := PerformHealthCheckCommand{
		NodeInstanceID: 1,
	}

	config := healthcheck.NewHealthCheck(1, healthcheck.CheckTypeTCP)
	record := healthcheck.NewHealthRecord(1, healthcheck.CheckTypeTCP, healthcheck.CheckStatusHealthy)
	record.SetLatency(50)

	repo.On("FindHealthCheckByNodeInstance", ctx, uint(1)).Return(config, nil)
	checker.On("Check", ctx, uint(1), config).Return(record, nil)
	repo.On("CreateHealthRecord", ctx, mock.AnythingOfType("*healthcheck.HealthRecord")).Return(nil)
	repo.On("FindHealthRecordsByNodeInstance", ctx, uint(1), 100).Return([]*healthcheck.HealthRecord{record}, nil)
	repo.On("FindQualityScoreByNodeInstance", ctx, uint(1)).Return(nil, healthcheck.ErrHealthCheckNotFound)
	repo.On("CreateOrUpdateQualityScore", ctx, mock.AnythingOfType("*healthcheck.QualityScore")).Return(nil)

	result, err := handler.Handle(ctx, cmd)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, healthcheck.CheckStatusHealthy, result.Status)

	repo.AssertExpectations(t)
	checker.AssertExpectations(t)
}

func TestCleanupOldRecordsHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewCleanupOldRecordsHandler(repo)

	cmd := CleanupOldRecordsCommand{
		RetentionDays: 30,
	}

	repo.On("DeleteOldHealthRecords", ctx, mock.Anything).Return(int64(10), nil)

	count, err := handler.Handle(ctx, cmd)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), count)

	repo.AssertExpectations(t)
}
