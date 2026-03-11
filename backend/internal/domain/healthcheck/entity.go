package healthcheck

import (
	"time"
)

// CheckType 健康检查类型
type CheckType string

const (
	CheckTypeTCP  CheckType = "tcp"
	CheckTypeHTTP CheckType = "http"
	CheckTypeICMP CheckType = "icmp"
)

// CheckStatus 健康检查状态
type CheckStatus string

const (
	CheckStatusHealthy   CheckStatus = "healthy"
	CheckStatusUnhealthy CheckStatus = "unhealthy"
	CheckStatusUnknown   CheckStatus = "unknown"
)

// HealthCheck 健康检查配置聚合根
type HealthCheck struct {
	ID               uint
	NodeInstanceID   uint
	Type             CheckType
	Enabled          bool
	Interval         int // 检查间隔（秒）
	Timeout          int // 超时时间（秒）
	Retries          int // 失败重试次数
	SuccessThreshold int // 成功阈值
	FailureThreshold int // 失败阈值
	HTTPPath         *string
	HTTPMethod       *string
	ExpectedStatus   *int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NewHealthCheck 创建健康检查配置
func NewHealthCheck(nodeInstanceID uint, checkType CheckType) *HealthCheck {
	return &HealthCheck{
		NodeInstanceID:   nodeInstanceID,
		Type:             checkType,
		Enabled:          true,
		Interval:         30,
		Timeout:          5,
		Retries:          3,
		SuccessThreshold: 2,
		FailureThreshold: 3,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// IsEnabled 是否启用
func (h *HealthCheck) IsEnabled() bool {
	return h.Enabled
}

// Enable 启用
func (h *HealthCheck) Enable() {
	h.Enabled = true
	h.UpdatedAt = time.Now()
}

// Disable 禁用
func (h *HealthCheck) Disable() {
	h.Enabled = false
	h.UpdatedAt = time.Now()
}

// UpdateConfig 更新配置
func (h *HealthCheck) UpdateConfig(interval, timeout, retries, successThreshold, failureThreshold int) {
	if interval > 0 {
		h.Interval = interval
	}
	if timeout > 0 {
		h.Timeout = timeout
	}
	if retries > 0 {
		h.Retries = retries
	}
	if successThreshold > 0 {
		h.SuccessThreshold = successThreshold
	}
	if failureThreshold > 0 {
		h.FailureThreshold = failureThreshold
	}
	h.UpdatedAt = time.Now()
}

// HealthRecord 健康检查记录
type HealthRecord struct {
	ID             uint
	NodeInstanceID uint
	CheckType      CheckType
	Status         CheckStatus
	Latency        *int // 延迟（毫秒）
	ErrorMessage   *string
	CheckedAt      time.Time
}

// NewHealthRecord 创建健康检查记录
func NewHealthRecord(nodeInstanceID uint, checkType CheckType, status CheckStatus) *HealthRecord {
	return &HealthRecord{
		NodeInstanceID: nodeInstanceID,
		CheckType:      checkType,
		Status:         status,
		CheckedAt:      time.Now(),
	}
}

// IsHealthy 是否健康
func (r *HealthRecord) IsHealthy() bool {
	return r.Status == CheckStatusHealthy
}

// SetLatency 设置延迟
func (r *HealthRecord) SetLatency(latency int) {
	r.Latency = &latency
}

// SetError 设置错误
func (r *HealthRecord) SetError(err string) {
	r.ErrorMessage = &err
}

// QualityScore 节点质量评分
type QualityScore struct {
	ID             uint
	NodeInstanceID uint
	LatencyScore   float64 // 延迟评分 (0-100)
	StabilityScore float64 // 稳定性评分 (0-100)
	LoadScore      float64 // 负载评分 (0-100)
	OverallScore   float64 // 综合评分 (0-100)
	AvgLatency     *int    // 平均延迟（毫秒）
	Uptime         float64 // 可用性 (0-100)
	SuccessRate    float64 // 成功率 (0-100)
	LastCheckedAt  *time.Time
	UpdatedAt      time.Time
}

// NewQualityScore 创建质量评分
func NewQualityScore(nodeInstanceID uint) *QualityScore {
	return &QualityScore{
		NodeInstanceID: nodeInstanceID,
		LatencyScore:   0,
		StabilityScore: 0,
		LoadScore:      80, // 默认负载评分
		OverallScore:   0,
		Uptime:         0,
		SuccessRate:    0,
		UpdatedAt:      time.Now(),
	}
}

// CalculateOverallScore 计算综合评分
func (s *QualityScore) CalculateOverallScore() {
	// 权重：延迟 30%，稳定性 40%，负载 30%
	s.OverallScore = s.LatencyScore*0.3 + s.StabilityScore*0.4 + s.LoadScore*0.3
	s.UpdatedAt = time.Now()
}

// UpdateFromRecords 根据健康检查记录更新评分
func (s *QualityScore) UpdateFromRecords(records []*HealthRecord) {
	if len(records) == 0 {
		return
	}

	var totalLatency int64
	var healthyCount int
	var latencyCount int

	for _, record := range records {
		if record.IsHealthy() {
			healthyCount++
		}
		if record.Latency != nil {
			totalLatency += int64(*record.Latency)
			latencyCount++
		}
	}

	// 计算平均延迟
	if latencyCount > 0 {
		avgLatency := int(totalLatency / int64(latencyCount))
		s.AvgLatency = &avgLatency

		// 计算延迟评分
		s.LatencyScore = s.calculateLatencyScore(avgLatency)
	}

	// 计算成功率和稳定性评分
	s.SuccessRate = float64(healthyCount) / float64(len(records)) * 100
	s.StabilityScore = s.SuccessRate
	s.Uptime = s.SuccessRate

	// 更新最后检查时间
	s.LastCheckedAt = &records[0].CheckedAt

	// 计算综合评分
	s.CalculateOverallScore()
}

// calculateLatencyScore 计算延迟评分
func (s *QualityScore) calculateLatencyScore(avgLatency int) float64 {
	// 0-50ms: 100分, 50-100ms: 90分, 100-200ms: 70分, 200-500ms: 40分, >500ms: 10分
	switch {
	case avgLatency <= 50:
		return 100
	case avgLatency <= 100:
		return 90
	case avgLatency <= 200:
		return 70
	case avgLatency <= 500:
		return 40
	default:
		return 10
	}
}

// HealthStats 健康统计
type HealthStats struct {
	TotalChecks   int
	HealthyChecks int
	SuccessRate   float64
	AvgLatency    int
	Duration      string
}

// NewHealthStats 创建健康统计
func NewHealthStats(records []*HealthRecord, duration string) *HealthStats {
	stats := &HealthStats{
		TotalChecks: len(records),
		Duration:    duration,
	}

	if len(records) == 0 {
		return stats
	}

	var totalLatency int64
	var latencyCount int

	for _, record := range records {
		if record.IsHealthy() {
			stats.HealthyChecks++
		}
		if record.Latency != nil {
			totalLatency += int64(*record.Latency)
			latencyCount++
		}
	}

	if latencyCount > 0 {
		stats.AvgLatency = int(totalLatency / int64(latencyCount))
	}

	stats.SuccessRate = float64(stats.HealthyChecks) / float64(stats.TotalChecks) * 100

	return stats
}
