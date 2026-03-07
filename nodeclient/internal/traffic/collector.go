package traffic

import (
	"log"
	"os"
	"sync"
	"time"

	"nodepass-pro/nodeclient/internal/api"
	"nodepass-pro/nodeclient/internal/nodepass"
)

// Collector 定义流量收集与上报服务。
type Collector struct {
	apiClient *api.Client
	nodePass  *nodepass.Integration
	interval  time.Duration
	stopCh    chan struct{}

	logger   *log.Logger
	stopOnce sync.Once
}

// NewCollector 创建流量收集器。
func NewCollector(
	apiClient *api.Client,
	nodePass *nodepass.Integration,
	interval time.Duration,
	logger *log.Logger,
) *Collector {
	if logger == nil {
		logger = log.New(os.Stdout, "[traffic] ", log.LstdFlags)
	}

	return &Collector{
		apiClient: apiClient,
		nodePass:  nodePass,
		interval:  interval,
		stopCh:    make(chan struct{}),
		logger:    logger,
	}
}

// Start 启动定时收集与上报循环。
func (c *Collector) Start() {
	if c.apiClient == nil || c.nodePass == nil {
		c.logger.Printf("[WARN] 流量收集依赖未就绪，跳过启动")
		return
	}
	if c.interval <= 0 {
		c.logger.Printf("[WARN] 流量上报间隔无效，跳过启动")
		return
	}

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	c.logger.Printf("[INFO] 流量收集服务已启动，间隔=%s", c.interval)
	for {
		select {
		case <-ticker.C:
			c.collectAndReport()
		case <-c.stopCh:
			c.logger.Printf("[INFO] 流量收集服务已停止")
			return
		}
	}
}

// Stop 停止流量收集服务。
func (c *Collector) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
}

func (c *Collector) collectAndReport() {
	stats := c.nodePass.GetTrafficStats()
	if len(stats) == 0 {
		return
	}

	records := make([]api.TrafficReport, 0, len(stats))
	reportedStats := make([]nodepass.TrafficStat, 0, len(stats))
	now := time.Now().UTC().Format(time.RFC3339)

	for _, stat := range stats {
		if stat.DeltaIn <= 0 && stat.DeltaOut <= 0 {
			continue
		}

		records = append(records, api.TrafficReport{
			RuleID:     stat.RuleID,
			TrafficIn:  stat.DeltaIn,
			TrafficOut: stat.DeltaOut,
			Timestamp:  now,
		})
		reportedStats = append(reportedStats, stat)
	}

	if len(records) == 0 {
		return
	}

	if err := c.apiClient.ReportTraffic(records); err != nil {
		c.logger.Printf("[WARN] 流量上报失败: %v (数据将在下次上报时累计)", err)
		return
	}

	c.nodePass.MarkTrafficReported(reportedStats)
}
