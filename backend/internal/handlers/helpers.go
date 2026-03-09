package handlers

import "strings"

// coalesceString 返回第一个非空的字符串值
func coalesceString(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// normalizeEmail 规范化邮箱地址（转小写并去除空格）
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
