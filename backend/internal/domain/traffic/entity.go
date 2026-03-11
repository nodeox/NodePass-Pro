package traffic

import (
	"errors"
	"time"
)

// TrafficRecord 流量记录实体
type TrafficRecord struct {
	ID         uint
	UserID     uint
	TunnelID   uint
	TrafficIn  int64
	TrafficOut int64
	RecordedAt time.Time
	CreatedAt  time.Time
}

// TrafficQuota 流量配额实体
type TrafficQuota struct {
	UserID       uint
	Quota        int64
	Used         int64
	ResetAt      time.Time
	LastResetAt  *time.Time
}

// 领域错误
var (
	ErrQuotaExceeded    = errors.New("流量配额已超限")
	ErrInvalidTraffic   = errors.New("无效的流量数据")
	ErrRecordNotFound   = errors.New("流量记录不存在")
)

// HasQuota 检查是否有足够配额
func (q *TrafficQuota) HasQuota(required int64) bool {
	if q.Quota < 0 {
		return true // 无限制
	}
	return q.Used+required <= q.Quota
}

// Consume 消耗流量
func (q *TrafficQuota) Consume(amount int64) error {
	if !q.HasQuota(amount) {
		return ErrQuotaExceeded
	}
	q.Used += amount
	return nil
}

// Reset 重置配额
func (q *TrafficQuota) Reset() {
	q.Used = 0
	now := time.Now()
	q.LastResetAt = &now
	q.ResetAt = now.AddDate(0, 1, 0) // 下个月
}

// ShouldReset 检查是否应该重置
func (q *TrafficQuota) ShouldReset() bool {
	return time.Now().After(q.ResetAt)
}

// GetUsagePercent 获取使用百分比
func (q *TrafficQuota) GetUsagePercent() float64 {
	if q.Quota <= 0 {
		return 0
	}
	return float64(q.Used) / float64(q.Quota) * 100
}

// IsNearLimit 检查是否接近限制（80%）
func (q *TrafficQuota) IsNearLimit() bool {
	return q.GetUsagePercent() >= 80.0
}

// Validate 验证流量记录
func (r *TrafficRecord) Validate() error {
	if r.UserID == 0 {
		return errors.New("用户 ID 不能为空")
	}
	if r.TrafficIn < 0 || r.TrafficOut < 0 {
		return ErrInvalidTraffic
	}
	return nil
}

// GetTotal 获取总流量
func (r *TrafficRecord) GetTotal() int64 {
	return r.TrafficIn + r.TrafficOut
}
