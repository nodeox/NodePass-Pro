package tunneltemplate

import "errors"

var (
	// ErrTemplateNotFound 模板不存在
	ErrTemplateNotFound = errors.New("模板不存在")

	// ErrTemplateAlreadyExists 模板已存在
	ErrTemplateAlreadyExists = errors.New("模板已存在")

	// ErrInvalidTemplateName 无效的模板名称
	ErrInvalidTemplateName = errors.New("无效的模板名称")

	// ErrInvalidProtocol 无效的协议
	ErrInvalidProtocol = errors.New("无效的协议")

	// ErrInvalidConfig 无效的配置
	ErrInvalidConfig = errors.New("无效的配置")

	// ErrUnauthorized 未授权
	ErrUnauthorized = errors.New("未授权访问此模板")
)
