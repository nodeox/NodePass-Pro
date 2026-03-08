package middleware

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// IPWhitelistConfig IP 白名单配置
type IPWhitelistConfig struct {
	AllowedIPs   []string
	AllowedCIDRs []*net.IPNet
	SkipPaths    []string
}

// NewIPWhitelistConfig 创建 IP 白名单配置
func NewIPWhitelistConfig(ips []string, cidrs []string) (*IPWhitelistConfig, error) {
	config := &IPWhitelistConfig{
		AllowedIPs:   ips,
		AllowedCIDRs: make([]*net.IPNet, 0),
	}

	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, err
		}
		config.AllowedCIDRs = append(config.AllowedCIDRs, ipNet)
	}

	return config, nil
}

// IsAllowed 检查 IP 是否允许
func (c *IPWhitelistConfig) IsAllowed(ip string) bool {
	// 如果没有配置白名单，则允许所有 IP
	if len(c.AllowedIPs) == 0 && len(c.AllowedCIDRs) == 0 {
		return true
	}

	// 检查精确匹配
	for _, allowedIP := range c.AllowedIPs {
		if ip == allowedIP {
			return true
		}
	}

	// 检查 CIDR 匹配
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, ipNet := range c.AllowedCIDRs {
		if ipNet.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// IPWhitelistMiddleware IP 白名单中间件
func IPWhitelistMiddleware(config *IPWhitelistConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过指定路径
		for _, path := range config.SkipPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		clientIP := c.ClientIP()
		// 处理可能的 IPv6 格式
		if strings.Contains(clientIP, ":") {
			if host, _, err := net.SplitHostPort(clientIP); err == nil {
				clientIP = host
			}
		}

		if !config.IsAllowed(clientIP) {
			c.JSON(http.StatusForbidden, gin.H{
				"success":   false,
				"message":   "IP 地址不在白名单中",
				"timestamp": time.Now().UTC(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
