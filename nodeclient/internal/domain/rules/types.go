package rules

// RuleStatus 表示单条规则实例状态。
type RuleStatus struct {
	RuleID int    `json:"rule_id"`
	Mode   string `json:"mode"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// TrafficStat 表示单条规则流量统计。
type TrafficStat struct {
	RuleID     int   `json:"rule_id"`
	TrafficIn  int64 `json:"traffic_in"`
	TrafficOut int64 `json:"traffic_out"`
	DeltaIn    int64 `json:"delta_in"`
	DeltaOut   int64 `json:"delta_out"`
}
