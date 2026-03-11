package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"
	panelws "nodepass-pro/backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	deployAssetTypeScript   = "script"
	deployAssetTypeImage    = "image"
	defaultDeployAssetDir   = "./data/deploy-assets"
	maxDeployScriptFileSize = 2 * 1024 * 1024         // 2MB
	maxDeployImageFileSize  = 10 * 1024 * 1024 * 1024 // 10GB
)

var (
	deployAssetFileNameRegex   = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	deployAssetUnsafeCharRegex = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
)

// SystemHandler 系统配置与统计处理器。
type SystemHandler struct {
	systemService *services.SystemService
	hub           *panelws.Hub
}

// NewSystemHandler 创建系统处理器。
func NewSystemHandler(db *gorm.DB, hub *panelws.Hub) *SystemHandler {
	return &SystemHandler{
		systemService: services.NewSystemService(db),
		hub:           hub,
	}
}

// GetConfig GET /api/v1/system/config
func (h *SystemHandler) GetConfig(c *gin.Context) {
	configMap, err := h.systemService.GetConfig()
	if err != nil {
		writeServiceError(c, err, "GET_SYSTEM_CONFIG_FAILED")
		return
	}
	utils.Success(c, configMap)
}

// UpdateConfig PUT /api/v1/system/config
func (h *SystemHandler) UpdateConfig(c *gin.Context) {
	type requestEntry struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	type batchPayload struct {
		Items []requestEntry `json:"items"`
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数读取失败")
		return
	}
	if len(rawBody) == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求体不能为空")
		return
	}

	entries := make([]services.SystemConfigEntry, 0)
	var batchReq batchPayload
	if unmarshalErr := json.Unmarshal(rawBody, &batchReq); unmarshalErr == nil && len(batchReq.Items) > 0 {
		for _, item := range batchReq.Items {
			entries = append(entries, services.SystemConfigEntry{
				Key:   item.Key,
				Value: item.Value,
			})
		}
	} else {
		var listReq []requestEntry
		if unmarshalErr := json.Unmarshal(rawBody, &listReq); unmarshalErr == nil && len(listReq) > 0 {
			for _, item := range listReq {
				entries = append(entries, services.SystemConfigEntry{
					Key:   item.Key,
					Value: item.Value,
				})
			}
		} else {
			var singleReq requestEntry
			if unmarshalErr := json.Unmarshal(rawBody, &singleReq); unmarshalErr != nil {
				utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+unmarshalErr.Error())
				return
			}
			entries = append(entries, services.SystemConfigEntry{
				Key:   singleReq.Key,
				Value: singleReq.Value,
			})
		}
	}

	if err := h.systemService.UpdateConfigs(entries); err != nil {
		writeServiceError(c, err, "UPDATE_SYSTEM_CONFIG_FAILED")
		return
	}

	if h.hub != nil {
		changedKeys := make([]string, 0, len(entries))
		for _, item := range entries {
			key := strings.TrimSpace(item.Key)
			if key != "" {
				changedKeys = append(changedKeys, key)
			}
		}
		_ = h.hub.Broadcast(panelws.MessageTypeConfigUpdated, gin.H{"keys": changedKeys})
	}

	utils.SuccessResponse(c, nil, "系统配置更新成功")
}

// GetStats GET /api/v1/system/stats
func (h *SystemHandler) GetStats(c *gin.Context) {
	stats, err := h.systemService.GetStats()
	if err != nil {
		writeServiceError(c, err, "GET_SYSTEM_STATS_FAILED")
		return
	}
	utils.Success(c, stats)
}

// UploadDeployAsset POST /api/v1/system/deploy-assets/upload
func (h *SystemHandler) UploadDeployAsset(c *gin.Context) {
	assetType := strings.ToLower(strings.TrimSpace(c.PostForm("asset_type")))
	if !isSupportedDeployAssetType(assetType) {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "asset_type 仅支持 script/image")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请上传文件")
		return
	}
	if fileHeader.Size <= 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "上传文件不能为空")
		return
	}
	if fileHeader.Size > maxDeployAssetFileSize(assetType) {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "文件大小超出限制")
		return
	}

	originalName := sanitizeUploadFileName(fileHeader.Filename)
	if !isValidDeployAssetFile(assetType, originalName) {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "文件类型不支持")
		return
	}

	uploadedAt := time.Now().UTC()
	storedName := fmt.Sprintf("%d_%s", uploadedAt.Unix(), originalName)
	baseDir := resolveDeployAssetBaseDir()
	targetDir := filepath.Join(baseDir, assetType)
	if mkErr := os.MkdirAll(targetDir, 0o755); mkErr != nil {
		utils.Error(c, http.StatusInternalServerError, "UPLOAD_FAILED", "创建上传目录失败")
		return
	}

	targetPath := filepath.Join(targetDir, storedName)
	if saveErr := c.SaveUploadedFile(fileHeader, targetPath); saveErr != nil {
		utils.Error(c, http.StatusInternalServerError, "UPLOAD_FAILED", "保存上传文件失败")
		return
	}

	configMap, _ := h.systemService.GetConfig()
	publicBaseURL := normalizePublicBaseURL(configMap["site_url"])
	if publicBaseURL == "" {
		publicBaseURL = normalizePublicBaseURL(inferPanelURL(c))
	}
	downloadPath := fmt.Sprintf("/api/v1/public/deploy-assets/%s/%s", assetType, storedName)
	downloadURL := downloadPath
	if publicBaseURL != "" {
		downloadURL = publicBaseURL + downloadPath
	}

	uploadResponse := gin.H{
		"asset_type":   assetType,
		"file_name":    originalName,
		"stored_name":  storedName,
		"size":         fileHeader.Size,
		"download_url": downloadURL,
		"uploaded_at":  uploadedAt.Format(time.RFC3339),
	}

	configEntries := make([]services.SystemConfigEntry, 0, 4)
	if assetType == deployAssetTypeScript {
		oneClickCommand := fmt.Sprintf("bash <(curl -fsSL '%s')", downloadURL)
		uploadResponse["one_click_command"] = oneClickCommand
		configEntries = append(configEntries,
			services.SystemConfigEntry{Key: "deploy_script_url", Value: downloadURL},
			services.SystemConfigEntry{Key: "deploy_script_name", Value: originalName},
			services.SystemConfigEntry{Key: "deploy_script_updated_at", Value: uploadedAt.Format(time.RFC3339)},
			services.SystemConfigEntry{Key: "deploy_script_command", Value: oneClickCommand},
		)
	} else {
		configEntries = append(configEntries,
			services.SystemConfigEntry{Key: "deploy_image_url", Value: downloadURL},
			services.SystemConfigEntry{Key: "deploy_image_name", Value: originalName},
			services.SystemConfigEntry{Key: "deploy_image_updated_at", Value: uploadedAt.Format(time.RFC3339)},
		)
	}

	if updateErr := h.systemService.UpdateConfigs(configEntries); updateErr != nil {
		writeServiceError(c, updateErr, "UPDATE_SYSTEM_CONFIG_FAILED")
		return
	}

	if h.hub != nil {
		changedKeys := make([]string, 0, len(configEntries))
		for _, entry := range configEntries {
			changedKeys = append(changedKeys, entry.Key)
		}
		_ = h.hub.Broadcast(panelws.MessageTypeConfigUpdated, gin.H{"keys": changedKeys})
	}

	utils.Success(c, uploadResponse)
}

// DownloadDeployAsset GET /api/v1/public/deploy-assets/:assetType/:fileName
func (h *SystemHandler) DownloadDeployAsset(c *gin.Context) {
	assetType := strings.ToLower(strings.TrimSpace(c.Param("assetType")))
	if !isSupportedDeployAssetType(assetType) {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "文件不存在")
		return
	}

	fileName := strings.TrimSpace(c.Param("fileName"))
	if !isSafeDeployAssetFileName(fileName) {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "文件名不合法")
		return
	}

	baseDir := resolveDeployAssetBaseDir()
	typeDir := filepath.Clean(filepath.Join(baseDir, assetType))
	fullPath := filepath.Clean(filepath.Join(typeDir, fileName))
	if !strings.HasPrefix(fullPath, typeDir+string(os.PathSeparator)) {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "文件路径不合法")
		return
	}

	info, statErr := os.Stat(fullPath)
	if statErr != nil || info.IsDir() {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "文件不存在")
		return
	}

	if assetType == deployAssetTypeScript {
		c.Header("Content-Type", "text/x-shellscript; charset=utf-8")
	}
	c.File(fullPath)
}

func isSupportedDeployAssetType(assetType string) bool {
	return assetType == deployAssetTypeScript || assetType == deployAssetTypeImage
}

func maxDeployAssetFileSize(assetType string) int64 {
	if assetType == deployAssetTypeScript {
		return maxDeployScriptFileSize
	}
	return maxDeployImageFileSize
}

func isValidDeployAssetFile(assetType string, fileName string) bool {
	lowerName := strings.ToLower(strings.TrimSpace(fileName))
	switch assetType {
	case deployAssetTypeScript:
		return strings.HasSuffix(lowerName, ".sh")
	case deployAssetTypeImage:
		return strings.HasSuffix(lowerName, ".tar") ||
			strings.HasSuffix(lowerName, ".tar.gz") ||
			strings.HasSuffix(lowerName, ".tgz") ||
			strings.HasSuffix(lowerName, ".zip") ||
			strings.HasSuffix(lowerName, ".img")
	default:
		return false
	}
}

func sanitizeUploadFileName(fileName string) string {
	baseName := strings.TrimSpace(filepath.Base(fileName))
	if baseName == "" || baseName == "." || baseName == ".." {
		baseName = "upload.bin"
	}
	baseName = strings.ReplaceAll(baseName, " ", "_")
	baseName = deployAssetUnsafeCharRegex.ReplaceAllString(baseName, "_")
	if baseName == "" || baseName == "." || baseName == ".." {
		baseName = "upload.bin"
	}
	return baseName
}

func isSafeDeployAssetFileName(fileName string) bool {
	trimmed := strings.TrimSpace(fileName)
	if trimmed == "" {
		return false
	}
	if filepath.Base(trimmed) != trimmed {
		return false
	}
	return deployAssetFileNameRegex.MatchString(trimmed)
}

func resolveDeployAssetBaseDir() string {
	customPath := strings.TrimSpace(os.Getenv("NODEPASS_DEPLOY_ASSET_DIR"))
	if customPath == "" {
		return defaultDeployAssetDir
	}
	return customPath
}

func normalizePublicBaseURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	parsedURL, err := url.Parse(value)
	if err != nil || strings.TrimSpace(parsedURL.Host) == "" {
		return ""
	}
	return strings.TrimRight(parsedURL.Scheme+"://"+parsedURL.Host, "/")
}
