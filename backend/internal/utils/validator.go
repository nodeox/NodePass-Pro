package utils

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

var (
	// 域名正则表达式
	domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	// 邮箱正则表达式
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// ValidatePort 验证端口号是否有效（1-65535）。
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("端口号必须在 1-65535 之间")
	}
	return nil
}

// ValidateIP 验证 IP 地址是否有效（支持 IPv4 和 IPv6）。
func ValidateIP(ip string) error {
	if strings.TrimSpace(ip) == "" {
		return fmt.Errorf("IP 地址不能为空")
	}
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("无效的 IP 地址: %s", ip)
	}
	return nil
}

// ValidateDomain 验证域名是否有效。
func ValidateDomain(domain string) error {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return fmt.Errorf("域名不能为空")
	}
	if len(domain) > 253 {
		return fmt.Errorf("域名长度不能超过 253 个字符")
	}
	if !domainRegex.MatchString(domain) {
		return fmt.Errorf("无效的域名格式: %s", domain)
	}
	return nil
}

// ValidateHost 验证主机地址（IP 或域名）。
func ValidateHost(host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("主机地址不能为空")
	}

	// 尝试作为 IP 地址验证
	if net.ParseIP(host) != nil {
		return nil
	}

	// 尝试作为域名验证
	if domainRegex.MatchString(host) {
		return nil
	}

	return fmt.Errorf("无效的主机地址（必须是 IP 或域名）: %s", host)
}

// ValidateEmail 验证邮箱地址是否有效。
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("邮箱地址不能为空")
	}
	if len(email) > 254 {
		return fmt.Errorf("邮箱地址长度不能超过 254 个字符")
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("无效的邮箱格式: %s", email)
	}
	return nil
}

// ValidateUsername 验证用户名是否有效。
// 用户名规则：3-32 个字符，只能包含字母、数字、下划线、连字符。
func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if len(username) < 3 {
		return fmt.Errorf("用户名长度不能少于 3 个字符")
	}
	if len(username) > 32 {
		return fmt.Errorf("用户名长度不能超过 32 个字符")
	}

	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("用户名只能包含字母、数字、下划线、连字符")
	}
	return nil
}

// ValidatePassword 验证密码强度。
// 密码规则：至少 6 个字符。
func ValidatePassword(password string) error {
	if len(password) < 6 {
		return fmt.Errorf("密码长度不能少于 6 个字符")
	}
	if len(password) > 128 {
		return fmt.Errorf("密码长度不能超过 128 个字符")
	}
	return nil
}

// ValidateProtocol 验证协议类型是否有效。
func ValidateProtocol(protocol string) error {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	validProtocols := map[string]bool{
		"tcp":  true,
		"udp":  true,
		"both": true,
	}
	if !validProtocols[protocol] {
		return fmt.Errorf("无效的协议类型: %s（支持 tcp、udp、both）", protocol)
	}
	return nil
}

// ValidateNodeName 验证节点名称是否有效。
func ValidateNodeName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("节点名称不能为空")
	}
	if len(name) > 100 {
		return fmt.Errorf("节点名称长度不能超过 100 个字符")
	}
	return nil
}

// ValidateRuleName 验证规则名称是否有效。
func ValidateRuleName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("规则名称不能为空")
	}
	if len(name) > 100 {
		return fmt.Errorf("规则名称长度不能超过 100 个字符")
	}
	return nil
}
