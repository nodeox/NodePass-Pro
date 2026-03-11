package interfaces

// MetricsSink 定义 Prometheus 指标收集接口。
type MetricsSink interface {
	RecordHeartbeatAttempt()
	RecordHeartbeatSuccess(timestamp float64)
	RecordHeartbeatFailure()
	SetConfigVersion(version int)
	SetRuleStats(total, running, stopped, errored int)
	RecordRuleRestart(ruleID, mode string)
	SetTrafficStats(ruleID string, inBytes, outBytes int64)
	SetActiveConnections(ruleID string, count int64)
	SetSystemStats(cpuUsage, memoryUsage, diskUsage float64, bandwidthIn, bandwidthOut int64)
}
