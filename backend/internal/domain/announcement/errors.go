package announcement

import "errors"

var (
	// ErrAnnouncementNotFound 公告不存在
	ErrAnnouncementNotFound = errors.New("公告不存在")

	// ErrInvalidTitle 无效的标题
	ErrInvalidTitle = errors.New("无效的标题")

	// ErrInvalidContent 无效的内容
	ErrInvalidContent = errors.New("无效的内容")

	// ErrInvalidType 无效的类型
	ErrInvalidType = errors.New("无效的类型")

	// ErrInvalidTimeRange 无效的时间范围
	ErrInvalidTimeRange = errors.New("结束时间不能早于开始时间")

	// ErrUnauthorized 未授权
	ErrUnauthorized = errors.New("未授权操作")
)
