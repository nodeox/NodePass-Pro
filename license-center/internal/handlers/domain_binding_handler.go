package handlers

import (
	"net/http"
	"strconv"

	"nodepass-license-center/internal/services"
	"nodepass-license-center/internal/utils"

	"github.com/gin-gonic/gin"
)

// DomainBindingHandler 域名绑定处理器
type DomainBindingHandler struct {
	domainService *services.DomainBindingService
}

// NewDomainBindingHandler 创建域名绑定处理器
func NewDomainBindingHandler(domainService *services.DomainBindingService) *DomainBindingHandler {
	return &DomainBindingHandler{
		domainService: domainService,
	}
}

// ChangeDomain 更换域名
func (h *DomainBindingHandler) ChangeDomain(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数无效")
		return
	}

	var req struct {
		NewDomain string `json:"new_domain" binding:"required"`
		Reason    string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数错误")
		return
	}

	operatorID := c.GetUint("admin_id")
	config := services.DefaultDomainBindingConfig

	if err := h.domainService.ChangeDomain(uint(id), req.NewDomain, req.Reason, operatorID, config); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, nil, "域名更换成功")
}

// UnbindDomain 解绑域名
func (h *DomainBindingHandler) UnbindDomain(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数无效")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	operatorID := c.GetUint("admin_id")

	if err := h.domainService.UnbindDomain(uint(id), req.Reason, operatorID); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, nil, "域名解绑成功")
}

// LockDomain 锁定域名
func (h *DomainBindingHandler) LockDomain(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数无效")
		return
	}

	var req struct {
		Domain string `json:"domain" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数错误")
		return
	}

	operatorID := c.GetUint("admin_id")

	if err := h.domainService.LockDomain(uint(id), req.Domain, operatorID); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, nil, "域名锁定成功")
}

// GetBindingHistory 获取绑定历史
func (h *DomainBindingHandler) GetBindingHistory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数无效")
		return
	}

	history, err := h.domainService.GetBindingHistory(uint(id))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", "查询失败")
		return
	}

	utils.Success(c, history, "ok")
}
