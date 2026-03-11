package tunnel

import (
	"errors"
	"time"
)

// Tunnel 隧道实体
type Tunnel struct {
	ID          uint
	UserID      uint
	Name        string
	Description string
	
	// 规则配置
	Protocol    string
	Mode        string
	ListenHost  string
	ListenPort  int
	TargetHost  string
	TargetPort  int
	
	// 节点配置
	EntryNodeID uint
	ExitNodeID  uint
	
	// 状态
	Status      string
	IsEnabled   bool
	
	// 流量统计
	TrafficIn   int64
	TrafficOut  int64
	
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// 领域错误
var (
	ErrTunnelNotFound     = errors.New("隧道不存在")
	ErrTunnelDisabled     = errors.New("隧道已禁用")
	ErrPortConflict       = errors.New("端口冲突")
	ErrInvalidProtocol    = errors.New("不支持的协议")
	ErrInvalidMode        = errors.New("不支持的模式")
	ErrNodeNotAvailable   = errors.New("节点不可用")
	ErrQuotaExceeded      = errors.New("超出配额限制")
)

// IsRunning 检查隧道是否运行中
func (t *Tunnel) IsRunning() bool {
	return t.Status == "running" && t.IsEnabled
}

// CanStart 检查是否可以启动
func (t *Tunnel) CanStart() bool {
	return t.IsEnabled && t.Status != "running"
}

// Start 启动隧道
func (t *Tunnel) Start() error {
	if !t.CanStart() {
		return errors.New("隧道无法启动")
	}
	t.Status = "running"
	return nil
}

// Stop 停止隧道
func (t *Tunnel) Stop() {
	t.Status = "stopped"
}

// Enable 启用隧道
func (t *Tunnel) Enable() {
	t.IsEnabled = true
}

// Disable 禁用隧道
func (t *Tunnel) Disable() {
	t.IsEnabled = false
	t.Status = "stopped"
}

// UpdateTraffic 更新流量统计
func (t *Tunnel) UpdateTraffic(inBytes, outBytes int64) {
	t.TrafficIn += inBytes
	t.TrafficOut += outBytes
}

// Validate 验证隧道配置
func (t *Tunnel) Validate() error {
	// 验证协议
	validProtocols := map[string]bool{"tcp": true, "udp": true, "http": true, "https": true}
	if !validProtocols[t.Protocol] {
		return ErrInvalidProtocol
	}
	
	// 验证模式
	validModes := map[string]bool{"single": true, "relay": true}
	if !validModes[t.Mode] {
		return ErrInvalidMode
	}
	
	// 验证端口
	if t.ListenPort <= 0 || t.ListenPort > 65535 {
		return errors.New("监听端口无效")
	}
	if t.TargetPort <= 0 || t.TargetPort > 65535 {
		return errors.New("目标端口无效")
	}
	
	return nil
}
