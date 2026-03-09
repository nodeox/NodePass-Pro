package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"nodepass-license-unified/internal/services"
	"nodepass-license-unified/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	defaultReleaseUploadDir = "./uploads/releases"
	maxReleaseUploadSize    = int64(512 * 1024 * 1024)
)

// UnifiedHandler 统一业务处理器。
type UnifiedHandler struct {
	service          *services.UnifiedService
	releaseUploadDir string
}

// NewUnifiedHandler 创建处理器。
func NewUnifiedHandler(service *services.UnifiedService) *UnifiedHandler {
	return &UnifiedHandler{
		service:          service,
		releaseUploadDir: defaultReleaseUploadDir,
	}
}

// SetReleaseUploadDir 设置版本安装包上传目录。
func (h *UnifiedHandler) SetReleaseUploadDir(dir string) {
	trimmed := strings.TrimSpace(dir)
	if trimmed == "" {
		return
	}
	h.releaseUploadDir = trimmed
}

// Verify 统一校验公开接口。
func (h *UnifiedHandler) Verify(c *gin.Context) {
	var req services.VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	result, err := h.service.Verify(&req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "VERIFY_FAILED", err.Error())
		return
	}

	utils.Success(c, result, "ok")
}

// ListPlans 查询套餐。
func (h *UnifiedHandler) ListPlans(c *gin.Context) {
	items, err := h.service.ListPlans()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, items, "ok")
}

// CreatePlan 创建套餐。
func (h *UnifiedHandler) CreatePlan(c *gin.Context) {
	var req services.CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	item, err := h.service.CreatePlan(&req)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "创建成功")
}

// UpdatePlan 更新套餐。
func (h *UnifiedHandler) UpdatePlan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	var req services.CreatePlanRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	item, err := h.service.UpdatePlan(uint(id), &req)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "更新成功")
}

// ClonePlan 克隆套餐。
func (h *UnifiedHandler) ClonePlan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	var req services.ClonePlanRequest
	if err = c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	item, err := h.service.ClonePlan(uint(id), &req)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "CLONE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "克隆成功")
}

// DeletePlan 删除套餐。
func (h *UnifiedHandler) DeletePlan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	force := false
	forceRaw := strings.TrimSpace(c.Query("force"))
	if forceRaw != "" {
		force, err = strconv.ParseBool(forceRaw)
		if err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "force 必须为 true/false")
			return
		}
	}
	if err = h.service.DeletePlanWithForce(uint(id), force); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "套餐不存在")
			return
		}
		utils.Error(c, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"id": id, "force": force}, "删除成功")
}

// GenerateLicenses 生成授权码。
func (h *UnifiedHandler) GenerateLicenses(c *gin.Context) {
	var req services.GenerateLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	adminID, _ := c.Get("admin_id")
	items, err := h.service.GenerateLicenses(&req, adminID.(uint))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "GENERATE_FAILED", err.Error())
		return
	}
	utils.Success(c, items, "生成成功")
}

// ListLicenses 查询授权码。
func (h *UnifiedHandler) ListLicenses(c *gin.Context) {
	planID, _ := strconv.ParseUint(c.Query("plan_id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	expireFrom, err := parseTimeQuery(c.Query("expire_from"), false)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "expire_from 时间格式无效")
		return
	}
	expireTo, err := parseTimeQuery(c.Query("expire_to"), true)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "expire_to 时间格式无效")
		return
	}
	if expireFrom != nil && expireTo != nil && expireFrom.After(*expireTo) {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "expire_from 不能大于 expire_to")
		return
	}
	sortBy, sortOrder, err := parseLicenseSortQuery(c.Query("sort_by"), c.Query("sort_order"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	result, err := h.service.ListLicenses(services.LicenseFilter{
		Status:     c.Query("status"),
		Customer:   c.Query("customer"),
		PlanID:     uint(planID),
		ExpireFrom: expireFrom,
		ExpireTo:   expireTo,
		SortBy:     sortBy,
		SortOrder:  sortOrder,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, result, "ok")
}

func parseTimeQuery(raw string, endOfDay bool) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err == nil {
		return &parsed, nil
	}

	dateOnly, dateErr := time.Parse("2006-01-02", trimmed)
	if dateErr != nil {
		return nil, err
	}
	if endOfDay {
		value := dateOnly.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return &value, nil
	}
	return &dateOnly, nil
}

func parseLicenseSortQuery(rawSortBy, rawSortOrder string) (string, string, error) {
	sortBy := strings.ToLower(strings.TrimSpace(rawSortBy))
	sortOrder := strings.ToLower(strings.TrimSpace(rawSortOrder))

	if sortBy == "" {
		if sortOrder != "" {
			return "", "", errors.New("sort_order 需要配合 sort_by 使用")
		}
		return "", "", nil
	}

	switch sortBy {
	case "created_at", "expires_at", "status":
	default:
		return "", "", errors.New("sort_by 仅支持 created_at/expires_at/status")
	}

	if sortOrder == "" {
		return sortBy, "desc", nil
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		return "", "", errors.New("sort_order 仅支持 asc/desc")
	}
	return sortBy, sortOrder, nil
}

// GetLicense 获取授权详情。
func (h *UnifiedHandler) GetLicense(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	item, err := h.service.GetLicense(uint(id))
	if err != nil {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "授权不存在")
		return
	}
	utils.Success(c, item, "ok")
}

// UpdateLicense 更新授权。
func (h *UnifiedHandler) UpdateLicense(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	var req services.UpdateLicenseRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	item, err := h.service.UpdateLicense(uint(id), &req)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "更新成功")
}

// RevokeLicense 吊销授权。
func (h *UnifiedHandler) RevokeLicense(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	if err = h.service.RevokeLicense(uint(id)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"id": id, "status": "revoked"}, "已吊销")
}

// RestoreLicense 恢复授权。
func (h *UnifiedHandler) RestoreLicense(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	if err = h.service.RestoreLicense(uint(id)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"id": id, "status": "active"}, "已恢复")
}

// DeleteLicense 删除授权（物理删除）。
func (h *UnifiedHandler) DeleteLicense(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	if err = h.service.DeleteLicense(uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "授权不存在")
			return
		}
		utils.Error(c, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"id": id}, "删除成功")
}

type batchDeleteLicensesRequest struct {
	LicenseIDs []uint `json:"license_ids"`
}

// BatchDeleteLicenses 批量删除授权（物理删除）。
func (h *UnifiedHandler) BatchDeleteLicenses(c *gin.Context) {
	var req batchDeleteLicensesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	deletedCount, err := h.service.BatchDeleteLicenses(req.LicenseIDs)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "未找到可删除授权")
			return
		}
		utils.Error(c, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"deleted_count": deletedCount}, "批量删除成功")
}

// ListActivations 查询绑定。
func (h *UnifiedHandler) ListActivations(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	items, err := h.service.ListActivations(uint(id))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, items, "ok")
}

// UnbindActivation 解绑单个设备绑定。
func (h *UnifiedHandler) UnbindActivation(c *gin.Context) {
	licenseID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || licenseID == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	activationID, err := strconv.ParseUint(c.Param("activation_id"), 10, 64)
	if err != nil || activationID == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "activation_id 无效")
		return
	}

	if err = h.service.UnbindActivationByID(uint(licenseID), uint(activationID)); err != nil {
		utils.Error(c, http.StatusBadRequest, "UNBIND_FAILED", err.Error())
		return
	}

	utils.Success(c, gin.H{
		"license_id":    licenseID,
		"activation_id": activationID,
	}, "解绑成功")
}

// ClearActivations 清空授权下所有绑定。
func (h *UnifiedHandler) ClearActivations(c *gin.Context) {
	licenseID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || licenseID == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	rows, err := h.service.ClearActivationsWithOperator(uint(licenseID), getAdminIDFromContext(c))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "CLEAR_ACTIVATIONS_FAILED", err.Error())
		return
	}

	utils.Success(c, gin.H{
		"license_id":      licenseID,
		"cleared_count":   rows,
		"remaining_count": 0,
	}, "清空绑定成功")
}

// ListReleases 查询版本发布记录。
func (h *UnifiedHandler) ListReleases(c *gin.Context) {
	items, err := h.service.ListReleases(c.Query("product"), c.Query("channel"))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, items, "ok")
}

// ListDeletedReleases 查询回收站中的版本发布记录。
func (h *UnifiedHandler) ListDeletedReleases(c *gin.Context) {
	items, err := h.service.ListDeletedReleases(c.Query("product"), c.Query("channel"))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, items, "ok")
}

// ListVersionSyncConfigs 查询全部 GitHub 镜像同步配置。
func (h *UnifiedHandler) ListVersionSyncConfigs(c *gin.Context) {
	items, err := h.service.ListVersionSyncConfigs()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, items, "ok")
}

// GetVersionSyncConfig 查询 GitHub 镜像同步配置。
func (h *UnifiedHandler) GetVersionSyncConfig(c *gin.Context) {
	item, err := h.service.GetVersionSyncConfigByProduct(c.Query("product"))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "ok")
}

// UpdateVersionSyncConfig 更新 GitHub 镜像同步配置。
func (h *UnifiedHandler) UpdateVersionSyncConfig(c *gin.Context) {
	var req services.UpdateVersionSyncConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	if req.Product == nil {
		product := strings.TrimSpace(c.Query("product"))
		if product != "" {
			req.Product = &product
		}
	}

	item, err := h.service.UpdateVersionSyncConfigWithOperator(&req, getAdminIDFromContext(c))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "更新成功")
}

type manualVersionSyncRequest struct {
	Product string `json:"product"`
}

// ManualSyncVersionMirror 手动触发 GitHub 镜像拉取。
func (h *UnifiedHandler) ManualSyncVersionMirror(c *gin.Context) {
	var req manualVersionSyncRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
			return
		}
	}
	product := strings.TrimSpace(req.Product)
	if product == "" {
		product = strings.TrimSpace(c.Query("product"))
	}
	result, err := h.service.ManualSyncVersionMirrorWithOperator(product, getAdminIDFromContext(c))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "SYNC_FAILED", err.Error())
		return
	}
	utils.Success(c, result, "同步完成")
}

// CreateRelease 创建版本发布记录。
func (h *UnifiedHandler) CreateRelease(c *gin.Context) {
	var req services.CreateReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	item, err := h.service.CreateRelease(&req)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "创建成功")
}

// UpdateRelease 更新版本发布记录。
func (h *UnifiedHandler) UpdateRelease(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	var req services.UpdateReleaseRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	item, err := h.service.UpdateReleaseWithOperator(uint(id), &req, getAdminIDFromContext(c))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "发布记录不存在")
			return
		}
		utils.Error(c, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "更新成功")
}

// UploadRelease 手动上传版本安装包并创建发布记录。
func (h *UnifiedHandler) UploadRelease(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "缺少上传文件 file")
		return
	}
	if file.Size <= 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "上传文件不能为空")
		return
	}
	if file.Size > maxReleaseUploadSize {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "上传文件超过 512MB 限制")
		return
	}

	req, err := parseCreateReleaseRequestFromForm(c)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	uploadDir := strings.TrimSpace(h.releaseUploadDir)
	if uploadDir == "" {
		uploadDir = defaultReleaseUploadDir
	}
	if err = os.MkdirAll(uploadDir, 0o755); err != nil {
		utils.Error(c, http.StatusInternalServerError, "UPLOAD_FAILED", "创建上传目录失败")
		return
	}

	fileName := sanitizeUploadFileName(file.Filename)
	storedName := fmt.Sprintf("%d_%s", time.Now().UTC().UnixNano(), fileName)
	filePath := filepath.Join(uploadDir, storedName)
	if err = c.SaveUploadedFile(file, filePath); err != nil {
		utils.Error(c, http.StatusInternalServerError, "UPLOAD_FAILED", "保存上传文件失败")
		return
	}

	fileSHA256, err := computeFileSHA256(filePath)
	if err != nil {
		_ = os.Remove(filePath)
		utils.Error(c, http.StatusInternalServerError, "UPLOAD_FAILED", "计算文件摘要失败")
		return
	}

	item, err := h.service.CreateReleaseWithPackage(req, &services.ReleasePackageInfo{
		FileName:   fileName,
		FilePath:   filePath,
		FileSize:   file.Size,
		FileSHA256: fileSHA256,
	})
	if err != nil {
		_ = os.Remove(filePath)
		utils.Error(c, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}

	utils.Success(c, item, "上传并创建发布成功")
}

// ReplaceReleasePackage 替换发布安装包。
func (h *UnifiedHandler) ReplaceReleasePackage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "缺少上传文件 file")
		return
	}
	if file.Size <= 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "上传文件不能为空")
		return
	}
	if file.Size > maxReleaseUploadSize {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "上传文件超过 512MB 限制")
		return
	}

	uploadDir := strings.TrimSpace(h.releaseUploadDir)
	if uploadDir == "" {
		uploadDir = defaultReleaseUploadDir
	}
	if err = os.MkdirAll(uploadDir, 0o755); err != nil {
		utils.Error(c, http.StatusInternalServerError, "UPLOAD_FAILED", "创建上传目录失败")
		return
	}

	fileName := sanitizeUploadFileName(file.Filename)
	storedName := fmt.Sprintf("%d_%s", time.Now().UTC().UnixNano(), fileName)
	filePath := filepath.Join(uploadDir, storedName)
	if err = c.SaveUploadedFile(file, filePath); err != nil {
		utils.Error(c, http.StatusInternalServerError, "UPLOAD_FAILED", "保存上传文件失败")
		return
	}

	fileSHA256, err := computeFileSHA256(filePath)
	if err != nil {
		_ = os.Remove(filePath)
		utils.Error(c, http.StatusInternalServerError, "UPLOAD_FAILED", "计算文件摘要失败")
		return
	}

	item, oldPath, err := h.service.ReplaceReleasePackageWithOperator(uint(id), &services.ReleasePackageInfo{
		FileName:   fileName,
		FilePath:   filePath,
		FileSize:   file.Size,
		FileSHA256: fileSHA256,
	}, getAdminIDFromContext(c))
	if err != nil {
		_ = os.Remove(filePath)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "发布记录不存在")
			return
		}
		utils.Error(c, http.StatusBadRequest, "REPLACE_FAILED", err.Error())
		return
	}

	if strings.TrimSpace(oldPath) != "" && strings.TrimSpace(oldPath) != strings.TrimSpace(filePath) {
		_ = os.Remove(oldPath)
	}
	utils.Success(c, item, "安装包替换成功")
}

// DeleteRelease 删除发布记录。
func (h *UnifiedHandler) DeleteRelease(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	_, err = h.service.DeleteReleaseWithOperator(uint(id), getAdminIDFromContext(c))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "发布记录不存在")
			return
		}
		utils.Error(c, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"id": id}, "已移入回收站")
}

// RestoreRelease 从回收站恢复发布记录。
func (h *UnifiedHandler) RestoreRelease(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	item, err := h.service.RestoreReleaseWithOperator(uint(id), getAdminIDFromContext(c))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "回收站中未找到该发布")
			return
		}
		utils.Error(c, http.StatusBadRequest, "RESTORE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "恢复成功")
}

// PurgeRelease 从回收站永久删除发布记录。
func (h *UnifiedHandler) PurgeRelease(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	oldPath, err := h.service.PurgeReleaseWithOperator(uint(id), getAdminIDFromContext(c))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "回收站中未找到该发布")
			return
		}
		utils.Error(c, http.StatusBadRequest, "PURGE_FAILED", err.Error())
		return
	}

	if strings.TrimSpace(oldPath) != "" {
		_ = os.Remove(oldPath)
	}
	utils.Success(c, gin.H{"id": id}, "永久删除成功")
}

// DownloadReleaseFile 下载版本安装包。
func (h *UnifiedHandler) DownloadReleaseFile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	release, err := h.service.GetReleaseByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "发布记录不存在")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}

	filePath := strings.TrimSpace(release.FilePath)
	if filePath == "" {
		utils.Error(c, http.StatusBadRequest, "FILE_NOT_FOUND", "该发布未上传安装包")
		return
	}
	if _, statErr := os.Stat(filePath); statErr != nil {
		if os.IsNotExist(statErr) {
			utils.Error(c, http.StatusNotFound, "FILE_NOT_FOUND", "安装包文件不存在")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "READ_FILE_FAILED", "读取安装包失败")
		return
	}

	fileName := strings.TrimSpace(release.FileName)
	if fileName == "" {
		fileName = filepath.Base(filePath)
	}
	c.FileAttachment(filePath, fileName)
}

// ListVersionPolicies 查询版本策略。
func (h *UnifiedHandler) ListVersionPolicies(c *gin.Context) {
	items, err := h.service.ListVersionPolicies(c.Query("product"), c.Query("channel"))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, items, "ok")
}

// CreateVersionPolicy 创建版本策略。
func (h *UnifiedHandler) CreateVersionPolicy(c *gin.Context) {
	var req services.CreateVersionPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	item, err := h.service.CreateVersionPolicy(&req)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "创建成功")
}

// UpdateVersionPolicy 更新版本策略。
func (h *UnifiedHandler) UpdateVersionPolicy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	var req services.UpdateVersionPolicyRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	item, err := h.service.UpdateVersionPolicy(uint(id), &req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "版本策略不存在")
			return
		}
		utils.Error(c, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "更新成功")
}

// DeleteVersionPolicy 删除版本策略。
func (h *UnifiedHandler) DeleteVersionPolicy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	if err = h.service.DeleteVersionPolicy(uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "版本策略不存在")
			return
		}
		utils.Error(c, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"id": id}, "删除成功")
}

// ListVerifyLogs 查询统一校验日志。
func (h *UnifiedHandler) ListVerifyLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.service.ListVerifyLogs(services.VerifyLogFilter{
		LicenseKey: c.Query("license_key"),
		Status:     c.Query("status"),
		Product:    c.Query("product"),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, result, "ok")
}

// Dashboard 获取统计。
func (h *UnifiedHandler) Dashboard(c *gin.Context) {
	stats, err := h.service.GetDashboardStats()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, stats, "ok")
}

func parseCreateReleaseRequestFromForm(c *gin.Context) (*services.CreateReleaseRequest, error) {
	product := strings.TrimSpace(c.PostForm("product"))
	if product == "" {
		return nil, errors.New("product 不能为空")
	}
	version := strings.TrimSpace(c.PostForm("version"))
	if version == "" {
		return nil, errors.New("version 不能为空")
	}

	req := &services.CreateReleaseRequest{
		Product:      product,
		Version:      version,
		Channel:      strings.TrimSpace(c.PostForm("channel")),
		ReleaseNotes: strings.TrimSpace(c.PostForm("release_notes")),
	}

	if mandatoryRaw := strings.TrimSpace(c.PostForm("is_mandatory")); mandatoryRaw != "" {
		mandatory, err := strconv.ParseBool(mandatoryRaw)
		if err != nil {
			return nil, errors.New("is_mandatory 必须为 true/false")
		}
		req.IsMandatory = mandatory
	}

	if activeRaw := strings.TrimSpace(c.PostForm("is_active")); activeRaw != "" {
		active, err := strconv.ParseBool(activeRaw)
		if err != nil {
			return nil, errors.New("is_active 必须为 true/false")
		}
		req.IsActive = &active
	}

	if publishedAtRaw := strings.TrimSpace(c.PostForm("published_at")); publishedAtRaw != "" {
		publishedAt, err := parseTimeQuery(publishedAtRaw, false)
		if err != nil {
			return nil, errors.New("published_at 时间格式无效")
		}
		req.PublishedAt = publishedAt
	}

	return req, nil
}

func sanitizeUploadFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "release-package.bin"
	}

	base = strings.ReplaceAll(base, " ", "_")
	var builder strings.Builder
	builder.Grow(len(base))
	for _, ch := range base {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '.' || ch == '-' || ch == '_' {
			builder.WriteRune(ch)
		}
	}
	result := strings.TrimSpace(builder.String())
	if result == "" {
		return "release-package.bin"
	}
	return result
}

func computeFileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
