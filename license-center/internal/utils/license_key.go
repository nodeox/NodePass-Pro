package utils

import (
	"crypto/rand"
	"strings"
)

const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// GenerateLicenseKey 生成授权码。
func GenerateLicenseKey(prefix string) string {
	if strings.TrimSpace(prefix) == "" {
		prefix = "NP"
	}
	segments := []int{4, 4, 4}
	parts := make([]string, 0, len(segments)+1)
	parts = append(parts, strings.ToUpper(prefix))
	for _, size := range segments {
		parts = append(parts, randomSegment(size))
	}
	return strings.Join(parts, "-")
}

func randomSegment(size int) string {
	if size <= 0 {
		return ""
	}
	buf := make([]byte, size)
	_, _ = rand.Read(buf)
	out := make([]byte, size)
	for i, b := range buf {
		out[i] = charset[int(b)%len(charset)]
	}
	return string(out)
}
