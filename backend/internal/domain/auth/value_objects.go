package auth

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	emailRegex           = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	usernameRegex        = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
	passwordLowerRegex   = regexp.MustCompile(`[a-z]`)
	passwordUpperRegex   = regexp.MustCompile(`[A-Z]`)
	passwordDigitRegex   = regexp.MustCompile(`[0-9]`)
	passwordSpecialRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)
)

// Email 邮箱值对象
type Email struct {
	value string
}

// NewEmail 创建邮箱值对象
func NewEmail(email string) (*Email, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, fmt.Errorf("邮箱不能为空")
	}
	if len(email) > 254 {
		return nil, fmt.Errorf("邮箱长度不能超过 254 个字符")
	}
	if !emailRegex.MatchString(email) {
		return nil, fmt.Errorf("邮箱格式无效")
	}
	return &Email{value: email}, nil
}

// String 返回邮箱字符串
func (e *Email) String() string {
	return e.value
}

// Username 用户名值对象
type Username struct {
	value string
}

// NewUsername 创建用户名值对象
func NewUsername(username string) (*Username, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}
	if len(username) < 3 {
		return nil, fmt.Errorf("用户名长度不能少于 3 个字符")
	}
	if len(username) > 32 {
		return nil, fmt.Errorf("用户名长度不能超过 32 个字符")
	}
	if !usernameRegex.MatchString(username) {
		return nil, fmt.Errorf("用户名只能包含字母、数字、下划线、连字符")
	}
	return &Username{value: username}, nil
}

// String 返回用户名字符串
func (u *Username) String() string {
	return u.value
}

// Password 密码值对象
type Password struct {
	value string
}

// NewPassword 创建密码值对象（验证强度）
func NewPassword(password string) (*Password, error) {
	if len(password) < 8 {
		return nil, fmt.Errorf("密码长度不能少于 8 个字符")
	}
	if len(password) > 128 {
		return nil, fmt.Errorf("密码长度不能超过 128 个字符")
	}
	if !passwordLowerRegex.MatchString(password) {
		return nil, fmt.Errorf("密码必须包含至少一个小写字母")
	}
	if !passwordUpperRegex.MatchString(password) {
		return nil, fmt.Errorf("密码必须包含至少一个大写字母")
	}
	if !passwordDigitRegex.MatchString(password) {
		return nil, fmt.Errorf("密码必须包含至少一个数字")
	}
	if !passwordSpecialRegex.MatchString(password) {
		return nil, fmt.Errorf("密码必须包含至少一个特殊字符")
	}
	return &Password{value: password}, nil
}

// String 返回密码字符串
func (p *Password) String() string {
	return p.value
}
