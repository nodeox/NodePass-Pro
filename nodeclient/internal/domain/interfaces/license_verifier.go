package interfaces

// VerifyStatus 表示 nodeclient 授权校验状态。
type VerifyStatus struct {
	Allowed       bool
	Message       string
	Status        string
	LicenseID     uint
	Plan          string
	Customer      string
	ExpiresAt     string
	VersionStatus string
}

// LicenseVerifier 定义授权校验接口。
type LicenseVerifier interface {
	Enabled() bool
	Verify(currentVersion string) (*VerifyStatus, error)
}
