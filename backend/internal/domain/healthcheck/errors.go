package healthcheck

import "errors"

var (
	// ErrHealthCheckNotFound 健康检查配置不存在
	ErrHealthCheckNotFound = errors.New("健康检查配置不存在")

	// ErrHealthCheckAlreadyExists 健康检查配置已存在
	ErrHealthCheckAlreadyExists = errors.New("健康检查配置已存在")

	// ErrInvalidCheckType 无效的检查类型
	ErrInvalidCheckType = errors.New("无效的检查类型")

	// ErrNodeInstanceNotFound 节点实例不存在
	ErrNodeInstanceNotFound = errors.New("节点实例不存在")

	// ErrInvalidConfiguration 无效的配置
	ErrInvalidConfiguration = errors.New("无效的配置")
)
