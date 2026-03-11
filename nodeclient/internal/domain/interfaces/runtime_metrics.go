package interfaces

// RuntimeMetricsProvider 定义运行时流量统计接口。
type RuntimeMetricsProvider interface {
	SnapshotHeartbeatMetrics() (trafficIn, trafficOut, activeConnections int64)
}
