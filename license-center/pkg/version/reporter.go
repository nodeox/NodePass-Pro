package version

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"
)

// BuildInfo 构建信息（编译时注入）
var (
	Version   = "dev"
	GitCommit = "unknown"
	GitBranch = "unknown"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
)

// ComponentType 组件类型
type ComponentType string

const (
	ComponentBackend       ComponentType = "backend"
	ComponentFrontend      ComponentType = "frontend"
	ComponentNodeClient    ComponentType = "node_client"
	ComponentLicenseCenter ComponentType = "license_center"
)

// ReportRequest 版本上报请求
type ReportRequest struct {
	Component   ComponentType `json:"component"`
	Version     string        `json:"version"`
	GitCommit   string        `json:"git_commit,omitempty"`
	GitBranch   string        `json:"git_branch,omitempty"`
	BuildTime   string        `json:"build_time,omitempty"`
	Description string        `json:"description,omitempty"`
}

// Reporter 版本上报器
type Reporter struct {
	LicenseCenterURL string
	Component        ComponentType
	Token            string
	client           *http.Client
}

// NewReporter 创建版本上报器
func NewReporter(licenseCenterURL string, component ComponentType, token string) *Reporter {
	return &Reporter{
		LicenseCenterURL: licenseCenterURL,
		Component:        component,
		Token:            token,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Report 上报版本信息
func (r *Reporter) Report() error {
	req := ReportRequest{
		Component:   r.Component,
		Version:     Version,
		GitCommit:   GitCommit,
		GitBranch:   GitBranch,
		BuildTime:   BuildTime,
		Description: fmt.Sprintf("Auto-reported from %s", r.Component),
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request failed: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/versions/components", r.LicenseCenterURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if r.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+r.Token)
	}

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("report failed with status: %d", resp.StatusCode)
	}

	return nil
}

// ReportAsync 异步上报版本信息（不阻塞启动）
func (r *Reporter) ReportAsync() {
	go func() {
		// 延迟上报，等待服务完全启动
		time.Sleep(5 * time.Second)

		if err := r.Report(); err != nil {
			// 只记录日志，不影响服务启动
			fmt.Fprintf(os.Stderr, "Failed to report version: %v\n", err)
		} else {
			fmt.Printf("Version reported successfully: %s %s\n", r.Component, Version)
		}
	}()
}

// GetInfo 获取版本信息
func GetInfo() map[string]string {
	return map[string]string{
		"version":    Version,
		"git_commit": GitCommit,
		"git_branch": GitBranch,
		"build_time": BuildTime,
		"go_version": GoVersion,
	}
}

// PrintInfo 打印版本信息
func PrintInfo() {
	fmt.Printf("Version:    %s\n", Version)
	fmt.Printf("Git Commit: %s\n", GitCommit)
	fmt.Printf("Git Branch: %s\n", GitBranch)
	fmt.Printf("Build Time: %s\n", BuildTime)
	fmt.Printf("Go Version: %s\n", GoVersion)
}
