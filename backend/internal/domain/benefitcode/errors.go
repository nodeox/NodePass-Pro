package benefitcode

import "errors"

var (
	// ErrBenefitCodeNotFound 权益码不存在
	ErrBenefitCodeNotFound = errors.New("权益码不存在")

	// ErrBenefitCodeAlreadyExists 权益码已存在
	ErrBenefitCodeAlreadyExists = errors.New("权益码已存在")

	// ErrBenefitCodeInvalid 权益码无效
	ErrBenefitCodeInvalid = errors.New("权益码无效")

	// ErrBenefitCodeExpired 权益码已过期
	ErrBenefitCodeExpired = errors.New("权益码已过期")

	// ErrBenefitCodeAlreadyUsed 权益码已使用
	ErrBenefitCodeAlreadyUsed = errors.New("权益码已使用")

	// ErrBenefitCodeAlreadyRevoked 权益码已撤销
	ErrBenefitCodeAlreadyRevoked = errors.New("权益码已撤销")

	// ErrBenefitCodeDisabled 权益码已禁用
	ErrBenefitCodeDisabled = errors.New("权益码已禁用")

	// ErrInvalidVIPLevel 无效的 VIP 等级
	ErrInvalidVIPLevel = errors.New("无效的 VIP 等级")

	// ErrInvalidDuration 无效的时长
	ErrInvalidDuration = errors.New("无效的时长")

	// ErrInvalidCount 无效的数量
	ErrInvalidCount = errors.New("无效的数量")
)
