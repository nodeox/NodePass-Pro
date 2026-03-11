package alert

import "errors"

var (
	// ErrAlertNotFound 告警不存在
	ErrAlertNotFound = errors.New("告警不存在")

	// ErrAlertRuleNotFound 告警规则不存在
	ErrAlertRuleNotFound = errors.New("告警规则不存在")

	// ErrInvalidAlertLevel 无效的告警级别
	ErrInvalidAlertLevel = errors.New("无效的告警级别")

	// ErrInvalidAlertType 无效的告警类型
	ErrInvalidAlertType = errors.New("无效的告警类型")

	// ErrAlertAlreadyResolved 告警已解决
	ErrAlertAlreadyResolved = errors.New("告警已解决")
)
