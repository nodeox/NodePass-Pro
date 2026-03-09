package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/models"

	"gopkg.in/yaml.v3"
)

// TunnelExportFormat 导出格式。
type TunnelExportFormat string

const (
	TunnelExportFormatJSON TunnelExportFormat = "json"
	TunnelExportFormatYAML TunnelExportFormat = "yaml"
)

// TunnelExportData 隧道导出数据。
type TunnelExportData struct {
	Version   string                `json:"version" yaml:"version"`
	ExportAt  time.Time             `json:"export_at" yaml:"export_at"`
	Tunnels   []TunnelExportItem    `json:"tunnels" yaml:"tunnels"`
}

// TunnelExportItem 单个隧道导出项。
type TunnelExportItem struct {
	Name         string               `json:"name" yaml:"name"`
	Description  *string              `json:"description,omitempty" yaml:"description,omitempty"`
	Protocol     string               `json:"protocol" yaml:"protocol"`
	ListenHost   string               `json:"listen_host" yaml:"listen_host"`
	ListenPort   *int                 `json:"listen_port,omitempty" yaml:"listen_port,omitempty"`
	RemoteHost   string               `json:"remote_host" yaml:"remote_host"`
	RemotePort   int                  `json:"remote_port" yaml:"remote_port"`
	Config       *models.TunnelConfig `json:"config,omitempty" yaml:"config,omitempty"`
}

// TunnelImportRequest 隧道导入请求。
type TunnelImportRequest struct {
	Format       TunnelExportFormat `json:"format" binding:"required"`
	Data         string             `json:"data" binding:"required"`
	EntryGroupID uint               `json:"entry_group_id" binding:"required"`
	ExitGroupID  *uint              `json:"exit_group_id"`
	SkipErrors   bool               `json:"skip_errors"`
}

// TunnelImportResult 导入结果。
type TunnelImportResult struct {
	Total     int                    `json:"total"`
	Success   int                    `json:"success"`
	Failed    int                    `json:"failed"`
	Errors    []TunnelImportError    `json:"errors,omitempty"`
	Tunnels   []models.Tunnel        `json:"tunnels"`
}

// TunnelImportError 导入错误。
type TunnelImportError struct {
	Index   int    `json:"index"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// ExportTunnels 导出隧道配置。
func (s *TunnelService) ExportTunnels(userID uint, tunnelIDs []uint, format TunnelExportFormat) (string, error) {
	if s == nil || s.db == nil {
		return "", fmt.Errorf("tunnel service 未初始化")
	}

	if len(tunnelIDs) == 0 {
		return "", fmt.Errorf("%w: 请选择要导出的隧道", ErrInvalidParams)
	}

	query := s.db.Model(&models.Tunnel{})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	query = query.Where("id IN ?", tunnelIDs)

	var tunnels []models.Tunnel
	if err := query.Find(&tunnels).Error; err != nil {
		return "", fmt.Errorf("查询隧道失败: %w", err)
	}

	if len(tunnels) == 0 {
		return "", fmt.Errorf("%w: 未找到可导出的隧道", ErrNotFound)
	}

	exportData := TunnelExportData{
		Version:  "1.0",
		ExportAt: time.Now(),
		Tunnels:  make([]TunnelExportItem, 0, len(tunnels)),
	}

	for _, tunnel := range tunnels {
		config, err := tunnel.GetConfig()
		if err != nil {
			return "", fmt.Errorf("解析隧道配置失败: %w", err)
		}

		item := TunnelExportItem{
			Name:        tunnel.Name,
			Description: tunnel.Description,
			Protocol:    tunnel.Protocol,
			ListenHost:  tunnel.ListenHost,
			RemoteHost:  tunnel.RemoteHost,
			RemotePort:  tunnel.RemotePort,
			Config:      config,
		}

		if tunnel.ListenPort > 0 {
			item.ListenPort = &tunnel.ListenPort
		}

		exportData.Tunnels = append(exportData.Tunnels, item)
	}

	switch format {
	case TunnelExportFormatJSON:
		data, err := json.MarshalIndent(exportData, "", "  ")
		if err != nil {
			return "", fmt.Errorf("序列化 JSON 失败: %w", err)
		}
		return string(data), nil

	case TunnelExportFormatYAML:
		data, err := yaml.Marshal(exportData)
		if err != nil {
			return "", fmt.Errorf("序列化 YAML 失败: %w", err)
		}
		return string(data), nil

	default:
		return "", fmt.Errorf("%w: 不支持的导出格式: %s", ErrInvalidParams, format)
	}
}

// ImportTunnels 导入隧道配置。
func (s *TunnelService) ImportTunnels(userID uint, req *TunnelImportRequest) (*TunnelImportResult, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("tunnel service 未初始化")
	}
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	data := strings.TrimSpace(req.Data)
	if data == "" {
		return nil, fmt.Errorf("%w: 导入数据不能为空", ErrInvalidParams)
	}

	var exportData TunnelExportData
	var err error

	switch req.Format {
	case TunnelExportFormatJSON:
		err = json.Unmarshal([]byte(data), &exportData)
	case TunnelExportFormatYAML:
		err = yaml.Unmarshal([]byte(data), &exportData)
	default:
		return nil, fmt.Errorf("%w: 不支持的导入格式: %s", ErrInvalidParams, req.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("解析导入数据失败: %w", err)
	}

	if len(exportData.Tunnels) == 0 {
		return nil, fmt.Errorf("%w: 导入数据中没有隧道配置", ErrInvalidParams)
	}

	result := &TunnelImportResult{
		Total:   len(exportData.Tunnels),
		Success: 0,
		Failed:  0,
		Errors:  make([]TunnelImportError, 0),
		Tunnels: make([]models.Tunnel, 0),
	}

	for i, item := range exportData.Tunnels {
		createReq := &CreateTunnelRequest{
			Name:         item.Name,
			Description:  item.Description,
			EntryGroupID: req.EntryGroupID,
			ExitGroupID:  req.ExitGroupID,
			Protocol:     item.Protocol,
			ListenHost:   &item.ListenHost,
			ListenPort:   item.ListenPort,
			RemoteHost:   item.RemoteHost,
			RemotePort:   item.RemotePort,
			Config:       item.Config,
		}

		tunnel, createErr := s.Create(userID, createReq)
		if createErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, TunnelImportError{
				Index:   i,
				Name:    item.Name,
				Message: createErr.Error(),
			})

			if !req.SkipErrors {
				return result, fmt.Errorf("导入第 %d 个隧道失败: %w", i+1, createErr)
			}
			continue
		}

		result.Success++
		result.Tunnels = append(result.Tunnels, *tunnel)
	}

	return result, nil
}

// BatchExportAllTunnels 批量导出所有隧道。
func (s *TunnelService) BatchExportAllTunnels(userID uint, format TunnelExportFormat) (string, error) {
	if s == nil || s.db == nil {
		return "", fmt.Errorf("tunnel service 未初始化")
	}

	query := s.db.Model(&models.Tunnel{})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	var tunnels []models.Tunnel
	if err := query.Find(&tunnels).Error; err != nil {
		return "", fmt.Errorf("查询隧道失败: %w", err)
	}

	if len(tunnels) == 0 {
		return "", fmt.Errorf("%w: 没有可导出的隧道", ErrNotFound)
	}

	tunnelIDs := make([]uint, len(tunnels))
	for i, tunnel := range tunnels {
		tunnelIDs[i] = tunnel.ID
	}

	return s.ExportTunnels(userID, tunnelIDs, format)
}
