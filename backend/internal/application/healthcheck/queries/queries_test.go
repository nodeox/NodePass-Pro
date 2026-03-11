package queries

import (
	"context"
	"testing"
	"time"

	"nodepass-pro/backend/internal/domain/healthcheck"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetHealthCheckHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewGetHealthCheckHandler(repo)

	query := GetHealthCheckQuery{
		NodeInstanceID: 1,
	}

	expected := healthcheck.NewHealthCheck(1, healthcheck.CheckTypeTCP)
	repo.On("FindHealthCheckByNodeInstance", ctx, uint(1)).Return(expected, nil)

	result, err := handler.Handle(ctx, query)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	repo.AssertExpectations(t)
}

func TestGetHealthRecordsHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewGetHealthRecordsHandler(repo)

	query := GetHealthRecordsQuery{
		NodeInstanceID: 1,
		Limit:          50,
	}

	records := []*healthcheck.HealthRecord{
		healthcheck.NewHealthRecord(1, healthcheck.CheckTypeTCP, healthcheck.CheckStatusHealthy),
	}
	repo.On("FindHealthRecordsByNodeInstance", ctx, uint(1), 50).Return(records, nil)

	result, err := handler.Handle(ctx, query)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	repo.AssertExpectations(t)
}

func TestGetHealthRecordsHandler_Handle_DefaultLimit(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewGetHealthRecordsHandler(repo)

	query := GetHealthRecordsQuery{
		NodeInstanceID: 1,
		Limit:          0, // 应该使用默认值 100
	}

	records := []*healthcheck.HealthRecord{}
	repo.On("FindHealthRecordsByNodeInstance", ctx, uint(1), 100).Return(records, nil)

	result, err := handler.Handle(ctx, query)
	assert.NoError(t, err)
	assert.Len(t, result, 0)

	repo.AssertExpectations(t)
}

func TestGetHealthRecordsHandler_Handle_MaxLimit(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewGetHealthRecordsHandler(repo)

	query := GetHealthRecordsQuery{
		NodeInstanceID: 1,
		Limit:          2000, // 超过最大值，应该使用 1000
	}

	records := []*healthcheck.HealthRecord{}
	repo.On("FindHealthRecordsByNodeInstance", ctx, uint(1), 1000).Return(records, nil)

	result, err := handler.Handle(ctx, query)
	assert.NoError(t, err)
	assert.Len(t, result, 0)

	repo.AssertExpectations(t)
}

func TestGetHealthStatsHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewGetHealthStatsHandler(repo)

	query := GetHealthStatsQuery{
		NodeInstanceID: 1,
		Duration:       24 * time.Hour,
	}

	records := []*healthcheck.HealthRecord{
		healthcheck.NewHealthRecord(1, healthcheck.CheckTypeTCP, healthcheck.CheckStatusHealthy),
		healthcheck.NewHealthRecord(1, healthcheck.CheckTypeTCP, healthcheck.CheckStatusUnhealthy),
	}

	// 使用 mock.MatchedBy 来匹配时间参数
	repo.On("FindHealthRecordsByTimeRange", ctx, uint(1), mock.MatchedBy(func(t time.Time) bool {
		return true // 接受任何时间
	})).Return(records, nil)

	result, err := handler.Handle(ctx, query)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.TotalChecks)
	assert.Equal(t, 1, result.HealthyChecks)

	repo.AssertExpectations(t)
}

func TestGetQualityScoreHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewGetQualityScoreHandler(repo)

	query := GetQualityScoreQuery{
		NodeInstanceID: 1,
	}

	expected := healthcheck.NewQualityScore(1)
	repo.On("FindQualityScoreByNodeInstance", ctx, uint(1)).Return(expected, nil)

	result, err := handler.Handle(ctx, query)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	repo.AssertExpectations(t)
}

func TestListQualityScoresHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewListQualityScoresHandler(repo)

	query := ListQualityScoresQuery{
		UserID: 1,
	}

	scores := []*healthcheck.QualityScore{
		healthcheck.NewQualityScore(1),
		healthcheck.NewQualityScore(2),
	}
	repo.On("ListQualityScoresByUser", ctx, uint(1)).Return(scores, nil)

	result, err := handler.Handle(ctx, query)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	repo.AssertExpectations(t)
}
