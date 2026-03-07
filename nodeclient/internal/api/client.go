package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"nodepass-pro/nodeclient/internal/config"
)

// APIResponse 是与面板通信使用的统一响应结构。
type APIResponse[T any] struct {
	Success   bool   `json:"success"`
	Data      T      `json:"data"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// APIErrorBody 表示 API 错误详情。
type APIErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// APIErrorResponse 表示 API 错误响应结构。
type APIErrorResponse struct {
	Success   bool         `json:"success"`
	Error     APIErrorBody `json:"error"`
	Timestamp string       `json:"timestamp"`
}

// HostPort 表示主机端口对。
type HostPort = config.HostPort

// RuleConfig 表示节点侧单条规则配置。
type RuleConfig = config.RuleConfig

// Settings 表示节点运行设置。
type Settings = config.Settings

// NodeConfig 表示完整节点配置。
type NodeConfig = config.NodeConfig

// RegisterRequest 表示节点注册请求。
type RegisterRequest struct {
	Token    string `json:"token"`
	Hostname string `json:"hostname"`
	Version  string `json:"version"`
}

// RegisterResponse 表示节点注册响应。
type RegisterResponse struct {
	NodeID        int         `json:"node_id"`
	ConfigVersion int         `json:"config_version"`
	Config        *NodeConfig `json:"config,omitempty"`
}

// SystemInfo 表示节点系统状态。
type SystemInfo struct {
	CPU          float64 `json:"cpu_usage"`
	Memory       float64 `json:"memory_usage"`
	Disk         float64 `json:"disk_usage"`
	BandwidthIn  int64   `json:"bandwidth_in"`
	BandwidthOut int64   `json:"bandwidth_out"`
	Connections  int64   `json:"connections"`
}

// RuleRuntimeStatus 表示节点上规则实例状态。
type RuleRuntimeStatus struct {
	RuleID int    `json:"rule_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// HeartbeatRequest 表示节点心跳请求。
type HeartbeatRequest struct {
	Token         string              `json:"token,omitempty"`
	ConfigVersion int                 `json:"current_config_version"`
	SystemInfo    SystemInfo          `json:"system_info"`
	RulesStatus   []RuleRuntimeStatus `json:"rules_status"`
}

// HeartbeatResponse 表示节点心跳响应。
type HeartbeatResponse struct {
	ConfigUpdated    bool        `json:"config_updated"`
	NewConfigVersion int         `json:"new_config_version"`
	Config           *NodeConfig `json:"config,omitempty"`
}

// TrafficReport 表示单条流量上报记录。
type TrafficReport struct {
	RuleID     int    `json:"rule_id"`
	TrafficIn  int64  `json:"traffic_in"`
	TrafficOut int64  `json:"traffic_out"`
	Timestamp  string `json:"timestamp,omitempty"`
}

// TrafficReportRequest 表示批量流量上报请求。
type TrafficReportRequest struct {
	Token   string          `json:"token"`
	Records []TrafficReport `json:"records"`
}

// Client 定义面板 HTTP 客户端。
type Client struct {
	hubURL     string
	token      string
	httpClient *http.Client
}

// NewClient 创建面板 API 客户端。
func NewClient(hubURL string, token string) *Client {
	return &Client{
		hubURL: strings.TrimRight(hubURL, "/"),
		token:  strings.TrimSpace(token),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Register 向面板注册节点并获取初始配置。
func (c *Client) Register(hostname string, version string) (*RegisterResponse, error) {
	req := RegisterRequest{
		Token:    c.token,
		Hostname: hostname,
		Version:  version,
	}

	var data RegisterResponse
	if err := c.doJSONRequest(http.MethodPost, "/api/v1/nodes/register", req, nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// Heartbeat 发送节点心跳并按需接收增量配置。
func (c *Client) Heartbeat(req *HeartbeatRequest) (*HeartbeatResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("heartbeat 请求不能为空")
	}

	payload := *req
	payload.Token = c.token

	var data HeartbeatResponse
	if err := c.doJSONRequest(http.MethodPost, "/api/v1/nodes/heartbeat", payload, nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// PullConfig 主动从面板拉取节点配置。
func (c *Client) PullConfig(nodeID int) (*NodeConfig, error) {
	if nodeID <= 0 {
		return nil, fmt.Errorf("nodeID 必须大于 0")
	}

	path := fmt.Sprintf("/api/v1/nodes/%d/config", nodeID)
	var data NodeConfig
	headers := map[string]string{
		"X-Node-Token": c.token,
	}
	if err := c.doJSONRequest(http.MethodGet, path, nil, headers, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// ReportTraffic 上报流量统计数据。
func (c *Client) ReportTraffic(records []TrafficReport) error {
	if records == nil {
		records = make([]TrafficReport, 0)
	}

	payload := TrafficReportRequest{
		Token:   c.token,
		Records: records,
	}
	return c.doJSONRequest(http.MethodPost, "/api/v1/nodes/traffic/report", payload, nil, nil)
}

func (c *Client) doJSONRequest(
	method string,
	path string,
	payload any,
	headers map[string]string,
	out any,
) error {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("序列化请求失败: %w", err)
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.hubURL+path, body)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		if strings.TrimSpace(value) != "" {
			req.Header.Set(key, value)
		}
	}

	startedAt := time.Now()
	log.Printf("[api] 请求开始 method=%s path=%s", method, path)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[api] 请求失败 method=%s path=%s err=%v duration=%s", method, path, err, time.Since(startedAt))
		return fmt.Errorf("请求面板失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}
	log.Printf("[api] 请求结束 method=%s path=%s status=%d duration=%s", method, path, resp.StatusCode, time.Since(startedAt))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr APIErrorResponse
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			return fmt.Errorf(
				"请求失败(%d): %s (%s)",
				resp.StatusCode,
				apiErr.Error.Message,
				apiErr.Error.Code,
			)
		}
		return fmt.Errorf("请求失败(%d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if out == nil {
		return nil
	}

	wrapper := APIResponse[json.RawMessage]{}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	if !wrapper.Success {
		return fmt.Errorf("接口返回失败: %s", wrapper.Message)
	}
	if len(wrapper.Data) == 0 {
		return nil
	}

	if err := json.Unmarshal(wrapper.Data, out); err != nil {
		return fmt.Errorf("解析响应数据失败: %w", err)
	}
	return nil
}
