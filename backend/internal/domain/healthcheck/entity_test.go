package healthcheck

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewHealthCheck(t *testing.T) {
	check := NewHealthCheck(1, CheckTypeTCP)

	assert.Equal(t, uint(1), check.NodeInstanceID)
	assert.Equal(t, CheckTypeTCP, check.Type)
	assert.True(t, check.Enabled)
	assert.Equal(t, 30, check.Interval)
	assert.Equal(t, 5, check.Timeout)
	assert.Equal(t, 3, check.Retries)
	assert.Equal(t, 2, check.SuccessThreshold)
	assert.Equal(t, 3, check.FailureThreshold)
}

func TestHealthCheck_IsEnabled(t *testing.T) {
	check := NewHealthCheck(1, CheckTypeTCP)
	assert.True(t, check.IsEnabled())

	check.Disable()
	assert.False(t, check.IsEnabled())

	check.Enable()
	assert.True(t, check.IsEnabled())
}

func TestHealthCheck_UpdateConfig(t *testing.T) {
	check := NewHealthCheck(1, CheckTypeTCP)

	check.UpdateConfig(60, 10, 5, 3, 5)

	assert.Equal(t, 60, check.Interval)
	assert.Equal(t, 10, check.Timeout)
	assert.Equal(t, 5, check.Retries)
	assert.Equal(t, 3, check.SuccessThreshold)
	assert.Equal(t, 5, check.FailureThreshold)
}

func TestNewHealthRecord(t *testing.T) {
	record := NewHealthRecord(1, CheckTypeTCP, CheckStatusHealthy)

	assert.Equal(t, uint(1), record.NodeInstanceID)
	assert.Equal(t, CheckTypeTCP, record.CheckType)
	assert.Equal(t, CheckStatusHealthy, record.Status)
	assert.True(t, record.IsHealthy())
}

func TestHealthRecord_SetLatency(t *testing.T) {
	record := NewHealthRecord(1, CheckTypeTCP, CheckStatusHealthy)
	record.SetLatency(50)

	assert.NotNil(t, record.Latency)
	assert.Equal(t, 50, *record.Latency)
}

func TestHealthRecord_SetError(t *testing.T) {
	record := NewHealthRecord(1, CheckTypeTCP, CheckStatusUnhealthy)
	record.SetError("connection timeout")

	assert.NotNil(t, record.ErrorMessage)
	assert.Equal(t, "connection timeout", *record.ErrorMessage)
}

func TestNewQualityScore(t *testing.T) {
	score := NewQualityScore(1)

	assert.Equal(t, uint(1), score.NodeInstanceID)
	assert.Equal(t, 0.0, score.LatencyScore)
	assert.Equal(t, 0.0, score.StabilityScore)
	assert.Equal(t, 80.0, score.LoadScore)
	assert.Equal(t, 0.0, score.OverallScore)
}

func TestQualityScore_CalculateOverallScore(t *testing.T) {
	score := NewQualityScore(1)
	score.LatencyScore = 100
	score.StabilityScore = 90
	score.LoadScore = 80

	score.CalculateOverallScore()

	// 100*0.3 + 90*0.4 + 80*0.3 = 30 + 36 + 24 = 90
	assert.Equal(t, 90.0, score.OverallScore)
}

func TestQualityScore_UpdateFromRecords(t *testing.T) {
	score := NewQualityScore(1)

	// 创建测试记录
	records := []*HealthRecord{
		{NodeInstanceID: 1, Status: CheckStatusHealthy, Latency: intPtr(30), CheckedAt: time.Now()},
		{NodeInstanceID: 1, Status: CheckStatusHealthy, Latency: intPtr(40), CheckedAt: time.Now()},
		{NodeInstanceID: 1, Status: CheckStatusHealthy, Latency: intPtr(50), CheckedAt: time.Now()},
		{NodeInstanceID: 1, Status: CheckStatusUnhealthy, Latency: intPtr(100), CheckedAt: time.Now()},
	}

	score.UpdateFromRecords(records)

	// 平均延迟: (30+40+50+100)/4 = 55
	assert.NotNil(t, score.AvgLatency)
	assert.Equal(t, 55, *score.AvgLatency)

	// 成功率: 3/4 = 75%
	assert.Equal(t, 75.0, score.SuccessRate)
	assert.Equal(t, 75.0, score.StabilityScore)
	assert.Equal(t, 75.0, score.Uptime)

	// 延迟评分: 55ms 应该是 90 分
	assert.Equal(t, 90.0, score.LatencyScore)

	// 综合评分: 90*0.3 + 75*0.4 + 80*0.3 = 27 + 30 + 24 = 81
	assert.Equal(t, 81.0, score.OverallScore)
}

func TestQualityScore_CalculateLatencyScore(t *testing.T) {
	score := NewQualityScore(1)

	tests := []struct {
		latency  int
		expected float64
	}{
		{30, 100.0},
		{50, 100.0},
		{80, 90.0},
		{100, 90.0},
		{150, 70.0},
		{200, 70.0},
		{300, 40.0},
		{500, 40.0},
		{600, 10.0},
	}

	for _, tt := range tests {
		result := score.calculateLatencyScore(tt.latency)
		assert.Equal(t, tt.expected, result, "latency=%d", tt.latency)
	}
}

func TestNewHealthStats(t *testing.T) {
	records := []*HealthRecord{
		{Status: CheckStatusHealthy, Latency: intPtr(30)},
		{Status: CheckStatusHealthy, Latency: intPtr(40)},
		{Status: CheckStatusUnhealthy, Latency: intPtr(100)},
	}

	stats := NewHealthStats(records, "24h")

	assert.Equal(t, 3, stats.TotalChecks)
	assert.Equal(t, 2, stats.HealthyChecks)
	assert.Equal(t, 56, stats.AvgLatency) // (30+40+100)/3 = 56
	assert.InDelta(t, 66.67, stats.SuccessRate, 0.01)
	assert.Equal(t, "24h", stats.Duration)
}

func TestNewHealthStats_EmptyRecords(t *testing.T) {
	stats := NewHealthStats([]*HealthRecord{}, "24h")

	assert.Equal(t, 0, stats.TotalChecks)
	assert.Equal(t, 0, stats.HealthyChecks)
	assert.Equal(t, 0, stats.AvgLatency)
	assert.Equal(t, 0.0, stats.SuccessRate)
}

func intPtr(i int) *int {
	return &i
}
