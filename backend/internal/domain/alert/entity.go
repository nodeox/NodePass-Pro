package alert

import "time"

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelError    AlertLevel = "error"
	AlertLevelCritical AlertLevel = "critical"
)

// AlertStatus 告警状态
type AlertStatus string

const (
	AlertStatusPending      AlertStatus = "pending"
	AlertStatusFiring       AlertStatus = "firing"
	AlertStatusResolved     AlertStatus = "resolved"
	AlertStatusSilenced     AlertStatus = "silenced"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
)

// Alert 告警聚合根
type Alert struct {
	ID              uint
	Type            string
	Level           AlertLevel
	Status          AlertStatus
	Title           string
	Message         string
	Fingerprint     string
	ResourceType    string
	ResourceID      uint
	ResourceName    string
	Value           string
	Threshold       string
	FirstFiredAt    time.Time
	LastFiredAt     time.Time
	ResolvedAt      *time.Time
	AcknowledgedAt  *time.Time
	SilencedUntil   *time.Time
	AcknowledgedBy  uint
	ResolvedBy      uint
	Notes           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewAlert 创建告警
func NewAlert(alertType, title, message string, level AlertLevel, resourceType string, resourceID uint) *Alert {
	now := time.Now()
	return &Alert{
		Type:         alertType,
		Level:        level,
		Status:       AlertStatusFiring,
		Title:        title,
		Message:      message,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		FirstFiredAt: now,
		LastFiredAt:  now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// IsFiring 是否正在触发
func (a *Alert) IsFiring() bool {
	return a.Status == AlertStatusFiring
}

// IsResolved 是否已解决
func (a *Alert) IsResolved() bool {
	return a.Status == AlertStatusResolved
}

// IsSilenced 是否已静默
func (a *Alert) IsSilenced() bool {
	if a.Status == AlertStatusSilenced && a.SilencedUntil != nil {
		return time.Now().Before(*a.SilencedUntil)
	}
	return false
}

// Resolve 解决告警
func (a *Alert) Resolve(resolvedBy uint, notes string) error {
	if a.IsResolved() {
		return ErrAlertAlreadyResolved
	}

	now := time.Now()
	a.Status = AlertStatusResolved
	a.ResolvedAt = &now
	a.ResolvedBy = resolvedBy
	a.Notes = notes
	a.UpdatedAt = now

	return nil
}

// Acknowledge 确认告警
func (a *Alert) Acknowledge(acknowledgedBy uint) {
	now := time.Now()
	a.Status = AlertStatusAcknowledged
	a.AcknowledgedAt = &now
	a.AcknowledgedBy = acknowledgedBy
	a.UpdatedAt = now
}

// Silence 静默告警
func (a *Alert) Silence(duration time.Duration) {
	silencedUntil := time.Now().Add(duration)
	a.Status = AlertStatusSilenced
	a.SilencedUntil = &silencedUntil
	a.UpdatedAt = time.Now()
}

// Fire 触发告警
func (a *Alert) Fire(value string) {
	now := time.Now()
	a.Status = AlertStatusFiring
	a.LastFiredAt = now
	a.Value = value
	a.UpdatedAt = now
}
