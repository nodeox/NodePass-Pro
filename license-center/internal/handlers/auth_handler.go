package handlers

import (
	"net/http"

	"nodepass-license-center/internal/services"
	"nodepass-license-center/internal/utils"

	"github.com/gin-gonic/gin"
)

// AuthHandler 管理员认证处理器。
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler 创建认证处理器。
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login 管理员登录。
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	res, err := h.authService.Login(&req)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}
	utils.Success(c, res, "登录成功")
}

// Me 当前管理员信息。
func (h *AuthHandler) Me(c *gin.Context) {
	adminID := getAdminID(c)
	if adminID == 0 {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证")
		return
	}
	admin, err := h.authService.GetAdmin(adminID)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "管理员不存在")
		return
	}
	utils.Success(c, admin, "ok")
}

func getAdminID(c *gin.Context) uint {
	value, ok := c.Get("adminID")
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case uint:
		return v
	case int:
		if v > 0 {
			return uint(v)
		}
	case int64:
		if v > 0 {
			return uint(v)
		}
	case float64:
		if v > 0 {
			return uint(v)
		}
	}
	return 0
}
