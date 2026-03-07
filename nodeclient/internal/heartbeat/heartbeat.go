package heartbeat

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	gnet "github.com/shirou/gopsutil/v3/net"
	"nodepass-pro/nodeclient/internal/api"
)

// AgentInterface 定义心跳服务依赖的 Agent 能力。
type AgentInterface interface {
	GetRulesStatus() []api.RuleRuntimeStatus
	GetConfigVersion() int
	SetOnline(online bool)
	ApplyNewConfig(cfg *api.NodeConfig)
}

// Service 定义心跳上报服务。
type Service struct {
	apiClient *api.Client
	agent     AgentInterface
	interval  time.Duration
	stopCh    chan struct{}

	logger   *log.Logger
	stopOnce sync.Once
}

// NewService 创建心跳服务。
func NewService(
	apiClient *api.Client,
	agent AgentInterface,
	interval time.Duration,
	logger *log.Logger,
) *Service {
	if logger == nil {
		logger = log.New(os.Stdout, "[heartbeat] ", log.LstdFlags)
	}
	return &Service{
		apiClient: apiClient,
		agent:     agent,
		interval:  interval,
		stopCh:    make(chan struct{}),
		logger:    logger,
	}
}

// Start 启动心跳循环。
func (s *Service) Start() {
	if s.apiClient == nil || s.agent == nil {
		s.logger.Printf("[WARN] 心跳服务依赖未就绪，跳过启动")
		return
	}
	if s.interval <= 0 {
		s.logger.Printf("[WARN] 心跳间隔无效，跳过启动")
		return
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.logger.Printf("[INFO] 心跳服务已启动，间隔=%s", s.interval)
	for {
		select {
		case <-ticker.C:
			s.sendHeartbeat()
		case <-s.stopCh:
			s.logger.Printf("[INFO] 心跳服务已停止")
			return
		}
	}
}

// Stop 停止心跳服务。
func (s *Service) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

func (s *Service) sendHeartbeat() {
	sysInfo := collectSystemInfo()
	rulesStatus := s.agent.GetRulesStatus()

	resp, err := s.apiClient.Heartbeat(&api.HeartbeatRequest{
		ConfigVersion: s.agent.GetConfigVersion(),
		SystemInfo:    sysInfo,
		RulesStatus:   rulesStatus,
	})
	if err != nil {
		s.logger.Printf("[WARN] 心跳发送失败: %v (继续运行，不影响现有规则)", err)
		s.agent.SetOnline(false)
		return
	}

	s.agent.SetOnline(true)

	if resp == nil || !resp.ConfigUpdated {
		return
	}

	s.logger.Printf("[INFO] 收到新配置 v%d", resp.NewConfigVersion)
	if resp.Config == nil {
		s.logger.Printf("[WARN] 心跳返回配置更新标记，但未携带配置内容")
		return
	}
	s.agent.ApplyNewConfig(resp.Config)
}

func collectSystemInfo() api.SystemInfo {
	info := api.SystemInfo{}

	if cpuPercent, err := cpu.Percent(0, false); err == nil && len(cpuPercent) > 0 {
		info.CPU = cpuPercent[0]
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		info.Memory = vm.UsedPercent
	}

	if usage, err := disk.Usage("/"); err == nil {
		info.Disk = usage.UsedPercent
	}

	if ioCounters, err := gnet.IOCounters(false); err == nil && len(ioCounters) > 0 {
		info.BandwidthIn = int64(ioCounters[0].BytesRecv)
		info.BandwidthOut = int64(ioCounters[0].BytesSent)
	}

	if conns, err := gnet.Connections("all"); err == nil {
		info.Connections = int64(len(conns))
	}

	return info
}
