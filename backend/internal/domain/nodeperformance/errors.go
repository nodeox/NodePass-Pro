package nodeperformance

import "errors"

var (
	// ErrMetricNotFound 指标不存在
	ErrMetricNotFound = errors.New("性能指标不存在")

	// ErrAlertNotFound 告警配置不存在
	ErrAlertNotFound = errors.New("告警配置不存在")

	// ErrInvalidMetric 无效的指标
	ErrInvalidMetric = errors.New("无效的性能指标")
)
