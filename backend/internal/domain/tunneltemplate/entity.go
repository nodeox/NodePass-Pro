package tunneltemplate

import (
	"time"
)

// TunnelTemplate 隧道模板聚合根
type TunnelTemplate struct {
	ID          uint
	UserID      uint
	Name        string
	Description *string
	Protocol    string
	Config      *TemplateConfig
	IsPublic    bool
	UsageCount  int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TemplateConfig 模板配置
type TemplateConfig struct {
	ListenHost          *string
	ListenPort          *int
	RemoteHost          string
	RemotePort          int
	LoadBalanceStrategy string
	IPType              string
	EnableProxyProtocol bool
	ForwardTargets      []ForwardTarget
	HealthCheckInterval int
	HealthCheckTimeout  int
	ProtocolConfig      map[string]interface{}
}

// ForwardTarget 转发目标
type ForwardTarget struct {
	Host   string
	Port   int
	Weight int
}

// NewTunnelTemplate 创建隧道模板
func NewTunnelTemplate(userID uint, name, protocol string, config *TemplateConfig) *TunnelTemplate {
	return &TunnelTemplate{
		UserID:     userID,
		Name:       name,
		Protocol:   protocol,
		Config:     config,
		IsPublic:   false,
		UsageCount: 0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// IsOwnedBy 检查是否属于指定用户
func (t *TunnelTemplate) IsOwnedBy(userID uint) bool {
	return t.UserID == userID
}

// CanBeAccessedBy 检查是否可以被指定用户访问
func (t *TunnelTemplate) CanBeAccessedBy(userID uint) bool {
	return t.IsOwnedBy(userID) || t.IsPublic
}

// MakePublic 设为公开
func (t *TunnelTemplate) MakePublic() {
	t.IsPublic = true
	t.UpdatedAt = time.Now()
}

// MakePrivate 设为私有
func (t *TunnelTemplate) MakePrivate() {
	t.IsPublic = false
	t.UpdatedAt = time.Now()
}

// UpdateInfo 更新基本信息
func (t *TunnelTemplate) UpdateInfo(name string, description *string) {
	t.Name = name
	t.Description = description
	t.UpdatedAt = time.Now()
}

// UpdateConfig 更新配置
func (t *TunnelTemplate) UpdateConfig(config *TemplateConfig) {
	t.Config = config
	t.UpdatedAt = time.Now()
}

// IncrementUsage 增加使用次数
func (t *TunnelTemplate) IncrementUsage() {
	t.UsageCount++
	t.UpdatedAt = time.Now()
}

// ListFilter 列表过滤条件
type ListFilter struct {
	UserID   uint
	Protocol *string
	IsPublic *bool
	Page     int
	PageSize int
}
