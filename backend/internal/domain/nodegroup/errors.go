package nodegroup

import "errors"

var (
	// ErrNodeGroupNotFound 节点组不存在
	ErrNodeGroupNotFound = errors.New("节点组不存在")

	// ErrNodeGroupNameExists 节点组名称已存在
	ErrNodeGroupNameExists = errors.New("节点组名称已存在")

	// ErrInvalidNodeGroupType 无效的节点组类型
	ErrInvalidNodeGroupType = errors.New("无效的节点组类型")

	// ErrInvalidPortRange 无效的端口范围
	ErrInvalidPortRange = errors.New("无效的端口范围")

	// ErrNodeGroupDisabled 节点组已禁用
	ErrNodeGroupDisabled = errors.New("节点组已禁用")
)
