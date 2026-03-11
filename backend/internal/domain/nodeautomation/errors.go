package nodeautomation

import "errors"

var (
	// ErrPolicyNotFound 策略不存在
	ErrPolicyNotFound = errors.New("自动化策略不存在")

	// ErrActionNotFound 操作记录不存在
	ErrActionNotFound = errors.New("自动化操作记录不存在")

	// ErrIsolationNotFound 隔离记录不存在
	ErrIsolationNotFound = errors.New("节点隔离记录不存在")

	// ErrInvalidPolicy 无效的策略
	ErrInvalidPolicy = errors.New("无效的自动化策略")
)
