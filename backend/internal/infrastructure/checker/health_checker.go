package checker

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"nodepass-pro/backend/internal/domain/healthcheck"
	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// HealthChecker 健康检查器实现
type HealthChecker struct {
	db *gorm.DB
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(db *gorm.DB) *HealthChecker {
	return &HealthChecker{db: db}
}

// Check 执行健康检查
func (c *HealthChecker) Check(ctx context.Context, nodeInstanceID uint, config *healthcheck.HealthCheck) (*healthcheck.HealthRecord, error) {
	// 获取节点实例
	var instance models.NodeInstance
	if err := c.db.WithContext(ctx).First(&instance, nodeInstanceID).Error; err != nil {
		return nil, healthcheck.ErrNodeInstanceNotFound
	}

	// 根据检查类型执行检查
	var record *healthcheck.HealthRecord
	switch config.Type {
	case healthcheck.CheckTypeTCP:
		record = c.performTCPCheck(&instance, config)
	case healthcheck.CheckTypeHTTP:
		record = c.performHTTPCheck(&instance, config)
	case healthcheck.CheckTypeICMP:
		record = c.performICMPCheck(&instance, config)
	default:
		record = healthcheck.NewHealthRecord(nodeInstanceID, config.Type, healthcheck.CheckStatusUnknown)
		record.SetError("不支持的健康检查类型")
	}

	// 更新节点状态
	c.updateNodeStatus(&instance, record)

	return record, nil
}

// performTCPCheck 执行 TCP 健康检查
func (c *HealthChecker) performTCPCheck(instance *models.NodeInstance, config *healthcheck.HealthCheck) *healthcheck.HealthRecord {
	record := healthcheck.NewHealthRecord(instance.ID, healthcheck.CheckTypeTCP, healthcheck.CheckStatusUnknown)

	if instance.Host == nil || instance.Port == nil {
		record.Status = healthcheck.CheckStatusUnhealthy
		record.SetError("节点地址或端口未配置")
		return record
	}

	address := fmt.Sprintf("%s:%d", *instance.Host, *instance.Port)
	timeout := time.Duration(config.Timeout) * time.Second

	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, timeout)
	latency := int(time.Since(start).Milliseconds())
	record.SetLatency(latency)

	if err != nil {
		record.Status = healthcheck.CheckStatusUnhealthy
		record.SetError(err.Error())
		return record
	}
	defer conn.Close()

	record.Status = healthcheck.CheckStatusHealthy
	return record
}

// performHTTPCheck 执行 HTTP 健康检查
func (c *HealthChecker) performHTTPCheck(instance *models.NodeInstance, config *healthcheck.HealthCheck) *healthcheck.HealthRecord {
	record := healthcheck.NewHealthRecord(instance.ID, healthcheck.CheckTypeHTTP, healthcheck.CheckStatusUnknown)

	if instance.Host == nil || instance.Port == nil {
		record.Status = healthcheck.CheckStatusUnhealthy
		record.SetError("节点地址或端口未配置")
		return record
	}

	path := "/"
	if config.HTTPPath != nil {
		path = *config.HTTPPath
	}

	method := "GET"
	if config.HTTPMethod != nil {
		method = *config.HTTPMethod
	}

	url := fmt.Sprintf("http://%s:%d%s", *instance.Host, *instance.Port, path)
	timeout := time.Duration(config.Timeout) * time.Second

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		record.Status = healthcheck.CheckStatusUnhealthy
		record.SetError(err.Error())
		return record
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := int(time.Since(start).Milliseconds())
	record.SetLatency(latency)

	if err != nil {
		record.Status = healthcheck.CheckStatusUnhealthy
		record.SetError(err.Error())
		return record
	}
	defer resp.Body.Close()

	// 检查状态码
	expectedStatus := 200
	if config.ExpectedStatus != nil {
		expectedStatus = *config.ExpectedStatus
	}

	if resp.StatusCode == expectedStatus {
		record.Status = healthcheck.CheckStatusHealthy
	} else {
		record.Status = healthcheck.CheckStatusUnhealthy
		record.SetError(fmt.Sprintf("HTTP 状态码不匹配: 期望 %d, 实际 %d", expectedStatus, resp.StatusCode))
	}

	return record
}

// performICMPCheck 执行 ICMP (Ping) 健康检查
func (c *HealthChecker) performICMPCheck(instance *models.NodeInstance, config *healthcheck.HealthCheck) *healthcheck.HealthRecord {
	record := healthcheck.NewHealthRecord(instance.ID, healthcheck.CheckTypeICMP, healthcheck.CheckStatusUnknown)

	if instance.Host == nil {
		record.Status = healthcheck.CheckStatusUnhealthy
		record.SetError("节点地址未配置")
		return record
	}

	// 简化实现：使用 TCP 连接测试代替 ICMP
	// 真实的 ICMP 需要 root 权限，这里用 TCP 80 端口测试
	address := fmt.Sprintf("%s:80", *instance.Host)
	timeout := time.Duration(config.Timeout) * time.Second

	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, timeout)
	latency := int(time.Since(start).Milliseconds())
	record.SetLatency(latency)

	if err != nil {
		record.Status = healthcheck.CheckStatusUnhealthy
		record.SetError(err.Error())
		return record
	}
	defer conn.Close()

	record.Status = healthcheck.CheckStatusHealthy
	return record
}

// updateNodeStatus 根据健康检查结果更新节点状态
func (c *HealthChecker) updateNodeStatus(instance *models.NodeInstance, record *healthcheck.HealthRecord) {
	var newStatus models.NodeInstanceStatus

	if record.IsHealthy() {
		newStatus = models.NodeInstanceStatusOnline
	} else {
		newStatus = models.NodeInstanceStatusOffline
	}

	if instance.Status != newStatus {
		c.db.Model(&models.NodeInstance{}).
			Where("id = ?", instance.ID).
			Update("status", newStatus)
	}
}
